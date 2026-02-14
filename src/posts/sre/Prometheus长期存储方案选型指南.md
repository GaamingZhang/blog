---
date: 2026-02-11
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - SRE
tag:
  - SRE
  - ClaudeCode
---

# Prometheus 长期存储方案选型指南

## Prometheus 原生存储的边界

Prometheus 内置的 TSDB（时序数据库）是为"本地、短期、高密度写入"场景而设计的。它将数据按时间窗口切分成 Block，每个 Block 是一组不可变的压缩文件，查询时通过索引快速定位。这套设计极其高效，但有三个根本性约束：

**保留时长受磁盘限制。** 默认 15 天，调长需要成比例扩磁盘。一个中等规模集群（5000 个时序，15 秒采集间隔）每天产生约 1～2 GB 数据，保留一年就需要 500 GB 以上的单机磁盘。

**单节点无高可用。** TSDB 不支持分布式写入，两个 Prometheus 实例无法共享同一个存储。要做 HA 只能运行两个完全独立的副本，查询时数据重复，不经过去重的 Dashboard 会显示双倍数据。

**无横向扩展能力。** 当指标量增长到单机瓶颈，唯一选择是纵向加内存和 CPU，没有分片机制。

这三个约束在小规模自用场景下不是问题，但在生产环境、多集群、合规要求长期归档的场景中，必须引入外部存储方案。本文系统梳理四种主流路径，以及它们各自适合的规模和场景。

---

## 方案一：Remote Write 协议

### 机制原理

Prometheus 在采集完数据写入本地 TSDB 的同时，还可以通过 `remote_write` 配置将数据**并行推送**到外部存储。这是所有外部存储方案的基础协议。

```
Scrape Targets → Prometheus TSDB（本地短期存储）
                      ↓ remote_write（并行推送）
               外部存储后端（长期存储）
```

Remote Write 使用 HTTP POST，数据用 Snappy 压缩的 Protocol Buffers 格式传输，每个请求携带一批时序样本。Prometheus 在内存中维护一个 WAL（Write-Ahead Log）作为 remote write 的发送队列，即使网络临时中断也不会丢数据，恢复后会重新发送。

关键配置项：

```yaml
remote_write:
  - url: "http://remote-storage:9201/write"
    queue_config:
      max_samples_per_send: 10000    # 每批最多样本数
      capacity: 100000               # 队列容量
      max_shards: 30                 # 并发写入分片数
    write_relabel_configs:           # 写入前过滤/改写标签
      - source_labels: [__name__]
        regex: "go_.*"
        action: drop                 # 丢弃 Go 运行时指标
```

`max_shards` 控制并发写入线程数，是吞吐调优的核心参数。如果发现 `prometheus_remote_storage_samples_pending` 指标持续增长，说明队列积压，需要增大 `max_shards` 或提升后端写入性能。

### 常见后端概览

Remote Write 协议已成为事实标准，几乎所有 TSDB 都支持接收：

| 后端 | 定位 | 特点 |
|------|------|------|
| InfluxDB | 通用时序数据库 | 有自己的查询语言 Flux，与 PromQL 不完全兼容 |
| M3DB | Uber 开源 | 分布式、高性能，运维复杂 |
| Cortex | CNCF 毕业项目 | 多租户、水平扩展，Grafana Mimir 的前身 |
| Thanos | CNCF 孵化项目 | 以 Sidecar 模式为主，也支持接收 remote write |
| VictoriaMetrics | 轻量高效 | PromQL 完全兼容，资源占用低 |

后三者是当前生产环境最主流的选择，下面分别详述。

---

## 方案二：Thanos

### 架构哲学

Thanos 的核心思想是**"不改变 Prometheus，在其周围构建扩展层"**。每个 Prometheus 实例保持原样，Thanos 通过 Sidecar 和对象存储将它们串联成一个统一的查询系统。

```
┌─────────────────────────────────────────────────────┐
│  集群 A                          集群 B              │
│  [Prometheus] ←→ [Sidecar]     [Prometheus] ←→ [Sidecar]
│         ↓ 上传 TSDB Block              ↓ 上传 TSDB Block
└──────────┬─────────────────────────────┬────────────┘
           │                             │
           ▼                             ▼
     ┌─────────────────────────────────────┐
     │       对象存储（S3 / GCS / MinIO）   │
     └──────────────────┬──────────────────┘
                        │
              [Store Gateway 读取历史数据]
                        │
     ┌──────────────────▼──────────────────┐
     │         Thanos Query（聚合查询层）    │
     │  ← 接收来自 Sidecar + Store Gateway  │
     │    的查询结果并合并去重               │
     └──────────────────┬──────────────────┘
                        │
               [Grafana 数据源]
```

### 五个核心组件

**Sidecar** 是 Thanos 与 Prometheus 的接口。它以边车容器运行在同一个 Pod 内，做两件事：一是通过 gRPC Store API 将该 Prometheus 的实时数据暴露给 Thanos Query；二是监视 Prometheus 的数据目录，将已落盘的 TSDB Block 上传到对象存储。Sidecar 只上传已完成的 Block（默认 2 小时一个），实时数据仍由 Sidecar 直接代理查询。

**Store Gateway** 是历史数据的读取代理。它读取对象存储中的 Block 元数据，将其暴露为 gRPC Store API，同样接入 Thanos Query。Store Gateway 在启动时构建索引缓存，查询时不需要下载整个 Block，只拉取需要的 chunk，IO 效率较高。

**Querier** 是统一查询入口。它实现了完整的 Prometheus HTTP API，向所有注册的 Store（Sidecar + Store Gateway）并发发起子查询，合并结果后返回。Grafana 将其当作普通 Prometheus 数据源接入即可。Querier 的去重机制通过 `--query.replica-label=replica` 参数启用：查询时，具有相同 `cluster` 标签但不同 `replica` 标签的时序被认为是 HA 副本，自动合并为一条。

**Compactor** 是后台压缩任务，持续对对象存储中的 Block 做降采样和合并。原始数据保留全精度，5 分钟以上的历史额外存一份 5 分钟降采样，1 小时以上再存一份 1 小时降采样。查询大时间范围时，Querier 自动选择合适精度的降采样数据，避免扫描海量原始点。

**Ruler** 提供全局 Recording Rules 和 Alert Rules 的计算能力。与 Prometheus 内置的 Rules 不同，Ruler 从 Thanos Query 读取数据（因此可以访问跨集群数据），计算结果写回对象存储或通过 remote write 发出。

### Kubernetes 部署片段（Helm）

```yaml
# values.yaml for bitnami/thanos chart
query:
  enabled: true
  replicaCount: 2
  stores:
    - dnssrv+_grpc._tcp.thanos-storegateway
  replicaLabel: replica

storegateway:
  enabled: true
  persistence:
    enabled: true
    size: 20Gi   # 索引缓存

compactor:
  enabled: true
  retentionResolutionRaw: 30d
  retentionResolution5m: 120d
  retentionResolution1h: 365d

objstoreConfig: |-
  type: S3
  config:
    bucket: thanos-metrics
    endpoint: minio:9000
    access_key: minio
    secret_key: minio123
    insecure: true
```

### Thanos 的优缺点

**优点**：无缝接入现有 Prometheus 集群，不需要替换任何采集侧组件；HA 去重机制成熟；依托对象存储理论上无限期保留历史数据；Store API 标准化，各组件可独立扩展。

**缺点**：组件数量多（至少 4 个），运维认知成本较高；Sidecar 和 Querier 之间的实时数据路径存在 2 小时左右的"近期数据由 Sidecar 代理、历史数据由 Store Gateway 代理"的切换边界，偶尔引起查询边界异常；大规模场景下 Querier 的扇出查询（fan-out）可能成为延迟瓶颈。

---

## 方案三：VictoriaMetrics

### 定位与差异

VictoriaMetrics 的思路与 Thanos 完全不同：**它是一个完整的、独立的、高性能时序数据库**，不依赖 Prometheus 的存储，而是直接接管存储和查询职责。Prometheus 退化为纯采集器，通过 remote write 将数据推送给 VictoriaMetrics。

这个差异在工程实践中意味着：部署更简单，资源占用更低，但架构上 Prometheus 不再是数据的权威来源，需要接受这个思维转变。

### 单机版架构

单机版（`victoriametrics`）是一个单一二进制，适合中小规模场景：

```
Prometheus（或 vmagent）
     ↓ remote_write
VictoriaMetrics（单机）
  - 接收数据（HTTP :8428）
  - 存储压缩（比 Prometheus 节省 7～10 倍磁盘）
  - PromQL 查询（/api/v1/query）
     ↑
   Grafana
```

启动极为简单：

```bash
docker run -d \
  -p 8428:8428 \
  -v /data/victoria:/victoria-metrics-data \
  victoriametrics/victoria-metrics:latest \
  -retentionPeriod=12   # 保留 12 个月
```

Prometheus 只需在 `remote_write` 指向 VictoriaMetrics 即可，无需任何其他改动。

### 集群版架构

集群版将单机的职责拆分为三个专用组件，分别独立扩展：

```
            vmagent（轻量采集器，替代 Prometheus remote_write）
                ↓
         vminsert（写入路由层）
         / 按 metric 名称或标签哈希分片 \
vmstore-0      vmstore-1      vmstore-2   （存储节点）
         \ 读取时从所有节点聚合 /
         vmselect（查询路由层）
                ↑
             Grafana
```

- **vminsert** 接收 remote write 请求，根据一致性哈希将时序路由到对应的 vmstore 节点，支持多副本写入
- **vmstorage** 是纯存储节点，只负责写入和读取本分片的数据
- **vmselect** 接收 PromQL 查询，并发查询所有 vmstore，合并结果返回

这种架构的好处是每个组件可以独立扩容：写入压力大就增加 vminsert 副本；数据量大就增加 vmstore 节点；查询并发高就增加 vmselect 副本。

### 高压缩比的原理

VictoriaMetrics 的存储压缩比远超 Prometheus TSDB，核心来自两个技术决策：

一是**时间戳编码优化**。Prometheus 使用 Gorilla 算法对时间戳做 delta-of-delta 编码，假设采集间隔规律。VictoriaMetrics 则对时间戳序列整体做 XOR + variable-length 编码，并允许时间戳不规律（适应 pull/push 混合场景），压缩效率更高。

二是**列式存储布局**。所有时序的同一字段（时间戳、值）分开存储，相邻值之间的差异极小，压缩算法能发挥最大效果。Prometheus TSDB 则是将同一时序的时间戳和值打包在一起存储的。

实测下来，VictoriaMetrics 的磁盘占用通常是 Prometheus 的 1/7～1/10。

### vmAgent：轻量采集替代品

vmAgent 是 VictoriaMetrics 生态中的采集器，兼容 Prometheus 的 `scrape_configs` 语法，但资源占用更低（没有本地存储开销），且内置持久化队列，网络中断时数据不丢失。在大规模场景下，可以将 Prometheus 完全替换为 vmAgent，每个节点只负责采集和推送，存储完全由 VictoriaMetrics 集群接管：

```yaml
# vmagent 配置示例（与 prometheus.yml 格式一致）
scrape_configs:
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: "true"

# remote_write 配置（推送到 VictoriaMetrics 集群）
remote_write:
  - url: http://vminsert:8480/insert/0/prometheus/api/v1/write
```

### Kubernetes 集群版部署片段

```yaml
# vmcluster CRD（使用 victoria-metrics-operator）
apiVersion: operator.victoriametrics.com/v1beta1
kind: VMCluster
metadata:
  name: vmcluster
spec:
  retentionPeriod: "12"     # 12 个月
  replicationFactor: 2      # 每份数据写入 2 个 vmstorage

  vmstorage:
    replicaCount: 3
    storage:
      volumeClaimTemplate:
        spec:
          storageClassName: fast-ssd
          resources:
            requests:
              storage: 200Gi

  vminsert:
    replicaCount: 2

  vmselect:
    replicaCount: 2
    cacheMountPath: "/select-cache"
    storage:
      volumeClaimTemplate:
        spec:
          resources:
            requests:
              storage: 5Gi
```

---

## 方案四：Cortex 与 Grafana Mimir

### Cortex 的定位

Cortex 是 CNCF 毕业项目，最初由 Weaveworks 开发，面向的场景是**多租户、大规模 SaaS 监控平台**。它将 Prometheus 的存储层完全分布式化，每个查询和写入请求都经过一系列微服务处理，每个服务都可以独立扩展。

```
remote_write → Distributor → Ingester（内存+WAL）
                                  ↓ 定期刷盘
                            对象存储（S3/GCS）
                                  ↑
                            Store Gateway + Querier
                                  ↑
                               Grafana
```

Cortex 引入了**租户（Tenant）**概念：每个 HTTP 请求通过 `X-Scope-OrgID` 头指定租户 ID，数据和查询完全隔离。这使得一套 Cortex 集群可以安全地为数百个团队或客户提供服务，各自只能查询自己的数据。

### Grafana Mimir：Cortex 的进化版

2022 年，Grafana Labs 在 Cortex 基础上推出 Mimir，主要改进了部署复杂度和查询性能。Mimir 将 Cortex 十几个微服务合并为几个逻辑组件，并支持"单体模式"（所有组件合并为一个二进制）方便小规模入门，再按需拆分扩展。

Mimir 的查询性能显著优于 Cortex，尤其在大时间范围的聚合查询上，通过引入新的 Streaming 查询引擎减少了内存压力。

### 适用场景

Cortex/Mimir 的主要使用场景是**平台工程团队为内部多个业务部门或外部客户提供统一监控服务**。它的多租户隔离、细粒度 Rate Limiting、租户级配额管理是其他方案没有的特性。但这也意味着它的运维复杂度是所有方案中最高的：组件最多，依赖最重（需要 memcached、etcd 等），调优参数繁多。

对于单一团队运维的中小规模场景，Cortex/Mimir 的复杂度远超实际需求。

---

## 方案对比

| 维度 | Thanos | VictoriaMetrics | Cortex / Mimir |
|------|--------|-----------------|----------------|
| 架构复杂度 | 中（4～5 个组件） | 低（单机）/ 中（集群） | 高（10+ 组件） |
| 与现有 Prometheus 集成 | 无缝（Sidecar 模式） | 需要改 remote_write | 需要改 remote_write |
| 资源消耗 | 中（对象存储 IO 较重） | 极低（高压缩比） | 高（多服务 + 依赖） |
| 查询性能 | 中（扇出合并有延迟） | 高 | 高（Mimir 优化后） |
| 高可用 | 支持（Querier 多副本） | 支持（集群版多副本） | 支持（设计目标） |
| 多租户 | 不支持 | 企业版支持 | 原生支持 |
| 长期存储 | 对象存储（无限） | 本地磁盘（或 S3 Tier） | 对象存储（无限） |
| 社区活跃度 | 高（CNCF 孵化） | 高（快速增长） | 中（Mimir 较新）  |
| 适用规模 | 中大型（多集群） | 中小到大型（单/多集群） | 大型（多租户平台） |

---

## 选型建议

**小团队 / 单集群场景（指标量 < 100 万 active series）**

直接使用 VictoriaMetrics 单机版。部署是一个 Docker 容器，Prometheus 改一行 remote_write 配置，即可获得 12 个月的历史数据和 10 倍的磁盘节省。没有任何额外的运维负担。

**中等规模 / 多集群场景（集群数 2～10，指标量 100 万～1000 万 series）**

有两个方向：

- 如果已有多个 Prometheus 集群且不希望改动采集架构，选 **Thanos**。Sidecar 零侵入，对象存储接管历史数据，Querier 提供统一查询，这套架构在 Kubernetes 上的部署已经相当成熟。
- 如果愿意以 vmAgent 替换采集侧，选 **VictoriaMetrics 集群版**。资源占用和运维复杂度均低于 Thanos，PromQL 兼容性几乎无缝，尤其适合没有跨集群历史数据查询需求的场景。

**大规模 / 多团队平台场景（多租户，集群数 > 10，需要租户隔离）**

选 **Grafana Mimir**。多租户隔离和细粒度配额是其他方案无法替代的特性，运维成本的增加在这个规模下是合理的投入。如果团队已经在使用 Grafana Cloud，Mimir 也是其后端存储，文档和最佳实践非常丰富。

---

## 小结

- Prometheus 原生存储的三个根本限制（保留时长、单点无 HA、无横向扩展）是引入外部存储的驱动力
- **Remote Write** 是所有外部存储方案的基础协议，理解其队列机制有助于调优写入性能
- **Thanos** 以"不动 Prometheus"为原则，通过 Sidecar + 对象存储 + Querier 构建跨集群长期存储，适合多集群现有架构的平滑扩展
- **VictoriaMetrics** 是性价比最高的方案，高压缩比和低资源占用使其成为中小规模场景的首选，集群版也能支撑大规模需求
- **Cortex / Mimir** 适合需要多租户隔离的平台化场景，复杂度最高，但特性也最完整
- 选型的核心维度是：集群规模、是否需要多租户、团队的运维能力、以及是否愿意改动采集侧架构

---

## 常见问题

### Q1：Prometheus 的 remote_write 是否会影响本地采集的性能？

Remote Write 是异步发送，采集写入本地 TSDB 和推送到远端是并行的，**正常情况下不影响采集**。真正需要关注的是 WAL（Write-Ahead Log）的磁盘 IO：remote write 的发送队列依赖 WAL 存储，如果远端持续慢、队列积压，WAL 会持续增长消耗磁盘空间。监控 `prometheus_remote_storage_samples_pending`（队列积压样本数）和 `prometheus_remote_storage_queue_highest_sent_timestamp_seconds`（发送进度）这两个指标，可以及时发现队列异常。

### Q2：Thanos 的 Sidecar 如何处理 Prometheus 重启或数据丢失的情况？

Sidecar 上传的是 Prometheus 已经**落盘且关闭**的 TSDB Block（通常 2 小时生成一个），上传成功后对象存储中的数据就独立于 Prometheus 存在了。Prometheus 本地数据丢失（如 PVC 被删除）只影响最近 2 小时内尚未上传的数据，历史数据通过 Store Gateway 仍然可以查询。这也意味着 Thanos 的 RTO（恢复时间目标）非常短：重新启动 Prometheus + Sidecar 后，历史数据立刻可用，只有那一小段近期数据存在空隙。

### Q3：VictoriaMetrics 和 Prometheus 的 PromQL 兼容性如何？有哪些不兼容的地方？

VictoriaMetrics 对 PromQL 的兼容性超过 99%，日常使用的查询语句几乎没有差异。主要的差异点有：VictoriaMetrics 扩展了 MetricsQL 方言，增加了若干聚合函数（如 `any()`、`bottomk_avg()`）；对某些边缘语义的处理与 Prometheus 略有不同，例如 `@` 修饰符和负偏移（`offset -5m`）的行为；`subquery` 语法支持但行为略有差异。对于 Grafana Dashboard 和 Alerting 规则的迁移，实践中几乎不需要修改查询语句。

### Q4：在 Kubernetes 环境中，Thanos 和 VictoriaMetrics 哪个部署更简单？

VictoriaMetrics 单机版的 Kubernetes 部署更简单：一个 Deployment + 一个 PVC，再加一个 Service，不超过 30 行 YAML。Thanos 即使使用 Helm chart，也需要理解 Sidecar、Store Gateway、Querier、Compactor 四个组件的配置，以及对象存储的对接。如果使用 victoria-metrics-operator，通过 VMCluster CRD 部署集群版也非常简洁。总体而言，入门曲线 VictoriaMetrics < Thanos < Cortex/Mimir。

### Q5：长期存储方案是否会影响告警的实时性？

不会，只要告警规则仍然在 Prometheus 本地执行。Prometheus 的 Alert Rules 使用本地数据计算，与 remote write 完全无关，延迟与不接入长期存储时完全相同。需要注意的场景是使用 Thanos Ruler 或 VictoriaMetrics 的 vmalert 在**全局层面**执行跨集群告警规则，此时查询数据需经过 Querier，会引入额外的网络延迟（通常在几百毫秒以内）。对于绝大多数告警规则（评估间隔 1 分钟），这点延迟完全可以忽略。

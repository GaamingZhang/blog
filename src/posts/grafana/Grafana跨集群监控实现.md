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

# Grafana Dashboard 如何实现跨集群监控

## 问题的本质

跨集群监控面临的根本矛盾是：**每个集群有独立的 Prometheus 实例，数据天然隔离，而 Grafana 的一个查询只能面向一个数据源**。

想在同一个 Dashboard 上看到 `prod-east`、`prod-west`、`staging` 三个集群的 QPS 对比，本质上需要解决两个问题：

1. **数据汇聚**：如何让多个集群的指标在逻辑上可以被统一查询
2. **集群区分**：查询到的数据如何区分来自哪个集群

这两个问题有三种典型的解决路径，复杂度和适用规模各不相同。

## 关键基础：external_labels

无论采用哪种架构，`external_labels` 都是跨集群监控的基础标签机制。在每个集群的 Prometheus 配置中加入：

```yaml
# prod-east 集群的 prometheus.yml
global:
  external_labels:
    cluster: prod-east    # 集群标识
    env: prod             # 环境标识
    region: us-east-1     # 可选：区域信息
```

`external_labels` 的作用是将这些标签**附加到该 Prometheus 实例产生的所有时序数据和告警上**。这样，来自不同集群的相同指标（如 `http_requests_total`）就能通过 `cluster` 标签区分。

有了 `cluster` 标签之后，PromQL 可以按集群聚合或过滤：

```promql
# 按集群汇总 QPS
sum by (cluster) (rate(http_requests_total[5m]))

# 只看 prod-east 的 QPS
rate(http_requests_total{cluster="prod-east"}[5m])
```

但这个前提是：数据必须先汇聚到一个可查询的地方。下面介绍三种汇聚方案。

## 方案一：多数据源 + Data Source Variable

### 原理

最简单的方案：不汇聚数据，而是**在 Grafana 层面提供切换能力**。每个集群的 Prometheus 在 Grafana 中注册为独立的数据源，通过 Dashboard 变量让用户选择当前查看的集群。

```
集群 prod-east  →  Prometheus A  →  Grafana 数据源: "Prometheus-prod-east"
集群 prod-west  →  Prometheus B  →  Grafana 数据源: "Prometheus-prod-west"
集群 staging    →  Prometheus C  →  Grafana 数据源: "Prometheus-staging"
                                         ↓
                                   Dashboard 变量: $datasource
                                   用户通过下拉切换
```

### Data Source Variable 的工作机制

在 Dashboard 的变量配置中，创建一个 **Data Source 类型**的变量：

- **Variable type**：Data Source
- **Data source type**：Prometheus
- **Instance name filter**（可选）：用正则过滤只展示特定前缀的数据源，如 `Prometheus-.*`

创建完成后，Dashboard 顶部出现集群选择下拉框。所有 Panel 的数据源设置改为 `${datasource}`，Grafana 会在查询时将其替换为用户当前选中的实际数据源。

```
Panel 配置:
  Data source: ${datasource}   ← 变量引用
  Query: rate(http_requests_total[5m])

用户选择 "Prometheus-prod-east" →
  实际查询: 向 Prometheus A 发送 rate(http_requests_total[5m])

用户切换到 "Prometheus-prod-west" →
  实际查询: 向 Prometheus B 发送 rate(http_requests_total[5m])
```

### 局限性

这种方案**每次只能看一个集群**，无法在同一张图上对比多个集群的指标趋势。适合集群数量少（2～5 个）、主要需求是逐集群查看的场景。

---

## 方案二：Prometheus Federation（联邦）

### 原理

Federation 通过 Prometheus 内置的 `/federate` 接口实现：**一个"全局 Prometheus"定期从各集群的 Prometheus 拉取选定的指标**，将多集群数据汇聚到单一存储中。

```
集群 Prometheus A ←── 全局 Prometheus 定期抓取 /federate
集群 Prometheus B ←── 全局 Prometheus 定期抓取 /federate
集群 Prometheus C ←── 全局 Prometheus 定期抓取 /federate
                           ↓
                      单一 Prometheus（全局数据）
                           ↓
                         Grafana（单一数据源）
```

### /federate 接口的工作方式

每个集群的 Prometheus 都暴露 `/federate` 端点，支持通过 `match[]` 参数查询要导出的指标：

```
GET http://cluster-prometheus:9090/federate?match[]=up&match[]={job="kubernetes-nodes"}
```

全局 Prometheus 在其配置中将各集群的 `/federate` 作为抓取目标：

```yaml
# 全局 Prometheus 的 prometheus.yml
scrape_configs:
  - job_name: 'federate-prod-east'
    honor_labels: true          # 关键：保留原始标签，不覆盖
    metrics_path: '/federate'
    params:
      match[]:
        - '{__name__=~"job:.*"}'           # 只拉取预聚合的 recording rule 结果
        - 'up'
        - 'http_requests_total'
    static_configs:
      - targets: ['prometheus-prod-east:9090']
        labels:
          cluster: prod-east    # 加上集群标识（如果原始数据没有）

  - job_name: 'federate-prod-west'
    honor_labels: true
    metrics_path: '/federate'
    params:
      match[]:
        - '{__name__=~"job:.*"}'
        - 'up'
    static_configs:
      - targets: ['prometheus-prod-west:9090']
        labels:
          cluster: prod-west
```

### honor_labels 的关键作用

`honor_labels: true` 告诉全局 Prometheus：**如果抓取到的数据已经有某个标签，就保留原始值，不用全局 Prometheus 自己的配置覆盖它**。

如果各集群已经通过 `external_labels` 设置了 `cluster` 标签，`honor_labels: true` 能确保这些标签被完整保留到全局 Prometheus 中。

### Federation 的适用场景与局限

**适合的场景**：
- 只需要跨集群查看**汇总指标**（如各集群的总 QPS、错误率）
- 各集群已有 Recording Rules 预计算聚合值
- 集群数量不多（5 个以内），每个集群指标量不大

**主要局限**：
- **数据选择性**：必须预先声明要联邦哪些指标，无法按需查询全量数据
- **时延叠加**：全局 Prometheus 的抓取间隔叠加在集群 Prometheus 的采集间隔上，数据时效性略差
- **扩展性差**：集群数量增多时，全局 Prometheus 的压力线性增长，成为单点瓶颈
- **不支持历史数据**：全局 Prometheus 只有自己启动后联邦的数据，无法查询联邦前各集群的历史

---

## 方案三：Thanos（推荐用于生产多集群）

### 为什么需要 Thanos

Federation 的核心问题在于：它仍然是一个"抓取"模型，需要预定义想要哪些数据，且规模有限。Thanos 则采用完全不同的思路：**让各集群的 Prometheus 保持独立，在查询层做跨集群的透明聚合**。

### Thanos 核心组件

**Thanos Sidecar**

以 Sidecar 容器的形式与每个 Prometheus 实例共同部署，承担两项职责：

1. **向 Thanos Query 暴露该 Prometheus 的数据**（通过 gRPC Store API）
2. **将 Prometheus 数据上传到对象存储**（S3/GCS 等），实现长期保存

```
Pod: Prometheus + Thanos Sidecar
     Prometheus ──→ Thanos Sidecar ──→ 对象存储（S3）
                         ↑
                   Thanos Query 通过 gRPC 查询
```

**Thanos Query**

跨集群查询的核心组件。它实现了 Prometheus HTTP API，Grafana 将其作为普通 Prometheus 数据源接入。收到查询请求后，Thanos Query **并发向所有注册的 Store（各集群的 Sidecar）发起查询**，将结果合并后返回。

```
Grafana → Thanos Query → Sidecar (prod-east)
                       → Sidecar (prod-west)
                       → Sidecar (staging)
                       → Thanos Store Gateway (历史数据，读对象存储)
         ← 合并所有结果 ←
```

对 Grafana 来说，Thanos Query 的接口与 Prometheus 完全兼容，接入方式完全相同。

**Thanos Store Gateway**

读取对象存储中的历史数据块，以 gRPC Store API 的形式暴露给 Thanos Query。实现了**超长历史数据的查询**，而 Prometheus 本身的保留时长通常只有 2 周。

### Thanos 的去重机制

在 Prometheus HA 部署中（两个相同配置的 Prometheus 同时运行），同一条时序数据会被存储两份，查询时会出现重复。Thanos 通过 `external_labels` 来解决这个问题。

两个 HA Prometheus 实例使用相同的 `cluster` 标签，但加上不同的 `replica` 标签：

```yaml
# Prometheus 实例 1
global:
  external_labels:
    cluster: prod-east
    replica: "0"

# Prometheus 实例 2
global:
  external_labels:
    cluster: prod-east
    replica: "1"
```

Thanos Query 查询时，指定 `--query.replica-label=replica`，查询引擎会自动对 `replica` 标签不同但其他标签相同的时序进行去重，最终结果中 `replica` 标签被移除，数据只保留一份。

### 在 Grafana 中使用 Thanos

接入方式与普通 Prometheus 数据源完全相同，只是将地址指向 Thanos Query：

```
Data Source Type: Prometheus
URL: http://thanos-query:9090
```

由于 Thanos Query 已经合并了所有集群的数据，查询时各集群的 `cluster` 标签自动可见，无需任何特殊配置。

Dashboard 变量可以直接查询 `cluster` 标签的所有值：

```
Variable Type: Query
Data Source: Thanos Query 数据源
Query: label_values(up, cluster)    # 自动列出所有集群
```

### 三种方案对比

| 维度 | 多数据源切换 | Federation | Thanos |
|------|------------|------------|--------|
| 实现复杂度 | 低 | 中 | 高 |
| 多集群同屏对比 | 不支持 | 支持 | 支持 |
| 全量指标查询 | 支持（单集群） | 不支持（需预定义） | 支持 |
| 历史数据长度 | 受各集群限制 | 受全局 Prometheus 限制 | 无限（对象存储） |
| HA 去重 | 不支持 | 不支持 | 支持 |
| 适用集群数 | 2～5 个 | 5 个以内 | 不限 |
| 运维成本 | 极低 | 低 | 较高 |

---

## Dashboard 变量：跨集群查询的 UI 层

无论采用哪种数据汇聚方案，Dashboard 变量都是提供集群选择交互的核心机制。以 Thanos 方案为例，设计多集群 Dashboard 的变量链：

### 变量链设计

```
$datasource（Data Source 变量，可选）
      ↓
$cluster（Query 变量：label_values(up, cluster)）
      ↓
$namespace（Query 变量：label_values(up{cluster="$cluster"}, namespace)）
      ↓
$pod（Query 变量：label_values(up{cluster="$cluster", namespace="$namespace"}, pod)）
```

每级变量依赖上级的值，通过级联过滤减少选项数量，同时保证选项的有效性。

### 在 Panel 中使用集群变量

Panel 的查询通过 `$cluster` 变量实现集群过滤，同时支持多选（`Multi-value`）实现多集群同屏对比：

```promql
# 启用 Multi-value 后，$cluster 展开为正则：prod-east|prod-west
sum by (cluster) (
  rate(http_requests_total{cluster=~"$cluster"}[5m])
)
```

启用 `Include All option` 和 `Multi-value` 后，用户可以选择"全部集群"或任意组合，在同一张图上叠加多个集群的曲线进行横向对比。

### 跨集群对比 Panel 的设计技巧

**使用 Legend 区分集群**

在 Panel 的 Legend 配置中，将 `cluster` 标签加入 Legend Format，使每条曲线都标注集群名称：

```
Legend Format: {{ cluster }} - {{ job }}
```

**使用 Override 为不同集群着色**

在 Panel 的 Field Override 中，按 `cluster` 标签值为曲线设置固定颜色，避免颜色随时间变动导致视觉混乱。

---

## 典型生产架构

结合以上机制，一套完整的多集群 Grafana 监控架构如下：

```
┌──────────────────────────────────────────────────────────┐
│                    各业务集群                              │
│                                                          │
│  [prod-east]                [prod-west]                  │
│  Prometheus + Sidecar       Prometheus + Sidecar         │
│  external_labels:           external_labels:             │
│    cluster: prod-east         cluster: prod-west         │
│    replica: "0"               replica: "0"               │
└──────────────┬───────────────────────┬───────────────────┘
               │ gRPC Store API        │
               ▼                       ▼
┌─────────────────────────────────────────────────────────┐
│                    监控中心                               │
│                                                         │
│  Thanos Query（合并查询 + 去重）                          │
│      ↑                                                  │
│  Thanos Store Gateway（历史数据，读 S3）                  │
│                                                         │
│  Grafana                                                │
│    数据源: Thanos Query                                  │
│    变量:   $cluster = label_values(up, cluster)         │
│    Panel:  rate(http_requests_total{cluster=~"$cluster"}│
└─────────────────────────────────────────────────────────┘
```

## 总结

跨集群监控没有万能方案，选型应根据规模和需求决定：

- **集群少、只需切换查看**：多数据源 + Data Source Variable，零额外组件
- **需要汇总视图、集群不多**：Prometheus Federation，适度复杂
- **生产规模、需要全量查询和长期历史**：Thanos，一次性投入高但功能完整

无论哪种方案，`external_labels` 中规范的 `cluster` 标签是基础，Dashboard 变量链提供交互能力。两者是跨集群 Dashboard 设计的不变核心。

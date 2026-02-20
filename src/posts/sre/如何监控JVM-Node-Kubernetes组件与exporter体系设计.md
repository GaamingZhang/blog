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

# 如何监控 JVM、Node、Kubernetes 组件？exporter 体系如何组织？

## 从一个告警说起

某天凌晨，生产环境的 API 响应时间突然飙升。你打开 Grafana，发现服务 Pod 的 CPU 利用率正常、内存也没有异常。问题找了二十分钟，才发现是 JVM 的 Old Gen GC 频率突然增加，每次 Full GC 停顿接近两秒，导致请求超时。

这个场景暴露的问题不是告警慢，而是**监控体系分层不完整**：应用层（JVM）、系统层（Node）、调度层（Kubernetes）三个维度各自孤立，看不到完整的因果链。

这正是 Prometheus exporter 体系要解决的问题：**为每一层提供标准化的指标采集，并通过统一的标签体系将它们串联起来**。

## Exporter 的设计哲学

Prometheus 的数据采集遵循一个核心设计决策：**拉取（Pull）而非推送（Push）**。Prometheus Server 主动向各个 exporter 发起 HTTP 请求，拉取 `/metrics` 端点暴露的指标。

这个决策带来了一个自然的分工：**被监控的对象不需要知道 Prometheus 的存在**，只需要暴露一个 `/metrics` 端点，遵循 Prometheus 的文本格式（OpenMetrics）即可。Exporter 就是这个"翻译层"——它连接被监控系统，将系统内部的状态转换为 Prometheus 可以理解的指标格式。

```
被监控系统  →  Exporter（翻译层）→  /metrics 端点
                                        ↑
                              Prometheus Server 定期拉取
```

不同层次的 exporter 各司其职：

```
┌─────────────────────────────────────────────────────┐
│  应用层    JMX Exporter     →  JVM 内部状态           │
├─────────────────────────────────────────────────────┤
│  系统层    node_exporter    →  OS 资源（CPU/内存/磁盘）│
├─────────────────────────────────────────────────────┤
│  容器层    cAdvisor         →  容器资源使用            │
├─────────────────────────────────────────────────────┤
│  集群层    kube-state-metrics →  K8s 对象状态         │
├─────────────────────────────────────────────────────┤
│  组件层    各组件原生暴露    →  API Server/etcd 等     │
└─────────────────────────────────────────────────────┘
```

## JVM 监控：JMX Exporter

### 两种部署模式的本质差异

JMX Exporter 有两种工作模式，选择哪种取决于你能否修改 JVM 启动参数。

**Agent 模式**（推荐）将 exporter 作为 JVM agent 加载，与 JVM 进程共享内存空间，直接读取 JVM 内部数据：

```bash
java -javaagent:/opt/jmx_prometheus_javaagent.jar=9090:/opt/config.yaml \
     -jar your-application.jar
```

这种模式的优势在于零网络开销——exporter 直接从 JVM 内存读取数据，不需要通过 JMX 远程协议（RMI），延迟极低。

**Standalone 模式**则是一个独立进程，通过 JMX 远程协议连接到目标 JVM，适合不能修改应用启动参数的遗留系统，但需要目标 JVM 开启 JMX 远程访问，存在一定安全风险。

在 Kubernetes 环境中，Agent 模式通常以 Init Container 或 Volume Mount 的方式注入 jar 包：

```yaml
initContainers:
  - name: jmx-exporter-init
    image: busybox
    command: ['sh', '-c', 'cp /jmx_prometheus_javaagent.jar /shared/']
    volumeMounts:
      - name: jmx-jar
        mountPath: /shared
volumes:
  - name: jmx-jar
    emptyDir: {}
```

然后在应用容器的 JVM 参数中引用：

```yaml
env:
  - name: JAVA_OPTS
    value: "-javaagent:/shared/jmx_prometheus_javaagent.jar=9090:/config/jmx-config.yaml"
```

### 关键 JVM 指标与告警

JVM 监控的核心是**内存分区**和 **GC 行为**。理解指标的前提是理解 JVM 堆的分区模型：Eden → Survivor → Old Gen，每个区域的满溢行为都对应不同的 GC 类型。

**堆内存使用率**，区分新生代和老年代：

```promql
# 老年代使用率（Old Gen 是 Full GC 的触发器）
jvm_memory_bytes_used{area="heap", pool="G1 Old Gen"}
/ jvm_memory_bytes_max{area="heap", pool="G1 Old Gen"} * 100
```

**GC 停顿时间**是影响服务延迟的直接因素：

```promql
# 过去5分钟内 Full GC（Old Gen GC）的平均停顿时间（毫秒）
rate(jvm_gc_collection_seconds_sum{gc="G1 Old Generation"}[5m])
/ rate(jvm_gc_collection_seconds_count{gc="G1 Old Generation"}[5m]) * 1000
```

**GC 频率**反映内存压力趋势：

```promql
# Young GC 每分钟次数
rate(jvm_gc_collection_seconds_count{gc="G1 Young Generation"}[5m]) * 60
```

**线程状态**可以发现死锁或线程泄漏：

```promql
# 阻塞线程数
jvm_threads_state{state="BLOCKED"}
```

告警规则示例——Old Gen 使用率超过 80% 时提前预警：

```yaml
- alert: JvmOldGenHighUsage
  expr: |
    jvm_memory_bytes_used{pool="G1 Old Gen"}
    / jvm_memory_bytes_max{pool="G1 Old Gen"} > 0.8
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "JVM Old Gen 使用率过高（{{ $value | humanizePercentage }}）"
    description: "实例 {{ $labels.instance }} 的 Old Gen 已使用 {{ $value | humanizePercentage }}，有 Full GC 风险"
```

## Node 监控：node_exporter

### 部署方式

node_exporter 是系统层监控的标准 exporter，在裸金属或 VM 上直接安装运行即可。在 Kubernetes 环境中，推荐以 **DaemonSet** 形式部署，确保每个节点都有一个采集实例：

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-exporter
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: node-exporter
  template:
    spec:
      hostNetwork: true      # 使用主机网络，避免网络命名空间隔离
      hostPID: true          # 访问主机进程信息
      containers:
        - name: node-exporter
          image: prom/node-exporter:v1.7.0
          args:
            - --path.rootfs=/host           # 挂载主机根文件系统
            - --collector.filesystem.mount-points-exclude=^/(dev|proc|sys|var/lib/docker/.+)
          volumeMounts:
            - name: root
              mountPath: /host
              readOnly: true
      volumes:
        - name: root
          hostPath:
            path: /
```

`hostNetwork: true` 和 `hostPID: true` 是 node_exporter 正确工作的关键——它需要"看到"宿主机的网络接口和进程列表，而不是容器的网络命名空间。

### 关键系统指标

**CPU 使用率**需要通过 `rate` 计算，因为 `node_cpu_seconds_total` 是一个累加计数器：

```promql
# 节点 CPU 使用率（排除 idle 模式）
1 - avg by (instance) (
  rate(node_cpu_seconds_total{mode="idle"}[5m])
)
```

**内存可用量**要区分"空闲"和"可用"——`MemFree` 是真正空闲的内存，而 `MemAvailable` 是 Linux 内核评估的实际可用量（包括可回收的 buffer/cache），后者更有实际意义：

```promql
# 内存使用率（基于 MemAvailable）
1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes
```

**磁盘 IO** 的关键指标是 `iowait` 和吞吐量：

```promql
# 磁盘 iowait 比例（高 iowait 意味着存储成为瓶颈）
rate(node_cpu_seconds_total{mode="iowait"}[5m])

# 磁盘读写吞吐量
rate(node_disk_read_bytes_total[5m])
rate(node_disk_written_bytes_total[5m])
```

**网络接口**的带宽使用：

```promql
rate(node_network_receive_bytes_total{device!="lo"}[5m])
rate(node_network_transmit_bytes_total{device!="lo"}[5m])
```

常见告警规则——磁盘空间不足预警：

```yaml
- alert: NodeDiskSpaceLow
  expr: |
    node_filesystem_avail_bytes{mountpoint="/", fstype!="tmpfs"}
    / node_filesystem_size_bytes{mountpoint="/"} < 0.15
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "节点磁盘剩余空间不足 15%"
```

## Kubernetes 组件监控

Kubernetes 的监控比前两层更复杂，因为它既有**资源使用类指标**（Pod 用了多少 CPU），也有**状态类指标**（Deployment 期望几个副本、现在有几个就绪）。这两类指标由不同的组件提供。

### kube-state-metrics：集群状态的"翻译者"

kube-state-metrics 监听 Kubernetes API Server 的事件，将 K8s 对象的**声明状态**转换为 Prometheus 指标。它关注的不是资源消耗，而是 K8s 对象的逻辑状态：

```
# Deployment 副本数对比
kube_deployment_spec_replicas        # 期望副本数
kube_deployment_status_replicas_ready  # 就绪副本数

# Pod 状态
kube_pod_status_phase               # Running/Pending/Failed/Succeeded
kube_pod_container_status_restarts_total  # 容器重启次数

# Job 完成状态
kube_job_status_succeeded
kube_job_status_failed
```

一个非常实用的告警——检测部署是否出现"副本不足"：

```yaml
- alert: DeploymentReplicasMismatch
  expr: |
    kube_deployment_spec_replicas
    != kube_deployment_status_replicas_ready
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Deployment {{ $labels.namespace }}/{{ $labels.deployment }} 副本数不匹配"
```

检测容器频繁重启（CrashLoopBackOff 的早期信号）：

```yaml
- alert: ContainerRestartingFrequently
  expr: |
    rate(kube_pod_container_status_restarts_total[15m]) * 60 > 3
  for: 5m
  labels:
    severity: warning
```

### cAdvisor：容器资源的"观察者"

cAdvisor 内置在 kubelet 中（无需单独部署），通过 kubelet 的 `/metrics/cadvisor` 端点暴露容器维度的资源使用指标。它回答的是"这个容器实际用了多少资源"：

```promql
# 容器 CPU 使用率（对比 CPU 请求值）
rate(container_cpu_usage_seconds_total{container!=""}[5m])
/ on (pod, namespace, container)
kube_pod_container_resource_requests{resource="cpu"}
```

```promql
# 容器内存使用量（RSS，排除 cache）
container_memory_rss{container!=""}
```

容器内存使用率告警（超过 limit 的 90%，OOM kill 前预警）：

```yaml
- alert: ContainerMemoryNearLimit
  expr: |
    container_memory_rss{container!=""}
    / on (pod, namespace, container)
    kube_pod_container_resource_limits{resource="memory"} > 0.9
  for: 5m
  labels:
    severity: warning
```

### 核心组件监控

Kubernetes 控制面组件原生暴露 Prometheus 格式的 `/metrics` 端点，无需额外 exporter。

**API Server** 是集群的入口，关注请求延迟和错误率：

```promql
# API Server P99 延迟（按请求类型）
histogram_quantile(0.99,
  sum by (le, verb) (
    rate(apiserver_request_duration_seconds_bucket[5m])
  )
)

# API Server 错误率（5xx）
rate(apiserver_request_total{code=~"5.."}[5m])
/ rate(apiserver_request_total[5m])
```

**etcd** 是集群状态存储，磁盘写入延迟是关键指标：

```promql
# etcd 磁盘 fsync 延迟 P99（超过 10ms 需关注）
histogram_quantile(0.99,
  rate(etcd_disk_wal_fsync_duration_seconds_bucket[5m])
)
```

**Scheduler 和 Controller Manager** 主要关注调度延迟和队列深度：

```promql
# 调度延迟 P99
histogram_quantile(0.99,
  rate(scheduler_scheduling_algorithm_duration_seconds_bucket[5m])
)
```

### ServiceMonitor 与 PodMonitor：声明式采集配置

在 Kubernetes 环境中，Prometheus Operator 将 scrape 配置抽象为 CRD。ServiceMonitor 用于采集带 Service 的目标：

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: jvm-app-monitor
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: jvm-app        # 匹配 Service 的标签
  namespaceSelector:
    matchNames:
      - production
  endpoints:
    - port: metrics       # Service 中定义的端口名称
      path: /metrics
      interval: 30s
```

PodMonitor 则直接匹配 Pod，适合没有 Service 或需要采集每个 Pod 独立指标的场景（如有状态服务）：

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: jvm-pod-monitor
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: jvm-app
  podMetricsEndpoints:
    - port: jmx-metrics   # Pod 中定义的容器端口名称
      path: /metrics
```

ServiceMonitor 和 PodMonitor 的本质是 Prometheus Operator 监听这些 CRD，自动更新 Prometheus 的 scrape 配置。这避免了每次部署新服务都需要手动修改 Prometheus 配置文件的运维负担。

## Exporter 体系的组织原则

### 标签统一是基础

多层监控数据能否有效关联，取决于标签是否统一。以下几个标签在所有层次都应该保持一致：

| 标签 | 含义 | 来源 |
|------|------|------|
| `cluster` | 集群标识 | Prometheus `external_labels` |
| `namespace` | K8s 命名空间 | Kubernetes 服务发现自动附加 |
| `pod` | Pod 名称 | Kubernetes 服务发现自动附加 |
| `node` | 节点名称 | Kubernetes 服务发现自动附加 |
| `job` | 采集任务名称 | scrape config 中定义 |

通过 `pod` 标签，可以将 kube-state-metrics（Pod 状态）、cAdvisor（容器资源）和应用 JVM 指标关联到同一个工作负载。

### 服务发现替代静态配置

在 Kubernetes 环境中，应全面使用 Kubernetes 服务发现（`kubernetes_sd_configs`）替代静态 IP 配置。服务发现的优势在于：Pod 扩缩容、滚动更新时，Prometheus 自动感知新旧目标，无需人工维护 scrape target 列表。

```yaml
scrape_configs:
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      # 只采集有 prometheus.io/scrape: "true" 注解的 Pod
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: 'true'
      # 读取注解中指定的 metrics 路径
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
```

这样，开发团队只需在 Pod 上加一个注解 `prometheus.io/scrape: "true"`，Prometheus 就会自动开始采集，降低了运维摩擦。

### 分层告警，避免告警风暴

多层监控带来的一个副作用是**告警重复和告警风暴**。一个节点下线可能同时触发：node_exporter 的"节点不可达"告警、kube-state-metrics 的"Node NotReady"告警、以及该节点上所有 Pod 的"Target 不可达"告警。

合理的做法是**建立告警抑制规则**：当上层（基础设施层）告警触发时，抑制下层的衍生告警：

```yaml
# AlertManager 抑制规则
inhibit_rules:
  - source_match:
      alertname: NodeDown      # 节点故障告警作为"源"
    target_match_re:
      alertname: 'PodCrash|ContainerRestart|TargetDown'  # 抑制这些衍生告警
    equal: ['node']            # 同一节点上的下层告警才被抑制
```

### 完整的监控架构

将三层 exporter 体系组合，完整架构如下：

```
┌──────────────────────────────────────────────────────────────┐
│  应用层（JVM / 应用自定义指标）                                │
│  JMX Exporter (agent)   →   Pod 9090 端口   →  ServiceMonitor│
├──────────────────────────────────────────────────────────────┤
│  容器/调度层                                                  │
│  cAdvisor (kubelet内置) →  kubelet /metrics/cadvisor         │
│  kube-state-metrics     →  Deployment 暴露 /metrics           │
├──────────────────────────────────────────────────────────────┤
│  节点系统层                                                   │
│  node_exporter (DaemonSet) →  hostNetwork 9100 端口          │
├──────────────────────────────────────────────────────────────┤
│  K8s 控制面                                                   │
│  API Server / etcd / Scheduler / Controller Manager          │
│  各组件原生 /metrics 端点                                     │
└──────────────────────────────────────────────────────────────┘
                            ↓
                    Prometheus Server
                   (Kubernetes SD 服务发现)
                            ↓
              Grafana（分层 Dashboard：节点/集群/应用）
```

## 小结

- exporter 是"翻译层"，将各系统内部状态转为 Prometheus 格式，分应用层（JMX）、系统层（node_exporter）、容器层（cAdvisor）、集群状态层（kube-state-metrics）四个维度
- JVM 监控优先用 Agent 模式，核心关注 Old Gen 使用率和 GC 停顿时间
- node_exporter 在 K8s 中以 DaemonSet 部署，需要 `hostNetwork` 和 `hostPID` 才能正确采集宿主机数据
- kube-state-metrics 提供 K8s 对象的逻辑状态，cAdvisor 提供资源实际消耗，两者互补
- ServiceMonitor 和 PodMonitor 是 K8s 环境中 scrape 配置的推荐方式，实现声明式自动化采集
- 标签统一（cluster、pod、node）是多层数据关联的前提，AlertManager 抑制规则是避免告警风暴的关键

## 参考资源

- [Prometheus JMX Exporter](https://github.com/prometheus/jmx_exporter)
- [Node Exporter 官方文档](https://github.com/prometheus/node_exporter)
- [Kubernetes 监控架构](https://kubernetes.io/docs/concepts/cluster-administration/system-metrics/)

---

## 常见问题

### Q1：JMX Exporter 的 Agent 模式和 Standalone 模式如何选择？

Agent 模式通过 `-javaagent` 参数加载，与 JVM 进程在同一 JVM 实例中运行，直接访问 JVM 内部 MBean，无网络开销，是性能和安全性的最优选择。Standalone 模式需要目标 JVM 开启 JMX 远程访问（通常需要配置 RMI 端口），存在额外的网络跳转，并且 JMX 远程协议（RMI）有防火墙穿透难题。**只有在无法修改应用启动参数的遗留系统场景下才使用 Standalone 模式**。在 Kubernetes 中，可以用 Init Container 或 ConfigMap 挂载 jar 包，让 Agent 模式几乎在所有场景都可行。

### Q2：kube-state-metrics 和 cAdvisor 的根本区别是什么？

两者监控的维度完全不同。kube-state-metrics 关注的是 **K8s 对象的声明状态**：Deployment 期望几个副本、Pod 处于什么阶段（Pending/Running/Failed）、PVC 是否绑定等，这些是 Kubernetes 控制平面管理的对象状态。cAdvisor 关注的是**容器的实际资源消耗**：CPU 使用了多少核、内存 RSS 是多少字节、网络收发了多少包，这些是运行时的物理资源数据。一个判断"Deployment 是否健康"（副本数是否符合预期），另一个判断"容器是否高负荷"（资源是否接近 limit），需要结合使用才能完整评估一个工作负载。

### Q3：在 Kubernetes 中，ServiceMonitor 和直接配置 scrape_configs 有什么区别？

本质区别在于**配置的管理方式**。直接配置 `scrape_configs` 需要修改 Prometheus 的 ConfigMap 并 reload，属于命令式操作，每次新增服务都需要运维介入。ServiceMonitor 是 Prometheus Operator 提供的 CRD，开发团队可以在自己的命名空间中独立创建，Operator 监听到 ServiceMonitor 创建后自动更新 Prometheus 配置——这是声明式的、自服务的方式，符合 GitOps 工作流。此外，ServiceMonitor 有 `namespaceSelector` 字段，可以精确控制采集范围，比手动维护 scrape 列表更安全。

### Q4：节点宕机时，为什么会出现大量的衍生告警？如何处理？

当一个节点宕机，该节点上的所有采集目标（node_exporter、各 Pod 的应用 exporter）都无法访问，Prometheus 会为每个 target 触发 `up == 0` 的告警，同时 kube-state-metrics 会触发 Node NotReady 告警，该节点上的所有 Pod 也会进入非 Running 状态进而触发各自的 Pod 级别告警。这会产生几十甚至上百条同时发生的告警，淹没真正的根因信息。解决方案是在 AlertManager 中配置**告警抑制规则**（inhibit_rules）：以节点级别的 `NodeDown` 告警为"源"，抑制与其 `node` 标签相同的所有 Pod 级别告警。根因只需通知一次，其余衍生告警静默。

### Q5：如何为多租户集群设计合理的 exporter 标签体系？

多租户集群的核心挑战是：同一个 `pod_name` 可能在不同团队的 namespace 下都存在，单靠 `pod` 标签无法区分归属。推荐的标签层次是：`cluster`（集群标识）→ `namespace`（命名空间，通常对应团队或业务线）→ `pod`（工作负载实例）。`cluster` 标签通过 Prometheus 的 `external_labels` 统一注入，`namespace` 和 `pod` 由 Kubernetes 服务发现自动附加，无需手动维护。在 Grafana Dashboard 中，变量链设计为 `$cluster → $namespace → $deployment → $pod`，通过级联过滤将查询权限自然收敛到各租户的业务范围，同时运维团队可以在顶层选择"所有 namespace"进行全局视图。

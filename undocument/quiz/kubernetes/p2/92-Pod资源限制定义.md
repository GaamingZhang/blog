---
date: 2026-03-12
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Pod
  - 资源管理
---

# Kubernetes Pod 资源限制定义详解

## 引言

在 Kubernetes 集群中，资源管理是确保应用稳定运行和集群高效利用的关键。通过为 Pod 定义资源请求（Requests）和资源限制（Limits），可以控制容器对 CPU、内存等资源的使用，防止资源争抢和系统不稳定。

理解 Kubernetes 的资源管理机制，合理配置资源请求和限制，是构建生产级 Kubernetes 应用的必备技能。本文将深入剖析资源限制的定义方式、工作原理和最佳实践。

## 资源类型概述

### 可管理的资源类型

| 资源类型 | 说明 | 单位 |
|---------|------|------|
| **cpu** | CPU 资源 | millicore (m) 或 core |
| **memory** | 内存资源 | Ki, Mi, Gi, Ti |
| **ephemeral-storage** | 临时存储 | Ki, Mi, Gi, Ti |
| **hugepages-<size>** | 大页内存 | Ki, Mi, Gi |

### CPU 资源单位

```yaml
# CPU 单位说明
# 1 CPU = 1000m (millicore)
# 0.5 CPU = 500m
# 0.1 CPU = 100m

resources:
  requests:
    cpu: "250m"    # 0.25 CPU
  limits:
    cpu: "500m"    # 0.5 CPU
```

### 内存资源单位

```yaml
# 内存单位说明
# 1 Ki = 1024 bytes
# 1 Mi = 1024 Ki
# 1 Gi = 1024 Mi

resources:
  requests:
    memory: "256Mi"
  limits:
    memory: "512Mi"
```

## Requests 和 Limits 概念

### Requests（资源请求）

- **定义**：容器启动所需的最小资源量
- **作用**：用于调度决策，决定 Pod 可以调度到哪些节点
- **影响**：节点必须有足够的可用资源才能调度 Pod

### Limits（资源限制）

- **定义**：容器可以使用的最大资源量
- **作用**：限制容器的资源使用上限
- **影响**：超过限制会触发 OOM 或 CPU 节流

### Requests 与 Limits 的关系

```
┌─────────────────────────────────────────────────────────────┐
│                Requests 与 Limits 关系                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    Limits (上限)                     │   │
│  │  ┌─────────────────────────────────────────────┐   │   │
│  │  │           实际使用量                         │   │   │
│  │  │  ┌─────────────────────────────────────┐   │   │   │
│  │  │  │         Requests (保证量)           │   │   │   │
│  │  │  └─────────────────────────────────────┘   │   │   │
│  │  └─────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Requests <= 实际使用量 <= Limits                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 资源配置详解

### 基本配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: resource-demo
spec:
  containers:
  - name: app
    image: nginx:1.21
    resources:
      requests:
        cpu: "250m"
        memory: "256Mi"
        ephemeral-storage: "1Gi"
      limits:
        cpu: "500m"
        memory: "512Mi"
        ephemeral-storage: "2Gi"
```

### 多容器 Pod 配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: multi-container
spec:
  containers:
  - name: app
    image: my-app:v1
    resources:
      requests:
        cpu: "500m"
        memory: "512Mi"
      limits:
        cpu: "1000m"
        memory: "1Gi"
  - name: sidecar
    image: log-collector:v1
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
      limits:
        cpu: "200m"
        memory: "256Mi"
```

### Deployment 配置

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
      - name: web
        image: nginx:1.21
        resources:
          requests:
            cpu: "250m"
            memory: "256Mi"
          limits:
            cpu: "500m"
            memory: "512Mi"
```

## QoS 服务质量等级

### QoS 等级分类

| QoS 等级 | 条件 | 优先级 | 说明 |
|---------|------|--------|------|
| **Guaranteed** | 所有容器都设置了 requests 和 limits，且 requests = limits | 最高 | 资源有保证 |
| **Burstable** | 至少一个容器设置了 requests 或 limits | 中等 | 资源有下限 |
| **BestEffort** | 没有设置任何 requests 和 limits | 最低 | 无资源保证 |

### Guaranteed 配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: guaranteed-pod
spec:
  containers:
  - name: app
    image: nginx
    resources:
      requests:
        cpu: "500m"
        memory: "512Mi"
      limits:
        cpu: "500m"      # requests = limits
        memory: "512Mi"  # requests = limits
```

### Burstable 配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: burstable-pod
spec:
  containers:
  - name: app
    image: nginx
    resources:
      requests:
        cpu: "250m"
        memory: "256Mi"
      limits:
        cpu: "500m"      # limits > requests
        memory: "512Mi"  # limits > requests
```

### BestEffort 配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: besteffort-pod
spec:
  containers:
  - name: app
    image: nginx
    # 没有设置任何 resources
```

### QoS 与 OOM 关系

```
┌─────────────────────────────────────────────────────────────┐
│                    OOM 优先级                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  节点内存不足时，按以下顺序驱逐：                            │
│                                                              │
│  1. BestEffort Pod（最先被驱逐）                            │
│  2. Burstable Pod（超出 requests 部分）                     │
│  3. Guaranteed Pod（最后被驱逐）                            │
│                                                              │
│  OOM Score:                                                 │
│  - Guaranteed: 0 (最低)                                     │
│  - Burstable: 2-1000                                        │
│  - BestEffort: 1000 (最高)                                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## CPU 资源管理

### CPU 请求

```yaml
resources:
  requests:
    cpu: "500m"  # 保证 0.5 CPU
```

**调度影响**：
- 节点必须有 500m 可用 CPU
- CPU 请求用于调度决策
- 多个 Pod 可以共享 CPU（通过时间片）

### CPU 限制

```yaml
resources:
  limits:
    cpu: "1000m"  # 最多使用 1 CPU
```

**运行时影响**：
- 使用 CFS quota 限制 CPU 使用
- 超过限制会被节流，不会杀死容器
- 容器可以短暂超过 limits（如果节点有空闲）

### CPU 节流示例

```bash
# 查看容器 CPU 使用
kubectl top pod <pod-name>

# 查看容器 CPU 节流
cat /sys/fs/cgroup/cpu/kubepods/burstable/pod<uid>/cpu.stat
```

## 内存资源管理

### 内存请求

```yaml
resources:
  requests:
    memory: "512Mi"  # 保证 512Mi 内存
```

**调度影响**：
- 节点必须有 512Mi 可用内存
- 内存请求用于调度决策

### 内存限制

```yaml
resources:
  limits:
    memory: "1Gi"  # 最多使用 1Gi 内存
```

**运行时影响**：
- 超过限制会触发 OOM Killed
- 容器会被强制终止
- 可能导致数据丢失

### OOM 处理

```bash
# 查看 OOM 事件
kubectl describe pod <pod-name> | grep OOM

# 查看容器退出码
kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[0].lastState.terminated.exitCode}'

# OOM Killed 退出码为 137
```

## LimitRange 默认配置

### LimitRange 概述

LimitRange 可以为 Namespace 设置默认的资源请求和限制：

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
  namespace: default
spec:
  limits:
  - type: Container
    default:              # 默认 limits
      cpu: "500m"
      memory: "512Mi"
    defaultRequest:       # 默认 requests
      cpu: "250m"
      memory: "256Mi"
    min:                  # 最小值
      cpu: "50m"
      memory: "64Mi"
    max:                  # 最大值
      cpu: "2"
      memory: "4Gi"
    maxLimitRequestRatio: # limits/requests 最大比例
      cpu: "4"
      memory: "2"
```

### LimitRange 类型

| 类型 | 说明 |
|-----|------|
| **Container** | 容器级别限制 |
| **Pod** | Pod 级别限制 |
| **PersistentVolumeClaim** | PVC 存储限制 |

## ResourceQuota 配额管理

### ResourceQuota 概述

ResourceQuota 限制 Namespace 的资源使用总量：

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-quota
  namespace: development
spec:
  hard:
    requests.cpu: "10"
    requests.memory: "20Gi"
    limits.cpu: "20"
    limits.memory: "40Gi"
    pods: "50"
    services: "10"
    secrets: "20"
    configmaps: "20"
    persistentvolumeclaims: "10"
```

### 查看配额使用

```bash
# 查看配额
kubectl get resourcequota -n development

# 查看详细信息
kubectl describe resourcequota compute-quota -n development
```

## 资源监控

### 查看资源使用

```bash
# 查看节点资源
kubectl top nodes

# 查看 Pod 资源
kubectl top pods -n default

# 查看容器资源
kubectl top pod <pod-name> --containers
```

### Prometheus 监控指标

```yaml
# CPU 使用率
rate(container_cpu_usage_seconds_total[5m])

# 内存使用
container_memory_working_set_bytes

# CPU 限制
kube_pod_container_resource_limits{resource="cpu"}

# 内存限制
kube_pod_container_resource_limits{resource="memory"}
```

### 告警规则

```yaml
groups:
- name: resource-alerts
  rules:
  - alert: PodCPUThrottling
    expr: |
      rate(container_cpu_cfs_throttled_seconds_total[5m]) 
      / rate(container_cpu_cfs_periods_total[5m]) > 0.5
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Pod {{ $labels.pod }} CPU throttled"
      
  - alert: PodMemoryHigh
    expr: |
      container_memory_working_set_bytes 
      / kube_pod_container_resource_limits{resource="memory"} > 0.9
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Pod {{ $labels.pod }} memory usage > 90%"
```

## 最佳实践

### 1. 合理设置资源

```yaml
resources:
  requests:
    # 基于实际使用设置，略高于平均值
    cpu: "250m"
    memory: "256Mi"
  limits:
    # 设置为 requests 的 2-4 倍
    cpu: "500m"
    memory: "512Mi"
```

### 2. 使用 HPA 自动扩缩

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: web-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: web
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 3. 生产环境推荐配置

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: production-app
spec:
  template:
    spec:
      containers:
      - name: app
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "1000m"
            memory: "1Gi"
```

### 4. 测试环境推荐配置

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  template:
    spec:
      containers:
      - name: app
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "200m"
            memory: "256Mi"
```

## 常见问题

### Q1: 如何确定合适的资源配置？

```bash
# 查看历史资源使用
kubectl top pod <pod-name> --containers

# 使用 VPA 推荐
kubectl get vpa <vpa-name> -o yaml
```

### Q2: Pod 一直 Pending 怎么办？

```bash
# 查看调度失败原因
kubectl describe pod <pod-name>

# 检查节点资源
kubectl describe nodes | grep -A 5 "Allocated resources"
```

### Q3: 如何处理 OOM Killed？

```bash
# 查看事件
kubectl describe pod <pod-name>

# 增加内存限制
# 或优化应用内存使用
```

## 面试回答

**问题**: Pod 的资源请求限制如何定义？

**回答**: Kubernetes 通过 `resources` 字段定义 Pod 的资源请求和限制。**Requests（请求）** 定义容器启动所需的最小资源量，用于调度决策，节点必须有足够的可用资源才能调度 Pod。**Limits（限制）** 定义容器可以使用的最大资源量，超过限制会触发 CPU 节流或 OOM Killed。

资源配置示例：
```yaml
resources:
  requests:
    cpu: "250m"      # 0.25 CPU
    memory: "256Mi"  # 256 MiB
  limits:
    cpu: "500m"      # 0.5 CPU
    memory: "512Mi"  # 512 MiB
```

CPU 资源单位是 millicore，1000m = 1 CPU。内存单位支持 Ki、Mi、Gi。CPU 超过 limits 会被节流，不会杀死容器；内存超过 limits 会触发 OOM Killed，容器会被强制终止。

根据资源配置，Pod 分为三个 QoS 等级：**Guaranteed** 是最高优先级，所有容器的 requests = limits；**Burstable** 是中等优先级，至少设置了 requests 或 limits；**BestEffort** 是最低优先级，没有设置任何资源配置。节点资源不足时，按 BestEffort -> Burstable -> Guaranteed 的顺序驱逐。

生产环境建议：requests 基于实际使用设置，limits 设置为 requests 的 2-4 倍，配合 LimitRange 设置默认值，配合 ResourceQuota 限制命名空间资源总量，配合 HPA 实现自动扩缩容。

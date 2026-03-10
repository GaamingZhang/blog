---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - 资源管理
  - 性能优化
---

# Kubernetes requests和limits详解

## 概述

在Kubernetes中，`requests`和`limits`是资源管理的核心概念，用于控制Pod对CPU和内存资源的使用。正确理解和配置这两个参数对于集群的稳定运行至关重要。

## requests和limits的作用

```
+------------------+----------------------------------------+
|      参数        |                 作用                   |
+------------------+----------------------------------------+
| requests         | 容器启动所需的最小资源量               |
|                  | 用于调度决策，保证资源预留             |
+------------------+----------------------------------------+
| limits           | 容器能使用的最大资源量                 |
|                  | 用于资源限制，防止资源滥用             |
+------------------+----------------------------------------+
```

## 资源类型

### CPU资源

```yaml
resources:
  requests:
    cpu: "500m"    # 0.5个CPU核心
  limits:
    cpu: "1000m"   # 1个CPU核心

# CPU单位说明：
# 1 = 1个CPU核心（1000m）
# 500m = 0.5个CPU核心
# 100m = 0.1个CPU核心
```

### 内存资源

```yaml
resources:
  requests:
    memory: "256Mi"   # 256 MiB
  limits:
    memory: "512Mi"   # 512 MiB

# 内存单位说明：
# Ki = KiB (1024 bytes)
# Mi = MiB (1024 * 1024 bytes)
# Gi = GiB (1024 * 1024 * 1024 bytes)
```

## 工作原理

### 调度阶段

```
+------------------+     +------------------+     +------------------+
|   Pod创建请求    |     |   Scheduler      |     |   Node选择       |
+------------------+     +------------------+     +------------------+
         |                       |                        |
         v                       v                        v
+------------------+     +------------------+     +------------------+
| 读取requests     |---->| 计算Node可用资源 |---->| 选择满足requests |
| 资源需求         |     | 是否足够         |     | 的Node调度       |
+------------------+     +------------------+     +------------------+
```

### 运行阶段

```
+------------------+
|   容器运行时     |
+------------------+
         |
         v
+------------------+     +------------------+
| CPU资源使用      |     | 内存资源使用     |
+------------------+     +------------------+
| < requests: 正常 |     | < limits: 正常   |
| > limits: 限流   |     | > limits: OOMKilled|
+------------------+     +------------------+
```

## 配置示例

### 基础配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: resource-demo
spec:
  containers:
  - name: app
    image: nginx
    resources:
      requests:
        cpu: "250m"
        memory: "64Mi"
      limits:
        cpu: "500m"
        memory: "128Mi"
```

### 多容器配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: multi-container
spec:
  containers:
  - name: app
    image: myapp
    resources:
      requests:
        cpu: "500m"
        memory: "256Mi"
      limits:
        cpu: "1000m"
        memory: "512Mi"
  - name: sidecar
    image: log-collector
    resources:
      requests:
        cpu: "100m"
        memory: "64Mi"
      limits:
        cpu: "200m"
        memory: "128Mi"
```

### Deployment配置

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
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "500m"
            memory: "256Mi"
```

## QoS服务质量等级

Kubernetes根据requests和limits的配置，将Pod分为三个QoS等级：

```
+------------------+----------------------------------------+
|    QoS等级       |              配置条件                  |
+------------------+----------------------------------------+
| Guaranteed       | 所有容器都设置了CPU和内存的           |
| (最高优先级)     | requests和limits，且两者相等          |
+------------------+----------------------------------------+
| Burstable        | 至少一个容器设置了requests或limits    |
| (中等优先级)     | 但不满足Guaranteed条件                |
+------------------+----------------------------------------+
| BestEffort       | 没有设置任何requests和limits          |
| (最低优先级)     |                                       |
+------------------+----------------------------------------+
```

### QoS配置示例

```yaml
# Guaranteed - 最高优先级
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
        memory: "256Mi"
      limits:
        cpu: "500m"      # 与requests相等
        memory: "256Mi"  # 与requests相等

---
# Burstable - 中等优先级
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
        memory: "128Mi"
      limits:
        cpu: "500m"      # limits > requests
        memory: "256Mi"

---
# BestEffort - 最低优先级
apiVersion: v1
kind: Pod
metadata:
  name: besteffort-pod
spec:
  containers:
  - name: app
    image: nginx
    # 没有设置resources
```

## 资源配额（ResourceQuota）

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-quota
  namespace: development
spec:
  hard:
    requests.cpu: "4"        # 总CPU requests上限
    requests.memory: 8Gi     # 总内存requests上限
    limits.cpu: "8"          # 总CPU limits上限
    limits.memory: 16Gi      # 总内存limits上限
    pods: "10"               # Pod数量上限
```

## LimitRange默认值

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
  namespace: default
spec:
  limits:
  - default:              # 默认limits
      cpu: "500m"
      memory: "256Mi"
    defaultRequest:       # 默认requests
      cpu: "100m"
      memory: "64Mi"
    type: Container
```

## 监控与排查

### 查看Pod资源使用

```bash
# 查看Pod资源使用
kubectl top pod

# 查看Node资源使用
kubectl top node

# 查看Pod详细信息
kubectl describe pod <pod-name> | grep -A 5 "Containers:"
```

### 查看资源配额使用

```bash
# 查看ResourceQuota
kubectl get resourcequota -n <namespace>

# 查看LimitRange
kubectl get limitrange -n <namespace>
```

## 最佳实践

### 1. 合理设置requests

```
requests设置建议：
- 基于实际使用量设置
- 参考历史监控数据
- 预留一定缓冲空间
- 过大会浪费资源，过小会导致调度失败
```

### 2. 合理设置limits

```
limits设置建议：
- limits >= requests
- 避免limits过大导致资源竞争
- CPU limits可适当放宽
- 内存limits需要严格控制
```

### 3. QoS选择建议

```
+------------------+----------------------------------------+
|    场景          |              建议QoS                   |
+------------------+----------------------------------------+
| 关键服务         | Guaranteed                             |
| 普通服务         | Burstable                              |
| 批处理任务       | Burstable                              |
| 测试环境         | BestEffort                             |
+------------------+----------------------------------------+
```

## 参考资源

- [Kubernetes官方文档 - 管理容器的计算资源](https://kubernetes.io/zh/docs/concepts/configuration/manage-resources-containers/)
- [Kubernetes官方文档 - 资源服务质量](https://kubernetes.io/zh/docs/concepts/workloads/pods/pod-qos/)

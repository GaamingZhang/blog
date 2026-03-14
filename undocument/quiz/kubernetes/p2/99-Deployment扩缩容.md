---
date: 2026-03-12
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Deployment
  - 扩缩容
---

# Kubernetes Deployment 扩缩容详解

## 引言

在 Kubernetes 集群中，应用的扩容和缩容是日常运维中最常见的操作之一。无论是应对流量高峰需要快速扩容，还是在低峰期节省资源需要缩容，Deployment 都提供了灵活的扩缩容机制。

理解 Deployment 的扩缩容原理和操作方法，对于实现应用的弹性伸缩和资源优化至关重要。本文将深入剖析 Deployment 的手动扩缩容、自动扩缩容（HPA）以及相关的最佳实践。

## Deployment 扩缩容概述

### 扩缩容类型

| 类型 | 说明 | 触发方式 |
|-----|------|---------|
| **手动扩缩容** | 手动修改副本数 | kubectl scale |
| **自动扩缩容** | 基于指标自动调整 | HPA |
| **定时扩缩容** | 按时间计划调整 | CronHPA |

### 扩缩容原理

```
┌─────────────────────────────────────────────────────────────┐
│                  Deployment 扩缩容原理                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Deployment Controller                   │   │
│  │  - 监控 Pod 数量                                     │   │
│  │  - 对比期望副本数与实际副本数                        │   │
│  │  - 创建/删除 Pod 以达到期望状态                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  扩容流程：                                                  │
│  期望副本数 > 实际副本数 -> 创建新 Pod                      │
│                                                              │
│  缩容流程：                                                  │
│  期望副本数 < 实际副本数 -> 删除多余 Pod                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 手动扩缩容

### kubectl scale 命令

#### 基本扩容

```bash
# 将副本数扩展到 5 个
kubectl scale deployment my-app --replicas=5

# 查看扩容结果
kubectl get deployment my-app
kubectl get pods -l app=my-app
```

#### 条件扩容

```bash
# 仅当当前副本数为 3 时才扩容
kubectl scale deployment my-app --current-replicas=3 --replicas=5
```

#### 批量扩容

```bash
# 扩容多个 Deployment
kubectl scale deployment my-app1 my-app2 --replicas=5

# 扩容指定文件中的所有 Deployment
kubectl scale -f deployment.yaml --replicas=5
```

### 修改 YAML 文件

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 5  # 修改副本数
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: app
        image: nginx:1.21
```

```bash
# 应用修改
kubectl apply -f deployment.yaml
```

### kubectl edit 命令

```bash
# 直接编辑 Deployment
kubectl edit deployment my-app

# 在编辑器中修改 spec.replicas 字段
# 保存后自动生效
```

### kubectl patch 命令

```bash
# 使用 patch 修改副本数
kubectl patch deployment my-app -p '{"spec":{"replicas":5}}'
```

## 自动扩缩容（HPA）

### HPA 概述

Horizontal Pod Autoscaler（HPA）根据 CPU 使用率、内存使用率或自定义指标自动调整 Pod 副本数。

```
┌─────────────────────────────────────────────────────────────┐
│                    HPA 工作原理                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Metrics Server                          │   │
│  │  - 收集 Pod 资源使用指标                            │   │
│  │  - 提供 metrics.k8s.io API                          │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         │ 获取指标                          │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              HPA Controller                          │   │
│  │  - 计算期望副本数                                   │   │
│  │  - 比较当前指标与目标指标                           │   │
│  │  - 调整 Deployment 副本数                           │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         │ 更新副本数                        │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Deployment                              │   │
│  │  - 创建/删除 Pod                                    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### HPA 计算公式

```
期望副本数 = ceil[当前副本数 * (当前指标值 / 目标指标值)]
```

### 基于 CPU 的 HPA

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: my-app-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### 基于内存的 HPA

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: my-app-hpa-memory
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 多指标 HPA

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: my-app-hpa-multi
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
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

### 自定义指标 HPA

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: my-app-hpa-custom
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Pods
    pods:
      metric:
        name: requests_per_second
      target:
        type: AverageValue
        averageValue: "1000"
```

### HPA 行为配置

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: my-app-hpa-behavior
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
      - type: Pods
        value: 2
        periodSeconds: 60
      selectPolicy: Min
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
      - type: Pods
        value: 4
        periodSeconds: 15
      selectPolicy: Max
```

### HPA 操作命令

```bash
# 创建 HPA
kubectl apply -f hpa.yaml

# 查看 HPA 状态
kubectl get hpa

# 查看 HPA 详情
kubectl describe hpa my-app-hpa

# 查看 HPA 事件
kubectl get events --field-selector involvedObject.name=my-app-hpa

# 删除 HPA
kubectl delete hpa my-app-hpa
```

## 扩缩容策略

### 扩容策略

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1        # 扩容时最多可以多创建的 Pod 数
      maxUnavailable: 0  # 扩容时最多不可用的 Pod 数
```

### 缩容策略

```yaml
# Pod Disruption Budget 限制缩容速度
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: my-app-pdb
spec:
  minAvailable: 2  # 至少保持 2 个 Pod 可用
  selector:
    matchLabels:
      app: my-app
```

## 扩缩容最佳实践

### 1. 配置资源请求和限制

```yaml
resources:
  requests:
    cpu: "250m"
    memory: "256Mi"
  limits:
    cpu: "500m"
    memory: "512Mi"
```

### 2. 配置就绪探针

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### 3. 设置合理的 HPA 参数

```yaml
spec:
  minReplicas: 2        # 至少 2 个副本保证高可用
  maxReplicas: 10       # 设置合理的上限
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70  # CPU 使用率 70% 时触发扩容
```

### 4. 配置冷却时间

```yaml
behavior:
  scaleDown:
    stabilizationWindowSeconds: 300  # 缩容冷却时间 5 分钟
  scaleUp:
    stabilizationWindowSeconds: 60   # 扩容冷却时间 1 分钟
```

### 5. 监控扩缩容事件

```yaml
# Prometheus 告警规则
- alert: HPAScalingActive
  expr: rate(kube_hpa_status_condition[5m]) > 0
  for: 1m
  labels:
    severity: info
  annotations:
    summary: "HPA {{ $labels.hpa }} is scaling"
```

## 常见问题排查

### HPA 无法获取指标

```bash
# 检查 Metrics Server 是否运行
kubectl get pods -n kube-system -l k8s-app=metrics-server

# 检查 Metrics Server 是否可用
kubectl top nodes
kubectl top pods

# 查看 Metrics Server 日志
kubectl logs -n kube-system <metrics-server-pod>
```

### HPA 扩容不生效

```bash
# 检查 HPA 状态
kubectl describe hpa my-app-hpa

# 检查 Pod 资源配置
kubectl get pod <pod-name> -o jsonpath='{.spec.containers[0].resources}'

# 检查是否达到 maxReplicas
kubectl get hpa my-app-hpa
```

### 扩容后 Pod 无法启动

```bash
# 检查节点资源
kubectl describe nodes

# 检查 Pod 事件
kubectl describe pod <pod-name>

# 检查资源配额
kubectl get resourcequota -n <namespace>
```

## 面试回答

**问题**: Deployment 怎么扩容或缩容？

**回答**: Kubernetes Deployment 支持手动扩缩容和自动扩缩容两种方式。

**手动扩缩容**有几种方法：使用 `kubectl scale deployment <name> --replicas=<n>` 命令直接修改副本数；使用 `kubectl edit deployment <name>` 编辑 Deployment 的 `spec.replicas` 字段；使用 `kubectl patch deployment <name> -p '{"spec":{"replicas":<n>}}'` 命令；或者修改 YAML 文件后使用 `kubectl apply` 应用。

**自动扩缩容**通过 Horizontal Pod Autoscaler（HPA）实现。HPA 根据 CPU 使用率、内存使用率或自定义指标自动调整 Pod 副本数。HPA 的计算公式是：期望副本数 = ceil[当前副本数 × (当前指标值 / 目标指标值)]。配置 HPA 需要指定最小副本数、最大副本数和目标指标值。

HPA 的关键配置包括：`minReplicas` 和 `maxReplicas` 设置副本数范围；`metrics` 定义扩缩容的触发条件；`behavior` 配置扩缩容行为，如冷却时间、缩容策略等。生产环境建议配置资源请求和限制、就绪探针、Pod Disruption Budget，并设置合理的 HPA 参数和冷却时间，避免频繁扩缩容导致服务不稳定。

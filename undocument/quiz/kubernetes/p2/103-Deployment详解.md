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
  - 工作负载
---

# Kubernetes Deployment 详解

## 引言

Deployment 是 Kubernetes 中最常用的工作负载控制器，它管理 ReplicaSet，提供声明式的应用更新能力。通过 Deployment，可以轻松实现应用的部署、滚动更新、回滚和扩缩容。

理解 Deployment 的工作原理和配置方法，是掌握 Kubernetes 应用管理的关键。本文将深入剖析 Deployment 的概念、工作原理、更新策略和最佳实践。

## Deployment 概述

### 什么是 Deployment

Deployment 是 Kubernetes 中的一种控制器，用于管理 Pod 和 ReplicaSet。它提供了声明式的应用更新能力，支持滚动更新、回滚等高级功能。

### Deployment 架构

```
┌─────────────────────────────────────────────────────────────┐
│                  Deployment 架构                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    Deployment                        │   │
│  │  - 管理应用版本                                      │   │
│  │  - 支持滚动更新                                      │   │
│  │  - 支持回滚                                          │   │
│  │  - 维护期望状态                                      │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         │ 管理                              │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   ReplicaSet                         │   │
│  │  - 维护 Pod 副本数                                   │   │
│  │  - 每个版本一个 ReplicaSet                          │   │
│  │  - 由 Deployment 管理                               │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         │ 创建                              │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                      Pod                             │   │
│  │  - 运行容器                                          │   │
│  │  - 由 ReplicaSet 管理                               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Deployment 的作用

| 功能 | 说明 |
|-----|------|
| **部署应用** | 创建 Pod 和 ReplicaSet |
| **滚动更新** | 无缝更新应用版本 |
| **回滚** | 回退到历史版本 |
| **扩缩容** | 调整 Pod 副本数 |
| **自愈** | 自动恢复失败的 Pod |

## Deployment 配置

### 基本配置

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  labels:
    app: my-app
spec:
  replicas: 3
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
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "200m"
            memory: "256Mi"
        livenessProbe:
          httpGet:
            path: /health
            port: 80
        readinessProbe:
          httpGet:
            path: /ready
            port: 80
```

### 配置字段说明

| 字段 | 说明 |
|-----|------|
| **replicas** | 期望的 Pod 副本数量 |
| **selector** | 标签选择器，匹配 Pod |
| **template** | Pod 模板 |
| **strategy** | 更新策略 |
| **revisionHistoryLimit** | 保留的历史版本数 |

## 更新策略

### RollingUpdate（滚动更新）

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
      maxSurge: 1        # 最多可以多创建的 Pod 数
      maxUnavailable: 0  # 最多不可用的 Pod 数
  template:
    spec:
      containers:
      - name: app
        image: nginx:1.21
```

### Recreate（重建更新）

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 3
  strategy:
    type: Recreate
  template:
    spec:
      containers:
      - name: app
        image: nginx:1.21
```

### 策略对比

| 策略 | 说明 | 优点 | 缺点 |
|-----|------|------|------|
| **RollingUpdate** | 滚动更新 | 零停机 | 可能同时运行两个版本 |
| **Recreate** | 先删后建 | 版本一致 | 有停机时间 |

### RollingUpdate 参数详解

```
┌─────────────────────────────────────────────────────────────┐
│                  RollingUpdate 参数                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  maxSurge: 1                                                │
│  maxUnavailable: 0                                          │
│  replicas: 3                                                │
│                                                              │
│  更新过程：                                                  │
│                                                              │
│  初始状态：3 个旧 Pod                                        │
│  ┌───────┐ ┌───────┐ ┌───────┐                             │
│  │ Pod 1 │ │ Pod 2 │ │ Pod 3 │  (v1)                       │
│  └───────┘ └───────┘ └───────┘                             │
│                                                              │
│  步骤 1：创建 1 个新 Pod（maxSurge=1）                       │
│  ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐                   │
│  │ Pod 1 │ │ Pod 2 │ │ Pod 3 │ │ Pod 4 │  (v1+v2)         │
│  └───────┘ └───────┘ └───────┘ └───────┘                   │
│                                                              │
│  步骤 2：Pod 4 就绪后，删除 1 个旧 Pod                       │
│  ┌───────┐ ┌───────┐ ┌───────┐                             │
│  │ Pod 2 │ │ Pod 3 │ │ Pod 4 │  (v1+v2)                    │
│  └───────┘ └───────┘ └───────┘                             │
│                                                              │
│  步骤 3：创建新 Pod                                          │
│  ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐                   │
│  │ Pod 2 │ │ Pod 3 │ │ Pod 4 │ │ Pod 5 │  (v1+v2)         │
│  └───────┘ └───────┘ └───────┘ └───────┘                   │
│                                                              │
│  最终状态：3 个新 Pod                                        │
│  ┌───────┐ ┌───────┐ ┌───────┐                             │
│  │ Pod 4 │ │ Pod 5 │ │ Pod 6 │  (v2)                       │
│  └───────┘ └───────┘ └───────┘                             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 滚动更新

### 触发更新

```bash
# 方式1：修改镜像
kubectl set image deployment/my-app app=nginx:1.22

# 方式2：修改配置
kubectl edit deployment my-app

# 方式3：应用新配置
kubectl apply -f deployment.yaml
```

### 查看更新状态

```bash
# 查看更新状态
kubectl rollout status deployment/my-app

# 查看 ReplicaSet
kubectl get rs -l app=my-app

# 查看事件
kubectl describe deployment my-app
```

### 更新过程

```bash
# 查看 ReplicaSet 变化
kubectl get rs -w

# 输出示例
NAME                DESIRED   CURRENT   READY   AGE
my-app-6b7f9c8d4    3         3         3       10m
my-app-7c8g9d9e5    0         0         0       0s
my-app-7c8g9d9e5    1         0         0       0s
my-app-7c8g9d9e5    1         1         0       1s
my-app-6b7f9c8d4    2         3         3       1m
my-app-7c8g9d9e5    2         1         1       1s
...
```

## 回滚

### 查看历史版本

```bash
# 查看历史版本
kubectl rollout history deployment/my-app

# 查看特定版本详情
kubectl rollout history deployment/my-app --revision=2
```

### 回滚到上一版本

```bash
# 回滚到上一版本
kubectl rollout undo deployment/my-app
```

### 回滚到指定版本

```bash
# 回滚到指定版本
kubectl rollout undo deployment/my-app --to-revision=2
```

### 配置历史版本数量

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  revisionHistoryLimit: 10  # 保留 10 个历史版本
  template:
    spec:
      containers:
      - name: app
        image: nginx:1.21
```

## 暂停和恢复更新

### 暂停更新

```bash
# 暂停更新
kubectl rollout pause deployment/my-app

# 进行多次修改
kubectl set image deployment/my-app app=nginx:1.22
kubectl set resources deployment/my-app -c app --limits=cpu=200m,memory=256Mi
```

### 恢复更新

```bash
# 恢复更新
kubectl rollout resume deployment/my-app
```

## 扩缩容

### 手动扩缩容

```bash
# 扩容到 5 个副本
kubectl scale deployment/my-app --replicas=5

# 条件扩容
kubectl scale deployment/my-app --current-replicas=3 --replicas=5
```

### 自动扩缩容

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

## Deployment 状态

### 状态字段

```yaml
status:
  replicas: 3                # 当前副本数
  updatedReplicas: 3         # 已更新副本数
  readyReplicas: 3           # 就绪副本数
  availableReplicas: 3       # 可用副本数
  unavailableReplicas: 0     # 不可用副本数
  conditions:                # 状态条件
  - type: Progressing
    status: "True"
  - type: Available
    status: "True"
```

### 状态条件

| 条件 | 说明 |
|-----|------|
| **Progressing** | 正在进行更新 |
| **Available** | 服务可用 |
| **ReplicaFailure** | 副本创建失败 |

## Deployment 操作命令

### 创建

```bash
# 创建 Deployment
kubectl create deployment my-app --image=nginx:1.21 --replicas=3

# 从 YAML 创建
kubectl apply -f deployment.yaml
```

### 查看

```bash
# 查看 Deployment
kubectl get deployments

# 查看详情
kubectl describe deployment my-app

# 查看 YAML
kubectl get deployment my-app -o yaml
```

### 更新

```bash
# 更新镜像
kubectl set image deployment/my-app app=nginx:1.22

# 更新资源
kubectl set resources deployment/my-app -c app --limits=cpu=200m,memory=256Mi

# 应用更新
kubectl apply -f deployment.yaml
```

### 删除

```bash
# 删除 Deployment
kubectl delete deployment my-app

# 从文件删除
kubectl delete -f deployment.yaml
```

## 最佳实践

### 1. 配置资源限制

```yaml
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "200m"
    memory: "256Mi"
```

### 2. 配置健康检查

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 80
  initialDelaySeconds: 30
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /ready
    port: 80
  initialDelaySeconds: 5
  periodSeconds: 5
```

### 3. 配置优雅终止

```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 15"]
terminationGracePeriodSeconds: 60
```

### 4. 使用标签和注解

```yaml
metadata:
  labels:
    app: my-app
    version: v1
    environment: production
  annotations:
    description: "My application deployment"
```

### 5. 配置 Pod Disruption Budget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: my-app-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: my-app
```

## 常见问题排查

### Pod 无法启动

```bash
# 查看 Pod 状态
kubectl get pods -l app=my-app

# 查看 Pod 事件
kubectl describe pod <pod-name>

# 查看日志
kubectl logs <pod-name>
```

### 更新卡住

```bash
# 查看更新状态
kubectl rollout status deployment/my-app

# 查看 Deployment 事件
kubectl describe deployment my-app

# 查看 ReplicaSet
kubectl get rs -l app=my-app
```

### 回滚失败

```bash
# 查看历史版本
kubectl rollout history deployment/my-app

# 查看特定版本
kubectl rollout history deployment/my-app --revision=2
```

## 面试回答

**问题**: 什么是 Deployment？

**回答**: Deployment 是 Kubernetes 中最常用的工作负载控制器，它管理 ReplicaSet，提供声明式的应用更新能力。Deployment 的核心功能包括：部署应用、滚动更新、回滚、扩缩容和自愈。

Deployment 与 ReplicaSet 的关系是：Deployment 管理 ReplicaSet，每个 Deployment 版本对应一个 ReplicaSet。当更新 Deployment 时，会创建新的 ReplicaSet，逐步将 Pod 从旧 ReplicaSet 迁移到新 ReplicaSet，实现滚动更新。

Deployment 支持两种更新策略：**RollingUpdate**（滚动更新）是默认策略，支持零停机更新，通过 maxSurge 和 maxUnavailable 参数控制更新速度；**Recreate**（重建更新）先删除旧 Pod 再创建新 Pod，会有短暂停机，但保证同一时刻只有一个版本运行。

Deployment 的关键配置包括：replicas 指定副本数，selector 定义标签选择器，template 定义 Pod 模板，strategy 定义更新策略，revisionHistoryLimit 保留历史版本数。

生产环境最佳实践：配置资源限制和健康检查、设置合理的更新策略参数、配置 Pod Disruption Budget 保证服务可用性、使用 HPA 实现自动扩缩容、保留足够的历史版本用于回滚。

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
  - 调度
---

# Kubernetes Pod 调度方式详解

## 引言

在 Kubernetes 集群中，Pod 调度是一个核心功能。当一个 Pod 被创建时，Kubernetes 调度器需要决定将这个 Pod 运行在哪个节点上。调度决策需要考虑多种因素，包括资源需求、硬件约束、亲和性规则、污点容忍等。

理解 Kubernetes 的调度机制和各种调度方式，对于优化集群资源利用率、提高应用可用性和实现精细化部署控制至关重要。本文将深入剖析 Kubernetes 的各种调度方式及其应用场景。

## Kubernetes 调度概述

### 调度器工作原理

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes 调度流程                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐                                            │
│  │  Pod 创建   │                                            │
│  │  (Pending)  │                                            │
│  └──────┬──────┘                                            │
│         │                                                    │
│         ▼                                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              调度器 (kube-scheduler)                 │   │
│  │  ┌─────────────────┐  ┌─────────────────────────┐  │   │
│  │  │    过滤阶段      │  │       打分阶段          │  │   │
│  │  │  (Predicates)   │->│      (Priorities)      │  │   │
│  │  │  找出可用节点    │  │     选择最优节点        │  │   │
│  │  └─────────────────┘  └─────────────────────────┘  │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              绑定 Pod 到节点                          │   │
│  │              kubelet 启动容器                         │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 调度过程

1. **过滤（Predicates）**：排除不满足条件的节点
2. **打分（Priorities）**：对可用节点进行打分
3. **绑定（Binding）**：将 Pod 绑定到得分最高的节点

## 一、nodeName 调度

### 概述

`nodeName` 是最简单的调度方式，直接指定 Pod 运行的节点名称。

### 配置示例

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  nodeName: node-1
  containers:
  - name: nginx
    image: nginx:1.21
    ports:
    - containerPort: 80
```

### 特点

| 优点 | 缺点 |
|-----|------|
| 简单直接 | 绕过调度器 |
| 不依赖调度器 | 节点不存在时 Pod 会一直 Pending |
| 适用于调试 | 不考虑资源限制 |

### 使用场景

- 调试和测试
- 特定节点部署
- 绕过调度器问题

## 二、nodeSelector 调度

### 概述

`nodeSelector` 是最简单的节点选择约束，通过标签选择节点。

### 节点标签管理

```bash
# 添加标签
kubectl label nodes node-1 disktype=ssd
kubectl label nodes node-1 zone=east

# 查看标签
kubectl get nodes --show-labels

# 删除标签
kubectl label nodes node-1 disktype-
```

### 配置示例

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  nodeSelector:
    disktype: ssd
    zone: east
  containers:
  - name: nginx
    image: nginx:1.21
```

### 特点

| 优点 | 缺点 |
|-----|------|
| 简单易用 | 只支持精确匹配 |
| 支持多标签 | 不支持复杂条件 |
| 广泛支持 | 功能有限 |

### 使用场景

- 简单的节点分类
- 硬件类型选择（SSD/HDD）
- 环境隔离

## 三、节点亲和性调度

### 概述

节点亲和性（Node Affinity）提供了更灵活的节点选择方式，支持多种操作符和优先级。

### 类型

1. **requiredDuringSchedulingIgnoredDuringExecution**：硬性要求，必须满足
2. **preferredDuringSchedulingIgnoredDuringExecution**：软性要求，优先满足

### 配置示例

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: disktype
            operator: In
            values:
            - ssd
            - nvme
          - key: zone
            operator: In
            values:
            - east
            - west
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        preference:
          matchExpressions:
          - key: zone
            operator: In
            values:
            - east
      - weight: 50
        preference:
          matchExpressions:
          - key: disktype
            operator: In
            values:
            - nvme
  containers:
  - name: nginx
    image: nginx:1.21
```

### 操作符

| 操作符 | 说明 | 示例 |
|-------|------|------|
| **In** | 标签值在列表中 | `zone In [east, west]` |
| **NotIn** | 标签值不在列表中 | `env NotIn [prod]` |
| **Exists** | 标签存在 | `gpu Exists` |
| **DoesNotExist** | 标签不存在 | `legacy DoesNotExist` |
| **Gt** | 标签值大于指定值 | `cpu Gt 4` |
| **Lt** | 标签值小于指定值 | `memory Lt 16` |

### 特点

| 优点 | 缺点 |
|-----|------|
| 支持复杂条件 | 配置相对复杂 |
| 支持优先级 | 学习成本较高 |
| 支持多种操作符 | |

### 使用场景

- 复杂的节点选择
- 多条件组合
- 优先级调度

## 四、Pod 亲和性与反亲和性

### 概述

Pod 亲和性（Pod Affinity）和反亲和性（Pod Anti-Affinity）允许根据已运行的 Pod 来调度新的 Pod。

### 类型

1. **Pod 亲和性**：将 Pod 调度到与指定 Pod 相同的拓扑域
2. **Pod 反亲和性**：将 Pod 调度到与指定 Pod 不同的拓扑域

### 配置示例

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
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
      affinity:
        podAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app: cache
            topologyKey: kubernetes.io/hostname
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: web
              topologyKey: kubernetes.io/hostname
      containers:
      - name: web
        image: nginx:1.21
```

### 拓扑键

| 拓扑键 | 说明 |
|-------|------|
| **kubernetes.io/hostname** | 节点级别 |
| **topology.kubernetes.io/zone** | 可用区级别 |
| **topology.kubernetes.io/region** | 区域级别 |

### 典型应用场景

#### 1. 服务共置

```yaml
# Web 服务与缓存服务共置
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  template:
    spec:
      affinity:
        podAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app: redis
            topologyKey: kubernetes.io/hostname
```

#### 2. 高可用部署

```yaml
# 同一应用的 Pod 分散到不同节点
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 3
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app: api
            topologyKey: kubernetes.io/hostname
```

#### 3. 跨可用区部署

```yaml
# 跨可用区分散
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 3
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: api
              topologyKey: topology.kubernetes.io/zone
```

## 五、污点和容忍

### 概念

- **污点（Taint）**：标记节点，阻止 Pod 调度
- **容忍（Toleration）**：Pod 的属性，允许调度到有污点的节点

### 污点管理

```bash
# 添加污点
kubectl taint nodes node-1 key=value:NoSchedule

# 查看污点
kubectl describe node node-1 | grep Taints

# 删除污点
kubectl taint nodes node-1 key:NoSchedule-
```

### 污点效果

| 效果 | 说明 |
|-----|------|
| **NoSchedule** | 不调度新 Pod |
| **PreferNoSchedule** | 尽量不调度新 Pod |
| **NoExecute** | 不调度新 Pod，且驱逐已有 Pod |

### 配置示例

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  tolerations:
  - key: "key1"
    operator: "Equal"
    value: "value1"
    effect: "NoSchedule"
  - key: "key2"
    operator: "Exists"
    effect: "NoExecute"
    tolerationSeconds: 3600
  containers:
  - name: nginx
    image: nginx:1.21
```

### 操作符

| 操作符 | 说明 |
|-------|------|
| **Equal** | key 和 value 都必须匹配 |
| **Exists** | 只需 key 存在 |

### 典型应用场景

#### 1. 专用节点

```bash
# 标记 GPU 节点
kubectl taint nodes gpu-node gpu=true:NoSchedule
```

```yaml
# GPU 应用容忍污点
apiVersion: v1
kind: Pod
metadata:
  name: gpu-app
spec:
  tolerations:
  - key: "gpu"
    operator: "Equal"
    value: "true"
    effect: "NoSchedule"
  nodeSelector:
    gpu: "true"
  containers:
  - name: gpu-app
    image: gpu-app:v1
```

#### 2. Master 节点

```bash
# Master 节点默认污点
node-role.kubernetes.io/master:NoSchedule
```

```yaml
# 允许调度到 Master
tolerations:
- key: "node-role.kubernetes.io/master"
  operator: "Exists"
  effect: "NoSchedule"
```

#### 3. 节点故障处理

```yaml
# 设置容忍时间
tolerations:
- key: "node.kubernetes.io/unreachable"
  operator: "Exists"
  effect: "NoExecute"
  tolerationSeconds: 300
- key: "node.kubernetes.io/not-ready"
  operator: "Exists"
  effect: "NoExecute"
  tolerationSeconds: 300
```

## 六、优先级和抢占

### 概念

优先级（Priority）和抢占（Preemption）允许高优先级 Pod 抢占低优先级 Pod 的资源。

### PriorityClass 配置

```yaml
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: high-priority
value: 1000000
globalDefault: false
description: "High priority class for critical applications"
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: medium-priority
value: 100000
globalDefault: true
description: "Medium priority class"
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: low-priority
value: 1000
globalDefault: false
description: "Low priority class for batch jobs"
```

### Pod 使用优先级

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: critical-app
spec:
  priorityClassName: high-priority
  containers:
  - name: app
    image: critical-app:v1
```

### 系统优先级类

| PriorityClass | 值 | 说明 |
|--------------|-----|------|
| system-cluster-critical | 2000000000 | 集群关键组件 |
| system-node-critical | 2000001000 | 节点关键组件 |

## 七、调度器配置

### 自定义调度器

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  schedulerName: my-scheduler
  containers:
  - name: nginx
    image: nginx:1.21
```

### 调度器配置

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
- schedulerName: default-scheduler
  plugins:
    queueSort:
      enabled:
      - name: PrioritySort
    preFilter:
      enabled:
      - name: NodeResourcesFit
    filter:
      enabled:
      - name: NodeResourcesFit
      - name: NodeName
      - name: NodePorts
      - name: TaintToleration
    score:
      enabled:
      - name: NodeResourcesBalancedAllocation
        weight: 1
      - name: NodeResourcesFit
        weight: 1
```

## 八、手动调度

### Binding API

```yaml
apiVersion: v1
kind: Binding
metadata:
  name: nginx
target:
  apiVersion: v1
  kind: Node
  name: node-1
```

```bash
# 手动绑定 Pod 到节点
kubectl create -f binding.yaml
```

## 调度方式对比

| 调度方式 | 灵活性 | 复杂度 | 适用场景 |
|---------|-------|--------|---------|
| **nodeName** | 低 | 低 | 调试、测试 |
| **nodeSelector** | 中 | 低 | 简单分类 |
| **节点亲和性** | 高 | 中 | 复杂条件 |
| **Pod 亲和性** | 高 | 高 | 服务共置 |
| **污点容忍** | 高 | 中 | 专用节点 |
| **优先级** | 中 | 低 | 资源竞争 |

## 最佳实践

### 1. 合理使用标签

```yaml
# 节点标签规划
labels:
  # 硬件类型
  disktype: ssd
  cpu: high
  memory: large
  
  # 网络拓扑
  zone: east
  region: us-east-1
  
  # 环境和用途
  environment: production
  purpose: database
```

### 2. 高可用部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 3
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app: api
            topologyKey: kubernetes.io/hostname
        nodeAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            preference:
              matchExpressions:
              - key: zone
                operator: In
                values:
                - east
                - west
                - north
```

### 3. 资源感知调度

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  containers:
  - name: app
    image: app:v1
    resources:
      requests:
        memory: "1Gi"
        cpu: "500m"
      limits:
        memory: "2Gi"
        cpu: "1000m"
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: memory
            operator: Gt
            values:
            - "4"
```

## 面试回答

**问题**: Kubernetes 中 Pod 的常见调度方式有哪些？

**回答**: Kubernetes 提供了多种 Pod 调度方式。最简单的是 **nodeName**，直接指定节点名称，绕过调度器，适用于调试场景。**nodeSelector** 通过节点标签选择节点，支持简单的等值匹配，适用于简单的节点分类场景。

**节点亲和性（Node Affinity）** 提供了更灵活的节点选择，支持 In、NotIn、Exists、Gt、Lt 等操作符，分为硬性要求和软性偏好两种类型，适用于复杂的节点选择条件。

**Pod 亲和性与反亲和性** 根据已运行的 Pod 来调度新的 Pod。Pod 亲和性用于服务共置，如将 Web 服务调度到缓存服务所在节点；Pod 反亲和性用于高可用部署，如将同一应用的 Pod 分散到不同节点或可用区。

**污点和容忍** 用于专用节点场景。污点标记节点阻止 Pod 调度，容忍允许 Pod 调度到有污点的节点。污点有三种效果：NoSchedule 不调度新 Pod，PreferNoSchedule 尽量不调度，NoExecute 还会驱逐已有 Pod。

**优先级和抢占** 允许高优先级 Pod 抢占低优先级 Pod 的资源，通过 PriorityClass 定义优先级。生产环境通常组合使用这些调度方式，如使用节点亲和性选择硬件类型，使用 Pod 反亲和性实现高可用，使用污点容忍部署到专用节点。

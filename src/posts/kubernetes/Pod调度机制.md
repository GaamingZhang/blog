---
date: 2026-01-30
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# Pod调度机制

## 为什么需要了解Pod调度？

当你创建一个Pod时，Kubernetes需要决定把它放到哪台机器（节点）上运行。这个决策过程就叫做"调度"。

想象你是一家公司的HR，现在来了一个新员工，你需要给他安排工位。你会考虑什么？
- 这个人需要高配电脑吗？（资源需求）
- 他需要坐在研发部附近吗？（亲和性）
- 有些工位是预留给特殊岗位的，他能坐吗？（污点和容忍）

Kubernetes的调度器就是在做类似的事情，只不过它安排的是Pod，分配的是节点。

## 调度的基本流程

调度分为两个阶段：

**第一阶段：过滤（Filtering）**
找出所有"能用"的节点。比如：
- 这个节点的CPU和内存够不够？
- 这个节点有没有Pod要求的标签？
- 这个节点是不是被标记为"不接客"了？

**第二阶段：打分（Scoring）**
在"能用"的节点中选出"最合适"的。比如：
- 哪个节点资源更充裕？
- 哪个节点已经有相关服务在运行（减少网络延迟）？

最后，得分最高的节点胜出，Pod就被安排到那里。

## NodeSelector：最简单的节点选择

NodeSelector是最直接的方式——通过标签匹配节点。

### 场景：GPU任务只能跑在GPU节点上

首先，给有GPU的节点打上标签：
```bash
kubectl label nodes gpu-node-1 accelerator=nvidia
```

然后，在Pod中指定只能调度到有这个标签的节点：
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-task
spec:
  nodeSelector:
    accelerator: nvidia
  containers:
    - name: cuda-app
      image: cuda-app:1.0
```

这就像给新员工说："你只能坐在有双显示器的工位"。简单直接，但不够灵活。

## Node Affinity：更灵活的节点选择

Node Affinity（节点亲和性）是NodeSelector的增强版，支持更复杂的匹配规则。

### 硬性要求 vs 软性偏好

- **硬性要求（required）**：必须满足，否则Pod无法调度。就像"必须坐在研发部"
- **软性偏好（preferred）**：尽量满足，不满足也行。就像"最好靠窗，但不靠窗也可以"

### 示例：优先选择SSD节点，但普通节点也能接受

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: database
spec:
  affinity:
    nodeAffinity:
      # 硬性要求：必须是Linux系统
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
          - matchExpressions:
              - key: kubernetes.io/os
                operator: In
                values: ["linux"]
      # 软性偏好：优先选择SSD节点
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 80
          preference:
            matchExpressions:
              - key: disk-type
                operator: In
                values: ["ssd"]
```

**关于那个超长的名字**：`requiredDuringSchedulingIgnoredDuringExecution`虽然看着吓人，但含义很简单：
- DuringScheduling：调度时检查这个规则
- IgnoredDuringExecution：Pod已经在运行了就不管了（即使节点标签后来变了）

## Pod Affinity/Anti-Affinity：根据其他Pod来决定位置

有时候，你希望某些Pod待在一起（亲和），或者分开部署（反亲和）。

### Pod亲和性：让相关服务靠近

**场景**：Web服务和缓存服务放在同一台机器上，减少网络延迟。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: web-server
spec:
  affinity:
    podAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchLabels:
              app: redis
          topologyKey: kubernetes.io/hostname  # 同一台机器
```

这告诉调度器："把我和标签为app=redis的Pod放在同一台机器上"。

### Pod反亲和性：让相同服务分散

**场景**：3个Web服务副本分散到不同节点，避免单点故障。

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-server
spec:
  replicas: 3
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: web-server
              topologyKey: kubernetes.io/hostname
```

这就像说："我们部门的3个人不能坐在同一排，万一那排空调坏了全军覆没"。

### 拓扑域（topologyKey）是什么？

拓扑域定义了"在一起"或"分开"的范围：
- `kubernetes.io/hostname`：节点级别（同一台机器）
- `topology.kubernetes.io/zone`：可用区级别（同一个机房）
- `topology.kubernetes.io/region`：区域级别（同一个城市）

## Taints和Tolerations：污点和容忍

这是一种"反向选择"机制：节点说"我有这个特点，一般Pod别来"，Pod说"我能接受这个特点"。

### 理解污点和容忍

想象节点是酒店房间：
- **污点（Taint）**：房间贴了"仅限VIP"的标签
- **容忍（Toleration）**：客人有VIP卡，可以住这个房间

没有VIP卡的客人看到这个标签就会选择其他房间，有VIP卡的客人可以选择住这里（但不是必须住这里）。

### 污点的三种效果

| 效果 | 含义 |
|------|------|
| `NoSchedule` | 不接受新Pod（已经住进来的不赶走） |
| `PreferNoSchedule` | 尽量不接受，但实在没地方也行 |
| `NoExecute` | 不接受新的，已经在运行的也赶走 |

### 示例：专用GPU节点

```bash
# 给GPU节点打上污点
kubectl taint nodes gpu-node-1 gpu=true:NoSchedule
```

普通Pod不会被调度到这个节点。GPU任务的Pod需要声明容忍：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-task
spec:
  tolerations:
    - key: "gpu"
      operator: "Equal"
      value: "true"
      effect: "NoSchedule"
  containers:
    - name: cuda-app
      image: cuda-app:1.0
```

**注意**：容忍只是"允许"调度到有污点的节点，不是"必须"。如果还想保证一定调度到GPU节点，需要配合NodeSelector或NodeAffinity使用。

## Topology Spread Constraints：均匀分布

当你希望Pod均匀分布在各个节点或可用区时，使用Topology Spread Constraints。

### 示例：在节点间均匀分布

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
spec:
  replicas: 6
  template:
    spec:
      topologySpreadConstraints:
        - maxSkew: 1                           # 最大不均衡度
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: DoNotSchedule
          labelSelector:
            matchLabels:
              app: web-app
```

`maxSkew: 1`意味着任意两个节点上的Pod数量差异最多为1。如果有3个节点和6个Pod，理想分布是2-2-2，允许的最大不均衡是3-2-1。

## 各种调度方式的对比

| 方式 | 作用 | 适用场景 |
|------|------|----------|
| NodeSelector | 简单标签匹配 | 简单需求，如指定节点类型 |
| Node Affinity | 灵活的节点选择 | 复杂的节点选择逻辑 |
| Pod Affinity | 让Pod靠近 | 减少网络延迟 |
| Pod Anti-Affinity | 让Pod分散 | 高可用部署 |
| Taints/Tolerations | 排斥普通Pod | 专用节点、隔离环境 |
| Topology Spread | 均匀分布 | 跨区域高可用 |

## 常见问题

### Q1: Pod一直Pending，显示"0/3 nodes are available"？

这意味着没有节点满足Pod的要求。排查步骤：

```bash
kubectl describe pod <pod-name>
```

查看Events部分的具体原因，常见的有：
- **Insufficient cpu/memory**：资源不足，需要释放资源或添加节点
- **node(s) didn't match node selector**：没有节点有匹配的标签
- **node(s) had taints that the pod didn't tolerate**：节点有污点，Pod没有容忍

### Q2: NodeSelector和Node Affinity有什么区别？该用哪个？

Node Affinity是NodeSelector的超集，功能更强大：
- NodeSelector只支持精确匹配，Node Affinity支持In、NotIn、Exists等操作符
- NodeSelector没有软性偏好，Node Affinity有
- 官方建议使用Node Affinity，NodeSelector将来可能被废弃

### Q3: Pod Anti-Affinity导致Pod无法调度怎么办？

如果你用了`required`（硬性要求）的反亲和，但节点数量不够，Pod就会一直Pending。

解决方案：
1. 增加节点数量
2. 改用`preferred`（软性偏好），允许在必要时妥协
3. 减少副本数
4. 把topologyKey从hostname改成zone，放宽"分散"的范围

### Q4: 如何实现"某些Pod必须独占节点"？

组合使用Taint和NodeSelector：

```bash
# 给节点打污点和标签
kubectl taint nodes special-node dedicated=special:NoSchedule
kubectl label nodes special-node node-type=special
```

```yaml
spec:
  nodeSelector:
    node-type: special
  tolerations:
    - key: "dedicated"
      value: "special"
      effect: "NoSchedule"
```

这样只有配置了这些规则的Pod才能调度到这个节点，而且这些Pod也只会调度到这个节点。

### Q5: 调度器多久检查一次？

调度器是实时工作的。当Pod创建后，调度器会立即尝试为其找到合适的节点。如果当前没有合适的节点，Pod会进入Pending状态，调度器会持续重试。

## 小结

- **NodeSelector**最简单，适合基本的节点选择
- **Node Affinity**更灵活，支持软硬要求和复杂匹配
- **Pod Affinity/Anti-Affinity**基于其他Pod的位置来决定，用于性能优化和高可用
- **Taints/Tolerations**是"反向选择"，用于保护专用节点
- **Topology Spread**用于实现均匀分布
- 调度失败时，先用`kubectl describe pod`查看具体原因

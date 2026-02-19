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

# VPA垂直自动扩缩

## 为什么需要VPA？

HPA（水平Pod自动扩缩器）通过增加Pod数量来应对负载。但有时候，增加数量不是最好的解决方案。

**场景一：有状态应用**

比如数据库，你不能简单地多开几个实例来提升性能。这时候，给单个实例更多的CPU和内存才是正解。

**场景二：资源配置不准确**

你为应用设置了`requests: 1Gi`内存，但实际只用了200Mi——浪费了80%的预留资源。或者设置了256Mi，结果经常OOM——配置不够用。

这就是VPA（Vertical Pod Autoscaler，垂直Pod自动扩缩器）要解决的问题：**自动调整单个Pod的资源配置**。

## HPA与VPA的区别

用餐厅来比喻：

- **HPA（水平扩缩）**：客人多了就多开几桌。适合普通用餐区，加桌子就行。
- **VPA（垂直扩缩）**：把小桌子换成大桌子。适合包厢，桌子数量固定，只能换更大的。

| 特性 | HPA | VPA |
|------|-----|-----|
| 扩缩方式 | 增减Pod数量 | 调整Pod资源配额 |
| 生效方式 | 立即生效 | 需要重建Pod |
| 适用场景 | 无状态应用 | 有状态应用、资源优化 |
| 服务中断 | 无（新Pod启动后旧Pod才下线） | 有（需要重建Pod） |

## VPA的工作原理

VPA由三个组件组成：

```
┌─────────────────────────────────────────────────────────┐
│                      VPA 系统                            │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐ │
│  │  Recommender │  │   Updater    │  │ Admission     │ │
│  │  (推荐器)    │  │  (更新器)    │  │ Controller    │ │
│  └──────────────┘  └──────────────┘  └───────────────┘ │
│        ↓                 ↓                   ↓          │
│   分析历史指标      决定是否需要        在Pod创建时      │
│   计算推荐值        重建Pod            注入推荐的资源    │
└─────────────────────────────────────────────────────────┘
```

**工作流程**：
1. Recommender持续监控Pod的资源使用，计算推荐值
2. Updater检查Pod当前资源是否与推荐值差距过大
3. 如果差距大，Updater驱逐Pod，触发重建
4. Pod重建时，Admission Controller自动注入推荐的资源配置

## VPA的四种模式

VPA提供四种工作模式，满足不同需求：

### Off模式：只看不动

```yaml
updatePolicy:
  updateMode: "Off"
```

VPA只计算推荐值，不做任何实际调整。你可以查看推荐值，然后手动决定是否采纳。

**适用场景**：刚开始使用VPA，想先观察一段时间；或者想保持手动控制。

### Initial模式：只管新人

```yaml
updatePolicy:
  updateMode: "Initial"
```

VPA只在Pod创建时注入推荐值，不会更新正在运行的Pod。

**适用场景**：不希望运行中的Pod被重建中断，但希望新Pod能用上推荐配置。

### Recreate模式：主动更新

```yaml
updatePolicy:
  updateMode: "Recreate"
```

VPA会驱逐资源配置不合适的Pod，触发重建并应用新配置。

**适用场景**：可以接受短暂中断的应用。

### Auto模式：自动选择

```yaml
updatePolicy:
  updateMode: "Auto"
```

目前等同于Recreate模式。未来Kubernetes支持原地更新时，Auto会自动选择最佳方式。

## 基本使用

### 前提条件

VPA不是Kubernetes自带的，需要单独安装：

```bash
# 克隆autoscaler仓库
git clone https://github.com/kubernetes/autoscaler.git
cd autoscaler/vertical-pod-autoscaler

# 安装VPA
./hack/vpa-up.sh

# 验证安装
kubectl get pods -n kube-system | grep vpa
```

### 最简单的VPA配置

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: my-app-vpa
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
  updatePolicy:
    updateMode: "Auto"
```

这个配置会：
- 监控my-app这个Deployment
- 自动计算推荐的CPU和内存配置
- 自动更新Pod（通过重建）

### 设置资源边界

为了防止VPA推荐的值过高或过低，可以设置边界：

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: my-app-vpa
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
      - containerName: "*"         # 应用于所有容器
        minAllowed:
          cpu: "100m"
          memory: "128Mi"
        maxAllowed:
          cpu: "4"
          memory: "8Gi"
```

**为什么要设置边界？**

- **minAllowed**：防止资源给得太少，导致应用无法启动
- **maxAllowed**：防止资源给得太多，超出节点容量或浪费资源

## 查看VPA推荐值

```bash
kubectl describe vpa my-app-vpa
```

输出中会有类似这样的推荐：

```
Recommendation:
  Container Recommendations:
    Container Name:  my-app
    Lower Bound:           # 最小值（保守估计）
      Cpu:     50m
      Memory:  128Mi
    Target:                # 推荐值（建议使用）
      Cpu:     200m
      Memory:  512Mi
    Upper Bound:           # 最大值（安全边界）
      Cpu:     1
      Memory:  2Gi
```

**各个值的含义**：

| 值 | 含义 | 使用场景 |
|---|------|----------|
| Lower Bound | 第10百分位值 | 资源紧张时可以用这个值 |
| Target | 第90百分位值 | 推荐使用这个值 |
| Upper Bound | 第95百分位值 | 峰值情况下的需求 |

## 实际应用场景

### 场景一：只观察，手动调整

对于关键业务，你可能不想让VPA自动重建Pod。可以用Off模式：

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: production-app-vpa
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: production-app
  updatePolicy:
    updateMode: "Off"   # 只看推荐值，不自动更新
```

然后定期检查推荐值，在维护窗口手动更新。

### 场景二：跳过某些容器

如果Pod中有sidecar容器（如日志收集器），你可能不想VPA调整它们：

```yaml
resourcePolicy:
  containerPolicies:
    - containerName: main-app
      minAllowed:
        cpu: "100m"
        memory: "256Mi"
      maxAllowed:
        cpu: "2"
        memory: "4Gi"
    - containerName: log-collector
      mode: "Off"          # 不调整这个容器
```

### 场景三：VPA与HPA配合

VPA和HPA可以同时使用，但**不能基于相同的指标**。

正确的配合方式：
- VPA：管理CPU和内存配置
- HPA：基于自定义指标（如请求数）扩缩容

```yaml
# VPA配置
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
spec:
  targetRef:
    kind: Deployment
    name: my-app
  updatePolicy:
    updateMode: "Auto"

---
# HPA配置 - 注意不使用CPU/内存指标
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
spec:
  scaleTargetRef:
    kind: Deployment
    name: my-app
  metrics:
    - type: Pods
      pods:
        metric:
          name: http_requests_per_second  # 使用自定义指标
        target:
          type: AverageValue
          averageValue: "1000"
```

## VPA的局限性

### 1. 需要重建Pod

VPA更新资源需要重建Pod，这意味着：
- 短暂的服务中断
- 有状态应用可能丢失内存中的状态
- 长连接会断开

**缓解措施**：配合PodDisruptionBudget使用，确保总有一定数量的Pod在运行。

### 2. 不支持原地更新

目前Kubernetes不支持在Pod运行时修改资源配置。这是底层限制，VPA无能为力。

### 3. JVM应用需特殊考虑

Java应用的内存由JVM管理。即使VPA给了更多内存，JVM也不会自动使用——需要同步调整JVM参数。

**建议**：对JVM应用，VPA只管CPU，内存手动管理。

```yaml
resourcePolicy:
  containerPolicies:
    - containerName: java-app
      controlledResources: ["cpu"]  # 只让VPA管理CPU
```

## 常见问题

### Q1: VPA推荐值一直是空的？

**排查步骤**：

1. 检查VPA组件是否正常运行：
```bash
kubectl get pods -n kube-system | grep vpa
```

2. 检查Metrics Server是否正常：
```bash
kubectl top pods
```

3. 等待足够的数据收集时间（至少几分钟）

### Q2: VPA更新导致服务中断怎么办？

**解决方案**：

1. 使用PodDisruptionBudget：
```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: my-app-pdb
spec:
  minAvailable: 2    # 始终保持至少2个Pod
  selector:
    matchLabels:
      app: my-app
```

2. 改用Initial模式，只在Pod创建时应用，不中断运行中的Pod。

3. 对于关键应用，用Off模式观察，在维护窗口手动更新。

### Q3: VPA推荐值和实际需求差很多？

**可能原因**：
- 负载模式变化（工作日和周末不同）
- 观察时间太短
- 存在内存泄漏

**建议**：
- 延长观察期
- 设置合理的minAllowed和maxAllowed边界
- 检查应用是否有资源使用异常

### Q4: 什么时候用VPA，什么时候用HPA？

| 情况 | 推荐方案 |
|------|----------|
| 无状态Web应用，可以水平扩展 | HPA |
| 数据库等有状态应用 | VPA |
| 单实例应用 | VPA |
| 不确定资源配置是否合理 | VPA的Off模式观察 |
| 流量波动大，需要快速响应 | HPA |

### Q5: VPA能不能和ResourceQuota一起用？

可以，但要注意VPA的maxAllowed不能超过ResourceQuota的限制，否则Pod可能无法创建。

## 小结

- VPA通过**调整单个Pod的资源**来优化性能，与HPA（增减数量）互补
- 四种模式：**Off（只看）、Initial（只管新的）、Recreate（主动更新）、Auto**
- 更新需要**重建Pod**，会有短暂中断
- 配合**PodDisruptionBudget**使用，减少服务影响
- 对于关键应用，建议先用**Off模式观察**，再决定是否自动更新
- VPA和HPA可以配合使用，但要**基于不同指标**

## 参考资源

- [Kubernetes VPA 官方文档](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)
- [VPA 最佳实践指南](https://github.com/kubernetes/autoscaler/blob/master/vertical-pod-autoscaler/docs/faq.md)
- [HPA 与 VPA 配合使用](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)

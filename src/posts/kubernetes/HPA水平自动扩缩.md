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

# HPA水平自动扩缩

## 为什么需要HPA？

想象你开了一家奶茶店。平时只需要2个店员就够了，但周末人流量大，需要5个店员。如果一直养5个店员，平时浪费工资；如果只有2个，周末忙不过来。

最理想的方案是：**根据客流量自动调整店员数量**——忙的时候多叫几个人，闲的时候让人休息。

这就是HPA（Horizontal Pod Autoscaler，水平Pod自动扩缩器）做的事情。它监控你的应用负载，自动增加或减少Pod的数量。

## HPA的工作原理

HPA的工作流程很简单：

```
                  每15秒
    ┌─────────────────────────────┐
    │                             │
    ▼                             │
观察当前指标 → 计算需要的副本数 → 调整副本数
 (CPU/内存等)     (目标值算法)     (Scale API)
```

### 计算公式

```
期望副本数 = 当前副本数 × (当前指标值 / 目标指标值)
```

**举个例子**：
- 当前有2个Pod
- 当前平均CPU使用率：80%
- 目标CPU使用率：50%
- 期望副本数 = 2 × (80% / 50%) = 3.2，向上取整 = 4个

所以HPA会把Pod数量从2个扩到4个。

### 多指标时怎么计算？

如果你配置了多个指标（比如CPU和内存），HPA会分别计算每个指标需要的副本数，然后**取最大值**。

这确保了所有指标都能满足目标要求。

## 基本使用

### 前提条件

HPA需要知道Pod的资源使用情况，这需要安装Metrics Server：

```bash
# 检查是否已安装
kubectl top pods

# 如果报错"Metrics API not available"，需要安装metrics-server
```

另外，你的Pod必须设置了`resources.requests`，否则HPA无法计算使用率。

### 最简单的HPA配置

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: web-app-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: web-app          # 要扩缩的Deployment名称
  minReplicas: 2           # 最少保持2个Pod
  maxReplicas: 10          # 最多扩到10个Pod
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70  # CPU使用率超过70%就扩容
```

这个配置的意思是：
- 监控web-app这个Deployment
- 保持CPU平均使用率在70%左右
- 副本数在2-10之间波动

### 命令行快速创建

```bash
kubectl autoscale deployment web-app --cpu-percent=70 --min=2 --max=10
```

## 扩缩容行为控制

### 问题：扩缩容太快怎么办？

默认情况下，HPA可能反应过于灵敏。CPU突然上升，立刻扩容；CPU一下降，又立刻缩容。这种"抖动"会让系统不稳定。

### 解决方案：稳定窗口

```yaml
behavior:
  scaleDown:
    stabilizationWindowSeconds: 300  # 缩容前观察5分钟
  scaleUp:
    stabilizationWindowSeconds: 0    # 扩容立即响应
```

**稳定窗口**的意思是：在这段时间内，取需要的副本数的最大值（扩容时）或最小值（缩容时）。

比如设置了5分钟的缩容稳定窗口：
- 第1分钟：计算需要3个Pod
- 第2分钟：计算需要2个Pod
- 第3分钟：计算需要4个Pod
- ...
- 5分钟内最小值是2，所以5分钟后才会缩到2个

### 速率限制

你还可以限制扩缩容的速度：

```yaml
behavior:
  scaleUp:
    policies:
      - type: Percent
        value: 100
        periodSeconds: 15  # 每15秒最多扩容100%
      - type: Pods
        value: 4
        periodSeconds: 15  # 或每15秒最多扩4个Pod
    selectPolicy: Max      # 取两种策略中更多的那个

  scaleDown:
    policies:
      - type: Percent
        value: 10
        periodSeconds: 60  # 每分钟最多缩容10%
    selectPolicy: Min      # 取更保守的策略
```

**为什么扩容要快、缩容要慢？**

扩容是为了应对突发流量，要快速响应，否则用户体验会下降。

缩容不那么急迫，而且缩太快可能导致：
- 刚缩完流量又来了，又要扩
- 连接还没处理完就被终止

## 基于自定义指标的HPA

CPU和内存并不总是最好的指标。比如一个消息队列的消费者，CPU可能很低，但积压了大量消息。这时候应该根据队列长度来扩缩容。

### 示例：基于每秒请求数

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: custom-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: web-app
  minReplicas: 2
  maxReplicas: 20
  metrics:
    - type: Pods
      pods:
        metric:
          name: http_requests_per_second  # 自定义指标名
        target:
          type: AverageValue
          averageValue: "100"  # 每个Pod平均处理100 RPS
```

这需要配合Prometheus Adapter等组件，把你的监控指标暴露给Kubernetes的Custom Metrics API。

### 结合多种指标

你可以同时使用多种指标，HPA会取需要副本数最多的那个：

```yaml
metrics:
  # CPU指标
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70

  # 自定义指标：请求数
  - type: Pods
    pods:
      metric:
        name: http_requests_per_second
      target:
        type: AverageValue
        averageValue: "500"
```

这样，无论是CPU高还是请求多，都会触发扩容。

## HPA vs 手动扩缩容

### HPA会覆盖手动调整

如果你用`kubectl scale`手动调整了副本数，HPA会在下次计算时把它改回去。

**临时禁用HPA的方法**：

```bash
# 方法1：调整minReplicas（推荐）
kubectl patch hpa web-app-hpa -p '{"spec":{"minReplicas":10}}'

# 方法2：禁用缩容
kubectl patch hpa web-app-hpa -p '{"spec":{"behavior":{"scaleDown":{"selectPolicy":"Disabled"}}}}'
```

## HPA与VPA的区别

| 特性 | HPA（水平扩缩） | VPA（垂直扩缩） |
|------|-----------------|-----------------|
| 扩缩方式 | 增减Pod数量 | 调整单个Pod资源 |
| 适用场景 | 无状态应用 | 有状态应用、资源优化 |
| 生效方式 | 立即生效 | 需要重建Pod |
| 能否并用 | 可以，但需要错开指标 | 可以，但需要错开指标 |

一般来说：
- 无状态Web应用优先用HPA
- 有状态应用或单实例应用考虑VPA
- 两者可以配合使用：VPA调整资源配置，HPA基于自定义指标扩缩

## 常见问题

### Q1: HPA显示`<unknown>`怎么办？

```bash
kubectl get hpa
# NAME          REFERENCE        TARGETS         MINPODS   MAXPODS
# web-app-hpa   Deployment/web   <unknown>/70%   2         10
```

**常见原因**：

1. **Metrics Server没安装或异常**
```bash
kubectl top pods  # 如果报错，说明metrics有问题
```

2. **Pod没有设置resources.requests**
HPA需要知道requests才能计算使用率。检查你的Pod配置是否有：
```yaml
resources:
  requests:
    cpu: "100m"
```

3. **Pod刚创建**
等待约1分钟让指标收集完成

### Q2: 扩容太慢跟不上流量怎么办？

调整扩容策略，让响应更快：

```yaml
behavior:
  scaleUp:
    stabilizationWindowSeconds: 0  # 立即响应
    policies:
      - type: Percent
        value: 200                  # 每次可以翻倍
        periodSeconds: 15
```

同时考虑提高minReplicas，保持一定的基础容量。

### Q3: HPA频繁扩缩容（抖动）怎么办？

**原因**：指标波动大，或目标值设得太接近实际使用。

**解决方案**：

1. 增加稳定窗口：
```yaml
behavior:
  scaleDown:
    stabilizationWindowSeconds: 600  # 10分钟
```

2. 调整目标值，留更大缓冲：
```yaml
target:
  averageUtilization: 50  # 从70%降到50%
```

3. 限制缩容速度。

### Q4: 如何实现缩容到0？

HPA的minReplicas最小是1，不能缩到0。

如果需要缩到0（比如没有消息时完全关闭消费者），可以使用KEDA（Kubernetes Event-driven Autoscaling）。

### Q5: HPA根据什么时间间隔检查？

默认每15秒检查一次（由`--horizontal-pod-autoscaler-sync-period`控制）。

如果指标波动很大，可以配合稳定窗口来平滑行为。

## 小结

- HPA通过**增减Pod数量**来应对负载变化
- 基本公式：期望副本数 = 当前副本数 × (当前值 / 目标值)
- **扩容要快，缩容要慢**——通过behavior配置实现
- 可以基于**CPU、内存、自定义指标**来扩缩容
- 使用HPA前确保安装了**Metrics Server**，Pod配置了**resources.requests**
- 如果显示`<unknown>`，先检查metrics和资源配置

## 参考资源

- [Kubernetes HPA 官方文档](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
- [HPA 行为配置](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#configurable-scaling-behavior)
- [Metrics Server 安装](https://github.com/kubernetes-sigs/metrics-server)

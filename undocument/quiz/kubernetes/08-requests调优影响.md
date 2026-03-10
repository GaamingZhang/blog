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

# Kubernetes requests调优影响分析

## 概述

在Kubernetes中，`requests`参数的调整会直接影响Pod的调度、资源分配和集群的整体效率。本文将详细分析requests调大调小对系统各方面的影响。

## requests的核心作用

```
+------------------+----------------------------------------+
|      作用        |                 说明                   |
+------------------+----------------------------------------+
| 调度决策         | Scheduler根据requests选择合适的Node    |
| 资源预留         | 在Node上预留相应的资源给Pod            |
| QoS等级          | 影响Pod的服务质量等级                   |
| 资源配额         | 计入Namespace的资源配额使用量          |
+------------------+----------------------------------------+
```

## requests调大的影响

### 正面影响

```
+------------------+----------------------------------------+
|      影响        |                 说明                   |
+------------------+----------------------------------------+
| 资源保障更强     | 确保Pod获得足够的资源                  |
| 性能更稳定       | 减少资源竞争导致的性能波动              |
| 优先级提升       | 在资源紧张时更容易被保留                |
| 调度更精确       | 避免过度调度导致的资源争抢              |
+------------------+----------------------------------------+
```

### 负面影响

```
+------------------+----------------------------------------+
|      影响        |                 说明                   |
+------------------+----------------------------------------+
| 资源利用率降低   | 预留资源可能无法被充分利用              |
| 调度难度增加     | 可能找不到足够资源的Node                |
| 成本增加         | 需要更多的Node来承载相同数量的Pod       |
| 扩展性受限       | 单个Node能容纳的Pod数量减少             |
+------------------+----------------------------------------+
```

### 调大场景示例

```yaml
# 调大前
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"

# 调大后
resources:
  requests:
    cpu: "500m"      # CPU requests增加5倍
    memory: "512Mi"  # 内存requests增加4倍
```

**影响分析：**

```
假设Node资源：4核CPU，8Gi内存

调大前：
- 每个Pod占用：0.1核CPU，128Mi内存
- Node可容纳：40个Pod（CPU限制）

调大后：
- 每个Pod占用：0.5核CPU，512Mi内存
- Node可容纳：8个Pod（CPU限制）

结果：同样的Node数量，可运行的Pod数量减少80%
```

## requests调小的影响

### 正面影响

```
+------------------+----------------------------------------+
|      影响        |                 说明                   |
+------------------+----------------------------------------+
| 资源利用率提高   | 更多Pod可以调度到同一Node              |
| 调度更容易       | 更容易找到满足条件的Node               |
| 成本降低         | 相同负载需要更少的Node                 |
| 扩展性增强       | 单个Node能容纳更多Pod                  |
+------------------+----------------------------------------+
```

### 负面影响

```
+------------------+----------------------------------------+
|      影响        |                 说明                   |
+------------------+----------------------------------------+
| 资源竞争加剧     | 多个Pod可能争抢同一资源                |
| 性能不稳定       | 高负载时可能出现性能下降                |
| OOM风险增加      | 内存不足时可能被OOMKilled              |
| 调度不准确       | 可能导致Node过载                        |
+------------------+----------------------------------------+
```

### 调小场景示例

```yaml
# 调小前
resources:
  requests:
    cpu: "500m"
    memory: "512Mi"

# 调小后
resources:
  requests:
    cpu: "100m"      # CPU requests减少80%
    memory: "128Mi"  # 内存requests减少75%
```

**风险分析：**

```
假设Node资源：4核CPU，8Gi内存
实际负载：每个Pod实际使用300m CPU，400Mi内存

调小后调度：
- 按requests计算，Node可容纳40个Pod
- 实际资源需求：40 × 300m = 12核CPU（超出Node容量）
- 实际资源需求：40 × 400Mi = 16Gi内存（超出Node容量）

结果：严重的资源竞争和性能问题
```

## 调优策略

### 1. 基于监控数据设置

```yaml
# Prometheus查询实际资源使用
# CPU使用
rate(container_cpu_usage_seconds_total{container="app"}[5m])

# 内存使用
container_memory_working_set_bytes{container="app"}

# 建议设置公式
requests.cpu = P95(实际CPU使用) × 1.2
requests.memory = P95(实际内存使用) × 1.3
```

### 2. 分层设置策略

```
+------------------+------------------+------------------+
|    服务类型      |   CPU requests   |   内存requests   |
+------------------+------------------+------------------+
| 关键服务         | 实际使用 × 1.5   | 实际使用 × 1.5   |
| 普通服务         | 实际使用 × 1.2   | 实际使用 × 1.3   |
| 批处理任务       | 实际使用 × 1.0   | 实际使用 × 1.2   |
| 开发测试         | 实际使用 × 0.8   | 实际使用 × 1.0   |
+------------------+------------------+------------------+
```

### 3. VPA自动调优

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: app-vpa
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: app
      minAllowed:
        cpu: "100m"
        memory: "128Mi"
      maxAllowed:
        cpu: "2000m"
        memory: "2Gi"
      controlledResources: ["cpu", "memory"]
```

## 调优影响矩阵

```
+------------------+----------+----------+----------+----------+
|      指标        | 调大很多 | 调大一点 | 调小一点 | 调小很多 |
+------------------+----------+----------+----------+----------+
| 调度成功率       | 降低     | 略降     | 提高     | 提高     |
| 资源利用率       | 降低     | 略降     | 提高     | 提高     |
| 性能稳定性       | 提高     | 提高     | 略降     | 降低     |
| 资源成本         | 增加     | 略增     | 降低     | 降低     |
| OOM风险          | 降低     | 降低     | 略增     | 增加     |
| 资源竞争         | 降低     | 降低     | 略增     | 增加     |
+------------------+----------+----------+----------+----------+
```

## 实际案例分析

### 案例一：Java应用内存调优

```yaml
# 问题：频繁OOM
# 原因：requests设置过低

# 调优前
resources:
  requests:
    memory: "256Mi"
  limits:
    memory: "512Mi"

# 分析：Java应用JVM堆内存 + 元空间 + 直接内存
# 实际需要：堆512Mi + 元空间128Mi + 其他64Mi = 700Mi

# 调优后
resources:
  requests:
    memory: "768Mi"   # 基于实际需求
  limits:
    memory: "1Gi"
```

### 案例二：高并发Web服务CPU调优

```yaml
# 问题：CPU throttling导致响应慢
# 原因：requests设置过低，调度过于密集

# 调优前
resources:
  requests:
    cpu: "100m"
  limits:
    cpu: "500m"

# 分析：高并发时实际CPU使用达到800m
# 导致CPU限流，响应时间增加

# 调优后
resources:
  requests:
    cpu: "400m"   # 提高requests
  limits:
    cpu: "1000m"  # 提高limits
```

## 监控与告警

### Prometheus监控指标

```yaml
# 资源使用率告警
groups:
  - name: resource_alerts
    rules:
      - alert: HighCPUUsage
        expr: |
          rate(container_cpu_usage_seconds_total[5m]) 
          / kube_pod_container_resource_requests_cpu_cores 
          > 0.9
        for: 5m
        annotations:
          summary: "Pod CPU使用接近requests限制"

      - alert: HighMemoryUsage
        expr: |
          container_memory_working_set_bytes 
          / kube_pod_container_resource_requests_memory_bytes 
          > 0.9
        for: 5m
        annotations:
          summary: "Pod内存使用接近requests限制"
```

## 最佳实践总结

### 1. 设置原则

```
- requests应该基于实际使用量设置
- 预留适当的缓冲空间（20%-50%）
- limits应该大于等于requests
- 关键服务适当提高requests
```

### 2. 调优流程

```
1. 监控实际资源使用
2. 分析P95/P99使用量
3. 设置合理的requests
4. 持续监控和调整
5. 使用VPA自动化
```

### 3. 注意事项

```
- 避免requests过小导致Node过载
- 避免requests过大导致资源浪费
- 定期审查和调整requests配置
- 结合limits一起考虑
```

## 参考资源

- [Kubernetes资源管理最佳实践](https://kubernetes.io/zh/docs/concepts/configuration/manage-resources-containers/)
- [VPA官方文档](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)

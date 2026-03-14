---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Prometheus
  - 监控
---

# Prometheus采集K8S资源副本数和重启指标

## kube-state-metrics简介

kube-state-metrics是采集Kubernetes资源对象状态指标的核心组件，它通过监听Kubernetes API，将资源对象的状态转换为Prometheus格式的指标。

```
┌─────────────────────────────────────────────────────────────┐
│               kube-state-metrics工作原理                     │
│                                                              │
│  Kubernetes API                                              │
│       │                                                      │
│       ↓ Watch                                                │
│  ┌─────────────────────────────────────────────────────┐    │
│  │           kube-state-metrics                        │    │
│  │                                                      │    │
│  │  List/Watch资源对象：                                │    │
│  │  - Pod                                               │    │
│  │  - Deployment                                        │    │
│  │  - Node                                              │    │
│  │  - Service                                           │    │
│  │  - PVC                                               │    │
│  │  - ...                                               │    │
│  │                                                      │    │
│  │  转换为Prometheus指标格式                            │    │
│  └─────────────────────────────────────────────────────┘    │
│       │                                                      │
│       ↓ HTTP /metrics                                        │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Prometheus                              │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 副本数相关指标

### Deployment副本数指标

| 指标名称 | 说明 |
|----------|------|
| kube_deployment_spec_replicas | 期望副本数 |
| kube_deployment_status_replicas | 当前副本数 |
| kube_deployment_status_replicas_available | 可用副本数 |
| kube_deployment_status_replicas_unavailable | 不可用副本数 |
| kube_deployment_status_replicas_updated | 已更新副本数 |
| kube_deployment_status_replicas_ready | 就绪副本数 |

### StatefulSet副本数指标

| 指标名称 | 说明 |
|----------|------|
| kube_statefulset_replicas | 期望副本数 |
| kube_statefulset_status_replicas | 当前副本数 |
| kube_statefulset_status_replicas_ready | 就绪副本数 |
| kube_statefulset_status_replicas_current | 当前版本副本数 |
| kube_statefulset_status_replicas_updated | 已更新副本数 |

### ReplicaSet副本数指标

| 指标名称 | 说明 |
|----------|------|
| kube_replicaset_spec_replicas | 期望副本数 |
| kube_replicaset_status_replicas | 当前副本数 |
| kube_replicaset_status_replicas_ready | 就绪副本数 |

### DaemonSet副本数指标

| 指标名称 | 说明 |
|----------|------|
| kube_daemonset_status_desired_number_scheduled | 期望调度数 |
| kube_daemonset_status_current_number_scheduled | 当前调度数 |
| kube_daemonset_status_number_ready | 就绪数 |
| kube_daemonset_status_number_available | 可用数 |
| kube_daemonset_status_number_unavailable | 不可用数 |

## 重启相关指标

### Pod重启指标

| 指标名称 | 说明 |
|----------|------|
| kube_pod_container_status_restarts_total | 容器重启总次数 |
| kube_pod_container_status_last_terminated_reason | 最后一次终止原因 |
| kube_pod_container_status_last_terminated_exitcode | 最后一次退出码 |

### 容器状态指标

| 指标名称 | 说明 |
|----------|------|
| kube_pod_container_status_ready | 容器是否就绪 |
| kube_pod_container_status_running | 容器是否运行中 |
| kube_pod_container_status_terminated | 容器是否已终止 |
| kube_pod_container_status_waiting | 容器是否等待中 |

### Pod状态指标

| 指标名称 | 说明 |
|----------|------|
| kube_pod_status_phase | Pod状态阶段 |
| kube_pod_status_ready | Pod是否就绪 |
| kube_pod_status_scheduled | Pod是否已调度 |
| kube_pod_status_unschedulable | Pod是否不可调度 |

## 常用查询示例

### 1. 查看Deployment副本数状态

```promql
kube_deployment_spec_replicas{deployment="nginx"}
kube_deployment_status_replicas_available{deployment="nginx"}
```

### 2. 副本数不一致的Deployment

```promql
kube_deployment_spec_replicas != kube_deployment_status_replicas_available
```

### 3. 容器重启次数

```promql
kube_pod_container_status_restarts_total{namespace="default"}
```

### 4. 最近1小时内重启次数

```promql
increase(kube_pod_container_status_restarts_total[1h])
```

### 5. 重启次数最多的10个Pod

```promql
topk(10, sum by (pod, namespace) (increase(kube_pod_container_status_restarts_total[1h])))
```

### 6. 频繁重启的Pod（1小时内重启超过5次）

```promql
increase(kube_pod_container_status_restarts_total[1h]) > 5
```

### 7. DaemonSet未完全调度

```promql
kube_daemonset_status_desired_number_scheduled != kube_daemonset_status_current_number_scheduled
```

### 8. Pod处于异常状态

```promql
kube_pod_status_phase{phase=~"Failed|Unknown"}
```

### 9. 容器终止原因统计

```promql
sum by (reason) (kube_pod_container_status_last_terminated_reason)
```

### 10. Pending状态的Pod

```promql
kube_pod_status_phase{phase="Pending"}
```

## 告警规则配置

### Deployment副本数异常

```yaml
groups:
- name: deployment
  rules:
  - alert: DeploymentReplicasMismatch
    expr: kube_deployment_spec_replicas != kube_deployment_status_replicas_available
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Deployment副本数不一致"
      description: "Deployment {{ $labels.namespace }}/{{ $labels.deployment }} 期望副本数 {{ $value }} 与可用副本数不一致"

  - alert: DeploymentUnavailable
    expr: kube_deployment_status_replicas_unavailable > 0
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Deployment存在不可用副本"
      description: "Deployment {{ $labels.namespace }}/{{ $labels.deployment }} 有 {{ $value }} 个不可用副本"
```

### Pod重启告警

```yaml
groups:
- name: pod
  rules:
  - alert: PodCrashLooping
    expr: rate(kube_pod_container_status_restarts_total[15m]) * 60 * 15 > 0
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Pod频繁重启"
      description: "Pod {{ $labels.namespace }}/{{ $labels.pod }} 容器 {{ $labels.container }} 在过去15分钟内重启"

  - alert: PodHighRestartRate
    expr: increase(kube_pod_container_status_restarts_total[1h]) > 5
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Pod重启次数过多"
      description: "Pod {{ $labels.namespace }}/{{ $labels.pod }} 在过去1小时内重启 {{ $value }} 次"
```

### DaemonSet异常告警

```yaml
groups:
- name: daemonset
  rules:
  - alert: DaemonSetNotScheduled
    expr: kube_daemonset_status_desired_number_scheduled - kube_daemonset_status_current_number_scheduled > 0
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "DaemonSet未完全调度"
      description: "DaemonSet {{ $labels.namespace }}/{{ $labels.daemonset }} 有 {{ $value }} 个Pod未调度"

  - alert: DaemonSetRollingUpdateStuck
    expr: kube_daemonset_status_number_updated != kube_daemonset_status_desired_number_scheduled
    for: 15m
    labels:
      severity: warning
    annotations:
      summary: "DaemonSet滚动更新卡住"
      description: "DaemonSet {{ $labels.namespace }}/{{ $labels.daemonset }} 滚动更新未完成"
```

### StatefulSet异常告警

```yaml
groups:
- name: statefulset
  rules:
  - alert: StatefulSetReplicasMismatch
    expr: kube_statefulset_replicas != kube_statefulset_status_replicas_ready
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "StatefulSet副本数不一致"
      description: "StatefulSet {{ $labels.namespace }}/{{ $labels.statefulset }} 副本数不一致"
```

## Grafana Dashboard示例

### Deployment状态面板

```json
{
  "title": "Deployment状态",
  "type": "stat",
  "targets": [
    {
      "expr": "kube_deployment_status_replicas_available{deployment=\"$deployment\"}",
      "legendFormat": "可用副本"
    },
    {
      "expr": "kube_deployment_spec_replicas{deployment=\"$deployment\"}",
      "legendFormat": "期望副本"
    }
  ]
}
```

### Pod重启趋势面板

```json
{
  "title": "Pod重启趋势",
  "type": "graph",
  "targets": [
    {
      "expr": "increase(kube_pod_container_status_restarts_total{namespace=\"$namespace\"}[1h])",
      "legendFormat": "{{ pod }} - {{ container }}"
    }
  ]
}
```

### 容器终止原因饼图

```json
{
  "title": "容器终止原因分布",
  "type": "piechart",
  "targets": [
    {
      "expr": "sum by (reason) (kube_pod_container_status_last_terminated_reason)",
      "legendFormat": "{{ reason }}"
    }
  ]
}
```

## 最佳实践

### 1. 监控所有工作负载类型

```yaml
scrape_configs:
- job_name: 'kube-state-metrics'
  static_configs:
  - targets: ['kube-state-metrics:8080']
```

### 2. 设置合理的告警阈值

```yaml
- alert: PodCrashLooping
  expr: rate(kube_pod_container_status_restarts_total[15m]) * 60 * 15 > 0
  for: 5m
```

### 3. 区分重启原因

```promql
kube_pod_container_status_last_terminated_reason{reason="OOMKilled"}
kube_pod_container_status_last_terminated_reason{reason="Error"}
kube_pod_container_status_last_terminated_reason{reason="Completed"}
```

### 4. 结合资源监控

将副本数指标与资源使用指标结合分析：

```promql
kube_deployment_status_replicas_available{deployment="nginx"}
and on(namespace, pod)
container_memory_working_set_bytes
```

## 参考资源

- [kube-state-metrics文档](https://github.com/kubernetes/kube-state-metrics)
- [Prometheus监控Kubernetes](https://prometheus.io/docs/guides/kubernetes/)

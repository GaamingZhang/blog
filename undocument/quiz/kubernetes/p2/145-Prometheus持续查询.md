---
date: 2026-03-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Prometheus
  - 持续查询
  - Recording Rules
---

# Prometheus 持续查询

## 引言

Prometheus 的 Recording Rules（记录规则）允许预先计算经常需要的或计算昂贵的表达式，将结果保存为新的时间序列。这被称为持续查询或预计算，可以提高查询性能并简化复杂查询。

## Recording Rules 概述

### Recording Rules 作用

```
┌─────────────────────────────────────────────────────────────┐
│              Recording Rules 作用                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 预计算复杂查询                                           │
│     • 提前计算耗时表达式                                     │
│     • 查询时直接使用预计算结果                               │
│                                                              │
│  2. 提高查询性能                                             │
│     • 减少实时计算开销                                       │
│     • 加速 Dashboard 加载                                   │
│                                                              │
│  3. 简化查询                                                 │
│     • 将复杂查询封装为简单指标                               │
│     • 提高可读性                                             │
│                                                              │
│  4. 跨集群聚合                                               │
│     • 聚合多个集群的数据                                     │
│     • 生成全局指标                                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 工作原理

```
┌─────────────────────────────────────────────────────────────┐
│              Recording Rules 工作原理                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  原始指标：                                                  │
│  http_requests_total{method="GET", status="200"}            │
│  http_requests_total{method="POST", status="200"}           │
│                                                              │
│  Recording Rule：                                            │
│  sum(rate(http_requests_total[5m])) by (method)             │
│                                                              │
│  预计算结果：                                                │
│  http_requests_per_second:rate5m{method="GET"}              │
│  http_requests_per_second:rate5m{method="POST"}             │
│                                                              │
│  查询时直接使用：                                            │
│  http_requests_per_second:rate5m                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Recording Rules 配置

### 基本配置

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-rules
  namespace: monitoring
data:
  recording-rules.yml: |
    groups:
    - name: http_requests
      interval: 30s
      rules:
      - record: http_requests_per_second:rate5m
        expr: sum(rate(http_requests_total[5m])) by (method)
        labels:
          aggregated: "true"
```

### Prometheus 配置

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s

    rule_files:
    - /etc/prometheus/rules/*.yml
```

## 常用 Recording Rules 示例

### CPU 使用率

```yaml
groups:
- name: cpu_rules
  rules:
  - record: instance:cpu_usage:rate5m
    expr: 100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)
```

### 内存使用率

```yaml
groups:
- name: memory_rules
  rules:
  - record: instance:memory_usage:percentage
    expr: (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100
```

### 请求速率

```yaml
groups:
- name: http_rules
  rules:
  - record: job:http_requests:rate5m
    expr: sum by (job, method) (rate(http_requests_total[5m]))

  - record: job:http_requests:rate1h
    expr: sum by (job, method) (rate(http_requests_total[1h]))
```

### 错误率

```yaml
groups:
- name: error_rules
  rules:
  - record: job:http_errors:rate5m
    expr: sum by (job) (rate(http_requests_total{status=~"5.."}[5m]))

  - record: job:http_error_rate:ratio5m
    expr: |
      sum by (job) (rate(http_requests_total{status=~"5.."}[5m]))
      /
      sum by (job) (rate(http_requests_total[5m]))
```

### 延迟分位数

```yaml
groups:
- name: latency_rules
  rules:
  - record: job:http_latency:p50_5m
    expr: histogram_quantile(0.50, sum by (job, le) (rate(http_request_duration_seconds_bucket[5m])))

  - record: job:http_latency:p95_5m
    expr: histogram_quantile(0.95, sum by (job, le) (rate(http_request_duration_seconds_bucket[5m])))

  - record: job:http_latency:p99_5m
    expr: histogram_quantile(0.99, sum by (job, le) (rate(http_request_duration_seconds_bucket[5m])))
```

## 命名规范

### 命名格式

```
<level>:<metric_name>:<suffix>

level: 聚合级别（instance, job, cluster 等）
metric_name: 指标名称
suffix: 后缀（rate5m, rate1h, ratio 等）

示例：
instance:cpu_usage:rate5m
job:http_requests:rate5m
cluster:memory_usage:percentage
```

### 常用后缀

| 后缀 | 说明 |
|-----|------|
| rate5m | 5 分钟速率 |
| rate1h | 1 小时速率 |
| ratio | 比率 |
| percentage | 百分比 |
| p50/p95/p99 | 分位数 |

## Recording Rules 与 Alerting Rules

### 组合使用

```yaml
groups:
- name: recording_and_alerting
  rules:
  - record: job:http_error_rate:ratio5m
    expr: |
      sum by (job) (rate(http_requests_total{status=~"5.."}[5m]))
      /
      sum by (job) (rate(http_requests_total[5m]))

  - alert: HighErrorRate
    expr: job:http_error_rate:ratio5m > 0.05
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate detected"
```

## 最佳实践

### 1. 预计算常用查询

```yaml
- record: job:http_requests:rate5m
  expr: sum by (job) (rate(http_requests_total[5m]))
```

### 2. 使用合理的命名

```yaml
- record: instance:cpu_usage:rate5m
  expr: 100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)
```

### 3. 设置合理的评估间隔

```yaml
groups:
- name: expensive_rules
  interval: 1m
  rules:
  - record: cluster:cpu_usage:rate5m
    expr: avg(instance:cpu_usage:rate5m)
```

### 4. 分组管理规则

```yaml
groups:
- name: node_rules
  rules: [...]
- name: http_rules
  rules: [...]
- name: business_rules
  rules: [...]
```

## 面试回答

**问题**: 什么是 Prometheus 的持续查询？

**回答**: Prometheus 持续查询（Recording Rules）是预先计算并保存查询结果的功能：

**作用**：**预计算复杂查询**提前计算耗时表达式，查询时直接使用预计算结果；**提高查询性能**减少实时计算开销，加速 Dashboard 加载；**简化查询**将复杂查询封装为简单指标，提高可读性；**跨集群聚合**聚合多个集群的数据，生成全局指标。

**配置方式**：在 Prometheus 配置文件的 rule_files 中指定规则文件。规则文件定义 groups，每个 group 包含多个 rules。每个 rule 包含 record（新指标名称）、expr（PromQL 表达式）、labels（可选标签）。

**命名规范**：格式为 `<level>:<metric_name>:<suffix>`。level 是聚合级别（instance、job、cluster），metric_name 是指标名称，suffix 是后缀（rate5m、ratio、percentage）。示例：`instance:cpu_usage:rate5m`、`job:http_error_rate:ratio5m`。

**常用场景**：预计算 CPU/内存使用率；预计算请求速率和错误率；预计算延迟分位数（P50/P95/P99）；聚合集群级别指标。

**最佳实践**：预计算常用查询；使用合理的命名规范；设置合理的评估间隔；分组管理规则；Recording Rules 结果可用于 Alerting Rules。

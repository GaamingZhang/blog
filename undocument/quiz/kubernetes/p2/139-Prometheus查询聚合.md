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
  - PromQL
  - 查询
---

# Prometheus 查询和聚合函数

## 引言

PromQL（Prometheus Query Language）是 Prometheus 的查询语言，用于查询和聚合时序数据。掌握 PromQL 的查询操作和聚合函数，是使用 Prometheus 进行监控分析的核心技能。

## PromQL 基础

### 数据类型

```
┌─────────────────────────────────────────────────────────────┐
│                  PromQL 数据类型                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  即时向量（Instant Vector）：                                │
│  • 一组时间序列，每个序列有一个采样值                        │
│  • 示例：http_requests_total                                │
│                                                              │
│  区间向量（Range Vector）：                                  │
│  • 一组时间序列，每个序列有一组采样值                        │
│  • 示例：http_requests_total[5m]                            │
│                                                              │
│  标量（Scalar）：                                            │
│  • 一个简单的数值                                            │
│  • 示例：100                                                │
│                                                              │
│  字符串（String）：                                          │
│  • 一个字符串值                                              │
│  • 示例："hello"                                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 基本查询

```promql
http_requests_total

http_requests_total{method="GET"}

http_requests_total{method="GET", status="200"}

http_requests_total{method=~"GET|POST"}

http_requests_total{status!="500"}
```

### 时间范围

```promql
http_requests_total[5m]

http_requests_total[1h]

http_requests_total[5m] offset 1h
```

## 查询操作

### 比较操作

```promql
http_requests_total > 100

http_requests_total < 1000

http_requests_total == 100

http_requests_total != 0
```

### 数学运算

```promql
http_requests_total * 2

http_requests_total / 60

http_requests_total + 100

http_requests_total - 50
```

### 集合操作

```promql
http_requests_total and http_requests_duration

http_requests_total or http_requests_errors

http_requests_total unless http_requests_ignored
```

## 聚合函数

### sum - 求和

```promql
sum(http_requests_total)

sum by (method) (http_requests_total)

sum by (method, status) (http_requests_total)

sum without (instance) (http_requests_total)
```

### avg - 平均值

```promql
avg(node_cpu_seconds_total)

avg by (mode) (node_cpu_seconds_total)
```

### min / max - 最小/最大值

```promql
min(node_memory_MemTotal_bytes)

max(http_requests_total)
```

### count - 计数

```promql
count(http_requests_total)

count by (job) (up == 0)
```

### topk / bottomk - 前 K 个

```promql
topk(5, http_requests_total)

topk(10, sum by (instance) (node_cpu_seconds_total))

bottomk(3, http_requests_total)
```

## 常用函数

### rate - 计算速率

```promql
rate(http_requests_total[5m])

rate(http_requests_total[1h])

sum(rate(http_requests_total[5m])) by (method)
```

### irate - 瞬时速率

```promql
irate(http_requests_total[5m])

irate(http_requests_total[1m])
```

### increase - 增长量

```promql
increase(http_requests_total[1h])

increase(http_requests_total[5m])
```

### delta - 变化量

```promql
delta(cpu_temp_celsius[1h])

delta(memory_usage_bytes[5m])
```

## 时间函数

### time - 当前时间

```promql
time()

time() - process_start_time_seconds
```

### timestamp - 时间戳

```promql
timestamp(http_requests_total)
```

### 时间计算

```promql
hour()

minute()

month()

year()

day_of_week()

day_of_month()
```

## 数学函数

### abs - 绝对值

```promql
abs(delta(cpu_temp_celsius[1h]))
```

### round - 四舍五入

```promql
round(http_requests_total, 10)
```

### floor / ceil - 向下/向上取整

```promql
floor(http_requests_total)

ceil(http_requests_total)
```

### sqrt - 平方根

```promql
sqrt(http_requests_total)
```

## 标签操作

### label_replace - 替换标签

```promql
label_replace(up, "host", "$1", "instance", "(.*):.*")
```

### label_join - 连接标签

```promql
label_join(up, "new_label", "-", "job", "instance")
```

## 常用查询示例

### CPU 使用率

```promql
100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)
```

### 内存使用率

```promql
(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100
```

### 磁盘使用率

```promql
(1 - (node_filesystem_avail_bytes{fstype!="tmpfs"} / node_filesystem_size_bytes{fstype!="tmpfs"})) * 100
```

### 网络流量

```promql
rate(node_network_receive_bytes_total[5m]) * 8

rate(node_network_transmit_bytes_total[5m]) * 8
```

### 请求 QPS

```promql
sum(rate(http_requests_total[5m])) by (method)
```

### 请求延迟 P99

```promql
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))
```

### 错误率

```promql
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100
```

### Pod 重启次数

```promql
sum(increase(kube_pod_container_status_restarts_total[1h])) by (namespace, pod)
```

## 最佳实践

### 1. 使用 rate 计算速率

```promql
rate(http_requests_total[5m])
```

### 2. 使用 by 聚合

```promql
sum by (method) (rate(http_requests_total[5m]))
```

### 3. 使用 histogram_quantile 计算分位数

```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### 4. 使用 offset 比较历史数据

```promql
http_requests_total - (http_requests_total offset 1h)
```

## 面试回答

**问题**: Prometheus 支持哪些查询操作和聚合函数？

**回答**: PromQL 提供丰富的查询操作和聚合函数：

**查询操作**：**标签匹配**支持 =（等于）、!=（不等于）、=~（正则匹配）、!~（正则不匹配）；**比较操作**支持 >、<、==、!=、>=、<=；**数学运算**支持 +、-、*、/、%（取模）、^（幂）；**集合操作**支持 and（交集）、or（并集）、unless（差集）。

**聚合函数**：**sum** 求和；**avg** 平均值；**min/max** 最小/最大值；**count** 计数；**topk/bottomk** 前 K/后 K 个值；**stddev** 标准差；**stdvar** 方差。聚合支持 by（保留指定标签）和 without（排除指定标签）。

**常用函数**：**rate** 计算计数器的平均速率，最常用；**irate** 计算瞬时速率，更敏感；**increase** 计算增长量；**delta** 计算变化量；**histogram_quantile** 计算分位数（如 P99）。

**常用查询示例**：CPU 使用率使用 `100 - avg(irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100`；内存使用率使用 `(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100`；请求 QPS 使用 `sum(rate(http_requests_total[5m]))`；错误率使用 `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))`。

**最佳实践**：使用 rate 计算计数器速率；使用 by 进行合理聚合；使用 histogram_quantile 计算延迟分位数；使用 offset 比较历史数据。

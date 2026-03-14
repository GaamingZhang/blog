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
  - 监控
  - 指标
---

# Prometheus 监控指标定义

## 引言

Prometheus 是 Kubernetes 监控的核心组件，监控指标是 Prometheus 数据模型的基础。理解如何定义和使用监控指标，对于构建有效的监控系统至关重要。

## Prometheus 指标概述

### 指标类型

```
┌─────────────────────────────────────────────────────────────┐
│                  Prometheus 指标类型                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Counter（计数器）：                                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • 只增不减的累计值                                  │   │
│  │  • 用于记录事件发生次数                              │   │
│  │  • 示例：请求总数、错误总数                          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Gauge（仪表盘）：                                           │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • 可增可减的瞬时值                                  │   │
│  │  • 用于记录当前状态                                  │   │
│  │  • 示例：当前温度、内存使用量                        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Histogram（直方图）：                                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • 对观测值进行采样并统计分布                        │   │
│  │  • 生成 _bucket、_sum、_count 指标                  │   │
│  │  • 示例：请求延迟分布、响应大小分布                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Summary（摘要）：                                           │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • 类似 Histogram，但在客户端计算分位数              │   │
│  │  • 生成 _sum、_count 指标                           │   │
│  │  • 示例：请求延迟分位数                              │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 指标格式

```
<metric_name>{<label_name>=<label_value>, ...} <value> [timestamp]

示例：
http_requests_total{method="GET", status="200"} 1000
http_requests_total{method="POST", status="201"} 500
```

## 指标定义

### Counter 定义

```python
from prometheus_client import Counter

REQUEST_COUNT = Counter(
    'http_requests_total',
    'Total HTTP requests',
    ['method', 'endpoint', 'status']
)

REQUEST_COUNT.labels(method='GET', endpoint='/api/users', status='200').inc()
```

### Gauge 定义

```python
from prometheus_client import Gauge

MEMORY_USAGE = Gauge(
    'memory_usage_bytes',
    'Current memory usage in bytes',
    ['service', 'instance']
)

MEMORY_USAGE.labels(service='myapp', instance='pod-1').set(1024000)
```

### Histogram 定义

```python
from prometheus_client import Histogram

REQUEST_LATENCY = Histogram(
    'http_request_duration_seconds',
    'HTTP request latency in seconds',
    ['method', 'endpoint'],
    buckets=[0.1, 0.5, 1, 2, 5, 10]
)

with REQUEST_LATENCY.labels(method='GET', endpoint='/api/users').time():
    # 处理请求
    pass
```

### Summary 定义

```python
from prometheus_client import Summary

REQUEST_LATENCY = Summary(
    'http_request_latency_seconds',
    'HTTP request latency in seconds',
    ['method', 'endpoint']
)

with REQUEST_LATENCY.labels(method='GET', endpoint='/api/users').time():
    # 处理请求
    pass
```

## 指标命名规范

### 命名约定

```
┌─────────────────────────────────────────────────────────────┐
│                  指标命名规范                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  格式：<namespace>_<name>_<unit>                            │
│                                                              │
│  命名空间：                                                  │
│  • process_：进程相关指标                                   │
│  • http_：HTTP 相关指标                                     │
│  • node_：节点相关指标                                      │
│                                                              │
│  单位后缀：                                                  │
│  • _total：累计值（Counter）                               │
│  • _bytes：字节                                            │
│  • _seconds：秒                                            │
│  • _count：计数                                            │
│                                                              │
│  示例：                                                      │
│  • http_requests_total                                     │
│  • process_cpu_seconds_total                               │
│  • node_memory_usage_bytes                                 │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 常用指标示例

```yaml
# 请求相关
http_requests_total{method="GET", status="200"}
http_request_duration_seconds{method="GET", endpoint="/api"}

# 资源相关
process_cpu_seconds_total
process_resident_memory_bytes
container_memory_usage_bytes

# 业务相关
orders_total{status="completed"}
users_active_count
```

## 指标暴露

### Python 应用暴露指标

```python
from prometheus_client import start_http_server, Counter, Gauge, Histogram
import time

REQUEST_COUNT = Counter(
    'app_requests_total',
    'Total app requests',
    ['method', 'endpoint']
)

REQUEST_LATENCY = Histogram(
    'app_request_latency_seconds',
    'Request latency in seconds',
    ['method', 'endpoint']
)

ACTIVE_CONNECTIONS = Gauge(
    'app_active_connections',
    'Active connections'
)

def handle_request(method, endpoint):
    REQUEST_COUNT.labels(method=method, endpoint=endpoint).inc()
    with REQUEST_LATENCY.labels(method=method, endpoint=endpoint).time():
        time.sleep(0.1)

if __name__ == '__main__':
    start_http_server(8080)
    while True:
        handle_request('GET', '/api/users')
        time.sleep(1)
```

### Kubernetes Pod 配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
spec:
  containers:
  - name: app
    image: myapp:v1
    ports:
    - containerPort: 8080
```

## 指标标签设计

### 标签设计原则

```
┌─────────────────────────────────────────────────────────────┐
│                  标签设计原则                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  原则：                                                      │
│  1. 标签值应该是有限的（避免高基数）                        │
│  2. 标签应该有意义且可聚合                                  │
│  3. 避免使用动态值作为标签                                  │
│                                                              │
│  好的标签：                                                  │
│  • method="GET"                                            │
│  • status="200"                                            │
│  • environment="production"                                │
│                                                              │
│  不好的标签：                                                │
│  • user_id="12345"（高基数）                               │
│  • timestamp="2024-01-01"（动态值）                        │
│  • request_id="abc-123"（唯一值）                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 常用标签

```yaml
labels:
  method: GET
  status: "200"
  endpoint: /api/users
  service: myapp
  instance: pod-1
  namespace: production
  version: v1.0.0
```

## 最佳实践

### 1. 使用合适的指标类型

```python
# 计数使用 Counter
REQUEST_COUNT = Counter('requests_total', 'Total requests')

# 当前状态使用 Gauge
ACTIVE_USERS = Gauge('active_users', 'Active users count')

# 延迟使用 Histogram
LATENCY = Histogram('latency_seconds', 'Latency in seconds')
```

### 2. 遵循命名规范

```python
# 好的命名
http_requests_total
process_cpu_seconds_total
memory_usage_bytes

# 不好的命名
requests
cpu
memory
```

### 3. 合理使用标签

```python
# 好的标签设计
REQUEST_COUNT.labels(
    method='GET',
    status='200',
    endpoint='/api/users'
)

# 避免高基数标签
REQUEST_COUNT.labels(
    user_id='12345'  # 不推荐
)
```

## 面试回答

**问题**: 如何在 Prometheus 中定义监控指标？

**回答**: Prometheus 监控指标定义包括指标类型选择、命名规范、标签设计等方面：

**指标类型**：**Counter（计数器）**只增不减的累计值，用于记录事件发生次数，如请求总数、错误总数；**Gauge（仪表盘）**可增可减的瞬时值，用于记录当前状态，如内存使用量、当前连接数；**Histogram（直方图）**对观测值采样并统计分布，生成 bucket、sum、count 指标，用于请求延迟分布；**Summary（摘要）**类似 Histogram，在客户端计算分位数。

**命名规范**：格式为 `<namespace>_<name>_<unit>`。命名空间如 http_、process_、node_；单位后缀如 _total（累计值）、_bytes（字节）、_seconds（秒）。示例：http_requests_total、process_cpu_seconds_total。

**标签设计**：标签值应该是有限的，避免高基数；标签应该有意义且可聚合；避免使用动态值（如 user_id、timestamp）作为标签。好的标签：method、status、endpoint、service、namespace。

**指标暴露**：应用通过 /metrics 端点暴露指标，Prometheus 定期抓取。Kubernetes Pod 通过注解 prometheus.io/scrape、prometheus.io/port 配置抓取。

**最佳实践**：选择合适的指标类型；遵循命名规范；合理使用标签避免高基数；使用 Histogram 记录延迟分布。

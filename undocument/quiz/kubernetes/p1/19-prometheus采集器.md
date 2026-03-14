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

# Prometheus采集器详解

## 什么是Exporter？

Exporter是Prometheus监控体系中的数据采集组件，它将各种系统的指标转换为Prometheus可以抓取的格式。

```
┌─────────────────────────────────────────────────────────────┐
│                     Exporter工作原理                         │
│                                                              │
│  被监控系统                Exporter              Prometheus  │
│  ┌─────────┐            ┌─────────┐           ┌─────────┐  │
│  │ MySQL   │ ──指标──→ │ MySQL   │ ──HTTP──→ │Prometheus│  │
│  │         │            │Exporter │           │         │  │
│  └─────────┘            └─────────┘           └─────────┘  │
│                                                              │
│  ┌─────────┐            ┌─────────┐                         │
│  │ Redis   │ ──指标──→ │ Redis   │                         │
│  │         │            │Exporter │                         │
│  └─────────┘            └─────────┘                         │
│                                                              │
│  ┌─────────┐            ┌─────────┐                         │
│  │ Node    │ ──指标──→ │ Node    │                         │
│  │         │            │Exporter │                         │
│  └─────────┘            └─────────┘                         │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 常用Exporter分类

### 1. 基础设施Exporter

| Exporter | 说明 | 默认端口 |
|----------|------|----------|
| Node Exporter | 节点指标 | 9100 |
| cAdvisor | 容器指标 | 4194 |
| JMX Exporter | JVM指标 | 9404 |

### 2. 数据库Exporter

| Exporter | 说明 | 默认端口 |
|----------|------|----------|
| MySQL Exporter | MySQL指标 | 9104 |
| PostgreSQL Exporter | PostgreSQL指标 | 9187 |
| Redis Exporter | Redis指标 | 9121 |
| MongoDB Exporter | MongoDB指标 | 9216 |
| Elasticsearch Exporter | ES指标 | 9114 |

### 3. 消息队列Exporter

| Exporter | 说明 | 默认端口 |
|----------|------|----------|
| Kafka Exporter | Kafka指标 | 9308 |
| RabbitMQ Exporter | RabbitMQ指标 | 9419 |
| ActiveMQ Exporter | ActiveMQ指标 | 9161 |

### 4. Web服务Exporter

| Exporter | 说明 | 默认端口 |
|----------|------|----------|
| Nginx Exporter | Nginx指标 | 9113 |
| Apache Exporter | Apache指标 | 9117 |
| HAProxy Exporter | HAProxy指标 | 9101 |

### 5. 云服务Exporter

| Exporter | 说明 | 默认端口 |
|----------|------|----------|
| AWS CloudWatch Exporter | AWS指标 | 9106 |
| Azure Monitor Exporter | Azure指标 | 9276 |
| GCP Stackdriver Exporter | GCP指标 | 9255 |

## Node Exporter详解

### 功能

Node Exporter采集节点级别的硬件和操作系统指标。

### 部署

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-exporter
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: node-exporter
  template:
    metadata:
      labels:
        app: node-exporter
    spec:
      hostNetwork: true
      hostPID: true
      containers:
      - name: node-exporter
        image: prom/node-exporter:v1.6.0
        args:
        - --path.procfs=/host/proc
        - --path.sysfs=/host/sys
        - --path.rootfs=/host/root
        - --collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($$|/)
        ports:
        - containerPort: 9100
        volumeMounts:
        - name: proc
          mountPath: /host/proc
          readOnly: true
        - name: sys
          mountPath: /host/sys
          readOnly: true
        - name: root
          mountPath: /host/root
          readOnly: true
      volumes:
      - name: proc
        hostPath:
          path: /proc
      - name: sys
        hostPath:
          path: /sys
      - name: root
        hostPath:
          path: /
```

### 主要采集器

| 采集器 | 说明 | 默认启用 |
|--------|------|----------|
| cpu | CPU指标 | 是 |
| meminfo | 内存指标 | 是 |
| filesystem | 文件系统指标 | 是 |
| diskstats | 磁盘统计 | 是 |
| netdev | 网络设备指标 | 是 |
| loadavg | 负载平均值 | 是 |
| stat | 系统统计 | 是 |
| time | 时间指标 | 是 |

### 常用指标

```promql
node_cpu_seconds_total{mode="idle"}
node_memory_MemTotal_bytes
node_memory_MemAvailable_bytes
node_filesystem_size_bytes
node_filesystem_avail_bytes
node_network_receive_bytes_total
node_load1
```

## MySQL Exporter详解

### 功能

MySQL Exporter采集MySQL数据库的性能和状态指标。

### 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql-exporter
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mysql-exporter
  template:
    metadata:
      labels:
        app: mysql-exporter
    spec:
      containers:
      - name: mysql-exporter
        image: prom/mysqld-exporter:v0.15.0
        env:
        - name: DATA_SOURCE_NAME
          value: "user:password@(mysql:3306)/"
        ports:
        - containerPort: 9104
```

### 主要指标

```promql
mysql_global_status_queries
mysql_global_status_connections
mysql_global_status_slow_queries
mysql_global_status_threads_connected
mysql_global_status_buffer_pool_size
mysql_global_status_innodb_buffer_pool_reads
```

### 常用查询

**QPS**：
```promql
rate(mysql_global_status_queries[5m])
```

**连接数**：
```promql
mysql_global_status_threads_connected
```

**慢查询比例**：
```promql
rate(mysql_global_status_slow_queries[5m]) / rate(mysql_global_status_queries[5m])
```

## Redis Exporter详解

### 功能

Redis Exporter采集Redis数据库的性能和状态指标。

### 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis-exporter
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis-exporter
  template:
    metadata:
      labels:
        app: redis-exporter
    spec:
      containers:
      - name: redis-exporter
        image: oliver006/redis_exporter:v1.50.0
        env:
        - name: REDIS_ADDR
          value: "redis://redis:6379"
        ports:
        - containerPort: 9121
```

### 主要指标

```promql
redis_connected_clients
redis_memory_used_bytes
redis_memory_max_bytes
redis_keyspace_keys_total
redis_commands_processed_total
redis_instantaneous_ops_per_sec
```

### 常用查询

**内存使用率**：
```promql
redis_memory_used_bytes / redis_memory_max_bytes * 100
```

**连接数**：
```promql
redis_connected_clients
```

**OPS**：
```promql
redis_instantaneous_ops_per_sec
```

## Nginx Exporter详解

### 功能

Nginx Exporter采集Nginx的状态指标，需要Nginx开启stub_status模块。

### Nginx配置

```nginx
server {
    location /stub_status {
        stub_status;
        allow 127.0.0.1;
        deny all;
    }
}
```

### 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-exporter
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-exporter
  template:
    metadata:
      labels:
        app: nginx-exporter
    spec:
      containers:
      - name: nginx-exporter
        image: nginx/nginx-prometheus-exporter:v0.11.0
        args:
        - -nginx.scrape-uri=http://nginx:8080/stub_status
        ports:
        - containerPort: 9113
```

### 主要指标

```promql
nginx_connections_accepted
nginx_connections_active
nginx_connections_handled
nginx_http_requests_total
nginx_upstream_server_response_time_seconds
```

## Kafka Exporter详解

### 功能

Kafka Exporter采集Kafka集群的指标。

### 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kafka-exporter
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kafka-exporter
  template:
    metadata:
      labels:
        app: kafka-exporter
    spec:
      containers:
      - name: kafka-exporter
        image: danielqsj/kafka-exporter:v1.7.0
        args:
        - --kafka.server=kafka:9092
        ports:
        - containerPort: 9308
```

### 主要指标

```promql
kafka_topic_partition_current_offset
kafka_consumergroup_current_offset
kafka_consumergroup_lag
kafka_brokers
kafka_topic_partitions
```

### 常用查询

**消费者延迟**：
```promql
kafka_consumergroup_lag
```

**Topic数量**：
```promql
count(kafka_topic_partition_current_offset) by (topic)
```

## Blackbox Exporter详解

### 功能

Blackbox Exporter用于黑盒监控，探测HTTP、TCP、DNS等端点。

### 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: blackbox-exporter
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: blackbox-exporter
  template:
    metadata:
      labels:
        app: blackbox-exporter
    spec:
      containers:
      - name: blackbox-exporter
        image: prom/blackbox-exporter:v0.24.0
        ports:
        - containerPort: 9115
        volumeMounts:
        - name: config
          mountPath: /etc/blackbox_exporter
      volumes:
      - name: config
        configMap:
          name: blackbox-config
```

### 配置示例

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: blackbox-config
  namespace: monitoring
data:
  config.yml: |
    modules:
      http_2xx:
        prober: http
        timeout: 5s
        http:
          valid_http_versions: ["HTTP/1.1", "HTTP/2.0"]
          valid_status_codes: [200]
          method: GET
      tcp_connect:
        prober: tcp
        timeout: 5s
      dns_tcp:
        prober: dns
        timeout: 5s
        dns:
          transport_protocol: tcp
```

### 使用方式

```yaml
scrape_configs:
- job_name: 'blackbox'
  metrics_path: /probe
  params:
    module: [http_2xx]
  static_configs:
  - targets:
    - https://example.com
    - https://api.example.com/health
  relabel_configs:
  - source_labels: [__address__]
    target_label: __param_target
  - source_labels: [__param_target]
    target_label: instance
  - target_label: __address__
    replacement: blackbox-exporter:9115
```

## kube-state-metrics详解

### 功能

kube-state-metrics采集Kubernetes资源对象的状态指标。

### 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-state-metrics
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kube-state-metrics
  template:
    metadata:
      labels:
        app: kube-state-metrics
    spec:
      serviceAccountName: kube-state-metrics
      containers:
      - name: kube-state-metrics
        image: registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.10.0
        ports:
        - containerPort: 8080
          name: http-metrics
        - containerPort: 8081
          name: telemetry
```

### 主要指标

```promql
kube_pod_status_phase
kube_pod_container_status_restarts_total
kube_deployment_status_replicas
kube_node_status_condition
kube_persistentvolumeclaim_status_phase
```

## 自定义Exporter开发

### 原理

自定义Exporter需要实现/metrics端点，返回Prometheus格式的指标。

### Python示例

```python
from prometheus_client import Counter, Gauge, start_http_server
import random
import time

REQUEST_COUNT = Counter('myapp_requests_total', 'Total requests')
REQUEST_LATENCY = Gauge('myapp_request_latency_seconds', 'Request latency')

def process_request():
    REQUEST_COUNT.inc()
    latency = random.random()
    REQUEST_LATENCY.set(latency)
    time.sleep(latency)

if __name__ == '__main__':
    start_http_server(8000)
    while True:
        process_request()
```

### Go示例

```go
package main

import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "myapp_requests_total",
            Help: "Total number of requests",
        },
        []string{"method", "path"},
    )
)

func init() {
    prometheus.MustRegister(requestsTotal)
}

func main() {
    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":8080", nil)
}
```

## 最佳实践

### 1. 合理设置采集间隔

```yaml
global:
  scrape_interval: 15s
```

### 2. 使用标签区分环境

```yaml
static_configs:
- targets:
  - node-exporter:9100
  labels:
    environment: production
    datacenter: beijing
```

### 3. 监控Exporter本身

```yaml
- job_name: 'prometheus'
  static_configs:
  - targets: ['localhost:9090']
```

### 4. 使用Relabel配置

```yaml
relabel_configs:
- source_labels: [__address__]
  target_label: instance
  replacement: $1
```

## 参考资源

- [Prometheus Exporter列表](https://prometheus.io/docs/instrumenting/exporters/)
- [Node Exporter文档](https://github.com/prometheus/node_exporter)
- [自定义Exporter开发](https://prometheus.io/docs/instrumenting/writing_exporters/)

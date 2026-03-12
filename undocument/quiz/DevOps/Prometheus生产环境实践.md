---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Monitoring
tag:
  - Prometheus
  - Monitoring
  - DevOps
---

# Prometheus 生产环境实践

当你的微服务架构从 10 个服务扩展到 100 个服务时,监控系统的挑战开始显现:指标采集延迟、存储容量不足、查询性能下降、告警风暴。Prometheus 作为云原生监控的事实标准,其强大的数据模型和查询语言使其成为大规模监控的首选。但 Prometheus 并不是"开箱即用"——架构设计、数据采集、存储优化、告警规则都需要深入理解才能在生产环境稳定运行。

本文将从架构设计、数据采集、存储优化、告警规则、高可用部署五个维度,系统梳理 Prometheus 生产环境的实践经验。

## 一、架构设计

### Prometheus 架构组件

```
┌─────────────────────────────────────────────────────────────┐
│                    Prometheus 架构                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Prometheus Server                                    │  │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐       │  │
│  │  │ Retrieval  │ │    TSDB    │ │ HTTP Server│       │  │
│  │  │ (采集器)   │ │  (存储)    │ │  (查询)    │       │  │
│  │  └────────────┘ └────────────┘ └────────────┘       │  │
│  └──────────────────────────────────────────────────────┘  │
│         │                │                │                 │
│         ▼                ▼                ▼                 │
│  ┌────────────┐   ┌────────────┐   ┌────────────┐         │
│  │ Exporters  │   │   Alert    │   │  Grafana   │         │
│  │ (指标源)   │   │  Manager   │   │ (可视化)   │         │
│  └────────────┘   └────────────┘   └────────────┘         │
│         │                │                                  │
│         ▼                ▼                                  │
│  ┌────────────┐   ┌────────────┐                          │
│  │ Pushgateway│   │   Service  │                          │
│  │ (推送网关) │   │  Discovery │                          │
│  └────────────┘   └────────────┘                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**核心组件说明**:

| 组件 | 功能 | 说明 |
|------|------|------|
| Prometheus Server | 数据采集、存储、查询 | 核心组件 |
| Exporters | 暴露指标 | Node Exporter、MySQL Exporter 等 |
| Pushgateway | 接收推送指标 | 短生命周期任务使用 |
| Alertmanager | 告警处理 | 去重、分组、路由、通知 |
| Grafana | 可视化 | Dashboard 展示 |
| Service Discovery | 服务发现 | Kubernetes、Consul、DNS 等 |

### 部署架构选择

**单机架构**:

适合中小规模场景(指标数 < 100 万):

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  storage:
    tsdb:
      retention.time: 15d
      retention.size: 50GB

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node1:9100', 'node2:9100', 'node3:9100']

  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
```

**联邦架构**:

适合大规模场景(指标数 > 100 万):

```
┌─────────────────────────────────────────────────────────────┐
│                    Prometheus 联邦架构                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Global Prometheus (联邦主节点)                       │  │
│  │  - 聚合全局指标                                       │  │
│  │  - 跨集群查询                                         │  │
│  │  - 长期存储                                           │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│          ┌──────────┼──────────┐                           │
│          │          │          │                           │
│          ▼          ▼          ▼                           │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐             │
│  │ Prometheus │ │ Prometheus │ │ Prometheus │             │
│  │ (集群 A)   │ │ (集群 B)   │ │ (集群 C)   │             │
│  │ - 采集本集群│ │ - 采集本集群│ │ - 采集本集群│             │
│  │ - 短期存储 │ │ - 短期存储 │ │ - 短期存储 │             │
│  └────────────┘ └────────────┘ └────────────┘             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

```yaml
# 联邦配置
scrape_configs:
  - job_name: 'federate'
    scrape_interval: 15s
    honor_labels: true
    metrics_path: '/federate'
    params:
      'match[]':
        - '{job="prometheus"}'
        - '{__name__=~"job:.*"}'
    static_configs:
      - targets:
        - 'prometheus-cluster-a:9090'
        - 'prometheus-cluster-b:9090'
        - 'prometheus-cluster-c:9090'
```

## 二、数据采集

### 服务发现配置

**Kubernetes 服务发现**:

```yaml
scrape_configs:
  # Pod 指标采集
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - production
            - staging
    relabel_configs:
      # 只采集有 prometheus.io/scrape=true 注解的 Pod
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      # 读取 prometheus.io/path 注解
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      # 读取 prometheus.io/port 注解
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
      # 添加 Kubernetes 标签
      - action: labelmap
        regex: __meta_kubernetes_pod_label_(.+)
      - source_labels: [__meta_kubernetes_namespace]
        action: replace
        target_label: kubernetes_namespace
      - source_labels: [__meta_kubernetes_pod_name]
        action: replace
        target_label: kubernetes_pod_name

  # Service 指标采集
  - job_name: 'kubernetes-services'
    kubernetes_sd_configs:
      - role: service
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_probe]
        action: keep
        regex: true

  # Node 指标采集
  - job_name: 'kubernetes-nodes'
    kubernetes_sd_configs:
      - role: node
    relabel_configs:
      - source_labels: [__address__]
        regex: '(.*):10250'
        replacement: '${1}:9100'
        target_label: __address__
```

**Consul 服务发现**:

```yaml
scrape_configs:
  - job_name: 'consul-services'
    consul_sd_configs:
      - server: 'consul.example.com:8500'
        services: ['web', 'api', 'db']
        tags: ['production']
        token: 'your-consul-token'
    relabel_configs:
      - source_labels: [__meta_consul_service]
        target_label: service
      - source_labels: [__meta_consul_tags]
        target_label: tags
```

### Exporter 配置

**Node Exporter**:

```yaml
# DaemonSet 部署
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
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9100"
    spec:
      hostNetwork: true
      hostPID: true
      containers:
        - name: node-exporter
          image: prom/node-exporter:latest
          args:
            - '--path.procfs=/host/proc'
            - '--path.sysfs=/host/sys'
            - '--path.rootfs=/host/root'
            - '--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($$|/)'
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
              mountPropagation: HostToContainer
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

**自定义 Exporter**:

```python
from prometheus_client import Counter, Histogram, Gauge, start_http_server
import time
import random

# 定义指标
REQUEST_COUNT = Counter(
    'app_request_count',
    'Application Request Count',
    ['method', 'endpoint', 'status']
)

REQUEST_LATENCY = Histogram(
    'app_request_latency_seconds',
    'Application Request Latency',
    ['method', 'endpoint']
)

ACTIVE_CONNECTIONS = Gauge(
    'app_active_connections',
    'Active Connections'
)

def simulate_request():
    # 模拟请求
    method = random.choice(['GET', 'POST', 'PUT', 'DELETE'])
    endpoint = random.choice(['/api/users', '/api/orders', '/api/products'])
    status = random.choice(['200', '201', '400', '404', '500'])
    
    # 记录请求计数
    REQUEST_COUNT.labels(method=method, endpoint=endpoint, status=status).inc()
    
    # 记录请求延迟
    latency = random.uniform(0.01, 1.0)
    REQUEST_LATENCY.labels(method=method, endpoint=endpoint).observe(latency)
    
    # 更新活跃连接数
    ACTIVE_CONNECTIONS.set(random.randint(10, 100))

if __name__ == '__main__':
    # 启动 HTTP 服务器
    start_http_server(8000)
    
    while True:
        simulate_request()
        time.sleep(1)
```

### 指标标签管理

**标签设计原则**:

```
┌─────────────────────────────────────────────────────────────┐
│                    指标标签设计原则                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 基数控制                                                 │
│     - 避免高基数标签(如 user_id、request_id)                │
│     - 标签组合数 < 10000                                     │
│                                                              │
│  2. 命名规范                                                 │
│     - 使用 snake_case                                       │
│     - 避免保留标签(__name__, job, instance)                 │
│                                                              │
│  3. 标签用途                                                 │
│     - 用于聚合和查询                                         │
│     - 避免存储业务数据                                       │
│                                                              │
│  示例:                                                       │
│  ✅ method="GET", endpoint="/api/users", status="200"       │
│  ❌ user_id="12345", request_id="abc-def-ghi"               │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Relabel 配置**:

```yaml
scrape_configs:
  - job_name: 'my-app'
    static_configs:
      - targets: ['app1:8000', 'app2:8000']
    relabel_configs:
      # 添加环境标签
      - target_label: environment
        replacement: production
      
      # 添加区域标签
      - source_labels: [__address__]
        regex: 'app1.*'
        target_label: region
        replacement: 'us-east-1'
      
      - source_labels: [__address__]
        regex: 'app2.*'
        target_label: region
        replacement: 'us-west-2'
      
      # 过滤标签
      - regex: '__meta_kubernetes_pod_label_(.+)'
        action: labelmap
```

## 三、存储优化

### TSDB 存储原理

```
┌─────────────────────────────────────────────────────────────┐
│                    TSDB 存储架构                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Head Block (内存块)                                  │  │
│  │  - 最新数据(约 2 小时)                                │  │
│  │  - 内存存储                                           │  │
│  │  - 快速写入                                           │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│                     ▼ (Compaction)                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Persistent Blocks (持久化块)                         │  │
│  │  - Block 1: 2 小时                                    │  │
│  │  - Block 2: 2 小时                                    │  │
│  │  - Block 3: 2 小时                                    │  │
│  │  - Block 4: 6 小时(合并后)                            │  │
│  │  - Block 5: 18 小时(合并后)                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  Compaction 过程:                                            │
│  1. Head Block 满后刷盘为 Block                              │
│  2. 多个小 Block 合并为大 Block                              │
│  3. 删除过期数据                                             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**存储配置**:

```yaml
# prometheus.yml
global:
  storage:
    tsdb:
      # 数据保留时间
      retention.time: 15d
      # 数据保留大小
      retention.size: 50GB
      # 最小 Block 时间
      min_block_duration: 2h
      # 最大 Block 时间
      max_block_duration: 6h
      # Compaction 并发数
      max_concurrent_compactions: 2

# 启动参数
# --storage.tsdb.path=/data/prometheus
# --storage.tsdb.retention.time=15d
# --storage.tsdb.retention.size=50GB
```

### 性能优化

**写入优化**:

```yaml
# prometheus.yml
global:
  # 采集间隔
  scrape_interval: 15s
  # 评估间隔
  evaluation_interval: 15s

# 启动参数
# --storage.tsdb.max-block-duration=2h
# --storage.tsdb.min-block-duration=2h
```

**查询优化**:

```yaml
# prometheus.yml
global:
  # 查询超时
  query_timeout: 2m
  # 查询并发数
  query_concurrency: 20
  # 查询最大样本数
  query_max_samples: 50000000

# 启动参数
# --query.timeout=2m
# --query.concurrency=20
# --query.max-samples=50000000
```

**内存优化**:

```yaml
# 启动参数
# --storage.tsdb.head-block-timeout=15m
# --storage.tsdb.wal-segment-size=32MB
# --storage.tsdb.wal-compression
```

### 远程存储集成

**Thanos 集成**:

```yaml
# prometheus.yml
global:
  external_labels:
    cluster: 'production'
    region: 'us-east-1'

# Thanos Sidecar
# thanos sidecar \
#   --tsdb.path=/data/prometheus \
#   --prometheus.url=http://localhost:9090 \
#   --objstore.config-file=/etc/thanos/object-store.yaml \
#   --grpc-address=0.0.0.0:10901 \
#   --http-address=0.0.0.0:10902
```

**VictoriaMetrics 集成**:

```yaml
# prometheus.yml
remote_write:
  - url: http://victoriametrics:8428/api/v1/write
    queue_config:
      max_samples_per_send: 10000
      capacity: 20000
      max_shards: 30

remote_read:
  - url: http://victoriametrics:8428/api/v1/read
    read_recent: true
```

## 四、告警规则

### 告警规则配置

**基础告警规则**:

```yaml
# alert_rules.yml
groups:
  - name: node_alerts
    rules:
      # CPU 使用率告警
      - alert: HighCPUUsage
        expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High CPU usage on {{ $labels.instance }}"
          description: "CPU usage is {{ $value }}% (threshold: 80%)"

      # 内存使用率告警
      - alert: HighMemoryUsage
        expr: (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > 85
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage on {{ $labels.instance }}"
          description: "Memory usage is {{ $value }}% (threshold: 85%)"

      # 磁盘使用率告警
      - alert: HighDiskUsage
        expr: (1 - (node_filesystem_avail_bytes{fstype!="tmpfs"} / node_filesystem_size_bytes{fstype!="tmpfs"})) * 100 > 85
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High disk usage on {{ $labels.instance }}"
          description: "Disk usage is {{ $value }}% (threshold: 85%)"

  - name: pod_alerts
    rules:
      # Pod 重启告警
      - alert: PodRestartingTooOften
        expr: increase(kube_pod_container_status_restarts_total[1h]) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Pod {{ $labels.namespace }}/{{ $labels.pod }} is restarting frequently"
          description: "Pod has restarted {{ $value }} times in the last hour"

      # Pod 状态异常
      - alert: PodNotReady
        expr: kube_pod_status_phase{phase=~"Pending|Unknown|Failed"} > 0
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Pod {{ $labels.namespace }}/{{ $labels.pod }} is not ready"
          description: "Pod has been in {{ $labels.phase }} state for more than 10 minutes"
```

**高级告警规则**:

```yaml
groups:
  - name: slo_alerts
    rules:
      # 可用性 SLO 告警
      - alert: SLONotMet
        expr: |
          (
            sum(rate(http_requests_total{status!~"5.."}[1h]))
            /
            sum(rate(http_requests_total[1h]))
          ) < 0.99
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "SLO availability below 99%"
          description: "Availability is {{ $value | humanizePercentage }} (threshold: 99%)"

      # 延迟 SLO 告警
      - alert: HighLatency
        expr: |
          histogram_quantile(0.95,
            sum(rate(http_request_duration_seconds_bucket[5m])) by (le)
          ) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High P95 latency"
          description: "P95 latency is {{ $value }}s (threshold: 1s)"

      # 错误率告警
      - alert: HighErrorRate
        expr: |
          (
            sum(rate(http_requests_total{status=~"5.."}[5m]))
            /
            sum(rate(http_requests_total[5m]))
          ) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate"
          description: "Error rate is {{ $value | humanizePercentage }} (threshold: 5%)"
```

### Alertmanager 配置

**基础配置**:

```yaml
# alertmanager.yml
global:
  resolve_timeout: 5m
  smtp_smarthost: 'smtp.example.com:587'
  smtp_from: 'alertmanager@example.com'
  smtp_auth_username: 'alertmanager@example.com'
  smtp_auth_password: 'password'

route:
  receiver: 'default-receiver'
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  group_by: ['alertname', 'cluster', 'service']
  routes:
    - match:
        severity: critical
      receiver: 'critical-receiver'
      group_wait: 10s
      repeat_interval: 1h
    
    - match:
        severity: warning
      receiver: 'warning-receiver'
      group_wait: 30s
      repeat_interval: 4h

receivers:
  - name: 'default-receiver'
    email_configs:
      - to: 'team@example.com'
        send_resolved: true

  - name: 'critical-receiver'
    email_configs:
      - to: 'oncall@example.com'
        send_resolved: true
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/xxx'
        channel: '#alerts-critical'
        send_resolved: true

  - name: 'warning-receiver'
    email_configs:
      - to: 'team@example.com'
        send_resolved: true

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'cluster', 'service']
```

**告警分组和去重**:

```yaml
route:
  # 分组等待时间
  group_wait: 30s
  # 同组告警间隔
  group_interval: 5m
  # 重复告警间隔
  repeat_interval: 4h
  # 分组依据
  group_by: ['alertname', 'cluster', 'service']

inhibit_rules:
  # 抑制规则:critical 告警抑制 warning 告警
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'instance']
```

## 五、高可用部署

### Prometheus 高可用

**多副本部署**:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: monitoring
spec:
  replicas: 2
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
        - name: prometheus
          image: prom/prometheus:latest
          args:
            - '--config.file=/etc/prometheus/prometheus.yml'
            - '--storage.tsdb.path=/data/prometheus'
            - '--storage.tsdb.retention.time=15d'
            - '--web.enable-lifecycle'
            - '--web.external-url=http://prometheus.example.com'
          ports:
            - containerPort: 9090
          volumeMounts:
            - name: config
              mountPath: /etc/prometheus
            - name: data
              mountPath: /data/prometheus
          resources:
            requests:
              cpu: 500m
              memory: 2Gi
            limits:
              cpu: 2
              memory: 8Gi
      volumes:
        - name: config
          configMap:
            name: prometheus-config
        - name: data
          persistentVolumeClaim:
            claimName: prometheus-data
```

**Thanos 架构**:

```yaml
# Thanos Query
apiVersion: apps/v1
kind: Deployment
metadata:
  name: thanos-query
  namespace: monitoring
spec:
  replicas: 2
  selector:
    matchLabels:
      app: thanos-query
  template:
    metadata:
      labels:
        app: thanos-query
    spec:
      containers:
        - name: thanos-query
          image: thanosio/thanos:latest
          args:
            - query
            - --log.level=info
            - --query.replica-label=replica
            - --store=dnssrv+_grpc._tcp.thanos-store.monitoring.svc.cluster.local:10901
            - --store=dnssrv+_grpc._tcp.thanos-sidecar.monitoring.svc.cluster.local:10901
          ports:
            - containerPort: 10902
              name: http
            - containerPort: 10901
              name: grpc
```

### Alertmanager 高可用

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: alertmanager
  namespace: monitoring
spec:
  serviceName: alertmanager
  replicas: 3
  selector:
    matchLabels:
      app: alertmanager
  template:
    metadata:
      labels:
        app: alertmanager
    spec:
      containers:
        - name: alertmanager
          image: prom/alertmanager:latest
          args:
            - '--config.file=/etc/alertmanager/alertmanager.yml'
            - '--storage.path=/data/alertmanager'
            - '--cluster.listen-address=0.0.0.0:9094'
            - '--cluster.peer=alertmanager-0.alertmanager:9094'
            - '--cluster.peer=alertmanager-1.alertmanager:9094'
            - '--cluster.peer=alertmanager-2.alertmanager:9094'
          ports:
            - containerPort: 9093
              name: http
            - containerPort: 9094
              name: cluster
          volumeMounts:
            - name: config
              mountPath: /etc/alertmanager
            - name: data
              mountPath: /data/alertmanager
      volumes:
        - name: config
          configMap:
            name: alertmanager-config
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ['ReadWriteOnce']
        resources:
          requests:
            storage: 10Gi
```

## 小结

- **架构设计**:单机架构适合中小规模,联邦架构适合大规模场景,选择合适的架构保证性能和可扩展性
- **数据采集**:使用 Kubernetes 服务发现自动发现目标,合理配置 Exporter,遵循标签设计原则避免高基数
- **存储优化**:理解 TSDB 存储原理,配置合理的保留时间和大小,集成远程存储实现长期存储
- **告警规则**:设计合理的告警规则,使用 Alertmanager 实现分组、去重、路由,避免告警风暴
- **高可用部署**:多副本部署保证可用性,Thanos 实现长期存储和全局查询,Alertmanager 集群保证告警可靠性

---

## 常见问题

### Q1:Prometheus 的指标类型有哪些?

**四种指标类型**:

| 类型 | 说明 | 示例 |
|------|------|------|
| Counter | 只增不减的计数器 | 请求总数、错误总数 |
| Gauge | 可增可减的度量 | 当前温度、内存使用 |
| Histogram | 直方图,统计分布 | 请求延迟分布 |
| Summary | 摘要,统计分位数 | 请求延迟 P99 |

**使用示例**:

```python
from prometheus_client import Counter, Gauge, Histogram, Summary

# Counter
REQUEST_COUNT = Counter('request_count', 'Total request count')
REQUEST_COUNT.inc()

# Gauge
ACTIVE_CONNECTIONS = Gauge('active_connections', 'Active connections')
ACTIVE_CONNECTIONS.set(100)
ACTIVE_CONNECTIONS.inc()
ACTIVE_CONNECTIONS.dec()

# Histogram
REQUEST_LATENCY = Histogram('request_latency_seconds', 'Request latency')
REQUEST_LATENCY.observe(0.5)

# Summary
REQUEST_SUMMARY = Summary('request_summary_seconds', 'Request summary')
REQUEST_SUMMARY.observe(0.5)
```

### Q2:如何避免 Prometheus 的高基数问题?

**高基数问题**:

```
高基数标签组合数 > 10000,导致:
- 内存占用过高
- 查询性能下降
- 存储空间不足
```

**解决方案**:

1. **避免高基数标签**:

```python
# 错误:使用 user_id 标签
REQUEST_COUNT = Counter('request_count', 'Request count', ['user_id'])

# 正确:使用低基数标签
REQUEST_COUNT = Counter('request_count', 'Request count', ['method', 'endpoint', 'status'])
```

2. **使用 Recording Rules**:

```yaml
groups:
  - name: aggregation_rules
    rules:
      # 预聚合高基数指标
      - record: job:http_requests:rate5m
        expr: sum by (job, status) (rate(http_requests_total[5m]))
```

3. **使用 relabel 过滤标签**:

```yaml
relabel_configs:
  - regex: 'user_id|request_id'
    action: labeldrop
```

### Q3:Prometheus 如何实现长期存储?

**方案对比**:

| 方案 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| 增加保留时间 | 修改 retention.time | 简单 | 单机存储限制 |
| Thanos | 对象存储 + Sidecar | 无限存储 | 架构复杂 |
| VictoriaMetrics | 远程存储 | 高性能 | 需要额外部署 |
| Cortex | 多租户存储 | 多租户 | 架构复杂 |

**Thanos 配置**:

```yaml
# Prometheus 配置
global:
  external_labels:
    cluster: 'production'
    replica: 'prometheus-1'

# Thanos Sidecar
thanos sidecar \
  --tsdb.path=/data/prometheus \
  --prometheus.url=http://localhost:9090 \
  --objstore.config-file=/etc/thanos/object-store.yaml

# object-store.yaml
type: S3
config:
  bucket: thanos-storage
  endpoint: s3.amazonaws.com
  access_key: xxx
  secret_key: xxx
```

### Q4:Prometheus 的 PromQL 如何优化?

**优化原则**:

1. **使用 Recording Rules 预计算**:

```yaml
groups:
  - name: recording_rules
    rules:
      - record: job:http_requests:rate5m
        expr: sum by (job) (rate(http_requests_total[5m]))
```

2. **避免高基数聚合**:

```promql
# 错误:聚合所有标签
sum(http_requests_total)

# 正确:指定聚合标签
sum by (job) (http_requests_total)
```

3. **使用 rate 而非 increase**:

```promql
# 推荐
rate(http_requests_total[5m])

# 不推荐
increase(http_requests_total[5m])
```

4. **合理使用时间窗口**:

```promql
# 短时间窗口
rate(http_requests_total[1m])  # 更实时

# 长时间窗口
rate(http_requests_total[5m])  # 更平滑
```

### Q5:Prometheus 如何与 Kubernetes 集成?

**kube-prometheus-stack**:

```yaml
# Helm 安装
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack

# 包含组件:
# - Prometheus Operator
# - Prometheus Server
# - Alertmanager
# - Grafana
# - Node Exporter
# - Kube State Metrics
```

**自定义 ServiceMonitor**:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: my-app
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: my-app
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
  namespaceSelector:
    matchNames:
      - production
```

## 参考资源

- [Prometheus 官方文档](https://prometheus.io/docs/)
- [Prometheus 最佳实践](https://prometheus.io/docs/practices/)
- [Thanos 文档](https://thanos.io/)
- [VictoriaMetrics 文档](https://docs.victoriametrics.com/)
- [kube-prometheus-stack](https://github.com/prometheus-operator/kube-prometheus)

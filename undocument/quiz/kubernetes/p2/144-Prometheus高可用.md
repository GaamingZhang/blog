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
  - 高可用
  - HA
---

# Prometheus 高可用部署

## 引言

在生产环境中，监控系统的可用性至关重要。Prometheus 本身不提供内置的高可用方案，需要通过架构设计实现高可用。本文介绍 Prometheus 高可用部署的方案和最佳实践。

## 高可用架构

### 高可用方案

```
┌─────────────────────────────────────────────────────────────┐
│              Prometheus 高可用方案                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  方案一：多副本独立部署                                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Prometheus-1  ◄────┐                              │   │
│  │  Prometheus-2  ◄────┼──── Alertmanager（集群）     │   │
│  │  Prometheus-3  ◄────┘                              │   │
│  │  • 各实例独立采集                                   │   │
│  │  • Alertmanager 去重                               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  方案二：Thanos 方案                                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Prometheus + Sidecar ──► Thanos Query              │   │
│  │  Prometheus + Sidecar ──► Thanos Query              │   │
│  │  • 统一查询接口                                     │   │
│  │  • 对象存储长期存储                                 │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  方案三：VictoriaMetrics 集群                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  vmagent ──► vminsert ──► vmstorage ──► vmselect   │   │
│  │  • 分布式架构                                       │   │
│  │  • 内置高可用                                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 多副本部署

### Prometheus 多副本

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
        image: prom/prometheus:v2.45.0
        args:
        - "--config.file=/etc/prometheus/prometheus.yml"
        - "--storage.tsdb.path=/prometheus"
        - "--storage.tsdb.retention.time=15d"
        - "--web.external-url=http://prometheus:9090"
        ports:
        - containerPort: 9090
        volumeMounts:
        - name: config
          mountPath: /etc/prometheus
        - name: storage
          mountPath: /prometheus
      volumes:
      - name: config
        configMap:
          name: prometheus-config
      - name: storage
        emptyDir: {}
```

### Alertmanager 集群

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: alertmanager
  namespace: monitoring
spec:
  replicas: 3
  serviceName: alertmanager
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
        image: prom/alertmanager:v0.25.0
        args:
        - "--config.file=/etc/alertmanager/alertmanager.yml"
        - "--storage.path=/alertmanager"
        - "--cluster.listen-address=0.0.0.0:9094"
        - "--cluster.peer=alertmanager-0.alertmanager:9094"
        - "--cluster.peer=alertmanager-1.alertmanager:9094"
        - "--cluster.peer=alertmanager-2.alertmanager:9094"
        ports:
        - containerPort: 9093
        - containerPort: 9094
        volumeMounts:
        - name: config
          mountPath: /etc/alertmanager
        - name: storage
          mountPath: /alertmanager
      volumes:
      - name: config
        configMap:
          name: alertmanager-config
      - name: storage
        emptyDir: {}
```

### Alertmanager 集群配置

```yaml
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 1h
  receiver: 'email-notifications'

receivers:
- name: 'email-notifications'
  email_configs:
  - to: 'admin@example.com'
    send_resolved: true
```

## Thanos 高可用

### Thanos 组件

```
┌─────────────────────────────────────────────────────────────┐
│                  Thanos 组件                                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Sidecar：                                                   │
│  • 与 Prometheus 一起部署                                   │
│  • 上传数据到对象存储                                       │
│  • 暴露 Store API                                          │
│                                                              │
│  Query：                                                     │
│  • 统一查询接口                                             │
│  • 查询多个数据源                                           │
│  • 实现 Prometheus API                                      │
│                                                              │
│  Store：                                                     │
│  • 查询对象存储中的历史数据                                 │
│                                                              │
│  Compact：                                                   │
│  • 压缩对象存储中的数据                                     │
│                                                              │
│  Receive：                                                   │
│  • 接收远程写入                                             │
│  • 可选组件                                                 │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Thanos Query 部署

```yaml
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
        image: thanosio/thanos:v0.31.0
        args:
        - "query"
        - "--http-address=0.0.0.0:19192"
        - "--store=dnssrv+_grpc._tcp.prometheus.monitoring.svc.cluster.local"
        - "--store=dnssrv+_grpc._tcp.thanos-store.monitoring.svc.cluster.local"
        ports:
        - containerPort: 19192
```

### Thanos Store 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: thanos-store
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: thanos-store
  template:
    metadata:
      labels:
        app: thanos-store
    spec:
      containers:
      - name: thanos-store
        image: thanosio/thanos:v0.31.0
        args:
        - "store"
        - "--data-dir=/data"
        - "--objstore.config-file=/etc/thanos/objectstore.yaml"
        ports:
        - containerPort: 10901
        volumeMounts:
        - name: data
          mountPath: /data
        - name: thanos-config
          mountPath: /etc/thanos
      volumes:
      - name: data
        emptyDir: {}
      - name: thanos-config
        configMap:
          name: thanos-config
```

## VictoriaMetrics 集群

### VictoriaMetrics 集群架构

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vminsert
  namespace: monitoring
spec:
  replicas: 2
  selector:
    matchLabels:
      app: vminsert
  template:
    spec:
      containers:
      - name: vminsert
        image: victoriametrics/vminsert:latest
        args:
        - "--storageNode=vmstorage-0:8400,vmstorage-1:8400"
        ports:
        - containerPort: 8480
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vmselect
  namespace: monitoring
spec:
  replicas: 2
  selector:
    matchLabels:
      app: vmselect
  template:
    spec:
      containers:
      - name: vmselect
        image: victoriametrics/vmselect:latest
        args:
        - "--storageNode=vmstorage-0:8401,vmstorage-1:8401"
        ports:
        - containerPort: 8481
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: vmstorage
  namespace: monitoring
spec:
  replicas: 2
  serviceName: vmstorage
  selector:
    matchLabels:
      app: vmstorage
  template:
    spec:
      containers:
      - name: vmstorage
        image: victoriametrics/vmstorage:latest
        args:
        - "--retentionPeriod=12"
        - "--storageDataPath=/storage"
        ports:
        - containerPort: 8400
        - containerPort: 8401
        volumeMounts:
        - name: storage
          mountPath: /storage
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: vmstorage-pvc
```

## 负载均衡

### Service 配置

```yaml
apiVersion: v1
kind: Service
metadata:
  name: prometheus
  namespace: monitoring
spec:
  selector:
    app: prometheus
  ports:
  - port: 9090
    targetPort: 9090
---
apiVersion: v1
kind: Service
metadata:
  name: thanos-query
  namespace: monitoring
spec:
  selector:
    app: thanos-query
  ports:
  - port: 19192
    targetPort: 19192
```

## 最佳实践

### 1. 多副本部署

```yaml
spec:
  replicas: 2
```

### 2. Alertmanager 集群

```yaml
args:
- "--cluster.peer=alertmanager-0.alertmanager:9094"
- "--cluster.peer=alertmanager-1.alertmanager:9094"
```

### 3. 使用 Thanos 或 VictoriaMetrics

```yaml
remote_write:
- url: "http://thanos-receive:19291/api/v1/receive"
```

### 4. 监控监控系统

```yaml
- job_name: 'prometheus'
  static_configs:
  - targets: ['localhost:9090']
```

## 面试回答

**问题**: Prometheus 是否支持高可用性部署？

**回答**: Prometheus 本身不提供内置高可用，但可以通过架构设计实现高可用：

**多副本部署**：部署多个 Prometheus 实例，各自独立采集数据。通过 Alertmanager 集群实现告警去重。优点：简单易实现；缺点：数据冗余、资源消耗大。

**Alertmanager 集群**：使用 StatefulSet 部署 Alertmanager 集群（通常 3 个节点）。通过 --cluster.peer 参数配置集群节点。集群自动选举 Leader，实现告警去重和高可用。

**Thanos 方案**：Thanos Sidecar 与 Prometheus 一起部署，上传数据到对象存储。Thanos Query 提供统一查询接口，查询多个 Prometheus 实例和对象存储。支持长期存储、多集群查询、高可用。

**VictoriaMetrics 集群**：分布式架构，分为 vminsert（写入）、vmselect（查询）、vmstorage（存储）三个组件。支持水平扩展、内置高可用、高性能。

**最佳实践**：生产环境至少部署 2 个 Prometheus 副本；Alertmanager 使用 3 节点集群；长期存储使用 Thanos 或 VictoriaMetrics；监控系统自身也需要被监控；配置资源限制避免资源耗尽。

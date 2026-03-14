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
  - 持久化存储
  - 长期存储
---

# Prometheus 持久化存储配置

## 引言

Prometheus 默认使用本地存储，数据保留时间有限。对于需要长期保存监控数据的场景，需要配置持久化存储或使用远程存储方案。本文介绍 Prometheus 本地持久化存储和远程存储的配置方法。

## Prometheus 存储概述

### 存储架构

```
┌─────────────────────────────────────────────────────────────┐
│              Prometheus 存储架构                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  本地存储：                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  TSDB (时序数据库)                                   │   │
│  │  • 数据块存储                                       │   │
│  │  • WAL 预写日志                                     │   │
│  │  • 压缩和保留                                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  远程存储：                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Remote Write -> Thanos/Cortex/VictoriaMetrics      │   │
│  │  Remote Read  <- Thanos/Cortex/VictoriaMetrics      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 本地持久化存储

### 使用 PersistentVolume

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: prometheus-pvc
  namespace: monitoring
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
  storageClassName: standard
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: monitoring
spec:
  template:
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:v2.45.0
        args:
        - "--config.file=/etc/prometheus/prometheus.yml"
        - "--storage.tsdb.path=/prometheus"
        - "--storage.tsdb.retention.time=30d"
        - "--storage.tsdb.retention.size=80GB"
        volumeMounts:
        - name: storage
          mountPath: /prometheus
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: prometheus-pvc
```

### 存储参数配置

```bash
prometheus \
  --storage.tsdb.path=/prometheus \
  --storage.tsdb.retention.time=30d \
  --storage.tsdb.retention.size=80GB \
  --storage.tsdb.min-block-duration=2h \
  --storage.tsdb.max-block-duration=6h \
  --storage.tsdb.wal-compression
```

### 参数说明

| 参数 | 说明 | 默认值 |
|-----|------|--------|
| storage.tsdb.path | 数据存储路径 | data/ |
| storage.tsdb.retention.time | 数据保留时间 | 15d |
| storage.tsdb.retention.size | 数据保留大小 | 无限制 |
| storage.tsdb.min-block-duration | 最小块时长 | 2h |
| storage.tsdb.max-block-duration | 最大块时长 | 6h |
| storage.tsdb.wal-compression | WAL 压缩 | false |

## 远程存储配置

### Remote Write 配置

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      external_labels:
        cluster: 'production'
        region: 'beijing'

    remote_write:
    - url: "http://thanos-receive:19291/api/v1/receive"
      queue_config:
        capacity: 10000
        max_shards: 200
        max_samples_per_send: 500
        batch_send_deadline: 5s
        min_shards: 1
        min_backoff: 30ms
        max_backoff: 100ms
```

### Remote Read 配置

```yaml
remote_read:
- url: "http://thanos-query:19192/api/v1/read"
  read_recent: true
  required_matchers:
    cluster: production
```

## Thanos 长期存储

### Thanos 架构

```
┌─────────────────────────────────────────────────────────────┐
│                  Thanos 架构                                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Prometheus                           │  │
│  │  • 本地存储（短期）                                  │  │
│  │  • Sidecar 上传数据到对象存储                        │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Thanos Sidecar                       │  │
│  │  • 读取本地数据                                      │  │
│  │  • 上传到对象存储                                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Object Storage                       │  │
│  │  • S3/GCS/MinIO                                      │  │
│  │  • 长期存储                                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Thanos Query                         │  │
│  │  • 统一查询接口                                      │  │
│  │  • 查询本地和远程数据                                │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Thanos Sidecar 部署

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: prometheus
  namespace: monitoring
spec:
  template:
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:v2.45.0
        args:
        - "--config.file=/etc/prometheus/prometheus.yml"
        - "--storage.tsdb.path=/prometheus"
        - "--storage.tsdb.retention.time=6h"
        - "--storage.tsdb.max-block-duration=2h"
        ports:
        - containerPort: 9090
        volumeMounts:
        - name: storage
          mountPath: /prometheus
      - name: thanos-sidecar
        image: thanosio/thanos:v0.31.0
        args:
        - "sidecar"
        - "--tsdb.path=/prometheus"
        - "--prometheus.url=http://localhost:9090"
        - "--objstore.config-file=/etc/thanos/objectstore.yaml"
        ports:
        - containerPort: 10902
        volumeMounts:
        - name: storage
          mountPath: /prometheus
        - name: thanos-config
          mountPath: /etc/thanos
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: prometheus-pvc
      - name: thanos-config
        configMap:
          name: thanos-config
```

### Thanos 对象存储配置

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: thanos-config
  namespace: monitoring
data:
  objectstore.yaml: |
    type: S3
    config:
      bucket: "prometheus-data"
      endpoint: "s3.amazonaws.com"
      region: "us-east-1"
      access_key: "<access-key>"
      secret_key: "<secret-key>"
```

## VictoriaMetrics 配置

### VictoriaMetrics 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: victoriametrics
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: victoriametrics
  template:
    metadata:
      labels:
        app: victoriametrics
    spec:
      containers:
      - name: victoriametrics
        image: victoriametrics/victoria-metrics:latest
        args:
        - "--storageDataPath=/storage"
        - "--retentionPeriod=12"
        ports:
        - containerPort: 8428
        volumeMounts:
        - name: storage
          mountPath: /storage
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: vm-pvc
```

### Prometheus 远程写入 VictoriaMetrics

```yaml
remote_write:
- url: "http://victoriametrics:8428/api/v1/write"
  queue_config:
    max_samples_per_send: 10000
```

## 最佳实践

### 1. 本地存储配置

```yaml
args:
- "--storage.tsdb.retention.time=15d"
- "--storage.tsdb.retention.size=50GB"
- "--storage.tsdb.wal-compression"
```

### 2. 使用远程存储

```yaml
remote_write:
- url: "http://thanos-receive:19291/api/v1/receive"
  queue_config:
    capacity: 10000
    max_shards: 200
```

### 3. 监控存储使用

```yaml
- alert: PrometheusStorageNearFull
  expr: prometheus_tsdb_retention_limit_bytes - prometheus_tsdb_storage_size_bytes < 1GB
  for: 5m
  labels:
    severity: warning
```

### 4. 定期备份

```bash
# 备份 Prometheus 数据
kubectl exec prometheus-0 -n monitoring -- \
  wget -O- http://localhost:9090/api/v1/snapshot > snapshot.tar
```

## 面试回答

**问题**: 如何在 Prometheus 中配置持久化存储？

**回答**: Prometheus 持久化存储分为本地持久化和远程存储两种方式：

**本地持久化存储**：使用 PersistentVolume 挂载存储卷到 Prometheus 容器。配置 storage.tsdb.path 指定存储路径；storage.tsdb.retention.time 设置数据保留时间（默认 15d）；storage.tsdb.retention.size 设置存储大小限制。启用 storage.tsdb.wal-compression 压缩 WAL 日志减少存储空间。

**远程存储**：通过 remote_write 将数据写入远程存储系统，支持 Thanos、Cortex、VictoriaMetrics 等。remote_read 从远程存储读取历史数据。适用于长期存储、多集群查询场景。

**Thanos 方案**：Thanos Sidecar 与 Prometheus 一起部署，读取本地数据并上传到对象存储（S3/GCS/MinIO）。Thanos Query 提供统一查询接口，查询本地和远程数据。支持长期存储（数年）、高可用、多集群查询。

**VictoriaMetrics 方案**：高性能时序数据库，兼容 Prometheus 协议。配置 Prometheus remote_write 到 VictoriaMetrics。支持更高的压缩率和查询性能。

**最佳实践**：本地存储设置合理的保留时间和大小限制；长期存储使用 Thanos 或 VictoriaMetrics；启用 WAL 压缩；监控存储使用情况；定期备份数据。

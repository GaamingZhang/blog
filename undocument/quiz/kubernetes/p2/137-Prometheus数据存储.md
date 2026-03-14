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
  - 数据存储
  - 保留策略
---

# Prometheus 数据存储和保留策略

## 引言

Prometheus 存储大量的时序监控数据，合理配置数据存储和保留策略对于系统性能和存储成本至关重要。理解 Prometheus 的存储机制和保留策略配置，是运维 Prometheus 的关键技能。

## Prometheus 存储概述

### 存储架构

```
┌─────────────────────────────────────────────────────────────┐
│              Prometheus 存储架构                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Ingestion                            │  │
│  │  • 接收抓取数据                                      │  │
│  │  • 追加到内存                                        │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Head Block                           │  │
│  │  • 内存中的最新数据                                  │  │
│  │  • 写入 WAL（预写日志）                              │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Compaction                           │  │
│  │  • 合并数据块                                        │  │
│  │  • 压缩数据                                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Persistent Blocks                    │  │
│  │  • 磁盘上的数据块                                    │  │
│  │  • 按时间范围组织                                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 存储组件

| 组件 | 说明 |
|-----|------|
| Head Block | 内存中的最新数据 |
| WAL | 预写日志，保证数据持久性 |
| Compaction | 数据压缩和合并 |
| Persistent Blocks | 磁盘上的持久化数据块 |

## 数据保留策略

### 时间保留

```yaml
global:
  retention: 15d
```

### 启动参数配置

```bash
prometheus \
  --storage.tsdb.path=/prometheus \
  --storage.tsdb.retention.time=15d
```

### 大小保留

```bash
prometheus \
  --storage.tsdb.retention.size=50GB
```

### 同时配置时间和大小

```bash
prometheus \
  --storage.tsdb.retention.time=30d \
  --storage.tsdb.retention.size=100GB
```

## 存储配置

### 本地存储配置

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
      storage: 50Gi
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
        - "--storage.tsdb.retention.time=15d"
        - "--storage.tsdb.retention.size=50GB"
        volumeMounts:
        - name: storage
          mountPath: /prometheus
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: prometheus-pvc
```

### 存储路径结构

```
/prometheus/
├── 01H1A2B3C4D5E6F7G8H9/    # 数据块目录
│   ├── chunks/              # 数据文件
│   ├── index/               # 索引文件
│   └── meta.json            # 元数据
├── wal/                     # 预写日志
│   ├── 000001
│   ├── 000002
│   └── checkpoint.000001
└── queries.active/          # 活跃查询
```

## 数据压缩

### 压缩参数

```bash
prometheus \
  --storage.tsdb.min-block-duration=2h \
  --storage.tsdb.max-block-duration=6h
```

### 压缩过程

```
┌─────────────────────────────────────────────────────────────┐
│                  数据压缩过程                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Head Block 数据写入 WAL                                  │
│       │                                                      │
│       ▼                                                      │
│  2. Head Block 数据刷新到磁盘（2 小时块）                    │
│       │                                                      │
│       ▼                                                      │
│  3. 小块合并为大块（6 小时块）                               │
│       │                                                      │
│       ▼                                                      │
│  4. 大块继续合并（1 天块）                                   │
│       │                                                      │
│       ▼                                                      │
│  5. 删除过期数据块                                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 远程存储

### 远程写入配置

```yaml
remote_write:
- url: "http://remote-storage:8080/write"
  queue_config:
    capacity: 10000
    max_shards: 200
    max_samples_per_send: 500
    batch_send_deadline: 5s
```

### 远程读取配置

```yaml
remote_read:
- url: "http://remote-storage:8080/read"
  read_recent: true
```

### 常用远程存储

```
┌─────────────────────────────────────────────────────────────┐
│                  常用远程存储                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  长期存储：                                                  │
│  • Thanos：支持长期存储、高可用                             │
│  • Cortex：支持多租户、长期存储                             │
│  • VictoriaMetrics：高性能、压缩率高                        │
│                                                              │
│  云存储：                                                    │
│  • Amazon Cortex                                            │
│  • Google Cloud Cortex                                      │
│  • Azure Monitor                                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 存储优化

### 1. 减少高基数指标

```yaml
metric_relabel_configs:
- source_labels: [__name__]
  regex: 'high_cardinality_metric.*'
  action: drop
```

### 2. 过滤不需要的指标

```yaml
metric_relabel_configs:
- source_labels: [__name__]
  regex: 'go_.*'
  action: drop
```

### 3. 增加抓取间隔

```yaml
global:
  scrape_interval: 30s
```

### 4. 使用远程存储

```yaml
remote_write:
- url: "http://thanos-receive:19291/api/v1/receive"
```

## 存储监控

### 存储相关指标

```
prometheus_tsdb_head_series
prometheus_tsdb_head_chunks
prometheus_tsdb_wal_corruptions_total
prometheus_tsdb_compactions_total
prometheus_tsdb_size_retentions_total
prometheus_tsdb_time_retentions_total
```

### 查看存储状态

```bash
curl http://localhost:9090/api/v1/status/tsdb

du -sh /prometheus
```

## 最佳实践

### 1. 合理设置保留时间

```bash
--storage.tsdb.retention.time=15d
```

### 2. 设置存储大小限制

```bash
--storage.tsdb.retention.size=50GB
```

### 3. 使用持久化存储

```yaml
volumeMounts:
- name: storage
  mountPath: /prometheus
```

### 4. 监控存储使用

```yaml
- alert: PrometheusStorageNearFull
  expr: prometheus_tsdb_retention_limit_bytes - prometheus_tsdb_storage_size_bytes < 1GB
```

## 面试回答

**问题**: Prometheus 如何处理数据存储和保留策略？

**回答**: Prometheus 数据存储和保留策略包括本地存储机制、保留策略配置、远程存储等方面：

**存储机制**：Prometheus 使用 TSDB（时序数据库）存储数据。数据首先写入内存中的 Head Block，同时写入 WAL（预写日志）保证持久性。定期将 Head Block 数据刷新到磁盘，形成持久化数据块。后台进程进行数据压缩和合并，小块合并为大块。

**保留策略**：支持两种保留策略。**时间保留**通过 `--storage.tsdb.retention.time` 配置，默认 15 天；**大小保留**通过 `--storage.tsdb.retention.size` 配置，限制存储大小。可以同时配置，任一条件满足时触发数据删除。

**存储配置**：使用 PersistentVolume 持久化存储数据。存储路径包含数据块目录、WAL 目录、活跃查询目录。数据块按时间范围组织，包含 chunks、index、meta.json。

**远程存储**：通过 remote_write 配置将数据写入远程存储，支持 Thanos、Cortex、VictoriaMetrics 等。remote_read 配置从远程存储读取历史数据。适用于长期存储和多集群场景。

**存储优化**：减少高基数指标；过滤不需要的指标；增加抓取间隔；使用远程存储分担本地存储压力。

**最佳实践**：合理设置保留时间和大小限制；使用持久化存储；监控存储使用情况；配置告警规则监控存储容量。

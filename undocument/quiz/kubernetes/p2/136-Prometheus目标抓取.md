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
  - 目标抓取
  - 配置
---

# Prometheus 目标抓取配置

## 引言

Prometheus 通过抓取（Scrape）目标来采集监控指标。配置目标抓取是 Prometheus 监控的基础，包括静态配置、服务发现、抓取参数等。理解目标抓取配置，对于构建完整的监控系统至关重要。

## 目标抓取概述

### 抓取流程

```
┌─────────────────────────────────────────────────────────────┐
│                  Prometheus 抓取流程                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Prometheus Server                        │  │
│  │  • 读取配置文件                                      │  │
│  │  • 服务发现目标                                      │  │
│  │  • 定期抓取指标                                      │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Target                               │  │
│  │  • /metrics 端点                                     │  │
│  │  • 暴露 Prometheus 格式指标                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  TSDB                                 │  │
│  │  • 存储时序数据                                      │  │
│  │  • 压缩数据                                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 抓取配置结构

```yaml
scrape_configs:
- job_name: '<job-name>'
  scrape_interval: <duration>
  scrape_timeout: <duration>
  metrics_path: <path>
  scheme: <scheme>
  static_configs:
  - targets: ['<host>:<port>']
```

## 静态配置

### 基本静态配置

```yaml
scrape_configs:
- job_name: 'prometheus'
  scrape_interval: 15s
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: http
  static_configs:
  - targets:
    - 'localhost:9090'
```

### 多目标配置

```yaml
scrape_configs:
- job_name: 'node-exporter'
  static_configs:
  - targets:
    - 'node-1:9100'
    - 'node-2:9100'
    - 'node-3:9100'
    labels:
      environment: 'production'
```

### 分组配置

```yaml
scrape_configs:
- job_name: 'node-exporter'
  static_configs:
  - targets:
    - 'node-1:9100'
    - 'node-2:9100'
    labels:
      datacenter: 'beijing'
  - targets:
    - 'node-3:9100'
    - 'node-4:9100'
    labels:
      datacenter: 'shanghai'
```

## Kubernetes 服务发现

### Pod 服务发现

```yaml
scrape_configs:
- job_name: 'kubernetes-pods'
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    action: keep
    regex: true
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    action: replace
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
    action: replace
    regex: ([^:]+)(?::\d+)?;(\d+)
    replacement: $1:$2
    target_label: __address__
```

### Service 服务发现

```yaml
scrape_configs:
- job_name: 'kubernetes-services'
  kubernetes_sd_configs:
  - role: service
  relabel_configs:
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape]
    action: keep
    regex: true
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scheme]
    action: replace
    target_label: __scheme__
    regex: (https?)
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_path]
    action: replace
    target_label: __metrics_path__
    regex: (.+)
```

### Node 服务发现

```yaml
scrape_configs:
- job_name: 'kubernetes-nodes'
  kubernetes_sd_configs:
  - role: node
  relabel_configs:
  - source_labels: [__address__]
    regex: (.+):(.+)
    target_label: __address__
    replacement: ${1}:9100
```

### Endpoints 服务发现

```yaml
scrape_configs:
- job_name: 'kubernetes-endpoints'
  kubernetes_sd_configs:
  - role: endpoints
  relabel_configs:
  - source_labels: [__meta_kubernetes_endpoints_name]
    action: keep
    regex: node-exporter
```

## 抓取参数配置

### 基本参数

```yaml
scrape_configs:
- job_name: 'myapp'
  scrape_interval: 30s
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: https
  static_configs:
  - targets:
    - 'app.example.com:8080'
```

### HTTP 参数

```yaml
scrape_configs:
- job_name: 'myapp'
  metrics_path: /metrics
  params:
    format: ['prometheus']
    filter: ['cpu,memory']
  static_configs:
  - targets:
    - 'app:8080'
```

### 认证配置

```yaml
scrape_configs:
- job_name: 'myapp'
  basic_auth:
    username: admin
    password: secret
  static_configs:
  - targets:
    - 'app:8080'
```

### TLS 配置

```yaml
scrape_configs:
- job_name: 'myapp'
  scheme: https
  tls_config:
    ca_file: /etc/prometheus/ca.crt
    cert_file: /etc/prometheus/client.crt
    key_file: /etc/prometheus/client.key
    insecure_skip_verify: false
  static_configs:
  - targets:
    - 'app:8080'
```

## Relabel 配置

### Relabel 功能

```
┌─────────────────────────────────────────────────────────────┐
│                  Relabel 功能                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  动作类型：                                                  │
│  • keep：保留匹配的目标                                      │
│  • drop：丢弃匹配的目标                                      │
│  • replace：替换标签值                                       │
│  • labelmap：批量重命名标签                                  │
│  • labeldrop：删除标签                                       │
│                                                              │
│  常用场景：                                                  │
│  • 过滤目标                                                  │
│  • 添加标签                                                  │
│  • 修改标签                                                  │
│  • 删除标签                                                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 过滤目标

```yaml
relabel_configs:
- source_labels: [__meta_kubernetes_namespace]
  action: keep
  regex: production|staging
```

### 添加标签

```yaml
relabel_configs:
- source_labels: [__meta_kubernetes_namespace]
  target_label: kubernetes_namespace
  action: replace
```

### 修改标签

```yaml
relabel_configs:
- source_labels: [__address__]
  regex: '([^:]+):\d+'
  target_label: instance
  replacement: '$1'
```

### 删除标签

```yaml
metric_relabel_configs:
- regex: 'go_.*'
  action: labeldrop
```

## 全局配置

### 基本全局配置

```yaml
global:
  scrape_interval: 15s
  scrape_timeout: 10s
  evaluation_interval: 15s
  external_labels:
    cluster: 'production'
    environment: 'prod'
```

### 完整配置示例

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'production'

scrape_configs:
- job_name: 'prometheus'
  static_configs:
  - targets: ['localhost:9090']

- job_name: 'kubernetes-pods'
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    action: keep
    regex: true
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    action: replace
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
    action: replace
    regex: ([^:]+)(?::\d+)?;(\d+)
    replacement: $1:$2
    target_label: __address__
  - action: labelmap
    regex: __meta_kubernetes_pod_label_(.+)
  - source_labels: [__meta_kubernetes_namespace]
    action: replace
    target_label: kubernetes_namespace
  - source_labels: [__meta_kubernetes_pod_name]
    action: replace
    target_label: kubernetes_pod_name
```

## 最佳实践

### 1. 合理设置抓取间隔

```yaml
global:
  scrape_interval: 15s

scrape_configs:
- job_name: 'critical-service'
  scrape_interval: 5s
```

### 2. 使用标签区分环境

```yaml
static_configs:
- targets: ['app:8080']
  labels:
    environment: 'production'
    region: 'beijing'
```

### 3. 过滤不需要的指标

```yaml
metric_relabel_configs:
- source_labels: [__name__]
  regex: 'go_.*'
  action: drop
```

## 面试回答

**问题**: 如何配置 Prometheus 进行目标抓取？

**回答**: Prometheus 目标抓取配置包括静态配置、服务发现、抓取参数、Relabel 等部分：

**静态配置**：在 static_configs 中直接指定目标地址，适用于固定目标。配置 targets 列表和可选的 labels。

**Kubernetes 服务发现**：通过 kubernetes_sd_configs 配置，支持 pod、service、node、endpoints 四种角色。Pod 角色发现所有 Pod，通过注解 prometheus.io/scrape、prometheus.io/port、prometheus.io/path 控制抓取行为。

**抓取参数**：scrape_interval（抓取间隔，默认 15s）、scrape_timeout（超时时间，默认 10s）、metrics_path（指标路径，默认 /metrics）、scheme（协议，http/https）、params（URL 参数）、basic_auth（基本认证）、tls_config（TLS 配置）。

**Relabel 配置**：用于过滤、添加、修改、删除标签。常用动作：keep（保留匹配目标）、drop（丢弃匹配目标）、replace（替换标签值）、labelmap（批量重命名）、labeldrop（删除标签）。relabel_configs 在抓取前处理，metric_relabel_configs 在存储前处理。

**最佳实践**：合理设置抓取间隔，关键服务使用更短的间隔；使用标签区分环境和区域；过滤不需要的指标减少存储；配置认证和 TLS 保证安全。

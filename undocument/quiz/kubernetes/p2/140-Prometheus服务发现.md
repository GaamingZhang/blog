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
  - 服务发现
  - SD
---

# Prometheus 服务发现机制

## 引言

服务发现是 Prometheus 的核心功能之一，它允许 Prometheus 自动发现监控目标，无需手动配置。在动态变化的 Kubernetes 环境中，服务发现机制尤为重要，可以自动感知新创建的 Pod、Service 等资源。

## 服务发现概述

### 服务发现类型

```
┌─────────────────────────────────────────────────────────────┐
│              Prometheus 服务发现类型                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  静态配置：                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • 手动配置目标地址                                 │   │
│  │  • 适用于固定目标                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  文件服务发现：                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • 从 JSON/YAML 文件读取目标                        │   │
│  │  • 文件更新自动重新加载                             │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Kubernetes 服务发现：                                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • 自动发现 Kubernetes 资源                         │   │
│  │  • 支持 Pod、Service、Node、Endpoints               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  云平台服务发现：                                            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • AWS EC2                                          │   │
│  │  • Azure                                            │   │
│  │  • GCE                                              │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  其他服务发现：                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • Consul                                           │   │
│  │  • DNS                                              │   │
│  │  • Zookeeper                                        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 服务发现流程

```
┌─────────────────────────────────────────────────────────────┐
│              服务发现流程                                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Service Discovery                        │  │
│  │  • 发现目标                                          │  │
│  │  • 附加元数据标签                                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Relabel Configs                          │  │
│  │  • 过滤目标                                          │  │
│  │  • 修改标签                                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Scrape                                   │  │
│  │  • 抓取指标                                          │  │
│  │  • 存储数据                                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
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
  - action: labelmap
    regex: __meta_kubernetes_pod_label_(.+)
  - source_labels: [__meta_kubernetes_namespace]
    action: replace
    target_label: kubernetes_namespace
  - source_labels: [__meta_kubernetes_pod_name]
    action: replace
    target_label: kubernetes_pod_name
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
  - source_labels: [__address__, __meta_kubernetes_service_annotation_prometheus_io_port]
    action: replace
    target_label: __address__
    regex: ([^:]+)(?::\d+)?;(\d+)
    replacement: $1:$2
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
  - source_labels: [__meta_kubernetes_node_name]
    target_label: instance
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

## Kubernetes 元数据标签

### Pod 元数据

```
__meta_kubernetes_namespace
__meta_kubernetes_pod_name
__meta_kubernetes_pod_ip
__meta_kubernetes_pod_label_<labelname>
__meta_kubernetes_pod_annotation_<annotationname>
__meta_kubernetes_pod_container_name
__meta_kubernetes_pod_container_port_name
__meta_kubernetes_pod_container_port_number
__meta_kubernetes_pod_ready
__meta_kubernetes_pod_phase
```

### Service 元数据

```
__meta_kubernetes_namespace
__meta_kubernetes_service_name
__meta_kubernetes_service_label_<labelname>
__meta_kubernetes_service_annotation_<annotationname>
__meta_kubernetes_service_cluster_ip
__meta_kubernetes_service_port_name
__meta_kubernetes_service_port_protocol
```

### Node 元数据

```
__meta_kubernetes_node_name
__meta_kubernetes_node_label_<labelname>
__meta_kubernetes_node_annotation_<annotationname>
__meta_kubernetes_node_address_<address_type>
```

## 文件服务发现

### JSON 文件格式

```json
[
  {
    "targets": ["localhost:9090", "localhost:9100"],
    "labels": {
      "job": "prometheus",
      "environment": "production"
    }
  },
  {
    "targets": ["app-1:8080", "app-2:8080"],
    "labels": {
      "job": "myapp",
      "environment": "staging"
    }
  }
]
```

### YAML 文件格式

```yaml
- targets:
  - localhost:9090
  - localhost:9100
  labels:
    job: prometheus
    environment: production
- targets:
  - app-1:8080
  - app-2:8080
  labels:
    job: myapp
    environment: staging
```

### 配置文件服务发现

```yaml
scrape_configs:
- job_name: 'file-sd'
  file_sd_configs:
  - files:
    - /etc/prometheus/targets/*.json
    - /etc/prometheus/targets/*.yml
    refresh_interval: 5m
```

## DNS 服务发现

```yaml
scrape_configs:
- job_name: 'dns-sd'
  dns_sd_configs:
  - names:
    - myapp.default.svc.cluster.local
    - node-exporter.monitoring.svc.cluster.local
    type: A
    port: 9100
    refresh_interval: 30s
```

## Consul 服务发现

```yaml
scrape_configs:
- job_name: 'consul-sd'
  consul_sd_configs:
  - server: 'consul-server:8500'
    services: ['myapp', 'node-exporter']
    tags: ['production']
    refresh_interval: 30s
```

## 最佳实践

### 1. 使用注解控制抓取

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

### 2. 使用标签过滤

```yaml
relabel_configs:
- source_labels: [__meta_kubernetes_namespace]
  action: keep
  regex: production|staging
```

### 3. 合理设置刷新间隔

```yaml
kubernetes_sd_configs:
- role: pod
  refresh_interval: 30s
```

### 4. 使用命名空间选择器

```yaml
kubernetes_sd_configs:
- role: pod
  namespaces:
    names:
    - production
    - staging
```

## 面试回答

**问题**: 什么是 Prometheus 的服务发现机制？

**回答**: Prometheus 服务发现机制允许自动发现监控目标，无需手动配置，特别适合动态变化的 Kubernetes 环境：

**服务发现类型**：**静态配置**手动指定目标地址，适用于固定目标；**文件服务发现**从 JSON/YAML 文件读取目标，文件更新自动重新加载；**Kubernetes 服务发现**自动发现 Kubernetes 资源，支持 Pod、Service、Node、Endpoints 四种角色；**云平台服务发现**支持 AWS EC2、Azure、GCE 等；**其他服务发现**支持 Consul、DNS、Zookeeper 等。

**Kubernetes 服务发现**：通过 kubernetes_sd_configs 配置，role 指定发现角色。Pod 角色发现所有 Pod，通过注解 prometheus.io/scrape、prometheus.io/port、prometheus.io/path 控制抓取行为。Service 角色发现 Service，通过 Endpoints 获取后端 Pod。Node 角色发现节点，常用于节点监控。Endpoints 角色发现 Endpoints 资源。

**元数据标签**：服务发现会附加丰富的元数据标签，如 __meta_kubernetes_namespace、__meta_kubernetes_pod_name、__meta_kubernetes_pod_label_* 等。通过 relabel_configs 可以过滤目标、修改标签、添加新标签。

**最佳实践**：使用注解控制 Pod 抓取行为；使用 relabel_configs 过滤不需要的目标；合理设置刷新间隔；使用命名空间选择器限制发现范围。

---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
  - Prometheus
tag:
  - Kubernetes
  - Prometheus
  - 服务发现
---

# 手动部署Prometheus自动发现Pod端点

## 问题背景

在Kubernetes环境中，Pod是动态变化的——它们会被创建、删除、重新调度。如果手动配置Prometheus的监控目标，每次Pod变化都需要更新配置，这显然不可行。

Prometheus的服务发现机制可以自动发现监控目标，本文介绍如何手动部署Prometheus并配置自动发现Pod端点。

## Prometheus服务发现机制

### 服务发现类型

Prometheus支持多种服务发现方式：

| 类型 | 说明 | 适用场景 |
|------|------|----------|
| static_configs | 静态配置 | 固定目标 |
| file_sd_configs | 文件服务发现 | 动态更新目标列表 |
| kubernetes_sd_configs | Kubernetes服务发现 | K8S环境 |
| consul_sd_configs | Consul服务发现 | 服务注册中心 |
| dns_sd_configs | DNS服务发现 | DNS轮询 |

### Kubernetes服务发现

`kubernetes_sd_configs`是专门为Kubernetes设计的服务发现机制，可以自动发现：

- Node：集群节点
- Pod：所有Pod
- Service：所有Service
- Endpoint：Service的后端Pod
- Ingress：Ingress资源

## 配置自动发现Pod端点

### 方式一：通过Endpoints发现

这是最常用的方式，通过Service的Endpoints发现后端Pod。

```yaml
global:
  scrape_interval: 15s

scrape_configs:
- job_name: 'kubernetes-endpoints'
  kubernetes_sd_configs:
  - role: endpoints
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
  - action: labelmap
    regex: __meta_kubernetes_service_label_(.+)
  - source_labels: [__meta_kubernetes_namespace]
    action: replace
    target_label: kubernetes_namespace
  - source_labels: [__meta_kubernetes_service_name]
    action: replace
    target_label: kubernetes_name
```

**使用方式**：在Service上添加注解

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-app
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
spec:
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
```

### 方式二：通过Pod发现

直接发现所有Pod，通过Pod注解筛选需要监控的Pod。

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

**使用方式**：在Pod上添加注解

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
spec:
  containers:
  - name: my-app
    image: my-app:latest
    ports:
    - containerPort: 8080
```

### 方式三：通过Service发现

发现所有Service，适合监控Service级别的指标。

```yaml
scrape_configs:
- job_name: 'kubernetes-services'
  kubernetes_sd_configs:
  - role: service
  metrics_path: /probe
  params:
    module: [http_2xx]
  relabel_configs:
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_probe]
    action: keep
    regex: true
  - source_labels: [__address__]
    target_label: __param_target
  - target_label: __address__
    replacement: blackbox-exporter:9115
  - source_labels: [__param_target]
    target_label: instance
  - action: labelmap
    regex: __meta_kubernetes_service_label_(.+)
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
  - source_labels: [__meta_kubernetes_service_name]
    target_label: kubernetes_name
```

## 完整部署示例

### 1. 创建命名空间和RBAC

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: monitoring
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prometheus
  namespace: monitoring
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: prometheus
rules:
- apiGroups: [""]
  resources:
  - nodes
  - nodes/proxy
  - nodes/metrics
  - services
  - endpoints
  - pods
  verbs: ["get", "list", "watch"]
- nonResourceURLs: ["/metrics", "/metrics/cadvisor"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus
subjects:
- kind: ServiceAccount
  name: prometheus
  namespace: monitoring
```

### 2. 创建Prometheus配置

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
      evaluation_interval: 15s

    scrape_configs:
    - job_name: 'kubernetes-apiservers'
      kubernetes_sd_configs:
      - role: endpoints
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        insecure_skip_verify: true
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      relabel_configs:
      - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
        action: keep
        regex: default;kubernetes;https

    - job_name: 'kubernetes-nodes'
      kubernetes_sd_configs:
      - role: node
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        insecure_skip_verify: true
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      relabel_configs:
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)
      - target_label: __address__
        replacement: kubernetes.default.svc:443
      - source_labels: [__meta_kubernetes_node_name]
        regex: (.+)
        target_label: __metrics_path__
        replacement: /api/v1/nodes/${1}/proxy/metrics

    - job_name: 'kubernetes-nodes-cadvisor'
      kubernetes_sd_configs:
      - role: node
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        insecure_skip_verify: true
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      relabel_configs:
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)
      - target_label: __address__
        replacement: kubernetes.default.svc:443
      - source_labels: [__meta_kubernetes_node_name]
        regex: (.+)
        target_label: __metrics_path__
        replacement: /api/v1/nodes/${1}/proxy/metrics/cadvisor

    - job_name: 'kubernetes-endpoints'
      kubernetes_sd_configs:
      - role: endpoints
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
      - action: labelmap
        regex: __meta_kubernetes_service_label_(.+)
      - source_labels: [__meta_kubernetes_namespace]
        action: replace
        target_label: kubernetes_namespace
      - source_labels: [__meta_kubernetes_service_name]
        action: replace
        target_label: kubernetes_name

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

### 3. 部署Prometheus

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      serviceAccountName: prometheus
      containers:
      - name: prometheus
        image: prom/prometheus:v2.45.0
        args:
        - "--config.file=/etc/prometheus/prometheus.yml"
        - "--storage.tsdb.path=/prometheus"
        - "--storage.tsdb.retention.time=15d"
        - "--web.enable-lifecycle"
        ports:
        - containerPort: 9090
        volumeMounts:
        - name: config
          mountPath: /etc/prometheus
        - name: data
          mountPath: /prometheus
      volumes:
      - name: config
        configMap:
          name: prometheus-config
      - name: data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus
  namespace: monitoring
spec:
  ports:
  - port: 9090
    targetPort: 9090
  selector:
    app: prometheus
```

## 服务发现元数据

Prometheus会为每个发现的目标添加丰富的元数据标签：

### Pod角色元数据

| 标签 | 说明 |
|------|------|
| __meta_kubernetes_pod_name | Pod名称 |
| __meta_kubernetes_pod_namespace | Pod命名空间 |
| __meta_kubernetes_pod_ip | Pod IP |
| __meta_kubernetes_pod_label_* | Pod标签 |
| __meta_kubernetes_pod_annotation_* | Pod注解 |
| __meta_kubernetes_pod_container_name | 容器名称 |
| __meta_kubernetes_pod_container_port_number | 容器端口 |

### Endpoints角色元数据

| 标签 | 说明 |
|------|------|
| __meta_kubernetes_namespace | 命名空间 |
| __meta_kubernetes_service_name | Service名称 |
| __meta_kubernetes_service_label_* | Service标签 |
| __meta_kubernetes_service_annotation_* | Service注解 |
| __meta_kubernetes_endpoint_port_name | 端口名称 |

## Relabel配置详解

relabel_configs用于过滤和转换标签：

### 常用action

| action | 说明 |
|--------|------|
| keep | 保留匹配的目标 |
| drop | 删除匹配的目标 |
| replace | 替换标签值 |
| labelmap | 批量重命名标签 |
| labeldrop | 删除标签 |

### 示例：只监控特定命名空间

```yaml
relabel_configs:
- source_labels: [__meta_kubernetes_namespace]
  action: keep
  regex: (default|production)
```

### 示例：添加自定义标签

```yaml
relabel_configs:
- target_label: environment
  replacement: production
```

## 验证服务发现

### 查看发现的目标

访问Prometheus UI的 `/targets` 页面，可以看到所有发现的目标。

### 使用API查询

```bash
curl http://prometheus:9090/api/v1/targets
```

### 查看原始标签

```bash
curl http://prometheus:9090/api/v1/targets/metadata
```

## 常见问题

### Q1: 目标没有被发现

检查：
1. RBAC权限是否正确
2. 注解是否正确添加
3. Pod/Service是否存在

### Q2: 目标显示为down

检查：
1. 目标是否可达
2. 端口是否正确
3. 路径是否正确

### Q3: 标签丢失

检查relabel配置，确保没有错误删除标签。

## 参考资源

- [Prometheus服务发现文档](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#kubernetes_sd_config)
- [Kubernetes服务发现](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#kubernetes_sd_config)

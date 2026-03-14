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
  - Operator
---

# Prometheus Operator自动发现机制

## 什么是Prometheus Operator？

Prometheus Operator是CoreOS开发的Kubernetes Operator，用于管理Prometheus相关资源。它通过CRD（Custom Resource Definition）定义了多种资源类型，让Prometheus的部署和配置更加声明式、自动化。

## 核心概念

### CRD资源类型

| 资源类型 | 说明 |
|----------|------|
| Prometheus | Prometheus实例 |
| Alertmanager | Alertmanager实例 |
| ServiceMonitor | Service级别的监控配置 |
| PodMonitor | Pod级别的监控配置 |
| Probe | 黑盒监控配置 |
| PrometheusRule | 告警规则配置 |

### 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                   Prometheus Operator架构                     │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Prometheus Operator                      │    │
│  │                                                      │    │
│  │  监听CRD资源变化                                      │    │
│  │  生成Prometheus配置                                   │    │
│  │  管理Prometheus生命周期                                │    │
│  └─────────────────────────────────────────────────────┘    │
│                            │                                 │
│         ┌──────────────────┼──────────────────┐             │
│         ↓                  ↓                  ↓             │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐       │
│  │ServiceMonitor│   │ PodMonitor  │   │    Probe    │       │
│  └─────────────┘   └─────────────┘   └─────────────┘       │
│         │                  │                  │              │
│         └──────────────────┼──────────────────┘             │
│                            ↓                                 │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                    Prometheus                         │    │
│  │                                                      │    │
│  │  自动生成scrape_configs                              │    │
│  │  自动发现监控目标                                     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 自动发现的核心组件

### ServiceMonitor

ServiceMonitor是最常用的自动发现方式，它通过Service发现后端Pod。

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: my-app
  namespace: monitoring
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app: my-app
  namespaceSelector:
    matchNames:
    - default
  endpoints:
  - port: metrics
    path: /metrics
    interval: 30s
```

**工作流程**：

```
1. ServiceMonitor定义选择器
       ↓
2. Operator查找匹配的Service
       ↓
3. 通过Service找到对应的Endpoints
       ↓
4. 生成Prometheus的scrape_configs
       ↓
5. Prometheus开始采集
```

### PodMonitor

PodMonitor直接发现Pod，不需要Service。

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: my-app
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: my-app
  namespaceSelector:
    matchNames:
    - default
  podMetricsEndpoints:
  - port: metrics
    path: /metrics
    interval: 30s
```

**适用场景**：
- 不需要Service的应用
- Headless Service
- 每个Pod需要单独监控

### Probe

Probe用于黑盒监控，探测外部目标。

```yaml
apiVersion: monitoring.coreos.com/v1
kind: Probe
metadata:
  name: blackbox
  namespace: monitoring
spec:
  module: http_2xx
  prober:
    url: blackbox-exporter:9115
  targets:
    staticConfig:
      static:
      - https://example.com
      - https://api.example.com/health
```

## 完整部署示例

### 1. 安装Prometheus Operator

```bash
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml
```

或使用Helm：

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus-operator prometheus-community/kube-prometheus-stack
```

### 2. 创建Prometheus实例

```yaml
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: prometheus
  namespace: monitoring
spec:
  serviceAccountName: prometheus
  serviceMonitorSelector:
    matchLabels:
      release: prometheus
  podMonitorSelector:
    matchLabels:
      release: prometheus
  ruleSelector:
    matchLabels:
      release: prometheus
  alerting:
    alertmanagers:
    - name: alertmanager
      namespace: monitoring
      port: web
  resources:
    requests:
      memory: 400Mi
    limits:
      memory: 800Mi
  retention: 15d
  storage:
    volumeClaimTemplate:
      spec:
        storageClassName: standard
        resources:
          requests:
            storage: 50Gi
```

### 3. 创建Alertmanager实例

```yaml
apiVersion: monitoring.coreos.com/v1
kind: Alertmanager
metadata:
  name: alertmanager
  namespace: monitoring
spec:
  replicas: 3
  serviceAccountName: alertmanager
  alertmanagerConfiguration:
    name: alertmanager-config
  storage:
    volumeClaimTemplate:
      spec:
        storageClassName: standard
        resources:
          requests:
            storage: 10Gi
```

### 4. 创建ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: my-app
  namespace: monitoring
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app: my-app
  namespaceSelector:
    matchNames:
    - default
  endpoints:
  - port: metrics
    path: /metrics
    interval: 30s
    scrapeTimeout: 10s
    relabelings:
    - sourceLabels: [__meta_kubernetes_pod_name]
      targetLabel: pod
    - sourceLabels: [__meta_kubernetes_namespace]
      targetLabel: namespace
```

### 5. 创建告警规则

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: my-app-rules
  namespace: monitoring
  labels:
    release: prometheus
spec:
  groups:
  - name: my-app
    rules:
    - alert: MyAppHighErrorRate
      expr: rate(my_app_errors_total[5m]) > 0.1
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "应用错误率过高"
        description: "应用错误率超过10%"
```

## ServiceMonitor详解

### selector

选择要监控的Service：

```yaml
selector:
  matchLabels:
    app: my-app
  matchExpressions:
  - key: environment
    operator: In
    values:
    - production
    - staging
```

### namespaceSelector

指定要监控的命名空间：

```yaml
namespaceSelector:
  matchNames:
  - default
  - production
```

或监控所有命名空间：

```yaml
namespaceSelector:
  any: true
```

### endpoints

定义采集端点：

```yaml
endpoints:
- port: metrics
  path: /metrics
  interval: 30s
  scrapeTimeout: 10s
  scheme: HTTP
  tlsConfig:
    insecureSkipVerify: true
  basicAuth:
    username:
      name: auth-secret
      key: username
    password:
      name: auth-secret
      key: password
  relabelings:
  - sourceLabels: [__meta_kubernetes_pod_name]
    targetLabel: pod
  metricRelabelings:
  - sourceLabels: [__name__]
    regex: 'go_.*'
    action: drop
```

## 与手动部署的对比

| 特性 | 手动部署 | Operator部署 |
|------|----------|--------------|
| 配置方式 | ConfigMap | CRD |
| 自动发现 | 手动配置 | 自动配置 |
| 服务发现 | 手动配置 | ServiceMonitor |
| 配置更新 | 重启Prometheus | 自动热更新 |
| 高可用 | 手动配置 | 内置支持 |
| 存储 | 手动配置 | 自动PVC |
| RBAC | 手动配置 | 自动创建 |

## 最佳实践

### 1. 使用标签选择器

```yaml
metadata:
  labels:
    release: prometheus
    app: my-app
```

### 2. 合理设置采集间隔

```yaml
endpoints:
- port: metrics
  interval: 30s
  scrapeTimeout: 10s
```

### 3. 使用命名空间选择器

```yaml
namespaceSelector:
  matchNames:
  - default
  - production
```

### 4. 配置资源限制

```yaml
resources:
  requests:
    cpu: 100m
    memory: 400Mi
  limits:
    cpu: 500m
    memory: 800Mi
```

## 参考资源

- [Prometheus Operator官方文档](https://prometheus-operator.dev/)
- [ServiceMonitor配置](https://prometheus-operator.dev/docs/operator/api/#servicemonitor)
- [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)

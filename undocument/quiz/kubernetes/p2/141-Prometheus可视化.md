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
  - 可视化
  - Grafana
---

# Prometheus 可视化和查询界面

## 引言

Prometheus 自带了基本的 Web UI 界面，可以进行查询和简单的可视化。但对于复杂的监控仪表板需求，通常需要结合 Grafana 使用。本文介绍 Prometheus 自带的查询界面和 Grafana 可视化方案。

## Prometheus Web UI

### 访问界面

```bash
kubectl port-forward svc/prometheus 9090:9090 -n monitoring

http://localhost:9090
```

### 界面功能

```
┌─────────────────────────────────────────────────────────────┐
│              Prometheus Web UI 功能                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Graph 页面：                                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • PromQL 查询输入                                  │   │
│  │  • 图表展示                                         │   │
│  │  • 时间范围选择                                     │   │
│  │  • 查询历史                                         │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Alerts 页面：                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • 查看告警规则                                     │   │
│  │  • 查看告警状态                                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Status 页面：                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • 运行时信息                                       │   │
│  │  • 配置信息                                         │   │
│  │  • 目标状态                                         │   │
│  │  • 服务发现                                         │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 查询示例

```promql
http_requests_total

rate(http_requests_total[5m])

sum by (method) (rate(http_requests_total[5m]))
```

## Grafana 集成

### Grafana 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      securityContext:
        fsGroup: 472
        runAsUser: 472
      containers:
      - name: grafana
        image: grafana/grafana:10.0.0
        ports:
        - containerPort: 3000
        env:
        - name: GF_SECURITY_ADMIN_USER
          value: admin
        - name: GF_SECURITY_ADMIN_PASSWORD
          valueFrom:
            secretKeyRef:
              name: grafana-secret
              key: admin-password
        volumeMounts:
        - name: storage
          mountPath: /var/lib/grafana
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: grafana-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
  namespace: monitoring
spec:
  type: NodePort
  selector:
    app: grafana
  ports:
  - port: 3000
    targetPort: 3000
    nodePort: 30300
```

### 配置 Prometheus 数据源

```yaml
apiVersion: 1
datasources:
- name: Prometheus
  type: prometheus
  access: proxy
  url: http://prometheus:9090
  isDefault: true
  editable: false
```

## Dashboard 创建

### Dashboard JSON 示例

```json
{
  "dashboard": {
    "title": "Kubernetes Cluster Monitoring",
    "uid": "k8s-cluster",
    "panels": [
      {
        "title": "Cluster CPU Usage",
        "type": "graph",
        "gridPos": {
          "x": 0,
          "y": 0,
          "w": 12,
          "h": 8
        },
        "targets": [
          {
            "expr": "sum(rate(container_cpu_usage_seconds_total{container!=\"\"}[5m])) by (node)",
            "legendFormat": "{{node}}"
          }
        ]
      },
      {
        "title": "Cluster Memory Usage",
        "type": "graph",
        "gridPos": {
          "x": 12,
          "y": 0,
          "w": 12,
          "h": 8
        },
        "targets": [
          {
            "expr": "sum(container_memory_working_set_bytes{container!=\"\"}) by (node)",
            "legendFormat": "{{node}}"
          }
        ]
      }
    ]
  }
}
```

### 常用面板类型

| 面板类型 | 用途 |
|---------|------|
| Graph | 时间序列图表 |
| Stat | 单值显示 |
| Gauge | 仪表盘 |
| Table | 表格 |
| Heatmap | 热力图 |
| Pie Chart | 饼图 |

## Dashboard 配置

### ConfigMap 方式导入

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-dashboards
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  kubernetes-cluster.json: |
    {
      "dashboard": {
        "title": "Kubernetes Cluster",
        "panels": [...]
      }
    }
```

### Sidecar 自动导入

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
spec:
  template:
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:10.0.0
      - name: grafana-sc-dashboard
        image: quay.io/kiwigrid/k8s-sidecar:1.24.0
        env:
        - name: METHOD
          value: WATCH
        - name: LABEL
          value: grafana_dashboard
        - name: FOLDER
          value: /tmp/dashboards
        volumeMounts:
        - name: dashboards
          mountPath: /tmp/dashboards
```

## 常用 Dashboard

### Node Exporter Dashboard

```json
{
  "title": "Node Exporter",
  "panels": [
    {
      "title": "CPU Usage",
      "targets": [
        {
          "expr": "100 - (avg by (instance) (irate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)"
        }
      ]
    },
    {
      "title": "Memory Usage",
      "targets": [
        {
          "expr": "(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100"
        }
      ]
    }
  ]
}
```

### Kubernetes Pod Dashboard

```json
{
  "title": "Kubernetes Pods",
  "panels": [
    {
      "title": "Pod CPU",
      "targets": [
        {
          "expr": "sum(rate(container_cpu_usage_seconds_total{container!=\"\"}[5m])) by (pod)"
        }
      ]
    },
    {
      "title": "Pod Memory",
      "targets": [
        {
          "expr": "sum(container_memory_working_set_bytes{container!=\"\"}) by (pod)"
        }
      ]
    }
  ]
}
```

## 最佳实践

### 1. 使用变量

```json
{
  "templating": {
    "list": [
      {
        "name": "namespace",
        "type": "query",
        "query": "label_values(kube_pod_info, namespace)"
      }
    ]
  }
}
```

### 2. 设置告警

```json
{
  "alert": {
    "conditions": [
      {
        "evaluator": {
          "type": "gt",
          "params": [80]
        }
      }
    ],
    "executionErrorState": "alerting",
    "frequency": "60s"
  }
}
```

### 3. 使用模板

```json
{
  "annotations": {
    "list": [
      {
        "datasource": "Prometheus",
        "enable": true,
        "expr": "ALERTS{alertstate=\"firing\"}"
      }
    ]
  }
}
```

## 面试回答

**问题**: Prometheus 的可视化和查询界面是什么？

**回答**: Prometheus 可视化和查询界面包括 Prometheus 自带的 Web UI 和 Grafana：

**Prometheus Web UI**：Prometheus 自带基本的 Web 界面，访问地址 http://prometheus:9090。主要功能包括：**Graph 页面**输入 PromQL 查询，以图表形式展示结果，支持时间范围选择；**Alerts 页面**查看告警规则和告警状态；**Status 页面**查看运行时信息、配置、目标状态、服务发现等。

**Grafana**：专业的可视化平台，是 Prometheus 的最佳搭档。支持丰富的图表类型（Graph、Stat、Gauge、Table、Heatmap 等）；支持 Dashboard 模板和变量；支持告警配置；支持多种数据源。

**Grafana 集成**：部署 Grafana，配置 Prometheus 数据源，创建 Dashboard。可以通过 ConfigMap 或 Sidecar 方式自动导入 Dashboard。使用变量实现动态查询，如按命名空间、Pod 过滤。

**常用 Dashboard**：Node Exporter Dashboard 展示节点 CPU、内存、磁盘、网络指标；Kubernetes Pod Dashboard 展示 Pod 资源使用；Kubernetes Cluster Dashboard 展示集群整体状态。

**最佳实践**：使用变量实现动态过滤；配置告警规则；使用模板复用 Dashboard；导入社区 Dashboard（如 Grafana.com 上的官方 Dashboard）。

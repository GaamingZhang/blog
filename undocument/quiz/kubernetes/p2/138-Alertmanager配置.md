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
  - Alertmanager
  - 告警
---

# Alertmanager 告警配置

## 引言

Alertmanager 是 Prometheus 生态中的告警管理组件，负责接收、去重、分组、路由和发送告警通知。合理配置 Alertmanager，可以确保告警及时、准确地送达相关人员，提高运维效率。

## Alertmanager 概述

### 告警流程

```
┌─────────────────────────────────────────────────────────────┐
│                  告警流程                                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Prometheus                           │  │
│  │  • 评估告警规则                                      │  │
│  │  • 发送告警到 Alertmanager                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Alertmanager                         │  │
│  │  • 接收告警                                          │  │
│  │  • 去重                                              │  │
│  │  • 分组                                              │  │
│  │  • 路由                                              │  │
│  │  • 发送通知                                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Receivers                            │  │
│  │  • Email                                             │  │
│  │  • Slack                                             │  │
│  │  • 钉钉                                              │  │
│  │  • 企业微信                                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 核心功能

| 功能 | 说明 |
|-----|------|
| 去重 | 合并相同告警 |
| 分组 | 将相关告警合并 |
| 路由 | 根据标签分发告警 |
| 静默 | 临时抑制告警 |
| 抑制 | 根据条件抑制告警 |

## Prometheus 告警规则

### 告警规则配置

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-rules
  namespace: monitoring
data:
  alert-rules.yml: |
    groups:
    - name: node-alerts
      rules:
      - alert: NodeHighCPU
        expr: 100 - (avg by(instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "节点 CPU 使用率过高"
          description: "节点 {{ $labels.instance }} CPU 使用率超过 80%，当前值：{{ $value }}%"

      - alert: NodeHighMemory
        expr: (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > 85
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "节点内存使用率过高"
          description: "节点 {{ $labels.instance }} 内存使用率超过 85%"

    - name: pod-alerts
      rules:
      - alert: PodCrashLooping
        expr: rate(kube_pod_container_status_restarts_total[15m]) * 60 * 15 > 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Pod 频繁重启"
          description: "Pod {{ $labels.namespace }}/{{ $labels.pod }} 在过去 15 分钟内重启"
```

### Prometheus 配置

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s

    alerting:
      alertmanagers:
      - static_configs:
        - targets:
          - alertmanager:9093

    rule_files:
    - /etc/prometheus/rules/*.yml
```

## Alertmanager 配置

### 基本配置

```yaml
global:
  resolve_timeout: 5m
  smtp_smarthost: 'smtp.example.com:587'
  smtp_from: 'alertmanager@example.com'
  smtp_auth_username: 'alertmanager@example.com'
  smtp_auth_password: 'password'

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

### 路由配置

```yaml
route:
  group_by: ['alertname', 'namespace']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 1h
  receiver: 'default-receiver'
  routes:
  - match:
      severity: critical
    receiver: 'critical-receiver'
  - match:
      severity: warning
    receiver: 'warning-receiver'
  - match_re:
      namespace: production|staging
    receiver: 'prod-receiver'
```

### 接收器配置

```yaml
receivers:
- name: 'email-receiver'
  email_configs:
  - to: 'admin@example.com'
    send_resolved: true

- name: 'slack-receiver'
  slack_configs:
  - api_url: 'https://hooks.slack.com/services/xxx'
    channel: '#alerts'
    send_resolved: true

- name: 'webhook-receiver'
  webhook_configs:
  - url: 'http://webhook-server:8080/alerts'
    send_resolved: true

- name: 'dingtalk-receiver'
  webhook_configs:
  - url: 'https://oapi.dingtalk.com/robot/send?access_token=xxx'
```

## 告警分组

### 分组配置

```yaml
route:
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 1h
```

### 分组说明

```
┌─────────────────────────────────────────────────────────────┐
│                  告警分组说明                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  group_by：分组依据                                          │
│  • 相同标签值的告警分为一组                                  │
│  • 同一组告警合并发送                                        │
│                                                              │
│  group_wait：初始等待时间                                    │
│  • 收到第一个告警后等待时间                                  │
│  • 等待更多同类告警到达                                      │
│                                                              │
│  group_interval：发送间隔                                    │
│  • 同一组告警的发送间隔                                      │
│  • 控制告警频率                                              │
│                                                              │
│  repeat_interval：重复发送间隔                               │
│  • 同一告警重复发送的间隔                                    │
│  • 避免告警被忽略                                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 告警抑制

### 抑制规则

```yaml
inhibit_rules:
- source_match:
    severity: 'critical'
  target_match:
    severity: 'warning'
  equal: ['alertname', 'instance']
```

### 抑制场景

```yaml
inhibit_rules:
- source_match:
    alertname: 'NodeDown'
  target_match_re:
    alertname: '.*'
  equal: ['instance']
```

## 告警静默

### 创建静默

```bash
amtool silence add alertname=NodeHighCPU instance=node-1 --duration=1h

amtool silence add severity=warning --duration=30m --comment="维护窗口"
```

### 查看静默

```bash
amtool silence query

amtool silence query --all
```

### 删除静默

```bash
amtool silence expire <silence-id>
```

## Alertmanager 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alertmanager
  namespace: monitoring
spec:
  replicas: 1
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
        ports:
        - containerPort: 9093
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
---
apiVersion: v1
kind: Service
metadata:
  name: alertmanager
  namespace: monitoring
spec:
  selector:
    app: alertmanager
  ports:
  - port: 9093
    targetPort: 9093
```

## 最佳实践

### 1. 合理设置告警级别

```yaml
labels:
  severity: critical   # 严重，需要立即处理
  severity: warning    # 警告，需要关注
  severity: info       # 信息，仅供参考
```

### 2. 设置合理的等待时间

```yaml
for: 5m  # 告警持续 5 分钟才触发
```

### 3. 分组发送告警

```yaml
group_by: ['alertname', 'namespace']
group_wait: 30s
```

### 4. 使用模板

```yaml
receivers:
- name: 'email-receiver'
  email_configs:
  - to: 'admin@example.com'
    html: '{{ template "email.html" . }}'
```

## 面试回答

**问题**: 如何设置警报规则并配置 Alertmanager？

**回答**: Prometheus 告警系统分为告警规则和 Alertmanager 两部分：

**告警规则**：在 Prometheus 中配置，定义告警触发条件。规则包含 alert（告警名称）、expr（PromQL 表达式）、for（持续时间）、labels（标签）、annotations（注解）。告警规则在 Prometheus 中评估，满足条件时发送到 Alertmanager。

**Alertmanager 配置**：**global** 全局配置，包括 SMTP、超时时间等；**route** 路由配置，定义告警如何分组、分发；**receivers** 接收器配置，定义告警发送目标（Email、Slack、钉钉等）；**inhibit_rules** 抑制规则，根据条件抑制告警。

**告警分组**：group_by 指定分组标签，相同标签值的告警分为一组；group_wait 初始等待时间，等待更多同类告警；group_interval 同组告警发送间隔；repeat_interval 同一告警重复发送间隔。

**告警抑制**：当 source_match 的告警触发时，抑制 target_match 的告警。用于避免发送不必要的告警，如节点故障时抑制该节点上的所有告警。

**告警静默**：临时抑制特定告警，适用于维护窗口等场景。使用 amtool 或 Web UI 创建静默规则。

**最佳实践**：合理设置告警级别（critical/warning/info）；设置合理的 for 持续时间避免误报；分组发送告警减少噪音；配置抑制规则避免重复告警；使用模板美化告警内容。

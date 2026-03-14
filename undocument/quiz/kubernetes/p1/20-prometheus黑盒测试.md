---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - 监控
tag:
  - Prometheus
  - 监控
  - 黑盒测试
---

# Prometheus黑盒测试监控详解

## 概述

黑盒测试（Blackbox Exporter）是Prometheus生态中用于进行黑盒探测的组件，通过HTTP、HTTPS、DNS、TCP、ICMP等协议对目标进行探测，从外部视角监控服务的可用性和响应时间。

## 黑盒测试架构

```
+------------------+     +------------------+     +------------------+
|   Prometheus     |     | Blackbox Exporter|     |    目标服务      |
+------------------+     +------------------+     +------------------+
         |                        |                        |
         | 1. 抓取请求            |                        |
         | (带目标参数)           |                        |
         |----------------------->|                        |
         |                        | 2. 发起探测请求        |
         |                        |----------------------->|
         |                        |                        |
         |                        | 3. 返回探测结果        |
         |                        |<-----------------------|
         | 4. 返回指标数据        |                        |
         |<-----------------------|                        |
         |                        |                        |
         v                        v                        v
+------------------+     +------------------+     +------------------+
|   存储指标       |     |   探测日志       |     |   服务日志       |
+------------------+     +------------------+     +------------------+
```

## Blackbox Exporter部署

### Docker部署

```bash
docker run -d \
  --name blackbox_exporter \
  -p 9115:9115 \
  -v /path/to/blackbox.yml:/etc/blackbox_exporter/config.yml \
  prom/blackbox-exporter:latest
```

### Kubernetes部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: blackbox-exporter
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: blackbox-exporter
  template:
    metadata:
      labels:
        app: blackbox-exporter
    spec:
      containers:
      - name: blackbox-exporter
        image: prom/blackbox-exporter:latest
        ports:
        - containerPort: 9115
        volumeMounts:
        - name: config
          mountPath: /etc/blackbox_exporter
      volumes:
      - name: config
        configMap:
          name: blackbox-config

---
apiVersion: v1
kind: Service
metadata:
  name: blackbox-exporter
  namespace: monitoring
spec:
  ports:
  - port: 9115
    targetPort: 9115
  selector:
    app: blackbox-exporter
```

## Blackbox配置文件

```yaml
# blackbox.yml
modules:
  # HTTP探测配置
  http_2xx:
    prober: http
    timeout: 10s
    http:
      valid_http_versions: ["HTTP/1.1", "HTTP/2.0"]
      valid_status_codes: [200]
      method: GET
      follow_redirects: true
      fail_if_ssl: false
      fail_if_not_ssl: false
      tls_config:
        insecure_skip_verify: false
      preferred_ip_protocol: "ip4"

  # HTTP POST探测
  http_post_2xx:
    prober: http
    timeout: 10s
    http:
      method: POST
      headers:
        Content-Type: application/json
      body: '{"status":"ok"}'
      valid_status_codes: [200, 201]
      preferred_ip_protocol: "ip4"

  # HTTPS探测（不验证证书）
  https_2xx_skip_verify:
    prober: http
    timeout: 10s
    http:
      valid_http_versions: ["HTTP/1.1", "HTTP/2.0"]
      valid_status_codes: [200]
      tls_config:
        insecure_skip_verify: true
      preferred_ip_protocol: "ip4"

  # TCP探测
  tcp_connect:
    prober: tcp
    timeout: 10s

  # ICMP探测
  icmp:
    prober: icmp
    timeout: 5s
    icmp:
      preferred_ip_protocol: "ip4"

  # DNS探测
  dns_udp:
    prober: dns
    timeout: 5s
    dns:
      transport_protocol: "udp"
      preferred_ip_protocol: "ip4"
      query_name: "kubernetes.default.svc.cluster.local"
      query_type: "A"

  # DNS TCP探测
  dns_tcp:
    prober: dns
    timeout: 5s
    dns:
      transport_protocol: "tcp"
      preferred_ip_protocol: "ip4"
```

## Prometheus配置

### 抓取配置

```yaml
# prometheus.yml
scrape_configs:
  # HTTP探测示例
  - job_name: 'blackbox-http'
    metrics_path: /probe
    params:
      module: [http_2xx]
    static_configs:
      - targets:
        - http://www.example.com
        - http://api.example.com/health
        - https://secure.example.com
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: blackbox-exporter:9115

  # TCP端口探测
  - job_name: 'blackbox-tcp'
    metrics_path: /probe
    params:
      module: [tcp_connect]
    static_configs:
      - targets:
        - mysql:3306
        - redis:6379
        - rabbitmq:5672
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: blackbox-exporter:9115

  # ICMP探测
  - job_name: 'blackbox-icmp'
    metrics_path: /probe
    params:
      module: [icmp]
    static_configs:
      - targets:
        - 192.168.1.1
        - 192.168.1.2
        - 8.8.8.8
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: blackbox-exporter:9115

  # DNS探测
  - job_name: 'blackbox-dns'
    metrics_path: /probe
    params:
      module: [dns_udp]
    static_configs:
      - targets:
        - 8.8.8.8
        - 8.8.4.4
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: blackbox-exporter:9115
```

### Kubernetes服务探测

```yaml
# 自动发现Kubernetes服务进行HTTP探测
scrape_configs:
  - job_name: 'kubernetes-services'
    metrics_path: /probe
    params:
      module: [http_2xx]
    kubernetes_sd_configs:
      - role: service
    relabel_configs:
      # 只探测有prometheus.io/probe注解的服务
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
        target_label: namespace
      - source_labels: [__meta_kubernetes_service_name]
        target_label: service
```

## 核心指标说明

```
+-----------------------------+-------------------------------------+
|          指标名称           |                说明                 |
+-----------------------------+-------------------------------------+
| probe_success               | 探测是否成功 (1=成功, 0=失败)       |
| probe_duration_seconds      | 探测总耗时（秒）                     |
| probe_http_status_code      | HTTP状态码                          |
| probe_http_content_length   | HTTP响应内容长度                     |
| probe_http_version          | HTTP协议版本                        |
| probe_dns_lookup_time_seconds| DNS解析耗时                        |
| probe_tcp_connection_seconds| TCP连接耗时                         |
| probe_tls_handshake_seconds | TLS握手耗时                         |
| probe_icmp_duration_seconds | ICMP响应耗时                        |
| probe_icmp_reply_packet_loss_percent | ICMP丢包率                   |
+-----------------------------+-------------------------------------+
```

## 告警规则配置

```yaml
# /etc/prometheus/rules/blackbox_alerts.yml
groups:
  - name: blackbox_alerts
    rules:
      - alert: ServiceDown
        expr: probe_success == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "服务探测失败: {{ $labels.instance }}"
          description: "目标服务 {{ $labels.instance }} 探测失败已超过1分钟"

      - alert: SlowResponse
        expr: probe_duration_seconds > 5
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "服务响应缓慢: {{ $labels.instance }}"
          description: "目标服务 {{ $labels.instance }} 响应时间 {{ $value | printf \"%.2f\" }} 秒"

      - alert: HTTPStatusError
        expr: |
          probe_http_status_code >= 400
          and probe_http_status_code < 600
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "HTTP状态码异常: {{ $labels.instance }}"
          description: "目标服务 {{ $labels.instance }} 返回状态码 {{ $value }}"

      - alert: SSLCertificateExpiringSoon
        expr: |
          (probe_ssl_earliest_cert_expiry - time()) / 86400 < 30
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "SSL证书即将过期: {{ $labels.instance }}"
          description: "SSL证书将在 {{ $value | printf \"%.0f\" }} 天后过期"

      - alert: HighPacketLoss
        expr: probe_icmp_reply_packet_loss_percent > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "ICMP丢包率过高: {{ $labels.instance }}"
          description: "目标主机 {{ $labels.instance }} 丢包率 {{ $value }}%"
```

## Grafana Dashboard

### 关键面板配置

```json
{
  "panels": [
    {
      "title": "服务可用性",
      "type": "stat",
      "targets": [
        {
          "expr": "avg(probe_success)",
          "legendFormat": "可用率"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "unit": "percentunit",
          "mappings": [
            {
              "type": "value",
              "options": {
                "0": {"text": "DOWN", "color": "red"},
                "1": {"text": "UP", "color": "green"}
              }
            }
          ]
        }
      }
    },
    {
      "title": "响应时间分布",
      "type": "graph",
      "targets": [
        {
          "expr": "probe_duration_seconds",
          "legendFormat": "{{ instance }}"
        }
      ]
    },
    {
      "title": "HTTP状态码",
      "type": "table",
      "targets": [
        {
          "expr": "probe_http_status_code",
          "format": "table",
          "instant": true
        }
      ]
    }
  ]
}
```

## 高级配置示例

### 带认证的HTTP探测

```yaml
modules:
  http_basic_auth:
    prober: http
    timeout: 10s
    http:
      method: GET
      basic_auth:
        username: admin
        password: secret
      valid_status_codes: [200]
```

### 带自定义请求头的探测

```yaml
modules:
  http_with_headers:
    prober: http
    timeout: 10s
    http:
      method: GET
      headers:
        X-API-Key: "your-api-key"
        User-Agent: "Prometheus-Blackbox-Exporter"
      valid_status_codes: [200]
```

### 响应内容验证

```yaml
modules:
  http_body_check:
    prober: http
    timeout: 10s
    http:
      method: GET
      valid_status_codes: [200]
      fail_if_body_not_matches_regexp:
        - "status.*ok"
```

## 最佳实践

### 1. 探测频率设置

```yaml
# 根据服务重要性设置不同的抓取间隔
scrape_configs:
  - job_name: 'critical-services'
    scrape_interval: 15s
    # ...

  - job_name: 'normal-services'
    scrape_interval: 30s
    # ...
```

### 2. 超时设置

```yaml
# 超时时间应小于抓取间隔
modules:
  http_2xx:
    prober: http
    timeout: 10s  # 小于scrape_interval
```

### 3. 多维度探测

```yaml
# 对同一服务进行多种探测
- job_name: 'service-multi-probe'
  metrics_path: /probe
  static_configs:
    - targets:
      - http://api.example.com/health  # HTTP探测
  relabel_configs:
    # ...

- job_name: 'service-tcp-probe'
  metrics_path: /probe
  params:
    module: [tcp_connect]
  static_configs:
    - targets:
      - api.example.com:443  # TCP探测
```

## 参考资源

- [Blackbox Exporter GitHub](https://github.com/prometheus/blackbox_exporter)
- [Prometheus配置文档](https://prometheus.io/docs/prometheus/latest/configuration/configuration/)

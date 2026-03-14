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
  - 推模式
  - 拉模式
---

# Prometheus 推模式和拉模式

## 引言

监控系统的数据采集方式主要分为推模式和拉模式。Prometheus 采用拉模式，但也支持通过 Pushgateway 实现推模式。理解两种模式的特点和适用场景，对于设计监控系统架构至关重要。

## 推模式与拉模式对比

### 模式对比

```
┌─────────────────────────────────────────────────────────────┐
│              推模式 vs 拉模式                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  推模式：                                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  应用 ──────主动推送──────▶ 监控系统                │   │
│  │                                                      │   │
│  │  优点：                                              │   │
│  │  • 应用控制推送时机                                  │   │
│  │  • 支持短生命周期任务                                │   │
│  │  • 穿透防火墙                                        │   │
│  │                                                      │   │
│  │  缺点：                                              │   │
│  │  • 监控系统需要处理大量连接                          │   │
│  │  • 难以感知应用健康状态                              │   │
│  │  • 配置管理复杂                                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  拉模式：                                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  监控系统 ──────主动拉取──────▶ 应用                │   │
│  │                                                      │   │
│  │  优点：                                              │   │
│  │  • 监控系统控制采集频率                              │   │
│  │  • 易于感知应用健康状态                              │   │
│  │  • 配置集中管理                                      │   │
│  │                                                      │   │
│  │  缺点：                                              │   │
│  │  • 需要应用暴露端点                                  │   │
│  │  • 短生命周期任务难以采集                            │   │
│  │  • 需要服务发现                                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Prometheus 拉模式

```
┌─────────────────────────────────────────────────────────────┐
│              Prometheus 拉模式架构                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Prometheus                           │  │
│  │  • 定期抓取目标                                      │  │
│  │  • 控制采集频率                                      │  │
│  │  • 服务发现                                          │  │
│  └──────────────────────────────────────────────────────┘  │
│           │              │              │                   │
│           ▼              ▼              ▼                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   App 1     │  │   App 2     │  │   App 3     │        │
│  │  /metrics   │  │  /metrics   │  │  /metrics   │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
│                                                              │
│  Prometheus 主动拉取各应用的 /metrics 端点                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Prometheus 拉模式

### 配置示例

```yaml
scrape_configs:
- job_name: 'myapp'
  scrape_interval: 15s
  static_configs:
  - targets:
    - 'app-1:8080'
    - 'app-2:8080'
    - 'app-3:8080'
```

### 拉模式优势

| 优势 | 说明 |
|-----|------|
| 统一采集频率 | Prometheus 控制采集频率，避免过载 |
| 健康感知 | 抓取失败可以感知应用异常 |
| 配置集中 | 所有目标配置在 Prometheus 端 |
| 服务发现 | 支持多种服务发现机制 |

## Pushgateway 推模式

### Pushgateway 架构

```
┌─────────────────────────────────────────────────────────────┐
│              Pushgateway 架构                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │  批处理任务 │  │  定时任务   │  │  短生命周期 │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
│         │                │                │                 │
│         └────────────────┼────────────────┘                 │
│                          │                                   │
│                          ▼ 推送指标                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Pushgateway                          │  │
│  │  • 接收推送的指标                                    │  │
│  │  • 缓存指标                                          │  │
│  │  • 暴露 /metrics 端点                                │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼ 拉取指标                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Prometheus                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Pushgateway 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pushgateway
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pushgateway
  template:
    metadata:
      labels:
        app: pushgateway
    spec:
      containers:
      - name: pushgateway
        image: prom/pushgateway:v1.6.0
        ports:
        - containerPort: 9091
---
apiVersion: v1
kind: Service
metadata:
  name: pushgateway
  namespace: monitoring
spec:
  selector:
    app: pushgateway
  ports:
  - port: 9091
    targetPort: 9091
```

### Prometheus 配置

```yaml
scrape_configs:
- job_name: 'pushgateway'
  honor_labels: true
  static_configs:
  - targets:
    - 'pushgateway:9091'
```

### 推送指标

```bash
echo "batch_job_duration_seconds 42" | curl --data-binary @- http://pushgateway:9091/metrics/job/batch_job

curl -X POST http://pushgateway:9091/metrics/job/batch_job/instance/$(hostname)
```

### Python 推送示例

```python
from prometheus_client import CollectorRegistry, Gauge, push_to_gateway

registry = CollectorRegistry()

duration = Gauge('batch_job_duration_seconds', 'Duration of batch job', registry=registry)
duration.set(42)

push_to_gateway('pushgateway:9091', job='batch_job', registry=registry)
```

## 推模式适用场景

### 适用场景

```
┌─────────────────────────────────────────────────────────────┐
│              Pushgateway 适用场景                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 批处理任务                                               │
│     • 任务执行时间短                                         │
│     • 任务完成后无法被抓取                                   │
│                                                              │
│  2. 定时任务                                                 │
│     • Cron Job                                              │
│     • 间歇性运行的任务                                       │
│                                                              │
│  3. 短生命周期服务                                           │
│     • 函数计算                                              │
│     • 临时容器                                              │
│                                                              │
│  4. 防火墙内服务                                             │
│     • 无法被 Prometheus 直接访问                             │
│     • 需要主动推送指标                                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 不适用场景

```
┌─────────────────────────────────────────────────────────────┐
│              Pushgateway 不适用场景                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 长期运行的服务                                           │
│     • 应该使用拉模式                                         │
│     • 让 Prometheus 直接抓取                                 │
│                                                              │
│  2. 需要健康检查的服务                                       │
│     • 推模式无法感知服务状态                                 │
│     • 拉模式可以检测服务是否存活                             │
│                                                              │
│  3. 高频率指标                                               │
│     • 推送频率过高会增加网络开销                             │
│     • 应该让 Prometheus 控制采集频率                         │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 最佳实践

### 1. 优先使用拉模式

```yaml
scrape_configs:
- job_name: 'myapp'
  kubernetes_sd_configs:
  - role: pod
```

### 2. 仅在必要时使用 Pushgateway

```python
if is_batch_job:
    push_to_gateway('pushgateway:9091', job='batch_job', registry=registry)
```

### 3. 设置合理的过期时间

```yaml
scrape_configs:
- job_name: 'pushgateway'
  honor_labels: true
  static_configs:
  - targets: ['pushgateway:9091']
```

### 4. 清理过期指标

```bash
curl -X DELETE http://pushgateway:9091/metrics/job/batch_job
```

## 面试回答

**问题**: 什么是 Prometheus 的推模式和拉模式？

**回答**: Prometheus 主要采用拉模式，但也支持通过 Pushgateway 实现推模式：

**拉模式**：Prometheus 主动从目标应用拉取指标。应用暴露 /metrics 端点，Prometheus 定期抓取。优点：Prometheus 控制采集频率，避免过载；抓取失败可以感知应用异常；配置集中管理；支持服务发现自动发现目标。缺点：需要应用暴露端点；短生命周期任务难以采集；需要网络可达。

**推模式（Pushgateway）**：应用主动推送指标到 Pushgateway，Prometheus 从 Pushgateway 拉取。Pushgateway 作为中间层缓存指标。优点：支持短生命周期任务（批处理、定时任务）；支持防火墙内的服务；应用控制推送时机。缺点：Pushgateway 成为单点；无法感知应用健康状态；需要管理指标生命周期。

**适用场景**：**拉模式**适用于长期运行的服务、需要健康检查的服务、标准 Kubernetes 应用。**推模式**适用于批处理任务、定时任务（Cron Job）、短生命周期服务、防火墙内无法被访问的服务。

**最佳实践**：优先使用拉模式；仅在必要时使用 Pushgateway；设置合理的过期时间清理过期指标；使用 honor_labels 保留原始标签。

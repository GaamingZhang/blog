---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Monitoring
tag:
  - Datadog
  - Monitoring
  - APM
  - DevOps
---

# Datadog 生产环境实践

当你的微服务架构从单体演进到分布式系统时,传统的监控系统开始力不从心:基础设施监控、应用性能监控(APM)、日志管理、用户行为分析分散在不同工具中,数据孤岛问题严重。Datadog 作为一站式可观测性平台,将指标、链路追踪、日志、用户体验监控整合在统一平台,大幅降低运维复杂度。但 Datadog 并不是"安装 Agent 就能用"——Agent 部署、指标采集、APM 集成、日志管理、告警配置都需要深入理解才能在生产环境稳定运行。

本文将从 Agent 部署、指标采集、APM 集成、日志管理、告警配置五个维度,系统梳理 Datadog 生产环境的实践经验。

## 一、Agent 部署

### Agent 架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Datadog Agent 架构                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Datadog Agent                                        │  │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐       │  │
│  │  │ Collector  │ │ Forwarder  │ │ DogStatsD  │       │  │
│  │  │ (采集器)   │ │ (转发器)   │ │ (统计服务) │       │  │
│  │  └────────────┘ └────────────┘ └────────────┘       │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│         ┌──────────┼──────────┐                           │
│         │          │          │                           │
│         ▼          ▼          ▼                           │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐             │
│  │ System     │ │ Docker     │ │ Kubernetes │             │
│  │ Metrics    │ │ Metrics    │ │ Metrics    │             │
│  └────────────┘ └────────────┘ └────────────┘             │
│         │          │          │                           │
│         ▼          ▼          ▼                           │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐             │
│  │ Logs       │ │ Traces     │ │ Processes  │             │
│  └────────────┘ └────────────┘ └────────────┘             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Agent 组件说明**:

| 组件 | 功能 | 端口 |
|------|------|------|
| Collector | 系统指标采集 | - |
| Forwarder | 数据转发到 Datadog | 443/TCP |
| DogStatsD | 自定义指标接收 | 8125/UDP |
| Trace Agent | APM 数据接收 | 8126/TCP |
| Logs Agent | 日志收集 | - |
| Process Agent | 进程监控 | - |

### Kubernetes 部署

**DaemonSet 部署**:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: datadog-agent
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: datadog-agent
  template:
    metadata:
      labels:
        app: datadog-agent
      name: datadog-agent
    spec:
      serviceAccountName: datadog-agent
      containers:
        - name: agent
          image: gcr.io/datadoghq/agent:latest
          ports:
            - containerPort: 8125
              name: dogstatsdport
              protocol: UDP
            - containerPort: 8126
              name: traceport
              protocol: TCP
          env:
            - name: DD_API_KEY
              valueFrom:
                secretKeyRef:
                  name: datadog-secret
                  key: api-key
            - name: DD_SITE
              value: "datadoghq.com"
            - name: DD_CLUSTER_NAME
              value: "production"
            - name: DD_TAGS
              value: "env:production,region:us-east-1"
            - name: DD_APM_ENABLED
              value: "true"
            - name: DD_LOGS_ENABLED
              value: "true"
            - name: DD_LOGS_CONFIG_CONTAINER_COLLECT_ALL
              value: "true"
            - name: DD_PROCESS_AGENT_ENABLED
              value: "true"
            - name: DD_ORCHESTRATOR_EXPLORER_ENABLED
              value: "true"
          resources:
            requests:
              cpu: 200m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 512Mi
          volumeMounts:
            - name: dockersocket
              mountPath: /var/run/docker.sock
            - name: procdir
              mountPath: /host/proc
              readOnly: true
            - name: cgroups
              mountPath: /host/sys/fs/cgroup
              readOnly: true
            - name: pointerdir
              mountPath: /opt/datadog-agent/run
      volumes:
        - name: dockersocket
          hostPath:
            path: /var/run/docker.sock
        - name: procdir
          hostPath:
            path: /proc
        - name: cgroups
          hostPath:
            path: /sys/fs/cgroup
        - name: pointerdir
          hostPath:
            path: /opt/datadog-agent/run
            type: DirectoryOrCreate
```

**RBAC 配置**:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: datadog-agent
  namespace: monitoring
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: datadog-agent
rules:
  - apiGroups:
      - ""
    resources:
      - services
      - events
      - endpoints
      - pods
      - nodes
      - namespaces
      - componentstatuses
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - configmaps
    resourceNames:
      - datadogtoken
      - datadog-leader-elector
    verbs:
      - get
      - update
  - apiGroups:
      - ""
    resources:
      - configmaps
    verbs:
      - create
  - nonResourceURLs:
      - "/metrics"
      - "/healthz"
      - "/healthz/*"
    verbs:
      - get
  - apiGroups:
      - "quota"
    resources:
      - resourcequotas
    verbs:
      - list
      - watch
  - apiGroups:
      - "apps"
    resources:
      - deployments
      - replicasets
      - statefulsets
      - daemonsets
    verbs:
      - get
      - list
      - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: datadog-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: datadog-agent
subjects:
  - kind: ServiceAccount
    name: datadog-agent
    namespace: monitoring
```

### ECS 部署

```json
{
  "family": "datadog-agent",
  "containerDefinitions": [
    {
      "name": "datadog-agent",
      "image": "gcr.io/datadoghq/agent:latest",
      "essential": true,
      "environment": [
        {
          "name": "DD_API_KEY",
          "value": "your-api-key"
        },
        {
          "name": "DD_SITE",
          "value": "datadoghq.com"
        },
        {
          "name": "DD_APM_ENABLED",
          "value": "true"
        },
        {
          "name": "DD_LOGS_ENABLED",
          "value": "true"
        }
      ],
      "mountPoints": [
        {
          "containerPath": "/var/run/docker.sock",
          "sourceVolume": "docker-socket",
          "readOnly": true
        },
        {
          "containerPath": "/host/proc",
          "sourceVolume": "proc",
          "readOnly": true
        },
        {
          "containerPath": "/host/sys/fs/cgroup",
          "sourceVolume": "cgroup",
          "readOnly": true
        }
      ],
      "portMappings": [
        {
          "containerPort": 8125,
          "protocol": "udp"
        },
        {
          "containerPort": 8126,
          "protocol": "tcp"
        }
      ]
    }
  ],
  "volumes": [
    {
      "name": "docker-socket",
      "host": {
        "sourcePath": "/var/run/docker.sock"
      }
    },
    {
      "name": "proc",
      "host": {
        "sourcePath": "/proc"
      }
    },
    {
      "name": "cgroup",
      "host": {
        "sourcePath": "/sys/fs/cgroup"
      }
    }
  ]
}
```

## 二、指标采集

### 自定义指标

**DogStatsD 客户端**:

```python
from datadog import DogStatsd

# 初始化客户端
statsd = DogStatsd(host='localhost', port=8125)

# Counter
statsd.increment('app.request.count', tags=['endpoint:/api/users', 'method:GET'])

# Gauge
statsd.gauge('app.active.connections', 100, tags=['service:api'])

# Histogram
statsd.histogram('app.request.latency', 0.5, tags=['endpoint:/api/users'])

# Timing
import time
start_time = time.time()
# ... 业务逻辑 ...
statsd.timing('app.request.duration', (time.time() - start_time) * 1000)

# Distribution
statsd.distribution('app.response.size', 1024, tags=['endpoint:/api/users'])
```

**Python 装饰器**:

```python
from datadog import DogStatsd
import time
import functools

statsd = DogStatsd()

def metric_wrapper(metric_name, tags=None):
    def decorator(func):
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            start_time = time.time()
            try:
                result = func(*args, **kwargs)
                statsd.increment(f'{metric_name}.success', tags=tags)
                return result
            except Exception as e:
                statsd.increment(f'{metric_name}.error', tags=tags + ['error:' + type(e).__name__])
                raise
            finally:
                duration = (time.time() - start_time) * 1000
                statsd.timing(f'{metric_name}.duration', duration, tags=tags)
        return wrapper
    return decorator

@metric_wrapper('app.api.request', tags=['endpoint:/api/users'])
def get_users():
    # 业务逻辑
    return users
```

### 集成配置

**MySQL 集成**:

```yaml
# datadog.yaml
init_config:

instances:
  - server: 'mysql:3306'
    user: 'datadog'
    pass: 'password'
    tags:
      - 'service:production-db'
    options:
      replication: true
      galera_cluster: true
      extra_innodb_metrics: true
      extra_performance_metrics: true
      schema_size_metrics: true
```

**PostgreSQL 集成**:

```yaml
init_config:

instances:
  - host: 'postgres'
    port: 5432
    username: 'datadog'
    password: 'password'
    dbname: 'production'
    tags:
      - 'service:production-db'
    relations:
      - relation_name: 'orders'
        schemas:
          - 'public'
```

**Redis 集成**:

```yaml
init_config:

instances:
  - host: 'redis'
    port: 6379
    password: 'password'
    tags:
      - 'service:cache'
    keys:
      - 'session:*'
      - 'cache:*'
```

### Kubernetes 集成

**自动发现注解**:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  annotations:
    ad.datadoghq.com/app.check_names: '["prometheus"]'
    ad.datadoghq.com/app.init_configs: '[{}]'
    ad.datadoghq.com/app.instances: |
      [{
        "prometheus_url": "http://%%host%%:8000/metrics",
        "namespace": "my_app",
        "metrics": ["request_count", "request_latency"]
      }]
spec:
  template:
    metadata:
      annotations:
        ad.datadoghq.com/app.check_names: '["prometheus"]'
        ad.datadoghq.com/app.init_configs: '[{}]'
        ad.datadoghq.com/app.instances: |
          [{
            "prometheus_url": "http://%%host%%:8000/metrics",
            "namespace": "my_app"
          }]
    spec:
      containers:
        - name: app
          image: my-app:latest
          ports:
            - containerPort: 8000
```

## 三、APM 集成

### 应用性能监控

**Python APM**:

```python
from ddtrace import tracer, patch_all
from ddtrace.contrib.flask import FlaskMiddleware

# 自动埋点
patch_all()

# Flask 应用
from flask import Flask
app = Flask(__name__)

# 配置 Datadog APM
FlaskMiddleware(app, service='my-flask-app', distributed_tracing=True)

# 手动埋点
@app.route('/api/users')
def get_users():
    with tracer.trace('get_users', service='my-flask-app', resource='/api/users'):
        # 业务逻辑
        users = query_database()
        return users

# 自定义 Span
def process_order(order_id):
    with tracer.trace('process_order', service='order-service') as span:
        span.set_tag('order_id', order_id)
        # 业务逻辑
        result = do_something()
        span.set_tag('result', result)
        return result

if __name__ == '__main__':
    app.run()
```

**Node.js APM**:

```javascript
const ddTrace = require('dd-trace');

// 初始化 tracer
ddTrace.init({
  service: 'my-node-app',
  env: 'production',
  version: '1.0.0',
  logInjection: true,
  runtimeMetrics: true
});

const express = require('express');
const app = express();

// 自动埋点
app.use(ddTrace.express.middleware());

// 手动埋点
app.get('/api/users', (req, res) => {
  const span = ddTrace.scope().active();
  span.setTag('user.id', req.user.id);
  
  // 业务逻辑
  const users = getUsers();
  
  res.json(users);
});

// 自定义 Span
async function processOrder(orderId) {
  const span = ddTrace.startSpan('process_order', {
    tags: {
      'order.id': orderId
    }
  });
  
  try {
    const result = await doSomething();
    span.setTag('result', result);
    return result;
  } catch (error) {
    span.setTag('error', error.message);
    throw error;
  } finally {
    span.finish();
  }
}

app.listen(3000);
```

**Java APM**:

```java
import datadog.trace.api.Trace;
import datadog.trace.api.DDTags;
import io.opentracing.Span;
import io.opentracing.util.GlobalTracer;

public class OrderService {
    
    @Trace(operationName = "processOrder", resourceName = "OrderService.processOrder")
    public Order processOrder(String orderId) {
        Span span = GlobalTracer.get().activeSpan();
        span.setTag(DDTags.SERVICE_NAME, "order-service");
        span.setTag("order.id", orderId);
        
        // 业务逻辑
        Order order = doSomething(orderId);
        
        span.setTag("order.status", order.getStatus());
        return order;
    }
}
```

### 分布式追踪

**跨服务追踪**:

```python
from ddtrace import tracer
import requests

def call_external_service(url, headers):
    # 注入追踪头
    headers = headers or {}
    tracer.inject(tracer.active_span, 'http_headers', headers)
    
    # 调用外部服务
    response = requests.get(url, headers=headers)
    
    return response

# 服务 B 接收追踪
from ddtrace.propagation.http import HTTPPropagator

def handle_request(request):
    # 提取追踪上下文
    context = HTTPPropagator.extract(request.headers)
    
    # 继续追踪
    with tracer.start_span('handle_request', child_of=context) as span:
        # 业务逻辑
        result = process_request(request)
        return result
```

### APM 性能优化

**采样配置**:

```python
from ddtrace import tracer

# 配置采样率
tracer.configure(
    sampling_rules=[
        {'sample_rate': 0.1},  # 全局采样率 10%
        {'service': 'critical-service', 'sample_rate': 1.0},  # 关键服务 100%
    ]
)
```

**过滤敏感信息**:

```python
from ddtrace import tracer

# 过滤敏感标签
tracer.set_tags({
    'env': 'production',
    'version': '1.0.0'
})

# 自定义 Span 过滤
def filter_span(span):
    # 移除敏感标签
    if 'password' in span.get_tags():
        span.set_tag('password', '***')
    return span

tracer.configure(span_filter=filter_span)
```

## 四、日志管理

### 日志采集配置

**Kubernetes 日志采集**:

```yaml
# datadog.yaml
logs_enabled: true
logs_config:
  container_collect_all: true
  logs_dd_url: "agent-intake.logs.datadoghq.com:10516"
```

**自定义日志采集**:

```yaml
# /etc/datadog-agent/conf.d/custom_log.d/conf.yaml
logs:
  - type: file
    path: /var/log/myapp/*.log
    service: myapp
    source: custom
    sourcecategory: sourcecode
    tags:
      - env:production
      - service:myapp
  
  - type: tcp
    port: 10518
    service: myapp
    source: custom
```

**应用日志配置**:

```python
import logging
from datadog import DogStatsd

# 配置日志格式
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

# 添加 Datadog 追踪 ID
def log_with_trace(message, level='INFO'):
    from ddtrace import tracer
    span = tracer.active_span()
    if span:
        trace_id = span.trace_id
        span_id = span.span_id
        logger.log(
            getattr(logging, level),
            f"[dd.trace_id={trace_id} dd.span_id={span_id}] {message}"
        )
    else:
        logger.log(getattr(logging, level), message)

# 使用示例
log_with_trace('Processing order', 'INFO')
```

### 日志查询与分析

**日志查询语法**:

```
# 查询特定服务的错误日志
service:myapp status:error

# 查询特定时间范围的日志
service:myapp @timestamp:2026-03-11T10:00:00Z..2026-03-11T11:00:00Z

# 查询包含特定关键字的日志
service:myapp "error" OR "exception"

# 查询特定标签的日志
service:myapp env:production region:us-east-1

# 组合查询
service:myapp status:error env:production "timeout"
```

**日志聚合**:

```python
# 发送日志到 Datadog
import requests

def send_log_to_datadog(message, service='myapp', level='INFO', tags=None):
    url = 'https://http-intake.logs.datadoghq.com/v1/input'
    headers = {
        'Content-Type': 'application/json',
        'DD-API-KEY': 'your-api-key'
    }
    
    payload = {
        'message': message,
        'ddsource': 'python',
        'ddtags': ','.join(tags or []),
        'hostname': 'my-host',
        'service': service,
        'status': level.lower()
    }
    
    requests.post(url, json=payload, headers=headers)
```

## 五、告警配置

### 监控器配置

**指标监控器**:

```python
from datadog import initialize, api

options = {
    'api_key': 'your-api-key',
    'app_key': 'your-app-key'
}

initialize(**options)

# 创建监控器
monitor = api.Monitor.create(
    type='metric alert',
    query='avg(last_5m):sum:system.cpu.user{host:my-host} > 80',
    name='High CPU usage',
    message='CPU usage is above 80% @team@example.com',
    tags=['env:production', 'service:api'],
    options={
        'thresholds': {
            'critical': 80,
            'warning': 70
        },
        'notify_no_data': True,
        'no_data_timeframe': 10,
        'evaluation_delay': 60,
        'include_tags': True,
        'require_full_window': True
    }
)
```

**日志监控器**:

```python
monitor = api.Monitor.create(
    type='log alert',
    query='logs("service:myapp status:error").index("main").rollup("count").last("5m") > 10',
    name='High error rate',
    message='Error rate is above 10 errors in 5 minutes @team@example.com',
    tags=['env:production', 'service:api'],
    options={
        'thresholds': {
            'critical': 10,
            'warning': 5
        },
        'enable_logs_sample': True
    }
)
```

**APM 监控器**:

```python
monitor = api.Monitor.create(
    type='metric alert',
    query='avg(last_5m):trace.http.request{service:myapp,env:production} > 1000',
    name='High latency',
    message='P95 latency is above 1000ms @team@example.com',
    tags=['env:production', 'service:api'],
    options={
        'thresholds': {
            'critical': 1000,
            'warning': 800
        }
    }
)
```

### 告警路由

**通知渠道**:

```python
# Slack 集成
monitor = api.Monitor.create(
    type='metric alert',
    query='avg(last_5m):sum:system.cpu.user{host:my-host} > 80',
    name='High CPU usage',
    message='@slack-my-team CPU usage is above 80%'
)

# PagerDuty 集成
monitor = api.Monitor.create(
    type='metric alert',
    query='avg(last_5m):sum:system.cpu.user{host:my-host} > 90',
    name='Critical CPU usage',
    message='@pagerduty-Critical CPU usage is above 90%'
)

# Email 通知
monitor = api.Monitor.create(
    type='metric alert',
    query='avg(last_5m):sum:system.cpu.user{host:my-host} > 80',
    name='High CPU usage',
    message='CPU usage is above 80% @team@example.com'
)
```

**告警静默**:

```python
# 创建静默
downtime = api.Downtime.create(
    scope=['env:staging'],
    start=1640995200,  # Unix timestamp
    end=1641081600,
    message='Scheduled maintenance'
)

# 取消静默
api.Downtime.delete(downtime['id'])
```

### SLO 配置

**创建 SLO**:

```python
slo = api.ServiceLevelObjective.create(
    title='API Availability SLO',
    description='99.9% availability for API service',
    type='metric',
    thresholds=[
        {
            'timeframe': '7d',
            'target': 99.9,
            'warning': 99.95
        },
        {
            'timeframe': '30d',
            'target': 99.9,
            'warning': 99.95
        }
    ],
    query={
        'numerator': 'sum:trace.http.request{service:myapp,status:200}.as_count()',
        'denominator': 'sum:trace.http.request{service:myapp}.as_count()'
    },
    tags=['service:api', 'env:production']
)
```

## 小结

- **Agent 部署**:在 Kubernetes 使用 DaemonSet 部署,配置 RBAC 权限,启用 APM、日志、进程监控
- **指标采集**:使用 DogStatsD 发送自定义指标,配置集成采集第三方服务指标,使用自动发现简化配置
- **APM 集成**:使用自动埋点减少代码侵入,配置分布式追踪实现跨服务追踪,优化采样率和过滤敏感信息
- **日志管理**:配置日志采集路径,使用结构化日志,集成追踪 ID 实现日志与链路关联
- **告警配置**:创建指标、日志、APM 监控器,配置多渠道通知,使用 SLO 定义服务目标

---

## 常见问题

### Q1:Datadog 的定价模式是什么?

**定价模型**:

| 产品 | 计费单位 | 价格 |
|------|---------|------|
| Infrastructure | 主机数 | $15/主机/月 |
| APM | 主机数 | $31/主机/月 |
| Log Management | 数据量 | $0.10/GB |
| Custom Metrics | 指标数 | $0.01/自定义指标/月 |
| Synthetics | 测试数 | $0.50/1000 测试 |

**成本优化**:

```yaml
# 减少自定义指标数量
# 使用聚合指标而非高基数标签

# 优化日志采集
logs_config:
  container_collect_all: false
  container_collect_using_files: true

# 配置日志过滤
logs:
  - type: file
    path: /var/log/myapp/*.log
    service: myapp
    source: custom
    log_processing_rules:
      - type: exclude_at_match
        name: exclude_health_checks
        pattern: 'health check'
```

### Q2:Datadog Agent 如何排查问题?

**排查步骤**:

```bash
# 1. 检查 Agent 状态
datadog-agent status

# 2. 检查 Agent 日志
tail -f /var/log/datadog/agent.log

# 3. 检查配置
datadog-agent configcheck

# 4. 测试连接
datadog-agent diagnose

# 5. 检查集成状态
datadog-agent check mysql

# 6. 检查指标采集
datadog-agent metric

# 7. 重启 Agent
systemctl restart datadog-agent
```

**常见问题**:

```bash
# 问题 1: Agent 无法连接 Datadog
# 检查网络连接
curl -v https://api.datadoghq.com/api/v1/validate?api_key=xxx

# 问题 2: 指标未上报
# 检查 DogStatsD 端口
netstat -an | grep 8125

# 问题 3: 日志未采集
# 检查日志文件权限
ls -la /var/log/myapp/

# 问题 4: APM 数据未上报
# 检查 Trace Agent 端口
netstat -an | grep 8126
```

### Q3:Datadog 如何实现多租户?

**方案一:多组织**:

```
组织 A (生产环境)
  - 独立的 API Key
  - 独立的 Dashboard
  - 独立的监控器

组织 B (测试环境)
  - 独立的 API Key
  - 独立的 Dashboard
  - 独立的监控器
```

**方案二:标签隔离**:

```yaml
# 使用标签区分环境
DD_TAGS: "env:production,team:backend,service:api"

# 查询时过滤
env:production AND service:api

# 监控器配置
monitor = api.Monitor.create(
    query='avg(last_5m):sum:system.cpu.user{env:production} > 80',
    tags=['env:production']
)
```

### Q4:Datadog 如何与现有工具集成?

**Prometheus 集成**:

```yaml
# datadog.yaml
init_config:

instances:
  - prometheus_url: http://prometheus:9090/metrics
    namespace: prometheus
    metrics:
      - prometheus_http_requests_total
      - prometheus_http_request_duration_seconds
```

**Grafana 集成**:

```yaml
# Grafana 数据源配置
apiVersion: 1
datasources:
  - name: Datadog
    type: grafana-datadog-datasource
    access: proxy
    jsonData:
      api_key: xxx
      app_key: xxx
```

**PagerDuty 集成**:

```python
# 创建 PagerDuty 集成
from datadog import api

integration = api.Integration.create(
    type='pagerduty',
    services=[
        {
            'service_name': 'Critical',
            'service_key': 'xxx'
        }
    ]
)

# 监控器中使用
monitor = api.Monitor.create(
    query='avg(last_5m):sum:system.cpu.user{host:my-host} > 90',
    message='@pagerduty-Critical CPU usage is above 90%'
)
```

### Q5:Datadog 如何保证数据安全?

**数据加密**:

```yaml
# datadog.yaml
# 启用 TLS
api_key: xxx
site: datadoghq.com
tls_verify: true
tls_ca_cert: /etc/datadog-agent/certs/ca.crt
```

**敏感信息过滤**:

```python
# 过滤敏感标签
from ddtrace import tracer

def filter_sensitive_info(span):
    tags = span.get_tags()
    if 'password' in tags:
        span.set_tag('password', '***')
    if 'credit_card' in tags:
        span.set_tag('credit_card', '***')
    return span

tracer.configure(span_filter=filter_sensitive_info)
```

**RBAC 配置**:

```yaml
# Datadog RBAC
roles:
  - name: ReadOnly
    permissions:
      - dashboards:read
      - monitors:read
      - logs:read
  
  - name: Admin
    permissions:
      - dashboards:write
      - monitors:write
      - logs:write
      - apm:write
```

## 参考资源

- [Datadog 官方文档](https://docs.datadoghq.com/)
- [Datadog Agent 文档](https://docs.datadoghq.com/agent/)
- [Datadog APM 文档](https://docs.datadoghq.com/tracing/)
- [Datadog API 文档](https://docs.datadoghq.com/api/)
- [Datadog 集成列表](https://docs.datadoghq.com/integrations/)

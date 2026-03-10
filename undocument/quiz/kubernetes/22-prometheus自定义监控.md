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
  - 监控
---

# Prometheus自定义监控实现

## 为什么需要自定义监控？

虽然Prometheus有很多现成的Exporter，但业务系统往往需要监控特定的业务指标：

- 业务指标：订单量、用户数、交易额
- 自定义性能指标：缓存命中率、队列长度
- 业务健康指标：服务依赖状态、功能可用性

## 自定义监控的方式

```
┌─────────────────────────────────────────────────────────────┐
│                   自定义监控实现方式                          │
│                                                              │
│  方式一：应用内嵌指标                                         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  应用代码 ────→ 指标库 ────→ /metrics端点            │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  方式二：独立Exporter                                         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  应用 ────→ API/日志 ────→ Exporter ────→ /metrics   │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  方式三：Pushgateway                                          │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  短生命周期任务 ────→ Pushgateway ────→ Prometheus   │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 方式一：应用内嵌指标

### Go语言实现

```go
package main

import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10},
        },
        []string{"method", "path"},
    )

    activeConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_connections",
            Help: "Current number of active connections",
        },
    )

    ordersTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "orders_total",
            Help: "Total number of orders",
        },
        []string{"status", "product_type"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
    prometheus.MustRegister(activeConnections)
    prometheus.MustRegister(ordersTotal)
}

func main() {
    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":8080", nil)
}
```

### Python实现

```python
from prometheus_client import Counter, Gauge, Histogram, start_http_server
import time
import random

REQUEST_COUNT = Counter(
    'app_request_count',
    'Application Request Count',
    ['method', 'endpoint', 'http_status']
)

REQUEST_LATENCY = Histogram(
    'app_request_latency_seconds',
    'Application Request Latency',
    ['method', 'endpoint'],
    buckets=[0.1, 0.5, 1, 2.5, 5, 10]
)

ACTIVE_USERS = Gauge(
    'app_active_users',
    'Number of active users'
)

ORDERS_TOTAL = Counter(
    'app_orders_total',
    'Total orders',
    ['status', 'product_type']
)

def process_request(method, endpoint):
    start_time = time.time()
    
    try:
        time.sleep(random.random())
        status = 200
    except Exception:
        status = 500
    
    REQUEST_COUNT.labels(method=method, endpoint=endpoint, http_status=status).inc()
    REQUEST_LATENCY.labels(method=method, endpoint=endpoint).observe(time.time() - start_time)
    
    return status

def record_order(status, product_type):
    ORDERS_TOTAL.labels(status=status, product_type=product_type).inc()

if __name__ == '__main__':
    start_http_server(8000)
    
    while True:
        process_request('GET', '/api/users')
        ACTIVE_USERS.set(random.randint(100, 500))
        time.sleep(1)
```

### Java实现（Spring Boot）

```java
import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.Gauge;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Timer;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.actuate.autoconfigure.metrics.MetricsProperties;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.concurrent.atomic.AtomicInteger;

@SpringBootApplication
@RestController
public class Application {
    private final Counter requestCounter;
    private final Counter orderCounter;
    private final Timer requestTimer;
    private final AtomicInteger activeUsers;

    public Application(MeterRegistry registry) {
        this.requestCounter = Counter.builder("app.requests")
                .description("Total requests")
                .tag("endpoint", "/api")
                .register(registry);

        this.orderCounter = Counter.builder("app.orders")
                .description("Total orders")
                .tag("status", "success")
                .register(registry);

        this.requestTimer = Timer.builder("app.request.duration")
                .description("Request duration")
                .register(registry);

        this.activeUsers = new AtomicInteger(0);
        Gauge.builder("app.active.users", activeUsers, AtomicInteger::get)
                .description("Active users")
                .register(registry);
    }

    @GetMapping("/api/users")
    public String getUsers() {
        requestCounter.increment();
        return requestTimer.record(() -> {
            return "users";
        });
    }

    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}
```

## 方式二：独立Exporter

当无法修改应用代码时，可以编写独立的Exporter。

### Python Exporter示例

```python
from prometheus_client import start_http_server, Gauge
import requests
import time

class CustomExporter:
    def __init__(self, app_url):
        self.app_url = app_url
        
        self.cache_hit_rate = Gauge(
            'app_cache_hit_rate',
            'Cache hit rate',
            ['cache_name']
        )
        
        self.queue_length = Gauge(
            'app_queue_length',
            'Queue length',
            ['queue_name']
        )
        
        self.api_response_time = Gauge(
            'app_api_response_time_seconds',
            'API response time',
            ['endpoint']
        )

    def collect_metrics(self):
        try:
            response = requests.get(f"{self.app_url}/api/stats")
            data = response.json()
            
            self.cache_hit_rate.labels(cache_name='redis').set(data['cache']['hit_rate'])
            self.queue_length.labels(queue_name='orders').set(data['queues']['orders'])
            self.api_response_time.labels(endpoint='/api/users').set(data['api']['response_time'])
            
        except Exception as e:
            print(f"Error collecting metrics: {e}")

if __name__ == '__main__':
    exporter = CustomExporter('http://localhost:8080')
    start_http_server(8000)
    
    while True:
        exporter.collect_metrics()
        time.sleep(15)
```

### Go Exporter示例

```go
package main

import (
    "net/http"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

type CustomExporter struct {
    cacheHitRate   *prometheus.GaugeVec
    queueLength    *prometheus.GaugeVec
    apiLatency     *prometheus.GaugeVec
}

func NewCustomExporter() *CustomExporter {
    return &CustomExporter{
        cacheHitRate: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "app_cache_hit_rate",
                Help: "Cache hit rate",
            },
            []string{"cache_name"},
        ),
        queueLength: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "app_queue_length",
                Help: "Queue length",
            },
            []string{"queue_name"},
        ),
        apiLatency: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "app_api_latency_seconds",
                Help: "API latency",
            },
            []string{"endpoint"},
        ),
    }
}

func (e *CustomExporter) Collect(ch chan<- prometheus.Metric) {
    e.cacheHitRate.WithLabelValues("redis").Set(0.85)
    e.queueLength.WithLabelValues("orders").Set(100)
    e.apiLatency.WithLabelValues("/api/users").Set(0.05)
    
    e.cacheHitRate.Collect(ch)
    e.queueLength.Collect(ch)
    e.apiLatency.Collect(ch)
}

func (e *CustomExporter) Describe(ch chan<- *prometheus.Desc) {
    e.cacheHitRate.Describe(ch)
    e.queueLength.Describe(ch)
    e.apiLatency.Describe(ch)
}

func main() {
    exporter := NewCustomExporter()
    prometheus.MustRegister(exporter)
    
    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":8000", nil)
}
```

## 方式三：Pushgateway

适用于短生命周期任务的指标上报。

### 部署Pushgateway

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

### Prometheus配置

```yaml
scrape_configs:
- job_name: 'pushgateway'
  honor_labels: true
  static_configs:
  - targets: ['pushgateway:9091']
```

### 推送指标示例

```python
from prometheus_client import Counter, push_to_gateway

job_counter = Counter(
    'batch_job_processed_total',
    'Total processed items in batch job',
    ['job_name', 'status']
)

def process_batch_job():
    processed = 100
    failed = 5
    
    job_counter.labels(job_name='data_import', status='success').inc(processed)
    job_counter.labels(job_name='data_import', status='failed').inc(failed)
    
    push_to_gateway('pushgateway:9091', job='data_import', registry=job_counter.registry)

if __name__ == '__main__':
    process_batch_job()
```

## 指标类型详解

### Counter（计数器）

只增不减，用于累计值。

```go
var requestTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "requests_total",
        Help: "Total requests",
    },
    []string{"method", "status"},
)

requestTotal.WithLabelValues("GET", "200").Inc()
```

**常用查询**：
```promql
rate(requests_total[5m])
increase(requests_total[1h])
```

### Gauge（仪表盘）

可增可减，用于瞬时值。

```go
var temperature = prometheus.NewGauge(
    prometheus.GaugeOpts{
        Name: "room_temperature_celsius",
        Help: "Current room temperature",
    },
)

temperature.Set(25.5)
temperature.Inc()
temperature.Dec()
```

**常用查询**：
```promql
room_temperature_celsius
delta(room_temperature_celsius[1h])
```

### Histogram（直方图）

用于分布统计。

```go
var requestDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "request_duration_seconds",
        Help:    "Request duration",
        Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10},
    },
    []string{"method"},
)

requestDuration.WithLabelValues("GET").Observe(0.5)
```

**常用查询**：
```promql
histogram_quantile(0.99, rate(request_duration_seconds_bucket[5m]))
rate(request_duration_seconds_sum[5m]) / rate(request_duration_seconds_count[5m])
```

### Summary（摘要）

类似Histogram，但计算分位数在客户端。

```go
var requestDuration = prometheus.NewSummaryVec(
    prometheus.SummaryOpts{
        Name:       "request_duration_seconds",
        Help:       "Request duration",
        Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
    },
    []string{"method"},
)

requestDuration.WithLabelValues("GET").Observe(0.5)
```

## 最佳实践

### 1. 命名规范

```
<namespace>_<name>_<unit>
```

示例：
- `http_requests_total`
- `node_memory_usage_bytes`
- `http_request_duration_seconds`

### 2. 添加标签

```go
httpRequestsTotal.WithLabelValues("GET", "/api/users", "200").Inc()
```

### 3. 使用Histogram而非Summary

Histogram可以在服务端聚合，更适合分布式系统。

### 4. 合理设置Bucket

```go
Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10}
```

根据实际业务场景设置。

### 5. 添加HELP说明

```go
prometheus.CounterOpts{
    Name: "requests_total",
    Help: "Total number of HTTP requests processed",
}
```

## 参考资源

- [Prometheus客户端库](https://prometheus.io/docs/instrumenting/clientlibs/)
- [指标命名最佳实践](https://prometheus.io/docs/practices/naming/)
- [Pushgateway文档](https://github.com/prometheus/pushgateway)

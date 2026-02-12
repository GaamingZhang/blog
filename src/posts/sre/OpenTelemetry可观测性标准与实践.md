---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: true
article: true
category:
  - SRE
tag:
  - OpenTelemetry
  - 可观测性
  - 链路追踪
  - Prometheus
---

# OpenTelemetry 可观测性标准：统一 Traces、Metrics、Logs

想象这样一个场景：你的团队用 Jaeger 收集链路追踪、用 Prometheus 收集指标、用 ELK 收集日志。某天线上出现一次慢请求，你需要在三个系统之间来回切换，用 TraceId 手动关联日志，再对照时间戳找 Metrics 异动。更糟糕的是，当你决定把 Jaeger 替换成 Grafana Tempo 时，所有服务里的 Jaeger SDK 代码都得重写。

这正是 OpenTelemetry（OTel）要解决的核心问题：**可观测性数据的碎片化和厂商锁定**。

## 1. 为什么需要 OpenTelemetry

### 可观测性的碎片化困境

传统可观测性体系有三座孤岛：

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Jaeger    │    │ Prometheus  │    │     ELK     │
│   Traces    │    │   Metrics   │    │    Logs     │
│  jaeger-sdk │    │  micrometer │    │   logback   │
└─────────────┘    └─────────────┘    └─────────────┘
       ↑                  ↑                  ↑
   各自埋点             各自埋点            各自埋点
   互不关联             互不关联            互不关联
```

三套 SDK、三套配置、三套数据流，彼此没有关联。当你想从一条慢 Trace 跳到对应时间段的 Metrics，再跳到带有 TraceId 的日志，整个过程完全依赖人工对齐。

### 厂商锁定的代价

OpenTracing 和 OpenCensus 曾试图解决这个问题，但走了不同的路。两者的演进历程如下：

| 项目 | 创建时间 | 背景 | 覆盖信号 | 结局 |
|------|---------|------|---------|------|
| OpenTracing | 2016 | CNCF孵化，统一追踪API | 仅 Traces | 2022年归档，并入OTel |
| OpenCensus | 2018 | Google/Microsoft主导 | Traces + Metrics | 2023年归档，并入OTel |
| OpenTelemetry | 2019 | 两者合并，CNCF毕业 | Traces + Metrics + Logs | 现行标准 |

OpenTelemetry 是 OpenTracing 和 OpenCensus 合并的产物，它继承了两者的优点，并加入了 Logs 信号，形成了真正统一的可观测性标准。

### OTel 的设计哲学

OTel 的核心思路是**关注点分离**：把埋点（Instrumentation）和后端存储完全解耦。

```
应用代码
   │
   ▼
OTel SDK（统一 API）
   │
   ▼
OTel Collector（统一管道）
   │
   ├──▶ Jaeger / Tempo（Traces 后端）
   ├──▶ Prometheus / Thanos（Metrics 后端）
   └──▶ Elasticsearch / Loki（Logs 后端）
```

你的业务代码只需要面向 OTel API 埋点一次，后端想换就换，无需改动任何业务代码。

## 2. OTel 核心概念

### 三大信号

OTel 将可观测性数据统一归纳为三类信号（Signal）：

- **Traces（链路追踪）**：描述一次请求在多个服务中的完整执行路径，解决"这个请求慢在哪里"的问题。
- **Metrics（指标）**：描述系统在某个时间点的聚合状态，解决"系统现在健康吗"的问题。
- **Logs（日志）**：描述某个时间点发生的具体事件，解决"这里到底发生了什么"的问题。

三者分别擅长不同的问题域，OTel 的价值在于将它们**关联**起来——一条日志能跳转到对应的 Trace，一个 Metric 异常能关联到同时段的慢 Trace。

### 数据模型核心要素

**Resource**：描述产生遥测数据的实体，是一组静态 Attribute，例如服务名、实例 IP、K8s Pod 名称。Resource 对应的是"谁产生了这条数据"。

**Attribute**：键值对，用于描述上下文信息。OTel 定义了语义约定（Semantic Conventions），例如 HTTP 请求统一使用 `http.method`、`http.status_code`，数据库操作统一使用 `db.system`、`db.statement`，这是 OTel 实现跨语言可分析性的关键。

**Span**：Trace 的基本单元，代表一个操作。一个 Span 包含：操作名称、开始/结束时间戳、TraceId、SpanId、ParentSpanId、Attributes、Events（时间戳事件，相当于日志）、Status（OK/Error）。

**Scope（InstrumentationScope）**：标识产生遥测数据的库或模块，例如 `io.opentelemetry.spring-webmvc-6.0`，便于区分自动埋点和手动埋点的来源。

### Context Propagation：跨服务的上下文传递

Context Propagation 是链路追踪能跨服务工作的核心机制。当服务 A 调用服务 B 时，需要把 TraceId 和 SpanId"注入"到请求头里，服务 B 再"提取"出来，继续在同一个 Trace 下创建子 Span。

OTel 默认使用 **W3C TraceContext** 标准，通过 HTTP 头 `traceparent` 传递：

```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
              版本  ←────── TraceId(128bit) ──────→ ←SpanId(64bit)→ 标志位
```

其中标志位最后一位为 `01` 表示该请求被采样，`00` 表示未采样。这让下游服务无需做采样决策，只需遵从上游的决定。

### Sampling：采样的两种哲学

全量采集所有请求的 Trace 数据代价太高，采样策略决定了哪些请求值得被记录。

**Head-based Sampling（头部采样）**：在请求进入系统的第一个服务时就做出采样决定，决定后所有下游服务都遵从这个决定。优点是实现简单、开销低；缺点是决策时还不知道这个请求是否"有趣"（比如后续会出错），可能丢掉重要的异常请求。

**Tail-based Sampling（尾部采样）**：先收集所有 Span，等整条 Trace 完成后，再根据完整信息（比如是否有错误、总耗时是否超阈值）决定是否保留。优点是可以精准保留"有价值"的 Trace；缺点是需要在 Collector 层缓存大量临时数据，实现复杂且有内存压力。

:::tip 生产环境建议
对于大多数团队，可以先用 Head-based Sampling（1%~10%）降低存储成本，同时对所有错误请求设置 100% 采样。等遥测体系成熟后，再考虑在 OTel Collector 中引入 Tail-based Sampling。
:::

### OTLP 协议

OTLP（OpenTelemetry Protocol）是 OTel 定义的数据传输协议，支持两种格式：

- **gRPC**：默认端口 4317，适合高吞吐量场景，支持双向流式传输。
- **HTTP/protobuf**：默认端口 4318，适合防火墙限制 gRPC 的环境，也支持 JSON 格式。

几乎所有现代后端（Jaeger 1.35+、Grafana Tempo、Grafana Loki、Elasticsearch）都已原生支持 OTLP 接收，这意味着你的数据可以不经过任何转换直接写入后端。

## 3. OTel Collector 架构

OTel Collector 是整个 OTel 生态的枢纽，它是一个独立部署的代理/网关，承担数据接收、处理、转发的职责。

### 三阶段流水线

```
┌────────────────────────────────────────────┐
│              OTel Collector                │
│                                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │ Receiver │─▶│Processor │─▶│ Exporter │ │
│  └──────────┘  └──────────┘  └──────────┘ │
└────────────────────────────────────────────┘
```

**Receiver（接收器）**：负责从各种来源接收数据。常用的有：
- `otlp`：接收 OTel SDK 发送的 gRPC/HTTP 数据
- `prometheus`：抓取 Prometheus 格式的 `/metrics` 端点
- `jaeger`：兼容接收旧版 Jaeger 客户端的数据
- `zipkin`：兼容接收 Zipkin 格式数据

**Processor（处理器）**：对数据进行加工。常用的有：
- `batch`：将数据攒批后再发送，减少网络请求次数（几乎必配）
- `memory_limiter`：设置内存上限，防止 OOM（几乎必配）
- `filter`：根据条件过滤掉不需要的数据，例如过滤健康检查 Span
- `transform`：修改 Attribute，例如脱敏、字段重命名
- `tail_sampling`：尾部采样处理器

**Exporter（导出器）**：将处理后的数据发送到后端。常用的有：
- `otlp`：通过 OTLP 协议发送到 Jaeger/Tempo 等后端
- `prometheus`：将 Metrics 以 Prometheus 格式暴露，供 Prometheus 抓取
- `elasticsearch`：将 Logs 写入 Elasticsearch

### Agent 模式 vs Gateway 模式

在 Kubernetes 中，Collector 有两种典型的部署拓扑：

```
┌─────────────────────────────────────────────────────────┐
│  Agent 模式（每个节点一个 Collector，DaemonSet 部署）       │
│                                                         │
│  Pod A ──OTLP──▶ Node Collector ──OTLP──▶ 后端          │
│  Pod B ──OTLP──▶ Node Collector                         │
│                                                         │
│  优点：就近处理，延迟低，无需跨节点网络                      │
│  适用：Metrics、Logs 的本地采集和转发                      │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│  Gateway 模式（集中部署，Deployment）                      │
│                                                         │
│  Node Collector ──▶ ┐                                   │
│  Node Collector ──▶ ├─▶ Gateway Collector ──▶ 后端      │
│  Node Collector ──▶ ┘                                   │
│                                                         │
│  优点：集中做尾部采样、全局限流、认证鉴权                    │
│  适用：需要跨节点聚合的 Tail-based Sampling               │
└─────────────────────────────────────────────────────────┘
```

推荐的生产拓扑是**两层部署**：DaemonSet Agent 负责从 Pod 收集数据并做初步处理，Gateway Collector 负责尾部采样和向后端集中写入。

### 完整的 Collector 配置示例

```yaml
# otelcol-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
  prometheus:
    config:
      scrape_configs:
        - job_name: 'otel-collector'
          scrape_interval: 10s
          static_configs:
            - targets: ['localhost:8888']

processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 400
    spike_limit_mib: 100
  batch:
    timeout: 5s
    send_batch_size: 1024
  filter/drop_health_check:
    traces:
      span:
        - 'attributes["http.route"] == "/health"'

exporters:
  otlp/jaeger:
    endpoint: jaeger-collector:4317
    tls:
      insecure: true
  prometheus:
    endpoint: "0.0.0.0:8889"
    namespace: otel
  otlphttp/tempo:
    endpoint: http://tempo:4318

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, filter/drop_health_check, batch]
      exporters: [otlp/jaeger, otlphttp/tempo]
    metrics:
      receivers: [otlp, prometheus]
      processors: [memory_limiter, batch]
      exporters: [prometheus]
    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [otlphttp/tempo]
```

:::warning 注意 processors 的顺序
`memory_limiter` 必须放在 processors 列表的第一位。它的作用是在内存达到上限时拒绝新数据，如果放在后面，前面的 processor 已经处理了数据再被拒绝，会造成数据丢失。
:::

## 4. 语言 SDK 接入：以 Go 为例

### 初始化 Tracer Provider

Go SDK 的使用分为两步：初始化全局 Provider，然后在业务逻辑中创建 Span。

```go
package telemetry

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
    "google.golang.org/grpc"
)

func InitTracerProvider(ctx context.Context, serviceName string) (*sdktrace.TracerProvider, error) {
    // 1. 创建 OTLP gRPC 导出器，指向 OTel Collector
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithInsecure(),
        otlptracegrpc.WithEndpoint("otel-collector:4317"),
        otlptracegrpc.WithDialOption(grpc.WithBlock()),
    )
    if err != nil {
        return nil, err
    }

    // 2. 定义 Resource，描述这个服务实例
    res := resource.NewWithAttributes(
        semconv.SchemaURL,
        semconv.ServiceName(serviceName),
        semconv.ServiceVersion("1.0.0"),
        semconv.DeploymentEnvironment("production"),
    )

    // 3. 创建 TracerProvider，配置采样策略
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.ParentBased(
            sdktrace.TraceIDRatioBased(0.1), // 10% 采样
        )),
    )

    // 4. 注册为全局 Provider
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

### 在 HTTP Handler 中手动埋点

```go
package handler

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "net/http"
)

var tracer = otel.Tracer("order-service")

func CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
    // 从请求中提取上游传递的 Context（包含 TraceId）
    ctx := r.Context()

    // 创建当前服务的 Span
    ctx, span := tracer.Start(ctx, "CreateOrder")
    defer span.End()

    // 添加业务相关的 Attribute
    userID := r.Header.Get("X-User-Id")
    span.SetAttributes(
        attribute.String("user.id", userID),
        attribute.String("http.method", r.Method),
    )

    // 调用下游服务时，传递 ctx，SDK 自动注入 TraceContext 到请求头
    if err := callInventoryService(ctx, orderID); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "inventory check failed")
        http.Error(w, "Internal Error", 500)
        return
    }

    span.SetStatus(codes.Ok, "")
    w.WriteHeader(http.StatusCreated)
}
```

### W3C TraceContext 的 HTTP 传播

OTel Go SDK 通过 `otelhttp` 中间件自动处理 `traceparent` 头的注入和提取，无需手动操作：

```go
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

// 服务端：自动从请求头提取 TraceContext，创建子 Span
mux.Handle("/orders", otelhttp.NewHandler(handler, "CreateOrder"))

// 客户端：自动将当前 Span 的 TraceContext 注入到外发请求头
client := &http.Client{
    Transport: otelhttp.NewTransport(http.DefaultTransport),
}
```

`otelhttp.NewTransport` 会在发送请求前自动向请求头写入 `traceparent`，被调用的服务收到这个头后，`otelhttp.NewHandler` 会自动提取并创建子 Span。这整个过程对业务代码完全透明。

## 5. Java 自动埋点：Zero-code Instrumentation

Java 生态是 OTel 自动埋点能力最强的平台。通过 Java Agent，无需修改任何业务代码，OTel 会在 JVM 启动时通过字节码增强（ByteBuddy）自动为 Spring MVC、JDBC、Redis、Kafka 等数百个框架注入追踪逻辑。

```bash
# 只需在启动命令中挂载 Java Agent
java -javaagent:opentelemetry-javaagent.jar \
     -Dotel.service.name=order-service \
     -Dotel.exporter.otlp.endpoint=http://otel-collector:4317 \
     -Dotel.traces.sampler=parentbased_traceidratio \
     -Dotel.traces.sampler.arg=0.1 \
     -jar order-service.jar
```

Java Agent 的工作原理：JVM 在加载类文件时，Agent 拦截目标类（如 `HttpServlet`、`JdbcTemplate`）的加载过程，动态插入创建/关闭 Span 的代码，整个过程对应用代码不可见。这对于改造存量 Java 服务非常友好——不需要改一行代码，重启即可接入。

## 6. Logs 与 Traces 关联

链路追踪真正发挥威力的地方在于三大信号的关联。Logs 与 Traces 的关联依赖于在日志中注入 `trace_id` 和 `span_id`。

### Go（zap）手动注入

```go
import (
    "go.opentelemetry.io/otel/trace"
    "go.uber.org/zap"
)

func logWithTrace(ctx context.Context, logger *zap.Logger, msg string) {
    span := trace.SpanFromContext(ctx)
    sc := span.SpanContext()

    logger.Info(msg,
        zap.String("trace_id", sc.TraceID().String()),
        zap.String("span_id", sc.SpanID().String()),
        zap.Bool("trace_sampled", sc.IsSampled()),
    )
}
```

### Java（Logback）自动注入

Java Agent 配合 MDC（Mapped Diagnostic Context）可自动注入，只需修改 `logback.xml` 的 Pattern：

```xml
<pattern>%d{HH:mm:ss} %-5level [%X{trace_id},%X{span_id}] %logger{36} - %msg%n</pattern>
```

OTel Java Agent 会在每次请求处理时自动将 `trace_id`、`span_id` 写入 MDC，Logback 直接通过 `%X{}` 读取即可。

日志中有了 TraceId，在 Grafana Loki 或 Kibana 中就可以设置跳转链接，点击日志中的 TraceId 直接在 Jaeger 中打开对应的 Trace，实现 Logs→Traces 的无缝跳转。

## 7. 与 Jaeger 和 Prometheus 的整合

### OTel → Jaeger 的现代链路

Jaeger 1.35+ 已原生支持 OTLP 接收，整合非常简洁：

```
OTel SDK ──OTLP gRPC──▶ OTel Collector ──OTLP──▶ Jaeger
```

OTel Collector 到 Jaeger 的 exporter 配置如前文 `otlp/jaeger` 段所示，指向 Jaeger Collector 的 4317 端口即可。

从旧版 Jaeger SDK 迁移到 OTel SDK 时，只需：
1. 替换依赖：移除 `jaeger-client-go`，添加 OTel Go SDK 依赖
2. 修改初始化代码：从 Jaeger Config 换成 OTel TracerProvider
3. Span 创建 API 基本一致，只需调整包名

### OTel Metrics → Prometheus

OTel Collector 的 `prometheus` exporter 会将 OTel Metrics 以 Prometheus 格式暴露在 `/metrics` 端点，Prometheus 直接抓取这个端点即可。这种方式让 OTel Metrics 与原有的 Prometheus Exporter 生态完全兼容，可以在同一个 Grafana Dashboard 中展示。

:::tip OTel Metrics 与原有 Exporter 的共存
业务指标通过 OTel SDK 定义，基础设施指标（JVM、Node Exporter、kube-state-metrics）仍走原有 Prometheus Exporter 体系，两者都由 Prometheus 统一抓取，互不干扰。
:::

## 8. Kubernetes 中的 OTel 部署

### OTel Operator

OTel Operator 是一个 Kubernetes Operator，提供两个核心能力：

1. **自动注入 Java/Python Agent**：通过 `Instrumentation` CR 声明注入配置，Operator 会自动向匹配的 Pod 注入 Agent，免去手动修改每个 Deployment 的繁琐操作。

2. **管理 Collector 配置**：通过 `OpenTelemetryCollector` CR 声明 Collector 的部署模式（DaemonSet/Deployment）和配置，Operator 负责创建对应的 Kubernetes 资源。

### 完整的 Kubernetes 部署 YAML

```yaml
# 1. Instrumentation CR：定义自动注入策略
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: java-instrumentation
  namespace: production
spec:
  exporter:
    endpoint: http://otel-collector-agent:4317
  propagators:
    - tracecontext
    - baggage
  sampler:
    type: parentbased_traceidratio
    argument: "0.1"
  java:
    env:
      - name: OTEL_INSTRUMENTATION_LOGBACK_APPENDER_ENABLED
        value: "true"
---
# 2. Agent Collector（DaemonSet）：每节点一个，就近收集
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: otel-agent
  namespace: production
spec:
  mode: daemonset
  config: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
    processors:
      memory_limiter:
        limit_mib: 200
        check_interval: 1s
      batch:
        timeout: 5s
    exporters:
      otlp:
        endpoint: otel-gateway-collector:4317
        tls:
          insecure: true
    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [memory_limiter, batch]
          exporters: [otlp]
        metrics:
          receivers: [otlp]
          processors: [memory_limiter, batch]
          exporters: [otlp]
---
# 3. 为 Java 服务开启自动注入：只需加一个 annotation
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
spec:
  template:
    metadata:
      annotations:
        instrumentation.opentelemetry.io/inject-java: "production/java-instrumentation"
    spec:
      containers:
        - name: order-service
          image: order-service:1.0.0
```

Operator 检测到带有 `instrumentation.opentelemetry.io/inject-java` 注解的 Pod 后，会自动在 Pod 中注入一个 init container 下载 Java Agent，并设置 `JAVA_TOOL_OPTIONS` 环境变量，Pod 重启后即自动完成接入。

## 小结

- OTel 解决了可观测性碎片化和厂商锁定两大核心问题，通过统一 API/SDK + 灵活 Collector 实现"埋点一次，后端随意切换"。
- 三大信号（Traces/Metrics/Logs）由 OTel 统一采集，TraceId 注入日志后实现 Logs→Traces 跳转，是 OTel 相比传统方案最大的价值增量。
- OTel Collector 的三段式流水线（Receiver→Processor→Exporter）是数据处理的核心，`memory_limiter` 和 `batch` 是生产环境的必配 Processor。
- Head-based Sampling 实现简单，适合大多数场景；Tail-based Sampling 能精准捕获异常，但需要 Collector 层的内存缓冲支持。
- Java 生态通过零代码自动埋点（Java Agent）可以极低成本改造存量服务；Go 等语言通过手动 API 埋点提供更精细的控制。
- Kubernetes 中推荐 OTel Operator + DaemonSet Agent + Gateway 两层架构，通过 Annotation 实现自动注入，运维成本极低。

---

## 常见问题

### Q1：OTel SDK 和 OTel Collector 是否都必须使用？

不是。OTel SDK 可以直接将数据发送到支持 OTLP 的后端（如 Jaeger、Grafana Tempo），不经过 Collector。但在生产环境中强烈推荐引入 Collector，原因有三：一是 Collector 承担批处理和重试逻辑，应用 SDK 不需要关注网络抖动问题；二是 Collector 可以统一做数据过滤、字段脱敏等处理；三是后端地址集中在 Collector 配置中，后端切换时应用无需重启。

### Q2：Tail-based Sampling 为什么需要 Gateway 模式的 Collector？

尾部采样需要等一条 Trace 的所有 Span 都到齐后才能做决策，而同一条 Trace 的 Span 可能来自部署在不同节点的服务，发送到了不同节点上的 Agent Collector。因此尾部采样必须在一个集中的 Gateway Collector 上进行，它能汇聚来自所有 Agent 的 Span，才能看到完整的 Trace 并做出采样决策。如果在 DaemonSet Agent 上做尾部采样，则因为数据不完整而无法正确判断。

### Q3：如何从 OpenTracing/Jaeger 旧 SDK 迁移到 OTel？

迁移分三步：第一步替换依赖包，移除旧 SDK 添加 OTel SDK；第二步修改初始化代码，用 `TracerProvider` 替代旧的 Tracer 初始化逻辑；第三步调整 Span API 调用，OTel API 与 OpenTracing API 高度相似，主要是包名变化。整个过程不影响已有的 Collector 和后端，因为 Collector 的 `jaeger` receiver 仍可接收旧格式。建议逐服务迁移，新旧并存期间两套数据可在同一个 Jaeger 中查看。

### Q4：OTel Metrics 和原有 Prometheus Exporter 如何选择？

对于新开发的业务指标，推荐使用 OTel SDK 定义 Metrics，优点是能与 Traces 共享 Resource 信息，且天然支持多后端导出。对于已有的基础设施指标（Node Exporter、kube-state-metrics、JVM Exporter），建议保持原有 Prometheus Exporter 体系不变，通过 OTel Collector 的 `prometheus` receiver 接收后统一转发，避免引入不必要的迁移成本。两者可以长期共存，Prometheus 统一抓取即可。

### Q5：OTel 的 Attribute 语义约定（Semantic Conventions）有多重要？

语义约定非常重要，是 OTel 实现跨语言、跨框架可观测性的基础。例如，所有 HTTP 服务统一使用 `http.method`、`http.route`、`http.status_code`，就可以在 Grafana 中写一个通用 Dashboard 展示所有服务的 HTTP 错误率，无需为每个服务定制。违反语义约定会导致跨服务分析困难，Grafana/Jaeger 的内置查询面板也依赖这些约定才能正常工作。OTel 官方维护的语义约定文档（`opentelemetry.io/docs/specs/semconv`）是接入时必读的参考资料。

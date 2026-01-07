---
date: 2025-12-25
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - SRE
tag:
  - SRE
  - DevOps
---

# Jaeger基本概念

## 1. Jaeger的定义与发展背景

### 1.1 什么是Jaeger？

Jaeger是一个开源的、端到端的分布式追踪系统，专为微服务架构设计，用于监控、追踪和诊断分布式系统中的事务性能问题。它提供了完整的可观测性解决方案，帮助开发人员理解系统行为、识别性能瓶颈并快速定位故障。

### 1.2 分布式追踪的重要性

随着微服务架构的普及，应用程序被拆分为多个独立的服务，这些服务通过网络进行通信。这种架构带来了以下挑战：

- **复杂性增加**：请求可能跨越多个服务，涉及数百个组件
- **性能问题难以定位**：某个服务的延迟可能影响整个请求链路
- **故障传播**：一个服务的故障可能级联影响其他服务
- **服务依赖关系不清晰**：难以理解服务之间的调用关系

分布式追踪通过在请求进入系统时创建一个唯一的Trace ID，并在整个调用链路上传递这个ID，记录每个服务的处理时间、状态和元数据，从而解决了这些挑战。

### 1.3 Jaeger的发展背景

Jaeger是由Uber Technologies开发并于2016年开源的分布式追踪系统。它的设计灵感来自于Google的Dapper论文，该论文介绍了Google内部使用的分布式追踪系统。

Jaeger的发展历程：
- 2016年：Uber开源Jaeger项目
- 2017年：成为Cloud Native Computing Foundation (CNCF)的孵化项目
- 2019年：从CNCF毕业，成为顶级开源项目
- 至今：持续发展，成为最受欢迎的分布式追踪系统之一

Jaeger的主要设计目标是：
- 提供高可靠性和可扩展性
- 支持多种编程语言和框架
- 与现代微服务生态系统（如Kubernetes、Prometheus）无缝集成
- 提供直观的UI界面，便于可视化和分析追踪数据

## 2. Jaeger的核心架构与组件

### 2.1 Jaeger的整体架构

Jaeger采用了分布式架构设计，主要由以下组件组成：

![Jaeger架构图](https://www.jaegertracing.io/img/architecture.png)

### 2.2 核心组件详解

#### 2.2.1 Jaeger客户端库（Jaeger Clients）

Jaeger客户端库是嵌入在应用程序中的SDK，负责：
- 生成和传播追踪上下文（Trace ID和Span ID）
- 记录Span数据（操作名称、时间戳、标签、日志等）
- 将Span数据发送到Jaeger Agent或直接发送到Collector

Jaeger客户端支持多种编程语言，包括Go、Java、Python、Node.js、C#等，并兼容OpenTracing标准API。

#### 2.2.2 Jaeger Agent

Jaeger Agent是一个轻量级的网络守护进程，部署在每个主机上，负责：
- 接收来自客户端库的Span数据
- 批量处理和压缩Span数据，减少网络传输开销
- 将数据转发到Jaeger Collector

Agent的主要优势是：
- 减少客户端与Collector之间的连接数
- 提供负载均衡和故障转移功能
- 支持多种传输协议（如UDP、gRPC）

#### 2.2.3 Jaeger Collector

Jaeger Collector是处理和存储追踪数据的核心组件，负责：
- 接收来自Agent或直接来自客户端的Span数据
- 验证和处理追踪数据
- 应用采样策略（可选）
- 将数据存储到后端存储系统

Collector的设计特点是：
- 可水平扩展，支持高吞吐量
- 模块化设计，支持多种存储后端
- 内置采样器，可控制数据收集量

#### 2.2.4 Query Service

Query Service提供了API和UI界面，负责：
- 从存储系统检索追踪数据
- 处理用户查询请求
- 提供直观的Web界面，用于可视化和分析追踪数据

UI界面支持：
- 按服务、操作、标签等条件搜索追踪
- 可视化展示调用链路和依赖关系
- 分析性能指标和延迟分布

#### 2.2.5 Storage

Storage是Jaeger的数据持久化层，支持多种后端存储系统：
- **Cassandra**：适合大规模部署，提供高可用性和可扩展性
- **Elasticsearch**：提供强大的搜索和分析能力
- **Kafka**：用于缓冲和处理高吞吐量的追踪数据
- **内存存储**：仅用于开发和测试环境

### 2.3 Jaeger的追踪数据模型

Jaeger使用以下核心概念来表示追踪数据：

#### 2.3.1 Trace（追踪）

Trace是一个请求在分布式系统中完整的调用链路，由多个Span组成。每个Trace都有一个唯一的Trace ID。

#### 2.3.2 Span（跨度）

Span是Trace中的一个基本单元，表示一个服务或组件中的一个操作。每个Span包含：
- 操作名称（如HTTP请求路径）
- 开始和结束时间戳
- Span ID（唯一标识符）
- Parent Span ID（父Span的标识符，根Span没有父ID）
- 标签（Tags）
- 日志（Logs）
- 进程信息（Process）

#### 2.3.3 Tags（标签）

Tags是键值对，用于存储Span的元数据，例如：
- HTTP方法和状态码
- 服务名称和实例ID
- 错误状态
- 自定义业务数据

#### 2.3.4 Logs（日志）

Logs是时间戳事件，用于记录Span执行过程中的重要事件，例如：
- 错误信息和堆栈跟踪
- 调试信息
- 业务事件

#### 2.3.5 Process（进程）

Process表示生成Span的服务或进程，包含：
- 服务名称
- 主机名和IP地址
- 自定义属性

## 3. Jaeger的优势与应用场景

### 3.1 Jaeger的核心优势

#### 3.1.1 开源与标准化兼容

- Jaeger是CNCF毕业项目，具有活跃的社区支持和持续的发展
- 完全兼容OpenTracing标准，支持与其他兼容OpenTracing的系统互操作
- 开放的API设计，便于扩展和集成

#### 3.1.2 高性能与可扩展性

- 采用分布式架构设计，支持水平扩展
- 高效的数据压缩和批量处理机制，减少网络传输开销
- 支持高吞吐量的追踪数据处理，适合大规模微服务架构

#### 3.1.3 丰富的可视化功能

- 提供直观的Web界面，支持多种可视化视图
- 可以展示完整的调用链路图和服务依赖关系图
- 支持按时间、服务、操作等维度过滤和分析数据
- 提供性能指标统计和延迟分布分析

#### 3.1.4 强大的生态系统集成

- 与Kubernetes深度集成，支持自动部署和配置
- 与Prometheus、Grafana等监控工具无缝协作
- 支持与日志系统（如Elasticsearch、Fluentd）集成，实现日志与追踪的关联
- 提供与CI/CD管道的集成支持

#### 3.1.5 灵活的采样策略

- 支持多种采样策略，包括概率采样、速率限制采样、基于请求的采样等
- 可以根据业务需求和系统负载动态调整采样率
- 支持客户端和服务器端采样，灵活控制数据收集量

#### 3.1.6 多语言支持

- 提供丰富的客户端库，支持主流编程语言（Go、Java、Python、Node.js、C#等）
- 统一的API设计，便于跨语言开发和集成

### 3.2 Jaeger的典型应用场景

#### 3.2.1 微服务性能监控与分析

- 监控跨服务请求的端到端性能
- 分析每个服务的响应时间和处理延迟
- 识别系统中的性能瓶颈和异常情况

#### 3.2.2 分布式故障定位与诊断

- 快速定位故障发生的服务和组件
- 分析故障传播路径和影响范围
- 结合日志和指标数据进行根因分析

#### 3.2.3 服务依赖关系可视化

- 自动发现和绘制服务之间的调用关系图
- 识别不必要的服务依赖和潜在的架构问题
- 帮助架构师优化系统设计

#### 3.2.4 性能瓶颈识别

- 分析请求链路中的热点服务和操作
- 识别数据库查询、网络调用等耗时操作
- 提供性能优化建议和方向

#### 3.2.5 容量规划与优化

- 分析系统在不同负载下的性能表现
- 预测系统容量需求和扩展趋势
- 优化资源配置，提高系统效率

#### 3.2.6 安全审计与合规

- 记录和追踪敏感操作的执行路径
- 提供完整的操作审计日志
- 满足合规性要求，如PCI DSS、GDPR等

### 3.3 Jaeger的局限性

虽然Jaeger具有很多优势，但在使用时也需要考虑以下局限性：

- **性能开销**：启用分布式追踪会带来一定的性能开销，需要合理配置采样率
- **存储成本**：追踪数据的存储和管理需要考虑成本因素
- **学习曲线**：对于复杂的微服务架构，需要一定的学习成本来有效使用Jaeger
- **数据关联**：需要额外的工作来关联追踪数据与日志、指标等其他可观测性数据

## 4. Jaeger的使用方法与集成方式

### 4.1 Jaeger的安装与部署

#### 4.1.1 快速入门（All-in-one模式）

对于开发和测试环境，可以使用Jaeger提供的All-in-one镜像快速部署：

```bash
docker run -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 9411:9411 \
  jaegertracing/all-in-one:latest
```

此命令会启动一个包含所有Jaeger组件的容器，包括：
- Agent (端口5775, 6831, 6832)
- Collector (端口14268)
- Query Service (端口16686)
- Zipkin兼容API (端口9411)

访问 `http://localhost:16686` 即可打开Jaeger UI界面。

#### 4.1.2 Docker Compose部署

对于更复杂的测试环境，可以使用Docker Compose部署：

```yaml
version: '3.8'

services:
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "5775:5775/udp"
      - "6831:6831/udp"
      - "6832:6832/udp"
      - "5778:5778"
      - "16686:16686"
      - "14268:14268"
      - "9411:9411"
    environment:
      - COLLECTOR_ZIPKIN_HOST_PORT=:9411
  
  # 可选：添加应用服务示例
  example-service:
    image: example-service:latest
    depends_on:
      - jaeger
    environment:
      - JAEGER_AGENT_HOST=jaeger
      - JAEGER_AGENT_PORT=6831
```

#### 4.1.3 Kubernetes部署

对于生产环境，推荐使用Kubernetes部署Jaeger。Jaeger提供了官方的Helm Chart：

```bash
# 添加Jaeger Helm仓库
helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
helm repo update

# 安装Jaeger
helm install jaeger jaegertracing/jaeger \
  --namespace jaeger \
  --create-namespace \
  --set collector.service.type=LoadBalancer \
  --set query.service.type=LoadBalancer
```

### 4.2 与主流框架的集成

#### 4.2.1 Java Spring Boot集成

使用Spring Cloud Sleuth与Jaeger集成：

1. 添加依赖：
```xml
<dependency>
    <groupId>org.springframework.cloud</groupId>
    <artifactId>spring-cloud-starter-sleuth</artifactId>
</dependency>
<dependency>
    <groupId>org.springframework.cloud</groupId>
    <artifactId>spring-cloud-starter-jaeger</artifactId>
</dependency>
```

2. 配置`application.yml`：
```yaml
spring:
  application:
    name: example-service
  sleuth:
    sampler:
      probability: 0.1  # 10%的采样率
  jaeger:
    enabled: true
    udp-sender:
      host: jaeger-agent
      port: 6831
```

#### 4.2.2 Go语言集成

使用Jaeger Go客户端：

1. 安装依赖：
```bash
go get github.com/uber/jaeger-client-go
```

2. 配置Jaeger客户端：
```go
import (
    "github.com/opentracing/opentracing-go"
    "github.com/uber/jaeger-client-go/config"
)

func initJaeger(serviceName string) (opentracing.Tracer, io.Closer) {
    cfg := config.Configuration{
        ServiceName: serviceName,
        Sampler: &config.SamplerConfig{
            Type:  "const",
            Param: 1,
        },
        Reporter: &config.ReporterConfig{
            LogSpans:            true,
            LocalAgentHostPort: "jaeger-agent:6831",
        },
    }
    
    tracer, closer, err := cfg.NewTracer()
    if err != nil {
        panic(fmt.Sprintf("Could not initialize jaeger tracer: %v", err))
    }
    
    opentracing.SetGlobalTracer(tracer)
    return tracer, closer
}
```

#### 4.2.3 Node.js集成

使用Jaeger Node.js客户端：

1. 安装依赖：
```bash
npm install jaeger-client opentracing
```

2. 配置Jaeger客户端：
```javascript
const { initTracer } = require('jaeger-client');

function initJaegerTracer(serviceName) {
    const config = {
        serviceName: serviceName,
        sampler: {
            type: "const",
            param: 1,
        },
        reporter: {
            logSpans: true,
            agentHost: "jaeger-agent",
            agentPort: 6831,
        },
    };
    
    const options = {};
    return initTracer(config, options);
}
```

### 4.3 Jaeger的使用最佳实践

#### 4.3.1 合理配置采样策略

- 根据系统负载和存储容量选择合适的采样率
- 对于关键业务路径可以使用更高的采样率
- 考虑使用基于请求的采样策略，确保重要请求被追踪

#### 4.3.2 提供有意义的Span名称和标签

- 使用清晰、描述性的操作名称
- 添加关键的业务标签（如用户ID、订单ID等）
- 记录错误信息和堆栈跟踪

#### 4.3.3 关联日志与追踪数据

- 在日志中包含Trace ID和Span ID
- 使用结构化日志，便于查询和分析
- 考虑使用ELK Stack或类似工具关联日志和追踪数据

#### 4.3.4 监控Jaeger自身

- 监控Jaeger组件的性能和健康状态
- 设置适当的告警阈值
- 定期检查存储使用情况

#### 4.3.5 与其他可观测性工具集成

- 结合Prometheus收集指标数据
- 使用Grafana创建统一的监控仪表板
- 实现日志、指标和追踪数据的关联分析

## 5. 与Jaeger相关的常见问题及答案

### 5.1 Jaeger基础概念

#### 5.1.1 什么是Jaeger？它主要解决什么问题？

**答案：** Jaeger是一个开源的端到端分布式追踪系统，专为微服务架构设计。它主要解决以下问题：
- 微服务架构中请求链路的可视化和追踪
- 性能瓶颈的识别和定位
- 分布式系统中的故障快速定位和诊断
- 服务依赖关系的自动发现和可视化
- 系统性能的监控和分析

#### 5.1.2 Jaeger的Trace和Span是什么关系？各自包含哪些信息？

**答案：** Trace是一个请求在分布式系统中的完整调用链路，由多个Span组成。每个Trace都有一个唯一的Trace ID。

Span是Trace中的基本单元，表示一个服务或组件中的一个操作。每个Span包含：
- 操作名称（如HTTP请求路径）
- 开始和结束时间戳
- Span ID（唯一标识符）
- Parent Span ID（父Span的标识符，根Span没有父ID）
- 标签（Tags）：键值对元数据
- 日志（Logs）：时间戳事件
- 进程信息（Process）：生成Span的服务或进程信息

### 5.2 Jaeger架构与组件

#### 5.2.1 Jaeger的核心组件有哪些？各自的职责是什么？

**答案：** Jaeger的核心组件包括：

1. **Jaeger客户端库**：嵌入在应用程序中，负责生成和传播追踪上下文、记录Span数据并发送到Agent或Collector
2. **Jaeger Agent**：轻量级网络守护进程，接收客户端的Span数据，批量处理后转发到Collector
3. **Jaeger Collector**：处理和存储追踪数据，验证、处理、应用采样策略并存储到后端
4. **Query Service**：提供API和UI界面，检索和展示追踪数据
5. **Storage**：数据持久化层，支持Cassandra、Elasticsearch、Kafka等

#### 5.2.2 Jaeger的数据平面和控制平面分别指什么？

**答案：** 
- **数据平面**：由Jaeger客户端和Agent组成，负责处理实际的追踪数据采集、传播和转发
- **控制平面**：由Collector、Query Service和Storage组成，负责追踪数据的处理、存储和查询，以及系统的配置和管理

### 5.3 Jaeger集成与使用

#### 5.3.1 Jaeger如何与微服务架构集成？

**答案：** Jaeger与微服务架构的集成方式：
1. 在每个微服务中嵌入Jaeger客户端库
2. 配置客户端库连接到Jaeger Agent或直接连接到Collector
3. 在代码中使用客户端API创建和管理Span
4. 部署Jaeger服务端组件（Agent、Collector、Query Service、Storage）
5. 访问Jaeger UI查看和分析追踪数据

#### 5.3.2 在Kubernetes环境中如何部署Jaeger？

**答案：** 在Kubernetes环境中部署Jaeger的常用方法：
1. **使用官方Helm Chart**：
   ```bash
   helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
   helm repo update
   helm install jaeger jaegertracing/jaeger --namespace jaeger --create-namespace
   ```
2. **使用Operator**：Jaeger提供了Kubernetes Operator，简化部署和管理
3. **自定义YAML部署**：根据需求创建自定义的Deployment、Service等资源

### 5.4 Jaeger性能与优化

#### 5.4.1 Jaeger支持哪些采样策略？如何选择合适的采样策略？

**答案：** Jaeger支持多种采样策略：

1. **常量采样（Const）**：要么全部采样，要么全部不采样
2. **概率采样（Probabilistic）**：按指定概率采样
3. **速率限制采样（Rate Limiting）**：按指定速率采样
4. **远程采样（Remote）**：从控制平面获取采样策略

选择采样策略时应考虑：
- 开发测试环境：可以使用常量采样（100%）
- 生产环境低负载：可以使用概率采样（如1%）
- 生产环境高负载：可以使用速率限制采样或远程采样
- 关键业务路径：可以使用更高的采样率

#### 5.4.2 如何在生产环境中优化Jaeger的性能？

**答案：** 生产环境中优化Jaeger性能的方法：
1. **合理配置采样率**：根据系统负载和存储容量调整采样率
2. **使用Agent**：通过Agent转发数据，减少客户端与Collector的连接数
3. **批量处理**：配置客户端和Agent的批量处理参数
4. **选择合适的存储后端**：根据数据量和查询需求选择存储系统
5. **水平扩展**：对Collector和Query Service进行水平扩展
6. **监控Jaeger自身**：监控Jaeger组件的性能和健康状态

### 5.5 Jaeger与其他系统比较

#### 5.5.1 Jaeger与其他分布式追踪系统（如Zipkin）有什么区别？

**答案：** Jaeger与Zipkin的主要区别：

| 特性 | Jaeger | Zipkin |
|------|--------|--------|
| 开源组织 | CNCF | OpenZipkin |
| 兼容性 | 兼容OpenTracing和OpenTelemetry | 兼容OpenTracing和OpenTelemetry |
| 架构 | 更现代的分布式架构 | 相对简单的架构 |
| UI界面 | 更丰富的可视化功能 | 相对简洁的界面 |
| 采样策略 | 更多采样策略选项 | 基本的采样策略 |
| 存储支持 | Cassandra、Elasticsearch、Kafka等 | Cassandra、Elasticsearch、MySQL等 |
| 性能 | 更高的性能和可扩展性 | 性能较好，但扩展性相对较弱 |

#### 5.5.2 Jaeger的存储后端有哪些选择？各有什么优缺点？

**答案：** Jaeger支持的存储后端及优缺点：

1. **Cassandra**：
   - 优点：高可用性、可扩展性强、适合大规模部署
   - 缺点：查询能力相对较弱、运维复杂度较高

2. **Elasticsearch**：
   - 优点：强大的搜索和分析能力、支持复杂查询
   - 缺点：资源消耗较大、存储成本较高

3. **Kafka**：
   - 优点：高吞吐量、支持实时数据流处理
   - 缺点：不适合长期存储、需要配合其他存储系统使用

4. **内存存储**：
   - 优点：性能高、部署简单
   - 缺点：数据易丢失、仅适合开发测试环境

### 5.6 Jaeger高级应用

#### 5.6.1 Jaeger如何实现跨服务的请求追踪？

**答案：** Jaeger通过以下机制实现跨服务的请求追踪：
1. 在请求进入系统时创建一个唯一的Trace ID
2. 为每个服务处理操作创建一个Span，并生成唯一的Span ID
3. 通过HTTP头或RPC元数据传播Trace ID和Span ID
4. 子服务接收请求时，基于传入的Trace ID和Span ID创建新的Span作为子Span
5. 所有相关的Span通过Trace ID关联起来，形成完整的调用链路

#### 5.6.2 如何将Jaeger与日志系统和指标系统集成？

**答案：** 将Jaeger与日志系统和指标系统集成的方法：

1. **与日志系统集成**：
   - 在日志中包含Trace ID和Span ID
   - 使用结构化日志格式，便于查询和分析
   - 配置日志收集系统（如ELK Stack、Fluentd）关联日志和追踪数据

2. **与指标系统集成**：
   - 结合Prometheus收集系统和应用指标
   - 使用Grafana创建统一的监控仪表板，整合追踪数据和指标
   - 实现基于追踪数据的自定义指标
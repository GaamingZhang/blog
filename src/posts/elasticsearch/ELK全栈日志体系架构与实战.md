---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: true
article: true
category:
  - Elasticsearch
tag:
  - Elasticsearch
  - ELK
  - Logstash
  - Kibana
  - 日志
---

# ELK 全栈日志体系：Logstash 采集、ES 存储与 Kibana 分析实战

## 从一次排障说起

系统出现故障，你打开 Kibana，按服务名和时间范围搜索，五秒内定位到触发问题的具体请求、完整的调用链和上下文。这是 ELK 体系运转正常时应有的体验。

但在很多团队，现实是：Logstash 内存溢出频繁重启，ES 写入持续报错 `circuit_breaking_exception`，Kibana 查询超时，日志时不时出现丢失。问题的根源不是 ELK 本身，而是体系中每一层的配置和机制没有被真正理解。

本文从架构选型开始，逐层拆解 Logstash 的 Pipeline 机制、Filebeat 的采集原理、ES 的索引设计，以及 Kibana 的实战用法，目标是让你不仅知道怎么配，更知道为什么这么配。

---

## ELK vs EFK vs PLG：架构选型

在搭建日志体系之前，必须先做出架构选型，因为选错了后续迁移代价极高。

### 三种主流方案

**ELK**（Elasticsearch + Logstash + Kibana）是最经典的组合。Logstash 负责采集和转换，ES 负责存储和索引，Kibana 负责可视化。它的优势是生态完整、插件丰富，Logstash 的 Grok、mutate、aggregate 等过滤器处理复杂日志格式的能力无出其右。代价是 Logstash 基于 JVM，内存需求高，通常需要 1-2 GB 的 JVM 堆。

**EFK**（Elasticsearch + Fluentd 或 Fluent Bit + Kibana）用更轻量的采集器替换 Logstash。Fluent Bit 用 C 编写，内存占用通常在 20-50 MB，非常适合作为 Kubernetes 每个节点的 DaemonSet。当日志格式已经是 JSON 结构化时，Fluent Bit 的过滤能力完全够用，没有必要引入重量级的 Logstash。

**PLG**（Promtail + Loki + Grafana）是 Grafana Labs 推出的轻量替代方案。Loki 不对日志内容建立全文索引，只索引标签（Label），存储成本比 ES 低一个数量级。代价是查询能力受限：只能按标签过滤，不支持全文搜索和聚合分析。如果你的主要需求是"按服务名和时间范围查日志"而非"在所有日志中搜关键词"，PLG 是极具性价比的选择。

### 三方案对比

| 维度 | ELK | EFK | PLG（Loki） |
|------|-----|-----|------------|
| 采集层内存 | Logstash 1-2 GB | Fluent Bit 20-50 MB | Promtail < 50 MB |
| 存储成本 | 高（全文索引开销大） | 高（同 ES） | 低（仅索引标签） |
| 查询能力 | 强（全文搜索 + 聚合） | 强（同 ES） | 弱（标签过滤为主） |
| 日志转换能力 | 强（Grok/mutate 等丰富插件） | 中（Fluent Bit 插件较少） | 弱 |
| Kubernetes 适配 | 需要 DaemonSet + 独立 Logstash | 原生适配 | 原生适配 |
| 运维复杂度 | 高 | 中 | 低 |
| 适用场景 | 复杂日志解析、多格式来源 | 容器化环境、JSON 结构化日志 | 轻量查询、成本敏感场景 |

### 选型建议

- **有大量非结构化日志（Nginx、老旧 Java 应用）需要 Grok 解析**，或者需要复杂的日志路由、多输出目标：选 ELK
- **Kubernetes 环境，应用已输出 JSON 结构化日志**，需要低资源消耗：选 EFK
- **存储成本是核心约束，查询需求主要是按服务/时间范围过滤**：选 PLG

:::tip
两种方案可以混用：Fluent Bit 作为 Kubernetes 节点的轻量采集器（DaemonSet），将日志发往 Kafka；Logstash 从 Kafka 消费，执行重量级的格式转换。这种两级架构兼顾了资源效率和转换能力，是生产中最常见的形态。
:::

---

## Logstash 核心机制：Pipeline 模型

### Pipeline 的本质

Logstash 的所有工作都在 Pipeline 中完成。一个 Pipeline 是一条单向的数据处理流水线：

```
Input（采集）→ Queue（缓冲）→ Filter（转换）→ Output（输出）
```

每个 Pipeline 有独立的线程池。`pipeline.workers` 控制 Filter 和 Output 阶段的并行工作线程数，默认等于 CPU 核心数。`pipeline.batch.size` 控制每个 worker 每次从 Queue 中取出并处理的最大事件数，默认 125 条。这两个参数是 Logstash 性能调优的核心旋钮。

Logstash 支持配置多个独立 Pipeline（多 pipeline 模式），不同来源的日志走不同 Pipeline，互不干扰，避免一个 Pipeline 的处理延迟影响另一个。

### Input 插件

- **beats**：接收来自 Filebeat 的数据，生产中最常用的 Input
- **kafka**：从 Kafka topic 消费日志，适合两级架构中的下游处理节点
- **file**：直接读取本地文件，适合 Logstash 与应用同机部署的场景
- **http**：暴露 HTTP 端点接收推送，适合 Webhook 或自定义发送方

### Filter 插件：日志转换的核心

**grok**：解析非结构化文本日志的主力插件。Grok 的本质是命名正则组合——它内置了几百个常用模式（`%{IP}`、`%{NUMBER}`、`%{HTTPDATE}` 等），让你用简洁的语法描述日志格式。

Nginx access log 解析示例：

```ruby
filter {
  grok {
    match => {
      "message" => '%{IPORHOST:client_ip} - %{USER:ident} \[%{HTTPDATE:log_timestamp}\] "%{WORD:http_method} %{URIPATH:request_path}(?:%{URIPARAM:query_string})? HTTP/%{NUMBER:http_version}" %{NUMBER:status_code:int} %{NUMBER:response_bytes:int} "%{DATA:referer}" "%{DATA:user_agent}"'
    }
    remove_field => ["message"]
  }
}
```

Grok 解析失败时，事件会被打上 `_grokparsefailure` 标签。线上监控这个标签的比例，超过 5% 说明规则需要优化。

**mutate**：字段的增删改查。重命名字段、删除不需要的字段、类型转换都在这里完成：

```ruby
filter {
  mutate {
    rename      => { "log_timestamp" => "request_time" }
    remove_field => ["ident", "http_version"]
    convert     => { "status_code" => "integer" }
    add_field   => { "environment" => "production" }
  }
}
```

**date**：将日志中的时间字符串解析为 `@timestamp`：

```ruby
filter {
  date {
    match   => ["request_time", "dd/MMM/yyyy:HH:mm:ss Z"]
    target  => "@timestamp"
    remove_field => ["request_time"]
  }
}
```

:::warning
如果不配置 `date` 过滤器，`@timestamp` 会是 Logstash **接收到日志的时间**，而非日志产生的时间。在日志积压场景下，两者可能相差数小时，会导致 Kibana 时间轴完全错乱。
:::

**json**：解析已经是 JSON 格式的日志字段：

```ruby
filter {
  json {
    source => "message"
    target => "app"          # 解析后的字段放在 app 命名空间下
  }
  # 如果日志已是顶层 JSON，不指定 target，字段会直接展平到顶层
}
```

**aggregate**：多行日志合并，Java 异常堆栈是最典型的场景：

```ruby
filter {
  # 先用 grok 识别堆栈的起始行
  grok {
    match => { "message" => "(?<exception_start>^[A-Z].*Exception.*)" }
    tag_on_failure => []
  }

  aggregate {
    task_id      => "%{host}-%{log_file}"
    code         => "
      map['stack_trace'] ||= ''
      map['stack_trace'] += event.get('message') + '\n'
      event.cancel()
    "
    push_map_as_event_on_timeout => true
    timeout                      => 3
    timeout_tags                 => ["aggregated"]
  }
}
```

### Output 插件

**elasticsearch output** 是最常见的输出目标：

```ruby
output {
  elasticsearch {
    hosts         => ["http://es-node1:9200", "http://es-node2:9200"]
    index         => "logs-%{[service]}-%{+YYYY.MM.dd}"
    user          => "logstash_writer"
    password      => "${LOGSTASH_ES_PASSWORD}"
    ilm_enabled   => true
    ilm_rollover_alias => "logs"
    ilm_policy    => "logs-ilm-policy"
    # 批量写入配置
    bulk_max_size => 2000
    flush_size    => 500
  }
}
```

### 完整 logstash.conf 示例

以下是处理 Nginx 访问日志和 Java 应用日志的完整配置：

```ruby
# /etc/logstash/conf.d/app-logs.conf

input {
  beats {
    port => 5044
  }
  kafka {
    bootstrap_servers => "kafka-broker:9092"
    topics            => ["app-logs-java"]
    group_id          => "logstash-consumer"
    codec             => "json"
    consumer_threads  => 4
  }
}

filter {
  # 根据 Filebeat 发来的 tags 字段区分日志类型
  if "nginx" in [tags] {
    grok {
      match => {
        "message" => '%{IPORHOST:client_ip} - %{USER:ident} \[%{HTTPDATE:log_timestamp}\] "%{WORD:http_method} %{URIPATH:request_path}(?:%{URIPARAM:query_string})? HTTP/%{NUMBER:http_version}" %{NUMBER:status_code:int} %{NUMBER:response_bytes:int} "%{DATA:referer}" "%{DATA:user_agent}" %{NUMBER:upstream_response_time:float}'
      }
      remove_field => ["message", "ident", "http_version"]
    }
    date {
      match        => ["log_timestamp", "dd/MMM/yyyy:HH:mm:ss Z"]
      target       => "@timestamp"
      remove_field => ["log_timestamp"]
    }
    mutate {
      add_field => { "log_type" => "nginx_access" }
    }
  }

  if "java-app" in [tags] {
    # 应用已输出 JSON，直接解析
    json {
      source => "message"
    }
    date {
      match        => ["timestamp", "ISO8601"]
      target       => "@timestamp"
      remove_field => ["timestamp"]
    }
    mutate {
      add_field => { "log_type" => "java_app" }
    }
  }

  # 通用清理
  mutate {
    remove_field => ["agent", "ecs", "input", "log"]
  }
}

output {
  elasticsearch {
    hosts              => ["http://es-hot-1:9200", "http://es-hot-2:9200"]
    index              => "logs-%{[service]:unknown}-%{+YYYY.MM.dd}"
    ilm_enabled        => true
    ilm_rollover_alias => "logs"
    ilm_policy         => "logs-policy"
    user               => "logstash_writer"
    password           => "${ES_PASSWORD}"
    bulk_max_size      => 2000
  }
}
```

### Pipeline 性能调优

```yaml
# /etc/logstash/logstash.yml
pipeline.workers: 8           # 建议等于 CPU 核心数
pipeline.batch.size: 500      # 每批处理事件数，默认 125，日志场景可适当调大
pipeline.batch.delay: 50      # 批次超时（毫秒），防止低流量时长时间等待凑批

# JVM 堆配置（/etc/logstash/jvm.options）
# -Xms2g
# -Xmx2g
# JVM 堆建议设为相同的最小值和最大值，避免运行时动态扩缩堆带来的 GC 停顿
```

---

## Filebeat：采集层的核心机制

### Registry：at-least-once 保证的实现

Filebeat 的可靠性依赖 **Registry 机制**。Registry 是一个本地数据库文件（默认路径 `/var/lib/filebeat/registry/`），记录了每个被监控日志文件的**当前读取偏移量（offset）**。

工作流程是：Filebeat 读取日志文件的一批数据，发送给 Logstash 或 ES，收到确认响应后，才将该批数据的偏移量更新到 Registry。这确保了即使 Filebeat 进程崩溃重启，也能从上次确认的位置继续读取，不会丢失数据。

代价是 **at-least-once**（至少一次）而非恰好一次：如果 Filebeat 发送成功但在更新 Registry 之前崩溃，重启后会重新发送最后一批数据，导致重复。接收方（Logstash 或 ES）需要有幂等处理能力，或者接受少量重复数据。

:::warning
如果 Registry 文件损坏或被删除，Filebeat 无法恢复偏移量，默认会从文件**末尾**开始读取（由 `ignore_older` 配置决定），导致历史日志丢失。Registry 文件在容器化部署时必须挂载到宿主机路径，不能放在容器的临时文件系统里。
:::

### Filebeat 配置示例

```yaml
# /etc/filebeat/filebeat.yml

filebeat.inputs:
  # Nginx 访问日志
  - type: log
    enabled: true
    paths:
      - /var/log/nginx/access.log
    tags: ["nginx"]
    fields:
      service: nginx
      environment: production
    fields_under_root: true   # 字段直接放到顶层而非 fields 子键下

  # Java 应用日志（多行合并异常堆栈）
  - type: log
    enabled: true
    paths:
      - /var/log/app/*.log
    tags: ["java-app"]
    fields:
      service: user-service
      environment: production
    fields_under_root: true
    multiline:
      pattern: '^[[:space:]]+(at|\.{3})\b|^Caused by:'
      negate: false
      match: after
      max_lines: 500
      timeout: 5s

# 直发 Logstash（需要 Logstash 复杂处理时）
output.logstash:
  hosts: ["logstash:5044"]
  loadbalance: true
  bulk_max_size: 2048

# 也可以直发 ES（日志格式已是 JSON 且无需复杂转换时）
# output.elasticsearch:
#   hosts: ["http://es-node:9200"]
#   index: "logs-%{[service]}-%{+yyyy.MM.dd}"

# Filebeat 监控（用于观测 Filebeat 自身性能）
monitoring.enabled: true
monitoring.elasticsearch:
  hosts: ["http://es-node:9200"]
```

### Filebeat + Kubernetes 自动发现

在 Kubernetes 中，Pod 不断创建销毁，手动配置日志路径不现实。Filebeat 的 Autodiscovery 机制可以通过 Kubernetes API 动态发现 Pod，并根据 Pod 的 annotations 决定采集策略：

```yaml
# filebeat-kubernetes.yml（关键片段）
filebeat.autodiscover:
  providers:
    - type: kubernetes
      node: ${NODE_NAME}     # 每个 DaemonSet Pod 只处理本节点的容器
      hints.enabled: true    # 从 Pod annotations 读取采集配置
      hints.default_config:
        type: container
        paths:
          - /var/log/containers/*-${data.kubernetes.container.id}.log

# Pod 侧通过 annotations 控制采集行为：
# co.elastic.logs/enabled: "true"
# co.elastic.logs/multiline.pattern: "^[0-9]{4}"
# co.elastic.logs/multiline.negate: "true"
# co.elastic.logs/multiline.match: "after"
```

### 直发 ES vs 经过 Logstash 的选择

| 场景 | 推荐选择 |
|------|---------|
| 日志已是 JSON 结构化，字段无需转换 | 直发 ES |
| 日志是文本格式，需要 Grok 解析 | 经过 Logstash |
| 多输出目标（ES + Kafka + S3） | 经过 Logstash |
| Kubernetes 轻量采集，资源受限 | Fluent Bit 替代 Filebeat |

---

## ES 日志索引设计

### 滚动索引与 Index Alias

日志数据按时间持续增长，不能无限写入一个索引。生产中使用 **Index Alias + Rollover** 机制实现动态滚动：

写入始终通过一个固定的 Alias（例如 `logs-write`）完成，ES 在后台维护实际的物理索引（`logs-000001`、`logs-000002`...）。当物理索引满足滚动条件（大小或文档数超过阈值、时间超过设定）时，Rollover API 自动创建新索引并将 Alias 指向新索引，旧索引对外只读。

这对应用层完全透明——写入代码永远只写 `logs-write`，不关心底层是第几号索引。

### ILM：索引生命周期管理

ILM（Index Lifecycle Management）是自动化管理索引生命周期的核心机制。一个典型的日志索引会经历四个阶段：

```
Hot（活跃写入）
  ↓（当索引超过 50GB 或 7 天）
Warm（只读，降低副本数，迁移到低性能节点）
  ↓（30 天后）
Cold（极低频访问，冻结索引，仅保留 Metadata）
  ↓（90 天后）
Delete（自动删除）
```

完整 ILM 策略 JSON：

```json
PUT _ilm/policy/logs-policy
{
  "policy": {
    "phases": {
      "hot": {
        "min_age": "0ms",
        "actions": {
          "rollover": {
            "max_primary_shard_size": "50gb",
            "max_age": "7d"
          },
          "set_priority": {
            "priority": 100
          }
        }
      },
      "warm": {
        "min_age": "7d",
        "actions": {
          "shrink": {
            "number_of_shards": 1
          },
          "forcemerge": {
            "max_num_segments": 1
          },
          "allocate": {
            "require": {
              "data": "warm"
            },
            "number_of_replicas": 1
          },
          "set_priority": {
            "priority": 50
          }
        }
      },
      "cold": {
        "min_age": "30d",
        "actions": {
          "allocate": {
            "require": {
              "data": "cold"
            },
            "number_of_replicas": 0
          },
          "freeze": {},
          "set_priority": {
            "priority": 0
          }
        }
      },
      "delete": {
        "min_age": "90d",
        "actions": {
          "delete": {}
        }
      }
    }
  }
}
```

### 索引模板：禁用 Dynamic Mapping

Dynamic Mapping 允许 ES 自动为新字段创建索引。听起来方便，但在日志场景是灾难：某个服务将 `user_id` 打成了字符串，ES 自动创建 `keyword` 类型的 `user_id` 字段；另一个服务用的是 `long` 类型——两者冲突，后者全部写入失败。

```json
PUT _index_template/logs-template
{
  "index_patterns": ["logs-*"],
  "priority": 200,
  "template": {
    "settings": {
      "number_of_shards": 2,
      "number_of_replicas": 1,
      "refresh_interval": "30s",
      "index.lifecycle.name": "logs-policy",
      "index.lifecycle.rollover_alias": "logs-write"
    },
    "mappings": {
      "dynamic": "strict",
      "properties": {
        "@timestamp":            { "type": "date" },
        "level":                 { "type": "keyword" },
        "service":               { "type": "keyword" },
        "version":               { "type": "keyword" },
        "environment":           { "type": "keyword" },
        "log_type":              { "type": "keyword" },
        "traceId":               { "type": "keyword" },
        "spanId":                { "type": "keyword" },
        "message":               {
          "type": "text",
          "fields": { "keyword": { "type": "keyword", "ignore_above": 512 } }
        },
        "client_ip":             { "type": "ip" },
        "http_method":           { "type": "keyword" },
        "request_path":          { "type": "keyword" },
        "status_code":           { "type": "integer" },
        "response_bytes":        { "type": "long" },
        "upstream_response_time":{ "type": "float" },
        "user_agent":            { "type": "keyword" },
        "duration_ms":           { "type": "long" },
        "host":                  { "type": "keyword" },
        "kubernetes": {
          "properties": {
            "namespace": { "type": "keyword" },
            "pod_name":  { "type": "keyword" },
            "node_name": { "type": "keyword" }
          }
        }
      }
    }
  }
}
```

### 分片规划

日志场景的分片数建议：单个主分片大小控制在 **20-50 GB** 之间。分片过小（< 5 GB）会导致大量小分片，查询时协调节点汇总开销高；分片过大（> 50 GB）会造成节点间分片迁移缓慢，且 Merge 耗时过长。

对于每天写入 100 GB 日志、保留 7 天热数据的场景，ILM 配置 `max_primary_shard_size: 50gb`，每天约产生 2-3 个新索引滚动，整体热层保持 15-20 个分片，是合理的规模。

---

## Kibana 实战

### Discover：日志搜索与 KQL

Discover 是日志搜索的主界面。Kibana 使用 **KQL（Kibana Query Language）** 作为搜索语法，比 Lucene 语法更直观：

```
# 按字段精确匹配
level: "ERROR"
service: "user-service"

# 范围查询
status_code >= 500
upstream_response_time > 1.0

# 多条件组合
level: "ERROR" and service: "user-service"
(level: "ERROR" or level: "WARN") and environment: "production"

# 通配符（注意性能，避免前置通配符）
request_path: "/api/users/*"

# 存在性检查
traceId: *

# 按 traceId 追踪完整调用链（最常用的排障查询）
traceId: "7b4f2e1a9c3d5b8f"

# 查找慢请求 Top N（配合 Dashboard 使用）
service: "order-service" and duration_ms > 3000
```

### Dashboard：日志分析面板设计

一个实用的日志分析 Dashboard 至少包含以下几个面板：

- **错误率趋势折线图**：以 1 分钟为粒度，聚合 `level: ERROR` 的日志数量，与总日志数相比得出错误率
- **HTTP 5xx 状态码趋势**：`status_code >= 500` 的请求数随时间变化
- **慢请求 Top N 表格**：按 `upstream_response_time` 降序排列，展示 `service`、`request_path`、最大/P99 响应时间
- **各服务日志量分布饼图**：快速发现某个服务日志量异常激增（通常意味着循环打印错误）
- **Kubernetes 命名空间日志量柱状图**：按 `kubernetes.namespace` 分组，监控各团队日志量

### Lens：可视化构建器

Lens 是 Kibana 的拖拽式可视化工具，相比传统的 Aggregation-based visualization 更直观。构建"每分钟错误日志数"图表的步骤：

1. 选择数据源（index pattern）
2. Y 轴：选择 Count of records，加入 KQL 过滤器 `level: "ERROR"`
3. X 轴：选择 `@timestamp`，设置 Date histogram，间隔 1 分钟
4. Breakdown by：选择 `service`，按服务着色区分

### Alerts：基于日志的告警

Kibana Alerting 支持直接基于 ES 查询结果设置告警。两类典型场景：

**错误数量告警**：每 5 分钟执行一次查询，如果最近 5 分钟内 `level: ERROR` 的日志超过 100 条，触发告警：

```
Rule type: Elasticsearch query
KQL: level: "ERROR" and environment: "production"
Time window: 5 minutes
Threshold: count > 100
Action: 发送 Slack 通知 / 调用 Webhook
```

**关键词告警**：监控日志中出现特定字符串（如 `OutOfMemoryError`、`database connection refused`）：

```
Rule type: Elasticsearch query
KQL: message: "OutOfMemoryError" or message: "connection refused"
Time window: 1 minute
Threshold: count > 0
```

---

## 生产部署架构

### Kafka 作为缓冲层

直接让 Filebeat 写 Logstash、Logstash 写 ES，是没有缓冲的同步链路。业务高峰期日志量激增时，ES 写入压力超过处理能力，会导致 Logstash 背压堆积，进而拖慢 Filebeat 的发送，最终可能造成日志丢失。

引入 Kafka 作为缓冲层后，链路变为：

```
Filebeat → Kafka → Logstash → ES
（生产者）  （缓冲） （消费者）  （存储）
```

Kafka 解耦了生产速率和消费速率：Filebeat 按日志产生的速率写入 Kafka，Logstash 按 ES 能承受的速率消费 Kafka。ES 故障维护期间，日志积压在 Kafka 而非丢失，恢复后可以正常回放。

Kafka topic 的分区数建议：以峰期日志吞吐量（MB/s）除以单分区吞吐上限（通常 10-20 MB/s），再乘以 1.5 作为余量。Logstash 的 `consumer_threads` 应等于 topic 分区数，确保并行消费。

### Kubernetes 中的完整部署架构

```
每个节点                    独立部署                  ES 集群
┌─────────────┐            ┌──────────────┐          ┌──────────────┐
│ Fluent Bit  │            │  Logstash    │          │ Master Node  │
│ (DaemonSet) │──→ Kafka ──│ (StatefulSet)│──→       │ Data Node x3 │
│             │            │  x3 实例     │          │ (Hot/Warm)   │
└─────────────┘            └──────────────┘          └──────────────┘
```

ES 集群节点角色分离是高可用的关键：
- **Master Node**（3 个，纯 Master 角色）：负责集群状态管理，不承担数据存储和查询，防止 Master 被写入压力拖垮导致集群不稳定
- **Hot Data Node**（3-6 个，高性能 SSD）：承接活跃写入，配置 `node.attr.data: hot`
- **Warm Data Node**（2-4 个，大容量 HDD）：承接 ILM 迁移过来的历史数据，配置 `node.attr.data: warm`
- **Coordinating Node**（可选，2 个）：专门处理 Kibana 的查询请求，避免查询压力影响写入节点

---

## 常见问题排查

### 日志丢失排查

日志丢失通常发生在以下环节：

**Filebeat Registry 损坏**：Filebeat 无法读取 Registry 时，默认从文件末尾开始，导致历史数据丢失。检查方式：`filebeat test config` + 查看 Registry 目录权限和文件完整性。

**Kafka 消息积压过期**：Kafka topic 的 `retention.ms` 默认 7 天，如果 Logstash 消费中断超过 7 天（例如 ES 长期故障），积压的消息会被 Kafka 自动删除。监控 Kafka 的 Consumer Group Lag，及时发现消费滞后。

**Logstash Dead Letter Queue**：Logstash 处理失败的事件会被丢弃。开启 DLQ 可以将这些事件写入本地文件，事后排查：

```yaml
# logstash.yml
dead_letter_queue.enable: true
dead_letter_queue.max_bytes: 1024mb
```

### ES 写入慢：bulk 调优

ES 写入慢的排查方向：
- 查看 `_cat/thread_pool/bulk?v`，如果 `queue` 长期大于 0，说明写入线程池已满
- `bulk_max_size` 过小（频繁小批次写入）或过大（单批超过 100MB 导致超时）都会降低吞吐
- 日志索引的 `refresh_interval` 应设为 30s-60s 而非默认 1s

### Logstash JVM 内存溢出

Logstash OOM 的根本原因通常是 pipeline 中事件积压超过 JVM 堆上限：

- 将 JVM 堆设为固定值（`-Xms2g -Xmx2g`），避免动态扩缩带来的 GC 压力
- 启用 Persistent Queue（持久化队列），将队列从内存移到磁盘，防止 Logstash 重启时事件丢失：

```yaml
queue.type: persisted
queue.max_bytes: 4gb
```

- 检查 Filter 中是否有性能低下的 Grok 规则（贪婪匹配 `.*`），使用 Logstash 的慢日志（`slowlog.threshold.warn`）定位慢规则

### Kibana 查询超时

- 分片数过多：单次查询广播到几百个分片，协调节点汇总超时。通过 ILM 的 `shrink` 动作将 Warm 阶段的索引收缩到 1 个分片
- 查询时间范围过大：用户在 Kibana 选择了"最近 1 年"，触发对上千个索引的扫描。通过 Kibana 的 `timeFieldName` 配置确保所有查询都带上时间范围过滤
- `text` 字段的高基数聚合：对 `message` 字段做 Terms Aggregation 是性能杀手，应改为对 `message.keyword` 且配合 `ignore_above: 512` 的子字段操作

---

## 小结

- ELK 适合需要复杂日志转换的场景，EFK 适合 Kubernetes 轻量采集，PLG 适合存储成本敏感场景；三者可以分层混用
- Logstash 的 Pipeline 模型（Input → Filter → Output）是其核心，`pipeline.workers` 和 `batch.size` 是性能调优的核心旋钮；Grok 是解析非结构化日志的主力，`_grokparsefailure` 率是规则质量的量化指标
- Filebeat 的 Registry 机制保证了 at-least-once 采集，Registry 必须持久化存储，容器化部署时不能放在临时文件系统
- ES 日志索引设计的三板斧：禁用 Dynamic Mapping（防字段类型冲突）、配置 ILM（自动化生命周期管理）、合理规划分片数（单分片 20-50 GB）
- Kafka 缓冲层是日志管道高可用的关键，解耦了采集速率和消费速率，防止 ES 故障时日志丢失
- 日志索引的 `refresh_interval` 应设为 30-60s，而非默认的 1s——日志场景不需要近实时搜索，但频繁 Refresh 会显著增加写入压力

---

## 常见问题

### Q1：Logstash 的 Persistent Queue 和 Kafka 缓冲层有什么区别，什么场景下用哪个？

两者都是防止数据丢失的缓冲机制，但层次不同。

**Logstash Persistent Queue** 是 Logstash 内部的磁盘队列，位于 Input 和 Filter 之间。它解决的是 Logstash 自身的问题：进程崩溃或重启时，已接收但尚未处理完的事件不会丢失。它的容量受单台 Logstash 主机磁盘限制，通常设置 4-8 GB 即可。

**Kafka 缓冲层** 是系统级的解耦缓冲，位于采集层（Filebeat/Fluent Bit）和处理层（Logstash）之间。它解决的是采集速率和消费速率不匹配的问题：ES 故障维护时，日志持续写入 Kafka，Logstash 恢复后可以回放数小时乃至数天的积压数据。容量受 Kafka 集群规模控制，可以达到 TB 级。

实践中，两者应该同时使用：Kafka 防止长时间 ES 故障导致的数据丢失，Logstash Persistent Queue 防止 Logstash 自身重启导致的数据丢失。

### Q2：ILM 策略的 Warm 阶段为什么要执行 shrink 和 forcemerge？

Warm 阶段索引已不再写入，执行这两个操作是为了优化存储效率和查询性能。

`shrink` 将索引收缩为更少的分片（通常 1 个主分片）。Hot 阶段配置多个分片是为了并行写入，但历史索引不再写入时，多余的分片只会增加查询时的协调开销和存储元数据负担。假设你有 100 个历史索引，每个 5 个分片，ES 集群就有 500 个分片需要维护——远超实际需要。

`forcemerge` 将每个分片内的多个 Segment 合并为 1 个 Segment，物理删除被标记为删除的文档，降低磁盘占用，同时让后续偶发的历史查询直接扫描单个 Segment，无需遍历多个小 Segment。

注意：`forcemerge` 是重量级操作，执行时会占用大量 I/O，因此 ILM 中的 `forcemerge` 应该放在 Warm 阶段而非 Hot 阶段，确保在数据迁移到低性能节点、不影响活跃写入后再执行。

### Q3：ES 集群主分片不可修改，如果初期分片规划失误怎么补救？

这是生产中很常见的问题。ES 索引创建后主分片数不可修改（因为路由公式 `hash(id) % number_of_primary_shards` 依赖这个值，修改会导致已有数据找不到）。

补救方式有两种：

**方法一：Reindex**。创建一个分片数正确的新索引，使用 `POST _reindex` 将旧索引数据迁移过来，完成后更新 Alias 指向新索引。缺点是对大索引（几百 GB 以上）耗时很长，期间需要读写共存。

**方法二：借助 ILM 的滚动机制自然修复**。修改 Index Template 中的分片数配置，然后触发 Rollover，新产生的索引会使用新的分片数。旧索引无需迁移，随着 ILM 生命周期到达 Delete 阶段自动删除。这种方式影响最小，适合不需要立刻修正的场景。

根本预防方案：按照"单分片 20-50 GB"的原则，结合预期每日日志量和保留期，在 ILM 的 Rollover 条件中用 `max_primary_shard_size` 控制大小，让 ES 自动管理索引数量，而非人工猜测每个索引应该设几个分片。

### Q4：如何判断 Logstash 的 Pipeline 出现了性能瓶颈？以及如何定位是哪个 Filter 慢？

Logstash 暴露了完整的监控 API，用于诊断 Pipeline 性能。

**判断是否存在瓶颈**：

```bash
# 查看 Pipeline 统计数据
curl http://localhost:9600/_node/stats/pipelines

# 关注以下指标：
# pipeline.events.in vs pipeline.events.out：如果 in 远大于 out，说明有积压
# pipeline.queue.events_count：持久化队列中的积压事件数
# pipeline.events.filtered/duration_in_millis：平均每个事件的处理时间
```

**定位慢 Filter**：开启 Logstash 慢日志：

```yaml
# logstash.yml
slowlog.threshold.warn: 2s    # 处理超过 2s 的事件记录到慢日志
slowlog.threshold.info: 1s
```

慢日志会记录每个 Filter 插件的执行时间，能精确定位是哪个 Grok 规则或哪段 Ruby 代码拖慢了整个 Pipeline。

常见的性能杀手：使用 `.*` 的贪婪 Grok 规则（在复杂日志上可能触发指数级回溯）、Ruby filter 中的重量级计算、aggregate filter 在高并发下的状态维护开销。

### Q5：Kibana 如何通过 traceId 实现日志与分布式 Trace 的联动跳转？

这需要在 Kibana 的 Data View 配置中添加 **Field formatters（字段格式化器）** 或 **Derived Fields**。

配置步骤：进入 Kibana → Stack Management → Data Views，找到日志的 Data View，选择 `traceId` 字段，设置格式化器类型为 **Url**，URL 模板填写 Jaeger 或 Tempo 的 Trace 查询地址：

```
# Jaeger
https://jaeger.internal/trace/{{value}}

# Grafana Tempo
https://grafana.internal/explore?orgId=1&left={"datasource":"Tempo","queries":[{"query":"{{value}}"}]}
```

配置完成后，在 Kibana Discover 中，`traceId` 字段会渲染为可点击的链接，点击直接跳转到 Jaeger/Tempo 中对应的完整调用链视图——从日志到 Trace，零复制粘贴。

反向联动（从 Trace 跳转到日志）需要在 Grafana 的 Tempo 数据源配置 Derived Fields，将 Trace 中的 `service.name` 和 `traceId` 映射为一个 Elasticsearch/Loki 的查询链接，实现双向导航。

## 参考资源

- [Elasticsearch 官方文档](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html)
- [Logstash 配置指南](https://www.elastic.co/guide/en/logstash/current/configuration.html)
- [Filebeat 官方文档](https://www.elastic.co/guide/en/beats/filebeat/current/index.html)

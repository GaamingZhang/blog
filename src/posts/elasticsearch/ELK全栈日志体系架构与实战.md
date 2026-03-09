---
date: 2026-02-13
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Elasticsearch
tag:
  - Elasticsearch
  - ClaudeCode
---

# ELK 全栈日志体系：Logstash 采集、ES 存储与 Kibana 分析实战

## 从一次排障说起

系统出现故障，你打开 Kibana，按服务名和时间范围搜索，五秒内定位到触发问题的具体请求、完整的调用链和上下文。这是 ELK 体系运转正常时应有的体验。

但在很多团队，现实是：Logstash 内存溢出频繁重启，ES 写入持续报错 `circuit_breaking_exception`，Kibana 查询超时，日志时不时出现丢失。问题的根源不是 ELK 本身，而是每一层的配置和机制没有被真正理解。

---

## ELK vs EFK vs PLG：架构选型

| 维度 | ELK | EFK | PLG（Loki） |
|------|-----|-----|------------|
| 采集层内存 | Logstash 1-2 GB | Fluent Bit 20-50 MB | Promtail < 50 MB |
| 存储成本 | 高（全文索引） | 高（同 ES） | 低（仅索引标签） |
| 查询能力 | 强（全文搜索 + 聚合） | 强 | 弱（标签过滤） |
| 日志转换能力 | 强（丰富插件） | 中 | 弱 |
| 适用场景 | 复杂日志解析 | 容器化环境 | 成本敏感场景 |

**选型建议**：非结构化日志需 Grok 解析选 ELK；Kubernetes 环境 JSON 日志选 EFK；成本敏感选 PLG。

:::tip
可混用：Fluent Bit 作轻量采集器，Logstash 从 Kafka 消费执行格式转换，兼顾资源效率和转换能力。
:::

---

## Logstash 核心机制：Pipeline 模型

### Pipeline 的本质

Logstash 所有工作在 Pipeline 中完成：

```
Input（采集）→ Queue（缓冲）→ Filter（转换）→ Output（输出）
```

`pipeline.workers` 控制并行线程数（默认 CPU 核心数），`pipeline.batch.size` 控制每次处理事件数（默认 125 条）。

### Filter 插件：日志转换核心

**grok** 解析非结构化文本。Nginx access log 示例：

```ruby
filter {
  grok {
    match => {
      "message" => '%{IPORHOST:client_ip} - %{USER:ident} \[%{HTTPDATE:log_timestamp}\] "%{WORD:http_method} %{URIPATH:request_path} HTTP/%{NUMBER:http_version}" %{NUMBER:status_code:int} %{NUMBER:response_bytes:int}'
    }
    remove_field => ["message"]
  }
  date {
    match => ["log_timestamp", "dd/MMM/yyyy:HH:mm:ss Z"]
    target => "@timestamp"
  }
}
```

Grok 解析失败会打上 `_grokparsefailure` 标签，线上监控该标签比例超过 5% 说明规则需优化。

:::warning
不配置 `date` 过滤器，`@timestamp` 会是 Logstash 接收时间而非日志产生时间，日志积压时会导致 Kibana 时间轴错乱。
:::

### 核心配置示例

```ruby
input {
  beats { port => 5044 }
}

filter {
  if "nginx" in [tags] {
    grok {
      match => { "message" => '%{IPORHOST:client_ip} - %{USER:ident} \[%{HTTPDATE:log_timestamp}\] "%{WORD:http_method} %{URIPATH:request_path} HTTP/%{NUMBER:http_version}" %{NUMBER:status_code:int} %{NUMBER:response_bytes:int}' }
    }
    date { match => ["log_timestamp", "dd/MMM/yyyy:HH:mm:ss Z"] }
  }
}

output {
  elasticsearch {
    hosts => ["http://es-node1:9200"]
    index => "logs-%{[service]}-%{+YYYY.MM.dd}"
    ilm_enabled => true
    ilm_policy => "logs-policy"
  }
}
```

---

## Filebeat：采集层核心机制

### Registry：at-least-once 保证

Filebeat 可靠性依赖 **Registry 机制**，记录每个日志文件的当前读取偏移量。Filebeat 发送数据并收到确认后才更新偏移量，确保进程崩溃重启后能从上次确认位置继续读取。代价是 **at-least-once**：发送成功但更新 Registry 前崩溃，重启后会重发最后一批数据。

:::warning
Registry 文件在容器化部署时必须挂载到宿主机，不能放在容器临时文件系统里。
:::

### Filebeat 配置示例

```yaml
filebeat.inputs:
  - type: log
    paths: ["/var/log/nginx/access.log"]
    tags: ["nginx"]
    fields: {service: nginx, environment: production}
    fields_under_root: true

  - type: log
    paths: ["/var/log/app/*.log"]
    tags: ["java-app"]
    multiline:
      pattern: '^[[:space:]]+(at|\.{3})\b|^Caused by:'
      negate: false
      match: after

output.logstash:
  hosts: ["logstash:5044"]
  loadbalance: true
```

---

## ES 日志索引设计

### ILM：索引生命周期管理

ILM 自动化管理索引生命周期：

```
Hot（活跃写入）→ Warm（只读迁移）→ Cold（冻结）→ Delete（删除）
```

核心配置：

```json
PUT _ilm/policy/logs-policy
{
  "policy": {
    "phases": {
      "hot": {
        "actions": {
          "rollover": {
            "max_primary_shard_size": "50gb",
            "max_age": "7d"
          }
        }
      },
      "warm": {
        "min_age": "7d",
        "actions": {
          "shrink": { "number_of_shards": 1 },
          "forcemerge": { "max_num_segments": 1 }
        }
      },
      "delete": {
        "min_age": "90d",
        "actions": { "delete": {} }
      }
    }
  }
}
```

### 禁用 Dynamic Mapping

Dynamic Mapping 允许 ES 自动为新字段创建索引，但在日志场景是灾难：某服务将 `user_id` 打成字符串，ES 自动创建 `keyword` 类型；另一服务用 `long` 类型——两者冲突，后者全部写入失败。

```json
PUT _index_template/logs-template
{
  "index_patterns": ["logs-*"],
  "template": {
    "mappings": {
      "dynamic": "strict",
      "properties": {
        "@timestamp": { "type": "date" },
        "level": { "type": "keyword" },
        "service": { "type": "keyword" },
        "message": { "type": "text" }
      }
    }
  }
}
```

### 分片规划

单个主分片大小控制在 **20-50 GB**。分片过小导致大量小分片，查询汇总开销高；分片过大会造成节点间迁移缓慢。

---

## Kibana 实战

### Discover：日志搜索与 KQL

Kibana 使用 **KQL** 作为搜索语法：

```
# 按字段精确匹配
level: "ERROR"
service: "user-service"

# 多条件组合
level: "ERROR" and service: "user-service"

# 按 traceId 追踪调用链
traceId: "7b4f2e1a9c3d5b8f"
```

### Dashboard：日志分析面板

实用 Dashboard 应包含：
- **错误率趋势折线图**：1 分钟粒度聚合 `level: ERROR` 日志数量
- **HTTP 5xx 趋势**：`status_code >= 500` 请求数随时间变化
- **慢请求 Top N**：按 `upstream_response_time` 降序排列
- **服务日志量分布**：快速发现异常激增

---

## 生产部署架构

### Kafka 缓冲层

```
Filebeat → Kafka → Logstash → ES
```

Kafka 解耦生产与消费速率，ES 故障期间日志积压在 Kafka，恢复后可回放。

### Kubernetes 部署架构

```
Fluent Bit(DaemonSet) → Kafka → Logstash(StatefulSet) → ES集群
```

ES 集群节点角色分离：Master Node（3个）管理集群状态；Hot Data Node（3-6个）承接写入；Warm Data Node（2-4个）存储历史数据。

---

## 小结

- ELK 适合复杂日志转换场景，EFK 适合 Kubernetes 轻量采集，PLG 适合成本敏感场景
- Logstash Pipeline 模型是核心，`pipeline.workers` 和 `batch.size` 是性能调优关键
- Filebeat Registry 保证 at-least-once 采集，必须持久化存储
- ES 日志索引设计三板斧：禁用 Dynamic Mapping、配置 ILM、合理规划分片
- Kafka 缓冲层是日志管道高可用关键，解耦采集与消费速率

---

## 常见问题

### Q1：Logstash 内存溢出如何排查和解决？

**排查步骤**：

1. **检查 JVM 堆内存使用**：通过 JMX 或 Logstash API 查看 `jvm_memory_used_bytes`，确认是否接近堆内存上限
2. **分析 GC 日志**：启用 GC 日志（`-Xlog:gc*`），观察 Full GC 频率和耗时，频繁 Full GC 说明内存不足
3. **检查管道配置**：`pipeline.batch.size` 过大或 `pipeline.workers` 过多会导致内存占用飙升

**解决方案**：

```ruby
# 1. 增加堆内存（建议不超过物理内存的 50%）
-Xms4g
-Xmx4g

# 2. 优化管道配置
pipeline.workers: 2  # 根据 CPU 核心数调整
pipeline.batch.size: 100  # 降低批次大小
pipeline.batch.delay: 50  # 增加批次延迟

# 3. 启用持久化队列，防止数据丢失
queue.type: persisted
path.queue: /data/logstash/queue
```

### Q2：ES 写入报错 circuit_breaking_exception 怎么处理？

**原因**：JVM 内存不足触发熔断器保护机制，ES 拒绝新请求以防止 OOM。

**排查**：

```bash
# 查看熔断器状态
GET _nodes/stats/breaker

# 查看当前内存使用
GET _nodes/stats/jvm
```

**解决方案**：

```json
# 1. 调整熔断器阈值（不推荐，治标不治本）
PUT _cluster/settings
{
  "persistent": {
    "indices.breaker.total.limit": "70%"
  }
}

# 2. 优化查询和聚合，减少内存占用
# - 避免高基数字段聚合
# - 使用 doc_values 替代 fielddata
# - 限制聚合桶数量

# 3. 增加节点内存或扩容节点数量
```

### Q3：Filebeat 日志丢失或重复怎么排查？

**日志丢失排查**：

1. **检查 Registry 文件**：确认 Registry 文件持久化到宿主机，容器重启后 Registry 仍然存在
2. **检查输出目标状态**：查看 Logstash 或 Kafka 是否正常接收，网络是否稳定
3. **查看 Filebeat 日志**：`/var/log/filebeat/filebeat.log` 中是否有发送失败记录

**日志重复排查**：

1. **at-least-once 语义**：Filebeat 发送成功但更新 Registry 前崩溃，重启后会重发
2. **去重配置**：在 Logstash 或 ES 层配置去重

```ruby
# Logstash 使用 fingerprint 插件去重
filter {
  fingerprint {
    source => ["message", "@timestamp"]
    target => "[@metadata][_id]"
    method => "SHA1"
  }
}

output {
  elasticsearch {
    document_id => "%{[@metadata][_id]}"
  }
}
```

### Q4：Kibana 查询超时如何优化？

**优化查询条件**：

```
# 缩小时间范围，避免全量扫描
@timestamp >= "2026-02-27" and @timestamp < "2026-02-28"

# 使用精确匹配而非通配符
service: "user-service"  # 好
service: *user*          # 差，性能差

# 避免高开销操作
# - 避免对 text 字段排序
# - 避免深度分页（from + size > 10000）
```

**优化索引设计**：

```json
# 1. 增加刷新间隔，减少实时性换取性能
PUT logs-*/_settings
{
  "index": {
    "refresh_interval": "30s"
  }
}

# 2. 启用索引排序，加速时间范围查询
PUT _index_template/logs-template
{
  "template": {
    "settings": {
      "index.sort.field": ["@timestamp"],
      "index.sort.order": ["desc"]
    }
  }
}

# 3. 使用索引生命周期管理，及时删除过期数据
```

### Q5：ILM 策略不生效怎么排查？

**排查步骤**：

```bash
# 1. 检查索引是否应用 ILM 策略
GET logs-*/_settings?filter_path=**.settings.index.lifecycle

# 2. 检查 ILM 策略状态
GET _ilm/explain/logs-2026.02.27

# 3. 查看是否有错误信息
GET _ilm/explain
```

**常见问题**：

1. **索引未应用 ILM 策略**：检查索引模板是否正确配置 `lifecycle.name`

```json
PUT _index_template/logs-template
{
  "template": {
    "settings": {
      "index.lifecycle.name": "logs-policy",
      "index.lifecycle.rollover_alias": "logs"
    }
  }
}
```

2. **别名配置错误**：ILM rollover 需要索引别名

```json
# 创建别名指向初始索引
POST _aliases
{
  "actions": [
    {
      "add": {
        "index": "logs-000001",
        "alias": "logs",
        "is_write_index": true
      }
    }
  ]
}
```

3. **阶段条件未满足**：检查 `min_age` 和 rollover 条件是否满足

```bash
# 手动触发 rollover 测试
POST logs/_rollover
{
  "conditions": {
    "max_age": "7d",
    "max_primary_shard_size": "50gb"
  }
}
```

---

## 参考资源

- [Elasticsearch 官方文档](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html)
- [Logstash 配置指南](https://www.elastic.co/guide/en/logstash/current/configuration.html)
- [Filebeat 官方文档](https://www.elastic.co/guide/en/beats/filebeat/current/index.html)

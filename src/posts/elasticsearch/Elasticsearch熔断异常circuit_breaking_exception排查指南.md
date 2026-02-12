---
date: 2026-02-11
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Elasticsearch
tag:
  - Elasticsearch
  - ClaudeCode
---

# Elasticsearch 熔断异常 circuit_breaking_exception 排查指南

## 熔断器是保护机制，不是 Bug

线上某个时间段，业务日志突然出现大量如下错误：

```
[circuit_breaking_exception] [parent] Data too large, data for
[<http_request>] would be [8589934592/8gb], which is larger than the
limit of [7516192768/7gb], real usage: [7412345678/6.9gb], new bytes
reserved: [1073741824/1gb]
```

第一反应往往是"集群挂了"。但实际上，这是 ES 的自我保护机制在正常工作——它检测到这次请求如果继续执行，将导致 JVM 堆溢出（OOM），于是主动拒绝，让请求快速失败，而不是让整个节点崩溃。

熔断器的设计哲学是：**用可预见的请求失败，换取不可预见的节点崩溃**。单次请求失败，客户端可以重试或降级；但 JVM OOM 会造成节点不可用，进而触发分片重新分配，引发集群雪崩。

持续触发 `circuit_breaking_exception` 本身不代表集群故障，但它是一个明确的信号：**当前资源已接近极限，必须介入处理**。

---

## ES 熔断器体系

ES 并不是只有一个熔断器，而是有一套分层的熔断体系，每个熔断器负责不同的内存维度。

```
JVM Heap（总内存）
  │
  └── parent 熔断器（总守卫，默认 95% heap）
        ├── fielddata 熔断器（fielddata 缓存，默认 40% heap）
        ├── request 熔断器（单次请求估算内存，默认 60% heap）
        ├── in_flight_requests 熔断器（传输中的请求数据，默认 100% heap）
        └── accounting 熔断器（已加载 Lucene 对象，默认 100% heap）
```

`parent` 是所有子熔断器的总守卫：即使每个子熔断器都没有超出自身阈值，只要它们加在一起超过了 `parent` 的上限，`parent` 就会触发。

| 熔断器 | 默认阈值 | 保护目标 | 常见触发原因 |
|--------|---------|---------|------------|
| `parent` | 95% heap | 总内存上限 | 各类内存综合过高 |
| `fielddata` | 40% heap | fielddata 缓存 | 对 text 字段做聚合/排序 |
| `request` | 60% heap | 单次请求内存 | 大聚合、高基数 Terms |
| `in_flight_requests` | 100% heap | 传输中请求体 | 超大 bulk 请求 |
| `accounting` | 100% heap | Lucene segment 对象 | 过多 Segment，mapping 膨胀 |

`in_flight_requests` 和 `accounting` 默认阈值虽然是 100%，但它们受 `parent` 约束，实际上不可能真正达到 100%。

---

## 快速定位：是哪个熔断器触发了

排查的第一步是读懂错误信息，而不是直接翻配置。错误响应中包含了所有需要的线索。

**典型错误响应：**

```json
{
  "error": {
    "root_cause": [
      {
        "type": "circuit_breaking_exception",
        "reason": "[fielddata] Data too large, data for [user_name] would be
                   [2147483648/2gb], which is larger than the limit of
                   [1073741824/1gb]",
        "bytes_wanted": 2147483648,
        "bytes_limit": 1073741824,
        "durability": "PERMANENT"
      }
    ],
    "type": "circuit_breaking_exception"
  },
  "status": 429
}
```

关键字段解读：
- `reason` 中的括号内容（如 `[fielddata]`、`[parent]`、`[request]`）——**直接告诉你是哪个熔断器**
- `bytes_wanted`：本次请求需要的内存
- `bytes_limit`：当前熔断器阈值
- `durability: PERMANENT`：表示即使清理缓存也无法解除，需要重新评估查询或扩容；`TRANSIENT` 则说明缓存清理后可能恢复

**查看熔断器当前状态：**

```
GET /_nodes/stats/breaker
```

响应中重点关注每个节点的熔断器指标：

```json
{
  "nodes": {
    "node-1": {
      "breakers": {
        "fielddata": {
          "limit_size_in_bytes": 1073741824,
          "limit_size": "1gb",
          "estimated_size_in_bytes": 956301312,
          "estimated_size": "912mb",
          "overhead": 1.03,
          "tripped": 42
        },
        "parent": {
          "limit_size_in_bytes": 8053063680,
          "limit_size": "7.5gb",
          "estimated_size_in_bytes": 7900000000,
          "estimated_size": "7.3gb",
          "overhead": 1.0,
          "tripped": 5
        }
      }
    }
  }
}
```

`tripped` 字段是历史触发次数累计值。**监控 `tripped` 的增量**比看绝对值更有意义——如果这个数字在持续增长，说明问题还在持续发生。

---

## 场景一：fielddata 熔断器触发

### 根因

`fielddata` 是 ES 为支持对 `text` 类型字段做聚合和排序而引入的内存结构。当你在一个 `text` 字段上执行 Terms 聚合或排序时，ES 需要将该字段所有文档的值全部加载到 JVM 堆内存，构建一个"字段值 → 文档列表"的映射表，这就是 fielddata。

一个百万文档的索引，若 `user_name` 字段为 `text` 类型，一次聚合可能就需要加载几个 GB 的 fielddata。

### 排查步骤

**第一步：确认字段类型**

```
GET /my_index/_mapping
```

如果看到查询涉及的字段是 `text` 类型且没有 `fielddata: true` 的显式设置，但查询仍然在执行聚合，说明已经开启了 fielddata（或 mapping 中有 `fielddata: true`）。

**第二步：查看 fielddata 当前占用**

```
GET /_nodes/stats/indices/fielddata?fields=*
```

**第三步：找出消耗最大的索引**

```
GET /_cat/fielddata?v&s=size:desc
```

响应示例：

```
id                     host      ip        node    field       size
n1Xz1qXzTYKpCeFb6qaqQ 10.0.0.1  10.0.0.1  node-1  user_name  1.2gb
n1Xz1qXzTYKpCeFb6qaqQ 10.0.0.1  10.0.0.1  node-1  status     456mb
```

这张表直接告诉你哪个字段的 fielddata 占用最大。

### 解决方案

**根本修复：修改 mapping，使用 keyword 子字段**

这是唯一彻底的解决方案。将需要聚合/排序的字段改为 `keyword` 类型：

```json
PUT /my_index/_mapping
{
  "properties": {
    "user_name": {
      "type": "text",
      "fields": {
        "keyword": {
          "type": "keyword",
          "ignore_above": 256
        }
      }
    }
  }
}
```

之后聚合时使用 `user_name.keyword` 而非 `user_name`。`keyword` 类型使用 **doc_values**（列式存储在磁盘上，查询时按需加载），而非 fielddata（全量加载到堆内存），内存占用从 GB 级降到可忽略不计。

**为什么 keyword 不走 fielddata？** doc_values 在索引阶段以列式结构写入磁盘，聚合时直接读取磁盘上的列式数据，不需要将整个字段的所有值装入内存，也不需要构建反向映射表。这是 ES 5.x 以后处理聚合/排序字段的推荐方式。

**临时缓解：清除 fielddata 缓存**

```
POST /_cache/clear?fielddata=true
```

清除后内存立即释放，熔断器解除，但下次执行相同查询时 fielddata 会再次加载。治标不治本，仅用于应急。

**调整阈值（最后手段）：**

```json
PUT /_cluster/settings
{
  "persistent": {
    "indices.breaker.fielddata.limit": "60%"
  }
}
```

调大阈值给的是缓冲时间，不解决根本问题，且存在 OOM 风险。

---

## 场景二：request 熔断器触发

### 根因

`request` 熔断器保护的是单次查询请求的**临时内存**，主要包括聚合过程中构建的中间数据结构。高基数（High Cardinality）的 Terms 聚合是最常见的触发原因：

- `user_id` 字段有 1000 万个唯一值，执行 Terms 聚合时 ES 需要为每个唯一值维护一个桶（bucket），内存开销与唯一值数量成正比。
- 嵌套聚合（Nested Aggregation）会使内存消耗呈乘积增长。
- 大时间范围的 `date_histogram` 聚合，若 `interval` 设置过小，桶数量同样爆炸。

### 排查步骤

**第一步：用 Profile API 分析查询内存消耗**

```json
POST /my_index/_search
{
  "profile": true,
  "aggs": {
    "user_terms": {
      "terms": { "field": "user_id" }
    }
  }
}
```

Profile 响应中的 `aggregations` 部分会显示每个聚合阶段的耗时，结合 `_nodes/stats/breaker` 的实时数据，能定位哪个聚合触发了熔断。

**第二步：检查 slow query log**

```
GET /my_index/_settings?include_defaults=true
```

确认是否开启了慢查询日志（`search.slowlog.threshold.query.warn`），慢查询日志中的 `[total_shards_hit]` 和执行时间是定位问题查询的直接线索。

### 解决方案

**限制 Terms 聚合的 size 参数**

大多数业务场景并不需要 Top-1000 万，限制 `size` 到实际需要的数量（如 100 或 1000）可以大幅降低内存消耗：

```json
{
  "aggs": {
    "user_terms": {
      "terms": {
        "field": "user_id",
        "size": 100
      }
    }
  }
}
```

**改用 composite 聚合分页**

当确实需要遍历所有唯一值时，`composite` 聚合支持分页，避免一次性将所有桶加载到内存：

```json
{
  "aggs": {
    "user_composite": {
      "composite": {
        "size": 1000,
        "sources": [
          { "user_id": { "terms": { "field": "user_id" } } }
        ],
        "after": { "user_id": "last_user_id_from_previous_page" }
      }
    }
  }
}
```

每次只取 1000 个桶，通过 `after` 游标翻页，内存消耗固定不随数据量增长。

---

## 场景三：parent 熔断器触发（最严重）

`parent` 熔断器触发说明整个 JVM 堆的使用率接近上限，是最严重的情况，单纯调整某个子熔断器的阈值无法解决问题。

### 排查步骤

**第一步：查看 JVM 堆使用率**

```
GET /_cat/nodes?v&h=name,heap.percent,heap.current,heap.max,gc.collectors.old.collection_count,gc.collectors.old.collection_time
```

响应示例：

```
name    heap.percent heap.current heap.max  gc.collectors.old.collection_count gc.collectors.old.collection_time
node-1  91           7.3gb        8gb       128                                 45230ms
node-2  72           5.8gb        8gb       12                                  890ms
```

`heap.percent > 85%` 且 `gc.collectors.old.collection_count` 持续增长，说明 Full GC 频繁，JVM 已经在竭力回收内存。`collection_time` 的累计值（45230ms = 45 秒）代表节点在 GC 上总共花费的时间——这个数字越大，代表节点越繁忙。

**第二步：查看各组件内存分布**

```
GET /_nodes/stats/indices?human
```

重点关注以下字段，找出内存大户：

```json
{
  "indices": {
    "fielddata": { "memory_size": "2.1gb" },
    "query_cache": { "memory_size": "512mb" },
    "request_cache": { "memory_size": "256mb" },
    "segments": {
      "memory": "1.8gb",
      "terms_memory": "980mb",
      "stored_fields_memory": "450mb",
      "doc_values_memory": "320mb"
    }
  }
}
```

`segments.memory` 是 Lucene Segment 本身占用的堆内存，与 Segment 数量和索引大小正相关。ES 7.0 之后大量 Segment 元数据已迁移到堆外内存（Off-Heap），但 term dictionary 等结构仍在堆上。

**第三步：统计各索引的 Segment 数量**

```
GET /_cat/indices?v&h=index,docs.count,store.size,segments.count&s=segments.count:desc
```

Segment 数量异常多（单个节点上某个索引超过 1000 个 Segment）通常是因为 Merge 跟不上写入速度，大量小 Segment 积压。

### 解决方案优先级

**立即措施：释放缓存**

```
POST /_cache/clear
```

`_cache/clear` 会清理 query cache、request cache 和 fielddata cache，通常能立即释放数百 MB 到数 GB 的内存，使 heap 从红色恢复到安全水位。但这是临时措施，下次查询会重新填充缓存。

**短期措施：Force Merge 冷索引**

对不再写入的历史索引执行 Force Merge，将大量小 Segment 合并为 1 个，显著减少 Segment 占用的堆内存：

```
POST /logs-2025-12/_forcemerge?max_num_segments=1
```

注意：**Force Merge 会占用大量 I/O 和 CPU，绝对不要对正在活跃写入的索引执行**，且需要在业务低峰期操作。

**扩容方向**

如果排查后发现内存大户合理（数据量确实大），需要从以下两个方向扩容：

1. **纵向扩容**：增大 JVM 堆内存。但有一条铁则：**堆内存不超过物理内存的 50%，且不超过 31GB**。
2. **横向扩容**：增加数据节点，通过 reindex 或 `_shrink`/`_split` 重新分配分片。

---

## JVM 堆内存配置的正确姿势

### 为什么上限是 31GB

这是 JVM 的 **CompressedOops（压缩对象指针）** 机制决定的。

在 64 位 JVM 中，默认情况下每个对象指针占 8 字节。当堆内存不超过约 32GB 时，JVM 会自动启用 CompressedOops，将对象指针压缩到 4 字节表示，内存占用减半，GC 效率大幅提升。一旦堆内存超过这个临界点，CompressedOops 失效，相同数量的对象需要更多内存，导致在堆从 31GB 增长到 32GB 时，实际可用的有效内存不升反降。

实际临界值因 JVM 版本略有差异，保守做法是将堆内存上限设置为 **30GB**，确保 CompressedOops 始终有效。

```
# jvm.options 配置示例
-Xms30g
-Xmx30g
```

### 物理内存分配原则

ES 不是只用 JVM 堆内存，Lucene 大量使用操作系统的 **Page Cache**（堆外内存）缓存 Segment 文件。Page Cache 越大，文件读取越快。因此：

```
物理内存分配原则：
  JVM 堆 ≤ 物理内存的 50%
  剩余 50% 留给操作系统 Page Cache
```

一台 64GB 物理内存的节点，推荐分配 30GB 给 JVM 堆，剩余 34GB 由操作系统管理，用作 Lucene 的 Page Cache。

---

## 预防措施：监控与告警

排查问题是被动响应，真正的稳定性来自主动监控。以下是熔断相关的关键监控指标：

| 指标 | 告警阈值 | 说明 |
|------|---------|------|
| `jvm.mem.heap_used_percent` | > 75% 告警，> 85% 严重 | 堆使用率持续高位说明有内存泄漏或容量不足 |
| `breaker.*.tripped`（增量） | > 0 即告警 | 任何熔断器触发都值得关注 |
| `gc.collectors.old.collection_time`（增量） | 每分钟 > 10s | Full GC 停顿时间过长影响集群稳定性 |
| `indices.fielddata.memory_size` | > fielddata limit 的 70% | 接近上限时提前告警留出响应时间 |

使用 Prometheus + elasticsearch_exporter 时，以下告警规则可作为参考：

```yaml
- alert: ElasticsearchCircuitBreakerTripped
  expr: increase(elasticsearch_breakers_tripped_total[5m]) > 0
  for: 0m
  labels:
    severity: warning
  annotations:
    summary: "ES 熔断器触发"
    description: "节点 {{ $labels.node }} 的 {{ $labels.breaker }} 熔断器在过去 5 分钟内触发"

- alert: ElasticsearchHeapHighUsage
  expr: elasticsearch_jvm_memory_used_bytes{area="heap"} /
        elasticsearch_jvm_memory_max_bytes{area="heap"} > 0.85
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "ES 堆内存使用率过高"
    description: "节点 {{ $labels.node }} 堆使用率超过 85% 持续 5 分钟"
```

---

## 排查流程总览

```
收到 circuit_breaking_exception
          │
          ↓
  读取 reason 字段，判断熔断器类型
          │
  ┌───────┼───────┬───────────┐
  │       │       │           │
fielddata request parent  in_flight
  │       │       │
  ↓       ↓       ↓
检查    检查    查 JVM
mapping 聚合     堆使用
text字段  size    和 GC
是否开了  是否过大
fielddata
  │       │       │
  ↓       ↓       ↓
改为   限制size  清缓存
keyword或  或改用   + Force
doc_values composite  Merge
聚合     分页聚合   + 扩容
          │
          ↓
  调整阈值（最后手段，治标）
  + 根本优化（mapping/查询/容量）
```

---

## 小结

- `circuit_breaking_exception` 是 ES 的主动保护，触发时说明请求的内存需求超过了熔断器设定的上限，ES 选择快速失败而非让节点 OOM 崩溃。
- ES 有多个熔断器（parent、fielddata、request、in_flight_requests、accounting），排查的第一步是从错误信息中确认是哪个熔断器触发。
- `fielddata` 熔断器触发的根本原因通常是对 `text` 字段做了聚合/排序；根本修复是改用 `keyword` 类型（使用 doc_values 而非 fielddata）。
- `request` 熔断器触发通常与高基数 Terms 聚合或嵌套聚合有关；根本修复是限制 `size` 参数或改用 `composite` 聚合分页。
- `parent` 熔断器触发是最严重的情况，代表整体堆内存接近上限，需要通过清缓存、Force Merge、优化 mapping 和查询、扩容节点等组合手段解决。
- JVM 堆内存遵循两条铁则：**不超过物理内存的 50%**，**不超过 31GB**（压缩指针临界值）。
- 调大熔断器阈值是最后手段，而非解决方案；根本解法在于合理的 mapping 设计、优化聚合查询、控制数据量和 Segment 数量。

---

## 常见问题

### Q1：fielddata 和 doc_values 的本质区别是什么，为什么 keyword 字段不会触发 fielddata 熔断器？

fielddata 是一种**行式到列式的运行时转换**：ES 在执行聚合时，临时将索引中的行式数据（倒排索引结构）转换为列式格式，加载到 JVM 堆内存，供聚合操作消费。这个转换过程不仅消耗内存，还消耗 CPU。

doc_values 是**索引阶段就以列式结构写入磁盘**的数据结构：当你定义一个 `keyword`（或 `numeric`、`date`）类型的字段时，ES 在写入文档的同时将该字段值以列式格式（类似 Parquet 的列存储）写入磁盘上的 `.dvm` 文件。聚合时直接读取这些列式文件，不需要构建内存映射表，且 Lucene 会利用操作系统的 Page Cache 缓存频繁访问的部分，而 Page Cache 使用的是堆外内存，不占用 JVM 堆。

简而言之：fielddata 是"查询时构建，存在堆内存"，doc_values 是"写入时构建，存在磁盘+Page Cache"。对于需要聚合和排序的字段，**始终优先使用可使用 doc_values 的类型（keyword、numeric、date）**，而非 text 类型加 fielddata。

### Q2：清除 fielddata 缓存后，ES 会对正在执行的查询有影响吗？

`POST /_cache/clear?fielddata=true` 会清除内存中已加载的 fielddata 缓存，但**不会中断正在执行的查询**。已经在使用 fielddata 进行聚合的请求会继续持有对应内存引用直到完成。缓存清除后，**下一个**需要 fielddata 的查询会重新从磁盘加载，此时会有一次明显的延迟峰值。

如果清除缓存是在业务高峰期执行，需要预期下一批查询的响应时间会显著变长（重建 fielddata 的开销）。建议在清除后，使用 `warmup` 策略提前跑一批查询，让 fielddata 重新进入缓存，避免用户侧感知到延迟跳升。

### Q3：`durability: PERMANENT` 和 `durability: TRANSIENT` 的区别，哪种情况更严重？

这两个值描述的是熔断器触发的持久性：

- `TRANSIENT`（临时性）：内存压力是暂时的，通常由查询完成后内存被释放，或清除缓存后可以恢复。例如 fielddata 熔断器触发，清除缓存后再次查询即可正常执行。
- `PERMANENT`（永久性）：即使清除缓存，当前请求也无法完成，因为请求本身需要的内存量就超过了熔断器限制。例如一个需要 10GB 临时内存的聚合，在 8GB 堆的机器上无论如何清缓存都无法执行。

`PERMANENT` 更严重——它意味着**这个查询在当前硬件配置下根本无法执行**，必须要么优化查询降低内存需求，要么扩大堆内存/增加节点。遇到 `PERMANENT` 错误时，优先检查查询设计（是否有不合理的大聚合），而非调整熔断器阈值。

### Q4：in_flight_requests 熔断器触发时，应该如何定位是哪个客户端发送了超大请求？

`in_flight_requests` 熔断器保护的是当前**正在网络传输中的请求体**总大小，最常见触发场景是 bulk 写入的单个批次过大（如单批 200MB 的文档）。

定位步骤：

1. **查看当前活跃的 HTTP 连接和请求**：
   ```
   GET /_nodes/hot_threads
   GET /_tasks?detailed=true&actions=*bulk*
   ```

2. **检查 HTTP 访问日志**（如果 ES 开启了 access log）：大请求在日志中表现为 `content-length` 异常大。

3. **通过网络层监控**：在反向代理（Nginx/HAProxy）层记录请求体大小。

根本解决方案是在客户端控制 bulk 批次大小。官方建议单批次 5MB~15MB 或 1000~5000 条文档，实际最优值需要通过压测确定。可以通过 `http.max_content_length`（默认 100mb）在服务端限制单请求体大小，超过限制直接返回 400，比触发熔断器更友好。

### Q5：为什么熔断器阈值调高了，问题还是会重复出现？

这是最常见的误解：调高熔断器阈值只是移动了"警戒线"，并没有减少实际的内存消耗。这类似于把水位告警线从 80% 提高到 90%——水还是那么多，只是你被告知得更晚了。

根本问题在于：
1. **mapping 设计不合理**：text 字段开启 fielddata，所有聚合请求都在消耗内存
2. **查询设计不合理**：无限制的高基数聚合，每次都需要巨量内存
3. **容量不足**：数据量或并发量已经超出当前硬件能力

将阈值从 40% 调到 60%，在 8GB 堆的机器上只是让 fielddata 从最多 3.2GB 变成 4.8GB。如果根本问题是大量 text 字段聚合，内存消耗依然会持续增长，最终触发 parent 熔断器，甚至引发 OOM。

正确的处理优先级应该是：**优化 mapping（用 keyword 替代 text 字段聚合）→ 优化查询（限制聚合 size，改用 composite）→ 通过 ILM 控制索引数据量 → 扩容节点** → 最后才考虑调整阈值。

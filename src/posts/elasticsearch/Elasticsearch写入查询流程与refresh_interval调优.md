---
date: 2026-02-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Elasticsearch
tag:
  - Elasticsearch
  - ClaudeCode
---

# Elasticsearch 写入查询流程与 refresh_interval 调优

## 问题的本质

很多工程师在使用 Elasticsearch 时，第一次碰到这样的现象：明明刚刚写入了一条文档，立刻查询却返回空结果；等了一秒钟后，数据才"出现"。这就是 ES 的"近实时"（Near Real-Time）搜索的本质，也是理解 ES 性能调优的起点。

要真正理解这一秒延迟从何而来，需要深入 ES 的写入链路。同时，当我们把 `refresh_interval` 从 1s 调大到 30s 乃至 -1 时，系统的行为发生了哪些变化？本文从底层机制出发，逐一拆解。

## 核心层次：从 Index 到 Segment

在进入流程分析之前，先建立层次关系：

```
ES Index（逻辑概念）
  └── Shard（分片，物理单元）
        └── Lucene Index（每个 Shard 底层是一个完整的 Lucene 实例）
              └── Segment（Lucene 最小的不可变存储单元）
```

每个分片背后是一个独立的 Lucene 实例。Lucene 以**不可变 Segment**为存储单元：一旦一个 Segment 被写入，其内容永不修改，删除只是标记，更新是删除旧文档 + 写入新文档。

这个设计带来了一个问题：如果每写入一条文档就生成一个 Segment 并刷盘，磁盘 I/O 开销将极为巨大。于是 ES 引入了**分层的持久化机制**：内存 Buffer → Filesystem Cache → 磁盘，配合 Translog 保证可靠性。

---

## 写入流程：分层持久化机制

### 第一步：请求路由

客户端将写入请求发往集群中的任意节点，该节点成为本次请求的**协调节点（Coordinating Node）**。协调节点根据路由公式计算目标主分片：

```
shard = hash(routing) % number_of_primary_shards
```

`routing` 默认是文档的 `_id`。这也是为什么主分片数量一旦确定便不可修改——改变分片数会导致路由结果变化，已有数据将"找不到"。

> 关于索引分片规划，请参考《Elasticsearch索引分片规划与主分片不可变原理》。

### 第二步：写入内存 Buffer 与 Translog

文档到达主分片所在节点后，**同时**进行两件事：

1. **写入 In-memory Buffer**：将文档暂存在内存缓冲区，此时文档尚不可搜索。
2. **追加到 Translog**：以追加写（Append-only）的方式写入事务日志文件。

Translog 的角色等同于数据库中的 WAL（Write-Ahead Log）。即使节点在 Refresh 之前崩溃，重启后也能通过 Translog 重放恢复内存中尚未持久化的数据。

**Translog 的 fsync 策略**是可配置的：

- `index.translog.durability: request`（默认）：每次写请求完成后都 fsync，保证强一致性。
- `index.translog.durability: async`：按 `sync_interval`（默认 5s）异步 fsync，写入吞吐量更高，但节点崩溃时最多丢失 5s 的数据。

### 第三步：Refresh——让数据可搜索

这是"近实时"延迟的来源。Refresh 过程将 In-memory Buffer 中的文档写入**操作系统的 Filesystem Cache（Page Cache）**，并生成一个新的 Segment。

关键点：**Segment 写入 Page Cache 后即可被搜索**，无需等待 fsync 到磁盘。这就是为什么 ES 是"近实时"而非"实时"——从文档写入到可被搜索，最长需要等待一个 `refresh_interval` 周期。

### 第四步：Flush——从 Page Cache 到磁盘

Flush 完成真正的持久化：将 Segment 从 Page Cache fsync 到磁盘，并清空 Translog。Flush 的触发条件包括 Translog 大小达到阈值（默认 512MB）或定时触发（约 30 分钟）。

### 第五步：副本同步

主分片完成写入后，将操作**并行转发**给所有副本分片。默认情况下，ES 等待所有副本确认后才向客户端返回成功。

> 关于集群状态排查，请参考《Elasticsearch集群黄色红色状态排查与恢复》。

---

## 查询流程：分散与聚合

ES 的搜索查询分为两个阶段：

**Query Phase（分散阶段）**：协调节点将请求广播到所有相关分片，每个分片在本地执行查询，返回 `from + size` 个文档的 doc_id 和 `_score`，不返回完整文档内容。协调节点收到所有分片结果后进行全局排序。

**Fetch Phase（聚合阶段）**：协调节点拿到全局 top-N 的 doc_id 列表后，仅针对这些文档向对应分片发起 multi-get 请求，拉取完整文档内容。

Query Phase 和 Fetch Phase 的分离是 ES 性能设计的关键——将高成本的排序计算和数据回查分开，最大限度减少网络传输量。

**Filter vs Query**：Filter 只判断是否匹配，不计算相关性分数，结果以 Bitset 形式缓存；Query 计算相关性分数，结果不缓存。能用 Filter 的条件，永远不要写在 Query 里。

---

## refresh_interval 深度分析

### 默认值 1s 的设计权衡

1 秒的默认值是 ES 在**数据可见延迟**与**写入吞吐量**之间做出的折中：对大多数搜索场景，1 秒的延迟是可以接受的；对写入而言，每秒产生一个 Segment 不会造成过大的积压。

### 调大 refresh_interval 的影响

**正面影响**：减少 Segment 生成频率，降低 Merge 触发频率，写入吞吐量提升。

**负面影响**：数据可见延迟增加；内存 Buffer 积累更多文档，极端情况下可能触发强制 Refresh。

### 不同场景的推荐配置

| 场景 | refresh_interval | 说明 |
|------|-----------------|------|
| 日志写入（高吞吐，允许延迟） | 30s ～ 60s | 写入优先，牺牲可见延迟 |
| 搜索场景（低延迟可见性） | 1s（默认） | 近实时搜索 |
| 批量数据导入 | -1 | 导入期间禁用，完成后手动 refresh |
| 实时报警/监控 | 5s ～ 10s | 在延迟与吞吐之间取中间值 |

### 与 Translog 配置联动调优

单独调大 `refresh_interval` 只解决了 Segment 生成频率问题，真正的写入性能调优需要将 `refresh_interval` 与 Translog 配置一起考虑：

```json
PUT /logs-index/_settings
{
  "index.refresh_interval": "30s",
  "index.translog.durability": "async",
  "index.translog.sync_interval": "30s",
  "index.translog.flush_threshold_size": "1gb"
}
```

需要权衡的是：`async` 模式下，节点崩溃最多丢失一个 `sync_interval` 内的数据。对日志场景通常可以接受，对金融交易数据则不可取。

---

## 生产调优实践

### 批量导入的标准流程

```json
// 1. 写入前：禁用自动 Refresh，副本数设为 0
PUT /my_index/_settings
{
  "index.refresh_interval": "-1",
  "index.number_of_replicas": 0
}

// 2. 使用 Bulk API 批量写入

// 3. 写入后：恢复设置并手动 Refresh
PUT /my_index/_settings
{
  "index.refresh_interval": "1s",
  "index.number_of_replicas": 1
}
POST /my_index/_refresh
```

### 写入性能关键参数汇总

| 参数 | 默认值 | 调优方向 | 说明 |
|------|--------|---------|------|
| `refresh_interval` | 1s | 调大（30s ～ 60s） | 减少 Segment 碎片 |
| `translog.durability` | request | 改为 async | 减少 fsync 开销 |
| `translog.flush_threshold_size` | 512mb | 调大（1gb） | 降低 Flush 频率 |
| `number_of_replicas` | 1 | 批量写入时设为 0 | 减少写入放大 |
| Bulk 批次大小 | — | 5MB ～ 15MB/批 | 过小浪费网络，过大占用内存 |

### 查询性能关键参数汇总

| 优化手段 | 原理 | 注意事项 |
|---------|------|---------|
| 使用 Filter 替代 Query | Filter 结果可缓存 | 需要计算相关性分数时仍需 Query |
| 使用 routing 定向查询 | 将查询路由到特定分片 | 适合有明确业务维度分区的数据 |
| 减少分片数 | 降低协调节点汇总开销 | 单分片不宜超过 50GB |
| forcemerge 静态索引 | 合并为 1 个 Segment | 只对不再写入的冷索引执行 |

---

## 小结

- ES 写入链路的核心是**分层持久化**：Buffer → Filesystem Cache → 磁盘，Translog 提供崩溃恢复保障。
- **Refresh 是"近实时"的实现机制**，每次 Refresh 将 Buffer 写入 Page Cache 并生成新 Segment。
- 查询分为 **Query Phase** 和 **Fetch Phase** 两阶段，Filter 缓存命中率是影响查询性能的关键。
- `refresh_interval` 是写入调优的核心旋钮：调大可提升写入吞吐、减少 Merge 压力，代价是数据可见延迟增加。

---

## 常见问题

### Q1：为什么 refresh 后数据仍然不可见？

常见原因：写入时未指定 `refresh` 参数（默认行为），Refresh 周期未到；查询命中了尚未 Refresh 完成的副本；或使用 `_id` 直接 GET（GET by ID 会直接读 Translog，即使未 Refresh 也能读到）。

### Q2：Translog 和 Flush 的关系，节点崩溃时数据会丢失吗？

数据是否丢失取决于 Translog 的 fsync 策略：`durability: request`（默认）每次写请求都 fsync，不丢数据；`durability: async` 异步 fsync，崩溃时最多丢失一个 `sync_interval` 内的数据。Flush 将 Segment 写入磁盘并清空 Translog，Flush 频率影响 Translog 大小和重启恢复时间。

### Q3：调大 refresh_interval 后，索引的 Segment 数会立刻减少吗？

不会。已有的 Segment 不会消失，只是不再频繁产生新的小 Segment。Segment 数量的减少依赖后台 **Merge** 操作。对不再写入的索引，可手动执行 `POST /my_index/_forcemerge?max_num_segments=1`。

### Q4：分片数和 refresh_interval 如何协同影响写入性能？

分片数越多，每个分片都有独立的 Buffer、Translog 和 Segment 管理开销。在分片数较多的情况下，如果 `refresh_interval` 仍为 1s，每秒会产生大量新 Segment，Merge 压力倍增。**分片数越多，调大 `refresh_interval` 的收益越显著**。

### Q5：Segment Merge 是什么？为什么调大 refresh_interval 能减少 Merge 压力？

Segment Merge 是 Lucene 后台自动执行的数据整理过程，将多个小 Segment 合并为较大的 Segment。每次 Refresh 都会产生一个新 Segment，如果 `refresh_interval` 很小（如 1s），每秒都会产生新 Segment，导致 Segment 数量快速增长。过多的 Segment 会增加搜索时的 I/O 开销和内存占用，Merge 需要频繁执行来合并这些碎片。调大 `refresh_interval` 后，每次 Refresh 处理更多文档，产生的 Segment 更大但数量更少，从而降低 Merge 的触发频率和系统开销。

## 参考资源

- [Elasticsearch 写入性能调优](https://www.elastic.co/guide/en/elasticsearch/reference/current/tune-for-indexing-speed.html)
- [Refresh 机制官方文档](https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-refresh.html)

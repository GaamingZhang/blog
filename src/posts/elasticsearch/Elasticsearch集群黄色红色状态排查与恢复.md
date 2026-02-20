---
date: 2026-02-12
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Elasticsearch
tag:
  - Elasticsearch
  - ClaudeCode
---

# Elasticsearch 集群黄色/红色状态排查与恢复

## 状态灯变色，你第一反应是什么

凌晨三点，监控告警炸了。Kibana 的集群健康仪表盘从绿色变成了红色，业务方开始反馈搜索请求报错。你盯着屏幕，该从哪里下手？

ES 集群健康状态不是一个单纯的"好/坏"二元判断，而是包含了分片分配状态的精确诊断信号。读懂它，才能把排查时间从两小时压缩到二十分钟。

---

## 集群健康状态的本质

### 三种状态的精确定义

ES 集群的健康状态由分片的分配情况决定，而非节点数量或查询是否正常。

```
green  → 所有主分片 + 所有副本分片均已分配
yellow → 所有主分片已分配，但至少一个副本分片未分配
red    → 至少一个主分片未分配（该分片的数据当前不可访问）
```

这个定义有一个重要推论：**集群整体状态是所有索引中最差状态的反映**。100 个索引中有 99 个是 green，只要 1 个是 red，集群状态就是 red。

### `_cluster/health` API 关键字段解读

```bash
GET /_cluster/health
```

```json
{
  "cluster_name": "prod-es",
  "status": "yellow",
  "timed_out": false,
  "number_of_nodes": 3,
  "number_of_data_nodes": 3,
  "active_primary_shards": 45,
  "active_shards": 80,
  "relocating_shards": 0,
  "initializing_shards": 2,
  "unassigned_shards": 10,
  "delayed_unassigned_shards": 0,
  "number_of_pending_tasks": 0,
  "active_shards_percent_as_number": 88.88
}
```

| 字段 | 含义 | 排查价值 |
|------|------|---------|
| `unassigned_shards` | 未分配分片总数 | 非零时开始排查 |
| `initializing_shards` | 正在初始化的分片 | 节点刚加入或恢复中，可短暂等待 |
| `relocating_shards` | 正在迁移的分片 | 数值稳定减少是正常现象 |
| `delayed_unassigned_shards` | 延迟分配的分片 | 受 `index.unassigned.node_left.delayed_timeout` 控制 |
| `active_shards_percent_as_number` | 活跃分片占比 | 低于 100% 即有问题 |

`delayed_unassigned_shards` 是一个经常被忽视的字段。当节点离开集群时，ES 默认等待 1 分钟（可配置）才开始重新分配分片，这段时间内未分配的分片计入 `delayed_unassigned_shards` 而非 `unassigned_shards`。如果节点很快重新加入，这些分片会直接恢复，避免了不必要的数据迁移。

---

## Yellow 状态：副本分片未分配

Yellow 状态说明数据是完整可用的（主分片都在），但容灾能力下降。不紧急，但需要尽快处理。

### 常见原因

**原因一：节点数不够**

这是最常见的原因。一个索引设置了 `number_of_replicas: 1`，意味着每个主分片需要一个副本放在**不同节点**上。如果只有 1 个节点，副本没有地方放，永远是 yellow。

```
索引配置：1 个主分片 + 1 个副本
节点数：1
结果：副本永远无法分配 → yellow
```

**原因二：磁盘水位线触发**

节点磁盘使用率超过 `high watermark`（默认 85%）后，ES 停止将新分片（包括副本）分配到该节点。

**原因三：分片路由被禁用**

```bash
GET /_cluster/settings
```

如果看到 `cluster.routing.allocation.enable: none` 或 `primaries`，副本分配被人工禁用了。

### 排查步骤

**第一步：定位哪些索引是 yellow**

```bash
GET /_cluster/health?level=indices
```

输出按索引列出状态，快速找到问题索引。

**第二步：查看未分配分片的具体状态**

```bash
GET /_cat/shards?v&h=index,shard,prirep,state,unassigned.reason,node
```

```
index        shard prirep state      unassigned.reason node
my-logs-2026 0     r      UNASSIGNED NODE_LEFT         -
my-logs-2026 1     r      UNASSIGNED ALLOCATION_FAILED -
```

`prirep` 字段：`p` = primary，`r` = replica。这里两个副本分片未分配，原因分别是节点离开和分配失败。

**第三步：获取分配失败的详细原因**

```bash
GET /_cluster/allocation/explain
```

不带参数时，ES 会自动选择一个未分配分片进行解释。如果想指定：

```json
GET /_cluster/allocation/explain
{
  "index": "my-logs-2026",
  "shard": 1,
  "primary": false
}
```

返回结果会明确告诉你分配失败的原因，是最重要的诊断工具：

```json
{
  "index": "my-logs-2026",
  "shard": 1,
  "primary": false,
  "current_state": "unassigned",
  "unassigned_info": {
    "reason": "ALLOCATION_FAILED",
    "last_allocation_status": "no_valid_shard_copy"
  },
  "can_allocate": "no",
  "allocate_explanation": "cannot allocate because a previous attempt failed",
  "node_allocation_decisions": [
    {
      "node_name": "node-1",
      "deciders": [
        {
          "decider": "disk_threshold",
          "decision": "NO",
          "explanation": "the node is above the high watermark ... 87.2% used"
        }
      ]
    }
  ]
}
```

`deciders` 列表里每一项都是一个分配决策器，`NO` 表示该决策器否决了分配，`explanation` 给出具体原因。

### Yellow 修复方案

| 原因 | 修复方案 | 命令 |
|------|---------|------|
| 节点数不足 | 增加节点，或临时调低副本数 | `PUT /my-index/_settings {"number_of_replicas": 0}` |
| 磁盘水位线触发 | 清理磁盘空间，或临时调整水位线 | 见磁盘水位线章节 |
| 路由被禁用 | 恢复路由分配 | `PUT /_cluster/settings {"transient": {"cluster.routing.allocation.enable": "all"}}` |
| 分配多次失败 | 重试分配 | `POST /_cluster/reroute?retry_failed=true` |

:::tip 单节点开发环境
在只有 1 个节点的环境（如本地开发、测试）中，yellow 是预期行为。可以通过 `PUT /_settings {"number_of_replicas": 0}` 将所有索引副本数设为 0，集群恢复 green。
:::

---

## Red 状态：主分片未分配

Red 状态意味着数据不可访问，是需要立刻处理的紧急情况。对 red 分片的任何读写请求都会返回错误。

### 常见原因

- 数据节点宕机，且该节点上的分片没有副本（或副本也丢失了）
- 节点磁盘目录损坏（Lucene index corruption）
- 索引快照恢复失败或中途中断
- Allocation Filter 配置错误，导致主分片无处可分配
- 集群重启后，主节点选举超时，Gateway 恢复阶段异常

### 排查步骤

**第一步：确认哪些索引是 red**

```bash
GET /_cat/indices?v&health=red
```

```
health status index        pri rep docs.count store.size
red    open   orders-2025  5   1   1250000    12.3gb
```

**第二步：确认未分配的是主分片还是副本**

```bash
GET /_cat/shards?v&h=index,shard,prirep,state,unassigned.reason&s=state
```

找到 `prirep=p` 且 `state=UNASSIGNED` 的行，这些才是真正导致 red 的主分片。

**第三步：获取主分片的分配解释**

```json
GET /_cluster/allocation/explain
{
  "index": "orders-2025",
  "shard": 0,
  "primary": true
}
```

这是最关键的一步，结果会告诉你：分片曾经存在于哪些节点，现在这些节点是否在线，数据是否还存在。

**第四步：检查节点状态**

```bash
GET /_cat/nodes?v&h=name,ip,heap.percent,disk.used_percent,node.role,master
```

确认是否有节点下线，数据节点数量是否符合预期。

**第五步：检查各节点磁盘情况**

```bash
GET /_cat/allocation?v
```

```
shards disk.indices disk.used disk.avail disk.total disk.percent node
    45       23.1gb    85.3gb     14.7gb    100.0gb           85 node-1
    45       23.1gb    91.2gb      8.8gb    100.0gb           91 node-2
```

`disk.percent > 85%` 意味着高水位线已触发，`> 95%` 意味着 flood_stage 已触发（索引变只读）。

### Red 修复方案矩阵

| 原因 | 修复思路 | 备注 |
|------|---------|------|
| 节点宕机，副本存活 | 等待或手动触发副本晋升为主分片 | ES 会自动处理，通常无需干预 |
| 节点宕机，无副本 | 从快照恢复，或 `allocate_stale_primary`（有数据丢失风险） | 最后手段 |
| Allocation Filter 配置错误 | 检查并修正 `index.routing.allocation` 相关设置 | 见 Allocation Filtering 章节 |
| 磁盘空间不足 | 清理空间或扩容磁盘 | 优先处理 flood_stage |
| 节点临时重启 | 等待节点重新加入集群 | 配合 `delayed_timeout` 使用 |

:::danger Red 状态的紧迫性
Red 状态下，对应索引的所有写入请求会返回 `ClusterBlockException`，读取请求返回部分结果或报错。如果是业务核心索引，需要立刻介入，而不是等到下班后处理。
:::

---

## unassigned.reason 详解

`_cat/shards` 中 `unassigned.reason` 字段记录了分片**最初**变为未分配的原因（注意：不一定是当前无法分配的原因，当前原因要看 `allocation/explain`）。

| reason 值 | 含义 | 处理策略 |
|-----------|------|---------|
| `NODE_LEFT` | 承载该分片的节点离开集群 | 等待节点重新加入；若节点永久下线则需从快照恢复或重新分配 |
| `ALLOCATION_FAILED` | 分配尝试失败（如磁盘不足、Filter 不满足） | 执行 `reroute?retry_failed=true`，同时修复根本原因 |
| `INDEX_CREATED` | 索引刚创建，分片等待首次分配 | 正常状态，集群资源足够时会自动分配 |
| `CLUSTER_RECOVERED` | 集群完整重启后的 Gateway 恢复阶段 | 正常现象，等待恢复完成；若长时间不恢复检查节点数是否达到法定人数 |
| `REPLICA_ADDED` | 副本数量被调高，新副本等待分配 | 确保有足够节点和磁盘空间 |
| `DANGLING_INDEX_IMPORTED` | 发现孤立索引数据（节点曾属于另一个集群）并导入 | 审查此索引是否应该存在，不需要可删除 |
| `REINITIALIZED` | 分片被强制重新初始化（通常是 reroute 操作后） | 等待初始化完成 |

---

## 磁盘水位线：最常见的隐性故障

磁盘水位线是 ES 分片分配中最容易被忽视的约束，也是生产环境 yellow/red 状态的高频原因。

### 三条水位线的含义

```
低水位线 (low watermark)     默认 85%
  └── 触发后：不再将新分片分配到该节点（已有分片不受影响）

高水位线 (high watermark)    默认 90%
  └── 触发后：尝试将该节点上的分片迁移到其他节点

洪水位线 (flood_stage)       默认 95%
  └── 触发后：将该节点上的所有索引设置为只读（read_only_allow_delete）
```

### 触发洪水位线后的症状

所有写入请求返回：

```json
{
  "error": {
    "type": "cluster_block_exception",
    "reason": "index [orders-2025] blocked by: [TOO_MANY_REQUESTS/12/disk usage exceeded flood-stage watermark, index has read-only-allow-delete block]"
  }
}
```

此时即使清理了磁盘空间，索引的只读 block 不会自动解除，需要手动操作。

### 修复流程

**第一步：确认磁盘用量和 block 状态**

```bash
GET /_cat/allocation?v
GET /_cat/indices?v&h=index,blocks
```

**第二步：临时调整水位线（争取时间清理空间）**

```bash
PUT /_cluster/settings
{
  "transient": {
    "cluster.routing.allocation.disk.watermark.low": "93%",
    "cluster.routing.allocation.disk.watermark.high": "95%",
    "cluster.routing.allocation.disk.watermark.flood_stage": "97%"
  }
}
```

**第三步：清理磁盘空间后解除索引只读 block**

```bash
PUT /orders-2025/_settings
{
  "index.blocks.read_only_allow_delete": null
}
```

或一次性解除所有索引的 block：

```bash
PUT /_settings
{
  "index.blocks.read_only_allow_delete": null
}
```

**第四步：将水位线恢复为默认值**

```bash
PUT /_cluster/settings
{
  "transient": {
    "cluster.routing.allocation.disk.watermark.low": null,
    "cluster.routing.allocation.disk.watermark.high": null,
    "cluster.routing.allocation.disk.watermark.flood_stage": null
  }
}
```

:::warning 设置 null 的含义
在 ES 集群设置中，将某个配置项设置为 `null` 等于删除该 transient 设置，使其恢复到 persistent 设置或默认值。这是正确的"恢复默认值"方式，而不是设置字符串 `"null"`。
:::

---

## 强制重新分配：reroute API

当 ES 的自动分配机制无法处理时，可以使用 reroute API 进行手动干预。

### 重试失败的分配

这是最安全的操作，应该优先尝试：

```bash
POST /_cluster/reroute?retry_failed=true
```

ES 会重新尝试所有因暂时性原因（如短暂的磁盘不足或网络抖动）失败的分片分配。

### 手动移动分片

将分片从一个节点迁移到另一个节点：

```json
POST /_cluster/reroute
{
  "commands": [
    {
      "move": {
        "index": "orders-2025",
        "shard": 0,
        "from_node": "node-1",
        "to_node": "node-2"
      }
    }
  ]
}
```

### 强制分配副本分片

当副本分片卡在 UNASSIGNED 状态时：

```json
POST /_cluster/reroute
{
  "commands": [
    {
      "allocate_replica": {
        "index": "orders-2025",
        "shard": 0,
        "node": "node-2"
      }
    }
  ]
}
```

### 强制分配过期主分片（高风险）

```json
POST /_cluster/reroute
{
  "commands": [
    {
      "allocate_stale_primary": {
        "index": "orders-2025",
        "shard": 0,
        "node": "node-1",
        "accept_data_loss": true
      }
    }
  ]
}
```

:::danger 谨慎使用 allocate_stale_primary
`allocate_stale_primary` 用于将一个"过期的"分片副本强制提升为主分片，会**不可逆地丢失数据**（因为该副本可能不是最新的）。`accept_data_loss: true` 不是一个可以随便填写的参数，它意味着你明确知道并接受将要发生的数据丢失。只有在快照也不可用的极端情况下，才考虑使用此命令。
:::

---

## Allocation Filtering 配置排查

Allocation Filter 允许精细控制分片的分配规则，但配置不当是生产环境 yellow/red 的高频原因。

### 三种过滤规则

```bash
# 要求分片必须分配到满足条件的节点（白名单，严格匹配）
PUT /my-index/_settings
{"index.routing.allocation.require.zone": "zone-a"}

# 允许分片分配到满足条件的节点（白名单，至少满足一个）
PUT /my-index/_settings
{"index.routing.allocation.include.zone": "zone-a,zone-b"}

# 禁止分片分配到满足条件的节点（黑名单）
PUT /my-index/_settings
{"index.routing.allocation.exclude.zone": "zone-c"}
```

这些过滤器基于节点的 **自定义属性**（通过 `elasticsearch.yml` 中的 `node.attr.*` 设置）或内置属性（`_name`、`_ip`、`_host`）。

### 排查方法

**问题场景**：执行 `allocation/explain` 后看到如下输出：

```json
{
  "decider": "filter",
  "decision": "NO",
  "explanation": "node does not match index setting [index.routing.allocation.require] filters [zone:\"zone-a\"]"
}
```

这意味着索引要求分片必须在 `zone=zone-a` 的节点上，但集群中没有带此标签的节点（或所有此类节点已满）。

**排查步骤**：

```bash
# 查看索引当前的路由设置
GET /my-index/_settings?filter_path=*.settings.index.routing

# 查看所有节点的属性标签
GET /_cat/nodeattrs?v
```

**修复**：要么修改索引的 routing filter，要么为目标节点添加正确的属性标签（需修改 `elasticsearch.yml` 并重启节点）。

### Awareness 配置排查

Shard Allocation Awareness 让 ES 感知集群的物理拓扑（机架、可用区），避免将主分片和其副本放在同一个物理故障域。

```yaml
# elasticsearch.yml
cluster.routing.allocation.awareness.attributes: rack_id
```

如果配置了 awareness，但某个 `rack_id` 值下的节点全部宕机，那么原本在那些节点上的副本分片将无法被迁移到同一 rack 的其他节点，集群变为 yellow 甚至 red。

```bash
# 检查 awareness 配置
GET /_cluster/settings?include_defaults=true&filter_path=*.cluster.routing.allocation.awareness
```

---

## 集群恢复速度优化

发现并修复了问题原因后，ES 开始恢复分片。恢复速度受几个参数控制，在紧急情况下可以调高以加速恢复，恢复完成后应调回正常值，避免长期占用过多带宽和 I/O。

### 关键参数

```bash
PUT /_cluster/settings
{
  "transient": {
    "cluster.routing.allocation.node_concurrent_recoveries": 4,
    "cluster.routing.allocation.node_initial_primaries_recoveries": 8,
    "indices.recovery.max_bytes_per_sec": "200mb"
  }
}
```

| 参数 | 默认值 | 作用 |
|------|-------|------|
| `node_concurrent_recoveries` | 2 | 每个节点同时进行的分片恢复数量 |
| `node_initial_primaries_recoveries` | 4 | 集群重启时，主分片从本地恢复的并发数 |
| `indices.recovery.max_bytes_per_sec` | 40mb | 分片恢复的网络带宽上限 |

### 监控恢复进度

```bash
GET /_cat/recovery?v&active_only=true&h=index,shard,stage,time,type,source_node,target_node,bytes_percent,files_percent
```

```
index        shard stage   time  type  source_node target_node bytes_percent files_percent
orders-2025  0     index   45s   peer  node-1      node-3      67.3%         71.2%
orders-2025  1     translog 12s  peer  node-2      node-3      100.0%        100.0%
```

`stage` 值说明：
- `init`：初始化
- `index`：传输 Lucene 文件
- `verify_index`：校验数据完整性
- `translog`：重放 translog（确保数据最新）
- `finalize`：收尾工作
- `done`：完成

---

## 预防措施与监控告警

### 关键 Prometheus 告警规则

```yaml
groups:
  - name: elasticsearch_cluster
    rules:
      - alert: ElasticsearchClusterNotGreen
        expr: elasticsearch_cluster_health_status{color="green"} == 0
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "ES 集群非 green 状态"
          description: "集群 {{ $labels.cluster }} 持续 2 分钟处于非 green 状态"

      - alert: ElasticsearchClusterRed
        expr: elasticsearch_cluster_health_status{color="red"} == 1
        for: 0m
        labels:
          severity: critical
        annotations:
          summary: "ES 集群 RED 状态，数据不可访问"
          description: "集群 {{ $labels.cluster }} 处于 red 状态，请立即排查"

      - alert: ElasticsearchUnassignedShards
        expr: elasticsearch_cluster_health_unassigned_shards > 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "ES 存在未分配分片"
          description: "集群 {{ $labels.cluster }} 有 {{ $value }} 个分片持续 5 分钟未分配"

      - alert: ElasticsearchDiskHigh
        expr: elasticsearch_filesystem_data_available_bytes / elasticsearch_filesystem_data_size_bytes < 0.15
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "ES 节点磁盘空间不足"
          description: "节点 {{ $labels.node }} 磁盘剩余空间低于 15%"
```

### 日常巡检命令清单

```bash
# 1. 集群整体健康状态
GET /_cluster/health

# 2. 节点状态和资源使用
GET /_cat/nodes?v&h=name,heap.percent,disk.used_percent,cpu,load_1m,node.role

# 3. 未分配分片一览
GET /_cat/shards?v&h=index,shard,prirep,state,unassigned.reason&s=state

# 4. 各节点磁盘使用
GET /_cat/allocation?v

# 5. 索引级别健康状态（快速发现问题索引）
GET /_cat/indices?v&health=yellow,red&s=health

# 6. 当前正在恢复的分片
GET /_cat/recovery?v&active_only=true
```

---

## 排查流程总览

```
集群状态变为 yellow 或 red
          │
          ↓
GET /_cluster/health  →  查看 unassigned_shards 数量
          │
          ↓
GET /_cat/shards?v&h=index,shard,prirep,state,unassigned.reason
          │
    ┌─────┴─────┐
  prirep=r    prirep=p
  (副本)      (主分片)
    │             │
  yellow        red ← 立刻处理！
    │             │
    └──────┬──────┘
           ↓
GET /_cluster/allocation/explain
           │
    ┌──────┼──────┬──────────┐
    │      │      │          │
 磁盘   节点   Filter    分配
 超限  下线   配置错误  失败重试
    │      │      │          │
    ↓      ↓      ↓          ↓
清理  等待  修正   reroute
磁盘  节点  Filter ?retry_
解除  恢复  配置   failed=true
block
    │      │      │          │
    └──────┴──────┴──────────┘
                  │
                  ↓
       GET /_cluster/health  →  验证恢复 green
```

---

## 小结

- 集群健康状态本质上是分片分配状态的映射：green = 全部分配，yellow = 副本缺失，red = 主分片缺失。
- `_cluster/allocation/explain` 是排查分片未分配原因的核心工具，不要跳过它直接猜原因。
- 磁盘水位线（特别是 flood_stage）是生产环境黄/红状态的高频原因，触发后除了扩容还需要手动解除索引的 read_only block。
- `allocate_stale_primary` 是最后手段，一旦执行将不可逆地接受数据丢失，使用前必须确认快照恢复已经不可行。
- 集群恢复速度可以通过调整 `node_concurrent_recoveries` 和 `indices.recovery.max_bytes_per_sec` 来加速，但要在恢复后恢复默认值。
- 预防优于治疗：配置 Prometheus 告警，在 unassigned_shards > 0 持续 5 分钟时发出预警，在 red 状态时立刻触发告警。

---

## 常见问题

### Q1：集群变 yellow 后，查询是否受影响？

yellow 状态下查询**不受影响**。因为 yellow 表示所有主分片都已分配并可访问，只是副本分片缺失。副本分片的主要作用是容灾（主分片宕机时副本可以接管）和分担查询压力（ES 会将查询路由到主分片或副本分片）。副本缺失意味着：查询压力全部落在主分片上（吞吐量可能下降），以及如果此时主分片所在节点宕机，集群会变 red。所以 yellow 是"有隐患但当前可用"的状态，需要处理但不是紧急故障。

### Q2：节点重启后集群变 red，等多久才会自动恢复？

这取决于 `index.unassigned.node_left.delayed_timeout`（默认 1 分钟）和数据量大小。节点离开后，ES 会先等待 delayed_timeout，观察节点是否会重新加入。如果节点在此时间内重新加入，分片直接恢复，通常几分钟内完成。如果节点超时未返回，ES 会将其他节点上的副本提升为主分片（若有副本），集群从 red 变 yellow，再进行副本的重新分配。整个过程少则几分钟，多则数十分钟，取决于数据量和 `indices.recovery.max_bytes_per_sec` 的配置。可以通过 `GET /_cat/recovery?v&active_only=true` 监控恢复进度。

### Q3：如何区分分片分配的"暂时失败"和"永久失败"？

`_cluster/allocation/explain` 的输出是关键。看 `can_allocate` 字段：值为 `yes` 表示当前可以分配（只是还没分配）；值为 `no` 表示有决策器明确阻止了分配；值为`awaiting_info` 表示正在等待节点信息。进一步看 `node_allocation_decisions` 中每个节点的 `deciders` 列表：`disk_threshold`（磁盘超限）、`filter`（路由过滤器不满足）、`same_shard`（主副本在同一节点）是可以修复的条件；`no_valid_shard_copy` 则意味着没有找到有效的分片数据，这种情况需要从快照恢复或考虑 `allocate_stale_primary`。

### Q4：`reroute?retry_failed=true` 和手动 `allocate_replica` 有什么区别？

`reroute?retry_failed=true` 是批量重试所有因暂时性原因失败的分片分配，它不会改变任何分配规则，只是让 ES 再次尝试之前失败的分配任务。适用于：短暂的磁盘抖动或网络超时导致分配失败，根本原因已经解决，只需要触发一次重试。`allocate_replica` 是强制指定将某个副本分配到特定节点，会绕过部分分配规则（但仍然尊重硬性限制如磁盘水位线）。适用于：ES 的自动分配反复失败，你明确知道应该将分片放在哪个节点，并且该节点条件满足。优先用 `retry_failed`，只有在 retry 无效时才使用 `allocate_replica`。

### Q5：磁盘水位线的百分比和字节数哪个优先级更高？

ES 的磁盘水位线支持两种设置方式：百分比（如 `"85%"`）和绝对剩余字节数（如 `"15gb"`）。它们不是互斥的——你可以同时指定两种方式，ES 会取**更严格**的那个作为实际水位线。例如，设置 `low: "85%"` 且磁盘只有 20GB，那么低水位线是 17GB（85% × 20GB），即剩余 3GB 时触发；如果同时设置了 `low: "5gb"`，则剩余 5GB 时就触发，以更严格的 5GB 为准。在实际生产中，推荐为大容量磁盘（如 2TB 以上）设置绝对字节数而非百分比，避免 95% 时仍有 100GB 可用空间却触发 flood_stage 的尴尬。

---

## 参考资源

- [Elasticsearch 集群健康官方文档](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-health.html)
- [分片分配解释 API](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-allocation-explain.html)
- [Elasticsearch 故障排查指南](https://www.elastic.co/guide/en/elasticsearch/reference/current/troubleshooting.html)

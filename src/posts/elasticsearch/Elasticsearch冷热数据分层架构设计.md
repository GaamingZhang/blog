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

# Elasticsearch 冷热数据分层架构设计

## 问题的起点

设想一个日志平台的典型访问模式：过去 24 小时的日志几乎每隔几秒就有人查询，上周的日志偶尔翻一翻，三个月前的日志只有在排查历史问题时才会被翻出来，而半年前的数据几乎从未被访问。

如果所有数据都存在同一批高性能 NVMe SSD 节点上，这意味着你用最昂贵的硬件来存储几乎不被读取的冷数据。数据量越大，浪费越明显——一个日均写入 100GB 的平台，半年的数据就是 18TB，但真正频繁被查询的可能只有最近 7 天的 700GB。

冷热分层要解决的核心问题是：**在保证各层数据可查询的前提下，让不同访问频率的数据住在与其价值匹配的硬件上**，大幅降低整体存储成本，同时不影响热数据的查询性能。

本文从节点角色设计、ILM 自动化策略、Searchable Snapshot 到完整生产配置，系统梳理 ES 8.x 冷热分层架构的实现原理与实践细节。

---

## 节点角色体系

ES 8.x 通过 `node.roles` 将节点分为明确的数据层角色。理解各层角色的设计意图，是规划集群硬件和分片迁移策略的前提。

### 四层节点角色

```
写入请求
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  data_hot   │ 高 CPU + 大内存 + NVMe SSD    │ 写入 + 实时查询 │
├─────────────────────────────────────────────────────────────┤
│  data_warm  │ 中等内存 + SATA SSD 或 HDD   │ 近期只读查询    │
├─────────────────────────────────────────────────────────────┤
│  data_cold  │ 低内存 + 大容量 HDD           │ 历史只读查询    │
├─────────────────────────────────────────────────────────────┤
│  data_frozen│ 极低内存 + 对象存储（S3/MinIO）│ 归档数据按需加载│
└─────────────────────────────────────────────────────────────┘
```

**Hot 层**是写入的唯一入口，也是高频查询的主战场。这一层要求低延迟的随机写入性能，因此 NVMe SSD 是标配。内存需要足够大，以便将 Segment 文件充分缓存在 OS Page Cache 中，减少磁盘 I/O。

**Warm 层**只接受迁入的只读数据，不承接写入流量。硬件要求相比 Hot 层明显降低，SATA SSD 或高转速 HDD 均可接受，内存配置可减半。这一层通常存放最近 8～30 天的数据。

**Cold 层**进一步降低硬件规格，使用大容量 HDD，配置较小的内存。这一层对查询延迟不敏感——偶发的历史查询，即便需要几秒钟也是可以接受的。

**Frozen 层**是成本最极致的一层，本质上数据存储在 S3、MinIO 等对象存储中，本地节点只保留一个共享缓存（Shared Cache），数据按需从对象存储拉取，查询结束后缓存可被驱逐。这一层的节点几乎可以用最低配的机器。

### elasticsearch.yml 角色配置

每台节点的 `elasticsearch.yml` 中，通过 `node.roles` 声明该节点属于哪一层：

```yaml
# Hot 节点
node.roles: [ data_hot, ingest ]

# Warm 节点
node.roles: [ data_warm ]

# Cold 节点
node.roles: [ data_cold ]

# Frozen 节点
node.roles: [ data_frozen ]

# Master 节点（生产环境建议独立部署，不混合数据角色）
node.roles: [ master ]
```

### 分片分配：tier_preference 机制

节点角色只是声明了节点"属于哪一层"，分片实际落到哪一层，由索引的 `_tier_preference` 路由设置控制。

ES 内置的 ILM（下文详述）在迁移分片时，会自动更新这一设置：

```json
PUT /logs-app-000001/_settings
{
  "index.routing.allocation.require._tier_preference": "data_warm,data_hot"
}
```

上述配置表示：优先将分片分配到 Warm 层节点；若 Warm 层无可用节点，则回退到 Hot 层。这一机制保证了即使某一层节点临时缩容，分片也不会变为 unassigned，集群仍然可用。

---

## ILM：分层自动化的核心机制

手动迁移索引既繁琐又容易出错，ILM（Index Lifecycle Management）是 ES 提供的索引全生命周期自动化引擎，负责将上述分层设计付诸实践。

ILM 将索引生命周期抽象为四个阶段：**Hot → Warm → Cold → Delete**，每个阶段可以配置一系列动作（Action），由 ILM 后台定期轮询（默认每 10 分钟）执行状态转换。

### Hot 阶段：写入与滚动

Hot 阶段是数据写入的起点，核心动作是 **Rollover**——当索引满足特定条件时，自动创建新的写入索引，并将旧索引"封存"以备后续迁移。

Rollover 的触发条件可以同时设置多个，满足任意一个即触发：

```json
"hot": {
  "actions": {
    "rollover": {
      "max_age": "7d",
      "max_primary_shard_size": "50gb",
      "max_docs": 200000000
    },
    "forcemerge": {
      "max_num_segments": 1
    },
    "set_priority": {
      "priority": 100
    }
  }
}
```

`forcemerge` 在 Rollover 后将索引强制合并为单个 Segment，理由是已封存的索引不再写入，此时合并到 1 个 Segment 可以最大化查询性能，同时消除已标记删除文档的空间浪费。这个操作会消耗大量 I/O，因此通常配置 `"index.forcemerge.after_expiry": true`，让它在非高峰期异步完成。

### Warm 阶段：迁移与压缩

Rollover 发生后，旧索引在 Hot 阶段继续等待 `min_age`（相对于 Rollover 时间）条件满足，随后进入 Warm 阶段。

```json
"warm": {
  "min_age": "1d",
  "actions": {
    "allocate": {
      "require": {
        "_tier_preference": "data_warm"
      }
    },
    "shrink": {
      "number_of_shards": 1
    },
    "readonly": {},
    "set_priority": {
      "priority": 50
    }
  }
}
```

**Shrink**（收缩）是 Warm 阶段最重要的动作。索引在 Hot 阶段通常配置多个主分片（如 3 个）以支撑写入并发，但进入 Warm 阶段后只需读取，过多的分片带来的是额外的元数据开销和查询协调成本。Shrink 将主分片数合并为 1，本质上是创建一个新的单分片索引，将原索引的所有段硬链接过去，因此几乎不产生额外磁盘 I/O。

**Shrink 的前提条件**是原索引的所有主分片必须先分配到同一个节点，否则操作会失败（这是生产中最常见的踩坑点，后文详述）。

**readonly** 动作将索引设置为 `index.blocks.write: true`，防止意外写入。

### Cold 阶段：归档与降本

Cold 阶段进一步降低存储成本，此时数据已经很少被查询，可以做更激进的优化。

```json
"cold": {
  "min_age": "30d",
  "actions": {
    "allocate": {
      "require": {
        "_tier_preference": "data_cold"
      }
    },
    "freeze": {},
    "set_priority": {
      "priority": 0
    }
  }
}
```

`freeze` 动作（ES 7.x 引入，8.x 中语义有所调整）会释放索引在内存中的数据结构（Field Data Cache 等），仅在查询时按需加载，极大降低内存占用。Cold 层节点可以用较小内存承载大量归档索引。

在 ES 8.x 中，更推荐的做法是在 Cold 阶段直接启用 **Searchable Snapshot**（后文详述），将数据迁移到对象存储，彻底释放本地磁盘空间。

### Delete 阶段：自动清理

```json
"delete": {
  "min_age": "180d",
  "actions": {
    "delete": {}
  }
}
```

超过保留期的索引由 ILM 自动删除，无需人工干预。

### 完整 ILM 策略示例

```json
PUT _ilm/policy/logs-lifecycle-policy
{
  "policy": {
    "phases": {
      "hot": {
        "min_age": "0ms",
        "actions": {
          "rollover": {
            "max_age": "7d",
            "max_primary_shard_size": "50gb"
          },
          "forcemerge": {
            "max_num_segments": 1
          },
          "set_priority": {
            "priority": 100
          }
        }
      },
      "warm": {
        "min_age": "1d",
        "actions": {
          "allocate": {
            "require": {
              "_tier_preference": "data_warm"
            }
          },
          "shrink": {
            "number_of_shards": 1
          },
          "readonly": {},
          "set_priority": {
            "priority": 50
          }
        }
      },
      "cold": {
        "min_age": "30d",
        "actions": {
          "searchable_snapshot": {
            "snapshot_repository": "my-s3-repo"
          },
          "set_priority": {
            "priority": 0
          }
        }
      },
      "delete": {
        "min_age": "180d",
        "actions": {
          "delete": {}
        }
      }
    }
  }
}
```

---

## 索引模板与 Data Stream

ILM 策略定义好后，还需要通过索引模板和 Data Stream 将其与实际写入数据关联起来。

### 索引模板绑定 ILM

索引模板（Index Template）是创建新索引时的"配置蓝图"，通过模式匹配（index_patterns）自动应用于符合命名规则的索引：

```json
PUT _index_template/logs-app-template
{
  "index_patterns": ["logs-app-*"],
  "data_stream": {},
  "template": {
    "settings": {
      "number_of_shards": 2,
      "number_of_replicas": 1,
      "index.lifecycle.name": "logs-lifecycle-policy",
      "index.routing.allocation.require._tier_preference": "data_hot"
    },
    "mappings": {
      "properties": {
        "@timestamp": { "type": "date" },
        "level": { "type": "keyword" },
        "message": { "type": "text" },
        "service": { "type": "keyword" }
      }
    }
  },
  "priority": 200
}
```

### Data Stream：时序数据的原生方案

ES 7.9 引入的 **Data Stream** 是专为时序/日志数据设计的抽象层，它本质上是一组自动滚动的 backing index 集合，通过统一的写入别名（与 Data Stream 同名）接收数据。

```
写入到 "logs-app"（Data Stream 名称）
         │
         ▼
   .ds-logs-app-2026.02.11-000001  ← 当前写入 backing index（Hot 层）
   .ds-logs-app-2026.02.04-000002  ← 已滚动（Warm 层）
   .ds-logs-app-2026.01.05-000003  ← 归档（Cold 层）
```

创建 Data Stream 只需一个 API 调用（索引模板中声明 `"data_stream": {}` 后）：

```json
PUT _data_stream/logs-app
```

随后直接向 `logs-app` 写入文档即可，所有分层和滚动由 ILM + Data Stream 自动管理。

Data Stream 相比手动管理 Rollover 别名的优势在于：命名规范统一、时间戳字段强制为 `@timestamp`、跨层范围查询更自然。对于新建的日志类系统，优先选择 Data Stream 模式。

---

## Searchable Snapshot：将冷数据推入对象存储

Searchable Snapshot（可搜索快照）是冷热分层架构的"成本杀手"，它允许将索引数据迁移到 S3/MinIO/GCS 等廉价对象存储，同时保持 ES 查询接口的兼容性。

### 工作原理

传统快照是"备份后还原"的模式，查询前需要将快照完整恢复为本地索引，占用大量磁盘。Searchable Snapshot 的核心差异在于：**数据不需要完整恢复，Lucene 的底层文件被映射到对象存储上，按需分块拉取**。

```
查询请求
    │
    ▼
Frozen 节点
    ├── 本地 Shared Cache（LRU，命中则直接返回）
    │         │（未命中）
    │         ▼
    └── 对象存储（S3/MinIO）→ 拉取对应的 Lucene 文件块 → 写入缓存
```

Lucene 的文件结构天然友好于这种访问模式：倒排索引（`.tim`/`.tip` 文件）和文档存储（`.fdx`/`.fdt` 文件）是独立的，查询时通常只需拉取与命中文档相关的少量文件块，不需要读取整个索引文件。

### Full Copy vs Shared Cache

Searchable Snapshot 有两种挂载模式：

| 模式 | 本地存储 | 适用层 | 查询性能 |
|------|---------|-------|---------|
| Full Copy | 完整拷贝到本地磁盘 | Cold 层 | 接近本地索引 |
| Shared Cache | 仅共享缓存，按需拉取 | Frozen 层 | 有冷启动延迟 |

Full Copy 模式下，第一次挂载时会将所有文件从对象存储下载到本地，之后的查询性能与普通索引无异，适合仍有一定查询频率的 Cold 层。Shared Cache 模式不做完整下载，节点内存中维护一个 LRU 缓存，首次查询时才按需拉取，适合几乎不被查询的 Frozen 层。

### 注册 Repository 与启用流程

使用 Searchable Snapshot 前，需要先注册对象存储仓库：

```json
PUT _snapshot/my-s3-repo
{
  "type": "s3",
  "settings": {
    "bucket": "es-cold-data",
    "base_path": "snapshots",
    "region": "cn-hangzhou",
    "endpoint": "oss-cn-hangzhou.aliyuncs.com"
  }
}
```

注册完成后，ILM 的 Cold 阶段配置中引用该仓库名（如前文完整策略中的 `"snapshot_repository": "my-s3-repo"`），ILM 会在索引进入 Cold 阶段时自动创建快照并以 Searchable Snapshot 方式挂载。

### 成本对比

以 100TB 归档数据为例（某云平台参考价格）：

```
本地 HDD（0.1 元/GB/月）：   100TB × 1024 × 0.1 = 10,240 元/月
对象存储（0.015 元/GB/月）： 100TB × 1024 × 0.015 = 1,536 元/月
```

归档层的存储成本可降低约 85%，对于大规模日志平台，这是相当可观的节约。

---

## 生产规划参考

### 各层数据保留周期（日志场景）

| 层次 | 进入条件（相对 Rollover） | 数据特征 | 建议保留 |
|------|------------------------|---------|---------|
| Hot | 写入期间 | 高频写入 + 实时查询 | 7 天 |
| Warm | Rollover 后 1 天 | 低频读，近期历史 | 7～30 天 |
| Cold | 进入 Warm 后 30 天 | 偶发读，归档查询 | 30～90 天 |
| Delete | 进入 Cold 后 90 天 | — | 总计 ≤ 180 天 |

### 各层硬件规格参考

| 层次 | CPU | 内存 | 存储 | 节点数参考 |
|------|-----|------|------|-----------|
| Hot | 32 核+ | 64GB+ | NVMe SSD 2TB+ | ≥ 3 |
| Warm | 16 核 | 32GB | SATA SSD / HDD 8TB+ | ≥ 2 |
| Cold | 8 核 | 16GB | HDD 20TB+ | ≥ 2 |
| Frozen | 8 核 | 16GB（主要用于缓存） | SSD 500GB（缓存盘） | ≥ 1 |

### 分片策略与分层配合

Hot 层索引配置 2～3 个主分片，以支撑并发写入。进入 Warm 阶段后，通过 Shrink 合并为 1 个主分片：

```
Hot 层：logs-app-000001  主分片 ×2，副本 ×1  → 共 4 个分片占用节点
Warm 层：Shrink 后       主分片 ×1，副本 ×1  → 共 2 个分片
Cold 层：副本数降为 0    主分片 ×1，副本 ×0  → 共 1 个分片（Snapshot 保障可恢复性）
```

副本数在 Cold 层降为 0 是合理的——Searchable Snapshot 已经将数据持久化在对象存储中，即使节点故障，数据也可以重新挂载。这进一步降低了 Cold 层的存储占用。

---

## 常见踩坑与排查

### Q1：ILM 策略更新后，现有索引没有按新策略执行，怎么处理？

ILM 策略更新对已存在的索引不会立即生效，原因是每个索引会缓存其关联的策略版本。有两种处理方式：

第一种是等待，ILM 在下一个轮询周期（默认 10 分钟）会重新评估策略，大多数情况下会自动拾取新版本。

第二种是强制触发重试，适用于索引处于 ERROR 状态时：

```json
POST /logs-app-000001/_ilm/retry
```

如果需要让大量历史索引立刻应用新策略，可以通过更新索引设置，将策略重新绑定一次以触发状态机重置。

### Q2：Shrink 操作失败，提示分片未分配到同一节点

Shrink 要求被 Shrink 的索引所有主分片**先全部迁移到同一个 Hot 节点**，ES 才能在本地完成文件硬链接操作。如果分片分散在多个节点，操作会失败。

排查方式：

```json
GET /_cat/shards/logs-app-000001?v&h=index,shard,prirep,state,node
```

手动将分片迁移到同一节点（执行 Shrink 前的准备步骤）：

```json
PUT /logs-app-000001/_settings
{
  "index.routing.allocation.require._name": "hot-node-1",
  "index.blocks.write": true
}
```

等待分片重新分配完成后，Shrink 即可成功执行。生产中更常见的方案是通过 ILM 的 Warm 阶段配置 `allocate` 动作指定目标节点数为 1，让 ES 在 Shrink 前自动完成这一准备。

### Q3：Searchable Snapshot 查询超时或极慢

首次查询 Frozen 层索引时，对象存储的文件块尚未进入本地缓存（Shared Cache），所有数据都需要从对象存储拉取，延迟可能达到数十秒。这是 Shared Cache 模式的固有特性，称为"冷启动"。

优化方向有两个：

1. **预热缓存**：在业务低峰期提前发出探针查询，使相关文件块进入缓存。
2. **调整 Shared Cache 大小**：通过 `xpack.searchable.snapshot.shared_cache.size` 设置更大的本地缓存（如 `100gb`），减少缓存被驱逐的频率。
3. **将高频归档查询的索引改用 Full Copy 模式**：如果某些 Cold 层索引被查询的频率比预期高，考虑将其从 Shared Cache 模式改为 Full Copy 模式。

### Q4：跨层聚合查询变慢，如何隔离不同层的查询

在冷热分层架构中，一个针对"全量数据"的聚合查询会同时命中 Hot、Warm、Cold 三层，Cold 层的延迟会拖慢整体响应时间。

解决思路是在应用层通过**时间范围参数**控制查询命中的层次范围：

```json
GET /logs-app/_search
{
  "query": {
    "range": {
      "@timestamp": {
        "gte": "now-7d/d",
        "lt": "now/d"
      }
    }
  }
}
```

ES 的 Data Stream 查询支持按时间范围裁剪，对于明确指定了时间范围的查询，协调节点会跳过时间范围不匹配的 backing index，实现查询剪枝。在 Kibana / 可观测平台中，默认时间选择器的存在，天然帮助大多数用户规避了跨层的全量扫描。

对于需要跨层的统计类查询（如全量日志的 error 比例），建议使用 **Rollup Job** 或 **Transform**，提前将冷数据的聚合结果固化到独立的统计索引中，避免在线实时扫描冷层。

### Q5：ILM 中 min_age 是相对哪个时间点计算的？

`min_age` 的计算基准根据阶段不同而有所区别：

- **Hot 阶段**：`min_age` 是相对索引创建时间。
- **Warm、Cold、Delete 阶段**：`min_age` 默认相对于 **Rollover 时间**（而非索引创建时间）。

这个细节非常重要。如果一个索引在创建 7 天后才触发 Rollover（因为数据写入量不足），而 Warm 阶段的 `min_age` 设置为 `1d`，则该索引进入 Warm 层的实际时间是 **Rollover 后 1 天**，而不是索引创建后 8 天。

可以通过以下 API 查看索引的 ILM 执行状态和时间节点：

```json
GET /logs-app-000001/_ilm/explain
```

返回的 `age`、`lifecycle_date` 等字段清晰显示了当前状态和下次状态转换的预计时间，是排查 ILM 执行异常的第一手段。

---

## 小结

- ES 冷热分层的硬件基础是 **data_hot / data_warm / data_cold / data_frozen** 四层节点角色，分片通过 `_tier_preference` 路由设置自动落到对应层次。
- **ILM** 是分层自动化的核心引擎，通过 Rollover、Shrink、Allocate、Searchable Snapshot 四类动作，将索引从 Hot 逐步迁移到 Delete，全程无需人工干预。
- **Data Stream** 是时序数据的推荐管理模式，统一写入别名、自动管理 backing index 命名和滚动，与 ILM 深度集成。
- **Searchable Snapshot** 是降低冷数据成本的关键技术，将数据迁入对象存储的同时保持查询接口兼容，Frozen 层场景下可将归档成本降低 80% 以上。
- 生产落地的三个关键决策：**各层保留时长的规划**、**Shrink 前分片分配的准备**、**Shared Cache 大小与查询性能的平衡**，直接决定分层架构能否稳定运行。

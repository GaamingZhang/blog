---
date: 2026-01-14
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Elasticsearch
tag:
  - Elasticsearch
---

# Elasticsearch基本概念

> **版本说明**：本文基于 Elasticsearch 8.x 编写，部分特性在 7.x 版本中已有变化，请根据实际使用的版本进行调整。

## 一、从一个问题开始

假设你刚接手一个电商项目的搜索功能，产品经理提出了这些需求：

- 用户输入"无线蓝牙耳机"，要能搜到相关商品
- 搜索结果要按相关度排序，还要支持按价格、销量筛选
- 数据量预计有上千万商品，查询响应要在 200ms 以内
- 系统要高可用，单台服务器挂了不能影响服务

你可能会想：用 MySQL 的 `LIKE '%关键词%'` 不就行了？

但很快你会发现问题：
- `LIKE` 查询走不了索引，千万级数据查询慢得让人绝望
- 没法做相关度排序，"蓝牙耳机"和"耳机蓝牙"是一样的吗？
- 中文分词怎么处理？"无线蓝牙耳机"应该拆成"无线"、"蓝牙"、"耳机"
- 高并发下数据库压力太大

这时候，Elasticsearch 登场了。

**Elasticsearch 是一个基于 Apache Lucene 构建的分布式搜索和分析引擎**，专门解决上述问题：

| 问题 | Elasticsearch 的解决方案 |
|------|-------------------------|
| 全文搜索慢 | 倒排索引 + 分词器，毫秒级响应 |
| 相关度排序 | TF-IDF/BM25 算法自动计算相关性得分 |
| 中文分词 | 支持 IK 等中文分词器 |
| 海量数据 | 分片机制实现水平扩展 |
| 高可用 | 副本机制提供数据冗余 |

本文将系统介绍 Elasticsearch 的核心概念，帮助你建立完整的知识体系。对于分片规划、写入流程、集群状态、冷热分层等专题内容，本文仅作概述，详细内容请参考相应的专题文章。

### 核心特性

- **分布式架构**：天然支持集群部署，可横向扩展
- **近实时搜索**：文档从索引到可搜索通常只需 1 秒
- **RESTful API**：通过 HTTP 请求进行所有操作
- **多租户支持**：一个集群可以包含多个索引
- **Schema-free**：支持动态映射，无需预定义结构

## 二、核心概念详解

### 2.1 集群(Cluster)

集群是一个或多个节点的集合，这些节点共同保存所有数据并提供跨所有节点的联合索引和搜索功能。

**关键特性：**
- 每个集群有唯一的名称标识(默认为"elasticsearch")
- 节点通过集群名称加入集群
- 一个集群可以只有一个节点

**配置示例：**
```yaml
cluster.name: my-application
```

### 2.2 节点(Node)

节点是集群中的单个服务器实例，存储数据并参与集群的索引和搜索功能。

**节点类型：**

1. **主节点(Master Node)**
   - 负责集群级别的操作(创建/删除索引、跟踪节点等)
   - 配置:`node.master: true`

2. **数据节点(Data Node)**
   - 存储数据并执行数据相关操作(CRUD、搜索、聚合)
   - 配置:`node.data: true`

3. **协调节点(Coordinating Node)**
   - 处理请求路由、搜索结果汇总
   - 所有节点默认都是协调节点

4. **摄入节点(Ingest Node)**
   - 数据预处理
   - 配置:`node.ingest: true`

**配置示例：**
```yaml
node.name: node-1
node.master: true
node.data: true
node.ingest: false
```

### 2.3 索引(Index)

索引是具有相似特征的文档集合，类似于关系数据库中的"数据库"。

**特点：**
- 索引名称必须全部小写
- 通过索引名称进行文档的增删改查

**命名规范：**
- 小写字母
- 不能包含 `\`、`/`、`*`、`?`、`"`、`<`、`>`、`|`、空格、`,`、`#`
- 不能以 `-`、`_`、`+` 开头
- 不能是 `.` 或 `..`

**创建索引示例：**
```json
PUT /my_index
{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 2
  },
  "mappings": {
    "properties": {
      "title": { "type": "text" },
      "created_at": { "type": "date" }
    }
  }
}
```
// 创建索引 my_index，设置 3 个主分片、2 个副本，并定义字段映射

### 2.4 类型(Type)——已废弃

类型曾是索引内的逻辑分类，类似于关系数据库中的"表"。**该概念已废弃**。

**废弃时间线：**
- **Elasticsearch 6.x**：一个索引只能有一个类型（限制多类型）
- **Elasticsearch 7.x**：类型概念废弃，默认使用 `_doc` 类型，API 中不再需要指定类型
- **Elasticsearch 8.x**：完全移除类型概念，所有类型相关 API 已删除

**迁移建议：**
- 使用独立的索引代替类型
- 或在文档中添加 `type` 字段进行区分

### 2.5 文档(Document)

文档是可以被索引的基本信息单元，以JSON格式表示，类似于关系数据库中的"行"。

**文档特性：**
- 每个文档有唯一的ID(可自定义或自动生成)
- 文档是自包含的：包含字段和值
- 文档可以是层次化的：字段值可以是子文档

**文档示例：**
```json
{
  "_index": "products",
  "_id": "1",
  "_source": {
    "name": "Elasticsearch Guide",
    "price": 59.99,
    "category": "Books",
    "tags": ["search", "analytics"],
    "author": {
      "name": "John Doe",
      "email": "john@example.com"
    },
    "publish_date": "2024-01-15"
  }
}
```

### 2.6 分片(Shard)

分片是索引的物理分区，每个分片本身就是一个功能完整且独立的"索引"。

**为什么需要分片：**
- 水平分割/扩展内容容量
- 分布式并行操作，提高性能和吞吐量

**分片类型：**

1. **主分片(Primary Shard)**
   - 索引创建时指定数量，创建后不可直接修改（可通过 Split/Shrink API 调整）
   - 每个文档只存在于一个主分片
   - **默认值**：7.0 之前为 5 个，7.0 及之后为 1 个

2. **副本分片(Replica Shard)**
   - 主分片的副本，提供数据冗余和查询性能
   - 可以随时调整数量
   - 不会与对应的主分片分配在同一节点

**配置示例：**
```json
PUT /my_index
{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 2
  }
}
```
// 创建索引 my_index，设置 3 个主分片、2 个副本分片

**调整副本数：**
```json
PUT /my_index/_settings
{
  "number_of_replicas": 1
}
```
// 将副本数从 2 调整为 1，该操作即时生效

> **分片规划详解**：关于主分片数量为何不可修改、如何选择合适的分片数量、Split/Shrink API 的使用方法，请参考专题文章 [Elasticsearch索引分片规划与主分片不可变原理](./Elasticsearch索引分片规划与主分片不可变原理.md)。

### 2.7 映射(Mapping)

映射定义了文档及其包含的字段如何存储和索引，类似于关系数据库中的"schema"。

**映射类型：**

1. **动态映射(Dynamic Mapping)**
   - Elasticsearch自动推断字段类型
   - 适用于快速开发，但可能不够精确

2. **显式映射(Explicit Mapping)**
   - 手动定义字段类型
   - 提供更精确的控制

**常用字段类型：**

**基本类型：**
- `text`：全文本，会被分词
- `keyword`：精确值，不分词
- `long`、`integer`、`short`、`byte`：整数
- `double`、`float`：浮点数
- `boolean`：布尔值
- `date`：日期类型
- `binary`：二进制

**复杂类型：**
- `object`：JSON对象
- `nested`：嵌套对象数组
- `geo_point`：地理位置点
- `geo_shape`：地理位置形状
- `ip`：IP地址

**映射示例：**
```json
PUT /my_index
{
  "mappings": {
    "properties": {
      "title": {
        "type": "text",
        "analyzer": "standard"
      },
      "status": {
        "type": "keyword"
      },
      "price": {
        "type": "double"
      },
      "publish_date": {
        "type": "date",
        "format": "yyyy-MM-dd"
      },
      "author": {
        "type": "object",
        "properties": {
          "name": { "type": "text" },
          "age": { "type": "integer" }
        }
      },
      "tags": {
        "type": "keyword"
      }
    }
  }
}
```
// 定义索引映射：title 使用 standard 分词器、status 用于精确匹配、author 是嵌套对象

**动态映射配置：**
```json
PUT /my_index
{
  "mappings": {
    "dynamic": "strict",
    "properties": {
      "name": { "type": "text" }
    }
  }
}
```
// strict: 拒绝未知字段（写入会报错）| true: 自动添加字段映射 | false: 忽略未知字段

### 2.8 分析器(Analyzer)

分析器用于文本分析，将文本转换为倒排索引中的词条(term)。

**分析器组成：**
1. **字符过滤器(Character Filter)**：处理原始文本(如去除HTML标签)
2. **分词器(Tokenizer)**：将文本分割成词条
3. **词条过滤器(Token Filter)**：处理词条(如转小写、去除停用词)

**内置分析器：**
- `standard`：默认分析器，按词分割，小写处理
- `simple`：按非字母字符分割，小写处理
- `whitespace`:按空格分割
- `stop`：类似simple，但会去除停用词
- `keyword`：不分词，整个文本作为单个词条
- `pattern`:使用正则表达式分割
- `language`:特定语言分析器(如`english`、`chinese`)

**中文分析器：**
- IK分词器(需安装插件)
  - `ik_smart`:粗粒度分词
  - `ik_max_word`:细粒度分词

**自定义分析器示例：**
```json
PUT /my_index
{
  "settings": {
    "analysis": {
      "analyzer": {
        "my_custom_analyzer": {
          "type": "custom",
          "char_filter": ["html_strip"],
          "tokenizer": "standard",
          "filter": ["lowercase", "stop"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "content": {
        "type": "text",
        "analyzer": "my_custom_analyzer"
      }
    }
  }
}
```
// 创建自定义分析器：先去除 HTML 标签，再按词分割，最后转小写并去除停用词

**测试分析器：**
```json
POST /my_index/_analyze
{
  "analyzer": "standard",
  "text": "The Quick Brown Fox"
}
```
// 返回分词结果：["the", "quick", "brown", "fox"]

### 2.9 倒排索引(Inverted Index)

倒排索引是Elasticsearch实现快速全文搜索的核心数据结构。

**原理：**
- 传统索引：文档ID → 内容
- 倒排索引：词条 → 文档ID列表

**示例：**

假设有以下文档:
- Doc1: "Elasticsearch is powerful"
- Doc2: "Elasticsearch is fast"
- Doc3: "Fast and powerful search"

倒排索引结构:
```
Term          | Document IDs
------------- | ------------
elasticsearch | [1, 2]
is            | [1, 2]
powerful      | [1, 3]
fast          | [2, 3]
and           | [3]
search        | [3]
```

**倒排索引组成：**
1. **词条字典(Term Dictionary)**：所有文档的词条集合
2. **倒排列表(Posting List)**：词条对应的文档ID列表及其他信息

**附加信息：**
- 词频(TF)：词条在文档中出现的次数
- 位置(Position)：词条在文档中的位置
- 偏移(Offset)：词条的字符偏移量

## 三、Elasticsearch vs 关系数据库

| Elasticsearch | 关系数据库 | 说明 |
|--------------|---------|------|
| Index | Database | 数据库 |
| ~~Type~~ | Table | 表(7.x后废弃) |
| Document | Row | 行/记录 |
| Field | Column | 列/字段 |
| Mapping | Schema | 结构定义 |
| Query DSL | SQL | 查询语言 |
| GET | SELECT | 查询操作 |
| PUT/POST | INSERT | 插入操作 |
| DELETE | DELETE | 删除操作 |
| POST(with _update) | UPDATE | 更新操作 |

**关键区别：**
- ES是文档型数据库，关系数据库是关系型
- ES擅长全文搜索，关系数据库擅长事务处理
- ES是Schema-free，关系数据库是Schema-based
- ES的JOIN能力有限，关系数据库支持复杂JOIN

## 四、数据的读写流程

### 4.1 写入流程概述

当客户端向 Elasticsearch 写入一条文档时，请求会经过以下步骤：

1. **客户端发送请求**：写请求可以发送到任意节点
2. **协调节点处理**：接收请求的节点成为协调节点，根据文档 ID 计算应该存储的分片
3. **路由到主分片**：协调节点将请求转发到对应的主分片所在节点
4. **主分片处理**：主分片执行写操作
5. **同步到副本**：主分片并行地将操作转发到所有副本分片
6. **返回结果**：所有副本确认后，主分片向协调节点报告成功，协调节点向客户端返回结果

**路由公式：**
```
shard = hash(routing) % number_of_primary_shards
```

其中 `routing` 默认为文档 ID，这就是为什么主分片数量创建后不能修改的原因——修改后会导致已有文档的路由失效。

> **写入流程详解**：关于写入流程的详细分析、refresh_interval 调优、Translog 机制、写入性能优化策略，请参考专题文章 [Elasticsearch写入查询流程与refresh_interval调优](./Elasticsearch写入查询流程与refresh_interval调优.md)。

### 4.2 读取流程概述

**通过 ID 查询：**
1. 客户端发送 GET 请求到任意节点
2. 协调节点根据文档 ID 计算分片位置
3. 协调节点使用轮询策略选择主分片或副本分片
4. 目标分片返回文档
5. 协调节点返回给客户端

**搜索查询（两阶段）：**
1. **Query 阶段**：协调节点广播请求到所有相关分片，每个分片本地执行查询并返回文档 ID 和排序值，协调节点合并结果
2. **Fetch 阶段**：协调节点根据文档 ID 到对应分片获取完整文档，返回最终结果

### 4.3 更新和删除流程

**更新流程：**
1. 检索完整文档
2. 修改文档内容
3. 删除旧文档（标记删除）
4. 索引新文档

**删除流程：**
1. 不会立即物理删除
2. 标记为已删除（.del 文件）
3. 段合并时真正删除

## 五、近实时搜索原理

Elasticsearch 被称为"近实时"搜索引擎，是因为文档从索引到可搜索有短暂的延迟（默认约 1 秒）。

### 5.1 核心机制概述

**Refresh 机制：**
- 新文档首先写入内存缓冲区
- 每隔 1 秒（默认），缓冲区内容写入新的 Segment，变为可搜索状态
- 这就是"近实时"的原因

**Translog 机制：**
- 文档写入内存缓冲区的同时追加到 Translog（事务日志）
- 防止数据丢失，确保持久性

**Segment Merging：**
- 后台自动合并小 Segment 为大 Segment
- 清理已删除的文档，提高查询性能

### 5.2 配置示例

**调整 refresh_interval：**
```json
PUT /my_index/_settings
{
  "refresh_interval": "30s"
}
// 将刷新间隔从默认 1 秒调整为 30 秒，可提高写入性能
```

**手动刷新：**
```json
POST /my_index/_refresh
// 立即刷新，使所有未刷新的文档变为可搜索
```

> **写入流程详解**：关于 Refresh、Translog、Flush 的完整工作流程、性能调优策略、写入优化最佳实践，请参考专题文章 [Elasticsearch写入查询流程与refresh_interval调优](./Elasticsearch写入查询流程与refresh_interval调优.md)。

## 六、集群健康与状态

### 6.1 集群健康状态

**三种状态：**
- **Green（绿色）**：所有主分片和副本分片都正常
- **Yellow（黄色）**：所有主分片正常，但部分副本分片不可用
- **Red（红色）**：部分主分片不可用

**查看集群健康：**
```json
GET /_cluster/health
```
// 返回集群整体健康状态，包括节点数、分片数、未分配分片数等信息

**响应示例：**
```json
{
  "cluster_name": "my-cluster",
  "status": "yellow",
  "timed_out": false,
  "number_of_nodes": 3,
  "number_of_data_nodes": 3,
  "active_primary_shards": 10,
  "active_shards": 15,
  "relocating_shards": 0,
  "initializing_shards": 0,
  "unassigned_shards": 5
}
```
// status 为 yellow 表示有 5 个副本分片未分配，但所有主分片正常

> **集群状态排查详解**：关于 Yellow/Red 状态的详细排查步骤、分片分配失败原因分析、恢复方案，请参考专题文章 [Elasticsearch集群黄色红色状态排查与恢复](./Elasticsearch集群黄色红色状态排查与恢复.md)。

### 6.2 分片分配原则

- 主分片和副本分片不在同一节点
- 同一索引的分片尽量均匀分布
- 考虑节点的磁盘使用率

**分片分配设置：**
```json
PUT /_cluster/settings
{
  "transient": {
    "cluster.routing.allocation.enable": "all"
  }
}
```
// all: 允许所有分片分配 | primaries: 仅主分片 | new_primaries: 仅新主分片 | none: 禁止分配

## 七、性能优化建议

### 7.1 索引设计优化

**分片数量：**
- 分片不是越多越好
- 单个分片建议20-40GB
- 每个节点的分片数不超过20个/GB堆内存

**副本策略：**
- 生产环境至少1个副本
- 读多写少：增加副本数
- 写多读少：减少副本数

**Mapping优化：**
- 明确定义Mapping，避免动态映射
- 不需要搜索的字段设置`"index": false`
- 不需要评分的字段使用`keyword`类型
- 关闭不需要的字段特性(doc_values、norms等)

### 7.2 写入性能优化

**批量操作：**
```json
POST /_bulk
{ "index": { "_index": "my_index", "_id": "1" } }
{ "title": "Document 1" }
{ "index": { "_index": "my_index", "_id": "2" } }
{ "title": "Document 2" }
```
// 使用 Bulk API 批量写入，减少网络开销，建议每批 5-15MB

**其他优化：**
- 增大 `refresh_interval`
- 写入时禁用副本，写入完成后再启用
- 增大 `index.translog.flush_threshold_size`
- 使用自动生成的 ID 而不是自定义 ID
- 调整 JVM 堆内存（建议不超过 32GB）

### 7.3 查询性能优化

**索引优化：**
- 使用 Filter Context 代替 Query Context（可缓存）
- 避免深度分页
- 使用 scroll 或 search_after 代替 from/size
- 预先过滤数据，减少搜索范围

**查询示例：**
```json
GET /my_index/_search
{
  "query": {
    "bool": {
      "must": [
        { "match": { "title": "elasticsearch" } }
      ],
      "filter": [
        { "term": { "status": "published" } },
        { "range": { "date": { "gte": "2024-01-01" } } }
      ]
    }
  }
}
```
// filter 中的条件会被缓存，适合精确匹配；must 中的条件参与评分

## 八、常见应用场景

### 8.1 全文搜索

**电商搜索示例：**
```json
GET /products/_search
{
  "query": {
    "multi_match": {
      "query": "wireless headphones",
      "fields": ["name^3", "description", "brand^2"],
      "type": "best_fields"
    }
  },
  "highlight": {
    "fields": {
      "name": {},
      "description": {}
    }
  }
}
```
// 多字段搜索：name 权重 x3，brand 权重 x2，description 默认权重；结果高亮显示匹配内容

### 8.2 日志分析

ELK Stack（Elasticsearch + Logstash + Kibana）：
- Logstash：收集和处理日志
- Elasticsearch：存储和搜索日志
- Kibana：可视化展示

**日志查询示例：**
```json
GET /logs-*/_search
{
  "query": {
    "bool": {
      "must": [
        { "match": { "level": "ERROR" } }
      ],
      "filter": [
        { "range": { "@timestamp": { "gte": "now-1h" } } }
      ]
    }
  },
  "aggs": {
    "error_by_service": {
      "terms": { "field": "service.keyword" }
    }
  }
}
```
// 查询最近 1 小时的 ERROR 日志，并按 service 字段聚合统计各服务的错误数量

### 8.3 实时数据分析

**聚合分析示例：**
```json
GET /orders/_search
{
  "size": 0,
  "aggs": {
    "sales_by_month": {
      "date_histogram": {
        "field": "order_date",
        "calendar_interval": "month"
      },
      "aggs": {
        "total_sales": {
          "sum": { "field": "amount" }
        }
      }
    }
  }
}
```
// 按月统计销售额：size=0 表示不返回文档，仅返回聚合结果

## 九、常见问题

### 1. 为什么主分片数量创建后不能修改？

**原因：**
文档路由到分片使用公式：`shard = hash(routing) % number_of_primary_shards`

如果改变主分片数量，已有文档的路由会改变，导致无法找到原有数据。

**解决方案：**
- 使用 Split API 增加分片数（只能增加到原来的倍数）
- 使用 Shrink API 减少分片数（新分片数必须是原分片数的因子）
- 更常见的方案是 Reindex 到新索引

> **详细分析**：关于主分片不可变的深层原理、Split/Shrink API 的完整使用方法，请参考专题文章 [Elasticsearch索引分片规划与主分片不可变原理](./Elasticsearch索引分片规划与主分片不可变原理.md)。

### 2. Text 和 Keyword 类型有什么区别？

**Text 类型：**
- 用于全文搜索
- 会被分词器分析
- 不能用于排序和聚合
- 适用场景：文章内容、产品描述

**Keyword 类型：**
- 用于精确匹配
- 不分词，整个值作为一个词条
- 可用于排序、聚合和过滤
- 适用场景：邮箱、状态码、标签、ID

**示例对比：**
```json
// Text 字段
"content": {
  "type": "text",
  "analyzer": "standard"
}
// "Hello World" 被分词为 ["hello", "world"]

// Keyword 字段
"status": {
  "type": "keyword"
}
// "Hello World" 保持原样 ["Hello World"]
```

**多字段类型（fields）：**
```json
"title": {
  "type": "text",
  "fields": {
    "keyword": {
      "type": "keyword"
    }
  }
}
// 可以用 title 进行全文搜索，用 title.keyword 进行精确匹配和聚合
```

### 3. 如何避免深度分页问题？

**深度分页问题：**
当使用 `from + size` 进行深度分页时，协调节点需要从每个分片获取 `from + size` 个文档，然后排序，性能极差。

例如：获取第 10000 页，每页 10 条，需要从每个分片获取 100010 个文档。

**解决方案：**

**方案 1：Scroll API（快照查询）**
```json
// 初始化 scroll
POST /my_index/_search?scroll=1m
{
  "size": 100,
  "query": { "match_all": {} }
}
// 返回第一批结果和 scroll_id，用于后续获取

// 继续获取
POST /_search/scroll
{
  "scroll": "1m",
  "scroll_id": "scroll_id_from_previous_response"
}
// 使用上一次返回的 scroll_id 获取下一批数据
```

**适用场景：**
- 导出全部数据
- 不适合实时查询（数据快照）

**方案 2：Search After（推荐）**
```json
// 第一页
GET /my_index/_search
{
  "size": 10,
  "query": { "match_all": {} },
  "sort": [
    { "date": "desc" },
    { "_id": "desc" }
  ]
}
// 返回结果中包含 sort 值，用于下一页查询

// 下一页（使用上一页最后一个文档的 sort 值）
GET /my_index/_search
{
  "size": 10,
  "query": { "match_all": {} },
  "search_after": ["2024-01-15", "doc_id_123"],
  "sort": [
    { "date": "desc" },
    { "_id": "desc" }
  ]
}
// search_after 值来自上一页最后一条记录的 sort 字段
```

**适用场景：**
- 实时查询
- 下一页/上一页导航
- 不支持跳页

**方案 3：限制分页深度**
```json
PUT /my_index/_settings
{
  "index.max_result_window": 10000
}
// 默认值 10000，超过此限制的 from+size 查询会报错
```

### 4. 集群状态为 Yellow 的常见原因？

**Yellow 状态表示：**所有主分片正常，但部分副本分片未分配。

**常见原因：**
- **单节点集群**：副本分片不能和主分片在同一节点，单节点无法分配副本
- **节点磁盘空间不足**：默认磁盘使用率超过 85% 时停止分配副本
- **分片分配被禁用**：集群配置问题导致分片无法分配

> **详细排查**：关于 Yellow/Red 状态的完整排查步骤和恢复方案，请参考专题文章 [Elasticsearch集群黄色红色状态排查与恢复](./Elasticsearch集群黄色红色状态排查与恢复.md)。

### 5. 如何选择合适的分片数量？

**基本原则：**
- 单个分片建议大小：日志/时序数据 20-40GB，搜索场景 10-30GB
- 计算方法：`分片数 = 预估数据量 / 单个分片目标大小`
- 每个节点的分片数不超过 `20 × 堆内存(GB)`

> **详细规划**：关于分片数量的选择策略、性能权衡、时序数据的滚动索引方案，请参考专题文章 [Elasticsearch索引分片规划与主分片不可变原理](./Elasticsearch索引分片规划与主分片不可变原理.md)。

## 十、实战最佳实践

### 10.1 索引命名规范

**推荐命名模式：**
```
{业务类型}-{环境}-{日期}
例如:
- user-prod-2024-01
- order-dev-2024-01-15
- logs-staging-2024-w03
```

**优势：**
- 便于管理和维护
- 支持通配符查询
- 便于实施生命周期管理

### 10.2 索引模板(Index Template)

索引模板可以在创建新索引时自动应用预定义的设置和映射。

**创建索引模板：**
```json
PUT /_index_template/logs_template
{
  "index_patterns": ["logs-*"],
  "priority": 100,
  "template": {
    "settings": {
      "number_of_shards": 3,
      "number_of_replicas": 1,
      "refresh_interval": "30s"
    },
    "mappings": {
      "properties": {
        "@timestamp": {
          "type": "date"
        },
        "level": {
          "type": "keyword"
        },
        "message": {
          "type": "text"
        },
        "service": {
          "type": "keyword"
        }
      }
    }
  }
}
```

**组件模板(Component Template)：**
```json
PUT /_component_template/common_settings
{
  "template": {
    "settings": {
      "number_of_shards": 3,
      "number_of_replicas": 1
    }
  }
}

PUT /_component_template/logs_mappings
{
  "template": {
    "mappings": {
      "properties": {
        "@timestamp": { "type": "date" },
        "message": { "type": "text" }
      }
    }
  }
}

PUT /_index_template/logs_template
{
  "index_patterns": ["logs-*"],
  "composed_of": ["common_settings", "logs_mappings"]
}
```

### 10.3 索引生命周期管理(ILM)

ILM（Index Lifecycle Management）用于自动管理索引的生命周期，特别适合日志、指标等时序数据。

**生命周期阶段：**
- **Hot**：频繁写入和查询
- **Warm**：不再写入，仍需查询
- **Cold**：很少查询，可压缩存储
- **Frozen**：极少访问，最小资源占用
- **Delete**：删除索引

**ILM 策略示例：**
```json
PUT /_ilm/policy/logs_policy
{
  "policy": {
    "phases": {
      "hot": {
        "actions": {
          "rollover": { "max_size": "50GB", "max_age": "7d" }
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
// 定义日志索引的生命周期：热数据阶段滚动索引，7天后收缩合并，90天后删除

> **冷热分层详解**：关于 Hot-Warm-Cold 架构设计、节点角色配置、数据自动迁移策略、ILM 与冷热分层结合的最佳实践，请参考专题文章 [Elasticsearch冷热数据分层架构设计](./Elasticsearch冷热数据分层架构设计.md)。

### 10.4 别名(Alias)使用

别名是索引的逻辑名称，可以指向一个或多个索引。

**创建别名：**
```json
POST /_aliases
{
  "actions": [
    {
      "add": {
        "index": "logs-2024-01-01",
        "alias": "logs"
      }
    }
  ]
}
```

**带过滤的别名：**
```json
POST /_aliases
{
  "actions": [
    {
      "add": {
        "index": "logs-2024-01",
        "alias": "error_logs",
        "filter": {
          "term": { "level": "ERROR" }
        }
      }
    }
  ]
}
```

**零停机重建索引：**
```json
// 1. 创建新索引
PUT /my_index_v2
{
  "mappings": { ... }
}

// 2. 重建索引
POST /_reindex
{
  "source": { "index": "my_index_v1" },
  "dest": { "index": "my_index_v2" }
}

// 3. 原子切换别名
POST /_aliases
{
  "actions": [
    { "remove": { "index": "my_index_v1", "alias": "my_index" } },
    { "add": { "index": "my_index_v2", "alias": "my_index" } }
  ]
}

// 4. 删除旧索引
DELETE /my_index_v1
```

### 10.5 监控和诊断

**集群状态监控：**
```json
// 集群健康
GET /_cluster/health

// 节点信息
GET /_cat/nodes?v

// 索引信息
GET /_cat/indices?v&s=store.size:desc

// 分片分配
GET /_cat/shards?v

// 线程池状态
GET /_cat/thread_pool?v

// 待处理任务
GET /_cat/pending_tasks?v
```

**性能分析：**
```json
// 慢查询日志配置
PUT /my_index/_settings
{
  "index.search.slowlog.threshold.query.warn": "10s",
  "index.search.slowlog.threshold.query.info": "5s",
  "index.search.slowlog.threshold.fetch.warn": "1s",
  "index.indexing.slowlog.threshold.index.warn": "10s"
}

// Profile API分析查询性能
GET /my_index/_search
{
  "profile": true,
  "query": {
    "match": { "title": "elasticsearch" }
  }
}
```

**热点线程分析：**
```json
GET /_nodes/hot_threads
```

## 十一、安全性配置

### 11.1 安全基础

**启用安全特性(X-Pack Security)：**
```yaml
# elasticsearch.yml
xpack.security.enabled: true
xpack.security.transport.ssl.enabled: true
xpack.security.http.ssl.enabled: true
```

**创建用户：**
```bash
# 设置内置用户密码
./bin/elasticsearch-setup-passwords interactive

# 或自动生成密码
./bin/elasticsearch-setup-passwords auto
```

### 11.2 角色和权限

**创建角色：**
```json
POST /_security/role/logs_reader
{
  "cluster": ["monitor"],
  "indices": [
    {
      "names": ["logs-*"],
      "privileges": ["read", "view_index_metadata"]
    }
  ]
}
```

**创建用户并分配角色：**
```json
POST /_security/user/log_user
{
  "password": "secure_password",
  "roles": ["logs_reader"],
  "full_name": "Log Reader User"
}
```

### 11.3 字段级别安全

**限制字段访问：**
```json
POST /_security/role/limited_user
{
  "indices": [
    {
      "names": ["users"],
      "privileges": ["read"],
      "field_security": {
        "grant": ["name", "email"],
        "except": ["password", "credit_card"]
      }
    }
  ]
}
```

### 11.4 文档级别安全

**基于查询的访问控制：**
```json
POST /_security/role/department_reader
{
  "indices": [
    {
      "names": ["employees"],
      "privileges": ["read"],
      "query": {
        "term": { "department": "engineering" }
      }
    }
  ]
}
```

## 十二、备份与恢复

### 12.1 快照(Snapshot)

**注册快照仓库：**
```json
PUT /_snapshot/my_backup
{
  "type": "fs",
  "settings": {
    "location": "/mount/backups/my_backup",
    "compress": true
  }
}
```

**创建快照：**
```json
// 备份所有索引
PUT /_snapshot/my_backup/snapshot_1
{
  "indices": "*",
  "ignore_unavailable": true,
  "include_global_state": true
}

// 备份指定索引
PUT /_snapshot/my_backup/snapshot_2
{
  "indices": "index_1,index_2",
  "ignore_unavailable": true,
  "include_global_state": false
}
```

**自动快照策略(SLM)：**
```json
PUT /_slm/policy/daily_backup
{
  "schedule": "0 30 1 * * ?",
  "name": "<daily-snap-{now/d}>",
  "repository": "my_backup",
  "config": {
    "indices": ["*"],
    "ignore_unavailable": true,
    "include_global_state": true
  },
  "retention": {
    "expire_after": "30d",
    "min_count": 5,
    "max_count": 50
  }
}
```

### 12.2 恢复数据

**查看快照：**
```json
GET /_snapshot/my_backup/_all
GET /_snapshot/my_backup/snapshot_1
```

**恢复快照：**
```json
// 恢复所有索引
POST /_snapshot/my_backup/snapshot_1/_restore

// 恢复指定索引
POST /_snapshot/my_backup/snapshot_1/_restore
{
  "indices": "index_1,index_2",
  "ignore_unavailable": true,
  "include_global_state": false,
  "rename_pattern": "index_(.+)",
  "rename_replacement": "restored_index_$1"
}
```

**监控恢复进度：**
```json
GET /_recovery?human
GET /index_1/_recovery
```

## 十三、常见故障排查

### 13.1 集群Red状态

**诊断步骤：**

1. **查看未分配的分片：**
```json
GET /_cat/shards?v&h=index,shard,prirep,state,unassigned.reason
```

2. **查看分片分配失败原因：**
```json
GET /_cluster/allocation/explain
{
  "index": "my_index",
  "shard": 0,
  "primary": true
}
```

3. **常见原因和解决方案：**
   - **磁盘空间不足**:清理数据或增加存储
   - **分片损坏**:从快照恢复
   - **节点宕机**:重启节点
   - **配置错误**:检查elasticsearch.yml

### 13.2 内存问题

**堆内存溢出(OOM)：**

**检查堆内存使用：**
```json
GET /_cat/nodes?v&h=name,heap.percent,heap.current,heap.max,ram.percent
GET /_nodes/stats/jvm
```

**解决方案：**
- 增加JVM堆内存(不超过32GB)
- 减少查询复杂度
- 增加节点数量
- 优化映射和查询

**配置堆内存：**
```bash
# jvm.options
-Xms16g
-Xmx16g
```

### 13.3 查询性能问题

**慢查询诊断：**

1. **启用慢查询日志：**
```json
PUT /my_index/_settings
{
  "index.search.slowlog.threshold.query.warn": "2s",
  "index.search.slowlog.threshold.query.info": "1s",
  "index.search.slowlog.threshold.fetch.warn": "500ms"
}
```

2. **使用Profile API：**
```json
GET /my_index/_search
{
  "profile": true,
  "query": { ... }
}
```

3. **优化建议：**
   - 使用filter代替query(可缓存)
   - 避免使用wildcard和regex查询
   - 减少返回字段(_source filtering)
   - 使用批量查询
   - 增加副本分片数

### 13.4 写入性能问题

**诊断指标：**
```json
GET /_cat/thread_pool/write?v
GET /_nodes/stats/indices/indexing
```

**优化措施：**
- 使用Bulk API批量写入
- 增加refresh_interval
- 临时禁用副本
- 增加索引缓冲区大小
- 优化分片数量

**缓冲区配置：**
```json
PUT /_cluster/settings
{
  "transient": {
    "indices.memory.index_buffer_size": "20%"
  }
}
```

## 十四、版本升级注意事项

### 14.1 滚动升级

**升级流程：**

1. **禁用分片分配：**
```json
PUT /_cluster/settings
{
  "persistent": {
    "cluster.routing.allocation.enable": "primaries"
  }
}
```

2. **停止一个节点并升级**
```bash
sudo systemctl stop elasticsearch
# 升级软件包
sudo systemctl start elasticsearch
```

3. **等待节点加入集群并恢复：**
```json
GET /_cat/health?v
```

4. **重新启用分片分配：**
```json
PUT /_cluster/settings
{
  "persistent": {
    "cluster.routing.allocation.enable": "all"
  }
}
```

5. **重复步骤2-4升级其他节点**

### 14.2 重要变更

**7.x主要变更：**
- 移除Type概念
- 默认主分片数改为1
- 引入Sequence ID和Global Checkpoint
- 废弃Translog的sync_interval设置

**8.x主要变更：**
- 完全移除Type
- 安全特性默认启用
- 移除Joda Time，使用Java Time
- 改进的搜索性能

## 十五、总结

Elasticsearch作为一个强大的分布式搜索和分析引擎，其核心概念包括：

**数据组织：**
- 集群(Cluster)→ 节点(Node)→ 索引(Index)→ 文档(Document)→ 字段(Field)

**数据分布：**
- 分片(Shard)：主分片和副本分片
- 路由(Routing)：文档到分片的映射
- 副本(Replica)：数据冗余和高可用

**数据处理：**
- 映射(Mapping)：定义字段类型和索引方式
- 分析器(Analyzer)：文本分析和分词
- 倒排索引(Inverted Index)：快速全文搜索的核心

**性能优化：**
- 合理设计分片数量
- 使用合适的映射类型
- 优化查询和写入策略
- 实施生命周期管理

**运维管理：**
- 监控集群健康
- 定期备份数据
- 安全访问控制
- 及时处理故障

掌握这些基本概念是高效使用 Elasticsearch 的基础，在实际应用中需要根据具体场景灵活运用这些知识，并结合实践不断优化。

## 延伸阅读

本文作为 Elasticsearch 入门文章，对核心概念进行了全面介绍。以下专题文章对特定主题进行了深入探讨，建议按需阅读：

| 主题 | 文章 | 核心内容 |
|------|------|----------|
| 分片规划 | [Elasticsearch索引分片规划与主分片不可变原理](./Elasticsearch索引分片规划与主分片不可变原理.md) | 主分片不可变的深层原理、分片数量选择策略、Split/Shrink API |
| 写入优化 | [Elasticsearch写入查询流程与refresh_interval调优](./Elasticsearch写入查询流程与refresh_interval调优.md) | 写入流程详解、Refresh/Translog 机制、性能调优实践 |
| 集群运维 | [Elasticsearch集群黄色红色状态排查与恢复](./Elasticsearch集群黄色红色状态排查与恢复.md) | Yellow/Red 状态排查、分片分配失败分析、恢复方案 |
| 架构设计 | [Elasticsearch冷热数据分层架构设计](./Elasticsearch冷热数据分层架构设计.md) | Hot-Warm-Cold 架构、节点角色配置、ILM 与冷热分层结合 |

## 参考资源

- [Elasticsearch 官方文档](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html)
- [Elasticsearch 权威指南](https://www.elastic.co/guide/en/elasticsearch/guide/current/index.html)
- [Elasticsearch 最佳实践](https://www.elastic.co/guide/en/elasticsearch/reference/current/best-practices.html)
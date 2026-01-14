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

## 一、Elasticsearch简介

Elasticsearch 是一个基于 Apache Lucene 构建的开源、分布式、RESTful 搜索和分析引擎。它能够快速地存储、搜索和分析海量数据,广泛应用于全文搜索、日志分析、实时数据分析等场景。

### 核心特性

- **分布式架构**:天然支持集群部署,可横向扩展
- **近实时搜索**:文档从索引到可搜索通常只需1秒
- **RESTful API**:通过 HTTP 请求进行所有操作
- **多租户支持**:一个集群可以包含多个索引
- **Schema-free**:支持动态映射,无需预定义结构

## 二、核心概念详解

### 2.1 集群(Cluster)

集群是一个或多个节点的集合,这些节点共同保存所有数据并提供跨所有节点的联合索引和搜索功能。

**关键特性:**
- 每个集群有唯一的名称标识(默认为"elasticsearch")
- 节点通过集群名称加入集群
- 一个集群可以只有一个节点

**配置示例:**
```yaml
cluster.name: my-application
```

### 2.2 节点(Node)

节点是集群中的单个服务器实例,存储数据并参与集群的索引和搜索功能。

**节点类型:**

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

**配置示例:**
```yaml
node.name: node-1
node.master: true
node.data: true
node.ingest: false
```

### 2.3 索引(Index)

索引是具有相似特征的文档集合,类似于关系数据库中的"数据库"。

**特点:**
- 索引名称必须全部小写
- 一个索引可以包含多个类型(7.x后废弃多类型)
- 通过索引名称进行文档的增删改查

**命名规范:**
- 小写字母
- 不能包含`\`、`/`、`*`、`?`、`"`、`<`、`>`、`|`、空格、`,`、`#`
- 不能以`-`、`_`、`+`开头
- 不能是`.`或`..`

**创建索引示例:**
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

### 2.4 类型(Type)

类型是索引内的逻辑分类,类似于关系数据库中的"表"。

**重要变更:**
- Elasticsearch 6.x:一个索引只能有一个类型
- Elasticsearch 7.x:类型被废弃
- Elasticsearch 8.x:完全移除类型

**迁移建议:**
- 使用独立的索引代替类型
- 或在文档中添加type字段进行区分

### 2.5 文档(Document)

文档是可以被索引的基本信息单元,以JSON格式表示,类似于关系数据库中的"行"。

**文档特性:**
- 每个文档有唯一的ID(可自定义或自动生成)
- 文档是自包含的:包含字段和值
- 文档可以是层次化的:字段值可以是子文档

**文档示例:**
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

分片是索引的物理分区,每个分片本身就是一个功能完整且独立的"索引"。

**为什么需要分片:**
- 水平分割/扩展内容容量
- 分布式并行操作,提高性能和吞吐量

**分片类型:**

1. **主分片(Primary Shard)**
   - 索引创建时指定数量,后续不可更改(7.x后可通过Split API调整)
   - 每个文档只存在于一个主分片
   - 默认5个(7.x后改为1个)

2. **副本分片(Replica Shard)**
   - 主分片的副本,提供数据冗余和查询性能
   - 可以随时调整数量
   - 不会与对应的主分片分配在同一节点

**配置示例:**
```json
PUT /my_index
{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 2
  }
}
```

**调整副本数:**
```json
PUT /my_index/_settings
{
  "number_of_replicas": 1
}
```

### 2.7 映射(Mapping)

映射定义了文档及其包含的字段如何存储和索引,类似于关系数据库中的"schema"。

**映射类型:**

1. **动态映射(Dynamic Mapping)**
   - Elasticsearch自动推断字段类型
   - 适用于快速开发,但可能不够精确

2. **显式映射(Explicit Mapping)**
   - 手动定义字段类型
   - 提供更精确的控制

**常用字段类型:**

**基本类型:**
- `text`:全文本,会被分词
- `keyword`:精确值,不分词
- `long`、`integer`、`short`、`byte`:整数
- `double`、`float`:浮点数
- `boolean`:布尔值
- `date`:日期类型
- `binary`:二进制

**复杂类型:**
- `object`:JSON对象
- `nested`:嵌套对象数组
- `geo_point`:地理位置点
- `geo_shape`:地理位置形状
- `ip`:IP地址

**映射示例:**
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

**动态映射配置:**
```json
PUT /my_index
{
  "mappings": {
    "dynamic": "strict",  // strict: 拒绝未知字段, true: 自动添加, false: 忽略
    "properties": {
      "name": { "type": "text" }
    }
  }
}
```

### 2.8 分析器(Analyzer)

分析器用于文本分析,将文本转换为倒排索引中的词条(term)。

**分析器组成:**
1. **字符过滤器(Character Filter)**:处理原始文本(如去除HTML标签)
2. **分词器(Tokenizer)**:将文本分割成词条
3. **词条过滤器(Token Filter)**:处理词条(如转小写、去除停用词)

**内置分析器:**
- `standard`:默认分析器,按词分割,小写处理
- `simple`:按非字母字符分割,小写处理
- `whitespace`:按空格分割
- `stop`:类似simple,但会去除停用词
- `keyword`:不分词,整个文本作为单个词条
- `pattern`:使用正则表达式分割
- `language`:特定语言分析器(如`english`、`chinese`)

**中文分析器:**
- IK分词器(需安装插件)
  - `ik_smart`:粗粒度分词
  - `ik_max_word`:细粒度分词

**自定义分析器示例:**
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

**测试分析器:**
```json
POST /my_index/_analyze
{
  "analyzer": "standard",
  "text": "The Quick Brown Fox"
}
```

### 2.9 倒排索引(Inverted Index)

倒排索引是Elasticsearch实现快速全文搜索的核心数据结构。

**原理:**
- 传统索引:文档ID → 内容
- 倒排索引:词条 → 文档ID列表

**示例:**

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

**倒排索引组成:**
1. **词条字典(Term Dictionary)**:所有文档的词条集合
2. **倒排列表(Posting List)**:词条对应的文档ID列表及其他信息

**附加信息:**
- 词频(TF):词条在文档中出现的次数
- 位置(Position):词条在文档中的位置
- 偏移(Offset):词条的字符偏移量

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

**关键区别:**
- ES是文档型数据库,关系数据库是关系型
- ES擅长全文搜索,关系数据库擅长事务处理
- ES是Schema-free,关系数据库是Schema-based
- ES的JOIN能力有限,关系数据库支持复杂JOIN

## 四、数据的读写流程

### 4.1 写入流程

1. **客户端发送请求**:写请求可以发送到任意节点
2. **协调节点处理**:接收请求的节点成为协调节点,根据文档ID计算应该存储的分片
3. **路由到主分片**:协调节点将请求转发到对应的主分片所在节点
4. **主分片处理**:主分片执行写操作
5. **同步到副本**:主分片并行地将操作转发到所有副本分片
6. **返回结果**:所有副本确认后,主分片向协调节点报告成功,协调节点向客户端返回结果

**路由公式:**
```
shard = hash(routing) % number_of_primary_shards
```

其中routing默认为文档ID,这就是为什么主分片数量创建后不能修改的原因。

**写入性能优化:**
- 批量写入(Bulk API)
- 调整refresh_interval
- 增加副本前先写入数据
- 优化分片数量

### 4.2 读取流程

**通过ID查询:**
1. 客户端发送GET请求到任意节点
2. 协调节点根据文档ID计算分片位置
3. 协调节点使用轮询策略选择主分片或副本分片
4. 目标分片返回文档
5. 协调节点返回给客户端

**搜索查询:**
1. **Query阶段**:
   - 客户端发送搜索请求到协调节点
   - 协调节点广播请求到所有相关分片(主分片或副本)
   - 每个分片本地执行查询,返回文档ID和排序值
   - 协调节点合并结果,排序并选择top N

2. **Fetch阶段**:
   - 协调节点根据文档ID到对应分片获取完整文档
   - 返回最终结果给客户端

**搜索类型:**
- `query_then_fetch`:默认,先查询后获取
- `dfs_query_then_fetch`:改进相关性评分的查询方式

### 4.3 更新和删除流程

**更新流程:**
1. 检索完整文档
2. 修改文档内容
3. 删除旧文档(标记删除)
4. 索引新文档

**删除流程:**
1. 不会立即物理删除
2. 标记为已删除(.del文件)
3. 段合并时真正删除

## 五、近实时搜索原理

### 5.1 Refresh机制

**Refresh过程:**
1. 新文档首先写入内存缓冲区(In-memory Buffer)
2. 每隔1秒(默认),缓冲区内容写入新的Segment
3. Segment被打开,变为可搜索状态
4. 清空缓冲区

这就是为什么ES是"近实时"的原因,默认有最多1秒的延迟。

**配置refresh_interval:**
```json
PUT /my_index/_settings
{
  "refresh_interval": "30s"  // 30秒刷新一次
}
```

**手动刷新:**
```json
POST /my_index/_refresh
```

### 5.2 Translog机制

为了防止数据丢失,ES使用Translog(事务日志):

**写入过程:**
1. 文档写入内存缓冲区
2. 同时追加到translog
3. 每隔5秒或每次写请求完成后,translog fsync到磁盘

**Flush过程:**
1. 所有内存缓冲区文档写入新的Segment
2. 清空缓冲区
3. 写入commit point(记录所有Segment信息)
4. 文件系统缓存fsync到磁盘
5. 删除旧的translog

**默认flush策略:**
- translog大小达到512MB
- 或每30分钟

### 5.3 Segment Merging

随着时间推移,会产生大量小Segment,影响性能。

**合并策略:**
- 后台自动合并小Segment为大Segment
- 删除已标记删除的文档
- 减少Segment数量,提高查询性能

**手动触发合并:**
```json
POST /my_index/_forcemerge?max_num_segments=1
```

## 六、集群健康与状态

### 6.1 集群健康状态

**三种状态:**
- **Green(绿色)**:所有主分片和副本分片都正常
- **Yellow(黄色)**:所有主分片正常,但部分副本分片不可用
- **Red(红色)**:部分主分片不可用

**查看集群健康:**
```json
GET /_cluster/health
```

**响应示例:**
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

### 6.2 分片分配

**分片分配原则:**
- 主分片和副本分片不在同一节点
- 同一索引的分片尽量均匀分布
- 考虑节点的磁盘使用率

**分片分配设置:**
```json
PUT /_cluster/settings
{
  "transient": {
    "cluster.routing.allocation.enable": "all"  // all, primaries, new_primaries, none
  }
}
```

## 七、性能优化建议

### 7.1 索引设计优化

**分片数量:**
- 分片不是越多越好
- 单个分片建议20-40GB
- 每个节点的分片数不超过20个/GB堆内存

**副本策略:**
- 生产环境至少1个副本
- 读多写少:增加副本数
- 写多读少:减少副本数

**Mapping优化:**
- 明确定义Mapping,避免动态映射
- 不需要搜索的字段设置`"index": false`
- 不需要评分的字段使用`keyword`类型
- 关闭不需要的字段特性(doc_values、norms等)

### 7.2 写入性能优化

**批量操作:**
```json
POST /_bulk
{ "index": { "_index": "my_index", "_id": "1" } }
{ "title": "Document 1" }
{ "index": { "_index": "my_index", "_id": "2" } }
{ "title": "Document 2" }
```

**其他优化:**
- 增大`refresh_interval`
- 写入时禁用副本,写入完成后再启用
- 增大`index.translog.flush_threshold_size`
- 使用自动生成的ID而不是自定义ID
- 调整JVM堆内存(建议不超过32GB)

### 7.3 查询性能优化

**索引优化:**
- 使用Filter Context代替Query Context(可缓存)
- 避免深度分页
- 使用scroll或search_after代替from/size
- 预先过滤数据,减少搜索范围

**查询示例:**
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

## 八、常见应用场景

### 8.1 全文搜索

**电商搜索示例:**
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

### 8.2 日志分析

ELK Stack(Elasticsearch + Logstash + Kibana):
- Logstash:收集和处理日志
- Elasticsearch:存储和搜索日志
- Kibana:可视化展示

**日志查询示例:**
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

### 8.3 实时数据分析

**聚合分析示例:**
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

## 九、常见问题

### 1. 为什么主分片数量创建后不能修改?

**原因:**
文档路由到分片使用公式:`shard = hash(routing) % number_of_primary_shards`

如果改变主分片数量,已有文档的路由会改变,导致无法找到原有数据。

**解决方案:**
- Elasticsearch 7.x引入了Split API和Shrink API
- Split API:增加分片数(只能增加到原来的倍数)
- Shrink API:减少分片数(新分片数必须是原分片数的因子)
- 更常见的方案是Reindex到新索引

```json
POST /my_index/_split/new_index
{
  "settings": {
    "index.number_of_shards": 6  // 原来是3个
  }
}
```

### 2. Text和Keyword类型有什么区别?

**Text类型:**
- 用于全文搜索
- 会被分词器分析
- 不能用于排序和聚合
- 适用场景:文章内容、产品描述

**Keyword类型:**
- 用于精确匹配
- 不分词,整个值作为一个词条
- 可用于排序、聚合和过滤
- 适用场景:邮箱、状态码、标签、ID

**示例对比:**
```json
// Text字段
"content": {
  "type": "text",
  "analyzer": "standard"
}
// "Hello World" → ["hello", "world"]

// Keyword字段  
"status": {
  "type": "keyword"
}
// "Hello World" → ["Hello World"]
```

**多字段类型(fields):**
```json
"title": {
  "type": "text",
  "fields": {
    "keyword": {
      "type": "keyword"
    }
  }
}
// 可以用 title 进行全文搜索,用 title.keyword 进行精确匹配和聚合
```

### 3. 如何避免深度分页问题?

**深度分页问题:**
当使用`from + size`进行深度分页时,协调节点需要从每个分片获取`from + size`个文档,然后排序,性能极差。

例如:获取第10000页,每页10条,需要从每个分片获取100010个文档。

**解决方案:**

**方案1: Scroll API(快照查询)**
```json
// 初始化scroll
POST /my_index/_search?scroll=1m
{
  "size": 100,
  "query": { "match_all": {} }
}

// 继续获取
POST /_search/scroll
{
  "scroll": "1m",
  "scroll_id": "scroll_id_from_previous_response"
}
```

**适用场景:**
- 导出全部数据
- 不适合实时查询(数据快照)

**方案2: Search After(推荐)**
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

// 下一页(使用上一页最后一个文档的sort值)
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
```

**适用场景:**
- 实时查询
- 下一页/上一页导航
- 不支持跳页

**方案3: 限制分页深度**
```json
PUT /my_index/_settings
{
  "index.max_result_window": 10000  // 默认10000
}
```

### 4. 集群状态为Yellow的常见原因和解决方法?

**Yellow状态表示:**所有主分片正常,但部分副本分片未分配。

**常见原因:**

**原因1: 单节点集群**
- 副本分片不能和主分片在同一节点
- 单节点无法分配副本

**解决方法:**
```json
// 临时方案:将副本数设为0
PUT /my_index/_settings
{
  "number_of_replicas": 0
}

// 长期方案:添加节点到集群
```

**原因2: 节点磁盘空间不足**
- 默认磁盘使用率超过85%时停止分配副本

**解决方法:**
```json
// 查看磁盘使用情况
GET /_cat/allocation?v

// 清理数据或增加磁盘空间
// 调整阈值(不推荐)
PUT /_cluster/settings
{
  "transient": {
    "cluster.routing.allocation.disk.watermark.low": "90%",
    "cluster.routing.allocation.disk.watermark.high": "95%"
  }
}
```

**原因3: 分片分配被禁用**
```json
// 检查分配设置
GET /_cluster/settings

// 启用分配
PUT /_cluster/settings
{
  "transient": {
    "cluster.routing.allocation.enable": "all"
  }
}
```

### 5. 如何选择合适的分片数量?

**分片数量影响因素:**

**单个分片大小建议:**
- 日志/时序数据:20-40GB
- 搜索场景:10-30GB
- 不要超过50GB

**计算方法:**
```
分片数 = 预估数据量 / 单个分片目标大小
```

**示例:**
- 预计存储500GB数据
- 单个分片目标25GB
- 主分片数 = 500GB / 25GB ≈ 20

**其他考虑因素:**

**节点数量:**
- 分片数应该是节点数的整数倍,便于均衡分布
- 每个节点的分片数不超过 `20 × 堆内存(GB)`

**查询性能:**
- 分片过多:查询需要合并更多结果,协调开销大
- 分片过少:无法充分利用集群资源

**建议:**
- 开始时使用较少的分片(3-5个)
- 根据实际数据量和性能调整
- 使用Index Lifecycle Management(ILM)管理索引生命周期
- 时序数据使用滚动索引(如按天创建索引)

**时序数据示例:**
```
logs-2024-01-01  (3 shards)
logs-2024-01-02  (3 shards)
logs-2024-01-03  (3 shards)
```

每个索引分片较少,但通过多个索引实现水平扩展。

## 十、实战最佳实践

### 10.1 索引命名规范

**推荐命名模式:**
```
{业务类型}-{环境}-{日期}
例如:
- user-prod-2024-01
- order-dev-2024-01-15
- logs-staging-2024-w03
```

**优势:**
- 便于管理和维护
- 支持通配符查询
- 便于实施生命周期管理

### 10.2 索引模板(Index Template)

索引模板可以在创建新索引时自动应用预定义的设置和映射。

**创建索引模板:**
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

**组件模板(Component Template):**
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

**生命周期阶段:**
- **Hot**:频繁写入和查询
- **Warm**:不再写入,仍需查询
- **Cold**:很少查询,可压缩存储
- **Frozen**:极少访问,最小资源占用
- **Delete**:删除索引

**ILM策略示例:**
```json
PUT /_ilm/policy/logs_policy
{
  "policy": {
    "phases": {
      "hot": {
        "actions": {
          "rollover": {
            "max_size": "50GB",
            "max_age": "7d",
            "max_docs": 10000000
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
          "set_priority": {
            "priority": 50
          }
        }
      },
      "cold": {
        "min_age": "30d",
        "actions": {
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

**应用ILM策略:**
```json
PUT /logs-000001
{
  "settings": {
    "index.lifecycle.name": "logs_policy",
    "index.lifecycle.rollover_alias": "logs"
  },
  "aliases": {
    "logs": {
      "is_write_index": true
    }
  }
}
```

### 10.4 别名(Alias)使用

别名是索引的逻辑名称,可以指向一个或多个索引。

**创建别名:**
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

**带过滤的别名:**
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

**零停机重建索引:**
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

**集群状态监控:**
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

**性能分析:**
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

**热点线程分析:**
```json
GET /_nodes/hot_threads
```

## 十一、安全性配置

### 11.1 安全基础

**启用安全特性(X-Pack Security):**
```yaml
# elasticsearch.yml
xpack.security.enabled: true
xpack.security.transport.ssl.enabled: true
xpack.security.http.ssl.enabled: true
```

**创建用户:**
```bash
# 设置内置用户密码
./bin/elasticsearch-setup-passwords interactive

# 或自动生成密码
./bin/elasticsearch-setup-passwords auto
```

### 11.2 角色和权限

**创建角色:**
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

**创建用户并分配角色:**
```json
POST /_security/user/log_user
{
  "password": "secure_password",
  "roles": ["logs_reader"],
  "full_name": "Log Reader User"
}
```

### 11.3 字段级别安全

**限制字段访问:**
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

**基于查询的访问控制:**
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

**注册快照仓库:**
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

**创建快照:**
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

**自动快照策略(SLM):**
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

**查看快照:**
```json
GET /_snapshot/my_backup/_all
GET /_snapshot/my_backup/snapshot_1
```

**恢复快照:**
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

**监控恢复进度:**
```json
GET /_recovery?human
GET /index_1/_recovery
```

## 十三、常见故障排查

### 13.1 集群Red状态

**诊断步骤:**

1. **查看未分配的分片:**
```json
GET /_cat/shards?v&h=index,shard,prirep,state,unassigned.reason
```

2. **查看分片分配失败原因:**
```json
GET /_cluster/allocation/explain
{
  "index": "my_index",
  "shard": 0,
  "primary": true
}
```

3. **常见原因和解决方案:**
   - **磁盘空间不足**:清理数据或增加存储
   - **分片损坏**:从快照恢复
   - **节点宕机**:重启节点
   - **配置错误**:检查elasticsearch.yml

### 13.2 内存问题

**堆内存溢出(OOM):**

**检查堆内存使用:**
```json
GET /_cat/nodes?v&h=name,heap.percent,heap.current,heap.max,ram.percent
GET /_nodes/stats/jvm
```

**解决方案:**
- 增加JVM堆内存(不超过32GB)
- 减少查询复杂度
- 增加节点数量
- 优化映射和查询

**配置堆内存:**
```bash
# jvm.options
-Xms16g
-Xmx16g
```

### 13.3 查询性能问题

**慢查询诊断:**

1. **启用慢查询日志:**
```json
PUT /my_index/_settings
{
  "index.search.slowlog.threshold.query.warn": "2s",
  "index.search.slowlog.threshold.query.info": "1s",
  "index.search.slowlog.threshold.fetch.warn": "500ms"
}
```

2. **使用Profile API:**
```json
GET /my_index/_search
{
  "profile": true,
  "query": { ... }
}
```

3. **优化建议:**
   - 使用filter代替query(可缓存)
   - 避免使用wildcard和regex查询
   - 减少返回字段(_source filtering)
   - 使用批量查询
   - 增加副本分片数

### 13.4 写入性能问题

**诊断指标:**
```json
GET /_cat/thread_pool/write?v
GET /_nodes/stats/indices/indexing
```

**优化措施:**
- 使用Bulk API批量写入
- 增加refresh_interval
- 临时禁用副本
- 增加索引缓冲区大小
- 优化分片数量

**缓冲区配置:**
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

**升级流程:**

1. **禁用分片分配:**
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

3. **等待节点加入集群并恢复:**
```json
GET /_cat/health?v
```

4. **重新启用分片分配:**
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

**7.x主要变更:**
- 移除Type概念
- 默认主分片数改为1
- 引入Sequence ID和Global Checkpoint
- 废弃Translog的sync_interval设置

**8.x主要变更:**
- 完全移除Type
- 安全特性默认启用
- 移除Joda Time,使用Java Time
- 改进的搜索性能

## 十五、总结

Elasticsearch作为一个强大的分布式搜索和分析引擎,其核心概念包括:

**数据组织:**
- 集群(Cluster)→ 节点(Node)→ 索引(Index)→ 文档(Document)→ 字段(Field)

**数据分布:**
- 分片(Shard):主分片和副本分片
- 路由(Routing):文档到分片的映射
- 副本(Replica):数据冗余和高可用

**数据处理:**
- 映射(Mapping):定义字段类型和索引方式
- 分析器(Analyzer):文本分析和分词
- 倒排索引(Inverted Index):快速全文搜索的核心

**性能优化:**
- 合理设计分片数量
- 使用合适的映射类型
- 优化查询和写入策略
- 实施生命周期管理

**运维管理:**
- 监控集群健康
- 定期备份数据
- 安全访问控制
- 及时处理故障

掌握这些基本概念是高效使用Elasticsearch的基础,在实际应用中需要根据具体场景灵活运用这些知识,并结合实践不断优化。
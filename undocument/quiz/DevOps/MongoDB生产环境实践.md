---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Database
tag:
  - MongoDB
  - Database
  - NoSQL
  - DevOps
---

# MongoDB 生产环境实践

当你的应用从关系型数据库迁移到文档数据库时,MongoDB 的灵活 schema 设计和水平扩展能力成为关键优势。但 MongoDB 并不是"存进去就能快"——分片策略、索引设计、读写分离、监控告警都需要深入理解才能在生产环境稳定运行。一个未经优化的 MongoDB 集群,可能在千万级文档时就出现查询缓慢、分片不均衡、内存耗尽等问题。

本文将从集群架构、分片策略、索引优化、性能监控、备份恢复五个维度,系统梳理 MongoDB 生产环境的实践经验。

## 一、集群架构设计

### 副本集架构

MongoDB 的副本集(Replica Set)提供数据冗余和高可用:

```
┌─────────────────────────────────────────────────────────────┐
│                    MongoDB 副本集架构                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Primary (主节点)                                     │  │
│  │  - 接收所有写入                                       │  │
│  │  - Oplog 同步到 Secondary                            │  │
│  │  - 选举投票                                           │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│          ┌──────────┼──────────┐                           │
│          │          │          │                           │
│          ▼          ▼          ▼                           │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐             │
│  │Secondary 1 │ │Secondary 2 │ │  Arbiter   │             │
│  │ (数据节点) │ │ (数据节点) │ │ (投票节点) │             │
│  │ - 只读查询 │ │ - 只读查询 │ │ - 仅投票   │             │
│  │ - 故障转移 │ │ - 故障转移 │ │ - 无数据   │             │
│  └────────────┘ └────────────┘ └────────────┘             │
│                                                              │
│  选举机制:                                                   │
│  1. Primary 故障后,Secondary 发起选举                       │
│  2. 获得大多数投票的 Secondary 成为新 Primary                │
│  3. 客户端自动连接到新 Primary                              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**副本集配置**:

```javascript
// 初始化副本集
rs.initiate({
  _id: "rs0",
  members: [
    { _id: 0, host: "mongo1:27017", priority: 2 },
    { _id: 1, host: "mongo2:27017", priority: 1 },
    { _id: 2, host: "mongo3:27017", priority: 1 }
  ]
})

// 查看副本集状态
rs.status()

// 查看副本集配置
rs.conf()

// 添加仲裁节点
rs.addArb("mongo-arb:27017")

// 移除节点
rs.remove("mongo3:27017")
```

**读写分离配置**:

```javascript
// 连接字符串
mongodb://user:password@mongo1:27017,mongo2:27017,mongo3:27017/production?replicaSet=rs0&readPreference=secondaryPreferred

// Node.js 驱动配置
const { MongoClient } = require('mongodb');

const client = new MongoClient('mongodb://mongo1:27017,mongo2:27017,mongo3:27017', {
  replicaSet: 'rs0',
  readPreference: 'secondaryPreferred',  // 优先从从节点读取
  writeConcern: 'majority',  // 写入大多数节点
  maxPoolSize: 100,
  minPoolSize: 10
});

// 读偏好选项
// primary: 只从主节点读取(默认)
// primaryPreferred: 优先主节点,主节点不可用时从从节点读取
// secondary: 只从从节点读取
// secondaryPreferred: 优先从节点,从节点不可用时从主节点读取
// nearest: 从网络延迟最低的节点读取
```

**写关注配置**:

```javascript
// 写关注级别
// w: 0 - 不确认写入(最快,可能丢数据)
// w: 1 - 确认主节点写入(默认)
// w: majority - 确认大多数节点写入(推荐生产环境)
// w: 2 - 确认 2 个节点写入

// 插入数据时指定写关注
db.orders.insertOne(
  { order_id: "12345", user_id: "user123" },
  { writeConcern: { w: "majority", j: true, wtimeout: 5000 } }
)

// j: true - 确保写入 journal 日志(持久化)
// wtimeout: 超时时间(毫秒)
```

### 分片集群架构

分片集群(Sharded Cluster)实现水平扩展:

```
┌─────────────────────────────────────────────────────────────┐
│                    MongoDB 分片集群架构                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  mongos (路由节点)                                    │  │
│  │  - 路由查询到对应分片                                  │  │
│  │  - 合并查询结果                                       │  │
│  │  - 负载均衡                                           │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│                     ▼                                       │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Config Servers (配置服务器)                          │  │
│  │  - 存储集群元数据                                     │  │
│  │  - 分片范围信息                                       │  │
│  │  - 副本集部署                                         │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│          ┌──────────┼──────────┐                           │
│          │          │          │                           │
│          ▼          ▼          ▼                           │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐             │
│  │  Shard 1   │ │  Shard 2   │ │  Shard 3   │             │
│  │ (副本集)   │ │ (副本集)   │ │ (副本集)   │             │
│  │ user_id:   │ │ user_id:   │ │ user_id:   │             │
│  │ 0-100万    │ │ 100万-200万│ │ 200万-300万│             │
│  └────────────┘ └────────────┘ └────────────┘             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**分片集群配置**:

```bash
# 1. 启动 Config Server
mongod --configsvr --replSet configRs --dbpath /data/config --port 27019

# 2. 初始化 Config Server
mongo --port 27019
rs.initiate({
  _id: "configRs",
  configsvr: true,
  members: [
    { _id: 0, host: "config1:27019" },
    { _id: 1, host: "config2:27019" },
    { _id: 2, host: "config3:27019" }
  ]
})

# 3. 启动 Shard
mongod --shardsvr --replSet shard1 --dbpath /data/shard1 --port 27018

# 4. 初始化 Shard
mongo --port 27018
rs.initiate({
  _id: "shard1",
  members: [
    { _id: 0, host: "shard1-1:27018" },
    { _id: 1, host: "shard1-2:27018" },
    { _id: 2, host: "shard1-3:27018" }
  ]
})

# 5. 启动 mongos
mongos --configdb configRs/config1:27019,config2:27019,config3:27019 --port 27017

# 6. 添加分片
mongo --port 27017
sh.addShard("shard1/shard1-1:27018,shard1-2:27018,shard1-3:27018")
sh.addShard("shard2/shard2-1:27018,shard2-2:27018,shard2-3:27018")
```

**启用分片**:

```javascript
// 启用数据库分片
sh.enableSharding("production")

// 对集合分片
sh.shardCollection("production.orders", { user_id: 1 })

// 查看分片状态
sh.status()

// 查看分片分布
db.orders.getShardDistribution()
```

## 二、分片策略

### 分片键选择

分片键决定了数据分布和查询性能:

```
┌─────────────────────────────────────────────────────────────┐
│                    分片键选择原则                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 高基数(High Cardinality)                                 │
│     - 分片键值越多,数据分布越均匀                            │
│     - 避免: 性别、状态等低基数字段                           │
│     - 推荐: user_id、order_id 等高基数字段                   │
│                                                              │
│  2. 查询模式(Query Pattern)                                  │
│     - 分片键应包含在常用查询条件中                           │
│     - 路由查询: 查询条件包含分片键,直接定位分片              │
│     - 广播查询: 查询条件不包含分片键,查询所有分片            │
│                                                              │
│  3. 写入分布(Write Distribution)                             │
│     - 写入应均匀分布到所有分片                               │
│     - 避免: 单调递增的分片键(如时间戳)                       │
│     - 推荐: 哈希分片键或复合分片键                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**分片键类型**:

| 类型 | 说明 | 适用场景 | 优点 | 缺点 |
|------|------|---------|------|------|
| 范围分片 | 按分片键值范围划分 | 范围查询 | 范围查询高效 | 数据分布不均 |
| 哈希分片 | 按分片键哈希值划分 | 随机查询 | 数据分布均匀 | 范围查询低效 |
| 复合分片 | 多个字段组合 | 复杂查询 | 灵活 | 配置复杂 |

**范围分片示例**:

```javascript
// 范围分片
sh.shardCollection("production.orders", { user_id: 1 })

// 查看分片范围
db.orders.getShardDistribution()

// 手动分割 chunk
sh.splitAt("production.orders", { user_id: 1000000 })
sh.splitAt("production.orders", { user_id: 2000000 })

// 移动 chunk
sh.moveChunk("production.orders", { user_id: 1500000 }, "shard2")
```

**哈希分片示例**:

```javascript
// 哈希分片
sh.shardCollection("production.orders", { user_id: "hashed" })

// 哈希分片适合随机写入,数据分布均匀
// 但不支持范围查询
```

**复合分片示例**:

```javascript
// 复合分片键
sh.shardCollection("production.orders", { user_id: 1, created_at: 1 })

// 查询条件必须包含分片键前缀
db.orders.find({ user_id: 123 })  // 路由查询
db.orders.find({ user_id: 123, created_at: { $gt: ISODate("2026-01-01") } })  // 路由查询
db.orders.find({ created_at: { $gt: ISODate("2026-01-01") } })  // 广播查询
```

### Chunk 管理与均衡

**Chunk 大小配置**:

```javascript
// 查看默认 chunk 大小(默认 128MB)
use config
db.settings.find({ _id: "chunksize" })

// 修改 chunk 大小
db.settings.save({ _id: "chunksize", value: 64 })  // 64MB
```

**均衡器配置**:

```javascript
// 查看均衡器状态
sh.getBalancerState()

// 停止均衡器
sh.stopBalancer()

// 启动均衡器
sh.startBalancer()

// 设置均衡窗口
db.settings.update(
  { _id: "balancer" },
  { $set: { activeWindow: { start: "02:00", stop: "06:00" } } },
  { upsert: true }
)

// 查看均衡状态
db.locks.find({ _id: "balancer" })
```

**监控分片均衡**:

```javascript
// 查看各分片数据量
db.orders.getShardDistribution()

// 查看迁移状态
db.adminCommand({ balancerStatus: 1 })

// 查看正在迁移的 chunk
db.adminCommand({ moveChunk: "production.orders", find: { user_id: 1000 }, to: "shard2" })
```

## 三、索引优化

### 索引类型

MongoDB 支持多种索引类型:

```
┌─────────────────────────────────────────────────────────────┐
│                    MongoDB 索引类型                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 单字段索引(Single Field)                                 │
│     db.users.createIndex({ email: 1 })                      │
│                                                              │
│  2. 复合索引(Compound)                                       │
│     db.orders.createIndex({ user_id: 1, created_at: -1 })   │
│                                                              │
│  3. 多键索引(Multikey)                                       │
│     db.products.createIndex({ tags: 1 })                    │
│                                                              │
│  4. 文本索引(Text)                                           │
│     db.articles.createIndex({ content: "text" })            │
│                                                              │
│  5. 地理空间索引(Geospatial)                                 │
│     db.stores.createIndex({ location: "2dsphere" })         │
│                                                              │
│  6. 哈希索引(Hashed)                                         │
│     db.users.createIndex({ user_id: "hashed" })             │
│                                                              │
│  7. 通配符索引(Wildcard)                                     │
│     db.users.createIndex({ "metadata.$**": 1 })             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**索引创建示例**:

```javascript
// 创建单字段索引
db.users.createIndex({ email: 1 }, { unique: true, background: true })

// 创建复合索引
db.orders.createIndex(
  { user_id: 1, created_at: -1 },
  { 
    name: "idx_user_created",
    background: true,
    partialFilterExpression: { status: "active" }
  }
)

// 创建 TTL 索引(自动删除过期文档)
db.sessions.createIndex(
  { created_at: 1 },
  { expireAfterSeconds: 3600 }  // 1小时后删除
)

// 创建部分索引
db.orders.createIndex(
  { status: 1 },
  { partialFilterExpression: { amount: { $gt: 1000 } } }
)

// 查看索引
db.orders.getIndexes()

// 删除索引
db.orders.dropIndex("idx_user_created")
```

### 索引策略

**ESR 原则**(Equality, Sort, Range):

```javascript
// 查询条件
db.orders.find({ user_id: 123, status: "active" })
         .sort({ created_at: -1 })
         .limit(10)

// 索引设计
// 1. Equality(等值查询): user_id, status
// 2. Sort(排序): created_at
// 3. Range(范围查询): 无

// 创建复合索引
db.orders.createIndex({ user_id: 1, status: 1, created_at: -1 })

// 索引顺序:
// 1. 等值查询字段在前
// 2. 排序字段其次
// 3. 范围查询字段最后
```

**覆盖索引**:

```javascript
// 创建覆盖索引
db.orders.createIndex({ user_id: 1, status: 1, amount: 1 })

// 查询只返回索引字段
db.orders.find(
  { user_id: 123, status: "active" },
  { _id: 0, user_id: 1, status: 1, amount: 1 }
)

// 执行计划显示 "stage": "PROJECTION_COVERED"
// 无需回表,性能最优
```

**执行计划分析**:

```javascript
// 查看执行计划
db.orders.find({ user_id: 123, status: "active" }).explain("executionStats")

// 关键指标:
// - stage: COLLSCAN(全表扫描) 或 IXSCAN(索引扫描)
// - indexUsed: 使用的索引名称
// - totalDocsExamined: 扫描的文档数
// - totalKeysExamined: 扫描的索引键数
// - nReturned: 返回的文档数

// 理想情况:
// totalDocsExamined ≈ nReturned
// totalKeysExamined ≈ nReturned
```

### 索引维护

**索引使用情况分析**:

```javascript
// 查看索引访问统计
db.orders.aggregate([
  { $indexStats: {} }
])

// 输出示例:
// {
//   "name": "idx_user_created",
//   "accesses": {
//     "ops": 1000,  // 使用次数
//     "since": ISODate("2026-01-01T00:00:00Z")
//   }
// }

// 查找未使用的索引
db.orders.aggregate([
  { $indexStats: {} },
  { $match: { "accesses.ops": 0 } }
])
```

**索引重建**:

```javascript
// 重建索引
db.orders.reIndex()

// 后台创建索引
db.orders.createIndex({ user_id: 1 }, { background: true })

// 监控索引创建进度
db.currentOp({
  $or: [
    { op: "command", "command.createIndexes": { $exists: true } },
    { op: "none", ns: /system.indexes/ }
  ]
})
```

## 四、性能监控

### 关键监控指标

**内存使用**:

```javascript
// 查看内存使用
db.serverStatus().mem

// 输出示例:
// {
//   "bits": 64,
//   "resident": 2048,  // 物理内存(MB)
//   "virtual": 8192,   // 虚拟内存(MB)
//   "supported": true
// }

// WiredTiger 缓存
db.serverStatus().wiredTiger.cache

// 关键指标:
// - "bytes currently in the cache": 缓存使用量
// - "maximum bytes configured": 最大缓存配置
// - "pages read into cache": 从磁盘读取的页数
// - "pages written from cache": 写入磁盘的页数
```

**连接数监控**:

```javascript
// 查看连接数
db.serverStatus().connections

// 输出示例:
// {
//   "current": 100,
//   "available": 900,
//   "totalCreated": 10000
// }

// 查看连接来源
db.currentOp(true).inprog.forEach(function(op) {
  if (op.client) {
    print(op.client + " -> " + op.ns);
  }
})
```

**慢查询监控**:

```javascript
// 启用慢查询日志
db.setProfilingLevel(1, 100)  // 记录超过 100ms 的查询

// 查看慢查询
db.system.profile.find({ millis: { $gt: 1000 } })
  .sort({ ts: -1 })
  .limit(10)

// 查看慢查询统计
db.system.profile.aggregate([
  { $group: {
      _id: "$ns",
      count: { $sum: 1 },
      avgTime: { $avg: "$millis" },
      maxTime: { $max: "$millis" }
    }
  },
  { $sort: { avgTime: -1 } }
])
```

### 性能分析工具

**db.currentOp()**:

```javascript
// 查看当前操作
db.currentOp({ "secs_running": { $gt: 10 } })

// 输出示例:
// {
//   "inprog": [
//     {
//       "opid": 12345,
//       "secs_running": 15,
//       "ns": "production.orders",
//       "query": { "user_id": 123 }
//     }
//   ]
// }

// 终止长时间运行的查询
db.killOp(12345)
```

**db.collection.stats()**:

```javascript
// 查看集合统计信息
db.orders.stats()

// 关键指标:
// - count: 文档数
// - size: 数据大小(字节)
// - storageSize: 存储大小(字节)
// - totalIndexSize: 索引大小(字节)
// - avgObjSize: 平均文档大小

// 查看分片统计
db.orders.stats({ scale: 1024 * 1024 })  // 单位 MB
```

**mongostat 和 mongotop**:

```bash
# 实时监控 MongoDB 状态
mongostat --host localhost --port 27017 --username admin --password

# 输出示例:
# insert query update delete getmore command dirty used flushes vsize   res qrw arw net_in net_out conn
#     *0    *0     *0     *0       0     2|0  0.0% 0.0%       0 1.5G   62M 0|0 1|0   158b   44.1k    2

# 监控集合读写时间
mongotop --host localhost --port 27017

# 输出示例:
# ns                   total    read    write
# production.orders    15ms    10ms     5ms
# production.users      5ms     5ms     0ms
```

### 监控告警配置

**关键告警指标**:

| 指标 | 告警阈值 | 说明 |
|------|---------|------|
| 连接数 | > 80% max connections | 连接池耗尽 |
| 复制延迟 | > 10 秒 | 主从不同步 |
| 慢查询 | > 1% 查询 | 性能下降 |
| 缓存命中率 | < 90% | 内存不足 |
| 磁盘使用 | > 80% | 空间不足 |
| 分片不均衡 | > 20% 差异 | 数据倾斜 |

**Prometheus 监控配置**:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'mongodb'
    static_configs:
      - targets: ['mongodb-exporter:9216']

# MongoDB Exporter 启动
mongodb_exporter --mongodb.uri=mongodb://admin:password@localhost:27017
```

**Grafana Dashboard**:

```json
{
  "dashboard": {
    "title": "MongoDB Monitoring",
    "panels": [
      {
        "title": "Operations",
        "targets": [
          {
            "expr": "rate(mongodb_op_counters_total{type=\"query\"}[5m])"
          }
        ]
      },
      {
        "title": "Connections",
        "targets": [
          {
            "expr": "mongodb_connections{state=\"current\"}"
          }
        ]
      },
      {
        "title": "Replication Lag",
        "targets": [
          {
            "expr": "mongodb_mongod_replset_member_replication_lag"
          }
        ]
      }
    ]
  }
}
```

## 五、备份恢复

### 备份策略

**逻辑备份(mongodump)**:

```bash
# 全量备份
mongodump --host localhost --port 27017 \
  --username admin --password \
  --out /backup/mongodb_$(date +%Y%m%d)

# 备份指定数据库
mongodump --host localhost --port 27017 \
  --db production \
  --out /backup/production_$(date +%Y%m%d)

# 备份指定集合
mongodump --host localhost --port 27017 \
  --db production --collection orders \
  --out /backup/orders_$(date +%Y%m%d)

# 压缩备份
mongodump --host localhost --port 27017 \
  --gzip \
  --out /backup/mongodb_$(date +%Y%m%d)
```

**物理备份**:

```bash
# 1. 停止写入
db.fsyncLock()

# 2. 复制数据文件
cp -r /data/db /backup/db_$(date +%Y%m%d)

# 3. 解锁
db.fsyncUnlock()

# 或使用 LVM 快照
lvcreate -L 10G -s -n mongodb_snapshot /dev/vg0/mongodb
```

**增量备份**:

```bash
# 使用 Oplog 增量备份
mongodump --host localhost --port 27017 \
  -d local -c oplog.rs \
  --query '{ "ts" : { $gt : Timestamp(1640995200, 0) } }' \
  --out /backup/oplog_$(date +%Y%m%d)
```

### 恢复操作

**逻辑恢复(mongorestore)**:

```bash
# 恢复全量备份
mongorestore --host localhost --port 27017 \
  --username admin --password \
  /backup/mongodb_20260311

# 恢复指定数据库
mongorestore --host localhost --port 27017 \
  --db production \
  /backup/production_20260311/production

# 恢复指定集合
mongorestore --host localhost --port 27017 \
  --db production --collection orders \
  /backup/orders_20260311/production/orders.bson

# 恢复压缩备份
mongorestore --host localhost --port 27017 \
  --gzip \
  /backup/mongodb_20260311
```

**Oplog 重放**:

```bash
# 1. 恢复全量备份
mongorestore --host localhost --port 27017 \
  /backup/mongodb_20260311

# 2. 重放 Oplog
mongorestore --host localhost --port 27017 \
  --oplogReplay \
  /backup/oplog_20260311
```

### 自动化备份脚本

```bash
#!/bin/bash
# MongoDB 自动备份脚本

BACKUP_DIR="/backup/mongodb"
DATE=$(date +%Y%m%d_%H%M%S)
MONGO_HOST="localhost"
MONGO_PORT="27017"
MONGO_USER="admin"
MONGO_PASS="password"

# 创建备份目录
mkdir -p $BACKUP_DIR/$DATE

# 全量备份
mongodump --host $MONGO_HOST --port $MONGO_PORT \
  --username $MONGO_USER --password $MONGO_PASS \
  --gzip \
  --out $BACKUP_DIR/$DATE

# 删除 7 天前的备份
find $BACKUP_DIR -type d -mtime +7 -exec rm -rf {} \;

# 上传到 S3
aws s3 sync $BACKUP_DIR/$DATE s3://my-bucket/mongodb-backup/$DATE

echo "Backup completed: $BACKUP_DIR/$DATE"
```

## 小结

- **集群架构**:副本集提供高可用,分片集群实现水平扩展,读写分离提升性能
- **分片策略**:选择高基数、符合查询模式的分片键,范围分片适合范围查询,哈希分片适合随机查询
- **索引优化**:遵循 ESR 原则设计复合索引,使用覆盖索引减少回表,定期分析索引使用情况
- **性能监控**:监控内存、连接数、慢查询、复制延迟,使用 mongostat、mongotop、db.currentOp 分析性能
- **备份恢复**:定期全量备份 + Oplog 增量备份,测试恢复流程,自动化备份脚本

---

## 常见问题

### Q1:MongoDB 的副本集和分片集群有什么区别?

**副本集**:

- **目的**:数据冗余和高可用
- **架构**:1 个 Primary + 多个 Secondary
- **数据**:所有节点数据相同
- **扩展**:垂直扩展(提升单节点配置)
- **适用**:数据量 < 单节点容量

**分片集群**:

- **目的**:水平扩展
- **架构**:多个 Shard(每个 Shard 是副本集)
- **数据**:数据分散到多个 Shard
- **扩展**:水平扩展(增加 Shard)
- **适用**:数据量 > 单节点容量

**选择建议**:

```
数据量 < 1TB: 副本集
数据量 > 1TB: 分片集群
```

### Q2:如何选择 MongoDB 的分片键?

**选择原则**:

1. **高基数**:分片键值越多,数据分布越均匀
2. **查询模式**:分片键应包含在常用查询条件中
3. **写入分布**:写入应均匀分布到所有分片

**示例**:

```javascript
// 错误:低基数字段
sh.shardCollection("production.users", { gender: 1 })  // 只有 2 个值

// 错误:单调递增字段
sh.shardCollection("production.orders", { created_at: 1 })  // 所有新数据写入同一分片

// 正确:高基数 + 随机分布
sh.shardCollection("production.orders", { user_id: "hashed" })

// 正确:复合分片键
sh.shardCollection("production.orders", { user_id: 1, created_at: 1 })
```

### Q3:MongoDB 的内存管理如何优化?

**WiredTiger 缓存配置**:

```javascript
// 查看缓存配置
db.serverStatus().wiredTiger.cache

// 默认:50% 可用内存 - 1GB
// 修改配置(mongod.conf):
storage:
  wiredTiger:
    engineConfig:
      cacheSizeGB: 8

// 监控缓存命中率
db.serverStatus().wiredTiger.cache["pages read into cache"] /
db.serverStatus().wiredTiger.cache["pages requested from cache"]
```

**内存优化建议**:

1. **缓存大小**:设置为物理内存的 50%
2. **索引优化**:减少索引数量,降低内存占用
3. **文档设计**:避免嵌套过深,减少文档大小
4. **查询优化**:使用覆盖索引,减少内存使用

### Q4:MongoDB 如何处理事务?

**多文档事务**(MongoDB 4.0+):

```javascript
// 开启事务
session = db.getMongo().startSession()
session.startTransaction()

try {
  // 操作 1
  session.getDatabase("production").orders.insertOne({
    order_id: "12345",
    user_id: "user123"
  })
  
  // 操作 2
  session.getDatabase("production").inventory.updateOne(
    { product_id: "prod123" },
    { $inc: { stock: -1 } }
  )
  
  // 提交事务
  session.commitTransaction()
} catch (error) {
  // 回滚事务
  session.abortTransaction()
  throw error
} finally {
  session.endSession()
}
```

**事务限制**:

- 仅支持副本集和分片集群
- 分片集群事务需要 MongoDB 4.2+
- 事务时长默认 60 秒
- 事务会影响性能,谨慎使用

### Q5:MongoDB 如何实现数据压缩?

**WiredTiger 压缩**:

```yaml
# mongod.conf
storage:
  wiredTiger:
    collectionConfig:
      blockCompressor: snappy  # 压缩算法: snappy, zlib, zstd
    indexConfig:
      prefixCompression: true  # 索引前缀压缩
```

**压缩算法对比**:

| 算法 | 压缩率 | CPU 消耗 | 适用场景 |
|------|--------|---------|---------|
| snappy | 低 | 低 | 默认,平衡性能和压缩 |
| zlib | 中 | 中 | 存储优先 |
| zstd | 高 | 中 | 高压缩率 |

**压缩效果监控**:

```javascript
// 查看压缩统计
db.orders.stats()

// 输出示例:
// {
//   "size": 1073741824,  // 原始大小
//   "storageSize": 536870912,  // 存储大小
//   "compressionRatio": 2.0
// }
```

## 参考资源

- [MongoDB 官方文档](https://docs.mongodb.com/)
- [MongoDB 分片集群](https://docs.mongodb.com/manual/sharding/)
- [MongoDB 索引优化](https://docs.mongodb.com/manual/indexes/)
- [MongoDB 性能监控](https://docs.mongodb.com/manual/administration/monitoring/)
- [MongoDB 备份恢复](https://docs.mongodb.com/manual/core/backups/)

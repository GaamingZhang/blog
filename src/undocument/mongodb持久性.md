# MongoDB 持久性

## 目录

- [什么是数据持久性](#什么是数据持久性)
- [MongoDB 存储引擎](#mongodb-存储引擎)
- [Journal 日志系统](#journal-日志系统)
- [写关注机制](#写关注机制)
- [检查点机制](#检查点机制)
- [数据文件与内存管理](#数据文件与内存管理)
- [副本集中的持久性](#副本集中的持久性)
- [故障恢复机制](#故障恢复机制)
- [持久性配置最佳实践](#持久性配置最佳实践)
- [常见问题 FAQ](#常见问题-faq)

---

## 什么是数据持久性

数据持久性(Durability)是指数据库系统确保已提交的数据在系统故障(如崩溃、断电)后仍然存在且不会丢失的能力。在 ACID 特性中,D 代表的就是持久性。

MongoDB 通过多种机制保证数据持久性:
- **Journal 日志**: 预写式日志确保数据可恢复
- **检查点机制**: 定期将内存数据刷新到磁盘
- **写关注(Write Concern)**: 控制写操作的确认级别
- **副本集复制**: 数据冗余存储在多个节点

---

## MongoDB 存储引擎

### WiredTiger 存储引擎

从 MongoDB 3.2 开始,WiredTiger 成为默认存储引擎,提供了更好的性能和持久性保证。

**核心特性:**

1. **文档级并发控制**: 使用 MVCC(多版本并发控制)提高并发性能
2. **压缩支持**: 支持多种压缩算法(Snappy、zlib、zstd)
3. **检查点机制**: 定期持久化数据
4. **Journal 日志**: 提供崩溃恢复能力
5. **内存缓存**: 可配置的内存缓存大小

```javascript
// 查看当前存储引擎
db.serverStatus().storageEngine

// 输出示例
{
  "name": "wiredTiger",
  "supportsCommittedReads": true,
  "persistent": true
}
```

**WiredTiger 配置:**

```yaml
# mongod.conf
storage:
  dbPath: /var/lib/mongodb
  journal:
    enabled: true
  engine: wiredTiger
  wiredTiger:
    engineConfig:
      # 缓存大小(默认为物理内存的50% - 1GB)
      cacheSizeGB: 2
      # 日志压缩
      journalCompressor: snappy
    collectionConfig:
      # 块压缩
      blockCompressor: snappy
    indexConfig:
      prefixCompression: true
```

### MMAPv1 存储引擎(已废弃)

MMAPv1 是 MongoDB 3.2 之前的默认存储引擎,现已被弃用。

**特点:**
- 使用内存映射文件
- 集合级锁
- 无内置压缩
- 不建议在新项目中使用

---

## Journal 日志系统

Journal 是 MongoDB 实现持久性的核心机制,采用预写式日志(Write-Ahead Logging, WAL)策略。

### Journal 工作原理

```
写操作流程:
1. 写入内存中的数据结构
2. 写入 Journal 日志文件
3. 定期将 Journal 应用到数据文件
4. Journal 文件循环使用
```

**Journal 文件结构:**

```
/var/lib/mongodb/journal/
├── WiredTigerLog.0000000001
├── WiredTigerLog.0000000002
└── WiredTigerLog.0000000003
```

### Journal 提交频率

WiredTiger 默认每 **50 毫秒**或累积 **100MB** 数据时提交一次 Journal。

```javascript
// 配置 Journal 提交间隔
storage:
  journal:
    enabled: true
    commitIntervalMs: 100  // 提交间隔(毫秒)
```

**提交频率权衡:**
- **更短的间隔**: 更好的持久性保证,但可能影响性能
- **更长的间隔**: 更好的性能,但崩溃时可能丢失更多数据

### 强制 Journal 同步

可以在写操作时要求立即同步 Journal。

```javascript
// 写入时等待 Journal 持久化
db.users.insertOne(
  { name: "张三", balance: 1000 },
  { writeConcern: { w: 1, j: true } }
)

// j: true 表示等待 Journal 确认
// 这会确保即使系统崩溃,数据也不会丢失
```

**性能对比:**

```javascript
// 测试不带 j:true 的写入
const start1 = new Date()
for (let i = 0; i < 10000; i++) {
  db.test.insertOne({ value: i }, { writeConcern: { w: 1 } })
}
const time1 = new Date() - start1
print(`不带 j:true: ${time1}ms`)

// 测试带 j:true 的写入
const start2 = new Date()
for (let i = 0; i < 10000; i++) {
  db.test.insertOne({ value: i }, { writeConcern: { w: 1, j: true } })
}
const time2 = new Date() - start2
print(`带 j:true: ${time2}ms`)

// 通常 j:true 会慢 2-5 倍
```

### Journal 日志清理

Journal 日志会自动清理,无需手动管理。

**清理时机:**
- 检查点完成后,旧的 Journal 文件会被删除
- Journal 文件达到大小限制时会滚动到新文件
- 关闭数据库时会清理所有 Journal 文件

```javascript
// 查看 Journal 统计信息
db.serverStatus().wiredTiger.log

// 输出示例
{
  "log bytes written": 1048576000,
  "log records processed by log scan": 0,
  "log sync operations": 1234,
  "maximum log file size": 104857600
}
```

---

## 写关注机制

写关注(Write Concern)控制 MongoDB 确认写操作成功的级别,直接影响持久性保证。

### 写关注参数

```javascript
{
  w: <value>,           // 确认级别
  j: <boolean>,         // 是否等待 Journal
  wtimeout: <number>    // 超时时间(毫秒)
}
```

### w 参数详解

#### w: 0 - 不确认

```javascript
db.users.insertOne(
  { name: "测试用户" },
  { writeConcern: { w: 0 } }
)

// 客户端不等待任何确认
// 最快,但完全不保证持久性
// 适用场景: 日志记录、统计数据等可丢失的数据
```

**特点:**
- ✅ 最高性能
- ❌ 无持久性保证
- ❌ 无法获知写入是否成功

#### w: 1 - 单节点确认(默认)

```javascript
db.users.insertOne(
  { name: "张三" },
  { writeConcern: { w: 1 } }
)

// 等待主节点确认写入内存
// 适用于大多数场景
```

**特点:**
- ✅ 较好性能
- ⚠️ 主节点内存中的数据,崩溃可能丢失
- ✅ 可以知道写入是否成功

#### w: "majority" - 大多数节点确认

```javascript
db.users.insertOne(
  { name: "李四" },
  { writeConcern: { w: "majority" } }
)

// 等待大多数节点确认
// 在副本集中提供最强持久性保证
```

**特点:**
- ✅ 强持久性保证
- ⚠️ 性能较低
- ✅ 防止数据丢失和回滚

#### w: <number> - 指定节点数量

```javascript
db.users.insertOne(
  { name: "王五" },
  { writeConcern: { w: 2, wtimeout: 5000 } }
)

// 等待至少 2 个节点确认
// 5秒超时
```

### j 参数 - Journal 确认

```javascript
// 等待主节点 Journal 持久化
db.transactions.insertOne(
  { 
    from: "account1",
    to: "account2",
    amount: 1000
  },
  { writeConcern: { w: 1, j: true } }
)

// 即使主节点崩溃,数据也能通过 Journal 恢复
```

**j: true 的影响:**
- ✅ 单节点环境下的最强持久性保证
- ❌ 性能开销较大
- ⚠️ 在副本集中,通常使用 w: "majority" 代替

### wtimeout - 超时设置

```javascript
db.users.insertOne(
  { name: "赵六" },
  { 
    writeConcern: { 
      w: "majority", 
      wtimeout: 5000  // 5秒超时
    } 
  }
)

// 如果 5 秒内未达到确认级别,操作会报错
// 但数据可能已经部分写入
```

**超时处理:**

```javascript
try {
  db.orders.insertOne(
    { orderId: 12345, amount: 999.99 },
    { writeConcern: { w: "majority", wtimeout: 3000 } }
  )
  print("订单创建成功")
} catch (error) {
  if (error.code === 64) {  // WriteConcernError
    print("写关注超时,但数据可能已写入")
    // 需要检查数据是否实际存在
    const order = db.orders.findOne({ orderId: 12345 })
    if (order) {
      print("数据已存在")
    }
  }
}
```

### 写关注级别对比

| 级别 | 持久性 | 性能 | 适用场景 |
|------|--------|------|----------|
| `w: 0` | ❌ 无保证 | ⭐⭐⭐⭐⭐ | 日志、指标、可丢失数据 |
| `w: 1` | ⚠️ 内存 | ⭐⭐⭐⭐ | 一般应用 |
| `w: 1, j: true` | ✅ 单节点持久 | ⭐⭐⭐ | 单节点关键数据 |
| `w: "majority"` | ✅ 强持久性 | ⭐⭐ | 副本集关键数据 |
| `w: "majority", j: true` | ✅ 最强 | ⭐ | 金融交易等 |

### 默认写关注设置

```javascript
// 连接时设置默认写关注
const client = new MongoClient(uri, {
  writeConcern: {
    w: "majority",
    wtimeout: 5000
  }
})

// 数据库级别设置
db.runCommand({
  setDefaultRWConcern: 1,
  defaultWriteConcern: {
    w: "majority"
  }
})

// 集合级别可以覆盖默认设置
db.users.insertOne(
  { name: "孙七" },
  { writeConcern: { w: 1 } }  // 覆盖默认设置
)
```

---

## 检查点机制

检查点(Checkpoint)是 WiredTiger 将内存中的数据持久化到磁盘数据文件的过程。

### 检查点工作原理

```
1. WiredTiger 每 60 秒或写入 2GB 数据时创建检查点
2. 检查点包含所有数据的一致性快照
3. 旧的 Journal 文件在检查点后被清理
4. 系统崩溃后,从最后一个检查点 + Journal 恢复
```

**时间线示例:**

```
T0: 检查点1 创建
T1: 写入操作1 (仅在内存和Journal)
T2: 写入操作2 (仅在内存和Journal)
T60: 检查点2 创建 (操作1,2持久化到数据文件)
T61: 清理 T0-T60 之间的Journal
```

### 检查点配置

```yaml
# mongod.conf
storage:
  wiredTiger:
    engineConfig:
      # 检查点间隔(秒)
      checkpointSizeMB: 1000  # 或达到此大小时触发
```

```javascript
// 手动触发检查点
db.adminCommand({ fsync: 1 })

// 带锁的检查点(阻塞写入)
db.adminCommand({ fsync: 1, lock: true })

// 解锁
db.fsyncUnlock()
```

### 检查点监控

```javascript
// 查看检查点统计
db.serverStatus().wiredTiger.transaction

// 输出示例
{
  "transaction checkpoint currently running": 0,
  "transaction checkpoint generation": 1234,
  "transaction checkpoint max time (msecs)": 500,
  "transaction checkpoint min time (msecs)": 100,
  "transaction checkpoint most recent time (msecs)": 250
}
```

---

## 数据文件与内存管理

### 数据文件结构

```
/var/lib/mongodb/
├── WiredTiger                    # WiredTiger 元数据
├── WiredTiger.wt                 # WiredTiger 数据文件
├── collection-*.wt               # 集合数据文件
├── index-*.wt                    # 索引文件
├── sizeStorer.wt                 # 大小统计
├── journal/                      # Journal 目录
│   ├── WiredTigerLog.0000000001
│   └── WiredTigerLog.0000000002
└── diagnostic.data/              # 诊断数据
```

### WiredTiger 缓存

WiredTiger 使用内部缓存来提高性能。

```yaml
# 配置缓存大小
storage:
  wiredTiger:
    engineConfig:
      # 默认为 max(50% 物理内存 - 1GB, 256MB)
      cacheSizeGB: 4
```

**缓存使用监控:**

```javascript
db.serverStatus().wiredTiger.cache

// 关键指标
{
  "bytes currently in the cache": 2147483648,
  "maximum bytes configured": 4294967296,
  "percentage overhead": 8,
  "tracked dirty bytes in the cache": 524288000,
  "tracked bytes belonging to internal pages in the cache": 1024000,
  "pages evicted by application threads": 1000,
  "pages read into cache": 50000,
  "pages written from cache": 45000
}
```

**缓存优化建议:**

```javascript
// 计算合适的缓存大小
// 公式: (物理内存 - 操作系统 - 其他应用) * 0.5

// 例如 16GB 服务器:
// MongoDB 缓存 = (16GB - 2GB) * 0.5 = 7GB

storage:
  wiredTiger:
    engineConfig:
      cacheSizeGB: 7
```

### 内存压力处理

当缓存压力过大时,WiredTiger 会驱逐页面。

```javascript
// 监控缓存驱逐
db.serverStatus().wiredTiger.cache["pages evicted by application threads"]

// 如果这个值持续增长,说明缓存不足
// 解决方案:
// 1. 增加缓存大小
// 2. 增加物理内存
// 3. 优化查询减少内存使用
// 4. 使用索引减少数据扫描
```

---

## 副本集中的持久性

### 复制机制

副本集通过将数据复制到多个节点来提供持久性保证。

```
主节点(Primary) → Oplog → 从节点1(Secondary)
                 ↓
                从节点2(Secondary)
```

**复制流程:**

```
1. 主节点接收写操作
2. 写操作写入 Oplog (操作日志)
3. 从节点拉取 Oplog 并应用
4. 从节点确认复制完成
```

### Oplog (操作日志)

Oplog 是一个特殊的固定大小集合,存储所有修改操作。

```javascript
// 查看 Oplog 信息
use local
db.oplog.rs.stats()

// 输出示例
{
  "ns": "local.oplog.rs",
  "size": 10737418240,        // 10GB
  "count": 1000000,
  "avgObjSize": 256,
  "storageSize": 10737418240,
  "capped": true,             // 固定集合
  "max": -1
}

// 查看 Oplog 最早和最新条目
db.oplog.rs.find().sort({$natural: 1}).limit(1).pretty()   // 最早
db.oplog.rs.find().sort({$natural: -1}).limit(1).pretty()  // 最新
```

**Oplog 条目示例:**

```javascript
{
  "ts": Timestamp(1705555200, 1),
  "h": NumberLong("1234567890"),
  "v": 2,
  "op": "i",                    // 操作类型: i=insert, u=update, d=delete
  "ns": "mydb.users",           // 命名空间
  "o": {                        // 操作内容
    "_id": ObjectId("..."),
    "name": "张三",
    "age": 28
  }
}
```

### Oplog 大小配置

```yaml
# mongod.conf
replication:
  oplogSizeMB: 10240  # 10GB
```

**Oplog 大小选择:**

```javascript
// 计算公式:
// Oplog大小 = 峰值写入速率 * 维护窗口时间 * 安全系数

// 示例:
// 峰值写入: 100MB/小时
// 维护窗口: 24小时
// 安全系数: 2
// Oplog = 100 * 24 * 2 = 4800 MB (约 5GB)
```

### 写关注与副本集

```javascript
// w: "majority" 在副本集中的含义
// 3节点副本集: 需要 2 个节点确认
// 5节点副本集: 需要 3 个节点确认

db.orders.insertOne(
  { orderId: 123, total: 999.99 },
  { writeConcern: { w: "majority", wtimeout: 5000 } }
)

// 这确保了:
// 1. 数据已复制到大多数节点
// 2. 即使主节点故障,数据不会丢失
// 3. 不会发生回滚
```

### 读关注 (Read Concern)

读关注控制读取数据的一致性级别。

```javascript
// local - 读取本地数据(默认)
db.users.find().readConcern("local")

// majority - 读取已被大多数节点确认的数据
db.users.find().readConcern("majority")

// linearizable - 线性化读取(最强一致性)
db.users.find().readConcern("linearizable")

// snapshot - 快照读取(用于事务)
session.startTransaction({ readConcern: { level: "snapshot" } })
```

**读关注级别对比:**

| 级别 | 一致性 | 性能 | 说明 |
|------|--------|------|------|
| `local` | 弱 | ⭐⭐⭐⭐⭐ | 可能读到未提交或会回滚的数据 |
| `available` | 弱 | ⭐⭐⭐⭐⭐ | 分片环境下的 local |
| `majority` | 强 | ⭐⭐⭐ | 只读取已确认不会回滚的数据 |
| `linearizable` | 最强 | ⭐ | 保证读取反映所有成功的写入 |
| `snapshot` | 事务 | ⭐⭐ | 事务中的一致性快照 |

---

## 故障恢复机制

### 单节点恢复

**崩溃恢复流程:**

```
1. MongoDB 启动
2. 读取最后一个检查点
3. 从检查点位置开始重放 Journal
4. 应用所有 Journal 中的操作
5. 恢复完成,接受连接
```

```javascript
// 启动时查看恢复日志
// mongod.log

2025-01-18T10:00:00.000+0800 I CONTROL  [initandlisten] MongoDB starting
2025-01-18T10:00:01.000+0800 I STORAGE  [initandlisten] WiredTiger recoveryTimestamp. Ts: Timestamp(1705550400, 1)
2025-01-18T10:00:02.000+0800 I STORAGE  [initandlisten] WiredTiger recovering from checkpoint
2025-01-18T10:00:03.000+0800 I STORAGE  [initandlisten] WiredTiger recovery complete
```

### 副本集故障恢复

**场景1: 主节点故障**

```
1. 从节点检测到主节点不可用
2. 触发选举流程
3. 选举新的主节点
4. 新主节点开始接受写入
5. 旧主节点恢复后成为从节点
```

```javascript
// 模拟主节点故障
rs.stepDown(60)  // 主节点降级 60 秒

// 查看副本集状态
rs.status()

// 输出会显示新的主节点
{
  "members": [
    {
      "name": "mongodb1:27017",
      "stateStr": "PRIMARY",
      "health": 1
    },
    {
      "name": "mongodb2:27017",
      "stateStr": "SECONDARY",
      "health": 1
    }
  ]
}
```

**场景2: 从节点故障**

```
1. 从节点崩溃
2. 从节点重启
3. 从 Oplog 同步缺失的操作
4. 追上主节点后恢复 SECONDARY 状态
```

**场景3: 网络分区**

```
网络分区导致副本集分裂:
分区1: 主节点 + 从节点1 (大多数)
分区2: 从节点2 (少数)

结果:
- 分区1: 继续提供服务
- 分区2: 从节点2 进入 SECONDARY 状态,不接受读写
```

### 数据回滚

当主节点故障且未同步的写入存在时,可能发生回滚。

```javascript
// 回滚场景:
// 1. 主节点 A 接受写入,但未同步到从节点
// 2. 主节点 A 崩溃
// 3. 从节点 B 被选为新主节点
// 4. 原主节点 A 恢复
// 5. A 的未同步数据被回滚

// 回滚的数据会保存到 rollback 目录
// /var/lib/mongodb/rollback/
```

**防止回滚:**

```javascript
// 使用 w: "majority" 确保数据不会回滚
db.criticalData.insertOne(
  { important: "data" },
  { writeConcern: { w: "majority" } }
)

// 配置副本集默认写关注
cfg = rs.conf()
cfg.settings = cfg.settings || {}
cfg.settings.getLastErrorDefaults = { w: "majority" }
rs.reconfig(cfg)
```

### 备份与恢复

#### 1. mongodump/mongorestore

```bash
# 备份整个数据库
mongodump --uri="mongodb://localhost:27017" --out=/backup/dump-2025-01-18

# 备份特定数据库
mongodump --db=mydb --out=/backup/mydb-backup

# 备份特定集合
mongodump --db=mydb --collection=users --out=/backup/users-backup

# 恢复数据
mongorestore --uri="mongodb://localhost:27017" /backup/dump-2025-01-18

# 恢复到不同数据库
mongorestore --db=mydb_restore /backup/mydb-backup/mydb
```

#### 2. 文件系统快照

```bash
# 1. 锁定数据库(可选,确保一致性)
mongo --eval "db.fsyncLock()"

# 2. 创建快照(以 LVM 为例)
lvcreate --size 10G --snapshot --name mongo-snapshot /dev/vg/mongodb

# 3. 解锁数据库
mongo --eval "db.fsyncUnlock()"

# 4. 挂载快照并备份
mount /dev/vg/mongo-snapshot /mnt/snapshot
rsync -av /mnt/snapshot/ /backup/mongodb-snapshot/

# 5. 清理
umount /mnt/snapshot
lvremove /dev/vg/mongo-snapshot
```

#### 3. 云备份(MongoDB Atlas)

```javascript
// MongoDB Atlas 提供自动备份
// - 连续备份,RPO(恢复点目标)< 1分钟
// - 基于快照的备份
// - 点击即可恢复

// 通过 API 触发备份
curl --user "{PUBLIC_KEY}:{PRIVATE_KEY}" \
  --digest \
  --header "Content-Type: application/json" \
  --request POST \
  "https://cloud.mongodb.com/api/atlas/v1.0/groups/{GROUP_ID}/clusters/{CLUSTER_NAME}/backup/snapshots"
```

---

## 持久性配置最佳实践

### 场景1: 高持久性要求(金融、医疗)

```javascript
// 连接配置
const client = new MongoClient(uri, {
  writeConcern: {
    w: "majority",
    j: true,
    wtimeout: 5000
  },
  readConcern: {
    level: "majority"
  }
})

// 服务器配置
// mongod.conf
storage:
  journal:
    enabled: true
    commitIntervalMs: 50  # 更频繁的提交
  wiredTiger:
    engineConfig:
      cacheSizeGB: 8
      journalCompressor: snappy

replication:
  oplogSizeMB: 51200  # 50GB,确保足够的恢复窗口
```

**特点:**
- ✅ 最强持久性保证
- ❌ 性能开销较大
- ✅ 适合关键业务数据

### 场景2: 平衡性能与持久性(电商、社交)

```javascript
// 默认写关注
const client = new MongoClient(uri, {
  writeConcern: {
    w: "majority",
    wtimeout: 3000
  }
})

// 关键操作使用更强保证
db.orders.insertOne(
  orderData,
  { writeConcern: { w: "majority", j: true } }
)

// 非关键操作使用默认
db.logs.insertOne(
  logData,
  { writeConcern: { w: 1 } }
)

// mongod.conf
storage:
  journal:
    enabled: true
    commitIntervalMs: 100
  wiredTiger:
    engineConfig:
      cacheSizeGB: 4
```

**特点:**
- ✅ 较好的性能
- ✅ 合理的持久性保证
- ✅ 适合大多数应用

### 场景3: 高性能要求(缓存、日志)

```javascript
// 低写关注
const client = new MongoClient(uri, {
  writeConcern: {
    w: 1,
    wtimeout: 1000
  }
})

// 日志可以使用 w: 0
db.accessLogs.insertOne(
  logEntry,
  { writeConcern: { w: 0 } }
)

// mongod.conf
storage:
  journal:
    enabled: true
    commitIntervalMs: 300  # 较长的提交间隔
  wiredTiger:
    engineConfig:
      cacheSizeGB: 16  # 更大的缓存
```

**特点:**
- ✅ 最佳性能
- ⚠️ 可能丢失部分数据
- ✅ 适合可容忍数据丢失的场景

### 场景4: 副本集配置

```javascript
// 3节点副本集配置示例
rs.initiate({
  _id: "myReplicaSet",
  members: [
    { 
      _id: 0, 
      host: "mongodb1:27017",
      priority: 2  // 更高优先级
    },
    { 
      _id: 1, 
      host: "mongodb2:27017",
      priority: 1
    },
    { 
      _id: 2, 
      host: "mongodb3:27017",
      priority: 1
    }
  ],
  settings: {
    // 默认写关注
    getLastErrorDefaults: {
      w: "majority",
      wtimeout: 5000
    },
    // 心跳间隔
    heartbeatIntervalMillis: 2000,
    // 选举超时
    electionTimeoutMillis: 10000
  }
})
```

### 监控持久性指标

```javascript
// 定期检查以下指标

// 1. Journal 统计
db.serverStatus().wiredTiger.log

// 2. 检查点统计
db.serverStatus().wiredTiger.transaction

// 3. 缓存使用
db.serverStatus().wiredTiger.cache

// 4. Oplog 信息(副本集)
use local
db.oplog.rs.stats()

// 5. 复制延迟(副本集)
rs.printReplicationInfo()
rs.printSecondaryReplicationInfo()

// 6. 写关注统计
db.serverStatus().opWriteConcernCounters
```

---

## 常见问题 FAQ

### 1. MongoDB 在什么情况下会丢失数据?

**可能丢失数据的情况:**

**情况1: 使用低写关注级别**
```javascript
// w: 0 - 完全不保证
db.logs.insertOne(
  { message: "test" },
  { writeConcern: { w: 0 } }
)
// 系统崩溃 → 数据丢失
```

**情况2: 未启用 Journal**
```yaml
storage:
  journal:
    enabled: false  # 不推荐!
```
系统崩溃后,自上次检查点以来的数据会丢失(最多60秒数据)。

**情况3: 单节点且未使用 j: true**
```javascript
db.users.insertOne(
  { name: "张三" },
  { writeConcern: { w: 1 } }  // 仅内存确认
)
// 写入后立即崩溃 → 未持久化到 Journal 的数据丢失
```

**情况4: 副本集回滚**
```javascript
// 主节点接受写入但未同步
db.data.insertOne({ value: 1 }, { writeConcern: { w: 1 } })
// 主节点立即崩溃 → 未同步数据在新主节点选举后回滚
```

**如何避免数据丢失:**

```javascript
// 单节点环境
db.critical.insertOne(
  data,
  { writeConcern: { w: 1, j: true } }
)

// 副本集环境(推荐)
db.critical.insertOne(
  data,
  { writeConcern: { w: "majority", wtimeout: 5000 } }
)

// 最强保证
db.critical.insertOne(
  data,
  { writeConcern: { w: "majority", j: true, wtimeout: 5000 } }
)
```

### 2. Journal 日志和 Oplog 有什么区别?

**Journal(日志系统):**

- **目的**: 崩溃恢复,保证单节点持久性
- **位置**: 每个节点都有自己的 Journal
- **内容**: 底层数据修改操作(页面级别)
- **格式**: 二进制格式,WiredTiger 特定
- **生命周期**: 检查点后自动清理
- **大小**: 通常几百MB到几GB

```javascript
// Journal 示例
// 记录的是存储引擎级别的修改
{
  "lsn": 12345,
  "page_id": 678,
  "modifications": "binary data..."
}
```

**Oplog(操作日志):**

- **目的**: 副本集数据复制
- **位置**: `local.oplog.rs` 集合
- **内容**: 数据库级别的操作(insert、update、delete等)
- **格式**: BSON 文档,人类可读
- **生命周期**: 固定大小集合,循环覆盖
- **大小**: 通常几GB到几十GB

```javascript
// Oplog 示例
{
  "ts": Timestamp(1705555200, 1),
  "op": "i",  // insert
  "ns": "mydb.users",
  "o": { "_id": 1, "name": "张三" }
}
```

**对比总结:**

| 特性 | Journal | Oplog |
|------|---------|-------|
| 作用域 | 单节点 | 副本集 |
| 用途 | 崩溃恢复 | 数据复制 |
| 级别 | 存储引擎 | 数据库 |
| 格式 | 二进制 | BSON |
| 可读性 | 不可读 | 可读 |

### 3. 如何选择合适的写关注级别?

**决策树:**

```
是否是副本集?
├─ 否(单节点)
│  └─ 数据是否关键?
│     ├─ 是 → { w: 1, j: true }
│     └─ 否 → { w: 1 }
└─ 是(副本集)
   └─ 数据是否关键?
      ├─ 是 → { w: "majority", wtimeout: 5000 }
      │       或 { w: "majority", j: true, wtimeout: 5000 }
      ├─ 一般 → { w: "majority", wtimeout: 3000 }
      └─ 否(日志等) → { w: 1 } 或 { w: 0 }
```

**实际应用示例:**

```javascript
// 用户注册(关键)
db.users.insertOne(
  userData,
  { writeConcern: { w: "majority", wtimeout: 5000 } }
)

// 订单创建(非常关键)
db.orders.insertOne(
  orderData,
  { writeConcern: { w: "majority", j: true, wtimeout: 5000 } }
)

// 访问日志(不关键)
db.accessLogs.insertOne(
  logData,
  { writeConcern: { w: 1 } }
)

// 实时指标(可丢失)
db.metrics.insertOne(
  metricData,
  { writeConcern: { w: 0 } }
)
```

**性能对比测试:**

```javascript
// 测试函数
function testWriteConcern(wc, count = 1000) {
  const start = new Date()
  for (let i = 0; i < count; i++) {
    db.test.insertOne({ value: i }, { writeConcern: wc })
  }
  const duration = new Date() - start
  print(`${JSON.stringify(wc)}: ${duration}ms, ${(count/duration*1000).toFixed(0)} ops/sec`)
}

// 运行测试
testWriteConcern({ w: 0 })
testWriteConcern({ w: 1 })
testWriteConcern({ w: 1, j: true })
testWriteConcern({ w: "majority" })
testWriteConcern({ w: "majority", j: true })

// 典型结果(3节点副本集):
// {"w":0}: 500ms, 2000 ops/sec
// {"w":1}: 1000ms, 1000 ops/sec
// {"w":1,"j":true}: 3000ms, 333 ops/sec
// {"w":"majority"}: 2000ms, 500 ops/sec
// {"w":"majority","j":true}: 4000ms, 250 ops/sec
```

### 4. 系统崩溃后 MongoDB 如何恢复数据?

**恢复流程详解:**

**阶段1: 启动检查**
```
1. MongoDB 启动
2. 读取 WiredTiger 元数据
3. 确定最后一个检查点位置
```

**阶段2: 检查点恢复**
```
4. 加载检查点时的数据状态
5. 此时数据状态为最后一次检查点(最多60秒前)
```

**阶段3: Journal 重放**
```
6. 扫描 Journal 文件
7. 从检查点时间戳开始重放操作
8. 应用所有 Journal 中的修改
```

**阶段4: 完成恢复**
```
9. 验证数据完整性
10. 启动服务,接受连接
```

**时间线示例:**

```
T0:00 - 检查点创建
T0:10 - 插入文档 A (写入内存 + Journal)
T0:20 - 更新文档 B (写入内存 + Journal)
T0:30 - 删除文档 C (写入内存 + Journal)
T0:35 - 系统崩溃!

恢复过程:
T1:00 - 启动,读取 T0:00 检查点
T1:05 - 重放 Journal: 应用 A、B、C 操作
T1:10 - 恢复完成,数据包含 A、B、C 的修改
```

**日志示例:**

```
2025-01-18T10:00:00.000+0800 I CONTROL  [initandlisten] MongoDB starting
2025-01-18T10:00:01.000+0800 I STORAGE  [initandlisten] WiredTiger recoveryTimestamp. Ts: Timestamp(1705555200, 1)
2025-01-18T10:00:02.000+0800 I STORAGE  [initandlisten] WiredTiger recovering from checkpoint
2025-01-18T10:00:03.000+0800 I STORAGE  [initandlisten] WiredTiger recovery replaying 1234 operations
2025-01-18T10:00:05.000+0800 I STORAGE  [initandlisten] WiredTiger recovery complete
2025-01-18T10:00:06.000+0800 I NETWORK  [initandlisten] Waiting for connections on port 27017
```

**副本集环境的恢复:**

```
1. 节点启动
2. 本地 Journal 恢复(如上)
3. 连接到副本集
4. 从主节点 Oplog 同步缺失操作
5. 追上主节点,变为 SECONDARY
```

### 5. 如何监控和调优 MongoDB 的持久性性能?

**关键监控指标:**

**1. Journal 性能**
```javascript
// 监控 Journal 同步次数和延迟
const journalStats = db.serverStatus().wiredTiger.log

print("Journal 统计:")
print(`写入字节: ${journalStats["log bytes written"]}`)
print(`同步操作: ${journalStats["log sync operations"]}`)
print(`同步时间: ${journalStats["log sync time duration (usecs)"]}us`)

// 计算平均同步时间
const avgSyncTime = journalStats["log sync time duration (usecs)"] / 
                    journalStats["log sync operations"]
print(`平均同步时间: ${avgSyncTime.toFixed(2)}us`)

// 警告阈值: 平均同步时间 > 10ms
if (avgSyncTime > 10000) {
  print("警告: Journal 同步慢,可能是磁盘I/O瓶颈")
}
```

**2. 检查点性能**
```javascript
const checkpointStats = db.serverStatus().wiredTiger.transaction

print("检查点统计:")
print(`检查点次数: ${checkpointStats["transaction checkpoints"]}`)
print(`最大时间: ${checkpointStats["transaction checkpoint max time (msecs)"]}ms`)
print(`最小时间: ${checkpointStats["transaction checkpoint min time (msecs)"]}ms`)
print(`最近时间: ${checkpointStats["transaction checkpoint most recent time (msecs)"]}ms`)

// 警告阈值: 检查点时间 > 1000ms
if (checkpointStats["transaction checkpoint most recent time (msecs)"] > 1000) {
  print("警告: 检查点耗时过长")
}
```

**3. 缓存效率**
```javascript
const cacheStats = db.serverStatus().wiredTiger.cache

const cacheHitRate = (cacheStats["pages read into cache"] === 0) ? 100 :
  ((cacheStats["pages read into cache"] - cacheStats["pages requested from the cache"]) / 
   cacheStats["pages read into cache"]) * 100

print(`缓存命中率: ${cacheHitRate.toFixed(2)}%`)

// 警告阈值: 命中率 < 90%
if (cacheHitRate < 90) {
  print("警告: 缓存命中率低,考虑增加缓存大小")
}
```

**4. 复制延迟(副本集)**
```javascript
// 在主节点执行
rs.printSecondaryReplicationInfo()

// 输出示例:
// source: mongodb2:27017
//   syncedTo: Thu Jan 18 2025 10:00:00 GMT+0800
//   0 secs (0 hrs) behind the primary

// 编程方式检查
const status = rs.status()
status.members.forEach(member => {
  if (member.stateStr === "SECONDARY") {
    const lag = (status.date - member.optimeDate) / 1000
    print(`${member.name} 延迟: ${lag.toFixed(2)}秒`)
    
    // 警告阈值: 延迟 > 10秒
    if (lag > 10) {
      print(`警告: ${member.name} 复制延迟过大`)
    }
  }
})
```

**性能调优建议:**

**1. 硬件优化**
```bash
# 使用 SSD 存储
# 推荐: NVMe SSD

# 分离 Journal 和数据文件
storage:
  dbPath: /data/mongodb
  directoryPerDB: true
  wiredTiger:
    engineConfig:
      directoryForIndexes: true

# 将 Journal 放在不同的磁盘
ln -s /journal/path /data/mongodb/journal
```

**2. 系统配置优化**
```bash
# 禁用透明大页
echo never > /sys/kernel/mm/transparent_hugepage/enabled
echo never > /sys/kernel/mm/transparent_hugepage/defrag

# 调整文件描述符限制
ulimit -n 64000

# I/O 调度器设置(SSD)
echo noop > /sys/block/sda/queue/scheduler
```

**3. MongoDB 配置优化**
```yaml
storage:
  journal:
    enabled: true
    commitIntervalMs: 100  # 默认50,可适当增加
  wiredTiger:
    engineConfig:
      cacheSizeGB: 8  # 根据工作集大小调整
      journalCompressor: snappy  # 或 zstd(更高压缩率)
    collectionConfig:
      blockCompressor: snappy

# 如果主要是读操作,可以增加缓存
# 如果主要是写操作,确保足够的内存用于 Journal
```

**4. 应用层优化**
```javascript
// 批量写入
const bulk = db.users.initializeUnorderedBulkOp()
for (let i = 0; i < 10000; i++) {
  bulk.insert({ value: i })
}
bulk.execute({ writeConcern: { w: "majority" } })

// 使用合适的写关注
// 根据数据重要性选择,不要一刀切
```

**监控脚本示例:**
```javascript
// monitor_durability.js
function monitorDurability() {
  const stats = db.serverStatus()
  
  // Journal
  const journal = stats.wiredTiger.log
  const journalSyncAvg = journal["log sync time duration (usecs)"] / 
                         journal["log sync operations"]
  
  // 检查点
  const checkpoint = stats.wiredTiger.transaction
  const recentCheckpoint = checkpoint["transaction checkpoint most recent time (msecs)"]
  
  // 缓存
  const cache = stats.wiredTiger.cache
  const cacheUsage = (cache["bytes currently in the cache"] / 
                      cache["maximum bytes configured"] * 100).toFixed(2)
  
  print(`=== MongoDB 持久性监控 ===`)
  print(`时间: ${new Date()}`)
  print(`Journal 平均同步: ${(journalSyncAvg/1000).toFixed(2)}ms`)
  print(`最近检查点耗时: ${recentCheckpoint}ms`)
  print(`缓存使用率: ${cacheUsage}%`)
  
  // 副本集延迟
  if (rs.status().ok) {
    rs.status().members.forEach(m => {
      if (m.stateStr === "SECONDARY") {
        const lag = (rs.status().date - m.optimeDate) / 1000
        print(`${m.name} 复制延迟: ${lag.toFixed(2)}s`)
      }
    })
  }
  
  print(`========================\n`)
}

// 每30秒监控一次
while (true) {
  monitorDurability()
  sleep(30000)
}
```

---

## 总结

MongoDB 的持久性通过多层机制保障:

**核心机制:**
- **Journal 日志**: 预写式日志提供崩溃恢复能力
- **检查点**: 定期持久化内存数据到磁盘
- **写关注**: 灵活控制写操作的确认级别
- **副本集**: 数据冗余提供高可用性

**最佳实践:**
- 根据数据重要性选择合适的写关注级别
- 在副本集环境使用 `w: "majority"` 防止数据丢失
- 监控 Journal、检查点和复制延迟指标
- 使用 SSD 存储提升 I/O 性能
- 定期备份,建立灾难恢复计划

**权衡考虑:**
- 持久性 ↔ 性能: 更强的持久性保证通常意味着更高的性能开销
- 一致性 ↔ 可用性: 在网络分区时需要在两者间做出选择
- 成本 ↔ 可靠性: 副本集节点数量影响成本和可靠性

理解并正确配置 MongoDB 的持久性机制,是构建可靠应用的关键基础。
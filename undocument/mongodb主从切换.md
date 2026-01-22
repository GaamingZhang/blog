# MongoDB 主从切换

## 目录

- [副本集概述](#副本集概述)
- [副本集架构](#副本集架构)
- [选举机制](#选举机制)
- [主从切换流程](#主从切换流程)
- [心跳与故障检测](#心跳与故障检测)
- [Oplog 与数据同步](#oplog-与数据同步)
- [手动主从切换](#手动主从切换)
- [主从切换的影响](#主从切换的影响)
- [高可用性配置](#高可用性配置)
- [常见问题 FAQ](#常见问题-faq)

---

## 副本集概述

MongoDB 使用**副本集(Replica Set)**实现高可用性,而不是传统的主从复制。副本集是一组维护相同数据集的 MongoDB 服务器,提供数据冗余和自动故障转移。

### 为什么不叫"主从复制"?

虽然副本集中有主节点(Primary)和从节点(Secondary),但 MongoDB 官方称其为"副本集"而非"主从复制",主要原因:

1. **自动故障转移**: 从节点可以自动选举成为新的主节点
2. **多个从节点平等**: 所有从节点地位相同,都可能成为主节点
3. **无需人工干预**: 故障恢复自动完成
4. **分布式共识**: 基于 Raft 共识算法的选举机制

### 副本集的核心优势

- ✅ **高可用性**: 主节点故障时自动切换
- ✅ **数据冗余**: 多个节点存储相同数据
- ✅ **读扩展**: 可以从从节点读取数据
- ✅ **灾难恢复**: 支持跨数据中心部署
- ✅ **零停机维护**: 滚动升级不影响服务

---

## 副本集架构

### 节点类型

#### 1. Primary (主节点)

**职责:**
- 接收所有写操作
- 记录操作到 Oplog
- 默认处理读操作

**特点:**
- 副本集中只有一个主节点
- 主节点故障会触发选举
- 写操作必须在主节点执行

```javascript
// 查看当前主节点
rs.status().members.filter(m => m.stateStr === "PRIMARY")

// 输出示例
[
  {
    "_id": 0,
    "name": "mongodb1:27017",
    "health": 1,
    "state": 1,
    "stateStr": "PRIMARY",
    "uptime": 86400,
    "optime": { "ts": Timestamp(1705555200, 1), "t": 5 }
  }
]
```

#### 2. Secondary (从节点)

**职责:**
- 从主节点复制数据
- 参与选举投票
- 可选地提供读服务

**特点:**
- 可以有多个从节点
- 持续从 Oplog 同步数据
- 可配置为主节点候选

```javascript
// 查看从节点
rs.status().members.filter(m => m.stateStr === "SECONDARY")

// 输出示例
[
  {
    "_id": 1,
    "name": "mongodb2:27017",
    "health": 1,
    "state": 2,
    "stateStr": "SECONDARY",
    "uptime": 86400,
    "optimeDate": ISODate("2025-01-18T10:00:00Z"),
    "syncSourceHost": "mongodb1:27017",
    "syncSourceId": 0
  }
]
```

#### 3. Arbiter (仲裁节点)

**职责:**
- 仅参与选举投票
- 不存储数据
- 不提供读写服务

**特点:**
- 资源占用极小
- 用于打破偶数节点的平局
- 不能成为主节点

```javascript
// 添加仲裁节点
rs.addArb("mongodb-arbiter:27017")

// 查看仲裁节点
rs.status().members.filter(m => m.stateStr === "ARBITER")
```

**是否使用仲裁节点?**

```
推荐配置:
✅ 3节点: Primary + 2 Secondary (无需仲裁)
✅ 5节点: Primary + 4 Secondary (无需仲裁)
⚠️ 2节点: Primary + Secondary + Arbiter (资源受限时)

不推荐:
❌ Primary + Arbiter (无数据冗余)
❌ 大规模使用仲裁节点
```

### 副本集拓扑示例

#### 标准三节点配置

```
┌─────────────┐
│   Primary   │ ←────┐
│  mongodb1   │      │ 写操作
└─────────────┘      │
      │              │
      │ Oplog 复制   │
      ↓              │
┌─────────────┐      │
│  Secondary  │      │
│  mongodb2   │      │
└─────────────┘      │
      │              │
      │              │
      ↓              │
┌─────────────┐      │
│  Secondary  │      │
│  mongodb3   │ ─────┘
└─────────────┘
```

#### 跨数据中心配置

```
数据中心 A (主)          数据中心 B (灾备)
┌─────────────┐         ┌─────────────┐
│   Primary   │────────→│  Secondary  │
│  mongodb1   │  复制   │  mongodb3   │
└─────────────┘         └─────────────┘
      │
      │ 复制
      ↓
┌─────────────┐
│  Secondary  │
│  mongodb2   │
└─────────────┘
```

### 初始化副本集

```javascript
// 1. 启动三个 MongoDB 实例
// mongod --replSet myReplicaSet --port 27017 --dbpath /data/db1
// mongod --replSet myReplicaSet --port 27018 --dbpath /data/db2
// mongod --replSet myReplicaSet --port 27019 --dbpath /data/db3

// 2. 连接到其中一个实例
mongo --port 27017

// 3. 初始化副本集
rs.initiate({
  _id: "myReplicaSet",
  members: [
    { _id: 0, host: "mongodb1:27017" },
    { _id: 1, host: "mongodb2:27018" },
    { _id: 2, host: "mongodb3:27019" }
  ]
})

// 4. 等待初始化完成
rs.status()

// 5. 验证配置
rs.conf()
```

---

## 选举机制

MongoDB 使用基于 **Raft 共识算法**的选举机制,确保在主节点故障时快速选出新的主节点。

### 选举触发条件

1. **副本集初始化**: 第一次启动时选举主节点
2. **主节点故障**: 主节点宕机或网络分区
3. **主节点降级**: 手动执行 `rs.stepDown()`
4. **配置变更**: 修改副本集配置可能触发选举
5. **优先级变化**: 高优先级节点上线

### 选举过程详解

```
阶段1: 故障检测
- 从节点通过心跳检测主节点状态
- 连续多次心跳失败(默认10秒)
- 节点状态变为 "UNKNOWN"

阶段2: 发起选举
- 满足条件的从节点进入 CANDIDATE 状态
- 增加选举任期号(term)
- 向其他节点发送选举请求

阶段3: 投票过程
- 每个节点每个任期只能投一票
- 投票给满足条件的候选者
- 需要获得大多数票(n/2 + 1)

阶段4: 成为主节点
- 获得多数票的候选者成为 PRIMARY
- 开始接受写操作
- 其他候选者变回 SECONDARY
```

**完整流程图:**

```
主节点故障
    ↓
心跳超时(10秒)
    ↓
从节点A: "我要竞选主节点!" (Term: 5 → 6)
    ↓
┌────────────────┐
│  投票阶段       │
│ 从节点A → 从节点B: "投我一票?"  │
│ 从节点A → 从节点C: "投我一票?"  │
└────────────────┘
    ↓
从节点B: "好,投给你" (1票)
从节点C: "好,投给你" (2票)
    ↓
从节点A获得3票(包括自己) ≥ 大多数(2票)
    ↓
从节点A 成为新的主节点
    ↓
其他节点同步新主节点的数据
```

### 选举优先级

每个节点可以配置优先级(0-1000),影响选举结果。

```javascript
// 配置优先级
cfg = rs.conf()
cfg.members[0].priority = 2   // 最高优先级
cfg.members[1].priority = 1   // 中等优先级
cfg.members[2].priority = 0   // 不能成为主节点
rs.reconfig(cfg)

// priority = 0: 被动节点,不会成为主节点
// priority > 1: 更容易成为主节点
// priority = 1: 默认值
```

**优先级使用场景:**

```javascript
// 场景1: 跨数据中心,优先本地机房
rs.initiate({
  _id: "myReplicaSet",
  members: [
    { _id: 0, host: "dc1-node1:27017", priority: 2 },  // 主数据中心
    { _id: 1, host: "dc1-node2:27017", priority: 2 },  // 主数据中心
    { _id: 2, host: "dc2-node1:27017", priority: 1 }   // 灾备数据中心
  ]
})

// 场景2: 只读节点(用于分析)
{
  _id: 3,
  host: "analytics:27017",
  priority: 0,        // 永不成为主节点
  hidden: true,       // 对应用隐藏
  slaveDelay: 3600    // 延迟1小时(防误删)
}
```

### 选举约束条件

候选节点必须满足以下条件才能被选为主节点:

1. **优先级 > 0**: priority 不为 0
2. **数据最新**: 拥有最新的 oplog
3. **可达性**: 能与大多数节点通信
4. **配置版本**: 拥有最新的副本集配置
5. **投票权**: 具有投票权(votes = 1)

```javascript
// 查看节点选举资格
rs.status().members.forEach(m => {
  print(`${m.name}:`)
  print(`  状态: ${m.stateStr}`)
  print(`  优先级: ${rs.conf().members[m._id].priority}`)
  print(`  最新optime: ${m.optimeDate}`)
  print(`  健康: ${m.health}`)
})
```

### 选举超时与重试

```javascript
// 配置选举超时
cfg = rs.conf()
cfg.settings = {
  electionTimeoutMillis: 10000,     // 选举超时: 10秒
  heartbeatIntervalMillis: 2000,    // 心跳间隔: 2秒
  heartbeatTimeoutSecs: 10          // 心跳超时: 10秒
}
rs.reconfig(cfg)
```

**选举失败重试:**

```
第1次选举失败
    ↓
等待随机时间(避免冲突)
    ↓
term + 1
    ↓
重新发起选举
    ↓
...重复直到成功
```

---

## 主从切换流程

### 自动故障转移

当主节点故障时,副本集自动执行故障转移。

**详细时间线:**

```
T0:00 - 主节点正常运行
T0:05 - 主节点突然崩溃
T0:07 - 从节点1: 心跳失败 (1次)
T0:09 - 从节点1: 心跳失败 (2次)
T0:11 - 从节点1: 心跳失败 (3次)
T0:13 - 从节点1: 心跳失败 (4次)
T0:15 - 从节点1: 检测到主节点故障,发起选举
T0:16 - 从节点1: 收集投票
T0:17 - 从节点1: 获得大多数票,成为新主节点
T0:18 - 应用重新连接,写操作恢复
T0:20 - 旧主节点恢复,作为从节点加入

总故障转移时间: 约 12-15 秒
```

**实际测试:**

```javascript
// 测试脚本: 监控主从切换
function monitorFailover() {
  const startTime = new Date()
  
  while (true) {
    try {
      const status = rs.status()
      const primary = status.members.find(m => m.stateStr === "PRIMARY")
      
      print(`${new Date() - startTime}ms: Primary = ${primary ? primary.name : "NONE"}`)
      
      if (!primary) {
        print(">>> 正在选举新主节点...")
      }
      
      sleep(1000)
    } catch (e) {
      print(`错误: ${e}`)
      sleep(1000)
    }
  }
}

// 在另一个终端杀掉主节点进程
// kill -9 <primary_pid>

// 观察输出:
// 0ms: Primary = mongodb1:27017
// 1000ms: Primary = mongodb1:27017
// ...
// 10000ms: Primary = mongodb1:27017
// 11000ms: Primary = NONE
// >>> 正在选举新主节点...
// 12000ms: Primary = NONE
// 13000ms: Primary = mongodb2:27017  ← 新主节点选出
```

### 手动故障转移测试

```javascript
// 连接到主节点
mongo mongodb1:27017

// 确认当前状态
rs.isMaster()

// 输出
{
  "ismaster": true,
  "secondary": false,
  "primary": "mongodb1:27017",
  "hosts": [
    "mongodb1:27017",
    "mongodb2:27017",
    "mongodb3:27017"
  ]
}

// 模拟主节点故障
db.adminCommand({ shutdown: 1 })

// 在另一个节点观察选举
mongo mongodb2:27017
rs.status()

// 新主节点将在 10-15 秒内选出
```

### 网络分区场景

**场景1: 少数派分区**

```
网络分区发生:
┌──────────────────┐    ┌──────────────────┐
│ 主节点 + 从节点1  │ ││ │    从节点2       │
│   (大多数)       │ ││ │    (少数)       │
└──────────────────┘    └──────────────────┘

结果:
- 左侧(大多数): 主节点继续提供服务
- 右侧(少数): 从节点2变为 SECONDARY,拒绝读写
```

**场景2: 对等分区**

```
3节点副本集网络分区:
┌──────────┐    ┌──────────┐    ┌──────────┐
│  主节点  │ ││ │  从节点1  │ ││ │  从节点2  │
└──────────┘    └──────────┘    └──────────┘

结果:
- 所有节点都无法获得大多数
- 主节点降级为 SECONDARY
- 副本集变为只读状态
```

**场景3: 跨数据中心分区**

```javascript
// 5节点配置: DC1(3节点) + DC2(2节点)
rs.initiate({
  _id: "myReplicaSet",
  members: [
    { _id: 0, host: "dc1-node1:27017", priority: 2 },
    { _id: 1, host: "dc1-node2:27017", priority: 2 },
    { _id: 2, host: "dc1-node3:27017", priority: 2 },
    { _id: 3, host: "dc2-node1:27017", priority: 1 },
    { _id: 4, host: "dc2-node2:27017", priority: 1 }
  ]
})

// DC1 与 DC2 网络中断
// DC1 有大多数(3/5),继续提供服务
// DC2 没有大多数(2/5),变为只读
```

---

## 心跳与故障检测

### 心跳机制

副本集成员之间通过心跳检测彼此的状态。

```javascript
// 心跳配置
cfg = rs.conf()
cfg.settings = {
  heartbeatIntervalMillis: 2000,   // 每2秒发送一次心跳
  heartbeatTimeoutSecs: 10,        // 10秒无响应判定故障
  electionTimeoutMillis: 10000     // 10秒后发起选举
}
rs.reconfig(cfg)
```

**心跳信息内容:**

```javascript
{
  "replSetHeartbeat": "myReplicaSet",
  "configVersion": 5,
  "hbmsg": "",
  "from": "mongodb2:27017",
  "fromId": 1,
  "term": 3,
  "primaryId": 0
}
```

### 故障检测流程

```
正常状态:
主节点 ←心跳→ 从节点1
   ↑              ↑
   └──心跳→ 从节点2 ┘

主节点故障后:
从节点1 → 主节点: 心跳 (无响应)
从节点2 → 主节点: 心跳 (无响应)
   ↓              ↓
从节点1 ←→ 从节点2: 确认主节点不可达
   ↓
发起选举
```

**故障检测参数调优:**

```javascript
// 快速故障转移(适合局域网)
cfg.settings = {
  heartbeatIntervalMillis: 1000,   // 1秒心跳
  heartbeatTimeoutSecs: 5,         // 5秒超时
  electionTimeoutMillis: 5000      // 5秒选举
}
// 故障转移时间: ~7-10秒

// 慢速网络(适合跨数据中心)
cfg.settings = {
  heartbeatIntervalMillis: 5000,   // 5秒心跳
  heartbeatTimeoutSecs: 30,        // 30秒超时
  electionTimeoutMillis: 30000     // 30秒选举
}
// 故障转移时间: ~40-60秒
```

### 健康状态监控

```javascript
// 查看所有成员健康状态
rs.status().members.forEach(m => {
  print(`${m.name}:`)
  print(`  状态: ${m.stateStr}`)
  print(`  健康: ${m.health === 1 ? "正常" : "异常"}`)
  print(`  Ping: ${m.pingMs}ms`)
  print(`  上线时间: ${m.uptime}秒`)
  if (m.lastHeartbeat) {
    print(`  最后心跳: ${m.lastHeartbeat}`)
    print(`  最后心跳消息: ${m.lastHeartbeatMessage || "无"}`)
  }
})

// 输出示例
// mongodb1:27017:
//   状态: PRIMARY
//   健康: 正常
//   Ping: 0ms
//   上线时间: 86400秒
//
// mongodb2:27017:
//   状态: SECONDARY
//   健康: 正常
//   Ping: 5ms
//   上线时间: 86400秒
//   最后心跳: 2025-01-18T10:00:00.000Z
//   最后心跳消息: 无
```

---

## Oplog 与数据同步

### Oplog 结构

Oplog (操作日志) 是副本集数据同步的核心,存储在 `local.oplog.rs` 集合中。

```javascript
// 查看 Oplog
use local
db.oplog.rs.find().sort({$natural: -1}).limit(5).pretty()

// Oplog 条目示例
{
  "ts": Timestamp(1705555200, 1),    // 时间戳
  "t": NumberLong(5),                 // 选举任期
  "h": NumberLong("1234567890"),      // 哈希值
  "v": 2,                             // 版本
  "op": "i",                          // 操作类型
  "ns": "mydb.users",                 // 命名空间
  "ui": UUID("..."),                  // 集合UUID
  "o": {                              // 操作内容
    "_id": ObjectId("..."),
    "name": "张三",
    "age": 28
  },
  "wall": ISODate("2025-01-18T10:00:00.000Z")  // 时钟时间
}
```

**操作类型:**

| op | 操作 | 说明 |
|----|------|------|
| `i` | insert | 插入文档 |
| `u` | update | 更新文档 |
| `d` | delete | 删除文档 |
| `c` | command | 数据库命令 |
| `n` | noop | 空操作(心跳) |

### 数据同步流程

```
1. 主节点写入数据
   ↓
2. 写入操作记录到 Oplog
   ↓
3. 从节点拉取 Oplog (每秒轮询)
   ↓
4. 从节点应用 Oplog 操作
   ↓
5. 从节点更新同步位置
```

**详细同步过程:**

```javascript
// 主节点
db.users.insertOne({ name: "张三" })

// 主节点 Oplog
{
  "ts": Timestamp(1705555200, 1),
  "op": "i",
  "ns": "mydb.users",
  "o": { "_id": ObjectId("..."), "name": "张三" }
}

// 从节点拉取并应用
// 1. 从节点连接到主节点
// 2. 查询 Oplog: db.oplog.rs.find({ ts: { $gt: 最后同步位置 } })
// 3. 应用操作: db.users.insertOne({ "_id": ObjectId("..."), "name": "张三" })
// 4. 更新同步位置
```

### 初始同步 (Initial Sync)

新节点加入副本集时需要执行初始同步。

```
步骤1: 克隆数据
- 从同步源节点复制所有数据库
- 除了 local 数据库

步骤2: 应用 Oplog
- 克隆期间的新操作通过 Oplog 同步
- 确保数据一致性

步骤3: 构建索引
- 在本地重建所有索引

步骤4: 完成同步
- 节点状态变为 SECONDARY
- 开始正常的增量同步
```

```javascript
// 添加新节点
rs.add("mongodb4:27017")

// 查看初始同步进度
rs.status().members[3]

// 输出示例
{
  "_id": 3,
  "name": "mongodb4:27017",
  "stateStr": "STARTUP2",           // 正在初始同步
  "infoMessage": "initial sync cloning db: mydb",
  "syncSourceHost": "mongodb1:27017",
  "initialSyncStatus": {
    "fetchedMissingDocs": 0,
    "appliedOps": 1000,
    "totalInitialSyncElapsedMillis": 30000,
    "databases": {
      "mydb": {
        "clonedCollections": 5,
        "clonedBytes": 1048576
      }
    }
  }
}
```

### 复制延迟监控

```javascript
// 方法1: 使用 rs.printSecondaryReplicationInfo()
rs.printSecondaryReplicationInfo()

// 输出
// source: mongodb2:27017
//   syncedTo: Thu Jan 18 2025 10:00:00 GMT+0800 (CST)
//   0 secs (0 hrs) behind the primary
// source: mongodb3:27017
//   syncedTo: Thu Jan 18 2025 09:59:50 GMT+0800 (CST)
//   10 secs (0 hrs) behind the primary

// 方法2: 编程方式检查
function checkReplicationLag() {
  const status = rs.status()
  const primary = status.members.find(m => m.stateStr === "PRIMARY")
  const primaryOptime = primary.optimeDate
  
  status.members.forEach(m => {
    if (m.stateStr === "SECONDARY") {
      const lag = (primaryOptime - m.optimeDate) / 1000
      print(`${m.name}: ${lag.toFixed(2)}秒延迟`)
      
      if (lag > 10) {
        print(`  警告: 复制延迟过大!`)
      }
    }
  })
}

checkReplicationLag()

// 方法3: 使用 db.getReplicationInfo()
rs.printReplicationInfo()

// 输出
// configured oplog size:   10240MB
// log length start to end: 3600secs (1hrs)
// oplog first event time:  Thu Jan 18 2025 09:00:00 GMT+0800
// oplog last event time:   Thu Jan 18 2025 10:00:00 GMT+0800
// now:                     Thu Jan 18 2025 10:00:05 GMT+0800
```

### Oplog 大小配置

```javascript
// 查看当前 Oplog 大小
use local
db.oplog.rs.stats().maxSize / 1024 / 1024  // MB

// 修改 Oplog 大小(需要逐个节点操作)
// 1. 关闭从节点
db.adminCommand({ shutdown: 1 })

// 2. 以单机模式启动
mongod --dbpath /data/db --port 27017

// 3. 修改 Oplog 大小
use local
db.adminCommand({ replSetResizeOplog: 1, size: 20480 })  // 20GB

// 4. 重启为副本集模式
```

**Oplog 大小计算:**

```
Oplog大小 = 峰值写入速率 × 维护窗口 × 安全系数

示例:
- 峰值写入: 200MB/小时
- 维护窗口: 24小时
- 安全系数: 2
- Oplog = 200 × 24 × 2 = 9600 MB (约10GB)
```

---

## 手动主从切换

### 主动降级 (stepDown)

管理员可以手动让主节点降级,触发选举。

```javascript
// 连接到主节点
mongo mongodb1:27017/admin

// 方法1: 临时降级(60秒内不能重新成为主节点)
rs.stepDown(60)

// 方法2: 降级并指定从节点追赶时间
rs.stepDown(
  60,      // 降级持续时间(秒)
  30       // 等待从节点追赶的最长时间(秒)
)

// 方法3: 强制降级(即使从节点未追赶)
rs.stepDown(60, 0)

// 降级后查看状态
rs.status()
```

**stepDown 过程:**

```
T0: 执行 rs.stepDown(60)
    ↓
T0: 主节点停止接受新的写操作
    ↓
T0: 等待从节点追赶(最多30秒)
    ↓
T0+5: 从节点追赶完成
    ↓
T0+5: 主节点降级为 SECONDARY
    ↓
T0+5: 其他节点发起选举
    ↓
T0+10: 选出新主节点
    ↓
T0+60: 旧主节点可以重新参与选举
```

### 维护模式切换

进行节点维护时的最佳实践:

```javascript
// 场景: 需要升级 mongodb2 节点

// 步骤1: 确保 mongodb2 不是主节点
rs.status().members.find(m => m.name === "mongodb2:27017").stateStr
// 如果是主节点,执行 rs.stepDown()

// 步骤2: 如果是从节点,直接关闭
mongo mongodb2:27017/admin
db.adminCommand({ shutdown: 1 })

// 步骤3: 执行维护(升级、配置等)
// ...

// 步骤4: 重新启动
mongod --config /etc/mongod.conf

// 步骤5: 验证节点重新加入
rs.status()

// 步骤6: 等待数据同步完成
rs.printSecondaryReplicationInfo()
```

### 计划内切换流程

```javascript
// 完整的主从切换流程(零停机)

// 步骤1: 检查副本集健康状态
rs.status()

// 步骤2: 确认所有从节点已同步
rs.printSecondaryReplicationInfo()
// 确保所有从节点延迟 < 5秒

// 步骤3: 降低主节点优先级(可选)
cfg = rs.conf()
cfg.members[0].priority = 0.5  // 降低当前主节点优先级
cfg.members[1].priority = 2    // 提高目标节点优先级
rs.reconfig(cfg)

// 步骤4: 执行主节点降级
rs.stepDown(120)

// 步骤5: 等待选举完成(通常5-15秒)
sleep(15000)
rs.status()

// 步骤6: 验证新主节点
rs.isMaster()

// 步骤7: 恢复优先级配置(如需要)
cfg = rs.conf()
cfg.members[0].priority = 1
cfg.members[1].priority = 1
rs.reconfig(cfg)
```

### 强制重新配置 (紧急情况)

```javascript
// 警告: 仅在紧急情况使用,可能导致数据丢失

// 场景: 大多数节点不可用,无法选举
// 3节点副本集,2个节点崩溃

// 步骤1: 连接到唯一存活的节点
mongo mongodb3:27017/admin

// 步骤2: 获取当前配置
cfg = rs.conf()

// 步骤3: 移除故障节点
cfg.members = [cfg.members[2]]  // 只保留当前节点

// 步骤4: 强制重新配置
rs.reconfig(cfg, { force: true })

// 步骤5: 节点变为主节点
rs.status()

// 注意: 这会形成新的副本集历史,可能与旧节点冲突
```

---

## 主从切换的影响

### 对应用的影响

#### 写操作中断

```javascript
// 主从切换期间的写操作失败
try {
  db.users.insertOne({ name: "张三" })
} catch (e) {
  print(e)
  // NotMaster: not master
  // 或
  // NotMasterOrSecondary: not master or secondary
}
```

**影响时间:**
- **检测时间**: 5-10秒(心跳超时)
- **选举时间**: 5-10秒(选举过程)
- **总中断**: 10-20秒

**应用层处理:**

```javascript
// Node.js 示例: 自动重试
const { MongoClient } = require('mongodb')

const client = new MongoClient(uri, {
  replicaSet: 'myReplicaSet',
  retryWrites: true,           // 自动重试写操作
  retryReads: true,            // 自动重试读操作
  serverSelectionTimeoutMS: 30000,
  heartbeatFrequencyMS: 2000
})

// 带重试的写入函数
async function insertWithRetry(data, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      await db.users.insertOne(data)
      return
    } catch (error) {
      if (error.name === 'MongoNotConnectedError' || 
          error.code === 10107) {  // NotMaster
        console.log(`重试 ${i + 1}/${maxRetries}...`)
        await new Promise(resolve => setTimeout(resolve, 2000))
      } else {
        throw error
      }
    }
  }
  throw new Error('写入失败,已达最大重试次数')
}
```

#### 读操作影响

```javascript
// 读偏好配置
const client = new MongoClient(uri, {
  readPreference: 'primaryPreferred'  // 优先主节点,主节点不可用时读从节点
})

// 不同读偏好在切换时的行为:
// - primary: 主节点不可用时读取失败
// - primaryPreferred: 自动切换到从节点
// - secondary: 不受影响,继续从从节点读
// - secondaryPreferred: 不受影响
// - nearest: 从最近的节点读,影响最小
```

### 数据一致性

#### 回滚风险

```javascript
// 危险场景: w:1 写入未同步即切换

// T0: 主节点A接收写入
db.orders.insertOne(
  { orderId: 123, amount: 999.99 },
  { writeConcern: { w: 1 } }  // 仅主节点确认
)
// 写入成功返回

// T1: 主节点A崩溃(数据未同步到从节点)

// T2: 从节点B被选为新主节点

// T3: 原主节点A恢复
// orderId:123 的数据被回滚!
// 回滚的数据保存在 /data/db/rollback/ 目录
```

**防止回滚:**

```javascript
// 使用 w:"majority"
db.orders.insertOne(
  { orderId: 123, amount: 999.99 },
  { writeConcern: { w: "majority", wtimeout: 5000 } }
)

// 这确保数据已复制到大多数节点
// 新主节点必定包含这条数据
// 不会回滚
```

#### 读一致性

```javascript
// 场景: 读到未提交的数据

// T0: 主节点写入
db.users.insertOne({ name: "张三" }, { writeConcern: { w: 1 } })

// T1: 从主节点读取(成功)
db.users.findOne({ name: "张三" })  // 找到

// T2: 主节点崩溃,数据回滚

// T3: 从新主节点读取(失败)
db.users.findOne({ name: "张三" })  // 找不到!

// 解决方案: 使用 readConcern: "majority"
db.users.find({ name: "张三" }).readConcern("majority")
// 只读取已被大多数节点确认的数据
```

### 性能影响

```javascript
// 切换期间的性能监控

function monitorDuringFailover() {
  const metrics = {
    writesSucceeded: 0,
    writesFailed: 0,
    readsSucceeded: 0,
    readsFailed: 0,
    latencies: []
  }
  
  const interval = setInterval(() => {
    // 尝试写入
    const writeStart = Date.now()
    try {
      db.test.insertOne({ ts: new Date() })
      metrics.writesSucceeded++
      metrics.latencies.push(Date.now() - writeStart)
    } catch (e) {
      metrics.writesFailed++
    }
    
    // 尝试读取
    try {
      db.test.findOne()
      metrics.readsSucceeded++
    } catch (e) {
      metrics.readsFailed++
    }
    
    // 打印统计
    print(`写成功: ${metrics.writesSucceeded}, 写失败: ${metrics.writesFailed}`)
    print(`读成功: ${metrics.readsSucceeded}, 读失败: ${metrics.readsFailed}`)
    if (metrics.latencies.length > 0) {
      const avg = metrics.latencies.reduce((a,b) => a+b) / metrics.latencies.length
      print(`平均延迟: ${avg.toFixed(2)}ms`)
    }
  }, 1000)
  
  return () => clearInterval(interval)
}

// 运行监控
const stop = monitorDuringFailover()

// 在另一终端触发切换
// rs.stepDown(60)

// 典型输出:
// T0-10: 写成功: 10, 写失败: 0, 平均延迟: 5ms
// T11-15: 写成功: 0, 写失败: 5 (选举中)
// T16-20: 写成功: 5, 写失败: 0, 平均延迟: 8ms (新主节点)
```

---

## 高可用性配置

### 推荐配置

#### 生产环境标准配置

```javascript
// 5节点副本集(推荐)
rs.initiate({
  _id: "prodReplicaSet",
  members: [
    { 
      _id: 0, 
      host: "mongodb1:27017",
      priority: 2          // 优先主节点
    },
    { 
      _id: 1, 
      host: "mongodb2:27017",
      priority: 2
    },
    { 
      _id: 2, 
      host: "mongodb3:27017",
      priority: 1
    },
    { 
      _id: 3, 
      host: "mongodb4:27017",
      priority: 1
    },
    { 
      _id: 4, 
      host: "mongodb5:27017",
      priority: 0,         // 延迟节点
      slaveDelay: 3600,    // 延迟1小时
      hidden: true         // 隐藏节点
    }
  ],
  settings: {
    chainingAllowed: true,  // 允许链式复制
    heartbeatIntervalMillis: 2000,
    heartbeatTimeoutSecs: 10,
    electionTimeoutMillis: 10000,
    catchUpTimeoutMillis: -1,  // 新主节点等待追赶的时间(-1=无限)
    getLastErrorDefaults: {
      w: "majority",
      wtimeout: 5000
    }
  }
})
```

#### 跨数据中心配置

```javascript
// 主数据中心3节点 + 灾备数据中心2节点
rs.initiate({
  _id: "geoReplicaSet",
  members: [
    // 主数据中心(北京)
    { 
      _id: 0, 
      host: "beijing-1:27017",
      priority: 3,
      tags: { dc: "beijing", zone: "bj-1" }
    },
    { 
      _id: 1, 
      host: "beijing-2:27017",
      priority: 3,
      tags: { dc: "beijing", zone: "bj-2" }
    },
    { 
      _id: 2, 
      host: "beijing-3:27017",
      priority: 2,
      tags: { dc: "beijing", zone: "bj-3" }
    },
    
    // 灾备数据中心(上海)
    { 
      _id: 3, 
      host: "shanghai-1:27017",
      priority: 1,
      tags: { dc: "shanghai", zone: "sh-1" }
    },
    { 
      _id: 4, 
      host: "shanghai-2:27017",
      priority: 1,
      tags: { dc: "shanghai", zone: "sh-2" }
    }
  ],
  settings: {
    getLastErrorDefaults: {
      w: "majority",
      wtimeout: 10000  // 跨数据中心需要更长超时
    }
  }
})

// 配置读偏好(优先本地数据中心)
db.users.find().readPreference(
  "nearest",
  [{ dc: "beijing" }]  // 优先北京数据中心
)
```

### 连接字符串最佳实践

```javascript
// Node.js 连接字符串
const uri = "mongodb://mongodb1:27017,mongodb2:27017,mongodb3:27017/" +
            "?replicaSet=myReplicaSet" +
            "&readPreference=primaryPreferred" +
            "&w=majority" +
            "&wtimeoutMS=5000" +
            "&maxPoolSize=50" +
            "&retryWrites=true" +
            "&retryReads=true" +
            "&serverSelectionTimeoutMS=30000" +
            "&heartbeatFrequencyMS=2000"

const client = new MongoClient(uri, {
  // 连接池配置
  maxPoolSize: 50,
  minPoolSize: 10,
  maxIdleTimeMS: 60000,
  
  // 服务器选择
  serverSelectionTimeoutMS: 30000,
  
  // 心跳配置
  heartbeatFrequencyMS: 2000,
  
  // 重试配置
  retryWrites: true,
  retryReads: true,
  
  // 默认写关注
  writeConcern: {
    w: 'majority',
    wtimeout: 5000
  },
  
  // 默认读偏好
  readPreference: 'primaryPreferred'
})
```

### 监控与告警

```javascript
// 监控脚本
function setupMonitoring() {
  // 1. 检查副本集健康
  function checkHealth() {
    const status = rs.status()
    
    // 检查是否有主节点
    const primary = status.members.find(m => m.stateStr === "PRIMARY")
    if (!primary) {
      alert("严重: 没有主节点!")
    }
    
    // 检查节点健康
    status.members.forEach(m => {
      if (m.health !== 1) {
        alert(`警告: ${m.name} 健康状态异常`)
      }
    })
    
    // 检查复制延迟
    if (primary) {
      status.members.forEach(m => {
        if (m.stateStr === "SECONDARY") {
          const lag = (primary.optimeDate - m.optimeDate) / 1000
          if (lag > 10) {
            alert(`警告: ${m.name} 复制延迟 ${lag}秒`)
          }
        }
      })
    }
  }
  
  // 2. 检查 Oplog 空间
  function checkOplog() {
    use local
    const stats = db.oplog.rs.stats()
    const oplogTime = db.getReplicationInfo()
    
    const hours = oplogTime.logSizeMB / oplogTime.usedMB * oplogTime.timeDiff / 3600
    
    if (hours < 24) {
      alert(`警告: Oplog 仅能支持 ${hours.toFixed(1)} 小时`)
    }
  }
  
  // 3. 定期执行
  setInterval(() => {
    checkHealth()
    checkOplog()
  }, 60000)  // 每分钟
}

function alert(message) {
  print(`[${new Date().toISOString()}] ${message}`)
  // 实际环境中发送到监控系统
}
```

---

## 常见问题 FAQ

### 1. 主从切换需要多长时间?会影响服务吗?

**典型切换时间:**

```
组件           时间        说明
------------------------------------
故障检测       5-10秒      心跳超时
选举过程       5-10秒      投票和确认
客户端重连     1-5秒       驱动重新发现主节点
------------------------------------
总计           11-25秒     通常15秒左右
```

**影响分析:**

**对写操作:**
- ❌ 切换期间写操作会失败
- ✅ 使用 `retryWrites: true` 可自动重试
- ⏱️ 中断时间: 10-20秒

```javascript
// 应用层最佳实践
const client = new MongoClient(uri, {
  retryWrites: true,  // 自动重试写操作
  w: 'majority',      // 防止回滚
  wtimeout: 5000
})

// 手动重试示例
async function robustWrite(data) {
  const maxRetries = 3
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await db.collection.insertOne(data)
    } catch (err) {
      if (err.code === 10107 && i < maxRetries - 1) {  // NotMaster
        await sleep(2000)
        continue
      }
      throw err
    }
  }
}
```

**对读操作:**
- ✅ 使用 `readPreference: "primaryPreferred"` 几乎无影响
- ⚠️ 使用 `readPreference: "primary"` 会短暂失败
- ✅ 使用 `readPreference: "secondary"` 完全不受影响

**减少影响的方法:**

1. **优化心跳检测**
```javascript
cfg = rs.conf()
cfg.settings.heartbeatIntervalMillis = 1000  // 1秒心跳
cfg.settings.electionTimeoutMillis = 5000    // 5秒选举
rs.reconfig(cfg)
// 切换时间减少到 7-12秒
```

2. **使用合适的读偏好**
```javascript
db.collection.find().readPreference("primaryPreferred")
```

3. **应用层重试机制**
```javascript
// 设置合理的超时和重试
const client = new MongoClient(uri, {
  serverSelectionTimeoutMS: 30000,
  retryWrites: true,
  retryReads: true
})
```

### 2. 如何判断当前哪个节点是主节点?

**方法1: 使用 rs.status()**

```javascript
// 最全面的状态信息
rs.status().members.forEach(m => {
  if (m.stateStr === "PRIMARY") {
    print(`主节点: ${m.name}`)
  }
})
```

**方法2: 使用 rs.isMaster()**

```javascript
// 快速检查
const status = rs.isMaster()
print(`主节点: ${status.primary}`)
print(`当前节点是主节点: ${status.ismaster}`)

// 输出示例
// 主节点: mongodb1:27017
// 当前节点是主节点: true
```

**方法3: 使用 db.hello()**

```javascript
// MongoDB 5.0+ 推荐方法
const hello = db.hello()
print(`主节点: ${hello.primary}`)
print(`所有节点: ${hello.hosts.join(", ")}`)
```

**方法4: 编程方式(Node.js)**

```javascript
const { MongoClient } = require('mongodb')

async function findPrimary() {
  const client = await MongoClient.connect(uri)
  const admin = client.db().admin()
  const status = await admin.command({ replSetGetStatus: 1 })
  
  const primary = status.members.find(m => m.stateStr === "PRIMARY")
  console.log(`主节点: ${primary.name}`)
  
  await client.close()
}
```

**方法5: 命令行快速查看**

```bash
# 使用 mongo shell
mongo --eval "rs.isMaster().primary"

# 输出: mongodb1:27017

# 或使用 mongosh
mongosh --eval "db.hello().primary"
```

**监控主节点变化:**

```javascript
// 持续监控主节点
function monitorPrimary() {
  let lastPrimary = null
  
  setInterval(() => {
    const current = rs.isMaster().primary
    
    if (current !== lastPrimary) {
      print(`[${new Date().toISOString()}] 主节点变化: ${lastPrimary} → ${current}`)
      lastPrimary = current
    } else {
      print(`[${new Date().toISOString()}] 主节点: ${current}`)
    }
  }, 5000)
}

monitorPrimary()
```

### 3. 副本集中的节点数量应该是奇数还是偶数?

**强烈推荐奇数节点!**

**原因1: 选举需要大多数票**

```
3节点(奇数): 需要2票 = (3/2) + 1
4节点(偶数): 需要3票 = (4/2) + 1

容错能力:
3节点: 可容忍1个节点故障,仍有2个节点(大多数)
4节点: 可容忍1个节点故障,仍有3个节点(大多数)

结论: 3节点和4节点容错能力相同,但4节点成本更高!
```

**原因2: 网络分区问题**

```
偶数节点的问题:
4节点分区为2+2:
┌─────────┐  ┌─────────┐
│ 节点1,2  │  │ 节点3,4  │
│ 无大多数 │  │ 无大多数 │
└─────────┘  └─────────┘
结果: 整个集群不可用!

奇数节点的优势:
3节点分区为2+1:
┌─────────┐  ┌─────────┐
│ 节点1,2  │  │  节点3   │
│ 有大多数 │  │ 无大多数 │
│ 继续服务 │  │ 只读模式 │
└─────────┘  └─────────┘
结果: 至少一侧可以继续服务
```

**推荐配置:**

```javascript
// ✅ 推荐: 3节点
rs.initiate({
  _id: "myReplicaSet",
  members: [
    { _id: 0, host: "mongodb1:27017" },
    { _id: 1, host: "mongodb2:27017" },
    { _id: 2, host: "mongodb3:27017" }
  ]
})

// ✅ 推荐: 5节点(高可用)
rs.initiate({
  _id: "myReplicaSet",
  members: [
    { _id: 0, host: "mongodb1:27017" },
    { _id: 1, host: "mongodb2:27017" },
    { _id: 2, host: "mongodb3:27017" },
    { _id: 3, host: "mongodb4:27017" },
    { _id: 4, host: "mongodb5:27017" }
  ]
})

// ⚠️ 不推荐: 4节点
// 如果只有4个节点,使用3个数据节点 + 1个仲裁节点

// ❌ 禁止: 2节点(无容错能力)
```

**特殊场景:**

```javascript
// 场景: 只有2个服务器
// 方案: 2数据节点 + 1仲裁节点(轻量级)

rs.initiate({
  _id: "myReplicaSet",
  members: [
    { _id: 0, host: "mongodb1:27017" },
    { _id: 1, host: "mongodb2:27017" },
    { _id: 2, host: "arbiter:27017", arbiterOnly: true }
  ]
})

// 仲裁节点不存储数据,资源占用极小
// 可以和应用服务器共存
```

**成本对比:**

```
需求: 容忍1个节点故障

方案A: 3个数据节点
- 节点数: 3
- 成本: 3x
- 容错: 1
- 推荐: ✅

方案B: 4个数据节点
- 节点数: 4
- 成本: 4x
- 容错: 1
- 推荐: ❌ 浪费资源

方案C: 2个数据节点 + 1个仲裁节点
- 节点数: 3
- 成本: 2.1x (仲裁节点很轻)
- 容错: 1
- 推荐: ⚠️ 仅资源受限时
```

### 4. 主节点故障后,未同步的数据会丢失吗?

**答案: 取决于写关注级别。**

**场景1: 使用 w: 1 (仅主节点确认)**

```javascript
// T0: 主节点接收写入
db.orders.insertOne(
  { orderId: 123, amount: 999.99 },
  { writeConcern: { w: 1 } }  // 仅主节点内存确认
)
// 立即返回成功

// T1: 主节点崩溃(未同步到从节点)

// T2: 从节点成为新主节点

// 结果: orderId: 123 的数据丢失!
```

**场景2: 使用 w: "majority" (大多数确认)**

```javascript
// T0: 主节点接收写入
db.orders.insertOne(
  { orderId: 123, amount: 999.99 },
  { writeConcern: { w: "majority", wtimeout: 5000 } }
)

// 主节点等待至少一个从节点确认
// 数据已复制到至少2个节点(3节点集群)

// T1: 主节点崩溃

// T2: 从节点成为新主节点(已有数据)

// 结果: orderId: 123 的数据安全!
```

**数据丢失的真实案例:**

```javascript
// 测试脚本
async function testDataLoss() {
  // 使用 w:1
  for (let i = 0; i < 1000; i++) {
    await db.test.insertOne(
      { seq: i, data: "important" },
      { writeConcern: { w: 1 } }
    )
  }
  
  // 立即杀掉主节点
  // 在另一终端: db.adminCommand({ shutdown: 1 })
  
  // 等待新主节点选出
  await sleep(15000)
  
  // 检查数据
  const count = await db.test.countDocuments()
  print(`写入: 1000, 实际: ${count}, 丢失: ${1000 - count}`)
  
  // 典型结果: 丢失最后 5-20 条数据
}
```

**回滚目录:**

```bash
# 回滚的数据保存在这里
ls /var/lib/mongodb/rollback/

# 文件格式
# orderId.123.bson
# 可以用 bsondump 查看
bsondump /var/lib/mongodb/rollback/orders.2025-01-18T10-00-00.0.bson
```

**最佳实践:**

```javascript
// 关键数据使用 w:"majority"
db.orders.insertOne(orderData, { 
  writeConcern: { w: "majority", wtimeout: 5000 } 
})

// 非关键数据可以用 w:1
db.logs.insertOne(logData, { 
  writeConcern: { w: 1 } 
})

// 特别关键的数据
db.financialTransactions.insertOne(txData, { 
  writeConcern: { w: "majority", j: true, wtimeout: 5000 } 
})
```

**数据丢失统计:**

```
写关注级别      主节点崩溃数据丢失风险
------------------------------------------
w: 0           100% (不等待任何确认)
w: 1           10-50% (取决于同步速度)
w: 1, j: true  < 1% (Journal 保护)
w: "majority"  0% (已复制到大多数)
```

### 5. 如何进行副本集的滚动升级而不影响服务?

**完整滚动升级流程:**

**准备阶段:**

```javascript
// 1. 检查副本集健康
rs.status()

// 2. 确认所有节点同步
rs.printSecondaryReplicationInfo()
// 确保延迟 < 10秒

// 3. 备份数据(可选但推荐)
mongodump --out=/backup/before-upgrade

// 4. 测试新版本(在测试环境)
```

**升级从节点:**

```javascript
// 步骤1: 升级第一个从节点
// 1.1 连接到从节点
mongo mongodb2:27017/admin

// 1.2 确认是从节点
rs.isMaster().secondary  // 应为 true

// 1.3 正常关闭
db.adminCommand({ shutdown: 1 })

// 1.4 升级二进制文件
sudo systemctl stop mongod
sudo apt-get install -y mongodb-org=6.0.0  # 或使用二进制包
sudo systemctl start mongod

// 1.5 验证升级
mongo mongodb2:27017
db.version()  // 确认新版本

// 1.6 确认节点重新加入
rs.status()

// 1.7 等待数据同步
rs.printSecondaryReplicationInfo()

// 步骤2: 重复升级其他从节点
// 逐个升级 mongodb3, mongodb4 等
```

**升级主节点:**

```javascript
// 步骤3: 升级主节点(最后)
// 3.1 连接到主节点
mongo mongodb1:27017/admin

// 3.2 确认所有从节点已升级
rs.status().members.forEach(m => {
  if (m.stateStr === "SECONDARY") {
    print(`${m.name} 版本: ${m.version || "需连接查询"}`)
  }
})

// 3.3 主节点降级(触发选举)
rs.stepDown(120)  // 120秒内不会重新成为主节点

// 3.4 等待新主节点选出(通常5-15秒)
sleep(15000)
rs.status()

// 3.5 升级原主节点(现在是从节点)
db.adminCommand({ shutdown: 1 })

// 在服务器上执行
sudo systemctl stop mongod
sudo apt-get install -y mongodb-org=6.0.0
sudo systemctl start mongod

// 3.6 验证
mongo mongodb1:27017
db.version()
rs.status()
```

**验证与监控:**

```javascript
// 完整性检查
function verifyUpgrade() {
  const status = rs.status()
  
  print("=== 升级验证 ===")
  
  // 1. 检查所有节点在线
  const offline = status.members.filter(m => m.health !== 1)
  if (offline.length > 0) {
    print(`警告: ${offline.length} 个节点离线`)
  } else {
    print("✓ 所有节点在线")
  }
  
  // 2. 检查是否有主节点
  const primary = status.members.find(m => m.stateStr === "PRIMARY")
  if (primary) {
    print(`✓ 主节点: ${primary.name}`)
  } else {
    print("✗ 没有主节点!")
  }
  
  // 3. 检查版本一致性
  // (需要连接到每个节点查询)
  
  // 4. 检查复制延迟
  status.members.forEach(m => {
    if (m.stateStr === "SECONDARY") {
      const lag = (primary.optimeDate - m.optimeDate) / 1000
      if (lag > 10) {
        print(`警告: ${m.name} 延迟 ${lag}秒`)
      } else {
        print(`✓ ${m.name} 同步正常`)
      }
    }
  })
  
  print("=== 验证完成 ===")
}

verifyUpgrade()
```

**关键注意事项:**

1. **版本兼容性**: 只能升级到下一个主版本
```
✓ 4.4 → 5.0 → 6.0
✗ 4.4 → 6.0 (跳过主版本)
```

2. **特性兼容性版本(FCV)**
```javascript
// 升级后设置FCV
db.adminCommand({ setFeatureCompatibilityVersion: "6.0" })

// 这会启用新版本特性
// 警告: 设置后不能降级到旧版本
```

3. **升级顺序**: 从节点 → 主节点
4. **逐个升级**: 一次只升级一个节点
5. **监控日志**: 关注错误和警告
6. **保持备份**: 升级前备份数据

**零停机升级时间估算:**

```
3节点副本集升级时间:
- 从节点1: 5-10分钟
- 从节点2: 5-10分钟
- 主节点降级: 10-20秒
- 主节点升级: 5-10分钟
---------------------------
总计: 15-30分钟

写入中断时间: 仅主节点降级时的 10-20秒
```

---

## 总结

MongoDB 的主从切换是通过**副本集(Replica Set)**实现的高可用性机制:

**核心特性:**
- ✅ **自动故障转移**: 主节点故障时自动选举新主节点
- ✅ **基于 Raft 的选举**: 分布式共识保证一致性
- ✅ **快速恢复**: 通常 10-20 秒完成切换
- ✅ **零停机维护**: 支持滚动升级

**关键概念:**
- **Primary**: 处理所有写操作
- **Secondary**: 复制数据,参与选举
- **Arbiter**: 仅投票,不存储数据
- **Oplog**: 操作日志,实现数据同步

**最佳实践:**
- 使用奇数节点(3或5个)
- 配置写关注 `w: "majority"` 防止数据丢失
- 跨数据中心部署提高容灾能力
- 应用层实现重试机制
- 持续监控副本集健康状态

理解主从切换机制是构建高可用 MongoDB 应用的基础,合理配置副本集可以实现接近零停机的生产环境。
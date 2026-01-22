# MongoDB 事务 ACID 的实现

## 目录

- [什么是 ACID](#什么是-acid)
- [MongoDB 事务演进历史](#mongodb-事务演进历史)
- [单文档事务](#单文档事务)
- [多文档事务](#多文档事务)
- [ACID 特性实现](#acid-特性实现)
- [事务与副本集](#事务与副本集)
- [事务与分片集群](#事务与分片集群)
- [事务性能与限制](#事务性能与限制)
- [最佳实践](#最佳实践)
- [常见问题 FAQ](#常见问题-faq)

---

## 什么是 ACID

ACID 是数据库事务的四个基本特性,确保数据的完整性和一致性。

### A - Atomicity (原子性)

事务中的所有操作要么全部成功,要么全部失败,不存在部分成功的情况。

```javascript
// 转账示例:原子性确保要么都成功,要么都失败
转账事务 {
  账户A: -100元
  账户B: +100元
}

// 不会出现:
// ❌ A扣款成功,B入账失败(钱丢了)
// ❌ A扣款失败,B入账成功(凭空多钱)
```

### C - Consistency (一致性)

事务执行前后,数据库从一个一致性状态转换到另一个一致性状态,不会违反任何完整性约束。

```javascript
// 一致性示例
转账前:
  账户A余额: 1000元
  账户B余额: 500元
  总额: 1500元

转账后(A转100给B):
  账户A余额: 900元
  账户B余额: 600元
  总额: 1500元 ✓ 一致

// 一致性约束:
// - 总金额不变
// - 余额不能为负
// - 账户必须存在
```

### I - Isolation (隔离性)

并发执行的事务之间互不干扰,一个事务的中间状态对其他事务不可见。

```javascript
// 隔离性示例
事务1: 读取账户A余额 → 1000元
事务2: 读取账户A余额 → 1000元

事务1: 扣款100元 → 900元
事务1: 提交

事务2: 扣款200元 → 800元 (基于最新余额900元)
事务2: 提交

最终余额: 800元 ✓ 正确

// 如果没有隔离性:
// 事务2可能基于旧值1000元计算,导致余额错误
```

### D - Durability (持久性)

事务一旦提交,其结果就是永久性的,即使系统崩溃也不会丢失。

```javascript
// 持久性示例
事务提交 → 写入日志(Journal) → 写入数据文件

系统崩溃
  ↓
重启恢复 → 从日志重放 → 数据仍然存在 ✓
```

---

## MongoDB 事务演进历史

MongoDB 的事务支持经历了渐进式的发展:

```
MongoDB 版本历史:

2009 - MongoDB 1.0
├─ 单文档原子性
└─ 无多文档事务

2018 - MongoDB 4.0
├─ 副本集多文档事务
└─ ACID 支持(单分片)

2019 - MongoDB 4.2
├─ 分片集群多文档事务
├─ 分布式事务支持
└─ 完整 ACID 支持

2020 - MongoDB 4.4
├─ 事务性能优化
└─ 精细化可重试写入

2021 - MongoDB 5.0+
├─ 更好的事务性能
├─ 原生时间序列集合
└─ 快照隔离优化
```

**版本要求:**

```javascript
// 副本集事务: MongoDB 4.0+
// 分片集群事务: MongoDB 4.2+
// 生产环境推荐: MongoDB 5.0+

// 查看当前版本
db.version()
```

---

## 单文档事务

MongoDB 对单个文档的操作天然具有原子性,这是 MongoDB 最早就支持的特性。

### 单文档原子性示例

```javascript
// 示例1: 更新单个文档的多个字段(原子性)
db.accounts.updateOne(
  { _id: "account_123" },
  {
    $inc: { balance: -100 },
    $push: { 
      transactions: {
        amount: -100,
        timestamp: new Date(),
        type: "withdrawal"
      }
    },
    $set: { lastModified: new Date() }
  }
)

// 这三个操作(减少余额、添加交易记录、更新时间)
// 要么全部成功,要么全部失败,不会部分成功
```

```javascript
// 示例2: 嵌入式文档操作
db.orders.updateOne(
  { _id: ObjectId("...") },
  {
    $set: { status: "completed" },
    $inc: { "items.$[].quantity": -1 },  // 减少库存
    $push: { 
      statusHistory: {
        status: "completed",
        timestamp: new Date()
      }
    }
  }
)

// 所有嵌入字段的修改都是原子性的
```

### 单文档原子性的优势

```javascript
// 传统关系数据库需要事务的场景
// SQL:
BEGIN TRANSACTION;
UPDATE accounts SET balance = balance - 100 WHERE id = 123;
INSERT INTO transactions (account_id, amount, type) VALUES (123, -100, 'withdrawal');
UPDATE accounts SET last_modified = NOW() WHERE id = 123;
COMMIT;

// MongoDB: 单个原子操作
db.accounts.updateOne(
  { _id: 123 },
  {
    $inc: { balance: -100 },
    $push: { 
      transactions: { amount: -100, type: "withdrawal" }
    },
    $currentDate: { lastModified: true }
  }
)

// 优势:
// ✅ 更简单,无需显式事务
// ✅ 更高性能
// ✅ 无死锁风险
// ✅ 自动重试
```

### findAndModify 操作

```javascript
// 原子性的查找和修改
const result = db.counters.findAndModify({
  query: { _id: "order_counter" },
  update: { $inc: { sequence: 1 } },
  new: true,  // 返回更新后的文档
  upsert: true  // 不存在则创建
})

const orderNumber = result.sequence
print(`订单号: ${orderNumber}`)

// 多个进程并发执行也不会产生重复序号
// 原子性保证
```

### 乐观并发控制

```javascript
// 使用版本号实现乐观锁
const product = db.products.findOne({ _id: "product_123" })

// 业务逻辑处理
product.price = product.price * 1.1
product.version = product.version + 1

// 更新时检查版本号
const result = db.products.updateOne(
  { 
    _id: "product_123",
    version: product.version - 1  // 检查版本未变
  },
  {
    $set: { price: product.price },
    $inc: { version: 1 }
  }
)

if (result.modifiedCount === 0) {
  print("更新失败,数据已被其他进程修改,请重试")
} else {
  print("更新成功")
}
```

---

## 多文档事务

MongoDB 4.0+ 支持跨多个文档、集合、数据库的 ACID 事务。

### 基本事务 API

#### 核心会话 API

```javascript
// 1. 创建会话
const session = db.getMongo().startSession()

// 2. 开始事务
session.startTransaction({
  readConcern: { level: "snapshot" },
  writeConcern: { w: "majority" },
  readPreference: "primary"
})

try {
  // 3. 在会话中执行操作
  const accountsCollection = session.getDatabase("mydb").accounts
  
  accountsCollection.updateOne(
    { _id: "account_A" },
    { $inc: { balance: -100 } },
    { session: session }
  )
  
  accountsCollection.updateOne(
    { _id: "account_B" },
    { $inc: { balance: 100 } },
    { session: session }
  )
  
  // 4. 提交事务
  session.commitTransaction()
  print("转账成功")
  
} catch (error) {
  // 5. 回滚事务
  session.abortTransaction()
  print("转账失败,已回滚: " + error)
  
} finally {
  // 6. 结束会话
  session.endSession()
}
```

### 完整转账示例

```javascript
// 实现转账功能(多文档事务)
function transferMoney(fromAccount, toAccount, amount) {
  const session = db.getMongo().startSession()
  
  session.startTransaction({
    readConcern: { level: "snapshot" },
    writeConcern: { w: "majority" },
    maxCommitTimeMS: 30000  // 30秒超时
  })
  
  try {
    const accountsDB = session.getDatabase("bank")
    const accounts = accountsDB.accounts
    const transactions = accountsDB.transactions
    
    // 1. 检查源账户余额
    const fromAcc = accounts.findOne(
      { _id: fromAccount },
      { session: session }
    )
    
    if (!fromAcc) {
      throw new Error("源账户不存在")
    }
    
    if (fromAcc.balance < amount) {
      throw new Error("余额不足")
    }
    
    // 2. 检查目标账户存在
    const toAcc = accounts.findOne(
      { _id: toAccount },
      { session: session }
    )
    
    if (!toAcc) {
      throw new Error("目标账户不存在")
    }
    
    // 3. 扣款
    accounts.updateOne(
      { _id: fromAccount },
      { 
        $inc: { balance: -amount },
        $set: { lastModified: new Date() }
      },
      { session: session }
    )
    
    // 4. 入账
    accounts.updateOne(
      { _id: toAccount },
      { 
        $inc: { balance: amount },
        $set: { lastModified: new Date() }
      },
      { session: session }
    )
    
    // 5. 记录交易
    const txnId = new ObjectId()
    transactions.insertOne(
      {
        _id: txnId,
        from: fromAccount,
        to: toAccount,
        amount: amount,
        timestamp: new Date(),
        status: "completed"
      },
      { session: session }
    )
    
    // 6. 提交事务
    session.commitTransaction()
    
    return {
      success: true,
      transactionId: txnId
    }
    
  } catch (error) {
    session.abortTransaction()
    
    return {
      success: false,
      error: error.message
    }
    
  } finally {
    session.endSession()
  }
}

// 使用示例
const result = transferMoney("alice", "bob", 100)
print(JSON.stringify(result))
```

### Node.js 中的事务

```javascript
const { MongoClient } = require('mongodb')

async function transferMoney(client, from, to, amount) {
  const session = client.startSession()
  
  try {
    await session.withTransaction(async () => {
      const accounts = client.db('bank').collection('accounts')
      
      // 1. 检查余额
      const fromAccount = await accounts.findOne(
        { _id: from },
        { session }
      )
      
      if (!fromAccount || fromAccount.balance < amount) {
        throw new Error('余额不足')
      }
      
      // 2. 扣款
      await accounts.updateOne(
        { _id: from },
        { $inc: { balance: -amount } },
        { session }
      )
      
      // 3. 入账
      await accounts.updateOne(
        { _id: to },
        { $inc: { balance: amount } },
        { session }
      )
      
    }, {
      readConcern: { level: 'snapshot' },
      writeConcern: { w: 'majority' },
      readPreference: 'primary'
    })
    
    console.log('转账成功')
    
  } catch (error) {
    console.error('转账失败:', error.message)
    throw error
    
  } finally {
    await session.endSession()
  }
}

// 使用示例
async function main() {
  const client = new MongoClient('mongodb://localhost:27017/bank?replicaSet=rs0')
  
  try {
    await client.connect()
    await transferMoney(client, 'alice', 'bob', 100)
  } finally {
    await client.close()
  }
}

main().catch(console.error)
```

### 事务回调 API

```javascript
// MongoDB 4.2+ 支持的简化 API
async function runTransactionWithRetry(txnFunc, session) {
  while (true) {
    try {
      await txnFunc(session)
      await session.commitTransaction()
      print("事务提交成功")
      break
      
    } catch (error) {
      print("事务遇到错误: " + error)
      
      // 如果是临时错误,重试
      if (error.hasErrorLabel('TransientTransactionError')) {
        print("TransientTransactionError, 重试事务...")
        continue
      } else {
        throw error
      }
    }
  }
}

// 使用示例
async function transferWithRetry(from, to, amount) {
  const session = client.startSession()
  
  try {
    await runTransactionWithRetry(async (session) => {
      const accounts = client.db('bank').collection('accounts')
      
      await accounts.updateOne(
        { _id: from },
        { $inc: { balance: -amount } },
        { session }
      )
      
      await accounts.updateOne(
        { _id: to },
        { $inc: { balance: amount } },
        { session }
      )
      
    }, session)
    
  } finally {
    await session.endSession()
  }
}
```

---

## ACID 特性实现

### Atomicity (原子性) 实现

MongoDB 通过 **两阶段提交协议** 和 **Oplog** 实现原子性。

**实现机制:**

```
1. 准备阶段 (Prepare Phase)
   - 所有操作记录到事务表
   - 检查冲突和约束
   - 获取必要的锁

2. 提交阶段 (Commit Phase)
   - 写入 commitTransaction 操作到 Oplog
   - 应用所有事务操作
   - 释放锁

3. 回滚机制
   - 如果任何阶段失败,回滚所有操作
   - 通过 abortTransaction 操作
```

**Oplog 事务条目:**

```javascript
// 查看事务在 Oplog 中的表示
use local
db.oplog.rs.find({ 
  "o.applyOps": { $exists: true } 
}).limit(1).pretty()

// 示例输出
{
  "ts": Timestamp(1705555200, 1),
  "t": NumberLong(5),
  "op": "c",  // command
  "ns": "admin.$cmd",
  "o": {
    "applyOps": [
      {
        "op": "u",
        "ns": "bank.accounts",
        "o": { "$set": { "balance": 900 } },
        "o2": { "_id": "alice" }
      },
      {
        "op": "u",
        "ns": "bank.accounts",
        "o": { "$set": { "balance": 1100 } },
        "o2": { "_id": "bob" }
      }
    ],
    "lsid": { "id": UUID("...") },  // 逻辑会话ID
    "txnNumber": NumberLong(1),
    "commitTransaction": 1
  }
}
```

**内部实现:**

```javascript
// 事务状态机
事务状态:
  - inProgress: 事务进行中
  - prepared: 准备提交
  - committed: 已提交
  - aborted: 已回滚

状态转换:
  start → inProgress
  inProgress → prepared (prepare)
  prepared → committed (commit)
  inProgress → aborted (abort)
  prepared → aborted (abort)
```

### Consistency (一致性) 实现

MongoDB 通过 **文档验证** 和 **约束检查** 确保一致性。

**文档验证规则:**

```javascript
// 创建带验证的集合
db.createCollection("accounts", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["_id", "balance", "status"],
      properties: {
        _id: {
          bsonType: "string",
          description: "账户ID,必填"
        },
        balance: {
          bsonType: "number",
          minimum: 0,  // 余额不能为负
          description: "账户余额,必须≥0"
        },
        status: {
          enum: ["active", "frozen", "closed"],
          description: "账户状态"
        },
        email: {
          bsonType: "string",
          pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
        }
      }
    }
  },
  validationLevel: "strict",  // 严格模式
  validationAction: "error"   // 违反规则时报错
})

// 测试验证
db.accounts.insertOne({
  _id: "test",
  balance: -100,  // 违反规则
  status: "active"
})
// 错误: Document failed validation
```

**唯一索引约束:**

```javascript
// 创建唯一索引
db.users.createIndex({ email: 1 }, { unique: true })

// 事务中的唯一性检查
session.startTransaction()

try {
  // 插入重复邮箱会导致整个事务失败
  db.users.insertOne(
    { name: "Alice", email: "alice@example.com" },
    { session }
  )
  
  db.users.insertOne(
    { name: "Bob", email: "alice@example.com" },  // 重复!
    { session }
  )
  
  session.commitTransaction()  // 不会执行到这里
  
} catch (error) {
  print("违反唯一性约束: " + error)
  session.abortTransaction()  // 回滚,Alice 也不会被插入
}
```

**引用完整性检查:**

```javascript
// 应用层实现外键检查
function createOrder(customerId, items, session) {
  const customers = session.getDatabase("shop").customers
  const orders = session.getDatabase("shop").orders
  
  // 检查客户是否存在
  const customer = customers.findOne(
    { _id: customerId },
    { session }
  )
  
  if (!customer) {
    throw new Error("客户不存在,无法创建订单")
  }
  
  // 检查商品库存
  for (const item of items) {
    const product = session.getDatabase("shop").products.findOne(
      { _id: item.productId },
      { session }
    )
    
    if (!product || product.stock < item.quantity) {
      throw new Error(`商品 ${item.productId} 库存不足`)
    }
  }
  
  // 创建订单
  orders.insertOne(
    {
      customerId: customerId,
      items: items,
      total: calculateTotal(items),
      createdAt: new Date()
    },
    { session }
  )
}
```

### Isolation (隔离性) 实现

MongoDB 使用 **快照隔离(Snapshot Isolation)** 实现事务隔离。

**快照隔离级别:**

```javascript
// MongoDB 支持的读关注级别
readConcern 级别:
  - local: 读取本地最新数据(可能未提交)
  - majority: 读取已被大多数节点确认的数据
  - snapshot: 事务快照隔离(默认)
  - linearizable: 线性化读取(最强)

// 事务默认使用 snapshot
session.startTransaction({
  readConcern: { level: "snapshot" }
})
```

**快照隔离示例:**

```javascript
// 场景:两个并发事务读取和修改同一账户

// 时间线:
// T0: 账户余额 = 1000

// 事务1开始
const session1 = db.getMongo().startSession()
session1.startTransaction({ readConcern: { level: "snapshot" } })

// T1: 事务1读取余额
const balance1 = session1.getDatabase("bank").accounts.findOne(
  { _id: "alice" },
  { session: session1 }
).balance
// balance1 = 1000 (事务1的快照时间点)

// 事务2开始并完成
const session2 = db.getMongo().startSession()
session2.startTransaction()

session2.getDatabase("bank").accounts.updateOne(
  { _id: "alice" },
  { $inc: { balance: 200 } },  // +200
  { session: session2 }
)

session2.commitTransaction()
session2.endSession()
// T2: 实际余额 = 1200 (但事务1看不到)

// T3: 事务1继续
// 事务1仍然看到余额 = 1000 (快照隔离)
// 事务1尝试更新
session1.getDatabase("bank").accounts.updateOne(
  { _id: "alice" },
  { $inc: { balance: -100 } },  // 想要 -100
  { session: session1 }
)

// T4: 事务1尝试提交
try {
  session1.commitTransaction()
} catch (error) {
  // 错误: WriteConflict
  // 因为事务2已经修改了同一文档
  print("写冲突: " + error)
  session1.abortTransaction()
}

// 正确的结果:
// - 事务1因为写冲突而失败,需要重试
// - 实际余额仍然是 1200 (只有事务2的修改)
// - 事务1需要重新开始,重新读取余额1200,然后-100 = 1100
```

**快照隔离的正确理解:**

```javascript
// 快照隔离保证:
// 1. 事务看到开始时刻的一致性快照
// 2. 事务内的读取不受其他事务影响
// 3. 但提交时会检测写冲突

// 更好的示例: 读写隔离
session1.startTransaction({ readConcern: { level: "snapshot" } })

// 事务1读取多次,看到的数据一致
const read1 = session1.getDatabase("bank").accounts.findOne(
  { _id: "alice" },
  { session: session1 }
)
// balance = 1000

// 此时其他事务修改了数据
// (外部操作,不在事务中)
db.accounts.updateOne({ _id: "alice" }, { $set: { balance: 5000 } })

// 事务1再次读取,仍然看到旧值
const read2 = session1.getDatabase("bank").accounts.findOne(
  { _id: "alice" },
  { session: session1 }
)
// balance = 1000 (快照隔离,看不到外部修改)

session1.commitTransaction()

// 这就是快照隔离:
// 事务内看到的是一致的快照,不受外部干扰
```

**写冲突检测:**

```javascript
// 当两个事务修改同一文档时
session1.startTransaction()
session2.startTransaction()

// 事务1修改文档
session1.getDatabase("db").collection.updateOne(
  { _id: 1 },
  { $set: { value: "A" } },
  { session: session1 }
)

// 事务2尝试修改同一文档
session2.getDatabase("db").collection.updateOne(
  { _id: 1 },
  { $set: { value: "B" } },
  { session: session2 }
)
// 事务2会等待事务1完成

// 事务1提交
session1.commitTransaction()

// 现在事务2继续
// 但会检测到写冲突,抛出 WriteConflict 错误
try {
  session2.commitTransaction()
} catch (error) {
  // WriteConflict: Write conflict during plan execution
  print("写冲突,需要重试事务")
  session2.abortTransaction()
}
```

**隔离级别对比:**

```javascript
// 测试不同隔离级别

// 1. local (可能读到未提交数据)
db.accounts.find({ _id: "alice" }).readConcern("local")

// 2. snapshot (事务快照)
session.startTransaction({ readConcern: { level: "snapshot" } })
// 只能看到事务开始时的快照

// 3. majority (已提交数据)
db.accounts.find({ _id: "alice" }).readConcern("majority")
// 只能看到已被大多数节点确认的数据

// 4. linearizable (线性化)
db.accounts.find({ _id: "alice" }).readConcern("linearizable")
// 保证读取反映所有已提交的写入
// 最强一致性,性能最低
```

### Durability (持久性) 实现

MongoDB 通过 **Journal 日志** 和 **写关注** 确保持久性。

**Journal 持久化:**

```javascript
// 事务提交时的持久化保证
session.startTransaction({
  writeConcern: { 
    w: "majority",  // 大多数节点确认
    j: true         // 等待 Journal 持久化
  }
})

// 提交流程:
// 1. 事务操作写入内存
// 2. commitTransaction 写入 Journal
// 3. Journal fsync 到磁盘
// 4. 返回成功确认
// 5. 后台异步写入数据文件
```

**持久性级别:**

```javascript
// 级别1: 仅内存(最快,可能丢失)
writeConcern: { w: 1, j: false }

// 级别2: 主节点 Journal(单节点持久)
writeConcern: { w: 1, j: true }

// 级别3: 大多数节点内存(防回滚)
writeConcern: { w: "majority", j: false }

// 级别4: 大多数节点 Journal(最强持久性)
writeConcern: { w: "majority", j: true }
```

**崩溃恢复示例:**

```
场景: 事务提交后系统立即崩溃

T0: 开始事务
T1: 更新账户A (仅在内存)
T2: 更新账户B (仅在内存)
T3: commitTransaction 写入 Journal
T4: fsync Journal 到磁盘
T5: 返回客户端"提交成功"
T6: 系统崩溃! (数据文件还未更新)

恢复流程:
T7: 系统重启
T8: 读取 Journal 日志
T9: 重放事务操作
T10: 账户A和B的修改恢复 ✓

结果: 数据不丢失,持久性保证
```

---

## 事务与副本集

### 副本集中的事务执行

```javascript
// 副本集事务要求
// 1. MongoDB 4.0+
// 2. 至少3个节点的副本集
// 3. 使用 WiredTiger 存储引擎

// 事务在副本集中的执行
rs.initiate({
  _id: "myReplicaSet",
  members: [
    { _id: 0, host: "mongodb1:27017" },
    { _id: 1, host: "mongodb2:27017" },
    { _id: 2, host: "mongodb3:27017" }
  ]
})

// 连接到副本集
const client = new MongoClient(
  "mongodb://mongodb1:27017,mongodb2:27017,mongodb3:27017/" +
  "?replicaSet=myReplicaSet"
)

// 事务只能在主节点执行
const session = client.startSession()
session.startTransaction({
  readPreference: "primary",  // 必须是 primary
  readConcern: { level: "snapshot" },
  writeConcern: { w: "majority" }
})
```

### 事务复制流程

```
主节点:
1. 执行事务操作
2. 记录到 Oplog (作为单个条目)
3. 提交事务

从节点:
1. 拉取 Oplog 中的事务条目
2. 原子性地应用整个事务
3. 确认复制完成

保证:
- 事务在所有节点都是原子的
- 不会出现部分应用的情况
- 保持数据一致性
```

**查看事务复制状态:**

```javascript
// 在主节点执行事务
const session = db.getMongo().startSession()
session.startTransaction({ writeConcern: { w: "majority" } })

db.accounts.updateOne(
  { _id: "alice" },
  { $inc: { balance: -100 } },
  { session }
)

db.accounts.updateOne(
  { _id: "bob" },
  { $inc: { balance: 100 } },
  { session }
)

session.commitTransaction()  // 等待大多数节点确认

// 检查复制延迟
rs.printSecondaryReplicationInfo()

// 输出
// source: mongodb2:27017
//   syncedTo: Thu Jan 18 2025 10:00:00 GMT+0800
//   0 secs behind the primary  ✓ 事务已复制
```

### 副本集故障转移期间的事务

```javascript
// 场景: 事务执行期间主节点故障

session.startTransaction()

try {
  // 操作1成功
  db.accounts.updateOne(
    { _id: "alice" },
    { $inc: { balance: -100 } },
    { session }
  )
  
  // 主节点故障,选举新主节点 (10-15秒)
  
  // 操作2失败(no primary)
  db.accounts.updateOne(
    { _id: "bob" },
    { $inc: { balance: 100 } },
    { session }
  )
  // 抛出错误: NotMaster
  
  session.commitTransaction()  // 不会执行
  
} catch (error) {
  // 自动回滚
  session.abortTransaction()
  print("事务失败,已回滚: " + error)
}

// 结果: 两个操作都不会生效,保持一致性 ✓
```

**可重试事务:**

```javascript
// MongoDB 4.2+ 支持自动重试
async function runTransactionWithRetry(txnFunc) {
  const session = client.startSession()
  
  try {
    await session.withTransaction(
      txnFunc,
      {
        readConcern: { level: 'snapshot' },
        writeConcern: { w: 'majority' },
        readPreference: 'primary'
      }
    )
    // withTransaction 会自动处理:
    // 1. TransientTransactionError: 临时错误,自动重试
    // 2. UnknownTransactionCommitResult: 提交结果未知,重试提交
    
  } finally {
    await session.endSession()
  }
}

// 使用示例
await runTransactionWithRetry(async (session) => {
  await db.accounts.updateOne(
    { _id: "alice" },
    { $inc: { balance: -100 } },
    { session }
  )
  
  await db.accounts.updateOne(
    { _id: "bob" },
    { $inc: { balance: 100 } },
    { session }
  )
})
```

---

## 事务与分片集群

### 分片集群事务要求

```javascript
// MongoDB 4.2+ 支持分片集群事务

// 分片集群架构:
Config Servers (配置服务器)
    ↓
Mongos (路由服务器)
    ↓
Shard1, Shard2, Shard3 (分片)

// 事务可以跨多个分片执行
```

### 分布式事务执行

```javascript
// 连接到 mongos
const client = new MongoClient("mongodb://mongos1:27017")

const session = client.startSession()

session.startTransaction({
  readConcern: { level: "snapshot" },
  writeConcern: { w: "majority" },
  readPreference: "primary"
})

try {
  // 操作可能分布在不同分片上
  await client.db("shop").collection("customers").updateOne(
    { _id: "customer_123" },  // 可能在 Shard1
    { $inc: { totalSpent: 99.99 } },
    { session }
  )
  
  await client.db("shop").collection("orders").insertOne(
    {
      customerId: "customer_123",  // 可能在 Shard2
      amount: 99.99,
      items: [...]
    },
    { session }
  )
  
  await session.commitTransaction()
  print("跨分片事务提交成功")
  
} catch (error) {
  await session.abortTransaction()
  print("事务失败: " + error)
  
} finally {
  await session.endSession()
}
```

### 两阶段提交协议

分片集群使用 **两阶段提交(2PC)** 协议确保分布式事务的原子性。

```
阶段1: 准备阶段 (Prepare)
  Coordinator (mongos):
    → 向所有参与分片发送 prepare 请求
  
  Shard1, Shard2, Shard3:
    → 执行事务操作
    → 记录到本地事务表
    → 获取锁
    → 返回 "prepared" 或 "abort"
  
  Coordinator:
    → 收集所有分片的响应
    → 如果全部 prepared,进入阶段2
    → 如果任何一个 abort,回滚所有分片

阶段2: 提交阶段 (Commit)
  Coordinator:
    → 向所有分片发送 commit 请求
  
  Shard1, Shard2, Shard3:
    → 应用事务操作
    → 写入 Oplog
    → 释放锁
    → 返回 "committed"
  
  Coordinator:
    → 确认事务完成
    → 返回客户端
```

**示例流程:**

```javascript
// 跨分片转账(用户A在Shard1,用户B在Shard2)

Mongos 接收事务:
  ↓
准备阶段:
  → Shard1: prepare (扣款用户A)
  ← Shard1: prepared ✓
  → Shard2: prepare (入账用户B)
  ← Shard2: prepared ✓
  ↓
提交阶段:
  → Shard1: commit
  ← Shard1: committed ✓
  → Shard2: commit
  ← Shard2: committed ✓
  ↓
返回客户端: 成功
```

### 分片事务性能考虑

```javascript
// 影响性能的因素:

// 1. 跨分片操作数量
// 差: 操作分散在所有分片
session.startTransaction()
for (let i = 0; i < 100; i++) {
  db.collection.updateOne({ _id: i }, ..., { session })
}
session.commitTransaction()

// 好: 操作集中在少数分片
session.startTransaction()
db.collection.updateOne(
  { customerId: "same_customer" },  // 所有操作同一分片键
  ...,
  { session }
)
session.commitTransaction()

// 2. 事务持续时间
// 建议: < 60 秒(默认超时)

// 3. 数据位置
// 最佳: 单分片事务
// 可接受: 2-3个分片
// 避免: >5个分片

// 4. 网络延迟
// 分片间网络延迟影响2PC性能
```

---

## 事务性能与限制

### 性能特点

```javascript
// 事务 vs 单文档操作性能对比

// 单文档操作(无事务)
const start1 = Date.now()
for (let i = 0; i < 1000; i++) {
  db.test.insertOne({ value: i })
}
const time1 = Date.now() - start1
print(`单文档操作: ${time1}ms, ${1000/time1*1000} ops/sec`)
// 典型: 500ms, ~2000 ops/sec

// 多文档事务
const start2 = Date.now()
for (let i = 0; i < 1000; i++) {
  const session = db.getMongo().startSession()
  session.startTransaction()
  db.test.insertOne({ value: i }, { session })
  session.commitTransaction()
  session.endSession()
}
const time2 = Date.now() - start2
print(`事务操作: ${time2}ms, ${1000/time2*1000} ops/sec`)
// 典型: 3000ms, ~333 ops/sec (慢6倍)
```

### 事务限制

#### 1. 时间限制

```javascript
// 默认事务超时: 60秒
// 可以配置:

// 方法1: 启动时配置
mongod --setParameter transactionLifetimeLimitSeconds=30

// 方法2: 运行时配置
db.adminCommand({ 
  setParameter: 1, 
  transactionLifetimeLimitSeconds: 30 
})

// 方法3: 单个事务配置
session.startTransaction({
  maxCommitTimeMS: 30000  // 30秒
})

// 超时后自动中止
// 错误: TransactionExceededLifetimeLimitSeconds
```

#### 2. 大小限制

```javascript
// 单个事务大小限制: 16MB (Oplog 条目大小)

// 错误示例:
session.startTransaction()

for (let i = 0; i < 100000; i++) {
  db.test.insertOne(
    { 
      data: "x".repeat(1000)  // 每个文档1KB
    },
    { session }
  )
}

session.commitTransaction()
// 错误: TransactionTooLarge

// 解决方案: 分批处理
const batchSize = 1000
for (let i = 0; i < 100000; i += batchSize) {
  const session = db.getMongo().startSession()
  session.startTransaction()
  
  for (let j = 0; j < batchSize && i + j < 100000; j++) {
    db.test.insertOne({ data: "x".repeat(1000) }, { session })
  }
  
  session.commitTransaction()
  session.endSession()
}
```

#### 3. 集合限制

```javascript
// 事务中不支持的操作:

// ❌ 创建集合
session.startTransaction()
db.createCollection("newCollection", { session })
// 错误: Cannot create collection in transaction

// ❌ 创建索引
db.collection.createIndex({ field: 1 }, { session })
// 错误: Cannot create index in transaction

// ❌ DDL 操作
db.collection.drop({ session })
db.renameCollection("old", "new", { session })

// ✅ 允许的操作: CRUD
db.collection.find({}, { session })
db.collection.insertOne({}, { session })
db.collection.updateOne({}, {}, { session })
db.collection.deleteOne({}, { session })
```

#### 4. 写冲突

```javascript
// 两个事务修改同一文档会产生写冲突

// 事务1
const session1 = db.getMongo().startSession()
session1.startTransaction()
db.accounts.updateOne(
  { _id: "alice" },
  { $inc: { balance: 100 } },
  { session: session1 }
)

// 事务2(并发)
const session2 = db.getMongo().startSession()
session2.startTransaction()
db.accounts.updateOne(
  { _id: "alice" },  // 同一文档
  { $inc: { balance: 200 } },
  { session: session2 }
)

// 事务1提交
session1.commitTransaction()  // 成功

// 事务2提交
session2.commitTransaction()
// 错误: WriteConflict

// 需要重试事务2
```

### 监控事务

```javascript
// 1. 查看当前活跃事务
db.currentOp({
  "lsid": { $exists: true },
  "transaction": { $exists: true }
})

// 输出示例
{
  "type": "op",
  "host": "mongodb1:27017",
  "desc": "conn123",
  "active": true,
  "currentOpTime": "2025-01-18T10:00:00.000Z",
  "transaction": {
    "parameters": {
      "txnNumber": NumberLong(1),
      "autocommit": false,
      "readConcern": { "level": "snapshot" }
    },
    "timeActiveMicros": NumberLong(5000000),  // 5秒
    "timeInactiveMicros": NumberLong(0)
  }
}

// 2. 查看事务统计
db.serverStatus().transactions

// 输出
{
  "currentActive": 2,
  "currentInactive": 0,
  "currentOpen": 2,
  "totalAborted": 15,
  "totalCommitted": 1250,
  "totalStarted": 1267
}

// 3. 终止长时间运行的事务
db.killOp(<opid>)
```

---

## 最佳实践

### 1. 优先使用单文档事务

```javascript
// ❌ 不必要的多文档事务
session.startTransaction()
db.users.updateOne(
  { _id: "user123" },
  { $set: { name: "Alice" } },
  { session }
)
db.users.updateOne(
  { _id: "user123" },
  { $set: { email: "alice@example.com" } },
  { session }
)
session.commitTransaction()

// ✅ 使用单文档原子性
db.users.updateOne(
  { _id: "user123" },
  {
    $set: {
      name: "Alice",
      email: "alice@example.com"
    }
  }
)
// 更快,更简单,无死锁风险
```

### 2. 合理设计文档结构

```javascript
// ❌ 需要事务的设计
// orders 集合
{ _id: 1, customerId: 100, total: 0 }

// order_items 集合
{ orderId: 1, product: "A", price: 10 }
{ orderId: 1, product: "B", price: 20 }

// 需要事务保证一致性
session.startTransaction()
db.orders.insertOne({ _id: 1, customerId: 100, total: 30 })
db.order_items.insertMany([
  { orderId: 1, product: "A", price: 10 },
  { orderId: 1, product: "B", price: 20 }
])
session.commitTransaction()

// ✅ 嵌入式文档设计
db.orders.insertOne({
  _id: 1,
  customerId: 100,
  total: 30,
  items: [
    { product: "A", price: 10 },
    { product: "B", price: 20 }
  ]
})
// 单文档原子性,无需事务
```

### 3. 保持事务简短

```javascript
// ❌ 长时间事务
session.startTransaction()

// 外部API调用(可能很慢)
const exchangeRate = await fetchExchangeRate()

// 复杂计算
const result = await performComplexCalculation()

// 数据库操作
db.accounts.updateOne({ _id: "alice" }, ...)

session.commitTransaction()

// ✅ 先准备数据,事务中仅做数据库操作
const exchangeRate = await fetchExchangeRate()
const result = await performComplexCalculation()

session.startTransaction()
db.accounts.updateOne({ _id: "alice" }, ...)
session.commitTransaction()
```

### 4. 实现重试逻辑

```javascript
// 完整的事务重试机制
async function runTransactionWithRetry(txnFunc, maxRetries = 3) {
  let retryCount = 0
  
  while (retryCount < maxRetries) {
    const session = client.startSession()
    
    try {
      await session.withTransaction(async () => {
        await txnFunc(session)
      }, {
        readConcern: { level: 'snapshot' },
        writeConcern: { w: 'majority', wtimeout: 5000 },
        readPreference: 'primary'
      })
      
      return  // 成功,退出
      
    } catch (error) {
      retryCount++
      
      if (error.hasErrorLabel('TransientTransactionError')) {
        console.log(`临时错误,重试 ${retryCount}/${maxRetries}`)
        await new Promise(resolve => setTimeout(resolve, 1000))
        continue
      }
      
      if (error.hasErrorLabel('UnknownTransactionCommitResult')) {
        console.log(`提交结果未知,重试 ${retryCount}/${maxRetries}`)
        await new Promise(resolve => setTimeout(resolve, 1000))
        continue
      }
      
      throw error  // 不可重试的错误
      
    } finally {
      await session.endSession()
    }
  }
  
  throw new Error(`事务失败,已达最大重试次数 ${maxRetries}`)
}

// 使用
await runTransactionWithRetry(async (session) => {
  await db.accounts.updateOne(
    { _id: "alice" },
    { $inc: { balance: -100 } },
    { session }
  )
  
  await db.accounts.updateOne(
    { _id: "bob" },
    { $inc: { balance: 100 } },
    { session }
  )
})
```

### 5. 使用适当的写关注

```javascript
// 根据业务需求选择写关注

// 场景1: 关键金融交易
session.startTransaction({
  writeConcern: { 
    w: "majority",  // 大多数节点
    j: true,        // Journal 持久化
    wtimeout: 5000
  }
})

// 场景2: 一般业务
session.startTransaction({
  writeConcern: { 
    w: "majority",
    wtimeout: 3000
  }
})

// 场景3: 性能优先(可容忍少量数据丢失)
session.startTransaction({
  writeConcern: { 
    w: 1,
    wtimeout: 1000
  }
})
```

### 6. 监控和告警

```javascript
// 设置事务监控

// 1. 监控长时间运行的事务
db.currentOp({
  "transaction.timeActiveMicros": { $gt: 30000000 }  // >30秒
}).forEach(op => {
  print(`警告: 事务运行时间过长 ${op.secs_running}秒`)
  print(`  连接: ${op.desc}`)
  print(`  操作: ${op.op}`)
})

// 2. 监控事务中止率
const stats = db.serverStatus().transactions
const abortRate = stats.totalAborted / stats.totalStarted * 100
print(`事务中止率: ${abortRate.toFixed(2)}%`)

if (abortRate > 10) {
  print("警告: 事务中止率过高,检查写冲突")
}

// 3. 设置告警阈值
function checkTransactionHealth() {
  const stats = db.serverStatus().transactions
  
  if (stats.currentActive > 100) {
    alert("活跃事务数过多: " + stats.currentActive)
  }
  
  if (stats.currentOpen > 200) {
    alert("打开事务数过多: " + stats.currentOpen)
  }
}
```

---

## 常见问题 FAQ

### 1. MongoDB 的事务性能与关系型数据库相比如何?

**性能对比:**

```javascript
// MongoDB 事务性能特点

优势:
✅ 单文档操作速度快(无需事务开销)
✅ 文档模型减少 JOIN,减少事务范围
✅ 水平扩展能力强

劣势:
❌ 多文档事务比单文档慢 3-10 倍
❌ 分片事务有 2PC 开销
❌ 快照隔离的内存开销

// 实际测试(3节点副本集)

// 单文档插入
单文档 w:1:      ~5000 ops/sec
单文档 w:majority: ~2000 ops/sec

// 多文档事务
事务(2个操作):   ~500 ops/sec
事务(10个操作):  ~200 ops/sec

// 关系数据库(MySQL)
单表插入:        ~3000 ops/sec
事务(2个表):     ~1000 ops/sec
事务(10个表):    ~400 ops/sec
```

**性能建议:**

```javascript
// 1. 优先使用单文档原子性
// 通过文档设计避免事务

// 2. 批量操作优于多次事务
// ❌ 慢
for (let i = 0; i < 1000; i++) {
  session.startTransaction()
  db.collection.insertOne({ value: i }, { session })
  session.commitTransaction()
}

// ✅ 快
session.startTransaction()
db.collection.insertMany(
  Array.from({ length: 1000 }, (_, i) => ({ value: i })),
  { session }
)
session.commitTransaction()

// 3. 减少事务持续时间
// 事务外完成计算和准备工作

// 4. 使用适当的写关注
// w:1 比 w:majority 快 2-3 倍
```

**何时使用 MongoDB 事务:**

```
✅ 适用场景:
- 跨集合的数据一致性要求
- 金融交易、库存管理
- 多步骤操作的原子性
- 审计追踪

❌ 不适用场景:
- 高吞吐量的简单操作
- 可以通过文档设计解决的场景
- 对延迟极度敏感的应用
```

### 2. 为什么有时候事务会失败并要求重试?

**常见失败原因:**

**1. 写冲突(WriteConflict)**

```javascript
// 原因: 两个事务修改同一文档

// 事务A
session1.startTransaction()
db.accounts.updateOne(
  { _id: "alice" },
  { $inc: { balance: 100 } },
  { session: session1 }
)
// ... 还未提交

// 事务B(并发)
session2.startTransaction()
db.accounts.updateOne(
  { _id: "alice" },  // 同一文档!
  { $inc: { balance: 200 } },
  { session: session2 }
)

// 事务A提交
session1.commitTransaction()

// 事务B提交失败
session2.commitTransaction()
// 错误: WriteConflict

// 解决: 重试事务B
```

**2. 临时网络错误**

```javascript
// TransientTransactionError

// 原因:
// - 网络抖动
// - 副本集选举
// - 临时的资源不可用

// 示例
session.startTransaction()
db.collection.insertOne({ data: "test" }, { session })

// 此时发生网络问题或主节点切换
session.commitTransaction()
// 错误: TransientTransactionError

// 这类错误应该自动重试
if (error.hasErrorLabel('TransientTransactionError')) {
  // 重试整个事务
}
```

**3. 事务超时**

```javascript
// TransactionExceededLifetimeLimitSeconds

session.startTransaction()

// 执行耗时操作
for (let i = 0; i < 1000000; i++) {
  db.collection.insertOne({ value: i }, { session })
}
// 超过 60 秒

session.commitTransaction()
// 错误: TransactionExceededLifetimeLimitSeconds

// 解决: 分批处理,减小事务范围
```

**4. 未知的提交结果**

```javascript
// UnknownTransactionCommitResult

// 原因:
// - 提交请求发送后,网络中断
// - 不确定提交是否成功

session.commitTransaction()
// 错误: UnknownTransactionCommitResult

// 应该重试提交(幂等操作)
if (error.hasErrorLabel('UnknownTransactionCommitResult')) {
  // 重试 commitTransaction
  session.commitTransaction()
}
```

**完整的错误处理:**

```javascript
async function robustTransaction(txnFunc) {
  const maxRetries = 3
  let attempt = 0
  
  while (attempt < maxRetries) {
    const session = client.startSession()
    
    try {
      await session.withTransaction(
        async () => await txnFunc(session),
        {
          readConcern: { level: 'snapshot' },
          writeConcern: { w: 'majority' },
          readPreference: 'primary'
        }
      )
      
      return  // 成功
      
    } catch (error) {
      attempt++
      
      // 可重试的错误
      if (error.hasErrorLabel('TransientTransactionError') ||
          error.hasErrorLabel('UnknownTransactionCommitResult')) {
        
        console.log(`重试 ${attempt}/${maxRetries}: ${error.message}`)
        await new Promise(r => setTimeout(r, Math.min(1000 * attempt, 5000)))
        continue
      }
      
      // 不可重试的错误(如业务逻辑错误)
      throw error
      
    } finally {
      await session.endSession()
    }
  }
  
  throw new Error(`事务失败,已重试 ${maxRetries} 次`)
}
```

### 3. 事务中能否创建集合或索引?

**简短答案: 不能直接在事务中创建集合或索引。**

```javascript
// ❌ 不支持的操作
session.startTransaction()

// 创建集合
db.createCollection("newCollection", { session })
// 错误: Cannot create collection in transaction

// 创建索引
db.collection.createIndex({ field: 1 }, { session })
// 错误: Cannot create index in transaction

// Drop 集合
db.collection.drop({ session })
// 错误: Cannot drop collection in transaction

session.commitTransaction()
```

**原因:**

```
DDL 操作的特点:
1. 需要修改元数据
2. 影响整个集合/索引结构
3. 可能耗时很长
4. 需要获取集合级锁

与事务的冲突:
- 事务应该快速完成
- DDL 操作可能阻塞其他事务
- 回滚 DDL 操作复杂且耗时
```

**解决方案:**

**方案1: 事务外创建集合**

```javascript
// 先创建集合
db.createCollection("orders")
db.orders.createIndex({ customerId: 1 })

// 然后在事务中使用
session.startTransaction()
db.orders.insertOne({ customerId: 123 }, { session })
session.commitTransaction()
```

**方案2: 隐式集合创建**

```javascript
// MongoDB 会在首次写入时自动创建集合
session.startTransaction()

// 如果 "newCollection" 不存在,会自动创建
db.newCollection.insertOne({ data: "test" }, { session })

session.commitTransaction()
// ✓ 成功

// 注意: 只支持默认选项的集合
// 如果需要特殊选项(如验证、上限集合),必须事先创建
```

**方案3: 预先准备索引**

```javascript
// 在应用启动时创建所需的索引
async function ensureIndexes() {
  await db.users.createIndex({ email: 1 }, { unique: true })
  await db.orders.createIndex({ customerId: 1, createdAt: -1 })
  await db.products.createIndex({ category: 1, price: 1 })
}

// 启动时执行一次
await ensureIndexes()

// 之后事务中只做 CRUD
session.startTransaction()
db.users.insertOne({ email: "user@example.com" }, { session })
session.commitTransaction()
```

**支持的操作:**

```javascript
// ✅ 事务中支持的操作

session.startTransaction()

// CRUD 操作
db.collection.find({}, { session })
db.collection.insertOne({}, { session })
db.collection.insertMany([], { session })
db.collection.updateOne({}, {}, { session })
db.collection.updateMany({}, {}, { session })
db.collection.deleteOne({}, { session })
db.collection.deleteMany({}, { session })
db.collection.findOneAndUpdate({}, {}, { session })
db.collection.findOneAndDelete({}, { session })
db.collection.findOneAndReplace({}, {}, { session })

// 聚合操作
db.collection.aggregate([], { session })

// 计数
db.collection.countDocuments({}, { session })

session.commitTransaction()
```

### 4. 如何在分片集群中使用事务?

**基本要求:**

```javascript
// 1. MongoDB 4.2+
// 2. 分片集群配置正确
// 3. 连接到 mongos

// 分片集群架构
Config Servers (副本集)
     ↓
Mongos (路由)
     ↓
Shard1, Shard2, Shard3 (每个都是副本集)
```

**连接配置:**

```javascript
// 连接到 mongos(不是直接连接分片)
const client = new MongoClient(
  'mongodb://mongos1:27017,mongos2:27017/?retryWrites=true'
)

await client.connect()

// 创建会话
const session = client.startSession()

// 开始事务
session.startTransaction({
  readConcern: { level: 'snapshot' },
  writeConcern: { w: 'majority' },
  readPreference: 'primary'
})
```

**跨分片事务示例:**

```javascript
// 假设:
// - customers 集合分片键: customerId
// - orders 集合分片键: customerId
// - products 集合分片键: productId

async function createOrder(customerId, productId, quantity) {
  const session = client.startSession()
  
  try {
    await session.withTransaction(async () => {
      const db = client.db('shop')
      
      // 操作1: 检查库存(可能在 Shard1)
      const product = await db.collection('products').findOne(
        { _id: productId },
        { session }
      )
      
      if (!product || product.stock < quantity) {
        throw new Error('库存不足')
      }
      
      // 操作2: 减少库存(Shard1)
      await db.collection('products').updateOne(
        { _id: productId },
        { $inc: { stock: -quantity } },
        { session }
      )
      
      // 操作3: 创建订单(可能在 Shard2)
      await db.collection('orders').insertOne(
        {
          customerId: customerId,
          productId: productId,
          quantity: quantity,
          total: product.price * quantity,
          createdAt: new Date()
        },
        { session }
      )
      
      // 操作4: 更新客户统计(Shard2)
      await db.collection('customers').updateOne(
        { _id: customerId },
        { 
          $inc: { 
            totalOrders: 1,
            totalSpent: product.price * quantity
          }
        },
        { session }
      )
      
      // 两阶段提交确保所有分片操作原子性
    })
    
    console.log('订单创建成功')
    
  } catch (error) {
    console.error('订单创建失败:', error.message)
    throw error
    
  } finally {
    await session.endSession()
  }
}

// 调用
await createOrder('customer123', 'product456', 2)
```

**性能优化:**

```javascript
// 1. 尽量让事务操作在同一分片
// ✅ 好: 使用相同分片键
await db.orders.insertOne(
  { customerId: 'alice', ... },  // 分片键
  { session }
)
await db.orderItems.insertMany(
  [
    { customerId: 'alice', ... },  // 相同分片键
    { customerId: 'alice', ... }
  ],
  { session }
)
// 可能在同一分片,性能更好

// ❌ 差: 分散在多个分片
await db.orders.insertOne(
  { customerId: 'alice' },
  { session }
)
await db.products.updateOne(
  { productId: 'product123' },  // 不同分片键
  { $inc: { sold: 1 } },
  { session }
)
// 跨分片,需要 2PC,性能较差

// 2. 减少跨分片操作数量
// 3. 使用合适的分片策略
```

**监控分片事务:**

```javascript
// 连接到 mongos
use admin

// 查看分片事务
db.currentOp({
  "transaction": { $exists: true }
})

// 查看协调器事务
db.getSiblingDB("config").transactions.find()

// 输出
{
  "_id": {
    "lsid": { "id": UUID("...") },
    "txnNumber": NumberLong(1)
  },
  "participants": [
    {
      "shardId": "shard1",
      "coordinator": false
    },
    {
      "shardId": "shard2",
      "coordinator": true
    }
  ],
  "commitTimestamp": Timestamp(1705555200, 1)
}
```

### 5. 单文档操作真的不需要事务吗?

**简短答案: 是的,单文档操作天然具有原子性。**

**单文档原子性保证:**

```javascript
// 复杂的单文档更新也是原子的
db.orders.updateOne(
  { _id: ObjectId("...") },
  {
    // 所有这些操作要么全部成功,要么全部失败
    $set: { status: "completed", completedAt: new Date() },
    $inc: { "items.$[].quantity": -1 },  // 减少所有商品数量
    $push: { 
      statusHistory: {
        status: "completed",
        timestamp: new Date(),
        user: "admin"
      }
    },
    $unset: { tempData: "" },
    $mul: { priority: 0.5 }
  },
  {
    arrayFilters: [{ "item.shipped": false }]
  }
)

// 原子性保证:
// ✓ 其他客户端看不到中间状态
// ✓ 要么全部更新,要么都不更新
// ✓ 不会被其他操作打断
```

**与事务的对比:**

```javascript
// 场景: 更新订单状态和添加历史记录

// 方法1: 单文档操作(推荐)
db.orders.updateOne(
  { _id: orderId },
  {
    $set: { status: "shipped" },
    $push: { 
      statusHistory: { 
        status: "shipped", 
        timestamp: new Date() 
      }
    }
  }
)
// 优势:
// ✓ 更快(无事务开销)
// ✓ 更简单(无需会话管理)
// ✓ 无死锁风险
// ✓ 自动重试

// 方法2: 多文档事务(不必要)
session.startTransaction()
db.orders.updateOne(
  { _id: orderId },
  { $set: { status: "shipped" } },
  { session }
)
db.orderHistory.insertOne(
  { 
    orderId: orderId,
    status: "shipped",
    timestamp: new Date()
  },
  { session }
)
session.commitTransaction()
// 劣势:
// ✗ 更慢(3-10倍)
// ✗ 更复杂
// ✗ 可能死锁
// ✗ 需要处理错误重试
```

**何时必须使用事务:**

```javascript
// ✅ 场景1: 跨文档的数据一致性
// 转账: 必须原子性地修改两个账户
session.startTransaction()
db.accounts.updateOne(
  { _id: "alice" },
  { $inc: { balance: -100 } },
  { session }
)
db.accounts.updateOne(
  { _id: "bob" },
  { $inc: { balance: 100 } },
  { session }
)
session.commitTransaction()

// ✅ 场景2: 跨集合的引用完整性
// 删除用户时,必须同时删除所有相关数据
session.startTransaction()
db.users.deleteOne({ _id: userId }, { session })
db.posts.deleteMany({ userId: userId }, { session })
db.comments.deleteMany({ userId: userId }, { session })
session.commitTransaction()

// ✅ 场景3: 读取后写入的一致性
session.startTransaction()
const product = db.products.findOne(
  { _id: productId },
  { session }
)
if (product.stock > 0) {
  db.products.updateOne(
    { _id: productId },
    { $inc: { stock: -1 } },
    { session }
  )
  db.orders.insertOne({ productId, ... }, { session })
}
session.commitTransaction()
```

**设计建议:**

```javascript
// 通过文档设计避免事务

// ❌ 需要事务的设计
// 用户集合
{ _id: "user123", name: "Alice" }

// 地址集合
{ userId: "user123", street: "...", city: "..." }

// 更新时需要事务
session.startTransaction()
db.users.updateOne({ _id: "user123" }, { $set: { name: "Alice Wang" } })
db.addresses.updateMany({ userId: "user123" }, { $set: { userName: "Alice Wang" } })
session.commitTransaction()

// ✅ 嵌入式设计(无需事务)
{
  _id: "user123",
  name: "Alice",
  addresses: [
    { type: "home", street: "...", city: "..." },
    { type: "work", street: "...", city: "..." }
  ]
}

// 单文档更新,自动原子性
db.users.updateOne(
  { _id: "user123" },
  { $set: { name: "Alice Wang" } }
)
```

---

## 总结

MongoDB 的 ACID 事务支持经历了从单文档原子性到完整分布式事务的演进:

**核心特性:**
- ✅ **Atomicity**: 通过两阶段提交和 Oplog 实现
- ✅ **Consistency**: 文档验证和约束检查保证
- ✅ **Isolation**: 快照隔离提供事务间隔离
- ✅ **Durability**: Journal 和写关注确保持久性

**版本支持:**
- MongoDB 4.0+: 副本集多文档事务
- MongoDB 4.2+: 分片集群分布式事务
- MongoDB 5.0+: 性能优化和增强

**最佳实践:**
- 优先使用单文档原子性
- 通过文档设计减少事务需求
- 保持事务简短(< 60秒)
- 实现重试逻辑处理临时错误
- 使用适当的读写关注级别
- 监控事务性能和中止率

**使用建议:**
- 单文档操作无需显式事务
- 跨文档一致性需求使用事务
- 分片集群注意 2PC 性能开销
- 合理设计分片键减少跨分片事务

理解 MongoDB 事务机制有助于在保证数据一致性的同时,获得最佳性能表现。
# MongoDB 基本概念

## 目录

- [什么是 MongoDB](#什么是-mongodb)
- [核心概念](#核心概念)
- [数据模型](#数据模型)
- [基本操作](#基本操作)
- [索引机制](#索引机制)
- [聚合框架](#聚合框架)
- [事务支持](#事务支持)
- [复制与分片](#复制与分片)
- [常见问题 FAQ](#常见问题-faq)

---

## 什么是 MongoDB

MongoDB 是一个基于分布式文件存储的开源 NoSQL 数据库系统,由 C++ 语言编写。它将数据存储为文档(Document),使用类似 JSON 的 BSON 格式,提供了高性能、高可用性和易扩展性的数据库解决方案。

### 主要特点

- **文档型数据库**: 使用 BSON(Binary JSON)格式存储数据
- **灵活的模式**: 无需预定义表结构,支持动态模式
- **高性能**: 支持嵌入式数据模型减少 I/O 操作
- **高可用性**: 通过副本集提供自动故障转移
- **水平扩展**: 支持分片机制实现海量数据存储
- **丰富的查询语言**: 支持复杂的查询、聚合和地理空间查询

---

## 核心概念

### 1. Database (数据库)

数据库是集合的物理容器,每个数据库都有自己的文件系统。一个 MongoDB 服务器可以有多个数据库。

```javascript
// 查看所有数据库
show dbs

// 切换/创建数据库
use myDatabase

// 查看当前数据库
db
```

### 2. Collection (集合)

集合是 MongoDB 文档的分组,类似于关系型数据库中的表。集合存在于数据库中,不强制要求固定的模式。

```javascript
// 创建集合
db.createCollection("users")

// 查看所有集合
show collections

// 删除集合
db.users.drop()
```

**特点:**
- 集合名称不能是空字符串
- 不能包含 `\0` (空字符)
- 不能以 `system.` 开头(系统保留)
- 集合名称区分大小写

### 3. Document (文档)

文档是 MongoDB 中数据的基本单元,类似于关系型数据库中的行。文档是一组键值对的集合,采用 BSON 格式。

```javascript
{
  "_id": ObjectId("507f1f77bcf86cd799439011"),
  "name": "张三",
  "age": 28,
  "email": "zhangsan@example.com",
  "address": {
    "city": "北京",
    "district": "朝阳区"
  },
  "hobbies": ["阅读", "游泳", "编程"]
}
```

**文档特性:**
- 键是字符串类型
- 键值对是有序的
- 文档的键是区分大小写的
- 文档不能有重复的键
- 最大文档大小为 16MB

### 4. Field (字段)

字段是文档中的键值对,类似于关系型数据库中的列。

### 5. ObjectId

`_id` 是文档的主键,MongoDB 默认会为每个文档自动生成一个唯一的 `_id` 字段。

**ObjectId 结构 (12 字节):**
- 4 字节:时间戳(秒级)
- 5 字节:随机值
- 3 字节:递增计数器

```javascript
// 生成新的 ObjectId
ObjectId()

// 获取时间戳
ObjectId("507f1f77bcf86cd799439011").getTimestamp()
```

---

## 数据模型

### BSON 数据类型

MongoDB 使用 BSON(Binary JSON)格式存储数据,支持以下主要类型:

| 类型 | 说明 | 示例 |
|------|------|------|
| String | 字符串 | `"Hello World"` |
| Integer | 整数(32位/64位) | `123` |
| Double | 浮点数 | `3.14` |
| Boolean | 布尔值 | `true` / `false` |
| Array | 数组 | `["a", "b", "c"]` |
| Object | 嵌入式文档 | `{name: "test"}` |
| Null | 空值 | `null` |
| Date | 日期时间 | `new Date()` |
| ObjectId | 对象ID | `ObjectId()` |
| Binary Data | 二进制数据 | 图片、文件等 |
| Regular Expression | 正则表达式 | `/pattern/` |
| Code | JavaScript 代码 | `function() {}` |

### 数据建模策略

#### 1. 嵌入式文档 (Embedded Documents)

适用于一对一或一对少的关系,将相关数据嵌入到同一个文档中。

```javascript
// 用户及其地址信息
{
  "_id": 1,
  "name": "李四",
  "email": "lisi@example.com",
  "addresses": [
    {
      "type": "home",
      "city": "上海",
      "street": "南京路123号"
    },
    {
      "type": "work",
      "city": "上海",
      "street": "淮海路456号"
    }
  ]
}
```

**优点:**
- 单次查询获取所有相关数据
- 原子性操作,更新一致性好
- 更好的读取性能

**缺点:**
- 文档大小限制(16MB)
- 数据冗余
- 不适合频繁独立访问嵌入数据的场景

#### 2. 引用关系 (References)

适用于一对多或多对多的关系,通过存储关联文档的 ID 来建立关系。

```javascript
// 用户集合
{
  "_id": ObjectId("user_id_1"),
  "name": "王五",
  "email": "wangwu@example.com"
}

// 订单集合
{
  "_id": ObjectId("order_id_1"),
  "user_id": ObjectId("user_id_1"),  // 引用用户ID
  "total": 299.99,
  "items": [...]
}
```

**优点:**
- 避免数据冗余
- 文档大小更小
- 更灵活的数据关系

**缺点:**
- 需要多次查询
- 应用层需要处理关联

---

## 基本操作

### 创建操作 (Create)

#### insertOne - 插入单个文档

```javascript
db.users.insertOne({
  name: "赵六",
  age: 25,
  email: "zhaoliu@example.com",
  createdAt: new Date()
})

// 返回结果
{
  "acknowledged": true,
  "insertedId": ObjectId("...")
}
```

#### insertMany - 插入多个文档

```javascript
db.users.insertMany([
  {
    name: "孙七",
    age: 30,
    email: "sunqi@example.com"
  },
  {
    name: "周八",
    age: 28,
    email: "zhouba@example.com"
  }
])

// 返回结果
{
  "acknowledged": true,
  "insertedIds": [
    ObjectId("..."),
    ObjectId("...")
  ]
}
```

### 查询操作 (Read)

#### find - 查询文档

```javascript
// 查询所有文档
db.users.find()

// 条件查询
db.users.find({ age: 28 })

// 多条件查询(AND)
db.users.find({ 
  age: { $gte: 25 },
  city: "北京"
})

// OR 查询
db.users.find({
  $or: [
    { age: { $lt: 25 } },
    { age: { $gt: 30 } }
  ]
})

// 字段投影(只返回指定字段)
db.users.find(
  { age: { $gte: 25 } },
  { name: 1, email: 1, _id: 0 }
)

// 排序
db.users.find().sort({ age: -1 })  // -1 降序, 1 升序

// 分页
db.users.find()
  .skip(10)   // 跳过前10条
  .limit(5)   // 返回5条

// 统计数量
db.users.countDocuments({ age: { $gte: 25 } })
```

#### 常用查询操作符

| 操作符 | 说明 | 示例 |
|--------|------|------|
| `$eq` | 等于 | `{ age: { $eq: 25 } }` |
| `$ne` | 不等于 | `{ age: { $ne: 25 } }` |
| `$gt` | 大于 | `{ age: { $gt: 25 } }` |
| `$gte` | 大于等于 | `{ age: { $gte: 25 } }` |
| `$lt` | 小于 | `{ age: { $lt: 25 } }` |
| `$lte` | 小于等于 | `{ age: { $lte: 25 } }` |
| `$in` | 在数组中 | `{ age: { $in: [25, 30] } }` |
| `$nin` | 不在数组中 | `{ age: { $nin: [25, 30] } }` |
| `$regex` | 正则匹配 | `{ name: { $regex: /^张/ } }` |
| `$exists` | 字段是否存在 | `{ email: { $exists: true } }` |

#### 查询嵌套文档和数组

```javascript
// 嵌套文档查询
db.users.find({ "address.city": "北京" })

// 数组查询
db.users.find({ hobbies: "编程" })  // 数组包含"编程"

// 数组所有元素匹配
db.users.find({ 
  hobbies: { $all: ["编程", "阅读"] }
})

// 数组大小
db.users.find({ 
  hobbies: { $size: 3 }
})

// 数组元素查询
db.products.find({
  "reviews.rating": { $gte: 4 }
})
```

### 更新操作 (Update)

#### updateOne - 更新单个文档

```javascript
db.users.updateOne(
  { name: "张三" },              // 查询条件
  { 
    $set: { age: 29 },           // 设置字段值
    $currentDate: { 
      lastModified: true 
    }
  }
)
```

#### updateMany - 更新多个文档

```javascript
db.users.updateMany(
  { age: { $lt: 25 } },
  { 
    $inc: { age: 1 }             // 字段值增加1
  }
)
```

#### replaceOne - 替换文档

```javascript
db.users.replaceOne(
  { name: "李四" },
  {
    name: "李四",
    age: 31,
    email: "lisi_new@example.com"
  }
)
```

#### 常用更新操作符

| 操作符 | 说明 | 示例 |
|--------|------|------|
| `$set` | 设置字段值 | `{ $set: { age: 30 } }` |
| `$unset` | 删除字段 | `{ $unset: { age: "" } }` |
| `$inc` | 增加数值 | `{ $inc: { age: 1 } }` |
| `$mul` | 乘以数值 | `{ $mul: { price: 1.1 } }` |
| `$rename` | 重命名字段 | `{ $rename: { "name": "fullName" } }` |
| `$min` | 设置最小值 | `{ $min: { score: 50 } }` |
| `$max` | 设置最大值 | `{ $max: { score: 100 } }` |
| `$currentDate` | 设置当前日期 | `{ $currentDate: { lastModified: true } }` |

#### 数组更新操作符

```javascript
// 添加元素到数组
db.users.updateOne(
  { name: "张三" },
  { $push: { hobbies: "游泳" } }
)

// 添加多个元素
db.users.updateOne(
  { name: "张三" },
  { $push: { 
      hobbies: { 
        $each: ["游泳", "跑步"] 
      }
    }
  }
)

// 删除数组元素
db.users.updateOne(
  { name: "张三" },
  { $pull: { hobbies: "游泳" } }
)

// 删除第一个或最后一个元素
db.users.updateOne(
  { name: "张三" },
  { $pop: { hobbies: 1 } }  // 1删除最后一个, -1删除第一个
)

// 添加唯一元素
db.users.updateOne(
  { name: "张三" },
  { $addToSet: { hobbies: "编程" } }
)
```

### 删除操作 (Delete)

#### deleteOne - 删除单个文档

```javascript
db.users.deleteOne({ name: "张三" })
```

#### deleteMany - 删除多个文档

```javascript
db.users.deleteMany({ age: { $lt: 18 } })

// 删除所有文档
db.users.deleteMany({})
```

---

## 索引机制

索引能够显著提高查询性能,MongoDB 支持多种类型的索引。

### 索引类型

#### 1. 单字段索引

```javascript
// 创建升序索引
db.users.createIndex({ age: 1 })

// 创建降序索引
db.users.createIndex({ age: -1 })
```

#### 2. 复合索引

```javascript
// 创建复合索引
db.users.createIndex({ age: 1, name: 1 })

// 查询会使用该索引
db.users.find({ age: 25, name: "张三" })
db.users.find({ age: 25 })  // 前缀也可以使用

// 不会使用该索引
db.users.find({ name: "张三" })  // 没有包含索引前缀
```

#### 3. 多键索引 (Multikey Index)

对数组字段创建索引,MongoDB 会为数组中的每个元素创建索引项。

```javascript
db.users.createIndex({ hobbies: 1 })

// 可以高效查询
db.users.find({ hobbies: "编程" })
```

#### 4. 文本索引

用于全文搜索。

```javascript
// 创建文本索引
db.articles.createIndex({ content: "text" })

// 文本搜索
db.articles.find({ $text: { $search: "MongoDB 教程" } })
```

#### 5. 地理空间索引

```javascript
// 2dsphere 索引(用于球面几何)
db.places.createIndex({ location: "2dsphere" })

// 查询附近的地点
db.places.find({
  location: {
    $near: {
      $geometry: {
        type: "Point",
        coordinates: [116.4074, 39.9042]  // 经度, 纬度
      },
      $maxDistance: 5000  // 5000米内
    }
  }
})
```

#### 6. 唯一索引

```javascript
// 创建唯一索引
db.users.createIndex({ email: 1 }, { unique: true })

// 尝试插入重复email会失败
db.users.insertOne({ email: "test@example.com" })  // OK
db.users.insertOne({ email: "test@example.com" })  // Error
```

### 索引管理

```javascript
// 查看集合的所有索引
db.users.getIndexes()

// 查看查询计划
db.users.find({ age: 25 }).explain("executionStats")

// 删除索引
db.users.dropIndex("age_1")

// 删除所有索引(除了_id索引)
db.users.dropIndexes()

// 创建后台索引(不阻塞数据库操作)
db.users.createIndex({ age: 1 }, { background: true })

// 创建带TTL的索引(文档过期自动删除)
db.sessions.createIndex(
  { createdAt: 1 }, 
  { expireAfterSeconds: 3600 }  // 1小时后过期
)
```

### 索引最佳实践

1. **为常用查询字段创建索引**: 分析查询模式,为经常查询的字段创建索引
2. **使用复合索引**: 对于多字段查询,创建复合索引而非多个单字段索引
3. **索引顺序很重要**: 复合索引中字段的顺序影响性能
4. **避免过多索引**: 索引会占用空间并影响写入性能
5. **使用 explain() 分析**: 定期检查查询计划,确保索引被正确使用

---

## 聚合框架

聚合框架用于数据分析和处理,类似于 SQL 的 GROUP BY。

### 聚合管道

聚合操作通过管道(Pipeline)处理文档,每个阶段对文档进行特定操作。

```javascript
db.orders.aggregate([
  // 第一阶段: 匹配条件
  { 
    $match: { 
      status: "completed" 
    } 
  },
  
  // 第二阶段: 分组并计算
  { 
    $group: {
      _id: "$customerId",
      totalAmount: { $sum: "$amount" },
      orderCount: { $sum: 1 },
      avgAmount: { $avg: "$amount" }
    }
  },
  
  // 第三阶段: 排序
  { 
    $sort: { 
      totalAmount: -1 
    } 
  },
  
  // 第四阶段: 限制结果数量
  { 
    $limit: 10 
  }
])
```

### 常用聚合阶段

#### $match - 过滤文档

```javascript
{ $match: { age: { $gte: 18 } } }
```

#### $group - 分组聚合

```javascript
{
  $group: {
    _id: "$category",              // 分组字段
    count: { $sum: 1 },            // 计数
    totalSales: { $sum: "$price" }, // 求和
    avgPrice: { $avg: "$price" },   // 平均值
    maxPrice: { $max: "$price" },   // 最大值
    minPrice: { $min: "$price" }    // 最小值
  }
}
```

#### $project - 字段投影

```javascript
{
  $project: {
    name: 1,
    age: 1,
    // 计算字段
    isAdult: { 
      $cond: { 
        if: { $gte: ["$age", 18] }, 
        then: true, 
        else: false 
      } 
    }
  }
}
```

#### $lookup - 关联查询

```javascript
{
  $lookup: {
    from: "orders",           // 要关联的集合
    localField: "_id",        // 本集合的字段
    foreignField: "userId",   // 关联集合的字段
    as: "userOrders"          // 输出字段名
  }
}
```

#### $unwind - 展开数组

```javascript
// 将数组字段展开为多个文档
{ $unwind: "$hobbies" }

// 原文档
{ name: "张三", hobbies: ["阅读", "游泳"] }

// 展开后
{ name: "张三", hobbies: "阅读" }
{ name: "张三", hobbies: "游泳" }
```

#### $sort - 排序

```javascript
{ $sort: { age: -1, name: 1 } }
```

#### $limit 和 $skip

```javascript
{ $skip: 10 }
{ $limit: 5 }
```

### 聚合示例

```javascript
// 统计每个城市的用户数量和平均年龄
db.users.aggregate([
  {
    $match: { 
      status: "active" 
    }
  },
  {
    $group: {
      _id: "$city",
      userCount: { $sum: 1 },
      avgAge: { $avg: "$age" },
      users: { $push: "$name" }  // 收集用户名到数组
    }
  },
  {
    $sort: { userCount: -1 }
  }
])

// 订单统计与用户信息关联
db.orders.aggregate([
  {
    $match: {
      orderDate: { 
        $gte: ISODate("2025-01-01") 
      }
    }
  },
  {
    $lookup: {
      from: "users",
      localField: "userId",
      foreignField: "_id",
      as: "userInfo"
    }
  },
  {
    $unwind: "$userInfo"
  },
  {
    $group: {
      _id: "$userInfo.city",
      totalOrders: { $sum: 1 },
      totalRevenue: { $sum: "$amount" }
    }
  },
  {
    $project: {
      city: "$_id",
      totalOrders: 1,
      totalRevenue: 1,
      avgOrderValue: { 
        $divide: ["$totalRevenue", "$totalOrders"] 
      }
    }
  }
])
```

---

## 事务支持

MongoDB 4.0+ 支持多文档 ACID 事务。

### 单文档事务

MongoDB 对单个文档的操作是原子性的,无需显式使用事务。

```javascript
// 单文档操作自动具有原子性
db.accounts.updateOne(
  { _id: 1 },
  { 
    $inc: { balance: -100 },
    $push: { 
      transactions: { 
        amount: -100, 
        date: new Date() 
      } 
    }
  }
)
```

### 多文档事务

```javascript
// 启动会话
const session = db.getMongo().startSession()

// 开始事务
session.startTransaction()

try {
  const accountsCollection = session.getDatabase("mydb").accounts
  
  // 转账操作
  accountsCollection.updateOne(
    { _id: "account1" },
    { $inc: { balance: -100 } },
    { session }
  )
  
  accountsCollection.updateOne(
    { _id: "account2" },
    { $inc: { balance: 100 } },
    { session }
  )
  
  // 提交事务
  session.commitTransaction()
  print("转账成功")
  
} catch (error) {
  // 回滚事务
  session.abortTransaction()
  print("转账失败: " + error)
  
} finally {
  session.endSession()
}
```

### 事务注意事项

- 事务有 60 秒的默认超时时间
- 事务中的操作总大小不能超过 16MB
- 事务只能在副本集或分片集群上使用
- 避免长时间运行的事务,影响性能

---

## 复制与分片

### 副本集 (Replica Set)

副本集是一组维护相同数据集的 MongoDB 服务器,提供冗余和高可用性。

**副本集架构:**
- **Primary**: 主节点,接收所有写操作
- **Secondary**: 从节点,复制主节点的数据,可提供读操作
- **Arbiter**: 仲裁节点,不存储数据,仅参与选举

```javascript
// 初始化副本集
rs.initiate({
  _id: "myReplicaSet",
  members: [
    { _id: 0, host: "mongodb0.example.com:27017" },
    { _id: 1, host: "mongodb1.example.com:27017" },
    { _id: 2, host: "mongodb2.example.com:27017" }
  ]
})

// 查看副本集状态
rs.status()

// 查看副本集配置
rs.conf()
```

**读偏好设置:**

```javascript
// 只从主节点读取(默认)
db.collection.find().readPref("primary")

// 优先从主节点读取
db.collection.find().readPref("primaryPreferred")

// 只从从节点读取
db.collection.find().readPref("secondary")

// 优先从从节点读取
db.collection.find().readPref("secondaryPreferred")
```

### 分片 (Sharding)

分片是 MongoDB 的水平扩展方案,将数据分布在多个服务器上。

**分片架构组件:**
- **Shard**: 分片服务器,存储数据子集
- **Config Servers**: 配置服务器,存储集群元数据
- **Mongos**: 路由服务器,查询路由器

```javascript
// 启用数据库分片
sh.enableSharding("myDatabase")

// 对集合进行分片
sh.shardCollection(
  "myDatabase.users",
  { userId: 1 }  // 分片键
)

// 查看分片状态
sh.status()
```

**分片策略:**
- **范围分片**: 根据分片键的值范围分配数据
- **哈希分片**: 根据分片键的哈希值分配数据
- **区域分片**: 根据地理位置或其他业务规则分配数据

---

## 常见问题 FAQ

### 1. MongoDB 与关系型数据库(如 MySQL)的主要区别是什么?

**数据模型:**
- MongoDB 使用文档模型(JSON/BSON),schema 灵活,适合半结构化数据
- MySQL 使用关系模型,需要预定义表结构,适合结构化数据

**查询语言:**
- MongoDB 使用 MQL(MongoDB Query Language),基于 JavaScript
- MySQL 使用 SQL(Structured Query Language)

**扩展性:**
- MongoDB 原生支持水平扩展(分片)
- MySQL 主要通过垂直扩展,水平扩展需要额外方案

**事务支持:**
- MongoDB 4.0+ 支持多文档 ACID 事务
- MySQL 长期支持完整的 ACID 事务

**使用场景:**
- MongoDB: 大数据、实时分析、内容管理、物联网数据
- MySQL: 财务系统、ERP、需要复杂关联查询的应用

### 2. 什么时候应该使用嵌入式文档,什么时候使用引用关系?

**使用嵌入式文档的场景:**
- 一对一关系(如用户和个人资料)
- 一对少的关系(如博客文章和评论,评论数量有限)
- 数据总是一起读取和更新
- 子文档不会独立查询
- 总文档大小不会超过 16MB 限制

**使用引用关系的场景:**
- 一对多关系,且"多"的一方数量很大
- 多对多关系
- 子文档需要独立查询和更新
- 避免数据冗余很重要
- 文档大小可能超过限制

**示例对比:**

```javascript
// 嵌入式 - 博客文章和少量评论
{
  title: "MongoDB 教程",
  content: "...",
  comments: [
    { author: "张三", text: "很好的文章" },
    { author: "李四", text: "学到了" }
  ]
}

// 引用 - 用户和大量订单
// users 集合
{ _id: 1, name: "张三" }

// orders 集合
{ _id: 101, userId: 1, amount: 99.99 }
{ _id: 102, userId: 1, amount: 199.99 }
```

### 3. 如何优化 MongoDB 的查询性能?

**索引优化:**
- 为常用查询字段创建合适的索引
- 使用复合索引覆盖多字段查询
- 避免创建过多索引(影响写入性能)
- 使用 `explain()` 分析查询计划

```javascript
// 创建复合索引
db.users.createIndex({ age: 1, city: 1 })

// 分析查询
db.users.find({ age: 25, city: "北京" }).explain("executionStats")
```

**查询优化:**
- 使用投影只返回需要的字段
- 避免扫描整个集合,使用合适的查询条件
- 合理使用 `limit()` 限制返回结果
- 对于大数据集使用游标分批处理

```javascript
// 只返回需要的字段
db.users.find(
  { age: { $gte: 25 } },
  { name: 1, email: 1, _id: 0 }
)
```

**数据模型优化:**
- 根据查询模式设计文档结构
- 合理使用嵌入或引用
- 避免深层嵌套

**硬件和配置:**
- 使用 SSD 存储
- 增加内存(工作集应该能放入内存)
- 启用压缩减少 I/O
- 使用副本集分散读负载

### 4. MongoDB 的 \_id 字段可以自定义吗?

可以。虽然 MongoDB 默认会生成 ObjectId 作为 `_id`,但你完全可以自定义。

```javascript
// 使用数字作为 _id
db.users.insertOne({
  _id: 1001,
  name: "张三"
})

// 使用字符串作为 _id
db.products.insertOne({
  _id: "PROD-12345",
  name: "笔记本电脑"
})

// 使用复合键作为 _id
db.logs.insertOne({
  _id: { date: "2025-01-18", userId: 100 },
  action: "login"
})
```

**注意事项:**
- `_id` 必须唯一
- `_id` 字段是不可变的(插入后不能修改)
- 自定义 `_id` 时要确保唯一性
- ObjectId 包含时间戳,自定义 ID 可能需要额外的时间字段

**最佳实践:**
- 如果有自然主键(如用户名、产品编码),可以使用自定义 ID
- 如果没有明确的自然主键,使用默认的 ObjectId
- 对于时序数据,ObjectId 的时间戳特性很有用

### 5. MongoDB 如何保证数据一致性和持久化?

**写关注 (Write Concern):**

控制写操作的确认级别。

```javascript
// 等待写入到主节点确认(默认)
db.users.insertOne(
  { name: "张三" },
  { writeConcern: { w: 1 } }
)

// 等待写入到大多数节点确认
db.users.insertOne(
  { name: "李四" },
  { writeConcern: { w: "majority" } }
)

// 等待写入到指定数量的节点
db.users.insertOne(
  { name: "王五" },
  { writeConcern: { w: 2, wtimeout: 5000 } }  // 5秒超时
)

// 等待写入日志
db.users.insertOne(
  { name: "赵六" },
  { writeConcern: { w: 1, j: true } }  // j: journal
)
```

**读关注 (Read Concern):**

控制读取数据的一致性级别。

```javascript
// 读取本地数据(默认)
db.users.find().readConcern("local")

// 读取大多数节点确认的数据
db.users.find().readConcern("majority")

// 线性化读取(最强一致性)
db.users.find().readConcern("linearizable")
```

**持久化机制:**
- **Journal 日志**: 写操作先写入日志,确保崩溃后可恢复
- **检查点**: 定期将内存数据刷新到磁盘
- **副本集**: 数据复制到多个节点,防止单点故障

**配置建议:**
- 对于关键数据,使用 `w: "majority"` 和 `j: true`
- 对于可容忍短暂数据丢失的场景,可以降低写关注级别以提高性能
- 根据业务需求平衡一致性和性能

```javascript
// 关键交易使用强一致性
db.transactions.insertOne(
  { 
    from: "account1", 
    to: "account2", 
    amount: 1000 
  },
  { 
    writeConcern: { 
      w: "majority", 
      j: true,
      wtimeout: 5000
    } 
  }
)
```

---

## 总结

MongoDB 作为领先的 NoSQL 数据库,提供了灵活的文档模型、强大的查询能力和优秀的扩展性。理解其核心概念、数据建模策略、索引机制和聚合框架是高效使用 MongoDB 的关键。

**关键要点:**
- 文档模型提供了灵活性,但需要根据查询模式合理设计
- 索引是提升查询性能的关键,但要避免过度索引
- 聚合框架提供了强大的数据处理能力
- 副本集和分片提供了高可用性和扩展性
- 根据业务需求选择合适的一致性级别

持续学习和实践是掌握 MongoDB 的最佳途径。建议通过实际项目加深理解,并关注 MongoDB 官方文档获取最新特性和最佳实践。
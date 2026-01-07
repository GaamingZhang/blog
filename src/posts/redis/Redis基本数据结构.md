# Redis基本数据结构

Redis 是一个开源的高性能键值存储系统，以其丰富的数据结构和卓越的性能而闻名。作为一名软件开发工程师，深入理解 Redis 的基本数据结构是掌握 Redis 核心功能的基础。Redis 提供了多种数据结构，每种结构都有其特定的使用场景和性能特性。

## Redis 主要数据结构

Redis 支持多种数据结构，每种结构都针对不同的应用场景进行了优化：

- **String（字符串）**：最基本的数据结构，可存储文本、数字或二进制数据
- **List（列表）**：有序的字符串集合，支持两端操作
- **Hash（哈希表）**：键值对集合，适合存储对象
- **Set（集合）**：无序的字符串集合，支持交、并、差等集合操作
- **Sorted Set（有序集合）**：带权重的有序集合，支持范围查询

本文将详细介绍这些数据结构的底层实现、常用命令、性能特性和使用场景，帮助您更好地理解和使用 Redis。

## 详细数据结构

### String（字符串）

String 是 Redis 最基本的数据结构，也是其他数据结构的基础。它可以存储文本、数字或二进制数据，最大长度为 512MB。

#### 底层实现

Redis 并没有直接使用 C 语言的字符串实现，而是自己实现了一种名为 **SDS（Simple Dynamic String）** 的动态字符串结构。SDS 具有以下特点：

- **动态扩容**：当字符串长度增加时，会自动扩展内存空间
- **二进制安全**：可以存储任何二进制数据，包括空字符
- **O(1) 时间复杂度获取字符串长度**：通过内部维护的 len 属性
- **空间预分配**：减少内存分配次数，提高性能

SDS 的结构如下：

```c
struct sdshdr {
    int len;      // 已使用的字节数
    int free;     // 未使用的字节数
    char buf[];   // 字符数组
};
```

#### 常用命令

| 命令 | 功能 | 示例 |
|------|------|------|
| `SET key value` | 设置字符串值 | `SET name "Redis"` |
| `GET key` | 获取字符串值 | `GET name` |
| `INCR key` | 自增整数 | `INCR counter` |
| `DECR key` | 自减整数 | `DECR counter` |
| `INCRBY key increment` | 增加指定整数 | `INCRBY counter 5` |
| `DECRBY key decrement` | 减少指定整数 | `DECRBY counter 3` |
| `APPEND key value` | 追加字符串 | `APPEND name " Database"` |
| `STRLEN key` | 获取字符串长度 | `STRLEN name` |
| `SETNX key value` | 仅当键不存在时设置值 | `SETNX lock "active"` |
| `GETSET key value` | 获取旧值并设置新值 | `GETSET counter 0` |

#### 性能特性

- 设置和获取值的时间复杂度为 O(1)
- 字符串长度小于 1MB 时，扩容空间加倍；大于 1MB 时，每次增加 1MB
- 整数类型的字符串支持原子自增/自减操作

#### 使用场景

- 缓存用户信息、配置信息等
- 计数器（如页面访问量、点赞数）
- 分布式锁（使用 SETNX 命令）
- 存储 JSON 序列化的数据
- 分布式 ID 生成器

### List（列表）

List 是 Redis 中的有序字符串集合，支持在两端进行插入和删除操作。List 中的元素可以重复，并且保持插入顺序。

#### 底层实现

在 Redis 3.2 之前，List 使用两种数据结构实现：
- **ziplist**：当列表元素较少且较小时使用
- **linkedlist**：当列表元素较多或较大时使用

从 Redis 3.2 开始，List 使用 **quicklist**（快速列表）作为底层实现。quicklist 是 ziplist 和 linkedlist 的结合体，它将列表分割成多个 ziplist，然后用链表将这些 ziplist 连接起来。这种设计兼顾了内存效率和操作性能。

quicklist 的结构如下：

```c
struct quicklist {
    quicklistNode *head;      // 头节点
    quicklistNode *tail;      // 尾节点
    unsigned long count;      // 元素总数
    unsigned int len;         // ziplist 节点数量
    int fill : 16;            // ziplist 大小限制
    unsigned int compress : 16;// 压缩深度
};

struct quicklistNode {
    struct quicklistNode *prev;  // 前一个节点
    struct quicklistNode *next;  // 后一个节点
    unsigned char *zl;           // ziplist 指针
    unsigned int sz;             // ziplist 字节大小
    unsigned int count : 16;     // ziplist 元素数量
    unsigned int encoding : 2;   // 编码方式
    unsigned int container : 2;  // 容器类型
    unsigned int recompress : 1; // 是否需要重新压缩
    unsigned int attempted_compress : 1; // 压缩尝试标志
    unsigned int extra : 10;     // 预留字段
};
```

#### 常用命令

| 命令 | 功能 | 示例 |
|------|------|------|
| `LPUSH key value [value ...]` | 从左侧插入元素 | `LPUSH numbers 1 2 3` |
| `RPUSH key value [value ...]` | 从右侧插入元素 | `RPUSH numbers 4 5 6` |
| `LPOP key` | 从左侧弹出元素 | `LPOP numbers` |
| `RPOP key` | 从右侧弹出元素 | `RPOP numbers` |
| `LLEN key` | 获取列表长度 | `LLEN numbers` |
| `LRANGE key start stop` | 获取指定范围的元素 | `LRANGE numbers 0 -1` |
| `LINDEX key index` | 获取指定索引的元素 | `LINDEX numbers 2` |
| `LSET key index value` | 设置指定索引的元素 | `LSET numbers 2 "three"` |
| `LREM key count value` | 删除指定数量的元素 | `LREM numbers 2 3` |
| `BLPOP key [key ...] timeout` | 阻塞式左侧弹出 | `BLPOP queue 0` |
| `BRPOP key [key ...] timeout` | 阻塞式右侧弹出 | `BRPOP queue 0` |

#### 性能特性

- **两端操作**：LPUSH、RPUSH、LPOP、RPOP 命令的时间复杂度为 O(1)
- **中间操作**：LINDEX、LSET、LREM 等命令的时间复杂度为 O(n)
- **范围查询**：LRANGE 命令的时间复杂度为 O(start + stop)
- **内存效率**：quicklist 结合了 ziplist 的内存效率和 linkedlist 的灵活性

#### 使用场景

- **消息队列**：使用 LPUSH 和 BRPOP 实现简单的消息队列
- **最新消息展示**：使用 LPUSH 和 LRANGE 获取最新的 N 条消息
- **任务队列**：存储待处理的任务，使用 BRPOP 阻塞获取任务
- **数据栈**：使用 LPUSH 和 LPOP 实现栈结构
- **双向链表**：利用 List 的两端操作特性实现双向链表

### Hash（哈希表）

Hash 是 Redis 中的键值对集合，类似于其他编程语言中的字典或对象。每个 Hash 可以存储多个键值对，适合用于存储对象数据。

#### 底层实现

Redis Hash 的底层实现有两种：

1. **ziplist（压缩列表）**：当 Hash 中的键值对数量较少且键值都较小时使用
2. **hashtable（哈希表）**：当 Hash 中的键值对数量较多或键值较大时使用

切换条件由以下两个配置决定：
- `hash-max-ziplist-entries`：默认 512，当键值对数量超过此值时切换到 hashtable
- `hash-max-ziplist-value`：默认 64 字节，当任意键值长度超过此值时切换到 hashtable

ziplist 是一种紧凑的内存结构，适合存储小数据；hashtable 则提供了更快的查找速度，但内存占用较大。

#### 常用命令

| 命令 | 功能 | 示例 |
|------|------|------|
| `HSET key field value` | 设置哈希表字段值 | `HSET user:1 name "Redis"` |
| `HGET key field` | 获取哈希表字段值 | `HGET user:1 name` |
| `HMSET key field1 value1 field2 value2 ...` | 批量设置哈希表字段值 | `HMSET user:1 age 30 gender "male"` |
| `HMGET key field1 field2 ...` | 批量获取哈希表字段值 | `HMGET user:1 name age` |
| `HGETALL key` | 获取哈希表所有字段和值 | `HGETALL user:1` |
| `HDEL key field1 field2 ...` | 删除哈希表字段 | `HDEL user:1 age` |
| `HLEN key` | 获取哈希表字段数量 | `HLEN user:1` |
| `HEXISTS key field` | 判断哈希表字段是否存在 | `HEXISTS user:1 name` |
| `HKEYS key` | 获取哈希表所有字段 | `HKEYS user:1` |
| `HVALS key` | 获取哈希表所有值 | `HVALS user:1` |
| `HINCRBY key field increment` | 增加哈希表字段整数 | `HINCRBY user:1 score 10` |

#### 性能特性

- **ziplist 实现**：
  - 设置和获取字段值的时间复杂度为 O(n)
  - 内存占用小，适合存储小对象
- **hashtable 实现**：
  - 设置和获取字段值的时间复杂度为 O(1)
  - 内存占用较大，但查找速度快
- **字段数限制**：理论上没有限制，但实际使用中应避免存储过多字段

#### 使用场景

- **存储对象**：如用户信息、商品信息、配置信息等
- **计数器集合**：存储多个相关计数器，如用户的点赞数、评论数、分享数
- **缓存数据**：缓存数据库查询结果，以键值对形式存储
- **会话管理**：存储用户会话信息，每个会话对应一个 Hash
- **统计数据**：存储各类统计指标，如网站访问量、API 调用次数

### Set（集合）

Set 是 Redis 中的无序字符串集合，集合中的元素具有唯一性（不允许重复）。Set 支持丰富的集合操作，如交集、并集、差集等。

#### 底层实现

Redis Set 的底层实现有两种：

1. **intset（整数集合）**：当 Set 中的元素都是整数且数量较少时使用
2. **hashtable（哈希表）**：当 Set 中的元素包含非整数或数量较多时使用

切换条件由 `set-max-intset-entries` 配置决定，默认值为 512。当整数元素数量超过此值时，会自动切换到 hashtable 实现。

intset 是一种紧凑的内存结构，适合存储小范围整数；hashtable 则提供了更快的元素查找和操作速度。

#### 常用命令

| 命令 | 功能 | 示例 |
|------|------|------|
| `SADD key member [member ...]` | 添加集合元素 | `SADD fruits "apple" "banana"` |
| `SMEMBERS key` | 获取所有集合元素 | `SMEMBERS fruits` |
| `SISMEMBER key member` | 判断元素是否在集合中 | `SISMEMBER fruits "apple"` |
| `SREM key member [member ...]` | 删除集合元素 | `SREM fruits "banana"` |
| `SCARD key` | 获取集合元素数量 | `SCARD fruits` |
| `SPOP key [count]` | 随机弹出集合元素 | `SPOP fruits 2` |
| `SRANDMEMBER key [count]` | 随机获取集合元素 | `SRANDMEMBER fruits 3` |
| `SINTER key1 [key2 ...]` | 获取多个集合的交集 | `SINTER set1 set2` |
| `SUNION key1 [key2 ...]` | 获取多个集合的并集 | `SUNION set1 set2` |
| `SDIFF key1 [key2 ...]` | 获取多个集合的差集 | `SDIFF set1 set2` |
| `SINTERSTORE destination key1 [key2 ...]` | 交集结果存储到新集合 | `SINTERSTORE set3 set1 set2` |
| `SUNIONSTORE destination key1 [key2 ...]` | 并集结果存储到新集合 | `SUNIONSTORE set3 set1 set2` |
| `SDIFFSTORE destination key1 [key2 ...]` | 差集结果存储到新集合 | `SDIFFSTORE set3 set1 set2` |

#### 性能特性

- **intset 实现**：
  - 添加、删除、查找元素的时间复杂度为 O(n)
  - 内存占用小，适合存储小范围整数
- **hashtable 实现**：
  - 添加、删除、查找元素的时间复杂度为 O(1)
  - 内存占用较大，但操作速度快
- **集合操作**：
  - 交集、并集、差集的时间复杂度取决于参与操作的集合大小
  - 对于大数据量的集合操作，建议使用 STORE 命令将结果存储起来

#### 使用场景

- **标签系统**：存储用户或内容的标签，支持标签搜索和过滤
- **好友关系**：存储用户的好友列表，支持好友推荐（通过交集查找共同好友）
- **去重操作**：去除重复数据，如统计网站独立访客
- **抽奖系统**：使用 SPOP 或 SRANDMEMBER 实现随机抽奖功能
- **权限管理**：存储用户的权限集合，支持权限检查和权限组管理

### Sorted Set（有序集合）

Sorted Set（简称 ZSet）是 Redis 中的有序集合，它结合了 Set 的唯一性和 List 的有序性。每个元素都有一个分数（score），用于确定元素在集合中的排序位置。

#### 底层实现

Redis ZSet 的底层实现有两种：

1. **ziplist（压缩列表）**：当 ZSet 中的元素数量较少且元素和分数都较小时使用
2. **skiplist + dict（跳跃表 + 字典）**：当 ZSet 中的元素数量较多或元素和分数较大时使用

切换条件由以下两个配置决定：
- `zset-max-ziplist-entries`：默认 128，当元素数量超过此值时切换
- `zset-max-ziplist-value`：默认 64 字节，当任意元素或分数长度超过此值时切换

跳跃表是一种有序数据结构，通过在每个节点中维护多个指向其他节点的指针，从而实现快速的查找、插入和删除操作。字典则用于快速查找元素的分数。

#### 常用命令

| 命令 | 功能 | 示例 |
|------|------|------|
| `ZADD key score member [score member ...]` | 添加有序集合元素 | `ZADD leaderboard 100 "user1" 90 "user2"` |
| `ZRANGE key start stop [WITHSCORES]` | 获取指定范围的元素（按分数升序） | `ZRANGE leaderboard 0 10 WITHSCORES` |
| `ZREVRANGE key start stop [WITHSCORES]` | 获取指定范围的元素（按分数降序） | `ZREVRANGE leaderboard 0 4 WITHSCORES` |
| `ZSCORE key member` | 获取元素的分数 | `ZSCORE leaderboard "user1"` |
| `ZINCRBY key increment member` | 增加元素的分数 | `ZINCRBY leaderboard 10 "user1"` |
| `ZREM key member [member ...]` | 删除有序集合元素 | `ZREM leaderboard "user2"` |
| `ZCARD key` | 获取有序集合元素数量 | `ZCARD leaderboard` |
| `ZRANK key member` | 获取元素的排名（按分数升序） | `ZRANK leaderboard "user1"` |
| `ZREVRANK key member` | 获取元素的排名（按分数降序） | `ZREVRANK leaderboard "user1"` |
| `ZRANGEBYSCORE key min max [WITHSCORES] [LIMIT offset count]` | 按分数范围获取元素 | `ZRANGEBYSCORE leaderboard 80 100` |
| `ZCOUNT key min max` | 统计分数范围内的元素数量 | `ZCOUNT leaderboard 90 100` |
| `ZREMRANGEBYRANK key start stop` | 删除指定排名范围的元素 | `ZREMRANGEBYRANK leaderboard 0 9` |
| `ZREMRANGEBYSCORE key min max` | 删除指定分数范围的元素 | `ZREMRANGEBYSCORE leaderboard 0 50` |

#### 性能特性

- **ziplist 实现**：
  - 插入、删除、查找的时间复杂度为 O(n)
  - 内存占用小，适合存储小数据集
- **skiplist + dict 实现**：
  - 插入、删除、查找的时间复杂度为 O(log n)
  - 范围查询的时间复杂度为 O(log n + m)，其中 m 是返回的元素数量
  - 内存占用较大，但操作速度快
- **分数精度**：使用双精度浮点数存储分数，可能存在精度问题

#### 使用场景

- **排行榜**：游戏排行榜、用户积分排行榜等
- **优先级队列**：根据任务优先级进行调度
- **范围查询**：根据时间戳或分数范围获取数据
- **延迟队列**：结合 ZADD 和 ZRANGEBYSCORE 实现延迟任务
- **计数器排序**：对多个计数器进行排序，如商品销量排行

## 常见问题

### 1. Redis的String和其他编程语言中的String有什么区别？

Redis的String与其他编程语言中的String有以下主要区别：

- **底层实现**：Redis使用SDS（Simple Dynamic String）实现String，而不是直接使用C语言的字符串
- **二进制安全**：Redis的String可以存储任何二进制数据，包括空字符，而传统String通常以空字符结尾
- **动态扩容**：Redis的String会自动扩展内存空间，而传统String需要手动管理内存
- **O(1)获取长度**：Redis的String通过内部len属性可以在O(1)时间内获取长度，而传统String需要遍历整个字符串
- **丰富的操作**：Redis提供了原子的自增、自减、追加等操作，而传统String通常没有这些内置操作

### 2. List、Set和Sorted Set有什么区别？各适合什么场景？

| 数据结构 | 有序性 | 唯一性 | 主要特点 | 适用场景 |
|----------|--------|--------|----------|----------|
| List | 有序（插入顺序） | 不唯一 | 两端操作高效 | 消息队列、最新消息展示、栈/队列实现 |
| Set | 无序 | 唯一 | 支持集合运算 | 标签系统、好友关系、去重操作 |
| Sorted Set | 有序（分数排序） | 唯一 | 带权重的有序集合 | 排行榜、优先级队列、范围查询、延迟队列 |

### 3. Hash和String都可以存储对象，应该如何选择？

选择Hash还是String存储对象主要取决于以下因素：

- **访问需求**：如果需要频繁修改对象的某个字段，Hash更高效，因为可以只修改特定字段而不需要重新设置整个对象
- **对象大小**：对于较小的对象，String可能更节省内存（特别是当序列化后的字符串较小时）
- **查询需求**：如果需要查询对象的多个字段，Hash可以批量获取字段，而String需要先反序列化整个对象
- **数据结构**：如果对象结构简单且不需要单独操作字段，String（存储JSON或序列化数据）可能更方便

一般来说，对于复杂对象且需要频繁操作其字段的场景，Hash是更好的选择；对于简单对象或需要整体操作的场景，String可能更合适。

### 4. 为什么Redis的Sorted Set使用跳跃表而不是红黑树？

Redis选择跳跃表而不是红黑树作为Sorted Set的主要实现，主要有以下原因：

- **范围查询性能**：跳跃表在范围查询方面比红黑树更高效，因为可以直接通过层级指针快速定位到范围的起始位置
- **实现复杂度**：跳跃表的实现比红黑树更简单，容易理解和维护
- **插入/删除性能**：虽然平均时间复杂度都是O(log n)，但跳跃表的常数因子通常更小，实际性能可能更好
- **内存占用**：跳跃表的内存占用相对可控，可以通过调整最大层级来平衡性能和内存使用

### 5. 如何选择合适的Redis数据结构来解决实际问题？

选择合适的Redis数据结构需要考虑以下几个方面：

1. **数据的特性**：是否需要有序性、唯一性、带权重等
2. **操作的需求**：需要哪些操作（增删改查、范围查询、集合运算等）
3. **性能的要求**：操作的时间复杂度和内存占用
4. **使用的场景**：具体的业务场景和数据访问模式

一个简单的决策流程：
- 如果需要存储简单的键值对：使用String
- 如果需要存储对象且频繁操作其字段：使用Hash
- 如果需要有序的列表且两端操作频繁：使用List
- 如果需要唯一的元素集合：使用Set
- 如果需要带权重的有序集合：使用Sorted Set

在实际应用中，有时也需要结合多种数据结构来解决复杂问题。
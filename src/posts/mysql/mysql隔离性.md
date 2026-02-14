---
date: 2026-01-17
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 数据库
tag:
  - 数据库
---

# MySQL 的隔离性：从锁到 MVCC 的并发控制

## 为什么需要隔离性？

想象一个经典场景：两个人同时转账到同一个账户。

```
初始状态: 账户余额 = 1000元

事务A: 转入500元
    1. 读取余额: 1000
    2. 计算新余额: 1000 + 500 = 1500
    3. 写入新余额: 1500

事务B: 转入300元
    1. 读取余额: 1000
    2. 计算新余额: 1000 + 300 = 1300
    3. 写入新余额: 1300

最终结果: 1300元 ← 错误！应该是1800元
```

问题在于：事务B覆盖了事务A的修改。**隔离性就是为了防止并发事务相互干扰**。

## 三种并发问题

### 脏读（Dirty Read）

读到了其他事务**未提交**的数据。

```
时刻    事务A                          事务B
T1     BEGIN;
T2     UPDATE balance = 900;
       (未提交)
T3                                    BEGIN;
T4                                    SELECT balance;
                                      → 读到900（脏数据）
T5     ROLLBACK;
       (A的修改被回滚)
T6                                    基于900做决策
                                      → 错误！
```

**为什么叫"脏"？** 因为这个数据可能会被回滚，是"不干净的"、不可靠的。

### 不可重复读（Non-Repeatable Read）

同一个事务内，**同样的查询读到了不同的结果**。

```
时刻    事务A                          事务B
T1     BEGIN;
T2     SELECT balance;
       → 读到1000
T3                                    BEGIN;
T4                                    UPDATE balance = 800;
                                      COMMIT;
T5     SELECT balance;
       → 读到800 ← 与T2不一致
```

**为什么是问题？** 因为事务A的两次读取结果不一致，违反了"可重复读"的预期。

### 幻读（Phantom Read）

同一个事务内，**同样的查询，结果集的行数变了**。

```
时刻    事务A                          事务B
T1     BEGIN;
T2     SELECT COUNT(*) FROM orders WHERE user_id=1;
       → 结果: 5条记录
T3                                    BEGIN;
T4                                    INSERT INTO orders (user_id, ...) VALUES (1, ...);
                                      COMMIT;
T5     SELECT COUNT(*) FROM orders WHERE user_id=1;
       → 结果: 6条记录 ← 出现了"幻影行"
```

**为什么叫"幻"？** 因为凭空出现了新行（或消失了旧行），像幻觉一样。

**与不可重复读的区别**：
- 不可重复读：**已有行的值变了**
- 幻读：**行数变了**（出现新行或行消失）

## 四种隔离级别

MySQL 提供四种隔离级别，本质是**在并发性能和数据一致性之间的不同权衡**。

```
隔离级别         脏读    不可重复读    幻读    实现方式
────────────────────────────────────────────────────
READ UNCOMMITTED  可能    可能         可能    不加锁
READ COMMITTED    避免    可能         可能    MVCC + 行锁
REPEATABLE READ   避免    避免         部分避免 MVCC + Next-Key Lock
SERIALIZABLE      避免    避免         避免    锁表/锁全部扫描行
```

### READ UNCOMMITTED：完全不隔离

**实现方式**：读操作不加任何锁，直接读最新数据。

**问题**：会读到未提交的数据（脏读）。

**使用场景**：几乎不用。唯一可能的场景是：对一致性要求极低的实时统计。

### READ COMMITTED：读已提交

**实现方式**：
- 写操作加排他锁（X锁），事务结束才释放
- 读操作使用 MVCC，**每次读取都创建新的 Read View**

**核心机制**：每次 SELECT 都重新判断"哪些事务的修改可见"。

```
事务A                                  事务B
BEGIN;
SELECT balance WHERE id=1;
→ 创建 Read View1，读到1000

                                      BEGIN;
                                      UPDATE balance=800 WHERE id=1;
                                      COMMIT;

SELECT balance WHERE id=1;
→ 创建 Read View2（重新判断可见性）
→ 事务B已提交，可见
→ 读到800
```

**特点**：
- 避免了脏读（不会读到未提交的）
- 无法避免不可重复读（每次读取创建新快照）
- 无法避免幻读

**适用场景**：Oracle、PostgreSQL 的默认级别。适合对实时性要求高的场景。

### REPEATABLE READ：可重复读

**实现方式**：
- 写操作加排他锁
- 读操作使用 MVCC，**事务开始时创建 Read View，整个事务期间复用**

**核心机制**：一次快照，全程使用。

```
事务A                                  事务B
BEGIN;
第一次读取：
  → 创建 Read View（记录此刻的活跃事务）
  → 读到1000

                                      BEGIN;
                                      UPDATE balance=800;
                                      COMMIT;

第二次读取：
  → 复用同一个 Read View
  → 事务B在Read View创建时还活跃，不可见
  → 仍然读到1000（可重复读）
```

**InnoDB 如何避免部分幻读？**

通过 **Next-Key Lock（记录锁 + 间隙锁）**。

假设表中有记录：id = 5, 10, 15
```
SELECT * FROM t WHERE id > 8 FOR UPDATE;

加锁范围：
- 记录锁：锁住 id=10, id=15
- 间隙锁：锁住区间 (5, 10), (10, 15), (15, +∞)

效果：
  其他事务无法在这些区间插入新行
  → 避免了幻读
```

**但是**：如果不使用当前读（FOR UPDATE/LOCK IN SHARE MODE），仍可能出现幻读：
```
事务A:
  SELECT COUNT(*) FROM t WHERE id > 8;  ← 快照读，不加锁
  → 结果: 2行

事务B:
  INSERT INTO t (id) VALUES (12);
  COMMIT;

事务A:
  SELECT COUNT(*) FROM t WHERE id > 8;  ← 仍然是快照读
  → 结果: 仍然2行（MVCC看不到新插入的）

  UPDATE t SET status=1 WHERE id > 8;   ← 当前读，会看到最新数据
  → 影响了3行 ← 出现幻读
```

**适用场景**：MySQL InnoDB 的默认级别。适合大多数场景。

### SERIALIZABLE：串行化

**实现方式**：
- 所有 SELECT 自动加共享锁（S锁）
- 所有修改操作加排他锁（X锁）
- 锁一直持有到事务结束

**效果**：事务完全串行执行，就像同一时刻只有一个事务在运行。

```
事务A:
  BEGIN;
  SELECT * FROM t WHERE id=1;  ← 自动加S锁

事务B:
  BEGIN;
  UPDATE t SET name='x' WHERE id=1;  ← 需要X锁
  → 阻塞，等待A释放S锁
```

**特点**：
- 完全避免脏读、不可重复读、幻读
- 性能极差，并发能力最低

**适用场景**：对一致性要求极高且并发量很低的场景（如银行日终批处理）。

## 锁机制的工作原理

### 两种基本锁

**共享锁（S锁，Shared Lock）**
- 允许读取，禁止修改
- 多个事务可以同时持有同一数据的S锁
- 用法：`SELECT ... LOCK IN SHARE MODE`

**排他锁（X锁，Exclusive Lock）**
- 禁止其他事务读取或修改
- 只有一个事务能持有某数据的X锁
- 自动加锁：UPDATE、DELETE、INSERT
- 手动加锁：`SELECT ... FOR UPDATE`

**兼容性矩阵**：
```
       持有S锁    持有X锁
请求S锁   ✓        ✗
请求X锁   ✗        ✗
```

### 锁的粒度

**行锁（Row Lock）**
- 只锁定特定的行
- 并发性能最好
- InnoDB 默认使用行锁

**间隙锁（Gap Lock）**
- 锁定索引记录之间的"间隙"
- 防止其他事务在间隙中插入数据
- 用于解决幻读问题

**Next-Key Lock = 记录锁 + 间隙锁**
- 锁定记录本身 + 记录前的间隙
- REPEATABLE READ 默认使用

**示例**：
```
表中数据: id = 10, 20, 30

事务A: SELECT * FROM t WHERE id = 20 FOR UPDATE;

加锁情况（REPEATABLE READ）:
  - 记录锁: 锁住 id=20
  - 间隙锁: 锁住 (10, 20) 这个区间

阻塞的操作:
  - INSERT INTO t (id) VALUES (15);  ← 在间隙中，被阻塞
  - UPDATE t SET ... WHERE id=20;    ← 记录被锁，阻塞
  - DELETE FROM t WHERE id=20;       ← 记录被锁，阻塞

不阻塞的操作:
  - INSERT INTO t (id) VALUES (25);  ← 不在锁定区间
  - UPDATE t SET ... WHERE id=10;    ← 不同的记录
```

### 锁的释放时机

**关键原则**：锁在事务**提交或回滚**时才释放。

```
BEGIN;
UPDATE t SET balance=900 WHERE id=1;  ← 此时加X锁
-- 执行其他操作...
-- X锁一直持有
COMMIT;  ← 此时才释放X锁
```

这就是为什么长事务会导致锁等待——它长时间持有锁，阻塞其他事务。

## MVCC：无锁的并发控制

### MVCC 的核心思想

**快照读**（Snapshot Read）：
- 读取数据的历史版本，不加锁
- 不同事务看到不同的数据快照
- 实现读写并发

**当前读**（Current Read）：
- 读取数据的最新版本，加锁
- 包括：FOR UPDATE、LOCK IN SHARE MODE、UPDATE、DELETE、INSERT

### Read View 的工作机制（复习）

每个事务的 Read View 包含：
```
- m_ids: 创建Read View时，所有活跃事务的ID列表
- min_trx_id: m_ids中最小的事务ID
- max_trx_id: 下一个将被分配的事务ID
- creator_trx_id: 创建该Read View的事务ID
```

**可见性判断**：对于数据行的事务ID（DB_TRX_ID）：
```
1. DB_TRX_ID < min_trx_id
   → 该版本在所有活跃事务之前提交，可见

2. DB_TRX_ID >= max_trx_id
   → 该版本是在Read View创建后产生的，不可见

3. min_trx_id <= DB_TRX_ID < max_trx_id
   → 如果DB_TRX_ID在m_ids中（事务还活跃），不可见
   → 如果DB_TRX_ID不在m_ids中（事务已提交），可见

4. DB_TRX_ID == creator_trx_id
   → 是自己修改的，可见
```

### READ COMMITTED vs REPEATABLE READ

**READ COMMITTED**：
- 每次 SELECT 都创建新的 Read View
- 能看到其他事务新提交的数据
- 导致不可重复读

**REPEATABLE READ**：
- 事务开始时（第一次读取时）创建 Read View
- 整个事务期间复用同一个 Read View
- 看不到其他事务的新提交
- 实现可重复读

## 隔离级别的选择

### 性能 vs 一致性的权衡

```
隔离级别          并发性能    数据一致性    锁开销
────────────────────────────────────────────
READ UNCOMMITTED   最高       最低         无
READ COMMITTED     高         中           低（主要MVCC）
REPEATABLE READ    中         高           中（MVCC + 间隙锁）
SERIALIZABLE       最低       最高         最高（全加锁）
```

### 实际场景建议

**金融核心交易**：
- 使用 REPEATABLE READ 或 SERIALIZABLE
- 关键操作使用 FOR UPDATE（当前读）
- 宁可牺牲性能也要保证一致性

**电商订单系统**：
- 使用 REPEATABLE READ（默认）
- 库存扣减使用 FOR UPDATE
- 一般查询使用快照读

**社交网络、内容平台**：
- 可以考虑 READ COMMITTED
- 对实时性要求高
- 允许一定的不一致（如点赞数、评论数）

**数据仓库/分析**：
- READ UNCOMMITTED 用于粗略统计
- 或者使用只读副本，不影响主库

## 常见陷阱

### 陷阱1：以为 REPEATABLE READ 完全避免了幻读

```
事务A:
  SELECT COUNT(*) FROM orders WHERE status='pending';
  → 10条

  -- 其他事务插入了新的pending订单

  UPDATE orders SET status='processing' WHERE status='pending';
  → 影响了11条 ← 幻读！

UPDATE是当前读，会看到其他事务提交的新行。
```

**解决方案**：用 FOR UPDATE 锁住查询范围
```sql
SELECT COUNT(*) FROM orders WHERE status='pending' FOR UPDATE;
→ 加Next-Key Lock，阻止新插入
```

### 陷阱2：死锁

```
事务A                              事务B
BEGIN;                             BEGIN;
UPDATE t SET ... WHERE id=1;
(持有id=1的X锁)
                                   UPDATE t SET ... WHERE id=2;
                                   (持有id=2的X锁)
UPDATE t SET ... WHERE id=2;
(等待id=2的锁)
                                   UPDATE t SET ... WHERE id=1;
                                   (等待id=1的锁)

← 死锁！
```

**InnoDB 的处理**：
- 自动检测死锁（超时或等待图算法）
- 回滚其中一个事务（通常是持有锁少的）
- 另一个事务可以继续

**避免死锁**：
- 按相同顺序访问资源
- 缩短事务时间
- 使用较低的隔离级别（如RC）

### 陷阱3：间隙锁导致的意外阻塞

```
表中数据: id = 10, 20, 30

事务A:
  SELECT * FROM t WHERE id = 15 FOR UPDATE;
  → 没有id=15的记录
  → 但是加了间隙锁 (10, 20)

事务B:
  INSERT INTO t (id) VALUES (12);
  → 被阻塞！
```

这是 REPEATABLE READ 防止幻读的代价。如果不需要防幻读，可以考虑 READ COMMITTED。

## 小结

**隔离性的实现依赖两大机制**：

**锁机制**：
- 共享锁（S锁）允许并发读
- 排他锁（X锁）保证独占修改
- Next-Key Lock 防止幻读
- 代价：性能开销，可能死锁

**MVCC（多版本并发控制）**：
- 基于 Undo Log 的版本链
- Read View 控制可见性
- 快照读不加锁，读写并发
- 代价：Undo 空间占用，长事务影响

**四种隔离级别的本质**：
- 不同的锁策略 + MVCC 的不同使用方式
- 在性能和一致性之间权衡
- 没有"最好的"级别，只有"最合适的"

**实践建议**：
- 大多数场景使用默认的 REPEATABLE READ
- 关键操作使用当前读（FOR UPDATE）
- 避免长事务
- 根据业务需求在性能和一致性间权衡

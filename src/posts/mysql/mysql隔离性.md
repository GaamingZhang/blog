# MySQL的隔离性

## 概述

隔离性(Isolation)是ACID事务特性中的"I",指的是并发执行的事务之间相互隔离,一个事务的执行不应被其他事务干扰。隔离性确保即使多个事务同时操作数据库,每个事务都感觉像是独占系统一样。

**隔离性的核心目标:**

- **防止脏读(Dirty Read)**: 读取到未提交的数据
- **防止不可重复读(Non-Repeatable Read)**: 同一查询返回不同结果
- **防止幻读(Phantom Read)**: 查询结果集中出现新行
- **保证数据一致性**: 并发操作不破坏数据完整性

**隔离性的实现机制:**

- **锁机制**: 通过锁控制并发访问
- **MVCC(多版本并发控制)**: 通过数据版本实现读写分离
- **时间戳排序**: 根据时间戳确定事务执行顺序

## 事务隔离级别

SQL标准定义了四种事务隔离级别,从低到高依次为:

### 隔离级别概览

| 隔离级别 | 脏读 | 不可重复读 | 幻读 | 实现方式 | 并发性能 |
|---------|------|-----------|------|----------|----------|
| READ UNCOMMITTED | 可能 | 可能 | 可能 | 不加锁 | 最高 |
| READ COMMITTED | 不可能 | 可能 | 可能 | MVCC + 行锁 | 较高 |
| REPEATABLE READ | 不可能 | 不可能 | InnoDB避免 | MVCC + Next-Key Lock | 中等 |
| SERIALIZABLE | 不可能 | 不可能 | 不可能 | 锁表 | 最低 |

### 查看和设置隔离级别

```sql
-- 查看当前会话隔离级别
SELECT @@transaction_isolation;
-- 或旧版语法
SELECT @@tx_isolation;

-- 查看全局隔离级别
SELECT @@global.transaction_isolation;

-- 设置当前会话隔离级别
SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
SET SESSION TRANSACTION ISOLATION LEVEL SERIALIZABLE;

-- 设置全局隔离级别(影响新连接)
SET GLOBAL TRANSACTION ISOLATION LEVEL REPEATABLE READ;

-- 为下一个事务设置隔离级别
SET TRANSACTION ISOLATION LEVEL READ COMMITTED;
START TRANSACTION;
-- 此事务使用READ COMMITTED
COMMIT;
```

**配置文件设置:**

```ini
# my.cnf
[mysqld]
transaction-isolation = REPEATABLE-READ
```

## READ UNCOMMITTED(读未提交)

### 特性说明

READ UNCOMMITTED是最低的隔离级别,允许读取未提交的数据,存在脏读问题。

**特点:**

- 事务可以读取其他事务未提交的修改
- 不加任何读锁
- 性能最高,但数据一致性最差
- 实际生产环境几乎不使用

### 脏读演示

```sql
-- 会话1:修改数据但不提交
SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
START TRANSACTION;

UPDATE accounts SET balance = 1000 WHERE account_id = 'A001';
-- 未提交

-- 会话2:读取未提交的数据(脏读)
SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
START TRANSACTION;

SELECT balance FROM accounts WHERE account_id = 'A001';
-- 结果: 1000 (读取到会话1未提交的修改)

-- 会话1:回滚事务
ROLLBACK;

-- 会话2:再次读取
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 结果: 原始值(如500),会话2读取的1000是无效数据(脏数据)

COMMIT;
```

**实际问题示例:**

```sql
-- 场景:银行转账

-- 会话1(转账事务)
SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
START TRANSACTION;

-- 从A账户扣款
UPDATE accounts SET balance = balance - 1000 WHERE account_id = 'A001';

-- 此时发生错误,准备回滚
-- 但在回滚前...

-- 会话2(查询余额)
SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 读取到扣款后的余额(脏数据)

-- 会话1回滚
ROLLBACK;

-- 会话2读取的数据是错误的,可能导致业务判断失误
```

### 适用场景

```sql
-- 几乎不推荐使用,除非:
-- 1. 对数据一致性要求极低
-- 2. 只读取统计数据,允许一定误差
-- 3. 临时数据分析

-- 示例:粗略的实时统计
SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
SELECT COUNT(*) AS approximate_count FROM large_table;
-- 允许读取未提交的插入,获得大致的实时计数
```

## READ COMMITTED(读已提交)

### 特性说明

READ COMMITTED只允许读取已提交的数据,解决了脏读问题,但存在不可重复读。

**特点:**

- 只能读取已提交的数据
- 每次读取都获取最新的已提交版本
- 使用MVCC实现,读不加锁
- Oracle、SQL Server的默认隔离级别
- MySQL中需要显式设置

### 不可重复读演示

```sql
-- 会话1:多次读取同一数据
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;
START TRANSACTION;

-- 第一次读取
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 结果: 1000

-- 会话2:修改并提交
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;
START TRANSACTION;
UPDATE accounts SET balance = 1500 WHERE account_id = 'A001';
COMMIT;

-- 会话1:再次读取同一数据
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 结果: 1500 (与第一次读取不一致,不可重复读)

COMMIT;
```

**实际问题示例:**

```sql
-- 场景:生成对账报表

-- 会话1(生成报表)
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;
START TRANSACTION;

-- 第一步:查询总收入
SELECT SUM(amount) AS total_income 
FROM transactions 
WHERE type = 'income' AND date = '2024-01-15';
-- 结果: 10000

-- 会话2:新增交易记录
START TRANSACTION;
INSERT INTO transactions (type, amount, date) 
VALUES ('income', 500, '2024-01-15');
COMMIT;

-- 会话1:第二步:查询总支出
SELECT SUM(amount) AS total_expense 
FROM transactions 
WHERE type = 'expense' AND date = '2024-01-15';
-- 结果: 8000

-- 会话1:第三步:计算净收入(再次查询总收入)
SELECT SUM(amount) AS total_income 
FROM transactions 
WHERE type = 'income' AND date = '2024-01-15';
-- 结果: 10500 (包含了会话2新增的500)

-- 报表数据不一致:
-- 第一次查询总收入: 10000
-- 第三次查询总收入: 10500
-- 导致报表前后数据矛盾
```

### MVCC实现原理

```sql
-- READ COMMITTED的MVCC特性
-- 每次SELECT都创建新的ReadView(读视图)

-- 会话1
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;
START TRANSACTION;

-- 时刻T1:创建ReadView1
SELECT * FROM users WHERE user_id = 1;
-- 读取版本V1

-- 会话2修改数据
UPDATE users SET name = 'Bob' WHERE user_id = 1;  -- 创建版本V2
COMMIT;

-- 会话1:时刻T2:创建新的ReadView2
SELECT * FROM users WHERE user_id = 1;
-- 读取版本V2(最新已提交版本)

COMMIT;
```

### 适用场景

```sql
-- READ COMMITTED适合:
-- 1. 不需要可重复读的场景
-- 2. 读取最新已提交数据更重要

-- 示例1:查询最新订单状态
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;
SELECT status FROM orders WHERE order_id = 12345;
-- 总是返回最新的订单状态

-- 示例2:实时数据展示
SELECT COUNT(*) AS current_online_users 
FROM sessions 
WHERE last_active > DATE_SUB(NOW(), INTERVAL 5 MINUTE);
-- 读取最新的在线用户数

-- 示例3:数据导出(不需要一致性快照)
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;
SELECT * FROM large_table INTO OUTFILE '/tmp/export.csv';
```

## REPEATABLE READ(可重复读)

### 特性说明

REPEATABLE READ是MySQL InnoDB的默认隔离级别,保证在同一事务中多次读取同一数据结果一致。

**特点:**

- 事务开始时创建一致性快照
- 同一事务中的读取操作看到相同的数据版本
- InnoDB通过MVCC + Next-Key Lock避免幻读
- 平衡了性能和数据一致性

### 可重复读演示

```sql
-- 会话1:多次读取同一数据
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
START TRANSACTION;

-- 第一次读取
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 结果: 1000

-- 会话2:修改并提交
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
START TRANSACTION;
UPDATE accounts SET balance = 1500 WHERE account_id = 'A001';
COMMIT;

-- 会话1:再次读取
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 结果: 仍然是1000 (可重复读,读取快照版本)

-- 会话1:提交后再读取
COMMIT;
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 结果: 1500 (新事务读取最新版本)
```

### MVCC机制详解

**ReadView创建时机:**

```sql
-- REPEATABLE READ:事务开始时创建ReadView,整个事务使用同一ReadView

-- 会话1
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
START TRANSACTION;

-- 首次SELECT时创建ReadView
SELECT * FROM users WHERE user_id = 1;
-- ReadView记录此刻的活跃事务列表

-- 会话2:创建新版本
START TRANSACTION;
UPDATE users SET age = 30 WHERE user_id = 1;  -- 创建新版本
COMMIT;

-- 会话1:使用相同的ReadView
SELECT * FROM users WHERE user_id = 1;
-- 仍然读取旧版本(根据ReadView判断新版本不可见)

COMMIT;
```

**版本链机制:**

```sql
-- InnoDB为每行记录维护版本链

-- 初始数据
-- user_id=1, name='Alice', age=25, trx_id=100

-- 事务101修改
UPDATE users SET age = 26 WHERE user_id = 1;
-- 版本链: [age=26,trx_id=101] -> [age=25,trx_id=100]

-- 事务102修改
UPDATE users SET age = 27 WHERE user_id = 1;
-- 版本链: [age=27,trx_id=102] -> [age=26,trx_id=101] -> [age=25,trx_id=100]

-- 事务103读取(REPEATABLE READ)
-- ReadView: min_trx_id=101, max_trx_id=103, active_ids=[101,102]
-- 遍历版本链,找到第一个可见版本
-- trx_id=102 在active_ids中,不可见
-- trx_id=101 在active_ids中,不可见
-- trx_id=100 < min_trx_id,可见
-- 读取: age=25
```

### 幻读及其避免

**幻读的定义:**

```sql
-- 幻读:同一查询在不同时刻返回不同的行集

-- 会话1
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
START TRANSACTION;

-- 第一次查询
SELECT * FROM orders WHERE amount > 1000;
-- 结果: 10行

-- 会话2:插入新数据
START TRANSACTION;
INSERT INTO orders (order_id, amount) VALUES (101, 1500);
COMMIT;

-- 会话1:再次查询
SELECT * FROM orders WHERE amount > 1000;
-- 在其他数据库可能返回11行(幻读)
-- 但InnoDB通过MVCC仍返回10行(快照读)

COMMIT;
```

**InnoDB如何避免幻读:**

**方法1: 快照读(MVCC)**

```sql
-- 普通SELECT是快照读,不加锁
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
START TRANSACTION;

SELECT * FROM orders WHERE amount > 1000;
-- 使用MVCC,读取事务开始时的快照,不会看到新插入的行

COMMIT;
```

**方法2: 当前读(Next-Key Lock)**

```sql
-- SELECT ... FOR UPDATE/LOCK IN SHARE MODE是当前读
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
START TRANSACTION;

-- 使用Next-Key Lock锁定范围
SELECT * FROM orders WHERE amount > 1000 FOR UPDATE;
-- 锁定满足条件的现有行 + 间隙(防止插入新行)

-- 会话2尝试插入会被阻塞
-- INSERT INTO orders (order_id, amount) VALUES (101, 1500);
-- 等待锁释放

COMMIT;
```

**Next-Key Lock示例:**

```sql
-- 表数据: id = 1, 5, 10, 15, 20
-- 索引值: (−∞, 1], (1, 5], (5, 10], (10, 15], (15, 20], (20, +∞)

-- 会话1
START TRANSACTION;
SELECT * FROM test WHERE id > 5 AND id <= 15 FOR UPDATE;
-- 加锁范围:
-- 记录锁: id=10, id=15
-- 间隙锁: (5, 10), (10, 15)
-- Next-Key Lock: (5, 10], (10, 15]

-- 会话2:以下操作会被阻塞
INSERT INTO test VALUES (6);   -- 间隙(5, 10)
INSERT INTO test VALUES (7);   -- 间隙(5, 10)
INSERT INTO test VALUES (11);  -- 间隙(10, 15)
UPDATE test SET value = 100 WHERE id = 10;  -- 记录锁
DELETE FROM test WHERE id = 15;  -- 记录锁

-- 会话2:以下操作不会被阻塞
INSERT INTO test VALUES (3);   -- 间隙(1, 5],不在锁范围
INSERT INTO test VALUES (17);  -- 间隙(15, 20],不在锁范围

COMMIT;
```

### 适用场景

```sql
-- REPEATABLE READ适合(推荐默认使用):
-- 1. 需要一致性读的报表生成
-- 2. 复杂的数据处理流程
-- 3. 大部分OLTP场景

-- 示例1:生成对账报表
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
START TRANSACTION;

-- 整个报表生成过程看到一致的数据快照
SELECT SUM(amount) AS total_income FROM transactions WHERE type = 'income';
SELECT SUM(amount) AS total_expense FROM transactions WHERE type = 'expense';
SELECT SUM(balance) AS total_balance FROM accounts;
-- 所有查询基于同一时刻的数据快照

COMMIT;

-- 示例2:数据迁移
START TRANSACTION;
SELECT * FROM old_table;  -- 读取一致的数据快照
-- 处理并插入到新表
INSERT INTO new_table SELECT * FROM old_table WHERE condition;
COMMIT;

-- 示例3:批量更新
START TRANSACTION;
-- 基于一致的读取结果进行更新
SELECT user_id, score FROM user_scores WHERE score > 100;
UPDATE users SET level = 'premium' WHERE user_id IN (...);
COMMIT;
```

## SERIALIZABLE(串行化)

### 特性说明

SERIALIZABLE是最高的隔离级别,通过强制事务串行执行,完全避免并发问题。

**特点:**

- 完全隔离,杜绝脏读、不可重复读、幻读
- 读操作加共享锁,写操作加排他锁
- 性能最低,并发度最差
- 可能导致大量锁等待和死锁

### 串行化演示

```sql
-- 会话1:读取数据
SET SESSION TRANSACTION ISOLATION LEVEL SERIALIZABLE;
START TRANSACTION;

SELECT * FROM accounts WHERE account_id = 'A001';
-- InnoDB加共享锁(S锁)

-- 会话2:尝试修改(被阻塞)
SET SESSION TRANSACTION ISOLATION LEVEL SERIALIZABLE;
START TRANSACTION;
UPDATE accounts SET balance = 2000 WHERE account_id = 'A001';
-- 等待会话1释放共享锁

-- 会话1:提交,释放锁
COMMIT;

-- 会话2:获得锁,执行更新
-- UPDATE成功
COMMIT;
```

**范围查询的锁定:**

```sql
-- 会话1:范围查询
SET SESSION TRANSACTION ISOLATION LEVEL SERIALIZABLE;
START TRANSACTION;

SELECT * FROM orders WHERE amount > 1000;
-- 锁定满足条件的所有行 + 间隙

-- 会话2:无法插入、修改、删除锁定范围内的数据
START TRANSACTION;
INSERT INTO orders (amount) VALUES (1500);  -- 被阻塞
UPDATE orders SET status = 'completed' WHERE amount = 1200;  -- 被阻塞
DELETE FROM orders WHERE amount = 1100;  -- 被阻塞

-- 会话1提交
COMMIT;

-- 会话2的操作才能继续
```

### 死锁风险

```sql
-- SERIALIZABLE容易产生死锁

-- 会话1
SET SESSION TRANSACTION ISOLATION LEVEL SERIALIZABLE;
START TRANSACTION;
SELECT * FROM accounts WHERE account_id = 'A001';  -- 加S锁
-- 准备更新...

-- 会话2
SET SESSION TRANSACTION ISOLATION LEVEL SERIALIZABLE;
START TRANSACTION;
SELECT * FROM accounts WHERE account_id = 'A002';  -- 加S锁
-- 准备更新...

-- 会话1:尝试更新A002
UPDATE accounts SET balance = 1000 WHERE account_id = 'A002';
-- 等待会话2的S锁

-- 会话2:尝试更新A001
UPDATE accounts SET balance = 2000 WHERE account_id = 'A001';
-- 等待会话1的S锁

-- 死锁!MySQL自动检测并回滚其中一个事务
-- ERROR 1213: Deadlock found when trying to get lock
```

### 适用场景

```sql
-- SERIALIZABLE适合:
-- 1. 对数据一致性要求极高的场景
-- 2. 并发度很低的场景
-- 3. 金融系统的关键操作

-- 示例1:银行总账对账
SET SESSION TRANSACTION ISOLATION LEVEL SERIALIZABLE;
START TRANSACTION;

-- 确保对账期间数据完全不变
SELECT SUM(balance) AS total_balance FROM accounts;
SELECT SUM(amount) AS total_transactions FROM transactions;
-- 验证总账平衡

COMMIT;

-- 示例2:库存最终核对
SET SESSION TRANSACTION ISOLATION LEVEL SERIALIZABLE;
START TRANSACTION;

SELECT SUM(stock) AS system_stock FROM products;
SELECT SUM(quantity) AS warehouse_stock FROM warehouse_inventory;
-- 核对系统库存与实际库存

COMMIT;

-- 注意:大部分场景不需要SERIALIZABLE,
-- 使用REPEATABLE READ + 适当的锁就足够了
```

## 隔离级别的选择

### 选择决策树

```
需要读取未提交数据?
  └─ 是 → READ UNCOMMITTED (不推荐)
  └─ 否 → 继续

需要读取最新已提交数据?
  └─ 是 → READ COMMITTED
  └─ 否 → 继续

需要可重复读?
  └─ 是 → REPEATABLE READ (推荐默认)
  └─ 否 → READ COMMITTED

需要完全串行化?
  └─ 是 → SERIALIZABLE (谨慎使用)
  └─ 否 → REPEATABLE READ
```

### 不同场景的推荐

```sql
-- 场景1:一般OLTP应用
-- 推荐: REPEATABLE READ (MySQL默认)
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;

-- 场景2:需要读取最新数据的实时系统
-- 推荐: READ COMMITTED
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;
-- 例如:在线订单状态查询、实时库存查询

-- 场景3:数据仓库、报表系统
-- 推荐: REPEATABLE READ 或 READ COMMITTED
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
-- 长时间运行的报表需要一致性快照

-- 场景4:高并发秒杀系统
-- 推荐: READ COMMITTED + 乐观锁
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;
UPDATE products SET stock = stock - 1 
WHERE product_id = 1001 AND stock > 0;

-- 场景5:关键金融交易
-- 推荐: REPEATABLE READ + SELECT FOR UPDATE
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
START TRANSACTION;
SELECT balance FROM accounts WHERE account_id = 'A001' FOR UPDATE;
-- 业务逻辑
UPDATE accounts SET balance = balance - 100 WHERE account_id = 'A001';
COMMIT;
```

### 性能对比测试

```sql
-- 创建测试表
CREATE TABLE isolation_test (
    id INT PRIMARY KEY AUTO_INCREMENT,
    value INT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 插入测试数据
INSERT INTO isolation_test (value)
SELECT FLOOR(RAND() * 1000)
FROM information_schema.COLUMNS c1, information_schema.COLUMNS c2
LIMIT 100000;

-- 测试不同隔离级别的性能
-- (需要使用压测工具如sysbench进行实际测试)

-- READ UNCOMMITTED: 最高TPS,最低延迟
-- READ COMMITTED: 较高TPS,较低延迟  
-- REPEATABLE READ: 中等TPS,中等延迟 (默认)
-- SERIALIZABLE: 最低TPS,最高延迟

-- 实际性能差异取决于:
-- - 并发度
-- - 事务持续时间
-- - 锁冲突频率
-- - 硬件配置
```

## 锁机制与隔离性

### 锁的类型

**共享锁(S锁):**

```sql
-- 读锁,多个事务可以同时持有
START TRANSACTION;
SELECT * FROM accounts WHERE account_id = 'A001' LOCK IN SHARE MODE;
-- 加S锁,允许其他事务读取,不允许写入

-- 其他事务可以:
SELECT * FROM accounts WHERE account_id = 'A001';  -- 不加锁读取
SELECT * FROM accounts WHERE account_id = 'A001' LOCK IN SHARE MODE;  -- 加S锁

-- 其他事务不能:
UPDATE accounts SET balance = 1000 WHERE account_id = 'A001';  -- 等待

COMMIT;
```

**排他锁(X锁):**

```sql
-- 写锁,独占访问
START TRANSACTION;
SELECT * FROM accounts WHERE account_id = 'A001' FOR UPDATE;
-- 加X锁,其他事务不能读取(加锁读)或写入

-- 其他事务可以:
SELECT * FROM accounts WHERE account_id = 'A001';  -- 快照读,不加锁

-- 其他事务不能:
SELECT * FROM accounts WHERE account_id = 'A001' LOCK IN SHARE MODE;  -- 等待
SELECT * FROM accounts WHERE account_id = 'A001' FOR UPDATE;  -- 等待
UPDATE accounts SET balance = 1000 WHERE account_id = 'A001';  -- 等待

COMMIT;
```

### 锁的粒度

```sql
-- 行级锁(InnoDB)
START TRANSACTION;
UPDATE accounts SET balance = 1000 WHERE account_id = 'A001';
-- 只锁定account_id='A001'这一行

-- 其他行不受影响
UPDATE accounts SET balance = 2000 WHERE account_id = 'A002';  -- 不会等待

COMMIT;

-- 间隙锁(Gap Lock)
START TRANSACTION;
SELECT * FROM orders WHERE id > 10 AND id < 20 FOR UPDATE;
-- 锁定id在(10, 20)范围内的间隙,防止插入

-- 阻塞插入
INSERT INTO orders (id, amount) VALUES (15, 100);  -- 等待

-- 不阻塞范围外的插入
INSERT INTO orders (id, amount) VALUES (5, 100);  -- 成功
INSERT INTO orders (id, amount) VALUES (25, 100);  -- 成功

COMMIT;

-- Next-Key Lock = 记录锁 + 间隙锁
-- 既锁定记录,又锁定间隙
```

### 意向锁

```sql
-- 意向锁是表级锁,表明事务准备在行上加锁

-- 事务1:加行级X锁
START TRANSACTION;
SELECT * FROM accounts WHERE account_id = 'A001' FOR UPDATE;
-- InnoDB自动在表上加意向排他锁(IX)
-- 在行上加排他锁(X)

-- 事务2:尝试加表级锁
LOCK TABLES accounts WRITE;
-- 检查意向锁,发现IX存在,等待

-- 意向锁的作用:
-- 快速判断表中是否有行被锁定,避免全表扫描
```

## 死锁检测与处理

### 死锁产生示例

```sql
-- 经典的死锁场景

-- 会话1
START TRANSACTION;
UPDATE accounts SET balance = balance - 100 WHERE account_id = 'A001';
-- 等待会话2...

-- 会话2
START TRANSACTION;
UPDATE accounts SET balance = balance + 100 WHERE account_id = 'A002';
-- 等待会话1...

-- 会话1继续
UPDATE accounts SET balance = balance + 100 WHERE account_id = 'A002';
-- 等待会话2持有的A002锁

-- 会话2继续
UPDATE accounts SET balance = balance - 100 WHERE account_id = 'A001';
-- 等待会话1持有的A001锁

-- 死锁形成!
-- MySQL检测到死锁,自动回滚其中一个事务
-- ERROR 1213 (40001): Deadlock found when trying to get lock
```

### 死锁检测配置

```sql
-- 查看死锁检测配置
SHOW VARIABLES LIKE 'innodb_deadlock_detect';
-- ON: 启用死锁检测(默认)

-- 查看锁等待超时
SHOW VARIABLES LIKE 'innodb_lock_wait_timeout';
-- 默认50秒

-- 查看最近的死锁信息
SHOW ENGINE INNODB STATUS\G
-- 查看LATEST DETECTED DEADLOCK部分

-- 示例输出:
/*
*** (1) TRANSACTION:
TRANSACTION 1234, ACTIVE 5 sec starting index read
mysql tables in use 1, locked 1
LOCK WAIT 2 lock struct(s), heap size 1136, 1 row lock(s)
UPDATE accounts SET balance = balance + 100 WHERE account_id = 'A002'

*** (2) TRANSACTION:
TRANSACTION 1235, ACTIVE 3 sec starting index read
mysql tables in use 1, locked 1
3 lock struct(s), heap size 1136, 2 row lock(s)
UPDATE accounts SET balance = balance - 100 WHERE account_id = 'A001'

*** WE ROLL BACK TRANSACTION (2)
*/
---
date: 2026-01-15
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 数据库
tag:
  - 数据库
---

# MySQL 的原子性：Undo Log 的完整机制

## 原子性的本质

原子性（Atomicity）是 ACID 中最直观的特性：**事务中的操作要么全做，要么全不做**。

经典场景：转账操作包含两步：
```
步骤1：账户A 减少 100元
步骤2：账户B 增加 100元
```

如果步骤1成功但步骤2失败，钱就凭空消失了。原子性确保：要么两步都执行（转账成功），要么都不执行（转账失败，A的钱退回来）。

但这里有个核心问题：**步骤1已经修改了磁盘上的数据，怎么"退回来"？**

## Undo Log：时光倒流的钥匙

MySQL 通过 Undo Log（回滚日志）实现原子性。核心思想：**在修改数据之前，先记录旧值**。

### Undo Log 的工作流程

```
时刻1: 事务开始
    账户A余额: 1000元

时刻2: 执行UPDATE语句（扣款）
    步骤1: 生成Undo Log
        记录: "账户A的旧值是1000元"
        存储位置: Undo表空间（ibdata或独立undo文件）

    步骤2: 修改数据页
        将账户A余额改为900元
        写入Buffer Pool（内存）

时刻3: 发生错误，需要回滚
    读取Undo Log: "账户A的旧值是1000元"
    将账户A余额恢复为1000元

时刻4: 事务提交（如果没有回滚）
    标记该Undo Log可以清理（不是立即删除）
```

### Undo Log 的存储结构

Undo Log 不是简单的文本记录，而是精心设计的数据结构：

```
Undo Log Record 结构：
┌──────────────────────────────────────────┐
│ Undo Type (操作类型)                       │  INSERT/UPDATE/DELETE
│ Undo No (日志编号)                         │  事务内的序号
│ Table ID (表ID)                           │  哪个表的数据
│ Primary Key (主键值)                       │  哪一行数据
│ Old Values (旧值)                          │  每个修改列的旧值
│ TRX_ID (事务ID)                           │  哪个事务产生的
│ Roll Pointer (回滚指针)                    │  指向更早的版本
└──────────────────────────────────────────┘
```

**关键字段说明**：

**Roll Pointer（回滚指针）**
- 每个 Undo Log 指向该行数据的前一个版本
- 形成一条版本链：当前版本 → 版本1 → 版本2 → ...
- 这个链同时支持 MVCC 的一致性读

**Undo No（日志编号）**
- 同一个事务内的 Undo Log 按执行顺序编号
- 回滚时按**相反顺序**应用（后进先出）

### 不同操作类型的 Undo Log

**INSERT 操作的 Undo**
```
执行: INSERT INTO users (id, name) VALUES (10, 'Alice');

生成的 Undo Log:
    类型: TRX_UNDO_INSERT_REC
    内容: 主键值 = 10
    回滚操作: DELETE FROM users WHERE id = 10;
```

INSERT 的 Undo 很简单，只需记录主键，回滚时删除即可。

**DELETE 操作的 Undo**
```
执行: DELETE FROM users WHERE id = 5;

生成的 Undo Log:
    类型: TRX_UNDO_DEL_MARK_REC
    内容: 完整的行数据（所有列的值）
    回滚操作: 重新插入该行
```

DELETE 不会立即物理删除数据，而是打上"删除标记"（delete mark）。Undo Log 记录完整的行数据，回滚时去掉删除标记。

**UPDATE 操作的 Undo**

UPDATE 是最复杂的，分两种情况：

**情况1：不修改主键**
```
执行: UPDATE users SET age = 30 WHERE id = 5;

生成的 Undo Log:
    类型: TRX_UNDO_UPD_EXIST_REC
    内容: 主键值=5, 旧age值=25
    回滚操作: UPDATE users SET age = 25 WHERE id = 5;
```

**情况2：修改主键**
```
执行: UPDATE users SET id = 15 WHERE id = 5;

等价于:
    DELETE FROM users WHERE id = 5;
    INSERT INTO users VALUES (15, ...);

生成两条 Undo Log:
    1. DELETE的Undo (保存完整行数据)
    2. INSERT的Undo (保存新主键值)
```

为什么修改主键这么麻烦？因为 InnoDB 的数据是按主键组织的聚簇索引，修改主键相当于移动数据在 B+ 树中的位置。

## 回滚的执行机制

### 单语句回滚

```
事务执行：
    UPDATE accounts SET balance = balance - 100 WHERE id = 1;
    → 生成 Undo1: {id=1, 旧balance=1000}

    UPDATE accounts SET balance = balance + 100 WHERE id = 2;
    → 生成 Undo2: {id=2, 旧balance=500}

    -- 发现id=2的账户不存在，操作失败

回滚过程（逆序应用）：
    1. 应用 Undo2: 将id=2的操作撤销（如果有修改的话）
    2. 应用 Undo1: 将id=1的balance恢复为1000
```

**关键点**：Undo Log 必须按**逆序**应用，否则会出错。

举例说明为什么要逆序：
```
操作序列:
    1. UPDATE SET x = x + 1;  (1 → 2)  Undo1: {旧值=1}
    2. UPDATE SET x = x * 2;  (2 → 4)  Undo2: {旧值=2}

正确的回滚（逆序）：
    先应用 Undo2: 恢复为2
    再应用 Undo1: 恢复为1  ← 正确

错误的回滚（顺序）：
    先应用 Undo1: 恢复为1
    再应用 Undo2: 恢复为2  ← 错误！最终应该是1
```

### 事务回滚

用户可以显式回滚整个事务：
```sql
BEGIN;
UPDATE users SET status = 'inactive' WHERE id IN (1,2,3);
DELETE FROM orders WHERE user_id = 1;
INSERT INTO logs VALUES (...);
ROLLBACK;  ← 撤销所有操作
```

回滚流程：
1. 找到该事务的所有 Undo Log（通过事务ID）
2. 按逆序遍历 Undo Log 链表
3. 依次应用每条 Undo Log
4. 释放事务持有的锁

### 崩溃恢复时的回滚

MySQL 崩溃重启后，会进行崩溃恢复：
```
启动时的恢复流程：

1. 扫描 Redo Log
   → 重做已提交但未刷盘的事务

2. 扫描 Undo Log
   → 找到未提交的事务（事务开始了但没有提交记录）

3. 回滚未提交事务
   → 应用这些事务的 Undo Log
   → 确保数据库恢复到一致状态

4. 清理 Undo Log
   → 已回滚的事务的 Undo Log 可以删除
```

这就是为什么 MySQL 崩溃后重启可能需要较长时间——它在回滚所有未完成的事务。

## Undo Log 与 Redo Log 的协作

Undo Log 和 Redo Log 看起来矛盾，实际上是互补的：

**Undo Log**：
- 用途：事务回滚
- 记录内容：数据的旧值
- 何时写入：修改数据**之前**

**Redo Log**：
- 用途：崩溃恢复（持久性）
- 记录内容：数据的新值
- 何时写入：修改数据**之后**

**协作场景：Undo Log 本身也需要持久化**

这里有个有趣的递归问题：
1. 修改数据需要写 Undo Log
2. 写 Undo Log 也是一种"修改"（修改 Undo 表空间）
3. 如何保证 Undo Log 不丢失？答案：**通过 Redo Log**

完整流程：
```
执行 UPDATE accounts SET balance = 900 WHERE id = 1;

步骤1: 生成 Undo Log
    在内存中构造 Undo Log Record

步骤2: 将 Undo Log 写入 Undo 表空间
    修改 Buffer Pool 中的 Undo 页

步骤3: 为 Undo 页生成 Redo Log
    Redo Log 记录: "Undo页X在偏移Y处写入Z"

步骤4: 修改数据页
    修改 Buffer Pool 中的数据页

步骤5: 为数据页生成 Redo Log
    Redo Log 记录: "数据页A在偏移B处写入C"

步骤6: 提交时刷 Redo Log
    将 Redo Log 写入磁盘（fsync）

崩溃恢复时:
    先重放 Redo Log（恢复 Undo Log 和数据页）
    再应用 Undo Log（回滚未提交事务）
```

**关键点**：Undo Log 的写入也被 Redo Log 保护。这确保了崩溃后能够正确回滚。

## Undo Log 的清理机制

Undo Log 不会立即删除，因为：
1. 可能还有其他事务需要读取历史版本（MVCC）
2. 可能还需要回滚

### Purge 线程的工作

MySQL 有专门的 Purge 线程负责清理 Undo Log：

```
判断 Undo Log 是否可以清理：

1. 该事务已经提交
   AND
2. 没有任何活跃事务需要读取该版本
   （所有事务的 Read View 都看不到这个版本了）

如果满足条件:
    物理删除带 delete mark 的行
    清理对应的 Undo Log
```

**为什么长事务会导致 Undo Log 暴涨？**

假设有一个长时间运行的事务（比如一个分析查询，运行了1小时）：
```
T1: 长事务开始，创建 Read View
T2: 其他事务修改了大量数据，生成大量 Undo Log
T3: 其他事务都提交了
T4: Purge 线程想清理 Undo Log
    → 但长事务的 Read View 还能看到旧版本
    → 无法清理，Undo Log 不断堆积
T60: 长事务结束
    → Purge 线程才能清理这1小时累积的 Undo Log
```

这就是为什么要避免长事务——它会：
- 占用大量 Undo 表空间
- 影响 Purge 效率
- 增加 Buffer Pool 的负担

## 原子性的边界情况

### 语句级原子性

即使不显式开启事务，每条 SQL 语句也是原子的：
```sql
UPDATE accounts SET balance = balance - 100 WHERE user_id IN (1,2,3,4,5);
```

如果更新第3个用户时出错，前2个用户的修改也会回滚。这是因为 MySQL 会为每条语句隐式开启事务。

### 多语句事务的原子性

```sql
BEGIN;
UPDATE accounts SET balance = balance - 100 WHERE id = 1;
-- 连接断开，或 MySQL 崩溃
```

这个未提交的事务会被自动回滚（通过 Undo Log）。客户端不需要做任何事情。

### DDL 语句的特殊性

**DDL（数据定义语言）语句不能回滚**：
```sql
BEGIN;
UPDATE users SET age = 30 WHERE id = 1;  ← 可以回滚
ALTER TABLE users ADD COLUMN email VARCHAR(100);  ← 不能回滚
ROLLBACK;  -- UPDATE 会被回滚，但 ALTER TABLE 已经生效
```

为什么？因为 DDL 会隐式提交当前事务（implicit commit）。这是 MySQL 的设计限制。

## 小结

MySQL 通过 Undo Log 实现原子性的核心机制：

**写入时机**：
- 修改数据**之前**先写 Undo Log
- 记录旧值，用于回滚

**Undo Log 类型**：
- INSERT → 记录主键，回滚时删除
- DELETE → 记录完整行，回滚时恢复
- UPDATE → 记录修改列的旧值，回滚时恢复

**回滚机制**：
- 按逆序应用 Undo Log
- 支持语句级、事务级、崩溃恢复的回滚

**与 Redo Log 的协作**：
- Undo Log 的写入也被 Redo Log 保护
- 先 Redo 再 Undo，确保崩溃后能正确恢复

**清理机制**：
- Purge 线程异步清理
- 长事务会阻止 Undo Log 清理，导致空间暴涨

理解 Undo Log 的工作原理，是理解 MySQL 事务机制的关键一步。它不仅实现了原子性，还为 MVCC 的一致性读提供了基础。

## 参考资源

- [MySQL 官方文档 - Undo Log](https://dev.mysql.com/doc/refman/8.0/en/innodb-undo-logs.html)
- [MySQL 官方文档 - 事务回滚](https://dev.mysql.com/doc/refman/8.0/en/innodb-transaction-model.html)
- [MySQL 官方文档 - InnoDB 多版本控制](https://dev.mysql.com/doc/refman/8.0/en/innodb-multi-versioning.html)

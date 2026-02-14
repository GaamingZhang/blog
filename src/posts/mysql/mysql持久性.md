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

# MySQL 的持久性：Redo Log 与崩溃恢复

## 持久性的挑战

持久性（Durability）的核心承诺：**事务一旦提交，数据永不丢失，即使系统崩溃**。

但这里有个根本矛盾：
- 修改数据需要写磁盘（慢）
- 事务提交需要快速响应（快）

如果每次提交都等待数据完全写入磁盘，性能会很差。MySQL 的解决方案是 **WAL（Write-Ahead Logging）**：先写日志，再写数据。

## WAL：先写日志的智慧

### 核心思想

```
传统方式（慢）：
    修改数据 → 等待数据刷盘 → 返回成功
    每次都要等待随机写（慢）

WAL方式（快）：
    修改数据（内存）→ 写Redo Log（顺序写）→ 返回成功
    数据异步刷盘
```

**为什么快？**
- Redo Log 是顺序写，远快于随机写
- Redo Log 很小，写入速度快
- 数据刷盘可以延迟到系统空闲时批量进行

**为什么安全？**
- 崩溃后，Redo Log 还在磁盘上
- 根据 Redo Log 重做丢失的修改
- 数据最终一致

### Redo Log 的完整流程

```
执行：UPDATE users SET age=30 WHERE id=1;

步骤1: 在Buffer Pool中修改数据页
    从磁盘读取包含id=1的数据页到Buffer Pool（如果未加载）
    在内存中修改age字段
    标记该页为"脏页"（dirty page）

步骤2: 生成Redo Log记录
    记录: "在表空间X，页Y，偏移Z，将字节A修改为B"
    写入 Redo Log Buffer（内存）

步骤3: 提交事务
    将 Redo Log Buffer 刷到磁盘（fsync）
    返回"提交成功"
    此时数据页还在内存，未写磁盘

步骤4: 异步刷脏页
    后台线程定期将脏页写入磁盘
    时机：Buffer Pool空间不足、Checkpoint、系统空闲

崩溃恢复：
    如果在步骤4之前崩溃（数据未写盘）
    → 重启时读取Redo Log，重做修改
    → 数据恢复，不丢失
```

### Redo Log 的物理结构

**循环使用的文件组**：
```
ib_logfile0    ib_logfile1
┌────────┐    ┌────────┐
│████████│    │░░░░░░░░│  ← 已使用/未使用
│████████│    │░░░░░░░░│
│████░░░░│    │░░░░░░░░│
│░░░░░░░░│    │░░░░░░░░│
└────────┘    └────────┘
     ↑              ↑
  write pos    checkpoint

write pos: 当前写入位置
checkpoint: 已刷盘的脏页对应的日志位置
```

**工作机制**：
- 新事务在 write pos 位置写入 Redo Log
- 脏页刷盘后，checkpoint 前移
- 如果 write pos 追上 checkpoint → 需要等待（刷脏页）

## innodb_flush_log_at_trx_commit：安全与性能的抉择

这个参数控制 Redo Log 的刷盘策略，直接影响持久性保证。

### 值=1：最安全（默认）

```
每次事务提交：
    Redo Log Buffer → OS Cache → 磁盘（fsync）

时序：
    T1: BEGIN;
    T2: UPDATE ...
    T3: COMMIT;
        → 立即fsync，确保写入磁盘
        → 等待磁盘确认
        → 返回成功

保证：即使下一秒断电，数据也不丢失
代价：TPS受磁盘IOPS限制（每秒几千次）
```

### 值=0：性能最高

```
每秒刷盘一次（后台线程）：
    Redo Log Buffer → OS Cache → 每秒fsync

时序：
    T1: BEGIN;
    T2: UPDATE ...
    T3: COMMIT;
        → 写入Redo Log Buffer（内存）
        → 立即返回成功（无磁盘IO）

    后台线程每秒执行一次fsync

保证：可能丢失最近1秒的事务
代价：断电/崩溃会丢失未刷盘的数据
```

### 值=2：折中方案

```
每次提交写入OS Cache，每秒fsync：
    Redo Log Buffer → OS Cache（立即）
    OS Cache → 磁盘（每秒）

时序：
    T1: BEGIN;
    T2: UPDATE ...
    T3: COMMIT;
        → 写入OS Cache（不调用fsync）
        → 立即返回成功

    后台线程每秒执行一次fsync

保证：MySQL崩溃不丢数据（OS Cache还在）
      操作系统崩溃/断电可能丢失1秒数据
代价：折中的安全性和性能
```

### 三种策略对比

```
策略   MySQL崩溃  OS崩溃/断电   性能    适用场景
─────────────────────────────────────────────────
 =1    不丢失     不丢失       最慢    金融、核心业务
 =0    可能丢失   可能丢失     最快    日志、临时数据
 =2    不丢失     可能丢失     较快    一般业务
```

**性能差异**：
- =1：受磁盘IOPS限制，TPS约几千
- =0：不受磁盘限制，TPS可达几万
- =2：略低于=0，但安全性更好

## 崩溃恢复机制

MySQL 崩溃后重启时，会进行自动恢复。

### 恢复流程

```
步骤1: 扫描Redo Log
    从checkpoint位置开始读取
    找出所有已提交但未刷盘的事务

步骤2: 重做（Redo）
    按Redo Log顺序重新执行修改
    恢复丢失的数据页修改
    此时：已提交事务的数据恢复了

步骤3: 扫描Undo Log
    找出未提交的事务（开始了但没提交）

步骤4: 回滚（Undo）
    应用Undo Log，撤销未提交事务
    此时：数据库恢复到一致状态

步骤5: 清理
    删除已处理的日志
    数据库可以正常服务
```

### 具体示例

```
崩溃前：
    事务A: BEGIN; UPDATE x=1; COMMIT;  ← 已提交，Redo Log已刷盘
    事务B: BEGIN; UPDATE y=2; 未提交  ← 未提交，有Undo Log
    此时崩溃，数据页都未刷盘

崩溃后恢复：
    步骤1: 读取Redo Log
        发现"UPDATE x=1"的日志

    步骤2: 重做
        执行"UPDATE x=1"
        → 事务A的修改恢复

    步骤3: 读取Undo Log
        发现事务B未提交

    步骤4: 回滚
        执行"恢复y的旧值"
        → 事务B的修改被撤销

    结果：
        x=1（事务A已提交，数据保留）
        y=旧值（事务B未提交，被回滚）
```

## Double Write Buffer：防止页损坏

Redo Log 保证了数据不丢失，但还有一个问题：**页部分写入**。

### 页部分写入的问题

InnoDB 的数据页大小是 16KB，但磁盘写入的原子单位通常是 512B 或 4KB。

```
正常写入16KB页：
    [4KB][4KB][4KB][4KB]  ← 完整写入

页损坏（部分写入）：
    [4KB][4KB][✗✗✗][✗✗✗]  ← 写了一半，崩溃

问题：
    页结构破坏，Redo Log无法应用（需要完整的旧页）
    数据无法恢复
```

### Double Write Buffer 的解决方案

在写入数据页之前，先写两次：

```
步骤1: 写入共享表空间的Double Write Buffer区域
    将要刷盘的脏页先写到这个区域（顺序写，快）
    这是一个备份

步骤2: fsync，确保Double Write Buffer落盘

步骤3: 写入实际的数据文件位置
    将脏页写到真正的表空间位置

步骤4: 如果步骤3崩溃
    重启时，从Double Write Buffer恢复完整的页
    然后应用Redo Log
```

**为什么叫"Double Write"？**
因为数据被写了两次：一次在 Double Write Buffer，一次在实际位置。

**代价**：
- 增加了一次写入开销
- 但因为是顺序写，性能影响不大（约5%-10%）

## Binlog：另一层保护

Binlog（Binary Log）是 MySQL Server 层的日志，与 InnoDB 的 Redo Log 互补。

### Binlog vs Redo Log

```
特性          Redo Log               Binlog
─────────────────────────────────────────────────
层次          InnoDB引擎层           MySQL Server层
内容          物理日志（页的修改）    逻辑日志（SQL语句）
用途          崩溃恢复               主从复制、数据恢复
大小          固定，循环覆盖         持续增长，可归档
何时写        事务执行时             事务提交时
```

### 两阶段提交（2PC）

为了保证 Redo Log 和 Binlog 的一致性，MySQL 使用两阶段提交：

```
执行：UPDATE users SET age=30 WHERE id=1;

阶段1: Prepare
    写入Redo Log（标记为prepare状态）
    此时崩溃：事务回滚

阶段2: Commit
    步骤1: 写入Binlog
    步骤2: 提交Redo Log（标记为commit状态）

崩溃恢复的判断：
    如果Redo Log是prepare，但Binlog没有
        → 回滚事务

    如果Redo Log是prepare，Binlog已写入
        → 提交事务（认为用户已收到成功响应）

    如果Redo Log是commit
        → 事务已完成
```

**为什么需要两阶段提交？**

假设没有2PC：
```
场景1: 先写Redo Log，后写Binlog
    Redo Log已写，Binlog未写，崩溃
    → 恢复后：主库有数据，从库没有（主从不一致）

场景2: 先写Binlog，后写Redo Log
    Binlog已写，Redo Log未写，崩溃
    → 恢复后：主库没数据，从库有数据（主从不一致）
```

两阶段提交确保：要么两个日志都有，要么都没有。

### sync_binlog 参数

控制 Binlog 的刷盘策略：

```
=0: 由OS决定何时刷盘（最快，可能丢失）
=1: 每次事务提交都刷盘（最安全，默认）
=N: 每N个事务刷盘一次（折中）
```

**最安全的组合**：
```ini
[mysqld]
innodb_flush_log_at_trx_commit = 1  # Redo Log每次刷盘
sync_binlog = 1                      # Binlog每次刷盘
```

这样可以保证：
- 不丢失已提交事务
- 主从数据一致
- 支持时间点恢复

代价是性能最低，但金融等关键系统必须这样配置。

## 持久性的多层保护

MySQL 的持久性是通过多个机制共同保证的：

**第一层：Redo Log**
- WAL机制，快速提交
- 崩溃恢复的基础

**第二层：Double Write Buffer**
- 防止页损坏
- 保证Redo Log能正确应用

**第三层：Binlog**
- 逻辑备份
- 支持主从复制和时间点恢复

**第四层：定期备份**
- 全量备份+增量备份
- 灾难恢复的最后防线

## 性能调优建议

### 场景1：金融核心系统

```ini
[mysqld]
innodb_flush_log_at_trx_commit = 1
sync_binlog = 1
innodb_flush_method = O_DIRECT  # 绕过OS缓存，避免双重缓存
```

特点：最安全，性能较低

### 场景2：一般业务系统

```ini
[mysqld]
innodb_flush_log_at_trx_commit = 2
sync_binlog = 10
```

特点：折中方案，可能丢失少量数据，但性能好很多

### 场景3：日志/分析系统

```ini
[mysqld]
innodb_flush_log_at_trx_commit = 0
sync_binlog = 0
```

特点：性能最高，但数据可靠性最低，仅适合可重建的数据

### 提升持久性性能的方法

**使用SSD**：
- 大幅降低fsync延迟
- =1的性能接近=2

**增加Redo Log大小**：
- 减少checkpoint频率
- 但恢复时间变长

**批量提交**：
- 应用层攒一批事务再提交
- 减少fsync次数

**使用NVMe**：
- 超低延迟
- 突破IOPS瓶颈

## 小结

MySQL 持久性的核心机制：

**WAL（Write-Ahead Logging）**：
- 先写Redo Log（顺序写），再写数据页（异步）
- 崩溃后通过Redo Log恢复

**关键参数**：
- `innodb_flush_log_at_trx_commit`：控制安全性和性能权衡
- =1最安全，=0最快，=2折中

**Double Write Buffer**：
- 防止页部分写入
- 确保崩溃恢复时有完整的页

**两阶段提交**：
- 保证Redo Log和Binlog一致
- 避免主从数据不一致

**恢复流程**：
- 先Redo（恢复已提交事务）
- 再Undo（回滚未提交事务）
- 自动恢复到一致状态

理解持久性机制，是数据库调优和故障恢复的关键。在实际应用中，需要根据业务特点在安全性和性能之间找到平衡点。

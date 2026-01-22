# MySQL的持久性

## 概述

持久性(Durability)是ACID事务特性中的"D",指的是一旦事务提交,其所做的修改就会永久保存到数据库中。即使系统崩溃、断电或其他故障,已提交的数据也不会丢失。

**持久性的核心保证:**

- **事务提交后数据永久保存**: 不会因系统故障丢失
- **崩溃恢复能力**: 系统重启后能恢复到一致性状态
- **数据完整性**: 已提交的事务修改完整保留
- **可靠性**: 提供多层次的数据保护机制

**持久性的实现机制:**

- **Redo Log(重做日志)**: 记录数据修改,支持崩溃恢复
- **Binlog(二进制日志)**: 记录所有DDL和DML操作,支持主从复制和数据恢复
- **Double Write Buffer**: 防止页损坏
- **刷盘策略**: 控制数据从内存同步到磁盘的时机

## Redo Log(重做日志)

### 基本原理

Redo Log是InnoDB存储引擎的事务日志,采用WAL(Write-Ahead Logging)机制,即先写日志再写数据。

**工作流程:**

```
1. 事务开始,修改Buffer Pool中的数据页
   ↓
2. 将修改操作记录到Redo Log Buffer
   ↓
3. 事务提交时,将Redo Log Buffer刷新到Redo Log File
   ↓
4. 异步将脏页(修改过的数据页)刷新到磁盘
   ↓
5. 如果崩溃,重启时使用Redo Log恢复数据
```

### Redo Log的配置

```sql
-- 查看Redo Log相关配置
SHOW VARIABLES LIKE 'innodb_log%';

-- 重要参数:
-- innodb_log_file_size: 单个redo log文件大小(默认48MB)
-- innodb_log_files_in_group: redo log文件数量(默认2)
-- innodb_log_buffer_size: redo log缓冲区大小(默认16MB)
```

**配置示例(my.cnf):**

```ini
[mysqld]
# Redo Log文件大小(建议512MB-4GB)
innodb_log_file_size = 512M

# Redo Log文件数量(通常2个)
innodb_log_files_in_group = 2

# Redo Log缓冲区大小(默认16MB通常够用)
innodb_log_buffer_size = 16M

# Redo Log刷盘策略(重要!)
innodb_flush_log_at_trx_commit = 1
# 0: 每秒刷盘(性能最高,可能丢失1秒数据)
# 1: 每次事务提交都刷盘(最安全,默认)
# 2: 每次提交写入OS缓存,每秒刷盘(折中)
```

### innodb_flush_log_at_trx_commit详解

这是影响持久性和性能的最关键参数:

**值=1(最安全,默认):**

```sql
-- 每次事务提交都刷盘
SET GLOBAL innodb_flush_log_at_trx_commit = 1;

START TRANSACTION;
INSERT INTO users (username) VALUES ('alice');
COMMIT;
-- COMMIT时:
-- 1. Redo Log写入Redo Log Buffer
-- 2. 调用fsync()将Redo Log刷新到磁盘
-- 3. 返回提交成功
-- 即使立即断电,数据也不会丢失
```

**性能影响:**

```
- 每次提交都需要磁盘IO(fsync)
- TPS受限于磁盘IOPS
- 适合: 金融系统、关键业务数据
```

**值=0(性能最高,风险最大):**

```sql
-- 每秒刷盘一次
SET GLOBAL innodb_flush_log_at_trx_commit = 0;

START TRANSACTION;
INSERT INTO users (username) VALUES ('bob');
COMMIT;
-- COMMIT时:
-- 1. Redo Log写入Redo Log Buffer
-- 2. 立即返回提交成功(未刷盘)
-- 后台线程每秒执行一次fsync()

-- 风险: MySQL崩溃或断电,最多丢失1秒内的事务
```

**性能影响:**

```
- 大幅减少磁盘IO
- TPS可以达到=1时的数倍
- 适合: 日志系统、临时数据、可容忍少量数据丢失的场景
```

**值=2(折中方案):**

```sql
-- 每次提交写入OS缓存,每秒刷盘
SET GLOBAL innodb_flush_log_at_trx_commit = 2;

START TRANSACTION;
INSERT INTO users (username) VALUES ('charlie');
COMMIT;
-- COMMIT时:
-- 1. Redo Log写入Redo Log Buffer
-- 2. 调用write()将数据写入OS文件系统缓存
-- 3. 立即返回提交成功
-- OS每秒将缓存数据fsync()到磁盘

-- 风险: MySQL崩溃不会丢数据,但OS崩溃或断电可能丢失1秒数据
```

**性能影响:**

```
- 性能介于0和1之间
- MySQL崩溃不丢数据,但OS崩溃可能丢失
- 适合: 一般业务系统,平衡性能和安全性
```

### 性能对比测试

```sql
-- 创建测试表
CREATE TABLE perf_test (
    id INT PRIMARY KEY AUTO_INCREMENT,
    data VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 测试脚本(插入10000条记录)
DELIMITER //
CREATE PROCEDURE test_insert(IN p_count INT)
BEGIN
    DECLARE i INT DEFAULT 0;
    WHILE i < p_count DO
        INSERT INTO perf_test (data) VALUES (MD5(RAND()));
        SET i = i + 1;
    END WHILE;
END //
DELIMITER ;

-- 测试innodb_flush_log_at_trx_commit=1
SET GLOBAL innodb_flush_log_at_trx_commit = 1;
CALL test_insert(10000);
-- 耗时: 约30-60秒(取决于磁盘性能)

-- 测试innodb_flush_log_at_trx_commit=2
SET GLOBAL innodb_flush_log_at_trx_commit = 2;
TRUNCATE TABLE perf_test;
CALL test_insert(10000);
-- 耗时: 约10-20秒

-- 测试innodb_flush_log_at_trx_commit=0
SET GLOBAL innodb_flush_log_at_trx_commit = 0;
TRUNCATE TABLE perf_test;
CALL test_insert(10000);
-- 耗时: 约5-10秒

-- 性能差距: 0 > 2 > 1 (可达2-6倍差距)
```

### Redo Log的循环使用

```
Redo Log文件是循环写入的:

[ib_logfile0] → [ib_logfile1] → [ib_logfile0] → ...

写入指针(write pos): 当前写入位置
检查点(checkpoint): 已刷盘的数据页位置

可用空间 = checkpoint到write pos之间的空间

如果write pos追上checkpoint:
- 必须先将脏页刷盘,推进checkpoint
- 这会导致性能抖动
```

**查看Redo Log状态:**

```sql
SHOW ENGINE INNODB STATUS\G

-- 关注LOG部分:
/*
---
LOG
---
Log sequence number          123456789
Log flushed up to            123456789
Pages flushed up to          123456789
Last checkpoint at           123456789
0 pending log flushes, 0 pending chkp writes
*/

-- Log sequence number: 当前LSN(日志序列号)
-- Last checkpoint: 最后检查点位置
-- 如果差距过大,说明有大量脏页待刷盘
```

## Binlog(二进制日志)

### 基本原理

Binlog是MySQL Server层的日志,记录所有修改数据的SQL语句或行变更,用于主从复制和数据恢复。

**Binlog vs Redo Log:**

| 特性     | Binlog                | Redo Log         |
| -------- | --------------------- | ---------------- |
| 层次     | Server层              | InnoDB引擎层     |
| 作用     | 主从复制、数据恢复    | 崩溃恢复         |
| 格式     | STATEMENT/ROW/MIXED   | 物理格式(页修改) |
| 写入方式 | 追加写                | 循环写           |
| 内容     | 逻辑日志(SQL或行变更) | 物理日志(页修改) |

### Binlog的配置

```sql
-- 查看binlog配置
SHOW VARIABLES LIKE 'log_bin%';
SHOW VARIABLES LIKE 'binlog%';
SHOW VARIABLES LIKE 'sync_binlog';

-- 查看binlog文件
SHOW BINARY LOGS;
SHOW MASTER STATUS;
```

**配置示例(my.cnf):**

```ini
[mysqld]
# 启用binlog
log_bin = /var/lib/mysql/mysql-bin

# binlog格式
binlog_format = ROW
# STATEMENT: 记录SQL语句
# ROW: 记录行变更(推荐,最安全)
# MIXED: 混合模式

# binlog刷盘策略
sync_binlog = 1
# 0: 依赖OS刷盘
# 1: 每次事务提交都刷盘(最安全)
# N: 每N个事务刷盘一次

# 单个binlog文件大小
max_binlog_size = 1G

# binlog保留时间(秒)
binlog_expire_logs_seconds = 604800  # 7天

# binlog缓存大小
binlog_cache_size = 32K
```

### Binlog格式对比

**STATEMENT格式:**

```sql
SET SESSION binlog_format = 'STATEMENT';

-- 记录执行的SQL语句
DELETE FROM users WHERE created_at < '2023-01-01';

-- Binlog内容:
-- DELETE FROM users WHERE created_at < '2023-01-01'

-- 优点: binlog文件小
-- 缺点: 
-- - 使用UUID()、NOW()等非确定性函数可能导致主从不一致
-- - 使用LIMIT但没有ORDER BY可能导致主从不一致
```

**ROW格式(推荐):**

```sql
SET SESSION binlog_format = 'ROW';

-- 记录每一行的变更
DELETE FROM users WHERE created_at < '2023-01-01';

-- Binlog内容(示意):
-- DELETE user_id=1, username='alice', ...
-- DELETE user_id=2, username='bob', ...
-- ... (所有被删除的行)

-- 优点: 
-- - 保证主从数据一致
-- - 可以精确恢复每一行的变更
-- 缺点: 
-- - binlog文件较大
-- - 批量操作产生大量binlog
```

**MIXED格式:**

```sql
SET SESSION binlog_format = 'MIXED';

-- MySQL自动选择:
-- - 大部分情况使用STATEMENT
-- - 可能导致不一致时自动切换到ROW

-- 例如:
UPDATE users SET last_login = NOW();  -- 使用ROW
UPDATE users SET status = 'active';   -- 使用STATEMENT
```

### sync_binlog参数详解

**值=1(最安全):**

```sql
SET GLOBAL sync_binlog = 1;

START TRANSACTION;
INSERT INTO orders (amount) VALUES (100);
COMMIT;

-- COMMIT时:
-- 1. 写入binlog cache
-- 2. 刷新binlog cache到binlog文件
-- 3. 调用fsync()同步到磁盘
-- 4. 返回提交成功

-- 保证: 即使断电,binlog不会丢失
```

**值=0(性能最高,风险最大):**

```sql
SET GLOBAL sync_binlog = 0;

-- COMMIT时:
-- 1. 写入binlog cache
-- 2. 刷新到binlog文件的OS缓存
-- 3. 依赖OS刷盘(时机不确定)

-- 风险: OS崩溃或断电可能丢失binlog
```

**值=N(N>1,折中):**

```sql
SET GLOBAL sync_binlog = 10;

-- 每10个事务提交,执行一次fsync()
-- 性能好,但可能丢失最多N个事务的binlog
```

### 双1配置(最安全)

```ini
[mysqld]
# Redo Log每次事务提交都刷盘
innodb_flush_log_at_trx_commit = 1

# Binlog每次事务提交都刷盘
sync_binlog = 1

# 这是最安全的配置,保证数据完全持久化
# 但性能最低,适合金融、核心业务系统
```

**性能优化建议:**

```ini
# 如果可以容忍少量数据丢失,可以调整为:
innodb_flush_log_at_trx_commit = 2
sync_binlog = 10

# 或使用SSD、电池保护的RAID卡:
innodb_flush_log_at_trx_commit = 2
sync_binlog = 100

# 使用组提交(Group Commit)优化:
binlog_group_commit_sync_delay = 100  # 微秒
binlog_group_commit_sync_no_delay_count = 10
# 等待100微秒或10个事务,批量提交
```

## Double Write Buffer

### 工作原理

Double Write Buffer用于防止部分页写入(partial page write)导致的数据页损坏。

**问题场景:**

```
InnoDB数据页大小: 16KB
磁盘扇区大小: 4KB或512字节

写入16KB页时,可能只写入了一部分就断电:
[4KB已写入][4KB已写入][4KB未写入][4KB未写入]

结果: 数据页损坏,既无法恢复旧数据,也无法应用redo log
```

**Double Write流程:**

```
1. 脏页刷盘前,先写入Double Write Buffer(内存)
   ↓
2. Double Write Buffer写入共享表空间的doublewrite区域(顺序写,快)
   ↓
3. 调用fsync()确保doublewrite区域持久化
   ↓
4. 将脏页写入实际的数据文件(随机写)
   ↓
5. 如果步骤4失败(部分写入),可以从doublewrite区域恢复
```

### 配置和监控

```sql
-- 查看Double Write配置
SHOW VARIABLES LIKE 'innodb_doublewrite%';

-- innodb_doublewrite = ON (默认,推荐开启)
```

**配置示例:**

```ini
[mysqld]
# 启用Double Write Buffer(默认ON)
innodb_doublewrite = ON

# 如果使用支持原子写的文件系统(如Fusion-io),可以禁用
# innodb_doublewrite = OFF
```

**监控:**

```sql
SHOW GLOBAL STATUS LIKE 'Innodb_dblwr%';

-- Innodb_dblwr_writes: doublewrite写入次数
-- Innodb_dblwr_pages_written: 写入的页数
-- 比值可以看出每次doublewrite写入多少页
```

**性能影响:**

```
- 写入放大: 每次刷脏页需要写2次
- 但doublewrite区域是顺序写,比随机写快
- 总体性能影响: 约5-10%
- 数据安全收益远大于性能损失
```

## 崩溃恢复机制

### 恢复流程

```
MySQL启动时:

1. 读取redo log,找到最后一个checkpoint
   ↓
2. 从checkpoint开始,重放所有redo log
   ↓
3. 恢复所有已提交事务的修改
   ↓
4. 使用undo log回滚未提交的事务
   ↓
5. 数据库恢复到崩溃前的一致性状态
```

### 恢复示例场景

**场景1: 事务已提交,但脏页未刷盘**

```sql
-- 崩溃前状态
START TRANSACTION;
UPDATE accounts SET balance = 1000 WHERE account_id = 'A001';
COMMIT;  -- redo log已刷盘,但数据页还在Buffer Pool

-- 此时断电崩溃

-- 重启后:
-- 1. 读取redo log,发现UPDATE操作
-- 2. 重放redo log,恢复balance=1000
-- 3. 数据完整恢复(持久性保证)
```

**场景2: 事务未提交**

```sql
-- 崩溃前状态
START TRANSACTION;
UPDATE accounts SET balance = 1000 WHERE account_id = 'A001';
-- 未COMMIT

-- 此时断电崩溃

-- 重启后:
-- 1. 读取undo log,发现未提交的事务
-- 2. 回滚该事务
-- 3. balance恢复到修改前的值
```

**场景3: 部分页写入(使用Double Write)**

```sql
-- 崩溃前状态
-- 脏页刷盘过程中断电,数据页损坏

-- 重启后:
-- 1. 检测到数据页checksum错误
-- 2. 从doublewrite区域读取完整的页
-- 3. 恢复数据页
-- 4. 应用redo log
-- 5. 数据完整恢复
```

### 查看恢复日志

```bash
# MySQL错误日志中会记录恢复过程
cat /var/log/mysql/error.log

# 典型的恢复日志:
# InnoDB: Starting crash recovery
# InnoDB: Last MySQL binlog file position 0 154, file name mysql-bin.000001
# InnoDB: Doing recovery: scanned up to log sequence number 123456789
# InnoDB: Starting an apply batch of log records
# InnoDB: Apply batch completed
# InnoDB: Applying a batch of 0 redo log records
# InnoDB: Rollback of non-prepared transactions completed
# InnoDB: Crash recovery finished
```

## 数据恢复策略

### 物理备份(最快)

```bash
# 使用Percona XtraBackup
# 全量备份
xtrabackup --backup --target-dir=/backup/full

# 增量备份
xtrabackup --backup --target-dir=/backup/inc1 \
  --incremental-basedir=/backup/full

# 恢复
xtrabackup --prepare --target-dir=/backup/full
xtrabackup --copy-back --target-dir=/backup/full
```

### 逻辑备份

```bash
# mysqldump备份
mysqldump -u root -p --single-transaction \
  --master-data=2 --routines --triggers \
  mydb > backup.sql

# 恢复
mysql -u root -p mydb < backup.sql
```

### Binlog恢复

```bash
# 场景:误删除数据,需要恢复到误操作前

# 1. 恢复最近的全量备份
mysql -u root -p mydb < backup.sql

# 2. 应用binlog到误操作前
mysqlbinlog --start-datetime="2024-01-15 00:00:00" \
  --stop-datetime="2024-01-15 14:30:00" \
  mysql-bin.000001 mysql-bin.000002 | mysql -u root -p

# 3. 跳过误操作,继续应用后续binlog
mysqlbinlog --start-datetime="2024-01-15 14:35:00" \
  mysql-bin.000002 | mysql -u root -p
```

### 闪回(Flashback)

```bash
# 使用binlog2sql工具生成回滚SQL
# GitHub: https://github.com/danfengcao/binlog2sql

# 生成回滚SQL
python binlog2sql.py -h127.0.0.1 -P3306 -uroot -p'password' \
  --start-file='mysql-bin.000001' \
  --start-datetime='2024-01-15 14:00:00' \
  --stop-datetime='2024-01-15 15:00:00' \
  -B > rollback.sql

# 执行回滚
mysql -u root -p < rollback.sql
```

## 高可用与持久性

### 主从复制

```sql
-- 主库配置
[mysqld]
server-id = 1
log_bin = mysql-bin
binlog_format = ROW

-- 从库配置
[mysqld]
server-id = 2
relay_log = relay-bin
read_only = 1

-- 配置复制
CHANGE MASTER TO
  MASTER_HOST='master_host',
  MASTER_PORT=3306,
  MASTER_USER='repl',
  MASTER_PASSWORD='password',
  MASTER_LOG_FILE='mysql-bin.000001',
  MASTER_LOG_POS=154;

START SLAVE;
SHOW SLAVE STATUS\G
```

### 半同步复制

```ini
# 主库安装插件
INSTALL PLUGIN rpl_semi_sync_master SONAME 'semisync_master.so';

# 从库安装插件
INSTALL PLUGIN rpl_semi_sync_slave SONAME 'semisync_slave.so';

# 主库配置
[mysqld]
rpl_semi_sync_master_enabled = 1
rpl_semi_sync_master_timeout = 1000  # 1秒超时

# 从库配置
rpl_semi_sync_slave_enabled = 1

# 效果: 至少一个从库确认收到binlog后,主库事务才提交成功
# 保证数据不丢失,但降低性能
```

### MGR(MySQL Group Replication)

```sql
-- 多主复制,Paxos协议保证一致性

-- 配置MGR
[mysqld]
plugin_load_add='group_replication.so'
group_replication_group_name="aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
group_replication_start_on_boot=off
group_replication_local_address="192.168.1.1:33061"
group_replication_group_seeds="192.168.1.1:33061,192.168.1.2:33061,192.168.1.3:33061"

-- 启动MGR
SET GLOBAL group_replication_bootstrap_group=ON;
START GROUP_REPLICATION;
SET GLOBAL group_replication_bootstrap_group=OFF;

-- 优势: 自动故障切换,数据强一致性
```

## 监控与告警

### 关键监控指标

```sql
-- 1. Redo Log使用率
SHOW ENGINE INNODB STATUS\G

-- 关注:
-- Log sequence number与Last checkpoint差距
-- 如果差距过大,说明脏页过多

-- 2. Binlog生成速率
SHOW MASTER STATUS;
-- 定期记录binlog position,计算增长速度

-- 3. 脏页比例
SHOW GLOBAL STATUS LIKE 'Innodb_buffer_pool_pages_dirty';
SHOW GLOBAL STATUS LIKE 'Innodb_buffer_pool_pages_total';

-- 计算脏页比例
SELECT 
  (SELECT VARIABLE_VALUE FROM performance_schema.global_status 
   WHERE VARIABLE_NAME='Innodb_buffer_pool_pages_dirty') /
  (SELECT VARIABLE_VALUE FROM performance_schema.global_status 
   WHERE VARIABLE_NAME='Innodb_buffer_pool_pages_total') * 100 
AS dirty_page_pct;

-- 4. 复制延迟(主从)
SHOW SLAVE STATUS\G
-- Seconds_Behind_Master: 延迟秒数
```

### 告警规则

```sql
-- 建议的告警阈值:

-- 1. 复制延迟 > 60秒
-- Seconds_Behind_Master > 60

-- 2. Binlog磁盘使用率 > 80%
-- du -sh /var/lib/mysql/mysql-bin.*

-- 3. 脏页比例 > 75%
-- dirty_page_pct > 75

-- 4. Redo Log空间不足
-- 检查error log是否有
-- "InnoDB: Waiting for the background threads to finish..."

-- 5. Double Write写入异常
-- Innodb_dblwr_writes增长异常
```

## 常见问题

### 1. innodb_flush_log_at_trx_commit设置为0或2,真的会丢数据吗?如何权衡性能和安全?

**丢数据的场景分析:**

**设置为0:**

```sql
SET GLOBAL innodb_flush_log_at_trx_commit = 0;

START TRANSACTION;
INSERT INTO orders (amount) VALUES (100);
COMMIT;  -- 返回成功

-- COMMIT后的状态:
-- - Redo Log在Redo Log Buffer中(内存)
-- - 后台线程每秒刷盘一次

-- 丢数据场景:
-- 1. MySQL进程崩溃 → 丢失最多1秒的事务
-- 2. 操作系统崩溃 → 丢失最多1秒的事务
-- 3. 断电 → 丢失最多1秒的事务

-- 实际测试:
-- 模拟MySQL崩溃(kill -9 mysql_pid)
-- 重启后检查,最后1秒内提交的事务丢失
```

**设置为2:**

```sql
SET GLOBAL innodb_flush_log_at_trx_commit = 2;

START TRANSACTION;
INSERT INTO orders (amount) VALUES (100);
COMMIT;  -- 返回成功

-- COMMIT后的状态:
-- - Redo Log在OS文件系统缓存中
-- - OS负责刷盘(通常每秒一次)

-- 丢数据场景:
-- 1. MySQL进程崩溃 → 不丢数据(OS缓存保留)
-- 2. 操作系统崩溃 → 丢失OS缓存中的数据(最多1秒)
-- 3. 断电 → 丢失OS缓存中的数据(最多1秒)

-- 实际测试:
-- 模拟MySQL崩溃(kill -9 mysql_pid) → 数据不丢失
-- 模拟断电(立即关机) → 最后1秒的事务丢失
```

**权衡建议:**

| 场景                 | 推荐设置 | 原因                             |
| -------------------- | -------- | -------------------------------- |
| 金融交易系统         | 1        | 绝对不能丢数据                   |
| 电商订单系统         | 1        | 订单数据非常重要                 |
| 一般业务系统         | 2        | 平衡性能和安全,MySQL崩溃不丢数据 |
| 日志系统             | 0或2     | 可容忍少量日志丢失               |
| 缓存/会话数据        | 0        | 临时数据,性能优先                |
| 使用电池保护的RAID卡 | 2        | RAID卡缓存有电池保护             |
| 使用SSD              | 2        | SSD有超级电容保护                |

**组合优化策略:**

```ini
# 策略1: 高性能+基本安全
innodb_flush_log_at_trx_commit = 2
sync_binlog = 10
# MySQL崩溃不丢数据,OS崩溃丢失少量数据

# 策略2: 极致性能(开发/测试环境)
innodb_flush_log_at_trx_commit = 0
sync_binlog = 0
# 性能最高,可能丢失数据

# 策略3: 极致安全(生产环境)
innodb_flush_log_at_trx_commit = 1
sync_binlog = 1
# 双1配置
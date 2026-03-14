---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - MySQL
  - 性能优化
tag:
  - MySQL
  - CPU
  - 性能排查
  - 优化
---

# MySQL数据库CPU利用率高原因排查

## 概述

MySQL数据库CPU利用率过高是运维中常见的问题，可能导致系统响应缓慢甚至服务不可用。本文将系统性地介绍MySQL CPU高利用率的常见原因及排查方法。

## CPU高利用率的常见原因

```
+------------------+----------------------------------------+
|      原因分类    |                具体场景                |
+------------------+----------------------------------------+
| 慢查询           | 全表扫描、复杂JOIN、无索引查询         |
| 锁等待           | 行锁争用、死锁、长事务                 |
| 连接数过多       | 连接池配置不当、连接泄漏               |
| 内存不足         | InnoDB缓冲池不足、临时表过大           |
| 硬件瓶颈         | 磁盘I/O慢、CPU核心数不足               |
| 配置不当         | 参数配置不合理                         |
| 主从延迟         | 从库追赶主库、大量写入                 |
+------------------+----------------------------------------+
```

## 排查流程图

```
                    +------------------+
                    |  CPU利用率告警   |
                    +--------+---------+
                             |
                             v
                    +--------+---------+
                    |  确认MySQL进程   |
                    |  是否为CPU高     |
                    +--------+---------+
                             |
              +--------------+--------------+
              |                             |
              v                             v
     +--------+--------+          +--------+--------+
     | MySQL是CPU高    |          | 其他进程CPU高    |
     +--------+--------+          +--------+--------+
              |                             |
              v                             v
     +--------+--------+          +--------+--------+
     | 查看当前连接     |          | 排查其他服务     |
     | 和正在执行的SQL  |          +-----------------+
     +--------+--------+
              |
              v
     +--------+--------+
     | 分析慢查询日志   |
     +--------+--------+
              |
              v
     +--------+--------+
     | 检查锁等待情况   |
     +--------+--------+
              |
              v
     +--------+--------+
     | 检查系统资源     |
     +--------+--------+
              |
              v
     +--------+--------+
     | 定位问题并优化   |
     +-----------------+
```

## 第一步：确认问题来源

### 查看系统CPU使用情况

```bash
# 使用top查看CPU使用
$ top -c
# 按 P 键按CPU排序

# 使用htop查看详细CPU使用
$ htop

# 使用pidstat查看MySQL进程CPU使用
$ pidstat -p $(pidof mysqld) 1 5
# 01:30:45 PM   UID       PID    %usr %system  %guest    %CPU   CPU  Command
# 01:30:46 PM   999     12345   85.00   15.00    0.00  100.00     1  mysqld

# 查看MySQL线程CPU使用
$ top -H -p $(pidof mysqld)
```

### 确认是否为MySQL导致

```bash
# 查看各进程CPU使用
$ ps aux --sort=-%cpu | head -10

# 如果MySQL进程CPU使用率最高，则继续排查
```

## 第二步：查看当前执行的SQL

### 查看当前连接和执行的SQL

```sql
-- 查看当前所有连接
SHOW PROCESSLIST;

-- 查看完整SQL（需要PROCESS权限）
SHOW FULL PROCESSLIST;

-- 查看正在执行的SQL
SELECT * FROM information_schema.PROCESSLIST 
WHERE COMMAND != 'Sleep' 
ORDER BY TIME DESC;

-- 查看长时间运行的查询
SELECT ID, USER, HOST, DB, TIME, STATE, INFO
FROM information_schema.PROCESSLIST
WHERE TIME > 10
ORDER BY TIME DESC;
```

### 关键字段解读

```
+--------+------------------------------------------+
|  字段  |                 说明                      |
+--------+------------------------------------------+
| ID     | 连接标识符                                |
| USER   | 连接用户                                  |
| HOST   | 客户端主机                                |
| DB     | 当前数据库                                |
| COMMAND| 命令类型（Query/Sleep等）                 |
| TIME   | 当前状态持续时间（秒）                    |
| STATE  | 线程状态                                  |
| INFO   | 执行的SQL语句                             |
+--------+------------------------------------------+
```

### 常见STATE状态

```
+---------------------+----------------------------------+
|       STATE         |               含义               |
+---------------------+----------------------------------+
| Sending data        | 正在发送数据，可能全表扫描       |
| Sorting result      | 正在排序                         |
| Creating tmp table  | 正在创建临时表                   |
| Copying to tmp table| 正在复制到临时表                 |
| Locked              | 被其他查询锁定                   |
| Waiting for lock    | 等待锁释放                       |
| Updating            | 正在更新数据                     |
| Statistics          | 正在计算统计信息                 |
+---------------------+----------------------------------+
```

## 第三步：分析慢查询日志

### 开启慢查询日志

```sql
-- 查看慢查询配置
SHOW VARIABLES LIKE '%slow_query%';
SHOW VARIABLES LIKE '%long_query_time%';

-- 动态开启慢查询日志
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 2;  -- 超过2秒记录
SET GLOBAL log_queries_not_using_indexes = 'ON';  -- 记录无索引查询
```

### 分析慢查询日志

```bash
# 使用mysqldumpslow分析
$ mysqldumpslow -s t -t 10 /var/log/mysql/slow.log

# 参数说明：
# -s t: 按查询时间排序
# -t 10: 显示前10条
# -s c: 按查询次数排序
# -s l: 按锁定时间排序
# -s r: 按返回记录数排序

# 使用pt-query-digest分析（推荐）
$ pt-query-digest /var/log/mysql/slow.log > slow_report.txt
```

### pt-query-digest输出示例

```
# 3600ms user time, 10ms system time, 24.57M rss, 4.17G vsz
# Current date: Mon Mar 11 10:30:00 2026

# Profile
# Rank Query ID           Response time  Calls R/Call  V/M   Item
# ==== ================== ============== ===== ======= ===== ==========
#    1 0x1234567890ABCDEF 1200.00 50.0%   100 12.0000  0.50 SELECT orders
#    2 0xABCDEF1234567890  600.00 25.0%    50 12.0000  0.30 SELECT users
#    3 0xFEDCBA0987654321  300.00 12.5%    30 10.0000  0.20 UPDATE products

# Query 1: 100 QPS, 1.2s concurrency, ID 0x1234567890ABCDEF
# Scores: V/M = 0.50
# Time range: 2026-03-11 10:00:00 to 11:00:00
# Attribute    pct   total     min     max     avg     95%  stddev  median
# ============ === ======= ======= ======= ======= ======= ======= =======
# Count         50     100
# Exec time     50   1200s   100ms    30s     12s     25s      5s     10s
# Lock time     60    500s     1ms    20s      5s     15s      3s      2s
# Rows sent     40  100.00k       0   2.00k   1.00k   1.96k   500.00   1.00k
# Rows examine  80   10.00M   1.00k  500.00k 100.00k 400.00k  50.00k  80.00k

# EXPLAIN
SELECT * FROM orders WHERE status = 'pending' AND create_time > '2026-01-01';
```

## 第四步：检查锁等待情况

### 查看锁等待

```sql
-- MySQL 5.7+
SELECT * FROM sys.innodb_lock_waits;

-- 查看InnoDB锁信息
SELECT * FROM information_schema.INNODB_LOCKS;
SELECT * FROM information_schema.INNODB_LOCK_WAITS;

-- 查看当前事务
SELECT * FROM information_schema.INNODB_TRX;

-- 查看长时间运行的事务
SELECT trx_id, trx_state, trx_started, 
       TIMESTAMPDIFF(SECOND, trx_started, NOW()) as duration,
       trx_mysql_thread_id, trx_query
FROM information_schema.INNODB_TRX
ORDER BY trx_started;
```

### 查看死锁信息

```sql
-- 查看最近一次死锁
SHOW ENGINE INNODB STATUS\G

-- 在输出中查找 LATEST DETECTED DEADLOCK 部分
```

### 锁等待排查SQL

```sql
-- 查看阻塞的会话
SELECT 
    r.trx_id waiting_trx_id,
    r.trx_mysql_thread_id waiting_thread,
    r.trx_query waiting_query,
    b.trx_id blocking_trx_id,
    b.trx_mysql_thread_id blocking_thread,
    b.trx_query blocking_query
FROM information_schema.INNODB_LOCK_WAITS w
INNER JOIN information_schema.INNODB_TRX b ON b.trx_id = w.blocking_trx_id
INNER JOIN information_schema.INNODB_TRX r ON r.trx_id = w.requesting_trx_id;
```

## 第五步：检查系统资源

### 内存使用情况

```sql
-- 查看InnoDB缓冲池使用
SHOW STATUS LIKE 'Innodb_buffer_pool%';

-- 查看缓冲池命中率
SELECT 
    (1 - (Innodb_buffer_pool_reads / Innodb_buffer_pool_read_requests)) * 100 
    AS buffer_pool_hit_rate
FROM (
    SELECT variable_value AS Innodb_buffer_pool_reads
    FROM performance_schema.global_status
    WHERE variable_name = 'Innodb_buffer_pool_reads'
) r,
(
    SELECT variable_value AS Innodb_buffer_pool_read_requests
    FROM performance_schema.global_status
    WHERE variable_name = 'Innodb_buffer_pool_read_requests'
) rr;

-- 命中率应 > 99%
```

```bash
# 查看系统内存
$ free -h

# 查看MySQL内存使用
$ ps aux | grep mysqld | awk '{print $6/1024 "MB"}'
```

### 磁盘I/O情况

```bash
# 查看磁盘I/O
$ iostat -x 1 5

# 查看MySQL数据目录I/O
$ iotop -oP

# 查看磁盘空间
$ df -h
```

### 网络情况

```bash
# 查看网络连接数
$ netstat -antp | grep 3306 | wc -l

# 查看连接状态分布
$ netstat -antp | grep 3306 | awk '{print $6}' | sort | uniq -c
```

## 常见问题场景及解决方案

### 场景一：慢查询导致CPU高

```
问题特征：
- SHOW PROCESSLIST 显示长时间运行的查询
- STATE 为 "Sending data" 或 "Sorting result"
- 慢查询日志有大量记录

排查步骤：
```

```sql
-- 1. 找到慢查询
SELECT ID, TIME, STATE, INFO 
FROM information_schema.PROCESSLIST 
WHERE TIME > 10;

-- 2. 分析执行计划
EXPLAIN SELECT * FROM orders WHERE status = 'pending';

-- 3. 检查索引
SHOW INDEX FROM orders;

-- 4. 创建合适的索引
CREATE INDEX idx_status_createtime ON orders(status, create_time);
```

### 场景二：锁争用导致CPU高

```
问题特征：
- 大量查询 STATE 为 "Waiting for lock"
- 存在长时间运行的事务
- InnoDB锁等待表有记录

排查步骤：
```

```sql
-- 1. 查看锁等待
SELECT * FROM sys.innodb_lock_waits;

-- 2. 找到阻塞的事务
SELECT * FROM information_schema.INNODB_TRX 
ORDER BY trx_started;

-- 3. 终止长时间事务（谨慎操作）
KILL <thread_id>;

-- 4. 优化事务，减少锁持有时间
-- 5. 考虑使用读写分离
```

### 场景三：连接数过多导致CPU高

```
问题特征：
- 大量连接处于活跃状态
- Threads_running 值很高
- 连接创建销毁频繁
```

```sql
-- 查看连接数
SHOW STATUS LIKE 'Threads%';
SHOW STATUS LIKE 'Max_used_connections';
SHOW VARIABLES LIKE 'max_connections';

-- 查看当前连接分布
SELECT USER, HOST, DB, COUNT(*) as cnt
FROM information_schema.PROCESSLIST
GROUP BY USER, HOST, DB
ORDER BY cnt DESC;
```

```bash
# 优化建议：
# 1. 使用连接池
# 2. 调整 max_connections
# 3. 设置 wait_timeout 和 interactive_timeout
# 4. 检查连接泄漏
```

### 场景四：临时表导致CPU高

```
问题特征：
- STATE 为 "Creating tmp table" 或 "Copying to tmp table"
- 大量磁盘临时表
```

```sql
-- 查看临时表使用情况
SHOW STATUS LIKE 'Created_tmp%';

-- 查看临时表配置
SHOW VARIABLES LIKE 'tmp_table_size';
SHOW VARIABLES LIKE 'max_heap_table_size';

-- 优化建议：
-- 1. 增大 tmp_table_size 和 max_heap_table_size
-- 2. 为排序和分组字段添加索引
-- 3. 优化SQL避免创建临时表
```

### 场景五：主从同步追赶导致CPU高

```
问题特征：
- 从库CPU高
- 复制延迟大
- 大量写入操作
```

```sql
-- 查看从库状态
SHOW SLAVE STATUS\G

-- 查看复制延迟
SELECT 
    MASTER_POS_WAIT('mysql-bin.000100', 1000, 5) as delay;
```

```bash
# 优化建议：
# 1. 开启多线程复制
SET GLOBAL slave_parallel_workers = 4;
SET GLOBAL slave_parallel_type = 'LOGICAL_CLOCK';

# 2. 调整复制参数
SET GLOBAL relay_log_recovery = ON;
SET GLOBAL sync_binlog = 0;  # 从库可关闭
```

## 性能优化建议

### SQL优化

```sql
-- 1. 避免SELECT *
SELECT id, name, status FROM users WHERE id = 1;

-- 2. 使用覆盖索引
CREATE INDEX idx_covering ON orders(user_id, status, amount);
SELECT user_id, status, amount FROM orders WHERE user_id = 100;

-- 3. 优化LIKE查询
-- 不推荐
SELECT * FROM users WHERE name LIKE '%张%';
-- 推荐
SELECT * FROM users WHERE name LIKE '张%';

-- 4. 避免在索引列上使用函数
-- 不推荐
SELECT * FROM orders WHERE DATE(create_time) = '2026-03-11';
-- 推荐
SELECT * FROM orders WHERE create_time >= '2026-03-11' AND create_time < '2026-03-12';

-- 5. 使用LIMIT分页优化
-- 不推荐（深度分页）
SELECT * FROM orders LIMIT 100000, 10;
-- 推荐
SELECT * FROM orders WHERE id > 100000 LIMIT 10;
```

### 索引优化

```sql
-- 查看表的索引使用情况
SELECT 
    TABLE_NAME,
    INDEX_NAME,
    CARDINALITY,
    SEQ_IN_INDEX
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = 'your_database'
ORDER BY TABLE_NAME, INDEX_NAME;

-- 分析索引使用效率
SELECT * FROM sys.schema_index_statistics;

-- 查找未使用的索引
SELECT * FROM sys.schema_unused_indexes;
```

### 参数优化

```ini
# /etc/my.cnf

[mysqld]
# InnoDB缓冲池（物理内存的60-80%）
innodb_buffer_pool_size = 8G

# 日志文件大小
innodb_log_file_size = 1G

# 并发线程数
innodb_thread_concurrency = 16

# I/O能力
innodb_io_capacity = 2000
innodb_io_capacity_max = 4000

# 查询缓存（MySQL 8.0已移除）
query_cache_type = 0

# 临时表大小
tmp_table_size = 256M
max_heap_table_size = 256M

# 连接数
max_connections = 1000
back_log = 512

# 超时设置
wait_timeout = 600
interactive_timeout = 600
```

## 监控与告警

### Prometheus监控指标

```yaml
# MySQL CPU相关告警规则
groups:
  - name: mysql_cpu_alerts
    rules:
      - alert: MySQLHighCPU
        expr: |
          rate(process_cpu_seconds_total{job="mysql"}[5m]) > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "MySQL CPU使用率过高"
          description: "MySQL实例 {{ $labels.instance }} CPU使用率超过80%"

      - alert: MySQLTooManySlowQueries
        expr: |
          rate(mysql_global_status_slow_queries[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "MySQL慢查询过多"
          description: "MySQL实例 {{ $labels.instance }} 慢查询速率过高"

      - alert: MySQLTooManyConnections
        expr: |
          mysql_global_status_threads_connected / mysql_global_variables_max_connections > 0.8
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "MySQL连接数过高"
```

### 自动化排查脚本

```bash
#!/bin/bash
# mysql_cpu_check.sh - MySQL CPU高排查脚本

MYSQL_USER="monitor"
MYSQL_PASS="password"
MYSQL_HOST="localhost"

echo "========== MySQL CPU排查报告 =========="
echo "时间: $(date)"
echo ""

echo "========== 当前连接数 =========="
mysql -u$MYSQL_USER -p$MYSQL_PASS -h$MYSQL_HOST -e "
SHOW STATUS LIKE 'Threads_connected';
SHOW STATUS LIKE 'Threads_running';
SHOW STATUS LIKE 'Max_used_connections';
"

echo ""
echo "========== 长时间运行的查询 =========="
mysql -u$MYSQL_USER -p$MYSQL_PASS -h$MYSQL_HOST -e "
SELECT ID, USER, HOST, DB, TIME, STATE, LEFT(INFO, 100) as SQL_PREVIEW
FROM information_schema.PROCESSLIST
WHERE TIME > 5 AND COMMAND != 'Sleep'
ORDER BY TIME DESC
LIMIT 10;
"

echo ""
echo "========== 锁等待情况 =========="
mysql -u$MYSQL_USER -p$MYSQL_PASS -h$MYSQL_HOST -e "
SELECT COUNT(*) as lock_wait_count FROM sys.innodb_lock_waits;
"

echo ""
echo "========== InnoDB状态 =========="
mysql -u$MYSQL_USER -p$MYSQL_PASS -h$MYSQL_HOST -e "
SHOW ENGINE INNODB STATUS\G
" | grep -A 50 "LATEST DETECTED DEADLOCK\|TRANSACTIONS"

echo ""
echo "========== 缓冲池命中率 =========="
mysql -u$MYSQL_USER -p$MYSQL_PASS -h$MYSQL_HOST -e "
SELECT 
    (1 - (SELECT VARIABLE_VALUE FROM performance_schema.global_status 
          WHERE VARIABLE_NAME = 'Innodb_buffer_pool_reads') / 
          (SELECT VARIABLE_VALUE FROM performance_schema.global_status 
           WHERE VARIABLE_NAME = 'Innodb_buffer_pool_read_requests')) * 100 
    AS buffer_pool_hit_rate;
"

echo ""
echo "========== 慢查询统计 =========="
mysql -u$MYSQL_USER -p$MYSQL_PASS -h$MYSQL_HOST -e "
SHOW GLOBAL STATUS LIKE 'Slow_queries';
"
```

## 参考资源

- [MySQL 8.0 Reference Manual - Optimization](https://dev.mysql.com/doc/refman/8.0/en/optimization.html)
- [Percona Toolkit Documentation](https://www.percona.com/doc/percona-toolkit/)
- [MySQL Performance Blog](https://www.percona.com/blog/)
- `man mysqldumpslow`, `man pt-query-digest`

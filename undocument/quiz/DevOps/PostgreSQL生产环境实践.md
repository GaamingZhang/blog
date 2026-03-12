---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Database
tag:
  - PostgreSQL
  - Database
  - Performance
  - DevOps
---

# PostgreSQL 生产环境实践

当你的业务数据量从 GB 级别增长到 TB 级别时,PostgreSQL 的性能优化就不再是锦上添花,而是必须面对的挑战。一个未经优化的 PostgreSQL 实例,可能在百万级数据时就出现查询缓慢、连接池耗尽、磁盘 I/O 瓶颈等问题。而正确的配置和优化,可以让 PostgreSQL 在十亿级数据下依然保持毫秒级响应。

本文将从性能优化策略、集群架构设计、索引优化、查询优化、监控与维护五个维度,系统梳理 PostgreSQL 生产环境的实践经验。

## 一、性能优化策略

### 内存配置优化

PostgreSQL 的内存配置直接影响查询性能,关键参数包括:

```
┌─────────────────────────────────────────────────────────────┐
│                    PostgreSQL 内存架构                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Shared Buffers (共享缓冲区)                          │  │
│  │  - 数据和索引的缓存                                   │  │
│  │  - 默认: 128MB                                        │  │
│  │  - 推荐: 总内存的 25%                                 │  │
│  │  - 最大: 8GB (超过收益递减)                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Work Memory (工作内存)                               │  │
│  │  - 排序、哈希操作的内存                               │  │
│  │  - 默认: 4MB                                          │  │
│  │  - 推荐: 64-256MB                                     │  │
│  │  - 注意: 每个操作独立分配                             │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Maintenance Work Memory (维护内存)                   │  │
│  │  - VACUUM、CREATE INDEX 等操作                        │  │
│  │  - 默认: 64MB                                         │  │
│  │  - 推荐: 512MB-2GB                                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  WAL Buffers (WAL 缓冲区)                             │  │
│  │  - 预写日志缓冲                                       │  │
│  │  - 默认: 自动(Shared Buffers 的 3.125%)               │  │
│  │  - 推荐: 64MB                                         │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**配置示例**:

```sql
-- 查看当前配置
SHOW shared_buffers;
SHOW work_mem;
SHOW maintenance_work_mem;
SHOW effective_cache_size;

-- 修改配置(需要重启)
ALTER SYSTEM SET shared_buffers = '2GB';
ALTER SYSTEM SET work_mem = '256MB';
ALTER SYSTEM SET maintenance_work_mem = '1GB';
ALTER SYSTEM SET effective_cache_size = '6GB';
ALTER SYSTEM SET wal_buffers = '64MB';

-- 重启生效
SELECT pg_reload_conf();
```

**内存配置计算公式**:

```
总内存: 16GB

shared_buffers = 16GB × 25% = 4GB
effective_cache_size = 16GB × 75% = 12GB
work_mem = (16GB - shared_buffers) / max_connections / 4
         = (16GB - 4GB) / 200 / 4 = 15MB
maintenance_work_mem = 1GB
wal_buffers = 64MB
```

### 连接池管理

PostgreSQL 的连接是重量级资源,每个连接消耗约 10MB 内存:

**连接数配置**:

```sql
-- 查看当前连接数
SELECT count(*) FROM pg_stat_activity;

-- 查看最大连接数
SHOW max_connections;

-- 修改最大连接数
ALTER SYSTEM SET max_connections = 200;
```

**使用连接池(PgBouncer)**:

```ini
# pgbouncer.ini
[databases]
production = host=127.0.0.1 port=5432 dbname=production

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 20
min_pool_size = 5
reserve_pool_size = 5
reserve_pool_timeout = 3
max_db_connections = 50
```

**连接池模式对比**:

| 模式 | 说明 | 适用场景 |
|------|------|---------|
| session | 会话级别复用 | 需要会话级特性(SET、PREPARE) |
| transaction | 事务级别复用 | 大多数应用(推荐) |
| statement | 语句级别复用 | 无事务的简单查询 |

### 检查点与 WAL 优化

**检查点配置**:

```sql
-- 查看检查点配置
SHOW checkpoint_timeout;
SHOW max_wal_size;
SHOW min_wal_size;
SHOW checkpoint_completion_target;

-- 优化配置
ALTER SYSTEM SET checkpoint_timeout = '15min';
ALTER SYSTEM SET max_wal_size = '2GB';
ALTER SYSTEM SET min_wal_size = '512MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
```

**WAL 配置**:

```sql
-- WAL 级别
SHOW wal_level;

-- 同步提交
SHOW synchronous_commit;

-- 优化配置
ALTER SYSTEM SET wal_level = replica;
ALTER SYSTEM SET synchronous_commit = on;
ALTER SYSTEM SET wal_compression = on;
```

## 二、集群架构设计

### 主从复制架构

PostgreSQL 支持流复制(Streaming Replication)实现主从同步:

```
┌─────────────────────────────────────────────────────────────┐
│                    PostgreSQL 主从架构                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Primary (主库)                                       │  │
│  │  - 接收所有写入                                       │  │
│  │  - WAL 日志发送                                       │  │
│  │  - 同步/异步复制                                      │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│          ┌──────────┼──────────┐                           │
│          │          │          │                           │
│          ▼          ▼          ▼                           │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐             │
│  │ Standby 1  │ │ Standby 2  │ │ Standby 3  │             │
│  │ (同步)     │ │ (异步)     │ │ (异步)     │             │
│  │ - 只读查询 │ │ - 只读查询 │ │ - 只读查询 │             │
│  │ - 故障转移 │ │ - 负载均衡 │ │ - 负载均衡 │             │
│  └────────────┘ └────────────┘ └────────────┘             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**主库配置**:

```bash
# postgresql.conf
listen_addresses = '*'
wal_level = replica
max_wal_senders = 10
wal_keep_size = 1GB
synchronous_standby_names = 'standby1'
synchronous_commit = on

# pg_hba.conf
host replication replicator 192.168.1.0/24 md5
```

**从库配置**:

```bash
# 基础备份
pg_basebackup -h primary_host -D /var/lib/postgresql/data -U replicator -P -v -R -X stream -C -S standby1

# postgresql.conf
primary_conninfo = 'host=primary_host port=5432 user=replicator password=xxx'
hot_standby = on
hot_standby_feedback = on
```

**监控复制延迟**:

```sql
-- 主库查询
SELECT 
    client_addr,
    state,
    sent_lsn,
    write_lsn,
    flush_lsn,
    replay_lsn,
    pg_wal_lsn_diff(sent_lsn, replay_lsn) as replication_lag
FROM pg_stat_replication;

-- 从库查询
SELECT 
    pg_is_in_recovery(),
    pg_last_wal_receive_lsn(),
    pg_last_wal_replay_lsn(),
    pg_last_xact_replay_timestamp();
```

### 高可用方案:Patroni

Patroni 是 PostgreSQL 的高可用解决方案,基于 etcd 或 Consul 实现自动故障转移:

```yaml
# patroni.yml
scope: postgres-cluster
namespace: /db/
name: node1

restapi:
  listen: 0.0.0.0:8008
  connect_address: 192.168.1.10:8008

etcd:
  hosts: 192.168.1.100:2379,192.168.1.101:2379,192.168.1.102:2379

bootstrap:
  dcs:
    ttl: 30
    loop_wait: 10
    retry_timeout: 10
    maximum_lag_on_failover: 1048576
    postgresql:
      use_pg_rewind: true
      parameters:
        max_connections: 200
        shared_buffers: 2GB
        wal_level: replica
        max_wal_senders: 10
        synchronous_commit: on

  initdb:
    - encoding: UTF8
    - data-checksums

postgresql:
  listen: 0.0.0.0:5432
  connect_address: 192.168.1.10:5432
  data_dir: /var/lib/postgresql/data
  authentication:
    replication:
      username: replicator
      password: xxx
    superuser:
      username: postgres
      password: xxx

tags:
  nofailover: false
  noloadbalance: false
  clonefrom: false
  nosync: false
```

**启动 Patroni**:

```bash
# 启动 Patroni
patroni /etc/patroni/patroni.yml

# 查看集群状态
patronictl -c /etc/patroni/patroni.yml list
```

### 读写分离

**应用层读写分离**:

```python
import psycopg2
from contextlib import contextmanager

class PostgresConnection:
    def __init__(self, primary_config, standby_configs):
        self.primary_config = primary_config
        self.standby_configs = standby_configs
    
    @contextmanager
    def get_write_connection(self):
        """获取写连接(主库)"""
        conn = psycopg2.connect(**self.primary_config)
        try:
            yield conn
        finally:
            conn.close()
    
    @contextmanager
    def get_read_connection(self):
        """获取读连接(从库)"""
        import random
        config = random.choice(self.standby_configs)
        conn = psycopg2.connect(**config)
        try:
            yield conn
        finally:
            conn.close()

# 使用示例
db = PostgresConnection(
    primary_config={'host': 'primary.example.com', 'database': 'production'},
    standby_configs=[
        {'host': 'standby1.example.com', 'database': 'production'},
        {'host': 'standby2.example.com', 'database': 'production'}
    ]
)

# 写操作
with db.get_write_connection() as conn:
    cursor = conn.cursor()
    cursor.execute("INSERT INTO users (name) VALUES (%s)", ('Alice',))
    conn.commit()

# 读操作
with db.get_read_connection() as conn:
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM users")
    results = cursor.fetchall()
```

**PgBouncer 读写分离**:

```ini
# pgbouncer.ini
[databases]
production_write = host=primary.example.com port=5432 dbname=production
production_read = host=standby1.example.com port=5432 dbname=production

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 20
```

## 三、索引优化

### 索引类型选择

PostgreSQL 提供多种索引类型:

| 索引类型 | 适用场景 | 示例 |
|---------|---------|------|
| B-Tree | 等值查询、范围查询、排序 | 默认索引类型 |
| Hash | 等值查询 | `CREATE INDEX ... USING HASH` |
| GiST | 几何数据、全文搜索 | PostGIS、全文索引 |
| GIN | 数组、JSONB、全文搜索 | `CREATE INDEX ... USING GIN` |
| BRIN | 大表、有序数据 | 时间序列数据 |

**索引创建示例**:

```sql
-- B-Tree 索引
CREATE INDEX idx_users_email ON users(email);

-- 复合索引
CREATE INDEX idx_orders_user_date ON orders(user_id, created_at);

-- 部分索引
CREATE INDEX idx_orders_active ON orders(user_id) WHERE status = 'active';

-- 表达式索引
CREATE INDEX idx_users_lower_email ON users(LOWER(email));

-- GIN 索引(JSONB)
CREATE INDEX idx_users_metadata ON users USING GIN(metadata);

-- BRIN 索引(时间序列)
CREATE INDEX idx_logs_timestamp ON logs USING BRIN(timestamp);
```

### 索引使用分析

**查看索引使用情况**:

```sql
-- 查看表的索引
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'users';

-- 查看索引大小
SELECT 
    indexrelname AS index_name,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
    idx_scan AS index_scans,
    idx_tup_read AS tuples_read,
    idx_tup_fetch AS tuples_fetched
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY pg_relation_size(indexrelid) DESC;

-- 查找未使用的索引
SELECT 
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
    idx_scan AS index_scans
FROM pg_stat_user_indexes
WHERE idx_scan = 0
AND indexrelname NOT LIKE '%_pkey'
ORDER BY pg_relation_size(indexrelid) DESC;
```

**索引优化建议**:

```sql
-- 查看缺失索引建议
SELECT 
    schemaname,
    tablename,
    attname,
    n_distinct,
    correlation
FROM pg_stats
WHERE schemaname = 'public'
AND n_distinct > 100
ORDER BY n_distinct DESC;

-- 分析查询计划
EXPLAIN ANALYZE
SELECT * FROM orders WHERE user_id = 123 AND status = 'active';

-- 查看索引建议
SELECT * FROM pg_stat_user_tables
WHERE seq_scan > 0
AND seq_scan > idx_scan
ORDER BY seq_scan DESC;
```

### 索引维护

**重建索引**:

```sql
-- 重建单个索引
REINDEX INDEX idx_users_email;

-- 重建表的所有索引
REINDEX TABLE users;

-- 并发重建(不锁表)
REINDEX INDEX CONCURRENTLY idx_users_email;

-- 检查索引膨胀
SELECT 
    current_database(),
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
JOIN pg_index ON pg_stat_user_indexes.indexrelid = pg_index.indexrelid
WHERE pg_index.indisvalid = false;
```

## 四、查询优化

### 执行计划分析

**EXPLAIN 输出解读**:

```sql
EXPLAIN ANALYZE
SELECT u.name, COUNT(o.id) as order_count
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
WHERE u.created_at > '2025-01-01'
GROUP BY u.id, u.name
HAVING COUNT(o.id) > 10
ORDER BY order_count DESC
LIMIT 100;
```

```
┌─────────────────────────────────────────────────────────────┐
│                    执行计划关键指标                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Scan Type (扫描类型)                                     │
│     - Seq Scan: 全表扫描(慢)                                │
│     - Index Scan: 索引扫描(快)                              │
│     - Index Only Scan: 仅索引扫描(最快)                     │
│     - Bitmap Scan: 位图扫描(多条件)                         │
│                                                              │
│  2. Join Type (连接类型)                                     │
│     - Nested Loop: 嵌套循环(小表驱动大表)                   │
│     - Hash Join: 哈希连接(大表连接)                         │
│     - Merge Join: 合并连接(有序数据)                        │
│                                                              │
│  3. Cost (成本)                                              │
│     - startup cost: 启动成本                                │
│     - total cost: 总成本                                    │
│     - rows: 预估行数                                        │
│     - width: 行宽度(字节)                                   │
│                                                              │
│  4. Actual Time (实际时间)                                   │
│     - actual time: 实际执行时间(毫秒)                       │
│     - actual rows: 实际行数                                 │
│     - actual loops: 循环次数                                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**常见问题诊断**:

```sql
-- 1. 全表扫描
EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'test@example.com';
-- 解决: CREATE INDEX idx_users_email ON users(email);

-- 2. 排序性能差
EXPLAIN ANALYZE SELECT * FROM orders ORDER BY created_at DESC LIMIT 100;
-- 解决: CREATE INDEX idx_orders_created_at ON orders(created_at DESC);

-- 3. 连接性能差
EXPLAIN ANALYZE SELECT * FROM orders o JOIN users u ON o.user_id = u.id;
-- 解决: CREATE INDEX idx_orders_user_id ON orders(user_id);

-- 4. 聚合性能差
EXPLAIN ANALYZE SELECT user_id, COUNT(*) FROM orders GROUP BY user_id;
-- 解决: CREATE INDEX idx_orders_user_id ON orders(user_id);
```

### 查询重写优化

**避免 SELECT ***:

```sql
-- 错误
SELECT * FROM users WHERE id = 123;

-- 正确
SELECT id, name, email FROM users WHERE id = 123;
```

**使用覆盖索引**:

```sql
-- 创建覆盖索引
CREATE INDEX idx_users_email_name ON users(email, name);

-- 使用覆盖索引
SELECT email, name FROM users WHERE email = 'test@example.com';
-- Index Only Scan,无需回表
```

**优化分页查询**:

```sql
-- 错误:大偏移量
SELECT * FROM orders ORDER BY id LIMIT 10 OFFSET 100000;

-- 正确:使用游标
SELECT * FROM orders WHERE id > 100000 ORDER BY id LIMIT 10;

-- 正确:使用子查询
SELECT * FROM orders o
JOIN (SELECT id FROM orders ORDER BY id LIMIT 10 OFFSET 100000) tmp
ON o.id = tmp.id;
```

**优化 COUNT 查询**:

```sql
-- 错误:全表扫描
SELECT COUNT(*) FROM orders WHERE status = 'active';

-- 正确:使用索引
CREATE INDEX idx_orders_status ON orders(status);
SELECT COUNT(*) FROM orders WHERE status = 'active';

-- 更好:使用物化视图
CREATE MATERIALIZED VIEW order_counts AS
SELECT status, COUNT(*) as count
FROM orders
GROUP BY status;

-- 定期刷新
REFRESH MATERIALIZED VIEW order_counts;
```

### 统计信息更新

**手动更新统计信息**:

```sql
-- 更新单个表的统计信息
ANALYZE users;

-- 更新所有表的统计信息
ANALYZE;

-- 更新并设置采样比例
ANALYZE users WITH (default_statistics_target = 1000);

-- 查看统计信息
SELECT 
    tablename,
    attname,
    n_distinct,
    correlation
FROM pg_stats
WHERE tablename = 'users';
```

**自动清理配置**:

```sql
-- 查看自动清理配置
SHOW autovacuum;
SHOW autovacuum_vacuum_threshold;
SHOW autovacuum_analyze_threshold;

-- 优化配置
ALTER SYSTEM SET autovacuum = on;
ALTER SYSTEM SET autovacuum_vacuum_threshold = 50;
ALTER SYSTEM SET autovacuum_analyze_threshold = 50;
ALTER SYSTEM SET autovacuum_vacuum_scale_factor = 0.1;
ALTER SYSTEM SET autovacuum_analyze_scale_factor = 0.05;
```

## 五、监控与维护

### 关键监控指标

**连接监控**:

```sql
-- 当前连接数
SELECT count(*) FROM pg_stat_activity;

-- 连接来源
SELECT 
    usename,
    application_name,
    client_addr,
    state,
    count(*)
FROM pg_stat_activity
GROUP BY usename, application_name, client_addr, state
ORDER BY count(*) DESC;

-- 长时间运行的查询
SELECT 
    pid,
    now() - pg_stat_activity.query_start AS duration,
    query,
    state,
    usename
FROM pg_stat_activity
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes'
AND state = 'active'
ORDER BY duration DESC;
```

**锁监控**:

```sql
-- 当前锁等待
SELECT 
    blocked_locks.pid AS blocked_pid,
    blocked_activity.usename AS blocked_user,
    blocking_locks.pid AS blocking_pid,
    blocking_activity.usename AS blocking_user,
    blocked_activity.query AS blocked_statement,
    blocking_activity.query AS current_statement_in_blocking_process
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
    AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
    AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
    AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
    AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
    AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
    AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
    AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
    AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
    AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
    AND blocking_locks.pid != blocked_locks.pid
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;

-- 终止阻塞查询
SELECT pg_terminate_backend(pid);
```

**表膨胀监控**:

```sql
-- 查看表膨胀
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    n_dead_tup,
    n_live_tup,
    round(100.0 * n_dead_tup / NULLIF(n_live_tup + n_dead_tup, 0), 2) AS dead_ratio
FROM pg_stat_user_tables
WHERE n_dead_tup > 0
ORDER BY n_dead_tup DESC;

-- 手动 VACUUM
VACUUM ANALYZE users;

-- VACUUM FULL(锁表,慎用)
VACUUM FULL users;
```

### 性能监控视图

**pg_stat_statements**:

```sql
-- 启用 pg_stat_statements
CREATE EXTENSION pg_stat_statements;

-- 查看最慢的查询
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    max_time,
    rows
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 10;

-- 查看最频繁的查询
SELECT 
    query,
    calls,
    total_time,
    mean_time
FROM pg_stat_statements
ORDER BY calls DESC
LIMIT 10;

-- 重置统计
SELECT pg_stat_statements_reset();
```

**自定义监控查询**:

```sql
-- 缓存命中率
SELECT 
    sum(heap_blks_read) as heap_read,
    sum(heap_blks_hit) as heap_hit,
    round(100.0 * sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)), 2) as cache_hit_ratio
FROM pg_statio_user_tables;

-- 索引使用率
SELECT 
    sum(idx_scan) as idx_scan,
    sum(seq_scan) as seq_scan,
    round(100.0 * sum(idx_scan) / NULLIF(sum(idx_scan) + sum(seq_scan), 0), 2) as idx_scan_ratio
FROM pg_stat_user_tables;

-- 表大小排行
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
LIMIT 20;
```

### 定期维护任务

**VACUUM 自动化**:

```sql
-- 配置自动清理
ALTER TABLE users SET (
    autovacuum_vacuum_threshold = 1000,
    autovacuum_vacuum_scale_factor = 0.1,
    autovacuum_analyze_threshold = 500,
    autovacuum_analyze_scale_factor = 0.05
);
```

**索引维护脚本**:

```bash
#!/bin/bash
# 定期重建膨胀严重的索引

psql -h localhost -U postgres -d production <<EOF
-- 查找膨胀严重的索引
SELECT 
    indexrelid::regclass AS index_name,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
    idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan > 1000
AND pg_relation_size(indexrelid) > 100 * 1024 * 1024  -- 大于 100MB
ORDER BY pg_relation_size(indexrelid) DESC;

-- 并发重建索引
REINDEX INDEX CONCURRENTLY idx_users_email;
REINDEX INDEX CONCURRENTLY idx_orders_user_id;
EOF
```

**备份策略**:

```bash
# 全量备份
pg_dump -h localhost -U postgres -d production -F c -f production_$(date +%Y%m%d).dump

# 并行备份
pg_dump -h localhost -U postgres -d production -F d -j 4 -f production_backup/

# WAL 归档备份
# postgresql.conf
archive_mode = on
archive_command = 'cp %p /backup/wal/%f'

# PITR 恢复
pg_restore -h localhost -U postgres -d production_restore production_20260311.dump
```

## 小结

- **内存配置**:shared_buffers 设置为总内存的 25%,work_mem 根据连接数合理分配,effective_cache_size 设置为总内存的 75%
- **连接池**:使用 PgBouncer 管理连接池,transaction 模式适合大多数应用,避免连接数过多导致内存耗尽
- **集群架构**:主从复制实现读写分离,Patroni 实现自动故障转移,同步复制保证数据零丢失
- **索引优化**:选择合适的索引类型(B-Tree、GIN、BRIN),使用复合索引优化多条件查询,定期维护索引
- **查询优化**:分析执行计划避免全表扫描,使用覆盖索引减少回表,优化分页和聚合查询
- **监控维护**:监控连接数、锁等待、表膨胀,使用 pg_stat_statements 分析慢查询,定期 VACUUM 和重建索引

---

## 常见问题

### Q1:PostgreSQL 的 VACUUM 有什么作用,如何优化?

**VACUUM 的作用**:

1. **回收空间**:清理被 DELETE 或 UPDATE 标记为死元组的空间
2. **更新统计信息**:ANALYZE 更新表统计信息,帮助查询优化器
3. **防止事务 ID 回卷**:冻结旧事务 ID,避免事务 ID 耗尽

**VACUUM 类型**:

| 类型 | 说明 | 锁级别 |
|------|------|--------|
| VACUUM | 清理死元组,不锁表 | SHARE UPDATE EXCLUSIVE |
| VACUUM FULL | 重建表,回收空间 | ACCESS EXCLUSIVE(锁表) |
| VACUUM ANALYZE | 清理并更新统计信息 | SHARE UPDATE EXCLUSIVE |

**优化配置**:

```sql
-- 自动清理配置
ALTER SYSTEM SET autovacuum = on;
ALTER SYSTEM SET autovacuum_vacuum_threshold = 50;
ALTER SYSTEM SET autovacuum_vacuum_scale_factor = 0.1;
ALTER SYSTEM SET autovacuum_analyze_threshold = 50;
ALTER SYSTEM SET autovacuum_analyze_scale_factor = 0.05;
ALTER SYSTEM SET autovacuum_vacuum_cost_limit = 200;
ALTER SYSTEM SET autovacuum_vacuum_cost_delay = 20;

-- 表级别配置
ALTER TABLE orders SET (
    autovacuum_vacuum_threshold = 1000,
    autovacuum_vacuum_scale_factor = 0.05
);
```

### Q2:如何处理 PostgreSQL 的慢查询?

**诊断步骤**:

```sql
-- 1. 启用慢查询日志
ALTER SYSTEM SET log_min_duration_statement = 1000;  -- 1秒
SELECT pg_reload_conf();

-- 2. 查看慢查询
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    max_time
FROM pg_stat_statements
WHERE mean_time > 1000
ORDER BY total_time DESC;

-- 3. 分析执行计划
EXPLAIN ANALYZE <slow_query>;

-- 4. 检查缺失索引
SELECT 
    schemaname,
    tablename,
    attname
FROM pg_stats
WHERE schemaname = 'public'
AND n_distinct > 100;
```

**优化策略**:

1. **添加索引**:为 WHERE、JOIN、ORDER BY 字段添加索引
2. **重写查询**:避免 SELECT *,优化 JOIN 顺序
3. **更新统计信息**:ANALYZE 表
4. **调整参数**:增加 work_mem、shared_buffers

### Q3:PostgreSQL 如何实现高可用?

**高可用方案对比**:

| 方案 | 优点 | 缺点 | 适用场景 |
|------|------|------|---------|
| 主从复制 | 简单、成熟 | 手动故障转移 | 读多写少 |
| Patroni | 自动故障转移 | 依赖 etcd | 生产环境 |
| PgPool-II | 连接池、负载均衡 | 性能瓶颈 | 中小规模 |
| Stolon | Kubernetes 原生 | 复杂度高 | K8s 环境 |

**Patroni 高可用配置**:

```yaml
# 3 节点 Patroni 集群
# node1: primary
# node2: synchronous standby
# node3: asynchronous standby

bootstrap:
  dcs:
    synchronous_mode: true
    synchronous_standby_names: ['node2']
    postgresql:
      parameters:
        synchronous_commit: on
        synchronous_standby_names: 'node2'
```

**故障转移流程**:

```
1. Primary 故障
2. Patroni 检测到 Primary 不可用
3. 从 Synchronous Standby 中选举新 Primary
4. 更新 etcd 中的集群信息
5. 其他节点连接到新 Primary
6. 应用层通过 HAProxy 或 VIP 自动切换
```

### Q4:PostgreSQL 的大表如何优化?

**分区表**:

```sql
-- 创建分区表
CREATE TABLE orders (
    id BIGSERIAL,
    user_id INTEGER,
    created_at TIMESTAMP,
    amount DECIMAL
) PARTITION BY RANGE (created_at);

-- 创建分区
CREATE TABLE orders_2025_01 PARTITION OF orders
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE orders_2025_02 PARTITION OF orders
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

-- 自动创建分区
CREATE OR REPLACE FUNCTION create_monthly_partition()
RETURNS void AS \$\$
DECLARE
    partition_date DATE;
    partition_name TEXT;
BEGIN
    partition_date := DATE_TRUNC('month', CURRENT_DATE + INTERVAL '1 month');
    partition_name := 'orders_' || TO_CHAR(partition_date, 'YYYY_MM');
    
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF orders 
         FOR VALUES FROM (%L) TO (%L)',
        partition_name,
        partition_date,
        partition_date + INTERVAL '1 month'
    );
END;
\$\$ LANGUAGE plpgsql;
```

**索引优化**:

```sql
-- 使用 BRIN 索引(时间序列)
CREATE INDEX idx_orders_created_at ON orders USING BRIN(created_at);

-- 部分索引
CREATE INDEX idx_orders_active ON orders(user_id) WHERE status = 'active';

-- 并发创建索引
CREATE INDEX CONCURRENTLY idx_orders_user_id ON orders(user_id);
```

### Q5:PostgreSQL 的 JSONB 如何优化?

**GIN 索引**:

```sql
-- 创建 GIN 索引
CREATE INDEX idx_users_metadata ON users USING GIN(metadata);

-- 查询优化
SELECT * FROM users WHERE metadata @> '{"role": "admin"}';
SELECT * FROM users WHERE metadata->>'role' = 'admin';

-- 创建特定路径索引
CREATE INDEX idx_users_metadata_role ON users((metadata->>'role'));
```

**JSONB 查询优化**:

```sql
-- 错误:使用 ->> 导致全表扫描
SELECT * FROM users WHERE metadata->>'role' = 'admin';

-- 正确:使用 @> 操作符利用 GIN 索引
SELECT * FROM users WHERE metadata @> '{"role": "admin"}';

-- 正确:创建表达式索引
CREATE INDEX idx_users_metadata_role ON users((metadata->>'role'));
SELECT * FROM users WHERE metadata->>'role' = 'admin';
```

## 参考资源

- [PostgreSQL 官方文档](https://www.postgresql.org/docs/current/index.html)
- [PostgreSQL 性能优化指南](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [Patroni 文档](https://patroni.readthedocs.io/)
- [PgBouncer 文档](https://www.pgbouncer.org/)
- [PostgreSQL 索引类型](https://www.postgresql.org/docs/current/indexes-types.html)

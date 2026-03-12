---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Database
tag:
  - MySQL
  - Database
  - Performance
  - DevOps
---

# MySQL 生产环境实践

当你的电商平台的订单表突破 1 亿条记录时,MySQL 的性能瓶颈开始显现:慢查询频繁出现、主从延迟增大、连接池耗尽。这些问题并非 MySQL 本身的缺陷,而是缺乏正确的配置和优化。一个经过优化的 MySQL 实例,可以在十亿级数据下保持毫秒级响应,支撑每秒数万次查询。

本文将从性能调优、主从复制、分库分表、慢查询优化、高可用架构五个维度,系统梳理 MySQL 生产环境的实践经验。

## 一、性能调优

### InnoDB 缓冲池配置

InnoDB 缓冲池是 MySQL 性能的核心:

```
┌─────────────────────────────────────────────────────────────┐
│                    InnoDB 缓冲池架构                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Buffer Pool (缓冲池)                                 │  │
│  │  - 数据页缓存                                         │  │
│  │  - 索引页缓存                                         │  │
│  │  - 自适应哈希索引                                     │  │
│  │  - 推荐大小: 物理内存的 70-80%                        │  │
│  │  - 最大: 128TB(MySQL 8.0)                            │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Buffer Pool Instances                                │  │
│  │  - 减少锁竞争                                         │  │
│  │  - 推荐: CPU 核心数                                   │  │
│  │  - 每个实例 ≥ 1GB                                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Log Buffer (日志缓冲区)                              │  │
│  │  - Redo Log 缓冲                                      │  │
│  │  - 默认: 16MB                                         │  │
│  │  - 推荐: 64-256MB                                     │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**配置示例**:

```sql
-- 查看当前配置
SHOW VARIABLES LIKE 'innodb_buffer_pool_size';
SHOW VARIABLES LIKE 'innodb_buffer_pool_instances';
SHOW VARIABLES LIKE 'innodb_log_buffer_size';

-- 修改配置
SET GLOBAL innodb_buffer_pool_size = 8589934592;  -- 8GB
SET GLOBAL innodb_buffer_pool_instances = 8;
SET GLOBAL innodb_log_buffer_size = 67108864;  -- 64MB
```

**配置文件(my.cnf)**:

```ini
[mysqld]
# InnoDB 配置
innodb_buffer_pool_size = 8G
innodb_buffer_pool_instances = 8
innodb_log_buffer_size = 64M
innodb_log_file_size = 1G
innodb_flush_log_at_trx_commit = 1
innodb_flush_method = O_DIRECT

# 连接配置
max_connections = 1000
thread_cache_size = 100
back_log = 512

# 查询缓存(MySQL 8.0 已移除)
query_cache_type = 0
query_cache_size = 0

# 慢查询配置
slow_query_log = 1
long_query_time = 2
slow_query_log_file = /var/log/mysql/slow.log

# 临时表配置
tmp_table_size = 256M
max_heap_table_size = 256M

# 排序缓冲区
sort_buffer_size = 2M
join_buffer_size = 2M
read_buffer_size = 1M
read_rnd_buffer_size = 1M
```

### 连接管理

**连接池配置**:

```python
# Python 连接池示例
import pymysql
from dbutils.pooled_db import PooledDB

db_pool = PooledDB(
    creator=pymysql,
    maxconnections=100,        # 最大连接数
    mincached=10,              # 初始空闲连接
    maxcached=20,              # 最大空闲连接
    maxshared=10,              # 最大共享连接
    blocking=True,             # 连接池耗尽时阻塞
    host='localhost',
    port=3306,
    user='root',
    password='password',
    database='production',
    charset='utf8mb4',
    connect_timeout=10,
    read_timeout=30,
    write_timeout=30
)

def query_db(sql):
    conn = db_pool.connection()
    cursor = conn.cursor()
    cursor.execute(sql)
    result = cursor.fetchall()
    cursor.close()
    conn.close()
    return result
```

**连接监控**:

```sql
-- 查看当前连接数
SHOW STATUS LIKE 'Threads_connected';
SHOW STATUS LIKE 'Max_used_connections';
SHOW VARIABLES LIKE 'max_connections';

-- 查看连接详情
SHOW PROCESSLIST;

-- 查看连接来源
SELECT 
    USER,
    HOST,
    DB,
    COMMAND,
    COUNT(*) as connection_count
FROM information_schema.PROCESSLIST
GROUP BY USER, HOST, DB, COMMAND
ORDER BY connection_count DESC;

-- 终止长时间运行的查询
SELECT 
    ID,
    USER,
    HOST,
    DB,
    TIME,
    STATE,
    INFO
FROM information_schema.PROCESSLIST
WHERE TIME > 60
AND COMMAND != 'Sleep';

-- 终止查询
KILL <process_id>;
```

### 查询优化配置

**关键参数**:

```sql
-- 查看配置
SHOW VARIABLES LIKE 'sort_buffer_size';
SHOW VARIABLES LIKE 'join_buffer_size';
SHOW VARIABLES LIKE 'read_buffer_size';
SHOW VARIABLES LIKE 'read_rnd_buffer_size';

-- 修改配置
SET GLOBAL sort_buffer_size = 2097152;  -- 2MB
SET GLOBAL join_buffer_size = 2097152;  -- 2MB
SET GLOBAL read_buffer_size = 1048576;  -- 1MB
SET GLOBAL read_rnd_buffer_size = 1048576;  -- 1MB
```

**临时表优化**:

```sql
-- 查看临时表使用情况
SHOW STATUS LIKE 'Created_tmp%';

-- 配置临时表大小
SET GLOBAL tmp_table_size = 268435456;  -- 256MB
SET GLOBAL max_heap_table_size = 268435456;  -- 256MB

-- 查看磁盘临时表
SHOW STATUS LIKE 'Created_tmp_disk_tables';
SHOW STATUS LIKE 'Created_tmp_tables';

-- 计算磁盘临时表比例
SELECT 
    (SELECT VARIABLE_VALUE FROM information_schema.GLOBAL_STATUS 
     WHERE VARIABLE_NAME = 'Created_tmp_disk_tables') /
    (SELECT VARIABLE_VALUE FROM information_schema.GLOBAL_STATUS 
     WHERE VARIABLE_NAME = 'Created_tmp_tables') * 100 as disk_tmp_ratio;
```

## 二、主从复制

### 复制架构

MySQL 支持多种复制模式:

```
┌─────────────────────────────────────────────────────────────┐
│                    MySQL 主从复制架构                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Master (主库)                                        │  │
│  │  - 接收所有写入                                       │  │
│  │  - Binlog 记录                                        │  │
│  │  - GTID 或 基于位置复制                               │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│          ┌──────────┼──────────┐                           │
│          │          │          │                           │
│          ▼          ▼          ▼                           │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐             │
│  │ Slave 1    │ │ Slave 2    │ │ Slave 3    │             │
│  │ (异步)     │ │ (半同步)   │ │ (异步)     │             │
│  │ - 只读查询 │ │ - 只读查询 │ │ - 只读查询 │             │
│  │ - 备份     │ │ - 故障转移 │ │ - 报表     │             │
│  └────────────┘ └────────────┘ └────────────┘             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**主库配置**:

```ini
# my.cnf
[mysqld]
server-id = 1
log-bin = mysql-bin
binlog_format = ROW
binlog_row_image = FULL
gtid_mode = ON
enforce_gtid_consistency = ON
sync_binlog = 1
innodb_flush_log_at_trx_commit = 1
```

**从库配置**:

```ini
# my.cnf
[mysqld]
server-id = 2
log-bin = mysql-bin
binlog_format = ROW
gtid_mode = ON
enforce_gtid_consistency = ON
relay-log = relay-bin
read_only = ON
super_read_only = ON
```

**配置复制**:

```sql
-- 主库创建复制用户
CREATE USER 'repl'@'%' IDENTIFIED BY 'password';
GRANT REPLICATION SLAVE ON *.* TO 'repl'@'%';
FLUSH PRIVILEGES;

-- 从库配置复制(GTID)
CHANGE MASTER TO
    MASTER_HOST='master_host',
    MASTER_PORT=3306,
    MASTER_USER='repl',
    MASTER_PASSWORD='password',
    MASTER_AUTO_POSITION=1;

-- 启动复制
START SLAVE;

-- 查看复制状态
SHOW SLAVE STATUS\G
```

### 半同步复制

**配置半同步复制**:

```sql
-- 主库安装插件
INSTALL PLUGIN rpl_semi_sync_master SONAME 'semisync_master.so';
SET GLOBAL rpl_semi_sync_master_enabled = 1;
SET GLOBAL rpl_semi_sync_master_timeout = 1000;  -- 1秒超时

-- 从库安装插件
INSTALL PLUGIN rpl_semi_sync_slave SONAME 'semisync_slave.so';
SET GLOBAL rpl_semi_sync_slave_enabled = 1;

-- 重启复制
STOP SLAVE;
START SLAVE;
```

**监控半同步状态**:

```sql
-- 主库查看
SHOW STATUS LIKE 'Rpl_semi_sync_master%';

-- 从库查看
SHOW STATUS LIKE 'Rpl_semi_sync_slave%';
```

### 复制延迟监控

**监控脚本**:

```sql
-- 从库执行
SELECT 
    MASTER_HOST,
    MASTER_PORT,
    MASTER_USER,
    MASTER_LOG_FILE,
    READ_MASTER_LOG_POS,
    RELAY_MASTER_LOG_FILE,
    EXEC_MASTER_LOG_POS,
    SECONDS_BEHIND_MASTER,
    SLAVE_IO_RUNNING,
    SLAVE_SQL_RUNNING,
    LAST_IO_ERROR,
    LAST_SQL_ERROR
FROM information_schema.GLOBAL_STATUS
WHERE VARIABLE_NAME IN ('SLAVE_IO_RUNNING', 'SLAVE_SQL_RUNNING', 'SECONDS_BEHIND_MASTER');
```

**延迟告警脚本**:

```python
import pymysql
import time

def check_replication_lag():
    conn = pymysql.connect(
        host='slave_host',
        user='monitor',
        password='password',
        database='information_schema'
    )
    
    cursor = conn.cursor()
    cursor.execute("SHOW SLAVE STATUS")
    result = cursor.fetchone()
    
    if result:
        seconds_behind = result[32]  # SECONDS_BEHIND_MASTER
        if seconds_behind and seconds_behind > 30:
            send_alert(f"Replication lag: {seconds_behind} seconds")
    
    cursor.close()
    conn.close()

while True:
    check_replication_lag()
    time.sleep(60)
```

## 三、分库分表

### 分库分表策略

**垂直分库**:

```
┌─────────────────────────────────────────────────────────────┐
│                    垂直分库架构                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  单库架构:                                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Database: production                                 │  │
│  │  - users 表                                          │  │
│  │  - orders 表                                         │  │
│  │  - products 表                                       │  │
│  │  - payments 表                                       │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  垂直分库后:                                                 │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐             │
│  │ user_db    │ │ order_db   │ │ product_db │             │
│  │ - users    │ │ - orders   │ │ - products │             │
│  └────────────┘ └────────────┘ └────────────┘             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**水平分表**:

```
┌─────────────────────────────────────────────────────────────┐
│                    水平分表架构                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  单表: orders (1亿条记录)                                    │
│                                                              │
│  分表后:                                                     │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐             │
│  │ orders_00  │ │ orders_01  │ │ orders_02  │ ...         │
│  │ user_id % 3│ │ user_id % 3│ │ user_id % 3│             │
│  │ = 0        │ │ = 1        │ │ = 2        │             │
│  └────────────┘ └────────────┘ └────────────┘             │
│                                                              │
│  分表键: user_id                                             │
│  分表数量: 3                                                  │
│  路由规则: user_id % 3                                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 分库分表中间件

**ShardingSphere-JDBC**:

```yaml
# application.yml
spring:
  shardingsphere:
    datasource:
      names: ds0,ds1
      ds0:
        type: com.zaxxer.hikari.HikariDataSource
        driver-class-name: com.mysql.cj.jdbc.Driver
        jdbc-url: jdbc:mysql://localhost:3306/db0
        username: root
        password: password
      ds1:
        type: com.zaxxer.hikari.HikariDataSource
        driver-class-name: com.mysql.cj.jdbc.Driver
        jdbc-url: jdbc:mysql://localhost:3306/db1
        username: root
        password: password
    
    rules:
      sharding:
        tables:
          orders:
            actual-data-nodes: ds$->{0..1}.orders_$->{0..1}
            table-strategy:
              standard:
                sharding-column: user_id
                sharding-algorithm-name: orders-inline
            key-generate-strategy:
              column: id
              key-generator-name: snowflake
        
        sharding-algorithms:
          orders-inline:
            type: INLINE
            props:
              algorithm-expression: orders_$->{user_id % 2}
        
        key-generators:
          snowflake:
            type: SNOWFLAKE
            props:
              worker-id: 1
```

**MyCat 配置**:

```xml
<!-- schema.xml -->
<schema name="production" checkSQLschema="false" sqlMaxLimit="100">
    <table name="orders" primaryKey="id" dataNode="dn1,dn2,dn3" rule="mod-long" />
</schema>

<dataNode name="dn1" dataHost="localhost1" database="db1" />
<dataNode name="dn2" dataHost="localhost1" database="db2" />
<dataNode name="dn3" dataHost="localhost1" database="db3" />

<dataHost name="localhost1" maxCon="1000" minCon="10" balance="0"
          dbType="mysql" dbDriver="native" switchType="1" slaveThreshold="100">
    <heartbeat>select user()</heartbeat>
    <writeHost host="hostM1" url="localhost:3306" user="root" password="password">
        <readHost host="hostS1" url="localhost:3307" user="root" password="password" />
    </writeHost>
</dataHost>
```

```xml
<!-- rule.xml -->
<tableRule name="mod-long">
    <rule>
        <columns>user_id</columns>
        <algorithm>mod-long</algorithm>
    </rule>
</tableRule>

<function name="mod-long" class="io.mycat.route.function.PartitionByMod">
    <property name="count">3</property>
</function>
```

### 分库分表后的挑战

**跨库 JOIN**:

```sql
-- 错误:跨库 JOIN
SELECT u.name, o.order_id
FROM db1.users u
JOIN db2.orders o ON u.id = o.user_id;

-- 解决方案 1:应用层聚合
users = query_users(user_ids)
orders = query_orders(user_ids)
result = merge(users, orders)

-- 解决方案 2:冗余数据
-- 在 orders 表中冗余 user_name
SELECT order_id, user_name
FROM orders
WHERE user_id = 123;
```

**分布式事务**:

```java
// 使用 Seata 分布式事务
@GlobalTransactional
public void placeOrder(Order order) {
    // 扣减库存(库存库)
    inventoryService.deductStock(order.getProductId(), order.getQuantity());
    
    // 创建订单(订单库)
    orderService.createOrder(order);
    
    // 扣减余额(用户库)
    userService.deductBalance(order.getUserId(), order.getAmount());
}
```

**全局唯一 ID**:

```java
// Snowflake 算法生成唯一 ID
public class SnowflakeIdGenerator {
    private final long workerId;
    private final long epoch = 1640995200000L;  // 2022-01-01 00:00:00
    private final long workerIdBits = 10L;
    private final long maxWorkerId = -1L ^ (-1L << workerIdBits);
    private final long sequenceBits = 12L;
    private final long workerIdShift = sequenceBits;
    private final long timestampLeftShift = sequenceBits + workerIdBits;
    private final long sequenceMask = -1L ^ (-1L << sequenceBits);
    
    private long sequence = 0L;
    private long lastTimestamp = -1L;
    
    public SnowflakeIdGenerator(long workerId) {
        if (workerId > maxWorkerId || workerId < 0) {
            throw new IllegalArgumentException("worker Id can't be greater than %d or less than 0");
        }
        this.workerId = workerId;
    }
    
    public synchronized long nextId() {
        long timestamp = System.currentTimeMillis();
        
        if (timestamp < lastTimestamp) {
            throw new RuntimeException("Clock moved backwards");
        }
        
        if (lastTimestamp == timestamp) {
            sequence = (sequence + 1) & sequenceMask;
            if (sequence == 0) {
                timestamp = tilNextMillis(lastTimestamp);
            }
        } else {
            sequence = 0L;
        }
        
        lastTimestamp = timestamp;
        
        return ((timestamp - epoch) << timestampLeftShift) |
               (workerId << workerIdShift) |
               sequence;
    }
    
    private long tilNextMillis(long lastTimestamp) {
        long timestamp = System.currentTimeMillis();
        while (timestamp <= lastTimestamp) {
            timestamp = System.currentTimeMillis();
        }
        return timestamp;
    }
}
```

## 四、慢查询优化

### 慢查询分析

**启用慢查询日志**:

```sql
-- 查看配置
SHOW VARIABLES LIKE 'slow_query_log';
SHOW VARIABLES LIKE 'long_query_time';
SHOW VARIABLES LIKE 'slow_query_log_file';

-- 启用慢查询日志
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 2;  -- 2秒
SET GLOBAL log_queries_not_using_indexes = 'ON';

-- 查看慢查询
SELECT * FROM mysql.slow_log
WHERE start_time > DATE_SUB(NOW(), INTERVAL 1 DAY)
ORDER BY query_time DESC
LIMIT 10;
```

**使用 mysqldumpslow 分析**:

```bash
# 分析慢查询日志
mysqldumpslow -s t -t 10 /var/log/mysql/slow.log

# 参数说明
# -s t: 按查询时间排序
# -t 10: 显示前 10 条
# -s c: 按查询次数排序
# -s l: 按锁定时间排序
# -s r: 按返回记录数排序
```

**使用 pt-query-digest 分析**:

```bash
# 安装 Percona Toolkit
yum install percona-toolkit

# 分析慢查询
pt-query-digest /var/log/mysql/slow.log > slow_report.txt

# 分析特定时间段的慢查询
pt-query-digest --since '2026-03-01 00:00:00' --until '2026-03-11 23:59:59' /var/log/mysql/slow.log
```

### 执行计划分析

**EXPLAIN 输出解读**:

```sql
EXPLAIN SELECT * FROM orders WHERE user_id = 123;
```

```
┌─────────────────────────────────────────────────────────────┐
│                    EXPLAIN 关键指标                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. type (访问类型)                                          │
│     - ALL: 全表扫描(最慢)                                   │
│     - index: 索引全扫描                                     │
│     - range: 索引范围扫描                                   │
│     - ref: 非唯一索引查找                                   │
│     - eq_ref: 唯一索引查找                                  │
│     - const: 常量查找(最快)                                 │
│                                                              │
│  2. key (使用的索引)                                         │
│     - NULL: 未使用索引                                      │
│     - PRIMARY: 主键                                         │
│     - idx_name: 指定索引                                    │
│                                                              │
│  3. rows (预估扫描行数)                                      │
│     - 越小越好                                              │
│     - 大于实际返回行数说明索引效率低                         │
│                                                              │
│  4. Extra (额外信息)                                         │
│     - Using index: 覆盖索引                                 │
│     - Using where: WHERE 过滤                               │
│     - Using filesort: 文件排序(慢)                          │
│     - Using temporary: 临时表(慢)                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**常见问题诊断**:

```sql
-- 1. 全表扫描
EXPLAIN SELECT * FROM orders WHERE status = 'active';
-- type: ALL
-- 解决: CREATE INDEX idx_orders_status ON orders(status);

-- 2. 索引失效
EXPLAIN SELECT * FROM orders WHERE YEAR(created_at) = 2026;
-- type: ALL, 索引失效
-- 解决: SELECT * FROM orders WHERE created_at >= '2026-01-01' AND created_at < '2027-01-01';

-- 3. 文件排序
EXPLAIN SELECT * FROM orders ORDER BY created_at DESC LIMIT 100;
-- Extra: Using filesort
-- 解决: CREATE INDEX idx_orders_created_at ON orders(created_at DESC);

-- 4. 临时表
EXPLAIN SELECT user_id, COUNT(*) FROM orders GROUP BY user_id;
-- Extra: Using temporary
-- 解决: CREATE INDEX idx_orders_user_id ON orders(user_id);
```

### 索引优化

**索引设计原则**:

```sql
-- 1. 选择性高的列
SELECT 
    COUNT(DISTINCT user_id) / COUNT(*) as selectivity
FROM orders;
-- selectivity > 0.1 适合建索引

-- 2. 复合索引顺序
-- 最左前缀原则
CREATE INDEX idx_orders_user_status ON orders(user_id, status);
-- 有效: WHERE user_id = 123
-- 有效: WHERE user_id = 123 AND status = 'active'
-- 无效: WHERE status = 'active'

-- 3. 覆盖索引
CREATE INDEX idx_orders_user_status_amount ON orders(user_id, status, amount);
SELECT user_id, status, amount FROM orders WHERE user_id = 123;
-- Extra: Using index, 无需回表

-- 4. 前缀索引
CREATE INDEX idx_users_email ON users(email(20));
-- 适合长字符串列
```

**索引维护**:

```sql
-- 查看索引使用情况
SELECT 
    TABLE_SCHEMA,
    TABLE_NAME,
    INDEX_NAME,
    CARDINALITY,
    SEQ_IN_INDEX
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = 'production'
ORDER BY TABLE_NAME, INDEX_NAME;

-- 查看未使用的索引
SELECT 
    OBJECT_SCHEMA,
    OBJECT_NAME,
    INDEX_NAME
FROM performance_schema.table_io_waits_summary_by_index_usage
WHERE INDEX_NAME IS NOT NULL
AND COUNT_STAR = 0
AND OBJECT_SCHEMA = 'production'
ORDER BY OBJECT_SCHEMA, OBJECT_NAME;

-- 分析表
ANALYZE TABLE orders;

-- 优化表
OPTIMIZE TABLE orders;
```

## 五、高可用架构

### MHA 架构

MHA(Master High Availability)是 MySQL 经典的高可用方案:

```
┌─────────────────────────────────────────────────────────────┐
│                    MHA 架构                                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  MHA Manager                                          │  │
│  │  - 监控 Master 状态                                   │  │
│  │  - 自动故障转移                                       │  │
│  │  - 提升新 Master                                      │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│          ┌──────────┼──────────┐                           │
│          │          │          │                           │
│          ▼          ▼          ▼                           │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐             │
│  │ Master     │ │ Slave 1    │ │ Slave 2    │             │
│  │ (Primary)  │ │ (Candidate)│ │ (Candidate)│             │
│  └────────────┘ └────────────┘ └────────────┘             │
│                                                              │
│  故障转移流程:                                               │
│  1. MHA Manager 检测到 Master 故障                          │
│  2. 从 Slave 中选择数据最新的作为新 Master                   │
│  3. 其他 Slave 同步到新 Master                              │
│  4. VIP 漂移到新 Master                                     │
│  5. 应用连接到新 Master                                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**MHA 配置**:

```ini
# /etc/masterha/app1.cnf
[server default]
manager_workdir=/var/log/masterha/app1
manager_log=/var/log/masterha/app1/manager.log
user=mha
password=mha_password
ssh_user=root
repl_user=repl
repl_password=repl_password
ping_interval=3
secondary_check_script=/usr/local/bin/masterha_secondary_check -s slave1 -s slave2
master_ip_failover_script=/usr/local/bin/master_ip_failover
shutdown_script=/usr/local/bin/power_manager

[server1]
hostname=master
candidate_master=1

[server2]
hostname=slave1
candidate_master=1

[server3]
hostname=slave2
candidate_master=0
```

**启动 MHA**:

```bash
# 检查 SSH 连接
masterha_check_ssh --conf=/etc/masterha/app1.cnf

# 检查复制状态
masterha_check_repl --conf=/etc/masterha/app1.cnf

# 启动 MHA Manager
nohup masterha_manager --conf=/etc/masterha/app1.cnf > /var/log/masterha/app1/manager.log 2>&1 &

# 检查 MHA 状态
masterha_check_status --conf=/etc/masterha/app1.cnf
```

### Orchestrator 架构

Orchestrator 是现代化的 MySQL 高可用工具:

```json
{
  "Debug": true,
  "ListenAddress": ":3000",
  "MySQLTopologyUser": "orchestrator",
  "MySQLTopologyPassword": "orchestrator_password",
  "MySQLTopologyCredentialsConfigFile": "",
  "MySQLTopologySSLPrivateKeyFile": "",
  "MySQLTopologySSLCertFile": "",
  "MySQLTopologySSLCAFile": "",
  "MySQLTopologySSLSkipVerify": true,
  "MySQLTopologyUseMutualTLS": false,
  "MySQLOrchestratorHost": "127.0.0.1",
  "MySQLOrchestratorPort": 3306,
  "MySQLOrchestratorDatabase": "orchestrator",
  "MySQLOrchestratorUser": "orchestrator",
  "MySQLOrchestratorPassword": "orchestrator_password",
  "MySQLConnectTimeoutSeconds": 1,
  "MySQLDiscoveryReadTimeoutSeconds": 3,
  "MySQLDiscoveryTimeoutSeconds": 3,
  "MySQLTopologyReadTimeoutSeconds": 3,
  "MySQLTopologyReadTimeoutSeconds": 3,
  "RecoveryPeriodBlockSeconds": 3600,
  "RecoveryIgnoreHostnameFilters": [],
  "RecoverMasterClusterFilters": ["*"],
  "RecoverIntermediateMasterClusterFilters": ["*"],
  "OnFailureDetectionProcesses": [
    "echo 'Detected {failureType} on {failureCluster}. Affected replicas: {countSlaves}' >> /tmp/recovery.log"
  ],
  "PreFailoverProcesses": [
    "echo 'Will recover from {failureType} on {failureCluster}' >> /tmp/recovery.log"
  ],
  "PostFailoverProcesses": [
    "echo '(for all types) Recovered from {failureType} on {failureCluster}. Failed: {failedHost}:{failedPort}; Promoted: {successorHost}:{successorPort}' >> /tmp/recovery.log"
  ],
  "PostUnsuccessfulFailoverProcesses": [],
  "PostMasterFailoverProcesses": [
    "echo 'Recovered from {failureType} on {failureCluster}. Failed: {failedHost}:{failedPort}; Promoted: {successorHost}:{successorPort}' >> /tmp/recovery.log"
  ],
  "PostIntermediateMasterFailoverProcesses": [
    "echo 'Recovered from {failureType} on {failureCluster}. Failed: {failedHost}:{failedPort}; Successor: {successorHost}:{successorPort}' >> /tmp/recovery.log"
  ],
  "CoMasterRecoveryMustPromoteOtherCoMaster": true,
  "DetachLostSlavesAfterMasterFailover": true,
  "ApplyMySQLPromotionAfterMasterFailover": true,
  "PreventCrossDataCenterMasterFailover": false,
  "MasterFailoverLostInstancesDowntimeMinutes": 10,
  "PostponeReplicaRecoveryOnLagMinutes": 0
}
```

**启动 Orchestrator**:

```bash
# 启动 Orchestrator
orchestrator --config=/etc/orchestrator.conf.json http

# 访问 Web UI
http://localhost:3000

# CLI 命令
orchestrator -c discover -i master:3306
orchestrator -c topology -i master:3306
orchestrator -c reset-replica -i slave1:3306
```

## 小结

- **性能调优**:InnoDB 缓冲池设置为物理内存的 70-80%,合理配置连接池和查询缓冲区,监控临时表和排序性能
- **主从复制**:GTID 复制简化管理,半同步复制保证数据安全,监控复制延迟及时告警
- **分库分表**:垂直分库按业务拆分,水平分表按分片键拆分,使用中间件(ShardingSphere、MyCat)简化开发
- **慢查询优化**:启用慢查询日志定位问题,分析执行计划优化索引,遵循索引设计原则
- **高可用架构**:MHA 经典可靠,Orchestrator 现代化易用,选择合适的方案保证业务连续性

---

## 常见问题

### Q1:MySQL 的 InnoDB 和 MyISAM 有什么区别?

**对比**:

| 特性 | InnoDB | MyISAM |
|------|--------|--------|
| 事务支持 | 支持 | 不支持 |
| 锁粒度 | 行锁 | 表锁 |
| 外键 | 支持 | 不支持 |
| 崩溃恢复 | 自动恢复 | 需手动修复 |
| 全文索引 | MySQL 5.6+ 支持 | 支持 |
| 缓存 | 缓存数据和索引 | 只缓存索引 |
| 适用场景 | OLTP、高并发 | OLAP、只读 |

**推荐**:生产环境优先使用 InnoDB,除非有特殊需求(如全文索引在 MySQL 5.5 及以下版本)。

### Q2:如何处理 MySQL 的主从延迟?

**原因**:

1. 主库写入压力大,从库跟不上
2. 从库硬件配置低
3. 大事务导致延迟
4. 网络延迟

**解决方案**:

```sql
-- 1. 并行复制(MySQL 5.7+)
-- 从库配置
slave_parallel_type = LOGICAL_CLOCK
slave_parallel_workers = 8

-- 2. 半同步复制
-- 主库配置
rpl_semi_sync_master_enabled = 1
rpl_semi_sync_master_timeout = 1000

-- 3. 读写分离
-- 写操作走主库,读操作走从库
-- 关键读操作走主库

-- 4. 分库分表
-- 减少单库写入压力
```

### Q3:MySQL 如何避免死锁?

**死锁原因**:

```sql
-- 事务 1
UPDATE orders SET status = 'paid' WHERE id = 1;
UPDATE orders SET status = 'paid' WHERE id = 2;

-- 事务 2
UPDATE orders SET status = 'paid' WHERE id = 2;
UPDATE orders SET status = 'paid' WHERE id = 1;
-- 死锁:事务 1 等待事务 2 释放 id=2 的锁,事务 2 等待事务 1 释放 id=1 的锁
```

**避免死锁**:

```sql
-- 1. 按相同顺序访问表
-- 事务 1 和事务 2 都按 id 升序访问
UPDATE orders SET status = 'paid' WHERE id = 1;
UPDATE orders SET status = 'paid' WHERE id = 2;

-- 2. 减少事务持有锁的时间
-- 事务开始 → 获取锁 → 执行操作 → 提交事务

-- 3. 使用乐观锁
UPDATE orders SET status = 'paid', version = version + 1 
WHERE id = 1 AND version = 10;

-- 4. 设置锁等待超时
SET innodb_lock_wait_timeout = 50;  -- 50秒
```

### Q4:MySQL 的 Binlog 有什么作用?

**Binlog 作用**:

1. **主从复制**:从库通过 Binlog 同步数据
2. **数据恢复**:基于时间点恢复数据
3. **审计**:记录所有数据变更

**Binlog 格式**:

| 格式 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| STATEMENT | 记录 SQL 语句 | 日志小 | 不确定函数导致不一致 |
| ROW | 记录行变更 | 数据一致 | 日志大 |
| MIXED | 混合模式 | 兼顾大小和一致性 | 复杂 |

**推荐**:生产环境使用 ROW 格式,保证数据一致性。

```sql
-- 查看配置
SHOW VARIABLES LIKE 'binlog_format';

-- 修改配置
SET GLOBAL binlog_format = 'ROW';
```

### Q5:MySQL 如何进行数据备份和恢复?

**备份方式**:

```bash
# 1. 逻辑备份(mysqldump)
mysqldump -h localhost -u root -p \
  --single-transaction \
  --routines \
  --triggers \
  --events \
  --all-databases \
  > backup_$(date +%Y%m%d).sql

# 2. 物理备份(Percona XtraBackup)
xtrabackup --backup --target-dir=/backup/full

# 3. 增量备份
xtrabackup --backup --target-dir=/backup/inc1 \
  --incremental-basedir=/backup/full

# 4. Binlog 备份
mysqlbinlog --read-from-remote-server \
  --host=localhost \
  --raw \
  mysql-bin.000001
```

**恢复方式**:

```bash
# 1. 恢复逻辑备份
mysql -h localhost -u root -p < backup_20260311.sql

# 2. 恢复物理备份
xtrabackup --prepare --target-dir=/backup/full
xtrabackup --copy-back --target-dir=/backup/full

# 3. 基于时间点恢复
mysqlbinlog --start-datetime='2026-03-11 10:00:00' \
  --stop-datetime='2026-03-11 11:00:00' \
  mysql-bin.000001 | mysql -u root -p
```

## 参考资源

- [MySQL 官方文档](https://dev.mysql.com/doc/)
- [MySQL 性能优化](https://dev.mysql.com/doc/refman/8.0/en/optimization.html)
- [MHA 文档](https://github.com/yoshinorim/mha4mysql-manager)
- [Orchestrator 文档](https://github.com/openark/orchestrator)
- [ShardingSphere 文档](https://shardingsphere.apache.org/)
- [Percona XtraBackup](https://www.percona.com/software/mysql-database/percona-xtrabackup)

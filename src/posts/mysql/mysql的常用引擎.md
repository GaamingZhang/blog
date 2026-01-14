# MySQL的常用引擎

## 快速回顾

- InnoDB
- MyISAM
- Memory

## 概述

存储引擎是MySQL的核心组件之一,负责数据的存储、检索和管理。不同的存储引擎有不同的特性和适用场景,选择合适的存储引擎对数据库性能和功能有重要影响。

MySQL采用插件式存储引擎架构,允许用户根据实际需求选择不同的存储引擎。可以通过以下命令查看支持的存储引擎:

```sql
SHOW ENGINES;
```

## InnoDB引擎

### 基本特性

InnoDB是MySQL 5.5版本后的默认存储引擎,也是目前最常用的存储引擎。它是一个事务安全的存储引擎,提供了完整的ACID事务支持。

**核心特性:**

- **事务支持**: 完整支持ACID特性(原子性、一致性、隔离性、持久性)
- **行级锁**: 使用行级锁定机制,支持更高的并发性能
- **外键约束**: 支持外键(FOREIGN KEY)约束,维护数据完整性
- **崩溃恢复**: 通过redo log和undo log实现崩溃恢复能力
- **MVCC**: 多版本并发控制,提高读写并发性能

### 存储结构

InnoDB采用聚簇索引(Clustered Index)组织数据:

- **表空间**: 数据存储在表空间(Tablespace)中,可以是共享表空间或独立表空间
- **聚簇索引**: 数据按主键顺序物理存储,主键索引的叶子节点包含完整的行数据
- **二级索引**: 非主键索引的叶子节点存储主键值,需要回表查询

**文件组成:**

- `.ibd`文件: 独立表空间模式下,每个表有独立的.ibd文件存储数据和索引
- `ibdata1`: 共享表空间文件,存储数据字典、undo日志等
- `ib_logfile`: redo日志文件,用于崩溃恢复

### 锁机制

InnoDB实现了细粒度的锁机制:

**锁类型:**

- **共享锁(S锁)**: 读锁,允许多个事务同时读取同一资源
- **排他锁(X锁)**: 写锁,阻止其他事务读取或写入
- **意向锁**: 表级锁,用于表明事务准备在行上加锁
- **行锁**: 锁定具体的行记录
- **间隙锁(Gap Lock)**: 锁定索引记录之间的间隙,防止幻读
- **临键锁(Next-Key Lock)**: 行锁+间隙锁的组合

**锁算法:**

```sql
-- 记录锁示例
SELECT * FROM users WHERE id = 1 FOR UPDATE;

-- 间隙锁示例(防止id在1-10之间插入新记录)
SELECT * FROM users WHERE id BETWEEN 1 AND 10 FOR UPDATE;
```

### 事务隔离级别

InnoDB支持SQL标准定义的四种隔离级别:

| 隔离级别 | 脏读 | 不可重复读 | 幻读 | 说明 |
|---------|------|-----------|------|------|
| READ UNCOMMITTED | 可能 | 可能 | 可能 | 读取未提交数据 |
| READ COMMITTED | 不可能 | 可能 | 可能 | 只读取已提交数据 |
| REPEATABLE READ | 不可能 | 不可能 | 可能(InnoDB通过MVCC避免) | 可重复读,InnoDB默认级别 |
| SERIALIZABLE | 不可能 | 不可能 | 不可能 | 串行化,最高隔离级别 |

```sql
-- 设置事务隔离级别
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;

-- 查看当前隔离级别
SELECT @@transaction_isolation;
```

### 适用场景

InnoDB适合以下场景:

- 需要事务支持的应用(如金融系统、电商订单)
- 高并发读写场景
- 需要外键约束维护数据完整性
- 需要崩溃恢复能力的关键业务系统
- 大部分OLTP(在线事务处理)应用

### 性能优化建议

**主键设计:**
- 使用自增主键,避免页分裂
- 主键尽量短小,减少二级索引空间占用

**缓冲池配置:**
```sql
-- 设置InnoDB缓冲池大小(通常设置为物理内存的70-80%)
innodb_buffer_pool_size = 8G
```

**日志优化:**
```sql
-- 控制redo日志刷盘策略
innodb_flush_log_at_trx_commit = 1  -- 最安全,每次事务提交都刷盘
innodb_flush_log_at_trx_commit = 2  -- 较快,每秒刷盘
```

## MyISAM引擎

### 基本特性

MyISAM是MySQL早期的默认存储引擎,设计简单,读取速度快,但不支持事务。

**核心特性:**

- **表级锁**: 只支持表级锁定,并发写入性能较差
- **不支持事务**: 无ACID特性,不支持事务回滚
- **无外键**: 不支持外键约束
- **全文索引**: 支持全文索引(MySQL 5.6前InnoDB不支持)
- **压缩**: 支持表压缩,节省存储空间

### 存储结构

MyISAM将数据和索引分开存储:

**文件组成:**

- `.frm`文件: 存储表结构定义
- `.MYD`文件: 存储表数据(MYData)
- `.MYI`文件: 存储索引(MYIndex)

**索引特点:**

- 使用非聚簇索引
- 索引和数据分离存储
- 索引叶子节点存储数据文件的物理地址

### 锁机制

MyISAM只支持表级锁:

- **读锁(共享锁)**: 多个读操作可以并发执行
- **写锁(排他锁)**: 写操作会阻塞所有其他读写操作

```sql
-- 手动加表锁
LOCK TABLES users READ;
-- 执行查询操作
UNLOCK TABLES;

-- 加写锁
LOCK TABLES users WRITE;
-- 执行写操作
UNLOCK TABLES;
```

### 适用场景

MyISAM适合以下场景:

- 以读为主、写操作少的应用(如日志系统、档案系统)
- 不需要事务支持的场景
- 对数据一致性要求不高的应用
- 需要全文索引的场景(MySQL 5.6之前)
- 数据仓库、报表系统等OLAP场景

### 注意事项

- MyISAM表在崩溃后可能损坏,需要使用`REPAIR TABLE`修复
- 大量并发写入会导致严重的锁等待
- 不建议在高并发写入场景使用
- MySQL 8.0已经将默认引擎完全切换为InnoDB

## Memory引擎

### 基本特性

Memory引擎(也称HEAP引擎)将数据存储在内存中,提供极快的访问速度。

**核心特性:**

- **内存存储**: 所有数据存储在内存中
- **表级锁**: 只支持表级锁
- **不支持事务**: 无ACID特性
- **固定长度行**: 使用固定长度的行格式,VARCHAR被当作CHAR处理
- **快速访问**: 读写速度极快

### 存储特点

**索引类型:**

- 支持HASH索引(默认)
- 支持BTREE索引

```sql
-- 创建使用HASH索引的Memory表
CREATE TABLE cache_data (
    id INT PRIMARY KEY,
    data VARCHAR(100),
    KEY idx_data (data) USING HASH
) ENGINE=MEMORY;

-- 创建使用BTREE索引的Memory表
CREATE TABLE temp_result (
    id INT,
    score DECIMAL(5,2),
    KEY idx_score (score) USING BTREE
) ENGINE=MEMORY;
```

**数据持久化:**

- 数据只存在于内存,服务器重启后数据丢失
- 表结构(.frm文件)会持久化到磁盘

### 适用场景

Memory引擎适合以下场景:

- 临时数据存储(如会话数据、临时计算结果)
- 缓存常用的查询结果
- 需要快速访问的小型查找表
- 作为中间表进行复杂查询的临时存储
- 可以接受数据丢失的高速读写场景

### 使用限制

- **内存限制**: 表大小受`max_heap_table_size`参数限制
- **数据丢失**: 重启后数据丢失
- **不支持TEXT/BLOB**: 不能存储大对象类型
- **行长度固定**: 可能造成空间浪费

```sql
-- 设置Memory表最大大小
SET max_heap_table_size = 256 * 1024 * 1024;  -- 256MB
```

## Archive引擎

### 基本特性

Archive引擎专门用于存储大量的历史数据和归档数据,提供高压缩比。

**核心特性:**

- **高压缩**: 使用zlib压缩,压缩比可达1:10
- **只支持INSERT和SELECT**: 不支持DELETE和UPDATE操作
- **表级锁**: 插入时使用表级锁
- **不支持索引**: 除了自增主键外不支持其他索引

### 存储特点

**文件组成:**

- `.frm`文件: 表结构定义
- `.ARZ`文件: 压缩的数据文件

**操作限制:**

```sql
-- 创建Archive表
CREATE TABLE logs_archive (
    id INT AUTO_INCREMENT PRIMARY KEY,
    log_time TIMESTAMP,
    log_message TEXT,
    user_id INT
) ENGINE=ARCHIVE;

-- 允许的操作
INSERT INTO logs_archive VALUES (NULL, NOW(), 'User login', 1001);
SELECT * FROM logs_archive WHERE id > 1000;

-- 不允许的操作
-- UPDATE logs_archive SET log_message = 'New message' WHERE id = 1;  -- 错误
-- DELETE FROM logs_archive WHERE id < 100;  -- 错误
```

### 适用场景

Archive引擎适合以下场景:

- 日志归档系统
- 历史数据存储
- 数据仓库的事实表
- 只需要插入和查询的场景
- 存储空间有限,需要高压缩比的场景

## CSV引擎

### 基本特性

CSV引擎以CSV(逗号分隔值)格式存储数据,可以方便地与其他应用交换数据。

**核心特性:**

- **CSV格式**: 数据以纯文本CSV格式存储
- **不支持索引**: 无法创建索引
- **不支持NULL**: 所有字段必须定义为NOT NULL
- **易于交换**: 可以直接编辑.CSV文件

### 存储特点

**文件组成:**

- `.frm`文件: 表结构定义
- `.CSV`文件: 数据文件,可用文本编辑器打开
- `.CSM`文件: 元数据文件

```sql
-- 创建CSV表
CREATE TABLE export_data (
    id INT NOT NULL,
    name VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL
) ENGINE=CSV;

-- 插入数据
INSERT INTO export_data VALUES (1, 'Alice', 'alice@example.com');
```

### 适用场景

CSV引擎适合以下场景:

- 需要与Excel等工具交换数据
- 数据导入导出的中间表
- 简单的数据记录和日志
- 需要人工编辑数据文件的场景

## 存储引擎对比

### 功能对比表

| 特性 | InnoDB | MyISAM | Memory | Archive | CSV |
|------|--------|--------|--------|---------|-----|
| 事务支持 | ✓ | ✗ | ✗ | ✗ | ✗ |
| 锁粒度 | 行级锁 | 表级锁 | 表级锁 | 表级锁 | 表级锁 |
| 外键 | ✓ | ✗ | ✗ | ✗ | ✗ |
| MVCC | ✓ | ✗ | ✗ | ✗ | ✗ |
| 崩溃恢复 | ✓ | ✗ | ✗ | ✓ | ✗ |
| 全文索引 | ✓(5.6+) | ✓ | ✗ | ✗ | ✗ |
| 压缩 | ✓ | ✓ | ✗ | ✓ | ✗ |
| 数据缓存 | ✓ | ✗ | ✓ | ✗ | ✗ |
| 索引缓存 | ✓ | ✓ | ✓ | ✗ | ✗ |
| 存储位置 | 磁盘 | 磁盘 | 内存 | 磁盘 | 磁盘 |

### 性能对比

**读性能排序:**
Memory > InnoDB ≈ MyISAM > Archive > CSV

**写性能排序:**
InnoDB > Memory > MyISAM > Archive > CSV

**存储空间:**
Archive(最小) < InnoDB < MyISAM < CSV < Memory(内存)

## 选择存储引擎的建议

### 决策流程

1. **是否需要事务支持?**
   - 需要 → InnoDB
   - 不需要 → 继续判断

2. **读写比例如何?**
   - 读多写少 → 考虑MyISAM(但建议仍用InnoDB)
   - 写多或读写均衡 → InnoDB

3. **是否需要高速访问?**
   - 需要且数据可丢失 → Memory
   - 需要且数据不可丢失 → InnoDB

4. **是否只做归档存储?**
   - 是 → Archive
   - 否 → InnoDB

5. **是否需要数据交换?**
   - 需要CSV格式 → CSV
   - 否 → InnoDB

### 最佳实践

**通用建议:**

- **默认使用InnoDB**: 适合绝大多数场景,是最安全的选择
- **避免混用引擎**: 同一应用尽量使用同一种引擎,便于管理
- **考虑业务特点**: 根据具体业务需求选择合适的引擎

**特殊场景:**

```sql
-- 电商订单表(需要事务)
CREATE TABLE orders (
    order_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    total_amount DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- 访问日志表(只插入和查询)
CREATE TABLE access_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    ip VARCHAR(45),
    url VARCHAR(500),
    visit_time TIMESTAMP
) ENGINE=Archive;

-- 会话缓存表(临时数据)
CREATE TABLE sessions (
    session_id VARCHAR(64) PRIMARY KEY,
    user_id INT,
    last_activity TIMESTAMP,
    data TEXT
) ENGINE=Memory;
```

## 存储引擎管理

### 查看和修改引擎

```sql
-- 查看所有支持的引擎
SHOW ENGINES;

-- 查看表使用的引擎
SHOW TABLE STATUS LIKE 'table_name';

-- 查看表的创建语句
SHOW CREATE TABLE table_name;

-- 修改表的存储引擎
ALTER TABLE table_name ENGINE=InnoDB;

-- 创建表时指定引擎
CREATE TABLE new_table (
    id INT PRIMARY KEY,
    data VARCHAR(100)
) ENGINE=InnoDB;
```

### 设置默认引擎

```sql
-- 查看默认存储引擎
SHOW VARIABLES LIKE 'default_storage_engine';

-- 设置会话级别默认引擎
SET default_storage_engine=InnoDB;

-- 在配置文件中设置(my.cnf或my.ini)
[mysqld]
default-storage-engine=InnoDB
```

### 引擎转换注意事项

**转换前的检查:**

- 备份数据
- 检查表的大小和索引
- 评估停机时间(大表转换耗时较长)
- 测试应用兼容性

**转换方法:**

```sql
-- 方法1: ALTER TABLE(会锁表)
ALTER TABLE large_table ENGINE=InnoDB;

-- 方法2: 创建新表并导入数据(推荐)
CREATE TABLE large_table_new LIKE large_table;
ALTER TABLE large_table_new ENGINE=InnoDB;
INSERT INTO large_table_new SELECT * FROM large_table;
RENAME TABLE large_table TO large_table_old, large_table_new TO large_table;
DROP TABLE large_table_old;
```

## 常见问题

### 1. InnoDB和MyISAM的主要区别是什么?应该如何选择?

**核心区别:**

- **事务支持**: InnoDB支持事务(ACID特性),MyISAM不支持
- **锁机制**: InnoDB使用行级锁,支持更高并发;MyISAM只有表级锁
- **外键**: InnoDB支持外键约束,MyISAM不支持
- **崩溃恢复**: InnoDB有自动崩溃恢复能力,MyISAM表崩溃后需手动修复
- **存储结构**: InnoDB使用聚簇索引,数据和主键索引存储在一起;MyISAM的索引和数据分离

**选择建议:**

绝大多数情况下应该选择InnoDB,它是MySQL的默认引擎且功能更完善。只有在极少数特殊场景(如纯归档查询、不需要事务的日志系统)下才考虑MyISAM,但即使这些场景,InnoDB通常也能胜任。

### 2. 为什么InnoDB必须有主键?如果不显式创建主键会怎样?

**InnoDB的聚簇索引特性:**

InnoDB使用聚簇索引组织数据,必须依赖主键来确定数据的物理存储顺序。如果不显式创建主键,InnoDB会采用以下策略:

1. 查找第一个非空的唯一索引作为主键
2. 如果没有合适的唯一索引,InnoDB会自动创建一个6字节的隐藏主键(row_id)

**最佳实践:**

应该始终显式定义主键,推荐使用自增整数作为主键:

```sql
-- 推荐:使用自增主键
CREATE TABLE users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50),
    email VARCHAR(100)
) ENGINE=InnoDB;

-- 避免:使用过长的字符串作为主键
-- 会导致二级索引占用空间过大
CREATE TABLE users (
    email VARCHAR(100) PRIMARY KEY,  -- 不推荐
    username VARCHAR(50)
) ENGINE=InnoDB;
```

### 3. Memory引擎的数据会在什么时候丢失?如何避免数据丢失?

**数据丢失场景:**

- MySQL服务器重启
- MySQL服务器崩溃
- 执行`TRUNCATE TABLE`或`DROP TABLE`
- 内存不足导致表被清空

**避免策略:**

Memory引擎设计上就是临时存储,无法完全避免数据丢失。如果数据重要,应该:

1. **使用InnoDB代替**: 配置足够大的缓冲池来获得类似的性能
2. **定期备份到持久化存储**: 将Memory表数据定期同步到InnoDB表
3. **使用Redis等专业缓存**: 对于缓存场景,Redis等专业工具更合适
4. **应用层重建机制**: 服务启动时自动重建Memory表数据

```sql
-- 示例:将Memory表数据备份到InnoDB表
CREATE TABLE cache_data_backup LIKE cache_data;
ALTER TABLE cache_data_backup ENGINE=InnoDB;

-- 定期执行同步
INSERT INTO cache_data_backup SELECT * FROM cache_data
ON DUPLICATE KEY UPDATE data=VALUES(data);
```

### 4. 什么时候应该使用Archive引擎?它有什么限制?

**适用场景:**

Archive引擎专为历史数据归档设计,适合:

- 日志归档系统(访问日志、操作日志)
- 历史订单、历史交易数据
- 数据仓库的事实表
- 只需要插入和少量查询的数据

**核心限制:**

- 只支持INSERT和SELECT操作,不支持UPDATE和DELETE
- 除自增主键外不支持任何索引,查询性能受限
- 使用表级锁,并发插入性能一般
- 不支持事务

**使用示例:**

```sql
-- 适合:访问日志归档
CREATE TABLE access_logs_2024 (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    url VARCHAR(500),
    access_time TIMESTAMP,
    ip VARCHAR(45)
) ENGINE=Archive;

-- 不适合:需要更新的业务数据
CREATE TABLE orders (  -- 应该用InnoDB
    order_id BIGINT PRIMARY KEY,
    status VARCHAR(20),  -- 需要更新状态
    amount DECIMAL(10,2)
) ENGINE=Archive;  -- 错误选择
```

### 5. 如何在不停机的情况下将大表从MyISAM转换为InnoDB?

**推荐方法:创建新表迁移**

这种方法可以避免长时间锁表,适合生产环境:

```sql
-- 步骤1: 创建新的InnoDB表结构
CREATE TABLE users_new LIKE users;
ALTER TABLE users_new ENGINE=InnoDB;

-- 步骤2: 分批迁移数据(避免锁表过久)
-- 假设使用id作为主键
SET @batch_size = 10000;
SET @max_id = 0;

-- 循环执行直到所有数据迁移完成
INSERT INTO users_new 
SELECT * FROM users 
WHERE id > @max_id 
ORDER BY id 
LIMIT @batch_size;

-- 更新@max_id继续下一批...

-- 步骤3: 处理迁移期间的增量数据(根据业务设计)
-- 可以通过触发器、双写等方式保证数据一致性

-- 步骤4: 切换表名(快速操作)
RENAME TABLE users TO users_old, users_new TO users;

-- 步骤5: 验证后删除旧表
DROP TABLE users_old;
```

**使用工具方案:**

对于超大表,可以使用专业工具:

- **pt-online-schema-change**(Percona Toolkit): 在线DDL工具,通过触发器同步增量数据
- **gh-ost**(GitHub): GitHub开源的在线schema变更工具,使用binlog同步数据

```bash
# 使用pt-online-schema-change示例
pt-online-schema-change \
  --alter "ENGINE=InnoDB" \
  D=mydb,t=users \
  --execute
```

这些工具可以实现真正的零停机迁移,适合生产环境的大表转换。
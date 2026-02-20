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

# MySQL中的三种删除操作的区别

## 概述

MySQL提供了三种主要的删除数据的方式:DELETE、TRUNCATE和DROP。虽然它们都可以删除数据,但在实现机制、性能、事务支持、权限要求等方面存在显著差异。选择合适的删除操作对数据安全和性能优化至关重要。

**三种删除操作的定位:**

- **DELETE**: 删除表中的部分或全部行数据,支持WHERE条件过滤
- **TRUNCATE**: 快速清空整个表的数据,不支持WHERE条件
- **DROP**: 删除整个表结构和数据,表将不再存在

## DELETE操作

### 基本语法

```sql
-- 删除符合条件的行
DELETE FROM table_name WHERE condition;

-- 删除所有行(保留表结构)
DELETE FROM table_name;

-- 限制删除数量
DELETE FROM table_name WHERE condition LIMIT n;

-- 多表删除
DELETE t1, t2 FROM t1 INNER JOIN t2 ON t1.id = t2.id WHERE condition;
```

### 工作原理

DELETE是DML(数据操作语言)操作,逐行删除数据并记录每一行的删除操作到日志中。

**执行过程:**

```
1. 根据WHERE条件定位要删除的行
   ↓
2. 逐行删除,每删除一行:
   - 记录undo log(用于事务回滚)
   - 记录binlog(用于主从复制)
   - 标记行为已删除(InnoDB)
   ↓
3. 更新表的统计信息
   ↓
4. 提交事务(如果是自动提交模式)
```

### 详细示例

**基本删除:**

```sql
-- 创建测试表
CREATE TABLE users (
    user_id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50),
    email VARCHAR(100),
    status VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 插入测试数据
INSERT INTO users (username, email, status) VALUES
    ('alice', 'alice@example.com', 'active'),
    ('bob', 'bob@example.com', 'inactive'),
    ('charlie', 'charlie@example.com', 'active'),
    ('david', 'david@example.com', 'inactive');

-- 删除单条记录
DELETE FROM users WHERE user_id = 1;

-- 删除多条记录
DELETE FROM users WHERE status = 'inactive';

-- 删除所有记录
DELETE FROM users;
```

**批量删除优化:**

```sql
-- 分批删除大量数据,避免长事务和锁表
DELIMITER //

CREATE PROCEDURE batch_delete(
    IN p_table VARCHAR(64),
    IN p_condition VARCHAR(500),
    IN p_batch_size INT
)
BEGIN
    DECLARE v_affected_rows INT DEFAULT 0;
    
    REPEAT
        SET @sql = CONCAT(
            'DELETE FROM ', p_table,
            ' WHERE ', p_condition,
            ' LIMIT ', p_batch_size
        );
        
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
        
        SET v_affected_rows = ROW_COUNT();
        
        -- 短暂休眠,释放锁资源
        DO SLEEP(0.1);
        
    UNTIL v_affected_rows = 0 END REPEAT;
END //

DELIMITER ;

-- 使用示例:分批删除旧数据
CALL batch_delete(
    'logs', 
    'created_at < DATE_SUB(NOW(), INTERVAL 90 DAY)', 
    1000
);
```

**多表关联删除:**

```sql
-- 删除订单及其相关的订单项
DELETE orders, order_items
FROM orders
INNER JOIN order_items ON orders.order_id = order_items.order_id
WHERE orders.status = 'cancelled' 
  AND orders.created_at < DATE_SUB(NOW(), INTERVAL 1 YEAR);

-- 仅删除主表记录
DELETE orders
FROM orders
LEFT JOIN order_items ON orders.order_id = order_items.order_id
WHERE order_items.order_id IS NULL;  -- 没有订单项的订单
```

### DELETE的特性

**1. 事务支持**

```sql
-- DELETE支持事务,可以回滚
START TRANSACTION;

DELETE FROM users WHERE status = 'inactive';

-- 查看删除结果
SELECT * FROM users;

-- 可以回滚
ROLLBACK;

-- 数据恢复
SELECT * FROM users;  -- 删除的数据已恢复
```

**2. 触发器支持**

```sql
-- DELETE会触发BEFORE DELETE和AFTER DELETE触发器
DELIMITER //

CREATE TRIGGER trg_users_before_delete
BEFORE DELETE ON users
FOR EACH ROW
BEGIN
    -- 记录删除日志
    INSERT INTO deleted_users_log (user_id, username, deleted_at)
    VALUES (OLD.user_id, OLD.username, NOW());
END //

DELIMITER ;

-- 执行删除时自动记录日志
DELETE FROM users WHERE user_id = 1;
```

**3. 自增值保留**

```sql
-- DELETE不重置自增值
INSERT INTO users (username) VALUES ('alice');  -- user_id = 1
INSERT INTO users (username) VALUES ('bob');    -- user_id = 2

DELETE FROM users WHERE user_id = 2;

INSERT INTO users (username) VALUES ('charlie');  -- user_id = 3(不是2)

-- 查看当前自增值
SELECT AUTO_INCREMENT 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'mydb' AND TABLE_NAME = 'users';
-- AUTO_INCREMENT = 4
```

**4. 空间释放**

```sql
-- DELETE不会立即释放磁盘空间
-- InnoDB将删除的空间标记为可重用,但不返还给操作系统

-- 查看表大小
SELECT 
    table_name,
    ROUND((data_length + index_length) / 1024 / 1024, 2) AS size_mb
FROM information_schema.TABLES
WHERE table_schema = 'mydb' AND table_name = 'users';

-- 删除大量数据后,表大小可能不变

-- 释放空间需要使用OPTIMIZE TABLE
OPTIMIZE TABLE users;
```

### DELETE的性能考虑

**1. 使用索引加速**

```sql
-- 在WHERE条件列上创建索引
CREATE INDEX idx_status ON users(status);
CREATE INDEX idx_created_at ON users(created_at);

-- 快速定位要删除的行
DELETE FROM users WHERE status = 'inactive';
DELETE FROM users WHERE created_at < '2023-01-01';

-- 查看执行计划
EXPLAIN DELETE FROM users WHERE status = 'inactive';
```

**2. 避免全表扫描**

```sql
-- 不好的做法:无索引的列
DELETE FROM users WHERE YEAR(created_at) = 2023;  -- 函数导致索引失效

-- 好的做法:使用范围查询
DELETE FROM users 
WHERE created_at >= '2023-01-01' AND created_at < '2024-01-01';
```

**3. 控制删除数量**

```sql
-- 大量删除时使用LIMIT分批执行
DELETE FROM logs WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY) LIMIT 10000;

-- 配合循环执行
WHILE (SELECT COUNT(*) FROM logs WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY)) > 0
DO
    DELETE FROM logs WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY) LIMIT 10000;
    -- 休眠避免持续高负载
    SELECT SLEEP(1);
END WHILE;
```

## TRUNCATE操作

### 基本语法

```sql
-- 清空整个表
TRUNCATE TABLE table_name;

-- TRUNCATE等价于(但更快)
DELETE FROM table_name;
```

### 工作原理

TRUNCATE是DDL(数据定义语言)操作,通过删除和重建表来快速清空数据。

**执行过程:**

```
1. 删除原表的数据文件(.ibd)
   ↓
2. 重新创建空的数据文件
   ↓
3. 重置AUTO_INCREMENT计数器
   ↓
4. 更新数据字典
```

### 详细示例

```sql
-- 创建测试表并插入数据
CREATE TABLE test_table (
    id INT PRIMARY KEY AUTO_INCREMENT,
    data VARCHAR(100)
);

INSERT INTO test_table (data) VALUES ('row1'), ('row2'), ('row3');

-- 使用TRUNCATE清空表
TRUNCATE TABLE test_table;

-- 验证结果
SELECT * FROM test_table;  -- 空表

-- 插入新数据,自增值从1开始
INSERT INTO test_table (data) VALUES ('new_row');
SELECT * FROM test_table;  -- id = 1
```

### TRUNCATE的特性

**1. 不支持事务回滚**

```sql
-- TRUNCATE无法回滚(在InnoDB中实际可以回滚,但不推荐依赖)
START TRANSACTION;

TRUNCATE TABLE test_table;  -- DDL语句会导致隐式提交

ROLLBACK;  -- 无效,数据已被删除

-- 注意:在某些MySQL版本和配置下,TRUNCATE在事务中可能可以回滚
-- 但这不是标准行为,不应依赖
```

**2. 不触发DELETE触发器**

```sql
-- 创建DELETE触发器
DELIMITER //

CREATE TRIGGER trg_test_before_delete
BEFORE DELETE ON test_table
FOR EACH ROW
BEGIN
    INSERT INTO delete_log VALUES (OLD.id, NOW());
END //

DELIMITER ;

-- DELETE会触发触发器
DELETE FROM test_table WHERE id = 1;  -- 触发器执行

-- TRUNCATE不会触发触发器
TRUNCATE TABLE test_table;  -- 触发器不执行
```

**3. 重置自增值**

```sql
-- 演示自增值重置
CREATE TABLE counters (
    id INT PRIMARY KEY AUTO_INCREMENT,
    value VARCHAR(50)
);

INSERT INTO counters (value) VALUES ('a'), ('b'), ('c');
-- id: 1, 2, 3

DELETE FROM counters WHERE id = 3;

INSERT INTO counters (value) VALUES ('d');
-- id: 4 (自增值继续)

TRUNCATE TABLE counters;

INSERT INTO counters (value) VALUES ('e');
-- id: 1 (自增值重置)
```

**4. 立即释放空间**

```sql
-- TRUNCATE会立即释放磁盘空间
-- 查看表大小
SELECT 
    table_name,
    ROUND((data_length + index_length) / 1024 / 1024, 2) AS size_mb
FROM information_schema.TABLES
WHERE table_schema = 'mydb' AND table_name = 'test_table';

-- TRUNCATE后表大小恢复到最小
TRUNCATE TABLE test_table;

-- 再次查看,size_mb接近0
```

**5. 外键约束限制**

```sql
-- 创建父子表
CREATE TABLE parent (
    id INT PRIMARY KEY,
    name VARCHAR(50)
);

CREATE TABLE child (
    id INT PRIMARY KEY,
    parent_id INT,
    FOREIGN KEY (parent_id) REFERENCES parent(id)
);

-- 插入数据
INSERT INTO parent VALUES (1, 'parent1');
INSERT INTO child VALUES (1, 1);

-- 尝试TRUNCATE父表会失败
TRUNCATE TABLE parent;
-- ERROR: Cannot truncate a table referenced in a foreign key constraint

-- 必须先TRUNCATE子表
TRUNCATE TABLE child;
TRUNCATE TABLE parent;  -- 现在可以成功
```

### TRUNCATE的性能优势

```sql
-- 性能对比测试
-- 创建大表
CREATE TABLE large_table (
    id INT PRIMARY KEY AUTO_INCREMENT,
    data VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 插入100万行数据
INSERT INTO large_table (data)
SELECT CONCAT('data_', seq)
FROM (
    SELECT @row := @row + 1 AS seq
    FROM information_schema.COLUMNS, (SELECT @row := 0) r
    LIMIT 1000000
) t;

-- 测试DELETE(非常慢)
-- DELETE FROM large_table;  -- 可能需要几分钟

-- 测试TRUNCATE(非常快)
TRUNCATE TABLE large_table;  -- 通常不到1秒
```

## DROP操作

### 基本语法

```sql
-- 删除表
DROP TABLE table_name;

-- 删除多个表
DROP TABLE table1, table2, table3;

-- 如果表存在才删除
DROP TABLE IF EXISTS table_name;

-- 删除临时表
DROP TEMPORARY TABLE temp_table_name;
```

### 工作原理

DROP是DDL操作,删除表的定义和所有数据,包括表结构、数据、索引、触发器、约束等。

**执行过程:**

```
1. 检查表是否存在
   ↓
2. 检查是否有外键依赖
   ↓
3. 删除相关的触发器
   ↓
4. 删除表数据文件(.ibd)
   ↓
5. 删除表结构定义(.frm, MySQL 8.0前)
   ↓
6. 更新数据字典,移除表元数据
```

### 详细示例

**基本删除:**

```sql
-- 创建表
CREATE TABLE temp_data (
    id INT PRIMARY KEY,
    value VARCHAR(100)
);

-- 删除表
DROP TABLE temp_data;

-- 表不存在,再次删除会报错
DROP TABLE temp_data;
-- ERROR: Unknown table 'mydb.temp_data'

-- 安全删除
DROP TABLE IF EXISTS temp_data;  -- 不会报错
```

**删除有依赖关系的表:**

```sql
-- 创建父子表
CREATE TABLE departments (
    dept_id INT PRIMARY KEY,
    dept_name VARCHAR(100)
);

CREATE TABLE employees (
    emp_id INT PRIMARY KEY,
    name VARCHAR(100),
    dept_id INT,
    FOREIGN KEY (dept_id) REFERENCES departments(dept_id)
);

-- 尝试删除父表会失败
DROP TABLE departments;
-- ERROR: Cannot drop table 'departments': referenced by a foreign key constraint

-- 必须先删除子表
DROP TABLE employees;
DROP TABLE departments;  -- 现在可以成功

-- 或者一次性删除多个表
DROP TABLE IF EXISTS employees, departments;
```

**删除临时表:**

```sql
-- 创建临时表
CREATE TEMPORARY TABLE temp_calculations (
    id INT,
    result DECIMAL(10,2)
);

-- 插入数据
INSERT INTO temp_calculations VALUES (1, 100.50);

-- 删除临时表
DROP TEMPORARY TABLE temp_calculations;

-- 临时表在会话结束时自动删除
```

### DROP的特性

**1. 不可恢复**

```sql
-- DROP操作无法回滚,即使在事务中
START TRANSACTION;

DROP TABLE test_table;  -- 表立即被删除

ROLLBACK;  -- 无效,表无法恢复

-- 恢复只能通过备份
```

**2. 删除所有相关对象**

```sql
-- 创建表和相关对象
CREATE TABLE products (
    product_id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100),
    price DECIMAL(10,2)
);

-- 创建索引
CREATE INDEX idx_name ON products(name);

-- 创建触发器
DELIMITER //
CREATE TRIGGER trg_products_insert
BEFORE INSERT ON products
FOR EACH ROW
BEGIN
    SET NEW.name = UPPER(NEW.name);
END //
DELIMITER ;

-- DROP TABLE会删除表、索引、触发器等所有相关对象
DROP TABLE products;

-- 所有相关对象都被删除
SHOW TRIGGERS LIKE 'products';  -- 空结果
```

**3. 立即释放所有资源**

```sql
-- DROP会立即释放:
-- - 磁盘空间
-- - 表锁
-- - 元数据锁
-- - 缓存资源

-- 查看表占用空间
SELECT 
    table_name,
    ROUND((data_length + index_length) / 1024 / 1024, 2) AS size_mb
FROM information_schema.TABLES
WHERE table_schema = 'mydb' AND table_name = 'large_table';

-- DROP后空间立即释放
DROP TABLE large_table;
```

**4. 权限要求**

```sql
-- DROP需要DROP权限
-- 查看权限
SHOW GRANTS FOR 'username'@'localhost';

-- 授予DROP权限
GRANT DROP ON mydb.* TO 'username'@'localhost';

-- 撤销DROP权限(防止误操作)
REVOKE DROP ON mydb.* FROM 'username'@'localhost';
```

## 三种操作的对比

### 功能对比表

| 特性 | DELETE | TRUNCATE | DROP |
|------|--------|----------|------|
| 操作类型 | DML | DDL | DDL |
| 删除范围 | 可选择性删除(WHERE) | 删除所有行 | 删除整个表 |
| 表结构 | 保留 | 保留 | 删除 |
| 索引 | 保留 | 保留 | 删除 |
| 触发器 | 触发 | 不触发 | 删除 |
| 自增值 | 保留 | 重置 | 删除 |
| 事务支持 | 支持回滚 | 不支持(DDL隐式提交) | 不支持 |
| WHERE条件 | 支持 | 不支持 | 不支持 |
| 性能 | 慢(逐行删除) | 快(重建表) | 快(删除文件) |
| 空间释放 | 不立即释放 | 立即释放 | 立即释放 |
| 日志记录 | 记录每行 | 记录操作 | 记录操作 |
| 可恢复性 | 可通过binlog恢复 | 难以恢复 | 无法恢复 |

### 性能对比

```sql
-- 创建测试表
CREATE TABLE performance_test (
    id INT PRIMARY KEY AUTO_INCREMENT,
    data VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_created_at (created_at)
);

-- 插入100万行测试数据
INSERT INTO performance_test (data)
SELECT CONCAT('test_', seq)
FROM (
    SELECT @row := @row + 1 AS seq
    FROM information_schema.COLUMNS c1, information_schema.COLUMNS c2,
         (SELECT @row := 0) r
    LIMIT 1000000
) t;

-- 性能测试(大致耗时,实际取决于硬件)

-- DELETE全表(最慢: 可能需要几分钟到几十分钟)
-- DELETE FROM performance_test;
-- 约需: 60-300秒(取决于硬件和配置)

-- TRUNCATE(快: 通常1-2秒)
TRUNCATE TABLE performance_test;
-- 约需: 1-2秒

-- DROP(最快: 通常不到1秒)
DROP TABLE performance_test;
-- 约需: <1秒
```

### 适用场景对比

**使用DELETE的场景:**

```sql
-- 1. 删除部分数据
DELETE FROM orders WHERE status = 'cancelled' AND created_at < '2023-01-01';

-- 2. 需要事务保护
START TRANSACTION;
DELETE FROM user_sessions WHERE expires_at < NOW();
-- 可能回滚
ROLLBACK;

-- 3. 需要触发器执行
DELETE FROM users WHERE user_id = 1;  -- 触发审计日志

-- 4. 删除少量数据
DELETE FROM cache WHERE id = 123;

-- 5. 外键关联的子表
DELETE FROM order_items WHERE order_id = 456;
```

**使用TRUNCATE的场景:**

```sql
-- 1. 快速清空大表
TRUNCATE TABLE logs;

-- 2. 重置测试数据
TRUNCATE TABLE test_users;

-- 3. 清空临时表
TRUNCATE TABLE temp_calculations;

-- 4. 需要重置自增值
TRUNCATE TABLE counters;

-- 5. 定期清理全部数据的表
TRUNCATE TABLE daily_cache;
```

**使用DROP的场景:**

```sql
-- 1. 删除临时表
DROP TEMPORARY TABLE temp_results;

-- 2. 删除不再需要的表
DROP TABLE IF EXISTS old_archive_2020;

-- 3. 数据库架构变更
DROP TABLE deprecated_feature;

-- 4. 删除测试表
DROP TABLE IF EXISTS test_table;

-- 5. 清理过期的分区表
DROP TABLE sales_2019_q1;
```

## 安全删除的最佳实践

### 1. 删除前备份

```sql
-- 方法1: 创建备份表
CREATE TABLE users_backup AS SELECT * FROM users;

-- 然后执行删除
DELETE FROM users WHERE status = 'inactive';

-- 如果出错,可以从备份恢复
-- INSERT INTO users SELECT * FROM users_backup;

-- 方法2: 导出到文件
SELECT * INTO OUTFILE '/tmp/users_backup.csv'
FIELDS TERMINATED BY ',' 
ENCLOSED BY '"'
LINES TERMINATED BY '\n'
FROM users;

-- 方法3: 使用mysqldump
-- mysqldump -u root -p mydb users > users_backup.sql
```

### 2. 使用事务保护

```sql
-- 开启事务
START TRANSACTION;

-- 查看要删除的数据
SELECT * FROM users WHERE status = 'inactive';

-- 执行删除
DELETE FROM users WHERE status = 'inactive';

-- 验证结果
SELECT COUNT(*) FROM users;

-- 确认无误后提交
COMMIT;

-- 如果有问题,回滚
-- ROLLBACK;
```

### 3. 分步验证

```sql
-- 步骤1: 查询要删除的数据
SELECT COUNT(*), MIN(created_at), MAX(created_at)
FROM logs
WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY);

-- 步骤2: 限制删除数量测试
DELETE FROM logs 
WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY)
LIMIT 10;

-- 步骤3: 验证结果
SELECT * FROM logs ORDER BY created_at LIMIT 10;

-- 步骤4: 确认无误后批量删除
DELETE FROM logs 
WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY);
```

### 4. 软删除策略

```sql
-- 使用deleted标志而不是真正删除
ALTER TABLE users ADD COLUMN deleted BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP NULL;

-- "删除"操作实际是更新
UPDATE users 
SET deleted = TRUE, deleted_at = NOW()
WHERE user_id = 123;

-- 查询时过滤已删除数据
SELECT * FROM users WHERE deleted = FALSE;

-- 创建视图简化查询
CREATE VIEW active_users AS
SELECT * FROM users WHERE deleted = FALSE;

-- 定期物理删除旧数据
DELETE FROM users 
WHERE deleted = TRUE 
  AND deleted_at < DATE_SUB(NOW(), INTERVAL 1 YEAR);
```

### 5. 权限控制

```sql
-- 限制DELETE权限
-- 只给予SELECT和INSERT权限,不给DELETE权限
GRANT SELECT, INSERT, UPDATE ON mydb.* TO 'app_user'@'localhost';

-- 创建专门的删除账户
CREATE USER 'delete_admin'@'localhost' IDENTIFIED BY 'secure_password';
GRANT DELETE ON mydb.* TO 'delete_admin'@'localhost';

-- 使用存储过程封装删除操作
DELIMITER //
CREATE PROCEDURE safe_delete_user(IN p_user_id INT)
SQL SECURITY DEFINER  -- 使用定义者权限
BEGIN
    -- 添加业务验证
    IF p_user_id > 0 THEN
        DELETE FROM users WHERE user_id = p_user_id;
    END IF;
END //
DELIMITER ;

-- 授予执行权限,而不是DELETE权限
GRANT EXECUTE ON PROCEDURE safe_delete_user TO 'app_user'@'localhost';
```

### 6. 审计日志

```sql
-- 创建删除审计表
CREATE TABLE delete_audit (
    audit_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    table_name VARCHAR(64),
    deleted_count INT,
    deleted_by VARCHAR(50),
    delete_condition TEXT,
    deleted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 记录删除操作
DELIMITER //
CREATE PROCEDURE delete_with_audit(
    IN p_table_name VARCHAR(64),
    IN p_condition TEXT
)
BEGIN
    DECLARE v_count INT;
    
    -- 计算要删除的行数
    SET @count_sql = CONCAT(
        'SELECT COUNT(*) INTO @delete_count FROM ',
        p_table_name, ' WHERE ', p_condition
    );
    PREPARE stmt FROM @count_sql;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
    
    -- 执行删除
    SET @delete_sql = CONCAT(
        'DELETE FROM ', p_table_name, ' WHERE ', p_condition
    );
    PREPARE stmt FROM @delete_sql;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
    
    -- 记录审计
    INSERT INTO delete_audit (table_name, deleted_count, deleted_by, delete_condition)
    VALUES (p_table_name, @delete_count, USER(), p_condition);
END //
DELIMITER ;

-- 使用示例
CALL delete_with_audit('logs', 'created_at < DATE_SUB(NOW(), INTERVAL 90 DAY)');
```

## 常见问题

### 1. DELETE和TRUNCATE在性能上差距有多大?如何选择?

**性能差距分析:**

DELETE和TRUNCATE的性能差距主要来自于它们的实现机制:

**DELETE的开销:**
- 逐行扫描和删除
- 每行生成undo log(用于回滚)
- 每行生成binlog(用于主从复制)
- 更新索引
- 维护事务信息

**TRUNCATE的开销:**
- 直接删除数据文件并重建
- 只记录DDL操作日志
- 不需要逐行处理

**实际测试:**

```sql
-- 创建测试表(100万行)
CREATE TABLE perf_test (
    id INT PRIMARY KEY AUTO_INCREMENT,
    data VARCHAR(100),
    INDEX idx_data (data)
);

-- 插入数据
INSERT INTO perf_test (data)
SELECT MD5(RAND())
FROM information_schema.COLUMNS c1, information_schema.COLUMNS c2
LIMIT 1000000;

-- 测试DELETE(耗时:约60-180秒)
SET profiling = 1;
DELETE FROM perf_test;
SHOW PROFILES;

-- 重新插入数据后测试TRUNCATE(耗时:约1-2秒)
TRUNCATE TABLE perf_test;
SHOW PROFILES;

-- 性能差距:TRUNCATE比DELETE快50-100倍以上
```

**选择建议:**

```sql
-- 使用DELETE的情况:
-- 1. 需要删除部分数据
DELETE FROM logs WHERE created_at < '2023-01-01';

-- 2. 需要事务回滚能力
START TRANSACTION;
DELETE FROM temp_data;
-- 可能需要回滚
ROLLBACK;

-- 3. 需要触发器执行
DELETE FROM users WHERE user_id = 123;  -- 触发审计记录

-- 4. 有外键约束的子表
DELETE FROM order_items WHERE order_id = 456;

-- 使用TRUNCATE的情况:
-- 1. 清空大表(百万行以上)
TRUNCATE TABLE access_logs;

-- 2. 重置测试环境
TRUNCATE TABLE test_data;

-- 3. 需要重置自增ID
TRUNCATE TABLE counters;

-- 4. 定期清空的临时表/缓存表
TRUNCATE TABLE session_cache;
```

### 2. 如何安全地删除大表中的海量数据?

删除大表数据是高风险操作,可能导致:
- 长时间锁表影响业务
- 主从复制延迟
- binlog膨胀
- undo log空间不足

**方案1: 分批删除(推荐)**

```sql
-- 创建分批删除存储过程
DELIMITER //

CREATE PROCEDURE batch_delete_safe(
    IN p_table VARCHAR(64),
    IN p_where_condition VARCHAR(1000),
    IN p_batch_size INT,
    IN p_sleep_seconds DECIMAL(3,2)
)
BEGIN
    DECLARE v_affected INT DEFAULT 1;
    DECLARE v_total INT DEFAULT 0;
    
    WHILE v_affected > 0 DO
        -- 执行批量删除
        SET @sql = CONCAT(
            'DELETE FROM ', p_table,
            ' WHERE ', p_where_condition,
            ' LIMIT ', p_batch_size
        );
        
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        SET v_affected = ROW_COUNT();
        DEALLOCATE PREPARE stmt;
        
        SET v_total = v_total + v_affected;
        
        -- 输出进度
        SELECT CONCAT('Deleted: ', v_affected, ' rows, Total: ', v_total) AS progress;
        
        -- 休眠,释放锁和资源
        DO SLEEP(p_sleep_seconds);
    END WHILE;
    
    SELECT CONCAT('Complete! Total deleted: ', v_total, ' rows') AS result;
END //

DELIMITER ;

-- 使用示例:每批删除10000行,休眠0.5秒
CALL batch_delete_safe(
    'logs',
    'created_at < DATE_SUB(NOW(), INTERVAL 90 DAY)',
    10000,
    0.5
);
```

**方案2: 创建新表迁移(适合保留少量数据)**

```sql
-- 1. 创建新表结构
CREATE TABLE logs_new LIKE logs;

-- 2. 复制需要保留的数据
INSERT INTO logs_new
SELECT * FROM logs
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 90 DAY);

-- 3. 验证数据
SELECT COUNT(*) FROM logs_new;

-- 4. 重命名表(快速切换)
RENAME TABLE logs TO logs_old, logs_new TO logs;

-- 5. 删除旧表
DROP TABLE logs_old;
```

**方案3: 分区表删除(最优)**

```sql
-- 创建分区表
CREATE TABLE logs_partitioned (
    log_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    message TEXT,
    created_at TIMESTAMP
)
PARTITION BY RANGE (UNIX_TIMESTAMP(created_at)) (
    PARTITION p202301 VALUES LESS THAN (UNIX_TIMESTAMP('2023-02-01')),
    PARTITION p202302 VALUES LESS THAN (UNIX_TIMESTAMP('2023-03-01')),
    PARTITION p202303 VALUES LESS THAN (UNIX_TIMESTAMP('2023-04-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);

-- 删除整个分区(几乎瞬间完成)
ALTER TABLE logs_partitioned DROP PARTITION p202301;

-- 添加新分区
ALTER TABLE logs_partitioned ADD PARTITION (
    PARTITION p202404 VALUES LESS THAN (UNIX_TIMESTAMP('2023-05-01'))
);
```

**方案4: 使用pt-archiver工具(Percona Toolkit)**

```bash
# 归档并删除旧数据
pt-archiver \
  --source h=localhost,D=mydb,t=logs \
  --where "created_at < DATE_SUB(NOW(), INTERVAL 90 DAY)" \
  --limit 10000 \
  --commit-each \
  --sleep 1 \
  --purge  # 删除而不归档
  
# 或归档到文件
pt-archiver \
  --source h=localhost,D=mydb,t=logs \
  --where "created_at < '2023-01-01'" \
  --file '/backup/logs_archive_%Y%m%d.txt' \
  --limit 5000 \
  --purge
```

### 3. TRUNCATE在事务中可以回滚吗?不同存储引擎有区别吗?

**标准行为:**

TRUNCATE是DDL语句,理论上会导致隐式提交,不支持回滚。但实际情况取决于MySQL版本和存储引擎。

**InnoDB存储引擎:**

```sql
-- MySQL 5.5及以前:TRUNCATE不可回滚
START TRANSACTION;
TRUNCATE TABLE test_table;
ROLLBACK;  -- 无效,数据已删除

-- MySQL 5.6及以后:在某些情况下TRUNCATE可以回滚
START TRANSACTION;

CREATE TABLE test_rollback (id INT, data VARCHAR(50));
INSERT INTO test_rollback VALUES (1, 'test');

TRUNCATE TABLE test_rollback;

SELECT * FROM test_rollback;  -- 空表

ROLLBACK;

SELECT * FROM test_rollback;  -- MySQL 5.6+可能恢复数据

-- 但这不是标准行为,不应依赖!
```

**实际测试(MySQL 8.0):**

```sql
-- 测试1:普通表的TRUNCATE
CREATE TABLE t1 (id INT) ENGINE=InnoDB;
INSERT INTO t1 VALUES (1), (2), (3);

START TRANSACTION;
TRUNCATE TABLE t1;
ROLLBACK;

SELECT * FROM t1;
-- 结果:在MySQL 8.0中,数据可能被恢复(取决于配置)
-- 但官方文档说明TRUNCATE会隐式提交

-- 测试2:有外键的表
CREATE TABLE parent (id INT PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE child (
    id INT PRIMARY KEY, 
    parent_id INT,
    FOREIGN KEY (parent_id) REFERENCES parent(id)
) ENGINE=InnoDB;

INSERT INTO parent VALUES (1);
INSERT INTO child VALUES (1, 1);

START TRANSACTION;
TRUNCATE TABLE child;  -- 成功
TRUNCATE TABLE parent;  -- 失败:外键约束
ROLLBACK;

-- child表的TRUNCATE无法回滚
```

**MyISAM存储引擎:**

```sql
-- MyISAM不支持事务
CREATE TABLE test_myisam (id INT) ENGINE=MyISAM;
INSERT INTO test_myisam VALUES (1), (2);

START TRANSACTION;
TRUNCATE TABLE test_myisam;
ROLLBACK;

SELECT * FROM test_myisam;  -- 空表,无法回滚
```

**最佳实践:**

```sql
-- 不要依赖TRUNCATE的回滚特性
-- 如果需要回滚能力,使用DELETE

-- 需要清空表且不需要回滚
TRUNCATE TABLE temp_table;

-- 需要回滚能力
START TRANSACTION;
DELETE FROM temp_table;
-- 可以安全回滚
ROLLBACK;

-- 生产环境建议
-- 1. 清空表前先备份
CREATE TABLE temp_table_backup AS SELECT * FROM temp_table;

-- 2. 确认后再TRUNCATE
TRUNCATE TABLE temp_table;

-- 3. 如果出错,从备份恢复
-- INSERT INTO temp_table SELECT * FROM temp_table_backup;
```

### 4. 删除操作对主从复制有什么影响?如何避免复制延迟?

**DELETE的复制影响:**

```sql
-- DELETE在主从复制中的问题

-- 主库执行
DELETE FROM large_table WHERE created_at < '2023-01-01';
-- 影响100万行,耗时60秒

-- binlog记录(ROW格式):记录每一行的删除
-- binlog大小可能达到几GB

-- 从库重放:
-- - 需要相同的时间(60秒+)
-- - 可能导致严重的复制延迟
-- - Seconds_Behind_Master显著增加

-- 检查复制延迟
SHOW SLAVE STATUS\G
-- Seconds_Behind_Master: 可能是几百到几千秒
```

**TRUNCATE的复制影响:**

```sql
-- 主库执行
TRUNCATE TABLE large_table;
-- 几乎瞬间完成

-- binlog记录:只记录一条TRUNCATE语句
-- binlog大小:几百字节

-- 从库重放:
-- - 几乎瞬间完成
-- - 复制延迟最小
-- - Seconds_Behind_Master几乎不变
```

**避免复制延迟的策略:**

**策略1: 分批删除**

```sql
-- 小批量分次删除,减少单次影响
DELIMITER //

CREATE PROCEDURE delete_with_replication_check(
    IN p_table VARCHAR(64),
    IN p_condition VARCHAR(500),
    IN p_batch_size INT,
    IN p_max_lag_seconds INT
)
BEGIN
    DECLARE v_affected INT DEFAULT 1;
    DECLARE v_lag INT;
    
    WHILE v_affected > 0 DO
        -- 执行批量删除
        SET @sql = CONCAT(
            'DELETE FROM ', p_table,
            ' WHERE ', p_condition,
            ' LIMIT ', p_batch_size
        );
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        SET v_affected = ROW_COUNT();
        DEALLOCATE PREPARE stmt;
        
        -- 检查从库延迟(需要配置从库信息)
        -- SELECT MAX(Seconds_Behind_Master) INTO v_lag FROM slave_status;
        
        -- 如果延迟过大,等待
        WHILE v_lag > p_max_lag_seconds DO
            DO SLEEP(5);
            -- SELECT MAX(Seconds_Behind_Master) INTO v_lag FROM slave_status;
        END WHILE;
        
        DO SLEEP(0.5);
    END WHILE;
END //

DELIMITER ;
```

**策略2: 使用延迟从库**

```sql
-- 配置一个延迟从库用于数据恢复
-- 在从库配置(my.cnf)
[mysqld]
# 从库延迟1小时执行
CHANGE MASTER TO MASTER_DELAY = 3600;

-- 这样如果主库误删除,可以从延迟从库恢复
-- 在延迟窗口内执行:
STOP SLAVE SQL_THREAD;
-- 从延迟从库导出数据
-- 恢复到主库
```

**策略3: 使用STATEMENT格式binlog**

```sql
-- 对于大批量DELETE,考虑使用STATEMENT格式
-- 注意:可能导致主从数据不一致

SET SESSION binlog_format = 'STATEMENT';

DELETE FROM large_table WHERE created_at < '2023-01-01';
-- binlog只记录SQL语句,大小很小

SET SESSION binlog_format = 'ROW';  -- 恢复默认

-- 警告:STATEMENT格式在某些情况下不安全
-- - 使用LIMIT但没有ORDER BY
-- - 使用UUID()、NOW()等非确定性函数
-- - 触发器中的操作
```

**策略4: 在业务低峰期执行**

```sql
-- 创建定时任务,在凌晨执行
CREATE EVENT evt_cleanup_old_logs
ON SCHEDULE EVERY 1 DAY
STARTS '2024-01-01 02:00:00'
DO
BEGIN
    -- 分批删除,每批5000行
    CALL batch_delete_safe('logs', 
        'created_at < DATE_SUB(NOW(), INTERVAL 90 DAY)', 
        5000, 
        1.0
    );
END;
```

**策略5: 使用分区表**

```sql
-- 分区表的优势:DROP PARTITION不产生大量binlog
CREATE TABLE logs (
    log_id BIGINT AUTO_INCREMENT,
    message TEXT,
    created_at TIMESTAMP,
    PRIMARY KEY (log_id, created_at)
)
PARTITION BY RANGE (TO_DAYS(created_at)) (
    PARTITION p20240101 VALUES LESS THAN (TO_DAYS('2024-02-01')),
    PARTITION p20240201 VALUES LESS THAN (TO_DAYS('2024-03-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);

-- 删除整个分区(极快,binlog很小)
ALTER TABLE logs DROP PARTITION p20240101;
-- 从库几乎无延迟
```

### 5. 如何实现软删除?软删除和物理删除各有什么优缺点?

**软删除实现方案:**

**方案1: 使用deleted标志位**

```sql
-- 基础软删除表设计
CREATE TABLE users (
    user_id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100),
    deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP NULL,
    deleted_by INT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_deleted (deleted),
    INDEX idx_deleted_at (deleted_at)
);

-- 软删除操作
UPDATE users 
SET deleted = TRUE, 
    deleted_at = NOW(),
    deleted_by = 1001  -- 当前用户ID
WHERE user_id = 123;

-- 查询时过滤已删除数据
SELECT * FROM users WHERE deleted = FALSE;

-- 创建视图简化查询
CREATE VIEW active_users AS
SELECT * FROM users WHERE deleted = FALSE;

-- 使用视图
SELECT * FROM active_users WHERE username LIKE 'john%';
```

**方案2: 使用deleted_at时间戳**

```sql
-- 更简洁的设计:NULL表示未删除
CREATE TABLE products (
    product_id INT PRIMARY KEY AUTO_INCREMENT,
    product_name VARCHAR(100),
    price DECIMAL(10,2),
    deleted_at TIMESTAMP NULL,
    INDEX idx_deleted_at (deleted_at)
);

-- 软删除
UPDATE products 
SET deleted_at = NOW()
WHERE product_id = 456;

-- 查询活动数据
SELECT * FROM products WHERE deleted_at IS NULL;

-- 查询已删除数据
SELECT * FROM products WHERE deleted_at IS NOT NULL;

-- 恢复删除
UPDATE products 
SET deleted_at = NULL
WHERE product_id = 456;
```

**方案3: 状态枚举方式**

```sql
CREATE TABLE orders (
    order_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    customer_id INT,
    total_amount DECIMAL(10,2),
    status ENUM('active', 'cancelled', 'deleted', 'completed'),
    status_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status)
);

-- 软删除
UPDATE orders SET status = 'deleted' WHERE order_id = 789;

-- 查询活动订单
SELECT * FROM orders WHERE status IN ('active', 'completed');
```

**软删除的封装:**

```sql
-- 创建软删除存储过程
DELIMITER //

CREATE PROCEDURE soft_delete_user(
    IN p_user_id INT,
    IN p_deleted_by INT,
    OUT p_result VARCHAR(100)
)
BEGIN
    DECLARE v_exists INT;
    DECLARE v_already_deleted BOOLEAN;
    
    -- 检查用户是否存在
    SELECT COUNT(*), MAX(deleted) INTO v_exists, v_already_deleted
    FROM users WHERE user_id = p_user_id;
    
    IF v_exists = 0 THEN
        SET p_result = 'ERROR: User not found';
    ELSEIF v_already_deleted THEN
        SET p_result = 'ERROR: User already deleted';
    ELSE
        UPDATE users 
        SET deleted = TRUE, 
            deleted_at = NOW(), 
            deleted_by = p_deleted_by
        WHERE user_id = p_user_id;
        
        SET p_result = 'SUCCESS: User soft deleted';
    END IF;
END //

-- 创建恢复存储过程
CREATE PROCEDURE undelete_user(
    IN p_user_id INT,
    OUT p_result VARCHAR(100)
)
BEGIN
    UPDATE users 
    SET deleted = FALSE, 
        deleted_at = NULL, 
        deleted_by = NULL
    WHERE user_id = p_user_id AND deleted = TRUE;
    
    IF ROW_COUNT() > 0 THEN
        SET p_result = 'SUCCESS: User restored';
    ELSE
        SET p_result = 'ERROR: User not found or not deleted';
    END IF;
END //

DELIMITER ;

-- 使用示例
CALL soft_delete_user(123, 1001, @result);
SELECT @result;

CALL undelete_user(123, @result);
SELECT @result;
```

**软删除 vs 物理删除对比:**

| 特性 | 软删除 | 物理删除 |
|------|--------|----------|
| 数据恢复 | 容易恢复 | 困难(需要备份) |
| 数据审计 | 保留完整历史 | 无法追溯 |
| 存储空间 | 占用空间持续增长 | 释放空间 |
| 查询性能 | 需要过滤deleted | 性能更好 |
| 唯一约束 | 需要特殊处理 | 简单 |
| 外键关联 | 复杂 | 简单(级联删除) |
| 业务复杂度 | 较高 | 较低 |
| 合规性 | 可能违反数据保护法规 | 符合"被遗忘权" |

**软删除的问题和解决:**

**问题1: 唯一约束冲突**

```sql
-- 问题:软删除后无法创建同名用户
CREATE TABLE users (
    user_id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE,
    deleted BOOLEAN DEFAULT FALSE
);

INSERT INTO users (username) VALUES ('alice');
UPDATE users SET deleted = TRUE WHERE username = 'alice';  -- 软删除
INSERT INTO users (username) VALUES ('alice');  -- ERROR: Duplicate entry

-- 解决方案1:唯一约束包含deleted字段
CREATE TABLE users (
    user_id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50),
    deleted BOOLEAN DEFAULT FALSE,
    UNIQUE KEY uk_username_deleted (username, deleted)
);
-- 缺点:deleted=TRUE的记录也只能有一个

-- 解决方案2:使用NULL表示未删除
CREATE TABLE users (
    user_id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50),
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY uk_username (username, deleted_at)
);
-- NULL != NULL,所以可以有多个deleted_at=NULL的同名记录

-- 解决方案3:重命名被删除的记录
UPDATE users 
SET username = CONCAT(username, '_deleted_', user_id),
    deleted = TRUE
WHERE user_id = 123;
```

**问题2: 查询性能下降**

```sql
-- 大量已删除数据影响查询性能
SELECT * FROM users WHERE deleted = FALSE AND status = 'active';
-- 需要扫描大量deleted=TRUE的记录

-- 解决方案1:定期归档
-- 将已删除超过1年的数据移到归档表
CREATE TABLE users_archive LIKE users;

INSERT INTO users_archive
SELECT * FROM users
WHERE deleted = TRUE AND deleted_at < DATE_SUB(NOW(), INTERVAL 1 YEAR);

DELETE FROM users
WHERE deleted = TRUE AND deleted_at < DATE_SUB(NOW(), INTERVAL 1 YEAR);

-- 解决方案2:分区表
CREATE TABLE users (
    user_id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50),
    deleted BOOLEAN DEFAULT FALSE,
    KEY (deleted)
)
PARTITION BY LIST(deleted) (
    PARTITION p_active VALUES IN (0),
    PARTITION p_deleted VALUES IN (1)
);
-- 查询只扫描p_active分区

-- 解决方案3:使用索引优化
CREATE INDEX idx_deleted_composite ON users(deleted, status, created_at);
-- 覆盖索引提高查询性能
```

**混合策略(推荐):**

```sql
-- 结合软删除和物理删除的优点
CREATE TABLE users (
    user_id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50),
    email VARCHAR(100),
    deleted_at TIMESTAMP NULL,
    INDEX idx_deleted_at (deleted_at)
);

-- 1. 软删除(保留30天)
UPDATE users SET deleted_at = NOW() WHERE user_id = 123;

-- 2. 定期物理删除(30天后)
CREATE EVENT evt_cleanup_deleted_users
ON SCHEDULE EVERY 1 DAY
DO
BEGIN
    -- 物理删除软删除超过30天的数据
    DELETE FROM users 
    WHERE deleted_at IS NOT NULL 
      AND deleted_at < DATE_SUB(NOW(), INTERVAL 30 DAY)
    LIMIT 10000;
END;

-- 这样既保留了短期恢复能力,又避免了长期的性能和存储问题
```

## 参考资源

- [MySQL 官方文档 - DELETE 语句](https://dev.mysql.com/doc/refman/8.0/en/delete.html)
- [MySQL 官方文档 - TRUNCATE TABLE 语句](https://dev.mysql.com/doc/refman/8.0/en/truncate-table.html)
- [MySQL 官方文档 - DROP TABLE 语句](https://dev.mysql.com/doc/refman/8.0/en/drop-table.html)
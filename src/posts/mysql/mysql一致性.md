---
date: 2026-01-15
author: Gaaming Zhang
isOriginal: true
article: true
category:
  - 数据库
tag:
  - 数据库
---

# MySQL的数据一致性

## 概述

数据一致性是ACID事务特性中的"C"(Consistency),指的是数据库从一个一致性状态转换到另一个一致性状态。一致性确保数据库中的数据满足所有定义的规则、约束和业务逻辑要求。

**一致性的核心含义:**

- **约束一致性**: 数据满足所有完整性约束(主键、外键、唯一性、非空等)
- **业务一致性**: 数据符合业务规则(如账户余额不为负、库存不超卖)
- **状态一致性**: 事务执行前后,数据库从一个有效状态到另一个有效状态
- **逻辑一致性**: 相关数据之间保持逻辑关联(如订单总额等于明细之和)

**一致性与其他ACID特性的关系:**

- **原子性(A)**: 确保事务的完整执行,是实现一致性的基础
- **隔离性(I)**: 防止并发事务破坏一致性
- **持久性(D)**: 保证已提交的一致性状态永久保存

## 数据库约束与一致性

### 主键约束(PRIMARY KEY)

主键约束确保表中每一行都有唯一标识,不允许重复和NULL值。

```sql
-- 创建表时定义主键
CREATE TABLE users (
    user_id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL
);

-- 或者使用CONSTRAINT语法
CREATE TABLE orders (
    order_id BIGINT AUTO_INCREMENT,
    user_id INT NOT NULL,
    total_amount DECIMAL(10,2),
    PRIMARY KEY (order_id)
);

-- 复合主键
CREATE TABLE order_items (
    order_id BIGINT,
    product_id INT,
    quantity INT,
    PRIMARY KEY (order_id, product_id)
);
```

**主键的一致性保证:**

```sql
-- 插入重复主键会失败
INSERT INTO users (user_id, username, email) 
VALUES (1, 'alice', 'alice@example.com');

INSERT INTO users (user_id, username, email) 
VALUES (1, 'bob', 'bob@example.com');
-- ERROR: Duplicate entry '1' for key 'PRIMARY'

-- 插入NULL主键会失败
INSERT INTO users (user_id, username, email) 
VALUES (NULL, 'charlie', 'charlie@example.com');
-- ERROR: Column 'user_id' cannot be null
```

### 外键约束(FOREIGN KEY)

外键约束维护表之间的引用完整性,确保关联数据的一致性。

```sql
-- 创建父表
CREATE TABLE customers (
    customer_id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE
) ENGINE=InnoDB;

-- 创建子表,带外键约束
CREATE TABLE orders (
    order_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    customer_id INT NOT NULL,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_amount DECIMAL(10,2),
    FOREIGN KEY (customer_id) REFERENCES customers(customer_id)
        ON DELETE RESTRICT
        ON UPDATE CASCADE
) ENGINE=InnoDB;
```

**外键的级联操作:**

```sql
-- ON DELETE 选项:
-- RESTRICT/NO ACTION: 禁止删除(默认)
-- CASCADE: 级联删除子表记录
-- SET NULL: 子表外键设为NULL
-- SET DEFAULT: 子表外键设为默认值

-- ON UPDATE 选项:
-- RESTRICT/NO ACTION: 禁止更新
-- CASCADE: 级联更新子表外键
-- SET NULL: 子表外键设为NULL

-- 示例:级联删除
CREATE TABLE order_items (
    item_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id BIGINT NOT NULL,
    product_id INT NOT NULL,
    quantity INT,
    FOREIGN KEY (order_id) REFERENCES orders(order_id)
        ON DELETE CASCADE  -- 删除订单时自动删除订单项
) ENGINE=InnoDB;
```

**外键的一致性保证:**

```sql
-- 插入不存在的外键值会失败
INSERT INTO orders (customer_id, total_amount) 
VALUES (999, 100.00);
-- ERROR: Cannot add or update a child row: a foreign key constraint fails

-- 删除被引用的父记录会失败(RESTRICT模式)
DELETE FROM customers WHERE customer_id = 1;
-- ERROR: Cannot delete or update a parent row: a foreign key constraint fails

-- 必须先删除子记录
DELETE FROM orders WHERE customer_id = 1;
DELETE FROM customers WHERE customer_id = 1;
-- 成功
```

### 唯一约束(UNIQUE)

唯一约束确保列中的值不重复,但允许NULL值。

```sql
-- 单列唯一约束
CREATE TABLE users (
    user_id INT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100),
    phone VARCHAR(20),
    UNIQUE KEY uk_email (email),
    UNIQUE KEY uk_phone (phone)
);

-- 复合唯一约束
CREATE TABLE enrollment (
    student_id INT,
    course_id INT,
    semester VARCHAR(20),
    UNIQUE KEY uk_enrollment (student_id, course_id, semester)
);
```

**唯一约束的特性:**

```sql
-- 重复值会失败
INSERT INTO users (user_id, username, email) 
VALUES (1, 'alice', 'alice@example.com');

INSERT INTO users (user_id, username, email) 
VALUES (2, 'alice', 'bob@example.com');
-- ERROR: Duplicate entry 'alice' for key 'username'

-- 多个NULL值是允许的
INSERT INTO users (user_id, username, email) VALUES (1, 'alice', NULL);
INSERT INTO users (user_id, username, email) VALUES (2, 'bob', NULL);
-- 成功,因为NULL != NULL
```

### 非空约束(NOT NULL)

非空约束确保列不能包含NULL值。

```sql
CREATE TABLE products (
    product_id INT PRIMARY KEY,
    product_name VARCHAR(100) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    description TEXT,  -- 允许NULL
    stock_quantity INT NOT NULL DEFAULT 0
);

-- 违反非空约束
INSERT INTO products (product_id, product_name, price) 
VALUES (1, NULL, 99.99);
-- ERROR: Column 'product_name' cannot be null
```

### 检查约束(CHECK)

CHECK约束定义列值必须满足的条件(MySQL 8.0.16+支持)。

```sql
-- 创建带CHECK约束的表
CREATE TABLE accounts (
    account_id INT PRIMARY KEY,
    account_name VARCHAR(100) NOT NULL,
    balance DECIMAL(10,2) NOT NULL,
    account_type ENUM('savings', 'checking', 'credit'),
    -- CHECK约束
    CONSTRAINT chk_balance CHECK (balance >= 0),
    CONSTRAINT chk_credit_limit CHECK (
        account_type != 'credit' OR balance >= -10000
    )
);

-- 违反CHECK约束
INSERT INTO accounts (account_id, account_name, balance, account_type)
VALUES (1, 'John Doe', -100, 'savings');
-- ERROR: Check constraint 'chk_balance' is violated

-- 满足约束的数据
INSERT INTO accounts (account_id, account_name, balance, account_type)
VALUES (1, 'John Doe', 1000, 'savings');
-- 成功
```

**复杂的CHECK约束:**

```sql
CREATE TABLE orders (
    order_id BIGINT PRIMARY KEY,
    order_date DATE NOT NULL,
    ship_date DATE,
    delivery_date DATE,
    status VARCHAR(20),
    -- 发货日期必须晚于订单日期
    CONSTRAINT chk_ship_after_order CHECK (ship_date IS NULL OR ship_date >= order_date),
    -- 交付日期必须晚于发货日期
    CONSTRAINT chk_delivery_after_ship CHECK (
        delivery_date IS NULL OR ship_date IS NULL OR delivery_date >= ship_date
    ),
    -- 已完成订单必须有交付日期
    CONSTRAINT chk_completed_has_delivery CHECK (
        status != 'completed' OR delivery_date IS NOT NULL
    )
);
```

## 事务与一致性

### 事务的一致性保证

事务确保一组操作作为整体执行,维护数据的一致性状态。

```sql
-- 转账操作:保证总金额不变
START TRANSACTION;

-- 检查余额
SELECT balance INTO @from_balance 
FROM accounts WHERE account_id = 'A001' FOR UPDATE;

IF @from_balance < 1000 THEN
    ROLLBACK;
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Insufficient balance';
END IF;

-- 执行转账
UPDATE accounts SET balance = balance - 1000 WHERE account_id = 'A001';
UPDATE accounts SET balance = balance + 1000 WHERE account_id = 'A002';

-- 验证一致性
SELECT 
    (SELECT SUM(balance) FROM accounts WHERE account_id IN ('A001', 'A002'))
    INTO @total_after;

IF @total_after != @total_before THEN
    ROLLBACK;
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Balance inconsistency detected';
END IF;

COMMIT;
```

### 存储过程中的一致性控制

```sql
DELIMITER //

CREATE PROCEDURE transfer_with_validation(
    IN p_from_account VARCHAR(50),
    IN p_to_account VARCHAR(50),
    IN p_amount DECIMAL(10,2),
    OUT p_result VARCHAR(100)
)
BEGIN
    DECLARE v_from_balance DECIMAL(10,2);
    DECLARE v_total_before DECIMAL(10,2);
    DECLARE v_total_after DECIMAL(10,2);
    
    -- 异常处理
    DECLARE EXIT HANDLER FOR SQLEXCEPTION
    BEGIN
        ROLLBACK;
        SET p_result = 'ERROR: Transaction failed';
    END;
    
    START TRANSACTION;
    
    -- 记录转账前总金额
    SELECT SUM(balance) INTO v_total_before
    FROM accounts 
    WHERE account_id IN (p_from_account, p_to_account);
    
    -- 检查源账户余额
    SELECT balance INTO v_from_balance
    FROM accounts 
    WHERE account_id = p_from_account
    FOR UPDATE;
    
    IF v_from_balance < p_amount THEN
        ROLLBACK;
        SET p_result = 'ERROR: Insufficient balance';
    ELSE
        -- 执行转账
        UPDATE accounts 
        SET balance = balance - p_amount 
        WHERE account_id = p_from_account;
        
        UPDATE accounts 
        SET balance = balance + p_amount 
        WHERE account_id = p_to_account;
        
        -- 验证总金额不变
        SELECT SUM(balance) INTO v_total_after
        FROM accounts 
        WHERE account_id IN (p_from_account, p_to_account);
        
        IF v_total_after != v_total_before THEN
            ROLLBACK;
            SET p_result = 'ERROR: Balance verification failed';
        ELSE
            COMMIT;
            SET p_result = 'SUCCESS: Transfer completed';
        END IF;
    END IF;
END //

DELIMITER ;

-- 使用存储过程
CALL transfer_with_validation('A001', 'A002', 500.00, @result);
SELECT @result;
```

### 触发器维护一致性

触发器可以自动维护派生数据和业务规则的一致性。

```sql
-- 场景:订单总额自动计算

-- 创建订单表
CREATE TABLE orders (
    order_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    customer_id INT,
    total_amount DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建订单明细表
CREATE TABLE order_items (
    item_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id BIGINT,
    product_id INT,
    quantity INT,
    price DECIMAL(10,2),
    FOREIGN KEY (order_id) REFERENCES orders(order_id) ON DELETE CASCADE
);

-- 插入订单项时更新订单总额
DELIMITER //

CREATE TRIGGER trg_order_items_insert
AFTER INSERT ON order_items
FOR EACH ROW
BEGIN
    UPDATE orders 
    SET total_amount = (
        SELECT SUM(quantity * price) 
        FROM order_items 
        WHERE order_id = NEW.order_id
    )
    WHERE order_id = NEW.order_id;
END //

-- 更新订单项时更新订单总额
CREATE TRIGGER trg_order_items_update
AFTER UPDATE ON order_items
FOR EACH ROW
BEGIN
    UPDATE orders 
    SET total_amount = (
        SELECT SUM(quantity * price) 
        FROM order_items 
        WHERE order_id = NEW.order_id
    )
    WHERE order_id = NEW.order_id;
END //

-- 删除订单项时更新订单总额
CREATE TRIGGER trg_order_items_delete
AFTER DELETE ON order_items
FOR EACH ROW
BEGIN
    UPDATE orders 
    SET total_amount = COALESCE((
        SELECT SUM(quantity * price) 
        FROM order_items 
        WHERE order_id = OLD.order_id
    ), 0)
    WHERE order_id = OLD.order_id;
END //

DELIMITER ;
```

**使用示例:**

```sql
-- 创建订单
INSERT INTO orders (order_id, customer_id) VALUES (1, 1001);

-- 添加订单项,total_amount自动更新
INSERT INTO order_items (order_id, product_id, quantity, price)
VALUES (1, 2001, 2, 99.99);  -- total_amount = 199.98

INSERT INTO order_items (order_id, product_id, quantity, price)
VALUES (1, 2002, 1, 149.99);  -- total_amount = 349.97

-- 验证一致性
SELECT order_id, total_amount FROM orders WHERE order_id = 1;
-- total_amount = 349.97

SELECT order_id, SUM(quantity * price) as calculated_total
FROM order_items WHERE order_id = 1;
-- calculated_total = 349.97
```

## 并发控制与一致性

### 脏读问题

脏读是指一个事务读取了另一个未提交事务修改的数据。

```sql
-- 会话1
START TRANSACTION;
UPDATE accounts SET balance = balance - 100 WHERE account_id = 'A001';
-- 未提交

-- 会话2 (READ UNCOMMITTED隔离级别)
SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
START TRANSACTION;
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 读取到会话1未提交的修改(脏读)

-- 会话1回滚
ROLLBACK;

-- 会话2读取的数据变为无效数据
```

**解决方案:使用READ COMMITTED或更高隔离级别**

```sql
-- 会话2
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;
START TRANSACTION;
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 只能读取已提交的数据,避免脏读
```

### 不可重复读问题

不可重复读是指在同一事务中,多次读取同一数据返回不同的结果。

```sql
-- 会话1
START TRANSACTION;
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 结果:1000

-- 会话2
START TRANSACTION;
UPDATE accounts SET balance = 1500 WHERE account_id = 'A001';
COMMIT;

-- 会话1再次读取
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 结果:1500 (与第一次读取不一致)
```

**解决方案:使用REPEATABLE READ隔离级别**

```sql
-- 会话1
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
START TRANSACTION;

SELECT balance FROM accounts WHERE account_id = 'A001';
-- 结果:1000

-- 即使会话2修改并提交,会话1仍然读取快照
SELECT balance FROM accounts WHERE account_id = 'A001';
-- 结果:仍然是1000 (可重复读)

COMMIT;
```

### 幻读问题

幻读是指在同一事务中,多次执行相同查询返回不同的行集。

```sql
-- 会话1
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
START TRANSACTION;

SELECT COUNT(*) FROM orders WHERE customer_id = 1001;
-- 结果:10

-- 会话2
START TRANSACTION;
INSERT INTO orders (customer_id, total_amount) VALUES (1001, 100);
COMMIT;

-- 会话1再次查询
SELECT COUNT(*) FROM orders WHERE customer_id = 1001;
-- InnoDB使用Next-Key Lock,结果仍然是10,避免了幻读
```

**InnoDB的幻读防护:**

InnoDB在REPEATABLE READ级别通过MVCC和Next-Key Lock避免幻读:

```sql
-- 演示Next-Key Lock
START TRANSACTION;

-- 查询并锁定范围
SELECT * FROM orders 
WHERE customer_id = 1001 AND order_id BETWEEN 1 AND 100
FOR UPDATE;

-- 其他事务无法在此范围内插入新记录
-- INSERT INTO orders (order_id, customer_id) VALUES (50, 1001);
-- 会被阻塞

COMMIT;
```

### 丢失更新问题

丢失更新是指两个事务同时更新同一数据,后提交的事务覆盖了先提交的事务的更新。

```sql
-- 场景:库存扣减

-- 会话1
START TRANSACTION;
SELECT stock INTO @stock1 FROM products WHERE product_id = 1001;
-- @stock1 = 100

-- 会话2
START TRANSACTION;
SELECT stock INTO @stock2 FROM products WHERE product_id = 1001;
-- @stock2 = 100

-- 会话1扣减库存
UPDATE products SET stock = @stock1 - 10 WHERE product_id = 1001;
COMMIT;
-- stock = 90

-- 会话2扣减库存
UPDATE products SET stock = @stock2 - 20 WHERE product_id = 1001;
COMMIT;
-- stock = 80 (会话1的更新丢失)

-- 实际应该是: 100 - 10 - 20 = 70
```

**解决方案1:使用FOR UPDATE锁定**

```sql
-- 会话1
START TRANSACTION;
SELECT stock INTO @stock1 FROM products WHERE product_id = 1001 FOR UPDATE;
-- 锁定该行

UPDATE products SET stock = @stock1 - 10 WHERE product_id = 1001;
COMMIT;

-- 会话2会等待会话1提交后再执行
START TRANSACTION;
SELECT stock INTO @stock2 FROM products WHERE product_id = 1001 FOR UPDATE;
-- 等待会话1释放锁

UPDATE products SET stock = @stock2 - 20 WHERE product_id = 1001;
COMMIT;
-- 正确结果: stock = 70
```

**解决方案2:使用原子更新**

```sql
-- 直接使用UPDATE语句,不需要先SELECT
UPDATE products SET stock = stock - 10 WHERE product_id = 1001;
-- 自动保证原子性,避免丢失更新
```

**解决方案3:乐观锁(版本号)**

```sql
-- 添加版本号字段
ALTER TABLE products ADD COLUMN version INT NOT NULL DEFAULT 0;

-- 更新时检查版本号
START TRANSACTION;

SELECT stock, version INTO @stock, @ver 
FROM products WHERE product_id = 1001;

-- 更新时验证版本号
UPDATE products 
SET stock = @stock - 10, version = version + 1
WHERE product_id = 1001 AND version = @ver;

-- 检查是否更新成功
IF ROW_COUNT() = 0 THEN
    -- 版本号不匹配,说明数据已被修改
    ROLLBACK;
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Data has been modified';
ELSE
    COMMIT;
END IF;
```

## 分布式环境下的一致性

### 主从复制的一致性问题

主从复制可能导致主库和从库数据不一致。

```sql
-- 主库
INSERT INTO users (username, email) VALUES ('alice', 'alice@example.com');

-- 从库可能存在复制延迟
-- 立即在从库查询可能查不到新插入的数据
SELECT * FROM users WHERE username = 'alice';
-- 可能返回空结果(复制延迟)
```

**解决方案:**

```sql
-- 1. 关键操作读主库
-- 在应用层配置读写分离时,对于刚写入的数据读主库

-- 2. 使用半同步复制
-- 配置MySQL半同步复制,确保至少一个从库确认接收binlog

-- 3. 检查复制延迟
SHOW SLAVE STATUS\G
-- 查看Seconds_Behind_Master,确认复制延迟

-- 4. 使用读写分离中间件(如ProxySQL)
-- 自动检测复制延迟,延迟过大时路由到主库
```

### 分库分表的一致性

分库分表场景下,跨库事务难以保证一致性。

```sql
-- 场景:订单表和库存表在不同数据库

-- 数据库1: 创建订单
START TRANSACTION;
INSERT INTO orders (order_id, user_id, product_id, quantity)
VALUES (1001, 1, 2001, 5);
COMMIT;

-- 数据库2: 减少库存
START TRANSACTION;
UPDATE products SET stock = stock - 5 WHERE product_id = 2001;
COMMIT;

-- 如果数据库2的操作失败,数据库1的订单无法回滚
-- 导致数据不一致
```

**解决方案1:分布式事务(XA)**

```sql
-- 使用XA事务
XA START 'xid1';
INSERT INTO orders (order_id, user_id, product_id, quantity)
VALUES (1001, 1, 2001, 5);
XA END 'xid1';
XA PREPARE 'xid1';

-- 在另一个数据库
XA START 'xid2';
UPDATE products SET stock = stock - 5 WHERE product_id = 2001;
XA END 'xid2';
XA PREPARE 'xid2';

-- 如果两个PREPARE都成功,提交
XA COMMIT 'xid1';
XA COMMIT 'xid2';

-- 如果任一失败,回滚
XA ROLLBACK 'xid1';
XA ROLLBACK 'xid2';
```

**解决方案2:最终一致性(消息队列)**

```sql
-- 数据库1: 创建订单并发送消息
START TRANSACTION;

INSERT INTO orders (order_id, user_id, product_id, quantity)
VALUES (1001, 1, 2001, 5);

-- 同时写入本地消息表
INSERT INTO outbox_messages (message_id, event_type, payload, status)
VALUES (UUID(), 'order_created', '{"product_id":2001,"quantity":5}', 'pending');

COMMIT;

-- 定时任务扫描消息表,发送到消息队列
-- 消费者收到消息后减少库存

-- 数据库2: 消费消息更新库存
START TRANSACTION;
UPDATE products SET stock = stock - 5 WHERE product_id = 2001;
-- 记录消息已处理
INSERT INTO processed_messages (message_id) VALUES ('xxx');
COMMIT;
```

**解决方案3:TCC模式**

```sql
-- Try阶段:预留资源
-- 数据库1
START TRANSACTION;
INSERT INTO orders (order_id, status) VALUES (1001, 'trying');
COMMIT;

-- 数据库2
START TRANSACTION;
UPDATE products SET stock = stock - 5, frozen_stock = frozen_stock + 5
WHERE product_id = 2001;
COMMIT;

-- Confirm阶段:确认操作
-- 数据库1
UPDATE orders SET status = 'confirmed' WHERE order_id = 1001;

-- 数据库2
UPDATE products SET frozen_stock = frozen_stock - 5 WHERE product_id = 2001;

-- 或Cancel阶段:取消操作
-- 数据库1
DELETE FROM orders WHERE order_id = 1001;

-- 数据库2
UPDATE products SET stock = stock + 5, frozen_stock = frozen_stock - 5
WHERE product_id = 2001;
```

## 应用层的一致性保证

### 乐观锁实现

```sql
-- 使用版本号实现乐观锁
CREATE TABLE inventory (
    product_id INT PRIMARY KEY,
    product_name VARCHAR(100),
    stock INT NOT NULL,
    version INT NOT NULL DEFAULT 0
);

-- 应用层代码(伪代码)
function updateStock(productId, quantity) {
    -- 读取当前版本
    result = SELECT stock, version FROM inventory WHERE product_id = productId;
    currentStock = result.stock;
    currentVersion = result.version;
    
    -- 计算新库存
    newStock = currentStock - quantity;
    if (newStock < 0) {
        throw "Insufficient stock";
    }
    
    -- 更新时检查版本号
    affectedRows = UPDATE inventory 
                   SET stock = newStock, version = version + 1
                   WHERE product_id = productId AND version = currentVersion;
    
    if (affectedRows == 0) {
        -- 版本号已变化,重试
        retry updateStock(productId, quantity);
    }
}
```

### 悲观锁实现

```sql
-- 使用SELECT FOR UPDATE实现悲观锁
START TRANSACTION;

-- 锁定行
SELECT stock INTO @current_stock
FROM inventory 
WHERE product_id = 1001
FOR UPDATE;

-- 检查库存
IF @current_stock < 10 THEN
    ROLLBACK;
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Insufficient stock';
END IF;

-- 扣减库存
UPDATE inventory 
SET stock = stock - 10
WHERE product_id = 1001;

COMMIT;
```

### 幂等性设计

确保重复执行同一操作不会导致数据不一致。

```sql
-- 使用唯一业务ID保证幂等性
CREATE TABLE payments (
    payment_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    transaction_id VARCHAR(64) UNIQUE NOT NULL,  -- 业务唯一ID
    user_id INT NOT NULL,
    amount DECIMAL(10,2),
    status VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 幂等的支付操作
INSERT INTO payments (transaction_id, user_id, amount, status)
VALUES ('TXN20240115001', 1001, 100.00, 'completed')
ON DUPLICATE KEY UPDATE status = VALUES(status);
-- 重复执行不会创建多条支付记录
```

**存储过程实现幂等性:**

```sql
DELIMITER //

CREATE PROCEDURE process_payment_idempotent(
    IN p_transaction_id VARCHAR(64),
    IN p_user_id INT,
    IN p_amount DECIMAL(10,2),
    OUT p_result VARCHAR(100)
)
BEGIN
    DECLARE v_exists INT;
    
    -- 检查是否已处理
    SELECT COUNT(*) INTO v_exists
    FROM payments
    WHERE transaction_id = p_transaction_id;
    
    IF v_exists > 0 THEN
        SET p_result = 'ALREADY_PROCESSED';
    ELSE
        START TRANSACTION;
        
        -- 插入支付记录
        INSERT INTO payments (transaction_id, user_id, amount, status)
        VALUES (p_transaction_id, p_user_id, p_amount, 'completed');
        
        -- 扣除用户余额
        UPDATE users 
        SET balance = balance - p_amount
        WHERE user_id = p_user_id;
        
        COMMIT;
        SET p_result = 'SUCCESS';
    END IF;
END //

DELIMITER ;
```

## 一致性监控与验证

### 数据一致性检查

```sql
-- 检查订单总额与明细是否一致
SELECT 
    o.order_id,
    o.total_amount as recorded_total,
    COALESCE(SUM(oi.quantity * oi.price), 0) as calculated_total,
    o.total_amount - COALESCE(SUM(oi.quantity * oi.price), 0) as difference
FROM orders o
LEFT JOIN order_items oi ON o.order_id = oi.order_id
GROUP BY o.order_id
HAVING ABS(difference) > 0.01;  -- 允许0.01的浮点误差

-- 检查库存数据一致性
SELECT 
    p.product_id,
    p.product_name,
    p.stock as recorded_stock,
    p.stock - COALESCE(SUM(i.quantity), 0) as calculated_stock,
    CASE 
        WHEN p.stock != COALESCE(SUM(i.quantity), 0) THEN 'INCONSISTENT'
        ELSE 'OK'
    END as status
FROM products p
LEFT JOIN inventory_logs i ON p.product_id = i.product_id
    AND i.created_at >= DATE_SUB(NOW(), INTERVAL 1 DAY)
GROUP BY p.product_id;

-- 检查账户余额一致性
SELECT 
    a.account_id,
    a.balance as current_balance,
    COALESCE(SUM(t.amount), 0) as transaction_sum,
    a.balance - COALESCE(SUM(t.amount), 0) as difference
FROM accounts a
LEFT JOIN transactions t ON a.account_id = t.account_id
GROUP BY a.account_id
HAVING ABS(difference) > 0.01;
```

### 定期一致性校验

```sql
-- 创建一致性检查任务表
CREATE TABLE consistency_check_log (
    check_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    check_type VARCHAR(50),
    check_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_records INT,
    inconsistent_records INT,
    status VARCHAR(20),
    details TEXT
);

-- 一致性检查存储过程
DELIMITER //

CREATE PROCEDURE check_order_consistency()
BEGIN
    DECLARE v_total_orders INT;
    DECLARE v_inconsistent_orders INT;
    
    -- 统计总订单数
    SELECT COUNT(*) INTO v_total_orders FROM orders;
    
    -- 统计不一致的订单数
    SELECT COUNT(*) INTO v_inconsistent_orders
    FROM (
        SELECT o.order_id
        FROM orders o
        LEFT JOIN order_items oi ON o.order_id = oi.order_id
        GROUP BY o.order_id
        HAVING ABS(o.total_amount - COALESCE(SUM(oi.quantity * oi.price), 0)) > 0.01
    ) AS inconsistent;
    
    -- 记录检查结果
    INSERT INTO consistency_check_log 
        (check_type, total_records, inconsistent_records, status)
    VALUES 
        ('order_total', v_total_orders, v_inconsistent_orders,
         IF(v_inconsistent_orders = 0, 'OK', 'WARNING'));
    
    -- 如果发现不一致,可以发送告警
    IF v_inconsistent_orders > 0 THEN
        SIGNAL SQLSTATE '45000' 
        SET MESSAGE_TEXT = 'Order consistency check failed';
    END IF;
END //

DELIMITER ;

-- 定期执行检查(使用事件调度器)
CREATE EVENT evt_daily_consistency_check
ON SCHEDULE EVERY 1 DAY
STARTS '2024-01-01 02:00:00'
DO
    CALL check_order_consistency();
```

### 审计日志

```sql
-- 创建审计日志表
CREATE TABLE audit_log (
    audit_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    table_name VARCHAR(64),
    operation VARCHAR(10),  -- INSERT/UPDATE/DELETE
    record_id VARCHAR(100),
    old_value JSON,
    new_value JSON,
    changed_by VARCHAR(50),
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_table_record (table_name, record_id),
    INDEX idx_changed_at (changed_at)
);

-- 为关键表创建审计触发器
DELIMITER //

CREATE TRIGGER trg_accounts_audit_update
AFTER UPDATE ON accounts
FOR EACH ROW
BEGIN
    INSERT INTO audit_log (table_name, operation, record_id, old_value, new_value, changed_by)
    VALUES (
        'accounts',
        'UPDATE',
        NEW.account_id,
        JSON_OBJECT('balance', OLD.balance, 'status', OLD.status),
        JSON_OBJECT('balance', NEW.balance, 'status', NEW.status),
        USER()
    );
END //

CREATE TRIGGER trg_accounts_audit_delete
AFTER DELETE ON accounts
FOR EACH ROW
BEGIN
    INSERT INTO audit_log (table_name, operation, record_id, old_value, changed_by)
    VALUES (
        'accounts',
        'DELETE',
        OLD.account_id,
        JSON_OBJECT('balance', OLD.balance, 'status', OLD.status),
        USER()
    );
END //

DELIMITER ;
```

## 最佳实践

### 1. 合理使用约束

```sql
-- 完整的表设计示例
CREATE TABLE products (
    product_id INT PRIMARY KEY AUTO_INCREMENT,
    product_code VARCHAR(50) UNIQUE NOT NULL,
    product_name VARCHAR(200) NOT NULL,
    category_id INT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    cost DECIMAL(10,2),
    stock INT NOT NULL DEFAULT 0,
    min_stock INT DEFAULT 0,
    max_stock INT,
    status ENUM('active', 'inactive', 'discontinued') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- 外键约束
    FOREIGN KEY (category_id) REFERENCES categories(category_id),
    
    -- CHECK约束
    CONSTRAINT chk_price_positive CHECK (price > 0),
    CONSTRAINT chk_cost_reasonable CHECK (cost IS NULL OR cost >= 0),
    CONSTRAINT chk_stock_range CHECK (stock >= 0 AND (max_stock IS NULL OR stock <= max_stock)),
    CONSTRAINT chk_min_max_stock CHECK (max_stock IS NULL OR min_stock <= max_stock),
    CONSTRAINT chk_price_cost CHECK (cost IS NULL OR price >= cost),
    
    -- 索引
    INDEX idx_category (category_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB;
```

### 2. 事务设计原则

```sql
-- 好的事务设计
START TRANSACTION;

-- 1. 事务尽量短小
-- 2. 先执行可能失败的操作
-- 3. 避免在事务中执行耗时操作(如外部API调用)
-- 4. 使用合适的隔离级别

-- 示例:订单创建
INSERT INTO orders (customer_id, status) VALUES (1001, 'pending');
SET @order_id = LAST_INSERT_ID();

INSERT INTO order_items (order_id, product_id, quantity, price)
SELECT @order_id, product_id, quantity, price 
FROM shopping_cart WHERE customer_id = 1001;

UPDATE products p
INNER JOIN shopping_cart sc ON p.product_id = sc.product_id
SET p.stock = p.stock - sc.quantity
WHERE sc.customer_id = 1001;

DELETE FROM shopping_cart WHERE customer_id = 1001;

COMMIT;
```

### 3. 避免常见陷阱

```sql
-- 陷阱1: 隐式类型转换导致索引失效
-- 不好的写法
SELECT * FROM users WHERE user_id = '123';  -- user_id是INT类型

-- 好的写法
SELECT * FROM users WHERE user_id = 123;

-- 陷阱2: NULL值的处理
-- 不好的写法
SELECT * FROM products WHERE stock != 100;  -- 不会返回stock为NULL的行

-- 好的写法
SELECT * FROM products WHERE stock != 100 OR stock IS NULL;

-- 陷阱3: 浮点数比较
-- 不好的写法
SELECT * FROM orders WHERE total_amount = 99.99;

-- 好的写法
SELECT * FROM orders WHERE ABS(total_amount - 99.99) < 0.01;

-- 陷阱4: 在事务中使用DDL
START TRANSACTION;
UPDATE accounts SET balance = balance - 100 WHERE account_id = 'A001';
-- DDL会导致隐式提交
ALTER TABLE accounts ADD COLUMN last_updated TIMESTAMP;  -- 前面的UPDATE已提交!
-- 不好的做法
```

### 4. 数据恢复策略

```sql
-- 启用binlog
SET GLOBAL binlog_format = 'ROW';  -- 推荐使用ROW格式

-- 定期备份
-- 使用mysqldump
mysqldump -u root -p --single-transaction --master-data=2 mydb > backup.sql

-- 或使用物理备份(Percona XtraBackup)
xtrabackup --backup --target-dir=/backup/

-- 创建恢复点
FLUSH LOGS;  -- 创建新的binlog文件

-- 误操作后的数据恢复
-- 1. 停止应用,防止继续写入
-- 2. 恢复最近的备份
-- 3. 应用binlog增量恢复到误操作前
mysqlbinlog --start-datetime="2024-01-15 10:00:00" \
            --stop-datetime="2024-01-15 14:30:00" \
            mysql-bin.000001 | mysql -u root -p
```

## 常见问题

### 1. 如何在高并发场景下保证库存扣减的一致性?

高并发库存扣减是典型的一致性场景,常见的解决方案有:

**方案1: 悲观锁(SELECT FOR UPDATE)**

```sql
START TRANSACTION;

-- 锁定库存记录
SELECT stock INTO @current_stock
FROM products
WHERE product_id = 1001
FOR UPDATE;

-- 检查库存
IF @current_stock >= 10 THEN
    UPDATE products SET stock = stock - 10 WHERE product_id = 1001;
    COMMIT;
ELSE
    ROLLBACK;
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Insufficient stock';
END IF;
```

**优点:** 强一致性,不会超卖
**缺点:** 并发性能差,容易产生锁等待

**方案2: 乐观锁(版本号)**

```sql
-- 添加版本号字段
ALTER TABLE products ADD COLUMN version INT DEFAULT 0;

-- 应用层实现
START TRANSACTION;

SELECT stock, version INTO @stock, @ver FROM products WHERE product_id = 1001;

UPDATE products 
SET stock = stock - 10, version = version + 1
WHERE product_id = 1001 AND version = @ver AND stock >= 10;

IF ROW_COUNT() = 0 THEN
    ROLLBACK;
    -- 重试或返回失败
ELSE
    COMMIT;
END IF;
```

**优点:** 并发性能好,无锁等待
**缺点:** 冲突时需要重试,可能多次失败

**方案3: 原子更新(推荐)**

```sql
-- 直接使用UPDATE,不需要先SELECT
UPDATE products 
SET stock = stock - 10
WHERE product_id = 1001 AND stock >= 10;

-- 检查影响行数
IF ROW_COUNT() = 0 THEN
    -- 库存不足
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Insufficient stock';
END IF;
```

**优点:** 最简单,性能最好,MySQL自动保证原子性
**缺点:** 需要在WHERE条件中验证约束

**方案4: Redis预扣减+异步扣减**

```sql
-- 1. 先在Redis中预扣减(原子操作)
-- DECR stock:1001

-- 2. 异步消费消息,扣减MySQL库存
START TRANSACTION;
UPDATE products SET stock = stock - 10 WHERE product_id = 1001;
COMMIT;

-- 3. 定期对账,修正Redis和MySQL的差异
```

**优点:** 高性能,支持超高并发
**缺点:** 实现复杂,需要保证最终一致性

### 2. 主从复制延迟导致的数据不一致如何解决?

**问题场景:**

```sql
-- 主库写入
INSERT INTO users (username) VALUES ('alice');

-- 立即从从库读取
SELECT * FROM users WHERE username = 'alice';
-- 可能查不到(复制延迟)
```

**解决方案:**

**方案1: 读写都走主库(强一致性)**

```sql
-- 对于刚写入的数据,短时间内从主库读取
-- 应用层实现读写分离时,标记需要读主库的请求

-- 示例伪代码
user_id = db_master.insert("INSERT INTO users...")
user = db_master.query("SELECT * FROM users WHERE user_id = ?", user_id)
```

**方案2: 半同步复制**

```ini
# my.cnf配置
[mysqld]
# 主库配置
rpl_semi_sync_master_enabled=1
rpl_semi_sync_master_timeout=1000  # 1秒超时

# 从库配置
rpl_semi_sync_slave_enabled=1
```

确保至少一个从库确认收到binlog后才返回成功。

**方案3: 检查复制延迟**

```sql
-- 在从库检查复制延迟
SHOW SLAVE STATUS\G

-- 查看Seconds_Behind_Master
-- 如果延迟超过阈值,路由到主库
```

**方案4: 使用中间件(ProxySQL/MaxScale)**

```sql
-- ProxySQL自动检测复制延迟
-- 配置规则:延迟超过5秒的从库不接收读请求
UPDATE mysql_servers 
SET max_replication_lag=5 
WHERE hostgroup_id=2;
```

**方案5: 应用层缓存**

```sql
-- 写入后缓存数据
INSERT INTO users (username) VALUES ('alice');
SET @user_id = LAST_INSERT_ID();

-- 将数据缓存到Redis,设置短期过期时间(如5秒)
-- SETEX user:@user_id 5 "{username: 'alice'}"

-- 读取时先查缓存,未命中再查数据库
```

### 3. 分布式事务场景下如何保证数据一致性?

分布式事务是指跨多个数据库或服务的事务操作,保证一致性有以下几种方案:

**方案1: 两阶段提交(2PC) - 强一致性**

```sql
-- 协调者
-- Phase 1: Prepare
XA START 'db1_xid';
UPDATE db1.accounts SET balance = balance - 100 WHERE account_id = 'A';
XA END 'db1_xid';
XA PREPARE 'db1_xid';

XA START 'db2_xid';
UPDATE db2.accounts SET balance = balance + 100 WHERE account_id = 'B';
XA END 'db2_xid';
XA PREPARE 'db2_xid';

-- Phase 2: Commit
IF all_prepared THEN
    XA COMMIT 'db1_xid';
    XA COMMIT 'db2_xid';
ELSE
    XA ROLLBACK 'db1_xid';
    XA ROLLBACK 'db2_xid';
END IF;
```

**优点:** 强一致性
**缺点:** 性能差,存在阻塞,协调者单点故障

**方案2: TCC模式 - 强一致性**

```sql
-- Try阶段:预留资源
-- DB1
UPDATE accounts SET balance = balance - 100, frozen = frozen + 100 
WHERE account_id = 'A';

-- DB2
UPDATE accounts SET balance = balance, reserved = reserved + 100
WHERE account_id = 'B';

-- Confirm阶段:确认提交
-- DB1
UPDATE accounts SET frozen = frozen - 100 WHERE account_id = 'A';

-- DB2
UPDATE accounts SET balance = balance + 100, reserved = reserved - 100
WHERE account_id = 'B';

-- Cancel阶段:取消回滚
-- DB1
UPDATE accounts SET balance = balance + 100, frozen = frozen - 100
WHERE account_id = 'A';

-- DB2
UPDATE accounts SET reserved = reserved - 100 WHERE account_id = 'B';
```

**优点:** 无锁等待,性能较好
**缺点:** 实现复杂,需要三个阶段

**方案3: SAGA模式 - 最终一致性**

```sql
-- 步骤1:扣款
START TRANSACTION;
UPDATE accounts SET balance = balance - 100 WHERE account_id = 'A';
INSERT INTO saga_log (saga_id, step, action, compensation)
VALUES ('saga_001', 1, 'debit_A', 'credit_A');
COMMIT;

-- 步骤2:入账
START TRANSACTION;
UPDATE accounts SET balance = balance + 100 WHERE account_id = 'B';
INSERT INTO saga_log (saga_id, step, action, compensation)
VALUES ('saga_001', 2, 'credit_B', 'debit_B');
COMMIT;

-- 如果步骤2失败,执行补偿
START TRANSACTION;
UPDATE accounts SET balance = balance + 100 WHERE account_id = 'A';
UPDATE saga_log SET status = 'compensated' 
WHERE saga_id = 'saga_001' AND step = 1;
COMMIT;
```

**优点:** 性能好,易于扩展
**缺点:** 只保证最终一致性,需要补偿逻辑

**方案4: 本地消息表 - 最终一致性**

```sql
-- DB1:扣款并记录消息
START TRANSACTION;

UPDATE accounts SET balance = balance - 100 WHERE account_id = 'A';

INSERT INTO outbox_messages (message_id, event_type, payload, status)
VALUES (UUID(), 'transfer', '{"to":"B","amount":100}', 'pending');

COMMIT;

-- 定时任务扫描消息表,发送到MQ
-- 消费者处理消息

-- DB2:消费消息入账
START TRANSACTION;
UPDATE accounts SET balance = balance + 100 WHERE account_id = 'B';
INSERT INTO processed_messages (message_id) VALUES ('xxx');
COMMIT;
```

**优点:** 简单可靠,基于消息最终一致
**缺点:** 有延迟,需要消息去重

### 4. 如何设计和实现乐观锁来避免并发更新冲突?

**基础版本号方案:**

```sql
-- 表设计
CREATE TABLE products (
    product_id INT PRIMARY KEY,
    product_name VARCHAR(100),
    price DECIMAL(10,2),
    stock INT,
    version INT NOT NULL DEFAULT 0,  -- 版本号
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 更新操作
-- 1. 读取数据和版本号
SELECT product_id, stock, version 
INTO @id, @stock, @ver
FROM products WHERE product_id = 1001;

-- 2. 业务逻辑计算
SET @new_stock = @stock - 10;

-- 3. 更新时验证版本号
UPDATE products 
SET stock = @new_stock, version = version + 1
WHERE product_id = 1001 AND version = @ver;

-- 4. 检查更新结果
IF ROW_COUNT() = 0 THEN
    -- 版本号不匹配,数据已被其他事务修改
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Concurrent modification detected';
END IF;
```

**时间戳方案:**

```sql
-- 使用时间戳代替版本号
CREATE TABLE products (
    product_id INT PRIMARY KEY,
    stock INT,
    last_modified TIMESTAMP(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
);

-- 更新操作
SELECT stock, last_modified INTO @stock, @timestamp
FROM products WHERE product_id = 1001;

UPDATE products 
SET stock = @stock - 10
WHERE product_id = 1001 AND last_modified = @timestamp;

IF ROW_COUNT() = 0 THEN
    -- 数据已被修改
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Data has been modified';
END IF;
```

**CAS(Compare-And-Swap)方案:**

```sql
-- 直接比较旧值
UPDATE products 
SET stock = @new_value, version = version + 1
WHERE product_id = 1001 
  AND stock = @old_value  -- 比较期望值
  AND version = @expected_version;
```

**应用层完整实现(Java示例):**

```java
public class OptimisticLockService {
    private static final int MAX_RETRIES = 3;
    
    public void updateStock(int productId, int quantity) {
        int retries = 0;
        
        while (retries < MAX_RETRIES) {
            try {
                // 1. 读取当前数据
                Product product = productRepository.findById(productId);
                int currentStock = product.getStock();
                int currentVersion = product.getVersion();
                
                // 2. 业务逻辑
                if (currentStock < quantity) {
                    throw new InsufficientStockException();
                }
                int newStock = currentStock - quantity;
                
                // 3. 乐观锁更新
                int updated = jdbcTemplate.update(
                    "UPDATE products SET stock = ?, version = version + 1 " +
                    "WHERE product_id = ? AND version = ?",
                    newStock, productId, currentVersion
                );
                
                if (updated == 1) {
                    // 更新成功
                    return;
                } else {
                    // 版本冲突,重试
                    retries++;
                    Thread.sleep(10 * retries);  // 指数退避
                }
                
            } catch (Exception e) {
                retries++;
                if (retries >= MAX_RETRIES) {
                    throw new ConcurrentUpdateException("Update failed after retries");
                }
            }
        }
    }
}
```

**乐观锁 vs 悲观锁选择:**

- **读多写少**: 使用乐观锁,减少锁竞争
- **写多读少**: 使用悲观锁,避免频繁重试
- **冲突概率低**: 使用乐观锁
- **冲突概率高**: 使用悲观锁
- **长事务**: 避免悲观锁(长时间持锁)

### 5. 触发器在维护数据一致性时有哪些最佳实践和注意事项?

**最佳实践:**

**1. 保持触发器简单快速**

```sql
-- 好的触发器:简单的数据同步
DELIMITER //
CREATE TRIGGER trg_update_order_total
AFTER INSERT ON order_items
FOR EACH ROW
BEGIN
    UPDATE orders 
    SET total_amount = (
        SELECT SUM(quantity * price) 
        FROM order_items 
        WHERE order_id = NEW.order_id
    )
    WHERE order_id = NEW.order_id;
END //
DELIMITER ;

-- 避免:复杂的业务逻辑
-- 不要在触发器中执行复杂查询、调用外部服务等
```

**2. 避免触发器递归**

```sql
-- 可能导致递归的例子
CREATE TRIGGER trg_products_update
AFTER UPDATE ON products
FOR EACH ROW
BEGIN
    -- 危险:可能触发自身
    UPDATE products SET updated_at = NOW() WHERE product_id = NEW.product_id;
END;

-- 安全的做法:使用BEFORE触发器
CREATE TRIGGER trg_products_before_update
BEFORE UPDATE ON products
FOR EACH ROW
BEGIN
    SET NEW.updated_at = NOW();  -- 修改NEW变量,不触发其他UPDATE
END;
```

**3. 正确处理错误**

```sql
DELIMITER //
CREATE TRIGGER trg_check_stock_before_order
BEFORE INSERT ON order_items
FOR EACH ROW
BEGIN
    DECLARE v_stock INT;
    
    SELECT stock INTO v_stock
    FROM products
    WHERE product_id = NEW.product_id;
    
    IF v_stock < NEW.quantity THEN
        SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = 'Insufficient stock for product';
    END IF;
END //
DELIMITER ;
```

**注意事项:**

**1. 性能影响**

```sql
-- 触发器会降低DML操作性能
-- 大批量数据操作时考虑禁用触发器

-- 临时禁用触发器(需要SUPER权限)
SET @DISABLE_TRIGGERS = 1;

-- 在触发器中检查
IF @DISABLE_TRIGGERS IS NULL OR @DISABLE_TRIGGERS = 0 THEN
    -- 执行触发器逻辑
END IF;
```

**2. 事务中的触发器**

```sql
-- 触发器在同一事务中执行
START TRANSACTION;

INSERT INTO orders (customer_id) VALUES (1001);
-- 触发器自动执行,如果触发器失败,整个事务回滚

COMMIT;
```

**3. 避免过度使用**

```sql
-- 不推荐:所有业务逻辑都在触发器中
-- 推荐:触发器只用于:
-- - 自动更新时间戳
-- - 维护派生数据(如统计字段)
-- - 数据验证
-- - 审计日志

-- 复杂业务逻辑应该在应用层或存储过程中
```

**4. 文档化触发器**

```sql
-- 为触发器添加注释
CREATE TRIGGER trg_maintain_inventory_count
AFTER INSERT ON inventory_transactions
FOR EACH ROW
-- 功能:自动更新产品库存数量
-- 作者:Zhang San
-- 创建时间:2024-01-15
-- 依赖:products表的stock字段
BEGIN
    UPDATE products 
    SET stock = stock + NEW.quantity
    WHERE product_id = NEW.product_id;
END;
```
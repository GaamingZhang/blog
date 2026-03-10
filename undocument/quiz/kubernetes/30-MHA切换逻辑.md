---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - MySQL
  - 高可用
tag:
  - MySQL
  - MHA
  - 高可用
  - 故障切换
---

# MHA切换逻辑详解

## 概述

MHA（Master High Availability）是一套优秀的MySQL高可用切换软件，由Perl语言编写，能够在MySQL主从复制环境中实现主库故障自动切换。本文将详细介绍MHA的切换逻辑和工作原理。

## MHA架构组件

### 组件架构图

```
+------------------+     +------------------+     +------------------+
|   MHA Manager    |     |   MHA Node       |     |   MySQL Server   |
+------------------+     +------------------+     +------------------+
| - masterha_      |     | - save_binary_   |     | - Master         |
|   manager        |     |   logs           |     | - Slave1         |
| - masterha_      |     | - apply_diff_    |     | - Slave2         |
|   check_ssh      |     |   relay_logs     |     | - Slave3         |
| - masterha_      |     | - purge_relay_   |     +------------------+
|   check_repl     |     |   logs           |              |
| - masterha_      |     +------------------+              |
|   master_monitor |              |                        |
+------------------+              |                        |
         |                        |                        |
         +------------------------+------------------------+
                              SSH连接
```

### 核心组件说明

| 组件 | 角色 | 说明 |
|------|------|------|
| MHA Manager | 管理节点 | 监控Master，执行故障切换 |
| MHA Node | 数据节点 | 部署在每个MySQL服务器上 |
| masterha_manager | 管理程序 | 主监控程序 |
| masterha_master_monitor | 监控程序 | 检测Master是否存活 |
| masterha_master_switch | 切换程序 | 执行故障切换 |

## MHA切换流程

### 整体切换流程图

```
                    +------------------+
                    |   Master故障     |
                    +--------+---------+
                             |
                             v
                    +--------+---------+
                    |   检测到故障     |
                    | (连续ping失败)   |
                    +--------+---------+
                             |
                             v
                    +--------+---------+
                    |   尝试SSH连接    |
                    |   Master主机     |
                    +--------+---------+
                             |
              +--------------+--------------+
              |                             |
              v                             v
     +--------+--------+          +--------+--------+
     | SSH可达         |          | SSH不可达        |
     | (可挽救场景)    |          | (完全故障)       |
     +--------+--------+          +--------+--------+
              |                             |
              v                             v
     +--------+--------+          +--------+--------+
     | 保存Master      |          | 直接进入         |
     | 二进制日志      |          | 切换流程         |
     +--------+--------+          +--------+--------+
              |                             |
              +--------------+--------------+
                             |
                             v
                    +--------+---------+
                    |   选举新Master   |
                    +--------+---------+
                             |
                             v
                    +--------+---------+
                    |   同步差异日志   |
                    +--------+---------+
                             |
                             v
                    +--------+---------+
                    |   提升新Master   |
                    +--------+---------+
                             |
                             v
                    +--------+---------+
                    |   重构主从关系   |
                    +--------+---------+
```

### 详细切换步骤

```
步骤1: 故障检测
+------------------+
| MHA Manager持续  |
| ping Master      |
| (默认3次失败)    |
+------------------+

步骤2: SSH可达性检查
+------------------+
| 尝试SSH连接      |
| Master主机       |
| 判断是否可获取   |
| 二进制日志       |
+------------------+

步骤3: 选举新Master
+------------------+
| 根据配置优先级   |
| 和数据完整性     |
| 选择最佳Slave    |
+------------------+

步骤4: 同步差异日志
+------------------+
| 将差异relay log  |
| 应用到新Master   |
| 确保数据完整     |
+------------------+

步骤5: 提升新Master
+------------------+
| 执行CHANGE       |
| MASTER TO        |
| 开启写入         |
+------------------+

步骤6: 重构主从关系
+------------------+
| 其他Slave指向    |
| 新Master         |
| 恢复复制         |
+------------------+
```

## Master选举机制

### 选举优先级

```
+------------------+------------------------------------------+
|   优先级因素     |   说明                                   |
+------------------+------------------------------------------+
| candidate_master | 配置文件中指定的候选Master优先级         |
| check_repl_delay | 是否检查复制延迟（默认检查）             |
| 最新数据         | relay log最新的Slave优先                 |
| 复制过滤规则     | 没有复制过滤的优先                       |
+------------------+------------------------------------------+
```

### 配置示例

```ini
# /etc/mha/app1.cnf

[server default]
manager_workdir=/var/log/mha/app1
manager_log=/var/log/mha/app1/manager.log
user=mha_user
password=mha_password
repl_user=repl
repl_password=repl_password
ssh_user=root
ping_interval=3

[server1]
hostname=192.168.1.101
port=3306
candidate_master=1
check_repl_delay=0

[server2]
hostname=192.168.1.102
port=3306
candidate_master=1

[server3]
hostname=192.168.1.103
port=3306
candidate_master=0
no_master=1
```

### 选举流程伪代码

```
function elect_new_master(servers):
    # 1. 排除no_master=1的服务器
    candidates = filter(servers, no_master != 1)
    
    # 2. 检查复制延迟
    if check_repl_delay:
        candidates = filter(candidates, delay < threshold)
    
    # 3. 按candidate_master排序
    sort(candidates, by=candidate_master, desc=True)
    
    # 4. 选择数据最新的
    for candidate in candidates:
        if has_latest_relay_log(candidate):
            return candidate
    
    # 5. 返回第一个候选者
    return candidates[0]
```

## 日志同步机制

### 差异日志处理

```
场景：Master突然宕机，Slave的relay log可能不完整

+------------------+          +------------------+
|     Master       |          |     Slave        |
|   (已宕机)       |          |   (候选Master)   |
+------------------+          +------------------+
| binlog:          |          | relay log:       |
| mysql-bin.000100 |          | mysql-relay.0001 |
|   - 位置: 500    |          |   - 已执行: 400  |
|   - 最新: 800    |          |   - 差异: 100    |
+------------------+          +------------------+
         |                             |
         | SSH可达时                   |
         v                             |
+------------------+                   |
| save_binary_logs |                   |
| 提取差异binlog   |-------------------+
| 位置: 400-800    |
+------------------+
         |
         v
+------------------+
| apply_diff_relay |
| _logs            |
| 应用差异日志     |
+------------------+
```

### save_binary_logs脚本

```perl
# 当Master的SSH可达时，保存二进制日志

# 1. 确定最后的binlog位置
my $last_binlog = get_last_applied_binlog_from_slaves();

# 2. 从Master保存binlog
save_binary_logs --binlog_dir=/var/lib/mysql --last_file=mysql-bin.000100 --target_dir=/tmp/binlog_save

# 3. 将差异日志传输到新Master
scp /tmp/binlog_save/* new_master:/tmp/
```

### apply_diff_relay_logs脚本

```perl
# 在新Master上应用差异日志

# 1. 生成差异relay log
apply_diff_relay_logs --command=generate_diff --slave_host=slave1 --slave_port=3306

# 2. 应用差异日志
apply_diff_relay_logs --command=apply --diff_file=/tmp/diff.log
```

## 切换模式

### 自动切换模式

```bash
# 启动MHA Manager
$ masterha_manager --conf=/etc/mha/app1.cnf

# MHA Manager会持续监控Master
# 检测到故障后自动执行切换

# 日志输出示例
[info] Checking master status..
[info] Master 192.168.1.101(192.168.1.101:3306) is down!
[info] Checking SSH connection to the master..
[info] HealthCheck: SSH to 192.168.1.101 is reachable.
[info] Starting master failover 192.168.1.101->192.168.1.102..
[info] 192.168.1.102(192.168.1.102:3306): OK: Applying all logs succeeded.
[info] 192.168.1.103(192.168.1.103:3306): OK: Slave started, replicating from 192.168.1.102.
[info] Master failover to 192.168.1.102(192.168.1.102:3306) completed successfully.
```

### 手动切换模式

```bash
# 手动执行切换（在线切换）
$ masterha_master_switch --conf=/etc/mha/app1.cnf --master_state=alive --new_master_host=192.168.1.102 --new_master_port=3306

# 手动故障切换
$ masterha_master_switch --conf=/etc/mha/app1.cnf --master_state=dead --dead_master_host=192.168.1.101 --dead_master_port=3306 --new_master_host=192.168.1.102
```

### 在线切换流程

```
+------------------+     +------------------+
|   原Master       |     |   新Master       |
|   192.168.1.101  |     |   192.168.1.102  |
+------------------+     +------------------+
         |                       |
         | 1. 设置只读           |
         |    SET GLOBAL         |
         |    read_only=ON       |
         |                       |
         | 2. 等待Slave同步      |
         |    SHOW PROCESSLIST   |
         |                       |
         | 3. 停止复制线程       |
         |    STOP SLAVE         |
         |                       |
         | 4. 提升新Master       |
         |    RESET SLAVE ALL    |
         |    SET GLOBAL         |
         |    read_only=OFF      |
         |                       |
         | 5. 重构主从关系       |
         |    CHANGE MASTER TO   |
         |                       |
         v                       v
+------------------+     +------------------+
|   新Slave        |     |   新Master       |
|   192.168.1.101  |     |   192.168.1.102  |
+------------------+     +------------------+
```

## 切换脚本配置

### 切换前脚本

```ini
# /etc/mha/app1.cnf
[server default]
# 切换前执行的脚本
master_ip_failover_script=/usr/local/bin/master_ip_failover
shutdown_script=/usr/local/bin/shutdown_script
```

```bash
#!/usr/bin/env perl
# /usr/local/bin/master_ip_failover

use strict;
use warnings;
use Getopt::Long;

my $command;
my $ssh_user;
my $orig_master_host;
my $new_master_host;
my $new_master_port = 3306;
my $vip = '192.168.1.100/24';
my $interface = 'eth0';

GetOptions(
    'command=s'          => \$command,
    'ssh_user=s'         => \$ssh_user,
    'orig_master_host=s' => \$orig_master_host,
    'new_master_host=s'  => \$new_master_host,
    'new_master_port=i'  => \$new_master_port,
);

if ($command eq "stop" || $command eq "stopssh") {
    # 在原Master上移除VIP
    my $cmd = "ssh $ssh_user\@$orig_master_host \"ip addr del $vip dev $interface\"";
    system($cmd);
    
} elsif ($command eq "start") {
    # 在新Master上添加VIP
    my $cmd = "ssh $ssh_user\@$new_master_host \"ip addr add $vip dev $interface\"";
    system($cmd);
}
```

### 切换后脚本

```bash
#!/bin/bash
# /usr/local/bin/send_notification.sh

# 发送通知
send_notification() {
    local subject="MHA Failover Completed"
    local body="Master failover completed.\nNew Master: $1\nOld Master: $2"
    
    echo -e "$body" | mail -s "$subject" admin@example.com
    
    # 或者调用企业微信/钉钉webhook
    curl -X POST "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx" \
        -H "Content-Type: application/json" \
        -d "{\"msgtype\":\"text\",\"text\":{\"content\":\"$body\"}}"
}

send_notification "$NEW_MASTER_HOST" "$ORIG_MASTER_HOST"
```

## 常见切换场景

### 场景一：Master主机宕机但SSH可达

```
+------------------+
| Master主机宕机   |
| MySQL进程崩溃    |
| SSH仍然可达      |
+--------+---------+
         |
         v
+--------+---------+
| 1. SSH连接Master |
| 2. 保存binlog    |
| 3. 传输差异日志  |
| 4. 应用到新Master|
| 5. 完成切换      |
+------------------+

数据丢失：最小（仅丢失最后时刻的事务）
```

### 场景二：Master主机完全不可达

```
+------------------+
| Master主机断电   |
| 网络完全中断     |
| SSH不可达        |
+--------+---------+
         |
         v
+--------+---------+
| 1. 无法获取binlog|
| 2. 直接选举      |
|    最新Slave     |
| 3. 其他Slave同步 |
| 4. 完成切换      |
+------------------+

数据丢失：可能丢失未同步的事务
```

### 场景三：在线维护切换

```bash
# 计划维护切换
$ masterha_master_switch --conf=/etc/mha/app1.cnf \
    --master_state=alive \
    --new_master_host=192.168.1.102 \
    --orig_master_is_new_slave \
    --running_updates_limit=10000

# 参数说明：
# --master_state=alive      原Master在线
# --orig_master_is_new_slave 原Master变为新Slave
# --running_updates_limit    允许的最大更新时间
```

## 切换验证

### 切换后检查清单

```bash
# 1. 检查新Master状态
$ mysql -h new_master -e "SHOW MASTER STATUS\G"

# 2. 检查复制状态
$ mysql -h slave -e "SHOW SLAVE STATUS\G"

# 3. 检查VIP
$ ip addr show | grep 192.168.1.100

# 4. 检查应用连接
$ mysql -h vip -e "SELECT @@hostname"

# 5. 检查MHA日志
$ tail -100 /var/log/mha/app1/manager.log
```

### 切换后恢复

```bash
# 1. 修复原Master
# 2. 配置为Slave
mysql> CHANGE MASTER TO
    -> MASTER_HOST='192.168.1.102',
    -> MASTER_PORT=3306,
    -> MASTER_USER='repl',
    -> MASTER_PASSWORD='repl_password',
    -> MASTER_AUTO_POSITION=1;
mysql> START SLAVE;

# 3. 更新MHA配置
# 修改 /etc/mha/app1.cnf

# 4. 重启MHA Manager
$ masterha_manager --conf=/etc/mha/app1.cnf
```

## 最佳实践

### 1. 配置建议

```ini
[server default]
# ping间隔，建议3秒
ping_interval=3

# ping类型，建议使用insert
ping_type=INSERT

# 切换超时时间
shutdown_timeout=5

# 复制用户权限检查
check_repl_priv=1
```

### 2. 监控告警

```yaml
# Prometheus告警规则
groups:
  - name: mha_alerts
    rules:
      - alert: MHAManagerDown
        expr: mha_manager_up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "MHA Manager is down"

      - alert: MySQLReplicationLag
        expr: mysql_slave_lag_seconds > 30
        for: 2m
        labels:
          severity: warning
```

### 3. 定期演练

```bash
# 定期执行切换演练
# 1. 选择低峰期
# 2. 通知相关人员
# 3. 执行在线切换
# 4. 验证应用正常
# 5. 记录演练结果
```

## 参考资源

- [MHA GitHub](https://github.com/yoshinorim/mha4mysql-manager)
- [MHA官方文档](https://github.com/yoshinorim/mha4mysql-manager/wiki)
- [MySQL High Availability Solutions](https://dev.mysql.com/doc/mysql-ha-scalability/en/)

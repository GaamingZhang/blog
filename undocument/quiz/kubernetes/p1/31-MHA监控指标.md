---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - MySQL
  - 监控
tag:
  - MySQL
  - MHA
  - 监控
  - Prometheus
---

# MHA监控指标详解

## 概述

MHA（Master High Availability）作为MySQL高可用方案，需要完善的监控体系来确保其正常运行。本文将详细介绍MHA相关的监控指标，包括MySQL复制状态、MHA Manager状态以及相关的告警配置。

## 监控架构

```
+------------------+     +------------------+     +------------------+
|   Prometheus     |     |   Node Exporter  |     |  MySQL Exporter  |
+------------------+     +------------------+     +------------------+
         |                        |                        |
         |                        |                        |
         v                        v                        v
+------------------+     +------------------+     +------------------+
|   MHA Manager    |     |   MySQL Master   |     |   MySQL Slave    |
|   监控脚本       |     |   192.168.1.101  |     |   192.168.1.102  |
+------------------+     +------------------+     +------------------+
         |                        |                        |
         +------------------------+------------------------+
                                  |
                                  v
                         +------------------+
                         |   Alertmanager   |
                         +------------------+
```

## 核心监控指标

### 1. MHA Manager状态指标

```
+------------------------+----------------------------------------+
|        指标名称        |                 说明                   |
+------------------------+----------------------------------------+
| mha_manager_up         | MHA Manager进程是否运行 (1/0)          |
| mha_manager_last_check | 上次检查时间戳                          |
| mha_master_status      | 当前Master状态 (1=正常, 0=异常)         |
| mha_failover_count     | 累计切换次数                            |
| mha_failover_last_time | 上次切换时间戳                          |
+------------------------+----------------------------------------+
```

### 2. MySQL复制状态指标

```
+-----------------------------+-------------------------------------+
|          指标名称           |                说明                 |
+-----------------------------+-------------------------------------+
| mysql_slave_status          | Slave状态 (1=正常, 0=异常)           |
| mysql_slave_io_running      | IO线程状态 (1=Yes, 0=No)             |
| mysql_slave_sql_running     | SQL线程状态 (1=Yes, 0=No)            |
| mysql_slave_lag_seconds     | 复制延迟秒数                          |
| mysql_master_binlog_file    | Master当前binlog文件                 |
| mysql_master_binlog_pos     | Master当前binlog位置                 |
| mysql_slave_relay_log_file  | Slave当前relay log文件               |
| mysql_slave_relay_log_pos   | Slave当前relay log位置               |
+-----------------------------+-------------------------------------+
```

### 3. MySQL服务器状态指标

```
+-----------------------------+-------------------------------------+
|          指标名称           |                说明                 |
+-----------------------------+-------------------------------------+
| mysql_up                    | MySQL服务是否运行                    |
| mysql_connections           | 当前连接数                           |
| mysql_threads_running       | 运行中的线程数                        |
| mysql_queries_total         | 查询总数                             |
| mysql_slow_queries_total    | 慢查询总数                           |
| mysql_innodb_buffer_pool    | InnoDB缓冲池使用率                   |
| mysql_innodb_row_lock_waits | 行锁等待次数                         |
+-----------------------------+-------------------------------------+
```

## Prometheus配置

### MySQL Exporter部署

```yaml
# docker-compose.yml
version: '3'
services:
  mysql-exporter:
    image: prom/mysqld-exporter:latest
    container_name: mysql-exporter
    ports:
      - "9104:9104"
    environment:
      - DATA_SOURCE_NAME=exporter:exporter_password@(192.168.1.101:3306)/
    command:
      - '--collect.info_schema.processlist'
      - '--collect.info_schema.innodb_metrics'
      - '--collect.info_schema.tablestats'
      - '--collect.info_schema.userstats'
      - '--collect.engine_innodb_status'
      - '--collect.slave_status'
```

### Prometheus抓取配置

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'mysql'
    static_configs:
      - targets:
          - '192.168.1.101:9104'
          - '192.168.1.102:9104'
          - '192.168.1.103:9104'
        labels:
          group: 'mysql-cluster'

  - job_name: 'mha-manager'
    static_configs:
      - targets:
          - '192.168.1.100:9100'
        labels:
          role: 'mha-manager'
```

## 自定义MHA监控脚本

### MHA Manager监控脚本

```python
#!/usr/bin/env python3
# mha_exporter.py - MHA监控指标导出脚本

import os
import re
import time
import subprocess
from prometheus_client import start_http_server, Gauge, Counter

# 定义指标
MHA_MANAGER_UP = Gauge('mha_manager_up', 'MHA Manager process status')
MHA_MASTER_STATUS = Gauge('mha_master_status', 'Current master status')
MHA_SLAVE_COUNT = Gauge('mha_slave_count', 'Number of healthy slaves')
MHA_FAILOVER_COUNT = Counter('mha_failover_count_total', 'Total failover count')
MHA_LAST_CHECK = Gauge('mha_last_check_timestamp', 'Last check timestamp')
MHA_SLAVE_LAG = Gauge('mha_slave_lag_seconds', 'Slave replication lag', ['slave_host'])

def check_mha_manager():
    """检查MHA Manager进程是否运行"""
    try:
        result = subprocess.run(
            ['pgrep', '-f', 'masterha_manager'],
            capture_output=True,
            text=True
        )
        return 1 if result.returncode == 0 else 0
    except Exception:
        return 0

def parse_mha_status(config_file):
    """解析MHA状态"""
    try:
        result = subprocess.run(
            ['masterha_check_status', '--conf', config_file],
            capture_output=True,
            text=True
        )
        output = result.stdout
        
        # 解析状态
        if 'running(0:PING_OK)' in output:
            return {'master_status': 1, 'status': 'running'}
        elif 'running(0:PING_FAILED)' in output:
            return {'master_status': 0, 'status': 'master_down'}
        else:
            return {'master_status': 0, 'status': 'unknown'}
    except Exception:
        return {'master_status': 0, 'status': 'error'}

def check_slave_status(slave_host, slave_port=3306):
    """检查Slave状态"""
    try:
        result = subprocess.run(
            ['mysql', '-h', slave_host, '-P', str(slave_port), '-e',
             "SHOW SLAVE STATUS\\G"],
            capture_output=True,
            text=True
        )
        
        output = result.stdout
        lag = 0
        io_running = False
        sql_running = False
        
        for line in output.split('\n'):
            if 'Seconds_Behind_Master:' in line:
                lag = int(line.split(':')[1].strip() or 0)
            elif 'Slave_IO_Running: Yes' in line:
                io_running = True
            elif 'Slave_SQL_Running: Yes' in line:
                sql_running = True
        
        return {
            'lag': lag,
            'healthy': io_running and sql_running
        }
    except Exception:
        return {'lag': -1, 'healthy': False}

def main():
    config_file = os.environ.get('MHA_CONFIG', '/etc/mha/app1.cnf')
    slaves = os.environ.get('MHA_SLAVES', '192.168.1.102,192.168.1.103').split(',')
    
    start_http_server(9100)
    print("MHA Exporter started on port 9100")
    
    while True:
        # 更新MHA Manager状态
        MHA_MANAGER_UP.set(check_mha_manager())
        
        # 更新Master状态
        status = parse_mha_status(config_file)
        MHA_MASTER_STATUS.set(status['master_status'])
        
        # 更新检查时间
        MHA_LAST_CHECK.set(time.time())
        
        # 检查Slave状态
        healthy_slaves = 0
        for slave in slaves:
            slave_status = check_slave_status(slave)
            if slave_status['healthy']:
                healthy_slaves += 1
            MHA_SLAVE_LAG.labels(slave_host=slave).set(slave_status['lag'])
        
        MHA_SLAVE_COUNT.set(healthy_slaves)
        
        time.sleep(10)

if __name__ == '__main__':
    main()
```

### Systemd服务配置

```ini
# /etc/systemd/system/mha-exporter.service
[Unit]
Description=MHA Prometheus Exporter
After=network.target

[Service]
Type=simple
User=root
Environment="MHA_CONFIG=/etc/mha/app1.cnf"
Environment="MHA_SLAVES=192.168.1.102,192.168.1.103"
ExecStart=/usr/local/bin/mha_exporter.py
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## Prometheus告警规则

### MHA相关告警

```yaml
# /etc/prometheus/rules/mha_alerts.yml
groups:
  - name: mha_alerts
    rules:
      - alert: MHAManagerDown
        expr: mha_manager_up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "MHA Manager进程已停止"
          description: "MHA Manager进程在实例 {{ $labels.instance }} 上已停止运行，请立即检查"

      - alert: MHAMasterDown
        expr: mha_master_status == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "MySQL Master状态异常"
          description: "MHA检测到Master状态异常，可能正在执行故障切换"

      - alert: MHAAllSlavesDown
        expr: mha_slave_count == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "所有MySQL Slave异常"
          description: "没有健康的Slave节点，高可用架构已失效"

      - alert: MHALongFailover
        expr: |
          time() - mha_failover_last_timestamp > 300
          and mha_master_status == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "MHA切换时间过长"
          description: "MHA故障切换已超过5分钟仍未完成"
```

### MySQL复制告警

```yaml
# /etc/prometheus/rules/mysql_replication_alerts.yml
groups:
  - name: mysql_replication_alerts
    rules:
      - alert: MySQLReplicationLag
        expr: mysql_slave_lag_seconds > 30
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "MySQL复制延迟过高"
          description: "Slave {{ $labels.instance }} 复制延迟 {{ $value }} 秒"

      - alert: MySQLReplicationLagCritical
        expr: mysql_slave_lag_seconds > 120
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "MySQL复制延迟严重"
          description: "Slave {{ $labels.instance }} 复制延迟 {{ $value }} 秒，可能导致数据不一致"

      - alert: MySQLReplicationBroken
        expr: |
          mysql_slave_io_running == 0 or mysql_slave_sql_running == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "MySQL复制线程异常"
          description: "Slave {{ $labels.instance }} 的IO或SQL线程已停止"

      - alert: MySQLReplicationStopped
        expr: |
          mysql_slave_status == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "MySQL复制已停止"
          description: "Slave {{ $labels.instance }} 复制状态异常"
```

### MySQL服务器告警

```yaml
# /etc/prometheus/rules/mysql_server_alerts.yml
groups:
  - name: mysql_server_alerts
    rules:
      - alert: MySQLDown
        expr: mysql_up == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "MySQL服务不可用"
          description: "MySQL实例 {{ $labels.instance }} 无法连接"

      - alert: MySQLTooManyConnections
        expr: |
          mysql_global_status_threads_connected / mysql_global_variables_max_connections > 0.8
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "MySQL连接数过高"
          description: "MySQL {{ $labels.instance }} 连接使用率超过80%"

      - alert: MySQLSlowQueries
        expr: rate(mysql_global_status_slow_queries[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "MySQL慢查询过多"
          description: "MySQL {{ $labels.instance }} 每分钟慢查询数: {{ $value }}"

      - alert: MySQLInnodbLogWaits
        expr: rate(mysql_global_status_innodb_log_waits[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "InnoDB日志等待过多"
          description: "InnoDB日志写入等待过多，可能需要调整日志文件大小"
```

## Grafana Dashboard

### Dashboard JSON配置

```json
{
  "dashboard": {
    "title": "MHA MySQL Cluster Monitor",
    "panels": [
      {
        "title": "MHA Manager Status",
        "type": "stat",
        "targets": [
          {
            "expr": "mha_manager_up",
            "legendFormat": "Manager Status"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "mappings": [
              {
                "type": "value",
                "options": {
                  "0": {"text": "DOWN", "color": "red"},
                  "1": {"text": "UP", "color": "green"}
                }
              }
            ]
          }
        }
      },
      {
        "title": "Master Status",
        "type": "stat",
        "targets": [
          {
            "expr": "mha_master_status",
            "legendFormat": "Master"
          }
        ]
      },
      {
        "title": "Slave Replication Lag",
        "type": "graph",
        "targets": [
          {
            "expr": "mysql_slave_lag_seconds",
            "legendFormat": "{{ instance }}"
          }
        ],
        "alert": {
          "conditions": [
            {
              "evaluator": {"type": "gt", "params": [30]}
            }
          ]
        }
      },
      {
        "title": "Slave IO/SQL Thread Status",
        "type": "table",
        "targets": [
          {
            "expr": "mysql_slave_io_running",
            "format": "table",
            "instant": true
          },
          {
            "expr": "mysql_slave_sql_running",
            "format": "table",
            "instant": true
          }
        ]
      },
      {
        "title": "MySQL Connections",
        "type": "graph",
        "targets": [
          {
            "expr": "mysql_global_status_threads_connected",
            "legendFormat": "{{ instance }}"
          },
          {
            "expr": "mysql_global_variables_max_connections",
            "legendFormat": "{{ instance }} max"
          }
        ]
      }
    ]
  }
}
```

## 监控指标采集SQL

### 复制状态查询

```sql
-- 查看Slave状态
SHOW SLAVE STATUS\G

-- 关键字段
-- Slave_IO_Running: Yes/No
-- Slave_SQL_Running: Yes/No
-- Seconds_Behind_Master: 延迟秒数
-- Master_Log_File: 当前读取的Master binlog
-- Read_Master_Log_Pos: 当前读取的位置
-- Relay_Master_Log_File: 当前执行的Master binlog
-- Exec_Master_Log_Pos: 当前执行的位置
```

### Master状态查询

```sql
-- 查看Master状态
SHOW MASTER STATUS\G

-- 查看连接的Slave
SHOW SLAVE HOSTS;

-- 查看复制用户权限
SHOW GRANTS FOR 'repl'@'%';
```

### 性能指标查询

```sql
-- 查看当前连接数
SHOW STATUS LIKE 'Threads_connected';
SHOW STATUS LIKE 'Max_used_connections';
SHOW VARIABLES LIKE 'max_connections';

-- 查看慢查询
SHOW STATUS LIKE 'Slow_queries';

-- 查看InnoDB状态
SHOW ENGINE INNODB STATUS\G

-- 查看锁等待
SELECT * FROM information_schema.INNODB_LOCK_WAITS;
SELECT * FROM information_schema.INNODB_LOCKS;
```

## 监控检查脚本

```bash
#!/bin/bash
# check_mha_status.sh - MHA状态检查脚本

MHA_CONF="/etc/mha/app1.cnf"
LOG_FILE="/var/log/mha/check.log"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') $1" | tee -a $LOG_FILE
}

# 检查MHA Manager进程
check_manager_process() {
    if pgrep -f "masterha_manager" > /dev/null; then
        log "[OK] MHA Manager进程运行中"
        return 0
    else
        log "[CRITICAL] MHA Manager进程未运行"
        return 1
    fi
}

# 检查Master状态
check_master_status() {
    STATUS=$(masterha_check_status --conf $MHA_CONF 2>&1)
    if echo "$STATUS" | grep -q "PING_OK"; then
        log "[OK] Master状态正常"
        return 0
    else
        log "[CRITICAL] Master状态异常: $STATUS"
        return 1
    fi
}

# 检查Slave状态
check_slave_status() {
    SLAVES=$(grep -A5 "\[server" $MHA_CONF | grep hostname | awk -F= '{print $2}')
    
    for slave in $SLAVES; do
        IO_RUNNING=$(mysql -h $slave -e "SHOW SLAVE STATUS\G" 2>/dev/null | grep "Slave_IO_Running:" | awk '{print $2}')
        SQL_RUNNING=$(mysql -h $slave -e "SHOW SLAVE STATUS\G" 2>/dev/null | grep "Slave_SQL_Running:" | awk '{print $2}')
        LAG=$(mysql -h $slave -e "SHOW SLAVE STATUS\G" 2>/dev/null | grep "Seconds_Behind_Master:" | awk '{print $2}')
        
        if [ "$IO_RUNNING" = "Yes" ] && [ "$SQL_RUNNING" = "Yes" ]; then
            log "[OK] Slave $slave 状态正常, 延迟: ${LAG}s"
        else
            log "[WARNING] Slave $slave 状态异常: IO=$IO_RUNNING, SQL=$SQL_RUNNING"
        fi
    done
}

# 主检查流程
main() {
    log "========== MHA状态检查开始 =========="
    check_manager_process
    check_master_status
    check_slave_status
    log "========== MHA状态检查结束 =========="
}

main
```

## 监控最佳实践

### 1. 监控指标优先级

```
+------------------+------------+
|     指标         |   优先级   |
+------------------+------------+
| MHA Manager状态  |   P0       |
| Master状态       |   P0       |
| 复制线程状态     |   P0       |
| 复制延迟         |   P1       |
| 连接数           |   P1       |
| 慢查询           |   P2       |
+------------------+------------+
```

### 2. 告警通知配置

```yaml
# alertmanager.yml
route:
  group_by: ['alertname', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 1h
  receiver: 'default'
  routes:
    - match:
        severity: critical
      receiver: 'critical'
    - match:
        severity: warning
      receiver: 'warning'

receivers:
  - name: 'critical'
    webhook_configs:
      - url: 'http://webhook-server/alert'
    email_configs:
      - to: 'dba@example.com'
        from: 'alert@example.com'
        smarthost: 'smtp.example.com:25'
```

### 3. 定期健康检查

```bash
# 添加到crontab
*/5 * * * * /usr/local/bin/check_mha_status.sh
0 8 * * * /usr/local/bin/mha_health_check.sh --full
```

## 参考资源

- [MySQL Exporter](https://github.com/prometheus/mysqld_exporter)
- [MHA Documentation](https://github.com/yoshinorim/mha4mysql-manager/wiki)
- [Prometheus Alerting Rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)

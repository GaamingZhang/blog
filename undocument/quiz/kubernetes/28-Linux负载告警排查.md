---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Linux
  - 运维
tag:
  - Linux
  - 性能排查
  - 监控
---

# Linux主机负载告警排查思路

## 概述

Linux系统负载（Load Average）是衡量系统运行状态的重要指标。当收到负载告警时，需要系统性地排查问题根源。本文将详细介绍Linux主机负载告警的排查思路和方法。

## 理解系统负载

### 负载的含义

系统负载表示在特定时间间隔内，运行队列中的平均进程数。它包括：

```
+------------------+
|   系统负载组成    |
+------------------+
|  R - 运行中进程   |  ← 正在CPU上运行
|  D - 不可中断睡眠 |  ← 等待I/O（磁盘/网络）
+------------------+
```

### 负载与CPU核心数的关系

```
负载值含义（以4核CPU为例）：

负载 < 4    → 系统空闲或轻载
负载 ≈ 4    → 系统满载，运行正常
负载 > 4    → 系统过载，存在瓶颈
负载 > 8    → 严重过载，需要立即处理
```

## 排查流程图

```
                    +------------------+
                    |   收到负载告警    |
                    +--------+---------+
                             |
                             v
                    +--------+---------+
                    |  确认负载数值     |
                    |  uptime / top    |
                    +--------+---------+
                             |
                             v
                    +--------+---------+
                    |  判断负载类型     |
                    +--------+---------+
                             |
          +------------------+------------------+
          |                  |                  |
          v                  v                  v
   +------+------+    +------+------+    +------+------+
   |  CPU密集型   |    |  I/O密集型   |    |  进程过多   |
   +------+------+    +------+------+    +------+------+
          |                  |                  |
          v                  v                  v
   +-------------+    +-------------+    +-------------+
   | 检查CPU使用 |    | 检查I/O等待 |    | 检查进程数  |
   +-------------+    +-------------+    +-------------+
          |                  |                  |
          v                  v                  v
   +-------------+    +-------------+    +-------------+
   | 定位高CPU   |    | 定位高I/O   |    | 定位异常    |
   | 进程        |    | 进程        |    | 进程        |
   +-------------+    +-------------+    +-------------+
```

## 第一步：确认负载状态

### 使用 uptime 命令

```bash
$ uptime
 10:30:45 up 30 days,  5:20,  3 users,  load average: 4.52, 3.80, 2.90
#                                              1分钟  5分钟  15分钟
```

负载趋势判断：

| 负载趋势 | 含义 |
|---------|------|
| 1分钟 > 15分钟 | 负载正在上升 |
| 1分钟 < 15分钟 | 负载正在下降 |
| 三个值接近 | 负载稳定 |

### 使用 top 命令

```bash
$ top
top - 10:30:45 up 30 days,  5:20,  3 users,  load average: 4.52, 3.80, 2.90
Tasks: 156 total,   2 running, 154 sleeping,   0 stopped,   0 zombie
%Cpu(s): 75.3 us,  15.2 sy,   0.0 ni,  5.1 id,  4.2 wa,  0.0 hi,  0.2 si
MiB Mem :   7823.5 total,    512.3 free,   4096.2 used,   3215.0 buff/cache
```

关键指标解读：

```
%Cpu(s):
  us (user)    - 用户空间CPU使用率
  sy (system)  - 内核空间CPU使用率
  id (idle)    - 空闲CPU百分比
  wa (I/O wait)- I/O等待百分比
  hi (hardware irq) - 硬件中断
  si (software irq) - 软件中断
```

## 第二步：判断负载类型

### CPU密集型负载特征

```bash
# top显示高CPU使用率
%Cpu(s): 85.0 us,  10.0 sy,   0.0 ni,  0.0 id,  0.0 wa

# 特征：us + sy 接近100%，id接近0%，wa很低
```

### I/O密集型负载特征

```bash
# top显示高I/O等待
%Cpu(s): 20.0 us,  10.0 sy,   0.0 ni, 30.0 id, 40.0 wa

# 特征：wa值很高，表示大量进程等待I/O
```

### 进程过多型负载特征

```bash
# 大量进程处于运行队列
Tasks: 500 total, 50 running, 450 sleeping

# 特征：running进程数很多
```

## 第三步：定位问题进程

### CPU密集型排查

```bash
# 查看CPU占用最高的进程
$ top -c
# 按 P 键按CPU使用率排序

# 或使用 ps 命令
$ ps aux --sort=-%cpu | head -10

# 查看进程的线程CPU使用
$ top -H -p <PID>
# 按 H 显示线程

# 使用 pidstat 持续监控
$ pidstat -p <PID> 1 5
```

### I/O密集型排查

```bash
# 安装 iotop
$ yum install -y iotop

# 查看I/O占用最高的进程
$ iotop -oP

# 使用 iostat 查看磁盘I/O
$ iostat -x 1 5
#              device: sda
# rrqm/s   wrqm/s     r/s     w/s    rkB/s    wkB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
#   0.00     5.00    0.00   50.00     0.00   256.00    10.24     1.50   30.00    0.00   30.00   5.00  25.00

# 关键指标：
# %util > 80%  - 磁盘利用率高
# await > 20ms - I/O响应时间长
# avgqu-sz > 2 - I/O队列长
```

### 进程数量排查

```bash
# 查看进程总数
$ ps aux | wc -l

# 查看各状态进程数
$ ps aux | awk '{print $8}' | sort | uniq -c

# 查看运行中的进程
$ ps aux | awk '$8 ~ /R/ {print}'

# 查看不可中断睡眠的进程
$ ps aux | awk '$8 ~ /D/ {print}'
```

## 第四步：深入分析

### 分析高CPU进程

```bash
# 查看进程调用栈
$ pstack <PID>

# 使用 perf 分析
$ perf top -p <PID>

# 使用 strace 跟踪系统调用
$ strace -c -p <PID>

# 查看进程打开的文件
$ lsof -p <PID>
```

### 分析高I/O进程

```bash
# 查看进程的磁盘读写
$ iotop -p <PID>

# 查看进程打开的文件
$ lsof -p <PID>

# 查看进程的文件描述符
$ ls -la /proc/<PID>/fd/

# 使用 blktrace 分析块设备
$ blktrace -d /dev/sda -o - | blkparse -i -
```

### 分析内存使用

```bash
# 查看内存使用
$ free -h

# 查看进程内存使用
$ top -c
# 按 M 键按内存使用率排序

# 使用 vmstat 查看系统状态
$ vmstat 1 5
# procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
#  r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
#  2  0      0 512300 102400 3215000    0    0    10    20  100  200 75 15  5  0  0

# 关键指标：
# r - 运行队列中的进程数
# b - 不可中断睡眠的进程数
# si/so - swap in/out，频繁交换表示内存不足
```

## 常见场景排查

### 场景一：Java应用CPU高

```bash
# 1. 找到Java进程
$ jps -l
$ ps aux | grep java

# 2. 找到高CPU线程
$ top -H -p <PID>

# 3. 将线程ID转换为16进制
$ printf "%x\n" <TID>

# 4. 查看线程堆栈
$ jstack <PID> | grep <HEX_TID> -A 30

# 5. 分析GC情况
$ jstat -gc <PID> 1000 10
$ jmap -histo <PID> | head -20
```

### 场景二：数据库I/O高

```bash
# MySQL排查
$ mysql -e "SHOW PROCESSLIST;"
$ mysql -e "SHOW ENGINE INNODB STATUS\G"

# 查看慢查询
$ mysql -e "SELECT * FROM information_schema.processlist WHERE TIME > 10;"

# PostgreSQL排查
$ psql -c "SELECT * FROM pg_stat_activity WHERE state = 'active';"
```

### 场景三：磁盘空间不足

```bash
# 查看磁盘使用
$ df -h

# 查看目录大小
$ du -sh /* | sort -rh | head -10

# 查找大文件
$ find / -type f -size +100M 2>/dev/null

# 查看被删除但仍占用空间的文件
$ lsof | grep deleted
```

### 场景四：网络I/O高

```bash
# 查看网络连接
$ netstat -antp | grep ESTABLISHED | wc -l

# 查看网络流量
$ iftop

# 查看连接状态分布
$ ss -s

# 查看各进程网络连接数
$ netstat -antp | awk '{print $7}' | sort | uniq -c | sort -rn
```

## 排查命令速查表

| 场景 | 命令 | 说明 |
|------|------|------|
| 查看负载 | `uptime` | 系统负载概览 |
| 实时监控 | `top` / `htop` | CPU、内存、进程 |
| I/O监控 | `iotop` / `iostat` | 磁盘I/O |
| 内存监控 | `free` / `vmstat` | 内存使用 |
| 网络监控 | `netstat` / `ss` / `iftop` | 网络连接 |
| 进程详情 | `ps` / `pstree` | 进程状态 |
| 系统调用 | `strace` / `ltrace` | 系统调用跟踪 |
| 性能分析 | `perf` | CPU性能分析 |
| 文件查看 | `lsof` | 打开的文件 |

## 排查脚本示例

```bash
#!/bin/bash
# load_check.sh - 负载排查脚本

echo "========== 系统负载 =========="
uptime

echo ""
echo "========== CPU使用TOP10 =========="
ps aux --sort=-%cpu | head -11

echo ""
echo "========== 内存使用TOP10 =========="
ps aux --sort=-%mem | head -11

echo ""
echo "========== 磁盘I/O =========="
iostat -x 1 3 | tail -n +4

echo ""
echo "========== 网络连接数 =========="
netstat -antp | awk '{print $6}' | sort | uniq -c | sort -rn

echo ""
echo "========== D状态进程 =========="
ps aux | awk '$8 ~ /D/ {print}'

echo ""
echo "========== 磁盘空间 =========="
df -h | grep -v tmpfs
```

## 最佳实践

### 1. 建立基线

```bash
# 记录正常状态下的各项指标
$ sar -o /var/log/sa/sa$(date +%d) 10 86400 &

# 查看历史数据
$ sar -f /var/log/sa/sa01
```

### 2. 设置告警阈值

```yaml
# Prometheus告警规则示例
groups:
  - name: node_alerts
    rules:
      - alert: HighLoad
        expr: node_load1 / on(instance) count(node_cpu_seconds_total{mode="idle"}) by (instance) > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High load on {{ $labels.instance }}"
```

### 3. 日志收集

```bash
# 配置rsyslog收集关键日志
# /etc/rsyslog.conf
*.warn;*.err    /var/log/warnings.log
```

## 参考资源

- [Linux Performance](https://www.brendangregg.com/linuxperf.html)
- [Red Hat Performance Tuning Guide](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/monitoring_and_managing_system_status_and_performance)
- `man top`, `man iostat`, `man vmstat`

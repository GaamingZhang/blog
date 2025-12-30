---
date: 2025-07-01
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 操作系统
tag:
  - 操作系统
---

# linux查看性能瓶颈相关命令

## 核心概念

Linux 系统性能瓶颈通常包括 **CPU、内存、磁盘 I/O、网络** 四个关键方面。诊断性能问题需要使用不同的工具来监控和分析各项指标，找出系统的瓶颈所在。

**性能诊断的重要性**：
- 预防系统故障和宕机
- 优化资源利用率，降低成本
- 提升用户体验和系统响应速度
- 为容量规划提供数据支持

**性能诊断的三个层次**：
1. **概览型**：快速了解整体系统状态（top、htop、dstat）
2. **深度型**：详细分析具体指标（vmstat、iostat、netstat）
3. **专项型**：针对特定问题的诊断（perf、strace、tcpdump）

**性能诊断方法论**：
- **USE方法**（Utilization、Saturation、Errors）：检查资源使用率、饱和度和错误数
- **RED方法**（Rate、Errors、Duration）：监控服务的请求速率、错误率和延迟时间
- **分层分析**：从系统层面→进程层面→应用层面逐步深入
- **关联分析**：综合多个指标（如高load但低CPU使用率可能是I/O瓶颈）

**关键指标关系**：
- CPU高使用率 + 高运行队列：CPU计算能力不足
- CPU低使用率 + 高IO等待：磁盘I/O瓶颈
- 高内存使用率 + 频繁swap：内存不足
- 网络高延迟 + 丢包：网络链路问题或配置不当

---

## CPU 性能分析

**1. top 命令（实时 CPU/内存监控）**：
```bash
top
# 关键指标：
# %Cpu(s): us(用户) sy(系统) ni(优先级) id(空闲) wa(IO等待)
# 高 wa 值表示 CPU 在等待 IO，可能是磁盘瓶颈
# 高 us 值表示应用消耗 CPU，可能是计算密集型任务
# 高 sy 值表示系统调用频繁，可能存在锁竞争或高频系统调用

# 按 CPU 使用率排序
top -o %CPU

# 监控特定进程
top -p PID

# 监控特定用户的进程
top -u username

# 批处理模式（用于脚本）
top -b -n 1 | head -10
```

**2. htop（top 的增强版）**：
```bash
htop
# 功能特点：
# - 彩色界面，更易读
# - 支持鼠标操作
# - 可横向滚动查看完整进程名
# - 支持排序和过滤

# 监控特定进程
htop -p PID1,PID2
```

**3. vmstat（虚拟内存统计）**：
```bash
vmstat 2 5  # 每 2 秒采样 5 次
# 关键列：
# r: 运行队列长度（待运行的进程数）
# b: 阻塞队列长度（等待 IO 的进程数）
# us, sy, id, wa: CPU 时间分配
# 高 r 值表示 CPU 竞争激烈，高 b 值表示磁盘 IO 瓶颈
# in: 每秒中断数，cs: 每秒上下文切换数

# 例：vmstat 1
# procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
#  r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa
#  2  0      0  8000000 100000 2000000  0    0   100   50  200  150 30 10 50 10
```

**4. mpstat（多 CPU 监控）**：
```bash
# 监控所有 CPU
mpstat -P ALL 1 5

# 监控特定 CPU（CPU 0）
mpstat -P 0 1 5

# 关键指标：
# %usr, %sys, %iowait, %idle
# 用于识别 CPU 负载不均衡问题
```

**5. pidstat（进程级统计）**：
```bash
# 监控进程 CPU 使用
pidstat -u 1 5

# 监控特定进程
pidstat -u -p PID 1 5

# 监控线程级 CPU 使用
pidstat -t -p PID 1 5
```

**6. perf 工具（性能分析利器）**：
```bash
# 系统级热点分析
perf top

# 进程热点分析
perf top -p PID

# 记录性能数据
sudo perf record -p PID -- sleep 10
sudo perf report

# 跟踪系统调用
sudo perf trace -p PID

# 统计函数调用次数
sudo perf stat -p PID

# 分析进程指令级热点
sudo perf record -p PID -F 99 -g -- sleep 10
sudo perf report --sort=dso,symbol
```

**7. strace（系统调用跟踪）**：
```bash
# 跟踪进程的系统调用
strace -p PID

# 统计系统调用次数
strace -c -p PID

# 跟踪特定系统调用
strace -e open,read,write -p PID

# 查看系统调用耗时
strace -T -p PID
```

**8. sar（系统活动记录）**：
```bash
# CPU 统计（最近 24 小时）
sar -u 1 5

# 查看历史 CPU 数据
sar -u -f /var/log/sa/sa22

# 监控特定 CPU
sar -P 0 1 5

# 上下文切换统计
sar -w 1 5
```

**9. dstat（综合监控工具）**：
```bash
# 综合监控 CPU、内存、磁盘、网络
dstat

# 显示详细信息
dstat -cdngy --top-cpu --top-mem

# 按 CPU 核心显示
dstat -C 0,1,total
```

**10. taskset（CPU 亲和性）**：
```bash
# 查看进程的 CPU 亲和性
taskset -p PID

# 设置进程绑定到特定 CPU
taskset -p 0x00000001 PID  # 绑定到 CPU 0

taskset -c 0,1 PID  # 绑定到 CPU 0 和 1
```

---

## 内存性能分析

**1. free 命令（内存快照）**：
```bash
free -h  # 以人类易读格式显示
# 输出示例：
#               total        used        free      shared  buff/cache   available
# Mem:            16Gi       8Gi        2Gi         100Mi      5Gi          7Gi
# Swap:           8Gi        1Gi        7Gi

# 关键概念：
# available = free + buff/cache（实际可用内存，包括可回收缓存）
# used - buff/cache = 实际应用使用的内存
# 内存压力 = (used - buff/cache) / total > 90% 表示压力高

# 查看内存变化趋势
watch -n 5 free -h
```

**2. 详细内存信息（/proc/meminfo）**：
```bash
# 查看完整内存统计
cat /proc/meminfo

# 过滤关键指标
cat /proc/meminfo | grep -E "MemTotal|MemFree|MemAvailable|Buffers|Cached|SwapTotal|SwapFree|Active|Inactive"

# 解释关键指标：
# Active: 最近使用的内存页
# Inactive: 最近未使用的内存页
# SwapCached: 交换到磁盘后又被读回的内存页
# Dirty: 等待写回磁盘的脏页
```

**3. ps 和 pmap（进程内存分析）**：
```bash
# 按内存使用排序
ps aux --sort=-%mem | head -10

# 查看进程详细内存
pmap -x PID
pmap -XX PID  # 更详细的内存映射信息

# 统计所有进程内存
ps aux | awk '{sum+=$6} END {print sum/1024 "MB"}'

# 查看进程内存段
cat /proc/PID/maps
```

**4. smaps（进程内存段详细信息）**：
```bash
# 查看进程内存段详细信息
cat /proc/PID/smaps

# 统计进程各个内存段的大小
sudo cat /proc/PID/smaps | awk '/^Size:/ {sum += $2} END {print sum/1024 "KB"}'

# 统计私有内存和共享内存
sudo cat /proc/PID/smaps | awk '/^Private.*:/ {sum_private += $2} /^Shared.*:/ {sum_shared += $2} END {print "Private: " sum_private/1024 "KB, Shared: " sum_shared/1024 "KB"}'
```

**5. 内核内存分析（slabtop）**：
```bash
# 实时查看内核 slab 缓存使用
slabtop

# 查看内核内存统计
cat /proc/slabinfo

# 内核内存泄漏检测
sudo slabtop -o -s c | head -20
```

**6. 内存泄漏诊断工具**：
```bash
# 1. 监控特定进程内存增长
watch -n 1 'ps aux | grep process_name'

# 2. 使用 valgrind（需要编译时支持）
valgrind --leak-check=full --show-leak-kinds=all ./program
valgrind --tool=massif ./program  # 堆内存分析
ms_print massif.out.PID  # 查看 massif 结果

# 3. 使用 memleak（需要内核支持）
sudo perf memleak --live-only

# 4. 使用 AddressSanitizer（编译时支持）
gcc -fsanitize=address -g program.c -o program
./program

# 5. 使用 mtrace（glibc 内置工具）
export MALLOC_TRACE=mtrace.out
exec ./program
mtrace ./program mtrace.out
```

**7. 内存性能监控工具**：
```bash
# htop 内存监控
htop

# dstat 内存统计
dstat -m

# vmstat 内存相关指标
vmstat 1 5
# si: 每秒从 swap 读入内存的大小
# so: 每秒从内存写入 swap 的大小
# 如果 si/so 持续大于 0，表示内存不足

# pidstat 进程内存使用
pidstat -r 1 5
pidstat -r -p PID 1 5  # 特定进程
```

**8. 内存性能优化建议**：
```bash
# 清理缓存（谨慎使用）
sync; echo 3 > /proc/sys/vm/drop_caches

# 调整 swappiness（默认 60，值越低越倾向使用内存）
sysctl vm.swappiness=10  # 临时修改
echo "vm.swappiness=10" >> /etc/sysctl.conf  # 永久修改

# 调整内存分配策略
# 禁用透明大页（某些应用可能受影响）
echo never > /sys/kernel/mm/transparent_hugepage/enabled
```

---

## 磁盘 I/O 分析

**1. iostat（全面磁盘 I/O 统计）**：
```bash
# 详细模式，每 2 秒采样 5 次
iostat -x 2 5

# 关键指标详解：
# r/s, w/s: 每秒读写 IO 次数
# rkB/s, wkB/s: 每秒读写吞吐量
# %util: 磁盘使用率（> 80% 表示接近饱和）
# await: 平均 IO 等待时间（包括队列时间 + 服务时间）
# svctm: 磁盘平均服务时间（纯粹的磁盘处理时间）
# avgqu-sz: 平均请求队列长度

# 只显示磁盘统计
iostat -d 2 5

# 监控特定磁盘
iostat -x -p sda 2 5

# 查看所有分区
iostat -x -t 1
```

**2. iotop（实时磁盘 IO 监控）**：
```bash
# 只显示有 IO 操作的进程
iotop -o

# 显示实际使用的带宽（不是百分比）
iotop -b -n 1  # 批处理模式

# 只显示读取或写入操作
iotop -o -a -P  # 累计模式，按进程显示

# 监控特定进程
iotop -p PID

# 关键指标：
# TID: 线程 ID
# PRIO: 优先级
# USER: 用户名
# DISK READ: 读取速率
# DISK WRITE: 写入速率
# SWAPIN:  swap 使用率
# IO>: IO 等待百分比
```

**3. lsof（文件句柄分析）**：
```bash
# 找出最多打开文件的进程
lsof | awk '{print $1}' | sort | uniq -c | sort -rn | head -10

# 查看特定进程打开的文件
lsof -p PID

# 查看特定文件被哪些进程打开
lsof /path/to/file

# 监控文件删除后仍占用的空间
lsof | grep deleted

# 查看网络文件句柄
lsof -i

# 统计系统总文件句柄数
lsof | wc -l
```

**4. du 和 df（磁盘空间管理）**：
```bash
# 磁盘使用情况
df -h

# 查看 inode 使用情况
df -i

# 查找大目录
du -sh /* | sort -rh | head -10

# 递归查找大文件（> 100MB）
find / -type f -size +100M -exec ls -lh {} \;

# 查找最近修改的文件
find / -type f -mtime -7 -size +50M -exec ls -lh {} \;

# 统计特定目录下的文件数
find /path/to/dir -type f | wc -l
```

**5. fio（磁盘性能基准测试）**：
```bash
# 顺序读测试
fio --name=seqread --ioengine=libaio --direct=1 --rw=read --bs=4k --size=1G --numjobs=4 --iodepth=16

# 随机读写混合测试（70%读，30%写）
fio --name=randrw --ioengine=libaio --direct=1 --rw=randrw --rwmixread=70 --bs=4k --size=1G --numjobs=4 --iodepth=16

# 顺序写测试
fio --name=seqwrite --ioengine=libaio --direct=1 --rw=write --bs=64k --size=1G

# 关键指标：
# IOPS: 每秒 IO 操作数
# BW: 带宽 (MB/s)
# lat: 延迟 (ms)
# clat: 完成延迟（从请求提交到完成的时间）
# slat: 提交延迟（从请求准备到提交的时间）
# %util: 磁盘利用率
```

**6. blktrace（块设备跟踪）**：
```bash
# 跟踪磁盘 IO
sudo blktrace -d /dev/sda -o - | blkparse -i -

# 生成报告
sudo blktrace -d /dev/sda -o sda_trace
sudo blkparse -i sda_trace -o sda_trace.txt
sudo btt -i sda_trace
```

**7. hdparm（磁盘性能测试）**：
```bash
# 测试磁盘读取速度
hdparm -t /dev/sda

# 测试缓存读取速度
hdparm -T /dev/sda

# 查看磁盘信息
hdparm -I /dev/sda
```

**8. 磁盘性能优化建议**：
```bash
# 1. 文件系统优化
# 调整 ext4 挂载参数
echo "LABEL=root / ext4 defaults,noatime,nodiratime,data=ordered 0 1" >> /etc/fstab

# 2. 调整 IO 调度器
# 查看当前调度器
cat /sys/block/sda/queue/scheduler

# 设置为 deadline 调度器
echo deadline > /sys/block/sda/queue/scheduler

# 3. 增加读写缓存
echo 2048 > /proc/sys/vm/dirty_background_ratio
echo 30 > /proc/sys/vm/dirty_ratio

# 4. 限制进程 IO 优先级
# 使用 ionice 设置进程 IO 优先级
ionice -c 2 -n 7 -p PID  # 设置为低优先级

# 5. 定期清理磁盘碎片（针对 ext4/xfs）
e4defrag /dev/sda
sudo xfs_fsr /dev/sda1
```

---

## 网络性能分析

**1. ss 和 netstat（网络连接统计）**：
```bash
# ss 是 netstat 的现代替代品，更快
# 显示所有 TCP 连接
ss -tuln

# 显示所有 ESTABLISHED 连接
ss -t state established

# 显示 TCP 连接状态统计
ss -s

# 统计不同状态的连接
ss -tan | awk '{print $2}' | sort | uniq -c

# 查看特定端口的连接
ss -tuln | grep 8080

# 监控网络连接（每 2 秒刷新）
watch -n 2 'ss -tan | awk "{print $2}" | sort | uniq -c'

# 查看连接详细信息
ss -o state established '( dport = :80 or sport = :80 )'
```

**2. ifconfig 和 ip（网络接口信息）**：
```bash
# 查看网络接口信息
ifconfig
ip addr show

# 查看网络接口状态
ip link show

# 查看路由表
ip route show

# 查看 ARP 缓存
ip neigh show
arp -a

# 查看网络统计
ip -s link
netstat -i
```

**3. 网络吞吐量监控**：
```bash
# ifstat（简洁的吞吐量显示）
ifstat -i eth0 1

# iftop（实时带宽监控，需要 root）
iftop -i eth0 -n
iftop -i eth0 -n -P  # 显示端口号
iftop -i eth0 -n -F 192.168.1.0/24  # 过滤特定网段

# nload（可视化吞吐量监控）
nload -i 1024 -o 1024 eth0

# bmon（带宽监控）
bmon
```

**4. tcpdump 和 wireshark（数据包分析）**：
```bash
# 捕获 HTTP 流量
tcpdump -i eth0 'tcp port 80'

# 捕获 HTTPS 流量（只能看到握手）
tcpdump -i eth0 'tcp port 443'

# 捕获特定 IP 的流量
tcpdump -i eth0 host 192.168.1.100

# 捕获特定端口范围的流量
tcpdump -i eth0 'tcp portrange 1000-2000'

# 保存为文件供后续分析
tcpdump -i eth0 -w traffic.pcap -s 0  # -s 0 捕获完整数据包

# 读取 pcap 文件
tcpdump -r traffic.pcap

# 统计源 IP 连接数
tcpdump -i eth0 -nn 'tcp' | awk '{print $3}' | cut -d. -f1-4 | sort | uniq -c | sort -rn

# 使用 wireshark 分析（图形界面）
wireshark traffic.pcap
```

**5. 网络诊断工具**：
```bash
# ping（检测网络连通性）
ping -c 4 google.com
ping -i 0.5 -c 10 google.com  # 间隔 0.5 秒，发送 10 个包

# traceroute（追踪路由路径）
traceroute google.com
traceroute -I google.com  # 使用 ICMP

# mtr（结合 ping 和 traceroute）
mtr google.com

# nslookup 和 dig（DNS 解析）
nslookup google.com
dig google.com

# 检查 DNS 响应时间
dig @8.8.8.8 google.com +stats

# curl（HTTP 测试）
curl -v https://www.google.com
curl -w "%{time_total}" -o /dev/null -s https://www.google.com  # 测试响应时间

# telnet（端口连通性测试）
telnet google.com 80
telnet google.com 443
```

**6. 网络性能基准测试**：
```bash
# iperf（网络带宽测试）
# 服务端
iperf -s

# 客户端
iperf -c server_ip

# 使用 TCP 测试带宽
iperf -c server_ip -t 10 -i 1

# 使用 UDP 测试带宽
iperf -c server_ip -u -b 100M

# iperf3（新版本）
iperf3 -s
iperf3 -c server_ip

# 测试双向带宽
iperf3 -c server_ip -d
```

**7. 高级网络分析**：
```bash
# tcptraceroute（TCP 端口路由追踪）
tcptraceroute google.com 80

# arping（ARP 层级 ping）
arping -c 4 192.168.1.1

# ss 查看 TCP 连接的重传情况
ss -t state retransmit

# 查看 TCP 连接的详细信息，包括重传
sudo ss -t -o state established '( dport = :80 )' | head -5
```

**8. 网络性能优化建议**：
```bash
# 调整 TCP 缓冲区大小
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
sysctl -w net.ipv4.tcp_rmem="4096 87380 16777216"
sysctl -w net.ipv4.tcp_wmem="4096 65536 16777216"

# 启用 TCP 快速打开
sysctl -w net.ipv4.tcp_fastopen=3

# 调整 TIME_WAIT 状态超时
sysctl -w net.ipv4.tcp_fin_timeout=30

# 启用 TCP 重传快速恢复
sysctl -w net.ipv4.tcp_frto=1

# 保存设置
echo "net.core.rmem_max=16777216" >> /etc/sysctl.conf
sysctl -p
```

---

## 综合诊断思路

**CPU 瓶颈诊断**：
```bash
# 1. 查看整体 CPU 使用率
top -n 1 | head -3

# 2. 查看运行队列
vmstat 1 5 | grep -A 1 "procs"

# 3. 找出高 CPU 进程
ps aux --sort=-%cpu | head -5

# 4. 分析进程热点（需要 perf）
perf record -p PID -- sleep 10
perf report
```

**磁盘 IO 瓶颈诊断**：
```bash
# 1. 查看磁盘使用率
iostat -x 2 5 | grep %util

# 2. 找出高 IO 进程
iotop -o -n 1

# 3. 查看等待 IO 的进程
ps aux | grep D+  # D 表示磁盘睡眠

# 4. 分析 IO 模式
iostat -x 2 5  # 查看 r/s, w/s, await
```

**内存压力诊断**：
```bash
# 1. 检查可用内存
free -h

# 2. 查看内存占用进程
ps aux --sort=-%mem | head -10

# 3. 检查 swap 使用
free -h | grep Swap

# 4. 检查内存泄漏
watch -n 2 'ps aux | grep process | head -1'
```

**网络瓶颈诊断**：
```bash
# 1. 检查网络吞吐量
ifstat -i eth0 1 5

# 2. 查看连接统计
ss -s

# 3. 找出高流量连接
iftop -i eth0 -n

# 4. 分析丢包和延迟
ping -c 4 target_host
traceroute target_host
```

---

## 快速参考命令速查表

| 场景         | 命令                       | 关键指标               |
| ------------ | -------------------------- | ---------------------- |
| **整体状态** | `top`, `htop`, `dstat`     | CPU%, MEM%, LOAD       |
| **CPU 详情** | `vmstat 2 5`, `sar -u`     | r, wa, us, sy          |
| **磁盘 IO**  | `iostat -x 2 5`, `iotop`   | r/s, w/s, await, %util |
| **内存详情** | `free -h`, `ps aux`        | available, used, swap  |
| **网络状态** | `ss -s`, `ifstat`, `iftop` | ESTABLISHED, RX, TX    |
| **进程追踪** | `strace`, `ltrace`         | 系统调用，函数调用     |
| **性能基准** | `fio`, `sysbench`          | IOPS, BW, latency      |

---

## 实战分析案例

**案例 1：CPU 高占用**：
```bash
# 第 1 步：查看总体 CPU
top -n 1

# 第 2 步：找出消耗 CPU 的进程
ps aux --sort=-%cpu | head -5

# 第 3 步：分析进程热点
perf record -p PID -- sleep 10
perf report

# 第 4 步：优化代码或扩容
# 可能的原因：算法低效、无限循环、高频系统调用
```

**案例 2：内存持续增长**：
```bash
# 第 1 步：确认内存增长
watch -n 5 'free -h'

# 第 2 步：找出消耗内存的进程
ps aux --sort=-%mem | head -5

# 第 3 步：检查内存泄漏
valgrind --leak-check=full ./program

# 第 4 步：确定增长速率
# 每小时增长多少，推算何时内存溢出
```

**案例 3：磁盘 IO 高**：
```bash
# 第 1 步：查看磁盘忙碌度
iostat -x 2 5 | tail

# 第 2 步：找出高 IO 进程
iotop -o -n 1

# 第 3 步：分析 IO 模式
# 是顺序读写还是随机？是频繁小 IO 还是大 IO？

# 第 4 步：优化
# 增加缓存、批量操作、使用更快的存储设备
```

---

### 相关高频面试题

#### Q1: vmstat 输出中的 r 和 b 字段各表示什么？如何判断系统性能状态？

**答案**：

```bash
# r (runnable processes): 运行队列中的进程数
# - 高 r 值表示 CPU 竞争激烈，进程等待 CPU 时间较长
# - 经验值：r > CPU核心数 2 倍时，CPU 成为瓶颈

# b (blocked processes): 因等待 IO 而被阻塞的进程数
# - 高 b 值表示磁盘 IO 成为瓶颈，进程在等待磁盘读写
# - 高 b 值通常伴随高 wa（IO 等待时间）

# 诊断示例：
vmstat 2 10

# 若 r 持续 > 8（4核CPU），说明 CPU 不足
# 若 b 持续 > 0，说明磁盘 IO 成为瓶颈
# 若 wa > 30%，说明大量时间在等待 IO
```

#### Q2: iostat 中 await 和 svctm 的区别是什么？

**答案**：

```bash
# await: 平均 IO 等待时间（包括排队时间 + 服务时间）
# 较高的 await 表示硬盘繁忙或响应慢

# svctm: 硬盘平均服务时间（纯粹的磁盘处理时间）
# svctm 高表示硬盘性能下降

# 区别：
# - await = 队列等待 + svctm
# - await 高但 svctm 低：表示磁盘繁忙，队列排队多
# - 两者都高：表示硬盘本身有问题，需要更换或维修

# 示例分析：
iostat -x 2 5
# await=50ms, svctm=5ms 时，队列等待=45ms，需要提高吞吐量
# await=50ms, svctm=45ms 时，硬盘响应慢，需要更换硬盘
```

#### Q3: 如何快速判断服务器当前的性能瓶颈？

**答案**：

```bash
# 三个命令组合快速诊断：

# 1. 查看 CPU 状态
top -n 1 | head -4
# wa% 高 → 磁盘 IO 瓶颈，us% 高 → CPU 瓶颈

# 2. 查看内存状态
free -h
# available < total 的 10% → 内存压力大

# 3. 查看磁盘状态
iostat -x 1 3 | tail -1
# %util > 80% 和 await > 100ms → 磁盘 IO 瓶颈

# 快速判断：
# - %util > 80% && wa% > 30% → 磁盘 IO 瓶颈
# - us% > 80% && wa% < 5% → CPU 瓶颈
# - available < 10% → 内存瓶颈
# - RX/TX 接近 NIC 限制 → 网络瓶颈
```

#### Q4: 如何监控特定进程的性能指标？

**答案**：

```bash
# 1. 实时监控 CPU/内存
top -p PID  # 只显示特定进程
watch -n 1 'ps aux | grep PID | head -1'

# 2. 监控进程打开的文件数
watch -n 1 'lsof -p PID | wc -l'

# 3. 监控进程网络连接
watch -n 2 'netstat -np | grep PID | wc -l'

# 4. 监控进程 IO 操作
iotop -p PID

# 5. 追踪进程的系统调用
strace -p PID -o trace.log
# 分析高频系统调用
awk '{print $1}' trace.log | sort | uniq -c | sort -rn | head

# 6. 查看进程详细内存
pmap -x PID
```

#### Q5: 网络连接 TIME_WAIT 过多会造成什么问题？如何处理？

**答案**：

```bash
# TIME_WAIT 是 TCP 关闭时的正常状态，用于确保延迟报文处理完

# 问题：
# - 占用文件描述符（socket 资源）
# - 占用内存
# - 达到端口限制时，无法建立新连接

# 诊断：
netstat -an | grep TIME_WAIT | wc -l

# 解决方案：

# 1. 应用层优化（最根本）
# - 使用 HTTP Keep-Alive（复用连接）
# - 增加连接池大小
# - 使用长连接而非短连接

# 2. 系统层优化（调整参数）
# 减少 TIME_WAIT 等待时间
sysctl -w net.ipv4.tcp_fin_timeout=30

# 允许 TIME_WAIT socket 重用
sysctl -w net.ipv4.tcp_tw_reuse=1

# 快速回收 TIME_WAIT（需谨慎）
sysctl -w net.ipv4.tcp_tw_recycle=1  # 已弃用

# 3. 增加端口范围
sysctl -w net.ipv4.ip_local_port_range="1024 65535"
```

#### Q6: CPU 使用率和 load average 有什么区别？

**答案**：

```bash
# CPU 使用率：CPU 正在运行的时间占总时间的百分比
# - top 命令显示的 Cpu(s)
# - 0-100% 范围，显示当前时刻的使用情况

# Load Average：过去 1/5/15 分钟内平均运行队列长度
# - uptime 命令显示
# - 通常没有百分比上限，受 CPU 核心数影响
# 
# 示例：
uptime
# load average: 2.5, 2.0, 1.8

# 在 4 核 CPU 上：
# - load 2.5 表示有 2.5 个进程在等待 CPU
# - 如果 CPU 使用率只有 50%，说明有进程被阻塞（IO 等待）

# 判断标准：
# - load = CPU 核心数时，系统处于最优利用
# - load > CPU 核心数 × 2 时，性能已严重下降
# - 如果 load 高但 CPU 使用率低，说明 IO 等待严重
```

#### Q7: 如何使用 perf 工具进行 CPU 热点分析？

**答案**：

```bash
# perf 是 Linux 内核自带的性能分析工具，功能强大

# 1. 安装 perf（不同发行版命令不同）
sudo apt install linux-tools-common linux-tools-generic  # Debian/Ubuntu
sudo yum install perf  # CentOS/RHEL

# 2. 实时查看 CPU 热点
sudo perf top

# 3. 记录特定进程的性能数据
sudo perf record -p PID -- sleep 10

# 4. 查看性能报告
sudo perf report

# 5. 分析调用栈
sudo perf record -p PID -g -- sleep 10
sudo perf report --call-graph=graph

# 6. 统计函数调用次数
sudo perf stat -p PID

# 7. 跟踪系统调用
sudo perf trace -p PID

# 8. 生成火焰图（需要额外工具）
git clone https://github.com/brendangregg/FlameGraph.git
perf record -a -g -- sleep 10
perf script > out.perf
./FlameGraph/stackcollapse-perf.pl out.perf > out.folded
./FlameGraph/flamegraph.pl out.folded > flamegraph.svg
```

#### Q8: 如何检测和分析内存泄漏？

**答案**：

```bash
# 内存泄漏是指程序分配的内存未能及时释放，导致内存持续增长

# 1. 监控内存增长趋势
watch -n 5 "free -h && ps aux --sort=-%mem | head -5"

# 2. 使用 valgrind 检测内存泄漏
gcc -g program.c -o program
valgrind --leak-check=full --show-leak-kinds=all ./program

# 3. 使用 massif 分析堆内存分配
valgrind --tool=massif ./program
ms_print massif.out.*  # 查看分析结果

# 4. 使用 perf memleak
sudo perf memleak --live-only

# 5. 使用 AddressSanitizer（编译时支持）
gcc -fsanitize=address -g program.c -o program
./program

# 6. 使用 mtrace（glibc 内置工具）
export MALLOC_TRACE=mtrace.out
exec ./program
mtrace ./program mtrace.out

# 7. 分析进程内存段
pmap -x PID
cat /proc/PID/smaps
```

#### Q9: 磁盘 I/O 调度器有哪些？如何选择和配置？

**答案**：

```bash
# I/O 调度器决定了磁盘 I/O 请求的处理顺序

# 1. 查看当前支持的调度器
cat /sys/block/sda/queue/scheduler
# 输出示例：noop deadline [cfq] （cfq 是当前使用的）

# 2. 主要调度器介绍
# - noop: 简单的 FIFO 队列，适合 SSD 和高端存储设备
# - deadline: 注重请求完成的截止时间，平衡读/写延迟
# - cfq (Completely Fair Queueing): 为每个进程分配时间片，公平但效率较低
# - bfq (Budget Fair Queueing): cfq 的改进版，更适合桌面系统

# 3. 临时设置调度器
echo deadline > /sys/block/sda/queue/scheduler

# 4. 永久设置调度器（需要修改 GRUB）
sudo vi /etc/default/grub
# 在 GRUB_CMDLINE_LINUX 中添加 elevator=deadline
sudo update-grub  # Debian/Ubuntu
sudo grub2-mkconfig -o /boot/grub2/grub.cfg  # CentOS/RHEL

# 5. 调度器选择建议
# - SSD/NVMe: noop 或 deadline
# - 传统机械硬盘: deadline（数据库等延迟敏感应用）或 cfq（通用服务器）
# - 低延迟要求: deadline
# - 高吞吐量要求: noop（SSD）或 cfq（机械硬盘）
```

#### Q10: 如何分析和优化 TCP 网络性能？

**答案**：

```bash
# TCP 性能受多种因素影响：拥塞控制、窗口大小、超时设置等

# 1. 查看当前 TCP 配置
sysctl net.ipv4.tcp_* | grep -E "congestion_control|window|timeout"

# 2. 查看当前使用的拥塞控制算法
sysctl net.ipv4.tcp_congestion_control

# 3. 常用 TCP 性能优化参数

# 调整 TCP 缓冲区大小
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
sysctl -w net.ipv4.tcp_rmem="4096 87380 16777216"
sysctl -w net.ipv4.tcp_wmem="4096 65536 16777216"

# 启用 TCP 快速打开
sysctl -w net.ipv4.tcp_fastopen=3

# 调整 TIME_WAIT 超时
sysctl -w net.ipv4.tcp_fin_timeout=30

# 启用窗口缩放
sysctl -w net.ipv4.tcp_window_scaling=1

# 调整拥塞控制算法
sysctl -w net.ipv4.tcp_congestion_control=cubic  # 或 bbr（如果支持）

# 4. 网络性能测试
iperf3 -s  # 服务端
iperf3 -c server_ip -t 10 -i 1

# 5. 诊断 TCP 连接问题
tcpdump -i eth0 -w tcp.pcap
tcptrace -T tcp.pcap
```

---

### 性能诊断决策树

```
性能问题
├─ 系统响应慢
│  ├─ top: Cpu(s) us% > 80%? 
│  │  └─ YES → CPU 瓶颈，检查 ps aux
│  ├─ top: Cpu(s) wa% > 30%?
│  │  └─ YES → 磁盘 IO 瓶颈，运行 iostat
│  └─ top: available < 10%?
│     └─ YES → 内存压力，检查 ps aux --sort=-%mem
│
├─ 应用内存持续增长
│  ├─ ps aux | grep app
│  └─ watch -n 5 'ps aux | grep app'
│     → 确认增长速率和内存泄漏
│
├─ 网络延迟高
│  ├─ ping 目标主机
│  ├─ traceroute 分析路径
│  └─ tcpdump 抓包分析
│
└─ 磁盘满
   ├─ df -h 查看挂载点
   └─ du -sh /* 查找大目录
```

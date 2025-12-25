---
date: 2025-07-01
author: Gaaming Zhang
category:
  - 操作系统
tag:
  - 操作系统
  - 还在施工中
---

# top命令的基本使用方法

## 核心概念

**top** 是 Linux/Unix 系统中最常用的实时系统监控工具，可以动态显示系统整体运行状态、进程资源占用情况、CPU、内存使用等信息。它是系统管理员和开发人员排查性能问题的第一工具。

---

## top 命令输出界面详解

**完整的 top 输出示例**：

```
top - 15:30:45 up 10 days,  2:15,  3 users,  load average: 0.52, 0.58, 0.59
Tasks: 245 total,   1 running, 244 sleeping,   0 stopped,   0 zombie
%Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.3 id,  0.3 wa,  0.0 hi,  0.1 si,  0.0 st
MiB Mem :  16384.0 total,   2048.5 free,   8192.3 used,   6143.2 buff/cache
MiB Swap:   8192.0 total,   6144.0 free,   2048.0 used.   6800.5 avail Mem

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
 1234 root      20   0 5123456 512000  10240 S   5.0   3.1  10:23.45 mysqld
 5678 www       20   0 2048576 256000   8192 S   2.0   1.6   5:12.34 nginx
 9012 app       20   0 1024000 128000   4096 R   1.5   0.8   2:34.56 python
 3456 root      20   0  512000  64000   2048 S   0.5   0.4   1:23.45 sshd
```

---

## 第一部分：系统概要信息

**第一行 - 系统时间和负载**：

```
top - 15:30:45 up 10 days,  2:15,  3 users,  load average: 0.52, 0.58, 0.59
      ^^^^^^^^    ^^^^^^^^^^        ^^^^^^^^  ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
      当前时间    运行时长          登录用户   系统负载（1分钟、5分钟、15分钟）
```

**字段说明**：
- **15:30:45**：当前系统时间
- **up 10 days, 2:15**：系统已运行 10 天 2 小时 15 分钟
- **3 users**：当前有 3 个用户登录
- **load average: 0.52, 0.58, 0.59**：系统负载平均值
  - 第一个值：最近 1 分钟的平均负载
  - 第二个值：最近 5 分钟的平均负载
  - 第三个值：最近 15 分钟的平均负载

**负载解读**：
```
负载值的含义（假设是 4 核 CPU）：
- 0.00-0.70：轻负载，系统空闲
- 0.70-1.00：中等负载，还可以
- 1.00-4.00：负载较高，需要关注
- > 4.00：过载，可能有性能问题

判断标准：
- 负载 < CPU 核心数：正常
- 负载 = CPU 核心数：满负载
- 负载 > CPU 核心数：过载（有进程在等待）

查看 CPU 核心数：
grep -c processor /proc/cpuinfo
```

**第二行 - 进程统计**：

```
Tasks: 245 total,   1 running, 244 sleeping,   0 stopped,   0 zombie
       ^^^^^^^^^^   ^^^^^^^^^^  ^^^^^^^^^^^^^   ^^^^^^^^^^   ^^^^^^^^^
       总进程数     运行中      睡眠中          停止         僵尸进程
```

**字段说明**：
- **total**：系统总进程数
- **running**：正在运行的进程数（通常只有 1-2 个）
- **sleeping**：睡眠状态的进程（等待事件或资源）
- **stopped**：被停止的进程（通常用 Ctrl+Z 或 kill -STOP）
- **zombie**：僵尸进程（已终止但父进程未回收）
  - 如果数量多，说明有程序 bug（父进程未调用 wait）

**第三行 - CPU 使用率**：

```
%Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.3 id,  0.3 wa,  0.0 hi,  0.1 si,  0.0 st
          ^^^^^    ^^^^^    ^^^^^   ^^^^^^^   ^^^^^    ^^^^^    ^^^^^    ^^^^^
          用户态   系统态   nice    空闲      等待I/O  硬中断   软中断   虚拟化
```

**字段详解**：

| 字段   | 全称               | 说明                       | 正常值 | 异常值            |
| ------ | ------------------ | -------------------------- | ------ | ----------------- |
| **us** | user               | 用户态 CPU 占用            | 30-70% | >90% CPU 密集     |
| **sy** | system             | 系统态（内核）CPU 占用     | 5-20%  | >30% 系统调用频繁 |
| **ni** | nice               | nice 值调整的进程 CPU 占用 | 0-5%   | -                 |
| **id** | idle               | CPU 空闲百分比             | 20-70% | <10% CPU 不足     |
| **wa** | iowait             | 等待 I/O 的 CPU 时间       | 0-5%   | >20% I/O 瓶颈     |
| **hi** | hardware interrupt | 硬中断 CPU 占用            | 0-1%   | >5% 硬件问题      |
| **si** | software interrupt | 软中断 CPU 占用            | 0-2%   | >10% 网络/中断多  |
| **st** | steal time         | 虚拟机被宿主机偷走的 CPU   | 0%     | >5% 物理机超卖    |

**关键性能指标**：
```
CPU 性能分析：
1. us 高：应用程序 CPU 密集
   → 优化算法，使用缓存，减少计算

2. sy 高：系统调用频繁
   → 减少系统调用，使用批处理，检查锁竞争

3. wa 高：I/O 瓶颈
   → 优化磁盘 I/O，使用 SSD，增加缓存

4. si 高：网络或软中断多
   → 检查网络流量，优化中断处理

5. st 高：虚拟机资源不足
   → 联系云服务商，调整资源配额
```

**第四、五行 - 内存使用情况**：

```
MiB Mem :  16384.0 total,   2048.5 free,   8192.3 used,   6143.2 buff/cache
           ^^^^^^^^^^^^     ^^^^^^^^^^^    ^^^^^^^^^^^    ^^^^^^^^^^^^^^^^
           总内存           空闲内存       已用内存        缓冲/缓存

MiB Swap:   8192.0 total,   6144.0 free,   2048.0 used.   6800.5 avail Mem
            ^^^^^^^^^^^^     ^^^^^^^^^^^    ^^^^^^^^^^^    ^^^^^^^^^^^^^^^^
            Swap总量         Swap空闲       Swap已用       可用内存
```

**内存计算公式**：
```
Linux 内存管理：
total = used + free + buff/cache

实际可用内存：
avail Mem ≈ free + buff/cache（大部分可回收）

关键指标：
1. avail Mem：实际可用内存（最重要）
2. used：应用程序实际使用的内存
3. buff/cache：文件缓存（可释放）
4. Swap used：交换分区使用情况

内存压力判断：
- avail Mem > 20%：正常
- avail Mem 10-20%：需要关注
- avail Mem < 10%：内存紧张
- Swap used > 50%：严重问题（性能下降）
```

**Swap 使用分析**：
```
Swap 高的影响：
- 内存页被换出到磁盘
- 访问速度慢 100-1000 倍
- 系统响应缓慢

解决方案：
1. 增加物理内存
2. 优化应用内存使用
3. 调整 swappiness（控制换页积极性）
   echo 10 > /proc/sys/vm/swappiness  # 默认 60
4. 找出内存占用大的进程并优化
```

---

## 第二部分：进程列表

**列标题说明**：

```
  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
```

| 列名        | 说明     | 详细解释                                      |
| ----------- | -------- | --------------------------------------------- |
| **PID**     | 进程 ID  | 进程的唯一标识符                              |
| **USER**    | 用户     | 进程所属用户                                  |
| **PR**      | 优先级   | 内核动态调整的优先级（20 为基准）             |
| **NI**      | Nice 值  | 用户设置的优先级（-20 到 19，越小优先级越高） |
| **VIRT**    | 虚拟内存 | 进程使用的虚拟内存总量（包括所有映射）        |
| **RES**     | 常驻内存 | 实际占用的物理内存（Resident）                |
| **SHR**     | 共享内存 | 与其他进程共享的内存                          |
| **S**       | 状态     | 进程状态（R/S/D/Z/T/I）                       |
| **%CPU**    | CPU 占用 | CPU 使用百分比（单核 100%，多核可超过 100%）  |
| **%MEM**    | 内存占用 | 物理内存使用百分比                            |
| **TIME+**   | 运行时间 | 进程累计使用的 CPU 时间                       |
| **COMMAND** | 命令     | 启动进程的命令或程序名                        |

**进程状态（S 列）详解**：

| 状态  | 说明       | 含义                            |
| ----- | ---------- | ------------------------------- |
| **R** | Running    | 运行中或在运行队列中等待        |
| **S** | Sleeping   | 可中断睡眠（等待事件）          |
| **D** | Disk sleep | 不可中断睡眠（通常是 I/O 等待） |
| **Z** | Zombie     | 僵尸进程（已终止但未被回收）    |
| **T** | Stopped    | 被停止（Ctrl+Z 或信号）         |
| **I** | Idle       | 内核空闲线程                    |

**内存字段详解**：

```
VIRT（虚拟内存）：
- 包含：代码段 + 数据段 + 堆 + 栈 + 共享库 + mmap 映射
- 特点：可能很大，但不代表实际占用
- 示例：Java 程序 VIRT 可能有几个 GB

RES（常驻内存，重要）：
- 实际占用的物理内存
- 不包括被换出的页
- 这是评估程序实际内存占用的关键指标

SHR（共享内存）：
- 与其他进程共享的内存
- 包括：共享库、共享内存段
- RES - SHR = 进程独占内存

关系：
VIRT ≥ RES ≥ SHR
```

**%CPU 字段说明**：

```
单核系统：
- 最大 100%（占满一个核心）

多核系统（如 4 核）：
- 单线程进程：最大 100%
- 多线程进程：最大 400%（4 核 × 100%）

例如：
%CPU = 350% → 占用了 3.5 个核心的资源
```

---

## 基本使用方法

**1. 启动和退出**

```bash
# 启动 top
top

# 退出 top
按 q 键

# 以批处理模式运行（非交互）
top -b

# 批处理模式，只显示一次
top -b -n 1

# 输出到文件
top -b -n 1 > top_output.txt
```

**2. 常用快捷键**

**显示控制**：

| 快捷键         | 功能         | 说明                  |
| -------------- | ------------ | --------------------- |
| **h** 或 **?** | 帮助         | 显示快捷键帮助        |
| **q**          | 退出         | 退出 top              |
| **d** 或 **s** | 刷新间隔     | 修改刷新时间（秒）    |
| **空格**       | 立即刷新     | 手动刷新显示          |
| **1**          | CPU 详情     | 切换显示每个 CPU 核心 |
| **t**          | CPU 显示模式 | 切换 CPU 行的显示样式 |
| **m**          | 内存显示模式 | 切换内存行的显示样式  |
| **l**          | 负载行       | 切换显示负载行        |
| **i**          | 空闲进程     | 隐藏/显示空闲进程     |
| **c**          | 命令行       | 切换显示完整命令行    |
| **V**          | 树形显示     | 按进程树显示          |

**排序**：

| 快捷键 | 功能           | 说明                       |
| ------ | -------------- | -------------------------- |
| **P**  | 按 CPU 排序    | 默认按 %CPU 降序（大写 P） |
| **M**  | 按内存排序     | 按 %MEM 降序               |
| **T**  | 按运行时间排序 | 按 TIME+ 降序              |
| **N**  | 按 PID 排序    | 按进程 ID 排序             |
| **<**  | 向左移动排序列 | 切换排序字段               |
| **>**  | 向右移动排序列 | 切换排序字段               |
| **R**  | 反转排序       | 升序/降序切换              |

**过滤和搜索**：

| 快捷键         | 功能       | 说明                 |
| -------------- | ---------- | -------------------- |
| **u**          | 指定用户   | 只显示指定用户的进程 |
| **L**          | 查找字符串 | 搜索命令名           |
| **o** 或 **O** | 过滤器     | 添加过滤条件         |
| **=**          | 清除过滤   | 清除所有过滤器       |

**进程管理**：

| 快捷键 | 功能         | 说明                       |
| ------ | ------------ | -------------------------- |
| **k**  | 杀死进程     | 输入 PID 和信号（默认 15） |
| **r**  | 修改 nice 值 | 调整进程优先级             |

**3. 命令行参数**

```bash
# 基本用法
top [选项]

# 常用参数：

# -d <秒数>：设置刷新间隔
top -d 2            # 每 2 秒刷新一次

# -n <次数>：指定刷新次数后退出
top -n 3            # 刷新 3 次后退出

# -p <PID>：监控指定进程
top -p 1234         # 只显示 PID 1234
top -p 1234,5678    # 监控多个进程

# -u <用户>：只显示指定用户的进程
top -u root         # 只显示 root 用户的进程
top -u www          # 只显示 www 用户的进程

# -b：批处理模式（非交互，适合脚本）
top -b -n 1 > output.txt

# -H：线程模式（显示线程而不是进程）
top -H              # 显示所有线程
top -H -p 1234      # 显示进程 1234 的所有线程

# -i：不显示空闲进程
top -i

# -c：显示完整命令行
top -c

# -o <字段>：按指定字段排序
top -o %CPU         # 按 CPU 排序
top -o %MEM         # 按内存排序
top -o TIME+        # 按运行时间排序

# 组合使用
top -d 1 -n 10 -b -u www -o %MEM > www_processes.txt
# 每秒刷新，共 10 次，批处理模式，只看 www 用户，按内存排序，输出到文件
```

---

## 实用技巧和高级用法

**1. 查看多核 CPU 详情**

```bash
# 启动 top 后按 1 键
top
# 按 1

# 显示效果：
%Cpu0  :  5.0 us,  2.0 sy,  0.0 ni, 92.5 id,  0.5 wa,  0.0 hi,  0.0 si,  0.0 st
%Cpu1  :  8.2 us,  3.1 sy,  0.0 ni, 88.2 id,  0.5 wa,  0.0 hi,  0.0 si,  0.0 st
%Cpu2  :  3.5 us,  1.5 sy,  0.0 ni, 94.5 id,  0.5 wa,  0.0 hi,  0.0 si,  0.0 st
%Cpu3  : 95.0 us,  5.0 sy,  0.0 ni,  0.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
                                                         ↑
                                          CPU3 接近满负载，可能是单线程瓶颈
```

**2. 查看完整命令行**

```bash
# 按 c 键切换
top
# 按 c

# 显示效果：
COMMAND
mysqld --datadir=/var/lib/mysql --socket=/var/lib/mysql/mysql.sock
nginx: worker process
/usr/bin/python3 /opt/app/main.py --config=/etc/app/config.yaml
```

**3. 过滤特定进程**

```bash
# 交互模式中按 o 或 O
top
# 按 o
# 输入：COMMAND=nginx
# 只显示命令中包含 nginx 的进程

# 其他过滤示例：
# %CPU>50        # CPU 占用超过 50%
# %MEM>10        # 内存占用超过 10%
# RES>1000000    # 常驻内存超过 1GB

# 清除过滤：按 =
```

**4. 按用户过滤**

```bash
# 交互模式中按 u
top
# 按 u
# 输入用户名：www
# 只显示 www 用户的进程

# 或命令行指定
top -u www
```

**5. 杀死进程**

```bash
# 在 top 中按 k
top
# 按 k
# 输入 PID：1234
# 输入信号：15（默认，SIGTERM）

# 常用信号：
# 15 (SIGTERM)：优雅终止（默认）
# 9 (SIGKILL)：强制杀死
# 1 (SIGHUP)：重新加载配置
```

**6. 调整进程优先级**

```bash
# 在 top 中按 r
top
# 按 r
# 输入 PID：1234
# 输入 nice 值：-5（需要 root 权限）

# Nice 值范围：-20 到 19
# -20：最高优先级
# 0：默认优先级
# 19：最低优先级
```

**7. 显示进程树**

```bash
# 按 V 键（大写）
top
# 按 V

# 显示父子进程关系（类似 pstree）
```

**8. 监控特定进程及其线程**

```bash
# 监控进程 1234 的所有线程
top -H -p 1234

# 输出示例：
  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
 1234 mysql     20   0 5123456 512000  10240 S   5.0   3.1  10:23.45 mysqld
 1235 mysql     20   0 5123456 512000  10240 S   2.0   3.1   5:12.34 mysqld-thread1
 1236 mysql     20   0 5123456 512000  10240 S   1.5   3.1   3:45.67 mysqld-thread2
 1237 mysql     20   0 5123456 512000  10240 R  10.0   3.1  15:34.21 mysqld-thread3
                                                        ^^^^
                                                        某个线程 CPU 高
```

**9. 保存 top 配置**

```bash
# 在 top 中按 W（大写）
top
# 调整好显示方式（按 1, c, V 等）
# 按 W
# 配置保存到 ~/.toprc

# 下次启动 top 自动应用配置
```

**10. 批处理模式用于监控**

```bash
# 持续监控 CPU 和内存前 10 的进程
top -b -n 1440 -d 60 | grep -E "^(top|Tasks|Cpu|Mem|PID)" > top_monitor.log
# -n 1440：运行 1440 次（24 小时）
# -d 60：每 60 秒一次

# 只输出进程信息
top -b -n 1 | tail -n +8

# 监控特定进程的资源变化
watch -n 1 "top -b -n 1 -p 1234 | tail -n +8"
```

---

## 常见问题分析

**1. CPU 占用高的问题**

```bash
# 场景：系统响应缓慢

# 步骤 1：按 P 键按 CPU 排序（默认）
top
# 按 P

# 查看 %CPU 列，找到占用高的进程
  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
 5678 app       20   0 2048576 256000   8192 R  95.0   1.6 120:45.67 my_app
                                                  ^^^^
                                                  异常高

# 步骤 2：查看具体是哪个线程（按 H 切换线程模式）
top -H -p 5678

# 步骤 3：记录 CPU 高的线程 PID，用其他工具分析
# 如使用 perf、strace、pstack 等

# 分析 CPU 高的常见原因：
# - 死循环
# - 大量计算
# - 正则表达式性能问题
# - 锁竞争导致忙等待
```

**2. 内存占用高的问题**

```bash
# 按 M 键按内存排序
top
# 按 M

# 查看 %MEM 和 RES 列
  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
 1234 mysql     20   0 8192000 7168000  10240 S   5.0  43.8  10:23.45 mysqld
                              ^^^^^^^       ^^^^
                              7GB 物理内存   43.8%

# 重点关注：
# 1. RES：实际物理内存占用
# 2. %MEM：内存占用百分比
# 3. VIRT 很大但 RES 小：可能是内存映射文件，正常
# 4. RES 持续增长：可能是内存泄漏

# 进一步分析
ps aux | grep <pid>
pmap -x <pid>           # 查看内存映射详情
cat /proc/<pid>/status  # 查看详细内存信息
```

**3. I/O 等待高的问题**

```bash
# 查看 %Cpu(s) 行的 wa（iowait）
%Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 72.3 id, 20.3 wa,  0.0 hi,  0.1 si,  0.0 st
                                              ^^^^^
                                              I/O 等待高

# 找到处于 D 状态的进程
top
# 按 S 列观察，查找状态为 D 的进程

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
 2345 root      20   0  512000  64000   2048 D   0.0   0.4   1:23.45 backup
                                                  ^
                                                  不可中断睡眠（I/O 等待）

# 进一步分析
iostat -x 1         # 查看磁盘 I/O
iotop               # 查看进程 I/O 占用
```

**4. 僵尸进程过多**

```bash
# 查看 Tasks 行
Tasks: 245 total,   1 running, 230 sleeping,   0 stopped,  14 zombie
                                                            ^^^^^^^^^
                                                            14 个僵尸进程

# 找到僵尸进程
top
# 查找状态为 Z 的进程

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
 3456 app       20   0      0      0      0 Z   0.0   0.0   0:00.00 defunct
                                             ^
                                             僵尸进程

# 僵尸进程不占用资源，但数量多说明父进程有 bug
# 解决方法：杀死父进程或重启父进程

# 找到父进程
ps -ef | grep <zombie_pid>
# 或
cat /proc/<zombie_pid>/status | grep PPid

# 杀死父进程
kill -9 <parent_pid>
# 或者让父进程正确处理 SIGCHLD 信号
```

**5. Swap 使用高的问题**

```bash
# 查看 Swap 行
MiB Swap:   8192.0 total,   1024.0 free,   7168.0 used.   1500.5 avail Mem
                                           ^^^^^          ^^^^^
                                           87% Swap 被用   可用内存很少

# 严重性能问题信号！

# 解决步骤：

# 1. 找出内存占用最大的进程
top
# 按 M 排序

# 2. 优化或重启占用大的进程

# 3. 清理缓存（临时措施）
sync
echo 3 > /proc/sys/vm/drop_caches

# 4. 减少 swap 使用倾向
echo 10 > /proc/sys/vm/swappiness  # 降低换页积极性

# 5. 根本解决：增加物理内存或优化应用
```

---

## top 的替代工具

**htop - 增强版 top**：

```bash
# 安装
sudo apt install htop   # Debian/Ubuntu
sudo yum install htop   # CentOS/RHEL

# 优势：
# 1. 彩色界面，更直观
# 2. 支持鼠标操作
# 3. 横向显示所有 CPU 核心
# 4. 可以直接滚动查看所有进程
# 5. 更友好的进程树显示

# 启动
htop

# 快捷键：
# F1: 帮助
# F2: 设置
# F3: 搜索
# F4: 过滤
# F5: 树形视图
# F6: 排序
# F9: 杀死进程
# F10: 退出
```

**atop - 高级系统监控**：

```bash
# 安装
sudo apt install atop

# 优势：
# 1. 历史数据记录
# 2. 更详细的磁盘和网络统计
# 3. 可以回放历史数据

# 启动
atop

# 查看历史
atop -r /var/log/atop/atop_20231220
```

**glances - 全面系统监控**：

```bash
# 安装
pip install glances

# 优势：
# 1. 跨平台（Linux/Mac/Windows）
# 2. 显示更多信息（网络、磁盘 I/O、传感器）
# 3. 支持 Web 界面
# 4. 可以导出数据

# 启动
glances

# Web 模式
glances -w
# 访问 http://localhost:61208
```

---

#### 实战案例

**案例 1：定位 CPU 占用高的线程**

```bash
# 1. 找到 CPU 占用高的进程
top -d 1
# 按 P 排序，记录 PID（假设 1234）

# 2. 查看该进程的所有线程
top -H -p 1234
# 找到 CPU 高的线程 PID（假设 1250）

# 3. 将线程 PID 转为 16 进制
printf "0x%x\n" 1250
# 输出：0x4e2

# 4. 查看线程堆栈
jstack 1234 | grep 0x4e2 -A 30
# 或
pstack 1234 | grep -A 10 1250

# 5. 分析代码，找到 CPU 热点
```

**案例 2：监控系统资源变化**

```bash
# 创建监控脚本
cat > monitor.sh << 'EOF'
#!/bin/bash
while true; do
    echo "=== $(date) ==="
    top -b -n 1 | head -5
    echo ""
    top -b -n 1 | head -12 | tail -5
    echo ""
    sleep 60
done
EOF

chmod +x monitor.sh
./monitor.sh | tee monitor.log

# 定时记录，方便事后分析
```

**案例 3：找出内存泄漏**

```bash
# 持续监控某个进程的内存
watch -n 5 "top -b -n 1 -p 1234 | tail -1"

# 或使用脚本记录
while true; do
    date >> mem_monitor.log
    top -b -n 1 -p 1234 | grep 1234 >> mem_monitor.log
    sleep 60
done

# 分析日志，看 RES 是否持续增长
awk '{print $1, $2, $6}' mem_monitor.log | grep 1234
```

---

### 相关高频面试题

#### Q1: top 命令中 load average 的三个值是什么意思？如何判断系统负载是否正常？

**答案**：

**load average 含义**：
- 第一个值：最近 1 分钟的平均负载
- 第二个值：最近 5 分钟的平均负载  
- 第三个值：最近 15 分钟的平均负载

**负载的定义**：
处于运行状态（R）和不可中断睡眠状态（D）的进程平均数量。

**判断标准**（假设 N 核 CPU）：
```
负载值 < N：系统正常，有空闲资源
负载值 = N：系统满负载
负载值 > N：系统过载，有进程在等待

例如 4 核 CPU：
load average: 2.5, 2.8, 3.0
- 2.5 < 4：正常，CPU 利用率约 62.5%
- 趋势平稳，系统稳定

load average: 6.0, 5.5, 4.8
- 6.0 > 4：过载，有约 2 个进程在等待
- 数值递增：负载在上升，需要关注

load average: 0.5, 2.0, 4.0
- 数值递减：负载在下降，情况好转
```

**分析技巧**：
```bash
# 查看 CPU 核心数
nproc
# 或
grep -c processor /proc/cpuinfo

# 负载高的原因：
# 1. CPU 密集：检查 top 中 %CPU 高的进程
# 2. I/O 等待：检查 D 状态进程，wa 值高
# 3. 进程过多：检查 Tasks 总数

# 对应措施：
# CPU 密集 → 优化代码或增加 CPU
# I/O 等待 → 优化磁盘 I/O 或使用 SSD  
# 进程过多 → 减少并发或优化调度
```

#### Q2: top 中的 VIRT、RES、SHR 三个内存指标有什么区别？哪个最重要？

**答案**：

**VIRT（虚拟内存）**：
```
定义：进程可以访问的所有虚拟内存
包含：
- 代码段
- 数据段（堆、栈）
- 共享库
- mmap 映射的文件
- malloc 分配但未使用的内存

特点：
- 可能很大（GB 级别）
- 不代表实际占用
- Java 程序 VIRT 通常很大（预分配堆）
```

**RES（常驻内存，最重要）**：
```
定义：实际占用的物理内存（不包括 swap）
特点：
- 真实内存占用
- 评估内存使用的关键指标
- 包括私有内存和共享内存
- 这个值大才是真正占用内存多

判断内存问题：
- RES 持续增长 → 可能内存泄漏
- RES 突然增大 → 程序加载大量数据
- RES 接近物理内存 → 内存不足
```

**SHR（共享内存）**：
```
定义：与其他进程共享的内存
包含：
- 共享库（如 libc.so）
- 共享内存段（System V 或 POSIX）
- mmap 映射的共享文件

计算：
进程独占内存 ≈ RES - SHR

示例：
VIRT: 2048000 KB
RES:   512000 KB  ← 实际占用 500 MB
SHR:    64000 KB  ← 64 MB 是共享的
独占:  448000 KB  ← 真正独占 448 MB
```

**实际案例**：
```bash
# MySQL 进程
  PID USER   VIRT    RES    SHR
 1234 mysql  8.0g   4.0g   20m

分析：
- VIRT 8GB：预分配了大量虚拟内存
- RES 4GB：实际使用 4GB 物理内存（重要）
- SHR 20MB：共享库占 20MB
- 独占：4GB - 20MB ≈ 3.98GB

# 多个 nginx worker
  PID USER   VIRT    RES    SHR
 2001 www   500m    50m    30m
 2002 www   500m    50m    30m
 2003 www   500m    50m    30m

分析：
- 每个进程 RES 50MB
- 但 SHR 30MB（共享 nginx 二进制和库）
- 3 个进程总内存 ≈ 3×20MB + 30MB = 90MB
  （不是 3×50MB = 150MB）
```

#### Q3: top 中 CPU 的 wa（iowait）值高说明什么问题？如何排查？

**答案**：

**iowait 的含义**：
```
定义：CPU 空闲且等待 I/O 完成的时间百分比

关键点：
- 不是 CPU 在做 I/O（CPU 不能直接做 I/O）
- 而是 CPU 空闲，因为进程在等待 I/O
- I/O 由磁盘控制器或 DMA 完成

wa 高的含义：
- 有很多进程在等待 I/O 完成
- CPU 空闲但无法做其他事（进程阻塞在 I/O）
- 系统瓶颈在磁盘，不在 CPU
```

**判断标准**：
```
wa < 5%：正常，I/O 不是瓶颈
wa 5-20%：需要关注
wa > 20%：I/O 瓶颈严重
wa > 50%：严重 I/O 问题
```

**排查步骤**：

```bash
# 1. 确认是 I/O 问题
top
# 观察 wa 值和 D 状态进程

%Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 72.3 id, 20.3 wa
                                              ^^^^^
                                              I/O 等待高

# 2. 查看磁盘 I/O 统计
iostat -x 1 5

# 关注指标：
# %util：磁盘繁忙程度（>80% 说明接近饱和）
# await：平均等待时间（>20ms 较慢）
# r/s, w/s：每秒读写次数
# rkB/s, wkB/s：每秒读写量

# 3. 找出 I/O 占用高的进程
iotop -o
# 或
pidstat -d 1

# 4. 查看具体文件操作
lsof -p <pid>
strace -p <pid> -e trace=read,write,open,close

# 5. 分析进程状态
ps aux | grep D
# 找出处于 D 状态的进程

# 6. 检查磁盘健康
dmesg | grep -i error
smartctl -a /dev/sda
```

**常见原因和解决**：
```
1. 磁盘性能不足
   解决：使用 SSD、RAID、优化文件系统

2. 大量小文件 I/O
   解决：批量操作、使用缓存、异步 I/O

3. 数据库查询慢
   解决：添加索引、优化查询、增加缓存

4. 日志写入频繁
   解决：异步日志、批量写入、使用内存文件系统

5. Swap 频繁换页
   解决：增加内存、优化应用

6. 网络文件系统（NFS）慢
   解决：优化网络、使用本地缓存
```

#### Q4: 如何用 top 命令找出内存泄漏的进程？

**答案**：

**内存泄漏的特征**：
- RES（常驻内存）持续增长
- 不会释放已分配的内存
- 最终可能导致 OOM（Out of Memory）

**排查方法**：

**方法 1：持续观察 RES 变化**
```bash
# 监控特定进程
top -d 5 -p <pid>
# 按 M 按内存排序
# 观察 RES 列是否持续增长

# 记录内存变化
while true; do
    echo -n "$(date '+%H:%M:%S') "
    top -b -n 1 -p <pid> | grep <pid> | awk '{print $6}'
    sleep 60
done | tee mem_growth.log

# 分析趋势
# 正常：RES 稳定或有升有降
# 泄漏：RES 只升不降
```

**方法 2：对比多个时间点**
```bash
# 获取初始内存
top -b -n 1 -p <pid> | grep <pid> > mem_before.txt

# 等待一段时间（如 1 小时）
sleep 3600

# 获取当前内存
top -b -n 1 -p <pid> | grep <pid> > mem_after.txt

# 对比
diff mem_before.txt mem_after.txt
```

**方法 3：使用脚本监控所有进程**
```bash
#!/bin/bash
# find_memory_leak.sh

echo "监控 5 分钟，找出内存增长的进程"

# 第一次采样
ps aux | awk 'NR>1 {print $2, $6}' | sort -n > /tmp/mem1.txt
sleep 300  # 5 分钟

# 第二次采样
ps aux | awk 'NR>1 {print $2, $6}' | sort -n > /tmp/mem2.txt

# 对比并找出增长的进程
join /tmp/mem1.txt /tmp/mem2.txt | awk '{
    pid=$1
    mem1=$2
    mem2=$3
    growth=mem2-mem1
    if (growth > 10000) {  # 增长超过 10MB
        print pid, growth/1024 "MB", mem2/1024 "MB"
    }
}' | sort -k2 -rn | head -10

echo "PID   增长量   当前内存"
```

**方法 4：结合其他工具**
```bash
# 查看进程内存映射
pmap -x <pid>

# 查看详细内存信息
cat /proc/<pid>/status
# 关注：
# VmRSS：常驻内存
# VmData：数据段大小
# VmStk：栈大小

# 使用 valgrind 检测（开发环境）
valgrind --leak-check=full ./my_program

# 使用 pmap 持续监控
watch -n 60 "pmap -x <pid> | tail -1"
```

**典型案例**：
```bash
# 场景：发现某个 Python 进程内存持续增长

# 1. 确认是内存泄漏
top -d 10 -p 1234
# RES 从 500M → 800M → 1.2G（30 分钟内）

# 2. 查看详细内存
pmap -x 1234 | grep -i heap
# heap 区持续增长

# 3. 使用 Python 内存分析工具
# 在代码中添加：
import tracemalloc
tracemalloc.start()
# ... 运行一段时间
snapshot = tracemalloc.take_snapshot()
top_stats = snapshot.statistics('lineno')
for stat in top_stats[:10]:
    print(stat)

# 4. 找到泄漏点并修复
```

#### Q5: top 显示某个进程 %CPU 超过 100%（如 350%）是什么意思？

**答案**：

**%CPU 超过 100% 的含义**：

```
单核系统：
- %CPU 最大 100%（占满一个 CPU 核心）
- 超过 100% → 不可能

多核系统（如 4 核）：
- %CPU 最大 400%（4 × 100%）
- %CPU = 350% → 占用了 3.5 个核心
- 说明程序是多线程的
```

**计算方式**：
```
top 的 %CPU 计算：
%CPU = (进程 CPU 时间 / 总 CPU 时间) × 100 × CPU 核心数

例如 4 核 CPU，1 秒内：
- 进程使用了 3.5 秒的 CPU 时间
- 总 CPU 时间 = 4 秒
- %CPU = (3.5 / 4) × 100 × 4 = 350%

或者：
- 进程有 4 个线程都在满负载运行
- 其中 3 个 100%，1 个 50%
- 总计 350%
```

**查看具体线程占用**：
```bash
# 方法 1：使用 top 的线程模式
top -H -p <pid>

# 示例输出（假设 PID 1234）：
  PID USER   %CPU  COMMAND
 1234 app    0.0   my_app (主线程，可能在等待)
 1235 app   100.0  my_app (工作线程1)
 1236 app   100.0  my_app (工作线程2)
 1237 app   100.0  my_app (工作线程3)
 1238 app    50.0  my_app (工作线程4)
                   ^^^^^ 总计 350%

# 方法 2：使用 ps
ps -eLo pid,tid,ppid,pcpu,comm | grep <pid>

# 方法 3：使用 htop
htop -p <pid>
# 按 H 显示线程
```

**实际意义**：
```
场景 1：%CPU = 100%（单核系统或单线程程序）
- 程序占满了一个 CPU 核心
- 可能是计算密集型任务
- 如果持续 100%，检查是否死循环

场景 2：%CPU = 350%（4 核系统）
- 程序有效利用了多核（3.5 个核心）
- 通常是正常的多线程程序
- 良好的并行性能

场景 3：%CPU = 50%（4 核系统）
- 可能是：
  1. 单线程程序（占用半个核心）
  2. I/O 密集型（部分时间在等待 I/O）
  3. 程序限流（主动控制 CPU 使用）

场景 4：%CPU = 800%（4 核系统）
- 错误！超过了核心总数
- 可能是统计错误或系统问题
- 实际不可能超过 400%
```

**优化建议**：
```
如果 %CPU 低（如 4 核但只有 50%）：
1. 检查是否充分利用多线程
2. 看是否被 I/O 阻塞（wa 高）
3. 检查是否有锁竞争

如果 %CPU 高（接近核心数上限）：
1. 正常（如果是计算密集型任务）
2. 检查是否有死循环或性能问题
3. 考虑优化算法降低 CPU 消耗

检查线程利用率：
# 查看每个线程的 CPU 使用
top -H -p <pid>
# 如果线程数远大于 CPU 核心数，但 %CPU 不高
# 说明线程之间可能有阻塞或等待
```

**示例分析**：
```bash
# 4 核 CPU，某个 Java 程序显示 %CPU = 380%

# 1. 查看线程分布
top -H -p <pid>

# 可能的情况：
# a) 4 个线程各占 95% → 充分利用多核，正常
# b) 38 个线程各占 10% → 线程过多，频繁切换
# c) 1 个线程 380% → 错误，不可能

# 2. 如果是情况 b，优化方案：
# - 减少线程数（使用线程池）
# - 检查线程是否在忙等待
# - 使用异步 I/O 代替多线程
```

---

### 关键点总结

**top 核心功能**：
- 实时监控系统资源使用情况
- 按 CPU、内存、时间等排序进程
- 交互式管理进程（杀死、调优先级）

**重要指标**：
- **load average**：系统负载（< CPU 核心数正常）
- **%CPU wa**：I/O 等待（< 5% 正常）
- **RES**：实际物理内存占用（最重要）
- **%CPU**：CPU 占用（多核可超 100%）
- **Swap used**：交换分区使用（应接近 0）

**常用快捷键**：
```
P - 按 CPU 排序
M - 按内存排序
1 - 显示每个 CPU 核心
c - 显示完整命令
k - 杀死进程
h - 帮助
q - 退出
```

**替代工具**：
- `htop`：更友好的界面
- `atop`：历史数据记录
- `glances`：全面的系统监控
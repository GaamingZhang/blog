---
date: 2026-01-06
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 操作系统
tag:
  - 操作系统
---

# /proc/\<pid\>/ 路径下的文件与作用

## 一、什么是/proc文件系统

/proc文件系统是Linux内核提供的一种伪文件系统，它以文件系统的方式为用户空间提供了内核数据的访问接口。/proc文件系统并不存储在磁盘上，而是存在于内存中，由内核动态生成和管理。通过读取和写入/proc下的文件，用户可以获取系统信息、进程状态，甚至修改内核参数。

/proc文件系统具有以下特点：
- 所有文件都位于/proc目录下，采用树形结构组织
- 文件内容由内核动态生成，不占用实际磁盘空间
- 大多数文件是只读的，少数文件支持写入以修改内核参数
- 文件权限通常为644或444，限制写入操作
- 文件名通常为数字或特定名称，具有特定含义

## 二、/proc/\<pid\>/目录概述

在/proc文件系统中，每个运行中的进程都有一个以其PID（进程ID）命名的子目录，如/proc/1/代表PID为1的进程（通常是init或systemd）。这些目录包含了对应进程的详细信息，是系统管理员和开发者调试、监控进程的重要工具。

/proc/\<pid\>/目录下的文件和子目录可以分为以下几类：
- 进程基本信息（如命令行、状态、优先级等）
- 内存管理信息（如内存映射、堆和栈的使用情况）
- 文件描述符和文件系统信息
- 线程和任务信息
- CPU使用情况
- 网络连接信息
- 环境变量和资源限制

## 三、进程基本信息文件

### 1. /proc/\<pid\>/cmdline

存储进程的命令行参数，各参数之间以NULL字符（\0）分隔。

**示例内容**：
```
bash\0-c\0ls\0-l
```

**读取方法**：
```bash
cat /proc/1234/cmdline | tr '\0' ' '
# 输出：bash -c ls -l
```

### 2. /proc/\<pid\>/comm

存储进程的命令名（不包含路径和参数）。

**示例内容**：
bash

### 3. /proc/\<pid\>/status

包含进程的状态信息，采用键值对格式，内容丰富。

**主要字段**：
- Name：进程名称
- State：进程状态（R-运行, S-睡眠, D-不可中断睡眠, Z-僵尸, T-停止, X-死亡）
- Tgid：线程组ID（与PID相同，除非是线程）
- Pid：进程ID
- PPid：父进程ID
- TracerPid：跟踪该进程的进程ID
- Uid：用户ID（Real, Effective, Saved Set, File System）
- Gid：组ID（Real, Effective, Saved Set, File System）
- FDSize：文件描述符表大小
- Groups：附加组ID列表
- NStgid：命名空间中的线程组ID
- NSpid：命名空间中的进程ID
- NSpgid：命名空间中的进程组ID
- NSsid：命名空间中的会话ID
- VmPeak：虚拟内存峰值
- VmSize：当前虚拟内存大小
- VmLck：锁定的内存大小
- VmPin：固定的内存大小
- VmHWM：物理内存使用峰值
- VmRSS：当前物理内存使用大小
- VmData：数据段大小
- VmStk：栈段大小
- VmExe：代码段大小
- VmLib：共享库大小
- VmPTE：页表项大小
- VmPMD：页中间目录大小
- VmSwap：交换空间使用大小
- HugetlbPages：大页内存使用大小
- Threads：线程数量
- SigQ：信号队列信息（当前/最大）
- SigPnd：待处理信号（线程）
- ShdPnd：待处理信号（进程）
- SigBlk：阻塞的信号
- SigIgn：忽略的信号
- SigCgt：捕获的信号
- CapInh：继承的能力
- CapPrm：有效的能力
- CapEff：有效的能力
- CapBnd：边界能力
- CapAmb：可继承的能力
- NoNewPrivs：是否禁止提升权限
- Seccomp：seccomp过滤模式
- Cpus_allowed：CPU亲和性掩码
- Cpus_allowed_list：CPU亲和性列表
- Mems_allowed：内存节点亲和性掩码
- Mems_allowed_list：内存节点亲和性列表
- voluntary_ctxt_switches：自愿上下文切换次数
- nonvoluntary_ctxt_switches：非自愿上下文切换次数

### 4. /proc/\<pid\>/stat

包含进程的状态信息，采用空格分隔的数值字段，适合程序解析。

**字段说明（部分）**：
1. pid：进程ID
2. comm：命令名（带括号）
3. state：进程状态
4. ppid：父进程ID
5. pgrp：进程组ID
6. session：会话ID
7. tty_nr：控制终端
8. tpgid：终端进程组ID
9. flags：进程标志
10. minflt：次要页错误次数
11. cminflt：子进程次要页错误次数
12. majflt：主要页错误次数
13. cmajflt：子进程主要页错误次数
14. utime：用户态CPU时间（jiffies）
15. stime：内核态CPU时间（jiffies）
16. cutime：子进程用户态CPU时间（jiffies）
17. cstime：子进程内核态CPU时间（jiffies）
18. priority：动态优先级
19. nice：静态优先级
20. num_threads：线程数量
21. itrealvalue：间隔定时器值
22. starttime：进程启动时间（jiffies）
23. vsize：虚拟内存大小（字节）
24. rss：驻留集大小（页）
25. rsslim：驻留集大小限制（字节）

### 5. /proc/\<pid\>/statm

包含进程的内存使用统计信息，以页为单位。

**字段说明**：
1. size：虚拟内存大小（页）
2. resident：驻留集大小（页）
3. shared：共享内存大小（页）
4. text：文本段大小（页）
5. lib：共享库大小（页）
6. data：数据段和栈大小（页）
7. dt：脏页数量（页）

## 四、内存管理相关文件

### 1. /proc/\<pid\>/maps

显示进程的内存映射信息，包括地址范围、权限、偏移量、设备号、inode号和映射的文件路径。

**示例内容**：
```
00400000-00452000 r-xp 00000000 08:01 12345678  /bin/bash
00651000-00652000 r--p 00051000 08:01 12345678  /bin/bash
00652000-00655000 rw-p 00052000 08:01 12345678  /bin/bash
00655000-0068e000 rw-p 00000000 00:00 0          [heap]
7f1234567000-7f123458e000 r-xp 00000000 08:01 23456789  /lib/x86_64-linux-gnu/libtinfo.so.5.9
7f123458e000-7f123478d000 ---p 00027000 08:01 23456789  /lib/x86_64-linux-gnu/libtinfo.so.5.9
7f123478d000-7f1234791000 r--p 00026000 08:01 23456789  /lib/x86_64-linux-gnu/libtinfo.so.5.9
7f1234791000-7f1234792000 rw-p 0002a000 08:01 23456789  /lib/x86_64-linux-gnu/libtinfo.so.5.9
7f1234792000-7f12347b8000 r-xp 00000000 08:01 34567890  /lib/x86_64-linux-gnu/libdl-2.31.so
7f12347b8000-7f12349b7000 ---p 00026000 08:01 34567890  /lib/x86_64-linux-gnu/libdl-2.31.so
7f12349b7000-7f12349b8000 r--p 00025000 08:01 34567890  /lib/x86_64-linux-gnu/libdl-2.31.so
7f12349b8000-7f12349b9000 rw-p 00026000 08:01 34567890  /lib/x86_64-linux-gnu/libdl-2.31.so
7f12349b9000-7f1234b7f000 r-xp 00000000 08:01 45678901  /lib/x86_64-linux-gnu/libc-2.31.so
7f1234b7f000-7f1234d7e000 ---p 001c6000 08:01 45678901  /lib/x86_64-linux-gnu/libc-2.31.so
7f1234d7e000-7f1234d82000 r--p 001c5000 08:01 45678901  /lib/x86_64-linux-gnu/libc-2.31.so
7f1234d82000-7f1234d84000 rw-p 001c9000 08:01 45678901  /lib/x86_64-linux-gnu/libc-2.31.so
7f1234d84000-7f1234d88000 rw-p 00000000 00:00 0
7f1234d88000-7f1234db0000 r-xp 00000000 08:01 56789012  /lib/x86_64-linux-gnu/ld-2.31.so
7f1234f74000-7f1234f77000 rw-p 00000000 00:00 0
7f1234f9e000-7f1234f9f000 rw-p 00000000 00:00 0
7f1234f9f000-7f1234fa0000 r--p 00027000 08:01 56789012  /lib/x86_64-linux-gnu/ld-2.31.so
7f1234fa0000-7f1234fa1000 rw-p 00028000 08:01 56789012  /lib/x86_64-linux-gnu/ld-2.31.so
7f1234fa1000-7f1234fa2000 rw-p 00000000 00:00 0
7ffd12345000-7ffd12366000 rw-p 00000000 00:00 0          [stack]
7ffd123ff000-7ffd12400000 r-xp 00000000 00:00 0          [vdso]
ffffffffff600000-ffffffffff601000 r-xp 00000000 00:00 0  [vsyscall]
```

**字段说明**：
- 地址范围：如00400000-00452000，表示内存映射的起始和结束地址
- 权限：r-xp表示可读、可执行、不可写、私有映射
  - r：可读
  - w：可写
  - x：可执行
  - p：私有映射（private）
  - s：共享映射（shared）
- 偏移量：文件在内存中的偏移量
- 设备号：文件所在设备的主设备号和次设备号（如08:01）
- inode号：文件的inode号
- 路径：映射的文件路径，或特殊区域如[heap]、[stack]、[vdso]等

### 2. /proc/\<pid\>/smaps

显示进程的详细内存映射信息，比maps更详细，包含每个映射区域的内存使用情况。

**示例内容**：
```
00400000-00452000 r-xp 00000000 08:01 12345678  /bin/bash
Size:                328 kB
KernelPageSize:        4 kB
MMUPageSize:           4 kB
Rss:                  88 kB
Pss:                  88 kB
Shared_Clean:          0 kB
Shared_Dirty:          0 kB
Private_Clean:        88 kB
Private_Dirty:         0 kB
Referenced:           88 kB
Anonymous:             0 kB
LazyFree:              0 kB
AnonHugePages:         0 kB
ShmemPmdMapped:        0 kB
FilePmdMapped:         0 kB
Shared_Hugetlb:        0 kB
Private_Hugetlb:       0 kB
Swap:                  0 kB
SwapPss:               0 kB
Locked:                0 kB
THPeligible:    0
VmFlags: rd ex mr mw me dw sd
```

### 3. /proc/\<pid\>/pagemap

提供进程页表的访问接口，用于查询物理页和虚拟页的映射关系。该文件为二进制格式，需要特殊工具解析。

**页表项格式**：
每个页表项占用8字节（64位），包含以下信息：
- 第0位：页存在标志（PTE_P）
- 第1位：页脏标志（PTE_D）
- 第2位：页引用标志（PTE_A）
- 第3位：页保护标志（PTE_U）
- 第55-62位：物理页帧号（PFN）

**使用示例**：
```bash
# 查看进程的某个虚拟地址对应的物理地址
pid=1234
virt_addr=0x7f1234567000
page_offset=$((virt_addr / 4096))
offset=$((page_offset * 8))

# 读取pagemap文件中的页表项
table_entry=$(hexdump -s $offset -n 8 -e '1/8 "%016x\n"' /proc/$pid/pagemap)

# 解析物理页帧号
pfn=$((0x${table_entry:9:7}))  # 提取第55-62位

# 计算物理地址
phys_addr=$((pfn * 4096 + (virt_addr % 4096)))

echo "虚拟地址: 0x$virt_addr"
echo "物理页帧号: $pfn"
echo "物理地址: 0x$phys_addr"
```

## 五、文件系统相关文件

### 1. /proc/\<pid\>/fd

包含进程打开的文件描述符的符号链接，每个链接指向对应的文件或设备。

**示例**：
```bash
$ ls -la /proc/1234/fd
lrwx------ 1 user user 64 Jan  6 10:00 0 -> /dev/pts/0
lrwx------ 1 user user 64 Jan  6 10:00 1 -> /dev/pts/0
lrwx------ 1 user user 64 Jan  6 10:00 2 -> /dev/pts/0
lr-x------ 1 user user 64 Jan  6 10:00 3 -> /proc/1234/stat
lr-x------ 1 user user 64 Jan  6 10:00 4 -> /proc/1234/status
lrwx------ 1 user user 64 Jan  6 10:00 5 -> 'socket:[123456]'
```

**文件描述符说明**：
- 0：标准输入（stdin）
- 1：标准输出（stdout）
- 2：标准错误（stderr）
- 3及以上：其他打开的文件、套接字、设备等

### 2. /proc/\<pid\>/fdinfo

包含进程打开的文件描述符的详细信息，每个文件对应一个文件描述符。

**示例内容**（/proc/1234/fdinfo/3）：
```
pos:    0
flags:  02000000
mnt_id: 1
ino:    12345
```

**字段说明**：
- pos：当前文件偏移量
- flags：文件打开标志（如O_RDONLY、O_WRONLY、O_RDWR等）
- mnt_id：挂载点ID
- ino：文件inode号

### 3. /proc/\<pid\>/root

指向进程的根目录的符号链接，通常为/，除非进程使用了chroot或namespace改变了根目录。

### 4. /proc/\<pid\>/cwd

指向进程当前工作目录的符号链接。

### 5. /proc/\<pid\>/mounts

显示进程可见的挂载点信息，与/etc/mtab类似，但反映的是进程命名空间中的挂载情况。

**示例内容**：
```
/dev/sda1 / ext4 rw,relatime,errors=remount-ro 0 0
/dev/sda2 /home ext4 rw,relatime 0 0
proc /proc proc rw,nosuid,nodev,noexec,relatime 0 0
sysfs /sys sysfs rw,nosuid,nodev,noexec,relatime 0 0
devtmpfs /dev devtmpfs rw,nosuid,size=4096k,nr_inodes=1024000,mode=755 0 0
devpts /dev/pts devpts rw,nosuid,noexec,relatime,gid=5,mode=620,ptmxmode=000 0 0
tmpfs /run tmpfs rw,nosuid,noexec,relatime,size=819200k,mode=755 0 0
```

### 6. /proc/\<pid\>/mountinfo

提供更详细的挂载点信息，格式更适合程序解析。

**示例内容**：
```
1 0 8:1 / / rw,relatime shared:1 - ext4 /dev/sda1 rw,errors=remount-ro
2 1 8:2 /home /home rw,relatime shared:2 - ext4 /dev/sda2 rw
3 1 0:2 / /proc rw,nosuid,nodev,noexec,relatime shared:3 - proc proc rw
4 1 0:3 / /sys rw,nosuid,nodev,noexec,relatime shared:4 - sysfs sys rw
5 1 0:4 / /dev rw,nosuid shared:5 - devtmpfs dev rw,size=4096k,nr_inodes=1024000,mode=755
6 1 0:5 / /dev/pts rw,nosuid,noexec,relatime shared:6 - devpts devpts rw,gid=5,mode=620,ptmxmode=000
7 1 0:6 / /run rw,nosuid,noexec,relatime shared:7 - tmpfs run rw,size=819200k,mode=755
```

## 六、线程和任务相关文件

### 1. /proc/\<pid\>/task

包含进程所有线程的子目录，每个子目录以线程ID（TID）命名，结构与/proc/\<pid\>/类似。

**示例**：
```bash
$ ls -la /proc/1234/task
total 0
dr-xr-xr-x 7 user user 0 Jan  6 10:00 .
dr-xr-xr-x 9 user user 0 Jan  6 10:00 ..
dr-xr-xr-x 6 user user 0 Jan  6 10:00 1234
dr-xr-xr-x 6 user user 0 Jan  6 10:00 1235
dr-xr-xr-x 6 user user 0 Jan  6 10:00 1236
dr-xr-xr-x 6 user user 0 Jan  6 10:00 1237
dr-xr-xr-x 6 user user 0 Jan  6 10:00 1238
```

### 2. /proc/\<pid\>/sched

包含进程的调度信息，包括调度策略、优先级、运行时间等。

**示例内容**：
```
bash (1234, #threads: 1)
-------------------------------------------------------------------
runnable tasks:
  task   PID         tree-key  switches  prio     wait-time             sum-exec        sum-sleep
-------------------------------------------------------------------
bash      1234     0.000000000       123     120     0.000000000        0.010000000        0.500000000

-------------------------------------------------------------------
affinity: 0-3
```

### 3. /proc/\<pid\>/schedstat

包含进程的调度统计信息，格式为三个数字：

**示例内容**：
```
1234567890 123456789 1234
```

**字段说明**：
1. 进程在CPU上运行的时间（纳秒）
2. 进程等待CPU的时间（纳秒）
3. 进程的上下文切换次数

## 七、CPU使用相关文件

### 1. /proc/\<pid\>/stat

包含进程的CPU使用时间信息，如用户态和内核态CPU时间（utime和stime字段）。

### 2. /proc/\<pid\>/schedstat

包含进程的调度统计信息，如运行时间、等待时间和上下文切换次数。

## 八、网络相关文件

### 1. /proc/\<pid\>/net

包含进程的网络连接信息，结构与/proc/net类似，但仅显示该进程的连接。

**主要子目录**：
- /proc/\<pid\>/net/tcp：TCP连接信息
- /proc/\<pid\>/net/tcp6：TCPv6连接信息
- /proc/\<pid\>/net/udp：UDP连接信息
- /proc/\<pid\>/net/udp6：UDPv6连接信息
- /proc/\<pid\>/net/raw：RAW套接字信息
- /proc/\<pid\>/net/unix：Unix域套接字信息

**示例内容**（/proc/\<pid\>/net/tcp）：
```
  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1388 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 123456 1 0000000000000000 100 0 0 10 0
   1: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 789012 1 0000000000000000 100 0 0 10 0
```

### 2. /proc/\<pid\>/fdinfo/*（套接字）

对于套接字类型的文件描述符，fdinfo文件包含额外的网络信息。

**示例内容**（/proc/\<pid\>/fdinfo/5，其中5是套接字文件描述符）：
```
pos:    0
flags:  02004002
mnt_id: 1
ino:    123456
uid:    1000
gid:    1000
socket: [12345678]
locks:
pos:	0  fl_flags:	0x0  fl_type:	0 (none)
```

## 九、其他重要文件

### 1. /proc/\<pid\>/environ

存储进程的环境变量，各变量之间以NULL字符（\0）分隔。

**示例内容**：
```
SHELL=/bin/bash\0TERM=xterm\0USER=user\0PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\0PWD=/home/user\0HOME=/home/user\0LANG=en_US.UTF-8\0
```

**读取方法**：
```bash
cat /proc/1234/environ | tr '\0' '\n'
```

### 2. /proc/\<pid\>/limits

显示进程的资源限制，包括软限制和硬限制。

**示例内容**：
```
Limit                     Soft Limit           Hard Limit           Units
Max cpu time              unlimited            unlimited            seconds
Max file size             unlimited            unlimited            bytes
Max data size             unlimited            unlimited            bytes
Max stack size            8388608              unlimited            bytes
Max core file size        0                    unlimited            bytes
Max resident set          unlimited            unlimited            bytes
Max processes             31444                31444                processes
Max open files            1024                 4096                 files
Max locked memory         65536                65536                bytes
Max address space         unlimited            unlimited            bytes
Max file locks            unlimited            unlimited            locks
Max pending signals       31444                31444                signals
Max msgqueue size         819200               819200               bytes
Max nice priority         0                    0
Max realtime priority     0                    0
Max realtime timeout      unlimited            unlimited            us
```

### 3. /proc/\<pid\>/oom_score

显示进程的OOM（Out of Memory）分数，用于当系统内存不足时决定哪个进程会被终止。分数范围为0到1000，分数越高，进程越容易被OOM killer终止。

**OOM分数的计算过程**：

OOM分数的计算基于多种因素，主要包括：

1. **基础内存分数**：
   - 首先计算进程使用的内存量（RSS + Swap + 共享内存的比例）
   - 将内存使用量与系统总内存的比例转换为0-1000的基础分数
   - 内存使用量越高，基础分数越高

2. **oom_score_adj调整**：
   - 将`oom_score_adj`的值直接加到基础分数上
   - 调整范围为-1000到1000
   - 例如，如果基础分数是300，`oom_score_adj`是-100，则最终分数为200

3. **进程特性调整**：
   - 根进程（PID=1）：自动获得-1000的调整值，除非明确修改
   - 特权进程：可能获得较低的分数
   - 正在处理实时任务的进程：可能获得较高的分数
   - 进程的CPU使用时间：CPU密集型进程可能获得较低分数
   - 进程的运行时间：长时间运行的进程可能获得较低分数

4. **最终分数限制**：
   - 最终分数被限制在0到1000之间
   - 如果`oom_score_adj`设置为-1000，最终分数始终为0，进程不会被终止
   - 如果`oom_score_adj`设置为1000，最终分数始终为1000，进程会被优先终止

**示例内容**：
```
256
```

### 4. /proc/\<pid\>/oom_score_adj

用于调整进程的OOM分数。值的范围为-1000到1000，其中：
- -1000：进程不会被OOM killer终止
- 0：默认值，OOM分数仅基于进程的内存使用情况
- 1000：进程总是会被首先终止

**示例内容**：
```
0
```

## 十、实用示例

### 1. 查看进程的命令行参数
```bash
pid=1234
cat /proc/$pid/cmdline | tr '\0' ' '
```

### 2. 查看进程的内存使用情况
```bash
pid=1234
echo "进程 $pid 的内存使用情况："
echo "虚拟内存大小：$(grep VmSize /proc/$pid/status | awk '{print $2}') KB"
echo "物理内存使用：$(grep VmRSS /proc/$pid/status | awk '{print $2}') KB"
echo "内存峰值：$(grep VmPeak /proc/$pid/status | awk '{print $2}') KB"
```

### 3. 查看进程打开的文件
```bash
pid=1234
echo "进程 $pid 打开的文件："
ls -la /proc/$pid/fd | grep -v '\[.*\]' | awk '{print $NF}'
```

### 4. 查看进程的网络连接
```bash
pid=1234
echo "进程 $pid 的TCP连接："
awk '{print "本地地址：" $2 ", 远程地址：" $3 ", 状态：" $4}' /proc/$pid/net/tcp
```

### 5. 查看进程的环境变量
```bash
pid=1234
echo "进程 $pid 的环境变量："
cat /proc/$pid/environ | tr '\0' '\n'
```

### 6. 查看进程的线程信息
```bash
pid=1234
echo "进程 $pid 的线程数：$(grep Threads /proc/$pid/status | awk '{print $2}')"
echo "线程列表："
ls -la /proc/$pid/task | grep -v '\.' | awk '{print $9}'
```

### 7. 查看进程的OOM分数
```bash
pid=1234
echo "进程 $pid 的OOM分数：$(cat /proc/$pid/oom_score)"
echo "进程 $pid 的OOM调整值：$(cat /proc/$pid/oom_score_adj)"
```

### 8. 修改进程的OOM调整值
```bash
# 将进程的OOM调整值设置为-500（降低被终止的概率）
pid=1234
echo -500 > /proc/$pid/oom_score_adj

# 查看修改后的OOM分数
cat /proc/$pid/oom_score
```

## 十一、常见问题（FAQ）

### 1. /proc/\<pid\>/目录下的文件占用磁盘空间吗？

不占用。/proc文件系统是内存文件系统，所有文件内容都由内核动态生成和管理，不占用实际磁盘空间。

### 2. 如何获取进程的命令行参数？

可以通过读取/proc/\<pid\>/cmdline文件获取，该文件包含进程的完整命令行参数，各参数之间以NULL字符分隔。可以使用`cat /proc/\<pid\>/cmdline | tr '\0' ' '`命令查看格式化后的命令行。

### 3. 如何查看进程打开的所有文件？

可以通过列出/proc/\<pid\>/fd目录下的所有符号链接来查看进程打开的文件，每个符号链接指向对应的文件或设备。可以使用`ls -la /proc/\<pid\>/fd`命令查看。

### 4. 如何获取进程的内存使用情况？

可以通过读取/proc/\<pid\>/status文件中的VmSize、VmRSS、VmPeak等字段获取进程的内存使用情况。其中VmSize表示虚拟内存大小，VmRSS表示物理内存使用大小，VmPeak表示内存使用峰值。

### 5. 如何查看和调整进程的OOM分数？

可以通过读取/proc/\<pid\>/oom_score文件查看进程的OOM分数，通过/proc/\<pid\>/oom_score_adj文件调整OOM分数：
- 查看OOM分数：`cat /proc/\<pid\>/oom_score`
- 查看OOM调整值：`cat /proc/\<pid\>/oom_score_adj`
- 调整OOM分数：`echo -500 > /proc/\<pid\>/oom_score_adj`（降低被终止的概率）

OOM分数范围为0-1000，分数越高越容易被终止。oom_score_adj的范围为-1000到1000，设置为-1000可防止进程被OOM killer终止。

## 十二、总结

/proc/\<pid\>/目录下的文件提供了丰富的进程信息，是Linux系统中调试、监控和管理进程的重要工具。通过熟练掌握这些文件的作用和使用方法，系统管理员和开发者可以更深入地了解进程的运行状态，快速定位和解决问题。

需要注意的是，/proc文件系统的接口可能会随着内核版本的变化而变化，因此在编写脚本或工具时，应考虑版本兼容性问题。同时，由于/proc文件系统提供了直接访问内核数据的接口，不当的使用可能会影响系统性能或稳定性，应谨慎使用。
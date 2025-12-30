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

# Linux 系统启动流程

## 核心概念

Linux 系统从电源启动到可用状态，经历多个阶段：**BIOS/UEFI → Bootloader → Kernel → Init 系统**。每个阶段的作用不同，理解启动流程有助于诊断启动问题、优化启动速度和深入理解系统。

---

## 启动流程详解

### 第一阶段：硬件自检和 BIOS/UEFI

```
1. 按下电源开关
2. BIOS/UEFI 启动（固件初始化）
   - 进行硬件自检（POST - Power On Self Test）
   - 检查 CPU、内存、硬盘等设备
   - 初始化硬件设备
3. BIOS/UEFI 查找启动设备（根据启动顺序）
4. 读取引导扇区（MBR 或 GPT）
```

#### BIOS 与 UEFI 的详细对比

| 特性 | BIOS | UEFI |
|------|------|------|
| **固件类型** | 16位实模式 | 32位/64位保护模式 |
| **引导方式** | 传统MBR（主引导记录） | GPT（GUID分区表） |
| **最大磁盘容量** | 2.2TB | 18EB（理论值） |
| **分区数量** | 最多4个主分区 | 最多128个主分区 |
| **安全启动** | 不支持 | 支持（Secure Boot） |
| **启动速度** | 较慢 | 较快（支持并行加载） |
| **界面** | 文本模式，有限功能 | 图形界面，丰富配置 |
| **网络支持** | 不支持 | 原生支持 |
| **驱动程序** | 依赖操作系统 | 内置驱动程序 |
| **兼容性** | 所有旧硬件 | 新硬件优先，支持旧硬件兼容模式 |

UEFI 是 BIOS 的继任者，提供更强大的功能和更好的性能，特别是在处理大容量硬盘和现代硬件方面。

### 第二阶段：Bootloader（引导加载器）

主流 Bootloader 是 **GRUB（GRand Unified Bootloader）**。

```bash
BIOS/UEFI 加载 Bootloader
     ↓
GRUB Stage 1
  - 位置：MBR 的前 446 字节或 BIOS Boot Partition
  - 作用：定位和加载 Stage 1.5 或 Stage 2
     ↓
GRUB Stage 1.5（可选）
  - 位置：MBR 之后，第一个分区之前
  - 作用：理解文件系统，加载 Stage 2
     ↓
GRUB Stage 2
  - 位置：/boot/grub/ 目录
  - 功能：
    * 显示启动菜单（可选）
    * 加载内核镜像到内存
    * 加载初始化内存磁盘（initrd/initramfs）
    * 将控制权交给 Kernel
```

**关键配置文件**：
```bash
/etc/grub.d/          # GRUB 配置脚本
/boot/grub/grub.cfg   # GRUB 最终配置文件（自动生成）

# 修改 GRUB 配置
vim /etc/default/grub
update-grub            # Debian/Ubuntu
grub2-mkconfig -o /boot/grub2/grub.cfg  # RedHat/CentOS
```

#### 常见 Bootloader 对比

| Bootloader | 特性 | 适用场景 |
|------------|------|----------|
| **GRUB** | 支持多操作系统、多文件系统、UEFI/BIOS | 主流 Linux 发行版（Ubuntu、CentOS、Fedora） |
| **LILO** | 简单可靠、不支持 UEFI | 旧版 Linux 系统、对启动速度有要求的环境 |
| **SYSLINUX** | 轻量级、支持网络启动 | 小型系统、Live CD/USB、网络启动 |
| **rEFInd** | 图形界面、自动检测系统 | EFI 系统、多操作系统引导（Linux/Windows/macOS） |
| **GRUB Legacy** | 旧版本 GRUB | 非常旧的 Linux 系统（CentOS 5 及更早） |

GRUB 是目前最主流的 Bootloader，支持所有现代硬件和引导方式，而 LILO 等传统 Bootloader 逐渐被淘汰。

### 第三阶段：Kernel 启动

```bash
GRUB 加载 Kernel
     ↓
Kernel 自解压和初始化
  - 设置 CPU 为保护模式
  - 初始化内存分配
  - 解析启动参数（来自 GRUB）
  - 加载驱动程序（来自 initramfs）
     ↓
Kernel 挂载根文件系统
  - 挂载 initramfs（临时根文件系统）
  - 初始化根文件系统
  - 加载必要的驱动（如磁盘驱动）
     ↓
Kernel 调用第一个用户程序
  - Systemd（现代 Linux）
  - Upstart（Ubuntu 老版本）
  - SysVinit（老版本）
```

**查看 Kernel 启动参数**：
```bash
cat /proc/cmdline
# 示例输出：
# BOOT_IMAGE=/boot/vmlinuz-5.15.0 root=UUID=xxxxx ro quiet splash

# 关键参数含义：
# root=      : 根文件系统的位置
# ro         : 以只读方式挂载根文件系统
# quiet      : 不输出启动消息
# init=      : 指定第一个进程（默认 /sbin/init）
```

#### initramfs 深入解析

**什么是 initramfs？**
initramfs（initial RAM file system）是一个临时的根文件系统，在 Kernel 启动时加载到内存中，包含启动所需的核心驱动程序和工具。

**initramfs 的作用**：
```
1. 提供根文件系统所在设备的驱动程序（如 SCSI、RAID、NVMe）
2. 支持复杂的根文件系统配置（如加密、LVM、RAID）
3. 执行根文件系统挂载前的初始化任务
4. 提供调试环境（当真实根文件系统无法挂载时）
```

**查看 initramfs 内容**：
```bash
# Debian/Ubuntu
lsinitramfs /boot/initrd.img-$(uname -r) | head -20

# RedHat/CentOS
dracut -l /boot/initramfs-$(uname -r).img

# 通用方法
mkdir /tmp/initramfs
gunzip -c /boot/initrd.img-$(uname -r) | cpio -idv -D /tmp/initramfs
ls -la /tmp/initramfs
```

**重建 initramfs**：
```bash
# Debian/Ubuntu
update-initramfs -u              # 更新当前内核的 initramfs
update-initramfs -c -k all       # 为所有内核创建新的 initramfs
update-initramfs -c -k $(uname -r)  # 为特定内核创建

# RedHat/CentOS 7+
dracut --force --hostonly /boot/initramfs-$(uname -r).img $(uname -r)

# 通用方法
mkinitramfs -o /boot/initrd.img-$(uname -r) $(uname -r)
```

**常见问题与解决方案**：
- **initramfs 损坏**：使用 Live CD 启动，重新挂载根文件系统，执行重建命令
- **缺少驱动**：在 /etc/initramfs-tools/modules 中添加缺失的驱动名称，然后重建
- **加密根文件系统**：确保 cryptsetup 工具包含在 initramfs 中

### 第四阶段：Init 系统（Systemd）

现代 Linux（Debian 8+、CentOS 7+ 等）使用 **Systemd**。

```bash
Systemd 启动（PID=1）
     ↓
读取目标（Target）配置
  - /etc/systemd/system/default.target（默认启动目标）
  - 通常链接到 multi-user.target 或 graphical.target
     ↓
解析依赖关系
  - 从目标配置中读取所有需要启动的 Service
  - 确定启动顺序（After=, Before= 字段）
     ↓
并行启动 Services
  - 执行所有 Before 的 Services
  - 同时启动无依赖关系的 Services（并行加速）
  - 等待依赖 Services 完成
     ↓
启动完成
  - 进入目标状态（如 multi-user.target）
  - 显示登录提示
```

**重要目录和文件**：
```bash
# 系统启动目标
ls -la /etc/systemd/system/default.target

# 查看启动目标
systemctl get-default
# 输出：multi-user.target

# 启动流程中的 Units 位置
/usr/lib/systemd/system/       # 官方 Units
/etc/systemd/system/            # 用户自定义 Units
/run/systemd/system/            # 临时 Units

# 查看启动耗时
systemd-analyze                # 总耗时
systemd-analyze blame          # 各 Unit 耗时排名
systemd-analyze plot           # 生成启动图表
```

#### Systemd Service 管理高级示例

**基本 Service 操作**：
```bash
# 启动/停止/重启 Service
systemctl start docker.service
systemctl stop docker.service
systemctl restart docker.service

# 查看 Service 状态
systemctl status docker.service -l  # 详细输出
systemctl is-active docker.service  # 仅显示激活状态
systemctl is-enabled docker.service # 仅显示是否开机自启

# 设置开机自启/禁用
systemctl enable docker.service
systemctl disable docker.service
systemctl enable --now docker.service  # 立即启动并设置自启
```

**Service 依赖管理**：
```bash
# 查看 Service 的依赖关系
systemctl list-dependencies docker.service
systemctl list-dependencies docker.service --reverse  # 反向依赖

# 查看 Service 的 Unit 文件
systemctl cat docker.service

# 编辑 Service 的 Unit 文件
systemctl edit docker.service  # 创建覆盖文件
systemctl edit --full docker.service  # 编辑完整文件
```

**Target 管理**：
```bash
# 查看所有可用 Target
systemctl list-units --type=target

# 切换 Target（立即生效，不修改默认）
systemctl isolate multi-user.target   # 切换到命令行模式
systemctl isolate graphical.target    # 切换到图形界面
systemctl isolate rescue.target       # 进入救援模式
systemctl isolate emergency.target    # 进入紧急模式

# 临时禁用 graphical.target
systemctl mask graphical.target
systemctl unmask graphical.target  # 恢复

# 查看 Target 依赖关系
systemctl list-dependencies multi-user.target
```

**Systemd 日志管理**：
```bash
# 查看 Service 的日志
journalctl -u docker.service
journalctl -u docker.service -n 50  # 最近50行
journalctl -u docker.service -f     # 实时跟踪
journalctl -u docker.service --since "2024-01-01" --until "2024-01-02"

# 按优先级过滤日志
journalctl -p err                  # 错误日志
journalctl -p err -b               # 当前启动的错误日志

# 清理旧日志
journalctl --vacuum-size=100M      # 保留最近100MB
journalctl --vacuum-time=1w        # 保留最近1周
```

Systemd 提供了强大的 Service 管理功能，通过这些命令可以精确控制系统服务的启动、运行和停止。

---

## 启动过程时间线

```
电源启动
  ↓ (几秒钟，硬件自检)
BIOS/UEFI → Bootloader
  ↓ (< 1 秒，通常被 splash 屏幕隐藏)
GRUB 显示菜单 → 加载 Kernel
  ↓ (0.5-2 秒，Kernel 初始化)
Kernel 初始化 → 调用 Systemd
  ↓ (1-5 秒，依赖于 Units 数量)
Systemd 启动 Units → 系统就绪
  ↓ (total: 10-30 秒，取决于硬件和配置)
用户登录界面出现
```

---

## 查看和分析启动信息

### 1. 查看引导日志
```bash
# Systemd 日志（完整的启动过程）
journalctl -b                 # 当前启动的所有日志
journalctl -b -0              # 当前启动
journalctl -b -1              # 上一次启动
journalctl -xe                # 最后 50 行日志加上错误标记

# Kernel 日志（启动阶段）
dmesg                         # Kernel 日志缓冲
dmesg | head -50              # 启动时的 Kernel 消息
```

### 2. 分析启动性能
```bash
# 查看总启动耗时
systemd-analyze
# 输出示例：
# Startup finished in 2.105s (kernel) + 3.467s (userspace) = 5.572s

# 找出耗时最长的 Units
systemd-analyze blame
# 输出示例：
# 1.563s docker.service
# 0.824s networking.service
# 0.512s snapd.service

# 关键的链式依赖
systemd-analyze critical-chain
```

### 3. 故障诊断
```bash
# 查看启动失败的 Units
systemctl --failed

# 查看特定 Service 的状态
systemctl status docker.service

# 查看 Service 的启动输出
journalctl -u docker.service -n 50

# 以调试模式启动（显示更多信息）
# 在 GRUB 启动参数中添加 systemd.log_level=debug
```

---

## 启动阶段的常见问题

### 问题 1：系统无法启动（卡在 GRUB）
```bash
# 症状：卡在 GRUB 界面或 GRUB 报错

# 排查步骤：
1. 检查 GRUB 配置
   cat /boot/grub/grub.cfg | grep menuentry

2. 修复 GRUB（如果 MBR 损坏）
   grub-install /dev/sda
   grub-mkconfig -o /boot/grub/grub.cfg

3. 尝试手动启动
   # 在 GRUB 命令行输入：
   grub> ls
   grub> insmod linux
   grub> linux /vmlinuz-5.15 root=/dev/sda1
   grub> boot
```

### 问题 2：系统启动缓慢
```bash
# 排查步骤：
1. 分析启动耗时
   systemd-analyze blame

2. 禁用不需要的 Services
   systemctl disable service_name

3. 并行启动优化
   - 大多数 Services 已默认并行启动
   - 可在 Unit 文件中调整依赖关系

4. 检查是否卡在某个 Unit
   systemctl status -l  # 显示正在运行的 Unit
```

### 问题 3：根文件系统无法挂载
```bash
# 症状：Kernel 启动后卡住，显示 "Unable to mount root fs"

# 原因：
- 根设备不存在或路径错误
- 驱动程序未加载（initramfs 缺少驱动）
- 文件系统损坏

# 排查：
1. 检查 GRUB 中的 root 参数
   cat /proc/cmdline

2. 验证根设备是否存在
   ls -la /dev/sda1
   blkid  # 查看所有分区和 UUID

3. 重建 initramfs（包含驱动程序）
   mkinitramfs -o /boot/initrd.img-5.15 5.15
   # 或
   mkinitrd /boot/initrd.img-5.15 5.15
```

---

## 启动流程的文件清单

```
BIOS/UEFI 阶段
└─ 固件代码（不涉及文件系统）

Bootloader 阶段
├─ /boot/grub/            # GRUB 配置和模块
├─ /boot/vmlinuz-*        # Kernel 镜像（压缩的）
├─ /boot/initrd.img-*     # 初始化内存磁盘
└─ /boot/grub/grub.cfg    # GRUB 配置文件

Kernel 阶段
└─ 内存中的 Kernel 代码

Init 系统阶段（Systemd）
├─ /lib/systemd/system/           # 官方 Units
│  ├─ multi-user.target
│  ├─ graphical.target
│  ├─ networking.service
│  └─ ...
├─ /etc/systemd/system/           # 用户 Units
├─ /etc/systemd/system/default.target  # 启动目标
└─ /etc/fstab              # 文件系统挂载表
```

---

## 快速参考

```bash
# 查看当前运行级别（Target）
systemctl get-default
runlevel  # 兼容命令

# 改变启动目标
systemctl set-default multi-user.target
systemctl set-default graphical.target

# 查看启动耗时
systemd-analyze time
systemd-analyze blame | head -10

# 诊断启动问题
journalctl -b -p err  # 启动时的错误
systemctl --failed    # 启动失败的 Units

# 重建 Bootloader
grub-install /dev/sda
grub-mkconfig -o /boot/grub/grub.cfg

# 重建 Initramfs
update-initramfs -u  # Debian/Ubuntu
mkinitramfs -o /boot/initrd.img-$(uname -r) $(uname -r)  # 通用
```

---

### 相关高频面试题

#### Q1: GRUB 和 initramfs 的作用分别是什么？

**答案**：

```bash
# GRUB（GRand Unified Bootloader）：
# - 作用：从 BIOS/UEFI 和硬盘之间搭建桥梁
# - 功能：
#   1. 加载 Kernel 镜像到内存
#   2. 加载 initramfs 到内存
#   3. 解析启动参数（root=, ro, quiet 等）
#   4. 将控制权转交给 Kernel

# initramfs（初始 RAM 文件系统）：
# - 作用：提供最小的临时根文件系统
# - 功能：
#   1. 包含启动所需的驱动程序（磁盘、文件系统等）
#   2. 挂载真实根文件系统前的初始化
#   3. 运行初始化脚本，扫描硬件并加载驱动
#   4. 最后切换到真实的根文件系统

# 查看 initramfs 内容：
file /boot/initrd.img-*        # 检查格式（gzip、cpio 等）
gunzip -c initrd.img | cpio -tv  # 列出内容
```

#### Q2: Systemd 如何实现快速启动？

**答案**：

```bash
# 3 个核心机制：

# 1. 并行启动（Parallelization）
#    - 不依赖的 Units 同时启动
#    - 大大加快总启动时间

# 2. 按需激活（On-demand Activation）
#    - Services 不立即启动，而是按需启动
#    - 减少启动阶段加载的 Services 数量

# 3. 即插即用（Plug and Play）
#    - 使用 systemd 的 Unit 依赖关系
#    - After=, Before=, Requires=, Wants= 等字段精确控制顺序

# 优化启动的方法：
systemd-analyze blame           # 找出耗时 Units
systemctl disable service       # 禁用不需要的 Services

# 查看并行启动情况：
systemd-analyze critical-chain  # 关键路径链
systemd-analyze plot > boot.svg # 生成启动关系图
```

#### Q3: 如果 Kernel 在启动时找不到根文件系统怎么办？

**答案**：

```bash
# 症状：
# VFS: Unable to mount root fs on unknown-block(0,0)

# 原因分析：

# 1. root 参数错误
#    - GRUB 中的 root=UUID=xxx 或 root=/dev/sda1 不存在
#    排查：cat /proc/cmdline 检查 root 参数

# 2. 驱动程序缺失
#    - Kernel 没有根文件系统所在磁盘的驱动
#    - initramfs 中缺少必要驱动
#    排查：lspci, lsblk 检查硬件

# 3. 文件系统损坏
#    - 磁盘坏道或 inode 损坏
#    排查：fsck -n /dev/sda1（检查不修复）

# 解决方案：

# 方案 1：修复 root 参数
#   进入 GRUB 编辑界面（按 e），修改 root= 参数
#   使用 blkid 确认正确的 UUID 或设备路径

# 方案 2：重建 initramfs（包含驱动）
#   mkinitramfs -o /boot/initrd.img-$(uname -r) $(uname -r)

# 方案 3：使用 LiveCD/USB 修复
#   从 LiveCD 启动，挂载根文件系统，检查和修复
```

#### Q4: Systemd Target 和老版本的 Runlevel 对应关系是什么？

**答案**：

```bash
# 对应关系：
# Runlevel 0 ↔ poweroff.target       # 关机
# Runlevel 1 ↔ rescue.target         # 单用户模式
# Runlevel 2 ↔ multi-user.target     # 多用户无图形
# Runlevel 3 ↔ multi-user.target     # 多用户无图形
# Runlevel 4 ↔ multi-user.target     # 多用户无图形
# Runlevel 5 ↔ graphical.target      # 多用户带图形
# Runlevel 6 ↔ reboot.target         # 重启

# Systemd 查看和切换：
systemctl get-default                # 查看默认 Target
systemctl list-units --type=target    # 列出所有 Targets
systemctl set-default graphical.target

# 临时切换 Target（不修改默认）：
systemctl isolate multi-user.target   # 切换到多用户模式
systemctl isolate graphical.target    # 切换到图形界面
systemctl isolate rescue.target       # 进入救援模式
```

#### Q5: 如何调试和追踪系统启动过程？

**答案**：

```bash
# 1. 启用详细日志输出
#    在 GRUB 中修改启动参数：
#    - 移除 quiet splash
#    - 添加 systemd.log_level=debug
#    - 添加 log_buf_len=1M（增加日志缓冲）

# 2. 查看启动日志
journalctl -b            # 当前启动的所有日志
journalctl -b -p err     # 只显示错误
journalctl -b -f         # 实时跟踪（仅 systemd 阶段后）

# 3. 查看 Kernel 启动信息
dmesg | head -100        # 启动时的 Kernel 消息
dmesg | grep -i error    # 搜索 Kernel 错误

# 4. 追踪特定 Service
systemctl status docker.service
journalctl -u docker.service -n 100

# 5. 性能分析
systemd-analyze          # 总耗时
systemd-analyze blame    # 各 Unit 耗时
systemd-analyze plot > boot.svg  # 可视化

# 6. 进入启动调试（需在 GRUB 中添加 systemd.unit=rescue.target）
#    系统会在启动失败时进入 rescue shell
```

#### Q6: 如何优化 Linux 系统的启动速度？

**答案**：

```bash
# 诊断：
systemd-analyze blame | head -10   # 找出耗时最长的 Services

# 优化策略：

# 1. 禁用不需要的 Services
for service in snapd.service cups.service blueetooth.service; do
  systemctl disable $service
done

# 2. 启用并行启动（通常已默认）
#    在 /etc/systemd/system.conf 中检查：
#    DefaultEnvironment="SYSTEMD_SHOW_STATUS=1"

# 3. 调整启动超时时间
#    减少等待失败 Services 的时间
#    在相应 Service 文件中设置：
#    TimeoutStartSec=5

# 4. 使用 systemd.unit=multi-user.target
#    跳过图形界面启动（如不需要）

# 5. 检查和修复文件系统
#    坏块或损坏的文件系统会大幅拖累启动
#    fsck -f /dev/sda1

# 6. 使用 SSD 而非 HDD
#    磁盘读写速度直接影响启动时间

# 效果验证：
systemd-analyze time  # 修改前后对比
```

#### Q7: UEFI 安全启动（Secure Boot）的作用是什么？如何禁用它？

**答案**：

```bash
# Secure Boot 的作用：
# 1. 防止未签名的恶意 Bootloader 或 Kernel 被加载
# 2. 确保只有经过验证的固件和操作系统组件才能启动
# 3. 保护系统免受 Rootkit 和引导级恶意软件的攻击

# 禁用 Secure Boot 的步骤：

# 1. 进入 UEFI 设置界面（通常在启动时按 F2/F10/F12/DEL）
# 2. 找到 "Secure Boot" 或 "Boot Security" 选项
# 3. 将其从 "Enabled" 改为 "Disabled"
# 4. 保存设置并重启系统

# 验证 Secure Boot 状态：
mokutil --sb-state
# 输出：SecureBoot disabled
```

#### Q8: Systemd 中的 Unit 文件包含哪些主要部分？

**答案**：

```bash
# Systemd Unit 文件的主要部分：

# 1. [Unit] 部分
#    - 描述：Unit 文件的元数据和依赖关系
#    - 关键字：Description, After, Before, Requires, Wants, Conflicts

# 2. [Service] 部分（仅 Service Unit）
#    - 描述：服务的运行方式和命令
#    - 关键字：Type, ExecStart, ExecStop, ExecReload, Restart, User, Group

# 3. [Socket] 部分（仅 Socket Unit）
#    - 描述：套接字的配置
#    - 关键字：ListenStream, ListenDatagram, SocketMode

# 4. [Target] 部分（仅 Target Unit）
#    - 描述：启动目标的配置
#    - 关键字：Wants, Requires, After, Before

# 5. [Install] 部分
#    - 描述：Unit 的安装信息
#    - 关键字：WantedBy, RequiredBy, Alias

# 查看 Unit 文件示例：
systemctl cat ssh.service
```

#### Q9: 什么是 initramfs 和 initrd？它们的区别是什么？

**答案**：

```bash
# initramfs（initial RAM file system）：
# - 是一个临时的根文件系统，在 Kernel 启动时加载到内存中
# - 使用 cpio 格式打包，无需解压缩到内存
# - 由内核直接挂载，无需额外的文件系统驱动
# - 现代 Linux 系统默认使用

# initrd（initial RAM disk）：
# - 是一个临时的 RAM 磁盘，在 Kernel 启动时加载到内存中
# - 使用 ext2 等文件系统格式，需要解压缩到内存
# - 需要内核支持相应的文件系统驱动
# - 较旧的 Linux 系统使用

# 区别总结：
# 1. 格式：initramfs 是 cpio 归档，initrd 是文件系统镜像
# 2. 挂载：initramfs 直接由内核挂载，initrd 需要文件系统驱动
# 3. 效率：initramfs 更高效，无需额外的文件系统层
# 4. 灵活性：initramfs 支持更复杂的启动配置

# 查看当前系统使用的是哪种：
ls -la /boot/ | grep -E "initrd|initramfs"
# Debian/Ubuntu：initrd.img-* （实际是 initramfs）
# RedHat/CentOS：initramfs-*.img
```

#### Q10: 如何在 GRUB 中添加一个新的启动项？

**答案**：

```bash
# 方法 1：直接编辑 GRUB 配置文件（不推荐，会被 update-grub 覆盖）
# vim /boot/grub/grub.cfg

# 方法 2：通过 /etc/grub.d/ 脚本添加（推荐）

# 1. 创建自定义脚本
cat > /etc/grub.d/40_custom << 'EOF'
#!/bin/sh
cat << 'EOL'
menuentry "My Custom Linux" {
    insmod ext2
    set root=(hd0,1)
    linux /boot/vmlinuz-custom root=/dev/sda1 ro quiet splash
    initrd /boot/initrd.img-custom
}
EOL
EOF

# 2. 赋予脚本执行权限
chmod +x /etc/grub.d/40_custom

# 3. 更新 GRUB 配置
update-grub            # Debian/Ubuntu
grub2-mkconfig -o /boot/grub2/grub.cfg  # RedHat/CentOS

# 方法 3：使用 GRUB 自定义菜单项
# 在 /etc/default/grub 中添加：
# GRUB_CUSTOM_MENU_ITEMS="menuentry 'My Custom' {...}"
```

#### Q11: Systemd 如何管理服务的依赖关系？

**答案**：

```bash
# Systemd 使用以下关键字管理服务依赖：

# 1. 顺序依赖
#    - After=：当前服务在指定服务之后启动
#    - Before=：当前服务在指定服务之前启动

# 2. 硬依赖
#    - Requires=：当前服务依赖的其他服务，若依赖服务失败，当前服务也会失败
#    - Requisite=：比 Requires 更严格，依赖服务不存在时当前服务立即失败

# 3. 软依赖
#    - Wants=：当前服务希望依赖的其他服务，依赖服务失败不影响当前服务
#    - BindsTo=：与 Requires 类似，但依赖服务停止时当前服务也会停止

# 4. 触发依赖
#    - PartOf=：当前服务是指定服务的一部分，指定服务停止时当前服务也会停止
#    - RequiredBy=, WantedBy=：用于 [Install] 部分，指定当前服务被哪些服务依赖

# 示例：
cat > /etc/systemd/system/myapp.service << 'EOF'
[Unit]
Description=My Application
After=network.target mysql.service
Requires=mysql.service
Wants=redis.service

[Service]
Type=simple
ExecStart=/usr/bin/myapp

[Install]
WantedBy=multi-user.target
EOF

# 重新加载配置并启动
systemctl daemon-reload
systemctl start myapp.service
```

---

### 启动故障速查表

| 症状               | 可能原因                | 排查方法                          |
| ------------------ | ----------------------- | --------------------------------- |
| GRUB 不显示        | BIOS 设置、MBR 损坏     | 检查 BIOS 启动顺序、重装 GRUB     |
| 卡在 GRUB          | 配置文件错误            | 进入 GRUB 编辑，手动指定 root     |
| Kernel 无法加载    | initramfs 损坏          | 重建 initramfs                    |
| 根文件系统无法挂载 | root 参数错误、驱动缺失 | 检查 cmdline、重建 initramfs      |
| 系统启动缓慢       | 不必要的 Services       | systemd-analyze blame，禁用服务   |
| Units 启动失败     | 依赖缺失、配置错误      | journalctl -xe，检查 Service 文件 |
| 系统卡住（黑屏）   | 某个 Unit 阻塞          | 按 Ctrl+Alt+F2 进入 tty，查看日志 |


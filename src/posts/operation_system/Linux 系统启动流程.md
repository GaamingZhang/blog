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

# Linux 系统启动:从上电到登录的完整旅程

## Linux 启动是什么

Linux 系统启动是从按下电源开关到显示登录界面的完整过程,就像接力赛一样,每个阶段完成特定任务后将控制权交给下一个阶段。理解启动流程有助于:
- 诊断启动失败问题
- 优化系统启动速度
- 深入理解操作系统架构

---

## 启动流程概览

### 四阶段模型

Linux 启动分为四个主要阶段:

```
按下电源
   ↓
┌─────────────────────┐
│  1. BIOS/UEFI       │  固件初始化
│  (硬件自检)         │  找到启动设备
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│  2. Bootloader      │  加载内核
│  (GRUB)             │  加载initramfs
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│  3. Kernel          │  初始化硬件
│  (Linux内核)        │  挂载根文件系统
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│  4. Init系统        │  启动服务
│  (systemd)          │  进入用户态
└──────────┬──────────┘
           ↓
       登录界面
```

**时间分配**(典型情况):
- BIOS/UEFI: 2-5秒 (硬件自检)
- Bootloader: < 1秒 (通常被splash屏幕隐藏)
- Kernel: 1-3秒 (内核初始化)
- Init系统: 3-10秒 (启动服务)
- **总计**: 10-20秒

---

## 第一阶段:BIOS/UEFI

### 固件的作用

**BIOS/UEFI**是主板上的固件程序,负责:

```
上电
  ↓
加载固件代码
  ↓
POST(硬件自检)
├─ 检测CPU
├─ 检测内存
├─ 检测硬盘
└─ 检测其他设备
  ↓
按启动顺序查找启动设备
  ↓
读取启动设备的引导扇区
  ↓
将控制权交给Bootloader
```

### BIOS vs UEFI

| 特性       | BIOS(传统)       | UEFI(现代)         |
| ---------- | ---------------- | ------------------ |
| 运行模式   | 16位实模式       | 32/64位保护模式    |
| 分区表     | MBR              | GPT                |
| 最大磁盘   | 2.2TB            | 18EB               |
| 最大分区数 | 4个主分区        | 128个主分区        |
| 启动速度   | 慢               | 快(支持并行)       |
| 安全启动   | 不支持           | 支持Secure Boot    |
| 界面       | 文本             | 图形               |

**为什么UEFI更好**:
- 支持大容量硬盘(>2TB)
- 启动速度更快
- 更安全(Secure Boot防止恶意启动)
- 支持网络启动

---

## 第二阶段:Bootloader (GRUB)

### Bootloader的作用

**Bootloader**(引导加载器)是BIOS和操作系统之间的桥梁,负责:

```
BIOS/UEFI找到启动设备
  ↓
加载GRUB Stage 1
├─ 位置: MBR的前446字节
└─ 作用: 定位Stage 2
  ↓
加载GRUB Stage 2
├─ 位置: /boot/grub/目录
└─ 作用: 显示菜单,加载内核
  ↓
GRUB工作
├─ 显示启动菜单(可选)
├─ 读取配置(/boot/grub/grub.cfg)
├─ 加载内核镜像到内存
├─ 加载initramfs到内存
└─ 设置启动参数
  ↓
将控制权交给Kernel
```

### GRUB的关键文件

```
/boot/grub/grub.cfg       # GRUB配置文件(自动生成)
/boot/vmlinuz-*           # Linux内核镜像
/boot/initrd.img-*        # 初始化内存磁盘
/etc/default/grub         # GRUB默认配置
```

### 启动参数的意义

GRUB将启动参数传递给内核:

```bash
# 典型的启动参数
linux /vmlinuz root=UUID=xxx ro quiet splash

# 参数解释:
# root=UUID=xxx  : 根文件系统位置
# ro             : 以只读方式挂载(稍后改为读写)
# quiet          : 不显示详细启动信息
# splash         : 显示启动画面
```

查看当前启动参数:
```bash
cat /proc/cmdline
```

---

## 第三阶段:Linux Kernel

### Kernel的启动过程

```
GRUB加载内核到内存
  ↓
Kernel自解压
  ↓
初始化CPU
├─ 设置保护模式
├─ 启用分页
└─ 初始化中断
  ↓
初始化内存管理
  ↓
挂载initramfs
├─ 临时根文件系统
├─ 包含必要驱动
└─ 运行初始化脚本
  ↓
切换到真实根文件系统
  ↓
启动第一个进程(PID=1)
└─ 通常是systemd
```

### initramfs的作用

**为什么需要initramfs?**

问题:Kernel需要驱动才能访问硬盘,但驱动通常在硬盘上。

解决:initramfs提供临时根文件系统,包含:
- 硬盘驱动程序(SCSI, NVMe, SATA等)
- 文件系统驱动(ext4, xfs等)
- 初始化脚本

```
Kernel启动
  ↓
加载initramfs到内存
  ↓
挂载initramfs为临时根文件系统
  ↓
从initramfs加载硬盘驱动
  ↓
现在可以访问真实硬盘了!
  ↓
挂载真实根文件系统
  ↓
切换到真实根文件系统
  ↓
释放initramfs内存
```

**常见应用场景**:
- LVM逻辑卷
- 软RAID阵列
- 加密文件系统
- 网络文件系统

---

## 第四阶段:Init系统 (systemd)

### systemd的作用

**systemd**是现代Linux的初始化系统(CentOS 7+, Ubuntu 15.04+, Debian 8+):

```
Kernel启动systemd(PID=1)
  ↓
读取启动目标
├─ 默认目标:/etc/systemd/system/default.target
├─ 多用户模式:multi-user.target
└─ 图形界面:graphical.target
  ↓
解析依赖关系
├─ 哪些服务需要先启动(Before/After)
├─ 哪些服务必须启动(Requires)
└─ 哪些服务可选(Wants)
  ↓
并行启动服务
├─ 网络服务
├─ 日志服务
├─ 登录服务
└─ 其他服务
  ↓
系统就绪
└─ 显示登录界面
```

### systemd的核心优势

**传统Init(SysVinit)**:
- 串行启动(一个接一个)
- 启动慢
- Shell脚本管理

**systemd**:
- 并行启动(同时启动多个)
- 启动快
- 统一的配置格式
- 按需启动服务

### Target vs Runlevel

**老概念**(SysVinit):Runlevel 0-6

**新概念**(systemd):Target

| 老Runlevel | 新Target             | 说明           |
| ---------- | -------------------- | -------------- |
| 0          | poweroff.target      | 关机           |
| 1          | rescue.target        | 单用户模式     |
| 3          | multi-user.target    | 多用户无图形   |
| 5          | graphical.target     | 多用户带图形   |
| 6          | reboot.target        | 重启           |

查看当前目标:
```bash
systemctl get-default
```

---

## 启动流程的关键文件

### 文件位置总览

```
BIOS/UEFI阶段
└─ 固件代码(ROM芯片)

Bootloader阶段
├─ /boot/grub/grub.cfg        # GRUB配置
├─ /boot/vmlinuz-*             # 内核镜像
└─ /boot/initrd.img-*          # initramfs

Kernel阶段
├─ /proc/cmdline               # 启动参数
└─ 内存中的内核代码

Systemd阶段
├─ /etc/systemd/system/default.target   # 启动目标
├─ /lib/systemd/system/                 # 系统服务
└─ /etc/systemd/system/                 # 用户自定义服务
```

---

## 诊断启动问题

### 问题1:卡在GRUB

**症状**:GRUB界面无法加载内核

**原因**:
- 配置文件损坏
- 内核文件丢失
- 启动参数错误

**解决**:
1. 在GRUB按`e`编辑启动项
2. 检查`linux`行的root参数
3. 手动指定内核和initramfs

### 问题2:Kernel Panic

**症状**:内核无法挂载根文件系统

**原因**:
- root参数指向不存在的设备
- initramfs缺少驱动
- 文件系统损坏

**解决**:
1. 检查`/proc/cmdline`中的root参数
2. 使用`blkid`查看实际设备UUID
3. 重建initramfs

### 问题3:服务启动失败

**症状**:启动过程卡住或进入emergency模式

**原因**:
- 某个服务配置错误
- 依赖服务失败
- 文件系统挂载失败

**解决**:
```bash
# 查看失败的服务
systemctl --failed

# 查看服务日志
journalctl -xe
```

---

## 优化启动速度

### 分析启动时间

```bash
# 查看总启动时间
systemd-analyze

# 输出示例:
# Startup finished in 2.1s (kernel) + 3.5s (userspace) = 5.6s

# 查看各服务耗时
systemd-analyze blame | head -10
```

### 优化策略

**1. 禁用不需要的服务**:
```bash
systemctl disable bluetooth.service
systemctl disable cups.service
```

**2. 延迟启动非关键服务**:
- 某些服务可以在系统启动后再启动

**3. 使用SSD**:
- 硬盘速度直接影响启动时间

**4. 减少启动检查**:
- 禁用文件系统检查(不推荐)

---

## 启动模式的选择

### 命令行模式 vs 图形界面

**设置默认启动模式**:

```bash
# 设置为命令行模式(节省资源)
systemctl set-default multi-user.target

# 设置为图形界面
systemctl set-default graphical.target
```

**临时切换模式**:

```bash
# 临时进入命令行模式
systemctl isolate multi-user.target

# 临时进入图形界面
systemctl isolate graphical.target
```

### 单用户模式(救援模式)

**用途**:
- 修复系统问题
- 重置root密码
- 修复文件系统

**进入方法**:
1. 在GRUB按`e`编辑启动项
2. 在`linux`行末尾添加`systemd.unit=rescue.target`
3. 按Ctrl+X启动

---

## 启动过程的可见性

### 查看启动日志

**查看当前启动的日志**:
```bash
journalctl -b
```

**查看上次启动的日志**:
```bash
journalctl -b -1
```

**查看启动错误**:
```bash
journalctl -b -p err
```

**查看内核消息**:
```bash
dmesg | head -100
```

### 启用详细输出

**问题**:启动时只看到splash画面,看不到详细信息

**解决**:
1. 编辑`/etc/default/grub`
2. 找到`GRUB_CMDLINE_LINUX_DEFAULT`
3. 移除`quiet splash`参数
4. 运行`update-grub`

这样启动时会显示详细的启动消息。

---

## 常见问题

### 什么是GRUB?

GRUB是引导加载器,负责在BIOS和操作系统之间搭桥:
- 显示启动菜单
- 加载内核到内存
- 传递启动参数给内核

### initramfs有什么用?

initramfs是临时根文件系统,提供:
- 硬盘驱动程序(让内核能访问硬盘)
- 初始化脚本(挂载真实根文件系统)
- 支持复杂配置(LVM, RAID, 加密)

### 为什么systemd启动快?

systemd的三个优势:
1. **并行启动**:同时启动多个服务
2. **按需激活**:不立即启动所有服务
3. **依赖管理**:精确控制启动顺序

### 如何修复启动失败?

步骤:
1. 查看失败的服务:`systemctl --failed`
2. 查看日志:`journalctl -xe`
3. 修复配置或禁用服务
4. 重启验证

### 如何查看启动耗时?

```bash
systemd-analyze          # 总耗时
systemd-analyze blame    # 各服务耗时
```

---

## 核心要点

**Linux启动的本质**:四个阶段的接力赛,每个阶段完成特定任务后交棒。

**四个阶段**:
- **BIOS/UEFI**:硬件自检,找到启动设备
- **Bootloader(GRUB)**:加载内核和initramfs
- **Kernel**:初始化硬件,挂载根文件系统
- **Init(systemd)**:启动系统服务,进入用户态

**关键概念**:
- **initramfs**:临时根文件系统,提供启动所需驱动
- **启动参数**:GRUB传递给内核的配置
- **systemd Target**:启动目标,类似旧的Runlevel
- **并行启动**:systemd同时启动多个服务加快速度

**重要文件**:
- `/boot/grub/grub.cfg`:GRUB配置
- `/boot/vmlinuz-*`:内核镜像
- `/boot/initrd.img-*`:initramfs
- `/proc/cmdline`:启动参数

**常见问题**:
- 卡在GRUB → 检查配置和内核文件
- Kernel Panic → 检查root参数和initramfs
- 服务失败 → 使用systemctl --failed诊断

**优化启动**:
- 禁用不需要的服务
- 使用systemd-analyze分析耗时
- 考虑使用SSD
- 设置合适的启动目标

**调试方法**:
- 查看启动日志:`journalctl -b`
- 查看内核消息:`dmesg`
- 移除quiet参数看详细输出
- 进入救援模式修复问题

**最佳实践**:
- 定期备份GRUB配置
- 保留旧内核以防新内核失败
- 了解系统启动流程便于快速诊断
- 使用systemd-analyze优化启动速度

## 参考资源

- [Linux 内核启动文档](https://www.kernel.org/doc/html/latest/admin-guide/kernel-parameters.html)
- [systemd 官方文档](https://systemd.io/)
- [GRUB 手册](https://www.gnu.org/software/grub/manual/grub/grub.html)

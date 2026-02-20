---
date: 2026-02-09
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# 单体服务迁移至Kubernetes（一）：迁移评估与容器化

## 为什么要迁移到Kubernetes

凌晨三点被电话惊醒，线上服务宕机了——手动重启、盯监控、确认恢复。第二天产品经理说下周大促要扩容五倍——手动申请机器、配置环境、部署应用。这些场景对传统部署模式下的开发者再熟悉不过了。而Kubernetes要解决的，正是这些重复性运维痛点。

### 传统部署模式的痛点

在物理机或虚拟机上直接部署应用，看起来简单直接，但随着应用规模增长和团队成熟度提升,会逐渐暴露出几个核心问题：

**扩缩容靠手动操作**。流量高峰来临时需要人工申请资源、配置环境、部署应用，流量下降后资源又忘记回收。没有自动化的扩缩容机制，运维成本高昂且响应缓慢。

**环境一致性难以保证**。开发、测试、生产环境的操作系统版本、依赖库版本、配置参数往往存在差异。"在我机器上能跑"背后，是无数次因环境不一致导致的线上故障。

**发布缺乏标准化流程**。不同应用有不同的发布脚本，没有统一的回滚机制。蓝绿部署、金丝雀发布等高级部署策略在传统模式下实现成本高昂。

**资源利用率低下**。按峰值流量预留资源导致大部分时间闲置，多个应用共享机器又可能相互影响，很多团队选择一机一应用，进一步降低利用率。

**故障恢复依赖人工介入**。进程崩溃需要人工重启，机器宕机需要人工迁移。没有自动的健康检查和自愈机制，需要7×24小时值班。

### Kubernetes解决的核心问题

Kubernetes的设计哲学是"声明式管理"加"自动化运维"。你不需要告诉Kubernetes如何一步步操作，只需要声明你期望的状态，Kubernetes会自动把当前状态调整到期望状态，并持续维持这个状态。

**声明式而非命令式**。传统部署时，你需要执行一系列命令：登录服务器、停止旧进程、复制新文件、启动新进程。而在Kubernetes中，你只需要声明"我想要3个Pod运行v2.0版本的镜像"，Kubernetes会自动完成升级和替换过程。这种范式转变带来的是更高的抽象层次和更少的人为错误。

**自动化的自愈能力**。Kubernetes通过控制器（Controller）模式持续监控集群状态。应用进程崩溃了？自动重启。节点宕机了？自动在其他节点重建Pod。健康检查失败了？自动剔除流量并尝试恢复。这些能力不需要人工介入，让系统具备了真正的弹性。

**弹性伸缩能力**。通过Horizontal Pod Autoscaler（HPA），可以根据CPU、内存或自定义指标自动调整副本数量。流量增加时自动扩容，流量下降时自动缩容，既保证了服务质量，又避免了资源浪费。配合Cluster Autoscaler，甚至可以自动增删节点，实现集群级别的弹性。

**标准化的发布和回滚**。Kubernetes内置了滚动更新、蓝绿部署、金丝雀发布等策略。每次发布都有版本记录，回滚只需要一个命令。发布过程中如果探针检测到问题，会自动停止发布，避免故障扩大。

### 迁移的成本收益分析

尽管Kubernetes带来了诸多好处，但迁移本身是有成本的，需要理性评估。

**学习成本**。Kubernetes的概念体系相当复杂：Pod、Deployment、Service、Ingress、ConfigMap、Secret、PV、PVC等，每个概念背后都有深刻的设计考量。团队需要时间学习和适应这套新体系，这个过程可能需要数周甚至数月。

**改造成本**。应用需要容器化，配置需要外部化，日志需要标准化。对于有历史包袱的老应用来说，改造工作量可能不小。如果应用强依赖本地文件系统或使用了特殊的网络配置，改造难度会更大。

**基础设施成本**。需要搭建和维护Kubernetes集群，配置监控、日志、镜像仓库等周边设施。对于小团队来说，这些基础设施的维护成本不容忽视。不过，使用托管的Kubernetes服务（如EKS、GKE、ACK）可以大幅降低这部分成本。

**适用场景判断**。如果你的应用规模很小（比如只有一两个服务），流量稳定，团队规模不大，那么Kubernetes可能是"杀鸡用牛刀"。相反，如果你有多个微服务，需要频繁发布，流量波动大，团队希望实现DevOps自动化，那么迁移的收益会非常明显。

一个经验法则是：当你有3个以上的服务，或者单个服务需要5个以上的副本时，Kubernetes的价值开始显现。当服务数量达到10个以上时，不使用Kubernetes反而会更痛苦。

## 迁移前的评估框架

盲目迁移是灾难的开始。在动手写第一个Dockerfile或YAML之前，需要对现有应用做一次全面而系统的评估。这个评估过程本身就是一次宝贵的学习机会，能让你深入理解应用的真实状态。

### 应用特征分析

首先要回答的核心问题是：**你的应用是无状态的还是有状态的？**这个判断直接决定了迁移的难度和策略。

**无状态应用的特征**：
- 应用不在本地磁盘持久化任何业务数据
- 任何一个实例都能处理任何一个请求，实例之间完全对等
- 实例可以随时被销毁和重建而不影响业务连续性
- 会话数据存储在外部（如Redis），而不是应用进程的内存中
- 典型例子：RESTful API服务、无状态的Web前端、计算任务处理器

**有状态应用的特征**：
- 应用在本地文件系统存储数据，对存储有持久化要求
- 实例之间可能有主从关系或分片关系，不能简单地互相替换
- 需要稳定的网络标识（主机名或IP），其他组件会记录这个标识
- 启动顺序可能很重要，比如主节点必须先于从节点启动
- 典型例子：数据库、消息队列、分布式存储系统

这里需要澄清一个常见误解：有状态不等于"不能迁移到Kubernetes"，而是说迁移策略要不同。无状态应用用Deployment管理，支持快速滚动更新和弹性伸缩。有状态应用用StatefulSet管理，提供稳定的网络标识和顺序启动保证，配合PersistentVolume实现数据持久化。

**建议的迁移策略**：先迁移无状态应用，积累经验、建立信心后，再逐步迁移有状态应用。对于核心数据库等关键有状态服务，建议在迁移初期继续保持在传统基础设施上，等团队对Kubernetes的存储方案有了充分理解后再考虑容器化。

### 外部依赖梳理

应用很少是孤立存在的，通常会依赖大量外部服务。容器化后，网络环境会发生变化，所有外部依赖的连接方式都需要重新审视。

**数据库依赖**：
- 连接方式是内网IP、域名还是服务发现？
- 是否有IP白名单限制？容器的IP是动态分配的
- 连接池配置是否合理？容器环境下网络延迟特性可能不同
- 是否需要特殊的驱动或认证方式？

**缓存服务依赖**：
- Redis、Memcached的连接地址是否可配置？
- 是否使用了哨兵或集群模式？客户端需要支持
- 连接超时和重试策略是否适合容器环境？

**消息队列依赖**：
- Kafka、RabbitMQ的连接串是否硬编码？
- 消费者组的配置是否合理？容器重启会频繁改变实例标识
- 是否依赖特定的网络拓扑（如broker亲和性）？

**第三方API依赖**：
- 是否有IP白名单或地域限制？容器的出口IP会变化
- 是否依赖特定的DNS配置？
- 超时和重试策略是否健壮？容器环境下网络不稳定性可能增加

**文件系统依赖**：
- 是否读写特定路径（如/data、/opt）？容器内文件系统是隔离的
- 是否依赖NFS、CIFS等网络文件系统？需要考虑挂载方案
- 临时文件的处理方式是否合理？容器销毁后本地临时文件会丢失

建议为每个依赖项创建一个检查清单，逐一确认在容器环境中的可用性。对于有问题的依赖，提前制定改造方案或替代方案。

### 配置管理现状评估

配置管理的现状直接决定了容器化改造的工作量。根据当前配置方式的不同，改造难度差异巨大。

| 当前方式 | 示例 | 迁移难度 | 改造方向 | 工作量估算 |
|----------|------|----------|----------|------------|
| 硬编码在代码中 | `String dbHost = "192.168.1.10"` | 高 | 代码重构，提取为环境变量或配置文件 | 数天到数周 |
| 编译时注入 | Maven Profile、环境变量替换 | 中高 | 改为运行时配置，避免为每个环境构建不同镜像 | 数天 |
| 本地配置文件 | `/etc/app/config.ini` | 中 | ConfigMap挂载或环境变量注入 | 数小时到数天 |
| 配置中心 | Apollo、Nacos、Spring Cloud Config | 低 | 确保网络可达，几乎不需要改造 | 数小时 |
| 环境变量 | 从环境变量读取配置 | 极低 | 天然适合容器化 | 几乎无需改造 |

**最佳实践建议**：
- 所有环境差异性配置（数据库地址、API密钥、功能开关）必须外部化
- 尽量使用环境变量，这是容器生态的标准做法
- 敏感信息（密码、证书）应该用Secret管理，普通配置用ConfigMap
- 如果使用配置文件，考虑ConfigMap挂载为文件的方式

### 迁移难度评估矩阵

综合以上因素，可以为每个应用评估迁移难度，制定优先级策略。

```
迁移难度 = 应用状态性 + 依赖复杂度 + 配置改造难度

              低难度（优先迁移）          中难度                  高难度（后期迁移）
应用特征      无状态                     无状态，依赖多           有状态
              │                          │                      │
外部依赖      依赖少且标准化             依赖多但可配置           依赖多且复杂
              │                          │                      │
配置方式      环境变量或配置中心         本地配置文件             硬编码或特殊配置
              │                          │                      │
示例          RESTful API               传统Web应用              数据库、缓存集群
              单页应用前端               消息消费者               有状态的计算任务
```

**建议的迁移顺序**：
1. 第一批：无状态、依赖少、配置规范的应用（如API Gateway、前端服务）
2. 第二批：无状态但依赖较多的应用（如业务服务）
3. 第三批：有状态但架构相对简单的应用（如单点的定时任务）
4. 第四批：核心有状态服务（如数据库、缓存），需要充分测试和预案

## 容器化的底层原理

在进入具体的容器化改造之前，理解容器的底层原理至关重要。很多人把容器类比为"轻量级虚拟机"，这个类比虽然有助于快速理解，但也容易产生误解。容器和虚拟机有着本质的不同，理解这些差异能帮助你做出更好的设计决策。

### 容器不是虚拟机

虚拟机是在物理硬件上模拟出完整的硬件环境，每个虚拟机运行自己的操作系统内核。虚拟机管理器（Hypervisor）负责硬件资源的虚拟化和隔离。这种架构的好处是隔离性强，不同虚拟机之间完全独立，但代价是资源开销大、启动速度慢。

```
虚拟机架构：
┌─────────────────────────────────────────┐
│  VM1        VM2        VM3              │
│  ┌───────┐  ┌───────┐  ┌───────┐       │
│  │ App A │  │ App B │  │ App C │       │
│  ├───────┤  ├───────┤  ├───────┤       │
│  │ OS    │  │ OS    │  │ OS    │       │ 每个VM有完整OS
│  └───────┘  └───────┘  └───────┘       │
├─────────────────────────────────────────┤
│        Hypervisor (ESXi/KVM)            │
├─────────────────────────────────────────┤
│        Physical Hardware                │
└─────────────────────────────────────────┘

容器架构：
┌─────────────────────────────────────────┐
│  Container1  Container2  Container3     │
│  ┌───────┐   ┌───────┐   ┌───────┐     │
│  │ App A │   │ App B │   │ App C │     │ 直接运行应用
│  └───────┘   └───────┘   └───────┘     │
├─────────────────────────────────────────┤
│    Container Runtime (containerd)       │
├─────────────────────────────────────────┤
│         Host Operating System           │ 共享OS内核
├─────────────────────────────────────────┤
│         Physical Hardware               │
└─────────────────────────────────────────┘
```

容器是进程级的隔离，所有容器共享宿主机的操作系统内核，只是通过Linux内核的特性（Namespace和Cgroups）实现隔离。容器的本质就是一个特殊的进程，只不过这个进程被"框"在了一个隔离的环境里。

这种架构带来的影响是：
- **启动速度**：虚拟机启动需要引导完整的操作系统，通常需要分钟级；容器启动只是启动一个进程，通常只需要秒级甚至毫秒级
- **资源占用**：虚拟机需要为每个OS实例分配内存和存储，一个VM可能占用数GB内存；容器只需要应用本身的内存，可能只有数十MB
- **隔离性**：虚拟机隔离性更强，容器隔离性相对较弱，但对大多数场景已经足够
- **可移植性**：虚拟机镜像包含完整OS，体积庞大；容器镜像只包含应用和依赖，体积小且易于分发

### Linux Namespace：进程隔离的魔法

Namespace是Linux内核提供的一种资源隔离机制。通过Namespace，可以让不同的进程组看到不同的系统资源视图，就好像每个进程组都在独立的系统中运行一样。

**PID Namespace**：隔离进程ID空间。在容器内部，进程看到的PID从1开始，但在宿主机上这些进程有不同的PID。

```bash
# 在宿主机上
$ ps aux | grep nginx
root     12345  nginx: master process

# 在容器内
$ ps aux
PID   USER     COMMAND
1     root     nginx: master process  # 在容器内PID是1
```

这个隔离的意义在于：容器内的主进程总是PID 1，这符合Unix的进程管理惯例。PID 1进程负责回收僵尸进程，容器的生命周期与PID 1进程绑定，PID 1进程退出则容器退出。

**Network Namespace**：隔离网络设备、IP地址、路由表、端口等网络资源。每个容器有自己的网络栈，不会与其他容器冲突。

```bash
# 容器1可以监听8080端口
# 容器2也可以监听8080端口
# 两者不冲突，因为在不同的Network Namespace中
```

容器的网络有多种模式：
- Bridge模式：容器有独立的IP，通过虚拟网桥与宿主机通信（默认模式）
- Host模式：容器共享宿主机的网络命名空间，性能最好但失去了隔离性
- Container模式：多个容器共享同一个网络命名空间（Kubernetes的Pod就是这个原理）

**Mount Namespace**：隔离文件系统挂载点。容器有自己的根文件系统，看不到宿主机的文件系统。

```bash
# 容器内看到的根目录
/ (容器的根文件系统)
├── bin/
├── etc/
├── usr/
└── app/  # 应用代码

# 实际上这个根文件系统可能在宿主机的某个目录下
/var/lib/docker/overlay2/xxx/merged
```

这就是为什么容器内的进程看不到宿主机的文件，也看不到其他容器的文件。每个容器有自己独立的文件系统视图。

**UTS Namespace**：隔离主机名和域名。每个容器可以有自己的主机名，互不影响。

```bash
# 容器1的主机名
hostname: web-server-1

# 容器2的主机名
hostname: web-server-2
```

在Kubernetes中，Pod的主机名默认就是Pod的名称，这使得应用可以通过主机名识别自己的身份。

**IPC Namespace**：隔离进程间通信资源（如消息队列、信号量、共享内存）。不同容器的进程无法通过IPC机制通信，除非显式配置共享IPC命名空间。

**User Namespace**：隔离用户ID和组ID。容器内的root用户（UID 0）可以映射到宿主机上的非特权用户，这大大增强了安全性。

```bash
# 容器内
$ id
uid=0(root) gid=0(root) groups=0(root)

# 宿主机上查看该进程
$ ps aux | grep nginx
100000   12345  nginx: master process  # 实际UID是100000
```

这就是User Namespace重映射，即使容器内进程以root运行,在宿主机上也是非特权用户，限制了潜在的安全风险。

### Cgroups：资源限制的守护者

Namespace解决了"看见什么"的问题（隔离），Cgroups（Control Groups）解决了"能用多少"的问题（限制）。

**CPU Cgroup**：限制进程的CPU使用。

Kubernetes中配置的`cpu: 500m`（0.5个核心）最终会转换为Cgroups的配置：
```bash
# CFS (Completely Fair Scheduler) 周期为100ms
cpu.cfs_period_us = 100000  # 100ms
# 配额为50ms，即这100ms中只能用50ms
cpu.cfs_quota_us = 50000    # 50ms
```

当容器的进程用完了50ms的CPU时间，内核会限制它，必须等到下个100ms周期才能继续运行。这就是为什么CPU是"可压缩资源"——超限时进程变慢但不会被杀。

**Memory Cgroup**：限制进程的内存使用。

```bash
# 设置内存限制为512MB
memory.limit_in_bytes = 536870912

# 设置OOM控制
memory.oom_control = 1  # 启用OOM Killer
```

当容器的内存使用达到限制时，内核的OOM Killer会被触发，选择容器内的某个进程杀掉（通常是占用内存最多的进程）。如果主进程被杀，整个容器就会退出。这就是为什么内存是"不可压缩资源"——超限时进程会被杀。

**IO Cgroup**：限制进程的磁盘读写速度。

```bash
# 限制读速度为100MB/s
blkio.throttle.read_bps_device = 100000000

# 限制IOPS
blkio.throttle.read_iops_device = 1000
```

这对于多租户环境很重要，防止某个容器的大量IO操作影响其他容器的磁盘性能。

**理解这些底层机制的实际意义**：

1. **资源配置的本质**：当你在Kubernetes中配置`resources.requests`和`resources.limits`时，最终都会转换为Cgroups的配置。理解了Cgroups的工作原理，就能理解为什么CPU和内存的行为如此不同。

2. **问题排查**：当容器被OOM Killed时，可以通过查看`/sys/fs/cgroup/memory/memory.stat`理解内存的实际使用情况。当容器响应变慢时，可以检查`cpu.stat`中的`throttled_time`判断是否被CPU限流。

3. **性能优化**：知道了容器的隔离是基于Namespace和Cgroups，就能理解为什么容器的性能开销很小——它不需要硬件虚拟化，也不需要运行独立的OS内核，只是进程级的隔离。

### 联合文件系统：镜像分层的秘密

Docker镜像为什么能做到如此高效的存储和分发？秘密就在于联合文件系统（Union File System）。

**分层的概念**：

一个Docker镜像不是一个完整的文件系统副本，而是由多个只读层叠加而成。每一层代表Dockerfile中的一条指令。

```dockerfile
FROM ubuntu:20.04        # 层1：基础镜像层
RUN apt-get update       # 层2：安装依赖的层
COPY app.jar /app.jar    # 层3：应用文件的层
CMD ["java", "-jar", "/app.jar"]  # 层4：元数据（不占存储）
```

```
运行时的文件系统视图：
┌────────────────────────┐
│  读写层（Container）     │  可写，容器运行时的修改在这里
├────────────────────────┤
│  层3: app.jar           │  只读
├────────────────────────┤
│  层2: 依赖和工具         │  只读
├────────────────────────┤
│  层1: Ubuntu基础系统     │  只读
└────────────────────────┘
```

**OverlayFS的工作原理**（Docker目前主要使用的联合文件系统）：

OverlayFS将多个目录"叠加"成一个目录。它有三个核心概念：
- **Lower层**：只读的底层，对应镜像的各个层
- **Upper层**：读写层，对应容器运行时的修改
- **Merged层**：容器内看到的合并后的文件系统

```bash
# OverlayFS的挂载命令（简化示意）
mount -t overlay overlay \
  -o lowerdir=/layer1:/layer2:/layer3,\
     upperdir=/upper,\
     workdir=/work \
  /merged
```

**写时复制（Copy-on-Write）机制**：

当容器需要修改一个来自镜像层的文件时：
1. 文件最初只存在于只读的lower层
2. 容器尝试写入时，OverlayFS将文件复制到upper层
3. 后续的读写都在upper层进行
4. 容器删除时，upper层被删除，镜像层保持不变

这个机制的优势：
- **镜像共享**：多个容器可以共享同一个镜像的只读层，大大节省存储空间
- **快速启动**：不需要复制完整的文件系统，只需要创建一个新的upper层
- **镜像缓存**：构建镜像时，未修改的层可以使用缓存，加速构建

**层缓存的失效机制**：

```dockerfile
FROM ubuntu:20.04
RUN apt-get update && apt-get install -y curl
COPY pom.xml /app/         # 改动这里会导致下面的层失效
RUN mvn dependency:resolve
COPY src /app/src          # 经常改动
RUN mvn package
```

Docker按顺序执行每条指令，每条指令创建一个新层。如果某层的内容发生变化（比如COPY的文件变了），该层以及之后的所有层都会失效，需要重新构建。

**优化策略**：
- 把不常变化的指令放在前面（如安装依赖）
- 把经常变化的指令放在后面（如复制源代码）
- 合并RUN指令减少层数（但要权衡可读性和缓存命中率）

**实际影响**：

理解了分层机制，就能理解为什么：
- 镜像仓库推送很快：只推送变化的层
- 多个应用共享基础镜像很高效：基础层只存储一份
- Dockerfile的顺序很重要：影响缓存命中率和构建速度
- 镜像大小很关键：每一层都会累加，删除文件不会减小镜像大小（因为删除操作是在upper层标记删除，lower层的文件仍然存在）

## 容器化改造的核心原则

理解了容器的底层原理后，我们来看如何改造应用以适应容器环境。这些原则来源于**12-Factor App方法论**，它总结了构建现代云原生应用的最佳实践。

### 配置与代码分离

**原则的本质**：代码是不变的逻辑，配置是可变的参数。应用的行为由代码决定，应用的实例特征由配置决定。

传统应用常见的反模式：
```java
// 反模式：配置硬编码在代码中
public class DatabaseConfig {
    private static final String DB_HOST = "192.168.1.10";
    private static final int DB_PORT = 3306;
    private static final String DB_NAME = "production_db";
}
```

这种做法的问题：
- 不同环境（开发、测试、生产）需要编译不同的代码或使用编译时变量替换
- 配置修改需要重新编译和发布
- 无法在运行时动态调整
- 配置信息可能硬编码在源码中，有安全风险

**容器化的正确做法**：
```java
// 正确：从环境变量读取配置
public class DatabaseConfig {
    private final String dbHost = System.getenv("DB_HOST");
    private final int dbPort = Integer.parseInt(
        System.getenv().getOrDefault("DB_PORT", "3306")
    );
    private final String dbName = System.getenv("DB_NAME");
}
```

在Kubernetes中注入配置：
```yaml
env:
  - name: DB_HOST
    value: "mysql.database.svc.cluster.local"
  - name: DB_PORT
    value: "3306"
  - name: DB_NAME
    valueFrom:
      configMapKeyRef:
        name: app-config
        key: database.name
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: db-credentials
        key: password
```

**深层原因**：容器镜像应该是不可变的（Immutable）。同一个镜像应该能部署到任何环境，环境差异完全由配置控制。这样可以：
- 确保开发和生产环境的一致性（同一个镜像）
- 简化CI/CD流程（构建一次，到处部署）
- 便于回滚（配置和代码分离,配置错误不需要重新构建镜像）
- 提高安全性（敏感配置不会被编码到镜像中）

### 日志输出到标准输出

**原则的本质**：应用只负责生成日志，不负责路由和存储日志。日志的收集、聚合、存储由基础设施层面统一处理。

传统应用常见的做法：
```java
// 反模式：日志写入文件
<appender name="FILE" class="ch.qos.logback.core.FileAppender">
    <file>/var/log/myapp/application.log</file>
    <encoder>
        <pattern>%d{HH:mm:ss.SSS} [%thread] %-5level %logger{36} - %msg%n</pattern>
    </encoder>
</appender>
```

这种做法在容器环境中的问题：
- 容器是短暂的，容器销毁时日志文件也丢失
- 需要额外配置Volume来持久化日志目录
- 需要进入容器或挂载Volume才能查看日志
- 日志轮转、清理需要额外的机制
- 多实例时日志分散在各个容器中，难以聚合查询

**容器化的正确做法**：
```java
// 正确：日志输出到stdout/stderr
<appender name="STDOUT" class="ch.qos.logback.core.ConsoleAppender">
    <encoder>
        <pattern>%d{HH:mm:ss.SSS} [%thread] %-5level %logger{36} - %msg%n</pattern>
    </encoder>
</appender>
```

**容器运行时如何处理日志**：

1. 应用将日志写入stdout/stderr
2. 容器运行时（如containerd）捕获这些输出
3. 存储到节点上的日志文件（通常是`/var/log/containers/`）
4. Kubernetes可以通过`kubectl logs`查看这些日志
5. 日志采集器（如Fluentd、Filebeat）从节点收集日志
6. 发送到中心化日志系统（如Elasticsearch、Loki）

```bash
# Kubernetes日志链路
应用 → stdout → 容器运行时 → 节点日志文件 → 日志采集器 → 中心化存储
```

**实际好处**：
- 简化应用代码，不需要管理日志文件和轮转
- 统一的日志收集机制，所有应用遵循同一套标准
- 可以通过`kubectl logs`直接查看，便于快速排查问题
- 支持实时查看和跟踪（`kubectl logs -f`）
- 便于与ELK、Loki等日志方案集成

### 进程无状态化

**原则的本质**：应用进程本身不保存需要在多个请求之间共享的状态。状态应该存储在外部的后端服务中。

典型的反模式：
```java
// 反模式：在内存中保存用户会话
public class SessionManager {
    private static Map<String, UserSession> sessions = new HashMap<>();

    public void saveSession(String sessionId, UserSession session) {
        sessions.put(sessionId, session);  // 存储在进程内存中
    }
}
```

这种做法的问题：
- 无法水平扩展：用户的下一个请求可能被路由到不同的实例
- 无法实现优雅重启：进程重启会丢失所有会话
- 内存泄漏风险：如果session不清理，内存会持续增长
- 无法实现自动伸缩：新增的实例没有历史session数据

**无状态设计的正确做法**：
```java
// 正确：会话存储在Redis中
public class SessionManager {
    private RedisClient redis;

    public void saveSession(String sessionId, UserSession session) {
        redis.set("session:" + sessionId,
                  JSON.toJSONString(session),
                  3600);  // 存储在外部Redis，1小时过期
    }
}
```

**无状态设计的范畴**：

- **用户会话**：存储在Redis或其他分布式缓存中
- **上传文件**：存储在对象存储（S3、MinIO）或分布式文件系统，不要存在容器本地
- **任务队列**：使用外部消息队列（Kafka、RabbitMQ），不要在内存中维护队列
- **缓存数据**：使用Redis等外部缓存，本地缓存只用于不重要的临时数据
- **锁和分布式协调**：使用Redis、Zookeeper、etcd等，不要用本地文件锁

**无状态的好处**：
- 任何实例挂掉都不会丢失数据
- 可以随意增删实例，实现弹性伸缩
- 简化负载均衡，不需要会话亲和性
- 便于实现滚动更新和蓝绿部署

**注意**：无状态不是说应用完全没有状态，而是说状态不保存在应用进程的本地存储中。应用仍然可以有状态，只是这些状态存储在外部的有状态服务中。

### 快速启动和优雅终止

**原则的本质**：应用应该能够快速启动（秒级），并且能够响应终止信号优雅关闭，不丢失正在处理的请求。

**快速启动的重要性**：
- 弹性伸缩依赖快速启动：如果启动需要数分钟，就无法快速响应流量变化
- 故障恢复依赖快速启动：应用崩溃后需要尽快重启恢复服务
- 滚动更新依赖快速启动：新版本实例启动慢会导致更新过程长，风险增加

**优化启动速度的方法**：
- 延迟加载：只加载启动必需的资源,其他资源按需加载
- 并行初始化：独立的初始化任务并行执行
- 预编译：避免启动时JIT编译（Java可以考虑GraalVM Native Image）
- 精简依赖：移除不必要的库和框架

**优雅终止的重要性**：

当Kubernetes决定终止一个Pod时（可能是缩容、更新、节点维护等），会经历以下过程：

```
1. Pod标记为Terminating，从Service端点列表移除
2. 同时，向容器发送SIGTERM信号
3. 等待terminationGracePeriodSeconds（默认30秒）
4. 如果进程仍未退出，发送SIGKILL强制杀死
```

问题在于，步骤1和步骤2是并行的，从端点移除有网络传播延迟。这可能导致：
- 负载均衡器还没更新端点列表，继续发送请求到正在终止的Pod
- Pod收到SIGTERM立即退出，正在处理的请求被中断

**实现优雅终止的正确姿势**：

```java
// 监听SIGTERM信号
Runtime.getRuntime().addShutdownHook(new Thread(() -> {
    log.info("收到终止信号，开始优雅关闭");

    // 1. 停止接受新请求
    server.stopAcceptingRequests();

    // 2. 等待现有请求处理完成（设置超时）
    server.awaitTermination(20, TimeUnit.SECONDS);

    // 3. 关闭资源（数据库连接池、线程池等）
    closeResources();

    log.info("优雅关闭完成");
}));
```

在Kubernetes中配合使用preStop钩子：
```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 5"]
```

这个sleep 5的作用是：给Service端点列表足够的时间传播到所有负载均衡器，确保不再有新请求路由到这个Pod。然后应用的优雅关闭逻辑会处理现有请求。

完整的流程：
```
1. Pod标记为Terminating，从端点移除
2. preStop执行：sleep 5秒（等待端点传播）
3. 发送SIGTERM，应用停止接受新请求
4. 等待现有请求处理完成（最多20秒）
5. 应用退出
6. 如果第2-5步超过30秒，SIGKILL强制杀死
```

## Dockerfile编写的关键考量

有了容器化原则后，下一步是将应用打包成镜像。Dockerfile是镜像构建的蓝图，写好它直接影响镜像的大小、构建速度、安全性和运行性能。

### 多阶段构建的深层原理

**传统单阶段构建的问题**：

```dockerfile
FROM maven:3.9-eclipse-temurin-21
WORKDIR /app
COPY . .
RUN mvn clean package
CMD ["java", "-jar", "target/app.jar"]
```

这个Dockerfile看起来很简单，但有严重问题：
- 最终镜像包含整个Maven和JDK，体积庞大（可能超过800MB）
- 包含了源代码和构建工具，有安全风险
- 构建依赖和运行依赖混在一起，无法优化

**多阶段构建的解决方案**：

```dockerfile
# 阶段1：构建阶段（Builder）
FROM maven:3.9-eclipse-temurin-21 AS builder
WORKDIR /app

# 先复制pom.xml，利用层缓存
COPY pom.xml .
RUN mvn dependency:go-offline

# 再复制源码并构建
COPY src ./src
RUN mvn package -DskipTests

# 阶段2：运行阶段（Runtime）
FROM eclipse-temurin:21-jre-alpine
WORKDIR /app

# 从builder阶段复制构建产物
COPY --from=builder /app/target/*.jar app.jar

# 创建非root用户
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/actuator/health || exit 1

EXPOSE 8080
ENTRYPOINT ["java", "-XX:+UseContainerSupport", "-XX:MaxRAMPercentage=75.0", "-jar", "app.jar"]
```

**多阶段构建的工作原理**：

Docker在构建时会依次执行每个FROM指令，每个FROM开始一个新的构建阶段。只有最后一个阶段的内容会成为最终镜像，中间阶段可以用`COPY --from=<阶段名>`来复制文件，但不会包含在最终镜像中。

```
构建过程：
┌──────────────────────────────┐
│ 阶段1 (builder)               │
│ ┌─────────────────────────┐  │
│ │ Maven + JDK           │  │  包含编译工具
│ │ 源代码                │  │
│ │ ↓编译                 │  │
│ │ target/app.jar         │  │  构建产物
│ └─────────────────────────┘  │
└──────────────────────────────┘
         │ COPY --from=builder
         ↓
┌──────────────────────────────┐
│ 阶段2 (最终镜像)              │
│ ┌─────────────────────────┐  │
│ │ JRE only              │  │  只有运行时
│ │ app.jar                │  │  只有jar包
│ └─────────────────────────┘  │
└──────────────────────────────┘
```

**镜像大小对比**：
- 单阶段（Maven + JDK）：约850MB
- 多阶段（JRE-Alpine）：约180MB
- 使用Distroless base：约120MB
- 使用GraalVM Native Image：约40MB

**优化技巧**：

1. **利用层缓存**：先复制pom.xml并下载依赖，这样源码变动时不需要重新下载依赖。

2. **并行构建阶段**：如果有独立的构建任务，可以并行：
```dockerfile
FROM node:18 AS frontend-builder
COPY frontend/ /app
RUN npm ci && npm run build

FROM maven:3.9 AS backend-builder
COPY backend/ /app
RUN mvn package

FROM eclipse-temurin:21-jre-alpine
COPY --from=backend-builder /app/target/app.jar .
COPY --from=frontend-builder /app/dist ./static
```

3. **使用构建缓存mount**（Docker BuildKit特性）：
```dockerfile
RUN --mount=type=cache,target=/root/.m2 mvn package
```
这会在多次构建之间共享Maven本地仓库，大大加快构建速度。

### 镜像分层机制详解

每条Dockerfile指令都会创建一个新层（除了一些元数据指令如CMD、EXPOSE等）。理解分层机制是优化Dockerfile的关键。

**层的特点**：
- 每一层是只读的
- 层是增量的：只包含与上一层的差异
- 层可以被共享：多个镜像可以共享相同的基础层
- 层是有大小的：即使后续层删除了文件，该文件仍然存在于之前的层中

**反模式示例**：
```dockerfile
# 反模式：每条指令一个RUN
RUN apt-get update
RUN apt-get install -y curl
RUN apt-get install -y vim
RUN apt-get clean  # 这不会减小镜像大小！
RUN rm -rf /var/lib/apt/lists/*  # 这也不会！
```

这个Dockerfile有5个RUN指令，创建了5个层：
- 层1：包含apt-get update的结果
- 层2：包含curl包
- 层3：包含vim包
- 层4：标记清理操作
- 层5：标记删除操作

**关键问题**：层4和层5的删除操作只是在新层中标记删除，底层1-3的文件仍然存在。最终镜像大小是所有层的叠加，删除操作不会减小大小。

**正确做法**：
```dockerfile
# 正确：在同一层中下载、安装、清理
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl vim && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
```

这样只创建一个层，清理操作在同一层中生效，最终层不包含临时文件。

**层缓存失效规则**：

Docker按顺序执行Dockerfile指令，对每条指令：
1. 检查是否有缓存的层可以复用
2. 对于ADD/COPY指令，比较文件内容的checksum
3. 对于RUN指令，比较指令字符串
4. 如果某层缓存失效，后续所有层都会失效

**优化Dockerfile顺序**：
```dockerfile
FROM maven:3.9-eclipse-temurin-21 AS builder
WORKDIR /app

# 不常变化的指令放前面
COPY pom.xml .
RUN mvn dependency:go-offline  # 依赖很少变化，缓存命中率高

# 经常变化的指令放后面
COPY src ./src
RUN mvn package  # 源码经常变，但至少依赖层是缓存的
```

### 非root用户运行的安全原理

默认情况下，容器内的进程以root用户（UID 0）运行。虽然容器有namespace隔离，但如果出现以下情况，仍然有安全风险：
- 容器逃逸漏洞（如runc漏洞）
- 挂载了宿主机目录（hostPath）
- 使用了特权容器（privileged）

**风险场景**：
```yaml
# 危险：特权容器 + root用户
securityContext:
  privileged: true
volumeMounts:
  - name: host-root
    mountPath: /host
    readOnly: false
```

如果容器以root运行且挂载了宿主机根目录，攻击者可以通过容器修改宿主机的任意文件，包括添加SSH密钥、修改sudo配置等。

**使用非root用户的正确做法**：

```dockerfile
# 创建用户和组
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# 修改文件所有权
RUN chown -R appuser:appgroup /app

# 切换到非root用户
USER appuser
```

在Kubernetes中进一步加强：
```yaml
securityContext:
  runAsNonRoot: true      # 强制非root
  runAsUser: 1000         # 指定UID
  runAsGroup: 1000        # 指定GID
  allowPrivilegeEscalation: false  # 禁止提权
  capabilities:
    drop:
      - ALL               # 移除所有capabilities
  readOnlyRootFilesystem: true  # 只读根文件系统
```

**注意事项**：
- 应用可能需要写入某些目录（如/tmp），使用emptyDir volume挂载
- 某些端口（<1024）需要root权限，改用高端口（如8080而不是80）
- 日志、缓存等目录需要确保非root用户有写权限

### 镜像精简策略

镜像越小，拉取越快，启动越快，攻击面越小。

**基础镜像选择**：

| 基础镜像 | 大小 | 特点 | 适用场景 |
|----------|------|------|----------|
| ubuntu:22.04 | ~80MB | 完整的Ubuntu，有包管理器 | 需要调试工具，不太在意大小 |
| alpine:3.19 | ~7MB | 极简系统，使用musl libc | 对大小敏感，应用简单 |
| distroless/java | ~200MB | 只有JRE和运行时依赖，无shell | 生产环境，注重安全 |
| scratch | 0MB | 空镜像，只能运行静态链接的二进制 | Go程序，Native Image |

**Alpine的优缺点**：
- 优点：体积极小，安全更新快
- 缺点：使用musl libc而非glibc，可能有兼容性问题，DNS解析在某些环境下有bug

**Distroless的优势**：
- 不包含shell、包管理器等工具，攻击面极小
- 只包含应用运行必需的库
- 无法进入容器执行命令（这是feature不是bug，生产环境不应该进入容器调试）

**使用Distroless的例子**：
```dockerfile
FROM maven:3.9 AS builder
WORKDIR /app
COPY pom.xml .
RUN mvn dependency:go-offline
COPY src ./src
RUN mvn package

FROM gcr.io/distroless/java21-debian12
COPY --from=builder /app/target/app.jar /app.jar
USER nonroot:nonroot
CMD ["app.jar"]
```

**精简技巧**：
- 移除不必要的文件（文档、示例代码、缓存）
- 使用.dockerignore排除不需要的文件
- 合并RUN指令并在同一层清理临时文件
- 对于Go程序，使用`CGO_ENABLED=0`编译静态二进制，可以使用scratch基础镜像

## 小结

单体服务迁移到Kubernetes的第一步，不是急着写YAML配置，而是做好充分的准备工作：

**迁移前的评估**：
- 明确迁移的动机和预期收益，避免为了技术而技术
- 梳理应用的特征（有状态vs无状态）和外部依赖
- 评估配置管理的现状，制定改造计划
- 根据难度制定分批迁移策略，先易后难

**深入理解容器原理**：
- 容器不是虚拟机，是基于Namespace和Cgroups的进程隔离
- Namespace提供隔离（看见什么），Cgroups提供限制（能用多少）
- 联合文件系统实现镜像分层和快速启动
- 理解这些原理能帮助你更好地排查问题和优化性能

**遵循容器化最佳实践**：
- 配置与代码分离，实现镜像的不可变性
- 日志输出到标准输出，融入容器日志生态
- 进程无状态化，支持水平扩展和弹性伸缩
- 快速启动和优雅终止，适应云原生环境

**编写高质量的Dockerfile**：
- 使用多阶段构建分离编译和运行环境
- 理解镜像分层机制，优化层缓存利用
- 使用非root用户运行，遵循最小权限原则
- 选择合适的基础镜像，在功能、大小和安全之间权衡

下一篇将介绍在Kubernetes中如何设计部署策略、配置资源限制、实现服务发现和暴露、设计健康检查机制等运行时层面的关键决策。

## 常见问题

### Q1: 单体应用一定要拆成微服务才能上Kubernetes吗？

完全不需要。这是一个常见的误解，很多人把"微服务"和"容器化"混为一谈。

Kubernetes管理的核心单元是容器，而不是微服务。一个单体应用完全可以作为一个整体打包成一个容器镜像，部署到Kubernetes上，享受Kubernetes带来的所有好处：自动扩缩容、滚动更新、自愈能力、声明式管理等。

微服务架构和容器化是两个独立的技术决策。微服务解决的是系统架构和团队协作的问题，容器化解决的是部署和运维的问题。它们可以结合使用，但不是必须的。

推荐的演进路径是：先将单体应用容器化并迁移到Kubernetes，在这个过程中熟悉Kubernetes的运作方式，积累经验。当系统运行稳定后，再根据业务需要评估是否拆分微服务。这样可以降低风险，避免同时进行架构改造和基础设施迁移这两个复杂的变更。

实际上，很多成功的案例都是"单体应用 + Kubernetes"的模式。只要应用本身设计合理（无状态、可扩展），单体架构在相当长的时间内都是可行的选择。

### Q2: 应用依赖的数据库也要迁移到Kubernetes里吗？

通常不建议在迁移初期就将数据库放入Kubernetes。数据库是典型的有状态服务，对存储的性能、可靠性、持久性有极高要求，迁移风险很大。

数据库在Kubernetes中运行需要解决的问题：
- **存储性能**：容器的存储IO性能能否满足数据库要求？需要使用高性能的StorageClass（如local-path或云厂商的SSD存储类）
- **数据持久化**：需要正确配置PersistentVolume，确保Pod重建后数据不丢失
- **备份恢复**：如何在容器环境中实现数据库备份和恢复策略？
- **主从复制**：如果是主从架构，如何保证主库先启动？如何处理主从切换？
- **监控告警**：容器化后的数据库如何监控？传统的监控工具可能不适用

更稳妥的做法是：
1. **初期**：数据库保持在传统基础设施上（物理机、虚拟机或云RDS），应用容器化后通过网络访问数据库。只需要确保网络连通性和连接配置外部化即可。
2. **中期**：团队对Kubernetes的存储方案（PV/PVC/StorageClass）有了充分理解和实践经验后，可以考虑在Kubernetes中运行非核心数据库（如开发测试环境的数据库）。
3. **后期**：使用Operator（如Percona Operator、Redis Operator）管理数据库，这些Operator封装了数据库的运维逻辑，降低了容器化数据库的复杂度。

对于核心生产数据库，使用云厂商的托管服务（RDS、Cloud SQL等）通常是更好的选择，既能享受容器化应用的灵活性，又不用承担容器化数据库的风险。

### Q3: 理解Namespace和Cgroups对实际工作有什么帮助？

理解这些底层原理在多个场景下都很有实际价值，而不仅仅是理论知识。

**排查性能问题时**：
- 当应用响应变慢时，可以检查`/sys/fs/cgroup/cpu/cpu.stat`查看`throttled_time`，判断是否被CPU限流。如果限流严重，说明CPU limits设置太低。
- 当容器频繁被OOM Killed时，可以查看`/sys/fs/cgroup/memory/memory.stat`了解内存的实际使用分布（cache、RSS、swap等），判断是真的内存不够还是配置不当。

**理解资源配置的本质**：
- 知道了Cgroups的工作原理，就能理解为什么CPU是"可压缩资源"（超限时被限流），而内存是"不可压缩资源"（超限时被杀死）。
- 理解了`cpu.cfs_period_us`和`cpu.cfs_quota_us`的关系，就能更精确地配置CPU资源。

**安全方面的考虑**：
- 理解了User Namespace，就知道为什么要配置`runAsNonRoot`和UID映射，以及它如何降低安全风险。
- 理解了Mount Namespace，就知道为什么容器看不到宿主机文件系统，以及为什么hostPath挂载有安全风险。

**网络问题排查**：
- 理解了Network Namespace，就能明白为什么每个Pod有独立的网络栈，以及在排查网络问题时应该在哪个namespace中执行命令。
- 可以使用`nsenter`进入容器的namespace进行深度调试。

**优化镜像和启动速度**：
- 理解了OverlayFS的分层和Copy-on-Write机制，就能明白为什么Dockerfile的指令顺序会影响构建速度和镜像大小。
- 知道了层缓存的失效规则，就能优化Dockerfile顺序，提高构建效率。

这些原理知识在日常工作中的价值，就像医生需要理解人体生理学一样——不是每天都要用到，但在遇到疑难杂症时，这些知识能让你快速定位问题根源，而不是盲目试错。

### Q4: 多阶段构建和单阶段构建的镜像在运行时有区别吗？

运行时完全没有区别。多阶段构建只影响构建过程和最终镜像的内容，不影响运行时行为。

当你运行一个容器时，Docker或Kubernetes只关心最终镜像（Dockerfile中最后一个FROM指令之后的内容）。无论这个镜像是通过单阶段构建还是多阶段构建产生的，只要最终的文件内容相同，运行时行为就完全一致。

多阶段构建的优势全部体现在构建产物上：
- **镜像更小**：不包含构建工具和中间文件，体积可能小数倍
- **更安全**：不包含源代码和构建工具，减少了攻击面
- **更快的分发**：镜像小意味着拉取快，冷启动时间短

但这些优势都是静态的（镜像本身），而不是动态的（运行时行为）。应用该怎么运行还是怎么运行，该消耗多少CPU和内存还是多少，不会因为构建方式不同而改变。

**一个比喻**：多阶段构建就像是包装商品时去掉了外箱、说明书、包装泡沫，只保留商品本身。商品的功能没有变化，但包裹变得更小更轻，运输更快更便宜。

因此，推荐所有生产环境的镜像都使用多阶段构建，这是没有副作用的纯收益优化。唯一需要注意的是，调试时如果需要进入容器使用shell或其他工具，多阶段构建的精简镜像可能没有这些工具。这时可以：
- 在开发环境使用包含调试工具的镜像
- 在生产环境使用精简镜像，配合`kubectl debug`等工具进行临时调试
- 使用Distroless镜像，彻底杜绝进入容器的可能性（这反而是生产环境的最佳实践）

### Q5: 如果应用强依赖本地文件系统怎么办？

首先要区分文件的用途和生命周期，针对不同情况采用不同的解决方案。

**临时文件（生命周期 = 容器生命周期）**：
- 用途：缓存、临时计算结果、进程间通信的socket文件
- 方案：使用`emptyDir` volume
```yaml
volumes:
  - name: tmp
    emptyDir: {}
volumeMounts:
  - name: tmp
    mountPath: /tmp
```
- 特点：Pod创建时自动创建，Pod删除时自动清理，不需要持久化

**需要持久化的业务文件（生命周期 > 容器生命周期）**：
- 用途：用户上传的文件、生成的报表、需要长期保存的数据
- **最佳方案**：改为使用对象存储（S3、MinIO、阿里云OSS等）
  - 优点：真正的持久化、可扩展、支持CDN加速、成本较低
  - 应用改造：将文件I/O改为对象存储API调用
- **次优方案**：使用PersistentVolume
```yaml
volumes:
  - name: data
    persistentVolumeClaim:
      claimName: app-data-pvc
```
  - 优点：对应用改造小，接口仍然是文件系统
  - 缺点：绑定特定节点（除非使用网络存储），限制Pod调度灵活性

**共享文件（多个Pod需要访问同一份文件）**：
- 用途：多个实例共享的配置、静态资源
- 方案1：使用ReadWriteMany模式的PV（需要NFS或CephFS等支持）
- 方案2：改为从配置中心或对象存储动态加载
- 方案3：将静态资源打包到镜像中（适合不常变化的资源）

**短期过渡方案**：使用`hostPath`
```yaml
volumes:
  - name: data
    hostPath:
      path: /mnt/app-data
      type: DirectoryOrCreate
```
- 警告：这会将Pod绑定到特定节点，破坏了容器的可移植性
- 仅适用于：开发环境、单节点集群、或明确需要节点亲和性的场景
- 生产环境应尽快改造为更合适的方案

**改造建议的优先级**：
1. 如果是临时文件 → 使用emptyDir，几乎不需要改造
2. 如果是需要持久化的业务数据 → 改为对象存储（工作量较大但收益最大）
3. 如果短期无法改造 → 使用PV（可行但限制调度灵活性）
4. 如果是遗留系统改造困难 → 先用hostPath过渡，但制定明确的改造计划

关键原则是：容器化的目标之一是实现应用的可移植性和弹性，强依赖本地文件系统会破坏这个目标。因此应该尽可能地将文件存储外部化，这样才能充分发挥Kubernetes的能力。

## 参考资源

- [12-Factor App 方法论](https://12factor.net/)
- [Docker 官方最佳实践](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [Kubernetes 迁移指南](https://kubernetes.io/docs/tasks/administer-cluster/)

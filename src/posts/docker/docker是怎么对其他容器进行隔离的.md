---
date: 2025-07-01
author: Gaaming Zhang
category:
  - docker
tag:
  - docker
  - 还在施工中
---

# Docker 是怎么对其他容器进行隔离的

## 核心隔离机制

Docker 容器隔离基于 Linux 内核的三大核心技术：**Namespace（命名空间）**、**Cgroups（控制组）** 和 **Union FS（联合文件系统）**，实现进程、网络、文件系统、资源的多维度隔离。这些技术共同作用，使得容器能够在共享宿主内核的同时，实现高度的隔离性和安全性。

---

## 1. Namespace（命名空间）- 视图隔离

Namespace 是 Linux 内核提供的一种轻量级虚拟化技术，通过为容器创建独立的系统资源视图，让容器"以为"自己独占整个系统。Linux 提供了 7 种不同类型的 Namespace，分别隔离不同的系统资源：

### PID Namespace（进程隔离）

- **功能**：为容器创建独立的进程树，每个容器都有自己的 PID 1 进程
- **隔离效果**：
  - 容器内进程看不到宿主机或其他容器的进程
  - 宿主机可以看到所有容器进程，但 PID 与容器内不同
  - 容器内进程的父进程关系仅在容器内可见
- **实现细节**：
  - 每个 PID Namespace 有自己的进程 ID 分配范围
  - 容器内的 PID 1 进程负责管理容器内的其他进程
  - 当 PID 1 进程终止时，容器内所有进程都会被终止

```bash
# 容器内查看进程（只看到容器内进程，PID 从 1 开始）
docker run --rm alpine ps aux

# 宿主机查看容器进程（看到真实 PID）
docker run -d --name nginx nginx
docker top nginx
ps aux | grep nginx
```

### Network Namespace（网络隔离）

- **功能**：为容器创建独立的网络栈，包括网卡、IP 地址、路由表、iptables 规则和端口号
- **隔离效果**：
  - 容器默认拥有自己的网络命名空间，与其他容器和宿主机隔离
  - 容器间需要通过网络连接（如 Docker 网络）才能通信
  - 支持多种网络模式，满足不同场景需求
- **Docker 网络模式**：
  - **bridge**：默认模式，容器连接到 Docker 桥接网络
  - **host**：共享宿主机网络栈，无网络隔离
  - **container**：共享其他容器的网络命名空间
  - **none**：无网络配置，完全隔离
  - **overlay**：用于跨主机容器通信的覆盖网络
  - **macvlan**：为容器分配 MAC 地址，直接连接物理网络

```bash
# 查看容器网络命名空间
docker inspect -f '{{.NetworkSettings.Networks}}' <container>

# 宿主机查看所有网络命名空间
ip netns list

# 验证网络隔离
docker run -d --name web1 nginx
docker run -d --name web2 nginx
docker run --rm alpine ping web1  # 默认无法访问，需同网络

# 创建自定义网络并验证通信
docker network create mynet
docker network connect mynet web1
docker network connect mynet web2
docker run --rm --network mynet alpine ping web1  # 可通信
```

### Mount Namespace（文件系统隔离）

- **功能**：为容器创建独立的挂载点，使容器内的挂载操作不影响宿主机
- **隔离效果**：
  - 容器内的文件系统挂载与宿主机完全隔离
  - 容器可以有自己的根文件系统
  - 结合 Union FS 实现分层文件系统
- **实现细节**：
  - 每个 Mount Namespace 有自己的挂载表
  - 容器启动时，内核会为其创建新的挂载命名空间
  - 容器内的 `mount` 和 `umount` 命令仅影响容器内的挂载点

### UTS Namespace（主机名隔离）

- **功能**：为容器创建独立的主机名（hostname）和域名（domain name）
- **隔离效果**：
  - 容器可以有自己的主机名，与宿主机和其他容器不同
  - 容器内的 `hostname` 命令仅影响容器自身
- **使用示例**：

```bash
# 设置容器主机名
docker run --rm --hostname mycontainer alpine hostname
# 输出: mycontainer
```

### IPC Namespace（进程间通信隔离）

- **功能**：隔离容器间的进程间通信机制，包括消息队列、信号量和共享内存
- **隔离效果**：
  - 容器内进程无法与其他容器或宿主机的进程进行 IPC 通信
  - 确保容器间的通信安全，避免信息泄露
- **使用场景**：
  - 适用于需要严格隔离的多租户环境
  - 防止恶意容器通过 IPC 机制攻击其他容器

### User Namespace（用户隔离）

- **功能**：将容器内的用户 ID 和组 ID 映射到宿主机上的非特权用户和组
- **隔离效果**：
  - 容器内的 root 用户（UID 0）映射到宿主机上的普通用户
  - 即使容器内的进程获得 root 权限，也无法在宿主机上获得特权
- **安全优势**：
  - 大大降低了容器逃逸的风险
  - 即使容器被攻破，也无法直接影响宿主机
- **配置方法**：
  - 需要在 Docker 守护进程配置中启用 `userns-remap` 选项
  - 配置文件：`/etc/docker/daemon.json`

```json
{
  "userns-remap": "default"
}
```

### Cgroup Namespace（控制组隔离，Linux 4.6+）

- **功能**：隔离容器内的 Cgroup 视图，使容器只能看到自己的 Cgroup 层次结构
- **隔离效果**：
  - 容器内无法查看或修改宿主机的 Cgroup 配置
  - 容器只能管理自己的资源限制
- **实现细节**：
  - 每个 Cgroup Namespace 有自己的 Cgroup 根目录
  - 容器内的 `/sys/fs/cgroup` 挂载点只显示容器自己的 Cgroup

---

## 2. Cgroups（控制组）- 资源限制

Cgroups（Control Groups）是 Linux 内核提供的一种资源管理机制，用于限制、记录和隔离进程组使用的物理资源（CPU、内存、磁盘 I/O、网络等）。

### Cgroups 的主要功能

- **资源限制**：限制进程组可以使用的资源总量
- **优先级分配**：为不同进程组分配不同的资源使用优先级
- **资源监控**：统计进程组使用的资源量
- **资源控制**：暂停/恢复进程组，或终止超出资源限制的进程

### CPU 资源限制

- **CPU 份额（shares）**：相对权重，用于 CPU 资源竞争时的分配比例
- **CPU 核心绑定（cpuset）**：将容器绑定到特定 CPU 核心
- **CPU 使用率限制（cpus）**：限制容器可以使用的 CPU 时间百分比
- **CPU 周期限制**：限制容器在特定时间周期内可以使用的 CPU 时间

```bash
# 限制 CPU 份额（默认 1024）
docker run --cpu-shares 512 <image>

# 限制 CPU 核心数
docker run --cpus 2 <image>  # 最多使用 2 个 CPU 核心
docker run --cpuset-cpus 0,1 <image>  # 仅使用 CPU 0 和 1

# 限制 CPU 周期
docker run --cpu-period 100000 --cpu-quota 50000 <image>  # 50% CPU 使用率
```

### 内存资源限制

- **内存总量限制**：限制容器可以使用的最大内存量
- **Swap 限制**：限制容器可以使用的 Swap 空间
- **OOM 控制**：配置内存不足时的行为

```bash
# 限制最大内存（512MB）
docker run -m 512m <image>

# 限制内存 + Swap（512MB 内存 + 512MB Swap）
docker run -m 512m --memory-swap 1g <image>

# 内存不足时不终止容器（需谨慎）
docker run --oom-kill-disable <image>

# 设置内存软限制（优先回收）
docker run --memory-reservation 256m <image>
```

### 磁盘 I/O 限制

- **带宽限制**：限制容器的磁盘读写速率
- **IOPS 限制**：限制容器的 I/O 操作次数
- **权重分配**：为不同容器分配不同的磁盘 I/O 优先级

```bash
# 限制读写速率（1MB/s）
docker run --device-read-bps /dev/sda:1mb --device-write-bps /dev/sda:1mb <image>

# 限制 IOPS
docker run --device-read-iops /dev/sda:100 --device-write-iops /dev/sda:100 <image>

# 设置 I/O 权重
docker run --blkio-weight 500 <image>
```

### 网络带宽限制

- Docker 原生不直接支持网络带宽限制
- 可通过以下方式实现：
  - 使用第三方工具如 `tc`（Traffic Control）
  - 使用 Docker 网络插件如 Weave Net、Calico
  - 在 Kubernetes 环境中使用网络策略和 QoS 配置

### 查看 Cgroup 配置

```bash
# 获取容器在宿主机的 PID
docker inspect -f '{{.State.Pid}}' <container>

# 查看该进程的 Cgroup 配置
cat /proc/<PID>/cgroup

# 查看具体资源限制
cat /sys/fs/cgroup/cpu/docker/<container_id>/cpu.shares
cat /sys/fs/cgroup/memory/docker/<container_id>/memory.limit_in_bytes
cat /sys/fs/cgroup/blkio/docker/<container_id>/blkio.throttle.read_bps_device
```

---

## 3. Union FS（联合文件系统）- 文件隔离

Union FS 是一种分层、轻量级的文件系统，允许将多个不同的文件系统挂载到同一个目录下，形成一个统一的文件系统视图。Docker 使用 Union FS 实现镜像的分层存储和容器的文件系统隔离。

### 分层存储机制

- **镜像层**：只读层，由 Dockerfile 中的指令生成
- **容器层**：可写层，在容器运行时创建，用于存储容器的修改
- **共享机制**：多个容器可以共享同一个镜像层，节省磁盘空间

### 写时复制（Copy-on-Write）

- **原理**：当容器修改文件时，系统会将该文件从只读的镜像层复制到可写的容器层，然后进行修改
- **优势**：
  - 节省磁盘空间，多个容器共享镜像层
  - 提高容器启动速度，无需复制整个镜像
  - 保证镜像的完整性，防止被意外修改

### 常见 Union FS 实现

| 类型 | 特点 | 适用场景 |
|------|------|----------|
| OverlayFS | 性能优异，内存占用低 | 现代 Linux 系统（推荐） |
| AUFS | 成熟稳定，支持多层叠加 | Debian/Ubuntu 系统 |
| Btrfs | 支持快照和克隆 | 对存储性能要求高的场景 |
| Device Mapper | 基于块设备，支持快照 | 早期 Docker 版本，RHEL/CentOS 系统 |
| ZFS | 高性能，支持大容量存储 | 大规模存储场景 |

### 文件系统操作示例

```bash
# 查看 Docker 存储驱动
docker info | grep "Storage Driver"

# 查看镜像层
docker history nginx
docker inspect nginx | grep -A 10 Layers

# 查看容器层路径
docker inspect -f '{{.GraphDriver.Data}}' <container>

# 查看容器文件系统内容
docker exec -it <container> ls -la /
```

---

## 4. 安全加固机制

除了核心隔离技术外，Docker 还提供了多种安全加固机制，进一步提高容器的安全性。

### Linux Capabilities

- **原理**：将传统的 root 权限划分为多个细粒度的能力，允许容器只获取必要的权限
- **默认配置**：Docker 默认移除了大部分危险的 Capabilities
- **常见 Capabilities**：
  - `CAP_NET_ADMIN`：网络管理权限
  - `CAP_SYS_ADMIN`：系统管理权限（危险）
  - `CAP_SYS_CHROOT`：chroot 权限
  - `CAP_NET_RAW`：原始套接字权限

```bash
# 查看容器默认 Capabilities
docker run --rm alpine capsh --print | grep Current

# 添加特定 Capability
docker run --cap-add NET_ADMIN --rm alpine ip link add dummy0 type dummy

# 移除所有 Capabilities，只保留必要的
docker run --cap-drop ALL --cap-add CHOWN --cap-add DAC_OVERRIDE --rm alpine chown root:root /tmp
```

### Seccomp（安全计算模式）

- **原理**：限制容器可以调用的系统调用，减少攻击面
- **默认配置**：Docker 默认使用一个严格的 Seccomp profile
- **隔离效果**：
  - 禁止危险的系统调用，如 `clone()`、`mount()` 等
  - 只允许容器运行所需的基本系统调用

```bash
# 查看默认 Seccomp profile
docker run --rm alpine grep -r "default" /etc/docker/

# 使用自定义 Seccomp profile
docker run --security-opt seccomp=profile.json <image>

# 禁用 Seccomp（不推荐）
docker run --security-opt seccomp=unconfined <image>
```

### AppArmor / SELinux

- **原理**：强制访问控制（MAC）机制，限制进程对文件、网络和能力的访问
- **AppArmor**：Ubuntu/Debian 系统默认使用
- **SELinux**：RHEL/CentOS 系统默认使用
- **隔离效果**：
  - 限制容器对宿主机文件系统的访问
  - 限制容器的网络访问
  - 限制容器的能力使用

```bash
# 查看容器的 AppArmor 配置
docker inspect <container> | grep AppArmorProfile

# 查看 SELinux 标签
docker run --rm centos ls -Z /

# 禁用 AppArmor（不推荐）
docker run --security-opt apparmor=unconfined <image>
```

### 只读文件系统

- **原理**：将容器的根文件系统设置为只读，防止恶意修改
- **使用场景**：运行不可变基础设施，提高安全性
- **配置方法**：

```bash
# 根目录只读，临时目录可写
docker run --read-only --tmpfs /tmp --tmpfs /run <image>

# 允许写入特定目录
docker run --read-only -v /data rw <image>
```

---

## 相关高频面试题及答案

### Q1: Docker 容器与虚拟机的隔离有什么区别？

**答案**：

| 特性 | Docker 容器 | 虚拟机 |
|------|------------|--------|
| 隔离级别 | 轻量级（共享内核） | 重量级（完全隔离） |
| 启动时间 | 毫秒级 | 秒级 |
| 资源占用 | 低（MB 级别） | 高（GB 级别） |
| 隔离技术 | Namespace + Cgroups + Union FS | Hypervisor + 独立内核 |
| 安全性 | 较低（共享内核） | 较高（完全隔离） |
| 性能 | 接近原生 | 有虚拟化开销 |
| 部署密度 | 高（数百个容器/主机） | 低（数十个 VM/主机） |

### Q2: 容器之间真的完全隔离吗？有哪些安全风险？

**答案**：

容器不是完全隔离的，主要安全风险包括：

1. **内核共享风险**：容器共享宿主机内核，内核漏洞可能导致容器逃逸
2. **权限提升风险**：不当的权限配置可能导致容器内进程获得宿主机特权
3. **网络安全风险**：网络配置不当可能导致容器间或容器与宿主机的网络攻击
4. **存储安全风险**：不安全的卷挂载可能导致数据泄露或篡改
5. **镜像安全风险**：使用不安全的基础镜像可能引入恶意代码

**安全加固措施**：
- 启用 User Namespace，隔离容器用户与宿主机用户
- 最小化容器权限，仅授予必要的 Capabilities
- 使用严格的 Seccomp 和 AppArmor/SELinux 配置
- 配置只读文件系统，防止恶意修改
- 使用私有镜像仓库，确保镜像安全
- 定期更新容器和宿主机内核

### Q3: 如何查看容器的 Namespace 和 Cgroup 配置？

**答案**：

```bash
# 1. 获取容器在宿主机的 PID
docker inspect -f '{{.State.Pid}}' <container>  # 假设输出为 12345

# 2. 查看该进程的 Namespace
ls -la /proc/12345/ns/  # 列出所有 Namespace

# 3. 进入容器的 Network Namespace
sudo nsenter --net=/proc/12345/ns/net ip addr  # 查看容器网络配置

# 4. 查看 Cgroup 配置
cat /proc/12345/cgroup  # 查看进程所属的 Cgroup

# 5. 查看具体资源限制
cat /sys/fs/cgroup/cpu/docker/<container_id>/cpu.shares
cat /sys/fs/cgroup/memory/docker/<container_id>/memory.limit_in_bytes
```

### Q4: Docker 的 --privileged 特权模式有什么风险？

**答案**：

--privileged 模式会赋予容器几乎所有的系统权限，包括：

1. **完整的 Capabilities**：授予容器所有 Linux Capabilities
2. **设备访问权限**：允许访问宿主机所有设备（/dev/*）
3. **禁用安全机制**：禁用 Seccomp 和 AppArmor/SELinux 限制
4. **挂载权限**：允许容器挂载宿主机文件系统

**风险**：
- 容器可以轻易逃逸到宿主机
- 容器可以修改宿主机文件系统
- 容器可以访问宿主机所有设备
- 容器可以执行危险的系统调用

**最佳实践**：
- 生产环境禁止使用 --privileged 模式
- 需要特权操作时，使用最小化 Capabilities 替代
- 对于需要访问特定设备的场景，使用 --device 参数单独授权

### Q5: 如何限制容器只能访问特定网络或服务？

**答案**：

1. **使用 Docker 自定义网络**：

```bash
# 创建隔离网络
docker network create --driver bridge isolated-net

# 运行容器在隔离网络
docker run --network isolated-net <image>

# 只允许特定容器加入该网络
docker network connect isolated-net <container>
```

2. **使用 iptables 规则**：

```bash
# 限制容器只能访问特定 IP 和端口
iptables -A DOCKER-USER -s <container_ip> -d <target_ip> --dport <target_port> -j ACCEPT
iptables -A DOCKER-USER -s <container_ip> -j DROP
```

3. **在 Kubernetes 环境中使用 NetworkPolicy**：

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: restrict-access
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: myapp
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: allowed-app
    ports:
    - protocol: TCP
      port: 80
  egress:
  - to:
    - ipBlock:
        cidr: 10.0.0.0/24
    ports:
    - protocol: TCP
      port: 53
    - protocol: UDP
      port: 53
```

### Q6: 容器的根文件系统存储在哪里？如何清理？

**答案**：

1. **存储位置**：
   - 默认存储路径：`/var/lib/docker/`
   - 具体位置取决于存储驱动：
     - OverlayFS：`/var/lib/docker/overlay2/`
     - AUFS：`/var/lib/docker/aufs/`
     - Device Mapper：`/var/lib/docker/devicemapper/`

2. **查看存储路径**：

```bash
docker info | grep "Docker Root Dir"
```

3. **清理命令**：

```bash
# 清理未使用的容器、镜像、网络和卷
docker system prune

# 清理所有未使用的资源（包括停止的容器和未使用的镜像）
docker system prune -a

# 清理所有未使用的资源，包括卷
docker system prune -a --volumes

# 清理特定类型的资源
docker container prune  # 清理停止的容器
docker image prune  # 清理未使用的镜像
docker volume prune  # 清理未使用的卷
docker network prune  # 清理未使用的网络
```

### Q7: 什么是 Docker 容器逃逸？如何防止？

**答案**：

**容器逃逸**：指容器内的进程突破容器隔离，获得对宿主机的访问权限。

**常见逃逸方式**：
1. **内核漏洞利用**：利用 Linux 内核漏洞突破 Namespace 隔离
2. **权限提升**：通过不当的权限配置获得宿主机特权
3. **危险挂载**：挂载宿主机敏感目录（如 /proc、/sys）并修改
4. **特权模式**：使用 --privileged 模式运行容器
5. **恶意镜像**：使用包含恶意代码的基础镜像

**防止措施**：
1. 保持内核更新，修复已知漏洞
2. 启用 User Namespace，隔离容器用户
3. 最小化容器权限，仅授予必要的 Capabilities
4. 使用严格的 Seccomp 和 AppArmor/SELinux 配置
5. 配置只读文件系统
6. 禁止使用 --privileged 模式
7. 安全挂载卷，避免挂载敏感目录
8. 使用可信的基础镜像，定期扫描镜像漏洞

### Q8: 如何监控容器的资源使用情况？

**答案**：

1. **Docker 内置命令**：

```bash
# 查看容器资源使用情况
docker stats <container>

# 查看容器详细统计信息
docker stats --all --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"
```

2. **cAdvisor**：Google 开发的容器监控工具，提供容器资源使用情况的实时监控和历史数据

```bash
docker run -d --name cadvisor -p 8080:8080 -v /:/rootfs:ro -v /var/run:/var/run:ro -v /sys:/sys:ro -v /var/lib/docker/:/var/lib/docker:ro gcr.io/cadvisor/cadvisor:latest
```

3. **Prometheus + Grafana**：
   - Prometheus 采集容器 metrics
   - Grafana 可视化展示

4. **Kubernetes 监控方案**：
   - metrics-server：提供基础资源指标
   - Prometheus Operator：完整的监控解决方案
   - Istio Service Mesh：服务级监控

### Q9: 什么是 Docker 镜像的分层存储？有什么优势？

**答案**：

**分层存储**：Docker 镜像由多个只读层叠加而成，每个层对应 Dockerfile 中的一条指令。当容器运行时，Docker 会在镜像层之上添加一个可写层，用于存储容器的修改。

**优势**：

1. **空间效率**：多个容器可以共享同一个镜像层，节省磁盘空间
2. **时间效率**：容器启动时无需复制整个镜像，只需创建一个可写层
3. **版本控制**：每个层都有唯一的哈希值，便于镜像版本管理和回滚
4. **增量更新**：更新镜像时只需修改差异层，减少网络传输量
5. **缓存机制**：构建镜像时可以利用缓存，加速构建过程

### Q10: 如何优化 Docker 容器的性能？

**答案**：

1. **使用合适的存储驱动**：优先选择 OverlayFS，性能优异
2. **优化镜像大小**：
   - 使用轻量级基础镜像（如 Alpine）
   - 多层构建，减少最终镜像大小
   - 清理不必要的依赖和文件
3. **优化容器运行**：
   - 限制容器资源，避免资源竞争
   - 使用合适的网络模式
   - 启用 TSO/GRO 等网络加速特性
4. **优化应用程序**：
   - 调整应用程序参数，适应容器环境
   - 使用高性能的应用程序框架
   - 优化数据库连接和查询
5. **使用容器编排工具**：
   - Kubernetes：自动调度和资源管理
   - Docker Swarm：简化多容器管理

---

## 隔离验证实验

### 实验 1：验证 PID 隔离

```bash
# 1. 宿主机查看进程
ps aux | grep nginx

# 2. 启动 nginx 容器
docker run -d --name nginx-test nginx

# 3. 容器内查看进程（只看到容器内进程，PID 从 1 开始）
docker exec nginx-test ps aux

# 4. 宿主机查看容器进程（看到真实 PID）
docker top nginx-test
ps aux | grep nginx

# 5. 清理实验环境
docker rm -f nginx-test
```

### 实验 2：验证 Network 隔离

```bash
# 1. 启动两个 nginx 容器
docker run -d --name web1 nginx
docker run -d --name web2 nginx

# 2. 验证默认网络隔离
docker run --rm alpine ping -c 2 web1  # 无法解析
docker run --rm alpine wget -qO- http://web1:80  # 无法访问

# 3. 创建自定义网络并连接容器
docker network create my-net
docker network connect my-net web1
docker network connect my-net web2

# 4. 验证网络连通性
docker run --rm --network my-net alpine ping -c 2 web1  # 可通
docker run --rm --network my-net alpine wget -qO- http://web1:80  # 可访问

# 5. 清理实验环境
docker rm -f web1 web2
docker network rm my-net
```

### 实验 3：验证资源限制

```bash
# 1. 验证内存限制（100MB）
docker run -m 100m --rm progrium/stress --vm 1 --vm-bytes 150M  # 触发 OOM kill

# 2. 验证 CPU 限制（50%）
docker run --cpus 0.5 --rm progrium/stress --cpu 2  # CPU 使用率不超过 50%

# 3. 验证磁盘 I/O 限制
docker run --device-write-bps /dev/sda:1mb --rm alpine dd if=/dev/zero of=/tmp/test bs=1M count=10  # 写入速率约 1MB/s
```

---

## 总结

Docker 容器隔离是通过 Linux 内核的三大核心技术实现的：

1. **Namespace**：提供资源视图隔离，使容器"以为"自己独占系统资源
2. **Cgroups**：实现资源限制和监控，防止容器过度占用资源
3. **Union FS**：实现镜像分层存储和容器文件系统隔离

此外，Docker 还提供了多种安全加固机制，如 Capabilities、Seccomp、AppArmor/SELinux 和只读文件系统，进一步提高容器的安全性。

理解 Docker 容器的隔离机制对于设计和部署安全、高效的容器化应用至关重要。在实际应用中，需要根据具体场景选择合适的隔离策略和安全配置，确保容器的安全性和性能。

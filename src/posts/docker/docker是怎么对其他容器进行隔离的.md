---
icon: pen-to-square
date: 2025-12-21
author: Gaaming Zhang
category:
  - docker
tag:
  - docker
---

# docker是怎么对其他容器进行隔离的

#### 核心隔离机制

Docker 容器隔离基于 Linux 内核的三大核心技术：**Namespace（命名空间）**、**Cgroups（控制组）** 和 **Union FS（联合文件系统）**，实现进程、网络、文件系统、资源的多维度隔离。

---

#### 1. Namespace（命名空间）- 视图隔离

Namespace 为容器创建独立的系统资源视图，让容器"以为"自己独占整个系统，Linux 提供 7 种 Namespace：

**PID Namespace（进程隔离）**：
- 每个容器拥有独立的进程树，容器内 PID 1 是主进程
- 容器内进程看不到宿主机或其他容器的进程
- 宿主机可看到所有容器进程（但 PID 不同）

```bash
# 容器内
ps aux  # 只看到容器内进程，PID 从 1 开始

# 宿主机
ps aux | grep <container>  # 看到容器进程，PID 为宿主机分配的真实 PID
```

**Network Namespace（网络隔离）**：
- 独立的网络栈：网卡、IP、路由表、iptables、端口
- 容器间默认网络隔离，需通过 veth pair + bridge 或 overlay 互通
- 支持多种网络模式：bridge（默认）、host（共享宿主网络栈）、container（共享其他容器网络）、none（无网络）

```bash
# 查看容器网络 namespace
docker inspect <container> | grep NetworkMode
ip netns list  # 宿主机查看所有网络命名空间
```

**Mount Namespace（文件系统隔离）**：
- 独立的挂载点，容器内挂载不影响宿主机
- 结合 Union FS 实现分层文件系统

**UTS Namespace（主机名隔离）**：
- 独立的 hostname 和 domain name
- `docker run --hostname custom-host` 设置容器主机名

**IPC Namespace（进程间通信隔离）**：
- 独立的消息队列、信号量、共享内存
- 容器间无法通过 IPC 通信（除非显式共享）

**User Namespace（用户隔离）**：
- 将容器内 root 映射为宿主机非特权用户，提升安全性
- 需内核支持与配置 `userns-remap`

**Cgroup Namespace（控制组隔离，Linux 4.6+）**：
- 隔离 cgroup 视图，容器内看到的 cgroup 根为自己的 cgroup

---

#### 2. Cgroups（控制组）- 资源限制

Cgroups 限制容器资源使用量，防止单容器耗尽宿主资源或影响其他容器：

**CPU 限制**：
```bash
# 限制 CPU 份额（相对权重）
docker run --cpu-shares 512 <image>

# 限制 CPU 核心数
docker run --cpus 2 <image>

# 绑定特定 CPU 核
docker run --cpuset-cpus 0,1 <image>
```

**内存限制**：
```bash
# 限制最大内存
docker run -m 512m <image>

# 限制内存 + swap
docker run -m 512m --memory-swap 1g <image>

# OOM 时不杀容器（需谨慎）
docker run --oom-kill-disable <image>
```

**磁盘 I/O 限制**：
```bash
# 限制读写速率（字节/秒）
docker run --device-read-bps /dev/sda:1mb <image>
docker run --device-write-bps /dev/sda:1mb <image>

# 限制 IOPS
docker run --device-read-iops /dev/sda:100 <image>
```

**网络带宽限制**（需 tc 或第三方工具）：
- Docker 原生不支持，需配合 `tc` 或网络插件

**资源隔离路径**：
```bash
# 查看容器 cgroup 配置
cat /sys/fs/cgroup/cpu/docker/<container_id>/cpu.shares
cat /sys/fs/cgroup/memory/docker/<container_id>/memory.limit_in_bytes
```

---

#### 3. Union FS（联合文件系统）- 文件隔离

**分层存储**：
- 镜像由只读层堆叠，容器运行时在顶部添加可写层
- 多个容器共享相同镜像层，节省磁盘与内存
- 常见实现：OverlayFS、AUFS、Btrfs、Device Mapper

**写时复制（Copy-on-Write）**：
- 容器修改文件时，从只读层复制到可写层
- 删除文件实际是标记为删除，不影响底层镜像

```bash
# 查看容器文件系统驱动
docker info | grep "Storage Driver"

# 查看镜像层
docker history <image>
docker inspect <image> | grep Layers
```

---

#### 4. 安全加固机制

**Capabilities（Linux 能力）**：
- 细粒度权限控制，避免赋予容器完整 root 权限
- 默认移除大部分危险 capabilities（如 CAP_SYS_ADMIN、CAP_NET_ADMIN）

```bash
# 添加/删除 capabilities
docker run --cap-add NET_ADMIN <image>
docker run --cap-drop ALL --cap-add CHOWN <image>
```

**Seccomp（系统调用过滤）**：
- 限制容器可调用的系统调用，减少攻击面
- Docker 默认使用 seccomp profile

```bash
# 禁用 seccomp（不推荐）
docker run --security-opt seccomp=unconfined <image>

# 自定义 profile
docker run --security-opt seccomp=profile.json <image>
```

**AppArmor / SELinux（强制访问控制）**：
- 限制容器进程对文件、网络、能力的访问
- 需宿主机内核支持

```bash
# 查看 AppArmor 状态
docker inspect <container> | grep AppArmorProfile

# 禁用（不推荐）
docker run --security-opt apparmor=unconfined <image>
```

**只读文件系统**：
```bash
# 根目录只读，临时目录可写
docker run --read-only --tmpfs /tmp <image>
```

---

#### 相关高频面试题与简答

**Q1: Docker 容器与虚拟机的隔离有什么区别？**
- 虚拟机：通过 Hypervisor 硬件虚拟化，独立内核与操作系统，强隔离但资源开销大。
- 容器：共享宿主内核，通过 Namespace/Cgroups 隔离，轻量高效但隔离强度弱于虚拟机；内核漏洞可能逃逸影响宿主。

**Q2: 容器之间真的完全隔离吗？有哪些安全风险？**
- 不完全隔离：共享内核、默认 root 用户、部分系统调用未限制。
- 风险：特权容器可访问宿主设备、内核漏洞逃逸、容器间侧信道攻击（CPU/内存竞争）。
- 加固：User Namespace、Seccomp、只读文件系统、最小化 capabilities、禁止特权容器。

**Q3: 如何查看容器的 Namespace 和 Cgroup 配置？**
```bash
# 查看容器进程在宿主机的 PID
docker inspect -f '{{.State.Pid}}' <container>

# 查看该进程的 namespace
ls -l /proc/<PID>/ns

# 查看 cgroup 配置
cat /proc/<PID>/cgroup
cat /sys/fs/cgroup/cpu/docker/<container_id>/cpu.shares
```

**Q4: Docker 的 --privileged 特权模式有什么风险？**
- 赋予容器几乎所有 capabilities，禁用 Seccomp/AppArmor
- 容器可访问宿主所有设备（/dev/*），挂载宿主文件系统
- 风险：容器可轻易逃逸到宿主机，生产环境禁用；需要特权操作（如 DinD）时应用最小化 capabilities 替代。

**Q5: 如何限制容器只能访问特定网络或服务？**
```bash
# 自定义网络隔离
docker network create --driver bridge isolated-net
docker run --network isolated-net <image>

# Kubernetes NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
```

**Q6: 容器的根文件系统存储在哪里？如何清理？**
```bash
# 查看存储路径
docker info | grep "Docker Root Dir"
# 默认 /var/lib/docker

# 容器层路径
/var/lib/docker/overlay2/<container_id>/

# 清理未使用的容器、镜像、卷
docker system prune -a --volumes
```

---

#### 隔离验证实验

**验证 PID 隔离**：
```bash
# 宿主机
ps aux | grep nginx

# 启动容器
docker run -d nginx

# 容器内
docker exec <container> ps aux  # 只看到 nginx 进程

# 宿主机再查看
ps aux | grep nginx  # 可看到容器内 nginx，但 PID 不同
```

**验证 Network 隔离**：
```bash
# 容器 A
docker run -d --name web nginx

# 容器 B（默认无法访问容器 A 的 80 端口，除非同网络）
docker run --rm alpine ping web  # 无法解析

# 创建自定义网络
docker network create mynet
docker network connect mynet web
docker run --rm --network mynet alpine ping web  # 可通
```

**验证资源限制**：
```bash
# 限制内存 100M
docker run -m 100m --rm progrium/stress --vm 1 --vm-bytes 150M
# 触发 OOM kill

# 限制 CPU
docker run --cpus 0.5 --rm progrium/stress --cpu 2
# 查看 CPU 使用率不超过 50%
```

---

#### 总结对比表

| 隔离维度 | 技术实现                   | 隔离效果             | 典型用途                          |
| -------- | -------------------------- | -------------------- | --------------------------------- |
| 进程     | PID Namespace              | 独立进程树           | 避免进程间干扰                    |
| 网络     | Network Namespace          | 独立网络栈           | 端口隔离、多容器同端口            |
| 文件系统 | Mount Namespace + Union FS | 独立挂载点与分层存储 | 镜像复用、写时复制                |
| 资源     | Cgroups                    | CPU/内存/I/O 限制    | 防止资源争抢                      |
| 用户     | User Namespace             | UID/GID 映射         | 安全加固（容器 root ≠ 宿主 root） |
| 系统调用 | Seccomp                    | 限制可用 syscall     | 减少攻击面                        |
| 访问控制 | AppArmor/SELinux           | 强制访问策略         | 限制文件/网络访问                 |

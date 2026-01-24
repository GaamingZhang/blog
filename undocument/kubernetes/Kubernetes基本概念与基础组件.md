---
date: 2025-07-01
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
---

# Kubernetes 基本概念与基础组件

## 核心概念

### Pod
- Kubernetes 中最小的可部署计算单元
- 包含一个或多个紧密关联的容器，共享网络、存储和进程命名空间
- **网络特性**：同一 Pod 内的容器通过 localhost 直接通信，共享相同的 IP 地址和端口空间
- **存储特性**：可以挂载相同的存储卷，实现容器间数据共享
- **命名空间共享**：默认共享网络命名空间，可选共享 PID 和 IPC 命名空间
- **健康检查**：支持三种探针机制确保容器健康：
  - liveness probe：检测容器是否存活
  - readiness probe：检测容器是否准备好接收请求
  - startup probe：检测容器是否完成启动
- **初始化容器**：Init Container 在主容器启动前执行，用于完成前置依赖检查和准备工作

### Service
- 为一组具有相同标签的 Pod 提供稳定的网络访问入口
- 核心特性：
  - 固定 IP 地址和 DNS 名称（在集群内部可解析）
  - 通过标签选择器（label selector）动态关联 Pod
  - 自动负载均衡（默认轮询算法）
- **Service 类型**：
  - **ClusterIP**：默认类型，仅集群内部可访问
  - **NodePort**：在每个节点上暴露一个端口，通过节点 IP:端口访问服务
  - **LoadBalancer**：结合云平台负载均衡器，提供外部访问入口
  - **ExternalName**：通过 DNS CNAME 记录将服务映射到外部域名
  - **Headless**：无头服务，不分配 ClusterIP，用于 StatefulSet 的服务发现
- **会话亲和性**：支持基于客户端 IP 的会话保持

### Deployment
- 用于管理无状态应用的部署、扩展和更新的控制器
- **核心功能**：
  - 通过 ReplicaSet 确保指定数量的 Pod 副本始终运行
  - 支持滚动更新（RollingUpdate）策略，确保服务在更新过程中不中断
  - 支持回滚操作，可以恢复到之前的稳定版本
  - 支持暂停和继续更新过程，便于灰度发布
- **更新策略配置**：
  - maxSurge：更新期间允许超出期望 Pod 数量的最大个数
  - maxUnavailable：更新期间允许不可用的最大 Pod 数量
- **典型应用场景**：Web 服务器、API 服务等无状态应用

### StatefulSet
- 用于管理有状态应用的控制器，为 Pod 提供唯一标识和稳定的存储
- **核心特性**：
  - **稳定的网络标识**：为每个 Pod 分配唯一的主机名和 DNS 记录
  - **稳定的持久化存储**：每个 Pod 对应独立的 PVC，即使 Pod 重新调度也保持不变
  - **有序部署和删除**：按照序号顺序创建和删除 Pod
  - **有序扩缩容**：扩缩容时保持 Pod 的序号顺序
- **典型应用场景**：数据库（如 MySQL、PostgreSQL）、分布式存储（如 ZooKeeper）、消息队列（如 Kafka）

### DaemonSet
- 确保所有（或部分）节点上都运行一个 Pod 副本的控制器
- **核心功能**：
  - 自动在新加入的节点上部署 Pod
  - 自动在节点移除时清理 Pod
  - 支持通过节点选择器选择特定节点
- **典型应用场景**：
  - 日志收集代理（如 Fluentd、Logstash）
  - 监控代理（如 Prometheus Node Exporter）
  - 网络插件（如 CNI 插件）

### Job/CronJob
- **Job**：用于执行一次性任务的控制器，确保任务成功完成
  - 可以并行执行多个 Pod 实例
  - 支持设置完成超时时间
  - 支持失败重试策略
- **CronJob**：基于时间调度的 Job，类似 Linux 系统的 cron
  - 支持标准 cron 表达式定义执行时间
  - 支持并发策略（允许、禁止、替换）
  - 支持历史任务保留策略
- **典型应用场景**：
  - 数据备份和恢复
  - 定时报表生成
  - 系统维护任务

### Namespace
- Kubernetes 中的资源隔离机制，用于创建多个虚拟集群
- **核心功能**：
  - 资源命名隔离：不同命名空间中的资源可以重名
  - 资源配额管理：通过 ResourceQuota 限制命名空间的资源使用
  - 资源限制管理：通过 LimitRange 限制单个 Pod 的资源使用
  - 访问控制：通过 RBAC 控制不同命名空间的访问权限
- **默认命名空间**：
  - **default**：默认的用户工作空间
  - **kube-system**：系统组件和控制器运行的空间
  - **kube-public**：所有用户可见的公共资源空间
  - **kube-node-lease**：节点心跳和租约信息的空间

### ConfigMap/Secret
- **ConfigMap**：用于存储非敏感配置信息的资源
  - 支持键值对形式的配置
  - 支持通过环境变量注入容器
  - 支持通过文件挂载到容器
  - 支持动态更新（部分类型需要重启 Pod）
- **Secret**：用于存储敏感信息的资源
  - 默认使用 Base64 编码存储
  - **Secret 类型**：
    - Opaque：通用密钥值对
    - docker-registry：Docker 镜像仓库认证信息
    - tls：TLS 证书和私钥
    - service-account-token：服务账号令牌
  - 支持加密存储（需要配置 etcd 加密）

### Volume
- Kubernetes 中的存储抽象，用于在容器之间和 Pod 重启后持久化数据
- **核心概念**：
  - **Volume**：Pod 级别的存储抽象
  - **PersistentVolume (PV)**：集群级别的存储资源池
  - **PersistentVolumeClaim (PVC)**：Pod 对存储资源的请求
  - **StorageClass**：定义存储类型和动态供应策略
- **Volume 类型**：
  - 本地存储：emptyDir、hostPath、local
  - 网络存储：NFS、iSCSI
  - 云存储：AWS EBS、GCE PD、Azure Disk
  - 分布式存储：Ceph、GlusterFS、Longhorn
- **生命周期管理**：
  - 静态供应：手动创建 PV 供 PVC 使用
  - 动态供应：通过 StorageClass 自动创建 PV

## 控制平面组件（Master）

### kube-apiserver
- Kubernetes 集群的统一 API 入口点和控制中心
- **核心功能**：
  - 提供 RESTful API 接口，处理所有客户端请求（kubectl、kubelet、控制器等）
  - 实现完整的安全机制：
    - 认证（Authentication）：验证请求者身份
    - 授权（Authorization）：验证请求者是否有权限执行操作
    - 准入控制（Admission Control）：在资源创建/更新前验证和修改请求
  - 作为集群的 "通信枢纽"，所有组件都通过它进行通信
  - 与 etcd 交互存储和获取集群状态数据
- **架构特点**：
  - 无状态设计，支持水平扩展
  - 分层 API 设计，支持不同版本的 API
  - 支持多租户隔离和资源配额管理

### etcd
- 高可用的分布式键值对（KV）存储系统，是 Kubernetes 集群的 "唯一事实来源"
- **核心特性**：
  - 强一致性保证（使用 Raft 共识算法）
  - 持久化存储集群所有配置和状态数据
  - 支持事务操作和监听机制
  - 提供安全的通信机制（TLS 加密）
- **存储内容**：
  - 集群配置数据（API Server 配置、调度器配置等）
  - 资源对象定义（Pod、Service、Deployment 等）
  - 集群状态信息（节点状态、Pod 状态等）
- **内部数据结构**：使用 MVCC（多版本并发控制）B+ 树存储数据
  - **Key 格式**：采用复合结构 `[key] + [revision]`
    - `key`：用户实际存储的键名（字节序列）
    - `revision`：64位整数版本号，用于实现多版本控制
      - 高32位：主版本号（main revision）
      - 低32位：子版本号（sub revision）
      - 采用小端序编码存储
  - **Value 格式**：使用 Protocol Buffers 序列化，包含以下字段
    ```protobuf
    message KeyValue {
      bytes key = 1;              // 实际键名
      int64 create_revision = 2;  // 创建时的版本号
      int64 mod_revision = 3;     // 最后修改时的版本号
      int64 version = 4;          // 键的版本计数（从1开始）
      int64 lease = 5;            // 关联的租约ID（0表示无租约）
      bytes value = 6;            // 实际存储的数据
    }
    ```
- **部署建议**：
  - 独立于 Kubernetes 控制平面部署
  - 通常采用奇数个节点（3、5 或 7 个）以实现高可用
  - 配置适当的备份策略，防止数据丢失

### kube-scheduler
- 负责将 Pod 调度到合适的节点上运行的控制平面组件
- **调度流程**：
  - **监听**：持续监听 kube-apiserver 中的未调度 Pod
  - **过滤**：根据硬约束（如资源需求、节点选择器、亲和性规则）筛选出可调度节点
  - **评分**：根据软约束（如资源利用率、负载均衡）对可选节点进行评分
  - **绑定**：将 Pod 绑定到得分最高的节点
- **调度策略**：
  - **资源感知调度**：考虑 CPU、内存、GPU 等资源需求
  - **亲和性与反亲和性**：
    - 节点亲和性：将 Pod 调度到特定类型的节点
    - Pod 亲和性：将相关 Pod 调度到同一节点或拓扑域
    - Pod 反亲和性：避免相关 Pod 调度到同一节点或拓扑域
  - **污点（Taint）和容忍度（Toleration）**：控制 Pod 是否可以调度到有污点的节点
  - **拓扑约束**：确保 Pod 分布在不同的可用区或机架

### kube-controller-manager
- 运行所有 Kubernetes 控制器的核心组件，确保集群实际状态与期望状态一致
- **主要控制器**：
  - **Node Controller**：监控节点状态，处理节点故障
  - **ReplicaSet Controller**：确保指定数量的 Pod 副本始终运行
  - **Deployment Controller**：管理 Deployment 资源的生命周期
  - **StatefulSet Controller**：管理有状态应用的部署和扩展
  - **Endpoints Controller**：维护 Service 与 Pod 之间的关联关系
  - **Service Account & Token Controller**：管理服务账号和认证令牌
  - **Namespace Controller**：管理命名空间的生命周期
  - **PersistentVolume Controller**：管理持久化存储的生命周期
- **控制器模式**：
  - 持续监控集群状态
  - 检测实际状态与期望状态的差异
  - 执行操作以消除差异

### cloud-controller-manager
- 用于将 Kubernetes 与云平台集成的控制平面组件
- **核心功能**：
  - 实现与云服务商的特定功能集成
  - 分离云平台特定的控制逻辑，避免核心代码与云平台耦合
- **主要控制器**：
  - **Node Controller**：与云平台交互，管理节点的创建和删除
  - **Route Controller**：配置节点间的网络路由
  - **Service Controller**：管理云平台负载均衡器的创建和配置
  - **Volume Controller**：管理云存储卷的创建、挂载和卸载
- **支持的云平台**：AWS、Azure、GCP、阿里云、腾讯云等

## Node 组件

### kubelet
- 运行在每个节点上的核心代理组件，是节点与控制平面通信的主要接口
- **核心职责**：
  - 管理 Pod 生命周期的各个阶段：创建、启动、监控、停止和销毁
  - 与 kube-apiserver 通信，接收 Pod 规范并确保节点上运行的容器与规范一致
  - 执行容器健康检查（通过探针机制）：
    - liveness probe：检测容器是否存活
    - readiness probe：检测容器是否准备好接收请求
    - startup probe：检测容器是否完成启动
  - 向 kube-apiserver 报告节点和 Pod 的状态、资源使用情况
  - 管理节点上的卷（Volume）和网络
- **工作流程**：
  1. 从 kube-apiserver 获取节点上的 Pod 列表
  2. 通过 CRI 接口与容器运行时通信，创建和管理容器
  3. 定期执行容器健康检查
  4. 向 kube-apiserver 报告节点和 Pod 状态

### kube-proxy
- 运行在每个节点上的网络代理组件，实现 Kubernetes Service 的网络功能
- **核心功能**：
  - 维护网络规则，将 Service 请求转发到后端 Pod
  - 提供简单的负载均衡功能（默认轮询算法）
  - 支持会话亲和性（Session Affinity）
- **工作模式**：
  - **iptables 模式**（默认）：使用 Linux iptables 规则实现网络转发
    - 优点：简单、稳定
    - 缺点：大规模集群性能下降
  - **ipvs 模式**：使用 Linux IPVS 实现网络转发
    - 优点：高性能、支持更多负载均衡算法
    - 缺点：需要内核支持 IPVS
- **Service 类型实现**：
  - ClusterIP 类型：通过 DNAT 实现内部访问
  - NodePort 类型：在每个节点上打开端口转发
  - LoadBalancer 类型：配合云平台负载均衡器

### 容器运行时
- 负责容器的实际运行环境，通过容器运行时接口（CRI）与 kubelet 交互
- **核心功能**：
  - 容器镜像的拉取和管理
  - 容器的创建、运行、停止和删除
  - 容器的网络、存储和资源隔离
  - 容器的生命周期管理
- **主流实现**：
  - **Docker**：最常用的容器运行时，但 Kubernetes 已逐渐减少对 Docker 的直接依赖
  - **containerd**：轻量级容器运行时，专注于核心功能
    - 特点：高性能、低资源消耗、支持多种容器格式
  - **CRI-O**：专门为 Kubernetes 设计的容器运行时
    - 特点：完全兼容 CRI 接口、轻量级设计
  - **rkt**：由 CoreOS 开发的容器运行时（已停止维护）
- **CRI 接口**：
  - 定义了 kubelet 与容器运行时之间的通信标准
  - 包含两个主要服务：
    - RuntimeService：负责容器和 Pod 的生命周期管理
    - ImageService：负责容器镜像的管理

## 常见问题

### 1. Deployment 与 StatefulSet 的区别是什么？
**答案：**
- **Deployment**：用于无状态应用，Pod 无固定身份，可随意替换，支持滚动更新和回滚。
- **StatefulSet**：用于有状态应用，Pod 有固定身份（主机名、网络标识），支持稳定的持久化存储，有序部署和删除。

### 2. kube-apiserver 的主要作用是什么？
**答案：**
- 作为集群的统一入口，提供 RESTful API 接口
- 处理和验证所有客户端请求
- 实现认证、授权和准入控制
- 作为集群的枢纽，所有组件都通过它进行通信
- 与 etcd 交互存储和获取集群状态数据

### 3. Service 的几种类型及其区别？
**答案：**
- **ClusterIP**：默认类型，仅集群内部可访问。
- **NodePort**：在每个节点上暴露端口，可通过节点 IP:端口访问。
- **LoadBalancer**：结合云平台负载均衡器，提供外部访问。
- **ExternalName**：通过 DNS CNAME 指向外部服务。
- **Headless**：无头服务，不分配 ClusterIP，用于 StatefulSet 服务发现。

### 4. Pod 探针有哪些类型，各有什么作用？
**答案：**
- **Liveness Probe**：存活探针，检测容器是否正常运行，失败则重启容器。
- **Readiness Probe**：就绪探针，检测容器是否准备好接收请求，失败则从 Service 中移除。
- **Startup Probe**：启动探针，检测容器是否完成启动，用于慢启动应用，启动期间不执行其他探针。

### 5. 什么是 Namespace，它的作用是什么？
**答案：**
- Namespace 是 Kubernetes 中的资源逻辑隔离机制。
- 作用：
  - 实现多租户环境的资源隔离
  - 提供资源配额（ResourceQuota）和限制范围（LimitRange）
  - 简化资源管理，便于分组和授权
- 默认命名空间：default、kube-system、kube-public、kube-node-lease
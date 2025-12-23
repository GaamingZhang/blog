---
date: 2025-07-01
author: Gaaming Zhang
category:
  - Kubernetes
tag:
  - Kubernetes
  - 已完工
---

# Kubernetes 基本概念与基础组件

## 核心概念

**Pod**
- K8s 最小部署单元，包含一个或多个紧密关联的容器
- 共享网络命名空间（同一 Pod 内容器通过 localhost 通信）
- 共享存储卷（可以挂载相同的 Volume）
- 共享 PID 和 IPC 命名空间（可选）
- 支持探针机制（liveness、readiness、startup）确保容器健康
- 包含 Init Container（初始化容器），在主容器启动前执行

**Service**
- 为 Pod 提供稳定的网络访问入口（固定 IP 和 DNS 名称）
- 通过标签选择器（label selector）动态关联 Pod
- 支持多种类型：
  - ClusterIP：集群内部访问（默认）
  - NodePort：暴露端口到所有节点
  - LoadBalancer：结合云平台负载均衡器
  - ExternalName：通过 DNS CNAME 指向外部服务
  - Headless：无头服务，不分配 ClusterIP，用于 StatefulSet 服务发现

**Deployment**
- 管理无状态应用的部署、扩展和更新
- 通过 ReplicaSet 控制 Pod 副本数量
- 支持滚动更新（RollingUpdate）和蓝绿部署（Blue-Green）
- 支持回滚到之前的版本
- 配置更新策略（maxSurge、maxUnavailable）控制更新过程

**StatefulSet**
- 管理有状态应用（如数据库、缓存）
- 为 Pod 提供稳定的网络标识（主机名、域名）
- 支持稳定的持久化存储
- 有序部署和删除（按序号）

**DaemonSet**
- 在每个节点上运行一个 Pod 副本
- 用于部署系统级服务（如日志收集、监控代理）
- 自动适配节点增减

**Job/CronJob**
- Job：执行一次性任务，完成后停止
- CronJob：定期执行任务（类似 Linux cron）

**Namespace**
- 逻辑隔离资源，实现多租户环境
- 提供资源配额（ResourceQuota）和限制范围（LimitRange）
- 默认命名空间：
  - default：默认用户空间
  - kube-system：系统组件空间
  - kube-public：公共空间，所有用户可读
  - kube-node-lease：节点心跳信息

**ConfigMap/Secret**
- ConfigMap：存储非敏感配置信息，支持环境变量和文件挂载
- Secret：存储敏感信息（密码、证书），默认 Base64 编码
  - Opaque：通用 Secret
  - docker-registry：Docker 镜像仓库认证
  - tls：TLS 证书

**Volume**
- 持久化存储抽象，生命周期与 Pod 无关
- 支持多种后端：本地存储、云存储、分布式存储（如 Ceph、GlusterFS）
- PVC（PersistentVolumeClaim）：动态申请存储
- PV（PersistentVolume）：存储资源池

## 控制平面组件（Master）

**kube-apiserver**
- 集群的统一入口，提供 RESTful API 接口
- 处理和验证所有客户端请求（kubectl、kubelet、控制器等）
- 实现认证（Authentication）、授权（Authorization）和准入控制（Admission Control）
- 作为集群的 "枢纽"，所有组件都通过它进行通信
- 与 etcd 交互存储和获取集群状态数据

**etcd**
- 分布式键值对（KV）存储系统
- 保存集群所有配置和状态数据（如 Pod 定义、Service、ReplicaSet 等）
- 提供强一致性保证，是集群的 "唯一事实来源"
- 支持高可用部署（通常为奇数个节点，如 3、5、7 个）

**kube-scheduler**
- 负责 Pod 的调度决策
- 监听待调度的 Pod，根据以下因素选择合适节点：
  - 资源需求（CPU、内存、GPU 等）
  - 节点资源可用性
  - 亲和力（Affinity）和反亲和力（Anti-Affinity）规则
  - 节点污点（Taint）和容忍度（Toleration）
  - 拓扑约束（Topology Constraints）
  - 其他调度策略

**kube-controller-manager**
- 运行各种控制器进程的组件
- 确保集群实际状态与期望状态一致
- 包含的主要控制器：
  - Node Controller：管理节点生命周期
  - Replication Controller：维持 Pod 副本数量
  - Deployment Controller：管理 Deployment 资源
  - StatefulSet Controller：管理有状态应用
  - Endpoints Controller：维护 Service 与 Pod 的关联
  - Service Account & Token Controller：管理服务账号和令牌

**cloud-controller-manager**
- 与云平台交互的控制器集合
- 实现与云服务商的特定功能集成
- 包括的主要控制器：
  - Node Controller：与云平台交互管理节点
  - Route Controller：配置网络路由
  - Service Controller：管理云平台负载均衡器
  - Volume Controller：管理云存储卷

## Node 组件

**kubelet**
- 运行在每个节点上的核心代理组件
- 负责管理 Pod 生命周期的各个阶段：创建、启动、监控、停止和销毁
- 与 kube-apiserver 通信，接收 Pod 规范并确保节点上运行的容器与规范一致
- 执行容器健康检查（通过探针机制）
- 向 kube-apiserver 报告节点和 Pod 的状态、资源使用情况
- 管理节点上的卷（Volume）和网络

**kube-proxy**
- 运行在每个节点上的网络代理组件
- 实现 Kubernetes Service 的网络功能
- 维护网络规则（使用 iptables 或 ipvs 模式）：
  - ClusterIP 类型：通过 DNAT 实现内部访问
  - NodePort 类型：在每个节点上打开端口转发
  - LoadBalancer 类型：配合云平台负载均衡器
- 提供简单的负载均衡功能（轮询算法）
- 支持会话亲和性（Session Affinity）

**容器运行时**
- 负责容器的实际运行环境
- 通过容器运行时接口（CRI）与 kubelet 交互
- 支持多种实现：
  - Docker：最常用的容器运行时（但 Kubernetes 已逐渐减少对 Docker 的直接依赖）
  - containerd：轻量级容器运行时，专注于核心功能
  - CRI-O：专门为 Kubernetes 设计的容器运行时
  - rkt：由 CoreOS 开发的容器运行时
- 负责容器镜像的拉取、容器的创建、运行、停止和删除
- 管理容器的网络、存储和资源隔离

## 高频面试题及答案

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

### 6. Kubernetes 中的控制器有哪些，它们的作用是什么？
**答案：**
- **Deployment Controller**：管理无状态应用的部署、扩展和更新。
- **StatefulSet Controller**：管理有状态应用的部署和扩展。
- **DaemonSet Controller**：确保每个节点运行一个 Pod 副本。
- **Job Controller**：管理一次性任务。
- **CronJob Controller**：管理定期执行的任务。
- **Node Controller**：管理节点生命周期。
- **Endpoints Controller**：维护 Service 与 Pod 的关联。
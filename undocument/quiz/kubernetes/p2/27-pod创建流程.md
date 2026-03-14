---
date: 2026-03-12
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Pod
  - 工作流程
---

# Kubernetes Pod创建流程深度解析

## 引言

作为Kubernetes中最小的部署单元，Pod的创建过程涉及多个组件的协同工作。深入理解Pod创建流程不仅有助于排查部署过程中的问题，更能帮助开发者和运维人员优化集群性能、设计更可靠的应用架构。当你在生产环境中遇到Pod长时间处于Pending状态、调度失败或容器启动异常时，掌握完整的创建流程将使你能够快速定位问题根源。

本文将从用户提交请求开始，逐步剖析Pod从无到有的完整生命周期，揭示每个阶段背后的技术原理和组件职责。

## Pod创建完整流程

### 一、用户提交请求阶段

Pod的创建始于用户通过kubectl命令行工具或直接调用Kubernetes API发起的创建请求。

**kubectl提交方式**：

```bash
kubectl apply -f pod.yaml
# 或
kubectl run nginx --image=nginx:latest
```

**API直接调用方式**：

用户也可以通过编程方式直接调用Kubernetes API Server的RESTful接口：

```bash
curl -X POST https://api-server:6443/api/v1/namespaces/default/pods \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d @pod.json
```

无论采用哪种方式，最终都会生成一个HTTP POST请求，携带Pod的完整定义（YAML或JSON格式）发送到API Server。

### 二、认证授权准入控制阶段

API Server接收到请求后，会依次经过三个关键的安全控制阶段：认证（Authentication）、授权（Authorization）和准入控制（Admission Control）。

#### 2.1 认证（Authentication）

认证阶段验证请求发起者的身份。Kubernetes支持多种认证方式：

- **客户端证书认证**：通过TLS客户端证书验证身份
- **Bearer Token认证**：使用ServiceAccount Token或OIDC Token
- **基本认证**：用户名密码方式（不推荐生产使用）
- **Webhook Token认证**：通过外部服务验证Token

认证成功后，请求会被标记上用户身份信息（用户名、组、UID等），用于后续的授权检查。

#### 2.2 授权（Authorization）

授权阶段确定认证用户是否有权限执行请求的操作。Kubernetes主要使用RBAC（Role-Based Access Control）进行授权：

- **Role/ClusterRole**：定义权限规则
- **RoleBinding/ClusterRoleBinding**：将角色绑定到用户或ServiceAccount

授权引擎会检查用户是否有权限对Pod资源执行create操作。如果授权失败，API Server会返回403 Forbidden错误。

#### 2.3 准入控制（Admission Control）

准入控制是请求被持久化到etcd之前的最后一道关卡。准入控制器可以修改请求对象或拒绝请求。常见的准入控制器包括：

- **NamespaceLifecycle**：确保请求的namespace存在且处于活跃状态
- **LimitRanger**：确保资源请求符合LimitRange约束
- **ServiceAccount**：为Pod自动挂载ServiceAccount Token
- **DefaultStorageClass**：为PVC设置默认StorageClass
- **ResourceQuota**：确保资源请求符合ResourceQuota限制
- **PodSecurityPolicy**：检查Pod是否符合安全策略（已废弃，使用PodSecurity Admission替代）
- **ValidatingAdmissionWebhook**：通过Webhook进行自定义验证
- **MutatingAdmissionWebhook**：通过Webhook进行自定义修改

准入控制分为两个阶段：首先是Mutating阶段（修改请求对象），然后是Validating阶段（验证请求对象）。这种设计确保了所有修改在验证之前完成。

### 三、写入etcd阶段

通过所有检查后，API Server会将Pod对象序列化为JSON格式，并写入etcd数据库。etcd是Kubernetes的持久化存储层，采用Raft协议保证数据一致性。

**写入过程的关键步骤**：

1. **对象版本管理**：为Pod分配初始ResourceVersion（用于乐观并发控制）
2. **数据序列化**：将Pod对象转换为JSON格式
3. **持久化存储**：通过etcd的Put API写入数据
4. **事件通知**：etcd的Watch机制会通知所有订阅者

此时，Pod对象已经在etcd中持久化，状态为**Pending**，等待调度器分配节点。

### 四、Scheduler调度阶段

Kubernetes Scheduler是负责Pod调度的核心组件，它通过Watch机制监听API Server，当发现未调度的Pod（spec.nodeName为空）时，会启动调度流程。

#### 4.1 调度流程概览

调度过程分为三个主要阶段：**预选（Predicates）**、**优选（Priorities）**和**绑定（Binding）**。

#### 4.2 预选阶段（Filtering）

预选阶段从所有节点中筛选出符合Pod运行要求的候选节点。预选策略包括：

- **PodFitsResources**：节点是否有足够的CPU、内存等资源
- **PodFitsHostPorts**：节点是否有可用的HostPort
- **PodMatchNodeSelector**：节点是否匹配Pod的NodeSelector
- **PodToleratesNodeTaints**：Pod是否容忍节点的Taints
- **CheckNodeCondition**：节点是否处于Ready状态
- **CheckNodeUnschedulable**：节点是否被标记为不可调度
- **PodFitsHost**：节点是否匹配Pod的NodeName字段

预选阶段采用并行处理，大幅提升大规模集群的调度效率。如果一个节点不满足任何预选条件，它将被排除在候选列表之外。

#### 4.3 优选阶段（Scoring）

优选阶段为每个候选节点打分，选择最优节点。优选策略包括：

- **LeastRequestedPriority**：优先选择资源利用率低的节点
- **BalancedResourceAllocation**：优先选择CPU和内存使用均衡的节点
- **NodeAffinityPriority**：优先选择匹配NodeAffinity的节点
- **InterPodAffinityPriority**：优先选择满足Pod亲和性的节点
- **TaintTolerationPriority**：优先选择Pod容忍Taint的节点
- **ImageLocalityPriority**：优先选择已有镜像的节点

每个优选策略会给出一个0-10的分数，最终分数是所有策略分数的加权和。调度器会选择分数最高的节点。

#### 4.4 绑定阶段（Binding）

选定节点后，Scheduler会向API Server发送绑定请求，将Pod的spec.nodeName字段设置为选定的节点名称。这个过程通过Binding对象完成：

```yaml
apiVersion: v1
kind: Binding
metadata:
  name: nginx-pod
target:
  apiVersion: v1
  kind: Node
  name: node-1
```

API Server接收到绑定请求后，会更新Pod对象，此时Pod的状态仍然为Pending，但已经分配了节点。

### 五、kubelet创建容器阶段

kubelet是运行在每个节点上的代理，它通过Watch机制监听API Server，当发现有Pod分配到自己管理的节点时，会启动Pod的生命周期管理。

#### 5.1 Pod创建流程

kubelet创建Pod的过程包含多个步骤：

**1. Pod同步检查**

kubelet首先检查Pod是否已经存在，避免重复创建。通过Pod的UID和namespace进行唯一性判断。

**2. 沙箱容器创建**

kubelet调用容器运行时接口（CRI）创建Pod沙箱（Pause容器）。Pause容器是Pod的根容器，它持有Pod的网络命名空间和IPC命名空间。

```
Pod网络命名空间
    └── Pause容器 (PID 1)
        ├── 业务容器1
        ├── 业务容器2
        └── ...
```

Pause容器的作用：
- 维持Pod的网络命名空间
- 作为Pod内其他容器的父进程
- 回收僵尸进程

**3. 拉取镜像**

kubelet调用容器运行时拉取Pod中定义的所有容器镜像。镜像拉取策略包括：
- **Always**：每次都拉取最新镜像
- **IfNotPresent**：本地不存在时才拉取（默认）
- **Never**：从不拉取，只使用本地镜像

**4. 创建容器**

kubelet为每个容器创建容器对象，包括：
- 设置容器镜像、命令、参数
- 配置资源限制（CPU、内存）
- 设置环境变量和ConfigMap/Secret挂载
- 配置健康检查探针
- 设置安全上下文（SecurityContext）

**5. 启动容器**

kubelet调用容器运行时启动容器。容器启动后，kubelet会持续监控容器状态。

#### 5.2 容器运行时接口（CRI）

kubelet通过CRI与容器运行时交互。CRI定义了一组gRPC接口，包括：

- **RuntimeService**：管理容器和Pod沙箱
- **ImageService**：管理镜像

主流的容器运行时包括：
- **containerd**：CNCF毕业项目，性能优异
- **CRI-O**：专为Kubernetes设计的轻量级运行时
- **Docker Engine**（通过dockershim，已废弃）

### 六、网络配置阶段

网络配置是Pod创建过程中的关键环节，它确保Pod能够与其他Pod、Service和外部网络通信。

#### 6.1 网络模型

Kubernetes要求网络实现满足以下要求：
- 所有Pod可以在不使用NAT的情况下相互通信
- 所有Node可以在不使用NAT的情况下与所有Pod通信
- Pod看到的自己IP与其他Pod看到的它的IP相同

#### 6.2 CNI插件工作流程

Kubernetes使用CNI（Container Network Interface）插件配置Pod网络。CNI插件的工作流程：

**1. kubelet调用CNI插件**

kubelet在创建Pod沙箱后，会调用CNI插件配置网络。CNI配置文件通常位于`/etc/cni/net.d/`目录。

**2. CNI插件创建网络接口**

CNI插件会在Pod的网络命名空间中创建虚拟网络接口（veth pair），一端在Pod内，另一端连接到宿主机的网桥或路由表。

**3. 分配IP地址**

CNI插件调用IPAM（IP Address Management）插件为Pod分配IP地址。IPAM插件维护IP地址池，确保IP地址不冲突。

**4. 配置路由和规则**

CNI插件配置宿主机的路由表和iptables规则，确保Pod的网络流量正确转发。

#### 6.3 常见CNI插件

- **Flannel**：简单易用，支持多种后端（VXLAN、Host-GW等）
- **Calico**：性能优异，支持网络策略和BGP路由
- **Weave Net**：自动配置，支持多主机网络
- **Cilium**：基于eBPF，支持高级网络功能和可观测性

### 七、状态更新与就绪阶段

容器启动后，kubelet会持续监控Pod状态，并向API Server报告。

#### 7.1 状态上报

kubelet通过以下方式监控容器状态：
- **容器运行时状态**：通过CRI查询容器状态
- **存活探针（Liveness Probe）**：检测容器是否存活
- **就绪探针（Readiness Probe）**：检测容器是否就绪

kubelet定期向API Server发送Pod状态更新，包括：
- Pod Phase（Pending、Running、Succeeded、Failed、Unknown）
- 容器状态（Waiting、Running、Terminated）
- Pod IP地址
- Pod Conditions（PodScheduled、Initialized、Ready、ContainersReady）

#### 7.2 就绪检查

当所有容器都启动成功且就绪探针检查通过时，kubelet会将Pod的Ready Condition设置为True。此时，Pod会被端点控制器添加到对应Service的Endpoints中，开始接收流量。

## 流程图示

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户提交请求                               │
│                  (kubectl apply / API调用)                       │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        API Server                                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              认证 (Authentication)                        │  │
│  │         验证用户身份（证书/Token/Basic）                   │  │
│  └────────────────────────┬─────────────────────────────────┘  │
│                           ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              授权 (Authorization)                         │  │
│  │           检查用户权限（RBAC/ABAC）                        │  │
│  └────────────────────────┬─────────────────────────────────┘  │
│                           ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │          准入控制 (Admission Control)                     │  │
│  │   Mutating Webhook → Validating Webhook                  │  │
│  └────────────────────────┬─────────────────────────────────┘  │
└───────────────────────────┼─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                          etcd                                    │
│              持久化Pod对象（状态：Pending）                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Scheduler                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │            Watch未调度的Pod                                │  │
│  └────────────────────────┬─────────────────────────────────┘  │
│                           ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │         预选 (Predicates/Filtering)                       │  │
│  │      筛选符合要求的节点（资源/端口/亲和性）                  │  │
│  └────────────────────────┬─────────────────────────────────┘  │
│                           ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │         优选 (Priorities/Scoring)                         │  │
│  │         为候选节点打分，选择最优节点                         │  │
│  └────────────────────────┬─────────────────────────────────┘  │
│                           ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │            绑定 (Binding)                                 │  │
│  │      更新Pod的spec.nodeName，写入etcd                      │  │
│  └────────────────────────┬─────────────────────────────────┘  │
└───────────────────────────┼─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                          kubelet                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │        Watch分配到本节点的Pod                              │  │
│  └────────────────────────┬─────────────────────────────────┘  │
│                           ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │      创建Pod沙箱（Pause容器）                              │  │
│  └────────────────────────┬─────────────────────────────────┘  │
│                           ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │          拉取容器镜像                                     │  │
│  └────────────────────────┬─────────────────────────────────┘  │
│                           ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │      创建并启动容器（通过CRI）                             │  │
│  └────────────────────────┬─────────────────────────────────┘  │
└───────────────────────────┼─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      CNI网络插件                                 │
│         配置Pod网络（IP分配、路由、iptables）                     │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      kubelet持续监控                             │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │        执行健康检查（Liveness/Readiness）                  │  │
│  └────────────────────────┬─────────────────────────────────┘  │
│                           ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │      更新Pod状态到API Server                              │  │
│  │          (Phase: Running, Ready: True)                    │  │
│  └────────────────────────┬─────────────────────────────────┘  │
└───────────────────────────┼─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Pod创建完成                                  │
│              容器运行中，开始接收流量                             │
└─────────────────────────────────────────────────────────────────┘
```

## 各组件职责汇总

| 组件 | 职责 | 关键功能 |
|------|------|----------|
| **kubectl** | 命令行工具 | 将用户请求转换为API调用，提供友好的交互界面 |
| **API Server** | 集群入口 | 认证授权、准入控制、RESTful API、数据验证、etcd代理 |
| **etcd** | 持久化存储 | 存储集群状态数据，提供Watch机制，保证数据一致性 |
| **Scheduler** | 调度决策 | 预选节点、优选打分、绑定节点、支持自定义调度器 |
| **kubelet** | 节点代理 | Pod生命周期管理、容器创建、健康检查、状态上报、资源监控 |
| **容器运行时** | 容器管理 | 镜像管理、容器创建/启动/停止、通过CRI与kubelet交互 |
| **CNI插件** | 网络配置 | 分配IP地址、创建网络接口、配置路由和iptables规则 |
| **Controller Manager** | 控制循环 | 维护期望状态，包括端点控制器、副本控制器等 |

## 常见问题与最佳实践

### 常见问题

#### 1. Pod长时间处于Pending状态

**原因分析**：
- 资源不足：集群没有足够的CPU、内存资源
- 节点选择器不匹配：Pod的NodeSelector或NodeAffinity无法匹配任何节点
- Taint/Toleration不匹配：节点有Taint但Pod没有对应的Toleration
- PVC未绑定：Pod依赖的PVC没有可用的PV
- 调度器异常：Scheduler未运行或配置错误

**排查方法**：
```bash
# 查看Pod事件
kubectl describe pod <pod-name> -n <namespace>

# 查看节点资源
kubectl describe nodes

# 查看调度器日志
kubectl logs -n kube-system <scheduler-pod>
```

#### 2. 容器启动失败

**原因分析**：
- 镜像不存在或无法拉取
- 容器命令或参数错误
- 资源限制过小（OOMKilled）
- 健康检查配置不当
- 挂载卷失败

**排查方法**：
```bash
# 查看容器日志
kubectl logs <pod-name> -n <namespace>

# 查看容器状态
kubectl get pod <pod-name> -o yaml

# 进入容器调试
kubectl exec -it <pod-name> -- /bin/sh
```

#### 3. Pod网络不通

**原因分析**：
- CNI插件未正确安装或配置
- 网络策略限制
- DNS解析失败
- 节点防火墙规则冲突

**排查方法**：
```bash
# 测试Pod间网络
kubectl exec -it <pod-name> -- ping <target-pod-ip>

# 测试DNS解析
kubectl exec -it <pod-name> -- nslookup kubernetes

# 查看网络策略
kubectl get networkpolicy -A
```

#### 4. 镜像拉取超时

**原因分析**：
- 镜像仓库网络不通
- 镜像仓库认证失败
- 镜像过大，拉取时间过长
- 节点磁盘空间不足

**解决方案**：
- 使用镜像加速器或私有镜像仓库
- 配置imagePullSecrets
- 优化镜像大小（多阶段构建）
- 清理节点磁盘空间

#### 5. 就绪探针检查失败

**原因分析**：
- 应用启动时间过长，探针初始延迟过短
- 探针检查路径或端口错误
- 应用依赖服务未就绪
- 探针超时时间过短

**最佳实践**：
```yaml
readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30  # 给应用足够的启动时间
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

### 最佳实践

#### 1. 资源配置优化

```yaml
resources:
  requests:
    cpu: 100m      # 保证最小资源
    memory: 128Mi
  limits:
    cpu: 500m      # 限制最大资源
    memory: 512Mi
```

- 合理设置requests和limits，避免资源浪费或OOM
- 使用LimitRange和ResourceQuota进行资源管理
- 监控资源使用情况，动态调整配置

#### 2. 健康检查配置

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

- 区分存活探针和就绪探针的用途
- 设置合理的初始延迟和检查间隔
- 探针路径应轻量快速，避免复杂逻辑

#### 3. 调度优化

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: disktype
          operator: In
          values:
          - ssd
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app
            operator: In
            values:
            - web
        topologyKey: kubernetes.io/hostname
```

- 使用节点亲和性将Pod调度到合适的节点
- 使用Pod反亲和性分散部署，提高可用性
- 合理使用Taint和Toleration实现节点隔离

#### 4. 安全配置

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  readOnlyRootFilesystem: true
  capabilities:
    drop:
    - ALL
```

- 限制容器权限，遵循最小权限原则
- 使用Pod Security Standards（Restricted、Baseline、Privileged）
- 避免使用特权容器

#### 5. 镜像管理

```yaml
image: myregistry.example.com/myapp:v1.2.3
imagePullPolicy: IfNotPresent
```

- 使用私有镜像仓库，配置imagePullSecrets
- 明确指定镜像版本标签，避免使用latest
- 优化镜像大小，使用多阶段构建

## 面试回答

**面试官问：请描述一下Kubernetes中创建Pod的完整流程。**

**回答**：

Kubernetes中创建Pod的流程可以分为六个关键阶段。首先，用户通过kubectl或API提交Pod创建请求，请求到达API Server。API Server会依次进行认证（验证用户身份）、授权（检查用户权限）和准入控制（执行各种策略检查和对象修改），通过后将Pod对象持久化到etcd，此时Pod状态为Pending。接着，Scheduler通过Watch机制监听到未调度的Pod，执行预选筛选符合要求的节点，再通过优选打分选择最优节点，最后将Pod绑定到该节点并更新到etcd。然后，目标节点上的kubelet监听到分配给自己的Pod，开始创建Pod沙箱（Pause容器）、拉取镜像、创建并启动容器。同时，CNI插件为Pod配置网络，包括分配IP地址、创建网络接口和配置路由。最后，kubelet执行健康检查，当容器就绪后将Pod状态更新为Running，整个创建流程完成。这个流程体现了Kubernetes声明式API和控制器模式的设计理念，各组件通过Watch机制协同工作，确保集群最终达到期望状态。

## 总结

Pod创建流程是Kubernetes核心机制的重要体现，涉及API Server、etcd、Scheduler、kubelet、容器运行时和CNI插件等多个组件的紧密协作。深入理解这一流程，不仅有助于排查部署问题，更能帮助开发者设计更可靠的应用架构。

从用户提交请求到容器运行，每个阶段都有其特定的职责和挑战。认证授权准入控制保证了集群的安全性，调度器实现了资源的合理分配，kubelet负责容器的生命周期管理，CNI插件确保网络连通性。掌握这些组件的工作原理和最佳实践，将使你在Kubernetes的使用和运维中游刃有余。

随着云原生技术的不断发展，Pod创建流程也在持续优化，例如调度器的性能改进、CRI和CNI接口的标准化、安全机制的增强等。持续关注这些变化，将帮助你更好地利用Kubernetes构建现代化的云原生应用。

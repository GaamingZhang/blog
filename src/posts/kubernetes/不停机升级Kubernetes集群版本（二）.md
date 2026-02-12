---
date: 2026-02-10
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# 不停机升级Kubernetes集群版本（二）：工作节点升级与生产实践

上一篇我们深入剖析了集群升级的底层原理：版本偏差策略的约束、etcd 的 Raft 协议如何保证数据一致性、API Server 的滚动升级机制、以及 Leader Election 如何保证控制器在升级过程中的高可用。控制平面升级完成后，集群的"大脑"已经运行在新版本之上，但所有工作负载还跑在旧版本的工作节点上。

接下来要面对的，是不停机升级中最贴近业务、风险最高的环节——工作节点升级。工作节点承载着所有业务 Pod，如何在将一个节点上的所有 Pod 全部迁走的同时，保证业务流量不中断？这背后有一套完整的机制在协同运转，理解它，是掌握生产级升级操作的前提。

## 工作节点升级的底层机制

### 节点的心跳与状态感知

在讨论 drain 之前，先理解 kubelet 与控制平面之间的基本通信模式，这有助于理解整个升级过程中信号的传播方式。

kubelet 通过两个机制向控制平面汇报节点状态：

**Node Status 心跳**：kubelet 每隔一定时间（默认 10 秒）向 API Server 更新 Node 对象的 `status.conditions`，报告节点健康状况（MemoryPressure、DiskPressure、Ready 等）。如果 Node Controller 在 `node-monitor-grace-period`（默认 40 秒）内没有收到心跳，会将节点标记为 `NotReady`。

**Lease 心跳**：从 Kubernetes 1.14 开始，kubelet 还会更新 `kube-node-lease` 命名空间下的 Lease 对象（每 10 秒一次）。Lease 是一个轻量级的心跳机制，只更新一个时间戳字段，比更新完整的 Node Status 开销小得多。Node Controller 优先通过 Lease 来判断节点是否存活。

了解这个机制的意义在于：当我们在升级过程中重启 kubelet 时，会有短暂时间（通常数十秒）节点的 Lease 无法更新。但只要 kubelet 在 `node-monitor-grace-period` 超时前重启成功，Node Controller 不会将节点标记为 `NotReady`，节点上的 Pod 也不会被驱逐。这就是为什么"kubelet 重启不会导致 Pod 退出"——只要重启速度足够快，控制平面感知不到节点离线。

### 节点排空（Drain）的完整流程

执行 `kubectl drain <node>` 时，感觉像是一个简单的命令，但底层的操作序列并不简单。它实际上分为两个阶段：先 Cordon，再逐步驱逐 Pod。

**Cordon：切断新 Pod 的流入**

Cordon 的本质是在 Node 对象上设置一个字段：`node.spec.unschedulable = true`。

调度器（kube-scheduler）在为每个待调度的 Pod 选择目标节点时，会执行一系列过滤器（Filter）和评分器（Scorer）。其中有一个专门的过滤器 `NodeUnschedulable`，它会跳过所有 `unschedulable=true` 的节点。Cordon 的生效是立即的，因为 kube-scheduler 通过 List/Watch 机制监听 Node 对象的变化，一旦 Node 对象被更新，调度器会立即感知到并更新自己内部的节点缓存。

这个设计非常重要：Cordon 之后节点上仍然有业务 Pod 在运行，流量也还在正常转发。Cordon 只是切断了新 Pod 流入这个节点的通道，给了我们一个"软着陆"的过渡窗口——当前节点上的工作负载会继续服务，直到被显式驱逐。

**Eviction API：为什么不直接删除 Pod**

Cordon 完成后，drain 开始驱逐节点上的 Pod。这里有一个关键的设计决策：drain 不会直接调用 Delete Pod API，而是通过 **Eviction API** 来发起驱逐。

直接删除 Pod 是一个强制操作，没有任何保护机制，类似于直接执行 `kubectl delete pod`。Eviction API 是一个语义不同的操作，它是一个"请求"，在执行驱逐之前，API Server 会先检查 **PodDisruptionBudget（PDB）约束**。如果驱逐某个 Pod 会违反 PDB，API Server 会返回 `429 Too Many Requests`，drain 会等待并重试，而不是强行删除 Pod。这个机制从根本上保护了应用的可用性。

从实现角度看，Eviction API 是 Pod 子资源（`/eviction`），调用它会创建一个 `Eviction` 对象，API Server 中的 Eviction 准入控制器会在这里检查 PDB，然后再真正删除 Pod。

Drain 还需要处理几类特殊的 Pod：

**DaemonSet Pod**：DaemonSet 控制器的设计目的就是让每个节点都运行一个 Pod 副本。驱逐 DaemonSet Pod 没有意义，因为控制器会立即在同一个节点重建它。Drain 默认会忽略 DaemonSet 管理的 Pod（通过 `--ignore-daemonsets` 标志控制），直接跳过这些 Pod。但如果不传该标志，drain 会报错并拒绝执行，以防误操作。

**使用 emptyDir 的 Pod**：emptyDir 是节点本地临时存储，Pod 离开节点后数据会永久丢失。Drain 默认不驱逐这类 Pod，需要显式传入 `--delete-emptydir-data` 确认可以接受数据丢失。

**孤儿 Pod（没有控制器管理的 Pod）**：这类 Pod 被驱逐后不会在其他节点重建，数据和状态会直接消失。需要显式传入 `--force` 标志才会驱逐，以防误操作删除不可恢复的单点进程。

### Pod 驱逐的详细时序

一个 Pod 从被发起驱逐到在新节点稳定运行，整个过程涉及多个控制器的协作。这个时序值得细看：

```
drain 进程                    API Server                  Pod 所在节点       新节点
     │                           │                             │               │
     │── Eviction 请求 ──────────>│                             │               │
     │                           │ 检查 PDB 约束                │               │
     │                           │ PDB 满足，允许驱逐            │               │
     │<─── 200 OK ───────────────│                             │               │
     │                           │─ Pod 标记为 Terminating ───>│               │
     │                           │                             │ 执行 preStop   │
     │                           │                             │ 等待 preStop  │
     │                           │                             │ 发送 SIGTERM  │
     │                           │                             │ 等待进程退出  │
     │                           │                             │ Pod 进程退出  │
     │                           │<── kubelet 报告 Pod 消失 ───│               │
     │                           │                             │               │
     │                           │（ReplicaSet 控制器发现副本不足）              │
     │                           │─────────── 创建新 Pod ─────────────────────>│
     │                           │                             │               │ 镜像拉取
     │                           │                             │               │ 容器启动
     │                           │                             │               │ 就绪探针通过
     │                           │<──────────── Pod Ready ─────────────────────│
     │                           │（Endpoint Controller 更新 Endpoints）        │
     │                           │（kube-proxy 更新 iptables/ipvs 规则）        │
     │                           │（新 Pod 开始接收流量）                        │
```

这里有一个值得关注的时序问题：Endpoint 的更新和 Pod 的 Terminating 是两条并行的路径，它们的传播速度不一样。一个 Pod 进入 Terminating 状态时，Endpoint Controller 会将其从 Service 的 Endpoints 中摘除，但这个摘除信息还需要经过 kube-proxy 的 Watch → iptables/ipvs 规则更新才能在每个节点上生效，这个链路通常需要数秒到十几秒。

如果应用的 preStop hook 有足够的等待时间（通常建议 5-15 秒的 sleep），这个传播延迟可以被消化。这就是为什么即使应用本身实现了完整的优雅关闭，也仍然需要 preStop hook——不是为了等待应用关闭，而是为了等待流量路由规则的传播。

### Uncordon 的原理与调度行为

升级完成后，执行 `kubectl uncordon <node>` 将节点重新加入调度。这个操作将 `node.spec.unschedulable` 设回 `false`，调度器立即感知到变化，新的 Pod 可以被调度到这里。

有一个容易被误解的行为：**Uncordon 不会触发任何 Pod 的迁移**。已经在其他节点稳定运行的 Pod 不会自动迁回来。Kubernetes 的调度器是"单向"的——它只负责为"待调度的 Pod"选择节点，不会主动重新平衡集群中的 Pod 分布。

这意味着，节点升级完成 Uncordon 之后，这个节点会暂时处于"空节点"状态，只有后续新创建的 Pod 或者某些 Pod 被重新调度时，才会有 Pod 被分配到这里。如果集群中各节点的 Pod 密度明显不均衡，可以考虑使用 Descheduler（一个独立工具）来主动触发 Pod 的重新分布。

### kubelet 重启与 Pod 的关系

一个重要但容易被误解的点：**kubelet 重启不会导致节点上正在运行的 Pod 退出**。

这是因为容器进程并不是 kubelet 进程的子进程。kubelet 是通过 CRI（Container Runtime Interface）与容器运行时（如 containerd）交互，容器的生命周期由容器运行时管理，而不是由 kubelet 直接持有。当 kubelet 重启时，容器运行时和容器进程都在继续运行。kubelet 重启后，会通过 CRI 重新 list 当前节点上所有正在运行的容器，重建自己内部的状态缓存，然后继续正常的调谐工作。

这个设计对于升级操作非常关键：工作节点升级的核心操作是停止 kubelet → 升级二进制 → 启动新版本 kubelet，整个过程中节点上的 Pod 一直在运行（只要不主动驱逐它们）。这也是 drain 需要在升级前完成的原因：通过 drain 将 Pod 主动迁走，而不是依赖 kubelet 停止时的副作用。

kubelet 停止和重启期间，节点对控制平面来说处于心跳中断的状态。如果重启时间超过 `node-monitor-grace-period`（默认 40 秒），Node Controller 会将节点标记为 `NotReady`，但不会立即驱逐 Pod。要触发 Pod 的自动迁移（非自愿驱逐），需要节点持续 `NotReady` 超过 `pod-eviction-timeout`（默认 5 分钟）。这给了 kubelet 足够宽裕的重启窗口——正常情况下 kubelet 重启只需要数秒到数十秒。

## PodDisruptionBudget 在集群升级中的关键作用

PDB 是 Kubernetes 在集群运维与应用高可用之间建立的契约。理解它的工作原理，是实现不停机升级的核心。

### 自愿中断与非自愿中断

Kubernetes 将 Pod 的中断分为两类：

**自愿中断（Voluntary Disruptions）**：由管理员或控制器主动发起的中断，包括：节点维护排空、集群版本升级、滚动发布、HPA 缩容、手动删除 Pod。这类中断是可以被预测和控制的。

**非自愿中断（Involuntary Disruptions）**：由不可控的外部事件引起，包括：节点硬件故障、内核 OOM 杀进程、虚拟机被宿主机驱逐、节点上的进程崩溃。这类中断无法完全避免。

PDB 只对**自愿中断**生效。当 drain 通过 Eviction API 发起驱逐时，这是自愿中断，PDB 的约束会介入。但如果节点突然宕机，Pod 直接消失，这是非自愿中断，PDB 无法阻止。

这个设计意味着 PDB 是一个"最大努力保护"机制，而不是绝对的可用性保证。对于非自愿中断，你仍然需要通过多副本、跨故障域（不同的可用区或节点池）的部署来应对。

### Disruption Controller 与 PDB 状态维护

PDB 不只是一个静态的配置，Kubernetes 有专门的 **Disruption Controller** 持续计算和维护每个 PDB 对象的 `status` 字段，包括：

- `currentHealthy`：当前健康（Ready 状态）的 Pod 数
- `desiredHealthy`：期望最少健康的 Pod 数（由 minAvailable 或 maxUnavailable 换算得来）
- `disruptionsAllowed`：当前允许中断的 Pod 数（`currentHealthy - desiredHealthy`）
- `disruptedPods`：正在被中断（处于 Terminating）的 Pod 列表

当 Eviction API 收到驱逐请求时，API Server 中的 Eviction 准入控制器读取目标 PDB 的 `status.disruptionsAllowed` 字段：如果值大于 0，允许驱逐并原子性地递减 `disruptionsAllowed`；如果值为 0，拒绝驱逐并返回 429。

**一个具体例子**：假设一个应用有 3 个 Pod，配置了 `minAvailable: 2`。

```
初始状态：3 个 Pod 全部 Ready
  currentHealthy = 3, desiredHealthy = 2, disruptionsAllowed = 1

节点 A 排空，驱逐 Pod-A：
  → disruptionsAllowed = 1 > 0，允许驱逐
  → Pod-A 进入 Terminating，disruptionsAllowed 递减为 0
  → currentHealthy 降为 2

节点 B 排空，同时尝试驱逐 Pod-B：
  → disruptionsAllowed = 0，拒绝驱逐，返回 429
  → drain 等待，每隔几秒重试...

Disruption Controller 监测到 Pod-A 被成功删除，新 Pod-A' 在其他节点启动 Ready：
  → currentHealthy 恢复为 3，disruptionsAllowed 恢复为 1
  → drain 重试驱逐 Pod-B，成功
```

这个机制保证了在任何时刻，集群中至少有 2 个 Pod 在正常服务。但这也说明了一个问题：如果同时对多个节点执行 drain，PDB 会自动序列化各节点的驱逐操作，升级时间会相应延长。这是安全性和速度之间必要的权衡。

### minAvailable 与 maxUnavailable 的选择

在配置 PDB 时，需要在 `minAvailable` 和 `maxUnavailable` 之间做选择。它们在数学上是等价的（`minAvailable + maxUnavailable = 期望副本数`），但在使用上有不同的适用场景：

`minAvailable` 更直觉——直接表达"我需要至少 N 个 Pod 健康"。当你的首要关注点是绝对可用性（比如对外提供 API 的服务，必须保证最少 2 个实例在线），用 `minAvailable` 是直接的。它也支持百分比，如 `minAvailable: 50%`。

`maxUnavailable` 则以相对视角来表达——"最多允许 N 个 Pod 不可用"。当你的首要关注点是升级速度（希望尽快完成迁移），或者副本数是动态变化的（HPA 管理），用 `maxUnavailable` 更灵活。因为 `maxUnavailable` 是基于当前期望副本数的百分比或绝对数，HPA 扩容时 PDB 允许中断的数量也会随之增加，不需要手动调整 PDB。

需要注意的是：不要将 `minAvailable` 设置为 `100%` 或等于总副本数的绝对值，这会导致任何驱逐都被拒绝，节点排空永远无法完成。

### PDB 阻塞 Drain 的场景分析

Drain 默认情况下会无限等待，直到 PDB 允许驱逐为止。在生产环境中，以下几种情况会导致 drain"卡住"：

**PDB 配置过于严格**：`minAvailable` 等于总副本数，`disruptionsAllowed` 永远为 0，任何驱逐都被拒绝。正确的配置应该让 `minAvailable` 小于副本数，留有驱逐容忍度。

**应用副本不足且全部健康不足**：某些 Pod 因为其他原因处于非 Ready 状态（如探针失败），导致 `currentHealthy` 低于 `desiredHealthy`，`disruptionsAllowed` 为负数或零。此时 drain 会一直等待，直到这些 Pod 恢复健康。

**新 Pod 无法就绪**：驱逐发出后，新 Pod 的调度、镜像拉取、启动都需要时间。如果新 Pod 因为某种原因（如资源不足、镜像拉取失败）无法变为 Ready，`currentHealthy` 无法恢复，下一个 Pod 的驱逐会被无限期阻塞。

可以通过 `--timeout` 参数为 drain 设置超时，超时后 drain 会以非零退出码退出，CI/CD 系统会检测到失败并告警，让运维人员介入排查。

## 不同场景的工作节点升级策略

### 原地升级（In-place Upgrade）

原地升级适用于使用 kubeadm 搭建的自建集群，在原有的节点机器上完成软件组件的版本升级，节点的操作系统和其他配置保持不变。

整体流程为：Drain 节点 → 升级 kubelet 和 kube-proxy 的二进制包 → 运行 `kubeadm upgrade node` → 重启 kubelet → Uncordon 节点。

`kubeadm upgrade node` 命令在节点上的实际动作包含几个关键步骤：

1. 向 API Server 请求最新的 `KubeletConfiguration`（这是 kubeadm 存储在 ConfigMap 中的集群级 kubelet 配置）
2. 将新的 kubelet 配置写入节点本地的配置文件（通常是 `/var/lib/kubelet/config.yaml`）
3. 更新 kubelet 的 systemd service 文件，指向新版本的二进制

然后重启 kubelet，新版本的 kubelet 进程启动，读取更新后的配置，向 API Server 重新注册自己（kubelet 启动时会 Patch Node 对象，更新 `status.nodeInfo.kubeletVersion`）。

**原地升级的主要风险点**在于节点状态的历史积累。历次升级的配置变更、日志文件、已删除但磁盘未释放的容器镜像层，都可能在节点上留下痕迹。此外，kubelet 的配置格式（`KubeletConfiguration` 的字段）在版本间可能有变化，如果你在节点上有自定义的 kubelet 配置，需要逐项检查字段兼容性，避免升级后 kubelet 因为无法解析配置而启动失败。

### 替换升级（Node Replacement）

替换升级的思路是用新版本节点替换旧版本节点，而不是在原节点上升级软件。这是云原生环境中更推荐的方式，也是"基础设施即代码（Infrastructure as Code）"理念的体现。

整体流程如下：

```
1. 创建新版本节点（使用新版本 Kubernetes 的节点镜像）
2. 新节点加入集群，向 API Server 注册，初始处于可调度状态
3. Cordon 旧节点（阻止新 Pod 调度到旧节点）
4. Drain 旧节点（驱逐现有 Pod，控制器将 Pod 重建到新节点）
5. 验证新节点上的 Pod 健康，应用指标正常
6. 删除旧节点（kubectl delete node），云上同步销毁旧虚拟机
```

这个方式的核心优势在于**节点状态的一致性**：新节点从一个干净的操作系统镜像启动，通过 cloud-init 或类似机制完成 Kubernetes 组件的安装，不携带任何历史遗留状态。节点配置完全由代码（Terraform、Ansible、节点池配置）描述和版本控制，任何节点都是可以随时销毁重建的。

在实践中，云环境通常通过**节点池（Node Pool）**的滚动替换来自动化这个流程：将节点池的目标版本调整为新版本，自动化系统依次创建新版本节点、将旧版本节点上的 Pod 驱逐到新节点、删除旧节点，整个过程可以配置并发数和等待时间。

替换升级对比原地升级的主要代价是时间成本：创建新节点（包括虚拟机创建、操作系统启动、Kubernetes 组件安装、节点注册）通常需要数分钟，而原地升级只需要重启 kubelet。这在节点数量较多时会带来明显的时间差异。

### 托管集群的升级方式

使用 EKS、GKE、AKS 等托管 Kubernetes 服务时，控制平面的升级完全由云厂商处理——etcd 备份、API Server 滚动升级、controller-manager 的 Leader Election 切换，这些操作都是透明的。但工作节点的升级仍然是用户的责任。

云厂商通常提供自动化的节点池升级能力，底层实现基本就是节点替换升级——创建新版本节点、将旧节点 drain、删除旧节点。这个过程通常可以配置并发更新的节点数（类似 `maxUnavailable`）和节点创建超时时间。

**托管集群升级中最容易被忽视的盲区是**：即使控制平面由云厂商维护，应用层面的兼容性完全是用户自己的责任。废弃的 API、变化的默认行为（如 Pod Security Standards 的策略变化）、Admission Webhook 的不兼容——这些问题云厂商无法帮你处理。云厂商通常会在控制台显示"已升级"，但不会告诉你哪些应用因为废弃 API 而失效了。

另一个托管集群的特殊场景是**控制平面与工作节点的版本差**：托管集群经常出现控制平面已经自动升级到新版本，但用户没有及时升级工作节点，导致版本偏差超出策略允许范围的情况。这时 kubelet 会开始打印兼容性警告，严重时某些 API 特性无法在旧版本 kubelet 上正常工作。

## 升级的完整检查清单

### 升级前：发现问题比修复问题更重要

**检查废弃 API（最容易忽视的坑）**

每个 Kubernetes 版本都会废弃一批 API，在若干版本后正式移除。典型例子：Ingress 的 `networking.k8s.io/v1beta1` 在 1.19 废弃，1.22 移除；PodSecurityPolicy 在 1.21 废弃，1.25 移除。如果你的 Deployment YAML、Helm Chart 或 Operator 中仍在使用这些被移除的 API，升级后这些资源相关的操作会直接失败。

工具 `kubent`（kube-no-trouble）和 `pluto` 可以扫描集群中所有正在使用的 API，自动标记出在目标版本中已被移除的 API。在升级前运行这些工具，是性价比最高的预防手段：

```bash
# kubent 扫描集群中使用的废弃 API
kubent

# pluto 扫描 Helm releases
pluto detect-helm --target-versions k8s=v1.31.0
```

**验证集群插件兼容性**

Kubernetes 集群依赖大量核心插件：CNI 网络插件（Calico、Cilium、Flannel）、CSI 存储插件、Ingress Controller 等。这些插件内部使用了 Kubernetes API 和各种特性，每个插件有自己的版本兼容矩阵。在升级集群之前，查阅每个插件的官方文档，确认当前版本与目标 Kubernetes 版本的兼容性。

CNI 插件的不兼容是最危险的。如果 CNI 插件在新版本 Kubernetes 上工作异常，节点升级后可能出现 Pod 无法分配 IP、跨节点通信失败等问题，影响范围是全集群所有工作负载。

**验证 Admission Webhook 兼容性**

MutatingWebhookConfiguration 和 ValidatingWebhookConfiguration 是常见的扩展机制，OPA/Gatekeeper、Istio、各种安全工具都会注册 Webhook。如果 Webhook 的 `failurePolicy` 设置为 `Fail`（这是严格安全策略的常见配置），Webhook 服务一旦不可用或返回错误，任何新 Pod 的创建都会被阻塞。

升级前需要梳理所有注册的 Webhook，确认对应服务版本与新版本 Kubernetes 的兼容性，同时评估 `failurePolicy` 的配置是否与升级窗口期的风险容忍度匹配。

**etcd 备份**

etcd 备份是升级前的非可选步骤，无论集群规模大小。备份应存储到与集群存储隔离的位置（独立的对象存储或备份服务器），备份完成后必须验证备份文件的完整性。

**确保 PDB 配置合理**

升级前巡检所有应用的 PDB 配置。对于关键应用，没有 PDB 意味着在 drain 时所有副本可能同时被驱逐（尤其是当多个副本碰巧在同一个节点时）。补充 PDB 配置是升级前最低成本的高可用保护手段，应在所有有服务连续性要求的应用上配置。

同时检查 `minAvailable` 的值是否合理：它应该小于副本数，留有驱逐容忍度。如果 `minAvailable` 等于副本数，drain 会被永久阻塞。

**健康度基线检查**

在升级操作开始之前，确认集群和所有 Pod 的当前状态是健康的：所有节点 Ready、所有 Pod Running、没有持续重启的 Pod。对一个本身不健康的集群执行升级，会极大增加问题诊断的难度——很难区分哪些问题是升级引入的，哪些是原本就存在的。

### 升级中：渐进与观察

**先控制平面，后工作节点**

这是版本偏差策略的硬性要求，不可违反。控制平面的 API Server 必须先于 kubelet 升级。kubelet 可以比控制平面低一个次要版本（1.30 的集群可以有 1.29 的 kubelet），但绝对不允许 kubelet 版本高于 API Server。

**逐个节点升级，而不是并发**

即使有 PDB 保护，生产环境中通常也不应该同时 drain 大量节点。并发 drain 会在同一时段内触发大量 Pod 的迁移，可能导致某些节点资源压力骤升，引发新一轮的 Pod 驱逐。逐个节点升级虽然慢，但每次只有一个节点的 Pod 在迁移，影响面可控，问题易于定位。

**每个节点升级后验证，再继续下一个**

节点 Uncordon 后，立即检查以下状态，确认正常后再继续：

```bash
# 确认节点版本、状态和可调度性
kubectl get node <node-name> -o wide

# 确认节点上所有 Pod 就绪
kubectl get pods --all-namespaces \
  --field-selector spec.nodeName=<node-name>

# 确认系统组件（kube-proxy、CNI Pod 等）正常
kubectl get pods -n kube-system \
  --field-selector spec.nodeName=<node-name>
```

同时观察应用监控面板，确认错误率和延迟在基线范围内，再继续下一个节点的升级。

### 升级后：持续监控与验证

升级完成并不意味着结束。常见的隐患往往在升级完成数小时乃至数天后才会显现：定时任务在升级后首次运行时触发了废弃 API 调用；HPA 在触发扩容时创建的新 Pod 遇到了 Webhook 兼容性问题；某些控制器因为版本差异在特定的代码路径上行为不同。

建议至少保持 24 小时的重点监控，持续关注以下指标：
- 所有节点和系统组件的状态（CoreDNS、kube-proxy、CNI Pod）
- 应用的错误率、响应时间和成功率
- Pod 的重启次数，持续重启通常是探针配置与新版本行为不匹配的信号
- HPA 的扩缩容动作是否正常触发和执行
- 节点的资源使用率是否在预期范围内

**版本一致性验证**：升级完成后，确认所有组件版本达到预期状态。包括所有工作节点的 kubelet 版本、kube-proxy 版本、核心插件的版本（CoreDNS、CNI）。版本不一致的节点可能成为后续操作中的隐患，比如版本偏差超限后，kubelet 拒绝某些来自 API Server 的特性请求。

**应用层功能验证**：仅靠指标监控是不够的，还需要主动验证关键业务路径的功能正确性。对于有端到端测试套件的团队，升级后立即运行一轮 E2E 测试；对于没有自动化测试的场景，与业务团队确认核心功能（登录、核心交易、报表生成等）均工作正常，形成书面记录。

## 升级对集群整体架构的影响

### Pod 分布与 Topology Spread

工作节点升级完成后，整个集群的 Pod 分布状态可能变得不均衡。这对使用了 `topologySpreadConstraints` 的应用尤其值得关注。

`topologySpreadConstraints` 允许声明 Pod 在节点或可用区之间的分布约束，例如要求相同应用的 Pod 在不同可用区之间均匀分布，或者要求同一节点上同一应用的 Pod 不超过特定数量。这个约束在调度新 Pod 时生效，但对已经运行的 Pod 无效。

升级过程中，节点轮流排空和重建，Pod 不断被迫迁移到当前可用的节点。迁移完成后，整个集群的 Pod 分布可能偏离原本期望的拓扑。例如，本来均匀分布在三个可用区的应用，在升级过程中可能因为某个可用区的节点临时不可用，而集中在剩余两个可用区。升级完成后，这些 Pod 不会自动重新平衡。

如果应用对拓扑分布有严格要求（如为了满足故障域隔离的合规要求），建议在完整升级完成后，通过滚动重启（`kubectl rollout restart`）触发 Pod 的重新调度，让调度器根据最新的集群拓扑重新决定 Pod 的分布。

### 节点资源变化对现有 Pod 的影响

工作节点升级（特别是替换升级）可能带来节点规格的变化——新版本的节点池可能使用了更大或更小的虚拟机实例。节点可分配资源（Allocatable Resources = 节点总容量 - 系统预留 - kubelet 预留）的变化，会直接影响节点上可以调度的 Pod 数量和资源上限。

在升级规划阶段，需要确认新节点的可分配资源与旧节点相当或更充裕。如果新节点的内存或 CPU 可分配量更小，可能导致 Pod 调度失败（Pending），或者触发 Node 上的资源压力（MemoryPressure、DiskPressure）进而引发 Pod 被节点驱逐。

## 跨多个版本升级的策略

Kubernetes 的版本偏差策略明确规定：每次升级只能跨一个次要版本（minor version）。从 1.28 升级到 1.31 需要完整地经历三次升级：1.28 → 1.29 → 1.30 → 1.31。

这个限制的根本原因在于 Kubernetes API 的演进模型。一个 API 的完整生命周期是：Alpha → Beta → GA → Deprecated → Removed。从 Deprecated 到 Removed 之间，Kubernetes 保证至少有两个次要版本的缓冲期。允许跳版本升级会打破这个保证：如果从 1.28 直接跳到 1.30，在 1.28 废弃并在 1.29 已经标记为"将在 1.30 移除"的 API，用户完全没有机会在中间版本接收到警告并进行修复。

此外，控制器逻辑和 API 的默认行为变化在版本间是递进式的，跳版本升级会让多个版本的行为变化叠加在一次操作中，问题排查的难度呈指数级上升。

**多版本升级的节奏控制**：每次升级都是一次完整的操作，需要执行完整的检查 → 升级 → 验证流程。建议两次升级之间至少间隔一周，让集群在新版本上充分运行，确认所有应用稳定后再进行下一次升级。这个间隔在有大量应用的生产集群上尤其重要，因为某些问题只有在特定场景（如月末批处理任务、流量峰值期间的自动扩容）触发时才会暴露。

**批量消除废弃 API 的思路**：在开始升级之前，先运行废弃 API 扫描工具，梳理所有使用了旧版本 API 的资源（Deployment YAML、Helm Chart、Kustomize 配置、自定义 Operator）。将所有资源的 API 版本更新到当前版本支持的最新稳定版本，这样每次升级只需要关注当前版本新增的废弃 API，积量不会越来越大。

**升级计划的制定**：对于有四个以上版本需要跨越的大跨度升级，建议先做一次完整的路径规划：列出每个中间版本的主要变化（可以在 Kubernetes changelog 中查阅，关注 "Breaking Changes" 和 "Deprecation" 部分），评估每个版本升级的风险点，识别哪些中间版本之间存在需要提前处理的 API 移除或行为变更。提前规划的好处是能够并行推进——在集群升级到下一个版本之前，应用团队可以提前修复已知的兼容性问题，而不是在升级时才发现并紧急修复，避免形成"升级阻塞"的局面。

## 常见踩坑与最佳实践

**踩坑一：没有配置 PDB，Drain 时应用短暂全量中断**

这是最常见的生产事故原因之一。一个 Deployment 的多个副本可能碰巧都在同一个节点上（特别是集群节点数较少时）。没有 PDB 时，drain 会同时驱逐这些 Pod，在新 Pod 在其他节点就绪之前，存在服务不可用的窗口。即使这个窗口很短（数十秒），对于关键业务也是不可接受的。

修复方式：为所有有服务连续性要求的应用配置 PDB。一个合理的起点是 `maxUnavailable: 25%`，或在副本数较少（如 2-3 个）的情况下，明确指定 `minAvailable: 1`。

**踩坑二：Pod 卡在 Terminating 状态，Drain 无法完成**

Pod 无法在 `terminationGracePeriodSeconds` 内正常退出的常见原因：
- 进程没有正确处理 SIGTERM 信号，忽略信号继续运行直到 SIGKILL
- `terminationGracePeriodSeconds` 设置不足，应用需要 60 秒优雅关闭，但配置只有 30 秒
- Finalizer 未被清理：某些 Operator 管理的 Pod 有 Finalizer，如果控制器出现问题，Finalizer 永远不被移除，Pod 就永远卡在 Terminating 状态，需要人工移除 Finalizer

**踩坑三：本地存储的 Pod 阻止 Drain**

Drain 默认拒绝驱逐使用 emptyDir 的 Pod，且不携带 `--delete-emptydir-data` 标志时会直接报错退出。这导致 drain 命令报错，运维人员容易习惯性地加上 `--delete-emptydir-data --force --ignore-daemonsets` 来"解决"问题。但这可能导致真正使用 emptyDir 存储业务数据（如本地队列缓存）的 Pod 被强制驱逐并丢失数据。

正确做法是先理解 emptyDir Pod 的用途，再决定是否可以安全驱逐。如果数据不可丢失，需要先将应用改造为使用 PV 或外部存储。

**踩坑四：废弃 API 导致升级后发版失败**

应用本身正常运行，但 CI/CD Pipeline、Helm Chart 或监控告警配置中使用了旧版本的 Kubernetes API。升级后旧 API 被移除，下一次通过 Pipeline 发版时，`kubectl apply` 报错 API 不存在。这类问题不会在升级完成时立即暴露，往往在下一次业务发版时才被发现，此时业务开发团队处于发版紧张状态，问题的紧迫性会被放大。

应在升级前使用 kubent 或 pluto 全面扫描，包括 Helm Chart 和 CI/CD 配置，而不仅仅是集群中当前运行的资源。

**踩坑五：CNI 插件不兼容新版本导致网络故障**

CNI 插件如果与新版本 Kubernetes 不兼容，表现形式多样：新创建的 Pod 长时间处于 ContainerCreating（CNI 无法分配 IP）、跨节点 Pod 通信失败（CNI 的路由规则与新版本内核或 kube-proxy 的行为不匹配）、NetworkPolicy 失效。

网络故障影响面极广，且问题现象和根因往往不直接相关，排查成本很高。在升级前查阅 CNI 插件官方的版本兼容矩阵，必要时在测试集群上先验证 CNI 插件与目标 Kubernetes 版本的兼容性，是规避此类问题的根本手段。

**踩坑六：升级期间 HPA 触发扩容，新 Pod 无法调度**

集群升级的节点排空阶段，部分节点处于 Cordon 状态（不可调度），同时有大量 Pod 在迁移，可用节点上的资源可能暂时紧张。如果此时业务流量上升触发了 HPA 扩容，新 Pod 可能因为资源不足而 Pending。

更棘手的情况是：PDB 会等待这些 Pending 的新 Pod 就绪后才允许下一个 Pod 被驱逐，而这些 Pod 因为资源不足永远无法 Ready——这形成了一个死锁。排查死锁的关键是先检查 Pending Pod 的事件（`kubectl describe pod`），确认是资源不足还是其他原因，再针对性处理（如临时增加节点）。

这也是为什么建议在业务低谷时段执行升级——低流量时 HPA 不容易触发扩容，节点上的 Pod 密度也相对较低，给迁移留有余量。

**最佳实践清单**

- 升级前在与生产环境尽量相似的测试集群上完成完整演练，包括所有检查步骤
- 为所有有可用性要求的应用配置 PDB、多副本，并配置正确的就绪探针和 preStop hook
- 使用 kubent 或 pluto 在升级前完成废弃 API 的全面扫描和清理，覆盖运行中的资源、Helm Chart 和 CI/CD 配置
- 升级操作安排在业务低谷时段执行，但高可用验证应该在高峰时段完成
- 升级过程中保持与应用团队的沟通，确保有人实时监控应用指标
- 为每次升级建立回滚预案，明确触发回滚的条件和操作步骤
- 升级完成后记录变更日志，包括过程中发现的问题和处理方式，积累团队知识
- 在测试环境模拟生产的 PDB 配置、副本分布和节点数，确保演练的有效性

### 升级就绪度评估框架

在判断一次升级是否真正"准备好了"时，可以用以下维度快速评估：

**应用维度**：集群中运行的所有应用是否都配置了 PDB？是否都配置了就绪探针？是否都实现了优雅关闭（或通过 preStop hook 进行了补偿）？如果一个应用的任何一项不满足，升级该应用所在节点时都存在潜在的服务中断风险。

**基础设施维度**：CNI 插件、CSI 存储插件、Ingress Controller 是否已确认与目标版本兼容？节点数是否充足，在一部分节点处于排空状态时，剩余节点能否承载全部工作负载的资源需求？这一点需要提前做容量规划，避免升级过程中因资源不足导致 Pod 无法迁移。

**流程维度**：etcd 备份是否已完成并验证？废弃 API 扫描是否完成并修复？是否有一份写明了"如果在某步骤出现 X 问题，执行 Y 操作"的操作手册？升级操作手册的作用不仅是指导执行，更是在高压情况下防止执行者因紧张而遗漏关键步骤。

## 小结

经过两篇文章，我们完整地走过了一次不停机 Kubernetes 集群版本升级的全过程。

第一篇聚焦于控制平面的原理：版本偏差策略的设计意图、etcd Raft 协议如何保证升级期间数据一致性、API Server 多实例滚动升级的无感知机制、Leader Election 如何让控制器在升级期间的主节点切换对工作负载透明。

这一篇聚焦于工作节点的实践：kubelet 心跳机制决定了重启 kubelet 的安全边界；Drain 通过 Eviction API 而非直接删除来触发 PDB 检查；Disruption Controller 持续维护 PDB 状态，精确控制每一刻可以中断的 Pod 数量；Pod 驱逐的时序决定了 preStop hook 和优雅关闭的配合方式；不同场景的升级策略（原地升级 vs 替换升级）各有适用范围；升级前的检查清单是规避已知问题的知识结晶。

回到最核心的问题：**不停机升级的本质是什么？**

归结起来是三个词：**冗余、渐进、验证**。

- **冗余**：多副本加上 PDB 是一切安全操作的基础，没有冗余就没有驱逐容忍度，升级就变成了停机升级
- **渐进**：逐节点升级、版本逐次跨越，将每次变更的影响面控制在最小范围内，问题可以被及时发现和止损
- **验证**：每一步操作之后的确认检查不是可选项，是缩短问题暴露时间、降低故障影响范围的最后防线

Kubernetes 提供了这些机制，但真正意义上的不停机升级，需要应用团队在设计时就为可运维性做好准备——应用无状态化、配置外部化、探针配置正确、优雅关闭实现完整。一个升级期间会全量中断的应用，再完善的集群级保护手段也无济于事。

集群升级是对整个技术团队的系统性考验，它暴露的不只是集群本身的健康状况，更是整个应用生态对"可中断性"的准备程度。那些在升级中暴露出高可用问题的应用，往往在日常运行中也存在隐患——只是没有机会被触发而已。从这个角度看，定期的集群升级也是一种强制性的高可用演练，帮助团队持续发现和修复应用架构中的薄弱环节，让整个系统在每次升级后都变得更加健壮。

## 常见问题

### Q1：工作节点升级期间，如果一个 Pod 的重建失败（如镜像拉取失败），会怎样？

Pod 的重建失败会导致该 Pod 持续处于 Pending 或 ImagePullBackOff 状态。这本身不会阻止 drain 完成——drain 只负责将原节点上的 Pod 驱逐，不负责确保新 Pod 在其他节点成功启动。从 drain 的视角看，它的任务在 Pod 被驱逐后就完成了。

但 PDB 机制会介入保护后续操作。如果 Pod 重建失败导致健康 Pod 数量降到 `minAvailable` 以下，Disruption Controller 会将 `disruptionsAllowed` 置为 0，针对其他节点的 drain 操作会被 PDB 阻塞，整个升级过程自动暂停。这是一个内建的安全阀：升级过程中只要出现应用可用性下降，后续的节点排空会自动等待，给运维团队发现和处理问题的时间。修复镜像问题后，Pod 恢复 Ready，drain 会自动继续。

### Q2：集群有 50 个工作节点，如何评估总升级时间？

时间由三个因素叠加决定：每个节点的 drain 时间、新 Pod 在其他节点就绪的时间、节点间的安全间隔。

对于典型场景（业务 Pod 配置了 30 秒优雅关闭、就绪探针通过需要 60 秒），单个节点的完整升级周期通常在 5-15 分钟。50 个节点逐个顺序升级，需要 4-12 小时。

如果需要控制总时间，可以在小范围内允许并发（如同时升级 3-5 个节点）。实际可行的并发度取决于集群的总 Pod 数、各应用的 PDB 约束，以及你愿意接受的同时处于迁移状态的 Pod 规模。大规模集群通常会将节点分批，每批 5-10 个节点，批次间做验证，而不是严格串行。

另一个影响时间的因素是节点上是否有使用了 initContainer 的 Pod。initContainer 会在主容器启动前顺序执行，如果 initContainer 本身执行时间较长（如数据库 schema 迁移），Pod 的整体启动时间会大幅延长，进而影响 PDB 的解锁时机，让整个升级过程比预期慢很多。在升级前梳理集群中有较长启动时间的 Pod，可以帮助更准确地估算升级窗口。

### Q3：升级过程中发现问题需要回滚，Kubernetes 支持回滚到旧版本吗？

Kubernetes 官方不支持版本降级（downgrade）。回滚到旧版本是一个复杂且风险极高的操作，主要原因是 etcd 的数据格式在版本间可能发生变化——新版本可能向 etcd 写入了旧版本 API Server 无法解析的资源格式，直接降级会导致 API Server 无法读取存量数据，后果是集群完全不可用。

正确的"回滚"策略是：在升级前做好 etcd 快照，升级出现不可接受的严重问题时，通过恢复 etcd 快照来回滚。但这意味着快照时间点之后的所有状态变更（新部署的应用、配置修改）都会丢失。

这就是为什么预防和分阶段升级如此重要——回滚的代价很高，目标是让回滚变成一个极少需要使用的选项，而不是一个常规操作。

### Q4：Drain 时配置了 `--timeout`，超时后发生了什么？

`--timeout` 指定 drain 的最大等待时间。超时后，drain 命令以非零退出码退出，报告超时错误。

重要的是，**超时不会强制驱逐任何 Pod**。超时时已经被驱逐成功的 Pod 已经离开了节点，但还没来得及驱逐的 Pod 仍然在原节点上运行。节点的 Cordon 状态不会被自动撤销，节点依然是不可调度的。

这意味着超时后需要人工介入：排查阻塞 drain 的原因（通常是 PDB 阻塞或 Pod 卡在 Terminating），修复问题后，重新对该节点执行 drain，或者根据情况决定是否强制驱逐剩余 Pod（`--force`）。不要跳过这一步直接对下一个节点执行 drain，因为当前节点可能仍然承担着业务流量，加上应用副本可能已经在上一步骤的驱逐中减少了。

### Q5：应用没有实现优雅关闭，如何在不修改代码的情况下减少升级时的请求中断？

在不修改应用代码的前提下，可以在 Pod 配置层面通过 preStop hook 和 `terminationGracePeriodSeconds` 来缓解问题。

在 Pod spec 中添加 preStop hook：

```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 15"]
terminationGracePeriodSeconds: 60
```

这个配置的作用机制是：Pod 进入 Terminating 状态后，Kubernetes 先执行 preStop hook（sleep 15 秒），这段时间内 Endpoint Controller 已经将该 Pod 从 Service Endpoints 移除，kube-proxy 也已在各节点更新了流量规则，不会有新请求路由到这个 Pod。Sleep 结束后，SIGTERM 发送给进程，`terminationGracePeriodSeconds` 给进程最多 60 秒的时间自然结束正在处理的请求，超时后 SIGKILL 强制结束。

这不如应用自身实现优雅关闭效果好，但在无法修改代码的约束下是一个可行的缓解手段。根本解决方案仍然是应用实现对 SIGTERM 的正确处理：停止接受新连接，等待现有请求处理完毕，然后主动退出，而不是被 SIGKILL 强制终止。

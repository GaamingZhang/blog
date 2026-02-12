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

# 不停机升级Kubernetes集群版本（一）：升级原理与控制平面

## 引言：两种不同性质的升级

想象一个场景：你的团队需要把生产集群从 Kubernetes 1.29 升级到 1.31。这个任务被排进了迭代计划，但负责的工程师犯了难——应用的滚动更新每天都在做，但集群本身的版本升级是另一回事。

这里有一个根本性的区别需要厘清。**应用升级**是把你的业务容器镜像从一个版本换成另一个版本，操作的主体是运行在集群上的工作负载，Kubernetes 平台本身保持不变。**集群升级**则是把 Kubernetes 平台自身——包括 kube-apiserver、etcd、kubelet 等核心组件——替换成新版本，这就像是在飞机飞行途中更换发动机，被操作的正是承载所有工作负载的基础设施。

不停机完成集群升级的前提只有一个：**高可用（HA）的控制平面**。如果你的集群只有单个控制平面节点，那就不存在"不停机"升级，因为升级该节点期间控制平面必然中断。HA 控制平面通常由三个或更多控制平面节点组成，配合负载均衡器对外提供统一的 API 入口。正是这个多实例架构，为滚动升级提供了空间。

另一个关键约束是：**一次只能跨一个小版本**。你不能把集群从 1.28 直接升到 1.31，必须依次经历 1.29、1.30、1.31 三次升级。这不是工具的限制，而是 Kubernetes 组件版本兼容性设计的必然结果，后面会详细解释。

---

## Kubernetes 版本策略

### 语义化版本号的含义

Kubernetes 使用 `Major.Minor.Patch` 的语义化版本格式，例如 `v1.31.4`。

- **Major（主版本）**：自 v1.0 发布以来一直是 1，代表 Kubernetes API 的核心设计稳定。升到 2.0 意味着破坏性的 API 变更，目前没有计划。
- **Minor（小版本）**：每次功能迭代都会递增，例如从 1.31 到 1.32。这是用户日常升级面对的版本跨度，每次升级都可能引入新特性、废弃旧 API，对集群管理有实质影响。
- **Patch（补丁版本）**：同一 Minor 版本内的安全修复和 bug 修复，例如 1.31.2 到 1.31.4。补丁升级通常只需要替换二进制文件，风险极低。

### 版本发布节奏与支持周期

从 Kubernetes 1.22 开始，社区采用了**每年发布三个 Minor 版本**的节奏（大约每四个月一个）。自 Kubernetes 1.19 开始，每个 Minor 版本的支持周期延长到**约 14 个月**（包含 12 个月常规支持和 2 个月补丁维护过渡期）。

这意味着在任意时刻，社区同时维护最近三个 Minor 版本。一旦某个版本超出支持窗口，不再接收安全补丁。对于生产集群，这要求运维团队每年进行两到三次 Minor 版本升级，否则集群将运行在不受维护的版本上。

### API 弃用策略

Kubernetes 的 API 弃用策略（Deprecation Policy）直接影响升级的可行性，理解它能帮你提前规避升级风险。

API 根据成熟度分为三个阶段：Alpha、Beta 和 GA（Generally Available）。

- **Alpha API**（例如 `v1alpha1`）：没有稳定性保证，可能在下一个 Minor 版本直接删除，不建议在生产中使用。
- **Beta API**（例如 `v1beta1`）：相对稳定，但在被 GA 取代后至少维持**九个月或三个 Minor 版本**（取较长者）。
- **GA API**（例如 `v1`）：已进入稳定阶段，在被废弃并移除前至少维持**十二个月或三个 Minor 版本**（取较长者）。

API 废弃是集群升级中最常踩的坑之一。举一个经典案例：`networking.k8s.io/v1beta1` 版本的 Ingress 在 Kubernetes 1.19 被废弃，在 1.22 中彻底移除。集群从 1.21 升到 1.22 时，那些仍然使用 `v1beta1` Ingress 的应用会立刻报错。

---

## 版本偏差策略：为什么只能逐版本升级

版本偏差策略（Version Skew Policy）是 Kubernetes 最核心的工程约束之一，它规定了集群中各组件之间允许的最大版本差距。理解这个策略，才能从根本上理解升级顺序和升级步长的限制。

### kube-apiserver 之间的偏差

在 HA 控制平面中，多个 kube-apiserver 实例同时运行。Kubernetes 规定，在任何给定时刻，**所有 kube-apiserver 实例之间的版本差距不能超过一个 Minor 版本**。

这个约束来自于滚动升级控制平面的需求。升级期间，你必然有一段时间新旧版本的 apiserver 同时存在。如果允许差距更大，不同版本的 apiserver 对同一个 API 对象的理解可能产生根本性分歧，导致数据一致性问题。

### kube-apiserver 与 kubelet 的偏差

这是最重要的偏差规则，直接影响升级顺序：

- **kubelet 不能比 kube-apiserver 版本更新**：这个规则保证了 kubelet 不会使用 apiserver 尚不支持的 API。
- **kubelet 可以比 kube-apiserver 最多旧两个 Minor 版本**（1.28 开始从旧一个变为旧两个，见 Kubernetes 1.28 版本偏差策略更新）。

用一个具体例子说明：如果 apiserver 是 v1.31，那么节点上的 kubelet 可以是 v1.31、v1.30 或 v1.29，但不能是 v1.28 或更旧。

这条规则决定了一个重要的工程实践：**工作节点的 kubelet 升级可以滞后于控制平面**，这为分批升级节点提供了基础。但如果你的集群从未升级过，节点的 kubelet 版本可能跨越多个版本，必须补齐到合规范围内。

### kube-apiserver 与 kube-controller-manager、kube-scheduler 的偏差

这两个组件的版本**必须不高于 kube-apiserver**，且最多比 apiserver 旧一个 Minor 版本。实践中通常让它们与 apiserver 版本一致，并在 apiserver 升级后立即升级这两个组件。

### kube-apiserver 与 kubectl 的偏差

kubectl 允许在 kube-apiserver 版本的**前后各一个 Minor 版本**范围内工作。这意味着当 apiserver 是 v1.31 时，v1.30、v1.31、v1.32 的 kubectl 都可以正常使用。这个宽松的规则方便了开发者在本地使用不同版本的 kubectl 管理集群。

### 偏差规则存在的根本原因

这些规则并非武断的工程决策，而是由三个技术现实决定的。

第一，**API 版本转换**。Kubernetes 支持同一资源的多个 API 版本（例如 `apps/v1` 和历史上的 `extensions/v1beta1`），apiserver 内部维护着版本转换逻辑。如果新旧版本的 apiserver 差距过大，转换逻辑可能不兼容，导致对象在两个实例间来回转换时丢失字段或产生错误。

第二，**Feature Gate（功能门控）**。Kubernetes 通过功能门控逐步推出新特性。某个功能在 v1.29 是 Alpha（默认关闭），在 v1.30 是 Beta（默认开启），在 v1.31 是 GA（稳定）。版本差距过大时，不同组件对同一个功能的行为预期就会产生矛盾。

第三，**序列化格式与字段演化**。同一个对象在不同版本的 apiserver 内部表示（internal representation）可能不同。字段被添加、重命名或移除，如果跨越多个版本，新旧组件对同一份 etcd 数据的解析结果可能不一致。

偏差规则通过限制版本差距，把这些兼容性问题约束在一个可管理的范围内，确保每次升级只处理"相邻版本"之间的差异。

### 证书有效期与版本偏差的交叉影响

版本偏差策略还与证书管理存在一个容易忽视的交叉问题。kubeadm 创建的集群中，客户端证书（包括 apiserver 访问 etcd 的客户端证书、controller-manager 和 scheduler 访问 apiserver 的客户端证书）默认有效期为一年。`kubeadm upgrade` 的一个副作用是**自动轮换这些证书**。

这意味着如果你的集群超过一年没有升级（也没有手动轮换证书），证书过期会导致控制平面组件无法互相认证，集群进入不可用状态。这个问题与版本偏差策略叠加，构成了一个强制集群持续升级的隐性约束：你不仅要在版本支持窗口内升级，还要保证证书不过期。

对于不使用 kubeadm 管理证书的集群（如部分自建集群或云厂商托管集群），这个问题的处理方式会有所不同，但同样需要关注。

---

## 升级前的评估与准备

### API 废弃检查

在升级之前，必须确认集群中没有正在使用即将被删除的 API。有两种常用方式。

一是使用 `kubectl api-versions` 和 `kubectl api-resources` 查看当前集群支持的 API 版本，并对照目标版本的 Kubernetes 迁移指南，确认哪些 API 即将消失。

二是使用开源工具 [kubent（kube-no-trouble）](https://github.com/doitintl/kube-no-trouble)，它会扫描集群中所有已部署的资源，检测其中使用了废弃 API 的对象，并给出迁移建议。这比手动比对高效得多。

### etcd 备份：升级前的必须步骤

etcd 是 Kubernetes 集群的"唯一数据源"，它存储了整个集群的状态——所有的 Deployment、Service、ConfigMap、Secret、RBAC 规则，以及 Kubernetes 自身的运行时状态（如 Lease 对象、事件等）。没有 etcd，集群状态无法恢复。

etcd 的备份采用**快照（snapshot）机制**。执行 `etcdctl snapshot save` 时，etcd 会在当前的 Raft 日志索引处创建一个一致性快照，将内存中的 B-tree 数据结构序列化写入文件。这个快照包含了快照时刻的完整集群数据，且是一致的（不会有部分写入的中间状态）。

备份文件应该存放在控制平面节点之外的位置（例如对象存储或 NFS），因为如果控制平面节点本身损坏，存在节点本地的备份也就无从访问。

恢复时，使用 `etcdctl snapshot restore` 将快照导入新的 etcd 数据目录，然后让 etcd 从这个数据目录启动。恢复后的集群状态会回到备份时刻，备份后发生的所有操作（新创建的 Pod、配置变更等）都会丢失。这就是为什么备份应该尽可能在升级操作开始前的最后时刻进行。

### 组件兼容性检查

控制平面和 kubelet 升级后，附加组件（Add-ons）是否兼容新版本是另一个重要检查点。

**CNI 插件**（如 Calico、Cilium、Flannel）负责容器网络，它们通过 CNI 接口与 kubelet 交互，并可能调用 Kubernetes API。新版本可能改变相关 API 或行为，需要确认 CNI 版本对新 Kubernetes 版本的支持声明。

**CSI 驱动**负责持久化存储的挂载与卸载，同样依赖特定的 Kubernetes API。如果 CSI 驱动不兼容新版本，可能导致新建 PVC 无法挂载，或已挂载的卷出现异常。

**Admission Webhook**（ValidatingWebhookConfiguration 和 MutatingWebhookConfiguration）是集群安全策略的关键环节，它们拦截 API 请求并做验证或修改。某些 Webhook 实现依赖特定版本的 API 资源，如果目标版本移除了这些 API，Webhook 服务本身可能报错，进而导致整个集群的资源创建操作失败（因为 Webhook 失败会阻塞请求）。

### 节点健康检查

升级前确保所有节点处于 `Ready` 状态。如果某个节点已经处于 `NotReady` 或 `Unknown` 状态，先解决这些问题再进行升级。升级过程中需要对节点执行驱逐操作，一个不健康的节点可能导致驱逐流程卡住。

同时检查是否存在 `Evicted` 或 `Terminating` 状态的 Pod 卡住。这类 Pod 往往是历史遗留问题的信号，放任它们存在会干扰升级过程中的状态判断。

### PodDisruptionBudget 的预检

升级前还需要检查集群中所有 PodDisruptionBudget（PDB）的配置是否合理。PDB 定义了在主动干扰（Voluntary Disruption，如节点驱逐）期间某个应用最少保持多少 Pod 可用。

一个常见的陷阱是：某个应用只有一个副本，但其 PDB 配置了 `minAvailable: 1`。这意味着在驱逐这个节点时，Kubernetes 会一直等待另一个可用的 Pod——但这个 Pod 永远不会出现，驱逐操作就会永远卡住。在开始升级前，使用 `kubectl get pdb -A` 检查所有命名空间的 PDB，确认它们的配置与实际副本数之间没有逻辑矛盾。

---

## 控制平面升级的底层机制

### 控制平面组件架构回顾

控制平面由四个核心组件构成：

```
+---------------------------+
|       Control Plane       |
|                           |
|  +---------------------+  |
|  |      etcd           |  |  <- 数据持久化层
|  +---------------------+  |
|           |                |
|  +---------------------+  |
|  |  kube-apiserver     |  |  <- API 入口
|  +---------------------+  |
|      |           |         |
|  +-------+  +--------+    |
|  |  KCM  |  |  Sched |    |  <- 控制循环
|  +-------+  +--------+    |
+---------------------------+
```

升级必须按照 etcd → kube-apiserver → kube-controller-manager/kube-scheduler 的顺序进行，原因在于每个上层组件都依赖下层组件的 API 稳定性。逆序升级会导致上层组件调用了下层尚不支持的 API，产生运行时错误。

### etcd 升级：Raft 协议的安全保证

etcd 是分布式键值存储，其一致性由 Raft 共识协议保证。理解 Raft 是理解 etcd 滚动升级安全性的关键。

Raft 协议将集群节点划分为三种角色：Leader、Follower 和 Candidate。所有写操作必须经过 Leader，Leader 将写操作作为日志条目（log entry）发送给所有 Follower，只有当**多数派（quorum，即超过半数节点）**确认接收后，这条日志才被提交（commit）并应用到状态机。

以三节点 etcd 集群为例，quorum 是 2。滚动升级时，一次只升级一个节点：

```
初始状态:   [A-旧] [B-旧] [C-旧]   quorum=2，正常运行

升级节点A:  [A-停止] [B-旧] [C-旧]
            B 或 C 成为新 Leader   仍有 2 节点，quorum 满足

节点A重启:  [A-新] [B-旧] [C-旧]   3 节点恢复，quorum=2

升级节点B:  [A-新] [B-停止] [C-旧]  仍有 2 节点，继续服务

...依此类推
```

在整个过程中，始终有至少两个节点（即 quorum）在线，读写请求从未中断。Raft 保证了即使 Leader 在升级过程中被停止，也会在剩余节点中自动选举出新的 Leader，这个选举通常在几秒内完成。

etcd 的数据持久化依赖**预写日志（Write-Ahead Log，WAL）**。所有写操作在被应用到内存中的 B-tree（bbolt 引擎）之前，都会先顺序追加到 WAL 文件中。WAL 的存在保证了即使 etcd 进程在写入过程中崩溃，重启后也能通过重放 WAL 恢复到崩溃前的一致状态。在滚动升级期间，被停止升级的节点重启后会先读取 WAL 恢复状态，然后通过 Raft 的日志同步机制补齐它停止期间错过的日志条目，最终重新加入集群。

etcd 的版本升级还涉及存储格式（storage format）。从 etcd v2 到 v3 是一次重大变革（v3 使用 MVCC 引擎取代了 v2 的目录树模型），但在当前 Kubernetes 生态中，etcd v3 的小版本升级不涉及存储格式变化，只需替换二进制文件并重启。

值得注意的是，etcd 的版本与 Kubernetes 版本之间存在对应关系，通常每个 Kubernetes Minor 版本会指定它支持的 etcd 版本范围。例如，Kubernetes 1.31 使用 etcd 3.5.x 系列。kubeadm 会在升级时自动拉取与 Kubernetes 版本配套的 etcd 镜像，不需要手动管理这个对应关系，但在非 kubeadm 管理的集群中，etcd 版本升级必须与 Kubernetes 版本升级协调进行。

### kube-apiserver 升级：存储版本与 API 转换

kube-apiserver 的升级是控制平面升级中机制最复杂的部分。

**存储版本（Storage Version）**

Kubernetes API 对象在 etcd 中以特定版本序列化存储，这个版本称为 Storage Version，由每个资源的 APIGroup 设定。例如，`Deployment` 的 Storage Version 在不同 Kubernetes 版本中经历了 `extensions/v1beta1` → `apps/v1beta1` → `apps/v1` 的演进过程。

当你访问 `GET /apis/apps/v1/deployments/my-app` 时，apiserver 的处理流程是：

```
1. 从 etcd 读取数据（以 storage version 序列化的二进制）
2. 解码为内部对象（internal object，版本无关的中间表示）
3. 将内部对象转换为请求的 API 版本（v1）
4. 序列化为 JSON/Protobuf 返回给客户端
```

写入时的流程相反：从请求版本转换为内部对象，再从内部对象转换为 storage version，最后写入 etcd。

这个双向转换机制是 HA 滚动升级期间 API 兼容性的核心保障。升级期间，v1.30 和 v1.31 的 apiserver 同时存在，它们可能对某些对象有不同的内部表示，但只要两者都能正确地从 etcd 的 storage version 解码，并能向客户端提供所请求的 API 版本，请求就能被正确处理。

**存储版本迁移（Storage Version Migration）**

一个容易被忽略的细节是：控制平面升级后，etcd 中已经存储的旧版本对象并不会被自动转换为新的 storage version。新的 apiserver 拥有读取旧格式的能力（向后兼容），但这些旧格式对象会一直停留在 etcd 中，直到被显式更新。

举例来说，假设某个 API 资源的 storage version 从 v1beta1 升级到 v1。etcd 中已有的对象仍然以 v1beta1 格式存储。新版本 apiserver 在读取这些对象时，会在内存中完成版本转换后返回给客户端，但不会回写 etcd。这意味着从 etcd 直接导出数据时，你可能会看到混合了新旧两个版本格式的数据。

如果要清理这种历史遗留状态，可以使用 `kube-storage-version-migrator` 工具，它会遍历指定类型的所有资源，执行一次 `GET` 后立即 `PUT` 回去，强制 apiserver 以当前 storage version 写入 etcd，完成格式迁移。在大规模集群上，这个过程可能需要较长时间，并对 etcd 产生写入压力，通常不需要在升级的当下立即执行。

**APIServerID 与对象版本的追踪**

Kubernetes 1.20 引入了 `StorageVersionHash` 机制，每个 APIGroup 资源在被 apiserver 注册时会计算一个版本哈希值。这个哈希值记录在 `APIResourceList` 中，客户端可以通过它感知资源的存储版本是否发生了变化，从而决定是否需要重新同步本地缓存。这个机制对于运行在集群内的控制器（如 controller-manager 内部的 informer 缓存）在升级期间保持一致性有重要意义。

**滚动升级过程**

HA 控制平面通常在 apiserver 前面放置一个负载均衡器（Layer 4 TCP 负载均衡）。升级流程如下：

```
负载均衡器
    |
    +--- apiserver-1 (v1.30)
    +--- apiserver-2 (v1.30)
    +--- apiserver-3 (v1.30)

步骤1：停止 apiserver-1，升级为 v1.31，重启
    +--- apiserver-1 (v1.31)   <- 新版本
    +--- apiserver-2 (v1.30)   <- 旧版本，继续服务
    +--- apiserver-3 (v1.30)   <- 旧版本，继续服务

步骤2：停止 apiserver-2，升级为 v1.31，重启
步骤3：停止 apiserver-3，升级为 v1.31，重启
```

负载均衡器的健康检查会在 apiserver 停止期间将其从后端摘除，等它重启并通过健康检查后再重新加入。整个过程中，始终有两个 apiserver 实例在服务。

需要注意一点：由于请求可能被路由到不同版本的 apiserver，客户端应该处理短暂的 503 或连接重置，并实现幂等重试。Kubernetes 官方客户端库（client-go）默认包含了这个重试逻辑。

### kube-controller-manager 与 kube-scheduler：Leader Election 机制

这两个组件只需要运行单个活跃实例（尽管可以部署多个副本实现高可用）。即使同时运行多个副本，同一时刻也只有一个实例在主动工作，其他实例处于待机状态，这是通过 **Leader Election（领导者选举）** 机制实现的。

**Lease 对象与领导者选举**

Kubernetes 使用 `coordination.k8s.io/v1` API 中的 `Lease` 对象作为分布式锁。每个需要选举的组件会创建或更新一个特定名称的 Lease 对象，Lease 记录了当前 Leader 的标识和最后续约时间。

选举过程如下：

```
1. 多个 KCM 实例启动，都尝试创建/更新同一个 Lease 对象
2. 由于 etcd 的 MVCC 保证写操作的原子性，只有一个实例能成功
3. 成功更新 Lease 的实例成为 Leader，开始执行控制循环
4. Leader 每隔 leaseDuration/2 更新 Lease 的 renewTime 字段
5. 其他实例（Follower）定期检查 Lease，如果 renewTime 超过 leaseDuration 未更新，认为 Leader 已故障，重新竞争选举
```

默认的 `leaseDuration` 是 15 秒，`renewDeadline` 是 10 秒，`retryPeriod` 是 2 秒。这意味着 Leader 故障后，新 Leader 最多需要 15 秒才能接管，这是控制平面升级期间可能出现的最长控制循环停顿。

**升级期间的 Leader 切换**

当你升级 kube-controller-manager 时，旧版本实例被停止，它持有的 Lease 不再被续约。经过最多 15 秒的 `leaseDuration`，新启动的 v1.31 实例会赢得选举，成为新 Leader，继续执行控制循环。

在这 15 秒内：
- 已经运行的 Pod 不受影响，kubelet 独立管理本地容器的生命周期；
- 新的 Deployment 副本调整、故障重建等依赖控制器的操作会短暂延迟；
- 这个延迟对大多数业务场景是无感知的。

这与 kube-scheduler 的行为完全类似——调度器停止期间，新 Pod 会停留在 `Pending` 状态，等调度器恢复后继续被调度。

---

## 控制平面升级对工作负载的影响

### 已运行的 Pod 不依赖控制平面

这是 Kubernetes 架构设计中一个非常重要的特性：**已经运行的 Pod 不需要控制平面的参与来维持运行状态**。

原因在于 kubelet 的工作方式。kubelet 运行在每个节点上，它负责维护本节点上所有 Pod 的生命周期。一旦 kubelet 从 apiserver 获取到 Pod 的配置信息并将其调度到本节点，Pod 就在 kubelet 的直接管理下运行。即使 apiserver 完全不可用，kubelet 依然会：

- 确保 Pod 中的容器按照 spec 运行；
- 在容器崩溃后按照 restartPolicy 重启容器；
- 执行健康探针并在探针失败时重启容器。

kubelet 通过本地维护一份 Pod 状态的内存缓存（podCache），即使与 apiserver 断开连接，也能继续管理容器。这个机制被称为 kubelet 的"自主运行能力"。

从底层实现来看，kubelet 与容器运行时（containerd 或 CRI-O）之间通过 CRI（Container Runtime Interface）gRPC 接口通信，这个通信路径完全绕过了 apiserver。kubelet 告知容器运行时"启动这个容器"、"停止这个容器"，容器运行时返回执行结果，整个过程与 Kubernetes 控制平面无关。控制平面不可用只是让 kubelet 无法将节点状态更新（Pod 状态、节点心跳）写回 apiserver，但不影响它管理容器的能力本身。

有一个边界情况值得注意：如果 kubelet 与 apiserver 的断开时间超过 `node-monitor-grace-period`（默认 40 秒）再加上一段缓冲时间，kube-controller-manager 中的 NodeLifecycleController 会将节点标记为 `Unknown` 状态，并可能触发节点上 Pod 的驱逐。但在正常的控制平面滚动升级过程中，apiserver 的中断时间远短于这个阈值，不会触发这个机制。

### Watch 机制在升级期间的行为

运行在集群内的控制器（包括各类 Operator 和 controller-manager 内部的控制器）通过 Kubernetes 的 Watch API 实现事件驱动，持续监听资源变更。当 apiserver 实例在升级期间重启时，这些 Watch 连接会被断开。

client-go 的 informer 框架内置了对 Watch 断开的处理逻辑：它会以指数退避的方式重新建立 Watch 连接，并使用 `ResourceVersion` 字段从断开点继续接收事件，而不需要从头列举所有对象。`ResourceVersion` 是 etcd 的全局修订号（revision），每次写操作都会递增，apiserver 利用这个单调递增的值来标记每个资源的版本，客户端可以通过指定 `ResourceVersion` 告诉 apiserver "给我这个版本之后发生的所有变更"。这个机制确保了集群内的控制器在 apiserver 重启后能够快速恢复，不会错过重要事件。

对于 kube-proxy 来说，它同样通过 Watch 机制监听 Service 和 EndpointSlice 的变更，并将其转换为节点上的 iptables 或 ipvs 规则。控制平面升级期间，kube-proxy 与 apiserver 的 Watch 连接断开后，它保持节点上现有的网络规则不变，已经建立的网络连接不受影响。Watch 恢复后，kube-proxy 会补齐断开期间错过的变更。因此，控制平面升级不会导致现有的 Service 访问中断。

### 升级期间受影响的操作

控制平面的短暂中断（滚动升级期间每次仅影响一个 apiserver 实例）会影响以下操作：

| 操作类型 | 影响 |
|----------|------|
| kubectl 命令 | 若请求落到正在升级的实例，会短暂报错，重试后成功 |
| 新 Pod 的调度 | kube-scheduler 停止期间，新 Pod 停在 Pending |
| Deployment 副本控制 | KCM 停止期间，副本调整有最多 15 秒延迟 |
| HPA 自动扩缩 | 依赖 KCM，同样有短暂延迟 |
| 新建 PVC 绑定 | 依赖 KCM 的 PersistentVolumeController |
| Admission Webhook | 依赖 apiserver，apiserver 升级期间对应实例不可用 |

### 最小化影响窗口的实践

在业务低峰期（例如夜间）执行升级，可以将影响降到最低。对于 kube-controller-manager 和 kube-scheduler，可以先启动新版本实例（此时旧版本仍是 Leader），确认新实例健康后再停止旧版本，这样 Leader 切换会更快（因为新实例已经参与选举竞争）。

控制平面升级期间，建议在应用层面配置合理的重试策略。由于 apiserver 的负载均衡器在检测到后端实例不可用时需要一定的检测时间（通常是几秒），这段时间内落到该实例的请求会报错。使用 client-go 构建的控制器通过 informer 的 resync 机制和 workqueue 的重试功能天然具备容错能力；但直接调用 kubectl 的脚本或 CI/CD 流水线应该添加显式的重试逻辑，避免因为升级期间的短暂波动导致操作误判为失败。

---

## kubeadm 升级控制平面的流程解析

kubeadm 是管理 Kubernetes 生命周期的官方工具，它把控制平面升级的复杂步骤封装成了两个主要命令。

### kubeadm upgrade plan

`kubeadm upgrade plan` 执行一系列预检查，告诉你当前集群是否可以升级，以及可以升级到哪些版本。它会检查：

- 当前集群所有控制平面节点的版本状态；
- 目标版本与当前版本的兼容性；
- 核心附加组件（CoreDNS、kube-proxy）的当前版本和可升级版本；
- 是否存在任何阻止升级的条件（如节点不健康）。

`upgrade plan` 还会输出一个重要的警告信息：即将被废弃的 API。这是 API 废弃检查的快速入口，但它只覆盖 kubeadm 本身知晓的变更，不能完全替代 kubent 等专门工具的全面扫描。

### kubeadm upgrade apply 的底层步骤

`kubeadm upgrade apply v1.31.x` 的执行过程比表面看起来要复杂，它依次完成：

1. **下载目标版本的配置**：从 Kubernetes 发布的配置仓库获取新版本的默认配置。kubeadm 使用 ConfigMap `kube-system/kubeadm-config` 中记录的配置作为基础，与新版本的默认配置合并，生成本次升级的实际配置。

2. **生成新的控制平面组件 manifest**：kubeadm 在 `/etc/kubernetes/manifests/` 目录下维护着 kube-apiserver、kube-controller-manager、kube-scheduler 的静态 Pod manifest 文件。这些 manifest 指定了组件的镜像版本和启动参数。kubeadm 会将这些文件更新为新版本。

3. **静态 Pod 的自动重启**：这是 kubeadm 升级机制中最巧妙的设计。kubelet 实现了一个**目录监视（directory watch）机制**，持续监控 `/etc/kubernetes/manifests/` 目录的文件变化。当 kubeadm 修改某个 manifest 文件时，kubelet 立即检测到变化，自动停止旧的控制平面容器并按照新 manifest 启动新版本的容器。整个过程不需要手动重启服务，也不需要 systemd 参与。静态 Pod 的重启是逐个进行的——先更新 kube-apiserver 并等待就绪，再更新 kube-controller-manager，最后更新 kube-scheduler，确保每一步都在上一步成功后再进行。

4. **等待 apiserver 就绪**：kubeadm 会轮询 apiserver 的 `/healthz` 端点，等待新版本的 apiserver 完全就绪后再继续后续步骤。这一轮询通常设置较长的超时（数分钟），以应对 apiserver 启动慢的情况。

5. **证书轮换**：如果距离证书过期时间不足 80% 的有效期，kubeadm 会自动轮换控制平面组件的客户端证书。这个行为是默认开启的，可以通过 `--certificate-renewal=false` 关闭。

6. **升级集群内的配置对象**：更新 kube-system 命名空间中的 ConfigMap（如 `kubeadm-config`、`kube-proxy` 配置），以及 ClusterRole 和 ClusterRoleBinding 等 RBAC 资源，使它们与新版本保持一致。

7. **升级附加组件**：更新 CoreDNS 和 kube-proxy 到与新 Kubernetes 版本匹配的版本。

对于 HA 集群中第二、三个控制平面节点，使用 `kubeadm upgrade node` 而非 `upgrade apply`，它只执行上述流程中与本节点相关的部分（更新 manifest，等待组件就绪），不会重复执行集群级别的配置更新。

### 升级过程中的回滚边界

理解 kubeadm 升级的一个关键局限：**升级过程通常不可逆**。一旦 kube-apiserver 成功启动并开始以新版本运行，etcd 中可能已经写入了新版本才能理解的数据格式（例如新版本的 storage version）。此时降级回旧版本的 apiserver 可能无法读取这些数据，导致集群异常。

这也是 etcd 备份如此重要的原因——它是你在升级出现严重问题时唯一可靠的回滚手段，通过将 etcd 恢复到备份时刻的状态，配合旧版本的控制平面镜像，才能实现真正意义上的版本回退。

---

## 控制平面证书管理与升级的关系

在 kubeadm 管理的集群中，控制平面组件之间的通信使用 TLS 双向认证，涉及多套证书体系：

- **CA 证书**（Certificate Authority）：根证书，有效期通常为 10 年，由 kubeadm 在初始化时生成，存放在 `/etc/kubernetes/pki/` 目录下。所有其他证书都由 CA 签发。
- **apiserver 服务端证书**：供客户端（kubelet、kubectl、controller-manager 等）验证 apiserver 身份，有效期默认 1 年。
- **apiserver 客户端证书**：apiserver 访问 kubelet 时使用的客户端证书，有效期默认 1 年。
- **controller-manager 和 scheduler 的客户端证书**：它们访问 apiserver 时使用，有效期默认 1 年。
- **etcd 的服务端与客户端证书**：用于 etcd 节点间通信和 apiserver 访问 etcd，有效期默认 1 年。

这些一年期证书的存在，与之前提到的版本支持周期（14 个月）共同构成了一个隐性的"强制升级"机制。在实践中，`kubeadm upgrade apply` 执行时会自动更新这些一年期证书，这是升级操作的一个重要副作用。

对于 kubelet 的客户端证书（kubelet 用于向 apiserver 认证自身的证书），如果集群开启了 `RotateKubeletClientCertificate` 功能门控（1.19 起默认开启），kubelet 会在证书到期前 80% 时自动向 apiserver 申请轮换，不依赖 kubeadm upgrade 来处理。这个自动轮换机制意味着工作节点的证书管理是独立进行的，不需要和控制平面升级捆绑在一起。

但 CA 证书的 10 年有效期是一个值得长远规划的问题。当 CA 证书到期时，所有由它签发的下级证书都会同时失效，这将是一次牵动整个集群的大规模操作。在 CA 证书到期前，需要提前更换 CA 并滚动更新所有下级证书，这个过程远比常规版本升级复杂。

---

## 小结

理解 Kubernetes 集群不停机升级的原理，关键在于把握以下几个核心机制：

- **版本偏差策略**决定了升级步长（一次只能跨一个 Minor 版本）和升级顺序（控制平面先于工作节点）；
- **Raft 多数派机制**保证了 etcd 滚动升级期间数据一致性不被破坏；
- **存储版本与 API 转换机制**使得不同版本的 apiserver 实例能够同时服务而不产生数据矛盾；
- **Leader Election（Lease 机制）**保证了 KCM 和 Scheduler 的升级切换几乎无感知；
- **kubelet 的自主运行能力**保证了控制平面升级期间已有工作负载不受影响；
- **静态 Pod 机制**使得 kubeadm 可以通过修改 manifest 文件触发控制平面组件的原地升级。

下一篇将聚焦工作节点的升级——节点驱逐、Pod 迁移、kubelet 升级，以及如何正确处理有状态工作负载的节点升级流程。

---

## 常见问题

### Q1：为什么 Kubernetes 不支持跨多个 Minor 版本直接升级，而必须逐版本进行？

根本原因是版本偏差策略的限制，而这个限制本身源自 API 兼容性设计。每个 Minor 版本都可能引入新的 API 字段、废弃旧的 API、改变 Feature Gate 的默认值。如果允许跨两个 Minor 版本升级，意味着两个相差两个版本的组件需要同时运行并互相通信——而 Kubernetes 的测试矩阵并不覆盖这种跨版本组合，其行为是未经验证的。逐版本升级的本质是把复杂的多版本兼容问题拆分为一系列相邻版本之间已经被充分验证的兼容性问题。

### Q2：etcd 升级期间如果 Leader 节点恰好被停止，会丢数据吗？

不会。Raft 协议保证，一条日志条目只有在被多数派（quorum）确认后才会被提交。当 Leader 节点停止升级时，所有已经被提交的条目已经存在于多数派节点上；尚未提交的条目（即还没有收到多数派确认的条目）在新 Leader 选出后会被处理——如果它已经被多数派接收，新 Leader 会继续提交它；如果没有，它会被丢弃。这个机制保证了数据不会在正常的滚动升级场景下丢失。需要强调的是，etcd 备份仍然是必须的，它防范的是非预期灾难（如磁盘损坏、误操作），而不是正常的升级过程。

### Q3：升级过程中 Admission Webhook 失败会导致什么问题，如何预防？

如果一个 ValidatingWebhookConfiguration 或 MutatingWebhookConfiguration 的 `failurePolicy` 被设置为 `Fail`（这是生产环境的常见配置），那么当 Webhook 服务不可用时，所有匹配该 Webhook 的 API 请求都会被拒绝。在控制平面升级期间，如果 Webhook 依赖的 API 被移除或 Webhook 服务本身因为不兼容新版本而崩溃，可能导致整个集群无法创建新资源。预防措施包括：升级前验证所有 Webhook 对新版本的兼容性；对于非关键 Webhook，考虑临时将 `failurePolicy` 改为 `Ignore`；确保 Webhook 服务本身部署了足够的副本，避免在升级期间因节点驱逐而全部不可用。

### Q4：kubelet 比 kube-apiserver 旧两个版本是否会有功能缺失？

会有，但 Kubernetes 的设计原则保证了这种差距不会导致关键功能的不可用。apiserver 对于旧版本 kubelet 的 API 请求，会以旧版本能够理解的方式响应，不会强制要求 kubelet 使用新版本的特性。但是，在旧 kubelet 节点上，需要新版本 kubelet 才能支持的功能（如新的容器运行时特性、新的 QoS 策略等）将无法使用。这种状态应该被视为临时过渡状态，工作节点应该尽快完成 kubelet 升级，避免长期在版本偏差范围的边缘运行。

### Q5：如何验证控制平面升级成功，有哪些健康检查手段？

控制平面升级后，应从以下几个维度验证健康状态。首先，检查所有控制平面节点的状态：`kubectl get nodes` 确认所有节点仍处于 `Ready` 状态，`kubectl get pods -n kube-system` 确认所有系统组件 Pod 正常运行。其次，验证 API 可用性：`kubectl cluster-info` 返回正确的 apiserver 地址，简单的读写操作（如创建并删除一个测试 Namespace）能正常完成。再次，检查组件的版本一致性：`kubectl version` 和 `kubectl get nodes -o wide` 应显示与目标版本一致的结果。最后，查看控制平面组件的日志：`kubectl logs -n kube-system kube-apiserver-<node>` 确认没有异常错误，特别是与 etcd 连接、证书有效性相关的错误。

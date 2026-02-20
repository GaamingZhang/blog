---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# CRI与容器运行时

## CRI是什么，为什么需要它

想象这样一个场景：你的公司在2016年既跑Docker又要试验rkt，而每一种运行时都需要Kubernetes团队在kubelet源码里专门写一套适配代码。增加一个新运行时，就必须修改kubelet、编译、测试、发版。这不是扩展性，这是技术债。

CRI（Container Runtime Interface）就是为了打破这个僵局而生的。它是Kubernetes在v1.5引入的**基于gRPC的标准化接口协议**，将容器运行时从kubelet源码中彻底解耦。任何实现了CRI接口的运行时，都可以作为Kubernetes的容器后端，而kubelet无需做任何修改。

### CRI的接口设计

CRI通过Unix Socket暴露两组gRPC服务：

```
RuntimeService
  ├── RunPodSandbox / StopPodSandbox / RemovePodSandbox
  ├── CreateContainer / StartContainer / StopContainer / RemoveContainer
  └── ExecSync / Exec / Attach / PortForward

ImageService
  ├── PullImage / ListImages / RemoveImage
  └── ImageStatus / ImageFsInfo
```

这里有一个设计上的核心概念值得深入理解：**PodSandbox**。在CRI模型中，Pod是一等公民——kubelet不是直接让运行时创建容器，而是先创建一个"沙箱"，再在沙箱中创建业务容器。沙箱负责持有Pod的网络命名空间，这也是为什么同一个Pod内的容器可以通过localhost互通。在大多数实现中，沙箱对应的就是那个神秘的pause容器。

## K8s v1.24为什么弃用Docker

### 一个绕弯的调用链

Docker从未原生实现CRI接口。为了让Docker能在Kubernetes中工作，团队在kubelet内部维护了一个叫**dockershim**的适配层。于是调用链变成了这样：

```
kubelet → dockershim → Docker Engine (dockerd) → containerd → runc
```

这里有一个关键讽刺：Docker Engine本身的底层运行时就是containerd。kubelet绕了dockershim和dockerd两层，最终还是找到了containerd。这两层不仅增加了延迟，还带来了两个额外的故障点，同时dockershim的维护负担完全由Kubernetes团队承担。

v1.20宣布废弃，v1.24正式移除。移除后，直接调用链变为：

```
kubelet → containerd（CRI原生）→ runc
```

一个容易被误解的事实：**弃用Docker对镜像毫无影响**。`docker build`构建的镜像遵循OCI标准，containerd、CRI-O都能直接使用。被弃用的只是Docker Engine作为Kubernetes容器运行时的角色。

## OCI规范：容器世界的通用语言

理解CRI必须先理解OCI（Open Container Initiative）。CRI定义的是kubelet和运行时之间的接口，而OCI定义的是运行时和操作系统之间的接口。

OCI由两个核心规范构成：

### OCI Image Spec：镜像的三层结构

一个OCI镜像由三个部分构成：

```
OCI Image
├── manifest.json      ← 清单，指向config和layers
├── config.json        ← 镜像配置（环境变量、入口命令、架构）
└── layers/            ← 按顺序叠加的文件系统层（tar格式）
    ├── sha256:abc...  ← 基础层
    ├── sha256:def...  ← 中间层
    └── sha256:xyz...  ← 最终层
```

manifest是入口，它用内容寻址（SHA256）引用config和每个layer。这种设计的精妙之处在于：不同镜像可以**共享相同的layer**，只要内容哈希相同，运行时就不会重复下载或存储，这是容器镜像分发效率极高的根本原因。

### OCI Runtime Spec：容器创建的规范书

当运行时要启动一个容器时，它需要一个**OCI Bundle**：一个包含rootfs目录和config.json文件的目录。config.json描述了容器应该有什么样的隔离环境：

- **namespaces**：指定需要创建或加入哪些Linux命名空间（pid、net、mnt、uts、ipc、user）
- **cgroups**：资源限制（CPU、内存、IO）
- **mounts**：需要挂载的文件系统（/proc、/sys、cgroup等）
- **capabilities**：赋予或移除的Linux能力

runc读取这个config.json，然后通过`clone()`系统调用创建子进程，传入相应的`CLONE_NEWPID`、`CLONE_NEWNET`等标志，再通过`unshare()`进一步隔离，最后在新的命名空间内执行容器的入口命令。OCI Runtime Spec就是这张"施工图"，runc是按图施工的工人。

## containerd的内部架构

containerd是目前使用最广泛的CRI运行时，也是GKE、EKS、AKS等主流云平台的默认选择。理解它的内部架构，对于生产环境的排障和优化至关重要。

### containerd-shim：为什么需要一个中间进程

containerd daemon是整个节点上所有容器的管理者。如果容器进程直接作为containerd的子进程运行，会产生一个严重的问题：**一旦containerd重启（升级、崩溃），所有容器进程都会随之消失**。

containerd-shim就是为了解决这个父子进程绑定问题而存在的。它的工作机制如下：

```
containerd daemon
    │
    ├── 创建 containerd-shim-runc-v2 进程
    │       │
    │       ├── 调用 runc create 创建容器
    │       ├── runc 退出后，shim 成为容器进程的父进程
    │       └── 负责转发标准输入输出（tty/pipe）
    │
    └── 与 shim 断开父子关系（double-fork 技术）
            容器进程此时的父进程是 shim，而非 containerd
```

当containerd重启时，它通过Unix Socket重新连接到各个shim进程，重新"认领"容器。容器进程完全不受影响，这就实现了**运行时daemon与容器生命周期的解耦**。

shim进程还承担另一个职责：收集容器的退出状态（exit code），在容器退出后暂存这个状态，直到containerd来读取。这就是为什么即使容器已经停止，你仍然能用crictl inspect查到它的退出码。

### 快照管理器：镜像层的存储原理

containerd通过**Snapshotter**管理容器的文件系统层。生产环境中最常用的是overlayfs snapshotter。

overlayfs的核心思想是将多个目录叠加为一个统一视图：

```
容器视图（merged）
     │
     ├── upperdir：容器可写层（COW，写时复制）
     └── lowerdir：只读的镜像层（可以多层叠加）
         ├── 镜像layer 3（最上层）
         ├── 镜像layer 2
         └── 镜像layer 1（基础层）
```

当容器修改一个文件时，overlayfs先将该文件从lowerdir**复制到upperdir**，再在upperdir中修改。删除操作则在upperdir中创建一个"whiteout"标记文件。lowerdir中的原始文件始终不会被修改，这使得同一镜像的多个容器实例可以共享相同的只读层，只有各自的upperdir是独立的。

containerd使用**命名空间（namespace）**隔离不同客户端的资源。这里要区分两个完全不同的命名空间概念：

| 概念 | 作用域 |
|------|--------|
| Linux namespace（pid/net/mnt等） | 进程级隔离，由内核提供 |
| containerd namespace | containerd内部的资源隔离，如k8s.io、moby |

Kubernetes默认使用`k8s.io`命名空间，Docker使用`moby`命名空间。这就是为什么在同一台机器上，`crictl images`和`docker images`看到的镜像列表不同——它们查看的是containerd中不同命名空间下的资源。

## CRI-O的内部架构

CRI-O（Container Runtime Interface + OCI）是专门为Kubernetes设计的轻量级运行时，2023年成为CNCF毕业项目，也是Red Hat OpenShift的默认选择。

### conmon：容器的专职监工

CRI-O中对应containerd-shim的组件是**conmon**（container monitor）。但conmon的设计哲学与shim有所不同：

- **每个容器一个conmon进程**：conmon直接是容器进程的父进程，负责监控容器生命周期
- **职责更聚焦**：仅负责监控容器、采集日志、传递退出状态，不负责容器创建
- **用C语言编写**：极小的内存占用（约1MB），降低节点整体开销

容器创建流程中，CRI-O调用runc（或其他OCI运行时）创建容器，然后由conmon接管监控职责。CRI-O daemon本身可以重启，conmon与容器的关系不会断开。

### containers/image与containers/storage

CRI-O使用两个来自containers项目的共享库处理镜像，这两个库同时也被podman、buildah等工具使用：

- **containers/image**：处理镜像拉取，支持docker-registry、OCI、直接复制等多种传输协议，内置镜像签名验证
- **containers/storage**：管理镜像存储和容器层，支持overlayfs、vfs等多种后端

这种共享库的设计使得CRI-O与整个containers生态高度一致。在OpenShift环境中，podman构建的镜像、CRI-O运行的容器，使用的是同一套底层存储，镜像缓存可以互通。

### 为什么CRI-O版本与K8s严格对齐

CRI-O的版本策略非常明确：**CRI-O v1.X.Y 仅支持且仅用于 Kubernetes v1.X.Z**。这是一个有意为之的设计决策，而非限制。

每当Kubernetes的CRI API发生变化，CRI-O跟随同步更新，确保两者始终处于已验证的兼容状态。这与containerd的独立发布周期形成对比——containerd需要维护对多个K8s版本的兼容，而CRI-O只需专注于当前版本的完美支持。对于追求稳定性的OpenShift来说，这是一种更可预测的升级路径。

## 容器创建的完整调用链

理解了各个组件后，我们来看一次完整的容器创建过程。以containerd为例，从kubelet发出指令到容器实际运行：

```
kubelet
  │
  │ 1. gRPC: RunPodSandbox(PodSandboxConfig)
  ▼
containerd (CRI plugin)
  │
  │ 2. 创建 pause 容器的 OCI Bundle
  │    配置 net/pid/ipc 命名空间
  │
  │ 3. 启动 containerd-shim-runc-v2
  ▼
containerd-shim
  │
  │ 4. 调用 runc create（传入 OCI Bundle 路径）
  ▼
runc
  │
  │ 5. 解析 config.json
  │ 6. clone(CLONE_NEWNET | CLONE_NEWPID | ...)
  │ 7. 执行 /pause 进程（占位，持有命名空间）
  │ 8. runc 退出，shim 成为 pause 的父进程
  ▼
pause 容器运行中（PodSandbox 就绪）

  │ 9. gRPC: CreateContainer(业务容器配置)
  ▼
containerd (CRI plugin)
  │
  │ 10. 创建业务容器的 OCI Bundle
  │     net/ipc 命名空间：加入 pause 容器（不新建）
  │     pid 命名空间：视配置决定
  │
  │ 11. 复用或新建 shim 进程
  ▼
runc
  │
  │ 12. clone 并通过 setns() 加入 pause 的命名空间
  │ 13. 执行业务容器的 entrypoint
  ▼
业务容器运行中
```

这个时序揭示了几个关键设计：

**PodSandbox先于业务容器创建**。pause容器是命名空间的持有者，它的存在确保了即使业务容器全部重启，Pod的网络配置（IP地址）也不会改变。

**业务容器通过setns()加入命名空间，而非新建**。这是同一Pod内容器网络共享的底层机制——它们并不是"网络相似"，而是字面意义上共享同一个Linux网络命名空间。

## 沙箱运行时：当runc的隔离不够用

标准的runc通过Linux命名空间和cgroups实现隔离，但这种隔离建立在**共享宿主机内核**的基础上。内核漏洞（如容器逃逸CVE）可能让攻击者突破命名空间的边界。对于多租户SaaS、边缘计算、运行不可信代码等场景，需要更强的隔离手段。

### Kata Containers：用虚拟机包裹容器

Kata Containers通过KVM虚拟化为每个Pod提供一个轻量级虚拟机，容器运行在虚拟机内部。攻击者即使突破容器，也只能到达虚拟机的内核，而非宿主机内核。

```
宿主机
  └── KVM 虚拟机（每个Pod一个）
        └── 轻量级内核（专为容器优化）
              └── 容器进程
```

代价是可见的：每个Pod的启动时间增加约100-200ms，内存开销额外增加约50MB（虚拟机内核）。但在安全要求高的场景下，这是值得的代价。

### gVisor：用户态内核拦截系统调用

gVisor的思路截然不同。它在用户态实现了一个**沙箱内核**（Sentry），拦截容器发出的所有系统调用，翻译后再以安全的方式调用宿主机内核。

```
容器进程
  │
  │ 系统调用（open/read/write/clone...）
  ▼
gVisor Sentry（用户态，Go编写）
  │ 验证参数，过滤危险调用
  ▼
宿主机内核（接触面极小）
```

gVisor的攻击面极小：宿主机内核只需暴露约50个系统调用给Sentry，而Sentry向容器暴露完整的Linux系统调用ABI。代价是系统调用密集型应用（如大量I/O操作）的性能下降约10-30%，但对于CPU密集型应用影响微乎其微。

### RuntimeClass：在K8s中配置多运行时

K8s通过**RuntimeClass**资源允许在同一集群中使用多种运行时：

```yaml
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: kata
handler: kata-qemu   # 对应节点上 containerd 中配置的 runtime handler

---
apiVersion: v1
kind: Pod
spec:
  runtimeClassName: kata   # 指定使用 kata 运行时
  containers:
  - name: app
    image: nginx
```

containerd通过配置文件中的`[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-qemu]`段注册额外的运行时handler。kubelet在创建Pod时，根据RuntimeClass找到对应的handler名称，通过CRI传递给containerd，containerd再选择对应的OCI运行时（kata-runtime而非runc）来创建容器。

## 生产环境选型

### containerd vs CRI-O：怎么选

| 维度 | containerd | CRI-O |
|------|-----------|-------|
| 使用场景 | K8s及非K8s均可 | 仅K8s |
| 版本管理 | 独立发布，需验证兼容性 | 与K8s版本强绑定，升级路径清晰 |
| 生态集成 | Docker、nerdctl、ECS等 | podman、buildah、OpenShift |
| 代码量/攻击面 | 较大，功能丰富 | 精简，攻击面小 |
| 社区支持 | CNCF，云厂商主导 | CNCF，Red Hat主导 |
| 调试工具 | crictl、nerdctl | crictl |

**选containerd的场景**：大多数公有云托管K8s（EKS/GKE/AKS默认）、需要在K8s之外也使用containerd API的场景（如自定义镜像构建pipeline）、使用nerdctl的开发环境。

**选CRI-O的场景**：OpenShift或基于Red Hat生态的集群、极度注重安全和攻击面最小化、希望运行时升级与K8s升级严格对齐、已在使用podman/buildah的团队。

### crictl：生产环境的排查工具

crictl是CRI的标准命令行客户端，直接通过CRI接口与运行时通信，适用于containerd和CRI-O：

```bash
# 列出所有容器（包括pause容器）
crictl ps -a

# 列出镜像
crictl images

# 查看Pod（沙箱）列表
crictl pods

# 进入容器
crictl exec -it <container-id> sh

# 拉取镜像
crictl pull nginx:latest

# 查看容器详细信息（含退出码、重启原因）
crictl inspect <container-id>

# 查看容器日志
crictl logs <container-id>
```

crictl的价值在于它**绕过了kubectl和API Server**，直接与节点上的运行时通信。当API Server不可用、Pod卡在Pending/Unknown状态、或需要排查节点级别的容器问题时，crictl是唯一可靠的工具。

## 小结

- **CRI是解耦的关键**：将运行时从kubelet中剥离，实现了容器运行时的可插拔
- **OCI是通用语言**：Image Spec定义镜像格式（manifest+config+layers），Runtime Spec定义容器创建规范（config.json）
- **shim/conmon是解耦的保障**：确保运行时daemon重启不影响容器生命周期
- **overlayfs是性能基础**：写时复制机制使多容器共享镜像层成为可能
- **PodSandbox先行**：pause容器持有命名空间，是同Pod容器网络共享的基础
- **沙箱运行时补充安全边界**：kata用VM隔离，gVisor用用户态内核拦截，RuntimeClass统一管理

---

## 常见问题

### Q1：containerd和Docker的关系是什么？弃用Docker后还能用Docker构建镜像吗？

Docker Engine在其架构演进过程中，将核心的容器管理功能剥离为独立项目containerd，Docker Engine自身更多地扮演用户界面和build工具的角色。在Kubernetes场景中，v1.24之前通过dockershim调用Docker Engine，最终还是落到containerd；v1.24之后直接调用containerd，中间层消失。

对镜像构建毫无影响。`docker build`生成的镜像遵循OCI Image Spec，containerd和CRI-O都能直接使用。你仍然可以在CI/CD中用Docker（或更高效的BuildKit、kaniko）构建镜像，推送到镜像仓库，再由Kubernetes拉取运行。

### Q2：为什么pause容器（Infra容器）对于Kubernetes至关重要？

pause容器的核心价值在于**命名空间的生命周期管理**。Linux命名空间与进程绑定，进程退出则命名空间销毁。如果直接让业务容器持有网络命名空间，一旦业务容器重启，网络命名空间就会消失，IP地址丢失，其他容器的网络连接也会中断。

pause进程极其简单，它只是调用`pause()`系统调用无限等待信号，几乎不消耗CPU。它的存在仅仅是为了持有命名空间。只要pause容器不退出，Pod的IP地址和网络配置就保持稳定，业务容器可以随意重启而不影响Pod的网络身份。

### Q3：overlayfs与devicemapper在生产环境中如何选择？

overlayfs是目前几乎所有生产环境的首选。它基于文件系统层面的联合挂载，对内核版本要求较低（4.0+普遍可用），性能优秀，运维简单。创建容器就是在镜像层上加一个upperdir，几乎零开销。

devicemapper在历史上是RHEL/CentOS早期版本的选择（overlayfs当时未在RHEL内核中支持）。它工作在块设备层面，需要单独划分LVM卷组，配置复杂，且loop-lvm模式在生产中性能很差。除非你的内核版本过于陈旧不支持overlayfs，否则没有理由选择devicemapper。

现代内核（5.x+）上，zfs snapshotter也是一个值得关注的选项，提供了更好的写放大特性和快照管理能力，但需要额外安装ZFS内核模块。

### Q4：kata-containers和gVisor分别适合什么工作负载，性能影响有多大？

两者在隔离机制和性能取舍上有本质区别。

**kata-containers**的性能开销主要体现在启动时间（需要启动VM，额外100-200ms）和内存开销（每个Pod约50MB的VM内核开销）。一旦容器运行起来，系统调用直接在VM内核处理，没有额外的拦截层，对CPU密集型和I/O密集型工作负载的运行时性能影响都很小（<5%）。适合需要强隔离但对启动速度不敏感的场景，如FaaS函数运行时、CI任务执行器。

**gVisor**对系统调用密集型应用影响较大（10-30%），因为每个系统调用都要经过Sentry的用户态处理。但对于纯计算型工作负载影响极小。gVisor的优势在于内存开销极低，不需要VM，适合需要运行大量小型、不可信、系统调用不频繁的工作负载（如代码评测、文档解析服务）。

### Q5：升级Kubernetes版本时，CRI运行时需要如何配合升级？

使用containerd时，由于其版本独立于K8s，需要关注兼容性矩阵。通常一个K8s版本支持两到三个containerd大版本。升级K8s前，先查阅目标K8s版本的CRI要求，确认当前containerd版本在支持范围内。如果需要升级containerd，由于containerd-shim与daemon解耦，可以先停止containerd、升级二进制、重新启动，**运行中的容器不受影响**，这是滚动升级的重要保障。

使用CRI-O时，由于版本与K8s严格对齐，策略更简单：升级K8s到v1.X就必须同时升级CRI-O到v1.X，不存在版本兼容性研究的问题。无论哪种运行时，节点排空（kubectl drain）后再升级是最安全的操作流程，确保升级过程中没有Pod在该节点上运行。

## 参考资源

- [Kubernetes CRI 官方文档](https://kubernetes.io/docs/concepts/architecture/cri/)
- [containerd 官方文档](https://containerd.io/docs/)
- [OCI Runtime Spec 规范](https://github.com/opencontainers/runtime-spec)

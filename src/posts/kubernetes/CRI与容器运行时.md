---
date: 2026-02-06
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

## 什么是CRI

CRI（Container Runtime Interface，容器运行时接口）是Kubernetes在**v1.5（2016年12月）** 引入的一套**基于gRPC的标准化接口协议**，用于定义kubelet与容器运行时之间的通信规范。

在CRI出现之前，Docker、rkt等运行时的代码**直接硬编码**在kubelet源码中。每增加一个新的运行时，就需要修改kubelet代码并重新编译，维护成本极高。CRI的本质是将容器运行时**从kubelet中解耦**，让任何实现了CRI接口的运行时都能无缝接入Kubernetes。

### CRI的接口设计

CRI通过Unix Socket暴露gRPC服务，定义了两组核心API：

**RuntimeService** —— 负责容器和Pod沙箱的生命周期管理：

- `RunPodSandbox` / `StopPodSandbox` / `RemovePodSandbox`：管理Pod沙箱（网络命名空间）
- `CreateContainer` / `StartContainer` / `StopContainer` / `RemoveContainer`：管理容器生命周期

**ImageService** —— 负责镜像管理：

- `PullImage` / `ListImages` / `RemoveImage`：镜像的拉取、查询与删除

kubelet作为gRPC客户端，向实现了CRI的运行时（gRPC服务端）发起调用，运行时收到请求后执行具体的容器操作。

## K8s v1.24为什么弃用Docker

### 架构上的根本矛盾

Docker**从未原生实现CRI接口**。为了让Docker能在Kubernetes中工作，Kubernetes团队在kubelet内部维护了一个叫**dockershim**的适配层，充当kubelet与Docker之间的翻译器。

使用Docker时，一次容器创建的调用链是这样的：

```
kubelet → dockershim → Docker Engine (dockerd) → containerd → runc
```

这里存在一个关键事实：Docker Engine本身就是用containerd作为底层运行时。也就是说，kubelet绕了一大圈，经过dockershim和Docker Engine两个中间层，最终还是调用了containerd。

### 弃用时间线

| 时间 | 事件 |
|------|------|
| 2020年12月 | K8s v1.20 宣布弃用dockershim（KEP-2221） |
| 2022年5月 | K8s v1.24 正式移除dockershim |

### 弃用的核心原因

1. **维护负担**：dockershim代码由Kubernetes团队维护，不属于Docker项目，增加了不必要的维护成本
2. **性能开销**：多出dockershim和dockerd两层调用，增加了延迟和故障点
3. **功能受限**：cgroups v2、用户命名空间等新特性难以通过dockershim高效透传
4. **架构冗余**：既然最终都是调用containerd，直接让kubelet通过CRI与containerd通信更加合理

移除dockershim后，调用链变为：

```
kubelet → containerd（CRI原生） → runc
```

> **注意**：弃用Docker不影响Docker构建的镜像。所有通过`docker build`构建的镜像都符合OCI标准，可以在任何CRI运行时中正常运行。

## 主流容器运行时

### containerd

containerd是从Docker项目中剥离出来的**通用容器运行时**，2019年成为CNCF毕业项目。

**架构特点**：

- 通过**内置CRI插件**直接实现CRI接口，kubelet可直接通过gRPC与其通信
- 使用**containerd-shim**进程管理容器，确保容器进程与containerd守护进程解耦
- 采用插件化架构，支持快照、存储、网络等功能扩展

**适用场景**：containerd是目前使用最广泛的CRI运行时，也是大多数云厂商（AWS EKS、GKE、AKS）的默认选择。它不仅服务于Kubernetes，还被Docker Engine、Amazon ECS等平台使用。

### CRI-O

CRI-O是**专门为Kubernetes设计**的轻量级容器运行时，2023年成为CNCF毕业项目。

**架构特点**：

- 只实现Kubernetes所需的功能，不包含任何多余能力
- 使用**conmon**（container monitor）进程监控容器，职责类似containerd-shim
- 利用`containers/image`库拉取镜像，`containers/storage`库管理存储
- **版本号与Kubernetes严格对齐**：CRI-O v1.30.x 对应 K8s v1.30.x

**适用场景**：CRI-O是Red Hat OpenShift的默认运行时，适合追求极简和安全的场景。更小的代码量意味着更小的攻击面。

### 对比总结

| 维度 | containerd | CRI-O |
|------|-----------|-------|
| 定位 | 通用容器运行时 | Kubernetes专用运行时 |
| 使用范围 | K8s、Docker、ECS等 | 仅Kubernetes |
| 进程管理 | containerd-shim | conmon |
| 版本策略 | 独立发布周期 | 与K8s版本严格对齐 |
| 代码规模 | 较大，功能丰富 | 精简，攻击面小 |
| 默认使用方 | 大多数云厂商 | Red Hat OpenShift |

两者底层都调用符合**OCI（Open Container Initiative）标准**的运行时（如runc）来创建容器，也都使用OCI镜像格式。选择哪个运行时取决于你的实际需求——需要通用性选containerd，追求极简选CRI-O。

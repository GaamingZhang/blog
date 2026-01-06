---
date: 2026-01-06
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Pod
  - 调度流程
---

# Pod被创建调度到Node的全过程

## 1. 引言和概述

在Kubernetes中，Pod是最基本的部署单元，了解Pod从创建到调度到Node的全过程对于理解Kubernetes的工作原理至关重要。这一过程涉及多个Kubernetes组件的协同工作，包括API服务器、调度器、控制器管理器等。

### 1.1 为什么需要了解Pod调度过程？

- **故障排查**：当Pod无法启动或调度失败时，了解调度过程有助于快速定位问题
- **性能优化**：理解调度策略可以帮助优化集群资源利用率
- **资源规划**：合理配置Node资源和Pod资源请求/限制
- **安全合规**：确保Pod被调度到符合安全策略的Node上

### 1.2 Pod调度的大致流程

Pod从创建到最终在Node上运行的过程可以概括为以下几个主要阶段：

1. **Pod创建**：用户或控制器通过API服务器创建Pod资源
2. **验证和准入控制**：API服务器验证Pod配置并执行准入控制
3. **调度决策**：调度器选择最合适的Node来运行Pod
4. **绑定**：将Pod绑定到选定的Node
5. **容器运行**：Node上的kubelet负责启动Pod中的容器

在接下来的章节中，我们将详细探讨每个阶段的具体实现和涉及的组件。

## 2. Pod创建流程

Pod的创建通常由用户直接发起，或通过控制器（如Deployment、StatefulSet、DaemonSet等）间接创建。无论哪种方式，最终都需要通过Kubernetes API服务器来完成。

### 2.1 API服务器的角色

Kubernetes API服务器是集群的入口点，负责处理所有API请求。当创建Pod时，API服务器会执行以下操作：

1. **接收请求**：接收来自kubectl、客户端库或其他组件的Pod创建请求
2. **验证请求**：验证请求的格式、参数和权限
3. **存储到etcd**：将Pod的配置信息存储到etcd数据存储中
4. **返回响应**：向客户端返回操作结果

### 2.2 Pod创建的具体步骤

#### 2.2.1 用户发起Pod创建请求

用户可以通过kubectl命令行工具、API客户端或Kubernetes Dashboard发起Pod创建请求：

```bash
# 通过kubectl创建Pod
kubectl run nginx --image=nginx:latest

# 通过yaml文件创建Pod
kubectl apply -f pod.yaml
```

#### 2.2.2 API请求的处理

API服务器接收到请求后，会执行以下验证和处理步骤：

1. **认证（Authentication）**：验证请求发起者的身份
2. **授权（Authorization）**：检查请求者是否有权限执行Pod创建操作
3. **准入控制（Admission Control）**：执行一系列准入控制器来验证和修改Pod配置
4. **对象验证**：验证Pod对象的结构和字段是否符合API规范
5. **存储到etcd**：将验证通过的Pod配置存储到etcd中

#### 2.2.3 准入控制器

准入控制器是API服务器的一部分，用于在对象创建或更新时执行额外的验证和修改。与Pod创建相关的主要准入控制器包括：

- **NamespaceExists**：检查Pod所属的命名空间是否存在
- **LimitRanger**：确保Pod的资源请求和限制符合命名空间的限制
- **ResourceQuota**：确保Pod的创建不会超出命名空间的资源配额
- **PodSecurity**：验证Pod是否符合指定的安全策略
- **MutatingAdmissionWebhooks**：可以修改Pod配置，如添加sidecar容器

### 2.3 Pod初始状态

当Pod成功创建并存储到etcd后，其初始状态为`Pending`，表示Pod正在等待调度。此时，Pod的`spec.nodeName`字段为空，因为还没有被调度到特定的Node上。

## 3. Pod调度流程

当Pod处于`Pending`状态且`spec.nodeName`字段为空时，调度器会开始工作，为Pod选择一个合适的Node。

### 3.1 调度器的角色

Kubernetes调度器（kube-scheduler）是一个独立的组件，负责为新创建的Pod选择最合适的Node。调度器的核心功能是：

1. **监听Pod创建事件**：持续监听API服务器，发现需要调度的Pod
2. **收集Node资源信息**：获取集群中所有Node的资源使用情况和状态
3. **执行调度算法**：根据Pod的资源需求和Node的可用资源选择最佳Node
4. **绑定Pod到Node**：将调度结果更新到API服务器

### 3.2 调度算法的两个阶段

调度器的调度过程分为两个主要阶段：

#### 3.2.1 过滤阶段（Filtering）

过滤阶段的目标是从所有Node中筛选出满足Pod资源需求和约束条件的Node，这些Node称为"可行Node"（Feasible Nodes）。

主要的过滤规则包括：

1. **PodFitsResources**：检查Node是否有足够的可用资源（CPU、内存等）来满足Pod的资源请求
2. **PodFitsHostPorts**：检查Pod请求的主机端口是否已被使用
3. **PodFitsHost**：如果Pod指定了`spec.host`，检查Node的主机名是否匹配
4. **PodMatchNodeSelector**：检查Node是否匹配Pod的节点选择器
5. **NoVolumeZoneConflict**：检查Pod使用的卷是否与Node所在的区域兼容
6. **CheckNodeMemoryPressure**：检查Node是否存在内存压力
7. **CheckNodeDiskPressure**：检查Node是否存在磁盘压力
8. **CheckNodePIDPressure**：检查Node的PID资源是否充足
9. **CheckNodeCondition**：检查Node的状态是否正常（Ready状态）
10. **PodToleratesNodeTaints**：检查Pod是否能容忍Node的污点

#### 3.2.2 打分阶段（Scoring）

打分阶段的目标是从可行Node中选择一个最合适的Node。调度器会为每个可行Node计算一个分数，分数最高的Node将被选中。

主要的打分规则包括：

1. **LeastRequestedPriority**：优先选择资源使用率最低的Node
   - 计算公式：(cpu((capacity-sum(requested))10/capacity) + memory((capacity-sum(requested))10/capacity))/2
2. **BalancedResourceAllocation**：优先选择资源使用最均衡的Node
3. **NodeAffinityPriority**：根据节点亲和性规则进行打分
4. **PodAffinityPriority**：根据Pod亲和性规则进行打分
5. **PodAntiAffinityPriority**：根据Pod反亲和性规则进行打分
6. **TaintTolerationPriority**：根据Pod对Node污点的容忍度进行打分
7. **ImageLocalityPriority**：优先选择已经缓存了Pod所需镜像的Node
8. **ServiceSpreadingPriority**：尽量将同一Service的Pod分散到不同的Node上
9. **EqualPriority**：给所有Node打相同的分数

### 3.3 调度决策的执行

当调度器完成打分后，会选择分数最高的Node作为Pod的目标Node。如果有多个Node分数相同，调度器会随机选择一个。

调度决策完成后，调度器会通过API服务器将Pod的`spec.nodeName`字段更新为选定的Node名称，这个过程称为"绑定"（Binding）。

## 4. Pod绑定和运行过程

一旦Pod被绑定到特定的Node，Node上的kubelet组件就会接管后续的工作，负责启动Pod中的容器。

### 4.1 kubelet的角色

kubelet是运行在每个Node上的代理，负责管理Node上的Pod和容器。当kubelet发现有新的Pod绑定到自己所在的Node时，会执行以下操作：

1. **拉取Pod配置**：从API服务器获取Pod的完整配置
2. **挂载卷**：为Pod挂载所需的存储卷
3. **拉取镜像**：从镜像仓库拉取Pod所需的容器镜像
4. **启动容器**：使用容器运行时（如containerd、CRI-O）启动Pod中的容器
5. **监控容器**：持续监控容器的状态，并向API服务器报告

### 4.2 容器运行时接口（CRI）

Kubernetes通过容器运行时接口（Container Runtime Interface，CRI）与不同的容器运行时交互。kubelet通过CRI接口请求容器运行时执行以下操作：

1. **创建容器**：创建Pod中的各个容器
2. **启动容器**：启动创建好的容器
3. **停止容器**：当Pod被删除时停止容器
4. **删除容器**：清理不再需要的容器

### 4.3 Pod的状态变化

在kubelet启动容器的过程中，Pod的状态会经历以下变化：

1. **Pending**：Pod已创建但未被调度，或已被调度但容器尚未启动
2. **ContainerCreating**：kubelet正在创建容器
3. **Running**：Pod中的所有容器都已启动并运行正常
4. **Succeeded**：Pod中的所有容器都已成功完成（适用于一次性任务）
5. **Failed**：Pod中的一个或多个容器失败

## 5. Pod生命周期管理

Pod在其生命周期中会经历多个状态转换，同时可能会触发一些生命周期钩子。

### 5.1 Pod生命周期阶段

Pod的生命周期可以分为以下几个主要阶段：

1. **Pending**：Pod已创建但未被调度，或已被调度但容器尚未启动
2. **Running**：Pod中的所有容器都已启动并运行正常
3. **Succeeded**：Pod中的所有容器都已成功完成
4. **Failed**：Pod中的一个或多个容器失败
5. **Unknown**：kubelet无法与API服务器通信，无法获取Pod状态

### 5.2 容器生命周期钩子

容器在其生命周期中有两个主要钩子：

1. **PostStart**：容器启动后立即执行
2. **PreStop**：容器停止前执行

这些钩子可以用于执行一些特定的操作，如初始化数据、清理资源等。

### 5.3 重启策略

Pod可以配置不同的重启策略（RestartPolicy）：

1. **Always**：总是重启失败的容器（默认值）
2. **OnFailure**：仅在容器失败时重启
3. **Never**：从不重启容器

重启策略会影响Pod的生命周期和自动修复能力。

## 6. 常见问题解答

### Q1: Pod一直处于Pending状态怎么办？

**A1:** Pod处于Pending状态通常是因为调度失败。可以通过以下步骤排查：

1. 查看Pod事件：`kubectl describe pod <pod-name>`
2. 检查调度器日志：`kubectl logs -n kube-system <scheduler-pod-name>`
3. 验证资源请求是否超过Node可用资源
4. 检查节点选择器、亲和性/反亲和性规则是否与可用Node匹配
5. 检查Taint和Toleration配置

### Q2: 如何强制将Pod调度到特定的Node？

**A2:** 可以使用以下方法将Pod调度到特定Node：

1. **节点选择器（NodeSelector）**：在Pod配置中添加`nodeSelector`字段
   ```yaml
   spec:
     nodeSelector:
       kubernetes.io/hostname: node-1
   ```

2. **节点亲和性（NodeAffinity）**：更灵活的节点选择方式
   ```yaml
   spec:
     affinity:
       nodeAffinity:
         requiredDuringSchedulingIgnoredDuringExecution:
           nodeSelectorTerms:
           - matchExpressions:
             - key: kubernetes.io/hostname
               operator: In
               values:
               - node-1
   ```

3. **节点名称（NodeName）**：直接指定Node名称，绕过调度器
   ```yaml
   spec:
     nodeName: node-1
   ```

### Q3: 调度器如何处理资源不足的情况？

**A3:** 当集群中所有Node的资源都不足以满足Pod的资源请求时，Pod将一直处于Pending状态。调度器会持续监控集群资源，一旦有足够的资源可用，就会立即调度这些Pending的Pod。

为了避免这种情况，可以采取以下措施：

1. 合理设置Pod的资源请求和限制
2. 扩容集群，添加更多Node
3. 优化现有Node的资源利用率
4. 使用Cluster Autoscaler自动扩容集群

### Q4: 如何查看调度器的详细日志？

**A4:** 可以通过以下命令查看调度器的日志：

```bash
# 获取调度器Pod名称
scheduler_pod=$(kubectl get pods -n kube-system | grep kube-scheduler | awk '{print $1}')

# 查看调度器日志
kubectl logs -n kube-system $scheduler_pod

# 实时查看调度器日志
kubectl logs -n kube-system $scheduler_pod -f

# 查看特定时间段的日志
kubectl logs -n kube-system $scheduler_pod --since=1h
```

### Q5: 如何配置自定义调度策略？

**A5:** Kubernetes支持多种方式配置自定义调度策略：

1. **调度器配置文件**：通过配置文件定制过滤和打分规则
   ```yaml
   apiVersion: kubescheduler.config.k8s.io/v1
   kind: KubeSchedulerConfiguration
   profiles:
   - schedulerName: default-scheduler
     plugins:
       filter:
         enabled:
         - name: "PodFitsResources"
         - name: "PodFitsHostPorts"
       score:
         enabled:
         - name: "LeastRequestedPriority"
           weight: 1
         - name: "BalancedResourceAllocation"
           weight: 1
   ```

2. **调度器扩展**：开发自定义调度器或使用调度器扩展（Scheduler Extender）

3. **调度器插件**：使用Kubernetes 1.19+支持的调度器插件框架

## 7. 总结

Pod从创建到调度到Node的全过程涉及多个Kubernetes组件的协同工作，包括API服务器、调度器、kubelet等。理解这一过程对于Kubernetes管理员和开发人员来说至关重要，有助于故障排查、性能优化和资源规划。

主要步骤包括：

1. **Pod创建**：通过API服务器创建Pod资源
2. **验证和准入控制**：API服务器验证配置并执行准入控制
3. **调度决策**：调度器通过过滤和打分选择最合适的Node
4. **绑定**：将Pod绑定到选定的Node
5. **容器运行**：kubelet负责启动和监控容器

通过合理配置Pod的资源请求/限制、节点选择器、亲和性/反亲和性规则等，可以优化Pod的调度和运行效果。同时，了解常见问题的排查方法和解决方案，可以提高集群的可靠性和稳定性。

# Kubernetes获取资源的机制是什么

## 1. 引言

在Kubernetes集群中，资源获取是一个核心且复杂的过程，它涉及从资源请求的提出到资源最终分配和使用的完整生命周期。理解Kubernetes的资源获取机制对于集群管理员和应用开发者来说至关重要，它直接影响着应用的性能、稳定性和资源利用率。

### 1.1 资源的概念

在Kubernetes中，**资源**指的是集群中可被Pod使用的计算能力和存储能力，主要包括：

- **计算资源**：CPU、内存（Memory）
- **存储资源**：临时存储（Ephemeral Storage）、持久卷（Persistent Volume）
- **扩展资源**：GPU、FPGA等特殊硬件资源

### 1.2 资源获取机制的重要性

Kubernetes的资源获取机制确保了：
1. **资源的公平分配**：避免个别应用占用过多资源导致其他应用饥饿
2. **应用的稳定性**：通过资源限制防止应用过度消耗资源导致节点故障
3. **集群的高效利用**：通过资源调度算法优化资源分配，提高集群利用率
4. **多租户隔离**：通过资源配额实现不同命名空间间的资源隔离

## 2. Kubernetes资源模型

### 2.1 资源类型

Kubernetes中的资源可以分为两大类：

#### 2.1.1 可压缩资源（Compressible Resources）
- **CPU**：以CPU核心数或毫核（milli-cores）为单位，1核=1000m
- 特点：使用过量时会被限制（throttled），但不会导致Pod被终止

#### 2.1.2 不可压缩资源（Incompressible Resources）
- **内存**：以字节为单位（如Mi、Gi）
- **临时存储**：以字节为单位
- 特点：使用过量时会导致Pod被终止（OOM - Out of Memory）

#### 2.1.3 扩展资源（Extended Resources）
- GPU、FPGA、高性能网卡等特殊硬件
- 需要设备插件（Device Plugin）支持

### 2.2 资源定义

在Pod的YAML配置中，资源通过`resources`字段定义：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: resource-demo
spec:
  containers:
  - name: demo-container
    image: nginx
    resources:
      requests:  # 资源请求：调度器用于决定将Pod调度到哪个节点
        cpu: "100m"  # 100毫核 = 0.1核
        memory: "128Mi"  # 128兆内存
      limits:  # 资源限制：运行时限制Pod可以使用的最大资源
        cpu: "200m"  # 最大使用200毫核
        memory: "256Mi"  # 最大使用256兆内存
```

## 3. 资源获取的核心组件

Kubernetes的资源获取机制涉及多个核心组件的协同工作：

### 3.1 API Server
- **作用**：接收并验证资源请求，将资源状态存储在etcd中
- **功能**：
  - 处理用户的资源请求（如创建Pod、Deployment等）
  - 验证资源请求的合法性
  - 维护资源的期望状态

### 3.2 Scheduler
- **作用**：负责将Pod调度到合适的节点
- **功能**：
  - 过滤（Filtering）：筛选出满足Pod资源请求的节点
  - 评分（Scoring）：对筛选后的节点进行评分，选择最优节点
  - 绑定（Binding）：将Pod与选定的节点绑定

### 3.3 Kubelet
- **作用**：运行在每个节点上，管理节点上的Pod和容器
- **功能**：
  - 从API Server获取Pod定义
  - 与容器运行时（如containerd、CRI-O）协作创建容器
  - 设置容器的资源限制
  - 监控容器的资源使用情况

### 3.4 Controller Manager
- **作用**：运行各种控制器，确保集群状态达到期望状态
- **相关控制器**：
  - Deployment Controller：管理无状态应用的资源
  - StatefulSet Controller：管理有状态应用的资源
  - ReplicaSet Controller：确保Pod副本数量符合期望

### 3.5 Container Runtime
- **作用**：负责容器的创建、运行和销毁
- **功能**：
  - 为容器设置资源限制（CPU、内存、存储）
  - 实现资源隔离
  - 提供资源使用的监控数据

## 4. 资源获取的完整工作流程

### 4.1 资源请求阶段

1. **用户请求**：用户通过kubectl或API Server创建资源（如Pod、Deployment）
2. **API Server验证**：验证请求的合法性和权限
3. **资源状态存储**：将资源的期望状态存储在etcd中

### 4.2 资源调度阶段

1. **调度器监听**：Scheduler持续监听API Server中未调度的Pod
2. **节点过滤**：
   - **PodFitsResources**：检查节点是否有足够的可用资源满足Pod的请求
   - **PodFitsHostPorts**：检查Pod请求的端口在节点上是否可用
   - **PodFitsHost**：检查Pod是否指定了特定节点
   - **MatchNodeSelector**：检查Pod的节点选择器是否与节点标签匹配
3. **节点评分**：
   - **LeastRequestedPriority**：优先选择资源使用率最低的节点
   - **BalancedResourceAllocation**：优先选择资源使用最均衡的节点
   - **NodeAffinityPriority**：考虑节点亲和性规则
   - **TaintTolerationPriority**：考虑节点污点和Pod容忍度
4. **节点绑定**：将Pod与得分最高的节点绑定

### 4.3 资源分配阶段

1. **Kubelet接收**：节点上的Kubelet接收到新的Pod绑定信息
2. **容器创建**：Kubelet与Container Runtime协作创建容器
3. **资源设置**：
   - **CPU限制**：使用cgroup的cpu.cfs_quota_us和cpu.cfs_period_us设置CPU限制
   - **内存限制**：使用cgroup的memory.limit_in_bytes设置内存限制
   - **存储限制**：使用cgroup的blkio和tmpfs限制临时存储
4. **资源使用监控**：Kubelet持续监控容器的资源使用情况

### 4.4 资源使用阶段

1. **资源消耗**：容器正常运行并消耗资源
2. **资源限制执行**：
   - CPU：当容器使用CPU超过限制时，会被内核限制CPU使用（throttled）
   - 内存：当容器使用内存超过限制时，会触发OOM killer终止容器
3. **资源监控报告**：Kubelet定期向API Server报告节点和Pod的资源使用情况

## 5. Kubernetes资源调度机制

### 5.1 调度器架构

Kubernetes Scheduler采用插件化架构，主要包括：

- **调度框架（Scheduling Framework）**：提供扩展点，允许开发者添加自定义调度逻辑
- **调度队列（Scheduling Queue）**：管理待调度的Pod，支持优先级和抢占
- **调度周期（Scheduling Cycle）**：单个Pod的调度过程
- **绑定周期（Binding Cycle）**：将Pod与节点绑定的过程

### 5.2 调度算法

#### 5.2.1 默认调度算法

Kubernetes默认使用的调度算法是**kube-scheduler**，它结合了多种调度策略：

1. **优先调度策略**：
   - **LeastRequestedPriority**：优先选择资源使用率最低的节点
   - **BalancedResourceAllocation**：优先选择CPU和内存使用率最均衡的节点
   - **NodeAffinityPriority**：考虑节点亲和性规则
   - **PodAffinityPriority**：考虑Pod亲和性和反亲和性规则
   - **TaintTolerationPriority**：考虑节点污点和Pod容忍度

2. **抢占调度策略**：
   - 当没有足够资源的节点时，调度器会尝试抢占低优先级Pod的资源
   - 抢占过程包括：选择目标节点、抢占低优先级Pod、重新调度被抢占的Pod

#### 5.2.2 自定义调度器

除了默认调度器外，Kubernetes还支持自定义调度器：

- **静态自定义调度器**：通过`spec.schedulerName`字段指定自定义调度器
- **动态调度器扩展**：通过调度框架插件扩展默认调度器功能

## 6. 资源配额和限制机制

### 6.1 资源请求与限制

#### 6.1.1 资源请求（Resource Requests）
- 定义：Pod运行所需的最小资源量
- 用途：调度器用于决定将Pod调度到哪个节点
- 影响：资源请求会影响节点的可调度性

#### 6.1.2 资源限制（Resource Limits）
- 定义：Pod运行时可以使用的最大资源量
- 用途：防止Pod过度消耗资源影响其他Pod
- 影响：超过限制会导致CPU被限制或内存OOM

### 6.2 资源配额（Resource Quotas）

Resource Quotas用于限制命名空间内的资源使用总量：

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-resources
  namespace: dev
spec:
  hard:
    requests.cpu: "10"
    requests.memory: 10Gi
    limits.cpu: "20"
    limits.memory: 20Gi
    pods: "100"
```

### 6.3 限制范围（Limit Ranges）

Limit Ranges用于限制命名空间内Pod和容器的默认资源请求和限制：

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: default-resources
  namespace: dev
spec:
  limits:
  - default:
      cpu: "1"
      memory: 512Mi
    defaultRequest:
      cpu: "500m"
      memory: 256Mi
    type: Container
```

## 7. 资源监控和回收机制

### 7.1 资源监控

Kubernetes提供了多种资源监控方式：

#### 7.1.1 Metrics Server
- 轻量级的集群资源监控组件
- 收集节点和Pod的CPU、内存使用率
- 为HPA（Horizontal Pod Autoscaler）提供数据支持

#### 7.1.2 Prometheus + Grafana
- 完整的监控解决方案
- 收集详细的资源使用指标
- 提供可视化的监控仪表板

#### 7.1.3 资源使用报告

Kubelet定期向API Server报告资源使用情况：
- **节点资源报告**：节点的总资源、已分配资源、可用资源
- **Pod资源报告**：Pod的资源请求、限制、实际使用情况

### 7.2 资源回收

#### 7.2.1 容器OOM回收
- 当容器使用内存超过限制时，内核OOM killer会终止容器
- Kubelet会根据Pod的重启策略决定是否重启容器

#### 7.2.2 节点压力驱逐（Node Pressure Eviction）
- 当节点资源不足时，Kubelet会启动驱逐流程
- 驱逐顺序：
  1. 内存压力：优先驱逐内存使用率高的Pod
  2. CPU压力：优先驱逐CPU使用率高的Pod
  3. 存储压力：优先驱逐使用临时存储多的Pod
- 驱逐策略：基于Pod的优先级和资源使用情况

#### 7.2.3 资源抢占
- 高优先级Pod可以抢占低优先级Pod的资源
- 抢占流程：
  1. 选择目标节点
  2. 驱逐低优先级Pod
  3. 将高优先级Pod调度到该节点

## 8. 常见问题（FAQ）

### 8.1 Kubernetes如何处理资源不足的情况？

当集群资源不足时，Kubernetes会采取以下措施：
1. **调度失败**：新的Pod会处于Pending状态，无法调度
2. **节点压力驱逐**：Kubelet会驱逐节点上低优先级的Pod
3. **资源抢占**：高优先级Pod可以抢占低优先级Pod的资源
4. **OOM终止**：当内存不足时，内核会终止使用内存最多的容器

### 8.2 如何优化Kubernetes集群的资源利用率？

优化资源利用率的方法包括：
1. **合理设置资源请求和限制**：根据应用实际需求设置，避免过度预留
2. **使用水平Pod自动扩缩容（HPA）**：根据资源使用率自动调整Pod副本数
3. **使用垂直Pod自动扩缩容（VPA）**：自动调整Pod的资源请求和限制
4. **节点自动扩缩容（Cluster Autoscaler）**：根据集群负载自动调整节点数量
5. **采用资源密集型和资源轻量型应用混合部署**：提高节点资源利用率

### 8.3 CPU请求和限制的单位"m"是什么意思？

在Kubernetes中，"m"表示毫核（milli-cores），1核=1000m。例如：
- 100m = 0.1核 = 10%的单个CPU核心
- 500m = 0.5核 = 50%的单个CPU核心
- 1000m = 1核 = 100%的单个CPU核心

### 8.4 为什么有时Pod的实际CPU使用率会超过限制？

CPU限制是通过cgroup的CPU限流机制实现的，它基于CPU时间片的分配。在短时间内，Pod的CPU使用率可能会超过限制，但从长期来看（如1秒），CPU使用率会被限制在设定的阈值内。这种现象称为"CPU突发（CPU Burst）"。

### 8.5 如何实现多租户环境下的资源隔离？

实现多租户资源隔离的方法包括：
1. **资源配额（Resource Quotas）**：限制每个命名空间的资源总量
2. **限制范围（Limit Ranges）**：限制Pod和容器的默认资源请求和限制
3. **Pod优先级和抢占**：确保高优先级应用获得足够资源
4. **节点亲和性和反亲和性**：将不同租户的Pod调度到不同节点
5. **网络策略（Network Policies）**：实现网络层面的租户隔离

## 9. 总结

Kubernetes的资源获取机制是一个复杂而完善的系统，它涉及从资源请求到分配、使用和回收的完整生命周期。理解这一机制对于确保应用的性能、稳定性和资源利用率至关重要。

核心要点包括：
1. **资源模型**：区分可压缩资源和不可压缩资源
2. **核心组件**：API Server、Scheduler、Kubelet、Controller Manager、Container Runtime
3. **工作流程**：请求验证、资源调度、资源分配、资源使用
4. **调度算法**：过滤、评分、绑定的完整过程
5. **资源控制**：请求、限制、配额、限制范围
6. **监控回收**：资源监控、OOM回收、节点压力驱逐、资源抢占

通过合理配置资源请求和限制，结合资源配额和自动扩缩容机制，可以实现Kubernetes集群资源的高效利用和应用的稳定运行。
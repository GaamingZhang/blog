---
date: 2026-02-09
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# 单体服务迁移至Kubernetes(二):资源规划与部署设计

## 从镜像到部署:需要回答的核心问题

上一篇介绍了容器化的评估和实施过程,解决了如何把应用装进容器的问题。但有了容器镜像只是完成了第一步。要在Kubernetes集群中运行应用还需要回答一系列更深层的问题:应该用什么资源对象管理应用、应该为容器分配多少CPU和内存、如何让外部流量进入应用、配置信息应该如何管理、如何保证应用的可用性、如何实现优雅的服务更新。这些问题的答案构成了Kubernetes部署设计的核心知识体系。

## 资源对象的选型

### Deployment与StatefulSet的本质差异

对于无状态应用Deployment是标准选择,对于有状态应用StatefulSet提供了额外的保证。但有状态和无状态到底意味着什么,它们在底层实现上有什么区别。

Deployment的设计理念是管理的Pod完全等价和可替换。Deployment并不直接管理Pod而是管理ReplicaSet,ReplicaSet再管理Pod。这种三层结构的设计是为了支持滚动更新:创建新版本的ReplicaSet逐步增加其副本数,同时逐步减少旧版本ReplicaSet的副本数,实现无缝的版本切换。

```
Deployment控制器工作流程:
┌──────────────────────────────────────┐
│ Deployment Controller                │
│ 目标: replicas=3, image=v2.0         │
└───────────┬──────────────────────────┘
            │ 比较期望状态与当前状态
            ↓
┌──────────────────────────────────────┐
│ ReplicaSet (v1)     ReplicaSet (v2)  │
│ replicas=1 → 0      replicas=2 → 3   │ 滚动更新
└───────────┬──────────────────────────┘
            │ 创建/删除Pod
            ↓
┌──────────────────────────────────────┐
│ Pod-1    Pod-2    Pod-3              │
│ 随机名称后缀 (如 web-7f9c8-xk2p)     │
│ 无序创建/删除                         │
│ 任意节点调度                          │
└──────────────────────────────────────┘
```

Deployment的核心特征包括:Pod名称由ReplicaSet名称加上随机后缀组成每次重建都会变化、Pod的创建和删除没有顺序保证可以并行进行、Pod重建后IP地址会变化主机名也会变化、Pod重建后不会自动挂载之前的存储。这些特征的设计目的是实现快速扩缩容和滚动更新。因为Pod是完全等价的所以可以随意创建和销毁不需要考虑顺序和状态保持。

StatefulSet的设计理念则为有状态应用提供三个关键保证:稳定的网络标识、稳定的持久化存储、有序的部署和扩缩容。StatefulSet直接管理Pod而没有中间的ReplicaSet层。每个Pod都有一个从零开始的序号这个序号会成为Pod名称的一部分。Pod按序号顺序创建前一个Pod必须Ready后才创建下一个,删除时则逆序进行。配合Headless Service每个Pod可以获得稳定的DNS记录格式为pod-name.service-name.namespace.svc.cluster.local。通过volumeClaimTemplates为每个Pod创建独立的PVC,Pod重建后会重新挂载同一个PVC。

以一个MySQL主从集群为例理解为什么必须使用StatefulSet。主库必须先于从库启动这需要有序启动保证。从库需要通过稳定的DNS连接主库比如mysql-0.mysql.default.svc.cluster.local,这需要稳定的网络标识。每个实例的数据不能混淆需要独立的持久化存储。Pod重建后必须找回原来的数据需要存储绑定。这些需求Deployment无法满足必须使用StatefulSet。

反过来对于无状态的Web应用,任何实例都能处理任何请求不需要稳定标识、实例之间完全等价不需要有序启动、不在本地存储数据不需要持久化存储、会话存储在Redis中不需要状态保持,使用Deployment即可无需StatefulSet的额外开销。

### Pod模板的核心设计

无论是Deployment还是StatefulSet都通过Pod模板定义容器的运行规格。生产环境的Pod模板应该包含资源限制、健康检查、环境变量、生命周期钩子、安全上下文等关键配置。每个配置项都有其存在的理由接下来将深入这些配置背后的机制。

## Kubernetes调度器工作原理

理解调度器的工作原理能帮助你更好地配置资源请求、预测Pod的调度行为以及排查调度失败的问题。

### 调度的两阶段模型

Kubernetes调度器采用两阶段调度模型:预选和优选。预选阶段从所有节点中筛选出满足条件的候选节点,优选阶段为每个候选节点打分选择得分最高的节点。

```
调度流程:
┌────────────────────────────────┐
│ 1. 待调度Pod进入队列            │
│    pod: requests.cpu=500m       │
└────────────┬───────────────────┘
             ↓
┌────────────────────────────────┐
│ 2. 预选阶段 (Filtering)         │
│    从N个节点中筛选出可用节点     │
│    节点1: 8核CPU, 剩余4核  ✓    │
│    节点2: 4核CPU, 剩余0.2核 ✗   │
│    节点3: 8核CPU, 剩余6核  ✓    │
│    → 剩余候选节点: 节点1, 节点3  │
└────────────┬───────────────────┘
             ↓
┌────────────────────────────────┐
│ 3. 优选阶段 (Scoring)           │
│    为每个候选节点打分            │
│    节点1: 75分 (资源使用均衡)    │
│    节点3: 85分 (剩余资源更多)    │
│    → 选择最高分: 节点3          │
└────────────┬───────────────────┘
             ↓
┌────────────────────────────────┐
│ 4. 绑定Pod到节点                │
│    更新Pod.Spec.NodeName=节点3  │
└────────────────────────────────┘
```

### 预选阶段的过滤器

预选阶段通过一系列过滤器排除不满足条件的节点。只要有一个过滤器返回失败该节点就会被排除。

PodFitsResources过滤器检查节点的可用资源是否满足Pod的资源请求。计算逻辑是节点总CPU减去已分配CPU requests得到可用CPU,然后判断是否大于Pod的CPU request。这里比较的是requests不是limits。即使节点的实际CPU使用率很高只要requests未分配完就认为有资源。这个设计的深层原因是Kubernetes假设你正确地配置了requests和limits,requests代表Pod的最低资源保证。调度器基于requests做决策保证了调度的确定性不会因为瞬时负载波动而影响调度。

PodMatchNodeSelector过滤器检查节点标签是否匹配Pod的nodeSelector。Pod可以通过nodeSelector指定必须调度到具有特定标签的节点,比如要求节点有SSD磁盘或位于特定的可用区。节点必须有所有指定的标签才能通过过滤。

CheckNodeAffinity过滤器检查节点是否满足Pod的亲和性和反亲和性规则。亲和性提供了比nodeSelector更灵活的节点选择机制,支持In、NotIn、Exists、DoesNotExist等操作符,并且区分硬性要求和软性偏好。

PodToleratesNodeTaints过滤器检查Pod是否容忍节点的污点。污点是节点的一种标记表示该节点有特殊性或问题。没有相应容忍的Pod不能调度到有污点的节点。污点和容忍的使用场景包括专用节点(GPU节点、高性能节点)、问题节点(正在维护、性能降级)、隔离环境(生产环境节点不接受测试Pod)。

CheckVolumeBinding过滤器检查Pod需要的PVC是否可以在该节点上绑定。如果Pod使用了PVC且PVC使用local-path等节点亲和的StorageClass则只能调度到该PV所在的节点。这个过滤器解释了为什么使用本地存储会限制Pod的调度灵活性。

### 优选阶段的打分策略

通过预选阶段的节点进入优选阶段,调度器为每个节点打分选择得分最高的节点。

LeastRequestedPriority策略优先选择资源请求量少的节点。打分公式是(capacity - requested) / capacity * 100。这个策略的目的是实现负载均衡避免资源在少数节点上过度集中。

BalancedResourceAllocation策略优先选择CPU和内存使用率接近的节点。这个策略避免了资源碎片化。如果一个节点CPU耗尽但内存充足或者内存耗尽但CPU充足都会导致资源浪费因为Pod通常同时需要CPU和内存。

NodeAffinityPriority策略根据Pod的preferredDuringSchedulingIgnoredDuringExecution规则打分。如果节点匹配偏好规则则加上相应的权重分数。

ImageLocalityPriority策略优先选择已经有镜像缓存的节点。选择已有镜像的节点可以避免镜像拉取时间加快Pod启动。

最终每个策略的得分乘以权重然后累加,选择总分最高的节点。

### QoS等级与OOM优先级

Kubernetes根据Pod的资源配置将Pod分为三个QoS等级。当节点内存不足时kubelet会根据QoS等级选择要驱逐的Pod。

Guaranteed是最高优先级,条件是所有容器都设置了requests和limits且requests等于limits。这个Pod需要的资源是确定的系统会严格保证其资源供应。OOM优先级最低最后被杀。

Burstable是中等优先级,条件是至少一个容器设置了requests但不满足Guaranteed条件。这个Pod保证有requests的资源但可以超用到limits。OOM优先级中等。

BestEffort是最低优先级,条件是所有容器都没有设置requests和limits。这个Pod尽力而为资源不足时首先牺牲。OOM优先级最高最先被杀。

节点内存不足时kubelet首先检测到内存压力,尝试回收可回收的资源比如删除未使用的镜像和容器。如果仍不足开始驱逐Pod:首先驱逐BestEffort的Pod然后驱逐Burstable的Pod按内存使用量超出requests的比例排序,最后驱逐Guaranteed的Pod按优先级排序。

如果kubelet来不及驱逐内核OOM Killer会介入。内核为不同QoS设置不同的oom_score_adj:Guaranteed是-998几乎不会被杀、Burstable根据内存使用比例计算在2到999之间、BestEffort是1000最容易被杀。

生产环境的关键应用应该配置为Guaranteed保证资源供应和稳定性。可以允许超用的应用配置为Burstable提高资源利用率。避免使用BestEffort除非是可以随时重启的一次性任务。

## 资源请求与限制的深度规划

### 基于监控数据的资源估算

不要凭感觉配置资源而应该基于实际的监控数据。估算方法是在传统环境中运行应用收集一段时间至少一周最好包含高峰期的资源使用数据,然后分析资源使用的分布包括平均值、P50、P90、P95、P99分位值、最大值。配置requests时CPU requests取P90或P95值Memory requests取P95值。配置limits时CPU limits取2-3倍requests允许突发Memory limits取1.2-1.5倍requests内存不允许太大突发。

### CPU limits的争议与CFS调度器

CPU limits是Kubernetes中最具争议的配置之一。理解CFS调度器的工作原理能帮助你做出明智的决策。

Linux CFS调度器使用两个参数控制进程的CPU时间:cpu.cfs_period_us是调度周期默认100000即100ms,cpu.cfs_quota_us是周期内允许使用的CPU时间。Kubernetes的CPU limits会转换为这两个参数。比如cpu.limits等于1000m即1核时,cpu.cfs_period_us等于100000即100ms,cpu.cfs_quota_us等于100000即100ms,含义是每100ms周期内进程最多使用100ms的CPU时间。如果cpu.limits等于500m即0.5核,cpu.cfs_quota_us等于50000即50ms,含义是每100ms周期内进程最多使用50ms的CPU时间。

当进程用完了quota时会被强制休眠到下个周期。问题是即使节点有空闲CPU进程也会被限流。对延迟敏感的应用突发请求可能因throttle而变慢。多线程应用的throttle问题更严重:4个线程每个跑25ms总共100ms,如果limits是500m则quota只有50ms,即使是单核的工作量也会被throttle。

真实案例是一个Java应用limits设置为1000m即1核,JVM使用了10个GC线程。在一个100ms周期内10个线程各跑了15ms总CPU时间是150ms,但quota只有100ms超出50ms结果应用被throttle响应变慢。而节点有8核CPU使用率只有20%。限流不是因为CPU不足而是因为limits的硬性限制。

CPU limits的争议在于支持设置limits的理由是防止应用失控比如死循环或CPU泄漏影响其他Pod,在多租户环境中实现资源隔离,保证资源分配的公平性。反对设置limits的理由是CPU是可压缩资源超用不会导致系统崩溃,throttle导致的性能问题难以排查,节点有空闲CPU却限制应用使用违背了资源利用率的初衷。

实践建议有三种方案。方案一是不设置CPU limits,只设置CPU requests和Memory limits。优点是避免throttle应用可以充分利用空闲CPU,缺点是失控的应用可能影响其他Pod,适用于信任的应用和非严格多租户环境。方案二是设置宽松的CPU limits比如requests是500m但limits是4000m远大于requests。优点是防止失控但允许合理的突发,适用于需要安全边界但希望保持灵活性的场景。方案三是使用cgroup v2的burst特性需要Kubernetes 1.22以上和cgroup v2,允许短时间内超过limits更符合突发场景。

检测throttle问题可以查看容器的CPU throttle统计,如果被throttle的周期数除以总周期数大于10%说明throttle比较严重应该考虑调整limits。

### 内存limits与OOM Kill机制

与CPU不同内存是不可压缩资源。容器超过内存limits时会被OOM Killed。

Kubernetes设置memory.limits后,Cgroups配置memory.limit_in_bytes为相应的字节数。内核行为是容器进程分配内存时内核记录使用量,当使用量达到limit时内核尝试回收内存比如清理page cache或触发应用的内存释放。如果无法回收足够内存触发OOM Killer选择容器内的一个进程杀死通常选择内存占用最大的进程,如果主进程被杀容器退出。

OOM Killed的现象是Pod状态显示OOMKilled,容器退出码为137即128加9,9是SIGKILL信号。

容器内存使用等于RSS加Cache加Swap。RSS是真正占用的物理内存不能被回收,Cache是文件系统缓存可以被回收,Swap是交换到磁盘的内存但Kubernetes默认禁用swap。OOM判断基于RSS加Cache减去可回收的cache。常见误解是监控显示内存使用500Mi但OOM Killed,原因是监控可能包含了可回收的cache而OOM基于实际不可回收的内存。

Java应用的内存使用有其特殊性需要特别关注。问题配置是resources.limits.memory是1Gi但JAVA_OPTS的Xmx是1g。问题在于堆内存Xmx只是JVM内存的一部分,JVM内存等于堆加元空间加栈加直接内存加JVM自身,实际内存等于JVM内存加OS缓存。如果Xmx等于1g实际使用可能达到1.2到1.5g,超过limits 1Gi时容器被OOM Killed。

正确的Java应用内存配置是resources.limits.memory是1Gi,JAVA_OPTS设置-XX:+UseContainerSupport让JVM感知容器的内存限制,-XX:MaxRAMPercentage=75.0表示堆内存最多使用容器内存的75%。这样容器limits是1Gi时JVM堆最大是768Mi剩余256Mi留给元空间、栈、JVM自身,JVM不会超出容器限制避免OOM。

内存requests和limits的关系需要注意反模式是requests和limits差异过大比如requests是512Mi但limits是4Gi。问题是调度时只看requests可能调度到小节点,但运行时可能使用到4Gi超出节点可用内存导致节点内存压力影响其他Pod。正确做法是limits不超过requests的1.5到2倍允许一定的突发但不过度。

## Service的底层机制

Service是Kubernetes中实现服务发现和负载均衡的核心抽象。理解Service的底层实现是掌握Kubernetes网络的关键。

### kube-proxy的三种模式

Service的负载均衡由kube-proxy组件实现,它有三种工作模式。

userspace模式已废弃。流量路径是客户端到iptables到kube-proxy用户态进程到后端Pod。工作原理是kube-proxy监听Service的端口,iptables规则将流量重定向到kube-proxy,kube-proxy在用户态做负载均衡然后转发流量到Pod。缺点是用户态和内核态切换性能差,kube-proxy是单点成为瓶颈。

iptables模式是当前默认模式。流量路径是客户端到iptables到后端Pod。工作原理是kube-proxy监听Service和Endpoint的变化创建iptables规则实现负载均衡,流量直接通过内核转发不经过kube-proxy进程,负载均衡在内核态完成性能好。

iptables规则链是PREROUTING链跳到KUBE-SERVICES链,KUBE-SERVICES链匹配Service的ClusterIP和端口跳到KUBE-SVC-XXX链,KUBE-SVC-XXX链通过随机概率跳到不同的KUBE-SEP-YYY链,KUBE-SEP-YYY链执行DNAT到具体的Pod IP。通过概率实现负载均衡比如三个Endpoint时第一条规则33%概率跳到SEP-1否则进入下一条规则,第二条规则剩余2/3中的1/2概率跳到SEP-2总概率是1/3,剩余1/3跳到SEP-3,结果是三个Endpoint概率相等。

iptables模式的问题包括:规则数量问题,每个Service有O(n)条规则n是endpoint数量,1000个Service每个10个Endpoint产生10000多条规则,iptables规则是线性匹配规则多时性能下降。负载均衡不精确,基于随机概率不是真正的轮询无法感知后端Pod的实际负载无法实现高级的负载均衡算法。不支持健康检查,iptables不知道后端Pod是否健康依赖kubelet的探针机制更新Endpoint如果探针不及时会路由到不健康的Pod。

IPVS模式是推荐模式。流量路径是客户端到IPVS内核到后端Pod。工作原理是kube-proxy监听Service和Endpoint的变化创建IPVS虚拟服务器和真实服务器,IPVS在内核态做负载均衡性能高支持多种调度算法。

IPVS相比iptables的优势是负载均衡算法支持rr、lc、dh等多种算法而不只是随机,规则匹配是哈希O(1)而不是线性O(n),大规模性能优秀,支持连接保持。

选择建议是集群规模小于100个Service时iptables足够,集群规模大或对性能有要求时使用IPVS,需要高级负载均衡算法时使用IPVS,节点不支持IPVS模块时使用iptables。

### Endpoint和EndpointSlice的更新机制

Service如何知道后端有哪些Pod是通过Endpoint对象实现的。

Endpoint的工作流程是创建Deployment和Service时Deployment有3个Pod标签是app=nginx,Service的selector匹配app=nginx。Endpoint Controller监听到Service发现selector是app=nginx,查询符合条件的Pod得到pod-1、pod-2、pod-3,创建Endpoint对象记录Pod的IP和端口。kube-proxy监听到Endpoint变化更新iptables或IPVS规则将流量负载均衡到三个Pod IP。Pod变化时比如扩缩容、重启、更新,kubelet更新Pod的Ready状态,Endpoint Controller更新Endpoint对象,kube-proxy更新转发规则。

Endpoint对象的结构包括与Service同名的metadata,subsets中的addresses数组包含Ready的Pod的IP、nodeName、targetRef,notReadyAddresses数组包含未Ready的Pod,ports数组包含端口和协议。

Endpoint的扩展性问题是一个Service有10000个Pod时Endpoint对象包含10000个IP地址大小约1MB。每次有Pod变化Endpoint Controller更新整个Endpoint对象,API Server广播给所有监听者,每个节点的kube-proxy都收到完整的1MB数据,1000个节点每次更新传输1GB数据。影响是API Server和etcd压力大网络带宽消耗大kube-proxy处理延迟高。

EndpointSlice的解决方案是将大的Endpoint对象分片每个片最多包含100个地址。10000个Pod的场景下Endpoint是1个对象1MB,EndpointSlice是100个对象每个约10KB。Pod变化时Endpoint是更新1MB对象广播到所有节点,EndpointSlice是只更新1个10KB的slice只广播这一个。优势是减少单次更新的数据量从1MB降到10KB,减少API Server和etcd的压力,提高kube-proxy的响应速度,支持多种地址类型包括IPv4、IPv6、FQDN。

为什么理解这些对迁移很重要。排查Service连不通问题时检查Service的selector是否正确,检查Endpoint是否有地址,检查Pod的探针状态因为未Ready的Pod不会加入Endpoint。理解探针和流量的关系是探针失败导致Pod标记为NotReady从Endpoint移除不再接收流量,这个链路的延迟决定了故障恢复的速度。性能优化方面大规模应用应启用EndpointSlice,Kubernetes 1.21以上默认启用,调整探针的频率和阈值避免频繁更新Endpoint。

## Ingress设计

Service解决了集群内部的服务发现但外部流量如何进入集群是Ingress的职责。

### Ingress Controller的工作原理

Ingress本身只是一个API对象描述了路由规则,真正实现路由的是Ingress Controller。

Ingress的架构是外部流量到Ingress Controller比如Nginx到Service到Pod。Ingress Controller监听Ingress、Service、Endpoint的变化,当Ingress对象创建或更新时解析Ingress的规则包括host、path、backend,查询backend对应的Service,查询Service对应的Endpoint即Pod IP,生成Nginx配置文件,Reload Nginx。流量处理是请求到达Nginx匹配Host和Path代理到Service的Endpoint即Pod IP返回响应。

Ingress替代传统Nginx的优势是传统Nginx反向代理需要手动维护nginx.conf后端IP变化需要手动更新配置需要手动reload Nginx配置存储在特定服务器不易管理。Kubernetes Ingress是声明式管理规则存储在etcd,后端Pod变化自动更新配置自动reload Nginx,配置与集群状态同步易于管理,支持蓝绿部署、金丝雀发布通过注解实现。

## ConfigMap与Secret管理

配置外部化是容器化的核心原则,Kubernetes通过ConfigMap和Secret实现配置管理。

### ConfigMap的存储机制

ConfigMap将配置数据存储在etcd中以key-value形式组织。ConfigMap在etcd中的存储路径是/registry/configmaps/namespace/name。

使用ConfigMap的方式有三种。方式一是环境变量,从configMapKeyRef读取单个key的值注入到环境变量。方式二是环境变量批量导入,从configMapRef读取整个ConfigMap的所有key作为环境变量。方式三是挂载为文件,通过volume挂载ConfigMap到容器内的路径,可以选择性挂载特定的key到特定的文件名。

ConfigMap的大小限制是单个ConfigMap最大1MB因为存储在etcd中受etcd的限制。建议是大文件不要放ConfigMap使用对象存储,如果必须使用考虑压缩或拆分成多个ConfigMap。

### Secret的编码机制

Secret与ConfigMap类似但有以下差异。Secret的特殊处理包括Base64编码注意不是加密,etcd中的存储默认情况是明文存储仅Base64编码启用加密需要使用EncryptionConfiguration加密存储,API访问限制是RBAC可以单独控制Secret的访问权限更严格的审计日志。

Secret的安全性考量包括Base64不是加密任何人都可以解码不要在日志、监控中暴露Secret的data字段。启用etcd加密需要配置EncryptionConfiguration指定加密算法和密钥。RBAC控制限制Secret的访问只允许特定的Role访问特定的Secret。外部Secret管理使用HashiCorp Vault、AWS Secrets Manager等外部系统通过External Secrets Operator同步到Kubernetes。

### 配置热更新机制

ConfigMap和Secret更新后如何同步到容器。

kubelet的sync loop机制是kubelet定期默认每分钟同步Pod的状态,检查挂载的ConfigMap或Secret是否有更新,如果有更新更新容器内的文件,如果是环境变量方式不会自动更新需要重建Pod。

环境变量与文件挂载的对比是环境变量不支持热更新需要重建Pod,文件挂载支持热更新最多1分钟延迟。环境变量是进程启动时读取运行时不变,ConfigMap更新不影响已运行的Pod需要滚动重启Pod才能生效。文件挂载是kubelet自动更新文件应用需要支持配置热加载比如监听文件变化不需要重启Pod。

最佳实践是经常变化的配置使用文件挂载加热加载比如日志级别、功能开关、限流阈值。很少变化的配置使用环境变量比如数据库地址、第三方API地址、服务名称。敏感信息使用Secret加文件挂载比如数据库密码、API密钥、证书。

## 健康检查探针的底层机制

探针是Kubernetes判断容器健康状态的方式直接影响流量路由和容器重启策略。

### kubelet如何执行探针检查

探针的执行由节点上的kubelet负责而不是控制平面组件。探针的执行流程是kubelet启动时为每个容器创建探针工作协程,工作协程按照periodSeconds周期执行探针,探针返回成功或失败,kubelet记录成功失败次数,达到阈值后执行相应动作比如重启或从Service移除。

### 探针的三种实现方式

HTTP探针的工作原理是kubelet向容器的IP和Port发送HTTP GET请求,如果返回状态码200到399认为成功,如果返回其他状态码、超时或连接失败认为失败。

TCP探针的工作原理是kubelet尝试与容器的端口建立TCP连接,如果连接成功认为探针成功,如果连接失败或超时认为探针失败。适用场景是应用没有HTTP接口比如Redis、MySQL或只需检查端口可达性。

Exec探针的工作原理是kubelet在容器内执行指定命令,如果命令退出码为0认为成功,如果命令退出码非0认为失败。适用场景是需要复杂的健康检查逻辑或应用没有HTTP和TCP接口。

### 探针参数的含义

探针参数包括initialDelaySeconds是容器启动后等待多久再开始探针,periodSeconds是每多少秒执行一次探针,timeoutSeconds是探针超时时间,successThreshold是连续多少次成功认为健康,failureThreshold是连续多少次失败认为不健康。

参数调优建议是initialDelaySeconds太短时应用还没启动完成探针失败导致频繁重启,太长时启动失败的容器长时间不被发现,建议设置为应用启动时间加10秒缓冲。periodSeconds太短时频繁探测增加负载,太长时故障检测延迟高,建议5到10秒。failureThreshold太小时偶尔的网络抖动导致重启,太大时故障检测延迟高,建议3到5次。

### 三种探针的使用场景

Liveness Probe存活探针检测容器是否还活着如果失败则重启容器。使用场景是应用陷入死锁无法处理请求但进程没有退出,应用内存泄漏频繁Full GC无法响应,应用的健康检查接口能真实反映应用状态。注意事项是探针不应该检查外部依赖比如数据库,如果检查数据库连接数据库故障会导致所有Pod重启加剧故障,只检查应用本身是否还能响应。

Readiness Probe就绪探针检测容器是否准备好接收流量如果失败则从Service端点移除。使用场景是应用启动需要加载大量数据或预热,应用依赖外部服务外部服务不可用时不应接收流量,应用正在处理长时间任务暂时不想接收新请求。与Liveness的区别是Liveness失败重启容器所有连接断开正在处理的请求失败造成服务中断,Readiness失败从Service移除不接收新流量但现有连接保持容器继续运行。使用建议是Readiness可以检查外部依赖比如数据库或缓存,依赖不可用时Readiness失败流量不会进来等待恢复避免雪崩效应,Liveness只检查应用自身是否死锁或卡死不检查外部依赖避免级联重启。

Startup Probe启动探针检测容器是否已经启动在启动探针成功之前不会执行其他探针。使用场景是应用启动很慢比如Java应用冷启动,避免Liveness探针过早杀死还在启动的容器。工作流程是容器启动执行startupProbe最多允许一定时间,startupProbe成功后停止执行startupProbe开始执行livenessProbe和readinessProbe,如果startupProbe在规定时间内未成功容器被杀死。优势是启动期间有更宽松的检测条件启动后有更严格的检测条件,避免Liveness配置冲突比如initialDelaySeconds太长或failureThreshold太大。

## 优雅终止的完整机制

优雅终止是生产环境必须正确实现的机制否则会在滚动更新、缩容时丢失请求。

### Pod终止的完整流程

当Kubernetes决定终止一个Pod时比如缩容、更新、节点维护会经历以下步骤。API Server标记Pod为Terminating设置Pod.Metadata.DeletionTimestamp为当前时间。然后并行执行两个操作:Endpoint Controller更新Endpoints移除Pod IP,kube-proxy更新iptables或IPVS规则延迟几秒;同时kubelet接收到删除请求执行preStop hook然后发送SIGTERM信号给容器主进程PID 1,等待容器优雅关闭最多terminationGracePeriodSeconds秒。如果超时未退出发送SIGKILL强制杀死。

### endpoint更新与SIGTERM的竞态条件

这是优雅终止中最容易被忽视的问题:步骤2a和2b是并行的。竞态条件的时间线是T0时Pod标记为Terminating,T1时Endpoint Controller看到Pod Terminating开始更新Endpoints同时kubelet看到Pod Terminating开始执行preStop,T2时kubelet执行完preStop发送SIGTERM,T3时应用收到SIGTERM停止接收新请求开始处理现有请求,T4时Endpoint Controller完成更新kube-proxy开始更新规则,T5时kube-proxy完成规则更新不再路由流量到该Pod,T6时应用处理完所有请求退出。

问题是T2到T5期间应用已经收到SIGTERM开始关闭但负载均衡器还在路由流量到该Pod,如果应用立即退出这些请求会失败。

具体场景分析是没有preStop hook的情况下T0时Pod标记Terminating,T0时Endpoint开始移除需要3到5秒传播,T0时发送SIGTERM,T0加0.1秒应用收到SIGTERM立即退出,T0加2秒新请求到达连接被拒绝。有preStop hook比如sleep 5的情况下T0时Pod标记Terminating,T0时Endpoint开始移除需要3到5秒传播,T0时执行preStop hook sleep 5,T0加5秒preStop完成发送SIGTERM,T0加5.1秒应用收到SIGTERM开始优雅关闭,此时Endpoint已经传播完成不再有新请求。

### preStop hook的作用

preStop hook不是用来处理业务逻辑而是用来延迟SIGTERM的发送给endpoint更新留出时间。preStop可以是exec类型执行命令比如sleep 5,作用是延迟5秒再发送SIGTERM给Endpoint Controller和kube-proxy时间更新规则确保SIGTERM发送时已经没有新流量进来。为什么是5秒因为Endpoint更新约1秒kube-proxy更新规则约2到3秒取决于规则数量网络传播延迟约1秒总计3到5秒是合理的缓冲。也可以用HTTP请求而不是sleep,应用可以在这个接口中从负载均衡器注销自己完成正在处理的请求清理资源。

### terminationGracePeriodSeconds的设计

这个参数定义了从发送SIGTERM到发送SIGKILL的最大等待时间。terminationGracePeriodSeconds等于preStop时间加应用优雅关闭时间。示例是preStop是sleep 5应用处理现有请求最多20秒缓冲5秒总计30秒。如果30秒内应用未退出kubelet发送SIGKILL容器被强制杀死可能丢失正在处理的请求。

如何确定合理的值是测量preStop时间如果有,测量应用处理请求的P99时间,terminationGracePeriodSeconds等于preStop加P99乘以2加缓冲。示例是preStop 5秒P99请求时间500ms长时间请求比如文件上传P99是10秒,设置5加10加5等于20秒。

## 小结

从容器镜像到生产环境运行需要解决一系列部署设计问题。资源对象选型是Deployment适合无状态应用提供快速的扩缩容和滚动更新StatefulSet适合有状态应用提供稳定的网络标识、持久化存储和有序管理。调度器原理是两阶段调度模型预选过滤不可用节点优选打分选择最佳节点,资源请求requests影响调度决策和QoS等级,QoS等级决定了OOM时的驱逐优先级。资源配置基于监控数据的P90或P95值配置requests而不是凭感觉,CPU limits可能导致throttle需要根据场景权衡,内存limits必须设置超限会被OOM Killed,Java应用需要特别注意JVM堆内存与容器内存的关系。Service机制是kube-proxy的三种模式iptables默认和IPVS高性能,Endpoint和EndpointSlice的作用连接Service和Pod,理解Service的底层实现能更好地排查网络问题。配置管理是ConfigMap和Secret的存储机制和使用方式,文件挂载支持热更新环境变量不支持,Secret的Base64编码不是加密需要启用etcd加密。健康检查是三种探针Liveness存活Readiness就绪Startup启动,探针的执行由kubelet负责支持HTTP、TCP、Exec三种方式,正确配置探针参数避免误杀或检测延迟。优雅终止是Pod终止的完整流程和endpoint更新的竞态条件,preStop hook的作用是延迟SIGTERM给endpoint更新留出时间,terminationGracePeriodSeconds要覆盖preStop加应用关闭时间,应用层需要实现SIGTERM信号处理停止接受新请求并处理完现有请求。

下一篇将介绍具体的迁移策略如何制定迁移计划、实施蓝绿部署和金丝雀发布、建立监控和告警体系以及生产环境的最佳实践。

## 常见问题

### Q1: CPU requests和limits都不设置会怎样

如果不设置CPU的requests和limits,Pod会被归类为BestEffort QoS如果内存也不设置或Burstable QoS如果内存设置了。

不设置CPU requests的影响包括调度阶段调度器认为Pod的CPU需求为0,Pod可以被调度到任何节点即使该节点已经很繁忙可能导致节点CPU过载影响所有Pod的性能。运行阶段没有CPU资源保证在节点繁忙时可能分不到CPU时间,与其他Pod竞争CPU性能不稳定,无法通过HPA基于CPU使用率自动扩缩容因为HPA需要requests作为基准。建议至少设置CPU requests保证基本的资源份额,即使不确定准确值也应该设置一个保守值比如100m。

不设置CPU limits的影响包括运行阶段Pod可以使用节点上的所有空闲CPU不会被throttle性能更好,失控的应用可能占用过多CPU影响其他Pod。适用场景是信任的应用不会出现CPU泄漏,希望应用充分利用空闲CPU,非严格的多租户环境。建议生产环境关键应用可以不设置limits提升性能,多租户环境或不信任的应用应设置limits防止影响他人。

### Q2: 为什么Service的ClusterIP在集群外访问不到

ClusterIP是Kubernetes集群内部的虚拟IP只在集群网络中有效这是设计的结果不是限制。

ClusterIP的工作原理是Service创建时分配ClusterIP比如10.96.1.100,kube-proxy在每个节点创建iptables或IPVS规则,当Pod访问10.96.1.100时iptables或IPVS截获流量,通过DNAT转换为后端Pod的真实IP流量直接到达Pod。关键点是ClusterIP只存在于iptables或IPVS规则中,只有安装了这些规则的节点才能路由流量,集群外的机器没有这些规则无法解析ClusterIP。

如何从集群外访问Service有三种方式。方式一是NodePort,Service类型设置为NodePort指定nodePort比如30080在所有节点的30080端口暴露。访问方式是http://任意节点IP:30080。工作原理是kube-proxy在每个节点监听30080端口,流量到达节点的30080端口时通过iptables或IPVS转发到Pod即使Pod不在该节点上也能正确转发。缺点是端口范围限制30000到32767需要知道节点IP不适合生产环境直接暴露。方式二是LoadBalancer,Service类型设置为LoadBalancer。工作原理是在云环境中比如AWS、GCP、Azure自动创建云厂商的负载均衡器,负载均衡器获得公网IP,负载均衡器将流量转发到NodePort,NodePort再转发到Pod。访问方式是http://LoadBalancer公网IP。缺点是需要云环境支持,每个Service创建一个负载均衡器成本高,自建集群无法使用除非安装MetalLB等实现。方式三是Ingress推荐,Ingress Controller比如Nginx以LoadBalancer或NodePort方式暴露,通过域名和路径路由到不同Service,一个Ingress Controller可以处理多个Service。访问方式是http://myapp.example.com。优势是一个公网IP可以服务多个应用,支持HTTPS、路径路由、域名路由,成本低易于管理。

推荐方案是开发测试使用kubectl port-forward访问http://localhost:8080,生产环境使用Ingress配置域名解析到Ingress Controller的公网IP通过域名访问应用统一管理SSL证书。

### Q3: Readiness探针失败后正在处理的请求会怎样

Readiness探针失败只影响新流量不会中断正在处理的请求。这是Kubernetes设计的重要细节。

Readiness探针失败的处理流程是T0时Readiness探针连续失败达到failureThreshold,T1时kubelet将Pod状态标记为NotReady,T2时Endpoint Controller检测到Pod NotReady,T3时Endpoint Controller从Service的Endpoint列表移除该Pod IP,T4时kube-proxy更新iptables或IPVS规则,T5时不再有新流量路由到该Pod。关键点是整个流程不会主动断开现有连接,已经建立的TCP连接继续有效,正在处理的HTTP请求继续处理,只是新的请求不再路由到该Pod。

具体场景分析包括短请求比如API调用100ms完成,T0时客户端发起请求建立连接,T0加50ms时Readiness探针失败,T0加60ms时Pod从Endpoint移除,T0加100ms时请求处理完成返回响应,结果请求正常完成不受影响。长请求比如文件上传60秒完成,T0时客户端开始上传文件,T0加10s时Readiness探针失败,T0加12s时Pod从Endpoint移除,T0加60s时文件上传完成返回响应,结果上传继续不受影响只要TCP连接不断开请求就能完成。连接池场景是客户端使用连接池持有长连接,T0时连接池中有10个连接到Pod A,T1时Pod A的Readiness探针失败,T2时Pod A从Endpoint移除,T3时新请求通过Service无法路由到Pod A,T4时但连接池中的10个连接仍然有效,T5时直接使用连接池中连接的请求仍能到达Pod A。注意这是为什么Pod终止时需要优雅关闭,如果Pod立即退出会断开这些连接。

与Liveness探针的对比是Readiness失败时Pod标记为NotReady从Service移除不接收新流量现有连接保持容器继续运行,Liveness失败时容器被杀死重启所有连接断开正在处理的请求失败造成服务中断。使用建议是Readiness可以检查外部依赖比如数据库或缓存,依赖不可用时Readiness失败暂时不接收流量等待恢复,Liveness只检查应用自身是否死锁或卡死不检查外部依赖避免级联重启。

### Q4: 为什么需要EndpointSlice不能直接优化Endpoint吗

EndpointSlice不仅仅是Endpoint的性能优化它引入了新的架构设计解决了Endpoint的根本性限制。

Endpoint的架构限制是Endpoint是一个单体对象一个Service对应一个Endpoint对象,Endpoint对象包含所有Pod的IP地址任何Pod的变化都需要更新整个Endpoint对象。问题包括对象大小限制etcd单个对象限制约1.5MB如果Service有10000个Pod Endpoint对象会超限无法支持超大规模Service,更新粒度问题一个Pod变化需要读取修改写入整个Endpoint并发更新冲突多个Pod同时变化API Server和etcd压力大,网络传输问题kube-proxy watch Endpoint变化每次变化都传输整个Endpoint对象1000个节点乘以1MB等于1GB网络流量。无法通过简单优化解决需要架构改变。

EndpointSlice的架构设计是EndpointSlice是分片架构一个Service对应多个EndpointSlice对象,每个Slice最多包含100个Endpoint可配置,Pod变化只影响一个Slice。

EndpointSlice的优势包括突破大小限制单个Slice只有约10KB可以支持任意数量的Pod受限于Slice数量而不是单个对象大小,减少更新冲突Pod分散在不同Slice中并发更新不同Slice不会冲突提高并发扩缩容的性能,减少网络流量Pod变化只传输影响的Slice 1000节点乘以10KB等于10MB而不是1GB减少99%的网络流量,支持多种地址类型Endpoint只支持IPv4而EndpointSlice支持IPv4、IPv6、FQDN,更丰富的元数据EndpointSlice包含nodeName该Pod所在节点zone该Pod所在可用区topology拓扑信息支持拓扑感知路由优先路由到同节点或同可用区的Pod减少跨可用区流量降低延迟和成本。

Kubernetes 1.21以上默认启用EndpointSlice,Endpoint仍然存在保持兼容性新组件使用EndpointSlice老组件仍然可以使用Endpoint。建议使用Kubernetes 1.21以上自动启用EndpointSlice,自定义控制器应该迁移到EndpointSlice API,大规模集群会明显感受到性能提升。

### Q5: preStop hook执行失败会怎样

preStop hook执行失败不会阻止容器终止但会影响优雅关闭的流程。

preStop hook的执行结果是Exec类型的preStop退出码非0或HTTP类型的preStop返回状态码500时,执行失败的行为是kubelet记录警告日志继续执行后续步骤发送SIGTERM不会因为preStop失败而重试或停止终止流程。Events显示Warning FailedPreStopHook preStop hook failed。

实际影响包括场景一preStop是sleep命令,如果容器没有sh或命令执行失败preStop立即失败没有延迟立即发送SIGTERM可能导致endpoint还没移除就收到SIGTERM丢失请求的风险增加。场景二preStop是HTTP调用,如果应用没有实现这个接口HTTP返回404 preStop失败立即发送SIGTERM同样可能丢失请求。场景三preStop超时,如果preStop是sleep 60但terminationGracePeriodSeconds是30,时间线是T0开始执行preStop sleep 60,T30时terminationGracePeriodSeconds到期kubelet强制杀死容器SIGKILL,preStop被强制中断。preStop的执行时间会计入terminationGracePeriodSeconds。

正确的preStop设计包括简单可靠的方案使用sleep,优点是简单不会失败兼容性好所有镜像都支持延迟时间可预测,缺点是固定延迟不够灵活。应用感知的方案使用HTTP hook,优点是应用可以执行自定义逻辑可以从负载均衡器注销自己更精细的控制,缺点是依赖应用实现实现错误可能导致preStop失败。混合方案使用带错误处理的脚本尝试HTTP调用如果失败fallback到sleep,优点是应用实现了接口就用接口应用没实现就fallback到sleep健壮性高。

建议简单应用使用sleep 5可靠不会失败大多数场景足够,复杂应用实现HTTP hook可以执行自定义清理逻辑记得处理错误避免preStop失败,生产环境测试preStop的执行手动删除Pod观察日志检查Events确认preStop正常执行没有FailedPreStopHook警告,时间规划terminationGracePeriodSeconds大于等于preStop时间加应用关闭时间加缓冲。

## 参考资源

- [Kubernetes Deployment 官方文档](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
- [配置 Pod 的服务质量](https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/)
- [Pod 生命周期](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)

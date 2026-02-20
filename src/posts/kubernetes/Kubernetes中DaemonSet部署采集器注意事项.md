---
date: 2026-02-11
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# Kubernetes 中 DaemonSet 部署采集器有什么注意事项？

DaemonSet 是监控采集器的标准部署方式。node_exporter 采集节点指标、Filebeat 收集容器日志、Fluent Bit 转发日志流、otel-collector 汇聚链路数据——这些组件几乎无一例外地跑在 DaemonSet 上。

看起来很简单：写一个 YAML，`kubectl apply`，完事。但真正在生产环境跑过之后，才会知道坑有多少。有人因为忘配 toleration 导致 control-plane 节点上监控断了好几天才发现，有人因为没设 PriorityClass 让节点紧张时 node_exporter 第一个被驱逐，有人因为 liveness 探针太激进导致采集器频繁重启指标丢点。

本文不是 DaemonSet 的基础教程（那篇已有 [DaemonSet详解](./DaemonSet详解.md)），而是聚焦在生产环境中用 DaemonSet 部署采集器时需要重点关注的七个方面，每个方面都结合真实踩坑经验展开。

---

## 一、资源配额：既不能饿死，也不能吃撑

采集器是"旁路"服务，永远不应该影响业务容器。但如果什么都不配，在节点资源紧张时它会被最先驱逐；如果配得太随意，它可能把节点打垮。

### QoS 等级的选择

Kubernetes 根据 requests 和 limits 的配置关系将 Pod 分为三个 QoS 等级：

```
Guaranteed:  requests == limits（驱逐时最后走）
Burstable:   requests != limits（中等优先级）
BestEffort:  没有任何 resources 配置（最先被驱逐）
```

大多数工程师给采集器配 Burstable，理由是"它的负载会随节点上容器数量变化，需要弹性"。这个理由本身没错，但问题在于当节点内存压力触发驱逐时，Burstable Pod 的驱逐顺序是：**内存使用量超出 requests 比例最高的先走**。

如果你的 Filebeat 在日志洪峰时临时使用了大量内存（超出 requests 很多），它很可能在关键时刻被驱逐——而这恰恰是日志量最大、最需要采集的时候。

**实践建议**：关键采集器优先考虑 Guaranteed QoS，即 requests 等于 limits。代价是需要提前容量规划，但换来的是采集的稳定性。

### 参考配置值

以下是生产中经过验证的参考值，需要根据实际节点规模和负载调整：

```yaml
# node_exporter（轻量，纯读取 /proc /sys）
resources:
  requests:
    cpu: 50m
    memory: 30Mi
  limits:
    cpu: 200m
    memory: 60Mi

# Filebeat（需要追踪文件偏移，日志量大时内存上升）
resources:
  requests:
    cpu: 100m
    memory: 100Mi
  limits:
    cpu: 500m
    memory: 300Mi

# Fluent Bit（比 Filebeat 更轻量，C 语言实现）
resources:
  requests:
    cpu: 50m
    memory: 50Mi
  limits:
    cpu: 200m
    memory: 150Mi
```

### 两个典型踩坑

**limits 设太低导致 OOMKilled**：某团队给 Filebeat 设了 `limits.memory: 64Mi`，在日志突增时容器被 OOM Kill，重启后从上次 checkpoint 重放，造成日志重复。排查时 `kubectl describe pod` 的 `Last State` 显示 `OOMKilled`，才找到原因。

**不设 limits 导致打垮节点**：某团队的 otel-collector DaemonSet 没有设任何 limits。某天业务链路数据量突增 10 倍，每个节点上的 collector 内存占用从 100Mi 飙升到 3Gi，触发节点级别的 OOM，业务 Pod 被批量杀掉，影响范围远超预期。

---

## 二、Tolerations：确保覆盖所有节点

这是生产中最容易遗漏的配置，也是发现最晚的问题之一。

### 为什么 control-plane 节点采集不到

Kubernetes 的 control-plane 节点（旧版本称 master）默认带有两个污点：

```
node-role.kubernetes.io/control-plane:NoSchedule
node-role.kubernetes.io/master:NoSchedule        # 旧版本兼容
```

DaemonSet 默认**不会**自动容忍这些污点。因此，如果你的 DaemonSet 没有配置对应的 toleration，control-plane 节点上就不会有采集器运行，而你在 Grafana 上看到的节点列表里也不会有 control-plane 节点的指标——这个"缺口"很可能被当成正常现象忽略掉。

### 完整的 tolerations 配置

生产环境推荐的 tolerations 配置，覆盖常见的污点场景：

```yaml
tolerations:
  # 容忍 control-plane / master 污点（确保在控制平面节点运行）
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
  - key: node-role.kubernetes.io/master
    operator: Exists
    effect: NoSchedule

  # 容忍节点 NotReady / Unreachable 状态（默认有超时，采集器应该坚守）
  - key: node.kubernetes.io/not-ready
    operator: Exists
    effect: NoExecute
    tolerationSeconds: 300
  - key: node.kubernetes.io/unreachable
    operator: Exists
    effect: NoExecute
    tolerationSeconds: 300

  # 容忍节点内存/磁盘压力污点（节点紧张时采集器更应该在线）
  - key: node.kubernetes.io/memory-pressure
    operator: Exists
    effect: NoSchedule
  - key: node.kubernetes.io/disk-pressure
    operator: Exists
    effect: NoSchedule
```

`tolerationSeconds: 300` 表示节点 NotReady 后，容忍 300 秒再驱逐。对于监控采集器，通常希望它在节点恢复之前尽量不被驱逐，这个值可以根据需要调大。

如果集群中有业务自定义污点（比如 `team=platform:NoSchedule`），采集器同样需要添加对应的容忍，否则那些节点上会没有采集。

### 与 nodeAffinity 的配合

toleration 解决的是"能不能去"，nodeAffinity 解决的是"要不要去"。两者是独立的过滤器，采集器通常不需要 nodeAffinity，但如果集群里有 Windows 节点或 ARM 节点，Linux only 的采集器需要用 nodeAffinity 排除这些节点（见第六节）。

---

## 三、宿主机资源访问：最小权限原则

采集器需要读取宿主机上的数据，但"需要访问"和"拿到全部权限"之间有很大的空间。

### hostNetwork 的必要性

node_exporter 需要采集真实的网络接口数据（网卡 RX/TX 字节数、连接数等）。容器默认使用独立的网络命名空间，读取的是容器内的虚拟网络接口，而不是宿主机的物理/虚拟网卡。

解决方案是开启 `hostNetwork: true`，让容器直接使用宿主机的网络命名空间，从而能读取到真实的网络指标：

```yaml
spec:
  hostNetwork: true
  # 注意：hostNetwork 下容器暴露的端口直接占用宿主机端口
  # node_exporter 默认端口 9100 将直接在节点上监听
```

开启 hostNetwork 的副作用是容器端口直接占用节点端口，多个同类采集器不能在同一节点上运行（但 DaemonSet 每节点只有一个，不存在冲突）。

**Filebeat 和 Fluent Bit 不需要 hostNetwork**，它们通过 hostPath 挂载日志目录即可，不需要宿主机网络命名空间。

### hostPath 挂载：只挂必要路径

```yaml
volumeMounts:
  # node_exporter 需要的宿主机路径
  - name: proc
    mountPath: /host/proc
    readOnly: true    # 明确只读
  - name: sys
    mountPath: /host/sys
    readOnly: true

  # Filebeat / Fluent Bit 需要的日志路径
  - name: varlog
    mountPath: /var/log
    readOnly: true
  - name: docker-containers
    mountPath: /var/lib/docker/containers
    readOnly: true

volumes:
  - name: proc
    hostPath:
      path: /proc
  - name: sys
    hostPath:
      path: /sys
  - name: varlog
    hostPath:
      path: /var/log
  - name: docker-containers
    hostPath:
      path: /var/lib/docker/containers
```

`readOnly: true` 不只是"好习惯"，在某些安全扫描工具（如 Falco）会将可写的 hostPath 挂载标记为告警。对于采集器来说，几乎没有写入宿主机文件系统的需求，全部使用只读模式。

### securityContext：避免 privileged

部分工程师在遇到采集器没有权限读取某些数据时，直接加上 `privileged: true` 了事。这种方式相当于给容器完整的 root 权限，容器可以操作任何宿主机资源，安全风险极高。

对于大多数采集器，通过精细化的 capabilities 配置就能满足需求：

```yaml
securityContext:
  runAsNonRoot: false       # node_exporter 需要 root 读取某些 /proc 路径
  readOnlyRootFilesystem: true
  capabilities:
    add:
      - SYS_PTRACE          # hostPID 时读取进程信息需要
    drop:
      - ALL                 # 先全部 drop，再按需 add
```

如果不确定需要哪些 capabilities，可以先不加任何 add，观察运行时报错，按需补充，这比直接 `privileged: true` 要安全得多。

---

## 四、滚动更新策略：不能让节点瞬间失去采集

DaemonSet 的更新与 Deployment 有一个关键区别：**没有 maxSurge**。Deployment 在滚动更新时可以先多起几个新 Pod 再删旧的，保持服务不中断。DaemonSet 每个节点只能有一个 Pod，更新时必然经历"旧的删了，新的还没起来"的短暂空窗。

### maxUnavailable 的影响

```yaml
updateStrategy:
  type: RollingUpdate
  rollingUpdate:
    maxUnavailable: 1    # 同时只有 1 个节点处于更新中
```

如果集群有 50 个节点，`maxUnavailable: 1` 意味着整个更新过程要走 50 轮，每轮都有一个节点的采集器短暂离线。这个配置对于监控数据来说通常可以接受，因为每个节点的采集间隔（通常 15-30 秒）内重启完毕，最多丢几个采集点。

`maxUnavailable` 绝对不能设为等于节点总数（或 100%），那等于所有节点同时更新，整个集群瞬间丧失采集能力。

对于关键的采集器（如 node_exporter），推荐：
- 小集群（< 20 节点）：`maxUnavailable: 1`
- 大集群（> 50 节点）：`maxUnavailable: 5` 或 `10%`（百分比写法：`"10%"`）

### OnDelete 策略的适用场景

```yaml
updateStrategy:
  type: OnDelete
```

OnDelete 策略下，DaemonSet 的 Pod 不会自动更新，只有手动删除旧 Pod 后才会按新配置重建。这适合以下场景：

- 新版本采集器有破坏性变更，需要在每个节点上手动验证后再推进
- 分批次按业务维度更新（比如先更新 dev 节点，验证一周后再更新 prod 节点）

代价是需要手动管理，忘记更新的节点会长期跑旧版本，容易造成版本碎片。

---

## 五、PriorityClass：关键采集器不能被驱逐

这是最容易被忽视的配置，但在资源紧张时影响最大。

### 系统内置优先级

Kubernetes 内置了两个高优先级的 PriorityClass：

| PriorityClass | 优先级值 | 适用场景 |
|---|---|---|
| system-node-critical | 2000001000 | 节点级关键组件（kubelet、网络插件） |
| system-cluster-critical | 2000000000 | 集群级关键组件（CoreDNS、kube-proxy） |

注意：`system-node-critical` 的优先级值反而更高（数字更大）。

### 采集器应该用什么优先级

node_exporter、Filebeat 这类采集器属于"基础设施组件"，比普通业务 Pod 重要，但比 CoreDNS、网络插件次要。推荐的策略：

**方案一**：使用 `system-cluster-critical`
- 适合：核心监控采集器，绝对不能在资源紧张时被驱逐
- 风险：与 CoreDNS 等集群组件同等优先级，可能在极端情况下影响集群功能

**方案二**：自定义 PriorityClass
```yaml
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: monitoring-critical
value: 1000000        # 低于系统关键组件，高于业务 Pod
globalDefault: false
description: "监控采集器专用优先级"
```

然后在 DaemonSet 中引用：

```yaml
spec:
  template:
    spec:
      priorityClassName: monitoring-critical
```

### 踩坑案例

某团队的集群在大促期间资源使用率飙升，调度器开始驱逐低优先级 Pod。由于所有 DaemonSet 采集器都没有配置 PriorityClass，默认优先级为 0，与普通业务 Pod 相同。节点内存压力触发驱逐时，按内存使用量超出 requests 比例排序，node_exporter 由于内存使用紧凑（requests 设得合理，没有超出多少），反而比那些内存使用超出 requests 很多的业务 Pod 优先被驱逐。

结果：大促高峰期，集群监控出现大量空洞。团队直到复盘时才发现是采集器被驱逐导致的。

---

## 六、节点亲和性与特殊节点处理

### GPU 节点的差异化采集

如果集群中有 GPU 节点，通常需要额外部署 nvidia-dcgm-exporter（采集 GPU 指标）。这个采集器只需要在有 GPU 的节点上运行，通过 nodeSelector 或 nodeAffinity 实现：

```yaml
spec:
  template:
    spec:
      nodeSelector:
        nvidia.com/gpu.present: "true"    # 节点上 GPU 设备发现插件打的标签
```

或者用 nodeAffinity（更灵活）：

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: nvidia.com/gpu.present
              operator: In
              values: ["true"]
```

### 排除 Windows / ARM 节点

如果集群是混合架构（Linux + Windows，或 amd64 + arm64），为 Linux/amd64 编译的采集器镜像无法在其他架构节点上运行，需要明确排除：

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: kubernetes.io/os
              operator: In
              values: ["linux"]
            - key: kubernetes.io/arch
              operator: In
              values: ["amd64"]
```

不配这个的后果是 DaemonSet Pod 在 Windows 节点上以 `CreateContainerConfigError` 状态卡住，虽然不影响功能，但告警系统会一直产生 Pod 异常的噪音。

---

## 七、完整示例：生产级 node_exporter DaemonSet

综合以上所有注意事项，一份可以直接用于生产环境的 node_exporter DaemonSet 配置如下：

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-exporter
  namespace: monitoring
  labels:
    app: node-exporter
spec:
  selector:
    matchLabels:
      app: node-exporter
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1           # 同时最多 1 个节点处于更新中
  template:
    metadata:
      labels:
        app: node-exporter
    spec:
      # 高优先级，防止资源紧张时被驱逐
      priorityClassName: system-cluster-critical

      # 使用宿主机网络，采集真实网络接口数据
      hostNetwork: true
      hostPID: true               # 采集进程级别指标需要

      # 容忍所有常见污点，确保全节点覆盖
      tolerations:
        - key: node-role.kubernetes.io/control-plane
          operator: Exists
          effect: NoSchedule
        - key: node-role.kubernetes.io/master
          operator: Exists
          effect: NoSchedule
        - key: node.kubernetes.io/not-ready
          operator: Exists
          effect: NoExecute
          tolerationSeconds: 300
        - key: node.kubernetes.io/unreachable
          operator: Exists
          effect: NoExecute
          tolerationSeconds: 300
        - key: node.kubernetes.io/memory-pressure
          operator: Exists
          effect: NoSchedule
        - key: node.kubernetes.io/disk-pressure
          operator: Exists
          effect: NoSchedule

      # 只在 Linux amd64 节点运行
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: kubernetes.io/os
                    operator: In
                    values: ["linux"]

      serviceAccountName: node-exporter   # 单独的 ServiceAccount，最小权限

      containers:
        - name: node-exporter
          image: quay.io/prometheus/node-exporter:v1.8.2
          args:
            - --path.procfs=/host/proc     # 指向 hostPath 挂载路径
            - --path.sysfs=/host/sys
            - --path.rootfs=/host/root
            - --collector.filesystem.mount-points-exclude=^/(dev|proc|sys|var/lib/docker/.+)($|/)
          ports:
            - containerPort: 9100
              hostPort: 9100              # hostNetwork 下直接占用节点端口
              name: metrics

          # Guaranteed QoS：requests == limits
          resources:
            requests:
              cpu: 100m
              memory: 50Mi
            limits:
              cpu: 250m
              memory: 100Mi

          # 最小权限：不使用 privileged
          securityContext:
            runAsNonRoot: false
            readOnlyRootFilesystem: true
            capabilities:
              add:
                - SYS_PTRACE
              drop:
                - ALL

          # 健康检查：采集器有 /metrics 端点，直接用 HTTP 检查
          livenessProbe:
            httpGet:
              path: /
              port: 9100
            initialDelaySeconds: 30
            periodSeconds: 30
            failureThreshold: 3
            timeoutSeconds: 5

          readinessProbe:
            httpGet:
              path: /
              port: 9100
            initialDelaySeconds: 10
            periodSeconds: 15
            failureThreshold: 2
            timeoutSeconds: 3

          volumeMounts:
            - name: proc
              mountPath: /host/proc
              readOnly: true
            - name: sys
              mountPath: /host/sys
              readOnly: true
            - name: root
              mountPath: /host/root
              readOnly: true
              mountPropagation: HostToContainer

      volumes:
        - name: proc
          hostPath:
            path: /proc
        - name: sys
          hostPath:
            path: /sys
        - name: root
          hostPath:
            path: /

      # 给采集器足够的终止时间，避免强杀导致数据不完整
      terminationGracePeriodSeconds: 30

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: node-exporter
  namespace: monitoring
  # node_exporter 不需要访问 API Server，无需绑定任何 ClusterRole
```

---

## 小结：DaemonSet 采集器部署检查清单

部署或评审采集器 DaemonSet 时，可以对照以下清单逐项检查：

**资源与 QoS**
- [ ] 设置了合理的 requests 和 limits（不是 BestEffort）
- [ ] 关键采集器考虑 Guaranteed QoS（requests == limits）
- [ ] 内存 limits 留有足够余量，避免日志洪峰导致 OOMKilled

**节点覆盖**
- [ ] 配置了 control-plane / master 节点的 toleration
- [ ] 配置了 NotReady / Unreachable 状态的 toleration（带 tolerationSeconds）
- [ ] 如有自定义污点的节点，已补充对应 toleration

**宿主机访问**
- [ ] 只挂载了必要的 hostPath，全部使用 readOnly: true
- [ ] securityContext 未使用 privileged: true，用 capabilities 最小化权限
- [ ] hostNetwork / hostPID 按实际需要配置，不默认开启

**更新策略**
- [ ] updateStrategy 为 RollingUpdate，maxUnavailable 未设为全量
- [ ] terminationGracePeriodSeconds 足够采集器优雅退出

**优先级**
- [ ] 配置了 priorityClassName，防止资源紧张时被驱逐

**健康检查**
- [ ] 配置了 livenessProbe，initialDelaySeconds 足够服务启动
- [ ] failureThreshold 不要太小，避免启动慢时被误杀重启

**节点亲和性**
- [ ] 混合架构集群配置了 nodeAffinity 排除非目标 OS / arch 节点
- [ ] 特殊采集器（GPU exporter 等）通过 nodeSelector 限制在目标节点

---

## 常见问题

### Q1：DaemonSet 采集器 Pod 在某些节点处于 Pending 状态，但节点看起来正常，怎么排查？

Pending 通常有四个原因：（1）节点有污点，DaemonSet 没有对应的 toleration；（2）节点资源不足，无法满足 Pod 的 requests；（3）nodeAffinity 或 nodeSelector 不匹配；（4）hostPort 冲突，节点上已有其他 Pod 占用了同一个端口。

排查步骤：先用 `kubectl describe pod <pending-pod>` 查看 Events 部分，调度失败的原因会在这里写明。如果是污点问题，Events 会提示 `x node(s) had taints that the pod didn't tolerate`；如果是资源不足，会提示 `Insufficient cpu` 或 `Insufficient memory`。

### Q2：node_exporter 采集到的网络指标是容器的虚拟网卡，而不是宿主机真实网卡，怎么解决？

这是忘记配置 `hostNetwork: true` 导致的。node_exporter 在容器网络命名空间内读取 `/proc/net/dev`，读到的是容器的网络接口（eth0 通常是 veth pair 的一端），不是宿主机的物理网卡。

解决方式：在 DaemonSet spec 中加入 `hostNetwork: true`，让容器直接使用宿主机网络命名空间，这样读取 `/proc/net/dev` 就能看到真实的物理接口（如 ens3、bond0 等）。

### Q3：DaemonSet 更新时，如何做到金丝雀验证——先在少数节点更新，观察无误后再全量推进？

DaemonSet 本身不支持金丝雀更新，但可以用以下方式模拟：将 `updateStrategy.type` 改为 `OnDelete`，然后手动删除少数几个节点上的 Pod，让它们按新版本配置重建；观察这些节点的采集器运行正常后，再批量删除剩余 Pod 完成全量更新。

另一种方式是给节点打标签，用 nodeSelector 控制哪些节点先用新版本（创建一个新的 DaemonSet 用于金丝雀节点，与旧 DaemonSet 并存一段时间后再替换）。

### Q4：采集器 DaemonSet 的 liveness 探针设置多激进合适？避免频繁误判重启导致指标丢失？

liveness 探针的核心参数是 `failureThreshold * periodSeconds`，这个值代表"连续多少秒探针失败才触发重启"。对于采集器：

- `initialDelaySeconds` 应该覆盖采集器完整的启动时间，包括初次配置加载和连接建立。通常 30-60 秒是合理的。
- `failureThreshold` 建议至少 3，配合 `periodSeconds: 30`，意味着连续 90 秒探针失败才重启，足以避免短暂的网络抖动或负载尖峰导致的误判。
- 对于 Filebeat 这类有状态采集器，重启会从 checkpoint 重放，可能导致日志重复，liveness 探针可以设得更宽松（failureThreshold: 5，甚至考虑只配 readinessProbe 不配 livenessProbe）。

### Q5：生产集群节点数很多，DaemonSet 每个节点都有一个 Pod，如何高效管理和排查？

几个实用技巧：

（1）用 `kubectl get pods -n monitoring -o wide | grep node-exporter` 快速查看所有节点的 Pod 分布和状态，`-o wide` 会显示 Pod 所在的节点名称。

（2）用标签选择器批量操作：`kubectl delete pod -n monitoring -l app=node-exporter --field-selector=status.phase=Failed`，清理所有 Failed 状态的采集器 Pod。

（3）为 DaemonSet 配置 Prometheus 告警规则，监控每个节点是否都有对应的 Pod 在运行：`count(kube_pod_info{namespace="monitoring",pod=~"node-exporter.*"}) != count(kube_node_info)`，节点数与采集器 Pod 数不一致时告警。这样任何节点缺少采集器都能第一时间发现。

## 参考资源

- [Kubernetes DaemonSet 官方文档](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/)
- [Prometheus Node Exporter 最佳实践](https://prometheus.io/docs/guides/node-exporter/)
- [Kubernetes 资源管理指南](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)

---
date: 2026-01-26
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# DaemonSet详解：每个节点的守护者

## 为什么需要DaemonSet？

有些程序需要在集群的**每个节点**上都运行一份，比如：

- **日志收集器**：需要收集每个节点上所有容器的日志
- **监控代理**：需要采集每个节点的CPU、内存、磁盘等指标
- **网络插件**：需要在每个节点上设置网络规则

这类程序有个共同特点：它们是"节点级别"的服务，而不是"应用级别"的服务。你不能用Deployment来管理它们，因为：

1. Deployment的Pod数量是固定的（replicas=3意味着只有3个Pod）
2. Deployment不保证每个节点都有Pod
3. 新节点加入时，Deployment不会自动在上面创建Pod

**DaemonSet就是专门为这种"每个节点都要运行"的场景设计的**。它确保：
- 每个节点（或指定的节点）上恰好运行一个Pod
- 新节点加入时，自动在上面创建Pod
- 节点被删除时，Pod也随之清理

## DaemonSet的工作原理

你可以把DaemonSet想象成一个"巡逻队长"，它的任务很简单：

1. 巡视集群中的所有节点
2. 发现有节点没有自己的Pod？立刻创建一个
3. 发现有节点被移除了？对应的Pod也自动清理

这个过程完全自动化。当你的集群从5个节点扩展到10个节点时，DaemonSet会自动在新增的5个节点上创建Pod，你什么都不用做。

## 基本示例：日志收集器

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd
  namespace: logging
spec:
  selector:
    matchLabels:
      app: fluentd
  template:
    metadata:
      labels:
        app: fluentd
    spec:
      containers:
        - name: fluentd
          image: fluent/fluentd:v1.14
          volumeMounts:
            - name: varlog
              mountPath: /var/log      # 挂载节点的日志目录
              readOnly: true
          resources:
            requests:
              cpu: 50m
              memory: 100Mi
            limits:
              cpu: 100m
              memory: 200Mi
      volumes:
        - name: varlog
          hostPath:
            path: /var/log
```

这个配置会在每个节点上启动一个Fluentd容器，并挂载节点的`/var/log`目录来收集日志。

## 典型使用场景

| 场景 | 常用工具 |
|------|---------|
| 日志收集 | Fluentd、Filebeat、Fluent Bit |
| 监控采集 | Prometheus Node Exporter、Datadog Agent |
| 网络插件 | Calico、Flannel、Cilium |
| 存储插件 | Ceph、GlusterFS |
| 安全扫描 | Falco、Trivy |

## 控制Pod运行在哪些节点

默认情况下，DaemonSet会在**所有节点**上创建Pod。但有时候你只想在部分节点运行，比如只在有GPU的节点上运行GPU驱动。

### 方法一：nodeSelector

最简单的方式，通过标签选择节点：

```yaml
spec:
  template:
    spec:
      nodeSelector:
        gpu: "true"    # 只在有gpu=true标签的节点上运行
```

### 方法二：nodeAffinity

更灵活的节点选择：

```yaml
spec:
  template:
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: node-type
                    operator: In
                    values: ["worker", "edge"]  # 在worker或edge节点运行
```

## 污点和容忍：让DaemonSet无处不在

Kubernetes的Master节点通常有"污点"（Taint），阻止普通Pod调度上去。但DaemonSet往往需要在所有节点运行，包括Master。

**污点**就像是节点挂的"请勿打扰"牌子，**容忍**则是Pod说"我能接受这个牌子"。

```yaml
spec:
  template:
    spec:
      tolerations:
        # 容忍Master节点的污点
        - key: node-role.kubernetes.io/control-plane
          effect: NoSchedule

        # 容忍所有污点（在任何节点都能运行）
        - operator: Exists
```

如果你希望DaemonSet真的在**每一个**节点运行，加上`operator: Exists`的容忍就行了。

## 更新策略

DaemonSet支持两种更新策略：

### RollingUpdate（默认）

滚动更新，逐个替换Pod。可以通过`maxUnavailable`控制同时更新的数量：

```yaml
updateStrategy:
  type: RollingUpdate
  rollingUpdate:
    maxUnavailable: 1    # 同时最多1个不可用
```

如果你有10个节点，设置`maxUnavailable: 1`意味着每次只更新一个Pod，更新完成后再更新下一个。

### OnDelete

手动控制更新。只有当你手动删除旧Pod后，新Pod才会创建。适合需要精确控制更新顺序的场景。

```yaml
updateStrategy:
  type: OnDelete
```

## DaemonSet与其他控制器的区别

| 特性 | DaemonSet | Deployment | StatefulSet |
|------|-----------|------------|-------------|
| Pod数量 | 每节点一个 | 固定副本数 | 固定副本数 |
| 扩缩容 | 跟随节点数变化 | 手动调整replicas | 手动调整replicas |
| 典型场景 | 节点级守护进程 | 无状态应用 | 有状态应用 |
| 存储 | 通常用hostPath | PVC或无 | 独立PVC |

**一句话记忆**：
- 需要每个节点都运行？用DaemonSet
- 普通的无状态应用？用Deployment
- 数据库等有状态应用？用StatefulSet

## 常见问题

### Q1: DaemonSet的Pod没有在某个节点上运行？

这是最常见的问题，按以下顺序排查：

1. **检查节点的污点**：
```bash
kubectl describe node <node-name> | grep Taints
```
如果节点有污点，确保DaemonSet配置了对应的容忍。

2. **检查节点标签是否匹配**：
如果DaemonSet使用了nodeSelector，确保目标节点有对应的标签。

3. **检查节点状态**：
```bash
kubectl get nodes
```
如果节点处于NotReady状态，Pod不会被调度上去。

4. **检查资源是否充足**：
如果节点资源不足（CPU/内存），Pod会处于Pending状态。

### Q2: 如何让DaemonSet在Master节点上也运行？

Master节点默认有污点阻止调度。添加对应的容忍即可：

```yaml
tolerations:
  - key: node-role.kubernetes.io/control-plane
    effect: NoSchedule
  - key: node-role.kubernetes.io/master     # 兼容旧版本
    effect: NoSchedule
```

### Q3: 更新DaemonSet时如何保证服务不中断？

1. **控制更新速度**：设置合理的`maxUnavailable`
2. **配置优雅终止**：给Pod足够的时间完成清理工作

```yaml
spec:
  template:
    spec:
      terminationGracePeriodSeconds: 60
```

对于日志收集器这类服务，短暂的中断通常是可接受的。如果需要更高的可用性，可以考虑使用PodDisruptionBudget。

### Q4: DaemonSet和Static Pod有什么区别？

**Static Pod**是kubelet直接管理的Pod，定义文件放在节点的本地目录（通常是`/etc/kubernetes/manifests/`）。Kubernetes的核心组件（如kube-apiserver、etcd）就是以Static Pod方式运行的。

| 特性 | DaemonSet | Static Pod |
|------|-----------|------------|
| 管理方式 | 通过API Server | kubelet直接管理 |
| 定义位置 | 存储在etcd | 节点本地文件 |
| kubectl操作 | 完全支持 | 只能查看，不能删除 |
| 更新方式 | 滚动更新 | 修改文件后自动重建 |

**一般的守护进程用DaemonSet**，只有Kubernetes核心组件才用Static Pod。

### Q5: 为什么DaemonSet通常使用hostPath？

因为DaemonSet运行的程序往往需要访问节点上的文件，比如：
- 日志收集器需要读取`/var/log`
- 监控代理需要读取`/proc`、`/sys`获取系统指标
- 网络插件需要配置节点的iptables规则

这些都需要直接访问节点文件系统，所以hostPath是DaemonSet的常见配置。

但要注意：**hostPath会带来安全风险**，应该只挂载必要的目录，并尽量使用readOnly模式。

## 小结

DaemonSet是Kubernetes中专门用于"每节点运行一个Pod"场景的控制器：

- **自动调度**：新节点加入时自动创建Pod，节点删除时自动清理
- **节点选择**：通过nodeSelector或nodeAffinity控制运行在哪些节点
- **污点容忍**：通过tolerations在有污点的节点（如Master）上运行
- **滚动更新**：支持RollingUpdate和OnDelete两种策略

记住关键点：
- 日志收集、监控、网络插件等节点级服务适合用DaemonSet
- 要在Master节点运行需要配置污点容忍
- DaemonSet的Pod数量由节点数决定，不能手动设置replicas

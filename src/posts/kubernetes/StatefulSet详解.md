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

# StatefulSet详解：有状态应用的守护者

## 为什么需要StatefulSet？

在学习StatefulSet之前，你需要先理解一个问题：**为什么Deployment不能用来部署数据库？**

Deployment的设计理念是：所有Pod都是"可替换的"。就像工厂流水线上的工人，任何一个人请假了，换一个人顶上就行，没有区别。这种设计非常适合无状态的Web应用——用户的请求被哪个Pod处理都无所谓。

但是数据库不一样。想象一下MySQL主从复制的场景：
- mysql-master是主库，负责写入
- mysql-slave-1和mysql-slave-2是从库，负责读取
- 它们之间有复制关系，从库需要知道主库的地址

如果用Deployment部署，会遇到这些问题：
1. **Pod名称不固定**：Pod重建后名称会变（从`mysql-abc123`变成`mysql-xyz789`），从库怎么找到主库？
2. **存储不独立**：Deployment的Pod共享同一个PVC，而数据库每个实例需要自己独立的数据目录
3. **启动顺序无法控制**：主库必须先启动，从库才能连接。Deployment是并行启动的

**StatefulSet就是为了解决这些问题而设计的**。它给每个Pod一个"身份证"，让有状态应用也能在Kubernetes中优雅地运行。

## StatefulSet的三大承诺

### 1. 稳定的网络标识

StatefulSet中的每个Pod都有固定的名称，格式是`<statefulset名称>-<序号>`：

```
mysql-0   # 第一个Pod，可以约定为主库
mysql-1   # 第二个Pod，从库
mysql-2   # 第三个Pod，从库
```

即使Pod被删除重建，名称也不会变。mysql-0永远是mysql-0。

配合Headless Service（稍后会讲），每个Pod还有固定的DNS名称：
```
mysql-0.mysql-headless.default.svc.cluster.local
mysql-1.mysql-headless.default.svc.cluster.local
```

这样，从库就可以用固定的地址连接主库了。

### 2. 稳定的存储

StatefulSet会为每个Pod创建独立的PVC。Pod和它的PVC是绑定的：
- mysql-0 使用 data-mysql-0
- mysql-1 使用 data-mysql-1
- mysql-2 使用 data-mysql-2

Pod删除后，PVC不会被删除。当Pod重建时，会重新挂载同一个PVC，数据不会丢失。

### 3. 有序的部署和更新

StatefulSet按顺序管理Pod：
- **创建时**：按序号从小到大创建（0→1→2），前一个Pod Ready后才创建下一个
- **删除时**：按序号从大到小删除（2→1→0）
- **更新时**：按序号从大到小更新

这种有序性对于主从架构至关重要——确保主库先启动，从库才能连接。

## StatefulSet与Deployment的对比

| 特性 | Deployment | StatefulSet |
|------|------------|-------------|
| Pod名称 | 随机后缀（app-xyz123） | 固定序号（app-0, app-1） |
| 存储 | 共享或不绑定 | 每个Pod独立PVC |
| 网络标识 | 无固定标识 | 固定DNS名称 |
| 启动顺序 | 并行启动 | 顺序启动 |
| 典型应用 | Web服务、API | 数据库、消息队列、ZooKeeper |

**一句话记忆**：无状态用Deployment，有状态用StatefulSet。

## Headless Service：给Pod一个固定地址

普通的Service会分配一个ClusterIP，客户端访问这个IP时由kube-proxy负载均衡到后端Pod。但StatefulSet的场景不一样——你需要直接访问特定的Pod（比如只访问主库）。

**Headless Service**就是一个没有ClusterIP的Service（`clusterIP: None`）。它不做负载均衡，而是为每个Pod创建DNS记录。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: mysql-headless
spec:
  clusterIP: None        # 关键：设为None就是Headless Service
  selector:
    app: mysql
  ports:
    - port: 3306
```

有了Headless Service后，DNS解析的行为是：
- 查询`mysql-headless` → 返回所有Pod的IP（不是单个ClusterIP）
- 查询`mysql-0.mysql-headless` → 返回mysql-0这个Pod的IP

## 核心配置详解

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  serviceName: mysql-headless    # 必须指定Headless Service的名称
  replicas: 3                    # 副本数
  selector:
    matchLabels:
      app: mysql

  template:                      # Pod模板，和Deployment一样
    metadata:
      labels:
        app: mysql
    spec:
      containers:
        - name: mysql
          image: mysql:8.0
          ports:
            - containerPort: 3306
          volumeMounts:
            - name: data
              mountPath: /var/lib/mysql

  volumeClaimTemplates:          # PVC模板，StatefulSet特有
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: standard
        resources:
          requests:
            storage: 10Gi
```

重点说一下`volumeClaimTemplates`：这不是一个具体的PVC，而是一个"模板"。StatefulSet会根据这个模板，为每个Pod自动创建PVC：
- data-mysql-0
- data-mysql-1
- data-mysql-2

## 更新策略

StatefulSet支持两种更新策略：

### RollingUpdate（默认）

滚动更新，按序号从大到小逐个更新Pod。每个Pod更新成功（Ready）后才继续下一个。

```yaml
updateStrategy:
  type: RollingUpdate
```

**金丝雀发布**：通过`partition`参数可以实现分批更新。比如设置`partition: 2`，则只有序号 >= 2的Pod会被更新，mysql-0和mysql-1保持旧版本。

```yaml
updateStrategy:
  type: RollingUpdate
  rollingUpdate:
    partition: 2  # 只更新mysql-2及以后的Pod
```

验证没问题后，把partition改为1，再改为0，逐步完成全部更新。

### OnDelete

设置为OnDelete后，修改StatefulSet不会自动更新Pod。只有当你手动删除某个Pod时，新创建的Pod才会使用新配置。

这种方式给了你完全的控制权，适合需要精确控制更新顺序的场景。

## Pod管理策略

默认的`OrderedReady`策略要求严格按顺序创建和删除Pod。但有些应用不需要这种顺序保证（比如Cassandra集群的节点是对等的）。

设置`podManagementPolicy: Parallel`可以让Pod并行创建和删除，加快扩缩容速度。

## 常见问题

### Q1: StatefulSet的Pod一直Pending怎么办？

最常见的原因是PVC无法绑定。按这个顺序排查：

1. **检查PVC状态**：
```bash
kubectl get pvc -l app=mysql
```

2. **检查StorageClass是否存在**：
```bash
kubectl get sc
```

3. **如果使用OrderedReady策略**：检查前一个Pod是否Ready。mysql-1要等mysql-0 Ready后才会创建。

### Q2: Pod删除后数据会丢失吗？

**不会**。这是StatefulSet的核心设计：PVC和Pod解耦。

删除Pod → Pod消失 → StatefulSet控制器创建新Pod → 新Pod挂载原来的PVC → 数据还在

只有当你手动删除PVC时，数据才会丢失（如果回收策略是Delete的话）。

**注意**：删除StatefulSet本身不会删除PVC。这是一种保护机制，防止误删数据。清理时需要手动删除PVC：
```bash
kubectl delete pvc -l app=mysql
```

### Q3: 如何实现有状态应用的高可用？

仅靠StatefulSet是不够的，你还需要：

1. **应用层面的高可用机制**：比如MySQL的主从复制、Redis的哨兵模式
2. **合理的探针配置**：确保只有健康的Pod才接收流量
3. **PodDisruptionBudget**：限制同时不可用的Pod数量

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: mysql-pdb
spec:
  minAvailable: 2        # 至少保持2个Pod可用
  selector:
    matchLabels:
      app: mysql
```

### Q4: 如何扩容StatefulSet的存储？

如果StorageClass支持扩容（`allowVolumeExpansion: true`），可以直接编辑PVC：

```bash
kubectl edit pvc data-mysql-0
# 修改 spec.resources.requests.storage 为更大的值
```

某些存储类型可能需要重启Pod才能生效。

### Q5: 什么时候不应该用StatefulSet？

- **应用本身无状态**：比如纯API服务，用Deployment更简单
- **不需要稳定标识**：应用不关心自己是哪个实例
- **不需要独立存储**：所有实例可以共享存储或者不需要持久化

记住：StatefulSet比Deployment复杂，不要为了用而用。

## 小结

StatefulSet是Kubernetes为有状态应用量身定制的控制器：

- **稳定的网络标识**：固定的Pod名称和DNS名称
- **稳定的存储**：每个Pod独立的PVC，Pod重建后数据不丢失
- **有序的操作**：按序号顺序创建、逆序删除和更新

记住关键点：
- 必须配合Headless Service使用
- volumeClaimTemplates为每个Pod创建独立PVC
- 删除StatefulSet不会删除PVC（保护数据）
- 适用于数据库、消息队列等有状态应用

## 参考资源

- [Kubernetes StatefulSet 官方文档](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
- [有状态应用部署最佳实践](https://kubernetes.io/docs/tutorials/stateful-application/)
- [Headless Service 详解](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services)

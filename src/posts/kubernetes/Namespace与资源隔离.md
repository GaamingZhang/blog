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

# Namespace与资源隔离：多租户的基础

## 为什么需要Namespace？

想象你在管理一栋办公楼。楼里有多个公司，每个公司都需要自己的办公空间。你不可能让所有人混在一起办公——财务部的文件不能让其他公司看到，研发团队需要独立的网络环境，每个公司的资源使用也需要单独计费。

在Kubernetes中，**Namespace就像是这栋办公楼里的独立办公室**。一个Kubernetes集群可以被划分成多个Namespace，每个Namespace就是一个相对独立的"虚拟集群"。

典型的使用场景包括：
- **多环境隔离**：开发环境、测试环境、生产环境各用一个Namespace
- **多团队隔离**：前端团队、后端团队、数据团队各有自己的空间
- **多项目隔离**：不同项目的资源放在不同Namespace中

## Namespace的本质

Namespace本质上是一种**逻辑分组机制**。它主要提供两个能力：

1. **名称隔离**：不同Namespace中可以有同名的资源。比如开发环境和生产环境都可以有叫`api-service`的服务，互不冲突。

2. **访问控制边界**：可以基于Namespace设置权限，让用户只能访问自己Namespace内的资源。

但要注意，**Namespace不是万能的隔离方案**：
- 它不提供网络隔离（默认情况下，不同Namespace的Pod可以互相通信）
- 它不限制资源使用（除非配合ResourceQuota）
- 某些资源是集群级别的，不属于任何Namespace（如Node、PersistentVolume）

## 系统默认的Namespace

Kubernetes集群创建后，会自动存在几个Namespace：

| Namespace | 作用 |
|-----------|------|
| `default` | 默认空间。不指定Namespace时，资源就创建在这里 |
| `kube-system` | 系统组件的家。DNS服务、网络插件等都在这里 |
| `kube-public` | 公共资源。所有人（包括未认证用户）都能访问 |
| `kube-node-lease` | 存放节点心跳信息，用于判断节点是否存活 |

初学者常犯的一个错误是把自己的应用部署到`kube-system`里——千万别这样做，那是系统组件专用的空间。

## 哪些资源属于Namespace？

并不是所有资源都受Namespace管辖：

**受Namespace限制的资源**（大多数日常使用的资源）：
- Pod、Deployment、Service、Ingress
- ConfigMap、Secret
- PersistentVolumeClaim
- Role、RoleBinding

**不受Namespace限制的资源**（集群级别）：
- Node（节点本身是物理/虚拟机，不属于任何Namespace）
- PersistentVolume（存储资源，供整个集群使用）
- ClusterRole、ClusterRoleBinding（集群级权限）
- Namespace本身

## 资源配额（ResourceQuota）

光有Namespace还不够。回到办公楼的比喻：如果不限制用电量，某个公司可能把整栋楼的电都用光了。

**ResourceQuota就是给Namespace设定的资源配额**，限制这个空间内最多能使用多少资源。

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: team-quota
  namespace: team-backend
spec:
  hard:
    pods: "20"                # 最多20个Pod
    requests.cpu: "10"        # CPU请求总量不超过10核
    requests.memory: "20Gi"   # 内存请求总量不超过20Gi
    limits.cpu: "20"          # CPU限制总量不超过20核
    limits.memory: "40Gi"     # 内存限制总量不超过40Gi
```

有了ResourceQuota之后：
- 创建Pod时，如果会导致超出配额，Kubernetes会拒绝创建
- 运维人员可以合理分配集群资源给各团队
- 防止某个团队"独占"集群资源

**重要提示**：一旦设置了ResourceQuota，该Namespace下创建的Pod就**必须**声明资源请求和限制，否则会创建失败。这是为了确保配额能够正确计算。

## 默认资源限制（LimitRange）

ResourceQuota管的是总量，那单个Pod呢？如果有人创建了一个超大的Pod，申请100核CPU，配额一下就用完了。

**LimitRange用来约束单个Pod/容器的资源使用范围**，并且可以设置默认值。

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
  namespace: team-backend
spec:
  limits:
    - type: Container
      default:          # 默认限制（不设置时自动应用）
        cpu: "500m"
        memory: "256Mi"
      defaultRequest:   # 默认请求（不设置时自动应用）
        cpu: "100m"
        memory: "128Mi"
      max:              # 最大允许值
        cpu: "2"
        memory: "2Gi"
      min:              # 最小允许值
        cpu: "50m"
        memory: "64Mi"
```

这样设置后：
- 创建Pod时不写资源配置？自动填充默认值
- 申请的资源超出max？拒绝创建
- 申请的资源低于min？也拒绝创建

**ResourceQuota和LimitRange的关系**：
- LimitRange确保每个Pod都有合理的资源配置
- ResourceQuota确保所有Pod加起来不超过总量
- 两者配合使用，才能实现完善的资源管理

## 网络隔离（NetworkPolicy）

前面说过，Namespace默认不提供网络隔离。但很多场景下，你希望生产环境的Pod不能被开发环境访问到。

这时候需要用**NetworkPolicy**来实现网络隔离。

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-from-other-namespaces
  namespace: production
spec:
  podSelector: {}  # 应用到这个Namespace的所有Pod
  policyTypes:
    - Ingress
  ingress:
    - from:
        - podSelector: {}  # 只允许同Namespace的Pod访问
```

这个策略的含义是：production空间内的Pod，只接受来自同一空间的流量，拒绝其他Namespace的访问。

**注意**：NetworkPolicy需要网络插件支持（如Calico、Cilium）。如果你的集群用的是不支持NetworkPolicy的网络插件，这些规则将不会生效。

## 跨Namespace访问服务

虽然有了隔离，但有时候确实需要跨Namespace访问服务。比如，所有Namespace都需要访问同一个数据库服务。

Kubernetes的DNS会为Service创建完整域名，格式是：
```
<service-name>.<namespace>.svc.cluster.local
```

所以，如果你在`team-a` Namespace中，想访问`team-b` Namespace的`api-service`，只需要使用：
```
http://api-service.team-b.svc.cluster.local:8080
```

在同一个Namespace内，可以直接用服务名（如`api-service`），因为会自动补全当前Namespace。

## 常见问题

### Q1: 删除Namespace时卡在Terminating状态怎么办？

这是非常常见的问题。删除Namespace会级联删除里面的所有资源，如果某个资源卡住了，Namespace就无法完成删除。

**排查思路**：

1. **找出卡住的资源**：通常是某些带有Finalizer的资源无法正常清理
2. **检查Finalizer**：Finalizer是一种"删除前必须完成某些操作"的机制，如果操作无法完成，资源就删不掉

**最后手段**（生产环境慎用）：
```bash
# 强制移除Finalizer，让Namespace可以删除
kubectl patch ns <namespace-name> -p '{"metadata":{"finalizers":null}}' --type=merge
```

### Q2: ResourceQuota和LimitRange有什么区别？

简单记忆：
- **ResourceQuota**：管"总量"，限制整个Namespace能用多少资源
- **LimitRange**：管"个体"，限制单个Pod/容器的资源范围，并提供默认值

两者应该配合使用：
- 只有ResourceQuota：大Pod可能一下用光配额
- 只有LimitRange：总资源使用量无法控制
- 两者配合：既限制个体，又限制总量

### Q3: Pod创建失败提示"exceeded quota"怎么办？

说明Namespace的资源配额用完了。处理方式：
1. 删除不需要的资源，释放配额
2. 优化Pod的资源请求，减少浪费
3. 申请增加Namespace的配额

### Q4: 如何在不同Namespace间共享ConfigMap或Secret？

Kubernetes原生不支持跨Namespace共享这些资源。每个Namespace是独立的。

解决方案：
1. **手动复制**：在多个Namespace创建相同内容的ConfigMap/Secret
2. **自动同步工具**：使用kubed、Reflector等工具自动同步
3. **外部配置中心**：使用Consul、Apollo等外部系统存储配置

### Q5: 怎样实现真正的"多租户"隔离？

单靠Namespace是不够的，需要组合多种机制：

| 隔离维度 | 使用的机制 |
|---------|-----------|
| 资源隔离 | ResourceQuota + LimitRange |
| 网络隔离 | NetworkPolicy |
| 权限隔离 | RBAC（Role/RoleBinding） |
| 存储隔离 | 专用StorageClass + 存储配额 |

如果需要更强的隔离（比如多个互不信任的租户），可能需要考虑使用多个独立的集群，或者采用虚拟集群方案（如vcluster）。

## 小结

Namespace是Kubernetes实现资源隔离的基础机制：

- **核心作用**：提供名称隔离和访问控制边界
- **不是万能的**：默认不隔离网络，不限制资源
- **配合使用**：ResourceQuota限制总量，LimitRange设置默认值和范围，NetworkPolicy隔离网络

记住关键点：
- 大多数资源属于Namespace，但Node、PV等是集群级别的
- 跨Namespace访问Service用完整DNS名称
- 真正的多租户隔离需要多种机制配合

## 参考资源

- [Kubernetes Namespace 官方文档](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/)
- [ResourceQuota 配额管理](https://kubernetes.io/docs/concepts/policy/resource-quotas/)
- [LimitRange 默认限制](https://kubernetes.io/docs/concepts/policy/limit-range/)

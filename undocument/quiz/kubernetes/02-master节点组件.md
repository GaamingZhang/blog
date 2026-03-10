---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Master节点
---

# Kubernetes Master节点组件详解

## 什么是Master节点？

Master节点是Kubernetes集群的"大脑"，负责管理集群状态、调度工作负载、响应API请求。如果把Kubernetes比作一个公司，Master节点就是公司的管理层——负责决策、协调、监控，但不直接参与"生产"工作。

Worker节点才是真正运行应用容器的地方，Master节点只运行控制平面组件。

## Master节点的核心组件

一个标准的Master节点运行以下四个核心组件：

```
┌─────────────────────────────────────────────────────────────┐
│                      Master Node                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ API Server  │  │ Scheduler   │  │ Controller Manager  │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                      etcd                            │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### 1. kube-apiserver

**角色**：集群的统一入口，所有请求都经过它

kube-apiserver是Kubernetes控制平面的前端，也是唯一一个直接与etcd通信的组件。它就像公司的前台接待处——所有来访请求都要先经过这里。

#### 核心职责

- **认证（Authentication）**：你是谁？
- **授权（Authorization）**：你有权限做这件事吗？
- **准入控制（Admission Control）**：你的请求符合规则吗？
- **RESTful API**：提供资源的增删改查接口
- **数据存储**：将资源状态写入etcd

#### 工作流程

```
kubectl get pods
    ↓
认证：验证用户身份（证书/token/用户名密码）
    ↓
授权：检查RBAC规则
    ↓
准入控制：执行ValidatingAdmissionWebhook等
    ↓
从etcd读取数据
    ↓
返回结果给用户
```

#### 关键配置

```yaml
apiServer:
  extraArgs:
    authorization-mode: Node,RBAC
    enable-admission-plugins: NodeRestriction,PodSecurityPolicy
    service-account-issuer: https://kubernetes.default.svc
    service-account-signing-key-file: /etc/kubernetes/pki/sa.key
```

#### 高可用部署

生产环境通常在API Server前面部署负载均衡器：

```
                    ┌─────────────┐
                    │ Load Balancer│
                    │  (VIP:6443)  │
                    └──────┬──────┘
           ┌───────────────┼───────────────┐
           ↓               ↓               ↓
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │ API Server  │ │ API Server  │ │ API Server  │
    │  Master1    │ │  Master2    │ │  Master3    │
    └─────────────┘ └─────────────┘ └─────────────┘
```

### 2. etcd

**角色**：集群的数据库，存储所有状态信息

etcd是一个高可用的键值存储系统，是Kubernetes的"记忆"。集群中所有的状态数据——Pod信息、Service配置、Secret、ConfigMap等——都存储在etcd中。

#### 核心特点

- **强一致性**：基于Raft协议，保证数据一致性
- **高可用**：通常部署3或5个节点的集群
- **Watch机制**：支持监听数据变化，实现事件驱动

#### 数据结构

etcd中的数据是扁平的键值对，Kubernetes使用前缀组织：

```
/registry/pods/default/nginx-pod
/registry/services/default/nginx-service
/registry/deployments/default/nginx-deployment
/registry/secrets/default/db-password
```

#### 关键配置

```bash
etcd \
  --name etcd1 \
  --data-dir /var/lib/etcd \
  --listen-client-urls https://192.168.1.10:2379 \
  --advertise-client-urls https://192.168.1.10:2379 \
  --listen-peer-urls https://192.168.1.10:2380 \
  --initial-advertise-peer-urls https://192.168.1.10:2380 \
  --initial-cluster etcd1=https://192.168.1.10:2380,etcd2=https://192.168.1.11:2380,etcd3=https://192.168.1.12:2380 \
  --initial-cluster-token my-etcd-token \
  --initial-cluster-state new
```

#### etcd集群健康检查

```bash
etcdctl endpoint status --cluster -w table

+------------------------+------------------+---------+---------+-----------+------------+
|        ENDPOINT        |        ID        | VERSION | DB SIZE | IS LEADER | IS LEARNER |
+------------------------+------------------+---------+---------+-----------+------------+
| https://192.168.1.10:2379 | 3e6e9c4b5f7a8d9c |   3.5.9 |   2.1 GB |      true |      false |
| https://192.168.1.11:2379 | 7a8b9c0d1e2f3a4b |   3.5.9 |   2.1 GB |     false |      false |
| https://192.168.1.12:2379 | b5c6d7e8f9a0b1c2 |   3.5.9 |   2.1 GB |     false |      false |
+------------------------+------------------+---------+---------+-----------+------------+
```

#### 备份与恢复

```bash
etcdctl snapshot save /backup/etcd-snapshot.db \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key

etcdctl snapshot restore /backup/etcd-snapshot.db
```

### 3. kube-scheduler

**角色**：决定Pod运行在哪个节点上

kube-scheduler是集群的"人力资源部门"，负责为新创建的Pod选择最合适的工作节点。

#### 调度流程

```
Pod创建请求
    ↓
调度器监听到未调度的Pod
    ↓
【过滤阶段】排除不满足条件的节点
    ↓
【打分阶段】为剩余节点打分
    ↓
选择得分最高的节点
    ↓
将Pod绑定到该节点（更新Pod的spec.nodeName）
```

#### 过滤阶段（Predicates）

排除不满足硬性条件的节点：

- **PodFitsResources**：节点资源是否足够
- **PodFitsHostPorts**：端口是否冲突
- **PodMatchNodeSelector**：节点选择器是否匹配
- **PodToleratesNodeTaints**：污点容忍是否满足
- **CheckNodeCondition**：节点是否Ready

#### 打分阶段（Priorities）

为候选节点打分，选择最优：

- **LeastRequestedPriority**：资源使用率低的优先
- **BalancedResourceAllocation**：CPU和内存使用均衡的优先
- **NodeAffinityPriority**：节点亲和性匹配的优先
- **InterPodAffinityPriority**：Pod亲和性匹配的优先
- **TaintTolerationPriority**：污点容忍度高的优先

#### 自定义调度器

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  schedulerName: my-scheduler
  containers:
  - name: nginx
    image: nginx
```

### 4. kube-controller-manager

**角色**：运行各种控制器，确保集群状态符合预期

kube-controller-manager是集群的"执行部门"，包含多个控制器，每个控制器负责一种资源的调谐。

#### 核心控制器

**Node Controller**
- 监控节点健康状态
- 当节点不可用时，驱逐Pod
- 管理节点心跳超时

**Replication Controller**
- 确保Pod副本数量符合预期
- 当Pod异常退出时，创建新Pod

**Deployment Controller**
- 管理Deployment的滚动更新
- 创建和管理ReplicaSet

**Service Controller**
- 管理Service与LoadBalancer的关联
- 云环境下创建云负载均衡器

**Endpoint Controller**
- 维护Service与Pod的映射关系
- 更新Endpoints对象

**Namespace Controller**
- 删除Namespace时清理资源

**ServiceAccount Controller**
- 为新Namespace创建默认ServiceAccount

**Token Controller**
- 为ServiceAccount生成JWT Token

#### 控制器工作模式

所有控制器都遵循"调谐循环"模式：

```
while true:
    获取当前状态
    获取期望状态
    if 当前状态 != 期望状态:
        执行操作使当前状态趋向期望状态
    等待一段时间
```

以ReplicaSet控制器为例：

```go
for {
    expectedReplicas := rs.Spec.Replicas
    currentReplicas := len(getPodsOwnedByRS(rs))
    
    if currentReplicas < expectedReplicas {
        createPod()
    } else if currentReplicas > expectedReplicas {
        deletePod()
    }
    
    time.Sleep(1 * time.Second)
}
```

## 组件间的协作关系

理解组件如何协作对于排查问题至关重要。

### 创建Pod的完整流程

```
用户执行: kubectl run nginx --image=nginx
    ↓
【1. API Server】
    认证 → 授权 → 准入控制 → 写入etcd
    ↓
【2. Scheduler】
    监听到未调度的Pod → 过滤节点 → 打分 → 绑定节点 → 更新etcd
    ↓
【3. Controller Manager】
    ReplicaSet控制器确保副本数正确
    ↓
【4. Worker节点上的kubelet】
    监听到分配给自己的Pod → 拉取镜像 → 创建容器 → 上报状态
    ↓
【5. API Server】
    更新Pod状态到etcd
```

### 各组件与etcd的关系

```
┌─────────────────────────────────────────────────────────────┐
│                         etcd                                 │
│                    (唯一数据源)                               │
└─────────────────────────────────────────────────────────────┘
                            ↑
                            │ 只有API Server直接读写
                            │
                    ┌───────┴───────┐
                    │  API Server   │
                    └───────┬───────┘
                            │
            ┌───────────────┼───────────────┐
            ↓               ↓               ↓
    ┌───────────────┐ ┌───────────────┐ ┌───────────────┐
    │   Scheduler   │ │  Controller   │ │    Kubelet    │
    │               │ │   Manager     │ │               │
    │  只读+更新    │ │    只读+更新   │ │    只读+更新   │
    │  Pod绑定      │ │  资源状态      │ │  Pod状态      │
    └───────────────┘ └───────────────┘ └───────────────┘
```

**关键点**：
- API Server是唯一直接读写etcd的组件
- 其他组件通过API Server间接访问数据
- 这保证了数据一致性和安全性

## 组件的高可用部署

### 堆叠式etcd（Stacked etcd）

etcd与Master组件部署在同一节点：

```
Master1: API Server + Scheduler + Controller Manager + etcd
Master2: API Server + Scheduler + Controller Manager + etcd
Master3: API Server + Scheduler + Controller Manager + etcd
```

优点：部署简单，节省资源
缺点：etcd故障可能影响Master节点

### 外部etcd（External etcd）

etcd独立部署：

```
Master1: API Server + Scheduler + Controller Manager
Master2: API Server + Scheduler + Controller Manager
Master3: API Server + Scheduler + Controller Manager

etcd1: etcd
etcd2: etcd
etcd3: etcd
```

优点：etcd独立扩展，故障隔离
缺点：需要更多节点

## 组件故障的影响

| 组件故障 | 影响 |
|----------|------|
| API Server | 无法执行kubectl命令，无法创建/更新资源，但已运行的Pod不受影响 |
| etcd | 集群完全瘫痪，无法读取任何状态信息 |
| Scheduler | 新Pod无法调度，一直Pending，已运行的Pod不受影响 |
| Controller Manager | 控制循环停止，副本数异常不会自动修复，已运行的Pod不受影响 |

**重要结论**：Master组件故障不会影响已运行的Pod，只会影响集群管理能力。

## 常见问题

### Q1: 如何查看组件状态？

```bash
kubectl get cs

NAME                 STATUS    MESSAGE                         ERROR
scheduler            Healthy   ok
controller-manager   Healthy   ok
etcd-0               Healthy   {"health":"true","reason":""}
```

### Q2: 组件日志在哪里？

```bash
journalctl -u kube-apiserver -f
journalctl -u kube-scheduler -f
journalctl -u kube-controller-manager -f
journalctl -u etcd -f
```

### Q3: 如何排查组件启动失败？

```bash
systemctl status kube-apiserver
journalctl -xeu kube-apiserver
```

常见原因：
- 证书过期或配置错误
- 端口被占用
- etcd连接失败
- 内存不足

### Q4: Master节点可以运行工作负载吗？

默认情况下，Master节点有污点，阻止普通Pod调度：

```bash
kubectl describe node master1 | grep Taints
Taints: node-role.kubernetes.io/control-plane:NoSchedule
```

如果需要运行工作负载，可以去除污点：

```bash
kubectl taint nodes master1 node-role.kubernetes.io/control-plane:NoSchedule-
```

## 最佳实践

1. **Master节点数量**：生产环境至少3个，确保高可用
2. **etcd备份**：定期备份，每天至少一次
3. **资源规划**：Master节点建议4核8G起步，etcd建议使用SSD
4. **证书管理**：监控证书过期时间，提前更新
5. **日志收集**：收集组件日志，便于问题排查

## 总结

Master节点的四个核心组件各司其职：

| 组件 | 职责 | 类比 |
|------|------|------|
| API Server | 统一入口，认证授权 | 前台接待 |
| etcd | 数据存储 | 档案室 |
| Scheduler | 调度决策 | 人力资源 |
| Controller Manager | 状态调谐 | 执行部门 |

理解这些组件的职责和协作方式，是深入理解Kubernetes的基础，也是排查集群问题的关键。

## 参考资源

- [Kubernetes组件架构](https://kubernetes.io/docs/concepts/overview/components/)
- [etcd官方文档](https://etcd.io/docs/)
- [kube-scheduler配置](https://kubernetes.io/docs/reference/scheduling/config/)

---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - etcd
  - 高可用
---

# etcd投票能力不足时Master组件的运行状态

## 问题背景

在Kubernetes高可用集群中，etcd通常以3节点或5节点集群形式部署。当部分节点故障导致etcd集群失去"投票能力"（即无法达成多数派共识）时，Master组件会如何表现？

这是一个非常实际的问题，理解它对于集群运维至关重要。

## etcd的Raft共识机制

### 为什么需要多数派？

etcd使用Raft协议保证数据一致性。Raft的核心原则是：**只有获得多数派节点的支持，才能进行写入操作**。

对于N个节点的集群，需要至少 `(N/2) + 1` 个节点存活才能正常工作：

| 集群规模 | 多数派 | 允许故障节点数 |
|----------|--------|----------------|
| 1 | 1 | 0 |
| 3 | 2 | 1 |
| 5 | 3 | 2 |
| 7 | 4 | 3 |

### 投票能力不足的场景

以3节点etcd集群为例：

```
正常状态:
┌─────────┐   ┌─────────┐   ┌─────────┐
│  etcd1  │   │  etcd2  │   │  etcd3  │
│ Leader  │   │Follower │   │Follower │
└─────────┘   └─────────┘   └─────────┘
     ↑              ↑             ↑
     └──────────────┴─────────────┘
              3节点全部正常

故障场景1：1个节点故障
┌─────────┐   ┌─────────┐   ┌─────────┐
│  etcd1  │   │  etcd2  │   │  etcd3  │
│ Leader  │   │Follower │   │   DOWN  │
└─────────┘   └─────────┘   └─────────┘
     ↑              ↑             
     └──────────────┴─────────────
        2/3存活，仍有多数派，集群可用

故障场景2：2个节点故障
┌─────────┐   ┌─────────┐   ┌─────────┐
│  etcd1  │   │  etcd2  │   │  etcd3  │
│ Leader  │   │   DOWN  │   │   DOWN  │
└─────────┘   └─────────┘   └─────────┘
     ↑              
     └──────────────
     1/3存活，失去多数派，集群不可用
```

## 投票能力不足时各组件的行为

### 1. etcd本身的行为

当etcd集群失去多数派时：

**Leader节点**：
- 检测到无法获得多数派响应
- 自动降级为Follower或Candidate状态
- 停止处理写入请求
- 只读请求可能仍能响应（取决于配置）

**Follower节点**：
- 无法与Leader通信
- 开始新的选举
- 无法获得多数派投票
- 保持Candidate状态

```bash
etcdctl endpoint status --cluster -w table

# 正常状态
+------------------------+------------------+---------+---------+-----------+
|        ENDPOINT        |        ID        | VERSION | DB SIZE | IS LEADER |
+------------------------+------------------+---------+---------+-----------+
| https://etcd1:2379     | 3e6e9c4b5f7a8d9c |   3.5.9 |   2.1 GB |      true |
| https://etcd2:2379     | 7a8b9c0d1e2f3a4b |   3.5.9 |   2.1 GB |     false |
| https://etcd3:2379     | b5c6d7e8f9a0b1c2 |   3.5.9 |   2.1 GB |     false |
+------------------------+------------------+---------+---------+-----------+

# 投票能力不足时
+------------------------+------------------+---------+---------+-----------+
|        ENDPOINT        |        ID        | VERSION | DB SIZE | IS LEADER |
+------------------------+------------------+---------+---------+-----------+
| https://etcd1:2379     | 3e6e9c4b5f7a8d9c |   3.5.9 |   2.1 GB |     false |  ← 降级
| https://etcd2:2379     |        -         |    -    |    -    |     -     |  ← 不可达
| https://etcd3:2379     |        -         |    -    |    -    |     -     |  ← 不可达
+------------------------+------------------+---------+---------+-----------+
```

### 2. kube-apiserver的行为

API Server是唯一直接与etcd通信的组件。当etcd不可用时：

**写入操作**：
- 创建Pod、Deployment等 → **失败**
- 更新资源状态 → **失败**
- 删除资源 → **失败**

**读取操作**：
- 从etcd读取 → **失败**
- 但可能从缓存返回数据（取决于配置）

**健康检查**：
- livenessProbe检查 `/livez` → 可能失败
- readinessProbe检查 `/readyz` → 失败

```bash
kubectl get pods
# 返回错误
The connection to the server 192.168.1.10:6443 was refused

# 或
Error from server: etcdserver: unhealthy cluster
```

**API Server日志**：

```
E0115 10:00:00.000000 1 controller.go:114] Unable to perform initial Kubernetes service initialization: etcdserver: unhealthy cluster
E0115 10:00:01.000000 1 storage_factory.go:200] Unable to create storage backend: etcdserver: unhealthy cluster
```

### 3. kube-scheduler的行为

Scheduler需要通过API Server获取未调度的Pod，并更新Pod的绑定信息。

**当API Server不可用时**：

- 无法获取待调度的Pod
- 无法更新Pod的绑定信息
- Leader选举机制失效

**Leader选举机制**：

Scheduler使用etcd进行Leader选举（通过ConfigMap）：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kube-scheduler
spec:
  containers:
  - command:
    - kube-scheduler
    - --leader-elect=true
    - --leader-elect-resource-namespace=kube-system
    - --leader-elect-resource-lease=configmaps
```

当etcd不可用时：
- 无法参与Leader选举
- 现有Leader失去Leader身份
- 所有Scheduler实例停止工作

**Scheduler日志**：

```
E0115 10:00:00.000000 1 leaderelection.go:325] error retrieving resource lock kube-system/kube-scheduler: etcdserver: unhealthy cluster
I0115 10:00:01.000000 1 leaderelection.go:282] failed to renew lease kube-system/kube-scheduler: etcdserver: unhealthy cluster
```

### 4. kube-controller-manager的行为

Controller Manager的行为与Scheduler类似：

- 无法从API Server获取资源状态
- 无法更新资源状态
- Leader选举失效
- 所有控制循环停止

**影响**：
- 副本数异常不会自动修复
- 节点故障不会触发Pod驱逐
- Service的Endpoints不会更新

**Controller Manager日志**：

```
E0115 10:00:00.000000 1 leaderelection.go:325] error retrieving resource lock kube-system/kube-controller-manager: etcdserver: unhealthy cluster
```

### 5. kubelet的行为

kubelet运行在每个节点上，它的行为比较特殊：

**与Master的通信**：
- 定期向API Server上报节点状态 → **失败**
- 从API Server获取Pod配置 → **失败**
- 但kubelet会缓存已获取的Pod配置

**对已运行Pod的影响**：
- **Pod继续运行**：kubelet不依赖etcd来运行容器
- 健康检查继续执行：liveness/readiness探针正常工作
- 容器重启：如果容器崩溃，kubelet会根据restartPolicy重启

**关键点**：kubelet是相对独立的，即使Master完全不可用，已运行的Pod仍能继续工作。

```
┌─────────────────────────────────────────────────────────────┐
│                     Worker Node                              │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                   kubelet                            │    │
│  │                                                      │    │
│  │  本地缓存Pod配置 ──────→ 继续管理容器运行             │    │
│  │                                                      │    │
│  │  健康检查 ────────────→ 正常执行                      │    │
│  │                                                      │    │
│  │  上报状态 ────────────→ 失败（但不影响容器）          │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                      │
│  │  Pod A  │  │  Pod B  │  │  Pod C  │  ← 继续运行          │
│  └─────────┘  └─────────┘  └─────────┘                      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 各组件行为总结

| 组件 | 行为 | 对业务的影响 |
|------|------|--------------|
| etcd | 停止服务，无法读写 | 集群控制平面瘫痪 |
| API Server | 无法处理请求 | 无法执行任何kubectl命令 |
| Scheduler | 停止调度 | 新Pod无法调度，一直Pending |
| Controller Manager | 停止控制循环 | 副本数异常不修复，节点故障不驱逐 |
| kubelet | 继续管理本地容器 | 已运行Pod不受影响 |
| kube-proxy | 继续维护iptables | Service转发正常 |

**核心结论**：etcd不可用时，**集群管理能力丧失，但已运行的业务Pod不受影响**。

## 恢复流程

### 场景1：临时网络分区

如果只是网络问题导致暂时失去多数派：

```
网络恢复 → etcd自动重新选举 → 集群恢复正常
```

### 场景2：节点永久故障

如果节点硬件故障无法恢复：

**步骤1：确认当前状态**

```bash
etcdctl member list -w table

+------------------+---------+-------+-----------+
|        ID        | STATUS  | NAME  | ENDPOINT  |
+------------------+---------+-------+-----------+
| 3e6e9c4b5f7a8d9c | started | etcd1 | etcd1:2379|
| 7a8b9c0d1e2f3a4b | started | etcd2 | etcd2:2379|  ← 故障
| b5c6d7e8f9a0b1c2 | started | etcd3 | etcd3:2379|  ← 故障
+------------------+---------+-------+-----------+
```

**步骤2：移除故障节点**

```bash
etcdctl member remove 7a8b9c0d1e2f3a4b
etcdctl member remove b5c6d7e8f9a0b1c2
```

**步骤3：添加新节点**

```bash
etcdctl member add etcd4 --peer-urls=https://etcd4:2380
etcdctl member add etcd5 --peer-urls=https://etcd5:2380
```

**步骤4：启动新etcd节点**

```bash
etcd --name etcd4 \
  --initial-cluster etcd1=https://etcd1:2380,etcd4=https://etcd4:2380,etcd5=https://etcd5:2380 \
  --initial-cluster-state existing
```

### 场景3：完全灾难恢复

如果所有etcd节点都丢失，需要从备份恢复：

```bash
etcdctl snapshot restore /backup/etcd-snapshot.db \
  --name etcd1 \
  --initial-cluster etcd1=https://etcd1:2380,etcd2=https://etcd2:2380,etcd3=https://etcd3:2380 \
  --initial-cluster-token my-etcd-token \
  --initial-advertise-peer-urls https://etcd1:2380
```

## 预防措施

### 1. 合理规划集群规模

| 场景 | 推荐配置 | 说明 |
|------|----------|------|
| 开发/测试 | 1节点etcd | 不需要高可用 |
| 小型生产 | 3节点etcd | 允许1节点故障 |
| 大型生产 | 5节点etcd | 允许2节点故障 |

### 2. 监控etcd健康

```yaml
groups:
- name: etcd
  rules:
  - alert: etcdInsufficientMembers
    expr: count(up{job="etcd"} == 1) < 2
    for: 3m
    labels:
      severity: critical
    annotations:
      summary: "etcd集群成员不足"
      description: "etcd集群只有 {{ $value }} 个成员在线，无法维持多数派"
```

### 3. 定期备份

```bash
#!/bin/bash
etcdctl snapshot save /backup/etcd-$(date +%Y%m%d).db
```

### 4. 使用外部etcd

将etcd与Master节点分离部署，降低故障影响范围：

```
Master节点: API Server + Scheduler + Controller Manager
etcd节点: 独立部署，使用SSD存储
```

## 常见问题

### Q1: etcd不可用时，如何查看Pod状态？

无法通过kubectl查看，但可以：

```bash
# 在节点上直接查看容器
crictl ps
docker ps

# 查看kubelet缓存
curl http://localhost:10250/pods
```

### Q2: etcd不可用时，能否创建新Pod？

不能。创建Pod需要写入etcd，而etcd不可用时无法写入。

### Q3: etcd不可用时，Service还能访问吗？

可以。kube-proxy维护的iptables规则仍然有效，Service转发正常工作。

### Q4: 如何快速判断etcd是否健康？

```bash
etcdctl endpoint health --cluster

https://etcd1:2379 is healthy: successfully committed proposal
https://etcd2:2379 is healthy: successfully committed proposal
https://etcd3:2379 is healthy: successfully committed proposal
```

### Q5: 3节点etcd集群，1节点故障后需要修复吗？

不需要立即修复。集群仍能正常工作（2/3多数派）。但建议尽快修复，因为再故障1个节点就会导致集群不可用。

## 总结

当etcd投票能力不足时：

1. **etcd本身**：停止服务，无法进行读写操作
2. **API Server**：无法处理请求，集群管理入口失效
3. **Scheduler**：停止调度，新Pod无法分配节点
4. **Controller Manager**：停止控制循环，异常状态无法自动修复
5. **kubelet**：继续管理本地容器，已运行Pod不受影响
6. **kube-proxy**：继续维护转发规则，Service正常工作

**关键认知**：etcd故障影响的是"管理能力"，而非"业务运行"。已部署的Pod会继续运行，只是无法进行管理操作。

## 参考资源

- [etcd灾难恢复](https://etcd.io/docs/v3.5/op-guide/recovery/)
- [Raft共识算法](https://raft.github.io/)
- [Kubernetes高可用拓扑](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/ha-topology/)

---
date: 2026-01-25
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# Deployment 工作原理与核心机制

## 引言

Deployment 是 Kubernetes 中最常用的工作负载资源，但很多初学者只是机械地编写 YAML 配置，却不理解背后的工作机制。本文将深入讲解 Deployment 的核心原理，帮助你真正理解它是如何工作的。

## Deployment 的设计哲学

### 为什么需要 Deployment？

在 Kubernetes 早期版本中，人们直接使用 ReplicationController 管理 Pod。但这种方式有个问题：**如何优雅地更新应用？**

假设你有一个运行着 v1 版本的应用，现在要升级到 v2。如果直接修改 ReplicationController，它会：
1. 立即删除所有旧 Pod
2. 创建新 Pod

这会导致服务中断。用户的请求会失败，这在生产环境是不可接受的。

**Deployment 的核心目标就是解决"零停机更新"问题**。它不是简单地管理 Pod，而是管理 Pod 的**变更过程**。

### Deployment 的三层架构

```
Deployment（声明期望状态）
    ↓ 管理
ReplicaSet（维护Pod数量）
    ↓ 管理
Pod（实际运行的容器）
```

这个三层架构的设计非常巧妙：

- **Deployment** 负责管理**版本变更**。每次更新时，它会创建一个新的 ReplicaSet
- **ReplicaSet** 负责维护**Pod 数量**。它确保指定数量的 Pod 副本始终运行
- **Pod** 是实际运行的工作负载

为什么要有 ReplicaSet 这个中间层？因为**滚动更新需要同时存在新旧两个版本的 Pod**。

想象更新过程：
```
更新前:
ReplicaSet-v1 → Pod-v1 (3个)

更新中:
ReplicaSet-v1 → Pod-v1 (2个)  ←减少中
ReplicaSet-v2 → Pod-v2 (1个)  ←增加中

更新后:
ReplicaSet-v1 → (保留，副本数=0，用于回滚)
ReplicaSet-v2 → Pod-v2 (3个)
```

Deployment 通过**控制两个 ReplicaSet 的副本数**来实现滚动更新。这就是为什么需要 ReplicaSet 作为中间层。

## 核心字段的工作机制

### replicas：不只是数字

```yaml
spec:
  replicas: 3
```

replicas 看起来只是一个数字，但它触发的是一个完整的**调谐循环**：

```
实际状态: 2个Pod正在运行
期望状态: 3个Pod应该运行
---
Deployment Controller发现差异
    ↓
计算需要创建1个Pod
    ↓
更新ReplicaSet的副本数
    ↓
ReplicaSet Controller创建新Pod
    ↓
持续监控，直到3个Pod都Running
```

这个过程叫做 **Reconciliation Loop**（调谐循环），是 Kubernetes 的核心设计模式。Controller 不断地对比"期望状态"和"实际状态"，自动采取行动来消除差异。

### selector：Pod 的"身份识别系统"

```yaml
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx  # 必须与selector匹配
```

selector 的工作原理：

1. **Deployment 并不直接"拥有" Pod**。它通过 label selector 来"认领" Pod
2. 任何带有 `app: nginx` 标签的 Pod 都会被这个 Deployment 管理
3. 如果你手动创建了一个标签匹配的 Pod，Deployment 会认为"副本数超了"，可能删除它

这种设计带来了灵活性。比如你可以：
- 手动修改某个 Pod 的标签，把它从 Deployment 的管理中"移除"
- 使用复杂的 selector 表达式（`matchExpressions`）来实现更精细的控制

**关键约束**：template 中的 labels 必须是 selector 的超集。否则 Deployment 创建的 Pod 无法被自己识别，会陷入无限循环创建的错误状态。

### strategy：滚动更新的精髓

#### RollingUpdate 算法

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1        # 最多允许超出目标副本数1个
    maxUnavailable: 0  # 最多允许0个Pod不可用
```

滚动更新的核心是**两个参数的博弈**：

**maxSurge（最大浪涌）**
- 允许临时超出目标副本数的 Pod 数量
- 值越大，更新越快（因为可以快速创建新 Pod）
- 代价是临时消耗更多资源

**maxUnavailable（最大不可用）**
- 允许暂时不可用的 Pod 数量
- 值越大，更新越快（可以快速删除旧 Pod）
- 代价是可能影响服务可用性

举例说明（假设 replicas=10）：

**场景1：快速更新，资源充足**
```yaml
maxSurge: 3
maxUnavailable: 2
```
更新过程：
1. 快速创建3个新Pod（总数13）
2. 等待新Pod就绪
3. 删除2个旧Pod（总数11）
4. 重复，直到全部更新完成

特点：速度快，但需要额外资源


**场景2：零停机，资源紧张**
```yaml
maxSurge: 1
maxUnavailable: 0
```
更新过程：
1. 创建1个新Pod（总数11）
2. 等待新Pod就绪
3. 删除1个旧Pod（总数10）
4. 重复10次

特点：绝对零停机，但更新慢


**场景3：快速重建，允许中断**
```yaml
maxSurge: 0
maxUnavailable: 100%
```
这等同于 Recreate 策略：删除所有旧 Pod，然后创建新 Pod。

#### 滚动更新的内部流程

```
第1步: 创建新的ReplicaSet
    ReplicaSet-v2 (replicas: 0)

第2步: 调整副本数（第1轮）
    计算: 新版本应有 = min(maxSurge, 目标副本数)
    ReplicaSet-v1: 10 → 9   （减少maxUnavailable个）
    ReplicaSet-v2: 0 → 1    （增加到maxSurge限制）

第3步: 等待新Pod就绪
    检查ReadinessProbe
    确认新Pod可以接收流量

第4步: 重复调整
    每次循环检查:
    - 旧版本Pod还有多少？
    - 新版本Pod就绪了多少？
    - 是否满足maxSurge和maxUnavailable约束？

第5步: 完成更新
    ReplicaSet-v1: 0
    ReplicaSet-v2: 10
    保留旧ReplicaSet用于回滚
```

### minReadySeconds：防止"闪现Pod"

```yaml
spec:
  minReadySeconds: 30
```

这个字段常被忽视，但它很重要。作用是：Pod 就绪后还要等待指定秒数，才被认为"真正可用"。

为什么需要它？考虑这个场景：
1. 新 Pod 启动，ReadinessProbe 通过
2. 滚动更新认为新 Pod 可用，删除旧 Pod
3. 但新 Pod 10秒后因为内存泄漏崩溃

如果设置 `minReadySeconds: 30`，Pod 需要持续健康30秒才被认为可用。如果这30秒内崩溃，滚动更新会暂停，旧 Pod 不会被删除。

**这是一个缓冲期，防止有问题的新版本迅速替换所有旧版本**。

### revisionHistoryLimit：回滚的"时光机"

```yaml
spec:
  revisionHistoryLimit: 10
```

Deployment 会保留历史 ReplicaSet（副本数设为0）。这些历史版本用于回滚：

```
当前运行:
ReplicaSet-v3 (replicas: 10)  ← 当前版本

历史版本（保留用于回滚）:
ReplicaSet-v2 (replicas: 0)
ReplicaSet-v1 (replicas: 0)
```

回滚时，Deployment 只需要调整 ReplicaSet 的副本数：
```
ReplicaSet-v3: 10 → 0  （缩减当前版本）
ReplicaSet-v2: 0 → 10  （恢复历史版本）
```

这比重新创建 Pod 快得多，因为 ReplicaSet 的 Pod 模板已经存在。

`revisionHistoryLimit` 控制保留多少个历史版本。设置过大会占用 etcd 空间，过小则限制了回滚范围。

## Deployment 的生命周期管理

### 创建阶段

```
用户执行: kubectl apply -f deployment.yaml
    ↓
API Server 验证并存储到 etcd
    ↓
Deployment Controller 监听到新资源
    ↓
计算期望状态: 需要1个ReplicaSet，10个Pod
    ↓
创建 ReplicaSet 对象
    ↓
ReplicaSet Controller 监听到新资源
    ↓
创建10个Pod对象
    ↓
Scheduler 为Pod选择节点
    ↓
Kubelet 拉取镜像并启动容器
    ↓
ReadinessProbe 检查Pod就绪状态
    ↓
Deployment 状态更新: Available=10, Ready=10
```

关键点：
- **Deployment 不直接创建 Pod**，它只创建 ReplicaSet
- **ReplicaSet 也不直接启动容器**，它只创建 Pod 对象
- **Kubelet 才是真正启动容器的组件**

这种分层设计使得每个 Controller 职责单一，易于维护和扩展。

### 更新阶段

当你修改 Deployment 的 Pod 模板（比如更新镜像）时：

```
kubectl set image deployment/nginx nginx=nginx:1.22
    ↓
Deployment Controller 检测到 template 变化
    ↓
创建新的 ReplicaSet（带有新镜像）
    ↓
【滚动更新循环开始】
    ↓
增加新 ReplicaSet 副本数
    ↓
等待新 Pod 就绪（检查 ReadinessProbe）
    ↓
减少旧 ReplicaSet 副本数
    ↓
检查是否满足 maxSurge 和 maxUnavailable 约束
    ↓
【循环】直到新 ReplicaSet 副本数 = 期望副本数
    ↓
更新完成，旧 ReplicaSet 副本数降为0（但保留用于回滚）
```

**重要细节**：
- 只有 `spec.template` 的变化会触发滚动更新
- 修改 `spec.replicas` 不会创建新 ReplicaSet，只会调整当前 ReplicaSet
- 修改 `spec.selector` 在大多数情况下不允许（会被 API 拒绝）

### 回滚阶段

```
kubectl rollout undo deployment/nginx
    ↓
查找上一个 ReplicaSet（revisionHistoryLimit 范围内）
    ↓
【滚动更新循环】（只是方向相反）
    ↓
增加旧 ReplicaSet 副本数
减少新 ReplicaSet 副本数
    ↓
回滚完成
```

回滚本质上就是**一次特殊的滚动更新**，目标是旧版本的 ReplicaSet。

## Pod 模板的关键配置

### 资源请求与限制的真实含义

```yaml
resources:
  requests:    # 调度时的承诺
    cpu: "100m"
    memory: "128Mi"
  limits:      # 运行时的限制
    cpu: "200m"
    memory: "256Mi"
```

**requests（请求）**
- 调度器保证节点至少有这么多可用资源
- 如果所有节点都无法满足 requests，Pod 会一直 Pending
- CPU requests 影响 CFS（完全公平调度器）的权重

**limits（限制）**
- 容器使用资源的硬上限
- CPU limit 被 cgroups 的 CFS quota 强制执行（会被限流，不会被杀）
- Memory limit 被 OOM Killer 强制执行（超过会被杀死）

常见误区：
- ❌ "requests 是最小值，limits 是最大值" → requests 不是最小值，容器可以使用更少
- ❌ "不设置 limits 就没有限制" → 会受节点资源总量限制，且可能被优先驱逐
- ✅ "requests 用于调度决策，limits 用于运行时保护"

### 健康检查探针的工作原理

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 1
```

**livenessProbe（存活探针）**
- 检测容器是否还活着
- 失败后 Kubelet 会重启容器
- **典型场景**：应用死锁、进程假死

探针失败的处理流程：
```
探针失败1次 → 记录，继续检查
探针失败2次 → 记录，继续检查
探针失败3次 → 达到failureThreshold，杀死容器
    ↓
Kubelet 根据 restartPolicy 决定是否重启
    ↓
容器重启，重新计数
```

**readinessProbe（就绪探针）**
- 检测容器是否准备好接收流量
- 失败后从 Service 的 Endpoints 中移除
- **典型场景**：依赖的服务未就绪、正在加载缓存

就绪探针的影响：
```
ReadinessProbe 失败
    ↓
Pod 状态仍然是 Running
    ↓
但 Pod 的 Ready 条件变为 False
    ↓
Endpoints Controller 从 Service 后端列表移除该 Pod
    ↓
Service 不再将流量路由到该 Pod
    ↓
ReadinessProbe 恢复成功
    ↓
Pod 重新加入 Service 后端列表
```

**关键区别**：
- liveness 失败 → 重启容器（"它死了，需要重生"）
- readiness 失败 → 摘除流量（"它还活着，但需要休息"）

### 优雅终止的完整流程

```yaml
terminationGracePeriodSeconds: 30
```

当 Pod 被删除时（比如滚动更新删除旧 Pod），会触发优雅终止流程：

```
第1步: API Server 标记 Pod 为 Terminating
    ↓
【并行发生两件事】

分支A: Kubelet 侧
    ↓
执行 PreStop Hook（如果配置了）
    ↓
向容器主进程发送 SIGTERM 信号
    ↓
等待容器自行退出
    ↓
如果超过 terminationGracePeriodSeconds（默认30秒）
    ↓
发送 SIGKILL 强制杀死

分支B: Endpoints Controller 侧
    ↓
从 Service 的 Endpoints 中移除该 Pod
    ↓
新的请求不再路由到该 Pod
```

**关键问题**：两个分支是并行的，存在竞态条件！

可能发生的问题：
1. Kubelet 已经发送 SIGTERM 给容器
2. 但 Endpoints 的更新还没传播到所有 kube-proxy
3. 此时仍有新请求被路由到这个"正在关闭"的 Pod
4. 请求失败（容器已经停止监听端口）

**解决方案**：在 PreStop Hook 中加入延迟
```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 5"]
```

这5秒的延迟让 Endpoints 更新有时间传播，确保不再有新请求进来。

## 常见问题与误区

### Q1: Deployment 的 Pod 名称为什么有随机后缀？

ReplicaSet 会为每个 Pod 生成唯一的名称：`<replicaset-name>-<random-suffix>`

原因：
- Pod 名称必须唯一（在同一命名空间内）
- ReplicaSet 不关心具体是哪个 Pod，只关心总数
- 随机后缀避免了名称冲突

如果你需要稳定的 Pod 名称，应该使用 StatefulSet。

### Q2: 为什么更新后旧 Pod 没有立即删除？

可能的原因：
1. **新 Pod 的 ReadinessProbe 一直失败** → 滚动更新会等待新 Pod 就绪
2. **达到 maxUnavailable 限制** → 必须等旧 Pod 删除后新 Pod 就绪，才能继续
3. **PodDisruptionBudget 限制** → 可能配置了最小可用 Pod 数量
4. **资源不足** → 新 Pod 无法调度，滚动更新停滞

调试命令：
```bash
kubectl rollout status deployment/nginx  # 查看更新进度
kubectl describe deployment nginx         # 查看事件和条件
```

### Q3: 如何实现真正的"蓝绿部署"？

Deployment 的滚动更新是"渐进式"的，不是真正的蓝绿部署。

真正的蓝绿部署需要：
1. **两个独立的 Deployment**（蓝和绿）
2. **Service 通过修改 selector 切换流量**

```yaml
# 绿色环境正在运行
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  selector:
    app: myapp
    version: green  # 流量指向绿色

---
# 部署蓝色环境（新版本）
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp-blue
spec:
  replicas: 10
  template:
    metadata:
      labels:
        app: myapp
        version: blue

# 测试蓝色环境...
# 确认无误后，修改 Service selector 切换流量
```

这种方式可以瞬间切换流量，也可以瞬间回滚。

## 总结

Deployment 的核心机制：

1. **三层架构**：Deployment → ReplicaSet → Pod，每层有清晰的职责
2. **调谐循环**：持续对比期望状态和实际状态，自动消除差异
3. **滚动更新**：通过控制两个 ReplicaSet 的副本数，实现零停机更新
4. **健康检查**：liveness 检测存活，readiness 检测就绪，两者互补
5. **优雅终止**：并行进行容器关闭和流量摘除，注意竞态条件

理解这些原理后，你就能：
- 设计合理的更新策略
- 快速诊断部署问题
- 编写健壮的应用配置
- 在正确的场景使用正确的资源（Deployment vs StatefulSet vs DaemonSet）

Deployment 不只是一个 YAML 文件，它是 Kubernetes 声明式 API 设计哲学的完美体现。

## 参考资源

- [Kubernetes Deployment 官方文档](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
- [滚动更新策略详解](https://kubernetes.io/docs/tutorials/kubernetes-basics/update/update-intro/)
- [Pod 生命周期官方文档](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)

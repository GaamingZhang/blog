---
date: 2026-03-12
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Pod
  - 故障排查
---

# Kubernetes Pod 删除失败排查完全指南

## 引言

在 Kubernetes 集群的日常运维中，Pod 删除失败是一个常见且令人头疼的问题。当你执行 `kubectl delete pod` 命令后，Pod 却长时间卡在 Terminating 状态，无法正常清理。这种情况不仅影响集群资源的释放，还可能阻塞应用的更新和扩缩容操作。

理解 Pod 删除失败的根因，需要深入掌握 Kubernetes 的删除机制。Pod 删除并非简单的资源移除，而是一个涉及多个组件协作的复杂流程，包括 API Server、Controller Manager、Kubelet 以及各种 Finalizers 的协调配合。任何一个环节出现问题，都可能导致删除操作卡住。

本文将系统性地分析 Pod 删除失败的各类场景，从现象到根因，从排查到解决，帮助你建立完整的故障排查知识体系。

## Pod 删除流程解析

在深入具体问题之前，理解 Pod 的标准删除流程至关重要。这有助于我们定位问题发生在哪个环节。

### 正常删除流程

```
用户执行 kubectl delete pod
        ↓
API Server 接收请求，标记 Pod 为删除状态
        ↓
Pod 进入 Terminating 状态，设置 deletionTimestamp
        ↓
Endpoint Controller 从 Service Endpoints 中移除该 Pod
        ↓
执行 PreStop Hook（如果配置）
        ↓
发送 SIGTERM 信号给容器主进程
        ↓
等待宽限期（默认 30 秒）
        ↓
Kubelet 清理 Pod 资源（卸载存储卷、停止容器等）
        ↓
Finalizers 列表为空？ → 是 → 从 etcd 中删除 Pod 记录
        ↓ 否
等待 Finalizers 完成
        ↓
Pod 完全删除
```

### 关键机制说明

**Deletion Grace Period**：删除宽限期，默认 30 秒。在此期间，Pod 会优雅关闭，完成清理工作。可以通过 `--grace-period` 参数调整。

**Deletion Timestamp**：API Server 在接收到删除请求后，会为 Pod 设置 `metadata.deletionTimestamp` 字段，标记删除开始时间。

**Finalizers**：一种拦截机制，确保 Pod 删除前完成必要的清理工作。只有当 Finalizers 列表为空时，Pod 才会从 etcd 中彻底删除。

## 常见删除失败场景

### 场景一：卡在 Terminating 状态

#### 现象

执行 `kubectl delete pod <pod-name>` 后，Pod 长时间停留在 Terminating 状态，无法完成删除。

```bash
$ kubectl get pod nginx-pod -n default
NAME        READY   STATUS        RESTARTS   AGE
nginx-pod   1/1     Terminating   0          5m
```

#### 原因分析

Pod 卡在 Terminating 状态通常有以下几种原因：

1. **容器进程未响应 SIGTERM 信号**：容器内的主进程没有正确处理终止信号，导致容器无法正常退出。

2. **PreStop Hook 执行超时**：如果配置了 PreStop Hook，但执行时间过长或卡住，会阻塞删除流程。

3. **Kubelet 与 API Server 通信异常**：节点上的 Kubelet 无法正常上报状态或接收指令。

4. **节点资源耗尽**：节点 CPU、内存或磁盘资源不足，无法执行清理操作。

#### 排查步骤

**步骤 1：检查 Pod 详细信息**

```bash
kubectl describe pod <pod-name> -n <namespace>
```

关注 Events 部分，查看是否有错误信息。同时检查 Finalizers 字段是否为空。

**步骤 2：检查节点状态**

```bash
kubectl get nodes
kubectl describe node <node-name>
```

确认 Pod 所在节点是否处于 Ready 状态，是否存在资源压力。

**步骤 3：检查容器运行时**

登录到 Pod 所在节点，检查容器状态：

```bash
# 查看容器状态
crictl ps -a | grep <pod-id>

# 查看容器日志
crictl logs <container-id>

# 检查容器进程
ps aux | grep <container-process>
```

**步骤 4：检查 Kubelet 日志**

```bash
journalctl -u kubelet -n 100 --no-pager | grep <pod-name>
```

#### 解决方案

**方案 1：强制删除（谨慎使用）**

```bash
kubectl delete pod <pod-name> -n <namespace> --force --grace-period=0
```

强制删除会绕过优雅关闭流程，直接从 etcd 中删除记录。但要注意，这可能导致：
- 存储卷未正确卸载
- 网络资源未释放
- 外部资源泄漏

**方案 2：排查并修复节点问题**

如果节点状态异常，需要先恢复节点：

```bash
# 重启 Kubelet
systemctl restart kubelet

# 检查容器运行时
systemctl status containerd
```

**方案 3：手动清理容器**

在节点上手动停止容器：

```bash
# 停止容器
crictl stop <container-id>

# 删除容器
crictl rm <container-id>
```

### 场景二：Finalizers 未清理

#### 现象

Pod 的 Finalizers 列表不为空，导致无法删除。常见的 Finalizer 包括：
- `kubernetes.io/pvc-protection`
- `foregroundDeletion`
- 自定义 Controller 添加的 Finalizer

```bash
$ kubectl get pod nginx-pod -o yaml
apiVersion: v1
kind: Pod
metadata:
  finalizers:
  - kubernetes.io/pvc-protection
  name: nginx-pod
  namespace: default
```

#### 原因分析

Finalizers 是 Kubernetes 的资源清理保护机制。当 Pod 被删除时，API Server 会检查 Finalizers 列表，只有列表为空时才会真正删除资源。

**常见 Finalizer 及其作用**：

| Finalizer | 作用 | 负责组件 |
|-----------|------|----------|
| `kubernetes.io/pvc-protection` | 保护 PVC，确保 Pod 删除前 PVC 正确处理 | PVC Controller |
| `foregroundDeletion` | 前台删除，确保依赖资源先删除 | Garbage Collector |
| `orphan` | 孤儿删除策略 | Garbage Collector |
| 自定义 Finalizer | 业务逻辑清理 | 自定义 Controller |

**Finalizer 未清理的原因**：

1. **PVC 保护机制**：Pod 使用了 PVC，但 PVC 删除失败或卡住。

2. **Controller 故障**：负责清理 Finalizer 的 Controller 异常或未运行。

3. **自定义 Controller Bug**：自定义 Controller 的 Finalizer 逻辑存在缺陷。

4. **资源依赖未解除**：存在依赖资源未删除，导致 Finalizer 无法清理。

#### 排查步骤

**步骤 1：检查 Pod 的 Finalizers**

```bash
kubectl get pod <pod-name> -n <namespace> -o jsonpath='{.metadata.finalizers}'
```

**步骤 2：检查相关资源**

如果 Finalizer 是 `kubernetes.io/pvc-protection`，检查 PVC 状态：

```bash
kubectl get pvc -n <namespace>
kubectl describe pvc <pvc-name> -n <namespace>
```

**步骤 3：检查 Controller 状态**

```bash
# 检查相关 Controller 的 Pod 状态
kubectl get pods -n kube-system | grep -E 'controller|pvc'

# 查看 Controller 日志
kubectl logs -n kube-system <controller-pod>
```

#### 解决方案

**方案 1：修复依赖资源**

如果是 PVC 保护问题，先处理 PVC：

```bash
# 检查 PVC 是否被其他 Pod 使用
kubectl describe pvc <pvc-name> -n <namespace>

# 删除使用该 PVC 的其他 Pod
kubectl delete pod <other-pod> -n <namespace>

# 删除 PVC
kubectl delete pvc <pvc-name> -n <namespace>
```

**方案 2：手动移除 Finalizer（谨慎使用）**

```bash
kubectl patch pod <pod-name> -n <namespace> -p '{"metadata":{"finalizers":[]}}'
```

或使用 JSON Patch：

```bash
kubectl patch pod <pod-name> -n <namespace> --type='json' -p='[{"op": "replace", "path": "/metadata/finalizers", "value":[]}]'
```

**方案 3：重启相关 Controller**

```bash
# 重启 PVC Controller（通常是 kube-controller-manager）
kubectl rollout restart deployment/kube-controller-manager -n kube-system
```

### 场景三：孤儿 Pod

#### 现象

Pod 在 API Server 中已删除，但在节点上仍然运行。或者相反，节点上的 Pod 已删除，但 API Server 中仍有记录。

```bash
# API Server 中没有该 Pod
$ kubectl get pod nginx-pod -n default
Error from server (NotFound): pods "nginx-pod" not found

# 但节点上容器仍在运行
$ crictl ps | grep nginx
abc123def456   nginx:latest   Running   0   10m
```

#### 原因分析

孤儿 Pod（Orphaned Pod）通常由以下原因导致：

1. **节点故障恢复**：节点长时间不可达后恢复，但 Pod 记录已被 GC 清理。

2. **强制删除后未清理**：使用 `--force --grace-period=0` 强制删除，但节点上的容器未停止。

3. **Kubelet 数据损坏**：Kubelet 的本地状态数据与 API Server 不一致。

4. **etcd 数据不一致**：etcd 数据损坏或同步问题。

#### 排查步骤

**步骤 1：对比 API Server 和节点状态**

```bash
# 获取 API Server 中的 Pod 列表
kubectl get pods --all-namespaces -o wide

# 登录节点，检查实际运行的容器
crictl ps -a
crictl pods
```

**步骤 2：检查 Kubelet 本地状态**

```bash
# 查看 Kubelet 的 Pod 配置目录
ls /var/lib/kubelet/pods/

# 检查 Pod 的静态文件
cat /var/lib/kubelet/pods/<pod-uid>/plugins/kubernetes.io~empty-dir/ready
```

**步骤 3：检查 Kubelet 日志**

```bash
journalctl -u kubelet --since "1 hour ago" | grep -i orphan
```

#### 解决方案

**方案 1：手动清理孤儿容器**

在节点上手动删除孤儿容器：

```bash
# 停止容器
crictl stop <container-id>

# 删除容器
crictl rm <container-id>

# 删除 Pod 沙箱
crictl stopp <pod-sandbox-id>
crictl rmp <pod-sandbox-id>
```

**方案 2：清理 Kubelet 本地状态**

```bash
# 停止 Kubelet
systemctl stop kubelet

# 清理孤儿 Pod 数据
rm -rf /var/lib/kubelet/pods/<pod-uid>

# 重启 Kubelet
systemctl start kubelet
```

**方案 3：重建节点（极端情况）**

如果孤儿 Pod 数量过多或状态混乱，考虑排空节点并重新加入集群：

```bash
# 排空节点
kubectl drain <node-name> --delete-emptydir-data --ignore-daemonsets --force

# 重置节点
kubeadm reset

# 重新加入集群
kubeadm join ...
```

### 场景四：节点不可达

#### 现象

Pod 所在节点处于 NotReady 或 Unknown 状态，导致 Pod 无法正常删除。

```bash
$ kubectl get nodes
NAME      STATUS     ROLES    AGE   VERSION
node-1    Ready      master   10d   v1.28.0
node-2    NotReady   <none>   10d   v1.28.0

$ kubectl get pod nginx-pod -o wide
NAME        READY   STATUS        NODE
nginx-pod   1/1     Terminating   node-2
```

#### 原因分析

节点不可达导致 Pod 删除失败的机制：

1. **Kubelet 无法响应**：节点上的 Kubelet 进程停止或网络不通，无法接收删除指令。

2. **节点网络隔离**：节点与 Master 节点网络中断，API Server 无法与 Kubelet 通信。

3. **节点资源耗尽**：节点 CPU、内存或磁盘耗尽，导致 Kubelet 无法正常工作。

4. **节点宕机**：物理机或虚拟机故障，节点完全不可用。

#### 排查步骤

**步骤 1：检查节点状态**

```bash
kubectl describe node <node-name>
```

关注 Conditions 部分，特别是 Ready、MemoryPressure、DiskPressure 等状态。

**步骤 2：检查节点连通性**

```bash
# 从 Master 节点 ping 目标节点
ping <node-ip>

# 检查 SSH 连接
ssh <node-ip>

# 检查 Kubelet 端口
nc -zv <node-ip> 10250
```

**步骤 3：检查 Master 组件日志**

```bash
# 查看 Controller Manager 日志
kubectl logs -n kube-system kube-controller-manager-<master-node>

# 查看 API Server 日志
kubectl logs -n kube-system kube-apiserver-<master-node>
```

#### 解决方案

**方案 1：恢复节点（优先）**

尝试恢复节点的网络或服务：

```bash
# 如果能 SSH 到节点
ssh <node-ip>

# 检查 Kubelet 状态
systemctl status kubelet

# 重启 Kubelet
systemctl restart kubelet

# 检查容器运行时
systemctl status containerd
```

**方案 2：强制删除并重建**

如果节点无法恢复，可以强制删除 Pod：

```bash
# 强制删除 Pod
kubectl delete pod <pod-name> -n <namespace> --force --grace-period=0
```

**方案 3：删除节点并重新加入**

```bash
# 删除节点
kubectl delete node <node-name>

# 在节点上重置
kubeadm reset

# 重新加入集群
kubeadm join ...
```

**方案 4：调整 Pod Eviction 超时**

修改 Controller Manager 配置，调整 Pod 驱逐超时时间：

```yaml
# /etc/kubernetes/manifests/kube-controller-manager.yaml
spec:
  containers:
  - command:
    - kube-controller-manager
    - --pod-eviction-timeout=60s  # 默认 5m0s
    - --node-monitor-grace-period=40s  # 默认 40s
```

### 场景五：存储卷未卸载

#### 现象

Pod 删除时卡在 Terminating 状态，Events 中显示存储卷卸载失败。

```bash
$ kubectl describe pod nginx-pod
Events:
  Type     Reason         Age   From               Message
  ----     ------         ----  ----               -------
  Normal   Killing        2m    kubelet            Stopping container nginx
  Warning  FailedUnmount  2m    kubelet            Unable to detach volume "pvc-xxx" from node "node-1"
```

#### 原因分析

存储卷卸载失败的常见原因：

1. **存储后端故障**：NFS、Ceph、iSCSI 等存储后端不可用。

2. **存储驱动 Bug**：CSI 驱动或 FlexVolume 插件存在缺陷。

3. **挂载点被占用**：存储卷挂载点被进程占用，无法卸载。

4. **网络问题**：节点与存储系统之间的网络中断。

5. **PV/PVC 状态异常**：PV 或 PVC 处于异常状态。

#### 排查步骤

**步骤 1：检查 PV 和 PVC 状态**

```bash
kubectl get pv,pvc -n <namespace>
kubectl describe pv <pv-name>
kubectl describe pvc <pvc-name> -n <namespace>
```

**步骤 2：检查节点上的挂载点**

登录到 Pod 所在节点：

```bash
# 查看挂载点
mount | grep <pvc-name>

# 查看挂载点占用情况
lsof | grep <mount-path>

# 查看磁盘使用
df -h | grep <mount-path>
```

**步骤 3：检查存储系统状态**

根据存储类型检查：

```bash
# NFS
showmount -e <nfs-server>

# iSCSI
iscsiadm -m session

# Ceph
ceph status
```

**步骤 4：检查 CSI 驱动日志**

```bash
# 查找 CSI 驱动 Pod
kubectl get pods -n kube-system | grep csi

# 查看日志
kubectl logs -n kube-system <csi-driver-pod>
```

#### 解决方案

**方案 1：修复存储系统**

优先恢复存储后端服务：

```bash
# 重启 NFS 服务
systemctl restart nfs-server

# 检查 Ceph 集群状态
ceph health detail
```

**方案 2：手动卸载存储卷**

在节点上手动卸载：

```bash
# 查找挂载点
mount | grep <pvc-name>

# 强制卸载
umount -f <mount-path>

# 如果占用严重，使用懒卸载
umount -l <mount-path>
```

**方案 3：重启 CSI 驱动**

```bash
# 重启 CSI Controller
kubectl rollout restart deployment/csi-controller -n kube-system

# 重启 CSI Node（在每个节点上）
kubectl rollout restart daemonset/csi-node -n kube-system
```

**方案 4：清理 PV Finalizer**

如果 PV 也卡住，可以清理 Finalizer：

```bash
kubectl patch pv <pv-name> -p '{"metadata":{"finalizers":[]}}'
```

**方案 5：强制删除（最后手段）**

```bash
# 强制删除 Pod
kubectl delete pod <pod-name> -n <namespace> --force --grace-period=0

# 强制删除 PVC
kubectl delete pvc <pvc-name> -n <namespace> --force --grace-period=0
```

## 删除失败场景对比表

| 场景 | 典型现象 | 根本原因 | 排查重点 | 解决优先级 |
|------|----------|----------|----------|------------|
| 卡在 Terminating | 长时间 Terminating 状态 | 容器未响应、节点异常 | Events、节点状态 | 高 |
| Finalizers 未清理 | Finalizers 列表非空 | Controller 故障、依赖资源 | Finalizers 字段、PVC 状态 | 高 |
| 孤儿 Pod | API Server 与节点状态不一致 | 强制删除、数据损坏 | 对比 API Server 和节点 | 中 |
| 节点不可达 | 节点 NotReady/Unknown | 节点故障、网络隔离 | 节点连通性、Kubelet | 高 |
| 存储卷未卸载 | Events 显示卸载失败 | 存储故障、挂载占用 | PV/PVC、挂载点 | 高 |

## 排查流程图

```
Pod 删除失败
    ↓
检查 Pod 状态
    ├─ Terminating？ → 检查 Events
    ├─ Finalizers 非空？ → 检查依赖资源
    └─ 已删除但容器运行？ → 孤儿 Pod
    ↓
检查节点状态
    ├─ Ready？ → 检查 Kubelet 日志
    ├─ NotReady？ → 检查节点连通性
    └─ Unknown？ → 检查网络和 Master 组件
    ↓
检查存储状态
    ├─ PV/PVC 正常？ → 检查挂载点
    ├─ 存储后端可用？ → 检查 CSI 驱动
    └─ 挂载点占用？ → 手动卸载
    ↓
选择解决方案
    ├─ 修复依赖资源
    ├─ 移除 Finalizers
    ├─ 恢复节点
    └─ 强制删除（最后手段）
```

## 常见问题 FAQ

### Q1: 强制删除 Pod 有什么风险？

强制删除（`--force --grace-period=0`）会绕过优雅关闭流程，可能导致：
- 存储卷未正确卸载，数据丢失或损坏
- 网络资源（如 LoadBalancer IP）未释放
- 外部资源（如云磁盘、数据库连接）泄漏
- StatefulSet 的 Pod 标识混乱

建议仅在确认无其他方法时使用，并手动检查相关资源。

### Q2: 如何避免 Finalizers 导致的删除卡住？

1. **监控 Finalizers**：定期检查集群中 Finalizers 非空的资源。

```bash
kubectl get pods --all-namespaces -o json | jq '.items[] | select(.metadata.finalizers != null) | {name: .metadata.name, namespace: .metadata.namespace, finalizers: .metadata.finalizers}'
```

2. **确保 Controller 健康**：监控自定义 Controller 的运行状态。

3. **合理设置 Finalizer**：自定义 Controller 应实现健壮的 Finalizer 清理逻辑，包括超时和错误处理。

### Q3: 节点不可达时，如何快速恢复服务？

1. **调整 Pod 驱逐超时**：缩短 `pod-eviction-timeout`，加快故障转移。

2. **使用 PodDisruptionBudget**：确保最小可用副本数。

3. **配置健康检查**：设置合理的 livenessProbe 和 readinessProbe。

4. **使用多副本部署**：避免单点故障。

### Q4: 存储卷卸载失败如何预防？

1. **选择可靠的存储方案**：使用成熟的 CSI 驱动。

2. **监控存储系统**：及时发现存储后端故障。

3. **合理配置存储参数**：如 NFS 的 `soft` 和 `timeo` 参数。

4. **定期检查挂载点**：清理僵尸挂载。

### Q5: 如何批量处理卡住的 Pod？

编写脚本批量处理：

```bash
#!/bin/bash

# 查找所有 Terminating 状态超过 5 分钟的 Pod
terminating_pods=$(kubectl get pods --all-namespaces --field-selector=status.phase=Failed -o json | \
  jq -r '.items[] | select(.metadata.deletionTimestamp != null) | "\(.metadata.namespace) \(.metadata.name)"')

# 强制删除
while read namespace pod; do
  echo "Force deleting pod $pod in namespace $namespace"
  kubectl delete pod $pod -n $namespace --force --grace-period=0
done <<< "$terminating_pods"
```

## 最佳实践

### 1. 优雅删除配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
spec:
  terminationGracePeriodSeconds: 60  # 设置合理的宽限期
  containers:
  - name: nginx
    image: nginx:latest
    lifecycle:
      preStop:
        exec:
          command: ["/bin/sh", "-c", "nginx -s quit && sleep 10"]  # 优雅关闭
```

### 2. 监控和告警

建立监控体系，及时发现删除异常：

```yaml
# Prometheus 告警规则示例
groups:
- name: pod-deletion
  rules:
  - alert: PodStuckInTerminating
    expr: kube_pod_status_phase{phase="Terminating"} > 300
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Pod {{ $labels.namespace }}/{{ $labels.pod }} stuck in Terminating"
      description: "Pod has been in Terminating state for more than 5 minutes"
```

### 3. 定期清理脚本

```bash
#!/bin/bash

# 清理孤儿 Pod
for pod in $(crictl pods -q); do
  pod_uid=$(crictl inspectp $pod | jq -r '.status.labels["io.kubernetes.pod.uid"]')
  if ! kubectl get pod -A -o jsonpath='{.items[*].metadata.uid}' | grep -q "$pod_uid"; then
    echo "Removing orphan pod: $pod"
    crictl rmp -f $pod
  fi
done
```

### 4. 节点维护流程

```bash
# 标准的节点维护流程
# 1. 标记节点不可调度
kubectl cordon <node-name>

# 2. 排空节点
kubectl drain <node-name> --delete-emptydir-data --ignore-daemonsets --grace-period=60 --timeout=300s

# 3. 执行维护操作
# ...

# 4. 恢复节点
kubectl uncordon <node-name>
```

### 5. 资源清理检查清单

在强制删除前，检查以下资源：

- [ ] PVC 是否正常卸载
- [ ] Service 是否已移除 Endpoints
- [ ] ConfigMap/Secret 是否被其他资源引用
- [ ] 网络策略是否清理
- [ ] 外部资源（云磁盘、负载均衡器）是否释放

## 面试回答

在面试中回答"Kubernetes 中 Pod 删除失败有哪些情况及如何解决"时，可以这样组织：

Pod 删除失败是 Kubernetes 运维中的常见问题，主要分为五种典型场景。第一种是卡在 Terminating 状态，通常由于容器进程未响应 SIGTERM 信号、PreStop Hook 执行超时或 Kubelet 异常导致，排查时需要检查 Pod Events、节点状态和 Kubelet 日志，可通过强制删除或修复节点解决。第二种是 Finalizers 未清理，Finalizers 是资源清理保护机制，只有列表为空时 Pod 才会真正删除，常见原因包括 PVC 保护、Controller 故障，需要先处理依赖资源或手动移除 Finalizer。第三种是孤儿 Pod，即 API Server 与节点状态不一致，多由强制删除或节点故障恢复导致，需要手动清理节点上的孤儿容器。第四种是节点不可达，节点 NotReady 或 Unknown 导致 Kubelet 无法响应删除指令，优先尝试恢复节点，无法恢复时才使用强制删除。第五种是存储卷未卸载，存储后端故障或挂载点占用导致，需要检查 PV/PVC 状态、存储系统健康度，必要时手动卸载。解决这类问题的核心思路是：先通过 describe 和日志定位问题环节，优先修复依赖资源和节点状态，强制删除作为最后手段并需评估风险。同时建议建立监控告警机制，及时发现 Terminating 状态异常，并制定标准的节点维护流程，避免操作不当引发删除问题。

---

**参考资料**：
- [Kubernetes 官方文档 - Pod 生命周期](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)
- [Kubernetes 官方文档 - Finalizers](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/)
- [Kubernetes 官方文档 - 节点维护](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)

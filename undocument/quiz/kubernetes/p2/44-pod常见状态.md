---
date: 2026-03-12
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Pod
  - 状态管理
---

# Kubernetes Pod 常见状态详解

## 引言

在 Kubernetes 集群的日常运维中，Pod 状态是判断应用运行健康程度的最直接指标。当执行 `kubectl get pods` 命令时，我们看到的 STATUS 列显示了 Pod 当前的运行状态。理解这些状态的含义、产生原因以及排查方法，是每个 Kubernetes 使用者必须掌握的核心技能。

一个 Pod 从创建到运行，再到终止，会经历多个状态阶段。有些状态表示正常运行，有些状态则预示着问题需要排查。本文将深入剖析 Kubernetes Pod 的各种常见状态，帮助您快速定位和解决问题。

## Pod 生命周期阶段

在深入了解具体状态之前，首先需要理解 Pod 的生命周期阶段（Phase）。Kubernetes 将 Pod 的生命周期划分为五个主要阶段：

| 阶段 | 说明 |
|-----|------|
| **Pending** | Pod 已被 Kubernetes 系统接受，但容器镜像尚未创建 |
| **Running** | Pod 已经绑定到某个节点，所有容器都已创建，至少有一个容器正在运行 |
| **Succeeded** | Pod 中的所有容器都已成功终止，并且不会重启 |
| **Failed** | Pod 中的所有容器都已终止，并且至少有一个容器以失败状态终止 |
| **Unknown** | 由于某些原因无法获取 Pod 的状态，通常是与节点通信失败 |

## 一、Pending 状态

### 状态含义

Pending 状态表示 Pod 已被 Kubernetes API Server 接受，但尚未被调度到节点上运行，或者正在下载容器镜像。这是 Pod 创建后的初始状态。

### 常见原因

1. **资源不足**
   - 节点 CPU、内存资源不足以满足 Pod 的 requests
   - 没有可用节点满足 Pod 的资源需求

2. **调度限制**
   - nodeSelector 或 nodeAffinity 不匹配任何节点
   - 节点存在污点（Taint），Pod 没有对应的容忍（Toleration）
   - Pod Anti-Affinity 规则阻止调度

3. **存储问题**
   - PVC 未绑定到 PV
   - StorageClass 不存在或配置错误
   - 动态供给失败

4. **镜像拉取问题**
   - 镜像不存在或镜像名称错误
   - 镜像仓库访问权限问题
   - 网络问题导致镜像拉取超时

### 排查方法

```bash
# 查看 Pod 详细信息
kubectl describe pod <pod-name> -n <namespace>

# 查看事件
kubectl get events -n <namespace> --sort-by='.lastTimestamp'

# 查看节点资源
kubectl describe nodes | grep -A 5 "Allocated resources"

# 查看调度器日志
kubectl logs -n kube-system <scheduler-pod-name>
```

### 典型事件示例

```
Events:
  Type     Reason            Age   From               Message
  ----     ------            ----  ----               -------
  Warning  FailedScheduling  10s   default-scheduler  0/3 nodes are available: 3 Insufficient cpu.
```

### 解决方案

```yaml
# 检查资源请求是否合理
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"

# 检查节点选择器
nodeSelector:
  disktype: ssd

# 检查容忍配置
tolerations:
- key: "node-role.kubernetes.io/master"
  operator: "Exists"
  effect: "NoSchedule"
```

## 二、Running 状态

### 状态含义

Running 状态表示 Pod 已经被调度到节点上，并且所有容器都已创建，至少有一个容器正在运行或正在启动。这是 Pod 正常运行时的状态。

### 注意事项

Running 状态并不意味着应用完全正常：
- 容器可能正在启动过程中
- 应用可能存在内部错误但进程仍在运行
- 健康检查可能尚未通过

### 状态确认

```bash
# 查看 Pod 详细状态
kubectl get pod <pod-name> -o wide

# 查看容器状态
kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[*].state}'

# 查看就绪状态
kubectl get pod <pod-name> -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
```

### Running 但不正常的情况

1. **容器启动中**
   ```
   State:          Waiting
   Reason:       ContainerCreating
   ```

2. **应用内部错误**
   - 进程运行但返回错误
   - 需要查看应用日志排查

3. **健康检查未通过**
   - 就绪探针失败，Pod 未进入 Service 端点
   - 存活探针失败，容器会被重启

## 三、Succeeded 状态

### 状态含义

Succeeded 状态表示 Pod 中的所有容器都已成功终止（退出码为 0），并且不会重启。这种状态通常出现在一次性任务（Job）或批处理任务中。

### 适用场景

- 数据迁移任务
- 批处理作业
- 定时任务（CronJob）
- 初始化任务

### 配置示例

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: data-migration
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: migration
        image: migration-tool:v1.0
        command: ["python", "migrate.py"]
```

### 查看完成的 Pod

```bash
# 查看已完成的 Pod
kubectl get pods --field-selector=status.phase=Succeeded

# 查看 Job 状态
kubectl get jobs

# 查看完成 Pod 的日志
kubectl logs <pod-name>
```

## 四、Failed 状态

### 状态含义

Failed 状态表示 Pod 中的所有容器都已终止，并且至少有一个容器以失败状态终止（非零退出码）。这通常意味着任务执行失败或容器运行出错。

### 常见原因

1. **应用错误**
   - 代码异常导致进程退出
   - 配置错误
   - 依赖服务不可用

2. **资源耗尽**
   - OOM Killed（内存不足）
   - CPU 限制被触发

3. **健康检查失败**
   - 存活探针持续失败
   - 启动超时

4. **外部因素**
   - 节点故障
   - 网络问题
   - 存储问题

### 排查方法

```bash
# 查看 Pod 详细信息
kubectl describe pod <pod-name>

# 查看容器退出码
kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[0].state.terminated.exitCode}'

# 查看终止原因
kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[0].state.terminated.reason}'

# 查看日志
kubectl logs <pod-name> --previous
```

### 常见退出码

| 退出码 | 含义 | 常见原因 |
|-------|------|---------|
| 0 | 正常退出 | 任务成功完成 |
| 1 | 应用错误 | 应用程序错误 |
| 126 | 命令无法执行 | 权限问题 |
| 127 | 命令未找到 | 命令或脚本不存在 |
| 128+N | 被信号终止 | 如 137 = SIGKILL |
| 137 | OOM Killed | 内存不足被杀死 |
| 139 | Segmentation Fault | 程序崩溃 |
| 143 | SIGTERM | 正常终止信号 |

## 五、Unknown 状态

### 状态含义

Unknown 状态表示 Kubernetes 无法获取 Pod 的当前状态，通常是由于与所在节点的 kubelet 通信失败导致的。

### 常见原因

1. **节点故障**
   - 节点宕机
   - 网络中断
   - kubelet 进程崩溃

2. **通信问题**
   - API Server 与 kubelet 通信失败
   - 网络分区

3. **资源压力**
   - 节点资源耗尽
   - kubelet 响应超时

### 排查方法

```bash
# 检查节点状态
kubectl get nodes

# 查看节点详情
kubectl describe node <node-name>

# 检查 kubelet 状态（在节点上执行）
systemctl status kubelet

# 查看 kubelet 日志
journalctl -u kubelet -f
```

### 处理方案

```bash
# 如果节点确实故障，可以删除 Pod 让其在其他节点重建
kubectl delete pod <pod-name> --force --grace-period=0

# 如果节点恢复，状态会自动更新
# 可以等待节点恢复或手动处理
```

## 六、CrashLoopBackOff 状态

### 状态含义

CrashLoopBackOff 是 Kubernetes 中最常见的错误状态之一，表示容器启动后立即崩溃，Kubelet 正在按照指数退避策略进行重启。

### 工作原理

当容器反复启动失败时，Kubernetes 会采用指数退避机制：
- 第 1 次重启：立即
- 第 2 次重启：等待 10 秒
- 第 3 次重启：等待 20 秒
- 第 4 次重启：等待 40 秒
- ...
- 最大等待时间：5 分钟

### 常见原因

1. **应用启动失败**
   - 配置错误
   - 依赖服务不可用
   - 代码异常

2. **资源问题**
   - 内存不足（OOM）
   - CPU 限制过低

3. **健康检查配置错误**
   - 存活探针检查路径错误
   - 初始延迟时间过短
   - 检查端口错误

4. **权限问题**
   - 文件权限不足
   - 用户权限问题

### 排查方法

```bash
# 查看容器日志
kubectl logs <pod-name>
kubectl logs <pod-name> --previous  # 查看上一个容器的日志

# 查看详细信息
kubectl describe pod <pod-name>

# 查看重启次数
kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[0].restartCount}'

# 进入容器调试（如果容器能短暂运行）
kubectl exec -it <pod-name> -- /bin/sh
```

### 解决方案示例

```yaml
# 调整健康检查配置
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 60  # 增加初始延迟
  periodSeconds: 10
  failureThreshold: 3

# 增加资源限制
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "1000m"

# 使用 init 容器检查依赖
initContainers:
- name: check-db
  image: busybox
  command: ['sh', '-c', 'until nc -z mysql 3306; do sleep 1; done']
```

## 七、ImagePullBackOff 状态

### 状态含义

ImagePullBackOff 表示 Kubernetes 无法拉取容器镜像，已经放弃重试。这通常发生在镜像名称错误、镜像不存在或认证失败的情况下。

### 常见原因

1. **镜像问题**
   - 镜像名称错误
   - 镜像标签不存在
   - 私有镜像仓库认证失败

2. **网络问题**
   - 镜像仓库不可访问
   - 网络超时

3. **权限问题**
   - imagePullSecrets 未配置或错误
   - 镜像仓库权限不足

### 排查方法

```bash
# 查看详细错误信息
kubectl describe pod <pod-name>

# 检查镜像是否存在
docker pull <image-name>

# 检查 Secret 配置
kubectl get secrets
kubectl describe secret <secret-name>
```

### 解决方案

```yaml
# 配置镜像拉取密钥
apiVersion: v1
kind: Secret
metadata:
  name: registry-secret
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: <base64-encoded-docker-config>
---
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  imagePullSecrets:
  - name: registry-secret
  containers:
  - name: app
    image: private-registry/my-app:v1.0
    imagePullPolicy: Always
```

## 八、ErrImagePull 状态

### 状态含义

ErrImagePull 与 ImagePullBackOff 类似，表示镜像拉取失败，但尚未进入退避重试阶段。

### 常见原因

与 ImagePullBackOff 基本相同，通常在多次重试后会变成 ImagePullBackOff。

### 处理方法

参考 ImagePullBackOff 的排查和解决方案。

## 九、ContainerCreating 状态

### 状态含义

ContainerCreating 表示 Pod 正在创建容器，这是一个过渡状态，通常持续时间很短。

### 长时间处于此状态的原因

1. **镜像拉取慢**
   - 镜像较大
   - 网络带宽不足
   - 镜像仓库响应慢

2. **存储挂载问题**
   - PV/PVC 绑定失败
   - 存储驱动问题
   - NFS 挂载超时

3. **CNI 网络配置**
   - 网络插件未就绪
   - IP 地址分配失败

### 排查方法

```bash
# 查看详细事件
kubectl describe pod <pod-name>

# 检查镜像大小
docker images | grep <image-name>

# 检查存储状态
kubectl get pv,pvc

# 检查网络插件
kubectl get pods -n kube-system | grep -E 'calico|flannel|weave'
```

## 十、Terminating 状态

### 状态含义

Terminating 表示 Pod 正在被删除，正在执行优雅终止过程。正常情况下，Pod 会在 terminationGracePeriodSeconds（默认 30 秒）内完成终止。

### 长时间处于 Terminating 的原因

1. **应用未处理 SIGTERM**
   - 应用未实现优雅关闭
   - 长连接未断开

2. **finalizer 未完成**
   - 资源清理阻塞
   - 控制器异常

3. **节点故障**
   - 节点不可达
   - kubelet 无法响应

### 强制删除 Pod

```bash
# 正常删除（等待优雅终止）
kubectl delete pod <pod-name>

# 强制删除（跳过优雅终止）
kubectl delete pod <pod-name> --force --grace-period=0

# 删除卡在 Terminating 的 Pod
kubectl delete pod <pod-name> --force --grace-period=0 --wait=false
```

## 十一、Evicted 状态

### 状态含义

Evicted 表示 Pod 被驱逐，通常是因为节点资源不足（磁盘、内存等）导致 kubelet 主动驱逐 Pod。

### 驱逐原因

1. **磁盘压力**
   - nodefs 可用空间不足
   - imagefs 可用空间不足

2. **内存压力**
   - 节点内存不足
   - OOM 风险

3. **PID 资源**
   - 进程数过多

### 排查方法

```bash
# 查看 Pod 驱逐原因
kubectl describe pod <pod-name> | grep -A 5 "Events:"

# 查看节点资源状态
kubectl describe node <node-name>

# 查看节点压力条件
kubectl get node <node-name> -o jsonpath='{.status.conditions}'
```

### 处理方案

```bash
# 清理已驱逐的 Pod
kubectl get pods --field-selector=status.phase=Failed -n <namespace> -o json | kubectl delete -f -

# 设置 Pod 反驱逐优先级
priorityClassName: high-priority
```

## 十二、Completed 状态

### 状态含义

Completed 状态通常用于 Job 类型的 Pod，表示任务已成功完成。这与 Succeeded 状态类似，但更多用于 Job 控制器管理的 Pod。

### 查看方法

```bash
# 查看已完成的 Job
kubectl get jobs

# 查看 Job 创建的 Pod
kubectl get pods -l job-name=<job-name>

# 查看完成日志
kubectl logs <pod-name>
```

## 状态对比总结

| 状态 | 含义 | 是否正常 | 典型场景 |
|-----|------|---------|---------|
| Pending | 等待调度 | 过渡状态 | Pod 创建初期 |
| Running | 运行中 | 正常 | 服务正常运行 |
| Succeeded | 成功完成 | 正常 | Job 任务完成 |
| Failed | 失败终止 | 异常 | 任务失败 |
| Unknown | 状态未知 | 异常 | 节点通信失败 |
| CrashLoopBackOff | 崩溃循环 | 异常 | 应用启动失败 |
| ImagePullBackOff | 镜像拉取失败 | 异常 | 镜像问题 |
| ContainerCreating | 容器创建中 | 过渡状态 | 启动过程 |
| Terminating | 终止中 | 过渡状态 | 删除过程 |
| Evicted | 被驱逐 | 异常 | 资源不足 |

## 最佳实践

### 1. 合理配置资源

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### 2. 配置健康检查

```yaml
livenessProbe:
  httpGet:
    path: /health
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
```

### 3. 设置合理的终止时间

```yaml
spec:
  terminationGracePeriodSeconds: 60
  containers:
  - name: app
    lifecycle:
      preStop:
        exec:
          command: ["/bin/sh", "-c", "sleep 15"]
```

### 4. 配置镜像拉取策略

```yaml
imagePullPolicy: IfNotPresent
imagePullSecrets:
- name: registry-secret
```

### 5. 设置 Pod 优先级

```yaml
priorityClassName: high-priority
```

### 6. 监控 Pod 状态

```yaml
# Prometheus 告警规则
- alert: PodCrashLooping
  expr: rate(kube_pod_container_status_restarts_total[15m]) * 60 * 15 > 0
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Pod {{ $labels.pod }} is crash looping"
```

## 面试回答

**问题**: Kubernetes 中 Pod 常见的状态有哪些？

**回答**: Kubernetes Pod 的状态可以分为几类。首先是生命周期阶段状态：**Pending** 表示 Pod 已被接受但尚未调度，可能是资源不足或调度限制导致；**Running** 表示 Pod 正在运行，至少有一个容器在运行；**Succeeded** 表示所有容器成功终止，常见于 Job 任务；**Failed** 表示容器以失败状态终止；**Unknown** 表示无法获取状态，通常是节点通信问题。

其次是异常状态：**CrashLoopBackOff** 是最常见的错误状态，表示容器反复崩溃，Kubelet 采用指数退避策略重启，常见原因是应用启动失败、资源不足或健康检查配置错误；**ImagePullBackOff** 表示镜像拉取失败，可能是镜像不存在、认证失败或网络问题；**Evicted** 表示 Pod 被驱逐，通常是因为节点磁盘或内存压力；**Terminating** 表示正在删除，如果卡住可能是应用未处理终止信号或节点故障。

排查 Pod 状态问题时，通常使用 `kubectl describe pod` 查看事件，使用 `kubectl logs` 查看日志，检查资源配置、健康检查配置和依赖服务状态。理解这些状态的含义和产生原因，是快速定位和解决 Kubernetes 问题的关键。

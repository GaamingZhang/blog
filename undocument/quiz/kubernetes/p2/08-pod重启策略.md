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
  - 重启策略
---

# Kubernetes Pod 重启策略详解

## 引言

在 Kubernetes 集群中,Pod 是最小的可部署单元,承载着应用程序的实际运行。然而,容器化应用在运行过程中难免会遇到各种异常情况,如进程崩溃、资源耗尽、健康检查失败等。当这些异常发生时,Pod 应该如何处理?是立即重启、延迟重启,还是保持失败状态?这正是 Pod 重启策略需要解决的问题。

Pod 重启策略定义了当容器退出时,Kubernetes 应该采取何种行为。合理配置重启策略不仅能够提高应用的可用性,还能避免不必要的资源浪费和无限重启循环。理解 Pod 重启策略的工作原理和适用场景,是每个 Kubernetes 使用者的必备技能。

## Pod 重启策略概述

Kubernetes 提供了三种 Pod 重启策略,通过 `spec.restartPolicy` 字段进行配置:

- **Always**: 无论容器以何种状态退出,都始终重启容器
- **OnFailure**: 仅当容器异常退出(退出码非 0)时才重启容器
- **Never**: 无论容器以何种状态退出,都不重启容器

需要注意的是,Pod 的重启策略作用于 Pod 内的所有容器。如果 Pod 中包含多个容器,重启策略将统一应用于所有容器。

## 一、Always 策略

### 工作原理

Always 策略是 Kubernetes 中最常用的重启策略,也是 Deployment、DaemonSet、StatefulSet 等控制器的默认策略。当配置为 Always 时,Kubelet 会监控 Pod 中所有容器的状态,一旦容器退出(无论退出码是 0 还是非 0),Kubelet 就会立即尝试重启该容器。

重启过程并非简单的立即重启,而是遵循指数退避机制:

1. **第一次重启**: 容器退出后立即重启
2. **第二次重启**: 等待 10 秒后重启
3. **第三次重启**: 等待 20 秒后重启
4. **第四次重启**: 等待 40 秒后重启
5. **第五次重启**: 等待 80 秒后重启
6. **后续重启**: 等待时间持续加倍,最大不超过 5 分钟

这种退避机制可以有效防止频繁重启导致的系统资源浪费,同时给予系统一定的恢复时间。

### 使用场景

Always 策略适用于需要长期运行的服务,包括:

- **Web 服务**: 如 Nginx、Apache、Tomcat 等需要持续提供服务的应用
- **API 服务**: RESTful API、GraphQL API 等后端服务
- **微服务**: Spring Boot、Go Micro、gRPC 等微服务应用
- **消息消费者**: Kafka Consumer、RabbitMQ Consumer 等需要持续监听消息队列的应用
- **后台任务**: 需要持续运行的守护进程

这些应用的特点是需要保持持续可用性,即使发生异常也应该自动恢复,而不是等待人工干预。

### 配置示例

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: web-server
  labels:
    app: nginx
spec:
  restartPolicy: Always  # 显式指定 Always 策略
  containers:
  - name: nginx
    image: nginx:1.21
    ports:
    - containerPort: 80
    resources:
      requests:
        memory: "128Mi"
        cpu: "100m"
      limits:
        memory: "256Mi"
        cpu: "200m"
    livenessProbe:
      httpGet:
        path: /
        port: 80
      initialDelaySeconds: 10
      periodSeconds: 10
```

在大多数情况下,使用 Deployment 等控制器时无需显式指定 `restartPolicy`,因为默认值就是 Always:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      # restartPolicy 默认为 Always,无需显式指定
      containers:
      - name: api
        image: my-api:v1.0
        ports:
        - containerPort: 8080
```

## 二、OnFailure 策略

### 工作原理

OnFailure 策略是一种条件性重启策略,只有当容器以非零退出码退出时才会触发重启。这意味着:

- 如果容器正常退出(退出码为 0),Kubelet 不会重启容器
- 如果容器异常退出(退出码非 0),Kubelet 会按照指数退避机制重启容器

这种策略特别适合批处理任务和定时任务,这些任务在正常完成后应该停止运行,而在失败时应该重试。

### 使用场景

OnFailure 策略适用于以下场景:

- **批处理任务**: 数据处理、文件转换、报表生成等一次性任务
- **定时任务**: 数据备份、日志清理、缓存刷新等周期性任务
- **数据迁移**: 数据库迁移、数据同步等需要重试的任务
- **机器学习训练**: 模型训练任务,失败后需要重试
- **CI/CD 任务**: 构建任务、测试任务等

这些任务的共同特点是:正常完成时应该退出,失败时需要重试。如果使用 Always 策略,即使任务正常完成也会被重启,导致无限循环。

### 配置示例

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: data-processing
spec:
  completions: 1           # 需要成功完成的 Pod 数量
  parallelism: 1           # 并行运行的 Pod 数量
  backoffLimit: 3          # 失败重试次数限制
  template:
    spec:
      restartPolicy: OnFailure  # 仅在失败时重启
      containers:
      - name: processor
        image: data-processor:v1.0
        command: ["python", "process.py"]
        args: ["--input", "/data/input.csv", "--output", "/data/output.csv"]
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
```

CronJob 示例:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: daily-backup
spec:
  schedule: "0 2 * * *"    # 每天凌晨 2 点执行
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
          - name: backup
            image: backup-tool:v1.0
            command: ["/bin/sh", "-c"]
            args:
            - |
              mysqldump -h mysql-server -u root -p$MYSQL_PASSWORD \
                --all-databases > /backup/dump-$(date +%Y%m%d).sql
            env:
            - name: MYSQL_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: mysql-secret
                  key: password
```

## 三、Never 策略

### 工作原理

Never 策略是最严格的重启策略,无论容器以何种状态退出,Kubelet 都不会重启容器。容器退出后将保持退出状态,直到被外部控制器(如 Job 控制器)删除或重新创建。

这种策略通常用于以下情况:
- 需要手动检查容器退出原因
- 任务失败后需要人工干预
- 配合其他控制器实现自定义重试逻辑

### 使用场景

Never 策略适用于以下场景:

- **调试和排错**: 需要保留容器退出状态以便分析问题
- **一次性任务**: 确保任务只执行一次,不重试
- **手动控制**: 需要人工判断是否重新执行的场景
- **自定义重试逻辑**: 配合外部控制器实现复杂的重试策略
- **测试任务**: 测试失败后需要保留现场

### 配置示例

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: manual-task
spec:
  completions: 1
  parallelism: 1
  backoffLimit: 0          # 不自动重试
  template:
    spec:
      restartPolicy: Never  # 从不重启
      containers:
      - name: task
        image: task-runner:v1.0
        command: ["./run-task.sh"]
```

调试场景示例:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: debug-pod
spec:
  restartPolicy: Never  # 不重启,保留现场
  containers:
  - name: app
    image: my-app:v1.0
    command: ["/bin/sh", "-c", "sleep 3600"]
  # 容器退出后可以执行 kubectl logs 或 kubectl describe 查看状态
```

## 重启策略与控制器的关系

Pod 重启策略与 Kubernetes 控制器之间存在密切的关系,不同的控制器对重启策略有不同的限制和要求:

### 控制器对重启策略的限制

| 控制器类型 | 允许的重启策略 | 默认策略 | 说明 |
|-----------|--------------|---------|------|
| Deployment | Always | Always | Deployment 只支持 Always 策略 |
| DaemonSet | Always, OnFailure | Always | 通常使用 Always,特殊场景可用 OnFailure |
| StatefulSet | Always | Always | 有状态应用通常需要 Always 策略 |
| Job | OnFailure, Never | OnFailure | Job 不支持 Always 策略 |
| CronJob | OnFailure, Never | OnFailure | CronJob 创建 Job,遵循 Job 的限制 |
| ReplicaSet | Always | Always | ReplicaSet 只支持 Always 策略 |

### 控制器行为差异

**Deployment/ReplicaSet/DaemonSet**:
- 这些控制器期望 Pod 持续运行
- 如果 Pod 失败,控制器会创建新的 Pod 替代
- 重启策略必须是 Always,否则控制器无法正常工作

**Job/CronJob**:
- 这些控制器管理一次性任务
- Pod 重启策略决定任务失败时的行为
- OnFailure: 在同一 Pod 内重启容器
- Never: 创建新的 Pod 重试任务

**StatefulSet**:
- 有状态应用通常需要稳定的网络标识和存储
- 重启策略为 Always,确保服务持续可用
- 即使 Pod 失败,也会在原位置重建

### 配置示例对比

Deployment(只支持 Always):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
spec:
  replicas: 3
  template:
    spec:
      restartPolicy: Always  # 必须是 Always
      containers:
      - name: web
        image: nginx:latest
```

Job(支持 OnFailure 或 Never):

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: batch-job
spec:
  template:
    spec:
      restartPolicy: OnFailure  # 可以是 OnFailure 或 Never
      containers:
      - name: batch
        image: batch-processor:latest
```

## 重启退避机制详解

### CrashLoopBackOff 状态

当 Pod 配置了 Always 或 OnFailure 重启策略,但容器持续启动失败时,Pod 会进入 CrashLoopBackOff 状态。这是 Kubernetes 的一种保护机制,防止无限重启导致资源浪费。

CrashLoopBackOff 状态的典型特征:
- Pod 状态显示为 `CrashLoopBackOff`
- 容器反复启动和退出
- 重启间隔遵循指数退避算法
- 最终稳定在 5 分钟的重启间隔

### 退避算法原理

Kubelet 使用指数退避算法计算重启延迟:

```
delay = min(backoff * 2^(restartCount-1), maxBackoff)
```

其中:
- `backoff`: 初始退避时间,默认 10 秒
- `restartCount`: 重启次数
- `maxBackoff`: 最大退避时间,默认 5 分钟(300 秒)

具体时间序列:
- 第 1 次重启: 0 秒(立即重启)
- 第 2 次重启: 10 秒
- 第 3 次重启: 20 秒
- 第 4 次重启: 40 秒
- 第 5 次重启: 80 秒
- 第 6 次重启: 160 秒
- 第 7 次重启: 300 秒(达到上限)
- 第 8 次及以后: 300 秒

### 查看 Pod 重启次数

可以通过以下方式查看 Pod 的重启次数和状态:

```bash
# 查看 Pod 状态和重启次数
kubectl get pod <pod-name> -o wide

# 查看详细的重启信息
kubectl describe pod <pod-name>

# 查看容器状态
kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[0].restartCount}'
```

### 排查 CrashLoopBackOff

当 Pod 进入 CrashLoopBackOff 状态时,应该按以下步骤排查:

1. **查看容器日志**:
```bash
kubectl logs <pod-name> --previous  # 查看上一个容器的日志
```

2. **查看 Pod 事件**:
```bash
kubectl describe pod <pod-name>
```

3. **检查容器退出码**:
```bash
kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[0].lastState.terminated.exitCode}'
```

常见退出码含义:
- **0**: 正常退出
- **1**: 应用错误
- **126**: 命令无法执行
- **127**: 命令未找到
- **128+N**: 被信号 N 终止(如 137 = 128 + 9,被 SIGKILL 终止)
- **137**: OOM Killed(内存不足)
- **139**: Segmentation Fault
- **143**: 被 SIGTERM 终止

4. **检查资源限制**:
```bash
kubectl describe pod <pod-name> | grep -A 5 "Limits:"
```

5. **检查健康检查配置**:
```bash
kubectl get pod <pod-name> -o yaml | grep -A 10 "livenessProbe"
```

## 重启策略对比

| 特性 | Always | OnFailure | Never |
|-----|--------|-----------|-------|
| **触发条件** | 任何退出 | 非零退出码 | 不触发 |
| **正常退出行为** | 重启 | 不重启 | 不重启 |
| **异常退出行为** | 重启 | 重启 | 不重启 |
| **适用控制器** | Deployment, DaemonSet, StatefulSet | Job, CronJob | Job, CronJob |
| **典型应用** | Web 服务, API 服务, 微服务 | 批处理任务, 定时任务 | 调试, 一次性任务 |
| **资源消耗** | 持续运行 | 任务完成后释放 | 任务完成后释放 |
| **可用性** | 高(自动恢复) | 中(失败重试) | 低(需人工干预) |
| **调试难度** | 中(日志可能被覆盖) | 低(保留失败现场) | 低(保留完整现场) |

## 常见问题

### 1. 为什么 Deployment 不支持 OnFailure 或 Never 策略?

Deployment 的设计目标是管理长期运行的服务,这些服务需要持续可用。如果使用 OnFailure 或 Never 策略,当容器正常退出(退出码 0)时,Pod 会停止运行,导致服务不可用。Deployment 控制器期望 Pod 始终处于运行状态,因此只支持 Always 策略。

### 2. Job 使用 OnFailure 和 Never 有什么区别?

对于 Job 控制器:
- **OnFailure**: 容器失败时在同一 Pod 内重启,保留之前的文件系统和网络状态,适合需要重试但不需要完全重新初始化的任务
- **Never**: 容器失败时创建新的 Pod,完全重新开始,适合需要干净环境或需要保留失败 Pod 用于调试的场景

### 3. 如何避免 Pod 进入 CrashLoopBackOff 状态?

避免 CrashLoopBackOff 的关键措施:
- 合理配置资源请求和限制,避免 OOM
- 正确配置健康检查,避免误判
- 确保应用启动逻辑正确,避免启动失败
- 检查依赖服务是否可用,如数据库、配置中心等
- 查看应用日志,修复代码错误
- 使用 init 容器进行启动前的依赖检查

### 4. Pod 重启会保留数据吗?

Pod 重启的数据保留情况:
- **容器内文件系统**: 重启后数据丢失(除非使用 emptyDir 或持久卷)
- **emptyDir 卷**: 重启后数据保留,Pod 删除后数据丢失
- **持久卷(PV/PVC)**: 重启和 Pod 删除后数据都保留
- **ConfigMap/Secret**: 作为卷挂载时,重启后数据保留

### 5. 重启策略与存活探针有什么关系?

重启策略和存活探针是两个独立但相关的机制:
- **存活探针(livenessProbe)**: 检测容器是否存活,失败时根据重启策略决定是否重启
- **重启策略**: 决定容器退出后的行为

存活探针失败会导致容器退出,退出码为 137(被 SIGKILL 终止),然后根据重启策略决定是否重启。如果配置为 Always 或 OnFailure,容器会被重启;如果配置为 Never,容器不会被重启。

## 最佳实践

### 1. 根据应用类型选择合适的重启策略

- **长期运行的服务**: 使用 Always 策略,配合 Deployment 控制器
- **批处理任务**: 使用 OnFailure 策略,配合 Job 控制器,设置合理的 backoffLimit
- **定时任务**: 使用 OnFailure 策略,配合 CronJob 控制器
- **调试场景**: 使用 Never 策略,保留失败现场

### 2. 合理配置资源限制

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

资源限制可以防止应用占用过多资源导致节点问题,同时避免因资源不足导致的 OOM。

### 3. 配置健康检查

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

健康检查可以及时发现应用异常,避免流量路由到不健康的 Pod。

### 4. 设置合理的 Job 重试次数

```yaml
spec:
  backoffLimit: 3          # 最多重试 3 次
  activeDeadlineSeconds: 3600  # 最长运行 1 小时
```

避免无限重试导致资源浪费。

### 5. 使用 init 容器进行依赖检查

```yaml
initContainers:
- name: check-dependencies
  image: busybox
  command: ['sh', '-c', 'until nc -z mysql-service 3306; do sleep 1; done']
```

确保依赖服务可用后再启动主容器,避免因依赖不可用导致的启动失败。

### 6. 记录应用日志到标准输出

```yaml
containers:
- name: app
  image: my-app:v1.0
  # 应用日志输出到 stdout 和 stderr
  # Kubernetes 会自动收集到日志系统
```

便于通过 `kubectl logs` 查看日志,即使 Pod 重启也能查看之前的日志。

### 7. 监控 Pod 重启次数

设置告警规则,当 Pod 重启次数异常时及时通知:

```yaml
# Prometheus 告警规则示例
- alert: PodRestartingTooOften
  expr: increase(kube_pod_container_status_restarts_total[1h]) > 5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Pod {{ $labels.pod }} 重启次数过多"
    description: "Pod {{ $labels.namespace }}/{{ $labels.pod }} 在过去 1 小时内重启了 {{ $value }} 次"
```

## 面试回答

**问题**: Kubernetes 中 Pod 故障重启策略有哪几种?

**回答**: Kubernetes 提供了三种 Pod 重启策略。第一种是 **Always** 策略,无论容器以何种状态退出都会重启,适用于 Web 服务、API 服务等需要长期运行的应用,也是 Deployment、DaemonSet、StatefulSet 等控制器的默认策略。第二种是 **OnFailure** 策略,仅当容器异常退出(退出码非 0)时才重启,适用于批处理任务、定时任务等正常完成后应该停止的场景,通常配合 Job 和 CronJob 使用。第三种是 **Never** 策略,无论容器以何种状态退出都不重启,适用于调试排错、一次性任务或需要人工干预的场景。

需要注意的是,不同的控制器对重启策略有限制:Deployment、ReplicaSet 只支持 Always;Job、CronJob 支持 OnFailure 和 Never。当容器持续启动失败时,Pod 会进入 CrashLoopBackOff 状态,Kubelet 会采用指数退避算法(10秒、20秒、40秒...最大5分钟)来避免频繁重启导致的资源浪费。在实际应用中,应该根据应用类型选择合适的重启策略,并配合资源限制、健康检查、依赖检查等机制,确保应用的稳定运行。

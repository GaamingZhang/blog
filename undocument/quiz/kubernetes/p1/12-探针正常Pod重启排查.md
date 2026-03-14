---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - 探针
  - 排查
---

# 存活探针正常但Pod一直重启的排查

## 问题现象

在Kubernetes集群中，有时会遇到这样的奇怪现象：
- 存活探针（Liveness Probe）检查正常通过
- 但Pod却在不断重启
- 重启次数持续增加

这种情况往往让人困惑：既然探针正常，为什么还会重启？

## Pod重启的可能原因

在深入排查之前，先了解Pod可能被重启的所有原因：

```
┌─────────────────────────────────────────────────────────────┐
│                    Pod重启原因分类                           │
│                                                              │
│  1. 存活探针失败                                             │
│     - HTTP检查失败                                           │
│     - TCP检查失败                                            │
│     - 命令检查失败                                           │
│                                                              │
│  2. 容器进程退出                                             │
│     - 主进程崩溃                                             │
│     - OOM被杀                                                │
│     - 信号导致退出                                           │
│                                                              │
│  3. 资源限制                                                 │
│     - 内存超限被OOM Kill                                     │
│     - CPU限流（不会重启，但会变慢）                           │
│                                                              │
│  4. 健康检查配置问题                                         │
│     - 探针配置不当                                           │
│     - 超时时间过短                                           │
│     - 初始延迟不足                                           │
│                                                              │
│  5. 节点问题                                                 │
│     - 节点故障                                               │
│     - 节点资源不足                                           │
│     - 节点驱逐                                               │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 排查步骤

### 第一步：查看Pod状态

```bash
kubectl get pod <pod-name> -o wide

NAME        READY   STATUS    RESTARTS   AGE   IP           NODE
nginx-xxx   1/1     Running   15         1h    10.1.1.100   node1
```

关注`RESTARTS`列，如果持续增加，说明Pod在不断重启。

### 第二步：查看Pod事件

```bash
kubectl describe pod <pod-name>

Events:
  Type     Reason     Age                From               Message
  ----     ------     ----               ----               -------
  Normal   Pulled     10m (x15 over 1h)  kubelet            Container image pulled
  Normal   Created    10m (x15 over 1h)  kubelet            Created container nginx
  Normal   Started    10m (x15 over 1h)  kubelet            Started container nginx
  Warning  BackOff    9m (x30 over 1h)   kubelet            Back-off restarting failed container
```

**关键信息**：
- `Back-off restarting`：容器退出后被重启
- `OOMKilled`：内存超限被杀
- `Error`：容器以非零状态码退出

### 第三步：查看容器退出状态

```bash
kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[0].lastState}' | jq

{
  "terminated": {
    "exitCode": 137,
    "reason": "OOMKilled",
    "startedAt": "2024-01-15T10:00:00Z",
    "finishedAt": "2024-01-15T10:05:00Z"
  }
}
```

**退出码含义**：

| 退出码 | 含义 |
|--------|------|
| 0 | 正常退出 |
| 1 | 应用错误 |
| 137 | SIGKILL（通常OOM） |
| 139 | SIGSEGV（段错误） |
| 143 | SIGTERM（正常终止） |

### 第四步：检查OOM情况

```bash
kubectl describe pod <pod-name> | grep -A5 "Last State"

Last State:     Terminated
  Reason:       OOMKilled
  Exit Code:    137
```

**如果是OOMKilled**：
- 检查内存限制是否合理
- 检查应用是否有内存泄漏
- 增加内存限制

### 第五步：查看容器日志

```bash
kubectl logs <pod-name> --previous
```

`--previous`参数查看上一个容器的日志，这对于排查重启原因非常重要。

### 第六步：检查探针配置

```bash
kubectl get pod <pod-name> -o yaml | grep -A20 livenessProbe

livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 1
  failureThreshold: 3
```

## 常见场景分析

### 场景1：OOM导致重启

**现象**：
- 存活探针正常
- Pod频繁重启
- 退出码137

**原因**：
- 内存限制设置过低
- 应用存在内存泄漏
- JVM等运行时内存配置不当

**排查**：

```bash
kubectl describe pod <pod-name> | grep -A5 "Last State"

Last State:     Terminated
  Reason:       OOMKilled
  Exit Code:    137
```

**解决**：

```yaml
resources:
  limits:
    memory: "512Mi"  # 增加内存限制
  requests:
    memory: "256Mi"
```

对于Java应用：

```yaml
env:
- name: JAVA_OPTS
  value: "-Xmx400m -Xms400m"
resources:
  limits:
    memory: "512Mi"
```

### 场景2：启动探针和存活探针冲突

**现象**：
- Pod启动后很快重启
- 存活探针检查正常
- 但在启动阶段就失败了

**原因**：
- 没有配置启动探针
- 存活探针的initialDelaySeconds太短
- 应用启动时间较长

**排查**：

```bash
kubectl describe pod <pod-name>

Events:
  Type     Reason     Age   From     Message
  ----     ------     ----  ----     -------
  Normal   Started    10s   kubelet  Started container
  Warning  Unhealthy  15s   kubelet  Liveness probe failed
```

**解决**：

```yaml
startupProbe:
  httpGet:
    path: /healthz
    port: 8080
  failureThreshold: 30
  periodSeconds: 10

livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
```

### 场景3：探针超时导致误判

**现象**：
- 应用在高负载时重启
- 正常情况下探针正常
- 探针偶尔超时

**原因**：
- timeoutSeconds设置过短
- 应用响应慢导致超时
- 探针检查逻辑耗时

**排查**：

```bash
kubectl logs <pod-name> | grep -i timeout
```

**解决**：

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  timeoutSeconds: 5      # 增加超时时间
  periodSeconds: 15      # 增加检查间隔
  failureThreshold: 5    # 增加失败阈值
```

### 场景4：应用内部错误导致退出

**现象**：
- 存活探针正常
- 应用主进程崩溃
- 退出码非0

**原因**：
- 应用代码bug
- 未捕获的异常
- 依赖服务不可用

**排查**：

```bash
kubectl logs <pod-name> --previous

# 查看退出码
kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[0].lastState.terminated.exitCode}'
```

**解决**：
- 修复应用bug
- 添加异常处理
- 检查依赖服务

### 场景5：探针检查了错误的端点

**现象**：
- 探针检查正常
- 但主进程崩溃
- 探针端点和主服务分离

**原因**：
- 探针检查的是健康检查端点
- 主服务进程崩溃
- 健康检查端点仍在运行

**排查**：

```yaml
livenessProbe:
  httpGet:
    path: /healthz    # 检查这个端点
    port: 8080
```

如果应用架构是：
- 主服务在8080端口
- 健康检查在单独的线程/进程

那么主服务崩溃时，健康检查可能仍然正常。

**解决**：
- 确保探针检查能反映主服务状态
- 或使用进程级别的健康检查

### 场景6：容器运行时问题

**现象**：
- Pod在不同节点表现不同
- 某些节点频繁重启

**原因**：
- 容器运行时配置问题
- 节点资源不足
- 节点内核问题

**排查**：

```bash
# 在节点上查看容器日志
journalctl -u containerd -f

# 查看节点资源
kubectl describe node <node-name>
```

## 完整排查脚本

```bash
#!/bin/bash

POD_NAME=$1

echo "=== 1. Pod基本信息 ==="
kubectl get pod $POD_NAME -o wide

echo -e "\n=== 2. 重启次数 ==="
kubectl get pod $POD_NAME -o jsonpath='{.status.containerStatuses[0].restartCount}'

echo -e "\n=== 3. 上次退出状态 ==="
kubectl get pod $POD_NAME -o jsonpath='{.status.containerStatuses[0].lastState}'

echo -e "\n=== 4. Pod事件 ==="
kubectl describe pod $POD_NAME | grep -A20 Events

echo -e "\n=== 5. 上一个容器日志 ==="
kubectl logs $POD_NAME --previous --tail=100

echo -e "\n=== 6. 当前容器日志 ==="
kubectl logs $POD_NAME --tail=100

echo -e "\n=== 7. 探针配置 ==="
kubectl get pod $POD_NAME -o yaml | grep -A20 "livenessProbe\|readinessProbe\|startupProbe"

echo -e "\n=== 8. 资源配置 ==="
kubectl get pod $POD_NAME -o yaml | grep -A10 resources
```

## 排查流程图

```
Pod频繁重启
    │
    ├─→ 退出码137？
    │       │
    │       └─→ OOMKilled → 增加内存限制
    │                   → 检查内存泄漏
    │
    ├─→ 退出码0？
    │       │
    │       └─→ 正常退出 → 检查应用逻辑
    │                   → 是否主动退出
    │
    ├─→ 退出码1？
    │       │
    │       └─→ 应用错误 → 查看日志
    │                   → 修复代码bug
    │
    ├─→ 探针失败？
    │       │
    │       └─→ 调整探针配置
    │                   → 增加超时时间
    │                   → 增加初始延迟
    │
    └─→ 节点问题？
            │
            └─→ 检查节点状态
                        → 检查节点资源
```

## 最佳实践

### 1. 合理配置探针

```yaml
startupProbe:
  httpGet:
    path: /healthz
    port: 8080
  failureThreshold: 30
  periodSeconds: 10

livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 15
  timeoutSeconds: 5
  failureThreshold: 5

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
  failureThreshold: 3
```

### 2. 合理配置资源

```yaml
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "500m"
    memory: "512Mi"
```

### 3. 监控Pod重启

```yaml
groups:
- name: pod-restart
  rules:
  - alert: PodRestartingFrequently
    expr: increase(kube_pod_container_status_restarts_total[1h]) > 5
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Pod频繁重启"
```

### 4. 日志收集

确保收集Pod日志，便于排查问题。

## 参考资源

- [Pod生命周期](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)
- [探针配置](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)

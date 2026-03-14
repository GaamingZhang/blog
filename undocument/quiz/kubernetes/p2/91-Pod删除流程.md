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
  - 生命周期
---

# Kubernetes Pod 删除流程详解

## 引言

在 Kubernetes 集群的日常运维中，Pod 删除是一个常见的操作。无论是应用更新、故障恢复还是资源释放，都需要删除 Pod。然而，Pod 的删除过程并非简单的"移除"，而是一个涉及多个组件协作的复杂流程。

理解 Pod 删除的完整流程，包括优雅终止、信号处理、资源清理等环节，对于确保应用平滑下线、避免数据丢失和实现零停机更新至关重要。本文将深入剖析 Kubernetes Pod 删除的完整流程。

## Pod 删除流程概述

### 删除流程图

```
┌─────────────────────────────────────────────────────────────┐
│                    Pod 删除完整流程                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐                                            │
│  │ kubectl     │                                            │
│  │ delete pod  │                                            │
│  └──────┬──────┘                                            │
│         │                                                    │
│         ▼                                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              API Server                              │   │
│  │  1. 验证请求权限                                      │   │
│  │  2. 更新 Pod 元数据（deletionTimestamp）              │   │
│  │  3. 触发优雅终止流程                                  │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Endpoints Controller                    │   │
│  │  1. 从 Endpoints 中移除 Pod IP                       │   │
│  │  2. 更新 Service 的 Endpoints                        │   │
│  │  3. kube-proxy 更新 iptables/IPVS                    │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              kubelet                                 │   │
│  │  1. 检测到 Pod 标记为删除                            │   │
│  │  2. 执行 preStop 钩子                                │   │
│  │  3. 发送 SIGTERM 信号                                │   │
│  │  4. 等待 terminationGracePeriodSeconds               │   │
│  │  5. 发送 SIGKILL 信号（如果容器未退出）              │   │
│  │  6. 清理容器和存储                                   │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              API Server                              │   │
│  │  1. 从 etcd 中删除 Pod 对象                          │   │
│  │  2. 删除完成                                         │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 关键时间参数

| 参数 | 默认值 | 说明 |
|-----|--------|------|
| **terminationGracePeriodSeconds** | 30 秒 | 优雅终止等待时间 |
| **preStop 钩子执行时间** | 计入优雅终止时间 | 容器停止前执行的钩子 |
| **SIGTERM 后等待时间** | 剩余优雅终止时间 | 应用处理终止信号的时间 |

## 详细流程解析

### 第一阶段：API Server 处理

#### 1. 接收删除请求

```bash
# 删除 Pod 的几种方式
kubectl delete pod <pod-name>
kubectl delete pod <pod-name> --grace-period=60
kubectl delete pod <pod-name> --force --grace-period=0
```

#### 2. 设置删除时间戳

当 API Server 接收到删除请求后，会设置 Pod 的 `deletionTimestamp`：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  deletionTimestamp: "2026-03-12T10:00:00Z"
  deletionGracePeriodSeconds: 30
  ...
```

#### 3. 触发优雅终止

API Server 会通知相关控制器开始优雅终止流程。

### 第二阶段：Endpoints 移除

#### 1. 从 Service Endpoints 中移除

Endpoints Controller 监听到 Pod 标记为删除后，会立即从 Endpoints 中移除该 Pod：

```yaml
# Endpoints 更新前
apiVersion: v1
kind: Endpoints
metadata:
  name: my-service
subsets:
- addresses:
  - ip: 10.244.1.5  # 即将删除的 Pod
  - ip: 10.244.1.6
  - ip: 10.244.1.7

# Endpoints 更新后
apiVersion: v1
kind: Endpoints
metadata:
  name: my-service
subsets:
- addresses:
  - ip: 10.244.1.6
  - ip: 10.244.1.7
```

#### 2. kube-proxy 更新规则

kube-proxy 监听到 Endpoints 变化后，会更新节点上的 iptables 或 IPVS 规则，确保新流量不再路由到即将删除的 Pod。

#### 3. 流量排空

这个过程确保：
- 新请求不会发送到正在终止的 Pod
- 正在处理的请求可以继续完成
- 实现平滑的流量迁移

### 第三阶段：kubelet 处理

#### 1. 检测删除标记

kubelet 通过 watch API 监听到 Pod 被标记为删除，开始执行终止流程。

#### 2. 执行 preStop 钩子

如果配置了 `preStop` 钩子，kubelet 会先执行它：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  containers:
  - name: app
    image: my-app:v1
    lifecycle:
      preStop:
        exec:
          command:
          - /bin/sh
          - -c
          - |
            # 通知应用准备关闭
            curl -X POST http://localhost:8080/shutdown
            # 等待处理中的请求完成
            sleep 10
```

**preStop 钩子特点**：
- 同步执行，会阻塞终止流程
- 执行时间计入 `terminationGracePeriodSeconds`
- 如果超时，会被强制终止
- 钩子失败不会阻止容器终止

#### 3. 发送 SIGTERM 信号

preStop 钩子执行完成后，kubelet 向容器发送 SIGTERM 信号：

```bash
# 容器内进程收到信号
kill -TERM <pid>
```

**应用处理 SIGTERM**：

```python
# Python 示例
import signal
import sys

def signal_handler(sig, frame):
    print('Received SIGTERM, shutting down gracefully...')
    # 停止接收新请求
    # 完成处理中的请求
    # 关闭数据库连接
    # 清理资源
    sys.exit(0)

signal.signal(signal.SIGTERM, signal_handler)
```

```go
// Go 示例
package main

import (
    "os"
    "os/signal"
    "syscall"
)

func main() {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM)
    
    <-sigChan
    // 优雅关闭
    shutdown()
}
```

#### 4. 等待优雅终止

kubelet 等待容器退出，最长等待 `terminationGracePeriodSeconds`：

```yaml
spec:
  terminationGracePeriodSeconds: 60  # 默认 30 秒
  containers:
  - name: app
    image: my-app:v1
```

**时间计算**：
```
总等待时间 = terminationGracePeriodSeconds
已用时间 = preStop 执行时间 + SIGTERM 后等待时间
剩余时间 = 总等待时间 - 已用时间
```

#### 5. 发送 SIGKILL 信号

如果容器在优雅终止时间内未退出，kubelet 会发送 SIGKILL 信号强制终止：

```bash
kill -KILL <pid>
```

**SIGKILL 特点**：
- 无法被捕获或忽略
- 立即终止进程
- 可能导致数据丢失

### 第四阶段：资源清理

#### 1. 停止容器

```bash
# kubelet 调用容器运行时停止容器
crictl stop <container-id>
```

#### 2. 清理存储

```bash
# 清理 emptyDir 卷
rm -rf /var/lib/kubelet/pods/<pod-uid>/volumes/kubernetes.io~empty-dir

# 解除 PVC 绑定（如果配置了删除策略）
```

#### 3. 清理网络

```bash
# 调用 CNI 插件清理网络
# 释放 IP 地址
# 清理网络接口
```

### 第五阶段：从 etcd 删除

#### 1. 确认终止完成

kubelet 确认所有容器都已停止后，通知 API Server。

#### 2. 从 etcd 删除对象

API Server 从 etcd 中删除 Pod 对象，Pod 彻底消失。

## 强制删除流程

### 强制删除命令

```bash
# 强制删除（跳过优雅终止）
kubectl delete pod <pod-name> --force --grace-period=0

# 强制删除卡住的 Pod
kubectl delete pod <pod-name> --force --grace-period=0 --wait=false
```

### 强制删除的影响

```
┌─────────────────────────────────────────────────────────────┐
│                    强制删除流程                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 绕过优雅终止流程                                         │
│  2. 直接从 etcd 删除 Pod 对象                               │
│  3. kubelet 可能仍在运行容器                                 │
│  4. 容器可能变成孤儿进程                                     │
│                                                              │
│  风险：                                                      │
│  - 数据丢失                                                  │
│  - 资源泄漏                                                  │
│  - 状态不一致                                                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 强制删除场景

- Pod 卡在 Terminating 状态
- 节点不可达
- 紧急故障恢复

## 删除过程中的 Pod 状态

### 状态变化

```
Running -> Terminating -> (Deleted)
```

### 查看 Terminating 状态

```bash
# 查看 Pod 状态
kubectl get pod <pod-name>

# 查看详细信息
kubectl describe pod <pod-name>

# 查看事件
kubectl get events --field-selector involvedObject.name=<pod-name>
```

### Terminating 状态卡住的原因

| 原因 | 说明 | 解决方案 |
|-----|------|---------|
| **preStop 钩子卡住** | 钩子执行时间过长 | 检查钩子脚本 |
| **应用不响应 SIGTERM** | 未处理终止信号 | 修改应用代码 |
| **存储卸载失败** | NFS 等存储问题 | 检查存储状态 |
| **Finalizer 未完成** | 控制器未清理 | 检查控制器状态 |
| **节点不可达** | kubelet 无法通信 | 检查节点状态 |

## 优雅终止最佳实践

### 1. 配置合理的终止时间

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  terminationGracePeriodSeconds: 60
  containers:
  - name: app
    image: my-app:v1
```

**建议值**：
- 简单应用：30 秒（默认）
- 需要处理请求的应用：60-120 秒
- 数据库等有状态应用：300 秒以上

### 2. 实现 preStop 钩子

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  containers:
  - name: app
    image: my-app:v1
    lifecycle:
      preStop:
        exec:
          command:
          - /bin/sh
          - -c
          - |
            # 1. 从服务发现中注销
            curl -X DELETE http://localhost:8080/deregister
            # 2. 等待处理中的请求完成
            sleep 15
```

### 3. 正确处理 SIGTERM

```yaml
# 应用代码示例
# 确保应用正确处理 SIGTERM 信号
```

```go
// Go 应用优雅关闭
func main() {
    server := &http.Server{Addr: ":8080"}
    
    go func() {
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()
    
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
    <-quit
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Printf("Server shutdown error: %v", err)
    }
}
```

### 4. 配合 Service 使用

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 3
  template:
    spec:
      terminationGracePeriodSeconds: 60
      containers:
      - name: app
        image: my-app:v1
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "sleep 15"]
```

### 5. 使用 Pod Disruption Budget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: my-app-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: my-app
```

## 删除流程监控

### 监控指标

```yaml
# Prometheus 指标
kube_pod_status_phase{phase="Running"}
kube_pod_container_state_terminated_reason{reason="Completed"}
kube_pod_container_state_terminated_reason{reason="Error"}
```

### 告警规则

```yaml
groups:
- name: pod-termination
  rules:
  - alert: PodStuckInTerminating
    expr: |
      time() - kube_pod_deletion_timestamp 
      > 300
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Pod {{ $labels.pod }} stuck in Terminating state"
      
  - alert: PodForceDeleted
    expr: |
      increase(kube_pod_container_state_terminated_reason{reason="ContainerStatusUnknown"}[5m]) > 0
    labels:
      severity: info
    annotations:
      summary: "Pod {{ $labels.pod }} was force deleted"
```

## 常见问题排查

### Q1: Pod 卡在 Terminating 状态

```bash
# 检查 Pod 详情
kubectl describe pod <pod-name>

# 检查 Finalizer
kubectl get pod <pod-name> -o jsonpath='{.metadata.finalizers}'

# 移除 Finalizer
kubectl patch pod <pod-name> -p '{"metadata":{"finalizers":null}}'

# 强制删除
kubectl delete pod <pod-name> --force --grace-period=0
```

### Q2: 应用不响应 SIGTERM

```bash
# 检查容器日志
kubectl logs <pod-name>

# 进入容器检查
kubectl exec -it <pod-name> -- /bin/sh

# 检查进程信号处理
kill -TERM 1
```

### Q3: preStop 钩子执行失败

```bash
# 查看事件
kubectl describe pod <pod-name>

# 检查钩子脚本
kubectl get pod <pod-name> -o yaml | grep -A 10 lifecycle
```

## 面试回答

**问题**: 简述 Kubernetes 中删除 Pod 的流程。

**回答**: Kubernetes 删除 Pod 是一个多阶段协作的优雅终止流程。首先是 **API Server 处理**，接收到删除请求后设置 `deletionTimestamp`，标记 Pod 为删除状态，触发优雅终止流程。

其次是 **Endpoints 移除**，Endpoints Controller 监听到删除标记后，从 Service 的 Endpoints 中移除该 Pod IP，kube-proxy 更新 iptables/IPVS 规则，确保新流量不再路由到即将删除的 Pod，实现流量排空。

第三是 **kubelet 处理**，kubelet 检测到删除标记后，按顺序执行：先执行 `preStop` 钩子（如果配置），然后发送 SIGTERM 信号给容器进程，等待 `terminationGracePeriodSeconds`（默认 30 秒），如果容器未退出则发送 SIGKILL 强制终止，最后清理容器、存储和网络资源。

第四是 **资源清理**，停止容器、清理 emptyDir 卷、解除 PVC 绑定、调用 CNI 清理网络。

最后是 **从 etcd 删除**，确认终止完成后，API Server 从 etcd 中删除 Pod 对象。

强制删除使用 `--force --grace-period=0` 参数，会跳过优雅终止直接从 etcd 删除，可能导致数据丢失和资源泄漏，仅在紧急情况使用。生产环境应配置合理的 `terminationGracePeriodSeconds`、实现 `preStop` 钩子、正确处理 SIGTERM 信号，确保应用平滑下线。

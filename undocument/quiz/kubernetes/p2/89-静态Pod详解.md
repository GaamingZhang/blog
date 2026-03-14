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
  - 静态Pod
---

# Kubernetes 静态 Pod 详解

## 引言

在 Kubernetes 集群中，大多数 Pod 都是由控制器（如 Deployment、StatefulSet）管理的，它们通过 API Server 创建和管理。然而，有一种特殊的 Pod 类型——静态 Pod（Static Pod），它由节点上的 kubelet 直接管理，而不经过 API Server。

静态 Pod 在 Kubernetes 集群的引导和系统组件部署中扮演着重要角色。理解静态 Pod 的工作原理、使用场景和配置方法，对于深入理解 Kubernetes 集群的运作机制至关重要。

## 静态 Pod 概述

### 什么是静态 Pod

静态 Pod 是由 kubelet 直接管理的 Pod，而不是通过 API Server 管理。kubelet 会监视特定的目录，自动创建该目录下定义的 Pod，并将它们反映到 API Server 中（作为只读的 Mirror Pod）。

### 静态 Pod 与普通 Pod 的区别

| 特性 | 静态 Pod | 普通 Pod |
|-----|---------|---------|
| **管理方式** | kubelet 直接管理 | API Server 管理 |
| **创建方式** | 文件系统上的 YAML 文件 | kubectl apply 或 API 调用 |
| **API Server 角色** | 只读镜像 | 完全控制 |
| **节点绑定** | 固定在特定节点 | 可被调度到任意节点 |
| **生命周期** | 依赖 kubelet 和配置文件 | 依赖控制器 |
| **重启策略** | 总是重启 | 可配置 |
| **适用场景** | 系统组件、自举 | 应用程序 |

### 静态 Pod 的特点

1. **节点固定**：静态 Pod 只能在定义它的节点上运行
2. **自愈能力**：kubelet 会自动重启失败的静态 Pod
3. **不依赖 API Server**：即使 API Server 不可用，静态 Pod 仍可运行
4. **只读镜像**：在 API Server 中只能查看，不能修改
5. **自动清理**：配置文件删除后，Pod 会被自动删除

## 静态 Pod 工作原理

### 工作流程

```
┌─────────────────────────────────────────────────────────────┐
│                    静态 Pod 工作流程                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────┐                                        │
│  │  配置文件目录    │  /etc/kubernetes/manifests/*.yaml      │
│  │  (manifests)    │                                        │
│  └────────┬────────┘                                        │
│           │                                                  │
│           ▼                                                  │
│  ┌─────────────────┐                                        │
│  │    kubelet      │  监视目录，检测文件变化                  │
│  │                 │  创建/更新/删除 Pod                     │
│  └────────┬────────┘                                        │
│           │                                                  │
│           ├──────────────────────┐                          │
│           │                      │                          │
│           ▼                      ▼                          │
│  ┌─────────────────┐    ┌─────────────────┐                │
│  │  本地 Pod       │    │  Mirror Pod     │                │
│  │  (实际运行)     │    │  (API Server)   │                │
│  └─────────────────┘    └─────────────────┘                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Mirror Pod 机制

当 kubelet 创建静态 Pod 时，它会在 API Server 中创建一个对应的 Mirror Pod：

- Mirror Pod 的名称以节点名作为后缀：`<pod-name>-<node-name>`
- Mirror Pod 是只读的，不能通过 kubectl 修改
- Mirror Pod 的 `ownerReferences` 指向节点
- 删除 Mirror Pod 不会删除实际的静态 Pod

```yaml
# Mirror Pod 示例
apiVersion: v1
kind: Pod
metadata:
  name: etcd-control-plane
  namespace: kube-system
  ownerReferences:
  - apiVersion: v1
    kind: Node
    name: control-plane
    uid: <node-uid>
  labels:
    component: etcd
    tier: control-plane
  annotations:
    kubernetes.io/config.hash: <hash>
    kubernetes.io/config.mirror: "true"
    kubernetes.io/config.source: file
```

## 静态 Pod 配置

### 配置目录

kubelet 通过 `--pod-manifest-path` 或 `staticPodPath` 配置项指定静态 Pod 的配置目录：

```yaml
# kubelet 配置文件
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
staticPodPath: /etc/kubernetes/manifests
```

或通过命令行参数：

```bash
kubelet --pod-manifest-path=/etc/kubernetes/manifests
```

### 创建静态 Pod

#### 方式一：直接创建 YAML 文件

```bash
# 创建配置文件
cat > /etc/kubernetes/manifests/static-web.yaml <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: static-web
  namespace: default
spec:
  containers:
  - name: web
    image: nginx:1.21
    ports:
    - containerPort: 80
EOF
```

#### 方式二：使用 kubeadm 部署控制平面

kubeadm 使用静态 Pod 部署 Kubernetes 控制平面组件：

```bash
# 查看控制平面静态 Pod
ls /etc/kubernetes/manifests/
# etcd.yaml
# kube-apiserver.yaml
# kube-controller-manager.yaml
# kube-scheduler.yaml
```

### 静态 Pod 配置示例

#### Nginx 静态 Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: static-nginx
  namespace: default
  labels:
    app: static-nginx
spec:
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
    readinessProbe:
      httpGet:
        path: /
        port: 80
      initialDelaySeconds: 5
      periodSeconds: 5
    volumeMounts:
    - name: html
      mountPath: /usr/share/nginx/html
  volumes:
  - name: html
    hostPath:
      path: /var/www/html
      type: Directory
```

#### 监控 Agent 静态 Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: node-exporter
  namespace: monitoring
  labels:
    app: node-exporter
spec:
  hostNetwork: true
  hostPID: true
  containers:
  - name: node-exporter
    image: prom/node-exporter:v1.3.0
    args:
    - --path.procfs=/host/proc
    - --path.sysfs=/host/sys
    - --path.rootfs=/host/root
    ports:
    - name: metrics
      containerPort: 9100
      hostPort: 9100
    volumeMounts:
    - name: proc
      mountPath: /host/proc
      readOnly: true
    - name: sys
      mountPath: /host/sys
      readOnly: true
    - name: root
      mountPath: /host/root
      readOnly: true
  volumes:
  - name: proc
    hostPath:
      path: /proc
  - name: sys
    hostPath:
      path: /sys
  - name: root
    hostPath:
      path: /
```

## 静态 Pod 使用场景

### 1. Kubernetes 控制平面组件

kubeadm 部署的集群使用静态 Pod 运行控制平面组件：

```yaml
# kube-apiserver.yaml
apiVersion: v1
kind: Pod
metadata:
  name: kube-apiserver
  namespace: kube-system
spec:
  containers:
  - command:
    - kube-apiserver
    - --advertise-address=192.168.1.10
    - --allow-privileged=true
    - --authorization-mode=Node,RBAC
    - --client-ca-file=/etc/kubernetes/pki/ca.crt
    - --enable-admission-plugins=NodeRestriction
    - --enable-bootstrap-token-auth=true
    - --etcd-servers=https://127.0.0.1:2379
    - --kubelet-client-certificate=/etc/kubernetes/pki/apiserver-kubelet-client.crt
    - --kubelet-client-key=/etc/kubernetes/pki/apiserver-kubelet-client.key
    - --secure-port=6443
    - --service-account-issuer=https://kubernetes.default.svc.cluster.local
    - --service-account-key-file=/etc/kubernetes/pki/sa.pub
    - --service-account-signing-key-file=/etc/kubernetes/pki/sa.key
    - --service-cluster-ip-range=10.96.0.0/12
    - --tls-cert-file=/etc/kubernetes/pki/apiserver.crt
    - --tls-private-key-file=/etc/kubernetes/pki/apiserver.key
    image: k8s.gcr.io/kube-apiserver:v1.25.0
    name: kube-apiserver
    volumeMounts:
    - mountPath: /etc/kubernetes/pki
      name: k8s-certs
      readOnly: true
    - mountPath: /etc/ssl/certs
      name: ca-certs
      readOnly: true
  hostNetwork: true
  priorityClassName: system-node-critical
  volumes:
  - hostPath:
      path: /etc/kubernetes/pki
      type: DirectoryOrCreate
    name: k8s-certs
  - hostPath:
      path: /etc/ssl/certs
      type: DirectoryOrCreate
    name: ca-certs
```

### 2. 节点级监控和日志

```yaml
# Fluentd 日志收集
apiVersion: v1
kind: Pod
metadata:
  name: fluentd
  namespace: logging
spec:
  containers:
  - name: fluentd
    image: fluent/fluentd:v1.14
    volumeMounts:
    - name: varlog
      mountPath: /var/log
      readOnly: true
    - name: varlibdockercontainers
      mountPath: /var/lib/docker/containers
      readOnly: true
  volumes:
  - name: varlog
    hostPath:
      path: /var/log
  - name: varlibdockercontainers
    hostPath:
      path: /var/lib/docker/containers
```

### 3. 网络插件

```yaml
# CNI 插件示例
apiVersion: v1
kind: Pod
metadata:
  name: calico-node
  namespace: kube-system
spec:
  hostNetwork: true
  containers:
  - name: calico-node
    image: calico/node:v3.23.0
    env:
    - name: DATASTORE_TYPE
      value: "kubernetes"
    - name: FELIX_LOGSEVERITYSCREEN
      value: "info"
    securityContext:
      privileged: true
    volumeMounts:
    - name: lib-modules
      mountPath: /lib/modules
      readOnly: true
    - name: var-run-calico
      mountPath: /var/run/calico
  volumes:
  - name: lib-modules
    hostPath:
      path: /lib/modules
  - name: var-run-calico
    hostPath:
      path: /var/run/calico
```

### 4. 自举场景

静态 Pod 可以用于集群自举，因为它不依赖 API Server：

- 在 API Server 启动前部署必要的服务
- 部署 API Server 本身
- 部署 etcd

## 静态 Pod 管理

### 查看静态 Pod

```bash
# 查看节点上的静态 Pod
kubectl get pods -n kube-system -o wide | grep <node-name>

# 查看静态 Pod 详情
kubectl describe pod <pod-name> -n kube-system

# 查看静态 Pod 配置
cat /etc/kubernetes/manifests/<pod-name>.yaml
```

### 更新静态 Pod

```bash
# 编辑配置文件
vi /etc/kubernetes/manifests/static-web.yaml

# kubelet 会自动检测变化并更新 Pod
# 更新过程是原地更新，不是滚动更新
```

### 删除静态 Pod

```bash
# 删除配置文件或移动到其他目录
rm /etc/kubernetes/manifests/static-web.yaml
# 或
mv /etc/kubernetes/manifests/static-web.yaml /tmp/

# kubelet 会自动删除 Pod
# 注意：通过 kubectl delete 删除 Mirror Pod 无效，Pod 会被重新创建
```

### 查看静态 Pod 日志

```bash
# 使用 kubectl 查看日志
kubectl logs <pod-name> -n kube-system

# 使用 crictl 查看日志（在节点上）
crictl logs <container-id>

# 使用 docker 查看日志（在节点上）
docker logs <container-id>
```

## 静态 Pod 与 DaemonSet 对比

| 特性 | 静态 Pod | DaemonSet |
|-----|---------|-----------|
| **管理方式** | kubelet | 控制器管理器 |
| **创建方式** | 文件系统 YAML | kubectl apply |
| **API Server 依赖** | 不依赖 | 依赖 |
| **节点选择** | 固定节点 | 可配置 nodeSelector |
| **更新方式** | 修改文件 | 滚动更新 |
| **监控** | 需要手动监控 | 自动管理 |
| **适用场景** | 系统组件、自举 | 每节点一个 Pod |

### 何时使用静态 Pod

**适合使用静态 Pod**：
- Kubernetes 控制平面组件
- 不依赖 API Server 的服务
- 需要在 API Server 启动前运行的服务
- 节点级系统服务

**适合使用 DaemonSet**：
- 日志收集 Agent
- 监控 Agent
- 网络插件
- 存储插件

## 静态 Pod 最佳实践

### 1. 配置文件管理

```bash
# 使用版本控制管理静态 Pod 配置
/etc/kubernetes/manifests/
├── etcd.yaml
├── kube-apiserver.yaml
├── kube-controller-manager.yaml
└── kube-scheduler.yaml

# 备份配置文件
cp -r /etc/kubernetes/manifests /etc/kubernetes/manifests.backup
```

### 2. 资源限制

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: static-app
spec:
  containers:
  - name: app
    image: my-app:v1
    resources:
      requests:
        memory: "256Mi"
        cpu: "250m"
      limits:
        memory: "512Mi"
        cpu: "500m"
```

### 3. 健康检查

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: static-app
spec:
  containers:
  - name: app
    image: my-app:v1
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
```

### 4. 安全配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: static-app
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
  containers:
  - name: app
    image: my-app:v1
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
```

### 5. 监控和告警

```yaml
# Prometheus 告警规则
- alert: StaticPodNotRunning
  expr: |
    kube_pod_status_phase{phase!="Running"} 
    and on(pod) kube_pod_annotations{annotation_kubernetes_io_config_source="file"}
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Static Pod {{ $labels.pod }} is not running"
```

## 常见问题

### Q1: 如何区分静态 Pod 和普通 Pod？

```bash
# 查看注解
kubectl get pod <pod-name> -n kube-system -o yaml | grep -A 5 annotations

# 静态 Pod 会有以下注解：
# kubernetes.io/config.source: file
# kubernetes.io/config.mirror: "true"
```

### Q2: 删除 Mirror Pod 后会发生什么？

Mirror Pod 删除后，kubelet 会自动重新创建它。要真正删除静态 Pod，必须删除节点上的配置文件。

### Q3: 静态 Pod 可以使用 ConfigMap 和 Secret 吗？

可以，但需要确保 ConfigMap 和 Secret 已经创建：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: static-app
spec:
  containers:
  - name: app
    image: my-app:v1
    env:
    - name: CONFIG
      valueFrom:
        configMapKeyRef:
          name: app-config
          key: config
```

### Q4: 如何更新静态 Pod 的镜像？

修改配置文件中的镜像版本，kubelet 会自动检测并更新：

```bash
# 编辑配置文件
sed -i 's|image: nginx:1.20|image: nginx:1.21|' /etc/kubernetes/manifests/static-web.yaml

# kubelet 会自动重启 Pod
```

## 面试回答

**问题**: 什么是静态 Pod？

**回答**: 静态 Pod 是由 kubelet 直接管理的 Pod，而不是通过 API Server 管理。kubelet 会监视特定的目录（通常是 `/etc/kubernetes/manifests`），自动创建该目录下定义的 Pod，并在 API Server 中创建一个只读的 Mirror Pod。

静态 Pod 的主要特点包括：**节点固定**，只能在定义它的节点上运行；**不依赖 API Server**，即使 API Server 不可用，静态 Pod 仍可运行；**自愈能力**，kubelet 会自动重启失败的静态 Pod；**自动清理**，配置文件删除后，Pod 会被自动删除。

静态 Pod 的典型使用场景是部署 Kubernetes 控制平面组件（kube-apiserver、kube-controller-manager、kube-scheduler、etcd），因为这些组件需要在 API Server 启动前运行，不能依赖 API Server 来管理。此外，静态 Pod 也用于部署节点级的监控和日志收集 Agent。

静态 Pod 与 DaemonSet 的主要区别是：静态 Pod 由 kubelet 管理，不依赖 API Server；DaemonSet 由控制器管理器管理，依赖 API Server。静态 Pod 适合系统组件和自举场景，DaemonSet 适合每节点一个 Pod 的应用场景。删除静态 Pod 必须删除节点上的配置文件，通过 kubectl delete 只会删除 Mirror Pod，Pod 会被 kubelet 重新创建。

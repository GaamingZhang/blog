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
  - 静态Pod
---

# Kubernetes Master节点组件的运行方式

## 一个有趣的问题

当你执行 `kubectl get pods -n kube-system` 时，会看到Master节点的组件以Pod的形式运行：

```bash
NAME                                       READY   STATUS    RESTARTS   AGE
etcd-master1                               1/1     Running   0          10d
kube-apiserver-master1                     1/1     Running   0          10d
kube-controller-manager-master1            1/1     Running   0          10d
kube-scheduler-master1                     1/1     Running   0          10d
```

但这带来一个"鸡生蛋"的问题：**这些组件是Pod，但Pod需要Kubernetes来管理。如果这些组件还没运行，Kubernetes怎么管理它们？**

答案就是：**静态Pod（Static Pod）**。

## 静态Pod：Kubernetes的自举机制

### 什么是静态Pod？

静态Pod是由kubelet直接管理的Pod，不需要API Server介入。它们是Kubernetes的"自举"机制——让控制平面组件能够以Pod的形式运行，而不需要先有一个运行中的Kubernetes。

### 静态Pod的工作原理

```
┌─────────────────────────────────────────────────────────────┐
│                        Master Node                           │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                   kubelet                            │    │
│  │                                                      │    │
│  │  监听 /etc/kubernetes/manifests/ 目录                │    │
│  │         ↓                                            │    │
│  │  发现Pod配置文件                                      │    │
│  │         ↓                                            │    │
│  │  创建Pod（不经过API Server）                          │    │
│  │         ↓                                            │    │
│  │  向API Server报告Pod状态（只读）                      │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  /etc/kubernetes/manifests/                                  │
│  ├── etcd.yaml                                               │
│  ├── kube-apiserver.yaml                                     │
│  ├── kube-controller-manager.yaml                            │
│  └── kube-scheduler.yaml                                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### kubelet配置

kubelet通过配置参数指定静态Pod的目录：

```bash
# 查看kubelet配置
ps aux | grep kubelet

# 输出中包含
--pod-manifest-path=/etc/kubernetes/manifests
# 或
--config=/var/lib/kubelet/config.yaml
```

在config.yaml中：

```yaml
staticPodPath: /etc/kubernetes/manifests
```

## Master组件的启动流程

### 完整启动序列

```
【第1步】系统启动kubelet和containerd
    systemctl start containerd
    systemctl start kubelet
    ↓
【第2步】kubelet扫描manifests目录
    发现静态Pod配置文件
    ↓
【第3步】kubelet创建静态Pod
    按顺序启动：etcd → api-server → scheduler → controller-manager
    ↓
【第4步】API Server启动后
    kubelet开始向API Server注册节点信息
    ↓
【第5步】集群可用
    所有组件正常运行
```

### 各组件的启动顺序

虽然kubelet会并发启动所有静态Pod，但组件之间有依赖关系：

```
etcd
  ↓ (etcd必须先启动)
kube-apiserver
  ↓ (API Server必须先启动)
kube-scheduler + kube-controller-manager
```

这是因为：
- API Server依赖etcd存储数据
- Scheduler和Controller Manager依赖API Server

### 静态Pod配置文件示例

**kube-apiserver.yaml**

```yaml
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
    - --etcd-cafile=/etc/kubernetes/pki/etcd/ca.crt
    - --etcd-certfile=/etc/kubernetes/pki/apiserver-etcd-client.crt
    - --etcd-keyfile=/etc/kubernetes/pki/apiserver-etcd-client.key
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
    image: registry.k8s.io/kube-apiserver:v1.28.0
    imagePullPolicy: IfNotPresent
    livenessProbe:
      failureThreshold: 8
      httpGet:
        host: 192.168.1.10
        path: /livez
        port: 6443
        scheme: HTTPS
      initialDelaySeconds: 10
      periodSeconds: 10
    name: kube-apiserver
    resources:
      requests:
        cpu: 250m
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

**kube-scheduler.yaml**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kube-scheduler
  namespace: kube-system
spec:
  containers:
  - command:
    - kube-scheduler
    - --authentication-kubeconfig=/etc/kubernetes/scheduler.conf
    - --authorization-kubeconfig=/etc/kubernetes/scheduler.conf
    - --bind-address=127.0.0.1
    - --kubeconfig=/etc/kubernetes/scheduler.conf
    - --leader-elect=true
    image: registry.k8s.io/kube-scheduler:v1.28.0
    imagePullPolicy: IfNotPresent
    livenessProbe:
      failureThreshold: 8
      httpGet:
        host: 127.0.0.1
        path: /healthz
        port: 10259
        scheme: HTTPS
      initialDelaySeconds: 10
      periodSeconds: 10
    name: kube-scheduler
    resources:
      requests:
        cpu: 100m
    volumeMounts:
    - mountPath: /etc/kubernetes/scheduler.conf
      name: kubeconfig
      readOnly: true
  hostNetwork: true
  priorityClassName: system-node-critical
  volumes:
  - hostPath:
      path: /etc/kubernetes/scheduler.conf
      type: FileOrCreate
    name: kubeconfig
```

## 静态Pod vs 普通Pod

| 特性 | 静态Pod | 普通Pod |
|------|---------|---------|
| 管理者 | kubelet | API Server + Controller |
| 配置来源 | 本地文件 | API Server |
| 创建方式 | kubelet直接创建 | 通过kubectl/API创建 |
| 可否删除 | 删除文件才能删除Pod | kubectl delete pod |
| 可否更新 | 修改文件自动更新 | kubectl apply |
| 镜像拉取策略 | 默认IfNotPresent | 默认Always |
| 健康检查 | kubelet执行 | kubelet执行 |
| 重启策略 | 总是重启 | 按restartPolicy |

## 静态Pod的镜像Pod

当kubelet创建静态Pod后，它会在API Server中创建一个"镜像Pod"（Mirror Pod）。这个镜像Pod只是状态的反映，不能通过API修改。

```bash
kubectl get pod kube-apiserver-master1 -n kube-system -o yaml

apiVersion: v1
kind: Pod
metadata:
  annotations:
    kubernetes.io/config.hash: 3d6e9c4b5f7a8d9c
    kubernetes.io/config.mirror: "true"
    kubernetes.io/config.seen: "2024-01-01T00:00:00.000000000Z"
    kubernetes.io/config.source: file
  name: kube-apiserver-master1
  namespace: kube-system
  ownerReferences:
  - apiVersion: v1
    kind: Node
    name: master1
    uid: xxx-xxx-xxx
```

**关键注解**：
- `kubernetes.io/config.source: file` 表示来源是静态文件
- `kubernetes.io/config.mirror: "true"` 表示这是镜像Pod

## 如何管理静态Pod

### 查看静态Pod配置

```bash
ls -la /etc/kubernetes/manifests/

-rw------- 1 root root 3456 Jan  1 00:00 etcd.yaml
-rw------- 1 root root 4567 Jan  1 00:00 kube-apiserver.yaml
-rw------- 1 root root 2345 Jan  1 00:00 kube-controller-manager.yaml
-rw------- 1 root root 2123 Jan  1 00:00 kube-scheduler.yaml
```

### 修改静态Pod配置

```bash
# 直接编辑配置文件
vim /etc/kubernetes/manifests/kube-apiserver.yaml

# kubelet会自动检测变化并重新创建Pod
```

### 临时停止静态Pod

```bash
# 将配置文件移出manifests目录
mv /etc/kubernetes/manifests/kube-apiserver.yaml /tmp/

# kubelet会自动删除对应的Pod
```

### 恢复静态Pod

```bash
# 将配置文件移回manifests目录
mv /tmp/kube-apiserver.yaml /etc/kubernetes/manifests/

# kubelet会自动创建Pod
```

## 其他运行方式

虽然kubeadm默认使用静态Pod，但Master组件也可以通过其他方式运行：

### 1. Systemd服务

传统部署方式，将组件作为系统服务运行：

```bash
systemctl status kube-apiserver
systemctl status kube-scheduler
systemctl status kube-controller-manager
systemctl status etcd
```

优点：
- 更传统，运维人员熟悉
- 不依赖容器运行时

缺点：
- 升级管理复杂
- 不能利用Kubernetes的自愈能力

### 2. 自托管（Self-hosted）

将控制平面组件作为普通Pod运行，由Kubernetes自己管理。这种方式需要引导集群。

优点：
- 统一管理方式
- 可以利用Kubernetes的特性

缺点：
- 启动复杂，需要引导
- 存在循环依赖问题

## kubeadm的选择

kubeadm选择静态Pod的原因：

1. **简单可靠**：不依赖API Server就能启动
2. **自包含**：所有配置都在manifests目录
3. **易于升级**：修改配置文件即可
4. **故障恢复**：kubelet自动重启失败的Pod
5. **一致性**：与普通Pod使用相同的容器镜像

## 常见问题

### Q1: 修改了静态Pod配置，但Pod没有更新？

检查kubelet是否正在运行：

```bash
systemctl status kubelet
journalctl -u kubelet -f
```

可能原因：
- kubelet未运行
- 配置文件格式错误
- 配置文件权限问题

### Q2: 静态Pod一直重启？

查看Pod日志：

```bash
kubectl logs kube-apiserver-master1 -n kube-system
# 或
crictl logs <container-id>
```

常见原因：
- 配置参数错误
- 证书问题
- 端口冲突
- 资源不足

### Q3: 如何备份静态Pod配置？

```bash
# 备份manifests目录
tar -czvf kube-manifests-backup.tar.gz /etc/kubernetes/manifests/

# 备份证书
tar -czvf kube-pki-backup.tar.gz /etc/kubernetes/pki/
```

### Q4: 删除了镜像Pod，它又自动出现了？

这是正常行为。静态Pod由kubelet管理，删除镜像Pod后，kubelet会重新创建。

要真正删除，需要删除manifests目录下的配置文件。

## 最佳实践

1. **不要手动修改**：使用kubeadm管理配置
2. **备份配置**：定期备份manifests目录
3. **监控日志**：关注kubelet和组件日志
4. **证书管理**：监控证书过期时间
5. **资源预留**：为Master组件预留足够资源

## 总结

Master节点组件通过**静态Pod**方式运行：

| 组件 | 运行方式 | 管理者 |
|------|----------|--------|
| etcd | 静态Pod | kubelet |
| kube-apiserver | 静态Pod | kubelet |
| kube-scheduler | 静态Pod | kubelet |
| kube-controller-manager | 静态Pod | kubelet |

静态Pod是Kubernetes的自举机制，让控制平面组件能够以Pod形式运行，同时不依赖运行中的Kubernetes。配置文件存放在 `/etc/kubernetes/manifests/` 目录，kubelet监听该目录并管理Pod的生命周期。

理解静态Pod的工作原理，对于排查Master组件问题、进行集群维护都非常重要。

## 参考资源

- [静态Pod官方文档](https://kubernetes.io/docs/tasks/configure-pod-container/static-pod/)
- [kubeadm设计文档](https://github.com/kubernetes/kubeadm/blob/main/docs/design/design_v1.8.md)

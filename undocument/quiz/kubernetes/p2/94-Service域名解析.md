---
date: 2026-03-12
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Service
  - DNS
---

# Kubernetes Service 域名解析详解

## 引言

在 Kubernetes 集群中，Service 是实现服务发现的核心组件。通过 Service，应用程序可以使用稳定的网络端点访问后端 Pod，而无需关心 Pod 的动态变化。Kubernetes 通过内置的 DNS 服务，为每个 Service 分配一个域名，使得服务发现变得简单而透明。

理解 Kubernetes Service 的域名解析机制，包括域名格式、解析流程、DNS 配置等，对于构建可靠的微服务架构至关重要。本文将深入剖析 Service 域名解析的各个方面。

## Service 域名格式

### 完整域名格式

Kubernetes Service 的完整域名格式为：

```
<service-name>.<namespace>.svc.<cluster-domain>
```

### 域名组成部分

| 组成部分 | 说明 | 示例 |
|---------|------|------|
| **service-name** | Service 名称 | my-service |
| **namespace** | 命名空间 | default |
| **svc** | 固定值，表示 Service | svc |
| **cluster-domain** | 集群域名 | cluster.local |

### 域名示例

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: production
spec:
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
```

该 Service 的完整域名为：
```
my-service.production.svc.cluster.local
```

### 域名缩写

在同一个命名空间内，可以使用缩写形式：

```bash
# 完整域名
my-service.production.svc.cluster.local

# 省略集群域名
my-service.production.svc

# 省略命名空间（同一命名空间内）
my-service

# 跨命名空间访问
my-service.production
```

## DNS 解析流程

### 解析架构

```
┌─────────────────────────────────────────────────────────────┐
│                    DNS 解析流程                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐                                            │
│  │   Pod       │                                            │
│  │  应用请求   │                                            │
│  └──────┬──────┘                                            │
│         │                                                    │
│         ▼                                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              /etc/resolv.conf                        │   │
│  │  nameserver 10.96.0.10                               │   │
│  │  search default.svc.cluster.local svc.cluster.local  │   │
│  │         cluster.local                                │   │
│  │  options ndots:5                                     │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              CoreDNS Service                         │   │
│  │              (10.96.0.10)                            │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              CoreDNS Pod                             │   │
│  │              解析域名 -> 返回 ClusterIP              │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Pod DNS 配置

```bash
# 查看 Pod 的 DNS 配置
kubectl exec -it <pod-name> -- cat /etc/resolv.conf

# 典型输出
nameserver 10.96.0.10
search default.svc.cluster.local svc.cluster.local cluster.local
options ndots:5
```

### 配置项说明

| 配置项 | 说明 |
|-------|------|
| **nameserver** | DNS 服务器地址（CoreDNS Service IP） |
| **search** | 域名搜索列表，用于补全短域名 |
| **ndots** | 域名中点号数量阈值，决定是否使用搜索列表 |

### ndots 参数详解

`ndots` 参数决定域名解析的行为：

- 如果域名中的点号数量 >= ndots，则直接解析
- 如果域名中的点号数量 < ndots，则依次添加 search 后缀尝试解析

```bash
# ndots:5
# 解析 my-service（0 个点号 < 5）
# 尝试顺序：
# 1. my-service.default.svc.cluster.local
# 2. my-service.svc.cluster.local
# 3. my-service.cluster.local
# 4. my-service

# 解析 my-service.production（1 个点号 < 5）
# 尝试顺序：
# 1. my-service.production.default.svc.cluster.local
# 2. my-service.production.svc.cluster.local
# 3. my-service.production.cluster.local
# 4. my-service.production

# 解析 my-service.production.svc.cluster.local（4 个点号 < 5）
# 尝试顺序：
# 1. my-service.production.svc.cluster.local.default.svc.cluster.local
# 2. my-service.production.svc.cluster.local.svc.cluster.local
# 3. my-service.production.svc.cluster.local.cluster.local
# 4. my-service.production.svc.cluster.local
```

## CoreDNS 配置

### CoreDNS ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    .:53 {
        errors
        health {
            lameduck 5s
        }
        ready
        kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
            ttl 30
        }
        prometheus :9153
        forward . /etc/resolv.conf {
            max_concurrent 1000
        }
        cache 30
        loop
        reload
        loadbalance
    }
```

### CoreDNS 配置项说明

| 配置项 | 说明 |
|-------|------|
| **errors** | 错误日志输出 |
| **health** | 健康检查端点 |
| **ready** | 就绪检查端点 |
| **kubernetes** | Kubernetes DNS 解析插件 |
| **prometheus** | Prometheus 指标端点 |
| **forward** | 转发外部 DNS 查询 |
| **cache** | DNS 缓存 |
| **loop** | 检测 DNS 解析循环 |
| **reload** | 自动重载配置 |
| **loadbalance** | 负载均衡 DNS 响应 |

### 自定义 DNS 配置

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    # 自定义域名解析
    example.com:53 {
        errors
        cache 30
        forward . 8.8.8.8:53
    }
    # Kubernetes 默认配置
    .:53 {
        errors
        health
        kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
        }
        forward . /etc/resolv.conf
        cache 30
    }
```

## Pod DNS 策略

### dnsPolicy 配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: dns-demo
spec:
  dnsPolicy: ClusterFirst
  containers:
  - name: app
    image: nginx
```

### DNS 策略类型

| 策略 | 说明 |
|-----|------|
| **ClusterFirst** | 默认策略，优先使用集群 DNS |
| **Default** | 继承节点的 DNS 配置 |
| **ClusterFirstWithHostNet** | 使用 hostNetwork 时的集群 DNS |
| **None** | 自定义 DNS 配置 |

### ClusterFirst 策略

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: cluster-first
spec:
  dnsPolicy: ClusterFirst
  containers:
  - name: app
    image: nginx
```

### Default 策略

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: default-dns
spec:
  dnsPolicy: Default
  containers:
  - name: app
    image: nginx
```

### 自定义 DNS 配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: custom-dns
spec:
  dnsPolicy: None
  dnsConfig:
    nameservers:
    - 8.8.8.8
    - 8.8.4.4
    searches:
    - default.svc.cluster.local
    - svc.cluster.local
    - cluster.local
    options:
    - name: ndots
      value: "2"
    - name: timeout
      value: "2"
  containers:
  - name: app
    image: nginx
```

### hostNetwork 与 DNS

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: host-network
spec:
  hostNetwork: true
  dnsPolicy: ClusterFirstWithHostNet
  containers:
  - name: app
    image: nginx
```

## Headless Service 域名解析

### Headless Service 配置

```yaml
apiVersion: v1
kind: Service
metadata:
  name: headless-service
spec:
  clusterIP: None
  selector:
    app: my-app
  ports:
  - port: 80
```

### 域名解析结果

```bash
# 解析 Headless Service
nslookup headless-service.default.svc.cluster.local

# 返回所有 Pod IP
Name:    headless-service.default.svc.cluster.local
Address: 10.244.1.5
Address: 10.244.1.6
Address: 10.244.1.7
```

### Pod 域名

对于 StatefulSet 管理的 Pod，每个 Pod 都有独立的域名：

```
<pod-name>.<service-name>.<namespace>.svc.cluster.local
```

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  serviceName: headless-service
  replicas: 3
  template:
    spec:
      containers:
      - name: web
        image: nginx
```

Pod 域名：
```
web-0.headless-service.default.svc.cluster.local
web-1.headless-service.default.svc.cluster.local
web-2.headless-service.default.svc.cluster.local
```

## Service 域名解析测试

### 使用 nslookup

```bash
# 进入 Pod
kubectl exec -it <pod-name> -- sh

# 解析 Service
nslookup my-service
nslookup my-service.production
nslookup my-service.production.svc.cluster.local

# 解析 Headless Service
nslookup headless-service

# 解析 Pod 域名
nslookup web-0.headless-service
```

### 使用 dig

```bash
# 解析 Service
dig my-service.default.svc.cluster.local

# 解析 Headless Service
dig headless-service.default.svc.cluster.local

# 查看解析过程
dig +trace my-service.default.svc.cluster.local
```

### 使用 getent

```bash
# 解析域名
getent hosts my-service
getent hosts my-service.production
```

## 外部域名解析

### ExternalName Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-service
spec:
  type: ExternalName
  externalName: external.example.com
```

```bash
# 解析 ExternalName Service
nslookup external-service.default.svc.cluster.local

# 返回 CNAME
external-service.default.svc.cluster.local -> external.example.com
```

### 配置外部 DNS 服务器

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: external-dns
spec:
  dnsPolicy: None
  dnsConfig:
    nameservers:
    - 8.8.8.8
    searches:
    - default.svc.cluster.local
  containers:
  - name: app
    image: nginx
```

## DNS 故障排查

### 常见问题

| 问题 | 原因 | 解决方案 |
|-----|------|---------|
| **DNS 解析失败** | CoreDNS 未运行 | 检查 CoreDNS Pod |
| **解析超时** | 网络问题 | 检查网络连通性 |
| **域名不存在** | Service 不存在 | 检查 Service 配置 |
| **解析结果错误** | CoreDNS 配置错误 | 检查 CoreDNS ConfigMap |

### 排查步骤

```bash
# 1. 检查 CoreDNS 状态
kubectl get pods -n kube-system -l k8s-app=kube-dns

# 2. 检查 CoreDNS 日志
kubectl logs -n kube-system <coredns-pod>

# 3. 检查 CoreDNS Service
kubectl get svc -n kube-system kube-dns

# 4. 检查 Pod DNS 配置
kubectl exec -it <pod-name> -- cat /etc/resolv.conf

# 5. 测试 DNS 解析
kubectl exec -it <pod-name> -- nslookup kubernetes

# 6. 检查 CoreDNS ConfigMap
kubectl get configmap coredns -n kube-system -o yaml
```

### DNS 调试 Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: dns-debug
spec:
  containers:
  - name: dns-debug
    image: tutum/dnsutils
    command:
    - sleep
    - "3600"
```

```bash
# 创建调试 Pod
kubectl apply -f dns-debug.yaml

# 进入 Pod 调试
kubectl exec -it dns-debug -- nslookup kubernetes
```

## 最佳实践

### 1. 使用完整域名

```yaml
# 推荐：使用完整域名
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  containers:
  - name: app
    image: my-app
    env:
    - name: DB_HOST
      value: "mysql.production.svc.cluster.local"
```

### 2. 合理配置 ndots

```yaml
# 对于频繁访问外部域名，降低 ndots
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  dnsConfig:
    options:
    - name: ndots
      value: "2"
  containers:
  - name: app
    image: my-app
```

### 3. 使用 Service 别名

```yaml
# 使用 ExternalName 创建服务别名
apiVersion: v1
kind: Service
metadata:
  name: db
spec:
  type: ExternalName
  externalName: mysql.production.svc.cluster.local
```

### 4. 监控 DNS 性能

```yaml
# Prometheus DNS 监控指标
- alert: CoreDNSDown
  expr: up{job="coredns"} == 0
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "CoreDNS is down"
```

## 面试回答

**问题**: Kubernetes Service 的域名解析格式是什么？

**回答**: Kubernetes Service 的完整域名格式为 `<service-name>.<namespace>.svc.<cluster-domain>`，其中 `cluster-domain` 默认为 `cluster.local`。例如，名为 `my-service` 的 Service 在 `production` 命名空间中的完整域名是 `my-service.production.svc.cluster.local`。

在同一命名空间内，可以使用短域名 `my-service` 访问；跨命名空间访问使用 `my-service.production`。这种域名解析由 CoreDNS 提供，Pod 通过 `/etc/resolv.conf` 配置使用集群 DNS 服务器。

`resolv.conf` 的关键配置包括：`nameserver` 指向 CoreDNS Service IP；`search` 域名搜索列表，用于补全短域名；`ndots` 参数决定域名解析策略，默认为 5，表示域名中点号数量小于 5 时会依次添加 search 后缀尝试解析。

对于 Headless Service（`clusterIP: None`），域名解析返回所有后端 Pod IP，而不是单个 ClusterIP。StatefulSet 管理的 Pod 还有独立的域名 `<pod-name>.<service-name>.<namespace>.svc.cluster.local`。

Pod 的 DNS 策略包括：`ClusterFirst` 优先使用集群 DNS（默认）；`Default` 继承节点 DNS；`ClusterFirstWithHostNet` 用于 hostNetwork 模式；`None` 允许自定义 DNS 配置。生产环境建议使用完整域名、合理配置 ndots 参数、监控 CoreDNS 性能。

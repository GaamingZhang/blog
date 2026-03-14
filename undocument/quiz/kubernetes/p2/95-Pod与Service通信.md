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
  - Pod
  - 通信
---

# Kubernetes Pod 与 Service 通信详解

## 引言

在 Kubernetes 集群中，Pod 与 Service 之间的通信是微服务架构的基础。Pod 是动态的，IP 地址会随着 Pod 的创建和销毁而变化；Service 提供了一个稳定的访问入口，屏蔽了后端 Pod 的变化。理解 Pod 与 Service 之间的通信机制，对于构建可靠的微服务应用至关重要。

本文将深入剖析 Pod 与 Service 通信的各个方面，包括 Service 的工作原理、流量路由机制、负载均衡策略以及常见问题的排查方法。

## Service 概述

### 为什么需要 Service

在 Kubernetes 中，Pod 的 IP 地址是动态分配的：

- Pod 被调度到节点时分配 IP
- Pod 终止后 IP 被回收
- Pod 重新创建后 IP 变化
- 扩缩容时会有新 Pod 和旧 Pod 共存

Service 提供了一个稳定的虚拟 IP（ClusterIP），将流量路由到后端 Pod：

```
┌─────────────────────────────────────────────────────────────┐
│                    Service 工作原理                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐                                            │
│  │   Client   │                                            │
│  │   Pod A    │                                            │
│  └──────┬──────┘                                            │
│         │ 访问 my-service                                   │
│         ▼                                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Service (my-service)                    │   │
│  │              ClusterIP: 10.96.0.100                   │   │
│  │              Selector: app=my-app                    │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│         ┌───────────────┼───────────────┐                  │
│         ▼               ▼               ▼                  │
│  ┌───────────┐   ┌───────────┐   ┌───────────┐           │
│  │  Pod B    │   │  Pod C    │   │  Pod D    │           │
│  │ 10.244.1.5│   │10.244.1.6 │   │10.244.1.7 │           │
│  └───────────┘   └───────────┘   └───────────┘           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Service 类型

### ClusterIP

ClusterIP 是默认的 Service 类型，只在集群内部可访问：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
```

### NodePort

NodePort 在每个节点上开放一个端口：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: NodePort
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30080
```

### LoadBalancer

LoadBalancer 使用云厂商的负载均衡器：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: LoadBalancer
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
```

### ExternalName

ExternalName 映射到外部域名：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ExternalName
  externalName: external.example.com
```

## Pod 与 Service 通信流程

### 通信流程详解

```
┌─────────────────────────────────────────────────────────────┐
│               Pod 与 Service 通信完整流程                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 创建 Service                                             │
│     ┌─────────────────────────────────────────────┐        │
│     │ Service Controller 创建 Endpoint             │        │
│     │ Endpoint Controller 监听 Pod 变化           │        │
│     │ 更新 Endpoints 对象                         │        │
│     └─────────────────────────────────────────────┘        │
│                         │                                   │
│  2. kube-proxy 监听变化                                     │
│     ┌─────────────────────────────────────────────┐        │
│     │ kube-proxy watch Endpoints                  │        │
│     │ 更新节点上的 iptables/IPVS 规则              │        │
│     └─────────────────────────────────────────────┘        │
│                         │                                   │
│  3. 客户端发起请求                                           │
│     ┌─────────────────────────────────────────────┐        │
│     │ Client Pod 解析域名 -> ClusterIP             │        │
│     │ 发送请求到 ClusterIP                        │        │
│     └─────────────────────────────────────────────┘        │
│                         │                                   │
│  4. 流量路由                                                │
│     ┌─────────────────────────────────────────────┐        │
│     │ iptables/IPVS 根据规则选择后端 Pod          │        │
│     │ 负载均衡到某个 Pod                          │        │
│     └─────────────────────────────────────────────┘        │
│                         │                                   │
│  5. 响应返回                                                │
│     ┌─────────────────────────────────────────────┐        │
│     │ 直接从目标 Pod 返回到 Client Pod             │        │
│     │ 不经过 Service（SNAT/DNAT）                 │        │
│     └─────────────────────────────────────────────┘        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 关键组件

| 组件 | 作用 |
|-----|------|
| **Service Controller** | 创建和管理 Service |
| **Endpoint Controller** | 维护 Endpoints（后端 Pod 列表） |
| **kube-proxy** | 同步 iptables/IPVS 规则 |
| **CoreDNS** | Service 域名解析 |
| **CNI 插件** | Pod 网络连通性 |

## kube-proxy 代理模式

### iptables 模式（默认）

```bash
# 查看 iptables 规则
iptables -L -t nat -n | grep KUBE-SERVICES

# 查看 Service 规则
iptables -L KUBE-SVC -t nat -n
```

iptables 规则结构：

```
┌─────────────────────────────────────────────────────────────┐
│                  iptables 规则结构                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  KUBE-SERVICES (全局 Service 链)                           │
│  ├── KUBE-MARK-MASQ (标记需要 SNAT)                        │
│  ├── KUBE-SVC-XXX (各 Service 入口)                       │
│  │   ├── KUBE-SEP-XXX1 (后端 Pod 1)                       │
│  │   ├── KUBE-SEP-XXX2 (后端 Pod 2)                       │
│  │   └── KUBE-SEP-XXX3 (后端 Pod 3)                       │
│  └── ...                                                    │
│                                                              │
│  负载均衡算法：随机（默认）                                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### IPVS 模式

```yaml
# 启用 IPVS 模式
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: IPVS
ipvs:
  scheduler: rr  # 轮询
  # wrr - 加权轮询
  # lc - 最少连接
  # wlc - 加权最少连接
```

```bash
# 查看 IPVS 规则
ipvsadm -L -n

# 查看 Service 规则
ipvsadm -L -n | grep <service-ip>
```

### 各模式对比

| 特性 | iptables | IPVS |
|-----|---------|------|
| **性能** | O(n) | O(1) |
| **负载均衡算法** | 随机 | 轮询、加权轮询、最少连接等 |
| **支持会话保持** | 需要额外配置 | 支持 |
| **扩展性** | 受限于规则数量 | 高 |
| **复杂度** | 简单 | 较复杂 |

## Service 端点管理

### Endpoints

```yaml
apiVersion: v1
kind: Endpoints
metadata:
  name: my-service
  namespace: default
subsets:
- addresses:
  - ip: 10.244.1.5
    targetRef:
      kind: Pod
      name: my-app-pod-1
      namespace: default
  - ip: 10.244.1.6
    targetRef:
      kind: Pod
      name: my-app-pod-2
      namespace: default
  ports:
  - port: 8080
    protocol: TCP
```

### EndpointSlice（推荐）

```yaml
apiVersion: discovery.k8s.io/v1
kind: EndpointSlice
metadata:
  name: my-service-abcde
  labels:
    kubernetes.io/service-name: my-service
addressType: IPv4
ports:
- name: http
  port: 8080
  protocol: TCP
endpoints:
- addresses:
  - 10.244.1.5
  conditions:
    ready: true
    serving: true
    terminating: false
- addresses:
  - 10.244.1.6
  conditions:
    ready: true
    serving: true
    terminating: false
```

## Pod 如何发现 Service

### 环境变量

```bash
# 进入 Pod 查看环境变量
kubectl exec -it <pod-name> -- env | grep SERVICE

# 输出示例
MY_SERVICE_SERVICE_HOST=10.96.0.100
MY_SERVICE_SERVICE_PORT=80
MY_SERVICE_PORT=tcp://10.96.0.100:80
MY_SERVICE_PORT_80_TCP=tcp://10.96.0.100:80
MY_SERVICE_PORT_80_TCP_PROTO=tcp
MY_SERVICE_PORT_80_TCP_PORT=80
MY_SERVICE_PORT_80_TCP_ADDR=10.96.0.100
```

### DNS 解析

```bash
# 使用 DNS 解析 Service
nslookup my-service

# 完整域名
nslookup my-service.default.svc.cluster.local

# 解析结果
Name:    my-service.default.svc.cluster.local
Address: 10.96.0.100
```

### 对比

| 发现方式 | 优点 | 缺点 |
|---------|------|------|
| **环境变量** | 简单，无需额外配置 | 依赖 Service 创建顺序 |
| **DNS** | 灵活，支持跨命名空间 | 需要 DNS 解析 |

## 流量路由机制

### 客户端访问流程

```bash
# 1. 解析域名
$ nslookup my-service
Name:    my-service.default.svc.cluster.local
Address: 10.96.0.100

# 2. 发送请求到 ClusterIP
$ curl http://10.96.0.100

# 3. iptables/IPVS 路由
# 请求被负载均衡到某个 Pod
```

### 负载均衡算法

```yaml
# kube-proxy iptables 模式
# 随机选择后端 Pod

# kube-proxy IPVS 模式
# 支持多种算法：
# - rr (round-robin)
# - wrr (weighted round-robin)
# - lc (least connections)
# - wlc (weighted least connections)
# - se (source hash)
# - de (destination hash)
```

### 会话保持

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800
```

## 常见问题排查

### Service 无法访问

```bash
# 1. 检查 Service 存在
kubectl get svc my-service

# 2. 检查 Endpoints
kubectl get endpoints my-service

# 3. 检查 Pod 标签匹配
kubectl get pods -l app=my-app

# 4. 检查 kube-proxy 状态
kubectl get pods -n kube-system -l k8s-app=kube-proxy

# 5. 检查 iptables 规则
iptables -L KUBE-SERVICES -t nat -n

# 6. 检查网络连通性
kubectl exec -it <pod-name> -- curl -v <service-ip>
```

### Pod 无法解析 Service

```bash
# 1. 检查 DNS 解析
kubectl exec -it <pod-name> -- nslookup my-service

# 2. 检查 CoreDNS
kubectl get pods -n kube-system -l k8s-app=kube-dns

# 3. 检查 /etc/resolv.conf
kubectl exec -it <pod-name> -- cat /etc/resolv.conf
```

### 部分 Pod 无法访问

```bash
# 1. 检查 Endpoint 数量
kubectl get endpoints my-service -o yaml

# 2. 检查 Pod 就绪状态
kubectl get pods -l app=my-app -o wide

# 3. 检查 Pod 健康状态
kubectl describe pod <pod-name>
```

## 最佳实践

### 1. 使用 Service 进行服务发现

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
  type: ClusterIP
```

### 2. 配置健康检查

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: my-app
        image: my-app:v1
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
```

### 3. 使用 Headless Service 进行有状态服务

```yaml
apiVersion: v1
kind: Service
metadata:
  name: mysql-headless
spec:
  clusterIP: None
  selector:
    app: mysql
  ports:
  - port: 3306
```

### 4. 配置外部流量策略

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  externalTrafficPolicy: Cluster  # 或 Local
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
```

- **Cluster**：流量可以路由到任何节点的后端 Pod
- **Local**：流量只路由到本节点的后端 Pod，保留客户端源 IP

### 5. 使用 Ingress 暴露 HTTP 服务

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
spec:
  rules:
  - host: myapp.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-service
            port:
              number: 80
```

## 面试回答

**问题**: Pod 与 Service 的通信是怎么样的？

**回答**: Pod 与 Service 的通信是 Kubernetes 服务发现的核心机制。**Service 机制**：Service 是一个稳定的虚拟 IP（ClusterIP），它通过标签选择器关联后端 Pod，并维护一个 Endpoints 列表记录所有后端 Pod 的 IP 地址。当客户端 Pod 访问 Service 时，流量会被路由到后端 Pod。

**通信流程**：首先，Endpoint Controller 监听 Pod 变化，更新 Endpoints 对象；然后，kube-proxy 监听 Endpoints 变化，在每个节点上同步 iptables 或 IPVS 规则；客户端 Pod 通过 DNS 解析 Service 域名获取 ClusterIP，或者通过环境变量获取 Service IP；请求到达节点后，iptables/IPVS 根据负载均衡规则选择一个后端 Pod 进行转发。

**服务发现方式**：Pod 可以通过两种方式发现 Service——环境变量（每个 Service 会生成对应的环境变量，如 `MY_SERVICE_SERVICE_HOST`）和 DNS（通过 CoreDNS 解析 `my-service.namespace.svc.cluster.local`）。

**负载均衡**：kube-proxy 支持 iptables 和 IPVS 两种模式。iptables 模式使用随机负载均衡，IPVS 模式支持轮询、加权轮询、最少连接等多种算法。

**注意事项**：Pod 必须在同一个集群内才能访问 Service；Service 的 Endpoints 为空时，流量会被丢弃；外部流量可以通过 NodePort、LoadBalancer 或 Ingress 暴露服务。生产环境应配置健康检查确保只将流量路由到健康的 Pod，使用 Headless Service 进行有状态服务部署，合理配置外部流量策略保留客户端源 IP。

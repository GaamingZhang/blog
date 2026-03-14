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
  - 服务发现
---

# Kubernetes Service 详解

## 引言

在 Kubernetes 集群中，Pod 是动态的，它们的 IP 地址会随着创建和销毁而变化。如何让客户端找到这些动态变化的 Pod？Service 就是解决这个问题的核心组件。

Service 是 Kubernetes 中实现服务发现的关键抽象，它为一组 Pod 提供稳定的访问入口。理解 Service 的工作原理和配置方法，是构建可靠微服务架构的基础。

## Service 概述

### 什么是 Service

Service 是 Kubernetes 中的一种抽象，定义了一组 Pod 的逻辑集合和访问策略。Service 通过标签选择器找到对应的 Pod，并提供稳定的访问入口。

### 为什么需要 Service

```
┌─────────────────────────────────────────────────────────────┐
│                    Service 解决的问题                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  问题：Pod IP 动态变化                                       │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Pod 1 (10.244.1.5)  ──删除──>  不存在              │   │
│  │  Pod 2 (10.244.1.6)  ──重启──>  10.244.1.8          │   │
│  │  Pod 3 (10.244.1.7)  ──创建──>  10.244.1.9          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  解决方案：Service 提供稳定访问入口                          │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Service (ClusterIP: 10.96.0.100)        │   │
│  │              通过标签选择器关联 Pod                   │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│         ┌───────────────┼───────────────┐                  │
│         ▼               ▼               ▼                  │
│  ┌───────────┐   ┌───────────┐   ┌───────────┐           │
│  │  Pod 1    │   │  Pod 2    │   │  Pod 3    │           │
│  │ app=web   │   │ app=web   │   │ app=web   │           │
│  └───────────┘   └───────────┘   └───────────┘           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Service 的作用

| 作用 | 说明 |
|-----|------|
| **服务发现** | 提供稳定的访问入口 |
| **负载均衡** | 将流量分发到多个 Pod |
| **服务抽象** | 屏蔽后端 Pod 变化 |

## Service 类型

### ClusterIP（默认）

ClusterIP 类型的 Service 只在集群内部可访问：

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
```

### NodePort

NodePort 类型的 Service 在每个节点上开放一个端口：

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

LoadBalancer 类型的 Service 使用云厂商的负载均衡器：

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

ExternalName 类型的 Service 映射到外部域名：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ExternalName
  externalName: external.example.com
```

### 类型对比

| 类型 | 访问范围 | ClusterIP | NodePort | LoadBalancer |
|-----|---------|-----------|----------|--------------|
| **ClusterIP** | 集群内部 | ✓ | - | - |
| **NodePort** | 集群外部 | ✓ | ✓ | - |
| **LoadBalancer** | 集群外部 | ✓ | ✓ | ✓ |
| **ExternalName** | 外部服务 | - | - | - |

## Service 配置

### 基本配置

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
  labels:
    app: my-app
spec:
  type: ClusterIP
  selector:
    app: my-app
  ports:
  - name: http
    port: 80
    targetPort: 8080
    protocol: TCP
```

### 多端口配置

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  selector:
    app: my-app
  ports:
  - name: http
    port: 80
    targetPort: 8080
    protocol: TCP
  - name: https
    port: 443
    targetPort: 8443
    protocol: TCP
  - name: metrics
    port: 9090
    targetPort: 9090
    protocol: TCP
```

### 端口配置说明

| 字段 | 说明 |
|-----|------|
| **port** | Service 暴露的端口 |
| **targetPort** | Pod 容器端口 |
| **nodePort** | 节点端口（NodePort 类型） |
| **protocol** | 协议（TCP/UDP/SCTP） |

### 无选择器的 Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-service
spec:
  ports:
  - port: 3306
    targetPort: 3306
---
apiVersion: v1
kind: Endpoints
metadata:
  name: external-service
subsets:
- addresses:
  - ip: 10.0.0.1
  ports:
  - port: 3306
```

## Service 工作原理

### 流量路由流程

```
┌─────────────────────────────────────────────────────────────┐
│                  Service 流量路由流程                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 客户端发起请求                                           │
│     ┌─────────────────────────────────────────────────┐    │
│     │ curl http://my-service                          │    │
│     └──────────────────────┬──────────────────────────┘    │
│                            │                                │
│                            ▼                                │
│  2. DNS 解析                                                │
│     ┌─────────────────────────────────────────────────┐    │
│     │ my-service -> 10.96.0.100 (ClusterIP)           │    │
│     └──────────────────────┬──────────────────────────┘    │
│                            │                                │
│                            ▼                                │
│  3. iptables/IPVS 匹配规则                                 │
│     ┌─────────────────────────────────────────────────┐    │
│     │ - 匹配 Service ClusterIP                        │    │
│     │ - 负载均衡选择后端 Pod                          │    │
│     │ - DNAT 转换目标地址                             │    │
│     └──────────────────────┬──────────────────────────┘    │
│                            │                                │
│                            ▼                                │
│  4. 流量转发到后端 Pod                                      │
│     ┌─────────────────────────────────────────────────┐    │
│     │ 10.96.0.100:80 -> 10.244.1.5:8080              │    │
│     └──────────────────────┬──────────────────────────┘    │
│                            │                                │
│                            ▼                                │
│  5. 后端 Pod 响应                                           │
│     ┌─────────────────────────────────────────────────┐    │
│     │ 响应直接返回客户端                              │    │
│     └─────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### kube-proxy 模式

#### iptables 模式

```bash
# 查看 iptables 规则
iptables -L KUBE-SERVICES -t nat -n
```

#### IPVS 模式

```bash
# 查看 IPVS 规则
ipvsadm -L -n
```

### Endpoints

```yaml
apiVersion: v1
kind: Endpoints
metadata:
  name: my-service
subsets:
- addresses:
  - ip: 10.244.1.5
    nodeName: node-1
  - ip: 10.244.1.6
    nodeName: node-2
  ports:
  - port: 8080
    protocol: TCP
```

## Headless Service

### 概念

Headless Service 是将 `clusterIP` 设置为 `None` 的 Service，不分配 ClusterIP。

### 配置示例

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service-headless
spec:
  clusterIP: None
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
```

### DNS 解析

```bash
# 解析 Headless Service
nslookup my-service-headless.default.svc.cluster.local

# 返回所有 Pod IP
Name:    my-service-headless.default.svc.cluster.local
Address: 10.244.1.5
Address: 10.244.1.6
Address: 10.244.1.7
```

### 使用场景

- StatefulSet
- 直接访问特定 Pod
- 客户端自行负载均衡

## Service 高级配置

### 会话保持

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
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800
```

### 外部流量策略

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: NodePort
  externalTrafficPolicy: Cluster  # 或 Local
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
```

| 策略 | 说明 |
|-----|------|
| **Cluster** | 流量可以路由到任何节点的后端 Pod |
| **Local** | 流量只路由到本节点的后端 Pod |

### 内部流量策略

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  internalTrafficPolicy: Local
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
```

### 拓扑感知路由

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    service.kubernetes.io/topology-mode: Auto
spec:
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
```

## Service 操作命令

### 创建 Service

```bash
# 创建 ClusterIP Service
kubectl create service clusterip my-service --tcp=80:8080

# 创建 NodePort Service
kubectl create service nodeport my-service --tcp=80:8080 --node-port=30080

# 从 YAML 创建
kubectl apply -f service.yaml
```

### 查看 Service

```bash
# 查看 Service
kubectl get svc

# 查看详情
kubectl describe svc my-service

# 查看 Endpoints
kubectl get endpoints my-service
```

### 删除 Service

```bash
# 删除 Service
kubectl delete svc my-service

# 从文件删除
kubectl delete -f service.yaml
```

## Service 与 Ingress

### Ingress 概述

Ingress 提供了 HTTP/HTTPS 路由功能：

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

### Service 配合 Ingress

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
---
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

## 最佳实践

### 1. 使用标签选择器

```yaml
spec:
  selector:
    app: my-app
    version: v1
```

### 2. 配置健康检查

```yaml
# Pod 配置 readinessProbe
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
```

### 3. 使用 Service Account

```yaml
spec:
  serviceAccountName: my-service-account
```

### 4. 配置资源限制

```yaml
# Pod 配置资源限制
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "200m"
    memory: "256Mi"
```

### 5. 使用 Network Policy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-my-service
spec:
  podSelector:
    matchLabels:
      app: my-app
  ingress:
  - from:
    - podSelector:
        matchLabels:
          access: allowed
    ports:
    - port: 8080
```

## 常见问题排查

### Service 无法访问

```bash
# 检查 Service 存在
kubectl get svc my-service

# 检查 Endpoints
kubectl get endpoints my-service

# 检查 Pod 标签
kubectl get pods --show-labels

# 检查 kube-proxy
kubectl get pods -n kube-system -l k8s-app=kube-proxy
```

### Endpoints 为空

```bash
# 检查标签选择器
kubectl get svc my-service -o jsonpath='{.spec.selector}'

# 检查 Pod 标签
kubectl get pods -l app=my-app

# 检查 Pod 就绪状态
kubectl get pods -l app=my-app
```

### DNS 解析失败

```bash
# 检查 DNS
kubectl exec -it <pod-name> -- nslookup my-service

# 检查 CoreDNS
kubectl get pods -n kube-system -l k8s-app=kube-dns
```

## 面试回答

**问题**: 什么是 Service？

**回答**: Service 是 Kubernetes 中实现服务发现的核心组件，它为一组 Pod 提供稳定的访问入口。Service 通过标签选择器关联后端 Pod，并提供负载均衡功能。

Service 有四种类型：**ClusterIP** 是默认类型，只在集群内部可访问，适用于内部服务；**NodePort** 在每个节点上开放一个端口，可以从集群外部访问；**LoadBalancer** 使用云厂商的负载均衡器，适用于对外暴露服务；**ExternalName** 映射到外部域名，用于访问外部服务。

Service 的工作原理是：客户端通过 Service 的 ClusterIP 或域名访问服务，kube-proxy 监听 Service 和 Endpoints 变化，在节点上配置 iptables 或 IPVS 规则，将流量负载均衡到后端 Pod。Endpoints 维护后端 Pod 的 IP 地址列表，根据 Pod 的就绪状态动态更新。

Headless Service（clusterIP: None）不分配 ClusterIP，DNS 解析返回所有 Pod IP，适用于 StatefulSet 和需要直接访问特定 Pod 的场景。

生产环境最佳实践：配置健康检查确保只有就绪的 Pod 接收流量，使用 NetworkPolicy 控制访问，配合 Ingress 实现 HTTP 路由，使用会话保持支持有状态应用。

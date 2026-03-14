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
  - Endpoint
  - kube-proxy
---

# Kubernetes Service、Endpoint、kube-proxy 关系详解

## 引言

在 Kubernetes 集群中，Service、Endpoint 和 kube-proxy 是实现服务发现和负载均衡的三个核心组件。它们协同工作，将客户端请求路由到正确的后端 Pod。理解这三个组件之间的关系和工作机制，对于深入理解 Kubernetes 网络模型至关重要。

本文将深入剖析 Service、Endpoint 和 kube-proxy 三者之间的关系、工作原理和协作机制。

## 三者关系概述

### 核心关系图

```
┌─────────────────────────────────────────────────────────────┐
│           Service、Endpoint、kube-proxy 关系                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   Service                            │   │
│  │  - 定义服务抽象                                      │   │
│  │  - 提供稳定的 ClusterIP                              │   │
│  │  - 通过 selector 选择后端 Pod                        │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         │ 关联                              │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   Endpoints                          │   │
│  │  - 维护后端 Pod IP 列表                              │   │
│  │  - 自动更新 Pod 变化                                  │   │
│  │  - 提供健康检查状态                                   │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         │ 监听                              │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   kube-proxy                         │   │
│  │  - 监听 Service 和 Endpoints 变化                    │   │
│  │  - 同步 iptables/IPVS 规则                           │   │
│  │  - 实现负载均衡                                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 各组件职责

| 组件 | 职责 | 类型 |
|-----|------|------|
| **Service** | 定义服务抽象，提供稳定访问入口 | API 对象 |
| **Endpoints** | 维护后端 Pod IP 列表 | API 对象 |
| **kube-proxy** | 实现负载均衡和流量转发 | 节点组件 |

## Service 详解

### Service 的作用

Service 是 Kubernetes 中定义服务抽象的 API 对象，它提供了一种将一组 Pod 暴露为网络服务的方法。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
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

### Service 关键字段

| 字段 | 说明 |
|-----|------|
| **selector** | 选择后端 Pod 的标签选择器 |
| **ports** | 服务端口配置 |
| **type** | Service 类型 |
| **clusterIP** | 虚拟 IP 地址 |

### Service 类型

```yaml
# ClusterIP（默认）
spec:
  type: ClusterIP
  
# NodePort
spec:
  type: NodePort
  ports:
  - port: 80
    nodePort: 30080

# LoadBalancer
spec:
  type: LoadBalancer

# ExternalName
spec:
  type: ExternalName
  externalName: external.example.com
```

## Endpoints 详解

### Endpoints 的作用

Endpoints 维护 Service 对应的后端 Pod IP 地址列表。当 Service 定义了 selector 时，Endpoints Controller 会自动创建同名的 Endpoints 对象。

```yaml
apiVersion: v1
kind: Endpoints
metadata:
  name: my-service
  namespace: default
subsets:
- addresses:
  - ip: 10.244.1.5
    nodeName: node-1
    targetRef:
      kind: Pod
      name: my-app-pod-1
      namespace: default
  - ip: 10.244.1.6
    nodeName: node-2
    targetRef:
      kind: Pod
      name: my-app-pod-2
      namespace: default
  ports:
  - name: http
    port: 8080
    protocol: TCP
```

### Endpoints 结构

```
┌─────────────────────────────────────────────────────────────┐
│                    Endpoints 结构                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  subsets:                                                   │
│  - addresses:      # 就绪的 Pod                             │
│    - ip: 10.244.1.5                                        │
│    - ip: 10.244.1.6                                        │
│    ports:                                                   │
│    - port: 8080                                            │
│                                                              │
│  - notReadyAddresses:  # 未就绪的 Pod                       │
│    - ip: 10.244.1.7                                        │
│    ports:                                                   │
│    - port: 8080                                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### EndpointSlice

EndpointSlice 是 Endpoints 的替代方案，提供更好的扩展性：

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
  nodeName: node-1
- addresses:
  - 10.244.1.6
  conditions:
    ready: true
  nodeName: node-2
```

### Endpoints vs EndpointSlice

| 特性 | Endpoints | EndpointSlice |
|-----|-----------|---------------|
| **扩展性** | 单对象限制 | 多对象支持 |
| **性能** | 大规模性能差 | 性能好 |
| **功能** | 基础功能 | 支持更多特性 |
| **状态** | 稳定 | 推荐使用 |

## kube-proxy 详解

### kube-proxy 的作用

kube-proxy 是运行在每个节点上的网络代理，负责实现 Service 的负载均衡和流量转发。

```
┌─────────────────────────────────────────────────────────────┐
│                    kube-proxy 工作原理                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              API Server                              │   │
│  │  - Service 定义                                      │   │
│  │  - Endpoints 数据                                    │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         │ Watch                             │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              kube-proxy                              │   │
│  │  - 监听 Service/Endpoints 变化                       │   │
│  │  - 生成负载均衡规则                                  │   │
│  │  - 同步到节点网络                                    │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │                                   │
│                         │ 同步                              │
│                         ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              iptables/IPVS                           │   │
│  │  - NAT 规则                                          │   │
│  │  - 负载均衡规则                                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### kube-proxy 模式

#### iptables 模式（默认）

```bash
# 查看 iptables 规则
iptables -L KUBE-SERVICES -t nat -n

# 规则结构
KUBE-SERVICES
├── KUBE-SVC-XXX (Service 入口)
│   ├── KUBE-SEP-XXX1 (后端 Pod 1)
│   ├── KUBE-SEP-XXX2 (后端 Pod 2)
│   └── KUBE-SEP-XXX3 (后端 Pod 3)
```

#### IPVS 模式

```bash
# 查看 IPVS 规则
ipvsadm -L -n

# 支持的负载均衡算法
# - rr: 轮询
# - wrr: 加权轮询
# - lc: 最少连接
# - wlc: 加权最少连接
# - sh: 源地址哈希
```

#### userspace 模式（已废弃）

通过用户空间代理转发流量，性能较差。

### 模式对比

| 特性 | iptables | IPVS | userspace |
|-----|---------|------|-----------|
| **性能** | O(n) | O(1) | 差 |
| **负载均衡** | 随机 | 多种算法 | 轮询 |
| **扩展性** | 中等 | 高 | 低 |
| **复杂度** | 简单 | 中等 | 简单 |
| **状态** | 默认 | 推荐 | 废弃 |

## 三者协作流程

### Pod 创建时的流程

```
┌─────────────────────────────────────────────────────────────┐
│                  Pod 创建时的协作流程                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 创建 Pod                                                 │
│     ┌─────────────────────────────────────────────────┐    │
│     │ kubectl create -f pod.yaml                      │    │
│     └──────────────────────┬──────────────────────────┘    │
│                            │                                │
│                            ▼                                │
│  2. Pod Controller 创建 Pod                                │
│     ┌─────────────────────────────────────────────────┐    │
│     │ - 分配 IP 地址                                   │    │
│     │ - 调度到节点                                     │    │
│     │ - 启动容器                                       │    │
│     └──────────────────────┬──────────────────────────┘    │
│                            │                                │
│                            ▼                                │
│  3. Endpoints Controller 更新 Endpoints                    │
│     ┌─────────────────────────────────────────────────┐    │
│     │ - 检测 Pod 标签匹配                              │    │
│     │ - 检测 Pod 就绪状态                              │    │
│     │ - 更新 Endpoints 对象                            │    │
│     └──────────────────────┬──────────────────────────┘    │
│                            │                                │
│                            ▼                                │
│  4. kube-proxy 同步规则                                    │
│     ┌─────────────────────────────────────────────────┐    │
│     │ - 监听 Endpoints 变化                            │    │
│     │ - 更新 iptables/IPVS 规则                        │    │
│     │ - 添加新 Pod 到负载均衡池                        │    │
│     └─────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 客户端访问流程

```
┌─────────────────────────────────────────────────────────────┐
│                  客户端访问流程                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 客户端 Pod 发起请求                                     │
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
│     │ 响应直接返回客户端（不经过 Service）            │    │
│     └─────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 详细配置示例

### 完整配置

```yaml
# 1. Deployment
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
      - name: app
        image: nginx:1.21
        ports:
        - containerPort: 8080
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
---
# 2. Service
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
# 3. Endpoints（自动创建，无需手动配置）
# kubectl get endpoints my-service
```

### 查看 Endpoints

```bash
# 查看 Endpoints
kubectl get endpoints my-service

# 查看详细信息
kubectl describe endpoints my-service

# 查看 EndpointSlice
kubectl get endpointslices -l kubernetes.io/service-name=my-service
```

### 查看 kube-proxy 状态

```bash
# 查看 kube-proxy Pod
kubectl get pods -n kube-system -l k8s-app=kube-proxy

# 查看 kube-proxy 日志
kubectl logs -n kube-system <kube-proxy-pod>

# 查看 kube-proxy 配置
kubectl get configmap kube-proxy -n kube-system -o yaml
```

## 常见问题排查

### Endpoints 为空

```bash
# 检查 Pod 标签
kubectl get pods -l app=my-app --show-labels

# 检查 Service selector
kubectl get svc my-service -o jsonpath='{.spec.selector}'

# 检查 Pod 就绪状态
kubectl get pods -l app=my-app
```

### 无法访问 Service

```bash
# 检查 Service ClusterIP
kubectl get svc my-service

# 检查 iptables 规则
iptables -L KUBE-SERVICES -t nat -n

# 检查 IPVS 规则
ipvsadm -L -n

# 检查 kube-proxy 状态
kubectl get pods -n kube-system -l k8s-app=kube-proxy
```

### 负载均衡不均匀

```bash
# 检查 Endpoints 数量
kubectl get endpoints my-service -o jsonpath='{.subsets[0].addresses}'

# 检查 kube-proxy 模式
kubectl get configmap kube-proxy -n kube-system -o jsonpath='{.data.config}'

# 考虑使用 IPVS 模式
```

## 面试回答

**问题**: Service、Endpoint、kube-proxy 三者的关系是什么？

**回答**: Service、Endpoint 和 kube-proxy 是 Kubernetes 实现服务发现和负载均衡的三个核心组件，它们协同工作将客户端请求路由到正确的后端 Pod。

**Service** 是定义服务抽象的 API 对象，它通过标签选择器关联后端 Pod，并提供一个稳定的 ClusterIP 作为访问入口。Service 本身不直接处理流量，而是定义了服务的抽象。

**Endpoints** 是维护后端 Pod IP 地址列表的 API 对象。当 Service 定义了 selector 时，Endpoints Controller 会自动创建同名的 Endpoints 对象，并根据 Pod 的就绪状态动态更新地址列表。Endpoints 将 Service 的抽象定义与具体的 Pod 实例关联起来。

**kube-proxy** 是运行在每个节点上的网络代理组件。它监听 API Server 中 Service 和 Endpoints 的变化，并在节点上同步 iptables 或 IPVS 规则。当客户端访问 Service 的 ClusterIP 时，kube-proxy 配置的规则会将流量负载均衡到后端 Pod。

三者的协作流程是：Service 定义服务抽象，Endpoints 维护后端 Pod 列表，kube-proxy 根据这些信息在节点上配置负载均衡规则。当 Pod 创建或删除时，Endpoints 自动更新，kube-proxy 同步更新规则，确保流量正确路由。现代 Kubernetes 推荐使用 EndpointSlice 替代 Endpoints，提供更好的扩展性和性能。

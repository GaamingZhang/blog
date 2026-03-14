---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Service
---

# Kubernetes Service类型详解

## 什么是Service？

在Kubernetes中，Pod是临时的，它们会被创建、销毁、重新调度。每个Pod都有自己的IP地址，但这个IP地址是动态的，不可靠。

Service提供了一个稳定的访问入口，无论后端Pod如何变化，客户端都可以通过Service访问应用。

```
┌─────────────────────────────────────────────────────────────┐
│                         Service                              │
│                                                              │
│  稳定的ClusterIP: 10.96.0.100                               │
│  稳定的DNS名称: my-service.default.svc.cluster.local        │
│  稳定的端口: 80                                              │
│                                                              │
│                      ↓ 负载均衡                              │
│         ┌────────────┼────────────┐                         │
│         ↓            ↓            ↓                         │
│    ┌─────────┐  ┌─────────┐  ┌─────────┐                   │
│    │  Pod-1  │  │  Pod-2  │  │  Pod-3  │                   │
│    │ 10.1.1.1│  │ 10.1.1.2│  │ 10.1.1.3│                   │
│    └─────────┘  └─────────┘  └─────────┘                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Service的四种类型

Kubernetes提供了四种Service类型：

| 类型 | 说明 | 使用场景 |
|------|------|----------|
| ClusterIP | 集群内部访问 | 内部服务通信 |
| NodePort | 通过节点端口暴露 | 开发测试、内部服务 |
| LoadBalancer | 云厂商负载均衡器 | 生产环境对外暴露 |
| ExternalName | DNS别名 | 访问外部服务 |

### 1. ClusterIP（默认类型）

ClusterIP类型的Service只在集群内部可访问。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 8080
```

**特点**：
- 分配一个集群内部的虚拟IP（ClusterIP）
- 只能在集群内部访问
- 默认类型，如果不指定type就是ClusterIP

**访问方式**：
```bash
# 在集群内部
curl http://my-service:80
curl http://my-service.default.svc.cluster.local:80
curl http://10.96.0.100:80
```

**适用场景**：
- 内部服务间通信
- 数据库、缓存等不需要外部访问的服务
- 微服务架构中的内部服务

### 2. NodePort

NodePort类型的Service通过每个节点的特定端口暴露服务。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: NodePort
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30080
```

**特点**：
- 在每个节点上开放一个端口（默认30000-32767）
- 可以通过任意节点的IP:端口访问
- 同时也会分配ClusterIP

**访问方式**：
```bash
# 从集群外部
curl http://<node-ip>:30080

# 从集群内部（仍然可用）
curl http://my-service:80
```

**端口范围**：
- 默认范围：30000-32767
- 可以通过API Server参数修改：`--service-node-port-range=30000-32767`

**适用场景**：
- 开发测试环境
- 内部服务需要临时外部访问
- 没有云负载均衡器的环境

### 3. LoadBalancer

LoadBalancer类型的Service会请求云厂商创建负载均衡器。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
spec:
  type: LoadBalancer
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 8080
```

**特点**：
- 会创建云厂商的负载均衡器（如AWS ALB/NLB、GCP Load Balancer）
- 自动分配外部IP
- 同时包含NodePort和ClusterIP的功能

**访问方式**：
```bash
# 通过负载均衡器的外部IP
curl http://<external-ip>:80
```

**云厂商特定注解**：

```yaml
# AWS
service.beta.kubernetes.io/aws-load-balancer-type: nlb
service.beta.kubernetes.io/aws-load-balancer-internal: "true"

# GCP
cloud.google.com/load-balancer-type: "Internal"

# Azure
service.beta.kubernetes.io/azure-load-balancer-internal: "true"
```

**适用场景**：
- 生产环境对外暴露服务
- 需要高可用的外部访问
- 云环境部署

### 4. ExternalName

ExternalName类型的Service将服务映射到外部DNS名称。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-service
spec:
  type: ExternalName
  externalName: external.example.com
```

**特点**：
- 不创建ClusterIP
- 不创建Endpoints
- 只返回CNAME记录

**访问方式**：
```bash
# 在集群内部
curl http://external-service:80
# 实际访问的是 external.example.com:80
```

**适用场景**：
- 访问集群外部的服务
- 服务迁移过渡期
- 统一服务访问入口

## Service的工作原理

### Endpoints

Service通过Endpoints关联后端Pod：

```yaml
apiVersion: v1
kind: Endpoints
metadata:
  name: my-service
subsets:
- addresses:
  - ip: 10.1.1.1
    nodeName: node1
    targetRef:
      name: nginx-pod-1
  - ip: 10.1.1.2
    nodeName: node2
    targetRef:
      name: nginx-pod-2
  ports:
  - port: 8080
    protocol: TCP
```

**自动创建**：当Service有selector时，Endpoints Controller会自动创建Endpoints。

**手动创建**：当Service没有selector时，需要手动创建Endpoints。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-db
spec:
  ports:
  - port: 3306
---
apiVersion: v1
kind: Endpoints
metadata:
  name: external-db
subsets:
- addresses:
  - ip: 192.168.1.100
  ports:
  - port: 3306
```

### kube-proxy

kube-proxy负责实现Service的负载均衡：

**三种模式**：

| 模式 | 说明 | 性能 |
|------|------|------|
| userspace | 用户空间代理 | 低 |
| iptables | iptables规则 | 中 |
| ipvs | IPVS负载均衡 | 高 |

**iptables模式**：

```bash
iptables -t nat -L KUBE-SERVICES

Chain KUBE-SERVICES (2 references)
target     prot opt source     destination
KUBE-SVC-XXX  tcp  --  anywhere  10.96.0.100  /* my-service */
```

**ipvs模式**：

```bash
ipvsadm -Ln

TCP  10.96.0.100:80 rr
  -> 10.1.1.1:8080          Masq    1      0          0
  -> 10.1.1.2:8080          Masq    1      0          0
```

## Service配置详解

### 端口配置

```yaml
ports:
- name: http
  port: 80
  targetPort: 8080
  protocol: TCP
- name: https
  port: 443
  targetPort: 8443
  protocol: TCP
  nodePort: 30443
```

**参数说明**：
- `port`：Service监听的端口
- `targetPort`：容器端口
- `nodePort`：节点端口（NodePort类型）
- `protocol`：协议，TCP或UDP

### 选择器

```yaml
selector:
  app: nginx
  version: v1
```

### 会话亲和性

```yaml
spec:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800
```

**作用**：同一客户端的请求始终路由到同一个Pod。

### 外部IP

```yaml
spec:
  type: ClusterIP
  externalIPs:
  - 192.168.1.100
```

**作用**：允许通过指定IP访问Service。

### 外部流量策略

```yaml
spec:
  type: NodePort
  externalTrafficPolicy: Local
```

**两种策略**：
- `Cluster`（默认）：流量可能转发到其他节点
- `Local`：只转发到本节点的Pod

**Local模式优点**：
- 保留客户端源IP
- 减少跨节点流量

**Local模式缺点**：
- 如果本节点没有Pod，请求会失败

## Headless Service

Headless Service是一种特殊的Service，不分配ClusterIP。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: headless-service
spec:
  clusterIP: None
  selector:
    app: nginx
  ports:
  - port: 80
```

**特点**：
- 不分配ClusterIP
- DNS返回所有Pod的IP
- 客户端需要自己做负载均衡

**DNS解析**：
```bash
nslookup headless-service.default.svc.cluster.local

Name:    headless-service.default.svc.cluster.local
Address: 10.1.1.1
Address: 10.1.1.2
Address: 10.1.1.3
```

**适用场景**：
- StatefulSet
- 需要直接访问Pod的场景
- 客户端自己做服务发现

## 多端口Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: multi-port-service
spec:
  selector:
    app: nginx
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: https
    port: 443
    targetPort: 8443
```

**注意**：多端口时必须为每个端口指定name。

## 常见问题

### Q1: Service无法访问怎么办？

```bash
# 检查Endpoints
kubectl get endpoints <service-name>

# 检查Pod标签
kubectl get pods --show-labels

# 检查Service配置
kubectl describe service <service-name>
```

### Q2: 如何保留客户端源IP？

```yaml
spec:
  type: NodePort
  externalTrafficPolicy: Local
```

### Q3: 如何访问外部服务？

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-service
spec:
  type: ExternalName
  externalName: external.example.com
```

### Q4: NodePort端口冲突怎么办？

```bash
# 查看已使用的端口
kubectl get svc --all-namespaces -o jsonpath='{.items[*].spec.ports[*].nodePort}'

# 修改端口范围
# API Server参数: --service-node-port-range=30000-32767
```

## 最佳实践

1. **命名规范**：Service名称应清晰表达其用途
2. **标签选择器**：使用明确的标签选择器
3. **健康检查**：配置readinessProbe确保只有健康的Pod在Endpoints中
4. **会话亲和性**：有状态服务使用会话亲和性
5. **外部流量策略**：需要源IP时使用Local模式
6. **端口命名**：多端口Service必须命名

## 参考资源

- [Service官方文档](https://kubernetes.io/docs/concepts/services-networking/service/)
- [Service类型](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types)

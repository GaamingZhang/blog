---
date: 2026-01-24
author: Gaaming Zhang
isOriginal: true
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# Service：Pod的稳定访问入口

## 为什么需要Service？

在理解Service之前，我们需要先认识Pod的一个"致命缺陷"：**Pod的IP地址是不稳定的**。

想象一下这个场景：你的前端应用需要调用后端API，后端运行在一个Pod里，IP是`10.244.1.5`。一切正常，直到这个Pod因为某种原因重启了——重启后它可能变成了`10.244.2.8`。你的前端应用还傻傻地往`10.244.1.5`发请求，然后得到连接超时。

更糟糕的是，如果你的后端有3个Pod副本来分担负载，前端难道要自己维护一份Pod IP列表，还要自己实现负载均衡？这显然不现实。

**Service就是为了解决这个问题而生的**。它提供：
- 一个稳定不变的IP地址（ClusterIP）
- 一个稳定的DNS名称
- 自动的负载均衡
- 自动的后端Pod发现

你可以把Service想象成一个"虚拟的前台"：不管后面的员工怎么换，前台的电话号码永远不变。客户只需要拨打前台电话，前台会自动帮你转接到合适的员工。

## Service的工作原理

### 核心机制

Service通过**标签选择器（Label Selector）**来发现后端Pod。它不关心Pod具体叫什么、IP是多少，只看标签匹配不匹配。

```
                    ┌─────────────────┐
                    │    Service      │
                    │  ClusterIP:     │
                    │  10.96.100.1    │
                    │  selector:      │
                    │    app: api     │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
        ┌─────────┐    ┌─────────┐    ┌─────────┐
        │  Pod 1  │    │  Pod 2  │    │  Pod 3  │
        │ app:api │    │ app:api │    │ app:api │
        │10.244.1.5    │10.244.2.8    │10.244.3.2
        └─────────┘    └─────────┘    └─────────┘
```

当你访问Service的ClusterIP时，kube-proxy会将请求转发到某一个后端Pod。这个转发过程对调用方完全透明。

### Endpoints：Service与Pod的桥梁

你可能会好奇：Service是怎么知道有哪些Pod可以转发的？

答案是**Endpoints**（或新版的EndpointSlice）。Kubernetes会自动创建一个与Service同名的Endpoints对象，里面记录着所有匹配标签的、处于Ready状态的Pod IP。

当Pod被创建、删除、或健康状态变化时，Endpoints会自动更新。这就是Service"自动发现"的秘密。

## Service的四种类型

### ClusterIP（默认类型）

这是最常用的类型。它会分配一个集群内部的虚拟IP，只能在集群内部访问。

**适用场景**：集群内部服务间的通信（如前端调用后端API）

```yaml
apiVersion: v1
kind: Service
metadata:
  name: api-service
spec:
  type: ClusterIP  # 可以省略，默认就是ClusterIP
  selector:
    app: api
  ports:
    - port: 80         # Service暴露的端口
      targetPort: 8080  # Pod实际监听的端口
```

创建后，集群内任何Pod都可以通过`api-service:80`或`api-service.default.svc.cluster.local:80`访问这个服务。

### NodePort：从集群外部访问

NodePort会在**每个节点**上开放一个端口（默认范围30000-32767），外部流量通过这个端口进入集群。

```
    外部用户
        │
        │ 访问任意节点IP:30080
        ▼
  ┌───────────────────────────────────────┐
  │          Kubernetes集群               │
  │  ┌─────────┐  ┌─────────┐  ┌─────────┐
  │  │ Node 1  │  │ Node 2  │  │ Node 3  │
  │  │ :30080  │  │ :30080  │  │ :30080  │
  │  └────┬────┘  └────┬────┘  └────┬────┘
  │       └───────────┬────────────┘      │
  │                   ▼                   │
  │             ┌──────────┐              │
  │             │ Service  │              │
  │             └────┬─────┘              │
  │                  │                    │
  │       ┌──────────┼──────────┐         │
  │       ▼          ▼          ▼         │
  │    Pod 1      Pod 2      Pod 3        │
  └───────────────────────────────────────┘
```

**适用场景**：开发测试环境快速暴露服务、没有云负载均衡器的环境

**缺点**：
- 端口范围受限（30000-32767）
- 需要知道节点IP
- 没有真正的负载均衡（需要外部LB）
- 每个节点都开放端口，有一定安全风险

### LoadBalancer：生产环境首选

LoadBalancer类型会自动调用云提供商的API，创建一个外部负载均衡器（如AWS ELB、GCP Load Balancer），并获得一个公网IP。

**适用场景**：云环境中的生产服务

**优点**：
- 自动获得外部可访问的IP
- 云提供商保障高可用
- 真正的负载均衡

**缺点**：
- 需要云环境支持
- 每个Service一个负载均衡器，成本较高
- 裸金属集群需要额外方案（如MetalLB）

### ExternalName：访问集群外部服务

ExternalName类型比较特殊，它不代理任何Pod，而是返回一个CNAME记录，用于访问集群外部的服务。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-db
spec:
  type: ExternalName
  externalName: database.example.com
```

**适用场景**：在集群内用统一的方式访问外部数据库、第三方API等

## 关于端口的理解

Service的端口配置经常让初学者困惑，这里详细解释：

```yaml
ports:
  - port: 80         # Service端口：客户端访问Service时使用的端口
    targetPort: 8080  # Pod端口：请求最终转发到Pod的哪个端口
    nodePort: 30080   # 节点端口：NodePort类型时，节点上开放的端口
```

**访问路径**：
- ClusterIP：`service-ip:80` → Pod的8080端口
- NodePort：`node-ip:30080` → Service的80端口 → Pod的8080端口

**小技巧**：`targetPort`可以是数字，也可以是Pod定义中端口的名称。用名称的好处是，当Pod的端口号变化时，不需要修改Service。

## 服务发现与DNS

Kubernetes集群内置了DNS服务（CoreDNS），会为每个Service自动创建DNS记录。

### DNS命名规则

Service的完整DNS名称格式是：
```
<service-name>.<namespace>.svc.cluster.local
```

比如`default`命名空间下的`api-service`，完整域名是：
```
api-service.default.svc.cluster.local
```

### 简写规则

好消息是，大多数情况下不需要写这么长：

| 场景 | 可以使用的名称 |
|------|---------------|
| 同一Namespace | `api-service` |
| 跨Namespace | `api-service.other-namespace` |
| 完整写法 | `api-service.other-namespace.svc.cluster.local` |

同一Namespace内，直接用服务名就够了，Kubernetes会自动补全。

## 会话保持（Session Affinity）

默认情况下，Service会将请求随机分发到后端Pod。但有些场景需要"会话保持"——同一个客户端的请求总是发到同一个Pod。

典型场景：
- 应用在内存中存储了用户会话
- WebSocket长连接
- 某些有状态的API

```yaml
spec:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 3600  # 会话保持1小时
```

启用后，来自同一客户端IP的请求会在指定时间内被转发到同一个Pod。

**注意**：如果你的应用设计良好（无状态），尽量不要依赖会话保持。它会导致负载不均衡，也不利于Pod的弹性伸缩。

## 无头服务（Headless Service）

有时候你不需要Service的负载均衡功能，而是想直接获取所有后端Pod的IP地址。这时候可以使用**无头服务**。

创建方法很简单：把ClusterIP设为`None`。

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

查询这个Service的DNS时，不会返回ClusterIP，而是返回所有后端Pod的IP地址（A记录）。

**典型使用场景**：
- StatefulSet：每个Pod需要稳定的网络标识
- 客户端需要自己实现负载均衡逻辑
- 某些数据库集群需要知道所有节点地址

## externalTrafficPolicy：保留客户端IP

当使用NodePort或LoadBalancer时，有个问题：流量可能经过多次转发，原始的客户端IP会丢失。

**两种模式**：

**Cluster模式（默认）**：
- 流量可以转发到任意节点的Pod
- 负载更均衡
- 但会丢失客户端源IP

**Local模式**：
- 流量只转发到接收请求那个节点上的Pod
- 保留客户端源IP
- 如果该节点没有Pod，请求会失败

```yaml
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local  # 保留源IP
```

**选择建议**：
- 需要获取客户端真实IP（如审计、限流）→ 用Local
- 追求负载均衡、对源IP无要求 → 用Cluster

## 常见问题

### Q1: Service访问不通怎么排查？

**排查步骤**：

1. **检查Service是否存在**
   ```bash
   kubectl get svc <service-name>
   ```

2. **检查Endpoints是否有后端Pod**
   ```bash
   kubectl get endpoints <service-name>
   ```
   如果Endpoints为空，说明没有Pod匹配Service的selector，或者Pod没有处于Ready状态。

3. **检查selector是否匹配**
   对比Service的selector和Pod的labels是否一致。

4. **检查Pod是否Ready**
   ```bash
   kubectl get pods -l <selector>
   ```

5. **从Pod内部测试**
   ```bash
   kubectl exec -it <some-pod> -- curl <service-name>:<port>
   ```

### Q2: Service和Ingress有什么区别？

| 特性 | Service | Ingress |
|------|---------|---------|
| 工作层级 | L4（TCP/UDP） | L7（HTTP/HTTPS） |
| 负载均衡 | 基于IP | 基于URL路径/域名 |
| TLS终止 | 不支持 | 支持 |
| 使用场景 | 集群内通信、简单外部访问 | Web应用、API网关 |

**简单理解**：
- Service是基础设施，解决"怎么找到Pod"的问题
- Ingress是上层路由，解决"怎么优雅地暴露HTTP服务"的问题

### Q3: 为什么Endpoints是空的？

**常见原因**：

1. **selector不匹配**：Service的selector和Pod的labels对不上
2. **Pod不是Ready状态**：可能探针失败或容器未启动
3. **Pod在不同Namespace**：Service默认只能选择同Namespace的Pod

### Q4: NodePort端口号可以自定义吗？

可以，但有限制：
- 默认范围是30000-32767
- 可以通过API Server参数修改范围
- 同一集群内NodePort不能重复

### Q5: 如何实现蓝绿部署或金丝雀发布？

Service通过selector选择Pod，你可以利用这一点：

**蓝绿部署**：
1. 部署新版本Pod，使用不同的版本标签
2. 修改Service的selector，一次性切换到新版本

**金丝雀发布**：
1. 保持selector不变
2. 部署少量新版本Pod（比如10%）
3. 由于Service随机分发，约10%流量会到新版本
4. 逐步增加新版本Pod比例

更精细的流量控制需要使用Ingress或服务网格（如Istio）。

## 小结

Service是Kubernetes网络模型的核心组件：

- **解决的问题**：Pod IP不稳定、需要负载均衡、需要服务发现
- **工作原理**：通过selector发现Pod，通过Endpoints跟踪Pod IP变化
- **四种类型**：ClusterIP（内部访问）、NodePort（节点端口）、LoadBalancer（云负载均衡）、ExternalName（外部服务）
- **服务发现**：内置DNS，通过服务名直接访问

**关键要点**：
- ClusterIP满足90%的内部通信需求
- 生产环境对外服务推荐LoadBalancer + Ingress组合
- 理解Endpoints对排查问题很有帮助
- 无状态设计优于会话保持

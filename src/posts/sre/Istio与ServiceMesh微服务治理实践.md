---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - SRE
tag:
  - Istio
  - Service Mesh
  - Kubernetes
  - 微服务
---

# Istio 与 Service Mesh：微服务流量治理实践

## 从一个真实困境说起

设想这样一个场景：你的团队维护着十几个微服务，每个服务用不同语言编写——Java、Go、Python 混杂。某天你被要求为所有服务间的通信开启 mTLS，同时对高优先级接口实施熔断保护，并为灰度发布提供流量染色能力。

如果走 SDK 路线，这意味着每个团队都要集成你选定的治理库，Java 服务用 Spring Cloud，Go 服务另找对应实现，Python 服务再找一套。不同语言的行为是否一致？SDK 升级如何推动十几个团队同步？这种"将治理能力嵌入业务代码"的方式，让基础设施能力和业务逻辑深度耦合，是微服务规模化后最棘手的问题之一。

Service Mesh 的回答是：**将这些能力彻底下沉到基础设施层，业务代码完全无感知**。

## 为什么需要 Service Mesh

### 微服务治理的本质矛盾

微服务架构的治理需求——熔断、重试、超时、限流、追踪、mTLS——无论用什么语言实现，它们的核心逻辑是同质的。让每个服务团队各自实现一遍，本质上是在重复造轮子，并且由于实现质量和版本不一致，实际行为会产生偏差。

更深层的矛盾在于：治理策略本来应该是**运维关心的事**，却不得不通过修改业务代码来实现。上线一个新的熔断规则需要发布应用版本，这在高频迭代的团队里代价极高。

### Sidecar 模式：代理所有流量

Service Mesh 的核心机制是 Sidecar 模式。每个应用 Pod 旁边注入一个代理进程（Envoy），这个代理**拦截该 Pod 所有进出流量**，在代理层面实施治理策略。

```
┌─────────────────── Pod ────────────────────┐
│                                            │
│  ┌──────────────┐    ┌──────────────────┐  │
│  │  业务容器    │◄──►│  Envoy Sidecar   │  │
│  │  (应用代码)  │    │  (流量代理)      │  │
│  └──────────────┘    └──────┬───────────┘  │
│                             │              │
└─────────────────────────────│──────────────┘
                              │ 所有进出流量
                              ▼
                       网络（其他服务）
```

从业务代码视角看，它只是在和 `localhost` 通信。流量被 iptables 规则透明拦截到 Envoy，再由 Envoy 根据控制面下发的策略转发。**业务代码无需任何改动，治理能力即可生效**。

### 与 SDK 方案的本质区别

| 维度 | SDK 方案（Spring Cloud / Sentinel） | Service Mesh（Istio） |
|------|-------------------------------------|-----------------------|
| 耦合度 | 与业务代码耦合，需要引用依赖包 | 完全解耦，基础设施层实现 |
| 多语言支持 | 每种语言需要独立 SDK | 语言无关，所有语言统一治理 |
| 策略变更 | 需要修改代码并重新发布 | 运行时动态下发，无需发布 |
| 可观测性 | 需要业务代码主动埋点 | 自动采集所有服务间流量指标 |
| 运维复杂度 | SDK 升级需要推动各团队 | 统一升级代理版本即可 |

:::tip
SDK 方案并非一无是处。对于单语言、小规模团队，Spring Cloud 生态的成熟度和开发调试体验可能更好。Service Mesh 的优势在规模化（10+ 服务、多语言）和对治理能力有高度运维自主性要求的场景下才充分体现。
:::

## Istio 架构与组件

Istio 分为两个平面：**数据面**负责实际转发流量，**控制面**负责下发配置和管理证书。

```
┌─────────────────────────────────────────────────────────┐
│                      控制面（Istiod）                    │
│                                                         │
│   ┌──────────┐    ┌──────────┐    ┌──────────────────┐  │
│   │  Pilot   │    │ Citadel  │    │     Galley       │  │
│   │（配置分发）│    │（证书CA）│    │（配置验证/注入）  │  │
│   └──────────┘    └──────────┘    └──────────────────┘  │
│         │               │                               │
└─────────│───────────────│───────────────────────────────┘
          │ xDS API        │ mTLS 证书
          ▼                ▼
┌─────────────────────────────────────────────────────────┐
│                      数据面                              │
│                                                         │
│  ┌────────────────────┐    ┌────────────────────┐       │
│  │ Pod A               │    │ Pod B               │      │
│  │ ┌──────┐ ┌───────┐ │    │ ┌──────┐ ┌───────┐ │      │
│  │ │ App  │ │Envoy  │ │───►│ │ App  │ │Envoy  │ │      │
│  │ └──────┘ └───────┘ │    │ └──────┘ └───────┘ │      │
│  └────────────────────┘    └────────────────────┘       │
└─────────────────────────────────────────────────────────┘
```

**Istiod** 是 Istio 1.5 之后将原来分散的 Pilot、Citadel、Galley 三个组件合并后的单体控制面进程，简化了部署和运维。

- **Pilot**：读取 Istio 的 CR（VirtualService、DestinationRule 等），将其转换为 Envoy 能理解的 **xDS 配置**（Endpoint Discovery Service、Cluster Discovery Service、Route Discovery Service 等），通过 gRPC 推送到每个 Envoy 实例。
- **Citadel**：扮演 CA（Certificate Authority）角色，为每个服务账号（Service Account）签发 X.509 证书，并负责证书的自动轮换。mTLS 的信任根就在这里。
- **Galley**：负责验证 Istio 配置的合法性，并在 Sidecar 注入时将 Envoy 配置注入到 Pod 中。

**Ingress Gateway / Egress Gateway** 是特殊的 Envoy 实例，以独立 Pod 形式运行，负责处理集群南北向流量（外部到集群、集群到外部）。

## Sidecar 注入原理

理解注入原理，才能在 Pod 没有被注入时有效排查问题。

### MutatingWebhookConfiguration 拦截

Sidecar 注入的入口是 Kubernetes 的准入控制机制——**MutatingWebhookConfiguration**。当你在 Namespace 上打上 `istio-injection=enabled` 标签，Istio 会向 API Server 注册一个 Webhook。

流程如下：

```
kubectl apply (Pod 创建请求)
        │
        ▼
  API Server 收到请求
        │
        ▼
  MutatingAdmissionWebhook
  调用 Istiod 的 Webhook 端点
        │
        ▼
  Istiod 修改 Pod 定义
  注入 istio-init (Init Container)
  注入 istio-proxy (Envoy Sidecar)
        │
        ▼
  修改后的 Pod 被调度运行
```

### iptables 流量劫持

注入的 Init Container `istio-init` 在 Pod 启动时执行 iptables 规则配置，这是 Sidecar 透明代理的关键：

```
所有出站流量  →  iptables OUTPUT 规则  →  重定向到 15001 端口（Envoy）
所有入站流量  →  iptables PREROUTING 规则  →  重定向到 15006 端口（Envoy）
```

Envoy 自身的流量被 iptables 用 UID 规则豁免（Envoy 以固定 UID 1337 运行），防止形成死循环。

业务容器感知到的是"自己在直接连目标服务"，实际上请求已经被透明地经过了 Envoy 的处理：

```
业务容器 → 发出 TCP 连接到目标 IP:Port
    → iptables 拦截，重定向到 Envoy 15001
    → Envoy 应用路由/限流/重试等策略
    → Envoy 发出 mTLS 加密连接到目标 Pod 的 Envoy
    → 目标 Pod 的 Envoy 解密，转发给业务容器
```

:::warning
如果 Pod 所在的 Namespace 标签正确，但 Pod 没有被注入，最常见的原因是 Pod 的 annotation 中有 `sidecar.istio.io/inject: "false"`，或者 Pod 使用了 `hostNetwork: true`（host 网络模式下 iptables 规则会影响宿主机，Istio 默认不注入）。
:::

## 流量管理核心 API

Istio 的流量治理通过四个核心 CRD 实现，理解它们各自的职责边界是关键。

### VirtualService：路由规则

VirtualService 定义"流量怎么走"，是 Istio 路由规则的核心载体。它匹配请求特征（URI、Header、权重），决定流量的去向。

**金丝雀发布示例**：90% 流量走 v1，10% 走 v2：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: order-service-vs
  namespace: production
spec:
  hosts:
    - order-service          # 匹配发往该 hostname 的流量
  http:
    - route:
        - destination:
            host: order-service
            subset: v1       # 对应 DestinationRule 中定义的 subset
          weight: 90
        - destination:
            host: order-service
            subset: v2
          weight: 10
```

**按 Header 灰度**：测试用户（携带特定 Header）走 v2，其余走 v1：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: order-service-vs
  namespace: production
spec:
  hosts:
    - order-service
  http:
    - match:
        - headers:
            x-canary-user:
              exact: "true"  # Header 精确匹配
      route:
        - destination:
            host: order-service
            subset: v2
    - route:                 # 默认路由（无 match 条件）
        - destination:
            host: order-service
            subset: v1
```

### DestinationRule：流量策略

DestinationRule 定义"到达目标后如何处理"，包括负载均衡策略、连接池配置和熔断（异常检测）。它需要和 VirtualService 配合使用——VirtualService 的 `subset` 字段引用 DestinationRule 中定义的版本分组。

**完整示例：定义版本分组 + 熔断配置**：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: order-service-dr
  namespace: production
spec:
  host: order-service
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100      # 最大连接数
      http:
        http1MaxPendingRequests: 50   # 等待队列长度
        http2MaxRequests: 200         # 最大并发请求数
    outlierDetection:              # 异常检测（熔断）
      consecutiveGatewayErrors: 5  # 连续 5 次网关错误触发熔断
      interval: 30s               # 检测间隔
      baseEjectionTime: 30s       # 最短驱逐时间
      maxEjectionPercent: 50      # 最多驱逐 50% 的实例
  subsets:
    - name: v1
      labels:
        version: v1              # 匹配 Pod 标签
    - name: v2
      labels:
        version: v2
```

:::tip
`outlierDetection`（异常检测）是 Istio 实现熔断的方式。它并非经典的"断路器"模式，而是**主动将请求失败率高的实例从负载均衡池中剔除**。被驱逐的实例在 `baseEjectionTime` 后自动重新纳入。这与 Hystrix 的线程隔离模式在机制上不同，但效果类似。
:::

### Gateway：集群入口

Gateway 定义集群边缘的 Ingress 监听规则（端口、协议、TLS 配置），通常与 VirtualService 配合，将外部流量路由到内部服务：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: production-gateway
  namespace: istio-system
spec:
  selector:
    istio: ingressgateway    # 应用到 Istio Ingress Gateway Pod
  servers:
    - port:
        number: 443
        name: https
        protocol: HTTPS
      tls:
        mode: SIMPLE
        credentialName: prod-tls-cert  # 引用 Kubernetes Secret
      hosts:
        - "api.example.com"
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: api-vs
spec:
  hosts:
    - "api.example.com"
  gateways:
    - production-gateway     # 绑定 Gateway
  http:
    - route:
        - destination:
            host: order-service
            port:
              number: 8080
```

### ServiceEntry：将外部服务纳入网格

默认情况下，Istio 会阻止或放通（取决于 `outboundTrafficPolicy` 配置）所有到网格外部的流量。ServiceEntry 将外部服务（如第三方 API、数据库）注册到 Istio 的服务注册表中，使其可以受 Istio 策略管理：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-payment-api
spec:
  hosts:
    - payment.example.com
  ports:
    - number: 443
      name: https
      protocol: HTTPS
  location: MESH_EXTERNAL
  resolution: DNS
```

## 安全：mTLS 零信任网络

### mTLS 的工作机制

传统 TLS 是单向认证：客户端验证服务端证书。mTLS（mutual TLS）是双向认证：双方都出示证书，互相验证身份。在 Istio 中，每个服务账号（Service Account）对应一个由 Citadel 签发的 SPIFFE 格式证书（`spiffe://cluster.local/ns/{namespace}/sa/{service-account}`），服务间通信时 Envoy 自动完成证书交换和验证。

这构建了零信任网络的基础：**每个服务有密码学意义上的身份，而不仅仅是 IP 地址**。

### PeerAuthentication：认证策略

PeerAuthentication 控制服务间 mTLS 的模式：

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: production
spec:
  mtls:
    mode: STRICT    # 只接受 mTLS 流量，拒绝明文
```

三种模式的含义：

| 模式 | 含义 | 适用场景 |
|------|------|----------|
| `STRICT` | 只接受 mTLS，拒绝明文 | 生产环境，全面启用 mTLS |
| `PERMISSIVE` | 同时接受 mTLS 和明文 | 迁移过渡期，兼容旧服务 |
| `DISABLE` | 禁用 mTLS | 临时调试（不推荐生产使用） |

:::danger
在全量切换到 STRICT 模式之前，务必确认所有调用方都已完成 Sidecar 注入。如果有未注入的客户端（如批处理 Job、监控探针）仍以明文发起调用，切换到 STRICT 后这些调用会立即失败。推荐迁移路径：先全局 PERMISSIVE → 用 Kiali 确认所有流量均为 mTLS → 切换 STRICT。
:::

### AuthorizationPolicy：访问控制

mTLS 解决了"是谁"的问题，AuthorizationPolicy 解决"能访问什么"的问题。它基于 Service Account 身份进行细粒度访问控制：

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: order-service-authz
  namespace: production
spec:
  selector:
    matchLabels:
      app: order-service     # 应用于 order-service 的所有 Pod
  action: ALLOW
  rules:
    - from:
        - source:
            principals:
              # 只允许来自 payment-service SA 的请求
              - "cluster.local/ns/production/sa/payment-service"
      to:
        - operation:
            methods: ["GET", "POST"]
            paths: ["/api/orders*"]
```

这条策略的效果：只有 `payment-service` 这个 Service Account 的请求，才能访问 `order-service` 的 `/api/orders*` 路径，其余所有来源一律拒绝。

**推荐的最小权限配置模式**：先下发一条全局 DENY 策略，再为每个服务单独添加白名单：

```yaml
# 命名空间级别默认拒绝所有流量
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: deny-all
  namespace: production
spec: {}    # 空 spec 表示拒绝所有流量
```

## 可观测性

Service Mesh 的一大优势是**零侵入的可观测性**。Envoy 自动记录所有经过它的请求，无需业务代码埋点。

### Envoy 自动采集的指标

每个 Envoy 实例暴露大量指标，以下是最关键的几类：

```
# 请求总量（按响应码分类）
istio_requests_total{
  source_app, destination_service,
  response_code, reporter
}

# 请求延迟分布（histogram）
istio_request_duration_milliseconds_bucket

# TCP 连接数
istio_tcp_connections_opened_total
istio_tcp_connections_closed_total
```

通过这些指标，可以计算出每对服务之间的 QPS、P99 延迟、错误率——这些数据在没有 Service Mesh 之前需要每个服务团队自行在应用层埋点实现。

### 与 Prometheus 集成

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: istio-envoy-metrics
  namespace: monitoring
spec:
  selector:
    matchLabels:
      istio: pilot             # 采集 Istiod 控制面指标
  endpoints:
    - port: http-monitoring
      path: /metrics
      interval: 15s
```

对于数据面（Envoy Sidecar）的指标，Prometheus Operator 可以通过 PodMonitor 匹配所有注入了 Sidecar 的 Pod 的 15090 端口进行采集。

### 分布式追踪

Envoy 自动传播追踪头（B3 或 W3C TraceContext），无论业务代码使用什么语言，只需应用代码在转发请求时**透传**收到的追踪 Header，Envoy 就能自动完成 Span 的上报。

:::warning
这里有一个常见误解：很多团队以为接入 Istio 后追踪就完全自动化了。实际上 Envoy 只负责入口 Span 和出口 Span，**业务代码内部的调用链（如数据库、缓存）仍然需要应用侧埋点**。而且业务代码必须将上游传入的 `x-b3-traceid` 等 Header 透传到下游请求，否则追踪链会在该节点断开。
:::

**Kiali** 是专为 Istio 设计的可视化工具，它读取 Istio 的配置和 Envoy 的指标，渲染出实时的服务依赖拓扑图，并标注每条链路的健康状态、流量权重、mTLS 状态，是排查 Istio 配置问题的利器。

## 故障注入与流量测试

故障注入是 Istio 最独特的能力之一。在生产环境，你可以直接向特定服务注入延迟或错误，而无需修改代码，用来验证熔断配置是否真正生效。

### 延迟注入

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: order-service-fault-test
  namespace: production
spec:
  hosts:
    - order-service
  http:
    - match:
        - headers:
            x-fault-inject:
              exact: "true"    # 只对带这个 Header 的请求注入故障
      fault:
        delay:
          percentage:
            value: 100         # 100% 的匹配请求注入延迟
          fixedDelay: 3s       # 固定延迟 3 秒
      route:
        - destination:
            host: order-service
            subset: v1
    - route:
        - destination:
            host: order-service
            subset: v1
```

### 中断注入（模拟服务崩溃）

```yaml
fault:
  abort:
    percentage:
      value: 50              # 50% 的请求返回错误
    httpStatus: 503          # 返回 503 状态码
```

通过组合延迟注入和 DestinationRule 中的 `outlierDetection` 配置，可以验证：当 order-service 持续响应缓慢时，调用方的 Envoy 是否会在达到 `consecutiveGatewayErrors` 阈值后将其从负载均衡池中驱逐。

## 性能开销与调优

### Sidecar 的资源代价

每个注入 Envoy 的 Pod 额外消耗约 **50m CPU（idle 状态）、60MB 内存**，在高吞吐场景下每 1000 QPS 额外消耗约 50m-100m CPU。这对于 Pod 数量不多的中型集群影响可以接受，但在 Pod 数量达到数百甚至上千的大型集群中，Sidecar 的累计开销不可忽视。

### Sidecar CR：缩减 Envoy 配置规模

默认情况下，Istio 会将集群中所有服务的路由信息推送到每个 Envoy 实例。当集群中有数百个服务时，每个 Envoy 的配置体积会非常庞大，导致内存占用升高和 xDS 配置同步延迟增加。

通过 Sidecar CR 可以限制每个 Envoy 的"可见范围"，让它只加载自己需要通信的服务的配置：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Sidecar
metadata:
  name: order-service-sidecar
  namespace: production
spec:
  workloadSelector:
    labels:
      app: order-service
  egress:
    - hosts:
        - "./payment-service"      # 只加载同 namespace 的 payment-service
        - "./inventory-service"
        - "istio-system/*"         # 必须保留，用于与控制面通信
```

在大型集群中，合理配置 Sidecar CR 可以将单个 Envoy 的内存占用从 200MB+ 降低到 60-80MB。

### 是否适合引入 Istio 的判断标准

Istio 并不是所有场景的最优解。引入前建议对照以下标准评估：

| 条件 | 适合引入 | 暂缓引入 |
|------|----------|----------|
| 服务数量 | 10+ 个微服务 | 3-5 个服务 |
| 语言栈 | 多语言混合 | 单一语言栈，SDK 方案成熟 |
| 团队规模 | 有专职平台/SRE 团队 | 小团队，没有专职运维 |
| 治理需求 | mTLS、细粒度访问控制、金丝雀 | 简单负载均衡 |
| 集群规模 | 资源充足，可承担 Sidecar 开销 | 资源紧张的小集群 |

:::tip
如果你的团队已经熟悉 Kubernetes，但对 Service Mesh 的运维复杂度有顾虑，可以考虑先以 Ambient Mesh 模式（Istio 1.21+ 正式 GA）评估。Ambient 模式将 Sidecar 替换为节点级别的 ztunnel 代理，消除了 Sidecar 注入的开销，也简化了运维。
:::

## 小结

- Service Mesh 通过 Sidecar 代理模式将流量治理能力下沉到基础设施层，业务代码无需关心熔断、重试、mTLS 等横切关注点
- Istio 控制面由 Istiod 统一承担（Pilot 负责 xDS 配置分发、Citadel 负责证书 CA、Galley 负责配置验证）
- Sidecar 注入通过 MutatingWebhookConfiguration 实现，iptables 规则在不修改业务代码的前提下完成流量透明劫持
- VirtualService 定义路由规则（去哪里），DestinationRule 定义流量策略（怎么处理），两者配合实现金丝雀发布、熔断等能力
- mTLS 通过 PeerAuthentication 控制认证模式，AuthorizationPolicy 基于 Service Account 身份实现细粒度访问控制，构建零信任网络
- Envoy 自动采集服务间流量指标和追踪信息，可观测性零侵入，但追踪头的透传仍需业务代码配合
- 大规模集群需通过 Sidecar CR 限制 Envoy 配置规模，避免内存和 xDS 同步开销失控

---

## 常见问题

### Q1：VirtualService 和 DestinationRule 的关系是什么，必须同时使用吗？

两者职责不同，可以独立使用，但配合使用才能发挥完整功能。VirtualService 负责流量路由：根据请求的 URI、Header、权重等特征决定流量发往哪个目标版本（subset）。DestinationRule 负责到达目标后的策略：负载均衡算法、连接池大小、熔断规则，以及定义 subset 与 Pod 标签的对应关系。

关键依赖点是：VirtualService 中引用的 `subset`（如 `v1`、`v2`）必须在对应的 DestinationRule 中有定义，否则 Envoy 在匹配时找不到目标，流量会返回 503。如果你只需要超时/重试控制而不需要版本路由，单独使用 VirtualService 即可；如果只需要熔断而不需要版本路由，单独使用 DestinationRule 的 `outlierDetection` 也可以生效。

### Q2：Istio 的 mTLS STRICT 模式切换后，为什么部分服务间调用返回 503？

最常见的原因有三类。第一，调用方 Pod 没有被注入 Sidecar（可能是 Namespace 没打标签，或 Pod 有 `sidecar.istio.io/inject: "false"` 注解），导致其以明文发起连接，被 STRICT 模式的 Envoy 拒绝。第二，调用方位于不同 Namespace，而该 Namespace 同样处于 STRICT 模式，但 AuthorizationPolicy 没有正确放通跨 Namespace 的调用来源。第三，存在非 HTTP/gRPC 的长连接服务（如数据库连接），连接在切换模式之前已建立，但切换后新建连接被要求走 mTLS。排查步骤：先确认双方 Pod 是否都有 `istio-proxy` 容器，再查看 `kubectl exec` 进入 Sidecar 容器执行 `pilot-agent request GET /config_dump` 确认 mTLS 配置，最后用 Kiali 观察具体连接的 mTLS 状态。

### Q3：Istio 故障注入和真实故障有什么区别？生产环境能用吗？

Istio 的故障注入在 Envoy 层面实现，它注入的是 **HTTP/gRPC 层面的延迟和错误码**，而真实的故障可能是网络丢包、TCP RST、进程崩溃等更底层的问题。因此故障注入测试的是"当上游返回 503/延迟时，我的熔断和重试逻辑是否正确"，而非完整的混沌工程测试。生产环境使用故障注入是安全的，因为可以通过 Header 匹配将影响范围精确限定在测试请求上（如携带 `x-fault-inject: true` 的请求）。推荐用法：在预发布环境做全量故障注入测试，在生产环境结合 Header 路由做针对性的验证测试，不要在生产环境对全量流量注入高比例故障。

### Q4：Istio 控制面挂掉后，数据面的流量是否会中断？

不会立即中断。Envoy 将最后收到的 xDS 配置缓存在内存中，控制面（Istiod）不可用时，Envoy 继续按照最后已知的配置转发流量。这是 Istio 数据面设计的核心原则之一：**控制面的可用性不影响数据面的基本转发能力**。但有几个重要的限制：首先，在控制面宕机期间，配置变更（新的 VirtualService、DestinationRule 修改）不会生效；其次，证书轮换会停止，如果控制面宕机时间足够长（Istio 默认证书有效期 24 小时），证书过期会导致 mTLS 握手失败；第三，新创建的 Pod 无法获取 xDS 配置，其 Sidecar 会处于不完整状态。因此，Istiod 本身需要多副本部署，并纳入平台可用性监控。

### Q5：Istio 升级风险很高，有没有安全的升级策略？

Istio 的升级相比普通应用复杂，因为涉及控制面和数据面（Envoy Sidecar）两个部分的版本兼容性。推荐的**金丝雀升级**策略是 Istio 官方提供的 `revision` 机制：部署一个新版本的 Istiod，打上新的 revision 标签（如 `istio.io/rev=1-21`），同时保留旧版本 Istiod。先将测试 Namespace 切换到新 revision（修改 Namespace 标签），观察稳定后再逐步迁移生产 Namespace，最后下线旧版本 Istiod。这种方式下，旧 Namespace 继续由旧 Istiod 管理，新旧版本并行运行，任何问题可以立即回滚（将 Namespace 标签切回旧 revision）。升级前还需要确认 Istio 版本与 Kubernetes 版本的兼容矩阵，Istio 通常只支持最新的三个 Kubernetes 次版本。

## 参考资源

- [Istio 官方文档](https://istio.io/latest/docs/)
- [Envoy 代理文档](https://www.envoyproxy.io/docs/envoy/latest/)
- [Istio 最佳实践指南](https://istio.io/latest/docs/ops/best-practices/)

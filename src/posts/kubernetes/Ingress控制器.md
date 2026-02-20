---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# Ingress 控制器深度解析

## 为什么需要 Ingress？

想象这样一个场景：你的电商平台跑在 Kubernetes 上，有商品服务、订单服务、用户服务、支付服务……每个服务都需要对外暴露 HTTPS 端点。如果用 LoadBalancer 类型的 Service，每个服务都会申请一个云厂商的外部负载均衡器，不仅成本线性增长，TLS 证书也要分散管理。

Ingress 解决的核心问题是：**七层流量的统一入口管理**。它在集群边缘做 HTTP/HTTPS 路由，一个外部 IP 处理所有域名和路径，TLS 在入口统一终止，后端服务只需暴露为 ClusterIP。

这个设计边界很重要：Service 处理的是四层（TCP/UDP）连接的服务发现，而 Ingress 处理的是七层（HTTP）的路由决策。两者职责不同，不是竞争关系。

## Ingress 与 Ingress Controller 的关系

这是最容易产生误解的地方。Kubernetes 核心项目中，Ingress 只是一个 API 对象定义——它描述"我想要的路由规则"，但本身不包含任何实现代码。

真正处理流量的是 Ingress Controller：这是一个独立部署在集群内的控制器，它：

1. 通过 Kubernetes API **Watch** Ingress、Service、Endpoints 等资源的变更
2. 将这些声明式规则**翻译**成反向代理软件（Nginx、Envoy 等）的配置格式
3. **动态热加载**新配置，无需重启进程

以 NGINX Ingress Controller 为例，其内部工作流程如下：

```
Ingress 资源变更
      │
      ▼
  Controller 监听到 Watch 事件
      │
      ▼
  模板引擎将 Ingress 规则渲染为 nginx.conf 片段
      │
      ▼
  对比当前配置，判断是否需要 reload
      │
      ├── 仅 upstream 变更 → Lua 动态更新，无需 reload
      └── 结构性变更（新 server block）→ nginx -s reload
```

这个"动态更新"细节值得关注：NGINX Ingress 通过内嵌 Lua（OpenResty）实现了对 Endpoints 变更的热更新。当 Pod 扩缩容导致后端 IP 列表变化时，不需要 reload Nginx 进程，Lua 代码直接修改共享内存中的 upstream 列表。只有当路由规则本身（新增 host、新增 path）发生变化时，才执行代价更高的 `nginx -s reload`。

## 多 Controller 共存：ingressClassName

生产集群中经常需要多个 Ingress Controller 并存：内网服务用 Nginx，公网服务用云厂商的 ALB，或者不同业务用不同的隔离实例。`ingressClassName` 字段就是解决这个问题的机制。

每个 Ingress Controller 在部署时会关联一个 `IngressClass` 资源：

```yaml
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: nginx-internal
spec:
  controller: k8s.io/ingress-nginx
```

Ingress 对象通过 `spec.ingressClassName` 声明自己属于哪个 Controller。Controller 启动时设置 `--watch-ingress-without-class` 等参数控制是否处理未指定 className 的 Ingress。

这个机制的本质是一个**租户划分**：多个 Controller 实例监听同一个 API Server，但通过 IngressClass 过滤出属于自己的 Ingress 对象，互不干扰。

## 主流控制器横向对比

不同 Ingress Controller 背后的数据面技术差异，决定了它们的性能特征和适用场景：

| 维度 | NGINX Ingress | Traefik | Contour | Istio Gateway |
|------|--------------|---------|---------|---------------|
| 数据面 | Nginx + Lua | Traefik（Go） | Envoy | Envoy |
| 配置热更新 | 部分支持（Lua） | 全量热更新 | 全量热更新（xDS） | 全量热更新（xDS） |
| 金丝雀能力 | Annotation 驱动 | 原生支持 | 需 HTTPProxy CRD | VirtualService CRD |
| 限流能力 | Nginx limit_req | 原生中间件 | 需扩展 | 原生支持 |
| 协议支持 | HTTP/1.1, HTTP/2, WebSocket | HTTP/1.1, HTTP/2, gRPC, TCP | HTTP/1.1, HTTP/2, gRPC | 全协议 |
| 适用场景 | 通用场景，成熟稳定 | 自动发现，证书自动化 | 需要强类型 CRD | 已引入服务网格 |

**选型建议**：没有引入服务网格的团队，NGINX Ingress 是最保险的选择——社区成熟度高、文档完善、问题排查路径清晰。如果已经在用 Istio，就直接用 Istio Gateway，避免引入额外的代理层。

Contour 的优势在于它的 `HTTPProxy` CRD 解决了 Ingress 原生 API 表达能力不足的问题，通过强类型的 CRD 替代大量 Annotation，对于需要精细化路由配置的团队有一定吸引力。

## 金丝雀发布原理与策略

金丝雀发布的核心问题是：**如何让一部分流量走新版本，其余流量走旧版本**。NGINX Ingress 通过 Annotation 支持三种分流维度。

### 基于权重的流量分割

这是最简单直接的方式。部署两个 Ingress 对象，分别指向旧版 Service 和新版 Service，新版 Ingress 加上 canary 注解：

```yaml
# 新版本 Ingress，承载 10% 流量
metadata:
  annotations:
    nginx.ingress.kubernetes.io/canary: "true"
    nginx.ingress.kubernetes.io/canary-weight: "10"
```

NGINX 内部通过 Lua 生成随机数来实现概率路由：每个请求生成一个 0~100 的随机数，小于 `canary-weight` 的请求走新版。**缺点**是同一个用户的多次请求可能路由到不同版本，缺乏会话一致性。

### 基于 Header 的精准路由

```yaml
annotations:
  nginx.ingress.kubernetes.io/canary: "true"
  nginx.ingress.kubernetes.io/canary-by-header: "X-Canary"
  nginx.ingress.kubernetes.io/canary-by-header-value: "true"
```

请求携带 `X-Canary: true` header 时进入新版本。这种方式适合内部测试：开发或 QA 人员在请求中手动加上 header，可以精确控制哪些请求走新版本，不影响普通用户。

### 基于 Cookie 的会话一致性

```yaml
annotations:
  nginx.ingress.kubernetes.io/canary: "true"
  nginx.ingress.kubernetes.io/canary-by-cookie: "canary_user"
```

当 Cookie 中 `canary_user=always` 时走新版本，`canary_user=never` 时强制走旧版本。这种方式实现了**粘性分流**：同一个用户始终看到同一个版本，适合需要 A/B 测试且要保证用户体验一致性的场景。

**三种策略的优先级**：Header 匹配 > Cookie 匹配 > 权重随机。实践中通常组合使用：先用 Header 做内部验证，再用 Cookie 做小范围用户灰度，最后切换为权重模式扩大覆盖面。

## 限流机制：漏桶算法在 Ingress 中的实现

NGINX 的限流基于**漏桶算法**（Leaky Bucket）：请求进入一个固定速率的"漏桶"，桶满时新请求被拒绝（返回 429）。这与令牌桶的区别在于：漏桶保证了绝对平滑的输出速率，而令牌桶允许突发流量。

NGINX Ingress 提供两个维度的限流控制：

**基于连接数限流**：限制单个客户端 IP 的并发连接数。
```yaml
nginx.ingress.kubernetes.io/limit-connections: "20"
```

**基于请求速率限流**：限制单个客户端 IP 每秒/每分钟的请求数。
```yaml
nginx.ingress.kubernetes.io/limit-rps: "100"   # 每秒100个请求
nginx.ingress.kubernetes.io/limit-rpm: "1000"  # 每分钟1000个请求
```

### 限流的关键问题：多副本下的状态共享

这是生产中最容易忽视的问题。当 NGINX Ingress Controller 有多个副本时，每个副本维护各自独立的限流计数器。如果你有 3 个副本，实际上每个客户端能享受到 3 倍于配置值的速率。

解决这个问题有两种思路：

1. **Ingress 单副本模式**：牺牲 Controller 的高可用，换取计数器一致性。适合内部服务。
2. **外部限流**：将限流能力下沉到 Redis，多个 Nginx 副本共享同一个计数器。NGINX Plus 支持这个特性，开源版需要自行通过 Lua 实现，或者引入 API Gateway（如 Kong）。

### 白名单豁免

```yaml
nginx.ingress.kubernetes.io/limit-whitelist: "10.0.0.0/8,172.16.0.0/12"
```

来自内网 CIDR 的请求不受限流规则约束。这在混合了内外部流量的入口上很有用。

## OAuth2 认证代理集成

在 Ingress 层统一做认证，是比在每个后端服务单独实现认证更简洁的方案。`oauth2-proxy` 是这个场景下的标准工具。

### 工作原理

`oauth2-proxy` 充当 OIDC（OpenID Connect）客户端，利用 NGINX Ingress 的 `auth_request` 模块工作：

```
用户请求
  │
  ▼
NGINX Ingress 收到请求
  │
  ├── 向 oauth2-proxy 发起 subrequest（auth_request）
  │     │
  │     ├── subrequest 携带原始请求的 Cookie/Header
  │     │
  │     └── oauth2-proxy 验证 Session Cookie 是否有效
  │           ├── 有效 → 返回 202，携带用户身份 Header
  │           └── 无效 → 返回 401，触发 OAuth2 授权流程
  │
  ├── subrequest 返回 202 → 转发原始请求到后端，附加用户身份 Header
  └── subrequest 返回 401 → 重定向用户到 OAuth2 Provider 登录页
```

关键点在于：认证逻辑完全在 `oauth2-proxy` 中，NGINX 只是通过 `auth_request` 机制把每个请求的"认证决策"外包给它。后端服务收到请求时，已经携带了 `X-Auth-User`、`X-Auth-Email` 等身份 Header，不需要自己处理 OAuth2 流程。

### Ingress 配置要点

需要两个 Ingress 对象协同：一个给 `oauth2-proxy` 自身（处理 `/oauth2/` 路径），一个给业务服务（配置 auth 注解）：

```yaml
# 业务服务的 Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/auth-url: "https://auth.example.com/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://auth.example.com/oauth2/start?rd=$escaped_request_uri"
    nginx.ingress.kubernetes.io/auth-response-headers: "X-Auth-User,X-Auth-Email"
```

`auth-url` 是 NGINX 发起 subrequest 的地址，`auth-signin` 是认证失败时的重定向地址，`auth-response-headers` 指定把 `oauth2-proxy` 返回的哪些 Header 透传给后端。

## 生产配置最佳实践

### 超时与连接管理

```yaml
nginx.ingress.kubernetes.io/proxy-connect-timeout: "5"      # 与后端建立连接超时（秒）
nginx.ingress.kubernetes.io/proxy-send-timeout: "60"        # 发送请求超时
nginx.ingress.kubernetes.io/proxy-read-timeout: "60"        # 等待响应超时
nginx.ingress.kubernetes.io/proxy-next-upstream: "error timeout"  # 失败时重试
```

`proxy-read-timeout` 是最常被忽视的。默认值 60 秒对于 SSE（Server-Sent Events）或 WebSocket 长连接来说太短，需要根据业务协议调整。

**Keepalive 优化**：NGINX 与后端 Service 之间默认使用短连接。对于高频调用的服务，开启 keepalive 可以显著减少 TCP 握手开销：

```yaml
nginx.ingress.kubernetes.io/upstream-keepalive-connections: "100"
nginx.ingress.kubernetes.io/upstream-keepalive-time: "1h"
```

### 资源配置

Ingress Controller 是集群的流量枢纽，资源配置直接影响全局稳定性：

```yaml
resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 2
    memory: 1Gi
```

**不要设置过低的 CPU limit**：Nginx worker 进程在处理 TLS 握手时是 CPU 密集型的，突发流量下 CPU 限制会导致明显的延迟上升。内存限制同样要留有余量，配置文件和 Lua 缓存都需要稳定的内存空间。

### 访问日志与可观测性

```yaml
# ConfigMap: ingress-nginx-controller
log-format-upstream: >
  {"time":"$time_iso8601","remote_addr":"$remote_addr",
   "host":"$host","request":"$request",
   "status":$status,"bytes_sent":$bytes_sent,
   "request_time":$request_time,"upstream_addr":"$upstream_addr",
   "upstream_response_time":$upstream_response_time}
```

将访问日志转为 JSON 格式，方便日志系统（Loki、Elasticsearch）解析。`upstream_response_time` 是后端处理时间，`request_time` 是总时间，两者之差反映了 Nginx 自身的处理延迟。

## 小结

Ingress 体系的核心认知：

- **分层理解**：Ingress 是 API 声明，Controller 是实现体，两者通过 Watch 机制连接，配置变更通过模板渲染转化为反向代理配置
- **ingressClassName 是多租户隔离手段**，同一集群可以运行多个 Controller 实例服务不同场景
- **金丝雀三策略组合使用**：Header 做内测 → Cookie 做定向灰度 → Weight 做全量放量
- **限流的多副本陷阱**：单机计数在多 Controller 副本下失效，生产环境需要评估是否引入共享状态
- **auth_request 模式**是 Ingress 层统一认证的标准实现，职责清晰，后端无感知

---

## 常见问题

### Q1：Ingress Controller 的 nginx.conf reload 会导致流量中断吗？

不会导致连接中断，但有性能影响。NGINX reload 采用优雅方式：新的 worker 进程加载新配置开始接受请求，旧的 worker 进程等待现有连接处理完毕后退出。客户端已建立的连接不会被强制断开。但 reload 本身有开销，频繁 reload（如每秒多次 Ingress 资源变更）会增加系统负载。NGINX Ingress Controller 内置了 Lua 动态更新机制来减少 reload 频率：Endpoints 变化（Pod 扩缩容）不触发 reload，只有 Ingress 规则结构变化才触发。生产中建议监控 `nginx_ingress_controller_config_last_reload_successful` 和 `nginx_ingress_controller_config_reloads_total` 指标。

### Q2：金丝雀发布时如何保证同一用户不会在新旧版本之间跳变？

权重模式本质上是无状态的随机路由，无法保证会话一致性。要实现同一用户固定路由，需要使用 Cookie 策略：将 `canary_user=always` 写入特定用户的浏览器 Cookie，这类用户每次请求都会命中新版本。在实际落地中，通常通过功能开关系统（Feature Flag）与 Cookie 联动：用户触发灰度条件时，由后端服务在响应中设置 canary Cookie，后续请求自动走 Ingress 的 Cookie 路由。这样就把"谁是灰度用户"的决策权留在业务层，Ingress 只负责执行路由规则。

### Q3：集群内多个业务团队共用一个 Nginx Ingress Controller 时，如何防止一个团队的配置影响其他团队？

共用 Controller 存在以下风险：某个 Ingress 配置了过大的 `proxy-body-size` 或错误的注解可能影响全局行为；某个服务的流量突增消耗 Controller 资源影响其他服务。解决方案有两个方向：一是为不同团队部署独立的 Controller 实例，通过 `ingressClassName` 隔离（推荐，彻底隔离故障域）；二是在共用 Controller 上通过 NGINX ConfigMap 设置全局限制作为安全兜底，同时配合 Kubernetes RBAC 限制各团队只能修改自己 Namespace 下的 Ingress 资源，防止误改他人配置。

### Q4：oauth2-proxy 与 Ingress 的 auth_request 模式，和直接在 Istio 中做 JWT 认证有何本质区别？

两者处于不同的架构层次，解决不同问题。`oauth2-proxy + auth_request` 是**会话（Session）级认证**：用户完成一次 OAuth2 登录后，Session 存储在 Redis 中，后续请求通过 Cookie 携带 Session ID，由 `oauth2-proxy` 验证；这套机制面向浏览器用户，有登录/登出流程。Istio 的 JWT 认证是**令牌（Token）级认证**：每个请求必须携带有效的 JWT，Envoy 在请求入口验证签名和声明，适合服务间调用和 API Client 场景。在实际架构中两者常常并存：外部用户流量走 Ingress + oauth2-proxy，服务网格内部的东西向流量走 Istio mTLS + JWT。

### Q5：如何对 Ingress Controller 本身进行高可用和性能调优？

高可用方面：将 Controller 以 DaemonSet 或 Deployment（至少 2 副本）部署，配合 PodDisruptionBudget 确保维护时不中断；使用 `externalTrafficPolicy: Local` 保留客户端真实 IP 的同时减少跨节点跳转（但需注意流量均衡性问题）。性能调优的关键参数：`worker-processes` 设为节点 CPU 核数；`max-worker-connections` 控制每个 worker 的并发连接数，默认 16384，高流量场景可适当调大；开启 `enable-brotli` 和 `use-gzip` 减少传输带宽；对于延迟敏感的服务，关闭 access log 或异步写日志可降低 P99 延迟。监控指标上重点关注：Controller 的 CPU/内存使用率、`nginx_ingress_controller_requests` 请求速率、`nginx_ingress_controller_ingress_upstream_latency_seconds` 上游延迟分布。

## 参考资源

- [Kubernetes Ingress 官方文档](https://kubernetes.io/docs/concepts/services-networking/ingress/)
- [NGINX Ingress Controller 文档](https://kubernetes.github.io/ingress-nginx/)
- [IngressClass 配置](https://kubernetes.io/docs/concepts/services-networking/ingress/#ingress-class)

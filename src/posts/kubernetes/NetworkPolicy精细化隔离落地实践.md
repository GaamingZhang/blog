---
date: 2026-02-06
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# NetworkPolicy 精细化隔离落地实践

## 为什么需要精细化隔离？

如果说基础的 NetworkPolicy 是给公寓楼装了门禁，那么精细化隔离就是**建立一套完整的访问控制体系**：不仅控制谁能进，还要控制能去哪个楼层、能访问哪个房间、在什么时间段访问。

在真实的生产环境中，你会面对这些场景：

- **多租户隔离**：不同团队/项目的服务必须完全隔离
- **微服务安全边界**：前端层、业务层、数据层之间需要清晰的访问控制
- **合规要求**：某些行业要求网络层的零信任模型
- **攻击面收敛**：即使某个 Pod 被攻破，也无法横向扩散

本文将聚焦于**如何设计和落地**一套完整的精细化隔离方案。

## 零信任网络模型的设计思路

传统的"城墙式"安全模型假设"内网是安全的"，这在云原生环境中已经不适用。零信任模型的核心原则是：

> **默认拒绝，显式允许；最小权限，持续验证**

在 Kubernetes 中落地这个原则，需要：

| 设计原则 | 实现方式 |
|---------|---------|
| **默认拒绝** | 在每个 Namespace 创建 deny-all 基线策略 |
| **最小权限** | 只开放必需的端口和协议，使用 Ingress + Egress 双向控制 |
| **身份标识** | 通过 Label 体系标识 Pod 的角色、环境、团队 |
| **持续验证** | 定期审计策略，结合监控告警 |

这就像建立一个"需要多重认证才能访问"的系统：不仅要验证你是谁（Label），还要验证你能做什么（策略规则）。

## Namespace 标签体系：落地的基石

**关键洞察**：Label 不是可选项，而是整个策略体系的基础架构。

设计一套结构化的标签体系：

```yaml
# 给 Namespace 打标签
kubectl label namespace frontend \
  env=prod \
  tier=web \
  team=platform

kubectl label namespace backend \
  env=prod \
  tier=api \
  team=platform

kubectl label namespace database \
  env=prod \
  tier=data \
  team=platform \
  sensitivity=high
```

**标签维度建议**：

| 标签键 | 用途 | 示例值 |
|-------|------|--------|
| `env` | 环境隔离 | prod, staging, dev |
| `tier` | 分层架构 | web, api, data, cache |
| `team` | 团队归属 | platform, business, data-team |
| `sensitivity` | 敏感级别 | high, medium, low |

这套体系让你可以用组合逻辑表达复杂的访问规则，比如"只允许生产环境的 API 层访问生产环境的数据层"。

## 场景一：仅允许指定 Namespace 的 Pod 通信

### 需求分析

典型的三层架构：
- Frontend（前端）→ Backend（后端 API）
- Backend（后端 API）→ Database（数据库）
- Database **不能**被 Frontend 直接访问

### 策略设计

**第一步：建立基线（拒绝所有）**

```yaml
# database namespace 的基线策略
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-baseline
  namespace: database
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
```

**第二步：显式允许必要的通信**

```yaml
# 允许 backend namespace 访问 database
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-backend
  namespace: database
spec:
  podSelector:
    matchLabels:
      app: postgres  # 只对数据库 Pod 生效
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              tier: api  # 关键：通过 tier 标签选择
              env: prod
        ports:
          - protocol: TCP
            port: 5432
```

**第三步：允许 DNS 和监控（容易遗漏）**

```yaml
# database namespace 允许 DNS 查询
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-dns-egress
  namespace: database
spec:
  podSelector: {}
  policyTypes:
    - Egress
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: kube-system
      ports:
        - protocol: UDP
          port: 53
    # 允许监控系统抓取指标
    - to:
        - namespaceSelector:
            matchLabels:
              name: monitoring
```

### 为什么这样设计有效？

- **namespaceSelector** 实现了跨 Namespace 的访问控制
- **tier 标签** 让策略不依赖具体的 Namespace 名称（更灵活）
- **端口限制** 进一步缩小攻击面（只开放 5432，其他端口仍然拒绝）

## 场景二：禁止外部未授权访问

### 需求分析

"外部"在 Kubernetes 中有多个含义：
1. 集群外部的请求（通过 Ingress Controller 进入）
2. 其他 Namespace 的 Pod
3. 没有正确标签的 Pod

### 策略设计：分层防护

**第一层：Ingress Controller 准入控制**

只允许来自 Ingress Controller Namespace 的流量：

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-ingress-controller
  namespace: frontend
spec:
  podSelector:
    matchLabels:
      expose: "true"  # 只有明确标记的 Pod 才允许对外暴露
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080
```

**第二层：内部服务的横向隔离**

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: internal-service-isolation
  namespace: backend
spec:
  podSelector:
    matchLabels:
      internal: "true"  # 内部服务标记
  policyTypes:
    - Ingress
  ingress:
    - from:
        # 只允许同 Namespace 且带有特定标签的 Pod
        - podSelector:
            matchLabels:
              tier: api
        # 以及来自同一环境的其他 API 服务
        - namespaceSelector:
            matchLabels:
              env: prod
              tier: api
```

**第三层：Egress 控制（防止数据泄露）**

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: restrict-egress
  namespace: backend
spec:
  podSelector: {}
  policyTypes:
    - Egress
  egress:
    # 允许访问数据层
    - to:
        - namespaceSelector:
            matchLabels:
              tier: data
      ports:
        - protocol: TCP
          port: 5432
        - protocol: TCP
          port: 6379  # Redis
    # 允许 DNS
    - to:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: kube-system
      ports:
        - protocol: UDP
          port: 53
    # 允许访问外部 HTTPS API（如支付接口）
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443
```

### 设计要点

- **expose 标签**：显式声明哪些服务可以对外暴露，避免"默认暴露"
- **internal 标签**：区分内部服务和边缘服务，内部服务不应被外部直接访问
- **Egress 白名单**：限制出站流量，防止被攻破后的数据外泄

## 生产落地的步骤与注意事项

### 灰度策略：三步走

| 阶段 | 操作 | 验证方法 |
|-----|------|---------|
| **观察模式** | 不创建策略，先确保所有 Label 正确打上 | `kubectl get ns --show-labels` |
| **日志模式** | 使用 Calico/Cilium 的日志功能，观察哪些流量会被拒绝 | 分析网络插件的日志 |
| **强制模式** | 创建 NetworkPolicy，先在测试环境验证 | 运行端到端测试 |

**关键原则**：不要在生产环境直接创建 deny-all 策略，一定要先在测试环境验证所有访问路径。

### 验证方法

**1. 功能测试**

```bash
# 从 frontend Pod 测试能否访问 backend
kubectl exec -it -n frontend frontend-pod -- curl http://backend-service.backend.svc.cluster.local:8080/health

# 从 frontend Pod 测试是否被阻止访问 database（应该失败）
kubectl exec -it -n frontend frontend-pod -- nc -zv postgres-service.database.svc.cluster.local 5432
```

**2. 策略审计**

```bash
# 列出所有 NetworkPolicy
kubectl get networkpolicy --all-namespaces

# 检查特定 Namespace 的策略
kubectl describe networkpolicy -n database

# 验证 Label 是否正确
kubectl get pods -n backend --show-labels
```

**3. 监控告警**

使用 Prometheus + Grafana 监控被拒绝的连接：

```promql
# Calico 拒绝的连接数
sum(rate(calico_denied_packets[5m])) by (namespace, pod)
```

### DNS 放行的最佳实践

**常见错误**：忘记放行 kube-dns 或 coredns 的流量

**正确做法**：

```yaml
# 方式一：通过 Namespace 选择
- to:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: kube-system
  ports:
    - protocol: UDP
      port: 53

# 方式二：通过 Pod 选择（更精确）
- to:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: kube-system
      podSelector:
        matchLabels:
          k8s-app: kube-dns
  ports:
    - protocol: UDP
      port: 53
```

**提示**：如果你的 DNS 服务使用的是 CoreDNS，Label 可能是 `k8s-app: kube-dns` 或 `k8s-app: coredns`，用以下命令确认：

```bash
kubectl get pods -n kube-system -l k8s-app=kube-dns --show-labels
```

## 与 Calico/Cilium 的结合

Kubernetes 原生的 NetworkPolicy 有一些局限性：

| 限制 | Calico/Cilium 的增强能力 |
|-----|-------------------------|
| 不支持按 IP CIDR 段控制 | Calico 支持 `ipBlock` 和全局策略 |
| 不支持 L7 规则（HTTP 路径） | Cilium 支持 L7 策略（基于 HTTP/gRPC） |
| 不支持全局策略 | Calico 的 GlobalNetworkPolicy |
| 不支持日志审计 | Calico/Cilium 都提供网络流日志 |

**简要建议**：
- 如果只需要基础的 Namespace/Pod 隔离，Kubernetes 原生策略足够
- 如果需要更细粒度的控制（如 HTTP 路径级别），考虑 Cilium
- 如果需要全局策略和 IP 白名单，考虑 Calico

## 常见问题

### Q1: 策略创建后，应用间歇性连接失败？

**可能原因**：
- **DNS 缓存问题**：DNS 解析成功但后续连接被拒绝
- **健康检查被阻止**：kubelet 的健康检查可能需要放行

**解决方法**：
```yaml
# 允许本节点的 kubelet 健康检查
- from:
    - ipBlock:
        cidr: 0.0.0.0/0  # 或者具体的节点 CIDR
  ports:
    - protocol: TCP
      port: 8080  # 健康检查端口
```

### Q2: 如何处理第三方服务（如消息队列）？

如果使用外部托管的服务（如 AWS RDS、云消息队列），需要在 Egress 中使用 `ipBlock`：

```yaml
egress:
  - to:
      - ipBlock:
          cidr: 10.0.5.0/24  # RDS 的 VPC CIDR
    ports:
      - protocol: TCP
        port: 5432
```

### Q3: 多个策略作用于同一个 Pod 时的行为？

**关键理解**：多个 NetworkPolicy 是**叠加**的（取并集），而不是覆盖。

例如：
- 策略 A 允许来自 Namespace X 的流量
- 策略 B 允许来自 Namespace Y 的流量
- 最终结果：X 和 Y 都可以访问

这就像多把钥匙都能开同一扇门。

### Q4: 如何回滚策略？

```bash
# 删除特定策略
kubectl delete networkpolicy <policy-name> -n <namespace>

# 删除 Namespace 下所有策略（慎用）
kubectl delete networkpolicy --all -n <namespace>
```

**最佳实践**：将策略存为 YAML 文件，放入 Git 版本控制，回滚时重新 apply 旧版本。

## 小结

| 实践要点 | 关键内容 |
|---------|---------|
| **设计原则** | 默认拒绝 + 显式允许；最小权限 + 双向控制 |
| **标签体系** | env、tier、team 等多维度标签，Namespace 和 Pod 都要打 |
| **Namespace 隔离** | 用 namespaceSelector + tier 标签实现分层访问控制 |
| **外部访问控制** | Ingress Controller 准入 + expose 标签 + Egress 白名单 |
| **落地步骤** | 观察模式 → 日志模式 → 强制模式；先测试环境验证 |
| **DNS 放行** | 必须显式允许 kube-system 的 UDP 53 端口 |
| **验证方法** | 功能测试 + 策略审计 + 监控告警 |

记住：精细化隔离不是一蹴而就的，需要逐步建立 Label 体系、逐步收紧策略、持续验证效果。就像建造一座安全的城堡，需要一砖一瓦地加固，而不是一夜之间完成。

## 参考资源

- [Kubernetes NetworkPolicy 官方文档](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [Calico 网络策略文档](https://docs.projectcalico.org/security/calico-network-policy)
- [Cilium 网络策略指南](https://docs.cilium.io/en/stable/policy/)

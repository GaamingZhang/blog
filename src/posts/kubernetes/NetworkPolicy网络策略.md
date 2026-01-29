# NetworkPolicy网络策略

## 为什么需要网络策略？

想象一下你住在一个大型公寓楼里。默认情况下，任何住户都可以敲任何人的门，这在某些情况下很方便，但也存在安全隐患。如果你想限制只有朋友才能来访，而不是任何陌生人，你就需要一个门禁系统。

Kubernetes的NetworkPolicy就是这样的"门禁系统"：

- **默认情况**：集群里所有Pod都可以互相通信，没有任何限制
- **有了NetworkPolicy**：你可以精确控制"谁能访问谁"

这对安全至关重要。比如，你的数据库Pod应该只允许后端应用访问，而不是被任何Pod都能连上。

## 核心概念

NetworkPolicy有两个方向的控制：

| 方向 | 含义 | 比喻 |
|------|------|------|
| **Ingress（入站）** | 控制谁能访问我 | 门卫检查谁能进来 |
| **Egress（出站）** | 控制我能访问谁 | 门卫检查我能去哪里 |

**重要原则**：

- 没有任何NetworkPolicy时 = 所有流量都允许
- 一旦对Pod应用了NetworkPolicy = 默认拒绝该方向的所有流量，只允许明确规定的

就像装了门禁后，默认是锁着的，只有刷卡的人才能进。

## 前提条件：需要CNI支持

这里有个很重要的点：**NetworkPolicy本身只是规则定义，需要网络插件（CNI）来执行**。

常见CNI对NetworkPolicy的支持：

| CNI插件 | 支持情况 |
|---------|----------|
| Calico | 完全支持 |
| Cilium | 完全支持 |
| Weave Net | 完全支持 |
| Flannel | 不支持（需配合Calico使用） |

如果你的集群用的是Flannel且没有Calico，那配置NetworkPolicy是不会生效的。

## 最简单的例子：拒绝所有入站流量

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-ingress
  namespace: production
spec:
  podSelector: {}  # 空的selector = 选中这个namespace的所有Pod
  policyTypes:
    - Ingress
  # 注意：这里没有定义ingress规则，意味着拒绝所有入站
```

这个策略的意思是："production命名空间里的所有Pod，拒绝任何入站流量"。

**为什么这样设计？** 因为一旦你指定了 `policyTypes: [Ingress]`，就表示"我要控制入站流量"。如果不写具体的允许规则，默认就是全部拒绝。

## 只允许特定Pod访问

假设你有一个API服务，只想让前端Pod访问：

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: api-allow-frontend
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: api  # 这个策略应用于带有app=api标签的Pod
  policyTypes:
    - Ingress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: frontend  # 只允许带有app=frontend标签的Pod访问
      ports:
        - protocol: TCP
          port: 8080  # 只允许访问8080端口
```

这个策略说的是："API Pod只允许frontend Pod通过TCP 8080端口访问"。

## 理解选择器：AND还是OR？

这是初学者经常困惑的地方。在NetworkPolicy中：

**同一个 `from` 下的多个选择器是AND关系**：

```yaml
from:
  - namespaceSelector:
      matchLabels:
        env: prod
    podSelector:
      matchLabels:
        app: web
  # 这表示：来自prod环境namespace 且 带有app=web标签的Pod
```

**不同的 `from` 条目是OR关系**：

```yaml
from:
  - podSelector:
      matchLabels:
        app: frontend
  - podSelector:
      matchLabels:
        app: monitoring
  # 这表示：frontend Pod 或者 monitoring Pod
```

用生活中的例子来说：
- AND就像"既是VIP客户，又持有邀请函的人才能进"
- OR就像"VIP客户可以进，或者持有邀请函的人可以进"

## 别忘了DNS！

这是一个非常常见的坑。当你创建了一个拒绝所有出站流量的策略后，Pod可能连服务名都解析不了，因为DNS查询也被阻断了。

**记住**：如果你控制了Egress，一定要允许DNS：

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-dns
  namespace: production
spec:
  podSelector: {}
  policyTypes:
    - Egress
  egress:
    - to:
        - namespaceSelector: {}
          podSelector:
            matchLabels:
              k8s-app: kube-dns
      ports:
        - protocol: UDP
          port: 53
```

这个策略允许所有Pod访问集群的DNS服务。

## 最佳实践：默认拒绝+按需允许

安全性最高的做法是：

1. 先创建一个"拒绝所有"的策略
2. 然后根据业务需求，一个一个添加允许规则

这就像先把所有门都锁上，然后只给需要的人发钥匙。

## 常见问题

### Q1: NetworkPolicy配置了但不生效？

**最常见的原因**：

1. **CNI不支持**：检查你的网络插件是否支持NetworkPolicy
2. **标签没匹配上**：用 `kubectl get pod --show-labels` 确认Pod的标签
3. **namespace标签缺失**：如果用了namespaceSelector，确保namespace有对应的标签

**原理解释**：NetworkPolicy是通过标签选择器来工作的。如果标签不匹配，策略就像"没有目标"一样，不会生效。

### Q2: 添加NetworkPolicy后，应用无法连接其他服务？

这通常是因为你控制了Egress但忘了放行必要的流量。

**检查清单**：
- DNS放行了吗？（UDP 53端口）
- 目标服务的端口放行了吗？
- 如果是跨namespace访问，namespaceSelector配对了吗？

### Q3: 如何允许来自Ingress Controller的流量？

Ingress Controller通常运行在 `ingress-nginx` 或类似的namespace里。你需要允许来自那个namespace的流量：

```yaml
ingress:
  - from:
      - namespaceSelector:
          matchLabels:
            name: ingress-nginx
```

**注意**：你可能需要先给ingress-nginx namespace添加标签：
```bash
kubectl label namespace ingress-nginx name=ingress-nginx
```

### Q4: Ingress和Egress可以同时控制吗？

可以。一个NetworkPolicy可以同时控制两个方向：

```yaml
policyTypes:
  - Ingress
  - Egress
```

但建议分开写，这样更清晰，也更容易调试。

## 小结

| 概念 | 解释 |
|------|------|
| NetworkPolicy | Kubernetes的网络防火墙，控制Pod间的流量 |
| Ingress | 入站规则，控制谁能访问这个Pod |
| Egress | 出站规则，控制这个Pod能访问谁 |
| podSelector | 策略应用于哪些Pod |
| 默认行为 | 无策略=全允许；有策略=默认拒绝，只允许明确规定的 |

记住最重要的一点：**NetworkPolicy是"白名单"机制**。一旦你开始控制某个方向的流量，就需要明确列出所有允许的来源/目标，其他的都会被拒绝。

---
date: 2026-02-09
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# 单体服务迁移至Kubernetes（三）：迁移策略与生产实践

## 选择迁移策略

前两篇文章解决了应用容器化和部署设计的问题，现在面临最关键的决策：如何安全地把生产流量从旧环境切换到Kubernetes？这个决策的核心是风险控制。生产环境容不得闪失，一次错误的切换可能导致大规模的业务中断。

### 大爆炸式迁移的风险

大爆炸式迁移是一次性将所有流量从旧环境切换到新环境。操作直接但风险最高。

```
切换前：用户 → 旧环境（100%流量）
切换后：用户 → Kubernetes（100%流量）
```

这种方式的问题在于影响面是100%。如果新环境有任何问题——性能不如预期、配置错误、网络故障、资源不足——所有用户都会受到影响。即使你做了充分的压测和预生产验证，生产环境的复杂性总会超出预期。

适用场景非常有限：流量小的内部系统、非核心业务、或者团队对Kubernetes已经有充分的实践经验。对于核心业务系统，不建议采用这种方式。

### 绞杀者模式的渐进迁移

绞杀者模式（Strangler Fig Pattern）是一种更稳妥的迁移策略。这个名字来源于热带雨林中的绞杀榕，它缠绕在宿主树上生长，逐渐吸收养分，最终完全取代宿主。在系统迁移中，新系统逐步接管旧系统的流量，直到完全替代。

```
阶段1：用户 → 旧环境（95%） + Kubernetes（5%）
阶段2：用户 → 旧环境（80%） + Kubernetes（20%）
阶段3：用户 → 旧环境（50%） + Kubernetes（50%）
阶段4：用户 → 旧环境（20%） + Kubernetes（80%）
阶段5：用户 → Kubernetes（100%），下线旧环境
```

每个阶段都是一个观察窗口。在放量前需要定义明确的通过标准：错误率不超过基线、P99延迟不高于旧环境、资源使用在预期范围内、没有频繁的Pod重启。只有当前阶段的指标全部符合预期，才进入下一个阶段。任何异常都立即回滚到上一阶段。

这种方式的优势是风险可控，问题影响面小。初期只有5%的流量进入新环境，即使出现问题也只影响小部分用户。团队有时间观察和调整，积累信心。缺点是迁移周期长，需要维护两套环境，但这是生产环境迁移的必要成本。

### 蓝绿部署与金丝雀发布的差异

蓝绿部署和金丝雀发布是两种常用的发布策略，但它们的目标和实现方式不同。

**蓝绿部署**的核心是环境级的原子切换。蓝色环境是当前运行的版本，绿色环境是新部署的版本。两个环境同时存在且完全独立，每个环境都能承载100%的流量。验证绿色环境没有问题后，通过负载均衡器或DNS一次性将流量从蓝色切换到绿色。如果绿色环境有问题，立即切回蓝色环境。

```
蓝绿部署：
┌─────────────────────────────────────┐
│ 准备阶段                             │
│ 蓝色环境（v1.0）100%流量             │
│ 绿色环境（v2.0）0%流量 ← 部署和验证   │
└─────────────────────────────────────┘
            ↓ 切换（瞬间完成）
┌─────────────────────────────────────┐
│ 切换后                               │
│ 蓝色环境（v1.0）0%流量 ← 待下线       │
│ 绿色环境（v2.0）100%流量             │
└─────────────────────────────────────┘
```

蓝绿部署的优点是切换速度快，回滚也快，对用户影响最小。缺点是需要双倍的资源，因为两套环境同时运行。适用于对可用性要求极高的核心系统，以及需要快速回滚能力的场景。

**金丝雀发布**的核心是流量级的渐进验证。新版本先部署到少量实例，只接收少量流量（如5%）。这些实例就像矿井中的金丝雀，用来探测危险。如果金丝雀实例运行正常，逐步增加新版本的实例数量和流量比例，最终完全替换旧版本。

```
金丝雀发布：
阶段1：v1.0（95%流量）+ v2.0金丝雀（5%流量）
阶段2：v1.0（80%流量）+ v2.0（20%流量）
阶段3：v1.0（50%流量）+ v2.0（50%流量）
阶段4：v1.0（20%流量）+ v2.0（80%流量）
阶段5：v2.0（100%流量）
```

金丝雀发布的优点是风险最低，问题影响面小，不需要双倍资源。缺点是发布周期长，需要更复杂的流量控制机制。适用于对稳定性要求高、能容忍较长发布周期的场景。

**两者的本质区别**：
- 蓝绿部署是环境级切换，关注的是快速切换和回滚
- 金丝雀发布是流量级渐进，关注的是风险控制和问题早发现
- 蓝绿部署的切换是瞬间的（0% → 100%），金丝雀是渐进的（5% → 100%）
- 蓝绿部署需要双倍资源，金丝雀不需要

在Kubernetes迁移场景中，可以结合两种策略：用蓝绿部署的思想维护旧环境和Kubernetes两套环境，用金丝雀发布的思想逐步增加Kubernetes的流量比例。这样既有快速回滚的能力（切回旧环境），又有渐进验证的安全性（逐步放量）。

## 滚动更新的底层机制

Kubernetes的滚动更新（Rolling Update）是实现零停机发布的核心机制。理解它的底层原理能帮助你更好地配置更新策略，排查更新过程中的问题。

### Deployment Controller的工作原理

Deployment并不直接管理Pod，而是通过ReplicaSet作为中间层。这种三层架构的设计是滚动更新的基础。

```
Deployment的三层架构：
┌──────────────────────────────────────┐
│ Deployment                           │
│ spec.replicas: 3                     │
│ spec.template.image: app:v2.0        │
└───────────┬──────────────────────────┘
            │ 控制
            ↓
┌──────────────────────────────────────┐
│ ReplicaSet-v1        ReplicaSet-v2   │
│ replicas: 0          replicas: 3     │
│ image: app:v1.0      image: app:v2.0 │
└───────────┬──────────────────────────┘
            │ 管理
            ↓
┌──────────────────────────────────────┐
│ Pod-v2-1   Pod-v2-2   Pod-v2-3       │
└──────────────────────────────────────┘
```

当你更新Deployment的镜像时（比如从v1.0到v2.0），Deployment Controller执行以下步骤：

1. **创建新ReplicaSet**：根据新的Pod模板（镜像v2.0）创建一个新的ReplicaSet，初始副本数为0
2. **逐步扩容新ReplicaSet**：增加新ReplicaSet的副本数
3. **逐步缩容旧ReplicaSet**：减少旧ReplicaSet的副本数
4. **重复步骤2-3**：直到新ReplicaSet达到期望副本数，旧ReplicaSet降为0

这个过程是自动的、可控的、可逆的。Deployment Controller会持续监控Pod的健康状态，只有当新Pod通过就绪探针后，才会继续创建下一个新Pod或删除旧Pod。

### maxSurge和maxUnavailable的精确计算

滚动更新的速度和风险由两个参数控制：`maxSurge`和`maxUnavailable`。

**maxSurge**定义了更新过程中可以创建多少个超出期望副本数的Pod。它可以是绝对数字（如2）或百分比（如25%）。

**maxUnavailable**定义了更新过程中可以有多少个Pod不可用。同样可以是绝对数字或百分比。

假设Deployment的期望副本数是10，maxSurge=2，maxUnavailable=1。计算逻辑如下：

```
最大Pod数 = 期望副本数 + maxSurge = 10 + 2 = 12
最小可用Pod数 = 期望副本数 - maxUnavailable = 10 - 1 = 9
```

更新过程的实际执行：

```
初始状态：
旧ReplicaSet: 10个Pod（全部Ready）
新ReplicaSet: 0个Pod
可用Pod数: 10

步骤1：创建新Pod
旧ReplicaSet: 10个Pod
新ReplicaSet: 2个Pod（创建中）← maxSurge允许超出2个
总Pod数: 12（达到最大值）
等待新Pod Ready...

步骤2：新Pod Ready，删除旧Pod
旧ReplicaSet: 9个Pod ← 删除1个
新ReplicaSet: 2个Pod（Ready）
可用Pod数: 11（旧9+新2）

步骤3：继续创建新Pod
旧ReplicaSet: 9个Pod
新ReplicaSet: 3个Pod（2 Ready + 1创建中）
总Pod数: 12
等待新Pod Ready...

步骤4：新Pod Ready，删除旧Pod
旧ReplicaSet: 8个Pod ← 删除1个
新ReplicaSet: 3个Pod（Ready）
可用Pod数: 11

...重复上述过程...

最终状态：
旧ReplicaSet: 0个Pod
新ReplicaSet: 10个Pod（全部Ready）
可用Pod数: 10
```

**参数配置的影响**：

- `maxSurge=0, maxUnavailable=1`：一次只替换一个Pod，最保守，更新最慢
- `maxSurge=1, maxUnavailable=0`：先创建新Pod再删除旧Pod，始终保持期望副本数可用，适合流量敏感的场景
- `maxSurge=50%, maxUnavailable=0`：快速创建大量新Pod，适合资源充足且希望快速更新的场景
- `maxSurge=1, maxUnavailable=1`：均衡策略，Kubernetes的默认值

**百分比计算的细节**：

如果期望副本数是3，maxSurge=50%，计算结果是3 * 0.5 = 1.5。Kubernetes会向上取整，结果是2。这意味着最多可以有5个Pod（3 + 2）。

如果maxUnavailable=50%，计算结果同样是1.5，但这里会向下取整，结果是1。这意味着最少要有2个可用Pod（3 - 1）。

为什么取整方向不同？因为maxSurge向上取整会让更新更快（多创建Pod），maxUnavailable向下取整会让服务更稳定（少删除Pod）。这是一种保守的设计。

### 滚动更新过程中Pod的生命周期

在滚动更新过程中，Pod的创建和销毁有明确的时序，理解这个时序对排查问题很重要。

**旧Pod的终止流程**：

1. Deployment Controller决定缩容旧ReplicaSet
2. ReplicaSet Controller选择一个Pod标记为Terminating
3. Pod从Endpoint移除（通过Readiness探针失败或直接移除）
4. 执行preStop hook（如果配置了）
5. 发送SIGTERM信号给容器
6. 等待容器优雅退出（最多terminationGracePeriodSeconds）
7. 如果超时，发送SIGKILL强制杀死
8. Pod删除

**新Pod的启动流程**：

1. Deployment Controller决定扩容新ReplicaSet
2. ReplicaSet Controller创建新Pod
3. Scheduler调度Pod到节点
4. kubelet拉取镜像（如果节点没有缓存）
5. 创建容器并启动
6. 执行postStart hook（如果配置了）
7. 等待Startup探针成功（如果配置了）
8. 开始执行Liveness和Readiness探针
9. Readiness探针成功，Pod加入Endpoint
10. 开始接收流量

**关键的时序重叠**：

Deployment Controller在决定下一步动作时，会检查当前有多少个Ready的Pod。只有当Ready的Pod数量满足`期望副本数 - maxUnavailable`时，才会继续删除旧Pod。只有当总Pod数量小于`期望副本数 + maxSurge`时，才会继续创建新Pod。

这意味着新Pod必须通过Readiness探针才能被计入Ready数量。如果新Pod的Readiness探针配置不当（比如initialDelaySeconds太短，应用还没启动完成），会导致滚动更新卡住——新Pod一直无法Ready，Deployment Controller不会继续更新。

### Deployment的版本历史与回滚原理

Deployment会保留历史版本的ReplicaSet，这是实现快速回滚的基础。

```
Deployment的版本历史：
┌──────────────────────────────────────┐
│ Deployment: my-app                   │
│ spec.revisionHistoryLimit: 10        │
└───────────┬──────────────────────────┘
            │ 保留历史
            ↓
┌──────────────────────────────────────┐
│ ReplicaSet-v1 (revision 1)           │
│ replicas: 0                          │
│ image: app:v1.0                      │
├──────────────────────────────────────┤
│ ReplicaSet-v2 (revision 2)           │
│ replicas: 0                          │
│ image: app:v2.0                      │
├──────────────────────────────────────┤
│ ReplicaSet-v3 (revision 3) ← 当前版本 │
│ replicas: 3                          │
│ image: app:v3.0                      │
└──────────────────────────────────────┘
```

`spec.revisionHistoryLimit`定义了保留多少个旧ReplicaSet，默认是10。超过限制的旧ReplicaSet会被自动删除。

**回滚的底层实现**：

当执行`kubectl rollout undo`时，Deployment Controller并不是重新创建Pod，而是直接操作ReplicaSet的副本数：

1. 找到目标版本的ReplicaSet（比如revision 2）
2. 增加目标ReplicaSet的副本数（从0到期望值）
3. 减少当前ReplicaSet的副本数（从期望值到0）
4. 遵循maxSurge和maxUnavailable的约束
5. 完成后，目标ReplicaSet成为当前版本

这个过程和正常的滚动更新完全一样，只是方向相反。因为旧ReplicaSet已经存在，不需要重新创建，回滚非常快速。

**回滚的关键限制**：

- 如果旧ReplicaSet已被删除（超过revisionHistoryLimit或手动删除），无法回滚到该版本
- 回滚只能恢复Pod模板（镜像、环境变量等），无法恢复ConfigMap或Secret的内容
- 如果ConfigMap或Secret也变化了，需要手动恢复它们，否则回滚的Pod可能使用新的配置

**查看和管理版本历史**：

```bash
# 查看历史版本
kubectl rollout history deployment/my-app

# 查看特定版本的详细信息
kubectl rollout history deployment/my-app --revision=2

# 回滚到上一个版本
kubectl rollout undo deployment/my-app

# 回滚到指定版本
kubectl rollout undo deployment/my-app --to-revision=2

# 暂停滚动更新（用于金丝雀发布）
kubectl rollout pause deployment/my-app

# 恢复滚动更新
kubectl rollout resume deployment/my-app
```

## 流量切换方案

渐进式迁移的核心是流量分配机制。需要一个"旋钮"来精确控制有多少流量进入Kubernetes，多少流量留在旧环境。

### DNS切换的TTL机制

DNS切换是最简单的流量切换方式，但它的行为受DNS缓存机制的影响。

**DNS解析的层级缓存**：

```
用户请求 app.example.com
    ↓
浏览器DNS缓存（受TTL控制）
    ↓
操作系统DNS缓存（受TTL控制）
    ↓
ISP的DNS服务器（受TTL控制）
    ↓
权威DNS服务器（返回IP，设置TTL）
```

当你修改DNS记录，将`app.example.com`从旧环境IP改为Kubernetes Ingress IP时，这个变化不会立即生效。变化的传播速度取决于TTL（Time To Live）。

假设TTL设置为300秒（5分钟）：

```
T0时刻：修改DNS记录
  旧环境IP: 192.168.1.100
  新环境IP: 10.0.0.50 ← 刚修改为这个

T1时刻（1分钟后）：
  已经过了缓存刷新周期的客户端开始解析到新IP
  但大部分客户端仍然缓存着旧IP

T5时刻（5分钟后）：
  TTL过期，大部分客户端的缓存失效
  新的DNS查询返回新IP

T10时刻（10分钟后）：
  基本完成切换，但仍可能有长时间缓存的客户端
```

**DNS切换的问题**：

1. **传播延迟不可控**：无法确切知道何时所有客户端完成切换
2. **无法精确控制比例**：不能实现"50%流量到新环境"这种精确控制
3. **回滚同样有延迟**：如果新环境有问题，切回旧环境同样需要等待TTL
4. **客户端可能不遵守TTL**：有些客户端或中间设备会忽略TTL，长时间缓存DNS结果

**优化DNS切换的策略**：

- 迁移前降低TTL：提前几天将TTL从3600秒降到60秒，让客户端适应短TTL
- 监控双边流量：在旧环境和新环境同时监控流量，观察切换的实际进度
- 保持旧环境运行：切换后至少等待10倍TTL时间（如TTL=300秒，等待50分钟）再考虑下线旧环境

**适用场景**：DNS切换适合对切换速度要求不高、能容忍一定时间窗口内新旧环境共存的场景，比如内部系统或对实时性要求不高的应用。

### Nginx upstream权重的负载均衡原理

在前端Nginx中配置upstream，将旧环境和Kubernetes同时作为后端，通过weight参数控制流量分配比例。

```nginx
upstream backend {
    server 192.168.1.100:8080 weight=90;   # 旧环境 90%
    server 10.0.0.50:8080     weight=10;   # Kubernetes 10%
}

server {
    listen 80;
    server_name app.example.com;

    location / {
        proxy_pass http://backend;
    }
}
```

**Nginx weight的工作原理**：

Nginx的权重负载均衡不是严格的百分比分配，而是基于加权轮询（Weighted Round Robin）算法。算法的核心是每个后端维护一个当前权重值，每次请求时选择当前权重最高的后端，并调整权重值。

每次请求到来时，算法为每个后端增加其权重值到当前权重，然后选择当前权重最高的后端，并减去总权重。经过足够的请求后，各后端的请求比例会精确符合权重设定。这个算法保证了长期的比例正确性，同时分布均匀（不会连续发送90个请求到A再发送10个到B）。

**连接级 vs 请求级的负载均衡**：

Nginx的upstream是在请求级别做负载均衡的。每个HTTP请求独立选择后端。即使客户端使用keep-alive保持连接，每个请求仍然可能被路由到不同的后端。

这与连接级负载均衡（如四层负载均衡器）不同。连接级负载均衡在建立TCP连接时选择后端，该连接的所有请求都发往同一个后端。

**Nginx方案的优势**：流量比例精确可控、实时生效（reload即可）、支持基于Header/Cookie的复杂路由、不依赖DNS。

**Nginx方案的劣势**：需要额外维护一层Nginx（需做高可用）、如果已有云负载均衡器会形成两层负载均衡。

如果应用需要会话保持，可以使用`ip_hash`指令，但这会破坏权重的精确控制。更好的方案是应用层无状态化（会话外部化到Redis）。

### Ingress灰度发布的实现原理

如果流量已经进入Kubernetes，可以利用Ingress Controller的金丝雀能力在集群内部做流量分配。以Nginx Ingress Controller为例。

```yaml
# 主Ingress（稳定版本）
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app-stable
spec:
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-app-v1
            port:
              number: 8080
---
# 金丝雀Ingress（新版本）
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app-canary
  annotations:
    nginx.ingress.kubernetes.io/canary: "true"
    nginx.ingress.kubernetes.io/canary-weight: "10"
spec:
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-app-v2
            port:
              number: 8080
```

**Nginx Ingress Controller的canary实现**：

Ingress Controller监听到带有`canary: "true"`注解的Ingress时，会在生成的Nginx配置中创建特殊的location块，使用Nginx的`split_clients`模块或Lua脚本实现流量分配。

生成的Nginx配置（简化版）：

```nginx
# 根据随机数决定路由
split_clients $request_id $canary_backend {
    10%     "canary";   # 10%流量到金丝雀
    *       "stable";   # 其余流量到稳定版本
}

location / {
    if ($canary_backend = "canary") {
        proxy_pass http://my-app-v2-service;
    }
    proxy_pass http://my-app-v1-service;
}
```

**支持的金丝雀策略**：

1. **基于权重**：`canary-weight: "10"`，10%的流量到金丝雀
2. **基于Header**：`canary-by-header: "x-canary"，canary-by-header-value: "true"`，只有Header中包含`x-canary: true`的请求到金丝雀
3. **基于Cookie**：`canary-by-cookie: "canary"`，只有Cookie中包含`canary=always`的请求到金丝雀

这些策略可以组合使用，比如先用Header策略让内部测试流量进入金丝雀，验证没问题后再用权重策略让真实用户流量渐进进入。

**Ingress灰度与滚动更新的区别**：

- 滚动更新是Deployment内部的版本替换，所有流量都进入同一个Service，只是Service背后的Pod逐步从v1替换为v2
- Ingress灰度是在Service层面做流量分配，v1和v2是两个独立的Service（可能背后是两个Deployment）

Ingress灰度适合这样的场景：应用已经在Kubernetes中，需要测试新版本，希望只有部分流量进入新版本，验证没问题后再全量。

**局限性**：

Ingress灰度主要适用于应用升级，而不是从外部环境迁移到Kubernetes。因为Ingress只能控制进入Kubernetes后的流量分配，无法控制进入Kubernetes之前的流量。

对于从外部迁移到Kubernetes的场景，通常需要在Ingress之前（比如云负载均衡器或前端Nginx）做流量分配。

### 服务网格的流量管理能力

服务网格（如Istio）通过在每个Pod旁边注入Envoy代理（Sidecar）拦截流量，提供了最精细的流量管理能力。通过VirtualService和DestinationRule，可以实现基于Header、权重、Cookie等多维度的流量路由，支持A/B测试、金丝雀发布、流量镜像等高级策略。

但Istio也带来了显著的复杂度：学习曲线陡峭、每个Pod多一个Sidecar增加资源开销、调试困难。**如果团队刚开始使用Kubernetes，不建议立即引入Istio**，先掌握Kubernetes本身的能力。等应用数量多、服务间调用复杂时，再考虑引入。

## CI/CD流水线改造

迁移到Kubernetes后，部署方式发生了根本性变化，CI/CD流水线需要相应改造。

### 从推模式到拉模式

传统的CI/CD流水线是"推模式"（Push Model）：CI系统构建完成后，主动连接到生产环境（SSH到服务器或调用API），推送代码或配置，执行部署操作。

```
传统推模式：
CI服务器（有生产环境凭证）
    ↓ SSH或API调用
生产服务器
    ↓ 执行部署脚本
应用更新
```

这种模式的问题：
- CI系统需要有生产环境的访问权限，安全风险高
- 部署过程由CI系统控制，生产环境是被动的
- 网络连接问题可能导致部署失败
- 难以审计：谁在什么时候部署了什么

Kubernetes和GitOps引入了"拉模式"（Pull Model）：生产环境的Operator持续监听Git仓库或镜像仓库，发现变化后主动拉取并应用，CI系统只负责构建和推送，不直接操作生产环境。

```
GitOps拉模式：
CI服务器（无生产环境凭证）
    ↓ 推送镜像到仓库
镜像仓库
    ↑ 定期轮询或Webhook通知
ArgoCD/FluxCD（运行在集群内）
    ↓ 应用到集群
Kubernetes集群
```

**拉模式的优势**：

- CI系统不需要生产环境凭证，降低安全风险
- Git仓库是单一事实来源（Single Source of Truth），所有配置变更都有记录
- 部署是声明式的，Operator确保集群状态与Git仓库一致
- 易于审计和回滚：Git的提交历史就是部署历史

**从推模式迁移到拉模式**：

传统模式：

```groovy
// Jenkinsfile
stage('Deploy') {
    steps {
        sshagent(['prod-server-key']) {
            sh '''
                scp target/app.jar prod-server:/opt/app/
                ssh prod-server "systemctl restart app"
            '''
        }
    }
}
```

Kubernetes推模式（过渡阶段）：

```groovy
// Jenkinsfile
stage('Deploy') {
    steps {
        withKubeConfig([credentialsId: 'k8s-prod']) {
            sh '''
                kubectl set image deployment/my-app \
                    app=${IMAGE_TAG}
            '''
        }
    }
}
```

GitOps拉模式（推荐）：

```groovy
// Jenkinsfile
stage('Update Manifest') {
    steps {
        git credentialsId: 'github', url: 'https://github.com/org/k8s-manifests'
        sh '''
            sed -i "s|image: .*|image: ${IMAGE_TAG}|" \
                deployment/my-app/deployment.yaml
            git add .
            git commit -m "Update my-app to ${IMAGE_TAG}"
            git push
        '''
    }
}
// ArgoCD自动检测到Git变化，更新集群
```

### 镜像标签策略的设计原则

镜像标签是连接代码版本和运行实例的桥梁，设计不当会导致无法追溯版本、无法精确回滚、甚至部署错误的版本。

**反模式：使用latest标签**

```dockerfile
# 构建镜像
docker build -t myapp:latest .
docker push myapp:latest

# Deployment引用
containers:
- image: myapp:latest
```

这种做法的问题：
- `latest`没有明确的版本信息，无法追溯到具体的代码提交
- 不同环境可能拉取到不同的`latest`镜像（构建时间不同）
- Kubernetes默认的`imagePullPolicy`对`latest`标签是Always，每次都拉取，但无法确定拉取到的是哪个版本
- 回滚时无法指定回滚到哪个版本
- 无法并行运行多个版本（金丝雀发布需要v1和v2同时运行）

**推荐：语义化版本标签**

基本格式：`<分支>-<commit-sha>-<构建时间>`

示例：`main-a3f8c2d-20260209143000`

```bash
# CI脚本中生成标签
BRANCH=$(git rev-parse --abbrev-ref HEAD | sed 's/\//-/g')
COMMIT_SHA=$(git rev-parse --short HEAD)
BUILD_TIME=$(date +%Y%m%d%H%M%S)
IMAGE_TAG="${REGISTRY}/myapp:${BRANCH}-${COMMIT_SHA}-${BUILD_TIME}"

docker build -t ${IMAGE_TAG} .
docker push ${IMAGE_TAG}
```

**标签的各部分含义**：

- **分支名**：快速识别来自哪个分支（main、develop、feature-xxx）
- **commit SHA**：精确追溯到代码版本，可以`git checkout <sha>`查看代码
- **构建时间**：帮助理解镜像的新旧，便于排序

**特殊环境的标签策略**：

开发环境可以使用：`develop-latest`（每次构建覆盖，快速迭代）

测试环境使用：`develop-<commit-sha>`（每次构建新标签，方便测试不同版本）

生产环境使用：`main-<commit-sha>-<build-time>`（完整信息，便于追溯和审计）

同时可以创建额外的版本标签：

```bash
# 创建语义化版本标签
docker tag ${IMAGE_TAG} ${REGISTRY}/myapp:v1.2.3
docker push ${REGISTRY}/myapp:v1.2.3
```

这样每个镜像有两个标签：
- 详细标签：`main-a3f8c2d-20260209143000`（用于追溯）
- 版本标签：`v1.2.3`（用于发布和沟通）

### 镜像不可变性原则

镜像一旦构建并打上标签，不应该被修改或覆盖。这是容器化的核心原则之一。

镜像一旦构建并打上标签，不应该被覆盖。覆盖标签会导致不同节点拉取到不同版本、无法回滚到真正的上一版本等问题。正确做法是每次构建生成唯一标签（如`myapp:1.0-build-123`）。

不可变性的好处：确定性（相同标签指向相同内容）、可追溯、可重复、易于回滚。

**Kubernetes的imagePullPolicy**也与此相关：`latest`标签默认为`Always`（每次拉取），其他标签默认为`IfNotPresent`（有缓存就用缓存）。如果覆盖了标签，现有Pod不会自动更新，可能导致同一Deployment的不同Pod运行不同版本的代码。

### 镜像安全扫描

镜像安全扫描用于在部署前发现安全漏洞。工具如Trivy、Clair等，工作流程是：解压镜像层、识别OS和软件包、对比CVE数据库、生成漏洞报告。

在CI/CD中集成扫描，使用`--exit-code 1`配置使CRITICAL/HIGH级别漏洞阻断部署。修复方法包括更新基础镜像、更新软件包、移除不必要的包。需要注意扫描只能发现已知漏洞，且应定期重新扫描生产镜像（漏洞数据库持续更新）。

## 可观测性架构

可观测性（Observability）是生产系统的眼睛，没有可观测性的迁移是盲飞。

### 可观测性的三大支柱

现代可观测性理论认为系统的可观测性由三个支柱构成：Metrics（指标）、Logging（日志）、Tracing（追踪）。

**Metrics（指标）**：
- 定义：随时间变化的数值数据，如CPU使用率、请求量、错误率
- 特点：结构化、可聚合、占用空间小
- 用途：监控系统健康状态、触发告警、容量规划
- 典型问题："为什么CPU突然升高？"、"错误率是否超过阈值？"

**Logging（日志）**：
- 定义：系统产生的离散事件记录，如请求日志、错误日志
- 特点：非结构化或半结构化、数据量大、保留时间有限
- 用途：问题排查、审计、理解系统行为
- 典型问题："用户ID 12345的请求为什么失败？"、"这个错误的详细堆栈是什么？"

**Tracing（追踪）**：
- 定义：请求在分布式系统中的完整调用链路
- 特点：有层级关系（Span tree）、采样（不记录所有请求）
- 用途：性能分析、依赖关系理解、瓶颈定位
- 典型问题："这个请求为什么慢？"、"时间花在哪个服务上？"

**三者的关系**：

```
用户请求
    ↓
生成Trace（追踪整个请求）
    ├─ Span1: API Gateway (10ms)
    ├─ Span2: Service A (50ms)
    │   └─ 记录Metrics: service_a_duration=50ms
    │   └─ 记录Logging: "Processed order 12345"
    └─ Span3: Service B (30ms)
        └─ 记录Metrics: service_b_duration=30ms
        └─ 记录Logging: "Inventory updated"
```

三者互补：
- Metrics告诉你有问题（错误率上升）
- Logging帮你理解问题（具体的错误信息）
- Tracing帮你定位问题（哪个服务慢）

### Prometheus的Pull模型原理

Prometheus是Kubernetes生态的标准监控方案，它采用Pull模型，这与传统的Push模型有本质区别。

**传统Push模型**：

```
应用主动推送指标
    ↓
监控系统被动接收
    ↓
存储和查询
```

示例：StatsD、InfluxDB的典型用法

```python
# 应用代码
statsd.increment('api.requests')
statsd.timing('api.duration', duration)
```

应用通过客户端库主动将指标推送到监控系统。

**Prometheus的Pull模型**：

```
应用暴露指标端点（HTTP）
    ↑
Prometheus主动拉取
    ↓
存储和查询
```

示例：

```python
# 应用代码
from prometheus_client import Counter, Histogram

request_count = Counter('api_requests_total', 'Total requests')
request_duration = Histogram('api_request_duration_seconds', 'Request duration')

@app.route('/api')
def api():
    request_count.inc()
    with request_duration.time():
        # 处理请求
        pass
```

应用暴露一个`/metrics`端点：

```
# HELP api_requests_total Total requests
# TYPE api_requests_total counter
api_requests_total 1234

# HELP api_request_duration_seconds Request duration
# TYPE api_request_duration_seconds histogram
api_request_duration_seconds_bucket{le="0.1"} 100
api_request_duration_seconds_bucket{le="0.5"} 500
api_request_duration_seconds_bucket{le="1.0"} 800
api_request_duration_seconds_sum 1234.56
api_request_duration_seconds_count 1000
```

Prometheus定期（默认15秒）访问这个端点，拉取指标。

**Pull模型的优势**：

1. **服务发现友好**：Prometheus可以通过Kubernetes API自动发现所有Pod，无需应用配置监控系统地址
2. **监控系统控制采集频率**：应用不需要关心多久发送一次指标，由Prometheus决定
3. **易于检测目标健康状态**：如果拉取失败，Prometheus知道目标可能有问题
4. **减少应用复杂度**：应用只需要暴露端点，不需要主动推送逻辑

**Pull模型的劣势**：

1. **短生命周期任务不适合**：如果Job运行30秒就结束，Prometheus可能来不及拉取
2. **网络限制**：Prometheus必须能访问所有目标，有防火墙限制时比较麻烦
3. **推送批量指标困难**：如果要推送历史数据或批量指标，Pull模型不适用

对于短生命周期任务，Prometheus提供了Pushgateway作为补充，应用推送指标到Pushgateway，Prometheus从Pushgateway拉取。

### ServiceMonitor的服务发现机制

Kubernetes环境中，Pod会频繁创建和销毁，IP地址会变化。Prometheus如何自动发现这些动态的目标？

**Prometheus Operator + ServiceMonitor**：

Prometheus Operator是一个Kubernetes Operator，它引入了ServiceMonitor这个CRD来声明式地定义监控目标。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-app
  labels:
    app: my-app
spec:
  selector:
    app: my-app
  ports:
  - name: metrics
    port: 8080
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: my-app
  labels:
    team: backend
spec:
  selector:
    matchLabels:
      app: my-app
  endpoints:
  - port: metrics
    path: /metrics
    interval: 30s
```

**工作流程**：

1. ServiceMonitor通过selector选择Service
2. Service通过selector选择Pod
3. Prometheus Operator监听ServiceMonitor的变化
4. 生成Prometheus的配置文件（scrape_configs）
5. Reload Prometheus
6. Prometheus根据配置拉取Pod的metrics端点

**动态更新**：

当Pod扩缩容时：
1. Deployment创建/删除Pod
2. Endpoints更新（加入或移除Pod IP）
3. Prometheus通过Kubernetes API watch Endpoints变化
4. 自动添加或移除scrape目标

整个过程自动化，无需手动配置。

**标签和注解的作用**：

Prometheus会自动添加一些标签到抓取的指标：

```
api_requests_total{
    pod="my-app-7f9c8-xk2p",
    namespace="default",
    service="my-app",
    instance="10.244.1.5:8080"
}
```

这些标签可以用于聚合查询：

```promql
# 每个Pod的请求量
sum(rate(api_requests_total[5m])) by (pod)

# 整个Service的请求量
sum(rate(api_requests_total[5m])) by (service)

# 整个Namespace的请求量
sum(rate(api_requests_total[5m])) by (namespace)
```

### 容器日志收集的底层原理

容器的日志管理与传统应用有很大不同，理解底层机制有助于设计日志方案。

**容器日志的生命周期**：

1. **应用写入stdout/stderr**：应用调用`print()`、`console.log()`、`logger.info()`等，输出到标准输出
2. **容器运行时捕获**：容器运行时（containerd、CRI-O）捕获这些输出
3. **写入节点日志文件**：存储到节点的文件系统，通常是`/var/log/containers/`或`/var/log/pods/`
4. **日志采集器收集**：DaemonSet形式的采集器（Fluent Bit、Filebeat）读取这些文件
5. **发送到中心化存储**：采集器将日志发送到Elasticsearch、Loki等
6. **用户查询**：通过Kibana、Grafana等UI查询日志

**日志文件的存储路径**：

```bash
# 容器运行时的日志文件
/var/log/containers/<pod-name>_<namespace>_<container-name>-<container-id>.log

# 实际是一个软链接，指向
/var/log/pods/<namespace>_<pod-name>_<pod-uid>/<container-name>/<restart-count>.log

# 最终指向容器运行时的存储
/var/lib/containerd/io.containerd.grpc.v1.cri/containers/<container-id>/...
```

**日志格式**：

容器运行时会添加元数据到每行日志：

```json
{
    "log": "2024-01-01 10:00:00 INFO Request processed\n",
    "stream": "stdout",
    "time": "2024-01-01T10:00:00.123456789Z"
}
```

**日志采集器的工作方式**：

以Fluent Bit为例（DaemonSet部署）：

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluent-bit
spec:
  template:
    spec:
      containers:
      - name: fluent-bit
        image: fluent/fluent-bit:2.0
        volumeMounts:
        - name: varlog
          mountPath: /var/log
          readOnly: true
        - name: varlibcontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibcontainers
        hostPath:
          path: /var/lib/docker/containers
```

Fluent Bit挂载节点的`/var/log`目录，读取容器日志文件，解析JSON格式，提取Pod、Namespace等元数据，添加到日志中，发送到Elasticsearch。

**日志轮转和清理**：

容器日志文件会无限增长吗？不会，kubelet会自动轮转和清理：

- 单个容器日志文件超过10MB时，会被轮转（重命名并压缩）
- 最多保留5个轮转文件
- Pod删除时，日志文件也会被删除

这些参数可以通过kubelet配置调整：

```yaml
# kubelet配置
containerLogMaxSize: 10Mi
containerLogMaxFiles: 5
```

**kubectl logs的工作原理**：

当你执行`kubectl logs my-pod`时：

1. kubectl发送请求到API Server
2. API Server转发请求到Pod所在节点的kubelet
3. kubelet读取容器日志文件
4. 返回内容到kubectl

`kubectl logs`是实时查看的快捷方式，但只能看到最近的日志（受日志文件大小限制）。如果需要查询历史日志或聚合多个Pod的日志，必须使用中心化日志系统。

### 分布式追踪的核心概念

分布式追踪用于理解请求在微服务架构中的完整流转路径，对于性能优化和问题排查非常有价值。

**Trace、Span、SpanContext的关系**：

```
一个Trace代表一个完整的请求：
┌─────────────────────────────────────────────────────────┐
│ Trace ID: abc123                                        │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │ Span: API Gateway                               │   │
│  │ Start: 0ms, Duration: 200ms                     │   │
│  └──┬──────────────────────────────────────────────┘   │
│     │                                                   │
│     ├─ ┌─────────────────────────────────────────┐     │
│     │  │ Span: Service A                         │     │
│     │  │ Start: 10ms, Duration: 100ms            │     │
│     │  │ Parent: API Gateway                     │     │
│     │  └──┬──────────────────────────────────────┘     │
│     │     │                                             │
│     │     └─ ┌────────────────────────────────┐        │
│     │        │ Span: Database Query           │        │
│     │        │ Start: 20ms, Duration: 50ms    │        │
│     │        │ Parent: Service A              │        │
│     │        └────────────────────────────────┘        │
│     │                                                   │
│     └─ ┌─────────────────────────────────────────┐     │
│        │ Span: Service B                         │     │
│        │ Start: 120ms, Duration: 80ms            │     │
│        │ Parent: API Gateway                     │     │
│        └─────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────┘
```

**Trace ID的传播**：分布式追踪的关键是Trace ID必须通过HTTP Header在所有服务间传播，这样就可以根据Trace ID查询到整个请求链路的所有日志和Span。

**采样策略**：记录所有Trace会产生海量数据，通常需要采样。常见策略包括固定比例采样（随机10%）、基于延迟的采样（只采样慢请求）、基于错误的采样（采样所有失败请求）。

**技术选型**：推荐使用OpenTelemetry SDK（统一的可观测性标准），数据可以发送到Jaeger、Zipkin等后端。

**应用接入追踪**：

```go
// Go示例
import "go.opentelemetry.io/otel"

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // 创建Span
    ctx, span := otel.Tracer("my-service").Start(r.Context(), "handleRequest")
    defer span.End()

    // 调用其他服务时传播Context
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://service-b", nil)
    resp, _ := http.DefaultClient.Do(req)
}
```

框架会自动注入Trace ID到HTTP Header，下游服务提取并继续传播。

### 迁移阶段的关键监控指标

迁移过程中需要重点监控哪些指标，如何定义告警规则？

**对比监控（最重要）**：

迁移的核心是验证Kubernetes环境和旧环境的行为一致。需要同时监控两个环境的指标，并对比：

```
指标维度              旧环境    Kubernetes   差异
─────────────────────────────────────────────
请求量（QPS）          1000      100         -
错误率（%）            0.5       0.6         +0.1%
P50延迟（ms）          50        52          +2ms
P99延迟（ms）          200       210         +10ms
CPU使用率（%）         40        45          +5%
内存使用（MB）         800       820         +20MB
```

如果差异在可接受范围内（比如P99延迟差异小于10%），说明Kubernetes环境运行正常，可以继续放量。

**关键指标清单**：

1. **应用层指标**：
   - 请求量（QPS）：`rate(http_requests_total[1m])`
   - 错误率：`rate(http_requests_total{status=~"5.."}[1m]) / rate(http_requests_total[1m])`
   - 延迟分位值：`histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m]))`

2. **Pod级指标**：
   - Pod重启次数：`kube_pod_container_status_restarts_total`
   - Pod状态：`kube_pod_status_phase{phase="Running"}`
   - 容器OOM次数：`kube_pod_container_status_terminated_reason{reason="OOMKilled"}`

3. **资源使用指标**：
   - CPU使用率：`rate(container_cpu_usage_seconds_total[1m])`
   - 内存使用：`container_memory_working_set_bytes`
   - CPU throttle：`rate(container_cpu_cfs_throttled_seconds_total[1m])`

4. **网络指标**：
   - 网络接收速率：`rate(container_network_receive_bytes_total[1m])`
   - 网络发送速率：`rate(container_network_transmit_bytes_total[1m])`
   - 连接数：`container_network_tcp_usage_total`

**告警规则设计**：

```yaml
# 错误率告警
alert: HighErrorRate
expr: |
  rate(http_requests_total{status=~"5.."}[5m])
  / rate(http_requests_total[5m]) > 0.01
for: 2m
annotations:
  summary: "错误率超过1%"

# Pod频繁重启告警
alert: PodCrashLooping
expr: rate(kube_pod_container_status_restarts_total[15m]) > 0
for: 5m
annotations:
  summary: "Pod在15分钟内重启过"

# OOM告警
alert: PodOOMKilled
expr: increase(kube_pod_container_status_terminated_reason{reason="OOMKilled"}[5m]) > 0
annotations:
  summary: "Pod被OOM Killed"

# CPU throttle告警
alert: HighCPUThrottle
expr: |
  rate(container_cpu_cfs_throttled_seconds_total[5m])
  / rate(container_cpu_cfs_periods_total[5m]) > 0.25
for: 10m
annotations:
  summary: "CPU限流超过25%"
```

**迁移阶段的告警策略**：

- 初期（5%流量）：告警阈值设置得比较敏感，任何异常都告警
- 中期（50%流量）：告警阈值稍微放宽，只告警明显的问题
- 后期（100%流量）：告警阈值与旧环境一致

**监控看板设计**：

建立迁移专用的Grafana看板，包含：
- 顶部：流量分配比例（旧环境 vs Kubernetes）
- 第一行：请求量、错误率、延迟的对比图（两条线，一条旧环境，一条Kubernetes）
- 第二行：Pod状态、重启次数、资源使用
- 第三行：详细的指标（按Service、按Endpoint）

这个看板在迁移期间是团队的"作战地图"，每次放量前后都要仔细观察。

## 回滚预案

回滚能力是迁移信心的基础。一个好的回滚预案能让团队在出现问题时快速恢复，而不是手忙脚乱。

### 应用级回滚：Deployment的版本管理

Kubernetes内置了应用级回滚能力，基于Deployment的ReplicaSet版本历史。

**回滚命令**：

```bash
# 查看部署历史
kubectl rollout history deployment/my-app

# 输出示例
REVISION  CHANGE-CAUSE
1         Initial deployment
2         Update to v2.0
3         Update to v3.0

# 查看特定版本的详细信息
kubectl rollout history deployment/my-app --revision=2

# 回滚到上一个版本
kubectl rollout undo deployment/my-app

# 回滚到指定版本
kubectl rollout undo deployment/my-app --to-revision=2
```

**回滚的底层操作**：

当执行回滚时，Deployment Controller会：
1. 找到目标revision对应的ReplicaSet
2. 将目标ReplicaSet的副本数从0增加到期望值
3. 将当前ReplicaSet的副本数从期望值减少到0
4. 遵循maxSurge和maxUnavailable的配置
5. 完成后，目标ReplicaSet成为当前版本，revision号递增

回滚与正常滚动更新使用同一机制，速度取决于Pod启动速度和maxSurge/maxUnavailable配置。如果需要更快回滚，可以临时将maxSurge调高。

**回滚的局限性**：

- 只能回滚Deployment的Pod模板，无法回滚ConfigMap、Secret、Service等其他资源
- 如果ConfigMap或Secret也变化了，需要手动回滚它们
- 如果数据库Schema变化了，应用回滚可能失败（需要数据库向后兼容）

**建立回滚触发条件**：

明确定义什么情况下必须回滚：

| 指标 | 阈值 | 动作 |
|------|------|------|
| 错误率 | >1% | 立即回滚 |
| P99延迟 | 增加50%以上 | 立即回滚 |
| Pod重启次数 | 5分钟内重启3次以上 | 立即回滚 |
| OOM Killed | 任何Pod被OOM Killed | 立即回滚 |
| CPU throttle | 持续throttle 10分钟以上 | 评估后回滚 |

这些条件应该写入文档并自动化（通过告警规则触发回滚脚本）。

### 环境级回滚：流量切回旧环境

如果整个Kubernetes环境出现问题（不是单个应用的问题，而是集群级的问题），需要能够快速将流量切回旧环境。

当集群出现网络故障、存储异常、API Server不可用、大规模节点故障等问题时，需要环境级回滚。根据流量切换方案的不同，回滚方式也不同：

**DNS切换方案**：
```bash
# 修改DNS记录，指向旧环境IP
# 等待TTL过期（5-10分钟）
# 验证流量已切回旧环境
```

**Nginx upstream方案**：
```nginx
# 修改Nginx配置
upstream backend {
    server 192.168.1.100:8080 weight=100;   # 旧环境 100%
    server 10.0.0.50:8080     weight=0;     # Kubernetes 0%
}

# Reload Nginx（秒级生效）
nginx -s reload
```

**云负载均衡器方案**：
```bash
# 修改负载均衡器的后端池
# 移除Kubernetes Ingress的IP
# 添加旧环境的IP
# 通常在1分钟内生效
```

**环境级回滚的关键**：

- 旧环境必须保持运行状态，不能过早下线
- 旧环境必须能承载100%的流量（资源充足）
- 回滚操作必须经过演练，确保团队熟悉流程
- 有明确的决策流程（谁有权限决定环境级回滚）

**旧环境的保留期**：最少2周，推荐4周，最多3个月。保留期内旧环境保持可用但不接收流量，每周演练切流验证。保留期结束后才可下线。

### 数据回滚的考量

应用可以快速回滚，但数据的回滚往往更复杂。

**数据库Schema的向后兼容**：

假设v1.0的应用使用数据库表结构A，v2.0需要表结构B。如果直接升级到B，再回滚应用到v1.0，v1.0无法理解表结构B，会导致故障。

**安全的升级策略**：

```
第1步：部署v1.5（兼容版本）
- 代码同时支持表结构A和B
- 读写仍然使用表结构A

第2步：执行数据库迁移
- 修改表结构从A到B
- v1.5的代码仍然能正常运行

第3步：部署v2.0
- 代码使用表结构B
- 如果有问题，可以回滚到v1.5（仍然兼容B）

第4步：（可选）清理兼容代码
- 部署v2.1，移除对表结构A的支持
```

**数据回滚的场景**：

如果v2.0写入了错误数据，回滚应用到v1.0不能解决数据问题。需要：
1. 回滚应用到v1.0（停止写入错误数据）
2. 修复数据（SQL脚本、数据清洗工具）
3. 验证数据一致性

**建议**：

- 数据库变更和应用部署解耦（先改数据库，再部署应用）
- 数据库变更必须向后兼容
- 准备数据回滚脚本（在测试环境验证过）
- 重要数据变更前先备份

## 常见踩坑与最佳实践

迁移过程中有一些容易遇到的问题，提前了解可以避免很多弯路。

### 时区问题

容器默认使用UTC时区，而你的应用可能依赖本地时区（如Asia/Shanghai）。

**表现**：
- 日志时间戳错误（差8小时）
- 定时任务执行时间错误
- 业务逻辑中的时间比较出错

**根因**：
容器镜像通常基于最小化的Linux发行版，没有配置时区，默认UTC。应用读取系统时区时得到UTC。

**解决方案1：环境变量**

```yaml
env:
- name: TZ
  value: "Asia/Shanghai"
```

大部分应用和编程语言会尊重`TZ`环境变量。

**解决方案2：挂载时区文件**

```yaml
volumeMounts:
- name: timezone
  mountPath: /etc/localtime
  readOnly: true
volumes:
- name: timezone
  hostPath:
    path: /etc/localtime
```

这让容器使用宿主机的时区。

**最佳实践**：
- 应用内部使用UTC时间存储和计算
- 只在展示层转换为用户的本地时区
- 避免依赖系统时区

### DNS解析延迟

容器内DNS解析可能出现5秒超时，影响应用性能。

**根因**：
glibc的DNS解析器会同时发起A记录（IPv4）和AAAA记录（IPv6）查询。如果网络或DNS服务器有问题，其中一个查询可能超时（默认5秒），导致解析延迟。

**具体场景**：
```bash
# 在容器内
$ time nslookup mysql.default.svc.cluster.local
# 可能等待5秒才返回结果
```

**解决方案1：修改dnsConfig**

```yaml
dnsConfig:
  options:
  - name: single-request-reopen
  - name: ndots
    value: "2"
```

- `single-request-reopen`：在发送A和AAAA查询时使用不同的源端口，避免某些NAT设备的bug
- `ndots: 2`：减少DNS查询的尝试次数

**解决方案2：使用Alpine镜像**

Alpine使用musl libc而不是glibc，DNS解析器实现不同，不会同时查询A和AAAA。

但Alpine有其他问题（兼容性），需要权衡。

**解决方案3：应用层缓存**

在应用代码中缓存DNS解析结果：

```python
# Python示例
import dns.resolver
import functools
import time

@functools.lru_cache(maxsize=128)
def resolve_cached(hostname):
    return socket.gethostbyname(hostname)
```

### Java应用容器感知

JDK 8u131之前的JVM不识别容器的内存限制，按宿主机内存设置堆大小，导致OOM。

**问题场景**：
```yaml
resources:
  limits:
    memory: 1Gi  # 容器限制1GB
```

旧版JVM的行为：
```bash
# 宿主机有16GB内存
# JVM认为可用内存是16GB
# 默认堆大小 = 16GB / 4 = 4GB
# 实际容器限制只有1GB
# 结果：OOM Killed
```

**解决方案1：显式设置堆大小**

```yaml
env:
- name: JAVA_OPTS
  value: "-Xmx768m -Xms768m"
```

留一些空间给非堆内存（元空间、栈、直接内存）。

**解决方案2：使用容器感知参数**

JDK 8u191+：
```yaml
env:
- name: JAVA_OPTS
  value: "-XX:+UseContainerSupport -XX:MaxRAMPercentage=75.0"
```

JVM会读取容器的内存限制，堆大小 = 容器限制 * 75%。

**解决方案3：升级到JDK 11或更高版本**

JDK 11默认支持容器感知，不需要额外参数。

### 滚动更新时出现502

在滚动更新过程中，偶尔出现502错误。

**根因**：
Pod终止时的竞态条件（前面"优雅终止"章节详细解释过）：
1. Pod标记为Terminating
2. Endpoint开始移除Pod（需要传播时间）
3. 同时发送SIGTERM给容器
4. 应用收到SIGTERM立即退出
5. 但负载均衡器还没更新，仍在发送请求
6. 连接被拒绝，返回502

**解决方案**：

```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 5"]
terminationGracePeriodSeconds: 30
```

配合应用层的优雅关闭：

```go
// Go示例
func main() {
    srv := &http.Server{Addr: ":8080"}

    go func() {
        srv.ListenAndServe()
    }()

    // 监听SIGTERM
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, syscall.SIGTERM)
    <-stop

    // 停止接受新请求，等待现有请求完成
    ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}
```

### 文件描述符和连接数限制

容器内默认的ulimit可能比宿主机低，导致"too many open files"错误。

**查看限制**：

```bash
# 在容器内
$ ulimit -n
1024

# 在宿主机
$ ulimit -n
65535
```

**问题**：
如果应用需要维持大量连接（如数据库连接池、HTTP连接池），1024可能不够。

**解决方案**：

在Pod的securityContext中设置：

```yaml
securityContext:
  # 不推荐：需要特权
  privileged: true

# 推荐：只调整需要的资源限制
containers:
- name: app
  resources:
    limits:
      # 注意：这不是ulimit
  securityContext:
    # Kubernetes不直接支持设置ulimit
    # 需要在容器entrypoint中设置
```

在容器的entrypoint脚本中：

```bash
#!/bin/sh
ulimit -n 65535
exec java -jar app.jar
```

或在Dockerfile中：

```dockerfile
RUN echo "* soft nofile 65535" >> /etc/security/limits.conf && \
    echo "* hard nofile 65535" >> /etc/security/limits.conf
```

**注意**：
调整ulimit需要容器有相应的权限。如果使用了严格的PodSecurityPolicy或SecurityContext，可能无法调整。这种情况下需要修改节点的默认配置。

### 最佳实践清单

**迁移前**：
- [ ] 完成容器化改造，镜像通过安全扫描
- [ ] 在测试环境验证所有功能
- [ ] 压测验证性能（至少达到旧环境的80%）
- [ ] 建立监控和告警（覆盖新旧环境）
- [ ] 准备回滚预案并演练
- [ ] 团队培训（Kubernetes基础、故障排查）

**迁移中**：
- [ ] 第一批选择非核心、低风险的应用
- [ ] 初始流量比例不超过5%
- [ ] 每个阶段至少观察24小时（覆盖高峰和低谷）
- [ ] 定义明确的放量标准（错误率、延迟、资源使用）
- [ ] 出现问题立即回滚，分析后再重试
- [ ] 保持旧环境运行，随时可以接管流量

**迁移后**：
- [ ] 持续监控至少2周
- [ ] 定期演练回滚流程（每周一次）
- [ ] 优化资源配置（根据实际使用调整requests和limits）
- [ ] 建立运维文档（部署流程、故障排查）
- [ ] 培养团队的Kubernetes运维能力
- [ ] 至少4周后再考虑下线旧环境

## 小结

这是单体服务迁移至Kubernetes系列的最后一篇，至此我们完成了从评估到生产的完整旅程。

**第一篇：迁移评估与容器化**
- 理解容器的底层原理（Namespace、Cgroups、联合文件系统）
- 评估应用的迁移难度（状态性、依赖、配置）
- 遵循12-Factor原则改造应用
- 编写高质量的Dockerfile（多阶段构建、安全性、镜像精简）

**第二篇：资源规划与部署设计**
- 选择合适的资源对象（Deployment vs StatefulSet）
- 理解调度器原理和QoS等级
- 配置资源requests和limits（CPU、内存的差异）
- 设计Service和Ingress（理解kube-proxy的工作原理）
- 配置健康检查探针（Liveness、Readiness、Startup）
- 实现优雅终止（理解endpoint更新的竞态条件）

**第三篇：迁移策略与生产实践**
- 选择迁移策略（绞杀者模式、蓝绿部署、金丝雀发布）
- 理解滚动更新的底层机制（Deployment Controller、maxSurge/maxUnavailable）
- 实施流量切换（DNS、Nginx权重、Ingress灰度）
- 改造CI/CD流水线（推模式到拉模式、镜像标签策略、安全扫描）
- 建立可观测性（Metrics、Logging、Tracing三大支柱）
- 准备回滚预案（应用级、环境级、数据级）
- 避免常见踩坑（时区、DNS、JVM、优雅终止、资源限制）

**迁移的本质是风险管理**。技术问题都有解决方案，真正的挑战是在创新和稳定之间找到平衡。渐进式迁移、充分的监控、完善的回滚预案，这些都是为了控制风险。

**迁移不是终点，而是起点**。迁移到Kubernetes后，团队才真正开始云原生的旅程。后续还有很多工作：资源优化（垂直扩缩容、节点自动扩缩容）、安全加固（网络策略、Pod安全策略、RBAC）、高可用架构（多可用区部署、灾备）、成本优化（Spot实例、资源超售）、GitOps实践（自动化发布、配置管理）。

但不要急于求成。Kubernetes是一个复杂的系统，需要时间积累经验。先把基础打牢，理解核心概念和机制，然后逐步探索高级特性。这个系列希望能为你的Kubernetes之旅打下坚实的基础。

## 常见问题

### Q1: 迁移过程中新旧环境的数据库如何保持一致？

如果新旧环境共享同一个数据库实例，天然保持一致，这是最简单也是推荐的做法。

如果必须使用不同的数据库实例，需要在数据库层面实现同步。常见方案包括：主从复制（旧环境为主库，新环境为从库）、CDC工具（Debezium、Canal捕获binlog实时同步）。不推荐双写方案，一致性问题难以处理。

**最佳实践**：迁移初期强烈建议共享数据库，减少复杂度。数据迁移应在应用迁移完成后单独进行，需要充分的演练和验证。

### Q2: 灰度期间如何保证用户会话的一致性？

如果应用的会话存储在内存中，灰度期间用户可能在新旧环境之间切换，导致会话丢失。

**问题场景**：
```
T0: 用户登录，请求路由到旧环境，session存储在旧环境的内存
T1: 用户下一个请求，路由到新环境（10%概率）
T2: 新环境没有session，用户被要求重新登录
```

**解决方案1：会话外部化（推荐）**

将会话存储在Redis等共享存储中：

```java
// Spring Session + Redis
@EnableRedisHttpSession
public class SessionConfig {
    // Spring Boot会自动配置
    // 会话数据存储在Redis中
    // 新旧环境共享同一个Redis
}
```

这样无论请求路由到哪个环境，都能获取到正确的会话。这也是容器化改造的要求（进程无状态）。

**解决方案2：会话亲和（不推荐）**

在负载均衡器层面配置会话保持，同一个用户的请求总是路由到同一个环境：

```nginx
upstream backend {
    ip_hash;  # 根据客户端IP哈希
    server old-env weight=90;
    server new-env weight=10;
}
```

但这会破坏灰度的精确性（不是10%的请求到新环境，而是10%的用户到新环境）。

**解决方案3：无会话设计**

使用JWT等无状态认证：

```
用户登录 → 返回JWT Token
后续请求 → 携带JWT Token
服务端验证 → 不需要查询会话存储
```

JWT自包含用户信息，不需要服务端存储会话，天然适合分布式环境。

**建议**：
- 优先选择方案1（会话外部化），兼容性最好
- 方案3（无会话设计）是长期方向，但需要改造应用
- 避免方案2（会话亲和），它只是掩盖问题而不是解决问题

### Q3: CI/CD流水线中的镜像构建很慢怎么办？

镜像构建慢通常是因为没有充分利用Docker的层缓存。

**问题示例**：

```dockerfile
FROM maven:3.9-eclipse-temurin-21
WORKDIR /app
COPY . .                      # 复制所有文件
RUN mvn package               # 每次都重新下载依赖
```

每次代码变化，整个`COPY . .`层失效，后续的`RUN mvn package`也失效，需要重新下载几百MB的Maven依赖。

**优化策略1：分离依赖层**

```dockerfile
FROM maven:3.9-eclipse-temurin-21
WORKDIR /app

# 先复制依赖文件
COPY pom.xml .
RUN mvn dependency:go-offline  # 下载依赖，这一层可以缓存

# 再复制源码
COPY src ./src
RUN mvn package -DskipTests   # 只要pom.xml不变，依赖层就是缓存的
```

这样源码变化时，只需要重新编译，不需要重新下载依赖。

**优化策略2：使用BuildKit缓存挂载**

```dockerfile
# 启用BuildKit
# export DOCKER_BUILDKIT=1

FROM maven:3.9-eclipse-temurin-21
WORKDIR /app
COPY pom.xml .
RUN --mount=type=cache,target=/root/.m2 \
    mvn dependency:go-offline
COPY src ./src
RUN --mount=type=cache,target=/root/.m2 \
    mvn package
```

`--mount=type=cache`会在多次构建之间共享Maven本地仓库，即使镜像层失效，依赖也不需要重新下载。

**优化策略3：使用私有镜像仓库**

基础镜像（如`maven:3.9`）可能有数百MB，从Docker Hub下载很慢。建立私有镜像仓库（Harbor、Nexus），并在其中缓存常用基础镜像。

```dockerfile
# 使用私有仓库的基础镜像
FROM registry.company.com/maven:3.9-eclipse-temurin-21
```

**优化策略4：并行构建多架构镜像**

如果需要构建amd64和arm64两种架构：

```bash
# 串行构建（慢）
docker build --platform linux/amd64 -t app:amd64 .
docker build --platform linux/arm64 -t app:arm64 .

# 并行构建（快）
docker buildx build --platform linux/amd64,linux/arm64 -t app:latest .
```

**效果对比**：

| 优化前 | 优化后 |
|--------|--------|
| 首次构建：8分钟 | 首次构建：8分钟 |
| 代码变化后构建：7分钟（重新下载依赖） | 代码变化后构建：1分钟（依赖缓存） |
| pom.xml变化后构建：8分钟 | pom.xml变化后构建：3分钟（有BuildKit缓存） |

### Q4: 如何确定灰度比例的递增节奏？

灰度放量的节奏取决于业务的风险容忍度和团队的信心。

**参考节奏**：

| 阶段 | 流量比例 | 观察时间 | 通过标准 |
|------|----------|----------|----------|
| 1 | 5% | 24小时 | 错误率<基线, P99延迟<基线+10%, 无Pod重启 |
| 2 | 10% | 24小时 | 同上 |
| 3 | 20% | 24小时 | 同上 |
| 4 | 50% | 48小时 | 同上，覆盖周末 |
| 5 | 100% | 持续监控2周 | 同上 |

**关键原则**：

1. **每个阶段至少观察24小时**：覆盖一个完整的业务周期（高峰和低谷），有些问题只在特定时段出现

2. **50%阶段要覆盖周末**：周末和工作日的流量模式可能不同，需要验证两种场景

3. **100%后不要立即下线旧环境**：至少保留2周，这是最后的安全网

4. **根据业务调整**：
   - 内部系统：可以更激进（10% → 50% → 100%）
   - 核心业务：更保守（1% → 5% → 10% → 25% → 50% → 75% → 100%）
   - 高峰期：暂停放量，等待低谷期再继续

5. **定义明确的通过标准**：
   - 错误率不超过基线
   - P99延迟不超过基线+10%
   - 没有Pod重启或OOM
   - CPU/内存使用在预期范围内
   - 没有告警触发

6. **任何阶段出现问题都回滚**：不要试图在当前阶段修复问题，先回滚，线下修复后重新开始灰度

**自动化放量**：

可以用脚本自动化放量过程（需要谨慎）：

```bash
#!/bin/bash
# 自动灰度脚本（示例）

STAGES=(5 10 20 50 100)
OBSERVE_HOURS=24

for STAGE in "${STAGES[@]}"; do
    echo "放量到 ${STAGE}%"
    update_nginx_weight $STAGE

    echo "观察 ${OBSERVE_HOURS} 小时"
    sleep ${OBSERVE_HOURS}h

    echo "检查指标"
    if ! check_metrics_ok; then
        echo "指标异常，回滚"
        update_nginx_weight 0
        exit 1
    fi

    echo "${STAGE}% 阶段通过"
done

echo "迁移完成"
```

但建议人工审核每个阶段，自动化只是辅助。

### Q5: 迁移完成后，原来的运维监控还需要保留吗？

需要逐步替换，而不是立即删除。

**迁移路径**：第1周建立Prometheus+Grafana新监控体系；第2-4周新旧并行运行，验证数据准确性；第5-6周切换告警到新系统；第7-8周确认覆盖完整后下线旧监控。

**分层处理**：应用指标（QPS、错误率）替换为Prometheus；容器和Pod指标是新增的（cAdvisor、kube-state-metrics）；节点硬件、机房环境、网络设备的监控不属于Kubernetes职责，需保留传统工具。

**关键原则**：保留至少1-2个月的overlap期，告警双写确保没有监控盲区。

## 参考资源

- [Kubernetes 滚动更新策略](https://kubernetes.io/docs/tutorials/kubernetes-basics/update/update-intro/)
- [Deployment 滚动更新配置](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment)
- [Prometheus 监控最佳实践](https://prometheus.io/docs/practices/)

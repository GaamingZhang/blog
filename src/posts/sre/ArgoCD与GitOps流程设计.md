---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - SRE
tag:
  - ArgoCD
  - GitOps
  - Kubernetes
  - CI/CD
---

# ArgoCD 与 GitOps：从原理到多集群生产实践

设想这样一个场景：你的团队正在维护十几个微服务，部署到三个 Kubernetes 集群（开发、预发布、生产）。每次发布，运维工程师需要登录不同集群，执行 `kubectl apply`，手动确认每个资源的状态。某一天，有人在生产集群上直接修改了某个 Deployment 的副本数来应急，但忘记更新 Git，导致配置漂移。下次发布时，这个修改被覆盖，故障再次出现。

这就是传统命令式部署模型的典型痛点：**缺乏对环境实际状态的持续感知，也没有机制保证环境与期望配置的一致性**。GitOps 与 ArgoCD 正是为解决这类问题而生。

## 一、GitOps 的核心原则

### 传统 CI/CD 的问题

传统 CI/CD 流水线在 CD 阶段通常是这样工作的：CI 系统构建完镜像后，通过 `kubectl apply` 或 `helm upgrade` 直接将变更推送（Push）到集群。这种**推送模型**存在几个根本性问题：

- **集群访问凭证暴露在 CI 系统中**：Jenkins、GitLab Runner 需要持有 Kubeconfig 或 ServiceAccount Token，这是一个攻击面
- **状态不可观测**：CI 流水线执行完毕后，无法持续感知集群实际状态与期望状态是否一致
- **配置漂移无感知**：运维人员直接在集群上的任何修改都不会被自动检测和修正
- **多集群管理复杂**：不同环境的部署脚本各自维护，容易出现不一致

### GitOps 四原则

GitOps 是 Weaveworks 在 2017 年提出的一套实践方法论，其核心思想是**以 Git 仓库作为系统期望状态的唯一可信来源**，并通过自动化机制持续将实际状态与期望状态对齐。

| 原则 | 含义 |
|------|------|
| **声明式（Declarative）** | 系统的期望状态通过声明式配置（如 Kubernetes YAML、Helm Chart）描述，而非命令序列 |
| **版本化（Versioned）** | 所有配置存储在 Git 中，每次变更都有完整的提交历史、审计记录和回滚能力 |
| **自动化（Automated）** | 一旦 Git 中的期望状态发生变化，自动化系统负责将变更应用到目标环境，无需人工干预 |
| **持续协调（Continuously Reconciled）** | 系统持续比较实际状态与期望状态，发现偏差时自动修正（Self-healing）|

### Push 模型 vs Pull 模型

理解 GitOps 的关键，是理解 Pull 模型与传统 Push 模型的本质区别：

```
Push 模型（传统 CI/CD）
┌─────────────┐     kubectl apply      ┌──────────────────┐
│  CI System  │ ──────────────────────► │  K8s Cluster     │
│  (Jenkins)  │   需要集群凭证          │                  │
└─────────────┘                         └──────────────────┘

Pull 模型（GitOps）
┌─────────────┐     git push      ┌──────────┐
│  Developer  │ ─────────────────► │   Git    │
└─────────────┘                    │   Repo   │
                                   └────┬─────┘
                                        │ watch & pull
                                   ┌────▼─────────────────┐
                                   │  ArgoCD (in-cluster)  │
                                   │  持续监听 Git 变更     │
                                   │  自动同步到集群        │
                                   └──────────────────────┘
```

Pull 模型的核心安全优势在于：**集群凭证永远不离开集群**。ArgoCD 运行在集群内部，主动拉取 Git 仓库的配置，而不是被外部系统推送。这意味着即使 CI 系统完全被攻破，攻击者也无法直接修改集群状态。

## 二、ArgoCD 架构与组件

ArgoCD 是目前最流行的 GitOps 实现工具，遵循 Kubernetes 原生设计理念，所有对象都以 CRD 形式存在。理解其架构是用好 ArgoCD 的基础。

```
                        ┌─────────────────────────────────────────┐
                        │              ArgoCD                      │
  Web UI / CLI          │                                          │
  gRPC / REST  ────────►│  ┌─────────────┐    ┌────────────────┐  │
                        │  │  API Server │    │  Repo Server   │  │
                        │  │  认证鉴权   │    │  克隆 Git 仓库  │  │
                        │  │  状态查询   │    │  渲染 Manifest │  │
                        │  └──────┬──────┘    └───────┬────────┘  │
                        │         │                   │           │
                        │         ▼                   ▼           │
                        │  ┌─────────────────────────────────┐    │
                        │  │           Redis                 │    │
                        │  │        缓存应用状态              │    │
                        │  └───────────────┬─────────────────┘    │
                        │                  │                       │
                        │  ┌───────────────▼──────────────────┐   │
                        │  │     Application Controller       │   │
                        │  │   对比期望状态 vs 实际状态         │   │
                        │  │   执行同步，管理资源生命周期        │   │
                        │  └───────────────┬──────────────────┘   │
                        └──────────────────┼──────────────────────┘
                                           │ kubectl
                                           ▼
                              ┌─────────────────────────┐
                              │    Kubernetes Cluster    │
                              └─────────────────────────┘
```

**API Server**：ArgoCD 的统一入口，处理来自 Web UI、argocd CLI 和 gRPC 客户端的所有请求。负责用户认证、RBAC 鉴权，以及将应用状态暴露给外部系统。

**Repo Server**：负责与 Git 仓库交互的核心组件。它克隆 Git 仓库，并根据配置渲染最终的 Kubernetes Manifest，支持 plain YAML、Helm Chart、Kustomize 和 Jsonnet 等多种格式。Repo Server 是无状态的，渲染后的 Manifest 缓存在 Redis 中。

**Application Controller**：ArgoCD 最核心的组件，实现了 GitOps 的协调循环（Reconciliation Loop）。它持续监听两个来源：一是 Repo Server 输出的期望状态（来自 Git），二是 Kubernetes API Server 报告的实际状态。一旦发现偏差（OutOfSync），Controller 根据同步策略决定是否自动执行同步操作。

**Redis**：作为缓存层，存储应用状态、Manifest 渲染结果以及集群信息，减少对 Git 和 Kubernetes API 的重复请求，提升整体性能。

## 三、Application 核心概念

### Application CR

`Application` 是 ArgoCD 的核心 CRD，它描述了"从哪个 Git 仓库的哪个路径，将什么部署到哪个集群的哪个命名空间"：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app
  namespace: argocd
  # 加上这个标签，Application 删除时会级联删除所管理的资源
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default

  # 来源：Git 仓库信息
  source:
    repoURL: https://github.com/my-org/my-app-config.git
    targetRevision: main          # 分支、Tag 或 Commit SHA
    path: overlays/production     # 仓库内的目录路径
    # 如果使用 Helm：
    # helm:
    #   valueFiles:
    #     - values-prod.yaml

  # 目标：部署到哪个集群和命名空间
  destination:
    server: https://kubernetes.default.svc   # 本集群用此地址
    namespace: my-app-prod

  # 同步策略（见下节详解）
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### 同步状态与健康状态

ArgoCD 对应用维护两个独立的状态维度：

**同步状态（Sync Status）**衡量 Git 期望状态与集群实际状态的一致性：

- `Synced`：集群实际状态与 Git 完全一致
- `OutOfSync`：存在差异，可能是 Git 有新提交，也可能是集群被手动修改
- `Unknown`：无法获取状态信息（如集群不可达）

**健康状态（Health Status）**衡量应用自身的运行状况：

- `Healthy`：所有资源都已正常运行（如 Deployment 的所有 Pod Ready）
- `Progressing`：资源正在滚动更新中
- `Degraded`：资源存在问题（如 Pod CrashLoopBackOff）
- `Suspended`：资源被人为挂起（如 CronJob 被暂停）
- `Missing`：资源在 Git 中定义但集群中不存在

:::tip 两个状态相互独立
`Synced` 不等于 `Healthy`。一个应用可以处于 `Synced + Degraded` 状态，意味着集群状态与 Git 一致，但应用本身存在问题（如镜像拉取失败）。排查问题时要分别关注两个状态。
:::

### App of Apps 模式

在生产环境中，往往有数十甚至上百个应用需要管理。手动为每个应用创建 Application CR 既繁琐又难以维护。**App of Apps** 模式解决了这个问题：创建一个特殊的 Application，其 source 指向一个存储着其他 Application YAML 的 Git 目录，由 ArgoCD 自动创建和管理这些子 Application。

```
Git Repository
└── apps/
    ├── parent-app.yaml          ← 根 Application
    └── applications/
        ├── frontend.yaml        ← 子 Application
        ├── backend.yaml         ← 子 Application
        └── database.yaml        ← 子 Application
```

这样，整个平台的应用管理本身也被纳入 GitOps 管控。

## 四、同步策略与自动化

### Manual vs Auto Sync

ArgoCD 支持手动和自动两种同步模式，需要根据环境特性选择：

| 场景 | 推荐模式 | 原因 |
|------|----------|------|
| 生产环境 | Manual（或带审批的 Auto）| 高风险变更需要人工确认 |
| 开发/测试环境 | Auto Sync | 快速迭代，降低操作成本 |
| Helm Chart 依赖 | Manual | Helm 的 hook 机制需要人工干预 |

### Prune 与 Self Heal

`prune: true` 告诉 ArgoCD，当 Git 仓库中删除了某个资源定义时，同步时自动从集群中删除对应资源。

:::warning Prune 需谨慎
在生产环境启用 `prune` 前，务必确保所有资源都已通过 Git 管理。如果有通过其他方式创建的资源（如手动创建的 Secret），开启 prune 可能导致意外删除。建议先在开发环境验证，再逐步推广到生产。
:::

`selfHeal: true` 是 GitOps 持续协调能力的体现。当有人直接通过 `kubectl edit` 修改了集群中的资源时，ArgoCD 会在检测到漂移后（默认 3 分钟内）自动将其恢复到 Git 中定义的期望状态。

:::danger 注意与手动操作的冲突
开启 selfHeal 后，任何在集群上的手动修改都会被自动撤销。这既是能力也是约束。团队需要建立共识：**所有变更必须通过 Git 提交，禁止直接在集群上修改**，否则 selfHeal 会造成困惑。
:::

### Sync Wave 和 Sync Hook

实际部署中，资源之间往往存在依赖关系，例如数据库 Migration Job 必须在应用 Deployment 之前完成。ArgoCD 提供两种机制来控制部署顺序：

**Sync Wave** 通过注解指定资源的部署波次，数值越小越先部署（支持负数）：

```yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "-1"   # 先部署
```

**Sync Hook** 允许在同步的特定阶段执行自定义资源：

```yaml
metadata:
  annotations:
    argocd.argoproj.io/hook: PreSync      # 可选：PreSync / Sync / PostSync / SyncFail
    argocd.argoproj.io/hook-delete-policy: HookSucceeded  # Hook 成功后自动清理
```

一个典型的数据库迁移场景：

```yaml
# db-migration-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migration
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
    argocd.argoproj.io/sync-wave: "-1"
spec:
  template:
    spec:
      containers:
        - name: migrate
          image: my-app:v2.0.0
          command: ["./migrate.sh"]
      restartPolicy: Never
```

## 五、多集群管理

### Hub-Spoke 架构

大型组织通常采用 **Hub-Spoke** 架构：在一个管理集群（Hub）上部署 ArgoCD，统一管理多个目标集群（Spoke）。这样可以集中管理所有环境的部署，同时保持各集群的独立性。

```
                     ┌──────────────────────┐
                     │   Management Cluster  │
                     │    (ArgoCD Hub)       │
                     └──────────┬───────────┘
              ┌─────────────────┼─────────────────┐
              │                 │                 │
     ┌────────▼──────┐  ┌───────▼───────┐  ┌─────▼───────────┐
     │  Dev Cluster  │  │  Staging      │  │  Prod Cluster   │
     │               │  │  Cluster      │  │                 │
     └───────────────┘  └───────────────┘  └─────────────────┘
```

注册外部集群只需要两步：

```bash
# 1. 使用 argocd CLI 登录 Hub 集群上的 ArgoCD
argocd login argocd.example.com --username admin

# 2. 注册目标集群（需要当前 kubeconfig 中包含目标集群的配置）
argocd cluster add prod-cluster-context --name production

# 查看已注册集群
argocd cluster list
```

ArgoCD 会在目标集群上创建一个 ServiceAccount 和对应的 RBAC 配置，并将凭证以 Secret 的形式存储在 ArgoCD 所在命名空间中。

### ApplicationSet：批量生成 Application

当需要将同一个应用部署到多个集群或多个环境时，逐一创建 Application 效率极低。`ApplicationSet` 控制器可以根据指定的生成器（Generator）批量创建和管理 Application。

**Cluster Generator** 根据集群标签自动生成 Application：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: my-app-all-clusters
  namespace: argocd
spec:
  generators:
    - clusters:
        selector:
          matchLabels:
            environment: production   # 匹配所有 environment=production 标签的集群
  template:
    metadata:
      name: "my-app-{{name}}"         # name 是集群名称
    spec:
      project: default
      source:
        repoURL: https://github.com/my-org/my-app-config.git
        targetRevision: main
        path: "overlays/{{metadata.labels.environment}}"
      destination:
        server: "{{server}}"          # server 是集群 API 地址
        namespace: my-app
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
```

**Git Directory Generator** 根据 Git 仓库目录结构自动生成 Application，非常适合多租户场景：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: tenant-apps
  namespace: argocd
spec:
  generators:
    - git:
        repoURL: https://github.com/my-org/tenants-config.git
        revision: main
        directories:
          - path: "tenants/*"         # 每个子目录对应一个租户
  template:
    metadata:
      name: "tenant-{{path.basename}}"
    spec:
      project: default
      source:
        repoURL: https://github.com/my-org/tenants-config.git
        targetRevision: main
        path: "{{path}}"
      destination:
        server: https://kubernetes.default.svc
        namespace: "{{path.basename}}"
      syncPolicy:
        syncOptions:
          - CreateNamespace=true
```

## 六、Secret 管理方案

GitOps 最常被问到的问题是：**配置都放 Git，那密码怎么办？** 明文 Secret 提交 Git 是绝对禁止的。以下是三种主流方案：

### 方案对比

| 方案 | 核心思路 | 优势 | 劣势 |
|------|----------|------|------|
| **Sealed Secrets** | 用公钥加密 Secret，加密后的 SealedSecret CR 可以安全存 Git | 完全 GitOps 友好，无外部依赖 | 密钥轮换麻烦，私钥泄露影响所有 Secret |
| **External Secrets Operator (ESO)** | 从 Vault/AWS SM/GCP SM 等外部系统同步 Secret 到 K8s | 集中管理，易轮换，支持多后端 | 需要维护外部系统 |
| **ArgoCD Vault Plugin** | 在 ArgoCD 渲染 Manifest 时，将占位符替换为 Vault 中的实际值 | 与 ArgoCD 深度集成 | 配置相对复杂，插件版本需维护 |

:::tip 生产推荐
对于已有 HashiCorp Vault 或云厂商 Secrets Manager 的团队，推荐使用 **External Secrets Operator**。它将 Secret 的管理权归还给专业的密钥管理系统，ArgoCD 的 Git 仓库中只存储 `ExternalSecret` CR（描述"从哪里取什么 Secret"），不包含任何敏感值。
:::

一个 ExternalSecret 示例：

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: my-app-secret
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: my-app-secret    # 生成的 K8s Secret 名称
  data:
    - secretKey: DB_PASSWORD
      remoteRef:
        key: secret/my-app/production
        property: db_password
```

## 七、回滚策略

GitOps 的一大优势是回滚极其简单：**回滚操作等于 Git 回滚**。

```bash
# 方式一：Git 回滚（推荐，符合 GitOps 理念）
git revert HEAD     # 创建一个新提交，撤销上次变更
git push origin main
# ArgoCD 检测到 Git 变更后自动同步，无需额外操作

# 方式二：通过 ArgoCD History 回滚到历史版本
argocd app history my-app           # 查看部署历史
argocd app rollback my-app <ID>     # 回滚到指定历史版本

# 方式三：通过 ArgoCD UI 操作（适合紧急情况）
# 在 UI 中找到应用 -> History and Rollback -> 选择目标版本
```

:::warning Rollback 与 Git 的分歧
通过 `argocd app rollback` 回滚后，集群状态会临时与 Git 不一致（此时应用处于 OutOfSync 状态）。**这只是临时措施**，必须同时将 Git 中的配置回退到对应版本，否则下次自动同步会将集群状态重新推进到 Git 中的最新版本。
:::

与 Helm Rollback 的区别：`helm rollback` 是命令式操作，将集群回滚到某个 Helm Release 历史，不会修改 Git，容易造成 Git 与集群状态不一致。在 GitOps 模式下，应优先使用 Git 回滚。

## 八、ArgoCD 与 Jenkins/GitLab CI 的配合

ArgoCD 专注于 CD（持续部署），不负责 CI（持续集成）。生产中最常见的架构是将两者分工合作：

```
┌─────────────────────────────────────────────────────────────────┐
│  CI 阶段 (Jenkins / GitLab CI)                                   │
│                                                                  │
│  1. 代码提交触发 CI                                               │
│  2. 构建 Docker 镜像，打 Tag（如 v1.2.3 或 git-sha）            │
│  3. 推送镜像到 Harbor                                            │
│  4. 更新 Config Repo 中的镜像 Tag（git commit + push）           │
└──────────────────────────────────────────────────────────────────┘
                                    │ git push config repo
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│  CD 阶段 (ArgoCD)                                                │
│                                                                  │
│  5. ArgoCD 检测到 Config Repo 变更                               │
│  6. 自动同步新的 Deployment（使用新镜像 Tag）                    │
│  7. 集群完成滚动更新                                             │
└──────────────────────────────────────────────────────────────────┘
```

**关键设计：应用仓库与配置仓库分离**。这样 CI 只需要写权限到配置仓库，ArgoCD 只需要读权限，职责清晰。

### Image Updater

如果不想让 CI 系统负责更新 Git 中的镜像 Tag，可以使用 **ArgoCD Image Updater**。它监听镜像仓库（如 Harbor），当有新镜像 Tag 时自动更新 Git 中的配置：

```yaml
# 在 Application 的注解中配置 Image Updater
metadata:
  annotations:
    argocd-image-updater.argoproj.io/image-list: my-app=harbor.example.com/my-org/my-app
    argocd-image-updater.argoproj.io/my-app.update-strategy: semver   # 按语义化版本更新
    argocd-image-updater.argoproj.io/my-app.allow-tags: regexp:^v[0-9]+\.[0-9]+\.[0-9]+$
    argocd-image-updater.argoproj.io/write-back-method: git           # 更新方式写回 Git
```

## 小结

- **GitOps 的本质**是以 Git 为单一可信来源，通过 Pull 模型持续协调集群实际状态与期望状态，解决传统 Push 模型的安全和一致性问题
- **ArgoCD 四大组件**各司其职：API Server 处理请求，Repo Server 渲染 Manifest，Application Controller 执行协调，Redis 提供缓存
- **同步状态与健康状态**是两个独立维度，`Synced` 不等于 `Healthy`，排查问题时要区分对待
- **ApplicationSet** 是多集群、多环境管理的利器，通过生成器批量创建 Application，避免重复配置
- **Secret 管理**建议使用 External Secrets Operator，将敏感信息的管理权交给专业系统
- **CI/CD 分离**是 GitOps 实践的最佳架构：CI 负责构建和推送镜像并更新 Config Repo，ArgoCD 监听 Config Repo 变更并自动同步到集群

---

## 常见问题

### Q1：ArgoCD 的同步频率是多少？如何调整？

ArgoCD 默认每 **3 分钟**从 Git 仓库拉取一次，检查是否有变更。这个间隔可以通过 `argocd-cm` ConfigMap 中的 `timeout.reconciliation` 参数调整。

```yaml
# argocd-cm ConfigMap
data:
  timeout.reconciliation: 180s   # 默认 3 分钟
```

更推荐的方式是配置 **Git Webhook**：当 Git 仓库有推送时，主动通知 ArgoCD，实现秒级响应。ArgoCD 提供了 `/api/webhook` 接口，支持 GitHub、GitLab、Bitbucket 等主流平台的 Webhook 格式。配置 Webhook 后，通常可在 30 秒内完成从 Git 提交到集群同步的全过程。

### Q2：ArgoCD 的 RBAC 如何与企业 SSO 集成？

ArgoCD 内置了与 Dex 的集成，Dex 是一个 OpenID Connect 联合身份认证服务，支持对接 LDAP、SAML 2.0、GitHub OAuth、GitLab OAuth 等主流身份提供商。配置完成后，用户可以使用企业账号登录 ArgoCD。

在 RBAC 配置上，ArgoCD 提供了基于角色的权限控制，权限对象包括 `applications`、`clusters`、`repositories` 等，支持精细到"哪个项目的哪个应用可以执行什么操作"。生产建议：

- 只读访问给开发人员
- 手动触发同步权限给 SRE
- 管理员权限严格限制人数
- 通过 AppProject 隔离不同团队的资源访问范围

### Q3：ApplicationSet 中如何防止批量误操作导致所有环境同时出问题？

ApplicationSet 提供了 `syncPolicy.applicationsSync` 字段来控制批量同步行为，可以限制并发更新数量。更重要的是在架构上做好隔离：

1. **渐进式发布**：不要在同一个 ApplicationSet 中管理开发和生产环境，分开管理，先验证开发再推生产
2. **Progressive Syncs（ArgoCD 2.6+）**：ApplicationSet 支持渐进式同步，先同步一个集群，验证通过后再继续其他集群，配合 Argo Rollouts 可以实现真正的金丝雀发布
3. **分支隔离**：不同环境对应不同 Git 分支，生产环境通过 PR 合并的方式变更，天然有审批流程

### Q4：ArgoCD 自身的高可用如何保证？

ArgoCD 各组件均支持多副本部署。生产推荐配置：

- **API Server**：2-3 副本，无状态，可水平扩展
- **Repo Server**：2-3 副本，克隆仓库操作有一定 CPU/内存开销，根据管理的应用数量调整
- **Application Controller**：通常 1 副本即可，它通过 Redis 做分布式锁；大规模场景（数百个 Application）可以开启分片模式（Sharding）
- **Redis**：可以使用 Redis Sentinel 或 Redis Cluster 实现高可用

ArgoCD 自身也应该通过 GitOps 管理（即 ArgoCD 管理自己），`argocd-autopilot` 工具可以帮助完成 ArgoCD 的自举（Bootstrap）部署。

### Q5：如何处理 ArgoCD 中的 Helm Hook 与 Sync Wave 的优先级问题？

当 Helm Chart 内部有 `helm.sh/hook` 注解，同时 ArgoCD 也有 `argocd.argoproj.io/hook` 和 `argocd.argoproj.io/sync-wave` 时，两套机制会产生交互。

ArgoCD 在处理 Helm Chart 时，**将 Helm Hook 转换为对应的 ArgoCD Hook**：`pre-install`/`pre-upgrade` 对应 `PreSync`，`post-install`/`post-upgrade` 对应 `PostSync`。两套 Hook 机制在同一个 Sync 生命周期内按以下顺序执行：

1. PreSync Hook（包含 Helm pre-install/upgrade hook）
2. Sync Wave 0 资源（按 wave 数值从小到大）
3. Sync（普通资源，按 wave 排序）
4. PostSync Hook（包含 Helm post-install/upgrade hook）

**推荐实践**：尽量避免在同一个应用中混用两套 Hook 机制，如果使用 Helm Chart，优先使用 Helm 原生的 Hook 注解，只在需要跨多个资源协调顺序时才引入 ArgoCD 的 Sync Wave。

## 参考资源

- [ArgoCD 官方文档](https://argo-cd.readthedocs.io/)
- [GitOps 最佳实践](https://opengitops.dev/)
- [ArgoCD ApplicationSet 控制器](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/)

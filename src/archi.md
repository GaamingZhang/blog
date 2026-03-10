---
date: 2026-03-09
author: Jiaming Zhang
isOriginal: true
icon: material-symbols:architecture
sticky: 1001
star: 1001
---

# 部署架构

本文详细介绍本项目在本地 Kubernetes 集群中的完整部署架构，包括代码仓库设计、双集群架构、CI/CD 流水线、ArgoCD GitOps 部署以及 Harbor 镜像仓库的使用。

## 一、整体架构概览

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              整体部署架构                                         │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐            │
│  │   开发环境       │     │   GitLab        │     │   Jenkins       │            │
│  │   (macOS)       │────▶│   (代码仓库)     │────▶│   (CI/CD)       │            │
│  └─────────────────┘     └─────────────────┘     └─────────────────┘            │
│                                                           │                       │
│                                                           ▼                       │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                          Harbor 镜像仓库                                 │    │
│  │  ┌─────────────────────┐     ┌─────────────────────┐                    │    │
│  │  │  Cluster1 Registry  │     │  Cluster2 Registry  │                    │    │
│  │  │  192.168.31.30:30002│     │  192.168.31.31:30002│                    │    │
│  │  └─────────────────────┘     └─────────────────────┘                    │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
│                                                           │                       │
│                                                           ▼                       │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                     ArgoCD GitOps 部署                                   │    │
│  │  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐    │    │
│  │  │  ArgoCD 仓库    │────▶│  ArgoCD Server  │────▶│  Kubernetes     │    │    │
│  │  │  (渲染后清单)   │     │  (GitOps引擎)   │     │  (双集群)       │    │    │
│  │  └─────────────────┘     └─────────────────┘     └─────────────────┘    │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## 二、代码仓库设计

### 2.1 仓库结构

本项目采用双仓库设计，实现代码与部署清单的分离：

| 仓库 | 用途 | 内容 |
|------|------|------|
| `blog` | 主代码仓库 | VuePress 博客源码、Helm Chart、Jenkinsfile |
| `gaamingblogkubernetesargocd` | ArgoCD 仓库 | 渲染后的 Kubernetes 清单文件 |

### 2.2 主仓库目录结构

```
blog/
├── src/                          # VuePress 博客源码
│   ├── .vuepress/               # VuePress 配置
│   └── posts/                   # 博客文章
├── k8s/                         # Kubernetes 配置
│   ├── helm/                    # Helm Chart
│   │   └── blog/
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       ├── values-cluster1.yaml
│   │       ├── values-cluster2.yaml
│   │       └── templates/
│   ├── kustomize/               # ArgoCD 配置
│   │   └── argocd/
│   │       ├── applications/
│   │       └── projects/
│   ├── jenkins/                 # Jenkins 配置
│   │   └── gaamingblog.Jenkinsfile
│   └── docs/                    # 部署文档
├── Dockerfile                   # Docker 构建文件
└── pipelines/                   # CI/CD 配置
```

### 2.3 ArgoCD 仓库结构

```
gaamingblogkubernetesargocd/
└── apps/
    └── blog/
        ├── cluster1/
        │   └── all.yaml         # Cluster1 的 Kubernetes 清单
        └── cluster2/
            └── all.yaml         # Cluster2 的 Kubernetes 清单
```

### 2.4 分支管理策略

本项目采用严格的分支管理策略，确保代码质量和部署的可追溯性：

```
┌─────────────────────────────────────────────────────────────────┐
│                    分支管理策略                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    开发分支 (Feature)                      │    │
│  │  users/jiaming/<fix_something>                           │    │
│  │  users/jiaming/<add_feature>                             │    │
│  │  users/jiaming/<update_docs>                             │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           │                                      │
│                           │ Pull Request                         │
│                           ▼                                      │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    主分支 (Main)                           │    │
│  │  main                                                    │    │
│  │  - 代码审查通过后合并                                      │    │
│  │  - 不直接用于构建部署                                      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           │                                      │
│                           │ 流水线触发                           │
│                           ▼                                      │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    构建分支 (Official)                     │    │
│  │  official.<version>                                      │    │
│  │  - 每次构建自动创建                                        │    │
│  │  - 用于 Kubernetes 集群部署                               │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### 分支类型说明

| 分支类型 | 命名规范 | 用途 | 生命周期 |
|---------|---------|------|---------|
| 主分支 | `main` | 稳定的生产代码基线 | 永久 |
| 开发分支 | `users/jiaming/<description>` | 日常开发、修复、功能添加 | 临时，合并后删除 |
| 构建分支 | `official.<version>` | CI/CD 流水线构建和部署 | 临时，构建完成后保留 |

#### 开发流程

```
┌─────────────────────────────────────────────────────────────────┐
│                    开发流程                                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. 创建开发分支                                                  │
│     └─ git checkout -b users/jiaming/fix_blog_layout            │
│                                                                  │
│  2. 开发和提交                                                    │
│     └─ git add . && git commit -m "fix: 修复博客布局问题"         │
│                                                                  │
│  3. 推送到远程                                                    │
│     └─ git push origin users/jiaming/fix_blog_layout            │
│                                                                  │
│  4. 创建 Pull Request                                            │
│     └─ 在 GitLab 上创建 PR，目标分支为 main                       │
│                                                                  │
│  5. 代码审查                                                      │
│     └─ 团队成员审查代码，提出修改建议                              │
│                                                                  │
│  6. 合并到主分支                                                  │
│     └─ 审查通过后，合并 PR 到 main 分支                           │
│                                                                  │
│  7. 触发流水线                                                    │
│     └─ Jenkins 自动创建 official.<version> 分支                  │
│                                                                  │
│  8. 构建和部署                                                    │
│     └─ 使用 official 分支代码进行 Kubernetes 集群部署             │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### 为什么不直接使用 main 分支构建？

1. **版本追溯**：每个 official 分支对应一个具体的构建版本，便于问题排查和回滚
2. **构建隔离**：避免在构建过程中 main 分支有新的提交，确保构建的一致性
3. **审计追踪**：保留每次构建的完整代码快照，满足审计要求
4. **安全控制**：main 分支受保护，构建分支由 CI 系统自动创建，减少人为错误

## 三、Ansible 自动化集群部署

本项目的 Kubernetes 集群完全由 Ansible 自动化部署，实现基础设施即代码（IaC）。

### 3.1 Ansible 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    Ansible 部署架构                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    控制节点 (Ubuntu)                       │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │    │
│  │  │  Ansible    │  │  Inventory  │  │   Playbooks │      │    │
│  │  │  Core       │  │  (主机清单)  │  │   (剧本)    │      │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              │ SSH                               │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    目标节点 (虚拟机)                       │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │    │
│  │  │ Cluster1    │  │ Cluster1    │  │ Cluster1    │      │    │
│  │  │ Master      │  │ Worker1     │  │ Worker2     │      │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘      │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │    │
│  │  │ Cluster2    │  │ Cluster2    │  │ Cluster2    │      │    │
│  │  │ Master      │  │ Worker1     │  │ Worker2     │      │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Ansible 项目结构

```
ansible-kubernetes/
├── inventory/
│   ├── cluster1/
│   │   └── hosts.ini           # Cluster1 主机清单
│   └── cluster2/
│       └── hosts.ini           # Cluster2 主机清单
├── group_vars/
│   ├── all.yml                 # 全局变量
│   ├── cluster1.yml            # Cluster1 变量
│   └── cluster2.yml            # Cluster2 变量
├── roles/
│   ├── common/                 # 基础配置
│   │   ├── tasks/
│   │   │   └── main.yml
│   │   └── templates/
│   ├── containerd/             # Containerd 安装
│   │   └── tasks/
│   │       └── main.yml
│   ├── kubernetes/             # Kubernetes 安装
│   │   ├── tasks/
│   │   │   └── main.yml
│   │   └── templates/
├── playbooks/
│   ├── init-cluster.yml        # 初始化集群
│   ├── install-harbor.yml      # 安装 Harbor
│   └── install-argocd.yml      # 安装 ArgoCD
└── ansible.cfg                 # Ansible 配置
```

## 四、Kubernetes 主备集群架构

### 4.1 集群角色规划

本项目采用**主备架构（Active-Standby）**，Cluster1 作为主集群处理所有请求，Cluster2 作为备份集群，平时不参与流量服务。

| 集群 | 角色 | 状态 | IP 地址 | 说明 |
|------|------|------|---------|------|
| Cluster1 | 主集群 | Active | 192.168.31.30 | 处理所有用户请求，运行 ArgoCD |
| Cluster2 | 备份集群 | Standby | 192.168.31.31 | 保持应用同步，故障时接管服务 |

### 4.2 主备架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                    主备集群架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                    ┌─────────────────┐                          │
│                    │   用户请求       │                          │
│                    │   local.blog    │                          │
│                    └────────┬────────┘                          │
│                             │                                    │
│                             ▼                                    │
│              ┌──────────────────────────────┐                   │
│              │                              │                   │
│              │   ┌─────────────────────┐    │                   │
│              │   │   Cluster1 (主)     │    │                   │
│              │   │   192.168.31.30     │    │                   │
│              │   │   Status: Active    │    │                   │
│              │   │                     │    │                   │
│              │   │   • 处理所有请求     │    │                   │
│              │   │   • ArgoCD 管理     │    │                   │
│              │   │   • Harbor 镜像     │    │                   │
│              │   └─────────────────────┘    │                   │
│              │                              │                   │
│              └──────────────────────────────┘                   │
│                             │                                    │
│                             │ 数据同步                           │
│                             ▼                                    │
│              ┌──────────────────────────────┐                   │
│              │                              │                   │
│              │   ┌─────────────────────┐    │                   │
│              │   │   Cluster2 (备)     │    │                   │
│              │   │   192.168.31.31     │    │                   │
│              │   │   Status: Standby   │    │                   │
│              │   │                     │    │                   │
│              │   │   • 应用保持同步     │    │                   │
│              │   │   • 不处理请求       │    │                   │
│              │   │   • 故障时接管       │    │                   │
│              │   └─────────────────────┘    │                   │
│              │                              │                   │
│              └──────────────────────────────┘                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 故障切换流程

当 Cluster1 发生故障时，需要手动或自动将流量切换到 Cluster2：

```
┌─────────────────────────────────────────────────────────────────┐
│                    故障切换流程                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. 检测故障                                                      │
│     └─ 监控系统发现 Cluster1 不可用                               │
│                                                                  │
│  2. 确认切换                                                      │
│     └─ 运维人员确认需要切换到 Cluster2                            │
│                                                                  │
│  3. DNS 切换 (模拟外部负载均衡流量切换)                                                       │
│     └─ 修改 local.blog 的 DNS 指向 Cluster2                      │
│                                                                  │
│  4. 验证服务                                                      │
│     └─ 确认 Cluster2 正常提供服务                                 │
│                                                                  │
│  5. 修复主集群                                                    │
│     └─ 修复 Cluster1 并恢复为 Active 状态                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.4 集群节点详情

**Cluster1:**

```
NAME               ROLE           INTERNAL-IP
cluster1-master    control-plane  192.168.31.30
cluster1-worker1   <none>         192.168.31.40
cluster1-worker2   <none>         192.168.31.41
```

**Cluster2:**

```
NAME               ROLE           INTERNAL-IP
cluster2-master    control-plane  192.168.31.31
cluster2-worker1   <none>         192.168.31.42
cluster2-worker2   <none>         192.168.31.43
```

### 4.5 网络架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        网络架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │           Cluster1 - 主集群 (Active)                     │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │    │
│  │  │   Master    │  │  Worker1    │  │  Worker2    │      │    │
│  │  │  .30        │  │  .40        │  │  .41        │      │    │
│  │  │  ArgoCD     │  │  Ingress ◀──│──│  Blog Pod   │      │    │
│  │  │  Harbor     │  │  Blog Pod   │  │             │      │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘      │    │
│  │        ▲                          处理所有用户请求         │    │
│  │        │                          local.blog             │    │
│  └────────│────────────────────────────────────────────────┘    │
│           │                                                       │
│           │ 数据同步（ArgoCD 推送清单）                            │
│           ▼                                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │           Cluster2 - 备份集群 (Standby)                  │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │    │
│  │  │   Master    │  │  Worker1    │  │  Worker2    │      │    │
│  │  │  .31        │  │  .42        │  │  .43        │      │    │
│  │  │  Harbor     │  │  Ingress    │  │  Blog Pod   │      │    │
│  │  │             │  │  (未启用)   │  │  (待命)     │      │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘      │    │
│  │                          不处理请求                        │    │
│  │                          故障时接管                        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## 五、Harbor 镜像仓库

### 5.1 Harbor 部署架构

每个集群部署独立的 Harbor 实例，实现镜像的本地化存储：

| Harbor 实例 | 地址 | 用途 |
|-------------|------|------|
| Harbor-Cluster1 | 192.168.31.30:30002 | Cluster1 镜像存储 |
| Harbor-Cluster2 | 192.168.31.31:30002 | Cluster2 镜像存储 |

### 5.2 镜像命名规范

```
192.168.31.30:30002/gaaming/blog:1.0.18
│                   │       │    │
│                   │       │    └── 版本号
│                   │       └─────── 镜像名称
│                   └─────────────── 项目名称
└─────────────────────────────────── Harbor 地址
```

### 5.3 镜像拉取认证

在 Kubernetes 中创建 Secret 用于镜像拉取认证：

```bash
kubectl create secret docker-registry harbor-registry-secret \
  --namespace=blog \
  --docker-server=192.168.31.30:30002 \
  --docker-username=<username> \
  --docker-password=<password>
```

## 六、Helm Chart 设计

### 6.1 Chart 结构

```
k8s/helm/blog/
├── Chart.yaml              # Chart 元数据
├── values.yaml             # 默认配置
├── values-cluster1.yaml    # Cluster1 配置
├── values-cluster2.yaml    # Cluster2 配置
└── templates/
    ├── _helpers.tpl        # 模板辅助函数
    ├── deployment.yaml     # Deployment 模板
    ├── service.yaml        # Service 模板
    ├── ingress.yaml        # Ingress 模板
    └── ingress-canary.yaml # Canary Ingress 模板
```

### 6.2 核心配置示例

**values-cluster1.yaml:**

```yaml
image:
  repository: 192.168.31.30:30002/gaaming/blog
  tag: latest

imagePullSecrets:
  - name: harbor-registry-secret

cluster: cluster1

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: local.blog
      paths:
        - path: /
          pathType: Prefix
```

### 6.3 Helm 渲染命令

```bash
# 渲染 Cluster1 配置
helm template blog k8s/helm/blog \
  --namespace blog \
  --set image.repository=192.168.31.30:30002/gaaming/blog \
  --set image.tag=1.0.18 \
  --set cluster=cluster1 \
  -f k8s/helm/blog/values-cluster1.yaml \
  > apps/blog/cluster1/all.yaml
```

## 七、CI/CD 流水线

### 7.1 Jenkins 流水线阶段

```
┌─────────────────────────────────────────────────────────────────┐
│                    Jenkins Pipeline                              │
├─────────────────────────────────────────────────────────────────┤
│  Stage 1: Trigger updateVersion                                  │
│     └─ 触发版本更新任务，生成新版本号                              │
├─────────────────────────────────────────────────────────────────┤
│  Stage 2: Checkout Official Branch                               │
│     └─ 检出 official.{version} 分支                               │
├─────────────────────────────────────────────────────────────────┤
│  Stage 3: Build Image                                            │
│     └─ 构建 Docker 镜像                                          │
├─────────────────────────────────────────────────────────────────┤
│  Stage 4: Push to Harbor - Cluster1                              │
│     └─ 推送镜像到 Cluster1 的 Harbor                              │
├─────────────────────────────────────────────────────────────────┤
│  Stage 5: Push to Harbor - Cluster2                              │
│     └─ 推送镜像到 Cluster2 的 Harbor                              │
├─────────────────────────────────────────────────────────────────┤
│  Stage 6: Render and Push to ArgoCD                              │
│     ├─ 克隆 ArgoCD 仓库                                          │
│     ├─ 使用 Helm 渲染 Kubernetes 清单                             │
│     └─ 推送渲染后的清单到 ArgoCD 仓库                              │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 关键流水线代码

```groovy
stage('Render and Push to ArgoCD') {
  steps {
    script {
      sh "cp -r ${env.WORKSPACE}/${WORKDIR}/${HELM_CHART_PATH} ./helm-chart"

      dir('helm-chart') {
        sh """
          helm template blog . \
            --namespace blog \
            --set image.repository=${env.HARBOR_URL_CLUSTER1}/gaaming/blog \
            --set image.tag=${imageTag} \
            --set cluster=cluster1 \
            -f values-cluster1.yaml \
            > ../apps/blog/cluster1/all.yaml
        """

        sh """
          helm template blog . \
            --namespace blog \
            --set image.repository=${env.HARBOR_URL_CLUSTER2}/gaaming/blog \
            --set image.tag=${imageTag} \
            --set cluster=cluster2 \
            -f values-cluster2.yaml \
            > ../apps/blog/cluster2/all.yaml
        """
      }

      withCredentials([sshUserPrivateKey(...)]) {
        sh """
          git add apps/blog
          git commit -m "feat: 更新 blog 镜像版本到 ${imageTag}"
          git push origin main
        """
      }
    }
  }
}
```

## 八、ArgoCD GitOps 部署

### 8.1 ArgoCD 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    ArgoCD 架构                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    ArgoCD Components                      │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │    │
│  │  │ API Server  │  │ Repo Server │  │ Application │      │    │
│  │  │             │  │             │  │ Controller  │      │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘      │    │
│  │  ┌─────────────┐  ┌─────────────┐                        │    │
│  │  │    Redis    │  │    Dex      │                        │    │
│  │  │  (缓存)     │  │  (SSO)      │                        │    │
│  │  └─────────────┘  └─────────────┘                        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    ArgoCD Resources                       │    │
│  │  ┌─────────────┐  ┌─────────────┐                        │    │
│  │  │ AppProject  │  │ Application │                        │    │
│  │  │   blog      │  │blog-cluster1│                        │    │
│  │  │             │  │blog-cluster2│                        │    │
│  │  └─────────────┘  └─────────────┘                        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 8.2 Application 配置

**blog-cluster1.yaml:**

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: blog-cluster1
  namespace: argocd
spec:
  project: blog
  source:
    repoURL: git@192.168.31.50:gaamingzhang/gaamingblogkubernetesargocd.git
    targetRevision: HEAD
    path: apps/blog/cluster1
  destination:
    server: https://kubernetes.default.svc
    namespace: blog
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

**blog-cluster2.yaml:**

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: blog-cluster2
  namespace: argocd
spec:
  project: blog
  source:
    repoURL: git@192.168.31.50:gaamingzhang/gaamingblogkubernetesargocd.git
    targetRevision: HEAD
    path: apps/blog/cluster2
  destination:
    name: cluster2
    namespace: blog
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

**关键配置说明：**

| 配置项 | Cluster1 | Cluster2 |
|--------|----------|----------|
| destination.server | `https://kubernetes.default.svc` | 不指定（使用 name） |
| destination.name | 不指定 | `cluster2` |
| path | `apps/blog/cluster1` | `apps/blog/cluster2` |

**注意：** Cluster2 使用 `destination.name: cluster2` 而不是 `server` URL，这需要先在 ArgoCD 中注册集群。

### 8.3 注册 Cluster2 到 ArgoCD

在 ArgoCD 能够部署应用到 Cluster2 之前，需要先将 Cluster2 注册到 ArgoCD：

```bash
# 1. 切换到 Cluster2 的 context
kubectl config use-context cluster2

# 2. 创建 ArgoCD 管理服务账号
kubectl create serviceaccount argocd-manager -n kube-system

# 3. 创建 ClusterRoleBinding
kubectl create clusterrolebinding argocd-manager-role-binding \
  --clusterrole=cluster-admin \
  --serviceaccount=kube-system:argocd-manager

# 4. 切换回 Cluster1 的 context（ArgoCD 所在集群）
kubectl config use-context cluster1

# 5. 注册 Cluster2 到 ArgoCD
argocd cluster add cluster2 --name cluster2

# 6. 验证集群注册成功
argocd cluster list
```

### 8.4 部署流程

```
┌─────────────────────────────────────────────────────────────────┐
│                    GitOps 部署流程                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. 开发者提交代码到 blog 仓库                                    │
│     │                                                            │
│     ▼                                                            │
│  2. Jenkins 触发构建流水线                                        │
│     │                                                            │
│     ▼                                                            │
│  3. 构建镜像并推送到 Harbor                                       │
│     │                                                            │
│     ▼                                                            │
│  4. Helm 渲染 Kubernetes 清单                                     │
│     │                                                            │
│     ▼                                                            │
│  5. 推送清单到 ArgoCD 仓库                                        │
│     │                                                            │
│     ▼                                                            │
│  6. ArgoCD 检测到变更（每 3 分钟轮询）                             │
│     │                                                            │
│     ▼                                                            │
│  7. ArgoCD 自动同步部署到 Kubernetes                              │
│     │                                                            │
│     ▼                                                            │
│  8. 验证部署状态                                                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 8.5 验证双集群部署

部署完成后，需要验证 blog 已成功部署到两个集群：

```bash
# 验证 Cluster1 部署状态
kubectl config use-context cluster1
kubectl get pods -n blog
kubectl get application blog-cluster1 -n argocd

# 验证 Cluster2 部署状态
kubectl config use-context cluster2
kubectl get pods -n blog
kubectl get application blog-cluster2 -n argocd

# 在 ArgoCD UI 中查看
# 访问 https://argocd.local，确认两个 Application 都显示 "Synced" 和 "Healthy"
```

**常见问题排查：**

| 问题 | 可能原因 | 解决方案 |
|------|---------|---------|
| blog-cluster2 显示 "Unknown" | Cluster2 未注册到 ArgoCD | 执行 `argocd cluster add cluster2` |
| blog-cluster2 显示 "Degraded" | 镜像拉取失败 | 检查 Harbor 认证和网络连通性 |
| blog-cluster2 不存在 | Application 未创建 | 执行 `kubectl apply -f blog-cluster2.yaml` |
| Cluster2 Pod 无法启动 | 资源不足或配置错误 | 检查 values-cluster2.yaml 配置 |

**确保 Cluster2 部署的检查清单：**

- [ ] Cluster2 已注册到 ArgoCD (`argocd cluster list`)
- [ ] blog-cluster2 Application 已创建 (`kubectl get applications -n argocd`)
- [ ] ArgoCD 仓库中存在 `apps/blog/cluster2/all.yaml`
- [ ] Cluster2 的 Harbor 可访问并包含镜像
- [ ] Cluster2 有足够的资源运行 Pod

## 九、本地访问配置

### 9.1 Ingress Controller 配置

使用 Nginx Ingress Controller，配置 hostNetwork 实现直接访问：

```bash
kubectl patch deployment ingress-nginx-controller -n ingress-nginx --type=json -p='[
  {"op":"add","path":"/spec/template/spec/hostNetwork","value":true},
  {"op":"add","path":"/spec/template/spec/dnsPolicy","value":"ClusterFirstWithHostNet"}
]'
```

### 9.2 访问架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    访问架构                                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐         ┌─────────────────────────────────┐    │
│  │   macOS     │         │        Kubernetes Cluster       │    │
│  │   浏览器    │────────▶│                                 │    │
│  │             │         │  ┌─────────────────────────┐    │    │
│  │ local.blog  │         │  │   Nginx Ingress         │    │    │
│  │ 192.168.    │         │  │   Controller            │    │    │
│  │   31.40     │         │  │   (hostNetwork: true)   │    │    │
│  └─────────────┘         │  └───────────┬─────────────┘    │    │
│                          │              │                   │    │
│                          │              ▼                   │    │
│                          │  ┌─────────────────────────┐    │    │
│                          │  │   Service: blog         │    │    │
│                          │  │   (ClusterIP)           │    │    │
│                          │  └───────────┬─────────────┘    │    │
│                          │              │                   │    │
│                          │              ▼                   │    │
│                          │  ┌─────────────────────────┐    │    │
│                          │  │   Pods: blog-xxx        │    │    │
│                          │  │   (Nginx + VuePress)    │    │    │
│                          │  └─────────────────────────┘    │    │
│                          └─────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## 十、远程部署架构

### 10.1 双云架构概览

本项目在公网环境采用**双云独立部署架构**，使用腾讯云和阿里云两台虚拟机独立部署应用，通过 DNS 域名解析实现流量分发。

```
┌─────────────────────────────────────────────────────────────────┐
│                    远程部署架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                    ┌─────────────────┐                          │
│                    │   用户请求       │                          │
│                    │   blog.xxx.com  │                          │
│                    └────────┬────────┘                          │
│                             │                                    │
│                             ▼                                    │
│              ┌──────────────────────────────┐                   │
│              │      DNS 域名解析             │                   │
│              │  (负载均衡层)                 │                   │
│              └──────────┬───────────────────┘                   │
│                         │                                        │
│           ┌─────────────┴─────────────┐                         │
│           │                           │                         │
│           ▼                           ▼                         │
│  ┌─────────────────┐       ┌─────────────────┐                 │
│  │  腾讯云 VM       │       │  阿里云 VM       │                 │
│  │  (主节点)        │       │  (备份节点)      │                 │
│  │                 │       │                 │                 │
│  │  • 处理主流量    │       │  • 处理少量流量  │                 │
│  │  • 90% 流量     │       │  • 10% 流量     │                 │
│  │  • Nginx 部署   │       │  • Nginx 部署   │                 │
│  │  • 静态文件托管  │       │  • 静态文件托管  │                 │
│  └─────────────────┘       └─────────────────┘                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 10.2 服务器配置

| 云厂商 | 角色 | 流量比例 | 配置 | 用途 |
|--------|------|---------|------|------|
| 腾讯云 | 主节点 | 90% | 2核4G | 处理主要用户请求 |
| 阿里云 | 备份节点 | 10% | 2核4G | 处理少量流量，故障时接管 |

### 10.3 DNS 负载均衡配置

通过 DNS 解析实现流量分发，无需额外的负载均衡器：

```
┌─────────────────────────────────────────────────────────────────┐
│                    DNS 解析配置                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  域名: blog.xxx.com                                              │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  A 记录配置                                               │    │
│  ├─────────────────────────────────────────────────────────┤    │
│  │  blog.xxx.com  →  腾讯云 IP (TTL: 600)                  │    │
│  │  blog.xxx.com  →  阿里云 IP (TTL: 600)                  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  流量分配策略：                                                   │
│  • 腾讯云: 权重 90 (主流量)                                       │
│  • 阿里云: 权重 10 (备份流量)                                     │
│                                                                  │
│  DNS 轮询机制：                                                   │
│  • DNS 服务器按照权重返回 IP 地址                                 │
│  • 客户端缓存 DNS 结果 (TTL 时间内)                               │
│  • 自动实现流量分发                                               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 10.4 部署架构详解

```
┌─────────────────────────────────────────────────────────────────┐
│                    单机部署架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    腾讯云虚拟机                           │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │              Nginx 环境                          │    │    │
│  │  │  ┌─────────────────────────────────────────┐    │    │    │
│  │  │  │         /var/www/blog                   │    │    │    │
│  │  │  │         (静态文件目录)                   │    │    │    │
│  │  │  │                                         │    │    │    │
│  │  │  │  • index.html                           │    │    │    │
│  │  │  │  • assets/                              │    │    │    │
│  │  │  │  • posts/                               │    │    │    │
│  │  │  └─────────────────────────────────────────┘    │    │    │
│  │  │                                                  │    │    │
│  │  │  ┌─────────────────────────────────────────┐    │    │    │
│  │  │  │         Nginx (:80, :443)               │    │    │    │
│  │  │  │         (Web 服务器)                    │    │    │    │
│  │  │  └─────────────────────────────────────────┘    │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    阿里云虚拟机                           │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │              Nginx 环境                          │    │    │
│  │  │  ┌─────────────────────────────────────────┐    │    │    │
│  │  │  │         /var/www/blog                   │    │    │    │
│  │  │  │         (静态文件目录)                   │    │    │    │
│  │  │  │                                         │    │    │    │
│  │  │  │  • index.html                           │    │    │    │
│  │  │  │  • assets/                              │    │    │    │
│  │  │  │  • posts/                               │    │    │    │
│  │  │  └─────────────────────────────────────────┘    │    │    │
│  │  │                                                  │    │    │
│  │  │  ┌─────────────────────────────────────────┐    │    │    │
│  │  │  │         Nginx (:80, :443)               │    │    │    │
│  │  │  │         (Web 服务器)                    │    │    │    │
│  │  │  └─────────────────────────────────────────┘    │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 10.5 Nginx 配置

**nginx.conf:**

```nginx
server {
    listen 80;
    server_name blog.xxx.com;
    
    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name blog.xxx.com;
    
    root /var/www/blog;
    index index.html;
    
    ssl_certificate /etc/nginx/ssl/blog.xxx.com.crt;
    ssl_certificate_key /etc/nginx/ssl/blog.xxx.com.key;
    
    # SSL 配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    
    # Gzip 压缩
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml;
    gzip_min_length 1000;
    
    # 静态文件缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
    
    # VuePress 路由支持
    location / {
        try_files $uri $uri/ $uri.html =404;
    }
    
    # 健康检查端点
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
```

### 10.6 CI/CD 部署流程

```
┌─────────────────────────────────────────────────────────────────┐
│                    远程部署流程                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. 开发者提交代码                                                │
│     └─ 推送到 GitLab main 分支                                   │
│                                                                  │
│  2. Jenkins 触发构建                                              │
│     └─ 执行 VuePress 构建                                        │
│                                                                  │
│  3. 生成静态文件                                                  │
│     └─ npm run build 生成 dist 目录                              │
│                                                                  │
│  4. 并行部署到两台服务器                                          │
│     ├─ SSH 连接腾讯云虚拟机                                       │
│     │  └─ scp 复制 dist/* 到 /var/www/blog/                     │
│     │                                                            │
│     └─ SSH 连接阿里云虚拟机                                       │
│        └─ scp 复制 dist/* 到 /var/www/blog/                     │
│                                                                  │
│  5. 健康检查                                                      │
│     ├─ 检查腾讯云服务状态                                         │
│     └─ 检查阿里云服务状态                                         │
│                                                                  │
│  6. 通知                                                          │
│     └─ 发送部署结果通知                                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 10.7 故障切换策略

```
┌─────────────────────────────────────────────────────────────────┐
│                    故障切换流程                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  正常状态:                                                        │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  腾讯云 (主) ──────── 90% 流量 ────────▶ 处理请求        │    │
│  │  阿里云 (备) ──────── 10% 流量 ────────▶ 处理请求        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  腾讯云故障时:                                                    │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  1. 监控系统检测到腾讯云不可用                           │    │
│  │  2. 修改 DNS 解析，移除腾讯云 IP                         │    │
│  │  3. 阿里云接管 100% 流量                                 │    │
│  │  4. 修复腾讯云服务                                       │    │
│  │  5. 恢复 DNS 配置，重新分配流量                          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  阿里云故障时:                                                    │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  1. 监控系统检测到阿里云不可用                           │    │
│  │  2. 修改 DNS 解析，移除阿里云 IP                         │    │
│  │  3. 腾讯云接管 100% 流量                                 │    │
│  │  4. 修复阿里云服务                                       │    │
│  │  5. 恢复 DNS 配置，重新分配流量                          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```
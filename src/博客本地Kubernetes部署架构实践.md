---
date: 2026-03-09
author: Gaaming Zhang
isOriginal: true
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ArgoCD
  - Helm
  - Harbor
  - CI/CD
---

# 博客本地部署架构

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

### 8.3 部署流程

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
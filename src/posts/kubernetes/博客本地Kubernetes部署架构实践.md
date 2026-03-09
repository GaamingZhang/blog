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

# 博客本地 Kubernetes 部署架构实践

本文详细介绍了一个基于 VuePress 的个人技术博客在本地 Kubernetes 集群中的完整部署架构，包括代码仓库设计、双集群架构、CI/CD 流水线、ArgoCD GitOps 部署以及 Harbor 镜像仓库的使用。

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

## 三、Kubernetes 双集群架构

### 3.1 集群规划

| 集群 | 角色 | IP 地址 | 组件 |
|------|------|---------|------|
| Cluster1 | 主集群 | 192.168.31.30 | Master + ArgoCD |
| Cluster2 | 从集群 | 192.168.31.31 | Worker Nodes |

### 3.2 集群节点详情

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

### 3.3 网络架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        网络架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Cluster1 (192.168.31.0/24)            │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │    │
│  │  │   Master    │  │  Worker1    │  │  Worker2    │      │    │
│  │  │  .30        │  │  .40        │  │  .41        │      │    │
│  │  │  ArgoCD     │  │  Ingress    │  │  Blog Pod   │      │    │
│  │  │  Harbor     │  │  Blog Pod   │  │             │      │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Cluster2 (192.168.31.0/24)            │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │    │
│  │  │   Master    │  │  Worker1    │  │  Worker2    │      │    │
│  │  │  .31        │  │  .42        │  │  .43        │      │    │
│  │  │  Harbor     │  │  Ingress    │  │  Blog Pod   │      │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## 四、Harbor 镜像仓库

### 4.1 Harbor 部署架构

每个集群部署独立的 Harbor 实例，实现镜像的本地化存储：

| Harbor 实例 | 地址 | 用途 |
|-------------|------|------|
| Harbor-Cluster1 | 192.168.31.30:30002 | Cluster1 镜像存储 |
| Harbor-Cluster2 | 192.168.31.31:30002 | Cluster2 镜像存储 |

### 4.2 镜像命名规范

```
192.168.31.30:30002/gaaming/blog:1.0.18
│                   │       │    │
│                   │       │    └── 版本号
│                   │       └─────── 镜像名称
│                   └─────────────── 项目名称
└─────────────────────────────────── Harbor 地址
```

### 4.3 镜像拉取认证

在 Kubernetes 中创建 Secret 用于镜像拉取认证：

```bash
kubectl create secret docker-registry harbor-registry-secret \
  --namespace=blog \
  --docker-server=192.168.31.30:30002 \
  --docker-username=<username> \
  --docker-password=<password>
```

### 4.4 Containerd 配置

由于 Harbor 使用 HTTP 协议，需要在所有节点配置 insecure registry：

```bash
# 创建配置目录
sudo mkdir -p /etc/containerd/certs.d/192.168.31.30:30002

# 创建 hosts.toml
sudo tee /etc/containerd/certs.d/192.168.31.30:30002/hosts.toml <<EOF
server = "http://192.168.31.30:30002"

[host."http://192.168.31.30:30002"]
  capabilities = ["pull", "resolve", "push"]
  skip_verify = true
EOF

# 重启 containerd
sudo systemctl restart containerd
```

## 五、Helm Chart 设计

### 5.1 Chart 结构

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

### 5.2 核心配置示例

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

### 5.3 Helm 渲染命令

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

## 六、CI/CD 流水线

### 6.1 Jenkins 流水线阶段

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

### 6.2 关键流水线代码

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

## 七、ArgoCD GitOps 部署

### 7.1 ArgoCD 架构

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

### 7.2 Application 配置

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

### 7.3 部署流程

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

## 八、本地访问配置

### 8.1 Ingress Controller 配置

使用 Nginx Ingress Controller，配置 hostNetwork 实现直接访问：

```bash
kubectl patch deployment ingress-nginx-controller -n ingress-nginx --type=json -p='[
  {"op":"add","path":"/spec/template/spec/hostNetwork","value":true},
  {"op":"add","path":"/spec/template/spec/dnsPolicy","value":"ClusterFirstWithHostNet"}
]'
```

### 8.2 macOS 访问配置

**1. 配置 /etc/hosts:**

```bash
# 添加 hosts 映射（指向运行 Ingress Controller 的节点）
192.168.31.40 local.blog
```

**2. 刷新 DNS 缓存:**

```bash
sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder
```

**3. 访问博客:**

```
http://local.blog
```

### 8.3 访问架构

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

## 九、故障排查指南

### 9.1 常见问题

**1. ImagePullBackOff**

```bash
# 检查 Secret 是否存在
kubectl get secret harbor-registry-secret -n blog

# 检查 containerd 配置
cat /etc/containerd/certs.d/192.168.31.30:30002/hosts.toml
```

**2. Ingress 无法访问**

```bash
# 检查 Ingress Controller 状态
kubectl get pods -n ingress-nginx -o wide

# 检查 Ingress 配置
kubectl get ingress -n blog
kubectl describe ingress blog-prod -n blog

# 检查 Service 和 Endpoints
kubectl get svc,endpoints -n blog
```

**3. ArgoCD 同步失败**

```bash
# 查看 Application 状态
kubectl get application blog-cluster1 -n argocd

# 查看 ArgoCD 日志
kubectl logs -n argocd deployment/argocd-application-controller

# 手动同步
argocd app sync blog-cluster1
```

### 9.2 调试命令

```bash
# 测试 Pod 内部访问
kubectl exec -n blog deployment/blog -- curl -s http://localhost/

# 测试 Service 访问
kubectl run test --image=curlimages/curl --rm -it --restart=Never --namespace=blog -- \
  curl -s http://blog/

# 测试 Ingress 访问
kubectl run test --image=curlimages/curl --rm -it --restart=Never --namespace=blog -- \
  curl -s -H "Host: local.blog" http://ingress-nginx-controller.ingress-nginx.svc.cluster.local/
```

## 十、总结

本文详细介绍了一个完整的博客本地 Kubernetes 部署架构，主要包括：

1. **双仓库设计**：代码仓库与部署清单分离，实现关注点分离
2. **双集群架构**：支持多集群部署，提高可用性
3. **Harbor 镜像仓库**：本地化镜像存储，加速镜像拉取
4. **Helm Chart**：模板化配置，支持多环境部署
5. **Jenkins CI/CD**：自动化构建和部署流程
6. **ArgoCD GitOps**：声明式部署，实现自动化同步

这套架构具有以下优点：

- **可追溯性**：所有变更都通过 Git 管理，便于审计和回滚
- **可重复性**：Helm 模板确保配置一致性
- **自动化**：从代码提交到部署完全自动化
- **可扩展性**：支持多集群、多环境部署

## 相关问答

**Q1: 为什么使用双仓库设计而不是单仓库？**

双仓库设计将代码开发和部署配置分离，有以下优势：
- 开发者只需关注代码仓库，不需要了解 Kubernetes 配置
- 部署清单由 CI/CD 自动生成，避免人为错误
- 权限分离，开发者和运维人员可以有不同的访问权限

**Q2: Harbor 为什么使用 HTTP 而不是 HTTPS？**

在本地开发环境中，配置 HTTPS 需要额外的证书管理。使用 HTTP 可以简化配置，但需要在所有节点配置 insecure registry。生产环境建议使用 HTTPS。

**Q3: ArgoCD 如何保证部署的一致性？**

ArgoCD 通过以下机制保证一致性：
- 定期轮询 Git 仓库（默认 3 分钟）
- 对比期望状态和实际状态
- 自动同步（配置 selfHeal）确保实际状态与期望状态一致

**Q4: 如何实现多集群部署？**

多集群部署需要：
- 在 ArgoCD 中注册多个集群
- 为每个集群创建独立的 Application
- 使用不同的 values 文件区分集群配置

**Q5: 如何实现 Canary 发布？**

通过 Helm Chart 的 Canary 配置：
```yaml
canary:
  enabled: true
  weight: 10  # 10% 流量到 Canary
  header:
    enabled: true
    name: X-Canary
    value: "true"
```
Nginx Ingress 会根据配置自动分流流量。

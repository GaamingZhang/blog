# ArgoCD 配置指南

## 一、安装 ArgoCD

### 1.1 在集群1（192.168.31.30）上安装

```bash
# 创建 namespace
kubectl create namespace argocd

# 安装 ArgoCD
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# 等待 ArgoCD 就绪
kubectl wait --for=condition=available --timeout=600s deployment/argocd-server -n argocd

# 获取初始密码
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo

# 端口转发（可选）
kubectl port-forward svc/argocd-server -n argocd 8080:443 --address 0.0.0.0 &
```

### 1.2 访问 ArgoCD UI

```bash
# 通过端口转发访问
# 浏览器打开: https://192.168.31.30:8080
# 用户名: admin
# 密码: 上一步获取的初始密码
```

## 二、配置 SSH Known Hosts

### 2.1 获取 Git 服务器的 SSH 密钥

```bash
# 获取 Git 服务器的 SSH 密钥
ssh-keyscan 192.168.31.50

# 输出示例：
# 192.168.31.50 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ...
```

### 2.2 配置 ArgoCD Known Hosts

```bash
# 编辑 ConfigMap
kubectl edit cm argocd-ssh-known-hosts-cm -n argocd
```

在 `data` 部分添加：

```yaml
data:
  ssh_known_hosts: |
    # 现有内容...
    192.168.31.50 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ...
```

或者使用 patch 方式：

```bash
# 创建 patch 文件
cat <<EOF > ssh-known-hosts-patch.yaml
data:
  ssh_known_hosts: |
    # 现有内容需要保留，添加新的内容
    192.168.31.50 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ...
EOF

# 应用 patch
kubectl patch cm argocd-ssh-known-hosts-cm -n argocd --patch-file ssh-known-hosts-patch.yaml
```

## 三、配置 SSH 私钥

### 3.1 创建 SSH 私钥 Secret

```bash
# 使用 Jenkins 的 SSH 密钥创建 Secret
kubectl create secret generic argocd-repo-ssh-key \
  --from-file=sshPrivateKey=/Users/gaamingzhang/git/gaamingzhangblog/internal_scripts/id_rsa_jenkins \
  --namespace=argocd

# 给 Secret 添加标签（重要：ArgoCD 通过此标签识别仓库凭证）
kubectl label secret argocd-repo-ssh-key -n argocd \
  argocd.argoproj.io/secret-type=repository
```

### 3.2 验证 Secret 创建成功

```bash
# 查看 Secret
kubectl get secret argocd-repo-ssh-key -n argocd

# 查看 Secret 详情
kubectl describe secret argocd-repo-ssh-key -n argocd
```

## 四、配置多集群（可选）

如果需要部署到多个集群，需要将 Cluster2 注册到 ArgoCD：

### 4.1 在 Cluster2 上创建 ServiceAccount

```bash
# 切换到 Cluster2 的 context
kubectl config use-context cluster2

# 创建 ServiceAccount
kubectl create serviceaccount argocd-manager -n kube-system

# 创建 ClusterRoleBinding
kubectl create clusterrolebinding argocd-manager-role-binding \
  --clusterrole=cluster-admin \
  --serviceaccount=kube-system:argocd-manager
```

### 4.2 在 Cluster1 上注册 Cluster2

```bash
# 切换回 Cluster1 的 context
kubectl config use-context cluster1

# 获取 Cluster2 的 API Server 地址
CLUSTER2_API_SERVER="https://192.168.31.31:6443"

# 注册 Cluster2
argocd cluster add cluster2 \
  --kubeconfig /path/to/cluster2/kubeconfig \
  --name cluster2
```

## 五、应用 ArgoCD 配置

### 5.1 应用 Project 和 Applications

```bash
# 方式一：使用 kustomize 应用所有配置
kubectl apply -k /Users/gaamingzhang/git/gaamingzhangblog/k8s/kustomize/argocd

# 方式二：单独应用
kubectl apply -f /Users/gaamingzhang/git/gaamingzhangblog/k8s/kustomize/argocd/projects/blog-project.yaml
kubectl apply -f /Users/gaamingzhang/git/gaamingzhangblog/k8s/kustomize/argocd/applications/blog-cluster1.yaml
kubectl apply -f /Users/gaamingzhang/git/gaamingzhangblog/k8s/kustomize/argocd/applications/blog-cluster2.yaml
```

### 5.2 验证配置应用成功

```bash
# 查看 Project
kubectl get appprojects -n argocd

# 查看 Applications
kubectl get applications -n argocd

# 查看 Application 详细信息
kubectl describe application blog-cluster1 -n argocd
```

## 六、Helm Chart 结构

### 6.1 目录结构

```
k8s/helm/blog/
├── Chart.yaml              # Chart 元数据
├── values.yaml             # 默认配置值
├── values-cluster1.yaml    # Cluster1 集群配置
├── values-cluster2.yaml    # Cluster2 集群配置
└── templates/
    ├── _helpers.tpl        # 模板辅助函数
    ├── deployment.yaml     # Deployment 模板
    ├── service.yaml        # Service 模板
    ├── ingress.yaml        # Ingress 模板
    └── ingress-canary.yaml # Canary Ingress 模板
```

### 6.2 主要配置项

**values.yaml 默认配置：**

```yaml
replicaCount: 2

image:
  repository: gaaming/blog
  tag: latest
  pullPolicy: Always

service:
  type: ClusterIP
  port: 80

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: local.blog
      paths:
        - path: /
          pathType: Prefix
```

### 6.3 本地渲染测试

```bash
# 渲染 Cluster1 配置
helm template blog k8s/helm/blog \
  --namespace blog \
  --set image.repository=192.168.31.30:30002/gaaming/blog \
  --set image.tag=v1.0.0 \
  --set cluster=cluster1 \
  -f k8s/helm/blog/values-cluster1.yaml

# 渲染 Cluster2 配置
helm template blog k8s/helm/blog \
  --namespace blog \
  --set image.repository=192.168.31.31:30002/gaaming/blog \
  --set image.tag=v1.0.0 \
  --set cluster=cluster2 \
  -f k8s/helm/blog/values-cluster2.yaml
```

## 七、验证部署状态

### 7.1 使用 kubectl 验证

```bash
# 查看 Applications 状态
kubectl get applications -n argocd

# 查看 Project
kubectl get appprojects -n argocd

# 查看 Application 详细信息
kubectl describe application blog-cluster1 -n argocd

# 查看 blog namespace 的部署状态
kubectl get all -n blog
kubectl get ingress -n blog
```

### 7.2 使用 ArgoCD CLI

```bash
# 登录 ArgoCD
argocd login 192.168.31.30:8080 --grpc-web

# 查看 Applications
argocd app list

# 查看 Application 详情
argocd app get blog-cluster1

# 手动同步 Application
argocd app sync blog-cluster1
argocd app sync blog-cluster2

# 查看同步状态
argocd app history blog-cluster1
```

### 7.3 使用 ArgoCD UI 验证

1. 打开浏览器访问 ArgoCD UI
2. 登录后可以看到 `blog-cluster1` 和 `blog-cluster2` 两个 Application
3. 点击 Application 可以查看详细的资源状态
4. 状态说明：
   - 🟢 Synced：已同步
   - 🟡 OutOfSync：需要同步
   - 🔴 Degraded：部署异常

## 八、CI/CD 流程

### 8.1 完整部署流程

```
┌─────────────────────────────────────────────────────────────────┐
│                    部署流程图                                     │
├─────────────────────────────────────────────────────────────────┤
│  1. 修改 Helm Chart                                              │
│     └─ 在本仓库中修改 k8s/helm/blog/ 下的文件                     │
├─────────────────────────────────────────────────────────────────┤
│  2. 提交代码                                                      │
│     └─ git commit & push 到 GitLab                               │
├─────────────────────────────────────────────────────────────────┤
│  3. Jenkins 构建                                                  │
│     ├─ 构建镜像并推送到 Harbor                                    │
│     └─ 使用 Helm 渲染 Kubernetes 清单                             │
├─────────────────────────────────────────────────────────────────┤
│  4. 推送到 ArgoCD 仓库                                            │
│     └─ 渲染后的清单推送到 gaamingblogkubernetesargocd 仓库        │
├─────────────────────────────────────────────────────────────────┤
│  5. ArgoCD 同步                                                   │
│     └─ 自动检测变更并部署到 Kubernetes 集群                        │
└─────────────────────────────────────────────────────────────────┘
```

### 8.2 Jenkins 流水线阶段

```
┌─────────────────────────────────────────────────────────────────┐
│                    Jenkins Pipeline                              │
├─────────────────────────────────────────────────────────────────┤
│  1. Trigger updateVersion                                        │
│     └─ 触发版本更新任务                                           │
├─────────────────────────────────────────────────────────────────┤
│  2. Checkout Official Branch                                     │
│     └─ 检出 official.{version} 分支                               │
├─────────────────────────────────────────────────────────────────┤
│  3. Build Image                                                  │
│     └─ 构建 Docker 镜像                                          │
├─────────────────────────────────────────────────────────────────┤
│  4. Push to Harbor - Cluster1                                    │
│     └─ 推送镜像到 Cluster1 的 Harbor                              │
├─────────────────────────────────────────────────────────────────┤
│  5. Push to Harbor - Cluster2                                    │
│     └─ 推送镜像到 Cluster2 的 Harbor                              │
├─────────────────────────────────────────────────────────────────┤
│  6. Render and Push to ArgoCD                                    │
│     ├─ 克隆 ArgoCD 仓库                                          │
│     ├─ 使用 Helm 渲染 Kubernetes 清单                             │
│     └─ 推送渲染后的清单到 ArgoCD 仓库                              │
└─────────────────────────────────────────────────────────────────┘
```

### 8.3 Helm 渲染命令

Jenkins 流水线中使用以下命令渲染清单：

```bash
# Cluster1
helm template blog ./helm-chart \
  --namespace blog \
  --set image.repository=192.168.31.30:30002/gaaming/blog \
  --set image.tag=${IMAGE_TAG} \
  --set cluster=cluster1 \
  -f values-cluster1.yaml \
  > apps/blog/cluster1/all.yaml

# Cluster2
helm template blog ./helm-chart \
  --namespace blog \
  --set image.repository=192.168.31.31:30002/gaaming/blog \
  --set image.tag=${IMAGE_TAG} \
  --set cluster=cluster2 \
  -f values-cluster2.yaml \
  > apps/blog/cluster2/all.yaml
```

## 九、故障排查

### 9.1 常见问题

1. **SSH 认证失败**
   ```bash
   # 检查 SSH 私钥是否正确
   kubectl get secret argocd-repo-ssh-key -n argocd -o yaml
   
   # 确保 Git 服务器的公钥已添加到 known_hosts
   kubectl get cm argocd-ssh-known-hosts-cm -n argocd -o yaml
   ```

2. **仓库访问失败**
   ```bash
   # 检查网络连接
   ping 192.168.31.50
   
   # 验证 Git 仓库权限
   ssh -T git@192.168.31.50
   ```

3. **Helm 渲染失败**
   ```bash
   # 检查 Helm Chart 语法
   helm lint k8s/helm/blog
   
   # 验证 values 文件格式
   helm template blog k8s/helm/blog --debug
   ```

4. **ArgoCD 同步失败**
   ```bash
   # 检查 Kubernetes 资源配置
   kubectl describe application blog-cluster1 -n argocd
   
   # 验证镜像拉取权限
   kubectl get secrets -n blog
   
   # 查看 ArgoCD 日志
   kubectl logs -n argocd deployment/argocd-server
   kubectl logs -n argocd deployment/argocd-repo-server
   ```

5. **Application 状态异常**
   ```bash
   # 查看 Application 事件
   kubectl describe application blog-cluster1 -n argocd
   
   # 查看 Application 日志
   argocd app logs blog-cluster1
   
   # 强制刷新
   argocd app get blog-cluster1 --refresh
   ```

### 9.2 调试命令

```bash
# 检查 Helm Chart 语法
helm lint k8s/helm/blog

# 查看渲染后的清单（不安装）
helm template blog k8s/helm/blog -f k8s/helm/blog/values-cluster1.yaml

# 查看 ArgoCD Application 状态
argocd app get blog-cluster1

# 手动同步
argocd app sync blog-cluster1

# 查看同步日志
argocd app logs blog-cluster1

# 查看 ArgoCD 组件状态
argocd admin cluster stats
```

## 十、配置 Canary 发布

### 10.1 启用 Canary

修改 values 文件启用 Canary 发布：

```yaml
canary:
  enabled: true
  weight: 10  # 10% 流量到 Canary
  header:
    enabled: true
    name: X-Canary
    value: "true"
```

### 10.2 Canary 发布流程

1. 修改 `values-cluster1.yaml` 或 `values-cluster2.yaml`
2. 提交代码触发流水线
3. ArgoCD 自动同步新配置
4. Nginx Ingress 根据 Canary 配置分流流量

## 十一、仓库说明

### 11.1 仓库关系

```
┌─────────────────────────────────────────────────────────────────┐
│                    blog (主仓库)                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  k8s/helm/blog/                                         │    │
│  │  ├── Chart.yaml                                         │    │
│  │  ├── values.yaml                                        │    │
│  │  ├── values-cluster1.yaml                               │    │
│  │  ├── values-cluster2.yaml                               │    │
│  │  └── templates/                                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              │ 提交代码                          │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Jenkins Pipeline                      │    │
│  │  1. 构建镜像 → 推送到 Harbor                              │    │
│  │  2. Helm 渲染 → 推送到 ArgoCD 仓库                        │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                               │
                               │ Helm 渲染结果
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│           gaamingblogkubernetesargocd (ArgoCD 仓库)              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  apps/blog/                                              │    │
│  │  ├── cluster1/all.yaml  (渲染后的 Kubernetes 清单)        │    │
│  │  └── cluster2/all.yaml                                   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              │ ArgoCD 监听                       │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Kubernetes 集群                        │    │
│  │  ├── Cluster1 (192.168.31.30)                           │    │
│  │  └── Cluster2 (192.168.31.31)                           │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 11.2 文件变更位置

| 变更类型 | 修改位置 | 说明 |
|---------|---------|------|
| 修改部署配置 | `k8s/helm/blog/values.yaml` | 副本数、资源限制等 |
| 修改集群配置 | `k8s/helm/blog/values-cluster1.yaml` | Cluster1 特定配置 |
| 修改 Ingress | `k8s/helm/blog/templates/ingress.yaml` | Ingress 模板 |
| 修改 Deployment | `k8s/helm/blog/templates/deployment.yaml` | Deployment 模板 |

## 十二、快速配置步骤（完整流程）

### 12.1 首次配置

```bash
# 步骤 1: 安装 ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl wait --for=condition=available --timeout=600s deployment/argocd-server -n argocd

# 步骤 2: 获取初始密码
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo

# 步骤 3: 配置 SSH Known Hosts
ssh-keyscan 192.168.31.50 | kubectl create configmap argocd-ssh-known-hosts-cm --from-file=ssh_known_hosts=/dev/stdin -n argocd --dry-run=client -o yaml | kubectl apply -f -

# 步骤 4: 配置 SSH 私钥
kubectl create secret generic argocd-repo-ssh-key \
  --from-file=sshPrivateKey=/path/to/your/ssh/key \
  --namespace=argocd
kubectl label secret argocd-repo-ssh-key -n argocd argocd.argoproj.io/secret-type=repository

# 步骤 5: 应用 ArgoCD 配置
kubectl apply -k /Users/gaamingzhang/git/gaamingzhangblog/k8s/kustomize/argocd

# 步骤 6: 验证部署
kubectl get applications -n argocd
kubectl get all -n blog
```

### 12.2 日常操作

```bash
# 查看部署状态
kubectl get applications -n argocd

# 手动同步
argocd app sync blog-cluster1

# 查看部署历史
argocd app history blog-cluster1

# 回滚到指定版本
argocd app rollback blog-cluster1 <revision>
```

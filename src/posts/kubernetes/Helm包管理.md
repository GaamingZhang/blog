# Helm 包管理

## 为什么需要 Helm？

在 Kubernetes 中部署一个应用，你可能需要创建 Deployment、Service、ConfigMap、Secret、Ingress 等一系列资源。每个资源都是一个 YAML 文件，一个简单的应用可能就需要 5-10 个文件。

现在想象一下：
- 你要部署这个应用到开发、测试、生产三个环境，每个环境的配置略有不同
- 你要升级应用版本，需要确保所有资源一起更新
- 升级出问题了，你要回滚到之前的版本

如果手动管理这些 YAML 文件，很容易出错。这就是 Helm 要解决的问题。

**Helm 就像是 Kubernetes 的 apt 或 yum**。它把一组相关的 Kubernetes 资源打包成一个"Chart"，让你可以像安装软件包一样部署应用。

## 核心概念

理解 Helm 需要先搞清楚三个概念：

### Chart（图表）

Chart 是 Helm 的包格式，包含了部署一个应用所需的所有 Kubernetes 资源定义。你可以把它理解为一个"安装包"。

比如，一个 Nginx 的 Chart 可能包含：
- Deployment 定义（怎么运行 Nginx）
- Service 定义（怎么访问 Nginx）
- ConfigMap 定义（Nginx 的配置文件）

### Release（发布）

当你安装一个 Chart 时，就会创建一个 Release。Release 是 Chart 的一个运行实例。

同一个 Chart 可以安装多次，每次安装都是一个独立的 Release。比如你可以用同一个 MySQL Chart 创建 `mysql-dev` 和 `mysql-prod` 两个 Release。

### Values（配置值）

Values 是 Chart 的可配置参数。Chart 的作者会定义哪些参数可以配置，使用者在安装时可以覆盖这些参数。

比如，你可以通过 Values 指定：
- 要部署几个副本
- 使用哪个版本的镜像
- 分配多少 CPU 和内存

## Helm 的工作原理

Helm 的核心是**模板引擎**。Chart 中的资源定义其实是模板文件，包含了一些占位符。安装时，Helm 会用你提供的 Values 替换这些占位符，生成最终的 Kubernetes 资源文件，然后提交给集群。

```
Chart 模板 + Values 配置 → Helm 渲染 → Kubernetes 资源 → 提交到集群
```

这样，一套模板就可以适应多种环境，你只需要修改 Values 即可。

## 基本使用

### 安装应用

```bash
# 添加一个 Chart 仓库
helm repo add bitnami https://charts.bitnami.com/bitnami

# 更新仓库索引
helm repo update

# 安装一个应用
helm install my-nginx bitnami/nginx
```

这三行命令就完成了 Nginx 的部署。`my-nginx` 是 Release 名称，`bitnami/nginx` 是 Chart 名称。

### 自定义配置

你可以通过 `--set` 覆盖默认配置：

```bash
# 部署3个副本
helm install my-nginx bitnami/nginx --set replicaCount=3
```

或者把配置写在文件里：

```yaml
# my-values.yaml
replicaCount: 3
service:
  type: LoadBalancer
```

```bash
helm install my-nginx bitnami/nginx -f my-values.yaml
```

### 升级和回滚

```bash
# 升级（修改配置或更新版本）
helm upgrade my-nginx bitnami/nginx --set replicaCount=5

# 查看历史版本
helm history my-nginx

# 回滚到上一个版本
helm rollback my-nginx

# 回滚到指定版本
helm rollback my-nginx 1
```

### 查看和卸载

```bash
# 查看已安装的 Release
helm list

# 查看 Release 状态
helm status my-nginx

# 卸载
helm uninstall my-nginx
```

## Chart 的结构

如果你想创建自己的 Chart，需要了解它的目录结构：

```
mychart/
├── Chart.yaml      # Chart 的元信息（名称、版本等）
├── values.yaml     # 默认配置值
├── templates/      # Kubernetes 资源模板
│   ├── deployment.yaml
│   ├── service.yaml
│   └── _helpers.tpl  # 可复用的模板片段
└── charts/         # 依赖的其他 Chart
```

### Chart.yaml

描述这个 Chart 是什么：

```yaml
apiVersion: v2
name: myapp
description: 我的应用
version: 1.0.0        # Chart 版本
appVersion: "2.0.0"   # 应用版本
```

注意区分两个版本号：
- `version`：Chart 本身的版本，每次修改 Chart 时更新
- `appVersion`：Chart 部署的应用版本

### values.yaml

定义可配置的参数和默认值：

```yaml
replicaCount: 2

image:
  repository: myapp
  tag: "1.0.0"

service:
  type: ClusterIP
  port: 80

resources:
  limits:
    cpu: 500m
    memory: 256Mi
```

### 模板语法

模板文件使用 Go 的模板语法，通过 `{{ }}` 引用 Values：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}-myapp
spec:
  replicas: {{ .Values.replicaCount }}
  template:
    spec:
      containers:
        - name: myapp
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
```

常用的内置对象：
- `.Values`：用户提供的配置值
- `.Release.Name`：Release 的名称
- `.Chart.Name`：Chart 的名称
- `.Chart.Version`：Chart 的版本

## 依赖管理

一个应用可能依赖其他服务，比如 Web 应用依赖 Redis。Helm 支持在 Chart.yaml 中声明依赖：

```yaml
dependencies:
  - name: redis
    version: "17.x.x"
    repository: https://charts.bitnami.com/bitnami
    condition: redis.enabled  # 可以通过 values 控制是否启用
```

然后运行 `helm dependency update` 下载依赖。安装时，依赖的 Chart 会一起安装。

## 调试技巧

在实际安装之前，你可以先看看 Helm 会生成什么：

```bash
# 渲染模板但不安装
helm template my-nginx bitnami/nginx -f my-values.yaml

# 模拟安装并显示调试信息
helm install my-nginx bitnami/nginx --debug --dry-run

# 检查 Chart 是否有问题
helm lint ./mychart
```

## 常见问题

### Q1: Release 升级失败了怎么办？

首先，不要慌。Helm 会保留历史版本，你可以回滚：

```bash
# 查看历史版本
helm history my-release

# 回滚到上一个正常的版本
helm rollback my-release 2
```

如果需要分析失败原因，可以查看 Pod 状态和事件：

```bash
kubectl get pods
kubectl describe pod <pod-name>
```

### Q2: 如何查看一个 Chart 有哪些可配置参数？

```bash
# 查看 Chart 的默认 values
helm show values bitnami/nginx

# 查看已安装 Release 的配置
helm get values my-nginx
```

### Q3: 多环境部署怎么管理？

推荐为每个环境创建一个 values 文件：

```
values-dev.yaml
values-staging.yaml
values-prod.yaml
```

部署时指定对应的文件：

```bash
helm install my-app ./mychart -f values-prod.yaml
```

你也可以叠加多个 values 文件，后面的会覆盖前面的：

```bash
helm install my-app ./mychart -f values.yaml -f values-prod.yaml
```

### Q4: 如何处理敏感信息（密码、密钥）？

不要把敏感信息直接写在 values 文件里提交到 Git。几个选择：

1. **引用已存在的 Secret**：在集群中预先创建 Secret，Chart 中只引用它的名字
2. **使用 Sealed Secrets**：加密后的 Secret 可以安全地提交到 Git
3. **CI/CD 中动态注入**：在部署流水线中通过 `--set` 传入敏感值

### Q5: Helm 2 和 Helm 3 有什么区别？

主要区别是 Helm 3 移除了 Tiller 组件。

在 Helm 2 中，需要在集群中部署一个叫 Tiller 的服务端组件，Helm 客户端通过它操作集群。这带来了安全和权限管理的复杂性。

Helm 3 直接使用你的 kubeconfig 凭证操作集群，更简单、更安全。如果你现在开始学习 Helm，直接用 Helm 3 即可。

## 总结

Helm 解决了 Kubernetes 应用部署的几个核心问题：

- **打包**：把多个资源打包成一个 Chart，方便分发和复用
- **模板化**：一套模板适应多种环境，通过 Values 差异化配置
- **版本管理**：跟踪每次部署，支持回滚
- **依赖管理**：自动处理应用之间的依赖关系

对于初学者，建议先熟悉基本的安装、升级、回滚命令，然后再学习如何创建自己的 Chart。Helm 的官方文档和社区 Chart 都是很好的学习资源。

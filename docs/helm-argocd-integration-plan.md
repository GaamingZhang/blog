# Helm + ArgoCD 集成部署计划

## 一、现状分析

### 1.1 当前部署架构

```
┌─────────────────────────────────────────────────────────────┐
│                    当前部署流程                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   Jenkins Pipeline                                          │
│       │                                                      │
│       ├─ pnpm install & build                               │
│       ├─ docker build & push → 192.168.31.40:30500          │
│       └─ kubectl apply -f k8s/*.yaml                        │
│                                                              │
│   问题：                                                      │
│   1. 手动管理多个 YAML 文件，容易出错                          │
│   2. 缺乏版本回滚机制                                         │
│   3. 多环境配置管理困难                                       │
│   4. 金丝雀发布流程复杂                                       │
│   5. 缺乏 GitOps 实践                                        │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 现有资源清单

| 资源类型 | 文件 | 说明 |
|---------|------|------|
| Deployment | `k8s/deployment.yaml` | 主应用部署（2副本） |
| Deployment | `k8s/deployment-canary.yaml` | 金丝雀版本部署 |
| Deployment | `k8s/deployment-stable.yaml` | 稳定版本部署 |
| Service | `k8s/service.yaml` | NodePort 服务（30080） |
| Ingress | `k8s/ingress.yaml` | Nginx Ingress（blog.local） |
| Istio | `k8s/istio/*.yaml` | 金丝雀流量管理 |
| Registry | `k8s/registry/*.yaml` | 私有镜像仓库 |

### 1.3 镜像仓库配置

- **私有仓库**: `192.168.31.40:30500` / `192.168.31.54:5001`
- **认证**: `registry-credentials` Secret
- **镜像**: `gaamingzhang-blog:latest` / `gaamingzhang-blog:canary`

---

## 二、目标架构

```
┌─────────────────────────────────────────────────────────────┐
│                    目标 GitOps 架构                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   Git Repository (GitOps 仓库)                               │
│       │                                                      │
│       ├─ charts/gaamingzhang-blog/     # Helm Chart         │
│       ├─ argocd/                       # ArgoCD 配置         │
│       └─ environments/                  # 多环境配置         │
│           ├─ dev/                                            │
│           └─ prod/                                           │
│                                                              │
│   ArgoCD (持续部署)                                          │
│       │                                                      │
│       ├─ 监听 Git 仓库变更                                    │
│       ├─ Helm 渲染 Manifest                                  │
│       └─ 自动同步到 K8s 集群                                  │
│                                                              │
│   Jenkins Pipeline (持续集成)                                │
│       │                                                      │
│       ├─ 代码构建 & 测试                                     │
│       ├─ 镜像构建 & 推送                                     │
│       └─ 更新 GitOps 仓库镜像版本                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 三、实施计划

### 阶段一：Helm 安装与配置（预计 2-3 小时）

#### 3.1.1 安装 Helm 客户端

```bash
# macOS
brew install helm

# 或使用官方脚本
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# 验证安装
helm version
```

#### 3.1.2 创建项目 Helm Chart 结构

```bash
# 在项目根目录创建 charts 目录
mkdir -p charts/gaamingzhang-blog

# 创建 Chart 骨架
helm create charts/gaamingzhang-blog

# 删除默认模板，保留需要的
rm -rf charts/gaamingzhang-blog/templates/*
```

**目标目录结构**：

```
charts/gaamingzhang-blog/
├── Chart.yaml              # Chart 元数据
├── values.yaml             # 默认配置
├── values-dev.yaml         # 开发环境配置
├── values-prod.yaml        # 生产环境配置
├── templates/
│   ├── _helpers.tpl        # 模板函数
│   ├── deployment.yaml     # Deployment 模板
│   ├── service.yaml        # Service 模板
│   ├── ingress.yaml        # Ingress 模板
│   ├── configmap.yaml      # ConfigMap 模板
│   ├── hpa.yaml            # HPA 模板
│   └── NOTES.txt           # 安装说明
├── charts/                 # 子 Chart 依赖
└── templates/tests/        # 测试模板
    └── test-connection.yaml
```

#### 3.1.3 编写 Chart.yaml

```yaml
apiVersion: v2
name: gaamingzhang-blog
description: A Helm chart for Jiaming Zhang's personal blog

type: application

version: 1.0.0
appVersion: "1.0.0"

kubeVersion: ">=1.21.0-0"

home: https://github.com/gaamingzhang/gaamingzhangblog
sources:
  - https://github.com/gaamingzhang/gaamingzhangblog

maintainers:
  - name: gaamingzhang
    email: your-email@example.com

keywords:
  - blog
  - vuepress
  - nginx

annotations:
  artifacthub.io/license: Apache-2.0
```

#### 3.1.4 编写 values.yaml（默认配置）

```yaml
replicaCount: 2

image:
  repository: 192.168.31.40:30500/gaamingzhang-blog
  pullPolicy: Always
  tag: "latest"

imagePullSecrets:
  - name: registry-credentials

nameOverride: ""
fullnameOverride: ""

serviceAccount:
  create: false
  name: ""

podAnnotations: {}

podSecurityContext:
  fsGroup: 101

securityContext:
  runAsNonRoot: true
  runAsUser: 101
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL

service:
  type: NodePort
  port: 80
  nodePort: 30080

ingress:
  enabled: true
  className: "nginx"
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
  hosts:
    - host: blog.local
      paths:
        - path: /
          pathType: Prefix
  tls: []

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 256Mi

autoscaling:
  enabled: false
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilizationPercentage: 80

nodeSelector: {}
tolerations: []
affinity: {}

livenessProbe:
  httpGet:
    path: /
    port: http
  initialDelaySeconds: 15
  periodSeconds: 20

readinessProbe:
  httpGet:
    path: /
    port: http
  initialDelaySeconds: 5
  periodSeconds: 10

canary:
  enabled: false
  replicaCount: 1
  image:
    tag: "canary"
  weight: 0
```

#### 3.1.5 编写 values-prod.yaml（生产环境配置）

```yaml
replicaCount: 3

image:
  tag: "v1.0.0"

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 200m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

ingress:
  hosts:
    - host: gaaming.com.cn
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: blog-tls
      hosts:
        - gaaming.com.cn
```

#### 3.1.6 编写 _helpers.tpl

```yaml
{{- define "gaamingzhang-blog.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "gaamingzhang-blog.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "gaamingzhang-blog.labels" -}}
helm.sh/chart: {{ include "gaamingzhang-blog.name" . }}-{{ .Chart.Version }}
{{ include "gaamingzhang-blog.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "gaamingzhang-blog.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gaamingzhang-blog.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "gaamingzhang-blog.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "gaamingzhang-blog.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
```

#### 3.1.7 编写 templates/deployment.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "gaamingzhang-blog.fullname" . }}
  labels:
    {{- include "gaamingzhang-blog.labels" . | nindent 4 }}
spec:
  {{- if not .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "gaamingzhang-blog.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
        {{- with .Values.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      labels:
        {{- include "gaamingzhang-blog.selectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "gaamingzhang-blog.serviceAccountName" . }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: {{ .Chart.Name }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: 80
              protocol: TCP
          {{- if .Values.livenessProbe }}
          livenessProbe:
            {{- toYaml .Values.livenessProbe | nindent 12 }}
          {{- end }}
          {{- if .Values.readinessProbe }}
          readinessProbe:
            {{- toYaml .Values.readinessProbe | nindent 12 }}
          {{- end }}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          {{- if .Values.extraVolumeMounts }}
          volumeMounts:
            {{- toYaml .Values.extraVolumeMounts | nindent 12 }}
          {{- end }}
      {{- if .Values.extraVolumes }}
      volumes:
        {{- toYaml .Values.extraVolumes | nindent 8 }}
      {{- end }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
```

#### 3.1.8 编写 templates/service.yaml

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "gaamingzhang-blog.fullname" . }}
  labels:
    {{- include "gaamingzhang-blog.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: http
      protocol: TCP
      name: http
      {{- if and (eq .Values.service.type "NodePort") .Values.service.nodePort }}
      nodePort: {{ .Values.service.nodePort }}
      {{- end }}
  selector:
    {{- include "gaamingzhang-blog.selectorLabels" . | nindent 4 }}
```

#### 3.1.9 编写 templates/ingress.yaml

```yaml
{{- if .Values.ingress.enabled -}}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "gaamingzhang-blog.fullname" . }}
  labels:
    {{- include "gaamingzhang-blog.labels" . | nindent 4 }}
  {{- with .Values.ingress.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  ingressClassName: {{ .Values.ingress.className }}
  {{- if .Values.ingress.tls }}
  tls:
    {{- range .Values.ingress.tls }}
    - hosts:
        {{- range .hosts }}
        - {{ . | quote }}
        {{- end }}
      secretName: {{ .secretName }}
    {{- end }}
  {{- end }}
  rules:
    {{- range .Values.ingress.hosts }}
    - host: {{ .host | quote }}
      http:
        paths:
          {{- range .paths }}
          - path: {{ .path }}
            pathType: {{ .pathType }}
            backend:
              service:
                name: {{ include "gaamingzhang-blog.fullname" $ }}
                port:
                  number: {{ $.Values.service.port }}
          {{- end }}
    {{- end }}
{{- end }}
```

#### 3.1.10 验证 Helm Chart

```bash
# Lint 检查
helm lint charts/gaamingzhang-blog

# 渲染模板预览
helm template gaamingzhang-blog charts/gaamingzhang-blog \
  --namespace default \
  --values charts/gaamingzhang-blog/values.yaml

# 模拟安装
helm install gaamingzhang-blog charts/gaamingzhang-blog \
  --namespace default \
  --dry-run --debug

# 本地测试安装
helm upgrade --install gaamingzhang-blog charts/gaamingzhang-blog \
  --namespace default \
  --create-namespace

# 验证部署
kubectl get all -n default -l app.kubernetes.io/name=gaamingzhang-blog

# 卸载测试
helm uninstall gaamingzhang-blog --namespace default
```

---

### 阶段二：ArgoCD 安装与配置（预计 2-3 小时）

#### 3.2.1 安装 ArgoCD

```bash
# 创建命名空间
kubectl create namespace argocd

# 安装 ArgoCD
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# 等待所有 Pod 就绪
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server -n argocd --timeout=300s

# 检查 ArgoCD 组件状态
kubectl get pods -n argocd
kubectl get svc -n argocd
```

#### 3.2.2 访问 ArgoCD UI

```bash
# 获取初始管理员密码
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

# 方式一：Port-Forward 访问
kubectl port-forward svc/argocd-server -n argocd 8080:443

# 访问 https://localhost:8080
# 用户名: admin
# 密码: 上一步获取的密码

# 方式二：Ingress 访问（推荐）
cat <<EOF | kubectl apply -n argocd -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argocd-server-ingress
  annotations:
    nginx.ingress.kubernetes.io/ssl-passthrough: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
spec:
  ingressClassName: nginx
  rules:
  - host: argocd.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: argocd-server
            port:
              name: https
EOF

# 添加 hosts 解析
echo "<INGRESS_IP> argocd.local" | sudo tee -a /etc/hosts
```

#### 3.2.3 安装 ArgoCD CLI

```bash
# macOS
brew install argocd

# Linux
curl -sSL -o argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
sudo install -m 555 argocd /usr/local/bin/argocd

# 登录 ArgoCD
argocd login argocd.local --grpc-web

# 修改管理员密码
argocd account update-password
```

#### 3.2.4 创建 GitOps 仓库结构

```bash
# 在项目根目录创建 GitOps 配置目录
mkdir -p argocd/apps
mkdir -p argocd/projects
mkdir -p argocd/repositories
```

**目录结构**：

```
argocd/
├── apps/
│   └── gaamingzhang-blog.yaml    # Application 定义
├── projects/
│   └── blog-project.yaml         # AppProject 定义
└── repositories/
    └── gitlab-repo.yaml          # Repository 凭证
```

#### 3.2.5 创建 AppProject

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: blog-project
  namespace: argocd
spec:
  description: Jiaming Zhang Blog Project

  sourceRepos:
    - '*'  # 允许所有 Git 仓库，生产环境应限制

  destinations:
    - namespace: default
      server: https://kubernetes.default.svc
    - namespace: blog
      server: https://kubernetes.default.svc

  clusterResourceWhitelist:
    - group: ''
      kind: Namespace

  namespaceResourceWhitelist:
    - group: ''
      kind: Deployment
    - group: ''
      kind: Service
    - group: ''
      kind: ConfigMap
    - group: ''
      kind: Secret
    - group: networking.k8s.io
      kind: Ingress
    - group: autoscaling
      kind: HorizontalPodAutoscaler

  roles:
    - name: admin
      description: Admin privileges for blog project
      policies:
        - p, proj:blog-project:admin, applications, *, blog-project/*, allow
        - p, proj:blog-project:admin, repositories, *, blog-project/*, allow
      groups:
        - blog-admins
```

#### 3.2.6 创建 Application

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: gaamingzhang-blog
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: blog-project

  source:
    repoURL: http://192.168.31.50/gaamingzhang/gaamingzhangblog.git
    targetRevision: main
    path: charts/gaamingzhang-blog
    helm:
      valueFiles:
        - values.yaml
      parameters:
        - name: image.tag
          value: "latest"

  destination:
    server: https://kubernetes.default.svc
    namespace: default

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
      allowEmpty: false
    syncOptions:
      - Validate=true
      - CreateNamespace=true
      - PrunePropagationPolicy=foreground
      - PruneLast=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m

  ignoreDifferences:
    - group: apps
      kind: Deployment
      jsonPointers:
        - /spec/replicas
```

#### 3.2.7 配置 Git 仓库凭证

```bash
# 方式一：通过 ArgoCD CLI 添加仓库
argocd repo add http://192.168.31.50/gaamingzhang/gaamingzhangblog.git \
  --username <username> \
  --password <password>

# 方式二：通过 Secret 配置
cat <<EOF | kubectl apply -n argocd -f -
apiVersion: v1
kind: Secret
metadata:
  name: repo-gaamingzhangblog
  labels:
    argocd.argoproj.io/secret-type: repository
stringData:
  type: git
  url: http://192.168.31.50/gaamingzhang/gaamingzhangblog.git
  username: <username>
  password: <password>
EOF
```

#### 3.2.8 应用 ArgoCD 配置

```bash
# 应用 AppProject
kubectl apply -f argocd/projects/blog-project.yaml

# 应用 Application
kubectl apply -f argocd/apps/gaamingzhang-blog.yaml

# 查看 Application 状态
argocd app get gaamingzhang-blog

# 手动同步
argocd app sync gaamingzhang-blog
```

---

### 阶段三：GitOps 流程设计（预计 1-2 小时）

#### 3.3.1 GitOps 工作流

```
┌─────────────────────────────────────────────────────────────┐
│                    GitOps 工作流                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   1. 代码提交 → Jenkins CI                                   │
│      ├─ 构建镜像：gaamingzhang-blog:v1.0.0                   │
│      └─ 推送镜像到私有仓库                                   │
│                                                              │
│   2. Jenkins 更新 GitOps 仓库                                │
│      ├─ 更新 values.yaml 中的 image.tag                      │
│      └─ 提交到 GitOps 仓库                                   │
│                                                              │
│   3. ArgoCD 自动检测变更                                     │
│      ├─ 检测到 Git 仓库变更                                  │
│      ├─ 渲染 Helm Chart                                      │
│      └─ 自动同步到 K8s 集群                                  │
│                                                              │
│   4. 部署验证                                                │
│      ├─ ArgoCD 健康检查                                      │
│      └─ 自动回滚（如果失败）                                 │
└─────────────────────────────────────────────────────────────┘
```

#### 3.3.2 修改 Jenkins Pipeline

在 `pipelines/deployMyBlog.Jenkinsfile` 中添加 GitOps 阶段：

```groovy
stage('Update GitOps Repository') {
  when {
    expression { params.DEPLOY_TO_KUBERNETES == true }
  }
  steps {
    script {
      withCredentials([
        usernamePassword(credentialsId: 'gitlab-credentials', usernameVariable: 'GIT_USERNAME', passwordVariable: 'GIT_PASSWORD')
      ]) {
        sh '''
          set -e
          
          # 克隆 GitOps 仓库
          git clone http://${GIT_USERNAME}:${GIT_PASSWORD}@192.168.31.50/gaamingzhang/gaamingzhangblog.git gitops-repo
          cd gitops-repo
          
          # 更新 Helm values 中的镜像版本
          yq -i '.image.tag = "'${VERSION}'"' charts/gaamingzhang-blog/values.yaml
          
          # 提交并推送
          git config user.name "Jenkins CI"
          git config user.email "jenkins@example.com"
          git add charts/gaamingzhang-blog/values.yaml
          git commit -m "chore: update image tag to ${VERSION}"
          git push origin main
        '''
      }
    }
  }
}
```

#### 3.3.3 多环境配置

```bash
# 创建环境目录
mkdir -p environments/dev
mkdir -p environments/prod
```

**environments/dev/kustomization.yaml**：

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../charts/gaamingzhang-blog

patchesStrategicMerge:
  - values-patch.yaml
```

**environments/dev/values-patch.yaml**：

```yaml
image:
  tag: "dev-latest"

replicaCount: 1

resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 50m
    memory: 128Mi
```

#### 3.3.4 ArgoCD ApplicationSet（多环境）

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: gaamingzhang-blog-envs
  namespace: argocd
spec:
  generators:
    - list:
        elements:
          - env: dev
            namespace: blog-dev
          - env: prod
            namespace: blog-prod
  template:
    metadata:
      name: 'gaamingzhang-blog-{{env}}'
    spec:
      project: blog-project
      source:
        repoURL: http://192.168.31.50/gaamingzhang/gaamingzhangblog.git
        targetRevision: main
        path: charts/gaamingzhang-blog
        helm:
          valueFiles:
            - values-{{env}}.yaml
      destination:
        server: https://kubernetes.default.svc
        namespace: '{{namespace}}'
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

---

### 阶段四：金丝雀发布集成（可选，预计 2 小时）

#### 3.4.1 使用 ArgoCD Rollouts（推荐）

```bash
# 安装 ArgoCD Rollouts Controller
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml

# 安装 kubectl argo-rollouts 插件
curl -LO https://github.com/argoproj/argo-rollouts/releases/latest/download/kubectl-argo-rollouts-darwin-amd64
chmod +x ./kubectl-argo-rollouts-darwin-amd64
sudo mv ./kubectl-argo-rollouts-darwin-amd64 /usr/local/bin/kubectl-argo-rollouts
```

#### 3.4.2 创建 Rollout 资源

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: gaamingzhang-blog
  labels:
    {{- include "gaamingzhang-blog.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  revisionHistoryLimit: 3
  selector:
    matchLabels:
      {{- include "gaamingzhang-blog.selectorLabels" . | nindent 6 }}
  template:
    # ... 与 Deployment 相同
  strategy:
    canary:
      steps:
        - setWeight: 5
        - pause: {duration: 10m}
        - setWeight: 20
        - pause: {duration: 10m}
        - setWeight: 50
        - pause: {duration: 10m}
        - setWeight: 80
        - pause: {duration: 10m}
      analysis:
        templates:
          - templateName: success-rate
        startingStep: 2
        args:
          - name: service-name
            value: gaamingzhang-blog-service
```

---

## 四、验证清单

### 4.1 Helm Chart 验证

```bash
# 1. Lint 检查
helm lint charts/gaamingzhang-blog --strict

# 2. 模板渲染验证
helm template test-release charts/gaamingzhang-blog \
  --values charts/gaamingzhang-blog/values.yaml \
  --debug

# 3. 本地测试安装
helm upgrade --install gaamingzhang-blog charts/gaamingzhang-blog \
  --namespace default \
  --dry-run

# 4. 实际安装测试
helm upgrade --install gaamingzhang-blog charts/gaamingzhang-blog \
  --namespace default \
  --wait --timeout 5m

# 5. 验证部署状态
kubectl get all -n default -l app.kubernetes.io/name=gaamingzhang-blog

# 6. 测试回滚
helm rollback gaamingzhang-blog 1
```

### 4.2 ArgoCD 验证

```bash
# 1. 检查 ArgoCD 组件状态
kubectl get pods -n argocd

# 2. 检查 Application 状态
argocd app list
argocd app get gaamingzhang-blog

# 3. 手动同步测试
argocd app sync gaamingzhang-blog

# 4. 检查同步状态
argocd app history gaamingzhang-blog

# 5. 测试自动同步
# 修改 Git 仓库中的 values.yaml，观察 ArgoCD 是否自动同步

# 6. 测试回滚
argocd app rollback gaamingzhang-blog <revision>
```

### 4.3 GitOps 流程验证

```bash
# 1. 提交代码变更
git add .
git commit -m "test: verify GitOps flow"
git push

# 2. 观察 Jenkins Pipeline 执行
# 3. 观察 ArgoCD 自动同步
# 4. 验证部署结果
kubectl get pods -n default -l app.kubernetes.io/name=gaamingzhang-blog
```

---

## 五、迁移步骤

### 5.1 从现有 YAML 迁移到 Helm

```bash
# 1. 备份现有配置
cp -r k8s k8s.backup

# 2. 使用 Helm 安装
helm upgrade --install gaamingzhang-blog charts/gaamingzhang-blog \
  --namespace default \
  --values charts/gaamingzhang-blog/values.yaml

# 3. 验证迁移成功
kubectl get all -n default

# 4. 清理旧的 YAML 资源（确认 Helm 部署成功后）
# 注意：如果资源名称相同，Helm 会接管现有资源
```

### 5.2 切换到 ArgoCD 管理

```bash
# 1. 应用 ArgoCD Application
kubectl apply -f argocd/apps/gaamingzhang-blog.yaml

# 2. 验证 ArgoCD 接管
argocd app get gaamingzhang-blog

# 3. 禁用 Jenkins 中的 kubectl apply 阶段
# 修改 Jenkinsfile，移除 kubectl apply 相关步骤
```

---

## 六、常见问题排查

### 6.1 Helm 相关

| 问题 | 解决方案 |
|------|----------|
| Chart 渲染失败 | `helm template --debug` 检查模板语法 |
| 镜像拉取失败 | 检查 `imagePullSecrets` 配置 |
| 资源冲突 | 使用 `--force` 或手动清理旧资源 |

### 6.2 ArgoCD 相关

| 问题 | 解决方案 |
|------|----------|
| Application 一直 OutOfSync | 检查 `syncPolicy` 配置 |
| 同步失败 | `argocd app logs <app-name>` 查看日志 |
| Git 仓库连接失败 | 检查 Repository 凭证配置 |

---

## 七、下一步优化

1. **监控集成**：添加 Prometheus ServiceMonitor
2. **日志收集**：集成 Loki 日志系统
3. **密钥管理**：集成 External Secrets Operator
4. **多集群部署**：使用 ArgoCD ApplicationSet
5. **策略引擎**：集成 OPA Gatekeeper 或 Kyverno

---

## 八、参考资源

- [Helm 官方文档](https://helm.sh/docs/)
- [ArgoCD 官方文档](https://argo-cd.readthedocs.io/)
- [ArgoCD Rollouts 文档](https://argoproj.github.io/argo-rollouts/)
- [GitOps 最佳实践](https://www.gitops.tech/)

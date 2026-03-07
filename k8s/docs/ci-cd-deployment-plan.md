# GaamingZhang Blog CI/CD 部署流程设计

## 部署流程概述

```
代码提交 → Jenkins构建 → 推送镜像到Harbor → 渲染YAML → 提交到GitLab → ArgoCD检测 → 自动部署
```

## 一、GitLab仓库准备

### 1.1 创建GitLab项目

在GitLab中创建项目用于存放ArgoCD监控的YAML文件：
- 项目名称：`gaamingzhang-blog-k8s`
- 可见性：Private
- 初始化：README.md

### 1.2 目录结构

```
gaamingzhang-blog-k8s/
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml
│   ├── configmap.yaml
│   └── kustomization.yaml
├── overlays/
│   ├── prod/
│   │   ├── kustomization.yaml
│   │   └── patches/
│   │       └── deployment-replicas.yaml
│   └── canary/
│       ├── kustomization.yaml
│       └── patches/
│           └── deployment-replicas.yaml
└── README.md
```

## 二、Kubernetes部署文件

### 2.1 Base Deployment (base/deployment.yaml)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gaamingzhang-blog
  labels:
    app: gaamingzhang-blog
spec:
  replicas: 2
  selector:
    matchLabels:
      app: gaamingzhang-blog
  template:
    metadata:
      labels:
        app: gaamingzhang-blog
    spec:
      containers:
      - name: gaamingzhang-blog
        image: IMAGE_PLACEHOLDER
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 5
```

### 2.2 Base Service (base/service.yaml)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: gaamingzhang-blog
  labels:
    app: gaamingzhang-blog
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  selector:
    app: gaamingzhang-blog
```

### 2.3 Base Ingress (base/ingress.yaml)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gaamingzhang-blog
  annotations:
    kubernetes.io/ingress.class: nginx
spec:
  rules:
  - host: blog.gaaming.com.cn
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: gaamingzhang-blog
            port:
              number: 80
```

### 2.4 Base Kustomization (base/kustomization.yaml)

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- deployment.yaml
- service.yaml
- ingress.yaml

commonLabels:
  app: gaamingzhang-blog
```

### 2.5 Prod Overlay (overlays/prod/kustomization.yaml)

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: gaamingblog-prod

resources:
- ../../base

patchesStrategicMerge:
- patches/deployment-replicas.yaml

images:
- name: IMAGE_PLACEHOLDER
  newTag: latest
```

### 2.6 Prod Deployment Patch (overlays/prod/patches/deployment-replicas.yaml)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gaamingzhang-blog
spec:
  replicas: 2
```

### 2.7 Canary Overlay (overlays/canary/kustomization.yaml)

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: gaamingblog-canary

resources:
- ../../base

patchesStrategicMerge:
- patches/deployment-replicas.yaml

images:
- name: IMAGE_PLACEHOLDER
  newTag: latest
```

### 2.8 Canary Deployment Patch (overlays/canary/patches/deployment-replicas.yaml)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gaamingzhang-blog
spec:
  replicas: 2
```

## 三、Jenkins Pipeline脚本

### 3.1 新建Jenkinsfile (k8s/jenkins/deploy-to-k8s.Jenkinsfile)

```groovy
pipeline {
  agent any

  parameters {
    choice(name: 'DEPLOY_ENV', choices: ['prod', 'canary'], description: '部署环境')
    choice(name: 'TARGET_CLUSTER', choices: ['cluster1', 'cluster2', 'both'], description: '目标集群')
    string(name: 'IMAGE_TAG', defaultValue: '', description: '镜像标签（留空则使用BUILD_NUMBER）')
  }

  environment {
    HARBOR_URL_CLUSTER1 = '192.168.31.30:30003'
    HARBOR_URL_CLUSTER2 = '192.168.31.31:30003'
    IMAGE_NAME = 'gaamingzhang/blog'
    GITLAB_REPO_URL = 'git@gitlab.com:gaamingzhang/gaamingzhang-blog-k8s.git'
    GITLAB_BRANCH = 'main'
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Build Image') {
      steps {
        script {
          def imageTag = params.IMAGE_TAG ?: env.BUILD_NUMBER
          env.IMAGE_TAG = imageTag
          
          sh '''
            docker build -t ${IMAGE_NAME}:${IMAGE_TAG} .
          '''
        }
      }
    }

    stage('Push to Harbor') {
      steps {
        script {
          def imageTag = env.IMAGE_TAG
          
          // 推送到集群1的Harbor
          if (params.TARGET_CLUSTER == 'cluster1' || params.TARGET_CLUSTER == 'both') {
            withCredentials([usernamePassword(credentialsId: 'harbor-cluster1-credentials', usernameVariable: 'HARBOR_USER', passwordVariable: 'HARBOR_PASSWORD')]) {
              sh """
                docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${HARBOR_URL_CLUSTER1}/${IMAGE_NAME}:${IMAGE_TAG}
                docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${HARBOR_URL_CLUSTER1}/${IMAGE_NAME}:latest
                
                echo "${HARBOR_PASSWORD}" | docker login ${HARBOR_URL_CLUSTER1} -u "${HARBOR_USER}" --password-stdin
                
                docker push ${HARBOR_URL_CLUSTER1}/${IMAGE_NAME}:${IMAGE_TAG}
                docker push ${HARBOR_URL_CLUSTER1}/${IMAGE_NAME}:latest
              """
            }
          }
          
          // 推送到集群2的Harbor
          if (params.TARGET_CLUSTER == 'cluster2' || params.TARGET_CLUSTER == 'both') {
            withCredentials([usernamePassword(credentialsId: 'harbor-cluster2-credentials', usernameVariable: 'HARBOR_USER', passwordVariable: 'HARBOR_PASSWORD')]) {
              sh """
                docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${HARBOR_URL_CLUSTER2}/${IMAGE_NAME}:${IMAGE_TAG}
                docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${HARBOR_URL_CLUSTER2}/${IMAGE_NAME}:latest
                
                echo "${HARBOR_PASSWORD}" | docker login ${HARBOR_URL_CLUSTER2} -u "${HARBOR_USER}" --password-stdin
                
                docker push ${HARBOR_URL_CLUSTER2}/${IMAGE_NAME}:${IMAGE_TAG}
                docker push ${HARBOR_URL_CLUSTER2}/${IMAGE_NAME}:latest
              """
            }
          }
        }
      }
    }

    stage('Render YAML') {
      steps {
        script {
          def imageTag = env.IMAGE_TAG
          def deployEnv = params.DEPLOY_ENV
          
          sh """
            # 克隆GitLab仓库
            git clone ${GITLAB_REPO_URL} /tmp/k8s-yaml
            cd /tmp/k8s-yaml
            
            # 更新镜像标签
            if [ "${deployEnv}" == "prod" ]; then
              sed -i "s|newTag:.*|newTag: ${imageTag}|g" overlays/prod/kustomization.yaml
            else
              sed -i "s|newTag:.*|newTag: ${imageTag}|g" overlays/canary/kustomization.yaml
            fi
            
            # 提交更改
            git config user.name "Jenkins CI"
            git config user.email "jenkins@gaaming.com.cn"
            git add .
            git commit -m "Update ${deployEnv} image to ${imageTag}"
            git push origin ${GITLAB_BRANCH}
          """
        }
      }
    }

    stage('Verify Deployment') {
      steps {
        script {
          def deployEnv = params.DEPLOY_ENV
          def namespace = deployEnv == 'prod' ? 'gaamingblog-prod' : 'gaamingblog-canary'
          
          // 等待ArgoCD同步（最多5分钟）
          sh """
            echo "等待ArgoCD同步部署..."
            sleep 300
            
            # 使用kubectl验证部署状态（需要配置kubeconfig）
            kubectl get pods -n ${namespace} -l app=gaamingzhang-blog
          """
        }
      }
    }
  }

  post {
    success {
      echo "部署成功！"
    }
    failure {
      echo "部署失败！"
    }
  }
}
```

## 四、ArgoCD配置

### 4.1 创建ArgoCD Application

在ArgoCD中创建Application，监控GitLab仓库：

**集群1 - Prod环境**:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: gaamingzhang-blog-prod
  namespace: argocd
spec:
  project: default
  source:
    repoURL: git@gitlab.com:gaamingzhang/gaamingzhang-blog-k8s.git
    targetRevision: main
    path: overlays/prod
  destination:
    server: https://kubernetes.default.svc
    namespace: gaamingblog-prod
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

**集群1 - Canary环境**:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: gaamingzhang-blog-canary
  namespace: argocd
spec:
  project: default
  source:
    repoURL: git@gitlab.com:gaamingzhang/gaamingzhang-blog-k8s.git
    targetRevision: main
    path: overlays/canary
  destination:
    server: https://kubernetes.default.svc
    namespace: gaamingblog-canary
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

**集群2 - Prod环境**:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: gaamingzhang-blog-prod-cluster2
  namespace: argocd
spec:
  project: default
  source:
    repoURL: git@gitlab.com:gaamingzhang/gaamingzhang-blog-k8s.git
    targetRevision: main
    path: overlays/prod
  destination:
    server: https://192.168.31.31:6443
    namespace: gaamingblog-prod
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

**集群2 - Canary环境**:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: gaamingzhang-blog-canary-cluster2
  namespace: argocd
spec:
  project: default
  source:
    repoURL: git@gitlab.com:gaamingzhang/gaamingzhang-blog-k8s.git
    targetRevision: main
    path: overlays/canary
  destination:
    server: https://192.168.31.31:6443
    namespace: gaamingblog-canary
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## 五、部署脚本

### 5.1 渲染YAML脚本 (scripts/render-yaml.sh)

```bash
#!/bin/bash

set -e

DEPLOY_ENV=${1:-prod}
IMAGE_TAG=${2:-latest}
GITLAB_REPO="git@gitlab.com:gaamingzhang/gaamingzhang-blog-k8s.git"
WORK_DIR="/tmp/k8s-yaml-render"

echo "渲染YAML文件..."
echo "环境: ${DEPLOY_ENV}"
echo "镜像标签: ${IMAGE_TAG}"

# 克隆仓库
rm -rf ${WORK_DIR}
git clone ${GITLAB_REPO} ${WORK_DIR}
cd ${WORK_DIR}

# 更新镜像标签
if [ "${DEPLOY_ENV}" == "prod" ]; then
  sed -i "s|newTag:.*|newTag: ${IMAGE_TAG}|g" overlays/prod/kustomization.yaml
else
  sed -i "s|newTag:.*|newTag: ${IMAGE_TAG}|g" overlays/canary/kustomization.yaml
fi

# 提交更改
git config user.name "Jenkins CI"
git config user.email "jenkins@gaaming.com.cn"
git add .
git commit -m "Update ${DEPLOY_ENV} image to ${IMAGE_TAG}"
git push origin main

echo "YAML渲染完成并已提交到GitLab"
```

### 5.2 Git提交脚本 (scripts/git-commit.sh)

```bash
#!/bin/bash

set -e

REPO_DIR=${1:-/tmp/k8s-yaml}
COMMIT_MSG=${2:-"Update deployment"}

cd ${REPO_DIR}

git config user.name "Jenkins CI"
git config user.email "jenkins@gaaming.com.cn"

git add .

if git diff --staged --quiet; then
  echo "没有更改需要提交"
  exit 0
fi

git commit -m "${COMMIT_MSG}"
git push origin main

echo "Git提交完成"
```

## 六、Jenkins配置要求

### 6.1 需要的Jenkins凭据

1. **harbor-cluster1-credentials**: Harbor集群1的用户名密码
2. **harbor-cluster2-credentials**: Harbor集群2的用户名密码
3. **gitlab-ssh-key**: GitLab SSH私钥（用于推送YAML文件）
4. **kubernetes-kubeconfig-cluster1**: 集群1的kubeconfig
5. **kubernetes-kubeconfig-cluster2**: 集群2的kubeconfig

### 6.2 需要的Jenkins工具

- Docker
- Git
- kubectl

## 七、部署流程

### 7.1 完整部署流程

1. **开发提交代码** → GitHub
2. **Jenkins触发构建**:
   - 构建Docker镜像
   - 推送到Harbor（集群1和/或集群2）
   - 渲染YAML文件
   - 提交到GitLab
3. **ArgoCD检测变更**:
   - 自动同步GitLab变更
   - 应用到Kubernetes集群
4. **验证部署**:
   - 检查Pod状态
   - 检查服务可用性

### 7.2 环境说明

- **prod环境**: 生产环境，稳定版本
- **canary环境**: 金丝雀环境，新版本测试

### 7.3 集群说明

- **cluster1** (192.168.31.30): 主集群
- **cluster2** (192.168.31.31): 备集群

## 八、使用说明

### 8.1 首次部署

1. 创建GitLab项目 `gaamingzhang-blog-k8s`
2. 上传所有YAML文件到GitLab
3. 在ArgoCD中创建4个Application
4. 在Jenkins中配置Pipeline和凭据
5. 运行Jenkins Pipeline进行首次部署

### 8.2 日常部署

1. 提交代码到GitHub
2. Jenkins自动触发构建
3. 选择部署环境和目标集群
4. 等待ArgoCD自动部署
5. 验证部署结果

## 九、注意事项

1. **镜像仓库认证**: 确保Kubernetes节点已配置Harbor认证
2. **ArgoCD SSH密钥**: 需要配置GitLab SSH密钥
3. **网络策略**: 确保Jenkins可以访问GitLab和Harbor
4. **资源限制**: 根据实际需求调整Pod资源限制
5. **健康检查**: 根据应用特点调整探针配置

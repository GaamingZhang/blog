# Kubernetes部署文件模板

本文档包含所有需要在项目中创建的Kubernetes部署文件。

## 一、项目目录结构

在项目根目录创建以下目录结构：

```
/Users/gaamingzhang/git/gaamingzhangblog/
├── k8s/
│   ├── argocd/
│   │   ├── base/
│   │   │   ├── deployment.yaml
│   │   │   ├── service.yaml
│   │   │   ├── ingress.yaml
│   │   │   └── kustomization.yaml
│   │   ├── overlays/
│   │   │   ├── prod/
│   │   │   │   ├── kustomization.yaml
│   │   │   │   └── patches/
│   │   │   │       └── deployment-replicas.yaml
│   │   │   └── canary/
│   │   │       ├── kustomization.yaml
│   │   │       └── patches/
│   │   │           └── deployment-replicas.yaml
│   │   └── applications/
│   │       ├── cluster1-prod.yaml
│   │       ├── cluster1-canary.yaml
│   │       ├── cluster2-prod.yaml
│   │       └── cluster2-canary.yaml
│   ├── jenkins/
│   │   └── deploy-to-k8s.Jenkinsfile
│   └── scripts/
│       ├── render-yaml.sh
│       └── git-commit.sh
```

## 二、Base部署文件

### 2.1 deployment.yaml (k8s/argocd/base/deployment.yaml)

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

### 2.2 service.yaml (k8s/argocd/base/service.yaml)

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

### 2.3 ingress.yaml (k8s/argocd/base/ingress.yaml)

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

### 2.4 kustomization.yaml (k8s/argocd/base/kustomization.yaml)

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

## 三、Prod环境Overlay

### 3.1 kustomization.yaml (k8s/argocd/overlays/prod/kustomization.yaml)

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
  newName: 192.168.31.30:30003/gaamingzhang/blog
  newTag: latest
```

### 3.2 deployment-replicas.yaml (k8s/argocd/overlays/prod/patches/deployment-replicas.yaml)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gaamingzhang-blog
spec:
  replicas: 2
```

## 四、Canary环境Overlay

### 4.1 kustomization.yaml (k8s/argocd/overlays/canary/kustomization.yaml)

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
  newName: 192.168.31.30:30003/gaamingzhang/blog
  newTag: latest
```

### 4.2 deployment-replicas.yaml (k8s/argocd/overlays/canary/patches/deployment-replicas.yaml)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gaamingzhang-blog
spec:
  replicas: 2
```

## 五、ArgoCD Application配置

### 5.1 cluster1-prod.yaml

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: gaamingzhang-blog-prod-cluster1
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

### 5.2 cluster1-canary.yaml

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: gaamingzhang-blog-canary-cluster1
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

### 5.3 cluster2-prod.yaml

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

### 5.4 cluster2-canary.yaml

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

## 六、Jenkins Pipeline

### 6.1 deploy-to-k8s.Jenkinsfile (k8s/jenkins/deploy-to-k8s.Jenkinsfile)

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

## 七、部署脚本

### 7.1 render-yaml.sh (k8s/scripts/render-yaml.sh)

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

### 7.2 git-commit.sh (k8s/scripts/git-commit.sh)

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

## 八、使用说明

### 8.1 文件创建步骤

1. 在项目根目录创建上述目录结构
2. 将每个文件的内容复制到对应位置
3. 设置脚本执行权限：
   ```bash
   chmod +x k8s/scripts/*.sh
   ```

### 8.2 GitLab仓库设置

1. 在GitLab创建项目 `gaamingzhang-blog-k8s`
2. 上传所有k8s/argocd目录下的文件
3. 配置SSH密钥

### 8.3 ArgoCD配置

1. 应用ArgoCD Application配置：
   ```bash
   kubectl apply -f k8s/argocd/applications/
   ```

### 8.4 Jenkins配置

1. 创建新的Pipeline Job
2. 配置Git仓库（GitHub）
3. 添加所需凭据
4. 运行Pipeline

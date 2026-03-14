---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - CI/CD
  - DevOps
---

# CI/CD流程详解

## 什么是CI/CD？

CI/CD是持续集成（Continuous Integration）和持续交付/部署（Continuous Delivery/Deployment）的缩写。

```
┌─────────────────────────────────────────────────────────────┐
│                     CI/CD概念图                              │
│                                                              │
│  CI (持续集成)                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  代码提交 → 自动构建 → 自动测试 → 代码质量检查       │    │
│  └─────────────────────────────────────────────────────┘    │
│                            ↓                                 │
│  CD (持续交付)                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  自动部署到测试环境 → 手动审批 → 部署到生产环境       │    │
│  └─────────────────────────────────────────────────────┘    │
│                            ↓                                 │
│  CD (持续部署)                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  自动部署到测试环境 → 自动部署到生产环境              │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 完整CI/CD流程

### 流程图

```
┌─────────────────────────────────────────────────────────────┐
│                    完整CI/CD流水线                           │
│                                                              │
│  开发者                                                      │
│     │                                                        │
│     ↓ git push                                               │
│  ┌──────────────┐                                            │
│  │  代码仓库     │ GitLab / GitHub / Bitbucket               │
│  │  (Git)       │                                            │
│  └──────┬───────┘                                            │
│         │ webhook                                            │
│         ↓                                                    │
│  ┌──────────────┐                                            │
│  │  CI服务器     │ Jenkins / GitLab CI / GitHub Actions      │
│  │              │                                            │
│  │  1. 拉取代码  │                                            │
│  │  2. 编译构建  │                                            │
│  │  3. 单元测试  │                                            │
│  │  4. 代码扫描  │                                            │
│  │  5. 构建镜像  │                                            │
│  │  6. 推送镜像  │                                            │
│  └──────┬───────┘                                            │
│         │                                                    │
│         ↓                                                    │
│  ┌──────────────┐                                            │
│  │  镜像仓库     │ Harbor / Docker Registry / ECR            │
│  └──────┬───────┘                                            │
│         │                                                    │
│         ↓                                                    │
│  ┌──────────────┐                                            │
│  │  CD服务器     │ ArgoCD / Spinnaker / Flux                 │
│  │              │                                            │
│  │  1. 更新配置  │                                            │
│  │  2. 部署应用  │                                            │
│  │  3. 健康检查  │                                            │
│  └──────┬───────┘                                            │
│         │                                                    │
│         ↓                                                    │
│  ┌──────────────┐                                            │
│  │  Kubernetes   │                                            │
│  │  集群        │                                            │
│  └──────────────┘                                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## CI阶段详解

### 1. 代码提交

开发者将代码推送到代码仓库：

```bash
git add .
git commit -m "feat: add new feature"
git push origin feature/new-feature
```

### 2. 触发构建

通过Webhook或定时触发CI流水线：

```yaml
# GitLab CI示例
stages:
  - build
  - test
  - deploy

build:
  stage: build
  script:
    - mvn clean package
  artifacts:
    paths:
      - target/*.jar
```

### 3. 编译构建

```yaml
# Maven构建
build:
  stage: build
  image: maven:3.8-openjdk-17
  script:
    - mvn clean package -DskipTests
  artifacts:
    paths:
      - target/*.jar
    expire_in: 1 hour
```

### 4. 单元测试

```yaml
test:
  stage: test
  image: maven:3.8-openjdk-17
  script:
    - mvn test
  artifacts:
    reports:
      junit: target/surefire-reports/*.xml
```

### 5. 代码质量检查

```yaml
sonarqube:
  stage: test
  image: sonarsource/sonar-scanner-cli
  script:
    - sonar-scanner
      -Dsonar.projectKey=myapp
      -Dsonar.sources=src
      -Dsonar.host.url=$SONAR_URL
      -Dsonar.login=$SONAR_TOKEN
```

### 6. 构建镜像

```yaml
docker-build:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker login -u $REGISTRY_USER -p $REGISTRY_PASS $REGISTRY_URL
    - docker build -t $REGISTRY_URL/myapp:$CI_COMMIT_SHA .
    - docker push $REGISTRY_URL/myapp:$CI_COMMIT_SHA
```

## CD阶段详解

### 1. 部署到开发环境

```yaml
deploy-dev:
  stage: deploy
  environment:
    name: development
    url: https://dev.example.com
  script:
    - kubectl set image deployment/myapp myapp=$REGISTRY_URL/myapp:$CI_COMMIT_SHA -n dev
    - kubectl rollout status deployment/myapp -n dev
  only:
    - develop
```

### 2. 部署到测试环境

```yaml
deploy-test:
  stage: deploy
  environment:
    name: testing
    url: https://test.example.com
  script:
    - kubectl set image deployment/myapp myapp=$REGISTRY_URL/myapp:$CI_COMMIT_SHA -n test
    - kubectl rollout status deployment/myapp -n test
  only:
    - /^release\/.*$/
  when: manual
```

### 3. 部署到生产环境

```yaml
deploy-prod:
  stage: deploy
  environment:
    name: production
    url: https://www.example.com
  script:
    - kubectl set image deployment/myapp myapp=$REGISTRY_URL/myapp:$CI_COMMIT_SHA -n prod
    - kubectl rollout status deployment/myapp -n prod
  only:
    - main
  when: manual
```

## GitOps流程

### 什么是GitOps？

GitOps是一种以Git仓库为单一事实来源的运维模式。

```
┌─────────────────────────────────────────────────────────────┐
│                     GitOps流程                               │
│                                                              │
│  应用代码仓库                 配置仓库                        │
│  ┌─────────────┐            ┌─────────────┐                │
│  │  源代码     │            │  K8s配置    │                │
│  │  Dockerfile │            │  Helm Chart │                │
│  │  CI配置    │            │  Kustomize  │                │
│  └──────┬──────┘            └──────┬──────┘                │
│         │                          │                        │
│         ↓ CI                       ↓ GitOps Operator        │
│  ┌─────────────┐            ┌─────────────┐                │
│  │  构建镜像   │            │  监听变更   │                │
│  │  推送镜像   │            │  同步配置   │                │
│  └──────┬──────┘            └──────┬──────┘                │
│         │                          │                        │
│         │    更新镜像版本PR         │                        │
│         └────────────────────────→ │                        │
│                                  │                        │
│                                  ↓                        │
│                           ┌─────────────┐                 │
│                           │ Kubernetes  │                 │
│                           │   集群      │                 │
│                           └─────────────┘                 │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### ArgoCD配置示例

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: myapp
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://gitlab.com/myorg/myapp-config.git
    targetRevision: HEAD
    path: overlays/production
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

## Jenkins Pipeline示例

### Declarative Pipeline

```groovy
pipeline {
    agent {
        kubernetes {
            yaml '''
            apiVersion: v1
            kind: Pod
            spec:
              containers:
              - name: maven
                image: maven:3.8-openjdk-17
                command: ['cat']
                tty: true
              - name: docker
                image: docker:latest
                command: ['cat']
                tty: true
            '''
        }
    }
    
    environment {
        REGISTRY = 'harbor.example.com'
        IMAGE = 'myapp'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Build') {
            steps {
                container('maven') {
                    sh 'mvn clean package -DskipTests'
                }
            }
        }
        
        stage('Test') {
            steps {
                container('maven') {
                    sh 'mvn test'
                }
            }
        }
        
        stage('Docker Build') {
            steps {
                container('docker') {
                    sh """
                        docker build -t ${REGISTRY}/${IMAGE}:${BUILD_NUMBER} .
                        docker push ${REGISTRY}/${IMAGE}:${BUILD_NUMBER}
                    """
                }
            }
        }
        
        stage('Deploy') {
            steps {
                sh """
                    kubectl set image deployment/${IMAGE} ${IMAGE}=${REGISTRY}/${IMAGE}:${BUILD_NUMBER}
                """
            }
        }
    }
}
```

## GitHub Actions示例

```yaml
name: CI/CD Pipeline

on:
  push:
    branches: [main, develop]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Build
      run: mvn clean package
    
    - name: Test
      run: mvn test
    
    - name: Docker Build & Push
      run: |
        docker build -t myapp:${{ github.sha }} .
        docker push myapp:${{ github.sha }}
  
  deploy:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
    - name: Deploy to K8s
      run: |
        kubectl set image deployment/myapp myapp=myapp:${{ github.sha }}
```

## 最佳实践

### 1. 分支策略

```
main (生产)
  │
  ├── release/* (发布)
  │
  └── develop (开发)
        │
        └── feature/* (功能)
```

### 2. 环境隔离

| 环境 | 用途 | 部署方式 |
|------|------|----------|
| 开发 | 开发测试 | 自动部署 |
| 测试 | 集成测试 | 手动触发 |
| 生产 | 正式环境 | 手动审批 |

### 3. 回滚策略

```bash
kubectl rollout undo deployment/myapp
kubectl rollout undo deployment/myapp --to-revision=2
```

## 参考资源

- [Jenkins官方文档](https://www.jenkins.io/doc/)
- [GitLab CI/CD](https://docs.gitlab.com/ee/ci/)
- [ArgoCD](https://argo-cd.readthedocs.io/)

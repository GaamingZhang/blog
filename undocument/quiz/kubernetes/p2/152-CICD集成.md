---
date: 2026-03-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - CI/CD
  - 持续集成
  - 持续部署
---

# Kubernetes 与 CI/CD 集成

## 引言

CI/CD（持续集成/持续部署）是现代软件开发的核心实践。Kubernetes 与 CI/CD 工具的集成，可以实现应用的自动化构建、测试和部署，提高开发效率和交付质量。本文介绍 Kubernetes 与主流 CI/CD 工具的集成方案。

## CI/CD 概述

### CI/CD 流程

```
┌─────────────────────────────────────────────────────────────┐
│              CI/CD 流程                                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  代码提交                                                    │
│       │                                                      │
│       ▼                                                      │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              持续集成（CI）                          │   │
│  │  • 代码检查                                         │   │
│  │  • 单元测试                                         │   │
│  │  • 构建镜像                                         │   │
│  │  • 推送镜像仓库                                     │   │
│  └─────────────────────────────────────────────────────┘   │
│       │                                                      │
│       ▼                                                      │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              持续部署（CD）                          │   │
│  │  • 更新 Kubernetes 配置                             │   │
│  │  • 部署到测试环境                                   │   │
│  │  • 自动化测试                                       │   │
│  │  • 部署到生产环境                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Jenkins 集成

### Jenkins Pipeline

```groovy
pipeline {
    agent {
        kubernetes {
            yaml '''
            apiVersion: v1
            kind: Pod
            spec:
              containers:
              - name: docker
                image: docker:latest
                command: ['cat']
                tty: true
              - name: kubectl
                image: bitnami/kubectl:latest
                command: ['cat']
                tty: true
            '''
        }
    }
    
    environment {
        REGISTRY = 'my-registry.example.com'
        IMAGE = "${REGISTRY}/myapp:${BUILD_NUMBER}"
    }
    
    stages {
        stage('Build') {
            steps {
                container('docker') {
                    sh 'docker build -t ${IMAGE} .'
                    sh 'docker push ${IMAGE}'
                }
            }
        }
        
        stage('Deploy to Dev') {
            steps {
                container('kubectl') {
                    sh 'kubectl set image deployment/myapp myapp=${IMAGE} -n dev'
                }
            }
        }
        
        stage('Deploy to Production') {
            when {
                branch 'main'
            }
            steps {
                container('kubectl') {
                    input 'Deploy to production?'
                    sh 'kubectl set image deployment/myapp myapp=${IMAGE} -n production'
                }
            }
        }
    }
}
```

## GitLab CI 集成

### .gitlab-ci.yml

```yaml
stages:
  - build
  - test
  - deploy

variables:
  IMAGE: my-registry.example.com/myapp:$CI_COMMIT_SHA

build:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker login -u $REGISTRY_USER -p $REGISTRY_PASSWORD $REGISTRY
    - docker build -t $IMAGE .
    - docker push $IMAGE

test:
  stage: test
  image: $IMAGE
  script:
    - npm test

deploy_dev:
  stage: deploy
  image: bitnami/kubectl:latest
  script:
    - kubectl set image deployment/myapp myapp=$IMAGE -n dev
  environment:
    name: development
  only:
    - develop

deploy_production:
  stage: deploy
  image: bitnami/kubectl:latest
  script:
    - kubectl set image deployment/myapp myapp=$IMAGE -n production
  environment:
    name: production
  when: manual
  only:
    - main
```

## ArgoCD 集成

### ArgoCD Application

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: myapp
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/org/app-config.git
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

### CI 流程更新配置

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Update image tag
      run: |
        cd overlays/production
        kustomize edit set image myapp=my-registry.example.com/myapp:${{ github.sha }}
        
    - name: Commit and push
      run: |
        git config user.name "CI"
        git config user.email "ci@example.com"
        git add .
        git commit -m "Update image to ${{ github.sha }}"
        git push
```

## Flux 集成

### Flux 配置

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: GitRepository
metadata:
  name: app-config
  namespace: flux-system
spec:
  interval: 1m
  url: https://github.com/org/app-config.git
  ref:
    branch: main
---
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: myapp
  namespace: flux-system
spec:
  interval: 5m
  path: ./overlays/production
  sourceRef:
    kind: GitRepository
    name: app-config
  prune: true
  validation: client
```

## 最佳实践

### 1. 使用 GitOps

```yaml
syncPolicy:
  automated:
    prune: true
    selfHeal: true
```

### 2. 环境隔离

```yaml
environments:
  - name: development
  - name: staging
  - name: production
```

### 3. 自动化测试

```yaml
test:
  stage: test
  script:
    - npm test
    - npm run e2e
```

### 4. 回滚机制

```bash
kubectl rollout undo deployment/myapp

helm rollback myapp
```

## 面试回答

**问题**: Kubernetes 如何与 CI/CD 集成？

**回答**: Kubernetes 与 CI/CD 集成有多种方案：

**Jenkins 集成**：使用 Jenkins Kubernetes Plugin，在 Kubernetes 中运行动态 Agent。Pipeline 定义构建、测试、部署流程。使用 kubectl 或 Helm 部署应用到 Kubernetes。

**GitLab CI 集成**：使用 .gitlab-ci.yml 定义 CI/CD 流程。GitLab Runner 执行构建和部署任务。支持多环境部署，手动审批生产部署。

**ArgoCD 集成（GitOps）**：ArgoCD 监控 Git 仓库中的配置，自动同步到 Kubernetes。CI 流程更新 Git 仓库中的镜像版本，ArgoCD 自动部署。支持自动同步、自愈、回滚。

**Flux 集成（GitOps）**：Flux 监控 Git 仓库和镜像仓库。支持自动更新镜像版本并部署。与 Kustomize、Helm 深度集成。

**CI/CD 流程**：代码提交 -> CI 构建、测试 -> 推送镜像 -> 更新 Git 配置 -> GitOps 工具自动部署。

**最佳实践**：使用 GitOps 管理部署；环境隔离（dev/staging/production）；自动化测试；配置回滚机制；使用镜像标签管理版本；配置审批流程。

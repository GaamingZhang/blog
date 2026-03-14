---
date: 2026-03-12
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Namespace
  - 资源隔离
---

# Kubernetes Namespace 详解

## 引言

在 Kubernetes 集群中，随着应用数量和团队规模的增长，如何有效地组织和管理资源成为一个重要问题。Namespace 作为 Kubernetes 的核心概念之一，提供了一种在集群内部进行资源隔离和组织的机制。

通过 Namespace，可以将一个物理集群划分为多个虚拟集群，不同团队或项目可以在各自的 Namespace 中独立工作，互不干扰。理解 Namespace 的作用、使用场景和最佳实践，是构建可扩展、可维护的 Kubernetes 集群的基础。

## Namespace 概述

### 什么是 Namespace

Namespace 是 Kubernetes 中用于实现多租户资源隔离的机制。它将集群划分为多个虚拟集群，每个 Namespace 内的资源名称必须唯一，但不同 Namespace 中的资源可以重名。

### Namespace 的核心作用

1. **资源隔离**：不同 Namespace 的资源相互隔离
2. **权限控制**：通过 RBAC 限制对 Namespace 的访问
3. **资源配额**：为每个 Namespace 设置资源使用上限
4. **网络策略**：控制不同 Namespace 之间的网络访问
5. **命名作用域**：同一 Namespace 内资源名称必须唯一

### 默认 Namespace

Kubernetes 集群创建后会自动创建以下 Namespace：

| Namespace | 说明 |
|-----------|------|
| **default** | 默认 Namespace，未指定 Namespace 的资源会被创建在此 |
| **kube-system** | Kubernetes 系统组件运行的 Namespace |
| **kube-public** | 公开可读的 Namespace，用于存储集群信息 |
| **kube-node-lease** | 用于节点心跳数据的 Namespace |

## Namespace 基本操作

### 创建 Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: development
  labels:
    name: development
    environment: dev
  annotations:
    description: "Development environment namespace"
```

```bash
# 命令行创建
kubectl create namespace development

# 从 YAML 文件创建
kubectl apply -f namespace.yaml
```

### 查看 Namespace

```bash
# 列出所有 Namespace
kubectl get namespaces

# 查看详细信息
kubectl describe namespace development

# 查看 Namespace 资源使用
kubectl get resourcequota -n development
```

### 删除 Namespace

```bash
# 删除 Namespace（会删除其中所有资源）
kubectl delete namespace development
```

## Namespace 资源隔离

### 资源类型分类

#### Namespace 作用域资源

这些资源只能在 Namespace 内创建：

| 资源类型 | 说明 |
|---------|------|
| Pod | 容器组 |
| Service | 服务 |
| Deployment | 部署 |
| ConfigMap | 配置映射 |
| Secret | 密钥 |
| PVC | 持久卷声明 |
| Job | 任务 |
| CronJob | 定时任务 |
| Ingress | 入口 |

#### 集群作用域资源

这些资源不属于任何 Namespace：

| 资源类型 | 说明 |
|---------|------|
| Node | 节点 |
| PV | 持久卷 |
| ClusterRole | 集群角色 |
| ClusterRoleBinding | 集群角色绑定 |
| Namespace | 命名空间本身 |
| StorageClass | 存储类 |
| CustomResourceDefinition | 自定义资源定义 |

### 查看资源作用域

```bash
# 查看 Namespace 作用域资源
kubectl api-resources --namespaced=true

# 查看集群作用域资源
kubectl api-resources --namespaced=false
```

## Namespace 资源配额

### ResourceQuota 配置

ResourceQuota 用于限制 Namespace 的资源使用总量：

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-quota
  namespace: development
spec:
  hard:
    requests.cpu: "10"
    requests.memory: "20Gi"
    limits.cpu: "20"
    limits.memory: "40Gi"
    pods: "50"
    services: "10"
    secrets: "20"
    configmaps: "20"
    persistentvolumeclaims: "10"
    replicationcontrollers: "20"
    resourcequotas: "1"
---
apiVersion: v1
kind: ResourceQuota
metadata:
  name: object-count-quota
  namespace: development
spec:
  hard:
    count/deployments.apps: "10"
    count/statefulsets.apps: "5"
    count/jobs.batch: "20"
    count/cronjobs.batch: "10"
```

### LimitRange 配置

LimitRange 用于限制单个 Pod 或容器的资源：

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: resource-limits
  namespace: development
spec:
  limits:
  - type: Container
    default:
      cpu: "500m"
      memory: "512Mi"
    defaultRequest:
      cpu: "100m"
      memory: "128Mi"
    min:
      cpu: "50m"
      memory: "64Mi"
    max:
      cpu: "2"
      memory: "4Gi"
    maxLimitRequestRatio:
      cpu: "10"
      memory: "4"
  - type: PersistentVolumeClaim
    max:
      storage: "50Gi"
    min:
      storage: "1Gi"
  - type: Pod
    max:
      cpu: "4"
      memory: "8Gi"
```

### 查看配额使用情况

```bash
# 查看配额
kubectl get resourcequota -n development

# 查看详细信息
kubectl describe resourcequota compute-quota -n development
```

## Namespace 权限控制

### RBAC 配置

```yaml
# 命名空间级别的 Role
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: developer
  namespace: development
rules:
- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets", "statefulsets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["pods/portforward", "pods/exec", "pods/log"]
  verbs: ["get", "create"]
---
# RoleBinding 绑定用户到 Role
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: developer-binding
  namespace: development
subjects:
- kind: User
  name: alice
  apiGroup: rbac.authorization.k8s.io
- kind: Group
  name: developers
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: developer
  apiGroup: rbac.authorization.k8s.io
---
# 只读用户
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: viewer
  namespace: development
rules:
- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets", "pods/log"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets", "statefulsets"]
  verbs: ["get", "list", "watch"]
```

### 限制 Namespace 访问

```yaml
# 使用 ClusterRole 限制只能访问特定 Namespace
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: namespace-viewer
rules:
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: restrict-to-development
  namespace: development
subjects:
- kind: User
  name: bob
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: admin
  apiGroup: rbac.authorization.k8s.io
```

## Namespace 网络隔离

### NetworkPolicy 配置

```yaml
# 禁止 Namespace 之间的网络访问
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-from-other-namespaces
  namespace: development
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector: {}
---
# 允许特定 Namespace 访问
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-production
  namespace: development
spec:
  podSelector:
    matchLabels:
      app: api
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: production
    ports:
    - protocol: TCP
      port: 8080
```

### Namespace 标签配置

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: development
  labels:
    name: development
    environment: dev
    team: backend
---
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    name: production
    environment: prod
    team: backend
```

## Namespace 典型使用场景

### 场景一：多环境隔离

```yaml
# 开发环境
apiVersion: v1
kind: Namespace
metadata:
  name: dev
  labels:
    environment: development
---
# 测试环境
apiVersion: v1
kind: Namespace
metadata:
  name: staging
  labels:
    environment: staging
---
# 生产环境
apiVersion: v1
kind: Namespace
metadata:
  name: prod
  labels:
    environment: production
```

### 场景二：多团队隔离

```yaml
# 后端团队
apiVersion: v1
kind: Namespace
metadata:
  name: team-backend
  labels:
    team: backend
---
# 前端团队
apiVersion: v1
kind: Namespace
metadata:
  name: team-frontend
  labels:
    team: frontend
---
# 数据团队
apiVersion: v1
kind: Namespace
metadata:
  name: team-data
  labels:
    team: data
```

### 场景三：多租户隔离

```yaml
# 租户 A
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-a
  labels:
    tenant: tenant-a
---
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tenant-a-quota
  namespace: tenant-a
spec:
  hard:
    requests.cpu: "4"
    requests.memory: "8Gi"
    pods: "20"
---
# 租户 B
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-b
  labels:
    tenant: tenant-b
---
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tenant-b-quota
  namespace: tenant-b
spec:
  hard:
    requests.cpu: "2"
    requests.memory: "4Gi"
    pods: "10"
```

## 跨 Namespace 访问

### Service 跨 Namespace 访问

```yaml
# 在 development Namespace 访问 production 的服务
apiVersion: v1
kind: Service
metadata:
  name: api-service
  namespace: production
spec:
  selector:
    app: api
  ports:
  - port: 80
    targetPort: 8080
```

```bash
# 在 development Namespace 中访问
# 格式: <service-name>.<namespace>.svc.cluster.local
curl http://api-service.production.svc.cluster.local
```

### 短名称访问

```bash
# 在同一 Namespace 内可以使用短名称
curl http://api-service

# 跨 Namespace 需要使用完整域名
curl http://api-service.production.svc.cluster.local
```

### ExternalName Service 跨 Namespace

```yaml
# 在 development Namespace 创建指向 production 的 Service
apiVersion: v1
kind: Service
metadata:
  name: prod-api
  namespace: development
spec:
  type: ExternalName
  externalName: api-service.production.svc.cluster.local
```

## Namespace 最佳实践

### 1. 命名规范

```yaml
# 推荐的命名格式
# <environment>-<team>-<application>
# 例如: dev-backend-api, prod-frontend-web

apiVersion: v1
kind: Namespace
metadata:
  name: dev-backend-api
  labels:
    environment: dev
    team: backend
    application: api
```

### 2. 标签和注解

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: my-namespace
  labels:
    name: my-namespace
    environment: production
    team: platform
    cost-center: engineering
  annotations:
    description: "Production namespace for platform team"
    owner: "platform-team@example.com"
    created-by: "admin"
```

### 3. 默认配置

```yaml
# 为 Namespace 设置默认配置
apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
  namespace: my-namespace
spec:
  limits:
  - type: Container
    default:
      cpu: "200m"
      memory: "256Mi"
    defaultRequest:
      cpu: "100m"
      memory: "128Mi"
```

### 4. 资源配额模板

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: default-quota
  namespace: my-namespace
spec:
  hard:
    requests.cpu: "4"
    requests.memory: "8Gi"
    limits.cpu: "8"
    limits.memory: "16Gi"
    pods: "50"
    services: "10"
    secrets: "20"
    configmaps: "20"
    persistentvolumeclaims: "10"
```

### 5. 网络隔离模板

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-network-policy
  namespace: my-namespace
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector: {}
  egress:
  - to:
    - podSelector: {}
  - to:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: kube-system
    podSelector:
      matchLabels:
        k8s-app: kube-dns
    ports:
    - protocol: UDP
      port: 53
```

## Namespace 常见问题

### Q1: 如何限制用户只能访问特定 Namespace？

通过 RBAC 配置 Role 和 RoleBinding，而不是 ClusterRole 和 ClusterRoleBinding：

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: user-binding
  namespace: development
subjects:
- kind: User
  name: alice
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: admin
  apiGroup: rbac.authorization.k8s.io
```

### Q2: 如何在删除 Namespace 时保留某些资源？

使用 finalizers 或在删除前手动导出资源：

```bash
# 导出资源
kubectl get all -n development -o yaml > backup.yaml

# 删除特定资源而不是整个 Namespace
kubectl delete deployment,service -n development --all
```

### Q3: 如何查看 Namespace 的资源使用情况？

```bash
# 查看资源配额使用
kubectl describe resourcequota -n development

# 查看资源使用
kubectl top pods -n development
kubectl top nodes

# 查看事件
kubectl get events -n development --sort-by='.lastTimestamp'
```

### Q4: Namespace 删除卡住怎么办？

```bash
# 查看 Namespace 状态
kubectl describe namespace development

# 检查是否有 finalizer 阻止删除
kubectl get namespace development -o json | jq '.spec.finalizers'

# 强制删除（谨慎使用）
kubectl delete namespace development --force --grace-period=0
```

## 面试回答

**问题**: Kubernetes 中 Namespace 的作用是什么？

**回答**: Namespace 是 Kubernetes 中实现多租户资源隔离的核心机制。它的主要作用包括：**资源隔离**，将一个物理集群划分为多个虚拟集群，不同 Namespace 的资源相互独立，资源名称在 Namespace 内必须唯一；**权限控制**，通过 RBAC 的 Role 和 RoleBinding 可以限制用户只能访问特定 Namespace 的资源，实现细粒度的权限管理；**资源配额**，通过 ResourceQuota 限制 Namespace 的资源使用总量，防止单个团队或项目占用过多资源，通过 LimitRange 限制单个 Pod 或容器的资源范围；**网络隔离**，通过 NetworkPolicy 可以控制不同 Namespace 之间的网络访问，实现安全隔离；**命名作用域**，同一 Namespace 内可以使用 Service 短名称访问，跨 Namespace 需要使用完整域名。

典型的使用场景包括：多环境隔离（dev、staging、prod）、多团队隔离（backend、frontend、data）、多租户隔离（tenant-a、tenant-b）。Kubernetes 默认创建四个 Namespace：default 是默认命名空间，kube-system 存放系统组件，kube-public 存放公开信息，kube-node-lease 用于节点心跳。生产环境建议为每个团队或环境创建独立的 Namespace，并配置资源配额和网络策略，实现安全和资源的有效管理。

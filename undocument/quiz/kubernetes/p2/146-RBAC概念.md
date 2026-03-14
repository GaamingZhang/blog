---
date: 2026-03-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - RBAC
  - 权限控制
  - 安全
---

# Kubernetes RBAC 概念

## 引言

RBAC（Role-Based Access Control，基于角色的访问控制）是 Kubernetes 授权的核心机制。通过 RBAC，可以精细控制用户和服务账户对 Kubernetes 资源的访问权限，实现最小权限原则，保障集群安全。

## RBAC 核心概念

### RBAC 组件

```
┌─────────────────────────────────────────────────────────────┐
│                  RBAC 核心组件                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Subject（主体）：                                           │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • User：外部用户（由外部系统管理）                 │   │
│  │  • Group：用户组                                    │   │
│  │  • ServiceAccount：Pod 使用的服务账户               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Role（角色）：                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • Role：命名空间级别权限                           │   │
│  │  • ClusterRole：集群级别权限                        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  RoleBinding（角色绑定）：                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • RoleBinding：绑定 Subject 到 Role                │   │
│  │  • ClusterRoleBinding：绑定 Subject 到 ClusterRole  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### RBAC 关系图

```
┌─────────────────────────────────────────────────────────────┐
│                  RBAC 关系图                                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐                                           │
│  │    User     │                                           │
│  │  (dev-user) │                                           │
│  └──────┬───────┘                                           │
│         │                                                    │
│         ▼                                                    │
│  ┌──────────────┐     ┌──────────────┐                     │
│  │ RoleBinding  │────▶│     Role     │                     │
│  │  (dev-bind)  │     │  (dev-role)  │                     │
│  └──────────────┘     └──────┬───────┘                     │
│                              │                              │
│                              ▼                              │
│                       ┌──────────────┐                     │
│                       │ Permissions  │                     │
│                       │  • get pods  │                     │
│                       │  • list pods │                     │
│                       └──────────────┘                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Role 和 ClusterRole

### Role（命名空间级别）

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pod-reader
  namespace: production
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list"]
```

### ClusterRole（集群级别）

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: node-reader
rules:
- apiGroups: [""]
  resources: ["nodes", "nodes/status"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
```

## RoleBinding 和 ClusterRoleBinding

### RoleBinding

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: read-pods
  namespace: production
subjects:
- kind: User
  name: dev-user
  apiGroup: rbac.authorization.k8s.io
- kind: ServiceAccount
  name: dev-sa
  namespace: production
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

### ClusterRoleBinding

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: read-nodes
subjects:
- kind: User
  name: admin-user
  apiGroup: rbac.authorization.k8s.io
- kind: Group
  name: system:admins
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: node-reader
  apiGroup: rbac.authorization.k8s.io
```

## ServiceAccount

### 创建 ServiceAccount

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: app-sa
  namespace: production
automountServiceAccountToken: false
```

### 在 Pod 中使用

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app-pod
  namespace: production
spec:
  serviceAccountName: app-sa
  automountServiceAccountToken: false
  containers:
  - name: app
    image: myapp:v1
```

## 权限规则详解

### apiGroups

```yaml
rules:
- apiGroups: [""]              # 核心 API 组
  resources: ["pods", "services"]
- apiGroups: ["apps"]          # apps API 组
  resources: ["deployments"]
- apiGroups: ["batch"]         # batch API 组
  resources: ["jobs"]
- apiGroups: ["*"]             # 所有 API 组
  resources: ["*"]
```

### resources

```yaml
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log", "pods/exec"]
- apiGroups: [""]
  resources: ["secrets"]
  resourceNames: ["my-secret"]
```

### verbs

```yaml
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch", "create", "update", "delete"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["*"]
```

## 内置 ClusterRole

### 常用内置角色

| ClusterRole | 说明 |
|-------------|------|
| cluster-admin | 集群管理员，所有权限 |
| admin | 命名空间管理员 |
| edit | 编辑权限 |
| view | 只读权限 |

### 使用内置角色

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: dev-view
  namespace: production
subjects:
- kind: User
  name: dev-user
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: view
  apiGroup: rbac.authorization.k8s.io
```

## RBAC 调试

### 检查权限

```bash
kubectl auth can-i get pods --as=dev-user

kubectl auth can-i get pods --as=dev-user -n production

kubectl auth can-i list secrets --as=system:anonymous
```

### 查看角色绑定

```bash
kubectl get rolebindings -n production

kubectl get clusterrolebindings

kubectl describe rolebinding read-pods -n production
```

## 最佳实践

### 1. 最小权限原则

```yaml
rules:
- apiGroups: [""]
  resources: ["pods"]
  resourceNames: ["specific-pod"]
  verbs: ["get"]
```

### 2. 使用命名空间隔离

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: namespace-admin
  namespace: production
```

### 3. 禁用自动挂载 Token

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: app-sa
automountServiceAccountToken: false
```

## 面试回答

**问题**: Kubernetes 中 RBAC 是什么？

**回答**: RBAC（Role-Based Access Control，基于角色的访问控制）是 Kubernetes 授权的核心机制：

**核心组件**：**Subject（主体）**包括 User（用户）、Group（用户组）、ServiceAccount（服务账户）；**Role（角色）**定义权限规则，Role 是命名空间级别，ClusterRole 是集群级别；**RoleBinding（角色绑定）**将 Subject 绑定到 Role，RoleBinding 绑定命名空间角色，ClusterRoleBinding 绑定集群角色。

**权限规则**：**apiGroups** 指定 API 组，如 ""（核心组）、"apps"、"batch"；**resources** 指定资源类型，如 pods、services、secrets；**verbs** 指定操作，如 get、list、watch、create、update、delete；**resourceNames** 可限制特定资源。

**ServiceAccount**：为 Pod 提供身份标识，Pod 通过 ServiceAccount 访问 Kubernetes API。建议禁用自动挂载 Token，仅在需要时启用。

**内置角色**：cluster-admin（集群管理员）、admin（命名空间管理员）、edit（编辑权限）、view（只读权限）。

**最佳实践**：遵循最小权限原则；使用命名空间隔离；禁用 ServiceAccount 自动挂载 Token；使用 Group 管理用户权限；定期审计权限配置。

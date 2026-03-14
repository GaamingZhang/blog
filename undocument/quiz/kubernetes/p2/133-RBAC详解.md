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
  - 安全
  - 访问控制
---

# Kubernetes RBAC 详解

## 引言

RBAC（Role-Based Access Control，基于角色的访问控制）是 Kubernetes 授权的核心机制。通过 RBAC，可以精细控制用户和服务账户对 Kubernetes 资源的访问权限，实现最小权限原则，保障集群安全。

## RBAC 概述

### RBAC 核心概念

```
┌─────────────────────────────────────────────────────────────┐
│                  RBAC 核心概念                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Subject（主体）：                                           │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • User：外部用户                                   │   │
│  │  • Group：用户组                                    │   │
│  │  • ServiceAccount：服务账户                         │   │
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
│  关系：Subject -> RoleBinding -> Role -> Permissions        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### RBAC 架构

```
┌─────────────────────────────────────────────────────────────┐
│                  RBAC 架构                                   │
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

## Role 和 RoleBinding

### 创建 Role

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

### 创建 RoleBinding

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

## ClusterRole 和 ClusterRoleBinding

### 创建 ClusterRole

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

### 创建 ClusterRoleBinding

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

### 绑定 ServiceAccount 到 Role

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: app-role-binding
  namespace: production
subjects:
- kind: ServiceAccount
  name: app-sa
  namespace: production
roleRef:
  kind: Role
  name: app-role
  apiGroup: rbac.authorization.k8s.io
```

### 在 Pod 中使用 ServiceAccount

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
  resources: ["deployments", "replicasets"]
- apiGroups: ["batch"]         # batch API 组
  resources: ["jobs", "cronjobs"]
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

## 常用 ClusterRole 示例

### 只读权限

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: view-all
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
```

### 管理权限

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: admin
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["apps"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["batch"]
  resources: ["*"]
  verbs: ["*"]
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

### 查看 Role 详情

```bash
kubectl get roles -n production

kubectl describe role pod-reader -n production
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

### 4. 使用 Group 管理用户

```yaml
subjects:
- kind: Group
  name: developers
  apiGroup: rbac.authorization.k8s.io
```

## 面试回答

**问题**: 如何进行安全访问控制（RBAC）？

**回答**: Kubernetes RBAC（基于角色的访问控制）通过以下机制实现安全访问控制：

**核心概念**：**Subject（主体）**包括 User（用户）、Group（用户组）、ServiceAccount（服务账户）；**Role（角色）**定义权限规则，Role 是命名空间级别，ClusterRole 是集群级别；**RoleBinding（角色绑定）**将 Subject 绑定到 Role，RoleBinding 绑定到命名空间角色，ClusterRoleBinding 绑定到集群角色。

**权限规则**：**apiGroups** 指定 API 组，如 ""（核心组）、"apps"、"batch"；**resources** 指定资源类型，如 pods、services、secrets；**verbs** 指定操作，如 get、list、watch、create、update、delete；**resourceNames** 可限制特定资源。

**配置示例**：创建 Role 定义权限规则，创建 RoleBinding 将用户或 ServiceAccount 绑定到 Role。命名空间级别使用 Role + RoleBinding，集群级别使用 ClusterRole + ClusterRoleBinding。

**ServiceAccount**：为 Pod 提供身份标识，Pod 通过 ServiceAccount 访问 Kubernetes API。建议禁用自动挂载 Token，仅在需要时启用。

**最佳实践**：遵循最小权限原则，只授予必要的权限；使用命名空间隔离不同环境；使用 Group 管理用户权限；禁用 ServiceAccount 自动挂载 Token；定期审计权限配置。

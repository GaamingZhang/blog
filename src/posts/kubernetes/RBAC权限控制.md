---
date: 2026-01-30
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# RBAC权限控制

## 为什么需要权限控制？

想象你是一家公司的IT管理员。公司里有开发人员、测试人员、运维人员、实习生等不同角色。你肯定不希望实习生能够删除生产环境的数据库，对吧？

Kubernetes的RBAC（Role-Based Access Control，基于角色的访问控制）就是解决这个问题的。它让你可以精确控制：
- **谁**（用户或程序）
- 可以对**什么资源**（Pod、Service、Secret等）
- 执行**什么操作**（查看、创建、删除等）

没有RBAC，所有人都是"超级管理员"，这在生产环境中是非常危险的。

## 核心概念：四个关键组件

RBAC的核心可以用一句话概括：**把权限赋予角色，再把角色赋予用户**。

| 组件 | 作用 | 比喻 |
|------|------|------|
| **Role/ClusterRole** | 定义一组权限 | 职位说明书（描述这个职位能做什么） |
| **RoleBinding/ClusterRoleBinding** | 把角色赋予用户 | 任命书（把某人任命为某职位） |
| **User/Group** | 外部用户或用户组 | 公司员工 |
| **ServiceAccount** | Pod使用的身份 | 机器人员工（程序使用的身份） |

**Role vs ClusterRole的区别**：
- **Role**：只在一个namespace内有效（"部门经理只管自己部门"）
- **ClusterRole**：在整个集群有效（"CEO可以管所有部门"）

## 权限是如何工作的？

当你用kubectl执行命令时，背后发生的事情是：

1. kubectl发送请求到API Server
2. API Server问："这个请求是谁发的？"（认证）
3. API Server再问："这个人有权限做这件事吗？"（授权，就是RBAC）
4. 如果有权限，执行操作；否则返回"Forbidden"

## 理解权限的三要素

定义权限时，你需要指定三样东西：

```yaml
rules:
  - apiGroups: [""]        # 1. API组（资源属于哪个API组）
    resources: ["pods"]     # 2. 资源类型
    verbs: ["get", "list"]  # 3. 允许的操作
```

**API组**常见的有：
- `""`（空字符串）= 核心API组，包含Pod、Service、ConfigMap等
- `"apps"` = Deployment、StatefulSet等
- `"rbac.authorization.k8s.io"` = RBAC相关资源

**常用动词（verbs）**：

| 动词 | 含义 |
|------|------|
| get | 获取单个资源 |
| list | 列出资源 |
| watch | 监听资源变化 |
| create | 创建资源 |
| update | 更新资源 |
| delete | 删除资源 |

## 实例：创建一个只读角色

假设你想创建一个"开发人员"角色，只能查看Pod，不能修改或删除：

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pod-reader
  namespace: development
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]  # 只有读取权限
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]  # 还允许查看Pod日志
```

然后把这个角色赋予某个用户：

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: read-pods
  namespace: development
subjects:
  - kind: User
    name: jane  # 用户名
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: pod-reader  # 引用上面创建的Role
  apiGroup: rbac.authorization.k8s.io
```

现在，用户jane在development命名空间里只能查看Pod，不能做其他操作。

## ServiceAccount：程序的身份

User是给人用的，但Pod里运行的程序也需要身份。这就是ServiceAccount的作用。

比如，你的监控程序需要读取所有Pod的信息。你需要：

1. 创建一个ServiceAccount
2. 创建一个ClusterRole（因为要读取所有namespace的Pod）
3. 把ClusterRole绑定到ServiceAccount

```yaml
# 1. 创建ServiceAccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: monitoring
  namespace: monitoring

---
# 2. 创建ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pod-reader
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]

---
# 3. 绑定
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: monitoring-pods
subjects:
  - kind: ServiceAccount
    name: monitoring
    namespace: monitoring
roleRef:
  kind: ClusterRole
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

然后在Pod中使用这个ServiceAccount：

```yaml
spec:
  serviceAccountName: monitoring  # 使用monitoring这个身份
```

## 预定义的ClusterRole

Kubernetes提供了一些内置的ClusterRole，不用自己从零开始：

| ClusterRole | 权限 |
|-------------|------|
| `cluster-admin` | 超级管理员，可以做任何事 |
| `admin` | namespace管理员，可以管理大部分资源 |
| `edit` | 可以编辑资源，但不能修改权限 |
| `view` | 只读权限 |

比如，想让某人成为development命名空间的管理员：

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: dev-admin
  namespace: development
subjects:
  - kind: User
    name: alice
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole  # 注意：用的是ClusterRole
  name: admin        # 但通过RoleBinding限制在一个namespace
  apiGroup: rbac.authorization.k8s.io
```

这个技巧很有用：**用ClusterRole定义权限，用RoleBinding限制范围**。

## 常见问题

### Q1: 遇到"Forbidden"错误怎么办？

这说明你没有执行这个操作的权限。

**排查步骤**：

1. 先确认你是谁：
```bash
kubectl auth whoami  # K8s 1.27+
```

2. 检查你是否有这个权限：
```bash
kubectl auth can-i create pods
kubectl auth can-i delete pods -n production
```

3. 如果你是管理员，可以检查某个ServiceAccount的权限：
```bash
kubectl auth can-i create pods --as=system:serviceaccount:default:myapp
```

**原理解释**：RBAC是"白名单"机制，没有明确授权的操作默认都是禁止的。

### Q2: Role和ClusterRole该用哪个？

**判断标准**：

| 场景 | 选择 |
|------|------|
| 只需要访问一个namespace的资源 | Role + RoleBinding |
| 需要访问集群级资源（Node、PV等） | ClusterRole + ClusterRoleBinding |
| 需要在多个namespace有相同权限 | ClusterRole + 多个RoleBinding |

**原理解释**：有些资源（如Node、PersistentVolume）不属于任何namespace，只能用ClusterRole访问。

### Q3: 最小权限原则怎么落地？

**实践建议**：

1. **只授予必需的权限**：不要因为图省事就给cluster-admin
2. **限定具体资源**：可以的话用`resourceNames`限制到具体的资源名
3. **优先用Role而非ClusterRole**：范围越小越安全
4. **为每个应用创建独立的ServiceAccount**：不要共用default

```yaml
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    resourceNames: ["app-config"]  # 只能操作这一个ConfigMap
    verbs: ["get", "update"]
```

### Q4: 如何撤销权限？

删除对应的RoleBinding或ClusterRoleBinding即可：

```bash
kubectl delete rolebinding read-pods -n development
```

权限会立即失效，不需要重启任何东西。

## 小结

| 概念 | 解释 |
|------|------|
| RBAC | 基于角色的访问控制，控制谁能做什么 |
| Role | namespace级别的权限定义 |
| ClusterRole | 集群级别的权限定义 |
| RoleBinding | 把Role赋予用户/ServiceAccount |
| ClusterRoleBinding | 把ClusterRole赋予用户/ServiceAccount |
| ServiceAccount | 给Pod使用的身份 |

记住核心思想：**权限给角色，角色给人（或程序）**。这样当一个人离职时，只需要删除他的RoleBinding，而不用修改Role本身。

## 参考资源

- [Kubernetes RBAC 官方文档](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [使用 RBAC 鉴权](https://kubernetes.io/zh/docs/reference/access-authn-authz/rbac/)
- [预定义角色参考](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#default-roles-and-role-bindings)

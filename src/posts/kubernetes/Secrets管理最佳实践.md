---
date: 2026-01-30
author: Gaaming Zhang
isOriginal: true
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# Secrets管理最佳实践

## 为什么Secret管理这么重要？

密码、API密钥、数据库凭证......这些敏感信息就像你家的钥匙。你不会把钥匙放在门垫下面贴个标签写"钥匙在这"吧？但很多团队在Kubernetes中管理Secret时，做的事情本质上就是这样的。

Secret泄露的后果非常严重：
- 数据库被删库
- 云账户被盗刷
- 用户数据泄露

所以，安全地管理Secret是每个Kubernetes管理员必须掌握的技能。

## Kubernetes Secret的真相

首先要理解一个重要的事实：**Kubernetes原生的Secret不是真正加密的**。

```bash
# 看起来是加密的
kubectl get secret my-secret -o yaml
# data:
#   password: cGFzc3dvcmQxMjM=

# 实际上只是Base64编码，任何人都能解码
echo "cGFzc3dvcmQxMjM=" | base64 -d
# 输出: password123
```

Base64只是一种编码方式，不是加密。就像把中文翻译成英文，懂英文的人都能看懂。

**默认情况下，Secret还有这些问题**：

| 问题 | 解释 |
|------|------|
| etcd明文存储 | Secret存在etcd数据库里，默认是明文的 |
| 权限控制不细 | 有namespace读取权限的人通常能读所有Secret |
| 容易误提交Git | YAML文件可能被不小心提交到代码仓库 |

## 基础安全措施

### 1. 启用etcd加密

这是最基本的，让Secret在etcd中以加密形式存储：

```yaml
# /etc/kubernetes/encryption-config.yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
      - secrets
    providers:
      - aescbc:
          keys:
            - name: key1
              secret: <32字节的密钥，base64编码>
      - identity: {}
```

这样即使有人直接访问etcd数据库，也看不到明文的Secret。

### 2. 用RBAC限制谁能读Secret

不要让所有人都能读取Secret：

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: secret-reader
  namespace: production
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["app-credentials"]  # 只能读这一个Secret
    verbs: ["get"]
```

**关键点**：用 `resourceNames` 限制到具体的Secret名称，而不是允许读取所有Secret。

## Secret应该怎么传给应用？

有两种方式把Secret传给Pod：

### 方式一：环境变量

```yaml
env:
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: db-credentials
        key: password
```

### 方式二：文件挂载

```yaml
volumeMounts:
  - name: secrets
    mountPath: /etc/secrets
    readOnly: true
volumes:
  - name: secrets
    secret:
      secretName: db-credentials
```

**推荐使用文件挂载**。为什么？

| 方式 | 优点 | 缺点 |
|------|------|------|
| 环境变量 | 使用方便 | 可能在日志、错误堆栈中泄露 |
| 文件挂载 | 更安全，支持热更新 | 应用需要读文件 |

环境变量容易在各种地方泄露：程序崩溃时的core dump、debug日志、进程列表......所以敏感信息尽量用文件方式。

## 进阶方案：使用外部密钥管理

对于生产环境，最佳实践是不在Kubernetes里存储真正的Secret，而是从外部密钥管理系统获取。

### 为什么要用外部系统？

| 优势 | 解释 |
|------|------|
| 集中管理 | 所有敏感信息在一个地方管理 |
| 审计日志 | 谁在什么时候访问了哪个密钥 |
| 自动轮换 | 自动定期更换密钥 |
| 权限控制 | 更细粒度的访问控制 |

### 常用方案

**External Secrets Operator** - 从外部系统同步Secret到Kubernetes：

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: database-credentials
spec:
  refreshInterval: 1h  # 每小时同步一次
  secretStoreRef:
    name: aws-secrets  # 使用AWS Secrets Manager
    kind: SecretStore
  target:
    name: db-credentials  # 在K8s中创建的Secret名称
  data:
    - secretKey: password
      remoteRef:
        key: prod/database
        property: password
```

好处是：
1. 真正的密钥存在AWS/HashiCorp Vault等专业系统中
2. Kubernetes中的Secret会自动保持同步
3. 密钥轮换时应用无需修改

**Sealed Secrets** - 让加密后的Secret可以安全提交到Git：

```bash
# 加密Secret
kubeseal --format yaml < secret.yaml > sealed-secret.yaml

# 加密后的文件可以安全提交到Git
# 只有集群内的controller能解密
```

## Secret轮换

密钥应该定期更换。问题是：更换后应用怎么知道？

**方案一：使用Reloader**

```yaml
metadata:
  annotations:
    reloader.stakater.com/auto: "true"
```

当Secret变化时，Reloader会自动重启Pod。

**方案二：应用监听文件变化**

如果用文件挂载Secret，Kubernetes会自动更新文件内容。应用可以用inotify监听文件变化，无需重启。

## 常见问题

### Q1: 如何防止Secret被误提交到Git？

**多层防护**：

1. 在 `.gitignore` 中排除Secret文件：
```
*-secret.yaml
*.key
.env
```

2. 使用Sealed Secrets，提交的是加密后的版本

3. 在CI/CD中加检查：如果检测到Secret关键词就失败

**原理解释**：Secret一旦泄露到Git历史中，即使删除也能被找回。所以要在提交之前就拦截。

### Q2: 环境变量和文件挂载怎么选？

**简单判断**：
- 不那么敏感的配置（如端口号）→ 环境变量
- 敏感信息（密码、密钥）→ 文件挂载

**深层原因**：环境变量会出现在 `/proc/<pid>/environ`，在进程列表中可见，在崩溃日志中可能被打印。文件的访问权限更容易控制。

### Q3: Secret更新后应用没有感知到？

这是正常的。Kubernetes不会自动重启Pod来应用新的Secret。

**解决方案**：
1. 使用Reloader等工具自动重启
2. 应用自己监听配置文件变化（如果用文件挂载）
3. 手动滚动更新：`kubectl rollout restart deployment myapp`

### Q4: 多个namespace需要用同一个Secret怎么办？

Kubernetes的Secret是namespace级别的，不能跨namespace共享。

**解决方案**：
1. 用External Secrets Operator，在每个namespace创建ExternalSecret指向同一个外部密钥
2. 用工具（如kubed）自动同步Secret到多个namespace

## 安全检查清单

在生产环境部署前，对照检查：

| 检查项 | 状态 |
|--------|------|
| etcd加密是否启用？ | |
| RBAC是否限制了Secret访问？ | |
| 敏感信息是否用文件挂载而非环境变量？ | |
| Git仓库是否排除了Secret文件？ | |
| 是否使用外部密钥管理系统？ | |
| Secret轮换机制是否就绪？ | |

## 小结

| 级别 | 措施 |
|------|------|
| 基础 | 启用etcd加密、RBAC限制访问 |
| 进阶 | 使用文件挂载、防止Git泄露 |
| 最佳实践 | 使用外部密钥管理系统（Vault、AWS Secrets Manager等） |

记住核心原则：
1. **Secret不是真正加密的** - 需要额外措施保护
2. **最小权限** - 只给真正需要的人/程序访问权限
3. **不要信任Git** - 即使是私有仓库也不要提交明文Secret
4. **外部管理更安全** - 专业的密钥管理系统比Kubernetes自己管理更靠谱

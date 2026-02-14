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

# Pod安全标准

## 为什么需要Pod安全限制？

想象你开了一家公司，给员工发工作电脑。你肯定会对电脑做一些限制：不能安装未经审批的软件，不能关闭杀毒软件，不能用管理员账户日常工作。为什么？因为如果某台电脑被攻破，这些限制能防止攻击者造成更大的破坏。

Pod安全标准的思路完全一样。容器看起来像独立的小机器，但其实它和宿主机共享同一个操作系统内核。如果容器能以root权限运行、能访问宿主机的文件系统，那一旦容器里的应用被攻破，整个节点甚至整个集群都可能沦陷。

所以，Pod安全标准就是：**给容器设置"安全围栏"，限制它能做的事情**。

## 三个安全级别

Kubernetes定义了三个安全级别，从宽松到严格：

| 级别 | 描述 | 适用场景 |
|------|------|----------|
| **Privileged（特权）** | 不受限制，可以做任何事 | 系统组件、需要特殊权限的工具 |
| **Baseline（基线）** | 防止常见的高危配置 | 大多数普通应用 |
| **Restricted（严格）** | 遵循最严格的安全实践 | 安全敏感的应用 |

**如何选择？**

- 如果你是初学者或不确定，从**Baseline**开始
- 如果你的应用处理敏感数据，升级到**Restricted**
- 只有系统组件才需要**Privileged**

## 这些限制到底限制了什么？

### Baseline级别禁止的事情

这些都是"明显危险"的配置：

| 禁止的配置 | 为什么危险 |
|-----------|-----------|
| 特权容器（privileged: true） | 特权容器几乎等于直接在宿主机上运行 |
| 使用宿主机网络（hostNetwork: true） | 可以监听宿主机的所有网络流量 |
| 使用宿主机PID命名空间 | 可以看到和操作宿主机的所有进程 |
| 挂载宿主机目录（hostPath） | 可以读写宿主机的任意文件 |

### Restricted级别额外要求的

在Baseline基础上，还要求：

| 要求 | 为什么重要 |
|------|-----------|
| 必须以非root用户运行 | root用户权限太大，即使在容器里 |
| 禁止特权提升 | 防止容器内的进程获取更多权限 |
| 必须设置Seccomp | 限制容器能调用的系统调用 |
| 只能drop capabilities，不能add | 最小权限原则 |

## 如何启用Pod安全标准？

Pod安全标准是在**namespace级别**通过标签启用的：

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    # 强制执行baseline级别
    pod-security.kubernetes.io/enforce: baseline
    # 对于restricted级别，只警告不拒绝（用于迁移过渡）
    pod-security.kubernetes.io/warn: restricted
```

**三种执行模式**：

| 模式 | 行为 |
|------|------|
| **enforce** | 违反策略的Pod会被拒绝创建 |
| **warn** | 允许创建，但会显示警告 |
| **audit** | 允许创建，但会记录到审计日志 |

**迁移建议**：先用warn模式观察，确认没问题再改成enforce。

## 让你的Pod符合安全标准

下面是一个符合Restricted级别的Pod配置：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: secure-app
spec:
  securityContext:
    runAsNonRoot: true  # 不允许以root运行
    seccompProfile:
      type: RuntimeDefault  # 使用默认的seccomp配置
  containers:
    - name: app
      image: myapp:1.0
      securityContext:
        allowPrivilegeEscalation: false  # 禁止特权提升
        runAsNonRoot: true
        runAsUser: 1000  # 指定运行用户ID
        capabilities:
          drop:
            - ALL  # 删除所有capabilities
        readOnlyRootFilesystem: true  # 根文件系统只读
      # 应用需要写入的目录用emptyDir
      volumeMounts:
        - name: tmp
          mountPath: /tmp
  volumes:
    - name: tmp
      emptyDir: {}
```

**关键点解释**：

1. **runAsNonRoot: true** - 强制使用非root用户。很多官方镜像默认是root，需要注意
2. **allowPrivilegeEscalation: false** - 防止进程通过setuid等方式获取更多权限
3. **drop: ALL** - capabilities是Linux的细粒度权限机制，删除全部是最安全的
4. **readOnlyRootFilesystem: true** - 防止恶意代码修改系统文件

## 常见问题

### Q1: Pod创建失败，提示违反安全策略？

这说明你的Pod配置不符合namespace的安全级别。

**排查步骤**：

1. 查看具体违反了什么：创建Pod时的错误信息会告诉你
2. 常见违规原因：
   - 没设置 `runAsNonRoot: true`
   - 使用了 `privileged: true`
   - 挂载了 `hostPath` 卷

**解决办法**：根据错误信息调整Pod配置，或者（如果确实需要）降低namespace的安全级别。

### Q2: 应用确实需要root权限怎么办？

首先问自己：**真的需要吗？**

很多时候应用"需要"root只是因为：
- 要绑定80端口 → 改用1024以上的端口，用Ingress做端口映射
- 要写某个系统目录 → 挂载emptyDir到那个路径

如果真的需要root（比如某些网络工具），有两个选择：
1. 把应用放到使用Baseline级别的namespace
2. 只授予必需的capability，而不是完全的root权限

### Q3: readOnlyRootFilesystem导致应用无法运行？

很多应用需要写入临时文件。解决办法是用emptyDir挂载可写目录：

```yaml
volumeMounts:
  - name: tmp
    mountPath: /tmp
  - name: cache
    mountPath: /var/cache
volumes:
  - name: tmp
    emptyDir: {}
  - name: cache
    emptyDir: {}
```

这样根文件系统保持只读，但应用可以写入指定的目录。

### Q4: 如何给系统组件（如kube-system）设置Privileged？

```bash
kubectl label namespace kube-system \
  pod-security.kubernetes.io/enforce=privileged
```

系统组件确实需要特权，这是正常的。关键是普通应用不要也跟着用Privileged。

## 安全级别对照表

| 配置项 | Privileged | Baseline | Restricted |
|--------|------------|----------|------------|
| 特权容器 | 允许 | 禁止 | 禁止 |
| hostNetwork/hostPID | 允许 | 禁止 | 禁止 |
| hostPath卷 | 允许 | 禁止 | 禁止 |
| runAsNonRoot | 不要求 | 不要求 | 必须true |
| allowPrivilegeEscalation | 不要求 | 不要求 | 必须false |
| Seccomp | 不要求 | 不要求 | 必须设置 |
| Capabilities | 不限制 | 限制添加 | 必须drop ALL |

## 小结

| 概念 | 解释 |
|------|------|
| Pod安全标准 | Kubernetes定义的容器安全配置规范 |
| Privileged | 无限制，用于系统组件 |
| Baseline | 防止常见危险配置，适合大多数应用 |
| Restricted | 最严格的安全实践 |
| SecurityContext | 在Pod/Container级别设置安全配置 |

记住核心原则：**最小权限**。容器应该只拥有运行所必需的权限，多一点都不应该给。这样即使应用被攻破，攻击者能做的事情也非常有限。

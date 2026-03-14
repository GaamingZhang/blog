---
date: 2026-03-12
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - kubeadm
  - 集群管理
---

# kubeadm 初始化的 k8s 集群，token 过期后，集群中加入新节点怎么办？

## 引言

在使用 kubeadm 初始化 Kubernetes 集群时，系统会生成一个 bootstrap token，用于新节点加入集群时的身份认证。然而，这个 token 默认有效期仅为 24 小时，过期后将无法使用。当需要在 token 过期后向集群添加新节点时，许多运维人员会感到困惑：是否需要重新初始化集群？如何安全地生成新的 token？本文将深入解析 kubeadm token 的工作机制，并提供完整的解决方案。

## Token 的作用和有效期

### Token 的核心作用

Bootstrap token 在 Kubernetes 集群中扮演着至关重要的角色，其主要职责包括：

1. **身份认证**：新节点使用 token 向 API Server 进行身份验证，获取加入集群的临时凭证
2. **双向 TLS 认证**：token 关联的密钥用于节点与控制平面之间的 TLS 引导（TLS Bootstrap）
3. **RBAC 授权**：token 对应的 ServiceAccount 拥有特定的权限，允许节点完成初始化配置

### Token 的有效期机制

kubeadm 生成的 token 具有以下特性：

| 属性 | 默认值 | 说明 |
|------|--------|------|
| 有效期 | 24 小时 | 从创建时刻开始计算 |
| 格式 | `[a-z0-9]{6}.[a-z0-9]{16}` | 前缀.后缀格式 |
| 存储位置 | Secret | 存储在 kube-system 命名空间 |
| 用途 | 节点加入 | 用于 kubeadm join 命令 |

Token 过期后，对应的 Secret 会被自动删除，无法继续用于新节点加入。但已加入的节点不受影响，因为它们已经完成了 TLS 证书的签发和配置。

## 查看现有 Token

在处理 token 过期问题前，首先需要了解如何查看集群中现有的 token 状态。

### 列出所有 Token

```bash
kubeadm token list
```

输出示例：

```
TOKEN                     TTL         EXPIRES                USAGES                   DESCRIPTION                                                EXTRA GROUPS
abcdef.0123456789abcdef   23h         2026-03-13T10:00:00Z   authentication,signing   <none>                                                     system:bootstrappers:kubeadm:default-node-token
```

### 输出字段解析

| 字段 | 含义 |
|------|------|
| TOKEN | token 字符串，格式为前缀.后缀 |
| TTL | 剩余有效时间 |
| EXPIRES | 过期时间戳 |
| USAGES | token 的用途类型 |
| EXTRA GROUPS | token 关联的用户组 |

如果命令输出为空或没有有效的 token，说明需要生成新的 token。

## 生成新 Token

当 token 过期后，可以在控制平面节点上轻松生成新的 token。

### 生成默认 Token

```bash
kubeadm token create
```

输出示例：

```
[output] abcdef.0123456789abcdef
```

### 生成指定有效期的 Token

```bash
# 生成有效期为 10 小时的 token
kubeadm token create --ttl 10h

# 生成永不过期的 token（不推荐生产环境使用）
kubeadm token create --ttl 0
```

### 生成指定描述的 Token

```bash
kubeadm token create --description "for worker node expansion"
```

### Token 生成参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| --ttl | 24h | token 有效期，设为 0 表示永不过期 |
| --description | 无 | token 的描述信息 |
| --groups | system:bootstrappers:kubeadm:default-node-token | token 关联的用户组 |
| --usages | authentication,signing | token 的用途 |

## 获取加入命令

生成新 token 后，需要构建完整的节点加入命令。kubeadm 提供了便捷的方式来生成加入命令。

### 方法一：自动生成完整命令

```bash
kubeadm token create --print-join-command
```

输出示例：

```bash
kubeadm join 192.168.1.100:6443 --token abcdef.0123456789abcdef \
    --discovery-token-ca-cert-hash sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
```

这个命令会自动生成包含 token 和 CA 证书 hash 的完整加入命令，直接复制到新节点执行即可。

### 方法二：手动构建加入命令

如果需要更灵活的控制，可以手动构建加入命令：

**步骤 1：生成 token**

```bash
kubeadm token create
```

**步骤 2：获取 CA 证书 hash**

```bash
openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outform der 2>/dev/null | \
   openssl dgst -sha256 -hex | sed 's/^.* //'
```

输出示例：

```
1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
```

**步骤 3：构建加入命令**

```bash
kubeadm join <CONTROL_PLANE_IP>:6443 \
    --token <TOKEN> \
    --discovery-token-ca-cert-hash sha256:<HASH>
```

### discovery-token-ca-cert-hash 的作用

`--discovery-token-ca-cert-hash` 参数用于验证控制平面的身份，防止中间人攻击。新节点在加入集群时，会使用这个 hash 值验证 API Server 的 CA 证书，确保连接的是正确的集群。

## 重新生成证书密钥

在某些情况下，除了 token 过期外，可能还需要重新生成 certificate-key（证书密钥），特别是在控制平面节点加入场景中。

### 什么是 Certificate Key

Certificate key 是用于加密和解密 kubeadm 集群证书的密钥，主要用于：

1. **控制平面节点加入**：新的 master 节点使用此密钥从集群下载证书
2. **证书分发**：确保证书在传输过程中的安全性

### 上传证书并生成新的 Certificate Key

```bash
kubeadm init phase upload-certs --upload-certs
```

输出示例：

```
[upload-certs] Storing the certificates in Secret "kubeadm-certs" in the "kube-system" Namespace
[upload-certs] Using certificate key:
1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
```

### 生成控制平面加入命令

```bash
kubeadm token create --print-join-command
```

然后手动添加 `--control-plane` 和 `--certificate-key` 参数：

```bash
kubeadm join 192.168.1.100:6443 \
    --token abcdef.0123456789abcdef \
    --discovery-token-ca-cert-hash sha256:1234567890abcdef... \
    --control-plane \
    --certificate-key 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
```

## 完整操作流程

### 场景一：Worker 节点加入

**在控制平面节点执行：**

```bash
# 1. 生成新的 token 和加入命令
kubeadm token create --print-join-command

# 输出示例：
# kubeadm join 192.168.1.100:6443 --token abcdef.0123456789abcdef \
#     --discovery-token-ca-cert-hash sha256:1234567890abcdef...
```

**在新的 Worker 节点执行：**

```bash
# 2. 执行加入命令
kubeadm join 192.168.1.100:6443 \
    --token abcdef.0123456789abcdef \
    --discovery-token-ca-cert-hash sha256:1234567890abcdef...

# 3. 验证节点状态
kubectl get nodes
```

### 场景二：控制平面节点加入

**在现有控制平面节点执行：**

```bash
# 1. 上传证书并生成 certificate-key
kubeadm init phase upload-certs --upload-certs

# 2. 生成 token
kubeadm token create --print-join-command

# 3. 构建完整的控制平面加入命令
# 将步骤 1 和步骤 2 的输出组合
```

**在新的控制平面节点执行：**

```bash
# 4. 执行加入命令
kubeadm join 192.168.1.100:6443 \
    --token abcdef.0123456789abcdef \
    --discovery-token-ca-cert-hash sha256:1234567890abcdef... \
    --control-plane \
    --certificate-key 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef

# 5. 配置 kubectl
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# 6. 验证集群状态
kubectl get nodes
kubectl get cs
```

## Token 管理最佳实践

### 1. 定期轮换 Token

```bash
# 创建脚本定期生成 token
#!/bin/bash
# 每周生成新的 token
kubeadm token create --ttl 168h --description "weekly token"
```

### 2. 使用配置文件管理 Token

创建 token 配置文件：

```yaml
# token-config.yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: BootstrapToken
token: abcdef.0123456789abcdef
ttl: 72h
description: "production cluster token"
usages:
  - authentication
  - signing
groups:
  - system:bootstrappers:kubeadm:default-node-token
```

应用配置：

```bash
kubeadm token create --config token-config.yaml
```

### 3. 监控 Token 状态

创建监控脚本：

```bash
#!/bin/bash
# 检查 token 是否即将过期
TOKEN_INFO=$(kubeadm token list | grep -v TOKEN | head -1)
if [ -z "$TOKEN_INFO" ]; then
    echo "Warning: No valid token found!"
    # 自动创建新 token
    kubeadm token create --description "auto-generated token"
fi
```

### 4. 安全建议

| 建议 | 说明 |
|------|------|
| 避免使用永不过期的 token | 增加安全风险，容易被滥用 |
| 使用强描述信息 | 便于审计和追踪 token 用途 |
| 定期清理过期 token | kubeadm 会自动清理，但建议定期检查 |
| 限制 token 权限 | 根据实际需求配置最小权限 |
| 记录 token 使用日志 | 便于安全审计 |

### 5. 高可用集群的 Token 管理

在高可用集群中，建议：

```bash
# 在负载均衡器后的任意控制平面节点生成 token
# 所有控制平面节点共享相同的 CA 证书，因此生成的 token 通用

# 使用负载均衡器地址
kubeadm join lb.k8s.local:6443 \
    --token abcdef.0123456789abcdef \
    --discovery-token-ca-cert-hash sha256:1234567890abcdef...
```

## 常见问题和解决方案

### 问题 1：执行 kubeadm token list 报错

**错误信息：**

```
failed to list bootstrap tokens: configmaps "cluster-info" is forbidden
```

**原因分析：**

当前用户没有访问 cluster-info ConfigMap 的权限。

**解决方案：**

```bash
# 使用 root 用户或配置正确的 kubeconfig
export KUBECONFIG=/etc/kubernetes/admin.conf

# 或者使用 sudo
sudo kubeadm token list
```

### 问题 2：新节点加入失败 - 证书验证错误

**错误信息：**

```
[discovery] Failed to request cluster-info, will try again: Get "https://192.168.1.100:6443/api/v1/namespaces/kube-public/configmaps/cluster-info": x509: certificate has expired or is not yet valid
```

**原因分析：**

1. 控制平面证书已过期
2. 新节点时间与控制平面时间不同步

**解决方案：**

```bash
# 检查证书有效期
kubeadm certs check-expiration

# 更新证书（如果已过期）
kubeadm certs renew all

# 确保所有节点时间同步
timedatectl status
ntpdate -u pool.ntp.org
```

### 问题 3：无法获取 CA 证书 Hash

**问题场景：**

执行 openssl 命令获取 hash 时失败。

**解决方案：**

```bash
# 方法 1：使用 kubeadm 命令
kubeadm token create --print-join-command

# 方法 2：从集群中获取
kubectl -n kube-public get configmap cluster-info -o jsonpath='{.data.kubeconfig}' | \
  grep 'certificate-authority-data' | \
  awk '{print $2}' | \
  base64 -d | \
  openssl x509 -pubkey -inform der | \
  openssl rsa -pubin -outform der 2>/dev/null | \
  openssl dgst -sha256 -hex | \
  sed 's/^.* //'
```

### 问题 4：Token 创建成功但节点无法加入

**错误信息：**

```
error execution phase preflight: [preflight] Some fatal errors occurred:
    [ERROR FileAvailable--etc-kubernetes-kubelet.conf]: /etc/kubernetes/kubelet.conf already exists
```

**原因分析：**

节点上存在旧的配置文件，可能是之前加入失败的残留。

**解决方案：**

```bash
# 重置节点配置
kubeadm reset -f

# 清理残留文件
rm -rf /etc/kubernetes/*
rm -rf /var/lib/kubelet/*
rm -rf /var/lib/etcd

# 重新加入集群
kubeadm join ...
```

### 问题 5：多控制平面节点证书不一致

**问题场景：**

新控制平面节点加入后，证书与现有节点不一致。

**解决方案：**

```bash
# 确保使用正确的 certificate-key
# 在现有控制平面节点重新上传证书
kubeadm init phase upload-certs --upload-certs

# 使用新生成的 certificate-key 加入
```

## Token 工作原理深入解析

### Bootstrap Token 认证流程

```
┌─────────────┐                 ┌──────────────┐                 ┌─────────────┐
│  新节点     │                 │  API Server  │                 │  控制平面   │
│             │                 │              │                 │             │
│ 1. 发起请求 │                 │              │                 │             │
│ (携带token) │────────────────>│              │                 │             │
│             │                 │ 2. 验证token │                 │             │
│             │                 │   查询Secret │                 │             │
│             │                 │              │                 │             │
│             │                 │ 3. 授权检查  │                 │             │
│             │                 │   RBAC规则   │                 │             │
│             │                 │              │                 │             │
│             │<────────────────│ 4. 返回临时  │                 │             │
│             │                 │   凭证       │                 │             │
│             │                 │              │                 │             │
│ 5. TLS引导  │                 │              │                 │             │
│   生成CSR   │────────────────>│              │                 │             │
│             │                 │ 6. 签发证书  │                 │             │
│             │<────────────────│              │                 │             │
│             │                 │              │                 │             │
│ 7. 使用证书 │                 │              │                 │             │
│   加入集群  │────────────────>│ 8. 确认加入  │                 │             │
│             │                 │              │────────────────>│             │
│             │                 │              │                 │ 9. 同步状态 │
└─────────────┘                 └──────────────┘                 └─────────────┘
```

### Token 的 Secret 结构

Bootstrap token 以 Secret 形式存储在 kube-system 命名空间：

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: bootstrap-token-abcdef
  namespace: kube-system
type: bootstrap.kubernetes.io/token
stringData:
  token-id: abcdef
  token-secret: 0123456789abcdef
  usage-bootstrap-authentication: "true"
  usage-bootstrap-signing: "true"
  auth-extra-groups: system:bootstrappers:kubeadm:default-node-token
  expiration: "2026-03-13T10:00:00Z"
  description: "kubeadm generated token"
```

### RBAC 权限绑定

kubeadm 自动创建以下 RBAC 资源，允许 bootstrap token 进行 TLS 引导：

```yaml
# ClusterRoleBinding 允许 bootstrap token 创建 CSR
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubeadm:kubelet-bootstrap
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:node-bootstrapper
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: system:bootstrappers:kubeadm:default-node-token
```

## 面试回答

在面试中遇到这个问题时，可以这样回答：

"kubeadm 初始化的 Kubernetes 集群，bootstrap token 默认有效期为 24 小时，过期后不影响已加入节点的正常运行，但新节点无法使用旧 token 加入集群。解决方案非常简单：在控制平面节点执行 `kubeadm token create --print-join-command` 即可生成新的 token 和完整的加入命令。这个命令会自动生成 token 并计算 CA 证书的 hash 值，直接复制到新节点执行即可完成加入。对于控制平面节点的加入，还需要额外执行 `kubeadm init phase upload-certs --upload-certs` 生成 certificate-key，用于证书的安全分发。实际生产中，建议建立 token 定期轮换机制，避免使用永不过期的 token，并做好 token 使用的审计日志。理解 token 的工作原理很重要，它本质上是存储在 kube-system 命名空间的 Secret，通过 RBAC 授权机制，允许新节点完成 TLS Bootstrap 流程，最终获得集群颁发的客户端证书，实现安全的节点认证。"

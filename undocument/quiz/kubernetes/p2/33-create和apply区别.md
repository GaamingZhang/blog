---
date: 2026-03-12
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - kubectl
---

# kubectl create 与 apply 深度解析：命令式与声明式管理的本质区别

## 引言

在日常的Kubernetes集群管理中，`kubectl create`和`kubectl apply`是最常用的两个资源创建命令。表面上看，它们都能创建Kubernetes资源，但在实际生产环境中，错误地使用这两个命令可能导致严重的运维事故。你是否遇到过这样的困惑：为什么使用`kubectl apply`更新资源时，某些字段被意外删除？为什么`kubectl create`不能重复执行？为什么生产环境强烈推荐使用`kubectl apply`？

深入理解这两个命令的区别，不仅关乎正确的操作姿势，更涉及到Kubernetes资源管理的核心理念——命令式（Imperative）与声明式（Declarative）管理的本质差异。本文将从底层实现原理出发，全面剖析这两个命令的工作机制、适用场景以及最佳实践，帮助你建立正确的Kubernetes资源管理思维。

## 一、kubectl create：命令式资源管理

### 1.1 工作原理

`kubectl create`是典型的命令式（Imperative）管理方式，它告诉Kubernetes"做什么"——即创建一个新资源。其核心工作流程如下：

```
用户 → kubectl create → API Server → 验证 → 直接创建资源 → etcd
```

**具体执行步骤**：

1. **解析配置文件**：kubectl读取YAML或JSON配置文件
2. **构造HTTP请求**：向API Server发送POST请求
3. **服务端验证**：API Server进行认证、授权、准入控制
4. **资源创建**：如果资源不存在，直接创建；如果已存在，返回错误
5. **持久化存储**：将资源对象写入etcd

### 1.2 关键特性

**一次性创建**：`kubectl create`是幂等性操作的反面，如果资源已存在，操作会失败：

```bash
# 第一次执行：成功创建
kubectl create -f nginx-deployment.yaml
# deployment.apps/nginx-deployment created

# 第二次执行：报错
kubectl create -f nginx-deployment.yaml
# Error from server (AlreadyExists): error when creating "nginx-deployment.yaml": deployments.apps "nginx-deployment" already exists
```

**全量提交**：`kubectl create`将配置文件中的所有字段作为资源的完整定义提交给API Server。这意味着后续如果使用`kubectl apply`更新资源，可能会遇到字段管理冲突的问题。

**无状态管理**：`kubectl create`不会记录资源的"期望状态"，它只是执行一次性的创建操作。创建完成后，kubectl不会保留任何关于如何管理该资源的信息。

### 1.3 实现机制

当执行`kubectl create -f deployment.yaml`时，kubectl会：

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
```

kubectl将这个YAML转换为JSON，构造如下HTTP请求：

```http
POST /apis/apps/v1/namespaces/default/deployments HTTP/1.1
Host: api-server:6443
Authorization: Bearer <token>
Content-Type: application/json

{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "nginx-deployment",
    "namespace": "default"
  },
  "spec": {
    "replicas": 3,
    ...
  }
}
```

API Server接收到请求后，会进行以下处理：

1. **对象验证**：检查Deployment结构是否符合API规范
2. **默认值填充**：为未指定的字段填充默认值（如strategy、revisionHistoryLimit等）
3. **准入控制**：执行Mutating和Validating准入控制器
4. **版本分配**：分配初始的ResourceVersion
5. **写入etcd**：持久化资源对象

## 二、kubectl apply：声明式资源管理

### 2.1 工作原理

`kubectl apply`是声明式（Declarative）管理方式，它告诉Kubernetes"期望的状态是什么"——即确保资源达到配置文件中定义的期望状态。其核心工作流程如下：

```
用户 → kubectl apply → 计算差异 → 合并策略 → API Server → 更新资源 → etcd
```

**具体执行步骤**：

1. **读取配置文件**：kubectl读取YAML或JSON配置文件
2. **获取当前状态**：从API Server查询资源的当前状态
3. **获取上次应用状态**：从annotation中读取`kubectl.kubernetes.io/last-applied-configuration`
4. **计算三路合并**：比较上次应用状态、当前状态和期望状态
5. **构造PATCH请求**：生成差异化的更新内容
6. **提交更新**：向API Server发送PATCH请求
7. **更新annotation**：将新的期望状态记录到last-applied-configuration

### 2.2 关键特性

**声明式管理**：`kubectl apply`关注的是"期望状态"，而不是具体的操作动作。无论资源是否存在，apply都能正确处理：

```bash
# 资源不存在：创建资源
kubectl apply -f nginx-deployment.yaml
# deployment.apps/nginx-deployment created

# 资源已存在：更新资源
kubectl apply -f nginx-deployment.yaml
# deployment.apps/nginx-deployment configured

# 资源已是期望状态：无变化
kubectl apply -f nginx-deployment.yaml
# deployment.apps/nginx-deployment unchanged
```

**增量更新**：`kubectl apply`只更新配置文件中指定的字段，保留其他字段的当前值。这是通过三路合并（3-way merge）机制实现的。

**状态管理**：`kubectl apply`会在资源的annotation中记录上次应用的配置，用于后续的合并计算：

```yaml
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"name":"nginx-deployment","namespace":"default"},"spec":{"replicas":3,"selector":{"matchLabels":{"app":"nginx"}},"template":{"metadata":{"labels":{"app":"nginx"}},"spec":{"containers":[{"image":"nginx:1.14.2","name":"nginx","ports":[{"containerPort":80}]}]}}}}
```

### 2.3 三路合并（3-way Merge）机制

三路合并是`kubectl apply`的核心算法，它通过比较三个版本来计算最终的更新内容：

**三个版本**：
- **原始版本（Original）**：上次apply时的配置，存储在`last-applied-configuration` annotation中
- **当前版本（Current）**：资源在集群中的实际状态
- **期望版本（Modified）**：本次apply提交的配置文件内容

**合并规则**：

| 场景 | 原始版本 | 当前版本 | 期望版本 | 合并结果 | 说明 |
|------|----------|----------|----------|----------|------|
| 新增字段 | 无 | 无 | 有 | 添加字段 | 用户明确添加了新字段 |
| 修改字段 | 值A | 值A | 值B | 值B | 用户修改了字段，且未被其他方修改 |
| 删除字段 | 值A | 值A | 无 | 删除字段 | 用户明确删除了字段 |
| 冲突修改 | 值A | 值B | 值C | 值C | 用户修改了字段，但当前值已被其他方修改，apply优先 |
| 外部修改 | 无 | 值B | 无 | 值B | 字段不在配置文件中，保留外部修改 |
| 外部新增 | 无 | 值B | 无 | 值B | 字段由外部系统添加，保留该字段 |

**示例说明**：

假设有一个Deployment，初始配置如下：

```yaml
# version1.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
```

执行`kubectl apply -f version1.yaml`后，annotation中记录了该配置。

现在，用户手动修改了replicas为5（通过`kubectl edit`或`kubectl scale`），集群中的实际状态变为：

```yaml
spec:
  replicas: 5  # 外部修改
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
```

用户更新配置文件，修改镜像版本：

```yaml
# version2.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.20.0  # 修改镜像版本
```

执行`kubectl apply -f version2.yaml`时，三路合并计算：

- **replicas字段**：原始=3，当前=5，期望=3 → 结果=3（用户明确指定，覆盖外部修改）
- **image字段**：原始=1.14.2，当前=1.14.2，期望=1.20.0 → 结果=1.20.0（用户修改）

最终资源状态：

```yaml
spec:
  replicas: 3  # 被apply还原
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.20.0  # 更新为新版本
```

### 2.4 资源版本管理

`kubectl apply`通过`last-applied-configuration` annotation实现资源版本管理。这个annotation存储了上次apply时的完整配置，是三路合并的关键。

**annotation的作用**：

1. **记录期望状态**：保存用户上次提交的配置，作为合并计算的基准
2. **字段所有权管理**：标识哪些字段由kubectl apply管理，哪些由其他方式管理
3. **冲突检测**：检测配置文件与集群状态的差异

**查看last-applied-configuration**：

```bash
kubectl get deployment nginx-deployment -o yaml | grep -A 20 "last-applied-configuration"
```

**手动管理annotation**：

```bash
# 查看资源的apply历史
kubectl apply -f deployment.yaml --dry-run=client -o yaml

# 强制覆盖last-applied-configuration
kubectl apply -f deployment.yaml --server-side --force-conflicts
```

### 2.5 Server-Side Apply

从Kubernetes 1.16开始，引入了Server-Side Apply（服务端应用），将合并逻辑从客户端（kubectl）移到了服务端（API Server）。

**Server-Side Apply的优势**：

1. **并发安全**：多个用户可以同时管理同一资源的不同字段
2. **字段管理**：精确跟踪每个字段的所有者
3. **冲突解决**：提供更智能的冲突检测和解决机制

**使用Server-Side Apply**：

```bash
kubectl apply -f deployment.yaml --server-side
```

**字段管理（Field Management）**：

Server-Side Apply引入了`managedFields`，记录每个字段的管理者：

```yaml
metadata:
  managedFields:
  - manager: kubectl
    operation: Apply
    apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:replicas: {}
        f:template:
          f:spec:
            f:containers:
              k:{"name":"nginx"}:
                f:image: {}
  - manager: kube-controller-manager
    operation: Update
    apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:status:
        f:availableReplicas: {}
```

这表明：
- `spec.replicas`和`spec.template.spec.containers[*].image`由kubectl管理
- `status.availableReplicas`由kube-controller-manager管理

## 三、命令式 vs 声明式管理

### 3.1 理念对比

| 维度 | 命令式（Imperative） | 声明式（Declarative） |
|------|---------------------|---------------------|
| **关注点** | 如何做（How） | 做什么（What） |
| **操作方式** | 执行具体命令 | 定义期望状态 |
| **状态管理** | 无状态记录 | 记录期望状态 |
| **可重复性** | 不幂等 | 幂等操作 |
| **版本控制** | 难以追踪 | 天然支持GitOps |
| **适用场景** | 快速测试、临时操作 | 生产环境、持续部署 |
| **学习曲线** | 简单直观 | 需要理解状态管理 |

### 3.2 命令式管理示例

**命令式命令**：

```bash
# 创建资源
kubectl create -f deployment.yaml

# 更新资源
kubectl set image deployment/nginx-deployment nginx=nginx:1.20.0

# 扩容
kubectl scale deployment nginx-deployment --replicas=5

# 删除资源
kubectl delete deployment nginx-deployment
```

**特点**：
- 每个命令对应一个具体的操作
- 需要记住各种命令和参数
- 操作历史难以追踪
- 不适合版本控制

### 3.3 声明式管理示例

**声明式配置**：

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 5
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.20.0
```

**应用配置**：

```bash
# 应用配置（创建或更新）
kubectl apply -f deployment.yaml

# 删除资源
kubectl delete -f deployment.yaml
```

**特点**：
- 配置文件描述期望状态
- 可以提交到Git仓库进行版本控制
- 支持GitOps工作流
- 易于审计和回滚

### 3.4 GitOps工作流

声明式管理天然支持GitOps，实现基础设施即代码（IaC）：

```
Git Repository (配置文件)
       ↓
   GitOps Tool (ArgoCD/Flux)
       ↓
   Kubernetes Cluster
```

**工作流程**：

1. 开发者提交配置文件到Git仓库
2. GitOps工具检测到变更
3. 自动执行`kubectl apply`同步到集群
4. 集群状态与Git仓库保持一致

## 四、create与apply对比分析

### 4.1 核心差异对比

| 对比维度 | kubectl create | kubectl apply |
|---------|---------------|---------------|
| **管理方式** | 命令式（Imperative） | 声明式（Declarative） |
| **资源存在时** | 报错退出 | 更新资源 |
| **资源不存在时** | 创建资源 | 创建资源 |
| **更新方式** | 不支持更新 | 增量更新（PATCH） |
| **状态记录** | 无 | last-applied-configuration |
| **合并策略** | 无 | 三路合并（3-way merge） |
| **字段管理** | 全量提交 | 增量管理 |
| **幂等性** | 非幂等 | 幂等 |
| **版本控制** | 不友好 | 天然支持 |
| **适用场景** | 临时测试、快速验证 | 生产环境、持续部署 |
| **学习曲线** | 简单 | 需要理解合并机制 |
| **错误风险** | 低 | 需注意字段删除 |

### 4.2 字段管理差异

**kubectl create的字段管理**：

当使用`kubectl create`创建资源后，如果后续使用`kubectl apply`更新，可能会遇到字段管理冲突：

```bash
# 使用create创建资源
kubectl create -f deployment.yaml

# 后续使用apply更新
kubectl apply -f deployment.yaml
# Warning: resource deployments/nginx-deployment is missing the kubectl.kubernetes.io/last-applied-configuration annotation which is required by kubectl apply. 
# kubectl apply should only be used on resources created declaratively.
```

这是因为`kubectl create`不会设置`last-applied-configuration` annotation，导致apply无法进行三路合并。

**解决方案**：

```bash
# 方案1：使用server-side apply
kubectl apply -f deployment.yaml --server-side --force-conflicts

# 方案2：先删除再创建
kubectl delete -f deployment.yaml
kubectl apply -f deployment.yaml
```

**kubectl apply的字段管理**：

`kubectl apply`通过annotation精确管理字段：

- **明确管理的字段**：配置文件中指定的字段
- **保留的外部字段**：不在配置文件中，由其他系统添加的字段
- **删除的字段**：配置文件中删除的字段，会从资源中移除

### 4.3 更新行为差异

**kubectl create的更新行为**：

`kubectl create`不支持更新，只能创建。如果需要更新，必须使用其他命令：

```bash
# 错误方式：重复create
kubectl create -f deployment.yaml
# Error: resource already exists

# 正确方式：使用replace
kubectl replace -f deployment.yaml

# 或使用edit
kubectl edit deployment nginx-deployment
```

**kubectl apply的更新行为**：

`kubectl apply`智能处理创建和更新：

```bash
# 第一次：创建资源
kubectl apply -f deployment.yaml
# deployment.apps/nginx-deployment created

# 第二次：更新资源
kubectl apply -f deployment.yaml
# deployment.apps/nginx-deployment configured

# 第三次：无变化
kubectl apply -f deployment.yaml
# deployment.apps/nginx-deployment unchanged
```

### 4.4 删除字段的行为差异

**kubectl create创建的资源**：

如果使用`kubectl create`创建资源后，配置文件中删除了某个字段，再使用`kubectl apply`更新，该字段会被删除：

```yaml
# 初始配置
spec:
  replicas: 3
  strategy:
    type: RollingUpdate

# 修改后的配置（删除strategy）
spec:
  replicas: 3
```

执行`kubectl apply`后，strategy字段会被删除，因为三路合并检测到该字段从配置中移除了。

**解决方案**：

- 使用`kubectl edit`手动修改资源
- 使用`kubectl patch`进行部分更新
- 使用Server-Side Apply并明确字段所有权

## 五、使用场景与最佳实践

### 5.1 kubectl create适用场景

**1. 快速测试和验证**

```bash
# 快速创建一个临时Pod
kubectl run test-pod --image=busybox --rm -it --restart=Never -- sh

# 创建临时ConfigMap
kubectl create configmap test-config --from-literal=key1=value1
```

**2. 一次性资源创建**

```bash
# 创建Namespace
kubectl create namespace production

# 创建Secret
kubectl create secret generic db-secret --from-literal=password=secretpass
```

**3. 资源不存在时的初始化**

```bash
# 在CI/CD脚本中，确保资源不存在
if ! kubectl get namespace staging > /dev/null 2>&1; then
  kubectl create namespace staging
fi
```

### 5.2 kubectl apply适用场景

**1. 生产环境部署**

```bash
# 应用所有配置文件
kubectl apply -f configs/

# 应用特定环境的配置
kubectl apply -k overlays/production/
```

**2. GitOps工作流**

```bash
# 在CI/CD流水线中
kubectl apply -f deployment.yaml

# 使用Kustomize
kubectl apply -k .
```

**3. 持续部署**

```bash
# 更新镜像版本
kubectl apply -f deployment.yaml

# 自动扩缩容
kubectl apply -f hpa.yaml
```

### 5.3 最佳实践

**1. 统一使用kubectl apply**

在生产环境中，建议统一使用`kubectl apply`管理资源，避免混用create和apply：

```bash
# 推荐：统一使用apply
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f configmap.yaml

# 不推荐：混用create和apply
kubectl create -f deployment.yaml
kubectl apply -f service.yaml  # 可能导致字段管理冲突
```

**2. 使用版本控制**

将所有配置文件提交到Git仓库：

```bash
# 项目结构
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── kustomization.yaml
├── overlays/
│   ├── production/
│   │   ├── kustomization.yaml
│   │   └── patches/
│   └── staging/
│       ├── kustomization.yaml
│       └── patches/
└── .git/
```

**3. 使用Kustomize进行环境管理**

```yaml
# base/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.20.0
```

```yaml
# overlays/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../../base

patchesStrategicMerge:
- patches/deployment-replicas.yaml
```

```yaml
# overlays/production/patches/deployment-replicas.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 10  # 生产环境扩容到10个副本
```

**4. 使用Server-Side Apply**

对于复杂场景，推荐使用Server-Side Apply：

```bash
# 启用Server-Side Apply
kubectl apply -f deployment.yaml --server-side

# 强制解决冲突
kubectl apply -f deployment.yaml --server-side --force-conflicts
```

**5. 避免直接编辑资源**

不要使用`kubectl edit`直接修改资源，这会导致配置文件与集群状态不一致：

```bash
# 不推荐
kubectl edit deployment nginx-deployment

# 推荐
vim deployment.yaml
kubectl apply -f deployment.yaml
git add deployment.yaml
git commit -m "Update nginx deployment"
```

**6. 使用dry-run验证**

在应用配置前，使用dry-run验证：

```bash
# 客户端dry-run（不发送到服务器）
kubectl apply -f deployment.yaml --dry-run=client

# 服务端dry-run（发送到服务器但不持久化）
kubectl apply -f deployment.yaml --dry-run=server
```

**7. 理解字段删除行为**

当从配置文件中删除字段时，apply会删除该字段：

```yaml
# 初始配置
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0

# 如果只想修改replicas，必须保留strategy字段
spec:
  replicas: 5
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
```

或者使用`kubectl patch`进行部分更新：

```bash
kubectl patch deployment nginx-deployment -p '{"spec":{"replicas":5}}'
```

## 六、常见问题与故障排查

### 6.1 常见问题

#### 问题1：apply报错"missing last-applied-configuration"

**原因**：资源是通过`kubectl create`创建的，没有`last-applied-configuration` annotation。

**解决方案**：

```bash
# 方案1：使用Server-Side Apply
kubectl apply -f deployment.yaml --server-side

# 方案2：手动添加annotation
kubectl create -f deployment.yaml --dry-run=client -o yaml | kubectl apply -f -
```

#### 问题2：apply意外删除了字段

**原因**：配置文件中删除了字段，apply的三路合并机制会删除该字段。

**解决方案**：

```bash
# 查看apply会做什么修改
kubectl apply -f deployment.yaml --dry-run=server -o yaml

# 使用diff查看差异
kubectl diff -f deployment.yaml
```

#### 问题3：apply与edit冲突

**原因**：使用`kubectl edit`修改资源后，配置文件与集群状态不一致。

**解决方案**：

```bash
# 导出当前状态
kubectl get deployment nginx-deployment -o yaml > deployment-current.yaml

# 对比差异
diff deployment.yaml deployment-current.yaml

# 更新配置文件后重新apply
kubectl apply -f deployment.yaml
```

#### 问题4：多个apply操作冲突

**原因**：多个用户同时apply同一资源，导致字段冲突。

**解决方案**：

```bash
# 使用Server-Side Apply
kubectl apply -f deployment.yaml --server-side

# 查看字段管理者
kubectl get deployment nginx-deployment -o yaml | grep -A 30 managedFields
```

#### 问题5：apply无法删除资源

**原因**：配置文件中删除了资源定义，但apply不会自动删除资源。

**解决方案**：

```bash
# 显式删除资源
kubectl delete -f deployment.yaml

# 或使用kustomize删除
kubectl delete -k .
```

### 6.2 故障排查技巧

**1. 查看资源变更历史**

```bash
# 查看资源的变更历史
kubectl describe deployment nginx-deployment

# 查看事件
kubectl get events --field-selector involvedObject.name=nginx-deployment
```

**2. 对比配置差异**

```bash
# 使用diff查看差异
kubectl diff -f deployment.yaml

# 查看last-applied-configuration
kubectl get deployment nginx-deployment -o jsonpath='{.metadata.annotations.kubectl\.kubernetes\.io/last-applied-configuration}'
```

**3. 验证配置正确性**

```bash
# 验证YAML语法
kubectl apply -f deployment.yaml --dry-run=client --validate=true

# 验证服务端接受
kubectl apply -f deployment.yaml --dry-run=server
```

**4. 查看字段管理者**

```bash
# 查看Server-Side Apply的字段管理者
kubectl get deployment nginx-deployment -o yaml | grep -A 50 managedFields
```

**5. 强制解决冲突**

```bash
# 强制覆盖冲突字段
kubectl apply -f deployment.yaml --server-side --force-conflicts
```

## 七、深度原理剖析

### 7.1 kubectl apply的PATCH请求

`kubectl apply`底层使用HTTP PATCH请求更新资源，而不是PUT请求。PATCH允许部分更新，而PUT需要提交完整资源。

**PATCH类型**：

Kubernetes支持三种PATCH类型：

1. **Strategic Merge Patch**：Kubernetes特有的合并策略，根据字段标签智能合并
2. **JSON Merge Patch**：标准JSON合并补丁（RFC 7386）
3. **JSON Patch**：标准JSON补丁（RFC 6902）

`kubectl apply`默认使用Strategic Merge Patch，它根据字段的patchStrategy标签决定合并行为：

```go
type DeploymentSpec struct {
    Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`
    
    // +patchStrategy=merge
    Template PodTemplateSpec `json:"template" protobuf:"bytes,3,opt,name=template"`
}
```

**示例**：

```yaml
# 原始资源
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
      - name: sidecar
        image: busybox

# apply的配置
spec:
  replicas: 5
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.20.0

# 合并结果（Strategic Merge Patch会保留sidecar容器）
spec:
  replicas: 5
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.20.0
      - name: sidecar
        image: busybox
```

### 7.2 Strategic Merge Patch vs JSON Merge Patch

**Strategic Merge Patch**：

- Kubernetes特有，理解资源结构
- 对于数组字段，根据patchStrategy合并
- 保留不在配置文件中的元素

**JSON Merge Patch**：

- 标准RFC 7386
- 对于数组字段，完全替换
- 删除配置文件中不存在的字段

**对比示例**：

```yaml
# 原始资源
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
  - name: sidecar
    image: busybox

# JSON Merge Patch配置
spec:
  containers:
  - name: nginx
    image: nginx:1.20.0

# JSON Merge Patch结果（sidecar被删除）
spec:
  containers:
  - name: nginx
    image: nginx:1.20.0

# Strategic Merge Patch结果（sidecar保留）
spec:
  containers:
  - name: nginx
    image: nginx:1.20.0
  - name: sidecar
    image: busybox
```

### 7.3 Server-Side Apply的实现

Server-Side Apply将合并逻辑从kubectl移到API Server，提供更精确的字段管理。

**工作流程**：

1. kubectl发送包含配置的PATCH请求
2. API Server读取资源的managedFields
3. API Server计算字段所有权
4. API Server执行合并
5. API Server更新managedFields

**字段所有权规则**：

- 每个字段有一个或多个管理者（manager）
- 管理者可以是kubectl、controller、用户等
- 当多个管理者修改同一字段时，产生冲突
- 冲突需要强制解决或协商

**冲突解决**：

```bash
# 查看冲突
kubectl apply -f deployment.yaml --server-side
# Error: Apply failed with 1 conflict: conflict with "kubectl-client-side-apply"

# 强制解决冲突
kubectl apply -f deployment.yaml --server-side --force-conflicts
```

## 八、面试回答

**面试官问：请解释kubectl create和kubectl apply的区别，以及在什么场景下使用它们？**

**回答**：

kubectl create和kubectl apply的核心区别在于管理理念：create是命令式管理，apply是声明式管理。kubectl create是一次性的创建操作，如果资源已存在会报错，它不会记录资源的状态信息，适合临时测试和快速验证场景。kubectl apply则是声明式的，它关注资源的期望状态，无论资源是否存在都能正确处理，通过三路合并机制实现增量更新，并在资源的annotation中记录last-applied-configuration用于后续的合并计算。apply的核心优势在于支持版本控制、GitOps工作流和持续部署，是生产环境的标准实践。具体来说，apply会比较上次应用的配置、当前集群状态和本次提交的配置，智能地计算出需要更新的字段，保留不在配置文件中的字段。从底层实现看，create使用POST请求，apply使用PATCH请求配合Strategic Merge Patch或Server-Side Apply。在生产环境中，我强烈推荐统一使用kubectl apply管理资源，配合Git仓库进行版本控制，使用Server-Side Apply解决并发冲突，避免混用create和apply导致的字段管理混乱。

## 九、总结

kubectl create和kubectl apply的区别，本质上是命令式与声明式管理理念的差异。create适合快速测试和一次性操作，apply适合生产环境和持续部署。理解三路合并、last-applied-configuration、Server-Side Apply等核心机制，能够帮助我们更好地管理Kubernetes资源，避免常见的运维陷阱。

在实际工作中，建议遵循以下原则：

1. **生产环境统一使用kubectl apply**，建立声明式管理思维
2. **将配置文件纳入版本控制**，实现GitOps工作流
3. **使用Kustomize管理多环境配置**，避免配置重复
4. **启用Server-Side Apply**，解决并发冲突和字段管理问题
5. **避免混用create和apply**，防止字段管理混乱

掌握这些原理和最佳实践，将使你在Kubernetes资源管理中游刃有余，构建可靠、可维护的云原生应用。

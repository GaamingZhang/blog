---
date: 2026-03-11
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - AWS
  - EKS
  - Kubernetes
  - DevOps
---

# AWS EKS 生产环境管理实践

当你在凌晨三点被 PagerDuty 唤醒，发现生产环境的 EKS 集群节点突然不可用，Pod 大量 Pending，核心服务面临中断风险时，你是否真正理解 EKS 控制平面与数据平面的协作机制？当集群规模从 10 个节点扩展到 100 个节点时，网络插件、IP 地址管理、安全组规则该如何设计才能避免踩坑？当每月的 AWS 账单超出预算 50% 时，你是否知道从哪些维度优化 EKS 成本？

本文将深入 EKS 的内部架构，剖析控制平面与数据平面的协作机制，分享生产环境中的架构设计、性能调优、安全加固和成本优化实践，帮助你在真实场景中构建稳定、高效、安全的 Kubernetes 集群。

## 一、EKS 架构深度剖析

### 控制平面与数据平面的分离架构

EKS 采用托管控制平面（Managed Control Plane）架构，AWS 负责管理 Kubernetes 的核心组件，用户只需管理数据平面（Worker Nodes）。这种架构设计背后有多个关键实现细节。

**控制平面的多可用区高可用设计**

```
┌─────────────────────────────────────────────────────────────┐
│                  EKS Control Plane (AWS Managed)             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ API Server   │  │ API Server   │  │ API Server   │      │
│  │  (AZ-1)      │  │  (AZ-2)      │  │  (AZ-3)      │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                 │                 │               │
│         └─────────────────┼─────────────────┘               │
│                           │                                 │
│                    ┌──────▼──────┐                          │
│                    │ NLB (Cross  │                          │
│                    │  AZ LB)     │                          │
│                    └──────┬──────┘                          │
│                           │                                 │
│  ┌──────────────┐  ┌──────▼──────┐  ┌──────────────┐      │
│  │ etcd Cluster │  │ Scheduler   │  │ Controller   │      │
│  │  (Quorum)    │  │  (HA)       │  │  Manager     │      │
│  └──────────────┘  └─────────────┘  └──────────────┘      │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  CloudWatch Logs  │  CloudTrail  │  ConfigMap Sync   │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

EKS 控制平面的关键特性：

| 组件 | 实现方式 | 关键配置 |
|------|---------|---------|
| API Server | 跨 3 个 AZ 部署，自动扩缩容 | 通过 VPC Endpoint 访问（Private）或公网访问（Public） |
| etcd | 5 节点 Quorum 集群，跨 AZ 部署 | 自动备份，保留 7 天，支持 Point-in-Time Recovery |
| Scheduler | 多副本部署，Leader Election | 自动故障转移，默认调度策略 |
| Controller Manager | 多副本部署，Leader Election | Node Controller、Replication Controller 等核心控制器 |

**数据平面的网络通信机制**

EKS 数据平面与控制平面之间的通信是理解 EKS 网络架构的关键：

```
┌──────────────────────────────────────────────────────────────┐
│                     VPC (10.0.0.0/16)                        │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Private Subnet (10.0.1.0/24) - AZ-1                 │    │
│  │  ┌──────────────┐  ┌──────────────┐                 │    │
│  │  │ Worker Node  │  │ Worker Node  │                 │    │
│  │  │  kubelet     │  │  kubelet     │                 │    │
│  │  │  kube-proxy  │  │  kube-proxy  │                 │    │
│  │  └──────┬───────┘  └──────┬───────┘                 │    │
│  │         │                  │                          │    │
│  │         └──────────┬───────┘                          │    │
│  │                    │                                  │    │
│  │            ┌───────▼────────┐                        │    │
│  │            │ VPC Endpoint   │                        │    │
│  │            │ (com.amazonaws.│                        │    │
│  │            │  eks.api)      │                        │    │
│  │            └───────┬────────┘                        │    │
│  └────────────────────┼──────────────────────────────────┘    │
│                       │                                       │
│                       │ PrivateLink                          │
│                       │                                       │
│              ┌────────▼────────┐                             │
│              │  EKS Control    │                             │
│              │  Plane Endpoint │                             │
│              │  (AWS Managed)  │                             │
│              └─────────────────┘                             │
└──────────────────────────────────────────────────────────────┘
```

**关键通信路径**：

1. **kubelet → API Server**：通过 VPC Endpoint（Private 集群）或公网 NLB（Public 集群）访问
2. **API Server → kubelet**：通过 NodePort 或 Private IP 访问（用于 logs、exec、port-forward）
3. **API Server → AWS Cloud**：调用 EC2 API 管理 Node Group、调用 IAM API 验证 Pod Identity

### EKS 控制平面的认证与授权机制

EKS 使用 IAM 与 Kubernetes RBAC 的联合认证机制，这是其与自建 Kubernetes 集群最大的区别。

**认证流程详解**

```
┌──────────────────────────────────────────────────────────────┐
│              EKS Authentication Flow                          │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  1. 用户执行 kubectl 命令                                     │
│     └─ kubectl get pods                                       │
│                                                               │
│  2. kubectl 读取 kubeconfig，调用 AWS CLI 生成 Token          │
│     └─ aws eks get-token --cluster-name my-cluster           │
│                                                               │
│  3. AWS CLI 调用 STS API，获取临时凭证                        │
│     ├─ 使用 IAM User/Role 的长期凭证                          │
│     └─ 生成 Pre-signed URL（有效期 15 分钟）                  │
│                                                               │
│  4. kubectl 将 Token 放入 HTTP Header                        │
│     └─ Authorization: Bearer <token>                         │
│                                                               │
│  5. EKS API Server 接收请求，调用 Webhook Token Review        │
│     ├─ 解析 Token 中的 IAM Identity                           │
│     └─ 映射到 Kubernetes User: iam:UserArn                   │
│                                                               │
│  6. RBAC 授权检查                                             │
│     ├─ 检查 ClusterRoleBinding / RoleBinding                 │
│     └─ 验证 User 是否有权限执行操作                           │
│                                                               │
│  7. 返回结果                                                  │
│     └─ 允许/拒绝请求                                          │
└──────────────────────────────────────────────────────────────┘
```

**IAM 到 Kubernetes User 的映射规则**

EKS 通过 `aws-auth` ConfigMap 实现 IAM Identity 到 Kubernetes RBAC 的映射：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapUsers: |
    - userarn: arn:aws:iam::123456789012:user/admin
      username: admin
      groups:
        - system:masters
    
    - userarn: arn:aws:iam::123456789012:user/dev-user
      username: dev-user
      groups:
        - developers
  
  mapRoles: |
    - rolearn: arn:aws:iam::123456789012:role/EKSWorkerNodeRole
      username: system:node:{{EC2PrivateDNSName}}
      groups:
        - system:bootstrappers
        - system:nodes
    
    - rolearn: arn:aws:iam::123456789012:role/EKSFargateProfileRole
      username: system:node:{{SessionName}}
      groups:
        - system:bootstrappers
        - system:nodes
```

**关键实现细节**：

- **Token 生成**：AWS CLI 使用 IAM 凭证调用 STS `GetCallerIdentity` API，生成 Pre-signed URL，该 URL 包含签名和时间戳
- **Token 验证**：EKS API Server 配置了 `--authentication-token-webhook-config-file`，指向一个 Webhook Service，该 Service 验证 Token 的有效性并返回 IAM Identity
- **User 映射**：Webhook 返回的 User 格式为 `iam:<userArn>`，EKS 通过 `aws-auth` ConfigMap 将其映射到 Kubernetes User 和 Group

### EKS 的证书轮换机制

EKS 控制平面会自动管理证书的签发和轮换，但理解其机制有助于排查证书相关问题。

**证书类型与轮换策略**

| 证书类型 | 有效期 | 轮换策略 | 管理方 |
|---------|-------|---------|--------|
| API Server 证书 | 1 年 | 自动轮换（到期前 30 天） | AWS |
| etcd 证书 | 1 年 | 自动轮换（到期前 30 天） | AWS |
| kubelet 证书 | 1 年 | 自动轮换（到期前 7 天） | kubelet |
| kube-proxy 证书 | 1 年 | 自动轮换（到期前 7 天） | AWS |

**kubelet 证书轮换流程**

```
┌──────────────────────────────────────────────────────────────┐
│              kubelet Certificate Rotation                     │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  1. kubelet 启动时生成 CSR                                    │
│     ├─ 生成私钥和 CSR（Certificate Signing Request）          │
│     └─ 提交 CSR 到 API Server                                 │
│                                                               │
│  2. EKS Controller Manager 自动批准 CSR                       │
│     ├─ 检查 CSR 的合法性                                      │
│     └─ 使用 CA 证书签发证书                                   │
│                                                               │
│  3. kubelet 获取证书并写入磁盘                                │
│     └─ /var/lib/kubelet/pki/kubelet-client-current.pem       │
│                                                               │
│  4. 定期检查证书有效期                                        │
│     ├─ 如果剩余有效期 < 7 天，触发轮换                        │
│     └─ 重复步骤 1-3                                           │
│                                                               │
│  5. 证书轮换完成后，重启 kubelet                              │
│     └─ 无需重启 Pod，kubelet 热加载新证书                     │
└──────────────────────────────────────────────────────────────┘
```

## 二、节点组管理的深度实践

### Managed Node Group vs Self-Managed Node Group

EKS 提供两种节点组管理方式，理解其底层差异有助于做出正确的架构决策。

**Managed Node Group 的实现机制**

Managed Node Group 通过 ASG（Auto Scaling Group）和 Launch Template 实现，但 AWS 在其上封装了一层管理逻辑：

```
┌──────────────────────────────────────────────────────────────┐
│           Managed Node Group Architecture                     │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  EKS API                                                      │
│    │                                                          │
│    ├─ Create Node Group Request                               │
│    │  ├─ nodeRole: arn:aws:iam::...:role/NodeRole            │
│    │  ├─ subnets: [subnet-1, subnet-2, ...]                  │
│    │  ├─ scalingConfig:                                       │
│    │  │   minSize: 3                                          │
│    │  │   maxSize: 10                                         │
│    │  │   desiredSize: 5                                      │
│    │  └─ instanceTypes: [m5.large, m5.xlarge]                │
│    │                                                          │
│    └─ AWS EKS Controller                                      │
│        ├─ 创建 Launch Template                                │
│        │  ├─ UserData: Bootstrap Script                       │
│        │  ├─ IAM Instance Profile: NodeRole                   │
│        │  ├─ Security Groups: Cluster SG + Node SG            │
│        │  └─ Tags: kubernetes.io/cluster/<name>              │
│        │                                                      │
│        ├─ 创建 Auto Scaling Group                             │
│        │  ├─ 使用 Launch Template                             │
│        │  ├─ 配置健康检查类型：EC2 + ELB                      │
│        │  └─ 启用 Instance Protection（可选）                 │
│        │                                                      │
│        └─ 监听 ASG 事件                                       │
│            ├─ Instance Launch: 等待 Node Ready                │
│            └─ Instance Terminate: Drain Node                  │
│                                                               │
│  关键特性：                                                    │
│  ├─ 自动化节点生命周期管理                                    │
│  ├─ 集成 Cluster Autoscaler                                   │
│  ├─ 支持 Spot Instance                                        │
│  └─ 自动应用安全补丁                                          │
└──────────────────────────────────────────────────────────────┘
```

**Managed Node Group 的 Bootstrap Script 解析**

当节点启动时，EKS 会注入一段 UserData，执行 Bootstrap 脚本：

```bash
#!/bin/bash
set -o xtrace
/etc/eks/bootstrap.sh my-cluster \
  --b64-cluster-ca $CLUSTER_CA \
  --apiserver-endpoint $API_SERVER_ENDPOINT \
  --kubelet-extra-args '--node-labels=env=prod,nodegroup-type=managed' \
  --dns-cluster-ip $CLUSTER_DNS_IP \
  --container-runtime containerd
```

Bootstrap 脚本的核心逻辑：

1. **安装必要工具**：aws-cli、kubectl、aws-iam-authenticator
2. **生成 kubeconfig**：使用节点 IAM Role 生成认证 Token
3. **启动 kubelet**：连接到 EKS API Server，注册节点
4. **配置容器运行时**：默认使用 containerd（EKS 1.24+）

**Self-Managed Node Group 的灵活性**

Self-Managed Node Group 需要用户自行管理 ASG 和 Launch Template，但提供了更大的灵活性：

```yaml
# 自定义 Launch Template
apiVersion: ec2.aws.upbound.io/v1beta1
kind: LaunchTemplate
metadata:
  name: my-node-template
spec:
  forProvider:
    region: us-west-2
    instanceType: m5.large
    iamInstanceProfile:
      name: my-node-profile
    blockDeviceMappings:
      - deviceName: /dev/xvda
        ebs:
          volumeSize: 100
          volumeType: gp3
          encrypted: true
          kmsKeyId: alias/aws/ebs
    networkInterfaces:
      - deviceIndex: 0
        securityGroups:
          - sg-12345
        deleteOnTermination: true
    userData: |
      #!/bin/bash
      /etc/eks/bootstrap.sh my-cluster \
        --kubelet-extra-args '--max-pods=200' \
        --container-runtime containerd
    metadataOptions:
      httpEndpoint: enabled
      httpTokens: required
      httpPutResponseHopLimit: 2
    tagSpecifications:
      - resourceType: instance
        tags:
          Name: my-eks-node
          kubernetes.io/cluster/my-cluster: owned
```

**关键差异对比**

| 特性 | Managed Node Group | Self-Managed Node Group |
|------|-------------------|------------------------|
| 节点生命周期管理 | 自动化（创建、更新、终止） | 手动管理 |
| 滚动更新 | 自动 Drain + 替换 | 需手动配置 Lifecycle Hook |
| Spot Instance | 原生支持，自动处理中断 | 需手动配置 Spot Interruption Handler |
| 自定义 AMI | 支持（通过 Launch Template） | 支持（完全自定义） |
| 自定义 UserData | 受限（只能添加 kubelet 参数） | 完全自定义 |
| 节点故障恢复 | 自动替换不健康节点 | 需配置 ASG 健康检查 |
| 成本 | 无额外费用 | 无额外费用 |

### 节点自动扩缩容的实现原理

EKS 集成 Cluster Autoscaler（CA）实现节点的自动扩缩容，理解其工作原理有助于优化扩缩容策略。

**Cluster Autoscaler 的工作流程**

```
┌──────────────────────────────────────────────────────────────┐
│         Cluster Autoscaler Reconciliation Loop                │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  1. 监听 Kubernetes Scheduler 的 Pending Pods                 │
│     └─ 通过 Informer 监听 Pod 事件                            │
│                                                               │
│  2. 分析 Pod Pending 的原因                                   │
│     ├─ PodUnschedulable: 资源不足                             │
│     ├─ Insufficient cpu/memory: 节点资源不足                  │
│     ├─ NodeAffinity: 没有匹配的节点                           │
│     └─ Taints/Tolerations: 节点污点不匹配                     │
│                                                               │
│  3. 计算 Node Group 的扩容需求                                │
│     ├─ 模拟调度：如果新增一个节点，Pod 能否调度？              │
│     ├─ 选择最优的 Node Group                                  │
│     │  ├─ 优先选择已有实例类型的 Node Group                   │
│     │  └─ 考虑 Spot Instance 的成本优化                       │
│     └─ 计算需要扩容的节点数量                                 │
│                                                               │
│  4. 调用 AWS ASG API 扩容                                     │
│     ├─ Update Auto Scaling Group                              │
│     │  └─ Set Desired Capacity                                │
│     └─ 等待新节点 Ready                                       │
│                                                               │
│  5. 缩容逻辑（可选）                                          │
│     ├─ 检查节点利用率 < 50%（默认阈值）                       │
│     ├─ 模拟迁移：Pod 能否调度到其他节点？                      │
│     ├─ 调用 ASG API 缩容                                      │
│     └─ Drain Node + Terminate Instance                        │
└──────────────────────────────────────────────────────────────┘
```

**CA 的关键配置参数**

```yaml
# Cluster Autoscaler Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cluster-autoscaler
  namespace: kube-system
spec:
  template:
    spec:
      containers:
        - image: k8s.gcr.io/autoscaling/cluster-autoscaler:v1.25.0
          name: cluster-autoscaler
          command:
            - ./cluster-autoscaler
            - --v=4
            - --stderrthreshold=info
            - --cloud-provider=aws
            - --skip-nodes-with-local-storage=false
            - --expander=least-waste
            - --balance-similar-node-groups
            - --skip-nodes-with-system-pods=false
            - --scale-down-unneeded-time=10m
            - --scale-down-delay-after-add=10m
            - --scale-down-delay-after-failure=3m
            - --scale-down-delay-after-delete=10s
            - --scale-down-unready-time=20m
            - --scale-down-utilization-threshold=0.5
            - --max-node-provision-time=15m
          env:
            - name: AWS_REGION
              value: us-west-2
```

**关键参数解析**：

| 参数 | 默认值 | 说明 |
|------|-------|------|
| `--expander` | random | 扩容策略：random（随机）、least-waste（最小浪费）、priority（优先级） |
| `--scale-down-utilization-threshold` | 0.5 | 节点利用率低于此值时考虑缩容 |
| `--scale-down-unneeded-time` | 10m | 节点空闲超过此时间才缩容 |
| `--scale-down-delay-after-add` | 10m | 扩容后多久开始考虑缩容 |
| `--max-node-provision-time` | 15m | 等待新节点 Ready 的超时时间 |

**Spot Instance 的最佳实践**

使用 Spot Instance 可以节省高达 90% 的成本，但需要正确处理中断：

```yaml
# Managed Node Group with Spot Instance
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig
metadata:
  name: my-cluster
  region: us-west-2
managedNodeGroups:
  - name: spot-workers
    instanceType: mixed
    desiredCapacity: 10
    minSize: 5
    maxSize: 20
    spot: true
    instanceSelector:
      vCPUs: 2-4
      memory: 4-8Gi
    labels:
      type: spot
    taints:
      - key: spot
        value: "true"
        effect: NoSchedule
    iam:
      withAddonPolicies:
        autoScaler: true
```

**Spot Instance 中断处理**：

```yaml
# AWS Node Termination Handler
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: aws-node-termination-handler
  namespace: kube-system
spec:
  template:
    spec:
      containers:
        - name: aws-node-termination-handler
          image: public.ecr.aws/aws-node-termination-handler/aws-node-termination-handler:v1.18.0
          args:
            - --node-name=$(NODE_NAME)
            - --drain-grace-period=120
            - --monitor-grace-period=600
            - --skip-taint-node
          env:
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
```

Node Termination Handler 的工作原理：

1. 监听 EC2 Spot Instance Interruption Notice（提前 2 分钟通知）
2. 调用 `kubectl drain` 驱逐 Pod
3. 等待 Pod 迁移完成
4. 标记节点为不可调度

## 三、网络架构的深度设计

### VPC CNI 插件的工作原理

EKS 默认使用 Amazon VPC CNI 插件，它直接将 Pod IP 映射到 VPC IP，理解其实现机制对于网络排错和性能优化至关重要。

**VPC CNI 的核心组件**

```
┌──────────────────────────────────────────────────────────────┐
│              Amazon VPC CNI Architecture                      │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Worker Node                                        │     │
│  │  ┌──────────────────────────────────────────────┐ │     │
│  │  │  aws-node DaemonSet                           │ │     │
│  │  │  ├─ aws-k8s-agent (CNI Plugin)                │ │     │
│  │  │  │  └─ 处理 ADD/DEL/CHECK 命令                │ │     │
│  │  │  │                                            │ │     │
│  │  │  ├─ aws-vpc-cni (IPAMD)                       │ │     │
│  │  │  │  ├─ 管理 ENI 和 Secondary IP               │ │     │
│  │  │  │  ├─ 维护 IP 地址池                         │ │     │
│  │  │  │  └─ 调用 EC2 API 分配/释放 IP              │ │     │
│  │  │  │                                            │ │     │
│  │  │  └─ aws-vpc-cni-init (Init Container)         │ │     │
│  │  │     └─ 配置主机网络（sysctl, iptables）       │ │     │
│  │  └──────────────────────────────────────────────┘ │     │
│  │                                                     │     │
│  │  ┌──────────────────────────────────────────────┐ │     │
│  │  │  Network Interfaces                           │ │     │
│  │  │  ├─ eth0 (Primary ENI)                        │ │     │
│  │  │  │  └─ IP: 10.0.1.10 (Node IP)               │ │     │
│  │  │  │                                            │ │     │
│  │  │  ├─ eth1 (Secondary ENI)                      │ │     │
│  │  │  │  ├─ IP: 10.0.1.11                         │ │     │
│  │  │  │  ├─ IP: 10.0.1.12 (Pod IP)                │ │     │
│  │  │  │  └─ IP: 10.0.1.13 (Pod IP)                │ │     │
│  │  │  │                                            │ │     │
│  │  │  └─ eth2 (Secondary ENI)                      │ │     │
│  │  │     ├─ IP: 10.0.1.14                         │ │     │
│  │  │     └─ IP: 10.0.1.15 (Pod IP)                │ │     │
│  │  └──────────────────────────────────────────────┘ │     │
│  └────────────────────────────────────────────────────┘     │
│                                                               │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Pod Network Namespace                             │     │
│  │  ├─ veth pair: vethxxxx <-> eth0 (in Pod)         │     │
│  │  └─ IP: 10.0.1.12 (Secondary IP on ENI)           │     │
│  └────────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────────┘
```

**IPAMD 的 IP 地址管理策略**

IPAMD（IP Address Management Daemon）负责管理节点上的 IP 地址池：

```go
// 简化的 IPAMD 逻辑
type IPAMD struct {
    eniManager      *ENIManager
    ipPool          *IPPool
    k8sClient       *K8sClient
    maxIPPerENI     int  // 每个 ENI 的最大 IP 数
    warmIPTarget    int  // 预热的 IP 数量
    minIPTarget     int  // 最小 IP 数量
}

func (ipamd *IPAMD) reconcile() {
    // 1. 检查当前 IP 池
    currentIPs := ipamd.ipPool.AvailableIPs()
    
    // 2. 如果 IP 不足，分配新的 ENI 或 IP
    if currentIPs < ipamd.warmIPTarget {
        if ipamd.canAllocateNewENI() {
            // 分配新的 ENI
            eni := ipamd.eniManager.AllocateENI()
            // 分配 Secondary IPs
            ips := ipamd.eniManager.AllocateIPs(eni, ipamd.maxIPPerENI)
            ipamd.ipPool.Add(ips)
        } else {
            // 在现有 ENI 上分配更多 IP
            for _, eni := range ipamd.eniManager.GetENIs() {
                if len(eni.IPs) < ipamd.maxIPPerENI {
                    ips := ipamd.eniManager.AllocateIPs(eni, ipamd.maxIPPerENI - len(eni.IPs))
                    ipamd.ipPool.Add(ips)
                }
            }
        }
    }
    
    // 3. 如果 IP 过多，释放多余的 IP
    if currentIPs > ipamd.maxIPTarget {
        ipamd.releaseExcessIPs()
    }
}
```

**关键配置参数**

```yaml
# aws-node ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: amazon-vpc-cni
  namespace: kube-system
data:
  # 每个 ENI 的最大 IP 数（取决于实例类型）
  # m5.large: 10 IPs/ENI, 3 ENIs → 30 Pods max
  # m5.xlarge: 15 IPs/ENI, 4 ENIs → 58 Pods max
  
  # 预热的 IP 数量（避免 Pod 创建时等待 IP 分配）
  WARM_IP_TARGET: "5"
  
  # 最小 IP 数量（确保有足够的 IP 可用）
  MINIMUM_IP_TARGET: "10"
  
  # 预热的 ENI 数量（避免等待 ENI 分配）
  WARM_ENI_TARGET: "1"
  
  # 启用 Pod ENI（每个 Pod 独占一个 ENI，用于高性能场景）
  ENABLE_POD_ENI: "false"
  
  # 启用前缀委派（IPv6 或 IPv4 前缀，大幅增加 Pod 密度）
  ENABLE_PREFIX_DELEGATION: "false"
```

**VPC CNI 的性能瓶颈与优化**

| 问题 | 原因 | 解决方案 |
|------|------|---------|
| Pod 创建慢 | IPAMD 需要调用 EC2 API 分配 IP | 配置 `WARM_IP_TARGET` 预热 IP |
| 节点 Pod 数量限制 | ENI 和 Secondary IP 数量有限 | 使用 `ENABLE_PREFIX_DELEGATION` 或更换实例类型 |
| IP 地址耗尽 | VPC 子网 IP 不足 | 扩大子网 CIDR 或使用自定义网络 |
| 跨节点通信延迟 | 安全组规则过多 | 优化安全组规则，使用 Security Group for Pods |

### Security Group for Pods 的实现机制

默认情况下，所有 Pod 共享节点的安全组。Security Group for Pods 允许为每个 Pod 或 Pod 组分配独立的安全组，实现更精细的网络隔离。

**实现原理**

```
┌──────────────────────────────────────────────────────────────┐
│         Security Group for Pods Architecture                  │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  1. 创建 SecurityGroupPolicy CRD                              │
│     apiVersion: vpcresources.k8s.aws/v1beta1                 │
│     kind: SecurityGroupPolicy                                 │
│     metadata:                                                 │
│       name: my-app-sg                                         │
│     spec:                                                     │
│       podSelector:                                            │
│         matchLabels:                                          │
│           app: my-app                                         │
│       securityGroups:                                         │
│         groupIds:                                             │
│           - sg-12345  # 允许访问数据库                        │
│           - sg-67890  # 允许访问外部 API                      │
│                                                               │
│  2. VPC Resource Controller 监听 Pod 创建                     │
│     ├─ 检查 Pod 是否匹配 SecurityGroupPolicy                  │
│     ├─ 调用 EC2 API 创建 ENI                                  │
│     │  └─ Attach Security Groups to ENI                       │
│     └─ 将 ENI 挂载到 Pod 的 Network Namespace                 │
│                                                               │
│  3. Pod 使用独立的 ENI                                        │
│     ├─ eth0: Pod 独占的 ENI                                   │
│     ├─ IP: 10.0.1.100                                        │
│     └─ Security Groups: sg-12345, sg-67890                   │
│                                                               │
│  4. 流量走向                                                  │
│     ├─ Pod → 数据库: 通过 sg-12345 允许                       │
│     └─ Pod → 外部 API: 通过 sg-67890 允许                     │
└──────────────────────────────────────────────────────────────┘
```

**关键限制**：

- 只支持 Linux 节点
- 每个节点最多 9 个分支 ENI（Branch ENI）
- 不支持 Windows 节点、Fargate、Host Network Pod
- 需要启用 `ENABLE_POD_ENI=true`

### 网络策略的实现与性能

EKS 默认不启用 Network Policy，需要安装 Calico 或 Cilium 等网络策略引擎。

**Calico 的实现机制**

```yaml
# 安装 Calico
kubectl apply -f https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/master/config/master/calico-operator.yaml

# Network Policy 示例
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: api-network-policy
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: api
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: production
        - podSelector:
            matchLabels:
              app: frontend
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: database
      ports:
        - protocol: TCP
          port: 5432
    - to:
        - namespaceSelector: {}
          podSelector:
            matchLabels:
              k8s-app: kube-dns
      ports:
        - protocol: UDP
          port: 53
```

**Calico 的性能优化**

```yaml
# Calico 配置
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  calicoNetwork:
    ipam:
      type: AmazonVPC
    mtu: 9000  # 启用 Jumbo Frame
    nodeAddressAutodetectionV4:
      interface: eth0
  # 性能调优
  felixConfiguration:
    bpfEnabled: true  # 启用 eBPF 数据平面
    bpfKubeProxyIptablesCleanupEnabled: true
    bpfExternalServiceMode: Tunnel
```

## 四、安全加固的深度实践

### Pod Security Standards 的落地

EKS 支持通过 Pod Security Admission（PSA）控制器实施 Pod 安全标准。

**PSA 的实现机制**

```yaml
# 在 Namespace 上配置 PSA
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/enforce-version: latest
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/audit-version: latest
    pod-security.kubernetes.io/warn: restricted
    pod-security.kubernetes.io/warn-version: latest
```

**Restricted 级别的检查项**

| 检查项 | 要求 | 说明 |
|--------|------|------|
| `runAsNonRoot` | true | Pod 必须以非 root 用户运行 |
| `runAsUser` | > 0 | 用户 ID 必须大于 0 |
| `allowPrivilegeEscalation` | false | 禁止权限提升 |
| `seccompProfile` | type: RuntimeDefault | 使用运行时默认的 seccomp 配置 |
| `capabilities.drop` | ALL | 丢弃所有 Linux Capabilities |
| `volumes` | 限制类型 | 只允许 configMap、emptyDir、projected、secret、downwardAPI、persistentVolumeClaim |

### IAM Roles for Service Accounts (IRSA)

IRSA 是 EKS 的核心安全特性，允许 Pod 使用 IAM Role 访问 AWS 服务，而无需在节点上配置 IAM 凭证。

**IRSA 的实现原理**

```
┌──────────────────────────────────────────────────────────────┐
│              IRSA Authentication Flow                         │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  1. 创建 IAM OIDC Provider                                    │
│     ├─ EKS 集群创建时自动创建 OIDC Identity Provider          │
│     └─ URL: https://oidc.eks.<region>.amazonaws.com/id/<ID> │
│                                                               │
│  2. 创建 IAM Role 和 Trust Policy                             │
│     {                                                         │
│       "Version": "2012-10-17",                               │
│       "Statement": [{                                        │
│         "Effect": "Allow",                                   │
│         "Principal": {                                       │
│           "Federated": "arn:aws:iam::...:oidc-provider/..." │
│         },                                                   │
│         "Action": "sts:AssumeRoleWithWebIdentity",           │
│         "Condition": {                                       │
│           "StringEquals": {                                  │
│             "oidc.eks...:aud": "sts.amazonaws.com",         │
│             "oidc.eks...:sub": "system:serviceaccount:ns:sa"│
│           }                                                  │
│         }                                                    │
│       }]                                                     │
│     }                                                         │
│                                                               │
│  3. 创建 ServiceAccount 并关联 IAM Role                       │
│     apiVersion: v1                                           │
│     kind: ServiceAccount                                     │
│     metadata:                                                 │
│       name: my-app-sa                                         │
│       namespace: production                                   │
│       annotations:                                            │
│         eks.amazonaws.com/role-arn: arn:aws:iam::...:role/..│
│                                                               │
│  4. Pod 启动时，EKS Pod Identity Webhook 注入环境变量         │
│     AWS_ROLE_ARN=arn:aws:iam::...:role/my-role               │
│     AWS_WEB_IDENTITY_TOKEN_FILE=/var/run/secrets/.../token   │
│                                                               │
│  5. AWS SDK 自动调用 STS AssumeRoleWithWebIdentity           │
│     ├─ 读取 ServiceAccount Token (JWT)                       │
│     ├─ 调用 STS API，传入 Token                              │
│     └─ 获取临时凭证（AccessKey + SecretKey + SessionToken）  │
│                                                               │
│  6. 使用临时凭证访问 AWS 服务                                 │
│     └─ S3、DynamoDB、Secrets Manager 等                      │
└──────────────────────────────────────────────────────────────┘
```

**关键实现细节**：

- **OIDC Token**：Kubernetes 为 ServiceAccount 生成的 JWT Token，包含 ServiceAccount 的身份信息
- **Token 轮换**：Token 有效期 1 小时，自动轮换
- **权限边界**：IAM Role 的权限策略控制 Pod 可以访问的 AWS 资源

### Secrets 管理的最佳实践

EKS 提供多种 Secrets 管理方案，生产环境推荐使用 External Secrets Operator。

**External Secrets Operator 架构**

```yaml
# 安装 External Secrets Operator
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: aws-secretsmanager
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-west-2
      auth:
        jwt:
          serviceAccountRef:
            name: external-secrets-sa
            namespace: external-secrets

---
# 从 AWS Secrets Manager 同步 Secret
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: database-credentials
  namespace: production
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secretsmanager
    kind: ClusterSecretStore
  target:
    name: database-credentials
    creationPolicy: Owner
  data:
    - secretKey: username
      remoteRef:
        key: prod/database/credentials
        property: username
    - secretKey: password
      remoteRef:
        key: prod/database/credentials
        property: password
```

**工作流程**：

1. External Secrets Controller 监听 `ExternalSecret` CR
2. 使用 IRSA 认证访问 AWS Secrets Manager
3. 获取 Secret 值并创建 Kubernetes Secret
4. 定期刷新（`refreshInterval`）确保值最新

## 五、性能调优与故障排查

### 控制平面性能调优

EKS 控制平面的性能直接影响集群的稳定性，以下是一些关键调优点。

**API Server 性能优化**

```yaml
# kube-apiserver 配置（EKS 托管，仅供参考）
--max-requests-inflight=800          # 最大并发请求数
--max-mutating-requests-inflight=400 # 最大并发 mutating 请求数
--request-timeout=60s                # 请求超时时间
--watch-cache-sizes=deployments.apps#100,replicasets.apps#100
```

**Controller Manager 调优**

```yaml
# kube-controller-manager 配置（EKS 托管，仅供参考）
--node-sync-period=10s               # 节点同步周期
--node-monitor-period=5s             # 节点监控周期
--node-monitor-grace-period=40s      # 节点故障判定时间
--pod-eviction-timeout=5m            # Pod 驱逐超时时间
--concurrent-deployment-syncs=10     # 并发 Deployment 同步数
--concurrent-replicaset-syncs=10     # 并发 ReplicaSet 同步数
```

### 节点性能调优

**kubelet 性能优化**

```bash
# /etc/kubernetes/kubelet/kubelet-config.json
{
  "kind": "KubeletConfiguration",
  "apiVersion": "kubelet.config.k8s.io/v1beta1",
  "maxPods": 110,
  "podPidsLimit": 4096,
  "imageMinimumGCAge": "2m",
  "imageGCHighThresholdPercent": 85,
  "imageGCLowThresholdPercent": 80,
  "evictionHard": {
    "memory.available": "500Mi",
    "nodefs.available": "10%",
    "nodefs.inodesFree": "5%",
    "imagefs.available": "15%"
  },
  "evictionSoft": {
    "memory.available": "750Mi",
    "nodefs.available": "15%"
  },
  "evictionSoftGracePeriod": {
    "memory.available": "1m30s",
    "nodefs.available": "1m30s"
  },
  "evictionMaxPodGracePeriod": 60,
  "kubeAPIQPS": 50,
  "kubeAPIBurst": 100,
  "serializeImagePulls": false,
  "registryPullQPS": 10,
  "registryBurst": 20
}
```

**内核参数调优**

```bash
# /etc/sysctl.d/99-kubernetes.conf
net.core.somaxconn=32768
net.core.netdev_max_backlog=32768
net.ipv4.tcp_max_syn_backlog=32768
net.ipv4.tcp_syncookies=1
net.ipv4.tcp_tw_reuse=1
net.ipv4.tcp_fin_timeout=30
net.ipv4.tcp_keepalive_time=600
net.ipv4.tcp_keepalive_intvl=30
net.ipv4.tcp_keepalive_probes=10
net.ipv4.tcp_max_tw_buckets=32768
net.ipv4.ip_local_port_range=1024 65535
fs.file-max=2097152
fs.inotify.max_user_instances=8192
fs.inotify.max_user_watches=524288
vm.max_map_count=262144
```

### 常见故障排查

**问题 1：节点 NotReady**

原因：kubelet 无法连接 API Server 或节点资源不足

排查步骤：

```bash
# 1. 检查 kubelet 日志
journalctl -u kubelet -f

# 2. 检查节点状态
kubectl describe node <node-name>

# 3. 检查节点资源使用
kubectl top node
ssh <node-ip> "df -h && free -m"

# 4. 检查网络连接
ssh <node-ip> "curl -k https://<api-server-endpoint>/healthz"

# 5. 检查 kubelet 配置
ssh <node-ip> "cat /etc/kubernetes/kubelet/kubelet-config.json"
```

**问题 2：Pod 一直 Pending**

原因：资源不足、节点选择器不匹配、污点容忍度问题

排查步骤：

```bash
# 1. 查看 Pod 事件
kubectl describe pod <pod-name> -n <namespace>

# 2. 检查调度器日志
kubectl logs -n kube-system deployment/cluster-autoscaler

# 3. 检查节点资源
kubectl describe nodes | grep -A 5 "Allocated resources"

# 4. 检查节点标签和污点
kubectl get nodes --show-labels
kubectl describe nodes | grep -A 3 Taints

# 5. 模拟调度
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: test-schedule
spec:
  containers:
    - name: test
      image: busybox
      resources:
        requests:
          cpu: 1
          memory: 1Gi
EOF
```

**问题 3：CoreDNS 解析失败**

原因：CoreDNS Pod 异常、ConfigMap 配置错误、网络策略限制

排查步骤：

```bash
# 1. 检查 CoreDNS Pod 状态
kubectl get pods -n kube-system -l k8s-app=kube-dns

# 2. 检查 CoreDNS 日志
kubectl logs -n kube-system -l k8s-app=kube-dns

# 3. 检查 CoreDNS ConfigMap
kubectl get configmap coredns -n kube-system -o yaml

# 4. 测试 DNS 解析
kubectl run -it --rm --restart=Never busybox --image=busybox:1.28 -- nslookup kubernetes.default

# 5. 检查 Service 和 Endpoints
kubectl get svc -n kube-system kube-dns
kubectl get endpoints -n kube-system kube-dns

# 6. 检查网络策略
kubectl get networkpolicy -n kube-system
```

**问题 4：VPC CNI IP 耗尽**

原因：子网 IP 地址不足、ENI 限制

排查步骤：

```bash
# 1. 检查节点 IP 池
kubectl exec -n kube-system aws-node-xxxx -- /app/aws-vpc-cni --ipamd

# 2. 检查子网 IP 使用情况
aws ec2 describe-subnets --subnet-ids <subnet-id> --query 'Subnets[0].[AvailableIpAddressCount,CidrBlock]'

# 3. 检查 ENI 配额
aws ec2 describe-network-interfaces --filters Name=attachment.instance-id,Values=<instance-id>

# 4. 检查 aws-node 日志
kubectl logs -n kube-system aws-node-xxxx

# 5. 临时解决：重启 aws-node Pod
kubectl delete pod -n kube-system -l k8s-app=aws-node
```

## 六、成本优化实践

### 资源请求与限制的优化

合理的资源请求和限制是成本优化的基础。

**资源请求优化策略**

```yaml
# 使用 Vertical Pod Autoscaler (VPA) 推荐资源请求
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: my-app-vpa
  namespace: production
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
  updatePolicy:
    updateMode: "Auto"  # 自动更新资源请求
  resourcePolicy:
    containerPolicies:
      - containerName: my-app
        minAllowed:
          cpu: 100m
          memory: 128Mi
        maxAllowed:
          cpu: 2
          memory: 2Gi
        controlledResources: ["cpu", "memory"]
        controlledValues: RequestsAndLimits
```

**VPA 的工作原理**

```
┌──────────────────────────────────────────────────────────────┐
│              VPA Recommendation Process                       │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  1. Prometheus 收集 Pod 资源使用指标                          │
│     ├─ container_cpu_usage_seconds_total                     │
│     └─ container_memory_working_set_bytes                    │
│                                                               │
│  2. VPA Recommender 分析历史数据                              │
│     ├─ 计算 CPU 和内存的 P95/P99 使用量                      │
│     ├─ 考虑峰值和突发需求                                     │
│     └─ 生成推荐值                                            │
│                                                               │
│  3. VPA Updater 应用推荐值                                    │
│     ├─ 检查 Pod 是否符合更新条件                              │
│     ├─ 驱逐 Pod（触发重建）                                   │
│     └─ 新 Pod 使用新的资源请求                                │
│                                                               │
│  4. VPA Admission Controller 注入资源请求                     │
│     └─ 在 Pod 创建时自动设置资源请求                          │
└──────────────────────────────────────────────────────────────┘
```

### Spot Instance 的成本优化

**Spot Instance 策略**

```yaml
# 使用 Spot Instance 的 Node Group
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig
metadata:
  name: my-cluster
  region: us-west-2
managedNodeGroups:
  # On-Demand 节点组（核心服务）
  - name: on-demand-core
    instanceType: m5.large
    desiredCapacity: 3
    minSize: 2
    maxSize: 5
    labels:
      type: on-demand
      workload-type: critical
    taints:
      - key: critical
        value: "true"
        effect: NoSchedule
  
  # Spot 节点组（可容忍中断的工作负载）
  - name: spot-workers
    instanceType: mixed
    desiredCapacity: 10
    minSize: 5
    maxSize: 20
    spot: true
    instanceSelector:
      vCPUs: 2-4
      memory: 4-8Gi
    labels:
      type: spot
      workload-type: burstable
    taints:
      - key: spot
        value: "true"
        effect: NoSchedule
```

**Pod 调度到 Spot 节点**

```yaml
# 使用 Node Affinity 和 Toleration
apiVersion: apps/v1
kind: Deployment
metadata:
  name: batch-job
spec:
  template:
    spec:
      affinity:
        nodeAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              preference:
                matchExpressions:
                  - key: type
                    operator: In
                    values:
                      - spot
      tolerations:
        - key: spot
          operator: Equal
          value: "true"
          effect: NoSchedule
      containers:
        - name: batch-job
          image: my-batch-job:latest
```

### 集群自动扩缩容优化

**Karpenter vs Cluster Autoscaler**

| 特性 | Cluster Autoscaler | Karpenter |
|------|-------------------|-----------|
| 扩容速度 | 分钟级（等待 ASG 更新） | 秒级（直接创建 EC2） |
| 实例类型选择 | 预定义 Node Group | 动态选择最优实例类型 |
| Spot Instance | 支持（需要配置多个 Node Group） | 原生支持，自动切换实例类型 |
| 成本优化 | 需要手动配置 | 自动选择最便宜的实例 |
| 扩缩容策略 | 基于规则 | 基于调度约束 |

**Karpenter 配置示例**

```yaml
# Karpenter Provisioner
apiVersion: karpenter.sh/v1alpha5
kind: Provisioner
metadata:
  name: default
spec:
  requirements:
    - key: karpenter.sh/capacity-type
      operator: In
      values: ["spot", "on-demand"]
    - key: kubernetes.io/arch
      operator: In
      values: ["amd64"]
    - key: karpenter.k8s.aws/instance-category
      operator: In
      values: ["c", "m", "r"]
    - key: karpenter.k8s.aws/instance-generation
      operator: Gt
      values: ["5"]
  limits:
    resources:
      cpu: 1000
      memory: 1000Gi
  provider:
    subnetSelector:
      karpenter.sh/discovery: my-cluster
    securityGroupSelector:
      karpenter.sh/discovery: my-cluster
  consolidation:
    enabled: true
```

## 小结

- **EKS 控制平面**采用托管架构，AWS 负责管理 API Server、etcd、Scheduler、Controller Manager 等核心组件，用户只需管理数据平面。控制平面跨 3 个可用区部署，通过 VPC Endpoint 或公网 NLB 访问
- **节点组管理**分为 Managed Node Group 和 Self-Managed Node Group，前者自动化程度高，后者灵活性更强。Cluster Autoscaler 通过监听 Pending Pod 触发扩容，通过检查节点利用率触发缩容
- **网络架构**基于 Amazon VPC CNI，Pod IP 直接映射到 VPC IP。IPAMD 管理 ENI 和 Secondary IP，支持 Security Group for Pods 实现精细化的网络隔离
- **安全加固**包括 Pod Security Standards、IRSA、Secrets 管理等。IRSA 通过 OIDC 联合认证，让 Pod 使用 IAM Role 访问 AWS 服务，无需在节点上配置凭证
- **性能调优**需要从控制平面、节点、网络等多个维度入手。关键参数包括 API Server 并发数、kubelet 资源配置、内核参数等
- **成本优化**通过合理的资源请求、Spot Instance、集群自动扩缩容等手段实现。Karpenter 相比 Cluster Autoscaler 提供更快的扩容速度和更智能的实例选择

---

## 常见问题

### Q1：EKS 控制平面与数据平面之间的网络延迟如何优化？

EKS 控制平面与数据平面之间的网络延迟主要来自以下几个方面：

1. **VPC Endpoint 优化**：使用 Private 集群时，确保 Worker Node 与 VPC Endpoint 在同一可用区，避免跨 AZ 流量。可以通过配置 `endpointPrivateAccess` 和 `endpointPublicAccess` 控制访问方式

2. **kubelet 与 API Server 的连接复用**：kubelet 默认使用 HTTP/2 连接 API Server，连接会被复用。确保 `--kube-api-qps` 和 `--kube-api-burst` 参数配置合理（默认 5/10，生产环境建议 50/100）

3. **减少 API Server 请求**：使用 Informer 机制替代直接 List-Watch，减少 API Server 压力。对于大规模集群，考虑使用分片（Sharding）机制

4. **优化 etcd 性能**：虽然 EKS 托管 etcd，但可以通过减少不必要的 Controller、优化 CRD 数量等方式减轻 etcd 压力

### Q2：如何处理 EKS 集群升级过程中的
---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - AWS
tag:
  - AWS
  - EKS
  - Kubernetes
  - DevOps
---

# AWS EKS 生产环境管理实践

当你的团队决定将核心业务迁移到 Kubernetes 时,第一个问题往往是:自建集群还是使用托管服务?如果选择 AWS,那么 EKS(Elastic Kubernetes Service)几乎是必然的选择。但 EKS 并不是"开箱即用"的——控制平面托管了,但数据平面、网络、安全、成本优化都需要深入理解才能在生产环境稳定运行。

本文将从架构设计、节点管理、网络配置、安全加固、成本优化五个维度,系统梳理 EKS 生产环境的实践经验,帮助你避开那些文档中没有明说的坑。

## 一、EKS 架构深度剖析

### 控制平面与数据平面的分离

EKS 的核心设计理念是"托管控制平面,自管数据平面"。这意味着 Kubernetes 的核心组件(API Server、etcd、Controller Manager、Scheduler)由 AWS 完全托管,用户无需关心高可用、备份、升级等运维负担。而工作节点(Node)则由用户自己管理(或使用托管节点组)。

```
┌─────────────────────────────────────────────────────────────┐
│                    AWS 托管的控制平面                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ API Server   │  │     etcd     │  │ Controller   │     │
│  │  (多AZ HA)   │  │  (多AZ HA)   │  │   Manager    │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
│                                                              │
│  ┌──────────────┐                                          │
│  │  Scheduler   │                                          │
│  └──────────────┘                                          │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   │ 通过 VPC Endpoint 访问
                   │
┌──────────────────▼──────────────────────────────────────────┐
│                    用户 VPC 中的数据平面                       │
│  ┌──────────────────┐  ┌──────────────────┐                │
│  │  Worker Node 1   │  │  Worker Node 2   │  ...           │
│  │  - kubelet       │  │  - kubelet       │                │
│  │  - kube-proxy    │  │  - kube-proxy    │                │
│  │  - Container     │  │  - Container     │                │
│  │    Runtime       │  │    Runtime       │                │
│  └──────────────────┘  └──────────────────┘                │
└─────────────────────────────────────────────────────────────┘
```

这种分离架构带来几个关键影响:

**网络访问模式**:API Server 通过 VPC Endpoint 暴露,工作节点通过私有网络访问,无需公网 IP。这要求 VPC 网络规划必须合理,避免跨 AZ 访问带来的延迟和成本。

**认证集成**:EKS 原生集成 IAM,通过 `aws eks get-token` 命令获取临时凭证,kubectl 配置无需长期密钥。这是 EKS 相比自建集群的一大优势。

**版本升级**:控制平面升级由 AWS 一键完成,但节点升级需要用户主动触发或使用托管节点组的自动升级功能。

### EKS 的网络模型

EKS 使用 AWS VPC CNI 作为默认网络插件,这是理解 EKS 网络的关键。VPC CNI 的核心机制是**为每个 Pod 分配一个 VPC IP 地址**。

```
┌─────────────────────────────────────────────────────────────┐
│                        VPC (10.0.0.0/16)                     │
│                                                              │
│  ┌────────────────────────────────────────────────────┐     │
│  │           Subnet (10.0.1.0/24) - AZ-a              │     │
│  │                                                      │     │
│  │  ┌──────────────────────────────────────┐          │     │
│  │  │         Worker Node                   │          │     │
│  │  │  Primary ENI: 10.0.1.10               │          │     │
│  │  │  Secondary ENI 1: 10.0.1.11, 10.0.1.12│          │     │
│  │  │  Secondary ENI 2: 10.0.1.13, 10.0.1.14│          │     │
│  │  │                                       │          │     │
│  │  │  Pod 1: 10.0.1.11 (ENI secondary IP)  │          │     │
│  │  │  Pod 2: 10.0.1.12 (ENI secondary IP)  │          │     │
│  │  │  Pod 3: 10.0.1.13 (ENI secondary IP)  │          │     │
│  │  └──────────────────────────────────────┘          │     │
│  └────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

**工作原理**:

1. 节点启动时,分配一个主 ENI(Elastic Network Interface)
2. VPC CNI 的 IPAMD 组件根据节点实例类型,预先附加多个 Secondary ENI
3. 每个 ENI 可以配置多个 Secondary IP(数量取决于实例类型)
4. Pod 创建时,从预分配的 IP 池中分配一个 IP,挂载到 Pod 的 network namespace

**关键配置参数**:

```yaml
# aws-node DaemonSet 环境变量
env:
  - name: WARM_ENI_TARGET
    value: "1"              # 预留 1 个完整 ENI 的 IP 数量
  - name: WARM_IP_TARGET
    value: "5"              # 或预留 5 个 IP 地址
  - name: MINIMUM_IP_TARGET
    value: "10"             # 最少预留 10 个 IP
  - name: MAX_IP_PER_ENI
    value: "29"             # 每个 ENI 最多 29 个 IP(c5.large 示例)
```

`WARM_IP_TARGET` 和 `WARM_ENI_TARGET` 的选择直接影响 Pod 启动速度:预留太少,批量创建 Pod 时需要等待 ENI 附加;预留太多,浪费 IP 地址。

### IP 地址耗尽问题

VPC CNI 的最大痛点是 IP 地址消耗快。一个 /24 子网只有 251 个可用 IP(前 4 个和最后 1 个保留),如果每个节点运行 30 个 Pod,8 个节点就会耗尽子网 IP。

**解决方案**:

**方案一:自定义网络模式(Custom Networking)**

将 Pod 网络与节点网络分离,Pod 使用独立的子网:

```yaml
# aws-node ConfigMap
data:
  eniConfig: |
    {
      "cni-custom-networking": "enabled"
    }

# ENIConfig 自定义资源
apiVersion: crd.k8s.amazonaws.com/v1alpha1
kind: ENIConfig
metadata:
  name: us-west-2a
spec:
  subnet: subnet-0a1b2c3d4e5f6g7h8  # Pod 专用子网
  securityGroups:
    - sg-0a1b2c3d4e5f6g7h8
```

**方案二:前缀委托模式(Prefix Delegation)**

EKS 1.18+ 支持,为每个 ENI 分配一个 /28 前缀(16 个 IP),而不是单个 IP:

```bash
# 启用前缀委托
kubectl set env daemonset aws-node -n kube-system ENABLE_PREFIX_DELEGATION=true
```

一个 c5.large 实例,默认模式下最多 29 个 Pod,启用前缀委托后可达 110 个 Pod。

## 二、节点组管理实践

### Managed Node Group vs Self-Managed Node Group

EKS 提供两种节点管理方式:

| 维度 | Managed Node Group | Self-Managed Node Group |
|------|-------------------|------------------------|
| 创建方式 | EKS Console / eksctl | 用户自己管理 ASG |
| 节点升级 | 一键滚动更新 | 手动更新 ASG Launch Template |
| 节点修复 | 自动替换不健康节点 | 需自行实现 |
| Spot 实例 | 支持,但终止处理有限 | 完全自定义 |
| 自定义 AMI | 支持 | 支持 |
| 运维负担 | 低 | 高 |

**生产环境推荐**:优先使用 Managed Node Group,除非有特殊需求(如自定义 kubelet 参数、特殊 AMI 构建流程)。

### eksctl 集群配置示例

```yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig

metadata:
  name: production-cluster
  region: us-west-2
  version: "1.28"

vpc:
  id: vpc-0a1b2c3d4e5f6g7h8
  subnets:
    private:
      us-west-2a:
        id: subnet-0a1b2c3d4e5f6g7h8
      us-west-2b:
        id: subnet-1a2b3c4d5e6f7g8h9
      us-west-2c:
        id: subnet-2a3b4c5d6e7f8g9h0

managedNodeGroups:
  - name: core-services
    instanceType: m6i.2xlarge
    desiredCapacity: 3
    minSize: 3
    maxSize: 10
    privateNetworking: true
    labels:
      workload: core
    taints:
      - key: dedicated
        value: core-services
        effect: NoSchedule
    iam:
      withAddonPolicies:
        autoScaler: true
        ebs: true
        efs: true
    tags:
      Environment: production
      NodeType: core

  - name: workload-spot
    instanceType: mixed
    desiredCapacity: 5
    minSize: 2
    maxSize: 20
    privateNetworking: true
    instancesDistribution:
      instanceTypes:
        - m6i.xlarge
        - m6a.xlarge
        - m5.xlarge
      onDemandBaseCapacity: 0
      onDemandPercentageAboveBaseCapacity: 20
      spotInstancePools: 3
    labels:
      workload: spot
      intent: apps
    iam:
      withAddonPolicies:
        autoScaler: true
    tags:
      Environment: production
      NodeType: spot
```

### 节点自动伸缩:Cluster Autoscaler vs Karpenter

**Cluster Autoscaler**:

传统的自动伸缩方案,通过监听 Pending Pod 触发 ASG 扩容:

```yaml
# Cluster Autoscaler 部署配置
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cluster-autoscaler
  namespace: kube-system
spec:
  template:
    spec:
      containers:
        - name: cluster-autoscaler
          image: k8s.gcr.io/autoscaling/cluster-autoscaler:v1.28.0
          command:
            - ./cluster-autoscaler
            - --v=4
            - --stderrthreshold=info
            - --cloud-provider=aws
            - --skip-nodes-with-local-storage=false
            - --expander=least-waste
            - --node-group-auto-discovery=asg:tag=k8s.io/cluster-autoscaler/enabled,k8s.io/cluster-autoscaler/production-cluster
          env:
            - name: AWS_REGION
              value: us-west-2
```

**Karpenter**:

AWS 推出的新一代自动伸缩工具,直接调用 EC2 API,无需 ASG:

```yaml
# Karpenter Provisioner 配置
apiVersion: karpenter.sh/v1beta1
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
  limits:
    resources:
      cpu: 1000
      memory: 1000Gi
  provider:
    subnetSelector:
      karpenter.sh/discovery: production-cluster
    securityGroupSelector:
      karpenter.sh/discovery: production-cluster
  consolidation:
    enabled: true
```

**Karpenter 的优势**:

1. **启动速度快**:无需等待 ASG 的健康检查周期,直接创建 EC2 实例
2. **灵活的实例选择**:根据 Pod 资源需求自动选择最优实例类型
3. **Spot 实例处理**:自动处理 Spot 中断,提前 2 分钟收到通知后优雅迁移
4. **资源利用率高**:Consolidation 功能自动优化节点资源

**生产建议**:

- 新集群优先使用 Karpenter
- 已有 Cluster Autoscaler 的集群可逐步迁移
- 关键业务节点组保留 Cluster Autoscaler 作为备份

### Spot 实例最佳实践

Spot 实例可以节省 60-90% 成本,但需要正确处理中断:

**1. 使用 Spot Instance Advisor 选择稳定的实例类型**

```bash
# 查看实例中断率
aws ec2 describe-spot-price-history \
  --instance-types m5.xlarge c5.xlarge r5.xlarge \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S)
```

**2. 配置 Pod Disruption Budget**

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: app-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: my-app
```

**3. 使用 AWS Node Termination Handler**

```yaml
# DaemonSet 监听 Spot 中断信号
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
          image: public.ecr.aws/aws-node-termination-handler/aws-node-termination-handler:v1.20.0
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

## 三、网络配置与安全加固

### 安全组设计原则

EKS 集群涉及多个安全组,需要合理规划:

```
┌─────────────────────────────────────────────────────────────┐
│                     集群安全组设计                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Cluster Security Group (eks-cluster-sg)             │  │
│  │  - 控制平面与节点间通信                                │  │
│  │  - Inbound: 443 (API Server) from VPC                │  │
│  │  - Outbound: All traffic                             │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Node Security Group (eks-node-sg)                   │  │
│  │  - 节点间通信                                         │  │
│  │  - Inbound: 1025-65535 from Cluster SG               │  │
│  │  - Inbound: 443 from Cluster SG (API Server)         │  │
│  │  - Inbound: Custom from ALB/NLB                      │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Pod Security Group (Security Group for Pods)        │  │
│  │  - 细粒度 Pod 级别网络隔离                             │  │
│  │  - 每个应用可使用独立安全组                            │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**最小权限配置示例**:

```bash
# 创建节点安全组
NODE_SG=$(aws ec2 create-security-group \
  --group-name eks-nodes-sg \
  --description "Security group for EKS worker nodes" \
  --vpc-id vpc-0a1b2c3d4e5f6g7h8 \
  --query 'GroupId' --output text)

# 允许节点与控制平面通信
aws ec2 authorize-security-group-ingress \
  --group-id $NODE_SG \
  --protocol tcp \
  --port 443 \
  --source-group $CLUSTER_SG

# 允许节点间通信
aws ec2 authorize-security-group-ingress \
  --group-id $NODE_SG \
  --protocol tcp \
  --port 1025-65535 \
  --source-group $NODE_SG
```

### Security Group for Pods

EKS 1.18+ 支持为 Pod 分配独立的安全组,实现细粒度网络隔离:

```yaml
# 创建 SecurityGroupPolicy
apiVersion: vpcresources.k8s.aws/v1beta1
kind: SecurityGroupPolicy
metadata:
  name: database-access
  namespace: production
spec:
  pods:
    matchLabels:
      access: database
  securityGroups:
    groupIds:
      - sg-0a1b2c3d4e5f6g7h8  # 允许访问 RDS 的安全组
```

**工作原理**:

1. Pod 创建时,VPC CNI 检查是否匹配 SecurityGroupPolicy
2. 如果匹配,为 Pod 创建独立的 ENI
3. 将指定的安全组附加到该 ENI
4. Pod 的网络流量通过独立 ENI,受独立安全组规则约束

**注意事项**:

- 启用 Security Group for Pods 后,Pod 启动时间会增加 2-5 秒(需要创建 ENI)
- 每个 Pod 独立 ENI 会消耗更多 IP 地址
- 建议仅对有特殊网络隔离需求的 Pod 启用

### IAM 权限管理:IRSA

EKS 的 IRSA(IAM Roles for Service Accounts)机制允许 Pod 使用独立的 IAM 角色,无需节点级别的权限:

**配置流程**:

```bash
# 1. 创建 OIDC Provider
eksctl utils associate-iam-oidc-provider \
  --cluster production-cluster \
  --approve

# 2. 创建 IAM 角色
cat > trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::ACCOUNT_ID:oidc-provider/OIDC_PROVIDER"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "OIDC_PROVIDER:sub": "system:serviceaccount:production:my-app-sa"
        }
      }
    }
  ]
}
EOF

aws iam create-role \
  --role-name my-app-role \
  --assume-role-policy-document file://trust-policy.json

# 3. 关联 ServiceAccount
kubectl annotate serviceaccount my-app-sa \
  -n production \
  eks.amazonaws.com/role-arn=arn:aws:iam::ACCOUNT_ID:role/my-app-role
```

**Pod 内使用**:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-app-sa
  namespace: production
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT_ID:role/my-app-role
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      serviceAccountName: my-app-sa
      containers:
        - name: app
          image: my-app:latest
          # AWS SDK 自动使用 IRSA 凭证
```

**权限边界控制**:

为避免权限过大,建议使用 Permission Boundary:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject"
      ],
      "Resource": "arn:aws:s3:::my-app-bucket/*"
    },
    {
      "Effect": "Deny",
      "Action": [
        "s3:DeleteBucket",
        "s3:DeleteObject"
      ],
      "Resource": "*"
    }
  ]
}
```

### Network Policy 实践

EKS 默认未启用 Network Policy,需要安装 Calico 或使用 AWS VPC CNI 的 Network Policy 支持(EKS 1.25+):

```yaml
# 安装 Calico
kubectl apply -f https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/master/config/master/calico-operator.yaml

# Network Policy 示例:限制 Pod 只能访问特定服务
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
```

## 四、性能调优实践

### kubelet 参数调优

EKS 节点的 kubelet 参数可以通过 UserData 或 Managed Node Group 的 Launch Template 自定义:

```bash
#!/bin/bash
set -o xtrace
/etc/eks/bootstrap.sh ${CLUSTER_NAME} \
  --kubelet-extra-args '--max-pods=110 \
    --eviction-hard=memory.available<500Mi,nodefs.available<10% \
    --eviction-soft=memory.available<1Gi,nodefs.available<15% \
    --eviction-soft-grace-period=memory.available=1m30s,nodefs.available=1m30s \
    --system-reserved=cpu=500m,memory=1Gi,ephemeral-storage=1Gi \
    --kube-reserved=cpu=500m,memory=1Gi,ephemeral-storage=1Gi'
```

**关键参数说明**:

- `--max-pods`: 最大 Pod 数量,默认取决于实例类型,启用前缀委托后可提升至 110
- `--eviction-hard`: 硬驱逐阈值,达到立即驱逐 Pod
- `--eviction-soft`: 软驱逐阈值,达到后等待 grace period 再驱逐
- `--system-reserved`: 为系统进程预留资源
- `--kube-reserved`: 为 Kubernetes 组件预留资源

### 内核参数优化

```bash
# /etc/sysctl.d/99-kubernetes.conf
net.core.somaxconn = 32768
net.ipv4.tcp_max_syn_backlog = 32768
net.core.netdev_max_backlog = 32768
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_keepalive_time = 300
net.ipv4.tcp_keepalive_probes = 5
net.ipv4.tcp_keepalive_intvl = 15
vm.max_map_count = 262144
fs.file-max = 2097152
fs.nr_open = 2097152
```

应用配置:

```bash
sysctl -p /etc/sysctl.d/99-kubernetes.conf
```

### 容器运行时优化

EKS 1.24+ 默认使用 containerd,可通过配置文件优化:

```toml
# /etc/containerd/config.toml
version = 2

[metrics]
address = "0.0.0.0:1338"

[plugins."io.containerd.grpc.v1.cri"]
  max_container_log_line_size = 16384

[plugins."io.containerd.grpc.v1.cri".containerd]
  snapshotter = "overlayfs"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
  runtime_type = "io.containerd.runc.v2"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
  SystemdCgroup = true
```

### 资源请求与限制的最佳实践

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-deployment
spec:
  template:
    spec:
      containers:
        - name: app
          image: my-app:latest
          resources:
            requests:
              cpu: "500m"
              memory: "512Mi"
            limits:
              cpu: "2000m"
              memory: "2Gi"
```

**生产建议**:

1. **必须设置 requests**:用于调度决策,确保节点有足够资源
2. **谨慎设置 limits**:CPU limit 会导致 CFS Quota 限流,内存 limit 超过会被 OOM Kill
3. **使用 LimitRange 设置默认值**:避免未设置资源的 Pod 影响集群稳定性

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
  namespace: production
spec:
  limits:
    - default:
        cpu: "1"
        memory: "512Mi"
      defaultRequest:
        cpu: "100m"
        memory: "128Mi"
      type: Container
```

## 五、成本优化策略

### 节点成本分析

使用 Kubecost 或 AWS Cost Explorer 分析集群成本:

```bash
# 安装 Kubecost
kubectl apply -f https://raw.githubusercontent.com/kubecost/cost-analyzer-helm-chart/master/kubecost.yaml

# 查看成本报告
kubectl port-forward --namespace kubecost deployment/kubecost-cost-analyzer 9090
```

**成本构成**:

| 成本项 | 占比 | 优化方向 |
|--------|------|---------|
| EC2 实例 | 60-70% | Spot 实例、Right-Sizing |
| EBS 卷 | 15-20% | 存储类型选择、快照管理 |
| 数据传输 | 5-10% | AZ 亲和性、VPC Endpoint |
| 负载均衡器 | 5-10% | 合并 Ingress、使用 NLB |

### Spot 实例策略

**混合实例策略**:

```yaml
managedNodeGroups:
  - name: mixed-instances
    instanceType: mixed
    instancesDistribution:
      instanceTypes:
        - m6i.xlarge
        - m6a.xlarge
        - m5.xlarge
      onDemandBaseCapacity: 2
      onDemandPercentageAboveBaseCapacity: 20
      spotInstancePools: 3
      spotAllocationStrategy: capacity-optimized
```

**关键参数**:

- `onDemandBaseCapacity`: 基础 On-Demand 节点数,保证核心服务稳定性
- `onDemandPercentageAboveBaseCapacity`: 超出基础容量后的 On-Demand 比例
- `spotInstancePools`: Spot 实例池数量,增加多样性降低中断风险
- `spotAllocationStrategy`: 分配策略,`capacity-optimized` 优先选择容量充足的实例类型

### 存储成本优化

**EBS 卷类型选择**:

| 类型 | IOPS | 吞吐量 | 成本 | 适用场景 |
|------|------|--------|------|---------|
| gp3 | 3,000-16,000 | 125-1000 MB/s | 低 | 通用工作负载 |
| io2 Block Express | 256,000 | 4000 MB/s | 高 | 数据库 |
| st1 | - | 500 MB/s | 低 | 日志、大数据 |

**动态存储配置**:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: gp3
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  encrypted: "true"
  kmsKeyId: "arn:aws:kms:us-west-2:ACCOUNT_ID:key/KEY_ID"
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

### 集群自动伸缩优化

**防止过度扩容**:

```yaml
# Cluster Autoscaler 配置
spec:
  template:
    spec:
      containers:
        - name: cluster-autoscaler
          command:
            - ./cluster-autoscaler
            - --scale-down-unneeded-time=5m
            - --scale-down-delay-after-add=10m
            - --scale-down-delay-after-failure=3m
            - --scale-down-delay-after-delete=5m
            - --balance-similar-node-groups
            - --skip-nodes-with-system-pods=false
```

**Karpenter Consolidation**:

```yaml
apiVersion: karpenter.sh/v1beta1
kind: Provisioner
spec:
  consolidation:
    enabled: true
```

Consolidation 会持续分析节点利用率,自动删除低利用率节点或替换为更小规格实例。

## 六、监控与故障排查

### 控制平面日志

启用 EKS 控制平面日志:

```bash
aws eks update-cluster-config \
  --name production-cluster \
  --logging '{"clusterLogging":[{"types":["api","audit","authenticator","controllerManager","scheduler"],"enabled":true}]}'
```

日志发送到 CloudWatch Logs,可配置日志组保留策略和订阅过滤器。

### 节点问题检测

部署 Node Problem Detector:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-problem-detector
  namespace: kube-system
spec:
  template:
    spec:
      hostNetwork: true
      containers:
        - name: node-problem-detector
          image: k8s.gcr.io/node-problem-detector:v0.8.12
          securityContext:
            privileged: true
          volumeMounts:
            - name: log
              mountPath: /var/log
            - name: kmsg
              mountPath: /dev/kmsg
      volumes:
        - name: log
          hostPath:
            path: /var/log
        - name: kmsg
          hostPath:
            path: /dev/kmsg
```

### 常见故障排查

**问题 1:节点 NotReady**

排查步骤:

```bash
# 1. 检查节点状态
kubectl describe node <node-name>

# 2. 检查 kubelet 日志
journalctl -u kubelet -f

# 3. 检查节点资源
kubectl top node

# 4. 检查 VPC CNI 日志
kubectl logs -n kube-system aws-node-<node-name>

# 5. 检查安全组规则
aws ec2 describe-security-groups --group-ids <sg-id>
```

**问题 2:Pod 一直 Pending**

```bash
# 1. 查看 Pod 事件
kubectl describe pod <pod-name>

# 2. 检查资源请求
kubectl get pod <pod-name> -o yaml | grep -A 5 resources

# 3. 检查节点资源
kubectl describe nodes | grep -A 5 "Allocated resources"

# 4. 检查污点和容忍
kubectl describe node <node-name> | grep -A 5 Taints
kubectl get pod <pod-name> -o yaml | grep -A 5 tolerations

# 5. 检查存储挂载
kubectl get pv,pvc
```

**问题 3:网络连接问题**

```bash
# 1. 检查 VPC CNI 状态
kubectl get pods -n kube-system -l k8s-app=aws-node

# 2. 检查 IP 地址分配
kubectl exec -n kube-system aws-node-<node-name> -- ip addr show

# 3. 检查安全组规则
aws ec2 describe-security-groups --group-ids <sg-id>

# 4. 测试 Pod 间连通性
kubectl run test-pod --image=busybox --rm -it --restart=Never -- ping <target-pod-ip>

# 5. 检查 DNS 解析
kubectl run test-pod --image=busybox --rm -it --restart=Never -- nslookup kubernetes.default
```

**问题 4:IRSA 不生效**

```bash
# 1. 检查 OIDC Provider
aws iam list-open-id-connect-providers

# 2. 检查 ServiceAccount 注解
kubectl get sa <sa-name> -n <namespace> -o yaml

# 3. 检查 IAM 角色信任策略
aws iam get-role --role-name <role-name> --query 'Role.AssumeRolePolicyDocument'

# 4. 检查 Pod 环境变量
kubectl exec <pod-name> -- env | grep AWS

# 5. 检查 Webhook 配置
kubectl get mutatingwebhookconfigurations -o yaml | grep -A 10 eks.amazonaws.com
```

## 小结

- **架构理解**:EKS 的控制平面托管与数据平面自管模式,要求深入理解 VPC CNI 的 IP 分配机制,合理规划子网和 IP 地址池
- **节点管理**:优先使用 Managed Node Group,Karpenter 相比 Cluster Autoscaler 提供更快的启动速度和更高的资源利用率,Spot 实例需要配合中断处理机制
- **网络与安全**:Security Group for Pods 实现细粒度网络隔离,IRSA 提供最小权限的 IAM 管理,Network Policy 是零信任网络的基础
- **性能调优**:kubelet 参数、内核参数、容器运行时配置需要根据实际负载调整,资源请求与限制的合理设置是集群稳定性的保障
- **成本优化**:Spot 实例可节省 60-90% 成本,但需要正确处理中断;存储类型选择和自动伸缩优化是降低成本的持续工作
- **监控排查**:控制平面日志、节点问题检测、系统化排查流程是快速定位问题的关键

---

## 常见问题

### Q1:EKS 控制平面升级会影响业务吗?

EKS 控制平面升级采用原地升级策略,API Server 在升级过程中会短暂不可用(通常几秒到几分钟),但已运行的 Pod 不受影响。建议:

1. **选择业务低峰期升级**
2. **提前测试新版本兼容性**:在测试环境验证应用在新版本的运行情况
3. **检查废弃 API**:使用 `kubent` 或 `pluto` 工具检查是否使用了已废弃的 API
4. **节点升级**:控制平面升级后,节点 kubelet 版本需要跟进,使用托管节点组的自动升级功能或手动触发滚动更新

### Q2:如何处理 EKS 集群升级过程中的兼容性问题?

升级前必须检查:

```bash
# 1. 检查废弃 API
kubectl get --all-namespaces -o json pvc,deploy,ds,sts,rs,rc,svc | \
  jq '.items[] | select(.apiVersion | contains("extensions/v1beta1")) | .kind + "/" + .metadata.name'

# 2. 检查 Helm Chart 兼容性
helm list --all-namespaces

# 3. 检查自定义资源定义(CRD)
kubectl get crd -o yaml | grep -A 5 "version"

# 4. 使用 kubent 工具
kubent -c ~/.kube/config
```

升级过程中:

1. **分批升级节点**:避免一次性升级所有节点导致服务中断
2. **监控应用日志**:关注升级过程中的错误日志
3. **准备回滚方案**:保留旧版本的 AMI 和配置,必要时快速回滚

### Q3:VPC CNI 的 IP 地址耗尽问题如何彻底解决?

综合方案:

1. **启用前缀委托**:将 Pod 密度从 29 提升到 110
2. **使用自定义网络**:Pod 使用独立子网,与节点网络分离
3. **合理规划子网**:根据集群规模预留足够的 IP 地址空间,建议 /16 VPC + 多个 /20 子网
4. **监控 IP 使用率**:

```bash
# 安装 CNI Metrics Helper
kubectl apply -f https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/master/config/v1.9/cni-metrics-helper.yaml

# 查看指标
kubectl logs -n kube-system cni-metrics-helper
```

### Q4:如何实现 EKS 集群的跨区域灾备?

EKS 本身不支持跨区域灾备,但可以通过以下方案实现:

**方案一:多集群 + GitOps**

```
┌─────────────────────────────────────────────────────────────┐
│                      GitOps Repository                       │
│                   (应用配置单一事实来源)                       │
└──────────────────┬──────────────────────────────────────────┘
                   │
        ┌──────────┴──────────┐
        │                     │
        ▼                     ▼
┌───────────────┐     ┌───────────────┐
│  EKS Cluster  │     │  EKS Cluster  │
│  Region A     │     │  Region B     │
│  (Active)     │     │  (Standby)    │
└───────────────┘     └───────────────┘
```

使用 ArgoCD 或 FluxCD 同步应用配置到两个集群,Route53 配置健康检查实现自动切换。

**方案二:数据层跨区域复制**

- RDS:跨区域只读副本
- S3:跨区域复制(CRR)
- DynamoDB:全局表(Global Table)

**方案三:使用 Velero 备份恢复**

```bash
# 定期备份
velero backup create daily-backup --schedule="0 2 * * *"

# 跨区域恢复
velero restore create --from-backup daily-backup
```

### Q5:EKS 的控制平面费用($0.10/小时)是否可以优化?

EKS 控制平面费用是固定成本,无法优化,但可以通过以下方式降低整体成本:

1. **合并集群**:将多个小集群合并为一个大集群,减少控制平面数量
2. **使用 EKS Anywhere**:在自有数据中心部署 EKS,无控制平面费用(但有运维成本)
3. **利用 Savings Plans**:对 EC2 实例和 EKS 控制平面费用承诺使用量,最高可节省 72%
4. **开发环境使用 Fargate**:无节点管理成本,按实际使用付费

## 参考资源

- [EKS 官方文档](https://docs.aws.amazon.com/eks/)
- [EKS 最佳实践指南](https://aws.github.io/aws-eks-best-practices/)
- [VPC CNI 深度解析](https://github.com/aws/amazon-vpc-cni-k8s)
- [Karpenter 官方文档](https://karpenter.sh/)
- [EKS 成本优化指南](https://aws.amazon.com/blogs/containers/best-practices-for-cost-optimization-on-amazon-eks/)

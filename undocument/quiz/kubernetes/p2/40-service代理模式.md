---
date: 2026-03-12
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Service
  - kube-proxy
---

# Kubernetes Service 代理模式深度解析

## 引言：为什么需要 Service 代理模式？

在 Kubernetes 集群中，Pod 是动态的、易变的。它们可能因为节点故障、扩缩容或滚动更新而被创建或销毁，每次创建都会分配新的 IP 地址。这就带来了一个核心问题：**如何为这些动态变化的 Pod 提供稳定的访问入口？**

Kubernetes Service 通过标签选择器（Label Selector）将一组 Pod 抽象为一个统一的服务端点，提供稳定的 ClusterIP 或外部访问入口。而 **kube-proxy** 正是实现 Service 负载均衡和网络代理的关键组件，它运行在每个节点上，负责将访问 Service 的流量转发到后端的 Pod。

kube-proxy 支持三种代理模式，每种模式在实现原理、性能特征和适用场景上都有显著差异。理解这些代理模式的工作机制，对于优化集群网络性能、排查服务访问问题至关重要。

## kube-proxy 的核心职责

在深入代理模式之前，我们需要明确 kube-proxy 的核心职责：

1. **服务发现**：监听 API Server 中 Service 和 Endpoints 对象的变化
2. **规则同步**：根据 Service 定义，在节点上配置流量转发规则
3. **负载均衡**：将访问 Service 的流量均匀分发到后端 Pod
4. **会话保持**：支持 ClientIP 会话亲和性配置

```
┌─────────────────────────────────────────────────────────┐
│                    Kubernetes Master                     │
│  ┌──────────────┐         ┌──────────────┐             │
│  │   Service    │         │  Endpoints   │             │
│  │  10.96.0.1   │────────▶│  Pod1, Pod2  │             │
│  └──────────────┘         └──────────────┘             │
└─────────────────────────────────────────────────────────┘
                          │
                          │ Watch API Server
                          ▼
┌─────────────────────────────────────────────────────────┐
│                   Worker Node                            │
│  ┌──────────────┐                                       │
│  │  kube-proxy  │ ◀─── Sync Rules ───▶ iptables/ipvs   │
│  └──────────────┘                                       │
│         │                                               │
│         ▼                                               │
│  ┌──────────────────────────────────────────┐          │
│  │   Traffic Forwarding Rules                │          │
│  │   (iptables / ipvs / userspace)          │          │
│  └──────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────┘
```

## 一、iptables 模式（默认模式）

### 工作原理

iptables 模式是 Kubernetes v1.2 引入的代理模式，也是目前大多数集群的默认选择。其核心思想是利用 Linux 内核的 Netfilter 框架，通过 iptables 规则实现流量拦截和转发。

#### 流量转发链路

```
Client Request
     │
     ▼
┌─────────────────┐
│  PREROUTING     │ ← DNAT: ClusterIP → PodIP
└─────────────────┘
     │
     ▼
┌─────────────────┐
│  KUBE-SERVICES  │ ← Service 规则入口链
└─────────────────┘
     │
     ├─▶ KUBE-SVC-XXX (Service Chain)
     │       │
     │       ├─▶ KUBE-SEP-XXX1 (Pod1) ──▶ DNAT to Pod1:Port
     │       ├─▶ KUBE-SEP-XXX2 (Pod2) ──▶ DNAT to Pod2:Port
     │       └─▶ KUBE-SEP-XXX3 (Pod3) ──▶ DNAT to Pod3:Port
     │
     ▼
┌─────────────────┐
│  POSTROUTING    │ ← MASQUERADE (SNAT)
└─────────────────┘
     │
     ▼
  Backend Pod
```

#### 规则生成机制

当 kube-proxy 监听到 Service 或 Endpoints 变化时，会在宿主机上生成以下 iptables 链：

1. **KUBE-SERVICES**：所有 Service 规则的入口链
2. **KUBE-SVC-XXX**：每个 Service 对应的规则链（XXX 为 Service ID）
3. **KUBE-SEP-XXX**：每个 Endpoints 对应的规则链（SEP = Service Endpoint）

以一个 ClusterIP Service 为例，生成的 iptables 规则如下：

```bash
# KUBE-SERVICES 链（入口）
-A KUBE-SERVICES -d 10.96.0.1/32 -p tcp --dport 80 -j KUBE-SVC-NWV5X2332I4OTOF

# KUBE-SVC 链（负载均衡规则）
-A KUBE-SVC-NWV5X2332I4OTOF -m statistic --mode random --probability 0.3333 -j KUBE-SEP-WNBA2I3ZYZPVJ4K
-A KUBE-SVC-NWV5X2332I4OTOF -m statistic --mode random --probability 0.5000 -j KUBE-SEP-X3P262DAGQE6GS3
-A KUBE-SVC-NWV5X2332I4OTOF -j KUBE-SEP-57KPRZ3JQVEEN3F

# KUBE-SEP 链（Pod 端点规则）
-A KUBE-SEP-WNBA2I3ZYZPVJ4K -s 10.244.1.2/32 -j KUBE-MARK-MASQ
-A KUBE-SEP-WNBA2I3ZYZPVJ4K -p tcp -m tcp -j DNAT --to-destination 10.244.1.2:8080
```

### 负载均衡实现

iptables 模式使用 **statistic 模块** 的 random 模式实现概率性负载均衡：

- 第一个规则：1/3 概率选择 Pod1
- 第二个规则：1/2 概率选择 Pod2（剩余流量的 1/2，即总流量的 1/3）
- 第三个规则：剩余流量选择 Pod3（总流量的 1/3）

这种实现方式保证了每个 Pod 获得大致相等的流量，但并非严格轮询。

### 优缺点分析

#### 优点

1. **性能优异**：规则在内核空间处理，无需用户态和内核态切换
2. **成熟稳定**：iptables 是 Linux 标准组件，经过长期生产验证
3. **资源占用低**：kube-proxy 不参与流量转发，仅负责规则同步
4. **无需额外依赖**：所有 Linux 发行版默认支持

#### 缺点

1. **规则更新延迟**：Service 和 Pod 数量较多时，规则同步耗时较长
2. **扩展性瓶颈**：iptables 规则是线性链表，O(n) 时间复杂度
3. **难以调试**：规则数量庞大时，排查问题困难
4. **不支持会话保持优化**：每次连接都需要遍历规则链

### 配置方式

通过 kube-proxy 配置文件或命令行参数启用：

```yaml
# kube-proxy 配置文件
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "iptables"  # 或 "" 表示自动选择（默认 iptables）
```

或通过命令行参数：

```bash
kube-proxy --proxy-mode=iptables
```

## 二、ipvs 模式（推荐模式）

### 工作原理

ipvs（IP Virtual Server）模式是 Kubernetes v1.8 引入，v1.11 达到 GA 状态。ipvs 是 Linux 内核级别的四层负载均衡器，专门为高性能负载均衡场景设计。

#### 核心组件

ipvs 模式依赖以下内核模块：

- **ip_vs**：核心负载均衡模块
- **ip_vs_rr**：轮询调度算法
- **ip_vs_wrr**：加权轮询调度算法
- **ip_vs_sh**：源地址哈希调度算法
- **ip_vs_lc**：最少连接调度算法
- **nf_conntrack_ipv4**：连接跟踪模块

#### 流量转发链路

```
Client Request
     │
     ▼
┌─────────────────────────────────────┐
│        Netfilter HOOK               │
│    (NF_INET_LOCAL_IN)               │
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│        IPVS Service                 │
│   Virtual IP: 10.96.0.1:80         │
│   Scheduler: rr/wrr/sh/lc          │
└─────────────────────────────────────┘
     │
     ├─▶ Real Server 1: 10.244.1.2:8080
     ├─▶ Real Server 2: 10.244.1.3:8080
     └─▶ Real Server 3: 10.244.1.4:8080
     │
     ▼
┌─────────────────────────────────────┐
│    Connection Tracking              │
│    (nf_conntrack)                   │
└─────────────────────────────────────┘
     │
     ▼
  Backend Pod
```

#### 数据结构优势

ipvs 使用 **哈希表** 存储服务规则，查询时间复杂度为 O(1)，而 iptables 是 O(n)。这使得 ipvs 在大规模集群中性能优势明显。

```
iptables 规则查找：
Rule1 → Rule2 → Rule3 → ... → RuleN  [O(n)]

ipvs 规则查找：
Hash(VIP:Port) → Service → Real Servers  [O(1)]
```

### 负载均衡算法

ipvs 支持多种负载均衡算法，通过 Service 的 `sessionAffinity` 和 annotations 配置：

| 算法 | 说明 | 配置方式 |
|------|------|----------|
| **rr** (Round Robin) | 轮询，默认算法 | 默认 |
| **wrr** (Weighted RR) | 加权轮询 | 通过 annotations 配置 |
| **sh** (Source Hashing) | 源地址哈希，实现会话保持 | `sessionAffinity: ClientIP` |
| **lc** (Least Connections) | 最少连接 | 需要内核模块支持 |
| **wlc** (Weighted LC) | 加权最少连接 | 需要内核模块支持 |

#### 会话保持配置示例

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  clusterIP: 10.96.0.1
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800  # 3小时
  ports:
  - port: 80
    targetPort: 8080
```

### 优缺点分析

#### 优点

1. **高性能**：O(1) 查询复杂度，支持大规模集群
2. **丰富算法**：支持多种负载均衡算法
3. **健康检查**：支持后端服务器健康检查
4. **规则更新快**：增量更新，无需重建所有规则
5. **连接复用**：支持连接跟踪，提高性能

#### 缺点

1. **内核依赖**：需要加载 ip_vs 内核模块
2. **兼容性**：部分老旧内核版本支持不完善
3. **调试工具**：相比 iptables，调试工具较少

### 配置方式

#### 1. 确保内核模块加载

```bash
# 检查 ipvs 模块是否加载
lsmod | grep ip_vs

# 手动加载模块
modprobe ip_vs
modprobe ip_vs_rr
modprobe ip_vs_wrr
modprobe ip_vs_sh

# 设置开机自动加载
cat > /etc/modules-load.d/ipvs.conf <<EOF
ip_vs
ip_vs_rr
ip_vs_wrr
ip_vs_sh
EOF
```

#### 2. kube-proxy 配置

```yaml
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "ipvs"
ipvs:
  scheduler: "rr"  # 负载均衡算法
  syncPeriod: "30s"
  minSyncPeriod: "5s"
  excludeCIDRs: []
```

#### 3. kubeadm 集群配置

```yaml
# kubeadm-config.yaml
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: ipvs
```

#### 4. 验证 ipvs 模式

```bash
# 查看 ipvs 规则
ipvsadm -Ln

# 输出示例
IP Virtual Server version 1.2.1 (size=4096)
Prot LocalAddress:Port Scheduler Flags
  -> RemoteAddress:Port           Forward Weight ActiveConn InActConn
TCP  10.96.0.1:80 rr
  -> 10.244.1.2:8080              Masq    1      0          0
  -> 10.244.1.3:8080              Masq    1      0          0
  -> 10.244.1.4:8080              Masq    1      0          0
```

## 三、userspace 模式（已废弃）

### 工作原理

userspace 模式是 Kubernetes 最早期的代理模式，在 v1.0 引入，现已废弃。其核心思想是 kube-proxy 在用户态监听 Service 端口，接收请求后转发到后端 Pod。

#### 流量转发链路

```
Client Request
     │
     ▼
┌─────────────────────────────────────┐
│   iptables REDIRECT                 │
│   (重定向到 kube-proxy 端口)        │
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│   kube-proxy (User Space)           │
│   - 监听 Service 端口               │
│   - 选择后端 Pod                    │
│   - 建立到 Pod 的连接               │
└─────────────────────────────────────┘
     │
     ▼
  Backend Pod
```

### 优缺点分析

#### 优点

1. **实现简单**：逻辑清晰，易于理解和调试
2. **跨平台**：不依赖特定内核功能

#### 缺点

1. **性能极差**：用户态和内核态频繁切换，性能损耗严重
2. **资源占用高**：每个连接都需要 kube-proxy 处理
3. **单点瓶颈**：kube-proxy 成为性能瓶颈
4. **已废弃**：不再维护，不推荐使用

### 配置方式（仅作了解）

```yaml
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "userspace"
```

## 四、iptables vs ipvs 深度对比

### 性能对比

#### 测试环境

- 节点数量：100 节点
- Service 数量：10,000 个
- 每个 Service 后端：5 个 Pod
- 总规则数：50,000 条

#### 性能指标

| 指标 | iptables 模式 | ipvs 模式 | 提升比例 |
|------|--------------|-----------|---------|
| 规则同步时间 | 5-10 分钟 | 5-10 秒 | **60-120x** |
| 连接新建速率 | 30,000 cps | 100,000 cps | **3.3x** |
| CPU 占用率 | 15-20% | 3-5% | **4-5x** |
| 内存占用 | 200 MB | 50 MB | **4x** |
| 规则查询延迟 | O(n) | O(1) | **显著提升** |

#### 规则更新机制对比

**iptables 模式**：
```
Service 变化
    │
    ▼
全量重建规则链
    │
    ├─▶ 删除所有 KUBE-SVC 链
    ├─▶ 删除所有 KUBE-SEP 链
    ├─▶ 重新生成所有规则
    └─▶ iptables-restore 原子更新
    │
    ▼
耗时：O(n)，n 为 Service 数量
```

**ipvs 模式**：
```
Service 变化
    │
    ▼
增量更新规则
    │
    ├─▶ 计算差异（新增/删除/修改）
    ├─▶ 仅更新变化的 Service
    └─▶ ipvsadm 增量操作
    │
    ▼
耗时：O(1)，与 Service 总数无关
```

### 功能对比

| 功能特性 | iptables 模式 | ipvs 模式 |
|---------|--------------|-----------|
| 负载均衡算法 | 随机概率 | rr/wrr/sh/lc/wlc |
| 会话保持 | 支持（有限） | 支持（完善） |
| 规则更新 | 全量更新 | 增量更新 |
| 健康检查 | 不支持 | 支持 |
| 连接跟踪 | 依赖 conntrack | 内置优化 |
| 规则持久化 | iptables-save | ipvsadm-save |
| 调试工具 | iptables -L | ipvsadm -Ln |

### 适用场景对比

#### iptables 模式适用场景

1. **小型集群**：Service 数量 < 1000
2. **低频变更**：Service 和 Pod 变更不频繁
3. **兼容性优先**：老旧内核或特殊环境
4. **快速部署**：无需额外配置内核模块

#### ipvs 模式适用场景

1. **大型集群**：Service 数量 > 1000
2. **高频变更**：频繁扩缩容、滚动更新
3. **高性能要求**：高并发、低延迟场景
4. **复杂负载均衡**：需要加权轮询、最少连接等算法

## 五、负载均衡算法详解

### iptables 模式的随机算法

iptables 使用 statistic 模块的 random 模式，通过概率计算实现负载均衡：

```bash
# 3 个 Pod 的负载均衡规则
# Pod1: 1/3 概率
-A KUBE-SVC-XXX -m statistic --mode random --probability 0.3333 -j KUBE-SEP-1

# Pod2: 1/2 概率（剩余流量的 1/2，即总流量的 1/3）
-A KUBE-SVC-XXX -m statistic --mode random --probability 0.5000 -j KUBE-SEP-2

# Pod3: 剩余流量（总流量的 1/3）
-A KUBE-SVC-XXX -j KUBE-SEP-3
```

**概率计算公式**：

```
P(Pod_i) = 1 / (N - i + 1)

其中：
- N: Pod 总数
- i: 当前 Pod 索引（从 1 开始）
```

### ipvs 模式的调度算法

#### 1. 轮询（Round Robin, rr）

```
请求序列: 1, 2, 3, 4, 5, 6, ...
Pod 分配: Pod1, Pod2, Pod3, Pod1, Pod2, Pod3, ...
```

#### 2. 加权轮询（Weighted Round Robin, wrr）

```
权重配置:
- Pod1: weight=3
- Pod2: weight=2
- Pod3: weight=1

请求序列: 1, 2, 3, 4, 5, 6, ...
Pod 分配: Pod1, Pod1, Pod1, Pod2, Pod2, Pod3, Pod1, Pod1, Pod1, Pod2, Pod2, Pod3, ...
```

**配置方式**：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    kube-proxy.kubernetes.io/weight: "3"  # 仅部分实现支持
spec:
  # ...
```

#### 3. 源地址哈希（Source Hashing, sh）

```
Client IP: 192.168.1.10 → Hash → Pod2
Client IP: 192.168.1.20 → Hash → Pod1
Client IP: 192.168.1.10 → Hash → Pod2  # 相同客户端总是访问同一 Pod
```

**实现会话保持**：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: session-service
spec:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 3600  # 1小时
```

#### 4. 最少连接（Least Connections, lc）

```
当前连接数:
- Pod1: 10 connections
- Pod2: 5 connections
- Pod3: 8 connections

新请求 → 选择 Pod2（连接数最少）
```

### 算法选择建议

| 场景 | 推荐算法 | 原因 |
|------|---------|------|
| 无状态服务 | rr/wrr | 简单高效，均匀分布 |
| 有状态服务 | sh | 会话保持，避免状态丢失 |
| 长连接服务 | lc/wlc | 动态负载均衡，避免单点过载 |
| 异构后端 | wrr | 根据性能分配权重 |

## 六、模式选择决策树

```
开始
  │
  ├─▶ Service 数量 > 1000?
  │     ├─ Yes ─▶ ipvs 模式
  │     └─ No ─▶ 继续
  │
  ├─▶ 需要高级负载均衡算法?
  │     ├─ Yes ─▶ ipvs 模式
  │     └─ No ─▶ 继续
  │
  ├─▶ 内核版本 >= 4.19?
  │     ├─ Yes ─▶ ipvs 模式（推荐）
  │     └─ No ─▶ iptables 模式
  │
  └─▶ 特殊环境或兼容性要求?
        ├─ Yes ─▶ iptables 模式
        └─ No ─▶ ipvs 模式（推荐）
```

## 七、常见问题与最佳实践

### 常见问题

#### Q1: 如何查看当前 kube-proxy 使用的代理模式?

**A**: 查看 kube-proxy 日志或 ConfigMap：

```bash
# 方法1: 查看 kube-proxy 日志
kubectl logs -n kube-system kube-proxy-xxxxx | grep "Using iptables Proxier"

# 方法2: 查看 ConfigMap
kubectl get configmap kube-proxy -n kube-system -o yaml | grep mode

# 方法3: 查看节点规则
iptables -t nat -L KUBE-SERVICES  # iptables 模式
ipvsadm -Ln                        # ipvs 模式
```

#### Q2: 如何从 iptables 模式切换到 ipvs 模式?

**A**: 修改 kube-proxy ConfigMap 并重启：

```bash
# 1. 编辑 ConfigMap
kubectl edit configmap kube-proxy -n kube-system

# 2. 修改 mode 为 ipvs
mode: "ipvs"

# 3. 删除 kube-proxy Pod 触发重建
kubectl delete pod -n kube-system -l k8s-app=kube-proxy
```

#### Q3: ipvs 模式下 Service 无法访问怎么办?

**A**: 排查步骤：

```bash
# 1. 检查 ipvs 模块是否加载
lsmod | grep ip_vs

# 2. 检查 ipvs 规则
ipvsadm -Ln

# 3. 检查 kube-proxy 日志
kubectl logs -n kube-system kube-proxy-xxxxx

# 4. 检查 conntrack
conntrack -L | grep <ClusterIP>
```

#### Q4: 为什么 iptables 模式下规则同步很慢?

**A**: 原因分析：

1. **规则数量过多**：每个 Service 生成多条规则，大规模集群可达数万条
2. **全量更新**：任何变更都触发全量重建
3. **iptables-restore 延迟**：原子更新需要锁定规则表

**优化方案**：

- 切换到 ipvs 模式
- 减少不必要的 Service
- 使用 NodePort 或 Ingress 替代部分 ClusterIP

#### Q5: ipvs 模式支持哪些内核版本?

**A**: 内核版本要求：

- **最低要求**：Linux Kernel 3.10（CentOS 7）
- **推荐版本**：Linux Kernel 4.19+（完整功能支持）
- **最佳版本**：Linux Kernel 5.0+（性能优化）

### 最佳实践

#### 1. 生产环境推荐配置

```yaml
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "ipvs"
ipvs:
  scheduler: "rr"
  syncPeriod: "30s"
  minSyncPeriod: "5s"
  excludeCIDRs: []
conntrack:
  maxPerCore: 32768
  min: 131072
  tcpCloseWaitTimeout: "1h"
  tcpEstablishedTimeout: "24h"
```

#### 2. 性能调优参数

```bash
# 增加连接跟踪表大小
echo "net.netfilter.nf_conntrack_max=1048576" >> /etc/sysctl.conf

# 优化哈希表大小
echo "net.netfilter.nf_conntrack_buckets=262144" >> /etc/sysctl.conf

# 应用配置
sysctl -p
```

#### 3. 监控指标

kube-proxy 暴露 Prometheus 指标：

```bash
# 访问指标端点
curl http://<node-ip>:10249/metrics

# 关键指标
- kubeproxy_sync_proxy_rules_duration_seconds: 规则同步耗时
- kubeproxy_sync_proxy_rules_iptables_total: iptables 规则数量
- kubeproxy_network_programming_duration_seconds: 网络编程耗时
```

#### 4. 故障排查流程

```
Service 无法访问
    │
    ├─▶ 1. 检查 Service 和 Endpoints
    │     kubectl get svc,eps
    │
    ├─▶ 2. 检查 kube-proxy 状态
    │     kubectl get pod -n kube-system -l k8s-app=kube-proxy
    │
    ├─▶ 3. 检查代理规则
    │     iptables -t nat -L KUBE-SERVICES
    │     ipvsadm -Ln
    │
    ├─▶ 4. 检查 Pod 网络连通性
    │     kubectl exec -it <pod> -- ping <backend-pod-ip>
    │
    └─▶ 5. 检查 kube-proxy 日志
          kubectl logs -n kube-system kube-proxy-xxxxx
```

## 八、总结与对比表格

### 核心特性对比

| 特性 | userspace | iptables | ipvs |
|------|-----------|----------|------|
| **引入版本** | v1.0 | v1.2 | v1.8 (GA v1.11) |
| **状态** | 已废弃 | 默认模式 | 推荐模式 |
| **性能** | 差 | 良 | 优 |
| **规则查询** | O(n) | O(n) | O(1) |
| **负载均衡算法** | 轮询 | 随机概率 | rr/wrr/sh/lc/wlc |
| **规则更新** | 增量 | 全量 | 增量 |
| **内核依赖** | 低 | 中 | 高 |
| **调试难度** | 低 | 高 | 中 |
| **适用规模** | 小型 | 中小型 | 中大型 |

### 性能对比（10,000 Service 场景）

| 指标 | iptables | ipvs | 差异 |
|------|----------|------|------|
| 规则同步时间 | 5-10 分钟 | 5-10 秒 | **60-120x** |
| 连接新建速率 | 30,000 cps | 100,000 cps | **3.3x** |
| CPU 占用 | 15-20% | 3-5% | **4-5x** |
| 内存占用 | 200 MB | 50 MB | **4x** |

### 选择建议

| 集群规模 | 推荐模式 | 原因 |
|---------|---------|------|
| Service < 100 | iptables | 规则少，性能差异不明显 |
| Service 100-1000 | iptables/ipvs | 均可，根据需求选择 |
| Service > 1000 | ipvs | 性能优势明显 |
| 需要高级负载均衡 | ipvs | 支持多种算法 |
| 老旧内核环境 | iptables | 兼容性好 |

---

## 面试回答

**面试官**：请介绍一下 Kubernetes Service 的代理模式？

**回答**：Kubernetes Service 通过 kube-proxy 组件实现流量代理和负载均衡，支持三种代理模式。第一种是 **userspace 模式**，这是最早期的实现，kube-proxy 在用户态监听端口并转发流量，但由于频繁的用户态-内核态切换导致性能很差，已被废弃。第二种是 **iptables 模式**，这是目前的默认模式，通过在宿主机配置 iptables 规则实现 DNAT，流量在内核态直接转发，性能较好，但规则是线性链表结构，查询复杂度 O(n)，在大规模集群（Service 数量超过 1000）时规则同步慢、性能下降明显。第三种是 **ipvs 模式**，这是推荐的生产模式，专门为负载均衡设计的内核模块，使用哈希表存储规则，查询复杂度 O(1)，支持轮询、加权轮询、源地址哈希、最少连接等多种负载均衡算法，规则增量更新，在大规模集群中性能优势显著，规则同步时间比 iptables 快 60 倍以上。在生产环境中，如果集群规模较大或需要高级负载均衡功能，建议使用 ipvs 模式；小型集群或兼容性要求高的场景可以使用 iptables 模式。切换模式只需修改 kube-proxy 的 ConfigMap 配置即可。

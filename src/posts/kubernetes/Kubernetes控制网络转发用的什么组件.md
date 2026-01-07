---
date: 2025-07-01
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
---

# Kubernetes控制网络转发用的什么组件

## 核心答案

Kubernetes 主要使用 **kube-proxy** 来控制网络转发，它通过以下三种模式实现 Service 的负载均衡和流量转发：

1. **iptables 模式**（默认，最常用）
2. **IPVS 模式**（高性能场景）
3. **userspace 模式**（已废弃）

此外，底层还依赖：
- **Linux 内核的 Netfilter 框架**（iptables/IPVS 的基础）
- **CNI 插件**（如 Calico、Flannel、Weave）处理 Pod 间的网络通信
- **网络命名空间**（Network Namespace）实现网络隔离

---

## 详细解析

**1. kube-proxy 的作用**

kube-proxy 是 Kubernetes 集群中每个节点上运行的网络代理组件，负责：
- 监听 API Server 中 Service 和 Endpoints 的变化
- 维护节点上的网络规则（iptables 或 IPVS 规则）
- 实现 Service 的虚拟 IP（ClusterIP）到 Pod IP 的转发
- 提供负载均衡功能

**2. 三种转发模式对比**

| 特性             | iptables 模式          | IPVS 模式                   | userspace 模式            |
| ---------------- | ---------------------- | --------------------------- | ------------------------- |
| **实现原理**     | 使用 iptables NAT 规则 | 使用 IPVS 负载均衡器        | 用户态代理                |
| **性能**         | 中等，规则多时性能下降 | 高，时间复杂度 O(1)         | 低，需要内核态/用户态切换 |
| **规则数量**     | Service × Endpoint（大规模集群规则爆炸） | 虚拟服务器 + RealServer（优化，使用哈希表） | N/A                       |
| **负载均衡算法** | 随机（无会话保持）     | 支持多种算法（rr/lc/sh/dh/sed/nq等）  | 轮询                      |
| **状态**         | 当前默认               | 推荐用于大规模集群          | 已废弃（Kubernetes 1.0）  |
| **适用场景**     | 中小规模集群（< 1000 Service） | 大规模集群（>1000 Service） | 不推荐使用                |
| **会话保持**     | 不支持                 | 支持（使用 sh 算法）        | 支持                      |
| **内核支持**     | 普遍支持               | 需要内核启用 IPVS 模块      | 无特殊要求                |
| **内存占用**     | 规则多时占用大         | 相对较小                   | 较小                      |
| **规则更新**     | 逐条更新，影响性能     | 批量更新，性能更好         | 实时更新                  |

**3. iptables 模式详解**

```bash
# 工作原理
ClusterIP:Port → iptables DNAT → PodIP:Port

# 规则链路
PREROUTING → KUBE-SERVICES → KUBE-SVC-XXX → KUBE-SEP-XXX → Pod
```

**特点**：
- 每个 Service 创建一条 `KUBE-SVC-*` 链
- 每个 Endpoint 创建一条 `KUBE-SEP-*` 链
- 使用随机概率实现负载均衡（通过 `--probability` 参数）
- 规则数量 = Service数 × Endpoint数，大规模集群性能问题

**查看 iptables 规则**：
```bash
# 查看 KUBE-SERVICES 链
iptables -t nat -L KUBE-SERVICES -n -v

# 查看具体 Service 的规则
iptables -t nat -L KUBE-SVC-XXXXX -n -v

# 查看 Endpoint 规则
iptables -t nat -L KUBE-SEP-XXXXX -n -v

# 示例输出
Chain KUBE-SERVICES (2 references)
target     prot opt source      destination         
KUBE-SVC-XXXXX  tcp  --  0.0.0.0/0   10.96.0.1   /* default/kubernetes:https cluster IP */

Chain KUBE-SVC-XXXXX (1 references)
target          prot opt source      destination         
KUBE-SEP-YYY1   all  --  0.0.0.0/0   0.0.0.0/0   /* default/my-service */ statistic mode random probability 0.33333
KUBE-SEP-YYY2   all  --  0.0.0.0/0   0.0.0.0/0   /* default/my-service */ statistic mode random probability 0.50000
KUBE-SEP-YYY3   all  --  0.0.0.0/0   0.0.0.0/0   /* default/my-service */

Chain KUBE-SEP-YYY1 (1 references)
target     prot opt source          destination         
DNAT       tcp  --  0.0.0.0/0       0.0.0.0/0       /* default/my-service */ tcp to:10.244.1.5:8080
```

**4. IPVS 模式详解**

```bash
# 工作原理
ClusterIP:Port → IPVS 虚拟服务器 → RealServer (PodIP:Port)

# 使用 ipvsadm 查看规则
ipvsadm -Ln
```

**特点**：
- 基于内核态的 IPVS（IP Virtual Server）
- 使用哈希表存储规则，查找速度 O(1)
- 支持丰富的负载均衡算法：
  - `rr`（Round Robin）：轮询
  - `lc`（Least Connection）：最少连接
  - `dh`（Destination Hashing）：目标地址哈希
  - `sh`（Source Hashing）：源地址哈希，实现会话保持
  - `sed`（Shortest Expected Delay）：最短期望延迟
  - `nq`（Never Queue）：永不排队
- 更好的性能，适合大规模集群

**启用 IPVS 模式**：

1. **检查内核支持**：
```bash
# 验证 IPVS 模块是否已加载
lsmod | grep ip_vs

# 如果未加载，手动加载必要模块
modprobe ip_vs
modprobe ip_vs_rr
modprobe ip_vs_wrr
modprobe ip_vs_sh
modprobe ip_vs_dh
modprobe ip_vs_sed
modprobe ip_vs_nq
modprobe nf_conntrack_ipv4
```

2. **修改 kube-proxy ConfigMap**：
```yaml
# kube-proxy ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-proxy
  namespace: kube-system
data:
  config.conf: |
    mode: "ipvs"
    ipvs:
      scheduler: "rr"  # 负载均衡算法
      minSyncPeriod: 0s
      syncPeriod: 30s
      strictARP: true  # 启用严格 ARP 检查
```

3. **重启 kube-proxy**：
```bash
kubectl delete pod -n kube-system -l k8s-app=kube-proxy
```

4. **验证 IPVS 模式**：
```bash
# 检查 kube-proxy 日志
kubectl logs -n kube-system kube-proxy-<pod-name> | grep "Using ipvs Proxier"

# 检查 IPVS 规则
ipvsadm -Ln
```

**查看 IPVS 规则**：
```bash
# 安装 ipvsadm
yum install ipvsadm -y  # CentOS
apt install ipvsadm -y  # Ubuntu

# 查看虚拟服务器
ipvsadm -Ln

# 示例输出
IP Virtual Server version 1.2.1 (size=4096)
Prot LocalAddress:Port Scheduler Flags
  -> RemoteAddress:Port           Forward Weight ActiveConn InActConn
TCP  10.96.0.1:443 rr
  -> 192.168.1.100:6443           Masq    1      0          0
  -> 192.168.1.101:6443           Masq    1      0          0
TCP  10.96.10.20:80 rr
  -> 10.244.1.5:8080              Masq    1      0          0
  -> 10.244.2.6:8080              Masq    1      0          0
  -> 10.244.3.7:8080              Masq    1      0          0

# 查看连接
ipvsadm -Lnc

# 查看统计
ipvsadm -Ln --stats
```

**5. Netfilter/iptables 框架**

Kubernetes 网络转发的底层是 Linux 内核的 **Netfilter** 框架：

```
数据包流向：
PREROUTING → FORWARD/INPUT → OUTPUT/POSTROUTING

五条链（Chains）：
1. PREROUTING   - 数据包到达网络接口后
2. INPUT        - 数据包进入本机进程前
3. FORWARD      - 数据包转发前
4. OUTPUT       - 本机进程发出数据包后
5. POSTROUTING  - 数据包离开网络接口前

四张表（Tables）：
1. filter  - 过滤（默认）
2. nat     - 网络地址转换（kube-proxy 主要使用）
3. mangle  - 修改数据包
4. raw     - 原始数据包处理
```

**kube-proxy 使用的 iptables 链**：
```bash
# NAT 表中的自定义链
KUBE-SERVICES      # 入口链，匹配 Service ClusterIP
KUBE-NODEPORTS     # NodePort 类型 Service
KUBE-POSTROUTING   # 源地址转换（SNAT）
KUBE-SVC-*         # 每个 Service 的规则链
KUBE-SEP-*         # 每个 Endpoint 的规则链
KUBE-MARK-MASQ     # 标记需要 MASQUERADE 的包
KUBE-MARK-DROP     # 标记需要丢弃的包
```

**6. CNI 插件的作用**

CNI（Container Network Interface）负责 Pod 的网络配置：

| CNI 插件    | 实现方式        | 特点                             |
| ----------- | --------------- | -------------------------------- |
| **Flannel** | VXLAN/host-gw   | 简单易用，适合小规模集群         |
| **Calico**  | BGP/IPIP        | 支持网络策略，高性能，大规模集群 |
| **Weave**   | UDP/TCP overlay | 自动发现，易部署                 |
| **Cilium**  | eBPF            | 高性能，可观测性强               |
| **Canal**   | Flannel+Calico  | 结合两者优势                     |

**CNI 工作流程**：
```bash
1. kubelet 调用 CNI 插件（通过 /opt/cni/bin 目录下的可执行文件）
2. CNI 插件创建 veth pair（虚拟网卡对）
3. 一端放入 Pod 网络命名空间（eth0），另一端连接到宿主机网桥
4. 配置 Pod IP 地址和路由表
5. 配置节点间的路由或隧道
6. 更新集群路由信息（如 Calico 使用 BGP 发布路由）

# 查看 CNI 配置
ls /etc/cni/net.d/

# 示例：Flannel 配置
cat /etc/cni/net.d/10-flannel.conflist
```

**CNI 配置文件结构**：
```json
{
  "name": "flannel",
  "cniVersion": "0.3.1",
  "plugins": [
    {
      "type": "flannel",
      "delegate": {
        "hairpinMode": true,
        "isDefaultGateway": true
      }
    },
    {
      "type": "portmap",
      "capabilities": {
        "portMappings": true
      }
    }
  ]
}
```

**CNI 插件的网络模型**：
- **Overlay 网络**：通过隧道技术在现有网络上构建虚拟网络（如 Flannel VXLAN、Weave）
- **Underlay 网络**：直接使用物理网络，需要额外的网络配置（如 Calico BGP）
- **混合网络**：结合 Overlay 和 Underlay 的优势（如 Canal）

**7. 完整的网络转发流程**

**场景：Pod A 访问 Service B**

```
步骤1: Pod A 发起请求
  Pod A (10.244.1.5) → Service B (ClusterIP: 10.96.10.20:80)

步骤2: iptables/IPVS 处理（在 Pod A 所在节点）
  - 数据包进入 PREROUTING 链
  - 匹配 KUBE-SERVICES 规则
  - DNAT 转换：10.96.10.20:80 → 10.244.2.6:8080 (Pod B)

步骤3: CNI 网络处理
  - 如果 Pod B 在同一节点：直接通过 bridge 转发
  - 如果 Pod B 在不同节点：
    * Flannel VXLAN: 封装到 UDP 隧道
    * Calico BGP: 通过路由表直接转发
    * 数据包通过物理网络到达目标节点

步骤4: 目标节点处理
  - 数据包到达 Pod B 所在节点
  - 根据路由规则转发到 Pod B 的网络命名空间
  - Pod B (10.244.2.6:8080) 接收请求

步骤5: 响应返回
  - Pod B 响应 → 源地址 10.244.2.6:8080
  - SNAT 处理（如需要）
  - 逆向路径返回到 Pod A
```

**8. Service 类型的网络转发差异**

```yaml
# 1. ClusterIP（默认）
# 只能集群内部访问，使用 iptables/IPVS 转发
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  clusterIP: 10.96.10.20
  ports:
  - port: 80
    targetPort: 8080

# 2. NodePort
# 在每个节点上开放端口，通过 KUBE-NODEPORTS 链处理
spec:
  type: NodePort
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30080  # 节点端口 30000-32767

# 转发流程：
# NodeIP:30080 → iptables DNAT → ClusterIP:80 → PodIP:8080

# 3. LoadBalancer
# 依赖云厂商的负载均衡器
# 外部 LB → NodePort → ClusterIP → Pod

# 4. ExternalName
# 返回 CNAME 记录，不涉及代理
spec:
  type: ExternalName
  externalName: my.database.example.com
```

**9. 网络策略（NetworkPolicy）**

控制 Pod 间的访问规则（需要 CNI 支持，如 Calico）：

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-frontend
spec:
  podSelector:
    matchLabels:
      app: backend
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
      port: 3306
```

**实现方式**（以 Calico 为例）：
- 使用 iptables 或 eBPF 实现
- 在 Pod 的网络命名空间中添加过滤规则

```bash
# 查看 Calico 生成的 iptables 规则
iptables -L -n -v | grep cali
```

**10. 排查网络转发问题**

```bash
# 1. 检查 kube-proxy 状态
kubectl get pods -n kube-system | grep kube-proxy
kubectl logs -n kube-system kube-proxy-xxxxx

# 2. 检查 kube-proxy 模式
kubectl logs -n kube-system kube-proxy-xxxxx | grep "Using"
# 输出: Using iptables Proxier 或 Using ipvs Proxier

# 3. 检查 Service 和 Endpoints
kubectl get svc my-service
kubectl get endpoints my-service

# 4. 检查 iptables 规则
iptables-save | grep <service-name>
iptables -t nat -L KUBE-SERVICES -n

# 5. 检查 IPVS 规则
ipvsadm -Ln | grep <service-ip>

# 6. 检查 CNI 插件
kubectl get pods -n kube-system | grep -E 'calico|flannel|weave'
kubectl logs -n kube-system <cni-pod-name>

# 7. 测试网络连通性
# 从一个 Pod 访问 Service
kubectl exec -it pod-a -- curl http://my-service:80

# 测试 ClusterIP
kubectl exec -it pod-a -- curl http://10.96.10.20:80

# 测试 Pod IP
kubectl exec -it pod-a -- curl http://10.244.2.6:8080

# 8. 抓包分析
# 在节点上抓包
tcpdump -i any -nn port 80

# 在 Pod 网络命名空间抓包
nsenter -t <pod-pid> -n tcpdump -i eth0 -nn
```

---

## 架构总结

```
┌─────────────────────────────────────────────────────────────┐
│                      Kubernetes 集群                          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐    API Server 监听     ┌──────────────┐   │
│  │  Service /   │ ◄─────────────────────► │  kube-proxy  │   │
│  │  Endpoints   │                         │  (每个节点)   │   │
│  └──────────────┘                         └───────┬──────┘   │
│                                                    │          │
│                                          生成/更新规则       │
│                                                    ▼          │
│                        ┌──────────────────────────────┐      │
│                        │   iptables / IPVS 规则       │      │
│                        │  (Netfilter 框架)            │      │
│                        └────────────┬─────────────────┘      │
│                                     │                        │
│                                  转发流量                    │
│                                     ▼                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │               CNI 插件网络                             │   │
│  │  ┌────────┐      ┌────────┐      ┌────────┐         │   │
│  │  │ Pod A  │ ───► │  veth  │ ───► │  CNI   │         │   │
│  │  │ (NS1)  │      │  pair  │      │ Bridge │         │   │
│  │  └────────┘      └────────┘      └────────┘         │   │
│  │                                      │               │   │
│  │                        物理网络 / 隧道 (VXLAN/BGP)    │   │
│  │                                      ▼               │   │
│  │  ┌────────┐      ┌────────┐      ┌────────┐         │   │
│  │  │ Pod B  │ ◄─── │  veth  │ ◄─── │  CNI   │         │   │
│  │  │ (NS2)  │      │  pair  │      │ Bridge │         │   │
│  │  └────────┘      └────────┘      └────────┘         │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

---

### 相关常见问题

### Q1: iptables 模式和 IPVS 模式有什么区别？如何选择？

**答案**：

**主要区别**：

| 对比项   | iptables 模式        | IPVS 模式                |
| -------- | -------------------- | ------------------------ |
| 性能     | O(n)，规则多时性能差 | O(1)，哈希表查找         |
| 负载均衡 | 随机，无会话保持     | 支持 rr/lc/sh 等多种算法 |
| 规则数量 | Service × Endpoint   | 虚拟服务器 + RealServer  |
| 内存占用 | 规则多时占用大       | 相对较小                 |
| 成熟度   | 默认，最成熟         | 需要内核支持 IPVS        |

**选择建议**：
- **小型集群（< 100 Service）**: iptables 模式，简单稳定
- **大型集群（> 1000 Service）**: IPVS 模式，性能更好
- **需要会话保持**: IPVS 模式，使用 `sh` (Source Hashing) 算法
- **需要更精细的负载均衡**: IPVS 模式，支持多种算法

**切换到 IPVS**：
```bash
# 1. 确保节点加载 IPVS 模块
modprobe ip_vs
modprobe ip_vs_rr
modprobe ip_vs_wrr
modprobe ip_vs_sh

# 2. 修改 kube-proxy ConfigMap
kubectl edit cm kube-proxy -n kube-system
# 设置 mode: "ipvs"

# 3. 重启 kube-proxy
kubectl delete pod -n kube-system -l k8s-app=kube-proxy

# 4. 验证
kubectl logs -n kube-system kube-proxy-xxxxx | grep "Using ipvs Proxier"
```

### Q2: Service 的 ClusterIP 是如何实现的？为什么能在集群内访问？

**答案**：

**实现原理**：

ClusterIP 是一个**虚拟 IP**，不对应任何实际的网络接口，完全由 **kube-proxy 通过 iptables/IPVS 规则实现**。

**工作流程**：

```bash
1. Service 创建时，API Server 分配 ClusterIP（从 service-cluster-ip-range）
2. kube-proxy 监听到 Service 创建事件
3. kube-proxy 在每个节点上创建 iptables/IPVS 规则：
   - 目标地址是 ClusterIP:Port 的数据包
   - DNAT 转换为 Pod IP:Port
4. 当 Pod 访问 ClusterIP 时，数据包经过本地 iptables/IPVS
5. 规则匹配后，目标地址被改写为某个 Pod IP
6. 通过 CNI 网络转发到目标 Pod
```

**关键点**：
- ClusterIP 只在集群内部有效（通过 iptables 规则实现）
- 每个节点都有完整的规则副本
- 不需要额外的路由配置，因为是通过 DNAT 实现
- 流量不经过 kube-proxy 进程（iptables/IPVS 在内核态完成）

**验证**：
```bash
# ClusterIP 不会出现在网卡上
ip addr  # 看不到 ClusterIP

# 但可以 ping 通（如果 Service 有对应的 Endpoints）
ping <cluster-ip>  # 通过 iptables 规则转发

# 查看 DNAT 规则
iptables -t nat -L KUBE-SERVICES -n | grep <cluster-ip>
```

### Q3: NodePort 的流量转发路径是什么？有什么性能影响？

**答案**：

**转发路径**：

```
外部客户端 → NodeIP:NodePort 
           ↓ (iptables DNAT)
         ClusterIP:Port
           ↓ (iptables DNAT)
         PodIP:TargetPort
```

**详细流程**：

```bash
# 1. 外部请求到达任意节点的 NodePort
curl http://192.168.1.100:30080

# 2. PREROUTING 链处理
iptables -t nat -A PREROUTING -m addrtype --dst-type LOCAL -j KUBE-NODEPORTS

# 3. KUBE-NODEPORTS 链匹配 NodePort
# 转换为 ClusterIP:Port
DNAT: 192.168.1.100:30080 → 10.96.10.20:80

# 4. KUBE-SERVICES 链继续处理
# 转换为 Pod IP
DNAT: 10.96.10.20:80 → 10.244.2.6:8080

# 5. 如果目标 Pod 在其他节点，通过 CNI 网络转发

# 6. POSTROUTING 链处理源地址转换
SNAT: 源地址改为节点 IP（否则响应无法返回）
```

**性能影响**：

1. **双层 DNAT**：NodePort → ClusterIP → Pod IP，增加延迟
2. **跨节点转发**：如果 Pod 不在当前节点，额外一次网络跳转
3. **SNAT 开销**：源地址转换，失去真实客户端 IP

**优化方案**：

```yaml
# 1. externalTrafficPolicy: Local
# 只转发到本地节点的 Pod，避免跨节点转发
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: NodePort
  externalTrafficPolicy: Local  # 关键配置
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30080

# 优点：
# - 保留客户端源 IP
# - 避免跨节点转发
# 缺点：
# - 流量分布不均（只到有 Pod 的节点）
# - 如果节点没有 Pod，请求失败

# 2. 使用 LoadBalancer 类型
# 云厂商的 LB 直接转发到健康的节点

# 3. 使用 Ingress
# 七层负载均衡，性能更好
```

### Q4: Kubernetes 中的 DNS 解析是如何实现的？

**答案**：

**CoreDNS 实现**：

Kubernetes 使用 **CoreDNS** 提供集群内的 DNS 服务：

```bash
# CoreDNS 以 Deployment 形式运行
kubectl get pods -n kube-system | grep coredns

# CoreDNS Service（kube-dns）
kubectl get svc -n kube-system kube-dns
# ClusterIP: 10.96.0.10（通常是 service-cluster-ip-range 的第10个）
```

**DNS 记录规则**：

```bash
# 1. Service 的 A 记录
<service-name>.<namespace>.svc.cluster.local → ClusterIP
my-service.default.svc.cluster.local → 10.96.10.20

# 2. Service 的短域名（同命名空间）
<service-name> → ClusterIP
my-service → 10.96.10.20

# 3. Headless Service（ClusterIP: None）
# 返回所有 Pod IP
<service-name>.<namespace>.svc.cluster.local → [10.244.1.5, 10.244.2.6, ...]

# 4. StatefulSet Pod 的 DNS
<pod-name>.<service-name>.<namespace>.svc.cluster.local → Pod IP
web-0.nginx.default.svc.cluster.local → 10.244.1.5

# 5. Pod 的反向 DNS
<pod-ip-with-dashes>.<namespace>.pod.cluster.local
10-244-1-5.default.pod.cluster.local → 10.244.1.5
```

**Pod DNS 配置**：

```bash
# 每个 Pod 的 /etc/resolv.conf
cat /etc/resolv.conf

nameserver 10.96.0.10           # CoreDNS ClusterIP
search default.svc.cluster.local svc.cluster.local cluster.local
options ndots:5
```

**DNS 策略**：

```yaml
spec:
  # 1. ClusterFirst（默认）：优先使用集群 DNS
  dnsPolicy: ClusterFirst
  
  # 2. Default：使用节点的 DNS 配置
  dnsPolicy: Default
  
  # 3. None：完全自定义
  dnsPolicy: None
  dnsConfig:
    nameservers:
    - 1.1.1.1
    searches:
    - my.domain.com
    options:
    - name: ndots
      value: "2"
  
  # 4. ClusterFirstWithHostNet：hostNetwork Pod 使用集群 DNS
  dnsPolicy: ClusterFirstWithHostNet
```

### Q5: 如何排查 Service 无法访问的问题？

**答案**：

**系统化排查流程**：

```bash
# 1. 检查 Service 是否存在
kubectl get svc my-service -n default

# 2. 检查 Endpoints 是否有 IP
kubectl get endpoints my-service -n default
# 如果为空或不完整，说明 Pod 选择器或就绪探针有问题

# 3. 检查 Pod 标签和 Service 选择器
kubectl get pods -l app=myapp --show-labels
kubectl describe svc my-service | grep Selector

# 4. 检查 Pod 是否就绪
kubectl get pods -l app=myapp
# READY 必须是 1/1，否则不会加入 Endpoints

# 5. 测试 Pod 直接访问
POD_IP=$(kubectl get pod <pod-name> -o jsonpath='{.status.podIP}')
kubectl run test --image=busybox --rm -it -- wget -O- http://$POD_IP:8080
# 如果 Pod IP 可访问，说明问题在 Service 层

# 6. 测试 ClusterIP 访问
kubectl run test --image=busybox --rm -it -- wget -O- http://<cluster-ip>:80
# 如果不通，检查 kube-proxy

# 7. 检查 kube-proxy 日志
kubectl logs -n kube-system kube-proxy-xxxxx | grep -i error

# 8. 检查 iptables 规则（iptables 模式）
iptables-save | grep <service-name>
iptables -t nat -L KUBE-SERVICES -n | grep <cluster-ip>

# 9. 检查 IPVS 规则（IPVS 模式）
ipvsadm -Ln | grep <cluster-ip>

# 10. 检查端口配置
kubectl get svc my-service -o yaml | grep -A 5 ports
# Service port、targetPort、containerPort 必须匹配

# 11. 检查 NetworkPolicy
kubectl get networkpolicy -n default
kubectl describe networkpolicy <policy-name>

# 12. 检查 DNS 解析（如果通过域名访问）
kubectl run test --image=busybox --rm -it -- nslookup my-service
```

**常见问题和解决**：

| 问题               | 原因               | 解决方法                            |
| ------------------ | ------------------ | ----------------------------------- |
| Endpoints 为空     | Selector 不匹配    | 修正 Service selector 或 Pod labels |
| Endpoints 为空     | Pod 未就绪         | 检查 readinessProbe，修复应用       |
| ClusterIP 不通     | kube-proxy 故障    | 重启 kube-proxy，检查日志           |
| ClusterIP 不通     | iptables 规则缺失  | 删除重建 Service                    |
| NodePort 不通      | 防火墙阻止         | 开放 NodePort 范围（30000-32767）   |
| 跨命名空间访问失败 | NetworkPolicy 阻止 | 添加允许规则                        |
| DNS 解析失败       | CoreDNS 故障       | 检查 CoreDNS Pod 状态               |

### Q6: externalTrafficPolicy: Local 和 Cluster 有什么区别？

**答案**：

**两种策略对比**：

| 特性           | Cluster（默认）    | Local                 |
| -------------- | ------------------ | --------------------- |
| **流量分布**   | 转发到集群所有 Pod | 只转发到本地节点 Pod  |
| **负载均衡**   | 跨节点负载均衡     | 仅本地 Pod 负载均衡   |
| **客户端 IP**  | 丢失（SNAT）       | 保留                  |
| **跨节点跳转** | 可能发生           | 不会发生              |
| **健康检查**   | 节点级别           | 节点 + Pod 级别       |
| **单点故障**   | 无影响             | 如果节点无 Pod 则失败 |

**Cluster 模式（默认）**：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: NodePort
  externalTrafficPolicy: Cluster  # 默认值
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30080
```

**流量路径**：
```
客户端 → 任意节点:30080 
       ↓ (SNAT + DNAT)
       可能转发到其他节点的 Pod
       ↓
       PodIP:8080（源 IP 已改为节点 IP）
```

**Local 模式**：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: NodePort
  externalTrafficPolicy: Local  # 关键
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30080
```

**流量路径**：
```
客户端 → 有 Pod 的节点:30080 
       ↓ (只 DNAT，不 SNAT)
       本地 Pod
       ↓
       PodIP:8080（保留客户端源 IP）
```

**使用场景**：

**Cluster 模式**：
- 需要均匀的负载分布
- 不关心客户端源 IP
- 需要高可用（任意节点都能处理）

**Local 模式**：
- 需要获取客户端真实 IP（日志、审计、限流）
- 延迟敏感应用（避免跨节点转发）
- 配合 LoadBalancer 的健康检查

**健康检查差异**：

```bash
# Cluster 模式
# LoadBalancer 检查节点端口，只要 kube-proxy 运行就返回成功
# 即使节点上没有 Pod

# Local 模式  
# LoadBalancer 检查节点端口，只有节点有健康 Pod 才返回成功
# kube-proxy 会检查本地 Endpoints

# 查看健康检查端点
kubectl get svc my-service -o yaml | grep healthCheckNodePort
```

**完整示例**：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: web-service
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: web

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 3
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
      - name: web
        image: nginx
        ports:
        - containerPort: 8080
        # 应用需要处理 X-Forwarded-For 或直接读取源 IP
```

---

### 常见问题补充

### Q7: kube-proxy 如何实现 Service 的负载均衡？

**答案**：
- **iptables 模式**：通过 DNAT 规则将 ClusterIP:Port 转发到随机选择的 Pod IP:Port
- **IPVS 模式**：使用 IPVS 负载均衡器，将虚拟服务器（ClusterIP:Port）转发到多个 RealServer（Pod IP:Port）
- **负载均衡算法**：
  - iptables：随机算法
  - IPVS：支持 rr（轮询）、lc（最少连接）、sh（源地址哈希）等多种算法
- **动态更新**：kube-proxy 监听 API Server 中 Service 和 Endpoints 的变化，实时更新转发规则

### Q8: Service 与 Endpoints 的关系是什么？

**答案**：
- **Service**：定义了一组 Pod 的访问方式，包括虚拟 IP（ClusterIP）、端口等
- **Endpoints**：维护了 Service 对应的 Pod IP 和端口列表
- **关系**：
  - Service 通过标签选择器（labelSelector）选择 Pod
  - Endpoints Controller 监听 Service 和 Pod 的变化，自动更新 Endpoints
  - kube-proxy 监听 Endpoints 的变化，更新转发规则
  - 如果 Service 没有标签选择器，则不会自动生成 Endpoints（手动管理）

### Q9: Headless Service 是如何工作的？有什么应用场景？

**答案**：
- **定义**：Headless Service 是没有 ClusterIP 的 Service（clusterIP: None）
- **工作原理**：
  - DNS 解析返回所有 Pod IP 列表（A 记录）
  - 不进行负载均衡，直接返回所有 Pod IP
  - StatefulSet 自动为每个 Pod 创建稳定的 DNS 记录：pod-name.service-name.namespace.svc.cluster.local
- **应用场景**：
  - StatefulSet 服务发现
  - 自定义负载均衡
  - 需要直接访问每个 Pod 的场景
  - 分布式系统中的成员发现

### Q10: Kubernetes 中的网络策略（NetworkPolicy）是如何实现的？

**答案**：
- **定义**：NetworkPolicy 是 Kubernetes 中用于控制 Pod 间通信的资源
- **实现依赖**：需要支持 NetworkPolicy 的 CNI 插件（如 Calico、Cilium、Weave）
- **实现方式**：
  - **Calico**：使用 iptables 或 eBPF 实现网络规则
  - **Cilium**：使用 eBPF 实现高性能网络策略
  - **Weave**：使用自定义内核模块实现
- **功能**：
  - 控制 ingress（入站）和 egress（出站）流量
  - 基于标签、命名空间、IP 地址的访问控制
  - 端口级别的精细控制

### 关键点总结

**核心组件**：
- **kube-proxy**: 维护转发规则（iptables/IPVS）
- **Netfilter**: Linux 内核网络框架
- **CNI 插件**: Pod 网络通信
- **CoreDNS**: 集群 DNS 服务

**转发模式**：
- **iptables**: 默认，适合中小集群（< 1000 Service）
- **IPVS**: 高性能，适合大规模集群（> 1000 Service）
- 选择依据：集群规模、负载均衡需求、会话保持需求

**关键路径**：
```
Service (ClusterIP) → kube-proxy 规则 → DNAT → Pod IP → CNI 网络 → Pod
```

**排查要点**：
1. Service → Endpoints 是否正确（标签选择器匹配）
2. kube-proxy 是否正常运行（检查日志和状态）
3. iptables/IPVS 规则是否正确生成
4. CNI 插件是否正常工作（Pod 网络是否连通）
5. NetworkPolicy 是否阻止了流量
6. DNS 解析是否正常（Service 域名解析）
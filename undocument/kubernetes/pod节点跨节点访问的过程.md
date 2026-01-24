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

# pod节点跨节点访问的过程

## Kubernetes网络模型基础

### Kubernetes网络三大原则
1. **所有Pod可以不使用NAT直接互相通信**
2. **所有节点可以不使用NAT直接与所有Pod通信（反之亦然）**
3. **Pod看到的自己的IP地址与其他Pod看到的它的IP地址相同**

### 网络模型的技术含义
- **唯一Pod IP**：每个Pod获得一个独立的IP地址，作为其在集群中的唯一标识符
- **直接通信**：Pod之间可以直接通过IP地址通信，无需进行端口映射或地址转换
- **透明性**：应用程序无需感知自己是否跨节点通信，网络层负责处理复杂性
- **平面网络**：Kubernetes网络是扁平化的，没有IP地址重叠，简化了网络管理

### Pod IP分配机制
Pod IP通常由CNI插件从预定义的CIDR池中分配：
- 集群级CIDR（如10.244.0.0/16）被划分为多个节点级CIDR（如10.244.1.0/24、10.244.2.0/24）
- 每个节点上的Pod从该节点的CIDR范围内获取IP地址
- CNI插件负责维护IP地址池和分配状态，确保IP地址的唯一性

## Pod跨节点访问的完整流程

```
场景：Node1上的Pod A (10.244.1.10) 访问 Node2上的Pod B (10.244.2.20)

┌─────────────────────────────────────────────────────────────┐
│                         Node 1                               │
│  ┌──────────────────────────────────────────────┐           │
│  │  Pod A (10.244.1.10)                         │           │
│  │  ┌──────────────────────────────────────┐   │           │
│  │  │  Container                            │   │           │
│  │  │  eth0: 10.244.1.10                   │   │           │
│  │  └───────────────┬──────────────────────┘   │           │
│  │                  │                            │           │
│  │            veth pair                          │           │
│  │                  │                            │           │
│  └──────────────────┼────────────────────────────┘           │
│                     │                                         │
│  ┌──────────────────▼────────────────────────────┐           │
│  │  cni0 网桥 (10.244.1.1)                      │           │
│  └──────────────────┬────────────────────────────┘           │
│                     │                                         │
│  ┌──────────────────▼────────────────────────────┐           │
│  │  Node网络栈 (eth0: 192.168.1.10)            │           │
│  └──────────────────┬────────────────────────────┘           │
└─────────────────────┼─────────────────────────────────────────┘
                      │
                      │  物理网络/Overlay网络
                      │
┌─────────────────────▼─────────────────────────────────────────┐
│                         Node 2                               │
│  ┌──────────────────┬────────────────────────────┐           │
│  │  Node网络栈 (eth0: 192.168.1.11)            │           │
│  └──────────────────┼────────────────────────────┘           │
│                     │                                         │
│  ┌──────────────────▼────────────────────────────┐           │
│  │  cni0 网桥 (10.244.2.1)                      │           │
│  └──────────────────┬────────────────────────────┘           │
│                     │                                         │
│  ┌──────────────────▼────────────────────────────┐           │
│  │            veth pair                          │           │
│  │                  │                            │           │
│  │  ┌───────────────▼──────────────────────┐   │           │
│  │  │  Container                            │   │           │
│  │  │  eth0: 10.244.2.20                   │   │           │
│  │  └──────────────────────────────────────┘   │           │
│  │  Pod B (10.244.2.20)                         │           │
│  └──────────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────────┘
```

## 详细的数据包传输过程

### 第一步：Pod A发送数据包

```
源地址: 10.244.1.10 (Pod A)
目标地址: 10.244.2.20 (Pod B)
```

1. **容器内部**：
   - 应用发送数据到 Pod B 的IP
   - 数据包从容器的 eth0 网卡发出
   - 容器的 eth0 实际上是 veth pair 的一端

2. **查找路由表**：
```bash
# Pod内查看路由
ip route
# 输出：
# default via 10.244.1.1 dev eth0
# 10.244.1.0/24 dev eth0 proto kernel scope link src 10.244.1.10
```

### 第二步：通过veth pair到达Node

3. **veth pair传输**：
   - veth pair像一根虚拟网线，连接容器和主机
   - 一端在容器内（eth0）
   - 另一端在主机上（vethxxxx），连接到网桥

4. **到达cni0网桥**：
   - 数据包到达Node1的cni0网桥
   - 网桥检查目标IP (10.244.2.20)
   - 发现不在本地网段 (10.244.1.0/24)

### 第三步：Node1路由决策

5. **Node路由表查找**：
```bash
# Node1上的路由表
ip route
# 输出示例：
# 10.244.1.0/24 dev cni0 proto kernel scope link src 10.244.1.1
# 10.244.2.0/24 via 192.168.1.11 dev eth0  # 通过Node2转发
# 10.244.3.0/24 via 192.168.1.12 dev eth0  # 通过Node3转发
```

6. **封装处理（取决于CNI实现）**：
   - **Overlay模式（如Flannel VXLAN）**：
     - 原始数据包被封装在VXLAN/UDP包中
     - 外层目标地址：192.168.1.11 (Node2的物理IP)
     - 内层目标地址：10.244.2.20 (Pod B)
   
   - **路由模式（如Calico BGP）**：
     - 不封装，直接路由
     - 依赖网络设备的路由表

### 第四步：跨节点传输

7. **网络传输**：
   - 数据包通过物理网络从 Node1 发往 Node2
   - 经过交换机、路由器等网络设备

### 第五步：Node2接收和解封装

8. **Node2接收**：
   - Node2的网卡接收到数据包
   - **Overlay模式**：解封装VXLAN包，提取原始数据包
   - **路由模式**：直接处理

9. **路由到目标Pod**：
   - 查找路由表，发现目标IP属于本地Pod网段
   - 数据包发送到cni0网桥

### 第六步：到达目标Pod

10. **网桥转发**：
    - cni0网桥根据MAC地址表转发
    - 找到对应的veth设备

11. **veth pair传输**：
    - 数据包通过veth pair进入Pod B的网络命名空间

12. **Pod B接收**：
    - 数据包到达Pod B容器的eth0
    - 应用程序接收数据

## 不同CNI插件的实现方式

### 1. Flannel - VXLAN模式（Overlay）

**特点**：
- 使用VXLAN封装，创建Overlay网络
- 简单易用，无需底层网络支持
- 有一定的性能开销（封装/解封装约5-10%的性能损失）
- 支持跨子网部署

**工作原理**：
```
原始包: [IP头: 10.244.1.10 -> 10.244.2.20][TCP][数据]
         ↓ 封装
VXLAN包: [外层IP: 192.168.1.10 -> 192.168.1.11]
         [UDP头: 源端口(随机) -> 8472]
         [VXLAN头: VNI=1, Flags=0x08]
         [原始包]
```

### VXLAN技术深度解析
- **VXLAN头结构**：8字节，包含VNI（VXLAN Network Identifier）字段，用于标识不同的VXLAN网络
- **封装后MTU**：默认MTU为1500的网络中，VXLAN数据包的有效MTU为1450（1500 - 50字节封装开销）
- **转发数据库（FDB）**：Flannel维护VXLAN隧道端点与Pod IP的映射关系
- **ARP响应代理**：Flannel监听ARP请求并直接响应，避免ARP广播风暴

**VXLAN数据包格式**：
```
+------------------------+
| 外层以太网帧头         |
+------------------------+
| 外层IP头 (Node1 -> Node2) |
+------------------------+
| 外层UDP头 (随机端口 -> 8472) |
+------------------------+
| VXLAN头 (VNI=1)        |
+------------------------+
| 内层以太网帧头         |
+------------------------+
| 内层IP头 (PodA -> PodB) |
+------------------------+
| 内层TCP/UDP头          |
+------------------------+
| 数据                   |
+------------------------+
```

**配置示例**：
```yaml
# Flannel ConfigMap
kind: ConfigMap
apiVersion: v1
metadata:
  name: kube-flannel-cfg
  namespace: kube-system
data:
  net-conf.json: |
    {
      "Network": "10.244.0.0/16",      # Pod网络范围
      "Backend": {
        "Type": "vxlan",                # 使用VXLAN
        "VNI": 1,                       # VXLAN网络标识
        "Port": 8472                    # VXLAN端口
      }
    }
```

**数据包路径**：
```
Pod A → veth → cni0 → flannel.1(VXLAN设备) → eth0(Node1)
  ↓ 网络传输
eth0(Node2) → flannel.1(VXLAN设备) → cni0 → veth → Pod B
```

**查看Flannel状态**：
```bash
# 查看VXLAN设备
ip -d link show flannel.1

# 查看转发数据库（FDB）
bridge fdb show dev flannel.1

# 查看路由
ip route | grep flannel
```

### 2. Flannel - Host-Gateway模式（路由）

**特点**：
- 直接路由，无封装
- 性能更好（无封装开销）
- 要求所有节点在同一个二层网络

**工作原理**：
```bash
# Node1路由表
10.244.2.0/24 via 192.168.1.11 dev eth0
# 直接路由到Node2，不封装
```

**限制**：
- 节点必须在同一子网（二层可达）
- 云环境可能不支持

### 3. Calico - BGP模式（路由）

**特点**：
- 使用BGP协议交换路由信息
- 纯三层网络，无封装开销，性能最佳
- 支持丰富的网络策略（NetworkPolicy）
- 可以与现有物理网络基础设施集成
- 支持大规模集群部署（>10000节点）

**工作原理**：
```
1. 每个节点运行BGP客户端（默认使用BIRD，可选GoBGP）
2. 节点通过BGP协议交换各自的Pod CIDR路由信息
3. 节点之间建立直接路由，数据包无需封装
4. 支持全互联模式或路由反射器模式

Node1 BGP宣告: "10.244.1.0/24 via 192.168.1.10"
Node2 BGP宣告: "10.244.2.0/24 via 192.168.1.11"
```

### BGP技术深度解析
- **BGP邻居关系**：节点之间建立TCP连接（端口179），交换路由信息
- **AS号（Autonomous System Number）**：默认使用私有AS号64512，可自定义
- **路由属性**：支持标准BGP属性如Local Preference、MED、AS Path等
- **路由反射器**：大规模集群中使用，减少BGP邻居数量（从O(n²)降至O(n)）
- **BGP联邦**：将大AS划分为多个小AS，简化大规模部署的路由管理

**BGP路由表示例**：
```bash
# 在Node1上查看BGP路由
ip route | grep bird
# 输出：
# 10.244.2.0/24 via 192.168.1.11 dev eth0 proto bird
# 10.244.3.0/24 via 192.168.1.12 dev eth0 proto bird
```

**BGP邻居状态检查**：
```bash
# 使用calicoctl查看BGP状态
calicoctl node status
# 输出示例：
# BGP IPv4 status:
# +---------------+-------------------+-------+------------+-------------+
# | PEER ADDRESS  |     PEER TYPE     | STATE |   SINCE    |    INFO     |
# +---------------+-------------------+-------+------------+-------------+
# | 192.168.1.11  | node-to-node mesh | up    | 2024-01-01 | Established |
# | 192.168.1.12  | node-to-node mesh | up    | 2024-01-01 | Established |
# +---------------+-------------------+-------+------------+-------------+
```

**BGP路由分发**：
```
┌────────────┐         BGP路由交换        ┌────────────┐
│   Node1    │◄──────────────────────────►│   Node2    │
│  10.244.1  │   "我负责10.244.2.0/24"   │  10.244.2  │
└────────────┘                             └────────────┘
       ▲                                         ▲
       │         BGP Route Reflector             │
       └─────────────────┬─────────────────────┘
                         │
                   (可选，用于大规模集群)
```

**配置示例**：
```yaml
# Calico配置
apiVersion: projectcalico.org/v3
kind: BGPConfiguration
metadata:
  name: default
spec:
  logSeverityScreen: Info
  nodeToNodeMeshEnabled: true    # 启用全互联模式
  asNumber: 64512                # AS号
```

**查看Calico状态**：
```bash
# 查看BGP邻居
calicoctl node status

# 查看路由
ip route | grep bird

# 查看网络策略
calicoctl get networkpolicy
```

### 4. Calico - IPIP模式（Overlay）

**特点**：
- 使用IP-in-IP封装
- 适用于节点不在同一子网的场景
- 比VXLAN开销稍小

**封装方式**：
```
原始包: [IP: 10.244.1.10 -> 10.244.2.20][TCP][数据]
         ↓ IPIP封装
IPIP包: [外层IP: 192.168.1.10 -> 192.168.1.11]
        [原始IP包]
```

### 5. Cilium - eBPF

**特点**：
- 使用eBPF（扩展伯克利包过滤器）技术，在内核层面处理网络数据包
- 性能极高，比传统CNI插件快3-10倍，延迟降低90%以上
- 支持L3/L4/L7全栈网络策略
- 内置服务网格功能，无需额外部署Istio
- 提供细粒度的可观测性和安全监控
- 可以完全替代kube-proxy，消除iptables/IPVS开销

**工作原理**：
```
1. eBPF程序通过Cilium Agent加载到Linux内核
2. 在内核的关键钩子点（如XDP、TC、socket）拦截网络数据包
3. 使用eBPF Map存储路由表、策略规则、连接状态等信息
4. 直接在内核中完成包处理，避免用户态/内核态切换
5. 支持即时编译（JIT），进一步提升性能
```

### eBPF技术深度解析
- **eBPF Map类型**：支持哈希表、数组、LRU缓存等多种数据结构，用于存储网络状态
- **内核钩子点**：
  - **XDP**：网络驱动层面的早期包处理，性能最高
  - **TC**：流量控制层面，支持复杂的包处理逻辑
  - **Socket Filter**：应用层socket层面的过滤
  - **kprobe/uprobe**：内核/用户函数的动态追踪
- **eBPF验证器**：确保加载的eBPF程序安全、无死循环、资源消耗可控
- **eBPF JIT编译器**：将eBPF字节码编译为机器码，提升执行效率

**Cilium eBPF路由实现**：
```bash
# 查看Cilium eBPF路由表
cilium bpf route list
# 输出示例：
# PodCIDR         NextHop       IfIndex  Flags
# 10.244.2.0/24    192.168.1.11    2       direct
# 10.244.3.0/24    192.168.1.12    2       direct
```

**eBPF程序加载验证**：
```bash
# 查看已加载的eBPF程序
bpftool prog list
# 查看eBPF Map
bpftool map list
# 查看Cilium eBPF程序详情
cilium bpf program list
```

**核心优势**：
- **高性能**：绕过iptables，减少上下文切换，延迟降低90%
- **高级网络策略**：支持基于L3/L4/L7的网络策略（如HTTP路径、SSL证书）
- **可观测性**：内核级别监控，无需额外工具即可获取详细的网络流量信息
- **服务网格**：内置Service Mesh功能，无需额外部署Istio
- **安全隔离**：基于eBPF的网络隔离，提供更强的安全保障

**Cilium路由模式**：
```bash
# Cilium使用eBPF实现路由，不依赖传统路由表
cilium status

# 查看eBPF路由表
cilium bpf route list
```

**L7网络策略示例**：
```yaml
apiVersion: "cilium.io/v2"
kind: CiliumNetworkPolicy
metadata:
  name: "l7-policy"
spec:
  endpointSelector:
    matchLabels:
      app: backend
  ingress:
  - fromEndpoints:
    - matchLabels:
        app: frontend
    toPorts:
    - ports:
      - port: "8080"
        protocol: TCP
      rules:
        http:
        - method: "GET"
          path: "/api/v1/*"  # 只允许GET请求访问/api/v1路径
```

## Service访问的跨节点流程

**场景**：Node1上的Pod A访问Service，Service后端Pod分布在多个节点上

```
┌───────────────────────────────────────────────────────────┐
│ 1. Pod A访问Service ClusterIP: 10.96.0.100:80           │
│    (目标地址：10.96.0.100:80)                            │
└───────────────────────────────────────────────────────────┘
                        ↓
┌───────────────────────────────────────────────────────────┐
│ 2. Node1上的kube-proxy进行负载均衡                       │
│    iptables/IPVS 选择一个后端Pod：10.244.2.20:8080       │
└───────────────────────────────────────────────────────────┘
                        ↓
┌───────────────────────────────────────────────────────────┐
│ 3. 进行DNAT转换                                          │
│    源地址：10.244.1.10 → 192.168.1.10 (Node1 IP)         │
│    目标地址：10.96.0.100:80 → 10.244.2.20:8080           │
└───────────────────────────────────────────────────────────┘
                        ↓
┌───────────────────────────────────────────────────────────┐
│ 4. 转换后的数据包通过跨节点网络传输到Node2               │
│    (按照前面的跨节点流程传输)                             │
└───────────────────────────────────────────────────────────┘
                        ↓
┌───────────────────────────────────────────────────────────┐
│ 5. Node2接收数据包，进行SNAT转换                         │
│    源地址：192.168.1.10 → 10.244.1.10 (Pod A IP)         │
└───────────────────────────────────────────────────────────┘
                        ↓
┌───────────────────────────────────────────────────────────┐
│ 6. 数据包转发到目标Pod B (10.244.2.20:8080)              │
└───────────────────────────────────────────────────────────┘
```

### kube-proxy的作用：
- 维护Service和Endpoints的关系
- 实现Service的负载均衡
- 处理Session Affinity（会话亲和性）
- 支持多种负载均衡算法

### iptables模式详细工作原理：
```bash
# 1. PREROUTING链：处理进入节点的数据包
-A PREROUTING -m comment --comment "kubernetes service portals" -j KUBE-SERVICES

# 2. OUTPUT链：处理节点本地产生的数据包  
-A OUTPUT -m comment --comment "kubernetes service portals" -j KUBE-SERVICES

# 3. 匹配Service ClusterIP
-A KUBE-SERVICES -d 10.96.0.100/32 -p tcp -m tcp --dport 80 -j KUBE-SVC-XXXXX

# 4. 负载均衡选择后端Pod（随机算法）
-A KUBE-SVC-XXXXX -m statistic --mode random --probability 0.33 -j KUBE-SEP-POD1
-A KUBE-SVC-XXXXX -m statistic --mode random --probability 0.50 -j KUBE-SEP-POD2
-A KUBE-SVC-XXXXX -j KUBE-SEP-POD3

# 5. DNAT转换到后端Pod IP
-A KUBE-SEP-POD1 -p tcp -j DNAT --to-destination 10.244.1.10:8080
-A KUBE-SEP-POD2 -p tcp -j DNAT --to-destination 10.244.2.20:8080
-A KUBE-SEP-POD3 -p tcp -j DNAT --to-destination 10.244.3.30:8080
```

### IPVS模式详细工作原理：
```bash
# 查看IPVS规则
ipvsadm -Ln

# 输出示例：
# Prot LocalAddress:Port Scheduler Flags
#   -> RemoteAddress:Port           Forward Weight ActiveConn InActConn
# TCP  10.96.0.100:80 rr             
#   -> 10.244.1.10:8080             Masq    1      0          0          
#   -> 10.244.2.20:8080             Masq    1      0          0          
#   -> 10.244.3.30:8080             Masq    1      0          0          
```

### kube-proxy会话亲和性配置：
```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
  sessionAffinity: ClientIP  # 基于客户端IP的会话亲和性
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800  # 会话超时时间（3小时）
```

### Cilium的kube-proxy替代方案：
```yaml
# Cilium可以完全替代kube-proxy，提供更好的性能
apiVersion: v1
kind: ConfigMap
metadata:
  name: cilium-config
  namespace: kube-system
data:
  kube-proxy-replacement: "strict"  # 严格模式，完全替代kube-proxy
```

## 网络性能对比

| CNI实现             | 类型    | 性能  | 复杂度 | 网络策略 | 跨子网支持 | 适用场景                     |
| ------------------- | ------- | ----- | ------ | -------- | ---------- | ---------------------------- |
| **Flannel VXLAN**   | Overlay | ★★★☆☆ | 低     | ❌        | ✅         | 简单部署，跨子网环境         |
| **Flannel Host-gw** | 路由    | ★★★★★ | 低     | ❌        | ❌         | 同子网，高性能要求，简单配置 |
| **Calico BGP**      | 路由    | ★★★★★ | 中     | ✅        | ✅         | 大规模集群，需要网络策略     |
| **Calico IPIP**     | Overlay | ★★★★☆ | 中     | ✅        | ✅         | 跨子网，需要网络策略         |
| **Cilium eBPF**     | eBPF    | ★★★★★ | 高     | ✅        | ✅         | 高性能，高级功能需求         |
| **Weave**           | Overlay | ★★★☆☆ | 中     | ✅        | ✅         | 自动加密，简单部署           |

**性能影响因素**：
1. **封装开销**：Overlay需要封装/解封装，增加CPU和延迟（VXLAN增加50字节，IPIP增加20字节）
2. **MTU设置**：封装会增加包头，需要调整MTU避免分片（推荐VXLAN网络使用1450 MTU）
3. **路由表大小**：大规模集群（>1000节点）路由表会变得很大，影响查找性能
4. **网络策略**：策略越复杂，性能开销越大（Calico iptables实现约5-10%开销，Cilium eBPF实现<1%）
5. **数据包大小**：小包（<64字节）受封装影响更大，大包需要调整MTU

**CNI插件选择建议**：
- **小型集群**：Flannel VXLAN（简单易用）
- **中型集群**：Calico（平衡性能和功能）
- **大型集群**：Calico BGP或Cilium（高性能，大规模支持）
- **高性能需求**：Cilium eBPF（内核级处理，低延迟）
- **网络策略需求**：Calico或Cilium（Flannel不支持网络策略）

## 实际排查和验证

### 1. 查看Pod网络配置

```bash
# 进入Pod查看网络
kubectl exec -it <pod-name> -- ip addr
kubectl exec -it <pod-name> -- ip route

# 查看Pod的网络命名空间
docker inspect <container-id> | grep NetworkMode
```

### 2. 查看Node网络配置

```bash
# 查看网桥
brctl show
ip link show type bridge

# 查看路由表
ip route

# 查看veth pair
ip link | grep veth

# 查看iptables规则
iptables -t nat -L -n | grep <service-ip>
```

### 3. 测试跨节点连通性

```bash
# 从Pod A ping Pod B
kubectl exec -it pod-a -- ping 10.244.2.20

# 追踪路由
kubectl exec -it pod-a -- traceroute 10.244.2.20

# 测试TCP连接
kubectl exec -it pod-a -- nc -zv 10.244.2.20 8080

# 抓包分析
# 在Node上抓包
tcpdump -i any -nn 'host 10.244.2.20'

# 在Pod内抓包（需要安装tcpdump）
kubectl exec -it pod-a -- tcpdump -i eth0 -nn
```

### 4. 性能测试

```bash
# 使用iperf3测试带宽
# Server端（Pod B）
kubectl exec -it pod-b -- iperf3 -s

# Client端（Pod A）
kubectl exec -it pod-a -- iperf3 -c 10.244.2.20

# 测试延迟
kubectl exec -it pod-a -- ping -c 100 10.244.2.20 | grep avg
```

## 常见问题

### 1. 如何为我的Kubernetes集群选择合适的CNI插件？

选择CNI插件应考虑以下因素：
- **集群规模**：小规模集群（<100节点）可选择Flannel；大规模集群建议使用Calico或Cilium
- **性能要求**：对性能要求高的场景优先选择BGP模式（Calico）或eBPF（Cilium）
- **网络策略需求**：需要网络策略时选择Calico或Cilium（Flannel不支持）
- **跨子网支持**：跨子网部署时选择VXLAN/IPIP模式或Cilium
- **高级功能**：需要L7网络策略、服务网格等高级功能时选择Cilium

### 2. Kubernetes如何确保Pod IP地址的唯一性？

Kubernetes通过以下机制确保Pod IP唯一性：
- **CIDR规划**：集群级CIDR被划分为不重叠的节点级CIDR
- **CNI插件管理**：CNI插件负责在每个节点上分配唯一的Pod IP
- **IP地址池**：CNI插件维护IP地址池，避免重复分配
- **节点隔离**：每个节点只能分配自己CIDR范围内的IP地址

### 3. VXLAN封装是什么？它对网络性能有什么影响？

VXLAN是一种Overlay网络技术，用于在现有网络上创建虚拟网络：
- **封装过程**：将原始数据包封装在UDP包中，添加50字节的额外开销
- **性能影响**：
  - 增加CPU使用率（封装/解封装操作）
  - 减少有效MTU（从1500降至1450）
  - 增加网络延迟（约5-10%的性能损失）
- **优势**：支持跨子网部署，对底层网络要求低

### 4. BGP模式和IPIP模式有什么区别？

| 特性 | BGP模式 | IPIP模式 |
|------|---------|----------|
| **封装** | 无 | IP-in-IP封装（20字节开销） |
| **性能** | 最高（无开销） | 较好（低开销） |
| **网络要求** | 底层网络需支持BGP路由 | 无特殊要求，支持跨子网 |
| **复杂性** | 中（需配置BGP） | 低 |
| **适用场景** | 大规模集群，同子网或有BGP支持 | 跨子网部署，需要网络策略 |

### 5. Cilium的eBPF技术为什么比传统CNI插件性能更好？

eBPF技术提供性能优势的原因：
- **内核级处理**：数据包在Linux内核中直接处理，避免用户态/内核态切换
- **精准拦截**：在网络栈的关键钩子点拦截数据包，减少不必要的处理
- **高效数据结构**：使用eBPF Map存储网络状态，查询速度极快
- **即时编译**：eBPF程序支持JIT编译，提升执行效率
- **消除中间层**：可以直接替代kube-proxy，消除iptables/IPVS的开销
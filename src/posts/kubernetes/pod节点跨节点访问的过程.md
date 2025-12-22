# pod节点跨节点访问的过程

#### Kubernetes网络模型基础

**Kubernetes网络三大原则**：
1. **所有Pod可以不使用NAT直接互相通信**
2. **所有节点可以不使用NAT直接与所有Pod通信（反之亦然）**
3. **Pod看到的自己的IP地址与其他Pod看到的它的IP地址相同**

这意味着：
- 每个Pod有唯一的IP地址
- Pod之间可以直接通过IP通信（无需端口映射）
- 跨节点通信对应用透明

#### Pod跨节点访问的完整流程

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

#### 详细的数据包传输过程

**第一步：Pod A发送数据包**

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

**第二步：通过veth pair到达Node**

3. **veth pair传输**：
   - veth pair像一根虚拟网线，连接容器和主机
   - 一端在容器内（eth0）
   - 另一端在主机上（vethxxxx），连接到网桥

4. **到达cni0网桥**：
   - 数据包到达Node1的cni0网桥
   - 网桥检查目标IP (10.244.2.20)
   - 发现不在本地网段 (10.244.1.0/24)

**第三步：Node1路由决策**

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

**第四步：跨节点传输**

7. **网络传输**：
   - 数据包通过物理网络从 Node1 发往 Node2
   - 经过交换机、路由器等网络设备

**第五步：Node2接收和解封装**

8. **Node2接收**：
   - Node2的网卡接收到数据包
   - **Overlay模式**：解封装VXLAN包，提取原始数据包
   - **路由模式**：直接处理

9. **路由到目标Pod**：
   - 查找路由表，发现目标IP属于本地Pod网段
   - 数据包发送到cni0网桥

**第六步：到达目标Pod**

10. **网桥转发**：
    - cni0网桥根据MAC地址表转发
    - 找到对应的veth设备

11. **veth pair传输**：
    - 数据包通过veth pair进入Pod B的网络命名空间

12. **Pod B接收**：
    - 数据包到达Pod B容器的eth0
    - 应用程序接收数据

#### 不同CNI插件的实现方式

**1. Flannel - VXLAN模式（Overlay）**

**特点**：
- 使用VXLAN封装，创建Overlay网络
- 简单易用，无需底层网络支持
- 有一定的性能开销（封装/解封装）

**工作原理**：
```
原始包: [IP头: 10.244.1.10 -> 10.244.2.20][TCP][数据]
         ↓ 封装
VXLAN包: [外层IP: 192.168.1.10 -> 192.168.1.11]
         [UDP: 8472]
         [VXLAN头: VNI=1]
         [原始包]
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

**2. Flannel - Host-Gateway模式（路由）**

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

**3. Calico - BGP模式（路由）**

**特点**：
- 使用BGP协议交换路由信息
- 纯三层网络，性能最好
- 支持网络策略（NetworkPolicy）
- 可以与物理网络集成

**工作原理**：
```
1. 每个节点运行BGP client (BIRD)
2. BGP交换Pod路由信息
3. 节点之间直接路由，不需要Overlay

Node1: "10.244.1.0/24的流量发给我"
Node2: "10.244.2.0/24的流量发给我"
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

**4. Calico - IPIP模式（Overlay）**

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

**5. Cilium - eBPF**

**特点**：
- 使用eBPF技术，性能极高
- 在内核层面处理网络
- 支持高级网络策略和可观测性

**优势**：
- 绕过iptables，减少开销
- 更高效的包处理
- 丰富的网络可见性

#### Service访问的跨节点流程

**场景**：Pod A访问Service，Service后端Pod在其他节点

```
┌───────────────────────────────────────────────────────────┐
│ 1. Pod A访问Service ClusterIP: 10.96.0.100:80           │
└───────────────────────────────────────────────────────────┘
                        ↓
┌───────────────────────────────────────────────────────────┐
│ 2. iptables/IPVS 进行DNAT转换                            │
│    10.96.0.100:80 → 10.244.2.20:8080 (选择一个后端Pod)  │
└───────────────────────────────────────────────────────────┘
                        ↓
┌───────────────────────────────────────────────────────────┐
│ 3. 转换后的目标IP是远程Pod，触发跨节点访问               │
│    按照前面的跨节点流程传输                               │
└───────────────────────────────────────────────────────────┘
```

**kube-proxy的作用**：

**iptables模式**：
```bash
# Service的iptables规则（简化）
# DNAT规则
-A KUBE-SERVICES -d 10.96.0.100/32 -p tcp -m tcp --dport 80 \
   -j KUBE-SVC-XXXXX

# 负载均衡到后端Pod（随机选择）
-A KUBE-SVC-XXXXX -m statistic --mode random --probability 0.33 \
   -j KUBE-SEP-POD1    # 到10.244.1.10:8080
-A KUBE-SVC-XXXXX -m statistic --mode random --probability 0.50 \
   -j KUBE-SEP-POD2    # 到10.244.2.20:8080
-A KUBE-SVC-XXXXX \
   -j KUBE-SEP-POD3    # 到10.244.3.30:8080

# DNAT转换
-A KUBE-SEP-POD2 -p tcp -j DNAT --to-destination 10.244.2.20:8080
```

**IPVS模式**（更高效）：
```bash
# 查看IPVS规则
ipvsadm -Ln

# 输出示例：
# TCP  10.96.0.100:80 rr
#   -> 10.244.1.10:8080    Masq    1      0          0
#   -> 10.244.2.20:8080    Masq    1      0          0
#   -> 10.244.3.30:8080    Masq    1      0          0
```

#### 网络性能对比

| CNI实现             | 类型    | 性能  | 复杂度 | 适用场景             |
| ------------------- | ------- | ----- | ------ | -------------------- |
| **Flannel VXLAN**   | Overlay | ★★★☆☆ | 低     | 简单部署，跨子网     |
| **Flannel Host-gw** | 路由    | ★★★★★ | 低     | 同子网，性能要求高   |
| **Calico BGP**      | 路由    | ★★★★★ | 中     | 大规模，需要网络策略 |
| **Calico IPIP**     | Overlay | ★★★★☆ | 中     | 跨子网，需要策略     |
| **Cilium eBPF**     | eBPF    | ★★★★★ | 高     | 高性能，高级功能     |
| **Weave**           | Overlay | ★★★☆☆ | 中     | 自动加密，简单       |

**性能影响因素**：
1. **封装开销**：Overlay需要封装/解封装，增加CPU和延迟
2. **MTU设置**：封装会增加包头，需要调整MTU避免分片
3. **路由表大小**：大规模集群路由表会很大
4. **网络策略**：策略越复杂，性能开销越大

#### 实际排查和验证

**1. 查看Pod网络配置**

```bash
# 进入Pod查看网络
kubectl exec -it <pod-name> -- ip addr
kubectl exec -it <pod-name> -- ip route

# 查看Pod的网络命名空间
docker inspect <container-id> | grep NetworkMode
```

**2. 查看Node网络配置**

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

**3. 测试跨节点连通性**

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

**4. 性能测试**

```bash
# 使用iperf3测试带宽
# Server端（Pod B）
kubectl exec -it pod-b -- iperf3 -s

# Client端（Pod A）
kubectl exec -it pod-a -- iperf3 -c 10.244.2.20

# 测试延迟
kubectl exec -it pod-a -- ping -c 100 10.244.2.20 | grep avg
```

#### 常见问题和调试

**问题1：Pod无法跨节点通信**

**排查步骤**：
```bash
# 1. 检查CNI插件是否正常
kubectl get pods -n kube-system | grep -E 'flannel|calico|cilium'

# 2. 检查路由
ip route | grep 10.244

# 3. 检查防火墙规则
# Flannel VXLAN需要UDP 8472端口
firewall-cmd --list-all
iptables -L -n | grep 8472

# 4. 检查网络策略
kubectl get networkpolicy --all-namespaces

# 5. 检查节点之间连通性
ping <other-node-ip>
```

**问题2：Service无法访问**

```bash
# 1. 检查kube-proxy
kubectl get pods -n kube-system | grep kube-proxy
kubectl logs -n kube-system kube-proxy-xxxxx

# 2. 检查Service和Endpoints
kubectl get svc
kubectl get endpoints <service-name>

# 3. 检查iptables/IPVS规则
iptables -t nat -L KUBE-SERVICES -n | grep <service-ip>
ipvsadm -Ln | grep <service-ip>
```

**问题3：网络性能差**

```bash
# 1. 检查MTU设置
ip link show | grep mtu

# 2. 调整MTU（对于VXLAN，通常设置为1450）
ip link set dev eth0 mtu 1500
ip link set dev cni0 mtu 1450
ip link set dev flannel.1 mtu 1450

# 3. 使用Host-Gateway或BGP模式（避免封装）

# 4. 启用网卡offload功能
ethtool -K eth0 tso on gso on
```

#### 网络策略示例

```yaml
# 限制Pod B只能被特定Pod访问
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-pod-a
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: pod-b
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: pod-a
    ports:
    - protocol: TCP
      port: 8080
```

---

### 相关面试题

#### Q1: Overlay网络和路由模式的区别是什么？各有什么优缺点？

**答案**：

**Overlay网络（如Flannel VXLAN、Calico IPIP）**：
- **原理**：在现有网络上构建虚拟网络，通过封装技术（VXLAN、IPIP）传输
- **优点**：
  - 不依赖底层网络，灵活性高
  - 可跨子网、跨数据中心
  - 配置简单，无需修改网络设备
- **缺点**：
  - 封装/解封装有性能开销（5-15%）
  - MTU问题，可能需要调整
  - 调试复杂，封装后不易追踪

**路由模式（如Flannel Host-gw、Calico BGP）**：
- **原理**：通过路由协议（BGP）或静态路由，直接路由Pod流量
- **优点**：
  - 性能最优，无封装开销
  - 网络调试简单，直接看路由表
  - 与物理网络集成好
- **缺点**：
  - 要求节点二层可达或网络设备支持BGP
  - 大规模集群路由表膨胀
  - 对网络环境有要求

**选择建议**：
- **云环境/跨子网**：Overlay（VXLAN）
- **私有云/同子网**：路由模式（BGP/Host-gw）
- **混合方案**：Calico支持混合（同子网BGP，跨子网IPIP）

#### Q2: kube-proxy的iptables模式和IPVS模式有什么区别？

**答案**：

**iptables模式**：
- **工作原理**：使用iptables规则实现Service负载均衡
- **负载均衡**：随机选择（基于概率）
- **性能**：
  - 规则数量O(n)，Service增多性能下降
  - 1000个Service约有几万条iptables规则
  - 匹配是顺序遍历，时间复杂度高
- **优点**：兼容性好，无需额外内核模块
- **缺点**：规则多时性能差，难以调试

**IPVS模式**：
- **工作原理**：使用IPVS（Linux内核负载均衡器）
- **负载均衡算法**：
  - rr（轮询）
  - lc（最少连接）
  - dh（目标地址哈希）
  - sh（源地址哈希）
  - sed（最短期望延迟）
- **性能**：
  - 使用哈希表，时间复杂度O(1)
  - 可处理大量Service（10000+）
  - 性能提升10-100倍
- **优点**：高性能，丰富的负载均衡算法
- **缺点**：需要IPVS内核模块，调试稍复杂

**对比表**：
| 特性         | iptables      | IPVS           |
| ------------ | ------------- | -------------- |
| 性能         | 中            | 高             |
| 规则复杂度   | O(n)          | O(1)           |
| 负载均衡算法 | 随机          | 8种算法        |
| 适用规模     | <1000 Service | 10000+ Service |
| 成熟度       | 高            | 较高           |

**启用IPVS**：
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
      scheduler: "rr"  # 轮询算法
```

#### Q3: 什么是veth pair？它在Kubernetes网络中的作用是什么？

**答案**：

**veth pair（Virtual Ethernet Pair）**：
- 一对虚拟网络设备，像一根虚拟网线
- 从一端进入的数据包会立即从另一端出来
- 常用于连接不同的网络命名空间

**在Kubernetes中的作用**：
```
Container Network Namespace  ←─── veth pair ───→  Host Network Namespace
        eth0                                              vethxxxx
   (10.244.1.10)                                      (连接到cni0网桥)
```

**工作原理**：
1. **创建Pod时**：
   - CNI插件创建网络命名空间
   - 创建veth pair
   - 一端（eth0）放入Pod的网络命名空间
   - 另一端（vethxxxx）留在主机，连接到网桥

2. **数据传输**：
   - Pod发送数据到eth0
   - 数据包从vethxxxx端出来
   - 通过网桥转发到目标

**查看veth pair**：
```bash
# 在主机上查看
ip link | grep veth

# 查看veth对应的Pod
for veth in $(ip link | grep veth | awk '{print $2}' | cut -d@ -f1); do
    echo "=== $veth ==="
    ethtool -S $veth | grep peer_ifindex
done

# 在Pod内查看
kubectl exec -it <pod> -- ip link
```

**特点**：
- 零拷贝，性能高
- 连接网络命名空间的标准方式
- 隔离性好，每个Pod有独立的网络栈

#### Q4: 如何解决Pod间通信的MTU问题？

**答案**：

**MTU（Maximum Transmission Unit，最大传输单元）问题**：

**产生原因**：
- Overlay网络（VXLAN、IPIP）增加额外的包头
- VXLAN增加50字节（14字节Ethernet + 8字节UDP + 8字节VXLAN + 20字节IP）
- 如果不调整MTU，会导致数据包分片，影响性能

**标准MTU**：
```
物理网卡 (eth0): 1500字节
VXLAN封装后需要: 1500 - 50 = 1450字节
因此容器网卡应设置: 1450字节
```

**解决方案**：

**方法1：调整Pod网络MTU**
```bash
# 在Node上调整cni0和VXLAN设备的MTU
ip link set dev cni0 mtu 1450
ip link set dev flannel.1 mtu 1450

# CNI配置中设置MTU
# /etc/cni/net.d/10-flannel.conflist
{
  "name": "cbr0",
  "cniVersion": "0.3.1",
  "plugins": [
    {
      "type": "flannel",
      "delegate": {
        "isDefaultGateway": true,
        "mtu": 1450
      }
    }
  ]
}
```

**方法2：启用Path MTU Discovery**
```bash
# 内核参数
sysctl -w net.ipv4.ip_no_pmtu_disc=0
```

**方法3：增加物理网络MTU（Jumbo Frame）**
```bash
# 如果网络支持，增加物理网卡MTU到9000
ip link set dev eth0 mtu 9000
# 这样容器网卡可以使用更大的MTU
```

**方法4：使用无封装的网络方案**
- Host-Gateway模式
- Calico BGP模式
- 无需封装，无MTU问题

**检测MTU问题**：
```bash
# 测试MTU
# -M do: 禁止分片
# -s 1472: 数据大小（1500 - 28字节IP+ICMP头）
ping -M do -s 1472 <target-ip>

# 如果失败，逐步减小直到成功
ping -M do -s 1422 <target-ip>  # 1450 MTU
```

**Flannel自动MTU检测**：
```yaml
# Flannel DaemonSet
env:
- name: FLANNEL_MTU
  value: "auto"  # 自动检测MTU
```

#### Q5: NetworkPolicy是如何实现的？它在哪一层生效？

**答案**：

**NetworkPolicy实现机制**：

**1. API层面**：
- Kubernetes资源，定义Pod间的访问规则
- 声明式配置，指定允许/拒绝的流量

**2. 实现层面**：
- **由CNI插件实现**（不是kube-proxy）
- Calico：使用iptables或eBPF
- Cilium：使用eBPF
- Flannel：不支持NetworkPolicy（需要配合Calico）

**3. 生效位置**：
- **在Pod所在的Node上生效**
- 在流量进入Pod之前进行过滤
- 双向控制：Ingress（入站）和Egress（出站）

**Calico实现原理**：
```bash
# Calico在每个Node上创建iptables规则
# 1. 在FORWARD链中插入规则
iptables -L FORWARD -n

# 2. 为每个Pod创建链
iptables -L cali-fw-<pod-hash> -n

# 3. 匹配NetworkPolicy规则
# 允许的流量：ACCEPT
# 拒绝的流量：DROP

# 示例规则
-A cali-fw-pod1 -s 10.244.1.0/24 -p tcp --dport 8080 -j ACCEPT
-A cali-fw-pod1 -j DROP  # 默认拒绝
```

**Cilium eBPF实现**：
```
更高效，在内核层面过滤
无需遍历iptables规则
性能更好，延迟更低
```

**NetworkPolicy示例**：
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: db-access
spec:
  podSelector:
    matchLabels:
      app: database
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: backend
    ports:
    - protocol: TCP
      port: 5432
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: monitoring
    ports:
    - protocol: TCP
      port: 9090
```

**特点**：
- **命名空间级别**：只影响指定命名空间的Pod
- **默认行为**：无Policy时全部允许，有Policy后默认拒绝
- **组合逻辑**：多个Policy的并集（OR关系）

**调试NetworkPolicy**：
```bash
# 查看NetworkPolicy
kubectl get networkpolicy
kubectl describe networkpolicy <name>

# Calico特定命令
calicoctl get networkpolicy -o yaml
calicoctl get globalnetworkpolicy

# 测试连通性
kubectl exec -it pod-a -- nc -zv pod-b 8080
```

#### Q6: 什么是CNI（Container Network Interface）？常见的CNI插件有哪些？

**答案**：

**CNI（Container Network Interface）**：
- **标准接口**：定义容器网络配置的标准
- **插件化**：通过插件实现具体网络方案
- **职责**：为容器配置网络接口、分配IP、设置路由

**CNI工作流程**：
```
1. kubelet创建Pod
2. 调用CNI插件（通过标准接口）
3. CNI插件执行：
   - 创建veth pair
   - 分配IP地址
   - 配置路由
   - 设置网桥/Overlay
4. 返回结果给kubelet
```

**常见CNI插件对比**：

| 插件            | 类型         | 特点                   | 适用场景                 |
| --------------- | ------------ | ---------------------- | ------------------------ |
| **Flannel**     | Overlay/路由 | 简单易用，文档丰富     | 中小规模，快速部署       |
| **Calico**      | 路由/Overlay | BGP路由，NetworkPolicy | 大规模，需要策略         |
| **Cilium**      | eBPF         | 高性能，高级功能       | 高性能要求，云原生       |
| **Weave**       | Overlay      | 自动网状网络，加密     | 简单部署，需要加密       |
| **Canal**       | 组合         | Flannel网络+Calico策略 | 需要策略但不想用纯Calico |
| **Antrea**      | Overlay      | VMware出品，OVS        | VMware环境               |
| **Kube-router** | 路由         | BGP+IPVS               | 简化架构                 |

**CNI配置文件**：
```json
// /etc/cni/net.d/10-flannel.conflist
{
  "name": "cbr0",
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

**CNI命令**：
```bash
# 查看CNI插件
ls /opt/cni/bin/

# 查看CNI配置
ls /etc/cni/net.d/

# 手动调用CNI（调试用）
echo '{"cniVersion":"0.3.1","name":"test","type":"bridge"}' | \
  /opt/cni/bin/bridge
```

**选择建议**：
- **新手/小规模**：Flannel（简单）
- **大规模/企业**：Calico（成熟、功能全）
- **高性能**：Cilium（eBPF）
- **云原生**：云厂商提供的CNI（AWS VPC CNI、Azure CNI）

#### Q7: Service的ClusterIP、NodePort、LoadBalancer有什么区别？网络流量如何转发？

**答案**：

**三种Service类型对比**：

**1. ClusterIP（默认）**：
- **访问范围**：仅集群内部可访问
- **IP地址**：虚拟IP（VIP），从Service CIDR分配
- **用途**：Pod间通信、内部服务

**流量路径**：
```
Pod A (10.244.1.10)
    ↓ 访问 Service ClusterIP
Service (10.96.0.100:80)
    ↓ kube-proxy DNAT转换
Backend Pod B (10.244.2.20:8080)
    ↓ 跨节点访问
通过CNI网络到达Pod B
```

**2. NodePort**：
- **访问范围**：通过任意节点IP+NodePort访问
- **端口范围**：30000-32767（可配置）
- **用途**：对外暴露服务（简单场景）

**流量路径**：
```
外部客户端
    ↓ 访问 NodeIP:NodePort
Node2 (192.168.1.11:31000)
    ↓ kube-proxy DNAT
Service (10.96.0.100:80)
    ↓ 再次DNAT
Backend Pod (10.244.1.10:8080)
    ↓ 可能在其他节点
跨节点转发到Pod
    ↓ 响应包
SNAT转换源地址为NodeIP
    ↓
返回客户端
```

**3. LoadBalancer**：
- **访问范围**：通过云厂商LoadBalancer访问
- **实现**：依赖云平台（AWS ELB、Azure LB等）
- **用途**：生产环境对外服务

**流量路径**：
```
外部客户端
    ↓
云Load Balancer (公网IP)
    ↓ LB转发到某个Node
NodeIP:NodePort
    ↓ 后续同NodePort
Service → Pod
```

**配置示例**：

```yaml
# ClusterIP
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP  # 默认
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080

---
# NodePort
apiVersion: v1
kind: Service
metadata:
  name: my-service-nodeport
spec:
  type: NodePort
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 31000  # 可选，不指定则自动分配

---
# LoadBalancer
apiVersion: v1
kind: Service
metadata:
  name: my-service-lb
spec:
  type: LoadBalancer
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
```

**kube-proxy转发细节**：

```bash
# iptables规则链路
PREROUTING → KUBE-SERVICES
            ↓
         KUBE-SVC-XXX (Service)
            ↓
       负载均衡选择
            ↓
    ┌───────┴───────┐
    ↓               ↓
KUBE-SEP-1    KUBE-SEP-2  (Endpoints)
    ↓               ↓
 DNAT转换     DNAT转换
    ↓               ↓
 Pod1:8080    Pod2:8080
```

**Session亲和性**：
```yaml
# 保持会话到同一Pod
spec:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800  # 3小时
```

#### Q8: 什么是Headless Service？它与普通Service有什么区别？

**答案**：

**Headless Service**：
- **定义**：不分配ClusterIP的Service（ClusterIP: None）
- **特点**：不做负载均衡，DNS直接返回Pod IP列表
- **用途**：StatefulSet、服务发现、客户端自己选择Pod

**与普通Service对比**：

| 特性      | 普通Service   | Headless Service     |
| --------- | ------------- | -------------------- |
| ClusterIP | 有（VIP）     | None                 |
| 负载均衡  | kube-proxy    | 无（DNS返回所有Pod） |
| DNS解析   | 返回ClusterIP | 返回所有Pod IP       |
| 用途      | 负载均衡      | 服务发现、有状态应用 |

**配置示例**：
```yaml
# Headless Service
apiVersion: v1
kind: Service
metadata:
  name: my-headless-service
spec:
  clusterIP: None  # 关键：设置为None
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
```

**DNS解析差异**：

```bash
# 普通Service DNS解析
nslookup my-service.default.svc.cluster.local
# 返回：
# Name:   my-service.default.svc.cluster.local
# Address: 10.96.0.100  # ClusterIP

# Headless Service DNS解析
nslookup my-headless-service.default.svc.cluster.local
# 返回：
# Name:   my-headless-service.default.svc.cluster.local
# Address: 10.244.1.10  # Pod1 IP
# Address: 10.244.2.20  # Pod2 IP
# Address: 10.244.3.30  # Pod3 IP
```

**StatefulSet + Headless Service**：

```yaml
# StatefulSet使用Headless Service
apiVersion: v1
kind: Service
metadata:
  name: mysql
spec:
  clusterIP: None
  selector:
    app: mysql
  ports:
  - port: 3306

---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  serviceName: mysql  # 关联Headless Service
  replicas: 3
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      containers:
      - name: mysql
        image: mysql:5.7
```

**每个Pod的DNS**：
```bash
# StatefulSet Pod有稳定的DNS名称
mysql-0.mysql.default.svc.cluster.local  → 10.244.1.10
mysql-1.mysql.default.svc.cluster.local  → 10.244.2.20
mysql-2.mysql.default.svc.cluster.local  → 10.244.3.30

# 可以直接访问特定Pod
mysql -h mysql-0.mysql.default.svc.cluster.local
```

**使用场景**：
1. **数据库集群**：主从复制，需要访问特定Pod
2. **消息队列**：客户端需要知道所有broker
3. **分布式缓存**：一致性哈希，客户端选择节点
4. **服务发现**：获取所有服务实例列表

---

### 关键点总结

**Pod跨节点通信核心流程**：
1. **Pod内**：容器eth0 → veth pair
2. **Node内**：veth → 网桥(cni0) → Node路由
3. **跨节点**：封装（Overlay）或直接路由
4. **目标Node**：解封装 → 网桥 → veth pair → Pod

**CNI插件选择**：
- **简单部署**：Flannel VXLAN
- **高性能**：Calico BGP / Flannel Host-gw
- **需要策略**：Calico
- **最高性能**：Cilium eBPF

**Service访问**：
- kube-proxy DNAT转换
- ClusterIP → Pod IP
- 跨节点按Pod通信流程转发

**关键技术**：
- **veth pair**：连接容器和主机
- **网桥(cni0)**：本地Pod通信
- **Overlay（VXLAN）**：跨子网封装
- **BGP路由**：高性能路由模式
- **iptables/IPVS**：Service负载均衡
- **NetworkPolicy**：网络隔离和安全
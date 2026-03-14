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
  - 故障排查
---

# Kubernetes Service 访问异常排查完全指南

## 引言

在 Kubernetes 集群中，Service 作为核心的资源对象，为 Pod 提供了稳定的服务发现和负载均衡能力。然而，在实际生产环境中，Service 访问异常是最常见的问题之一，直接影响应用的可用性和用户体验。当 Service 无法正常工作时，可能导致服务间调用失败、外部流量无法进入集群、负载分配不均等问题，严重时甚至引发整个系统的级联故障。

本文将深入分析 Kubernetes Service 访问异常的五大常见问题，从原理层面剖析问题根源，提供系统化的排查思路和解决方案，帮助运维人员和开发者快速定位并解决问题。

## Service 工作原理回顾

在深入排查之前，我们需要理解 Service 的工作机制。Service 通过 Label Selector 将请求路由到后端 Pod，其核心组件包括：

- **ClusterIP**：虚拟 IP，仅在集群内部可访问
- **kube-proxy**：负责实现 Service 的负载均衡规则
- **iptables/IPVS**：底层的流量转发规则
- **Endpoints**：记录后端 Pod 的 IP 和端口信息
- **CoreDNS**：提供 Service 的域名解析服务

当访问 Service 时，流量路径为：Client → Service ClusterIP → kube-proxy 规则 → Endpoints → Pod。任何一个环节出现问题，都会导致访问异常。

## 一、Endpoints 为空

### 现象描述

执行 `kubectl get endpoints <service-name>` 命令时，发现 Endpoints 列表为空，或者 Endpoints 的 ADDRESS 列没有任何 IP 地址。此时，通过 Service 访问应用会直接失败，提示连接被拒绝或超时。

```bash
$ kubectl get endpoints my-service
NAME        ENDPOINTS   AGE
my-service  <none>      5m
```

### 原因分析

Endpoints 为空的根本原因是 Service 无法找到匹配的 Pod。具体原因包括：

1. **Label Selector 不匹配**：Service 的 selector 与 Pod 的 labels 不一致
2. **Pod 未运行**：Pod 处于 Pending、CrashLoopBackOff 等非 Running 状态
3. **Pod 未就绪**：Pod 运行但未通过 Readiness Probe 检查
4. **命名空间不一致**：Service 和 Pod 位于不同的 namespace
5. **Service 端口配置错误**：Service 的 targetPort 与 Pod 的 containerPort 不匹配

### 排查步骤

**步骤 1：检查 Service 的 Label Selector**

```bash
# 查看 Service 的 selector
kubectl get svc my-service -o yaml | grep -A 5 selector

# 查看所有 Pod 的 labels
kubectl get pods --show-labels
```

**步骤 2：验证 Pod 状态**

```bash
# 检查 Pod 是否正常运行
kubectl get pods -l app=my-app

# 查看 Pod 详细信息
kubectl describe pod <pod-name>
```

**步骤 3：检查 Readiness Probe**

```bash
# 查看 Pod 的就绪状态
kubectl get pods -o wide

# 检查 Pod 的事件日志
kubectl describe pod <pod-name> | grep -A 10 Events
```

**步骤 4：确认端口配置**

```bash
# 查看 Service 端口配置
kubectl get svc my-service -o yaml | grep -A 10 ports

# 查看 Pod 容器端口
kubectl get pod <pod-name> -o yaml | grep -A 5 ports
```

### 解决方案

根据排查结果，采取相应的解决措施：

| 问题原因 | 解决方案 |
|---------|---------|
| Label Selector 不匹配 | 修改 Service 的 selector 或 Pod 的 labels，确保两者匹配 |
| Pod 未运行 | 检查 Pod 日志，修复应用启动问题或资源限制 |
| Pod 未就绪 | 调整 Readiness Probe 配置，检查应用健康状态 |
| 命名空间不一致 | 在同一 namespace 创建 Service 和 Pod，或使用跨 namespace 访问方式 |
| 端口配置错误 | 修正 Service 的 targetPort 或 Pod 的 containerPort |

**示例：修复 Label Selector 不匹配问题**

```yaml
# 原始 Service 配置（selector 错误）
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  selector:
    app: my-app-v1  # 错误的 label
  ports:
  - port: 80
    targetPort: 8080

# 修正后的 Service 配置
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  selector:
    app: my-app     # 正确的 label
  ports:
  - port: 80
    targetPort: 8080
```

## 二、ClusterIP 无法访问

### 现象描述

Service 的 ClusterIP 存在，但通过该 IP 访问服务时，连接超时或被拒绝。即使 Endpoints 正常，流量仍然无法到达后端 Pod。

```bash
# Service ClusterIP 存在
$ kubectl get svc my-service
NAME        TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
my-service  ClusterIP   10.96.100.100   <none>        80/TCP    5m

# 访问 ClusterIP 失败
$ curl 10.96.100.100:80
curl: (7) Failed to connect to 10.96.100.100 port 80: Connection timed out
```

### 原因分析

ClusterIP 无法访问的原因涉及网络多个层面：

1. **kube-proxy 异常**：kube-proxy 未正确运行或配置错误
2. **iptables/IPVS 规则缺失**：流量转发规则未正确生成
3. **网络插件问题**：CNI 插件配置错误或运行异常
4. **节点网络配置**：节点防火墙规则阻止流量
5. **Service CIDR 冲突**：ClusterIP 网段与物理网络冲突

### 排查步骤

**步骤 1：检查 kube-proxy 状态**

```bash
# 查看 kube-proxy Pod 状态
kubectl get pods -n kube-system -l k8s-app=kube-proxy

# 查看 kube-proxy 日志
kubectl logs -n kube-system <kube-proxy-pod-name>

# 检查 kube-proxy 配置
kubectl get configmap -n kube-system kube-proxy -o yaml
```

**步骤 2：验证 iptables/IPVS 规则**

```bash
# 查看 iptables 规则（iptables 模式）
sudo iptables -t nat -L KUBE-SERVICES -n -v

# 查看 IPVS 规则（IPVS 模式）
sudo ipvsadm -Ln

# 检查 Service 相关规则
sudo iptables -t nat -L KUBE-SVC-<hash> -n -v
```

**步骤 3：检查网络插件状态**

```bash
# 查看网络插件 Pod 状态（以 Calico 为例）
kubectl get pods -n kube-system -l k8s-app=calico-node

# 查看网络插件日志
kubectl logs -n kube-system <calico-pod-name>

# 检查节点网络接口
ip addr show
```

**步骤 4：测试网络连通性**

```bash
# 从节点直接访问 Pod IP
ping <pod-ip>
curl http://<pod-ip>:<pod-port>

# 检查节点防火墙规则
sudo iptables -L -n -v | grep DROP
```

### 解决方案

针对不同原因采取相应措施：

| 问题原因 | 解决方案 |
|---------|---------|
| kube-proxy 异常 | 重启 kube-proxy Pod，检查 kube-proxy 配置，确保与集群版本兼容 |
| iptables/IPVS 规则缺失 | 重启 kube-proxy 触发规则重建，检查 proxy-mode 配置 |
| 网络插件问题 | 检查 CNI 配置文件，重启网络插件，查看网络插件日志 |
| 节点防火墙阻止 | 调整防火墙规则，开放 Service 网段和 Pod 网段 |
| Service CIDR 冲突 | 重新规划集群网络，修改 kube-apiserver 的 --service-cluster-ip-range 参数 |

**示例：重启 kube-proxy**

```bash
# 删除 kube-proxy Pod，由 DaemonSet 自动重建
kubectl delete pods -n kube-system -l k8s-app=kube-proxy

# 等待 Pod 重建完成
kubectl get pods -n kube-system -l k8s-app=kube-proxy -w
```

**示例：检查并修复 iptables 规则**

```bash
# 查看完整的 Service 链规则
sudo iptables -t nat -L KUBE-SERVICES -n -v --line-numbers

# 如果规则缺失，重启 kube-proxy
systemctl restart kube-proxy  # 或通过删除 Pod 方式

# 验证规则是否生成
sudo iptables -t nat -S | grep <service-cluster-ip>
```

## 三、Service DNS 解析失败

### 现象描述

在集群内部通过 Service 域名访问服务时，DNS 解析失败，提示域名无法找到或解析超时。但通过 ClusterIP 访问正常。

```bash
# DNS 解析失败
$ nslookup my-service.default.svc.cluster.local
Server:     10.96.0.10
Address:    10.96.0.10#53

** server can't find my-service.default.svc.cluster.local: NXDOMAIN

# 直接访问 ClusterIP 正常
$ curl 10.96.100.100:80
Hello World
```

### 原因分析

Service DNS 解析失败主要与 CoreDNS 相关：

1. **CoreDNS Pod 异常**：CoreDNS 未运行或处于错误状态
2. **CoreDNS 配置错误**：CoreDNS ConfigMap 配置不正确
3. **DNS Service 不存在**：kube-dns Service 被误删或配置错误
4. **Pod DNS 配置错误**：Pod 的 dnsPolicy 或 dnsConfig 配置不当
5. **上游 DNS 问题**：外部域名解析时上游 DNS 不可达

### 排查步骤

**步骤 1：检查 CoreDNS 状态**

```bash
# 查看 CoreDNS Pod 状态
kubectl get pods -n kube-system -l k8s-app=kube-dns

# 查看 CoreDNS 日志
kubectl logs -n kube-system <coredns-pod-name>

# 查看 CoreDNS Service
kubectl get svc -n kube-system kube-dns
```

**步骤 2：验证 CoreDNS 配置**

```bash
# 查看 CoreDNS ConfigMap
kubectl get configmap -n kube-system coredns -o yaml

# 检查 CoreDNS 配置文件格式
kubectl get configmap -n kube-system coredns -o jsonpath='{.data.Corefile}'
```

**步骤 3：测试 DNS 解析**

```bash
# 创建测试 Pod 进行 DNS 解析
kubectl run dns-test --image=busybox:1.28 --rm -it --restart=Never -- nslookup kubernetes.default

# 从 Pod 内部测试 Service 解析
kubectl exec -it <pod-name> -- nslookup my-service.default.svc.cluster.local

# 查看 Pod 的 DNS 配置
kubectl exec -it <pod-name> -- cat /etc/resolv.conf
```

**步骤 4：检查 DNS 策略**

```bash
# 查看 Pod 的 DNS 策略
kubectl get pod <pod-name> -o yaml | grep -A 5 dnsPolicy

# 查看 Pod 的 DNS 配置
kubectl get pod <pod-name> -o yaml | grep -A 10 dnsConfig
```

### 解决方案

| 问题原因 | 解决方案 |
|---------|---------|
| CoreDNS Pod 异常 | 重启 CoreDNS Pod，检查资源限制和节点状态 |
| CoreDNS 配置错误 | 修正 CoreDNS ConfigMap，确保 Corefile 格式正确 |
| DNS Service 不存在 | 重新创建 kube-dns Service，确保 ClusterIP 正确 |
| Pod DNS 配置错误 | 调整 Pod 的 dnsPolicy 为 ClusterFirst，检查 dnsConfig |
| 上游 DNS 问题 | 配置正确的上游 DNS 服务器，检查网络连通性 |

**示例：修复 CoreDNS 配置**

```yaml
# CoreDNS ConfigMap 示例
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    .:53 {
        errors
        health {
           lameduck 5s
        }
        ready
        kubernetes cluster.local in-addr.arpa ip6.arpa {
           pods insecure
           fallthrough in-addr.arpa ip6.arpa
           ttl 30
        }
        prometheus :9153
        forward . /etc/resolv.conf {
           max_concurrent 1000
        }
        cache 30
        loop
        reload
        loadbalance
    }
```

**示例：Pod DNS 策略配置**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  dnsPolicy: ClusterFirst  # 优先使用集群 DNS
  containers:
  - name: my-container
    image: my-image
```

## 四、负载均衡不生效

### 现象描述

Service 后端有多个 Pod，但访问 Service 时，流量始终路由到同一个 Pod，其他 Pod 未接收到请求。负载均衡失效导致部分 Pod 负载过高，部分 Pod 闲置。

```bash
# 多个 Pod 正常运行
$ kubectl get pods -l app=my-app
NAME                      READY   STATUS    RESTARTS   AGE
my-app-5d8b9f7c6b-abcde   1/1     Running   0          5m
my-app-5d8b9f7c6b-fghij   1/1     Running   0          5m
my-app-5d8b9f7c6b-klmno   1/1     Running   0          5m

# 访问 Service 多次，始终返回同一个 Pod
$ for i in {1..10}; do curl http://my-service:80/pod-name; done
my-app-5d8b9f7c6b-abcde
my-app-5d8b9f7c6b-abcde
my-app-5d8b9f7c6b-abcde
...
```

### 原因分析

负载均衡不生效的原因涉及多个层面：

1. **Service 配置问题**：配置了 sessionAffinity 导致会话保持
2. **kube-proxy 模式问题**：iptables 模式的随机算法问题
3. **客户端连接复用**：客户端使用长连接，未重新建立连接
4. **Pod 就绪状态**：部分 Pod 未通过 Readiness Probe
5. **IPVS 配置问题**：IPVS 调度算法配置不当

### 排查步骤

**步骤 1：检查 Service 配置**

```bash
# 查看 Service 配置
kubectl get svc my-service -o yaml

# 检查 sessionAffinity 配置
kubectl get svc my-service -o jsonpath='{.spec.sessionAffinity}'

# 查看 Endpoints 分布
kubectl get endpoints my-service -o yaml
```

**步骤 2：验证 kube-proxy 模式**

```bash
# 查看 kube-proxy 配置
kubectl get configmap -n kube-system kube-proxy -o yaml | grep mode

# 检查当前使用的代理模式
kubectl logs -n kube-system <kube-proxy-pod-name> | grep -i "using.*mode"
```

**步骤 3：检查 IPVS 调度算法**

```bash
# 查看 IPVS 规则和调度算法
sudo ipvsadm -Ln

# 查看具体 Service 的调度算法
sudo ipvsadm -Ln | grep -A 5 <cluster-ip>
```

**步骤 4：测试负载均衡效果**

```bash
# 使用多个并发请求测试
for i in {1..20}; do
  curl -s http://my-service:80/pod-name &
done
wait

# 查看各 Pod 的请求计数
kubectl exec -it <pod-name> -- curl localhost:8080/metrics | grep request_count
```

### 解决方案

| 问题原因 | 解决方案 |
|---------|---------|
| sessionAffinity 配置 | 移除或调整 sessionAffinity 配置，设置为 None |
| iptables 模式问题 | 切换到 IPVS 模式，获得更好的负载均衡算法支持 |
| 客户端长连接 | 客户端实现连接池轮询，或定期重建连接 |
| Pod 未就绪 | 修复 Readiness Probe，确保所有 Pod 正常就绪 |
| IPVS 调度算法 | 配置合适的调度算法（rr、lc、dh、sh 等） |

**示例：移除 sessionAffinity 配置**

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
  sessionAffinity: None  # 禁用会话保持
```

**示例：配置 IPVS 调度算法**

```yaml
# kube-proxy ConfigMap 配置
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-proxy
  namespace: kube-system
data:
  config.conf: |
    apiVersion: kubeproxy.config.k8s.io/v1alpha1
    kind: KubeProxyConfiguration
    mode: ipvs
    ipvs:
      scheduler: rr  # 轮询调度算法
```

**示例：客户端连接轮询实现**

```python
# Python 示例：使用连接池轮询
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

session = requests.Session()
adapter = HTTPAdapter(pool_connections=10, pool_maxsize=10, max_retries=0)
session.mount('http://', adapter)

# 每次请求使用不同的连接
for i in range(10):
    response = session.get('http://my-service:80/api')
    print(response.text)
```

## 五、会话保持失效

### 现象描述

Service 配置了 sessionAffinity，期望同一客户端的请求始终路由到同一个 Pod，但实际上请求被分发到不同的 Pod，导致会话状态丢失。

```bash
# Service 配置了会话保持
$ kubectl get svc my-service -o yaml | grep -A 2 sessionAffinity
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800

# 同一客户端多次请求，返回不同的 Pod
$ for i in {1..5}; do curl http://my-service:80/pod-name; done
my-app-5d8b9f7c6b-abcde
my-app-5d8b9f7c6b-fghij
my-app-5d8b9f7c6b-klmno
my-app-5d8b9f7c6b-abcde
my-app-5d8b9f7c6b-fghij
```

### 原因分析

会话保持失效的原因包括：

1. **客户端 IP 变化**：请求经过代理或负载均衡器，源 IP 被修改
2. **externalTrafficPolicy 配置**：Service 的 externalTrafficPolicy 设置不当
3. **超时时间过短**：sessionAffinity 的 timeoutSeconds 设置过短
4. **kube-proxy 实现问题**：iptables 模式的会话保持实现有缺陷
5. **多 Service 访问**：客户端通过不同 Service 访问，导致会话不一致

### 排查步骤

**步骤 1：检查会话保持配置**

```bash
# 查看 Service 会话保持配置
kubectl get svc my-service -o yaml | grep -A 5 sessionAffinity

# 检查超时时间
kubectl get svc my-service -o jsonpath='{.spec.sessionAffinityConfig.clientIP.timeoutSeconds}'
```

**步骤 2：验证客户端 IP**

```bash
# 在 Pod 中查看请求的源 IP
kubectl logs <pod-name> | grep "Client IP"

# 检查是否经过代理
kubectl exec -it <pod-name> -- env | grep -i proxy
```

**步骤 3：检查 externalTrafficPolicy**

```bash
# 查看 Service 的 externalTrafficPolicy
kubectl get svc my-service -o jsonpath='{.spec.externalTrafficPolicy}'

# 查看 Service 类型
kubectl get svc my-service -o jsonpath='{.spec.type}'
```

**步骤 4：测试会话保持效果**

```bash
# 使用固定客户端 IP 测试
for i in {1..10}; do
  curl -H "X-Forwarded-For: 192.168.1.100" http://my-service:80/pod-name
done
```

### 解决方案

| 问题原因 | 解决方案 |
|---------|---------|
| 客户端 IP 变化 | 配置 externalTrafficPolicy: Local，保留源 IP |
| externalTrafficPolicy 配置 | 对于 NodePort/LoadBalancer 类型，设置 externalTrafficPolicy: Local |
| 超时时间过短 | 增加 timeoutSeconds 值，默认 10800 秒（3 小时） |
| kube-proxy 实现问题 | 切换到 IPVS 模式，会话保持更稳定 |
| 多 Service 访问 | 统一使用同一 Service 访问，或在应用层实现会话一致性 |

**示例：配置 externalTrafficPolicy 保留源 IP**

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: NodePort
  externalTrafficPolicy: Local  # 保留客户端源 IP
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30080
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800
```

**示例：使用 IPVS 模式增强会话保持**

```yaml
# kube-proxy ConfigMap 配置
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-proxy
  namespace: kube-system
data:
  config.conf: |
    apiVersion: kubeproxy.config.k8s.io/v1alpha1
    kind: KubeProxyConfiguration
    mode: ipvs
    ipvs:
      scheduler: rr
      syncPeriod: 30s
      minSyncPeriod: 5s
```

**示例：应用层实现会话一致性**

```nginx
# Nginx 配置基于 Cookie 的会话保持
upstream backend {
    server pod1:8080;
    server pod2:8080;
    server pod3:8080;
}

server {
    location / {
        proxy_pass http://backend;
        proxy_cookie_path / /;
        # 使用 sticky cookie 实现会话保持
        add_header Set-Cookie "backend_server=$upstream_addr; Path=/";
    }
}
```

## Service 访问异常排查流程图

```
┌─────────────────────────────────────────────────────────────┐
│                 Service 访问异常排查流程                      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
                ┌───────────────────────┐
                │  Service 是否存在？    │
                └───────────────────────┘
                     │           │
                    是           否
                     │           │
                     │           ▼
                     │    创建 Service
                     │
                     ▼
          ┌─────────────────────────┐
          │  Endpoints 是否为空？    │
          └─────────────────────────┘
               │           │
              否           是
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查 Label       │
               │    │ Selector         │
               │    └──────────────────┘
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查 Pod 状态     │
               │    └──────────────────┘
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查 Readiness   │
               │    │ Probe            │
               │    └──────────────────┘
               │
               ▼
          ┌─────────────────────────┐
          │  DNS 解析是否正常？      │
          └─────────────────────────┘
               │           │
              是           否
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查 CoreDNS     │
               │    │ 状态             │
               │    └──────────────────┘
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查 DNS 配置     │
               │    └──────────────────┘
               │
               ▼
          ┌─────────────────────────┐
          │  ClusterIP 是否可访问？  │
          └─────────────────────────┘
               │           │
              是           否
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查 kube-proxy  │
               │    └──────────────────┘
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查 iptables/   │
               │    │ IPVS 规则        │
               │    └──────────────────┘
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查网络插件     │
               │    └──────────────────┘
               │
               ▼
          ┌─────────────────────────┐
          │  负载均衡是否正常？      │
          └─────────────────────────┘
               │           │
              是           否
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查 session     │
               │    │ Affinity         │
               │    └──────────────────┘
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查 kube-proxy  │
               │    │ 模式             │
               │    └──────────────────┘
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查客户端连接   │
               │    └──────────────────┘
               │
               ▼
          ┌─────────────────────────┐
          │  会话保持是否生效？      │
          └─────────────────────────┘
               │           │
              是           否
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查客户端 IP    │
               │    └──────────────────┘
               │           │
               │           ▼
               │    ┌──────────────────┐
               │    │ 检查 external    │
               │    │ TrafficPolicy    │
               │    └──────────────────┘
               │
               ▼
          ┌─────────────────────────┐
          │     Service 正常工作     │
          └─────────────────────────┘
```

## 常见问题 FAQ

### Q1: Service 的 ClusterIP 是否可以 ping 通？

**A**: ClusterIP 是虚拟 IP，由 kube-proxy 在 iptables/IPVS 中创建规则，不能直接 ping 通。只有在访问 Service 端口时，流量才会被转发到后端 Pod。这是正常现象，不是故障。

### Q2: 为什么 Service 的 ClusterIP 网段不能与 Pod 网段重叠？

**A**: ClusterIP 网段和 Pod 网段必须在不同的 IP 地址空间，否则会导致路由冲突。ClusterIP 用于 Service 虚拟 IP，Pod 网段用于分配给实际 Pod 的 IP。两者重叠会导致网络不可达或路由错误。

### Q3: 如何选择 kube-proxy 的代理模式？

**A**: 
- **iptables 模式**：默认模式，性能较好，但规则更新时会有短暂延迟，适合小规模集群
- **IPVS 模式**：支持更多负载均衡算法，性能更优，规则更新更快，适合大规模集群
- **userspace 模式**：已废弃，不推荐使用

建议在生产环境使用 IPVS 模式。

### Q4: Service 的 targetPort 可以使用名称吗？

**A**: 可以。targetPort 支持使用端口号或端口名称。使用端口名称时，Pod 的 containerPort 必须定义相应的名称。这种方式更灵活，便于端口变更。

```yaml
# Pod 端口定义
ports:
- name: http
  containerPort: 8080

# Service 引用
ports:
- port: 80
  targetPort: http
```

### Q5: 如何实现跨 namespace 的 Service 访问？

**A**: Kubernetes 支持跨 namespace 访问 Service，使用完整的 Service 域名格式：`<service-name>.<namespace>.svc.cluster.local`。例如，访问 namespace `prod` 中的 Service `my-service`，使用 `my-service.prod.svc.cluster.local`。

## 最佳实践

### 1. Service 配置最佳实践

- **明确 Label Selector**：使用清晰、有意义的 label，避免歧义
- **配置健康检查**：为 Pod 配置 Readiness Probe 和 Liveness Probe
- **合理设置端口**：使用端口名称而非端口号，便于维护
- **资源限制**：为 Service 后端 Pod 设置合理的资源请求和限制

### 2. 网络配置最佳实践

- **网络规划**：提前规划 Service CIDR 和 Pod CIDR，避免冲突
- **网络插件选择**：根据集群规模和需求选择合适的 CNI 插件
- **防火墙规则**：确保节点防火墙允许 Service 和 Pod 网段流量
- **MTU 配置**：正确配置网络 MTU，避免分片问题

### 3. 监控和日志最佳实践

- **监控 CoreDNS**：监控 CoreDNS 的性能和可用性
- **监控 kube-proxy**：监控 kube-proxy 的运行状态和规则数量
- **日志收集**：收集 Service 相关组件的日志，便于排查问题
- **指标监控**：监控 Service 的请求量、延迟、错误率等指标

### 4. 故障排查最佳实践

- **分层排查**：从应用层、Service 层、网络层逐层排查
- **保留现场**：问题发生时保留日志、事件、配置等信息
- **文档记录**：记录常见问题和解决方案，形成知识库
- **定期演练**：定期进行故障演练，提高应急响应能力

## 总结

Kubernetes Service 访问异常是生产环境中常见的问题，涉及 Service 配置、DNS 解析、网络转发、负载均衡等多个层面。本文详细分析了 Endpoints 为空、ClusterIP 无法访问、DNS 解析失败、负载均衡不生效、会话保持失效五大常见问题的现象、原因、排查步骤和解决方案。

掌握系统化的排查方法和深入理解 Service 工作原理，能够帮助运维人员和开发者快速定位问题根源，采取有效的解决措施。在实际工作中，建议遵循最佳实践，建立完善的监控和日志体系，提高集群的稳定性和可维护性。

---

## 面试回答

**面试官问：请介绍一下 Kubernetes 中 Service 访问异常的常见问题及处理方法。**

**回答**：Kubernetes Service 访问异常是生产环境中最常见的问题之一，主要分为五大类。第一，Endpoints 为空，通常是因为 Label Selector 不匹配、Pod 未运行或未就绪、命名空间不一致等原因，需要检查 selector 配置、Pod 状态和 Readiness Probe。第二，ClusterIP 无法访问，涉及 kube-proxy 异常、iptables/IPVS 规则缺失、网络插件问题等，需要检查 kube-proxy 状态、验证转发规则、排查网络插件。第三，DNS 解析失败，主要是 CoreDNS 问题，需要检查 CoreDNS Pod 状态、ConfigMap 配置和 DNS 策略。第四，负载均衡不生效，可能是 sessionAffinity 配置、kube-proxy 模式或客户端长连接导致，需要调整配置或切换到 IPVS 模式。第五，会话保持失效，通常是因为客户端 IP 变化或 externalTrafficPolicy 配置不当，需要配置 Local 模式保留源 IP。排查时遵循分层原则，从应用层到网络层逐层定位，结合 kubectl 命令、日志分析和网络工具快速解决问题。建议在生产环境使用 IPVS 模式、配置完善的监控告警、定期进行故障演练，提高集群稳定性。

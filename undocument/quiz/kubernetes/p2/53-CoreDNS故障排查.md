---
date: 2026-03-12
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - CoreDNS
  - DNS
  - 故障排查
---

# CoreDNS 频繁重启故障排查完全指南

## 引言：CoreDNS 在 Kubernetes 中的关键角色

在 Kubernetes 集群中，CoreDNS 作为集群内部的 DNS 服务器，承担着服务发现和名称解析的核心职责。当 CoreDNS 出现频繁重启和报错时，会导致集群内服务无法正常通信，引发连锁故障。本文将深入剖析 CoreDNS 的工作机制，提供系统化的故障排查方法和解决方案。

CoreDNS 的主要职责包括：
- **服务发现**：解析 Service 名称到 ClusterIP
- **Pod 名称解析**：通过 Headless Service 解析 Pod IP
- **外部域名解析**：将集群外部域名请求转发到上游 DNS
- **自定义域名映射**：通过 CoreDNS 插件实现灵活的 DNS 策略

## 一、CoreDNS 架构与工作原理

### 1.1 核心架构组件

CoreDNS 采用模块化设计，由 CoreDNS 主程序和一系列插件组成：

| 组件 | 功能说明 | 关键作用 |
|------|---------|---------|
| **CoreDNS Pod** | 运行 DNS 服务的容器实例 | 处理 DNS 查询请求 |
| **Corefile** | CoreDNS 配置文件 | 定义 DNS 解析规则和插件链 |
| **kubernetes 插件** | 与 K8s API Server 交互 | 实现服务发现功能 |
| **forward 插件** | 转发外部 DNS 请求 | 处理集群外域名解析 |
| **cache 插件** | 缓存 DNS 响应 | 提升解析性能 |
| **loop 插件** | 检测 DNS 解析环路 | 防止无限循环 |

### 1.2 DNS 解析流程

```
客户端 Pod
    ↓
    ↓ DNS 查询请求 (Service 名称)
    ↓
CoreDNS Service (ClusterIP)
    ↓
CoreDNS Pod (随机选择)
    ↓
    ├─→ 集群内域名 → kubernetes 插件 → 查询 API Server → 返回结果
    └─→ 集群外域名 → forward 插件 → 上游 DNS 服务器 → 返回结果
```

### 1.3 CoreDNS 与 kube-dns 的区别

| 特性 | kube-dns (旧版) | CoreDNS (新版) |
|------|----------------|----------------|
| 架构 | 多容器 (dnsmasq + kubedns + sidecar) | 单容器 |
| 配置方式 | ConfigMap + 启动参数 | Corefile 配置文件 |
| 扩展性 | 有限 | 插件化架构，高度可扩展 |
| 性能 | 一般 | 更优，支持缓存优化 |
| 内存占用 | 较高 | 较低 |
| 灵活性 | 固定功能 | 可自定义插件链 |

## 二、CoreDNS 频繁重启的常见原因

### 2.1 资源限制问题

**内存不足 (OOMKilled)**

CoreDNS 默认资源限制较为保守，在高负载场景下容易被 OOM Killer 终止：

| 场景 | 内存需求 | 默认限制 | 风险等级 |
|------|---------|---------|---------|
| 小型集群 (< 50 节点) | 50-100Mi | 170Mi | 低 |
| 中型集群 (50-200 节点) | 100-200Mi | 170Mi | 中 |
| 大型集群 (> 200 节点) | 200-500Mi | 170Mi | 高 |
| 高 DNS 查询频率 | 300-800Mi | 170Mi | 极高 |

**CPU 限流导致超时**

CPU 资源不足会导致 DNS 查询超时，进而引发健康检查失败。

### 2.2 配置错误

**Corefile 语法错误**

配置文件格式错误会导致 CoreDNS 启动失败，进入 CrashLoopBackOff 状态。

**DNS 解析环路**

错误的 forward 配置可能导致 DNS 查询在 CoreDNS 和上游 DNS 之间无限循环：

```
CoreDNS → 上游 DNS → CoreDNS → 上游 DNS → ...
```

**上游 DNS 不可达**

forward 插件配置的上游 DNS 服务器无法访问，导致健康检查失败。

### 2.3 网络问题

**与 API Server 通信失败**

CoreDNS 需要与 API Server 通信以获取 Service 和 Pod 信息，网络策略或网络插件问题可能导致连接失败。

**Service CIDR 冲突**

集群 Service CIDR 与节点网络或 Pod 网络冲突，导致 CoreDNS 无法正常工作。

### 2.4 存储和配置问题

**ConfigMap 挂载失败**

CoreDNS 配置 ConfigMap 挂载异常，导致 Corefile 无法读取。

**节点存储压力**

节点磁盘压力导致容器运行时异常，引发 Pod 重启。

### 2.5 版本兼容性问题

**Kubernetes 版本不匹配**

CoreDNS 版本与 Kubernetes 版本不兼容，导致 API 调用失败。

**插件版本冲突**

自定义插件与 CoreDNS 主程序版本不兼容。

## 三、系统化排查步骤

### 3.1 初步诊断

**步骤 1：检查 CoreDNS Pod 状态**

```bash
# 查看 CoreDNS Pod 状态
kubectl get pods -n kube-system -l k8s-app=kube-dns -o wide

# 查看 Pod 详细信息
kubectl describe pod <coredns-pod-name> -n kube-system

# 查看 Pod 重启次数
kubectl get pods -n kube-system -l k8s-app=kube-dns \
  -o jsonpath='{.items[*].status.containerStatuses[0].restartCount}'
```

**步骤 2：分析 Pod 事件**

```bash
# 查看 Pod 事件日志
kubectl get events -n kube-system --field-selector involvedObject.name=<coredns-pod-name>

# 查看最近 1 小时的警告事件
kubectl get events -n kube-system --field-selector type=Warning --sort-by='.lastTimestamp'
```

**步骤 3：检查资源使用情况**

```bash
# 查看资源使用
kubectl top pods -n kube-system -l k8s-app=kube-dns

# 查看资源限制配置
kubectl get deployment coredns -n kube-system -o yaml | grep -A 10 resources
```

### 3.2 日志分析

**查看 CoreDNS 日志**

```bash
# 查看当前日志
kubectl logs -n kube-system <coredns-pod-name> --tail=100

# 查看前一个容器的日志（重启前）
kubectl logs -n kube-system <coredns-pod-name> --previous --tail=200

# 实时查看日志
kubectl logs -n kube-system <coredns-pod-name> -f
```

**启用详细日志**

修改 CoreDNS ConfigMap，启用 debug 插件：

```bash
kubectl edit configmap coredns -n kube-system
```

在 Corefile 中添加：

```
.:53 {
    errors
    debug          # 启用调试日志
    health
    ready
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
        ttl 30
    }
    prometheus :9153
    forward . /etc/resolv.conf
    cache 30
    loop
    reload
    loadbalance
}
```

### 3.3 配置验证

**检查 Corefile 配置**

```bash
# 查看 CoreDNS ConfigMap
kubectl get configmap coredns -n kube-system -o yaml

# 验证 Corefile 语法
kubectl exec -n kube-system <coredns-pod-name> -- cat /etc/coredns/Corefile
```

**检查上游 DNS 配置**

```bash
# 查看 CoreDNS Pod 的 DNS 配置
kubectl exec -n kube-system <coredns-pod-name> -- cat /etc/resolv.conf

# 测试上游 DNS 连通性
kubectl exec -n kube-system <coredns-pod-name> -- nslookup google.com <upstream-dns-ip>
```

### 3.4 网络连通性测试

**测试 API Server 连接**

```bash
# 获取 API Server 地址
kubectl cluster-info

# 从 CoreDNS Pod 测试连接
kubectl exec -n kube-system <coredns-pod-name> -- curl -k https://kubernetes.default.svc.cluster.local/api/v1/namespaces

# 检查 Service Account Token
kubectl exec -n kube-system <coredns-pod-name> -- ls -la /var/run/secrets/kubernetes.io/serviceaccount/
```

**测试 DNS 解析功能**

```bash
# 创建测试 Pod
kubectl run dns-test --image=busybox:1.28 --rm -it --restart=Never -- nslookup kubernetes.default

# 测试 Service 解析
kubectl exec -n kube-system <coredns-pod-name> -- nslookup kubernetes.default.svc.cluster.local 127.0.0.1

# 测试外部域名解析
kubectl exec -n kube-system <coredns-pod-name> -- nslookup google.com 127.0.0.1
```

### 3.5 性能分析

**检查 DNS 查询负载**

```bash
# 查看 CoreDNS 监控指标
kubectl port-forward -n kube-system svc/kube-dns 9153:9153 &

# 访问 Prometheus 指标
curl http://localhost:9153/metrics | grep coredns_dns_request_count_total
curl http://localhost:9153/metrics | grep coredns_dns_request_duration_seconds
```

**分析内存使用模式**

```bash
# 持续监控内存使用
watch -n 1 'kubectl top pods -n kube-system -l k8s-app=kube-dns'

# 查看内存详细使用
kubectl exec -n kube-system <coredns-pod-name> -- cat /sys/fs/cgroup/memory/memory.usage_in_bytes
kubectl exec -n kube-system <coredns-pod-name> -- cat /sys/fs/cgroup/memory/memory.limit_in_bytes
```

## 四、常见错误日志分析

### 4.1 错误日志分类

| 错误类型 | 典型日志 | 根本原因 | 影响范围 |
|---------|---------|---------|---------|
| **OOMKilled** | `OOMKilled` 或 `Exit Code 137` | 内存超限 | 服务中断 |
| **配置错误** | `plugin/forward: no upstream` | forward 配置错误 | 外部域名解析失败 |
| **DNS 环路** | `plugin/loop: loop detected` | DNS 解析环路 | 完全不可用 |
| **API 连接失败** | `Failed to list *v1.Service` | 无法连接 API Server | 集群内解析失败 |
| **健康检查失败** | `Readiness probe failed` | DNS 服务无响应 | Pod 被重启 |

### 4.2 详细错误分析

**OOMKilled 错误**

```
State:          Terminated
  Reason:       OOMKilled
  Exit Code:    137
```

**原因分析**：
- DNS 查询量激增，缓存占用大量内存
- 存在 DNS 查询攻击或异常流量
- 内存限制设置过低

**DNS 解析环路错误**

```
[ERROR] plugin/loop: Loop detected in forward chain
```

**原因分析**：
- CoreDNS 的 forward 配置指向了自身
- 上游 DNS 配置指向 CoreDNS Service IP
- /etc/resolv.conf 配置错误

**API Server 连接失败**

```
E1234 10:00:00.000000 1 reflector.go:156] pkg/mod/k8s.io/client-go@v0.0.0/tools/cache/reflector.go:108: Failed to list *v1.Service: Get https://10.96.0.1:443/api/v1/namespaces/default/services?limit=500&resourceVersion=0: dial tcp 10.96.0.1:443: connect: connection refused
```

**原因分析**：
- API Server 不可用或过载
- 网络策略阻止连接
- Service CIDR 配置错误

## 五、解决方案

### 5.1 资源优化方案

**调整资源限制**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
spec:
  template:
    spec:
      containers:
      - name: coredns
        resources:
          limits:
            memory: 512Mi    # 增加内存限制
            cpu: 500m        # 增加 CPU 限制
          requests:
            memory: 256Mi
            cpu: 200m
```

**优化缓存配置**

修改 Corefile，调整缓存参数：

```
.:53 {
    cache 30  # 缓存 30 秒
    cache {
        success 9984 30  # 成功响应缓存
        denial 9984 5    # 否定响应缓存
        prefetch 10      # 预取热门域名
    }
}
```

### 5.2 配置修复方案

**修复 DNS 环路**

检查并修复 /etc/resolv.conf 配置：

```bash
# 查看节点 resolv.conf
cat /etc/resolv.conf

# 确保 CoreDNS Pod 的 resolv.conf 不指向自身
kubectl exec -n kube-system <coredns-pod-name> -- cat /etc/resolv.conf
```

正确的 forward 配置：

```
forward . 8.8.8.8 8.8.4.4 {
    policy sequential
    health_check 0.5s
}
```

**修复上游 DNS 配置**

```bash
# 使用可靠的公共 DNS
kubectl edit configmap coredns -n kube-system
```

修改 Corefile：

```
forward . 8.8.8.8 8.8.4.4 114.114.114.114 {
    max_concurrent 1000
}
```

### 5.3 高可用配置

**增加副本数**

```bash
# 扩展 CoreDNS 副本数
kubectl scale deployment coredns -n kube-system --replicas=3

# 设置自动扩缩容
kubectl autoscale deployment coredns -n kube-system --min=2 --max=10 --cpu-percent=80
```

**配置 Pod 反亲和性**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: k8s-app
                  operator: In
                  values:
                  - kube-dns
              topologyKey: kubernetes.io/hostname
```

### 5.4 网络问题修复

**检查网络策略**

```bash
# 查看是否有网络策略限制
kubectl get networkpolicy -n kube-system

# 检查 CNI 插件状态
kubectl get pods -n kube-system -l k8s-app=<cni-plugin-name>
```

**修复 Service CIDR**

```bash
# 检查 kube-apiserver 配置
ps aux | grep kube-apiserver | grep service-cluster-ip-range

# 检查 kube-proxy 配置
kubectl get configmap kube-proxy -n kube-system -o yaml | grep clusterCIDR
```

### 5.5 监控和告警

**配置 Prometheus 监控**

```yaml
# CoreDNS ServiceMonitor 示例
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: coredns
  namespace: monitoring
spec:
  selector:
    matchLabels:
      k8s-app: kube-dns
  endpoints:
  - port: metrics
    interval: 30s
```

**关键监控指标**

| 指标名称 | 说明 | 告警阈值 |
|---------|------|---------|
| `coredns_dns_request_count_total` | DNS 请求总数 | 无 |
| `coredns_dns_request_duration_seconds` | 请求延迟 | P99 > 100ms |
| `coredns_dns_response_rcode_count_total` | 响应码统计 | SERVFAIL > 5% |
| `coredns_cache_hits_total` | 缓存命中数 | 命中率 < 50% |
| `coredns_forward_healthcheck_broken_count` | 上游 DNS 故障 | > 0 |

## 六、排查流程图

```
CoreDNS 频繁重启
        ↓
    检查 Pod 状态
        ↓
    ┌───┴───┐
    ↓       ↓
OOMKilled  其他原因
    ↓       ↓
增加资源  查看日志
限制      ↓
    ┌─────┴─────┐
    ↓           ↓
配置错误    网络问题
    ↓           ↓
检查      测试网络
Corefile   连通性
    ↓           ↓
修复配置   检查网络
    ↓       策略
验证修复    ↓
    ↓       修复网络
恢复正常    配置
            ↓
        验证修复
            ↓
        恢复正常
```

## 七、常见问题与最佳实践

### 7.1 五个常见问题

**问题 1：CoreDNS 内存占用持续增长，如何处理？**

**答**：这是 DNS 缓存累积的正常现象。解决方案：
- 启用 cache 插件的 `prefetch` 功能，自动清理冷门缓存
- 调整缓存 TTL，减少缓存时间
- 设置合理的内存限制，让系统自动管理
- 定期重启 CoreDNS Pod 释放内存（不推荐，治标不治本）

**问题 2：如何判断 CoreDNS 是否成为性能瓶颈？**

**答**：通过以下指标判断：
- DNS 查询延迟 P99 > 100ms
- CoreDNS CPU 使用率持续 > 80%
- DNS 查询超时率 > 1%
- 缓存命中率 < 50%
- 出现大量 DNS 查询排队

解决方案：增加副本数、优化缓存配置、启用负缓存、调整资源限制。

**问题 3：CoreDNS 无法解析外部域名，但内部域名正常，如何排查？**

**答**：这是 forward 插件配置问题：
- 检查 Corefile 中 forward 配置的上游 DNS 是否可达
- 测试从 CoreDNS Pod 访问外部 DNS：`nslookup google.com 8.8.8.8`
- 检查节点和 Pod 的网络出口是否受限
- 验证防火墙规则是否阻止 DNS 流量（UDP/TCP 53 端口）
- 检查是否有网络策略限制 CoreDNS 的出站流量

**问题 4：CoreDNS 副本数应该设置为多少？**

**答**：根据集群规模和负载确定：
- 小型集群（< 50 节点）：2 个副本
- 中型集群（50-200 节点）：3-5 个副本
- 大型集群（> 200 节点）：5-10 个副本
- 高负载场景：启用 HPA 自动扩缩容

建议配置 HPA，根据 CPU 使用率自动调整副本数。

**问题 5：如何防止 CoreDNS 单点故障？**

**答**：多层面保障：
- **副本冗余**：至少 2 个副本，分布在不同节点
- **Pod 反亲和性**：确保副本分散在不同节点
- **资源保障**：设置 Guaranteed QoS，避免资源竞争
- **健康检查**：配置合理的 liveness 和 readiness 探针
- **监控告警**：实时监控 CoreDNS 状态，及时发现问题
- **备份方案**：配置 NodeLocal DNSCache 作为本地缓存层

### 7.2 最佳实践

**资源配置最佳实践**

| 集群规模 | CPU Request | CPU Limit | Memory Request | Memory Limit |
|---------|------------|-----------|----------------|--------------|
| 小型 (< 50 节点) | 100m | 300m | 70Mi | 170Mi |
| 中型 (50-200 节点) | 200m | 500m | 150Mi | 300Mi |
| 大型 (> 200 节点) | 500m | 1000m | 256Mi | 512Mi |
| 超大型 (> 500 节点) | 1000m | 2000m | 512Mi | 1Gi |

**Corefile 配置最佳实践**

```
.:53 {
    errors                    # 错误日志
    health {                  # 健康检查
        lameduck 5s          # 优雅关闭时间
    }
    ready                     # 就绪检查
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        pods insecure         # Pod 解析模式
        fallthrough in-addr.arpa ip6.arpa
        ttl 30               # TTL 设置
    }
    prometheus :9153         # 监控指标
    forward . /etc/resolv.conf {
        max_concurrent 1000  # 最大并发数
        policy sequential    # 转发策略
    }
    cache 30 {               # 缓存配置
        success 9984 30
        denial 9984 5
        prefetch 10 10m 10%  # 预取热门域名
    }
    loop                     # 环路检测
    reload                   # 配置热加载
    loadbalance              # 负载均衡
}
```

**监控告警最佳实践**

```yaml
# PrometheusRule 示例
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: coredns-alerts
  namespace: monitoring
spec:
  groups:
  - name: coredns
    rules:
    - alert: CoreDNSDown
      expr: up{job="kube-dns"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "CoreDNS instance is down"
        
    - alert: CoreDNSHighLatency
      expr: histogram_quantile(0.99, rate(coredns_dns_request_duration_seconds_bucket[5m])) > 0.1
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "CoreDNS high latency detected"
        
    - alert: CoreDNSMemoryHigh
      expr: container_memory_usage_bytes{pod=~"coredns-.*"} / container_spec_memory_limit_bytes{pod=~"coredns-.*"} > 0.9
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "CoreDNS memory usage is high"
```

**运维最佳实践**

1. **定期检查日志**：每周检查 CoreDNS 错误日志，及时发现潜在问题
2. **监控资源使用**：设置资源使用告警，提前扩容
3. **备份配置**：定期备份 CoreDNS ConfigMap 和 Deployment 配置
4. **版本管理**：保持 CoreDNS 版本与 Kubernetes 版本兼容
5. **压力测试**：定期进行 DNS 压力测试，验证性能基线
6. **文档记录**：记录所有配置变更和故障处理过程

## 八、总结

CoreDNS 作为 Kubernetes 集群的核心组件，其稳定性直接影响整个集群的服务发现能力。通过本文的系统化排查方法，可以快速定位和解决 CoreDNS 频繁重启问题：

1. **资源层面**：合理配置 CPU 和内存限制，避免 OOMKilled
2. **配置层面**：确保 Corefile 语法正确，避免 DNS 环路和上游 DNS 配置错误
3. **网络层面**：保障 CoreDNS 与 API Server、上游 DNS 的网络连通性
4. **监控层面**：建立完善的监控告警体系，及时发现和处理问题
5. **高可用层面**：通过多副本、反亲和性、自动扩缩容保障服务可用性

---

## 面试回答

**面试官问：当遇到 CoreDNS 经常重启和报错，这种故障如何排查？**

**回答**：

CoreDNS 频繁重启是 Kubernetes 生产环境中常见的故障，我会从以下几个维度进行系统化排查：

首先，通过 `kubectl get pods -n kube-system -l k8s-app=kube-dns` 查看 Pod 状态，使用 `kubectl describe pod` 查看重启原因。如果看到 OOMKilled，说明是内存不足，需要增加内存限制或优化缓存配置。

其次，通过 `kubectl logs --previous` 查看重启前的日志，重点关注几类错误：一是配置错误，如 Corefile 语法错误、forward 配置的上游 DNS 不可达；二是 DNS 环路，日志会显示 "loop detected"；三是 API Server 连接失败，导致无法获取 Service 信息。

然后，我会测试网络连通性，包括 CoreDNS 与 API Server 的连接、与上游 DNS 的连接，以及 DNS 解析功能本身。通过 `nslookup` 命令分别测试集群内域名和外部域名解析。

最后，根据排查结果采取相应措施：如果是资源问题，调整 limits 和 requests；如果是配置问题，修复 Corefile；如果是网络问题，检查网络策略和 CNI 插件；同时建议配置 HPA 实现自动扩缩容，设置 Pod 反亲和性保证高可用，并建立完善的监控告警体系。

关键是要建立预防机制，包括合理的资源配置、定期的日志检查、性能监控和压力测试，从根源上减少故障发生。

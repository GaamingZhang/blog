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

# Kubernetes在生产环境中部署有哪些参数可以调整优化

## 核心优化维度

生产环境优化主要围绕：**资源管理、性能调优、可靠性保障、安全加固、成本控制**五个维度。

---

## 一、资源配置优化

**1. Pod资源请求和限制**

```yaml
resources:
  requests:      # 调度依据，保证资源
    memory: "256Mi"
    cpu: "500m"
  limits:        # 硬性上限
    memory: "512Mi"
    cpu: "1000m"
```

**关键参数**：
- **requests**: 决定调度和QoS等级
- **limits**: 防止资源耗尽，触发OOM或CPU限流
- **建议**: requests = limits（Guaranteed QoS）适用于关键业务

**2. QoS等级选择**

| QoS等级    | 条件               | 驱逐优先级 | 适用场景   |
| ---------- | ------------------ | ---------- | ---------- |
| Guaranteed | requests = limits  | 最低       | 核心服务   |
| Burstable  | 有requests或limits | 中等       | 一般业务   |
| BestEffort | 无资源配置         | 最高       | 批处理任务 |

**3. 节点资源预留**

```yaml
# kubelet配置
kubeletArgs:
  system-reserved:
    cpu: "1000m"
    memory: "2Gi"
  kube-reserved:
    cpu: "1000m"
    memory: "2Gi"
  eviction-hard:
    memory.available: "500Mi"
    nodefs.available: "10%"
```

---

## 二、调度策略优化

**1. 节点亲和性（Node Affinity）**

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:  # 硬性要求
      nodeSelectorTerms:
      - matchExpressions:
        - key: disktype
          operator: In
          values: ["ssd"]
    preferredDuringSchedulingIgnoredDuringExecution:  # 软性偏好
    - weight: 100
      preference:
        matchExpressions:
        - key: zone
          operator: In
          values: ["zone-a"]
```

**2. Pod亲和性和反亲和性**

```yaml
# 反亲和性：分散部署提高可用性
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
    - labelSelector:
        matchExpressions:
        - key: app
          operator: In
          values: ["web"]
      topologyKey: kubernetes.io/hostname  # 不同节点
```

**3. 污点和容忍**

```yaml
# 节点专用化
tolerations:
- key: "dedicated"
  operator: "Equal"
  value: "database"
  effect: "NoSchedule"
```

---

## 三、控制器配置优化

**1. Deployment更新策略**

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1           # 最多超出期望副本数
    maxUnavailable: 0     # 滚动更新时保证可用性
minReadySeconds: 10       # Pod就绪后等待时间
revisionHistoryLimit: 10  # 保留历史版本数
progressDeadlineSeconds: 600  # 更新超时时间
```

**2. HPA自动伸缩**

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
spec:
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70  # CPU目标使用率
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:  # 扩缩容行为
    scaleDown:
      stabilizationWindowSeconds: 300  # 缩容稳定窗口
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 30
```

**3. PDB保障可用性**

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: app-pdb
spec:
  minAvailable: 2        # 或 maxUnavailable: 1
  selector:
    matchLabels:
      app: myapp
```

---

## 四、健康检查优化

**三种探针配置**

```yaml
spec:
  containers:
  - name: app
    # 启动探针：慢启动应用
    startupProbe:
      httpGet:
        path: /startup
        port: 8080
      failureThreshold: 30      # 允许失败次数多
      periodSeconds: 10
    
    # 存活探针：检测死锁
    livenessProbe:
      httpGet:
        path: /healthz
        port: 8080
      initialDelaySeconds: 30   # 初始延迟
      periodSeconds: 10         # 检查间隔
      timeoutSeconds: 5         # 超时时间
      failureThreshold: 3       # 失败阈值
      successThreshold: 1
    
    # 就绪探针：流量控制
    readinessProbe:
      httpGet:
        path: /ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
      failureThreshold: 2
```

**探针类型选择**：
- **HTTP GET**: Web服务
- **TCP Socket**: 数据库等TCP服务
- **Exec**: 自定义脚本检查

---

## 五、网络性能优化

**1. Service配置**

```yaml
apiVersion: v1
kind: Service
spec:
  type: ClusterIP
  sessionAffinity: ClientIP  # 会话保持
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800
  ipFamilyPolicy: PreferDualStack  # 双栈支持
  internalTrafficPolicy: Local     # 本地流量策略
```

**2. Ingress优化**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-connect-timeout: "60"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "120"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/rate-limit: "100"  # 限流
```

**3. NetworkPolicy隔离**

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-specific
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
```

---

## 六、存储优化

**1. PVC配置**

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
spec:
  accessModes:
  - ReadWriteOnce
  storageClassName: fast-ssd  # 选择合适的存储类
  resources:
    requests:
      storage: 10Gi
  volumeMode: Filesystem
```

**2. EmptyDir优化**

```yaml
volumes:
- name: cache
  emptyDir:
    medium: Memory      # 使用内存，高性能临时存储
    sizeLimit: 1Gi
```

---

## 七、API Server优化

**关键参数**：

```bash
# kube-apiserver配置
--max-requests-inflight=400           # 并发请求数
--max-mutating-requests-inflight=200  # 变更请求数
--request-timeout=60s                 # 请求超时
--watch-cache-sizes=default=1000      # watch缓存大小
--enable-aggregator-routing=true      # 聚合路由
--audit-log-maxage=30                 # 审计日志保留天数
--audit-log-maxbackup=10
--audit-log-maxsize=100
```

---

## 八、etcd优化

**核心配置**：

```bash
# etcd参数
--quota-backend-bytes=8589934592      # 8GB存储配额
--heartbeat-interval=100              # 心跳间隔(ms)
--election-timeout=1000               # 选举超时(ms)
--snapshot-count=10000                # 快照间隔
--auto-compaction-retention=1         # 自动压缩
--max-request-bytes=1572864           # 最大请求大小
```

**监控指标**：
- etcd_disk_wal_fsync_duration_seconds (应 < 10ms)
- etcd_disk_backend_commit_duration_seconds (应 < 100ms)
- etcd_server_has_leader
- etcd_mvcc_db_total_size_in_bytes

---

## 九、kubelet优化

**关键配置**：

```yaml
# kubelet config
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
maxPods: 110                    # 每节点最大Pod数
podsPerCore: 10                 # 每核最大Pod数
imageGCHighThresholdPercent: 85 # 镜像GC高水位
imageGCLowThresholdPercent: 80  # 镜像GC低水位
evictionHard:                   # 硬驱逐阈值
  memory.available: "500Mi"
  nodefs.available: "10%"
  imagefs.available: "15%"
evictionSoft:                   # 软驱逐阈值
  memory.available: "1Gi"
  nodefs.available: "15%"
evictionSoftGracePeriod:
  memory.available: "1m30s"
  nodefs.available: "2m"
cpuManagerPolicy: static        # CPU管理策略
topologyManagerPolicy: best-effort  # NUMA拓扑
```

---

## 十、容器运行时优化

**containerd配置**：

```toml
[plugins."io.containerd.grpc.v1.cri".containerd]
  snapshotter = "overlayfs"
  default_runtime_name = "runc"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
  SystemdCgroup = true

[plugins."io.containerd.grpc.v1.cri".cni]
  bin_dir = "/opt/cni/bin"
  conf_dir = "/etc/cni/net.d"

[plugins."io.containerd.grpc.v1.cri".registry.mirrors]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
    endpoint = ["https://mirror.example.com"]
```

---

## 十一、安全配置

**1. RBAC最小权限**

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pod-reader
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list"]
```

**2. Pod Security Standards**

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

**3. SecurityContext**

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 2000
  capabilities:
    drop:
    - ALL
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  seccompProfile:
    type: RuntimeDefault
```

---

## 十二、监控和日志

**1. 资源监控**

```yaml
# 启用metrics-server
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# 配置Prometheus监控
- job_name: 'kubernetes-pods'
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    action: keep
    regex: true
```

**2. 日志收集**

```yaml
# Fluentd DaemonSet配置
resources:
  limits:
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 256Mi
```

---

## 十三、高可用配置

**控制平面HA**：

```yaml
# kube-apiserver
- 3个或以上实例
- 负载均衡器分发
- --apiserver-count=3

# etcd集群
- 至少3个节点（奇数）
- 分布在不同故障域
- 定期备份

# 工作节点
- 节点跨可用区部署
- 关键应用多副本
- 设置PodDisruptionBudget
```

---

## 十四、成本优化

**1. 资源超售配置**

```yaml
# 非关键业务使用Burstable
resources:
  requests:
    cpu: 100m      # 低requests
  limits:
    cpu: 1000m     # 高limits允许突发
```

**2. 节点池分类**

- **On-Demand**: 关键业务
- **Spot/抢占式**: 批处理任务
- **不同规格**: 按workload选择

**3. 集群自动伸缩**

```yaml
# cluster-autoscaler配置
--scale-down-delay-after-add=10m
--scale-down-unneeded-time=10m
--skip-nodes-with-local-storage=false
--skip-nodes-with-system-pods=false
```

---

### 优化检查清单

### 资源配置
- [ ] 所有Pod设置了resources requests和limits
- [ ] 核心服务使用Guaranteed QoS
- [ ] 节点设置了system-reserved和kube-reserved

### 可靠性
- [ ] 关键应用配置了PDB
- [ ] 多副本部署且分散在不同节点
- [ ] 配置了合理的健康探针
- [ ] 设置了适当的滚动更新策略

### 性能
- [ ] 使用HPA自动伸缩
- [ ] 高性能应用使用SSD存储
- [ ] 网络密集型应用考虑Host网络
- [ ] 配置了合适的Service sessionAffinity

### 安全
- [ ] 启用RBAC并遵循最小权限原则
- [ ] Pod使用非root用户运行
- [ ] 配置了NetworkPolicy限制流量
- [ ] 敏感信息使用Secret存储

### 监控
- [ ] 部署了metrics-server
- [ ] 配置了Prometheus监控
- [ ] 设置了关键指标告警
- [ ] 日志统一收集到中心化系统

---

### 常见问题

### Q1: requests和limits的区别？设置不当会有什么影响？

**答案**：
- **requests**: 调度和资源保证的依据，不会被超用
- **limits**: 硬性上限，超过会被限流(CPU)或OOM(内存)

**影响**：
- 只设requests无limits：可能耗尽节点资源
- requests过低：调度过度，运行时资源不足
- limits过低：频繁限流或OOM
- requests > limits：配置无效
- 不设置：BestEffort QoS，最先被驱逐

**最佳实践**：关键服务 requests = limits（Guaranteed）

### Q2: 如何选择合适的HPA指标和阈值？

**答案**：

**常用指标**：
- **CPU**: 计算密集型（目标70-80%）
- **Memory**: 内存密集型（目标80-90%，注意内存不可压缩）
- **自定义指标**: QPS、队列长度、业务指标

**设置原则**：
- 保留20-30%余量应对突发
- 考虑应用启动时间
- 设置stabilizationWindow避免抖动
- 扩容快、缩容慢

### Q3: 生产环境中节点资源如何预留？

**答案**：

```yaml
# 推荐配置
system-reserved:     # 操作系统
  cpu: "500m-1000m"
  memory: "1-2Gi"

kube-reserved:       # K8s组件
  cpu: "500m-1000m"
  memory: "1-2Gi"

eviction-hard:       # 硬驱逐
  memory.available: "500Mi"
  nodefs.available: "10%"
```

**计算公式**：
可分配资源 = 节点容量 - system-reserved - kube-reserved - eviction-threshold

### Q4: 如何保证关键服务的高可用性？

**答案**：

**多层保障**：
1. **多副本**: replicas >= 3
2. **反亲和性**: 分散到不同节点/可用区
3. **PDB**: 保证最小可用副本数
4. **健康检查**: readinessProbe + livenessProbe
5. **滚动更新**: maxUnavailable: 0
6. **资源保证**: Guaranteed QoS
7. **自动恢复**: Deployment自动重启
8. **监控告警**: 及时发现问题

### Q5: 大规模集群性能瓶颈在哪里？如何优化？

**答案**：

**常见瓶颈**：
1. **etcd**: 存储和性能上限
2. **API Server**: 请求并发限制
3. **节点数量**: 单集群上限（推荐5000节点内）
4. **Pod数量**: 单节点上限（推荐110个内）

**优化方案**：
```bash
# API Server
--max-requests-inflight=400
--watch-cache-sizes=配置合理大小

# etcd
--quota-backend-bytes=8GB
定期压缩和碎片整理

# 架构优化
- 集群联邦/多集群
- 减少list/watch操作
- 使用Informer机制
- 启用事件TTL
```

### Q6: 如何优化容器镜像拉取速度？

**答案**：

**优化策略**：
1. **镜像仓库优化**
   - 使用本地/区域镜像仓库
   - 配置镜像缓存代理
   
2. **镜像策略**
   ```yaml
   imagePullPolicy: IfNotPresent  # 本地有则不拉取
   ```

3. **预拉取镜像**
   - DaemonSet预热
   - Init容器预拉取

4. **镜像优化**
   - 使用小基础镜像（alpine）
   - 多阶段构建
   - 合并RUN层

5. **并发拉取**
   ```yaml
   # kubelet配置
   maxParallelImagePulls: 5
   serializeImagePulls: false
   ```

### Q7: Pod Disruption Budget (PDB)的作用是什么？如何正确配置？

**答案**：
- **作用**：限制自愿性中断（如节点维护、滚动更新）时的Pod中断数量，保障服务可用性。

**配置示例**：
```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: critical-app-pdb
spec:
  minAvailable: 2  # 或 maxUnavailable: 1
  selector:
    matchLabels:
      app: critical-app
```

**关键点**：
- `minAvailable`: 最少可用Pod数，绝对值或百分比
- `maxUnavailable`: 最多不可用Pod数，绝对值或百分比
- 不支持同时设置两个参数
- 需要配合Deployment/StatefulSet的副本数使用
- 仅适用于自愿性中断，不适用于故障性中断

### Q8: SecurityContext有哪些关键配置？如何提升容器安全性？

**答案**：
**关键配置**：
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 2000
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
  seccompProfile:
    type: RuntimeDefault
```

**安全性提升**：
- **非Root运行**：防止容器逃逸后的权限提升
- **只读根文件系统**：防止恶意写入系统文件
- **禁止特权升级**：限制setuid/setgid操作
- **最小化能力**：只保留必要的Linux Capabilities
- **Seccomp配置**：限制系统调用，减少攻击面
- **用户组隔离**：使用独立的用户和组运行容器

### Q9: 如何保证etcd数据的安全性和可用性？

**答案**：

**数据可用性**：
1. **集群部署**：3/5/7节点奇数集群，跨故障域
2. **合理配置**：
   ```bash
   --heartbeat-interval=100  # 心跳间隔
   --election-timeout=1000   # 选举超时
   ```

**数据安全性**：
1. **快照备份**：
   ```bash
   etcdctl snapshot save backup.db
   
   # 检查快照状态
   etcdctl snapshot status backup.db
   ```

2. **集群恢复**：
   ```bash
   etcdctl snapshot restore backup.db \
     --name node1 \
     --initial-cluster "node1=https://node1:2380,node2=https://node2:2380,node3=https://node3:2380" \
     --initial-advertise-peer-urls "https://node1:2380"
   ```

3. **加密传输**：启用TLS加密etcd通信
4. **访问控制**：使用RBAC限制etcd访问
5. **定期压缩**：
   ```bash
   --auto-compaction-retention=1  # 保留1小时历史
   ```

---

### 核心优化建议总结

**1. 资源配置三原则**
- 所有Pod必须设置resources
- 关键服务使用Guaranteed QoS
- 监控实际使用调整配置

**2. 高可用四要素**
- 多副本 + 反亲和性
- PDB + 健康检查
- 跨可用区部署
- 自动伸缩

**3. 性能优化五重点**
- 合理的资源限制
- 高效的调度策略
- 优化的网络配置
- 快速的镜像拉取
- 完善的监控体系

**4. 安全加固六层防护**
- RBAC权限控制
- NetworkPolicy网络隔离
- SecurityContext容器安全
- Secret敏感信息管理
- 审计日志记录
- 定期安全扫描

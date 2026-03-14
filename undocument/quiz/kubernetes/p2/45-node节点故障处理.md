---
date: 2026-03-12
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Node
  - 故障处理
---

# Kubernetes节点故障处理完全指南

## 引言

在Kubernetes集群中,Node节点是工作负载运行的基础设施单元。当节点发生故障时,可能导致运行在该节点上的所有Pod不可用,严重影响业务连续性。理解节点故障的处理机制,对于保障集群稳定性和业务高可用至关重要。

节点故障可能由硬件故障、网络问题、资源耗尽、kubelet异常等多种原因引起。Kubernetes通过节点健康检测、Pod驱逐、自动重调度等机制,确保在节点故障时能够快速恢复业务。本文将深入解析节点故障的处理流程,帮助您掌握从故障发现到恢复的完整链路。

## 一、节点故障的常见原因

### 1.1 故障类型分类

节点故障按照严重程度和影响范围,可以分为以下几类:

| 故障类型 | 具体表现 | 影响范围 | 检测难度 |
|---------|---------|---------|---------|
| **硬件故障** | CPU、内存、磁盘、网卡等硬件损坏 | 节点完全不可用 | 中等 |
| **网络故障** | 网络分区、延迟过高、丢包严重 | 节点通信中断 | 较高 |
| **资源耗尽** | CPU/内存/磁盘IO/网络带宽耗尽 | Pod运行缓慢或OOM | 较低 |
| **软件故障** | kubelet、容器运行时、操作系统异常 | 部分或全部Pod异常 | 中等 |
| **配置错误** | kubelet配置错误、证书过期 | 节点无法加入集群 | 较低 |

### 1.2 典型故障场景

**场景一: kubelet进程异常**

kubelet是节点上最重要的组件,负责与API Server通信、管理Pod生命周期。当kubelet进程崩溃或停止响应时,节点将进入NotReady状态。

**场景二: 资源压力导致的节点故障**

当节点资源(CPU、内存、磁盘)使用率持续过高时,可能触发系统级别的OOM Killer,导致关键进程被杀死,甚至节点假死。

**场景三: 网络分区导致的脑裂**

在网络分区场景下,节点可能无法与Master通信,但节点上的Pod仍在运行。这种情况下需要谨慎处理,避免数据不一致。

**场景四: 容器运行时故障**

容器运行时(Docker、containerd)异常会导致无法创建新容器,已有容器可能无法正常管理。

## 二、节点故障检测机制

### 2.1 Node Controller的工作原理

Node Controller是Kubernetes Master上的核心控制器,负责监控节点状态并执行相应操作。其工作流程如下:

```
┌─────────────────────────────────────────────────────────┐
│                  Node Controller                        │
├─────────────────────────────────────────────────────────┤
│  1. 定期同步节点状态(每5秒)                              │
│  2. 监控节点心跳(kubelet定期上报)                        │
│  3. 判断节点健康状态                                     │
│  4. 触发Pod驱逐逻辑                                      │
│  5. 更新节点Conditions                                  │
└─────────────────────────────────────────────────────────┘
```

### 2.2 节点状态Conditions

节点状态通过Conditions字段详细描述,每个Condition包含Type、Status、Reason、Message等信息:

| Condition Type | 说明 | 正常值 | 异常值 |
|---------------|------|--------|--------|
| Ready | 节点是否健康 | True | False/Unknown |
| MemoryPressure | 内存压力 | False | True |
| DiskPressure | 磁盘压力 | False | True |
| PIDPressure | 进程ID压力 | False | True |
| NetworkUnavailable | 网络不可用 | False | True |

查看节点详细状态:

```bash
kubectl describe node <node-name>
```

输出示例:

```
Conditions:
  Type                 Status  LastHeartbeatTime                 Reason                       Message
  ----                 ------  -----------------                 ------                       -------
  MemoryPressure       False   Wed, 12 Mar 2026 10:30:00 +0800   KubeletHasSufficientMemory   kubelet has sufficient memory available
  DiskPressure         False   Wed, 12 Mar 2026 10:30:00 +0800   KubeletHasNoDiskPressure     kubelet has no disk pressure
  PIDPressure          False   Wed, 12 Mar 2026 10:30:00 +0800   KubeletHasSufficientPID      kubelet has sufficient PID available
  Ready                True    Wed, 12 Mar 2026 10:30:00 +0800   KubeletReady                 kubelet is posting ready status
```

### 2.3 心跳机制详解

Kubernetes节点心跳机制包含两种方式:

**方式一: NodeStatus更新**

kubelet定期向API Server更新节点状态,默认间隔为10秒(通过`--node-status-update-frequency`参数配置)。

**方式二: Lease对象**

从Kubernetes 1.14开始,引入Lease对象优化心跳性能。kubelet在kube-node-lease命名空间中创建Lease对象,默认每10秒更新一次。

Node Controller的检测参数:

| 参数 | 默认值 | 说明 |
|-----|--------|------|
| --node-monitor-period | 5秒 | Node Controller同步节点状态的间隔 |
| --node-monitor-grace-period | 40秒 | 节点无响应多久标记为Unknown |
| --pod-eviction-timeout | 5分钟 | 节点NotReady多久后开始驱逐Pod |

### 2.4 节点状态流转

```
┌──────────┐
│   Ready  │
└────┬─────┘
     │ kubelet停止响应
     │ (超过40秒)
     ▼
┌──────────┐
│ Unknown  │
└────┬─────┘
     │ 超过pod-eviction-timeout
     │ (默认5分钟)
     ▼
┌──────────┐
│ 触发Pod  │
│ 驱逐流程 │
└──────────┘
```

## 三、Pod驱逐流程

### 3.1 驱逐机制原理

当节点被标记为NotReady或Unknown状态,并且持续时间超过`pod-eviction-timeout`后,Node Controller会触发Pod驱逐流程。驱逐流程的核心逻辑:

**步骤1: 判断Pod是否需要驱逐**

- DaemonSet管理的Pod不会被驱逐(因为有守护进程特性)
- 本地存储的Pod需要特殊处理
- 已容忍节点NotReady的Pod不会被驱逐

**步骤2: 计算优雅终止期**

Pod的优雅终止期由以下公式计算:

```
grace-period = pod-eviction-timeout - (当前时间 - 节点变为NotReady的时间)
```

最小值为默认的优雅终止期(30秒)。

**步骤3: 删除Pod对象**

API Server删除Pod对象,Controller Manager中的相应控制器(如Deployment、ReplicaSet)会创建新的Pod,调度到其他健康节点。

### 3.2 驱逐优先级

Pod驱逐遵循以下优先级:

1. **优先驱逐无本地存储的Pod**
2. **其次驱逐有本地存储的Pod**
3. **最后驱逐DaemonSet Pod(通常不驱逐)**

### 3.3 配置Pod容忍度

通过配置Toleration,可以控制Pod在节点故障时的行为:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: critical-app
spec:
  tolerations:
  - key: "node.kubernetes.io/not-ready"
    operator: "Exists"
    effect: "NoExecute"
    tolerationSeconds: 600  # 容忍10分钟
  - key: "node.kubernetes.io/unreachable"
    operator: "Exists"
    effect: "NoExecute"
    tolerationSeconds: 600
  containers:
  - name: app
    image: nginx:latest
```

**tolerationSeconds参数说明**:

- 不设置tolerationSeconds: Pod永远容忍节点NotReady
- 设置为0: 节点NotReady时立即驱逐Pod
- 设置为N秒: 节点NotReady后N秒开始驱逐Pod

### 3.4 驱逐过程监控

查看Pod驱逐事件:

```bash
kubectl get events --field-selector reason=NodeNotReady
kubectl get events --field-selector reason=NodeControllerEviction
```

查看Pod的Terminating状态:

```bash
kubectl get pods -o wide | grep Terminating
```

## 四、节点维护流程

### 4.1 Cordon: 标记节点不可调度

**原理**:

`kubectl cordon`命令将节点标记为不可调度(SchedulingDisabled),新的Pod不会被调度到该节点,但节点上已有的Pod继续运行。

**操作步骤**:

```bash
# 标记节点不可调度
kubectl cordon <node-name>

# 查看节点状态
kubectl get nodes
# 输出: <node-name>   Ready,SchedulingDisabled   ...

# 恢复节点可调度
kubectl uncordon <node-name>
```

**适用场景**:

- 节点需要临时维护,但允许已有Pod继续服务
- 需要逐步迁移工作负载
- 节点资源即将耗尽,防止新Pod调度

**注意事项**:

- Cordon不会驱逐已有Pod
- DaemonSet控制器会忽略SchedulingDisabled标记,继续在该节点创建Pod
- 需要手动执行uncordon恢复节点

### 4.2 Drain: 安全驱逐节点上的Pod

**原理**:

`kubectl drain`命令会执行以下操作:
1. 标记节点为不可调度(等同于cordon)
2. 驱逐节点上的所有Pod( DaemonSet Pod除外)
3. 等待Pod优雅终止

**操作步骤**:

```bash
# 基本驱逐命令
kubectl drain <node-name>

# 忽略DaemonSet管理的Pod
kubectl drain <node-name> --ignore-daemonsets

# 忽略本地存储的Pod
kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data

# 强制驱逐(忽略错误)
kubectl drain <node-name> --force --ignore-daemonsets --delete-emptydir-data

# 设置优雅终止期
kubectl drain <node-name> --grace-period=30 --ignore-daemonsets

# 设置超时时间
kubectl drain <node-name> --timeout=300s --ignore-daemonsets
```

**参数详解**:

| 参数 | 说明 | 使用场景 |
|-----|------|---------|
| --ignore-daemonsets | 忽略DaemonSet Pod | 节点上有DaemonSet时必须使用 |
| --delete-emptydir-data | 删除emptyDir数据 | Pod使用emptyDir卷时使用 |
| --force | 强制删除 | 删除未被控制器管理的Pod |
| --grace-period | 优雅终止期 | 控制Pod终止等待时间 |
| --timeout | 命令超时时间 | 防止drain命令卡住 |

**驱逐流程图**:

```
┌─────────────────────────────────────────────────────────┐
│                  kubectl drain流程                      │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
         ┌───────────────────────────┐
         │  标记节点SchedulingDisabled │
         └───────────┬───────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │  获取节点上所有Pod列表      │
         └───────────┬───────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │  过滤DaemonSet Pod         │
         └───────────┬───────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │  检查Pod是否有本地存储      │
         └───────────┬───────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │  发送SIGTERM信号           │
         └───────────┬───────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │  等待优雅终止期            │
         └───────────┬───────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │  发送SIGKILL信号           │
         └───────────┬───────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │  删除Pod对象               │
         └───────────────────────────┘
```

**注意事项**:

- 执行drain前确保集群有足够资源容纳迁移的Pod
- 对于StatefulSet应用,确保存储卷可以正常挂载到新节点
- 对于有本地存储的应用,需要先备份数据
- 建议在业务低峰期执行drain操作

### 4.3 Cordon vs Drain对比

| 对比项 | Cordon | Drain |
|-------|--------|-------|
| **影响范围** | 仅影响新Pod调度 | 影响新Pod调度和已有Pod运行 |
| **已有Pod** | 继续运行 | 被驱逐并重新调度 |
| **节点状态** | Ready,SchedulingDisabled | Ready,SchedulingDisabled |
| **适用场景** | 临时维护、资源保护 | 节点下线、重大维护 |
| **恢复操作** | uncordon | uncordon |
| **风险等级** | 低 | 中等 |

## 五、节点恢复步骤

### 5.1 故障诊断流程

**步骤1: 检查节点状态**

```bash
# 查看所有节点状态
kubectl get nodes

# 查看节点详细信息
kubectl describe node <node-name>

# 查看节点Conditions
kubectl get node <node-name> -o jsonpath='{.status.conditions}'
```

**步骤2: 检查kubelet状态**

```bash
# SSH登录到故障节点
ssh <node-ip>

# 检查kubelet服务状态
systemctl status kubelet

# 查看kubelet日志
journalctl -u kubelet -f

# 检查kubelet配置
cat /var/lib/kubelet/config.yaml
```

**步骤3: 检查容器运行时**

```bash
# 检查Docker状态(如果使用Docker)
systemctl status docker
docker ps -a

# 检查containerd状态(如果使用containerd)
systemctl status containerd
crictl ps -a
crictl pods
```

**步骤4: 检查系统资源**

```bash
# 检查CPU和内存
top
free -h

# 检查磁盘
df -h

# 检查磁盘IO
iostat -x 1

# 检查网络
ip addr
ping <master-ip>
```

### 5.2 常见故障恢复

**场景一: kubelet进程异常**

```bash
# 重启kubelet服务
systemctl restart kubelet

# 检查服务状态
systemctl status kubelet

# 如果kubelet无法启动,检查配置和证书
ls -la /etc/kubernetes/pki/
ls -la /var/lib/kubelet/pki/
```

**场景二: 容器运行时异常**

```bash
# 重启Docker
systemctl restart docker

# 重启containerd
systemctl restart containerd

# 清理异常容器
docker rm -f $(docker ps -aq -f status=exited)
crictl rm $(crictl ps -aq -s Exited)
```

**场景三: 磁盘空间不足**

```bash
# 查看磁盘使用情况
df -h

# 清理容器日志
find /var/lib/docker/containers -name "*.log" -size +100M -exec truncate -s 0 {} \;

# 清理无用镜像
docker image prune -a

# 清理无用容器
docker container prune

# 清理无用卷
docker volume prune
```

**场景四: 网络故障**

```bash
# 重启网络服务
systemctl restart network

# 检查防火墙规则
iptables -L -n

# 检查CNI插件
ls -la /etc/cni/net.d/
systemctl status kube-proxy
```

### 5.3 节点恢复验证

**验证步骤**:

```bash
# 1. 检查节点状态
kubectl get nodes

# 2. 检查节点详细信息
kubectl describe node <node-name>

# 3. 检查节点上的Pod
kubectl get pods --all-namespaces -o wide --field-selector spec.nodeName=<node-name>

# 4. 检查节点事件
kubectl get events --field-selector involvedObject.name=<node-name>

# 5. 验证Pod可以正常调度
kubectl run test-nginx --image=nginx --restart=Never --dry-run=client -o yaml | kubectl apply -f -
```

### 5.4 节点恢复流程图

```
┌─────────────────────────────────────────────────────────┐
│                  节点故障处理完整流程                    │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
         ┌───────────────────────────┐
         │  发现节点NotReady/Unknown  │
         └───────────┬───────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │  排查节点故障原因          │
         │  - kubelet状态            │
         │  - 容器运行时状态          │
         │  - 系统资源使用            │
         │  - 网络连通性              │
         └───────────┬───────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │  故障是否可以快速修复?      │
         └───────────┬───────────────┘
                     │
           ┌─────────┴─────────┐
           │                   │
           ▼                   ▼
    ┌────────────┐      ┌────────────┐
    │ 可以修复    │      │ 无法修复    │
    └──────┬─────┘      └──────┬─────┘
           │                   │
           ▼                   ▼
    ┌────────────┐      ┌────────────┐
    │ 修复故障    │      │ 执行drain  │
    │ 重启服务    │      │ 驱逐Pod    │
    └──────┬─────┘      └──────┬─────┘
           │                   │
           ▼                   ▼
    ┌────────────┐      ┌────────────┐
    │ 验证恢复    │      │ 节点下线    │
    └──────┬─────┘      │ 硬件维护    │
           │            └──────┬─────┘
           ▼                   │
    ┌────────────┐             │
    │ 恢复正常    │             ▼
    └────────────┘      ┌────────────┐
                        │ 重新加入集群 │
                        └──────┬─────┘
                               │
                               ▼
                        ┌────────────┐
                        │ uncordon   │
                        │ 恢复调度    │
                        └────────────┘
```

## 六、故障处理最佳实践

### 6.1 监控和告警

**关键监控指标**:

| 指标名称 | 说明 | 告警阈值 |
|---------|------|---------|
| kube_node_status_condition | 节点状态Condition | Ready=False超过5分钟 |
| kube_node_status_unschedulable | 节点不可调度 | 持续10分钟 |
| node_cpu_utilization | CPU使用率 | >80%持续5分钟 |
| node_memory_utilization | 内存使用率 | >85%持续5分钟 |
| node_filesystem_utilization | 磁盘使用率 | >85%持续5分钟 |
| kubelet_running_pods | 运行中的Pod数量 | 异常下降 |

**Prometheus告警规则示例**:

```yaml
groups:
- name: node-alerts
  rules:
  - alert: NodeNotReady
    expr: kube_node_status_condition{condition="Ready",status="true"} == 0
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Node {{ $labels.node }} is not ready"
      description: "Node {{ $labels.node }} has been unready for more than 5 minutes"

  - alert: NodeHighCPU
    expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Node {{ $labels.instance }} CPU usage high"
      description: "Node {{ $labels.instance }} CPU usage is above 80%"
```

### 6.2 高可用配置

**Pod Disruption Budget (PDB)**:

通过PDB确保应用在节点维护时的最小可用副本数:

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: nginx-pdb
spec:
  minAvailable: 2  # 或使用 maxUnavailable: 1
  selector:
    matchLabels:
      app: nginx
```

**多副本部署**:

- 确保关键应用至少有2-3个副本
- 使用反亲和性规则将Pod分散到不同节点
- 使用节点标签和节点选择器控制调度

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: critical-app
spec:
  replicas: 3
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - critical-app
            topologyKey: kubernetes.io/hostname
```

### 6.3 自动化运维

**自动节点修复工具**:

- **Node Problem Detector**: 检测节点问题并上报到API Server
- **Node Auto-Repair**: 自动修复特定类型的节点故障
- **Cluster Autoscaler**: 自动扩缩容节点

**Node Problem Detector配置示例**:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-problem-detector
spec:
  template:
    spec:
      containers:
      - name: node-problem-detector
        image: k8s.gcr.io/node-problem-detector:v0.8.7
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        volumeMounts:
        - name: log
          mountPath: /var/log
      volumes:
      - name: log
        hostPath:
          path: /var/log
```

### 6.4 故障演练

定期进行故障演练,验证故障处理流程的有效性:

**演练场景**:

1. 模拟kubelet故障: `systemctl stop kubelet`
2. 模拟网络分区: 使用iptables阻断网络
3. 模拟资源耗尽: 运行压力测试工具
4. 模拟节点下线: 执行drain操作

**演练检查清单**:

- [ ] 监控系统是否及时告警
- [ ] Pod是否自动迁移到其他节点
- [ ] 应用服务是否保持可用
- [ ] 数据是否完整无损
- [ ] 恢复流程是否顺利执行

## 七、常见问题FAQ

### Q1: 节点NotReady后,Pod多久会被驱逐?

**A**: 默认情况下,节点NotReady后5分钟开始驱逐Pod。这个时间由`--pod-eviction-timeout`参数控制。但Pod可以通过配置tolerationSeconds延长或缩短这个时间。

### Q2: 如何防止Pod在节点故障时被立即驱逐?

**A**: 有两种方式:

1. 在Pod中配置tolerations,设置tolerationSeconds为较大的值:

```yaml
tolerations:
- key: "node.kubernetes.io/not-ready"
  operator: "Exists"
  effect: "NoExecute"
  tolerationSeconds: 3600  # 容忍1小时
```

2. 为关键应用设置PodDisruptionBudget,确保最小可用副本数。

### Q3: Drain操作卡住怎么办?

**A**: Drain可能因为以下原因卡住:

1. Pod无法正常终止: 检查应用是否正确处理SIGTERM信号
2. PDB限制: 检查PDB配置,临时调整minAvailable
3. 本地存储: 使用`--delete-emptydir-data`参数
4. 超时设置: 使用`--timeout`参数限制总时间

强制终止命令:

```bash
kubectl drain <node-name> --force --ignore-daemonsets --delete-emptydir-data --grace-period=0
```

### Q4: 节点恢复后,如何让Pod重新调度回来?

**A**: 节点恢复后,执行以下步骤:

1. 检查节点状态: `kubectl get nodes`
2. 恢复节点可调度: `kubectl uncordon <node-name>`
3. 新创建的Pod会自动调度到该节点
4. 已有的Pod不会自动迁移回来,除非手动删除触发重新调度

### Q5: 如何区分节点故障和网络分区?

**A**: 判断方法:

1. **网络分区**: 节点状态Unknown,但SSH可能可以连接,节点上的Pod仍在运行
2. **节点故障**: 节点状态NotReady,SSH无法连接或kubelet停止,Pod可能停止运行

处理策略:

- 网络分区: 谨慎处理,避免数据不一致,可设置较长的tolerationSeconds
- 节点故障: 快速驱逐Pod,恢复服务

## 面试回答

在面试中回答"Node节点不能工作的处理"问题时,可以这样组织:

"当Kubernetes节点出现故障时,处理流程包括三个阶段:故障检测、Pod驱逐和节点恢复。首先,Node Controller通过心跳机制检测节点状态,当kubelet超过40秒未上报心跳,节点被标记为Unknown状态;超过5分钟后触发Pod驱逐流程。其次,Pod驱逐时会根据toleration配置决定驱逐时机,DaemonSet Pod通常不会被驱逐,其他Pod会被删除并由控制器重新调度到健康节点。对于节点维护,我们可以使用kubectl cordon标记节点不可调度,或使用kubectl drain安全驱逐节点上的所有Pod。最后,节点恢复需要排查kubelet、容器运行时、系统资源等故障原因,修复后通过uncordon恢复节点调度。最佳实践包括配置PodDisruptionBudget保障应用可用性、设置合理的tolerationSeconds控制驱逐时机、建立完善的监控告警体系,以及定期进行故障演练验证处理流程的有效性。"

---

**参考资料**:

- [Kubernetes官方文档 - Node](https://kubernetes.io/docs/concepts/architecture/nodes/)
- [Kubernetes官方文档 - Safely Drain a Node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)
- [Kubernetes官方文档 - Taints and Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)

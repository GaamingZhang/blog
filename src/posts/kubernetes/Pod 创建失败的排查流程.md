# 如何排查Kubernetes集群中Pod启动失败

## 概述

Pod启动失败是Kubernetes集群运维中最常见的问题之一。作为软件开发工程师，掌握系统化的排查方法能够快速定位问题根源，提升问题解决效率。本文将从实战角度出发，详细介绍Pod启动失败的排查思路、常用命令和典型场景。

## Pod生命周期简介

在深入排查之前,理解Pod的生命周期至关重要:

1. **Pending**: Pod已被Kubernetes接受,但容器尚未创建
2. **Running**: Pod已绑定到节点,所有容器已创建,至少一个容器正在运行
3. **Succeeded**: Pod中所有容器成功终止,不会重启
4. **Failed**: Pod中所有容器已终止,至少一个容器失败终止
5. **Unknown**: 无法获取Pod状态,通常是节点通信问题

## 排查流程总览

```
发现Pod异常
    ↓
查看Pod状态 (kubectl get pods)
    ↓
查看详细信息 (kubectl describe pod)
    ↓
检查容器日志 (kubectl logs)
    ↓
查看事件 (kubectl get events)
    ↓
检查节点状态
    ↓
验证配置文件
    ↓
问题定位与解决
```

## 第一步:快速查看Pod状态

### 基础命令

```bash
# 查看所有Pod状态
kubectl get pods -A

# 查看特定命名空间的Pod
kubectl get pods -n <namespace>

# 查看Pod详细信息(包括IP、节点等)
kubectl get pods -o wide

# 持续监控Pod状态变化
kubectl get pods -w
```

### 常见异常状态

| 状态 | 含义 | 常见原因 |
|------|------|----------|
| ImagePullBackOff | 镜像拉取失败 | 镜像不存在、无权限、网络问题 |
| CrashLoopBackOff | 容器启动后崩溃 | 应用错误、配置错误、健康检查失败 |
| Pending | 调度等待中 | 资源不足、节点选择器不匹配、PV挂载失败 |
| ErrImagePull | 镜像拉取错误 | 镜像名称错误、仓库不可达 |
| CreateContainerConfigError | 容器配置错误 | ConfigMap/Secret不存在或格式错误 |
| OOMKilled | 内存溢出被杀 | 内存限制过小或应用内存泄漏 |

## 第二步:查看Pod详细描述

`kubectl describe` 是排查的核心命令,提供最全面的诊断信息。

```bash
kubectl describe pod <pod-name> -n <namespace>
```

### 重点关注的信息段

#### 1. Events事件段

Events记录了Pod生命周期中的关键事件,通常在输出的最后部分:

```bash
Events:
  Type     Reason     Age                From               Message
  ----     ------     ----               ----               -------
  Normal   Scheduled  2m                 default-scheduler  Successfully assigned default/nginx to node1
  Normal   Pulling    2m                 kubelet            Pulling image "nginx:1.19"
  Warning  Failed     2m                 kubelet            Failed to pull image "nginx:1.19": rpc error: code = Unknown desc = Error response from daemon: pull access denied
  Warning  Failed     2m                 kubelet            Error: ErrImagePull
```

#### 2. Conditions状态条件

```yaml
Conditions:
  Type              Status
  Initialized       True    # Init容器是否成功完成
  Ready             False   # Pod是否就绪可服务
  ContainersReady   False   # 所有容器是否就绪
  PodScheduled      True    # Pod是否已调度
```

#### 3. 容器状态

```yaml
Containers:
  app:
    State:          Waiting
      Reason:       CrashLoopBackOff
    Last State:     Terminated
      Reason:       Error
      Exit Code:    1
    Ready:          False
    Restart Count:  5
```

## 第三步:检查容器日志

### 基础日志查看

```bash
# 查看Pod日志
kubectl logs <pod-name> -n <namespace>

# 查看多容器Pod中特定容器的日志
kubectl logs <pod-name> -c <container-name> -n <namespace>

# 查看前一个失败容器的日志(CrashLoopBackOff场景)
kubectl logs <pod-name> --previous -n <namespace>

# 实时跟踪日志
kubectl logs -f <pod-name> -n <namespace>

# 查看最近的日志(最近1小时)
kubectl logs --since=1h <pod-name> -n <namespace>

# 查看最后50行日志
kubectl logs --tail=50 <pod-name> -n <namespace>
```

### Init容器日志

Init容器失败会阻止主容器启动:

```bash
# 查看init容器列表
kubectl describe pod <pod-name> | grep -A 5 "Init Containers"

# 查看init容器日志
kubectl logs <pod-name> -c <init-container-name>
```

## 第四步:查看集群事件

事件提供集群级别的诊断信息:

```bash
# 查看所有事件,按时间排序
kubectl get events --sort-by=.metadata.creationTimestamp -A

# 查看特定命名空间的事件
kubectl get events -n <namespace>

# 过滤特定Pod的事件
kubectl get events --field-selector involvedObject.name=<pod-name> -n <namespace>

# 查看Warning级别的事件
kubectl get events --field-selector type=Warning -A
```

## 典型问题场景与解决方案

### 场景1: ImagePullBackOff - 镜像拉取失败

**症状识别:**
```bash
$ kubectl get pods
NAME                     READY   STATUS             RESTARTS   AGE
myapp-7d8f9c6b5d-xyz12   0/1     ImagePullBackOff   0          2m
```

**详细诊断:**
```bash
$ kubectl describe pod myapp-7d8f9c6b5d-xyz12
...
Events:
  Warning  Failed     2m   kubelet  Failed to pull image "myregistry.com/myapp:v1.0": rpc error: code = Unknown desc = Error response from daemon: pull access denied for myregistry.com/myapp, repository does not exist or may require 'docker login'
```

**常见原因:**

1. **镜像名称或标签错误**
   ```yaml
   # 错误示例
   image: nginx:1.199  # 标签不存在
   
   # 验证方法
   docker pull nginx:1.199  # 在本地测试
   ```

2. **私有仓库认证失败**
   ```bash
   # 检查imagePullSecrets是否配置
   kubectl describe pod <pod-name> | grep "Image Pull Secrets"
   
   # 验证Secret是否存在
   kubectl get secret <secret-name> -n <namespace>
   
   # 创建docker-registry类型的Secret
   kubectl create secret docker-registry regcred \
     --docker-server=<registry-server> \
     --docker-username=<username> \
     --docker-password=<password> \
     --docker-email=<email> \
     -n <namespace>
   ```

3. **网络问题导致无法访问仓库**
   ```bash
   # 在节点上测试连通性
   kubectl debug node/<node-name> -it --image=busybox
   # 然后在调试容器中
   wget -O- https://registry-1.docker.io/v2/
   ```

**解决方案:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  containers:
  - name: app
    image: myregistry.com/myapp:v1.0
  imagePullSecrets:  # 添加认证
  - name: regcred
```

### 场景2: CrashLoopBackOff - 容器反复崩溃

**症状识别:**
```bash
$ kubectl get pods
NAME                     READY   STATUS             RESTARTS   AGE
myapp-7d8f9c6b5d-abc45   0/1     CrashLoopBackOff   5          3m
```

**详细诊断步骤:**

1. **查看容器退出码**
   ```bash
   $ kubectl describe pod myapp-7d8f9c6b5d-abc45
   ...
   Last State:     Terminated
     Reason:       Error
     Exit Code:    137  # 重要!
   ```

   常见退出码含义:
   - `0`: 正常退出
   - `1`: 应用错误
   - `137`: 收到SIGKILL信号(通常是OOM)
   - `139`: 段错误
   - `143`: 收到SIGTERM信号

2. **查看应用日志**
   ```bash
   # 查看当前容器日志
   kubectl logs myapp-7d8f9c6b5d-abc45
   
   # 查看上一次崩溃的日志(更重要!)
   kubectl logs myapp-7d8f9c6b5d-abc45 --previous
   ```

**常见原因:**

1. **应用配置错误**
   ```bash
   # 典型日志示例
   Error: Failed to connect to database: connection refused
   panic: runtime error: invalid memory address
   ```
   
   解决方法:
   - 检查ConfigMap和Secret配置
   - 验证环境变量
   - 检查应用依赖服务是否可达

2. **健康检查配置不当**
   ```yaml
   # 问题配置
   livenessProbe:
     httpGet:
       path: /health
       port: 8080
     initialDelaySeconds: 5  # 太短!应用还没启动完成
     periodSeconds: 3
     failureThreshold: 1     # 太严格!
   ```
   
   优化配置:
   ```yaml
   livenessProbe:
     httpGet:
       path: /health
       port: 8080
     initialDelaySeconds: 30  # 给足启动时间
     periodSeconds: 10
     failureThreshold: 3      # 允许短暂失败
   readinessProbe:
     httpGet:
       path: /ready
       port: 8080
     initialDelaySeconds: 10
     periodSeconds: 5
   ```

3. **OOM内存溢出**
   ```bash
   # 检查是否OOM
   $ kubectl describe pod myapp-7d8f9c6b5d-abc45
   Last State:     Terminated
     Reason:       OOMKilled  # 内存溢出
   ```
   
   解决方案:
   ```yaml
   resources:
     requests:
       memory: "256Mi"
     limits:
       memory: "512Mi"  # 适当增加限制
   ```

### 场景3: Pending - Pod无法调度

**症状识别:**
```bash
$ kubectl get pods
NAME                     READY   STATUS    RESTARTS   AGE
myapp-7d8f9c6b5d-def67   0/1     Pending   0          5m
```

**详细诊断:**
```bash
$ kubectl describe pod myapp-7d8f9c6b5d-def67
...
Events:
  Warning  FailedScheduling  3m   default-scheduler  0/3 nodes are available: 1 node(s) had taint {node-role.kubernetes.io/master: }, that the pod didn't tolerate, 2 Insufficient cpu.
```

**常见原因:**

1. **资源不足**
   ```bash
   # 查看节点资源使用情况
   kubectl top nodes
   
   # 查看节点详细资源分配
   kubectl describe node <node-name>
   ```
   
   输出示例:
   ```
   Allocated resources:
     (Total limits may be over 100 percent, i.e., overcommitted.)
     Resource           Requests    Limits
     --------           --------    ------
     cpu                1950m (97%) 4000m (200%)
     memory             3Gi (75%)   6Gi (150%)
   ```
   
   解决方案:
   - 降低Pod资源请求
   - 扩容集群节点
   - 清理不必要的Pod

2. **节点选择器不匹配**
   ```yaml
   # Pod配置
   spec:
     nodeSelector:
       disktype: ssd  # 要求SSD磁盘
   ```
   
   ```bash
   # 检查节点标签
   kubectl get nodes --show-labels
   
   # 为节点添加标签
   kubectl label nodes <node-name> disktype=ssd
   ```

3. **污点和容忍度问题**
   ```bash
   # 查看节点污点
   kubectl describe node <node-name> | grep Taints
   
   Taints: node-role.kubernetes.io/master:NoSchedule
   ```
   
   添加容忍度:
   ```yaml
   spec:
     tolerations:
     - key: "node-role.kubernetes.io/master"
       operator: "Exists"
       effect: "NoSchedule"
   ```

4. **PersistentVolume绑定失败**
   ```bash
   # 检查PVC状态
   kubectl get pvc -n <namespace>
   
   # 查看PVC详情
   kubectl describe pvc <pvc-name>
   
   # 检查PV
   kubectl get pv
   ```

### 场景4: CreateContainerConfigError - 配置错误

**症状识别:**
```bash
$ kubectl get pods
NAME                     READY   STATUS                       RESTARTS   AGE
myapp-7d8f9c6b5d-ghi89   0/1     CreateContainerConfigError   0          1m
```

**详细诊断:**
```bash
$ kubectl describe pod myapp-7d8f9c6b5d-ghi89
...
Events:
  Warning  Failed  1m  kubelet  Error: configmap "app-config" not found
```

**常见原因:**

1. **引用的ConfigMap或Secret不存在**
   ```bash
   # 列出所有ConfigMap
   kubectl get configmap -n <namespace>
   
   # 列出所有Secret
   kubectl get secret -n <namespace>
   
   # 查看ConfigMap内容
   kubectl describe configmap <configmap-name>
   ```

2. **环境变量配置错误**
   ```yaml
   # 错误示例
   env:
   - name: DB_HOST
     valueFrom:
       configMapKeyRef:
         name: app-config
         key: database_host  # key名称错误
   ```
   
   验证方法:
   ```bash
   kubectl get configmap app-config -o yaml
   ```

3. **Volume挂载配置错误**
   ```yaml
   # 错误示例
   volumes:
   - name: config
     configMap:
       name: app-config
       items:
       - key: config.yaml
         path: app/config.yaml  # 路径不能包含../
   ```

### 场景5: Init容器失败

**症状识别:**
```bash
$ kubectl get pods
NAME                     READY   STATUS     RESTARTS   AGE
myapp-7d8f9c6b5d-jkl01   0/1     Init:0/1   0          2m
```

**详细诊断:**
```bash
# 查看Init容器状态
$ kubectl describe pod myapp-7d8f9c6b5d-jkl01
...
Init Containers:
  init-db:
    State:          Waiting
      Reason:       CrashLoopBackOff
    Last State:     Terminated
      Reason:       Error
      Exit Code:    1

# 查看Init容器日志
kubectl logs myapp-7d8f9c6b5d-jkl01 -c init-db
```

**典型场景:**

Init容器常用于等待依赖服务:
```yaml
initContainers:
- name: init-db
  image: busybox
  command: ['sh', '-c', 'until nslookup mysql-service; do echo waiting for mysql; sleep 2; done']
```

如果Init容器一直失败:
- 检查依赖服务是否正常运行
- 验证网络连通性
- 检查Init容器的逻辑是否正确

## 高级排查技巧

### 1. 使用kubectl debug进入容器

当容器无法启动时,传统的 `kubectl exec` 无法使用。可以使用临时调试容器:

```bash
# 创建临时调试容器(Kubernetes 1.23+)
kubectl debug <pod-name> -it --image=busybox --target=<container-name>

# 在节点上创建调试容器
kubectl debug node/<node-name> -it --image=ubuntu
```

### 2. 检查Pod的详细YAML配置

```bash
# 导出Pod的完整配置
kubectl get pod <pod-name> -o yaml > pod.yaml

# 检查是否有注入的配置(如sidecar)
cat pod.yaml | grep -A 10 "containers:"
```

### 3. 验证RBAC权限

某些Pod需要特定的ServiceAccount权限:

```bash
# 查看Pod使用的ServiceAccount
kubectl get pod <pod-name> -o jsonpath='{.spec.serviceAccountName}'

# 查看ServiceAccount的权限
kubectl describe serviceaccount <sa-name>

# 查看RoleBinding
kubectl get rolebinding -n <namespace>
```

### 4. 检查网络策略

NetworkPolicy可能阻止Pod通信:

```bash
# 列出网络策略
kubectl get networkpolicy -n <namespace>

# 查看详情
kubectl describe networkpolicy <policy-name>
```

### 5. 检查节点状态

```bash
# 查看节点健康状态
kubectl get nodes

# 详细检查节点
kubectl describe node <node-name>

# 查看节点上的Pod分布
kubectl get pods -A -o wide | grep <node-name>

# 检查节点资源压力
kubectl top node <node-name>
```

### 6. 验证存储相关问题

```bash
# 检查StorageClass
kubectl get storageclass

# 查看PV状态
kubectl get pv

# 查看PVC绑定情况
kubectl get pvc -A

# 检查PVC详细信息
kubectl describe pvc <pvc-name>
```

## 实用排查脚本

### 综合诊断脚本

```bash
#!/bin/bash
# pod-diagnose.sh - Pod问题综合诊断脚本

POD_NAME=$1
NAMESPACE=${2:-default}

if [ -z "$POD_NAME" ]; then
    echo "Usage: $0 <pod-name> [namespace]"
    exit 1
fi

echo "=== Pod基本信息 ==="
kubectl get pod $POD_NAME -n $NAMESPACE -o wide

echo -e "\n=== Pod详细描述 ==="
kubectl describe pod $POD_NAME -n $NAMESPACE

echo -e "\n=== Pod事件 ==="
kubectl get events -n $NAMESPACE --field-selector involvedObject.name=$POD_NAME --sort-by='.lastTimestamp'

echo -e "\n=== 容器日志(最近50行) ==="
kubectl logs $POD_NAME -n $NAMESPACE --tail=50

echo -e "\n=== 前一个容器日志(如果存在) ==="
kubectl logs $POD_NAME -n $NAMESPACE --previous --tail=50 2>/dev/null || echo "无前一个容器日志"

echo -e "\n=== Pod配置YAML ==="
kubectl get pod $POD_NAME -n $NAMESPACE -o yaml

echo -e "\n=== 节点信息 ==="
NODE=$(kubectl get pod $POD_NAME -n $NAMESPACE -o jsonpath='{.spec.nodeName}')
if [ -n "$NODE" ]; then
    kubectl describe node $NODE | grep -A 5 "Allocated resources"
fi
```

使用方法:
```bash
chmod +x pod-diagnose.sh
./pod-diagnose.sh myapp-pod-name my-namespace > diagnosis.log
```

### 批量检查异常Pod

```bash
#!/bin/bash
# check-unhealthy-pods.sh

echo "=== 非Running状态的Pod ==="
kubectl get pods -A --field-selector=status.phase!=Running,status.phase!=Succeeded

echo -e "\n=== 重启次数超过5次的Pod ==="
kubectl get pods -A -o json | jq -r '.items[] | select(.status.containerStatuses != null) | select(.status.containerStatuses[].restartCount > 5) | "\(.metadata.namespace)/\(.metadata.name) - Restarts: \(.status.containerStatuses[].restartCount)"'

echo -e "\n=== ImagePullBackOff状态的Pod ==="
kubectl get pods -A | grep ImagePullBackOff

echo -e "\n=== CrashLoopBackOff状态的Pod ==="
kubectl get pods -A | grep CrashLoopBackOff
```

## 日志收集与分析

### 结构化日志查询

如果使用ELK或Loki等日志系统:

```bash
# 使用stern多Pod日志跟踪工具
stern <pod-name-pattern> -n <namespace>

# 按标签过滤
stern -l app=myapp -n production

# 输出为JSON格式
stern <pod-name> --output json
```

### 日志持久化

```bash
# 导出Pod日志到文件
kubectl logs <pod-name> -n <namespace> > pod.log

# 导出所有容器日志
for pod in $(kubectl get pods -n <namespace> -o name); do
    kubectl logs $pod -n <namespace> > ${pod##*/}.log
done
```

## 预防性措施

### 1. 合理的资源配置

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "200m"
```

### 2. 完善的健康检查

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### 3. 镜像拉取策略

```yaml
spec:
  containers:
  - name: app
    image: myapp:v1.0
    imagePullPolicy: IfNotPresent  # 优先使用本地镜像
```

### 4. Pod反亲和性避免单点故障

```yaml
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app
            operator: In
            values:
            - myapp
        topologyKey: kubernetes.io/hostname
```

### 5. 使用PodDisruptionBudget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: myapp-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: myapp
```

## 监控告警建议

### 推荐监控指标

1. **Pod状态监控**
   - Pod重启次数
   - Pod状态异常(非Running)持续时间
   - 容器OOM次数

2. **资源使用监控**
   - CPU使用率超过80%
   - 内存使用率超过80%
   - Pod调度失败次数

3. **应用健康监控**
   - 健康检查失败率
   - 启动时间过长告警

### Prometheus监控示例

```yaml
# 告警规则示例
groups:
- name: pod_alerts
  rules:
  - alert: PodCrashLooping
    expr: rate(kube_pod_container_status_restarts_total[15m]) > 0
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Pod {{ $labels.pod }} is crash looping"
      
  - alert: PodNotReady
    expr: kube_pod_status_phase{phase!~"Running|Succeeded"} > 0
    for: 10m
    labels:
      severity: critical
    annotations:
      summary: "Pod {{ $labels.pod }} not ready for 10 minutes"
```

## 常用工具推荐

1. **kubectl-debug**: 容器调试工具
2. **stern**: 多Pod日志聚合查看
3. **k9s**: 终端UI管理工具
4. **kubectx/kubens**: 快速切换上下文和命名空间
5. **popeye**: 集群健康检查工具
6. **kube-capacity**: 资源容量查看工具

安装示例:
```bash
# kubectl-debug
kubectl krew install debug

# stern
brew install stern

# k9s
brew install k9s

# popeye
brew install derailed/popeye/popeye
```

## 总结

Pod启动失败的排查遵循以下黄金法则:

1. **从现象到本质**: 先看状态,再看事件,最后看日志
2. **由上至下**: 从Pod到容器,从应用到基础设施
3. **关注时间线**: Events按时间排序能还原问题发生过程
4. **验证假设**: 每个排查步骤都要有明确目的
5. **记录过程**: 保存日志和配置便于后续分析

掌握这套系统化的排查方法,能够快速定位90%以上的Pod启动问题。记住,经验来自于实践,多动手操作才能真正理解Kubernetes的运行机制。

---

## 高频常见问题FAQ

### Q1: Pod一直处于Pending状态超过10分钟,最可能是什么原因?

**A**: Pending状态表示Pod已被接受但未能调度到节点。最常见的三个原因按优先级排序:

1. **资源不足(80%概率)**: 使用 `kubectl describe pod <pod-name>` 查看Events,如果看到 "Insufficient cpu" 或 "Insufficient memory",说明集群所有节点资源都不足以满足Pod的资源请求(requests)。

2. **节点选择器不匹配(15%概率)**: Pod的 `nodeSelector` 或 `affinity` 配置要求特定标签的节点,但集群中没有匹配的节点。

3. **PersistentVolume未绑定(5%概率)**: Pod需要挂载PVC,但PVC处于Pending状态无法绑定到PV。

**快速诊断命令**:
```bash
kubectl describe pod <pod-name> | grep -A 10 Events
kubectl top nodes  # 查看节点资源
kubectl get nodes --show-labels  # 检查节点标签
```

### Q2: CrashLoopBackOff和ImagePullBackOff都带"BackOff",有什么区别?

**A**: 这是两个完全不同阶段的问题:

- **ImagePullBackOff**: 发生在容器创建前,无法拉取镜像。原因通常是镜像不存在、无权限或网络问题。此时容器还未运行。

- **CrashLoopBackOff**: 发生在容器运行时,容器成功启动但随后崩溃退出。Kubernetes按照指数退避策略(10s、20s、40s...最多5分钟)重启容器。原因通常是应用错误、配置问题或OOM。

**关键区别**:
- ImagePullBackOff: 查看 `kubectl describe pod` 中的Events和镜像拉取日志
- CrashLoopBackOff: 必须查看 `kubectl logs <pod> --previous` 获取崩溃前日志

### Q3: 如何区分是应用自身bug还是Kubernetes配置问题导致的Pod失败?

**A**: 按以下决策树判断:

1. **先看Exit Code**:
   ```bash
   kubectl describe pod <pod-name> | grep "Exit Code"
   ```
   - Exit Code 0: 正常退出,可能是Job类型Pod完成任务
   - Exit Code 1/2: 应用层错误,99%是代码问题
   - Exit Code 137: OOM被杀,可能是配置问题(内存限制太小)或应用内存泄漏
   - Exit Code 139: 段错误,通常是应用bug

2. **查看日志关键字**:
   ```bash
   kubectl logs <pod-name> --previous
   ```
   - 看到 "panic"、"NullPointerException"、"Segmentation fault": 应用bug
   - 看到 "connection refused"、"cannot connect to": 配置问题或依赖服务问题
   - 看到 "permission denied": RBAC或文件权限配置问题

3. **如果日志为空或无明显错误**: 检查健康检查配置是否过于严格,导致应用还没启动完就被杀掉。

**实战技巧**: 如果怀疑是K8s配置问题,可以先用 `docker run` 在本地运行同样的镜像和环境变量,如果本地能正常运行,则大概率是K8s配置问题。

### Q4: Pod日志显示正常,但Ready状态一直是0/1,如何排查?

**A**: Ready为False表示readinessProbe健康检查失败。这是最容易被忽视的配置问题:

```bash
# 查看健康检查配置
kubectl get pod <pod-name> -o yaml | grep -A 10 readinessProbe
```

**常见原因**:

1. **检查路径或端口配置错误**:
   ```yaml
   readinessProbe:
     httpGet:
       path: /health  # 确认应用确实暴露了这个路径
       port: 8080     # 确认端口正确
   ```

2. **initialDelaySeconds太短**: 应用启动需要10秒,但配置只给了3秒:
   ```yaml
   readinessProbe:
     httpGet:
       path: /health
       port: 8080
     initialDelaySeconds: 3  # 太短!改为15-30秒
   ```

3. **依赖服务未就绪**: 应用连接数据库失败导致健康检查失败,但日志没有明显报错。

**验证方法**:
```bash
# 进入Pod手动测试健康检查接口
kubectl exec -it <pod-name> -- wget -O- http://localhost:8080/health
# 或
kubectl exec -it <pod-name> -- curl http://localhost:8080/health
```

### Q5: 多个Pod同时启动失败,如何快速定位是节点问题还是应用问题?

**A**: 这种集群级别的问题需要横向对比分析:

1. **检查是否在同一节点**:
   ```bash
   kubectl get pods -o wide | grep <pod-pattern>
   ```
   如果所有失败Pod都在同一节点,90%是节点问题。

2. **验证节点健康**:
   ```bash
   kubectl describe node <node-name> | grep -i "condition\|pressure"
   ```
   查看是否有:
   - MemoryPressure: True (内存压力)
   - DiskPressure: True (磁盘压力)
   - PIDPressure: True (进程数压力)
   - Ready: False (节点不健康)

3. **检查是否是同时部署**:
   ```bash
   kubectl get events --sort-by='.lastTimestamp' | head -20
   ```
   如果是同时部署的多个Pod都失败,可能是:
   - 镜像仓库问题(同时拉取失败)
   - ConfigMap/Secret被误删除
   - 网络策略配置错误

4. **检查系统级日志**:
   ```bash
   # 查看节点kubelet日志
   kubectl logs -n kube-system <kubelet-pod> | tail -50
   ```

**快速判断技巧**: 
- 如果只有特定应用的Pod失败: 应用配置问题
- 如果某节点上所有新Pod都失败: 节点问题
- 如果集群范围内随机失败: 基础设施问题(网络、存储、镜像仓库)
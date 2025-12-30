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

# Pod 拉不起应该怎么排查

## 系统化排查流程

**排查原则**：从外到内，从简到繁，逐层深入

```
排查顺序：
1. 查看 Pod 状态 (kubectl get/describe)
2. 查看容器日志 (kubectl logs)
3. 查看事件信息 (kubectl events)
4. 检查资源配额 (Resource Quota)
5. 检查节点状态 (Node)
6. 检查网络策略 (Network Policy)
7. 检查存储卷 (PV/PVC)
8. 检查安全策略 (Security Context)
```

## 常见 Pod 状态及排查方法

| 状态                           | 说明              | 常见原因                         | 排查方法                 |
| ------------------------------ | ----------------- | -------------------------------- | ------------------------ |
| **Pending**                    | Pod已创建但未调度 | 资源不足、节点选择器、污点容忍   | 检查资源、节点、调度策略 |
| **ImagePullBackOff**           | 镜像拉取失败      | 镜像不存在、认证失败、网络问题   | 检查镜像名、Secret、网络 |
| **CrashLoopBackOff**           | 容器反复崩溃      | 应用错误、配置错误、健康检查失败 | 查看日志、检查配置       |
| **Error**                      | 容器异常退出      | 应用崩溃、OOM、启动失败          | 查看日志、资源限制       |
| **Running**                    | 运行中但不正常    | 应用逻辑错误、探针失败           | 查看日志、检查探针       |
| **Terminated**                 | 容器已终止        | 任务完成或异常退出               | 查看退出码和原因         |
| **Unknown**                    | 状态未知          | 节点失联、kubelet问题            | 检查节点、kubelet        |
| **CreateContainerConfigError** | 容器配置错误      | ConfigMap/Secret缺失             | 检查配置资源             |
| **Init:Error**                 | Init容器失败      | Init容器异常                     | 查看Init容器日志         |

## 详细排查步骤

**第一步：查看 Pod 基本信息**

```bash
# 查看 Pod 状态
kubectl get pod <pod-name> -n <namespace>

# 查看 Pod 详细信息
kubectl get pod <pod-name> -n <namespace> -o wide

# 查看 Pod 完整配置
kubectl get pod <pod-name> -n <namespace> -o yaml

# 查看所有 Pod（包括所有命名空间）
kubectl get pod --all-namespaces
```

**输出示例**：
```
NAME         READY   STATUS             RESTARTS   AGE
my-app-pod   0/1     ImagePullBackOff   0          5m
```

**第二步：查看 Pod 详细描述**

```bash
# 查看 Pod 详细信息和事件
kubectl describe pod <pod-name> -n <namespace>
```

**关键信息**：
- **Events**: 最重要的排查信息
- **Conditions**: Pod 的状态条件
- **Containers**: 容器状态和配置
- **Volumes**: 存储卷挂载情况
- **QoS Class**: 服务质量等级

**第三步：查看容器日志**

```bash
# 查看当前容器日志
kubectl logs <pod-name> -n <namespace>

# 查看指定容器日志（多容器Pod）
kubectl logs <pod-name> -c <container-name> -n <namespace>

# 查看上一次容器日志（容器重启后）
kubectl logs <pod-name> --previous -n <namespace>

# 实时查看日志
kubectl logs -f <pod-name> -n <namespace>

# 查看最近N行日志
kubectl logs --tail=100 <pod-name> -n <namespace>

# 查看Init容器日志
kubectl logs <pod-name> -c <init-container-name> -n <namespace>
```

**第四步：查看事件信息**

```bash
# 查看命名空间的所有事件
kubectl get events -n <namespace> --sort-by='.lastTimestamp'

# 查看特定Pod的事件
kubectl get events --field-selector involvedObject.name=<pod-name> -n <namespace>

# 查看所有命名空间的事件
kubectl get events --all-namespaces --sort-by='.lastTimestamp'
```

**第五步：进入容器调试**

```bash
# 进入容器（如果容器正在运行）
kubectl exec -it <pod-name> -n <namespace> -- /bin/sh
kubectl exec -it <pod-name> -n <namespace> -- /bin/bash

# 在容器中执行命令
kubectl exec <pod-name> -n <namespace> -- ps aux
kubectl exec <pod-name> -n <namespace> -- netstat -tunlp
kubectl exec <pod-name> -n <namespace> -- df -h

# 多容器Pod指定容器
kubectl exec -it <pod-name> -c <container-name> -n <namespace> -- /bin/sh
```

**第六步：使用临时调试容器（Kubernetes 1.18+）**

```bash
# 创建临时调试容器
kubectl debug <pod-name> -it --image=busybox -n <namespace>

# 在节点上调试
kubectl debug node/<node-name> -it --image=ubuntu
```

## 常见问题及解决方案

**问题1: Pending - 资源不足**

**现象**：
```yaml
Status: Pending
Events:
  Warning  FailedScheduling  0/3 nodes are available: 3 Insufficient cpu.
```

**原因**：
- CPU/内存资源不足
- 节点资源已被占满
- 资源请求过大

**排查**：
```bash
# 查看节点资源使用情况
kubectl top nodes

# 查看节点详细信息
kubectl describe node <node-name>

# 查看资源配额
kubectl describe resourcequota -n <namespace>

# 查看Pod资源请求
kubectl describe pod <pod-name> -n <namespace> | grep -A 5 "Requests"
```

**解决方案**：
```yaml
# 1. 调整资源请求和限制
resources:
  requests:
    memory: "64Mi"
    cpu: "100m"
  limits:
    memory: "128Mi"
    cpu: "500m"

# 2. 增加节点
# 3. 删除不必要的Pod释放资源
# 4. 使用水平扩展增加节点容量
```

**问题2: Pending - 节点选择器不匹配**

**现象**：
```yaml
Status: Pending
Events:
  Warning  FailedScheduling  0/3 nodes are available: 3 node(s) didn't match node selector.
```

**排查**：
```bash
# 查看Pod的节点选择器
kubectl get pod <pod-name> -n <namespace> -o yaml | grep -A 5 nodeSelector

# 查看节点标签
kubectl get nodes --show-labels

# 查看特定节点的标签
kubectl describe node <node-name> | grep Labels -A 10
```

**解决方案**：
```yaml
# 方法1: 修改Pod的节点选择器
nodeSelector:
  disktype: ssd  # 确保节点有此标签

# 方法2: 给节点添加标签
# kubectl label nodes <node-name> disktype=ssd

# 方法3: 删除节点选择器（如果不需要）
```

**问题3: Pending - 污点和容忍**

**现象**：
```yaml
Status: Pending
Events:
  Warning  FailedScheduling  0/3 nodes are available: 3 node(s) had taints that the pod didn't tolerate.
```

**排查**：
```bash
# 查看节点污点
kubectl describe node <node-name> | grep Taints

# 查看Pod的容忍配置
kubectl get pod <pod-name> -n <namespace> -o yaml | grep -A 5 tolerations
```

**解决方案**：
```yaml
# 添加容忍配置
tolerations:
- key: "key1"
  operator: "Equal"
  value: "value1"
  effect: "NoSchedule"
- key: "node.kubernetes.io/not-ready"
  operator: "Exists"
  effect: "NoExecute"
  tolerationSeconds: 300
```

**问题4: ImagePullBackOff - 镜像拉取失败**

**现象**：
```yaml
Status: ImagePullBackOff
Events:
  Warning  Failed  Failed to pull image "myapp:v1.0": rpc error: code = Unknown desc = Error response from daemon: pull access denied
```

**常见原因**：
1. 镜像名称或标签错误
2. 私有镜像仓库认证失败
3. 网络问题无法访问镜像仓库
4. 镜像不存在

**排查**：
```bash
# 查看镜像配置
kubectl describe pod <pod-name> -n <namespace> | grep -A 10 "Image"

# 查看imagePullSecrets
kubectl get pod <pod-name> -n <namespace> -o yaml | grep -A 5 imagePullSecrets

# 查看Secret
kubectl get secret <secret-name> -n <namespace> -o yaml

# 在节点上手动拉取镜像测试
ssh <node>
docker pull <image-name>
# 或
crictl pull <image-name>
```

**解决方案**：
```yaml
# 方法1: 修正镜像名称
image: myregistry.com/myapp:v1.0  # 确保名称正确

# 方法2: 创建镜像拉取Secret
# kubectl create secret docker-registry my-secret \
#   --docker-server=myregistry.com \
#   --docker-username=myuser \
#   --docker-password=mypassword \
#   --docker-email=myemail@example.com \
#   -n <namespace>

# 方法3: 在Pod中引用Secret
spec:
  imagePullSecrets:
  - name: my-secret
  containers:
  - name: myapp
    image: myregistry.com/myapp:v1.0

# 方法4: 使用公共镜像或更改镜像拉取策略
imagePullPolicy: IfNotPresent  # 或 Always, Never
```

**问题5: CrashLoopBackOff - 容器反复崩溃**

**现象**：
```yaml
Status: CrashLoopBackOff
Restart Count: 5
```

**常见原因**：
1. 应用启动失败（配置错误、依赖缺失）
2. 应用崩溃（代码bug、未捕获异常）
3. 健康检查失败
4. 资源限制（OOMKilled）
5. 启动命令错误

**排查**：
```bash
# 查看容器日志（当前和上一次）
kubectl logs <pod-name> -n <namespace>
kubectl logs <pod-name> --previous -n <namespace>

# 查看退出码
kubectl describe pod <pod-name> -n <namespace> | grep "Exit Code"

# 查看重启次数和原因
kubectl describe pod <pod-name> -n <namespace> | grep -A 10 "State"

# 常见退出码：
# 0: 正常退出
# 1: 应用错误
# 137: 被SIGKILL杀死（通常是OOM）
# 143: 被SIGTERM终止
# 255: 退出码超出范围
```

**解决方案**：
```yaml
# 1. 修复应用代码或配置
# 2. 调整资源限制
resources:
  limits:
    memory: "512Mi"  # 增加内存限制

# 3. 调整或禁用健康检查（临时）
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30  # 增加初始延迟
  periodSeconds: 10
  failureThreshold: 3      # 增加失败阈值

# 4. 修改启动命令
command: ["/bin/sh"]
args: ["-c", "echo starting && /app/start.sh"]

# 5. 使用startupProbe（Kubernetes 1.18+）
startupProbe:
  httpGet:
    path: /health
    port: 8080
  failureThreshold: 30
  periodSeconds: 10
```

**问题6: CreateContainerConfigError - 配置错误**

**现象**：
```yaml
Status: CreateContainerConfigError
Events:
  Warning  Failed  Error: configmap "my-config" not found
```

**常见原因**：
- ConfigMap 不存在
- Secret 不存在
- 引用的key不存在
- 命名空间不匹配

**排查**：
```bash
# 查看ConfigMap
kubectl get configmap -n <namespace>
kubectl describe configmap <configmap-name> -n <namespace>

# 查看Secret
kubectl get secret -n <namespace>
kubectl describe secret <secret-name> -n <namespace>

# 查看Pod引用的配置
kubectl get pod <pod-name> -n <namespace> -o yaml | grep -A 10 "configMap\|secret"
```

**解决方案**：
```bash
# 创建缺失的ConfigMap
kubectl create configmap my-config --from-literal=key1=value1 -n <namespace>

# 创建缺失的Secret
kubectl create secret generic my-secret --from-literal=password=mypassword -n <namespace>

# 或修改Pod配置，使用正确的资源名称
```

**问题7: PVC挂载失败**

**现象**：
```yaml
Status: ContainerCreating
Events:
  Warning  FailedMount  Unable to attach or mount volumes: unmounted volumes=[my-volume]
```

**排查**：
```bash
# 查看PVC状态
kubectl get pvc -n <namespace>

# 查看PVC详情
kubectl describe pvc <pvc-name> -n <namespace>

# 查看PV
kubectl get pv

# 查看StorageClass
kubectl get storageclass
```

**解决方案**：
```yaml
# 1. 确保PVC已绑定
# 2. 检查StorageClass是否存在
# 3. 检查节点是否支持该存储类型
# 4. 查看存储提供商的日志（如CSI驱动）
```

**问题8: 节点问题导致Pod不可用**

**现象**：
```yaml
Status: Unknown
Node: <node-name>
Events:
  Warning  NodeNotReady  Node is not ready
```

**排查**：
```bash
# 查看节点状态
kubectl get nodes

# 查看节点详情
kubectl describe node <node-name>

# 查看kubelet日志（SSH到节点）
journalctl -u kubelet -f

# 检查节点资源
kubectl top node <node-name>
```

**解决方案**：
- 重启kubelet服务
- 检查节点网络连接
- 检查磁盘空间
- 检查节点资源压力

## 实用排查脚本

**快速诊断脚本**：

```bash
#!/bin/bash
# pod-debug.sh - Pod问题快速诊断脚本

POD_NAME=$1
NAMESPACE=${2:-default}

if [ -z "$POD_NAME" ]; then
    echo "用法: $0 <pod-name> [namespace]"
    exit 1
fi

echo "===== Pod基本信息 ====="
kubectl get pod $POD_NAME -n $NAMESPACE -o wide

echo -e "\n===== Pod状态详情 ====="
kubectl describe pod $POD_NAME -n $NAMESPACE | grep -A 20 "^Status:\|^Conditions:\|^State:"

echo -e "\n===== 最近事件 ====="
kubectl get events --field-selector involvedObject.name=$POD_NAME -n $NAMESPACE --sort-by='.lastTimestamp' | tail -10

echo -e "\n===== 容器状态 ====="
kubectl get pod $POD_NAME -n $NAMESPACE -o jsonpath='{range .status.containerStatuses[*]}{.name}{"\t"}{.ready}{"\t"}{.restartCount}{"\t"}{.state}{"\n"}{end}'

echo -e "\n===== 资源请求和限制 ====="
kubectl describe pod $POD_NAME -n $NAMESPACE | grep -A 5 "Limits:\|Requests:"

echo -e "\n===== 最近日志 (最后20行) ====="
kubectl logs --tail=20 $POD_NAME -n $NAMESPACE 2>/dev/null || echo "无法获取日志"

echo -e "\n===== 上一次容器日志 (如果重启过) ====="
kubectl logs --previous --tail=20 $POD_NAME -n $NAMESPACE 2>/dev/null || echo "没有上一次日志"

echo -e "\n===== 节点信息 ====="
NODE=$(kubectl get pod $POD_NAME -n $NAMESPACE -o jsonpath='{.spec.nodeName}')
if [ -n "$NODE" ]; then
    echo "节点: $NODE"
    kubectl describe node $NODE | grep -A 5 "Allocated resources:"
fi
```

**使用方法**：
```bash
chmod +x pod-debug.sh
./pod-debug.sh my-pod-name my-namespace
```

**批量检查脚本**：

```bash
#!/bin/bash
# check-all-pods.sh - 检查所有异常Pod

echo "===== 检查所有命名空间的异常Pod ====="

# 查找非Running且非Completed的Pod
kubectl get pods --all-namespaces --field-selector=status.phase!=Running,status.phase!=Succeeded -o wide

echo -e "\n===== ImagePullBackOff状态的Pod ====="
kubectl get pods --all-namespaces | grep ImagePullBackOff

echo -e "\n===== CrashLoopBackOff状态的Pod ====="
kubectl get pods --all-namespaces | grep CrashLoopBackOff

echo -e "\n===== Pending状态的Pod ====="
kubectl get pods --all-namespaces | grep Pending

echo -e "\n===== 重启次数超过5次的Pod ====="
kubectl get pods --all-namespaces -o json | jq -r '.items[] | select(.status.containerStatuses[]?.restartCount > 5) | "\(.metadata.namespace)\t\(.metadata.name)\t\(.status.containerStatuses[].restartCount)"'

echo -e "\n===== 节点资源使用情况 ====="
kubectl top nodes
```

## 排查检查清单

**基础检查**：
- [ ] Pod状态是什么？（Pending/Running/CrashLoopBackOff等）
- [ ] 重启次数是多少？
- [ ] 最近的事件信息是什么？
- [ ] 容器日志显示什么错误？

**资源检查**：
- [ ] CPU/内存请求是否合理？
- [ ] 节点资源是否充足？
- [ ] 是否触发了ResourceQuota限制？
- [ ] 是否发生了OOM？

**调度检查**：
- [ ] 是否有节点选择器（nodeSelector）？
- [ ] 节点是否有匹配的标签？
- [ ] 是否有污点（Taints）需要容忍？
- [ ] 亲和性和反亲和性配置是否正确？

**镜像检查**：
- [ ] 镜像名称和标签是否正确？
- [ ] 镜像拉取策略是什么？
- [ ] 是否需要imagePullSecrets？
- [ ] 镜像仓库是否可访问？

**配置检查**：
- [ ] ConfigMap是否存在？
- [ ] Secret是否存在？
- [ ] 环境变量配置是否正确？
- [ ] 挂载路径是否冲突？

**存储检查**：
- [ ] PVC是否已绑定？
- [ ] PV是否可用？
- [ ] StorageClass是否存在？
- [ ] 存储卷挂载是否成功？

**网络检查**：
- [ ] Service是否正确指向Pod？
- [ ] NetworkPolicy是否阻止了流量？
- [ ] DNS解析是否正常？
- [ ] 端口配置是否正确？

**安全检查**：
- [ ] SecurityContext配置是否正确？
- [ ] ServiceAccount是否存在？
- [ ] RBAC权限是否足够？
- [ ] PodSecurityPolicy是否阻止？

## 高级排查技巧

**1. 使用kubectl get pod -o yaml查看完整状态**

```bash
kubectl get pod <pod-name> -n <namespace> -o yaml
```

关注字段：
- `status.conditions`: Pod状态条件
- `status.containerStatuses`: 容器状态详情
- `status.initContainerStatuses`: Init容器状态
- `status.message`: 状态消息
- `status.reason`: 状态原因

**2. 查看Pod的QoS等级**

```bash
kubectl get pod <pod-name> -n <namespace> -o jsonpath='{.status.qosClass}'
```

QoS等级：
- **Guaranteed**: 所有容器都有相等的requests和limits
- **Burstable**: 至少一个容器有requests或limits
- **BestEffort**: 没有任何requests和limits（最容易被驱逐）

**3. 模拟Pod调度**

```bash
# 查看Pod为什么不能调度到节点
kubectl get pod <pod-name> -n <namespace> -o yaml > pod.yaml
kubectl describe node <node-name>
```

**4. 临时禁用探针**

```bash
# 编辑Pod（仅限临时调试）
kubectl edit pod <pod-name> -n <namespace>

# 注释掉livenessProbe和readinessProbe
# 注意：这只对新创建的Pod有效，或者编辑Deployment等控制器
```

**5. 增加日志详细程度**

```bash
# 修改kubelet日志级别（在节点上）
# /var/lib/kubelet/config.yaml
# 添加: --v=5

# 查看更详细的Pod信息
kubectl describe pod <pod-name> -n <namespace> --v=8
```

---

## 相关面试题

### Q1: Pending、ImagePullBackOff、CrashLoopBackOff这三种状态的主要区别和排查重点是什么？

**答案**：

**Pending**：
- **含义**：Pod已被创建但未被调度到节点
- **排查重点**：调度问题
  - 资源不足：`kubectl top nodes` 检查资源
  - 节点选择器：检查nodeSelector和节点标签
  - 污点容忍：检查node taints和tolerations
  - 亲和性：检查affinity配置
- **关键命令**：`kubectl describe pod` 查看Scheduling事件

**ImagePullBackOff**：
- **含义**：镜像拉取失败，kubelet正在重试
- **排查重点**：镜像访问问题
  - 镜像名称是否正确
  - 镜像仓库认证（imagePullSecrets）
  - 网络连接性
  - 镜像是否存在
- **关键命令**：`kubectl describe pod` 查看Failed pull事件，检查镜像名称和Secret

**CrashLoopBackOff**：
- **含义**：容器启动后反复崩溃，kubelet正在回退重试
- **排查重点**：应用运行时问题
  - 应用启动错误：查看日志
  - 配置错误：检查ConfigMap/Secret
  - 健康检查失败：检查probe配置
  - 资源限制：检查是否OOM（exit code 137）
- **关键命令**：`kubectl logs --previous` 查看崩溃前的日志

### Q2: 如何判断Pod是因为OOM（内存溢出）被杀死的？如何解决？

**答案**：

**判断方法**：

1. **查看退出码**：
```bash
kubectl describe pod <pod-name> | grep "Exit Code"
# Exit Code: 137 表示被SIGKILL杀死（通常是OOM）
```

2. **查看容器状态**：
```bash
kubectl describe pod <pod-name> | grep -A 10 "Last State:"
# 输出: Reason: OOMKilled
```

3. **查看节点事件**：
```bash
kubectl get events --all-namespaces | grep OOM
```

4. **查看容器指标**：
```bash
kubectl top pod <pod-name>
```

**解决方案**：

```yaml
# 1. 增加内存限制
resources:
  limits:
    memory: "1Gi"  # 从512Mi增加到1Gi
  requests:
    memory: "512Mi"

# 2. 优化应用内存使用
# - 修复内存泄漏
# - 减少缓存大小
# - 优化数据结构

# 3. 使用QoS Guaranteed（防止被驱逐）
resources:
  limits:
    memory: "1Gi"
    cpu: "1000m"
  requests:
    memory: "1Gi"  # 与limits相同
    cpu: "1000m"

# 4. 启用内存监控告警
# 使用Prometheus监控内存使用趋势
```

### Q3: Init容器失败会导致什么问题？如何排查？

**答案**：

**影响**：
- Pod停留在`Init:Error`或`Init:CrashLoopBackOff`状态
- 主容器不会启动
- 整个Pod无法正常工作

**排查步骤**：

```bash
# 1. 查看Pod状态
kubectl get pod <pod-name>
# 显示: Init:0/2 表示有2个Init容器，第1个失败

# 2. 查看Init容器列表
kubectl describe pod <pod-name> | grep -A 20 "Init Containers:"

# 3. 查看Init容器日志
kubectl logs <pod-name> -c <init-container-name>

# 4. 查看Init容器的退出码和原因
kubectl describe pod <pod-name> | grep -A 10 "State:"
```

**常见失败原因**：
- 依赖服务未就绪
- 初始化脚本错误
- 配置文件下载失败
- 数据库迁移失败

**解决方案**：
```yaml
# 调整Init容器配置
initContainers:
- name: init-myservice
  image: busybox
  command: ['sh', '-c']
  args:
  - |
    until nslookup myservice; do
      echo waiting for myservice
      sleep 2
    done
  # 增加资源和超时配置
  resources:
    limits:
      cpu: 100m
      memory: 128Mi
```

### Q4: 如何排查Pod可以创建但Service访问不通的问题？

**答案**：

**排查步骤**：

```bash
# 1. 确认Pod运行正常
kubectl get pod -l app=myapp
kubectl logs <pod-name>

# 2. 检查Service配置
kubectl get svc myapp-service -o yaml

# 3. 检查Service的Endpoints
kubectl get endpoints myapp-service
# 如果为空或不完整，说明selector不匹配

# 4. 验证selector和label
kubectl get pod -l app=myapp --show-labels
kubectl describe svc myapp-service | grep Selector

# 5. 检查端口配置
# Service的targetPort必须与Pod的containerPort匹配

# 6. 测试Pod网络
kubectl exec -it <pod-name> -- wget -O- http://localhost:8080
kubectl exec -it <pod-name> -- netstat -tunlp

# 7. 检查NetworkPolicy
kubectl get networkpolicy -n <namespace>

# 8. 从另一个Pod测试Service
kubectl run test-pod --image=busybox --rm -it -- wget -O- http://myapp-service:80
```

**常见问题**：
- **Selector不匹配**：Service的selector与Pod的labels不一致
- **端口配置错误**：targetPort与containerPort不匹配
- **Ready探针失败**：Pod未通过readinessProbe，不会加入endpoints
- **NetworkPolicy阻止**：网络策略限制了流量

**解决示例**：
```yaml
# Service配置
apiVersion: v1
kind: Service
metadata:
  name: myapp-service
spec:
  selector:
    app: myapp  # 必须与Pod标签匹配
  ports:
  - port: 80
    targetPort: 8080  # 必须与Pod的containerPort匹配
    protocol: TCP

# Pod配置
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: myapp  # 与Service selector匹配
spec:
  containers:
  - name: myapp
    image: myapp:v1
    ports:
    - containerPort: 8080  # 与Service targetPort匹配
```

### Q5: 如何排查节点上Pod无法调度的问题（节点有足够资源但Pod仍Pending）？

**答案**：

**可能原因**：

1. **节点污点（Taints）**
2. **节点选择器不匹配**
3. **亲和性/反亲和性规则**
4. **PodDisruptionBudget限制**
5. **节点cordoned（禁止调度）**

**排查步骤**：

```bash
# 1. 查看Pod调度事件
kubectl describe pod <pod-name> | grep Events -A 20

# 2. 检查节点是否可调度
kubectl get nodes
# 查看STATUS列，如果是Ready,SchedulingDisabled则被禁止调度

# 3. 查看节点污点
kubectl describe node <node-name> | grep Taints
# 示例输出: node-role.kubernetes.io/master:NoSchedule

# 4. 查看Pod的容忍配置
kubectl get pod <pod-name> -o yaml | grep -A 10 tolerations

# 5. 检查节点选择器
kubectl get pod <pod-name> -o yaml | grep -A 5 nodeSelector
kubectl get node <node-name> --show-labels

# 6. 检查亲和性规则
kubectl get pod <pod-name> -o yaml | grep -A 20 affinity

# 7. 模拟调度决策
kubectl describe node <node-name> | grep -A 10 "Allocated resources"
```

**解决方案**：

```yaml
# 方案1: 添加容忍配置
tolerations:
- key: "node-role.kubernetes.io/master"
  operator: "Exists"
  effect: "NoSchedule"

# 方案2: 移除或修改节点污点
# kubectl taint nodes <node-name> node-role.kubernetes.io/master:NoSchedule-

# 方案3: 允许节点调度
# kubectl uncordon <node-name>

# 方案4: 修改节点选择器或添加标签
nodeSelector:
  disktype: ssd
# kubectl label nodes <node-name> disktype=ssd

# 方案5: 调整亲和性规则
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
          - node1
          - node2
```

### Q6: Pod卡在Terminating状态无法删除怎么办？

**答案**：

**常见原因**：
- Finalizers未完成
- 存储卷无法卸载
- kubelet无法联系到节点
- 容器无法停止

**排查和解决**：

```bash
# 1. 查看Pod详情
kubectl describe pod <pod-name> -n <namespace>

# 2. 查看Finalizers
kubectl get pod <pod-name> -n <namespace> -o yaml | grep -A 5 finalizers

# 3. 尝试强制删除（等待grace period）
kubectl delete pod <pod-name> -n <namespace> --grace-period=0

# 4. 强制立即删除（不推荐，可能导致数据丢失）
kubectl delete pod <pod-name> -n <namespace> --grace-period=0 --force

# 5. 如果仍无法删除，移除Finalizers
kubectl patch pod <pod-name> -n <namespace> -p '{"metadata":{"finalizers":null}}'

# 6. 检查节点状态
kubectl get nodes
# 如果节点NotReady，可能需要：
kubectl delete node <node-name>  # 然后重新加入节点

# 7. 检查kubelet日志（SSH到节点）
journalctl -u kubelet -f | grep <pod-name>
```

**预防措施**：
```yaml
# 设置合理的terminationGracePeriodSeconds
spec:
  terminationGracePeriodSeconds: 30  # 默认30秒

  # 容器应正确处理SIGTERM信号
  containers:
  - name: myapp
    lifecycle:
      preStop:
        exec:
          command: ["/bin/sh", "-c", "sleep 10"]  # 优雅关闭
```

### Q7: 如何排查Pod间网络不通的问题？

**答案**：

**排查步骤**：

```bash
# 1. 确认两个Pod都在运行
kubectl get pod pod-a pod-b -o wide

# 2. 获取Pod IP
kubectl get pod pod-b -o jsonpath='{.status.podIP}'

# 3. 从pod-a ping pod-b
kubectl exec pod-a -- ping <pod-b-ip>

# 4. 测试端口连通性
kubectl exec pod-a -- telnet <pod-b-ip> 8080
kubectl exec pod-a -- nc -zv <pod-b-ip> 8080

# 5. 检查NetworkPolicy
kubectl get networkpolicy -n <namespace>
kubectl describe networkpolicy <policy-name>

# 6. 检查DNS解析
kubectl exec pod-a -- nslookup kubernetes.default
kubectl exec pod-a -- nslookup pod-b-service

# 7. 检查iptables规则（在节点上）
iptables -L -n -v | grep <pod-ip>

# 8. 检查CNI插件状态
# Calico
kubectl get pods -n kube-system | grep calico
calicoctl node status

# Flannel  
kubectl get pods -n kube-system | grep flannel

# 9. 检查kube-proxy
kubectl get pods -n kube-system | grep kube-proxy
kubectl logs -n kube-system kube-proxy-xxxxx
```

**常见问题和解决**：

```yaml
# 问题1: NetworkPolicy阻止
# 解决: 添加允许规则
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-pod-a
spec:
  podSelector:
    matchLabels:
      app: pod-b
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: pod-a
    ports:
    - protocol: TCP
      port: 8080

# 问题2: CNI插件故障
# 解决: 重启CNI相关Pod
kubectl delete pod -n kube-system -l k8s-app=calico-node

# 问题3: kube-proxy问题
# 解决: 重启kube-proxy
kubectl delete pod -n kube-system -l k8s-app=kube-proxy
```

### Q8: 生产环境中，如何系统性地监控和预防Pod问题？

**答案**：

**1. 监控指标**：

```yaml
# 使用Prometheus监控
- Pod状态指标: kube_pod_status_phase
- 容器重启次数: kube_pod_container_status_restarts_total
- 资源使用: container_memory_usage_bytes, container_cpu_usage_seconds_total
- OOM事件: kube_pod_container_status_last_terminated_reason

# 告警规则示例
groups:
- name: pod-alerts
  rules:
  - alert: PodCrashLooping
    expr: rate(kube_pod_container_status_restarts_total[15m]) > 0
    for: 5m
    annotations:
      summary: "Pod {{ $labels.pod }} is crash looping"
      
  - alert: PodNotReady
    expr: kube_pod_status_phase{phase!="Running"} > 0
    for: 10m
    annotations:
      summary: "Pod {{ $labels.pod }} not ready for 10m"
```

**2. 日志聚合**：

```yaml
# 使用ELK/EFK或Loki
# 收集所有容器日志
# 设置日志告警规则

# 示例: Fluentd配置收集错误日志
<match kubernetes.**>
  @type elasticsearch
  host elasticsearch.logging
  port 9200
  logstash_format true
  # 过滤ERROR级别日志
</match>
```

**3. 健康检查最佳实践**：

```yaml
# 使用三种探针
spec:
  containers:
  - name: myapp
    # 启动探针: 避免慢启动容器被过早杀死
    startupProbe:
      httpGet:
        path: /startup
        port: 8080
      failureThreshold: 30
      periodSeconds: 10
    
    # 存活探针: 检测死锁，需要重启
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 10
      timeoutSeconds: 5
      failureThreshold: 3
    
    # 就绪探针: 控制流量，不会重启容器
    readinessProbe:
      httpGet:
        path: /ready
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 5
```

**4. 资源管理**：

```yaml
# 设置合理的资源请求和限制
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "256Mi"
    cpu: "500m"

# 使用ResourceQuota防止资源耗尽
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-quota
spec:
  hard:
    requests.cpu: "10"
    requests.memory: "20Gi"
    limits.cpu: "20"
    limits.memory: "40Gi"
    pods: "20"

# 使用LimitRange设置默认值
apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
spec:
  limits:
  - default:
      cpu: 500m
      memory: 512Mi
    defaultRequest:
      cpu: 100m
      memory: 128Mi
    type: Container
```

**5. 自动化运维**：

```bash
# 使用Operator模式
# - 自动修复常见问题
# - 自动扩缩容
# - 自动备份和恢复

# 使用准入控制器(Admission Controllers)
# - 验证Pod配置
# - 自动注入sidecar
# - 强制安全策略

# 示例: OPA Gatekeeper策略
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sRequiredLabels
metadata:
  name: must-have-labels
spec:
  match:
    kinds:
    - apiGroups: [""]
      kinds: ["Pod"]
  parameters:
    labels: ["app", "env"]
```

**6. 备份和恢复**：

```bash
# 使用Velero备份和恢复Kubernetes资源
# 安装Velero
velero install --provider aws --bucket velero-backups --secret-file ./credentials-velero

# 备份特定命名空间
velero backup create my-namespace-backup --include-namespaces myapp

# 备份整个集群
velero backup create full-cluster-backup

# 查看备份
velero backup get

# 恢复备份
velero restore create --from-backup my-namespace-backup

# 备份PVC数据
velero backup create pvc-backup --include-resources persistentvolumeclaims,persistentvolumes
```

**7. 灾难恢复演练**：

```yaml
# 定期进行灾难恢复演练
# 验证备份的完整性和可恢复性
# 测试恢复时间目标(RTO)和恢复点目标(RPO)

# 使用Chaos Engineering测试系统韧性
# 使用Litmus或Chaos Mesh进行故障注入
apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  name: pod-delete-chaos
  namespace: myapp
spec:
  annotationCheck: 'true'
  engineState: 'active'
  appinfo:
    appns: 'myapp'
    applabel: 'app=myapp'
    appkind: 'deployment'
  chaosServiceAccount: pod-delete-sa
  experiments:
  - name: pod-delete
    spec:
      components:
        env:
        - name: TOTAL_CHAOS_DURATION
          value: '60'
        - name: CHAOS_INTERVAL
          value: '10'
        - name: FORCE
          value: 'false'
```

**8. 定期备份etcd**：

```bash
# 备份etcd集群数据
ETCDCTL_API=3 etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  snapshot save /etc/kubernetes/etcd-snapshot.db

# 恢复etcd数据
ETCDCTL_API=3 etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  snapshot restore /etc/kubernetes/etcd-snapshot.db \
  --data-dir=/var/lib/etcd-restore \
  --initial-cluster-token etcd-cluster-1 \
  --initial-advertise-peer-urls https://127.0.0.1:2380
```

**9. 版本控制YAML配置**：

```bash
# 使用Git版本控制所有Kubernetes配置
git init
git add deployment.yaml service.yaml
git commit -m "Initial commit of Kubernetes configurations"

# 使用Helm管理应用配置
helm create myapp
helm install myapp ./myapp

# 使用Kustomize管理环境差异
kustomize build overlays/production | kubectl apply -f -
```

## 关键点总结

**排查思路**：
1. 查状态 → 看事件 → 查日志 → 进容器
2. 从外到内：集群 → 节点 → Pod → 容器
3. 从简到繁：基本信息 → 详细配置 → 深入调试

**常用命令**：
```bash
kubectl get pod <pod> -o wide          # 基本信息
kubectl describe pod <pod>             # 详细信息
kubectl logs <pod> [--previous]        # 查看日志
kubectl exec -it <pod> -- sh           # 进入容器
kubectl get events --sort-by=.lastTimestamp  # 查看事件
```

**重点关注**：
- Pod Status和Events（最重要）
- 容器日志和退出码
- 资源使用和限制
- 调度相关配置（nodeSelector, taints, affinity）
- 网络和存储配置

**预防措施**：
- 设置合理的资源requests和limits
- 配置正确的健康检查探针
- 使用监控和告警系统
- 做好日志收集和分析
- 建立标准化的部署流程
- 定期进行备份和灾难恢复演练

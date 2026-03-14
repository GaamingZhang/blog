---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Deployment
---

# Deployment常用参数详解

## Deployment的核心作用

Deployment是Kubernetes中最常用的工作负载控制器，它管理Pod的副本数量和更新策略。理解Deployment的参数配置，是管理Kubernetes应用的基础。

一个完整的Deployment配置包含多个层级的参数：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:     # 元数据
spec:         # 规格配置
  replicas:   # 副本数
  selector:   # 选择器
  strategy:   # 更新策略
  template:   # Pod模板
```

## Metadata参数

### name和namespace

```yaml
metadata:
  name: nginx-deployment
  namespace: default
```

- **name**：Deployment名称，在同一命名空间内唯一
- **namespace**：命名空间，默认为default

### labels和annotations

```yaml
metadata:
  labels:
    app: nginx
    environment: production
  annotations:
    description: "Nginx web server"
    owner: "team-frontend"
```

- **labels**：用于标识和选择资源
- **annotations**：存储非标识性的元数据

## Spec核心参数

### replicas（副本数）

```yaml
spec:
  replicas: 3
```

指定运行的Pod副本数量。默认值为1。

**注意事项**：
- 如果节点资源不足，Pod可能无法全部调度
- 建议生产环境至少2个副本实现高可用
- 配合HPA可以实现自动扩缩

### selector（选择器）

```yaml
spec:
  selector:
    matchLabels:
      app: nginx
    matchExpressions:
    - key: environment
      operator: In
      values:
      - production
      - staging
```

**matchLabels**：精确匹配标签

**matchExpressions**：表达式匹配
- `In`：标签值在指定列表中
- `NotIn`：标签值不在指定列表中
- `Exists`：存在该标签
- `DoesNotExist`：不存在该标签

**重要规则**：selector必须匹配template.labels

```yaml
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
```

### strategy（更新策略）

```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
```

**type**：更新策略类型
- `RollingUpdate`（默认）：滚动更新
- `Recreate`：先删除所有旧Pod，再创建新Pod

**rollingUpdate参数**：

| 参数 | 说明 | 默认值 | 推荐值 |
|------|------|--------|--------|
| maxSurge | 最多可以超出目标副本数的Pod数量 | 25% | 1或25% |
| maxUnavailable | 最多可以有多少Pod不可用 | 25% | 0或25% |

**不同场景的配置**：

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 0

strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 25%
    maxUnavailable: 25%

strategy:
  type: Recreate
```

### minReadySeconds

```yaml
spec:
  minReadySeconds: 30
```

Pod就绪后等待的秒数，才被认为可用。这是一个缓冲期，防止有问题的Pod迅速替换所有旧Pod。

**推荐值**：
- 快速启动应用：10-30秒
- 慢启动应用：60-120秒

### revisionHistoryLimit

```yaml
spec:
  revisionHistoryLimit: 10
```

保留的历史ReplicaSet数量，用于回滚。默认值为10。

**影响**：
- 值越大，可回滚的版本越多
- 值越大，etcd存储压力越大
- 建议值：5-10

### progressDeadlineSeconds

```yaml
spec:
  progressDeadlineSeconds: 600
```

部署超时时间。如果在这个时间内没有完成更新，Deployment会被标记为失败。默认值为600秒。

## Pod Template参数

### containers

```yaml
spec:
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.24
        ports:
        - containerPort: 80
```

**name**：容器名称

**image**：镜像地址

```yaml
image: nginx:1.24
image: registry.example.com/nginx:1.24
imagePullPolicy: IfNotPresent
```

**imagePullPolicy**：镜像拉取策略
- `Always`：每次都拉取
- `IfNotPresent`（默认）：本地没有才拉取
- `Never`：从不拉取

### ports

```yaml
ports:
- name: http
  containerPort: 80
  protocol: TCP
- name: https
  containerPort: 443
  protocol: TCP
```

### env和envFrom

```yaml
env:
- name: ENVIRONMENT
  value: "production"
- name: DB_HOST
  valueFrom:
    configMapKeyRef:
      name: app-config
      key: db-host
- name: DB_PASSWORD
  valueFrom:
    secretKeyRef:
      name: app-secret
      key: password

envFrom:
- configMapRef:
    name: app-config
- secretRef:
    name: app-secret
```

### resources

```yaml
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "500m"
    memory: "512Mi"
```

### livenessProbe和readinessProbe

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
  failureThreshold: 3
```

### volumeMounts和volumes

```yaml
volumeMounts:
- name: config
  mountPath: /etc/config
  readOnly: true
- name: data
  mountPath: /data

volumes:
- name: config
  configMap:
    name: app-config
- name: data
  persistentVolumeClaim:
    claimName: app-pvc
```

### nodeSelector

```yaml
nodeSelector:
  disktype: ssd
  zone: east
```

将Pod调度到具有指定标签的节点。

### affinity

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/arch
          operator: In
          values:
          - amd64
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 1
      preference:
        matchExpressions:
        - key: disktype
          operator: In
          values:
          - ssd
  podAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchLabels:
            app: cache
        topologyKey: kubernetes.io/hostname
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchLabels:
            app: nginx
        topologyKey: kubernetes.io/hostname
```

### tolerations

```yaml
tolerations:
- key: "node-role.kubernetes.io/master"
  operator: "Exists"
  effect: "NoSchedule"
- key: "dedicated"
  operator: "Equal"
  value: "gpu"
  effect: "NoSchedule"
```

### serviceAccountName

```yaml
serviceAccountName: app-sa
automountServiceAccountToken: false
```

### securityContext

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
```

### terminationGracePeriodSeconds

```yaml
terminationGracePeriodSeconds: 30
```

Pod终止前的等待时间，让应用优雅关闭。默认值为30秒。

### lifecycle

```yaml
lifecycle:
  postStart:
    exec:
      command:
      - /bin/sh
      - -c
      - "echo 'Pod started' > /tmp/start.log"
  preStop:
    exec:
      command:
      - /bin/sh
      - -c
      - "sleep 5 && nginx -s quit"
```

## 完整配置示例

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: default
  labels:
    app: nginx
    environment: production
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  minReadySeconds: 30
  revisionHistoryLimit: 10
  progressDeadlineSeconds: 600
  template:
    metadata:
      labels:
        app: nginx
        version: v1.24
    spec:
      serviceAccountName: nginx-sa
      terminationGracePeriodSeconds: 60
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
      - name: nginx
        image: nginx:1.24
        imagePullPolicy: IfNotPresent
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: CONFIG_PATH
          value: "/etc/nginx/nginx.conf"
        envFrom:
        - configMapRef:
            name: nginx-config
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "500m"
            memory: "512Mi"
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
          failureThreshold: 3
        volumeMounts:
        - name: config
          mountPath: /etc/nginx
          readOnly: true
        - name: cache
          mountPath: /var/cache/nginx
        - name: tmp
          mountPath: /tmp
        lifecycle:
          preStop:
            exec:
              command:
              - /bin/sh
              - -c
              - "sleep 5 && nginx -s quit"
      volumes:
      - name: config
        configMap:
          name: nginx-config
      - name: cache
        emptyDir: {}
      - name: tmp
        emptyDir: {}
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: nginx
              topologyKey: kubernetes.io/hostname
      tolerations:
      - key: "node.kubernetes.io/not-ready"
        operator: "Exists"
        effect: "NoExecute"
        tolerationSeconds: 300
```

## 参数速查表

### 副本管理

| 参数 | 说明 | 默认值 |
|------|------|--------|
| replicas | 副本数量 | 1 |
| selector | Pod选择器 | 必填 |
| revisionHistoryLimit | 历史版本保留数 | 10 |

### 更新策略

| 参数 | 说明 | 默认值 |
|------|------|--------|
| strategy.type | 更新策略类型 | RollingUpdate |
| strategy.rollingUpdate.maxSurge | 最大超出副本数 | 25% |
| strategy.rollingUpdate.maxUnavailable | 最大不可用副本数 | 25% |
| minReadySeconds | 最小就绪时间 | 0 |
| progressDeadlineSeconds | 部署超时时间 | 600s |

### 容器配置

| 参数 | 说明 |
|------|------|
| image | 镜像地址 |
| imagePullPolicy | 镜像拉取策略 |
| ports | 容器端口 |
| env | 环境变量 |
| resources | 资源限制 |
| livenessProbe | 存活探针 |
| readinessProbe | 就绪探针 |
| volumeMounts | 卷挂载 |

### 调度配置

| 参数 | 说明 |
|------|------|
| nodeSelector | 节点选择器 |
| affinity | 亲和性配置 |
| tolerations | 容忍配置 |
| priorityClassName | 优先级类名 |

## 常见问题

### Q1: 如何实现零停机更新？

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 0
```

配合readinessProbe和preStop钩子。

### Q2: 如何快速回滚？

```bash
kubectl rollout undo deployment/nginx-deployment
kubectl rollout undo deployment/nginx-deployment --to-revision=2
```

### Q3: 如何暂停和恢复更新？

```bash
kubectl rollout pause deployment/nginx-deployment
kubectl rollout resume deployment/nginx-deployment
```

### Q4: 如何查看更新状态？

```bash
kubectl rollout status deployment/nginx-deployment
kubectl rollout history deployment/nginx-deployment
```

## 最佳实践

1. **资源限制**：始终设置requests和limits
2. **健康检查**：配置livenessProbe和readinessProbe
3. **优雅关闭**：配置preStop钩子和terminationGracePeriodSeconds
4. **安全配置**：使用securityContext限制权限
5. **Pod反亲和**：使用podAntiAffinity分散Pod
6. **版本控制**：使用合理的revisionHistoryLimit

## 参考资源

- [Deployment官方文档](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
- [Pod生命周期](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)

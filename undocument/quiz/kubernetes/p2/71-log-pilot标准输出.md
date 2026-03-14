---
date: 2026-03-12
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - 日志
  - log-pilot
---

# Kubernetes 中 log-pilot 如何收集标准输出日志

## 引言：日志收集的痛点与 log-pilot 的诞生

在 Kubernetes 集群的运维实践中,日志收集是一个永恒的话题。传统的日志收集方案往往面临诸多挑战:需要为每个应用配置复杂的日志采集规则、容器生命周期短暂导致日志易丢失、多种日志类型(标准输出和文件日志)需要不同的处理方式、配置变更需要重启采集器等。这些问题不仅增加了运维复杂度,还容易导致日志丢失或重复采集。

阿里云开源的 log-pilot 正是为解决这些痛点而生。作为一款智能的容器日志采集工具,log-pilot 采用声明式配置方式,能够自动发现和采集容器日志,支持标准输出和容器内部文件日志,大大简化了 Kubernetes 环境下的日志收集工作。本文将深入剖析 log-pilot 收集标准输出日志的原理、机制和实践方法。

## 一、log-pilot 架构与核心特性

### 1.1 整体架构设计

log-pilot 采用 DaemonSet 部署模式,在每个 Kubernetes 节点上运行一个 log-pilot Pod,负责收集该节点上所有容器的日志。其架构如下:

```
┌─────────────────────────────────────────────────────────┐
│                    Kubernetes Node                       │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │  Pod A   │  │  Pod B   │  │  Pod C   │              │
│  │ stdout   │  │ stdout   │  │ stdout   │              │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘              │
│       │             │             │                      │
│       └─────────────┼─────────────┘                      │
│                     ▼                                    │
│           ┌──────────────────┐                          │
│           │   log-pilot      │                          │
│           │   (DaemonSet)    │                          │
│           │                  │                          │
│           │  ┌────────────┐  │                          │
│           │  │  Pilot     │  │  监听Docker事件          │
│           │  │  (Go)      │  │  解析日志配置            │
│           │  └────────────┘  │  动态生成配置            │
│           │  ┌────────────┐  │                          │
│           │  │ Filebeat/  │  │  采集日志                │
│           │  │ Fluentd    │  │  发送到后端              │
│           │  └────────────┘  │                          │
│           └────────┬─────────┘                          │
│                    │                                     │
└────────────────────┼─────────────────────────────────────┘
                     │
                     ▼
         ┌──────────────────────┐
         │  Elasticsearch/Kafka │
         │  阿里云日志服务       │
         └──────────────────────┘
```

### 1.2 核心组件解析

log-pilot 包含两个核心组件:

**Pilot 组件(Go 语言实现)**:
- 监听 Docker/Containerd 事件
- 解析容器的日志配置标签
- 动态生成日志采集配置文件
- 管理底层采集器的生命周期

**底层采集器(Filebeat 或 Fluentd)**:
- 根据动态生成的配置采集日志
- 解析日志格式
- 发送日志到后端存储

### 1.3 核心特性

log-pilot 具有以下核心特性:

| 特性 | 说明 | 优势 |
|------|------|------|
| **声明式配置** | 通过容器标签声明日志配置 | 无需修改采集器配置,应用自主声明 |
| **自动发现** | 自动发现新容器并采集日志 | 无需手动配置,动态感知 |
| **单进程模式** | 每个节点一个 log-pilot 进程 | 资源占用低,运维简单 |
| **多日志类型** | 支持 stdout 和文件日志 | 统一方案,无需多套系统 |
| **多后端支持** | 支持 ES、Kafka、阿里云日志服务等 | 灵活选择,适应不同场景 |
| **动态配置** | 配置变更无需重启采集器 | 实时生效,不影响业务 |

## 二、标准输出日志收集机制深度解析

### 2.1 标准输出日志的存储位置

在 Kubernetes 环境中,容器运行时(Docker、containerd 或 CRI-O)会自动捕获应用输出到 stdout 和 stderr 的日志,并将其存储在节点的特定目录:

**Docker 运行时**:
```
/var/lib/docker/containers/<container-id>/<container-id>-json.log
```

**containerd 运行时**:
```
/var/log/pods/<namespace>_<pod-name>_<pod-uid>/<container-name>/<restart-count>.log
```

**Kubernetes 统一接口**:
```
/var/log/containers/<pod-name>_<namespace>_<container-name>-<container-id>.log
```

这些日志文件以 JSON 格式存储,每行一条记录:

```json
{
  "log": "2026-03-12T10:30:00Z INFO Application started\n",
  "stream": "stdout",
  "time": "2026-03-12T10:30:00.123456789Z"
}
```

### 2.2 log-pilot 收集标准输出日志的工作流程

log-pilot 收集标准输出日志的完整流程如下:

```
步骤1: 容器启动
    ↓
步骤2: Pilot 监听到 Docker 事件(容器创建)
    ↓
步骤3: Pilot 解析容器的日志配置标签
    ├── 读取 aliyun.logs.xxx 环境变量
    └── 识别 stdout 配置
    ↓
步骤4: Pilot 动态生成采集配置
    ├── 生成 Filebeat/Fluentd 配置文件
    ├── 配置日志路径:/var/log/containers/<pod>_<ns>_<container>-<id>.log
    └── 配置输出目标(ES/Kafka等)
    ↓
步骤5: 底层采集器重新加载配置
    ├── Filebeat: 自动 reload 配置
    └── Fluentd: 通过信号重载
    ↓
步骤6: 采集器开始采集日志
    ├── 监听日志文件变化
    ├── 解析 JSON 格式
    └── 添加 Kubernetes 元数据
    ↓
步骤7: 发送到后端存储
    └── Elasticsearch/Kafka/阿里云日志服务
```

### 2.3 关键技术原理

#### 2.3.1 容器事件监听机制

log-pilot 通过 Docker API 或 Containerd CRI 监听容器事件:

```go
// 伪代码示例
func (p *Pilot) watchDockerEvents() {
    events, _ := dockerClient.Events(ctx, types.EventsOptions{})
    for event := range events {
        switch event.Action {
        case "start":
            p.onContainerStart(event.Actor.ID)
        case "die":
            p.onContainerStop(event.Actor.ID)
        }
    }
}
```

当容器启动时,Pilot 会:
1. 获取容器的完整信息(包括环境变量、标签)
2. 解析日志配置
3. 生成采集配置

#### 2.3.2 声明式配置解析

log-pilot 通过容器的环境变量或标签来声明日志配置。对于标准输出日志,配置格式为:

```yaml
env:
- name: aliyun.logs.<name>
  value: "stdout"
```

Pilot 解析这个配置后,会生成对应的采集配置:

```yaml
# Filebeat 配置示例
filebeat.inputs:
- type: log
  paths:
    - /var/log/containers/<pod>_<namespace>_<container>-*.log
  fields:
    log_topic: <name>
  json.keys_under_root: true
  json.add_error_key: true
```

#### 2.3.3 动态配置生成

log-pilot 的核心优势在于动态配置生成。当容器启动或停止时,Pilot 会:

1. **收集所有容器的日志配置**:遍历当前节点上所有容器,收集其日志配置
2. **生成全局配置文件**:将所有配置合并为一个配置文件
3. **触发采集器重载**:通知底层采集器重新加载配置

```go
// 配置生成伪代码
func (p *Pilot) generateConfig() {
    containers := p.listContainers()
    config := Config{}
    
    for _, container := range containers {
        logConfigs := p.parseLogConfig(container)
        for name, path := range logConfigs {
            if path == "stdout" {
                config.AddInput(generateStdoutInput(container, name))
            }
        }
    }
    
    config.WriteToFile(p.configPath)
    p.reloadCollector()
}
```

#### 2.3.4 Kubernetes 元数据注入

log-pilot 会自动为日志添加 Kubernetes 元数据,包括:

- Pod 名称
- Namespace
- Container 名称
- Pod UID
- 节点名称
- Pod 标签

这些元数据通过解析容器 ID 和调用 Kubernetes API 获取:

```json
{
  "@timestamp": "2026-03-12T10:30:00.123Z",
  "log": "Application started",
  "stream": "stdout",
  "kubernetes": {
    "pod_name": "nginx-app-12345-abcde",
    "namespace": "production",
    "container_name": "nginx",
    "pod_uid": "12345678-90ab-cdef",
    "node_name": "node-1",
    "labels": {
      "app": "nginx",
      "version": "v1.2.3"
    }
  }
}
```

## 三、log-pilot 配置详解

### 3.1 DaemonSet 部署配置

log-pilot 以 DaemonSet 形式部署在每个节点上:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: log-pilot
  namespace: kube-system
  labels:
    app: log-pilot
spec:
  selector:
    matchLabels:
      app: log-pilot
  template:
    metadata:
      labels:
        app: log-pilot
    spec:
      tolerations:
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      containers:
      - name: log-pilot
        image: registry.cn-hangzhou.aliyuncs.com/acs/log-pilot:0.9-filebeat
        env:
        # 节点名称
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        # 日志输出类型
        - name: LOGGING_OUTPUT
          value: "elasticsearch"
        # Elasticsearch 地址
        - name: ELASTICSEARCH_HOST
          value: "elasticsearch.logging.svc.cluster.local"
        - name: ELASTICSEARCH_PORT
          value: "9200"
        # ES 认证信息(可选)
        - name: ELASTICSEARCH_USER
          value: "elastic"
        - name: ELASTICSEARCH_PASSWORD
          value: "password"
        volumeMounts:
        # Docker socket
        - name: docker-sock
          mountPath: /var/run/docker.sock
        # 宿主机根目录
        - name: host-root
          mountPath: /host
          readOnly: true
        # 日志目录
        - name: var-log
          mountPath: /var/log
          readOnly: true
        # 容器日志目录
        - name: var-lib-docker
          mountPath: /var/lib/docker
          readOnly: true
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 100m
            memory: 200Mi
      volumes:
      - name: docker-sock
        hostPath:
          path: /var/run/docker.sock
      - name: host-root
        hostPath:
          path: /
      - name: var-log
        hostPath:
          path: /var/log
      - name: var-lib-docker
        hostPath:
          path: /var/lib/docker
```

### 3.2 应用日志配置

应用通过环境变量声明日志配置:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-app
  namespace: default
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    env:
    # 收集标准输出日志到 catalina 索引
    - name: aliyun.logs.catalina
      value: "stdout"
    # 收集访问日志文件
    - name: aliyun.logs.access
      value: "/var/log/nginx/access.log"
    volumeMounts:
    - name: nginx-logs
      mountPath: /var/log/nginx
  volumes:
  - name: nginx-logs
    emptyDir: {}
```

**配置说明**:

- `aliyun.logs.<name>`: 日志配置的键名,会作为 Elasticsearch 的索引名
- `value: "stdout"`: 表示收集标准输出日志
- `value: "/path/to/file"`: 表示收集容器内的文件日志

### 3.3 多容器 Pod 的日志配置

对于多容器 Pod,可以为每个容器单独配置:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: multi-container-app
spec:
  containers:
  # 主应用容器
  - name: app
    image: myapp:v1
    env:
    - name: aliyun.logs.app
      value: "stdout"
  
  # Sidecar 容器
  - name: sidecar
    image: sidecar:v1
    env:
    - name: aliyun.logs.sidecar
      value: "stdout"
```

### 3.4 高级配置选项

log-pilot 支持多种高级配置:

```yaml
env:
# 自定义索引名称
- name: aliyun.logs.app
  value: "stdout"
- name: aliyun.logs.app.tags
  value: "app=nginx,env=prod"

# 多行日志配置
- name: aliyun.logs.app.multiline
  value: "pattern:'^[0-9]{4}' negate:true match:after"

# 日志格式解析
- name: aliyun.logs.app.format
  value: "nginx"

# 日志过滤
- name: aliyun.logs.app.exclude
  value: "GET /healthz"
```

## 四、与其他日志收集方案对比

### 4.1 主流方案对比

| 对比维度 | log-pilot | Fluent Bit | Fluentd | Filebeat |
|---------|-----------|------------|---------|----------|
| **配置方式** | 声明式(容器标签) | 静态配置文件 | 静态配置文件 | 静态配置文件 |
| **自动发现** | ✅ 原生支持 | ✅ 需配置 | ✅ 需配置 | ✅ 需配置 |
| **配置变更** | 自动生效 | 需重启 | 需重启 | 需重启 |
| **资源占用** | 低(100-200MB) | 极低(10-20MB) | 中(100-200MB) | 低(20-50MB) |
| **标准输出支持** | ✅ | ✅ | ✅ | ✅ |
| **文件日志支持** | ✅ | ✅ | ✅ | ✅ |
| **Kubernetes 集成** | 优秀 | 优秀 | 优秀 | 良好 |
| **学习曲线** | 平缓 | 平缓 | 陡峭 | 平缓 |
| **社区活跃度** | 中 | 高 | 高 | 高 |
| **维护成本** | 低 | 中 | 中 | 中 |

### 4.2 log-pilot 的优势

**优势一:声明式配置**

传统方案需要在采集器中为每个应用配置日志路径,而 log-pilot 让应用自己声明:

```yaml
# 传统方案:需要在 Fluentd 配置文件中为每个应用配置
<source>
  @type tail
  path /var/log/containers/app1-*.log
  tag app1
</source>
<source>
  @type tail
  path /var/log/containers/app2-*.log
  tag app2
</source>

# log-pilot 方案:应用自己声明
# 应用1
env:
- name: aliyun.logs.app1
  value: "stdout"

# 应用2
env:
- name: aliyun.logs.app2
  value: "stdout"
```

**优势二:动态配置**

当新应用部署时,传统方案需要修改采集器配置并重启,而 log-pilot 自动感知并生效:

```
传统方案:
部署新应用 → 修改采集器配置 → 重启采集器 → 开始采集

log-pilot 方案:
部署新应用(带日志标签) → 自动发现 → 立即开始采集
```

**优势三:统一管理**

log-pilot 同时支持标准输出和文件日志,无需维护两套系统:

```yaml
# 同时收集标准输出和文件日志
env:
- name: aliyun.logs.stdout
  value: "stdout"
- name: aliyun.logs.file
  value: "/var/log/app/*.log"
```

### 4.3 log-pilot 的局限性

尽管 log-pilot 有诸多优势,但也存在一些局限性:

| 局限性 | 说明 | 应对策略 |
|--------|------|----------|
| **社区生态** | 相比 Fluentd 插件较少 | 可扩展开发,或结合其他工具 |
| **性能调优** | 配置选项相对较少 | 通过底层采集器(Filebeat/Fluentd)调优 |
| **复杂场景** | 复杂日志处理能力有限 | 结合 Logstash 进行二次处理 |
| **监控告警** | 内置监控能力有限 | 集成 Prometheus 监控 |

## 五、常见问题与最佳实践

### 5.1 常见问题

#### 问题一:日志丢失怎么办?

**现象**:部分日志未能成功收集到 Elasticsearch。

**排查步骤**:

```bash
# 1. 检查 log-pilot Pod 状态
kubectl get pods -n kube-system -l app=log-pilot

# 2. 查看 log-pilot 日志
kubectl logs -n kube-system <log-pilot-pod>

# 3. 检查容器日志文件是否存在
kubectl exec -n kube-system <log-pilot-pod> -- ls -la /var/log/containers/

# 4. 检查 Elasticsearch 连接
kubectl exec -n kube-system <log-pilot-pod> -- curl -I http://elasticsearch:9200
```

**常见原因及解决方案**:

| 原因 | 解决方案 |
|------|---------|
| Pod 已删除 | 使用集中日志系统及时采集 |
| 磁盘空间满 | 配置日志轮转,清理旧日志 |
| ES 写入失败 | 检查 ES 健康状态,调整索引设置 |
| 网络问题 | 检查网络连通性,配置重试机制 |
| 日志格式错误 | 检查日志格式,配置正确的 parser |

#### 问题二:如何处理多行日志?

**场景**:Java 异常堆栈等多行日志被拆分成多条记录。

**解决方案**:使用 multiline 配置:

```yaml
env:
- name: aliyun.logs.app
  value: "stdout"
- name: aliyun.logs.app.multiline
  value: "pattern:'^[0-9]{4}-[0-9]{2}-[0-9]{2}' negate:true match:after"
```

#### 问题三:如何过滤健康检查日志?

**场景**:Kubernetes 健康检查产生大量无用日志。

**解决方案**:使用 exclude 配置:

```yaml
env:
- name: aliyun.logs.app
  value: "stdout"
- name: aliyun.logs.app.exclude
  value: "GET /healthz,GET /readiness"
```

#### 问题四:如何自定义索引名称?

**场景**:需要根据应用、环境等维度创建不同的索引。

**解决方案**:使用 tags 配置:

```yaml
env:
- name: aliyun.logs.app
  value: "stdout"
- name: aliyun.logs.app.tags
  value: "app=nginx,env=production"
```

生成的索引名称格式:`app-2026.03.12`

#### 问题五:如何处理日志延迟?

**场景**:日志实时性要求高,但存在延迟。

**解决方案**:

1. **优化采集器配置**:
```yaml
# Filebeat 配置
filebeat.config.modules:
  reload.enabled: true
  reload.period: 10s

output.elasticsearch:
  bulk_max_size: 1000
  flush_interval: 1s
```

2. **增加资源限制**:
```yaml
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
```

3. **使用 Kafka 缓冲**:
```yaml
env:
- name: LOGGING_OUTPUT
  value: "kafka"
- name: KAFKA_BROKERS
  value: "kafka:9092"
```

### 5.2 最佳实践

#### 实践一:结构化日志输出

应用应输出结构化的 JSON 日志:

```json
{
  "timestamp": "2026-03-12T10:30:00.123Z",
  "level": "INFO",
  "service": "user-api",
  "trace_id": "abc123def456",
  "user_id": "12345",
  "message": "User login successful",
  "context": {
    "ip": "192.168.1.100",
    "duration_ms": 45
  }
}
```

**优势**:
- 自动解析,无需配置复杂的正则表达式
- 支持字段级别的查询和聚合
- 便于日志分析和可视化

#### 实践二:合理设置资源限制

为 log-pilot 设置合理的资源请求和限制:

```yaml
resources:
  limits:
    cpu: 500m
    memory: 500Mi
  requests:
    cpu: 100m
    memory: 200Mi
```

**建议**:
- 小规模集群(节点数 < 10): CPU 200m, Memory 200Mi
- 中规模集群(节点数 10-50): CPU 500m, Memory 500Mi
- 大规模集群(节点数 > 50): CPU 1000m, Memory 1Gi

#### 实践三:监控 log-pilot 性能

监控关键指标:

```yaml
# Prometheus ServiceMonitor 示例
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: log-pilot
spec:
  selector:
    matchLabels:
      app: log-pilot
  endpoints:
  - port: metrics
    interval: 30s
```

关键指标:
- 日志处理速率(records/s)
- 输出延迟
- 错误率
- 内存使用
- CPU 使用

#### 实践四:配置日志保留策略

在 Elasticsearch 中配置索引生命周期管理:

```json
{
  "policy": {
    "phases": {
      "hot": {
        "min_age": "0ms",
        "actions": {
          "rollover": {
            "max_size": "50GB",
            "max_age": "1d"
          }
        }
      },
      "delete": {
        "min_age": "30d",
        "actions": {
          "delete": {}
        }
      }
    }
  }
}
```

#### 实践五:使用标签和注解

利用 Kubernetes 标签和注解增强日志查询:

```yaml
metadata:
  labels:
    app: user-api
    version: v1.2.3
    environment: production
  annotations:
    logging/level: "info"
    logging/exclude: "false"
```

## 六、生产环境部署建议

### 6.1 高可用部署

在生产环境中,建议:

1. **多副本 Elasticsearch**:确保日志存储高可用
2. **Kafka 缓冲层**:在 log-pilot 和 ES 之间增加 Kafka,防止日志丢失
3. **监控告警**:监控 log-pilot 和 ES 的健康状态
4. **日志采样**:对高频日志进行采样,降低存储成本

### 6.2 安全配置

**RBAC 权限控制**:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: log-pilot
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: log-pilot
rules:
- apiGroups: [""]
  resources: ["pods", "namespaces", "nodes"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: log-pilot
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: log-pilot
subjects:
- kind: ServiceAccount
  name: log-pilot
  namespace: kube-system
```

**敏感信息过滤**:

在应用层面过滤敏感信息:

```python
# Python 示例
import logging
import re

class SensitiveDataFilter(logging.Filter):
    def filter(self, record):
        record.msg = re.sub(r'password=\S+', 'password=***', record.msg)
        record.msg = re.sub(r'token=\S+', 'token=***', record.msg)
        return True

logger = logging.getLogger()
logger.addFilter(SensitiveDataFilter())
```

### 6.3 性能优化

**Filebeat 调优**:

```yaml
# 通过环境变量配置 Filebeat
env:
- name: FILEBEAT_MAX_PROCS
  value: "2"
- name: FILEBEAT_QUEUE_MEM_EVENTS
  value: "4096"
- name: FILEBEAT_OUTPUT_BULK_MAX_SIZE
  value: "1000"
```

**Fluentd 调优**:

```yaml
env:
- name: FLUENTD_BUFFER_CHUNK_LIMIT
  value: "16MB"
- name: FLUENTD_BUFFER_QUEUE_LIMIT
  value: "256"
- name: FLUENTD_FLUSH_INTERVAL
  value: "5s"
```

## 七、面试回答

在面试中回答"Kubernetes 中 log-pilot 如何收集标准输出日志"这个问题时,可以这样组织答案:

"log-pilot 是阿里云开源的容器日志采集工具,它通过 DaemonSet 模式在每个节点上部署一个实例,采用声明式配置方式自动发现和采集容器日志。对于标准输出日志,log-pilot 的工作原理是:首先,Pilot 组件监听 Docker 或 Containerd 的容器事件,当容器启动时,解析容器的环境变量中的日志配置标签,如 `aliun.logs.xxx=stdout`;然后,Pilot 动态生成底层采集器(Filebeat 或 Fluentd)的配置文件,配置日志路径为 `/var/log/containers/` 下的容器日志文件;接着,底层采集器自动重载配置,开始采集日志,并添加 Kubernetes 元数据(Pod 名称、Namespace、标签等);最后,日志被发送到 Elasticsearch、Kafka 或阿里云日志服务等后端存储。log-pilot 的核心优势是声明式配置和动态发现,应用只需通过环境变量声明日志配置,无需修改采集器配置,新应用部署后日志采集立即生效,大大简化了 Kubernetes 环境下的日志收集工作。"

---

## 参考资源

- [log-pilot GitHub 仓库](https://github.com/AliyunContainerService/log-pilot)
- [Kubernetes 日志架构官方文档](https://kubernetes.io/docs/concepts/cluster-administration/logging/)
- [Filebeat 官方文档](https://www.elastic.co/guide/en/beats/filebeat/current/index.html)
- [Fluentd 官方文档](https://www.fluentd.org/)
- [阿里云容器服务日志收集实践](https://help.aliyun.com/document_detail/86552.html)

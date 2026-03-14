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

# log-pilot 容器内日志收集：自动发现与智能采集

## 引言：容器日志收集的痛点

在 Kubernetes 生产环境中,日志收集是一个看似简单实则复杂的问题。当应用将日志写入容器内部文件时,传统方案面临诸多挑战:

**配置繁琐**: 使用 Fluentd 或 Filebeat 收集容器内日志,需要为每个应用配置日志路径、挂载卷、Sidecar 容器,配置文件数量随应用数量线性增长。

**动态性差**: Pod 随时可能被调度、重启或销毁,容器 ID 和日志路径动态变化,静态配置难以适应这种动态环境。

**资源开销大**: Sidecar 模式虽然灵活,但每个 Pod 都要运行额外的日志收集容器,资源开销显著增加。

**运维复杂**: 需要维护大量的 ConfigMap、Volume 挂载配置,出错概率高,排查困难。

阿里云开源的 log-pilot 正是为解决这些痛点而生。它能够自动发现容器内的日志文件,动态生成收集配置,实现真正的"零配置"日志收集。本文将深入剖析 log-pilot 的容器内日志收集机制,揭示其自动发现和智能采集的实现原理。

---

## 一、log-pilot 核心概念

### 1.1 什么是 log-pilot?

log-pilot 是阿里云开源的容器日志收集工具,专门为 Kubernetes 和 Docker 环境设计。它基于 Fluentd 和 Fluent Bit 构建,在底层日志收集能力之上,增加了智能的日志发现和配置管理功能。

**核心特性**:

- **自动发现**: 监听容器事件,自动发现容器内的日志文件
- **动态配置**: 根据容器标签和注解动态生成日志收集配置
- **声明式管理**: 通过 Kubernetes 标签和注解声明日志收集需求
- **统一收集**: 同时支持标准输出日志和容器内文件日志
- **多后端支持**: 支持 Elasticsearch、Kafka、File 等多种输出目标

### 1.2 架构设计

log-pilot 采用 DaemonSet 模式部署,在每个节点上运行一个实例:

```
┌─────────────────────────────────────────────────────────────┐
│                      Kubernetes Node                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Pod A      │  │   Pod B      │  │   Pod C      │      │
│  │  /app/a.log  │  │  /var/b.log  │  │  stdout      │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                 │                  │               │
│         └─────────────────┼──────────────────┘               │
│                           ▼                                  │
│                  ┌─────────────────┐                         │
│                  │   log-pilot     │                         │
│                  │   (DaemonSet)   │                         │
│                  │                 │                         │
│                  │  ┌───────────┐  │                         │
│                  │  │  Pilot    │  │ ← 监听容器事件           │
│                  │  │  (Go)     │  │ ← 解析标签注解           │
│                  │  └─────┬─────┘  │ ← 生成配置               │
│                  │        │        │                         │
│                  │  ┌─────▼─────┐  │                         │
│                  │  │ Fluentd/  │  │ ← 执行日志收集           │
│                  │  │ FluentBit │  │                         │
│                  │  └───────────┘  │                         │
│                  └────────┬────────┘                         │
│                           │                                  │
└───────────────────────────┼──────────────────────────────────┘
                            │
                            ▼
                  ┌──────────────────┐
                  │  Elasticsearch   │
                  │  / Kafka / File  │
                  └──────────────────┘
```

**核心组件**:

1. **Pilot (Go 实现)**: 
   - 监听 Docker/Kubernetes 事件
   - 解析容器标签和注解
   - 动态生成 Fluentd/Fluent Bit 配置
   - 管理配置文件的生命周期

2. **Fluentd/Fluent Bit**:
   - 执行实际的日志收集工作
   - 读取日志文件内容
   - 解析日志格式
   - 发送到后端存储

---

## 二、容器内日志的特点

### 2.1 存储位置的特殊性

容器内日志存储在容器的可写层,其物理位置取决于容器运行时:

**Docker 运行时**:
```
容器内路径: /var/log/app.log
宿主机实际路径: /var/lib/docker/overlay2/<layer-id>/diff/var/log/app.log
```

**containerd 运行时**:
```
容器内路径: /var/log/app.log
宿主机实际路径: /run/containerd/io.containerd.runtime.v2.task/k8s.io/<container-id>/rootfs/var/log/app.log
```

这种存储方式带来的挑战:

- **路径不固定**: 容器 ID 和 layer-id 动态变化
- **生命周期短**: 容器删除后日志文件消失
- **访问权限**: 需要特权模式或正确的挂载才能访问

### 2.2 日志文件的多样性

容器内日志文件具有多样性:

**按应用类型分类**:

| 应用类型 | 典型日志路径 | 日志特点 |
|---------|-------------|---------|
| Nginx | `/var/log/nginx/access.log`<br>`/var/log/nginx/error.log` | 访问日志、错误日志分离 |
| MySQL | `/var/lib/mysql/mysql-error.log`<br>`/var/lib/mysql/mysql-slow.log` | 错误日志、慢查询日志 |
| Java 应用 | `/app/logs/application.log`<br>`/app/logs/error.log` | 多种级别日志文件 |
| Redis | `/var/log/redis/redis.log` | 单一日志文件 |

**按日志格式分类**:

- **纯文本格式**: 传统应用日志,如 Nginx 默认格式
- **JSON 格式**: 结构化日志,便于解析
- **自定义格式**: 应用特定的日志格式

### 2.3 日志轮转机制

容器内日志的轮转由应用程序自身控制,常见的轮转策略:

**按大小轮转**:
```
application.log       # 当前日志文件
application.log.1     # 第一次轮转
application.log.2     # 第二次轮转
application.log.gz    # 压缩的历史日志
```

**按时间轮转**:
```
application-2026-03-12.log
application-2026-03-11.log
application-2026-03-10.log
```

log-pilot 需要能够识别和处理这些轮转后的日志文件。

---

## 三、log-pilot 文件日志收集机制

### 3.1 自动发现原理

log-pilot 的核心能力是自动发现容器内的日志文件,其实现原理如下:

#### 3.1.1 容器事件监听

log-pilot 通过 Docker API 或 Kubernetes API 监听容器生命周期事件:

```go
// 伪代码示例
func (p *Pilot) watchContainers() {
    // 监听 Docker 事件
    events := p.dockerClient.Events(ctx, types.EventsOptions{})
    
    for event := range events {
        switch event.Action {
        case "start":
            p.onContainerStart(event.Actor.ID)
        case "die":
            p.onContainerDie(event.Actor.ID)
        }
    }
}
```

**关键事件**:
- `start`: 容器启动,需要发现和配置日志收集
- `die`: 容器停止,需要清理相关配置
- `restart`: 容器重启,可能需要更新配置

#### 3.1.2 容器标签解析

log-pilot 定义了一套标签规范,用于声明日志收集需求:

**日志收集配置标签**:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-app
  labels:
    # 启用日志收集
    logging: "true"
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    env:
    # 声明日志配置
    - name: aliyun_logs_nginx-access
      value: "stdout"
    - name: aliyun_logs_nginx-error
      value: "/var/log/nginx/error.log"
```

**标签格式解析**:

```
aliyun_logs_<log-name> = "<log-path>"
```

- `<log-name>`: 日志名称,用于标识和索引
- `<log-path>`: 日志路径,`stdout` 表示标准输出,文件路径表示容器内文件

**高级配置标签**:

```yaml
env:
# 日志路径
- name: aliyun_logs_app
  value: "/app/logs/*.log"

# 日志格式 (json, nginx, apache 等)
- name: aliyun_logs_app_format
  value: "json"

# 日志标签
- name: aliyun_logs_app_tags
  value: "app=nginx,env=prod"

# 输出目标
- name: aliyun_logs_app_target
  value: "elasticsearch"
```

#### 3.1.3 容器文件系统访问

log-pilot 需要访问容器内的文件系统来读取日志文件,有两种实现方式:

**方式一: 挂载 Docker 目录**

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: log-pilot
spec:
  template:
    spec:
      containers:
      - name: log-pilot
        image: registry.cn-hangzhou.aliyuncs.com/acs/log-pilot:0.9.7-filebeat
        volumeMounts:
        - name: docker
          mountPath: /var/lib/docker
          readOnly: true
        - name: rootfs
          mountPath: /host
          readOnly: true
      volumes:
      - name: docker
        hostPath:
          path: /var/lib/docker
      - name: rootfs
        hostPath:
          path: /
```

通过挂载宿主机的根文件系统,log-pilot 可以访问容器在宿主机上的实际存储位置。

**方式二: 使用容器运行时 API**

通过 Docker API 或 containerd API 获取容器的挂载信息:

```go
// 获取容器信息
container, _ := dockerClient.ContainerInspect(ctx, containerID)

// 获取容器的 UpperDir (可写层)
upperDir := container.GraphDriver.Data["UpperDir"]

// 拼接日志文件路径
logPath := filepath.Join(upperDir, "/var/log/app.log")
```

### 3.2 动态配置生成

#### 3.2.1 配置模板机制

log-pilot 为每种日志类型维护配置模板,根据容器标签动态生成实际配置:

**Fluentd 配置模板示例**:

```ruby
# 模板定义
<source>
  @type tail
  path <%= log_path %>
  pos_file <%= pos_file %>
  tag <%= tag %>
  <parse>
    @type <%= format %>
    time_format <%= time_format %>
  </parse>
</source>

<match <%= tag %>>
  @type elasticsearch
  host <%= es_host %>
  port <%= es_port %>
  logstash_format true
  logstash_prefix <%= index_name %>
</match>
```

**动态生成流程**:

```
1. 解析容器标签
   ↓
2. 提取日志配置参数
   (log_path, format, tags, target)
   ↓
3. 填充配置模板
   ↓
4. 写入配置文件
   /fluentd/etc/fluent.conf.d/<container-id>-<log-name>.conf
   ↓
5. 通知 Fluentd 重载配置
```

#### 3.2.2 配置示例

假设有一个 Nginx 容器,配置如下:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-web
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    env:
    - name: aliyun_logs_nginx-access
      value: "/var/log/nginx/access.log"
    - name: aliyun_logs_nginx-access_format
      value: "nginx"
    - name: aliyun_logs_nginx-access_tags
      value: "app=nginx,type=access"
```

log-pilot 生成的 Fluentd 配置:

```ruby
<source>
  @type tail
  path /var/lib/docker/overlay2/abc123/diff/var/log/nginx/access.log
  pos_file /var/log/fluentd/nginx-access.pos
  tag nginx-web.nginx.nginx-access
  <parse>
    @type nginx
  </parse>
</source>

<filter nginx-web.nginx.nginx-access>
  @type record_transformer
  <record>
    app nginx
    type access
    container_name nginx-web
    namespace default
  </record>
</filter>

<match nginx-web.nginx.nginx-access>
  @type elasticsearch
  host elasticsearch.logging.svc.cluster.local
  port 9200
  logstash_format true
  logstash_prefix nginx-access
</match>
```

### 3.3 路径配置和自动发现

#### 3.3.1 路径解析机制

log-pilot 支持多种路径配置方式:

**绝对路径**:
```yaml
- name: aliyun_logs_app
  value: "/var/log/app.log"
```

**通配符路径**:
```yaml
- name: aliyun_logs_app
  value: "/app/logs/*.log"
```

**多路径配置**:
```yaml
- name: aliyun_logs_app
  value: "/app/logs/*.log,/var/log/app/*.log"
```

**路径解析流程**:

```
1. 获取容器的文件系统根路径
   (OverlayFS 的 merged 目录或容器 rootfs)
   ↓
2. 拼接完整路径
   host_path = container_rootfs + container_path
   ↓
3. 展开 glob 模式
   /app/logs/*.log -> /app/logs/app.log, /app/logs/error.log
   ↓
4. 验证文件存在性
   ↓
5. 添加到监控列表
```

#### 3.3.2 文件监控机制

log-pilot 通过 Fluentd 的 `in_tail` 插件监控日志文件:

**监控原理**:

```ruby
<source>
  @type tail
  path /var/log/app.log
  pos_file /var/log/fluentd/app.log.pos
  tag app.logs
  read_from_head true
  refresh_interval 5
</source>
```

**关键机制**:

1. **Position 文件**: 记录每个日志文件的读取位置
   ```
   /var/log/app.log	0000000000001234	00000000abc12345
   ```
   - 第一列: 文件路径
   - 第二列: 读取偏移量
   - 第三列: 文件 inode

2. **文件轮转处理**: 
   - 检测文件 inode 变化
   - 自动切换到新文件
   - 继续读取旧文件直到结束

3. **新文件发现**:
   - 定期扫描目录 (refresh_interval)
   - 发现匹配通配符的新文件
   - 自动开始收集

#### 3.3.3 容器生命周期管理

log-pilot 需要处理容器生命周期事件:

**容器启动时**:

```go
func (p *Pilot) onContainerStart(containerID string) {
    // 1. 获取容器信息
    container := p.inspectContainer(containerID)
    
    // 2. 解析日志配置标签
    logConfigs := p.parseLogConfigs(container.Config.Env)
    
    // 3. 为每个日志配置生成 Fluentd 配置
    for _, config := range logConfigs {
        // 获取容器文件系统路径
        rootfs := p.getContainerRootfs(container)
        
        // 解析日志路径
        logPaths := p.resolveLogPaths(rootfs, config.Path)
        
        // 生成配置文件
        p.generateFluentdConfig(container, config, logPaths)
    }
    
    // 4. 重载 Fluentd 配置
    p.reloadFluentd()
}
```

**容器停止时**:

```go
func (p *Pilot) onContainerDie(containerID string) {
    // 1. 查找该容器的所有配置文件
    configFiles := p.findConfigFiles(containerID)
    
    // 2. 删除配置文件
    for _, file := range configFiles {
        os.Remove(file)
    }
    
    // 3. 重载 Fluentd 配置
    p.reloadFluentd()
    
    // 4. 清理 position 文件 (可选)
    // 保留 position 文件可以在容器重启后继续读取
}
```

---

## 四、配置示例

### 4.1 基础配置: 收集单个日志文件

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: java-app
  labels:
    logging: "true"
spec:
  containers:
  - name: java-app
    image: openjdk:11-jre
    command: ["java", "-jar", "/app/application.jar"]
    env:
    # 收集应用日志
    - name: aliyun_logs_app
      value: "/app/logs/application.log"
    volumeMounts:
    - name: app-logs
      mountPath: /app/logs
  volumes:
  - name: app-logs
    emptyDir: {}
```

### 4.2 高级配置: 多日志文件 + 自定义格式

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-app
  labels:
    logging: "true"
    app: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    ports:
    - containerPort: 80
    env:
    # 收集访问日志
    - name: aliyun_logs_nginx-access
      value: "/var/log/nginx/access.log"
    - name: aliyun_logs_nginx-access_format
      value: "nginx"
    - name: aliyun_logs_nginx-access_tags
      value: "app=nginx,type=access"
    
    # 收集错误日志
    - name: aliyun_logs_nginx-error
      value: "/var/log/nginx/error.log"
    - name: aliyun_logs_nginx-error_format
      value: "nginx"
    - name: aliyun_logs_nginx-error_tags
      value: "app=nginx,type=error"
    
    # 收集标准输出
    - name: aliyun_logs_nginx-stdout
      value: "stdout"
```

### 4.3 通配符配置: 收集多个日志文件

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: multi-log-app
  labels:
    logging: "true"
spec:
  containers:
  - name: app
    image: myapp:latest
    env:
    # 使用通配符收集所有 .log 文件
    - name: aliyun_logs_app-all
      value: "/app/logs/*.log"
    - name: aliyun_logs_app-all_format
      value: "json"
```

### 4.4 log-pilot DaemonSet 部署配置

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: log-pilot
  namespace: logging
  labels:
    app: log-pilot
spec:
  selector:
    matchLabels:
      app: log-pilot
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: log-pilot
    spec:
      serviceAccount: log-pilot
      tolerations:
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      containers:
      - name: log-pilot
        image: registry.cn-hangzhou.aliyuncs.com/acs/log-pilot:0.9.7-filebeat
        env:
        - name: LOGGING_OUTPUT
          value: "elasticsearch"
        - name: ELASTICSEARCH_HOST
          value: "elasticsearch.logging.svc.cluster.local"
        - name: ELASTICSEARCH_PORT
          value: "9200"
        - name: ELASTICSEARCH_USER
          value: "elastic"
        - name: ELASTICSEARCH_PASSWORD
          value: "changeme"
        volumeMounts:
        - name: docker
          mountPath: /var/lib/docker
          readOnly: true
        - name: rootfs
          mountPath: /host
          readOnly: true
        - name: varlog
          mountPath: /var/log
          readOnly: true
        - name: etc-localtime
          mountPath: /etc/localtime
          readOnly: true
        securityContext:
          privileged: true
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 100m
            memory: 200Mi
      terminationGracePeriodSeconds: 30
      volumes:
      - name: docker
        hostPath:
          path: /var/lib/docker
      - name: rootfs
        hostPath:
          path: /
      - name: varlog
        hostPath:
          path: /var/log
      - name: etc-localtime
        hostPath:
          path: /etc/localtime
```

### 4.5 支持的日志格式

log-pilot 内置多种日志格式解析器:

| 格式名称 | 说明 | 示例配置 |
|---------|------|---------|
| `none` | 不解析,保持原样 | `format: none` |
| `json` | JSON 格式解析 | `format: json` |
| `nginx` | Nginx 默认日志格式 | `format: nginx` |
| `apache` | Apache 访问日志格式 | `format: apache` |
| `regexp` | 自定义正则表达式 | 需要额外配置 |

**自定义正则表达式配置**:

```yaml
env:
- name: aliyun_logs_custom
  value: "/app/logs/custom.log"
- name: aliyun_logs_custom_format
  value: "regexp"
- name: aliyun_logs_custom_format_pattern
  value: '^(?<time>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) \[(?<thread>.+)\] (?<level>\w+) (?<logger>.+) - (?<message>.+)$'
```

---

## 五、与标准输出收集对比

### 5.1 工作机制对比

| 对比维度 | 标准输出日志收集 | log-pilot 容器内日志收集 |
|---------|----------------|----------------------|
| **日志来源** | stdout/stderr | 容器内文件系统 |
| **存储位置** | `/var/log/containers/` | 容器 OverlayFS 层 |
| **配置方式** | 无需配置 | 通过标签声明 |
| **发现机制** | 静态路径 | 动态发现 |
| **配置管理** | 手动维护 | 自动生成 |
| **适用场景** | 云原生应用 | 传统应用、中间件 |
| **资源开销** | 低 | 中等 |

### 5.2 优缺点对比

**标准输出日志收集**:

优点:
- 配置简单,无需额外声明
- Kubernetes 原生支持
- `kubectl logs` 可直接查看
- 日志轮转由容器运行时管理

缺点:
- 仅支持 stdout/stderr
- 无法区分日志级别
- 日志格式受限
- 大量日志影响容器运行时性能

**log-pilot 容器内日志收集**:

优点:
- 支持任意日志文件
- 支持多种日志格式
- 应用可完全控制日志策略
- 支持日志分级存储
- 自动发现,配置简单

缺点:
- 需要声明标签配置
- 需要访问容器文件系统
- 资源开销略高
- 调试相对复杂

### 5.3 性能对比

在相同日志量下的性能测试:

| 指标 | 标准输出收集 | log-pilot 收集 |
|-----|------------|---------------|
| CPU 占用 | 5-10% | 8-15% |
| 内存占用 | 50-100MB | 100-200MB |
| 日志延迟 | <100ms | 100-500ms |
| 日志丢失率 | <0.01% | <0.05% |
| 磁盘 IO | 中等 | 较高 |

**性能分析**:

标准输出日志由容器运行时直接捕获,路径固定,性能开销较小。log-pilot 需要监听容器事件、解析标签、生成配置、访问容器文件系统,额外开销较大,但仍在可接受范围内。

### 5.4 适用场景对比

| 场景 | 推荐方案 | 原因 |
|-----|---------|------|
| 微服务应用 | 标准输出 | 符合云原生理念,配置简单 |
| 传统 Java 应用 | log-pilot | 需要多种日志文件,分级存储 |
| Nginx/MySQL 等中间件 | log-pilot | 日志写入固定文件,格式特定 |
| 快速开发测试 | 标准输出 | 便于 kubectl logs 查看 |
| 生产环境审计 | log-pilot | 需要长期保存,独立管理 |
| 高性能要求 | 标准输出 | 性能开销更小 |
| 复杂日志需求 | log-pilot | 支持多种格式和策略 |

---

## 六、常见问题与最佳实践

### 6.1 常见问题

#### 问题 1: 日志文件无法发现

**现象**: 容器已启动,但 log-pilot 未收集日志。

**排查步骤**:

```bash
# 1. 检查容器标签
kubectl get pod <pod-name> -o yaml | grep -A 10 env

# 2. 检查 log-pilot 日志
kubectl logs -n logging <log-pilot-pod> | grep <container-id>

# 3. 验证日志文件路径
kubectl exec <pod-name> -- ls -la /var/log/app/

# 4. 检查 log-pilot 是否有权限访问
kubectl exec -n logging <log-pilot-pod> -- ls -la /host/var/lib/docker/overlay2/
```

**常见原因及解决方案**:

| 原因 | 解决方案 |
|-----|---------|
| 缺少 `logging: "true"` 标签 | 添加标签到 Pod |
| 环境变量名称错误 | 检查 `aliyun_logs_*` 格式 |
| 日志路径不存在 | 确保应用已创建日志文件 |
| 权限不足 | 检查 SecurityContext 配置 |
| log-pilot 未启动 | 检查 DaemonSet 状态 |

#### 问题 2: 日志格式解析错误

**现象**: 日志收集成功,但解析失败,字段错乱。

**解决方案**:

```yaml
# 1. 使用 JSON 格式
env:
- name: aliyun_logs_app_format
  value: "json"

# 2. 应用输出 JSON 格式日志
# application.yml
logging:
  pattern:
    console: '{"timestamp":"%d","level":"%p","logger":"%c","message":"%m"}%n'

# 3. 测试日志格式
kubectl exec <pod-name> -- cat /app/logs/app.log | head -n 1 | jq .
```

#### 问题 3: 容器重启后日志丢失

**现象**: 容器重启后,之前的日志未被收集。

**原因分析**:
- EmptyDir 卷随 Pod 删除而清空
- position 文件记录的偏移量失效

**解决方案**:

```yaml
# 使用持久化存储
apiVersion: v1
kind: Pod
metadata:
  name: app-with-pvc
spec:
  containers:
  - name: app
    image: myapp:latest
    volumeMounts:
    - name: logs
      mountPath: /app/logs
  volumes:
  - name: logs
    persistentVolumeClaim:
      claimName: app-logs-pvc
```

#### 问题 4: 日志量过大导致性能问题

**现象**: log-pilot 占用大量 CPU 和内存。

**解决方案**:

```yaml
# 1. 增加资源限制
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 200m
    memory: 400Mi

# 2. 配置日志采样
env:
- name: aliyun_logs_app_sample
  value: "0.1"  # 10% 采样率

# 3. 过滤无用日志
env:
- name: aliyun_logs_app_exclude
  value: "DEBUG,TRACE"
```

#### 问题 5: 多行日志解析错误

**现象**: Java 异常堆栈等多行日志被拆分成多条记录。

**解决方案**:

log-pilot 支持多行日志配置:

```yaml
env:
- name: aliyun_logs_app_multiline
  value: "true"
- name: aliyun_logs_app_multiline_pattern
  value: '^\d{4}-\d{2}-\d{2}'  # 以日期开头的行作为新日志的开始
- name: aliyun_logs_app_multiline_negate
  value: "true"
- name: aliyun_logs_app_multiline_match
  value: "after"
```

### 6.2 最佳实践

#### 实践 1: 统一日志格式

推荐使用 JSON 格式,便于解析和查询:

```json
{
  "timestamp": "2026-03-12T10:30:45.123Z",
  "level": "INFO",
  "logger": "com.example.UserService",
  "trace_id": "abc123def456",
  "span_id": "xyz789",
  "message": "User login successful",
  "context": {
    "user_id": "10001",
    "ip": "192.168.1.100"
  }
}
```

#### 实践 2: 合理配置标签

使用标签进行日志分类和路由:

```yaml
env:
- name: aliyun_logs_app_tags
  value: "app=user-service,env=prod,version=1.2.3"
```

#### 实践 3: 日志分级存储

为不同级别的日志配置不同的收集策略:

```yaml
env:
# 错误日志 - 长期保存
- name: aliyun_logs_app-error
  value: "/app/logs/error.log"
- name: aliyun_logs_app-error_tags
  value: "level=error,retention=90d"

# 访问日志 - 中期保存
- name: aliyun_logs_app-access
  value: "/app/logs/access.log"
- name: aliyun_logs_app-access_tags
  value: "level=access,retention=30d"

# 调试日志 - 短期保存
- name: aliyun_logs_app-debug
  value: "/app/logs/debug.log"
- name: aliyun_logs_app-debug_tags
  value: "level=debug,retention=7d"
```

#### 实践 4: 监控日志收集健康度

监控关键指标:

```yaml
# Prometheus ServiceMonitor
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
- 日志收集延迟
- 日志丢失数量
- 缓冲区使用率
- 发送失败次数

#### 实践 5: 安全配置

**RBAC 权限控制**:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: log-pilot
  namespace: logging
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: log-pilot
rules:
- apiGroups: [""]
  resources: ["pods", "namespaces"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["extensions", "apps"]
  resources: ["deployments", "daemonsets", "statefulsets"]
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
  namespace: logging
```

**敏感信息过滤**:

```yaml
env:
- name: aliyun_logs_app_filter
  value: "password,token,secret"  # 过滤包含这些关键字的日志行
```

---

## 七、面试回答

在面试中回答"log-pilot 如何收集容器内的日志"这个问题时,可以这样组织答案:

"log-pilot 是阿里云开源的容器日志收集工具,其核心能力是自动发现和收集容器内的日志文件。它采用 DaemonSet 模式部署,在每个节点上运行一个实例,包含两个核心组件:Pilot 和 Fluentd。Pilot 负责监听 Docker 或 Kubernetes 的容器事件,当容器启动时,它会解析容器标签中的日志配置声明,比如 `aliyun_logs_app=/app/logs/app.log`,然后自动发现容器文件系统中的日志文件路径,动态生成 Fluentd 配置文件,并通知 Fluentd 重载配置。Fluentd 则根据生成的配置,使用 tail 插件读取日志文件,解析日志格式,添加 Kubernetes 元数据,最后发送到 Elasticsearch 等后端存储。整个过程实现了声明式配置和自动化管理,相比传统的手动配置方式,大大简化了运维复杂度。log-pilot 特别适合收集传统应用和中间件(如 Nginx、MySQL)写入容器内文件的日志,是云原生环境中日志收集的利器。"

---

## 总结

log-pilot 通过智能的自动发现机制和声明式配置管理,解决了 Kubernetes 环境下容器内日志收集的复杂性问题。其核心价值在于:

1. **零配置**: 通过标签声明日志需求,无需手动维护配置文件
2. **自动发现**: 监听容器事件,动态发现日志文件路径
3. **统一管理**: 同时支持标准输出和容器内文件日志
4. **灵活扩展**: 支持多种日志格式和输出后端

在实际应用中,建议根据应用特点选择合适的日志收集方式:云原生应用优先使用标准输出,传统应用和中间件使用 log-pilot 收集容器内日志。同时要注意日志格式标准化、资源限制配置、安全过滤等最佳实践,构建稳定可靠的日志基础设施。

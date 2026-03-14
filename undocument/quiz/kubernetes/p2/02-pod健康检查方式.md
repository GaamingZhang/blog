---
date: 2026-03-12
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - 健康检查
  - 探针
---

# Kubernetes Pod 健康检查方式详解

## 为什么需要健康检查？

在传统的应用部署中，当进程退出或崩溃时，系统通常会自动重启服务。但在容器化环境中，这种简单的进程监控机制并不足够。容器可能处于运行状态，但内部的应用程序可能已经陷入死锁、资源耗尽或无响应状态。这种情况下，容器进程依然存活，但应用实际上已经无法正常提供服务。

Kubernetes 作为一个容器编排平台，需要更精细的健康检查机制来确保应用的高可用性。健康检查机制能够：

- **自动故障恢复**：检测到应用异常时自动重启容器，无需人工干预
- **流量管理**：确保只有健康的 Pod 接收请求，避免将流量路由到异常实例
- **滚动更新保障**：在发布过程中确保新版本 Pod 正常启动后才终止旧版本
- **服务可用性提升**：通过主动监控和自动恢复，大幅提高服务的整体可用性

Kubernetes 提供了两种核心的健康检查机制：**Liveness Probe（存活探针）** 和 **Readiness Probe（就绪探针）**。这两种探针协同工作，共同保障 Pod 的健康状态和服务质量。

## Liveness Probe（存活探针）

### 概念

Liveness Probe 用于检测容器是否存活。当存活探针检测失败时，Kubernetes 会认为容器已经处于不可恢复的故障状态，并根据重启策略（RestartPolicy）重启容器。这是一种"硬重启"机制，适用于处理应用死锁、资源耗尽等无法自愈的严重故障。

### 工作原理

存活探针的工作流程如下：

1. **周期性探测**：kubelet 按照配置的时间间隔周期性执行探针检查
2. **状态判断**：根据探针返回结果判断容器健康状态
3. **失败处理**：连续失败次数达到阈值（failureThreshold）时，触发容器重启
4. **重启执行**：kubelet 杀死当前容器，并根据 RestartPolicy 创建新容器

存活探针的核心价值在于处理那些进程依然运行但应用逻辑已经失效的场景。例如，Java 应用可能因为内存溢出而进入假死状态，进程依然存在但无法响应任何请求。此时，存活探针能够检测到异常并触发重启。

### 配置方式

存活探针支持三种检测方式，每种方式适用于不同的应用场景：

#### 1. HTTP GET 探测

HTTP GET 是最常用的探测方式，适用于提供 HTTP 服务的应用。kubelet 向指定路径发送 HTTP GET 请求，根据响应状态码判断健康状态。

**工作原理**：
- kubelet 向容器的指定端口和路径发送 HTTP GET 请求
- 如果响应状态码在 200-399 范围内，认为探测成功
- 如果响应状态码不在该范围或请求超时，认为探测失败

**配置示例**：

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
    httpHeaders:
    - name: Custom-Header
      value: health-check
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
  successThreshold: 1
```

**实现细节**：
- HTTP 请求由 kubelet 在节点本地发起，不经过集群网络
- 支持 HTTP 和 HTTPS 协议（通过 scheme 字段配置）
- 可以添加自定义 HTTP 头部，用于区分健康检查请求和正常业务请求
- 健康检查端点应该轻量、快速，避免执行复杂逻辑

#### 2. TCP Socket 探测

TCP Socket 探测适用于非 HTTP 服务，如数据库、Redis、消息队列等。kubelet 尝试与指定端口建立 TCP 连接，根据连接结果判断健康状态。

**工作原理**：
- kubelet 尝试与容器的指定端口建立 TCP 连接
- 如果连接建立成功，立即断开连接并认为探测成功
- 如果连接建立失败或超时，认为探测失败

**配置示例**：

```yaml
livenessProbe:
  tcpSocket:
    port: 3306
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

**实现细节**：
- TCP 探测只验证端口可达性，不验证应用逻辑
- 适用于那些不提供 HTTP 接口但监听 TCP 端口的服务
- 探测成功后会立即断开连接，不会保持长连接
- 对于需要认证的服务（如数据库），TCP 探测只能验证端口监听状态

#### 3. Exec 探测

Exec 探测通过在容器内执行命令来判断健康状态，适用于需要执行复杂检查逻辑的场景。kubelet 在容器内执行指定命令，根据命令退出码判断健康状态。

**工作原理**：
- kubelet 在容器内执行指定的命令
- 如果命令退出码为 0，认为探测成功
- 如果命令退出码非 0，认为探测失败

**配置示例**：

```yaml
livenessProbe:
  exec:
    command:
    - /bin/sh
    - -c
    - pg_isready -h localhost -U postgres
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

**实现细节**：
- 命令在容器内的命名空间中执行，具有容器的文件系统和环境变量
- 命令的超时时间由 timeoutSeconds 控制，超时视为失败
- 命令应该快速返回，避免长时间运行
- 适用于需要检查文件状态、进程状态或执行自定义健康检查脚本的场景

## Readiness Probe（就绪探针）

### 概念

Readiness Probe 用于检测容器是否准备好接收流量。当就绪探针检测失败时，Kubernetes 会将 Pod 从 Service 的 Endpoints 中移除，停止向该 Pod 发送请求，但不会重启容器。这是一种"软隔离"机制，适用于处理应用临时不可用、依赖服务未就绪等可恢复的故障。

### 工作原理

就绪探针的工作流程如下：

1. **周期性探测**：kubelet 按照配置的时间间隔周期性执行探针检查
2. **状态判断**：根据探针返回结果判断容器是否就绪
3. **端点管理**：探测失败时，将 Pod IP 从 Service 的 Endpoints 中移除
4. **恢复处理**：探测成功后，将 Pod IP 重新加入 Service 的 Endpoints

就绪探针的核心价值在于流量控制。应用启动时可能需要加载配置、预热缓存、建立数据库连接等初始化工作。在此期间，应用虽然进程正常，但还无法处理请求。就绪探针确保只有真正准备好的 Pod 才会接收流量。

### 配置方式

就绪探针同样支持三种检测方式，配置方式与存活探针类似：

#### 1. HTTP GET 探测

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
  successThreshold: 1
```

#### 2. TCP Socket 探测

```yaml
readinessProbe:
  tcpSocket:
    port: 6379
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

#### 3. Exec 探测

```yaml
readinessProbe:
  exec:
    command:
    - /bin/sh
    - -c
    - redis-cli ping | grep PONG
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

### 就绪探针的特殊行为

与存活探针不同，就绪探针有以下特殊行为：

1. **初始延迟**：Pod 启动时，默认处于未就绪状态，直到就绪探针首次成功
2. **流量隔离**：探测失败不会重启容器，只是隔离流量
3. **恢复机制**：探测成功后会自动恢复流量，无需重启
4. **滚动更新**：新 Pod 必须通过就绪检查后，才会终止旧 Pod

## 探针参数详解

Kubernetes 为探针提供了丰富的参数配置，用于精细控制探测行为：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| **initialDelaySeconds** | 0 | 容器启动后等待多少秒才开始探测，避免在应用未完全启动时就开始检查 |
| **periodSeconds** | 10 | 探测的间隔时间（秒），决定了健康检查的频率 |
| **timeoutSeconds** | 1 | 探测的超时时间（秒），超时视为探测失败 |
| **failureThreshold** | 3 | 连续探测失败多少次才认为探测失败，提供一定的容错空间 |
| **successThreshold** | 1 | 连续探测成功多少次才认为探测成功（仅对就绪探针有效） |
| **terminationGracePeriodSeconds** | Pod 的 terminationGracePeriodSeconds | 探测失败后等待容器优雅终止的时间 |

### 参数配置策略

**initialDelaySeconds** 的配置需要根据应用的启动时间来设置。对于 Java 等启动较慢的应用，通常需要设置 30-60 秒的初始延迟。配置过短会导致应用在启动过程中就被判定为不健康，配置过长则会延迟故障发现时间。

**periodSeconds** 的配置需要在检测及时性和系统开销之间平衡。对于关键服务，可以设置较短的间隔（如 5-10 秒）；对于非关键服务，可以设置较长的间隔（如 30-60 秒）以减少系统开销。

**failureThreshold** 的配置需要考虑应用的稳定性。对于网络波动较大的环境，可以适当增加阈值（如 5 次），避免因偶发网络问题导致误判。

**successThreshold** 仅对就绪探针有效。对于启动较慢的应用，可以设置为 2-3，确保应用真正稳定后再开始接收流量。

## 两种探针的区别对比

| 对比维度 | Liveness Probe | Readiness Probe |
|----------|----------------|-----------------|
| **核心作用** | 检测容器是否存活，判断是否需要重启 | 检测容器是否就绪，判断是否可以接收流量 |
| **失败后果** | 重启容器 | 从 Service Endpoints 中移除，停止接收流量 |
| **适用场景** | 应用死锁、资源耗尽、不可恢复的故障 | 应用初始化、依赖服务未就绪、临时过载 |
| **恢复方式** | 容器重启，应用重新初始化 | 自动恢复，无需重启 |
| **初始状态** | 默认健康，首次探测前认为容器存活 | 默认未就绪，首次探测成功前不接收流量 |
| **successThreshold** | 固定为 1，一次成功即认为健康 | 可配置，默认为 1 |
| **对 Service 的影响** | 无直接影响 | 直接控制 Pod 是否在 Endpoints 中 |
| **对滚动更新的影响** | 影响容器重启次数 | 决定新 Pod 何时可以替换旧 Pod |

### 协同工作机制

在实际应用中，两种探针通常协同工作：

1. **启动阶段**：Pod 启动后，先等待 initialDelaySeconds，然后开始执行存活探针和就绪探针
2. **初始化阶段**：应用进行初始化，就绪探针持续失败，Pod 不接收流量；存活探针成功，容器不会被重启
3. **就绪阶段**：应用初始化完成，就绪探针成功，Pod 开始接收流量
4. **运行阶段**：两种探针持续运行，存活探针保障容器健康，就绪探针保障服务可用
5. **故障阶段**：
   - 临时故障（如数据库连接超时）：就绪探针失败，流量隔离，等待恢复
   - 严重故障（如死锁）：存活探针失败，容器重启

## 实际配置示例

### Web 应用配置示例

以下是一个典型的 Web 应用配置，包含存活探针和就绪探针：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-application
spec:
  replicas: 3
  selector:
    matchLabels:
      app: web-app
  template:
    metadata:
      labels:
        app: web-app
    spec:
      containers:
      - name: web-app
        image: nginx:1.21
        ports:
        - containerPort: 80
        
        # 存活探针：检测应用是否存活
        livenessProbe:
          httpGet:
            path: /health
            port: 80
          initialDelaySeconds: 30    # 等待应用启动
          periodSeconds: 10          # 每10秒检查一次
          timeoutSeconds: 5          # 超时时间5秒
          failureThreshold: 3        # 连续失败3次才重启
        
        # 就绪探针：检测应用是否准备好接收流量
        readinessProbe:
          httpGet:
            path: /ready
            port: 80
          initialDelaySeconds: 5     # 启动后5秒开始检查
          periodSeconds: 5           # 每5秒检查一次
          timeoutSeconds: 3          # 超时时间3秒
          failureThreshold: 3        # 连续失败3次才移除
          successThreshold: 1        # 成功1次即认为就绪
        
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### 数据库应用配置示例

数据库应用通常使用 TCP Socket 探测：

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  serviceName: mysql
  replicas: 1
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
        image: mysql:8.0
        ports:
        - containerPort: 3306
        env:
        - name: MYSQL_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mysql-secret
              key: password
        
        # 存活探针：检测 MySQL 进程是否存活
        livenessProbe:
          exec:
            command:
            - /bin/sh
            - -c
            - mysqladmin ping -h localhost -u root -p${MYSQL_ROOT_PASSWORD}
          initialDelaySeconds: 60
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        
        # 就绪探针：检测 MySQL 是否准备好接收连接
        readinessProbe:
          exec:
            command:
            - /bin/sh
            - -c
            - mysql -h localhost -u root -p${MYSQL_ROOT_PASSWORD} -e "SELECT 1"
          initialDelaySeconds: 30
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        
        volumeMounts:
        - name: mysql-data
          mountPath: /var/lib/mysql
      
      volumes:
      - name: mysql-data
        persistentVolumeClaim:
          claimName: mysql-pvc
```

### 多层应用健康检查示例

对于复杂应用，可以设计多层次的检查端点：

```yaml
# 应用端点设计
# /health/live  - 存活探针：只检查进程是否存活，轻量级
# /health/ready - 就绪探针：检查依赖服务和应用状态

apiVersion: apps/v1
kind: Deployment
metadata:
  name: complex-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: complex-app
  template:
    metadata:
      labels:
        app: complex-app
    spec:
      containers:
      - name: app
        image: myapp:v1.0
        ports:
        - containerPort: 8080
        
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
          successThreshold: 2    # 连续成功2次才认为就绪
```

## 常见问题和最佳实践

### 常见问题

**Q1: 存活探针和就绪探针应该使用相同的端点吗？**

不建议使用相同端点。存活探针应该检查应用进程是否存活，逻辑应该轻量、快速；就绪探针应该检查应用是否准备好接收流量，可以包含依赖服务检查。使用相同端点可能导致误判，例如依赖服务暂时不可用导致就绪探针失败，如果存活探针也失败会触发不必要的重启。

**Q2: initialDelaySeconds 应该设置多大？**

应该根据应用的实际启动时间设置。可以通过观察应用启动日志，测量从容器启动到应用完全就绪的时间，然后在此基础上增加 10-20% 的缓冲时间。对于 Java 应用通常需要 30-60 秒，对于 Go 应用通常需要 5-15 秒。

**Q3: 探针检查端点需要认证吗？**

通常不需要。探针检查是由 kubelet 在节点本地发起的，不经过集群网络，安全性较高。如果必须添加认证，建议使用简单的 Token 或 HTTP 头部，避免复杂的认证逻辑影响探测性能。

**Q4: 就绪探针失败会导致 Pod 重启吗？**

不会。就绪探针失败只会将 Pod 从 Service 的 Endpoints 中移除，停止接收流量，但不会重启容器。只有存活探针失败才会触发容器重启。

**Q5: 如何处理探针检查导致的性能问题？**

探针检查应该轻量、快速，避免执行复杂逻辑。可以采取以下措施：
- 使用专门的轻量级健康检查端点
- 避免在健康检查中执行数据库查询或外部 API 调用
- 适当增加 periodSeconds 减少检查频率
- 使用缓存机制，避免重复计算

### 最佳实践

1. **分层设计健康检查端点**
   - 存活探针：检查进程存活状态，逻辑简单快速
   - 就绪探针：检查应用就绪状态，包含依赖检查
   - 避免在存活探针中检查外部依赖

2. **合理设置初始延迟**
   - 根据应用启动时间设置 initialDelaySeconds
   - 可以配合启动探针（Startup Probe）处理启动时间不确定的场景
   - 避免设置过短导致启动期误判

3. **配置适当的阈值**
   - failureThreshold 设置为 2-3 次，提供容错空间
   - 对于就绪探针，successThreshold 可以设置为 2，确保稳定后再接收流量
   - 根据网络环境和应用稳定性调整阈值

4. **监控探针状态**
   - 通过 Kubernetes 事件监控探针失败情况
   - 设置告警规则，及时发现频繁重启或未就绪的 Pod
   - 分析探针失败原因，优化应用和探针配置

5. **优雅处理探针检查**
   - 健康检查端点应该快速响应，建议在 1 秒内返回
   - 避免在健康检查中执行耗时操作
   - 使用连接池、缓存等技术优化性能

6. **区分不同类型的应用**
   - 无状态应用：重点配置就绪探针，确保滚动更新平滑
   - 有状态应用：重点配置存活探针，确保数据一致性
   - 批处理任务：可以不配置探针，依赖 Job 控制器管理

7. **测试探针配置**
   - 在测试环境验证探针配置的正确性
   - 模拟各种故障场景，验证探针行为
   - 测试滚动更新过程中探针的作用

## 面试回答

**面试官问：Kubernetes 中 Pod 服务健康检查方式有哪两种？**

**回答**：Kubernetes 提供了两种核心的健康检查机制：Liveness Probe（存活探针）和 Readiness Probe（就绪探针）。

**Liveness Probe** 用于检测容器是否存活。当探测失败时，Kubernetes 会认为容器已经处于不可恢复的故障状态，根据重启策略重启容器。这适用于处理应用死锁、资源耗尽等严重故障。存活探针支持三种检测方式：HTTP GET、TCP Socket 和 Exec，可以根据应用特点选择合适的方式。

**Readiness Probe** 用于检测容器是否准备好接收流量。当探测失败时，Kubernetes 会将 Pod 从 Service 的 Endpoints 中移除，停止向该 Pod 发送请求，但不会重启容器。这适用于处理应用初始化、依赖服务未就绪等临时性故障。就绪探针同样支持 HTTP GET、TCP Socket 和 Exec 三种检测方式。

两者的核心区别在于：存活探针失败会触发容器重启，是一种"硬重启"机制；就绪探针失败只会隔离流量，是一种"软隔离"机制。在实际应用中，两种探针通常协同工作，存活探针保障容器健康，就绪探针保障服务可用，共同确保应用的高可用性和稳定性。

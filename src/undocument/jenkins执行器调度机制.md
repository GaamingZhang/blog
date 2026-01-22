# Jenkins 执行器调度机制详解

## 引言

Jenkins 作为最流行的持续集成/持续部署（CI/CD）工具之一，其执行器调度机制是理解 Jenkins 工作原理的核心知识点。在大规模构建环境中，合理理解和配置执行器调度机制，能够显著提升构建效率和资源利用率。

本文将深入探讨 Jenkins 的执行器调度机制，包括其核心概念、调度策略、配置方法以及优化技巧。

## 核心概念

### 1. 执行器（Executor）

执行器是 Jenkins 中实际执行构建任务的工作单元。每个执行器在同一时间只能执行一个构建任务。

**关键特性：**
- 每个节点可以配置多个执行器
- 执行器数量决定了节点的并发构建能力
- Master 节点和 Agent 节点都可以配置执行器

**推荐配置：**
```
执行器数量 = CPU 核心数 × (1 + IO 等待时间比例)
```

对于 CPU 密集型任务，通常设置为 CPU 核心数；对于 IO 密集型任务，可以适当增加。

### 2. 节点（Node）

节点是 Jenkins 执行构建的机器，包括：
- **Master 节点**：Jenkins 主控节点，负责调度和管理
- **Agent 节点**：专门用于执行构建任务的从节点

### 3. 构建队列（Build Queue）

当所有执行器都在忙碌时，新提交的构建任务会进入构建队列等待执行。

### 4. 标签（Label）

标签用于标识节点的特性，实现任务与节点的精准匹配。例如：
- `linux`、`windows`、`macos`
- `docker`、`kubernetes`
- `high-memory`、`gpu`

## 默认调度机制

### 1. FIFO 队列策略

Jenkins 默认使用**先进先出（First In First Out）**的队列策略：

```
任务提交时间：Task1 -> Task2 -> Task3 -> Task4
执行顺序：     Task1 -> Task2 -> Task3 -> Task4
```

**特点：**
- 按照任务提交的时间顺序排队
- 公平性较好，避免任务饥饿
- 不考虑任务优先级和重要性

### 2. 节点匹配规则

Jenkins 在分配执行器时遵循以下匹配规则：

#### (1) 限定特定节点
```groovy
node('specific-node-name') {
    // 只在名为 specific-node-name 的节点上执行
}
```

#### (2) 使用标签表达式
```groovy
node('linux && docker') {
    // 在同时具有 linux 和 docker 标签的节点上执行
}
```

标签表达式支持：
- `&&`：逻辑与
- `||`：逻辑或
- `!`：逻辑非
- `()`：分组

示例：
```groovy
node('(linux || macos) && !windows && high-memory') {
    // 在高内存的 Linux 或 macOS 节点上执行，排除 Windows
}
```

#### (3) 不限定节点
```groovy
node {
    // 可以在任何可用节点上执行
}
```

### 3. 执行器分配流程

```
1. 任务进入队列
   ↓
2. 检查是否有空闲执行器
   ↓
3. 从队列头部取出任务
   ↓
4. 检查节点匹配规则
   ↓
5. 找到匹配的空闲执行器
   ↓
6. 分配执行器并开始构建
```

**详细过程：**

1. **任务入队**：新的构建任务提交后进入构建队列
2. **轮询检查**：Jenkins 定期检查是否有空闲执行器
3. **节点筛选**：根据任务的节点限制筛选可用节点
4. **标签匹配**：验证节点标签是否满足任务要求
5. **执行器分配**：将任务分配给第一个满足条件的空闲执行器
6. **任务执行**：执行器开始执行构建任务

### 4. 负载分配策略

Jenkins 默认的负载分配策略相对简单：

**策略特点：**
- 不会主动做精细的负载均衡
- 倾向于使用标签匹配的第一个可用节点
- Master 节点与 Agent 节点同等对待（如果任务无特殊限制）

**实际表现：**
```
场景：3 个 Agent 节点，每个 2 个执行器
任务队列：6 个相同的任务

可能的分配结果：
Agent-1: [Task1, Task2]
Agent-2: [Task3, Task4]
Agent-3: [Task5, Task6]

或者：
Agent-1: [Task1, Task2]
Agent-2: [Task3]
Agent-3: [Task4, Task5, Task6]  # 不保证均衡分配
```

## 调度机制的限制

### 1. 无优先级机制

默认情况下，Jenkins 不支持任务优先级：
- 所有任务平等对待
- 紧急任务无法插队
- 长时间运行的任务会阻塞后续任务

### 2. 无抢占机制

正在运行的任务不会被打断：
- 低优先级任务不会被高优先级任务抢占
- 执行器一旦分配就会执行到任务结束
- 无法中途释放执行器给更重要的任务

### 3. 简单的负载均衡

默认调度器不考虑：
- 节点的当前负载（CPU、内存使用率）
- 节点的历史性能表现
- 任务的预估执行时间

## 高级调度配置

### 1. 使用 Priority Sorter Plugin

安装 Priority Sorter 插件后，可以为任务设置优先级：

**安装方式：**
```
系统管理 -> 插件管理 -> 可选插件 -> 搜索 "Priority Sorter"
```

**配置示例：**
```groovy
properties([
    priority(5)  // 设置任务优先级为 5（数字越大优先级越高）
])

pipeline {
    agent any
    stages {
        stage('Build') {
            steps {
                echo 'Building...'
            }
        }
    }
}
```

**全局配置：**
```
系统管理 -> 系统配置 -> Priority Sorter
- 启用优先级策略
- 设置默认优先级
- 配置优先级范围
```

### 2. 使用 Throttle Concurrent Builds Plugin

控制并发构建数量，避免资源耗尽：

**Pipeline 配置示例：**
```groovy
pipeline {
    agent any
    options {
        throttle(['deployment'])  // 使用名为 deployment 的节流类别
    }
    stages {
        stage('Deploy') {
            steps {
                echo 'Deploying...'
            }
        }
    }
}
```

**全局配置：**
```
系统管理 -> 系统配置 -> Throttle Concurrent Builds
- 创建节流类别（例如：deployment）
- 设置最大并发数
- 选择节流范围（全局/每个节点）
```

### 3. 使用 Node Label Parameter Plugin

允许用户在构建时选择执行节点：

**配置示例：**
```groovy
properties([
    parameters([
        nodeParam(
            name: 'TARGET_NODE',
            description: '选择执行节点',
            allowedNodes: ['node1', 'node2', 'node3'],
            defaultValue: 'node1'
        )
    ])
])

pipeline {
    agent {
        label "${params.TARGET_NODE}"
    }
    stages {
        stage('Build') {
            steps {
                echo "Building on ${env.NODE_NAME}"
            }
        }
    }
}
```

### 4. 静默期（Quiet Period）

为任务设置静默期，延迟任务执行：

**用途：**
- 合并短时间内的多次触发
- 给系统预留准备时间
- 避免频繁的构建触发

**配置方式：**
```groovy
properties([
    quietPeriod(60)  // 设置 60 秒的静默期
])
```

或在任务配置中：
```
任务配置 -> 高级项目选项 -> Quiet period（秒）
```

### 5. 自定义队列策略

通过脚本自定义队列排序逻辑：

**示例：基于项目名称排序**
```groovy
import hudson.model.Queue

Queue.getInstance().getSorter().compare = { a, b ->
    // 优先执行名称包含 "hotfix" 的任务
    if (a.task.name.contains("hotfix") && !b.task.name.contains("hotfix")) {
        return -1
    }
    if (!a.task.name.contains("hotfix") && b.task.name.contains("hotfix")) {
        return 1
    }
    return 0
}
```

## 执行器配置最佳实践

### 1. Master 节点配置

**推荐做法：**
- 将 Master 节点的执行器数量设置为 0
- Master 节点仅用于管理和调度
- 所有构建任务在 Agent 节点上执行

**原因：**
- 避免构建任务影响 Jenkins 核心服务
- 提高系统稳定性
- 便于资源隔离和管理

**配置路径：**
```
系统管理 -> 节点管理 -> master -> 配置 -> # of executors = 0
```

### 2. Agent 节点配置

**执行器数量建议：**

| 任务类型 | 推荐执行器数量 | 说明 |
|---------|--------------|------|
| CPU 密集型 | CPU 核心数 | 编译、压缩等 |
| IO 密集型 | CPU 核心数 × 1.5~2 | 文件操作、网络传输 |
| 混合型 | CPU 核心数 × 1.2 | 一般构建任务 |
| Docker 构建 | CPU 核心数 | 避免资源竞争 |

**配置示例：**
```
节点管理 -> 选择节点 -> 配置
- Name: agent-linux-1
- # of executors: 4
- Remote root directory: /home/jenkins
- Labels: linux docker maven
- Usage: Use this node as much as possible
```

### 3. 标签使用规范

**推荐的标签分类：**

```
# 操作系统
linux, windows, macos

# 架构
x86_64, arm64

# 环境
docker, kubernetes, vm

# 工具链
maven, gradle, nodejs, python

# 资源特性
high-memory, gpu, ssd

# 用途
build, test, deploy
```

**标签组合示例：**
```groovy
// 复杂的标签表达式
node('linux && docker && (maven || gradle) && high-memory') {
    // 在高内存、支持 Docker 且安装了 Maven 或 Gradle 的 Linux 节点上执行
}
```

### 4. 合理使用节点限制

**场景 1：需要特定环境**
```groovy
pipeline {
    agent {
        label 'windows && msbuild'
    }
    stages {
        stage('Build') {
            steps {
                bat 'msbuild solution.sln'
            }
        }
    }
}
```

**场景 2：不同阶段使用不同节点**
```groovy
pipeline {
    agent none
    stages {
        stage('Build') {
            agent {
                label 'linux && maven'
            }
            steps {
                sh 'mvn clean package'
            }
        }
        stage('Test') {
            agent {
                label 'linux && docker'
            }
            steps {
                sh 'docker run test-image'
            }
        }
        stage('Deploy') {
            agent {
                label 'deploy-server'
            }
            steps {
                sh './deploy.sh'
            }
        }
    }
}
```

**场景 3：并行执行在不同节点**
```groovy
pipeline {
    agent none
    stages {
        stage('Parallel Tests') {
            parallel {
                stage('Unit Tests') {
                    agent { label 'test-fast' }
                    steps {
                        sh 'mvn test'
                    }
                }
                stage('Integration Tests') {
                    agent { label 'test-slow' }
                    steps {
                        sh 'mvn verify'
                    }
                }
            }
        }
    }
}
```

## 性能优化策略

### 1. 执行器饱和度监控

**监控指标：**
- 执行器利用率
- 队列长度
- 平均等待时间
- 任务执行时间

**Jenkins 内置视图：**
```
系统管理 -> 系统信息
- 查看各节点执行器状态
- 监控构建队列长度
```

**使用 Monitoring Plugin：**
安装 Monitoring 插件后可以查看详细的性能指标和图表。

### 2. 避免执行器浪费

**常见浪费场景：**

**场景 1：等待用户输入**
```groovy
// ❌ 不推荐：占用执行器等待
pipeline {
    agent any
    stages {
        stage('Deploy') {
            steps {
                input message: '确认部署到生产环境？'
                sh './deploy.sh'
            }
        }
    }
}

// ✅ 推荐：input 步骤不占用执行器
pipeline {
    agent none
    stages {
        stage('Confirm') {
            steps {
                input message: '确认部署到生产环境？'
            }
        }
        stage('Deploy') {
            agent any
            steps {
                sh './deploy.sh'
            }
        }
    }
}
```

**场景 2：长时间等待外部资源**
```groovy
// ❌ 不推荐：占用执行器等待
node {
    sh 'trigger-external-job.sh'
    sleep 3600  // 等待 1 小时
    sh 'fetch-result.sh'
}

// ✅ 推荐：使用轮询或回调机制
pipeline {
    agent any
    stages {
        stage('Trigger') {
            steps {
                sh 'trigger-external-job.sh'
            }
        }
    }
    // 外部任务完成后通过 webhook 触发后续构建
}
```

### 3. 优化构建任务

**并行化构建：**
```groovy
pipeline {
    agent any
    stages {
        stage('Parallel Build') {
            parallel {
                stage('Module A') {
                    steps {
                        sh 'mvn -pl module-a clean install'
                    }
                }
                stage('Module B') {
                    steps {
                        sh 'mvn -pl module-b clean install'
                    }
                }
                stage('Module C') {
                    steps {
                        sh 'mvn -pl module-c clean install'
                    }
                }
            }
        }
    }
}
```

**使用缓存减少构建时间：**
```groovy
pipeline {
    agent any
    stages {
        stage('Build') {
            steps {
                // 使用本地 Maven 仓库缓存
                sh 'mvn -Dmaven.repo.local=.m2/repository clean package'
            }
        }
    }
}
```

### 4. 合理设置超时

防止任务无限期占用执行器：

```groovy
pipeline {
    agent any
    options {
        timeout(time: 1, unit: 'HOURS')  // 整个 Pipeline 超时时间
    }
    stages {
        stage('Build') {
            options {
                timeout(time: 30, unit: 'MINUTES')  // 单个 Stage 超时时间
            }
            steps {
                sh 'mvn clean package'
            }
        }
        stage('Test') {
            steps {
                timeout(time: 10, unit: 'MINUTES') {  // 单个步骤超时时间
                    sh 'mvn test'
                }
            }
        }
    }
}
```

## 故障排查

### 1. 任务长时间在队列中

**可能原因：**
- 没有匹配的可用节点
- 所有执行器都在忙碌
- 节点离线或不可用
- 标签表达式错误

**排查步骤：**
```
1. 检查构建队列：首页 -> Build Queue -> Why?
2. 查看节点状态：系统管理 -> 节点管理
3. 验证标签配置：节点配置 -> Labels
4. 检查节点日志：节点 -> 日志
```

**解决方案：**
- 添加更多 Agent 节点
- 增加执行器数量
- 修正标签表达式
- 重启离线节点

### 2. 执行器利用率低

**可能原因：**
- 任务提交频率低
- 执行器数量配置过多
- 任务执行时间太短
- 节点标签过于细分

**优化方案：**
- 调整执行器数量
- 合并相似的节点标签
- 批量触发构建任务

### 3. 某些节点负载过高

**原因分析：**
- 标签配置不均衡
- 某些任务固定在特定节点
- 节点性能差异大

**解决方案：**
```groovy
// 使用更灵活的标签表达式
node('linux') {
    // 可以在任何 Linux 节点执行，而不是固定某一台
}

// 避免硬编码节点名称
// ❌ 不推荐
node('specific-agent-1') { }

// ✅ 推荐
node('linux && high-performance') { }
```

## 监控和日志

### 1. 查看执行器状态

**Web 界面：**
```
首页 -> Build Executor Status
- 查看所有节点的执行器状态
- 查看正在执行的任务
- 查看执行器空闲/忙碌情况
```

**REST API：**
```bash
# 获取所有执行器信息
curl -s http://jenkins-server/computer/api/json?pretty=true

# 获取特定节点信息
curl -s http://jenkins-server/computer/node-name/api/json?pretty=true
```

### 2. 构建队列分析

**查看队列原因：**
```
首页 -> Build Queue -> 点击任务 -> Why?
```

常见的等待原因：
- "Waiting for next available executor"
- "All nodes of label 'xxx' are offline"
- "There are no nodes with the label 'xxx'"
- "Waiting for an executor on node-name"

### 3. 性能指标收集

**使用 Jenkins Metrics Plugin：**
```groovy
// 记录自定义指标
import jenkins.metrics.api.Metrics

Metrics.metricRegistry().counter('custom.build.count').inc()
Metrics.metricRegistry().timer('custom.build.duration').update(duration, TimeUnit.MILLISECONDS)
```

**使用 Prometheus 监控：**
```
安装 Prometheus Metrics Plugin
配置 Prometheus 抓取 Jenkins 指标
创建 Grafana 仪表板可视化
```

## 常见场景解决方案

### 场景 1：构建高峰期任务堆积

**问题：**
每天上午 9-10 点，大量开发人员提交代码，导致构建任务堆积。

**解决方案：**

**方案 1：增加临时执行器**
```groovy
// 使用 Cloud 插件动态扩展节点（如 Docker、Kubernetes）
pipeline {
    agent {
        kubernetes {
            yaml '''
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: maven
    image: maven:3.8-jdk-11
    command: ['cat']
    tty: true
'''
        }
    }
    stages {
        stage('Build') {
            steps {
                container('maven') {
                    sh 'mvn clean package'
                }
            }
        }
    }
}
```

**方案 2：使用优先级插件**
```groovy
// 关键任务设置更高优先级
properties([
    priority(10)  // 紧急修复
])
```

**方案 3：错峰构建**
```groovy
// 非关键任务延迟到低峰期
properties([
    pipelineTriggers([
        cron('H 2 * * *')  // 凌晨 2 点执行
    ])
])
```

### 场景 2：不同团队共享 Jenkins

**问题：**
多个团队共享同一个 Jenkins 实例，需要资源隔离。

**解决方案：**

**方案 1：按团队划分节点**
```
团队 A 节点：agent-team-a-1, agent-team-a-2
标签：team-a

团队 B 节点：agent-team-b-1, agent-team-b-2
标签：team-b
```

**方案 2：使用文件夹级别的节点限制**
```
安装 Folders Plugin
系统管理 -> 配置系统 -> Folder-level node restrictions
为不同文件夹配置不同的节点访问权限
```

**方案 3：配置并发限制**
```groovy
// 团队 A 的任务最多同时运行 5 个
options {
    throttle(['team-a-quota'])
}

// 在全局配置中设置 team-a-quota 类别的最大并发数为 5
```

### 场景 3：资源敏感型任务

**问题：**
某些任务（如数据库迁移、部署）需要独占资源，不能并发执行。

**解决方案：**

**方案 1：使用 Lockable Resources Plugin**
```groovy
pipeline {
    agent any
    stages {
        stage('Deploy') {
            steps {
                lock('production-environment') {
                    sh './deploy-to-production.sh'
                }
            }
        }
    }
}
```

**方案 2：使用里程碑（Milestone）**
```groovy
pipeline {
    agent any
    stages {
        stage('Deploy') {
            steps {
                milestone 1
                sh './deploy.sh'
                milestone 2
            }
        }
    }
}
// 确保同一任务的多次触发按顺序执行，后续触发会取消前面未完成的构建
```

**方案 3：限制并发构建**
```groovy
properties([
    disableConcurrentBuilds()  // 禁止并发构建
])
```

## 总结

Jenkins 的执行器调度机制是一个看似简单但内涵丰富的系统：

**核心要点：**
1. **默认策略**：FIFO 队列 + 标签匹配
2. **配置原则**：Master 不执行构建，Agent 合理分配执行器
3. **优化方向**：避免执行器浪费，提高并发效率
4. **扩展能力**：通过插件实现优先级、限流等高级功能

**最佳实践建议：**
- 根据任务类型合理配置执行器数量
- 使用标签而非固定节点名称
- 监控执行器利用率，及时调整
- 大规模场景考虑使用动态节点（Cloud）
- 关键任务使用资源锁和优先级

理解和掌握 Jenkins 执行器调度机制，是构建高效 CI/CD 流水线的基础。在实际应用中，需要根据团队规模、项目特点、资源状况进行针对性的配置和优化。

---

## 常见问题

### 1. Master 节点的执行器应该设置为多少？

**推荐答案：**

强烈建议将 Master 节点的执行器数量设置为 **0**。

**原因：**
- **稳定性**：Master 节点负责整个 Jenkins 系统的调度和管理，如果在上面运行构建任务，可能导致系统资源不足，影响 Jenkins 核心服务
- **安全性**：构建脚本可能包含恶意代码或错误，在 Master 上执行存在安全风险
- **可维护性**：将构建任务隔离到 Agent 节点，便于资源管理和故障排查

**配置路径：**
```
系统管理 -> 节点管理 -> Built-In Node -> 配置 -> # of executors = 0
```

**例外情况：**
- 非常简单的测试环境
- 只有一台机器的小规模 Jenkins
- 执行非常轻量的管理任务（不推荐）

即使在例外情况下，也建议尽快迁移到 Master-Agent 架构。

### 2. 如何让紧急任务优先执行？

**解决方案：**

**方法 1：安装 Priority Sorter Plugin（推荐）**

```groovy
// 在 Pipeline 中设置优先级
properties([
    priority(100)  // 数字越大优先级越高，默认为 3
])

pipeline {
    agent any
    stages {
        stage('Hotfix') {
            steps {
                echo '紧急修复任务'
            }
        }
    }
}
```

**全局配置：**
```
1. 安装插件：系统管理 -> 插件管理 -> Priority Sorter
2. 配置策略：系统管理 -> 系统配置 -> Priority Sorter Configuration
   - 启用优先级策略
   - 设置默认优先级：3
   - 优先级范围：1-10
```

**方法 2：使用单独的执行器池**

为紧急任务保留专用节点：
```
创建标签为 "emergency" 的专用节点
紧急任务使用：node('emergency') { }
普通任务不使用此标签
```

**方法 3：手动干预**

在 Jenkins Web 界面中：
```
构建队列 -> 右键点击等待中的紧急任务 -> Move to top
```

**方法 4：中止低优先级任务**

如果情况紧急，可以：
```
Build Executor Status -> 选择正在运行的非关键任务 -> 点击 X 停止
```

### 3. 任务一直显示"Waiting for next available executor"怎么办？

**排查步骤：**

**步骤 1：检查是否有可用节点**
```
系统管理 -> 节点管理
- 查看节点是否在线（图标应为绿色）
- 检查节点是否临时离线
```

**步骤 2：验证标签配置**
```
1. 点击队列中的任务 -> Why?
2. 查看提示信息：
   - "There are no nodes with the label 'xxx'" → 标签不存在
   - "All nodes of label 'xxx' are offline" → 节点都离线了
```

**步骤 3：检查执行器状态**
```
首页 -> Build Executor Status
- 查看执行器是否都在忙碌
- 查看是否有执行器卡住（运行时间异常长）
```

**步骤 4：查看任务日志**
```
点击队列中的任务 -> 查看详细信息
可能显示的信息：
- 节点标签不匹配
- 所有执行器都在忙碌
- 等待资源锁
```

**常见解决方案：**

**问题 1：标签拼写错误**
```groovy
// ❌ 错误
node('liunx') {  // 拼写错误
}

// ✅ 正确
node('linux') {
}
```

**问题 2：节点离线**
```
节点管理 -> 选择离线节点 -> Launch agent
或
重启 Agent 服务
```

**问题 3：所有执行器都忙碌**
```
解决方案：
- 等待现有任务完成
- 增加节点执行器数量
- 添加新的 Agent 节点
- 停止非关键任务
```

**问题 4：执行器卡住**
```
Build Executor Status -> 点击异常任务的 X 按钮停止
检查任务日志找出卡住原因
```

### 4. 如何实现不同环境（开发/测试/生产）使用不同的节点？

**最佳实践方案：**

**方案 1：使用环境标签（推荐）**

**节点配置：**
```
开发环境节点：
- 标签：dev, linux, docker

测试环境节点：
- 标签：test, linux, docker

生产环境节点：
- 标签：prod, linux, docker
```

**Pipeline 使用：**
```groovy
pipeline {
    agent none
    
    parameters {
        choice(
            name: 'ENVIRONMENT',
            choices: ['dev', 'test', 'prod'],
            description: '选择部署环境'
        )
    }
    
    stages {
        stage('Build') {
            agent { label 'linux && docker' }
            steps {
                sh 'mvn clean package'
            }
        }
        
        stage('Deploy') {
            agent { label "${params.ENVIRONMENT}" }
            steps {
                script {
                    sh "./deploy.sh ${params.ENVIRONMENT}"
                }
            }
        }
    }
}
```

**方案 2：使用环境变量 + 标签组合**

```groovy
def getDeploymentLabel(env) {
    switch(env) {
        case 'dev':
            return 'dev-deploy-server'
        case 'test':
            return 'test-deploy-server'
        case 'prod':
            return 'prod-deploy-server && approved'
        default:
            return 'dev-deploy-server'
    }
}

pipeline {
    agent none
    
    stages {
        stage('Deploy to Dev') {
            agent { label getDeploymentLabel('dev') }
            when {
                branch 'develop'
            }
            steps {
                sh './deploy.sh dev'
            }
        }
        
        stage('Deploy to Test') {
            agent { label getDeploymentLabel('test') }
            when {
                branch 'release/*'
            }
            steps {
                sh './deploy.sh test'
            }
        }
        
        stage('Deploy to Production') {
            agent { label getDeploymentLabel('prod') }
            when {
                branch 'main'
            }
            steps {
                input message: '确认部署到生产环境？'
                sh './deploy.sh prod'
            }
        }
    }
}
```

**方案 3：使用专用的部署节点 + SSH**

```groovy
pipeline {
    agent { label 'build-server' }
    
    stages {
        stage('Build') {
            steps {
                sh 'mvn clean package'
            }
        }
        
        stage('Deploy') {
            steps {
                script {
                    def targetServer = ''
                    switch(params.ENVIRONMENT) {
                        case 'dev':
                            targetServer = 'dev-server.company.com'
                            break
                        case 'test':
                            targetServer = 'test-server.company.com'
                            break
                        case 'prod':
                            targetServer = 'prod-server.company.com'
                            break
                    }
                    
                    sh """
                        scp target/*.jar user@${targetServer}:/opt/app/
                        ssh user@${targetServer} 'systemctl restart app'
                    """
                }
            }
        }
    }
}
```

**方案 4：使用凭据 + 环境隔离**

```groovy
pipeline {
    agent any
    
    stages {
        stage('Deploy') {
            steps {
                script {
                    def credentialsId = ''
                    def kubeconfig = ''
                    
                    switch(params.ENVIRONMENT) {
                        case 'dev':
                            credentialsId = 'dev-kubeconfig'
                            kubeconfig = '/path/to/dev-kubeconfig'
                            break
                        case 'test':
                            credentialsId = 'test-kubeconfig'
                            kubeconfig = '/path/to/test-kubeconfig'
                            break
                        case 'prod':
                            credentialsId = 'prod-kubeconfig'
                            kubeconfig = '/path/to/prod-kubeconfig'
                            break
                    }
                    
                    withCredentials([file(credentialsId: credentialsId, variable: 'KUBECONFIG')]) {
                        sh 'kubectl apply -f deployment.yaml'
                    }
                }
            }
        }
    }
}
```

**安全建议：**
- 生产环境节点应设置严格的访问控制
- 使用 Jenkins 凭据管理敏感信息
- 生产部署应增加审批流程
- 配置不同环境的节点使用权限（Role-based Access Control）

### 5. 多个任务同时需要部署到同一个服务器，如何避免冲突？

**问题描述：**

当多个 Pipeline 同时部署到同一个服务器时，可能导致：
- 文件覆盖冲突
- 服务重启冲突
- 数据库迁移冲突
- 资源竞争

**解决方案：**

**方案 1：使用 Lockable Resources Plugin（最推荐）**

**安装配置：**
```
1. 安装插件：系统管理 -> 插件管理 -> Lockable Resources
2. 配置资源：系统管理 -> 系统配置 -> Lockable Resources Manager
   - Resource name: production-server
   - Description: 生产服务器部署锁
```

**Pipeline 使用：**
```groovy
pipeline {
    agent any
    
    stages {
        stage('Deploy') {
            steps {
                lock(resource: 'production-server', inversePrecedence: true) {
                    echo '获得生产服务器锁，开始部署'
                    sh './deploy.sh'
                    echo '部署完成，释放锁'
                }
            }
        }
    }
}
```

**高级用法：同时锁定多个资源**
```groovy
lock(label: 'production', quantity: 1) {
    // 从标签为 'production' 的资源池中获取 1 个资源
    sh './deploy.sh'
}
```

**方案 2：禁用并发构建**

```groovy
properties([
    disableConcurrentBuilds()  // 同一个任务不允许并发执行
])

pipeline {
    agent any
    stages {
        stage('Deploy') {
            steps {
                sh './deploy.sh'
            }
        }
    }
}
```

这样可以确保同一个部署任务不会并发执行，后触发的构建会等待前一个完成。

**方案 3：使用 Milestone 确保顺序**

```groovy
pipeline {
    agent any
    
    stages {
        stage('Build') {
            steps {
                milestone 1
                sh 'mvn clean package'
            }
        }
        
        stage('Deploy') {
            steps {
                milestone 2
                lock('production-server') {
                    sh './deploy.sh'
                }
                milestone 3
            }
        }
    }
}
```

Milestone 确保：
- 后触发的构建到达 milestone 时，会取消前面未到达此 milestone 的构建
- 防止旧的构建覆盖新的构建

**方案 4：使用队列和节流**

```groovy
properties([
    throttleJobProperty(
        categories: ['deployment'],
        throttleEnabled: true,
        throttleOption: 'category',
        maxConcurrentPerNode: 0,
        maxConcurrentTotal: 1  // 全局最多 1 个部署任务
    )
])

pipeline {
    agent any
    stages {
        stage('Deploy') {
            steps {
                sh './deploy.sh'
            }
        }
    }
}
```

**方案 5：使用外部锁服务**

对于分布式 Jenkins 或跨系统场景：

```groovy
pipeline {
    agent any
    
    stages {
        stage('Deploy') {
            steps {
                script {
                    // 使用 Redis 或 etcd 实现分布式锁
                    sh '''
                        # 获取锁
                        while ! redis-cli SET deployment_lock 1 NX EX 300; do
                            echo "等待其他部署完成..."
                            sleep 10
                        done
                        
                        # 执行部署
                        ./deploy.sh
                        
                        # 释放锁
                        redis-cli DEL deployment_lock
                    '''
                }
            }
        }
    }
}
```

**最佳实践建议：**

1. **资源命名规范：** 使用清晰的资源名称，如 `prod-web-server-1`、`test-database`
2. **设置超时：** 避免锁被无限期持有
   ```groovy
   lock(resource: 'production-server', inversePrecedence: true) {
       timeout(time: 30, unit: 'MINUTES') {
           sh './deploy.sh'
       }
   }
   ```
3. **记录日志：** 记录锁的获取和释放，便于排查问题
4. **监控告警：** 监控锁的等待时间，及时发现异常
5. **蓝绿部署：** 对于高可用场景，考虑使用蓝绿部署或滚动更新，减少对锁的依赖

**选择建议：**
- **小规模团队**：方案 2（禁用并发）即可
- **中等规模**：方案 1（Lockable Resources）最合适
- **大规模/复杂场景**：方案 5（外部锁服务）+ 蓝绿部署
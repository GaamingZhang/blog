---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - SRE
tag:
  - SRE
  - ClaudeCode
---

# Jenkins/GitLab CI 流水线设计实战

设想这样一个场景：团队已经有了 Jenkins 或 GitLab CI，流水线也跑起来了，但每次新项目接入都要从头复制粘贴那几百行 Groovy 或 YAML。测试阶段串行跑了 20 分钟，构建出来的镜像 Tag 永远叫 `latest`，部署到生产时没人知道这个镜像究竟是什么时候、从哪个提交构建的。数据库迁移直接写在部署脚本里，一旦失败就是人工介入。

这些问题不是工具配置错误，而是**流水线设计上的结构性缺陷**。本文从原理出发，聚焦生产级流水线设计中最核心的几个决策点。

## 一、Jenkins Shared Library：跨项目复用的设计原理

### 为什么需要 Shared Library

在没有 Shared Library 之前，Jenkins 的可复用单位是"复制一个 Jenkinsfile"。当十个项目都需要"构建镜像 + 推送 Harbor + 更新 Git 配置"这套逻辑时，修改一处推送地址就要改十个文件。这种维护成本随项目数线性增长，是不可持续的。

Jenkins Shared Library 的本质是**将 Pipeline 逻辑提升到一个独立的 Git 仓库，以库的形式供所有项目的 Jenkinsfile 引用**。它把"如何做"（构建镜像的步骤）从"做什么"（项目特定的配置）中分离出来。

### 三层目录结构

Shared Library 仓库遵循固定的目录约定，每个目录承担不同职责：

```
jenkins-shared-library/
├── vars/               # 全局步骤（直接在 Pipeline 中调用）
│   ├── buildAndPush.groovy
│   └── deployToK8s.groovy
├── src/                # 工具类（面向对象的 Groovy 代码）
│   └── com/example/
│       └── DockerUtils.groovy
└── resources/          # 静态资源（脚本、配置模板）
    └── templates/
        └── sonar-project.properties
```

**`vars/` 目录**是最常用的扩展点。该目录下的每个 `.groovy` 文件都自动注册为一个全局步骤（Global Variable），文件名即步骤名。这些步骤可以直接在业务项目的 Jenkinsfile 中调用，就像调用内置的 `sh`、`docker` 步骤一样。

**`src/` 目录**存放有状态的工具类。与 `vars/` 的区别在于：`vars/` 中的步骤是无状态的函数调用风格，而 `src/` 支持完整的面向对象模式，适合封装复杂的内部逻辑（如镜像 Tag 计算规则、版本比较工具等）。

**`resources/` 目录**用于存放需要在流水线中读取的静态文件。通过内置的 `libraryResource()` 方法加载，适合存放 SonarQube 配置模板、Dockerfile 模板等。

### 一个完整的 buildAndPush 步骤封装

```groovy
// vars/buildAndPush.groovy
def call(Map config = [:]) {
    def imageName = config.imageName ?: error("imageName is required")
    def registry  = config.registry  ?: "harbor.example.com"
    def tag       = config.tag       ?: generateTag()

    stage("Build Image") {
        sh "docker build -t ${registry}/${imageName}:${tag} ."
    }
    stage("Push Image") {
        withCredentials([usernamePassword(
            credentialsId: 'harbor-credentials',
            usernameVariable: 'HARBOR_USER',
            passwordVariable: 'HARBOR_PASS'
        )]) {
            sh "docker login ${registry} -u ${HARBOR_USER} -p ${HARBOR_PASS}"
            sh "docker push ${registry}/${imageName}:${tag}"
        }
    }
    return tag   // 返回实际使用的 Tag，供后续步骤使用
}

private String generateTag() {
    def sha = sh(script: "git rev-parse --short HEAD", returnStdout: true).trim()
    def branch = env.BRANCH_NAME.replaceAll("/", "-")
    return "${branch}-${sha}"
}
```

业务项目的 Jenkinsfile 只需：

```groovy
@Library('jenkins-shared-library@v1.2.0') _

pipeline {
    agent any
    stages {
        stage('Build & Push') {
            steps {
                script {
                    def tag = buildAndPush(imageName: 'my-service', registry: 'harbor.example.com')
                    env.IMAGE_TAG = tag
                }
            }
        }
    }
}
```

### 版本化策略：稳定性与迭代速度的平衡

Shared Library 的版本引用方式决定了更新时的影响范围：

```
@Library('jenkins-shared-library@develop')    // 始终跟随开发分支（不稳定）
@Library('jenkins-shared-library@v1.2.0')     // 锁定到语义化版本 Tag（推荐生产）
@Library('jenkins-shared-library@main')       // 跟随主干（较稳定，但变更无提前告知）
```

推荐的版本化策略是：**生产项目锁定具体 Tag（如 `@v1.2.0`），测试项目或新功能开发时引用 `@develop` 分支验证**。当 Shared Library 有破坏性变更时，通过升级主版本号（`v2.0.0`）进行通知，给各项目留出迁移窗口。

## 二、并行化设计：从串行到 DAG 的思维转变

### 串行 Pipeline 的时间浪费

大多数团队初期会将流水线设计成完全串行：编译 → 单元测试 → 镜像构建 → 集成测试 → 推送镜像。假设每个阶段各需 5 分钟，总耗时 25 分钟。然而，单元测试和集成测试之间并没有数据依赖——集成测试不需要等单元测试结束才能准备测试环境。

并行化的核心思维是**将 Pipeline 建模为有向无环图（DAG）**，只有存在真实数据依赖的阶段才串行执行，其余尽可能并行。

```
串行模型（总耗时 ~25min）：
编译(5) → 单测(5) → 镜像构建(5) → 集成测试(5) → 推送(5)

DAG 并行模型（总耗时 ~15min）：
                 ┌─ 单测(5min)        ──┐
编译(5min) ──►  ├─ 静态分析(3min)    ──┤──► 推送镜像(2min)
                 └─ 镜像构建(5min) ──►集成测试(5min) ──┘
```

### Jenkins parallel block 的机制

Jenkins 的 `parallel` 块在底层会为每个并行分支申请独立的执行器（Executor）。这意味着**并行化有前提：Jenkins 有足够的可用 Agent 节点**。如果 Jenkins 只有一个 Executor，并行块会退化为串行等待。

```groovy
stage('Parallel Checks') {
    parallel {
        stage('Unit Test') {
            steps { sh 'make test-unit' }
        }
        stage('Static Analysis') {
            steps { sh 'make sonar-scan' }
        }
        stage('Security Scan') {
            steps { sh 'trivy image my-service:${GIT_COMMIT}' }
        }
    }
}
```

Jenkins 的 `stash`/`unstash` 机制用于在并行 Stage 之间传递文件。编译阶段产出的 jar 包可以 stash 后，在多个测试 Stage 的不同 Executor 上 unstash 复用，避免重复编译。

```groovy
stage('Build') {
    steps {
        sh 'mvn package -DskipTests'
        stash name: 'build-artifacts', includes: 'target/*.jar'
    }
}
stage('Test') {
    parallel {
        stage('Unit Test') {
            steps {
                unstash 'build-artifacts'
                sh 'mvn test -Dtest=UnitTest*'
            }
        }
        stage('Integration Test') {
            steps {
                unstash 'build-artifacts'
                sh 'mvn test -Dtest=IntegrationTest*'
            }
        }
    }
}
```

### GitLab CI 的 needs 关键字与 DAG

GitLab CI 通过 `needs` 关键字实现 DAG 模型。默认情况下，GitLab CI 按 `stages` 数组定义的顺序串行执行每个 stage 内的所有 Job。而 `needs` 允许 Job 跳过 stage 顺序，声明自己只依赖某几个特定 Job，形成真正的 DAG。

```yaml
stages: [build, test, package, deploy]

build-app:
  stage: build
  script: mvn package -DskipTests
  artifacts:
    paths: [target/*.jar]

unit-test:
  stage: test
  needs: [build-app]      # 只等 build-app，不等其他 build stage 的 Job
  script: mvn test -Dtest=Unit*

security-scan:
  stage: test
  needs: [build-app]      # 与 unit-test 并行
  script: trivy fs .

push-image:
  stage: package
  needs: [unit-test, security-scan]   # 等两者都完成
  script: docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA
```

**Jenkins `parallel` vs GitLab CI `needs` 的本质区别**：Jenkins parallel 是 Pipeline 内部的并发控制，受限于 Jenkins Agent 数量，并行分支共享同一个 Pipeline 上下文；GitLab CI `needs` 是 Job 级别的依赖声明，每个 Job 运行在独立的 Runner 上，天然隔离，扩展性更好。

### 矩阵构建：多环境测试的标准化方案

当需要在多个 Java 版本或操作系统组合上运行测试时，GitLab CI 的 `parallel:matrix` 可以避免重复定义 Job：

```yaml
test-matrix:
  stage: test
  parallel:
    matrix:
      - JAVA_VERSION: ["11", "17", "21"]
        OS: ["ubuntu", "alpine"]
  image: eclipse-temurin:${JAVA_VERSION}-jdk-${OS}
  script: mvn test
```

这会自动生成 6 个并行 Job（3 × 2），每个 Job 使用不同的环境组合，测试结果在 GitLab UI 中以矩阵形式展示。

## 三、镜像 Tag 策略：制品不可变性的实现

### latest 是反模式

`latest` Tag 的问题在于它**违反了制品不可变性原则**。当你将 `latest` Tag 重新指向新镜像时，原来那个用于验证测试的镜像在生产上下文中已经无法追溯。生产事故的典型场景是：预发布环境测试的是周五下午推送的 `latest`，凌晨自动化脚本部署时拉取的是周六凌晨意外推送的 `latest`，两个镜像版本不同却都叫同一个名字。

不可变 Tag 的含义是：**一旦某个 Tag 的镜像被推送，它所指向的内容永远不变**。任何新构建都必须产生一个新 Tag。

### 四种 Tag 策略的 trade-off

| 策略 | 示例 | 优点 | 缺点 |
|------|------|------|------|
| SemVer | `v1.2.3` | 可读性强，有语义 | 需要人工管理版本号，CI 不易自动化 |
| Git SHA | `abc1234` | 100% 唯一，可追溯 | 可读性差，无法判断新旧 |
| 时间戳 | `20260213-153042` | 易于判断构建时间 | 与代码版本无关联 |
| Branch+SHA | `main-abc1234` | 兼顾来源和唯一性 | Tag 较长 |

生产推荐策略：**正式发布使用 SemVer（`v1.2.3`），日常构建使用 Branch+SHA 组合**。SemVer 用于表达"这是一个经过完整测试、可以发布的版本"，Branch+SHA 用于开发和测试阶段的构建追踪。

### 多环境镜像晋升机制

镜像晋升（Promotion）的核心原则是：**同一个镜像制品从开发环境晋升到生产环境时，不重新构建，只重新打 Tag**。这保证了"测试的就是部署的"。

```
CI 构建阶段：
  构建镜像 → harbor/my-service:main-abc1234

开发环境部署：
  使用 harbor/my-service:main-abc1234

通过验证，晋升至 Staging：
  docker pull harbor/my-service:main-abc1234
  docker tag  harbor/my-service:main-abc1234  harbor/my-service:staging-abc1234
  docker push harbor/my-service:staging-abc1234

通过验收测试，晋升至 Production（打 SemVer Tag）：
  docker tag  harbor/my-service:staging-abc1234  harbor/my-service:v1.2.3
  docker push harbor/my-service:v1.2.3
```

这个晋升过程可以封装在 CI 流水线的 `promote` 阶段，由人工触发（在 GitLab CI 中通过 `when: manual` 实现），配合审批流程。

### Harbor Tag 保留策略

制品仓库如果不做清理策略，存储成本会持续攀升。Harbor 的 Tag 保留策略可以基于规则自动清理历史 Tag：

```
保留规则示例：
- 保留最近 30 天内推送的所有 Tag
- 保留名称匹配 v*.*.* 的所有 SemVer Tag（永久保留正式版本）
- 保留最近 10 个名称包含 main- 前缀的 Tag
- 清理超过 60 天且不匹配上述规则的 Tag
```

关键原则：**正式版本 Tag（SemVer）永不自动清理，开发构建 Tag 按时间窗口轮转**。

## 四、数据库迁移安全机制：Expand-Contract 模式

### 迁移在 Pipeline 中的位置

数据库迁移放在 CI/CD 的哪个位置，决定了回滚的复杂程度。常见的错误做法是**将迁移脚本写在应用启动逻辑里**（如 Spring Boot 集成 Flyway 自动迁移）。这在单实例部署下没问题，但在滚动更新场景中，新旧两个版本的 Pod 会同时存在，新版本 Pod 执行迁移后，旧版本 Pod 可能因表结构变化而崩溃。

推荐的方式是**将迁移提取为独立的 Pre-deploy Job**，在应用 Pod 更新之前完成，并等待迁移 Job 成功后再继续部署流程。在 ArgoCD 中，这对应 PreSync Hook；在 Jenkins 中，这是一个独立的 `Database Migration` Stage。

### Expand-Contract 三阶段模式

这是解决"数据库迁移与应用版本兼容性"问题的标准模式，将一次破坏性变更拆解为三次可独立部署的变更：

```
传统方式（危险）：
  一次性删除旧列 column_old，添加新列 column_new
  → 部署窗口期旧版本应用崩溃

Expand-Contract 方式：

阶段一（Expand，扩展）：
  添加新列 column_new，保留旧列 column_old
  → 新旧两个版本的应用可以同时工作
  → 应用代码开始写入 column_new，同时继续读写 column_old

阶段二（Migrate，迁移）：
  数据迁移 Job：将 column_old 的历史数据复制到 column_new
  → 应用代码切换为只使用 column_new

阶段三（Contract，收缩）：
  删除旧列 column_old
  → 此时所有旧版本应用已下线，删除操作安全
```

每个阶段都是一次独立的部署，可以回滚。这比"一次性变更"的操作窗口缩小到每个阶段的几分钟内。

### Flyway 在 Pipeline 中的集成方式

```groovy
// Jenkins Pipeline 中的数据库迁移 Stage
stage('Database Migration') {
    steps {
        withCredentials([string(credentialsId: 'db-url', variable: 'DB_URL')]) {
            sh """
                flyway -url=${DB_URL} \
                       -locations=filesystem:./db/migration \
                       -validateOnMigrate=true \
                       migrate
            """
        }
    }
    post {
        failure {
            // 迁移失败时发送告警，不自动回滚（Flyway 不支持 DDL 自动回滚）
            slackSend channel: '#deploy-alerts',
                      message: "DB Migration FAILED for ${env.BUILD_TAG}"
            error "Database migration failed, deployment aborted"
        }
    }
}
```

迁移失败时的关键决策：**Flyway 本身不支持 DDL 语句的自动回滚**（DDL 通常是隐式提交的）。失败处理策略应该是：停止部署流程，人工分析失败原因，通过编写补偿迁移脚本来修复，而不是试图自动回滚。这也是 Expand-Contract 模式存在价值的原因之一——每个阶段的操作本身是向前兼容的，失败风险更小。

## 五、质量门禁设计：失败快速原则

### SonarQube Quality Gate 的异步集成

SonarQube 的分析是异步的。CI 调用 `sonar-scanner` 后，扫描结果不会立刻返回，而是由 SonarQube Server 在后台处理（通常需要 30 秒到几分钟）。直接在 CI 中轮询等待会占用 Executor，也不够优雅。

标准集成方式是使用 `waitForQualityGate()` 步骤（Jenkins SonarQube 插件提供），它通过 SonarQube Server 的 Webhook 机制实现：SonarQube 分析完成后主动回调 Jenkins，Jenkins 的 `waitForQualityGate()` 在收到回调前挂起（不占用 Executor 线程），收到通知后恢复并返回 Quality Gate 结果。

```groovy
stage('SonarQube Analysis') {
    steps {
        withSonarQubeEnv('sonarqube-server') {
            sh 'mvn sonar:sonar -Dsonar.projectKey=my-service'
        }
    }
}
stage('Quality Gate') {
    steps {
        // 等待 SonarQube Webhook 回调，超时 5 分钟
        timeout(time: 5, unit: 'MINUTES') {
            waitForQualityGate abortPipeline: true
        }
    }
}
```

### 失败快速原则与 DAG 前置

失败快速（Fail Fast）的含义是：**把最容易失败、执行速度最快的检查放在 DAG 的最前面**。单元测试（1-2 分钟）比集成测试（10 分钟）更应该前置，代码编译比镜像构建更应该前置。

```
差的设计（慢速任务在前，浪费等待时间）：
镜像构建(8min) → 安全扫描(5min) → 单测(2min) → 集成测试(10min)

好的设计（快速失败任务前置）：
单测(2min)    ──┐
静态分析(1min) ──┤── 镜像构建(8min) ─── 集成测试(10min)
代码编译(1min) ──┘
```

如果单测失败，无需等待镜像构建完成，节省了 8 分钟。

### 条件性质量门：分支差异化策略

对 feature 分支和主干分支采用相同严格程度的质量门是不合理的。feature 分支代码还在迭代中，强制要求 80% 覆盖率会降低开发效率；但主干合并时必须严格把关。

GitLab CI 通过 `rules` 实现条件性触发：

```yaml
sonar-strict:
  stage: quality
  script:
    - mvn sonar:sonar -Dsonar.qualitygate.wait=true
                      -Dsonar.coverage.minimum=80
  rules:
    - if: $CI_COMMIT_BRANCH == "main"   # 主干合并时严格检查

sonar-light:
  stage: quality
  script:
    - mvn sonar:sonar -Dsonar.qualitygate.wait=true
                      -Dsonar.coverage.minimum=60
  rules:
    - if: $CI_COMMIT_BRANCH != "main"   # feature 分支宽松策略
      allow_failure: true               # 不阻断 MR Pipeline
```

## 六、多分支策略与触发控制

### GitFlow vs Trunk-Based 对 Pipeline 的影响

GitFlow 有 `feature`、`develop`、`release`、`hotfix`、`main` 五类分支，每类分支需要不同的 Pipeline 行为。这导致 Pipeline 配置复杂，分支数量多时维护成本高。

Trunk-Based Development 只有一条主干，通过 Feature Flag 控制功能可见性。Pipeline 配置简单，只需区分"主干提交"和"短期 feature 分支提交"两种场景。

**对 Pipeline 设计的实质影响**：GitFlow 需要在 Pipeline 中维护大量 `when: 分支名匹配` 的条件逻辑；Trunk-Based 的 Pipeline 配置可以非常简洁，几乎不需要分支条件判断。三年以上的工程实践表明，团队规模超过 10 人后，GitFlow 的分支维护成本会显著上升，而 Trunk-Based 配合完善的测试体系和 Feature Flag 是更可维护的方向。

### MR Pipeline 与主干 Pipeline 的职责分离

MR（Merge Request）Pipeline 和主干 Pipeline 面向不同目标，应该有不同的内容：

```
MR Pipeline 职责（快速反馈，目标 < 10 分钟）：
  - 代码编译
  - 单元测试
  - 静态代码分析
  - 安全依赖扫描
  - 不构建镜像（节省时间和存储）

主干 Pipeline 职责（完整验证，目标 < 30 分钟）：
  - 全量测试（单测 + 集成测试）
  - 构建并推送镜像
  - 更新 Config Repo 的 Dev 环境 Tag
  - 触发 ArgoCD 同步到开发集群
```

GitLab CI 通过 `rules` 的 `$CI_PIPELINE_SOURCE` 和 `$CI_COMMIT_BRANCH` 区分触发场景：

```yaml
build-image:
  stage: build
  script:
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA
  rules:
    # 只在主干提交时构建镜像，MR Pipeline 跳过
    - if: $CI_COMMIT_BRANCH == "main" && $CI_PIPELINE_SOURCE == "push"

unit-test:
  stage: test
  script: mvn test
  rules:
    # MR 和主干提交都运行单元测试
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_COMMIT_BRANCH == "main"
```

## 小结

- **Shared Library** 的本质是将"如何构建"从"构建什么"中分离，`vars/` 目录的全局步骤机制是实现跨项目复用的关键，生产项目应锁定具体版本 Tag
- **并行化** 的正确思维是建立 DAG 模型，Jenkins parallel 受限于 Executor 数量，GitLab CI `needs` 以 Job 为单位独立扩展，两者机制不同
- **`latest` 是反模式**，生产应使用不可变 Tag，正式版本用 SemVer，日常构建用 Branch+SHA；镜像晋升机制保证"测试的就是部署的"
- **数据库迁移** 应提取为独立 Pre-deploy Job，使用 Expand-Contract 三阶段模式将破坏性变更拆解为向前兼容的小步骤
- **失败快速** 原则要求将高频失败的快速任务前置在 DAG 中，SonarQube 通过 Webhook 机制实现异步质量门
- **MR Pipeline** 和主干 Pipeline 职责应分离，前者快速反馈，后者完整验证，通过 `rules` 精细控制

---

## 常见问题

### Q1：什么情况下需要将 Shared Library 拆分为独立仓库，而不是放在主仓库中？

当以下条件满足时，应考虑拆分为独立仓库：首先，多个独立项目需要复用同一套 Pipeline 逻辑，放在单个项目仓库中意味着其他项目只能复制代码而无法复用；其次，Library 的变更节奏与业务代码的变更节奏不同，将它们混在一起会导致两者的提交历史相互干扰；第三，需要对 Library 进行独立的版本化管理，通过 Tag 让各项目按需升级。

反之，如果只有一个项目，或者 Library 内容高度特定于当前项目，拆分带来的管理成本（独立仓库、独立 CI、版本发布流程）会超过其收益，此时保持单仓库更合理。

判断的核心问题是：**这段逻辑是否存在被第二个独立项目复用的需求？** 如果答案是肯定的，就应该拆分。

### Q2：Jenkins Pipeline 与 GitLab CI 在并行执行机制上有何本质区别？

Jenkins `parallel` 块的并行发生在**单个 Pipeline 实例内部**。所有并行分支共享同一个 Pipeline 的上下文（环境变量、工作空间），每个分支需要从 Jenkins Master 分配一个独立的 Executor（Agent 上的执行槽位）。如果 Agent 的 Executor 数量不足，并行分支会排队等待，实际上退化为串行。其优势在于：并行分支之间可以通过 `stash`/`unstash` 共享构建产物，且配置集中，易于理解。

GitLab CI `needs` 关键字的并行是**Job 级别的**。每个 Job 运行在完全独立的 Runner 实例上，天然隔离，Runner 可以水平扩展，理论上并行上限很高。Job 间通过 `artifacts` 机制传递产物（上传至 GitLab Server，下游 Job 下载），有额外的网络开销但隔离性更强。另外，GitLab CI 的 `needs` 允许 Job 跨 stage 依赖，真正实现 DAG，而不是每个 stage 必须等上一个 stage 全部完成。

实践上，Jenkins 更适合需要紧密共享状态的复杂流水线，GitLab CI 更适合松耦合的独立 Job 场景。

### Q3：镜像晋升机制中，如何保证不同环境的 Tag 与镜像内容的对应关系是可审计的？

镜像晋升的审计链依赖两个机制协同工作。第一是**镜像摘要（Digest）不变性**：无论给镜像打多少个 Tag，其底层的 SHA256 Digest 始终不变。即使你在 dev 环境用 `main-abc1234` Tag，在 production 用 `v1.2.3` Tag，Harbor 中这两个 Tag 指向同一个 Digest，可以通过 `docker inspect` 验证。

第二是**Pipeline 中记录晋升元数据**：每次执行晋升操作时，在 Pipeline 日志或专门的部署记录系统（如 Spinnaker、自建的 Deploy DB）中记录：被晋升镜像的 Digest、源 Tag、目标 Tag、晋升时间、执行人、关联的 Pipeline 构建 ID。

实际上，完整的审计链是：`git commit SHA → CI Build ID → Image Digest → Dev Tag → Staging Tag → Production Tag`。任意一个节点都可以追溯到其他节点，这才是真正的制品可追溯性。

### Q4：在蓝绿部署场景下，数据库迁移的时机应该如何安排，才能最小化回滚风险？

蓝绿部署配合数据库迁移最大的挑战是：**蓝绿两套应用实例在切换流量的短暂窗口期内会同时访问数据库**，如果迁移涉及破坏性变更（删除旧列），旧版本（蓝）实例会立刻报错。

标准解法就是 Expand-Contract 模式。以"将 `user_name` 列重命名为 `full_name`"为例，蓝绿场景下的操作序列是：

第一步，在流量还指向蓝环境时，执行 Expand 迁移：添加 `full_name` 列，将蓝环境应用代码更新为同时写入 `user_name` 和 `full_name`（双写兼容）。此时不切换流量。第二步，数据迁移：将历史数据从 `user_name` 复制到 `full_name`。第三步，部署绿环境（新版本），只读写 `full_name`。将流量从蓝切换到绿。此时双写窗口期结束，蓝环境下线。第四步，单独执行 Contract 迁移：删除 `user_name` 列。

这样在任意时刻，至少一个版本的应用与数据库结构是完全兼容的，回滚只需将流量切回蓝环境，无需数据库回滚。

### Q5：当 Pipeline 越来越慢时，如何系统性地分析和优化，而不是靠直觉盲目调整？

优化 Pipeline 速度需要先测量，再优化。第一步是**建立可观测性**：收集每个 Stage 的历史执行时间（Jenkins 的 `Build History` 或 GitLab CI 的 Analytics 功能），绘制时序图，找出耗时最长的 P95 Stage。

第二步是**识别瓶颈类型**。Pipeline 慢通常有三类原因：串行等待（应该并行的 Stage 在串行执行）、重复工作（每个 Job 都在重新下载相同的依赖，可以用 Cache 优化）、资源竞争（并行 Job 在等待 Agent 或 Runner 资源）。

串行等待用 DAG 重构解决；重复工作通过 GitLab CI 的 `cache` 或 Jenkins 的 `stash` 解决——例如将 Maven 的 `~/.m2/repository` 目录纳入 Cache，在多次 Pipeline 运行之间复用；资源竞争需要扩展 Runner/Agent 节点或调整并发限制。

第三步是**关注关键路径（Critical Path）**：DAG 中从起点到终点耗时最长的那条路径决定了总耗时，对关键路径上的 Stage 优化才有实际效果，对非关键路径的优化对总耗时没有贡献。找到关键路径后，优先在关键路径上实施并行化或缓存策略，通常能获得最大的时间收益。

## 参考资源

- [Jenkins 官方文档](https://www.jenkins.io/doc/)
- [GitLab CI/CD 文档](https://docs.gitlab.com/ee/ci/)
- [Jenkins Shared Library](https://www.jenkins.io/doc/book/pipeline/shared-libraries/)

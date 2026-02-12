---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Others
tag:
  - Others
  - ClaudeCode
---

# SRE岗位需求综合分析

## 一、各 JD 汇总的核心技术要求

以下是从 11 家公司 JD 中提炼的技术要求，按出现频次排序：

| 技术领域 | 出现频次 | 代表性要求 |
| -------- | -------- | ---------- |
| Linux / Shell | 11/11 | 系统调优、性能诊断、脚本开发 |
| Kubernetes / Docker | 11/11 | 大规模集群管理、升级、迁移 |
| 监控体系（Prometheus / Grafana） | 9/11 | 告警设计、SLI/SLO 体系建设 |
| CI/CD 流水线 | 10/11 | Jenkins/GitLab CI、GitOps/ArgoCD |
| 云平台（AWS/Azure/阿里云） | 9/11 | 资源管理、IaC（Terraform） |
| Python / Go / Shell 开发 | 10/11 | 自动化工具开发 |
| 日志体系（ELK / EFK） | 8/11 | 日志收集、存储、分析 |
| 链路追踪（分布式可观测性） | 7/11 | Jaeger / OpenTelemetry、全链路追踪 |
| 中间件运维（Redis / Kafka / Nginx） | 8/11 | 性能调优、故障排查 |
| 高可用 / 灾备 / 应急演练 | 8/11 | 容灾方案设计、故障演练 |
| SLI / SLO / SLA 体系 | 6/11 | 服务质量目标定义与度量 |
| IaC（Terraform / Ansible） | 7/11 | 基础设施即代码 |
| Service Mesh（Istio） | 4/11 | 微服务治理、流量管理 |
| AIOps | 4/11（加分） | 异常检测、智能告警、根因分析 |
| 安全合规（RBAC / 密钥管理 / CIS） | 6/11 | 容器安全基线、证书管理 |
| 容量规划 | 5/11 | 性能基线、压测、预测性扩缩 |

---

## 二、博客已覆盖的重点主题

以下是与 SRE 岗位高度相关、博客已有完整内容的方向：

### 容器与编排（覆盖最强）

- Kubernetes（33 篇）：Pod 生命周期、调度机制、HPA/VPA 自动扩缩、RBAC、NetworkPolicy、Secrets 管理、Helm、Operator、集群不停机升级、单体服务迁移实战、日志收集方案、监控与告警
- Docker（25 篇）：容器隔离原理、镜像构建最佳实践、容器网络、容器安全、多架构镜像、生产配置

### 可观测性

- Prometheus 基本概念、告警规则设计、怎么保证告警有效性
- Jaeger 链路追踪基本概念 + 原理（2 篇）
- Kubernetes 监控与告警、业务监控关注点

### 故障诊断与应急响应

- cpu100 排查流程、线上问题如何定位
- Pod 创建失败排查流程
- 502 错误排查
- 线上业务证书更新、服务从证书转向 Azure AD 认证

### Linux / 操作系统（32 篇）

- 性能瓶颈排查命令（top/free/iostat）
- 进程/线程/协程、进程间通信
- 文件系统、内存、零拷贝、上下文切换
- 常用命令（awk/sed/curl/tail）

### 网络（24 篇）

- TCP/IP、HTTP/HTTPS、DNS、QUIC、BGP
- CDN、NAT、负载均衡（LVS、Nginx 调度算法）
- CSRF/XSS 攻击原理

### 中间件

- Redis（8 篇）：集群规划、高可用、缓存穿透/击穿/雪崩
- MySQL（10 篇）：索引、事务 ACID、死锁
- Kafka（3 篇）：基本概念、消息延迟
- Nginx（4 篇）：调度算法、流量架构角色
- Elasticsearch（1 篇）：基本概念

### CI/CD

- CI/CD 理解、CI/CD 工具链
- Ansible 基本概念

---

## 三、博客未覆盖的岗位要求（优先补充方向）

以下是多家 JD 明确要求、但博客目前缺失的内容，建议按优先级补充：

### 🔴 高频必须（多数 JD 明确要求）

| 缺失主题 | 对应 JD 要求 | 建议文章 |
| -------- | ------------ | -------- |
| Grafana 实战 | 9/11 家提及 | Grafana Dashboard 设计、数据源接入、可视化最佳实践 |
| ELK/EFK 全栈 | 8/11 家提及 | Logstash/Fluentd 采集原理、Kibana 分析、日志规范化 |
| Terraform / IaC | 7/11 家提及 | Terraform 核心概念、云资源管理、状态管理 |
| ArgoCD / GitOps | 5/11 家提及（盈米、拓竹、钛动加分） | ArgoCD 工作原理、GitOps 流程设计 |
| SLI / SLO / SLA 体系 | 6/11 家提及 | 如何定义 SLI/SLO、Error Budget 实践 |
| 灾备与应急演练 | 8/11 家提及 | 容灾方案设计、混沌工程（Chaos Engineering）基础 |

### 🟡 加分项（部分 JD 提及，差异化竞争力）

| 缺失主题 | 对应 JD | 建议文章 |
| -------- | ------- | -------- |
| Istio / Service Mesh | 盈米、拓竹（加分） | Istio 架构、流量治理、mTLS |
| OpenTelemetry | 可观测性现代标准 | OTel 标准介绍、与 Jaeger/Prometheus 整合 |
| AIOps | 盈米、九瓴、IMPLUS | 异常检测原理、AutoGen 在运维中的应用 |
| 容量规划方法论 | 盈米、阿里 | 性能基线建立、压测设计、容量预测模型 |
| CIS Benchmarks / 容器安全基线 | 拓竹 | K8s CIS 基线、Trivy 镜像扫描实战 |
| Nacos 服务发现 | 如祺出行明确要求 | Nacos 注册中心原理与运维 |
| Go 语言自动化开发 | 九瓴、IMPLUS | Go 编写运维工具的实践（当前仅有 go协程 1 篇） |
| Post-mortem 方法论 | 盈米、阿里 | 故障复盘流程、根因分析模板 |
| 云成本优化 | 钛动科技 | 云资源成本分析与优化策略 |

### 🟢 有内容但深度不足

| 现有文章 | 不足之处 | 建议补充 |
| -------- | -------- | -------- |
| Prometheus 基本概念 | 无 Grafana 配套、无告警规则实战 | 补充 PromQL 进阶、AlertManager 配置 |
| Elasticsearch 基本概念（1 篇） | 覆盖过于浅显 | 索引设计、查询优化、集群运维 |
| CI/CD 工具链 | 无 Jenkins/GitLab CI 实操 | 补充流水线设计实战案例 |
| Ansible 基本概念 | 无 Playbook 实战 | 补充 Ansible 自动化运维实战 |

---

## 四、改进优先级建议

### 高优先级（影响明显）

1. 补充 Grafana 实战系列（Dashboard 设计、数据源接入）
2. 补充 ELK/EFK 日志体系全栈内容
3. 补充 Terraform / IaC 基础设施即代码
4. 补充 ArgoCD / GitOps 流程设计
5. 补充 SLI / SLO / SLA 体系建设

### 中优先级

- 补充 Istio / Service Mesh 微服务治理
- 补充 OpenTelemetry 可观测性标准
- 补充灾备与应急演练（混沌工程）
- 深化 Prometheus 系列（PromQL 进阶、AlertManager）

### 低优先级

- AIOps 智能运维（加分项）
- 容量规划方法论
- CIS Benchmarks 容器安全基线
- 云成本优化

---

## 五、深度补充内容（针对高频考点）

### 5.1 SLI / SLO / SLA 体系

**核心概念：**
- SLI（Service Level Indicator）：服务质量指标，如可用性、延迟、吞吐量、错误率
- SLO（Service Level Objective）：服务质量目标，如"99.9% 可用性"、"P99 延迟 < 200ms"
- SLA（Service Level Agreement）：服务质量协议，包含未达标的赔偿条款
- Error Budget：错误预算 = 1 - SLO，用于平衡可靠性投入与功能迭代速度

**SLO 制定原则：**
1. 不要从 100% 开始，留出 Error Budget
2. 基于用户真实体验，而非系统内部指标
3. 区分"燃烧速度"（Burn Rate）：正常燃烧 vs 快速燃烧
4. 多窗口告警：长窗口（如 30 天）监控 SLO，短窗口（如 1 小时）触发告警

**高频面试题：**
1. 如何为微服务定义 SLI？举一个实际例子
2. Error Budget 用完了怎么办？如何决策是继续发布还是冻结？
3. 如何设计多窗口告警策略，既快速发现问题又避免误报？
4. SLO 和监控告警的关系是什么？如何用 PromQL 实现 SLO 告警？

---

### 5.2 Grafana 实战

**核心功能：**
- 数据源管理：Prometheus、Loki、Elasticsearch、MySQL 等
- Dashboard 设计：Panel → Row → Dashboard 层级，变量（Variable）实现动态过滤
- 告警规则：Alert Rule → Contact Point → Notification Policy 三层架构
- 权限管理：Organization → Folder → Dashboard 三级权限

**Dashboard 设计最佳实践：**
1. 信息层次：概览（Overview）→ 详情（Detail）→ 排查（Debug）
2. 变量设计：`$datasource`、`$namespace`、`$pod` 等级联过滤
3. 告警面板：单独的 Alert Status Panel，展示当前告警状态
4. 性能优化：避免过多 Panel，使用 `$__rate_interval` 变量

**高频面试题：**
1. 如何设计一个 K8s 集群监控 Dashboard？包含哪些核心指标？
2. Grafana Alerting 和 Prometheus AlertManager 的区别？如何配合使用？
3. 如何实现 Dashboard 的版本控制和复用？（JSON Export/Import、Provisioning）
4. 大规模监控场景下，Grafana 性能瓶颈在哪？如何优化？

---

### 5.3 ELK/EFK 日志体系

**架构对比：**
- ELK：Elasticsearch + Logstash + Kibana
- EFK：Elasticsearch + Fluentd/Fluent Bit + Kibana
- 选择依据：Logstash 功能强但资源消耗大，Fluent Bit 轻量适合边缘采集

**核心原理：**
- Elasticsearch 倒排索引：文档 → 分词 → Term Dictionary → Posting List
- 写入流程：Document → Buffer → Refresh（1s）→ Segment → Merge
- 查询流程：Query → 协调节点 → 分片查询 → 结果聚合
- 集群架构：Master（元数据）、Data（数据）、Coordinating（协调）节点角色分离

**日志规范化：**
- 统一日志格式：timestamp、level、service、trace_id、message
- 结构化日志：JSON 格式，便于 ES 索引和查询
- 日志分级：DEBUG、INFO、WARN、ERROR，生产环境 INFO 起步

**高频面试题：**
1. Elasticsearch 写入性能优化有哪些方法？（bulk、refresh_interval、索引设计）
2. 如何设计日志索引策略？（按天/按量、冷热分离、ILM 生命周期）
3. K8s 场景下如何采集容器日志？（DaemonSet Fluent Bit + Sidecar 模式对比）
4. 日志量爆炸时如何处理？（采样、过滤、分级存储）

---

### 5.4 Terraform 核心概念

**核心原理：**
- 声明式配置：描述期望状态，Terraform 自动计算差异并执行
- 状态管理：terraform.tfstate 记录资源与配置的映射关系
- 执行计划：terraform plan 预览变更，terraform apply 执行变更
- Provider 机制：通过 Provider 与云平台 API 交互

**状态管理最佳实践：**
1. 远程状态：使用 S3/OSS + DynamoDB 存储状态，支持团队协作
2. 状态锁定：防止并发执行导致状态冲突
3. 状态分离：按环境（dev/staging/prod）或模块分离状态文件
4. 敏感数据：使用 Vault 或云平台 Secrets 管理，不要硬编码

**高频面试题：**
1. Terraform 和 Ansible 的本质区别是什么？（声明式 vs 过程式）
2. 如何处理 Terraform 状态漂移？（terraform refresh、terraform import）
3. 多环境管理如何设计？（Workspace vs 目录分离 vs Terragrunt）
4. 如何实现 Terraform 模块的版本控制和复用？

---

### 5.5 ArgoCD / GitOps

**核心原理：**
- GitOps 原则：Git 是唯一事实来源，声明式配置，自动同步，持续协调
- ArgoCD 架构：API Server + Repo Server + Application Controller + Redis
- 同步策略：Manual（手动）、Auto（自动）、Prune（自动删除孤儿资源）
- 同步状态：Synced、OutOfSync、Unknown、Degraded

**多集群管理：**
- 单一 ArgoCD 管理多集群：通过 Service Account + Cluster Secret 注册
- Hub-Spoke 模式：中心 ArgoCD 管理多个 Spoke 集群
- ApplicationSet：批量生成 Application，支持 Cluster Generator、Git Directory Generator

**高频面试题：**
1. ArgoCD 和 Jenkins/GitLab CI 的关系是什么？如何配合使用？
2. 如何实现 ArgoCD 的多集群管理？有哪些架构模式？
3. ArgoCD 如何处理敏感配置？（Sealed Secrets、Vault Integration）
4. GitOps 的回滚机制是什么？如何快速回滚到上一个版本？

---

### 5.6 灾备与应急演练

**灾备架构模式：**
- Active-Active：双活，流量同时分布，RTO ≈ 0
- Active-Passive：主备，故障时切换，RTO 取决于切换速度
- Pilot Light：核心服务常备，其他按需启动
- Warm Standby：备用环境持续运行，数据实时同步

**混沌工程核心原则：**
1. 建立稳态假设：定义系统的"正常"行为
2. 变量注入：模拟真实故障（网络延迟、Pod 杀死、CPU 压力）
3. 最小爆炸半径：从小范围实验开始，逐步扩大
4. 自动化运行：集成到 CI/CD，持续验证

**常用工具：**
- Chaos Mesh：K8s 原生混沌工程平台
- Litmus：CNCF 混沌工程项目
- Chaos Monkey：Netflix 开源，随机终止实例

**高频面试题：**
1. 如何设计一个跨区域的灾备方案？RTO/RPO 如何权衡？
2. 混沌工程实验如何设计？如何控制爆炸半径？
3. 故障演练和灾备切换的区别是什么？各自的关注点？
4. 如何衡量灾备方案的有效性？（演练频率、切换成功率）

---

### 5.7 Istio / Service Mesh

**核心架构：**
- Envoy Sidecar：代理所有流量，实现流量管理、安全、可观测性
- Istiod：控制面，包含 Pilot（配置分发）、Citadel（证书管理）、Galley（配置验证）
- 流量管理：VirtualService（路由规则）、DestinationRule（负载均衡）、Gateway（入口网关）
- 安全：mTLS 自动证书轮换、AuthorizationPolicy 访问控制

**流量治理场景：**
- 金丝雀发布：基于权重或 Header 的流量切分
- 故障注入：延迟、中断，测试系统韧性
- 熔断：异常检测、连接池限制
- 重试与超时：可配置的重试策略

**高频面试题：**
1. Istio 如何实现零信任网络？mTLS 的工作原理？
2. Sidecar 注入原理是什么？如何控制哪些 Pod 注入 Sidecar？
3. Istio 性能开销有多大？如何优化？
4. Istio 和 K8s Service 的关系是什么？流量如何从 Service 到 Sidecar？

---

### 5.8 OpenTelemetry

**核心概念：**
- 三大信号：Traces（链路追踪）、Metrics（指标）、Logs（日志）
- OTLP 协议：OpenTelemetry Protocol，统一的遥测数据传输协议
- SDK 自动埋点：支持 Java、Python、Go、Node.js 等主流语言
- Collector：数据收集、处理、导出的中间件

**与现有工具整合：**
- Prometheus：OTel Collector 可以 Prometheus 格式暴露指标
- Jaeger：OTel Collector 可以导出 Trace 到 Jaeger
- Grafana：支持 OTLP 数据源

**高频面试题：**
1. OpenTelemetry 解决了什么问题？为什么成为可观测性标准？
2. OTel Collector 的部署模式有哪些？（Agent vs Gateway）
3. 如何从 Jaeger/Prometheus 迁移到 OpenTelemetry？
4. OTel 如何实现跨语言的链路追踪？

---

### 5.9 AIOps 智能运维

**核心场景：**
- 异常检测：基于历史数据自动发现异常模式，替代静态阈值
- 根因分析：关联分析、因果推断，快速定位故障根因
- 智能告警：告警聚合、降噪、优先级排序
- 容量预测：基于趋势分析预测资源需求

**常用技术：**
- 时序分析：ARIMA、Prophet、LSTM
- 异常检测：Isolation Forest、One-Class SVM、Autoencoder
- 因果推断：PC 算法、Granger 因果检验

**高频面试题：**
1. AIOps 在告警降噪场景如何应用？效果如何衡量？
2. 异常检测算法如何选择？静态阈值 vs 机器学习的权衡？
3. 如何解决 AIOps 模型的可解释性问题？
4. AIOps 落地的主要挑战是什么？

---

### 5.10 容量规划方法论

**核心流程：**
1. 基线建立：确定关键指标（QPS、延迟、资源利用率）
2. 压测设计：模拟真实负载，找到系统瓶颈
3. 容量模型：建立资源与性能的量化关系
4. 预测分析：基于业务增长预测资源需求
5. 弹性策略：定义自动扩缩容规则

**压测工具：**
- JMeter：通用压测工具，支持多种协议
- Locust：Python 编写，分布式压测
- K6：Go 编写，现代化压测工具
- Vegeta：HTTP 负载测试工具

**高频面试题：**
1. 如何设计一个完整的压测方案？包含哪些要素？
2. 压测结果如何指导容量规划？如何确定安全水位？
3. 如何处理突发流量？（预案、弹性、限流）
4. 容量规划和成本优化如何平衡？

---

### 5.11 CIS Benchmarks / 容器安全基线

**核心内容：**
- CIS Docker Benchmark：容器运行时安全配置
- CIS Kubernetes Benchmark：K8s 集群安全配置
- 检查项分类：API Server、etcd、kubelet、网络策略、RBAC 等

**常用工具：**
- kube-bench：检查 K8s 是否符合 CIS Benchmark
- Trivy：容器镜像漏洞扫描
- Falco：运行时安全监控

**高频面试题：**
1. 如何评估 K8s 集群的安全基线？
2. 容器镜像安全扫描的流程是什么？如何集成到 CI/CD？
3. 如何实现 Pod 安全策略？（PodSecurityPolicy vs PodSecurityAdmission）
4. 运行时安全监控如何实现？Falco 的工作原理？

---

### 5.12 Nacos 服务发现与配置管理

**核心原理：**
- Nacos = 注册中心 + 配置中心，支持 AP（服务发现，基于 Raft 变体）和 CP 模式切换
- 服务发现流程：服务启动时向 Nacos 注册（心跳维持），客户端订阅服务列表变更推送（长轮询）
- 与 Eureka 对比：Nacos 支持健康检查主动探测（而非仅依赖心跳），支持临时/持久实例
- 配置动态推送：客户端长轮询（29.5s 超时）检测配置变更，变更后服务端立即响应

**高频面试题：**
1. Nacos 集群如何保证一致性？CP 和 AP 模式分别在什么场景下切换？
2. 大规模服务注册（1000+ 实例）时 Nacos 的性能瓶颈在哪？如何优化？
3. 服务提供者异常下线（进程 crash）时，消费者最快多久感知到？（心跳超时 + 推送延迟）
4. Nacos 与 K8s Service/CoreDNS 的服务发现有什么本质区别？在云原生场景下怎么选型？

---

### 5.13 Post-mortem 故障复盘方法论

**核心原理：**
- 无责文化（Blameless Post-mortem）：关注系统和流程失效，而非追究个人责任
- 5 Why 根因分析：对"为什么"连续追问 5 次，找到根本原因而非表面症状
- 故障时间线重建：用 UTC 时间精确记录每个事件，避免事后记忆偏差
- 改进项优先级：按影响范围和实现成本排序，明确 Owner 和 Deadline

**Post-mortem 标准模板：**
```
- 影响范围（用户数、持续时间、影响的 SLO）
- 故障时间线
- 根因分析（5 Why）
- 贡献因素（触发因素 vs 根本原因的区别）
- 改进措施（检测改进 / 缓解改进 / 预防改进）
- 经验总结
```

**高频面试题：**
1. 你主导或参与过的最复杂故障是什么？如何组织复盘？
2. 5 Why 分析的陷阱是什么？什么情况下需要超过 5 次追问？
3. 如何区分"触发因素"和"根本原因"？举例说明
4. 改进项如何跟踪落地？如何防止 Post-mortem 变成形式主义？
5. 如何建立故障知识库，让团队从历史故障中持续学习？

---

### 5.14 Ansible 自动化运维实战（深度补充）

**核心原理：**
- 无 Agent 架构：通过 SSH 推送执行，控制节点需要 Python，被控节点只需 SSH + Python
- Playbook 执行模型：play → task → module，每个 task 幂等执行
- 变量优先级（从低到高）：role defaults → inventory → playbook vars → extra_vars（-e 参数）
- Handler 机制：只有 task notifies 且该 task 确实 changed 时才触发，避免重复重启

**高频面试题：**
1. Ansible 和 Shell 脚本批量执行的本质区别是什么？为什么强调幂等性？
2. Ansible Vault 如何管理加密的密钥和密码？在 CI/CD 中如何使用？
3. 如何优化 Ansible 执行速度？（pipelining、forks、facts caching）
4. 滚动更新（serial）如何配置？如何在批量更新中实现蓝绿发布？
5. Dynamic Inventory 是什么？如何从 AWS/阿里云动态获取主机列表？

---

### 5.15 CI/CD 流水线实战（深度补充）

**核心原理：**
- Jenkins Pipeline as Code：Declarative Pipeline（结构化）vs Scripted Pipeline（Groovy 灵活）
- GitLab CI 执行模型：Runner（注册到 GitLab）拉取 Job，支持 Docker executor / K8s executor
- 流水线安全：Secret 通过 CI/CD Variables 注入，不能出现在 job log 中，镜像需扫描 CVE
- 部署策略：Recreate（停机）/ Rolling Update / Blue-Green / Canary，各有不同 RTO

**高频面试题：**
1. 如何设计一条从代码提交到生产部署的完整流水线？包含哪些 Stage？
2. Jenkins Shared Library 的作用是什么？如何实现跨团队流水线复用？
3. GitLab CI 和 Jenkins 在大规模使用时各自的痛点是什么？
4. 如何实现流水线的失败快速反馈？（并行 stage、测试分级）
5. 镜像 Tag 策略如何设计？latest tag 在生产环境有什么风险？
6. 如何在流水线中实现数据库 Migration 的安全执行？（向后兼容原则）

---

## 六、面试准备优先级路线图

```
第一阶段（核心必考，2 周）
├── SLI/SLO/Error Budget 体系 → 理论框架 + PromQL 实现
├── Grafana + AlertManager → Dashboard 设计 + 告警路由
└── ELK/EFK 日志体系 → 架构设计 + ES 核心原理

第二阶段（差异化优势，2 周）
├── ArgoCD / GitOps → 实操演练 + 多集群场景
├── Terraform → 写一个实际云资源的 IaC 项目
└── 灾备 + 混沌工程 → 方案设计能力

第三阶段（加分项，按岗位定制）
├── Istio / Service Mesh → 原理理解 + 流量治理
├── OTel → 与现有 Jaeger 知识整合
├── CIS 容器安全 → 结合现有 K8s RBAC 知识扩展
└── AIOps → 结合 AutoGen 经验延伸
```

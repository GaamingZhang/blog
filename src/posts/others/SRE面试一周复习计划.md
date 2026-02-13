---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: true
article: true
category: SRE
tag:
  - 面试
  - SRE
  - 复习计划
---

# SRE 面试一周复习计划

> 基于对 11 家公司 SRE 岗位 JD 的综合分析，按**投资回报率**排序，帮助在一周内系统复习最高频考察方向。

---

## 📊 优先级依据

根据 JD 分析，各技术领域出现频次如下：

| 优先级 | 技术领域 | JD 频次 | 博客覆盖情况 |
|--------|---------|---------|-------------|
| 🔴 必须 | Linux / Shell | 11/11 | ✅ 已覆盖 |
| 🔴 必须 | Kubernetes / Docker | 11/11 | ✅ 深度覆盖 |
| 🔴 必须 | 监控体系（Prometheus / Grafana） | 9/11 | ✅ 已完善 |
| 🔴 必须 | CI/CD 流水线 | 10/11 | ✅ 已完善 |
| 🔴 必须 | Python / Go / Shell 开发 | 10/11 | ✅ 已完善 |
| 🔴 必须 | SLI/SLO/SLA 体系 | 6/11 | ✅ 已完善 |
| 🟡 加分 | ELK/EFK 日志体系 | 8/11 | ✅ 已完善 |
| 🟡 加分 | 灾备 / 混沌工程 | 8/11 | ✅ 已完善 |
| 🟡 加分 | IaC（Terraform / Ansible） | 7/11 | ✅ 已完善 |
| 🟡 加分 | ArgoCD / GitOps | 5/11 | ✅ 已完善 |
| 🟢 差异 | Service Mesh（Istio） | 4/11 | ✅ 已完善 |
| 🟢 差异 | AIOps | 4/11 | ✅ 已完善 |

---

## 📅 第一天：SRE 理论核心（必考 · 高频）

> **目标**：掌握 SRE 最核心的方法论，几乎所有面试都会问

### 上午：SLI/SLO/Error Budget

| 文章 | 链接 |
|------|------|
| SLI/SLO/SLA 体系与 Error Budget 实践 | [/posts/sre/SLI-SLO-SLA体系与ErrorBudget实践.html](https://www.gaaming.com.cn/posts/sre/SLI-SLO-SLA体系与ErrorBudget实践.html) |

**重点掌握**：
- SLI/SLO/SLA 三者的关系与区别
- Error Budget 的计算方式（1 - SLO 目标）
- Error Budget Policy：耗尽后如何冻结发布
- 用 PromQL 计算 28 天滚动窗口可用性
- Burn Rate 告警的优势（优于单纯阈值告警）

### 下午：故障复盘方法论

| 文章 | 链接 |
|------|------|
| Post-mortem 故障复盘方法论 | [/posts/sre/Post-mortem故障复盘方法论.html](https://www.gaaming.com.cn/posts/sre/Post-mortem故障复盘方法论.html) |

**重点掌握**：
- Blameless Post-mortem 文化
- 5 Why 根因分析方法
- 故障时间线重建的规范
- 改进项的分类（检测 / 缓解 / 预防）

---

## 📅 第二天：监控可观测性（9/11 公司必考）

> **目标**：完整掌握 Prometheus + Grafana + AlertManager 监控三件套

### 上午：Prometheus 核心与 Grafana 实战

| 文章 | 链接 |
|------|------|
| Prometheus 基本概念 | [/posts/others/Prometheus基本概念.html](https://www.gaaming.com.cn/posts/others/Prometheus基本概念.html) |
| Grafana 实战：Dashboard 设计与告警配置 | [/posts/grafana/Grafana实战-Dashboard设计与告警配置.html](https://www.gaaming.com.cn/posts/grafana/Grafana实战-Dashboard设计与告警配置.html) |
| Grafana 核心机制 | [/posts/grafana/Grafana核心机制.html](https://www.gaaming.com.cn/posts/grafana/Grafana核心机制.html) |
| Grafana 跨集群监控实现 | [/posts/grafana/Grafana跨集群监控实现.html](https://www.gaaming.com.cn/posts/grafana/Grafana跨集群监控实现.html) |
| PromQL 中 rate 和 irate 的区别 | [/posts/grafana/PromQL中rate和irate的区别.html](https://www.gaaming.com.cn/posts/grafana/PromQL中rate和irate的区别.html) |
| Prometheus Recording Rule 详解 | [/posts/others/Prometheus-recording-rule详解.html](https://www.gaaming.com.cn/posts/others/Prometheus-recording-rule详解.html) |

### 下午：AlertManager 告警设计

| 文章 | 链接 |
|------|------|
| AlertManager 如何避免告警风暴 | [/posts/others/AlertManager如何避免告警风暴.html](https://www.gaaming.com.cn/posts/others/AlertManager如何避免告警风暴.html) |
| AlertManager 多租户告警路由设计 | [/posts/others/AlertManager多租户告警路由设计.html](https://www.gaaming.com.cn/posts/others/AlertManager多租户告警路由设计.html) |
| 怎么保证报警的有效性 | [/posts/others/怎么保证报警的有效性.html](https://www.gaaming.com.cn/posts/others/怎么保证报警的有效性.html) |
| 告警阈值设计：静态与动态阈值的选型与实践 | [/posts/sre/告警阈值设计-静态与动态阈值的选型与实践.html](https://www.gaaming.com.cn/posts/sre/告警阈值设计-静态与动态阈值的选型与实践.html) |
| 如何监控 JVM/Node/Kubernetes 组件与 exporter 体系设计 | [/posts/sre/如何监控JVM-Node-Kubernetes组件与exporter体系设计.html](https://www.gaaming.com.cn/posts/sre/如何监控JVM-Node-Kubernetes组件与exporter体系设计.html) |
| Prometheus 长期存储方案选型指南 | [/posts/sre/Prometheus长期存储方案选型指南.html](https://www.gaaming.com.cn/posts/sre/Prometheus长期存储方案选型指南.html) |

---

## 📅 第三天：Kubernetes 深度复习（11/11 公司必考）

> **目标**：巩固 K8s 核心机制，突出大规模集群运维经验

### 上午：集群运维与安全

| 文章 | 链接 |
|------|------|
| 不停机升级 Kubernetes 集群版本（一） | [/posts/kubernetes/不停机升级Kubernetes集群版本（一）.html](https://www.gaaming.com.cn/posts/kubernetes/不停机升级Kubernetes集群版本（一）.html) |
| 不停机升级 Kubernetes 集群版本（二） | [/posts/kubernetes/不停机升级Kubernetes集群版本（二）.html](https://www.gaaming.com.cn/posts/kubernetes/不停机升级Kubernetes集群版本（二）.html) |
| Kubernetes 容器安全 CIS 基线实践 | [/posts/kubernetes/Kubernetes容器安全CIS基线实践.html](https://www.gaaming.com.cn/posts/kubernetes/Kubernetes容器安全CIS基线实践.html) |
| RBAC 权限控制 | [/posts/kubernetes/RBAC权限控制.html](https://www.gaaming.com.cn/posts/kubernetes/RBAC权限控制.html) |
| Secrets 管理最佳实践 | [/posts/kubernetes/Secrets管理最佳实践.html](https://www.gaaming.com.cn/posts/kubernetes/Secrets管理最佳实践.html) |
| NetworkPolicy 精细化隔离落地实践 | [/posts/kubernetes/NetworkPolicy精细化隔离落地实践.html](https://www.gaaming.com.cn/posts/kubernetes/NetworkPolicy精细化隔离落地实践.html) |

### 下午：高可用与可观测性

| 文章 | 链接 |
|------|------|
| Kubernetes 监控与告警 | [/posts/kubernetes/监控与告警.html](https://www.gaaming.com.cn/posts/kubernetes/监控与告警.html) |
| Kubernetes 日志收集方案 | [/posts/kubernetes/日志收集方案.html](https://www.gaaming.com.cn/posts/kubernetes/日志收集方案.html) |
| Ingress 控制器（金丝雀发布、限流、OAuth2） | [/posts/kubernetes/Ingress控制器.html](https://www.gaaming.com.cn/posts/kubernetes/Ingress控制器.html) |
| HPA 水平自动扩缩 | [/posts/kubernetes/HPA水平自动扩缩.html](https://www.gaaming.com.cn/posts/kubernetes/HPA水平自动扩缩.html) |
| VPA 垂直自动扩缩 | [/posts/kubernetes/VPA垂直自动扩缩.html](https://www.gaaming.com.cn/posts/kubernetes/VPA垂直自动扩缩.html) |
| CRI 与容器运行时 | [/posts/kubernetes/CRI与容器运行时.html](https://www.gaaming.com.cn/posts/kubernetes/CRI与容器运行时.html) |
| Pod 创建失败的排查流程 | [/posts/kubernetes/Pod创建失败的排查流程.html](https://www.gaaming.com.cn/posts/kubernetes/Pod创建失败的排查流程.html) |

---

## 📅 第四天：CI/CD 与 GitOps（10/11 公司必考）

> **目标**：覆盖现代 DevOps 完整交付链路

### 上午：流水线设计实战

| 文章 | 链接 |
|------|------|
| Jenkins/GitLab CI 流水线设计实战 | [/posts/sre/Jenkins-GitLab-CI流水线设计实战.html](https://www.gaaming.com.cn/posts/sre/Jenkins-GitLab-CI流水线设计实战.html) |
| CI/CD 的理解 | [/posts/sre/CICD的理解.html](https://www.gaaming.com.cn/posts/sre/CICD的理解.html) |
| CI/CD 工具链 | [/posts/sre/CICD工具链.html](https://www.gaaming.com.cn/posts/sre/CICD工具链.html) |

### 下午：GitOps 与 ArgoCD

| 文章 | 链接 |
|------|------|
| ArgoCD 与 GitOps 流程设计 | [/posts/sre/ArgoCD与GitOps流程设计.html](https://www.gaaming.com.cn/posts/sre/ArgoCD与GitOps流程设计.html) |

**重点掌握**：
- GitOps vs 传统 Push 模式的安全边界差异
- ArgoCD Application / AppProject / ApplicationSet 的职责
- Sync Wave、PreSync/PostSync Hook 的使用场景
- 多集群 Hub-Spoke 部署模式
- 蓝绿 / 金丝雀部署：ArgoCD Rollouts 原理

---

## 📅 第五天：IaC 与自动化（7/11 公司必考）

> **目标**：掌握基础设施即代码的核心能力

### 上午：Terraform 与云资源管理

| 文章 | 链接 |
|------|------|
| Terraform 核心概念与云资源管理实战 | [/posts/sre/Terraform核心概念与云资源管理实战.html](https://www.gaaming.com.cn/posts/sre/Terraform核心概念与云资源管理实战.html) |

**重点掌握**：
- State 文件作用 + Remote Backend + State Lock 机制
- Terraform vs Ansible 的场景边界
- 多环境管理：workspace vs 目录隔离
- Module 设计原则与循环依赖避免
- `terraform plan` 核心执行步骤

### 下午：Ansible 自动化运维

| 文章 | 链接 |
|------|------|
| Ansible 自动化运维 Playbook 实战 | [/posts/sre/Ansible自动化运维Playbook实战.html](https://www.gaaming.com.cn/posts/sre/Ansible自动化运维Playbook实战.html) |
| Ansible 基本概念 | [/posts/sre/Ansible基本概念.html](https://www.gaaming.com.cn/posts/sre/Ansible基本概念.html) |

**重点掌握**：
- Playbook 高级编排：Handlers、Block/Rescue/Always
- Ansible Vault 加密体系与 CI/CD 集成
- Dynamic Inventory 机制
- 滚动发布（serial）：金丝雀 + 熔断配置

---

## 📅 第六天：日志体系 + 灾备高可用（8/11 公司必考）

> **目标**：掌握完整日志架构设计与灾备方案

### 上午：ELK/EFK 全栈日志体系

| 文章 | 链接 |
|------|------|
| ELK 全栈日志体系架构与实战 | [/posts/elasticsearch/ELK全栈日志体系架构与实战.html](https://www.gaaming.com.cn/posts/elasticsearch/ELK全栈日志体系架构与实战.html) |
| Elasticsearch 集群黄色/红色状态排查与恢复 | [/posts/elasticsearch/Elasticsearch集群黄色红色状态排查与恢复.html](https://www.gaaming.com.cn/posts/elasticsearch/Elasticsearch集群黄色红色状态排查与恢复.html) |
| Elasticsearch 写入查询流程与 refresh_interval 调优 | [/posts/elasticsearch/Elasticsearch写入查询流程与refresh_interval调优.html](https://www.gaaming.com.cn/posts/elasticsearch/Elasticsearch写入查询流程与refresh_interval调优.html) |
| Elasticsearch 索引分片规划与主分片不可变原理 | [/posts/elasticsearch/Elasticsearch索引分片规划与主分片不可变原理.html](https://www.gaaming.com.cn/posts/elasticsearch/Elasticsearch索引分片规划与主分片不可变原理.html) |
| Elasticsearch 冷热数据分层架构设计 | [/posts/elasticsearch/Elasticsearch冷热数据分层架构设计.html](https://www.gaaming.com.cn/posts/elasticsearch/Elasticsearch冷热数据分层架构设计.html) |
| 生产环境日志规范化与结构化日志实践 | [/posts/sre/生产环境日志规范化与结构化日志实践.html](https://www.gaaming.com.cn/posts/sre/生产环境日志规范化与结构化日志实践.html) |
| Kubernetes DaemonSet 部署采集器注意事项 | [/posts/kubernetes/Kubernetes中DaemonSet部署采集器注意事项.html](https://www.gaaming.com.cn/posts/kubernetes/Kubernetes中DaemonSet部署采集器注意事项.html) |

### 下午：灾备与混沌工程

| 文章 | 链接 |
|------|------|
| 灾备方案设计与混沌工程实践 | [/posts/sre/灾备方案设计与混沌工程实践.html](https://www.gaaming.com.cn/posts/sre/灾备方案设计与混沌工程实践.html) |
| 容量规划方法论与压测实践 | [/posts/sre/容量规划方法论与压测实践.html](https://www.gaaming.com.cn/posts/sre/容量规划方法论与压测实践.html) |

**重点掌握**：
- RTO / RPO 的定义与业务换算
- 双活 vs 主备的数据一致性挑战
- Chaos Mesh 故障注入分类
- 容量规划五步法（基线 → 压测 → 预测 → 冗余 → 预案）
- Little's Law 在并发估算中的应用

---

## 📅 第七天：加分项与综合复习

> **目标**：差异化竞争力 + 查漏补缺

### 上午：Service Mesh 与可观测性进阶

| 文章 | 链接 |
|------|------|
| Istio 与 Service Mesh 微服务治理实践 | [/posts/sre/Istio与ServiceMesh微服务治理实践.html](https://www.gaaming.com.cn/posts/sre/Istio与ServiceMesh微服务治理实践.html) |
| OpenTelemetry 可观测性标准与实践 | [/posts/sre/OpenTelemetry可观测性标准与实践.html](https://www.gaaming.com.cn/posts/sre/OpenTelemetry可观测性标准与实践.html) |
| Jaeger 链路追踪原理 | [/posts/sre/Jaeger链路追踪原理.html](https://www.gaaming.com.cn/posts/sre/Jaeger链路追踪原理.html) |
| Jaeger 基本概念 | [/posts/sre/Jaeger基本概念.html](https://www.gaaming.com.cn/posts/sre/Jaeger基本概念.html) |

### 下午：运维开发 + 云成本 + AIOps

| 文章 | 链接 |
|------|------|
| Go 语言编写运维自动化工具实践 | [/posts/sre/Go语言编写运维自动化工具实践.html](https://www.gaaming.com.cn/posts/sre/Go语言编写运维自动化工具实践.html) |
| AIOps 智能运维实践 | [/posts/sre/AIOps智能运维实践.html](https://www.gaaming.com.cn/posts/sre/AIOps智能运维实践.html) |
| 云成本优化实践 | [/posts/sre/云成本优化实践.html](https://www.gaaming.com.cn/posts/sre/云成本优化实践.html) |
| Nacos 注册中心与配置管理实战 | [/posts/others/Nacos注册中心与配置管理实战.html](https://www.gaaming.com.cn/posts/others/Nacos注册中心与配置管理实战.html) |

---

## 🎯 面试高频场景题备忘

在复习文章的同时，要能流畅回答以下场景题（结合 STAR 模型）：

### 监控告警
- 如何设计多租户告警路由？（AlertManager routing tree）
- 告警风暴如何处理？（group_by + inhibit rules）
- 如何用 SLO Burn Rate 替代阈值告警？

### 故障排查
- CPU 100% 如何排查？→ [cpu100排查流程](https://www.gaaming.com.cn/posts/others/cpu100排查流程.html)
- 线上问题如何定位？→ [线上问题如何定位](https://www.gaaming.com.cn/posts/others/线上问题如何定位.html)
- ES 集群红色状态如何恢复？→ [ES集群黄色红色状态排查](https://www.gaaming.com.cn/posts/elasticsearch/Elasticsearch集群黄色红色状态排查与恢复.html)
- Pod 创建失败如何排查？→ [Pod创建失败排查流程](https://www.gaaming.com.cn/posts/kubernetes/Pod创建失败的排查流程.html)

### 系统设计
- 设计一个日志采集系统：规模、高可用、延迟
- 设计一个多云环境的监控体系
- 如何实现零停机 K8s 集群升级？

### Kubernetes 核心原理
- 调度器如何选择节点？→ [Pod调度机制](https://www.gaaming.com.cn/posts/kubernetes/Pod调度机制.html)
- HPA 扩缩容触发时机与稳定窗口？→ [HPA水平自动扩缩](https://www.gaaming.com.cn/posts/kubernetes/HPA水平自动扩缩.html)
- RBAC 最小权限如何落地？→ [RBAC权限控制](https://www.gaaming.com.cn/posts/kubernetes/RBAC权限控制.html)

---

## 📌 最强优势（面试重点强调）

面试中要主动提及这些已有深度的方向：

1. **Kubernetes 全栈**（33 篇）：从 Pod 生命周期、调度、存储到多集群升级，有完整的实战经验
2. **Docker / 容器原理**（25 篇）：镜像构建、隔离原理、安全加固
3. **故障诊断**：CPU 100%、OOM、Pod 崩溃、ES 集群红状态——有清晰的排查思路
4. **监控告警体系**：Prometheus + AlertManager + Grafana 完整链路，包含多租户告警路由设计
5. **链路追踪**：Jaeger 原理 + OpenTelemetry 标准迁移路径

---

## 📋 按公司定制的侧重点

| 公司 | 重点补充 |
|------|---------|
| **盈米基金** | SLI/SLO + ArgoCD + Istio + Post-mortem + 容量规划 |
| **阿里** | K8s 大规模运维 + 容量规划 + 云成本优化 |
| **字节跳动** | 大规模 K8s + CI/CD 流水线 + 监控体系 |
| **拓竹** | Terraform + ArgoCD + CIS 安全基线 + Istio |
| **九瓴科技** | Go 语言运维工具 + AIOps |
| **如祺出行** | Nacos + K8s + 灾备方案 |
| **钛动科技** | 云成本优化 + CI/CD + Go 开发 |
| **网易** | ELK 全栈 + K8s 监控 + 灾备 |
| **IMPLUS** | ArgoCD + AIOps + Go 开发 |

---
date: 2026-01-30
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# Kubernetes 学习指南

这是一份完整的 Kubernetes 学习资料目录，涵盖从入门到生产环境的全方位内容。

## 📚 目录导航

### 一、基础入门

- [常用命令](./常用命令.md) - Kubernetes 常用命令快速参考
- [本地开发环境](./本地开发环境.md) - 搭建本地 K8s 开发环境
- [CRI 与容器运行时](./CRI与容器运行时.md) - CRI 接口原理、Docker 弃用原因与 containerd/CRI-O 对比

### 二、核心概念

#### Pod 基础
- [Pod 基础与生命周期](./Pod基础与生命周期.md) - Pod 的基本概念和生命周期管理
- [探针机制](./探针机制.md) - 健康检查、存活探针和就绪探针
- [Pod 调度机制](./Pod调度机制.md) - Pod 调度策略与亲和性配置
- [Pod 安全标准](./Pod安全标准.md) - Pod 安全最佳实践
- [Pod 创建失败的排查流程](./Pod创建失败的排查流程.md) - 问题诊断与解决

#### 服务与发现
- [service 概念](./service概念.md) - Service 服务抽象与类型
- [CoreDNS 与服务发现](./CoreDNS与服务发现.md) - 集群内服务发现机制

#### 资源管理
- [Namespace 与资源隔离](./Namespace与资源隔离.md) - 命名空间与多租户隔离
- [资源请求与限制](./资源请求与限制.md) - CPU、内存资源的配置与管理

### 三、工作负载控制器

- [deployment 文件结构](./deployment文件结构.md) - Deployment 配置详解
- [StatefulSet 详解](./StatefulSet详解.md) - 有状态应用管理
- [DaemonSet 详解](./DaemonSet详解.md) - 守护进程集管理
- [Job 与 CronJob](./Job与CronJob.md) - 批处理任务和定时任务

### 四、配置与密钥管理

- [ConfigMap 与 Secret](./ConfigMap与Secret.md) - 配置信息和敏感数据管理
- [Secrets 管理最佳实践](./Secrets管理最佳实践.md) - 密钥安全管理策略

### 五、存储管理

- [Volume 与持久化存储](./Volume与持久化存储.md) - 数据卷和持久化存储方案

### 六、网络管理

- [service 概念](./service概念.md) - Service 网络服务基础
- [Ingress 控制器](./Ingress控制器.md) - 外部访问与路由管理
- [CoreDNS 与服务发现](./CoreDNS与服务发现.md) - DNS 解析与服务发现
- [NetworkPolicy 网络策略](./NetworkPolicy网络策略.md) - 网络访问控制策略
- [NetworkPolicy 精细化隔离落地实践](./NetworkPolicy精细化隔离落地实践.md) - 零信任网络模型与精细化隔离方案
- [跨节点通信](./跨节点通信.md) - 节点间网络通信原理

### 七、自动扩缩容

- [HPA 水平自动扩缩](./HPA水平自动扩缩.md) - 基于指标的水平扩展
- [VPA 垂直自动扩缩](./VPA垂直自动扩缩.md) - 资源请求自动调整

### 八、安全与权限

- [RBAC 权限控制](./RBAC权限控制.md) - 基于角色的访问控制
- [Pod 安全标准](./Pod安全标准.md) - Pod 安全策略与加固
- [Secrets 管理最佳实践](./Secrets管理最佳实践.md) - 敏感信息安全管理
- [NetworkPolicy 网络策略](./NetworkPolicy网络策略.md) - 网络安全隔离
- [NetworkPolicy 精细化隔离落地实践](./NetworkPolicy精细化隔离落地实践.md) - 生产环境网络隔离落地
- [Kubernetes 容器安全 CIS 基线实践](./Kubernetes容器安全CIS基线实践.md) - kube-bench、Trivy、Falco 构建纵深防御体系

### 九、运维与监控

- [监控与告警](./监控与告警.md) - 集群监控和告警方案
- [日志收集方案](./日志收集方案.md) - 日志聚合与分析
- [集群升级策略](./集群升级策略.md) - 集群版本升级最佳实践
- [DaemonSet 部署采集器注意事项](./Kubernetes中DaemonSet部署采集器注意事项.md) - 生产环境 DaemonSet 采集器的七个关键配置

### 十、工具与生态

- [Helm 包管理](./Helm包管理.md) - Kubernetes 应用包管理工具
- [Operator 核心原理](./Operator核心原理.md) - CRD、控制器模式与 Reconcile 循环
- [Operator 应用实践](./Operator应用实践.md) - Prometheus/MySQL/Cert-Manager Operator 及开发框架

---

## 🎯 学习路径建议

### 初学者路径
1. [本地开发环境](./本地开发环境.md)
2. [常用命令](./常用命令.md)
3. [Pod 基础与生命周期](./Pod基础与生命周期.md)
4. [service 概念](./service概念.md)
5. [deployment 文件结构](./deployment文件结构.md)
6. [ConfigMap 与 Secret](./ConfigMap与Secret.md)

### 进阶路径
1. [Pod 调度机制](./Pod调度机制.md)
2. [探针机制](./探针机制.md)
3. [StatefulSet 详解](./StatefulSet详解.md)
4. [DaemonSet 详解](./DaemonSet详解.md)
5. [Volume 与持久化存储](./Volume与持久化存储.md)
6. [Ingress 控制器](./Ingress控制器.md)
7. [CoreDNS 与服务发现](./CoreDNS与服务发现.md)
8. [跨节点通信](./跨节点通信.md)

### 生产环境路径
1. [资源请求与限制](./资源请求与限制.md)
2. [HPA 水平自动扩缩](./HPA水平自动扩缩.md)
3. [VPA 垂直自动扩缩](./VPA垂直自动扩缩.md)
4. [RBAC 权限控制](./RBAC权限控制.md)
5. [Pod 安全标准](./Pod安全标准.md)
6. [NetworkPolicy 网络策略](./NetworkPolicy网络策略.md)
7. [Secrets 管理最佳实践](./Secrets管理最佳实践.md)
8. [监控与告警](./监控与告警.md)
9. [日志收集方案](./日志收集方案.md)
10. [集群升级策略](./集群升级策略.md)
11. [DaemonSet 部署采集器注意事项](./Kubernetes中DaemonSet部署采集器注意事项.md)

### 运维与故障排查路径
1. [Pod 创建失败的排查流程](./Pod创建失败的排查流程.md)
2. [常用命令](./常用命令.md)
3. [监控与告警](./监控与告警.md)
4. [日志收集方案](./日志收集方案.md)

---

## 📝 快速索引

### 按主题分类

| 主题 | 核心文档 |
|------|----------|
| **基础概念** | [Pod 基础与生命周期](./Pod基础与生命周期.md) · [service 概念](./service概念.md) · [CRI 与容器运行时](./CRI与容器运行时.md) |
| **工作负载** | [deployment 文件结构](./deployment文件结构.md) · [StatefulSet 详解](./StatefulSet详解.md) · [DaemonSet 详解](./DaemonSet详解.md) |
| **配置管理** | [ConfigMap 与 Secret](./ConfigMap与Secret.md) · [Secrets 管理最佳实践](./Secrets管理最佳实践.md) |
| **存储方案** | [Volume 与持久化存储](./Volume与持久化存储.md) |
| **网络管理** | [Ingress 控制器](./Ingress控制器.md) · [CoreDNS 与服务发现](./CoreDNS与服务发现.md) · [NetworkPolicy 网络策略](./NetworkPolicy网络策略.md) · [NetworkPolicy 精细化隔离落地实践](./NetworkPolicy精细化隔离落地实践.md) |
| **自动扩缩** | [HPA 水平自动扩缩](./HPA水平自动扩缩.md) · [VPA 垂直自动扩缩](./VPA垂直自动扩缩.md) |
| **安全加固** | [RBAC 权限控制](./RBAC权限控制.md) · [Pod 安全标准](./Pod安全标准.md) · [CIS 基线实践](./Kubernetes容器安全CIS基线实践.md) |
| **运维监控** | [监控与告警](./监控与告警.md) · [日志收集方案](./日志收集方案.md) · [DaemonSet 部署采集器注意事项](./Kubernetes中DaemonSet部署采集器注意事项.md) |
| **故障排查** | [Pod 创建失败的排查流程](./Pod创建失败的排查流程.md) |
| **工具生态** | [Helm 包管理](./Helm包管理.md) · [Operator 核心原理](./Operator核心原理.md) · [Operator 应用实践](./Operator应用实践.md) |

### 按使用场景

| 场景 | 推荐文档 |
|------|----------|
| 部署无状态应用 | [deployment 文件结构](./deployment文件结构.md) · [service 概念](./service概念.md) · [Ingress 控制器](./Ingress控制器.md) |
| 部署有状态应用 | [StatefulSet 详解](./StatefulSet详解.md) · [Volume 与持久化存储](./Volume与持久化存储.md) |
| 配置应用 | [ConfigMap 与 Secret](./ConfigMap与Secret.md) · [Secrets 管理最佳实践](./Secrets管理最佳实践.md) |
| 网络配置 | [service 概念](./service概念.md) · [Ingress 控制器](./Ingress控制器.md) · [NetworkPolicy 网络策略](./NetworkPolicy网络策略.md) |
| 资源优化 | [资源请求与限制](./资源请求与限制.md) · [HPA 水平自动扩缩](./HPA水平自动扩缩.md) · [VPA 垂直自动扩缩](./VPA垂直自动扩缩.md) |
| 安全加固 | [RBAC 权限控制](./RBAC权限控制.md) · [Pod 安全标准](./Pod安全标准.md) · [NetworkPolicy 网络策略](./NetworkPolicy网络策略.md) · [NetworkPolicy 精细化隔离落地实践](./NetworkPolicy精细化隔离落地实践.md) · [CIS 基线实践](./Kubernetes容器安全CIS基线实践.md) |
| 故障排查 | [Pod 创建失败的排查流程](./Pod创建失败的排查流程.md) · [日志收集方案](./日志收集方案.md) · [监控与告警](./监控与告警.md) |

---

## 🔍 核心概念图谱

```
┌─────────────────────────────────────────────────────────────┐
│                        Kubernetes 核心                        │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐              │
│  │   Pod    │───▶│ Service  │───▶│ Ingress  │              │
│  └──────────┘    └──────────┘    └──────────┘              │
│       │                                                       │
│       ▼                                                       │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐              │
│  │Deployment│    │StatefulSet│   │DaemonSet │              │
│  └──────────┘    └──────────┘    └──────────┘              │
│       │                │                │                     │
│       └────────────────┴────────────────┘                     │
│                       ▼                                       │
│              ┌─────────────────┐                             │
│              │   ConfigMap     │                             │
│              │   Secret        │                             │
│              │   Volume        │                             │
│              └─────────────────┘                             │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

---

💡 **提示**: 点击任何链接即可跳转到对应的文档页面

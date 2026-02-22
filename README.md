# Gaaming Zhang Blog

> Gaaming Zhang 的个人技术博客，保存开发相关的学习记录。

## 项目简介

这是一个基于 VuePress Theme Hope 构建的个人技术博客。项目使用 GitLab 作为代码托管平台，并且同步到 GitHub 仓库。通过 Jenkins CI/CD 流水线实现自动化构建与部署。博客部署在腾讯云虚拟机中，同时部署在阿里云虚拟机中作为备份。本地部署到 Kubernetes 集群中作为测试环境。

**博客地址**: [https://www.gaaming.com.cn](https://www.gaaming.com.cn)

## 技术栈

| 技术 | 说明 |
|------|------|
| [VuePress 2.0](https://vuepress.vuejs.org/) | 基于 Vue 3 的静态站点生成器 |
| [vuepress-theme-hope](https://theme-hope.vuejs.press/) | 功能强大的 VuePress 主题 |
| Vite | 构建工具 |
| pnpm | 包管理器 |
| Jenkins | CI/CD 流水线 |
| Docker & Kubernetes | 容器化部署 |

## 内容统计

| 分类 | 文章数 |
|------|--------|
| LeetCode 算法题 | 134 |
| Kubernetes | 41 |
| 操作系统 | 31 |
| Docker | 25 |
| 其他技术 | 25 |
| SRE/DevOps | 24 |
| 网络协议 | 21 |
| Elasticsearch | 7 |
| Grafana | 4 |
| Nginx | 4 |
| Kafka | 3 |
| MySQL | 10 |
| Redis | 8 |
| RocketMQ | 2 |
| Zookeeper | 1 |
| 场景设计题 | 2 |
| **总计** | **344+** |

## 项目结构

```
gaamingzhangblog/
├── src/
│   ├── .vuepress/              # VuePress 配置
│   │   ├── config.ts           # 站点配置
│   │   ├── navbar.ts           # 导航栏配置
│   │   ├── sidebar.ts          # 侧边栏配置
│   │   └── theme.ts            # 主题配置
│   ├── algorithm/              # 算法相关
│   │   └── leetcode/           # LeetCode 题解 (134篇)
│   ├── posts/                  # 技术文章
│   │   ├── docker/             # Docker 容器技术
│   │   ├── elasticsearch/      # Elasticsearch 搜索引擎
│   │   ├── grafana/            # Grafana 可视化
│   │   ├── kafka/              # Kafka 消息队列
│   │   ├── kubernetes/         # Kubernetes 容器编排
│   │   ├── mysql/              # MySQL 数据库
│   │   ├── network/            # 网络协议
│   │   ├── nginx/              # Nginx Web服务器
│   │   ├── operation_system/   # 操作系统
│   │   ├── others/             # 其他技术
│   │   ├── redis/              # Redis 缓存
│   │   ├── rocketmq/           # RocketMQ 消息队列
│   │   ├── scenario_question/  # 场景设计题
│   │   ├── sre/                # SRE/DevOps
│   │   └── zookeeper/          # Zookeeper 分布式协调
│   ├── intro.md                # 博客介绍
│   └── README.md               # 首页
├── k8s/                        # Kubernetes 部署配置
├── pipelines/                  # Jenkins 流水线配置
├── undocument/                 # 待整理的文档 (71篇)
├── package.json
├── Dockerfile
└── README.md
```

## 内容分类

### 算法题解 (134篇)
LeetCode 经典算法题解，涵盖：
- 数组、链表、字符串
- 二叉树、图
- 动态规划、贪心算法
- 回溯、DFS/BFS
- 双指针、滑动窗口
- 栈、队列、堆
- 哈希表、前缀树

### Kubernetes (41篇)
- Pod 基础与生命周期
- Deployment、StatefulSet、DaemonSet
- Service、Ingress 网络管理
- ConfigMap、Secret 配置管理
- RBAC 权限控制
- HPA/VPA 自动扩缩
- Helm 包管理
- 集群升级与迁移

### Docker (25篇)
- Docker 基本概念与架构
- Dockerfile 最佳实践
- Docker Compose 实战
- 容器网络、存储、日志
- 多架构镜像构建
- 容器安全实践

### SRE/DevOps (24篇)
- CI/CD 工具链设计
- Prometheus 监控告警
- 链路追踪 (Jaeger, OpenTelemetry)
- GitOps 与 ArgoCD
- Terraform 基础设施即代码
- 故障复盘方法论
- SLI/SLO/SLA 体系

### 操作系统 (31篇)
- Linux 系统启动流程
- 进程、线程、协程
- 内存管理与文件系统
- 常用命令 (awk, sed, top, curl)
- 进程间通信与同步
- 零拷贝原理

### 网络协议 (21篇)
- HTTP/HTTPS 协议
- TCP/UDP 协议
- DNS、CDN 工作原理
- SSL/TLS 加密
- 网络安全 (XSS, CSRF)
- gRPC、QUIC 协议

### 数据库
- **MySQL**: 索引、事务、锁、ACID 特性
- **Redis**: 数据结构、集群模式、缓存策略
- **Elasticsearch**: 索引设计、集群管理、性能调优

### 消息队列
- **Kafka**: 基本概念、消息延迟分析
- **RocketMQ**: 基本概念、消息延迟
- **RabbitMQ vs Kafka**: 选型对比

## 快速开始

### 环境要求

- Node.js >= 18
- pnpm >= 8

### 安装依赖

```bash
pnpm install
```

### 开发模式

```bash
pnpm docs:dev
```

访问 [http://localhost:8080](http://localhost:8080) 查看效果

### 构建生产版本

```bash
pnpm docs:build
```

构建产物输出到 `src/.vuepress/dist` 目录

### 清除缓存

```bash
pnpm docs:clean-dev
```

## 部署架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   GitLab    │────▶│   Jenkins   │────▶│  Docker     │
│   (代码)    │     │   (CI/CD)   │     │  (构建镜像) │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  微信推送   │◀────│  腾讯云     │◀────│ Kubernetes  │
│  (构建通知) │     │  (部署)     │     │ (运行容器)  │
└─────────────┘     └─────────────┘     └─────────────┘
```

## 文章格式

每篇 Markdown 文件包含 frontmatter：

```markdown
---
date: 2025-07-01
author: Gaaming Zhang
isOriginal: true
article: true
category: 分类名称
tag:
  - 标签1
  - 标签2
---
```

## 待整理文档

`undocument/` 目录包含 71 篇待整理的文档，包括：
- 更多 LeetCode 题解
- MongoDB 相关内容
- 设计模式
- 流水线设计
- 服务网格 (Istio)

## 联系方式

- **作者**: Gaaming Zhang
- **博客**: [https://www.gaaming.com.cn](https://www.gaaming.com.cn)

## 许可证

本项目仅供个人学习记录使用。

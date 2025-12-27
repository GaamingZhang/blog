# Gaaming Zhang Blog

> Gaaming Zhang 的个人博客，保存开发相关的日常学习记录。

## 项目简介

这是一个基于 [VuePress](https://vuepress.vuejs.org/) 构建的个人技术博客，专门用于保存开发相关的日常学习记录。博客涵盖了算法、网络协议、操作系统、容器技术、数据库、DevOps 等多个技术领域的技术文章。项目存储在本地 gitlab 中，实现提交自动触发 Jenkins 构建，部署至腾讯云服务器。部署结果由 Jenkins 调用 go-wxpush-cli 推送到微信。

## 技术栈

- **框架**: [VuePress 2.0](https://vuepress.vuejs.org/) - 基于 Vue 3 的静态站点生成器
- **主题**: [vuepress-theme-hope](https://theme-hope.vuejs.press/) - 功能强大的 VuePress 主题
- **构建工具**: Vite
- **包管理器**: pnpm

## 项目结构

```
gaamingzhangblog/
├── src/
│   ├── .vuepress/          # VuePress 配置和主题文件
│   │   ├── config.ts       # 站点配置
│   │   ├── theme.js        # 主题配置
│   │   ├── dist/           # 构建输出目录
│   │   └── .temp/          # 临时文件
│   ├── algorithm/          # 算法相关内容
│   │   └── leetcode/       # LeetCode 算法题
│   ├── posts/              # 博客文章
│   │   ├── devops/         # DevOps 相关
│   │   ├── docker/         # Docker 相关
│   │   ├── kafka/          # Kafka 相关
│   │   ├── kubernetes/     # Kubernetes 相关
│   │   ├── mysql/          # MySQL 相关
│   │   ├── network/        # 网络协议
│   │   ├── nginx/          # Nginx 相关
│   │   ├── operation_system/  # 操作系统
│   │   ├── others/         # 其他技术
│   │   ├── redis/          # Redis 相关
│   │   └── scenario_question/  # 场景题
│   └── undocument/         # 未完成的文档
├── package.json
└── README.md
```

## 内容分类

### 算法题
包含 LeetCode 经典算法题，涵盖：
- 数组操作
- 链表操作
- 树和图
- 动态规划
- 回溯算法
- 双指针
- 滑动窗口
- 等等

### 网络协议
- HTTP/HTTPS 协议
- TCP/UDP 协议
- IP 协议
- DNS 工作原理
- SSL/TLS
- CDN
- 网络安全（XSS、CSRF）

### 操作系统
- Linux 系统启动流程
- 进程、线程、协程
- 内存管理
- 文件系统
- 系统命令（awk、sed、top、curl 等）
- 进程间通信
- 死锁处理

### 容器与编排
- Docker 基本概念
- Kubernetes 基本概念与组件
- Pod 管理
- 服务网格（Istio）
- 容器网络

### 数据库
- MySQL 基本概念
- 索引数据结构
- Redis 集群模式
- 高可用方案

### DevOps
- CI/CD 工具链
- 监控与告警
- 链路追踪
- 日志管理
- 负载均衡

### 其他技术
- Serverless
- Prometheus
- AI 相关新技术
- 场景设计题

## 快速开始

### 环境要求

- Node.js >= 18
- pnpm >= 8

### 安装依赖

```bash
pnpm install
```

### 开发模式

启动本地开发服务器：

```bash
pnpm docs:dev
```

访问 [http://localhost:8080](http://localhost:8080) 查看效果

### 构建生产版本

构建静态网站：

```bash
pnpm docs:build
```

构建产物将输出到 `src/.vuepress/dist` 目录

### 清除缓存并启动开发模式

```bash
pnpm docs:clean-dev
```

### 更新依赖

```bash
pnpm docs:update-package
```

## 文章格式

每篇 Markdown 文件都包含以下 frontmatter：

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

- `date`: 文章发布日期
- `author`: 作者
- `isOriginal`: 是否原创
- `article`: 是否作为文章显示（false 表示未完成）
- `category`: 文章分类
- `tag`: 文章标签

## 联系方式

- 作者: Gaaming Zhang
- 博客: [Gaaming Zhang Blog](https://www.gaaming.com.cn)

## 贡献

目前该项目仅从本地gitlab仓库进行管理，该github仓库为镜像仓库。

## 更新日志

### v2.0.0
- 升级到 VuePress 2.0
- 使用 vuepress-theme-hope 主题
- 优化构建性能
- 增加内存配置（4GB）

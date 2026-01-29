---
date: 2026-01-26
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Docker
tag:
  - Docker
  - ClaudeCode
---

# Docker 介绍

## 什么是 Docker

Docker 是一个开源的容器化平台,它允许开发者将应用程序及其依赖项打包到一个可移植的容器中,然后可以在任何支持 Docker 的系统上运行。Docker 通过操作系统级别的虚拟化技术,提供了一种轻量级、快速且一致的应用部署方式。

### Docker 的核心价值

- **环境一致性**: 开发、测试、生产环境完全一致,消除"在我机器上可以运行"的问题
- **快速部署**: 秒级启动,相比传统虚拟机分钟级启动有质的飞跃
- **资源高效**: 共享宿主机内核,资源占用远小于虚拟机
- **版本控制**: 镜像支持版本管理,便于回滚和追溯
- **微服务架构**: 天然适合微服务的独立部署和扩展

---

## Docker 的核心概念

### 1. 镜像 (Image)

Docker 镜像是一个只读的模板,包含了运行应用程序所需的代码、运行时环境、库、环境变量和配置文件。

**特点**:
- 分层存储结构,每一层只记录与上一层的差异
- 可以基于已有镜像创建新镜像
- 通过 Dockerfile 定义镜像构建过程
- 可以存储在 Docker Hub 或私有镜像仓库

**示例**:
```bash
# 拉取官方 nginx 镜像
docker pull nginx:latest

# 查看本地镜像
docker images

# 删除镜像
docker rmi nginx:latest
```

### 2. 容器 (Container)

容器是镜像的运行实例,是一个独立运行的应用程序及其运行环境。容器之间相互隔离,但共享宿主机的操作系统内核。

**特点**:
- 轻量级,启动速度快
- 可以被创建、启动、停止、删除、暂停
- 每个容器都是相互隔离的、安全的平台
- 容器中的修改可以提交为新的镜像

**示例**:
```bash
# 运行一个 nginx 容器
docker run -d -p 80:80 --name my-nginx nginx

# 查看运行中的容器
docker ps

# 查看所有容器(包括停止的)
docker ps -a

# 停止容器
docker stop my-nginx

# 启动容器
docker start my-nginx

# 删除容器
docker rm my-nginx
```

### 3. 仓库 (Repository)

Docker 仓库是集中存储和分发镜像的地方。Docker Hub 是官方提供的公共仓库,企业也可以搭建私有仓库。

**常用操作**:
```bash
# 登录 Docker Hub
docker login

# 推送镜像到仓库
docker push username/image-name:tag

# 从仓库拉取镜像
docker pull username/image-name:tag

# 搜索镜像
docker search nginx
```

---

## Docker 架构

Docker 采用客户端-服务器 (C/S) 架构模式,主要包含以下组件:

### Docker 客户端 (Client)

用户通过 Docker 客户端与 Docker 守护进程通信,客户端可以通过命令行工具 (docker) 或 REST API 与守护进程交互。

### Docker 守护进程 (Daemon)

Docker 守护进程 (dockerd) 负责管理 Docker 对象,如镜像、容器、网络和卷。守护进程监听 Docker API 请求并处理。

### Docker 注册中心 (Registry)

存储 Docker 镜像的仓库,Docker Hub 是默认的公共注册中心。

### Docker 对象

- **镜像**: 只读模板
- **容器**: 镜像的可运行实例
- **网络**: 容器间的通信机制
- **卷**: 持久化数据存储

---

## Dockerfile 详解

Dockerfile 是一个文本文件,包含了一系列指令,用于自动化构建 Docker 镜像。

### 常用指令

```dockerfile
# 指定基础镜像
FROM node:18-alpine

# 设置工作目录
WORKDIR /app

# 复制文件到容器
COPY package*.json ./

# 执行命令
RUN npm install

# 复制应用代码
COPY . .

# 暴露端口
EXPOSE 3000

# 设置环境变量
ENV NODE_ENV=production

# 容器启动时执行的命令
CMD ["node", "server.js"]
```

### Dockerfile 最佳实践

1. **使用官方基础镜像**: 优先选择官方维护的基础镜像
2. **最小化层数**: 合并 RUN 指令减少镜像层数
3. **利用构建缓存**: 将变化频繁的指令放在后面
4. **多阶段构建**: 分离编译环境和运行环境,减小最终镜像大小
5. **.dockerignore**: 排除不必要的文件,加快构建速度

**多阶段构建示例**:
```dockerfile
# 构建阶段
FROM node:18 AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# 生产阶段
FROM node:18-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
EXPOSE 3000
CMD ["node", "dist/server.js"]
```

---

## Docker 网络

Docker 提供了多种网络模式,满足不同场景的需求。

### 网络模式

1. **bridge (桥接模式)**: 默认模式,容器通过虚拟网桥互相通信
2. **host (主机模式)**: 容器直接使用宿主机网络
3. **none (无网络模式)**: 容器没有网络接口
4. **container (容器模式)**: 与其他容器共享网络命名空间
5. **自定义网络**: 用户自定义的桥接网络

### 网络操作示例

```bash
# 创建自定义网络
docker network create my-network

# 查看网络列表
docker network ls

# 运行容器并连接到指定网络
docker run -d --name app1 --network my-network nginx

# 查看网络详情
docker network inspect my-network

# 将运行中的容器连接到网络
docker network connect my-network app2

# 断开容器与网络的连接
docker network disconnect my-network app2

# 删除网络
docker network rm my-network
```

---

## Docker 数据持久化

容器中的数据默认是临时的,容器删除后数据也会丢失。Docker 提供了两种主要的数据持久化方式。

### 1. 数据卷 (Volume)

由 Docker 管理的数据存储,推荐使用方式。

```bash
# 创建数据卷
docker volume create my-volume

# 查看数据卷列表
docker volume ls

# 使用数据卷运行容器
docker run -d -v my-volume:/data --name app nginx

# 查看数据卷详情
docker volume inspect my-volume

# 删除数据卷
docker volume rm my-volume

# 清理未使用的数据卷
docker volume prune
```

### 2. 绑定挂载 (Bind Mount)

直接将宿主机的目录或文件挂载到容器中。

```bash
# 使用绑定挂载
docker run -d -v /host/path:/container/path --name app nginx

# 或使用 --mount 语法(更明确)
docker run -d \
  --mount type=bind,source=/host/path,target=/container/path \
  --name app nginx
```

### Volume vs Bind Mount

| 特性 | Volume | Bind Mount |
|------|--------|------------|
| 管理 | Docker 管理 | 用户管理 |
| 位置 | Docker 目录 | 任意位置 |
| 性能 | 更好 | 较好 |
| 可移植性 | 高 | 低 |
| 推荐场景 | 生产环境 | 开发环境 |

---

## Docker Compose

Docker Compose 是用于定义和运行多容器 Docker 应用程序的工具,通过 YAML 文件配置应用的服务。

### docker-compose.yml 示例

```yaml
version: '3.8'

services:
  web:
    build: ./web
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
      - DB_HOST=db
    depends_on:
      - db
    networks:
      - app-network
    volumes:
      - ./logs:/app/logs

  db:
    image: postgres:15
    environment:
      - POSTGRES_USER=admin
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=myapp
    volumes:
      - db-data:/var/lib/postgresql/data
    networks:
      - app-network

  redis:
    image: redis:7-alpine
    networks:
      - app-network

networks:
  app-network:
    driver: bridge

volumes:
  db-data:
```

### Compose 常用命令

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f web

# 停止所有服务
docker-compose stop

# 停止并删除容器、网络
docker-compose down

# 停止并删除容器、网络、卷
docker-compose down -v

# 重启服务
docker-compose restart web

# 执行命令
docker-compose exec web sh

# 构建或重新构建服务
docker-compose build
```

---

## Docker 常用命令速查

### 镜像相关

```bash
# 构建镜像
docker build -t image-name:tag .

# 列出镜像
docker images

# 删除镜像
docker rmi image-name:tag

# 导出镜像
docker save -o image.tar image-name:tag

# 导入镜像
docker load -i image.tar

# 查看镜像历史
docker history image-name:tag

# 标记镜像
docker tag source-image:tag target-image:tag
```

### 容器相关

```bash
# 运行容器
docker run -d --name container-name image-name

# 启动/停止/重启容器
docker start/stop/restart container-name

# 删除容器
docker rm container-name

# 强制删除运行中的容器
docker rm -f container-name

# 进入容器
docker exec -it container-name /bin/bash

# 查看容器日志
docker logs -f container-name

# 查看容器资源使用
docker stats container-name

# 复制文件
docker cp container-name:/path/in/container /path/on/host

# 查看容器详细信息
docker inspect container-name

# 暂停/恢复容器
docker pause/unpause container-name
```

### 系统相关

```bash
# 查看 Docker 信息
docker info

# 查看 Docker 版本
docker version

# 清理未使用的资源
docker system prune -a

# 查看磁盘使用情况
docker system df

# 实时查看容器资源使用
docker stats
```

---

## Docker 实战场景

### 场景 1: 开发环境统一

使用 Docker Compose 定义开发环境,确保团队成员环境一致。

```yaml
# docker-compose.dev.yml
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile.dev
    volumes:
      - .:/app
      - /app/node_modules
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=development
    command: npm run dev
```

### 场景 2: 微服务部署

```yaml
version: '3.8'

services:
  api-gateway:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - user-service
      - order-service

  user-service:
    build: ./services/user
    environment:
      - DB_HOST=user-db
    depends_on:
      - user-db

  order-service:
    build: ./services/order
    environment:
      - DB_HOST=order-db
    depends_on:
      - order-db

  user-db:
    image: postgres:15
    volumes:
      - user-db-data:/var/lib/postgresql/data

  order-db:
    image: postgres:15
    volumes:
      - order-db-data:/var/lib/postgresql/data

volumes:
  user-db-data:
  order-db-data:
```

### 场景 3: CI/CD 集成

```dockerfile
# Dockerfile for production
FROM node:18-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

COPY . .
RUN npm run build

FROM node:18-alpine
WORKDIR /app

COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
COPY package*.json ./

USER node
EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s \
  CMD node healthcheck.js

CMD ["node", "dist/server.js"]
```

---

## Docker 安全最佳实践

### 1. 使用官方镜像

优先使用官方维护的基础镜像,避免使用来源不明的镜像。

### 2. 最小权限原则

```dockerfile
# 不要使用 root 用户运行应用
FROM node:18-alpine

RUN addgroup -g 1001 -S nodejs
RUN adduser -S nodejs -u 1001

USER nodejs

WORKDIR /app
COPY --chown=nodejs:nodejs . .

CMD ["node", "server.js"]
```

### 3. 镜像扫描

```bash
# 使用 Docker Scan 扫描镜像漏洞
docker scan image-name:tag

# 使用 Trivy 扫描
trivy image image-name:tag
```

### 4. 限制资源使用

```bash
# 限制容器资源
docker run -d \
  --memory="512m" \
  --cpus="1.0" \
  --name app \
  nginx
```

### 5. 使用只读文件系统

```bash
# 使用只读根文件系统
docker run -d --read-only --name app nginx
```

### 6. 安全配置

```yaml
# docker-compose.yml 安全配置
version: '3.8'

services:
  app:
    image: myapp:latest
    read_only: true
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
```

---

## Docker 性能优化

### 1. 镜像优化

```dockerfile
# 使用更小的基础镜像
FROM node:18-alpine  # 而不是 node:18

# 合并 RUN 指令减少层数
RUN apt-get update && \
    apt-get install -y package1 package2 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# 利用构建缓存
COPY package*.json ./
RUN npm install
COPY . .
```

### 2. .dockerignore 文件

```
# .dockerignore
node_modules
npm-debug.log
.git
.gitignore
README.md
.env
.DS_Store
*.md
.vscode
coverage
.nyc_output
```

### 3. 多阶段构建

减小最终镜像大小,分离构建依赖和运行依赖。

```dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
CMD ["./server"]
```

### 4. 容器资源监控

```bash
# 查看容器资源使用
docker stats

# 限制容器资源
docker run -d \
  --memory="1g" \
  --memory-swap="2g" \
  --cpus="2.0" \
  --name app \
  myapp:latest
```

---

## Docker 与虚拟机的区别

| 特性 | Docker 容器 | 虚拟机 |
|------|------------|--------|
| 启动速度 | 秒级 | 分钟级 |
| 资源占用 | MB 级 | GB 级 |
| 性能 | 接近原生 | 有性能损耗 |
| 隔离级别 | 进程级 | 系统级 |
| 操作系统 | 共享宿主机内核 | 独立操作系统 |
| 可移植性 | 高 | 较低 |
| 系统支持 | 成百上千 | 几十个 |

**适用场景**:
- **Docker**: 微服务、快速部署、DevOps、CI/CD
- **虚拟机**: 需要完全隔离、运行不同操作系统、传统应用迁移

---

## Docker 生态系统

### 容器编排

- **Kubernetes (K8s)**: 最流行的容器编排平台,支持自动部署、扩展和管理
- **Docker Swarm**: Docker 官方的容器编排工具,配置简单
- **Apache Mesos**: 大规模集群管理系统

### 镜像仓库

- **Docker Hub**: 官方公共镜像仓库
- **Harbor**: 企业级私有镜像仓库
- **AWS ECR**: Amazon 容器镜像服务
- **Google Container Registry**: Google 云容器镜像服务
- **Azure Container Registry**: Microsoft 容器镜像服务

### 监控与日志

- **Prometheus + Grafana**: 监控和可视化
- **ELK Stack**: 日志收集和分析
- **cAdvisor**: 容器资源监控
- **Datadog**: 云监控平台

---

## 故障排查

### 常见问题排查思路

1. **容器无法启动**
```bash
# 查看容器日志
docker logs container-name

# 查看容器详细信息
docker inspect container-name

# 尝试交互式启动
docker run -it image-name /bin/sh
```

2. **网络连接问题**
```bash
# 检查容器网络
docker network inspect bridge

# 进入容器检查网络
docker exec -it container-name ping other-container

# 检查端口映射
docker port container-name
```

3. **磁盘空间不足**
```bash
# 查看磁盘使用
docker system df

# 清理未使用的资源
docker system prune -a --volumes
```

4. **性能问题**
```bash
# 查看资源使用
docker stats

# 检查容器进程
docker top container-name

# 查看容器事件
docker events
```

---

## 学习资源推荐

### 官方文档
- Docker 官方文档: https://docs.docker.com/
- Docker Hub: https://hub.docker.com/

### 实践平台
- Play with Docker: 免费的在线 Docker 环境
- Katacoda: 交互式 Docker 教程

### 社区资源
- Docker 官方博客
- GitHub Docker 项目
- Stack Overflow Docker 标签

---

## 总结

Docker 已经成为现代软件开发和部署的标准工具之一。通过容器化技术,Docker 解决了环境一致性、快速部署、资源利用等问题,极大地提升了开发和运维效率。

**核心要点**:
1. Docker 通过容器化提供轻量级的应用隔离
2. 镜像、容器、仓库是 Docker 的三大核心概念
3. Dockerfile 用于定义镜像构建过程
4. Docker Compose 简化多容器应用的管理
5. 掌握 Docker 网络和数据持久化是深入使用的关键
6. 安全性和性能优化是生产环境的重要考量

随着微服务架构和云原生应用的普及,Docker 及其生态系统将继续在软件开发领域发挥重要作用。

---

## 常见问题 (FAQ)

### 1. Docker 容器和虚拟机有什么本质区别?

**核心区别**在于隔离级别和资源使用方式:

- **Docker 容器**共享宿主机的操作系统内核,通过命名空间和控制组实现进程级隔离。容器启动快(秒级),资源占用少(MB 级),但隔离性相对较弱。

- **虚拟机**运行完整的客户操作系统,通过 Hypervisor 实现系统级隔离。启动慢(分钟级),资源占用大(GB 级),但提供更强的隔离性和安全性。

**选择建议**: 对于需要快速部署、高密度运行的微服务应用,选择 Docker;对于需要运行不同操作系统或要求强隔离的场景,选择虚拟机。两者也可以结合使用,在虚拟机中运行 Docker 容器。

### 2. 如何处理 Docker 容器中的数据持久化?

容器删除后数据会丢失,有三种主要的持久化方案:

**方案一: Volume (推荐)**
```bash
# 创建命名卷
docker volume create mydata
docker run -v mydata:/app/data myapp

# 优点: Docker 管理,可移植性好,性能优秀
```

**方案二: Bind Mount**
```bash
# 挂载宿主机目录
docker run -v /host/path:/container/path myapp

# 优点: 方便开发调试,实时同步
# 缺点: 依赖宿主机路径,可移植性差
```

**方案三: tmpfs Mount**
```bash
# 存储在内存中
docker run --tmpfs /app/temp myapp

# 适用: 临时敏感数据,不需要持久化
```

**最佳实践**: 生产环境使用 Volume,开发环境使用 Bind Mount,敏感临时数据使用 tmpfs。

### 3. 如何优化 Docker 镜像大小?

镜像太大会导致构建慢、部署慢、存储成本高。以下是优化策略:

**策略一: 选择合适的基础镜像**
```dockerfile
# 使用 alpine 版本(5MB) 而不是标准版(上百MB)
FROM node:18-alpine  # 而不是 FROM node:18
```

**策略二: 多阶段构建**
```dockerfile
# 构建阶段包含编译工具
FROM node:18 AS builder
WORKDIR /app
COPY . .
RUN npm install && npm run build

# 运行阶段只包含必要文件
FROM node:18-alpine
COPY --from=builder /app/dist ./dist
CMD ["node", "dist/server.js"]
```

**策略三: 减少镜像层数**
```dockerfile
# 合并 RUN 命令
RUN apt-get update && \
    apt-get install -y package1 package2 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
```

**策略四: 使用 .dockerignore**
```
node_modules
.git
*.md
.env
```

通过这些方法,可以将镜像从几GB优化到几十MB。

### 4. Docker 容器之间如何通信?

容器间通信有多种方式,取决于具体场景:

**方式一: 自定义网络(推荐)**
```bash
# 创建网络
docker network create mynetwork

# 容器加入同一网络
docker run --network mynetwork --name app1 image1
docker run --network mynetwork --name app2 image2

# app1 可以通过容器名访问 app2
# 例如: http://app2:3000
```

**方式二: Docker Compose (最简单)**
```yaml
version: '3.8'
services:
  web:
    image: nginx
  api:
    image: myapi
    # web 可以通过 http://api:8080 访问
```

**方式三: 容器连接 (legacy,不推荐)**
```bash
docker run --name db mysql
docker run --link db:database myapp
```

**方式四: Host 网络模式**
```bash
docker run --network host myapp
# 直接使用宿主机网络,性能最好但隔离性差
```

**最佳实践**: 使用自定义桥接网络或 Docker Compose,通过服务名进行通信,DNS 自动解析。

### 5. 如何保证 Docker 容器的安全性?

容器安全是生产环境的关键考量,需要多层防护:

**层面一: 镜像安全**
```dockerfile
# 使用官方镜像
FROM node:18-alpine

# 不使用 root 用户
RUN addgroup -g 1001 nodejs && \
    adduser -S nodejs -u 1001
USER nodejs

# 扫描漏洞
# docker scan myimage:latest
```

**层面二: 运行时安全**
```bash
# 只读文件系统
docker run --read-only myapp

# 限制权限
docker run --cap-drop=ALL --cap-add=NET_BIND_SERVICE myapp

# 资源限制
docker run --memory="512m" --cpus="1.0" myapp
```

**层面三: 网络安全**
```bash
# 使用自定义网络隔离
docker network create --internal backend-network

# 只暴露必要端口
docker run -p 127.0.0.1:3000:3000 myapp
```

**层面四: 密钥管理**
```bash
# 使用 Docker Secrets (Swarm)
echo "my_secret" | docker secret create db_password -
docker service create --secret db_password myapp

# 或使用环境变量 + 密钥管理工具
docker run -e DB_PASSWORD_FILE=/run/secrets/db_password myapp
```

**层面五: 定期更新**
```bash
# 定期更新基础镜像
docker pull node:18-alpine
docker build -t myapp:latest .

# 扫描和修复漏洞
trivy image myapp:latest
```

**综合建议**: 最小权限原则 + 定期扫描 + 网络隔离 + 密钥管理 + 持续更新,构建纵深防御体系。
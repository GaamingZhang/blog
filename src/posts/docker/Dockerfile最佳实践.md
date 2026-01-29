# Dockerfile最佳实践

## Dockerfile是什么？

如果Docker镜像是一道菜，那Dockerfile就是菜谱。它用文字描述了如何一步步"做出"这道菜：从什么基础镜像开始、安装什么软件、复制什么文件、最后运行什么命令。

理解Dockerfile的工作原理，能帮你写出更小、更快、更安全的镜像。

## 核心原则：层的概念

这是理解Dockerfile优化的关键：**Dockerfile中的每条指令都会创建一个新的"层"**。

```
FROM ubuntu:22.04        → 层1：基础镜像
RUN apt-get update       → 层2：更新包索引
RUN apt-get install nginx → 层3：安装nginx
COPY app.conf /etc/nginx/ → 层4：复制配置
```

每一层都是只读的，最终叠加在一起形成镜像。这意味着：

1. **层越多，镜像越大**：每层都有一定开销
2. **层可以被缓存**：如果某层没有变化，下次构建时可以复用
3. **删除文件不会减小镜像**：你只是在新层"标记"文件被删除，但文件仍存在于之前的层

## 选择合适的基础镜像

### 为什么基础镜像很重要？

基础镜像决定了你镜像的"底座"大小。选择一个500MB的基础镜像，你的镜像至少500MB起步。

| 基础镜像 | 大小 | 适用场景 |
|----------|------|----------|
| scratch | 0B | 静态编译的Go程序 |
| alpine | ~5MB | 大多数轻量级应用 |
| distroless | ~20MB | 安全敏感的生产环境 |
| debian-slim | ~50MB | 需要glibc的应用 |
| ubuntu | ~77MB | 需要完整工具链 |

### 推荐策略

**生产环境**：优先选择alpine或distroless
- 体积小，攻击面小
- 没有shell和包管理器（distroless），更安全

**开发环境**：可以用ubuntu/debian
- 工具齐全，调试方便

### 固定版本号

永远不要用`latest`标签！

```dockerfile
# 错误：不知道明天会变成什么版本
FROM python:latest

# 正确：版本固定，可重复构建
FROM python:3.11-slim-bookworm
```

`latest`今天可能是3.11，明天可能变成3.12，你的构建可能突然失败。

## 优化层的缓存

### 缓存的工作原理

Docker在构建时会检查每层是否有变化：
- 如果没有变化，直接使用缓存
- 如果有变化，这一层及之后所有层都要重新构建

这意味着：**把不常变化的指令放在前面，常变化的放在后面**。

### 依赖和代码分离

这是最重要的优化技巧之一：

```dockerfile
# 不好的写法：任何文件变化都导致重新安装依赖
COPY . .
RUN npm install

# 好的写法：只有package.json变化才重新安装
COPY package*.json ./
RUN npm install
COPY . .
```

第二种写法中：
- 如果只是代码变了，`npm install`可以用缓存
- 只有依赖文件变了，才会重新安装

对于一个有几百个依赖的项目，这能节省几分钟的构建时间。

## 减少镜像体积

### 合并RUN指令

每个RUN指令创建一层，在同一层内删除的临时文件不会占用空间：

```dockerfile
# 不好：三层，临时文件留在第一、第二层
RUN apt-get update
RUN apt-get install -y nginx
RUN rm -rf /var/lib/apt/lists/*

# 好：一层，临时文件在同一层被删除
RUN apt-get update && \
    apt-get install -y --no-install-recommends nginx && \
    rm -rf /var/lib/apt/lists/*
```

### 使用--no-install-recommends

apt默认会安装"推荐"的包，但你通常不需要它们：

```dockerfile
RUN apt-get install -y --no-install-recommends curl
```

### 使用.dockerignore

就像.gitignore一样，.dockerignore告诉Docker构建时忽略哪些文件：

```dockerignore
.git
node_modules
*.md
Dockerfile
.env
```

这些文件不会被COPY到镜像里，加快构建速度，减小镜像体积。

## 多阶段构建

这是现代Dockerfile的必备技能。

### 问题：构建工具污染镜像

编译一个Go程序需要Go编译器，但运行时不需要。如果把编译器也打包进镜像，白白增加几百MB。

### 解决方案：多阶段构建

```dockerfile
# 阶段1：构建
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o myapp

# 阶段2：运行
FROM alpine:3.18
COPY --from=builder /app/myapp /usr/local/bin/
CMD ["myapp"]
```

最终镜像只包含alpine和编译好的二进制文件，不包含Go编译器。

**效果对比**：
- 不用多阶段：~800MB（包含Go编译器）
- 使用多阶段：~15MB（只有alpine + 二进制）

### 多阶段的其他用途

**分离开发和生产依赖**：

```dockerfile
# 安装所有依赖（包括开发依赖）用于构建
FROM node:18-alpine AS builder
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# 只安装生产依赖用于运行
FROM node:18-alpine
COPY package*.json ./
RUN npm ci --only=production
COPY --from=builder /app/dist ./dist
CMD ["node", "dist/main.js"]
```

## 安全最佳实践

### 不要以root运行

容器内默认是root用户，这是个安全风险：

```dockerfile
# 创建普通用户
RUN adduser --disabled-password --no-create-home appuser

# 切换到普通用户
USER appuser
```

### 不要在镜像里存储密钥

```dockerfile
# 错误！密钥会留在镜像历史中
ENV API_KEY=secret123

# 正确：运行时通过环境变量传入
# docker run -e API_KEY=xxx myapp
```

即使你后来删除了这个ENV，它仍然存在于镜像的历史层中。

### 使用BuildKit的secrets

如果构建时确实需要密钥（比如私有npm仓库）：

```dockerfile
# syntax=docker/dockerfile:1
RUN --mount=type=secret,id=npm_token \
    NPM_TOKEN=$(cat /run/secrets/npm_token) npm install
```

```bash
docker build --secret id=npm_token,src=./.npmrc .
```

密钥只在构建时可用，不会留在镜像中。

## ENTRYPOINT vs CMD

这两个指令经常让人困惑。简单记忆：

- **ENTRYPOINT**：容器的"主程序"，不容易被覆盖
- **CMD**：默认参数，容易被覆盖

```dockerfile
ENTRYPOINT ["python", "app.py"]
CMD ["--port", "8080"]
```

运行时：
```bash
docker run myapp                    # 等于 python app.py --port 8080
docker run myapp --port 9090        # 等于 python app.py --port 9090
docker run myapp --debug            # 等于 python app.py --debug
```

### 推荐模式

```dockerfile
# 使用exec格式（不是shell格式）
ENTRYPOINT ["python", "app.py"]
CMD ["--config", "/app/config.json"]
```

exec格式`["python", "app.py"]`比shell格式`python app.py`更好，因为进程直接是PID 1，能正确接收信号。

## 健康检查

告诉Docker如何判断容器是否健康：

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1
```

- **interval**：多久检查一次
- **timeout**：检查超时时间
- **retries**：失败几次算不健康

健康检查结果会显示在`docker ps`中。

## 常见问题

### Q1: 构建很慢怎么办？

1. **优化层缓存**：不常变的指令放前面
2. **启用BuildKit**：`DOCKER_BUILDKIT=1`
3. **使用.dockerignore**：不复制不需要的文件
4. **使用缓存挂载**：对于pip、npm等包管理器

### Q2: 镜像太大怎么办？

1. 使用更小的基础镜像（alpine）
2. 使用多阶段构建
3. 合并RUN指令，在同一层清理临时文件
4. 删除不需要的文件（文档、测试、缓存）

### Q3: COPY和ADD有什么区别？

| 指令 | 功能 |
|------|------|
| COPY | 只复制文件 |
| ADD | 复制 + 自动解压tar + 支持URL |

**推荐**：优先用COPY，更明确可预测。只有需要自动解压时才用ADD。

### Q4: ARG和ENV有什么区别？

| 指令 | 作用时机 | 是否保留在镜像中 |
|------|----------|------------------|
| ARG | 只在构建时 | 不保留 |
| ENV | 构建时+运行时 | 保留 |

```dockerfile
ARG VERSION=1.0         # 只在构建时可用
ENV APP_VERSION=$VERSION # 运行时也可用
```

### Q5: 怎么调试Dockerfile构建？

```bash
# 查看详细构建过程
docker build --progress=plain .

# 构建到某个阶段停下
docker build --target builder .

# 进入某个阶段调试
docker run -it <中间镜像ID> /bin/sh
```

## 小结

| 原则 | 做法 |
|------|------|
| 选择合适的基础镜像 | 生产用alpine/distroless，固定版本号 |
| 优化缓存 | 不常变的指令放前面，依赖和代码分离 |
| 减小体积 | 合并RUN，清理临时文件，使用.dockerignore |
| 多阶段构建 | 分离构建环境和运行环境 |
| 安全 | 不用root，不存密钥，使用BuildKit secrets |
| 入口点 | 用exec格式，配合HEALTHCHECK |

记住：好的Dockerfile = 小镜像 + 快构建 + 安全可靠

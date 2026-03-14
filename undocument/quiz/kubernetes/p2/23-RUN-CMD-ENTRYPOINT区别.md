---
date: 2026-03-12
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Docker
tag:
  - Docker
  - Dockerfile
---

# Dockerfile 中 RUN、CMD、ENTRYPOINT 的区别

## 引言

在容器化技术日益普及的今天，Dockerfile 作为构建容器镜像的核心配置文件，其重要性不言而喻。然而，许多开发者在使用 Dockerfile 时，经常混淆 RUN、CMD 和 ENTRYPOINT 这三个指令的作用和使用场景。这种混淆不仅会导致镜像构建失败，还可能引发容器运行时的意外行为。

理解这三个指令的区别，对于编写高效、可维护的 Dockerfile 至关重要。RUN 指令决定了镜像的构建过程，CMD 和 ENTRYPOINT 则决定了容器的运行时行为。掌握它们的执行时机、格式差异以及组合使用方式，是成为一名优秀的容器化应用开发者的必备技能。

本文将深入剖析这三个指令的工作原理、执行机制和最佳实践，帮助您在实际项目中做出正确的技术决策。

## RUN 指令详解

### 执行时机与作用

RUN 指令是 Dockerfile 中最常用的指令之一，它在**镜像构建阶段**执行，用于在镜像层中执行命令并提交结果。每次执行 RUN 指令，都会创建一个新的镜像层（layer），这个层包含了命令执行后文件系统的变化。

### 核心机制

RUN 指令的执行过程可以分为以下几个步骤：

1. **启动临时容器**：基于当前镜像启动一个临时容器
2. **执行命令**：在临时容器中执行指定的命令
3. **提交变更**：将命令执行后的文件系统变更提交为新的镜像层
4. **更新镜像**：新镜像层成为后续指令的基础

### 两种格式

#### Shell 格式

```dockerfile
RUN apt-get update && apt-get install -y nginx
```

Shell 格式下，命令会通过 `/bin/sh -c` 执行。这意味着：

- 可以使用 shell 特性，如变量替换、通配符等
- 默认使用 `/bin/sh` 作为 shell 解释器
- 命令会被包装成 `/bin/sh -c "apt-get update && apt-get install -y nginx"`

#### Exec 格式

```dockerfile
RUN ["apt-get", "update"]
RUN ["apt-get", "install", "-y", "nginx"]
```

Exec 格式直接执行命令，不通过 shell：

- 命令以 JSON 数组形式提供
- 第一个元素是可执行文件路径，后续元素是参数
- 不会进行 shell 变量替换
- 需要明确指定可执行文件的完整路径

### 最佳实践

**合并多个命令**以减少镜像层数：

```dockerfile
# 不推荐：产生多个层
RUN apt-get update
RUN apt-get install -y nginx
RUN apt-get clean

# 推荐：合并为一个层
RUN apt-get update && apt-get install -y nginx \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
```

**清理临时文件**以减小镜像体积：

```dockerfile
RUN apt-get update && apt-get install -y \
    curl \
    && rm -rf /var/lib/apt/lists/*
```

## CMD 指令详解

### 执行时机与作用

CMD 指令在**容器运行时**执行，用于设置容器启动时默认执行的命令。与 RUN 不同，CMD 不会在构建阶段执行任何操作，它只是为镜像设置了默认的启动命令。

### 核心特性

CMD 指令具有以下关键特性：

- **可被覆盖**：通过 `docker run` 命令传递的参数会覆盖 CMD 指定的命令
- **只能有一个**：Dockerfile 中只有最后一个 CMD 会生效
- **默认行为**：为镜像提供默认的运行命令

### 三种格式

#### Shell 格式

```dockerfile
CMD echo "Hello, World!"
```

命令通过 `/bin/sh -c` 执行，支持 shell 变量替换。

#### Exec 格式（推荐）

```dockerfile
CMD ["nginx", "-g", "daemon off;"]
```

直接执行命令，不通过 shell，是生产环境推荐的方式。

#### 参数格式

```dockerfile
CMD ["param1", "param2"]
```

作为 ENTRYPOINT 的默认参数，需要与 ENTRYPOINT 配合使用。

### 执行机制

当容器启动时，Docker 会按照以下顺序确定要执行的命令：

1. 检查 `docker run` 是否提供了命令参数
2. 如果提供了，使用该命令覆盖 CMD
3. 如果没有提供，使用 CMD 指定的默认命令

### 使用示例

```dockerfile
# 示例1：启动 Web 服务器
CMD ["nginx", "-g", "daemon off;"]

# 示例2：启动应用程序
CMD ["python", "app.py"]

# 示例3：使用 shell 格式
CMD echo "Container started at $(date)"
```

## ENTRYPOINT 指令详解

### 执行时机与作用

ENTRYPOINT 同样在**容器运行时**执行，但它的作用是设置容器的**主命令**，使容器作为一个可执行程序运行。ENTRYPOINT 提供了比 CMD 更强的命令固定能力。

### 核心特性

ENTRYPOINT 的关键特性包括：

- **不易被覆盖**：`docker run` 的参数会追加到 ENTRYPOINT 后，而不是替换
- **可被强制覆盖**：使用 `--entrypoint` 参数可以覆盖
- **固定主命令**：适合将容器作为可执行程序

### 两种格式

#### Exec 格式（推荐）

```dockerfile
ENTRYPOINT ["docker-entrypoint.sh"]
```

这是最常用的格式，允许接收 CMD 或 `docker run` 传递的参数。

#### Shell 格式

```dockerfile
ENTRYPOINT echo "Hello"
```

Shell 格式会忽略 CMD 指令和 `docker run` 的参数，实际使用较少。

### 执行机制

ENTRYPOINT 的执行机制如下：

1. 容器启动时，ENTRYPOINT 指定的命令总是会被执行
2. CMD 的内容作为参数传递给 ENTRYPOINT
3. `docker run` 的参数会覆盖 CMD，然后传递给 ENTRYPOINT

### 典型应用场景

**数据库初始化脚本**：

```dockerfile
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["mysqld"]
```

**命令行工具封装**：

```dockerfile
ENTRYPOINT ["curl"]
CMD ["--help"]
```

用户运行 `docker run myimage http://example.com` 时，实际执行的命令是 `curl http://example.com`。

## CMD 和 ENTRYPOINT 的组合使用

### 组合机制

CMD 和 ENTRYPOINT 的组合遵循以下规则：

| ENTRYPOINT 格式 | CMD 格式 | 最终执行命令 |
|----------------|---------|-------------|
| Exec 格式 | Exec 格式 | ENTRYPOINT + CMD 参数 |
| Exec 格式 | Shell 格式 | Shell 格式的 CMD 被忽略 |
| Shell 格式 | 任意格式 | ENTRYPOINT 独立执行 |

### 经典组合模式

#### 模式一：ENTRYPOINT + CMD（推荐）

```dockerfile
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["postgres"]
```

这种模式允许：
- 默认运行 `docker-entrypoint.sh postgres`
- 通过 `docker run myimage mysql` 改为运行 `docker-entrypoint.sh mysql`

#### 模式二：ENTRYPOINT 作为可执行程序

```dockerfile
ENTRYPOINT ["python"]
CMD ["app.py"]
```

用户可以灵活使用：
- `docker run myimage` → 执行 `python app.py`
- `docker run myimage script.py` → 执行 `python script.py`
- `docker run myimage -m http.server` → 执行 `python -m http.server`

### 参数传递示例

```dockerfile
FROM alpine
ENTRYPOINT ["echo", "Hello"]
CMD ["World"]
```

执行结果：
- `docker run myimage` → 输出 `Hello World`
- `docker run myimage Docker` → 输出 `Hello Docker`

## Shell 格式和 Exec 格式的区别

### 核心差异对比

| 特性 | Shell 格式 | Exec 格式 |
|-----|-----------|----------|
| 执行方式 | 通过 `/bin/sh -c` | 直接执行 |
| Shell 变量替换 | 支持 | 不支持 |
| 信号处理 | PID 1 是 shell，信号被拦截 | PID 1 是应用本身 |
| 环境变量 | 可以使用 `$VAR` | 需要在命令中引用 |
| 性能 | 略低（多一层 shell） | 略高 |

### 信号处理机制

这是 Shell 格式和 Exec 格式最重要的区别之一。

**Shell 格式的问题**：

```dockerfile
CMD nginx -g "daemon off;"
```

容器启动后，进程树如下：

```
PID 1: /bin/sh -c nginx -g "daemon off;"
PID 7: nginx
```

当发送 SIGTERM 信号给容器时，信号被 `/bin/sh` 接收，但 `/bin/sh` 不会转发给 nginx，导致容器无法优雅关闭。

**Exec 格式的优势**：

```dockerfile
CMD ["nginx", "-g", "daemon off;"]
```

进程树变为：

```
PID 1: nginx
```

nginx 直接作为 PID 1 进程运行，可以正确接收和处理信号。

### 环境变量使用

**Shell 格式**：

```dockerfile
ENV NAME World
CMD echo "Hello, $NAME"
```

**Exec 格式**：

```dockerfile
ENV NAME World
CMD ["sh", "-c", "echo Hello, $NAME"]
```

Exec 格式需要显式调用 shell 才能使用变量替换。

## 三者对比表格

| 维度 | RUN | CMD | ENTRYPOINT |
|-----|-----|-----|-----------|
| **执行时机** | 镜像构建时 | 容器运行时 | 容器运行时 |
| **主要作用** | 构建镜像层 | 设置默认命令 | 设置主命令 |
| **可出现次数** | 多次 | 一次（最后一个生效） | 一次（最后一个生效） |
| **是否可覆盖** | N/A | docker run 参数可覆盖 | 需 --entrypoint 覆盖 |
| **是否创建层** | 是 | 否 | 否 |
| **推荐格式** | Exec 格式 | Exec 格式 | Exec 格式 |
| **典型场景** | 安装软件包、配置环境 | 设置默认启动命令 | 固定主命令、制作可执行镜像 |

## 多种使用场景示例

### 场景一：构建 Web 应用镜像

```dockerfile
FROM node:16-alpine

# RUN：构建阶段安装依赖
RUN apk add --no-cache tini

WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

COPY . .

# ENTRYPOINT：使用 tini 作为 init 进程
ENTRYPOINT ["/sbin/tini", "--"]

# CMD：默认启动命令
CMD ["node", "server.js"]
```

**说明**：
- RUN 用于安装系统依赖和 Node.js 包
- ENTRYPOINT 使用 tini 处理信号和僵尸进程
- CMD 设置默认启动命令

### 场景二：制作命令行工具镜像

```dockerfile
FROM python:3.9-slim

RUN pip install --no-cache-dir awscli

ENTRYPOINT ["aws"]
CMD ["--help"]
```

**使用方式**：

```bash
# 显示帮助信息
docker run myaws

# 执行 S3 命令
docker run myaws s3 ls

# 执行 EC2 命令
docker run myaws ec2 describe-instances
```

### 场景三：数据库镜像

```dockerfile
FROM postgres:13

# RUN：安装扩展
RUN apt-get update && apt-get install -y \
    postgresql-13-postgis \
    && rm -rf /var/lib/apt/lists/*

# ENTRYPOINT：初始化脚本
ENTRYPOINT ["docker-entrypoint.sh"]

# CMD：默认启动数据库
CMD ["postgres"]
```

**灵活使用**：

```bash
# 启动数据库
docker run mypostgres

# 启动时执行初始化
docker run mypostgres postgres -c shared_buffers=256MB
```

### 场景四：多阶段构建中的应用

```dockerfile
# 构建阶段
FROM golang:1.19 AS builder
WORKDIR /app
COPY . .
RUN go build -o myapp

# 运行阶段
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/myapp .

ENTRYPOINT ["./myapp"]
CMD ["--port", "8080"]
```

### 场景五：开发环境与生产环境区分

```dockerfile
FROM python:3.9

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .

# 开发环境：使用 Flask 开发服务器
CMD ["flask", "run", "--host=0.0.0.0"]

# 生产环境：覆盖 CMD
# docker run myimage gunicorn -w 4 -b 0.0.0.0 app:app
```

## 常见问题和最佳实践

### 常见问题

#### 问题 1：CMD 和 ENTRYPOINT 应该使用哪个？

**回答**：
- 如果容器作为可执行程序，使用 ENTRYPOINT
- 如果只是设置默认命令，使用 CMD
- 大多数情况下，组合使用 ENTRYPOINT + CMD 是最佳选择

#### 问题 2：为什么容器无法优雅关闭？

**回答**：通常是因为使用了 Shell 格式，导致应用不是 PID 1。解决方案：

```dockerfile
# 错误
CMD python app.py

# 正确
CMD ["python", "app.py"]
```

#### 问题 3：如何在 Exec 格式中使用环境变量？

**回答**：

```dockerfile
ENV APP_ENV production
CMD ["sh", "-c", "python app.py --env $APP_ENV"]
```

#### 问题 4：RUN 指令应该合并还是分开？

**回答**：
- 相关命令应该合并，减少镜像层数
- 不相关的命令可以分开，便于利用构建缓存
- 需要在层数和缓存利用之间权衡

#### 问题 5：如何调试 ENTRYPOINT 脚本？

**回答**：

```bash
# 覆盖 ENTRYPOINT 进入容器
docker run --entrypoint /bin/sh -it myimage

# 或者临时覆盖
docker run --rm --entrypoint="" myimage /bin/bash
```

### 最佳实践

#### 1. 优先使用 Exec 格式

```dockerfile
# 推荐
CMD ["nginx", "-g", "daemon off;"]
ENTRYPOINT ["docker-entrypoint.sh"]

# 避免
CMD nginx -g "daemon off;"
```

#### 2. 合理组合 ENTRYPOINT 和 CMD

```dockerfile
# 推荐：ENTRYPOINT 固定主命令，CMD 提供默认参数
ENTRYPOINT ["python"]
CMD ["app.py"]
```

#### 3. 优化 RUN 指令

```dockerfile
# 推荐：合并相关命令，清理缓存
RUN apt-get update && apt-get install -y \
    package1 \
    package2 \
    && rm -rf /var/lib/apt/lists/*
```

#### 4. 使用 .dockerignore 减小构建上下文

```
.git
node_modules
*.log
Dockerfile
```

#### 5. 为 ENTRYPOINT 脚本添加执行权限

```dockerfile
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["docker-entrypoint.sh"]
```

#### 6. 在 ENTRYPOINT 脚本中使用 exec

```bash
#!/bin/sh
set -e

# 初始化逻辑
echo "Initializing..."

# 使用 exec 替换当前进程，使应用成为 PID 1
exec "$@"
```

#### 7. 合理利用构建缓存

```dockerfile
# 将不常变化的层放在前面
FROM node:16
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
CMD ["npm", "start"]
```

## 面试回答

**面试官**：请解释 Dockerfile 中 RUN、CMD、ENTRYPOINT 的区别？

**回答**：

RUN、CMD 和 ENTRYPOINT 是 Dockerfile 中三个关键指令，它们的区别主要体现在执行时机和作用上。

RUN 指令在镜像构建阶段执行，每次执行都会创建一个新的镜像层，主要用于安装软件包、配置环境等构建时操作。CMD 和 ENTRYPOINT 都在容器运行时执行，但行为不同：CMD 设置容器的默认启动命令，可以被 docker run 的参数完全覆盖；ENTRYPOINT 设置容器的主命令，docker run 的参数会追加到 ENTRYPOINT 后而不是替换它，适合将容器作为可执行程序。

从格式上看，三者都支持 Shell 格式和 Exec 格式。Exec 格式是推荐的方式，因为它让应用直接作为 PID 1 运行，能够正确处理信号，实现优雅关闭。Shell 格式会通过 /bin/sh -c 执行，导致应用不是 PID 1，信号会被 shell 拦截。

实际应用中，最佳实践是组合使用 ENTRYPOINT 和 CMD：ENTRYPOINT 固定主命令，CMD 提供默认参数。例如数据库镜像通常使用 ENTRYPOINT 指定初始化脚本，CMD 指定默认的数据库启动命令。这样既保证了初始化逻辑的执行，又允许用户灵活覆盖启动参数。RUN 则应该合并相关命令以减少镜像层数，并在安装后清理缓存以减小镜像体积。

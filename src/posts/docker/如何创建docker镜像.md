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

# 如何创建Docker镜像

## 1. 引言和概述

Docker镜像是Docker容器的基础，它包含了运行应用程序所需的所有内容：代码、运行时环境、系统工具、库和配置文件等。创建高质量的Docker镜像对于确保应用程序的一致性、可移植性和安全性至关重要。

### 1.1 为什么需要Docker镜像？

- **一致性**：确保应用程序在不同环境中运行相同的代码和依赖
- **可移植性**：可以在任何支持Docker的平台上运行
- **隔离性**：容器之间相互隔离，提高安全性
- **版本控制**：可以对镜像进行版本管理和回滚
- **资源效率**：利用容器的轻量级特性，提高资源利用率

### 1.2 创建Docker镜像的方法

创建Docker镜像主要有两种方法：

1. **使用Dockerfile**：通过编写Dockerfile脚本，自动化构建镜像（推荐）
2. **基于现有容器创建**：通过修改现有容器，然后将其保存为新镜像

在本文中，我们将重点介绍第一种方法，即使用Dockerfile创建Docker镜像，这是最常用和推荐的方式。

## 2. Docker镜像的基本概念

### 2.1 什么是Docker镜像？

Docker镜像是一个轻量级、可执行的独立软件包，包含了运行某个软件所需的所有内容：

- **代码**：应用程序的源代码或二进制文件
- **运行时环境**：如JRE、Node.js运行时等
- **系统工具**：如shell、curl、vim等
- **系统库**：如glibc、OpenSSL等
- **配置文件**：如环境变量、配置文件等

镜像采用分层设计，每一层代表一个指令的结果，这种设计使得镜像可以共享和复用，提高了存储效率和构建速度。

### 2.2 Docker镜像的分层结构

Docker镜像由一系列只读层（Layers）组成，每一层对应Dockerfile中的一条指令。当你基于一个镜像创建容器时，Docker会在这些只读层之上添加一个可写层（Container Layer）。

- **只读层**：来自镜像，不可修改
- **可写层**：容器运行时创建，用于存储容器运行过程中的修改

分层结构的优势：
- **共享层**：不同镜像可以共享相同的基础层，节省存储空间
- **增量更新**：只需要更新修改的层，提高构建和传输效率
- **可追溯性**：每一层都有唯一的ID，可以精确追踪镜像的构建过程

### 2.3 镜像仓库和仓库

镜像仓库（Registry）是存储和分发Docker镜像的服务，而仓库（Repository）是存储同一类型镜像的集合。

- **Docker Hub**：Docker官方的公共镜像仓库
- **私有仓库**：企业或个人搭建的私有镜像存储服务
- **镜像标签**：用于标识镜像的不同版本，格式为`仓库名:标签`（如`nginx:1.21.6`）

### 2.4 镜像与容器的关系

容器是镜像的运行实例，一个镜像可以创建多个容器。容器包含了镜像的所有内容，再加上一个可写层和容器的运行环境。

```
Docker镜像（只读） + 容器层（可写） = Docker容器（运行中）
```

## 3. Dockerfile的编写规则

### 3.1 什么是Dockerfile？

Dockerfile是一个文本文件，包含了一系列用于构建Docker镜像的指令。Docker通过读取Dockerfile中的指令，自动构建出符合要求的镜像。

### 3.2 Dockerfile的基本结构

Dockerfile的基本结构包括：

- **注释**：以`#`开头的行
- **指令**：每条指令都以大写字母开头，后跟参数
- **构建上下文**：执行`docker build`命令时的当前目录

### 3.3 常用Dockerfile指令

#### 3.3.1 FROM

`FROM`指令指定基础镜像，必须是Dockerfile的第一条指令。

```dockerfile
# 使用官方Ubuntu作为基础镜像
FROM ubuntu:20.04

# 使用Alpine作为基础镜像（轻量级）
FROM alpine:3.15
```

#### 3.3.2 RUN

`RUN`指令在镜像构建过程中执行命令，用于安装软件包、配置环境等。

```dockerfile
# 安装nginx
RUN apt-get update && apt-get install -y nginx

# 执行多个命令
RUN apt-get update && \    apt-get install -y nginx && \    apt-get clean && \    rm -rf /var/lib/apt/lists/*
```

#### 3.3.3 COPY

`COPY`指令将文件或目录从构建上下文复制到镜像中。

```dockerfile
# 复制单个文件
COPY index.html /var/www/html/

# 复制整个目录
COPY app/ /app/

# 带通配符的复制
COPY *.js /app/
```

#### 3.3.4 ADD

`ADD`指令与`COPY`类似，但具有额外功能：
- 自动解压缩压缩文件
- 支持从URL下载文件

```dockerfile
# 复制并解压缩tar文件
ADD app.tar.gz /app/

# 从URL下载文件
ADD https://example.com/file.txt /tmp/
```

#### 3.3.5 CMD

`CMD`指令指定容器启动时默认执行的命令。

```dockerfile
# 启动nginx服务
CMD ["nginx", "-g", "daemon off;"]

# 使用shell形式
CMD nginx -g 'daemon off;'
```

#### 3.3.6 ENTRYPOINT

`ENTRYPOINT`指令指定容器的入口点，与CMD类似，但不可被`docker run`命令行参数覆盖。

```dockerfile
# 设置入口点
ENTRYPOINT ["nginx"]

# 结合CMD使用
ENTRYPOINT ["nginx"]
CMD ["-g", "daemon off;"]
```

#### 3.3.7 EXPOSE

`EXPOSE`指令声明容器运行时监听的端口。

```dockerfile
# 暴露80端口
EXPOSE 80

# 暴露多个端口
EXPOSE 80 443
```

#### 3.3.8 ENV

`ENV`指令设置环境变量。

```dockerfile
# 设置单个环境变量
ENV APP_HOME /app

# 设置多个环境变量
ENV DB_HOST=localhost DB_PORT=3306
```

#### 3.3.9 WORKDIR

`WORKDIR`指令设置工作目录，后续的指令将在该目录下执行。

```dockerfile
# 设置工作目录
WORKDIR /app

# 切换工作目录
WORKDIR /var/www/html
```

#### 3.3.10 VOLUME

`VOLUME`指令创建一个挂载点，用于挂载外部卷。

```dockerfile
# 创建挂载点
VOLUME /data

# 创建多个挂载点
VOLUME ["/data", "/logs"]
```

#### 3.3.11 USER

`USER`指令指定后续命令的执行用户。

```dockerfile
# 使用www-data用户
USER www-data

# 使用指定UID的用户
USER 1000
```

#### 3.3.12 LABEL

`LABEL`指令为镜像添加元数据。

```dockerfile
# 添加单个标签
LABEL maintainer="gaamingzhang@example.com"

# 添加多个标签
LABEL version="1.0.0" description="My Docker image"```

## 4. 镜像构建命令和参数

### 4.1 docker build命令的基本用法

使用`docker build`命令可以从Dockerfile构建Docker镜像：

```bash
# 基本用法
docker build [OPTIONS] PATH | URL | -

# 示例：从当前目录构建镜像
docker build .
```

### 4.2 常用构建参数

#### 4.2.1 -t, --tag

为构建的镜像指定名称和标签：

```bash
# 格式：镜像名:标签
docker build -t myapp:1.0 .

# 同时指定多个标签
docker build -t myapp:1.0 -t myapp:latest .

# 指定仓库和标签
docker build -t registry.example.com/myapp:1.0 .
```

#### 4.2.2 -f, --file

指定Dockerfile的路径：

```bash
# 使用指定的Dockerfile
docker build -f Dockerfile.prod .

# 使用相对路径的Dockerfile
docker build -f ./build/Dockerfile .
```

#### 4.2.3 --build-arg

传递构建时参数：

```bash
# 传递单个构建参数
docker build --build-arg VERSION=1.0 .

# 传递多个构建参数
docker build --build-arg VERSION=1.0 --build-arg ENV=prod .
```

在Dockerfile中使用构建参数：

```dockerfile
FROM node:16
ARG VERSION
ENV APP_VERSION=$VERSION
```

#### 4.2.4 --no-cache

构建时不使用缓存：

```bash
docker build --no-cache -t myapp:1.0 .
```

#### 4.2.5 --pull

构建前总是尝试拉取最新的基础镜像：

```bash
docker build --pull -t myapp:1.0 .
```

#### 4.2.6 --platform

指定构建平台（适用于多平台构建）：

```bash
# 构建x86_64平台的镜像
docker build --platform linux/amd64 -t myapp:1.0 .

# 构建arm64平台的镜像
docker build --platform linux/arm64 -t myapp:1.0 .
```

#### 4.2.7 --target

指定多阶段构建中的目标阶段：

```bash
docker build --target prod -t myapp:1.0 .
```

### 4.3 构建过程详解

当执行`docker build`命令时，Docker会执行以下步骤：

1. **准备构建上下文**：将指定的路径（通常是当前目录）作为构建上下文发送给Docker引擎
2. **解析Dockerfile**：读取并解析Dockerfile中的指令
3. **执行指令**：按照指令顺序执行，每条指令创建一个新的镜像层
4. **构建镜像**：将所有层组合成一个完整的镜像
5. **打标签**：根据-t参数为镜像添加名称和标签

### 4.4 多阶段构建

多阶段构建允许在一个Dockerfile中定义多个构建阶段，从而减小最终镜像的大小：

```dockerfile
# 第一阶段：构建阶段
FROM node:16 AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# 第二阶段：生产阶段
FROM nginx:1.21.6
COPY --from=builder /app/build /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

多阶段构建的优势：
- **减小镜像大小**：只包含运行时所需的文件，不包含构建工具
- **提高安全性**：减少镜像中的漏洞表面积
- **简化构建流程**：在一个Dockerfile中完成所有构建步骤

### 4.5 构建示例

一个完整的Node.js应用构建示例：

```bash
# 1. 创建项目结构
mkdir myapp && cd myapp

# 2. 创建package.json
cat > package.json <<EOF
{
  "name": "myapp",
  "version": "1.0.0",
  "main": "index.js",
  "scripts": {
    "start": "node index.js"
  },
  "dependencies": {
    "express": "^4.17.3"
  }
}
EOF

# 3. 创建index.js
cat > index.js <<EOF
const express = require('express');
const app = express();
const PORT = process.env.PORT || 3000;

app.get('/', (req, res) => {
  res.send('Hello, Docker!');
});

app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});
EOF

# 4. 创建Dockerfile
cat > Dockerfile <<EOF
FROM node:16
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 3000
CMD ["node", "index.js"]
EOF

# 5. 构建镜像
docker build -t myapp:1.0 .
```

## 5. 镜像管理

### 5.1 查看本地镜像

使用`docker images`或`docker image ls`命令可以查看本地的镜像：

```bash
# 基本用法
docker images

# 更详细的信息
docker image ls

# 过滤镜像
docker images nginx

# 显示所有镜像（包括中间层）
docker images -a
```

### 5.2 删除本地镜像

使用`docker rmi`或`docker image rm`命令可以删除本地镜像：

```bash
# 根据镜像名和标签删除
docker rmi myapp:1.0

# 根据镜像ID删除
docker rmi d70eaf7277ea

# 删除多个镜像
docker rmi myapp:1.0 nginx:latest

# 强制删除（即使有容器在使用）
docker rmi -f myapp:1.0

# 删除所有未使用的镜像
docker image prune
```

### 5.3 镜像标签管理

使用`docker tag`命令可以为镜像添加新标签：

```bash
# 为现有镜像添加新标签
docker tag myapp:1.0 myapp:2.0

# 将本地镜像标记为远程仓库镜像
docker tag myapp:1.0 registry.example.com/myapp:1.0
```

### 5.4 镜像导出和导入

#### 5.4.1 导出镜像

使用`docker save`命令可以将镜像导出为文件：

```bash
# 导出单个镜像
docker save -o myapp.tar myapp:1.0

# 导出多个镜像
docker save -o images.tar myapp:1.0 nginx:latest
```

#### 5.4.2 导入镜像

使用`docker load`命令可以从文件导入镜像：

```bash
# 从文件导入镜像
docker load -i myapp.tar
```

### 5.5 镜像上传到仓库

使用`docker push`命令可以将镜像上传到镜像仓库：

```bash
# 登录到镜像仓库
docker login registry.example.com

# 上传镜像
docker push registry.example.com/myapp:1.0

# 上传所有标签
docker push --all-tags registry.example.com/myapp
```

### 5.6 查看镜像详情

使用`docker inspect`命令可以查看镜像的详细信息：

```bash
# 查看镜像详情
docker inspect myapp:1.0

# 只查看特定信息
docker inspect --format='{{.Architecture}}' myapp:1.0
```

### 5.7 清理无用镜像

使用`docker image prune`命令可以清理无用的镜像：

```bash
# 清理悬空镜像（无标签的镜像）
docker image prune

# 清理所有未使用的镜像
docker image prune -a

# 清理特定日期之前的未使用镜像
docker image prune -a --filter "until=2023-01-01"
```

## 6. Dockerfile最佳实践

编写高质量的Dockerfile是创建优秀Docker镜像的关键。以下是一些Dockerfile的最佳实践：

### 6.1 使用最小化基础镜像

选择最小化的基础镜像可以减小镜像大小，提高安全性：

```dockerfile
# 不推荐：使用完整的Ubuntu镜像
FROM ubuntu:20.04

# 推荐：使用Alpine（轻量级）
FROM alpine:3.15

# 对于特定应用，可以使用官方提供的最小化镜像
FROM nginx:alpine
FROM node:16-alpine
FROM python:3.10-slim
```

### 6.2 合理使用分层结构

- **合并RUN指令**：使用`&&`和反斜杠`\`合并多个RUN指令，减少镜像层数
- **将经常变化的指令放在后面**：这样可以利用Docker的缓存机制，提高构建效率

```dockerfile
# 不推荐：每个命令单独一行
RUN apt-get update
RUN apt-get install -y nginx
RUN apt-get clean

# 推荐：合并为一个RUN指令
RUN apt-get update && \    apt-get install -y nginx && \    apt-get clean && \    rm -rf /var/lib/apt/lists/*

# 推荐：将变化频繁的内容放在后面
COPY package.json ./
RUN npm install
COPY . .  # 这行经常变化，放在后面
```

### 6.3 优化构建上下文

- **使用`.dockerignore`文件**：排除不需要的文件和目录，减小构建上下文大小
- **避免COPY或ADD整个目录**：只复制需要的文件

`.dockerignore`文件示例：

```
# 排除node_modules目录
node_modules/

# 排除日志文件
*.log

# 排除版本控制文件
.git/
.gitignore

# 排除构建输出目录
dist/
build/
```

### 6.4 使用多阶段构建

多阶段构建可以显著减小最终镜像的大小：

```dockerfile
# 第一阶段：构建阶段（较大）
FROM golang:1.18 AS builder
WORKDIR /app
COPY . .
RUN go build -o myapp main.go

# 第二阶段：生产阶段（较小）
FROM alpine:3.15
WORKDIR /app
COPY --from=builder /app/myapp .
EXPOSE 8080
CMD ["./myapp"]
```

### 6.5 遵循安全最佳实践

- **不要以root用户运行**：创建并使用普通用户
- **更新软件包**：确保使用最新的安全补丁
- **避免在镜像中存储敏感信息**：使用环境变量或Docker secrets

```dockerfile
# 创建并使用普通用户
FROM alpine:3.15
RUN addgroup -g 1000 myapp && \    adduser -u 1000 -G myapp -D myapp
USER myapp

# 更新软件包
RUN apt-get update && apt-get upgrade -y && ...
```

### 6.6 其他最佳实践

- **使用固定版本的基础镜像**：避免使用`latest`标签，确保构建的可重复性
- **添加元数据标签**：使用LABEL指令添加镜像的元数据
- **使用EXPOSE声明端口**：虽然不自动映射，但可以提供文档
- **使用VOLUME声明数据卷**：提示用户哪些目录应该持久化
- **保持Dockerfile简洁**：只包含必要的指令

```dockerfile
# 使用固定版本
FROM nginx:1.21.6

# 添加元数据
LABEL maintainer="gaamingzhang@example.com"
LABEL version="1.0.0"
LABEL description="My Nginx image"

# 声明端口和卷
EXPOSE 80
VOLUME /var/log/nginx
```

## 7. 常见问题解答

### Q1: 如何减小Docker镜像的大小？

减小Docker镜像大小的方法有：

- **使用最小化基础镜像**：如Alpine或slim版本的镜像
- **使用多阶段构建**：将构建过程和运行环境分离，只保留运行时所需文件
- **合并RUN指令**：减少镜像层数
- **清理临时文件**：在同一个RUN指令中安装软件并清理缓存
- **使用`.dockerignore`文件**：排除不需要的文件和目录
- **移除不必要的依赖**：只安装运行应用所需的软件包

### Q2: Dockerfile中CMD和ENTRYPOINT的区别是什么？

| 特性 | CMD | ENTRYPOINT |
|------|-----|------------|
| 作用 | 指定容器启动时执行的默认命令 | 指定容器的主命令（入口点） |
| 可覆盖性 | 可以被`docker run`命令行参数覆盖 | 不能被直接覆盖，只能通过`--entrypoint`参数修改 |
| 组合使用 | 当与ENTRYPOINT一起使用时，CMD提供默认参数 | 可以与CMD一起使用，CMD提供默认参数 |

**示例：**
```dockerfile
# CMD示例（可被覆盖）
CMD ["nginx", "-g", "daemon off;"]

# ENTRYPOINT示例（不可被直接覆盖）
ENTRYPOINT ["nginx"]

# 组合使用
ENTRYPOINT ["nginx"]
CMD ["-g", "daemon off;"]
```

### Q3: 如何在构建Docker镜像时传递参数？

可以使用`--build-arg`参数在构建时传递参数：

```bash
# 传递单个构建参数
docker build --build-arg VERSION=1.0 .

# 传递多个构建参数
docker build --build-arg VERSION=1.0 --build-arg ENV=prod .
```

然后在Dockerfile中使用`ARG`指令接收这些参数：

```dockerfile
FROM node:16
ARG VERSION
ARG ENV
ENV APP_VERSION=$VERSION
ENV APP_ENV=$ENV
```

### Q4: 什么是多阶段构建？为什么要使用它？

多阶段构建允许在一个Dockerfile中定义多个构建阶段，每个阶段使用不同的基础镜像和指令。

**使用多阶段构建的好处：**

- **减小镜像大小**：最终镜像只包含运行时所需的文件，不包含构建工具和依赖
- **提高安全性**：减少镜像中的漏洞表面积
- **简化构建流程**：在一个文件中完成所有构建步骤，无需维护多个Dockerfile
- **提高构建效率**：可以并行构建不同阶段

**示例：**
```dockerfile
# 构建阶段
FROM node:16 AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# 运行阶段
FROM nginx:alpine
COPY --from=builder /app/build /usr/share/nginx/html
EXPOSE 80
```

### Q5: 如何查看Docker镜像的构建历史？

可以使用`docker history`命令查看镜像的构建历史：

```bash
# 查看镜像构建历史
docker history myapp:1.0

# 查看更详细的信息
docker history --no-trunc myapp:1.0
```

这将显示镜像的每一层以及构建时使用的指令，有助于分析镜像的大小和优化构建过程。

## 8. 总结

本文详细介绍了如何创建Docker镜像，包括以下核心内容：

1. **Docker镜像的基本概念**：镜像的定义、分层结构、镜像与容器的关系、镜像仓库的作用
2. **Dockerfile的编写规则**：常用指令（FROM、RUN、COPY、CMD、ENTRYPOINT等）的使用方法和示例
3. **镜像构建命令和参数**：`docker build`命令的基本用法和常用参数（-t、-f、--build-arg等）
4. **多阶段构建**：通过分离构建和运行环境，减小最终镜像大小的方法
5. **镜像管理**：镜像的查看、删除、标签管理、导出导入、上传到仓库等操作
6. **Dockerfile最佳实践**：使用最小化基础镜像、合理使用分层结构、优化构建上下文等技巧
7. **常见问题解答**：关于镜像大小优化、CMD/ENTRYPOINT区别、构建参数传递等高频问题的解答

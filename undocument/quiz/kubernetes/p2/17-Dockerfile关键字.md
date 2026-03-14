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

# Dockerfile 关键字详解：从基础到实战

## 引言

在容器化技术日益普及的今天，Dockerfile 作为构建 Docker 镜像的核心配置文件，其重要性不言而喻。Dockerfile 定义了镜像构建的每一个步骤，从基础镜像的选择到应用程序的部署，每一行指令都直接影响着最终镜像的质量、大小和性能。

掌握 Dockerfile 关键字不仅是容器化开发的基础技能，更是优化镜像构建、提升部署效率的关键。一个优秀的 Dockerfile 应该具备以下特征：构建过程可重复、镜像体积最小化、构建缓存利用最大化、安全性最佳。本文将深入剖析 Dockerfile 的所有关键字，帮助开发者全面理解每个指令的作用机制和最佳实践。

## Dockerfile 关键字分类概览

Dockerfile 关键字可以按照功能划分为以下几大类：

| 分类 | 关键字 | 主要作用 |
|------|--------|----------|
| 基础指令 | FROM、MAINTAINER、LABEL | 定义镜像基础信息和元数据 |
| 文件操作 | COPY、ADD、WORKDIR | 文件系统操作和目录管理 |
| 执行命令 | RUN、CMD、ENTRYPOINT | 命令执行和容器启动配置 |
| 环境配置 | ENV、ARG、EXPOSE | 环境变量和端口声明 |
| 数据管理 | VOLUME | 数据卷挂载点 |
| 权限控制 | USER | 运行用户权限管理 |
| 健康检查 | HEALTHCHECK | 容器健康状态监控 |
| 多阶段构建 | AS | 构建阶段命名和优化 |

## 一、基础指令

### 1. FROM - 指定基础镜像

**作用**：FROM 是 Dockerfile 的第一条指令，用于指定构建镜像的基础镜像。所有后续指令都基于此镜像进行构建。

**语法**：
```dockerfile
FROM <image> [AS <name>]
FROM <image>[:<tag>] [AS <name>]
FROM <image>[@<digest>] [AS <name>]
```

**示例**：
```dockerfile
# 使用官方 Ubuntu 镜像
FROM ubuntu:20.04

# 使用多阶段构建命名
FROM golang:1.19 AS builder

# 使用 digest 精确指定镜像版本
FROM python@sha256:abc123...
```

**注意事项**：
- FROM 必须是 Dockerfile 的第一条非注释指令
- tag 默认为 latest，生产环境建议明确指定版本
- 多阶段构建时可以使用 AS 关键字命名构建阶段
- 选择基础镜像时应考虑镜像大小和安全性

### 2. MAINTAINER - 维护者信息（已废弃）

**作用**：指定镜像维护者的姓名和联系方式。此指令已被 LABEL 替代，但仍可使用。

**语法**：
```dockerfile
MAINTAINER <name>
```

**示例**：
```dockerfile
MAINTAINER John Doe <john.doe@example.com>
```

**注意事项**：
- 官方已不推荐使用，建议使用 LABEL 代替
- 不会影响镜像构建，仅作为元数据

### 3. LABEL - 镜像元数据标签

**作用**：为镜像添加键值对形式的元数据标签，用于描述镜像的各种信息，如版本、描述、维护者等。

**语法**：
```dockerfile
LABEL <key>=<value> <key>=<value> ...
```

**示例**：
```dockerfile
LABEL maintainer="John Doe <john.doe@example.com>"
LABEL version="1.0.0"
LABEL description="This is a web application container"
LABEL vendor="ACME Corp"
```

**注意事项**：
- 一个 LABEL 指令可以设置多个标签
- 标签值中包含空格时需要使用引号
- 可以通过 `docker inspect` 命令查看镜像标签
- 建议使用 LABEL 替代 MAINTAINER 指令

## 二、文件操作指令

### 4. COPY - 复制文件或目录

**作用**：将构建上下文中的文件或目录复制到镜像中的指定位置。这是最常用的文件操作指令。

**语法**：
```dockerfile
COPY <src>... <dest>
COPY ["<src>",... "<dest>"]  # 路径包含空格时使用
```

**示例**：
```dockerfile
# 复制单个文件
COPY package.json /app/

# 复制目录
COPY src/ /app/src/

# 复制多个文件
COPY file1.txt file2.txt /app/

# 使用通配符
COPY *.json /app/
```

**注意事项**：
- 源路径必须在构建上下文中，不能使用绝对路径
- 目标路径可以是绝对路径或相对于 WORKDIR 的相对路径
- 如果目标路径不存在，会自动创建
- 复制目录时，只复制目录内容，不包含目录本身
- 文件权限默认为 644，目录权限默认为 755

### 5. ADD - 添加文件（高级复制）

**作用**：类似于 COPY，但支持自动解压 tar 文件和 URL 下载功能。

**语法**：
```dockerfile
ADD <src>... <dest>
ADD ["<src>",... "<dest>"]
```

**示例**：
```dockerfile
# 自动解压 tar 文件
ADD application.tar.gz /app/

# 从 URL 下载文件
ADD http://example.com/file.txt /app/

# 复制本地文件
ADD config.json /app/config/
```

**注意事项**：
- 如果源文件是 tar 压缩包，会自动解压到目标目录
- 支持 URL 下载，但不推荐使用（应使用 RUN curl/wget）
- 官方建议优先使用 COPY，仅在需要自动解压时使用 ADD
- URL 下载的文件权限为 600

### 6. WORKDIR - 设置工作目录

**作用**：为后续的 RUN、CMD、ENTRYPOINT、COPY、ADD 指令设置工作目录。

**语法**：
```dockerfile
WORKDIR /path/to/workdir
```

**示例**：
```dockerfile
# 设置工作目录
WORKDIR /app

# 相对路径，基于上一个 WORKDIR
WORKDIR src
# 此时工作目录为 /app/src

# 可以使用环境变量
ENV BASE_DIR=/application
WORKDIR $BASE_DIR
```

**注意事项**：
- WORKDIR 可以在 Dockerfile 中多次使用
- 如果目录不存在，会自动创建
- 建议使用绝对路径，避免混淆
- 后续指令的相对路径都基于 WORKDIR

## 三、执行命令指令

### 7. RUN - 执行命令

**作用**：在镜像构建过程中执行命令，并将结果提交到新的镜像层。这是安装软件包和配置系统的主要方式。

**语法**：
```dockerfile
# Shell 形式
RUN <command>

# Exec 形式
RUN ["executable", "param1", "param2"]
```

**示例**：
```dockerfile
# Shell 形式
RUN apt-get update && apt-get install -y nginx

# 多行命令优化
RUN apt-get update \
    && apt-get install -y \
        nginx \
        curl \
        vim \
    && rm -rf /var/lib/apt/lists/*

# Exec 形式
RUN ["/bin/bash", "-c", "echo hello"]

# 安装 Node.js 应用
RUN npm install --production
```

**注意事项**：
- 每个 RUN 指令都会创建一个新的镜像层
- 应将多个命令合并为一个 RUN 指令，减少镜像层数
- 使用 `&&` 连接命令，确保前一条命令成功才执行下一条
- 安装后清理缓存文件，减小镜像体积
- Exec 形式不会启动 shell，变量替换不会生效

### 8. CMD - 容器启动命令

**作用**：指定容器启动时默认执行的命令。一个 Dockerfile 中只能有一个 CMD 指令，如果有多个，只有最后一个生效。

**语法**：
```dockerfile
# Exec 形式（推荐）
CMD ["executable", "param1", "param2"]

# 作为 ENTRYPOINT 的默认参数
CMD ["param1", "param2"]

# Shell 形式
CMD command param1 param2
```

**示例**：
```dockerfile
# Exec 形式启动 nginx
CMD ["nginx", "-g", "daemon off;"]

# 启动 Python 应用
CMD ["python", "app.py"]

# Shell 形式
CMD nginx -g 'daemon off;'

# 作为 ENTRYPOINT 参数
ENTRYPOINT ["python"]
CMD ["app.py"]
```

**注意事项**：
- CMD 可以被 `docker run` 命令行参数覆盖
- 推荐使用 Exec 形式，避免 shell 处理带来的问题
- Shell 形式会作为 `/bin/sh -c` 的参数执行
- 容器启动后应保持前台运行，避免容器退出

### 9. ENTRYPOINT - 容器入口点

**作用**：配置容器启动时执行的可执行程序，让容器以可执行程序的形式运行。与 CMD 配合使用可以提供更灵活的启动配置。

**语法**：
```dockerfile
# Exec 形式（推荐）
ENTRYPOINT ["executable", "param1", "param2"]

# Shell 形式
ENTRYPOINT command param1 param2
```

**示例**：
```dockerfile
# 定义入口点
ENTRYPOINT ["nginx"]
CMD ["-g", "daemon off;"]

# 启动脚本
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["postgres"]

# 应用程序入口
ENTRYPOINT ["java", "-jar", "app.jar"]
CMD ["--spring.profiles.active=prod"]
```

**注意事项**：
- ENTRYPOINT 的命令不会被 `docker run` 参数覆盖
- `docker run` 的参数会追加到 ENTRYPOINT 命令后
- 可以通过 `--entrypoint` 选项覆盖 ENTRYPOINT
- Exec 形式可以接收 CMD 和运行时参数
- Shell 形式会忽略 CMD 和运行时参数

**CMD 与 ENTRYPOINT 的区别**：

| 特性 | CMD | ENTRYPOINT |
|------|-----|------------|
| 是否可被覆盖 | 可被 docker run 参数覆盖 | 不易被覆盖 |
| 主要用途 | 提供默认参数 | 定义可执行程序 |
| 组合使用 | 作为 ENTRYPOINT 的参数 | 定义主命令 |
| 运行时修改 | 容易 | 需要 --entrypoint |

## 四、环境配置指令

### 10. ENV - 设置环境变量

**作用**：设置环境变量，这些变量在构建阶段和容器运行时都可用。

**语法**：
```dockerfile
ENV <key> <value>
ENV <key>=<value> ...
```

**示例**：
```dockerfile
# 设置单个环境变量
ENV APP_HOME /app

# 设置多个环境变量
ENV NODE_VERSION=18.0.0 \
    NPM_VERSION=9.0.0 \
    APP_ENV=production

# 使用环境变量
WORKDIR $APP_HOME
RUN echo $APP_ENV
```

**注意事项**：
- ENV 设置的环境变量会持久化到镜像中
- 可以在后续指令中使用环境变量
- 使用 `docker run -e` 可以覆盖环境变量
- 一个 ENV 可以设置多个变量
- 环境变量会影响镜像缓存，谨慎使用

### 11. ARG - 构建参数

**作用**：定义构建时的参数，仅在构建过程中可用，不会持久化到镜像中。

**语法**：
```dockerfile
ARG <name>[=<default value>]
```

**示例**：
```dockerfile
# 定义构建参数
ARG VERSION=latest
ARG BUILD_DATE

# 使用构建参数
FROM ubuntu:${VERSION}
LABEL build_date=$BUILD_DATE

# 构建时传递参数
# docker build --build-arg VERSION=20.04 --build-arg BUILD_DATE=2024-01-01 .
```

**注意事项**：
- ARG 只在构建阶段有效，运行时不可用
- 可以通过 `--build-arg` 传递构建参数
- ARG 有默认值，ENV 没有
- Docker 预定义了一些 ARG 变量（如 HTTP_PROXY）
- ARG 变量不会影响镜像缓存（从 Docker 1.13 开始）

**ENV 与 ARG 的区别**：

| 特性 | ENV | ARG |
|------|-----|-----|
| 作用范围 | 构建时 + 运行时 | 仅构建时 |
| 是否持久化 | 是 | 否 |
| 是否可覆盖 | docker run -e | --build-arg |
| 主要用途 | 运行时配置 | 构建时配置 |

### 12. EXPOSE - 声明端口

**作用**：声明容器在运行时监听的端口，主要用于文档说明和容器间通信。

**语法**：
```dockerfile
EXPOSE <port> [<port>/<protocol>...]
```

**示例**：
```dockerfile
# 声明单个端口
EXPOSE 80

# 声明多个端口
EXPOSE 80 443

# 指定协议
EXPOSE 53/udp
EXPOSE 8080/tcp
```

**注意事项**：
- EXPOSE 不会实际发布端口，仅作为声明
- 需要通过 `-p` 或 `-P` 参数实际映射端口
- 主要用于容器间通信和文档说明
- 建议声明应用实际使用的端口
- 可以指定 TCP 或 UDP 协议，默认为 TCP

## 五、数据管理指令

### 13. VOLUME - 定义数据卷

**作用**：创建一个挂载点，用于持久化数据或共享数据。容器运行时会自动将指定目录挂载到宿主机。

**语法**：
```dockerfile
VOLUME ["/data"]
VOLUME /data /log
```

**示例**：
```dockerfile
# 定义单个数据卷
VOLUME ["/var/lib/mysql"]

# 定义多个数据卷
VOLUME ["/data", "/log", "/config"]

# 数据库数据持久化
VOLUME /var/lib/postgresql/data
```

**注意事项**：
- VOLUME 指令后的数据变更不会提交到镜像
- 容器删除后，数据卷中的数据仍然保留
- 可以通过 `docker run -v` 覆盖挂载点
- 数据卷默认挂载到 `/var/lib/docker/volumes/`
- 建议在 VOLUME 之前完成数据初始化

## 六、权限控制指令

### 14. USER - 指定运行用户

**作用**：指定后续 RUN、CMD、ENTRYPOINT 指令的运行用户，提高容器安全性。

**语法**：
```dockerfile
USER <user>[:<group>]
USER <UID>[:<GID>]
```

**示例**：
```dockerfile
# 创建用户并切换
RUN groupadd -r appuser && useradd -r -g appuser appuser
USER appuser

# 使用 UID 和 GID
USER 1000:1000

# 切换用户后运行应用
USER appuser
CMD ["node", "app.js"]
```

**注意事项**：
- 使用非 root 用户运行容器是安全最佳实践
- 用户必须事先存在，否则会报错
- 影响后续所有指令，直到遇到下一个 USER
- 确保用户对所需文件有访问权限
- 可以在 `docker run --user` 中覆盖

## 七、健康检查指令

### 15. HEALTHCHECK - 容器健康检查

**作用**：定义容器健康检查命令，Docker 会定期执行该命令判断容器是否健康。

**语法**：
```dockerfile
HEALTHCHECK [OPTIONS] CMD command
HEALTHCHECK NONE  # 禁用健康检查
```

**参数说明**：
- `--interval=<duration>`：检查间隔，默认 30s
- `--timeout=<duration>`：超时时间，默认 30s
- `--start-period=<duration>`：启动等待时间，默认 0s
- `--retries=<number>`：重试次数，默认 3

**示例**：
```dockerfile
# HTTP 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 数据库健康检查
HEALTHCHECK --interval=10s --timeout=2s \
    CMD pg_isready -U postgres || exit 1

# 禁用基础镜像的健康检查
HEALTHCHECK NONE
```

**注意事项**：
- 命令返回 0 表示健康，返回 1 表示不健康
- 健康状态可通过 `docker ps` 或 `docker inspect` 查看
- 合理设置检查间隔和超时时间
- 健康检查命令应简单快速
- 确保健康检查端点可用

## 八、多阶段构建指令

### 16. AS - 构建阶段命名

**作用**：在多阶段构建中为构建阶段命名，便于后续引用。

**语法**：
```dockerfile
FROM <image> AS <name>
```

**示例**：
```dockerfile
# 构建阶段
FROM golang:1.19 AS builder
WORKDIR /app
COPY . .
RUN go build -o myapp

# 运行阶段
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/myapp .
CMD ["./myapp"]

# 从特定阶段复制文件
FROM node:18 AS build
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
```

**注意事项**：
- 多阶段构建可以大幅减小最终镜像体积
- 未被引用的构建阶段会被忽略
- 可以使用 `COPY --from=<stage>` 从其他阶段复制文件
- 构建阶段名称应具有描述性
- 可以有任意数量的 FROM 指令

## 关键字分类汇总表

| 关键字 | 分类 | 是否必需 | 是否可多次使用 | 主要用途 |
|--------|------|----------|----------------|----------|
| FROM | 基础指令 | 是 | 是（多阶段构建） | 指定基础镜像 |
| MAINTAINER | 基础指令 | 否 | 否 | 维护者信息（已废弃） |
| LABEL | 基础指令 | 否 | 是 | 镜像元数据 |
| COPY | 文件操作 | 否 | 是 | 复制文件 |
| ADD | 文件操作 | 否 | 是 | 添加文件（支持解压和URL） |
| WORKDIR | 文件操作 | 否 | 是 | 设置工作目录 |
| RUN | 执行命令 | 否 | 是 | 构建时执行命令 |
| CMD | 执行命令 | 否 | 否 | 容器启动命令 |
| ENTRYPOINT | 执行命令 | 否 | 否 | 容器入口点 |
| ENV | 环境配置 | 否 | 是 | 环境变量 |
| ARG | 环境配置 | 否 | 是 | 构建参数 |
| EXPOSE | 环境配置 | 否 | 是 | 声明端口 |
| VOLUME | 数据管理 | 否 | 是 | 数据卷 |
| USER | 权限控制 | 否 | 是 | 运行用户 |
| HEALTHCHECK | 健康检查 | 否 | 否 | 健康检查 |
| AS | 多阶段构建 | 否 | 是 | 阶段命名 |

## 常见问题与最佳实践

### 常见问题

**问题1：COPY 和 ADD 有什么区别，应该优先使用哪个？**

答：COPY 和 ADD 都可以复制文件到镜像中，但 ADD 具有两个额外功能：自动解压 tar 文件和从 URL 下载文件。官方推荐优先使用 COPY，因为 COPY 语义更清晰，功能单一。只有在需要自动解压 tar 文件时才使用 ADD。从 URL 下载文件应使用 RUN curl 或 RUN wget，这样可以更好地控制下载过程和错误处理。

**问题2：CMD 和 ENTRYPOINT 的区别是什么？如何选择？**

答：CMD 用于指定容器启动时的默认命令，可以被 `docker run` 参数覆盖。ENTRYPOINT 用于定义容器的可执行程序，`docker run` 参数会追加到 ENTRYPOINT 后面。最佳实践是使用 ENTRYPOINT 定义主命令，使用 CMD 提供默认参数。这样既保证了容器的可执行性，又提供了参数的灵活性。例如，ENTRYPOINT ["nginx"]，CMD ["-g", "daemon off;"]，运行时可以通过 `docker run myimage -t` 传递额外参数。

**问题3：ENV 和 ARG 的区别是什么？何时使用哪个？**

答：ENV 设置的环境变量会持久化到镜像中，在构建时和运行时都可用。ARG 定义的参数仅在构建时可用，不会写入镜像。使用场景：如果需要在容器运行时使用变量（如应用配置），使用 ENV；如果仅在构建时需要变量（如版本号、构建日期），使用 ARG。ARG 可以通过 `--build-arg` 传递，ENV 可以通过 `docker run -e` 覆盖。

**问题4：如何优化 Dockerfile 以减小镜像体积？**

答：优化镜像体积的关键策略包括：使用多阶段构建，只保留运行时必需的文件；选择精简的基础镜像，如 alpine；合并多个 RUN 指令，减少镜像层数；在安装软件后清理缓存和临时文件；使用 .dockerignore 排除不必要的文件；避免安装不必要的依赖包。例如，在 apt-get install 后使用 `rm -rf /var/lib/apt/lists/*` 清理缓存，可以将镜像体积减少几十 MB。

**问题5：为什么要在 Dockerfile 中使用 USER 指令？**

答：默认情况下，容器以 root 用户运行，这存在安全风险。如果容器被攻破，攻击者可能获得宿主机的 root 权限。使用 USER 指令切换到非特权用户运行应用，可以限制攻击者的权限范围，提高容器安全性。最佳实践是：创建专用用户和组，设置必要的文件权限，然后使用 USER 切换用户。这是容器安全加固的重要措施之一。

### 最佳实践

1. **镜像构建原则**
   - 使用明确的基础镜像版本标签，避免使用 latest
   - 合并多个 RUN 指令，减少镜像层数
   - 利用构建缓存，将不常变化的指令放在前面
   - 使用多阶段构建减小最终镜像体积

2. **安全性最佳实践**
   - 使用非 root 用户运行容器
   - 不要在镜像中存储敏感信息
   - 定期更新基础镜像，修复安全漏洞
   - 使用 COPY 而非 ADD，避免意外文件注入

3. **性能优化**
   - 合理使用 .dockerignore 排除无关文件
   - 安装软件后清理缓存和临时文件
   - 使用 alpine 等精简基础镜像
   - 优化命令顺序，最大化利用缓存

4. **可维护性**
   - 添加清晰的注释说明
   - 使用 LABEL 添加元数据信息
   - 保持 Dockerfile 结构清晰有序
   - 使用有意义的构建参数和环境变量名称

## 面试回答

**面试官问：Dockerfile 中都有哪些关键字？各自的作用是什么？**

答：Dockerfile 关键字可以分为几大类。首先是基础指令，FROM 指定基础镜像，是必需的第一条指令；LABEL 用于添加镜像元数据，替代已废弃的 MAINTAINER。其次是文件操作指令，COPY 用于复制文件，ADD 功能类似但支持自动解压 tar 和 URL 下载，WORKDIR 设置工作目录。执行命令类包括 RUN 在构建时执行命令，CMD 指定容器启动默认命令，ENTRYPOINT 定义容器入口点，后两者的区别在于 ENTRYPOINT 不易被覆盖，CMD 可以作为 ENTRYPOINT 的参数。环境配置类有 ENV 设置持久化环境变量，ARG 定义仅构建时可用的参数，EXPOSE 声明端口。此外还有 VOLUME 定义数据卷实现数据持久化，USER 指定运行用户提升安全性，HEALTHCHECK 配置健康检查，AS 用于多阶段构建命名。掌握这些关键字的正确使用，是编写高效、安全、可维护的 Dockerfile 的基础。

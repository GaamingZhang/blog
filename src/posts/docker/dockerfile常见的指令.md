---
date: 2025-07-01
author: Gaaming Zhang
category:
  - docker
tag:
  - docker
  - 已完工
---

# dockerfile常见的指令

## 概述
Dockerfile 通过声明式指令定义镜像构建过程，常见指令分为基础设置、文件操作、运行时配置三类：

**基础指令**：
- `FROM <image>:<tag>`：指定基础镜像，必须是第一条指令；多阶段构建可多次使用，如 `FROM golang:1.20 AS builder`。
- `LABEL <key>=<value>`：添加元数据，如版本、维护者、描述。

**文件与环境**：
- `COPY <src> <dest>`：复制本地文件到镜像，推荐用于源码；支持通配符与 `--chown` 设置权限。
- `ADD <src> <dest>`：类似 COPY 但支持 URL 下载与自动解压 tar，非必要不用（行为不透明）。
- `WORKDIR <path>`：设置工作目录，后续 RUN/CMD/ENTRYPOINT 的执行路径；不存在则自动创建。
- `ENV <key>=<value>`：设置环境变量，构建时与运行时均生效；如 `ENV PATH=/app/bin:$PATH`。
- `ARG <name>=<default>`：构建参数，仅构建时生效，可通过 `--build-arg` 传入；常用于版本号、代理配置。

**运行时配置**：
- `RUN <command>`：构建时执行命令并提交为新层，如安装依赖；推荐合并多条命令减少层数 `RUN apt update && apt install -y curl && rm -rf /var/lib/apt/lists/*`；使用 `\` 换行保持可读性。
- `CMD ["executable","param1"]`：容器启动默认命令，可被 `docker run` 参数覆盖；推荐 exec 格式避免 shell 包装，便于信号传递（如 SIGTERM）；shell 格式 `CMD command param1` 会先启动 shell，不利于信号处理。
- `ENTRYPOINT ["executable"]`：容器启动入口点，不可被 `docker run` 参数覆盖（仅追加参数）；与 CMD 配合使用，ENTRYPOINT 固定可执行文件，CMD 提供默认参数；推荐 exec 格式，shell 格式会覆盖 ENTRYPOINT 行为。
- `EXPOSE <port>[/<protocol>]`：声明容器监听端口与协议（默认 TCP），仅文档化作用，不实际映射；运行时需 `docker run -p host:container` 显式映射。
- `VOLUME ["/data"]`：声明挂载点，运行时自动创建匿名卷或绑定宿主目录；用于持久化数据（如数据库）、共享数据；VOLUME 内文件不会随镜像更新而改变，运行时修改会写入卷而非镜像。
- `USER <user>:<group>`：切换用户身份运行后续指令与容器进程；安全最佳实践，避免以 root 运行容器，减少权限提升风险；需确保用户在镜像中存在（可通过 `RUN useradd` 创建）。
- `HEALTHCHECK [OPTIONS] CMD <command>`：定义健康检查命令，监控容器运行状态；OPTIONS 包括 `--interval=30s`（检查间隔）、`--timeout=3s`（超时时间）、`--retries=3`（失败重试次数）、`--start-period=5s`（启动宽限期）；健康检查失败时容器状态变为 unhealthy，可用于自动重启或负载均衡剔除。
- `ONBUILD <instruction>`：定义触发器指令，当当前镜像被用作其他镜像的基础镜像时执行；用于构建模板，如 Node.js 镜像自动复制 package.json 并安装依赖。
- `STOPSIGNAL <signal>`：指定容器停止时接收的信号，默认 `SIGTERM`；用于优雅关闭应用，如 `STOPSIGNAL SIGINT` 配合 Node.js 的 `process.on('SIGINT', ...)`。
- `SHELL ["executable", "parameters"]`：设置默认 shell，用于 RUN/CMD/ENTRYPOINT 的 shell 格式；默认 Linux 为 `["/bin/sh", "-c"]`，Windows 为 `["cmd", "/S", "/C"]`；可用于指定 bash 等替代 shell。

## 相关高频面试题与简答
- 问：CMD 与 ENTRYPOINT 的区别？如何配合使用？
  答：CMD 可被 `docker run` 参数完全覆盖，ENTRYPOINT 不可覆盖只能追加参数；常见模式 `ENTRYPOINT ["python"]` + `CMD ["app.py"]`，允许覆盖脚本但保持解释器不变。

- 问：COPY 与 ADD 怎么选择？
  答：优先用 COPY，语义明确；ADD 会自动解压 tar 且支持 URL，行为不透明易出错，仅在需要解压时使用；下载文件推荐 `RUN curl` 后删除压缩包减少层体积。

- 问：如何减少 Dockerfile 构建层数与镜像体积？
  答：合并 RUN 命令（用 `&&` 连接）、删除临时文件与缓存（如 `rm -rf /var/lib/apt/lists/*`）、多阶段构建分离编译与运行环境、选用轻量 base 镜像（Alpine/distroless）、使用 .dockerignore 排除无关文件。

- 问：多阶段构建的典型场景？
  答：编译型语言（Go/Java/C++）：第一阶段编译生成二进制，第二阶段仅复制可执行文件到轻量镜像，避免将编译工具链打包进最终镜像，显著减小体积。

- 问：ARG 与 ENV 的区别？
  答：ARG 仅构建时生效，运行时不可见，用于传递构建参数（如版本号）；ENV 构建与运行时均生效，会持久化到镜像；敏感信息不应用 ARG（构建历史可见）或 ENV（环境变量泄露），应用 secret mount。

- 问：如何优化 Docker 镜像缓存利用率？
  答：按变更频率排序指令：先 COPY 依赖文件（package.json/requirements.txt）并 RUN 安装，再 COPY 源码，利用层缓存避免依赖重复安装；使用 `--cache-from` 指定缓存源。

- 问：HEALTHCHECK 失败会导致容器重启吗？如何配置？
  答：默认不会自动重启，需配合 `docker run --restart=on-failure` 或编排工具（如 Kubernetes）的健康检查策略；HEALTHCHECK 主要用于监控容器状态，便于外部系统决策。

- 问：为什么推荐使用非 root 用户运行容器？如何实现？
  答：安全最佳实践，减少容器被攻击后的权限提升风险；实现方式：1）使用基础镜像内置的非 root 用户（如 `USER nobody`）；2）通过 `RUN useradd -m appuser && USER appuser` 创建并切换用户；3）确保应用目录权限正确。

- 问：VOLUME 指令有什么作用？与 `docker run -v` 的区别？
  答：VOLUME 声明容器挂载点，运行时自动创建匿名卷或绑定宿主目录；`docker run -v` 是运行时实际挂载操作，优先级高于 VOLUME 声明；VOLUME 内文件不会随镜像更新而改变。

- 问：多阶段构建中如何共享数据？
  答：使用 `COPY --from=builder` 在阶段间复制文件；可通过命名阶段 `AS <name>` 提高可读性；共享数据需是可复制的文件，如编译产物、配置文件等。

- 问：.dockerignore 文件的作用？如何合理配置？
  答：类似 .gitignore，排除构建上下文无关的文件，减少构建时间与镜像体积；配置原则：排除 node_modules、.git、日志文件、临时文件、IDE 配置（.idea/.vscode）等。

- 问：如何确保 Dockerfile 的安全性？
  答：使用非 root 用户、避免使用最新标签（指定具体版本）、删除敏感信息（如密码、密钥）、定期更新基础镜像、使用多阶段构建减少攻击面、扫描镜像漏洞（如 Trivy）。

## 典型 Dockerfile 示例

```dockerfile
# 多阶段构建 - Go 应用示例
FROM golang:1.20-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o app

# 运行阶段
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /build/app .
USER nobody:nobody
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["./app"]
CMD ["--port=8080"]
```

```dockerfile
# 多阶段构建 - Node.js 应用示例
FROM node:18-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --only=production
COPY . .
RUN npm run build

# 运行阶段
FROM node:18-alpine
WORKDIR /app
COPY --from=builder /app/package*.json ./
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/build ./build
USER node
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=3s \
  CMD node -e "require('http').get('http://localhost:3000/health', (res) => process.exit(res.statusCode !== 200))"
ENTRYPOINT ["npm", "start"]
```

```dockerfile
# Python 应用示例
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
RUN useradd -m appuser && chown -R appuser:appuser /app
USER appuser
EXPOSE 5000
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD python -c "import requests; requests.get('http://localhost:5000/health').raise_for_status()" || exit 1
CMD ["gunicorn", "--bind", "0.0.0.0:5000", "app:app"]
```

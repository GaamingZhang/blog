---
date: 2026-03-12
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Docker
tag:
  - Docker
  - docker-compose
  - YAML
---

# Docker Compose YAML 文件编写完全指南

在容器化应用的开发和部署过程中，Docker Compose 是一个不可或缺的工具。它允许开发者通过一个 YAML 文件定义和运行多容器 Docker 应用。本文将深入探讨 docker-compose.yaml 文件的编写方法，从基础语法到高级配置，帮助开发者掌握这一核心技能。

## YAML 文件基本语法

YAML（YAML Ain't Markup Language）是一种人类可读的数据序列化格式，它的设计目标是易于阅读和编写。在编写 docker-compose.yaml 文件之前，理解 YAML 的基本语法规则至关重要。

### 核心语法规则

**缩进**：YAML 使用缩进表示层级关系，必须使用空格而非 Tab 键。通常每个缩进层级使用 2 个空格。缩进的一致性至关重要，同一层级的元素必须保持相同的缩进。

**键值对**：使用冒号和空格分隔键和值。例如：`key: value`。冒号后面必须有一个空格，这是 YAML 语法的强制要求。

**列表**：使用连字符加空格表示列表项。列表项可以包含简单的值，也可以包含复杂的对象结构。

```yaml
fruits:
  - apple
  - banana
  - orange
```

**注释**：使用 `#` 符号添加注释，注释可以独占一行，也可以跟在代码后面。良好的注释习惯能显著提升配置文件的可维护性。

**字符串**：字符串通常不需要引号，但在包含特殊字符或以特殊字符开头时，需要使用单引号或双引号包裹。单引号不解析转义字符，双引号会解析转义字符。

```yaml
name: "Hello\nWorld"  # 双引号会解析转义字符
path: 'C:\Users'      # 单引号保留原始内容
```

**多行字符串**：使用 `|` 保留换行符，使用 `>` 将换行符转换为空格。这在编写长配置脚本时非常有用。

```yaml
script: |
  echo "Hello"
  echo "World"

description: >
  This is a very long
  description that will
  be joined into one line.
```

## docker-compose.yaml 顶层配置

docker-compose.yaml 文件包含四个顶层配置项：version、services、networks 和 volumes。每个配置项都有特定的作用和配置方式。

### version 配置

version 字段指定 docker-compose 文件的格式版本。不同版本支持不同的功能特性。目前主流使用的是 3.x 版本系列。

```yaml
version: '3.8'
```

版本选择需要考虑 Docker Engine 的版本兼容性。version 3.x 专为 Docker Swarm 模式设计，同时也支持 standalone 模式。version 2.x 主要用于 standalone 模式。在生产环境中，建议使用最新的稳定版本，以获得更好的功能支持和性能优化。

### services 配置

services 是 docker-compose.yaml 文件的核心部分，定义了应用中的各个服务容器。每个服务都是一个独立的容器实例，可以配置镜像、构建方式、网络、存储等各种参数。

```yaml
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"

  database:
    image: mysql:5.7
    environment:
      MYSQL_ROOT_PASSWORD: example
```

### networks 配置

networks 配置定义了服务之间的网络拓扑结构。通过自定义网络，可以实现服务发现、负载均衡和网络隔离等功能。

```yaml
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true
```

### volumes 配置

volumes 配置定义了持久化存储卷。数据卷可以在容器之间共享数据，并且在容器删除后数据仍然保留。

```yaml
volumes:
  db-data:
    driver: local
  app-logs:
    driver: local
```

## services 配置详解

services 配置是 docker-compose.yaml 文件中最复杂的部分，包含了大量的配置选项。理解每个配置项的作用和使用场景，是编写高质量配置文件的关键。

### image 和 build 配置

image 指定服务使用的 Docker 镜像，可以从 Docker Hub 或私有仓库拉取。build 指定从本地 Dockerfile 构建镜像。

```yaml
services:
  # 使用已有镜像
  web:
    image: nginx:1.21
    restart: always

  # 从本地构建
  app:
    build:
      context: ./app
      dockerfile: Dockerfile.prod
      args:
        - BUILD_ENV=production
```

build 配置支持多个参数。context 指定构建上下文路径，dockerfile 指定 Dockerfile 文件名，args 指定构建参数。构建参数可以在 Dockerfile 中使用 ARG 指令接收。

### ports 和 expose 配置

ports 将容器端口映射到主机端口，expose 仅在容器之间暴露端口，不映射到主机。

```yaml
services:
  web:
    ports:
      - "8080:80"      # 主机端口:容器端口
      - "443:443"
      - "3000-3005:3000-3005"  # 端口范围映射

  database:
    expose:
      - "3306"         # 仅暴露给其他服务
```

端口映射的格式为 `HOST:CONTAINER`。可以指定协议类型，如 `80:80/tcp` 或 `53:53/udp`。在生产环境中，需要注意端口冲突和安全性问题。

### volumes 配置

volumes 用于挂载主机目录或命名卷到容器中，实现数据持久化和共享。

```yaml
services:
  web:
    volumes:
      # 命名卷
      - db-data:/var/lib/mysql
      # 主机目录挂载
      - ./app:/usr/src/app
      # 只读挂载
      - ./config:/etc/config:ro
      # 匿名卷
      - /tmp/app-data

volumes:
  db-data:
```

挂载格式为 `SOURCE:TARGET:OPTIONS`。SOURCE 可以是主机路径、命名卷或匿名卷。TARGET 是容器内的挂载路径。OPTIONS 包括 `ro`（只读）、`rw`（读写，默认）等。

### environment 和 env_file 配置

environment 用于设置环境变量，env_file 从文件中读取环境变量。

```yaml
services:
  database:
    environment:
      - MYSQL_ROOT_PASSWORD=root123
      - MYSQL_DATABASE=myapp
      - MYSQL_USER=user
      - MYSQL_PASSWORD=pass123
    env_file:
      - .env
      - .env.local
```

环境变量文件格式为 `KEY=VALUE`，每行一个变量。使用 env_file 可以将敏感信息与配置文件分离，提高安全性。建议将 `.env` 文件添加到 `.gitignore` 中，避免将敏感信息提交到版本控制系统。

### depends_on 配置

depends_on 用于定义服务之间的依赖关系，确保依赖服务先启动。

```yaml
services:
  web:
    depends_on:
      database:
        condition: service_healthy
      redis:
        condition: service_started

  database:
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5
```

depends_on 支持三种条件：`service_started`（服务启动）、`service_healthy`（健康检查通过）、`service_completed_successfully`（服务成功完成）。结合 healthcheck 配置，可以实现更精确的启动顺序控制。

### networks 配置

networks 将服务连接到一个或多个网络，实现网络隔离和服务发现。

```yaml
services:
  web:
    networks:
      - frontend
      - backend

  database:
    networks:
      - backend

networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true  # 内部网络，无法访问外部
```

服务连接到同一网络后，可以通过服务名称互相访问。Docker 内置 DNS 服务器会自动解析服务名称到容器 IP 地址。internal 网络可以限制容器访问外部网络，提高安全性。

### restart 配置

restart 配置定义容器的重启策略，确保服务的高可用性。

```yaml
services:
  web:
    restart: always      # 总是重启
  database:
    restart: on-failure  # 失败时重启
  worker:
    restart: unless-stopped  # 除非手动停止，否则总是重启
```

三种重启策略各有适用场景。`always` 适用于关键服务，`on-failure` 适用于可能失败但需要自动恢复的服务，`unless-stopped` 适用于需要手动控制的服务。

### healthcheck 配置

healthcheck 定义容器的健康检查机制，用于判断容器是否正常运行。

```yaml
services:
  web:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:80/health"]
      interval: 30s      # 检查间隔
      timeout: 10s       # 超时时间
      retries: 3         # 重试次数
      start_period: 40s  # 启动等待时间
```

健康检查对于服务的可靠性和负载均衡至关重要。interval 指定两次检查之间的间隔，timeout 指定单次检查的超时时间，retries 指定连续失败多少次后认为容器不健康，start_period 指定容器启动后开始健康检查的等待时间。

### command 和 entrypoint 配置

command 覆盖容器的默认命令，entrypoint 覆盖容器的默认入口点。

```yaml
services:
  web:
    image: nginx
    command: ["nginx", "-g", "daemon off;"]

  app:
    image: python:3.9
    entrypoint: ["python"]
    command: ["app.py"]
```

command 可以使用字符串或数组格式。数组格式更安全，不会被 shell 解析。entrypoint 通常用于设置可执行文件，command 用于传递参数。

## 网络配置详解

Docker Compose 的网络配置提供了灵活的网络管理能力，支持多种网络驱动和配置选项。

### 网络驱动类型

**bridge 驱动**：默认驱动，适用于单主机环境。容器可以通过服务名称互相访问，网络隔离性较好。

**overlay 驱动**：用于 Docker Swarm 集群，支持跨主机的容器通信。在 Swarm 模式下，可以创建覆盖网络实现服务的负载均衡和服务发现。

**host 驱动**：容器直接使用主机网络，性能最高但隔离性最差。适用于对网络性能要求极高的场景。

**macvlan 驱动**：为容器分配 MAC 地址，使其在网络上显示为物理设备。适用于需要容器直接接入物理网络的场景。

```yaml
networks:
  frontend:
    driver: bridge
    ipam:
      driver: default
      config:
        - subnet: 172.20.0.0/16
          gateway: 172.20.0.1

  backend:
    driver: overlay
    attachable: true  # 允许独立容器连接

  hostnet:
    driver: host
```

### 网络配置参数

**ipam 配置**：自定义 IP 地址管理，可以指定子网、网关和 IP 范围。

**internal 配置**：创建内部网络，容器无法访问外部网络，提高安全性。

**attachable 配置**：允许独立容器连接到 overlay 网络，主要用于调试和管理。

**aliases 配置**：为服务在网络中设置别名，一个服务可以有多个别名。

```yaml
services:
  web:
    networks:
      frontend:
        aliases:
          - app.local
          - webapp.local

networks:
  frontend:
    driver: bridge
```

## 数据卷配置详解

数据卷是 Docker 中数据持久化的核心机制，理解其工作原理对构建可靠的应用至关重要。

### 数据卷类型

**命名卷（Named Volumes）**：由 Docker 管理，存储在 Docker 的数据目录中。适合持久化应用数据，如数据库文件、日志等。

**绑定挂载（Bind Mounts）**：将主机目录挂载到容器中。适合开发环境，可以实现代码热更新。

**tmpfs 挂载**：将数据存储在内存中，容器停止后数据消失。适合存储临时数据和敏感信息。

```yaml
services:
  database:
    volumes:
      - db-data:/var/lib/mysql      # 命名卷
      - ./init:/docker-entrypoint-initdb.d  # 绑定挂载
    tmpfs:
      - /tmp                        # tmpfs 挂载
      - /run

volumes:
  db-data:
    driver: local
    driver_opts:
      type: none
      device: /data/mysql
      o: bind
```

### 卷驱动配置

Docker 支持多种卷驱动，可以将数据存储在不同的后端存储系统中。

```yaml
volumes:
  db-data:
    driver: local    # 本地存储

  nfs-data:
    driver: local
    driver_opts:
      type: nfs
      o: addr=192.168.1.100,rw
      device: ":/export/data"

  cloud-data:
    driver: rexray/s3fs  # S3 存储
```

## 完整示例：Web 应用栈

以下是一个完整的 Web 应用栈配置，包含前端、后端、数据库、缓存和反向代理。

```yaml
version: '3.8'

services:
  # 反向代理
  nginx:
    image: nginx:1.21-alpine
    container_name: nginx-proxy
    restart: always
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./ssl:/etc/nginx/ssl:ro
      - nginx-logs:/var/log/nginx
    networks:
      - frontend
    depends_on:
      - frontend
      - backend
    healthcheck:
      test: ["CMD", "nginx", "-t"]
      interval: 30s
      timeout: 10s
      retries: 3

  # 前端应用
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
      args:
        - NODE_ENV=production
    container_name: frontend-app
    restart: unless-stopped
    environment:
      - NODE_ENV=production
      - API_URL=http://backend:8000
    networks:
      - frontend
    depends_on:
      backend:
        condition: service_healthy

  # 后端 API
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: backend-api
    restart: unless-stopped
    environment:
      - DATABASE_URL=postgresql://user:pass@database:5432/myapp
      - REDIS_URL=redis://cache:6379
      - SECRET_KEY=${SECRET_KEY}
    volumes:
      - ./backend/app:/app
      - app-logs:/var/log/app
    networks:
      - frontend
      - backend
    depends_on:
      database:
        condition: service_healthy
      cache:
        condition: service_started
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  # PostgreSQL 数据库
  database:
    image: postgres:13-alpine
    container_name: postgres-db
    restart: always
    environment:
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
      - POSTGRES_DB=myapp
    volumes:
      - db-data:/var/lib/postgresql/data
      - ./init-scripts:/docker-entrypoint-initdb.d
    networks:
      - backend
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user -d myapp"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis 缓存
  cache:
    image: redis:6-alpine
    container_name: redis-cache
    restart: always
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data
    networks:
      - backend
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3

networks:
  frontend:
    driver: bridge
    ipam:
      driver: default
      config:
        - subnet: 172.20.0.0/16
  backend:
    driver: bridge
    internal: true

volumes:
  db-data:
    driver: local
  redis-data:
    driver: local
  app-logs:
    driver: local
  nginx-logs:
    driver: local
```

这个配置展示了一个典型的三层架构 Web 应用。前端和后端通过 frontend 网络通信，后端和数据库通过 backend 网络通信。数据库网络设置为 internal，确保数据库无法直接访问外部网络，提高了安全性。

## 核心配置参数速查表

### 服务配置参数

| 参数 | 说明 | 示例 |
|------|------|------|
| image | 指定镜像 | `image: nginx:latest` |
| build | 构建配置 | `build: ./app` |
| ports | 端口映射 | `ports: ["8080:80"]` |
| expose | 暴露端口 | `expose: ["3306"]` |
| volumes | 数据卷挂载 | `volumes: ["./data:/data"]` |
| environment | 环境变量 | `environment: ["DEBUG=true"]` |
| env_file | 环境变量文件 | `env_file: .env` |
| networks | 网络配置 | `networks: [frontend]` |
| depends_on | 服务依赖 | `depends_on: [db]` |
| restart | 重启策略 | `restart: always` |
| healthcheck | 健康检查 | `healthcheck: {...}` |
| command | 覆盖命令 | `command: npm start` |
| entrypoint | 覆盖入口点 | `entrypoint: [python]` |
| container_name | 容器名称 | `container_name: my-app` |
| hostname | 主机名 | `hostname: web-server` |
| labels | 元数据标签 | `labels: [env=prod]` |
| logging | 日志配置 | `logging: {...}` |
| deploy | 部署配置 | `deploy: {...}` |
| secrets | 敏感数据 | `secrets: [db_pass]` |
| configs | 配置文件 | `configs: [app_config]` |

### 网络配置参数

| 参数 | 说明 | 示例 |
|------|------|------|
| driver | 网络驱动 | `driver: bridge` |
| ipam | IP 管理 | `ipam: {...}` |
| internal | 内部网络 | `internal: true` |
| attachable | 可附加 | `attachable: true` |
| enable_ipv6 | 启用 IPv6 | `enable_ipv6: true` |
| labels | 标签 | `labels: [net=prod]` |

### 数据卷配置参数

| 参数 | 说明 | 示例 |
|------|------|------|
| driver | 卷驱动 | `driver: local` |
| driver_opts | 驱动选项 | `driver_opts: {...}` |
| external | 外部卷 | `external: true` |
| labels | 标签 | `labels: [vol=data]` |

## 常见问题与最佳实践

### 问题 1：如何处理敏感信息？

**解决方案**：不要将密码、密钥等敏感信息直接写入配置文件。使用环境变量文件（.env）或 Docker Secrets 管理敏感信息。

```yaml
services:
  database:
    environment:
      - MYSQL_ROOT_PASSWORD_FILE=/run/secrets/db_password
    secrets:
      - db_password

secrets:
  db_password:
    file: ./secrets/db_password.txt
```

### 问题 2：如何实现容器间的服务发现？

**解决方案**：Docker Compose 自动为服务创建 DNS 解析。连接到同一网络的服务可以通过服务名称互相访问。

```yaml
services:
  backend:
    environment:
      - DB_HOST=database  # 使用服务名称
      - REDIS_HOST=cache
```

### 问题 3：如何优化构建速度？

**解决方案**：合理利用 Docker 构建缓存，将不常变化的层放在前面。使用多阶段构建减小镜像体积。

```yaml
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
      cache_from:
        - myapp:latest
```

### 问题 4：如何处理数据持久化？

**解决方案**：为需要持久化的数据创建命名卷。定期备份数据卷，使用卷驱动支持分布式存储。

```yaml
volumes:
  db-data:
    driver: local
    driver_opts:
      type: nfs
      o: addr=nfs-server,rw
      device: ":/export/db"
```

### 问题 5：如何实现零停机部署？

**解决方案**：使用 Docker Swarm 或 Kubernetes 进行容器编排。配置健康检查和滚动更新策略。

```yaml
services:
  web:
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
        failure_action: rollback
      healthcheck:
        test: ["CMD", "curl", "-f", "http://localhost/health"]
```

### 最佳实践总结

**版本控制**：将 docker-compose.yaml 文件纳入版本控制，但不要提交包含敏感信息的 .env 文件。

**命名规范**：为服务、网络和数据卷使用有意义的名称，便于管理和维护。

**资源限制**：为服务设置资源限制，防止某个服务占用过多资源影响其他服务。

```yaml
services:
  app:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
```

**日志管理**：配置日志驱动和日志轮转，避免日志文件占用过多磁盘空间。

```yaml
services:
  app:
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
```

**网络隔离**：使用多个网络实现服务隔离，前端服务、后端服务和数据库服务分别处于不同网络。

**健康检查**：为关键服务配置健康检查，确保服务的可用性和可靠性。

**文档注释**：在配置文件中添加注释，说明配置的目的和注意事项，提高可维护性。

## 面试回答

**面试官**：会编写 docker-compose 的 YAML 文件吗？如何编写？

**回答**：是的，我非常熟悉 docker-compose.yaml 文件的编写。docker-compose.yaml 文件主要包含四个顶层配置项：version、services、networks 和 volumes。version 指定配置文件的格式版本，目前主流使用 3.8 版本。services 是核心部分，定义应用中的各个服务容器，每个服务可以配置镜像、构建方式、端口映射、数据卷挂载、环境变量、网络连接、依赖关系、重启策略和健康检查等参数。networks 定义服务之间的网络拓扑，支持 bridge、overlay 等多种驱动类型，可以实现服务发现和网络隔离。volumes 定义持久化存储卷，支持命名卷、绑定挂载和 tmpfs 挂载，确保容器删除后数据不丢失。

编写时需要注意 YAML 语法的正确性，使用空格缩进而非 Tab 键，键值对冒号后必须有空格。对于敏感信息如密码和密钥，应该使用环境变量文件或 Docker Secrets 管理，不要直接写入配置文件。生产环境建议配置健康检查、资源限制和日志轮转，使用多网络实现服务隔离，为关键数据配置持久化存储。通过合理的配置，可以实现应用的高可用性、可扩展性和安全性，大大简化多容器应用的部署和管理流程。

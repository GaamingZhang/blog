---
date: 2026-01-06
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Docker
tag:
  - Docker
---

# Docker Compose 文件结构

## 基本概念

Docker Compose 是 Docker 官方提供的用于定义和运行多容器 Docker 应用程序的工具。通过 Compose 文件（通常命名为 `docker-compose.yml`），用户可以使用 YAML 格式定义应用程序的服务、网络和存储卷等配置，然后通过单一命令启动、停止和管理整个应用程序栈。

### 核心优势
- **简化配置**：使用 YAML 文件统一管理多容器应用的所有配置
- **一键部署**：通过 `docker-compose up` 命令快速部署整个应用栈
- **环境一致性**：确保开发、测试和生产环境的配置一致性
- **版本控制**：Compose 文件可以纳入版本控制系统，便于团队协作和版本管理
- **扩展灵活**：支持通过环境变量和配置文件进行灵活的配置管理

## 文件格式与命名规范

### 文件格式
Docker Compose 文件使用 YAML 格式编写，需要遵循 YAML 的语法规则：
- 使用缩进表示层级关系（推荐使用 2 个空格）
- 使用 `#` 表示注释
- 使用键值对 `key: value` 表示配置项
- 使用短横线 `-` 表示列表项

### 命名规范
- **默认文件名**：`docker-compose.yml`（推荐使用）
- **替代文件名**：`docker-compose.yaml`（与默认文件名等效）
- **环境特定文件**：
  - `docker-compose.override.yml`：默认覆盖文件
  - `docker-compose.prod.yml`：生产环境配置
  - `docker-compose.dev.yml`：开发环境配置

### 文件版本
Compose 文件支持多个版本，不同版本对应不同的 Docker Engine 版本要求：

| Compose 文件版本 | Docker Engine 版本要求 |
|----------------|----------------------|
| 3.x            | 17.06.0+             |
| 2.4            | 17.12.0+             |
| 2.3            | 17.06.0+             |
| 2.2            | 1.13.0+              |
| 2.1            | 1.12.0+              |
| 2.0            | 1.10.0+              |
| 1.x            | 1.9.1+               |

**注意**：建议使用最新的 3.x 版本，以获得最完整的功能支持。

## 核心结构

Docker Compose 文件的核心结构由以下几个顶级配置部分组成：

```yaml
# Compose 文件版本
version: '3.8'

# 服务定义
services:
  # 服务名称
  service_name:
    # 服务配置
    build: .
    image: service_image
    ...

# 网络配置
networks:
  # 网络名称
  network_name:
    # 网络配置
    driver: bridge
    ...

# 存储卷配置
volumes:
  # 卷名称
  volume_name:
    # 卷配置
    driver: local
    ...

# 配置文件
configs:
  # 配置名称
  config_name:
    # 配置内容
    file: ./config.ini
    ...

# 机密信息
secrets:
  # 机密名称
  secret_name:
    # 机密内容
    file: ./secret.txt
    ...
```

### Services（核心配置）
`services` 是 Compose 文件中最重要的部分，用于定义应用程序的各个服务容器。每个服务可以包含以下配置项：

#### 镜像与构建
```yaml
services:
  web:
    # 使用已存在的镜像
    image: nginx:latest
    
    # 或从 Dockerfile 构建镜像
    build:
      context: .
      dockerfile: Dockerfile.prod
      args:
        - BUILD_ARG=value
```

#### 容器配置
```yaml
services:
  web:
    # 容器名称
    container_name: my_nginx
    
    # 重启策略
    restart: always  # always, on-failure, unless-stopped, no
    
    # 容器命令
    command: ["nginx", "-g", "daemon off;"]
    
    # 入口点
    entrypoint: ["/app/entrypoint.sh"]
    
    # 工作目录
    working_dir: /app
```

#### 网络配置
```yaml
services:
  web:
    # 端口映射
    ports:
      - "80:80"  # 主机端口:容器端口
      - "443:443"
      - "127.0.0.1:8080:80"  # 指定主机 IP
    
    # 网络连接
    networks:
      - frontend
      - backend
    
    # 主机名
    hostname: web-server
    
    # 域名解析
    extra_hosts:
      - "host.docker.internal:host-gateway"
      - "api.example.com:192.168.1.100"
```

#### 存储配置
```yaml
services:
  web:
    # 存储卷挂载
    volumes:
      - ./html:/usr/share/nginx/html  # 主机目录:容器目录
      - nginx_config:/etc/nginx/conf.d  # 命名卷:容器目录
      - type: bind
        source: ./logs
        target: /var/log/nginx
      - type: volume
        source: nginx_data
        target: /data
        read_only: true
    
    # 临时文件系统
    tmpfs:
      - /run
      - /tmp
```

#### 环境配置
```yaml
services:
  web:
    # 环境变量
    environment:
      - DEBUG=True
      - DATABASE_URL=postgres://user:password@db:5432/dbname
    
    # 环境变量文件
    env_file:
      - .env
      - .env.web
    
    # 容器标签
    labels:
      - "com.example.description=Web Server"
      - "com.example.version=1.0"
```

#### 资源限制
```yaml
services:
  web:
    # 资源限制
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
    
    # 容器特权模式
    privileged: true
    
    # 系统能力
    cap_add:
      - NET_ADMIN
      - SYS_TIME
    cap_drop:
      - ALL
```

#### 依赖关系
```yaml
services:
  web:
    # 依赖服务
    depends_on:
      - db
      - cache
    
    # 健康检查
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### Networks
`networks` 部分用于定义应用程序使用的网络，支持自定义网络驱动和配置：

```yaml
networks:
  # 自定义网络
  frontend:
    driver: bridge
    ipam:
      config:
        - subnet: 172.16.238.0/24
          gateway: 172.16.238.1
    
  # 外部网络（已存在的网络）
  backend:
    external:
      name: my_existing_network
    
  # 内部网络（仅容器间可见）
  internal_network:
    internal: true
    driver: overlay
```

### Volumes
`volumes` 部分用于定义应用程序使用的存储卷，支持持久化数据存储：

```yaml
volumes:
  # 命名卷
  db_data:
    driver: local
    driver_opts:
      type: "none"
      o: "bind"
      device: "/path/to/data"
    
  # 外部卷（已存在的卷）
  redis_cache:
    external:
      name: my_existing_volume
    
  # 临时卷
  temp_files:
    driver: local
    labels:
      - "com.example.description=Temporary Files"
```

### Configs
`configs` 部分用于定义可在服务间共享的配置文件（Docker Swarm 模式下使用）：

```yaml
configs:
  # 从文件加载配置
  app_config:
    file: ./app/config.ini
  
  # 内联配置
  db_config:
    content: |
      [database]
      host=db
      port=5432
      user=admin
      password=secret
  
  # 外部配置（已存在的配置）
  external_config:
    external: true
    name: my_existing_config
```

### Secrets
`secrets` 部分用于定义敏感数据，如密码、API 密钥等（Docker Swarm 模式下使用）：

```yaml
secrets:
  # 从文件加载机密
  db_password:
    file: ./secrets/db_password.txt
  
  # 内联机密（不推荐用于生产环境）
  api_key:
    content: "my_super_secret_api_key"
  
  # 外部机密（已存在的机密）
  external_secret:
    external: true
    name: my_existing_secret
```

## 完整示例

以下是一个包含 Web 服务、数据库和缓存的完整 Compose 文件示例：

```yaml
version: '3.8'

# 服务定义
services:
  # Web 服务
  web:
    image: nginx:1.21-alpine
    container_name: web_server
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./html:/usr/share/nginx/html
      - nginx_logs:/var/log/nginx
    depends_on:
      - app
    networks:
      - frontend
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/"]
      interval: 30s
      timeout: 10s
      retries: 3

  # 应用服务
  app:
    build:
      context: ./app
      dockerfile: Dockerfile
    container_name: application
    environment:
      - DEBUG=False
      - DATABASE_URL=postgres://app_user:app_password@db:5432/app_db
      - REDIS_URL=redis://cache:6379/0
    volumes:
      - ./app:/app
      - app_data:/app/data
    depends_on:
      - db
      - cache
    networks:
      - frontend
      - backend
    restart: unless-stopped
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M

  # 数据库服务
  db:
    image: postgres:14-alpine
    container_name: database
    environment:
      - POSTGRES_USER=app_user
      - POSTGRES_PASSWORD=app_password
      - POSTGRES_DB=app_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    networks:
      - backend
    restart: unless-stopped
    ports:
      - "5432:5432"

  # 缓存服务
  cache:
    image: redis:6-alpine
    container_name: redis_cache
    volumes:
      - redis_data:/data
    networks:
      - backend
    restart: unless-stopped
    ports:
      - "6379:6379"

# 网络配置
networks:
  frontend:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
  backend:
    driver: bridge
    internal: true
    ipam:
      config:
        - subnet: 172.21.0.0/16

# 存储卷配置
volumes:
  nginx_logs:
    driver: local
  app_data:
    driver: local
  postgres_data:
    driver: local
  redis_data:
    driver: local
```

## 最佳实践

### 文件组织
- 将 Compose 文件与应用代码分开存储
- 使用 `.env` 文件管理环境变量，避免在 Compose 文件中硬编码敏感信息
- 使用版本控制系统管理 Compose 文件和配置

### 命名规范
- 使用有意义的服务、网络和卷名称
- 遵循小写字母和下划线的命名约定
- 为容器添加描述性标签

### 配置管理
- 使用 `extends` 关键字复用配置（Compose 1.x 版本）或使用多个 Compose 文件
- 使用 `docker-compose.override.yml` 进行本地开发覆盖
- 为不同环境创建专门的 Compose 文件（如 `docker-compose.prod.yml`）

### 资源管理
- 为每个服务设置合理的资源限制和保留
- 使用 `depends_on` 定义服务启动顺序
- 配置健康检查以确保服务正常运行

### 安全性
- 避免在 Compose 文件中硬编码敏感信息
- 使用 `secrets` 管理敏感数据（Docker Swarm 模式）
- 限制容器的系统权限，仅授予必要的能力
- 使用非 root 用户运行容器

## 常见问题

### Compose 文件版本如何选择？
选择 Compose 文件版本时，应考虑以下因素：
- **Docker Engine 版本**：确保 Compose 文件版本与 Docker Engine 版本兼容
- **功能需求**：新版本通常支持更多功能，如健康检查、资源限制等
- **兼容性**：如果需要在多个环境中运行，选择兼容性较好的版本

建议使用最新的 3.x 版本，以获得最完整的功能支持。

### 如何在不同环境中使用不同的配置？
可以通过以下方式在不同环境中使用不同的配置：
- 使用 `docker-compose.prod.yml`、`docker-compose.dev.yml` 等环境特定文件
- 使用 `docker-compose -f docker-compose.yml -f docker-compose.prod.yml up` 合并多个 Compose 文件
- 使用环境变量和 `.env` 文件进行配置管理

### 如何处理服务间的依赖关系？
使用 `depends_on` 关键字可以定义服务间的依赖关系，但需要注意：
- `depends_on` 仅保证服务的启动顺序，不保证服务的可用性
- 对于需要等待服务完全可用的情况，应使用健康检查或应用程序内的重试机制
- 可以使用 `condition` 选项（Compose 2.x 版本）或自定义脚本进行更复杂的依赖处理

### 如何管理 Compose 文件中的敏感信息？
管理敏感信息的最佳实践包括：
- 使用 `.env` 文件存储敏感信息，并将 `.env` 文件添加到 `.gitignore` 中
- 使用 Docker Secrets（Docker Swarm 模式）管理敏感数据
- 避免在 Compose 文件中硬编码密码、API 密钥等敏感信息
- 使用外部密钥管理系统（如 Vault）集成获取敏感信息

### 如何优化 Compose 文件的性能？
优化 Compose 文件性能的建议：
- 为每个服务设置合理的资源限制和保留
- 使用轻量级基础镜像（如 Alpine）
- 合理配置卷挂载，避免不必要的文件共享
- 使用网络隔离，减少服务间的网络通信开销
- 配置适当的健康检查间隔，避免过多的健康检查请求

## 命令参考

### 基本命令
- `docker-compose up`：启动所有服务
- `docker-compose up -d`：后台启动所有服务
- `docker-compose down`：停止并移除所有服务、网络和卷
- `docker-compose ps`：列出所有服务容器
- `docker-compose logs`：查看所有服务日志
- `docker-compose exec <service> <command>`：在指定服务容器中执行命令

### 管理命令
- `docker-compose build`：构建或重新构建服务镜像
- `docker-compose pull`：拉取服务镜像
- `docker-compose push`：推送服务镜像
- `docker-compose restart`：重启所有服务
- `docker-compose stop`：停止所有服务
- `docker-compose start`：启动所有已停止的服务

### 环境变量命令
- `docker-compose config`：验证并显示 Compose 文件配置
- `docker-compose env`：显示环境变量
- `docker-compose run --rm <service> <command>`：运行一次性命令

## 总结

Docker Compose 是一个强大的工具，通过 YAML 文件可以轻松定义和管理多容器 Docker 应用程序。本文详细介绍了 Compose 文件的结构、配置选项和最佳实践，帮助用户快速掌握 Docker Compose 的使用方法。

通过合理的配置和管理，可以充分发挥 Docker Compose 的优势，提高应用程序的部署效率和运维质量。建议用户根据实际需求选择合适的 Compose 文件版本和配置选项，并遵循最佳实践进行应用程序开发和部署。

## 参考资料
- [Docker Compose 官方文档](https://docs.docker.com/compose/)
- [Docker Compose 文件参考](https://docs.docker.com/compose/compose-file/)
- [YAML 官方规范](https://yaml.org/spec/)

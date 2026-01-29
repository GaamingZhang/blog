# Docker Compose实战

## 为什么需要Docker Compose？

假设你要运行一个Web应用，它需要：
- 一个Nginx作为反向代理
- 一个Node.js应用服务
- 一个PostgreSQL数据库
- 一个Redis缓存

如果每个都用`docker run`命令启动，你需要：
1. 分别启动4个容器
2. 配置它们的网络连接
3. 设置正确的启动顺序
4. 记住所有的参数...

这太痛苦了！

**Docker Compose**就是为了解决这个问题：用一个YAML文件描述所有服务，然后一条命令全部启动。

```
传统方式：                          Docker Compose：
docker run nginx...                docker compose up
docker run node...                 （一条命令搞定）
docker run postgres...
docker run redis...
```

## 核心概念

### Compose文件是什么？

一个名为`docker-compose.yml`的YAML文件，描述了你的应用由哪些"服务"组成：

```yaml
services:
  web:        # 服务1：Web应用
    image: nginx
  api:        # 服务2：API服务
    build: ./api
  db:         # 服务3：数据库
    image: postgres
```

**服务（Service）**不是一个容器，而是一个"应用组件"的定义。一个服务可以运行多个容器实例。

### 项目的概念

Compose会自动给你的应用创建一个"项目"，默认用文件夹名作为项目名。同一个项目里的容器：
- 共享一个专用网络
- 可以用服务名互相访问

比如在`myapp`文件夹里运行compose，会创建：
- `myapp_web_1`容器
- `myapp_api_1`容器
- `myapp_db_1`容器
- `myapp_default`网络

## 最小配置示例

```yaml
# docker-compose.yml
services:
  web:
    image: nginx
    ports:
      - "80:80"
```

运行：
```bash
docker compose up
```

这等同于：
```bash
docker run -p 80:80 nginx
```

看起来没省多少事？当服务变多时，价值就体现出来了。

## 一个完整的Web应用示例

```yaml
services:
  # Nginx反向代理
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - api

  # Node.js API服务
  api:
    build: ./api                    # 从Dockerfile构建
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/myapp
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis

  # PostgreSQL数据库
  db:
    image: postgres:15
    environment:
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
      - POSTGRES_DB=myapp
    volumes:
      - db_data:/var/lib/postgresql/data   # 数据持久化

  # Redis缓存
  redis:
    image: redis:alpine

# 定义命名卷
volumes:
  db_data:
```

**一条命令启动所有服务**：
```bash
docker compose up -d
```

## 理解服务间的通信

这是Compose最强大的地方：**服务可以用服务名互相访问**。

在上面的例子中，API服务可以用`db:5432`连接数据库，用`redis:6379`连接缓存。不需要知道IP地址，Compose会自动处理DNS解析。

```
┌─────────────────────────────────────────────────┐
│              Compose 默认网络                    │
│                                                 │
│  ┌─────┐      ┌─────┐      ┌─────┐      ┌─────┐│
│  │nginx│─────>│ api │─────>│ db  │      │redis││
│  │:80  │      │:3000│      │:5432│      │:6379││
│  └─────┘      └─────┘      └─────┘      └─────┘│
│                                                 │
└─────────────────────────────────────────────────┘
```

## 启动顺序：depends_on的真相

`depends_on`只保证**启动顺序**，不保证服务**就绪**。

```yaml
api:
  depends_on:
    - db    # db容器会先启动，但数据库可能还没准备好接受连接
```

数据库容器启动后，数据库服务初始化可能还需要几秒钟。如果API立即连接，可能会失败。

**解决方案1**：使用健康检查

```yaml
api:
  depends_on:
    db:
      condition: service_healthy   # 等待db健康检查通过

db:
  image: postgres:15
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U postgres"]
    interval: 5s
    timeout: 5s
    retries: 5
```

**解决方案2**：应用自己实现重试逻辑（更可靠）

## 数据持久化

容器删除后数据会丢失，需要用卷来持久化：

```yaml
services:
  db:
    image: postgres:15
    volumes:
      - db_data:/var/lib/postgresql/data   # 命名卷

volumes:
  db_data:   # 在这里声明
```

**命名卷 vs 绑定挂载**：

| 类型 | 语法 | 适用场景 |
|------|------|----------|
| 命名卷 | `db_data:/path` | 数据库等持久化数据 |
| 绑定挂载 | `./local:/path` | 开发时同步代码 |

## 环境变量管理

三种方式传递环境变量：

**1. 直接在compose文件中**（不推荐敏感信息）
```yaml
environment:
  - API_KEY=secret123
```

**2. 使用.env文件**
```bash
# .env
API_KEY=secret123
```

```yaml
environment:
  - API_KEY=${API_KEY}
```

**3. 使用env_file**
```yaml
env_file:
  - .env.production
```

## 开发 vs 生产配置

通常需要不同的配置：
- 开发：挂载代码、开放调试端口
- 生产：资源限制、自动重启

**方案：配置文件覆盖**

```yaml
# docker-compose.yml（基础配置）
services:
  app:
    build: .
    depends_on:
      - db
```

```yaml
# docker-compose.override.yml（开发配置，自动加载）
services:
  app:
    volumes:
      - .:/app            # 挂载代码
    ports:
      - "9229:9229"       # 调试端口
    environment:
      - DEBUG=true
```

```yaml
# docker-compose.prod.yml（生产配置）
services:
  app:
    restart: always
    deploy:
      resources:
        limits:
          memory: 512M
```

```bash
# 开发环境（自动使用override）
docker compose up

# 生产环境
docker compose -f docker-compose.yml -f docker-compose.prod.yml up
```

## 常用命令

| 命令 | 作用 |
|------|------|
| `docker compose up` | 启动所有服务 |
| `docker compose up -d` | 后台启动 |
| `docker compose down` | 停止并删除容器 |
| `docker compose down -v` | 同时删除卷 |
| `docker compose ps` | 查看状态 |
| `docker compose logs -f` | 查看日志 |
| `docker compose exec api sh` | 进入容器 |
| `docker compose build` | 重新构建镜像 |
| `docker compose up -d --build` | 重新构建并更新 |

## 常见问题

### Q1: 端口冲突怎么办？

如果本机80端口被占用：
```yaml
ports:
  - "8080:80"    # 用8080映射到容器的80
```

### Q2: 怎么只重启一个服务？

```bash
docker compose restart api
```

### Q3: 修改代码后怎么更新？

如果是绑定挂载的代码，修改自动生效。如果需要重新构建：
```bash
docker compose up -d --build api
```

### Q4: 怎么查看某个服务的日志？

```bash
docker compose logs -f api     # -f 表示持续跟踪
```

### Q5: 服务启动失败怎么排查？

```bash
docker compose logs api        # 查看日志
docker compose config          # 验证配置是否正确
docker compose ps -a           # 查看容器状态（包括退出的）
```

### Q6: 怎么清理所有资源重新开始？

```bash
docker compose down -v --rmi all
# -v 删除卷
# --rmi all 删除镜像
```

## 小结

| 概念 | 作用 |
|------|------|
| **services** | 定义应用的各个组件 |
| **volumes** | 数据持久化 |
| **networks** | 服务间通信（默认自动创建） |
| **depends_on** | 定义启动顺序（不保证就绪） |
| **healthcheck** | 确保服务真正就绪 |

Docker Compose的核心价值：
- **一个文件描述整个应用**：所有配置集中管理
- **一条命令启动所有服务**：不用记复杂的docker run参数
- **自动处理网络**：服务可以用名字互相访问
- **方便的配置覆盖**：开发和生产用不同配置

记住：Compose适合**单机**多容器应用。如果需要跨多台机器运行，要考虑Kubernetes或Docker Swarm。

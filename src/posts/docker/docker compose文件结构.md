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

# Docker Compose：多容器编排的原理

## 为什么需要Docker Compose

想象你要部署一个Web应用，它包含：
- 一个Nginx反向代理
- 两个Node.js应用服务器
- 一个PostgreSQL数据库
- 一个Redis缓存
- 一个Prometheus监控

如果用`docker run`命令逐个启动，你需要：

```bash
docker network create myapp-net
docker run -d --name redis --network myapp-net redis
docker run -d --name postgres --network myapp-net -e POSTGRES_PASSWORD=secret postgres
docker run -d --name app1 --network myapp-net -e DB_HOST=postgres myapp:v1
docker run -d --name app2 --network myapp-net -e DB_HOST=postgres myapp:v1
docker run -d --name nginx --network myapp-net -p 80:80 nginx
docker run -d --name prometheus --network myapp-net prometheus
```

**问题**：
- 命令太多，容易出错
- 启动顺序难以控制
- 环境变量到处重复
- 团队协作时难以同步配置

**Docker Compose的解决方案**：用一个YAML文件描述所有服务，一条命令启动全部。

---

## Compose的核心概念

### Project（项目）

Compose会自动为你的应用创建一个"项目"，默认使用文件夹名：

```
/home/user/myapp/
  ├─ docker-compose.yml
  └─ app/

项目名：myapp
```

**项目的作用**：
- 所有资源都有项目前缀（`myapp_web_1`）
- 同一项目的容器共享专用网络
- 可以通过项目名隔离不同环境

### Service（服务）

服务不是容器，而是**容器的定义模板**。一个服务可以运行多个容器实例。

```
服务定义：web
  ↓
实际容器：myapp_web_1, myapp_web_2, myapp_web_3
```

### 自动网络

Compose自动创建一个网络，所有服务默认加入：

```
网络名：myapp_default

容器可以用服务名互相访问：
app → http://db:5432
app → redis://cache:6379
```

---

## Compose文件的基本结构

一个Compose文件由几个顶级键组成：

```yaml
version: '3.8'     # Compose文件格式版本

services:          # 服务定义（核心）
  web:
    ...
  db:
    ...

networks:          # 自定义网络（可选）
  frontend:
    ...

volumes:           # 持久化卷（可选）
  db_data:
    ...
```

---

## Services：应用的组件定义

### 镜像来源：image vs build

每个服务需要指定镜像来源：

**方式1：使用现成的镜像**

```yaml
services:
  db:
    image: postgres:15
```

**工作原理**：Compose直接使用这个镜像，不需要构建。

**方式2：从Dockerfile构建**

```yaml
services:
  app:
    build: ./app
```

**工作原理**：
1. Compose查找`./app/Dockerfile`
2. 执行`docker build`构建镜像
3. 使用构建的镜像创建容器

**方式3：构建并命名镜像**

```yaml
services:
  app:
    build: ./app
    image: myapp:latest
```

**工作原理**：构建后给镜像打上`myapp:latest`标签，可以推送到仓库。

---

## 环境变量：三种传递方式

### 方式1：直接定义

```yaml
services:
  app:
    environment:
      - DEBUG=true
      - DB_HOST=postgres
```

**优点**：简单直观
**缺点**：敏感信息暴露在配置文件中

### 方式2：从.env文件读取

```yaml
# docker-compose.yml
services:
  app:
    environment:
      - API_KEY=${API_KEY}
```

```bash
# .env文件
API_KEY=secret123
DB_PASSWORD=password
```

**工作原理**：
1. Compose自动读取`.env`文件
2. 变量替换：`${API_KEY}` → `secret123`
3. 传递给容器

**优点**：敏感信息不入版本控制（.gitignore添加.env）

### 方式3：从文件加载多个变量

```yaml
services:
  app:
    env_file:
      - .env.common
      - .env.production
```

**工作原理**：按顺序读取多个env文件，后面的覆盖前面的。

---

## 网络：容器间通信的桥梁

### 默认网络行为

Compose自动做了三件事：

1. **创建网络**：`项目名_default`
2. **连接所有服务**：每个容器都加入这个网络
3. **DNS解析**：服务名自动解析为容器IP

```
容器web尝试访问：http://api:3000
         ↓
Compose内置DNS：
"api" → 172.18.0.3
         ↓
连接成功
```

### 自定义网络：服务分组

```yaml
services:
  nginx:
    networks:
      - frontend

  app:
    networks:
      - frontend
      - backend

  db:
    networks:
      - backend

networks:
  frontend:
  backend:
```

**拓扑结构**：

```
┌───────────────────────────────┐
│      frontend 网络            │
│  ┌──────┐       ┌──────┐     │
│  │nginx │──────>│ app  │     │
│  └──────┘       └──────┘     │
└───────────────────┼───────────┘
                    │
┌───────────────────┼───────────┐
│      backend 网络  │           │
│                ┌──┴───┐       │
│                │ app  │       │
│                └───┬──┘       │
│                    │           │
│                ┌───▼──┐       │
│                │  db  │       │
│                └──────┘       │
└───────────────────────────────┘
```

**安全效果**：
- nginx只能访问app，不能直接访问db
- db完全与外部隔离，只有app能访问

---

## 数据持久化：volumes vs bind mounts

### 命名卷（Volume）

```yaml
services:
  db:
    volumes:
      - db_data:/var/lib/postgresql/data

volumes:
  db_data:
```

**工作原理**：
1. Compose创建名为`项目名_db_data`的卷
2. 卷数据存储在Docker管理的目录（`/var/lib/docker/volumes/`）
3. 容器删除后，卷数据仍然保留

**适用场景**：数据库数据、应用生成的文件

### 绑定挂载（Bind Mount）

```yaml
services:
  app:
    volumes:
      - ./app:/app
      - ./config.yml:/app/config.yml
```

**工作原理**：
1. 直接将宿主机的`./app`目录映射到容器的`/app`
2. 两边是同一块存储区域
3. 容器修改文件，宿主机立即看到

**适用场景**：
- 开发环境：修改代码自动生效
- 配置文件：从宿主机传入

### 临时文件系统（tmpfs）

```yaml
services:
  app:
    tmpfs:
      - /tmp
      - /run
```

**工作原理**：在内存中挂载文件系统，容器重启后数据消失。

**适用场景**：临时缓存、敏感数据（不留磁盘痕迹）

---

## depends_on：启动顺序控制

### 基础用法

```yaml
services:
  app:
    depends_on:
      - db
      - redis

  db:
    image: postgres

  redis:
    image: redis
```

**Compose的启动顺序**：
1. 启动db
2. 启动redis
3. 等db和redis启动后，启动app

### 重要误区

`depends_on`只保证**容器启动**，不保证**服务就绪**！

```
时间线：
0s   db容器启动
1s   app容器启动，尝试连接db → 失败！
2s   db服务初始化完成，可以接受连接
```

### 解决方案1：健康检查

```yaml
services:
  db:
    image: postgres
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "postgres"]
      interval: 5s
      timeout: 3s
      retries: 3

  app:
    depends_on:
      db:
        condition: service_healthy
```

**工作原理**：
1. db容器启动
2. 每5秒执行`pg_isready`检查
3. 连续3次成功 → 标记为healthy
4. app容器才开始启动

### 解决方案2：应用层重试

```javascript
// app代码内置重试
async function connectDB() {
  for (let i = 0; i < 10; i++) {
    try {
      await db.connect();
      return;
    } catch (e) {
      console.log('DB not ready, retrying...');
      await sleep(2000);
    }
  }
  throw new Error('DB connection failed');
}
```

**推荐**：应用层重试更可靠，因为即使在生产环境，数据库也可能短暂不可用。

---

## 资源限制：防止单个服务耗尽资源

```yaml
services:
  app:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

**工作原理**：

**limits（硬限制）**：
- CPU：最多使用1个核心
- 内存：超过512MB触发OOM killer

**reservations（软限制）**：
- CPU：保证至少0.5个核心
- 内存：保证至少256MB

**注意**：
- 单机模式下，`deploy`部分的某些配置不生效
- 使用`docker-compose --compatibility`可以部分支持

---

## 重启策略：容器崩溃后的行为

```yaml
services:
  app:
    restart: always
```

**策略选项**：

| 策略 | 行为 |
|------|------|
| `no` | 从不自动重启（默认） |
| `always` | 总是重启，即使手动停止 |
| `on-failure` | 仅在非正常退出时重启 |
| `unless-stopped` | 总是重启，除非手动停止 |

**工作原理**：

```
容器崩溃（exit code != 0）
         ↓
Docker检查重启策略
         ↓
restart: always → 立即重启
restart: on-failure → 重启
restart: no → 不重启
```

**生产环境推荐**：`unless-stopped`（允许手动停止，其他情况自动恢复）

---

## 配置覆盖：开发/生产环境分离

### 基础配置

```yaml
# docker-compose.yml（基础配置）
services:
  app:
    image: myapp
    depends_on:
      - db
```

### 开发环境覆盖

```yaml
# docker-compose.override.yml（自动加载）
services:
  app:
    volumes:
      - ./src:/app/src    # 代码热更新
    environment:
      - DEBUG=true
```

### 生产环境配置

```yaml
# docker-compose.prod.yml
services:
  app:
    restart: unless-stopped
    deploy:
      replicas: 3
      resources:
        limits:
          memory: 512M
```

**使用方式**：

```bash
# 开发环境（自动合并 docker-compose.yml + docker-compose.override.yml）
docker-compose up

# 生产环境（合并 docker-compose.yml + docker-compose.prod.yml）
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up
```

**合并规则**：
- 标量值（字符串、数字）：后者覆盖前者
- 列表值：后者追加到前者
- 对象值：递归合并

---

## Compose的工作流程

### docker-compose up的内部流程

```
1. 读取配置文件
   ├─ docker-compose.yml
   ├─ docker-compose.override.yml（如果存在）
   └─ .env（环境变量）

2. 验证配置
   └─ 检查YAML语法和配置项有效性

3. 创建资源
   ├─ 创建网络（如果不存在）
   └─ 创建卷（如果不存在）

4. 拉取/构建镜像
   ├─ image: 拉取镜像
   └─ build: 构建镜像

5. 创建容器（按depends_on顺序）
   ├─ 设置网络连接
   ├─ 挂载卷
   └─ 配置环境变量

6. 启动容器（按depends_on顺序）

7. 附加日志输出（如果是前台模式）
```

### docker-compose down的清理流程

```
1. 停止所有容器

2. 删除容器

3. 删除网络（如果没有其他容器使用）

4. 删除卷（如果指定了-v）

5. 删除镜像（如果指定了--rmi）
```

---

## 最佳实践

### 1. 不要在Compose文件中硬编码敏感信息

```yaml
# ❌ 错误
services:
  db:
    environment:
      POSTGRES_PASSWORD: mypassword

# ✅ 正确
services:
  db:
    environment:
      POSTGRES_PASSWORD: ${DB_PASSWORD}
```

### 2. 使用健康检查确保服务就绪

```yaml
services:
  db:
    healthcheck:
      test: ["CMD-SHELL", "pg_isready"]
      interval: 10s
      timeout: 5s
      retries: 5
```

### 3. 为生产环境设置资源限制

```yaml
services:
  app:
    deploy:
      resources:
        limits:
          cpus: '0.50'
          memory: 512M
```

### 4. 使用命名卷持久化数据

```yaml
services:
  db:
    volumes:
      - db_data:/var/lib/postgresql/data

volumes:
  db_data:
```

### 5. 利用网络隔离提高安全性

```yaml
services:
  web:
    networks:
      - frontend

  api:
    networks:
      - frontend
      - backend

  db:
    networks:
      - backend
```

---

## 总结

Docker Compose通过声明式配置简化了多容器应用的管理：

**核心概念**：
- **Project**：应用的逻辑分组，自动资源命名
- **Service**：容器的定义模板，可以扩展实例
- **Network**：自动DNS解析，支持服务间通信
- **Volume**：数据持久化，独立于容器生命周期

**工作原理**：
1. 解析YAML配置和环境变量
2. 创建网络、卷等基础资源
3. 按依赖顺序拉取/构建镜像
4. 创建并启动容器
5. 提供统一的管理接口

**关键理解**：
- Compose是编排工具，不是集群管理（单机使用）
- `depends_on`只控制启动顺序，不保证服务就绪
- 网络隔离是安全的重要手段
- 配置覆盖机制支持多环境部署

Docker Compose让"基础设施即代码"成为现实，团队只需共享一个YAML文件，就能保证环境的完全一致。

## 参考资源

- [Docker Compose 官方文档](https://docs.docker.com/compose/)
- [Compose 文件格式参考](https://docs.docker.com/compose/compose-file/)
- [Docker Compose 入门教程](https://docs.docker.com/compose/gettingstarted/)

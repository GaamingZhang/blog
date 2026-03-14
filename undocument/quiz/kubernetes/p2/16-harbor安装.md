---
date: 2026-03-12
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Docker
tag:
  - Docker
  - Harbor
  - 镜像仓库
---

# Harbor 安装全攻略：三种部署方式详解

## 引言：为什么企业需要 Harbor？

在容器化时代，Docker 镜像仓库是基础设施的核心组件。虽然 Docker Hub 提供了公共镜像服务，但企业在生产环境中面临着诸多挑战：网络延迟、安全合规、镜像管理效率等问题日益突出。Harbor 作为一个企业级的 Docker Registry 项目，应运而生。

Harbor 由 VMware 开源，基于 Docker Registry 开发，提供了企业级镜像仓库所需的完整功能集。它不仅解决了镜像存储的基本需求，更在安全性、管理效率、高可用性等方面提供了全面的解决方案。对于追求数据主权和安全合规的企业而言，Harbor 已成为私有镜像仓库的首选方案。

### Harbor 的核心优势

**安全性与合规性**：Harbor 支持基于角色的访问控制（RBAC），集成 LDAP/AD 认证，支持镜像漏洞扫描（Trivy、Clair 等），镜像签名验证（Notary），以及镜像内容信任机制。这些特性确保了镜像从构建到部署的全链路安全。

**企业级管理能力**：提供图形化管理界面，支持镜像复制（跨区域同步）、镜像标签保留策略、垃圾回收机制、配额管理等高级功能。这些能力使得大规模镜像管理变得可控和高效。

**高性能与可靠性**：支持多种存储后端（本地存储、S3、Swift、OSS 等），支持 Redis 缓存加速，支持高可用部署模式，满足企业级生产环境的需求。

## Harbor 架构深度解析

理解 Harbor 的架构对于正确部署和运维至关重要。Harbor 采用微服务架构，由多个核心组件协同工作。

### 核心组件架构

Harbor 的架构设计遵循了微服务原则，各组件职责明确，通过容器化方式部署：

```
┌─────────────────────────────────────────────────────────────┐
│                        Harbor 架构                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Nginx      │    │   Portal     │    │   Core       │  │
│  │  (Proxy)     │◄───│   (UI)       │◄───│   (API)      │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│         │                    │                    │          │
│         └────────────────────┼────────────────────┘          │
│                              │                               │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Registry   │    │   Redis      │    │  PostgreSQL  │  │
│  │  (存储)      │◄───│  (缓存)      │◄───│  (数据库)    │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│         │                                                    │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Trivy      │    │   Chart      │    │   Notary     │  │
│  │ (扫描器)     │    │   Museum     │    │  (签名)      │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Proxy (Nginx)**：作为统一入口，负责反向代理和负载均衡。所有外部请求（Docker 客户端推送/拉取镜像、用户访问 UI）都通过 Nginx 分发到相应的后端服务。Nginx 配置了 SSL/TLS 终止，确保传输层安全。

**Core Services**：Harbor 的核心 API 服务，处理所有业务逻辑。包括项目管理、用户认证、镜像元数据管理、配额控制、Webhook 触发等功能。Core 是 Harbor 的大脑，协调各个组件的工作。

**Registry**：基于 Docker Distribution 开源项目，负责实际的镜像存储和分发。Registry 处理 Docker 客户端的 push/pull 请求，将镜像层（layer）存储到配置的后端存储中。Harbor 通过 Hook 机制在 Registry 前后插入自定义逻辑，实现权限控制和元数据同步。

**Portal (UI)**：提供 Web 管理界面，用户可以通过浏览器进行项目管理、镜像浏览、扫描结果查看、复制规则配置等操作。Portal 是用户与 Harbor 交互的主要入口。

**PostgreSQL**：存储 Harbor 的所有元数据，包括用户信息、项目配置、镜像元数据、标签信息、扫描结果等。PostgreSQL 是 Harbor 的数据持久化核心，建议在生产环境中配置高可用方案。

**Redis**：用于缓存和会话管理。存储用户登录会话、临时数据、Job 状态等。Redis 提升了 Harbor 的响应速度，减轻了数据库压力。

**Trivy Scanner**：镜像漏洞扫描器。Trivy 是 Harbor 默认集成的扫描器，能够检测镜像中的操作系统包漏洞和应用依赖漏洞。扫描结果存储在数据库中，可通过 UI 或 API 查询。

**Chart Museum**（可选）：Helm Chart 仓库组件。如果需要在 Harbor 中存储和管理 Helm Chart，Chart Museum 提供了 Chart 的存储和索引服务。

**Notary**（可选）：镜像签名服务。基于 Docker Content Trust，实现镜像的签名和验证机制，确保镜像的完整性和来源可信。

### 数据流向与交互机制

当用户执行 `docker push` 时，请求流程如下：

1. Docker 客户端通过 HTTPS 连接到 Nginx（Proxy）
2. Nginx 将请求转发到 Core 服务进行认证和授权
3. 认证通过后，请求转发到 Registry
4. Registry 接收镜像层数据，存储到后端存储（如文件系统、S3）
5. Registry 通过 Webhook 通知 Core 更新元数据
6. Core 将镜像信息写入 PostgreSQL
7. 如果配置了自动扫描，Core 触发 Trivy 进行漏洞扫描

这种架构设计实现了关注点分离，各组件独立扩展，便于运维和故障排查。

## 安装方式一：在线安装（Online Installer）

在线安装是最简单快捷的方式，适合网络环境良好、能够访问外网的场景。安装器会自动从 Docker Hub 拉取所需的镜像组件。

### 安装前提条件

在开始安装前，确保目标服务器满足以下要求：

**硬件要求**：
- CPU：2 核以上（推荐 4 核）
- 内存：4GB 以上（推荐 8GB，生产环境建议 16GB）
- 磁盘：40GB 以上（根据镜像存储量规划，建议 100GB+）

**软件要求**：
- 操作系统：Linux（CentOS 7+/Ubuntu 18.04+/Debian 10+）
- Docker：17.06.0-ce+ 版本
- Docker Compose：1.18.0+ 版本
- OpenSSL：用于生成证书

### 安装步骤详解

**步骤 1：下载在线安装器**

```bash
# 访问 Harbor GitHub Release 页面获取最新版本
# https://github.com/goharbor/harbor/releases

# 下载在线安装包（示例版本 2.10.0）
wget https://github.com/goharbor/harbor/releases/download/v2.10.0/harbor-online-installer-v2.10.0.tgz

# 解压安装包
tar -xzf harbor-online-installer-v2.10.0.tgz

# 进入安装目录
cd harbor
```

**步骤 2：配置 harbor.yml**

Harbor 的核心配置文件是 `harbor.yml`，需要根据实际环境修改关键参数：

```yaml
# hostname 设置为 Harbor 服务器的域名或 IP
hostname: harbor.example.com

# HTTP/HTTPS 端口配置
http:
  port: 80
https:
  port: 443
  certificate: /your/certificate/path
  private_key: /your/private/key/path

# Harbor 管理员密码（首次登录后需修改）
harbor_admin_password: Harbor12345

# 数据库配置（生产环境建议外置）
database:
  password: root123
  max_idle_conns: 100
  max_open_conns: 900

# 数据存储路径
data_volume: /data/harbor

# 日志配置
log:
  level: info
  local:
    rotate_count: 50
    rotate_size: 200M
    location: /var/log/harbor
```

**关键配置项说明**：

- `hostname`：必须配置为客户端可访问的域名或 IP，不能使用 localhost 或 127.0.0.1
- `https`：生产环境强烈建议启用 HTTPS，配置有效的 SSL 证书
- `data_volume`：镜像数据的存储路径，确保磁盘空间充足
- `harbor_admin_password`：默认管理员 admin 的初始密码

**步骤 3：生成 SSL 证书（生产环境必需）**

```bash
# 创建证书目录
mkdir -p /data/cert
cd /data/cert

# 生成 CA 私钥
openssl genrsa -out ca.key 4096

# 生成 CA 证书
openssl req -x509 -new -nodes -sha512 -days 3650 \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=example/OU=IT/CN=harbor.example.com" \
  -key ca.key \
  -out ca.crt

# 生成服务器私钥
openssl genrsa -out harbor.example.com.key 4096

# 生成证书签名请求（CSR）
openssl req -sha512 -new \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=example/OU=IT/CN=harbor.example.com" \
  -key harbor.example.com.key \
  -out harbor.example.com.csr

# 生成 x509 v3 扩展文件（支持多域名和 IP）
cat > v3.ext <<-EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1=harbor.example.com
DNS.2=harbor
IP.1=192.168.1.100
EOF

# 使用 CA 签发服务器证书
openssl x509 -req -sha512 -days 3650 \
  -extfile v3.ext \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -in harbor.example.com.csr \
  -out harbor.example.com.crt
```

证书生成后，更新 `harbor.yml` 中的证书路径：

```yaml
https:
  port: 443
  certificate: /data/cert/harbor.example.com.crt
  private_key: /data/cert/harbor.example.com.key
```

**步骤 4：执行安装脚本**

```bash
# 在 harbor 目录下执行安装脚本
./install.sh

# 安装过程会自动：
# 1. 检查 Docker 和 Docker Compose 版本
# 2. 拉取所需的镜像组件
# 3. 生成 Docker Compose 配置文件
# 4. 启动所有服务容器

# 安装完成后，检查服务状态
docker-compose ps

# 输出示例：
# Name                     Command               State                    Ports
# -------------------------------------------------------------------------------------------
# harbor-core         /harbor/entrypoint.sh            Up
# harbor-db           /entrypoint.sh postgres          Up
# harbor-jobservice   /harbor/entrypoint.sh            Up
# harbor-log          /bin/sh -c /usr/local/bin/ ...   Up
# harbor-portal       nginx -g daemon off;             Up
# harbor-redis        redis-server /etc/redis.conf     Up
# harbor-registry     /home/harbor/entrypoint.sh       Up
# nginx               nginx -g daemon off;             Up
```

**步骤 5：验证安装**

```bash
# 浏览器访问
https://harbor.example.com

# 默认账号：admin
# 默认密码：Harbor12345（配置文件中设置的密码）

# Docker 客户端登录测试
docker login harbor.example.com
# Username: admin
# Password: Harbor12345

# 推送测试镜像
docker pull alpine:latest
docker tag alpine:latest harbor.example.com/library/alpine:test
docker push harbor.example.com/library/alpine:test
```

### 在线安装的优缺点

**优点**：
- 安装包体积小（仅包含配置和脚本）
- 自动拉取最新版本的组件镜像
- 安装过程简单，适合快速部署和测试

**缺点**：
- 依赖外网连接，网络不稳定可能导致安装失败
- 首次启动需要拉取镜像，耗时较长
- 不适合离线环境或安全隔离网络

## 安装方式二：离线安装（Offline Installer）

离线安装是为无外网环境或安全隔离网络设计的方案。安装包包含了所有必需的镜像文件，无需联网即可完成部署。

### 离线安装包的构成

离线安装包比在线安装包大得多（约 600MB-1GB），因为包含了所有组件的 Docker 镜像：

```
harbor-offline-installer-v2.10.0.tgz
├── harbor/
│   ├── harbor.yml.tmpl          # 配置模板
│   ├── install.sh               # 安装脚本
│   ├── prepare                  # 预处理脚本
│   ├── docker-compose.yml       # Docker Compose 配置
│   └── harbor.v2.10.0.tar.gz    # 所有组件镜像的打包文件
```

### 安装步骤详解

**步骤 1：下载离线安装包**

在有外网的机器上下载离线安装包：

```bash
# 下载离线安装包
wget https://github.com/goharbor/harbor/releases/download/v2.10.0/harbor-offline-installer-v2.10.0.tgz

# 将安装包传输到目标服务器（通过 U 盘、内网传输等方式）
scp harbor-offline-installer-v2.10.0.tgz user@target-server:/tmp/
```

**步骤 2：解压并配置**

```bash
# 在目标服务器上解压
cd /tmp
tar -xzf harbor-offline-installer-v2.10.0.tgz
cd harbor

# 复制配置模板
cp harbor.yml.tmpl harbor.yml

# 编辑配置文件（与在线安装相同）
vi harbor.yml

# 配置要点：
# 1. 设置 hostname
# 2. 配置 HTTPS 证书
# 3. 设置数据存储路径
# 4. 修改管理员密码
```

**步骤 3：加载镜像并安装**

```bash
# 离线安装包中的镜像需要先加载到本地 Docker
# install.sh 脚本会自动执行 docker load 操作

# 执行安装
./install.sh

# 安装脚本执行流程：
# 1. 检查依赖（Docker、Docker Compose）
# 2. 加载 harbor.v2.10.0.tar.gz 中的镜像
# 3. 生成配置文件
# 4. 启动服务

# 查看加载的镜像
docker images | grep goharbor

# 输出示例：
# goharbor/harbor-core          v2.10.0    abc123def456   2 minutes ago   150MB
# goharbor/harbor-portal        v2.10.0    def456ghi789   2 minutes ago   50MB
# goharbor/harbor-db            v2.10.0    ghi789jkl012   2 minutes ago   200MB
# ...
```

**步骤 4：验证与配置**

验证步骤与在线安装相同，访问 Web UI 和测试 Docker 登录。

### 离线安装的适用场景

**适用场景**：
- 生产环境安全隔离网络（无法访问外网）
- 网络带宽受限的环境
- 需要快速批量部署多套 Harbor 实例
- 对镜像版本有严格控制要求

**注意事项**：
- 离线安装包版本固定，升级需要重新下载新版本
- 安装包体积大，传输和存储需要考虑空间
- 建议在内网搭建文件服务器，统一管理离线安装包

## 安装方式三：Helm 安装（Kubernetes 环境）

在 Kubernetes 集群中部署 Harbor 是云原生环境的推荐方式。Helm Chart 提供了声明式配置、版本管理和滚动升级能力，适合生产环境的高可用部署。

### Helm 安装的优势

**云原生集成**：Harbor 作为 Kubernetes 原生应用运行，与集群生态深度集成。支持 Kubernetes 的服务发现、负载均衡、自动重启等特性。

**高可用部署**：通过调整副本数和配置，轻松实现 Harbor 的高可用。数据库和 Redis 可以使用外部集群，避免单点故障。

**声明式配置**：使用 values.yaml 文件管理所有配置，支持 GitOps 工作流，配置变更可追溯、可回滚。

**弹性伸缩**：根据负载动态调整 Harbor 组件的副本数，优化资源利用率。

### Helm 安装步骤详解

**步骤 1：准备 Kubernetes 集群**

```bash
# 确保 Kubernetes 集群正常运行
kubectl cluster-info

# 创建 Harbor 命名空间
kubectl create namespace harbor

# 确保 StorageClass 可用（用于持久化存储）
kubectl get storageclass

# 输出示例：
# NAME                 PROVISIONER                     AGE
# standard (default)   kubernetes.io/gce-pd           10d
```

**步骤 2：添加 Harbor Helm Chart 仓库**

```bash
# 添加 Harbor 官方 Helm 仓库
helm repo add harbor https://helm.goharbor.io

# 更新仓库索引
helm repo update

# 搜索 Harbor Chart
helm search repo harbor/harbor

# 输出示例：
# NAME            CHART VERSION   APP VERSION   DESCRIPTION
# harbor/harbor   1.14.0          2.10.0        An open source trusted cloud native...
```

**步骤 3：创建 values.yaml 配置文件**

创建自定义配置文件，根据实际需求调整参数：

```yaml
# harbor-values.yaml

# 暴露配置
expose:
  type: ingress
  ingress:
    hosts:
      core: harbor.example.com
    annotations:
      kubernetes.io/ingress.class: nginx
      cert-manager.io/cluster-issuer: letsencrypt-prod
    tls:
      enabled: true
      secretName: harbor-tls

# 外部 URL
externalURL: https://harbor.example.com

# 持久化存储配置
persistence:
  enabled: true
  resourcePolicy: "keep"
  persistentVolumeClaim:
    registry:
      storageClass: "standard"
      size: 100Gi
    chartmuseum:
      storageClass: "standard"
      size: 10Gi
    trivy:
      storageClass: "standard"
      size: 5Gi

# 数据库配置（生产环境建议外置）
database:
  type: internal
  internal:
    password: "change-this-password"
    persistentVolumeClaim:
      storageClass: "standard"
      size: 10Gi

# Redis 配置（生产环境建议外置）
redis:
  type: internal
  internal:
    password: "change-this-password"

# 副本数配置（高可用）
portal:
  replicas: 2
core:
  replicas: 2
registry:
  replicas: 2

# 镜像拉取策略
imagePullPolicy: IfNotPresent

# 日志级别
logLevel: info

# Harbor 初始密码
harborAdminPassword: "Harbor12345"
```

**关键配置项解析**：

**expose.type**：支持 `ingress`、`nodePort`、`loadBalancer` 三种方式。Ingress 是最常用的方式，需要集群已部署 Ingress Controller（如 Nginx Ingress）。

**persistence**：配置各组件的持久化存储。Registry 存储镜像数据，需要较大空间；ChartMuseum 存储 Helm Chart；Trivy 存储漏洞数据库。

**database/redis**：生产环境强烈建议使用外部数据库和 Redis 集群（如 PostgreSQL 集群、Redis Cluster），避免单点故障。

**replicas**：设置各组件副本数，实现高可用。Core、Portal、Registry 等无状态服务可设置多副本。

**步骤 4：安装 Harbor**

```bash
# 使用自定义配置安装 Harbor
helm install harbor harbor/harbor \
  -n harbor \
  -f harbor-values.yaml

# 查看安装状态
helm status harbor -n harbor

# 查看 Pod 状态
kubectl get pods -n harbor

# 输出示例：
# NAME                                    READY   STATUS    RESTARTS   AGE
# harbor-chartmuseum-6b8f9c5d4-x2k7m     1/1     Running   0          5m
# harbor-core-7d9f8c6b5-p3n8q             1/1     Running   0          5m
# harbor-database-0                       1/1     Running   0          5m
# harbor-jobservice-8c7f9d6e5-m4k9p       1/1     Running   0          5m
# harbor-nginx-5d8f7c9b6-q2w5e            1/1     Running   0          5m
# harbor-portal-6e9f8d7c5-r7t3y           1/1     Running   0          5m
# harbor-redis-0                          1/1     Running   0          5m
# harbor-registry-7f9g8e7d6-s8u4i         1/1     Running   0          5m
# harbor-trivy-8g0h9f8e7-t9v5j            1/1     Running   0          5m
```

**步骤 5：配置 DNS 和访问**

```bash
# 获取 Ingress 入口 IP
kubectl get ingress -n harbor

# 输出示例：
# NAME            CLASS    HOSTS                 ADDRESS         PORTS     AGE
# harbor-ingress  <none>   harbor.example.com    192.168.1.100   80, 443   10m

# 配置 DNS 解析（或在本地 /etc/hosts 添加）
# 192.168.1.100 harbor.example.com

# 浏览器访问
https://harbor.example.com
```

**步骤 6：配置 Docker 客户端**

```bash
# 在 Kubernetes 集群节点或外部客户端配置 Docker
# 创建 Docker 配置目录
mkdir -p /etc/docker

# 配置 insecure-registries（如果使用自签名证书）
cat > /etc/docker/daemon.json <<EOF
{
  "insecure-registries": ["harbor.example.com"],
  "registry-mirrors": ["https://mirror.example.com"]
}
EOF

# 重启 Docker
systemctl restart docker

# 登录 Harbor
docker login harbor.example.com
```

### Helm 安装的高级配置

**配置外部数据库**：

```yaml
database:
  type: external
  external:
    host: "postgresql.example.com"
    port: "5432"
    username: "harbor"
    password: "your-password"
    coreDatabase: "harbor_core"
    notaryServerDatabase: "notary_server"
    notarySignerDatabase: "notary_signer"
```

**配置外部 Redis**：

```yaml
redis:
  type: external
  external:
    addr: "redis.example.com:6379"
    password: "your-password"
    coreDatabaseIndex: "0"
    jobserviceDatabaseIndex: "1"
    registryDatabaseIndex: "2"
    chartmuseumDatabaseIndex: "3"
```

**配置 S3 存储**：

```yaml
persistence:
  imageChartStorage:
    type: s3
    s3:
      region: us-west-1
      bucket: harbor-storage
      accesskey: AKIAIOSFODNN7EXAMPLE
      secretkey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
      regionendpoint: https://s3.us-west-1.amazonaws.com
```

### Helm 安装的运维操作

**升级 Harbor**：

```bash
# 更新 Helm 仓库
helm repo update

# 查看可用版本
helm search repo harbor/harbor --versions

# 升级到新版本
helm upgrade harbor harbor/harbor \
  -n harbor \
  -f harbor-values.yaml \
  --version 1.15.0
```

**回滚版本**：

```bash
# 查看历史版本
helm history harbor -n harbor

# 回滚到指定版本
helm rollback harbor 1 -n harbor
```

**卸载 Harbor**：

```bash
# 卸载 Harbor（保留 PVC）
helm uninstall harbor -n harbor

# 删除命名空间（会删除 PVC）
kubectl delete namespace harbor
```

## 三种安装方式对比

| 对比维度 | 在线安装 | 离线安装 | Helm 安装 |
|---------|---------|---------|-----------|
| **网络要求** | 需要访问外网 | 无需外网 | 需要访问 Helm 仓库 |
| **安装包大小** | ~50MB | ~600MB-1GB | 无需下载（直接拉取镜像） |
| **安装速度** | 较慢（需拉取镜像） | 快（镜像已打包） | 中等（依赖镜像拉取速度） |
| **部署环境** | 单机 Docker | 单机 Docker | Kubernetes 集群 |
| **高可用支持** | 需手动配置 | 需手动配置 | 原生支持（调整副本数） |
| **配置管理** | YAML 文件 | YAML 文件 | Helm Values（声明式） |
| **升级维护** | 重新下载安装包 | 重新下载安装包 | Helm upgrade（滚动升级） |
| **适用场景** | 测试环境、有外网环境 | 生产环境、离线环境 | 云原生环境、生产环境 |
| **运维复杂度** | 低 | 低 | 中（需 K8s 知识） |
| **扩展性** | 有限 | 有限 | 强（云原生生态集成） |
| **资源利用率** | 固定资源 | 固定资源 | 动态调度、弹性伸缩 |

### 选择建议

**测试环境**：推荐在线安装，快速部署，节省时间。

**生产环境（传统架构）**：推荐离线安装，稳定可靠，不依赖外网。

**生产环境（云原生架构）**：推荐 Helm 安装，充分利用 Kubernetes 的编排能力，实现高可用和弹性伸缩。

**混合云/多云环境**：推荐 Helm 安装，便于跨集群部署和统一管理。

## Harbor 基本配置和使用

### 创建项目和用户管理

**创建项目**：

Harbor 使用项目（Project）组织镜像，支持公开和私有项目：

```bash
# 通过 Web UI 创建项目
# 1. 登录 Harbor Web UI
# 2. 点击"新建项目"
# 3. 填写项目名称（如 myapp）
# 4. 选择访问级别（公开/私有）
# 5. 设置存储配额（可选）

# 通过 API 创建项目
curl -X POST "https://harbor.example.com/api/v2.0/projects" \
  -H "Authorization: Basic YWRtaW46SGFyYm9yMTIzNDU=" \
  -H "Content-Type: application/json" \
  -d '{
    "project_name": "myapp",
    "public": false,
    "metadata": {
      "public": "false"
    }
  }'
```

**用户权限管理**：

Harbor 支持细粒度的 RBAC（基于角色的访问控制）：

| 角色 | 权限说明 |
|-----|---------|
| **访客（Guest）** | 只读权限，可拉取镜像 |
| **开发者（Developer）** | 读/写权限，可推送和拉取镜像 |
| **管理员（Admin）** | 项目管理员，可管理项目成员和配置 |
| **受限访客（Limited Guest）** | 只读权限，但无法查看镜像详情 |

```bash
# 添加项目成员（通过 Web UI）
# 1. 进入项目 -> 成员 -> 添加成员
# 2. 输入用户名或邮箱
# 3. 选择角色
# 4. 保存

# 通过 API 添加成员
curl -X POST "https://harbor.example.com/api/v2.0/projects/1/members" \
  -H "Authorization: Basic YWRtaW46SGFyYm9yMTIzNDU=" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": 2,
    "member_user": {
      "username": "developer1"
    }
  }'
```

### 镜像推送和拉取

**推送镜像**：

```bash
# 1. Docker 登录 Harbor
docker login harbor.example.com
# Username: admin
# Password: Harbor12345

# 2. 拉取测试镜像
docker pull nginx:latest

# 3. 标记镜像（打标签）
docker tag nginx:latest harbor.example.com/myapp/nginx:v1.0

# 4. 推送镜像到 Harbor
docker push harbor.example.com/myapp/nginx:v1.0

# 推送成功后，可在 Web UI 查看镜像
# 项目 -> myapp -> 仓库 -> nginx
```

**拉取镜像**：

```bash
# 从 Harbor 拉取镜像
docker pull harbor.example.com/myapp/nginx:v1.0

# 如果项目是私有的，需要先登录
docker login harbor.example.com
docker pull harbor.example.com/myapp/nginx:v1.0
```

### 镜像扫描和签名

**镜像漏洞扫描**：

Harbor 集成 Trivy 扫描器，可检测镜像中的已知漏洞：

```bash
# 通过 Web UI 扫描
# 1. 进入项目 -> 仓库 -> 选择镜像
# 2. 点击"扫描"按钮
# 3. 等待扫描完成
# 4. 查看扫描报告（漏洞等级、CVE 编号、修复建议）

# 通过 API 触发扫描
curl -X POST "https://harbor.example.com/api/v2.0/projects/myapp/repositories/nginx/artifacts/v1.0/scan" \
  -H "Authorization: Basic YWRtaW46SGFyYm9yMTIzNDU="

# 设置自动扫描（项目配置）
# 项目 -> 配置 -> 漏洞扫描 -> 勾选"自动扫描新推送的镜像"
```

**镜像签名（Notary）**：

启用镜像签名，确保镜像完整性和来源可信：

```bash
# 启用 Content Trust
export DOCKER_CONTENT_TRUST=1
export DOCKER_CONTENT_TRUST_SERVER=https://harbor.example.com:4443

# 推送签名镜像
docker push harbor.example.com/myapp/nginx:v1.0

# 拉取时验证签名
docker pull harbor.example.com/myapp/nginx:v1.0
# 如果镜像未签名或签名无效，拉取会失败
```

### 镜像复制和同步

Harbor 支持跨仓库镜像复制，适用于多数据中心或多云场景：

```bash
# 配置复制规则（Web UI）
# 1. 系统管理 -> 仓库管理 -> 新建目标
#    - 提供目标 Harbor 的 URL 和认证信息
# 2. 系统管理 -> 复制管理 -> 新建规则
#    - 选择源项目（如 myapp）
#    - 选择目标仓库
#    - 设置过滤器（如 nginx/*）
#    - 设置触发模式（手动/定时/事件驱动）

# 手动触发复制
# 复制管理 -> 选择规则 -> 复制

# 查看复制状态和日志
# 复制管理 -> 选择规则 -> 查看执行记录
```

### 垃圾回收和存储优化

Harbor 提供垃圾回收机制，清理无用的镜像层，释放存储空间：

```bash
# 手动触发垃圾回收（Web UI）
# 系统管理 -> 垃圾回收 -> 立即清理

# 配置定时垃圾回收
# 系统管理 -> 系统配置 -> 垃圾回收
# - 设置定时任务（如每天凌晨 2 点）
# - 勾选"删除无标签的镜像"

# 通过 API 触发垃圾回收
curl -X POST "https://harbor.example.com/api/v2.0/system/gc/schedule" \
  -H "Authorization: Basic YWRtaW46SGFyYm9yMTIzNDU=" \
  -H "Content-Type: application/json" \
  -d '{
    "schedule": {
      "type": "Daily",
      "cron": "0 0 2 * * *"
    }
  }'
```

**标签保留策略**：

设置镜像标签保留规则，自动清理旧版本：

```bash
# 配置标签保留策略（Web UI）
# 项目 -> 配置 -> 标签保留
# - 添加规则
# - 仓库匹配：nginx/*
# - 保留策略：保留最近 10 个标签
# - 定时执行：每天凌晨 3 点
```

## 常见问题与最佳实践

### 常见问题排查

**问题 1：Docker 登录失败**

```bash
# 错误信息：Error response from daemon: Get https://harbor.example.com/v2/: x509: certificate signed by unknown authority

# 原因：客户端不信任 Harbor 的自签名证书
# 解决方案 1：在客户端添加证书信任
# 将 Harbor CA 证书复制到客户端
scp /data/cert/ca.crt user@client:/etc/docker/certs.d/harbor.example.com/

# 重启 Docker
systemctl restart docker

# 解决方案 2：配置 Docker 忽略证书验证（不推荐生产环境）
# 修改 /etc/docker/daemon.json
{
  "insecure-registries": ["harbor.example.com"]
}

# 重启 Docker
systemctl restart docker
```

**问题 2：镜像推送失败**

```bash
# 错误信息：denied: requested access to the resource is denied

# 原因：权限不足或未登录
# 解决方案：
# 1. 确认已登录
docker login harbor.example.com

# 2. 确认用户有项目写入权限
# 项目 -> 成员 -> 检查用户角色

# 3. 确认镜像标签正确
docker tag nginx:latest harbor.example.com/myapp/nginx:v1.0
# 注意：myapp 项目必须存在
```

**问题 3：Harbor 服务异常**

```bash
# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f harbor-core

# 重启服务
docker-compose restart

# 完全重建
docker-compose down
docker-compose up -d
```

**问题 4：存储空间不足**

```bash
# 查看存储使用情况
df -h /data/harbor

# 清理无用镜像
# 1. 设置标签保留策略
# 2. 执行垃圾回收

# 扩容存储（如果使用 LVM）
lvextend -L +100G /dev/vg/harbor
resize2fs /dev/vg/harbor
```

**问题 5：Kubernetes 中 Harbor Pod 启动失败**

```bash
# 查看 Pod 状态
kubectl describe pod harbor-core-xxx -n harbor

# 常见原因：
# 1. PVC 未绑定（StorageClass 不存在或配额不足）
kubectl get pvc -n harbor

# 2. 镜像拉取失败（ImagePullBackOff）
# 检查镜像仓库访问权限

# 3. 资源不足
kubectl describe node <node-name>
# 查看资源分配情况

# 解决方案：
# 调整 values.yaml 中的资源配置
core:
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2
      memory: 4Gi
```

### 最佳实践

**安全最佳实践**：

1. **启用 HTTPS**：生产环境必须使用 HTTPS，配置有效的 SSL 证书。自签名证书需要在所有客户端手动信任。

2. **启用认证和 RBAC**：禁用匿名访问，为不同团队分配最小权限原则的角色。

3. **启用镜像扫描**：配置自动扫描，阻止高危漏洞镜像部署。

4. **启用镜像签名**：使用 Notary 实现镜像内容信任，防止镜像篡改。

5. **定期更新密码**：定期更换管理员密码和数据库密码，使用强密码策略。

**性能优化实践**：

1. **使用外部数据库**：生产环境使用外部 PostgreSQL 集群，避免单点故障。

2. **使用外部 Redis**：使用外部 Redis 集群，提升缓存性能。

3. **配置 CDN 加速**：对于跨地域场景，配置 CDN 加速镜像下载。

4. **优化存储后端**：使用高性能存储（如 SSD、S3），避免磁盘 I/O 瓶颈。

5. **调整并发参数**：根据负载调整 Registry 的并发上传/下载参数。

**运维最佳实践**：

1. **定期备份**：定期备份 PostgreSQL 数据库和配置文件。

2. **监控告警**：集成 Prometheus 和 Grafana，监控 Harbor 性能指标。

3. **日志管理**：集中收集和分析 Harbor 日志，便于故障排查。

4. **版本升级**：定期升级 Harbor 版本，获取安全补丁和新功能。

5. **灾难恢复**：制定灾难恢复计划，定期演练恢复流程。

## 常见问题快速解答

**Q1：Harbor 支持哪些存储后端？**

A：Harbor 支持多种存储后端，包括本地文件系统、Amazon S3、Google Cloud Storage、Azure Blob Storage、Aliyun OSS、Swift 等。生产环境推荐使用对象存储（如 S3），具备高可用和弹性扩展能力。

**Q2：如何实现 Harbor 高可用部署？**

A：高可用部署需要：1）使用外部 PostgreSQL 集群；2）使用外部 Redis 集群；3）配置多副本无状态服务（Core、Portal、Registry）；4）使用共享存储或对象存储；5）前端配置负载均衡器。

**Q3：Harbor 如何与 CI/CD 集成？**

A：Harbor 提供 RESTful API 和 Webhook，可与 Jenkins、GitLab CI、Tekton 等 CI/CD 工具集成。典型流程：代码提交 -> CI 构建镜像 -> 推送到 Harbor -> 触发镜像扫描 -> 自动部署到 Kubernetes。

**Q4：如何迁移现有镜像到 Harbor？**

A：使用 `docker save/load` 或 `skopeo` 工具批量迁移。推荐 skopeo，支持增量同步和跨仓库复制，无需解压镜像。示例：`skopeo copy docker://docker.io/nginx:latest docker://harbor.example.com/library/nginx:latest`。

**Q5：Harbor 的垃圾回收机制如何工作？**

A：垃圾回收清理未被任何镜像标签引用的层（layer）。当删除镜像标签或设置标签保留策略后，无用的层不会立即删除，需执行垃圾回收。垃圾回收期间，Registry 进入只读模式，建议在业务低峰期执行。

## 总结

Harbor 作为企业级 Docker Registry 解决方案，提供了丰富的功能和灵活的部署方式。在线安装适合快速测试，离线安装适合安全隔离环境，Helm 安装适合云原生生产环境。选择合适的安装方式，结合最佳实践，能够构建安全、高效、可靠的镜像管理平台，为容器化应用提供坚实的基础设施支撑。

---

## 面试回答

**面试官问：docker-harbor 是怎么安装的？有哪几种安装方式？**

**回答**：

Harbor 有三种主要的安装方式，各有其适用场景。

第一种是在线安装，这是最简单的方式。只需要下载一个很小的安装包（约 50MB），执行 `install.sh` 脚本后，会自动从 Docker Hub 拉取所有需要的组件镜像。这种方式适合测试环境或网络环境良好的场景，优点是安装包小、操作简单，缺点是依赖外网，首次启动较慢。

第二种是离线安装，专门为无外网环境设计。离线安装包包含了所有组件的镜像文件（约 600MB-1GB），解压后直接执行安装脚本即可，无需联网。这种方式适合生产环境或安全隔离网络，优点是安装快速、不依赖外网，缺点是安装包体积大、版本升级需要重新下载。

第三种是 Helm 安装，这是在 Kubernetes 集群中部署的标准方式。通过 Helm Chart 以声明式配置部署 Harbor，天然支持高可用和弹性伸缩。可以配置外部数据库和 Redis 集群，调整各组件副本数，充分利用 Kubernetes 的编排能力。这种方式适合云原生环境的生产部署，优点是配置管理规范、支持滚动升级、运维效率高，缺点是需要 Kubernetes 知识储备。

从架构上看，Harbor 采用微服务架构，核心组件包括 Nginx（入口代理）、Core（API 服务）、Registry（镜像存储）、Portal（Web UI）、PostgreSQL（元数据存储）、Redis（缓存）、Trivy（漏洞扫描）等。无论哪种安装方式，最终都是通过 Docker Compose 或 Kubernetes 编排这些组件容器。

在实际生产中，我建议：测试环境用在线安装快速验证；传统生产环境用离线安装保证稳定性；云原生环境用 Helm 安装实现高可用和自动化运维。同时要注意配置 HTTPS 证书、启用镜像扫描和签名、设置合理的标签保留策略和垃圾回收计划，这些都是生产环境的关键配置。
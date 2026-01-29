# Docker生产环境配置

## 概述

在生产环境中部署Docker需要更加严格的配置和最佳实践。本文介绍Docker生产环境的安装配置、性能优化、安全加固以及运维管理等方面的内容。

## 系统要求与安装

### 系统要求

```bash
# 检查系统要求
# 1. 64位Linux系统
uname -m  # 应该显示x86_64或aarch64

# 2. 内核版本 >= 3.10（建议4.x+）
uname -r

# 3. 存储驱动支持
# overlay2需要内核4.0+或3.10.0-514+
cat /proc/filesystems | grep overlay

# 4. cgroup支持
cat /proc/cgroups
```

### 生产环境安装

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io

# CentOS/RHEL
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install -y docker-ce docker-ce-cli containerd.io
sudo systemctl enable docker
sudo systemctl start docker
```

## Docker Daemon配置

### 生产环境daemon.json

```json
// /etc/docker/daemon.json
{
  // 存储配置
  "storage-driver": "overlay2",
  "data-root": "/data/docker",

  // 日志配置
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "5",
    "compress": "true"
  },

  // 网络配置
  "bip": "172.17.0.1/16",
  "fixed-cidr": "172.17.0.0/16",
  "dns": ["8.8.8.8", "8.8.4.4"],

  // 安全配置
  "live-restore": true,
  "userland-proxy": false,
  "no-new-privileges": true,

  // 资源限制
  "default-ulimits": {
    "nofile": {
      "Name": "nofile",
      "Hard": 65535,
      "Soft": 65535
    },
    "nproc": {
      "Name": "nproc",
      "Hard": 65535,
      "Soft": 65535
    }
  },

  // 注册表配置
  "insecure-registries": [],
  "registry-mirrors": [
    "https://mirror.ccs.tencentyun.com"
  ],

  // 其他
  "max-concurrent-downloads": 10,
  "max-concurrent-uploads": 5,
  "debug": false,
  "experimental": false
}
```

### 重要配置说明

```bash
# live-restore: 允许Docker daemon重启时保持容器运行
# 非常重要！避免daemon升级时中断服务
"live-restore": true

# storage-driver: 存储驱动
# overlay2是现代Linux的最佳选择
"storage-driver": "overlay2"

# data-root: Docker数据目录
# 建议使用独立的高性能存储
"data-root": "/data/docker"

# userland-proxy: 用户空间代理
# 禁用以提高性能，使用iptables直接转发
"userland-proxy": false
```

### 应用配置

```bash
# 验证配置
sudo dockerd --validate

# 重启Docker
sudo systemctl restart docker

# 验证配置生效
docker info
```

## 系统优化

### 内核参数

```bash
# /etc/sysctl.d/docker.conf
# 网络优化
net.ipv4.ip_forward = 1
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1

# 连接跟踪
net.netfilter.nf_conntrack_max = 1048576
net.nf_conntrack_max = 1048576

# TCP优化
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.core.netdev_max_backlog = 65535

# 文件描述符
fs.file-max = 1048576
fs.inotify.max_user_instances = 8192
fs.inotify.max_user_watches = 524288

# 内存
vm.swappiness = 10
vm.max_map_count = 262144

# 应用配置
sudo sysctl --system
```

### 文件描述符限制

```bash
# /etc/security/limits.d/docker.conf
*       soft    nofile      65535
*       hard    nofile      65535
*       soft    nproc       65535
*       hard    nproc       65535

# Docker服务限制
# /etc/systemd/system/docker.service.d/limits.conf
[Service]
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity

sudo systemctl daemon-reload
sudo systemctl restart docker
```

### 存储优化

```bash
# 使用SSD存储
# 确保/data/docker在SSD上

# XFS文件系统（推荐用于overlay2）
mkfs.xfs -n ftype=1 /dev/sdb1
mount -o noatime /dev/sdb1 /data/docker

# 定期清理
# 创建定时任务
# /etc/cron.daily/docker-cleanup
#!/bin/bash
docker system prune -f --filter "until=24h"
docker volume prune -f
```

## 高可用配置

### Docker Swarm模式

```bash
# 初始化Swarm
docker swarm init --advertise-addr 192.168.1.100

# 添加manager节点
docker swarm join-token manager
# 在其他节点执行输出的命令

# 添加worker节点
docker swarm join-token worker

# 查看节点状态
docker node ls
```

### 服务部署

```yaml
# docker-compose.yml
version: '3.8'

services:
  web:
    image: nginx:1.25
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
        failure_action: rollback
      rollback_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
        window: 120s
      placement:
        constraints:
          - node.role == worker
        preferences:
          - spread: node.labels.zone
      resources:
        limits:
          cpus: '0.5'
          memory: 256M
        reservations:
          cpus: '0.25'
          memory: 128M
    ports:
      - "80:80"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

```bash
# 部署服务
docker stack deploy -c docker-compose.yml myapp

# 查看服务状态
docker service ls
docker service ps myapp_web

# 扩缩容
docker service scale myapp_web=5

# 更新服务
docker service update --image nginx:1.26 myapp_web
```

## 监控配置

### 启用metrics

```json
// /etc/docker/daemon.json
{
  "metrics-addr": "0.0.0.0:9323",
  "experimental": true
}
```

### Prometheus配置

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'docker'
    static_configs:
      - targets: ['localhost:9323']

  - job_name: 'cadvisor'
    static_configs:
      - targets: ['localhost:8080']
```

### cAdvisor部署

```bash
docker run -d \
  --name cadvisor \
  --restart always \
  -p 8080:8080 \
  -v /:/rootfs:ro \
  -v /var/run:/var/run:ro \
  -v /sys:/sys:ro \
  -v /var/lib/docker:/var/lib/docker:ro \
  gcr.io/cadvisor/cadvisor:latest
```

## 安全加固

### TLS配置

```bash
# 生成CA证书
openssl genrsa -aes256 -out ca-key.pem 4096
openssl req -new -x509 -days 365 -key ca-key.pem -sha256 -out ca.pem

# 生成服务器证书
openssl genrsa -out server-key.pem 4096
openssl req -subj "/CN=docker-host" -sha256 -new -key server-key.pem -out server.csr

echo subjectAltName = DNS:docker-host,IP:192.168.1.100,IP:127.0.0.1 >> extfile.cnf
echo extendedKeyUsage = serverAuth >> extfile.cnf

openssl x509 -req -days 365 -sha256 -in server.csr \
  -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
  -out server-cert.pem -extfile extfile.cnf

# 配置Docker daemon
# /etc/docker/daemon.json
{
  "tls": true,
  "tlsverify": true,
  "tlscacert": "/etc/docker/certs/ca.pem",
  "tlscert": "/etc/docker/certs/server-cert.pem",
  "tlskey": "/etc/docker/certs/server-key.pem",
  "hosts": ["unix:///var/run/docker.sock", "tcp://0.0.0.0:2376"]
}
```

### 用户命名空间

```json
// /etc/docker/daemon.json
{
  "userns-remap": "default"
}
```

```bash
# 配置subordinate UID/GID
echo "dockremap:100000:65536" >> /etc/subuid
echo "dockremap:100000:65536" >> /etc/subgid

sudo systemctl restart docker
```

### 安全审计

```bash
# 审计Docker文件和命令
# /etc/audit/rules.d/docker.rules
-w /usr/bin/docker -k docker
-w /var/lib/docker -k docker
-w /etc/docker -k docker
-w /usr/lib/systemd/system/docker.service -k docker
-w /etc/docker/daemon.json -k docker

sudo auditctl -R /etc/audit/rules.d/docker.rules
```

## 备份与恢复

### 数据备份策略

```bash
#!/bin/bash
# /opt/scripts/docker-backup.sh

BACKUP_DIR=/backup/docker
DATE=$(date +%Y%m%d)

# 备份卷
for volume in $(docker volume ls -q); do
    docker run --rm \
        -v $volume:/source:ro \
        -v $BACKUP_DIR:/backup \
        alpine tar czf /backup/${volume}_${DATE}.tar.gz -C /source .
done

# 备份compose文件
cp -r /opt/docker-compose $BACKUP_DIR/compose_${DATE}

# 备份daemon配置
cp /etc/docker/daemon.json $BACKUP_DIR/daemon_${DATE}.json

# 清理旧备份（保留7天）
find $BACKUP_DIR -type f -mtime +7 -delete
```

### 恢复流程

```bash
#!/bin/bash
# 恢复卷数据
docker volume create myvolume
docker run --rm \
    -v myvolume:/target \
    -v /backup:/backup:ro \
    alpine tar xzf /backup/myvolume_20240115.tar.gz -C /target
```

## 日志管理

### 集中式日志

```yaml
# docker-compose.yml with logging
version: '3.8'

services:
  app:
    image: myapp
    logging:
      driver: fluentd
      options:
        fluentd-address: "fluentd:24224"
        tag: "docker.{{.Name}}"
        fluentd-async: "true"

  fluentd:
    image: fluent/fluentd:v1.16
    ports:
      - "24224:24224"
    volumes:
      - ./fluent.conf:/fluentd/etc/fluent.conf
```

### 日志轮转

```json
// /etc/docker/daemon.json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "5",
    "compress": "true"
  }
}
```

## 实用命令

```bash
# 生产环境检查清单
docker info                              # 查看配置
docker system df                         # 查看磁盘使用
docker stats                             # 查看资源使用
docker system events                     # 监控事件

# 健康检查
curl -s http://localhost:9323/metrics    # Docker metrics
docker ps --filter health=unhealthy      # 不健康的容器

# 维护操作
docker system prune -a --volumes -f      # 清理
docker update --restart=always $(docker ps -q)  # 更新重启策略
```

## 常见问题

### Q1: 如何平滑升级Docker？

```bash
# 1. 确保live-restore已启用
docker info | grep "Live Restore"

# 2. 备份当前配置
cp /etc/docker/daemon.json /etc/docker/daemon.json.bak

# 3. 升级Docker
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io

# 4. 验证容器仍在运行
docker ps

# 5. 验证版本
docker version
```

### Q2: 生产环境如何处理容器日志？

```bash
# 1. 配置日志轮转
# 2. 使用集中式日志系统（EFK/PLG）
# 3. 应用输出到stdout/stderr
# 4. 避免在容器内写日志文件

# 示例配置
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "5"
  }
}
```

### Q3: 如何实现零停机部署？

```bash
# 使用Swarm服务
docker service update \
  --update-parallelism 1 \
  --update-delay 10s \
  --update-failure-action rollback \
  --image newimage:tag \
  myservice

# 或使用蓝绿部署
# 1. 启动新版本容器
# 2. 健康检查通过后切换流量
# 3. 停止旧版本容器
```

### Q4: 如何监控Docker宿主机资源？

```bash
# 1. 安装node-exporter
docker run -d \
  --name node-exporter \
  --net host \
  --pid host \
  -v /:/host:ro,rslave \
  prom/node-exporter \
  --path.rootfs=/host

# 2. 配置Prometheus抓取
# 3. 在Grafana创建Dashboard
```

### Q5: 生产环境Docker网络如何规划？

```bash
# 1. 使用自定义bridge网络
docker network create --subnet 10.10.0.0/16 production

# 2. 为不同应用创建独立网络
docker network create --internal backend    # 内部网络
docker network create frontend              # 对外网络

# 3. 多主机使用overlay网络
docker network create --driver overlay --attachable prod-overlay
```

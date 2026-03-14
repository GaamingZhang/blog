---
date: 2026-03-12
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Node
  - 故障排查
---

# Kubernetes 节点 NotReady 状态全面排查指南

## 引言：节点 NotReady 的严重影响

在 Kubernetes 集群运维中，节点状态为 NotReady 是最常见也是最棘手的问题之一。当节点进入 NotReady 状态时，意味着该节点无法正常工作，kubelet 无法与 API Server 正常通信，或者节点上的容器运行时出现异常。这会导致：

- **Pod 调度失败**：调度器不会将新的 Pod 调度到 NotReady 节点上
- **服务中断风险**：节点上的 Pod 可能处于未知状态，影响业务连续性
- **数据丢失隐患**：如果节点长时间 NotReady，可能触发 Pod 驱逐，导致数据丢失
- **集群不稳定**：多个节点 NotReady 可能导致集群整体可用性下降

理解 NotReady 的各种原因及其排查方法，是每个 Kubernetes 运维工程师的必备技能。本文将深入剖析导致节点 NotReady 的各种场景，提供系统化的排查思路和解决方案。

## 一、kubelet 异常

### 1.1 现象描述

当 kubelet 进程异常时，节点状态会迅速变为 NotReady。具体表现为：

```bash
$ kubectl get nodes
NAME      STATUS     ROLES    AGE   VERSION
worker-1  NotReady   <none>   10d   v1.28.0
```

查看节点详情，Conditions 中 NodeReady 状态为 Unknown 或 False：

```bash
$ kubectl describe node worker-1
Conditions:
  Type                 Status  LastHeartbeatTime                 Reason                       Message
  ----                 ------  -----------------                 ------                       -------
  MemoryPressure       False   Wed, 12 Mar 2026 10:00:00 +0800   KubeletHasSufficientMemory   kubelet has sufficient memory available
  DiskPressure         False   Wed, 12 Mar 2026 10:00:00 +0800   KubeletHasNoDiskPressure     kubelet has no disk pressure
  PIDPressure          False   Wed, 12 Mar 2026 10:00:00 +0800   KubeletHasSufficientPID      kubelet has sufficient PID available
  Ready                False   Wed, 12 Mar 2026 10:00:00 +0800   KubeletNotReady              kubelet is not posting ready status
```

### 1.2 排查步骤

**步骤 1：检查 kubelet 服务状态**

```bash
# 检查 kubelet 服务是否运行
systemctl status kubelet

# 查看服务详细信息
systemctl status kubelet -l
```

**步骤 2：查看 kubelet 日志**

```bash
# 查看最近 100 行日志
journalctl -u kubelet -n 100 --no-pager

# 实时跟踪日志
journalctl -u kubelet -f

# 查看错误级别日志
journalctl -u kubelet -p err
```

**步骤 3：检查 kubelet 配置文件**

```bash
# 检查配置文件是否存在
ls -la /var/lib/kubelet/config.yaml

# 验证配置文件语法
kubelet --config=/var/lib/kubelet/config.yaml --dry-run
```

**步骤 4：检查 kubelet 进程**

```bash
# 查看进程是否存在
ps aux | grep kubelet

# 检查进程资源占用
top -p $(pgrep kubelet)
```

### 1.3 常见原因与解决方案

#### 原因 1：kubelet 服务未启动

```bash
# 启动服务
systemctl start kubelet

# 设置开机自启
systemctl enable kubelet
```

#### 原因 2：kubelet 配置错误

配置文件中常见错误包括：
- API Server 地址配置错误
- 集群 DNS 配置错误
- 认证证书路径错误

```bash
# 检查配置文件
cat /var/lib/kubelet/config.yaml | grep -E "clusterDNS|apiVersion"

# 修正配置后重启服务
systemctl restart kubelet
```

#### 原因 3：kubelet 版本与集群不兼容

```bash
# 检查 kubelet 版本
kubelet --version

# 检查集群版本
kubectl version

# 版本差异应控制在 +/- 1 个小版本内
```

#### 原因 4：资源耗尽导致 kubelet OOM

```bash
# 检查系统内存
free -h

# 查看 OOM 日志
dmesg | grep -i "out of memory"

# 查看 kubelet 内存占用
ps aux | grep kubelet | awk '{print $6}'

# 解决方案：增加系统内存或调整 kubelet 资源限制
```

## 二、容器运行时异常

### 2.1 现象描述

容器运行时（Container Runtime）是 Kubernetes 运行容器的基础设施。当容器运行时异常时，kubelet 无法管理节点上的容器，导致节点 NotReady。

```bash
$ kubectl describe node worker-1
Conditions:
  Ready            False   Wed, 12 Mar 2026 10:00:00 +0800   KubeletNotReady              container runtime is not ready
```

### 2.2 排查步骤

**步骤 1：检查容器运行时状态**

对于 Docker：

```bash
# 检查 Docker 服务状态
systemctl status docker

# 查看 Docker 信息
docker info

# 检查 Docker 是否响应
docker ps
```

对于 Containerd：

```bash
# 检查 containerd 服务状态
systemctl status containerd

# 使用 ctr 检查
ctr --address /run/containerd/containerd.sock version

# 使用 crictl 检查
crictl --runtime-endpoint unix:///run/containerd/containerd.sock ps
```

**步骤 2：检查容器运行时日志**

```bash
# Docker 日志
journalctl -u docker -n 200

# Containerd 日志
journalctl -u containerd -n 200
```

**步骤 3：检查容器运行时 Socket**

```bash
# 检查 Docker socket
ls -la /var/run/docker.sock

# 检查 Containerd socket
ls -la /run/containerd/containerd.sock

# 测试 socket 连接
curl --unix-socket /var/run/docker.sock http://localhost/version
```

**步骤 4：检查 kubelet 与运行时的连接**

```bash
# 查看 kubelet 配置中的运行时端点
cat /var/lib/kubelet/config.yaml | grep -A 5 containerRuntime

# 检查 crictl 配置
cat /etc/crictl.yaml
```

### 2.3 常见原因与解决方案

#### 原因 1：容器运行时服务停止

```bash
# 启动 Docker
systemctl start docker
systemctl enable docker

# 启动 Containerd
systemctl start containerd
systemctl enable containerd
```

#### 原因 2：容器运行时配置错误

**Docker 配置错误示例：**

```bash
# 检查 Docker daemon 配置
cat /etc/docker/daemon.json

# 常见错误：JSON 格式错误、配置项不合法
# 验证配置
dockerd --validate

# 重启服务
systemctl restart docker
```

**Containerd 配置错误示例：**

```bash
# 检查 containerd 配置
cat /etc/containerd/config.toml

# 验证配置
containerd config dump

# 重启服务
systemctl restart containerd
```

#### 原因 3：运行时 Socket 文件丢失

```bash
# 检查 socket 文件权限
ls -la /var/run/docker.sock
ls -la /run/containerd/containerd.sock

# 如果 socket 不存在，重启容器运行时
systemctl restart docker
# 或
systemctl restart containerd
```

#### 原因 4：容器运行时版本不兼容

```bash
# 检查运行时版本
docker version
# 或
containerd --version

# 检查 Kubernetes 支持的运行时版本
# 参考：https://kubernetes.io/docs/setup/production-environment/container-runtimes/
```

#### 原因 5：容器运行时资源耗尽

```bash
# 检查容器数量
docker ps -a | wc -l

# 检查镜像占用空间
docker system df

# 清理无用资源
docker system prune -a

# 检查运行时进程资源占用
ps aux | grep -E "dockerd|containerd"
```

## 三、网络问题

### 3.1 现象描述

网络问题是导致节点 NotReady 的常见原因，特别是节点与 API Server 之间的网络通信异常。

```bash
$ kubectl get nodes
NAME      STATUS     ROLES    AGE   VERSION
worker-1  NotReady   <none>   10d   v1.28.0

$ kubectl describe node worker-1
Conditions:
  Ready            Unknown   Wed, 12 Mar 2026 10:00:00 +0800   NodeStatusUnknown   Kubelet stopped posting status.
```

### 3.2 排查步骤

**步骤 1：检查节点与 API Server 的网络连通性**

```bash
# 获取 API Server 地址
kubectl cluster-info

# 从节点 ping API Server
ping <api-server-ip>

# 检查端口连通性
telnet <api-server-ip> 6443
# 或
nc -zv <api-server-ip> 6443

# 使用 curl 测试 HTTPS 连接
curl -k https://<api-server-ip>:6443/healthz
```

**步骤 2：检查防火墙规则**

```bash
# 检查 iptables 规则
iptables -L -n -v | grep 6443

# 检查 firewalld 状态
systemctl status firewalld

# 查看防火墙规则
firewall-cmd --list-all
```

**步骤 3：检查网络插件状态**

```bash
# 查看 CNI 插件配置
ls -la /etc/cni/net.d/

# 查看网络插件 Pod 状态
kubectl get pods -n kube-system | grep -E "calico|flannel|weave"

# 查看网络插件日志
kubectl logs -n kube-system <network-plugin-pod>
```

**步骤 4：检查节点网络接口**

```bash
# 查看网络接口
ip addr show

# 检查路由表
ip route show

# 检查 DNS 解析
nslookup kubernetes.default.svc.cluster.local
```

**步骤 5：检查 kube-proxy**

```bash
# 查看 kube-proxy Pod
kubectl get pods -n kube-system -l k8s-app=kube-proxy

# 查看 kube-proxy 日志
kubectl logs -n kube-system <kube-proxy-pod>

# 检查 iptables 规则是否由 kube-proxy 生成
iptables -t nat -L KUBE-SERVICES -n
```

### 3.3 常见原因与解决方案

#### 原因 1：防火墙阻止通信

```bash
# 临时关闭防火墙（测试用）
systemctl stop firewalld

# 永久开放必要端口
firewall-cmd --permanent --add-port=6443/tcp
firewall-cmd --permanent --add-port=10250/tcp
firewall-cmd --permanent --add-port=10251/tcp
firewall-cmd --permanent --add-port=10252/tcp
firewall-cmd --reload

# 或使用 iptables
iptables -I INPUT -p tcp --dport 6443 -j ACCEPT
iptables -I INPUT -p tcp --dport 10250 -j ACCEPT
service iptables save
```

#### 原因 2：网络插件异常

```bash
# 重启网络插件 Pod
kubectl delete pod -n kube-system <network-plugin-pod>

# 检查 CNI 配置文件
cat /etc/cni/net.d/*.conflist

# 确保 CNI 二进制文件存在
ls -la /opt/cni/bin/
```

#### 原因 3：节点 IP 变更

```bash
# 检查节点当前 IP
ip addr show

# 查看节点注册的 IP
kubectl get node worker-1 -o jsonpath='{.status.addresses}'

# 如果 IP 变更，需要更新节点配置
# 编辑 kubelet 配置
vim /var/lib/kubelet/config.yaml
# 修改 nodeIP 或使用 --node-ip 参数

# 重启 kubelet
systemctl restart kubelet
```

#### 原因 4：API Server 负载过高

```bash
# 检查 API Server 状态
kubectl get pods -n kube-system -l component=kube-apiserver

# 查看 API Server 日志
kubectl logs -n kube-system <apiserver-pod> --tail=100

# 检查 API Server 资源使用
kubectl top pod -n kube-system <apiserver-pod>

# 检查 API Server 指标
curl -k https://<api-server>:6443/metrics | grep apiserver_request_duration_seconds
```

#### 原因 5：DNS 解析失败

```bash
# 检查节点 DNS 配置
cat /etc/resolv.conf

# 测试 DNS 解析
nslookup kubernetes.default

# 检查 CoreDNS
kubectl get pods -n kube-system -l k8s-app=kube-dns

# 查看 CoreDNS 日志
kubectl logs -n kube-system <coredns-pod>
```

## 四、资源压力

### 4.1 现象描述

当节点资源（磁盘、内存、PID）不足时，kubelet 会触发资源压力机制，导致节点状态变为 NotReady。

```bash
$ kubectl describe node worker-1
Conditions:
  MemoryPressure   True    Wed, 12 Mar 2026 10:00:00 +0800   KubeletHasInsufficientMemory   kubelet has insufficient memory available
  DiskPressure     True    Wed, 12 Mar 2026 10:00:00 +0800   KubeletHasDiskPressure         kubelet has disk pressure
  PIDPressure      True    Wed, 12 Mar 2026 10:00:00 +0800   KubeletHasInsufficientPID      kubelet has insufficient PID available
  Ready            False   Wed, 12 Mar 2026 10:00:00 +0800   KubeletNotReady                kubelet is posting not-ready status
```

### 4.2 磁盘压力排查

**步骤 1：检查磁盘使用情况**

```bash
# 查看磁盘使用率
df -h

# 查看 inode 使用率
df -i

# 查看大文件
du -sh /* | sort -rh | head -20

# 查看容器存储使用情况
docker system df
```

**步骤 2：检查 kubelet 阈值配置**

```bash
# 查看磁盘阈值配置
cat /var/lib/kubelet/config.yaml | grep -A 10 eviction

# 常见配置项：
# evictionHard:
#   memory.available: "100Mi"
#   nodefs.available: "10%"
#   nodefs.inodesFree: "5%"
#   imagefs.available: "15%"
```

**步骤 3：清理磁盘空间**

```bash
# 清理 Docker 资源
docker system prune -a --volumes

# 清理无用镜像
docker image prune -a

# 清理退出容器
docker container prune

# 清理日志文件
find /var/log -type f -name "*.log" -mtime +7 -delete

# 清理容器日志
truncate -s 0 /var/lib/docker/containers/*/*-json.log
```

**步骤 4：配置日志轮转**

```bash
# 配置 Docker 日志轮转
cat > /etc/docker/daemon.json <<EOF
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "3"
  }
}
EOF

# 重启 Docker
systemctl restart docker
```

### 4.3 内存压力排查

**步骤 1：检查内存使用情况**

```bash
# 查看内存使用
free -h

# 查看详细内存信息
cat /proc/meminfo

# 查看进程内存占用
ps aux --sort=-%mem | head -20

# 查看容器内存占用
docker stats --no-stream
```

**步骤 2：检查 Pod 资源使用**

```bash
# 查看节点上 Pod 资源使用
kubectl describe node worker-1 | grep -A 10 "Allocated resources"

# 使用 metrics-server
kubectl top node
kubectl top pod --all-namespaces --sort-by=memory
```

**步骤 3：检查内存阈值**

```bash
# 查看可用内存
cat /proc/meminfo | grep MemAvailable

# 计算可用内存百分比
echo "scale=2; $(cat /proc/meminfo | grep MemAvailable | awk '{print $2}') / $(cat /proc/meminfo | grep MemTotal | awk '{print $2}') * 100" | bc
```

**步骤 4：释放内存**

```bash
# 清理缓存（谨慎使用）
sync && echo 3 > /proc/sys/vm/drop_caches

# 驱逐低优先级 Pod
kubectl delete pod <low-priority-pod> --grace-period=0 --force

# 调整 kubelet 内存预留
cat >> /var/lib/kubelet/config.yaml <<EOF
systemReserved:
  memory: "2Gi"
kubeReserved:
  memory: "1Gi"
evictionHard:
  memory.available: "500Mi"
EOF

# 重启 kubelet
systemctl restart kubelet
```

### 4.4 PID 压力排查

**步骤 1：检查 PID 使用情况**

```bash
# 查看当前 PID 数量
ps -e | wc -l

# 查看最大 PID 数量
cat /proc/sys/kernel/pid_max

# 查看进程树
pstree -p | head -50

# 查看进程数量最多的用户
ps -eo user | sort | uniq -c | sort -rn
```

**步骤 2：检查容器 PID 数量**

```bash
# 查看容器进程数
for container in $(docker ps -q); do
  echo "Container: $container"
  docker top $container | wc -l
done

# 查看特定容器的进程
docker top <container-id>
```

**步骤 3：调整 PID 限制**

```bash
# 临时增加系统 PID 限制
echo 4194303 > /proc/sys/kernel/pid_max

# 永久设置
echo "kernel.pid_max = 4194303" >> /etc/sysctl.conf
sysctl -p

# 配置 kubelet PID 阈值
cat >> /var/lib/kubelet/config.yaml <<EOF
evictionHard:
  pid.available: "10%"
EOF

# 重启 kubelet
systemctl restart kubelet
```

## 五、证书过期

### 5.1 现象描述

Kubernetes 集群使用 TLS 证书进行安全通信。当 kubelet 证书过期时，节点无法与 API Server 建立安全连接，导致节点 NotReady。

```bash
$ journalctl -u kubelet -n 50
Mar 12 10:00:00 worker-1 kubelet[12345]: E0312 10:00:00.123456 12345 reflector.go:178] k8s.io/client-go/informers/factory.go:135: Failed to list *v1.Node: Get "https://10.0.0.1:6443/api/v1/nodes?resourceVersion=0": x509: certificate has expired or is not yet valid
```

### 5.2 排查步骤

**步骤 1：检查证书有效期**

```bash
# 检查 kubelet 客户端证书
openssl x509 -in /var/lib/kubelet/pki/kubelet-client-current.pem -noout -text | grep -A 2 Validity

# 检查 kubelet 服务端证书
openssl x509 -in /var/lib/kubelet/pki/kubelet.crt -noout -text | grep -A 2 Validity

# 使用 kubeadm 检查所有证书
kubeadm certs check-expiration
```

**步骤 2：检查 API Server 证书**

```bash
# 检查 API Server 证书
openssl s_client -connect <api-server-ip>:6443 -showcerts </dev/null 2>/dev/null | openssl x509 -noout -text | grep -A 2 Validity

# 检查 CA 证书
openssl x509 -in /etc/kubernetes/pki/ca.crt -noout -text | grep -A 2 Validity
```

**步骤 3：检查系统时间**

```bash
# 检查系统时间
date

# 检查时区
timedatectl

# 检查 NTP 同步状态
timedatectl status
```

### 5.3 解决方案

#### 方案 1：自动续期（kubeadm 部署）

```bash
# 续期所有证书
kubeadm certs renew all

# 仅续期 kubelet 证书
kubeadm certs renew kubelet-apiserver-client

# 重启 kubelet
systemctl restart kubelet
```

#### 方案 2：手动生成证书

```bash
# 生成 CSR
openssl req -new -key /var/lib/kubelet/pki/kubelet-client-current.pem -out kubelet-client.csr -subj "/O=system:nodes/CN=system:node:worker-1"

# 使用 CA 签名
openssl x509 -req -in kubelet-client.csr -CA /etc/kubernetes/pki/ca.crt -CAkey /etc/kubernetes/pki/ca.key -CAcreateserial -out kubelet-client.crt -days 365

# 更新证书
cp kubelet-client.crt /var/lib/kubelet/pki/kubelet-client-current.pem

# 重启 kubelet
systemctl restart kubelet
```

#### 方案 3：启用证书自动轮转

```bash
# 检查 kubelet 配置
cat /var/lib/kubelet/config.yaml | grep rotateCertificates

# 启用自动轮转
echo "rotateCertificates: true" >> /var/lib/kubelet/config.yaml

# 或在 kubelet 启动参数中添加
# --rotate-certificates=true

# 重启 kubelet
systemctl restart kubelet

# 检查 CSR
kubectl get csr
kubectl certificate approve <csr-name>
```

#### 方案 4：修复时间同步问题

```bash
# 安装并启动 NTP
yum install -y ntp
systemctl start ntpd
systemctl enable ntpd

# 或使用 chrony
yum install -y chrony
systemctl start chronyd
systemctl enable chronyd

# 手动同步时间
ntpdate pool.ntp.org

# 或
chronyc makestep
```

## 六、时钟不同步

### 6.1 现象描述

时钟不同步会导致证书验证失败、日志时间戳混乱、调度异常等问题，严重时会导致节点 NotReady。

```bash
$ journalctl -u kubelet -n 50
Mar 12 10:00:00 worker-1 kubelet[12345]: E0312 10:00:00.123456 12345 reflector.go:178] clock skew detected
```

### 6.2 排查步骤

**步骤 1：检查系统时间**

```bash
# 查看当前时间
date

# 查看详细时间信息
timedatectl

# 检查时区
ls -la /etc/localtime
```

**步骤 2：检查时间同步状态**

```bash
# 检查 NTP 状态
timedatectl status | grep "NTP synchronized"

# 检查 NTP 服务
systemctl status ntpd
# 或
systemctl status chronyd

# 查看 NTP 服务器
ntpq -p
# 或
chronyc sources
```

**步骤 3：检查时间偏差**

```bash
# 比较节点时间与 API Server 时间
date && ssh master-1 date

# 检查最大时间偏差配置
cat /var/lib/kubelet/config.yaml | grep -A 5 "featureGates"
```

### 6.3 解决方案

#### 方案 1：配置 NTP 时间同步

```bash
# 安装 NTP
yum install -y ntp

# 配置 NTP 服务器
cat > /etc/ntp.conf <<EOF
server 0.pool.ntp.org iburst
server 1.pool.ntp.org iburst
server 2.pool.ntp.org iburst
server 3.pool.ntp.org iburst
EOF

# 启动服务
systemctl start ntpd
systemctl enable ntpd

# 验证同步
ntpq -p
```

#### 方案 2：配置 Chrony 时间同步

```bash
# 安装 Chrony
yum install -y chrony

# 配置时间服务器
cat > /etc/chrony.conf <<EOF
server 0.pool.ntp.org iburst
server 1.pool.ntp.org iburst
server 2.pool.ntp.org iburst
server 3.pool.ntp.org iburst
driftfile /var/lib/chrony/drift
makestep 1.0 3
rtcsync
logdir /var/log/chrony
EOF

# 启动服务
systemctl start chronyd
systemctl enable chronyd

# 验证同步
chronyc tracking
chronyc sources
```

#### 方案 3：手动同步时间

```bash
# 停止 NTP 服务
systemctl stop ntpd

# 手动同步
ntpdate pool.ntp.org

# 重启 NTP 服务
systemctl start ntpd
```

#### 方案 4：配置时区

```bash
# 查看可用时区
timedatectl list-timezones | grep Asia

# 设置时区
timedatectl set-timezone Asia/Shanghai

# 验证
date
```

## 七、排查流程图

以下是系统化的 NotReady 节点排查流程：

```
开始
  |
  v
检查节点状态
kubectl get nodes
  |
  v
节点是否 NotReady?
  |-- 否 --> 正常，结束
  |-- 是 --> 继续
  |
  v
查看节点详情
kubectl describe node <node-name>
  |
  v
检查 Conditions
  |
  +-- MemoryPressure=True --> 检查内存 --> 清理资源/扩容
  |
  +-- DiskPressure=True --> 检查磁盘 --> 清理空间/扩容
  |
  +-- PIDPressure=True --> 检查 PID --> 调整限制/清理进程
  |
  +-- Ready=False/Unknown --> 继续
  |
  v
检查 kubelet 服务
systemctl status kubelet
  |
  +-- 未运行 --> 启动服务
  |
  +-- 运行中 --> 查看日志
  |
  v
检查 kubelet 日志
journalctl -u kubelet -n 100
  |
  +-- 证书错误 --> 检查证书有效期 --> 续期证书
  |
  +-- 连接超时 --> 检查网络连通性
  |
  +-- 运行时错误 --> 检查容器运行时
  |
  v
检查容器运行时
systemctl status docker/containerd
  |
  +-- 未运行 --> 启动服务
  |
  +-- 运行中 --> 检查 socket/配置
  |
  v
检查网络连通性
ping <api-server-ip>
telnet <api-server-ip> 6443
  |
  +-- 不通 --> 检查防火墙/网络插件
  |
  +-- 通 --> 检查时间同步
  |
  v
检查时间同步
date && timedatectl status
  |
  +-- 不同步 --> 配置 NTP/Chrony
  |
  +-- 同步 --> 深入排查日志
  |
  v
问题解决
验证节点状态
kubectl get nodes
```

## 八、原因分类汇总表

| 分类 | 具体原因 | 关键现象 | 排查命令 | 解决方案 |
|------|---------|---------|---------|---------|
| **kubelet 异常** | 服务未启动 | systemctl status kubelet 显示 inactive/dead | `systemctl status kubelet` | 启动并设置开机自启 |
| | 配置错误 | kubelet 启动失败，配置文件语法错误 | `kubelet --config=xxx --dry-run` | 修正配置文件 |
| | 版本不兼容 | kubelet 版本与集群差异过大 | `kubelet --version` | 升级或降级 kubelet |
| | 资源耗尽 | OOM killed，内存不足 | `dmesg \| grep OOM` | 增加内存或调整资源限制 |
| **容器运行时异常** | 服务停止 | docker/containerd 服务未运行 | `systemctl status docker` | 启动容器运行时服务 |
| | 配置错误 | daemon.json 配置格式错误 | `dockerd --validate` | 修正配置文件 |
| | Socket 丢失 | socket 文件不存在 | `ls -la /var/run/docker.sock` | 重启容器运行时 |
| | 版本不兼容 | 运行时版本不支持 | `docker version` | 升级容器运行时 |
| **网络问题** | 防火墙阻止 | 无法连接 API Server 6443 端口 | `telnet <ip> 6443` | 开放必要端口 |
| | 网络插件异常 | CNI 配置错误或 Pod 异常 | `kubectl get pods -n kube-system` | 重启网络插件 |
| | 节点 IP 变更 | 节点注册 IP 与实际 IP 不符 | `ip addr show` | 更新 kubelet 配置 |
| | DNS 解析失败 | 无法解析 kubernetes.default | `nslookup kubernetes.default` | 修复 DNS 配置 |
| **资源压力** | 磁盘不足 | DiskPressure=True | `df -h` | 清理磁盘空间 |
| | 内存不足 | MemoryPressure=True | `free -h` | 清理内存或扩容 |
| | PID 不足 | PIDPressure=True | `ps -e \| wc -l` | 调整 PID 限制 |
| **证书过期** | kubelet 证书过期 | x509: certificate has expired | `openssl x509 -in xxx -noout -text` | 续期证书 |
| | CA 证书过期 | 集群证书全部过期 | `kubeadm certs check-expiration` | 续期所有证书 |
| **时钟不同步** | NTP 服务停止 | 时间偏差过大 | `timedatectl status` | 启动 NTP 服务 |
| | 时区错误 | 时间与实际相差数小时 | `date` | 设置正确时区 |

## 九、常见问题与最佳实践

### 9.1 常见问题

#### Q1：节点 NotReady 后，Pod 会怎样？

**A：** 节点 NotReady 后，Pod 的处理分为两个阶段：

1. **宽限期内（默认 40 秒）**：Pod 继续运行，状态为 Running
2. **宽限期后**：进入 Unknown 状态，等待 `pod-eviction-timeout`（默认 5 分钟）
3. **超时后**：触发 Pod 驱逐，控制器在其它节点重建 Pod

```bash
# 查看相关配置
kubectl describe node <node-name> | grep -A 5 Taint

# 查看 kube-controller-manager 配置
ps aux | grep kube-controller-manager | grep pod-eviction-timeout
```

#### Q2：如何避免节点 NotReady 导致服务中断？

**A：** 多层次保障措施：

1. **部署多副本**：确保关键服务至少 2 个副本
2. **Pod 反亲和性**：避免 Pod 集中在同一节点
3. **PDB（Pod Disruption Budget）**：限制同时中断的 Pod 数量
4. **监控告警**：及时发现节点异常
5. **自动扩缩容**：Cluster Autoscaler 自动补充节点

```yaml
# PDB 示例
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: nginx-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: nginx
```

#### Q3：节点频繁 NotReady 如何处理？

**A：** 系统化排查和优化：

1. **分析日志模式**：找出触发 NotReady 的根本原因
2. **资源规划**：确保节点资源充足
3. **网络优化**：检查网络稳定性
4. **组件升级**：修复已知 bug
5. **配置调优**：优化 kubelet 参数

```bash
# 查看节点历史事件
kubectl get events --field-selector involvedObject.kind=Node,involvedObject.name=<node-name>

# 查看节点状态变化历史
kubectl get node <node-name> -o jsonpath='{.status.conditions}' | jq
```

#### Q4：如何快速恢复 NotReady 节点？

**A：** 快速恢复流程：

```bash
# 1. 快速诊断
kubectl describe node <node-name>
systemctl status kubelet
systemctl status docker

# 2. 常见快速修复
# 重启 kubelet
systemctl restart kubelet

# 重启容器运行时
systemctl restart docker

# 清理资源
docker system prune -f

# 3. 验证恢复
kubectl get node <node-name>
```

#### Q5：如何预防节点 NotReady？

**A：** 预防措施清单：

| 预防措施 | 实施方法 | 检查频率 |
|---------|---------|---------|
| 资源监控 | Prometheus + AlertManager | 实时 |
| 磁盘清理 | 定时任务清理日志和镜像 | 每周 |
| 证书管理 | 自动续期 + 过期告警 | 每月检查 |
| 时间同步 | NTP/Chrony 自动同步 | 持续 |
| 健康检查 | Node Problem Detector | 实时 |
| 容量规划 | 定期评估资源使用趋势 | 每月 |

### 9.2 最佳实践

#### 1. kubelet 配置优化

```yaml
# /var/lib/kubelet/config.yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration

# 资源预留
systemReserved:
  cpu: "500m"
  memory: "1Gi"
  ephemeral-storage: "10Gi"
kubeReserved:
  cpu: "500m"
  memory: "1Gi"
  ephemeral-storage: "10Gi"

# 驱逐阈值
evictionHard:
  memory.available: "500Mi"
  nodefs.available: "10%"
  nodefs.inodesFree: "5%"
  imagefs.available: "15%"
  pid.available: "10%"

# 软驱逐（更温和）
evictionSoft:
  memory.available: "750Mi"
  nodefs.available: "15%"
evictionSoftGracePeriod:
  memory.available: "1m30s"
  nodefs.available: "2m"

# 证书自动轮转
rotateCertificates: true

# 节点状态更新频率
nodeStatusUpdateFrequency: 10s
nodeStatusReportFrequency: 5m
```

#### 2. 容器运行时优化

```json
// /etc/docker/daemon.json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "3"
  },
  "storage-driver": "overlay2",
  "live-restore": true,
  "default-ulimits": {
    "nofile": {
      "Name": "nofile",
      "Hard": 65535,
      "Soft": 65535
    }
  }
}
```

#### 3. 监控告警配置

```yaml
# Prometheus 告警规则示例
groups:
- name: node-alerts
  rules:
  - alert: NodeNotReady
    expr: kube_node_status_condition{condition="Ready",status="true"} == 0
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Node {{ $labels.node }} is NotReady"
      description: "Node {{ $labels.node }} has been NotReady for more than 5 minutes."

  - alert: NodeDiskPressure
    expr: kube_node_status_condition{condition="DiskPressure",status="true"} == 1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Node {{ $labels.node }} has DiskPressure"
      description: "Node {{ $labels.node }} is experiencing disk pressure."

  - alert: NodeMemoryPressure
    expr: kube_node_status_condition{condition="MemoryPressure",status="true"} == 1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Node {{ $labels.node }} has MemoryPressure"
      description: "Node {{ $labels.node }} is experiencing memory pressure."

  - alert: CertificateExpiringSoon
    expr: kubelet_certificate_manager_client_expiration_renew_errors > 0
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "Kubelet certificate renewal failing on {{ $labels.node }}"
      description: "Kubelet on node {{ $labels.node }} is failing to renew its client certificate."
```

#### 4. 日志管理最佳实践

```bash
# 定时清理日志脚本
cat > /usr/local/bin/clean-logs.sh <<'EOF'
#!/bin/bash

# 清理 7 天前的系统日志
find /var/log -type f -name "*.log" -mtime +7 -delete
find /var/log -type f -name "*.gz" -mtime +7 -delete

# 清理 journal 日志
journalctl --vacuum-time=7d

# 清理 Docker 日志
for container in $(docker ps -aq); do
  log_path=$(docker inspect $container | grep LogPath | awk -F'"' '{print $4}')
  if [ -f "$log_path" ]; then
    truncate -s 0 "$log_path"
  fi
done

# 清理 Docker 无用资源
docker system prune -f
EOF

chmod +x /usr/local/bin/clean-logs.sh

# 配置定时任务
cat > /etc/cron.d/clean-logs <<EOF
0 2 * * 0 root /usr/local/bin/clean-logs.sh >> /var/log/clean-logs.log 2>&1
EOF
```

#### 5. Node Problem Detector 部署

```yaml
# 部署 Node Problem Detector 监控节点问题
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-problem-detector
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: node-problem-detector
  template:
    metadata:
      labels:
        app: node-problem-detector
    spec:
      serviceAccountName: node-problem-detector
      hostNetwork: true
      hostPID: true
      containers:
      - name: node-problem-detector
        image: k8s.gcr.io/node-problem-detector:v0.8.12
        securityContext:
          privileged: true
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        volumeMounts:
        - name: log
          mountPath: /var/log
          readOnly: true
        - name: kmsg
          mountPath: /dev/kmsg
          readOnly: true
      volumes:
      - name: log
        hostPath:
          path: /var/log/
      - name: kmsg
        hostPath:
          path: /dev/kmsg
```

## 十、总结

Kubernetes 节点 NotReady 是集群运维中最常见的问题之一，其根本原因可以归纳为六大类：kubelet 异常、容器运行时异常、网络问题、资源压力、证书过期和时钟不同步。每一类问题都有其特定的现象和排查方法。

排查的核心思路是：**从节点状态入手，通过 Conditions 定位问题类型，再结合日志和系统命令深入分析**。掌握系统化的排查流程，能够帮助我们快速定位问题根源，减少故障恢复时间。

预防胜于治疗。通过合理的资源规划、完善的监控告警、定期的证书管理、稳定的网络环境，可以有效降低节点 NotReady 的发生概率。同时，部署 Node Problem Detector、配置 PDB、实施多副本部署，可以在节点异常时保障服务的连续性。

---

## 面试回答

**面试官问：Kubernetes 集群节点状态为 NotReady 都有哪些情况？**

**回答：**

Kubernetes 节点 NotReady 主要有六大类原因：

第一，**kubelet 异常**，包括 kubelet 服务停止、配置错误、版本不兼容或资源耗尽导致 OOM。这是最直接的原因，kubelet 无法正常工作就无法向 API Server 汇报节点状态。

第二，**容器运行时异常**，Docker 或 Containerd 服务停止、配置错误、Socket 文件丢失或版本不兼容，导致 kubelet 无法管理节点上的容器。

第三，**网络问题**，防火墙阻止了 kubelet 与 API Server 的通信（特别是 6443 端口）、网络插件异常、节点 IP 变更或 DNS 解析失败，都会导致节点无法与控制平面通信。

第四，**资源压力**，磁盘空间不足触发 DiskPressure、内存不足触发 MemoryPressure、PID 资源耗尽触发 PIDPressure，kubelet 会主动将节点标记为 NotReady 并触发 Pod 驱逐。

第五，**证书过期**，kubelet 客户端证书或 CA 证书过期，导致 TLS 握手失败，无法建立与 API Server 的安全连接。

第六，**时钟不同步**，节点时间与集群时间偏差过大，导致证书验证失败或日志时间戳混乱。

排查时，首先通过 `kubectl describe node` 查看 Conditions 字段判断问题类型，然后检查 kubelet 服务状态和日志，接着排查容器运行时和网络连通性，最后检查证书有效期和时间同步状态。预防措施包括配置资源预留、设置监控告警、启用证书自动轮转、部署 NTP 时间同步等。

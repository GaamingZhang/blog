# Istio部署与金丝雀发布配置指南

本文档提供了在Kubernetes集群中部署Istio并为博客应用配置金丝雀发布的完整步骤。

## 目录

1. [环境准备](#环境准备)
2. [Istio安装](#istio安装)
3. [博客应用Istio配置](#博客应用istio配置)
4. [金丝雀部署配置](#金丝雀部署配置)
5. [验证与测试](#验证与测试)
6. [故障排查](#故障排查)

## 环境准备

### 前置条件

- Kubernetes集群版本 >= 1.21
- kubectl已配置并可以访问集群
- 至少4GB内存和2核CPU可用

### 1. 检查集群状态

```bash
# 检查集群版本
kubectl version

# 检查节点状态
kubectl get nodes

# 检查现有部署
kubectl get deployments -n default
kubectl get services -n default
kubectl get ingress -n default
```

## Istio安装

### 1. 下载Istio

```bash
# 下载最新版本的Istio
curl -L https://istio.io/downloadIstio | sh -

# 进入Istio目录
cd istio-*

# 将istioctl添加到PATH
export PATH=$PWD/bin:$PATH

# 或者永久添加到PATH
sudo cp bin/istioctl /usr/local/bin/

# 验证安装
istioctl version
```

### 2. 安装Istio

```bash
# 使用demo配置文件安装（适合开发和测试）
istioctl install --set profile=demo -y

# 或者使用minimal配置文件安装（适合生产环境）
# istioctl install --set profile=minimal -y
```

### 3. 验证Istio安装

```bash
# 检查Istio组件状态
kubectl get pods -n istio-system

# 检查Istio服务状态
kubectl get services -n istio-system

# 验证安装
istioctl verify-install
```

### 4. 启用自动边车注入

```bash
# 为default命名空间启用自动边车注入
kubectl label namespace default istio-injection=enabled

# 验证标签
kubectl get namespace default --show-labels
```

## 博客应用Istio配置

### 1. 创建Istio配置目录

```bash
# 在项目根目录创建istio配置目录
mkdir -p k8s/istio
```

### 2. 创建DestinationRule（目标规则）

创建文件 `k8s/istio/destination-rule.yaml`:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: gaamingzhang-blog
  namespace: default
spec:
  host: gaamingzhang-blog-service
  subsets:
  - name: stable
    labels:
      version: stable
  - name: canary
    labels:
      version: canary
  trafficPolicy:
    loadBalancer:
      simple: ROUND_ROBIN
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 100
        maxRequestsPerConnection: 10
    outlierDetection:
      consecutiveErrors: 5
      interval: 10s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
```

### 3. 创建VirtualService（虚拟服务）

创建文件 `k8s/istio/virtual-service.yaml`:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: gaamingzhang-blog
  namespace: default
spec:
  hosts:
  - "gaamingzhang-blog-service"
  http:
  - route:
    - destination:
        host: gaamingzhang-blog-service
        subset: stable
      weight: 100
    - destination:
        host: gaamingzhang-blog-service
        subset: canary
      weight: 0
```

### 4. 创建Gateway（网关）

创建文件 `k8s/istio/gateway.yaml`:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: gaamingzhang-blog-gateway
  namespace: default
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "blog.local"
    - "*"
```

### 5. 创建外部访问的VirtualService

创建文件 `k8s/istio/virtual-service-external.yaml`:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: gaamingzhang-blog-external
  namespace: default
spec:
  hosts:
  - "blog.local"
  - "*"
  gateways:
  - gaamingzhang-blog-gateway
  http:
  - route:
    - destination:
        host: gaamingzhang-blog-service
        subset: stable
      weight: 100
    - destination:
        host: gaamingzhang-blog-service
        subset: canary
      weight: 0
```

### 6. 应用Istio配置

```bash
# 应用所有Istio配置
kubectl apply -f k8s/istio/

# 或者逐个应用
kubectl apply -f k8s/istio/destination-rule.yaml
kubectl apply -f k8s/istio/virtual-service.yaml
kubectl apply -f k8s/istio/gateway.yaml
kubectl apply -f k8s/istio/virtual-service-external.yaml
```

## 金丝雀部署配置

### 1. 准备金丝雀版本的Deployment

创建文件 `k8s/deployment-canary.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gaamingzhang-blog-canary
  namespace: default
  labels:
    app: gaamingzhang-blog
    version: canary
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gaamingzhang-blog
      version: canary
  template:
    metadata:
      labels:
        app: gaamingzhang-blog
        version: canary
    spec:
      imagePullSecrets:
      - name: registry-credentials
      containers:
      - name: gaamingzhang-blog
        image: 192.168.31.40:30500/gaamingzhang-blog:canary
        imagePullPolicy: Always
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: "100m"
            memory: "256Mi"
          limits:
            cpu: "500m"
            memory: "512Mi"
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 15
          periodSeconds: 20
```

### 2. 更新稳定版本的Deployment标签

更新 `k8s/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gaamingzhang-blog-stable
  namespace: default
  labels:
    app: gaamingzhang-blog
    version: stable
spec:
  replicas: 2
  selector:
    matchLabels:
      app: gaamingzhang-blog
      version: stable
  template:
    metadata:
      labels:
        app: gaamingzhang-blog
        version: stable
    spec:
      imagePullSecrets:
      - name: registry-credentials
      containers:
      - name: gaamingzhang-blog
        image: 192.168.31.40:30500/gaamingzhang-blog:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: "100m"
            memory: "256Mi"
          limits:
            cpu: "500m"
            memory: "512Mi"
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 15
          periodSeconds: 20
```

### 3. 金丝雀发布流程

#### 步骤1: 部署金丝雀版本

```bash
# 部署金丝雀版本
kubectl apply -f k8s/deployment-canary.yaml

# 检查金丝雀Pod状态
kubectl get pods -l app=gaamingzhang-blog,version=canary

# 等待金丝雀Pod就绪
kubectl rollout status deployment/gaamingzhang-blog-canary
```

#### 步骤2: 初始流量分配（5%流量到金丝雀）

更新 `k8s/istio/virtual-service-external.yaml`:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: gaamingzhang-blog-external
  namespace: default
spec:
  hosts:
  - "blog.local"
  - "*"
  gateways:
  - gaamingzhang-blog-gateway
  http:
  - route:
    - destination:
        host: gaamingzhang-blog-service
        subset: stable
      weight: 95
    - destination:
        host: gaamingzhang-blog-service
        subset: canary
      weight: 5
```

应用配置:

```bash
kubectl apply -f k8s/istio/virtual-service-external.yaml
```

#### 步骤3: 监控金丝雀版本

```bash
# 监控金丝雀Pod日志
kubectl logs -f -l app=gaamingzhang-blog,version=canary

# 监控金丝雀Pod指标
kubectl top pods -l app=gaamingzhang-blog,version=canary

# 检查Pod事件
kubectl describe pods -l app=gaamingzhang-blog,version=canary
```

#### 步骤4: 逐步增加流量

**10%流量到金丝雀**:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: gaamingzhang-blog-external
  namespace: default
spec:
  hosts:
  - "blog.local"
  - "*"
  gateways:
  - gaamingzhang-blog-gateway
  http:
  - route:
    - destination:
        host: gaamingzhang-blog-service
        subset: stable
      weight: 90
    - destination:
        host: gaamingzhang-blog-service
        subset: canary
      weight: 10
```

**25%流量到金丝雀**:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: gaamingzhang-blog-external
  namespace: default
spec:
  hosts:
  - "blog.local"
  - "*"
  gateways:
  - gaamingzhang-blog-gateway
  http:
  - route:
    - destination:
        host: gaamingzhang-blog-service
        subset: stable
      weight: 75
    - destination:
        host: gaamingzhang-blog-service
        subset: canary
      weight: 25
```

**50%流量到金丝雀**:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: gaamingzhang-blog-external
  namespace: default
spec:
  hosts:
  - "blog.local"
  - "*"
  gateways:
  - gaamingzhang-blog-gateway
  http:
  - route:
    - destination:
        host: gaamingzhang-blog-service
        subset: stable
      weight: 50
    - destination:
        host: gaamingzhang-blog-service
        subset: canary
      weight: 50
```

#### 步骤5: 完成金丝雀发布

**100%流量到金丝雀（新版本成为稳定版本）**:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: gaamingzhang-blog-external
  namespace: default
spec:
  hosts:
  - "blog.local"
  - "*"
  gateways:
  - gaamingzhang-blog-gateway
  http:
  - route:
    - destination:
        host: gaamingzhang-blog-service
        subset: stable
      weight: 0
    - destination:
        host: gaamingzhang-blog-service
        subset: canary
      weight: 100
```

#### 步骤6: 清理旧版本

```bash
# 删除旧的稳定版本
kubectl delete deployment gaamingzhang-blog-stable

# 将金丝雀版本重命名为稳定版本
kubectl patch deployment gaamingzhang-blog-canary -p '{"metadata":{"name":"gaamingzhang-blog-stable"}}'

# 或者创建新的稳定版本Deployment
kubectl apply -f k8s/deployment.yaml

# 删除金丝雀版本
kubectl delete deployment gaamingzhang-blog-canary
```

### 4. 金丝雀发布自动化脚本

创建文件 `scripts/canary-release.sh`:

```bash
#!/bin/bash

# 金丝雀发布脚本
# 用法: ./canary-release.sh <weight>

WEIGHT=${1:-5}
STABLE_WEIGHT=$((100 - WEIGHT))

echo "设置金丝雀流量权重为 ${WEIGHT}%"

cat <<EOF | kubectl apply -f -
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: gaamingzhang-blog-external
  namespace: default
spec:
  hosts:
  - "blog.local"
  - "*"
  gateways:
  - gaamingzhang-blog-gateway
  http:
  - route:
    - destination:
        host: gaamingzhang-blog-service
        subset: stable
      weight: ${STABLE_WEIGHT}
    - destination:
        host: gaamingzhang-blog-service
        subset: canary
      weight: ${WEIGHT}
EOF

echo "流量分配已更新:"
echo "- 稳定版本: ${STABLE_WEIGHT}%"
echo "- 金丝雀版本: ${WEIGHT}%"
```

使用方法:

```bash
# 添加执行权限
chmod +x scripts/canary-release.sh

# 设置5%流量到金丝雀
./scripts/canary-release.sh 5

# 设置10%流量到金丝雀
./scripts/canary-release.sh 10

# 设置25%流量到金丝雀
./scripts/canary-release.sh 25

# 设置50%流量到金丝雀
./scripts/canary-release.sh 50

# 设置100%流量到金丝雀
./scripts/canary-release.sh 100
```

## 验证与测试

### 1. 验证Istio配置

```bash
# 检查所有Istio资源
kubectl get destinationrules -n default
kubectl get virtualservices -n default
kubectl get gateways -n default

# 使用istioctl分析配置
istioctl analyze

# 检查Envoy配置
istioctl proxy-config clusters <pod-name>
istioctl proxy-config routes <pod-name>
```

### 2. 验证边车注入

```bash
# 检查Pod是否注入了边车
kubectl get pods -n default -o jsonpath='{.items[*].spec.containers[*].name}'

# 应该看到两个容器: gaamingzhang-blog 和 istio-proxy
```

### 3. 测试流量分配

```bash
# 获取Ingress Gateway的IP地址
export INGRESS_HOST=$(kubectl get svc istio-ingressgateway -n istio-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# 如果没有LoadBalancer，使用NodePort
export INGRESS_PORT=$(kubectl get svc istio-ingressgateway -n istio-system -o jsonpath='{.spec.ports[?(@.name=="http2")].nodePort}')
export INGRESS_HOST=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}')

# 测试访问
curl -s http://$INGRESS_HOST:$INGRESS_PORT/ -H "Host: blog.local"

# 发送多个请求测试流量分配
for i in {1..100}; do
  curl -s http://$INGRESS_HOST:$INGRESS_PORT/ -H "Host: blog.local" -o /dev/null -w "%{http_code}\n"
done | sort | uniq -c
```

### 4. 监控金丝雀发布

```bash
# 监控Pod状态
watch kubectl get pods -l app=gaamingzhang-blog

# 监控Pod资源使用
watch kubectl top pods -l app=gaamingzhang-blog

# 监控服务指标
kubectl exec -it <pod-name> -c istio-proxy -- curl localhost:15000/stats
```

## 故障排查

### 1. 边车注入失败

**症状**: Pod没有istio-proxy容器

**解决方案**:

```bash
# 检查命名空间标签
kubectl get namespace default --show-labels

# 如果没有istio-injection=enabled标签，添加它
kubectl label namespace default istio-injection=enabled

# 重启Pod
kubectl rollout restart deployment/gaamingzhang-blog-stable
kubectl rollout restart deployment/gaamingzhang-blog-canary
```

### 2. 流量路由不生效

**症状**: 流量没有按照预期分配

**解决方案**:

```bash
# 检查VirtualService配置
kubectl get virtualservice gaamingzhang-blog-external -n default -o yaml

# 检查DestinationRule配置
kubectl get destinationrule gaamingzhang-blog -n default -o yaml

# 使用istioctl分析配置
istioctl analyze

# 检查Envoy配置
istioctl proxy-config routes <pod-name> -n default
```

### 3. 服务间通信失败

**症状**: 服务无法相互访问

**解决方案**:

```bash
# 检查服务配置
kubectl get svc gaamingzhang-blog-service -n default

# 检查Pod标签
kubectl get pods -l app=gaamingzhang-blog --show-labels

# 检查网络策略
kubectl get networkpolicies -n default

# 测试服务连接
kubectl exec -it <pod-name> -c gaamingzhang-blog -- curl http://gaamingzhang-blog-service:80/
```

### 4. 金丝雀发布回滚

**症状**: 金丝雀版本出现问题，需要回滚

**解决方案**:

```bash
# 立即将所有流量切回稳定版本
cat <<EOF | kubectl apply -f -
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: gaamingzhang-blog-external
  namespace: default
spec:
  hosts:
  - "blog.local"
  - "*"
  gateways:
  - gaamingzhang-blog-gateway
  http:
  - route:
    - destination:
        host: gaamingzhang-blog-service
        subset: stable
      weight: 100
    - destination:
        host: gaamingzhang-blog-service
        subset: canary
      weight: 0
EOF

# 删除金丝雀版本
kubectl delete deployment gaamingzhang-blog-canary

# 验证稳定版本正常
kubectl get pods -l app=gaamingzhang-blog,version=stable
```

## 清理

### 1. 删除Istio配置

```bash
# 删除Istio资源
kubectl delete -f k8s/istio/

# 删除金丝雀Deployment
kubectl delete -f k8s/deployment-canary.yaml
```

### 2. 卸载Istio

```bash
# 卸载Istio
istioctl uninstall -y --purge

# 删除Istio命名空间
kubectl delete namespace istio-system

# 删除自动边车注入标签
kubectl label namespace default istio-injection-
```

## 最佳实践

### 1. 金丝雀发布策略

- **初始流量**: 从5%开始，逐步增加
- **监控指标**: 密切关注错误率、延迟、资源使用
- **回滚准备**: 准备好快速回滚的脚本和流程
- **测试验证**: 在每个流量阶段进行充分的测试

### 2. 监控和告警

- **关键指标**: 请求成功率、响应时间、错误率
- **告警规则**: 设置合理的告警阈值
- **日志收集**: 收集和分析应用日志
- **分布式追踪**: 使用Jaeger或Zipkin进行链路追踪

### 3. 安全考虑

- **mTLS**: 启用服务间的mTLS加密
- **访问控制**: 使用AuthorizationPolicy限制访问
- **证书管理**: 定期轮换证书
- **网络安全**: 配置NetworkPolicy限制网络访问

## 参考资料

- [Istio官方文档](https://istio.io/docs/)
- [Istio金丝雀部署](https://istio.io/docs/tasks/traffic-management/traffic-shifting/)
- [Kubernetes官方文档](https://kubernetes.io/docs/)

# Istio部署与金丝雀发布 - 完整配置总结

## 已创建的文件

### 1. Istio配置文件 (k8s/istio/)

- **README.md**: 详细的部署和配置指南
- **destination-rule.yaml**: 目标规则,定义服务子集和流量策略
- **virtual-service.yaml**: 内部虚拟服务,用于服务间通信
- **gateway.yaml**: Istio网关,用于外部流量入口
- **virtual-service-external.yaml**: 外部虚拟服务,用于外部访问和流量分配

### 2. Kubernetes部署文件 (k8s/)

- **deployment-stable.yaml**: 稳定版本的Deployment配置
- **deployment-canary.yaml**: 金丝雀版本的Deployment配置

### 3. 自动化脚本 (scripts/)

- **deploy-istio.sh**: 一键部署Istio和配置金丝雀发布
- **canary-release.sh**: 金丝雀发布流量控制脚本
- **rollback-canary.sh**: 金丝雀发布回滚脚本

### 4. 文档 (docs/)

- **istio-quick-reference.md**: 快速参考指南,包含常用命令和故障排查

## 部署步骤

### 方式一: 使用自动化脚本（推荐）

```bash
# 1. 执行一键部署脚本
./scripts/deploy-istio.sh

# 脚本会自动完成以下步骤:
# - 检查前置条件
# - 下载并安装Istio
# - 启用自动边车注入
# - 应用Istio配置
# - 部署应用
# - 验证部署
# - 显示访问信息
```

### 方式二: 手动部署

```bash
# 1. 下载Istio
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH

# 2. 安装Istio
istioctl install --set profile=demo -y

# 3. 启用自动边车注入
kubectl label namespace default istio-injection=enabled

# 4. 应用Istio配置
kubectl apply -f k8s/istio/

# 5. 部署应用
kubectl apply -f k8s/deployment-stable.yaml
kubectl apply -f k8s/service.yaml

# 6. 验证部署
kubectl get pods -l app=gaamingzhang-blog
kubectl get destinationrules,virtualservices,gateways -n default
istioctl analyze
```

## 金丝雀发布流程

### 1. 准备金丝雀镜像

```bash
# 构建金丝雀镜像
docker build -t 192.168.31.40:30500/gaamingzhang-blog:canary .
docker push 192.168.31.40:30500/gaamingzhang-blog:canary
```

### 2. 部署金丝雀版本

```bash
# 部署金丝雀版本
kubectl apply -f k8s/deployment-canary.yaml

# 等待金丝雀Pod就绪
kubectl rollout status deployment/gaamingzhang-blog-canary
```

### 3. 执行金丝雀发布

```bash
# 初始流量分配 (5%)
./scripts/canary-release.sh 5

# 监控一段时间后,逐步增加流量
./scripts/canary-release.sh 10   # 10%
./scripts/canary-release.sh 25   # 25%
./scripts/canary-release.sh 50   # 50%

# 完成发布 (100%)
./scripts/canary-release.sh 100
```

### 4. 清理旧版本

```bash
# 删除旧的稳定版本
kubectl delete deployment gaamingzhang-blog-stable

# 将金丝雀版本重命名为稳定版本
kubectl apply -f k8s/deployment-stable.yaml
kubectl delete deployment gaamingzhang-blog-canary
```

### 5. 回滚（如果需要）

```bash
# 执行回滚脚本
./scripts/rollback-canary.sh
```

## 验证和测试

### 1. 验证Istio安装

```bash
# 检查Istio组件状态
kubectl get pods -n istio-system

# 验证边车注入
kubectl get pods -n default -o jsonpath='{.items[*].spec.containers[*].name}'

# 分析Istio配置
istioctl analyze
```

### 2. 测试访问

```bash
# 获取Ingress Gateway地址
export INGRESS_HOST=$(kubectl get svc istio-ingressgateway -n istio-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# 如果没有LoadBalancer,使用NodePort
export INGRESS_PORT=$(kubectl get svc istio-ingressgateway -n istio-system -o jsonpath='{.spec.ports[?(@.name=="http2")].nodePort}')
export INGRESS_HOST=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}')

# 测试访问
curl -s http://$INGRESS_HOST:$INGRESS_PORT/ -H "Host: blog.local"
```

### 3. 测试流量分配

```bash
# 发送100个请求,查看流量分配
for i in {1..100}; do
  curl -s http://$INGRESS_HOST:$INGRESS_PORT/ -H "Host: blog.local" -o /dev/null -w "%{http_code}\n"
done | sort | uniq -c
```

## 监控和可观测性

### 1. 部署监控组件（可选）

```bash
# 部署Kiali（服务网格可视化）
kubectl apply -f https://raw.githubusercontent.com/istio/istio/master/samples/addons/kiali.yaml

# 部署Prometheus（监控系统）
kubectl apply -f https://raw.githubusercontent.com/istio/istio/master/samples/addons/prometheus.yaml

# 部署Grafana（数据可视化）
kubectl apply -f https://raw.githubusercontent.com/istio/istio/master/samples/addons/grafana.yaml

# 部署Jaeger（分布式追踪）
kubectl apply -f https://raw.githubusercontent.com/istio/istio/master/samples/addons/jaeger.yaml
```

### 2. 访问监控界面

```bash
# 启动Kiali控制台
istioctl dashboard kiali

# 启动Grafana控制台
istioctl dashboard grafana

# 启动Jaeger控制台
istioctl dashboard jaeger

# 启动Prometheus控制台
istioctl dashboard prometheus
```

## 常见问题

### 1. 边车注入失败

**症状**: Pod没有istio-proxy容器

**解决方案**:
```bash
# 检查命名空间标签
kubectl get namespace default --show-labels

# 添加标签
kubectl label namespace default istio-injection=enabled

# 重启Pod
kubectl rollout restart deployment/gaamingzhang-blog-stable
```

### 2. 流量路由不生效

**症状**: 流量没有按照预期分配

**解决方案**:
```bash
# 检查VirtualService配置
kubectl get virtualservice gaamingzhang-blog-external -n default -o yaml

# 分析配置
istioctl analyze

# 检查Envoy配置
istioctl proxy-config routes <pod-name>
```

### 3. 服务无法访问

**症状**: 无法访问服务

**解决方案**:
```bash
# 检查服务状态
kubectl get svc gaamingzhang-blog-service

# 检查Pod标签
kubectl get pods -l app=gaamingzhang-blog --show-labels

# 测试服务连接
kubectl exec -it <pod-name> -c gaamingzhang-blog -- curl http://gaamingzhang-blog-service:80/
```

## 下一步

1. **阅读详细文档**: 查看 `k8s/istio/README.md` 了解详细配置
2. **参考快速指南**: 查看 `docs/istio-quick-reference.md` 了解常用命令
3. **执行部署**: 运行 `./scripts/deploy-istio.sh` 开始部署
4. **测试金丝雀发布**: 使用 `./scripts/canary-release.sh` 测试流量分配
5. **配置监控**: 部署Kiali、Prometheus等监控组件

## 注意事项

1. **资源要求**: 确保集群有足够的资源（至少4GB内存和2核CPU）
2. **网络配置**: 确保Pod间网络通信正常
3. **镜像准备**: 提前准备好稳定版本和金丝雀版本的镜像
4. **监控告警**: 配置关键指标的监控和告警
5. **回滚准备**: 准备好快速回滚的方案

## 技术支持

如遇问题,请参考以下资源:

- [Istio官方文档](https://istio.io/docs/)
- [Kubernetes官方文档](https://kubernetes.io/docs/)
- [项目Istio配置文档](./k8s/istio/README.md)
- [快速参考指南](./docs/istio-quick-reference.md)

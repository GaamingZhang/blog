# Istio金丝雀发布快速参考

## 快速开始

### 1. 一键部署Istio

```bash
# 执行部署脚本
./scripts/deploy-istio.sh
```

### 2. 金丝雀发布

```bash
# 部署金丝雀版本
kubectl apply -f k8s/deployment-canary.yaml

# 等待金丝雀Pod就绪
kubectl rollout status deployment/gaamingzhang-blog-canary

# 设置流量权重
./scripts/canary-release.sh 5    # 5%流量到金丝雀
./scripts/canary-release.sh 10   # 10%流量到金丝雀
./scripts/canary-release.sh 25   # 25%流量到金丝雀
./scripts/canary-release.sh 50   # 50%流量到金丝雀
./scripts/canary-release.sh 100  # 100%流量到金丝雀
```

### 3. 回滚

```bash
# 执行回滚脚本
./scripts/rollback-canary.sh
```

## 常用命令

### Istio管理

```bash
# 检查Istio状态
kubectl get pods -n istio-system

# 检查边车注入
kubectl get namespace default --show-labels

# 启用边车注入
kubectl label namespace default istio-injection=enabled

# 禁用边车注入
kubectl label namespace default istio-injection-
```

### 应用管理

```bash
# 查看Pod状态
kubectl get pods -l app=gaamingzhang-blog

# 查看Pod日志
kubectl logs -f -l app=gaamingzhang-blog -c gaamingzhang-blog

# 查看Istio代理日志
kubectl logs -f -l app=gaamingzhang-blog -c istio-proxy

# 查看资源使用
kubectl top pods -l app=gaamingzhang-blog
```

### Istio配置

```bash
# 查看DestinationRule
kubectl get destinationrules -n default

# 查看VirtualService
kubectl get virtualservices -n default

# 查看Gateway
kubectl get gateways -n default

# 分析Istio配置
istioctl analyze

# 查看Envoy配置
istioctl proxy-config clusters <pod-name>
istioctl proxy-config routes <pod-name>
istioctl proxy-config listeners <pod-name>
```

### 流量管理

```bash
# 手动更新流量权重
kubectl patch virtualservice gaamingzhang-blog-external -n default --type=json -p='[
  {
    "op": "replace",
    "path": "/spec/http/0/route/0/weight",
    "value": 90
  },
  {
    "op": "replace",
    "path": "/spec/http/0/route/1/weight",
    "value": 10
  }
]'

# 查看当前流量分配
kubectl get virtualservice gaamingzhang-blog-external -n default -o yaml | grep -A 15 "http:"
```

### 故障排查

```bash
# 检查Pod事件
kubectl describe pod <pod-name>

# 检查服务端点
kubectl get endpoints gaamingzhang-blog-service

# 测试服务连接
kubectl exec -it <pod-name> -c gaamingzhang-blog -- curl http://gaamingzhang-blog-service:80/

# 查看Istio代理状态
kubectl exec -it <pod-name> -c istio-proxy -- curl localhost:15000/help
kubectl exec -it <pod-name> -c istio-proxy -- curl localhost:15000/config_dump
kubectl exec -it <pod-name> -c istio-proxy -- curl localhost:15000/clusters
kubectl exec -it <pod-name> -c istio-proxy -- curl localhost:15000/routes
```

## 金丝雀发布流程

### 标准流程

1. **准备金丝雀镜像**
   ```bash
   # 构建并推送金丝雀镜像
   docker build -t 192.168.31.40:30500/gaamingzhang-blog:canary .
   docker push 192.168.31.40:30500/gaamingzhang-blog:canary
   ```

2. **部署金丝雀版本**
   ```bash
   kubectl apply -f k8s/deployment-canary.yaml
   kubectl rollout status deployment/gaamingzhang-blog-canary
   ```

3. **初始流量分配 (5%)**
   ```bash
   ./scripts/canary-release.sh 5
   ```

4. **监控和验证**
   ```bash
   # 监控Pod状态
   watch kubectl get pods -l app=gaamingzhang-blog
   
   # 监控资源使用
   watch kubectl top pods -l app=gaamingzhang-blog
   
   # 查看日志
   kubectl logs -f -l app=gaamingzhang-blog,version=canary
   ```

5. **逐步增加流量**
   ```bash
   ./scripts/canary-release.sh 10   # 10%
   ./scripts/canary-release.sh 25   # 25%
   ./scripts/canary-release.sh 50   # 50%
   ```

6. **完成发布 (100%)**
   ```bash
   ./scripts/canary-release.sh 100
   ```

7. **清理旧版本**
   ```bash
   # 删除旧的稳定版本
   kubectl delete deployment gaamingzhang-blog-stable
   
   # 将金丝雀版本重命名为稳定版本
   kubectl apply -f k8s/deployment-stable.yaml
   kubectl delete deployment gaamingzhang-blog-canary
   ```

### 回滚流程

1. **立即回滚**
   ```bash
   ./scripts/rollback-canary.sh
   ```

2. **手动回滚**
   ```bash
   # 将所有流量切回稳定版本
   ./scripts/canary-release.sh 0
   
   # 删除金丝雀版本
   kubectl delete deployment gaamingzhang-blog-canary
   ```

## 监控指标

### 关键指标

- **请求成功率**: 应该保持在99%以上
- **响应时间**: P50 < 100ms, P99 < 500ms
- **错误率**: 应该保持在1%以下
- **资源使用**: CPU < 80%, Memory < 80%

### 监控命令

```bash
# 查看Pod资源使用
kubectl top pods -l app=gaamingzhang-blog

# 查看节点资源使用
kubectl top nodes

# 查看Istio代理统计
kubectl exec -it <pod-name> -c istio-proxy -- curl localhost:15000/stats | grep -E "downstream_rq|upstream_rq"
```

## 最佳实践

### 1. 金丝雀发布策略

- 从小流量开始（5%）
- 每个阶段停留足够时间（至少30分钟）
- 密切监控关键指标
- 准备好回滚方案

### 2. 镜像管理

- 使用明确的标签（如：stable, canary, v1.0.0）
- 不要使用latest标签
- 保持镜像仓库清洁

### 3. 配置管理

- 使用版本控制管理所有配置文件
- 变更前备份当前配置
- 使用GitOps流程

### 4. 监控告警

- 设置关键指标的告警
- 配置日志收集和分析
- 使用分布式追踪

## 故障场景

### 场景1: 金丝雀版本有Bug

**症状**: 错误率上升，用户投诉

**处理**:
```bash
# 1. 立即回滚
./scripts/rollback-canary.sh

# 2. 分析日志
kubectl logs -l app=gaamingzhang-blog,version=canary

# 3. 修复问题并重新构建镜像
```

### 场景2: 金丝雀版本性能问题

**症状**: 响应时间变长，资源使用率高

**处理**:
```bash
# 1. 减少金丝雀流量
./scripts/canary-release.sh 5

# 2. 分析性能瓶颈
kubectl top pods -l app=gaamingzhang-blog
kubectl exec -it <pod-name> -c istio-proxy -- curl localhost:15000/stats

# 3. 优化或回滚
```

### 场景3: Istio配置错误

**症状**: 服务无法访问

**处理**:
```bash
# 1. 检查Istio配置
istioctl analyze

# 2. 查看Envoy配置
istioctl proxy-config routes <pod-name>

# 3. 恢复正确配置
kubectl apply -f k8s/istio/
```

## 参考资料

- [Istio官方文档](https://istio.io/docs/)
- [Istio流量管理](https://istio.io/docs/tasks/traffic-management/)
- [Kubernetes官方文档](https://kubernetes.io/docs/)
- [项目Istio配置文档](./k8s/istio/README.md)

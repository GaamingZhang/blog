---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Service
  - 排查
---

# Pod正常但Service异常排查指南

## 问题现象

在Kubernetes集群中，经常会遇到这样的场景：
- 在节点上直接访问Pod是正常的
- Pod的ReadinessProbe检查通过
- 但通过Service访问时出现异常

这种问题通常表现为：
- Service访问超时
- Service访问返回错误
- Service访问不稳定

## 排查思路

```
┌─────────────────────────────────────────────────────────────┐
│                     问题排查流程                             │
│                                                              │
│  1. 检查Pod状态                                              │
│         ↓                                                    │
│  2. 检查Service配置                                          │
│         ↓                                                    │
│  3. 检查Endpoints                                            │
│         ↓                                                    │
│  4. 检查kube-proxy                                           │
│         ↓                                                    │
│  5. 检查网络策略                                             │
│         ↓                                                    │
│  6. 检查DNS解析                                              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 第一步：确认Pod状态

### 1.1 检查Pod是否正常运行

```bash
kubectl get pods -l app=nginx

NAME                     READY   STATUS    RESTARTS   AGE
nginx-6c8b5b5d4d-abcde   1/1     Running   0          10m
nginx-6c8b5b5d4d-fghij   1/1     Running   0          10m
nginx-6c8b5b5d4d-klmno   1/1     Running   0          10m
```

**关键点**：
- `READY`列显示`1/1`，表示容器就绪
- `STATUS`为`Running`

### 1.2 直接访问Pod验证

```bash
POD_IP=$(kubectl get pod nginx-6c8b5b5d4d-abcde -o jsonpath='{.status.podIP}')
kubectl exec -it test-pod -- curl http://$POD_IP:8080
```

如果直接访问Pod IP正常，说明应用本身没问题。

### 1.3 检查Pod标签

```bash
kubectl get pods --show-labels

NAME                     READY   STATUS    RESTARTS   AGE   LABELS
nginx-6c8b5b5d4d-abcde   1/1     Running   0          10m   app=nginx,version=v1
```

确认Pod的标签与Service的selector匹配。

## 第二步：检查Service配置

### 2.1 查看Service配置

```bash
kubectl get svc nginx-service -o yaml

apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
```

### 2.2 检查selector是否匹配

```bash
kubectl get pods -l app=nginx
```

**常见问题**：
- Service的selector与Pod的labels不匹配
- 标签名称拼写错误
- 标签值不一致

### 2.3 检查端口配置

```bash
kubectl describe svc nginx-service

Port:                     80/TCP
TargetPort:               8080/TCP
```

**常见问题**：
- `targetPort`与容器监听端口不一致
- `protocol`配置错误（TCP/UDP）

## 第三步：检查Endpoints

### 3.1 查看Endpoints

```bash
kubectl get endpoints nginx-service

NAME            ENDPOINTS                                   AGE
nginx-service   10.1.1.1:8080,10.1.1.2:8080,10.1.1.3:8080   10m
```

### 3.2 Endpoints为空的情况

如果Endpoints为空：

```bash
kubectl describe endpoints nginx-service

# 检查事件
Events:
  Type     Reason         Age   From                Message
  ----     ------         ----  ----                -------
  Warning  NoPods         10m   endpoint-controller No pods matched the selector
```

**可能原因**：
1. Service的selector与Pod标签不匹配
2. Pod的ReadinessProbe未通过
3. Pod处于NotReady状态

### 3.3 Endpoints部分缺失

如果Endpoints只有部分Pod：

```bash
kubectl get pods -o wide
kubectl get endpoints nginx-service -o yaml
```

对比Pod IP和Endpoints中的IP，找出缺失的Pod。

**可能原因**：
- 部分Pod的ReadinessProbe失败
- 部分Pod的标签被修改
- 部分Pod被删除

## 第四步：检查kube-proxy

### 4.1 检查kube-proxy状态

```bash
kubectl get pods -n kube-system -l k8s-app=kube-proxy

NAME               READY   STATUS    RESTARTS   AGE
kube-proxy-abcde   1/1     Running   0          10d
```

### 4.2 检查kube-proxy日志

```bash
kubectl logs -n kube-system kube-proxy-abcde
```

### 4.3 检查iptables规则

```bash
iptables -t nat -L KUBE-SERVICES | grep nginx

KUBE-SVC-XXX  tcp  --  anywhere  10.96.0.100  /* default/nginx-service */
```

### 4.4 检查IPVS规则（如果使用IPVS模式）

```bash
ipvsadm -Ln | grep 10.96.0.100

TCP  10.96.0.100:80 rr
  -> 10.1.1.1:8080          Masq    1      0          0
  -> 10.1.1.2:8080          Masq    1      0          0
  -> 10.1.1.3:8080          Masq    1      0          0
```

## 第五步：检查网络策略

### 5.1 查看NetworkPolicy

```bash
kubectl get networkpolicies
kubectl describe networkpolicy <policy-name>
```

### 5.2 网络策略可能的影响

NetworkPolicy可能阻止：
- 入站流量
- 出站流量
- 特定命名空间的流量

### 5.3 临时禁用NetworkPolicy测试

```bash
kubectl delete networkpolicy <policy-name>
```

## 第六步：检查DNS解析

### 6.1 测试Service DNS解析

```bash
kubectl exec -it test-pod -- nslookup nginx-service

Server:    10.96.0.10
Address:   10.96.0.10#53

Name:      nginx-service.default.svc.cluster.local
Address:   10.96.0.100
```

### 6.2 检查CoreDNS

```bash
kubectl get pods -n kube-system -l k8s-app=kube-dns

NAME                       READY   STATUS    RESTARTS   AGE
coredns-5644d7b6d9-abcde   1/1     Running   0          10d
coredns-5644d7b6d9-fghij   1/1     Running   0          10d
```

### 6.3 检查DNS配置

```bash
kubectl exec -it test-pod -- cat /etc/resolv.conf

nameserver 10.96.0.10
search default.svc.cluster.local svc.cluster.local cluster.local
options ndots:5
```

## 常见问题场景

### 场景1：selector不匹配

**问题**：
```yaml
Service selector:
  app: nginx

Pod labels:
  app: nginx-server
```

**解决**：
```yaml
Service selector:
  app: nginx-server
```

### 场景2：端口配置错误

**问题**：
```yaml
Service:
  targetPort: 80

Container:
  port: 8080
```

**解决**：
```yaml
Service:
  targetPort: 8080
```

### 场景3：ReadinessProbe失败

**问题**：Pod的ReadinessProbe失败，导致Pod不在Endpoints中。

**排查**：
```bash
kubectl describe pod <pod-name>

Events:
  Type     Reason     Age   From               Message
  ----     ------     ----  ----               -------
  Warning  Unhealthy  10m   kubelet            Readiness probe failed
```

**解决**：
- 检查ReadinessProbe配置
- 检查应用健康检查端点

### 场景4：kube-proxy异常

**问题**：kube-proxy未正确同步规则。

**排查**：
```bash
kubectl logs -n kube-system kube-proxy-abcde

E0115 10:00:00.000000 1 proxier.go:XXX] Failed to sync iptables rules
```

**解决**：
```bash
kubectl delete pod -n kube-system kube-proxy-abcde
```

### 场景5：NetworkPolicy阻止

**问题**：NetworkPolicy阻止了Service流量。

**排查**：
```bash
kubectl get networkpolicies -o yaml
```

**解决**：
- 修改NetworkPolicy允许Service流量
- 或临时删除NetworkPolicy测试

### 场景6：DNS解析失败

**问题**：CoreDNS无法解析Service名称。

**排查**：
```bash
kubectl logs -n kube-system coredns-xxx

[ERROR] plugin/errors: 2 nginx-service.default.svc.cluster.local. A: unreachable backend
```

**解决**：
```bash
kubectl delete pod -n kube-system coredns-xxx
```

## 完整排查脚本

```bash
#!/bin/bash

SERVICE_NAME="nginx-service"
NAMESPACE="default"

echo "=== 1. 检查Pod状态 ==="
kubectl get pods -n $NAMESPACE

echo -e "\n=== 2. 检查Service配置 ==="
kubectl get svc $SERVICE_NAME -n $NAMESPACE -o yaml

echo -e "\n=== 3. 检查Endpoints ==="
kubectl get endpoints $SERVICE_NAME -n $NAMESPACE

echo -e "\n=== 4. 检查Pod标签 ==="
kubectl get pods -n $NAMESPACE --show-labels

echo -e "\n=== 5. 检查kube-proxy ==="
kubectl get pods -n kube-system -l k8s-app=kube-proxy

echo -e "\n=== 6. 检查CoreDNS ==="
kubectl get pods -n kube-system -l k8s-app=kube-dns

echo -e "\n=== 7. 测试DNS解析 ==="
kubectl run test-dns --rm -it --image=busybox -- nslookup $SERVICE_NAME.$NAMESPACE.svc.cluster.local

echo -e "\n=== 8. 测试Service访问 ==="
kubectl run test-curl --rm -it --image=curlimages/curl -- curl -v http://$SERVICE_NAME.$NAMESPACE:80
```

## 排查流程图

```
Service访问异常
      │
      ├─→ Endpoints为空？
      │       │
      │       ├─→ 是 → 检查selector匹配
      │       │       检查ReadinessProbe
      │       │
      │       └─→ 否 → 检查kube-proxy
      │               检查iptables/IPVS规则
      │
      ├─→ DNS解析失败？
      │       │
      │       └─→ 检查CoreDNS状态
      │               检查DNS配置
      │
      └─→ 网络不通？
              │
              └─→ 检查NetworkPolicy
                      检查节点网络
```

## 最佳实践

### 1. 配置健康检查

```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### 2. 使用命名端口

```yaml
ports:
- name: http
  containerPort: 8080

Service:
ports:
- port: 80
  targetPort: http
```

### 3. 添加标签规范

```yaml
labels:
  app: nginx
  version: v1
  environment: production
```

### 4. 监控Service状态

```yaml
groups:
- name: service
  rules:
  - alert: ServiceEndpointsEmpty
    expr: kube_endpoint_address_available{endpoint=~".*"} == 0
    for: 5m
    labels:
      severity: critical
```

## 参考资源

- [Service官方文档](https://kubernetes.io/docs/concepts/services-networking/service/)
- [调试Service](https://kubernetes.io/docs/tasks/debug/debug-application/debug-service/)

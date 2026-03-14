---
date: 2026-03-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Deployment
  - HPA
  - 水平扩展
---

# Deployment 水平扩展

## 引言

水平扩展是 Kubernetes 应对负载变化的核心能力。通过调整 Deployment 的副本数量，可以快速增加或减少应用实例，实现弹性伸缩。本文介绍 Deployment 水平扩展的多种方式和最佳实践。

## 水平扩展概述

### 扩展方式

```
┌─────────────────────────────────────────────────────────────┐
│              Deployment 水平扩展方式                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  手动扩展：                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • kubectl scale 命令                               │   │
│  │  • 修改 replicas 字段                               │   │
│  │  • 适用于计划性扩缩容                               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  自动扩展：                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • HPA (HorizontalPodAutoscaler)                    │   │
│  │  • 基于指标自动调整副本数                           │   │
│  │  • 适用于动态负载变化                               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 扩展流程

```
┌─────────────────────────────────────────────────────────────┐
│              水平扩展流程                                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  扩展前：3 副本                                              │
│  ┌───┐ ┌───┐ ┌───┐                                         │
│  │Pod│ │Pod│ │Pod│                                         │
│  └───┘ └───┘ └───┘                                         │
│                                                              │
│  扩展命令：kubectl scale deployment myapp --replicas=5      │
│       │                                                      │
│       ▼                                                      │
│  扩展后：5 副本                                              │
│  ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐                            │
│  │Pod│ │Pod│ │Pod│ │Pod│ │Pod│                            │
│  └───┘ └───┘ └───┘ └───┘ └───┘                            │
│       ↑       ↑                                              │
│      新创建的 Pod                                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 手动扩展

### kubectl scale 命令

```bash
kubectl scale deployment myapp --replicas=5

kubectl scale deployment myapp --replicas=3 -n production

kubectl scale deployment myapp --current-replicas=3 --replicas=5
```

### 修改 YAML 文件

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 5
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: myapp:v1
```

```bash
kubectl apply -f deployment.yaml
```

### kubectl edit 命令

```bash
kubectl edit deployment myapp

kubectl edit deployment myapp -n production
```

## 自动扩展（HPA）

### HPA 工作原理

```
┌─────────────────────────────────────────────────────────────┐
│                  HPA 工作原理                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Metrics Server                           │  │
│  │  • 采集 Pod 资源使用指标                              │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              HPA Controller                           │  │
│  │  • 计算期望副本数                                    │  │
│  │  • 调整 Deployment replicas                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Deployment                           │  │
│  │  • 创建/删除 Pod                                     │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  计算公式：                                                  │
│  期望副本数 = ceil(当前副本数 × (当前指标值 / 目标指标值))  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 基于 CPU 的 HPA

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp-hpa
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### 基于内存的 HPA

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp-memory-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 基于自定义指标的 HPA

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp-custom-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Pods
    pods:
      metric:
        name: http_requests_per_second
      target:
        type: AverageValue
        averageValue: 1000
```

### 多指标 HPA

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp-multi-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: http_requests_per_second
      target:
        type: AverageValue
        averageValue: 1000
```

### HPA 行为配置

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp-behavior-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
      - type: Pods
        value: 2
        periodSeconds: 60
      selectPolicy: Min
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
      - type: Pods
        value: 4
        periodSeconds: 15
      selectPolicy: Max
```

## 查看 HPA 状态

### 查看 HPA 信息

```bash
kubectl get hpa

kubectl get hpa -n production

kubectl describe hpa myapp-hpa -n production
```

### 查看 HPA 详情

```bash
kubectl get hpa myapp-hpa -o yaml

kubectl describe hpa myapp-hpa
```

## 扩展验证

### 检查副本数

```bash
kubectl get deployment myapp

kubectl get pods -l app=myapp

kubectl get pods -l app=myapp -o wide
```

### 模拟负载测试

```bash
kubectl run -i --tty load-generator --rm --image=busybox --restart=Never -- /bin/sh -c "while sleep 0.01; do wget -q -O- http://myapp; done"
```

## 最佳实践

### 1. 设置合理的副本范围

```yaml
minReplicas: 2
maxReplicas: 10
```

### 2. 配置资源请求

```yaml
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
```

### 3. 配置健康检查

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
```

### 4. 配置扩展行为

```yaml
behavior:
  scaleDown:
    stabilizationWindowSeconds: 300
  scaleUp:
    stabilizationWindowSeconds: 0
```

## 面试回答

**问题**: 如何水平扩展 Deployment？

**回答**: Deployment 水平扩展分为手动扩展和自动扩展两种方式：

**手动扩展**：使用 `kubectl scale deployment myapp --replicas=5` 命令直接指定副本数；或修改 Deployment YAML 文件中的 replicas 字段后执行 `kubectl apply`；或使用 `kubectl edit deployment myapp` 直接编辑。

**自动扩展（HPA）**：使用 HorizontalPodAutoscaler 根据指标自动调整副本数。HPA 通过 Metrics Server 获取 Pod 资源使用指标，计算期望副本数：期望副本数 = ceil(当前副本数 × (当前指标值 / 目标指标值))。

**HPA 支持的指标**：**CPU 使用率**最常用，设置 averageUtilization 目标值；**内存使用率**设置 averageUtilization 目标值；**自定义指标**如 QPS、连接数等，设置 averageValue 目标值。支持多指标组合，取最大值。

**HPA 行为配置**：scaleDown 配置缩容策略，stabilizationWindowSeconds 设置稳定窗口防止频繁波动；scaleUp 配置扩容策略，支持 Percent 和 Pods 两种策略类型。

**最佳实践**：设置合理的 minReplicas 和 maxReplicas 范围；配置 Pod 资源请求（requests）；配置 readinessProbe 确保新 Pod 就绪后才接收流量；配置扩展行为避免频繁波动；监控 HPA 状态和效果。

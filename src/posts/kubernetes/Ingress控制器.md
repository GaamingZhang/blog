---
date: 2026-01-30
author: Gaaming Zhang
isOriginal: true
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# Ingress控制器

## 为什么需要Ingress？

想象你开了一家大型商场，里面有很多店铺（Service）。如果每家店铺都开一个门朝向大街，那街道会变得非常混乱，而且顾客也很难找到想去的店。更好的做法是什么？建一个统一的大门，设置一个导购台，根据顾客要去的地方指引他们。

这就是Ingress的作用。在Kubernetes中：

- **没有Ingress时**：每个Service都需要暴露自己的端口，管理起来非常麻烦
- **有了Ingress后**：所有外部流量从一个入口进来，根据规则分发到不同的Service

简单说，Ingress就是集群的"统一入口"加"智能路由"。

## Ingress和Ingress Controller的区别

这是很多初学者容易混淆的概念：

- **Ingress**：是一份"规则说明书"，告诉系统"访问app.example.com应该转到哪个Service"
- **Ingress Controller**：是真正执行这些规则的"工作人员"

打个比方，Ingress就像餐厅的菜单，Ingress Controller就像服务员。菜单定义了有什么菜，但真正把菜端上来的是服务员。

Kubernetes本身只提供了Ingress规则的定义能力，你需要自己安装Ingress Controller。最常用的是Nginx Ingress Controller。

## 工作原理图解

```
用户请求 app.example.com
        │
        ▼
┌───────────────────┐
│  Ingress Controller│  ← 根据Ingress规则判断该转发给谁
│  (如Nginx)        │
└───────────────────┘
        │
        ├── app.example.com → web-service
        ├── api.example.com → api-service
        └── admin.example.com → admin-service
```

## 最基础的Ingress配置

下面是一个最简单的例子，让 `app.example.com` 的访问转发到 `web-service`：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: simple-ingress
spec:
  ingressClassName: nginx  # 指定使用哪个Ingress Controller
  rules:
    - host: app.example.com  # 当用户访问这个域名时
      http:
        paths:
          - path: /           # 匹配所有路径
            pathType: Prefix
            backend:
              service:
                name: web-service  # 转发到这个Service
                port:
                  number: 80
```

这个配置说的是："当有人访问 `app.example.com` 的任何路径时，把请求转发给 `web-service` 的80端口"。

## 路径匹配类型

你可能注意到了 `pathType: Prefix`。Kubernetes支持两种主要的匹配方式：

| 类型 | 含义 | 例子 |
|------|------|------|
| `Prefix` | 前缀匹配 | `/api` 可以匹配 `/api`、`/api/users`、`/api/orders` |
| `Exact` | 精确匹配 | `/api` 只能匹配 `/api`，不匹配 `/api/users` |

大多数情况下，你会使用 `Prefix`。

## HTTPS配置

在生产环境中，你的网站需要HTTPS。Ingress让这变得很简单：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tls-ingress
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"  # 强制使用HTTPS
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - app.example.com
      secretName: app-tls-secret  # 存放证书的Secret
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: web-service
                port:
                  number: 80
```

证书存放在一个叫 `app-tls-secret` 的Secret里。Ingress Controller会自动使用这个证书来处理HTTPS请求。

## 常见问题

### Q1: 配置了Ingress但访问不了？

**先检查基础问题**：

1. **Ingress Controller装了吗？** Ingress只是规则，没有Controller执行就不会生效
2. **ingressClassName写对了吗？** 这个名字必须和你安装的Controller匹配
3. **后端Service存在吗？** Ingress需要转发到的Service必须存在且正常运行

**原理解释**：Ingress Controller会监听所有Ingress资源，只处理 `ingressClassName` 匹配的那些。如果写错了，Controller会直接忽略你的配置。

### Q2: 访问时出现502错误？

502意味着Ingress Controller收到了请求，但无法连接到后端Service。

**可能的原因**：
- 后端Pod还没准备好（正在启动或健康检查失败）
- Service的端口配置和Pod的实际端口不匹配
- Pod的健康检查没通过，被从Service的Endpoints中移除了

**原理解释**：Ingress Controller -> Service -> Pod 是一个链条。502说明第一步成功了，但后面断了。

### Q3: 上传大文件失败？

默认情况下，Nginx Ingress Controller限制请求体大小为1MB。如果你需要上传大文件，需要调整：

```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"  # 允许100MB
```

**原理解释**：这是一个安全限制，防止恶意用户上传超大文件耗尽服务器资源。根据你的业务需求适当调整即可。

### Q4: 如何让HTTP自动跳转到HTTPS？

```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
```

当用户访问 `http://app.example.com` 时，会自动重定向到 `https://app.example.com`。

## 选择哪个Ingress Controller？

| Controller | 适合场景 |
|------------|----------|
| **Nginx Ingress** | 大多数场景，功能全面，社区活跃 |
| **Traefik** | 需要自动发现和Let's Encrypt自动证书 |
| **AWS ALB** | 在AWS上运行，想用原生负载均衡器 |
| **Istio Gateway** | 已经在用Istio服务网格 |

如果你是初学者或者不确定选什么，就选Nginx Ingress，它是最成熟、文档最全的选择。

## 小结

- **Ingress是什么**：集群的统一流量入口，根据域名和路径将请求路由到不同的Service
- **为什么需要它**：简化外部访问管理，一个入口解决所有问题
- **核心概念**：Ingress定义规则，Ingress Controller执行规则
- **关键配置**：host定义域名，path定义路径，backend定义转发目标

理解了这些，你就掌握了Ingress的核心。其他高级功能（如金丝雀发布、限流、认证等）都是在这个基础上的扩展。

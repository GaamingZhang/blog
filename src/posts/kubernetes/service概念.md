---
date: 2026-01-24
author: Gaaming Zhang
isOriginal: true
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# 外部服务如何访问Kubernetes集群中的Service

## 概述

在Kubernetes集群中,Service是一种抽象,它定义了一组Pod的逻辑集合和访问这些Pod的策略。虽然Kubernetes默认的ClusterIP类型Service只能在集群内部访问,但在实际生产环境中,我们经常需要让外部用户或服务能够访问集群内的应用。

Kubernetes提供了多种方式来实现外部访问,每种方式都有其适用场景、优缺点和配置方法。选择合适的外部访问方式对于应用的性能、安全性和可维护性都至关重要。

本文将详细介绍外部访问Kubernetes Service的各种方法,包括它们的工作原理、配置方式、最佳实践以及常见问题的解决方案。

## Kubernetes Service类型概览

Kubernetes提供了四种主要的Service类型:

### 1. ClusterIP (默认)

ClusterIP是默认的Service类型,它会分配一个集群内部的虚拟IP地址,只能从集群内部访问。这种类型适合内部服务之间的通信。

**特点:**
- 仅集群内部可访问
- 通过集群内部DNS可以解析
- 最轻量级,性能最好
- 不支持外部访问

### 2. NodePort

NodePort在每个节点上开放一个静态端口,外部流量可以通过`<NodeIP>:<NodePort>`访问Service。

**特点:**
- 在ClusterIP基础上,额外在每个节点开放端口
- 端口范围默认为30000-32767
- 简单但不够灵活
- 适合开发测试环境

### 3. LoadBalancer

LoadBalancer类型会创建一个外部负载均衡器(需要云提供商支持),并自动配置将流量转发到Service。

**特点:**
- 需要云提供商支持(AWS ELB、GCP Load Balancer等)
- 自动获得外部可访问的IP地址
- 适合生产环境
- 成本较高(每个Service一个负载均衡器)

### 4. ExternalName

ExternalName类型将Service映射到外部DNS名称,用于访问集群外部的服务。

**特点:**
- 不创建代理
- 返回CNAME记录
- 用于访问外部服务
- 本文不重点讨论(因为是相反的使用场景)

## 外部访问方式详解

### 方式一: NodePort Service

NodePort是最简单直接的外部访问方式,它在集群的每个节点上开放一个相同的端口,外部流量可以通过任意节点的IP和这个端口访问Service。

#### 工作原理

1. 创建NodePort Service时,Kubernetes会:
   - 创建一个ClusterIP(用于集群内访问)
   - 在每个节点上开放指定的端口(默认范围30000-32767)
   - 配置iptables规则,将到达NodePort的流量转发到后端Pod

2. 外部访问流程:
   - 客户端 → 任意节点IP:NodePort
   - 节点 → kube-proxy(iptables/IPVS规则)
   - kube-proxy → 后端Pod(可能在其他节点)

#### 配置示例

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-nodeport-service
spec:
  type: NodePort
  selector:
    app: my-app
  ports:
  - port: 80          # Service端口
    targetPort: 8080  # Pod端口
    nodePort: 30080   # 节点端口(可选,不指定则自动分配)
    protocol: TCP
```

#### 使用Go客户端创建NodePort Service

```go
package main

import (
    "context"
    "fmt"
    "log"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/intstr"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

func createNodePortService(clientset *kubernetes.Clientset, namespace string) error {
    service := &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name: "my-app-nodeport",
        },
        Spec: corev1.ServiceSpec{
            Type: corev1.ServiceTypeNodePort,
            Selector: map[string]string{
                "app": "my-app",
            },
            Ports: []corev1.ServicePort{
                {
                    Name:       "http",
                    Protocol:   corev1.ProtocolTCP,
                    Port:       80,
                    TargetPort: intstr.FromInt(8080),
                    NodePort:   30080, // 可选,留空则自动分配
                },
            },
        },
    }

    result, err := clientset.CoreV1().Services(namespace).Create(
        context.TODO(),
        service,
        metav1.CreateOptions{},
    )

    if err != nil {
        return fmt.Errorf("failed to create service: %w", err)
    }

    fmt.Printf("Created NodePort service: %s\n", result.Name)
    fmt.Printf("Access via: <NodeIP>:%d\n", result.Spec.Ports[0].NodePort)
    return nil
}

func main() {
    // 加载kubeconfig
    config, err := clientcmd.BuildConfigFromFlags("", "/path/to/kubeconfig")
    if err != nil {
        log.Fatal(err)
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        log.Fatal(err)
    }

    if err := createNodePortService(clientset, "default"); err != nil {
        log.Fatal(err)
    }
}
```

#### NodePort优缺点

**优点:**
- 配置简单,不需要额外组件
- 不依赖云提供商
- 适合快速测试和开发环境
- 可以直接通过节点IP访问

**缺点:**
- 端口范围受限(30000-32767)
- 需要暴露节点IP给外部
- 无法提供负载均衡(需要外部负载均衡器)
- 端口管理复杂,容易冲突
- 安全性较差,每个节点都开放端口

#### 适用场景

- 开发和测试环境
- 临时演示
- 预算有限的小型项目
- 需要特定端口的应用
- 配合外部负载均衡器使用

### 方式二: LoadBalancer Service

LoadBalancer是云环境中最常用的外部访问方式,它会自动创建云提供商的负载均衡器,并将流量分发到Service。

#### 工作原理

1. 创建LoadBalancer Service时:
   - Kubernetes创建一个NodePort Service
   - 云控制器(Cloud Controller Manager)调用云提供商API
   - 云提供商创建外部负载均衡器
   - 负载均衡器配置指向所有节点的NodePort
   - Kubernetes更新Service状态,添加外部IP

2. 流量路径:
   - 客户端 → 外部负载均衡器
   - 负载均衡器 → 节点NodePort
   - 节点 → 后端Pod

#### 配置示例

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-loadbalancer-service
  annotations:
    # AWS示例
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    service.beta.kubernetes.io/aws-load-balancer-scheme: "internet-facing"
    # GCP示例
    cloud.google.com/load-balancer-type: "External"
spec:
  type: LoadBalancer
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
  # 可选: 指定外部流量策略
  externalTrafficPolicy: Local  # 或 Cluster(默认)
  # 可选: 指定负载均衡器IP(云提供商支持时)
  loadBalancerIP: "203.0.113.10"
  # 可选: 限制访问来源
  loadBalancerSourceRanges:
  - "192.168.1.0/24"
  - "10.0.0.0/8"
```

#### 使用Go客户端创建LoadBalancer Service

```go
package main

import (
    "context"
    "fmt"
    "time"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/intstr"
    "k8s.io/client-go/kubernetes"
)

func createLoadBalancerService(clientset *kubernetes.Clientset, namespace string) error {
    service := &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name: "my-app-lb",
            Annotations: map[string]string{
                // AWS NLB示例
                "service.beta.kubernetes.io/aws-load-balancer-type": "nlb",
                "service.beta.kubernetes.io/aws-load-balancer-scheme": "internet-facing",
            },
        },
        Spec: corev1.ServiceSpec{
            Type: corev1.ServiceTypeLoadBalancer,
            Selector: map[string]string{
                "app": "my-app",
            },
            Ports: []corev1.ServicePort{
                {
                    Name:       "http",
                    Protocol:   corev1.ProtocolTCP,
                    Port:       80,
                    TargetPort: intstr.FromInt(8080),
                },
            },
            ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeLocal,
        },
    }

    result, err := clientset.CoreV1().Services(namespace).Create(
        context.TODO(),
        service,
        metav1.CreateOptions{},
    )

    if err != nil {
        return fmt.Errorf("failed to create service: %w", err)
    }

    fmt.Printf("Created LoadBalancer service: %s\n", result.Name)
    fmt.Println("Waiting for external IP assignment...")

    // 等待外部IP分配
    return waitForLoadBalancerIP(clientset, namespace, result.Name)
}

func waitForLoadBalancerIP(clientset *kubernetes.Clientset, namespace, name string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("timeout waiting for external IP")
        case <-ticker.C:
            svc, err := clientset.CoreV1().Services(namespace).Get(
                context.TODO(),
                name,
                metav1.GetOptions{},
            )
            if err != nil {
                return err
            }

            if len(svc.Status.LoadBalancer.Ingress) > 0 {
                ingress := svc.Status.LoadBalancer.Ingress[0]
                if ingress.IP != "" {
                    fmt.Printf("External IP assigned: %s\n", ingress.IP)
                    return nil
                }
                if ingress.Hostname != "" {
                    fmt.Printf("External Hostname assigned: %s\n", ingress.Hostname)
                    return nil
                }
            }
            fmt.Println("Still waiting for external IP...")
        }
    }
}
```

#### ExternalTrafficPolicy详解

`externalTrafficPolicy`是LoadBalancer和NodePort Service的重要配置:

**Cluster模式(默认):**
- 流量可以转发到任意节点的Pod
- 可能会二次跳转(节点间转发)
- 更好的负载均衡
- 丢失客户端源IP(除非使用额外配置)

```yaml
spec:
  externalTrafficPolicy: Cluster
```

**Local模式:**
- 流量只转发到本节点的Pod
- 保留客户端源IP
- 避免节点间跳转,性能更好
- 可能导致负载不均衡
- 如果节点上没有Pod,流量会丢失

```yaml
spec:
  externalTrafficPolicy: Local
```

#### 实现源IP保留的示例

```go
package main

import (
    "fmt"
    "log"
    "net/http"
)

// HTTP服务器,用于验证是否保留了客户端IP
func sourceIPHandler(w http.ResponseWriter, r *http.Request) {
    // 获取客户端IP
    clientIP := r.Header.Get("X-Real-IP")
    if clientIP == "" {
        clientIP = r.Header.Get("X-Forwarded-For")
    }
    if clientIP == "" {
        clientIP = r.RemoteAddr
    }

    fmt.Fprintf(w, "Client IP: %s\n", clientIP)
    fmt.Fprintf(w, "X-Forwarded-For: %s\n", r.Header.Get("X-Forwarded-For"))
    fmt.Fprintf(w, "X-Real-IP: %s\n", r.Header.Get("X-Real-IP"))
    
    log.Printf("Request from: %s", clientIP)
}

func main() {
    http.HandleFunc("/", sourceIPHandler)
    log.Println("Starting server on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

对应的Service配置:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: source-ip-app
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local  # 保留源IP
  selector:
    app: source-ip-app
  ports:
  - port: 80
    targetPort: 8080
```

#### LoadBalancer优缺点

**优点:**
- 自动管理外部负载均衡器
- 获得稳定的外部IP或域名
- 云提供商原生支持,高可用
- 适合生产环境
- 支持多种协议(HTTP、TCP、UDP)

**缺点:**
- 需要云提供商支持
- 每个Service创建一个负载均衡器,成本高
- 配置选项受限于云提供商
- 裸金属集群不支持(需要额外方案如MetalLB)
- 可能有创建延迟

#### 适用场景

- 生产环境的主要应用
- 需要高可用和自动故障转移
- 云环境部署
- 需要稳定外部IP的场景
- 对成本不敏感的场景

### 方式三: Ingress

Ingress是Kubernetes中用于管理外部HTTP/HTTPS访问的API对象。它提供了基于域名和路径的路由功能,是生产环境中最推荐的外部访问方式。

#### 工作原理

1. Ingress架构:
   - Ingress资源: 定义路由规则
   - Ingress Controller: 实现路由规则的组件(如Nginx、Traefik)
   - Service: Ingress将流量转发到的后端Service

2. 流量路径:
   - 客户端 → Ingress Controller(通常通过LoadBalancer暴露)
   - Ingress Controller解析域名和路径
   - Ingress Controller → 后端Service
   - Service → Pod

#### Ingress Controller

Ingress本身只是一个API对象,需要Ingress Controller来实现功能。常见的Ingress Controller包括:

- **Nginx Ingress Controller**: 最流行,功能强大
- **Traefik**: 云原生,支持自动服务发现
- **HAProxy**: 高性能
- **Kong**: API网关功能
- **Istio Gateway**: 服务网格集成
- **AWS ALB Ingress Controller**: AWS原生
- **GCE Ingress Controller**: GCP原生

#### 基本Ingress配置

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: simple-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx  # 指定使用的Ingress Controller
  rules:
  - host: example.com
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

#### 多域名和路径路由

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: multi-path-ingress
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  # TLS配置
  tls:
  - hosts:
    - example.com
    - api.example.com
    secretName: tls-secret
  rules:
  # 主站点
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: frontend-service
            port:
              number: 80
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 8080
  # API子域名
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 8080
  # 管理后台
  - host: admin.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: admin-service
            port:
              number: 3000
```

#### 使用Go客户端管理Ingress

```go
package main

import (
    "context"
    "fmt"

    networkingv1 "k8s.io/api/networking/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

func createIngress(clientset *kubernetes.Clientset, namespace string) error {
    pathTypePrefix := networkingv1.PathTypePrefix

    ingress := &networkingv1.Ingress{
        ObjectMeta: metav1.ObjectMeta{
            Name: "my-ingress",
            Annotations: map[string]string{
                "nginx.ingress.kubernetes.io/rewrite-target": "/",
                "nginx.ingress.kubernetes.io/ssl-redirect":   "true",
                "cert-manager.io/cluster-issuer":             "letsencrypt-prod",
            },
        },
        Spec: networkingv1.IngressSpec{
            IngressClassName: stringPtr("nginx"),
            TLS: []networkingv1.IngressTLS{
                {
                    Hosts:      []string{"example.com", "www.example.com"},
                    SecretName: "tls-secret",
                },
            },
            Rules: []networkingv1.IngressRule{
                {
                    Host: "example.com",
                    IngressRuleValue: networkingv1.IngressRuleValue{
                        HTTP: &networkingv1.HTTPIngressRuleValue{
                            Paths: []networkingv1.HTTPIngressPath{
                                {
                                    Path:     "/",
                                    PathType: &pathTypePrefix,
                                    Backend: networkingv1.IngressBackend{
                                        Service: &networkingv1.IngressServiceBackend{
                                            Name: "frontend-service",
                                            Port: networkingv1.ServiceBackendPort{
                                                Number: 80,
                                            },
                                        },
                                    },
                                },
                                {
                                    Path:     "/api",
                                    PathType: &pathTypePrefix,
                                    Backend: networkingv1.IngressBackend{
                                        Service: &networkingv1.IngressServiceBackend{
                                            Name: "api-service",
                                            Port: networkingv1.ServiceBackendPort{
                                                Number: 8080,
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    result, err := clientset.NetworkingV1().Ingresses(namespace).Create(
        context.TODO(),
        ingress,
        metav1.CreateOptions{},
    )

    if err != nil {
        return fmt.Errorf("failed to create ingress: %w", err)
    }

    fmt.Printf("Created ingress: %s\n", result.Name)
    return nil
}

func stringPtr(s string) *string {
    return &s
}

// 获取Ingress状态
func getIngressStatus(clientset *kubernetes.Clientset, namespace, name string) error {
    ingress, err := clientset.NetworkingV1().Ingresses(namespace).Get(
        context.TODO(),
        name,
        metav1.GetOptions{},
    )

    if err != nil {
        return err
    }

    fmt.Printf("Ingress: %s\n", ingress.Name)
    fmt.Println("Load Balancer Ingress:")
    for _, lb := range ingress.Status.LoadBalancer.Ingress {
        if lb.IP != "" {
            fmt.Printf("  IP: %s\n", lb.IP)
        }
        if lb.Hostname != "" {
            fmt.Printf("  Hostname: %s\n", lb.Hostname)
        }
    }

    return nil
}
```

#### 高级Ingress功能

##### 1. 基于路径的路由重写

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rewrite-ingress
  annotations:
    # 移除路径前缀
    nginx.ingress.kubernetes.io/rewrite-target: /$2
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /app(/|$)(.*)
        pathType: Prefix
        backend:
          service:
            name: app-service
            port:
              number: 80
```

##### 2. 金丝雀发布(Canary Deployment)

```yaml
# 主Ingress
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: main-ingress
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app-v1
            port:
              number: 80
---
# 金丝雀Ingress - 10%流量到新版本
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: canary-ingress
  annotations:
    nginx.ingress.kubernetes.io/canary: "true"
    nginx.ingress.kubernetes.io/canary-weight: "10"
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app-v2
            port:
              number: 80
```

##### 3. 认证和授权

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: auth-ingress
  annotations:
    # Basic Auth
    nginx.ingress.kubernetes.io/auth-type: basic
    nginx.ingress.kubernetes.io/auth-secret: basic-auth
    nginx.ingress.kubernetes.io/auth-realm: "Authentication Required"
    
    # 或者 OAuth2代理
    nginx.ingress.kubernetes.io/auth-url: "https://oauth2.example.com/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://oauth2.example.com/oauth2/start"
spec:
  rules:
  - host: secure.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: secure-service
            port:
              number: 80
```

##### 4. 限流和速率限制

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rate-limit-ingress
  annotations:
    nginx.ingress.kubernetes.io/limit-rps: "10"           # 每秒请求数
    nginx.ingress.kubernetes.io/limit-connections: "5"    # 并发连接数
    nginx.ingress.kubernetes.io/limit-burst-multiplier: "3"
spec:
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 80
```

##### 5. CORS配置

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cors-ingress
  annotations:
    nginx.ingress.kubernetes.io/enable-cors: "true"
    nginx.ingress.kubernetes.io/cors-allow-origin: "https://example.com"
    nginx.ingress.kubernetes.io/cors-allow-methods: "GET, POST, PUT, DELETE, OPTIONS"
    nginx.ingress.kubernetes.io/cors-allow-headers: "DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization"
spec:
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 80
```

#### TLS/SSL配置

##### 创建TLS Secret

```bash
# 方法1: 从证书文件创建
kubectl create secret tls tls-secret \
  --cert=path/to/tls.crt \
  --key=path/to/tls.key

# 方法2: 使用cert-manager自动获取Let's Encrypt证书
```

##### 使用cert-manager自动化TLS

```yaml
# 安装cert-manager后,创建ClusterIssuer
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
---
# Ingress自动获取证书
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tls-ingress
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - example.com
    secretName: example-com-tls  # cert-manager会自动创建
  rules:
  - host: example.com
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

#### Ingress优缺点

**优点:**
- 单一入口点管理多个服务
- 基于域名和路径的智能路由
- 支持TLS/SSL终止
- 丰富的流量管理功能(重写、重定向、认证等)
- 节省成本(一个负载均衡器服务多个应用)
- 易于配置和管理

**缺点:**
- 需要额外安装和维护Ingress Controller
- 仅支持HTTP/HTTPS(Layer 7)
- 配置相对复杂
- 不同Controller的注解不兼容
- 增加了一层代理,可能影响性能

#### 适用场景

- 生产环境的Web应用
- 需要基于域名和路径路由的场景
- 需要TLS终止的HTTPS服务
- 微服务架构
- 需要流量管理功能(限流、认证等)
- 成本敏感,多个服务共享入口

### 方式四: ExternalIPs

ExternalIPs允许直接将外部IP地址绑定到Service,流量会直接路由到Service的ClusterIP。

#### 配置示例

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-external-ip-service
spec:
  type: ClusterIP  # 注意:这里用ClusterIP
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
  externalIPs:
  - 192.168.1.100  # 外部IP地址(必须是节点可路由的IP)
  - 203.0.113.10
```

#### ExternalIPs工作原理

1. 指定的外部IP必须由集群节点持有或可路由
2. kube-proxy会在所有节点上配置iptables规则
3. 发送到ExternalIP的流量会被转发到Service
4. 这不会创建实际的负载均衡器

#### 使用场景和注意事项

**适用场景:**
- 裸金属集群,有固定的外部IP
- 需要特定IP地址的场景
- 测试和开发环境

**注意事项:**
- ExternalIPs必须是集群可路由的
- 不提供高可用性
- 存在安全风险(用户可以劫持任意IP)
- 生产环境不推荐使用
- 许多集群管理员会限制ExternalIPs的使用

### 方式五: HostNetwork和HostPort

#### HostNetwork

使用主机网络模式,Pod直接使用节点的网络命名空间。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hostnetwork-pod
spec:
  hostNetwork: true  # 使用主机网络
  containers:
  - name: app
    image: myapp:latest
    ports:
    - containerPort: 80
      hostPort: 80  # 可选
```

**特点:**
- Pod使用节点的网络栈
- Pod的端口直接暴露在节点上
- 性能最好,无额外网络层
- 同一节点只能运行一个此类Pod
- 破坏了网络隔离

#### HostPort

HostPort在节点上开放端口,映射到容器端口,类似Docker的端口映射。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hostport-pod
spec:
  containers:
  - name: app
    image: myapp:latest
    ports:
    - containerPort: 8080
      hostPort: 80  # 节点的80端口映射到容器的8080端口
      protocol: TCP
```

**使用场景:**
- 特殊的系统级服务(如监控agent、日志收集)
- 需要直接访问节点网络的应用
- DaemonSet类型的Pod

**注意事项:**
- 限制Pod调度(端口冲突)
- 破坏可移植性
- 一般不推荐使用

### 方式六: 使用MetalLB(裸金属集群)

MetalLB是为裸金属Kubernetes集群提供LoadBalancer类型Service的实现。

#### MetalLB工作模式

**Layer 2模式:**
- 通过ARP/NDP协议宣告IP
- 简单,不需要路由器支持
- 存在单点故障
- 流量集中在一个节点

**BGP模式:**
- 通过BGP协议宣告路由
- 真正的负载均衡
- 需要路由器支持BGP
- 高可用性更好

#### MetalLB配置示例

```yaml
# MetalLB配置
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: metallb-system
  name: config
data:
  config: |
    address-pools:
    - name: default
      protocol: layer2
      addresses:
      - 192.168.1.240-192.168.1.250
---
# 使用LoadBalancer Service
apiVersion: v1
kind: Service
metadata:
  name: metallb-service
spec:
  type: LoadBalancer
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
```

#### MetalLB适用场景

- 裸金属集群或自建数据中心
- 不依赖云提供商
- 需要LoadBalancer功能
- 有可用的IP地址池

## 外部访问方式对比

下面是各种外部访问方式的详细对比:

| 特性 | NodePort | LoadBalancer | Ingress | ExternalIPs | HostNetwork |
|------|----------|--------------|---------|-------------|-------------|
| **复杂度** | 低 | 中 | 高 | 低 | 低 |
| **成本** | 无 | 高 | 中 | 无 | 无 |
| **负载均衡** | 需外部LB | 自动 | 自动 | 无 | 无 |
| **协议支持** | 任意 | 任意 | HTTP/HTTPS | 任意 | 任意 |
| **7层路由** | 否 | 否 | 是 | 否 | 否 |
| **TLS终止** | 否 | 部分支持 | 是 | 否 | 应用层 |
| **端口限制** | 30000-32767 | 任意 | 80/443 | 任意 | 任意 |
| **云依赖** | 否 | 是 | 否 | 否 | 否 |
| **生产推荐** | 否 | 是 | 强烈推荐 | 否 | 否 |

## 完整的外部访问方案示例

下面是一个完整的生产环境外部访问方案,结合多种技术:

### 架构设计

```
Internet
    ↓
Cloud LoadBalancer (LoadBalancer Service)
    ↓
Ingress Controller (Nginx)
    ↓
┌─────────────┬─────────────┬─────────────┐
│  Frontend   │   API       │   Admin     │
│  Service    │  Service    │  Service    │
└─────────────┴─────────────┴─────────────┘
```

### 完整配置示例

```yaml
# 1. Nginx Ingress Controller的LoadBalancer Service
apiVersion: v1
kind: Service
metadata:
  name: ingress-nginx-controller
  namespace: ingress-nginx
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local  # 保留源IP
  selector:
    app.kubernetes.io/name: ingress-nginx
    app.kubernetes.io/component: controller
  ports:
  - name: http
    port: 80
    targetPort: http
    protocol: TCP
  - name: https
    port: 443
    targetPort: https
    protocol: TCP
---
# 2. 前端应用的Deployment和Service
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - name: frontend
        image: frontend:v1
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: frontend-service
spec:
  selector:
    app: frontend
  ports:
  - port: 80
    targetPort: 80
---
# 3. API应用的Deployment和Service
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
      - name: api
        image: api:v1
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
---
apiVersion: v1
kind: Service
metadata:
  name: api-service
spec:
  selector:
    app: api
  ports:
  - port: 8080
    targetPort: 8080
---
# 4. ClusterIssuer for cert-manager
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod-key
    solvers:
    - http01:
        ingress:
          class: nginx
---
# 5. 主Ingress配置
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: main-ingress
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/limit-rps: "10"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - example.com
    - www.example.com
    - api.example.com
    secretName: example-com-tls
  rules:
  # 主网站
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: frontend-service
            port:
              number: 80
  - host: www.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: frontend-service
            port:
              number: 80
  # API
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 8080
```

### Go应用示例 - 处理外部流量

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
)

type Server struct {
    startTime time.Time
    hostname  string
}

func NewServer() *Server {
    hostname, _ := os.Hostname()
    return &Server{
        startTime: time.Now(),
        hostname:  hostname,
    }
}

// 健康检查端点
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

// 就绪检查端点
func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
    // 可以在这里检查数据库连接等
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Ready"))
}

// API端点示例
func (s *Server) apiHandler(w http.ResponseWriter, r *http.Request) {
    // 获取客户端信息
    clientIP := r.Header.Get("X-Real-IP")
    if clientIP == "" {
        clientIP = r.Header.Get("X-Forwarded-For")
    }
    if clientIP == "" {
        clientIP = r.RemoteAddr
    }

    response := map[string]interface{}{
        "message":    "Hello from Kubernetes!",
        "hostname":   s.hostname,
        "uptime":     time.Since(s.startTime).String(),
        "client_ip":  clientIP,
        "path":       r.URL.Path,
        "method":     r.Method,
        "user_agent": r.UserAgent(),
        "timestamp":  time.Now().Format(time.RFC3339),
    }

    // 记录请求
    log.Printf("Request from %s to %s", clientIP, r.URL.Path)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// 处理CORS预检请求
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next(w, r)
    }
}

// 日志中间件
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf(
            "%s %s %s %s",
            r.Method,
            r.RequestURI,
            r.RemoteAddr,
            time.Since(start),
        )
    })
}

func main() {
    server := NewServer()

    mux := http.NewServeMux()
    
    // 注册处理器
    mux.HandleFunc("/health", server.healthHandler)
    mux.HandleFunc("/ready", server.readyHandler)
    mux.HandleFunc("/api/", corsMiddleware(server.apiHandler))
    mux.HandleFunc("/", server.apiHandler)

    // 应用中间件
    handler := loggingMiddleware(mux)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    addr := fmt.Sprintf(":%s", port)
    log.Printf("Server starting on %s", addr)
    log.Printf("Hostname: %s", server.hostname)

    if err := http.ListenAndServe(addr, handler); err != nil {
        log.Fatal(err)
    }
}
```

## 安全最佳实践

### 1. 网络策略

使用NetworkPolicy限制Pod的网络访问:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: api-network-policy
spec:
  podSelector:
    matchLabels:
      app: api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # 只允许来自Ingress Controller的流量
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  # 允许访问DNS
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53
  # 允许访问数据库
  - to:
    - podSelector:
        matchLabels:
          app: database
    ports:
    - protocol: TCP
      port: 5432
```

### 2. TLS/SSL配置

始终使用TLS加密外部流量:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: secure-ingress
  annotations:
    # 强制HTTPS
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    # 使用现代TLS配置
    nginx.ingress.kubernetes.io/ssl-protocols: "TLSv1.2 TLSv1.3"
    nginx.ingress.kubernetes.io/ssl-ciphers: "ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384"
    # HSTS
    nginx.ingress.kubernetes.io/configuration-snippet: |
      add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
spec:
  tls:
  - hosts:
    - secure.example.com
    secretName: tls-secret
  rules:
  - host: secure.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: secure-service
            port:
              number: 80
```

### 3. IP白名单

限制访问来源IP:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: whitelist-ingress
  annotations:
    # Nginx Ingress
    nginx.ingress.kubernetes.io/whitelist-source-range: "10.0.0.0/8,192.168.1.0/24"
spec:
  rules:
  - host: admin.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: admin-service
            port:
              number: 80
```

或在LoadBalancer Service级别:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: restricted-lb
spec:
  type: LoadBalancer
  loadBalancerSourceRanges:
  - "203.0.113.0/24"
  - "198.51.100.0/24"
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
```

### 4. 认证和授权

在Ingress层面添加认证:

```yaml
# 创建htpasswd认证文件
# htpasswd -c auth admin
# kubectl create secret generic basic-auth --from-file=auth

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: auth-ingress
  annotations:
    nginx.ingress.kubernetes.io/auth-type: basic
    nginx.ingress.kubernetes.io/auth-secret: basic-auth
    nginx.ingress.kubernetes.io/auth-realm: "Authentication Required"
spec:
  rules:
  - host: secure.example.com
    http:
      paths:
      - path: /admin
        pathType: Prefix
        backend:
          service:
            name: admin-service
            port:
              number: 80
```

### 5. 限流和DDoS防护

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rate-limited-ingress
  annotations:
    # 全局限流
    nginx.ingress.kubernetes.io/limit-rps: "100"
    nginx.ingress.kubernetes.io/limit-connections: "10"
    
    # 基于IP的限流
    nginx.ingress.kubernetes.io/limit-whitelist: "10.0.0.0/8"
    
    # 请求体大小限制
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
spec:
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 8080
```

## 监控和可观测性

### 1. Prometheus监控

为Ingress Controller配置Prometheus监控:

```go
package main

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
    "time"
)

var (
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )

    activeConnections = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_connections",
            Help: "Number of active connections",
        },
    )
)

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

func metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        activeConnections.Inc()
        defer activeConnections.Dec()

        rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        next.ServeHTTP(rw, r)

        duration := time.Since(start).Seconds()
        httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
        httpRequestsTotal.WithLabelValues(
            r.Method,
            r.URL.Path,
            http.StatusText(rw.statusCode),
        ).Inc()
    })
}

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello World"))
    })
    mux.Handle("/metrics", promhttp.Handler())

    handler := metricsMiddleware(mux)
    http.ListenAndServe(":8080", handler)
}
```

### 2. 日志聚合

在应用中输出结构化日志:

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "time"
)

type LogEntry struct {
    Timestamp  string `json:"timestamp"`
    Method     string `json:"method"`
    Path       string `json:"path"`
    ClientIP   string `json:"client_ip"`
    UserAgent  string `json:"user_agent"`
    StatusCode int    `json:"status_code"`
    Duration   string `json:"duration"`
}

func logRequest(r *http.Request, statusCode int, duration time.Duration) {
    entry := LogEntry{
        Timestamp:  time.Now().Format(time.RFC3339),
        Method:     r.Method,
        Path:       r.URL.Path,
        ClientIP:   r.Header.Get("X-Real-IP"),
        UserAgent:  r.UserAgent(),
        StatusCode: statusCode,
        Duration:   duration.String(),
    }

    logJSON, _ := json.Marshal(entry)
    log.Println(string(logJSON))
}
```

## 故障排查

### 常用排查命令

```bash
# 查看Service详情
kubectl describe service <service-name>

# 查看Service端点
kubectl get endpoints <service-name>

# 查看Ingress详情
kubectl describe ingress <ingress-name>

# 查看Ingress Controller日志
kubectl logs -n ingress-nginx <ingress-controller-pod>

# 测试Service连通性(从集群内部)
kubectl run -it --rm debug --image=busybox --restart=Never -- sh
wget -O- http://<service-name>.<namespace>.svc.cluster.local

# 查看Service的iptables规则
iptables -t nat -L -n | grep <service-name>

# 查看kube-proxy日志
kubectl logs -n kube-system <kube-proxy-pod>
```

### 诊断工具脚本

```go
package main

import (
    "context"
    "fmt"
    "time"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

func diagnoseService(clientset *kubernetes.Clientset, namespace, serviceName string) {
    ctx := context.Background()

    // 获取Service信息
    svc, err := clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
    if err != nil {
        fmt.Printf("Error getting service: %v\n", err)
        return
    }

    fmt.Printf("Service: %s\n", svc.Name)
    fmt.Printf("Type: %s\n", svc.Spec.Type)
    fmt.Printf("ClusterIP: %s\n", svc.Spec.ClusterIP)
    
    // 检查端口配置
    fmt.Println("\nPorts:")
    for _, port := range svc.Spec.Ports {
        fmt.Printf("  %s: %d -> %s\n", port.Name, port.Port, port.TargetPort.String())
        if port.NodePort != 0 {
            fmt.Printf("  NodePort: %d\n", port.NodePort)
        }
    }

    // 检查Endpoints
    endpoints, err := clientset.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
    if err != nil {
        fmt.Printf("Error getting endpoints: %v\n", err)
        return
    }

    fmt.Println("\nEndpoints:")
    if len(endpoints.Subsets) == 0 {
        fmt.Println("  WARNING: No endpoints found! Check pod labels and readiness.")
    } else {
        for _, subset := range endpoints.Subsets {
            for _, addr := range subset.Addresses {
                fmt.Printf("  %s", addr.IP)
                if addr.TargetRef != nil {
                    fmt.Printf(" (Pod: %s)", addr.TargetRef.Name)
                }
                fmt.Println()
            }
        }
    }

    // 检查LoadBalancer状态
    if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
        fmt.Println("\nLoadBalancer Status:")
        if len(svc.Status.LoadBalancer.Ingress) == 0 {
            fmt.Println("  WARNING: No external IP assigned yet")
        } else {
            for _, ingress := range svc.Status.LoadBalancer.Ingress {
                if ingress.IP != "" {
                    fmt.Printf("  External IP: %s\n", ingress.IP)
                }
                if ingress.Hostname != "" {
                    fmt.Printf("  External Hostname: %s\n", ingress.Hostname)
                }
            }
        }
    }

    // 检查选择器匹配的Pod
    if svc.Spec.Selector != nil {
        fmt.Println("\nMatching Pods:")
        labelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{
            MatchLabels: svc.Spec.Selector,
        })
        pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
            LabelSelector: labelSelector,
        })
        if err != nil {
            fmt.Printf("Error listing pods: %v\n", err)
        } else if len(pods.Items) == 0 {
            fmt.Println("  WARNING: No pods match the service selector!")
            fmt.Printf("  Selector: %v\n", svc.Spec.Selector)
        } else {
            for _, pod := range pods.Items {
                fmt.Printf("  %s - Phase: %s, Ready: ", pod.Name, pod.Status.Phase)
                ready := false
                for _, cond := range pod.Status.Conditions {
                    if cond.Type == corev1.PodReady {
                        ready = cond.Status == corev1.ConditionTrue
                        break
                    }
                }
                fmt.Println(ready)
            }
        }
    }
}

func main() {
    config, err := clientcmd.BuildConfigFromFlags("", "/path/to/kubeconfig")
    if err != nil {
        panic(err)
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        panic(err)
    }

    diagnoseService(clientset, "default", "my-service")
}
```

## 总结

Kubernetes提供了多种外部访问Service的方式,每种方式都有其适用场景:

**推荐方案:**
1. **生产环境Web应用**: Ingress + LoadBalancer组合
2. **云环境非HTTP服务**: LoadBalancer Service
3. **裸金属集群**: MetalLB + Ingress或NodePort + 外部负载均衡器
4. **开发测试**: NodePort或kubectl port-forward

**关键决策因素:**
- 应用协议(HTTP vs TCP/UDP)
- 基础设施类型(云 vs 裸金属)
- 成本预算
- 流量管理需求
- 安全要求
- 运维复杂度

**最佳实践:**
- 优先使用Ingress处理HTTP/HTTPS流量
- 启用TLS/SSL加密
- 配置网络策略限制访问
- 实施限流和认证
- 监控和日志记录
- 保留客户端源IP(使用externalTrafficPolicy: Local)
- 使用cert-manager自动化证书管理

通过正确选择和配置外部访问方式,可以构建安全、高效、可扩展的Kubernetes服务对外暴露方案。

---

## 常见问题FAQ

### Q1: NodePort、LoadBalancer和Ingress有什么区别?应该如何选择?

**核心区别:**

**NodePort:**
- 在每个节点上开放30000-32767范围内的端口
- 通过`<任意节点IP>:<NodePort>`访问
- 不提供负载均衡,需要外部LB
- 最简单但功能有限

**LoadBalancer:**
- 自动创建云提供商的负载均衡器
- 获得外部可访问的IP/域名
- 支持任意协议(TCP/UDP)
- 每个Service一个LB,成本较高

**Ingress:**
- 7层(HTTP/HTTPS)路由和负载均衡
- 支持基于域名和路径的智能路由
- 多个服务共享一个入口点
- 功能最丰富(SSL终止、认证、限流等)

**选择建议:**

```yaml
# 场景1: HTTP/HTTPS应用 → 使用Ingress
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: web-app
spec:
  rules:
  - host: myapp.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: web-service
            port:
              number: 80

# 场景2: 非HTTP服务(如数据库、gRPC) → 使用LoadBalancer
apiVersion: v1
kind: Service
metadata:
  name: database
spec:
  type: LoadBalancer
  selector:
    app: postgres
  ports:
  - port: 5432

# 场景3: 开发测试环境 → 使用NodePort
apiVersion: v1
kind: Service
metadata:
  name: test-service
spec:
  type: NodePort
  selector:
    app: test-app
  ports:
  - port: 80
    nodePort: 30080
```

**组合使用:**
生产环境最佳实践是Ingress + LoadBalancer组合:LoadBalancer暴露Ingress Controller,Ingress管理应用路由。

### Q2: 如何在Kubernetes中实现零宕机的滚动更新和外部访问?

零宕机更新需要正确配置多个组件协同工作:

**1. 配置就绪探针:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1  # 最多1个Pod不可用
      maxSurge: 1        # 最多多1个Pod
  minReadySeconds: 10    # Pod稳定10秒后才视为就绪
  template:
    spec:
      containers:
      - name: app
        image: myapp:v2
        ports:
        - containerPort: 8080
        # 关键:就绪探针
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          successThreshold: 1
          failureThreshold: 1
        # 优雅关闭
        lifecycle:
          preStop:
            exec:
              command: ["sh", "-c", "sleep 15"]
        terminationGracePeriodSeconds: 30
```

**2. 应用层实现优雅关闭:**

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync/atomic"
    "syscall"
    "time"
)

type App struct {
    server *http.Server
    ready  atomic.Bool
}

func (app *App) gracefulShutdown() {
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
    <-quit

    log.Println("Shutting down server...")
    
    // 1. 标记为未就绪,停止接收新请求
    app.ready.Store(false)
    log.Println("Marked as not ready")
    
    // 2. 等待负载均衡器/Service更新(>就绪探针间隔)
    time.Sleep(15 * time.Second)
    
    // 3. 优雅关闭,等待现有请求完成
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := app.server.Shutdown(ctx); err != nil {
        log.Printf("Forced shutdown: %v", err)
    }
    log.Println("Server stopped")
}

func (app *App) readyHandler(w http.ResponseWriter, r *http.Request) {
    if app.ready.Load() {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
}

func main() {
    app := &App{}
    app.ready.Store(true)

    mux := http.NewServeMux()
    mux.HandleFunc("/ready", app.readyHandler)
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(2 * time.Second) // 模拟慢请求
        w.Write([]byte("OK"))
    })

    app.server = &http.Server{Addr: ":8080", Handler: mux}

    go app.gracefulShutdown()
    
    log.Println("Server starting on :8080")
    if err := app.server.ListenAndServe(); err != http.ErrServerClosed {
        log.Fatal(err)
    }
}
```

**3. Service和Ingress配置:**

```yaml
# Service使用合适的sessionAffinity
apiVersion: v1
kind: Service
metadata:
  name: myapp
spec:
  selector:
    app: myapp
  sessionAffinity: ClientIP  # 可选:保持会话亲和性
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 3600
  ports:
  - port: 80
    targetPort: 8080
---
# Ingress配置连接保持
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: myapp
  annotations:
    nginx.ingress.kubernetes.io/upstream-keepalive-connections: "50"
    nginx.ingress.kubernetes.io/upstream-keepalive-timeout: "60"
spec:
  rules:
  - host: myapp.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: myapp
            port:
              number: 80
```

**关键点:**
- preStop hook等待15秒,给Service时间更新端点
- 就绪探针快速检测(5秒间隔),及时移除不健康Pod
- minReadySeconds确保新Pod稳定后才接收流量
- 应用实现优雅关闭,等待请求完成
- maxUnavailable=1确保始终有足够副本提供服务

### Q3: 如何保留客户端真实IP地址?为什么有时候获取不到?

**问题原因:**

在Kubernetes中,流量经过多层代理(LoadBalancer → NodePort → Pod),默认情况下会丢失客户端源IP。

**解决方案1: 使用externalTrafficPolicy: Local**

```yaml
apiVersion: v1
kind: Service
metadata:
  name: preserve-ip-service
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local  # 关键配置
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
```

**工作原理:**
- `Cluster`模式(默认):流量可以转发到任意节点的Pod,会经过kube-proxy的SNAT,丢失源IP
- `Local`模式:流量只转发到接收流量节点上的Pod,避免额外跳转,保留源IP

**注意事项:**
- Local模式下,如果节点上没有Pod,流量会丢失
- 可能导致负载不均衡
- 建议配合Pod反亲和性使用

**解决方案2: Ingress层面保留源IP**

Nginx Ingress Controller默认会通过`X-Forwarded-For`和`X-Real-IP`头传递源IP:

```go
func getClientIP(r *http.Request) string {
    // 优先级: X-Real-IP > X-Forwarded-For > RemoteAddr
    if ip := r.Header.Get("X-Real-IP"); ip != "" {
        return ip
    }
    
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        // X-Forwarded-For可能包含多个IP,取第一个
        ips := strings.Split(xff, ",")
        if len(ips) > 0 {
            return strings.TrimSpace(ips[0])
        }
    }
    
    // 降级到RemoteAddr
    return r.RemoteAddr
}
```

**配置Ingress Controller保留源IP:**

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ingress-nginx-controller
  namespace: ingress-nginx
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local  # 保留源IP
  selector:
    app.kubernetes.io/name: ingress-nginx
  ports:
  - port: 80
    targetPort: http
  - port: 443
    targetPort: https
```

**验证源IP保留:**

```go
package main

import (
    "encoding/json"
    "net/http"
    "strings"
)

type IPInfo struct {
    RemoteAddr    string   `json:"remote_addr"`
    XRealIP       string   `json:"x_real_ip"`
    XForwardedFor string   `json:"x_forwarded_for"`
    ClientIP      string   `json:"client_ip"`
    AllHeaders    map[string][]string `json:"all_headers"`
}

func ipInfoHandler(w http.ResponseWriter, r *http.Request) {
    clientIP := r.Header.Get("X-Real-IP")
    if clientIP == "" {
        if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
            ips := strings.Split(xff, ",")
            clientIP = strings.TrimSpace(ips[0])
        }
    }
    if clientIP == "" {
        clientIP = r.RemoteAddr
    }

    info := IPInfo{
        RemoteAddr:    r.RemoteAddr,
        XRealIP:       r.Header.Get("X-Real-IP"),
        XForwardedFor: r.Header.Get("X-Forwarded-For"),
        ClientIP:      clientIP,
        AllHeaders:    r.Header,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(info)
}

func main() {
    http.HandleFunc("/ip-info", ipInfoHandler)
    http.ListenAndServe(":8080", nil)
}
```

### Q4: LoadBalancer类型的Service一直处于Pending状态,如何解决?

**问题诊断:**

```bash
# 查看Service状态
kubectl get svc my-loadbalancer-service
# 输出: EXTERNAL-IP列显示<pending>

# 查看详细信息
kubectl describe svc my-loadbalancer-service
# 查看Events部分的错误信息
```

**常见原因和解决方案:**

**原因1: 集群不支持LoadBalancer(裸金属集群)**

```bash
# 检查是否有cloud controller manager
kubectl get pods -n kube-system | grep cloud-controller

# 解决方案:安装MetalLB
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.0/config/manifests/metallb-native.yaml

# 配置IP地址池
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: metallb-system
  name: config
data:
  config: |
    address-pools:
    - name: default
      protocol: layer2
      addresses:
      - 192.168.1.240-192.168.1.250
EOF
```

**原因2: 云配额限制**

```bash
# AWS示例:检查ELB配额
aws elbv2 describe-account-limits

# 解决:增加配额或删除不用的负载均衡器
aws elbv2 delete-load-balancer --load-balancer-arn <arn>
```

**原因3: 配置错误或权限问题**

```yaml
# 检查Service配置
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    # AWS示例:检查注解是否正确
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    service.beta.kubernetes.io/aws-load-balancer-scheme: "internet-facing"
spec:
  type: LoadBalancer
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
```

**原因4: 网络配置问题**

```bash
# 检查VPC/子网配置(云环境)
# 确保子网有足够的IP地址
# 确保安全组规则正确

# 检查cloud controller manager日志
kubectl logs -n kube-system <cloud-controller-pod>
```

**临时解决方案:使用NodePort + 外部负载均衡器**

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: NodePort
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30080
```

然后手动配置外部负载均衡器指向所有节点的30080端口。

### Q5: 如何在一个Ingress中实现多个域名和路径的复杂路由?

Ingress支持非常灵活的路由配置,可以基于域名和路径组合进行路由:

**完整示例:多域名、多路径、多服务**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: complex-routing
  annotations:
    # 通用配置
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - example.com
    - www.example.com
    - api.example.com
    - admin.example.com
    secretName: multi-domain-tls
  rules:
  # 规则1: 主域名 - 前端应用
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: frontend
            port:
              number: 80
      - path: /blog
        pathType: Prefix
        backend:
          service:
            name: blog
            port:
              number: 3000
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: api
            port:
              number: 8080
  
  # 规则2: www子域名 - 重定向到主域名
  - host: www.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: frontend
            port:
              number: 80
  
  # 规则3: API子域名 - 完整API服务
  - host: api.example.com
    http:
      paths:
      - path: /v1
        pathType: Prefix
        backend:
          service:
            name: api-v1
            port:
              number: 8080
      - path: /v2
        pathType: Prefix
        backend:
          service:
            name: api-v2
            port:
              number: 8080
  
  # 规则4: 管理后台 - 需要认证
  - host: admin.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: admin
            port:
              number: 3000
```

**路径类型说明:**

```yaml
# Prefix:前缀匹配(最常用)
- path: /api
  pathType: Prefix  # 匹配 /api, /api/, /api/users 等

# Exact:精确匹配
- path: /api
  pathType: Exact  # 只匹配 /api

# ImplementationSpecific:依赖Ingress Controller实现
- path: /api/*
  pathType: ImplementationSpecific
```

**高级路由:使用正则表达式**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: regex-routing
  annotations:
    nginx.ingress.kubernetes.io/use-regex: "true"
    nginx.ingress.kubernetes.io/rewrite-target: /$2
spec:
  rules:
  - host: example.com
    http:
      paths:
      # 匹配 /api/v1/*, /api/v2/* 等
      - path: /api/(v[0-9]+)/(.*)
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 8080
```

**实现子路径应用部署:**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: subpath-apps
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /$2
    nginx.ingress.kubernetes.io/configuration-snippet: |
      rewrite ^(/app1)$ $1/ permanent;
spec:
  rules:
  - host: example.com
    http:
      paths:
      # /app1/* → app1-service/
      - path: /app1(/|$)(.*)
        pathType: Prefix
        backend:
          service:
            name: app1-service
            port:
              number: 80
      # /app2/* → app2-service/
      - path: /app2(/|$)(.*)
        pathType: Prefix
        backend:
          service:
            name: app2-service
            port:
              number: 80
```

**使用Go动态生成Ingress配置:**

```go
package main

import (
    "context"
    "fmt"

    networkingv1 "k8s.io/api/networking/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

type RouteConfig struct {
    Host    string
    Path    string
    Service string
    Port    int32
}

func createComplexIngress(clientset *kubernetes.Clientset, routes []RouteConfig) error {
    pathTypePrefix := networkingv1.PathTypePrefix
    
    // 按host分组路由
    hostRoutes := make(map[string][]RouteConfig)
    for _, route := range routes {
        hostRoutes[route.Host] = append(hostRoutes[route.Host], route)
    }
    
    // 构建Ingress规则
    var rules []networkingv1.IngressRule
    for host, routes := range hostRoutes {
        var paths []networkingv1.HTTPIngressPath
        for _, route := range routes {
            paths = append(paths, networkingv1.HTTPIngressPath{
                Path:     route.Path,
                PathType: &pathTypePrefix,
                Backend: networkingv1.IngressBackend{
                    Service: &networkingv1.IngressServiceBackend{
                        Name: route.Service,
                        Port: networkingv1.ServiceBackendPort{
                            Number: route.Port,
                        },
                    },
                },
            })
        }
        
        rules = append(rules, networkingv1.IngressRule{
            Host: host,
            IngressRuleValue: networkingv1.IngressRuleValue{
                HTTP: &networkingv1.HTTPIngressRuleValue{
                    Paths: paths,
                },
            },
        })
    }
    
    ingress := &networkingv1.Ingress{
        ObjectMeta: metav1.ObjectMeta{
            Name: "dynamic-ingress",
        },
        Spec: networkingv1.IngressSpec{
            IngressClassName: stringPtr("nginx"),
            Rules:            rules,
        },
    }
    
    _, err := clientset.NetworkingV1().Ingresses("default").Create(
        context.TODO(),
        ingress,
        metav1.CreateOptions{},
    )
    
    return err
}

func stringPtr(s string) *string {
    return &s
}

func main() {
    routes := []RouteConfig{
        {Host: "example.com", Path: "/", Service: "frontend", Port: 80},
        {Host: "example.com", Path: "/api", Service: "api", Port: 8080},
        {Host: "api.example.com", Path: "/v1", Service: "api-v1", Port: 8080},
        {Host: "api.example.com", Path: "/v2", Service: "api-v2", Port: 8080},
    }
    
    // ... 创建clientset
    // createComplexIngress(clientset, routes)
}
```

这样可以灵活地管理复杂的路由配置,支持多域名、多路径、多服务的场景。
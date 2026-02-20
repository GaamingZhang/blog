---
date: 2026-01-30
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# CoreDNS与服务发现

## 为什么需要服务发现？

假设你在一个大公司工作，公司里有财务部、人事部、技术部等很多部门。如果每次你需要找财务部报销，都要先问"财务部现在在哪个房间？电话多少？"，那效率会非常低。更好的方式是什么？公司有一个前台或者通讯录，你只需要说"我要找财务部"，前台就能帮你转接过去。

在Kubernetes中，CoreDNS就扮演这个"前台"的角色：

- 你的应用不需要知道其他服务的具体IP地址
- 只需要知道服务的名字，比如"mysql"或"redis"
- CoreDNS会帮你把名字翻译成实际的IP地址

这就是**服务发现**的核心价值：**让应用之间可以通过名字互相找到对方，而不用关心IP地址**。

## DNS是如何工作的？

当你在浏览器输入 `www.google.com` 时，电脑其实不知道Google服务器在哪里。它会问DNS服务器："google.com的IP是多少？"，DNS服务器回答"是142.250.xx.xx"，然后浏览器才能访问。

CoreDNS在Kubernetes里做的事情完全一样，只不过它管理的是集群内部的"域名"。

## Kubernetes中的DNS命名规则

在Kubernetes中，每个Service都会自动获得一个DNS名字，格式是：

```
<服务名>.<命名空间>.svc.cluster.local
```

举几个例子：
- `mysql.default.svc.cluster.local` - default命名空间里的mysql服务
- `redis.cache.svc.cluster.local` - cache命名空间里的redis服务
- `api.production.svc.cluster.local` - production命名空间里的api服务

**好消息是**：如果你在同一个命名空间里，可以省略后面的部分：
- 同namespace：直接用 `mysql` 就行
- 跨namespace：用 `redis.cache` 就行

## 实际使用示例

假设你有一个Web应用需要连接数据库，在Pod的配置中可以这样写：

```yaml
env:
  # 同namespace访问，直接用服务名
  - name: DB_HOST
    value: "mysql"

  # 跨namespace访问，加上namespace名
  - name: CACHE_HOST
    value: "redis.cache"
```

你的应用代码直接用这些名字就能连接到对应的服务，完全不用关心IP地址是多少。

## 为什么不直接用IP地址？

你可能会想：为什么不直接把Service的IP地址写死呢？原因有几个：

1. **Pod会重启**：Pod重启后IP会变化，写死的IP就失效了
2. **Service IP可能变化**：虽然Service IP相对稳定，但重新创建Service时会变
3. **难以维护**：想象一下要记住几十个IP地址...
4. **无法跨环境**：开发、测试、生产环境的IP肯定不同

用DNS名字就没有这些问题，因为名字是固定的，IP变化了DNS会自动更新。

## Headless Service：特殊的DNS需求

普通Service的DNS解析会返回Service的虚拟IP，所有请求会被负载均衡分发到后端Pod。但有时候你需要直接拿到每个Pod的IP，比如：

- 分布式数据库需要知道所有节点
- 需要点对点通信的应用

这时候就需要**Headless Service**（无头服务）：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: mysql-headless
spec:
  clusterIP: None  # 这一行让它变成Headless
  selector:
    app: mysql
  ports:
    - port: 3306
```

普通Service vs Headless Service的DNS解析：

| 类型 | DNS解析结果 |
|------|-------------|
| 普通Service | 返回Service的虚拟IP（一个IP） |
| Headless Service | 直接返回所有Pod的IP（多个IP） |

配合StatefulSet使用时，每个Pod还会有固定的DNS名字：
- `mysql-0.mysql-headless.default.svc.cluster.local`
- `mysql-1.mysql-headless.default.svc.cluster.local`

## Pod的DNS配置

每个Pod里都有一个 `/etc/resolv.conf` 文件，告诉Pod如何进行DNS查询：

```
nameserver 10.96.0.10
search default.svc.cluster.local svc.cluster.local cluster.local
options ndots:5
```

这里有个重要的参数 `ndots:5`，意思是：**如果域名里的点少于5个，就先尝试加上search域来查询**。

比如你查询 `mysql`，它会按顺序尝试：
1. `mysql.default.svc.cluster.local`
2. `mysql.svc.cluster.local`
3. `mysql.cluster.local`
4. `mysql`（作为绝对域名）

这就是为什么你只写 `mysql` 就能访问到 `mysql.default.svc.cluster.local`。

## 常见问题

### Q1: Pod无法解析Service名称？

**首先检查基础问题**：

1. **Service存在吗？** 确认Service已经创建
2. **命名空间对吗？** 跨namespace要写完整名字
3. **CoreDNS正常吗？** 检查kube-system里的CoreDNS Pod

**原理解释**：DNS查询失败可能是CoreDNS没运行，也可能是网络问题。先确认CoreDNS健康，再排查其他原因。

### Q2: 访问外部网站很慢或失败？

这通常是DNS查询效率问题。由于 `ndots:5` 的设置，访问 `api.example.com` 时会先尝试加search域：
1. `api.example.com.default.svc.cluster.local` （失败）
2. `api.example.com.svc.cluster.local` （失败）
3. ...

这些多余的查询会拖慢速度。

**解决办法**：在域名末尾加一个点，告诉DNS这是一个绝对域名：

```yaml
env:
  - name: EXTERNAL_API
    value: "api.example.com."  # 注意末尾的点
```

### Q3: 为什么同namespace可以用短名称？

因为Pod的 `/etc/resolv.conf` 配置了search域。当你查询 `mysql` 时，系统会自动尝试 `mysql.default.svc.cluster.local`，这正好是完整的Service DNS名称。

### Q4: ExternalName Service是什么？

有时候你想在集群内用一个统一的名字访问外部服务：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-db
spec:
  type: ExternalName
  externalName: database.company.com
```

这样，集群内的Pod访问 `external-db` 时，DNS会返回 `database.company.com` 的CNAME记录。好处是如果外部服务地址变了，只需要修改这一个地方。

## 核心概念小结

| 概念 | 解释 |
|------|------|
| CoreDNS | Kubernetes的DNS服务器，负责服务名到IP的解析 |
| 服务发现 | 通过名字找到服务，不需要知道IP |
| DNS名称格式 | `<服务>.<命名空间>.svc.cluster.local` |
| Headless Service | clusterIP为None，DNS直接返回Pod IP |
| ndots | 控制什么时候使用search域 |

理解了这些，你就掌握了Kubernetes服务发现的核心原理。记住最重要的一点：**用名字，不用IP**。

## 参考资源

- [Kubernetes DNS 官方文档](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/)
- [CoreDNS 文档](https://coredns.io/manual/toc/)
- [服务发现原理](https://kubernetes.io/docs/concepts/services-networking/service/#dns)

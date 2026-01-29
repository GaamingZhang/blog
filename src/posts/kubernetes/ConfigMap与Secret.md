---
date: 2026-01-26
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# ConfigMap与Secret：应用配置管理

## 为什么需要ConfigMap和Secret？

想象一下，你开发了一个应用程序，需要连接数据库。数据库的地址、端口、用户名、密码这些信息，你会放在哪里？

最简单的方式是直接写在代码里，但这样做有几个严重的问题：

1. **环境不同，配置也不同**：开发环境连接测试数据库，生产环境连接正式数据库。如果配置写死在代码里，每次切换环境都要改代码重新打包。

2. **安全风险**：密码写在代码里，意味着任何能看到代码的人都知道你的密码。代码提交到Git仓库后，密码就永久留在了历史记录中。

3. **修改成本高**：想改个日志级别，还得重新构建镜像、重新部署，这太麻烦了。

Kubernetes的解决方案是：**把配置从应用镜像中分离出来**。这就是ConfigMap和Secret存在的意义。

## ConfigMap和Secret是什么？

你可以把ConfigMap和Secret想象成两个"配置保险箱"：

- **ConfigMap**：存放普通配置，就像一个透明的文件夹，谁都可以看里面的内容
- **Secret**：存放敏感信息，像一个带锁的保险柜，虽然不是绝对安全，但至少不会随意暴露

两者的使用方式几乎完全一样，区别只在于**存放内容的敏感程度**：
- 数据库地址、日志级别、功能开关 → 放ConfigMap
- 数据库密码、API密钥、证书私钥 → 放Secret

## ConfigMap的工作原理

### 创建ConfigMap

最直观的方式是用YAML文件声明：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  # 简单的键值对
  database_host: "mysql.default.svc.cluster.local"
  log_level: "info"

  # 也可以存放整个配置文件的内容
  application.properties: |
    server.port=8080
    spring.datasource.url=jdbc:mysql://mysql:3306/mydb
```

这里的`data`部分就是你要存储的配置。可以是简单的键值对，也可以是完整的配置文件内容。

### 如何让Pod使用ConfigMap？

有两种主要方式，各有适用场景：

**方式一：作为环境变量注入**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  containers:
    - name: app
      image: myapp:1.0
      env:
        - name: DATABASE_HOST
          valueFrom:
            configMapKeyRef:
              name: app-config
              key: database_host
```

这种方式的特点是：应用通过读取环境变量获取配置，简单直接。但**环境变量在Pod启动时就确定了，ConfigMap更新后不会自动生效**，必须重启Pod。

**方式二：作为文件挂载**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  containers:
    - name: app
      image: myapp:1.0
      volumeMounts:
        - name: config-volume
          mountPath: /etc/config
  volumes:
    - name: config-volume
      configMap:
        name: app-config
```

这种方式会把ConfigMap的每个key变成`/etc/config`目录下的一个文件。好处是**ConfigMap更新后，文件内容会自动更新**（通常需要等待10秒到1分钟）。

### 配置热更新的注意事项

这里有个容易踩的坑：**不是所有挂载方式都支持热更新**。

- 以目录形式挂载（`mountPath: /etc/config`）→ 支持热更新
- 使用`subPath`挂载单个文件 → **不支持**热更新

而且，配置文件更新了不代表应用就会重新读取。你的应用需要：
- 要么能监听文件变化并重新加载配置
- 要么通过重启Pod来应用新配置

## Secret的工作原理

### Secret和ConfigMap的关键区别

Secret在使用上和ConfigMap几乎一样，但有几点不同：

1. **数据需要Base64编码**：这不是加密，只是一种编码格式
2. **内存存储**：Secret挂载到Pod时，数据存在内存中（tmpfs），不会写入磁盘
3. **可配置加密**：在etcd存储层面可以开启加密

### 创建Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-credentials
type: Opaque
stringData:  # 使用stringData可以直接写明文，系统会自动Base64编码
  username: admin
  password: "S3cr3tP@ss!"
```

### 使用Secret

方式和ConfigMap完全一样，只是把`configMapKeyRef`换成`secretKeyRef`：

```yaml
env:
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: db-credentials
        key: password
```

### 关于Secret安全性的真相

这里必须澄清一个常见的误解：**Secret的Base64编码不是加密**。任何人都可以轻松解码：

```bash
echo "YWRtaW4=" | base64 -d  # 输出: admin
```

Secret的"安全"主要体现在：
- 和ConfigMap分开存放，方便设置不同的访问权限
- 默认不会在`kubectl get`时显示内容
- 可以配置在etcd中加密存储

如果你需要真正的安全，应该考虑：
- 使用外部密钥管理系统（如HashiCorp Vault）
- 配置etcd静态加密
- 使用RBAC严格限制谁能访问Secret

## ConfigMap与Secret对比

| 特性 | ConfigMap | Secret |
|------|-----------|--------|
| 适用场景 | 普通配置 | 敏感数据 |
| 数据格式 | 明文 | Base64编码 |
| 存储位置 | etcd明文 | etcd可加密 |
| Pod中存储 | 普通文件系统 | 内存（tmpfs） |
| 大小限制 | 1MB | 1MB |

## 常见问题

### Q1: 更新了ConfigMap，但Pod中的配置没有变化？

这是最常见的问题。原因通常是：

1. **你用的是环境变量方式**：环境变量在Pod启动时就固定了，不会自动更新。解决方法是重启Pod。

2. **你用了subPath挂载**：subPath挂载的单个文件不支持热更新。解决方法是改为目录挂载，或者重启Pod。

3. **应用没有重新读取配置**：文件虽然更新了，但应用还在用内存中的旧配置。需要应用支持配置热加载。

最简单可靠的方式是触发滚动更新：
```bash
kubectl rollout restart deployment <deployment-name>
```

### Q2: Secret的Base64编码真的安全吗？

**不安全**。Base64只是一种编码方式，不是加密。之所以用Base64，是因为Secret可能存储二进制数据（如证书），Base64可以确保这些数据能被正确处理。

真正让Secret安全的做法是：
- 配置etcd加密
- 使用RBAC限制访问权限
- 更好的选择是使用外部密钥管理系统

### Q3: 如何防止Secret被提交到Git仓库？

几个常用的方法：

1. **在.gitignore中排除**：最基本的做法，但容易遗漏
2. **使用Sealed Secrets**：将Secret加密后再提交，只有集群能解密
3. **使用External Secrets Operator**：只在Git中保存对外部密钥管理系统的引用

### Q4: 1MB的大小限制不够用怎么办？

ConfigMap和Secret都有1MB的大小限制。如果配置文件太大：

1. **拆分**：把大配置拆成多个小的ConfigMap
2. **外部存储**：大文件放到PersistentVolume或对象存储中
3. **配置服务**：使用专门的配置中心（如Apollo、Nacos）

### Q5: 什么时候应该用不可变的ConfigMap/Secret？

从Kubernetes 1.21开始，可以将ConfigMap和Secret设置为不可变：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: immutable-config
data:
  version: "1.0"
immutable: true  # 设置后无法修改
```

适用场景：
- 配置很少变化，想防止意外修改
- 集群规模很大，不可变资源可以提升性能（不需要监听变化）

需要更新时，创建新的ConfigMap并更新Pod引用。

## 小结

ConfigMap和Secret解决的核心问题是：**将配置与应用镜像解耦**。

- 同一个镜像，配合不同的ConfigMap/Secret，就能在不同环境运行
- 修改配置不需要重新构建镜像
- 敏感信息不会暴露在代码或镜像中

记住关键点：
- ConfigMap存普通配置，Secret存敏感数据
- 环境变量方式简单但不支持热更新
- 文件挂载方式支持热更新（注意subPath例外）
- Secret的Base64不是加密，真正的安全需要额外措施

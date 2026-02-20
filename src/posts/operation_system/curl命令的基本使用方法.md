---
date: 2025-07-01
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 操作系统
tag:
  - 操作系统
---

# curl：命令行的 HTTP 客户端

## curl 是什么

**curl**（Client URL）是一个命令行工具，用于通过 URL 传输数据。它就像命令行里的浏览器，可以发送 HTTP 请求、下载文件、测试 API。

### 为什么需要 curl

**开发调试**：
- 快速测试 API 接口
- 查看 HTTP 响应头和状态码
- 调试网络请求问题

**自动化**：
- 在脚本中下载文件
- 定时监控网站可用性
- 批量处理 HTTP 请求

**学习工具**：
- 理解 HTTP 协议的工作方式
- 观察请求和响应的细节
- 实验不同的 HTTP 方法和头部

### curl vs 浏览器

| 特性     | 浏览器               | curl                       |
| -------- | -------------------- | -------------------------- |
| 界面     | 图形化               | 命令行                     |
| 用途     | 浏览网页             | 数据传输、API 测试、自动化 |
| 可编程性 | 低（需要浏览器扩展） | 高（命令行脚本）           |
| 协议支持 | 主要是 HTTP/HTTPS    | 20+ 种协议                 |
| 灵活性   | 受限于浏览器行为     | 完全控制请求细节           |

---

## HTTP 基础概念

### 请求和响应

HTTP 通信是请求-响应模式：

```
客户端 → 发送请求 → 服务器
客户端 ← 返回响应 ← 服务器
```

**请求包含**：
- **方法**：GET、POST、PUT、DELETE等
- **URL**：请求的资源地址
- **请求头**：元数据（如Content-Type、Authorization）
- **请求体**：发送的数据（POST/PUT等）

**响应包含**：
- **状态码**：200成功、404未找到、500服务器错误
- **响应头**：元数据（如Content-Length、Set-Cookie）
- **响应体**：返回的数据（HTML、JSON等）

### HTTP 方法

**GET**：获取资源
- 用途：读取数据，不应该修改服务器状态
- 特点：参数在URL中，可缓存

**POST**：创建资源
- 用途：提交数据，创建新资源
- 特点：数据在请求体中，不缓存

**PUT**：更新资源
- 用途：完整替换资源
- 特点：幂等（多次调用结果相同）

**PATCH**：部分更新
- 用途：修改资源的部分字段
- 特点：只更新指定字段

**DELETE**：删除资源
- 用途：删除指定资源
- 特点：幂等

---

## curl 的核心功能

### 发送简单请求

最基本的用法：获取网页内容
```bash
curl https://www.example.com
```

这会发送一个 GET 请求，并将响应打印到终端。

### 查看详细信息

**查看响应头**：
```bash
curl -i https://www.example.com
```
显示响应头和响应体，用于查看状态码、Content-Type等信息。

**只看响应头**：
```bash
curl -I https://www.example.com
```
发送 HEAD 请求，只返回响应头，不返回响应体。

**显示请求过程**：
```bash
curl -v https://www.example.com
```
显示详细的请求和响应过程，包括 DNS 解析、TCP 连接、TLS 握手等。

### 发送 POST 请求

**提交表单数据**：
```bash
curl -X POST -d "name=John&age=30" https://api.example.com/users
```

**提交 JSON 数据**：
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"John","age":30}' \
  https://api.example.com/users
```

**关键点**：
- `-X POST` 指定方法
- `-H` 设置请求头
- `-d` 提供请求体数据

### 设置请求头

请求头用于传递元数据：

```bash
curl -H "Authorization: Bearer token123" \
     -H "Content-Type: application/json" \
     https://api.example.com/users
```

**常见请求头**：
- **Content-Type**：请求体的数据格式
- **Accept**：期望的响应格式
- **Authorization**：身份认证信息
- **User-Agent**：客户端标识

### 身份认证

**基本认证**（HTTP Basic Auth）：
```bash
curl -u username:password https://api.example.com/users
```

**Bearer Token**（常用于 API）：
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  https://api.example.com/users
```

**API Key**：
```bash
curl -H "X-API-Key: YOUR_KEY" https://api.example.com/users
```

### 下载文件

**保存为原文件名**：
```bash
curl -O https://example.com/file.zip
```

**指定文件名**：
```bash
curl -o myfile.zip https://example.com/file.zip
```

**断点续传**：
```bash
curl -C - -O https://example.com/large-file.iso
```

---

## 常见使用场景

### API 测试

**测试 REST API**：
```bash
# 获取用户列表
curl https://api.example.com/users

# 创建新用户
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice"}' \
  https://api.example.com/users

# 更新用户
curl -X PUT \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Smith"}' \
  https://api.example.com/users/1

# 删除用户
curl -X DELETE https://api.example.com/users/1
```

### 调试网络问题

**查看完整请求过程**：
```bash
curl -v https://www.example.com
```

这会显示：
- DNS 解析
- TCP 连接
- TLS 握手（HTTPS）
- 请求头
- 响应头

**测试重定向**：
```bash
curl -L https://www.example.com
```
`-L` 选项会自动跟随重定向，适合测试 301/302 跳转。

**测量响应时间**：
```bash
curl -w "\nTime: %{time_total}s\n" -o /dev/null -s https://www.example.com
```

### 处理 Cookie

**保存 Cookie**：
```bash
curl -c cookies.txt https://www.example.com/login
```

**使用 Cookie**：
```bash
curl -b cookies.txt https://www.example.com/profile
```

**Cookie 的作用**：
- 维持会话状态
- 存储登录信息
- 跨请求传递数据

### 文件上传

**上传文件**：
```bash
curl -F "file=@/path/to/file.pdf" https://api.example.com/upload
```

**上传多个文件**：
```bash
curl -F "file1=@file1.jpg" \
     -F "file2=@file2.jpg" \
     https://api.example.com/upload
```

---

## 理解 curl 的输出

### 状态码

**2xx 成功**：
- **200 OK**：请求成功
- **201 Created**：资源创建成功
- **204 No Content**：成功但无响应体

**3xx 重定向**：
- **301 Moved Permanently**：永久重定向
- **302 Found**：临时重定向
- **304 Not Modified**：资源未修改（缓存有效）

**4xx 客户端错误**：
- **400 Bad Request**：请求格式错误
- **401 Unauthorized**：未认证
- **403 Forbidden**：无权限
- **404 Not Found**：资源不存在

**5xx 服务器错误**：
- **500 Internal Server Error**：服务器内部错误
- **502 Bad Gateway**：网关错误
- **503 Service Unavailable**：服务不可用

### 响应头信息

**Content-Type**：响应内容的格式
- `application/json`：JSON 数据
- `text/html`：HTML 页面
- `application/pdf`：PDF 文件

**Content-Length**：响应体的大小（字节）

**Set-Cookie**：服务器设置的 Cookie

**Cache-Control**：缓存策略

---

## 实用技巧

### 静默模式

**隐藏进度条**：
```bash
curl -s https://api.example.com/users
```
适合在脚本中使用，只输出响应内容。

**只显示错误**：
```bash
curl -sS https://api.example.com/users
```
静默模式但显示错误信息。

### 超时设置

**连接超时**：
```bash
curl --connect-timeout 10 https://www.example.com
```
如果 10 秒内无法建立连接，则放弃。

**总超时**：
```bash
curl --max-time 30 https://www.example.com
```
整个请求（包括下载）最多 30 秒。

### 使用代理

**HTTP 代理**：
```bash
curl -x http://proxy.example.com:8080 https://www.example.com
```

**SOCKS5 代理**：
```bash
curl --socks5 127.0.0.1:1080 https://www.example.com
```

### 忽略 SSL 证书验证

**不验证证书**（仅测试环境）：
```bash
curl -k https://self-signed.example.com
```

⚠️ **警告**：生产环境不要使用，会导致安全风险。

---

## 脚本中使用 curl

### 检查 HTTP 状态码

```bash
status=$(curl -s -o /dev/null -w "%{http_code}" https://www.example.com)
if [ $status -eq 200 ]; then
  echo "网站正常"
else
  echo "网站异常：$status"
fi
```

### 提取 JSON 字段

配合 jq 工具：
```bash
curl -s https://api.example.com/users | jq '.[] | .name'
```

### 循环请求

```bash
for i in {1..10}; do
  curl -s https://api.example.com/users/$i
  sleep 1
done
```

---

## 常见问题和解决方案

### 中文乱码

**问题**：响应中的中文显示为乱码
**原因**：终端编码与响应编码不匹配
**解决**：确保终端使用 UTF-8 编码

### 请求被拒绝

**问题**：403 Forbidden 或 401 Unauthorized
**原因**：
- 缺少认证信息
- User-Agent 被服务器拒绝
- IP 被封禁

**解决**：
- 添加认证头（Authorization）
- 设置正常的 User-Agent
- 使用代理

### SSL 证书错误

**问题**：SSL certificate problem
**原因**：
- 自签名证书
- 证书过期
- 证书不匹配

**解决**：
- 更新 CA 证书
- 使用 `-k` 忽略验证（仅测试）
- 检查系统时间是否正确

---

## 核心要点

**curl 的本质**：命令行的 HTTP 客户端，用于数据传输和 API 测试。

**核心概念**：
- HTTP 请求包含：方法、URL、请求头、请求体
- HTTP 响应包含：状态码、响应头、响应体
- 常用方法：GET（读取）、POST（创建）、PUT（更新）、DELETE（删除）

**典型场景**：
- **API 测试**：验证接口功能和响应
- **文件下载**：批量下载或自动化下载
- **网络调试**：查看请求细节和响应头
- **自动化脚本**：定时任务、监控、数据采集

**使用技巧**：
- `-v` 查看详细过程，用于调试
- `-i` 显示响应头，查看状态码和元数据
- `-s` 静默模式，适合脚本
- `-L` 跟随重定向
- `-o/-O` 保存响应到文件

**最佳实践**:
- 测试环境可以用 `-k` 忽略 SSL
- 生产环境必须验证证书
- 敏感数据不要直接写在命令行（使用文件或环境变量）
- 配合 jq、grep 等工具处理响应数据

## 参考资源

- [curl 官方文档](https://curl.se/docs/)
- [curl 命令行选项手册](https://curl.se/docs/manpage.html)
- [HTTP/1.1 规范 RFC 7231](https://datatracker.ietf.org/doc/html/rfc7231)

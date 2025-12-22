---
date: 2025-07-01
author: Gaaming Zhang
category:
  - 操作系统
tag:
  - 操作系统
  - 已完工
---

# curl命令的基本使用方法

## 核心概念

**curl**（Client URL）是一个强大的命令行工具，用于通过 URL 传输数据。它支持多种协议（HTTP、HTTPS、FTP、SMTP 等），是开发人员、运维人员进行 API 测试、调试网络请求、下载文件的必备工具。

**主要特点**：
- 支持 20+ 种协议（HTTP/HTTPS/FTP/FTPS/SCP/SFTP/SMTP/POP3 等）
- 支持 HTTPS、SSL/TLS 证书验证
- 支持代理、Cookie、认证等高级功能
- 跨平台（Linux/macOS/Windows）
- 可编写脚本自动化

---

## 基本语法

```bash
curl [options] [URL]
```

**最简单的用法**：
```bash
# 获取网页内容（GET 请求）
curl https://www.example.com

# 下载文件
curl -O https://example.com/file.zip

# 发送 POST 请求
curl -X POST https://api.example.com/users
```

---

## 常用选项详解

**1. HTTP 方法（-X, --request）**

```bash
# GET 请求（默认）
curl https://api.example.com/users
curl -X GET https://api.example.com/users

# POST 请求
curl -X POST https://api.example.com/users

# PUT 请求
curl -X PUT https://api.example.com/users/1

# DELETE 请求
curl -X DELETE https://api.example.com/users/1

# PATCH 请求
curl -X PATCH https://api.example.com/users/1

# HEAD 请求（只获取响应头）
curl -X HEAD https://api.example.com/users

# OPTIONS 请求（查看支持的方法）
curl -X OPTIONS https://api.example.com/users
```

**2. 发送数据（-d, --data）**

```bash
# 发送表单数据（默认 Content-Type: application/x-www-form-urlencoded）
curl -X POST -d "name=John&age=30" https://api.example.com/users

# 发送 JSON 数据
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"John","age":30}' \
  https://api.example.com/users

# 从文件读取数据
curl -X POST -d @data.json https://api.example.com/users

# 发送多个数据字段
curl -X POST \
  -d "name=John" \
  -d "age=30" \
  -d "city=Beijing" \
  https://api.example.com/users

# URL 编码数据（--data-urlencode）
curl -X POST --data-urlencode "message=Hello World!" \
  https://api.example.com/messages
```

**3. 设置请求头（-H, --header）**

```bash
# 设置单个请求头
curl -H "Content-Type: application/json" https://api.example.com/users

# 设置多个请求头
curl -H "Content-Type: application/json" \
     -H "Authorization: Bearer token123" \
     -H "User-Agent: MyApp/1.0" \
     https://api.example.com/users

# 设置 Accept 头
curl -H "Accept: application/json" https://api.example.com/users

# 自定义 User-Agent（简写）
curl -A "Mozilla/5.0" https://api.example.com/users

# 设置 Referer
curl -H "Referer: https://example.com" https://api.example.com/users
# 或简写
curl -e "https://example.com" https://api.example.com/users
```

**4. 身份认证**

```bash
# HTTP Basic Authentication
curl -u username:password https://api.example.com/users
# 或
curl --user username:password https://api.example.com/users

# 只提供用户名（会提示输入密码）
curl -u username https://api.example.com/users

# Bearer Token 认证
curl -H "Authorization: Bearer YOUR_TOKEN" https://api.example.com/users

# API Key 认证
curl -H "X-API-Key: YOUR_API_KEY" https://api.example.com/users

# Digest 认证
curl --digest -u username:password https://api.example.com/users
```

**5. 下载文件**

```bash
# 下载文件并保存为原文件名（-O）
curl -O https://example.com/file.zip

# 下载文件并指定保存名称（-o）
curl -o myfile.zip https://example.com/file.zip

# 下载多个文件
curl -O https://example.com/file1.zip -O https://example.com/file2.zip

# 断点续传（-C -）
curl -C - -O https://example.com/largefile.zip

# 限速下载（--limit-rate）
curl --limit-rate 100k -O https://example.com/file.zip
# 单位：k (KB/s), m (MB/s)

# 显示下载进度（-#）
curl -# -O https://example.com/file.zip

# 静默模式（-s, --silent）
curl -s -O https://example.com/file.zip

# 显示进度条但隐藏其他信息
curl -# -s -O https://example.com/file.zip
```

**6. Cookie 管理**

```bash
# 发送 Cookie（-b, --cookie）
curl -b "session=abc123; user_id=456" https://api.example.com/profile

# 从文件读取 Cookie
curl -b cookies.txt https://api.example.com/profile

# 保存响应的 Cookie 到文件（-c, --cookie-jar）
curl -c cookies.txt https://api.example.com/login

# 同时读取和保存 Cookie
curl -b cookies.txt -c cookies.txt https://api.example.com/profile

# Cookie 文件格式（Netscape 格式）：
# .example.com  TRUE  /  FALSE  0  session_id  abc123
```

**7. 重定向处理（-L, --location）**

```bash
# 跟随重定向（默认不跟随）
curl -L https://example.com/redirect

# 不跟随重定向（默认行为）
curl https://example.com/redirect

# 限制重定向次数（--max-redirs）
curl -L --max-redirs 5 https://example.com/redirect

# 查看重定向过程（-v）
curl -v -L https://example.com/redirect
```

**8. 显示响应信息**

```bash
# 只显示响应头（-I, --head）
curl -I https://api.example.com/users

# 显示响应头和响应体（-i, --include）
curl -i https://api.example.com/users

# 详细模式（-v, --verbose）显示请求和响应详情
curl -v https://api.example.com/users

# 输出示例：
# > GET /users HTTP/1.1          # > 表示请求
# > Host: api.example.com
# > User-Agent: curl/7.68.0
# < HTTP/1.1 200 OK               # < 表示响应
# < Content-Type: application/json

# 更详细的调试信息（--trace）
curl --trace trace.txt https://api.example.com/users

# 只显示响应体（默认）
curl https://api.example.com/users

# 获取 HTTP 状态码（-w）
curl -o /dev/null -s -w "%{http_code}\n" https://api.example.com/users
# 输出：200

# 获取多个信息
curl -o /dev/null -s -w "HTTP Code: %{http_code}\nTotal Time: %{time_total}s\n" \
  https://api.example.com/users
```

**9. 输出控制**

```bash
# 输出到文件（-o）
curl -o output.html https://example.com

# 输出到标准输出（默认）
curl https://example.com

# 输出到 /dev/null（丢弃输出）
curl -o /dev/null https://example.com

# 静默模式（不显示进度和错误）
curl -s https://example.com

# 只显示错误（-S, --show-error，通常与 -s 配合）
curl -sS https://example.com

# 失败时不输出内容（-f, --fail）
curl -f https://example.com/notfound
# 404 时不输出 HTML 错误页，直接返回错误码
```

**10. 代理设置（-x, --proxy）**

```bash
# HTTP 代理
curl -x http://proxy.example.com:8080 https://api.example.com/users

# SOCKS5 代理
curl -x socks5://proxy.example.com:1080 https://api.example.com/users

# 带认证的代理
curl -x http://proxy.example.com:8080 \
     -U proxyuser:proxypass \
     https://api.example.com/users

# 从环境变量读取代理
export http_proxy=http://proxy.example.com:8080
export https_proxy=http://proxy.example.com:8080
curl https://api.example.com/users

# 不使用代理（--noproxy）
curl --noproxy "*" https://api.example.com/users
```

**11. SSL/TLS 相关**

```bash
# 忽略 SSL 证书验证（不推荐，仅测试用）
curl -k https://self-signed.example.com
# 或
curl --insecure https://self-signed.example.com

# 指定 CA 证书
curl --cacert ca-bundle.crt https://api.example.com/users

# 使用客户端证书
curl --cert client.pem --key client-key.pem https://api.example.com/users

# 指定 SSL/TLS 版本
curl --tlsv1.2 https://api.example.com/users
curl --tlsv1.3 https://api.example.com/users

# 显示 SSL 证书信息
curl -v https://example.com 2>&1 | grep -A 10 "SSL certificate"
```

**12. 超时设置**

```bash
# 连接超时（--connect-timeout）
curl --connect-timeout 10 https://api.example.com/users
# 10 秒内无法建立连接则失败

# 最大执行时间（-m, --max-time）
curl -m 30 https://api.example.com/users
# 整个操作（包括传输）30 秒内完成

# 组合使用
curl --connect-timeout 5 -m 10 https://api.example.com/users
# 5 秒内建立连接，总共 10 秒内完成
```

**13. 上传文件**

```bash
# POST 上传文件（-F, --form，multipart/form-data）
curl -F "file=@/path/to/file.txt" https://api.example.com/upload

# 上传多个文件
curl -F "file1=@file1.txt" -F "file2=@file2.txt" \
  https://api.example.com/upload

# 上传文件并指定文件名
curl -F "file=@localfile.txt;filename=remotefile.txt" \
  https://api.example.com/upload

# 上传文件并添加其他字段
curl -F "file=@file.txt" -F "description=My File" \
  https://api.example.com/upload

# PUT 方式上传文件
curl -X PUT --upload-file file.txt https://api.example.com/files/file.txt
# 或
curl -T file.txt https://api.example.com/files/file.txt

# FTP 上传
curl -T file.txt ftp://ftp.example.com/upload/ -u username:password
```

**14. 性能测试和计时**

```bash
# 显示请求耗时详情（-w, --write-out）
curl -o /dev/null -s -w "\
time_namelookup:  %{time_namelookup}s\n\
time_connect:     %{time_connect}s\n\
time_appconnect:  %{time_appconnect}s\n\
time_pretransfer: %{time_pretransfer}s\n\
time_redirect:    %{time_redirect}s\n\
time_starttransfer: %{time_starttransfer}s\n\
time_total:       %{time_total}s\n" \
https://api.example.com/users

# 输出说明：
# time_namelookup:    DNS 解析时间
# time_connect:       TCP 连接建立时间
# time_appconnect:    SSL/TLS 握手时间
# time_pretransfer:   从开始到准备传输的时间
# time_starttransfer: 从开始到接收第一个字节的时间
# time_total:         总时间

# 简化版（只看总时间）
curl -o /dev/null -s -w "Total: %{time_total}s\n" \
  https://api.example.com/users

# 获取下载速度
curl -o /dev/null -s -w "Speed: %{speed_download} bytes/s\n" \
  https://example.com/file.zip

# 保存计时模板
cat > curl-format.txt << 'EOF'
    time_namelookup:  %{time_namelookup}s
       time_connect:  %{time_connect}s
    time_appconnect:  %{time_appconnect}s
   time_pretransfer:  %{time_pretransfer}s
      time_redirect:  %{time_redirect}s
 time_starttransfer:  %{time_starttransfer}s
                    ----------
         time_total:  %{time_total}s
EOF

curl -w "@curl-format.txt" -o /dev/null -s https://api.example.com/users
```

**15. 其他有用选项**

```bash
# 显示请求和响应的原始数据（--trace-ascii）
curl --trace-ascii debug.txt https://api.example.com/users

# 重试机制（--retry）
curl --retry 3 --retry-delay 5 https://api.example.com/users
# 失败后重试 3 次，每次间隔 5 秒

# 设置 Referer（-e, --referer）
curl -e "https://google.com" https://api.example.com/users

# 压缩（--compressed）
curl --compressed https://api.example.com/users
# 自动请求 gzip 压缩

# 解析 URL 中的 IP（--resolve）
curl --resolve example.com:443:127.0.0.1 https://example.com/
# 强制将 example.com 解析为 127.0.0.1

# 显示 DNS 解析信息
curl -w "Remote IP: %{remote_ip}\n" -o /dev/null -s \
  https://api.example.com/users

# 并发请求（需要配合工具）
# curl 本身不支持并发，但可以结合 xargs 或循环
seq 1 10 | xargs -I{} -P 5 curl -o /dev/null -s https://api.example.com/users
# -P 5: 5 个并发
```

---

## 实用示例

**1. 测试 RESTful API**

```bash
# GET 请求（列出所有用户）
curl -X GET https://jsonplaceholder.typicode.com/users

# GET 单个资源
curl -X GET https://jsonplaceholder.typicode.com/users/1

# POST 创建资源
curl -X POST https://jsonplaceholder.typicode.com/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com"
  }'

# PUT 更新资源（完整更新）
curl -X PUT https://jsonplaceholder.typicode.com/users/1 \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1,
    "name": "John Smith",
    "email": "john.smith@example.com"
  }'

# PATCH 更新资源（部分更新）
curl -X PATCH https://jsonplaceholder.typicode.com/users/1 \
  -H "Content-Type: application/json" \
  -d '{"email": "newemail@example.com"}'

# DELETE 删除资源
curl -X DELETE https://jsonplaceholder.typicode.com/users/1

# 带认证的请求
curl -X GET https://api.example.com/protected \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**2. 模拟表单提交**

```bash
# 普通表单提交（application/x-www-form-urlencoded）
curl -X POST https://example.com/login \
  -d "username=admin" \
  -d "password=secret123"

# 文件上传表单（multipart/form-data）
curl -X POST https://example.com/upload \
  -F "title=My Document" \
  -F "file=@/path/to/document.pdf" \
  -F "category=work"

# 带 Cookie 的表单提交
curl -X POST https://example.com/submit \
  -b "session_id=abc123" \
  -d "comment=Great article!"
```

**3. 下载和保存网页**

```bash
# 下载网页
curl -o homepage.html https://example.com

# 下载并显示（不保存）
curl https://example.com

# 下载并跟随重定向
curl -L -o page.html https://short.url/abc

# 下载并保留服务器文件名
curl -O https://example.com/files/document.pdf

# 批量下载
curl -O https://example.com/image[1-10].jpg
# 下载 image1.jpg 到 image10.jpg

# 下载并显示进度
curl -# -O https://example.com/largefile.zip
```

**4. 测试网站可用性**

```bash
# 检查 HTTP 状态码
curl -I https://example.com
# 或只获取状态码
curl -o /dev/null -s -w "%{http_code}\n" https://example.com

# 测试响应时间
curl -o /dev/null -s -w "Time: %{time_total}s\n" https://example.com

# 检查重定向链
curl -sL -w "%{url_effective}\n" -o /dev/null https://short.url

# 健康检查脚本
#!/bin/bash
URL="https://api.example.com/health"
STATUS=$(curl -o /dev/null -s -w "%{http_code}" $URL)
if [ $STATUS -eq 200 ]; then
  echo "Service is UP"
else
  echo "Service is DOWN (Status: $STATUS)"
fi
```

**5. 调试和排查问题**

```bash
# 详细输出（查看完整请求和响应）
curl -v https://api.example.com/users

# 保存完整的请求响应到文件
curl --trace-ascii trace.log https://api.example.com/users

# 查看 DNS 解析
curl -v https://example.com 2>&1 | grep "Trying"

# 测试不同 HTTP 版本
curl --http1.1 https://example.com
curl --http2 https://example.com

# 查看证书信息
curl -v https://example.com 2>&1 | grep -A 10 "Server certificate"

# 测试 API 返回格式
curl -H "Accept: application/json" https://api.example.com/users
curl -H "Accept: application/xml" https://api.example.com/users
```

**6. 与 Web 服务交互**

```bash
# GraphQL 查询
curl -X POST https://api.example.com/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "{ users { id name email } }"
  }'

# SOAP 请求
curl -X POST https://api.example.com/soap \
  -H "Content-Type: text/xml" \
  -d '<?xml version="1.0"?>
    <soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
      <soap:Body>
        <GetUser>
          <UserId>123</UserId>
        </GetUser>
      </soap:Body>
    </soap:Envelope>'

# WebSocket 连接（需要 websocat 或其他工具）
# curl 不直接支持 WebSocket
```

**7. 使用 Cookie 登录网站**

```bash
# 步骤 1：登录并保存 Cookie
curl -c cookies.txt -X POST https://example.com/login \
  -d "username=admin" \
  -d "password=secret123"

# 步骤 2：使用 Cookie 访问受保护页面
curl -b cookies.txt https://example.com/dashboard

# 步骤 3：登出
curl -b cookies.txt -c cookies.txt https://example.com/logout
```

**8. 并发测试（压力测试）**

```bash
# 使用 GNU Parallel（需安装）
seq 1 100 | parallel -j 10 curl -s -o /dev/null https://api.example.com/users

# 使用 xargs
seq 1 100 | xargs -I{} -P 10 curl -s -o /dev/null https://api.example.com/users

# 简单的 Bash 循环
for i in {1..100}; do
  curl -s -o /dev/null https://api.example.com/users &
done
wait

# 专业工具推荐：ab, wrk, hey
```

---

## 常见错误和解决方法

**1. SSL 证书验证失败**

```bash
# 错误信息：
# curl: (60) SSL certificate problem: self signed certificate

# 临时解决（不推荐用于生产）
curl -k https://self-signed.example.com

# 正确解决：添加证书
curl --cacert /path/to/ca-bundle.crt https://example.com

# 或更新系统证书
# Ubuntu/Debian
sudo update-ca-certificates

# CentOS/RHEL
sudo update-ca-trust
```

**2. 连接超时**

```bash
# 错误信息：
# curl: (28) Connection timed out

# 解决：增加超时时间
curl --connect-timeout 30 https://slow-server.com

# 或检查网络/防火墙
ping slow-server.com
telnet slow-server.com 443
```

**3. 重定向次数过多**

```bash
# 错误信息：
# curl: (47) Maximum (50) redirects followed

# 解决：限制重定向或检查重定向循环
curl -L --max-redirs 10 https://example.com

# 查看重定向链
curl -sL -w "%{url_effective}\n" -o /dev/null https://example.com
```

**4. 返回 401 Unauthorized**

```bash
# 检查认证信息是否正确
curl -v -u username:password https://api.example.com/users

# 检查 Token 是否有效
curl -v -H "Authorization: Bearer TOKEN" https://api.example.com/users

# 查看详细错误信息
curl -i https://api.example.com/users
```

**5. 返回 405 Method Not Allowed**

```bash
# 检查 HTTP 方法是否正确
curl -X OPTIONS https://api.example.com/users
# 查看 Allow 头，了解支持的方法

# 确认使用正确的方法
curl -X POST https://api.example.com/users  # 而不是 GET
```

---

## curl 与其他工具对比

**curl vs wget**

| 特性           | curl               | wget           |
| -------------- | ------------------ | -------------- |
| **主要用途**   | API 测试、数据传输 | 文件下载       |
| **协议支持**   | 20+ 种协议         | HTTP/HTTPS/FTP |
| **递归下载**   | 不支持             | 支持（-r）     |
| **断点续传**   | 支持（-C -）       | 支持（-c）     |
| **上传文件**   | 支持               | 不支持         |
| **自定义请求** | 强大               | 有限           |
| **输出**       | 默认标准输出       | 默认保存文件   |
| **学习曲线**   | 稍陡               | 较平缓         |

```bash
# curl 适合的场景
curl -X POST -H "Content-Type: application/json" -d '{"key":"value"}' \
  https://api.example.com/data

# wget 适合的场景
wget -r -np -k https://example.com/docs/
# 递归下载整个文档目录
```

**curl vs httpie**

```bash
# curl（原始强大）
curl -X POST https://api.example.com/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John"}'

# httpie（更友好的语法）
http POST https://api.example.com/users name=John

# httpie 优势：
# - 更简洁的语法
# - 自动格式化 JSON 输出
# - 彩色输出
# - 默认发送 JSON

# curl 优势：
# - 更广泛的协议支持
# - 更强大的选项
# - 系统预装
# - 更适合脚本
```

---

## 实用脚本示例

**1. API 健康检查脚本**

```bash
#!/bin/bash
# healthcheck.sh

API_URL="https://api.example.com/health"
EXPECTED_STATUS=200
TIMEOUT=10

# 获取状态码
STATUS=$(curl -o /dev/null -s -w "%{http_code}" --max-time $TIMEOUT $API_URL)

# 获取响应时间
RESPONSE_TIME=$(curl -o /dev/null -s -w "%{time_total}" --max-time $TIMEOUT $API_URL)

if [ "$STATUS" -eq "$EXPECTED_STATUS" ]; then
    echo "✓ API is healthy (Status: $STATUS, Time: ${RESPONSE_TIME}s)"
    exit 0
else
    echo "✗ API is unhealthy (Status: $STATUS)"
    exit 1
fi
```

**2. 批量 API 测试脚本**

```bash
#!/bin/bash
# api_test.sh

BASE_URL="https://api.example.com"
TOKEN="your_token_here"

# 测试用例
declare -a tests=(
    "GET:/users:200"
    "GET:/users/1:200"
    "POST:/users:201"
    "DELETE:/users/999:404"
)

for test in "${tests[@]}"; do
    IFS=':' read -r method endpoint expected <<< "$test"
    
    echo "Testing: $method $endpoint"
    
    status=$(curl -X $method -o /dev/null -s -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "$BASE_URL$endpoint")
    
    if [ "$status" -eq "$expected" ]; then
        echo "  ✓ PASS (Expected: $expected, Got: $status)"
    else
        echo "  ✗ FAIL (Expected: $expected, Got: $status)"
    fi
    echo ""
done
```

**3. 下载进度监控脚本**

```bash
#!/bin/bash
# download_with_progress.sh

URL="$1"
OUTPUT="$2"

if [ -z "$URL" ] || [ -z "$OUTPUT" ]; then
    echo "Usage: $0 <URL> <output_file>"
    exit 1
fi

curl -o "$OUTPUT" -# -L --progress-bar \
    -w "\nDownload completed:\n  Size: %{size_download} bytes\n  Speed: %{speed_download} bytes/s\n  Time: %{time_total}s\n" \
    "$URL"
```

**4. API 性能测试脚本**

```bash
#!/bin/bash
# performance_test.sh

URL="https://api.example.com/users"
ITERATIONS=100

echo "Running $ITERATIONS requests to $URL"

total_time=0
success_count=0
failed_count=0

for i in $(seq 1 $ITERATIONS); do
    time=$(curl -o /dev/null -s -w "%{time_total}" --max-time 5 $URL 2>/dev/null)
    status=$?
    
    if [ $status -eq 0 ]; then
        total_time=$(echo "$total_time + $time" | bc)
        success_count=$((success_count + 1))
        echo -n "."
    else
        failed_count=$((failed_count + 1))
        echo -n "x"
    fi
done

echo ""
echo "Results:"
echo "  Success: $success_count"
echo "  Failed: $failed_count"

if [ $success_count -gt 0 ]; then
    avg_time=$(echo "scale=3; $total_time / $success_count" | bc)
    echo "  Average time: ${avg_time}s"
fi
```

**5. 自动重试脚本**

```bash
#!/bin/bash
# retry_curl.sh

URL="$1"
MAX_RETRIES=5
RETRY_DELAY=3

for i in $(seq 1 $MAX_RETRIES); do
    echo "Attempt $i/$MAX_RETRIES..."
    
    response=$(curl -s -w "\n%{http_code}" $URL)
    status_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$status_code" -eq 200 ]; then
        echo "Success!"
        echo "$body"
        exit 0
    else
        echo "Failed with status $status_code"
        if [ $i -lt $MAX_RETRIES ]; then
            echo "Retrying in ${RETRY_DELAY}s..."
            sleep $RETRY_DELAY
        fi
    fi
done

echo "All retries failed"
exit 1
```

---

## 高级技巧

**1. 使用配置文件**

```bash
# 创建 curl 配置文件 ~/.curlrc
cat > ~/.curlrc << 'EOF'
# 默认显示进度条
progress-bar

# 跟随重定向
location

# 自动解压 gzip
compressed

# 设置默认 User-Agent
user-agent = "MyApp/1.0"

# 默认超时
connect-timeout = 10
max-time = 30
EOF

# 使用配置文件
curl https://example.com
# 自动应用 ~/.curlrc 的配置

# 忽略配置文件
curl -q https://example.com
```

**2. 使用变量和模板**

```bash
# 定义变量
BASE_URL="https://api.example.com"
TOKEN="your_token_here"
USER_ID=123

# 使用变量
curl -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/users/$USER_ID"

# 从文件读取 JSON 模板
cat > user_template.json << 'EOF'
{
  "name": "{{NAME}}",
  "email": "{{EMAIL}}"
}
EOF

# 替换并发送
NAME="John Doe" EMAIL="john@example.com" \
  envsubst < user_template.json | \
  curl -X POST -H "Content-Type: application/json" \
    -d @- https://api.example.com/users
```

**3. 管道和链式处理**

```bash
# 获取 API 数据并处理
curl -s https://api.example.com/users | jq '.[] | .name'

# 下载并解压
curl -s https://example.com/file.tar.gz | tar xz

# 获取并统计
curl -s https://api.example.com/data | wc -l

# 多个 API 调用链
USER_ID=$(curl -s https://api.example.com/me | jq -r '.id')
curl -s https://api.example.com/users/$USER_ID/posts | jq '.'
```

**4. 条件请求（缓存）**

```bash
# 使用 If-Modified-Since
curl -H "If-Modified-Since: Wed, 21 Oct 2015 07:28:00 GMT" \
  https://api.example.com/data

# 使用 ETag
ETAG=$(curl -I https://api.example.com/data | grep -i etag | cut -d' ' -f2)
curl -H "If-None-Match: $ETAG" https://api.example.com/data
# 如果未修改，返回 304 Not Modified
```

**5. 速率限制和节流**

```bash
# 每秒一个请求
for i in {1..10}; do
  curl -s https://api.example.com/users/$i | jq '.name'
  sleep 1
done

# 批量请求带延迟
cat user_ids.txt | while read id; do
  curl -s https://api.example.com/users/$id
  sleep 0.5
done
```

---

## 相关高频面试题

#### Q1: curl 如何发送 POST 请求并携带 JSON 数据？

**答案**：

```bash
# 基本方法：使用 -X POST 指定方法，-d 发送数据，-H 设置头
curl -X POST https://api.example.com/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John","age":30,"email":"john@example.com"}'

# 从文件读取 JSON
curl -X POST https://api.example.com/users \
  -H "Content-Type: application/json" \
  -d @user.json

# 使用 Here Document
curl -X POST https://api.example.com/users \
  -H "Content-Type: application/json" \
  -d @- << 'EOF'
{
  "name": "John Doe",
  "age": 30,
  "email": "john@example.com",
  "address": {
    "city": "Beijing",
    "country": "China"
  }
}
EOF

# 关键点：
# 1. -X POST：指定 HTTP 方法
# 2. -H "Content-Type: application/json"：必须设置正确的 Content-Type
# 3. -d '...'：发送数据（单引号避免 shell 解释）
# 4. @filename：从文件读取
# 5. @-：从标准输入读取

# 注意：使用 -d 时，curl 默认使用 POST 方法，所以 -X POST 可以省略
curl https://api.example.com/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John"}'
```

#### Q2: curl 如何处理认证？Bearer Token 和 Basic Auth 分别怎么使用？

**答案**：

**1. Bearer Token 认证（常用于 OAuth 2.0、JWT）**：

```bash
# 方法：在 Authorization 头中添加 Bearer Token
curl https://api.example.com/protected \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 使用变量（推荐）
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
curl https://api.example.com/protected \
  -H "Authorization: Bearer $TOKEN"

# 从文件读取 Token
TOKEN=$(cat token.txt)
curl https://api.example.com/protected \
  -H "Authorization: Bearer $TOKEN"
```

**2. Basic Authentication（HTTP 基本认证）**：

```bash
# 方法 1：使用 -u 选项（推荐）
curl -u username:password https://api.example.com/users

# 方法 2：手动编码 Base64（不推荐）
AUTH=$(echo -n "username:password" | base64)
curl -H "Authorization: Basic $AUTH" https://api.example.com/users

# 只提供用户名（交互式输入密码）
curl -u username https://api.example.com/users
# 会提示输入密码，密码不会显示在命令历史中

# 密码包含特殊字符
curl -u 'username:p@ssw0rd!' https://api.example.com/users
# 使用单引号防止 shell 解释特殊字符
```

**3. API Key 认证**：

```bash
# 在请求头中
curl -H "X-API-Key: your_api_key_here" https://api.example.com/data

# 在 URL 中（不推荐，可能泄露）
curl "https://api.example.com/data?api_key=your_api_key"
```

**4. Digest 认证（更安全的方式）**：

```bash
curl --digest -u username:password https://api.example.com/users
```

**安全建议**：
```bash
# 1. 避免在命令行明文密码（会记录在历史中）
# 不好：
curl -u admin:secret123 https://api.example.com

# 好：使用环境变量
export API_TOKEN="your_token"
curl -H "Authorization: Bearer $API_TOKEN" https://api.example.com

# 2. 从文件读取（文件设置适当权限）
chmod 600 credentials.txt
TOKEN=$(cat credentials.txt)
curl -H "Authorization: Bearer $TOKEN" https://api.example.com

# 3. 使用 .netrc 文件（curl 自动读取）
cat > ~/.netrc << 'EOF'
machine api.example.com
login myusername
password mypassword
EOF
chmod 600 ~/.netrc

curl -n https://api.example.com/users
# -n 选项使用 .netrc 认证
```

#### Q3: 如何用 curl 测量 API 的响应时间和性能？

**答案**：

**方法 1：使用 -w（--write-out）选项**

```bash
# 显示总时间
curl -o /dev/null -s -w "Total time: %{time_total}s\n" \
  https://api.example.com/users

# 详细的时间分解
curl -o /dev/null -s -w "\
DNS解析时间:        %{time_namelookup}s\n\
TCP连接时间:        %{time_connect}s\n\
SSL握手时间:        %{time_appconnect}s\n\
准备传输时间:       %{time_pretransfer}s\n\
开始传输时间:       %{time_starttransfer}s\n\
总时间:            %{time_total}s\n\
下载速度:          %{speed_download} bytes/s\n\
HTTP状态码:        %{http_code}\n" \
https://api.example.com/users

# 时间指标说明：
# time_namelookup:     DNS 解析耗时
# time_connect:        TCP 三次握手耗时
# time_appconnect:     SSL/TLS 握手耗时（HTTPS）
# time_pretransfer:    从请求开始到准备传输文件的时间
# time_starttransfer:  从请求开始到接收第一个字节的时间（TTFB）
# time_total:          整个请求的总时间
```

**方法 2：创建可重用的计时格式文件**

```bash
# 创建格式文件
cat > curl-timing-format.txt << 'EOF'
\n
==== Timing Breakdown ====
DNS Lookup:        %{time_namelookup}s
TCP Connection:    %{time_connect}s
TLS Handshake:     %{time_appconnect}s
Server Processing: %{time_starttransfer}s
Content Transfer:  %{time_total}s
\n
Speed Download:    %{speed_download} bytes/s
Total Size:        %{size_download} bytes
HTTP Code:         %{http_code}
==========================\n
EOF

# 使用格式文件
curl -w "@curl-timing-format.txt" -o /dev/null -s \
  https://api.example.com/users
```

**方法 3：多次测试取平均值**

```bash
#!/bin/bash
# 测试 10 次并计算平均响应时间

URL="https://api.example.com/users"
ITERATIONS=10

total=0
for i in $(seq 1 $ITERATIONS); do
    time=$(curl -o /dev/null -s -w "%{time_total}\n" $URL)
    total=$(echo "$total + $time" | bc)
    echo "Request $i: ${time}s"
done

average=$(echo "scale=3; $total / $ITERATIONS" | bc)
echo "Average time: ${average}s"
```

**方法 4：使用专业工具对比**

```bash
# curl 基准测试
curl -o /dev/null -s -w "%{time_total}\n" https://api.example.com/users

# 使用 Apache Bench (ab)
ab -n 100 -c 10 https://api.example.com/users
# -n 100: 总请求数
# -c 10: 并发数

# 使用 wrk（更现代）
wrk -t4 -c100 -d30s https://api.example.com/users
# -t4: 4 个线程
# -c100: 100 个连接
# -d30s: 持续 30 秒

# 使用 hey
hey -n 100 -c 10 https://api.example.com/users
```

**性能分析示例**：

```bash
# 完整性能分析脚本
#!/bin/bash

URL="https://api.example.com/users"

echo "Performance Analysis for: $URL"
echo "================================"

# 单次请求详细时间
curl -o /dev/null -s -w "\
DNS:       %{time_namelookup}s\n\
Connect:   %{time_connect}s\n\
TLS:       %{time_appconnect}s\n\
TTFB:      %{time_starttransfer}s\n\
Total:     %{time_total}s\n\
Size:      %{size_download} bytes\n\
Speed:     %{speed_download} bytes/s\n" $URL

echo ""
echo "Running 20 requests to calculate average..."

sum=0
min=999999
max=0

for i in {1..20}; do
    time=$(curl -o /dev/null -s -w "%{time_total}" $URL)
    sum=$(echo "$sum + $time" | bc)
    
    # 更新最小值
    if (( $(echo "$time < $min" | bc -l) )); then
        min=$time
    fi
    
    # 更新最大值
    if (( $(echo "$time > $max" | bc -l) )); then
        max=$time
    fi
done

avg=$(echo "scale=3; $sum / 20" | bc)

echo "Min:  ${min}s"
echo "Max:  ${max}s"
echo "Avg:  ${avg}s"
```

#### Q4: curl 如何上传文件？multipart/form-data 和 application/octet-stream 有什么区别？

**答案**：

**1. multipart/form-data（表单文件上传，常用）**：

```bash
# 基本用法：-F 选项
curl -F "file=@/path/to/file.txt" https://api.example.com/upload

# 上传多个文件
curl -F "file1=@file1.txt" -F "file2=@file2.jpg" \
  https://api.example.com/upload

# 上传文件并添加其他表单字段
curl -F "file=@document.pdf" \
     -F "title=My Document" \
     -F "category=work" \
     -F "description=Important file" \
     https://api.example.com/upload

# 指定文件名（服务器看到的文件名）
curl -F "file=@localfile.txt;filename=remotefile.txt" \
  https://api.example.com/upload

# 指定 MIME 类型
curl -F "file=@image.jpg;type=image/jpeg" \
  https://api.example.com/upload

# 完整示例
curl -X POST https://api.example.com/upload \
  -F "file=@report.pdf;filename=monthly_report.pdf;type=application/pdf" \
  -F "user_id=123" \
  -F "timestamp=$(date +%s)" \
  -H "Authorization: Bearer $TOKEN"
```

**2. application/octet-stream（二进制流上传）**：

```bash
# PUT 方式上传
curl -X PUT \
  -H "Content-Type: application/octet-stream" \
  --data-binary "@file.bin" \
  https://api.example.com/files/file.bin

# 或使用 -T（--upload-file）
curl -T file.bin https://api.example.com/files/file.bin

# POST 方式二进制上传
curl -X POST \
  -H "Content-Type: application/octet-stream" \
  --data-binary "@image.jpg" \
  https://api.example.com/upload
```

**两种方式的区别**：

| 特性             | multipart/form-data (-F) | application/octet-stream (--data-binary) |
| ---------------- | ------------------------ | ---------------------------------------- |
| **使用场景**     | HTML 表单文件上传        | 纯二进制文件传输                         |
| **Content-Type** | multipart/form-data      | application/octet-stream                 |
| **可携带元数据** | 是（文件名、字段名等）   | 否（纯数据流）                           |
| **支持多文件**   | 是                       | 否（需要多次请求）                       |
| **额外字段**     | 支持                     | 不支持                                   |
| **数据编码**     | Base64 或原始            | 原始二进制                               |
| **适用 API**     | Web 表单、RESTful API    | 对象存储（S3）、CDN                      |

**实际示例对比**：

```bash
# 场景 1：Web 表单上传（使用 multipart）
curl -F "avatar=@photo.jpg" \
     -F "username=john" \
     -F "email=john@example.com" \
     https://example.com/profile/update

# HTTP 请求头：
# Content-Type: multipart/form-data; boundary=----WebKitFormBoundary...

# 场景 2：AWS S3 上传（使用二进制）
curl -X PUT \
  -H "Content-Type: image/jpeg" \
  -H "x-amz-acl: public-read" \
  --data-binary "@photo.jpg" \
  https://bucket.s3.amazonaws.com/photo.jpg

# 场景 3：大文件分片上传
# 切分文件
split -b 10M largefile.zip chunk_

# 上传分片
for chunk in chunk_*; do
  curl -X POST -F "chunk=@$chunk" \
    https://api.example.com/upload/chunk
done
```

**3. FTP 上传**：

```bash
# FTP 上传文件
curl -T file.txt ftp://ftp.example.com/upload/ \
  -u username:password

# SFTP 上传
curl -T file.txt sftp://sftp.example.com/upload/ \
  -u username:password
```

**注意事项**：

```bash
# 1. 使用 --data-binary 而不是 -d（避免换行符转换）
# 错误：-d 会处理换行符
curl -X POST -d "@binary.dat" https://api.example.com/upload

# 正确：--data-binary 保持原始二进制
curl -X POST --data-binary "@binary.dat" https://api.example.com/upload

# 2. 文件路径前必须加 @
curl -F "file=@/absolute/path/to/file.txt" https://api.example.com/upload

# 3. 查看完整请求（调试）
curl -v -F "file=@test.txt" https://api.example.com/upload 2>&1 | less
```

#### Q5: 如何用 curl 调试 HTTPS 请求？如何查看 SSL 证书信息？

**答案**：

**1. 查看详细请求和响应过程（-v）**：

```bash
# 使用 -v（verbose）显示详细信息
curl -v https://api.example.com/users

# 输出示例：
# * Trying 93.184.216.34:443...
# * Connected to api.example.com (93.184.216.34) port 443
# * ALPN, offering h2
# * ALPN, offering http/1.1
# * successfully set certificate verify locations:
# * TLSv1.3 (OUT), TLS handshake, Client hello (1):
# * TLSv1.3 (IN), TLS handshake, Server hello (2):
# * TLSv1.3 (IN), TLS handshake, Encrypted Extensions (8):
# * TLSv1.3 (IN), TLS handshake, Certificate (11):     ← 证书信息
# * TLSv1.3 (IN), TLS handshake, CERT verify (15):
# * TLSv1.3 (IN), TLS handshake, Finished (20):
# > GET /users HTTP/2                                   ← 请求
# > Host: api.example.com
# > User-Agent: curl/7.68.0
# > Accept: */*
# < HTTP/2 200                                          ← 响应
# < content-type: application/json
# < date: Fri, 20 Dec 2024 10:00:00 GMT
```

**2. 查看 SSL 证书详细信息**：

```bash
# 方法 1：使用 -v 并过滤证书信息
curl -v https://api.example.com 2>&1 | grep -A 10 "Server certificate"

# 输出：
# * Server certificate:
# *  subject: CN=api.example.com
# *  start date: Nov 15 00:00:00 2024 GMT
# *  expire date: Nov 15 23:59:59 2025 GMT
# *  issuer: C=US; O=Let's Encrypt; CN=R3
# *  SSL certificate verify ok.

# 方法 2：使用 openssl（更详细）
echo | openssl s_client -servername api.example.com \
  -connect api.example.com:443 2>/dev/null | \
  openssl x509 -noout -text

# 方法 3：只查看证书有效期
echo | openssl s_client -servername api.example.com \
  -connect api.example.com:443 2>/dev/null | \
  openssl x509 -noout -dates

# 输出：
# notBefore=Nov 15 00:00:00 2024 GMT
# notAfter=Nov 15 23:59:59 2025 GMT
```

**3. 测试不同 TLS 版本**：

```bash
# 强制使用 TLS 1.2
curl --tlsv1.2 https://api.example.com

# 强制使用 TLS 1.3
curl --tlsv1.3 https://api.example.com

# 测试服务器支持的 TLS 版本
for version in tls1 tls1.1 tls1.2 tls1.3; do
  echo -n "Testing $version: "
  curl -s --$version https://api.example.com >/dev/null 2>&1 && \
    echo "Supported" || echo "Not supported"
done
```

**4. 调试 SSL 握手问题**：

```bash
# 详细的 SSL 调试信息
curl -v --trace-ascii ssl_debug.txt https://api.example.com

# 查看支持的加密套件
curl -v https://api.example.com 2>&1 | grep -i cipher

# 指定加密套件
curl --ciphers ECDHE-RSA-AES256-GCM-SHA384 https://api.example.com
```

**5. 处理自签名证书**：

```bash
# 跳过证书验证（不推荐用于生产）
curl -k https://self-signed.example.com
# 或
curl --insecure https://self-signed.example.com

# 使用自定义 CA 证书
curl --cacert /path/to/ca-bundle.crt https://internal.example.com

# 使用客户端证书（双向 TLS）
curl --cert client.pem --key client-key.pem \
  https://api.example.com
```

**6. 调试 SNI（Server Name Indication）问题**：

```bash
# 指定 SNI 主机名
curl --resolve api.example.com:443:1.2.3.4 https://api.example.com

# 测试不同的 SNI
curl -v --connect-to api.example.com:443:actual-server.com:443 \
  https://api.example.com
```

**7. 完整的 HTTPS 调试脚本**：

```bash
#!/bin/bash
# https_debug.sh

URL="$1"

if [ -z "$URL" ]; then
    echo "Usage: $0 <https_url>"
    exit 1
fi

HOST=$(echo $URL | sed -E 's#https?://([^/]+).*#\1#')

echo "=== HTTPS Debug for $URL ==="
echo ""

echo "1. DNS Resolution:"
dig +short $HOST
echo ""

echo "2. TCP Connection:"
timeout 5 bash -c "echo > /dev/tcp/$HOST/443" && \
  echo "  ✓ Port 443 is open" || echo "  ✗ Port 443 is closed"
echo ""

echo "3. SSL Certificate Info:"
echo | openssl s_client -servername $HOST -connect $HOST:443 2>/dev/null | \
  openssl x509 -noout -subject -issuer -dates
echo ""

echo "4. TLS Version Support:"
for v in tls1 tls1_1 tls1_2 tls1_3; do
    result=$(curl -s --${v//_/.} --max-time 3 $URL >/dev/null 2>&1 && echo "✓" || echo "✗")
    echo "  $v: $result"
done
echo ""

echo "5. HTTP Response:"
curl -I -s --max-time 5 $URL | head -1
echo ""

echo "6. Response Time:"
curl -o /dev/null -s -w "  Total: %{time_total}s (SSL: %{time_appconnect}s)\n" $URL
```

**常见 HTTPS 问题排查**：

```bash
# 问题 1：SSL certificate problem: self signed certificate
# 解决：
curl -k https://example.com  # 临时跳过验证
# 或添加证书到系统信任

# 问题 2：SSL certificate problem: unable to get local issuer certificate
# 解决：
curl --cacert ca-bundle.crt https://example.com
# 或更新系统 CA 证书

# 问题 3：SSL: certificate subject name does not match
# 解决：
curl --resolve example.com:443:1.2.3.4 https://example.com

# 问题 4：SSL handshake failed
# 排查：
curl -v https://example.com 2>&1 | grep -i "ssl\|tls\|handshake"

# 问题 5：查看完整握手过程
openssl s_client -connect example.com:443 -showcerts
```

---

## 关键点总结

**curl 核心功能**：
- 支持多种协议的数据传输工具
- API 测试、文件下载/上传、网络调试

**常用选项**：
```bash
-X：HTTP 方法（GET/POST/PUT/DELETE）
-d：发送数据
-H：设置请求头
-u：认证（username:password）
-o：保存到文件
-O：保存为原文件名
-L：跟随重定向
-v：详细输出
-s：静默模式
-i：显示响应头
-w：自定义输出格式
```

**最佳实践**：
- 使用变量存储敏感信息（Token、密码）
- 使用 -v 调试问题
- 使用 -w 测量性能
- 使用配置文件简化命令
- 编写脚本自动化重复任务

**与其他工具对比**：
- curl：API 测试、灵活强大
- wget：文件下载、递归下载
- httpie：更友好的 HTTP 客户端
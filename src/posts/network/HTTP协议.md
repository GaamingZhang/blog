---
date: 2025-07-01
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 网络
tag:
  - 网络
---

# HTTP协议

## 核心概念

**HTTP（HyperText Transfer Protocol）** 是应用层无状态、基于请求-响应的协议，运行在 TCP（或 TLS）之上。它定义了浏览器与服务器如何交换数据，常见版本有 **HTTP/1.1、HTTP/2、HTTP/3**。

**关键特性**：
- 无状态：一次请求完成后不保留会话状态（可用 Cookie/Session/Token 补充）
- 灵活：方法、头、URI 均为文本，易扩展
- 可缓存：通过 Cache-Control/ETag 等头减少重复请求
- 可协商：支持内容协商（语言、编码、媒体类型）
- 可靠传输：基于 TCP 确保数据可靠到达
- 应用层协议：专注于客户端-服务器之间的通信逻辑

**与其他协议的关系**：
```
应用层：HTTP/HTTPS → 传输层：TCP/TLS → 网络层：IP → 链路层：以太网/Wi-Fi
```

**URI 结构**：`scheme://host:port/path?query#fragment`
- scheme：协议（http/https）
- host：域名/IP地址
- port：端口号（默认80/443）
- path：资源路径
- query：查询参数（key=value&key2=value2）
- fragment：页面锚点（仅客户端使用）

---

## 请求与响应报文结构

**请求行**：`METHOD URI HTTP/version`
- 常用方法：GET（读）、POST（提交）、PUT/PATCH（更新）、DELETE（删除）、HEAD（仅头）、OPTIONS（探测）、CONNECT（隧道）

**请求头**：
- 常见：Host、User-Agent、Accept、Accept-Language、Accept-Encoding、Content-Type、Content-Length、Authorization、Cookie
- 缓存相关：Cache-Control、If-None-Match、If-Modified-Since
- 连接相关：Connection、Upgrade、Range

**请求体**：
- 仅 POST/PUT/PATCH 等方法包含
- 格式由 Content-Type 决定（如 application/json、application/x-www-form-urlencoded、multipart/form-data）

**完整请求示例**：
```
GET /api/users?page=1&limit=10 HTTP/1.1
Host: example.com
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36
Accept: application/json
Accept-Language: zh-CN,zh;q=0.9,en;q=0.8
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Cookie: session_id=abc123; user_token=def456
```

**响应行**：`HTTP/version status reason`
- 常见状态码：
	- 1xx：信息（100 Continue）
	- 2xx：成功（200 OK、201 Created、204 No Content）
	- 3xx：重定向（301/302/307/308，304 Not Modified）
	- 4xx：客户端错误（400、401、403、404、429）
	- 5xx：服务器错误（500、502、503、504）

**响应头**：
- 缓存：Cache-Control、ETag、Last-Modified, Expires
- 安全：Set-Cookie、Strict-Transport-Security、Content-Security-Policy、X-Frame-Options
- 传输：Content-Type、Content-Length、Transfer-Encoding、Content-Encoding

**响应体**：
- 包含实际的响应数据
- 格式由 Content-Type 决定

**完整响应示例**：
```
HTTP/1.1 200 OK
Date: Fri, 21 Dec 2025 10:00:00 GMT
Content-Type: application/json
Content-Length: 156
Cache-Control: max-age=3600
ETag: "abcd1234"
Set-Cookie: session_id=abc123; HttpOnly; Secure; SameSite=Lax

{
  "code": 200,
  "message": "success",
  "data": {
    "users": [
      {"id": 1, "name": "张三"},
      {"id": 2, "name": "李四"}
    ],
    "total": 100,
    "page": 1
  }
}
```

---

## 版本演进要点

### HTTP/1.1
**核心特性**：
- 持久连接（Keep-Alive）：默认开启，同一TCP连接可复用处理多个请求
- 管线化（Pipelining）：允许在一个TCP连接上连续发送多个请求，但受限于队头阻塞
- 分块传输编码（chunked）：支持流式传输，无需提前知道内容长度
- 虚拟主机（Host头）：单IP多域名支持
- 范围请求（Range头）：支持断点续传
- 缓存机制完善：ETag、Last-Modified等

### HTTP/2
**核心改进**：
- **二进制分帧**：将报文拆分为二进制帧，提高解析效率
- **多路复用**：单TCP连接上并行传输多个请求-响应，消除应用层队头阻塞
- **头部压缩**：HPACK算法压缩请求/响应头，减少冗余
- **优先级与流量控制**：为不同请求设置优先级，优化资源加载顺序
- **服务端推送**：服务器可主动推送资源到客户端（实践中多禁用）
- **首部字段优化**：静态表、动态表、Huffman编码减少头部开销

**架构对比**：
```
HTTP/1.1: 连接 → 请求1→响应1→请求2→响应2...（串行）
HTTP/2:   连接 → [帧1,帧2,帧3...帧n]（并行）
```

### HTTP/3
**核心特性**：
- **基于 QUIC/UDP**：替代TCP，彻底消除TCP队头阻塞
- **内置 TLS 1.3**：将传输层和加密层握手合并，减少往返次数
- **连接迁移**：基于Connection ID标识连接，支持网络切换（WiFi→4G）时的快速恢复
- **多路复用**：基于QUIC Stream机制，真正的独立并行传输
- **0-RTT 数据传输**：首次连接第二个RTT即可传输数据，后续支持0-RTT
- **可靠性保障**：QUIC内置重传、拥塞控制、流量控制等机制

**握手延迟对比**：
- HTTP/1.1 + TLS 1.2：3-4 RTT
- HTTP/1.1 + TLS 1.3：2 RTT
- HTTP/2 + TLS：同HTTP/1.1 + TLS版本
- HTTP/3：1-2 RTT（首次），0 RTT（后续）

### 版本选择建议
- 优先使用HTTP/2或HTTP/3，特别是对性能要求高的应用
- 保留HTTP/1.1支持以兼容旧客户端
- 全站HTTPS是现代Web应用的基本要求

---

## 缓存与协商

### 缓存分类

**强缓存**：
- 客户端直接使用本地缓存，无需发送请求到服务器
- **控制头**：
  - `Cache-Control: max-age=3600`（相对时间，推荐）
  - `Expires: Thu, 22 Dec 2025 10:00:00 GMT`（绝对时间，受客户端时钟影响）
- **缓存结果**：200 OK (from memory cache/disk cache)
- **常用指令**：
  - `no-cache`：强制协商缓存
  - `no-store`：完全不缓存
  - `public`：允许任何缓存存储
  - `private`：仅客户端缓存

**协商缓存**：
- 客户端发送请求到服务器验证缓存是否有效
- **验证方式**：
  - **ETag / If-None-Match**（推荐，基于内容哈希）：
    - 服务器返回：`ETag: "abcd1234"`
    - 客户端请求：`If-None-Match: "abcd1234"`
  - **Last-Modified / If-Modified-Since**（基于时间，精度秒）：
    - 服务器返回：`Last-Modified: Fri, 21 Dec 2025 10:00:00 GMT`
    - 客户端请求：`If-Modified-Since: Fri, 21 Dec 2025 10:00:00 GMT`
- **缓存结果**：304 Not Modified（节省带宽，不返回响应体）

### 缓存流程
```
客户端请求资源
├── 本地有缓存？
│   ├── 是 → 检查强缓存是否过期？
│   │   ├── 未过期 → 返回 200 (from cache)
│   │   └── 已过期 → 发送协商缓存请求
│   └── 否 → 发送新请求
└── 服务器处理
    ├── 协商缓存命中？
    │   ├── 是 → 返回 304 Not Modified
    │   └── 否 → 返回 200 OK + 新资源
    └── 返回 200 OK + 资源
```

### 内容协商
- **概念**：客户端与服务器协商确定返回的资源版本
- **类型**：
  - 媒体类型协商：`Accept` / `Content-Type`（如 application/json vs text/html）
  - 语言协商：`Accept-Language` / `Content-Language`（如 zh-CN vs en-US）
  - 编码协商：`Accept-Encoding` / `Content-Encoding`（如 gzip vs br）
  - 字符集协商：`Accept-Charset` / `Content-Type` charset

**示例**：
```
请求头：
Accept: application/json, text/plain;q=0.9
Accept-Language: zh-CN,zh;q=0.8,en;q=0.7
Accept-Encoding: gzip, deflate, br

响应头：
Content-Type: application/json; charset=utf-8
Content-Language: zh-CN
Content-Encoding: gzip
```

---

## 长连接与并发

### 连接模型对比

| 版本 | 连接类型 | 并发机制 | 队头阻塞 | 最大并发数 |
|------|---------|---------|---------|-----------|
| HTTP/1.0 | 短连接 | 无 | 无 | 1/请求 |
| HTTP/1.1 | 长连接(Keep-Alive) | 浏览器多连接并发 | 有 | 6-8/域名(浏览器限制) |
| HTTP/2 | 长连接 | 单连接多路复用 | 应用层无，TCP层有 | 理论无限 |
| HTTP/3 | 长连接 | 单连接多路复用 | 完全无 | 理论无限 |

### 长连接机制

**HTTP/1.0**：
- 默认短连接，每个请求-响应后关闭TCP连接
- 可通过 `Connection: Keep-Alive` 手动开启长连接

**HTTP/1.1**：
- 默认开启长连接，无需手动设置
- 可通过 `Connection: close` 关闭长连接
- 长连接超时时间由服务器和客户端共同决定

**HTTP/2/3**：
- 默认长连接，且支持真正的多路复用

### 队头阻塞问题

**HTTP/1.1 队头阻塞**：
- 同一TCP连接上串行处理请求-响应
- 前一个响应未完成时，后续请求被阻塞
- **缓解策略**：
  - 浏览器多连接并发（Chrome默认6个/域名）
  - 域名分片（domain sharding）
  - 资源内联（CSS/JS内联）
  - 减少请求数量

**HTTP/2 队头阻塞**：
- 应用层通过多路复用解决了队头阻塞
- 但TCP层仍存在队头阻塞（TCP按顺序传输数据）

**HTTP/3 队头阻塞**：
- 基于QUIC的Stream机制，每个Stream独立传输
- 单个Stream的丢包不影响其他Stream
- 彻底消除了TCP队头阻塞

### 并发优化策略
- **协议升级**：使用HTTP/2或HTTP/3
- **连接复用**：启用长连接
- **减少请求数**：合并CSS/JS，使用精灵图
- **资源预加载**：`link rel="preload"`、`preconnect`
- **CDN加速**：减少网络延迟
- **HTTP优化**：GZIP压缩、缓存策略等

---

## 常用头实践速查

### 传输与压缩
- **压缩**：
  - 请求：`Accept-Encoding: gzip, br`
  - 响应：`Content-Encoding: gzip` 或 `br`
  - **推荐**：优先使用 Brotli (br) 压缩率更高

- **传输**：
  - 大文件断点续传：`Range: bytes=0-1023` → 响应 `206 Partial Content`
  - 流式传输：`Transfer-Encoding: chunked`（无需 Content-Length）
  - 内容长度：`Content-Length: 1024`（固定长度传输）

### 重定向
- **永久重定向**：`301 Moved Permanently` 或 `308 Permanent Redirect`
- **临时重定向**：`302 Found` 或 `307 Temporary Redirect`
- **重定向地址**：`Location: https://example.com/new-url`
- **注意**：307/308 严格保留原请求方法，301/302 可能将 POST 转为 GET

### 安全头
- **CSP**：`Content-Security-Policy: default-src 'self'; script-src 'self' https://trusted.com`
- **HSTS**：`Strict-Transport-Security: max-age=31536000; includeSubDomains`
- **Cookie 安全**：
  ```
  Set-Cookie: session_id=abc123;
              HttpOnly;    # 防止 XSS 窃取
              Secure;      # 仅 HTTPS 传输
              SameSite=Lax; # 防止 CSRF
              Max-Age=3600 # 有效期
  ```
- **XSS 防护**：`X-XSS-Protection: 1; mode=block`
- **点击劫持防护**：`X-Frame-Options: DENY` 或 `SAMEORIGIN`

### 身份认证
- **Basic Auth**：`Authorization: Basic base64(username:password)`
- **Bearer Token**：`Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`
- **Digest Auth**：比 Basic Auth 更安全的认证方式

### 跨域资源共享 (CORS)
```
Access-Control-Allow-Origin: https://example.com
Access-Control-Allow-Methods: GET, POST, PUT
Access-Control-Allow-Headers: Content-Type, Authorization
Access-Control-Allow-Credentials: true
Access-Control-Max-Age: 86400
```

### 其他常用头
- **内容类型**：`Content-Type: application/json; charset=utf-8`
- **日期**：`Date: Fri, 21 Dec 2025 10:00:00 GMT`
- **服务器信息**：`Server: nginx/1.20.1`
- **自定义头**：`X-Custom-Header: value`（推荐使用 `X-` 前缀）

---

## 排查思路（简版）

### 1. 网络层排查
- **DNS 解析**：
  ```bash
  ping example.com          # 检查网络连通性
  nslookup example.com     # DNS 解析
  dig example.com          # 详细 DNS 信息
  ```
- **IP 直连**：
  ```bash
  curl -v --resolve example.com:443:1.2.3.4 https://example.com
  ```

### 2. 协议层排查
- **握手过程**：
  ```bash
  curl -v https://example.com  # 查看 TCP/TLS 握手过程
  ```
- **HTTP 版本**：
  ```bash
  curl -v --http2 https://example.com  # 强制 HTTP/2
  curl -v --http3 https://example.com  # 强制 HTTP/3
  ```

### 3. 应用层排查
- **状态码分析**：
  - 2xx：成功
  - 3xx：重定向（检查 Location 头）
  - 4xx：客户端错误（400：请求无效，401：认证失败，403：权限不足，404：资源不存在，429：请求过多）
  - 5xx：服务器错误（500：内部错误，502：网关错误，503：服务不可用，504：网关超时）

- **响应头分析**：
  ```bash
  curl -I https://example.com  # 仅查看响应头
  ```

- **缓存检查**：
  ```bash
  curl -I -H "If-None-Match: \"abcd1234\"" https://example.com
  curl -I -H "If-Modified-Since: Fri, 21 Dec 2025 10:00:00 GMT" https://example.com
  ```

### 4. 性能排查
- **延迟分析**：
  ```bash
  curl -w "连接时间: %{time_connect}\n开始传输时间: %{time_starttransfer}\n总时间: %{time_total}\n" https://example.com
  ```

- **带宽测试**：
  ```bash
  curl -o /dev/null -s -w "速度: %{speed_download} bytes/s\n" https://example.com/large-file
  ```

### 5. 跨域排查
- **CORS 检查**：
  ```bash
  curl -v -H "Origin: https://test.com" https://example.com/api
  ```

### 6. 安全排查
- **HTTPS 检查**：
  ```bash
  openssl s_client -connect example.com:443
  ```

### 7. 常见问题快速定位
| 问题 | 可能原因 | 排查方向 |
|------|---------|---------|
| 404 | 路径错误 | 检查 URL 路径、服务器路由配置 |
| 502 | 网关错误 | 检查后端服务是否正常运行 |
| 504 | 网关超时 | 检查后端服务响应时间、负载情况 |
| 429 | 请求过多 | 检查是否触发限流策略 |
| 304 | 缓存命中 | 检查缓存配置是否正确 |

---

## 相关常见问题

**Q1: HTTP/1.1 队头阻塞是什么，如何缓解？**
- **定义**：HTTP/1.1 同一 TCP 连接上串行处理请求-响应，前一个响应未完成时后续请求被阻塞。
- **产生原因**：TCP 按顺序传输数据，HTTP/1.1 复用 TCP 连接但未解决应用层的串行处理问题。
- **缓解策略**：
  - 开启浏览器多连接并发（Chrome 默认 6 个/域名）
  - 升级到 HTTP/2/3（从根本上解决）
  - 资源拆分（小图片、CSS/JS 拆分）
  - 使用 CDN 分散请求域名
  - 减少巨型响应（分页、压缩）
  - 禁用长连接（Connection: close）（极端情况）

**Q2: 301/302/307/308 的区别？**
- **持久化类型**：301/308 是永久重定向；302/307 是临时重定向。
- **方法保留**：
  - 301/302：早期实现可能将 POST 等非 GET 方法改写为 GET（现代浏览器基本已修复）
  - 307/308：严格保留原请求方法和请求体
- **使用场景**：
  - 301：旧域名迁移到新域名
  - 302：临时维护、A/B 测试
  - 307：需要保留原方法的临时重定向（如 POST 表单提交）
  - 308：需要保留原方法的永久重定向

**Q3: 强缓存与协商缓存的区别？**
- **查询方式**：
  - 强缓存：直接读取本地缓存，不发送请求到服务器
  - 协商缓存：需发送请求到服务器，通过条件请求头验证
- **返回状态码**：
  - 强缓存：200 OK (from memory cache/disk cache)
  - 协商缓存：304 Not Modified
- **控制头**：
  - 强缓存：Cache-Control (max-age, no-cache, no-store)、Expires
  - 协商缓存：If-None-Match/ETag、If-Modified-Since/Last-Modified
- **优先级**：Cache-Control > Expires > 协商缓存

**Q4: HTTP/2 相对 HTTP/1.1 的核心改进？**
- **二进制分帧**：将报文分割为二进制帧，提高解析效率
- **多路复用**：单 TCP 连接上并行传输多个请求-响应，消除应用层队头阻塞
- **头部压缩**：使用 HPACK 算法压缩请求/响应头，减少传输体积
- **优先级与流量控制**：支持为不同请求设置优先级，优化资源加载顺序
- **服务端推送**：服务器可主动推送资源到客户端，减少请求延迟
- **首部字段优化**：使用静态表、动态表和 Huffman 编码减少头部开销

**Q5: HTTP/3 为何能减少握手延迟？**
- **基于 QUIC/UDP**：避免 TCP 三次握手的延迟
- **集成 TLS 1.3**：将传输层和加密层握手合并，减少往返次数
- **0-RTT 支持**：首次连接的第二个 RTT 即可传输数据，后续连接支持 0-RTT 数据传输
- **连接 ID**：使用 Connection ID 标识连接，支持网络切换时的快速迁移
- **无 TCP 队头阻塞**：基于 QUIC 的 Stream 机制实现真正的多路复用

**Q6: Cookie 与 Token 的主要差异？**
- **传输方式**：
  - Cookie：由浏览器自动携带在请求头中
  - Token：通常放在 Authorization 头中（如 Bearer Token）
- **存储方式**：
  - Cookie：受浏览器同源策略限制，有大小限制（约 4KB）
  - Token：可存储在 Cookie、localStorage、sessionStorage 中
- **安全性**：
  - Cookie：易受 CSRF 攻击，需设置 SameSite/HttpOnly/Secure 等属性
  - Token：可防止 CSRF 攻击，需注意 XSS 攻击（避免存储在 localStorage）
- **跨域支持**：
  - Cookie：跨域需设置 CORS 和 withCredentials
  - Token：跨域友好，只需在请求头中携带
- **过期机制**：
  - Cookie：可设置 Expires 或 Max-Age
  - Token：通常包含过期时间，需主动刷新或重新获取

**Q7: 1 RTT 是多少次来回传输信息？**
- **定义**：RTT（Round-Trip Time，往返时间）是指**一次完整的来回传输**。
- **具体过程**：
  - 包含一次去程（发送方 → 接收方）和一次回程（接收方 → 发送方）
  - 即：发送方发送数据 → 接收方接收并处理 → 接收方返回响应 → 发送方接收响应
- **时间计算**：RTT = 去程时间 + 接收方处理时间 + 回程时间
- **实际意义**：
  - 反映网络延迟和传输效率
  - 是评估网络性能的重要指标
  - 在协议设计中用于优化（如减少握手 RTT 次数）

**Q8: HTTP 不同版本的握手延迟分别是多少？**
- **HTTP/1.1 + TLS 1.2**：通常需要 3-4 RTT
  - TCP 三次握手：1 RTT
  - TLS 1.2 握手：2 RTT
  - 首次数据传输：第 4 RTT
- **HTTP/1.1 + TLS 1.3**：需要 2 RTT
  - TCP 三次握手：1 RTT
  - TLS 1.3 握手：1 RTT（合并了部分步骤）
  - 首次数据传输：第 2 RTT
- **HTTP/2 + TLS**：与 HTTP/1.1 + TLS 同版本一致（HTTP/2 基于 TCP）
- **HTTP/3**：
  - 首次连接：1-2 RTT（基于 QUIC，合并了 TCP 三次握手和 TLS 握手）
  - 后续连接：0 RTT（使用会话票据复用连接）
**Q9: 如何优化 HTTP 连接的 RTT 延迟？**
- **协议升级**：使用 HTTP/3（基于 QUIC）减少握手 RTT
- **CDN 加速**：将资源部署在离用户更近的节点，减少物理传输距离
- **连接复用**：启用 Keep-Alive 或 HTTP/2 多路复用，避免频繁建立新连接
- **预连接/预加载**：
  - `<link rel="preconnect">`：提前建立 TCP + TLS 连接
  - `<link rel="dns-prefetch">`：提前解析 DNS
- **压缩与缓存**：减少数据传输量和重复请求
- **0-RTT 优化**：配置 TLS 1.3 0-RTT（需注意安全风险）
- **减少重定向**：避免不必要的 3xx 重定向

**Q10: HTTP 与 HTTPS 的主要区别？**
- **安全层**：HTTP 无加密，HTTPS 基于 TLS/SSL 加密传输
- **端口**：HTTP 默认 80，HTTPS 默认 443
- **性能**：HTTPS 因加密/解密增加 CPU 开销，且握手延迟更高
- **证书**：HTTPS 需要 CA 颁发的数字证书
- **SEO**：搜索引擎更偏好 HTTPS 网站
- **安全性**：HTTPS 提供数据完整性、保密性和身份认证

**Q11: 什么是 HTTP 幂等性？有哪些幂等方法？**
- **定义**：多次相同请求产生的副作用与单次请求相同
- **幂等方法**：
  - GET：读取资源，无副作用
  - HEAD：与 GET 类似，仅返回头部
  - PUT：更新资源，多次执行结果一致
  - DELETE：删除资源，多次执行结果一致
  - OPTIONS：探测服务器能力
- **非幂等方法**：
  - POST：创建资源，多次执行会创建多个资源
  - PATCH：部分更新，可能非幂等（取决于实现）

**Q12: HTTP 管线化（Pipelining）是什么？为什么没有广泛应用？**
- **定义**：HTTP/1.1 允许在一个 TCP 连接上连续发送多个请求，无需等待前一个响应
- **优势**：理论上可减少 RTT 延迟
- **未广泛应用的原因**：
  - 队头阻塞问题仍存在（TCP 层）
  - 服务器需支持并发处理请求
  - 部分中间设备可能不兼容
  - 错误处理复杂（某请求失败影响后续）
  - HTTP/2 多路复用提供了更好的解决方案

**Q13: 什么是 CORS？如何解决跨域问题？**
- **定义**：CORS（Cross-Origin Resource Sharing）是浏览器的安全策略，限制跨域请求
- **跨域场景**：
  - 协议不同（http vs https）
  - 域名不同（example.com vs test.com）
  - 端口不同（80 vs 8080）
- **解决方案**：
  - **服务器端设置 CORS 头**：
    ```
    Access-Control-Allow-Origin: https://example.com
    Access-Control-Allow-Methods: GET, POST, PUT
    Access-Control-Allow-Headers: Content-Type, Authorization
    ```
  - **JSONP**：利用 script 标签不受同源策略限制（仅支持 GET）
  - **代理服务器**：Nginx 或 Node.js 代理转发
  - **WebSocket**：不受同源策略限制

**Q14: CSRF 攻击是什么？如何防护？**
- **定义**：CSRF（Cross-Site Request Forgery）跨站请求伪造，攻击者利用用户已登录状态发起恶意请求
- **攻击流程**：
  1. 用户登录正常网站 A
  2. 攻击者诱导用户访问恶意网站 B
  3. B 向 A 发送恶意请求，利用用户的登录状态
- **防护措施**：
  - 设置 SameSite=Strict/Lax 的 Cookie
  - 使用 CSRF Token
  - 验证 Referer/Origin 头
  - 双重 Cookie 验证

**Q15: XSS 攻击与 HTTP 安全头有什么关系？**
- **定义**：XSS（Cross-Site Scripting）跨站脚本攻击，攻击者注入恶意脚本到网页
- **与 HTTP 安全头的关系**：
  - `Content-Security-Policy`：限制脚本来源，禁止内联脚本执行
  - `X-XSS-Protection`：启用浏览器内置 XSS 防护
  - `HttpOnly` Cookie：防止 JS 窃取 Cookie
- **防护措施**：
  ```
  Content-Security-Policy: default-src 'self'; script-src 'self' https://trusted.com
  Set-Cookie: session_id=abc123; HttpOnly
  ```

**Q16: HTTP/2 多路复用的原理是什么？**
- **定义**：HTTP/2 允许在单 TCP 连接上并行传输多个请求-响应
- **实现原理**：
  - **二进制分帧**：将报文拆分为二进制帧
  - **帧类型**：
    - HEADERS：请求/响应头
    - DATA：请求/响应体
    - PRIORITY：优先级设置
    - RST_STREAM：终止流
  - **流机制**：每个请求-响应分配一个流 ID，帧可交错传输
- **优势**：
  - 消除应用层队头阻塞
  - 减少连接建立的开销
  - 降低网络资源消耗

**Q17: QUIC 协议的核心优势有哪些？**
- **基于 UDP**：避免 TCP 三次握手延迟
- **内置 TLS 1.3**：合并传输层和加密层握手，减少 RTT
- **连接迁移**：基于 Connection ID 标识连接，支持网络切换（WiFi→4G）
- **无队头阻塞**：基于 Stream 机制，单个 Stream 丢包不影响其他 Stream
- **0-RTT 支持**：首次连接第二个 RTT 可传输数据，后续支持 0-RTT
- **可靠传输**：内置重传、拥塞控制、流量控制机制

**Q18: 如何优化 HTTP 缓存策略？**
- **静态资源**：
  - 设置长 Cache-Control: max-age=31536000
  - 使用哈希值命名（如 app.v123.js）
  - 禁用 ETag 减少服务器计算
- **动态资源**：
  - 设置 Cache-Control: no-cache 强制协商缓存
  - 使用 ETag 确保内容一致性
  - 合理设置 Last-Modified
- **缓存策略组合**：
  ```
  # 静态资源
  Cache-Control: public, max-age=31536000, immutable
  
  # API 响应
  Cache-Control: private, no-cache
  ETag: "abcd1234"
  ```

**Q19: 什么是 HTTP 内容安全策略（CSP）？如何配置？**
- **定义**：CSP 是一种安全机制，限制网页可以加载的资源来源
- **核心指令**：
  - `default-src`：默认资源策略
  - `script-src`：脚本来源
  - `style-src`：样式来源
  - `img-src`：图片来源
  - `font-src`：字体来源
  - `connect-src`：AJAX 请求来源
- **配置示例**：
  ```
  Content-Security-Policy: 
    default-src 'self';
    script-src 'self' https://analytics.google.com;
    style-src 'self' https://fonts.googleapis.com;
    img-src 'self' https://images.example.com;
    connect-src 'self' https://api.example.com;
  ```

**Q20: HTTP 与 HTTPS 的握手过程有什么区别？**
- **HTTP 握手**：
  1. 客户端发送 TCP 三次握手
  2. 客户端发送 HTTP 请求
  3. 服务器返回 HTTP 响应

- **HTTPS 握手**（TLS 1.3）：
  1. 客户端发送 Client Hello（包含 TLS 版本、加密套件、随机数）
  2. 服务器发送 Server Hello（选择 TLS 版本、加密套件、随机数、证书）
  3. 客户端验证证书，生成预主密钥，用服务器公钥加密发送
  4. 服务器用私钥解密预主密钥，双方计算会话密钥
  5. 客户端发送 Finished 消息（会话密钥加密）
  6. 服务器发送 Finished 消息（会话密钥加密）
  7. TLS 握手完成，开始传输 HTTP 数据

- **主要区别**：
  - HTTPS 多了 TLS 握手过程
  - HTTPS 加密传输数据
  - HTTPS 需要数字证书

**Q21: 什么是 HTTP 幂等性？为什么重要？**
- **定义**：多次相同请求产生的副作用与单次请求相同
- **幂等方法**：
  - GET：读取资源，无副作用
  - HEAD：与 GET 类似，仅返回头部
  - PUT：更新资源，多次执行结果一致
  - DELETE：删除资源，多次执行结果一致
  - OPTIONS：探测服务器能力
- **非幂等方法**：
  - POST：创建资源，多次执行会创建多个资源
  - PATCH：部分更新，可能非幂等（取决于实现）
- **重要性**：
  - 保证系统的可靠性
  - 支持重试机制
  - 简化分布式系统设计

**Q22: HTTP/3 如何解决 TCP 队头阻塞问题？**
- **TCP 队头阻塞**：TCP 按顺序传输数据，若某数据包丢失，后续数据包需等待重传
- **HTTP/3 解决方案**：
  - **基于 QUIC 协议**：使用 UDP 替代 TCP
  - **Stream 机制**：每个请求-响应分配独立 Stream
  - **独立重传**：单个 Stream 丢包仅重传该 Stream 数据
  - **无顺序依赖**：Stream 间数据传输无顺序依赖，互不影响
- **优势**：彻底消除 TCP 队头阻塞，提高传输效率

## 参考资源

- [MDN - HTTP 协议](https://developer.mozilla.org/zh-CN/docs/Web/HTTP)
- [RFC 7230-7235 - HTTP/1.1 规范](https://datatracker.ietf.org/doc/html/rfc7230)
- [HTTP/3 官方规范](https://www.rfc-editor.org/rfc/rfc9114.html)

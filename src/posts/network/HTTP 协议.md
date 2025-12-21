---
date: 2025-12-21
author: Gaaming Zhang
category:
  - 网络
tag:
  - 网络
---

# HTTP 协议

#### 核心概念

**HTTP（HyperText Transfer Protocol）** 是应用层无状态、基于请求-响应的协议，运行在 TCP（或 TLS）之上。它定义了浏览器与服务器如何交换数据，常见版本有 **HTTP/1.1、HTTP/2、HTTP/3**。

**关键特性**：
- 无状态：一次请求完成后不保留会话状态（可用 Cookie/Session/Token 补充）
- 灵活：方法、头、URI 均为文本，易扩展
- 可缓存：通过 Cache-Control/ETag 等头减少重复请求
- 可协商：支持内容协商（语言、编码、媒体类型）

---

#### 请求与响应报文结构

**请求行**：`METHOD URI HTTP/version`
- 常用方法：GET（读）、POST（提交）、PUT/PATCH（更新）、DELETE（删除）、HEAD（仅头）、OPTIONS（探测）、CONNECT（隧道）

**请求头**：
- 常见：Host、User-Agent、Accept、Accept-Language、Accept-Encoding、Content-Type、Content-Length、Authorization、Cookie
- 缓存相关：Cache-Control、If-None-Match、If-Modified-Since
- 连接相关：Connection、Upgrade、Range

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

---

#### 版本演进要点

- **HTTP/1.1**：持久连接（Keep-Alive）、管线化（但受限于队头阻塞）、分块传输编码（chunked）。
- **HTTP/2**：二进制分帧、多路复用、首部压缩（HPACK）、服务端推送（实践中多禁用）。解决了队头阻塞（TCP 层仍有）。
- **HTTP/3**：基于 QUIC（UDP），内置 TLS 1.3，连接迁移，彻底消除 TCP 队头阻塞，握手更快。

---

#### 缓存与协商

强缓存：`Cache-Control: max-age=3600` 或 `Expires`；命中则直接使用缓存。

协商缓存：
- ETag / If-None-Match（推荐，基于内容哈希）
- Last-Modified / If-Modified-Since（基于时间，精度秒）
命中协商缓存返回 304，节省带宽。

---

#### 长连接与并发

- HTTP/1.1 默认 Keep-Alive，同一 TCP 连接可复用多个请求；但队头阻塞导致通常需并发多连接。
- HTTP/2/3 支持单连接多路复用，减少连接数与握手成本。

---

#### 常用头实践速查

- 压缩：`Accept-Encoding: gzip, br`；响应 `Content-Encoding: gzip/br`
- 传输：大文件用 `Range`/`206 Partial Content` 断点续传；流式用 `Transfer-Encoding: chunked`
- 重定向：301/308（永久），302/307（临时保持方法），携带 `Location`
- 安全：`HSTS`（Strict-Transport-Security），`CSP`，`SameSite`/`HttpOnly`/`Secure` Cookie

---

#### 排查思路（简版）

1) 看 DNS/IP：`ping/nslookup/dig` 确认解析；`curl -v` 看握手。
2) 看状态码：4xx 多为客户端/鉴权；5xx 多为服务端。
3) 看缓存：是否被 304/缓存命中；是否缓存过期。
4) 看链路：`curl -v --http2 --resolve host:443:ip` 直连后端绕过域名。
5) 看延迟：`curl -w "%{time_connect} %{time_starttransfer} %{time_total}"`。

---

### 相关高频面试题

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
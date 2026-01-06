---
date: 2025-07-01
author: Gaaming Zhang
isOriginal: false
article: true
star: 900
category:
  - 网络
tag:
  - 网络
---

# DNS工作原理

## 1. 核心概念

**DNS（Domain Name System）** 是互联网的域名系统，将人类可读的域名（如 google.com）转换为机器可识别的 IP 地址（如 142.251.41.14）。DNS 使用**分布式数据库和递归查询**的方式，高效地完成这个转换过程。

**关键特点**：
- **分层结构**：根域名、顶级域、权威域名服务器
- **递归查询**：本地解析器代替用户进行完整查询
- **缓存机制**：减少查询次数，加快解析速度
- **UDP 协议**：大多数查询用 UDP（端口 53），部分用 TCP

---

## 2. DNS 查询流程（完整示例）

假设用户在浏览器输入 **www.google.com**，DNS 解析过程如下：

### 第一步：本地查询（客户端）
```
用户浏览器
  ↓
查看本地缓存（浏览器 DNS 缓存）
  ├─ 如果命中 → 直接返回 IP，流程结束
  └─ 如果未命中 → 继续下一步
  ↓
查询本地 DNS 缓存（操作系统缓存）
  ├─ Windows: ipconfig /displaydns
  ├─ Linux/Mac: 无系统级缓存（由应用管理）
  └─ 如果未命中 → 继续下一步
  ↓
查询 /etc/hosts 文件（本地映射）
  ├─ 如果命中 → 直接返回，流程结束
  └─ 如果未命中 → 继续下一步
```

### 第二步：递归查询（本地 DNS 解析器）

用户配置的本地 DNS 服务器（通常由 ISP 提供）进行递归查询：

```
本地 DNS 解析器（Recursive Resolver）
  例：8.8.8.8（Google DNS），1.1.1.1（CloudFlare）
  ↓
查询根域名服务器（Root Nameserver）
  位置：全球 13 个根服务器（a-m.root-servers.net）
  查询内容："www.google.com 的权威服务器在哪里？"
  回复："问顶级域名服务器（.com 服务器）去"
  ↓
查询顶级域名服务器（TLD Nameserver）
  位置：.com, .org, .net 等的服务器
  查询内容："google.com 的权威服务器在哪里？"
  回复："问 google.com 的权威服务器去"
  ↓
查询权威域名服务器（Authoritative Nameserver）
  位置：google.com 的 DNS 服务器
  查询内容："www.google.com 的 IP 是多少？"
  回复："142.251.41.14"
  ↓
本地 DNS 缓存该结果
  并返回给用户的 DNS 客户端
  ↓
用户操作系统缓存结果
用户浏览器缓存结果
  ↓
浏览器获得 IP 地址，发起 HTTP/HTTPS 连接
```

---

## 3. 反向 DNS 查询流程

反向 DNS（Reverse DNS）是将 IP 地址转换为域名的过程，与正向 DNS（域名到 IP）相反。反向 DNS 主要用于验证 IP 地址的合法性，如邮件服务器验证、网络安全审计等。

### 反向 DNS 查询原理

反向 DNS 查询使用特殊的域名格式 `in-addr.arpa`（IPv4）或 `ip6.arpa`（IPv6），并通过 **PTR 记录**（指针记录）实现 IP 到域名的映射。

**IPv4 反向 DNS 域名格式**：
1. 将 IP 地址的四段数字反转
2. 后缀添加 `.in-addr.arpa`

**示例**：
- IP 地址：`142.251.41.14`
- 反转后：`14.41.251.142`
- 完整反向域名：`14.41.251.142.in-addr.arpa`

**IPv6 反向 DNS 域名格式**：
1. 将 IPv6 地址的每个十六进制字符反转
2. 后缀添加 `.ip6.arpa`

**示例**：
- IPv6 地址：`2001:db8::1`
- 展开后：`2001:0db8:0000:0000:0000:0000:0000:0001`
- 反转后：`1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2`
- 完整反向域名：`1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa`

### 反向 DNS 查询过程

假设我们要查询 IP 地址 `142.251.41.14` 对应的域名，完整的反向 DNS 查询过程如下：

#### 第一步：本地查询（客户端）
```
用户/应用程序
  ↓
查看本地缓存（操作系统/应用缓存）
  ├─ 如果命中 → 直接返回域名，流程结束
  └─ 如果未命中 → 继续下一步
  ↓
查询 /etc/hosts 文件（本地映射）
  ├─ 如果命中 → 直接返回，流程结束
  └─ 如果未命中 → 继续下一步
```

#### 第二步：递归查询（本地 DNS 解析器）

本地 DNS 服务器进行递归查询：

```
本地 DNS 解析器（Recursive Resolver）
  例：8.8.8.8（Google DNS），1.1.1.1（CloudFlare）
  ↓
将 IP 地址转换为反向 DNS 域名
  142.251.41.14 → 14.41.251.142.in-addr.arpa
  ↓
查询根域名服务器（Root Nameserver）
  查询内容："14.41.251.142.in-addr.arpa 的 PTR 记录在哪里？"
  回复："问 .arpa 顶级域名服务器去"
  ↓
查询 .arpa 顶级域名服务器（TLD Nameserver）
  查询内容："14.41.251.142.in-addr.arpa 的 PTR 记录在哪里？"
  回复："问 in-addr.arpa 权威服务器去"
  ↓
查询 in-addr.arpa 权威服务器
  查询内容："14.41.251.142.in-addr.arpa 的 PTR 记录在哪里？"
  回复："问 142.in-addr.arpa 权威服务器去"
  ↓
查询 142.in-addr.arpa 权威服务器
  查询内容："14.41.251.142.in-addr.arpa 的 PTR 记录在哪里？"
  回复："问 251.142.in-addr.arpa 权威服务器去"
  ↓
查询 251.142.in-addr.arpa 权威服务器
  查询内容："14.41.251.142.in-addr.arpa 的 PTR 记录在哪里？"
  回复："14.41.251.142.in-addr.arpa 的 PTR 记录是 lga25s72-in-f14.1e100.net."
  ↓
本地 DNS 缓存该结果
  并返回给用户的 DNS 客户端
  ↓
用户操作系统/应用程序缓存结果
  ↓
应用程序获得域名，完成反向 DNS 解析
```

### 反向 DNS 查询示例

#### 使用 dig 查询反向 DNS
```bash
# 基本反向查询
$ dig -x 142.251.41.14

; <<>> DiG 9.10.6 <<>> -x 142.251.41.14
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 98765
;; flags: qr rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;14.41.251.142.in-addr.arpa.	IN	PTR

;; ANSWER SECTION:
14.41.251.142.in-addr.arpa. 300 IN PTR lga25s72-in-f14.1e100.net.

;; Query time: 15 msec
;; SERVER: 192.168.1.1#53(192.168.1.1)
;; WHEN: Mon Jan 05 10:33:10 CST 2026
;; MSG SIZE  rcvd: 89
```

#### 使用 nslookup 查询反向 DNS
```bash
$ nslookup 142.251.41.14
Server:		192.168.1.1
Address:	192.168.1.1#53

Non-authoritative answer:
14.41.251.142.in-addr.arpa	name = lga25s72-in-f14.1e100.net.

Authoritative answers can be found from:
251.142.in-addr.arpa	nameserver = ns1.google.com.
251.142.in-addr.arpa	nameserver = ns2.google.com.
251.142.in-addr.arpa	nameserver = ns3.google.com.
251.142.in-addr.arpa	nameserver = ns4.google.com.
```

### 反向 DNS 的应用场景

1. **邮件服务器验证**：
   - 防止垃圾邮件：很多邮件服务器会验证发件服务器的反向 DNS 记录
   - SPF、DKIM 等邮件验证机制的补充

2. **网络安全审计**：
   - 日志分析：将 IP 地址转换为域名，便于识别攻击源
   - 入侵检测：通过反向 DNS 验证连接的合法性

3. **服务器管理**：
   - 远程登录：通过域名识别服务器，提高管理效率
   - 负载均衡：某些负载均衡策略会使用反向 DNS 信息

4. **网络诊断**：
   - 故障排查：通过反向 DNS 快速定位网络问题
   - 性能分析：识别网络流量来源的域名

### 反向 DNS 配置示例

要为 IP 地址配置反向 DNS 记录，需要在对应的反向 DNS 区域文件中添加 PTR 记录：

#### IPv4 反向 DNS 配置
```bash
# 假设我们有 IP 地址 192.0.2.100
# 对应的反向域名为 100.2.0.192.in-addr.arpa

# 在 2.0.192.in-addr.arpa 区域文件中添加：
100 IN PTR www.example.com.
```

#### IPv6 反向 DNS 配置
```bash
# 假设我们有 IPv6 地址 2001:db8::100
# 对应的反向域名为 0.0.0.1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa

# 在 8.b.d.0.1.0.0.2.ip6.arpa 区域文件中添加：
0.0.0.1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0 IN PTR ipv6.example.com.
```

---

## 4. DNS 记录类型

**常见的 DNS 记录类型**：

| 记录类型  | 作用                  | 示例                            |
| --------- | --------------------- | ------------------------------- |
| **A**     | 域名指向 IPv4 地址    | google.com → 142.251.41.14      |
| **AAAA**  | 域名指向 IPv6 地址    | google.com → 2607:f8b0:4004:... |
| **CNAME** | 别名，指向另一个域名  | www.google.com → google.com     |
| **MX**    | 邮件交换记录          | gmail.com 的邮件服务器          |
| **NS**    | 名称服务器            | google.com 的权威 DNS 服务器    |
| **SOA**   | 授权开始记录          | 域的 DNS 主服务器等信息         |
| **TXT**   | 文本记录              | SPF、DKIM 验证，Google 验证等   |
| **PTR**   | 反向解析，IP 指向域名 | 142.251.41.14 → google.com      |

**查询 DNS 记录**：
```bash
# 查询 A 记录（默认）
nslookup google.com
dig google.com

# 查询特定类型
dig google.com MX          # 邮件服务器
dig google.com NS          # 权威 DNS 服务器
dig google.com TXT         # 文本记录

# 查询 CNAME
dig www.google.com CNAME

# 反向查询（IP 到域名）
dig -x 142.251.41.14       # 反向 DNS
nslookup 142.251.41.14
```

---

## 4.1 DNS 协议报文结构

DNS 报文采用二进制格式，分为查询报文和响应报文，结构如下：

```
+---------------------+
|      Header（12字节）     |
+---------------------+
|   Question（可变长度）    |
+---------------------+
|   Answer（可变长度）      |
+---------------------+
|  Authority（可变长度）     |
+---------------------+
| Additional（可变长度）    |
+---------------------+
```

### Header 字段详解
```
0  1  2  3  4  5  6  7  8  9  10 11 12 13 14 15
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                      ID                       |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|QR|   Opcode  |AA|TC|RD|RA|   Z    |   RCODE   |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    QDCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    ANCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    NSCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    ARCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
```

**关键字段说明**：
- **ID（16位）**：标识查询和响应的对应关系
- **QR（1位）**：0 表示查询，1 表示响应
- **Opcode（4位）**：操作码（0=标准查询，1=反向查询，2=服务器状态请求）
- **AA（1位）**：授权回答（仅响应有效）
- **TC（1位）**：截断标志（响应超过 UDP 大小时设置）
- **RD（1位）**：期望递归（查询时设置）
- **RA（1位）**：可用递归（服务器支持递归时设置）
- **RCODE（4位）**：响应码（0=无错误，3=域名不存在等）
- **QDCOUNT**：问题记录数
- **ANCOUNT**：回答记录数
- **NSCOUNT**：授权记录数
- **ARCOUNT**：附加记录数

### Question 字段结构
```
+---------------------+
|   QNAME（可变长度）   | 域名（标签格式）
+---------------------+
|   QTYPE（2字节）     | 查询类型（A=1, AAAA=28, MX=15等）
+---------------------+
|   QCLASS（2字节）    | 查询类（IN=1 表示互联网）
+---------------------+
```

### Resource Record（资源记录）结构
```
+---------------------+
|   NAME（可变长度）    | 域名
+---------------------+
|   TYPE（2字节）      | 记录类型
+---------------------+
|   CLASS（2字节）     | 记录类（IN=1）
+---------------------+
|   TTL（4字节）       | 生存时间（秒）
+---------------------+
|   RDLENGTH（2字节）  | 数据长度
+---------------------+
|   RDATA（可变长度）   | 记录数据
+---------------------+
```

**查看 DNS 报文**：
```bash
# 使用 dig +short +noall +answer 查看简洁输出
dig google.com +short +noall +answer

# 使用 tcpdump 抓取 DNS 报文
sudo tcpdump -i any -n port 53

# 使用 wireshark 分析 DNS 报文
# 过滤器：dns.qry.name == "google.com"
```

---

## 5. DNS 解析模式

### 1. 递归查询（Recursive Query）
```
客户端
  ↓ 全权委托给本地 DNS 服务器
本地 DNS 服务器
  ↓ 代替客户端，逐级查询
根 → TLD → 权威
  ↓ 返回完整答案给客户端
客户端获得最终 IP
```

**特点**：客户端只发送一次请求，由本地 DNS 服务器完成所有查询工作。

### 2. 迭代查询（Iterative Query）
```
本地 DNS → 根服务器："google.com 在哪里？"
根服务器："去问 TLD 服务器"（返回 TLD 地址）
  ↓
本地 DNS → TLD 服务器："google.com 在哪里？"
TLD 服务器："去问权威服务器"（返回权威服务器地址）
  ↓
本地 DNS → 权威服务器："www.google.com 的 IP？"
权威服务器："142.251.41.14"
  ↓
本地 DNS 返回给客户端
```

**特点**：每一步返回一个指针，由 DNS 服务器主动逐级查询。

---

## 5.1 DNSSEC（DNS Security Extensions）

DNSSEC 是 DNS 的安全扩展，通过数字签名验证 DNS 响应的真实性和完整性，防止 DNS 欺骗和缓存投毒攻击。

### DNSSEC 工作原理
```
根域名服务器（Root Zone）
  ↓ 签名（KSK + ZSK）
  ↓
顶级域名服务器（TLD Zone）
  ↓ 签名（KSK + ZSK）
  ↓
权威域名服务器（Authoritative Zone）
  ↓ 签名（KSK + ZSK）
  ↓
DNSSEC 验证链
```

### DNSSEC 关键记录类型
| 记录类型 | 作用 |
|---------|------|
| **DNSKEY** | 存储公钥（KSK 和 ZSK） |
| **RRSIG** | 资源记录的数字签名 |
| **DS** | 委派签名，连接父域和子域 |
| **NSEC** | 否定存在证明（证明域名不存在） |
| **NSEC3** | NSEC 的改进版本，防止域名枚举 |

### DNSSEC 验证过程
```
1. 客户端查询 example.com 的 A 记录
2. 服务器返回 A 记录 + RRSIG（签名）
3. 客户端获取 example.com 的 DNSKEY（公钥）
4. 使用 DNSKEY 验证 RRSIG 签名
5. 验证 DNSKEY 的真实性（通过父域的 DS 记录）
6. 递归验证到根域的信任锚点
```

### 查询 DNSSEC 记录
```bash
# 查询 DNSKEY 记录
dig example.com DNSKEY +dnssec

# 查询 DS 记录
dig example.com DS +dnssec

# 验证 DNSSEC
dig example.com +dnssec

# 查看验证状态
dig example.com +dnssec +multi
```

### DNSSEC 配置
```bash
# 生成密钥对
dnssec-keygen -a RSASHA256 -b 2048 -n ZONE example.com

# 签名区域文件
dnssec-signzone -A -3 $(head -c 1000 /dev/urandom | sha1sum | cut -b 1-16) -N INCREMENT -o example.com -t db.example.com

# 在 named.conf 中启用 DNSSEC
options {
    dnssec-validation auto;
};
```

---

## 5.2 DoH（DNS over HTTPS）和 DoT（DNS over TLS）

DoH 和 DoT 是加密 DNS 查询的协议，防止 DNS 查询被窃听或篡改。

### DoH（DNS over HTTPS）
**特点**：
- 使用 HTTPS 协议（端口 443）
- 查询伪装成 HTTPS 流量
- 难以被防火墙识别和拦截
- 性能开销较大（TLS + HTTP）

**DoH 服务器**：
- Google: https://dns.google/dns-query
- CloudFlare: https://1.1.1.1/dns-query
- 阿里云: https://dns.alidns.com/dns-query

**使用 DoH**：
```bash
# 使用 curl 查询 DoH
curl -H "accept: application/dns-json" "https://dns.google/resolve?name=example.com&type=A"

# 使用 dig 查询 DoH（需要支持 DoH 的 dig 版本）
dig @https://dns.google/dns-query example.com

# 浏览器配置 DoH
# Chrome: chrome://settings/security → 使用安全 DNS
# Firefox: 设置 → 常规 → 网络设置 → 启用基于 HTTPS 的 DNS
```

### DoT（DNS over TLS）
**特点**：
- 使用 TLS 协议（端口 853）
- 专门为 DNS 设计
- 性能开销较小（仅 TLS）
- 容易被防火墙识别和拦截

**DoT 服务器**：
- Google: dns.google:853
- CloudFlare: 1.1.1.1:853
- Quad9: 9.9.9.9:853

**使用 DoT**：
```bash
# 使用 kdig（Knot DNS）查询 DoT
kdig @dns.google +tls +port=853 example.com

# 使用 systemd-resolved 配置 DoT
sudo vim /etc/systemd/resolved.conf
# 添加：
# DNS=1.1.1.1
# DNSOverTLS=yes

# 使用 stubby 配置 DoT
sudo apt install stubby
sudo vim /etc/stubby/stubby.yml
# 配置 upstream_recursive_servers
```

### DoH vs DoT 对比
| 特性 | DoH | DoT |
|-----|-----|-----|
| 协议 | HTTPS | TLS |
| 端口 | 443 | 853 |
| 隐蔽性 | 高（伪装 HTTPS） | 低（专用端口） |
| 性能开销 | 较大 | 较小 |
| 部署难度 | 中等 | 简单 |
| 标准化 | RFC 8484 | RFC 7858 |

---

## 6. DNS 缓存和 TTL

### 缓存层级
```
浏览器缓存（几分钟）
  ↓（如果未命中）
操作系统缓存
  ↓（如果未命中）
本地 DNS 服务器缓存（根据 TTL）
  ↓（如果未命中）
递归查询（根 → TLD → 权威）
```

### TTL（Time To Live）
```bash
# TTL 是 DNS 记录的有效期，单位为秒

# 查看 TTL
dig google.com
# 输出示例：
# google.com.     300   IN  A  142.251.41.14
#                 ↑
#              TTL 300 秒

# TTL 值的含义：
# TTL = 300   : DNS 记录在缓存中保留 5 分钟
# TTL = 3600  : DNS 记录在缓存中保留 1 小时（常见）
# TTL = 86400 : DNS 记录在缓存中保留 1 天

# 更改 DNS 记录的生效时间：
# 修改 DNS 前，先降低 TTL（如改为 60 秒）
# 等待原 TTL 过期，再修改 DNS
# 修改完成后，可恢复 TTL 为较大值
```

---

## 6.1 DNS 负载均衡和故障转移

### DNS 负载均衡
DNS 负载均衡通过为同一域名配置多个 IP 地址，将流量分配到不同的服务器。

**实现方式**：
```bash
# 配置多个 A 记录
example.com.  300  IN  A  192.0.2.1
example.com.  300  IN  A  192.0.2.2
example.com.  300  IN  A  192.0.2.3

# DNS 服务器轮询返回 IP 地址
# 查询 1: 192.0.2.1
# 查询 2: 192.0.2.2
# 查询 3: 192.0.2.3
# 查询 4: 192.0.2.1（循环）
```

**优点**：
- 简单易实现
- 无需额外硬件
- 支持地理位置负载均衡（基于 GeoDNS）

**缺点**：
- 无法感知服务器负载
- 缓存导致负载不均衡
- 故障切换慢（受 TTL 限制）

### DNS 故障转移
DNS 故障转移通过健康检查自动切换到备用服务器。

**实现方式**：
```bash
# 主服务器健康时
example.com.  60  IN  A  192.0.2.1  # 主服务器
example.com.  60  IN  A  192.0.2.2  # 备用服务器

# 主服务器故障时
example.com.  60  IN  A  192.0.2.2  # 仅返回备用服务器
```

**健康检查工具**：
- **DNS Made Easy**：提供 DNS 故障转移服务
- **CloudFlare**：自动健康检查和故障转移
- **AWS Route 53**：基于路由的健康检查
- **Google Cloud DNS**：支持健康检查

**配置示例（AWS Route 53）**：
```bash
# 创建健康检查
aws route53 create-health-check \
  --caller-reference "example-com-health-check" \
  --health-check-config \
    IPAddress=192.0.2.1,Port=80,Type=HTTPS,ResourcePath=/health

# 创建故障转移记录
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567890ABC \
  --change-batch file://route53.json
```

**route53.json**：
```json
{
  "Changes": [{
    "Action": "CREATE",
    "ResourceRecordSet": {
      "Name": "example.com",
      "Type": "A",
      "SetIdentifier": "Primary",
      "Failover": "PRIMARY",
      "TTL": 60,
      "ResourceRecords": [{"Value": "192.0.2.1"}],
      "HealthCheckId": "health-check-id"
    }
  }, {
    "Action": "CREATE",
    "ResourceRecordSet": {
      "Name": "example.com",
      "Type": "A",
      "SetIdentifier": "Secondary",
      "Failover": "SECONDARY",
      "TTL": 60,
      "ResourceRecords": [{"Value": "192.0.2.2"}]
    }
  }]
}
```

### GeoDNS（地理 DNS）
GeoDNS 根据客户端的地理位置返回最近的服务器 IP。

**实现方式**：
```bash
# 配置不同地区的 A 记录
example.com.  300  IN  A  192.0.2.1  # 北美
example.com.  300  IN  A  192.0.2.2  # 欧洲
example.com.  300  IN  A  192.0.2.3  # 亚太

# DNS 服务器根据客户端 IP 的地理位置返回对应的服务器
```

**GeoDNS 服务商**：
- **CloudFlare**：自动 GeoDNS
- **AWS Route 53**：基于延迟的路由
- **Google Cloud DNS**：支持地理位置路由
- **NS1**：高级 GeoDNS 功能

---

## 6.2 DNS 监控和告警

### DNS 监控指标
**关键指标**：
- **解析延迟**：DNS 查询响应时间
- **可用性**：DNS 服务器在线率
- **错误率**：SERVFAIL、NXDOMAIN 等错误比例
- **缓存命中率**：本地缓存命中比例
- **查询量**：每秒查询数（QPS）

### 监控工具
**1. DNSPerf**
```bash
# 安装 DNSPerf
sudo apt install dnsperf

# 测试 DNS 性能
dnsperf -s 8.8.8.8 -d queryfile.txt -l 60

# queryfile.txt 示例：
# www.google.com A
# www.facebook.com A
# www.amazon.com A
```

**2. Namebench**
```bash
# 安装 Namebench
sudo apt install namebench

# 运行基准测试
namebench
```

**3. Prometheus + Grafana**
```bash
# 使用 dns_exporter 收集 DNS 指标
docker run -d \
  --name dns_exporter \
  -p 9153:9153 \
  prometheuscommunity/dns-exporter \
  --dns.server=8.8.8.8

# 在 Prometheus 中配置抓取
scrape_configs:
  - job_name: 'dns'
    static_configs:
      - targets: ['localhost:9153']
```

**4. dnspython 监控脚本**
```python
import dns.resolver
import time

def check_dns(domain, dns_server):
    resolver = dns.resolver.Resolver()
    resolver.nameservers = [dns_server]
    
    start_time = time.time()
    try:
        resolver.resolve(domain, 'A')
        latency = (time.time() - start_time) * 1000
        return {'status': 'success', 'latency': latency}
    except Exception as e:
        return {'status': 'error', 'error': str(e)}

# 检查多个 DNS 服务器
for server in ['8.8.8.8', '1.1.1.1', '208.67.222.222']:
    result = check_dns('google.com', server)
    print(f"{server}: {result}")
```

### 告警配置
**Prometheus 告警规则**：
```yaml
groups:
  - name: dns_alerts
    rules:
      - alert: DNSHighLatency
        expr: dns_query_duration_seconds > 1
        for: 5m
        annotations:
          summary: "DNS 查询延迟过高"
      
      - alert: DNSHighErrorRate
        expr: rate(dns_query_errors_total[5m]) > 0.1
        for: 5m
        annotations:
          summary: "DNS 错误率过高"
```

---

## 7. DNS 查询工具使用

### 1. nslookup（基础查询）

#### 基本用法
```bash
# 查询 A 记录
nslookup google.com

# 查询特定 DNS 服务器
nslookup google.com 8.8.8.8

# 查询特定记录类型
nslookup -type=MX google.com

# 交互模式
nslookup
> google.com
> set type=NS
> google.com
```

#### 返回样例和字段解释

**查询 A 记录**：
```bash
$ nslookup google.com
Server:		192.168.1.1
Address:	192.168.1.1#53

Non-authoritative answer:
Name:	google.com
Address: 142.251.41.14
Name:	google.com
Address: 142.251.41.15
```

**字段解释**：
- **Server**: 使用的 DNS 服务器地址
- **Address**: DNS 服务器的 IP 地址和端口（#53 表示 DNS 默认端口）
- **Non-authoritative answer**: 非权威回答（来自缓存，非权威 DNS 服务器）
- **Name**: 查询的域名
- **Address**: 域名对应的 IP 地址（可能返回多个，用于负载均衡）

**查询 MX 记录**：
```bash
$ nslookup -type=MX gmail.com
Server:		192.168.1.1
Address:	192.168.1.1#53

Non-authoritative answer:
gmail.com	mail exchanger = 5 gmail-smtp-in.l.google.com.
gmail.com	mail exchanger = 10 alt1.gmail-smtp-in.l.google.com.
gmail.com	mail exchanger = 20 alt2.gmail-smtp-in.l.google.com.
gmail.com	mail exchanger = 30 alt3.gmail-smtp-in.l.google.com.
gmail.com	mail exchanger = 40 alt4.gmail-smtp-in.l.google.com.

Authoritative answers can be found from:
gmail.com	nameserver = ns1.google.com.
gmail.com	nameserver = ns2.google.com.
gmail.com	nameserver = ns3.google.com.
gmail.com	nameserver = ns4.google.com.
```

**字段解释**：
- **mail exchanger = 5**: 优先级（数字越小优先级越高）
- **gmail-smtp-in.l.google.com**: 邮件服务器域名
- **Authoritative answers can be found from**: 权威 DNS 服务器列表
- **nameserver**: 权威 DNS 服务器地址

**反向查询**：
```bash
$ nslookup 142.251.41.14
Server:		192.168.1.1
Address:	192.168.1.1#53

Non-authoritative answer:
14.41.251.142.in-addr.arpa	name = lga25s72-in-f14.1e100.net.

Authoritative answers can be found from:
251.142.in-addr.arpa	nameserver = ns1.google.com.
251.142.in-addr.arpa	nameserver = ns2.google.com.
251.142.in-addr.arpa	nameserver = ns3.google.com.
251.142.in-addr.arpa	nameserver = ns4.google.com.
```

**字段解释**：
- **14.41.251.142.in-addr.arpa**: 反向 DNS 域名（IP 反转后加上 in-addr.arpa）
- **name**: IP 对应的域名

---

### 2. dig（详细查询）

#### 基本用法
```bash
# 标准查询
dig google.com

# 简洁输出
dig google.com +short

# 查询所有记录
dig google.com ANY

# 追踪完整递归过程
dig google.com +trace

# 指定 DNS 服务器
dig @8.8.8.8 google.com

# 反向查询
dig -x 142.251.41.14
```

#### 返回样例和字段解释

**查询 A 记录**：
```bash
$ dig google.com

; <<>> DiG 9.10.6 <<>> google.com
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 12345
;; flags: qr rd ra; QUERY: 1, ANSWER: 2, AUTHORITY: 0, ADDITIONAL: 1

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;google.com.			IN	A

;; ANSWER SECTION:
google.com.		300	IN	A	142.251.41.14
google.com.		300	IN	A	142.251.41.15

;; Query time: 12 msec
;; SERVER: 192.168.1.1#53(192.168.1.1)
;; WHEN: Mon Jan 05 10:30:45 CST 2026
;; MSG SIZE  rcvd: 74
```

**字段解释**：
- **HEADER 部分**：
  - **opcode**: 操作码（QUERY=标准查询）
  - **status**: 响应状态码（NOERROR=成功，NXDOMAIN=域名不存在，SERVFAIL=服务器错误）
  - **id**: 查询 ID（用于匹配请求和响应）
  - **flags**: 标志位
    - **qr**: 响应标志（query response）
    - **rd**: 期望递归（recursion desired）
    - **ra**: 可用递归（recursion available）
  - **QUERY**: 问题记录数
  - **ANSWER**: 回答记录数
  - **AUTHORITY**: 授权记录数
  - **ADDITIONAL**: 附加记录数

- **OPT PSEUDOSECTION**：
  - **EDNS**: 扩展 DNS 版本
  - **udp**: UDP 数据包大小（4096 字节）

- **QUESTION SECTION**：
  - **google.com.**: 查询的域名（末尾的点表示根域名）
  - **IN**: 查询类（IN=互联网）
  - **A**: 查询类型（A=IPv4 地址）

- **ANSWER SECTION**：
  - **google.com.**: 域名
  - **300**: TTL（生存时间，单位秒）
  - **IN**: 记录类
  - **A**: 记录类型
  - **142.251.41.14**: IP 地址

- **底部信息**：
  - **Query time**: 查询耗时（毫秒）
  - **SERVER**: 使用的 DNS 服务器
  - **WHEN**: 查询时间
  - **MSG SIZE rcvd**: 接收到的消息大小（字节）

**查询 MX 记录**：
```bash
$ dig google.com MX

; <<>> DiG 9.10.6 <<>> google.com MX
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 54321
;; flags: qr rd ra; QUERY: 1, ANSWER: 5, AUTHORITY: 0, ADDITIONAL: 1

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;google.com.			IN	MX

;; ANSWER SECTION:
google.com.		600	IN	MX	10 smtp.google.com.
google.com.		600	IN	MX	20 alt1.aspmx.l.google.com.
google.com.		600	IN	MX	30 alt2.aspmx.l.google.com.
google.com.		600	IN	MX	40 alt3.aspmx.l.google.com.
google.com.		600	IN	MX	50 alt4.aspmx.l.google.com.

;; Query time: 8 msec
;; SERVER: 192.168.1.1#53(192.168.1.1)
;; WHEN: Mon Jan 05 10:31:20 CST 2026
;; MSG SIZE  rcvd: 146
```

**字段解释**：
- **MX**: 邮件交换记录类型
- **10, 20, 30, 40, 50**: 优先级（数字越小优先级越高）
- **smtp.google.com.**: 邮件服务器域名

**查询 TXT 记录**：
```bash
$ dig google.com TXT

; <<>> DiG 9.10.6 <<>> google.com TXT
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 67890
;; flags: qr rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;google.com.			IN	TXT

;; ANSWER SECTION:
google.com.		300	IN	TXT	"v=spf1 include:_spf.google.com ~all"

;; Query time: 10 msec
;; SERVER: 192.168.1.1#53(192.168.1.1)
;; WHEN: Mon Jan 05 10:32:05 CST 2026
;; MSG SIZE  rcvd: 89
```

**字段解释**：
- **TXT**: 文本记录类型
- **"v=spf1 include:_spf.google.com ~all"**: SPF 记录内容（用于邮件验证）

**追踪递归过程**：
```bash
$ dig google.com +trace

; <<>> DiG 9.10.6 <<>> google.com +trace
;; global options: +cmd
.			518400	IN	NS	a.root-servers.net.
.			518400	IN	NS	b.root-servers.net.
.			518400	IN	NS	c.root-servers.net.
.			518400	IN	NS	d.root-servers.net.
.			518400	IN	NS	e.root-servers.net.
.			518400	IN	NS	f.root-servers.net.
.			518400	IN	NS	g.root-servers.net.
.			518400	IN	NS	h.root-servers.net.
.			518400	IN	NS	i.root-servers.net.
.			518400	IN	NS	j.root-servers.net.
.			518400	IN	NS	k.root-servers.net.
.			518400	IN	NS	l.root-servers.net.
.			518400	IN	NS	m.root-servers.net.
;; Received 228 bytes from 192.168.1.1#53(192.168.1.1) in 12 ms

com.			172800	IN	NS	a.gtld-servers.net.
com.			172800	IN	NS	b.gtld-servers.net.
com.			172800	IN	NS	c.gtld-servers.net.
com.			172800	IN	NS	d.gtld-servers.net.
com.			172800	IN	NS	e.gtld-servers.net.
com.			172800	IN	NS	f.gtld-servers.net.
com.			172800	IN	NS	g.gtld-servers.net.
com.			172800	IN	NS	h.gtld-servers.net.
com.			172800	IN	NS	i.gtld-servers.net.
com.			172800	IN	NS	j.gtld-servers.net.
com.			172800	IN	NS	k.gtld-servers.net.
com.			172800	IN	NS	l.gtld-servers.net.
com.			172800	IN	NS	m.gtld-servers.net.
;; Received 511 bytes from 192.5.6.30#53(a.root-servers.net) in 45 ms

google.com.		172800	IN	NS	ns1.google.com.
google.com.		172800	IN	NS	ns2.google.com.
google.com.		172800	IN	NS	ns3.google.com.
google.com.		172800	IN	NS	ns4.google.com.
;; Received 180 bytes from 192.33.14.30#53(b.gtld-servers.net) in 38 ms

google.com.		300	IN	A	142.251.41.14
google.com.		300	IN	A	142.251.41.15
;; Received 74 bytes from 216.239.32.10#53(ns1.google.com) in 22 ms
```

**字段解释**：
- **.**: 根域名服务器
- **com.**: 顶级域名服务器（.com）
- **google.com.**: 权威域名服务器
- **Received**: 从服务器接收到的数据
- **bytes**: 接收到的字节数
- **#53**: DNS 服务器端口
- **in XX ms**: 响应时间（毫秒）

**反向查询**：
```bash
$ dig -x 142.251.41.14

; <<>> DiG 9.10.6 <<>> -x 142.251.41.14
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 98765
;; flags: qr rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;14.41.251.142.in-addr.arpa.	IN	PTR

;; ANSWER SECTION:
14.41.251.142.in-addr.arpa. 300 IN PTR lga25s72-in-f14.1e100.net.

;; Query time: 15 msec
;; SERVER: 192.168.1.1#53(192.168.1.1)
;; WHEN: Mon Jan 05 10:33:10 CST 2026
;; MSG SIZE  rcvd: 89
```

**字段解释**：
- **-x**: 反向查询标志
- **14.41.251.142.in-addr.arpa**: 反向 DNS 域名（IP 反转后加上 in-addr.arpa）
- **PTR**: 指针记录类型（反向 DNS 记录）
- **lga25s72-in-f14.1e100.net**: IP 对应的域名

---

### 3. host（简化工具）

#### 基本用法
```bash
# 查询 A 记录
host google.com

# 查询 MX 记录
host -t MX google.com

# 查询 TXT 记录
host -t TXT google.com

# 查询特定 DNS 服务器
host google.com 8.8.8.8

# 反向查询
host 142.251.41.14

# 详细输出
host -v google.com
```

#### 返回样例和字段解释

**查询 A 记录**：
```bash
$ host google.com
google.com has address 142.251.41.14
google.com has address 142.251.41.15
google.com has IPv6 address 2607:f8b0:4004:801::200e
```

**字段解释**：
- **has address**: 域名对应的 IPv4 地址
- **has IPv6 address**: 域名对应的 IPv6 地址

**查询 MX 记录**：
```bash
$ host -t MX google.com
google.com mail is handled by 10 smtp.google.com.
google.com mail is handled by 20 alt1.aspmx.l.google.com.
google.com mail is handled by 30 alt2.aspmx.l.google.com.
google.com mail is handled by 40 alt3.aspmx.l.google.com.
google.com mail is handled by 50 alt4.aspmx.l.google.com.
```

**字段解释**：
- **mail is handled by**: 邮件由以下服务器处理
- **10, 20, 30, 40, 50**: 优先级（数字越小优先级越高）

**查询 TXT 记录**：
```bash
$ host -t TXT google.com
google.com descriptive text "v=spf1 include:_spf.google.com ~all"
```

**字段解释**：
- **descriptive text**: 描述性文本（TXT 记录内容）

**反向查询**：
```bash
$ host 142.251.41.14
14.41.251.142.in-addr.arpa domain name pointer lga25s72-in-f14.1e100.net.
```

**字段解释**：
- **domain name pointer**: 域名指针（反向 DNS 记录）
- **lga25s72-in-f14.1e100.net**: IP 对应的域名

**详细输出**：
```bash
$ host -v google.com
Trying "google.com"
Using domain server:
Name: 192.168.1.1
Address: 192.168.1.1#53
Aliases:

google.com has address 142.251.41.14
google.com has address 142.251.41.15
google.com has IPv6 address 2607:f8b0:4004:801::200e
```

**字段解释**：
- **Trying**: 尝试查询的域名
- **Using domain server**: 使用的 DNS 服务器
- **Name**: DNS 服务器名称
- **Address**: DNS 服务器地址和端口
- **Aliases**: 别名列表（如果有）

---

### 工具对比

| 特性 | nslookup | dig | host |
|------|----------|-----|------|
| **输出详细度** | 中等 | 非常详细 | 简洁 |
| **使用场景** | 快速查询 | 深度分析 | 简单查询 |
| **递归追踪** | 不支持 | 支持（+trace） | 不支持 |
| **DNSSEC 支持** | 有限 | 完整支持 | 有限 |
| **脚本友好** | 一般 | 优秀（+short） | 优秀 |
| **交互模式** | 支持 | 不支持 | 不支持 |
| **默认工具** | Windows/Linux | Linux | Linux |

**推荐使用场景**：
- **nslookup**: Windows 系统快速查询、交互式调试
- **dig**: Linux 系统深度分析、DNSSEC 验证、递归追踪
- **host**: 脚本自动化、简单查询、反向查询

---

## DNS 性能优化

### 1. 本地 DNS 缓存
```bash
# 启用 systemd-resolved（Linux）
sudo systemctl start systemd-resolved
sudo systemctl enable systemd-resolved

# 查看缓存统计
systemd-resolve --statistics

# 刷新缓存
systemd-resolve --flush-caches

# dnsmasq（轻量级 DNS 缓存）
sudo apt install dnsmasq
sudo systemctl start dnsmasq
```

### 2. 使用高速 DNS 服务
```bash
# 常见的公共 DNS：
# Google:    8.8.8.8, 8.8.4.4
# CloudFlare: 1.1.1.1, 1.0.0.1
# OpenDNS:   208.67.222.222, 208.67.220.220
# 阿里:     223.5.5.5, 223.6.6.6

# 修改 DNS（Ubuntu）
sudo vim /etc/resolv.conf
# 或
sudo netplan edit 01-netcfg.yaml
```

### 3. DNS 预解析（浏览器）
```html
<!-- HTML 中添加 DNS 预解析 -->
<link rel="dns-prefetch" href="//cdn.example.com">
<link rel="dns-prefetch" href="//api.example.com">

<!-- 预连接（包括 DNS、TCP、TLS） -->
<link rel="preconnect" href="//cdn.example.com">
```

---

## 9. DNS 常见问题

### 问题 1：DNS 解析慢怎么办？

**症状**：访问网站时等待时间长，浏览器显示"正在解析主机名"。

**原因分析**：
1. DNS 服务器响应慢：ISP 提供的 DNS 服务器性能不佳
2. 网络延迟高：到 DNS 服务器的网络连接不稳定
3. DNS 缓存未命中：频繁查询未缓存的域名
4. DNS 查询链路长：递归查询经过多个服务器

**解决方案**：
```bash
# 1. 更换为高速公共 DNS 服务器
# Google DNS: 8.8.8.8, 8.8.4.4
# CloudFlare: 1.1.1.1, 1.0.0.1
# 阿里云: 223.5.5.5, 223.6.6.6

# 2. 测试不同 DNS 服务器的响应时间
time dig @8.8.8.8 google.com
time dig @1.1.1.1 google.com
time dig @223.5.5.5 google.com

# 3. 启用本地 DNS 缓存
# Linux (systemd-resolved)
sudo systemctl start systemd-resolved
sudo systemctl enable systemd-resolved

# 4. 配置浏览器 DNS 预解析
# 在 HTML 中添加：
<link rel="dns-prefetch" href="//cdn.example.com">

# 5. 使用 DNS 性能测试工具
sudo apt install namebench
namebench
```

---

### 问题 2：DNS 无法解析域名是什么原因？

**症状**：访问网站时提示"找不到服务器"或"DNS_PROBE_FINISHED_NXDOMAIN"。

**原因分析**：
1. DNS 服务器配置错误：/etc/resolv.conf 配置不正确
2. DNS 服务器故障：DNS 服务不可用或宕机
3. 网络连接问题：无法连接到 DNS 服务器
4. 域名不存在：域名未注册或已过期
5. DNS 记录配置错误：权威 DNS 服务器配置问题

**解决方案**：
```bash
# 1. 检查 DNS 服务器配置
cat /etc/resolv.conf
# 确保有正确的 nameserver 配置

# 2. 测试网络连接
ping 8.8.8.8
# 如果 ping 不通，说明网络有问题

# 3. 使用不同 DNS 服务器测试
nslookup example.com 8.8.8.8
nslookup example.com 1.1.1.1

# 4. 检查域名是否存在
whois example.com

# 5. 查看 DNS 查询详细过程
dig example.com +trace

# 6. 检查本地 hosts 文件
cat /etc/hosts
# 确保没有错误的域名映射

# 7. 重启网络服务
sudo systemctl restart NetworkManager
# 或
sudo systemctl restart networking
```

---

### 问题 3：什么是 DNS 缓存，如何清除？

**症状**：修改 DNS 记录后，域名仍然解析到旧的 IP 地址。

**DNS 缓存层级**：
1. **浏览器缓存**：浏览器保存 DNS 解析结果（几分钟）
2. **操作系统缓存**：系统级 DNS 缓存（Windows 有，Linux/Mac 通常没有）
3. **本地 DNS 服务器缓存**：ISP 或本地 DNS 服务器缓存（根据 TTL）
4. **递归解析器缓存**：公共 DNS 服务器缓存（根据 TTL）

**清除 DNS 缓存的方法**：
```bash
# 1. 清除浏览器缓存
# Chrome: chrome://net-internals/#dns → Clear host cache
# Firefox: 设置 → 隐私与安全 → 清除数据 → 缓存的图像和文件

# 2. 清除操作系统缓存
# Windows
ipconfig /flushdns

# macOS
sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder

# Linux (systemd-resolved)
sudo systemd-resolve --flush-caches

# Linux (nscd)
sudo systemctl restart nscd

# Linux (dnsmasq)
sudo systemctl restart dnsmasq

# 3. 强制不使用缓存查询
dig example.com +norecurse
nslookup -norecurse example.com

# 4. 修改 DNS 记录前的最佳实践
# 1. 先降低 TTL（如改为 60 秒）
# 2. 等待原 TTL 过期（通常是 24 小时）
# 3. 修改 DNS 记录
# 4. 等待新 TTL 生效
# 5. 恢复 TTL 为正常值（如 3600 秒）
```

---

### 问题 4：DNS 污染和劫持是什么，如何解决？

**症状**：访问正常网站时跳转到其他页面，或无法访问某些网站。

**DNS 污染**：
- 攻击者在 DNS 响应中注入错误的 IP 地址
- 通常用于屏蔽特定网站或进行钓鱼攻击
- 常见于网络审查和恶意攻击

**DNS 劫持**：
- 攻击者篡改 DNS 服务器配置
- 将所有查询重定向到恶意 DNS 服务器
- 可能导致用户访问钓鱼网站

**检测方法**：
```bash
# 1. 使用不同 DNS 服务器查询，比较结果
dig example.com @8.8.8.8
dig example.com @1.1.1.1
dig example.com @223.5.5.5
# 如果结果不一致，可能存在 DNS 污染

# 2. 检查 DNS 响应是否经过签名验证
dig example.com +dnssec
# 如果显示 "NOERROR" 且有 RRSIG 记录，说明通过 DNSSEC 验证

# 3. 使用 DNS 检测工具
# DNS Leak Test: https://www.dnsleaktest.com
# DNS Spy: https://dnsspy.io

# 4. 检查本地 DNS 配置
cat /etc/resolv.conf
# 确保没有被篡改
```

**解决方案**：
```bash
# 1. 使用可信的公共 DNS 服务器
# Google DNS: 8.8.8.8, 8.8.4.4
# CloudFlare: 1.1.1.1, 1.0.0.1
# OpenDNS: 208.67.222.222, 208.67.220.220

# 2. 启用 DNSSEC 验证
# 在 /etc/resolv.conf 中添加：
options edns0
# 或在 named.conf 中配置：
options {
    dnssec-validation auto;
};

# 3. 使用加密 DNS（DoH/DoT）
# 配置 DNS over HTTPS
# Chrome: chrome://settings/security → 使用安全 DNS
# 选择：使用自定义提供商
# 输入：https://dns.google/dns-query

# 配置 DNS over TLS
# /etc/systemd/resolved.conf
[Resolve]
DNS=1.1.1.1
DNSOverTLS=yes

# 4. 使用 VPN 或代理
# 通过加密隧道绕过 DNS 污染

# 5. 使用 DNS over HTTPS 代理
# 安装 cloudflared
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o cloudflared
chmod +x cloudflared
sudo mv cloudflared /usr/local/bin/

# 运行 DoH 代理
cloudflared proxy dns

# 配置系统使用本地 DoH 代理
echo "nameserver 127.0.0.1" | sudo tee /etc/resolv.conf
```

---

### 问题 5：什么是 DNS over HTTPS (DoH) 和 DNS over TLS (DoT)？

**DoH（DNS over HTTPS）**：
- 将 DNS 查询封装在 HTTPS 协议中
- 使用标准 HTTPS 端口（443）
- 查询流量伪装成普通 HTTPS 流量
- 难以被防火墙识别和拦截

**DoT（DNS over TLS）**：
- 将 DNS 查询封装在 TLS 协议中
- 使用专用 DNS 端口（853）
- 专门为 DNS 设计的加密协议
- 容易被防火墙识别和拦截

**对比表格**：
| 特性 | DoH | DoT |
|-----|-----|-----|
| 协议 | HTTPS | TLS |
| 端口 | 443 | 853 |
| 隐蔽性 | 高（伪装 HTTPS） | 低（专用端口） |
| 性能开销 | 较大（TLS + HTTP） | 较小（仅 TLS） |
| 部署难度 | 中等 | 简单 |
| 标准化 | RFC 8484 | RFC 7858 |
| 浏览器支持 | Chrome, Firefox | 部分支持 |
| 防火墙检测 | 困难 | 容易 |

**使用场景**：
- **DoH**：适合需要隐蔽性的场景，如绕过网络审查
- **DoT**：适合需要性能的场景，如企业内部网络

**配置示例**：
```bash
# 1. 使用 DoH（curl）
curl -H "accept: application/dns-json" "https://dns.google/resolve?name=example.com&type=A"

# 2. 使用 DoT（kdig）
sudo apt install knot-dnsutils
kdig @dns.google +tls +port=853 example.com

# 3. 配置 systemd-resolved 使用 DoT
sudo vim /etc/systemd/resolved.conf
[Resolve]
DNS=1.1.1.1
DNSOverTLS=yes
FallbackDNS=8.8.8.8

sudo systemctl restart systemd-resolved

# 4. 配置浏览器使用 DoH
# Chrome: chrome://settings/security → 使用安全 DNS
# 选择：使用自定义提供商
# 输入：https://dns.google/dns-query

# Firefox: 设置 → 常规 → 网络设置 → 启用基于 HTTPS 的 DNS
# 选择：提供商
# 输入：https://dns.google/dns-query

# 5. 使用 cloudflared 作为 DoH 代理
cloudflared proxy dns --port 5353 --upstream https://1.1.1.1/dns-query

# 配置系统使用本地 DoH 代理
echo "nameserver 127.0.0.1" | sudo tee /etc/resolv.conf
```

**注意事项**：
- DoH/DoT 会增加 DNS 查询延迟（通常 10-50ms）
- 部分网络环境可能阻止 DoH/DoT 连接
- 需要确保 DNS 服务器支持 DoH/DoT
- 企业网络可能需要配置防火墙规则

---

## 10. DNS 与网络性能关系

```
DNS 解析时间（通常 10-100ms）
  ↓
TCP 建立（3 次握手，30-100ms）
  ↓
TLS 握手（HTTPS，30-100ms）
  ↓
HTTP 请求/响应
  ↓
内容渲染
  
总页面加载时间

DNS 解析占比：5-10%（对于重复访问可减少为 0%）
优化 DNS 可显著加快首次访问速度
```
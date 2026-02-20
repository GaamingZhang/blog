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

# DNS：互联网的电话簿

## 为什么需要 DNS？

### 计算机的困境

```
计算机通信的矛盾：
    计算机：只认识 IP 地址（142.251.41.14）
    人类：  只记得域名（google.com）

    问题：
        让人记住 IP 地址？ → 太难记忆
        让计算机理解域名？ → 需要转换机制
```

### 早期的解决方案：hosts 文件

```
古老的方式：
    每台计算机维护一个本地文件（/etc/hosts）
    记录域名到 IP 的映射

    google.com  142.251.41.14
    facebook.com 157.240.241.35
    ...

问题：
    - 手动维护，无法扩展
    - 新网站需要手动添加
    - 修改需要同步到所有机器
    - 互联网有数十亿域名，无法管理
```

### DNS 的诞生

```
核心思想：
    分布式数据库 + 层级结构

    不再由单个文件存储所有映射
    而是分散到全球的 DNS 服务器
    每个服务器只负责一部分域名
```

## DNS 的层级结构

### 四级服务器体系

```
完整的 DNS 层次结构：

                      . (根)
                      ↓
        ┌─────────────┼─────────────┐
        ↓             ↓             ↓
      .com          .org          .cn (顶级域 TLD)
        ↓             ↓             ↓
    ┌───┴───┐     ┌───┴───┐     ┌───┴───┐
    ↓       ↓     ↓       ↓     ↓       ↓
 google  facebook  wikipedia  baidu  (二级域)
    ↓       ↓         ↓         ↓
  www     api      cdn        mail   (三级域/子域)

每一层都有对应的服务器负责解析
```

### 域名的组成

```
完整域名：www.google.com.

从右到左读取：
    .       → 根域（通常省略）
    com.    → 顶级域（TLD）
    google. → 二级域（组织/公司）
    www.    → 三级域（主机/服务）

每个点代表一个层级
每个层级由对应的 DNS 服务器管理
```

### 四级服务器职责

```
1. 根域名服务器（Root Nameserver）
    位置：全球 13 个根服务器（a-m.root-servers.net）
    职责：告诉你去哪里找顶级域服务器
    数据：存储所有顶级域的地址

2. 顶级域服务器（TLD Nameserver）
    位置：每个顶级域一组服务器
    职责：告诉你去哪里找权威服务器
    示例：.com, .org, .net, .cn 等

3. 权威域名服务器（Authoritative Nameserver）
    位置：域名所有者的服务器
    职责：返回域名的真实 IP 地址
    示例：google.com 的权威服务器

4. 本地 DNS 解析器（Recursive Resolver）
    位置：ISP 或公共 DNS（8.8.8.8）
    职责：代替用户完成整个查询过程
    作用：缓存结果，减少查询次数
```

## DNS 查询的完整流程

### 正向查询：域名 → IP

```
场景：用户访问 www.google.com

第1步：浏览器缓存
    浏览器：有缓存吗？
    结果：有 → 直接使用，结束
         无 → 继续下一步

第2步：操作系统缓存
    OS：有缓存吗？
    结果：有 → 返回给浏览器，结束
         无 → 继续下一步

第3步：hosts 文件
    OS：/etc/hosts 有记录吗？
    结果：有 → 返回，结束
         无 → 继续下一步

第4步：本地 DNS 解析器（递归查询开始）
    Client → 本地 DNS (8.8.8.8)
    请求："www.google.com 的 IP 是多少？"

第5步：查询根服务器
    本地 DNS → 根服务器 (a.root-servers.net)
    请求："www.google.com 在哪里？"
    回复："我不知道具体地址，但你可以去问 .com 服务器"
    返回：.com 服务器的地址

第6步：查询顶级域服务器
    本地 DNS → .com 服务器
    请求："google.com 在哪里？"
    回复："我不知道具体地址，但你可以去问 google.com 的权威服务器"
    返回：google.com 权威服务器的地址

第7步：查询权威服务器
    本地 DNS → google.com 权威服务器
    请求："www.google.com 的 IP 是多少？"
    回复："142.251.41.14"

第8步：缓存并返回
    本地 DNS 缓存结果（TTL=300秒）
    本地 DNS → Client
    返回："142.251.41.14"

第9步：浏览器发起连接
    Client → 142.251.41.14
    建立 HTTP/HTTPS 连接
```

### 递归查询 vs 迭代查询

```
递归查询（Recursive Query）：
    客户端："帮我找到答案"
    本地 DNS："好的，我来帮你找完整答案"

    特点：
        客户端只发送一次请求
        本地 DNS 代劳所有工作
        返回最终结果

    过程：
        Client → 本地 DNS："www.google.com 的 IP？"
        本地 DNS 自己去查询根、TLD、权威服务器
        本地 DNS → Client："142.251.41.14"

迭代查询（Iterative Query）：
    本地 DNS："www.google.com 在哪里？"
    根服务器："我不知道，但你可以去问 .com"

    特点：
        每次查询返回一个指针
        查询方自己逐级查询
        不返回最终答案，只返回下一步去哪问

    过程：
        本地 DNS → 根："www.google.com？"
        根 → 本地 DNS："去问 .com（地址：x.x.x.x）"

        本地 DNS → .com："www.google.com？"
        .com → 本地 DNS："去问 google.com（地址：y.y.y.y）"

        本地 DNS → google.com："www.google.com？"
        google.com → 本地 DNS："142.251.41.14"

实际组合：
    客户端 → 本地 DNS：递归查询
    本地 DNS → 根/TLD/权威：迭代查询
```

### 反向查询：IP → 域名

```
场景：已知 IP 142.251.41.14，想知道域名

特殊域名：in-addr.arpa
    IP：142.251.41.14
    反转：14.41.251.142
    添加后缀：14.41.251.142.in-addr.arpa

查询流程：
    本地 DNS → 根服务器
    请求："14.41.251.142.in-addr.arpa 的 PTR 记录？"

    根 → .arpa 服务器
    .arpa → in-addr.arpa 服务器
    in-addr.arpa → 142.in-addr.arpa 服务器
    ...
    最终返回：lga25s72-in-f14.1e100.net

用途：
    - 邮件服务器验证（防垃圾邮件）
    - 日志分析（IP → 域名）
    - 安全审计
```

## DNS 记录类型

### 常用记录

```
A 记录（Address）：
    作用：域名 → IPv4 地址
    示例：google.com → 142.251.41.14

AAAA 记录：
    作用：域名 → IPv6 地址
    示例：google.com → 2607:f8b0:4004:801::200e

CNAME 记录（Canonical Name）：
    作用：别名 → 真实域名
    示例：www.google.com → google.com

    解析过程：
        查询 www.google.com
        → 发现 CNAME 指向 google.com
        → 再查询 google.com 的 A 记录
        → 得到 142.251.41.14

    限制：
        CNAME 不能与其他记录共存
        根域名不能使用 CNAME

MX 记录（Mail Exchange）：
    作用：指定邮件服务器
    示例：gmail.com 的邮件服务器

    包含优先级：
        gmail.com  MX  10  smtp1.google.com.
        gmail.com  MX  20  smtp2.google.com.

        数字越小优先级越高
        发邮件时先尝试 smtp1

NS 记录（Name Server）：
    作用：指定权威 DNS 服务器
    示例：google.com 的权威服务器

    google.com  NS  ns1.google.com.
    google.com  NS  ns2.google.com.

TXT 记录（Text）：
    作用：存储文本信息
    用途：
        - SPF（防垃圾邮件）
        - DKIM（邮件签名）
        - 域名验证（Google, SSL 证书）
        - 任意文本信息

PTR 记录（Pointer）：
    作用：IP → 域名（反向解析）
    示例：14.41.251.142.in-addr.arpa → google.com

SOA 记录（Start of Authority）：
    作用：记录域的权威信息
    内容：
        - 主 DNS 服务器
        - 管理员邮箱
        - 序列号（版本）
        - 刷新时间
        - 重试时间
        - 过期时间
        - 最小 TTL
```

### 记录示例

```
域名解析配置：

example.com.         IN  A      192.0.2.1
www.example.com.     IN  CNAME  example.com.
mail.example.com.    IN  A      192.0.2.10
example.com.         IN  MX  10 mail.example.com.
example.com.         IN  NS     ns1.example.com.
example.com.         IN  NS     ns2.example.com.
example.com.         IN  TXT    "v=spf1 mx -all"

解析过程：
    1. 查询 www.example.com
       → CNAME 指向 example.com
       → 查询 example.com 的 A 记录
       → 得到 192.0.2.1

    2. 查询 example.com 的 MX 记录
       → 得到 mail.example.com
       → 查询 mail.example.com 的 A 记录
       → 得到 192.0.2.10
```

## DNS 缓存与 TTL

### 缓存的层级

```
多级缓存体系：

Level 1: 浏览器缓存
    时长：几分钟（浏览器自定义）
    作用：同一页面重复访问

Level 2: 操作系统缓存
    时长：根据 TTL
    作用：不同应用共享

Level 3: 本地 DNS 缓存
    时长：根据 TTL
    作用：同一网络用户共享

Level 4: 权威服务器
    时长：永久
    作用：域名的真实记录

缓存命中率：
    首次访问：需要完整查询（4 步）
    重复访问：直接使用缓存（0 步）
```

### TTL（Time To Live）

```
含义：
    DNS 记录在缓存中的生存时间（秒）

工作原理：
    权威服务器返回记录时附带 TTL：
        google.com  300  IN  A  142.251.41.14
                    ↑
                  TTL=300秒

    本地 DNS 缓存这条记录 300 秒
    300 秒后，缓存失效，需要重新查询

TTL 的权衡：
    TTL 短（60秒）：
        优点：修改 DNS 快速生效
        缺点：查询次数多，负载高

    TTL 长（86400秒=1天）：
        优点：查询次数少，性能好
        缺点：修改 DNS 生效慢

    常见设置：
        静态网站：3600秒（1小时）
        CDN：      300秒（5分钟）
        负载均衡：  60秒（1分钟）
        修改前：    60秒（临时降低）
```

### DNS 修改的生效时间

```
场景：修改网站 IP 地址

问题：
    修改 DNS 后，为什么有些用户还访问旧 IP？

原因：
    旧记录仍在各级缓存中

    时间轴：
        T0: 原 TTL=3600秒，IP=1.1.1.1
        T1: 修改 IP=2.2.2.2
        T2: 用户查询
            如果缓存未过期 → 仍得到 1.1.1.1
            如果缓存已过期 → 得到 2.2.2.2

最佳实践：
    T-24h: 降低 TTL 为 60 秒
    T-1h:  等待旧 TTL 过期
    T0:    修改 DNS 记录
    T+1h:  确认生效
    T+24h: 恢复 TTL 为 3600 秒
```

## DNS 负载均衡

### 原理

```
同一域名配置多个 IP：
    example.com  A  192.0.2.1
    example.com  A  192.0.2.2
    example.com  A  192.0.2.3

DNS 服务器轮询返回：
    查询 1 → 192.0.2.1
    查询 2 → 192.0.2.2
    查询 3 → 192.0.2.3
    查询 4 → 192.0.2.1（循环）

结果：
    流量分散到多台服务器
```

### 地理负载均衡（GeoDNS）

```
根据客户端位置返回最近的服务器：

    北美用户 → 美国服务器（1.1.1.1）
    欧洲用户 → 欧洲服务器（2.2.2.2）
    亚洲用户 → 亚洲服务器（3.3.3.3）

实现：
    DNS 服务器根据客户端 IP 判断地理位置
    返回对应地区的服务器 IP

优点：
    - 减少延迟
    - 提高速度
    - 分散负载

提供商：
    CloudFlare, AWS Route 53, Google Cloud DNS
```

### 健康检查与故障转移

```
场景：主服务器故障

正常情况：
    example.com  A  192.0.2.1（主）
    example.com  A  192.0.2.2（备）

故障检测：
    DNS 服务器定期检查服务器健康
    主服务器：192.0.2.1（DOWN）
    备用服务器：192.0.2.2（UP）

故障转移：
    仅返回健康的服务器：
    example.com  A  192.0.2.2

恢复：
    主服务器恢复后，重新加入
    example.com  A  192.0.2.1
    example.com  A  192.0.2.2
```

## DNS 安全

### DNS 欺骗/投毒

```
攻击原理：
    1. 攻击者截获 DNS 查询
    2. 抢在真实服务器前返回假答案
    3. 用户访问假的 IP 地址

    用户 → 本地 DNS："google.com 的 IP？"
    攻击者抢答："123.45.67.89"（钓鱼网站）
    用户 → 访问钓鱼网站

防护：DNSSEC
    原理：数字签名验证

    权威服务器：
        使用私钥对记录签名
        google.com  A  142.251.41.14  签名:xxxxx

    本地 DNS：
        使用公钥验证签名
        签名正确 → 记录可信
        签名错误 → 拒绝

    信任链：
        根 → TLD → 权威服务器
        逐级验证，确保完整性
```

### DNS 劫持

```
攻击方式：
    1. 篡改路由器 DNS 设置
    2. 恶意软件修改系统 DNS
    3. ISP 劫持（广告注入）

特征：
    访问任何网站都跳转到广告页
    某些网站无法访问

防护：
    - 使用可信公共 DNS（8.8.8.8, 1.1.1.1）
    - 使用 HTTPS（防止内容篡改）
    - 启用 DNSSEC
```

### 加密 DNS：DoH 和 DoT

```
问题：
    传统 DNS 查询明文传输
    ISP/中间人可以看到你访问了哪些网站
    可以拦截和篡改

DoH（DNS over HTTPS）：
    端口：443（HTTPS）
    特点：DNS 查询伪装成 HTTPS 流量
    优点：难以被识别和拦截
    缺点：性能开销稍大

    请求：
        HTTPS://dns.google/dns-query?name=example.com

DoT（DNS over TLS）：
    端口：853（专用）
    特点：专门为 DNS 设计的加密
    优点：性能开销小
    缺点：容易被防火墙拦截

对比：
    传统 DNS：明文，端口 53
    DoT：     加密，端口 853（专用）
    DoH：     加密，端口 443（伪装）

使用场景：
    DoH：需要绕过审查
    DoT：企业内网，性能优先
```

## DNS 性能优化

### 本地缓存

```
目的：
    减少 DNS 查询次数
    加快访问速度

实现：
    浏览器缓存：几分钟
    OS 缓存：根据 TTL
    本地 DNS 服务：dnsmasq, systemd-resolved

效果：
    首次访问：需要查询（100ms）
    重复访问：使用缓存（0ms）
```

### DNS 预解析

```
问题：
    页面加载时才开始 DNS 解析
    DNS 查询阻塞后续请求

解决：
    浏览器预解析

    HTML 中添加：
        <link rel="dns-prefetch" href="//cdn.example.com">
        <link rel="dns-prefetch" href="//api.example.com">

    原理：
        浏览器提前解析这些域名
        真正请求时直接使用缓存

    效果：
        减少 DNS 解析延迟
        加快页面加载速度
```

### 选择快速 DNS

```
公共 DNS 服务：
    Google:     8.8.8.8, 8.8.4.4
    Cloudflare: 1.1.1.1, 1.0.0.1
    阿里云:     223.5.5.5, 223.6.6.6

对比 ISP DNS：
    ISP DNS：
        - 可能较慢
        - 可能有劫持
        - 可能记录查询日志

    公共 DNS：
        - 通常更快
        - 全球分布
        - 隐私保护更好

选择建议：
    测试不同 DNS 的响应时间
    选择最快的
```

## 小结

DNS 工作原理的核心要点：

**本质**：
- 人类记域名，计算机识 IP
- DNS 是分布式的域名 → IP 转换系统

**层级结构**：
- 根服务器（13 个）
- 顶级域服务器（.com, .org 等）
- 权威服务器（google.com 等）
- 本地 DNS 解析器（递归查询）

**查询流程**：
- 浏览器缓存 → OS 缓存 → 本地 DNS
- 本地 DNS → 根 → TLD → 权威
- 逐级查询，最终得到 IP

**查询模式**：
- 递归查询：客户端 → 本地 DNS（代劳）
- 迭代查询：本地 DNS → 根/TLD/权威（逐级）

**记录类型**：
- A（IPv4）、AAAA（IPv6）
- CNAME（别名）、MX（邮件）
- NS（权威服务器）、TXT（文本）
- PTR（反向解析）

**缓存机制**：
- 多级缓存（浏览器、OS、本地 DNS）
- TTL 控制缓存时间
- 平衡性能与灵活性

**负载均衡**：
- 多 IP 轮询
- 地理位置路由
- 健康检查与故障转移

**安全问题**：
- DNS 欺骗/投毒 → DNSSEC
- DNS 劫持 → 使用可信 DNS
- 查询泄露 → DoH/DoT 加密

**性能优化**：
- 本地缓存（减少查询）
- DNS 预解析（提前解析）
- 快速 DNS 服务（减少延迟）

理解 DNS 的分布式、层级化设计，是理解互联网基础设施的关键。DNS 不仅仅是域名解析，还承载着负载均衡、故障转移、安全防护等重要功能。

## 参考资源

- [RFC 1034 - DNS 概念](https://datatracker.ietf.org/doc/html/rfc1034)
- [RFC 1035 - DNS 协议](https://datatracker.ietf.org/doc/html/rfc1035)
- [Cloudflare DNS 学习中心](https://www.cloudflare.com/learning/dns/what-is-dns/)

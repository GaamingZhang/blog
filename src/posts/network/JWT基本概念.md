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

# JWT基本概念

## 概述

JWT（JSON Web Token）是一种开放标准（RFC 7519），用于在各方之间安全地传输信息。它是一种紧凑的、自包含的令牌格式，可以被用于身份认证和信息交换。JWT由三部分组成：头部（Header）、载荷（Payload）和签名（Signature），这三部分通过点（.）连接在一起。

### 产生背景

在传统的Web应用中，用户认证通常依赖于Session机制。服务器在内存或数据库中存储Session信息，并通过Cookie将Session ID发送给客户端。然而，随着微服务架构、移动应用和单页应用（SPA）的兴起，传统的Session机制面临以下挑战：

- **跨域问题**：Session默认不支持跨域访问
- **服务器负担**：服务器需要维护Session状态，增加服务器负担
- **扩展性差**：在分布式系统中，Session共享和同步复杂
- **移动应用支持**：移动应用对Cookie的支持有限

为了解决这些问题，JWT应运而生。JWT将用户信息编码到令牌中，客户端存储令牌，服务器通过验证令牌签名来确认用户身份，无需在服务器端维护Session状态。

### 核心价值

- **无状态**：服务器不需要存储Session状态，减轻服务器负担
- **跨域支持**：可以在不同的域名和平台之间传递
- **自包含**：令牌本身包含了所有必要的信息
- **可扩展**：可以在载荷中添加自定义声明
- **标准化**：遵循RFC 7519标准，具有良好的互操作性
- **安全性**：通过签名保证令牌的完整性和真实性

### 应用场景

- **身份认证**：用户登录后获取JWT，后续请求携带JWT进行认证
- **信息交换**：在各方之间安全地传递信息
- **单点登录（SSO）**：实现多个应用之间的统一认证
- **API授权**：保护API接口，验证调用者身份
- **微服务架构**：在微服务之间传递用户身份信息

## JWT结构

JWT由三部分组成，每部分之间用点（.）分隔：

```
Header.Payload.Signature
```

### 1. 头部（Header）

头部通常包含两部分信息：令牌的类型（typ）和所使用的签名算法（alg）。

```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

然后，这个JSON被Base64Url编码，形成JWT的第一部分。

#### 常见的签名算法

- **HS256**：HMAC SHA256（对称加密）
- **HS384**：HMAC SHA384
- **HS512**：HMAC SHA512
- **RS256**：RSA SHA256（非对称加密）
- **RS384**：RSA SHA384
- **RS512**：RSA SHA512
- **ES256**：ECDSA SHA256
- **ES384**：ECDSA SHA384
- **ES512**：ECDSA SHA512
- **PS256**：RSASSA-PSS SHA256

### 2. 载荷（Payload）

载荷包含声明（Claims），声明是关于实体（通常是用户）和其他数据的声明。声明分为三种类型：

#### 标准声明（Registered Claims）

这些是预定义的声明，不是强制性的，但推荐使用：

- **iss**（issuer）：签发人
- **sub**（subject）：主题
- **aud**（audience）：接收方
- **exp**（expiration time）：过期时间
- **nbf**（not before）：生效时间
- **iat**（issued at）：签发时间
- **jti**（JWT ID）：JWT的唯一标识

```json
{
  "sub": "1234567890",
  "name": "John Doe",
  "iat": 1516239022,
  "exp": 1516242622
}
```

#### 公共声明（Public Claims）

这些声明可以由使用JWT的人随意定义。但为了避免冲突，应该使用URI格式的名称，或者使用IANA JSON Web Token Registry中注册的名称。

```json
{
  "https://example.com/claims": {
    "role": "admin",
    "department": "engineering"
  }
}
```

#### 私有声明（Private Claims）

这些是自定义的声明，用于在同意使用它们的各方之间共享信息。

```json
{
  "userId": "123456",
  "username": "johndoe",
  "role": "admin"
}
```

然后，这个JSON被Base64Url编码，形成JWT的第二部分。

### 3. 签名（Signature）

签名用于验证消息在传输过程中是否被篡改，并且对于使用私钥签名的令牌，还可以验证发送者是谁。

签名的生成过程：

```
HMACSHA256(
  base64UrlEncode(header) + "." + base64UrlEncode(payload),
  secret
)
```

或者对于RSA签名：

```
RSASHA256(
  base64UrlEncode(header) + "." + base64UrlEncode(payload),
  privateKey
)
```

#### 完整的JWT示例

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
```

解码后：

- **Header**：
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

- **Payload**：
```json
{
  "sub": "1234567890",
  "name": "John Doe",
  "iat": 1516239022
}
```

- **Signature**：使用secret对header和payload进行签名

## 工作原理

### 认证流程

```
1. 用户登录
   ↓
2. 服务器验证用户凭证
   ↓
3. 服务器生成JWT（包含用户信息和过期时间）
   ↓
4. 服务器使用密钥对JWT进行签名
   ↓
5. 服务器将JWT返回给客户端
   ↓
6. 客户端存储JWT（通常在localStorage或Cookie中）
   ↓
7. 客户端在后续请求中携带JWT（通常在Authorization头中）
   ↓
8. 服务器验证JWT签名和过期时间
   ↓
9. 服务器从JWT中提取用户信息
   ↓
10. 服务器处理请求并返回响应
```

### 代码示例

#### 生成JWT

```javascript
const jwt = require('jsonwebtoken');

const payload = {
  userId: '123456',
  username: 'johndoe',
  role: 'admin'
};

const secret = 'your-secret-key';
const options = {
  expiresIn: '1h',
  issuer: 'example.com',
  audience: 'example-api'
};

const token = jwt.sign(payload, secret, options);
console.log(token);
```

#### 验证JWT

```javascript
const jwt = require('jsonwebtoken');

const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...';

const secret = 'your-secret-key';
const options = {
  issuer: 'example.com',
  audience: 'example-api'
};

try {
  const decoded = jwt.verify(token, secret, options);
  console.log(decoded);
} catch (err) {
  console.error('Token验证失败:', err.message);
}
```

#### 解码JWT（不验证签名）

```javascript
const jwt = require('jsonwebtoken');

const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...';

const decoded = jwt.decode(token);
console.log(decoded);
```

#### 中间件示例

```javascript
const jwt = require('jsonwebtoken');
const secret = 'your-secret-key';

function authenticateToken(req, res, next) {
  const authHeader = req.headers['authorization'];
  const token = authHeader && authHeader.split(' ')[1];

  if (!token) {
    return res.status(401).json({ error: '未提供访问令牌' });
  }

  jwt.verify(token, secret, (err, user) => {
    if (err) {
      return res.status(403).json({ error: '无效的访问令牌' });
    }

    req.user = user;
    next();
  });
}

app.get('/protected', authenticateToken, (req, res) => {
  res.json({ message: '访问成功', user: req.user });
});
```

## JWT与Session的区别

### 对比表格

| 特性 | JWT | Session |
|------|-----|---------|
| **存储位置** | 客户端（localStorage、Cookie） | 服务器端（内存、数据库、Redis） |
| **状态** | 无状态 | 有状态 |
| **服务器负担** | 低（无需存储Session） | 高（需要存储和维护Session） |
| **扩展性** | 好（支持分布式） | 差（需要Session共享机制） |
| **跨域支持** | 好（天然支持） | 差（需要特殊配置） |
| **安全性** | 中（无法主动失效） | 高（可以主动失效） |
| **令牌大小** | 较大（包含用户信息） | 较小（仅Session ID） |
| **过期控制** | 客户端（令牌过期前一直有效） | 服务器端（可以随时失效） |
| **移动应用** | 好 | 一般 |
| **单点登录** | 容易实现 | 需要额外机制 |

### 适用场景

#### JWT适用的场景

- **微服务架构**：多个服务之间需要共享用户身份信息
- **移动应用**：移动应用对Cookie支持有限
- **单页应用（SPA）**：前后端分离，需要跨域访问
- **API服务**：RESTful API需要无状态认证
- **单点登录（SSO）**：多个应用之间统一认证
- **第三方集成**：需要与外部系统进行身份验证

#### Session适用的场景

- **传统Web应用**：前后端不分离，使用Cookie
- **需要主动失效**：需要立即撤销用户权限
- **敏感操作**：需要更严格的会话控制
- **短期会话**：会话时间较短，不需要持久化
- **服务器资源充足**：服务器有足够的资源维护Session

## 安全问题和防护措施

### 1. 算法混淆攻击

#### 原理

攻击者将JWT的签名算法从非对称加密（如RS256）改为对称加密（如HS256），然后使用公钥作为密钥来伪造签名。

#### 防护措施

```javascript
const jwt = require('jsonwebtoken');

const publicKey = fs.readFileSync('public.key');
const privateKey = fs.readFileSync('private.key');

const token = jwt.sign(payload, privateKey, { algorithm: 'RS256' });

const decoded = jwt.verify(token, publicKey, {
  algorithms: ['RS256']
});
```

### 2. 空算法攻击

#### 原理

攻击者将JWT的算法设置为"none"，从而绕过签名验证。

#### 防护措施

```javascript
const decoded = jwt.verify(token, secret, {
  algorithms: ['HS256', 'RS256']
});
```

### 3. 密钥泄露

#### 原理

如果签名密钥泄露，攻击者可以伪造任意JWT。

#### 防护措施

```javascript
const jwt = require('jsonwebtoken');

const secret = process.env.JWT_SECRET;

const token = jwt.sign(payload, secret, {
  expiresIn: '1h',
  algorithm: 'HS256'
});

const decoded = jwt.verify(token, secret, {
  algorithms: ['HS256']
});
```

使用非对称加密：

```javascript
const jwt = require('jsonwebtoken');

const privateKey = fs.readFileSync('private.key');
const publicKey = fs.readFileSync('public.key');

const token = jwt.sign(payload, privateKey, {
  algorithm: 'RS256'
});

const decoded = jwt.verify(token, publicKey, {
  algorithms: ['RS256']
});
```

### 4. 令牌泄露

#### 原理

JWT存储在客户端，如果被窃取，攻击者可以冒充用户。

#### 防护措施

```javascript
const jwt = require('jsonwebtoken');

const token = jwt.sign(payload, secret, {
  expiresIn: '15m'
});

res.cookie('jwt', token, {
  httpOnly: true,
  secure: true,
  sameSite: 'strict',
  maxAge: 15 * 60 * 1000
});
```

使用短期令牌 + 刷新令牌：

```javascript
const accessToken = jwt.sign({ userId: user.id }, secret, {
  expiresIn: '15m'
});

const refreshToken = jwt.sign({ userId: user.id }, refreshSecret, {
  expiresIn: '7d'
});

res.json({
  accessToken,
  refreshToken
});
```

### 5. 令牌重放攻击

#### 原理

攻击者截获有效的JWT，并在过期前重复使用。

#### 防护措施

```javascript
const jwt = require('jsonwebtoken');

const token = jwt.sign(payload, secret, {
  expiresIn: '15m',
  jti: generateUniqueId()
});

const decoded = jwt.verify(token, secret, {
  algorithms: ['HS256']
});

if (isTokenBlacklisted(decoded.jti)) {
  throw new Error('令牌已被撤销');
}
```

### 6. 时间攻击

#### 原理

攻击者通过测量验证时间来推断密钥信息。

#### 防护措施

使用恒定时间比较：

```javascript
const crypto = require('crypto');

function constantTimeCompare(a, b) {
  if (a.length !== b.length) {
    return false;
  }
  return crypto.timingSafeEqual(Buffer.from(a), Buffer.from(b));
}
```

### 7. 信息泄露

#### 原理

JWT的载荷是Base64编码的，可以被任何人解码，不应在载荷中存储敏感信息。

#### 防护措施

```javascript
const payload = {
  userId: '123456',
  username: 'johndoe',
  role: 'admin'
};

const token = jwt.sign(payload, secret, {
  expiresIn: '1h'
});
```

不要在载荷中存储敏感信息：
- ❌ 密码
- ❌ 信用卡号
- ❌ 个人身份信息
- ❌ 敏感的业务数据

## 使用场景和最佳实践

### 1. 身份认证

```javascript
const express = require('express');
const jwt = require('jsonwebtoken');

const app = express();
const secret = 'your-secret-key';

app.post('/login', (req, res) => {
  const { username, password } = req.body;

  if (authenticate(username, password)) {
    const user = getUser(username);
    const token = jwt.sign(
      {
        userId: user.id,
        username: user.username,
        role: user.role
      },
      secret,
      {
        expiresIn: '1h',
        issuer: 'example.com',
        audience: 'example-api'
      }
    );

    res.json({ token });
  } else {
    res.status(401).json({ error: '认证失败' });
  }
});
```

### 2. 刷新令牌

```javascript
const jwt = require('jsonwebtoken');

const secret = 'your-secret-key';
const refreshSecret = 'your-refresh-secret';

app.post('/refresh', (req, res) => {
  const { refreshToken } = req.body;

  try {
    const decoded = jwt.verify(refreshToken, refreshSecret);
    const user = getUserById(decoded.userId);

    const newAccessToken = jwt.sign(
      {
        userId: user.id,
        username: user.username,
        role: user.role
      },
      secret,
      { expiresIn: '15m' }
    );

    res.json({ accessToken: newAccessToken });
  } catch (err) {
    res.status(401).json({ error: '无效的刷新令牌' });
  }
});
```

### 3. 权限控制

```javascript
function authorize(roles) {
  return (req, res, next) => {
    if (!req.user) {
      return res.status(401).json({ error: '未认证' });
    }

    if (!roles.includes(req.user.role)) {
      return res.status(403).json({ error: '权限不足' });
    }

    next();
  };
}

app.get('/admin', authenticateToken, authorize(['admin']), (req, res) => {
  res.json({ message: '管理员页面' });
});
```

### 4. 令牌黑名单

```javascript
const redis = require('redis');
const client = redis.createClient();

async function revokeToken(token) {
  const decoded = jwt.decode(token);
  const ttl = decoded.exp - Math.floor(Date.now() / 1000);
  await client.setex(`blacklist:${decoded.jti}`, ttl, '1');
}

async function isTokenBlacklisted(jti) {
  const result = await client.get(`blacklist:${jti}`);
  return result !== null;
}

app.post('/logout', authenticateToken, async (req, res) => {
  const token = req.headers['authorization'].split(' ')[1];
  await revokeToken(token);
  res.json({ message: '登出成功' });
});
```

### 5. 最佳实践

#### 使用HTTPS

```javascript
app.use((req, res, next) => {
  if (!req.secure && process.env.NODE_ENV === 'production') {
    return res.redirect('https://' + req.headers.host + req.url);
  }
  next();
});
```

#### 设置合理的过期时间

```javascript
const token = jwt.sign(payload, secret, {
  expiresIn: '15m'
});
```

#### 使用强密钥

```javascript
const crypto = require('crypto');

const secret = crypto.randomBytes(64).toString('hex');
```

#### 验证所有声明

```javascript
const decoded = jwt.verify(token, secret, {
  algorithms: ['HS256'],
  issuer: 'example.com',
  audience: 'example-api',
  maxAge: '1h'
});
```

#### 使用非对称加密

```javascript
const token = jwt.sign(payload, privateKey, {
  algorithm: 'RS256'
});

const decoded = jwt.verify(token, publicKey, {
  algorithms: ['RS256']
});
```

#### 不要在载荷中存储敏感信息

```javascript
const payload = {
  userId: '123456',
  username: 'johndoe',
  role: 'admin'
};

const token = jwt.sign(payload, secret);
```

#### 使用短期访问令牌 + 长期刷新令牌

```javascript
const accessToken = jwt.sign(payload, secret, {
  expiresIn: '15m'
});

const refreshToken = jwt.sign(payload, refreshSecret, {
  expiresIn: '7d'
});
```

#### 实现令牌黑名单

```javascript
async function revokeToken(token) {
  const decoded = jwt.decode(token);
  const ttl = decoded.exp - Math.floor(Date.now() / 1000);
  await redis.setex(`blacklist:${decoded.jti}`, ttl, '1');
}
```

#### 记录令牌使用情况

```javascript
app.use(authenticateToken, (req, res, next) => {
  logTokenUsage(req.user, req.ip, req.path);
  next();
});
```

## 常见问题

### 1. 什么是JWT？它由哪几部分组成？

JWT（JSON Web Token）是一种开放标准（RFC 7519），用于在各方之间安全地传输信息。它是一种紧凑的、自包含的令牌格式，可以被用于身份认证和信息交换。

JWT由三部分组成：

1. **头部（Header）**：包含令牌的类型（typ）和所使用的签名算法（alg）
2. **载荷（Payload）**：包含声明（Claims），声明是关于实体和其他数据的声明
3. **签名（Signature）**：用于验证消息在传输过程中是否被篡改

这三部分通过点（.）连接在一起：

```
Header.Payload.Signature
```

示例：

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
```

### 2. JWT和Session有什么区别？各自的优缺点是什么？

JWT和Session的主要区别：

1. **存储位置**：
   - JWT：存储在客户端（localStorage、Cookie）
   - Session：存储在服务器端（内存、数据库、Redis）

2. **状态**：
   - JWT：无状态
   - Session：有状态

3. **服务器负担**：
   - JWT：低（无需存储Session）
   - Session：高（需要存储和维护Session）

4. **扩展性**：
   - JWT：好（支持分布式）
   - Session：差（需要Session共享机制）

5. **跨域支持**：
   - JWT：好（天然支持）
   - Session：差（需要特殊配置）

6. **安全性**：
   - JWT：中（无法主动失效）
   - Session：高（可以主动失效）

JWT的优点：
- 无状态，服务器负担小
- 支持跨域和分布式
- 适合移动应用和API服务
- 易于实现单点登录

JWT的缺点：
- 无法主动失效令牌
- 令牌较大，增加网络传输
- 载荷信息可被解码，不应存储敏感信息
- 令牌泄露后无法立即撤销

Session的优点：
- 可以主动失效
- 安全性更高
- 适合传统Web应用

Session的缺点：
- 服务器负担大
- 扩展性差
- 不支持跨域

### 3. JWT的工作原理是什么？

JWT的工作原理：

1. **用户登录**：用户提供用户名和密码
2. **服务器验证**：服务器验证用户凭证
3. **生成JWT**：服务器生成JWT，包含用户信息和过期时间
4. **签名JWT**：服务器使用密钥对JWT进行签名
5. **返回JWT**：服务器将JWT返回给客户端
6. **存储JWT**：客户端存储JWT（通常在localStorage或Cookie中）
7. **携带JWT**：客户端在后续请求中携带JWT（通常在Authorization头中）
8. **验证JWT**：服务器验证JWT签名和过期时间
9. **提取信息**：服务器从JWT中提取用户信息
10. **处理请求**：服务器处理请求并返回响应

代码示例：

```javascript
const jwt = require('jsonwebtoken');

const secret = 'your-secret-key';

app.post('/login', (req, res) => {
  const { username, password } = req.body;

  if (authenticate(username, password)) {
    const user = getUser(username);
    const token = jwt.sign(
      {
        userId: user.id,
        username: user.username,
        role: user.role
      },
      secret,
      { expiresIn: '1h' }
    );

    res.json({ token });
  }
});

function authenticateToken(req, res, next) {
  const authHeader = req.headers['authorization'];
  const token = authHeader && authHeader.split(' ')[1];

  if (!token) {
    return res.status(401).json({ error: '未提供访问令牌' });
  }

  jwt.verify(token, secret, (err, user) => {
    if (err) {
      return res.status(403).json({ error: '无效的访问令牌' });
    }

    req.user = user;
    next();
  });
}
```

### 4. JWT有哪些常见的签名算法？如何选择？

JWT常见的签名算法：

**对称加密算法**：
- **HS256**：HMAC SHA256
- **HS384**：HMAC SHA384
- **HS512**：HMAC SHA512

**非对称加密算法**：
- **RS256**：RSA SHA256
- **RS384**：RSA SHA384
- **RS512**：RSA SHA512
- **ES256**：ECDSA SHA256
- **ES384**：ECDSA SHA384
- **ES512**：ECDSA SHA512
- **PS256**：RSASSA-PSS SHA256

选择算法的考虑因素：

1. **对称加密（HS256）**：
   - 优点：性能好，实现简单
   - 缺点：密钥管理复杂，所有服务都需要共享密钥
   - 适用场景：单服务应用，信任的服务之间

2. **非对称加密（RS256）**：
   - 优点：密钥管理简单，私钥签名，公钥验证
   - 缺点：性能较差，实现复杂
   - 适用场景：多服务应用，第三方集成

代码示例：

```javascript
const jwt = require('jsonwebtoken');

// 对称加密
const token1 = jwt.sign(payload, secret, { algorithm: 'HS256' });
const decoded1 = jwt.verify(token1, secret, { algorithms: ['HS256'] });

// 非对称加密
const token2 = jwt.sign(payload, privateKey, { algorithm: 'RS256' });
const decoded2 = jwt.verify(token2, publicKey, { algorithms: ['RS256'] });
```

### 5. 如何防止JWT被伪造？

防止JWT被伪造的方法：

1. **使用强密钥**：
```javascript
const crypto = require('crypto');
const secret = crypto.randomBytes(64).toString('hex');
```

2. **使用非对称加密**：
```javascript
const token = jwt.sign(payload, privateKey, { algorithm: 'RS256' });
const decoded = jwt.verify(token, publicKey, { algorithms: ['RS256'] });
```

3. **验证算法**：
```javascript
const decoded = jwt.verify(token, secret, {
  algorithms: ['HS256', 'RS256']
});
```

4. **设置合理的过期时间**：
```javascript
const token = jwt.sign(payload, secret, {
  expiresIn: '15m'
});
```

5. **使用HTTPS**：确保所有通信都使用HTTPS加密

6. **实现令牌黑名单**：
```javascript
async function revokeToken(token) {
  const decoded = jwt.decode(token);
  const ttl = decoded.exp - Math.floor(Date.now() / 1000);
  await redis.setex(`blacklist:${decoded.jti}`, ttl, '1');
}
```

7. **绑定客户端信息**：
```javascript
const payload = {
  userId: user.id,
  ip: req.ip,
  userAgent: req.headers['user-agent']
};
```

### 6. JWT的载荷可以存储敏感信息吗？为什么？

JWT的载荷**不应该**存储敏感信息。原因如下：

1. **Base64编码不是加密**：JWT的载荷使用Base64Url编码，可以被任何人解码

```javascript
const jwt = require('jsonwebtoken');

const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c';

const decoded = jwt.decode(token);
console.log(decoded);
// 输出：{ sub: '1234567890', name: 'John Doe', iat: 1516239022 }
```

2. **令牌可能被窃取**：如果JWT存储在客户端，可能被XSS攻击窃取

3. **令牌可能被截获**：如果不使用HTTPS，令牌可能被网络嗅探截获

4. **令牌无法主动失效**：JWT一旦签发，在过期前一直有效

正确的做法：

```javascript
const payload = {
  userId: '123456',
  username: 'johndoe',
  role: 'admin'
};

const token = jwt.sign(payload, secret);
```

不要在载荷中存储：
- ❌ 密码
- ❌ 信用卡号
- ❌ 个人身份信息
- ❌ 敏感的业务数据

### 7. 如何实现JWT的刷新令牌机制？

刷新令牌机制使用短期访问令牌和长期刷新令牌：

```javascript
const jwt = require('jsonwebtoken');

const secret = 'your-secret-key';
const refreshSecret = 'your-refresh-secret';

app.post('/login', (req, res) => {
  const { username, password } = req.body;

  if (authenticate(username, password)) {
    const user = getUser(username);

    const accessToken = jwt.sign(
      {
        userId: user.id,
        username: user.username,
        role: user.role
      },
      secret,
      { expiresIn: '15m' }
    );

    const refreshToken = jwt.sign(
      { userId: user.id },
      refreshSecret,
      { expiresIn: '7d' }
    );

    res.json({
      accessToken,
      refreshToken
    });
  }
});

app.post('/refresh', (req, res) => {
  const { refreshToken } = req.body;

  try {
    const decoded = jwt.verify(refreshToken, refreshSecret);
    const user = getUserById(decoded.userId);

    const newAccessToken = jwt.sign(
      {
        userId: user.id,
        username: user.username,
        role: user.role
      },
      secret,
      { expiresIn: '15m' }
    );

    res.json({ accessToken: newAccessToken });
  } catch (err) {
    res.status(401).json({ error: '无效的刷新令牌' });
  }
});
```

刷新令牌的最佳实践：

1. **访问令牌过期时间短**：通常15-30分钟
2. **刷新令牌过期时间长**：通常7-30天
3. **刷新令牌只使用一次**：使用后生成新的刷新令牌
4. **存储刷新令牌**：将刷新令牌存储在数据库中，可以主动撤销
5. **使用不同的密钥**：访问令牌和刷新令牌使用不同的密钥

### 8. 如何实现JWT的主动失效？

JWT本身无法主动失效，但可以通过以下方法实现：

1. **令牌黑名单**：
```javascript
const redis = require('redis');
const client = redis.createClient();

async function revokeToken(token) {
  const decoded = jwt.decode(token);
  const ttl = decoded.exp - Math.floor(Date.now() / 1000);
  await client.setex(`blacklist:${decoded.jti}`, ttl, '1');
}

async function isTokenBlacklisted(jti) {
  const result = await client.get(`blacklist:${jti}`);
  return result !== null;
}

app.use(authenticateToken, async (req, res, next) => {
  if (await isTokenBlacklisted(req.user.jti)) {
    return res.status(401).json({ error: '令牌已被撤销' });
  }
  next();
});
```

2. **版本号机制**：
```javascript
const payload = {
  userId: user.id,
  version: user.tokenVersion
};

const token = jwt.sign(payload, secret);

app.use(authenticateToken, async (req, res, next) => {
  const user = await getUserById(req.user.userId);
  if (user.tokenVersion !== req.user.version) {
    return res.status(401).json({ error: '令牌已失效' });
  }
  next();
});

async function revokeAllTokens(userId) {
  const user = await getUserById(userId);
  user.tokenVersion += 1;
  await saveUser(user);
}
```

3. **短期令牌 + 刷新令牌**：
```javascript
const accessToken = jwt.sign(payload, secret, {
  expiresIn: '15m'
});

const refreshToken = jwt.sign(payload, refreshSecret, {
  expiresIn: '7d'
});
```

4. **绑定客户端信息**：
```javascript
const payload = {
  userId: user.id,
  ip: req.ip,
  userAgent: req.headers['user-agent']
};

app.use(authenticateToken, (req, res, next) => {
  if (req.user.ip !== req.ip || req.user.userAgent !== req.headers['user-agent']) {
    return res.status(401).json({ error: '令牌已失效' });
  }
  next();
});
```

### 9. JWT有哪些常见的安全漏洞？如何防范？

JWT常见的安全漏洞及防范措施：

1. **算法混淆攻击**：
   - 原理：攻击者将算法从非对称加密改为对称加密
   - 防范：明确指定允许的算法
   ```javascript
   const decoded = jwt.verify(token, secret, {
     algorithms: ['RS256']
   });
   ```

2. **空算法攻击**：
   - 原理：攻击者将算法设置为"none"
   - 防范：明确指定允许的算法
   ```javascript
   const decoded = jwt.verify(token, secret, {
     algorithms: ['HS256', 'RS256']
   });
   ```

3. **密钥泄露**：
   - 原理：签名密钥泄露
   - 防范：使用环境变量，定期更换密钥
   ```javascript
   const secret = process.env.JWT_SECRET;
   ```

4. **令牌泄露**：
   - 原理：JWT被XSS或网络嗅探窃取
   - 防范：使用HttpOnly Cookie，使用HTTPS
   ```javascript
   res.cookie('jwt', token, {
     httpOnly: true,
     secure: true,
     sameSite: 'strict'
   });
   ```

5. **令牌重放攻击**：
   - 原理：攻击者重复使用有效的JWT
   - 防范：实现令牌黑名单，绑定客户端信息
   ```javascript
   const payload = {
     userId: user.id,
     jti: generateUniqueId(),
     ip: req.ip
   };
   ```

6. **时间攻击**：
   - 原理：攻击者通过测量验证时间推断密钥
   - 防范：使用恒定时间比较
   ```javascript
   const crypto = require('crypto');
   crypto.timingSafeEqual(Buffer.from(a), Buffer.from(b));
   ```

7. **信息泄露**：
   - 原理：在载荷中存储敏感信息
   - 防范：不在载荷中存储敏感信息
   ```javascript
   const payload = {
     userId: user.id,
     username: user.username
   };
   ```

### 10. JWT适用于哪些场景？不适用于哪些场景？

JWT适用的场景：

1. **微服务架构**：多个服务之间需要共享用户身份信息
2. **移动应用**：移动应用对Cookie支持有限
3. **单页应用（SPA）**：前后端分离，需要跨域访问
4. **API服务**：RESTful API需要无状态认证
5. **单点登录（SSO）**：多个应用之间统一认证
6. **第三方集成**：需要与外部系统进行身份验证
7. **分布式系统**：需要在多个服务之间传递用户信息

JWT不适用的场景：

1. **需要主动失效的场景**：如管理员撤销用户权限
2. **敏感操作**：如支付、转账等需要严格会话控制
3. **短期会话**：会话时间很短，不需要持久化
4. **传统Web应用**：前后端不分离，使用Cookie更合适
5. **需要频繁更新的数据**：如购物车、临时状态等

### 11. 如何在微服务架构中使用JWT？

在微服务架构中使用JWT的方法：

1. **统一认证服务**：
```javascript
const authServer = {
  async login(username, password) {
    const user = await authenticate(username, password);
    const token = jwt.sign(
      {
        userId: user.id,
        username: user.username,
        role: user.role
      },
      secret,
      { expiresIn: '1h' }
    );
    return token;
  }
};
```

2. **服务间通信**：
```javascript
const axios = require('axios');

async function callUserService(token) {
  const response = await axios.get('http://user-service/api/users', {
    headers: {
      Authorization: `Bearer ${token}`
    }
  });
  return response.data;
}
```

3. **共享公钥**：
```javascript
const publicKey = fs.readFileSync('public.key');

app.use((req, res, next) => {
  const token = req.headers['authorization']?.split(' ')[1];
  if (!token) {
    return res.status(401).json({ error: '未提供访问令牌' });
  }

  try {
    const decoded = jwt.verify(token, publicKey, {
      algorithms: ['RS256']
    });
    req.user = decoded;
    next();
  } catch (err) {
    res.status(403).json({ error: '无效的访问令牌' });
  }
});
```

4. **服务网关**：
```javascript
const gateway = express();

gateway.use(authenticateToken);

gateway.use('/user-service', proxy('http://user-service'));
gateway.use('/order-service', proxy('http://order-service'));
gateway.use('/payment-service', proxy('http://payment-service'));
```

5. **权限控制**：
```javascript
function authorize(roles) {
  return (req, res, next) => {
    if (!roles.includes(req.user.role)) {
      return res.status(403).json({ error: '权限不足' });
    }
    next();
  };
}

app.get('/admin', authenticateToken, authorize(['admin']), (req, res) => {
  res.json({ message: '管理员页面' });
});
```

### 12. JWT的标准声明有哪些？各自的作用是什么？

JWT的标准声明（Registered Claims）：

1. **iss（issuer）**：签发人
   - 作用：标识JWT的签发者
   - 示例：`"iss": "example.com"`

2. **sub（subject）**：主题
   - 作用：标识JWT的主题（通常是用户ID）
   - 示例：`"sub": "1234567890"`

3. **aud（audience）**：接收方
   - 作用：标识JWT的预期接收者
   - 示例：`"aud": "example-api"`

4. **exp（expiration time）**：过期时间
   - 作用：指定JWT的过期时间（Unix时间戳）
   - 示例：`"exp": 1516242622`

5. **nbf（not before）**：生效时间
   - 作用：指定JWT的生效时间（Unix时间戳）
   - 示例：`"nbf": 1516239022`

6. **iat（issued at）**：签发时间
   - 作用：指定JWT的签发时间（Unix时间戳）
   - 示例：`"iat": 1516239022`

7. **jti（JWT ID）**：JWT的唯一标识
   - 作用：唯一标识JWT，可用于防止重放攻击
   - 示例：`"jti": "unique-id"`

代码示例：

```javascript
const jwt = require('jsonwebtoken');

const payload = {
  sub: '1234567890',
  name: 'John Doe',
  iat: Math.floor(Date.now() / 1000),
  exp: Math.floor(Date.now() / 1000) + 3600,
  nbf: Math.floor(Date.now() / 1000),
  iss: 'example.com',
  aud: 'example-api',
  jti: generateUniqueId()
};

const token = jwt.sign(payload, secret);

const decoded = jwt.verify(token, secret, {
  issuer: 'example.com',
  audience: 'example-api'
});
```

### 13. 如何在前后端分离的架构中使用JWT？

在前后端分离的架构中使用JWT的方法：

1. **登录接口**：
```javascript
app.post('/api/login', (req, res) => {
  const { username, password } = req.body;

  if (authenticate(username, password)) {
    const user = getUser(username);
    const token = jwt.sign(
      {
        userId: user.id,
        username: user.username,
        role: user.role
      },
      secret,
      { expiresIn: '1h' }
    );

    res.json({
      token,
      user: {
        id: user.id,
        username: user.username,
        role: user.role
      }
    });
  } else {
    res.status(401).json({ error: '认证失败' });
  }
});
```

2. **前端存储JWT**：
```javascript
// 登录
async function login(username, password) {
  const response = await fetch('/api/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ username, password })
  });

  const data = await response.json();
  localStorage.setItem('token', data.token);
  localStorage.setItem('user', JSON.stringify(data.user));
}

// 请求拦截器
axios.interceptors.request.use(config => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});
```

3. **后端验证JWT**：
```javascript
function authenticateToken(req, res, next) {
  const authHeader = req.headers['authorization'];
  const token = authHeader && authHeader.split(' ')[1];

  if (!token) {
    return res.status(401).json({ error: '未提供访问令牌' });
  }

  jwt.verify(token, secret, (err, user) => {
    if (err) {
      return res.status(403).json({ error: '无效的访问令牌' });
    }

    req.user = user;
    next();
  });
}

app.get('/api/user', authenticateToken, (req, res) => {
  res.json({ user: req.user });
});
```

4. **响应拦截器**：
```javascript
axios.interceptors.response.use(
  response => response,
  error => {
    if (error.response.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
```

5. **刷新令牌**：
```javascript
async function refreshToken() {
  const refreshToken = localStorage.getItem('refreshToken');
  const response = await fetch('/api/refresh', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ refreshToken })
  });

  const data = await response.json();
  localStorage.setItem('token', data.accessToken);
  return data.accessToken;
}
```

### 14. JWT的过期时间应该如何设置？

JWT过期时间的设置原则：

1. **访问令牌**：
   - 时间：15-30分钟
   - 原因：平衡安全性和用户体验
   ```javascript
   const accessToken = jwt.sign(payload, secret, {
     expiresIn: '15m'
   });
   ```

2. **刷新令牌**：
   - 时间：7-30天
   - 原因：减少用户频繁登录
   ```javascript
   const refreshToken = jwt.sign(payload, refreshSecret, {
     expiresIn: '7d'
   });
   ```

3. **记住我令牌**：
   - 时间：30天或更长
   - 原因：用户选择"记住我"功能
   ```javascript
   const rememberMeToken = jwt.sign(payload, secret, {
     expiresIn: '30d'
   });
   ```

4. **邮件验证令牌**：
   - 时间：15-60分钟
   - 原因：邮件验证需要一定时间
   ```javascript
   const emailToken = jwt.sign(payload, secret, {
     expiresIn: '30m'
   });
   ```

5. **密码重置令牌**：
   - 时间：15-60分钟
   - 原因：密码重置需要一定时间
   ```javascript
   const resetToken = jwt.sign(payload, secret, {
     expiresIn: '30m'
   });
   ```

过期时间的考虑因素：

- **安全性**：过期时间越短，安全性越高
- **用户体验**：过期时间越长，用户体验越好
- **应用场景**：不同场景需要不同的过期时间
- **刷新机制**：有刷新机制可以使用短期令牌
- **敏感度**：敏感操作使用更短的过期时间

### 15. 如何在移动应用中使用JWT？

在移动应用中使用JWT的方法：

1. **登录获取JWT**：
```javascript
async function login(username, password) {
  const response = await fetch('https://api.example.com/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ username, password })
  });

  const data = await response.json();
  
  await AsyncStorage.setItem('accessToken', data.accessToken);
  await AsyncStorage.setItem('refreshToken', data.refreshToken);
  
  return data.user;
}
```

2. **请求携带JWT**：
```javascript
async function apiCall(url, options = {}) {
  const accessToken = await AsyncStorage.getItem('accessToken');
  
  const response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      'Authorization': `Bearer ${accessToken}`
    }
  });

  if (response.status === 401) {
    const newToken = await refreshToken();
    return fetch(url, {
      ...options,
      headers: {
        ...options.headers,
        'Authorization': `Bearer ${newToken}`
      }
    });
  }

  return response;
}
```

3. **刷新令牌**：
```javascript
async function refreshToken() {
  const refreshToken = await AsyncStorage.getItem('refreshToken');
  
  const response = await fetch('https://api.example.com/refresh', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ refreshToken })
  });

  const data = await response.json();
  
  await AsyncStorage.setItem('accessToken', data.accessToken);
  
  return data.accessToken;
}
```

4. **登出**：
```javascript
async function logout() {
  const accessToken = await AsyncStorage.getItem('accessToken');
  
  await fetch('https://api.example.com/logout', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${accessToken}`
    }
  });

  await AsyncStorage.removeItem('accessToken');
  await AsyncStorage.removeItem('refreshToken');
}
```

5. **安全考虑**：
- 使用HTTPS
- 不要在URL中传递JWT
- 使用短期访问令牌
- 实现刷新令牌机制
- 绑定设备信息
- 实现令牌黑名单

## 总结

JWT是一种强大且灵活的令牌格式，适用于现代Web应用和API服务。它具有无状态、跨域支持、易于扩展等优点，但也存在无法主动失效、令牌较大等缺点。在使用JWT时，需要注意安全问题，选择合适的签名算法，设置合理的过期时间，并实现适当的防护措施。JWT特别适合微服务架构、移动应用和单页应用等场景，但在需要主动失效或敏感操作的场景中，Session可能是更好的选择。

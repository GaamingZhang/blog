---
date: 2025-12-26
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Network
  - Web
tag:
  - Network
  - Cookie
  - HTTP
  - Web安全
  - 还在施工中
---

# Cookie基本概念

## 概述
Cookie是HTTP协议中的一种状态管理机制，用于在客户端和服务器之间存储和传递信息。它是由服务器发送给客户端的小型文本数据，客户端在后续请求中会将这些数据发送回服务器，从而实现会话保持、用户识别、个性化设置等功能。

### 产生背景
HTTP协议本身是无状态的，每个请求都是独立的，服务器无法识别请求是否来自同一个客户端。为了解决这个问题，Netscape在1994年引入了Cookie机制，使得服务器能够识别和跟踪用户会话。

### 核心价值
- **会话管理**：保持用户登录状态，实现会话持久化
- **个性化定制**：存储用户偏好设置，如语言、主题等
- **行为追踪**：记录用户行为，用于分析和优化
- **购物车功能**：在电商网站中保存购物车信息
- **广告追踪**：跨站点追踪用户行为，用于精准广告投放

## 工作原理

### Cookie的创建与传递
1. **服务器创建Cookie**：服务器通过HTTP响应头`Set-Cookie`发送Cookie给客户端
2. **客户端存储Cookie**：浏览器接收到Cookie后，将其存储在本地
3. **客户端发送Cookie**：在后续请求中，浏览器通过HTTP请求头`Cookie`将Cookie发送回服务器
4. **服务器读取Cookie**：服务器解析请求头中的Cookie，获取存储的信息

### HTTP头示例

**服务器设置Cookie**：
```
HTTP/1.1 200 OK
Set-Cookie: sessionid=abc123; Path=/; Domain=.example.com; HttpOnly; Secure; SameSite=Strict
```

**客户端发送Cookie**：
```
GET /api/user HTTP/1.1
Host: example.com
Cookie: sessionid=abc123; theme=dark
```

## Cookie的属性

### 基本属性

#### 1. Name和Value
- **Name**：Cookie的名称，用于标识Cookie
- **Value**：Cookie的值，存储实际数据
- **注意**：名称和值不能包含分号、逗号、空格等特殊字符，需要使用URL编码

#### 2. Domain
- **作用**：指定Cookie所属的域名
- **默认值**：当前文档的域名
- **子域名**：设置为`.example.com`时，`www.example.com`和`api.example.com`都可以访问该Cookie
- **示例**：`Domain=.example.com`

#### 3. Path
- **作用**：指定Cookie适用的路径
- **默认值**：当前文档的路径
- **匹配规则**：只有请求的URL路径匹配或包含该路径时，才会发送Cookie
- **示例**：`Path=/api`表示只有`/api`路径下的请求才会发送该Cookie

#### 4. Expires和Max-Age
- **Expires**：指定Cookie的过期时间（绝对时间）
  - 格式：`Wdy, DD Mon YYYY HH:MM:SS GMT`
  - 示例：`Expires=Wed, 21 Oct 2025 07:28:00 GMT`
- **Max-Age**：指定Cookie的有效期（相对时间，单位：秒）
  - 示例：`Max-Age=3600`表示Cookie在1小时后过期
- **优先级**：Max-Age优先于Expires
- **会话Cookie**：不设置Expires和Max-Age时，Cookie在浏览器关闭后失效

#### 5. Secure
- **作用**：指示Cookie只能通过HTTPS协议传输
- **安全性**：防止Cookie在HTTP连接中被窃取
- **建议**：所有涉及敏感信息的Cookie都应设置Secure属性

#### 6. HttpOnly
- **作用**：禁止JavaScript通过`document.cookie`访问Cookie
- **安全性**：防止XSS攻击窃取Cookie
- **建议**：所有会话Cookie都应设置HttpOnly属性

#### 7. SameSite
- **作用**：控制Cookie在跨站点请求中的发送行为
- **值**：
  - `Strict`：严格模式，仅在当前站点请求中发送Cookie
  - `Lax`：宽松模式，允许部分跨站点请求发送Cookie（如导航到站点的GET请求）
  - `None`：允许所有跨站点请求发送Cookie，必须配合Secure属性使用
- **示例**：`SameSite=Strict`

## Cookie的类型

### 按生命周期分类
- **会话Cookie**：浏览器关闭后失效
- **持久Cookie**：设置了Expires或Max-Age，在指定时间后失效

### 按作用域分类
- **第一方Cookie**：由当前访问的域名设置
- **第三方Cookie**：由其他域名设置（如广告追踪Cookie）

### 按安全性分类
- **安全Cookie**：设置了Secure和HttpOnly属性
- **普通Cookie**：未设置安全属性

## Cookie的存储限制

### 大小限制
- **单个Cookie大小**：通常限制为4KB
- **每个域名的Cookie数量**：通常限制为20-50个
- **浏览器总Cookie数量**：通常限制为300-500个

### 浏览器差异
不同浏览器对Cookie的限制略有不同，开发者应遵循保守策略：
- 单个Cookie不超过3KB
- 每个域名不超过20个Cookie

## Cookie的安全性问题

### 1. XSS攻击
- **原理**：攻击者通过注入恶意脚本窃取Cookie
- **防护**：设置HttpOnly属性，防止JavaScript访问Cookie

### 2. CSRF攻击
- **原理**：攻击者利用用户的登录状态执行未授权操作
- **防护**：设置SameSite属性，使用CSRF Token

### 3. 中间人攻击
- **原理**：攻击者拦截HTTP请求，窃取Cookie
- **防护**：设置Secure属性，强制使用HTTPS

### 4. Cookie劫持
- **原理**：攻击者通过XSS或网络嗅探获取Cookie
- **防护**：设置HttpOnly和Secure属性，定期更新Session ID

### 5. 固定会话攻击
- **原理**：攻击者预先设置Session ID，诱导用户使用该ID登录
- **防护**：登录后重新生成Session ID

## Cookie与Session的关系

### Session的工作原理
1. 用户首次访问时，服务器创建Session，生成唯一的Session ID
2. 服务器将Session ID通过Cookie发送给客户端
3. 客户端在后续请求中携带Session ID
4. 服务器根据Session ID查找对应的Session数据

### Cookie与Session的区别
- **存储位置**：Cookie存储在客户端，Session存储在服务器端
- **安全性**：Session更安全，数据不暴露给客户端
- **大小限制**：Cookie有大小限制，Session无限制
- **性能**：Cookie减少服务器压力，Session增加服务器压力

### Session的替代方案
- **Redis存储**：将Session存储在Redis中，支持分布式部署
- **JWT（JSON Web Token）**：将用户信息编码到Token中，无需服务器存储

## Cookie与其他存储方式的对比

### Cookie vs LocalStorage
| 特性 | Cookie | LocalStorage |
|------|--------|--------------|
| 大小限制 | 4KB | 5-10MB |
| 过期时间 | 可设置 | 永久存储 |
| 请求发送 | 自动发送到服务器 | 不发送 |
| 安全性 | 支持HttpOnly、Secure | 不支持 |
| 用途 | 会话管理、状态保持 | 本地数据存储 |

### Cookie vs SessionStorage
| 特性 | Cookie | SessionStorage |
|------|--------|----------------|
| 大小限制 | 4KB | 5-10MB |
| 过期时间 | 可设置 | 浏览器标签页关闭后失效 |
| 请求发送 | 自动发送到服务器 | 不发送 |
| 作用域 | 同域名 | 同标签页 |
| 用途 | 会话管理 | 临时数据存储 |

### Cookie vs IndexedDB
| 特性 | Cookie | IndexedDB |
|------|--------|-----------|
| 大小限制 | 4KB | 数百MB甚至更多 |
| 数据类型 | 字符串 | 结构化数据 |
| 同步/异步 | 同步 | 异步 |
| 查询能力 | 无 | 支持索引查询 |
| 用途 | 会话管理 | 复杂数据存储 |

## Cookie的最佳实践

### 1. 安全性
- 为敏感Cookie设置HttpOnly和Secure属性
- 使用SameSite=Strict或SameSite=Lax防止CSRF攻击
- 定期更新Session ID，防止会话固定攻击
- 避免在Cookie中存储敏感信息

### 2. 性能优化
- 减少Cookie的大小，减少网络传输开销
- 减少Cookie的数量，避免超过浏览器限制
- 为不同子域名设置不同的Cookie，避免不必要的传输
- 使用Path属性限制Cookie的作用范围

### 3. 隐私保护
- 提供Cookie管理界面，允许用户删除Cookie
- 遵守GDPR等隐私法规，告知用户Cookie的使用情况
- 使用第一方Cookie，减少第三方Cookie的使用
- 设置合理的过期时间，避免长期追踪

### 4. 开发规范
- 为Cookie设置明确的Domain和Path
- 使用URL编码处理特殊字符
- 为Cookie设置合理的过期时间
- 记录Cookie的使用目的和生命周期

## Cookie的JavaScript操作

### 设置Cookie
```javascript
function setCookie(name, value, days) {
  const expires = new Date();
  expires.setTime(expires.getTime() + days * 24 * 60 * 60 * 1000);
  document.cookie = `${name}=${value};expires=${expires.toUTCString()};path=/`;
}

setCookie('username', 'john', 7);
```

### 获取Cookie
```javascript
function getCookie(name) {
  const cookies = document.cookie.split(';');
  for (let cookie of cookies) {
    const [key, value] = cookie.trim().split('=');
    if (key === name) {
      return decodeURIComponent(value);
    }
  }
  return null;
}

const username = getCookie('username');
```

### 删除Cookie
```javascript
function deleteCookie(name) {
  document.cookie = `${name}=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/`;
}

deleteCookie('username');
```

## 相关高频面试题与简答

- 问：什么是Cookie？它解决了什么问题？
  答：Cookie是HTTP协议中的一种状态管理机制，用于在客户端和服务器之间存储和传递信息。它解决了HTTP协议无状态的问题，使得服务器能够识别和跟踪用户会话，实现会话管理、用户识别、个性化设置等功能。

- 问：Cookie的工作原理是什么？
  答：Cookie的工作原理包括四个步骤：
  1）服务器通过HTTP响应头`Set-Cookie`发送Cookie给客户端
  2）浏览器接收到Cookie后，将其存储在本地
  3）在后续请求中，浏览器通过HTTP请求头`Cookie`将Cookie发送回服务器
  4）服务器解析请求头中的Cookie，获取存储的信息

- 问：Cookie的HttpOnly和Secure属性有什么作用？
  答：HttpOnly属性禁止JavaScript通过`document.cookie`访问Cookie，防止XSS攻击窃取Cookie；Secure属性指示Cookie只能通过HTTPS协议传输，防止Cookie在HTTP连接中被窃取。建议所有涉及敏感信息的Cookie都应设置这两个属性。

- 问：Cookie的SameSite属性有哪些值？各有什么作用？
  答：SameSite属性有三个值：
  - `Strict`：严格模式，仅在当前站点请求中发送Cookie
  - `Lax`：宽松模式，允许部分跨站点请求发送Cookie（如导航到站点的GET请求）
  - `None`：允许所有跨站点请求发送Cookie，必须配合Secure属性使用
  SameSite属性主要用于防止CSRF攻击。

- 问：Cookie和Session的区别是什么？
  答：Cookie和Session的主要区别：
  - **存储位置**：Cookie存储在客户端，Session存储在服务器端
  - **安全性**：Session更安全，数据不暴露给客户端
  - **大小限制**：Cookie有大小限制（通常4KB），Session无限制
  - **性能**：Cookie减少服务器压力，Session增加服务器压力
  - **生命周期**：Cookie可以设置长期有效，Session通常在浏览器关闭后失效

- 问：Cookie、LocalStorage和SessionStorage的区别是什么？
  答：三者主要区别：
  - **Cookie**：大小限制4KB，自动发送到服务器，支持HttpOnly和Secure属性，用于会话管理
  - **LocalStorage**：大小限制5-10MB，永久存储，不发送到服务器，用于本地数据存储
  - **SessionStorage**：大小限制5-10MB，浏览器标签页关闭后失效，不发送到服务器，用于临时数据存储

- 问：如何防止Cookie被XSS攻击窃取？
  答：防止Cookie被XSS攻击窃取的方法：
  - 设置HttpOnly属性，禁止JavaScript访问Cookie
  - 设置Secure属性，强制使用HTTPS传输
  - 避免在Cookie中存储敏感信息
  - 对用户输入进行严格的过滤和转义
  - 使用Content Security Policy（CSP）限制脚本执行

- 问：如何防止CSRF攻击？
  答：防止CSRF攻击的方法：
  - 设置SameSite属性为Strict或Lax
  - 使用CSRF Token，在请求中携带随机生成的Token
  - 验证Referer或Origin头
  - 使用双重提交Cookie
  - 对于重要操作，要求用户重新输入密码或进行二次确认

- 问：Cookie的大小限制是多少？如何优化Cookie的使用？
  答：Cookie的大小限制：
  - 单个Cookie通常限制为4KB
  - 每个域名的Cookie数量通常限制为20-50个
  - 浏览器总Cookie数量通常限制为300-500个

  优化Cookie使用的方法：
  - 减少Cookie的大小，只存储必要的信息
  - 减少Cookie的数量，避免超过浏览器限制
  - 使用Path属性限制Cookie的作用范围
  - 为不同子域名设置不同的Cookie
  - 定期清理过期的Cookie

- 问：什么是第一方Cookie和第三方Cookie？有什么区别？
  答：第一方Cookie是由当前访问的域名设置的Cookie，主要用于会话管理、用户识别等功能；第三方Cookie是由其他域名设置的Cookie，主要用于广告追踪、行为分析等功能。区别在于设置Cookie的域名是否与当前访问的域名相同。由于隐私保护的需求，现代浏览器对第三方Cookie的限制越来越严格。

- 问：如何在JavaScript中设置、获取和删除Cookie？
  答：在JavaScript中操作Cookie的方法：
  - **设置Cookie**：`document.cookie = 'name=value;expires=date;path=/;domain=.example.com'`
  - **获取Cookie**：解析`document.cookie`字符串，按分号分割，查找指定的Cookie
  - **删除Cookie**：设置Cookie的过期时间为过去的时间，如`document.cookie = 'name=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/'`

- 问：Cookie的Expires和Max-Age属性有什么区别？
  答：Expires和Max-Age都用于设置Cookie的过期时间，但区别在于：
  - **Expires**：指定Cookie的过期时间（绝对时间），格式为`Wdy, DD Mon YYYY HH:MM:SS GMT`
  - **Max-Age**：指定Cookie的有效期（相对时间，单位：秒）
  - **优先级**：Max-Age优先于Expires
  - **兼容性**：Expires是旧属性，Max-Age是较新的属性，建议使用Max-Age

- 问：什么是会话固定攻击？如何防护？
  答：会话固定攻击是一种攻击方式，攻击者预先设置Session ID，诱导用户使用该ID登录，从而获取用户的会话权限。防护方法：
  - 登录后重新生成Session ID
  - 在登录前检查是否存在Session，如果存在则销毁
  - 设置合理的Session过期时间
  - 使用HTTPS防止Session ID被窃取
  - 在Session中绑定用户代理信息，检测异常访问

- 问：Cookie在跨域请求中如何传递？
  答：Cookie在跨域请求中的传递规则：
  - 默认情况下，浏览器不会在跨域请求中发送Cookie
  - 服务器需要设置`Access-Control-Allow-Credentials: true`响应头
  - 前端请求需要设置`withCredentials: true`
  - Cookie的SameSite属性不能设置为Strict
  - 如果SameSite设置为None，必须同时设置Secure属性

- 问：什么是JWT？它与Cookie有什么关系？
  答：JWT（JSON Web Token）是一种开放标准（RFC 7519），用于在各方之间安全地传输信息。JWT与Cookie的关系：
  - JWT通常存储在Cookie中，通过Cookie传递给服务器
  - JWT也可以存储在LocalStorage或SessionStorage中，通过Authorization头传递
  - 使用JWT时，服务器不需要存储Session数据，减轻服务器压力
  - JWT包含用户信息，需要加密签名，防止篡改
  - JWT的过期时间由Token本身决定，不依赖Cookie的过期时间

## 总结
Cookie是Web开发中重要的状态管理机制，它解决了HTTP协议无状态的问题，为会话管理、用户识别、个性化设置等功能提供了基础。通过理解Cookie的工作原理、属性配置、安全问题和最佳实践，开发者可以更好地使用Cookie，构建安全、高效的Web应用。

随着Web技术的发展，Cookie的使用也在不断演进。现代浏览器对第三方Cookie的限制越来越严格，新的技术如JWT、LocalStorage等也在不断涌现。开发者应根据实际需求选择合适的状态管理方案，在安全性、性能和用户体验之间找到平衡。

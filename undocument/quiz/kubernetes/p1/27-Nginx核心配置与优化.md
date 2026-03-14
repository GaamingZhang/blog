---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Web服务器
tag:
  - Nginx
  - 性能优化
---

# Nginx核心配置参数及优化建议

## Nginx架构概述

```
┌─────────────────────────────────────────────────────────────┐
│                     Nginx架构                                │
│                                                              │
│                    ┌─────────────┐                          │
│                    │   Master    │                          │
│                    │   Process   │                          │
│                    └──────┬──────┘                          │
│                           │ 管理                            │
│         ┌─────────────────┼─────────────────┐              │
│         ↓                 ↓                 ↓              │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐      │
│  │  Worker 1   │   │  Worker 2   │   │  Worker N   │      │
│  │  Process    │   │  Process    │   │  Process    │      │
│  └─────────────┘   └─────────────┘   └─────────────┘      │
│         │                 │                 │              │
│         └─────────────────┼─────────────────┘              │
│                           ↓                                 │
│                    ┌─────────────┐                          │
│                    │  连接池     │                          │
│                    │  事件驱动   │                          │
│                    └─────────────┘                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 核心配置参数

### 全局配置（main）

```nginx
user nginx;                    # 运行用户
worker_processes auto;         # worker进程数，auto=CPU核心数
worker_rlimit_nofile 65535;    # 每个worker最大打开文件数
error_log /var/log/nginx/error.log warn;  # 错误日志级别
pid /var/run/nginx.pid;        # PID文件位置

events {
    worker_connections 65535;  # 每个worker最大连接数
    use epoll;                 # 事件模型（Linux使用epoll）
    multi_accept on;           # 一次接受多个连接
}
```

### HTTP配置

```nginx
http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;
    
    # 日志格式
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';
    
    access_log /var/log/nginx/access.log main;
    
    # 性能优化
    sendfile on;                    # 使用sendfile系统调用
    tcp_nopush on;                  # 优化数据包发送
    tcp_nodelay on;                 # 禁用Nagle算法
    keepalive_timeout 65;           # 长连接超时
    types_hash_max_size 2048;       # 类型哈希表大小
    
    # 隐藏版本号
    server_tokens off;
    
    # Gzip压缩
    gzip on;
    gzip_vary on;
    gzip_min_length 1k;
    gzip_comp_level 6;
    gzip_types text/plain text/css application/json application/javascript;
    
    # 包含虚拟主机配置
    include /etc/nginx/conf.d/*.conf;
}
```

### Server配置

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name example.com www.example.com;
    root /var/www/html;
    index index.html index.htm;
    
    # 访问日志
    access_log /var/log/nginx/example.access.log main;
    error_log /var/log/nginx/example.error.log;
    
    # SSL配置
    listen 443 ssl http2;
    ssl_certificate /etc/nginx/ssl/example.crt;
    ssl_certificate_key /etc/nginx/ssl/example.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000" always;
    
    location / {
        try_files $uri $uri/ =404;
    }
    
    # 反向代理
    location /api/ {
        proxy_pass http://backend:8080/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # 静态文件缓存
    location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
        expires 30d;
        add_header Cache-Control "public, no-transform";
    }
    
    # 禁止访问隐藏文件
    location ~ /\. {
        deny all;
    }
}
```

## 性能优化参数

### Worker进程优化

```nginx
worker_processes auto;           # 自动检测CPU核心数
worker_cpu_affinity auto;        # 自动绑定CPU核心
worker_rlimit_nofile 100000;     # 最大打开文件数
```

**计算公式**：
- worker_processes = CPU核心数
- 最大并发连接数 = worker_processes × worker_connections

### 连接优化

```nginx
events {
    worker_connections 65535;    # 每个worker最大连接数
    use epoll;                   # Linux使用epoll
    multi_accept on;             # 一次接受多个连接
    accept_mutex off;            # 高并发时关闭accept互斥锁
}
```

### 缓冲区优化

```nginx
http {
    # 客户端请求体缓冲
    client_body_buffer_size 16k;
    client_header_buffer_size 1k;
    client_max_body_size 8m;
    large_client_header_buffers 4 8k;
    
    # 输出缓冲
    output_buffers 1 32k;
    postpone_output 1460;
    
    # FastCGI缓冲
    fastcgi_buffer_size 64k;
    fastcgi_buffers 4 64k;
    fastcgi_busy_buffers_size 128k;
    
    # Proxy缓冲
    proxy_buffer_size 4k;
    proxy_buffers 4 32k;
    proxy_busy_buffers_size 64k;
}
```

### 超时优化

```nginx
http {
    client_body_timeout 12;      # 请求体超时
    client_header_timeout 12;    # 请求头超时
    send_timeout 10;             # 响应超时
    
    # 长连接
    keepalive_timeout 65;
    keepalive_requests 1000;
    
    # 上游超时
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;
}
```

### Gzip压缩优化

```nginx
http {
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_min_length 1k;
    gzip_buffers 16 8k;
    gzip_http_version 1.1;
    gzip_types
        text/plain
        text/css
        text/xml
        text/javascript
        application/json
        application/javascript
        application/xml
        application/xml+rss
        application/x-javascript;
    gzip_disable "msie6";
}
```

## 负载均衡配置

### 负载均衡算法

```nginx
upstream backend {
    # 轮询（默认）
    server 192.168.1.1:8080;
    server 192.168.1.2:8080;
    
    # 权重
    server 192.168.1.3:8080 weight=3;
    server 192.168.1.4:8080 weight=1;
    
    # IP哈希
    ip_hash;
    
    # 最少连接
    least_conn;
    
    # 健康检查
    server 192.168.1.5:8080 max_fails=3 fail_timeout=30s;
    server 192.168.1.6:8080 backup;  # 备用服务器
    server 192.168.1.7:8080 down;    # 下线服务器
}
```

### 负载均衡示例

```nginx
upstream backend {
    least_conn;
    server 192.168.1.1:8080 weight=5 max_fails=3 fail_timeout=30s;
    server 192.168.1.2:8080 weight=5 max_fails=3 fail_timeout=30s;
    server 192.168.1.3:8080 backup;
    
    keepalive 32;  # 保持长连接数
}

server {
    listen 80;
    server_name api.example.com;
    
    location / {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## 缓存配置

### 静态文件缓存

```nginx
server {
    location ~* \.(jpg|jpeg|png|gif|ico|css|js|pdf|txt|woff)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
        access_log off;
    }
}
```

### 代理缓存

```nginx
http {
    proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=my_cache:10m 
                     max_size=10g inactive=60m use_temp_path=off;
    
    server {
        location / {
            proxy_cache my_cache;
            proxy_cache_valid 200 302 10m;
            proxy_cache_valid 404 1m;
            proxy_cache_use_stale error timeout updating http_500;
            proxy_cache_background_update on;
            proxy_cache_lock on;
            
            add_header X-Cache-Status $upstream_cache_status;
            proxy_pass http://backend;
        }
    }
}
```

## 安全配置

### 基础安全配置

```nginx
server {
    # 隐藏版本号
    server_tokens off;
    
    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    
    # 限制请求方法
    if ($request_method !~ ^(GET|HEAD|POST)$ ) {
        return 405;
    }
    
    # 禁止访问敏感文件
    location ~* \.(env|git|svn|htaccess|htpasswd)$ {
        deny all;
        return 404;
    }
    
    # 限制请求体大小
    client_max_body_size 10m;
}
```

### 限流配置

```nginx
http {
    # 定义限流区域
    limit_req_zone $binary_remote_addr zone=req_limit:10m rate=10r/s;
    limit_conn_zone $binary_remote_addr zone=conn_limit:10m;
    
    server {
        # 请求限流
        limit_req zone=req_limit burst=20 nodelay;
        
        # 连接限流
        limit_conn conn_limit 20;
        
        # 限流状态码
        limit_req_status 429;
        limit_conn_status 429;
    }
}
```

### IP访问控制

```nginx
server {
    # 允许/拒绝IP
    location /admin/ {
        allow 192.168.1.0/24;
        allow 10.0.0.0/8;
        deny all;
    }
    
    # 基于地理位置
    location / {
        if ($geoip_country_code = CN) {
            return 403;
        }
    }
}
```

## 监控配置

### 状态页面

```nginx
server {
    location /nginx_status {
        stub_status on;
        access_log off;
        allow 127.0.0.1;
        allow 192.168.1.0/24;
        deny all;
    }
}
```

### 日志格式优化

```nginx
log_format json_combined escape=json '{'
    '"time_local":"$time_local",'
    '"remote_addr":"$remote_addr",'
    '"remote_user":"$remote_user",'
    '"request":"$request",'
    '"status":"$status",'
    '"body_bytes_sent":"$body_bytes_sent",'
    '"request_time":"$request_time",'
    '"http_referrer":"$http_referer",'
    '"http_user_agent":"$http_user_agent",'
    '"http_x_forwarded_for":"$http_x_forwarded_for",'
    '"upstream_addr":"$upstream_addr",'
    '"upstream_response_time":"$upstream_response_time"'
'}';

access_log /var/log/nginx/access.log json_combined;
```

## 完整优化配置示例

```nginx
user nginx;
worker_processes auto;
worker_rlimit_nofile 100000;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 65535;
    use epoll;
    multi_accept on;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    
    log_format json_combined escape=json '{"time":"$time_local",'
        '"ip":"$remote_addr","method":"$request_method",'
        '"uri":"$request_uri","status":"$status",'
        '"bytes":"$body_bytes_sent","time":"$request_time"}';
    
    access_log /var/log/nginx/access.log json_combined;
    
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    keepalive_requests 1000;
    server_tokens off;
    
    client_body_buffer_size 16k;
    client_header_buffer_size 1k;
    client_max_body_size 8m;
    large_client_header_buffers 4 8k;
    
    gzip on;
    gzip_vary on;
    gzip_min_length 1k;
    gzip_comp_level 6;
    gzip_types text/plain text/css application/json application/javascript;
    
    limit_req_zone $binary_remote_addr zone=req_limit:10m rate=10r/s;
    limit_conn_zone $binary_remote_addr zone=conn_limit:10m;
    
    upstream backend {
        least_conn;
        server 192.168.1.1:8080 weight=5;
        server 192.168.1.2:8080 weight=5;
        keepalive 32;
    }
    
    server {
        listen 80;
        server_name example.com;
        
        limit_req zone=req_limit burst=20 nodelay;
        limit_conn conn_limit 20;
        
        add_header X-Frame-Options "SAMEORIGIN" always;
        add_header X-Content-Type-Options "nosniff" always;
        
        location / {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Connection "";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }
        
        location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
            expires 30d;
            add_header Cache-Control "public, immutable";
            access_log off;
        }
        
        location /nginx_status {
            stub_status on;
            access_log off;
            allow 127.0.0.1;
            deny all;
        }
    }
}
```

## 参考资源

- [Nginx官方文档](https://nginx.org/en/docs/)
- [Nginx性能优化](https://www.nginx.com/blog/tuning-nginx/)

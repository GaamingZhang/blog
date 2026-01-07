# 如何查看nginx里访问请求的延时

## 引言

在Web服务中，请求延时是衡量系统性能的重要指标之一。Nginx作为广泛使用的反向代理和Web服务器，准确查看和分析其处理请求的延时对于优化系统性能、提升用户体验至关重要。

本文将详细介绍如何在Nginx中查看访问请求的延时，包括通过日志配置、实时监控工具和第三方方案等多种方法，并提供延时分析和优化建议。

## Nginx请求处理流程与延时环节

要有效查看和分析Nginx请求延时，首先需要了解Nginx的请求处理流程以及延时可能产生的环节：

1. **连接建立阶段**：客户端与Nginx建立TCP连接的时间
2. **请求接收阶段**：Nginx接收完整HTTP请求的时间
3. **请求处理阶段**：Nginx内部处理请求的时间（包括路由匹配、访问控制等）
4. ** upstream通信阶段**：Nginx与后端服务器通信的时间（如果配置了反向代理）
5. **响应发送阶段**：Nginx向客户端发送响应的时间

在这些环节中，任何一个环节的延迟都会影响整体请求处理时间。

## 查看Nginx请求延时的方法

### 通过访问日志查看请求延时

#### Nginx日志格式与延时变量

Nginx提供了丰富的变量来记录请求处理的各个时间信息。默认情况下，Nginx的访问日志格式如下：

```nginx
log_format combined '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';
```

要查看请求延时，需要在日志格式中添加以下关键变量：

| 变量名 | 说明 |
|--------|------|
| `$request_time` | 从客户端接收第一个字节到发送最后一个字节的总时间（秒） |
| `$upstream_response_time` | 与upstream服务器建立连接后，到接收到最后一个响应字节的时间（秒） |
| `$upstream_connect_time` | 与upstream服务器建立连接所用的时间（秒） |
| `$upstream_header_time` | 从建立连接到接收到upstream响应头的时间（秒） |
| `$upstream_first_byte_time` | 从建立连接到接收到upstream第一个响应字节的时间（秒） |
| `$ssl_handshake_time` | SSL握手所用的时间（秒） |

这些变量可以帮助我们分析请求处理过程中各个阶段的耗时情况。

#### 配置包含延时信息的日志格式

在Nginx配置文件中（通常是`nginx.conf`），定义一个包含延时信息的日志格式：

```nginx
# 在http块中定义日志格式
log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                '$status $body_bytes_sent "$http_referer" '
                '"$http_user_agent" "$http_x_forwarded_for" '
                'rt=$request_time uct=$upstream_connect_time uht=$upstream_header_time urt=$upstream_response_time';
```

然后在server或location块中应用这个日志格式：

```nginx
server {
    listen 80;
    server_name example.com;
    
    access_log /var/log/nginx/example.com.access.log main;
    
    location / {
        proxy_pass http://backend;
    }
}
```

重启Nginx使配置生效：

```bash
sudo nginx -t  # 测试配置是否正确
sudo systemctl restart nginx  # 重启Nginx服务
```

#### 解析访问日志查看延时

配置完成后，Nginx的访问日志将包含延时信息。以下是一个日志示例：

```
192.168.1.1 - - [2023-05-20T10:30:45+08:00] "GET /api/users HTTP/1.1" 200 1024 "-" "Mozilla/5.0" "-" rt=0.345 uct=0.012 uht=0.156 urt=0.321
```

在这个日志条目中：
- `rt=0.345`：总请求时间为0.345秒
- `uct=0.012`：与upstream建立连接的时间为0.012秒
- `uht=0.156`：从建立连接到接收响应头的时间为0.156秒
- `urt=0.321`：upstream的总响应时间为0.321秒

#### 常用日志分析命令

使用Linux命令可以快速分析访问日志中的延时信息：

1. **查看所有请求的平均延时**
   ```bash
   awk '{print $NF}' /var/log/nginx/example.com.access.log | cut -d= -f2 | awk '{sum+=$1; count++} END {print "平均延时: " sum/count "秒"}'
   ```

2. **查看最长的10个请求**
   ```bash
   awk '{print $NF, $0}' /var/log/nginx/example.com.access.log | sort -nr | head -10
   ```

3. **统计不同URL路径的平均延时**
   ```bash
   awk '{split($7, path, "?"); print path[1] " " $NF}' /var/log/nginx/example.com.access.log | cut -d= -f1,3 | awk '{sum[$1]+=$2; count[$1]++} END {for (url in sum) print url ": " sum[url]/count[url] "秒"}'
   ```

4. **统计响应状态码与延时的关系**
   ```bash
   awk '{print $9 " " $NF}' /var/log/nginx/example.com.access.log | cut -d= -f2 | awk '{sum[$1]+=$2; count[$1]++} END {for (code in sum) print code ": " sum[code]/count[code] "秒"}'
   ```

### 使用实时监控模块

Nginx提供了一些内置模块可以实时监控请求延时，以下是常用的几个：

#### ngx_http_stub_status_module

这是Nginx官方提供的基础监控模块，可以查看连接数、请求数等基本状态，但不直接提供延时统计。

**配置方法**：

```nginx
# 在http块中添加
server {
    listen 8080;
    server_name localhost;
    
    location /nginx_status {
        stub_status on;
        allow 127.0.0.1;
        deny all;
    }
}
```

**查看状态**：

```bash
curl http://localhost:8080/nginx_status
```

**输出示例**：
```
Active connections: 2 
server accepts handled requests
 32 32 102 
Reading: 0 Writing: 1 Waiting: 1 
```

#### ngx_http_lua_module (OpenResty)

OpenResty的lua模块可以更灵活地监控请求延时，通过在请求处理的不同阶段注入lua代码来记录时间。

**配置示例**：

```nginx
# 在http块中添加
init_by_lua_block {
    require "resty.core"
}

lua_shared_dict timing 10m;

server {
    listen 80;
    server_name example.com;
    
    # 记录请求开始时间
    rewrite_by_lua_block {
        ngx.ctx.start_time = ngx.now()
    }
    
    location / {
        proxy_pass http://backend;
        
        # 记录upstream响应时间
        proxy_set_header X-Start-Time $request_time;
    }
    
    # 计算并记录总延时
    log_by_lua_block {
        local start_time = ngx.ctx.start_time or ngx.now()
        local total_time = ngx.now() - start_time
        
        -- 存储到共享字典
        local timing = ngx.shared.timing
        timing:incr("total_requests", 1, 0)
        timing:incr("total_time", total_time, 0)
        
        -- 计算平均延时
        local avg_time = timing:get("total_time") / timing:get("total_requests")
        ngx.log(ngx.INFO, "平均请求延时: " .. string.format("%.3f", avg_time) .. "秒")
    }
    
    # 提供监控接口
    location /monitor {
        default_type application/json;
        content_by_lua_block {
            local timing = ngx.shared.timing
            local total_requests = timing:get("total_requests") or 0
            local total_time = timing:get("total_time") or 0
            local avg_time = 0
            
            if total_requests > 0 then
                avg_time = total_time / total_requests
            end
            
            ngx.print('{"total_requests": "' .. total_requests .. '", "avg_response_time": "' .. string.format("%.3f", avg_time) .. '"}')
        }
        allow 127.0.0.1;
        deny all;
    }
}
```

**查看实时监控数据**：

```bash
curl http://localhost/monitor
```

#### ngx_http_status_module

这是一个第三方模块，提供更详细的状态信息，包括延时统计。

**安装方法**：

```bash
git clone https://github.com/vozlt/nginx-module-vts.git
# 重新编译Nginx，添加--add-module=path/to/nginx-module-vts
```

**配置示例**：

```nginx
# 在http块中添加
vhost_traffic_status_zone;
vhost_traffic_status_filter_by_host on;

server {
    listen 80;
    server_name example.com;
    
    location /status {
        vhost_traffic_status_display;
        vhost_traffic_status_display_format html;
        allow 127.0.0.1;
        deny all;
    }
}
```

### 利用第三方监控工具

除了Nginx内置模块外，还可以使用以下第三方工具来监控Nginx请求延时：

#### Prometheus + Grafana

Prometheus配合Grafana可以提供强大的监控和可视化功能。

**安装与配置**：

1. **安装nginx-prometheus-exporter**：
   ```bash
   wget https://github.com/nginxinc/nginx-prometheus-exporter/releases/download/v1.1.0/nginx-prometheus-exporter_1.1.0_linux_amd64.tar.gz
   tar -xzf nginx-prometheus-exporter_1.1.0_linux_amd64.tar.gz
   sudo cp nginx-prometheus-exporter /usr/local/bin/
   ```

2. **配置Nginx**（启用stub_status模块）：
   ```nginx
   server {
       listen 8080;
       location /stub_status {
           stub_status on;
           allow 127.0.0.1;
           deny all;
       }
   }
   ```

3. **启动exporter**：
   ```bash
   nginx-prometheus-exporter -nginx.scrape-uri=http://localhost:8080/stub_status
   ```

4. **配置Prometheus**：
   ```yaml
   scrape_configs:
     - job_name: 'nginx'
       static_configs:
         - targets: ['localhost:9113']
   ```

5. **在Grafana中导入Nginx仪表盘**（ID：9614）

#### OpenTelemetry + Jaeger

OpenTelemetry可以收集分布式追踪数据，Jaeger用于可视化这些数据，包括请求延时。

**配置示例**：

```nginx
# 在http块中添加
server {
    listen 80;
    server_name example.com;
    
    # 安装nginx-opentracing模块后配置
    opentracing on;
    opentracing_load_tracer /usr/local/lib/libjaegertracing_plugin.so "/etc/jaeger/jaeger-config.json";
    opentracing_trace_locations off;
    
    location / {
        opentracing_operation_name "$request_method $uri";
        opentracing_tag http.status "$status";
        opentracing_tag http.url "$scheme://$host$request_uri";
        opentracing_tag http.method "$request_method";
        
        proxy_pass http://backend;
        proxy_set_header OT-Tracer-Sampled 1;
        proxy_set_header OT-Tracer-TraceId $opentracing_trace_id;
        proxy_set_header OT-Tracer-SpanId $opentracing_span_id;
    }
}
```

#### ELK Stack

Elasticsearch、Logstash、Kibana可以用于集中收集、分析和可视化Nginx日志，包括延时信息。

**配置示例**：

1. **配置Logstash**（logstash.conf）：
   ```
   input {
     file {
       path => "/var/log/nginx/*.access.log"
       start_position => "beginning"
     }
   }
   
   filter {
     grok {
       match => { "message" => "%{IPORHOST:remote_addr} - %{DATA:remote_user} \[%{HTTPDATE:timestamp}\] \"%{DATA:method} %{DATA:request} HTTP/%{NUMBER:http_version}\" %{NUMBER:status} %{NUMBER:body_bytes_sent} \"%{DATA:referer}\" \"%{DATA:user_agent}\" \"%{DATA:x_forwarded_for}\" rt=%{NUMBER:request_time:float} uct=%{NUMBER:upstream_connect_time:float} uht=%{NUMBER:upstream_header_time:float} urt=%{NUMBER:upstream_response_time:float}" }
     }
     
     date {
       match => [ "timestamp", "dd/MMM/yyyy:HH:mm:ss Z" ]
       target => "@timestamp"
     }
   }
   
   output {
     elasticsearch {
       hosts => ["localhost:9200"]
       index => "nginx-%{+YYYY.MM.dd}"
     }
   }
   ```

2. **在Kibana中创建可视化**：
   - 创建索引模式 `nginx-*`
   - 使用`request_time`字段创建延时可视化图表
   - 设置监控面板实时查看延时趋势

## 延时分析与优化建议

收集到请求延时数据后，需要进行有效的分析并采取相应的优化措施。以下是一些常用的分析方法和优化建议：

### 延时分析方法

#### 识别主要延时环节

根据日志中的延时变量，可以定位延时主要发生在哪个环节：

- **连接建立阶段**：`$ssl_handshake_time` 或 `$upstream_connect_time` 异常大
- **请求处理阶段**：`$request_time` 远大于 `$upstream_response_time`
- **后端通信阶段**：`$upstream_response_time` 异常大
- **响应发送阶段**：`$request_time` 远大于 `$upstream_response_time` + `$upstream_connect_time`

#### 分析延时分布

除了平均延时，还需要关注延时分布情况：

```bash
# 计算P50、P95、P99延时
awk '{print $NF}' /var/log/nginx/example.com.access.log | cut -d= -f2 | sort -n | \
awk '{n++;a[n]=$1} END {print "P50: " a[int(n*0.5)] "秒"; print "P95: " a[int(n*0.95)] "秒"; print "P99: " a[int(n*0.99)] "秒"}'
```

#### 关联分析

将延时与其他维度关联分析：

- **时间维度**：分析延时是否与特定时间段相关（如高峰期）
- **请求类型**：分析不同URL路径或请求方法的延时差异
- **客户端维度**：分析不同客户端IP或用户代理的延时差异
- **状态码**：分析不同响应状态码的延时差异

### 常见延时原因

#### 网络层面问题

- 网络带宽不足或拥塞
- DNS解析延迟
- TCP连接建立延迟（三次握手）
- 网络路由问题

#### Nginx配置问题

- 连接数限制过低
- 缓冲区设置不合理
- 负载均衡算法不当
- 缺少必要的缓存配置

#### 后端服务问题

- 后端服务响应缓慢
- 后端服务连接数不足
- 数据库查询慢
- 后端服务资源不足（CPU、内存、磁盘I/O）

#### 资源限制问题

- Nginx进程数不足
- 系统文件描述符限制
- 内存不足导致频繁交换
- 磁盘I/O瓶颈

### 优化建议

#### 网络层面优化

1. **启用TCP Fast Open**：
   ```nginx
   http {
       tcp_fastopen on;
   }
   ```

2. **优化TCP参数**：
   ```nginx
   http {
       keepalive_timeout 65;
       keepalive_requests 100;
       send_timeout 30;
       
       # 优化TCP缓冲区
       tcp_nopush on;
       tcp_nodelay on;
   }
   ```

3. **使用DNS缓存**：
   ```nginx
   http {
       resolver 8.8.8.8 valid=300s;
       resolver_timeout 5s;
   }
   ```

#### Nginx配置优化

1. **调整工作进程数**：
   ```nginx
   worker_processes auto;
   worker_cpu_affinity auto;
   ```

2. **优化连接处理**：
   ```nginx
   events {
       worker_connections 10240;
       use epoll;
       multi_accept on;
   }
   ```

3. **启用缓存**：
   ```nginx
   http {
       proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=my_cache:10m max_size=10g inactive=60m use_temp_path=off;
       
       server {
           location / {
               proxy_cache my_cache;
               proxy_cache_key "$host$request_uri";
               proxy_cache_valid 200 304 10m;
               proxy_pass http://backend;
           }
       }
   }
   ```

4. **优化upstream配置**：
   ```nginx
   http {
       upstream backend {
           least_conn;  # 使用最少连接数算法
           server 127.0.0.1:8000 max_fails=3 fail_timeout=30s;
           server 127.0.0.1:8001 max_fails=3 fail_timeout=30s;
       }
   }
   ```

#### 后端服务优化

1. **优化应用代码**：减少不必要的计算和I/O操作
2. **数据库优化**：添加索引、优化查询、使用连接池
3. **增加后端实例**：横向扩展以提高处理能力
4. **使用异步处理**：对于耗时操作使用异步处理方式

#### 资源优化

1. **调整系统参数**：
   ```bash
   # 增加文件描述符限制
   echo "* soft nofile 65535" >> /etc/security/limits.conf
   echo "* hard nofile 65535" >> /etc/security/limits.conf
   
   # 优化内核参数
   echo "net.core.somaxconn = 10240" >> /etc/sysctl.conf
   echo "net.ipv4.tcp_max_syn_backlog = 10240" >> /etc/sysctl.conf
   sysctl -p
   ```

2. **监控资源使用**：
   - 使用`top`、`vmstat`、`iostat`等命令监控系统资源
   - 设置资源告警，及时发现资源瓶颈

## 常见问题解答（FAQ）

### 如何区分Nginx处理延时和后端服务延时？

可以通过比较`$request_time`和`$upstream_response_time`来区分：

- **Nginx处理延时** = `$request_time` - `$upstream_response_time`（如果没有配置反向代理，则`$upstream_response_time`为空）
- **后端服务延时** = `$upstream_response_time`

如果Nginx处理延时过大，可能是Nginx配置问题（如路由匹配复杂、访问控制严格等）；如果后端服务延时过大，则需要优化后端服务。

### 为什么`upstream_response_time`显示为"-"？

`upstream_response_time`显示为"-"通常有以下原因：

1. **未配置反向代理**：如果Nginx直接处理请求（如静态文件服务），没有配置`proxy_pass`，则不会有`upstream_response_time`
2. **请求在Nginx内部被拒绝**：如访问控制列表（ACL）拒绝、认证失败等
3. **连接后端失败**：如后端服务不可用、网络连接失败等
4. **请求被重定向**：如果请求被Nginx内部重定向，可能不会记录`upstream_response_time`

### 如何设置请求延时告警？

可以使用以下方法设置延时告警：

#### 使用Prometheus + Alertmanager

```yaml
# 在Prometheus告警规则文件中添加
groups:
- name: nginx_alerts
  rules:
  - alert: HighRequestLatency
    expr: nginx_http_request_duration_seconds{quantile="0.95"} > 5
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "高请求延时（{{ $value }}秒）"
      description: "Nginx 95%请求延时超过5秒"
```

#### 使用日志监控工具

在ELK Stack中，可以使用Watcher功能设置告警：

```json
{
  "trigger": {
    "schedule": {
      "interval": "5m"
    }
  },
  "input": {
    "search": {
      "request": {
        "body": {
          "query": {
            "range": {
              "request_time": {
                "gt": 5
              }
            }
          },
          "aggs": {
            "avg_latency": {
              "avg": {
                "field": "request_time"
              }
            }
          }
        },
        "indices": ["nginx-*"]
      }
    }
  },
  "condition": {
    "compare": {
      "ctx.payload.aggregations.avg_latency.value": {
        "gt": 5
      }
    }
  },
  "actions": {
    "send_email": {
      "email": {
        "to": ["admin@example.com"],
        "subject": "高Nginx请求延时告警",
        "body": "平均请求延时：{{ ctx.payload.aggregations.avg_latency.value }}秒"
      }
    }
  }
}
```

### `$request_time`和`$upstream_response_time`有什么区别？

- **`$request_time`**：从客户端接收第一个字节到发送最后一个字节的总时间，包括：
  - 建立连接的时间
  - 接收请求头和请求体的时间
  - Nginx处理请求的时间
  - 与后端服务器通信的时间
  - 发送响应的时间

- **`$upstream_response_time`**：仅指Nginx与后端服务器通信的时间，包括：
  - 与后端建立连接的时间
  - 发送请求到后端的时间
  - 等待后端响应的时间
  - 接收后端响应的时间

### 收集延时信息时如何减少日志对性能的影响？

可以通过以下方法减少日志对性能的影响：

1. **使用缓冲日志**：
   ```nginx
   http {
       access_log /var/log/nginx/access.log main buffer=32k flush=5s;
   }
   ```

2. **设置合理的日志级别**：
   ```nginx
   error_log /var/log/nginx/error.log warn;
   ```

3. **限制日志内容**：只记录必要的延时字段，避免记录过多无关信息

4. **使用异步日志**：编译Nginx时添加`--with-http_mp4_module`和`--with-threads`参数，启用异步日志

5. **定期归档和清理日志**：避免日志文件过大影响性能
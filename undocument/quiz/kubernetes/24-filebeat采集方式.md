---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Filebeat
  - 日志
---

# Filebeat行首行末采集方式详解

## Filebeat简介

Filebeat是Elastic公司推出的轻量级日志采集器，专门用于转发和集中日志数据。它作为代理安装在服务器上，监控指定的日志文件或位置，收集日志事件并转发到Elasticsearch或Logstash。

## 行首行末采集的概念

### 什么是行首行末？

在日志采集过程中，"行首行末"指的是日志行的边界识别方式：

```
┌─────────────────────────────────────────────────────────────┐
│                    日志行边界识别                            │
│                                                              │
│  行首（Line Start）                                          │
│  ↓                                                           │
│  2024-01-15 10:00:00 INFO Application started               │
│  2024-01-15 10:00:01 DEBUG Processing request               │
│  2024-01-15 10:00:02 ERROR Exception occurred               │
│  java.lang.NullPointerException                             │
│      at com.example.App.process(App.java:10)                │
│      at com.example.App.main(App.java:5)                    │
│  2024-01-15 10:00:03 INFO Request completed                 │
│                                              ↑               │
│                                           行末（Line End）    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 为什么需要关注行首行末？

1. **多行日志处理**：Java异常堆栈、JSON格式日志等可能跨越多行
2. **日志完整性**：确保每条日志被完整采集
3. **解析准确性**：正确的行边界是日志解析的基础

## Filebeat的行处理机制

### 默认行为

Filebeat默认按换行符分割日志，每行作为一条独立的事件。

```
日志文件:
line1
line2
line3

采集结果:
event1: line1
event2: line2
event3: line3
```

### 多行日志处理

对于多行日志（如Java异常堆栈），需要配置multiline选项：

```
日志文件:
2024-01-15 10:00:00 ERROR Exception occurred
java.lang.NullPointerException
    at com.example.App.process(App.java:10)
    at com.example.App.main(App.java:5)
2024-01-15 10:00:01 INFO Application resumed

期望采集结果:
event1: 2024-01-15 10:00:00 ERROR Exception occurred
        java.lang.NullPointerException
            at com.example.App.process(App.java:10)
            at com.example.App.main(App.java:5)
event2: 2024-01-15 10:00:01 INFO Application resumed
```

## Multiline配置详解

### 配置参数

| 参数 | 说明 |
|------|------|
| pattern | 匹配行的正则表达式 |
| negate | 是否对pattern结果取反 |
| match | 如何将匹配行合并（before/after） |

### 工作原理

```
┌─────────────────────────────────────────────────────────────┐
│                 Multiline工作原理                            │
│                                                              │
│  1. 读取一行                                                 │
│  2. 用pattern匹配该行                                        │
│  3. 根据negate决定是否取反                                   │
│  4. 根据match决定合并方式                                    │
│                                                              │
│  negate: false (默认)                                        │
│  - 匹配成功 → 符合条件的行                                   │
│  - 匹配失败 → 不符合条件的行                                 │
│                                                              │
│  negate: true                                                │
│  - 匹配成功 → 不符合条件的行                                 │
│  - 匹配失败 → 符合条件的行                                   │
│                                                              │
│  match: after                                                │
│  - 不符合条件的行追加到前一个匹配行之后                       │
│                                                              │
│  match: before                                               │
│  - 不符合条件的行追加到后一个匹配行之前                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 常见场景配置

#### 场景1：Java异常堆栈

日志格式：
```
2024-01-15 10:00:00 ERROR Exception occurred
java.lang.NullPointerException
    at com.example.App.process(App.java:10)
    at com.example.App.main(App.java:5)
2024-01-15 10:00:01 INFO Application resumed
```

配置：
```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  multiline:
    pattern: '^[0-9]{4}-[0-9]{2}-[0-9]{2}'
    negate: true
    match: after
```

**解释**：
- `pattern`：匹配以日期开头的行
- `negate: true`：取反，即不以日期开头的行
- `match: after`：不以日期开头的行追加到前一个匹配行之后

#### 场景2：以时间戳开头的日志

日志格式：
```
[2024-01-15 10:00:00] INFO Application started
[2024-01-15 10:00:01] DEBUG Processing request
  detail: user_id=123
  action=login
[2024-01-15 10:00:02] INFO Request completed
```

配置：
```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  multiline:
    pattern: '^\[[0-9]{4}-[0-9]{2}-[0-9]{2}'
    negate: true
    match: after
```

#### 场景3：JSON格式日志（多行）

日志格式：
```json
{
  "timestamp": "2024-01-15T10:00:00",
  "level": "INFO",
  "message": "Request processed",
  "details": {
    "user_id": 123,
    "action": "login"
  }
}
{
  "timestamp": "2024-01-15T10:00:01",
  "level": "ERROR",
  "message": "Exception occurred"
}
```

配置：
```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  multiline:
    pattern: '^\{'
    negate: true
    match: after
```

#### 场景4：以空格或制表符开头的续行

日志格式：
```
2024-01-15 10:00:00 ERROR Exception occurred
  java.lang.NullPointerException
      at com.example.App.process(App.java:10)
2024-01-15 10:00:01 INFO Application resumed
```

配置：
```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  multiline:
    pattern: '^[[:space:]]'
    negate: false
    match: after
```

**解释**：
- `pattern`：匹配以空格开头的行
- `negate: false`：不取反
- `match: after`：以空格开头的行追加到前一行之后

#### 场景5：C语言风格的日志

日志格式：
```
Starting application...
  Loading config...
  Connecting to database...
Application started
Processing request...
  Validating input
  Executing query
Request completed
```

配置：
```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  multiline:
    pattern: '^[[:space:]]'
    negate: false
    match: before
```

**解释**：
- `match: before`：以空格开头的行追加到后一个匹配行之前

## 行尾处理

### 行尾符

Filebeat支持多种行尾符：

| 行尾符 | 说明 |
|--------|------|
| \n | Unix/Linux换行符（默认） |
| \r\n | Windows换行符 |
| \r | 旧版Mac换行符 |

### 配置行尾符

```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  line_terminator: "\n"
```

### 行尾超时

对于没有换行符的行，可以设置超时：

```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  multiline:
    pattern: '^[0-9]{4}'
    negate: true
    match: after
    timeout: 10s
```

**timeout**：如果10秒内没有新行，则将当前缓冲区的内容作为一个事件发送。

## 从行首开始采集

### 使用tail模式

Filebeat默认从文件末尾开始采集（tail模式）：

```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  # 默认从文件末尾开始
```

### 从文件开头采集

使用head模式从文件开头采集：

```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  close_eof: true
```

### 从特定位置开始

使用offset配置：

```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  scan_frequency: 10s
```

### 使用注册文件

Filebeat使用注册文件记录采集位置：

```yaml
path.data: /var/lib/filebeat
path.logs: /var/log/filebeat

registry:
  path: /var/lib/filebeat/registry
  flush: 1s
```

## 从行末开始采集

### 实时采集新日志

Filebeat默认从文件末尾开始监听新日志：

```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  tail_files: true  # 默认值
```

### 忽略旧日志

```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  ignore_older: 24h  # 忽略24小时前的日志
```

### 关闭旧文件

```yaml
filebeat.inputs:
- type: log
  paths:
  - /var/log/app/*.log
  close_inactive: 5m   # 5分钟无新数据则关闭文件
  close_renamed: true  # 文件重命名时关闭
  close_removed: true  # 文件删除时关闭
  close_eof: false     # 到达文件末尾时不关闭
```

## 完整配置示例

### Java应用日志采集

```yaml
filebeat.inputs:
- type: log
  enabled: true
  paths:
  - /var/log/app/*.log
  multiline:
    pattern: '^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}'
    negate: true
    match: after
    timeout: 10s
  fields:
    app: myapp
    env: production
  fields_under_root: true
  ignore_older: 24h
  close_inactive: 5m

processors:
- add_host_metadata:
    when.not.contains.tags: forwarded
- add_cloud_metadata: ~
- add_docker_metadata: ~

output.logstash:
  hosts: ["logstash:5044"]
  bulk_max_size: 2048

logging.level: info
logging.to_files: true
logging.files:
  path: /var/log/filebeat
  name: filebeat
  keepfiles: 7
  permissions: 0644
```

### Kubernetes容器日志采集

```yaml
filebeat.inputs:
- type: container
  paths:
  - /var/log/containers/*.log
  processors:
  - add_kubernetes_metadata:
      host: ${NODE_NAME}
      matchers:
      - logs_path:
          logs_path: "/var/log/containers/"
  multiline:
    pattern: '^[0-9]{4}-[0-9]{2}-[0-9]{2}|^[A-Z]'
    negate: true
    match: after

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  indices:
  - index: "k8s-%{[kubernetes.namespace]}-%{+yyyy.MM.dd}"
```

## 常见问题

### Q1: 多行日志没有正确合并？

检查：
1. pattern是否正确匹配行首
2. negate和match组合是否正确
3. 是否设置了timeout

### Q2: 日志丢失？

检查：
1. 注册文件是否正常
2. close_inactive是否设置过小
3. 磁盘空间是否充足

### Q3: 内存占用过高？

检查：
1. multiline缓冲区是否过大
2. 是否有超大单行日志
3. bulk_max_size是否过大

## 最佳实践

### 1. 测试multiline配置

```bash
filebeat test config -c filebeat.yml
```

### 2. 设置合理的超时

```yaml
multiline:
  timeout: 10s
```

### 3. 监控采集状态

```yaml
monitoring:
  enabled: true
  elasticsearch:
    hosts: ["elasticsearch:9200"]
```

### 4. 使用处理器优化

```yaml
processors:
- drop_event:
    when:
      regexp:
        message: "^DEBUG"
```

## 参考资源

- [Filebeat官方文档](https://www.elastic.co/guide/en/beats/filebeat/current/configuring-howto-filebeat.html)
- [Multiline配置](https://www.elastic.co/guide/en/beats/filebeat/current/multiline-examples.html)

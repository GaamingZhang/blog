---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ELK
  - 日志
---

# ELK日志采集流程详解

## 什么是ELK？

ELK是三个开源项目的首字母缩写：
- **E**lasticsearch：搜索和分析引擎
- **L**ogstash：数据处理管道
- **K**ibana：数据可视化平台

后来加入了Beats（轻量级数据采集器），形成了现在的ELK Stack。

```
┌─────────────────────────────────────────────────────────────┐
│                     ELK Stack架构                            │
│                                                              │
│  ┌─────────┐   ┌─────────┐   ┌─────────────┐   ┌─────────┐ │
│  │  Beats  │ → │Logstash │ → │Elasticsearch│ → │ Kibana  │ │
│  │ (采集)  │   │ (处理)  │   │   (存储)    │   │ (展示)  │ │
│  └─────────┘   └─────────┘   └─────────────┘   └─────────┘ │
│                                                              │
│  Beats: Filebeat, Metricbeat, Packetbeat, Heartbeat...     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 完整日志采集流程

### 流程图

```
┌─────────────────────────────────────────────────────────────┐
│                    日志采集完整流程                           │
│                                                              │
│  应用容器                                                    │
│  ┌─────────────────┐                                        │
│  │  /var/log/app/  │                                        │
│  │    app.log      │                                        │
│  └────────┬────────┘                                        │
│           │                                                  │
│           ↓                                                  │
│  Filebeat (DaemonSet)                                        │
│  ┌─────────────────┐                                        │
│  │  读取日志文件    │                                        │
│  │  简单处理       │                                        │
│  │  发送到Logstash │                                        │
│  └────────┬────────┘                                        │
│           │                                                  │
│           ↓                                                  │
│  Logstash                                                    │
│  ┌─────────────────┐                                        │
│  │  Input          │                                        │
│  │  Filter (处理)  │                                        │
│  │  Output         │                                        │
│  └────────┬────────┘                                        │
│           │                                                  │
│           ↓                                                  │
│  Elasticsearch                                               │
│  ┌─────────────────┐                                        │
│  │  索引数据       │                                        │
│  │  分片存储       │                                        │
│  └────────┬────────┘                                        │
│           │                                                  │
│           ↓                                                  │
│  Kibana                                                      │
│  ┌─────────────────┐                                        │
│  │  搜索查询       │                                        │
│  │  可视化展示     │                                        │
│  │  仪表板        │                                        │
│  └─────────────────┘                                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 各组件详解

### 1. Filebeat

Filebeat是轻量级的日志采集器，负责从日志文件读取数据并发送到下游。

**部署方式（Kubernetes DaemonSet）**：

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: filebeat
  namespace: logging
spec:
  selector:
    matchLabels:
      app: filebeat
  template:
    metadata:
      labels:
        app: filebeat
    spec:
      serviceAccountName: filebeat
      containers:
      - name: filebeat
        image: docker.elastic.co/beats/filebeat:8.10.0
        args:
        - "-c"
        - "/etc/filebeat.yml"
        - "-e"
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        volumeMounts:
        - name: config
          mountPath: /etc/filebeat.yml
          readOnly: true
          subPath: filebeat.yml
        - name: data
          mountPath: /usr/share/filebeat/data
        - name: varlog
          mountPath: /var/log
          readOnly: true
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
      volumes:
      - name: config
        configMap:
          name: filebeat-config
      - name: data
        hostPath:
          path: /var/lib/filebeat-data
          type: DirectoryOrCreate
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers
```

**Filebeat配置**：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: filebeat-config
  namespace: logging
data:
  filebeat.yml: |-
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
    
    - type: log
      paths:
      - /var/log/app/*.log
      fields:
        app: myapp
      fields_under_root: true
    
    processors:
    - drop_event:
        when:
          equals:
            kubernetes.container.name: "filebeat"
    
    output.logstash:
      hosts: ["logstash:5044"]
      bulk_max_size: 2048
    
    logging.level: info
```

### 2. Logstash

Logstash是数据处理管道，负责解析、转换和过滤日志。

**部署方式**：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: logstash
  namespace: logging
spec:
  replicas: 2
  selector:
    matchLabels:
      app: logstash
  template:
    metadata:
      labels:
        app: logstash
    spec:
      containers:
      - name: logstash
        image: docker.elastic.co/logstash/logstash:8.10.0
        ports:
        - containerPort: 5044
        volumeMounts:
        - name: config
          mountPath: /usr/share/logstash/pipeline
          readOnly: true
        - name: config-volume
          mountPath: /usr/share/logstash/config/logstash.yml
          readOnly: true
          subPath: logstash.yml
      volumes:
      - name: config
        configMap:
          name: logstash-pipeline
      - name: config-volume
        configMap:
          name: logstash-config
---
apiVersion: v1
kind: Service
metadata:
  name: logstash
  namespace: logging
spec:
  ports:
  - port: 5044
    targetPort: 5044
  selector:
    app: logstash
```

**Logstash配置**：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: logstash-pipeline
  namespace: logging
data:
  logstash.conf: |-
    input {
      beats {
        port => 5044
      }
    }
    
    filter {
      if [kubernetes] {
        mutate {
          add_field => {
            "k8s_namespace" => "%{[kubernetes][namespace]}"
            "k8s_pod" => "%{[kubernetes][pod][name]}"
            "k8s_container" => "%{[kubernetes][container][name]}"
          }
        }
      }
      
      grok {
        match => { "message" => "%{TIMESTAMP_ISO8601:timestamp} %{LOGLEVEL:level} %{GREEDYDATA:msg}" }
      }
      
      date {
        match => [ "timestamp", "ISO8601" ]
        target => "@timestamp"
      }
      
      if [level] == "ERROR" {
        mutate {
          add_tag => ["error"]
        }
      }
    }
    
    output {
      elasticsearch {
        hosts => ["elasticsearch:9200"]
        index => "k8s-logs-%{+YYYY.MM.dd}"
      }
    }
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: logstash-config
  namespace: logging
data:
  logstash.yml: |-
    http.host: "0.0.0.0"
    path.config: /usr/share/logstash/pipeline
```

### 3. Elasticsearch

Elasticsearch是分布式搜索和分析引擎，负责存储和索引日志数据。

**部署方式**：

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: elasticsearch
  namespace: logging
spec:
  serviceName: elasticsearch
  replicas: 3
  selector:
    matchLabels:
      app: elasticsearch
  template:
    metadata:
      labels:
        app: elasticsearch
    spec:
      initContainers:
      - name: fix-permissions
        image: busybox
        command: ["sh", "-c", "chown -R 1000:1000 /usr/share/elasticsearch/data"]
        volumeMounts:
        - name: data
          mountPath: /usr/share/elasticsearch/data
      containers:
      - name: elasticsearch
        image: docker.elastic.co/elasticsearch/elasticsearch:8.10.0
        ports:
        - containerPort: 9200
          name: http
        - containerPort: 9300
          name: transport
        env:
        - name: cluster.name
          value: "k8s-logs"
        - name: node.name
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: discovery.seed_hosts
          value: "elasticsearch-0.elasticsearch,elasticsearch-1.elasticsearch,elasticsearch-2.elasticsearch"
        - name: cluster.initial_master_nodes
          value: "elasticsearch-0,elasticsearch-1,elasticsearch-2"
        - name: ES_JAVA_OPTS
          value: "-Xms2g -Xmx2g"
        volumeMounts:
        - name: data
          mountPath: /usr/share/elasticsearch/data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: standard
      resources:
        requests:
          storage: 100Gi
---
apiVersion: v1
kind: Service
metadata:
  name: elasticsearch
  namespace: logging
spec:
  ports:
  - port: 9200
    name: http
  - port: 9300
    name: transport
  selector:
    app: elasticsearch
```

### 4. Kibana

Kibana是可视化平台，用于搜索和展示日志。

**部署方式**：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kibana
  namespace: logging
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kibana
  template:
    metadata:
      labels:
        app: kibana
    spec:
      containers:
      - name: kibana
        image: docker.elastic.co/kibana/kibana:8.10.0
        ports:
        - containerPort: 5601
        env:
        - name: ELASTICSEARCH_HOSTS
          value: "http://elasticsearch:9200"
---
apiVersion: v1
kind: Service
metadata:
  name: kibana
  namespace: logging
spec:
  ports:
  - port: 5601
    targetPort: 5601
  selector:
    app: kibana
```

## 日志处理流程详解

### 1. 日志采集

Filebeat从以下位置采集日志：

```
/var/log/containers/     # Kubernetes容器日志（符号链接）
/var/log/pods/           # Kubernetes Pod日志
/var/lib/docker/containers/  # Docker容器日志
```

### 2. 日志解析

Logstash使用Grok模式解析日志：

```
原始日志:
2024-01-15T10:00:00.000Z INFO [main] Application started

解析后:
{
  "timestamp": "2024-01-15T10:00:00.000Z",
  "level": "INFO",
  "logger": "main",
  "msg": "Application started"
}
```

### 3. 日志索引

Elasticsearch按日期创建索引：

```
k8s-logs-2024.01.15
k8s-logs-2024.01.16
k8s-logs-2024.01.17
```

### 4. 日志查询

在Kibana中查询日志：

```
# 查询错误日志
level: ERROR

# 查询特定Pod
k8s_pod: nginx-*

# 查询特定命名空间
k8s_namespace: production

# 组合查询
level: ERROR AND k8s_namespace: production
```

## 常见日志格式解析

### JSON格式日志

```json
{
  "timestamp": "2024-01-15T10:00:00.000Z",
  "level": "INFO",
  "logger": "com.example.App",
  "message": "Request processed",
  "traceId": "abc123",
  "spanId": "def456",
  "userId": "user001",
  "request": {
    "method": "GET",
    "path": "/api/users",
    "duration": 150
  }
}
```

**Logstash配置**：

```
filter {
  json {
    source => "message"
    target => "json"
  }
  
  mutate {
    add_field => {
      "level" => "%{[json][level]}"
      "traceId" => "%{[json][traceId]}"
    }
  }
}
```

### 多行日志处理

```
2024-01-15 10:00:00 ERROR Exception occurred
java.lang.NullPointerException
    at com.example.App.process(App.java:10)
    at com.example.App.main(App.java:5)
```

**Filebeat配置**：

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

## 最佳实践

### 1. 使用Index Lifecycle Management

```json
PUT _ilm/policy/logs_policy
{
  "policy": {
    "phases": {
      "hot": {
        "actions": {
          "rollover": {
            "max_size": "50GB",
            "max_age": "1d"
          }
        }
      },
      "warm": {
        "min_age": "7d",
        "actions": {
          "shrink": {
            "number_of_shards": 1
          },
          "forcemerge": {
            "max_num_segments": 1
          }
        }
      },
      "delete": {
        "min_age": "30d",
        "actions": {
          "delete": {}
        }
      }
    }
  }
}
```

### 2. 合理设置分片

```json
PUT k8s-logs-*
{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 1
  }
}
```

### 3. 使用Pipeline处理日志

```json
PUT _ingest/pipeline/json_pipeline
{
  "processors": [
    {
      "json": {
        "field": "message",
        "target_field": "json"
      }
    },
    {
      "date": {
        "field": "json.timestamp",
        "formats": ["ISO8601"],
        "target_field": "@timestamp"
      }
    }
  ]
}
```

### 4. 监控集群健康

```yaml
groups:
- name: elasticsearch
  rules:
  - alert: ElasticsearchClusterRed
    expr: elasticsearch_cluster_health_status{color="red"} == 1
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Elasticsearch集群状态异常"
```

## 参考资源

- [Elastic官方文档](https://www.elastic.co/guide/)
- [Filebeat配置](https://www.elastic.co/guide/en/beats/filebeat/current/configuring-howto-filebeat.html)
- [Logstash配置](https://www.elastic.co/guide/en/logstash/current/configuration.html)

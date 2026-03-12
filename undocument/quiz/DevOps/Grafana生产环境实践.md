---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Monitoring
tag:
  - Grafana
  - Monitoring
  - Visualization
  - DevOps
---

# Grafana 生产环境实践

当你需要在同一个 Dashboard 上展示来自 Prometheus、InfluxDB、Elasticsearch、MySQL 等多个数据源的指标时,Grafana 是最强大的可视化工具。但 Grafana 并不是"配置好数据源就能用"——Dashboard 设计、告警集成、权限管理、性能优化都需要深入理解才能在生产环境稳定运行。

本文将从 Dashboard 设计、数据源配置、告警集成、权限管理、性能优化五个维度,系统梳理 Grafana 生产环境的实践经验。

## 一、Dashboard 设计

### Dashboard 设计原则

```
┌─────────────────────────────────────────────────────────────┐
│                    Dashboard 设计原则                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 层次化布局                                               │
│     - 顶部: 关键指标概览(SLO、可用性)                        │
│     - 中部: 详细指标(CPU、内存、网络)                        │
│     - 底部: 日志和事件                                       │
│                                                              │
│  2. 视觉一致性                                               │
│     - 统一的颜色编码(绿色=正常,红色=异常)                    │
│     - 统一的单位(GB、ms、%)                                  │
│     - 统一的时间范围                                         │
│                                                              │
│  3. 交互性                                                   │
│     - 变量下拉框(集群、命名空间、Pod)                        │
│     - 可点击跳转                                             │
│     - 时间范围选择                                           │
│                                                              │
│  4. 性能优化                                                 │
│     - 避免过多 Panel(建议 < 20 个)                          │
│     - 使用变量减少查询数量                                   │
│     - 合理设置刷新间隔                                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Dashboard JSON 结构

```json
{
  "dashboard": {
    "id": null,
    "title": "Kubernetes Cluster Monitoring",
    "tags": ["kubernetes", "monitoring"],
    "timezone": "browser",
    "schemaVersion": 16,
    "version": 0,
    "refresh": "30s",
    "time": {
      "from": "now-1h",
      "to": "now"
    },
    "templating": {
      "list": [
        {
          "name": "cluster",
          "type": "query",
          "datasource": "Prometheus",
          "refresh": 1,
          "query": "label_values(kube_pod_info, cluster)",
          "sort": 1
        },
        {
          "name": "namespace",
          "type": "query",
          "datasource": "Prometheus",
          "refresh": 1,
          "query": "label_values(kube_pod_info{cluster=\"$cluster\"}, namespace)",
          "sort": 1
        }
      ]
    },
    "panels": [
      {
        "id": 1,
        "title": "CPU Usage",
        "type": "graph",
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 0,
          "y": 0
        },
        "targets": [
          {
            "expr": "sum(rate(container_cpu_usage_seconds_total{cluster=\"$cluster\",namespace=\"$namespace\"}[5m])) by (pod)",
            "legendFormat": "{{pod}}",
            "refId": "A"
          }
        ]
      }
    ]
  }
}
```

### Panel 类型选择

| Panel 类型 | 适用场景 | 示例 |
|-----------|---------|------|
| Graph | 时间序列数据 | CPU、内存趋势 |
| Stat | 单个数值 | 当前在线用户数 |
| Gauge | 仪表盘 | 磁盘使用率 |
| Table | 表格展示 | Pod 列表 |
| Heatmap | 热力图 | 请求延迟分布 |
| Pie Chart | 饼图 | 资源分布 |
| Bar Chart | 柱状图 | 每日请求量 |

### 变量配置

**查询变量**:

```json
{
  "name": "pod",
  "type": "query",
  "datasource": "Prometheus",
  "refresh": 1,
  "query": "label_values(kube_pod_info{cluster=\"$cluster\",namespace=\"$namespace\"}, pod)",
  "sort": 1,
  "multi": true,
  "includeAll": true,
  "allValue": ".*"
}
```

**自定义变量**:

```json
{
  "name": "interval",
  "type": "custom",
  "options": [
    {"text": "1m", "value": "1m"},
    {"text": "5m", "value": "5m"},
    {"text": "10m", "value": "10m"},
    {"text": "30m", "value": "30m"},
    {"text": "1h", "value": "1h"}
  ],
  "current": {"text": "5m", "value": "5m"}
}
```

**链式变量**:

```json
// 变量 1: cluster
{
  "name": "cluster",
  "query": "label_values(kube_pod_info, cluster)"
}

// 变量 2: namespace(依赖 cluster)
{
  "name": "namespace",
  "query": "label_values(kube_pod_info{cluster=\"$cluster\"}, namespace)"
}

// 变量 3: pod(依赖 cluster 和 namespace)
{
  "name": "pod",
  "query": "label_values(kube_pod_info{cluster=\"$cluster\",namespace=\"$namespace\"}, pod)"
}
```

## 二、数据源配置

### Prometheus 数据源

```yaml
# Grafana Provisioning
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: false
    jsonData:
      httpMethod: POST
      manageAlerts: true
      prometheusType: Prometheus
      prometheusVersion: "2.40.0"
      cacheLevel: 'High'
    secureJsonData:
      basicAuthPassword: password
```

### Elasticsearch 数据源

```yaml
apiVersion: 1
datasources:
  - name: Elasticsearch
    type: elasticsearch
    access: proxy
    url: http://elasticsearch:9200
    database: "logstash-*"
    jsonData:
      esVersion: "7.10.0"
      timeField: "@timestamp"
      interval: Daily
      logMessageField: message
      logLevelField: log.level
```

### MySQL 数据源

```yaml
apiVersion: 1
datasources:
  - name: MySQL
    type: mysql
    access: proxy
    url: mysql:3306
    database: production
    user: grafana
    jsonData:
      maxOpenConns: 10
      maxIdleConns: 5
      connMaxLifetime: 14400
    secureJsonData:
      password: password
```

### 多数据源查询

```json
{
  "targets": [
    {
      "datasource": "Prometheus",
      "expr": "rate(http_requests_total[5m])",
      "refId": "A"
    },
    {
      "datasource": "MySQL",
      "query": "SELECT COUNT(*) as orders FROM orders WHERE created_at > NOW() - INTERVAL 1 HOUR",
      "refId": "B"
    }
  ]
}
```

## 三、告警集成

### Grafana 告警配置

**告警规则**:

```json
{
  "alerts": [
    {
      "condition": "query",
      "evaluator": {
        "params": [80],
        "type": "gt"
      },
      "operator": {
        "type": "and"
      },
      "query": {
        "params": ["A", "5m", "now"]
      },
      "reducer": {
        "params": [],
        "type": "avg"
      },
      "type": "query"
    }
  ],
  "targets": [
    {
      "expr": "100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)",
      "refId": "A"
    }
  ]
}
```

**通知渠道**:

```yaml
# Grafana Provisioning
apiVersion: 1
notifiers:
  - name: Slack
    type: slack
    uid: slack-1
    settings:
      url: https://hooks.slack.com/services/xxx
      recipient: '#alerts'
  
  - name: Email
    type: email
    uid: email-1
    settings:
      addresses: team@example.com
  
  - name: PagerDuty
    type: pagerduty
    uid: pagerduty-1
    settings:
      integrationKey: xxx
      severity: critical
```

### Alertmanager 集成

```yaml
# Grafana 数据源配置
apiVersion: 1
datasources:
  - name: Alertmanager
    type: alertmanager
    access: proxy
    url: http://alertmanager:9093
    jsonData:
      implementation: prometheus
```

**Grafana Alert 规则**:

```yaml
groups:
  - name: grafana_alerts
    rules:
      - uid: alert_1
        title: High CPU Usage
        condition: C
        data:
          - refId: A
            relativeTimeRange:
              from: 600
              to: 0
            datasourceUid: prometheus
            model:
              expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)
          
          - refId: B
            relativeTimeRange:
              from: 600
              to: 0
            datasourceUid: prometheus
            model:
              expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
          
          - refId: C
            relativeTimeRange:
              from: 600
              to: 0
            datasourceUid: __expr__
            model:
              type: reduce
              expression: B
              reducer: last
        noDataState: NoData
        execErrState: Alerting
        for: 5m
        annotations:
          description: "CPU usage is {{ $values.A }}%"
          summary: "High CPU usage detected"
        labels:
          severity: warning
```

## 四、权限管理

### 组织和团队

```
┌─────────────────────────────────────────────────────────────┐
│                    Grafana 权限架构                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Organization (组织)                                         │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  - 隔离不同环境(生产、测试、开发)                      │  │
│  │  - 独立的数据源和 Dashboard                            │  │
│  └──────────────────────────────────────────────────────┘  │
│                     │                                       │
│                     ▼                                       │
│  Team (团队)                                                 │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  - DevOps 团队: 管理所有 Dashboard                     │  │
│  │  - 开发团队: 只读访问相关 Dashboard                    │  │
│  │  - 业务团队: 只读访问业务 Dashboard                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                     │                                       │
│                     ▼                                       │
│  User (用户)                                                 │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  - Admin: 管理员权限                                   │  │
│  │  - Editor: 编辑权限                                    │  │
│  │  - Viewer: 只读权限                                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**配置示例**:

```yaml
# Grafana Provisioning
apiVersion: 1
teams:
  - name: DevOps
    email: devops@example.com
  - name: Developers
    email: developers@example.com

apiVersion: 1
users:
  - name: admin
    email: admin@example.com
    login: admin
    password: admin123
    isAdmin: true
  
  - name: developer1
    email: developer1@example.com
    login: developer1
    password: dev123
    teams:
      - Developers

apiVersion: 1
folders:
  - name: Production
    uid: production
  - name: Development
    uid: development

apiVersion: 1
permissions:
  - folderUid: production
    teamId: 1  # DevOps team
    permission: 4  # Edit
  
  - folderUid: production
    teamId: 2  # Developers team
    permission: 2  # View
  
  - folderUid: development
    teamId: 2  # Developers team
    permission: 4  # Edit
```

### LDAP 集成

```ini
# grafana.ini
[auth.ldap]
enabled = true
config_file = /etc/grafana/ldap.toml

# ldap.toml
[[servers]]
host = "ldap.example.com"
port = 389
use_ssl = false
start_tls = false

bind_dn = "cn=admin,dc=example,dc=com"
bind_password = 'password'

search_filter = "(cn=%s)"
search_base_dns = ["dc=example,dc=com"]

[servers.attributes]
name = "givenName"
surname = "sn"
username = "cn"
member_of = "memberOf"
email =  "email"

[[servers.group_mappings]]
group_dn = "cn=admins,ou=groups,dc=example,dc=com"
org_role = "Admin"

[[servers.group_mappings]]
group_dn = "cn=developers,ou=groups,dc=example,dc=com"
org_role = "Editor"

[[servers.group_mappings]]
group_dn = "*"
org_role = "Viewer"
```

### OAuth 集成

```ini
# grafana.ini
[auth.google]
enabled = true
client_id = xxx
client_secret = xxx
scopes = https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/userinfo.email
auth_url = https://accounts.google.com/o/oauth2/auth
token_url = https://accounts.google.com/o/oauth2/token
allowed_domains = example.com
allow_sign_up = true

[auth.github]
enabled = true
client_id = xxx
client_secret = xxx
scopes = user:email,read:org
auth_url = https://github.com/login/oauth/authorize
token_url = https://github.com/login/oauth/access_token
api_url = https://api.github.com/user
team_ids = 1,2
allowed_organizations = my-org
```

## 五、性能优化

### 查询优化

**减少查询数量**:

```json
// 错误:每个 Panel 独立查询
{
  "targets": [
    {"expr": "rate(http_requests_total{status=\"200\"}[5m])"},
    {"expr": "rate(http_requests_total{status=\"404\"}[5m])"},
    {"expr": "rate(http_requests_total{status=\"500\"}[5m])"}
  ]
}

// 正确:使用变量或聚合
{
  "targets": [
    {"expr": "sum by (status) (rate(http_requests_total[5m]))"}
  ]
}
```

**使用 Recording Rules**:

```yaml
# Prometheus Recording Rules
groups:
  - name: grafana_rules
    rules:
      - record: job:http_requests:rate5m
        expr: sum by (job) (rate(http_requests_total[5m]))
```

```json
// Grafana 查询使用预计算指标
{
  "targets": [
    {"expr": "job:http_requests:rate5m"}
  ]
}
```

### 缓存配置

```ini
# grafana.ini
[cache]
enabled = true
type = redis

[cache.redis]
host = redis:6379
password = ""
db = 0

[dataproxy]
# 数据源查询缓存
timeout = 30
send_user_header = false

# Dashboard 缓存
[dashboards]
versions_to_keep = 20
min_refresh_interval = 5s
```

### 数据库优化

```ini
# grafana.ini
[database]
type = postgres
host = postgres:5432
name = grafana
user = grafana
password = password
ssl_mode = disable
max_open_conn = 100
max_idle_conn = 50
conn_max_lifetime = 14400

[session]
provider = postgres
provider_config = host=postgres port=5432 user=grafana password=password dbname=grafana sslmode=disable
cookie_name = grafana_sess
cookie_secure = false
session_life_time = 86400
```

### 高可用部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: monitoring
spec:
  replicas: 2
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
        - name: grafana
          image: grafana/grafana:latest
          ports:
            - containerPort: 3000
          env:
            - name: GF_SECURITY_ADMIN_USER
              value: admin
            - name: GF_SECURITY_ADMIN_PASSWORD
              value: admin123
            - name: GF_DATABASE_TYPE
              value: postgres
            - name: GF_DATABASE_HOST
              value: postgres:5432
            - name: GF_DATABASE_NAME
              value: grafana
            - name: GF_DATABASE_USER
              value: grafana
            - name: GF_DATABASE_PASSWORD
              value: password
          volumeMounts:
            - name: config
              mountPath: /etc/grafana
            - name: dashboards
              mountPath: /var/lib/grafana/dashboards
          resources:
            requests:
              cpu: 250m
              memory: 512Mi
            limits:
              cpu: 1
              memory: 2Gi
      volumes:
        - name: config
          configMap:
            name: grafana-config
        - name: dashboards
          persistentVolumeClaim:
            claimName: grafana-dashboards
```

## 小结

- **Dashboard 设计**:遵循层次化布局、视觉一致性、交互性原则,合理使用变量和 Panel 类型
- **数据源配置**:支持多种数据源(Prometheus、Elasticsearch、MySQL),配置合理的查询参数
- **告警集成**:配置 Grafana Alert 或集成 Alertmanager,设置通知渠道(Slack、Email、PagerDuty)
- **权限管理**:使用组织和团队隔离权限,集成 LDAP 或 OAuth 实现统一认证
- **性能优化**:减少查询数量、使用缓存、优化数据库配置、高可用部署

---

## 常见问题

### Q1:Grafana 如何实现 Dashboard 版本控制?

**方案一:Grafana 内置版本控制**:

```ini
# grafana.ini
[dashboards]
versions_to_keep = 50
```

**方案二:Git 版本控制**:

```bash
# 导出 Dashboard
grafana-cli dashboards export --output /tmp/dashboards

# 或使用 API
curl -H "Authorization: Bearer xxx" \
  http://grafana:3000/api/dashboards/db/my-dashboard | jq '.dashboard' > my-dashboard.json

# 导入 Dashboard
curl -X POST -H "Authorization: Bearer xxx" \
  -H "Content-Type: application/json" \
  -d @my-dashboard.json \
  http://grafana:3000/api/dashboards/db
```

**方案三:Grafana Provisioning**:

```yaml
# provisioning/dashboards/dashboards.yaml
apiVersion: 1
providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards
```

### Q2:Grafana 如何实现 Dashboard 共享?

**快照共享**:

```bash
# 创建快照
curl -X POST -H "Authorization: Bearer xxx" \
  -H "Content-Type: application/json" \
  -d '{"dashboard": {...}, "expires": 3600}' \
  http://grafana:3000/api/snapshots

# 返回快照 URL
{
  "deleteKey": "xxx",
  "deleteUrl": "http://grafana:3000/api/snapshots-delete/xxx",
  "key": "yyy",
  "url": "http://grafana:3000/dashboard/snapshot/yyy"
}
```

**公开 Dashboard**:

```ini
# grafana.ini
[auth.anonymous]
enabled = true
org_name = Main Org.
org_role = Viewer
```

**嵌入外部网站**:

```html
<iframe 
  src="http://grafana:3000/d/xxx?orgId=1&kiosk=tv" 
  width="100%" 
  height="600" 
  frameborder="0">
</iframe>
```

### Q3:Grafana 如何优化 Dashboard 加载速度?

**优化策略**:

1. **减少 Panel 数量**:

```json
// 错误:30 个 Panel
// 正确:10-15 个 Panel,使用 Row 折叠
```

2. **使用变量**:

```json
// 错误:每个 Panel 独立查询
// 正确:使用变量共享查询
{
  "targets": [
    {"expr": "rate(http_requests_total{namespace=\"$namespace\"}[5m])"}
  ]
}
```

3. **增加刷新间隔**:

```json
{
  "refresh": "1m"  // 不要设置过短,如 5s
}
```

4. **使用缓存**:

```ini
# grafana.ini
[cache]
enabled = true
type = redis
```

5. **优化查询**:

```promql
// 错误:查询所有数据
rate(http_requests_total[5m])

// 正确:使用标签过滤
rate(http_requests_total{namespace="production"}[5m])
```

### Q4:Grafana 如何实现多租户?

**方案一:组织隔离**:

```yaml
# 创建多个组织
apiVersion: 1
orgs:
  - name: Team A
    id: 1
  - name: Team B
    id: 2

# 用户分配到组织
apiVersion: 1
users:
  - name: user1
    email: user1@example.com
    login: user1
    password: password
    orgId: 1
  
  - name: user2
    email: user2@example.com
    login: user2
    password: password
    orgId: 2
```

**方案二:团队权限**:

```yaml
apiVersion: 1
teams:
  - name: Team A
    email: team-a@example.com
  - name: Team B
    email: team-b@example.com

apiVersion: 1
permissions:
  - folderUid: team-a
    teamId: 1
    permission: 4  # Edit
  
  - folderUid: team-b
    teamId: 2
    permission: 4  # Edit
```

### Q5:Grafana 如何备份和恢复?

**备份配置**:

```bash
#!/bin/bash
# Grafana 备份脚本

GRAFANA_URL="http://grafana:3000"
API_KEY="xxx"
BACKUP_DIR="/backup/grafana/$(date +%Y%m%d)"

mkdir -p $BACKUP_DIR

# 备份 Dashboard
for uid in $(curl -s -H "Authorization: Bearer $API_KEY" \
  $GRAFANA_URL/api/search?type=dash-db | jq -r '.[].uid'); do
  curl -s -H "Authorization: Bearer $API_KEY" \
    $GRAFANA_URL/api/dashboards/uid/$uid > $BACKUP_DIR/dashboard-$uid.json
done

# 备份数据源
curl -s -H "Authorization: Bearer $API_KEY" \
  $GRAFANA_URL/api/datasources > $BACKUP_DIR/datasources.json

# 备份告警规则
curl -s -H "Authorization: Bearer $API_KEY" \
  $GRAFANA_URL/api/ruler/grafana/api/v1/rules > $BACKUP_DIR/alerts.json

echo "Backup completed: $BACKUP_DIR"
```

**恢复配置**:

```bash
#!/bin/bash
# Grafana 恢复脚本

GRAFANA_URL="http://grafana:3000"
API_KEY="xxx"
BACKUP_DIR="/backup/grafana/20260311"

# 恢复 Dashboard
for file in $BACKUP_DIR/dashboard-*.json; do
  dashboard=$(jq '.dashboard' $file)
  curl -X POST -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"dashboard\": $dashboard, \"overwrite\": true}" \
    $GRAFANA_URL/api/dashboards/db
done

# 恢复数据源
for ds in $(jq -c '.[]' $BACKUP_DIR/datasources.json); do
  curl -X POST -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d "$ds" \
    $GRAFANA_URL/api/datasources
done

echo "Restore completed"
```

## 参考资源

- [Grafana 官方文档](https://grafana.com/docs/grafana/latest/)
- [Grafana Dashboard 最佳实践](https://grafana.com/docs/grafana/latest/dashboards/)
- [Grafana Provisioning](https://grafana.com/docs/grafana/latest/administration/provisioning/)
- [Grafana API](https://grafana.com/docs/grafana/latest/http_api/)
- [Grafana Plugins](https://grafana.com/grafana/plugins/)

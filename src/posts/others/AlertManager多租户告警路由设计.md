---
date: 2026-02-11
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - others
tag:
  - others
  - ClaudeCode
---

# AlertManager 如何设计多租户告警：不同团队的告警路由到不同渠道

## 问题背景

当一个公司的所有团队共用同一套 Prometheus + Alertmanager 时，必然面临这样的需求：基础设施团队的磁盘告警发到 `#infra-alert` 频道，后端团队的服务错误率告警发到 `#backend-alert` 频道，而 critical 级别的告警同时呼叫 PagerDuty 的值班人员。

这本质上是一个**多租户路由**问题：一套 Alertmanager，多组接收方，告警按归属分流。Alertmanager 通过**路由树（Routing Tree）+ 标签匹配**机制来实现这一目标，本文深入分析其工作原理与实践设计。

## 路由树：多租户路由的基础

Alertmanager 的配置核心是一棵**路由树**，所有告警从根节点（`route`）出发，沿树向下寻找匹配的子节点，最终确定发送给哪个 Receiver。

```yaml
route:                         # 根节点（必须有 receiver，作为兜底）
  receiver: default-receiver
  routes:                      # 子节点列表
    - matchers: [...]
      receiver: team-a-receiver
      routes:                  # 可以继续嵌套
        - matchers: [...]
          receiver: team-a-critical-receiver
    - matchers: [...]
      receiver: team-b-receiver
```

### 路由匹配规则

告警从根节点开始向下匹配，**默认策略是找到第一个匹配的子节点就停止**，不再继续遍历同级的其他子节点。

```
告警到达
   ↓
根节点（route）
   ├── 子节点1：matchers=[team=infra]     ← 匹配到 → 发送，停止
   ├── 子节点2：matchers=[team=backend]   ← 如果子节点1不匹配才会检查
   └── 子节点3：matchers=[team=frontend]
```

如果没有任何子节点匹配，告警落回根节点的 `receiver` 处理，这就是为什么根节点必须配置 `receiver` —— 它是**所有未匹配告警的兜底接收方**。漏掉兜底 receiver 会导致告警被静默丢弃，这是生产事故的常见原因。

### 节点配置继承

子节点会继承父节点的配置，并可以选择性覆盖。可继承的字段包括：`group_by`、`group_wait`、`group_interval`、`repeat_interval`。

```yaml
route:
  group_wait: 30s          # 子节点默认继承 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: default

  routes:
    - matchers: [severity=critical]
      group_wait: 10s      # 覆盖为 10s，其余继承父节点
      receiver: pagerduty
```

## 标签：多租户路由的信息载体

路由匹配完全依赖**告警的标签（Labels）**。告警是否被路由到正确的团队，取决于它是否携带了正确的团队标识标签。

### 标签的来源

**来源一：Prometheus 告警规则中的 labels 字段**

这是最直接也是最推荐的方式，在写告警规则时就明确标注归属：

```yaml
# Prometheus alert rules
groups:
  - name: infra.rules
    rules:
      - alert: NodeDiskFull
        expr: disk_free_percent < 10
        for: 5m
        labels:
          severity: critical
          team: infra          # ← 明确标注所属团队
          channel: pagerduty   # ← 可以直接标注期望的通知渠道
        annotations:
          summary: "磁盘空间不足 {{ $labels.instance }}"

  - name: backend.rules
    rules:
      - alert: HighErrorRate
        expr: rate(http_errors_total[5m]) > 0.05
        labels:
          severity: warning
          team: backend
          service: api-gateway
```

**来源二：Prometheus 的 external_labels**

Prometheus 实例级别的全局标签，会附加到该实例产生的所有告警上：

```yaml
# prometheus.yml
global:
  external_labels:
    cluster: prod-east
    datacenter: dc1
```

这适合用于标识告警来自哪个集群或环境，配合路由实现按集群分发告警。

**来源三：Recording Rules 中间层**

当不便修改原始告警规则（如第三方 Exporter 提供的默认规则）时，可以通过 Recording Rules 添加元数据标签。不过更常见的做法是直接在告警规则中的 `labels` 字段添加，而不是借助中间层。

### 标签的设计原则

多租户路由的标签设计建议保持以下几个维度：

| 标签名 | 用途 | 示例值 |
|--------|------|--------|
| `team` | 告警归属团队 | `infra`, `backend`, `frontend`, `data` |
| `severity` | 告警严重程度 | `critical`, `warning`, `info` |
| `service` | 所属服务 | `api-gateway`, `payment`, `user-service` |
| `env` | 环境 | `prod`, `staging`, `dev` |

`team` 是多租户路由最核心的标签，`severity` 用于在团队内部做二次分流（critical 发 PagerDuty，warning 发 Slack）。

## 路由树的实战设计

### 基础多团队路由

最简单的多租户设计：按 `team` 标签将告警路由到不同 Slack 频道：

```yaml
route:
  receiver: default-receiver    # 兜底：无 team 标签的告警
  group_by: ['alertname', 'team']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

  routes:
    - matchers:
        - team="infra"
      receiver: infra-slack

    - matchers:
        - team="backend"
      receiver: backend-slack

    - matchers:
        - team="frontend"
      receiver: frontend-slack

receivers:
  - name: default-receiver
    slack_configs:
      - api_url: 'https://hooks.slack.com/...'
        channel: '#alert-uncategorized'

  - name: infra-slack
    slack_configs:
      - api_url: 'https://hooks.slack.com/...'
        channel: '#infra-alert'

  - name: backend-slack
    slack_configs:
      - api_url: 'https://hooks.slack.com/...'
        channel: '#backend-alert'

  - name: frontend-slack
    slack_configs:
      - api_url: 'https://hooks.slack.com/...'
        channel: '#frontend-alert'
```

### 团队内按严重程度二次分流

生产中更常见的需求：critical 告警呼人（PagerDuty），warning 告警发 Slack：

```yaml
route:
  receiver: default-receiver
  group_by: ['alertname', 'team']
  group_wait: 30s

  routes:
    - matchers:
        - team="infra"
      receiver: infra-slack         # 兜底：infra 的 warning/info 发 Slack
      group_wait: 30s
      routes:
        - matchers:
            - severity="critical"
          receiver: infra-pagerduty  # infra critical 发 PagerDuty
          group_wait: 10s            # critical 响应更快

    - matchers:
        - team="backend"
      receiver: backend-slack
      routes:
        - matchers:
            - severity="critical"
          receiver: backend-pagerduty

receivers:
  - name: infra-slack
    slack_configs:
      - channel: '#infra-alert'
        title: '[{{ .Status | toUpper }}] {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'

  - name: infra-pagerduty
    pagerduty_configs:
      - service_key: '<infra-team-pagerduty-key>'

  - name: backend-slack
    slack_configs:
      - channel: '#backend-alert'

  - name: backend-pagerduty
    pagerduty_configs:
      - service_key: '<backend-team-pagerduty-key>'
```

路由树示意：

```
根节点
 ├── team=infra
 │    ├── severity=critical → infra-pagerduty
 │    └── (其他) → infra-slack        ← 父节点的 receiver 作为兜底
 ├── team=backend
 │    ├── severity=critical → backend-pagerduty
 │    └── (其他) → backend-slack
 └── (无匹配) → default-receiver
```

### continue 标志：向多个接收方同时发送

默认情况下，一条告警匹配到子节点后就停止遍历。`continue: true` 打破这一行为，让告警在匹配当前节点后**继续向后匹配同级节点**：

```yaml
route:
  receiver: default-receiver
  routes:
    # 所有 critical 告警额外发送给管理层
    - matchers:
        - severity="critical"
      receiver: management-email
      continue: true             # ← 匹配后继续往下走

    # 正常的团队路由
    - matchers:
        - team="infra"
      receiver: infra-slack

    - matchers:
        - team="backend"
      receiver: backend-slack
```

这样，一条 `{team=infra, severity=critical}` 的告警会同时发给 `management-email`（第一个节点，continue=true）和 `infra-slack`（第二个节点，team=infra 匹配）。

`continue` 的典型使用场景：
- **全局监控**：某个中央运维团队需要接收所有 critical 告警的汇总
- **审计日志**：所有告警通过 Webhook 写入统一的告警记录系统
- **跨团队告知**：某个故障需要同时通知多个相关团队

### 多维度路由：环境与团队的组合

当需要按照 `env + team` 两个维度路由时：

```yaml
route:
  receiver: default-receiver

  routes:
    # prod 环境单独处理，优先级更高
    - matchers:
        - env="prod"
        - team="infra"
      receiver: infra-prod-pagerduty
      group_wait: 10s

    - matchers:
        - env="prod"
        - team="backend"
      receiver: backend-prod-pagerduty
      group_wait: 10s

    # staging/dev 环境统一发 Slack，不呼人
    - matchers:
        - env=~"staging|dev"
        - team="infra"
      receiver: infra-nonprod-slack

    - matchers:
        - env=~"staging|dev"
        - team="backend"
      receiver: backend-nonprod-slack

    # 兜底：prod 但无 team 标签
    - matchers:
        - env="prod"
      receiver: prod-default
```

注意：Alertmanager 的 matchers 支持正则匹配（`=~`），等效于 Prometheus 的标签选择器语法。

## Receiver 配置深入

### Slack Receiver 的消息模板

Alertmanager 使用 Go template 语法定制通知内容，合理的模板设计能让告警通知包含足够的上下文：

```yaml
receivers:
  - name: backend-slack
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/xxx'
        channel: '#backend-alert'
        send_resolved: true          # 告警恢复时也发通知
        title: >-
          [{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}]
          {{ .GroupLabels.alertname }}
        text: |
          {{ range .Alerts }}
          *告警*: {{ .Labels.alertname }}
          *服务*: {{ .Labels.service }}
          *实例*: {{ .Labels.instance }}
          *描述*: {{ .Annotations.summary }}
          *详情*: {{ .Annotations.description }}
          ---
          {{ end }}
        color: '{{ if eq .Status "firing" }}danger{{ else }}good{{ end }}'
```

### PagerDuty Receiver 按团队隔离

每个团队在 PagerDuty 中拥有独立的 Service，各自管理值班轮换：

```yaml
receivers:
  - name: infra-pagerduty
    pagerduty_configs:
      - service_key: 'infra-team-integration-key'
        description: '{{ .GroupLabels.alertname }}: {{ .CommonAnnotations.summary }}'
        severity: '{{ .CommonLabels.severity }}'
        details:
          team: '{{ .CommonLabels.team }}'
          cluster: '{{ .CommonLabels.cluster }}'

  - name: backend-pagerduty
    pagerduty_configs:
      - service_key: 'backend-team-integration-key'
```

### Webhook Receiver 统一告警记录

通过 Webhook 将所有告警写入集中式记录系统（如 ES/数据库），配合 `continue: true` 实现旁路审计：

```yaml
receivers:
  - name: alert-audit-webhook
    webhook_configs:
      - url: 'http://alert-recorder.internal/api/alerts'
        send_resolved: true
        http_config:
          bearer_token: 'your-token'
```

## 多租户路由的常见陷阱

**陷阱一：根节点没有配置 receiver**

```yaml
# 错误写法
route:
  # 忘记写 receiver
  routes:
    - matchers: [team=infra]
      receiver: infra-slack
```

没有任何子节点匹配的告警（如告警规则忘记加 `team` 标签）会被**静默丢弃**，不产生任何通知，也不报错。这是最危险的配置错误。

**陷阱二：子节点顺序决定匹配优先级**

Alertmanager 从上到下遍历子节点，第一个匹配的节点胜出。如果将宽泛的匹配条件放在前面，后面的规则可能永远不会被执行：

```yaml
routes:
  # 错误：这条太宽泛，会吞掉所有告警
  - matchers:
      - severity=~"critical|warning|info"
    receiver: catch-all

  # 这条永远不会被匹配到
  - matchers:
      - team="infra"
    receiver: infra-slack
```

正确做法：将越具体的匹配条件放在越前面，越宽泛的放在越后面。

**陷阱三：matchers 语法使用错误**

Alertmanager 0.22+ 使用新版 matchers 语法，与旧版 `match`/`match_re` 不同。新版语法更接近 PromQL 标签选择器：

```yaml
# 旧版（仍支持但不推荐）
match:
  team: infra
match_re:
  severity: critical|warning

# 新版（推荐）
matchers:
  - team="infra"
  - severity=~"critical|warning"
```

**陷阱四：忽视 continue 与分组的交互**

当 `continue: true` 的节点有自己的 `group_by` 配置时，同一条告警可能在不同节点形成不同的分组，导致重复通知。通常让 `continue` 节点继承父节点的分组配置即可。

## 实际架构建议

**建议一：标签规范先行**

在设计路由规则之前，先在团队内部约定标签规范（团队名称、severity 等级、服务命名），并通过 Prometheus Recording Rule 的 lint 工具或 CI 检查强制执行，否则路由规则形同虚设。

**建议二：分层路由，保持路由树扁平**

路由树嵌套过深难以维护，建议最多两层：第一层按团队分流，第二层按严重程度分流。更复杂的需求通过丰富的标签 + Receiver 内部模板来解决，而不是通过加深路由树层次。

**建议三：始终配置 send_resolved**

告警恢复通知（`send_resolved: true`）是闭环机制的关键，否则接收方无法知道问题是否已被修复，需要人工确认告警状态。

**建议四：用 amtool 验证路由**

Alertmanager 提供了 `amtool` 命令行工具，可以在不实际触发告警的情况下验证路由配置：

```bash
# 测试一条携带指定标签的告警会被路由到哪里
amtool config routes test --config.file=alertmanager.yml \
  team=infra severity=critical alertname=NodeDown

# 输出：
# Routing to: infra-pagerduty
# Continue: false
```

这在配置变更前的验证阶段非常有价值，避免路由配置上线后才发现告警发错地方。

## 总结

Alertmanager 多租户告警路由的本质是：**通过标签携带元数据，通过路由树实现条件分发**。整套设计可以归纳为三个层次：

```
第一层：约定标签体系（team/severity/service/env）
    ↓
第二层：Prometheus 告警规则在 labels 中注入标签
    ↓
第三层：Alertmanager 路由树按标签分流到各团队 Receiver
```

标签体系是基础，没有规范的标签，路由规则无法准确匹配；路由树是机制，它决定了告警的流向；Receiver 是终点，它负责最终的通知送达。三者缺一不可，协同构成了多租户告警分发体系。

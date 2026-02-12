---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: true
article: true
category:
  - Grafana
tag:
  - Grafana
  - Dashboard
  - Prometheus
  - 可观测性
---

# Grafana 实战：Dashboard 设计、变量联动与告警配置

## Dashboard 设计的三层模型

很多团队的 Dashboard 只有一层：把所有指标堆在一个页面上，Panel 越加越多，最终变成一张谁都看不懂的"指标大集合"。根本原因在于，没有在设计阶段想清楚**不同角色在不同场景下的信息需求是什么**。

一套可维护的 Dashboard 体系应该遵循三层设计原则：

```
Overview（概览层）
  ↓ 发现异常
Detail（详情层）
  ↓ 定位问题
Debug（排查层）
  ↓ 根因分析
```

**概览层**面向值班工程师，目标是在 5 秒内回答"系统现在是否健康"。这一层只展示 SLI 指标（错误率、延迟 P99、可用性），使用 Stat 和 Gauge 面板，配合阈值颜色（绿/黄/红）提供即时判断。面板数量控制在 10 个以内，一屏展示完毕。

**详情层**面向排查中的工程师，目标是在服务出现异常时快速定位是哪个维度（节点/命名空间/服务）出了问题。这一层使用时序图（Time Series）展示趋势，配合变量筛选，支持按 namespace、deployment 下钻。

**排查层**面向深度 Debug 场景，可以包含原始指标、分位数对比、容器级别的 CPU Throttling 明细等。这一层通常不需要常驻展示，用 Row 折叠，需要时展开。

### Panel 类型的选择逻辑

Panel 类型不是视觉偏好问题，而是**信息类型与展示方式的匹配**：

| Panel 类型 | 适用场景 | 典型指标 |
|-----------|---------|---------|
| Time Series | 趋势观察，发现异常时间点 | QPS、延迟、错误率随时间变化 |
| Stat | 单个当前值，强调状态 | 当前错误率、在线实例数 |
| Gauge | 当前值在范围内的位置 | CPU 使用率（0~100%） |
| Bar Chart | 多个维度的横向对比 | 各 Pod 内存用量排名 |
| Table | 结构化展示多维度信息 | Pod 列表+状态+重启次数 |

**信息密度原则**：概览层的 Panel 不超过 8 个，详情层每个 Row 不超过 6 个 Panel，排查层可以密集但必须折叠。一屏放不下的内容，优先考虑是否该拆成单独的 Dashboard，而不是无限滚动。

---

## 变量（Variables）使用实战

变量是 Dashboard 的交互核心。一个没有变量的 Dashboard 是"只读文档"，一个有合理变量设计的 Dashboard 才是真正可用的运维工具。

### Query 类型变量：从 Prometheus 动态拉取

Query 变量通过 PromQL 从 Prometheus 查询标签值，实现下拉选项的动态更新。最常用的是 `label_values()` 函数：

```
# 获取所有 namespace
label_values(kube_pod_info, namespace)

# 获取特定 namespace 下的所有 pod
label_values(kube_pod_info{namespace="$namespace"}, pod)

# 获取所有节点实例
label_values(node_cpu_seconds_total, instance)
```

在变量配置中，`Refresh` 选项决定变量何时重新查询 Prometheus：
- `On dashboard load`：适合变化不频繁的维度（namespace、cluster）
- `On time range change`：适合与时间范围相关的场景

### All 选项与 `$__all` 的处理

启用 `Include All option` 后，用户可以选择"全部"。但这里有一个关键细节：**All 的实际值不是字面量 `__all`，而是所有选项的正则联合**。

在变量配置的 `Custom all value` 中，如果不填，默认会展开为 `value1|value2|value3`。这意味着在 PromQL 中必须使用正则匹配符 `=~`：

```promql
# 错误用法：= 不支持多值
rate(http_requests_total{namespace="$namespace"}[5m])

# 正确用法：=~ 支持正则展开
rate(http_requests_total{namespace=~"$namespace"}[5m])
```

:::warning
对所有包含变量的 Label Matcher，一律使用 `=~` 而不是 `=`，这样无论用户选择单个值还是 All，查询都能正确工作。
:::

### 级联变量：先选 namespace 再选 pod

级联变量（Chained Variables）让变量之间建立依赖关系，后级变量的选项范围由前级变量的值决定。配置方法是在后级变量的 Query 中引用前级变量：

```
变量 $namespace（第一级）:
  Query: label_values(kube_pod_info, namespace)

变量 $pod（第二级，依赖 $namespace）:
  Query: label_values(kube_pod_info{namespace=~"$namespace"}, pod)

变量 $container（第三级，依赖 $pod）:
  Query: label_values(kube_pod_container_info{namespace=~"$namespace", pod=~"$pod"}, container)
```

当用户选择 `$namespace = production` 后，`$pod` 的下拉选项自动过滤为仅属于 `production` 的 Pod，再选 pod 后，`$container` 进一步过滤。

:::tip
在变量配置页，开启 `Multi-value` 后，要同步修改 `Regex` 字段来过滤掉不需要的选项，避免下拉列表过长。
:::

### `$__rate_interval` 的特殊用途

Grafana 内置了几个特殊变量，其中 `$__rate_interval` 专为 `rate()` 函数设计。它的值由 Grafana 自动计算：**至少是 Prometheus 抓取间隔的 4 倍**，确保 `rate()` 窗口内总有足够的样本点。

```promql
# 不推荐：固定窗口可能在抓取间隔较大时失效
rate(http_requests_total{namespace=~"$namespace"}[5m])

# 推荐：自适应窗口
rate(http_requests_total{namespace=~"$namespace"}[$__rate_interval])
```

`$__interval` 和 `$__rate_interval` 的区别在于：`$__interval` 是"当前时间范围对应的合适间隔"，适合 `avg_over_time` 类函数；`$__rate_interval` 专为 `rate/irate` 优化，包含了最小窗口保护。

### 完整的变量配置 JSON 示例

以下是一个 namespace + pod 级联变量的 Dashboard JSON 片段：

```json
{
  "templating": {
    "list": [
      {
        "name": "namespace",
        "type": "query",
        "datasource": "${datasource}",
        "query": "label_values(kube_pod_info, namespace)",
        "refresh": 1,
        "multi": true,
        "includeAll": true,
        "allValue": ".+",
        "sort": 1
      },
      {
        "name": "pod",
        "type": "query",
        "datasource": "${datasource}",
        "query": "label_values(kube_pod_info{namespace=~\"$namespace\"}, pod)",
        "refresh": 2,
        "multi": true,
        "includeAll": true,
        "allValue": ".+",
        "sort": 1
      }
    ]
  }
}
```

注意 `allValue` 设置为 `.+` 而非空，这让 PromQL 中的 `=~` 匹配所有非空值，等效于"不过滤"。

---

## Kubernetes 集群监控 Dashboard 实战

### 核心指标与 PromQL

**节点 CPU 使用率**

```promql
# 单节点 CPU 使用率（排除 idle 模式）
1 - avg by (instance) (
  rate(node_cpu_seconds_total{mode="idle", instance=~"$instance"}[$__rate_interval])
)
```

**内存 OOM 风险（内存使用率 > 85%）**

```promql
# 节点内存使用率
1 - (
  node_memory_MemAvailable_bytes{instance=~"$instance"}
  /
  node_memory_MemTotal_bytes{instance=~"$instance"}
)
```

**Pod 重启次数异常（过去 1 小时内重启超过 3 次）**

```promql
# 最近 1 小时内重启超过 3 次的 Pod
increase(kube_pod_container_status_restarts_total{
  namespace=~"$namespace"
}[1h]) > 3
```

**Deployment 副本不足（期望副本数与就绪副本数不一致）**

```promql
# 就绪副本不足的 Deployment
kube_deployment_status_replicas_ready{namespace=~"$namespace"}
  <
kube_deployment_spec_replicas{namespace=~"$namespace"}
```

### Row 折叠组织

用 Row 将 Panel 按观测维度分组，平时折叠，需要时展开。推荐的分组方式：

```
Row: 集群概览（默认展开）
  - 节点就绪数量、Pod 总数、告警数量

Row: 节点资源（默认折叠）
  - 各节点 CPU/内存/磁盘 使用率热力图

Row: 工作负载健康（默认展开）
  - Deployment 副本状态表、Pod 重启排行

Row: 网络与存储（默认折叠）
  - 节点网络流量、PVC 使用率
```

---

## Grafana Alerting 配置

### 新版告警架构

Grafana 10+ 的 Unified Alerting 将告警完全内化，不再依赖外部 AlertManager 也能独立运作。架构由三个层次组成：

```
Alert Rule（告警规则）
  ↓ 触发告警
Contact Point（联系点：Webhook/Email/Slack/PagerDuty）
  ↑
Notification Policy（通知策略：路由规则）
```

**Alert Rule** 定义"什么条件触发告警"，包含 PromQL 表达式、评估间隔、pending 期。

**Contact Point** 定义"告警发到哪里"，支持 Slack、Email、Webhook、PagerDuty 等。

**Notification Policy** 定义"哪类告警路由到哪个 Contact Point"，通过标签匹配规则实现分级路由（类似 AlertManager 的 routes 配置）。

### 与 Prometheus AlertManager 的关系

:::tip
Grafana Unified Alerting 和 Prometheus AlertManager 不是互斥的，可以共存。推荐的模式是：

- Prometheus AlertManager：处理基础设施层告警（节点宕机、磁盘满），由 Prometheus Recording Rule 驱动
- Grafana Alerting：处理业务层告警（SLO 违反、特定服务错误率），在 Dashboard 上可视化

两者都可以将通知推送到同一个 Slack 频道，通过 `source` 标签区分来源。
:::

### 告警规则的关键参数

**Pending 期（For）**：告警条件持续满足多久后才真正触发。设置 `For: 5m` 意味着指标超阈值后需要连续 5 分钟才产生告警，避免瞬时抖动。

**评估间隔（Evaluation Interval）**：每隔多久重新评估一次规则。通常设为 1m，与 Prometheus 抓取间隔对齐。

**告警标签（Labels）**：用于 Notification Policy 的路由匹配，例如 `severity: critical` 路由到 PagerDuty，`severity: warning` 路由到 Slack。

### 完整告警规则 YAML 示例

以下是通过 Grafana Provisioning 定义告警规则的 YAML 格式：

```yaml
apiVersion: 1

groups:
  - orgId: 1
    name: kubernetes-alerts
    folder: Kubernetes
    interval: 1m
    rules:
      - uid: pod-restart-alert
        title: Pod 频繁重启
        condition: B
        data:
          - refId: A
            datasourceUid: prometheus
            model:
              expr: |
                increase(kube_pod_container_status_restarts_total[1h]) > 3
              intervalMs: 1000
              maxDataPoints: 43200
          - refId: B
            datasourceUid: __expr__
            model:
              type: threshold
              conditions:
                - evaluator:
                    params: [0]
                    type: gt
        noDataState: NoData
        execErrState: Error
        for: 5m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "Pod {{ $labels.pod }} 在过去 1 小时内重启超过 3 次"
          description: "命名空间 {{ $labels.namespace }} 中的 Pod {{ $labels.pod }} 容器 {{ $labels.container }} 重启次数异常"
```

### 静默（Silence）与维护窗口

**Silence** 用于临时屏蔽已知告警。在 Grafana 的 Alerting → Silences 页面，通过标签匹配创建静默规则，设置开始和结束时间。例如计划维护期间，静默所有 `team=platform` 的告警：

```
Matchers:
  team = platform
Start: 2026-02-14 02:00
End:   2026-02-14 04:00
Comment: 定期维护窗口，预期服务重启
```

:::warning
Silence 只屏蔽通知，不会停止规则评估。维护结束后，规则状态仍会正常恢复。如果需要完全停止评估（减少 Prometheus 查询压力），应暂停告警规则（Pause Rule）。
:::

---

## Dashboard Provisioning：代码化管理

### 通过 ConfigMap 自动加载 Dashboard

在 Kubernetes 环境中，最佳实践是将 Dashboard JSON 存入 ConfigMap，通过 Grafana 的 Provisioning 机制自动加载，避免手动导入。

Grafana 的 Provisioning 配置（`provisioning/dashboards/default.yaml`）：

```yaml
apiVersion: 1

providers:
  - name: kubernetes-dashboards
    orgId: 1
    type: file
    disableDeletion: false
    updateIntervalSeconds: 30
    allowUiUpdates: false
    options:
      path: /var/lib/grafana/dashboards
      foldersFromFilesStructure: true
```

对应的 Kubernetes ConfigMap：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-dashboards-kubernetes
  namespace: monitoring
  labels:
    grafana_dashboard: "1"   # 配合 Grafana Sidecar 使用
data:
  kubernetes-overview.json: |
    {
      "title": "Kubernetes Overview",
      "uid": "k8s-overview",
      "panels": []
    }
```

如果使用 `kube-prometheus-stack` Helm Chart 部署 Grafana，可以通过 `grafana.sidecar.dashboards.enabled: true` 开启 Sidecar 模式，Sidecar 容器会监听带有特定 Label 的 ConfigMap，自动热加载 Dashboard，无需重启 Pod。

### Dashboard JSON 的版本控制

将 Dashboard JSON 纳入 Git 管理时，建议在 CI 流水线中加入格式检查和 uid 唯一性校验：

```bash
# 检查所有 Dashboard JSON 格式是否合法
for f in dashboards/*.json; do
  jq empty "$f" || echo "Invalid JSON: $f"
done

# 检查 uid 是否重复
jq -r '.uid' dashboards/*.json | sort | uniq -d
```

---

## 性能优化

### `$__rate_interval` vs `$__interval`

这是 Grafana 中最常见的误用场景。简单记忆：

- 使用 `rate()`、`irate()`、`increase()` → 用 `$__rate_interval`
- 使用 `avg_over_time()`、`max_over_time()` → 用 `$__interval`

`$__rate_interval` 保证窗口内至少有 4 个 Prometheus 样本，`rate()` 需要至少 2 个样本才能计算导数，4 个样本提供了足够的安全裕量。

### 避免全量扫描

每个 Panel 的查询都会向 Prometheus 发出一次 HTTP 请求。Panel 中的标签过滤越具体，Prometheus 需要扫描的时序越少：

```promql
# 低效：扫描所有时序
rate(http_requests_total[$__rate_interval])

# 高效：通过变量限制扫描范围
rate(http_requests_total{namespace=~"$namespace", job=~"$job"}[$__rate_interval])
```

### Recording Rule 预聚合

对于 Dashboard 中频繁查询的复杂表达式，应在 Prometheus 侧创建 Recording Rule，将结果预计算为新的指标：

```yaml
# prometheus-rules.yaml
groups:
  - name: aggregations
    rules:
      - record: namespace:http_requests:rate5m
        expr: |
          sum by (namespace, job) (
            rate(http_requests_total[5m])
          )
```

Dashboard 中的查询从原始指标改为查询预聚合结果：

```promql
# 原始查询（慢）
sum by (namespace) (rate(http_requests_total{namespace=~"$namespace"}[5m]))

# 预聚合查询（快）
namespace:http_requests:rate5m{namespace=~"$namespace"}
```

预聚合能将查询时间从秒级降到毫秒级，在 Dashboard 有大量 Panel 时效果显著。

---

## 权限管理

### 权限层次模型

Grafana 的权限体系从大到小依次为：

```
Organization（组织）
  └── Folder（文件夹）
        └── Dashboard
              └── Panel（不支持独立权限）

Team（团队）
  └── 成员 → 绑定到 Folder 或 Dashboard
```

实践建议：
- 按业务线或团队创建 Folder：`平台组/`、`业务组/`、`SRE/`
- 为每个团队创建对应的 Team，将 Folder 的 Edit 权限赋予对应 Team
- 只读用户（如管理层）在 Organization 层面设置 Viewer 角色，通过 Folder 权限覆盖实现按需升级

### Service Account 用于 API 访问

在 CI/CD 流水线或外部系统需要通过 Grafana API 自动操作（上传 Dashboard、创建 Snapshot）时，应使用 Service Account 而非个人用户凭据：

```bash
# 创建 Service Account
curl -X POST http://grafana:3000/api/serviceaccounts \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-pipeline", "role": "Editor"}'

# 为 Service Account 创建 Token
curl -X POST http://grafana:3000/api/serviceaccounts/1/tokens \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-token"}'
```

Service Account Token 可以设置过期时间，支持按 Folder 精细授权，比直接使用 admin 账号安全得多。

---

## 小结

- Dashboard 设计遵循三层模型（概览→详情→排查），避免信息堆砌
- 变量使用 `=~` 匹配符适配多值选择，`allValue: .+` 实现 All 的正确语义
- 级联变量通过在 Query 中引用上级变量实现下钻过滤
- `$__rate_interval` 专为 rate 函数设计，自动保证样本数量充足
- Grafana Unified Alerting 通过 Alert Rule → Notification Policy → Contact Point 三层路由告警
- Dashboard JSON 纳入 Git 管理，通过 ConfigMap + Provisioning 实现声明式部署
- Recording Rule 是 Dashboard 性能优化的最有效手段

---

## 常见问题

### Q1：Dashboard 变量选择 All 后 PromQL 查询报错或返回空数据，如何排查？

最常见的原因有两个：第一，PromQL 中使用了 `=` 而不是 `=~`。当用户选择 All 时，变量展开为 `value1|value2|...` 这样的正则表达式，`=` 不支持正则，必须改为 `=~`。第二，`allValue` 配置不当。如果 `Custom all value` 留空，Grafana 默认展开所有值并用 `|` 连接，但如果某个值包含特殊字符，可能导致正则解析失败。推荐将 `allValue` 设置为 `.+`，配合 `=~` 使用，`.+` 匹配所有非空字符串，等效于"不过滤"。排查时可以在变量配置页点击 "Preview of values" 查看实际展开的值，再对照 PromQL 分析匹配逻辑。

### Q2：Grafana 告警和 Prometheus AlertManager 应该如何选择，可以同时使用吗？

两者可以同时使用，且各有侧重。Prometheus AlertManager 更适合基础设施层的告警，因为它和 Prometheus 原生集成，支持复杂的路由树、抑制（Inhibit）、分组（Group）等高级功能，且告警数据不依赖 Grafana 服务的可用性。Grafana Unified Alerting 更适合业务层告警，优势是可以在 Dashboard 上直接关联告警规则，可视化告警状态，且支持非 Prometheus 数据源（如 Loki、InfluxDB）。两者同时使用时，建议通过告警标签（如 `source: prometheus` vs `source: grafana`）区分来源，避免同一告警在两个系统中重复触发通知。

### Q3：Dashboard 加载很慢（超过 10 秒），如何优化？

Dashboard 加载慢通常由三个原因导致：Panel 数量过多、查询时间范围过长、复杂 PromQL 没有预聚合。优化方向：第一，用 Row 折叠将非关键 Panel 隐藏，折叠状态下 Grafana 不会执行对应 Panel 的查询，可立即减少初始查询数量。第二，为 Dashboard 设置合理的默认时间范围（建议 3~6 小时而非 24 小时），减少每次查询的数据量。第三，对高频查询的复杂表达式创建 Recording Rule，将 Prometheus 侧的计算时间从秒级降到毫秒级。第四，检查是否有 Panel 在 `$__rate_interval` 极小时（例如时间范围只有 5 分钟）触发了大量查询，可以通过设置最小刷新间隔来规避。

### Q4：如何实现 Dashboard 的版本控制和多环境管理？

推荐的方案是将 Dashboard JSON 存入 Git 仓库，并通过 Provisioning 机制自动加载。具体步骤：将 Dashboard 从 Grafana UI 导出为 JSON，提交到 `dashboards/` 目录；在 Grafana Provisioning 配置中指定 JSON 目录；通过 Kubernetes ConfigMap 或 Helm Values 将 JSON 注入 Grafana Pod。多环境管理时，不同环境（dev/staging/prod）共享同一套 Dashboard JSON，通过变量的数据源（`$datasource`）切换到对应环境的 Prometheus 实例。需要注意 Dashboard JSON 中的 `datasource.uid` 字段，各环境的数据源 UID 可能不同，推荐使用数据源 provisioning 来保证各环境 UID 一致。

### Q5：Grafana 的 Silence 功能和 AlertManager 的 Silence 有什么区别？

两者都能屏蔽告警通知，但作用层次不同。Grafana Silence 作用于 Grafana Alerting 引擎生成的告警，通过标签匹配决定哪些告警在通知阶段被过滤掉；AlertManager Silence 作用于 AlertManager 收到的所有告警（包括来自 Prometheus 的），在路由阶段过滤。如果你同时使用两套系统，需要分别在各自的界面创建 Silence，否则来自另一系统的同名告警仍然会触发通知。另外，Grafana 的 Silence 不会停止规则评估，只屏蔽通知；如需完全暂停某条告警规则的执行，应在告警规则列表中点击 "Pause"。维护窗口场景建议通过 Silence 而非 Pause，这样维护结束后告警会自动恢复，无需手动操作。

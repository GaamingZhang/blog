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

# Prometheus Recording Rule 详解

## 什么是 Recording Rule

Recording Rule 是 Prometheus 的**预计算机制**：将一个 PromQL 表达式的计算结果定期写入一条新的时序数据，存储在 TSDB 中，供后续查询直接使用。

简单说：**把一个复杂查询的结果"物化"为一条新的指标**。

不同于告警规则（Alert Rule）在条件满足时触发通知，Recording Rule 的目的是**持续生成新的时序数据**，对外表现和普通采集的指标完全一样，可以被其他查询、Dashboard、告警规则引用。

### 基本配置格式

```yaml
# rules/aggregations.yml
groups:
  - name: example_aggregations
    interval: 1m        # 可选，覆盖全局 evaluation_interval
    rules:
      - record: job:http_requests_total:rate5m
        expr: sum by (job) (rate(http_requests_total[5m]))

      - record: job:http_errors_total:rate5m_ratio
        expr: |
          sum by (job) (rate(http_requests_total{status=~"5.."}[5m]))
          /
          sum by (job) (rate(http_requests_total[5m]))
        labels:
          aggregation: "true"    # 可选：给生成的时序附加额外标签
```

在 `prometheus.yml` 中加载规则文件：

```yaml
rule_files:
  - "rules/*.yml"
```

---

## 内部运行机制

理解 Recording Rule 的工作方式，是正确使用它的前提。

### 评估周期

Prometheus 有一个全局配置 `evaluation_interval`（默认 1m），这是所有规则的默认评估间隔。每隔一个间隔，Prometheus 执行一次规则评估：

```
t=0m   评估所有 rules → 写入 TSDB（时间戳 t=0m）
t=1m   评估所有 rules → 写入 TSDB（时间戳 t=1m）
t=2m   评估所有 rules → 写入 TSDB（时间戳 t=2m）
...
```

每次评估时，Prometheus 执行 `expr` 中的 PromQL，把结果以 `record` 指定的名称写入 TSDB。写入的时间戳是评估发生的时刻，不是查询窗口的终止时刻。

### 组内顺序评估

同一个 `group` 内的规则**按声明顺序依次执行**，后面的规则可以引用前面规则刚生成的指标：

```yaml
groups:
  - name: chained_rules
    rules:
      # 第一条：计算每个 job 的请求速率
      - record: job:http_requests:rate5m
        expr: sum by (job) (rate(http_requests_total[5m]))

      # 第二条：可以直接引用上面刚生成的指标（同一 group 内）
      - record: job:http_requests:rate5m_normalized
        expr: job:http_requests:rate5m / on(job) group_left job_weight
```

不同 `group` 之间的规则是**并行评估**的，跨组引用时存在时序上的细微差异（被引用的 group 可能还未完成当前轮评估），因此跨组的链式依赖需要谨慎，建议将有依赖关系的规则放在同一个 group 中。

### 数据延迟

Recording Rule 产生的数据相比原始采集数据，天然存在最多一个 `evaluation_interval` 的延迟。这对于实时性要求极高的场景（如秒级响应的 Dashboard）需要考虑，但对于大多数分钟级监控场景影响可以忽略。

### Staleness（过期标记）

如果某次评估时 `expr` 的结果为空（例如某个 job 的所有 pod 都消失了），Prometheus 会向该时序写入一个**过期标记（staleness marker）**，告知查询引擎这条时序已停止产生数据。这与普通指标的过期处理机制完全一致，避免 Dashboard 上出现"数据断连"后仍显示旧值的问题。

---

## 命名规范

Prometheus 官方推荐的 Recording Rule 命名格式为：

```
level:metric:operations
```

- **level**：聚合的维度粒度，如 `job`、`instance`、`cluster`、`namespace`
- **metric**：基础指标名称（去掉 `_total`、`_seconds` 等后缀），如 `http_requests`、`node_cpu`
- **operations**：应用的操作序列，如 `rate5m`、`sum`、`ratio`

示例：

| Recording Rule 名称 | 含义 |
|--------------------|------|
| `job:http_requests_total:rate5m` | 按 job 聚合的 5 分钟请求速率 |
| `job:http_errors:rate5m_ratio` | 按 job 聚合的 5 分钟错误率（比值） |
| `namespace:container_cpu_usage:sum` | 按 namespace 聚合的 CPU 用量总和 |
| `cluster:node_memory_available:avg` | 集群级别的平均可用内存 |

这套命名规范的价值在于：**看到名称就能推断计算逻辑**，团队协作时不需要每次都展开查看 `expr`。

---

## 四种核心使用场景

### 场景一：高基数查询的性能优化

这是 Recording Rule 最主要的使用场景。

**问题**：Kubernetes 集群中，`container_cpu_usage_seconds_total` 可能有数万个时序（每个 pod 每个 container 都有一条）。如果 Grafana 每次刷新都执行：

```promql
sum by (namespace) (
  rate(container_cpu_usage_seconds_total{container!=""}[5m])
)
```

Prometheus 需要遍历数万条时序，对每条做 `rate()` 计算，再聚合 —— 这在数据量大时会导致查询超时，也会给 Prometheus 带来较大的 CPU 压力。

**解决**：将这个聚合结果预计算为 Recording Rule：

```yaml
- record: namespace:container_cpu_usage_seconds_total:sum_rate5m
  expr: |
    sum by (namespace) (
      rate(container_cpu_usage_seconds_total{container!=""}[5m])
    )
```

Grafana 查询时直接读取 `namespace:container_cpu_usage_seconds_total:sum_rate5m`，数据已经是聚合好的，只有几十条时序（每个 namespace 一条），查询响应从秒级降到毫秒级。

**本质原理**：将查询代价从**读时计算**（query time）转移到**写时计算**（evaluation time）。写时计算在后台周期性执行，不影响用户请求；读时计算每次用户打开 Dashboard 都要发生，直接影响用户体验。

### 场景二：Dashboard 与告警的表达式一致性

**问题**：同一个业务指标（如"服务错误率"）在多个 Dashboard Panel 和多条告警规则中都要用到，如果每处都写一遍完整的 PromQL，改动时容易遗漏，也容易因细微差异导致 Dashboard 显示值和告警触发值不一致。

**解决**：将"服务错误率"的计算逻辑定义为一个 Recording Rule：

```yaml
- record: job:http_error_ratio:rate5m
  expr: |
    sum by (job) (rate(http_requests_total{status=~"5.."}[5m]))
    /
    sum by (job) (rate(http_requests_total[5m]))
```

所有 Dashboard 和告警规则都引用 `job:http_error_ratio:rate5m`，核心计算逻辑只存在一处：

```yaml
# 告警规则中直接引用
- alert: HighErrorRate
  expr: job:http_error_ratio:rate5m > 0.05

# Dashboard 中也直接引用
# Query: job:http_error_ratio:rate5m{job="payment-service"}
```

这样，Dashboard 看到告警触发时，去查对应的 Panel 数据，两者看到的是**同一条时序的同一份数据**，消除了逻辑不一致的隐患。

### 场景三：为 Federation 提供轻量数据

Prometheus Federation 要求预先声明要从子集群拉取哪些指标，并且通常只拉取汇总数据而非原始高基数数据（原因见上述性能分析）。

实践中，通常只联邦 Recording Rule 生成的聚合指标：

```yaml
# 各集群的 prometheus.yml 中定义 Recording Rules
# 全局 Prometheus 只通过 /federate 拉取这些预聚合结果

# 全局 prometheus.yml
scrape_configs:
  - job_name: 'federate'
    metrics_path: '/federate'
    params:
      match[]:
        - '{__name__=~"job:.*"}'      # 只拉取以 "job:" 开头的指标（Recording Rule 命名规范）
        - '{__name__=~"cluster:.*"}'   # 只拉取集群级别汇总
```

通过命名规范（`job:`、`cluster:` 前缀），可以用一个正则表达式精准匹配所有 Recording Rule 生成的指标，同时排除所有原始采集指标。

### 场景四：长时间窗口查询的性能提升

**问题**：查询过去 7 天的平均请求速率：

```promql
rate(http_requests_total[7d])
```

Prometheus 需要加载过去 7 天内所有 `http_requests_total` 的样本点到内存中进行计算，代价极高，往往超时。

**解决**：利用 Recording Rule 的链式特性，先预计算短窗口速率，再在长窗口内聚合：

```yaml
# 每分钟预计算 5 分钟速率
- record: job:http_requests_total:rate5m
  expr: sum by (job) (rate(http_requests_total[5m]))
```

查询过去 7 天的趋势时，Grafana 使用 `/api/v1/query_range` 查询 `job:http_requests_total:rate5m` 在过去 7 天的历史值，读取的是已经写入 TSDB 的预计算结果，而不是重新对原始数据计算，性能大幅提升。

---

## 什么时候不需要 Recording Rule

Recording Rule 并非万能，以下场景**不应该**使用：

**临时排查查询**：排查问题时的一次性复杂查询，不值得为其创建 Recording Rule，直接在 Prometheus 的 Expression Browser 或 Grafana 的 Explore 界面执行即可。

**低频访问的简单查询**：如果某个查询每天只用到一两次，且原始数据量不大，没有性能问题，创建 Recording Rule 只会增加存储压力（额外的时序 + 样本点）而没有收益。

**需要实时精确值的场景**：Recording Rule 的数据有最多 `evaluation_interval` 的延迟。如果业务场景要求秒级实时准确（如金融交易监控），需要考虑这个延迟是否可接受。

**标签基数会爆炸的聚合**：如果 `record` 的结果仍然有极高的基数（如 `by (pod, container, namespace, cluster, method, path)`），Recording Rule 不但没有降低查询代价，还会额外占用 TSDB 存储。Recording Rule 的核心价值在于**降低基数**，如果聚合后基数没有明显下降，需要重新审视设计。

---

## 评估与监控 Recording Rule 本身

Recording Rule 本身的运行状态也需要监控。Prometheus 暴露了以下内置指标：

```promql
# 规则评估耗时（按 group 和 rule 区分）
prometheus_rule_evaluation_duration_seconds

# 规则评估失败次数
prometheus_rule_evaluation_failures_total

# 规则组最近一次评估的时间戳
prometheus_rule_group_last_evaluation_timestamp_seconds
```

如果发现 `prometheus_rule_evaluation_duration_seconds` 持续超过 `evaluation_interval`，说明规则评估跟不上节奏，Prometheus 会跳过部分评估轮次，导致数据缺口。此时需要：

1. 检查是否有过于复杂的 `expr`，考虑拆分或优化
2. 增大 Prometheus 的 CPU 资源
3. 将规则分散到多个 group（不同 group 并行评估）

---

## 完整使用示例

以下是一套针对 HTTP 服务的典型 Recording Rule 集合，覆盖请求量、错误率、延迟三个核心指标：

```yaml
groups:
  - name: http_service_aggregations
    interval: 1m
    rules:
      # 请求速率：按 job 聚合
      - record: job:http_requests_total:rate5m
        expr: sum by (job) (rate(http_requests_total[5m]))

      # 错误请求速率：按 job 聚合
      - record: job:http_requests_errors:rate5m
        expr: sum by (job) (rate(http_requests_total{status=~"5.."}[5m]))

      # 错误率（比值）：引用上面两条（同 group 内顺序保证）
      - record: job:http_error_ratio:rate5m
        expr: |
          job:http_requests_errors:rate5m
          /
          job:http_requests_total:rate5m

      # P99 延迟：按 job 聚合
      - record: job:http_request_duration_seconds:p99_rate5m
        expr: |
          histogram_quantile(0.99,
            sum by (job, le) (
              rate(http_request_duration_seconds_bucket[5m])
            )
          )
```

对应的 Alert Rule 和 Dashboard 均直接引用这些 Recording Rule，保持逻辑一致性：

```yaml
groups:
  - name: http_service_alerts
    rules:
      - alert: HighErrorRate
        expr: job:http_error_ratio:rate5m > 0.01
        for: 5m
        labels:
          severity: critical

      - alert: HighP99Latency
        expr: job:http_request_duration_seconds:p99_rate5m > 1.0
        for: 5m
        labels:
          severity: warning
```

---

## 总结

Recording Rule 的本质是**以存储换计算**：用 TSDB 存储预计算结果，换取查询时的高性能响应。它的核心价值体现在三个方面：

| 价值 | 说明 |
|------|------|
| 性能 | 将高代价查询从读时移到写时，Dashboard 加载速度从秒级降到毫秒级 |
| 一致性 | 核心指标逻辑定义一处，Dashboard 和告警共用，消除不一致 |
| 可组合性 | 同 group 内顺序执行，支持将复杂计算分解为可读的链式步骤 |

使用决策很简单：**当一个 PromQL 表达式需要被重复查询、或查询性能不可接受时，就是引入 Recording Rule 的时机**。
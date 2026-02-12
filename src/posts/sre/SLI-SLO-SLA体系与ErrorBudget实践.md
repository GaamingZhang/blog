---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: true
article: true
category:
  - SRE
tag:
  - SRE
  - SLO
  - Prometheus
  - 可观测性
---

# SLI/SLO/SLA 体系建设与 Error Budget 实践

## 从告警阈值的困境说起

在[上一篇关于告警阈值设计](./告警阈值设计-静态与动态阈值的选型与实践.md)中，我们讨论了为什么 `CPU > 80%` 这样的静态告警容易产生噪音。但这个问题其实指向一个更深层的困境：我们用什么来衡量一个服务是否"健康"？

设想这样一个场景：你的团队在季度末想加快发布节奏，连续部署了三个版本。QA 说质量没问题，但稳定性团队说现在不能动，因为"最近线上不稳定"。争论的核心是：**到底多稳定才算稳定？** 如果没有量化标准，这个问题永远没有答案，只能变成意见之争。

这就是 SLI/SLO/SLA 体系存在的意义。它提供了一套共同语言，让研发、运维、产品、业务方都能基于同一份数据讨论"可靠性到底够不够"，并将这个讨论转化为工程决策依据。

---

## 核心概念的精确定义

### SLI：你在测量什么

**SLI（Service Level Indicator，服务级别指标）** 是对服务某一个行为维度的量化测量。Google SRE 手册将常用 SLI 归纳为四类：

- **可用性（Availability）**：成功请求占总请求的比例
- **延迟（Latency）**：请求的响应时间分布（通常关注 P99）
- **吞吐量（Throughput）**：单位时间处理的请求数量
- **错误率（Error Rate）**：失败请求的比例

SLI 的关键在于它**必须反映用户的真实体验**，而不是系统内部状态。`CPU 使用率` 不是 SLI，因为 CPU 高不一定影响用户；`请求成功率` 是 SLI，因为请求失败用户会直接感知到。

以下是几类典型 SLI 的 PromQL 实现：

**可用性 SLI（5 分钟窗口）**

```promql
# 请求成功率：HTTP 2xx/3xx 占总请求的比例
sum(rate(http_requests_total{status=~"2..|3.."}[5m]))
/
sum(rate(http_requests_total[5m]))
```

**延迟 SLI（P99）**

```promql
# P99 延迟：99% 的请求在多少毫秒内完成
histogram_quantile(
  0.99,
  sum(rate(http_request_duration_seconds_bucket[5m])) by (le)
)
```

**错误率 SLI**

```promql
# HTTP 5xx 错误率
sum(rate(http_requests_total{status=~"5.."}[5m]))
/
sum(rate(http_requests_total[5m]))
```

### SLO：你的承诺目标

**SLO（Service Level Objective，服务级别目标）** 是对 SLI 设定的目标值。例如："过去 30 天内，99.9% 的请求成功率"。

SLO 背后最重要的含义是它对应了多少允许失败的时间。下面这张对照表是工程实践中必须反复查阅的：

| SLO | 年允许停机 | 季度允许停机 | 月允许停机 | 周允许停机 |
|-----|-----------|-------------|-----------|-----------|
| 99% | 87.6 小时 | 21.9 小时 | 7.3 小时 | 1.68 小时 |
| 99.5% | 43.8 小时 | 10.9 小时 | 3.65 小时 | 50.4 分钟 |
| 99.9% | 8.76 小时 | 2.19 小时 | 43.8 分钟 | 10.1 分钟 |
| 99.95% | 4.38 小时 | 65.7 分钟 | 21.9 分钟 | 5.04 分钟 |
| 99.99% | 52.6 分钟 | 13.1 分钟 | 4.38 分钟 | 60.5 秒 |
| 99.999% | 5.26 分钟 | 78.8 秒 | 26.3 秒 | 6.05 秒 |

:::tip 99.9% 和 99.99% 的区别
从 99.9% 到 99.99%，听起来只差 0.09%，但代价是：月允许停机从 43 分钟缩短到 4 分钟。这意味着一次正常的发布重启都可能耗尽预算，你需要蓝绿部署、零停机发布等工程投入。SLO 不是越高越好，而是需要和工程成本匹配。
:::

### SLA：合同层面的承诺

**SLA（Service Level Agreement，服务级别协议）** 是与外部客户签订的合同，规定了服务未达到 SLO 时的赔偿条款。

SLA 和 SLO 的关系是：**SLO 是内部目标，SLA 是外部承诺，SLA 通常比 SLO 低一个档次。**

```
内部 SLO：99.95%（留有工程缓冲）
   ↓
对外 SLA：99.9%（承诺给客户的底线）
```

这个差值就是"工程缓冲"——当系统出现问题时，有一段时间窗口供团队处理，而不会立即触发违约赔偿。

### Error Budget：可靠性的"花销预算"

**Error Budget（错误预算）** = 1 - SLO

如果 SLO 是 99.9%，那么 Error Budget 是 0.1%，对应每月 43.8 分钟。这 43.8 分钟是团队在这段时间内被"允许"用来出故障的时间，可以花在：

- 计划内发布的短暂中断
- 意外故障的恢复时间
- 基础设施变更的影响窗口

:::warning 为什么 100% 可用性是错的
追求 100% 可用性在工程上无意义——你无法区分 100% 和 99.9999% 的差距，用户感知不到，仪器测量也有误差。100% 的目标会导致团队极度保守，不敢做任何变更，最终损害的反而是系统的长期可靠性（因为技术债无法清理、新功能无法发布）。Error Budget 的存在是为了让团队有"可控地犯错"的空间。
:::

Error Budget 的计算公式：

```
剩余 Error Budget = (当前 SLI - SLO) / (1 - SLO) × 100%

示例：
SLO = 99.9%，当前 30 天可用性 = 99.95%
剩余预算 = (99.95% - 99.9%) / (1 - 99.9%) × 100% = 50%
即已消耗 50% 的月度 Error Budget
```

---

## Error Budget 如何驱动工程决策

Error Budget 的真正价值不在于计算，而在于它如何改变团队的行为模式。

### Error Budget Policy

一套典型的 Error Budget Policy 将预算状态分为三档：

```
预算状态          剩余比例    工程决策
──────────────────────────────────────────────────────
健康（Green）     > 50%      正常发布节奏，可以推进新特性
告警（Yellow）    10%~50%    减慢发布节奏，优先修复可靠性问题
耗尽（Red）       < 10%      冻结所有非紧急变更，全力恢复可靠性
```

这个政策将"能不能发布"从主观判断变成了客观数据驱动。产品经理不能再说"这个功能很重要，必须今天上线"，工程师也不能说"我觉得应该等一等"——两者都需要遵守 Policy，而 Policy 由数据决定。

### Burn Rate：预算燃烧速率

仅仅知道剩余 Error Budget 是不够的，还需要知道它**消耗的速度**。这就是 Burn Rate（燃烧速率）。

Burn Rate = 1 表示恰好按时耗尽预算（例如，月度窗口内刚好消耗完 0.1%）。

```
Burn Rate = 实际错误率 / (1 - SLO)

示例：
SLO = 99.9%，当前错误率 = 1%
Burn Rate = 1% / 0.1% = 10x

这意味着以当前速度，会在 30天 / 10 = 3天 内耗尽月度预算
```

**快燃（Fast Burn）vs 慢燃（Slow Burn）** 是两种截然不同的风险模式：

```
快燃：Burn Rate 很高（如 50x）
→ 出现严重故障，短时间内大量请求失败
→ 用户立刻感知，需要分钟级响应
→ 短时间窗口（5min）足以检测

慢燃：Burn Rate 较低（如 2x）
→ 小范围的性能退化，悄悄消耗预算
→ 用户感知不明显，但若不处理会在几天内耗尽预算
→ 需要长时间窗口（6h/1h）才能检测
```

### 多窗口多燃烧速率告警

单窗口告警存在一个根本缺陷：短窗口会误报（5分钟的抖动触发告警），长窗口有延迟（故障发生 1 小时后才被发现）。Google SRE 提出的解决方案是**多窗口多燃烧速率（Multi-Window Multi-Burn-Rate）告警**。

核心思路：**同一条告警，要求短窗口和长窗口的 Burn Rate 同时超过阈值才触发。** 短窗口确保快速响应，长窗口过滤抖动。

```
告警级别    短窗口    长窗口    Burn Rate 阈值
──────────────────────────────────────────
Page        5min      1h        14x
Ticket      30min     6h        3x
```

这套参数的设计逻辑：
- **Page（14x burn rate）**：以 14 倍速消耗，1 小时内会耗尽月度预算的 14/720 ≈ 2%，约 5 分钟内耗尽每小时的预算，需要立即响应
- **Ticket（3x burn rate）**：以 3 倍速消耗，预计 10 天耗尽月度预算，影响下一个发布窗口，需要排期处理

---

## SLI 选取原则与 PromQL 实现

### 用户旅程 SLI

好的 SLI 设计从**用户旅程**出发，而不是从系统指标出发。以电商结账流程为例：

```
用户旅程：加购 → 结算 → 支付 → 完成
对应 SLI：
  - 加购接口成功率（可用性 SLI）
  - 结算页加载时间（延迟 SLI）
  - 支付接口成功率（可用性 SLI，权重最高）
  - 订单创建成功率（可用性 SLI）
```

**可用性 SLI（滑动窗口）的 Recording Rule**

```promql
# Recording Rule：预先计算各服务的成功率（减少查询开销）
- record: job:request_success_rate:ratio_rate5m
  expr: |
    sum(rate(http_requests_total{status=~"2..|3.."}[5m])) by (job)
    /
    sum(rate(http_requests_total[5m])) by (job)
```

**Burn Rate 计算**

```promql
# 1 小时窗口的 Burn Rate（相对于 99.9% SLO）
(
  1 - sum(rate(http_requests_total{status=~"2..|3.."}[1h])) by (job)
      / sum(rate(http_requests_total[1h])) by (job)
)
/ 0.001
```

### 依赖型 SLI

数据库和缓存是服务可靠性的关键依赖，需要独立跟踪：

```promql
# 数据库查询延迟 P99
histogram_quantile(
  0.99,
  sum(rate(db_query_duration_seconds_bucket[5m])) by (le, db_name)
)

# Redis 命中率
sum(rate(redis_hits_total[5m])) by (instance)
/
(
  sum(rate(redis_hits_total[5m])) by (instance)
  + sum(rate(redis_misses_total[5m])) by (instance)
)
```

### 多窗口告警规则完整配置

```yaml
# prometheus/rules/slo_alerts.yaml
groups:
  - name: slo_burn_rate
    rules:
      # 快燃告警：5min + 1h 双窗口，Burn Rate > 14x
      - alert: SLOFastBurn
        expr: |
          (
            (
              1 - sum(rate(http_requests_total{status=~"2..|3.."}[5m])) by (job)
                  / sum(rate(http_requests_total[5m])) by (job)
            ) / 0.001 > 14
          )
          and
          (
            (
              1 - sum(rate(http_requests_total{status=~"2..|3.."}[1h])) by (job)
                  / sum(rate(http_requests_total[1h])) by (job)
            ) / 0.001 > 14
          )
        for: 2m
        labels:
          severity: critical
          slo: availability
        annotations:
          summary: "{{ $labels.job }} 快速消耗 Error Budget（>14x burn rate）"
          description: "当前燃烧速率 {{ $value | humanize }}x，预计 {{ printf \"%.1f\" (div 720.0 $value) }} 小时内耗尽月度预算"

      # 慢燃告警：30min + 6h 双窗口，Burn Rate > 3x
      - alert: SLOSlowBurn
        expr: |
          (
            (
              1 - sum(rate(http_requests_total{status=~"2..|3.."}[30m])) by (job)
                  / sum(rate(http_requests_total[30m])) by (job)
            ) / 0.001 > 3
          )
          and
          (
            (
              1 - sum(rate(http_requests_total{status=~"2..|3.."}[6h])) by (job)
                  / sum(rate(http_requests_total[6h])) by (job)
            ) / 0.001 > 3
          )
        for: 15m
        labels:
          severity: warning
          slo: availability
        annotations:
          summary: "{{ $labels.job }} 持续消耗 Error Budget（>3x burn rate）"
          description: "当前燃烧速率 {{ $value | humanize }}x，若持续则 {{ printf \"%.1f\" (div 240.0 $value) }} 天内耗尽月度预算"
```

AlertManager 路由配置，将 SLO 告警独立路由：

```yaml
# alertmanager/config.yaml
route:
  receiver: default
  routes:
    - match:
        slo: availability
        severity: critical
      receiver: pagerduty-oncall
      group_wait: 30s
      group_interval: 5m
      repeat_interval: 1h

    - match:
        slo: availability
        severity: warning
      receiver: slack-engineering
      group_wait: 5m
      group_interval: 30m
      repeat_interval: 8h

receivers:
  - name: pagerduty-oncall
    pagerduty_configs:
      - routing_key: "<your-routing-key>"
        description: "{{ .CommonAnnotations.summary }}"

  - name: slack-engineering
    slack_configs:
      - channel: "#sre-alerts"
        title: "SLO 慢速消耗告警"
        text: "{{ .CommonAnnotations.description }}"
```

---

## SLO 制定实战

### 从历史数据推算合理 SLO

SLO 不应该凭直觉设定，应该从历史数据出发：

```promql
# 查询过去 90 天的实际可用性
avg_over_time(
  (
    sum(rate(http_requests_total{status=~"2..|3.."}[5m]))
    /
    sum(rate(http_requests_total[5m]))
  )[90d:5m]
)
```

一个合理的起点：**以历史实际水平作为初始 SLO，减去一个档次作为第一版目标。** 如果过去 90 天实际可用性是 99.95%，那么第一版 SLO 可以设为 99.9%——既反映了真实能力，又留了工程改进空间。

不要在没有历史数据的情况下直接设 99.99%，这会让 Error Budget 极小，任何小问题都会立即耗尽预算，导致团队陷入永久"预算耗尽"状态，Policy 形同虚设。

### 分层 SLO：核心链路 vs 非核心

不同业务路径对可靠性的要求不同，应该区别对待：

```
服务层级    示例               SLO 参考    告警响应
──────────────────────────────────────────────────
核心链路    支付、登录、下单    99.95%      Page（立即）
主要功能    搜索、商品详情      99.9%       Ticket（4h 内）
辅助功能    评论、收藏          99.5%       Ticket（工作日）
内部服务    数据分析、报表      99%         工单（24h 内）
```

用标签区分链路层级，让告警路由自动处理优先级：

```promql
# 在指标采集时打入 tier 标签
http_requests_total{job="payment-service", tier="critical"}
http_requests_total{job="search-service", tier="major"}
```

### SLO Dashboard 设计

Grafana 面板需要同时展示三个维度：当前 SLI 值、Error Budget 消耗量、Burn Rate 趋势。

```json
// Grafana Panel：Error Budget 消耗进度（本月）
{
  "type": "gauge",
  "title": "Error Budget 本月剩余",
  "targets": [
    {
      "expr": "1 - (sum(increase(http_requests_total{status!~\"2..|3..\"}[30d])) / sum(increase(http_requests_total[30d]))) / 0.001",
      "legendFormat": "剩余预算百分比"
    }
  ],
  "fieldConfig": {
    "defaults": {
      "thresholds": {
        "steps": [
          {"value": 0, "color": "red"},
          {"value": 0.1, "color": "yellow"},
          {"value": 0.5, "color": "green"}
        ]
      },
      "unit": "percentunit",
      "min": 0,
      "max": 1
    }
  }
}
```

一张实用的 SLO Dashboard 至少包含：

```
┌─────────────────────────────────────────────────┐
│  当前 SLI（实时）     Error Budget 剩余（本月）   │
│  99.97%               ██████████░░  67%          │
├─────────────────────────────────────────────────┤
│  Burn Rate 趋势（过去 24h）                       │
│  ▁▁▁▁▂▂▁▁▁▁▅▅▅▃▂▁▁▁▁▁  当前 0.8x               │
├─────────────────────────────────────────────────┤
│  历史 SLO 达标情况（过去 12 个月）                │
│  ✓ ✓ ✓ ✗ ✓ ✓ ✓ ✓ ✓ ✓ ✓ ✓                     │
└─────────────────────────────────────────────────┘
```

---

## Error Budget Policy 落地

### 三阶段联动机制

Error Budget Policy 只有与具体工程流程挂钩才有价值：

**阶段一：预算健康（剩余 > 50%）**
- 发布流水线正常运行，无额外审批
- 可以推进架构改造、技术债清理
- 鼓励在非核心功能上做更多实验

**阶段二：预算告警（剩余 10%~50%）**
- 新功能发布需要 TechLead 审批
- 每周 SLO Review 会议，跟踪预算消耗原因
- 暂停非紧急的基础设施变更

**阶段三：预算耗尽（剩余 < 10%）**
- 自动触发变更冻结，除紧急修复外所有发布暂停
- On-Call 升级为 P0，每天两次状态同步
- 产品路线图推迟，工程资源全部转向可靠性

### 与 On-Call 的结合

Error Budget 给 On-Call 提供了一个决策框架：

```
收到告警后的决策树：

  快燃告警触发？
  ├── YES → 立即介入，这是 P0 事故
  │         预算消耗速度 > 14x，可能数小时内影响 SLA
  └── NO  → 是慢燃告警？
            ├── YES → 预约处理，创建工单
            │         在下个工作日前解决即可
            └── NO  → 记录，不需要立即响应
```

### 与 Release 流程的结合

在 CI/CD 流水线中加入 Error Budget 检查节点：

```
代码提交
  ↓
自动化测试
  ↓
Error Budget 检查 ←── 查询当前剩余预算
  ├── 预算 > 50%：自动放行
  ├── 预算 10%~50%：需要审批人确认
  └── 预算 < 10%：自动阻断，仅允许 hotfix 标签的提交通过
  ↓
部署到生产
```

---

## 常见陷阱与反模式

### 反模式一：SLO 设置过高

许多团队在初期会把 SLO 设为 99.999%，原因是"我们是关键业务，不能出问题"。这是一个典型误区。

99.999% 意味着每月只有 26 秒的 Error Budget，一次普通的 Kubernetes Pod 滚动重启就能消耗完。结果是：告警永远处于"预算耗尽"状态，Policy 无法执行，团队把 SLO 当摆设。

**正确做法**：第一版 SLO 宁可设低，基于数据逐步提高。从 99.9% 开始，稳定运行两个季度后再考虑提升到 99.95%。

### 反模式二：用内部指标代替用户体验指标

```
错误做法：
  SLI = "Kubernetes Pod 可用率 > 95%"

问题：
  Pod 全部健康，但数据库连接池满了，用户请求全部失败
  ↓ 这个 SLI 无法反映用户体验

正确做法：
  SLI = "HTTP 请求成功率（从用户侧测量）"
  SLI = "P99 端到端响应时间（从负载均衡层测量）"
```

最佳测量点是尽可能靠近用户侧：优先用负载均衡日志，其次用 API Gateway 指标，最后才是服务内部指标。

### 反模式三：Error Budget Policy 不落地

这是最常见的失败模式。很多团队定义了 SLO 和 Policy，但在实际执行时，业务压力一来，Policy 就被绕过了——"这个版本很重要，错误预算的事下次再说"。

Error Budget Policy 必须有**强制执行机制**，而不仅仅是一个建议文档：

- 在发布系统中硬编码检查逻辑
- 变更冻结需要 CTO/VP 级别审批才能豁免
- 每季度 SLO Review 成为工程团队的强制仪式
- 违反 Policy 的案例纳入事后复盘，而不是被默默忽略

---

## 小结

- **SLI** 是用户体验的量化指标，必须从用户视角出发，而非系统内部视角
- **SLO** 是对 SLI 的目标承诺，其核心价值在于对应了量化的 Error Budget
- **SLA** 是对外合同，通常比内部 SLO 低一个档次，留有工程缓冲
- **Error Budget** 将可靠性从定性讨论变为定量决策，驱动发布节奏、工程优先级与 On-Call 响应
- **Burn Rate** 区分快燃和慢燃两种风险，多窗口双阈值告警兼顾响应速度与误报率
- SLO 第一版宜低不宜高，基于历史数据设定，随工程能力提升逐步收紧

---

## 常见问题

### Q1：SLO 应该由谁来制定，工程团队还是产品团队？

SLO 的制定需要工程和产品共同参与，但两方的角色不同。工程团队负责提供数据——历史可用性水平、达到更高 SLO 所需的工程投入（例如从 99.9% 到 99.99% 可能需要引入蓝绿部署、消除所有单点故障，需要两个季度的工程工作量）。产品团队负责提供业务判断——用户对可靠性的敏感程度、竞品的 SLA 水平、客户合同的约束。最终 SLO 是两方根据成本收益达成的共识，而不是工程单方面的技术决定。实践中建议每半年做一次 SLO Review，根据业务发展和工程能力变化重新校准。

### Q2：如何处理依赖外部服务导致的 Error Budget 消耗？

当你的服务依赖第三方 API 或云服务，外部故障会消耗你的 Error Budget，但这不在团队控制范围内。有几种处理方式：第一，在 SLI 计算时排除已知的外部故障时段（需要人工标注）；第二，为外部依赖设置独立的 SLO，当外部服务不可用时，相关的 Error Budget 消耗记录在"外部故障"账户下；第三，在架构层面增加降级和熔断机制，使外部故障对用户的影响最小化，这样外部故障就不会大量消耗你的 Error Budget。推荐的做法是先做第三条（降低外部依赖的影响），再做第二条（独立跟踪外部故障）。

### Q3：Burn Rate 告警和传统的错误率告警有什么本质区别？

传统错误率告警（如"错误率 > 1% 持续 5 分钟"）的问题在于它是绝对阈值，无法感知累积损害。如果错误率是 0.5%，低于告警阈值，但它持续了整整一个月，实际上已经消耗了全部 Error Budget 的 5 倍。Burn Rate 告警关注的是**消耗速度相对于 SLO 的比值**，它能捕捉到这种"低烈度但持续的"损伤。两者结合才完整：传统错误率告警捕捉突发的绝对故障（如服务完全宕机），Burn Rate 告警捕捉相对于 SLO 的超速消耗。实践中建议同时保留两类告警，但 Burn Rate 告警用于 Page 级别的 On-Call 唤醒，传统告警用于快速检测完全不可用的场景。

### Q4：Error Budget 在微服务架构下如何管理，每个服务单独跟踪还是统一跟踪？

两个维度都需要。每个微服务应该有独立的 SLO 和 Error Budget，这样可以精准定位是哪个服务在消耗预算，也让各服务团队对自己的可靠性负责。同时，面向用户的核心业务链路应该有一个端到端的 SLO（例如"结账全链路成功率 99.9%"），这个端到端 SLO 的 Error Budget 是各服务 SLO 的汇总视角，反映的是用户实际体验。在实践中，通常先从端到端 SLO 出发定义用户体验目标，再分解到各服务，要求每个服务的 SLO 足够高，使得整体链路的 SLO 能够达成。例如，4 个服务串联，整体需要 99.9%，那么每个服务至少需要 99.975%。

### Q5：如何向非技术的业务方解释 Error Budget 的概念？

可以用"运维预算"来类比。就像财务预算规定了一个季度能花多少钱一样，Error Budget 规定了一个月内系统"能出多少故障"。每当出现一次故障，就是在花费这个预算；每当做一次发布变更，也会消耗一小部分预算（因为发布可能引入问题）。当预算花光了，就像钱花完了一样，需要暂停"消费"（变更），让余额恢复（新的月份重置）。对业务方来说，最重要的信息是：Error Budget 不是工程团队阻碍业务发展的借口，而是一个透明的、对双方都公平的决策框架。如果业务方觉得发布节奏太慢，他们也可以选择降低 SLO，换取更多的 Error Budget，前提是接受相应的用户体验风险。这个 tradeoff 变得可量化、可讨论。

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

# PromQL 中 rate() 和 irate() 的区别

## 为什么需要 rate()

在回答两者区别之前，先理解它们共同解决的问题。

Prometheus 中的 **Counter** 类型指标是单调递增的累计值。以 HTTP 请求总数为例，`http_requests_total` 在服务运行期间只增不减。直接看这个值意义不大 —— 我们真正关心的是**每秒处理了多少请求**。

`rate()` 和 `irate()` 都是将 Counter 的累计值转换为**速率**（每秒的变化量）。它们的差异在于：**用哪些数据点来计算这个速率，以及如何处理时间窗口**。

## Counter Reset 处理

在深入两个函数之前，需要理解 Counter 重置（Reset）的概念。

当服务重启时，Counter 会从 0 重新计数。如果不处理这种情况，计算速率时就会出现负值（新值 < 旧值），结果完全错误。

Prometheus 的处理方式：当检测到当前采样点的值小于前一个采样点时，认定发生了 Reset，此时将重置前的最后一个值加到重置后的计数中，保证速率计算的正确性。`rate()` 和 `irate()` 都会自动处理 Counter Reset。

```
时间:    t1    t2    t3(重启)  t4    t5
值:       50   100     5       30    60

实际增量: +50   +50  +5(+100)  +25   +30
         ↑重置时加上重置前的最后值 100
```

## rate() 原理：平均变化率

`rate(v[d])` 计算的是**指定时间窗口内的平均每秒变化率**。

### 计算公式

```
rate = (end_value - start_value + counter_resets_correction) / duration
```

其中：
- `end_value`：窗口内最后一个采样点的值
- `start_value`：窗口内第一个采样点的值
- `duration`：窗口覆盖的实际时间跨度（秒）

举例说明：采集间隔为 15s，查询 `rate(http_requests_total[1m])`。

```
t=0s:   requests = 1000
t=15s:  requests = 1050
t=30s:  requests = 1120
t=45s:  requests = 1180
t=60s:  requests = 1240

rate = (1240 - 1000) / 60 = 4 req/s
```

这 4 req/s 是这一分钟内的**平均值**，抹平了中间的波动。

### 外推（Extrapolation）

Prometheus 对 `rate()` 做了一个细节优化：当窗口边界与实际采样点之间存在间隙时，会按照当前速率将结果**外推**到窗口边界，避免因采样点不完全落在窗口边界导致的系统性低估。这也是为什么 `rate()` 的结果并不总是整数。

## irate() 原理：瞬时变化率

`irate(v[d])` 只取窗口内**最后两个**采样点，计算这两个点之间的瞬时速率。

### 计算公式

```
irate = (last_value - second_last_value) / (last_timestamp - second_last_timestamp)
```

同样的数据，查询 `irate(http_requests_total[1m])`：

```
t=0s:   requests = 1000
t=15s:  requests = 1050
t=30s:  requests = 1120
t=45s:  requests = 1180
t=60s:  requests = 1240   ← last
t=45s:  requests = 1180   ← second_last（仅用这两个点）

irate = (1240 - 1180) / (60 - 45) = 60 / 15 = 4 req/s
```

在这个平稳的例子中结果相同，但一旦出现突刺，两者就会显著不同：

```
t=0s:   requests = 1000
t=15s:  requests = 1050
t=30s:  requests = 1120
t=45s:  requests = 1600   ← 突刺（+480）
t=60s:  requests = 1640

rate  = (1640 - 1000) / 60 ≈ 10.7 req/s
irate = (1640 - 1600) / 15 ≈ 2.7 req/s  ← 突刺已过去，只看最后15s
```

而如果查询时刻恰好在突刺之后：
```
irate = (1600 - 1120) / 15 = 32 req/s   ← 完整捕捉到突刺
```

这揭示了 `irate()` 的本质：**它的结果高度依赖查询时刻，对瞬时峰值极度敏感**。

### irate 中 range 的作用

一个常见的误解是：`irate(v[1m])` 中的 `1m` 决定了计算窗口。**实际上 `irate` 忽略了窗口内所有中间采样点，`1m` 只是用来限定"往回查找最后两个点"的搜索范围。**

如果在 `1m` 窗口内找不到两个采样点（例如采集中断），irate 会返回空值（no data）。因此建议将 range 设置为至少 **3～4 倍采集间隔**，以容忍偶发的采集失败。

## 核心差异对比

| 维度 | rate() | irate() |
|------|--------|---------|
| 计算依据 | 窗口内所有采样点（首尾两点） | 窗口内最后两个采样点 |
| 平滑程度 | 高，反映平均趋势 | 低，反映瞬时变化 |
| 对突刺的反应 | 稀释（被窗口时长平均） | 敏感（可捕捉但也可能错过） |
| 结果稳定性 | 稳定，图形平滑 | 抖动明显，图形锯齿 |
| 采集中断容忍度 | 高（只需窗口内有两个点） | 需要最后两点相对连续 |
| Counter Reset 处理 | 支持 | 支持 |

## 场景选择

### 使用 rate() 的场景

**1. 告警规则**

这是最重要的使用场景。告警需要稳定可靠，`irate()` 的抖动性会导致大量误报。

```promql
# 正确：用 rate() 做告警
alert: HighErrorRate
expr: rate(http_requests_total{status="500"}[5m]) > 10
```

`irate()` 在告警中几乎不推荐使用，因为单个瞬间的峰值就可能触发告警，而该峰值在下一个评估周期就消失了。

**2. 长时间趋势分析**

查看过去 24 小时、7 天的流量变化趋势，`rate()` 的平滑特性使图表更可读：

```promql
# 过去 5 分钟的平均 QPS
rate(http_requests_total[5m])
```

**3. 配合 histogram 计算延迟分位数**

`histogram_quantile` 通常需要先对 `_bucket` 指标用 `rate()` 计算速率：

```promql
histogram_quantile(0.99,
  sum by (le) (
    rate(http_request_duration_seconds_bucket[5m])
  )
)
```

这里用 `irate()` 会导致结果极不稳定，不适用。

**4. 计算 increase()**

`increase(v[d])` 本质上就是 `rate(v[d]) * duration`，同样用的是平均变化率思路，适合统计"最近 1 小时共发生了多少次"这类场景。

### 使用 irate() 的场景

**1. 实时监控 Dashboard**

当你需要一个"实时流量大屏"，希望看到流量的即时波动时，`irate()` 是合适的：

```promql
# 实时 QPS，每次刷新都反映最新状态
irate(http_requests_total[1m])
```

**2. 短时间突刺的发现**

如果你的 Dashboard 专门用来排查"刚刚发生的尖刺"，`irate()` 可以保留这些高频波动的细节，而 `rate()` 会将其平滑掉。

**3. 高采集频率场景**

当采集间隔非常短（如 5s）时，`irate()` 的两点计算在时间粒度上已经足够精细，同时避免了长窗口的平滑效应。

## 一个常见的选择误区

很多人认为"想看峰值就用 irate，想看平均就用 rate"，这只是表象。更准确的区分是：

- **rate()：时间窗口越长越稳定**。`rate(v[1m])` 比 `rate(v[5m])` 波动更大，因为稀释效果更弱。调整窗口长度是控制平滑程度的主要手段。

- **irate()：时间窗口长度几乎不影响结果**（只影响能否找到最后两点）。你无法通过调长窗口来"稳定" irate 的结果，它始终只看最后两个采样点。

这意味着，如果你想要一个"不那么激进但又比 rate 灵敏"的中间态，正确做法是**缩短 rate() 的窗口**（如从 5m 改为 1m），而不是切换到 irate()。

## 与 increase() 的关系

`increase(v[d])` 计算的是窗口内的总增量，内部等价于：

```
increase(v[d]) = rate(v[d]) * d（秒数）
```

同样使用首尾两点计算，同样做外推，适合回答"最近 N 分钟发生了多少次 xxx"。

```promql
# 最近 1 小时的 5xx 错误总数
increase(http_requests_total{status=~"5.."}[1h])
```

没有对应 `irate` 的 `iincrease`，如果需要"最近两点之间的增量"，直接用 `irate * 采集间隔` 来近似即可。

## 总结

`rate()` 和 `irate()` 并非优劣之分，而是适用场景不同：

- **rate()** 是默认选择，平滑、稳定，适合告警、趋势分析和大多数 Dashboard 场景
- **irate()** 是专用工具，灵敏、有噪，适合实时大屏展示和排查短时突刺

一个实用的决策规则：**如果不确定用哪个，用 rate()**。仅当明确需要观察实时瞬时变化，且能接受图形抖动时，再考虑 irate()。告警规则中永远优先 rate()。

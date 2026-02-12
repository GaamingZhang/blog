---
date: 2026-02-11
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - grafana
tag:
  - grafana
  - ClaudeCode
---

# Grafana 核心机制

## 概述

Grafana 是一个开源的数据可视化与监控分析平台，广泛应用于可观测性体系建设中。它本身并不存储数据，而是通过插件化的数据源机制连接各类时序数据库、日志系统和指标系统，将数据以直观的图表形式呈现。理解 Grafana 的核心机制，有助于我们更好地设计监控体系、排查展示问题，并充分利用其扩展能力。

本文从架构视角深入分析 Grafana 的数据源插件系统、查询处理流水线、面板渲染机制、告警引擎以及变量插值系统等核心模块的工作原理。

## 整体架构

Grafana 采用前后端分离架构：

- **后端**：Go 语言实现，提供 HTTP API、数据代理、告警评估、插件管理等核心能力
- **前端**：React + TypeScript 实现，负责 Dashboard 渲染、用户交互和面板展示
- **数据库**：SQLite/MySQL/PostgreSQL 存储配置信息（Dashboard 定义、用户、告警规则等），注意这里存的是**元数据**，不是监控数据本身

```
┌─────────────────────────────────────────────┐
│                  浏览器 (React)               │
│  Dashboard → Panel → Query → Visualization  │
└───────────────────┬─────────────────────────┘
                    │ HTTP/WebSocket
┌───────────────────▼─────────────────────────┐
│               Grafana 后端 (Go)              │
│  ┌──────────┐ ┌──────────┐ ┌─────────────┐  │
│  │  HTTP API│ │数据代理层 │ │  告警引擎   │  │
│  └──────────┘ └──────────┘ └─────────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌─────────────┐  │
│  │ 插件管理器│ │ 渲染服务 │ │  RBAC 系统  │  │
│  └──────────┘ └──────────┘ └─────────────┘  │
└──────┬────────────┬────────────┬─────────────┘
       │            │            │
   Prometheus    Loki/ES      ClickHouse
   (数据源)      (数据源)      (数据源)
```

## 数据源插件系统

### 插件架构设计

数据源是 Grafana 最核心的扩展点。每个数据源以**插件**的形式存在，分为两类：

- **前端插件（TypeScript）**：负责查询编辑器 UI、数据格式化、可视化配置
- **后端插件（Go）**：负责实际的数据查询执行、健康检查（可选，部分插件仅有前端）

内置数据源（如 Prometheus、Loki、InfluxDB）编译进 Grafana 二进制文件，第三方插件通过插件目录加载。

### 数据代理机制

Grafana 后端充当一个**安全代理层**，其设计目标是避免将数据源凭据暴露给浏览器：

```
浏览器请求 → Grafana 后端 (/api/datasources/proxy/{id}/...) → 实际数据源
```

代理层的工作流程：
1. 浏览器携带 Session Token 向 Grafana 发起 `/api/ds/query` 请求
2. Grafana 鉴权通过后，从数据库读取该数据源的连接配置（URL、认证凭据）
3. 后端以数据源的身份向实际数据源发起查询
4. 将响应结果转换为统一的 DataFrame 格式返回给前端

这种设计使得数据源的账号密码始终保留在服务端，浏览器只需持有 Grafana Session 即可。

### 后端插件的 gRPC 通信

后端插件（Go Plugin）通过 **go-plugin**（HashiCorp 开源的进程间通信框架）与 Grafana 主进程通信。每个后端插件作为独立的子进程运行，通过 Unix Socket 的 gRPC 连接与主进程交互：

```
Grafana 主进程
    │
    ├── gRPC (Unix Socket)
    │
后端插件子进程
(实现 QueryData、CheckHealth、CallResource 等接口)
```

这种进程隔离的设计保证了插件崩溃不会影响 Grafana 主进程，同时通过 gRPC 接口标准化了插件的调用契约。

## 查询处理流水线

理解 Grafana 的查询流程，是排查数据展示问题的关键。

### 查询生命周期

一次 Panel 数据刷新的完整流程如下：

```
1. 触发器（时间范围变化/手动刷新/自动刷新）
        ↓
2. 前端收集 Panel 的查询定义（targets）
        ↓
3. 变量插值：将 $variable 替换为实际值
        ↓
4. 发送 POST /api/ds/query 请求（携带 datasource UID、查询体）
        ↓
5. 后端数据源插件执行查询（可能并发执行多个 target）
        ↓
6. 返回 DataFrame 格式的数据
        ↓
7. 前端数据转换层（Transformations）处理
        ↓
8. Visualization 渲染
```

### DataFrame：统一数据模型

Grafana 内部使用 **DataFrame** 作为统一的数据传输格式，它是一种列式存储结构：

```typescript
interface DataFrame {
  name?: string;
  fields: Field[];   // 每列是一个 Field
  length: number;    // 行数
}

interface Field {
  name: string;
  type: FieldType;   // Time / Number / String / Boolean
  values: Vector;    // 该列的所有值
  config: FieldConfig; // 单位、阈值、映射等展示配置
  labels?: Labels;   // 标签（类似 Prometheus 的 label）
}
```

不论数据来自 Prometheus（返回 JSON）、MySQL（返回 rows）还是 Elasticsearch（返回聚合结果），后端插件都需要将其转换为 DataFrame，前端面板插件基于 DataFrame 进行渲染。这种抽象使得同一个可视化面板可以接入任意数据源。

### 数据转换（Transformations）

Grafana 提供了一套在**前端**运行的数据处理流水线，核心实现在 `@grafana/data` 包中。常见的 Transformation 包括：

- **Reduce**：将时序数据按 Field 归并为单个值（用于 Stat 面板）
- **Filter by name/value**：过滤行或列
- **Join by field**：按时间戳将多个 DataFrame 合并为一张宽表
- **Calculate field**：基于已有列计算新列（支持数学表达式）

Transformation 以**链式管道**方式执行，上一个 Transformation 的输出是下一个的输入，且全部在浏览器内存中完成，不产生额外的后端请求。

## 告警引擎

Grafana 10+ 使用 **Unified Alerting**（统一告警）架构，完全重写了早期的 legacy 告警系统。

### 核心组件

**Grafana Alerting** 由以下核心部分组成：

- **Alert Rules（告警规则）**：定义查询条件和触发阈值，存储在 Grafana 数据库中
- **Scheduler（调度器）**：按规则的评估间隔定时触发评估任务
- **Evaluation Engine（评估引擎）**：执行规则查询，判断是否触发
- **State Manager（状态管理器）**：维护每条告警实例的状态机
- **Alertmanager（告警路由）**：负责告警的路由、分组、静默和通知发送

### 告警评估流程

```
Scheduler 定时触发
      ↓
Evaluation Engine 执行数据源查询（同 Panel 查询流程）
      ↓
将查询结果与告警条件比对（PromQL 表达式 or Reduce + Threshold）
      ↓
生成 Alert Instance（每个唯一标签组合对应一个实例）
      ↓
State Manager 更新状态（Normal / Pending / Firing / NoData / Error）
      ↓
Firing 状态的实例发送给 Alertmanager
      ↓
Alertmanager 根据 Routing Tree 路由到对应 Receiver（邮件/PagerDuty/Webhook 等）
```

### 状态机设计

每条告警实例维护一个状态机，关键状态转换：

```
Normal ──[条件触发]──→ Pending ──[持续超过 For 时长]──→ Firing
                          │                                  │
                    [条件恢复]                         [条件恢复]
                          ↓                                  ↓
                       Normal ←──────────────────────── Normal
```

`Pending` 状态的引入是为了避免毛刺数据造成告警抖动：只有持续触发超过 `For` 配置的时长后，才真正进入 `Firing`。

### 与 Prometheus Alertmanager 的关系

Grafana Unified Alerting 内置了一个 Alertmanager（Go 实现），其路由配置语法与 Prometheus Alertmanager 完全兼容。在大型生产环境中，Grafana 可以配置为将告警发送到**外部 Alertmanager**（如 Prometheus 自带的 Alertmanager），实现统一的告警管理。

## 变量系统

Dashboard 变量（Variables）是 Grafana 实现动态交互的核心机制，使得一个 Dashboard 可以通过下拉选择切换不同的主机、集群或时间粒度。

### 变量类型

| 类型 | 数据来源 | 典型场景 |
|------|---------|---------|
| Query Variable | 查询数据源动态获取 | 从 Prometheus 获取所有 `instance` 标签值 |
| Custom Variable | 手动指定固定选项 | prod/staging/dev 环境切换 |
| Textbox Variable | 用户自由输入 | 自定义过滤条件 |
| Constant Variable | 固定值（不在 UI 展示）| 配置公共前缀/命名空间 |
| Datasource Variable | 切换数据源 | 多集群 Prometheus 切换 |
| Interval Variable | 时间间隔 | `$__interval` 自动匹配时间范围 |

### 变量插值机制

变量值在查询发出前由前端完成替换，支持多种插值语法：

```
$variable              → 原始值，多值时逗号分隔：value1,value2
${variable:pipe}       → 管道分隔：value1|value2
${variable:regex}      → 正则 OR 语法：(value1|value2)
${variable:csv}        → CSV 格式：value1,value2
${variable:json}       → JSON 数组：["value1","value2"]
${variable:sqlstring}  → SQL 转义：'value1','value2'
```

对于 PromQL，多值变量通常使用 `=~"${host:pipe}"` 的正则匹配语法：

```promql
rate(http_requests_total{instance=~"$host"}[5m])
```

### 变量依赖与级联

变量之间可以形成**依赖关系**：一个变量的查询语句可以引用另一个变量的当前值。例如先选择 `$cluster`，再根据 `$cluster` 查询对应的 `$namespace`。

依赖变量的刷新策略（`refresh`）控制何时重新查询选项值：
- `on_dashboard_load`：Dashboard 加载时刷新
- `on_time_range_change`：时间范围变化时刷新

当依赖链中的上游变量值变化时，Grafana 会按照依赖顺序依次刷新下游变量，最终触发所有 Panel 的数据重新查询。

## 权限与多租户

### RBAC 模型

Grafana 8+ 引入了细粒度的 **RBAC（Role-Based Access Control）** 系统，权限结构为：

```
Organization（组织）
    └── Team / User
            └── Role（角色）
                    └── Permissions（权限点）
```

内置角色包括：
- **Viewer**：只读查看 Dashboard
- **Editor**：创建/修改 Dashboard 和 Alert
- **Admin**：管理数据源、用户、组织配置

细粒度权限允许在 Dashboard、文件夹、数据源级别单独授权，例如允许某用户只查看特定 Team 的 Dashboard 而无法访问其他。

### 多组织隔离

Grafana 通过 **Organization**（组织）实现多租户隔离。不同 Organization 之间：
- 数据源配置完全隔离
- Dashboard、告警规则完全隔离
- 用户可以属于多个 Organization，切换时身份和权限随之变化

这种设计适合在单个 Grafana 实例上为不同业务团队提供独立的监控空间。

## 关键设计思路总结

| 核心机制 | 设计思路 |
|---------|---------|
| 数据源插件 | 前后端分离，后端插件用 gRPC 进程隔离，统一 DataFrame 模型屏蔽数据源差异 |
| 数据代理 | 凭据保留在服务端，浏览器不直连数据源，安全边界清晰 |
| 查询流水线 | 变量插值 → 并发查询 → DataFrame 转换 → 前端渲染，职责分层 |
| 告警引擎 | Pending 状态防抖，State Manager 维护实例状态机，兼容 Alertmanager 生态 |
| 变量系统 | 多种插值格式适配不同查询语言，依赖链级联刷新 |
| 多租户 | Organization 级别硬隔离，RBAC 提供细粒度权限控制 |

## 常见问题与简答

- **问：Grafana 本身会存储监控数据吗？**
  答：不会。Grafana 只存储配置元数据（Dashboard 定义、告警规则、用户信息等）到关系型数据库中，监控数据始终存储在各数据源（Prometheus、InfluxDB 等）中，Grafana 在查询时实时从数据源拉取。

- **问：同一个 Panel 配置了多个 Query（target），它们是串行还是并行执行的？**
  答：并行执行。Grafana 后端会将同一 Panel 的多个 Query 并发发送给数据源，待所有结果返回后再统一传递给前端进行 Transformation 和渲染。

- **问：告警规则的评估和 Dashboard Panel 的查询是同一套逻辑吗？**
  答：是的。Grafana Unified Alerting 的评估引擎复用了 Panel 查询的数据源插件体系，走相同的 `/api/ds/query` 查询路径，这保证了 Dashboard 所见即告警所用，避免了两套查询逻辑不一致的问题。

- **问：为什么 Grafana 要内置 Alertmanager 而不直接使用 Prometheus Alertmanager？**
  答：Grafana 需要对非 Prometheus 数据源（如 Loki、InfluxDB）的告警统一管理，而 Prometheus Alertmanager 只接收来自 Prometheus 的告警。内置 Alertmanager 使 Grafana 成为统一的告警中心，同时保持与 Prometheus Alertmanager 路由配置语法兼容，降低迁移成本。

- **问：Dashboard 变量的值变化时，所有 Panel 都会重新查询吗？**
  答：是的，变量值变化会触发所有使用了该变量的 Panel 重新执行查询。这也是为什么要合理设计变量的依赖关系和 `refresh` 策略，避免无谓的数据源查询压力。

- **问：Grafana 的后端插件和前端插件的职责边界是什么？**
  答：后端插件（Go）负责实际的网络请求、认证、数据获取和格式转换，适用于需要访问内网数据源或处理敏感凭据的场景；前端插件（TypeScript）负责查询编辑器 UI 和可视化渲染。并非所有数据源都需要后端插件，对于支持跨域的公开数据源，前端直接查询也可以。

## 总结

Grafana 的核心价值在于其高度插件化的架构设计：通过统一的 DataFrame 数据模型和数据源插件契约，将数据获取与数据展示解耦；通过数据代理层在安全性与便利性之间取得平衡；通过 Pending 状态机制和 Alertmanager 集成构建了生产级的告警体系。理解这些机制后，无论是排查数据不符合预期的展示问题、设计高性能的 Dashboard，还是构建自定义数据源插件，都能做到有的放矢。

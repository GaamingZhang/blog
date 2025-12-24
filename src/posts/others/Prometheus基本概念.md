---
date: 2025-12-24
author: Gaaming Zhang
category:
  - 监控
tag:
  - 监控
  - Prometheus
  - 还在施工中···
---

# Prometheus基本概念

## 详细解答

### 1. Prometheus基本概念

**定义**：Prometheus是一个开源的系统监控和告警工具包，最初由SoundCloud开发，现在是Cloud Native Computing Foundation（CNCF）的毕业项目。它采用时序数据库存储监控数据，提供了强大的数据模型和查询语言，能够高效地收集、存储和查询时间序列数据。

**核心特点**：
- **多维数据模型**：使用键值对（labels）标识时间序列数据，支持灵活的查询
- **强大的查询语言**：PromQL，支持复杂的时间序列数据查询和分析
- **高效的存储**：本地时间序列数据库，支持水平扩展
- **灵活的采集方式**：支持拉取（pull）和推送（push）两种数据采集模式
- **内置告警管理**：Alertmanager组件提供告警分组、抑制和路由功能
- **可视化支持**：内置简单的可视化界面，同时支持与Grafana等工具集成
- **开放性**：支持多种客户端库和 exporters，易于扩展

### 2. Prometheus核心概念

#### 2.1 时间序列（Time Series）
- **定义**：按时间顺序记录的一系列数据点
- **组成**：由指标名称（metric name）和一组标签（labels）唯一标识
- **表示方式**：`metric_name{label_name1="label_value1", label_name2="label_value2"}`

#### 2.2 指标（Metrics）
- **定义**：用于衡量系统或服务性能的数值
- **类型**：
  - **Counter**：单调递增的计数器，如请求总数、错误数
  - **Gauge**：可增可减的仪表，如CPU使用率、内存使用率
  - **Histogram**：样本分布的直方图，如请求延迟分布
  - **Summary**：样本分布的摘要，如请求延迟的分位数

#### 2.3 标签（Labels）
- **定义**：附加在指标上的键值对，用于标识和过滤时间序列
- **作用**：
  - 提供多维度数据查询能力
  - 支持灵活的聚合和分组操作
  - 便于区分不同实例、不同环境的数据

#### 2.4 样本（Sample）
- **定义**：时间序列中的单个数据点
- **组成**：包含一个浮点数的值和一个毫秒级的时间戳

#### 2.5 目标（Targets）
- **定义**：Prometheus监控的对象，可以是服务实例、应用程序等
- **配置方式**：通过静态配置或服务发现动态获取

#### 2.6 抓取（Scraping）
- **定义**：Prometheus从目标获取监控数据的过程
- **特点**：
  - 默认使用HTTP协议
  - 支持配置抓取间隔和超时时间
  - 支持基本认证和TLS加密

### 3. Prometheus架构

Prometheus采用模块化的架构设计，主要由以下组件组成：

#### 3.1 Prometheus Server
- **核心组件**：负责数据采集、存储和查询
- **主要功能**：
  - 从配置的目标中抓取（scrape）监控数据
  - 将数据存储到本地时序数据库
  - 处理PromQL查询请求
  - 生成告警规则

#### 3.2 客户端库（Client Libraries）
- **作用**：为应用程序提供埋点接口，用于暴露自定义监控指标
- **支持语言**：Go、Java、Python、Node.js等多种编程语言

#### 3.3 Exporters
- **作用**：将第三方系统的监控数据转换为Prometheus支持的格式
- **常见类型**：
  - Node Exporter：监控服务器硬件和操作系统
  - MySQL Exporter：监控MySQL数据库
  - Redis Exporter：监控Redis缓存
  - Blackbox Exporter：监控网络服务的可用性

#### 3.4 Alertmanager
- **作用**：处理Prometheus Server生成的告警
- **主要功能**：
  - 告警分组（Grouping）：将相关告警合并为一个通知
  - 告警抑制（Inhibition）：抑制次要告警
  - 告警路由（Routing）：根据规则将告警发送到不同的接收者
  - 告警静默（Silencing）：暂时关闭特定告警

#### 3.5 可视化工具
- **内置界面**：Prometheus Server提供简单的查询和可视化界面
- **外部集成**：
  - Grafana：提供强大的数据可视化和仪表盘功能
  - PromDash：Prometheus官方的仪表盘工具（已不再维护）

#### 3.6 服务发现（Service Discovery）
- **作用**：自动发现和管理监控目标
- **支持类型**：
  - 静态配置（Static Configuration）
  - DNS服务发现（DNS SD）
  - 文件服务发现（File SD）
  - Kubernetes服务发现
  - Consul服务发现
  - EC2服务发现等

### 4. Prometheus数据模型

Prometheus采用多维数据模型，主要包含以下几个核心概念：

#### 4.1 时间序列标识
- **唯一标识**：每个时间序列由指标名称（metric name）和一组标签（labels）唯一标识
- **指标名称**：描述监控目标的一般特征，如`http_requests_total`、`node_cpu_seconds_total`
- **标签**：键值对，用于区分同一指标的不同实例或维度，如`instance="server1:8080"`、`job="api-server"`

#### 4.2 指标类型详解

##### 4.2.1 Counter（计数器）
- **特点**：单调递增，只能增加或重置为0
- **适用场景**：统计请求总数、错误数、完成的任务数等
- **示例**：`http_requests_total{method="GET", status="200"}`
- **常用操作**：计算增长率`rate(http_requests_total[5m])`

##### 4.2.2 Gauge（仪表）
- **特点**：可增可减，反映当前状态
- **适用场景**：CPU使用率、内存使用率、当前连接数等
- **示例**：`node_memory_MemFree_bytes`
- **常用操作**：直接读取当前值或计算变化率`delta(node_memory_MemFree_bytes[5m])`

##### 4.2.3 Histogram（直方图）
- **特点**：对观察值（如请求延迟）进行采样，并在预定义的桶（buckets）中进行计数
- **组成**：
  - `_bucket`：包含标签`le="上界"`，表示小于等于该上界的样本数
  - `_sum`：所有观察值的总和
  - `_count`：观察值的总数（等于最后一个bucket的值）
- **示例**：`http_request_duration_seconds_bucket{le="0.1"}`
- **常用操作**：计算分位数`histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`

##### 4.2.4 Summary（摘要）
- **特点**：直接计算并存储分位数，无需预定义桶
- **组成**：
  - `_quantile`：包含标签`quantile="分位数"`，表示对应分位数的值
  - `_sum`：所有观察值的总和
  - `_count`：观察值的总数
- **示例**：`http_request_duration_seconds_summary{quantile="0.95"}`
- **适用场景**：需要精确分位数的场景

### 5. PromQL查询语言

PromQL（Prometheus Query Language）是Prometheus的核心查询语言，支持对时间序列数据进行复杂的查询、过滤、聚合和计算。

#### 5.1 基本语法

##### 5.1.1 选择器
- **指标选择器**：`http_requests_total`
- **标签过滤**：`http_requests_total{method="GET", status="200"}`
- **正则表达式**：`http_requests_total{method=~"GET|POST"}`
- **排除标签**：`http_requests_total{status!="500"}`

##### 5.1.2 时间范围
- **相对时间**：`http_requests_total[5m]`（最近5分钟）
- **时间偏移**：`http_requests_total offset 1h`（1小时前的数据）
- **持续时间单位**：s（秒）、m（分钟）、h（小时）、d（天）、w（周）、y（年）

#### 5.2 常用函数

##### 5.2.1 时间序列处理函数
- **rate()**：计算每秒增长率（适用于Counter）
  ```promql
  rate(http_requests_total[5m])
  ```
- **irate()**：计算每秒瞬时增长率（更敏感，适用于快速变化的Counter）
  ```promql
  irate(http_requests_total[1m])
  ```
- **delta()**：计算时间范围内的变化量（适用于Gauge）
  ```promql
  delta(node_cpu_seconds_total[5m])
  ```
- **increase()**：计算时间范围内的增加量（适用于Counter）
  ```promql
  increase(http_requests_total[5m])
  ```

##### 5.2.2 聚合函数
- **sum()**：求和
  ```promql
  sum(http_requests_total)
  ```
- **avg()**：平均值
  ```promql
  avg(rate(http_requests_total[5m]))
  ```
- **min()**：最小值
- **max()**：最大值
- **count()**：计数
- **topk()**：取前N个最大值
  ```promql
  topk(3, rate(http_requests_total[5m]))
  ```

##### 5.2.3 高级函数
- **histogram_quantile()**：计算直方图的分位数
  ```promql
  histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
  ```
- **predict_linear()**：基于线性回归预测未来值
  ```promql
  predict_linear(node_filesystem_free_bytes[1h], 3600) < 0
  ```
- **vector()**：将标量转换为向量
- **scalar()**：将向量转换为标量

#### 5.3 查询结果类型
- **即时向量（Instant Vector）**：特定时间点的多个时间序列
- **区间向量（Range Vector）**：特定时间范围内的多个时间序列
- **标量（Scalar）**：单个数值
- **字符串（String）**：单个字符串值（较少使用）

### 6. Prometheus常用组件说明

#### 6.1 Node Exporter
- **简介**：最常用的系统监控exporter，用于收集Linux/Unix系统的硬件和操作系统指标
- **核心功能**：
  - CPU使用率、负载、上下文切换等
  - 内存、交换空间使用情况
  - 磁盘空间、I/O统计
  - 网络接口流量、连接数
  - 系统进程统计
- **默认端口**：9100
- **常用指标**：
  - `node_cpu_seconds_total`：CPU使用时间
  - `node_memory_MemTotal_bytes`/`node_memory_MemFree_bytes`：内存总量/空闲量
  - `node_filesystem_size_bytes`/`node_filesystem_free_bytes`：文件系统大小/可用空间
  - `node_network_transmit_bytes_total`/`node_network_receive_bytes_total`：网络发送/接收字节数

#### 6.2 Alertmanager
- **简介**：Prometheus的告警管理组件，处理来自Prometheus Server的告警
- **核心功能**：
  - **告警分组**：将相关告警合并为一个通知，减少通知数量
  - **告警抑制**：当一个主要告警触发时，抑制相关的次要告警
  - **告警路由**：根据规则将告警发送到不同的接收渠道（邮箱、Slack、PagerDuty等）
  - **告警静默**：暂时关闭特定时间内的特定告警
- **默认端口**：9093
- **配置文件**：`alertmanager.yml`，包含路由规则、接收器配置和抑制规则

#### 6.3 Grafana
- **简介**：开源的数据可视化平台，与Prometheus完美集成，提供丰富的图表和仪表盘
- **核心功能**：
  - 支持多种数据源（Prometheus、InfluxDB、Elasticsearch等）
  - 丰富的图表类型（折线图、柱状图、饼图、仪表盘等）
  - 灵活的仪表盘配置和模板
  - 告警和通知功能
  - 权限管理和团队协作
- **默认端口**：3000
- **常用配置**：
  - 添加Prometheus数据源：`http://prometheus-server:9090`
  - 创建仪表盘：使用PromQL查询构建图表
  - 导出/导入仪表盘：分享仪表盘配置

#### 6.4 Pushgateway
- **简介**：用于接收短生命周期任务的指标数据，并将其暴露给Prometheus抓取
- **使用场景**：
  - 批处理作业
  - 一次性任务
  - 无法被Prometheus直接抓取的临时任务
- **默认端口**：9091
- **注意事项**：
  - Pushgateway会持久化存储接收的指标，需要手动清理
  - 建议为推送的指标添加`job`和`instance`标签
  - 使用示例：`echo "job_duration_seconds{job='backup'} $(date +%s)" | curl --data-binary @- http://pushgateway:9091/metrics/job/backup`

#### 6.5 Blackbox Exporter
- **简介**：用于探测外部网络服务的可用性和性能
- **支持协议**：HTTP、HTTPS、DNS、TCP、ICMP等
- **核心功能**：
  - HTTP请求状态码、响应时间、证书过期时间
  - TCP端口连通性
  - DNS解析时间
  - ICMP ping延迟
- **默认端口**：9115
- **使用场景**：
  - 监控网站可用性
  - 检查API端点健康状况
  - 监控网络设备连通性

#### 6.6 MySQL Exporter
- **简介**：用于收集MySQL数据库的性能指标
- **核心功能**：
  - 连接数、查询执行时间
  - 缓存命中率、锁等待时间
  - 表空间使用情况
  - InnoDB性能指标
- **默认端口**：9104
- **配置要求**：
  - 创建具有`PROCESS`、`REPLICATION CLIENT`和`SELECT`权限的MySQL用户
  - 配置环境变量或配置文件指定MySQL连接信息

#### 6.7 Redis Exporter
- **简介**：用于收集Redis服务器的性能指标
- **核心功能**：
  - 内存使用情况、连接数
  - 命令执行统计、键空间信息
  - 复制状态、持久化信息
- **默认端口**：9121
- **使用方式**：`redis_exporter --redis.addr=redis://redis-server:6379 --redis.password=password`

## 高频面试题

### 1. Prometheus与传统监控系统（如Zabbix）的区别是什么？

**答案**：
- **数据模型**：Prometheus使用多维数据模型（指标+标签），支持灵活的查询；Zabbix使用基于主机的层次化数据模型
- **查询语言**：Prometheus有强大的PromQL查询语言；Zabbix的查询功能相对有限
- **采集方式**：Prometheus主要采用拉取（pull）模式；Zabbix主要采用推送（push）模式
- **扩展性**：Prometheus通过exporters支持广泛的第三方集成；Zabbix通过插件扩展，但集成难度较高
- **适用场景**：Prometheus更适合云原生和微服务架构；Zabbix更适合传统的物理服务器和虚拟机环境

### 2. Prometheus的数据模型是什么？有哪些指标类型？

**答案**：
- **数据模型**：Prometheus使用时间序列数据模型，每个时间序列由指标名称和一组标签唯一标识
- **指标类型**：
  - **Counter**：单调递增的计数器，如请求总数
  - **Gauge**：可增可减的仪表，如CPU使用率
  - **Histogram**：样本分布的直方图，如请求延迟分布
  - **Summary**：样本分布的摘要，如请求延迟的分位数

### 3. 什么是PromQL？它的主要功能是什么？

**答案**：
- PromQL（Prometheus Query Language）是Prometheus的查询语言
- **主要功能**：
  - 查询和过滤时间序列数据
  - 对数据进行聚合、计算和转换
  - 支持时间范围查询和时间偏移
  - 支持复杂的数学运算和逻辑运算
  - 用于创建图表和告警规则

### 4. Prometheus支持哪些数据采集方式？各有什么优缺点？

**答案**：
- **拉取模式（Pull）**：
  - **优点**：可以检测目标是否存活，配置简单，适合大多数场景
  - **缺点**：需要目标可被Prometheus访问，不适合短生命周期任务
- **推送模式（Push）**：
  - **优点**：适合短生命周期任务和批处理作业
  - **缺点**：需要额外的Pushgateway组件，增加系统复杂度，无法检测目标存活

### 5. 如何在Prometheus中实现告警？

**答案**：
1. **定义告警规则**：在Prometheus配置文件中定义告警规则，基于PromQL查询
2. **告警触发**：当查询结果满足告警条件时，Prometheus Server生成告警
3. **告警处理**：Alertmanager接收告警，进行分组、抑制和路由
4. **告警通知**：Alertmanager将告警发送到指定的接收渠道（邮箱、Slack等）

### 6. 什么是Node Exporter？它有什么作用？

**答案**：
- Node Exporter是Prometheus最常用的系统监控exporter
- **作用**：
  - 收集Linux/Unix系统的硬件和操作系统指标
  - 包括CPU、内存、磁盘、网络等系统资源使用情况
  - 以Prometheus格式暴露指标，供Prometheus Server抓取

### 7. Alertmanager的主要功能是什么？

**答案**：
- **告警分组**：将相关告警合并为一个通知，减少通知数量
- **告警抑制**：当一个主要告警触发时，抑制相关的次要告警
- **告警路由**：根据规则将告警发送到不同的接收渠道
- **告警静默**：暂时关闭特定时间内的特定告警

### 8. Grafana与Prometheus的关系是什么？

**答案**：
- Grafana是一个数据可视化平台，Prometheus是一个监控和告警工具包
- Grafana可以将Prometheus作为数据源，展示Prometheus收集的监控数据
- Grafana提供了比Prometheus内置界面更丰富的可视化功能和仪表盘配置
- 两者通常一起使用，形成完整的监控和可视化解决方案

### 9. 什么是Pushgateway？什么时候需要使用它？

**答案**：
- Pushgateway是Prometheus的一个组件，用于接收短生命周期任务的指标数据
- **使用场景**：
  - 批处理作业
  - 一次性任务
  - 无法被Prometheus直接抓取的临时任务
  - 无固定IP或端口的任务

### 10. Prometheus的存储机制是什么？有什么优缺点？

**答案**：
- **存储机制**：Prometheus使用本地时间序列数据库，将数据存储在磁盘上的TSDB（Time Series Database）中
- **优点**：
  - 本地存储，无需外部依赖
  - 高性能，适合高并发查询
  - 支持数据压缩，节省磁盘空间
- **缺点**：
  - 单节点存储容量有限
  - 水平扩展相对复杂
  - 长期存储需要外部解决方案（如Thanos、Cortex）

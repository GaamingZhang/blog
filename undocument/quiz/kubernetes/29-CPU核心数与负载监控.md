---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Linux
  - 监控
tag:
  - Linux
  - CPU
  - 监控
  - Prometheus
---

# 如何根据CPU不同的核心数来监控Linux的负载

## 概述

在Linux系统监控中，负载（Load Average）是一个关键指标，但单纯的负载数值并不能准确反映系统状态。必须结合CPU核心数来评估系统负载是否正常。本文将详细介绍如何根据CPU核心数来合理监控Linux负载。

## CPU核心数获取方法

### 查看CPU核心数

```bash
# 方法1：查看/proc/cpuinfo
$ grep -c 'model name' /proc/cpuinfo
8

# 方法2：使用nproc命令
$ nproc
8

# 方法3：使用lscpu命令
$ lscpu | grep 'CPU(s):'
CPU(s):                8

# 方法4：查看CPU详细信息
$ lscpu
Architecture:        x86_64
CPU op-mode(s):      32-bit, 64-bit
CPU(s):              8
On-line CPU(s) list: 0-7
Thread(s) per core:  2
Core(s) per socket:  4
Socket(s):           1
NUMA node(s):        1
```

### CPU核心数计算

```
总逻辑CPU数 = 物理CPU数 × 每CPU物理核心数 × 每核心线程数

示例：
- 物理CPU数：1
- 每CPU物理核心数：4
- 每核心线程数：2（超线程）
- 总逻辑CPU数：1 × 4 × 2 = 8
```

```bash
# 物理CPU数
$ grep 'physical id' /proc/cpuinfo | sort -u | wc -l
1

# 每CPU物理核心数
$ grep 'cpu cores' /proc/cpuinfo | uniq
cpu cores	: 4

# 每核心线程数
$ grep 'siblings' /proc/cpuinfo | uniq
siblings	: 8
```

## 负载与CPU核心数的关系

### 负载计算原理

```
+------------------+
|   理想负载状态    |
+------------------+

单核CPU：
  负载 = 1.0  → 100%利用率（满载）
  负载 < 1.0  → 有空闲
  负载 > 1.0  → 进程等待

多核CPU（N核）：
  负载 = N    → 100%利用率（满载）
  负载 < N    → 有空闲
  负载 > N    → 进程等待
```

### 负载率计算公式

```
负载率 = 当前负载 / CPU核心数 × 100%

示例（8核CPU）：
  负载 = 4.0  → 负载率 = 4/8 × 100% = 50%
  负载 = 8.0  → 负载率 = 8/8 × 100% = 100%
  负载 = 12.0 → 负载率 = 12/8 × 100% = 150%
```

## 监控策略设计

### 告警阈值设计

```
+------------------+------------------+------------------+
|   负载率范围     |     状态         |     建议操作     |
+------------------+------------------+------------------+
|    < 70%         |     正常         |     无需处理     |
|   70% - 100%     |     警告         |     关注趋势     |
|  100% - 150%     |     严重         |     需要处理     |
|    > 150%        |     危急         |     立即处理     |
+------------------+------------------+------------------+
```

### 不同核心数的阈值示例

```
+------------+--------+--------+--------+--------+
| CPU核心数  |  正常  |  警告  |  严重  |  危急  |
+------------+--------+--------+--------+--------+
|     2      |  <1.4  | 1.4-2  |  2-3   |  >3    |
|     4      |  <2.8  | 2.8-4  |  4-6   |  >6    |
|     8      |  <5.6  | 5.6-8  |  8-12  |  >12   |
|    16      |  <11.2 |11.2-16 | 16-24  |  >24   |
|    32      |  <22.4 |22.4-32 | 32-48  |  >48   |
+------------+--------+--------+--------+--------+
```

## Prometheus监控配置

### Node Exporter指标

```yaml
# node_load1   - 1分钟负载
# node_load5   - 5分钟负载
# node_load15  - 15分钟负载

# CPU核心数指标
# count(node_cpu_seconds_total{mode="idle"}) by (instance)
```

### Prometheus告警规则

```yaml
# /etc/prometheus/rules/node_alerts.yml
groups:
  - name: cpu_load_alerts
    rules:
      - alert: HighLoadWarning
        expr: |
          node_load1 
          / on(instance) 
          count(node_cpu_seconds_total{mode="idle"}) by (instance) 
          > 0.7
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High load on {{ $labels.instance }}"
          description: "Load average (1m) is {{ $value | printf \"%.2f\" }} times the number of CPUs"

      - alert: HighLoadCritical
        expr: |
          node_load1 
          / on(instance) 
          count(node_cpu_seconds_total{mode="idle"}) by (instance) 
          > 1.0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Critical load on {{ $labels.instance }}"
          description: "Load average (1m) is {{ $value | printf \"%.2f\" }} times the number of CPUs"

      - alert: SustainedHighLoad
        expr: |
          node_load15 
          / on(instance) 
          count(node_cpu_seconds_total{mode="idle"}) by (instance) 
          > 0.8
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Sustained high load on {{ $labels.instance }}"
          description: "15-minute load average indicates sustained high load"
```

### Grafana Dashboard配置

```json
{
  "panels": [
    {
      "title": "Load Average per CPU",
      "type": "graph",
      "targets": [
        {
          "expr": "node_load1 / on(instance) count(node_cpu_seconds_total{mode=\"idle\"}) by (instance)",
          "legendFormat": "{{ instance }} - 1m"
        },
        {
          "expr": "node_load5 / on(instance) count(node_cpu_seconds_total{mode=\"idle\"}) by (instance)",
          "legendFormat": "{{ instance }} - 5m"
        },
        {
          "expr": "node_load15 / on(instance) count(node_cpu_seconds_total{mode=\"idle\"}) by (instance)",
          "legendFormat": "{{ instance }} - 15m"
        }
      ],
      "alert": {
        "conditions": [
          {
            "evaluator": {
              "type": "gt",
              "params": [0.7]
            }
          }
        ]
      }
    },
    {
      "title": "CPU Cores",
      "type": "stat",
      "targets": [
        {
          "expr": "count(node_cpu_seconds_total{mode=\"idle\"}) by (instance)"
        }
      ]
    }
  ]
}
```

## 自定义监控脚本

### Shell脚本监控

```bash
#!/bin/bash
# load_monitor.sh - 负载监控脚本

CPU_CORES=$(nproc)
LOAD_1MIN=$(awk '{print $1}' /proc/loadavg)
LOAD_5MIN=$(awk '{print $2}' /proc/loadavg)
LOAD_15MIN=$(awk '{print $3}' /proc/loadavg)

LOAD_RATIO=$(echo "scale=2; $LOAD_1MIN / $CPU_CORES" | bc)

echo "CPU Cores: $CPU_CORES"
echo "Load Average: $LOAD_1MIN, $LOAD_5MIN, $LOAD_15MIN"
echo "Load Ratio: ${LOAD_RATIO}x"

if (( $(echo "$LOAD_RATIO > 1.5" | bc -l) )); then
    echo "[CRITICAL] Load is ${LOAD_RATIO}x CPU cores!"
    exit 2
elif (( $(echo "$LOAD_RATIO > 1.0" | bc -l) )); then
    echo "[WARNING] Load is ${LOAD_RATIO}x CPU cores!"
    exit 1
elif (( $(echo "$LOAD_RATIO > 0.7" | bc -l) )); then
    echo "[NOTICE] Load is ${LOAD_RATIO}x CPU cores"
    exit 0
else
    echo "[OK] Load is normal"
    exit 0
fi
```

### Python监控脚本

```python
#!/usr/bin/env python3
import os
import json
import requests

def get_cpu_cores():
    return os.cpu_count()

def get_load_average():
    with open('/proc/loadavg', 'r') as f:
        load = f.read().split()[:3]
        return float(load[0]), float(load[1]), float(load[2])

def calculate_load_ratio():
    cores = get_cpu_cores()
    load_1, load_5, load_15 = get_load_average()
    ratio = load_1 / cores
    return {
        'cores': cores,
        'load_1': load_1,
        'load_5': load_5,
        'load_15': load_15,
        'ratio': round(ratio, 2),
        'status': get_status(ratio)
    }

def get_status(ratio):
    if ratio > 1.5:
        return 'critical'
    elif ratio > 1.0:
        return 'warning'
    elif ratio > 0.7:
        return 'notice'
    return 'ok'

def send_to_prometheus_pushgateway(data, gateway_url, job_name):
    metrics = f"""
# TYPE load_ratio gauge
load_ratio{{cores="{data['cores']}"}} {data['ratio']}
# TYPE load_1min gauge
load_1min{{cores="{data['cores']}"}} {data['load_1']}
# TYPE load_5min gauge
load_5min{{cores="{data['cores']}"}} {data['load_5']}
# TYPE load_15min gauge
load_15min{{cores="{data['cores']}"}} {data['load_15']}
"""
    requests.post(
        f"{gateway_url}/metrics/job/{job_name}",
        data=metrics
    )

if __name__ == '__main__':
    data = calculate_load_ratio()
    print(json.dumps(data, indent=2))
```

## Zabbix监控配置

### UserParameter配置

```bash
# /etc/zabbix/zabbix_agentd.d/load.conf

# CPU核心数
UserParameter=cpu.cores,nproc

# 负载率（1分钟负载/CPU核心数）
UserParameter=cpu.load.ratio,awk '{print $1}' /proc/loadavg | awk -v cores=$(nproc) '{printf "%.2f", $1/cores}'

# 原始负载
UserParameter=system.cpu.load[all,avg1],awk '{print $1}' /proc/loadavg
UserParameter=system.cpu.load[all,avg5],awk '{print $2}' /proc/loadavg
UserParameter=system.cpu.load[all,avg15],awk '{print $3}' /proc/loadavg
```

### Zabbix触发器配置

```xml
<!-- 负载率告警触发器 -->
<trigger>
    <expression>{template:cpu.load.ratio.last()}&gt;1.5</expression>
    <name>High CPU Load Ratio on {HOST.NAME}</name>
    <priority>4</priority>
    <description>Load ratio is {ITEM.LASTVALUE} (load/CPU cores)</description>
</trigger>

<trigger>
    <expression>{template:cpu.load.ratio.last()}&gt;1.0</expression>
    <name>Warning CPU Load Ratio on {HOST.NAME}</name>
    <priority>3</priority>
</trigger>
```

## 不同场景的监控策略

### Web服务器

```yaml
# Web服务器通常I/O等待较少
# 主要关注CPU使用率

alerts:
  - name: web_high_load
    condition: load_ratio > 0.8
    duration: 3m
    action: scale_out
```

### 数据库服务器

```yaml
# 数据库服务器可能有较多I/O等待
# 需要区分CPU负载和I/O负载

alerts:
  - name: db_high_load
    condition: load_ratio > 0.9 AND io_wait < 30%
    duration: 5m
    action: optimize_queries
  
  - name: db_high_io_wait
    condition: io_wait > 40%
    duration: 3m
    action: check_disk_io
```

### 计算节点

```yaml
# 计算节点预期高负载
# 阈值可以设置更高

alerts:
  - name: compute_high_load
    condition: load_ratio > 1.2
    duration: 10m
    action: redistribute_tasks
```

## 监控最佳实践

### 1. 多维度监控

```
+------------------+
|   监控维度        |
+------------------+
|  负载率          |  ← 主要指标
|  CPU使用率       |  ← 辅助指标
|  I/O等待         |  ← 区分负载类型
|  进程队列长度     |  ← 趋势分析
+------------------+
```

### 2. 趋势分析

```yaml
# 同时监控1分钟、5分钟、15分钟负载
# 判断负载趋势

rules:
  - alert: RisingLoad
    expr: node_load1 > node_load15 * 1.5
    for: 5m
    annotations:
      summary: "Load is rising rapidly"
```

### 3. 动态阈值

```yaml
# 根据业务时段设置不同阈值

groups:
  - name: dynamic_load_alerts
    rules:
      - alert: BusinessHoursHighLoad
        expr: |
          node_load1 / on(instance) count(node_cpu_seconds_total{mode="idle"}) by (instance) > 0.8
          and hour() >= 9 and hour() <= 18
        labels:
          severity: warning
```

## 常见问题

### Q1: 超线程如何计算核心数？

```
超线程技术让一个物理核心模拟两个逻辑核心

建议：
- 计算密集型任务：按物理核心数计算
- I/O密集型任务：按逻辑核心数计算
- 通用场景：按逻辑核心数计算，阈值适当提高
```

### Q2: 容器环境如何计算？

```yaml
# Kubernetes中获取CPU限制
# spec.containers[].resources.limits.cpu

# Prometheus查询容器CPU限制
sum(kube_pod_container_resource_limits{resource="cpu", unit="core"}) 
  by (pod, namespace)
```

### Q3: NUMA架构注意事项？

```bash
# 查看NUMA节点
$ numactl --hardware

# NUMA架构下，跨节点访问会影响性能
# 建议绑定进程到特定NUMA节点
$ numactl --cpunodebind=0 --membind=0 <command>
```

## 参考资源

- [Linux Load Averages: Solving the Mystery](https://www.brendangregg.com/blog/2017-08-08/linux-load-averages.html)
- [Prometheus Node Exporter](https://github.com/prometheus/node_exporter)
- `man proc` - /proc/loadavg documentation

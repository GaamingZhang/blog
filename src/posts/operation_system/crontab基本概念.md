---
date: 2026-01-07
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 操作系统
tag:
  - 操作系统
---

# crontab:Linux 的定时任务调度器

## crontab 是什么

**crontab** 是 Linux/Unix 系统中用于定期自动执行任务的工具,就像给系统设置闹钟一样,到点自动运行指定的命令或脚本。名称来源于"cron table",cron 在希腊语中意为"时间"。

### 为什么需要 crontab

**自动化运维**:
- 定期备份:每天凌晨自动备份数据库
- 日志清理:每周删除旧日志文件
- 系统维护:定期更新软件包

**监控告警**:
- 健康检查:每5分钟检查服务状态
- 资源监控:每小时记录CPU和内存使用
- 磁盘告警:检测磁盘空间不足

**数据处理**:
- 报表生成:每天生成业务报表
- 数据同步:定期从远程服务器拉取数据
- 统计分析:每小时统计访问日志

### crontab vs 手动执行

| 特性     | 手动执行     | crontab      |
| -------- | ------------ | ------------ |
| 执行方式 | 人工触发     | 自动触发     |
| 可靠性   | 容易忘记     | 准时执行     |
| 一致性   | 时间不固定   | 精确到分钟   |
| 工作负担 | 需要人工介入 | 完全自动化   |
| 适用场景 | 临时任务     | 周期性任务   |

---

## crontab 的工作原理

### cron 守护进程

crontab 的工作依赖于 cron 守护进程:

```
系统启动
   ↓
启动 cron 守护进程
   ↓
每分钟醒来一次
   ↓
检查所有 crontab 配置
   ↓
匹配当前时间?
├─ 是 → 执行对应任务
└─ 否 → 继续等待
   ↓
下一分钟再次检查
```

**关键特性**:
- **周期唤醒**:每分钟检查一次,精度为1分钟
- **多用户**:每个用户可以有独立的定时任务
- **后台运行**:作为守护进程持续运行

### 配置文件层级

crontab 有多个配置文件,按优先级加载:

```
1. 系统级配置
   ├─ /etc/crontab              (系统主配置)
   └─ /etc/cron.d/*             (系统服务的定时任务)

2. 目录级快捷方式
   ├─ /etc/cron.hourly/         (每小时执行的脚本)
   ├─ /etc/cron.daily/          (每天执行的脚本)
   ├─ /etc/cron.weekly/         (每周执行的脚本)
   └─ /etc/cron.monthly/        (每月执行的脚本)

3. 用户级配置
   └─ /var/spool/cron/username  (个人定时任务)
```

**配置建议**:
- 系统维护任务 → 放在 /etc/crontab 或 /etc/cron.d/
- 个人任务 → 使用 crontab -e 编辑用户配置
- 简单周期任务 → 直接放脚本到 /etc/cron.daily/ 等目录

### 任务执行流程

```
时间到达
   ↓
cron 进程检测到匹配
   ↓
检查权限
   ↓
创建子进程
   ↓
设置环境变量 (最小PATH)
   ↓
执行命令/脚本
   ↓
收集输出 (stdout + stderr)
   ↓
输出处理
├─ 有输出 → 发送邮件给用户
└─ 重定向 → 写入指定文件
```

**注意事项**:
- cron 任务的环境变量与登录会话不同
- 默认 PATH 只包含 /usr/bin 和 /bin
- 建议使用绝对路径执行命令

---

## crontab 的时间格式

### 五字段时间格式

crontab 使用5个字段定义时间:

```
* * * * * command
│ │ │ │ │
│ │ │ │ └─ 星期几 (0-7, 0和7都代表周日)
│ │ │ └─── 月份 (1-12)
│ │ └───── 日期 (1-31)
│ └─────── 小时 (0-23)
└───────── 分钟 (0-59)
```

**特殊字符含义**:

| 字符 | 含义           | 示例                      | 说明                   |
| ---- | -------------- | ------------------------- | ---------------------- |
| *    | 任意值         | `* * * * *`               | 每分钟执行             |
| ,    | 多个值         | `0 9,12,18 * * *`         | 9点、12点、18点        |
| -    | 范围           | `0 9-17 * * *`            | 9点到17点每小时        |
| /    | 步长           | `*/5 * * * *`             | 每5分钟                |

### 时间格式示例

**固定时间**:
```bash
# 每天凌晨2点
0 2 * * *

# 每周一早上9点
0 9 * * 1

# 每月1号凌晨3点
0 3 1 * *
```

**周期执行**:
```bash
# 每5分钟
*/5 * * * *

# 每小时的第30分钟
30 * * * *

# 每2小时
0 */2 * * *
```

**工作时间**:
```bash
# 工作日(周一到周五)每小时
0 * * * 1-5

# 工作日9点到18点每30分钟
*/30 9-18 * * 1-5
```

**特殊快捷方式**:

| 快捷方式  | 等价表达式  | 说明             |
| --------- | ----------- | ---------------- |
| @reboot   | (无)        | 系统启动时执行   |
| @hourly   | `0 * * * *` | 每小时           |
| @daily    | `0 0 * * *` | 每天凌晨         |
| @weekly   | `0 0 * * 0` | 每周日凌晨       |
| @monthly  | `0 0 1 * *` | 每月1号凌晨      |
| @yearly   | `0 0 1 1 *` | 每年1月1号凌晨   |

---

## crontab 的基本使用

### 常用命令

**查看当前定时任务**:
```bash
crontab -l
```

**编辑定时任务**:
```bash
crontab -e
```

**删除所有定时任务**:
```bash
crontab -r
```

**管理其他用户的定时任务**(需要root权限):
```bash
crontab -u username -l    # 查看
crontab -u username -e    # 编辑
```

### 任务格式

```bash
# 分 时 日 月 周 命令
0 2 * * * /usr/local/bin/backup.sh

# 可以添加注释
# 每天凌晨2点备份数据库
0 2 * * * /usr/local/bin/backup.sh

# 重定向输出到日志文件
0 2 * * * /usr/local/bin/backup.sh >> /var/log/backup.log 2>&1
```

**输出处理**:
- 不重定向:输出会通过邮件发送给用户
- `> /dev/null 2>&1`:丢弃所有输出
- `>> /path/to/log 2>&1`:追加到日志文件

---

## 常见使用场景

### 场景1:数据库备份

**需求**:每天凌晨2点备份MySQL数据库

```bash
0 2 * * * /usr/bin/mysqldump -u root -pPassword dbname > /backup/db_$(date +\%Y\%m\%d).sql
```

**注意**:`%`符号需要用反斜杠转义(`\%`)

### 场景2:日志清理

**需求**:每周日凌晨3点清理7天前的日志

```bash
0 3 * * 0 find /var/log/myapp -type f -mtime +7 -delete
```

### 场景3:系统监控

**需求**:每5分钟检查磁盘使用率

```bash
*/5 * * * * /usr/local/bin/check_disk.sh
```

### 场景4:定期同步

**需求**:每小时从远程服务器同步文件

```bash
0 * * * * rsync -avz user@remote:/data/ /local/backup/
```

### 场景5:报表生成

**需求**:每天早上9点生成前一天的业务报表

```bash
0 9 * * * /usr/local/bin/generate_report.sh
```

---

## 重要注意事项

### 环境变量问题

**问题**:cron 任务的环境变量与登录会话不同

**默认 PATH**:
```
PATH=/usr/bin:/bin
```

**解决方案**:

1. 使用绝对路径:
```bash
# 错误
*/5 * * * * python script.py

# 正确
*/5 * * * * /usr/bin/python3 /home/user/script.py
```

2. 在 crontab 文件开头设置环境变量:
```bash
PATH=/usr/local/bin:/usr/bin:/bin
SHELL=/bin/bash

*/5 * * * * my_command
```

### 特殊字符转义

**问题**:`%` 在 crontab 中是特殊字符,代表换行

**解决方案**:使用反斜杠转义

```bash
# 错误
0 0 * * * echo $(date +%Y-%m-%d) > /tmp/date.txt

# 正确
0 0 * * * echo $(date +\%Y-\%m-\%d) > /tmp/date.txt
```

### 权限问题

**问题**:脚本没有执行权限

**解决方案**:
```bash
chmod +x /usr/local/bin/script.sh
```

### 并发控制

**问题**:同一任务的多个实例同时运行

**解决方案**:使用文件锁
```bash
*/5 * * * * flock -n /tmp/script.lock -c "/usr/local/bin/script.sh"
```

### 长任务超时

**问题**:任务执行时间过长

**解决方案**:使用 timeout 命令限制时间
```bash
*/1 * * * * timeout 30m /usr/local/bin/long_task.sh
```

---

## crontab vs 其他工具

### systemd timers

**systemd timers** 是现代 Linux 系统的替代方案:

**优势**:
- 支持微秒级精度
- 与 systemd 服务集成
- 支持随机延迟(避免任务集中)
- 丰富的状态和日志

**劣势**:
- 配置复杂(需要.timer和.service两个文件)
- 学习曲线陡

**选择建议**:
- 简单定时任务 → crontab
- 复杂系统服务调度 → systemd timers

### anacron

**anacron** 适合不连续运行的系统(如笔记本电脑):

**优势**:
- 自动补执行错过的任务
- 配置简单

**劣势**:
- 只支持天级调度(不支持分钟级)
- 功能相对简单

**选择建议**:
- 服务器 → crontab
- 个人电脑 → anacron

### at 命令

**at** 用于一次性任务:

**用途**:
- 在指定时间执行一次命令
- 任务执行后自动删除

**示例**:
```bash
# 5分钟后执行
at now + 5 minutes
> echo "Hello" > /tmp/hello.txt
```

**选择建议**:
- 周期性任务 → crontab
- 一次性任务 → at

---

## 调试和排查

### 检查 cron 服务

```bash
# 查看服务状态
systemctl status cron

# 启动服务
systemctl start cron

# 设置开机自启
systemctl enable cron
```

### 查看执行日志

**系统日志**:
```bash
# CentOS/RHEL
tail -f /var/log/cron

# Debian/Ubuntu
tail -f /var/log/syslog | grep CRON
```

**自定义日志**:
```bash
# 在任务中添加日志输出
*/5 * * * * /usr/local/bin/script.sh >> /var/log/script.log 2>&1
```

### 快速测试

**方法1**:设置为每分钟执行
```bash
*/1 * * * * /usr/local/bin/script.sh >> /tmp/test.log 2>&1
```

**方法2**:手动模拟 cron 环境
```bash
env -i /bin/sh -c '/usr/local/bin/script.sh'
```

**方法3**:直接执行脚本
```bash
/usr/local/bin/script.sh
```

### 常见错误

**任务不执行**:
- 检查 cron 服务是否运行
- 确认时间格式正确
- 使用绝对路径
- 检查脚本执行权限

**输出为空**:
- 添加日志输出:`>> /tmp/debug.log 2>&1`
- 检查环境变量
- 手动执行脚本测试

**权限错误**:
- 确保脚本有执行权限(`chmod +x`)
- 检查任务所有者权限

---

## 核心要点

**crontab 的本质**:Linux 系统的定时任务调度器,按照设定的时间自动执行命令。

**核心概念**:
- **cron 守护进程**:每分钟检查一次配置,到时执行任务
- **五字段时间**:分 时 日 月 周,精确到分钟
- **配置层级**:系统级、目录级、用户级三层配置

**典型应用**:
- **数据备份**:定期备份数据库、文件
- **日志管理**:清理旧日志、归档
- **监控告警**:检查服务状态、磁盘空间
- **数据处理**:生成报表、同步数据

**使用技巧**:
- 使用绝对路径避免 PATH 问题
- `%`符号需要转义为`\%`
- 输出重定向到日志文件便于调试
- 使用 `flock` 避免任务并发

**常见陷阱**:
- cron 环境变量与登录会话不同
- 默认 PATH 只有 /usr/bin 和 /bin
- 忘记给脚本执行权限
- 时间格式错误导致任务不执行

**调试方法**:
- 检查 cron 服务状态
- 查看系统日志:`/var/log/cron` 或 `/var/log/syslog`
- 设置为每分钟执行快速验证
- 手动执行脚本测试

**最佳实践**:
- 重要任务添加日志输出
- 使用 `timeout` 限制长任务执行时间
- 定期备份 crontab 配置
- 避免在高峰期运行资源密集型任务
- 为关键任务添加告警通知

## 参考资源

- [Linux crontab 命令手册](https://man7.org/linux/man-pages/man5/crontab.5.html)
- [cron 守护进程文档](https://man7.org/linux/man-pages/man8/cron.8.html)
- [systemd timers 替代方案](https://www.freedesktop.org/software/systemd/man/systemd.timer.html)

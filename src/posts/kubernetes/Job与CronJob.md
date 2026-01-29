# Job与CronJob

## 为什么需要Job和CronJob？

在Kubernetes的世界里，大多数应用都是"长跑型选手"——比如Web服务器，它们启动后就一直运行，等待处理请求。但生活中还有很多"短跑型任务"：备份数据库、发送报表邮件、清理过期日志、批量处理数据等等。

想象一下，如果你让一个Web服务器来做"每天凌晨备份数据库"这件事，就像让一个全职员工每天只工作5分钟——太浪费了！这就是Job和CronJob存在的意义。

**Job**就像你雇佣的临时工：来了，干完活，走人。
**CronJob**就像定时闹钟+临时工的组合：闹钟响了，自动派一个临时工来干活。

## Job：一次性任务的执行者

### Job是什么？

Job是Kubernetes中专门用来运行"会结束的任务"的资源。它会创建一个或多个Pod来执行任务，并确保任务成功完成。

与Deployment不同的是：
- Deployment的Pod挂了会自动重启，永远保持运行
- Job的Pod完成任务后就正常退出，任务失败才会重试

### Job的核心概念

想象你是一个快递站的站长，现在有100个包裹要送：

- **completions（完成数）**：需要成功完成的任务数。比如100个包裹就是completions: 100
- **parallelism（并行数）**：同时派出几个快递员。parallelism: 5表示同时5个人在送
- **backoffLimit（重试次数）**：快递员送失败了，最多让他重试几次
- **activeDeadlineSeconds（超时时间）**：整个送货任务的最长时间，超时就算失败

### 最简单的Job示例

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: hello-job
spec:
  template:
    spec:
      containers:
        - name: hello
          image: busybox
          command: ["echo", "Hello, Kubernetes Job!"]
      restartPolicy: Never  # Job必须设置为Never或OnFailure
```

这个Job启动后，会打印一句话，然后结束。就像你让临时工来喊一嗓子，喊完就可以走了。

### 并行批处理示例

当你需要处理大量数据时，可以用并行Job：

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: batch-processor
spec:
  completions: 10     # 需要成功完成10个任务
  parallelism: 3      # 同时运行3个Pod
  backoffLimit: 5     # 失败最多重试5次
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: worker
          image: my-processor:1.0
```

这就像同时派3个快递员出去，直到10个包裹全部送达。

### 重启策略的选择

Job的Pod有两种重启策略：

- **Never**：失败后创建新Pod重试。优点是失败的Pod会保留，方便你查日志找原因
- **OnFailure**：失败后在同一个Pod里重启容器。优点是不会留下一堆失败的Pod

选择建议：开发调试阶段用Never（方便看日志），生产环境用OnFailure（更干净）。

## CronJob：定时任务调度器

### CronJob是什么？

CronJob = Cron（定时器）+ Job。它按照你设定的时间表，自动创建Job来执行任务。

如果说Job是临时工，那CronJob就是人力资源部门——它会在指定时间自动招一个临时工来干活。

### Cron表达式

Cron表达式是用来描述"什么时候执行"的语法，由5个字段组成：

```
┌───────────── 分钟 (0 - 59)
│ ┌───────────── 小时 (0 - 23)
│ │ ┌───────────── 日期 (1 - 31)
│ │ │ ┌───────────── 月份 (1 - 12)
│ │ │ │ ┌───────────── 星期 (0 - 6，0是周日)
│ │ │ │ │
* * * * *
```

**常用示例**：

| 表达式 | 含义 |
|--------|------|
| `0 2 * * *` | 每天凌晨2点 |
| `*/15 * * * *` | 每15分钟 |
| `0 9 * * 1-5` | 工作日早上9点 |
| `0 0 1 * *` | 每月1号午夜 |

### CronJob示例：每日备份

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: daily-backup
spec:
  schedule: "0 2 * * *"           # 每天凌晨2点
  timeZone: "Asia/Shanghai"       # 使用上海时区
  concurrencyPolicy: Forbid       # 禁止并发，上一个没跑完就跳过
  successfulJobsHistoryLimit: 3   # 保留最近3个成功的Job
  failedJobsHistoryLimit: 1       # 保留最近1个失败的Job
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: backup
              image: backup-tool:1.0
              command: ["/backup.sh"]
```

### 并发策略：当任务"撞车"时怎么办？

如果凌晨2点的备份任务还没跑完，2点的下一个任务又要开始了，怎么办？这就是`concurrencyPolicy`要解决的问题：

- **Allow**（默认）：允许同时运行多个Job。适合任务之间互不影响的场景
- **Forbid**：跳过新任务，等当前任务完成。适合不能并发的任务（如数据库备份）
- **Replace**：停掉正在运行的，启动新的。适合只关心最新数据的场景

## Job与CronJob的关系

```
CronJob（调度器）
    │
    │ 按schedule时间创建
    ▼
  Job（一次性任务）
    │
    │ 创建Pod执行
    ▼
  Pod（实际干活的）
```

你可以把它们理解为：
- CronJob是"排班表"
- Job是"工单"
- Pod是"实际干活的工人"

## 常见问题

### Q1: Job的Pod一直在Running，不结束怎么办？

这通常是因为你的程序没有正常退出。Job期望程序完成后返回退出码0（成功）或非0（失败），如果程序一直运行，Job就会一直等。

**排查方法**：
- 检查你的程序是否有正常的退出逻辑
- 查看Pod日志：`kubectl logs <pod-name>`
- 设置超时时间`activeDeadlineSeconds`作为保底

### Q2: CronJob没有按时执行？

**常见原因**：

1. **并发策略阻止**：如果设置了`Forbid`，上一个Job还在跑，新的就会被跳过
2. **时区问题**：没设置`timeZone`时使用的是集群时区，可能和你预期的不一样
3. **错过太多次**：如果因为各种原因错过了超过100次调度，CronJob会停止创建新Job

### Q3: 如何确保Job的任务不会重复执行？

在分布式系统中，Job可能因为各种原因重试。你需要让任务具备"幂等性"——即使执行多次，结果也和执行一次一样。

**实践建议**：
- 处理前先检查是否已经处理过（比如在数据库记录处理状态）
- 使用唯一标识来追踪任务
- 用数据库事务保证原子性

### Q4: 如何手动触发一个CronJob？

有时候你不想等到计划时间，想立即执行一次：

```bash
kubectl create job manual-backup --from=cronjob/daily-backup
```

这会基于CronJob的模板立即创建一个Job。

### Q5: Job失败了怎么查原因？

```bash
# 查看Job的状态和事件
kubectl describe job <job-name>

# 找到Job创建的Pod
kubectl get pods --selector=job-name=<job-name>

# 查看失败Pod的日志
kubectl logs <pod-name>
```

如果使用了`restartPolicy: Never`，失败的Pod会保留下来，方便你调试。

## 小结

- **Job**适合一次性任务：数据迁移、批量处理、测试任务等
- **CronJob**适合定时任务：定期备份、报表生成、日志清理等
- 选择合适的`restartPolicy`和`concurrencyPolicy`能避免很多问题
- 让你的任务具备幂等性，这样即使重试也不会出问题

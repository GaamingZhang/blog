---
date: 2026-02-06
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - ClaudeCode
---

# Operator应用实践

## Prometheus Operator——监控场景的最佳实践

### 为什么需要Prometheus Operator

在没有Operator之前,在Kubernetes上部署Prometheus需要:

1. 手动编写Prometheus配置文件,定义抓取目标(scrape_configs)
2. 每次新增服务时,修改ConfigMap并重启Prometheus
3. 配置告警规则(alert rules)时,同样需要修改ConfigMap并重载

这种方式存在两个核心问题:

- **配置与应用耦合**:服务开发者需要了解Prometheus配置语法
- **无法动态发现**:新增Pod时需要手动更新配置

Prometheus Operator通过引入**声明式CRD**解决了这些问题。

### 核心CRD

Prometheus Operator定义了5个关键CRD:

| CRD | 作用 |
|-----|------|
| Prometheus | 定义Prometheus实例(副本数、存储、资源限制) |
| ServiceMonitor | 定义监控目标:通过Label Selector选择Service |
| PodMonitor | 直接监控Pod,不经过Service |
| PrometheusRule | 定义告警和记录规则 |
| Alertmanager | 定义Alertmanager实例 |

### ServiceMonitor的工作原理

假设你有一个web应用,通过Service暴露指标端点:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: web-app
  labels:
    app: web
spec:
  selector:
    app: web
  ports:
  - name: metrics  # 端口名很重要,ServiceMonitor会引用
    port: 8080
```

创建对应的ServiceMonitor:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: web-monitor
spec:
  selector:
    matchLabels:
      app: web       # 选择带有app=web标签的Service
  endpoints:
  - port: metrics    # 引用Service中的端口名
    interval: 30s    # 抓取间隔
    path: /metrics   # 指标路径
```

**自动发现流程**:

1. Prometheus Operator的控制器监听ServiceMonitor资源
2. 根据`selector`找到匹配的Service
3. 通过Service的Endpoints找到后端所有Pod的IP
4. 生成Prometheus配置文件中的`scrape_configs`
5. 通过Prometheus的配置热加载接口(`/-/reload`)更新配置

核心优势:服务开发者只需创建ServiceMonitor,无需接触Prometheus配置文件。当Pod扩缩容时,Endpoints自动更新,Prometheus自动感知。

### PrometheusRule示例

定义告警规则同样是声明式的:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: web-alerts
spec:
  groups:
  - name: web-app
    interval: 30s
    rules:
    - alert: HighErrorRate
      expr: |
        rate(http_requests_total{status=~"5.."}[5m]) > 0.05
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "高错误率告警"
```

Operator会将这些规则聚合到Prometheus的配置中,并动态加载。

### 适用场景

Prometheus Operator适合:

- **微服务架构**:服务数量多,频繁扩缩容
- **多团队协作**:各团队独立管理自己的ServiceMonitor
- **GitOps流程**:监控配置代码化,通过PR审核

## MySQL Operator——有状态应用的复杂编排

### MySQL集群的运维挑战

MySQL主从集群的典型拓扑:

```
Primary (主节点)
  ├─ Replica-1 (从节点)
  └─ Replica-2 (从节点)
```

运维需要处理:

- **主从复制配置**:从节点执行`CHANGE MASTER TO`指向主节点
- **主节点故障切换**:检测主节点心跳丢失,提升某个从节点为新主
- **数据一致性保证**:切换前确保从节点数据已完全同步(GTID一致)
- **定期备份**:执行`mysqldump`或物理备份,上传到对象存储
- **版本升级**:逐个节点升级二进制,避免停机

这些操作如果手动执行,容易出错且无法应对大规模集群。

### Oracle MySQL Operator的实现

以**MySQL Operator for Kubernetes**(Oracle官方)为例,它定义了两个核心CRD:

**InnoDBCluster** —— 定义MySQL InnoDB Cluster(基于Group Replication):

```yaml
apiVersion: mysql.oracle.com/v2
kind: InnoDBCluster
metadata:
  name: mysql-cluster
spec:
  secretName: mysql-root-password
  instances: 3              # 集群节点数
  version: "8.0.35"         # MySQL版本
  router:                   # MySQL Router配置(读写分离)
    instances: 2
  backupProfiles:
  - name: daily-backup
    dumpInstance:
      storage:
        persistentVolumeClaim:
          claimName: backup-pvc
  backupSchedules:
  - name: daily
    schedule: "0 2 * * *"   # 每天凌晨2点备份
    backupProfileName: daily-backup
```

**MySQLBackup** —— 定义备份任务:

```yaml
apiVersion: mysql.oracle.com/v2
kind: MySQLBackup
metadata:
  name: on-demand-backup
spec:
  clusterName: mysql-cluster
  backupProfileName: daily-backup
```

### Operator的关键实现逻辑

**主节点选举**:

1. 创建InnoDBCluster后,Operator部署StatefulSet管理3个Pod
2. 在第一个Pod(`mysql-cluster-0`)中初始化Group Replication并设为Primary
3. 后续Pod启动后自动加入集群并配置为Secondary

**故障自愈**:

- StatefulSet保证Pod崩溃后自动重建
- Operator检测到Pod重建后,执行`STOP GROUP_REPLICATION; START GROUP_REPLICATION;`重新加入集群
- 如果Primary节点不可用,Group Replication的自动选举机制会提升新Primary,Operator更新Service指向新主

**备份恢复**:

- 备份时,Operator在集群的某个Secondary节点上执行`mysqlsh`的`dumpInstance`命令
- 恢复时,创建新的InnoDBCluster并从备份中加载数据

### 适用场景

MySQL Operator适合:

- **需要高可用**:自动故障切换,RTO通常在1分钟内
- **运维标准化**:所有MySQL集群使用统一的部署和备份策略
- **自助服务**:开发团队通过创建CR自行申请数据库,无需DBA介入

**不适合的场景**:

- 对Kubernetes不熟悉的团队(学习成本高)
- 需要精细调优的场景(Operator的抽象可能限制灵活性)
- 小规模部署(单节点数据库用StatefulSet已足够)

## Cert-Manager——自动化证书管理

### 解决的问题

传统HTTPS证书管理流程:

1. 向CA(如Let's Encrypt)申请证书
2. 完成域名验证(DNS-01或HTTP-01 Challenge)
3. 将证书和私钥存储为Kubernetes Secret
4. 配置Ingress引用该Secret
5. 证书过期前手动续期

Cert-Manager将整个流程自动化。

### 核心CRD

**Issuer/ClusterIssuer** —— 定义证书颁发者:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-account-key
    solvers:
    - http01:
        ingress:
          class: nginx
```

**Certificate** —— 定义证书需求:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: web-tls
spec:
  secretName: web-tls-secret  # 证书存储到的Secret名
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - example.com
  - www.example.com
```

创建Certificate后,Cert-Manager会:

1. 向Let's Encrypt发起证书申请
2. 创建临时Ingress响应HTTP-01验证
3. 验证通过后获取证书
4. 将证书写入指定的Secret
5. 在证书到期前30天自动续期

### 适用场景

- **大量域名管理**:每个微服务一个域名,手动管理不现实
- **安全合规**:证书自动轮换,降低泄露风险
- **云原生环境**:与Ingress、Gateway API无缝集成

## 选择Operator还是Helm Chart

| 维度 | Operator | Helm Chart |
|------|----------|-----------|
| 复杂度 | 需要理解CRD、控制器 | 模板化YAML,学习成本低 |
| 自动化能力 | 持续监控和调谐 | 仅安装时执行,无后续管理 |
| 适用场景 | 有状态应用、需要Day 2运维 | 无状态应用、简单部署 |
| 示例 | MySQL集群、Kafka集群 | Nginx、Redis单机版 |

**何时使用Operator**:

- 应用有复杂的生命周期管理需求(备份、升级、故障切换)
- 需要动态响应外部变化(如Prometheus自动发现)
- 运维知识可以标准化为代码

**何时不该用Operator**:

- 简单的无状态应用(用Deployment + Service即可)
- 团队没有Go开发能力且现有Operator无法满足需求
- 一次性部署任务(用Job或Helm更合适)

## Operator开发框架

### Operator SDK vs Kubebuilder

| 框架 | 维护方 | 特点 |
|------|--------|------|
| Operator SDK | Red Hat | 支持Helm、Ansible、Go三种开发方式 |
| Kubebuilder | Kubernetes SIG | Go原生,与controller-runtime深度集成 |

两者在Go模式下底层都使用**controller-runtime**库,差异主要在脚手架代码生成和CRD管理方式上。

### 快速开发流程(以Kubebuilder为例)

**1. 初始化项目**:

```bash
kubebuilder init --domain example.com --repo github.com/myorg/myoperator
```

**2. 创建API**:

```bash
kubebuilder create api --group apps --version v1 --kind MyApp
```

自动生成:
- `api/v1/myapp_types.go`:定义Spec和Status结构体
- `controllers/myapp_controller.go`:Reconcile逻辑框架
- CRD YAML文件

**3. 实现Reconcile逻辑**:

```go
func (r *MyAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
  // 获取MyApp对象
  app := &appsv1.MyApp{}
  if err := r.Get(ctx, req.NamespacedName, app); err != nil {
    return ctrl.Result{}, client.IgnoreNotFound(err)
  }

  // 创建或更新Deployment
  deployment := constructDeployment(app)
  if err := r.Create(ctx, deployment); err != nil {
    // 处理已存在的情况
  }

  return ctrl.Result{}, nil
}
```

**4. 部署到集群**:

```bash
make install   # 安装CRD
make run       # 本地运行Controller
```

生产环境使用:

```bash
make docker-build docker-push IMG=myregistry/myoperator:v1
make deploy IMG=myregistry/myoperator:v1
```

### Operator成熟度模型

Red Hat定义了5个成熟度等级:

1. **Basic Install**:自动化部署
2. **Seamless Upgrades**:支持无缝升级
3. **Full Lifecycle**:备份、恢复、故障切换
4. **Deep Insights**:暴露指标,集成监控
5. **Auto Pilot**:自动调优、自动扩缩容

开发Operator时应根据实际需求选择目标等级,避免过度设计。

## 小结

Operator适合将**运维专业知识产品化**的场景:

- **Prometheus Operator**:将监控配置从静态文件变为动态声明式资源
- **MySQL Operator**:将数据库运维经验(主从切换、备份恢复)编码为自动化流程
- **Cert-Manager**:将证书申请和续期从手动任务变为后台自动执行

选择或开发Operator前,需要评估:

1. **复杂度收益比**:运维成本节省是否大于开发和维护成本
2. **团队能力**:是否有Go开发经验和Kubernetes深度理解
3. **替代方案**:是否有成熟的开源Operator可直接使用

在合适的场景下,Operator能显著提升运维效率,但过度使用会增加系统复杂度。理解其原理后,才能做出正确的技术选型。

---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - 安全
  - CIS Benchmark
  - Trivy
  - Falco
---

# Kubernetes 容器安全 CIS 基线实践

某个深夜，一条告警触发：生产集群中有容器正在执行 `/bin/bash`，访问路径是 `/etc/shadow`。你打开日志一看，那个容器的镜像里携带了一个高危 CVE——三个月前就已发布补丁，但没有人知道。更糟糕的是，集群的 API Server 还在接受匿名请求。

这不是假设，而是很多团队在做第一次安全审计时会发现的真实状况。Kubernetes 的默认配置追求"能跑起来"，而不是"足够安全"。本文围绕三个核心工具——kube-bench、Trivy、Falco——讲清楚每个工具的工作原理，并结合 Pod 安全标准（PSA）和 RBAC 最小权限，构建一套完整的纵深防御体系。

---

## 一、Kubernetes 安全威胁面

在谈工具之前，先建立一个清晰的威胁模型。Kubernetes 的安全问题可以用四层模型来描述：

```
┌────────────────────────────────────────────────────┐
│  Layer 4: 应用代码                                  │
│  SQL 注入、XSS、业务逻辑漏洞                        │
├────────────────────────────────────────────────────┤
│  Layer 3: 容器与镜像                                │
│  CVE 漏洞、恶意镜像、敏感信息硬编码                 │
├────────────────────────────────────────────────────┤
│  Layer 2: 集群组件                                  │
│  API Server 配置、RBAC 权限、NetworkPolicy          │
├────────────────────────────────────────────────────┤
│  Layer 1: 云基础设施                                │
│  节点 SSH 访问、etcd 暴露、IAM 权限                 │
└────────────────────────────────────────────────────┘
```

攻击者通常从"最软的那层"入手。最常见的攻击路径是：

1. **初始入侵**：利用应用漏洞（RCE）或镜像中的已知 CVE 获得容器内的 Shell
2. **容器逃逸**：如果容器以 `privileged` 运行，或挂载了宿主机的 `/`，攻击者可以直接访问宿主机文件系统
3. **提权与横向移动**：利用宽松的 RBAC 权限，通过 ServiceAccount Token 调用 API Server，在其他命名空间创建特权 Pod

三个工具分别守住不同的层：kube-bench 关注 Layer 2（集群组件配置），Trivy 关注 Layer 3（容器镜像），Falco 在运行时监控 Layer 3 和 Layer 4 的异常行为。

### CIS Benchmark 是什么

CIS（Center for Internet Security）是一个非营利组织，发布各类系统的安全配置基准，称为 CIS Benchmark。Kubernetes 的 CIS Benchmark 文档列出了数百条具体的检查项，每条都明确说明：这个配置为什么危险、应该怎么设、如何验证。kube-bench 是 Aqua Security 开源的工具，将这套检查逻辑自动化——一次扫描就能知道集群哪里不合规。

---

## 二、kube-bench：集群安全基线检查

### 检查项的分类结构

kube-bench 按照 CIS Benchmark 的章节组织检查项，主要分为五个大类：

| 大类 | 检查内容 |
|------|----------|
| Master Node | API Server、etcd、Controller Manager、Scheduler 的启动参数和文件权限 |
| Worker Node | kubelet 的认证配置、配置文件的属主和权限 |
| ETCD | etcd 的 TLS 配置、访问控制、数据加密 |
| Control Plane | 审计日志、准入控制器、加密配置 |
| Policies | Pod 安全策略、网络策略、镜像仓库配置 |

每个检查项有三种结果：**PASS**（符合基线）、**FAIL**（不符合，存在风险）、**WARN**（需要人工判断，工具无法自动验证）。

### 以 Pod 方式运行 kube-bench

kube-bench 以 Pod 方式运行最为方便，不需要在节点上安装额外软件：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kube-bench
  namespace: default
spec:
  hostPID: true
  containers:
    - name: kube-bench
      image: aquasec/kube-bench:latest
      command: ["kube-bench"]
      args: ["--json", "--outputfile", "/output/results.json"]
      volumeMounts:
        - name: var-lib-etcd
          mountPath: /var/lib/etcd
          readOnly: true
        - name: etc-systemd
          mountPath: /etc/systemd
          readOnly: true
        - name: etc-kubernetes
          mountPath: /etc/kubernetes
          readOnly: true
        - name: usr-bin
          mountPath: /usr/local/mount-from-host/bin
          readOnly: true
        - name: output
          mountPath: /output
  restartPolicy: Never
  volumes:
    - name: var-lib-etcd
      hostPath:
        path: /var/lib/etcd
    - name: etc-systemd
      hostPath:
        path: /etc/systemd
    - name: etc-kubernetes
      hostPath:
        path: /etc/kubernetes
    - name: usr-bin
      hostPath:
        path: /usr/bin
    - name: output
      emptyDir: {}
```

运行完成后，通过 `kubectl logs kube-bench` 查看结果，JSON 格式输出可导入 Grafana 或 Elasticsearch 做可视化。

### 高危 FAIL 项修复示例

**1. API Server 匿名访问未关闭**

```
FAIL 1.2.1 确保 API Server 的 --anonymous-auth 参数设为 false
```

API Server 默认允许匿名请求（以 `system:anonymous` 身份），意味着集群内任何 Pod 都可以发送未认证请求。修复方式：

```yaml
# /etc/kubernetes/manifests/kube-apiserver.yaml
spec:
  containers:
  - command:
    - kube-apiserver
    - --anonymous-auth=false
```

:::warning
修改 kube-apiserver.yaml 后，Static Pod 会自动重启。生产环境操作前务必确认所有 Ingress Controller、Prometheus、CI 工具都已配置了正确的认证，否则会导致这些组件立即失联。
:::

**2. etcd 的 Secret 数据未加密**

```
FAIL 1.2.33 确保 etcd 中的 Secret 数据使用静态加密（Encryption at Rest）
```

Kubernetes 的 Secret 默认以 Base64 编码存储在 etcd 中，任何能访问 etcd 的人都可以直接读取内容。需要配置 `EncryptionConfiguration`：

```yaml
# /etc/kubernetes/encryption-config.yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
      - secrets
    providers:
      - aescbc:
          keys:
            - name: key1
              # 使用 head -c 32 /dev/urandom | base64 生成
              secret: <base64-encoded-32-byte-key>
      - identity: {}   # 兜底：允许读取未加密的旧数据
```

在 API Server 启动参数中引用该文件，配置生效后需执行以下命令触发已有 Secret 的重写：

```bash
kubectl get secrets --all-namespaces -o json | kubectl replace -f -
```

**3. kubelet 未启用认证**

```
FAIL 4.2.1 确保 kubelet 的 --anonymous-auth 参数设为 false
```

kubelet 在 10250 端口提供 API，若不要求认证，任何能访问该端口的人都可以在节点上执行命令：

```yaml
# /var/lib/kubelet/config.yaml
authentication:
  anonymous:
    enabled: false
  webhook:
    enabled: true       # 委托给 API Server 验证
authorization:
  mode: Webhook
```

:::tip
kube-bench 建议在集群部署时运行一次作为基线，之后通过 CronJob 每周定期扫描，避免因配置变更引入安全回退而无人察觉。
:::

---

## 三、Trivy：镜像漏洞扫描

### 扫描机制与能力

一个容器镜像通常包含三层依赖：基础镜像的 OS 软件包、语言运行时依赖、应用自身代码。每一层都可能携带已知漏洞（CVE）。

Trivy 的扫描原理分两步：首先从镜像中提取文件系统，识别所有已安装的软件包及版本；然后与其维护的漏洞数据库（来源包括 NVD、Red Hat、Debian Security、GitHub Advisory 等）做交叉比对，找出匹配的 CVE 并给出严重程度（CRITICAL、HIGH、MEDIUM、LOW）和修复版本。

除了 CVE 漏洞，Trivy 还能扫描：
- **配置错误**：Dockerfile 中的不安全写法（如 `USER root`）
- **Secret 泄露**：镜像层中硬编码的 API Key、密码、Token
- **SBOM（软件物料清单）**：导出镜像中所有软件包的完整列表，用于供应链安全审计

### 常用扫描命令

```bash
# 扫描公共镜像，默认表格输出
trivy image nginx:1.25

# 只显示高危和严重漏洞，过滤无修复版本的 CVE
trivy image --severity HIGH,CRITICAL --ignore-unfixed myapp:1.0

# 同时扫描 CVE 和配置错误
trivy image --scanners vuln,config myapp:1.0

# 扫描本地代码仓库（含 Secret 检测）
trivy fs --scanners vuln,config,secret .
```

### GitLab CI 集成：扫描失败阻断构建

下面是将 Trivy 插入构建流水线、在镜像推送前强制卡口的完整配置：

```yaml
# .gitlab-ci.yml
stages:
  - build
  - scan
  - push
  - deploy

variables:
  IMAGE_NAME: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA

build:
  stage: build
  script:
    - docker build -t $IMAGE_NAME .
    - docker save $IMAGE_NAME -o image.tar
  artifacts:
    paths:
      - image.tar

trivy-scan:
  stage: scan
  image:
    name: aquasec/trivy:latest
    entrypoint: [""]
  script:
    - trivy image --download-db-only
    # CRITICAL 或 HIGH 漏洞时退出码为 1，阻断流水线
    - trivy image
        --exit-code 1
        --severity CRITICAL,HIGH
        --ignore-unfixed
        --format json
        --output gl-container-scanning-report.json
        --input image.tar
  allow_failure: false
  artifacts:
    when: always
    reports:
      container_scanning: gl-container-scanning-report.json
    paths:
      - gl-container-scanning-report.json

push:
  stage: push
  script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
    - docker push $IMAGE_NAME
  needs: ["trivy-scan"]   # 只有 scan 通过才执行 push
```

这个流程的核心逻辑是：**先扫描，再推送**。未通过安全扫描的镜像永远不会进入镜像仓库，也就不可能被部署到集群。

:::danger
避免在 CI 中使用 `--exit-code 0`——这等于扫描了但不报错，形同虚设。安全卡口的意义在于真正阻断有问题的产物流入生产环境。
:::

### 镜像安全最佳实践

**使用精简基础镜像**：`alpine`（约 5MB）、`distroless`（连 Shell 都没有）比 `ubuntu`、`debian` 攻击面小得多。没有 bash 就无法执行交互式命令，distroless 镜像天然抵抗一类攻击。

**固定镜像版本**：`nginx:latest` 今天扫描可能干净，明天基础镜像更新引入新漏洞，你却无从知晓。使用 `nginx:1.25.4` 配合 Renovate/Dependabot 自动追踪更新。

**多阶段构建**：构建工具只在 builder 阶段存在，最终镜像只包含运行时必需的文件：

```dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o server .

# 最终镜像只包含编译产物，没有 Go 工具链
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/server /server
ENTRYPOINT ["/server"]
```

---

## 四、Falco：运行时威胁检测

### eBPF 内核探针工作原理

kube-bench 和 Trivy 属于静态检查——在运行前发现问题。Falco 是动态检测——在运行时发现异常行为。

Falco 的核心机制是系统调用（syscall）监控。Linux 上所有的文件访问、网络连接、进程创建，本质上都是系统调用。Falco 通过 **eBPF 内核探针**（在内核中插入一个小程序，拦截系统调用）实时捕获每一个系统调用，再由用户态的规则引擎判断是否异常：

```
┌─────────────────────────────────────────────────────┐
│                    容器/宿主机                        │
│  进程执行 → 文件访问 → 网络连接 → 系统调用            │
└───────────────────┬─────────────────────────────────┘
                    │ 系统调用
                    ▼
┌─────────────────────────────────────────────────────┐
│              eBPF 探针（内核态）                      │
│  拦截系统调用 → 提取上下文（进程名/用户/容器信息）   │
└───────────────────┬─────────────────────────────────┘
                    │ 事件流
                    ▼
┌─────────────────────────────────────────────────────┐
│              Falco 规则引擎（用户态）                 │
│  匹配规则 condition → 生成告警 output                │
└───────────────────┬─────────────────────────────────┘
                    │ 告警
                    ▼
        Slack / AlertManager / Elasticsearch
```

每条规则由三部分组成：`condition`（触发条件，支持对系统调用参数、进程信息、容器信息的复杂过滤）、`output`（告警输出模板）、`priority`（严重程度）。

eBPF 方式相比内核模块的优势：不需要编译内核模块，内核升级后无需重建，且经过内核的 BPF 验证器安全验证，更稳定。从 Falco 0.34 开始，eBPF 成为推荐方案。

### 内置规则示例

| 规则名 | 触发条件 | 对应攻击场景 |
|--------|----------|-------------|
| Terminal shell in container | 容器内启动了交互式 Shell | 攻击者通过 RCE 获得 bash |
| Read sensitive file trusted after startup | 读取 /etc/shadow、/etc/passwd | 密码爆破前的信息收集 |
| Write below etc | 写入 /etc 目录 | 篡改系统配置实现持久化 |
| Launch Privileged Container | 启动特权容器 | 容器逃逸准备 |
| Contact K8s API Server From Container | 容器内直接访问 API Server | 利用 ServiceAccount Token 提权 |

### 自定义规则示例：检测 kubectl exec

`kubectl exec` 是合法的运维操作，但也是攻击者获得容器 Shell 的常见手段。下面在生产 Namespace 中检测到 `kubectl exec` 时立即告警：

```yaml
# falco-custom-rules.yaml
- rule: Kubectl Exec into Production Container
  desc: |
    检测在 production 命名空间中执行 kubectl exec 的行为。
    kubectl exec 会在目标容器内启动新进程，
    Falco 通过检查进程的父进程链来识别这类操作。
  condition: >
    spawned_process
    and container
    and k8s.ns.name = "production"
    and proc.name in (shell_binaries)
    and proc.pname in (runc, containerd-shim, crio)
  output: >
    Shell spawned in production container via kubectl exec
    (user=%user.name pod=%k8s.pod.name ns=%k8s.ns.name
     container=%container.name shell=%proc.name
     parent=%proc.pname cmdline=%proc.cmdline)
  priority: WARNING
  tags: [shell, k8s, production]

- rule: Sensitive Mount in Container
  desc: 检测容器挂载了宿主机的敏感目录
  condition: >
    container
    and evt.type = mount
    and fd.name in (/etc, /var/run/docker.sock, /var/lib/kubelet, /)
  output: >
    Sensitive host path mounted in container
    (image=%container.image.repository pod=%k8s.pod.name
     mount_path=%fd.name user=%user.name)
  priority: CRITICAL
  tags: [container, mount, privilege-escalation]
```

:::tip
Falco 的规则语言（SYSDIG filter）表达力丰富，`k8s.ns.name`、`k8s.pod.name`、`container.image.repository` 这些字段可以直接在 condition 中使用，无需额外配置。
:::

### Falco DaemonSet 部署与 Sidekick 告警路由

Falco 以 DaemonSet 运行，每个节点一个实例（参见 [DaemonSet 部署采集器注意事项](./Kubernetes中DaemonSet部署采集器注意事项.md)）。通过 Helm 部署是最简便的方式：

```bash
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm repo update

helm install falco falcosecurity/falco \
  --namespace falco \
  --create-namespace \
  --set driver.kind=ebpf \
  --set falcosidekick.enabled=true \
  --set falcosidekick.config.slack.webhookurl="https://hooks.slack.com/..." \
  --set falcosidekick.config.alertmanager.hostport="http://alertmanager:9093" \
  -f falco-custom-rules.yaml
```

**Falco Sidekick** 是 Falco 的告警转发组件，支持将告警发送到 50+ 个目标（Slack、PagerDuty、AlertManager、Elasticsearch、Loki 等）。推荐同时将告警接入 AlertManager（严重告警分级升级）和 Loki（安全事件历史审计）。

---

## 五、Pod 安全标准（PSA）

### PodSecurityPolicy 的废弃与 PSA 的接替

Kubernetes 1.25 正式移除了 PodSecurityPolicy（PSP）。PSP 的问题在于它通过 RBAC 授权绑定，规则分散、难以审计，而且配置错误可能意外地给所有 Pod 赋予更高权限。

Pod Security Admission（PSA）是接替方案，以 Namespace 为粒度，通过三个预定义的安全级别直接控制 Pod 的许可：

| 安全级别 | 适用场景 | 限制范围 |
|----------|----------|----------|
| **Privileged** | 系统组件（kube-system） | 无限制，允许所有特权操作 |
| **Baseline** | 普通业务应用 | 禁止明显的高危配置（特权容器、hostNetwork、hostPID） |
| **Restricted** | 安全敏感应用 | 在 Baseline 基础上强制 runAsNonRoot、只读根文件系统等 |

### Namespace 级别 PSA 配置

PSA 通过在 Namespace 上添加 Label 来配置，每种模式有三种执行方式：`enforce`（拒绝违规 Pod）、`audit`（记录审计日志但不拒绝）、`warn`（向用户返回警告但不拒绝）：

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    # 强制拒绝不符合 restricted 级别的 Pod
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/enforce-version: latest
    # 记录审计日志，方便监控
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/audit-version: latest
    # 向用户返回警告
    pod-security.kubernetes.io/warn: restricted
    pod-security.kubernetes.io/warn-version: latest
```

:::tip
推荐在新 Namespace 直接使用 `enforce: restricted`。对于已有的 Namespace，先用 `warn` 和 `audit` 模式运行一段时间，观察有哪些 Pod 会违规，修复后再切换到 `enforce`，避免突然拒绝导致业务中断。
:::

### 常见违规修复

| 违规项 | 问题根源 | 修复方式 |
|--------|----------|----------|
| `runAsRoot` | 未指定非 root 用户 | 在 securityContext 中设置 `runAsNonRoot: true` 和 `runAsUser: 1000` |
| `hostNetwork: true` | 使用宿主机网络 | 去掉该字段，使用 Service 暴露端口 |
| `privileged: true` | 容器以特权模式运行 | 去掉该字段，仅申请所需的 Linux Capabilities |
| 可写根文件系统 | 未设置只读根文件系统 | 添加 `readOnlyRootFilesystem: true`，通过 emptyDir 挂载可写目录 |

---

## 六、RBAC 最小权限

### ServiceAccount 最佳实践

每个 Pod 默认挂载当前命名空间的 default ServiceAccount Token，且该 Token 默认自动挂载（`automountServiceAccountToken: true`）。如果攻击者获得了容器内 Shell，就能直接使用这个 Token 与 API Server 通信。

最小权限原则要求：**每个工作负载只拥有完成其职责所需的最小权限，不多一分**。

:::danger
绝对不要给业务 Pod 绑定 ClusterAdmin 角色，也不要为了"方便"给所有 ServiceAccount 赋予读取所有资源的权限。这是攻击者在获得容器 Shell 后进行横向移动的最大助力。
:::

### 最小权限实践示例

以一个只需要读取同命名空间 ConfigMap 的应用为例：

```yaml
# 1. 创建专用 ServiceAccount，明确禁止自动挂载 Token
apiVersion: v1
kind: ServiceAccount
metadata:
  name: config-reader
  namespace: production
automountServiceAccountToken: false  # 默认禁止，按需开启

---
# 2. 只授予业务实际需要的权限
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: configmap-reader
  namespace: production
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list"]
    # 进一步限制只能访问特定资源名称
    resourceNames: ["app-config"]

---
# 3. 绑定 Role 到 ServiceAccount
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: config-reader-binding
  namespace: production
subjects:
  - kind: ServiceAccount
    name: config-reader
    namespace: production
roleRef:
  kind: Role
  name: configmap-reader
  apiGroup: rbac.authorization.k8s.io
```

在 Pod 中显式指定该 ServiceAccount，并只在需要时才开启 Token 挂载：

```yaml
spec:
  serviceAccountName: config-reader
  automountServiceAccountToken: true   # 此处明确声明需要
```

关于 RBAC 的完整设计模式，可参阅 [RBAC 权限控制](./RBAC权限控制.md)。

---

## 七、安全扫描 CI/CD 完整流程

将三个工具串联起来，构成覆盖从代码提交到运行时的完整防线：

```
代码提交
    │
    ▼
镜像构建（docker build）
    │
    ▼
Trivy 扫描（CVE + 配置错误 + Secret 泄露）
    │  FAIL → 阻断流水线，通知开发者修复
    ▼
推送到镜像仓库
    │
    ▼
kubectl apply（部署到集群）
    │  PSA 准入控制：拒绝违规 Pod
    │  RBAC：最小化 ServiceAccount 权限
    ▼
kube-bench 定期扫描集群配置（CronJob，每周一次）
    │  FAIL → 工单 / 告警通知运维
    ▼
Falco 运行时监控（持续运行，DaemonSet）
    │  异常 → Sidekick → Slack / AlertManager
    ▼
安全事件响应
```

这四层叠加，才是真正意义上的纵深防御（Defense in Depth）：

```
准入控制（PSA）   →  阻止高危 Pod 创建
RBAC             →  限制已运行 Pod 的 API 权限
NetworkPolicy    →  限制已运行 Pod 的网络访问
Falco            →  检测以上限制被绕过时的运行时异常
kube-bench       →  定期审计集群配置是否回退
Trivy            →  在构建阶段拦截漏洞镜像进入仓库
```

---

## 小结

- **kube-bench** 自动化执行 CIS Kubernetes Benchmark 检查，重点修复 API Server 匿名访问、etcd 静态加密、kubelet 认证三个高危 FAIL 项
- **Trivy** 扫描镜像的 CVE、配置错误和 Secret 泄露，通过 `--exit-code 1` 集成到 GitLab CI，在镜像推送前强制卡口；配合 distroless/alpine 基础镜像减少攻击面
- **Falco** 通过 eBPF 内核探针实时监控系统调用，用规则引擎检测容器内异常行为；通过 Sidekick 将告警路由到 AlertManager 或 Slack
- **PSA** 在准入控制层通过 Namespace Label 强制 Restricted/Baseline 安全级别，阻止高危 Pod 创建
- **RBAC 最小权限** 通过专用 ServiceAccount + 细粒度 Role 限制 Pod 的 API 访问范围，禁止自动挂载 Token
- 六个机制构成纵深防御体系，覆盖构建时、准入时、运行时三个阶段

---

## 常见问题

### Q1：kube-bench 的 WARN 项和 FAIL 项有什么区别？应该优先处理哪个？

FAIL 项表示 kube-bench 能自动检测到不符合 CIS Benchmark 的配置，有明确的修复动作。WARN 项表示该检查需要人工判断，通常是因为"合规"的标准依赖具体环境（例如"审计日志保留时长需符合合规要求"，合规要求可能是 30 天或 90 天，工具无法自动判断）。

优先处理 FAIL 项，尤其是 Level 1 的 FAIL 项，修复成本低但风险高。WARN 项需要对照实际合规需求逐条过一遍，不要因为不自动报错就忽略——有些 WARN 项（如审计日志配置）在安全事件发生后会让排查极为困难。

### Q2：Trivy 扫描发现大量无修复版本的 MEDIUM 漏洞，应该怎么处理？

处理思路分三层。第一层是评估实际可利用性（exploitability）：CVSS 评分高不等于实际风险高，很多漏洞的利用条件极为苛刻（需要本地访问、特定网络配置等）。可以用 Trivy 的 VEX（Vulnerability Exploitability eXchange）文件标记已评估为不可利用的 CVE，避免重复告警。

第二层是切换精简基础镜像：大量 MEDIUM CVE 往往来自 `debian/ubuntu` 预装工具（如 `perl`、`gcc`），切换到 Alpine 或 distroless 可以直接消除这类噪音。

第三层是建立例外清单（Allowlist）：对已评估且接受风险的 CVE，在 Trivy 配置中明确豁免并附上原因和评估日期，而不是用 `--exit-code 0` 关掉整个卡口。

### Q3：Falco 在生产环境告警量很大，如何减少噪音而不错过真实威胁？

Falco 告警噪音主要来自两个来源：规则粒度太粗（正常操作触发了告警），以及合法的运维行为（如发布期间的 `kubectl exec`）。

精细化规则条件：在告警条件中加入白名单，例如内置的 `Terminal shell in container` 规则可修改为排除特定 Namespace 或镜像：

```yaml
- rule: Terminal shell in container
  condition: >
    spawned_process and container and shell_procs
    and not k8s.ns.name in (kube-system, monitoring)
    and not container.image.repository in (known-admin-tools)
```

按严重程度分级路由：CRITICAL 级告警直接触发 PagerDuty，WARNING 级进 Slack 频道，INFO 级仅写入日志。不要把所有告警发到同一个渠道，否则高优先级告警会被淹没。

### Q4：在已有生产集群上推行 PSA Restricted 级别，如何平滑迁移？

不要直接切 `enforce: restricted`，而是分三个阶段推进。第一阶段先给 Namespace 加上 `warn` 和 `audit` 模式，此时 Pod 仍然可以创建，但违规的 Pod 会在 `kubectl apply` 时输出警告，同时被写入审计日志。

第二阶段根据 `audit` 日志列出所有违规 Pod，按类型分类修复（runAsRoot、hostNetwork、privileged 等），可以参考前文的修复示例逐一处理。对于无法修复的系统组件（如 node-local-dns、某些 CNI 插件），保留在 `privileged` 级别的 Namespace 中。

第三阶段确认所有违规修复完毕后，将 `warn` 改为 `enforce`。建议先在非生产环境验证完整流程，再推广到生产。整个迁移过程中使用 `enforce-version: latest` 的 beta 版本可能会导致未预期的拒绝，优先使用具体的版本号如 `v1.29`。

### Q5：Falco 的 eBPF 模式和内核模块模式如何选择？

| 对比维度 | eBPF 模式 | 内核模块模式 |
|----------|-----------|-------------|
| 稳定性 | 高（BPF 验证器保证安全） | 中（模块 bug 可能引发 panic） |
| 兼容性 | 需要内核 4.14+（5.8+ 获得最佳支持） | 需要与内核版本匹配的模块 |
| 维护成本 | 低（无需重新编译） | 高（内核升级后需重新构建） |
| 推荐场景 | 内核 >= 5.8 的现代集群 | 旧版内核无法使用 eBPF 时的回退方案 |

云厂商托管 Kubernetes（EKS、GKE、AKS）的节点内核版本通常在 5.10 以上，优先使用 eBPF 模式。自建集群如果使用 CentOS 7（内核 3.10）则需要用内核模块模式，但强烈建议升级操作系统——内核 3.10 本身已有大量未修复的 CVE，使用 Falco 保护一个存在已知内核漏洞的节点，效果有限。

## 参考资源

- [CIS Kubernetes Benchmark 官方文档](https://www.cisecurity.org/benchmark/kubernetes)
- [kube-bench GitHub 仓库](https://github.com/aquasecurity/kube-bench)
- [Falco 官方文档](https://falco.org/docs/)
- [Trivy 官方文档](https://aquasecurity.github.io/trivy/)

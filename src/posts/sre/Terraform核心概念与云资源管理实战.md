---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - SRE
tag:
  - Terraform
  - IaC
  - 基础设施即代码
  - 云原生
---

# Terraform 核心概念与云资源管理实战

## 为什么需要基础设施即代码

想象这样一个场景：你的团队需要在生产环境新增一套 Redis 集群，于是有人登录云控制台，手动点击创建实例、配置安全组、绑定 VPC。三个月后，另一位工程师需要搭建同样规格的测试环境，却不知道当初选了哪个可用区、用了哪个子网。于是历史重演，又一轮手动点击开始。

这种工作方式存在三个本质缺陷：

- **不可复现**：操作步骤存在人脑里，换个人就换一个结果
- **不可审计**：没有变更记录，出问题后无法追溯是谁在什么时候改了什么
- **不可回滚**：误删了一条安全组规则，你不知道原来的规则是什么

基础设施即代码（Infrastructure as Code，IaC）的核心价值正是解决这三个问题。它的思路是：**用代码描述基础设施的期望状态，通过版本控制系统管理这份描述，通过工具将描述转化为真实资源**。

IaC 有两种主流实现哲学：

- **声明式（Declarative）**：告诉工具"我要什么"，由工具负责如何达到这个状态。Terraform 是典型代表。
- **命令式（Imperative）**：告诉工具"依次执行哪些步骤"。Ansible 的 Playbook 更接近这种方式。

这两种哲学分别适合不同的场景，这也是 Terraform 和 Ansible 经常被一起使用的原因——Terraform 负责把虚拟机、网络、存储等资源创建出来，Ansible 再负责在虚拟机上安装软件、写配置文件。两者是互补关系，而非竞争关系。

---

## Terraform 核心工作原理

### 声明式模型：描述期望状态

Terraform 使用 HashiCorp Configuration Language（HCL）来描述资源的期望状态。你不需要告诉它"先创建 VPC，再创建子网，再创建安全组"，你只需要声明这三个资源应该存在、分别具备哪些属性，Terraform 会自动分析它们之间的依赖关系，按正确的顺序执行。

这带来了两个好处：一是代码更具可读性，一眼就能看出基础设施的全貌；二是变更时只需修改声明，Terraform 自动计算出需要做什么。

### 状态文件（tfstate）：连接代码与现实的桥梁

Terraform 有一个核心机制：它在本地（或远程）维护一份 `terraform.tfstate` 文件，记录它上次管理的所有资源的实际状态。这份文件是 Terraform 最重要的数据，其作用是：

```
代码（期望状态）  ←→  tfstate（上次已知状态）  ←→  云平台 API（真实状态）
```

每次执行 `terraform plan`，Terraform 会做两件事：
1. 从云平台 API 获取当前真实状态
2. 与 tfstate 和代码进行对比，计算出需要做哪些变更

:::warning 状态文件的重要性
`terraform.tfstate` 包含了所有资源的 ID、属性，甚至可能包含密码等敏感信息。不要将其提交到 Git 仓库，也不要随意删除。一旦状态文件损坏或丢失，Terraform 将无法感知已有资源，可能导致重复创建或无法管理已有资源。
:::

### Provider：与云平台对话的插件

Terraform 本身不直接调用 AWS 或阿里云的 API，它通过 **Provider** 插件完成这个工作。Provider 是 Terraform 与外部系统交互的适配器，每个 Provider 封装了对应平台的 API 调用逻辑。

```
Terraform Core  →  Provider (AWS/AliCloud/Kubernetes)  →  云平台 API
```

Provider 在 `required_providers` 块中声明，执行 `terraform init` 时自动下载对应版本的 Provider 插件。常用 Provider 包括：

- `hashicorp/aws`：管理 AWS 资源
- `aliyun/alicloud`：管理阿里云资源
- `hashicorp/kubernetes`：管理 Kubernetes 资源
- `hashicorp/helm`：管理 Helm Chart

### 执行流程：init → plan → apply → destroy

Terraform 的核心工作流分为四个阶段：

```
terraform init
    ↓ 下载 Provider 插件，初始化工作目录
terraform plan
    ↓ 对比代码、tfstate 和真实状态，输出变更计划（只读，不执行）
terraform apply
    ↓ 执行变更计划，将真实资源调整为代码描述的期望状态
terraform destroy
    ↓ 销毁所有由 Terraform 管理的资源
```

`plan` 阶段是 Terraform 安全性的关键所在。它只读取状态，不做任何修改，输出的是一份类似 `git diff` 的变更预览。工程师可以在执行前仔细审查将要发生哪些变更，这个步骤在 CI/CD 流水线中尤为重要——可以作为 Merge Request 的评审内容。

:::tip Plan 的幂等性
多次执行 `terraform plan`，只要代码和真实资源没有变化，输出结果总是相同的。这保证了操作的可预测性。
:::

---

## HCL 语法核心

HCL 语法围绕几个核心构建块展开，理解它们的职责是读懂任何 Terraform 代码的基础。

### 核心构建块职责

| 构建块 | 作用 | 类比 |
|--------|------|------|
| `resource` | 声明一个需要被管理的真实资源 | 定义一个对象实例 |
| `data` | 读取已有资源的信息（只读） | 查询数据库 |
| `variable` | 声明输入变量，使代码可参数化 | 函数参数 |
| `output` | 声明输出值，供外部引用或展示 | 函数返回值 |
| `local` | 定义本地计算值，减少重复 | 局部变量 |

### 资源引用语法

在 HCL 中，资源之间的引用使用 `<resource_type>.<resource_name>.<attribute>` 语法。Terraform 通过分析引用关系自动构建资源依赖图，确保被依赖的资源先于依赖方创建。

```hcl
# 引用 vpc 资源的 id 属性
resource "alicloud_vswitch" "main" {
  vpc_id     = alicloud_vpc.main.id   # 引用 vpc 资源的 id
  cidr_block = "172.16.0.0/24"
  zone_id    = "cn-hangzhou-h"
}
```

### 完整示例：创建 VPC + ECS 实例

```hcl
# provider 配置
terraform {
  required_providers {
    alicloud = {
      source  = "aliyun/alicloud"
      version = "~> 1.200"
    }
  }
}

provider "alicloud" {
  region = var.region
}

# 输入变量
variable "region" {
  type        = string
  description = "部署地域"
  default     = "cn-hangzhou"
}

variable "instance_type" {
  type    = string
  default = "ecs.c6.large"
}

variable "instance_count" {
  type    = number
  default = 2
}

# 本地变量
locals {
  project_name = "myapp"
  common_tags = {
    Project     = local.project_name
    Environment = "production"
    ManagedBy   = "terraform"
  }
}

# 创建 VPC
resource "alicloud_vpc" "main" {
  vpc_name   = "${local.project_name}-vpc"
  cidr_block = "172.16.0.0/16"
  tags       = local.common_tags
}

# 创建交换机（子网）
resource "alicloud_vswitch" "main" {
  vpc_id     = alicloud_vpc.main.id
  cidr_block = "172.16.0.0/24"
  zone_id    = "cn-hangzhou-h"
  tags       = local.common_tags
}

# 创建安全组
resource "alicloud_security_group" "main" {
  name   = "${local.project_name}-sg"
  vpc_id = alicloud_vpc.main.id
  tags   = local.common_tags
}

# 安全组规则：允许 SSH
resource "alicloud_security_group_rule" "allow_ssh" {
  type              = "ingress"
  ip_protocol       = "tcp"
  nic_type          = "intranet"
  policy            = "accept"
  port_range        = "22/22"
  priority          = 1
  security_group_id = alicloud_security_group.main.id
  cidr_ip           = "0.0.0.0/0"
}

# 查询最新镜像（data source，只读）
data "alicloud_images" "ubuntu" {
  name_regex  = "^ubuntu_22_04"
  most_recent = true
  owners      = "system"
}

# 使用 for_each 创建多台 ECS 实例
resource "alicloud_instance" "web" {
  for_each = toset([for i in range(var.instance_count) : tostring(i)])

  instance_name        = "${local.project_name}-web-${each.key}"
  image_id             = data.alicloud_images.ubuntu.images[0].id
  instance_type        = var.instance_type
  vswitch_id           = alicloud_vswitch.main.id
  security_groups      = [alicloud_security_group.main.id]
  internet_max_bandwith_out = 0

  tags = merge(local.common_tags, {
    Role = "web"
  })
}

# 输出实例 IP
output "instance_private_ips" {
  description = "所有 Web 实例的私网 IP"
  value       = { for k, v in alicloud_instance.web : k => v.private_ip }
}

output "vpc_id" {
  description = "VPC ID"
  value       = alicloud_vpc.main.id
}
```

`for_each` 与 `count` 是 Terraform 管理同类资源的两种方式。推荐使用 `for_each`，因为它用具名键（而非数字下标）标识资源，删除某个实例时不会影响其他实例的状态。

---

## 状态管理（State Management）

### 本地状态 vs 远程状态

默认情况下，tfstate 存储在执行目录的本地文件中。这在个人开发时可以接受，但团队协作场景下会产生严重问题：

- 本地状态无法共享，团队成员各自持有一份可能不同步的状态文件
- 本地文件没有并发保护，两个人同时执行 apply 会导致状态混乱
- 本地文件容易丢失

**远程后端（Remote Backend）**解决了上述问题。以 AWS S3 + DynamoDB 为例：

```hcl
# backend.tf
terraform {
  backend "s3" {
    bucket         = "mycompany-terraform-state"
    key            = "production/main.tfstate"
    region         = "ap-southeast-1"
    encrypt        = true                        # 启用服务端加密

    # DynamoDB 表用于状态锁
    dynamodb_table = "terraform-state-lock"
  }
}
```

对应需要提前创建的 DynamoDB 表（用于状态锁），其主键固定为字符串类型的 `LockID`。

阿里云场景下，可使用 OSS + TableStore 实现相同效果：

```hcl
terraform {
  backend "oss" {
    bucket              = "mycompany-terraform-state"
    key                 = "production/main.tfstate"
    region              = "cn-hangzhou"
    encrypt             = true
    tablestore_endpoint = "https://terraform-lock.cn-hangzhou.ots.aliyuncs.com"
    tablestore_table    = "terraform_lock"
  }
}
```

### 状态锁（State Locking）

远程后端通常提供**状态锁**机制。当任何操作（plan、apply、destroy）开始时，Terraform 会向后端申请一把锁；操作结束后释放锁。若锁被占用（另一个操作正在进行），后续操作会等待或报错，从而防止并发执行导致的状态冲突。

:::danger 状态锁死
极少数情况下（如进程被强制终止），锁可能无法自动释放，导致后续所有操作都被阻塞并报错"Error acquiring the state lock"。此时可以通过以下命令强制解锁，但**必须确认没有其他操作正在进行**：

```bash
terraform force-unlock <LOCK_ID>
```

强制解锁后需仔细检查状态文件的完整性。
:::

### State 漂移（Drift）

真实资源与 tfstate 记录的状态不一致，称为**状态漂移**。常见触发原因：

- 有人手动在控制台修改了资源
- 云平台自动变更了某些属性（如证书续期）
- 执行了带外操作（在 Terraform 管理范围外创建/删除了资源）

处理漂移有两种思路：

```bash
# 方案一：terraform refresh（已废弃，不推荐）
# 将真实状态同步回 tfstate，但不变更真实资源
# Terraform 1.x 已将此功能合并入 plan/apply 的 -refresh-only 参数

terraform plan -refresh-only     # 查看漂移情况
terraform apply -refresh-only    # 将漂移更新到 tfstate

# 方案二：terraform import
# 将控制台手动创建的已有资源纳入 Terraform 管理
terraform import alicloud_vpc.main vpc-bp1234567890
```

### 敏感数据保护

tfstate 文件可能包含数据库密码、API 密钥等敏感信息。最佳实践是：

1. 启用远程后端的服务端加密（`encrypt = true`）
2. 对 S3/OSS Bucket 配置严格的访问策略，仅 CI/CD 服务账号可读写
3. 对于极度敏感的输出，在 `output` 块中标记 `sensitive = true`

```hcl
output "db_password" {
  value     = random_password.db.result
  sensitive = true   # 执行 apply 时不会在终端打印出来
}
```

---

## 多环境管理

管理 dev、staging、production 多套环境是 Terraform 最常见的工程难题。主流有三种方案：

### 方案一：Workspace（工作空间）

Workspace 是 Terraform 内置的轻量级多环境机制，同一套代码可以对应多个独立的 tfstate。

```bash
terraform workspace new dev
terraform workspace new prod
terraform workspace select dev
terraform apply   # 使用 dev workspace 的 tfstate
```

在代码中可以通过 `terraform.workspace` 变量感知当前环境：

```hcl
resource "alicloud_instance" "web" {
  instance_type = terraform.workspace == "prod" ? "ecs.c6.xlarge" : "ecs.c6.large"
}
```

### 方案二：目录分离（推荐）

将不同环境的配置放在独立目录下，通过公共模块复用基础设施定义：

```
infrastructure/
├── modules/
│   ├── vpc/          # 可复用的 VPC 模块
│   └── ecs-cluster/  # 可复用的 ECS 集群模块
├── environments/
│   ├── dev/
│   │   ├── main.tf   # 调用模块，传入 dev 参数
│   │   ├── backend.tf
│   │   └── terraform.tfvars
│   └── prod/
│       ├── main.tf   # 调用同一模块，传入 prod 参数
│       ├── backend.tf
│       └── terraform.tfvars
```

### 方案三：Terragrunt（进阶）

Terragrunt 是 Terraform 的封装工具，专门解决多环境下的代码重复问题，支持跨模块的依赖管理和远程 backend 配置继承。适合管理超过 10 个环境的大型团队。

### 三种方案对比

| 方案 | 适用场景 | 优点 | 缺点 |
|------|----------|------|------|
| Workspace | 简单的少量环境 | 内置支持，无需额外工具 | 环境间差异难以表达，state 混用风险 |
| 目录分离 | 中小型团队，2-5 个环境 | 清晰直观，环境完全隔离 | 存在代码重复 |
| Terragrunt | 大型团队，多环境多模块 | 消除重复，依赖管理强大 | 引入额外工具，学习成本较高 |

---

## 模块化（Modules）

### 模块的本质

Terraform 模块是一组放在同一目录下的 `.tf` 文件的集合。模块的本质是**可复用的资源封装**，让你可以像调用函数一样调用一组基础设施定义。

### 本地模块示例：封装 Kubernetes Namespace + RBAC

```hcl
# modules/k8s-namespace/variables.tf
variable "namespace" {
  type        = string
  description = "Kubernetes Namespace 名称"
}

variable "team_members" {
  type        = list(string)
  description = "拥有该 Namespace 操作权限的 ServiceAccount 列表"
  default     = []
}
```

```hcl
# modules/k8s-namespace/main.tf
resource "kubernetes_namespace" "this" {
  metadata {
    name = var.namespace
    labels = {
      managed-by = "terraform"
    }
  }
}

resource "kubernetes_role" "developer" {
  metadata {
    name      = "developer"
    namespace = kubernetes_namespace.this.metadata[0].name
  }

  rule {
    api_groups = ["apps", ""]
    resources  = ["deployments", "pods", "services", "configmaps"]
    verbs      = ["get", "list", "watch", "create", "update", "patch"]
  }
}

resource "kubernetes_role_binding" "developers" {
  for_each = toset(var.team_members)

  metadata {
    name      = "developer-${each.key}"
    namespace = kubernetes_namespace.this.metadata[0].name
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = kubernetes_role.developer.metadata[0].name
  }

  subject {
    kind      = "ServiceAccount"
    name      = each.key
    namespace = kubernetes_namespace.this.metadata[0].name
  }
}
```

```hcl
# modules/k8s-namespace/outputs.tf
output "namespace_name" {
  value = kubernetes_namespace.this.metadata[0].name
}
```

调用该模块时，只需提供参数即可：

```hcl
# environments/prod/main.tf
module "team_a_namespace" {
  source       = "../../modules/k8s-namespace"
  namespace    = "team-a"
  team_members = ["ci-bot", "alice", "bob"]
}

module "team_b_namespace" {
  source       = "../../modules/k8s-namespace"
  namespace    = "team-b"
  team_members = ["ci-bot", "charlie"]
}
```

### 模块版本锁定

引用公共 Registry 的模块时，务必锁定版本，避免上游变更导致意外的基础设施变更：

```hcl
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "= 5.5.1"   # 固定精确版本，不允许自动升级

  name = "production-vpc"
  cidr = "10.0.0.0/16"
}
```

:::tip 版本约束语法
`~> 5.5` 允许升级到 5.x 的最新版，但不允许跨大版本升级。`= 5.5.1` 则完全锁定，是生产环境的推荐做法。
:::

---

## Terraform 与 IaC 工具对比

| 维度 | Terraform | Ansible | CloudFormation |
|------|-----------|---------|----------------|
| 定位 | 基础设施资源编排 | 配置管理 / 应用部署 | AWS 原生资源编排 |
| 语言 | HCL（声明式） | YAML（命令式为主） | JSON/YAML（声明式） |
| 状态管理 | 有显式状态文件 | 无状态 | CloudFormation Stack |
| 多云支持 | 极强（900+ Provider） | 支持（通过模块） | 仅 AWS |
| 幂等性 | 天然幂等 | 依赖模块实现 | 天然幂等 |
| 资源销毁 | `terraform destroy` | 需手动编写 | `aws cloudformation delete-stack` |
| 社区生态 | 非常活跃 | 非常活跃 | AWS 官方维护 |

---

## Terraform 在 CI/CD 中的集成

将 Terraform 集成到 CI/CD 是实现基础设施变更可审计、可回溯的关键步骤。标准流程是：

1. 开发者提交代码变更，触发 CI 流水线
2. CI 执行 `terraform plan`，将结果作为 MR Comment 输出
3. 团队 Review plan 内容，确认变更符合预期后合并 MR
4. 合并到主分支后，CD 阶段执行 `terraform apply`

以下是一个完整的 GitLab CI 流水线示例：

```yaml
# .gitlab-ci.yml
stages:
  - validate
  - plan
  - apply

variables:
  TF_ROOT: ${CI_PROJECT_DIR}/environments/prod
  TF_STATE_NAME: production

default:
  image:
    name: hashicorp/terraform:1.7
    entrypoint: [""]
  before_script:
    - cd ${TF_ROOT}
    - terraform init -backend-config="token=${TF_BACKEND_TOKEN}"

validate:
  stage: validate
  script:
    - terraform validate
    - terraform fmt -check -recursive
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"

plan:
  stage: plan
  script:
    - terraform plan -out=plan.tfplan -no-color | tee plan.txt
    # 将 plan 结果作为 MR Comment 发布
    - |
      PLAN_OUTPUT=$(cat plan.txt)
      curl --request POST \
        --header "PRIVATE-TOKEN: ${GITLAB_API_TOKEN}" \
        --data-urlencode "body=## Terraform Plan\n\`\`\`\n${PLAN_OUTPUT}\n\`\`\`" \
        "${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/merge_requests/${CI_MERGE_REQUEST_IID}/notes"
  artifacts:
    name: plan
    paths:
      - ${TF_ROOT}/plan.tfplan
    expire_in: 1 week
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"

apply:
  stage: apply
  script:
    - terraform apply plan.tfplan
  dependencies:
    - plan
  rules:
    # 仅在 main 分支合并后执行
    - if: $CI_COMMIT_BRANCH == "main"
  environment:
    name: production
  when: manual   # 需要手动确认，增加一道安全阀
```

:::warning 关于 apply 的自动化程度
在 CI/CD 中设置 `when: manual` 要求人工确认 apply，是生产环境的最佳实践。全自动 apply（`when: on_success`）适合开发/测试环境，但生产环境的基础设施变更建议保留人工审核环节。
:::

---

## 常见操作与故障排查

### 状态操作命令

```bash
# 列出 tfstate 中管理的所有资源
terraform state list

# 查看某个资源的详细状态
terraform state show alicloud_instance.web["0"]

# 将资源从一个地址重命名到另一个地址（代码重构时使用，避免触发重建）
terraform state mv alicloud_instance.web alicloud_instance.app

# 从 tfstate 中移除资源（不删除真实资源，仅解除 Terraform 的管理）
terraform state rm alicloud_instance.web["0"]
```

### 将已有资源纳入管理

```bash
# 先在代码中声明对应的 resource 块，再执行 import
terraform import alicloud_vpc.main vpc-bp1234567890abcdef
```

Terraform 1.5+ 支持在 HCL 中直接编写 `import` 块，无需命令行操作：

```hcl
import {
  to = alicloud_vpc.main
  id = "vpc-bp1234567890abcdef"
}
```

### 强制重建资源

```bash
# 标记资源为"污点"，下次 apply 时强制销毁并重建
# （在 Terraform 0.15.2+ 中，taint 命令被 -replace 参数替代）
terraform apply -replace="alicloud_instance.web[\"0\"]"
```

### 常见错误处理

| 错误 | 原因 | 解决方法 |
|------|------|----------|
| `Error acquiring the state lock` | 另一个操作未正常释放锁 | 确认无并发操作后执行 `force-unlock` |
| `Provider authentication failed` | 云账号 AK/SK 未配置或已过期 | 检查环境变量或 Provider 配置 |
| `Resource already exists` | 真实资源存在但不在 tfstate 中 | 使用 `terraform import` 导入 |
| `Error: version constraint` | 本地 Provider 版本与代码要求不匹配 | 执行 `terraform init -upgrade` |

---

## Terraform 与 Ansible 的协作

在完整的自动化交付链路中，Terraform 和 Ansible 各司其职：

```
Terraform                         Ansible
┌─────────────────────┐          ┌─────────────────────────┐
│ 创建 VPC/子网         │          │ 安装 JDK/Nginx           │
│ 创建 ECS 实例        │   →→→    │ 部署应用程序包            │
│ 创建 RDS 数据库      │          │ 配置 Systemd Service     │
│ 创建负载均衡         │          │ 配置监控 Agent           │
└─────────────────────┘          └─────────────────────────┘
    基础设施层                         配置管理层
```

两者连接的关键是将 Terraform 的 `output` 输出转换为 Ansible 的 inventory。以下是一种常用方式：

```bash
# 从 Terraform 获取实例 IP，生成 Ansible inventory
terraform output -json instance_private_ips | \
  python3 -c "
import json, sys
ips = json.load(sys.stdin)
print('[webservers]')
for k, ip in ips.items():
    print(ip)
" > inventory.ini

# 使用生成的 inventory 执行 Ansible
ansible-playbook -i inventory.ini deploy-app.yml
```

更工程化的做法是使用 Ansible 的动态 inventory 插件直接读取 Terraform 状态，或者在 CI/CD 流水线中将两个工具串联成顺序步骤。

---

## 小结

- **Terraform 的核心价值**在于声明式管理资源状态，让基础设施变更可版本控制、可 Review、可回滚
- **tfstate** 是 Terraform 的核心数据，团队使用时必须配置远程后端并启用状态锁
- **执行流程** `init → plan → apply` 中，plan 是安全的关键卡点，要养成 plan 后仔细审查再 apply 的习惯
- **多环境管理**推荐使用目录分离方案，环境之间完全隔离
- **模块化**是避免代码重复的主要手段，版本锁定是模块化的必要实践
- **Terraform 与 Ansible 互补**，前者管资源，后者管配置，通过 output → inventory 串联

---

## 常见问题

### Q1：`terraform plan` 提示"No changes. Infrastructure is up-to-date."，但我确实修改了云上的资源，这是为什么？

Terraform 在执行 `plan` 时默认会先从云平台 API 刷新资源状态（即执行 refresh 操作）。如果 plan 显示无变化，说明 Terraform 检测到真实状态与代码完全一致。你在控制台手动做的修改可能已经被 Terraform 的代码覆盖，或者你修改的是 Terraform 代码中没有声明的属性（Terraform 只管理它声明过的属性，未声明的属性不在它的控制范围内）。如果你希望让 Terraform 接管那部分属性，需要在对应的 resource 块中补充声明。

### Q2：`terraform destroy` 销毁了所有资源，但我只想删除某一个资源怎么办？

有两种方法。第一种：使用 `-target` 参数指定要销毁的资源：`terraform destroy -target="alicloud_instance.web[\"0\"]"`。`-target` 同样适用于 `apply`，可用于单独创建或更新某个资源。第二种：从代码中删除该 resource 块，然后执行 `terraform apply`，Terraform 会自动删除代码中不再存在的资源。需要注意 `-target` 是应急手段，频繁使用会导致 tfstate 与代码不一致，日常操作应优先修改代码。

### Q3：多人协作时，如何防止两个人同时执行 `terraform apply` 导致冲突？

这正是**状态锁**的作用。配置了远程后端（如 S3 + DynamoDB）后，每次 apply 开始时 Terraform 会尝试在 DynamoDB 中创建一条锁记录。如果锁已存在（说明有人正在执行），后续的 apply 会立即报错并退出，不会继续执行。因此，团队协作的首要前提是将远程后端配置到位。此外，在 CI/CD 流水线中只允许 main 分支触发 apply，也能从流程上限制并发。

### Q4：`terraform.tfstate` 里存储了数据库密码，这非常危险，有什么最佳实践？

这是 Terraform 的已知问题，敏感值（如 `random_password` 生成的密码、证书私钥）会以明文存储在 tfstate 中。缓解措施：一、将 tfstate 存入支持服务端加密的远程后端，并严格控制 Bucket 的访问权限；二、对敏感输出使用 `sensitive = true`；三、考虑使用 Vault Provider 将密钥生命周期委托给 HashiCorp Vault 或云平台的密钥管理服务（KMS），让 Terraform 只存储密钥的引用，而非密钥本身；四、绝对不要将 tfstate 文件提交到 Git 仓库，在 `.gitignore` 中明确排除 `*.tfstate` 和 `*.tfstate.backup`。

### Q5：Terraform 能否管理 Kubernetes 资源？和直接写 YAML 有什么区别？

可以，通过 `hashicorp/kubernetes` Provider，Terraform 可以管理 Namespace、Deployment、Service、ConfigMap 等 Kubernetes 资源。相比直接写 YAML，Terraform 的优势在于：可以在同一套代码中同时描述云基础设施（VPC、EKS/ACK 集群本身）和集群内的 Kubernetes 资源，形成统一的管理入口；可以通过变量和模块实现 Kubernetes 资源的参数化复用；状态追踪让变更更安全。劣势在于：Kubernetes 的 YAML 生态工具链（Helm、Kustomize）更为丰富，且 Kubernetes 资源变更频繁时，Terraform 的执行速度相对较慢。实践中，通常推荐用 Terraform 管理集群本身和少量基础资源（Namespace、RBAC），应用的 Deployment/Service 则交由 Helm 或 ArgoCD 管理。

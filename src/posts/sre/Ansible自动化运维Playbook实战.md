---
date: 2026-02-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - SRE
tag:
  - SRE
  - ClaudeCode
---

# Ansible 自动化运维 Playbook 实战

上一篇文章介绍了 Ansible 的基础架构和核心概念。本文从生产环境视角出发，深入剖析四个进阶主题：Playbook 的高级编排机制、Vault 密钥管理体系、Dynamic Inventory 动态清单原理，以及 Rolling Update 滚动发布设计。这些内容的共同特点是：理解原理比记住语法更重要。

---

## 一、Playbook 编排机制深度解析

### Handlers：事件驱动的通知链

初学者常犯的错误是直接在任务后跟一个"重启服务"的任务。这种做法有两个问题：一是无法做到按需触发（无论配置是否变更，每次都重启）；二是当多个任务都修改了同一个服务的配置时，服务会被重启多次。

Handlers 解决的正是这个问题。其核心是一个**发布-订阅模型**：

```
task (修改配置) → notify → handler (重启服务)
```

Handlers 有两个关键行为值得深挖：

**行为一：去重合并执行。** 无论一次 Play 中有多少个 task 通知了同一个 handler，该 handler 只会在 Play 结束时执行一次。Ansible 内部维护了一个"待触发 handler 集合"，相同名称的 handler 只入队一次。这保证了配置变更后服务只被重启一次，不论变更来自几个不同的任务。

**行为二：执行时机是 Play 结束后，而非 task 完成后。** 这意味着所有 task 执行完毕后才统一触发 handlers。如果你需要在任务中途就触发某个 handler，需要显式调用 `meta: flush_handlers`。这个机制在滚动发布场景中尤为重要，后文会详细说明。

**通知链的传递：** Handler 自身也可以 notify 另一个 handler，形成通知链。例如：更新证书 → 通知 reload nginx → nginx reload 后通知 check_ssl_health。这种链式设计让复杂的服务重启编排变得清晰可维护。

### Tags：执行图的精细裁剪

生产环境中完整跑一次 Playbook 可能耗时数十分钟。Tags 允许你在执行时裁剪任务图，只运行标记了特定 tag 的任务子集。

Tags 的设计逻辑值得理解：它是静态标注，不是动态过滤条件。你在写 Playbook 时就决定了哪些任务属于哪些语义分类（install/config/deploy），执行时通过 `--tags` 或 `--skip-tags` 告诉 Ansible 只关心哪部分。

有几个特殊 tag 需要注意：

- `always`：无论 `--tags` 如何指定，标记了 `always` 的任务都会执行。适合用于收集 Facts、检查前置条件等必须运行的任务。
- `never`：与 `always` 相反，除非明确用 `--tags never` 指定，否则永远跳过。适合标注危险操作（如清空数据库）作为保护机制。
- `tagged` / `untagged`：分别匹配"有任何 tag 的任务"和"没有 tag 的任务"。

### Block/Rescue/Always：结构化错误处理

Ansible 的错误处理机制经历了从简陋到完善的演进过程。早期只有 `ignore_errors: yes` 这种粗糙的方式，现代 Ansible 借鉴了编程语言的 try/catch/finally 语义，引入了 Block。

```
Block    ← 正常执行的任务组（对应 try）
Rescue   ← Block 失败时执行的任务组（对应 catch）
Always   ← 无论成功失败都执行的任务组（对应 finally）
```

这个机制在有状态操作中极其重要。设想一个数据库迁移场景：

1. Block 中执行：备份数据库 → 运行迁移脚本 → 更新配置
2. 如果迁移脚本失败，Rescue 执行回滚操作并发送告警
3. Always 中执行清理临时文件（无论成功与否，临时文件都需要删除）

没有 Block 机制，这个逻辑需要大量的 `register` + `when` 条件判断来模拟，既繁琐又容易出错。

一个微妙的细节：Rescue 中的任务如果成功执行，整个 Block 被视为成功（play 不会因此失败）。这意味着 Rescue 是真正的"补救"——如果你在 Rescue 中成功回滚了状态，Ansible 不会把整个 play 标记为失败，从而允许后续 play 继续运行。

### 异步任务与轮询：突破 SSH 超时的束缚

默认情况下，Ansible 通过 SSH 执行任务并等待其完成。对于耗时超过 SSH 超时阈值（通常 30 分钟以上）的任务，例如大型数据库导入、固件升级，这种同步模式会导致连接超时失败。

异步模式的工作原理是：Ansible 在目标主机上启动一个后台进程来执行任务，同时在本地（一般位于 `~/.ansible_async/` 目录）保存一个 job ID。任务完成后 Ansible 可以通过 `async_status` 模块用 job ID 查询执行结果。

```yaml
- name: 执行耗时数据迁移
  command: /opt/scripts/migrate_data.sh
  async: 3600    # 最长运行 1 小时
  poll: 0        # 不等待，立即返回

- name: 等待迁移完成
  async_status:
    jid: "{{ migrate_task.ansible_job_id }}"
  register: job_result
  until: job_result.finished
  retries: 60
  delay: 60
```

`poll: 0` 是关键：它让 Ansible 不阻塞等待，而是把 job ID 存入 register 变量后继续执行后续任务。这样你可以在等待期间并行执行其他操作，最后再用 `async_status` 汇合结果。

### delegate_to：跨主机操作的桥梁

`delegate_to` 允许将某个任务委托给另一台主机执行，但上下文（变量、Facts）仍然来自当前目标主机。这是一个反直觉但极其实用的机制。

典型场景：在部署 web 节点前，需要从负载均衡器上摘除该节点。此时你希望遍历每台 web 节点，但"摘流"这个动作需要在负载均衡器上执行：

```yaml
- name: 从负载均衡器摘除节点
  uri:
    url: "http://lb.internal/api/disable/{{ inventory_hostname }}"
    method: POST
  delegate_to: loadbalancer.internal
```

这里 `inventory_hostname` 是当前循环的 web 节点主机名，但 `uri` 模块实际运行在 `loadbalancer.internal` 上。这种设计让你可以在同一个任务循环中协调多台主机的行为，而不需要把逻辑拆成多个 Play。

`delegate_to: localhost` 是一个特殊用法，代表在控制节点本地执行，常用于调用云 API、更新本地 DNS、通知监控系统等操作。

---

## 二、Ansible Vault 深度加密体系

### 文件级加密 vs 变量级加密

Vault 提供两种粒度的加密方式，选择哪种取决于机密数据的分布方式：

| 维度 | 文件级加密 | 变量级加密 |
|------|------------|------------|
| 操作单位 | 整个文件 | 单个变量值 |
| 可读性 | 加密后文件不可读 | 文件结构可读，只有值被加密 |
| diff 友好 | 不友好（文件变更无法 diff） | 友好（变量名和结构可见） |
| 使用场景 | 独立密钥文件、证书文件 | 混合在普通变量文件中的密码 |

变量级加密的格式是 `!vault |` 加上 AES256 加密的密文块，存放在普通 YAML 文件中：

```yaml
# group_vars/production/vars.yml
db_host: db.prod.internal   # 明文，可读
db_port: 5432               # 明文，可读
db_password: !vault |       # 只有这个值被加密
  $ANSIBLE_VAULT;1.1;AES256
  62343738316338633...
```

这种方式的优势在于 code review 时评审者可以看到变量名和文件结构，只有具体的密码值是不可读的，兼顾了安全性和可审查性。

### vault-id 多密码管理

当一个项目涉及多个环境（dev/staging/prod）或多个团队时，使用同一个 vault 密码会带来管理困境：你无法为不同的环境设置不同的访问权限。vault-id 解决了这个问题。

vault-id 的本质是给加密内容贴标签，标签用于在解密时匹配对应的密码来源：

```bash
# 用不同的 vault-id 加密不同环境的变量
ansible-vault encrypt_string 'prod_secret' --vault-id prod@prompt
ansible-vault encrypt_string 'dev_secret'  --vault-id dev@~/.vault-dev
```

加密后的密文头部会包含 vault-id 标识：`$ANSIBLE_VAULT;1.2;AES256;prod`。解密时 Ansible 通过这个标识找到对应的密码来源，无需尝试所有密码。

实际项目中的推荐实践：生产环境的 vault-id 使用 HashiCorp Vault 或 AWS Secrets Manager 存储，通过脚本动态读取；开发环境使用本地文件；CI/CD 环境使用环境变量注入。

### CI/CD 环境中的 Vault 集成

自动化流水线中无法交互式输入密码，有三种主流方案：

**方案一：vault-password-file 文件注入。** CI/CD 平台（Jenkins/GitLab CI）将 vault 密码作为 Secret 变量注入，pipeline 在执行前将其写入临时文件，任务结束后删除。

```bash
# GitLab CI 示例
deploy:
  script:
    - echo "$VAULT_PASSWORD" > /tmp/.vault-pass
    - ansible-playbook deploy.yml --vault-password-file /tmp/.vault-pass
    - rm -f /tmp/.vault-pass
```

**方案二：vault-password-file 脚本形式。** 密码文件可以是一个可执行脚本，Ansible 会执行该脚本并将其 stdout 作为密码。这允许动态从 KMS 或 Secrets Manager 获取密码，避免密码静态存储在 CI 系统中。

**方案三：与外部 Secret 存储集成。** 通过 `community.hashi_vault` 插件，直接在 Playbook 执行时从 HashiCorp Vault 动态拉取密钥，不需要预先加密变量，Vault 本身承担了加密存储的角色。

### 密码轮换（rotate-vault-password）的设计

vault 密码应该定期轮换，尤其是在人员变动后。`ansible-vault rekey` 命令可以用新密码重新加密所有 vault 文件。

轮换流程中最大的风险是：如果有未入库的本地加密文件使用了旧密码，轮换后这些文件将无法解密。因此好的实践是：

1. 所有加密文件必须纳入版本控制
2. 轮换前用 `git grep '$ANSIBLE_VAULT'` 找出所有加密内容
3. 轮换后立即在 CI/CD 中验证解密是否正常
4. 新旧密码短暂并存（通过 vault-id 机制），确保所有文件迁移完成后再废弃旧密码

---

## 三、Dynamic Inventory：让清单感知基础设施

### 静态 Inventory 的根本局限

静态 Inventory 的本质是"将时间的瞬态快照固化成代码"。在虚拟机随时创建销毁、弹性扩缩容的云原生时代，这种方式面临三个无法解决的问题：

1. **漂移问题**：新加入的节点不会自动出现在 Inventory 中
2. **过期问题**：已销毁的节点不会自动从 Inventory 中移除，导致任务失败
3. **分组语义缺失**：云平台的标签体系（Tag）无法映射到静态 Inventory 的分组

Dynamic Inventory 的核心思路是：**Inventory 不是文件，而是查询**。每次运行 Playbook 时，实时向权威数据源（云平台 API、CMDB）查询当前有哪些主机，再动态构建分组。

### Inventory 插件体系

现代 Ansible 通过 Inventory 插件来实现动态清单。与早期的脚本方式相比，插件方式是声明式的——你只需要一个 YAML 配置文件，描述你想查询哪个数据源、按照什么规则分组：

```yaml
# aws_ec2.yml
plugin: amazon.aws.aws_ec2
regions:
  - ap-northeast-1
filters:
  instance-state-name: running
  tag:Environment: production
keyed_groups:
  - key: tags.Role
    prefix: role
  - key: placement.availability_zone
    prefix: az
```

这个配置让 Ansible 自动按照 EC2 标签中的 `Role` 值和可用区来分组主机，无需人工维护分组关系。常用插件：

- `amazon.aws.aws_ec2`：AWS EC2 实例
- `google.cloud.gcp_compute`：GCP Compute Engine
- `kubernetes.core.k8s`：Kubernetes Pod/Node
- `community.vmware.vmware_vm_inventory`：VMware 虚拟机

### 自定义 Inventory 脚本协议

当没有合适的插件时，可以编写自定义 Inventory 脚本。脚本需要实现一个标准协议，支持两个参数：

**`--list` 参数**：返回完整的 inventory JSON，包含所有分组及其成员：

```json
{
  "webservers": {
    "hosts": ["10.0.1.10", "10.0.1.11"],
    "vars": { "http_port": 80 }
  },
  "_meta": {
    "hostvars": {
      "10.0.1.10": { "ansible_user": "ec2-user" }
    }
  }
}
```

**`--host <hostname>` 参数**：返回特定主机的变量（当 `_meta.hostvars` 已经包含所有主机变量时，Ansible 会跳过 `--host` 调用，避免对每台主机单独查询）。

`_meta.hostvars` 的设计是一个重要的性能优化点：通过在 `--list` 时一次性返回所有主机变量，避免了 N 次 `--host` 调用。对于有大量主机的环境，这个区别可能意味着 inventory 构建时间从几分钟降低到几秒。

### Inventory 缓存策略

即便使用插件，每次运行 Playbook 都实时查询云 API 仍然有代价：速度慢、产生 API 调用成本、在 API 限速时可能失败。缓存机制允许将查询结果存储一段时间：

```ini
# ansible.cfg
[inventory]
cache = true
cache_plugin = jsonfile
cache_connection = /tmp/ansible_inventory_cache
cache_timeout = 3600
```

缓存的粒度是整个 Inventory 的查询结果。需要注意的是：强制刷新缓存使用 `--flush-cache` 参数。在自动化场景中，应该在基础设施变更（如扩容）后显式刷新缓存，避免新节点无法被发现。

### 与 CMDB 集成的架构思路

企业环境通常有自建的 CMDB（配置管理数据库）作为主机信息的权威来源。CMDB 集成的本质是让 CMDB 成为 Inventory 的数据源：

```
CMDB (权威数据源)
      ↓ HTTP API
自定义 Inventory 脚本/插件
      ↓ JSON 协议
Ansible Inventory
      ↓
Playbook 执行
```

关键设计决策：**CMDB 中的分组维度如何映射到 Ansible 分组**。通常有两种映射策略：

- **直接映射**：CMDB 中的应用名称、环境、机房 → 直接作为 Ansible 分组名
- **标签映射**：CMDB 中的多维标签 → 通过 `keyed_groups` 机制动态生成 Ansible 分组

前者简单但耦合紧密，后者灵活但需要在 CMDB 数据规范化上投入。生产环境建议采用缓存 + CMDB 的架构：Inventory 脚本从 CMDB 拉取数据并缓存到本地，Playbook 运行时命中缓存，只有缓存过期或显式刷新时才重新查询 CMDB。

---

## 四、Rolling Update 滚动发布设计

### serial：控制并发批次的三种形式

`serial` 关键字控制同一时刻有多少台主机并发执行 Play。没有 `serial` 时，Ansible 对所有主机并行执行（受 `forks` 限制），这在发布场景下意味着"全量同时发布"——可用性最低的方案。

`serial` 支持三种写法，对应三种不同的发布策略：

```yaml
# 写法一：固定数量
serial: 2           # 每批 2 台

# 写法二：百分比
serial: "20%"       # 每批 20% 的主机

# 写法三：列表（阶梯式）
serial:
  - 1               # 第一批：1 台（金丝雀验证）
  - "10%"           # 第二批：10%（小规模验证）
  - "100%"          # 第三批：剩余全部
```

列表写法是生产环境最推荐的形式。它实现了金丝雀发布的精髓：先用极小规模验证新版本，确认无问题后再逐步扩大范围。每个批次都是一个完整的 Play 执行周期，包含所有 tasks 和 handlers。

### max_fail_percentage：熔断保护

单独使用 `serial` 控制了发布速度，但没有解决"发布途中发现问题如何止损"的问题。`max_fail_percentage` 提供了熔断机制：

```yaml
- hosts: webservers
  serial: "10%"
  max_fail_percentage: 20
```

含义是：如果当前批次中超过 20% 的主机任务失败，立即停止整个 Playbook，不再处理剩余批次。注意这个百分比是针对每个批次内部的，不是针对全量主机。

这个机制的设计与熔断器（Circuit Breaker）模式高度相似：在局部失败率超过阈值时主动停止，防止错误扩散到全量节点。将其设为 0 意味着"任何一台失败都停止"，适合高可用要求严格的生产发布；设为较高的值适合允许部分失败的场景。

### pre_tasks 和 post_tasks 生命周期钩子

`pre_tasks` 在 roles 和 tasks 之前执行，`post_tasks` 在之后执行。在滚动发布语境下，它们配合 `serial` 实现每批次的前置/后置操作：

```
[批次 N 开始]
pre_tasks  → 从 LB 摘流、等待连接排空
tasks      → 停服务、更新代码、启动服务
handlers   → 重载配置
post_tasks → 健康检查通过后挂回 LB
[批次 N 结束，开始批次 N+1]
```

关键点：`pre_tasks` 和 `post_tasks` 是在每个批次内执行的，不是在整个 Playbook 开始/结束时执行。这意味着对于 10 台主机 `serial: 2` 的发布，摘流/挂回操作会执行 5 次，每次针对当前批次的 2 台主机。

### 摘流与挂回的协调方案

滚动发布的核心挑战是：如何让负载均衡器与发布过程保持同步？这里有几种常见模式：

**模式一：直接 API 调用（适合云 LB）**

借助 `delegate_to: localhost` 在控制节点上调用云 LB 的 API，摘除/添加目标主机：

```yaml
pre_tasks:
  - name: 从 ALB 目标组摘除节点
    community.aws.elb_target:
      target_group_arn: "{{ target_group_arn }}"
      target_id: "{{ hostvars[inventory_hostname].instance_id }}"
      state: absent
    delegate_to: localhost

  - name: 等待现有连接排空
    pause:
      seconds: 30
```

**模式二：通过负载均衡器节点执行**

借助 `delegate_to` 在 LB 节点上执行 `upstream` 配置变更，适合 Nginx/HAProxy 等软件负载均衡器：

```yaml
pre_tasks:
  - name: 将节点标记为 maintenance
    community.general.haproxy:
      state: disabled
      host: "{{ inventory_hostname }}"
      socket: /run/haproxy/admin.sock
    delegate_to: "{{ groups['loadbalancers'][0] }}"
```

**连接排空等待：** 摘流后不能立即发布，需要等待已建立的连接处理完毕。`wait_for` 模块可以监控连接数降为 0，或者使用固定的 `pause` 等待时间。连接排空时间依赖于业务特性（长连接 vs 短连接），需要根据实际情况调整。

### Blue-Green 部署的 Ansible 实现思路

Blue-Green 部署的本质是维护两套完全相同的环境，通过切换路由来实现零停机发布。Ansible 实现 Blue-Green 的核心是 Inventory 分组的动态切换：

```
Inventory 分组设计：
  blue_webservers  → 当前活跃环境
  green_webservers → 待发布环境

发布流程：
1. 确定当前活跃色（读取 LB 配置或状态文件）
2. 向非活跃色的主机组发布新版本（green_webservers）
3. 在 green 环境执行冒烟测试
4. 切换 LB 路由：将流量从 blue 切到 green
5. 观察一段时间
6. 旧 blue 环境保留（用于快速回滚），下次发布时角色互换
```

与滚动发布的对比：Blue-Green 要求双倍资源，但发布风险更低（可以在流量切换前充分验证）；滚动发布资源效率高，但发布期间新旧版本并存，需要应用支持向后兼容。

在 Ansible 实现中，"当前活跃色"的状态需要持久化存储（可以是 LB 配置、本地文件、或 Consul/etcd 中的键值），Playbook 在执行时读取这个状态来决定操作哪个分组。

---

## 小结

- **Handlers** 的去重合并机制避免了服务被反复重启；`flush_handlers` 允许在 Play 中途强制触发
- **Block/Rescue/Always** 将错误处理从散乱的条件判断升级为结构化的异常处理框架
- **Vault** 的变量级加密比文件级加密更具 code review 友好性；vault-id 实现了多环境密码隔离
- **Dynamic Inventory** 的核心是让 Inventory 从静态快照变成实时查询；`_meta.hostvars` 是批量返回主机变量的性能优化关键
- **Rolling Update** 中 `serial` 的列表写法实现金丝雀发布，`max_fail_percentage` 提供熔断保护，`pre_tasks`/`post_tasks` 协调摘流挂回时序

---

## 常见问题

### Q1：Handlers 在哪些情况下不会被触发？

**A：** 有三种情况 handler 不会被触发：一是通知它的 task 没有产生变更（`changed: false`），handler 的前提是 task 返回 `changed` 状态，而非仅仅执行了；二是通知它的 task 执行失败，失败的 task 不会触发 handler（除非使用了 `force_handlers: yes` 配置）；三是 Play 在到达 handler 执行时机之前被 `--limit` 或错误中断。理解"handler 只在 task 变更时触发"这一前提，可以帮助排查大量"配置已修改但服务没有重启"的问题。

### Q2：为什么要使用 Dynamic Inventory 而不是写自动化脚本维护静态 Inventory？

**A：** 表面上看，定期运行脚本更新静态 Inventory 也能解决数据陈旧的问题，但这种方案存在一个根本性缺陷：**时序窗口问题**。从脚本更新 Inventory 到 Playbook 执行之间存在时间差，这段时间内发生的基础设施变更（扩容、缩容、节点替换）不会被反映。Dynamic Inventory 在 Playbook 执行时实时查询，消除了这个时序窗口。此外，维护一个"更新脚本"本身就是额外的运维负担，而 Inventory 插件是声明式配置，可维护性更高。

### Q3：Ansible Vault 和 HashiCorp Vault 有什么关系？如何选择？

**A：** 两者是不同的产品，解决的问题有所重叠但侧重不同。Ansible Vault 是 Ansible 内置的加密工具，专注于加密 Ansible Playbook 中的敏感变量，密文跟随代码存入 Git 仓库，优点是零额外依赖，缺点是密钥轮换和权限管理较粗糙。HashiCorp Vault 是专业的密钥管理系统，提供动态密钥（按需生成、自动过期）、细粒度访问控制、完整审计日志等企业级功能。选择原则：小型团队或简单场景使用 Ansible Vault 即可；有多团队协作、严格合规要求、需要动态密钥的场景，应该引入 HashiCorp Vault，Ansible 通过 `hashi_vault` lookup 插件读取，不需要在 Playbook 中存储任何密文。

### Q4：滚动发布过程中如何处理数据库 schema 变更？

**A：** 这是滚动发布中最复杂的问题之一。核心约束是：滚动发布期间新旧两个版本的应用同时运行，如果新版本的 DB schema 不向后兼容，老版本的应用会崩溃。推荐的处理方式是"展开-收缩（Expand-Contract）"模式，分三次发布：第一次只做向后兼容的 schema 扩展（加字段但不删）；第二次发布应用新版本，此时新旧应用都能正常工作；第三次清理不再需要的旧字段。在 Ansible 实现中，数据库迁移脚本应该在 `pre_tasks` 中作为单独步骤执行（在所有应用节点发布前），并且迁移脚本本身需要设计为幂等的。

### Q5：如何测试和验证 Playbook 在不同场景下的错误处理行为？

**A：** 验证 Playbook 错误处理主要有三种方法：一是使用 `--check` 模式（Dry Run），它会模拟执行但不实际变更，可以快速发现语法和逻辑错误，但无法模拟真实的运行时失败；二是使用 Molecule 框架，它可以在 Docker 容器或虚拟机中执行完整的 Playbook 生命周期，并支持编写 Verify 步骤来断言执行结果，是目前最接近真实环境的 Playbook 测试方案；三是在 staging 环境中使用故障注入：通过手动设置 `fail` 任务、修改文件权限等方式模拟各种失败场景，验证 Rescue 块和 `max_fail_percentage` 熔断逻辑是否按预期工作。对于生产发布，强烈建议 Playbook 中的每个 Block/Rescue 结构都在 staging 上做过失败路径验证，不要把生产环境当作错误处理逻辑的首次测试场地。

## 参考资源

- [Ansible 官方文档](https://docs.ansible.com/)
- [Ansible Best Practices](https://docs.ansible.com/ansible/latest/user_guide/playbooks_best_practices.html)
- [Molecule 测试框架](https://molecule.readthedocs.io/)

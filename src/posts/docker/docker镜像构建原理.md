---
date: 2026-02-11
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - docker
tag:
  - docker
  - ClaudeCode
---

# Docker 镜像构建原理

你可能已经写过很多 Dockerfile，也知道分层存储和缓存机制的基本概念。但当你执行 `docker build .` 的时候，Docker 引擎内部究竟发生了什么？为什么有时候改一行 RUN 指令缓存就全失效了？BuildKit 和旧版构建器有什么本质区别？

这篇文章不讲怎么写 Dockerfile，而是带你深入构建引擎的内部机制，理解那些"知其然"背后的"知其所以然"。

## 构建架构的演进：Legacy Builder vs BuildKit

在 Docker 18.09 之前，所有镜像构建都由一个叫 **Legacy Builder**（也叫 classic builder）的组件完成。它的工作模型非常直白：

```
Dockerfile 第 N 行指令
        ↓
  创建临时容器（基于上一层镜像）
        ↓
  在容器内执行该指令
        ↓
  docker commit（把容器文件系统差异保存为新层）
        ↓
  删除临时容器
        ↓
  Dockerfile 第 N+1 行指令
```

这个"临时容器 → 执行 → commit → 删除"的流水线，每条产生层的指令都要走一遍。它简单可靠，但有明显局限：

- **串行执行**：多阶段构建的各个阶段只能一个接一个地跑，即使它们互相独立
- **上下文全量传输**：构建上下文一次性全部发给 daemon，哪怕 Dockerfile 只用到其中一个文件
- **缓存能力有限**：缓存只能在本地使用，无法导出共享

**BuildKit** 是 Docker 从 18.09 开始引入的新一代构建引擎，从 Docker 23.0 起成为默认引擎。它的核心创新是引入了 **LLB（Low-Level Build）** 中间表示，把 Dockerfile 编译成一个有向无环图（DAG），再按依赖关系调度执行。

```
                   Dockerfile
                       ↓
              BuildKit 前端编译
                       ↓
              LLB DAG（有向无环图）
              /         |         \
           阶段A       阶段B       阶段C
          (独立)      (独立)     (依赖A,B)
            ↓           ↓
         并行执行     并行执行
              \         /
               合并依赖
                  ↓
                阶段C
```

有了 DAG，BuildKit 可以分析哪些阶段之间没有依赖关系，然后并行执行它们。对于一个有多个独立 base 阶段的多阶段构建，这意味着实际构建时间可以大幅缩短。

## 构建上下文的底层机制

执行 `docker build .` 时，末尾的 `.` 是构建上下文（build context）的路径。这个过程在客户端侧发生的事情，很多人没有仔细想过。

### 客户端打包与传输

Docker 客户端会把整个上下文目录打包成一个 **tar 流**，通过 Unix socket（或 TCP）发送给 daemon。注意两个关键点：

1. `.dockerignore` 的解析发生在**客户端**，且在打包之前。被忽略的文件不会进入 tar 流，根本不会传给 daemon。这也是为什么 `.dockerignore` 能有效减小构建上下文体积——它在数据离开本机之前就完成了过滤。

2. `.dockerignore` 的匹配规则遵循 Go 的 `filepath.Match` 语义，支持 `*`（匹配单个路径段内的任意字符）、`**`（跨目录匹配）、`!`（排除规则）。规则的顺序有意义：后面的规则可以覆盖前面的。

```
# .dockerignore 示例
node_modules          # 排除整个目录
**/*.log              # 排除所有 .log 文件
!important.log        # 但保留这一个
dist/                 # 排除 dist 目录
```

### BuildKit 对上下文传输的优化

Legacy Builder 把整个上下文都发送出去，但 BuildKit 支持**按需传输**：它分析 Dockerfile 中实际引用了哪些文件（COPY 指令的源路径），只传输真正需要的部分。

更进一步，BuildKit 还支持**增量传输**。当你指定远程 Git 仓库或 HTTP URL 作为上下文时，BuildKit 可以直接在 daemon 端拉取，完全绕开客户端打包这一步：

```bash
# 直接用 Git 仓库作为构建上下文
docker build https://github.com/user/repo.git#main
```

## 指令执行的内部过程

### Legacy Builder：每条指令的生命周期

以一个简单的 Dockerfile 为例，来看 Legacy Builder 处理每条指令时具体发生了什么：

```dockerfile
FROM ubuntu:22.04          # Step 1
RUN apt-get update         # Step 2
COPY app.py /app/          # Step 3
CMD ["python", "/app/app.py"] # Step 4
```

**Step 1（FROM）**：不创建新层，而是把基础镜像的最顶层作为工作起点。

**Step 2（RUN）**：
1. 基于当前镜像创建一个临时容器（容器 ID 类似 `abc123`）
2. 在容器内执行 `apt-get update`
3. 调用 `docker commit abc123` 把文件系统的差异（新增/修改/删除的文件）保存为一个新层
4. 删除临时容器 `abc123`

**Step 3（COPY）**：同样走"创建容器 → 操作 → commit → 删除"的流程，把文件复制到容器文件系统后再 commit。

**Step 4（CMD）**：这一步**不创建新层**。CMD 只是修改镜像的元数据（`config.json` 中的 `Cmd` 字段），不涉及任何文件系统变化，因此不需要 commit 操作。

### 哪些指令创建层，哪些不创建

这是一个值得明确记住的区分：

| 创建新层（写入文件系统）| 只修改镜像配置（不创建层）|
|---|---|
| RUN | CMD |
| COPY | ENTRYPOINT |
| ADD | ENV |
|  | EXPOSE |
|  | LABEL |
|  | USER |
|  | WORKDIR（但会记录在配置中）|
|  | ARG |
|  | STOPSIGNAL |
|  | HEALTHCHECK |

严格来说，WORKDIR 有些特殊：如果目标目录不存在，它会创建目录（产生文件系统变化），因此算作创建层；如果目录已存在，则只更新配置。

### BuildKit：Snapshot 机制替代临时容器

BuildKit 不使用临时容器来执行指令。它引入了 **Snapshot** 的概念，直接在存储驱动层面操作：

- 每个操作的输入是一个或多个 snapshot（只读）
- 操作的输出是一个新的 snapshot（最终会变成只读层）
- 操作本身在一个隔离的执行环境中运行，但这个环境是轻量的 rootfs 挂载，而不是完整意义上的 Docker 容器

这种方式的好处是：执行单元更轻量，不需要走完整的容器创建/删除流程；同时 BuildKit 可以更精细地控制 snapshot 的生命周期和复用。

## 内容寻址存储：层是如何被识别的

Docker 的存储系统采用**内容寻址存储（Content-Addressable Storage，CAS）**，也就是说，一个层的"名字"是由它的内容决定的，内容不变则名字不变。这个机制涉及几个容易混淆的概念。

### DiffID：单层的身份证

DiffID 是对**单个层的 tar 包内容**计算 SHA256 得到的摘要。只要两个层的文件内容完全相同，它们的 DiffID 就相同，即使它们来自不同的镜像。

```
sha256:a3d63ded...  ← 这就是一个 DiffID
```

### ChainID：层在栈中的位置标识

光有 DiffID 还不够，因为同一个层放在不同的"层叠位置"，语义是不同的。ChainID 的计算把层的位置信息编码进去：

```
# 最底层
ChainID(layer1) = DiffID(layer1)

# 第二层
ChainID(layer2) = SHA256("sha256:" + ChainID(layer1) + " " + DiffID(layer2))

# 第三层，以此类推
ChainID(layer3) = SHA256("sha256:" + ChainID(layer2) + " " + DiffID(layer3))
```

这种链式计算意味着：即使两个层的内容（DiffID）相同，如果它们在层栈中的位置不同（父层不同），它们的 ChainID 也不同。ChainID 是在 `image/overlay2/layerdb/sha256/` 目录下组织层元数据的依据。

### CacheID：overlay2 目录名的来源

在 `/var/lib/docker/overlay2/` 目录下，你会看到一堆看起来像随机字符串的目录名：

```
/var/lib/docker/overlay2/
├── 3c9c6e12a1b4f8d7e2...
├── 7a4f2b9d1c3e5f8a1...
└── b2e9f4a7d1c6e3b8f...
```

这些目录名就是 **CacheID**。CacheID 是 daemon 在本地为每个层随机生成的标识符，它不是内容的哈希，而是存储驱动（overlay2）用来管理磁盘上目录的内部 ID。

三者关系可以这样理解：

```
内容层面:  DiffID (我是谁)
              ↓
栈位置层面: ChainID (我在哪个位置)
              ↓
磁盘管理层面: CacheID (我住在哪个目录)
```

### 镜像配置（Image Config）和 Manifest

一个完整的镜像在 OCI 规范下由两部分组成：

**Image Config**（`config.json`）：JSON 文件，包含：
- `rootfs.diff_ids`：该镜像所有层的 DiffID 列表（有序）
- 运行时配置：Cmd、Entrypoint、Env、WorkingDir 等
- 构建历史：每条指令的信息（主要供人查看）

**Manifest**：描述镜像的组成，包含 Image Config 的摘要和每个层的压缩包（blob）摘要。Registry 上存的是压缩后的层，DiffID 是对解压后内容的哈希，而 Manifest 中的 digest 是对压缩包内容的哈希——这两者是不同的。

```
Registry 上:                    本地:
Manifest                        Image Config (config.json)
  ├── config.digest ──────────→   ├── rootfs.diff_ids
  └── layers[]:                   │     ├── DiffID(layer1)
        ├── compressed.digest     │     ├── DiffID(layer2)
        └── ...                   │     └── DiffID(layer3)
                                  └── (运行时配置...)
```

## 缓存机制的完整内部逻辑

缓存是构建速度的关键，理解它的工作原理能帮你写出缓存友好的 Dockerfile，也能帮你在缓存"不按预期工作"时准确定位原因。

### RUN 指令的缓存判断

对于 RUN 指令，Legacy Builder 的缓存 key 是：

```
(父层的 ChainID) + (指令字符串)
```

只要这两个输入不变，就命中缓存。这意味着：

- 你改了 RUN 指令的内容 → 缓存失效（指令字符串变了）
- 你改了这条 RUN 之前的某条指令 → 缓存失效（父层变了，导致 ChainID 变了）
- 你什么都没改 → 命中缓存

注意：RUN 指令的缓存**不检查外部资源的变化**。如果你的 `RUN apt-get update` 今天的结果和一个月前不同，但指令字符串和父层都没变，Docker 依然会使用旧缓存。这就是为什么有时候需要用 `--no-cache` 强制刷新。

### COPY/ADD 指令的缓存判断

COPY 和 ADD 指令的缓存 key 不只是指令字符串，还包含**被复制文件的内容 checksum**（SHA256）：

```
(父层的 ChainID) + (指令字符串) + (源文件内容的 checksum)
```

Docker 遍历 COPY 指令涉及的所有文件，计算它们内容的哈希，而**不是**看文件的修改时间（mtime）。这个设计很重要：即使你 `touch` 了一个文件更新时间，只要内容没变，COPY 的缓存依然有效。

### 缓存失效的链式传播

缓存失效会**向后级联**。一旦某条指令的缓存失效，它后面的所有指令也都必须重新执行，即使那些指令的输入完全没有变化。

```
FROM ubuntu:22.04          ← 缓存命中
RUN apt-get install -y python  ← 缓存命中
COPY . /app                ← 缓存失效（你改了某个源文件）
RUN pip install -r /app/requirements.txt  ← 强制重新执行
CMD ["python", "/app/main.py"]  ← 强制重新执行
```

这就是为什么 Dockerfile 中"把变化频率高的指令放后面"是一个重要原则——让缓存失效的影响范围尽可能小。

### BuildKit 的高级缓存能力

BuildKit 在缓存方面有几个 Legacy Builder 完全没有的能力：

**Cache Mount（持久化构建缓存目录）**

```dockerfile
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install -r requirements.txt
```

`/root/.cache/pip` 是 pip 的本地缓存目录。通过 `--mount=type=cache`，这个目录在多次构建之间会被**持久化保留**，不会被打进镜像层里，也不会因为层缓存失效而丢失。下次构建时，pip 可以直接复用已下载的包，大幅加速安装速度。

同样的机制适用于 Go 的 `GOMODCACHE`、Node 的 npm 缓存、Maven 的本地仓库等。

**Secret Mount（安全注入密钥）**

```dockerfile
RUN --mount=type=secret,id=npmrc,target=/root/.npmrc \
    npm install
```

构建时通过 `docker build --secret id=npmrc,src=.npmrc .` 传入密钥。密钥只在当前 RUN 指令执行期间挂载，**不会写入镜像层**。这解决了一个经典的安全问题：以前很多人会在 RUN 中写入密钥再删除，但由于层的不可变性，删除操作只会在新层中标记删除（whiteout），密钥内容在旧层中依然存在。

**SSH Mount（转发 SSH agent）**

```dockerfile
RUN --mount=type=ssh \
    git clone git@github.com:private/repo.git
```

BuildKit 把宿主机的 SSH agent socket 转发到构建环境中，私钥从不离开宿主机，镜像中不会留下任何凭证。

**缓存导入/导出**

BuildKit 支持把构建缓存导出到外部，再在其他机器上导入使用：

```bash
# 构建并导出缓存到本地目录
docker buildx build --cache-to type=local,dest=/tmp/cache .

# 使用导出的缓存（比如在 CI 机器上）
docker buildx build --cache-from type=local,src=/tmp/cache .
```

还支持把缓存内联在镜像中推送到 Registry（`type=inline`），或直接用 Registry 存缓存（`type=registry`）。这让 CI 流水线中的缓存复用成为可能，不再局限于单台机器的本地缓存。

## BuildKit 的 LLB：构建图的本质

前面多次提到 LLB（Low-Level Build），这里单独说清楚它是什么，以及为什么它很重要。

Dockerfile 是一种**高级描述语言**，对人友好但对机器来说不够灵活。BuildKit 在执行 Dockerfile 之前，会先把它编译成 **LLB**，一种基于 protobuf 的图数据结构。

LLB 把整个构建过程表示为一个 DAG，图中每个节点是一个操作（Op），每条边是数据依赖。节点类型包括：

- **Source Op**：从外部获取数据（比如拉取镜像、从 Git 仓库获取文件）
- **Exec Op**：执行一条命令（对应 RUN）
- **File Op**：文件操作（对应 COPY、ADD）
- **Image Op**：引用已有镜像（对应 FROM）

LLB 本身是与 Dockerfile 无关的——任何能生成合法 LLB 的工具都可以作为 BuildKit 的"前端"。这就是为什么除了 Dockerfile，还有 Buildpacks、Nixpacks 等不同的构建前端，它们都能利用 BuildKit 的执行引擎和缓存机制。

并行构建的实现原理很简单：BuildKit 分析 DAG，找出没有依赖关系（或者说没有共同祖先节点）的操作，把它们分配给不同的 worker 并发执行。

```
多阶段构建中的并行示例：

FROM golang AS builder-a    FROM python AS builder-b
RUN go build ...            RUN pip install ...
     \                            /
      \                          /
       FROM alpine AS final
       COPY --from=builder-a ...
       COPY --from=builder-b ...
```

在 Legacy Builder 中，builder-a 和 builder-b 必须依次执行。在 BuildKit 中，它们可以并行，最终阶段等待两者都完成后再继续。

## 小结

理解 Docker 镜像构建的底层原理，可以归纳为以下几个核心认知：

- **Legacy Builder 用临时容器模型**，每条指令都经历"创建容器 → 执行 → commit → 删除"，BuildKit 用 LLB DAG 替代这个流程，支持并行构建和更精细的控制
- **构建上下文在客户端被打包**，`.dockerignore` 在打包前过滤，BuildKit 进一步支持按需传输，减少不必要的数据传输
- **层的身份由三个概念分别描述**：DiffID（内容哈希）、ChainID（层在栈中的位置标识）、CacheID（磁盘存储目录名），理解它们的区别有助于理解缓存行为
- **RUN 缓存基于指令字符串 + 父层 ID，COPY 缓存还额外校验文件内容哈希**，缓存失效向后传播
- **BuildKit 的 Mount 类型** 解决了持久化缓存、密钥安全注入、SSH 转发等实际工程问题，是比多阶段构建更细粒度的优化手段

---

## 常见问题

### Q1：为什么我只改了一行 Python 代码，`pip install` 也要重新执行？

原因是缓存失效的链式传播。如果你的 Dockerfile 把 `COPY . /app` 写在 `RUN pip install` 之前，那么改动任何源文件都会导致 COPY 指令的缓存失效，进而让 pip install 也失效。

正确的写法是把依赖文件单独复制：

```dockerfile
COPY requirements.txt /app/
RUN pip install -r /app/requirements.txt
COPY . /app     # 改代码只会影响这行以后的指令
```

这样只要 `requirements.txt` 没变，pip install 就能命中缓存。

### Q2：`docker build --no-cache` 和 BuildKit 的 cache mount 是两码事吗？

是的，完全不同。`--no-cache` 禁用的是**层缓存**，也就是说每条 RUN/COPY 指令都强制重新执行，不使用之前保存的层。

而 `--mount=type=cache` 是**构建工具的本地缓存目录**（比如 pip 的包缓存、go 的模块缓存），这个目录的内容不会成为镜像层的一部分，而是以一个独立的 volume 形式在构建之间持久保留。即使你用了 `--no-cache`，cache mount 目录中的内容依然存在，构建工具依然可以利用这些缓存来加速。

### Q3：Secret Mount 和直接在 ARG 或 ENV 里传密钥有什么本质区别？

使用 ARG 或 ENV 传入密钥，这些值会被记录在镜像的 `config.json`（镜像配置）中，任何能访问该镜像的人都能通过 `docker inspect` 读出来。更严重的是，ENV 的值会在每个层的元数据中留下痕迹。

Secret Mount 的密钥**只在构建时临时挂载为文件**，不写入任何层，也不记录在镜像配置中。构建完成后，镜像里没有任何密钥的残留。这是在构建期间使用私有 npm registry 凭证、pip 私有源 token、Git 访问令牌等场景的正确姿势。

### Q4：ChainID 和 DiffID 在实际排查问题时有什么用？

当你想知道两个镜像是否共享某些层时，可以通过比较 DiffID 来判断。执行 `docker image inspect <image>` 看 `RootFS.Layers`，这里列出的就是 DiffID 列表。如果两个镜像的前几个 DiffID 相同，说明它们共享了底层。

ChainID 则用于定位本地存储中的层。在 `/var/lib/docker/image/overlay2/layerdb/sha256/` 目录下，目录名就是 ChainID。进入对应目录，`cache-id` 文件里保存的就是该层对应的 CacheID（即 `/var/lib/docker/overlay2/` 下的目录名）。这条查找链在排查"为什么这个层占了这么多空间"或"为什么层没有共享"时非常有用。

### Q5：如何判断我的项目是否需要从 Legacy Builder 切换到 BuildKit？

如果你的项目符合以下任一条件，切换到 BuildKit 会有显著收益：

- **多阶段构建且各阶段相对独立**：BuildKit 并行执行独立阶段，构建时间可以减半甚至更多
- **构建过程需要下载大量依赖**：使用 `--mount=type=cache` 把包管理器的缓存持久化，避免每次重复下载
- **构建过程需要访问私有资源**：使用 `--mount=type=secret` 或 `--mount=type=ssh`，不再需要用多阶段构建来"清理"凭证的残留
- **在 CI 环境中构建**：BuildKit 支持缓存导出/导入，可以把本地缓存推送到 Registry，让 CI 机器也能复用

Docker 23.0 以上版本默认启用 BuildKit，通常不需要额外配置。如果是旧版本，可以设置环境变量 `export DOCKER_BUILDKIT=1` 来启用。

## 参考资源

- [Docker BuildKit 官方文档](https://docs.docker.com/build/buildkit/)
- [Dockerfile 最佳实践](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [BuildKit 缓存管理](https://docs.docker.com/build/cache/)

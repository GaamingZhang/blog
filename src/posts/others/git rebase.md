---
date: 2026-01-12
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 运维
tag:
  - 运维
---

# Git rebase 命令详解

## 1. 引言

Git rebase 是 Git 版本控制系统中一个强大的分支管理命令，用于将一个分支的修改集成到另一个分支。与传统的 `git merge` 命令相比，rebase 可以创建更清晰、线性的提交历史，使项目的版本演进更加直观。

本文将详细介绍 Git rebase 的基本概念、工作原理、使用方法、与 merge 的区别、最佳实践以及常见问题，帮助读者全面掌握这一重要的 Git 命令。

## 2. Git rebase 的核心概念

### 2.1 什么是 Git rebase？

Git rebase（变基）是一种将一个分支的提交历史“移动”到另一个分支末尾的操作。它的核心思想是：**将当前分支的所有修改提交，重新基于目标分支的最新提交进行应用**。

### 2.2 rebase 的工作原理

rebase 操作的基本流程如下：

1. 找到当前分支与目标分支的**共同祖先提交**
2. 保存当前分支上从共同祖先到当前 HEAD 的所有提交（形成一个临时补丁集）
3. 将当前分支重置到目标分支的最新提交
4. 按照原顺序依次将保存的补丁集应用到当前分支上
5. 生成新的提交历史（每个提交会有新的 SHA-1 哈希值）

### 2.3 rebase 的目标

rebase 的主要目标是创建一个**干净、线性的提交历史**，避免了 merge 操作产生的“合并提交”。这使得代码审查和错误定位变得更加容易，同时也能更清晰地了解项目的演进过程。

## 3. Git rebase 的基本用法

### 3.1 基本命令格式

```bash
git rebase <目标分支>
```

### 3.2 典型使用场景

#### 3.2.1 将特性分支合并到主分支

假设我们有一个 `feature` 分支和一个 `main` 分支：

```bash
# 切换到 feature 分支
git checkout feature

# 将 feature 分支的修改重新基于 main 分支
git rebase main

# 切换回 main 分支
git checkout main

# 快速合并 feature 分支（此时应该是快进合并）
git merge feature
```

#### 3.2.2 交互式 rebase

交互式 rebase 允许我们在 rebase 过程中修改、删除、合并或重新排序提交：

```bash
git rebase -i <目标分支>
```

执行后会打开一个文本编辑器，显示当前分支的提交历史，我们可以对这些提交进行操作：

```
pick 1a2b3c4 第一次提交
pick 5d6e7f8 第二次提交
pick 9g0h1i2 第三次提交

# 命令说明：
# p, pick <提交> = 使用提交
# r, reword <提交> = 使用提交，但修改提交说明
# e, edit <提交> = 使用提交，但暂停以便修改提交
# s, squash <提交> = 使用提交，但将其与前一个提交合并
# f, fixup <提交> = 类似 "squash"，但丢弃提交说明
# x, exec <命令> = 在提交前执行命令
# d, drop <提交> = 删除提交
```

## 4. Git rebase 的高级用法

### 4.1 限定 rebase 范围

可以指定只 rebase 特定范围的提交：

```bash
git rebase <目标分支> <起始提交>..<结束提交>
```

### 4.2 自动解决冲突

```bash
git rebase -i --autosquash <目标分支>
```

`--autosquash` 选项会自动识别带有 `fixup!` 或 `squash!` 前缀的提交，并将它们与对应的提交合并。

### 4.3 强制推送到远程分支

由于 rebase 会改变提交历史，推送到远程分支时需要使用 `--force` 或 `--force-with-lease` 选项：

```bash
git push origin <分支名> --force-with-lease
```

**注意**：`--force-with-lease` 比 `--force` 更安全，它只会在远程分支与本地期望的状态一致时才进行强制推送。

### 4.4 在特定提交处暂停 rebase

使用 `edit` 命令可以在特定提交处暂停 rebase，以便修改代码：

```bash
# 1. 启动交互式 rebase
git rebase -i <目标分支>

# 2. 将需要修改的提交前面的 "pick" 改为 "edit"

# 3. 当 rebase 暂停时，修改代码
# ...

# 4. 提交修改
git commit --amend

# 5. 继续 rebase
git rebase --continue
```

## 5. Git rebase 与 Git merge 的区别

| 特性 | Git rebase | Git merge |
|------|------------|-----------|
| 提交历史 | 线性、清晰 | 保留分支结构，可能有大量合并提交 |
| 提交哈希 | 重新生成 | 保留原有提交哈希 |
| 冲突处理 | 逐个提交处理冲突 | 一次处理所有冲突 |
| 适用场景 | 个人分支或本地分支 | 公共分支或团队协作 |
| 风险 | 高（改变历史） | 低（不改变历史） |

## 6. Git rebase 的最佳实践

### 6.1 只在本地分支使用 rebase

**永远不要对已经推送到公共仓库的分支进行 rebase**，这会破坏其他开发者的工作环境。

### 6.2 定期使用 rebase 保持分支同步

在开发特性分支时，定期将主分支的更新 rebase 到特性分支，可以减少最终合并时的冲突：

```bash
git checkout feature
git fetch origin
git rebase origin/main
```

### 6.3 使用交互式 rebase 整理提交历史

在将特性分支合并到主分支之前，使用交互式 rebase 可以整理提交历史，使其更加清晰：

- 合并相关的提交
- 删除不必要的提交
- 修改提交说明使其更具描述性

### 6.4 处理冲突时要谨慎

在 rebase 过程中遇到冲突时，要仔细检查并解决冲突。解决后使用：

```bash
git add <冲突文件>
git rebase --continue
```

如果需要中止 rebase，可以使用：

```bash
git rebase --abort
```

### 6.5 使用 `--force-with-lease` 而不是 `--force`

当需要强制推送到远程分支时，使用 `--force-with-lease` 更加安全：

```bash
git push origin <分支名> --force-with-lease
```

## 7. 常见问题

### 7.1 Git rebase 会丢失提交吗？

在正常情况下，Git rebase 不会丢失提交。它只是将提交重新应用到新的基础上。但如果在交互式 rebase 中使用了 `drop` 命令，对应的提交将被删除。

如果不小心丢失了提交，可以使用 `git reflog` 命令查看所有操作历史，并恢复丢失的提交：

```bash
git reflog
git checkout <丢失的提交哈希>
```

### 7.2 什么时候应该使用 rebase 而不是 merge？

- 当你想要保持提交历史线性时
- 当你在本地开发特性分支，尚未推送到公共仓库时
- 当你想要整理提交历史，使其更加清晰时

### 7.3 如何撤销一个已经完成的 rebase？

可以使用 `git reflog` 找到 rebase 之前的分支状态，然后重置分支：

```bash
git reflog
git reset --hard <rebase 之前的 HEAD 哈希>
```

### 7.4 rebase 过程中遇到冲突怎么办？

1. 查看冲突文件：`git status`
2. 手动编辑文件，解决冲突
3. 标记冲突已解决：`git add <冲突文件>`
4. 继续 rebase：`git rebase --continue`
5. 如果需要中止 rebase：`git rebase --abort`

### 7.5 为什么 rebase 后提交哈希会改变？

Git 的提交哈希是基于提交内容、父提交、作者信息、提交时间等计算的。当使用 rebase 时，提交的父提交发生了变化，因此会生成新的提交哈希。

## 8. 总结

Git rebase 是一个强大的分支管理命令，可以帮助我们创建清晰、线性的提交历史。它的核心思想是将一个分支的修改重新基于另一个分支的最新提交进行应用。

虽然 rebase 功能强大，但也存在一定风险，特别是当对公共分支使用时。因此，我们需要遵循最佳实践，只在本地分支使用 rebase，并在必要时使用 `--force-with-lease` 进行强制推送。

通过合理使用 Git rebase，我们可以保持项目的提交历史清晰、易读，提高团队协作效率和代码质量。
---
date: 2025-07-01
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - 操作系统
tag:
  - 操作系统
---

# sed：流式文本编辑器

## sed 是什么

**sed**（Stream Editor）是一个流式文本编辑器，专门用于非交互式的文本处理。它逐行读取输入，根据指定的规则进行编辑，然后输出结果。

### 为什么需要 sed

**批量文本处理**：
- 修改配置文件（替换端口号、路径）
- 清洗日志文件（删除无用行、提取关键信息）
- 数据转换（格式化、规范化）

**自动化脚本**：
- 在脚本中自动修改文件
- 管道中处理文本流
- 无需手动打开编辑器

**快速实验**：
- 测试正则表达式
- 快速查看文件的特定行
- 预览修改效果（不改变原文件）

### sed vs 其他工具

| 工具   | 用途               | 特点                   |
| ------ | ------------------ | ---------------------- |
| sed    | 文本替换、删除行   | 流式处理，逐行编辑     |
| awk    | 文本分析、数据提取 | 面向列，适合结构化数据 |
| grep   | 文本搜索           | 只查找，不修改         |
| vim/vi | 交互式编辑         | 需要手动操作           |

---

## sed 的工作原理

### 流式处理模式

sed 的处理流程：

```
输入文件/流
    ↓
逐行读入 → 模式空间（Pattern Space）
    ↓
执行命令 → 匹配 + 编辑
    ↓
输出结果 → 标准输出
    ↓
读取下一行（循环）
```

**关键概念**：
- **模式空间**：临时缓冲区，存储当前处理的行
- **逐行处理**：一次处理一行，内存占用小
- **默认输出**：处理后自动打印到标准输出

### 基本语法

```bash
sed [选项] '命令' 文件
```

**常用选项**：
- `-n`：静默模式，不自动打印（需配合 `p` 命令）
- `-i`：原地编辑文件（直接修改文件）
- `-e`：执行多个命令
- `-r` 或 `-E`：使用扩展正则表达式

---

## 核心功能

### 1. 文本替换（最常用）

**基本替换**：
```bash
sed 's/old/new/' file.txt
```
- `s`：substitute，替换命令
- 只替换每行的第一个匹配

**全局替换**：
```bash
sed 's/old/new/g' file.txt
```
- `g`：global，替换每行的所有匹配

**忽略大小写**：
```bash
sed 's/old/new/i' file.txt
```
- `i`：ignore case

**直接修改文件**：
```bash
sed -i 's/old/new/g' file.txt
```
- `-i`：in-place，直接修改原文件

### 2. 删除行

**删除特定行号**：
```bash
sed '3d' file.txt          # 删除第 3 行
sed '2,5d' file.txt        # 删除 2-5 行
sed '$d' file.txt          # 删除最后一行
```

**删除匹配的行**：
```bash
sed '/pattern/d' file.txt   # 删除包含 pattern 的行
sed '/^$/d' file.txt        # 删除空行
sed '/^#/d' file.txt        # 删除注释行
```

**删除范围**：
```bash
sed '/start/,/end/d' file.txt   # 删除从 start 到 end 的区间
```

### 3. 打印特定行

**打印行号范围**：
```bash
sed -n '1,5p' file.txt      # 打印 1-5 行
sed -n '10p' file.txt       # 只打印第 10 行
```
- `-n` 关闭自动打印，`p` 显式打印

**打印匹配的行**：
```bash
sed -n '/pattern/p' file.txt    # 打印包含 pattern 的行
```
- 类似 grep，但可以配合其他 sed 命令

### 4. 插入和追加

**追加行**（在匹配行后面添加）：
```bash
sed '/pattern/a\new line' file.txt
```

**插入行**（在匹配行前面添加）：
```bash
sed '/pattern/i\new line' file.txt
```

**替换整行**：
```bash
sed '/pattern/c\new line' file.txt
```

---

## 常见使用场景

### 场景 1：修改配置文件

**替换端口号**：
```bash
sed -i 's/port=8080/port=9090/g' config.ini
```

**修改路径**：
```bash
sed -i 's|/old/path|/new/path|g' settings.conf
```
- 使用 `|` 作为分隔符，避免路径中的 `/` 冲突

**注释掉某一行**：
```bash
sed -i '/debug_mode/s/^/#/' config.ini
```

**取消注释**：
```bash
sed -i '/port/s/^#//' config.ini
```

### 场景 2：清洗日志文件

**删除空行和注释**：
```bash
sed '/^$/d; /^#/d' log.txt
```

**只保留错误日志**：
```bash
sed -n '/ERROR/p' app.log
```

**删除时间戳**：
```bash
sed 's/^[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\} [0-9]\{2\}:[0-9]\{2\}:[0-9]\{2\} //' log.txt
```

### 场景 3：格式化数据

**去除行首空格**：
```bash
sed 's/^[[:space:]]*//' file.txt
```

**去除行尾空格**：
```bash
sed 's/[[:space:]]*$//' file.txt
```

**统一替换多个空格为一个**：
```bash
sed 's/  */ /g' file.txt
```

### 场景 4：提取信息

**提取 IP 地址**：
```bash
sed -n 's/.*\([0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\).*/\1/p' access.log
```

**提取文件名（从路径）**：
```bash
echo "/path/to/file.txt" | sed 's|.*/||'
```

---

## 实用技巧

### 多个命令

**使用分号**：
```bash
sed 's/old/new/g; s/foo/bar/g' file.txt
```

**使用 -e**：
```bash
sed -e 's/old/new/g' -e 's/foo/bar/g' file.txt
```

### 地址范围

**指定行号范围**：
```bash
sed '10,20s/old/new/g' file.txt   # 只在 10-20 行替换
```

**匹配范围**：
```bash
sed '/start/,/end/s/old/new/g' file.txt   # 在 start 到 end 区间替换
```

### 取反操作

**对不匹配的行执行操作**：
```bash
sed '/pattern/!d' file.txt    # 删除不包含 pattern 的行（保留匹配行）
```

### 备份原文件

**修改文件前自动备份**：
```bash
sed -i.bak 's/old/new/g' file.txt
```
- 原文件备份为 `file.txt.bak`

---

## 正则表达式基础

### 常用元字符

**字符匹配**：
- `.`：任意单个字符
- `*`：前面字符重复 0 次或多次
- `+`：前面字符重复 1 次或多次（扩展正则）
- `?`：前面字符重复 0 次或 1 次（扩展正则）

**位置匹配**：
- `^`：行首
- `$`：行尾

**字符集**：
- `[abc]`：匹配 a、b 或 c
- `[^abc]`：不匹配 a、b、c
- `[a-z]`：匹配小写字母
- `[0-9]`：匹配数字

**预定义字符类**：
- `[[:digit:]]`：数字
- `[[:alpha:]]`：字母
- `[[:space:]]`：空白字符

### 捕获组

**捕获并引用**：
```bash
sed 's/\(pattern\)/[\1]/' file.txt
```
- `\(` 和 `\)`：捕获组
- `\1`：引用第一个捕获组

**示例**：
```bash
echo "hello world" | sed 's/\(hello\) \(world\)/\2 \1/'
# 输出：world hello
```

---

## 常见问题

### sed 和 awk 如何选择？

**使用 sed**：
- 简单的文本替换
- 删除/插入行
- 基于行号或模式的编辑

**使用 awk**：
- 处理列数据（CSV、TSV）
- 复杂的数据分析
- 需要计算和统计

**经验法则**：
- 只改不算 → sed
- 又改又算 → awk

### 为什么需要转义？

sed 使用基本正则表达式（BRE），某些字符需要转义：
- `\(` `\)`：捕获组
- `\{` `\}`：重复次数
- `\+`：一次或多次（基本正则）

使用扩展正则（`-E`）可以减少转义：
```bash
sed -E 's/(pattern)+/replacement/' file.txt
```

### 如何测试不修改文件？

**方法 1**：不使用 `-i`
```bash
sed 's/old/new/g' file.txt     # 只输出，不修改文件
```

**方法 2**：先备份
```bash
sed -i.bak 's/old/new/g' file.txt
```

**方法 3**：使用管道测试
```bash
cat file.txt | sed 's/old/new/g' | head
```

---

## 核心要点

**sed 的本质**：流式文本编辑器，用于批量非交互式文本处理。

**核心概念**：
- **流式处理**：逐行读取、编辑、输出
- **模式空间**：临时缓冲区，存储当前行
- **命令模式**：`s`（替换）、`d`（删除）、`p`（打印）、`a/i/c`（插入）

**典型场景**：
- **配置文件修改**：批量替换路径、端口、参数
- **日志清洗**：删除无用行、提取关键信息
- **数据格式化**：去空格、统一格式
- **文本提取**：基于模式提取内容

**使用技巧**：
- `-n` + `p`：只打印匹配行（类似 grep）
- `-i`：直接修改文件（慎用，建议先备份）
- `/pattern/d`：删除匹配行
- `s/old/new/g`：全局替换
- 使用 `|` 或其他分隔符避免 `/` 冲突

**最佳实践**:
- 先不加 `-i` 测试效果
- 重要文件先备份（`-i.bak`）
- 复杂操作考虑使用 awk 或脚本语言
- 配合管道使用（grep + sed + awk）

## 参考资源

- [GNU sed 手册](https://www.gnu.org/software/sed/manual/sed.html)
- [Linux sed 命令手册](https://man7.org/linux/man-pages/man1/sed.1.html)
- [正则表达式教程](https://www.gnu.org/software/sed/manual/html_node/Regular-Expressions.html)

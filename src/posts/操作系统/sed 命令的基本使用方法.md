---
date: 2025-12-22
author: Gaaming Zhang
category:
  - 操作系统
tag:
  - 操作系统
  - 已完工
---

# sed 命令的基本使用方法

## 面试口述要点（约 1 分钟）
sed（Stream Editor）是**流式文本编辑器**，用于非交互式批量文本处理，广泛应用于日志分析、配置文件修改、数据清洗等场景。

**核心工作原理**：
1. **流处理模式**：逐行读取输入文本到**模式空间**（Pattern Space）
2. **模式匹配**：对模式空间中的内容执行匹配
3. **编辑操作**：根据命令对匹配内容执行替换、删除、插入等操作
4. **结果输出**：默认将模式空间内容输出到标准输出
5. **循环处理**：重复上述步骤直到处理完所有输入行

**基础语法**：`sed [选项] '命令' 文件`
- 常用选项：`-n` 静默模式（不自动打印模式空间）、`-i` 原地编辑文件、`-e` 多条命令、`-r/-E` 扩展正则、`-f` 从文件读取命令

**核心命令**：
- `s/pattern/replacement/flags`：替换，flags 包括 `g` 全局、`p` 打印、`数字` 指定第几次、`i` 忽略大小写
- `p`：打印匹配行（需 `-n` 配合）
- `d`：删除匹配行（清空模式空间并跳过后续命令）
- `a\text`：在匹配行后追加、`i\text`：在匹配行前插入
- `c\text`：替换整行
- `w file`：将模式空间内容写入文件
- `!`：取反操作（对不匹配的行执行命令）
- `h/H`：将模式空间内容复制/追加到**保持空间**（Hold Space）
- `g/G`：将保持空间内容复制/追加到模式空间
- `x`：交换模式空间与保持空间内容
- `N`：读取下一行到模式空间（多行处理）
- `D`：删除模式空间中第一个换行符前的内容
- `P`：打印模式空间中第一个换行符前的内容

**地址范围**：指定操作范围
- `3d`：删除第 3 行
- `/pattern/d`：删除匹配行
- `2,5d`：删除 2-5 行
- `/start/,/end/d`：删除从 start 到 end 的区间
- `1~2d`：删除奇数行（步长）

**高级特性详解**：

1. **保持空间（Hold Space）机制**：
   - sed有两个内存缓冲区：**模式空间**（临时处理当前行）和**保持空间**（长期存储临时数据）
   - 用于实现复杂的多行处理，如合并行、逆序文件等
   - 核心命令组合：
     ```bash
     # 逆序输出文件（类似 tac）
     sed '1!G;h;$!d' file.txt
     
     # 合并连续两行
     sed 'N;s/\n/ /' file.txt
     
     # 只显示匹配模式的前后两行
     sed -n '/pattern/{=;x;p;x;p;x;n;p}' file.txt
     ```

2. **分支与测试命令**：
   - 实现条件分支和流程控制，类似编程语言的if-else和goto
   - `t label`：如果上一次替换成功，则跳转到label
   - `b label`：无条件跳转到label
   - `:label`：定义标签位置
   - 示例：
     ```bash
     # 如果包含error则替换为ERROR并停止后续处理
     sed -E 's/error/ERROR/;t; s/warning/WARNING/' log.txt
     
     # 批量修改不同的配置项
     sed -E '/^server_port:/b port; /^workers:/b workers; b
     :port
     s/[0-9]+/8080/
     b
     :workers
     s/[0-9]+/4/' config.yaml
     ```

3. **多行处理技巧**：
   - `N`：读取下一行到模式空间，形成多行模式
   - `D`：删除多行模式中第一个换行符前的内容
   - `P`：打印多行模式中第一个换行符前的内容
   - 示例：
     ```bash
     # 删除包含error的行及其下一行
     sed '/error/{N;d}' file.txt
     
     # 将跨行的日志合并为单行
     sed -E '/^[0-9]{4}-[0-9]{2}-[0-9]{2}/{x;p;x;}; 1!{H;d}; ${x;p}' log.txt
     ```

**典型应用场景**：

1. **日志分析与过滤**
```bash
# 提取错误日志
cat error.log | sed -n '/ERROR/p'

# 过滤出包含特定 IP 的访问日志
sed -n '/192\.168\.1\.1/p' access.log

# 将错误日志转换为标准格式
sed 's/\[ERROR\] \(.*\)/ERROR: \1/g' app.log
```

2. **配置文件批量修改**
```bash
# 批量修改服务器端口
sed -i 's/^server_port=.*/server_port=8080/' *.conf

# 启用所有注释的 debug 选项
sed -i 's/^#debug=/debug=/' config.ini

# 修改 Nginx 监听端口
sed -i '/listen 80;/s/80/443/' nginx.conf
```

3. **数据清洗与格式化**
```bash
# 移除 CSV 文件中的引号
sed 's/"//g' data.csv

# 替换日期格式（YYYY-MM-DD → DD/MM/YYYY）
sed -E 's/(20[0-9]{2})-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])/\3\/\2\/\1/g' dates.txt

# 将多行文本合并为单行，用逗号分隔
sed -z 's/\n/,/g' lines.txt
```

4. **文本替换与删除**
```bash
# 全局替换
sed 's/old/new/g' file.txt

# 原地修改文件（带备份）
sed -i.bak 's/old/new/g' file.txt
# macOS: sed -i '' 's/old/new/g' file.txt

# 删除空行
sed '/^$/d' file.txt

# 删除注释行
sed '/^#/d' file.txt
```

5. **特定行操作**
```bash
# 打印第 10-20 行
sed -n '10,20p' file.txt

# 在第 5 行后插入新内容
sed '5a\新内容' file.txt

# 在匹配行前插入注释
sed '/password/s/^/#/' config.txt
```

## 相关高频面试题与简答

**Q1: sed 替换时如何处理特殊字符（如 `/` 或 `&`）？**
```bash
# 1. 更换分隔符（推荐方法，避免转义）
sed 's|/path/to/file|/new/path|g' file.txt
sed 's#old#new#g' file.txt

# 2. 使用转义字符（传统方法）
sed 's/\/path\/to\/file/\/new\/path/g' file.txt

# 3. 处理 & 符号（特殊含义：代表匹配的整体）
sed 's/error/[&]/g' file.txt  # error -> [error]
sed 's/\([0-9]+\)/&秒/' file.txt  # 捕获组+& 使用

# 4. 处理其他特殊字符（$ ^ . * [ ] ( )）
sed 's/\$/USD/g' price.txt       # 替换 $ 符号
sed 's/\^/开始/g' file.txt        # 替换 ^ 符号
sed 's/\./点/g' version.txt       # 替换 . 符号
sed 's/\*/星号/g' pattern.txt     # 替换 * 符号

# 5. 在脚本中使用变量时的处理
new_path="/data/new"
sed "s|/old/path|$new_path|g" file.txt  # 使用双引号

# 6. 处理包含多个特殊字符的复杂路径
sed 's@/etc/nginx/sites-enabled/.*\.conf@/etc/nginx/sites-enabled/default.conf@g' nginx.conf
```

**Q2: sed 如何只替换每行第 N 次匹配？**
```bash
# 1. 替换每行第 N 次出现（基本用法）
sed 's/foo/bar/2' file.txt          # 替换每行第2次出现的foo

# 2. 替换第 N 次及之后所有（g修饰符）
sed 's/foo/bar/2g' file.txt         # 替换每行第2次及之后所有的foo

# 3. 结合行号和次数限制
sed '3s/foo/bar/2' file.txt         # 只替换第3行的第2次出现

sed '5,10s/foo/bar/3g' file.txt     # 替换5-10行的第3次及之后所有

# 4. 实际应用：修改CSV文件中的特定字段
# 假设CSV格式：name,age,email,phone
sed 's/,/;/2' data.csv              # 将第2个逗号替换为分号

sed 's/,/;/3g' data.csv             # 将第3个及之后的逗号替换为分号

# 5. 使用正则匹配并限制替换次数
# 替换每行第3个数字

sed -E 's/[0-9]+/X/3' text.txt      # 替换每行第3个数字为X

# 6. 结合保持空间实现更复杂的替换
# 只替换包含特定模式行的第2次匹配
sed '/pattern/s/foo/bar/2' file.txt
```

**Q3: 如何用 sed 实现配置文件参数修改？**
```bash
# 1. 修改 key=value 格式（最基本）
sed -i 's/^server_port=.*/server_port=8080/' config.ini

# 2. 修改带缩进的配置（YAML/TOML）
sed -i 's/^  workers:.*/  workers: 4/' config.yaml

# 3. 仅修改已存在的配置（避免误加新行）
sed -i '/^#.*server_port/! s/server_port=.*/server_port=8080/' config.ini

# 4. 修改多行配置块（如 Nginx server 块）
sed -i '/server_name example.com;/,/}/ s/listen 80;/listen 443 ssl;/' nginx.conf

# 5. 批量修改多个配置项
sed -i -E 's/^user=.*/user=app/; s/^log_level=.*/log_level=info/' app.ini

# 6. 保留注释并修改配置
# 先删除注释行的注释符号，再修改
# 注意：仅当配置项在注释和非注释中都存在时使用
sed -i -E 's/^#(server_port=).*/\18080/; s/^server_port=.*/server_port=8080/' config.ini
```

**Q4: sed 如何删除文件中的某些行？**
```bash
# 1. 删除空行
sed '/^$/d' file.txt

# 2. 删除注释行（#开头）
sed '/^#/d' file.txt

# 3. 删除注释与空行（组合使用）
sed '/^#/d; /^$/d' file.txt

# 4. 删除特定行范围
sed '5,10d' file.txt          # 删除第5-10行
sed '1,3d; 8,10d' file.txt     # 删除多个不连续范围

# 5. 删除包含特定模式的行
sed '/error/d' log.txt         # 删除包含error的行
sed '/^DEBUG/d' log.txt        # 删除以DEBUG开头的行
sed '/\.tmp$/d' filelist.txt   # 删除以.tmp结尾的行

# 6. 删除不包含特定模式的行（只保留匹配行）
sed '/keep/!d' file.txt       # 删除不包含keep的行
# 等价于 sed -n '/keep/p'（更常用）

# 7. 删除重复行（相邻重复）
sed '$!N; /^\(.*\)\n\1$/!P; D' file.txt
# 注意：完全重复行，不考虑顺序

# 8. 删除文件最后N行
sed -i '$d' file.txt          # 删除最后一行
sed -i ':a;$d;N;2,3ba' file.txt # 删除最后3行（复杂实现）
```

**Q5: sed 与 awk 如何选择？**
- sed：面向行的简单文本替换、删除、插入，模式匹配+单一操作。
- awk：面向列（字段）的复杂文本处理、统计、格式化输出，支持变量、函数、条件。
- 典型分工：sed 改配置文件/日志过滤，awk 提取特定列/计算/报表。

**Q6: macOS 的 sed 与 Linux sed 有什么区别？**
```bash
# macOS（BSD sed）必须提供备份扩展名
sed -i.bak 's/old/new/g' file.txt  # 生成 file.txt.bak
sed -i '' 's/old/new/g' file.txt   # 不备份需空字符串

# Linux（GNU sed）
sed -i 's/old/new/g' file.txt      # 直接修改

# 扩展正则
# macOS: -E
# Linux: -r 或 -E（新版）

# 解决跨平台：安装 GNU sed
# macOS: brew install gnu-sed，使用 gsed
```

**Q7: 如何使用 sed 实现文件内容的逆序输出？**
```bash
# 方法1：使用保持空间（最经典）
sed '1!G;h;$!d' file.txt
# 1!G: 非第一行时，将保持空间内容追加到模式空间
# h: 将模式空间内容复制到保持空间
# $!d: 非最后一行时，删除模式空间内容（不输出）

# 方法2：使用 tac 命令（更简洁）
tac file.txt
```

**Q8: 如何使用 sed 合并连续的空行为单行？**
```bash
# 合并连续空行
# /^$/!b：非空行直接输出
# n：读取下一行到模式空间
# /^$/!b：如果下一行不是空行，直接输出
# d：删除多余的空行
sed '/^$/!b;n;/^$/!b;d' file.txt

# 更简洁的方式（使用扩展正则）
sed -E '/^$/{N;/\n$/D}' file.txt
```

**Q9: 如何使用 sed 实现多行日志的合并？**
```bash
# 合并以时间戳开头的多行日志（常见于 Java 堆栈跟踪）
# 假设日志格式：2023-10-01 10:00:00 ERROR ...
sed -E '/^[0-9]{4}-[0-9]{2}-[0-9]{2}/{x;p;x;}; 1!{H;d}; ${x;p}' log.txt
# /^[0-9]{4}/：匹配时间戳行
# x;p;x：交换到保持空间，打印，再交换回来
# 1!{H;d}：非第一行，追加到保持空间并删除当前行
# ${x;p}：最后一行，交换并打印所有内容
```

**Q10: 如何使用 sed 实现条件替换（根据不同条件替换不同内容）？**
```bash
# 使用分支命令实现条件替换
# 如果包含 ERROR 替换为 [ERROR]，WARNING 替换为 [WARNING]，其他添加 [INFO]
sed -E '/ERROR/{s/ERROR/[ERROR]/;b}; /WARNING/{s/WARNING/[WARNING]/;b}; s/^/[INFO]/' log.txt

# 使用测试命令实现
# 如果替换成功则跳过后续替换
sed -E 's/ERROR/[ERROR]/;t; s/WARNING/[WARNING]/;t; s/^/[INFO]/' log.txt
```

## 实用技巧与注意事项

**生产环境最佳实践**：

1. **安全编辑策略**：
   - 先测试不加 `-i`，确认输出正确后再原地修改：`sed 's/old/new/g' file.txt | head -20`
   - 用 `-i.bak` 保留备份（推荐）：`sed -i.bak 's/old/new/g' file.txt`
   - 批量处理前先查看影响范围：`grep -r "old" --include="*.conf" . | wc -l`

2. **性能优化**：
   - 对于大文件，优先使用 `-E` 扩展正则（更高效）
   - 避免多次扫描同一文件：将多条命令合并为一个调用
     ```bash
     # 不推荐：多次扫描文件
     sed -i 's/old/new/g' file.txt
     sed -i 's/foo/bar/g' file.txt
     
     # 推荐：单次扫描完成所有替换
     sed -i 's/old/new/g; s/foo/bar/g' file.txt
     ```
   - 使用 `--posix` 参数确保可移植性（跨平台脚本）

3. **调试技巧**：
   - 使用 `-n` + `l` 命令查看不可见字符：`sed -n 'l' file.txt`
   - 添加 `p` 命令查看中间结果：`sed -n 's/old/new/p' file.txt`
   - 用 `set -x` 调试 sed 命令行：`set -x; sed 's/old/new/g' file.txt; set +x`

4. **复杂场景处理**：
   - 处理包含换行符的多行文本：使用 `-z` 参数（GNU sed）
     ```bash
     # 将文件内容作为单行处理
     sed -z 's/\n/ /g' file.txt
     ```
   - 从文件读取命令：对于复杂脚本，使用 `-f script.sed`
     ```bash
     # script.sed 内容
     s/old/new/g
     s/foo/bar/g
     /pattern/d
     
     # 执行
     sed -i -f script.sed file.txt
     ```

5. **跨平台兼容性**：
   - 避免使用 GNU sed 特有功能（如 `-z`, `-r` 在 macOS 上可能不可用）
   - 使用 `gsed` 在 macOS 上模拟 GNU sed 行为
   - 路径处理统一使用 `/` 或更换分隔符，避免平台差异

6. **大文件处理优化**：
   - 对于超大型文件（GB 级别），优先使用 `-n` 减少不必要的输出
   - 使用 `head`/`tail` 先查看部分内容，再执行完整命令
     ```bash
     # 先测试前 1000 行
     head -n 1000 large.log | sed 's/old/new/g' > test_result.txt
     ```
   - 避免使用保持空间（hold space）处理大文件，会增加内存开销
   - 考虑使用 `grep -l` 先定位需要修改的文件，再使用 sed 处理
     ```bash
     grep -l "old" *.log | xargs sed -i 's/old/new/g'
     ```

**常见坑与解决方案**：

- **坑1**：忘记 `-n` 导致重复输出（`p` 命令需配合 `-n`）
  ```bash
  # 错误：会输出所有行 + 匹配行
  sed '/pattern/p' file.txt
  
  # 正确：只输出匹配行
  sed -n '/pattern/p' file.txt
  ```

- **坑2**：macOS `-i` 后缺少备份扩展名
  ```bash
  # Linux 正确
  sed -i 's/old/new/g' file.txt
  
  # macOS 正确（必须提供备份扩展名）
  sed -i.bak 's/old/new/g' file.txt  # 保留备份
  sed -i '' 's/old/new/g' file.txt   # 不保留备份
  ```

- **坑3**：正则未转义导致字面匹配失败（`.` `*` `[]` 等）
  ```bash
  # 错误：. 匹配任意字符
  sed 's/1.2/1.3/g' file.txt
  
  # 正确：转义 . 匹配字面点
  sed 's/1\.2/1\.3/g' file.txt
  ```

- **坑4**：分隔符冲突（路径中有 `/`）
  ```bash
  # 错误：分隔符 / 与路径冲突
  sed 's//path/to/file//new/path/g' file.txt
  
  # 正确：更换分隔符
  sed 's|/path/to/file|/new/path|g' file.txt
  ```

- **坑5**：不了解模式空间与保持空间的区别导致处理失败
  ```bash
  # 复杂多行处理请使用保持空间相关命令（h/H/g/G/x）
  # 例如逆序文件：sed '1!G;h;$!d' file.txt
  ```

## 速查命令清单

| 操作       | 命令示例                       |
| ---------- | ------------------------------ |
| 全局替换   | `sed 's/old/new/g' file`       |
| 原地修改   | `sed -i 's/old/new/g' file`    |
| 删除空行   | `sed '/^$/d' file`             |
| 删除注释   | `sed '/^#/d' file`             |
| 打印匹配行 | `sed -n '/pattern/p' file`     |
| 打印指定行 | `sed -n '10,20p' file`         |
| 行后插入   | `sed '/pattern/a\text' file`   |
| 行前插入   | `sed '/pattern/i\text' file`   |
| 替换整行   | `sed '/pattern/c\text' file`   |
| 多命令执行 | `sed -e 'cmd1' -e 'cmd2' file` |
| 区间操作   | `sed '/start/,/end/d' file`    |
| 取反删除   | `sed '/keep/!d' file`          |

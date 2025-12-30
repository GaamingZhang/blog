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

# awk命令的基本使用方法

## 核心概念

**awk** 是一个专为文本处理设计的编程语言工具，它结合了模式匹配、数据提取和计算功能，特别适合结构化文本数据的处理和分析。awk的名称来源于其三位开发者Alfred Aho、Peter Weinberger和Brian Kernighan的姓氏首字母。

### 设计理念
awk的设计理念是"数据驱动"，它将输入文件视为由记录（行）和字段（列）组成的表格数据，通过简洁的语法实现复杂的文本处理任务。

### 适用场景
- 日志文件分析和统计
- 结构化数据提取和转换
- 报表生成和数据格式化
- 数据聚合和统计计算
- 简单的文本过滤和转换

**基本语法**：
```bash
awk [options] 'pattern { action }' file
```

---

## 工作原理和基本结构

### awk 的完整工作流程

awk 处理文本的过程分为三个核心阶段，形成了一个完整的数据处理流水线：

```
1. **BEGIN 块（初始化阶段）**：在处理任何输入文件之前执行
2. **主体块（数据处理阶段）**：对输入文件的每一行循环执行
3. **END 块（收尾阶段）**：在处理完所有输入文件的所有行之后执行
```

### 高级阶段用法

#### BEGIN 块的高级应用
BEGIN 块主要用于初始化环境、设置变量、打印表头或读取配置信息：

```bash
# 1. 初始化复杂数据结构
awk 'BEGIN {
    # 初始化城市代码映射表
    cityCode["Beijing"] = "BJ"
    cityCode["Shanghai"] = "SH"
    cityCode["Guangzhou"] = "GZ"
    print "城市代码表初始化完成"
}
{ print $1, cityCode[$3] }' data.txt

# 2. 设置命令行参数和环境变量
awk -v threshold=1000 'BEGIN {
    print "阈值设置为：", threshold
    print "当前用户：", ENVIRON["USER"]
}
$2 > threshold { print "超过阈值：", $0 }' file.txt

# 3. 打印格式化表头
awk 'BEGIN {
    printf "%-10s %-5s %-10s\n", "姓名", "年龄", "城市"
    printf "%-10s %-5s %-10s\n", "------", "----", "------"
}
{ printf "%-10s %-5d %-10s\n", $1, $2, $3 }' data.txt
```

#### END 块的高级应用
END 块用于生成最终报表、统计结果或执行清理操作：

```bash
# 1. 生成带统计信息的报表
awk '{ sum += $2; count[$3]++ } 
END {
    print "===== 统计结果 ====="
    print "平均年龄：", sum/NR
    print "各城市人数："
    for (city in count) {
        printf "  %-10s: %d\n", city, count[city]
    }
}' data.txt

# 2. 排序输出结果（使用外部sort工具）
awk '{ sum[$1] += $2 } 
END {
    for (name in sum) {
        printf "%s %d\n", name, sum[name]
    }
}' sales.txt | sort -k2 -rn

# 3. 执行复杂计算
awk '{ max = (NR==1 || $2>max) ? $2 : max;
       min = (NR==1 || $2<min) ? $2 : min;
       sum += $2
} 
END {
    printf "最大值：%d\n最小值：%d\n平均值：%.2f\n", max, min, sum/NR
}' numbers.txt
```

#### 主体块的处理逻辑
主体块是awk的核心，它对每一行输入执行指定的操作：

```bash
# 行处理的完整流程示例
awk 'BEGIN { FS=":"; OFS="\t" }
# 跳过空行和注释行
/^$/ || /^#/ { next }
# 处理符合条件的行
$3 >= 1000 { 
    # 对字段进行转换
    $2 = toupper($2)
    # 输出处理结果
    print NR, $1, $2, $3
}' data.txt
```

### 字段分割机制

awk 将每一行视为一个**记录**，默认以空格或制表符将记录分割为多个**字段**：

- `$0`：整行内容（完整记录）
- `$1, $2, $3...`：第 1、2、3... 个字段
- `NF`：当前记录的字段总数
- `NR`：当前记录的行号（全局递增）
- `FNR`：当前文件内的行号（处理多文件时重置）

**自定义分割符示例**：
```bash
# 使用冒号分割
awk -F: '{ print $1 }' /etc/passwd

# 使用正则表达式作为分割符
awk -F'[ :]+' '{ print $1, $2 }' file.txt

# 在BEGIN块中设置分割符
awk 'BEGIN { FS="|" } { print $1, $3 }' data.txt
```

### 跨平台兼容性说明
awk的核心语法在不同平台（Linux、macOS、Windows）上保持高度一致，尤其是基础的字段分割、模式匹配和数组操作。但需要注意：
- GNU awk (`gawk`) 提供了更多扩展功能（如数组排序、正则表达式增强）
- BSD awk (macOS默认) 在某些高级特性上可能有所差异
- 建议在跨平台脚本中使用awk的标准语法，避免使用平台特定的扩展

---

## 常用选项和变量

**命令行选项**：

```bash
# -F：指定字段分割符
awk -F: '{ print $1 }' /etc/passwd

# -v：定义变量
awk -v OFS="|" '{ print $1, $2 }' file.txt
# 使用竖线 | 作为输出分割符

# -f：从文件读取 awk 程序
awk -f script.awk file.txt
```

**内置变量**：

| 变量         | 说明                     | 示例                     |
| ------------ | ------------------------ | ------------------------ |
| **NR**       | 当前行号                 | `NR > 5` 处理第 5 行之后 |
| **NF**       | 字段数                   | `NF >= 3` 至少 3 个字段  |
| **FS**       | 输入分割符（默认空格）   | `FS=":"`                 |
| **OFS**      | 输出分割符（默认空格）   | `OFS=","`                |
| **ORS**      | 输出行分割符（默认换行） | `ORS="\n"`               |
| **FILENAME** | 当前文件名               | `print FILENAME, $0`     |
| **FNR**      | 当前文件内的行号         | 处理多文件时区分         |

**示例**：
```bash
# 打印行号和内容
awk '{ print NR ":", $0 }' file.txt

# 处理 /etc/passwd，以冒号分割
awk -F: '{ print $1 "\t" $3 }' /etc/passwd
# 输出：用户名和 UID

# 计数字段
awk '{ sum += $1 } END { print sum }' numbers.txt
```

---

## 模式和条件

**两种模式类型**：

**1. 正则表达式模式**：
```bash
# 匹配包含 "error" 的行
awk '/error/ { print }' logfile.txt

# 忽略大小写匹配
awk 'tolower($0) ~ /error/' logfile.txt

# 不匹配模式
awk '!/error/ { print }' logfile.txt
```

**2. 条件模式**：
```bash
# 打印工资大于 5000 的员工
awk '$3 > 5000 { print $1, $3 }' salary.txt

# 多条件
awk '$2 > 25 && $3 == "Beijing" { print }' data.txt

# 第一行到第五行
awk 'NR >= 1 && NR <= 5' file.txt
# 或简写
awk 'NR==1, NR==5' file.txt  # 范围模式
```

**3. 范围模式**：
```bash
# 从包含 "start" 的行到包含 "end" 的行
awk '/start/, /end/ { print }' file.txt
```

---

## 常用操作和函数

### 字符串和数学函数

#### 基础字符串函数
```bash
# 长度
awk '{ print length($0) }' file.txt

# 子串
awk '{ print substr($1, 1, 3) }' file.txt  # 从第 1 位取 3 个字符

# 查找位置
awk '{ print index($1, "abc") }' file.txt  # 找 "abc" 的位置

# 替换
awk '{ print gsub(/old/, "new") }' file.txt  # 替换所有
awk '{ print sub(/old/, "new") }' file.txt   # 替换第一个

# 转大小写
awk '{ print toupper($1), tolower($2) }' file.txt
```

#### 高级字符串函数
```bash
# 匹配和提取（类似于正则表达式的捕获组）
awk '{ match($0, /([0-9]+) ([a-z]+)/); print substr($0, RSTART, RLENGTH) }' file.txt
# RSTART：匹配的起始位置
# RLENGTH：匹配的长度

# 分割字符串为数组
awk '{ split($0, parts, ":"); for (i=1; i<=length(parts); i++) print parts[i] }' file.txt

# 连接字符串数组（仅 gawk 支持）
awk 'BEGIN {
    parts[1] = "Hello"
    parts[2] = "World"
    parts[3] = "!"
    print join(parts, " ")  # 需要自定义 join 函数
}'

# 自定义 join 函数
function join(array, sep) {
    result = ""
    for (i=1; i in array; i++) {
        if (i > 1) result = result sep
        result = result array[i]
    }
    return result
}
```

#### 数学函数
```bash
# 基础数学函数
awk '{ print sqrt($1), int($2) }' file.txt

# 高级数学函数
awk '{ 
    print "绝对值:", abs($1),
          "对数:", log($2),
          "指数:", exp($3),
          "正弦:", sin($4),
          "余弦:", cos($5),
          "随机数:", rand()
}' file.txt

# 随机数种子设置
awk 'BEGIN {
    srand()  # 使用当前时间作为种子
    print "随机数:", rand()
    
    srand(12345)  # 使用固定种子获得可重现的随机数
    print "固定种子随机数:", rand()
}'
```

#### 时间和日期函数
```bash
# 获取当前时间戳
awk 'BEGIN { print systime() }'  # 返回从1970-01-01 00:00:00 UTC到现在的秒数

# 格式化时间
awk 'BEGIN {
    timestamp = systime()
    print "当前时间:", strftime("%Y-%m-%d %H:%M:%S", timestamp)
    print "ISO格式:", strftime("%FT%T", timestamp)
    print "带时区:", strftime("%Y-%m-%d %H:%M:%S %Z", timestamp)
}'

# 解析时间字符串（仅 gawk 支持）
awk 'BEGIN {
    date_str = "2024-12-20 14:30:00"
    timestamp = mktime(gensub(/-/, " ", "g", gensub(/:/, " ", "g", date_str)))
    print "解析后的时间戳:", timestamp
}'

### 跨平台兼容性说明
时间函数在不同awk版本中差异较大：
- 基本时间函数（`systime()`、`strftime()`）在大多数awk版本中可用
- 高级时间函数（`mktime()`、`strftime()`的扩展格式）仅在gawk中完全支持
- BSD awk (macOS默认) 的`strftime()`函数支持的格式选项较少
- 对于需要复杂时间处理的跨平台脚本，建议使用外部工具（如`date`命令）辅助

#### 输入输出函数
```bash
# getline 函数：读取下一行输入
awk '{ print "当前行:", $0; getline; print "下一行:", $0 }' file.txt

# 从文件读取
awk 'BEGIN {
    while (getline < "data.txt") {
        print "读取数据:", $0
    }
    close("data.txt")
}'

# 写入文件
awk 'BEGIN {
    print "Hello" > "output.txt"
    print "World" >> "output.txt"  # 追加模式
    close("output.txt")
}'

# 管道通信
awk 'BEGIN {
    "ls -l" | getline
    print "当前目录第一个文件:", $9
}'
```

### 数组操作（关联数组）

awk 中所有数组都是**关联数组**（类似哈希表或字典），可以使用任意字符串作为索引：

```bash
# 1. 数组的基本特性
awk 'BEGIN {
    # 索引可以是字符串或数字
    arr["name"] = "John"
    arr[100] = "value"
    
    # 自动创建不存在的数组元素
    arr["new"]++  # 初始值为 0，++ 后为 1
    
    print arr["name"], arr[100], arr["new"]
}'

# 2. 数组元素的遍历
awk 'BEGIN {
    # 创建测试数组
    fruits["apple"] = 10
    fruits["banana"] = 20
    fruits["orange"] = 15
    
    # 使用 for-in 循环遍历
    print "水果库存："
    for (fruit in fruits) {
        print fruit, ":", fruits[fruit]
    }
    
    # 注意：遍历顺序不是插入顺序（gawk 4.0+ 支持数组排序）
}'

# 3. 数组存在性检查
awk 'BEGIN {
    colors["red"] = "#FF0000"
    
    # 检查键是否存在
    if ("red" in colors) {
        print "红色的十六进制代码：", colors["red"]
    }
    
    if (!("blue" in colors)) {
        print "蓝色未定义，设置默认值"
        colors["blue"] = "#0000FF"
    }
}'

# 4. 删除数组元素
awk 'BEGIN {
    data["a"] = 1
    data["b"] = 2
    data["c"] = 3
    
    delete data["b"]  # 删除特定元素
    
    # 重新遍历
    for (key in data) {
        print key, data[key]
    }  # 输出：a 1 和 c 3
    
    # 清空整个数组（使用循环）
    for (key in data) {
        delete data[key]
    }
}'

# 5. 模拟二维数组
awk 'BEGIN {
    # 使用 SUBSEP（默认是 \034）连接键
    matrix[1, 2] = 100
    matrix[3, 4] = 200
    
    # 等价于 matrix["1\0342"] = 100
    
    # 遍历二维数组
    for (cell in matrix) {
        # 分割键获取行列
        split(cell, coords, SUBSEP)
        print "行", coords[1], "列", coords[2], "值", matrix[cell]
    }
}'

# 6. 数组作为函数参数（仅 gawk 支持）
awk 'function printArray(arr, prefix) {
    for (key in arr) {
        print prefix, key, ":", arr[key]
    }
}
BEGIN {
    users["john"] = "john@example.com"
    users["jane"] = "jane@example.com"
    printArray(users, "用户邮箱：")
}'

### 跨平台兼容性说明
awk的关联数组核心功能在所有平台上都一致，但需要注意：
- 数组遍历顺序在不同awk版本中可能不同（无序）
- GNU awk (gawk) 4.0+ 支持数组排序功能（`asort` 和 `asorti`）
- BSD awk (macOS默认) 不支持直接的数组排序，需要借助外部`sort`命令
- 数组作为函数参数仅在gawk中支持，在BSD awk中需要通过变通方法实现

### 分组统计

```bash
# 统计不同城市的人数
awk '{ count[$3]++ } END { for (city in count) print city, count[city] }' data.txt

# 计算平均工资
awk '{ sum += $2; n++ } END { print sum/n }' salary.txt

# 按字段分组求和
awk '{ total[$1] += $2 } END { for (name in total) print name, total[name] }' sales.txt

# 计算分组的最大值和最小值
awk '{ 
    if (!($1 in max) || $2 > max[$1]) max[$1] = $2
    if (!($1 in min) || $2 < min[$1]) min[$1] = $2
} END {
    for (group in max) {
        print group, "最大:", max[group], "最小:", min[group]
    }
}' data.txt
```

### 输出和格式化

```bash
# printf 格式化输出
awk '{ printf "%s\t%d\t%.2f\n", $1, $2, $3 }' file.txt
# %s 字符串，%d 整数，%.2f 两位小数

# 条件输出
awk '{ if ($2 > 5000) print $1, "高薪"; else print $1, "普通" }' salary.txt
```

---

## 常见用法示例

**统计和聚合**：

```bash
# 统计行数（比 wc -l 快）
awk 'END { print NR }' file.txt

# 统计词频
awk '{ for (i=1; i<=NF; i++) count[$i]++ } END { for (w in count) print w, count[w] }' file.txt

# 列求和
awk '{ sum += $1 } END { print sum }' numbers.txt

# 列求平均
awk '{ sum += $1; n++ } END { print sum/n }' numbers.txt

# 列求最大最小值
awk '{ if (NR==1) { max=$1; min=$1 } else { if ($1>max) max=$1; if ($1<min) min=$1 } } END { print "Max:", max, "Min:", min }' numbers.txt
```

**数据提取和转换**：

```bash
# 提取特定列（类似 cut）
awk '{ print $2, $4 }' file.txt

# 去重（第一次出现的行）
awk '!seen[$0]++' file.txt
# 或
awk '!a[$0]++' file.txt

# 反转列顺序
awk '{ for (i=NF; i>=1; i--) print $i }' file.txt

# 列转行
awk '{ for (i=1; i<=NF; i++) print $i }' file.txt
```

**日志和文本分析**：

```bash
# 统计 HTTP 状态码
awk '{ count[$9]++ } END { for (code in count) print code, count[code] }' access.log

# 统计 IP 访问次数
awk '{ count[$1]++ } END { for (ip in count) print ip, count[ip] }' access.log

# 查找并计数错误日志
awk '/ERROR/ { error_count++ } END { print "Errors:", error_count }' app.log

# 统计各进程占用内存
ps aux | awk '{ memory[$11] += $6 } END { for (proc in memory) print proc, memory[proc] }'
```

**数据清洗和格式转换**：

```bash
# 删除重复空行
awk 'NF { print; p=1; next } p' file.txt

# 去掉开头和末尾空格
awk '{ gsub(/^[ \t]+|[ \t]+$/, ""); print }' file.txt

# 在特定行前后添加文本
awk '/pattern/ { print "prefix"; print $0; print "suffix"; next } { print }' file.txt

# CSV 转 TSV
awk -F, '{ OFS="\t"; print $1, $2, $3 }' file.csv
```

---

## 一句话 awk 示例汇总

```bash
# 1. 打印特定列
awk '{ print $1, $3 }' file.txt

# 2. 条件过滤
awk '$2 > 100 { print }' file.txt

# 3. 统计行数
awk 'END { print NR }' file.txt

# 4. 字段求和
awk '{ sum += $1 } END { print sum }' file.txt

# 5. 去重
awk '!a[$0]++' file.txt

# 6. 反转字段顺序
awk '{ for (i=NF; i>=1; i--) printf "%s ", $i; print "" }' file.txt

# 7. 按字段分组统计
awk '{ sum[$1] += $2 } END { for (k in sum) print k, sum[k] }' file.txt

# 8. 替换文本
awk '{ gsub(/old/, "new"); print }' file.txt

# 9. 格式化输出
awk '{ printf "%-10s %5d\n", $1, $2 }' file.txt

# 10. 计算平均值
awk '{ sum += $1; n++ } END { print sum/n }' file.txt
```

---

### 相关高频面试题

#### Q1: 如何用 awk 提取日志文件中的特定字段和统计数据？

**答案**：

```bash
# 示例日志格式：
# 192.168.1.1 - - [20/Dec/2024:10:00:00] "GET /index.html HTTP/1.1" 200 1234

# 1. 提取 IP 地址和状态码
awk '{ print $1, $9 }' access.log

# 2. 统计不同状态码的请求数
awk '{ count[$9]++ } END { for (code in count) print code, count[code] }' access.log

# 3. 统计各 IP 的请求次数（Top 10）
awk '{ count[$1]++ } END { for (ip in count) print ip, count[ip] }' access.log | sort -k2 -rn | head -10

# 4. 统计流量最高的 URL
awk '{ sum[$7] += $10 } END { for (url in sum) print url, sum[url] }' access.log | sort -k2 -rn | head -10

# 5. 统计特定时间段的请求
awk '$4 >= "[20/Dec/2024:10:00:00" && $4 < "[20/Dec/2024:11:00:00" { count++ } END { print count }' access.log
```

#### Q2: awk 中的数组有什么特点？如何使用关联数组？

**答案**：

```bash
# awk 的数组都是关联数组（类似哈希表），不是索引数组
# 键可以是任意字符串，不仅仅是数字

# 1. 计数
awk '{ count[$1]++ } END { for (key in count) print key, count[key] }' file.txt

# 2. 求和
awk '{ sum[$1] += $2 } END { for (key in sum) print key, sum[key] }' file.txt

# 3. 判断是否存在
awk '{ if ($1 in dict) print "exists"; else print "not exists" }' file.txt

# 4. 删除数组元素
awk '{ if (count[$1] > 1) delete count[$1] } END { for (k in count) print k }' file.txt

# 5. 二维数组
awk '{ arr[$1, $2]++ } END { for (k in arr) print k, arr[k] }' file.txt
# 访问：arr[key1, key2] 或 arr[key1 SUBSEP key2]
```

#### Q3: awk 的 gsub 和 sub 函数有什么区别？

**答案**：

```bash
# sub：替换第一个匹配
awk '{ sub(/old/, "new"); print }' file.txt
# "old old old" → "new old old"

# gsub：替换所有匹配
awk '{ gsub(/old/, "new"); print }' file.txt
# "old old old" → "new new new"

# 修改特定字段
awk '{ gsub(/[0-9]/, "X", $2); print }' file.txt  # 只修改第 2 个字段

# 返回值是替换次数
awk '{ n = gsub(/old/, "new"); print n, $0 }' file.txt
```

#### Q4: awk 中的 split 函数有什么用？

**答案**：

```bash
# 将字符串按分割符分割成数组

# 1. 基本用法
awk '{ n = split($0, arr, ":"); for (i=1; i<=n; i++) print arr[i] }' file.txt

# 2. 动态处理字段
echo "a:b:c" | awk '{ split($0, a, ":"); print a[1], a[3] }'
# 输出：a c

# 3. 统计分割后的数据
awk '{ split($0, parts, ","); sum += parts[2] } END { print sum }' file.csv

# 4. 处理嵌套数据
awk '{ split($0, a, "|"); split(a[1], b, ":"); print b[2] }' file.txt
```

#### Q5: awk 的 printf 如何格式化输出？

**答案**：

```bash
# printf 的常用格式符
# %d 整数，%f 浮点，%s 字符串，%x 十六进制
# 修饰符：- 左对齐，0 补零，. 精度

# 1. 基本格式
awk '{ printf "%s %d %.2f\n", $1, $2, $3 }' file.txt

# 2. 宽度和对齐
awk '{ printf "%-15s %5d %10.2f\n", $1, $2, $3 }' file.txt
# %-15s：左对齐 15 个字符
# %5d：右对齐 5 位整数
# %10.2f：10 位宽，2 位小数

# 3. 补零
awk '{ printf "%05d\n", $1 }' file.txt
# 12 → 00012

# 4. 百分比格式
awk '{ printf "%.1f%%\n", $1 * 100 }' file.txt
```

#### Q6: awk 如何处理多文件？FNR 和 NR 的区别？

**答案**：

```bash
# NR：总行号（所有文件）
# FNR：当前文件的行号（会重置）

# 示例：处理两个文件
awk '{ print FILENAME, FNR, NR, $0 }' file1.txt file2.txt

# 输出可能为：
# file1.txt 1 1 ...
# file1.txt 2 2 ...
# file2.txt 1 3 ...  ← FNR 重置，NR 继续增长
# file2.txt 2 4 ...

# 处理多文件的合并操作
awk 'FNR == 1 { print "=== " FILENAME " ===" } { print }' file1.txt file2.txt

# 只处理第一个文件
awk 'FNR == NR { print }' file1.txt file2.txt
# 或
awk 'FILENAME == "file1.txt" { print }' file1.txt file2.txt
```

#### Q7: 如何使用 awk 实现数据去重并保留最新记录？

**答案**：

```bash
# 示例数据（最后一列是时间戳，保留每个用户的最新记录）
# John 30 Beijing 1702987200
# Jane 25 Shanghai 1702987300
# John 31 Beijing 1702987400  # John 的更新记录

# 方法1：使用数组存储最新记录，然后输出所有记录
awk '{ 
    # 使用第一个字段作为键，保存最新的记录
    latest[$1] = $0
    # 保存时间戳用于排序（如果需要按时间顺序输出）
    timestamp[$1] = $4
}' 
END { 
    # 直接输出（顺序不一定是时间顺序）
    for (user in latest) {
        print latest[user]
    }
    
    # 按时间戳排序输出（需要外部sort工具）
    print "\n按时间顺序输出："
    for (user in latest) {
        print timestamp[user], latest[user]
    } | sort -k1 -n | cut -f2-  # 排序后去掉时间戳字段
}' data.txt

# 方法2：倒序读取文件，第一次出现的记录就是最新的
# 先反转文件，然后去重，再反转回来
awk '!seen[$1]++' <(tac data.txt) | tac
```

#### Q8: awk 中 getline 函数的使用场景和注意事项？

**答案**：

```bash
# getline 的基本功能是读取下一行输入

# 使用场景1：读取特定行的下一行数据
awk '/start/ { getline; print "找到start的下一行：", $0 }' file.txt

# 使用场景2：从文件读取数据
awk 'BEGIN {
    while (getline < "config.txt") {
        if ($1 == "PORT") {
            port = $2
            break
        }
    }
    close("config.txt")
    print "配置的端口：", port
}'

# 使用场景3：读取命令输出
awk 'BEGIN {
    "date +%Y-%m-%d" | getline currentDate
    print "今天日期：", currentDate
}'

# 注意事项1：getline 会改变当前记录和字段变量
awk '{ print "当前行：", $0; getline; print "getline后：", $0 }' file.txt

# 注意事项2：检查 getline 的返回值
awk 'BEGIN {
    while ((getline < "data.txt") > 0) {
        print $0
    }
    if (ERRNO) {
        print "读取文件错误：", ERRNO
    }
    close("data.txt")
}'

# 注意事项3：避免在循环中嵌套使用 getline
# 这可能导致不可预期的行为
```

#### Q9: 如何在 awk 中实现自定义函数？

**答案**：

```bash
# 自定义函数的基本语法
awk 'function function_name(parameter1, parameter2, ...) {
    # 函数体
    return result
}
{ 
    # 调用函数
    result = function_name($1, $2)
    print result
}' file.txt

# 示例1：计算阶乘
awk 'function factorial(n) {
    if (n <= 1) return 1
    return n * factorial(n-1)
}
{ print $1, "的阶乘：", factorial($1) }' numbers.txt

# 示例2：格式化输出
awk 'function format_name(first, last) {
    return toupper(substr(first, 1, 1)) substr(first, 2) " " \
           toupper(substr(last, 1, 1)) substr(last, 2)
}
{ print format_name($1, $2) }' names.txt

# 示例3：数据验证
awk 'function is_valid_email(email) {
    return match(email, /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/) != 0
}
{ 
    if (is_valid_email($2)) {
        print $1, "的邮箱有效：", $2
    } else {
        print $1, "的邮箱无效：", $2
    }
}' users.txt

# 注意：参数是按值传递的，数组参数需要特殊处理（仅 gawk 支持）
```

---

### 快速参考

**常用组合**：
```bash
# 统计字符频率
awk '{ for (i=1; i<=length($0); i++) count[substr($0,i,1)]++ } 
     END { for (c in count) print c, count[c] }' file.txt

# 打印第 N 到 M 行
awk 'NR >= 5 && NR <= 10' file.txt

# 按字段排序（使用外部工具）
awk '{ print }' file.txt | sort -k2 -n

# 统计文件大小最大的前 3 个
ls -l | awk '{ if (NR > 1) print $9, $5 }' | sort -k2 -rn | head -3
```

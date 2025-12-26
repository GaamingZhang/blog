#!/bin/bash

cd /Users/gaamingzhang/git/gaamingzhangblog/src/algorithm/leetcode

for cpp_file in *.cpp; do
    if [ -f "$cpp_file" ]; then
        # 获取文件名（不含扩展名）
        filename="${cpp_file%.cpp}"
        md_file="${filename}.md"
        
        echo "Processing: $cpp_file -> $md_file"
        
        # 提取文件名作为标题
        title="$filename"
        
        # 读取 cpp 文件内容
        content=$(cat "$cpp_file")
        
        # 检查是否已经有标题（以 # 开头）
        if echo "$content" | head -1 | grep -q "^#"; then
            # 已经有标题，直接用 ```cpp 包围代码部分
            # 找到第一个 ```cpp 的位置
            if echo "$content" | grep -q '```cpp'; then
                # 已经有代码块标记，保持原样
                echo "$content" > "$md_file"
            else
                # 需要添加代码块标记
                # 找到 class Solution 的位置
                awk '
                    BEGIN { in_code = 0 }
                    /^class Solution/ {
                        if (!in_code) {
                            print "```cpp"
                            in_code = 1
                        }
                    }
                    { print }
                    END {
                        if (in_code) {
                            print "```"
                        }
                    }
                ' "$cpp_file" > "$md_file"
            fi
        else
            # 没有标题，需要添加标题和代码块
            echo "# $title" > "$md_file"
            echo "" >> "$md_file"
            echo '```cpp' >> "$md_file"
            cat "$cpp_file" >> "$md_file"
            echo "" >> "$md_file"
            echo '```' >> "$md_file"
        fi
        
        echo "Created: $md_file"
    fi
done

echo "All .cpp files converted to .md"

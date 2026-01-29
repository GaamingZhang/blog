#!/usr/bin/env python3
import re
import sys

def fix_markdown_file(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 定义答案匹配模式
    answer_pattern = re.compile(r'答案：\s*!!\s*([A-Z])\s*!!')
    
    # 替换所有答案格式
    fixed_content = answer_pattern.sub(
        r'答案：::: spoiler \1\n解析（点击查看答案）\n:::', 
        content
    )
    
    with open(file_path, 'w', encoding='utf-8') as f:
        f.write(fixed_content)
    
    print(f"已修复文件: {file_path}")

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(f"使用方法: {sys.argv[0]} <markdown文件路径>")
        sys.exit(1)
    
    fix_markdown_file(sys.argv[1])

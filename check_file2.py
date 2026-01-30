#!/usr/bin/env python3

def check_file_integrity(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        lines = f.read().split('\n')
    
    print(f'文件总共有 {len(lines)} 行')
    
    # 检查代码块是否正确闭合
    code_block_open = False
    code_block_type = ''
    
    for i, line in enumerate(lines, 1):
        if line.startswith('```'):
            if code_block_open:
                print(f'代码块结束于第{i}行，类型: {code_block_type}')
                code_block_open = False
                code_block_type = ''
            else:
                code_block_type = line[3:]
                print(f'代码块开始于第{i}行，类型: {code_block_type}')
                code_block_open = True
    
    if code_block_open:
        print(f'错误: 未闭合的代码块，类型: {code_block_type}')
    else:
        print('代码块都已正确闭合')

# 检查文件
check_file_integrity('src/posts/docker/镜像原理与分层存储.md')

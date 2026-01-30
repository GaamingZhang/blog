#!/usr/bin/env python3

def check_category_file(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    print('文件内容前500字符:')
    print(content[:500])
    print('\n...\n')
    
    if 'export const categoriesMap' in content:
        print('✓ 发现 categoriesMap 导出')
    else:
        print('✗ 未发现 categoriesMap 导出')

# 检查文件
check_category_file('src/.vuepress/.temp/blog/category.js')

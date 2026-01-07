#!/usr/bin/env python3
import re

def check_headings(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    in_code_block = False
    headings = []
    lines = content.split('\n')
    
    for i, line in enumerate(lines):
        if line.strip().startswith('```'):
            in_code_block = not in_code_block
            continue
        
        if not in_code_block:
            heading_match = re.match(r'^(#{1,6})\s+(.*)$', line)
            if heading_match:
                level = len(heading_match.group(1))
                text = heading_match.group(2)
                headings.append((i+1, level, text))
    
    print('Headings in the document (excluding code blocks):')
    print('-' * 70)
    for line_num, level, text in headings:
        print(f'Line {line_num: 3d}: {' ' * (level-1)}#{level} {text}')
    
    print('\nHeading hierarchy check:')
    print('-' * 70)
    
    if not headings:
        print('✗ No headings found in the document')
        return False
    
    valid = True
    prev_level = headings[0][1]
    
    for i in range(1, len(headings)):
        current_level = headings[i][1]
        if current_level > prev_level + 1:
            print(f'✗ Invalid heading level jump at line {headings[i][0]}: #{prev_level} → #{current_level}')
            valid = False
        prev_level = current_level
    
    if valid:
        print('✓ All headings follow valid hierarchy (h1 → h2 → h3 → h4 → h5 → h6)')
    else:
        print('✗ Found invalid heading level jumps')
    
    return valid

if __name__ == '__main__':
    file_path = '/Users/gaamingzhang/git/gaamingzhangblog/src/posts/kubernetes/如何从外部访问Kubernetes Service.md'
    check_headings(file_path)

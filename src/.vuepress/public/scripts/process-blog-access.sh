#!/bin/bash

ACCESS_LOG="/var/log/nginx/blog-access.log"
PROCESSED_LOG="/var/log/nginx/blog-access.processed"
LOCK_FILE="/tmp/blog-access.lock"

# 加载环境变量
source /etc/wxpush-credentials.conf

# 检查锁文件
if [ -f "$LOCK_FILE" ]; then
    echo "锁文件存在，脚本已在运行"
    exit 0
fi

touch "$LOCK_FILE"

# 处理新日志
    if [ -f "$ACCESS_LOG" ]; then
        # 获取当前日志文件的inode
        CURRENT_INODE=$(stat -c '%i' "$ACCESS_LOG" 2>/dev/null || stat -f '%i' "$ACCESS_LOG")
        
        # 从处理记录文件读取上次的信息
        if [ -f "$PROCESSED_LOG" ]; then
            SAVED_INODE=$(sed -n '1p' "$PROCESSED_LOG" 2>/dev/null)
            LAST_LINE=$(sed -n '2p' "$PROCESSED_LOG" 2>/dev/null || echo "0")
        else
            SAVED_INODE=""
            LAST_LINE=0
        fi
        
        # 检查日志文件是否被滚动（inode是否变化）
        if [ "$CURRENT_INODE" != "$SAVED_INODE" ]; then
            echo "检测到日志文件已滚动，重置处理位置"
            LAST_LINE=0
        fi
        
        # 获取当前日志文件的行数
        CURRENT_LINE=$(wc -l < "$ACCESS_LOG")
        
        if [ "$CURRENT_LINE" -gt "$LAST_LINE" ]; then
            # 提取新日志并发送通知
            tail -n +$((LAST_LINE + 1)) "$ACCESS_LOG" | while read line; do
                # 解析日志
                IP=$(echo "$line" | awk '{print $1}')
                URI=$(echo "$line" | awk '{print $7}')
                
                echo "访问IP: $IP\n访问页面: $URI"
                
                # 发送到微信
                /var/wxpush/wxpush \
                    -appID "$WXPUSH_APPID" \
                    -secret "$WXPUSH_SECRET" \
                    -userID "$WXPUSH_USERID" \
                    -templateID "$WXPUSH_TEMPLATEID" \
                    -title "博客访问通知" \
                    -content "访问IP: $IP\n访问页面: $URI"
                echo "已发送通知"
            done
        fi
        
        # 更新处理记录（保存inode和行号）
        echo "$CURRENT_INODE" > "$PROCESSED_LOG"
        echo "$CURRENT_LINE" >> "$PROCESSED_LOG"
    fi
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
    # 获取上次处理的位置
    LAST_LINE=$(cat "$PROCESSED_LOG" 2>/dev/null || echo "0")
    CURRENT_LINE=$(wc -l < "$ACCESS_LOG")
    
    if [ "$CURRENT_LINE" -gt "$LAST_LINE" ]; then
        # 提取新日志并发送通知
        tail -n +$((LAST_LINE + 1)) "$ACCESS_LOG" | while read line; do
            # 解析日志
            IP=$(echo "$line" | awk '{print $1}')
            URI=$(echo "$line" | awk '{print $7}')
            
            # 发送到微信
            /var/wxpush/wxpush \
                -appID "$WXPUSH_APPID" \
                -secret "$WXPUSH_SECRET" \
                -userID "$WXPUSH_USERID" \
                -templateID "$WXPUSH_TEMPLATEID" \
                -title "博客访问通知" \
                -content "访问IP: $IP\n访问页面: $URI"
        done
        
        # 更新处理位置
        echo "$CURRENT_LINE" > "$PROCESSED_LOG"
    fi
fi
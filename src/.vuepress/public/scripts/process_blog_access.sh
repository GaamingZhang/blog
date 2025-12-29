#!/bin/bash

ACCESS_LOG="/var/log/nginx/blog-access.log"
PROCESSED_LOG="/var/log/nginx/blog-access.processed"
LOCK_FILE="/tmp/blog-access.lock"
IP_CACHE_FILE="/var/log/nginx/ip-city.cache"

# 加载环境变量
source /etc/wxpush-credentials.conf

# 查询IP属地函数（带缓存）
get_ip_city() {
    local ip=$1
    local city
    
    # 先从缓存查找
    if [ -f "$IP_CACHE_FILE" ]; then
        city=$(grep "^$ip," "$IP_CACHE_FILE" 2>/dev/null | cut -d',' -f2)
        if [ -n "$city" ]; then
            echo "$city"
            return
        fi
    fi
    
    # 缓存中没有，请求API
    city=$(curl -s "http://ip-api.com/json/$ip" | grep -o '"city":"[^"]*"' | cut -d'"' -f4)
    if [ -z "$city" ]; then
        city="未知"
    fi
    
    # 写入缓存
    echo "$ip,$city" >> "$IP_CACHE_FILE"
    
    echo "$city"
}

# 清理锁文件的函数
cleanup() {
    rm -f "$LOCK_FILE"
    echo "已清理锁文件"
}

# 注册退出时执行的清理函数
trap cleanup EXIT

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
            DATE=$(echo "$line" | awk '{print $4}')
            URI=$(echo "$line" | awk '{print $7}')
            USER_AGENT=$(echo "$line" | awk -F'"' '{print $6}')

            # 检查是否为爬虫请求
            if echo "$USER_AGENT" | grep -qiE '(bot|crawler|spider|scraper|curl|wget|python-requests|httpie|scrapy|phantomjs|splash|headless|chrome-lighthouse|googlebot|bingbot|yandexbot|baidu|360spider|sogou|sohu-search|sina|ia_archiver|archive.org|slurp)'; then
                echo "跳过爬虫请求: $IP,$DATE,$URI ($USER_AGENT)"
                continue
            fi

            # 查询IP属地
            CITY=$(get_ip_city "$IP")
            
            echo "$IP,$CITY,$DATE,$URI"

            # 发送到微信
            /var/wxpush/wxpush \
                -appID "$WXPUSH_APPID" \
                -secret "$WXPUSH_SECRET" \
                -userID "$WXPUSH_USERID" \
                -templateID "$WXPUSH_TEMPLATEID" \
                -title "博客访问通知" \
                -content "$IP,$CITY,$DATE,$URI"
            echo "已发送通知"
        done
    fi
    
    # 更新处理记录（保存inode和行号）
    echo "$CURRENT_INODE" > "$PROCESSED_LOG"
    echo "$CURRENT_LINE" >> "$PROCESSED_LOG"
fi
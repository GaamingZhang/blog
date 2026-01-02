package utils

import (
	"os"
	"strconv"
)

// GetMaxFileSize 从环境变量中获取最大上传文件大小（MB），如果未设置或无效，则返回默认值 50MB
// 可以通过配置环境变量 MAX_FILE_SIZE_MB 更改
func GetMaxFileSize() int64 {
	if sizeStr := os.Getenv("MAX_FILE_SIZE_MB"); sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && size > 0 {
			return size * 1024 * 1024
		}
	}
	return 50 * 1024 * 1024 // 默认 50MB
}

func GetMaxFileSizeMB() int64 {
	if sizeStr := os.Getenv("MAX_FILE_SIZE_MB"); sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && size > 0 {
			return size
		}
	}
	return 50 // default 50MB
}

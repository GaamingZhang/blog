help:
	@echo "开发工具:"
	@echo "  fmt               格式化代码"
	@echo "  lint              代码检查"
	@echo "  deps              安装依赖"

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run

# 下载依赖
deps:
	go mod download	
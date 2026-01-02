package mcp

import "errors"

var (
	// ErrUnsupportedTransport 不支持的传输类型错误
	ErrUnsupportedTransport = errors.New("unsupported transport type")

	// ErrNotConnected 客户端未连接错误
	ErrNotConnected = errors.New("client not connected")

	// ErrAlreadyConnected 客户端已连接错误
	ErrAlreadyConnected = errors.New("client already connected")

	// ErrInitializedFailed 初始化失败错误
	ErrInitializedFailed = errors.New("MCP initialize handshake failed")

	// ErrToolNotFound 工具未找到错误
	ErrToolNotFound = errors.New("tool not found")

	// ErrResourceNotFound 资源未找到错误
	ErrResourceNotFound = errors.New("resource not found")

	// ErrInvalidResponse 无效响应错误
	ErrInvalidResponse = errors.New("invalid response from server")

	// ErrTimeout 超时错误
	ErrTimeout = errors.New("operation timed out")

	// ErrConnectionClosed 连接已关闭错误
	ErrConnectionClosed = errors.New("connection closed")
)

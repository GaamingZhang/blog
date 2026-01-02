package types

// ContextKey定义了上下文键的类型，以避免字符串冲突
type ContextKey string

const (
	// TenantIDContextKey 是租户ID的上下文键
	TenantIDContextKey ContextKey = "TenantId"

	// TenantInfoContextKey 是租户信息的上下文键
	TenantInfoContextKey ContextKey = "TenantInfo"

	// RequestIDContextKey 是请求ID的上下文键
	RequestIDContextKey ContextKey = "RequestId"

	// LoggerContextKey 是日志记录器的上下文键
	LoggerContextKey ContextKey = "Logger"
)

// String 返回上下文键的字符串表示
func (c ContextKey) String() string {
	return string(c)
}

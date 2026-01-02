package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode 定义了错误码的类型
type ErrorCode int

const (
	// 常见错误类型 （1000-1999）
	ErrBadRequest          ErrorCode = 1000
	ErrUnauthorized        ErrorCode = 1001
	ErrForbidden           ErrorCode = 1002
	ErrNotFound            ErrorCode = 1003
	ErrMethodNotAllowed    ErrorCode = 1004
	ErrConflict            ErrorCode = 1005
	ErrTooManyRequests     ErrorCode = 1006
	ErrInternalServerError ErrorCode = 1007
	ErrServiceUnavailable  ErrorCode = 1008
	ErrTimeout             ErrorCode = 1009
	ErrValidation          ErrorCode = 1010

	// Tenant 相关错误码（2000-2999）
	ErrTenantNotFound      ErrorCode = 2000
	ErrTenantAlreadyExists ErrorCode = 2001
	ErrTenantInactive      ErrorCode = 2002
	ErrTenantNameRequired  ErrorCode = 2003
	ErrTenantInvalidStatus ErrorCode = 2004

	// Agent 相关错误码（2100-2199）
	ErrAgentMissingThinkingModel ErrorCode = 2100
	ErrAgentMissingAllowedTools  ErrorCode = 2101
	ErrAgentInvalidMaxIterations ErrorCode = 2102
	ErrAgentInvalidTemperature   ErrorCode = 2103

	// 更多的错误码请在这里添加
)

// AppError 定义了应用级别的错误结构体
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	// omitempty 确保在 Details 为空时不包含该字段
	Details any `json:"details,omitempty"`
	// 忽略 HTTPCode 字段，不包含在 JSON 响应中
	HTTPCode int `json:"-"`
}

// Error 实现 error 接口，返回错误码和错误消息
func (e *AppError) Error() string {
	return fmt.Sprintf("error code: %d, error message: %s", e.Code, e.Message)
}

// WithDetails 设置错误的详细信息
func (e *AppError) WithDetails(details any) *AppError {
	e.Details = details
	return e
}

// NewBadRequestError 创建一个新的 BadRequest 错误
func NewBadRequestError(message string) *AppError {
	return &AppError{
		Code:     ErrBadRequest,
		Message:  message,
		HTTPCode: http.StatusBadRequest,
	}
}

// NewUnauthorizedError 创建一个新的 Unauthorized 错误
func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:     ErrUnauthorized,
		Message:  message,
		HTTPCode: http.StatusUnauthorized,
	}
}

// NewForbiddenError 创建一个新的 Forbidden 错误
func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:     ErrForbidden,
		Message:  message,
		HTTPCode: http.StatusForbidden,
	}
}

// NewNotFoundError 创建一个新的 NotFound 错误
func NewNotFoundError(message string) *AppError {
	return &AppError{
		Code:     ErrNotFound,
		Message:  message,
		HTTPCode: http.StatusNotFound,
	}
}

// NewConflictError 创建一个新的 Conflict 错误
func NewConflictError(message string) *AppError {
	return &AppError{
		Code:     ErrConflict,
		Message:  message,
		HTTPCode: http.StatusConflict,
	}
}

// NewInternalServerError 创建一个新的 InternalServerError 错误
func NewInternalServerError(message string) *AppError {
	if message == "" {
		message = "服务器内部错误"
	}
	return &AppError{
		Code:     ErrInternalServerError,
		Message:  message,
		HTTPCode: http.StatusInternalServerError,
	}
}

// NewValidationError 创建一个新的 Validation 错误
func NewValidationError(message string) *AppError {
	return &AppError{
		Code:     ErrValidation,
		Message:  message,
		HTTPCode: http.StatusBadRequest,
	}
}

// NewTenantNotFoundError 创建一个新的 TenantNotFound 错误
func NewTenantNotFoundError() *AppError {
	return &AppError{
		Code:     ErrTenantNotFound,
		Message:  "租户不存在",
		HTTPCode: http.StatusNotFound,
	}
}

// NewTenantAlreadyExistsError 创建一个新的 TenantAlreadyExists 错误
func NewTenantAlreadyExistsError() *AppError {
	return &AppError{
		Code:     ErrTenantAlreadyExists,
		Message:  "租户已存在",
		HTTPCode: http.StatusConflict,
	}
}

// NewTenantInactiveError 创建一个新的 TenantInactive 错误
func NewTenantInactiveError() *AppError {
	return &AppError{
		Code:     ErrTenantInactive,
		Message:  "租户已停用",
		HTTPCode: http.StatusForbidden,
	}
}

// NewAgentMissingThinkingModelError 创建一个新的 AgentMissingThinkingModel 错误
func NewAgentMissingThinkingModelError() *AppError {
	return &AppError{
		Code:     ErrAgentMissingThinkingModel,
		Message:  "启用Agent模式前，请选选择思考模型",
		HTTPCode: http.StatusBadRequest,
	}
}

// NewAgentMissingAllowedToolsError 创建一个新的 AgentMissingAllowedTools 错误
func NewAgentMissingAllowedToolsError() *AppError {
	return &AppError{
		Code:     ErrAgentMissingAllowedTools,
		Message:  "至少需要选择一个允许的工具",
		HTTPCode: http.StatusBadRequest,
	}
}

// NewAgentInvalidMaxIterationsError 创建一个新的 AgentInvalidMaxIterations 错误
func NewAgentInvalidMaxIterationsError() *AppError {
	return &AppError{
		Code:     ErrAgentInvalidMaxIterations,
		Message:  "最大迭代次数必须在1-20之间",
		HTTPCode: http.StatusBadRequest,
	}
}

// NewAgentInvalidTemperatureError 创建一个新的 AgentInvalidTemperature 错误
func NewAgentInvalidTemperatureError() *AppError {
	return &AppError{
		Code:     ErrAgentInvalidTemperature,
		Message:  "温度参数必须在0-2之间",
		HTTPCode: http.StatusBadRequest,
	}
}

// IsAppError 检查错误是否为 AppError 类型
func IsAppError(err error) (*AppError, bool) {
	// 类型断言
	appErr, ok := err.(*AppError)
	return appErr, ok
}

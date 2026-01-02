package errors

import "errors"

var (
	// ErrSessionNotFound 会话未找到错误
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExpired 会话过期错误
	ErrSessionExpired = errors.New("session expired")

	// ErrSessionLimitExceeded 会话限制超出错误
	ErrSessionLimitExceeded = errors.New("session limit exceeded")

	// ErrSessionIDInvalid 会话ID无效错误
	ErrInvalidSessionID = errors.New("invalid session id")

	// ErrInvalidTenantID 租户ID无效错误
	ErrInvalidTenantID = errors.New("invalid tenant id")
)

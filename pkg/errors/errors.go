package errors

import (
	"errors"
	"net/http"
)

// AppError 应用错误类型
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// Unwrap 实现 errors.Unwrap 接口
func (e *AppError) Unwrap() error {
	return e.Err
}

// 常见错误
var (
	ErrUnauthorized     = &AppError{Code: http.StatusUnauthorized, Message: "unauthorized"}
	ErrForbidden        = &AppError{Code: http.StatusForbidden, Message: "forbidden"}
	ErrNotFound         = &AppError{Code: http.StatusNotFound, Message: "resource not found"}
	ErrBadRequest       = &AppError{Code: http.StatusBadRequest, Message: "bad request"}
	ErrInternalServer   = &AppError{Code: http.StatusInternalServerError, Message: "internal server error"}
	ErrTooManyRequests  = &AppError{Code: http.StatusTooManyRequests, Message: "too many requests"}
)

// NewAppError 创建应用错误
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WrapError 包装错误
func WrapError(err error, message string) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: message,
		Err:     err,
	}
}

// Is 实现 errors.Is 接口
func (e *AppError) Is(target error) bool {
	var appErr *AppError
	if errors.As(target, &appErr) {
		return e.Code == appErr.Code && e.Message == appErr.Message
	}
	return false
}

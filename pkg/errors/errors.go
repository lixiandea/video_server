package errors

import (
	"errors"
	"fmt"
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
	ErrUnauthorized    = &AppError{Code: http.StatusUnauthorized, Message: "unauthorized"}
	ErrForbidden       = &AppError{Code: http.StatusForbidden, Message: "forbidden"}
	ErrNotFound        = &AppError{Code: http.StatusNotFound, Message: "resource not found"}
	ErrBadRequest      = &AppError{Code: http.StatusBadRequest, Message: "bad request"}
	ErrInternalServer  = &AppError{Code: http.StatusInternalServerError, Message: "internal server error"}
	ErrTooManyRequests = &AppError{Code: http.StatusTooManyRequests, Message: "too many requests"}
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

// 按状态码构造带自定义消息的错误，供 service 层表达领域错误（如 not found）。
func NotFoundf(format string, args ...interface{}) *AppError {
	return &AppError{Code: http.StatusNotFound, Message: fmt.Sprintf(format, args...)}
}

func BadRequestf(format string, args ...interface{}) *AppError {
	return &AppError{Code: http.StatusBadRequest, Message: fmt.Sprintf(format, args...)}
}

func Unauthorizedf(format string, args ...interface{}) *AppError {
	return &AppError{Code: http.StatusUnauthorized, Message: fmt.Sprintf(format, args...)}
}

func Forbiddenf(format string, args ...interface{}) *AppError {
	return &AppError{Code: http.StatusForbidden, Message: fmt.Sprintf(format, args...)}
}

func Conflictf(format string, args ...interface{}) *AppError {
	return &AppError{Code: http.StatusConflict, Message: fmt.Sprintf(format, args...)}
}

// Internalf 包装底层错误，对调用方仅暴露通用消息。
func Internalf(err error, message string) *AppError {
	return &AppError{Code: http.StatusInternalServerError, Message: message, Err: err}
}

// StatusCode 返回错误对应的 HTTP 状态码；非 AppError 视为 500。
func StatusCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) && appErr.Code > 0 {
		return appErr.Code
	}
	return http.StatusInternalServerError
}

// Message 返回 AppError 的对外消息；非 AppError 返回 fallback。
func Message(err error, fallback string) string {
	var appErr *AppError
	if errors.As(err, &appErr) && appErr.Message != "" {
		return appErr.Message
	}
	return fallback
}

// Is 实现 errors.Is 接口：同类错误按状态码判定（404 家族彼此相等）。
func (e *AppError) Is(target error) bool {
	var appErr *AppError
	if errors.As(target, &appErr) {
		return e.Code == appErr.Code
	}
	return false
}

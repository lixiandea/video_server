// Package utils provides utility functions for the video server.
package utils

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/lixiandea/video_server/pkg/errors"
)

// ErrorResponse 是全部接口统一的错误响应体：
//
//	{"success": false, "code": 404, "error": "video not found"}
//
// 保留 error 字段是为了兼容既有前端（仅读取 error），新增 success/code 便于
// 调用方按状态码分支处理，无需解析文案。
type ErrorResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Error   string `json:"error"`
}

// Fail 写入统一错误响应并终止后续处理。
func Fail(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Success: false,
		Code:    status,
		Error:   message,
	})
}

// FailFrom 将 service 层错误映射为统一错误响应：
// 携带 AppError 的错误按其状态码与消息返回（例如资源不存在 → 404），
// 其余错误一律 500，且只返回 fallback，避免把数据库细节暴露给调用方。
func FailFrom(c *gin.Context, err error, fallback string) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) && appErr.Code > 0 {
		Fail(c, appErr.Code, appErr.Message)
		return
	}
	Fail(c, http.StatusInternalServerError, fallback)
}

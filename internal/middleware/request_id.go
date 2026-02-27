package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDKey 请求 ID 的上下文键
	RequestIDKey = "X-Request-ID"
)

// RequestIDMiddleware 为每个请求生成唯一的追踪 ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取 Request-ID
		requestID := c.GetHeader(RequestIDKey)
		
		// 如果没有，则生成新的 UUID
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 设置到上下文中
		c.Set(RequestIDKey, requestID)

		// 在响应头中返回 Request-ID
		c.Header(RequestIDKey, requestID)

		c.Next()
	}
}

// GetRequestID 从上下文获取请求 ID
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lixiandea/video_server/pkg/logging"
	"go.uber.org/zap"
)

// auditMethods 是需要审计的写操作方法。
var auditMethods = map[string]struct{}{
	http.MethodPost:   {},
	http.MethodPut:    {},
	http.MethodPatch:  {},
	http.MethodDelete: {},
}

// AuditMiddleware 把写操作记录为结构化审计事件（谁、做了什么、结果、耗时）。
// 只记录请求元数据，不记录请求体，避免口令等敏感信息落日志。
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := auditMethods[c.Request.Method]; !ok {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		fields := []zap.Field{
			zap.Bool("audit", true),
			zap.String("method", c.Request.Method),
			zap.String("route", c.FullPath()),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("latency_ms", time.Since(start).Milliseconds()),
			zap.String("remote_addr", c.Request.RemoteAddr),
		}
		if requestID := GetRequestID(c); requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}
		if userID, ok := c.Get("user_id"); ok {
			fields = append(fields, zap.Any("user_id", userID))
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
		}

		logging.GetLogger().Info("audit event", fields...)
	}
}

package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lixiandea/video_server/internal/models"
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

// AuditSink 接收审计事件（由异步落库实现）。
type AuditSink interface {
	Record(models.AuditLog)
}

// AuditMiddleware 把写操作记录为结构化审计事件（谁、做了什么、结果、耗时）。
// 只记录请求元数据，不记录请求体，避免口令等敏感信息落日志。
func AuditMiddleware(sink AuditSink) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := auditMethods[c.Request.Method]; !ok {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		entry := models.AuditLog{
			RequestID:  GetRequestID(c),
			Method:     c.Request.Method,
			Route:      c.FullPath(),
			Path:       c.Request.URL.Path,
			Status:     c.Writer.Status(),
			LatencyMs:  time.Since(start).Milliseconds(),
			RemoteAddr: c.Request.RemoteAddr,
		}
		if userID, ok := c.Get("user_id"); ok {
			if id, ok := userID.(uint); ok {
				entry.UserID = &id
			}
		}
		if len(c.Errors) > 0 {
			entry.ErrorMessage = c.Errors.String()
		}

		if sink != nil {
			sink.Record(entry)
		}

		fields := []zap.Field{
			zap.Bool("audit", true),
			zap.String("method", entry.Method),
			zap.String("route", entry.Route),
			zap.String("path", entry.Path),
			zap.Int("status", entry.Status),
			zap.Int64("latency_ms", entry.LatencyMs),
			zap.String("remote_addr", entry.RemoteAddr),
		}
		if entry.RequestID != "" {
			fields = append(fields, zap.String("request_id", entry.RequestID))
		}
		if entry.UserID != nil {
			fields = append(fields, zap.Uint("user_id", *entry.UserID))
		}
		if entry.ErrorMessage != "" {
			fields = append(fields, zap.String("error", entry.ErrorMessage))
		}

		logging.GetLogger().Info("audit event", fields...)
	}
}

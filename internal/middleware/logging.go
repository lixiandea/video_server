package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lixiandea/video_server/pkg/logging"
	"go.uber.org/zap"
)

// LoggingMiddleware adds structured logging to all requests
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get user ID from context if available
		var userID interface{}
		if uid, exists := c.Get("user_id"); exists {
			userID = uid
		}

		// Log the request with request ID
		logging.LogRequest(
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
			userID,
		)

		// Add timing header for client-side performance monitoring
		c.Header("X-Response-Time", duration.String())
	}
}

// ErrorLoggingMiddleware captures and logs application errors
func ErrorLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process request
		c.Next()
		
		// Log errors if any occurred
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logging.Error("Application error",
					zap.String("error", err.Error()),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.Int("status", c.Writer.Status()),
				)
			}
		}
	}
}

// DatabaseLoggingMiddleware logs database operations
func DatabaseLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// This middleware can be enhanced to track database operations
		// by wrapping the database calls or using database hooks
		
		// For now, we'll just pass through
		c.Next()
	}
}

// PerformanceMonitoringMiddleware tracks request performance metrics
func PerformanceMonitoringMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Process request
		c.Next()
		
		duration := time.Since(start)
		
		// Log slow requests (> 1 second)
		if duration > time.Second {
			logging.Warn("Slow request detected",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Duration("duration", duration),
				zap.Int("status", c.Writer.Status()),
			)
		}
	}
}
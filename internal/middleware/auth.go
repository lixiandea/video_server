package middleware

import (
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/lixiandea/video_server/internal/utils"
    "github.com/lixiandea/video_server/pkg/auth"
    "github.com/lixiandea/video_server/pkg/logging"
    "go.uber.org/zap"
    "golang.org/x/time/rate"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            utils.Fail(c, http.StatusUnauthorized, "Authorization header required")
            return
        }

        // Extract token from "Bearer {token}" format
        tokenStr := ""
        if strings.HasPrefix(authHeader, "Bearer ") {
            tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
        } else {
            utils.Fail(c, http.StatusUnauthorized, "Invalid authorization format")
            return
        }

        claims, err := auth.ParseJWT(tokenStr)
        if err != nil {
            logging.GetLogger().Warn("Invalid token", zap.Error(err))
            utils.Fail(c, http.StatusUnauthorized, "Invalid or expired token")
            return
        }

        // Set user info in context
        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        c.Set("session_id", claims.SessionID)

        c.Next()
    }
}

// StreamAuthMiddleware 仅用于播放路由。媒体元素（<video src>）无法设置请求头，
// 因此除标准 Authorization 头外，还接受 URL 上的短时效播放令牌。
func StreamAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        videoID := c.Param("video_id")

        if authHeader := c.GetHeader("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
            if claims, err := auth.ParseJWT(strings.TrimPrefix(authHeader, "Bearer ")); err == nil {
                c.Set("user_id", claims.UserID)
                c.Set("username", claims.Username)
                c.Set("session_id", claims.SessionID)
                c.Next()
                return
            } else {
                logging.GetLogger().Warn("Invalid bearer token on stream route", zap.Error(err))
            }
        }

        if tokenStr := c.Query("token"); tokenStr != "" {
            if claims, err := auth.ParseStreamToken(tokenStr, videoID); err == nil {
                c.Set("user_id", claims.UserID)
                c.Set("stream_token", true)
                c.Next()
                return
            } else {
                logging.GetLogger().Warn("Invalid stream token",
                    zap.String("video_id", videoID),
                    zap.Error(err))
            }
        }

        utils.Fail(c, http.StatusUnauthorized, "Valid access token or stream token required")
    }
}

// RateLimiter middleware for rate limiting
type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
    return &RateLimiter{
        limiters: make(map[string]*rate.Limiter),
        rate:     r,
        burst:    burst,
    }
}

// getLimiter returns or creates a rate limiter for the given key
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
    rl.mu.RLock()
    limiter, exists := rl.limiters[key]
    rl.mu.RUnlock()

    if exists {
        return limiter
    }

    rl.mu.Lock()
    defer rl.mu.Unlock()

    // Double-check after acquiring write lock
    if limiter, exists := rl.limiters[key]; exists {
        return limiter
    }

    limiter = rate.NewLimiter(rl.rate, rl.burst)
    rl.limiters[key] = limiter
    return limiter
}

// Middleware returns the rate limiting middleware function
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.ClientIP()
        limiter := rl.getLimiter(key)

        if !limiter.Allow() {
            logging.GetLogger().Warn("Rate limit exceeded", zap.String("ip", key))
            utils.Fail(c, http.StatusTooManyRequests, "Too many requests")
            return
        }

        c.Next()
    }
}

// RateLimiterConfig 限流配置
type RateLimiterConfig struct {
	Rate  float64 // 每秒请求数
	Burst int     // 突发请求数
}

// DefaultRateLimitConfig 默认限流配置
var DefaultRateLimitConfig = RateLimiterConfig{
	Rate:  10, // 10 requests per second
	Burst: 20, // burst of 20 requests
}

// rateLimiterWithConfig 带配置的限流器
var rateLimiterWithConfig *RateLimiter

// InitRateLimit 初始化限流配置
func InitRateLimit(config RateLimiterConfig) {
	rateLimiterWithConfig = NewRateLimiter(rate.Limit(config.Rate), config.Burst)
}

// RateLimitMiddleware returns the rate limiter middleware
func RateLimitMiddleware() gin.HandlerFunc {
	// 如果没有初始化，使用默认配置
	if rateLimiterWithConfig == nil {
		rateLimiterWithConfig = NewRateLimiter(rate.Limit(DefaultRateLimitConfig.Rate), DefaultRateLimitConfig.Burst)
	}
	return rateLimiterWithConfig.Middleware()
}

// Cleanup old limiters periodically to prevent memory leak
func init() {
    go func() {
        ticker := time.NewTicker(10 * time.Minute)
        defer ticker.Stop()
        for range ticker.C {
            cleanupRateLimiters()
        }
    }()
}

func cleanupRateLimiters() {
    // In a production system, you would track last access time
    // and remove stale limiters
}

// CORSMiddleware only sends CORS headers for allowlisted origins. Combining a
// wildcard origin with credentials is invalid, so "*" is handled separately and
// never receives Access-Control-Allow-Credentials.
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
    allowed := make(map[string]struct{}, len(allowedOrigins))
    allowAll := false
    for _, origin := range allowedOrigins {
        switch origin = strings.TrimSpace(origin); origin {
        case "":
        case "*":
            allowAll = true
        default:
            allowed[origin] = struct{}{}
        }
    }

    return func(c *gin.Context) {
        origin := c.GetHeader("Origin")
        _, explicitlyAllowed := allowed[origin]

        switch {
        case origin == "":
            // 非浏览器请求，不需要 CORS 头
        case allowAll:
            c.Header("Access-Control-Allow-Origin", "*")
        case explicitlyAllowed:
            c.Header("Access-Control-Allow-Origin", origin)
            c.Header("Access-Control-Allow-Credentials", "true")
        }

        if origin != "" && (allowAll || explicitlyAllowed) {
            c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
            c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
            c.Header("Vary", "Origin")
        }

        if c.Request.Method == "OPTIONS" {
            if origin != "" && !allowAll && !explicitlyAllowed {
                c.AbortWithStatus(http.StatusForbidden)
                return
            }
            c.AbortWithStatus(http.StatusNoContent)
            return
        }

        c.Next()
    }
}

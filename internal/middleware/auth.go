package middleware

import (
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lixiandea/video_server/internal/utils"
	"github.com/lixiandea/video_server/pkg/auth"
	apperrors "github.com/lixiandea/video_server/pkg/errors"
	"github.com/lixiandea/video_server/pkg/logging"
	"github.com/lixiandea/video_server/pkg/metrics"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// SessionValidator 校验访问令牌中的会话是否仍然有效（未被登出且未过期）。
// JWT 自身无法撤销，因此令牌只携带 session_id，真实状态以会话记录为准。
type SessionValidator interface {
	ValidateSession(sessionID string, userID uint) error
}

// validateSession 在签名校验通过后追加会话校验。
// 会话类错误按 401 返回，数据库等故障按 500 返回，避免把故障误报成「令牌无效」。
func validateSession(sessions SessionValidator, c *gin.Context, claims *auth.Claims) bool {
	if sessions == nil {
		logging.GetLogger().Error("Session validator is not configured")
		utils.Fail(c, http.StatusInternalServerError, "Authentication is misconfigured")
		return false
	}

	if err := sessions.ValidateSession(claims.SessionID, claims.UserID); err != nil {
		logging.GetLogger().Warn("Session validation failed",
			zap.Uint("user_id", claims.UserID),
			zap.Error(err))
		utils.Fail(c, apperrors.StatusCode(err), apperrors.Message(err, "Invalid or expired session"))
		return false
	}

	return true
}

// AuthMiddleware validates JWT tokens and the session they were issued for
func AuthMiddleware(sessions SessionValidator) gin.HandlerFunc {
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

		if !validateSession(sessions, c, claims) {
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
func StreamAuthMiddleware(sessions SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		videoID := c.Param("video_id")

		if authHeader := c.GetHeader("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			if claims, err := auth.ParseJWT(strings.TrimPrefix(authHeader, "Bearer ")); err == nil {
				if !validateSession(sessions, c, claims) {
					return
				}
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

// limiterEntry 记录单个客户端的限流器与最近使用时间。
// lastSeen 用原子值保存，使热路径只持读锁即可刷新时间戳，避免每个请求都抢写锁。
type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64 // Unix 秒
}

// RateLimiter middleware for rate limiting
type RateLimiter struct {
	limiters map[string]*limiterEntry
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*limiterEntry),
		rate:     r,
		burst:    burst,
	}
}

// getLimiter returns or creates a rate limiter for the given key
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	now := time.Now().Unix()

	rl.mu.RLock()
	entry, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		entry.lastSeen.Store(now)
		return entry.limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if entry, exists = rl.limiters[key]; exists {
		entry.lastSeen.Store(now)
		return entry.limiter
	}

	entry = &limiterEntry{limiter: rate.NewLimiter(rl.rate, rl.burst)}
	entry.lastSeen.Store(now)
	rl.limiters[key] = entry

	return entry.limiter
}

// Size 返回当前跟踪的客户端数量。
func (rl *RateLimiter) Size() int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return len(rl.limiters)
}

// Cleanup 回收空闲超过 ttl 的限流器，返回回收数量。
func (rl *RateLimiter) Cleanup(ttl time.Duration) int {
	cutoff := time.Now().Add(-ttl).Unix()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	removed := 0
	for key, entry := range rl.limiters {
		if entry.lastSeen.Load() < cutoff {
			delete(rl.limiters, key)
			removed++
		}
	}

	return removed
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

const (
	// limiterTTL 限流器空闲多久后被回收。10 分钟足以覆盖单个客户端的连续操作。
	limiterTTL = 10 * time.Minute
	// limiterCleanupInterval 回收扫描周期
	limiterCleanupInterval = 5 * time.Minute
)

// Cleanup old limiters periodically to prevent memory leak
func init() {
	go func() {
		ticker := time.NewTicker(limiterCleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			cleanupRateLimiters()
		}
	}()
}

// cleanupRateLimiters 回收空闲限流器，避免 map 随客户端数量无限增长（P2-3）。
func cleanupRateLimiters() {
	rl := rateLimiterWithConfig
	if rl == nil {
		return
	}

	if removed := rl.Cleanup(limiterTTL); removed > 0 {
		logging.GetLogger().Info("Evicted idle rate limiters", zap.Int("count", removed))
	}
	metrics.SetRateLimiterEntries(rl.Size())
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

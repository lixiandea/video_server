package middleware

import (
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
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
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        // Extract token from "Bearer {token}" format
        tokenStr := ""
        if strings.HasPrefix(authHeader, "Bearer ") {
            tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
        } else {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
            c.Abort()
            return
        }

        claims, err := auth.ParseJWT(tokenStr)
        if err != nil {
            logging.GetLogger().Warn("Invalid token", zap.Error(err))
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
            c.Abort()
            return
        }

        // Set user info in context
        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        c.Set("session_id", claims.SessionID)

        c.Next()
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
            c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
            c.Abort()
            return
        }

        c.Next()
    }
}

// Global rate limiter instance (10 requests per second, burst of 20)
var globalRateLimiter = NewRateLimiter(10, 20)

// RateLimitMiddleware returns the global rate limiter middleware
func RateLimitMiddleware() gin.HandlerFunc {
    return globalRateLimiter.Middleware()
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

// CORSMiddleware adds CORS headers
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Credentials", "true")
        c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
        c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}
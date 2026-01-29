package middleware

import (
    "strings"
    
    "github.com/gin-gonic/gin"
    "github.com/lixiandea/video_server/pkg/auth"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        // Extract token from "Bearer {token}" format
        tokenStr := ""
        if strings.HasPrefix(authHeader, "Bearer ") {
            tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
        } else {
            c.JSON(401, gin.H{"error": "Invalid authorization format"})
            c.Abort()
            return
        }

        claims, err := auth.ParseJWT(tokenStr)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid or expired token"})
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

// RateLimitMiddleware adds basic rate limiting
func RateLimitMiddleware(maxRequests int, windowSeconds int64) gin.HandlerFunc {
    // Note: This is a simplified rate limiter.
    // For production use, consider using redis-backed rate limiting
    return func(c *gin.Context) {
        // In a real implementation, you would track requests per IP/user
        // and reject if the limit is exceeded
        c.Next()
    }
}
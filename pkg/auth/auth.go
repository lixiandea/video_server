package auth

import (
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
)

// jwtSecret holds the signing key. It stays empty until InitJWT succeeds, so a
// missing JWT_SECRET can never fall back to a predictable default.
var jwtSecret []byte

// InitJWT validates and loads the JWT signing secret.
func InitJWT(secret string) error {
    secret = strings.TrimSpace(secret)
    if secret == "" {
        return errors.New("JWT_SECRET is required but not set")
    }
    if len(secret) < 32 {
        return fmt.Errorf("JWT_SECRET must be at least 32 characters, got %d", len(secret))
    }
    jwtSecret = []byte(secret)
    return nil
}

// 令牌用途（JWT Subject）。不同用途的令牌不能互相顶替使用。
const (
    // SubjectAccessToken 常规 API 访问令牌
    SubjectAccessToken = "access-token"
    // SubjectStreamToken 仅用于播放单个视频的短时效令牌
    SubjectStreamToken = "stream-token"
)

// DefaultStreamTokenTTL 播放令牌默认有效期。
// 媒体元素无法携带请求头，令牌只能走 URL，因此必须短时效且绑定单个视频。
const DefaultStreamTokenTTL = 2 * time.Hour

type Claims struct {
    UserID    uint   `json:"user_id"`
    Username  string `json:"username"`
    SessionID string `json:"session_id"`
    // VideoID 仅播放令牌使用，用于把令牌限定到单个视频
    VideoID string `json:"video_id,omitempty"`
    jwt.RegisteredClaims
}

// HashPassword hashes the given password
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

// CheckPasswordHash compares a password with its hash
func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

// GenerateJWT generates a JWT token for the user
func GenerateJWT(userID uint, username, sessionID string) (string, error) {
    if len(jwtSecret) == 0 {
        return "", errors.New("JWT secret not initialized")
    }

    claims := Claims{
        UserID:    userID,
        Username:  username,
        SessionID: sessionID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Subject:   SubjectAccessToken,
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(jwtSecret)
}

// parse 校验签名并确认令牌用途（Subject）符合预期。
func parse(tokenStr, expectedSubject string) (*Claims, error) {
    if len(jwtSecret) == 0 {
        return nil, errors.New("JWT secret not initialized")
    }

    token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
        }
        return jwtSecret, nil
    })

    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }
    if expectedSubject != "" && claims.Subject != expectedSubject {
        return nil, fmt.Errorf("unexpected token subject %q", claims.Subject)
    }

    return claims, nil
}

// ParseJWT parses and validates an access token.
// 播放令牌（SubjectStreamToken）在此被拒绝，避免只能看视频的令牌被当成完整访问令牌使用。
func ParseJWT(tokenStr string) (*Claims, error) {
    return parse(tokenStr, SubjectAccessToken)
}

// GenerateStreamToken 为 <video src> 签发播放令牌。
// 令牌有效期短，且通过 video_id 限定到单个视频。
func GenerateStreamToken(userID uint, videoID string, ttl time.Duration) (string, error) {
    if len(jwtSecret) == 0 {
        return "", errors.New("JWT secret not initialized")
    }
    if videoID == "" {
        return "", errors.New("video id is required")
    }
    if ttl <= 0 {
        ttl = DefaultStreamTokenTTL
    }

    claims := Claims{
        UserID:  userID,
        VideoID: videoID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Subject:   SubjectStreamToken,
        },
    }

    return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

// ParseStreamToken 校验播放令牌，并要求其确实签发给 videoID 对应的视频。
func ParseStreamToken(tokenStr, videoID string) (*Claims, error) {
    claims, err := parse(tokenStr, SubjectStreamToken)
    if err != nil {
        return nil, err
    }
    if claims.VideoID != videoID {
        return nil, errors.New("stream token does not belong to this video")
    }

    return claims, nil
}

// GenerateSessionID creates a unique session identifier
func GenerateSessionID() string {
    return uuid.New().String()
}

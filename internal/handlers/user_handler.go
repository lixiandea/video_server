package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lixiandea/video_server/internal/config"
	"github.com/lixiandea/video_server/internal/services"
	"github.com/lixiandea/video_server/internal/utils"
	"github.com/lixiandea/video_server/pkg/auth"
	"github.com/lixiandea/video_server/pkg/logging"
	"github.com/lixiandea/video_server/pkg/metrics"
	"github.com/lixiandea/video_server/pkg/validation"
	"go.uber.org/zap"
)

type UserHandler struct {
	userService    *services.UserService
	sessionService *services.SessionService
	cfg            *config.Config
}

func NewUserHandler(cfg *config.Config) *UserHandler {
	return &UserHandler{
		userService:    services.NewUserService(),
		sessionService: services.NewSessionService(),
		cfg:            cfg,
	}
}

const refreshCookieName = "video_server_refresh"

func (h *UserHandler) setRefreshCookie(c *gin.Context, sessionID, refreshToken string) {
	maxAge := int(auth.RefreshTokenTTL.Seconds())
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		refreshCookieName,
		sessionID+"."+refreshToken,
		maxAge,
		"/api/v1",
		h.cfg.Server.CookieDomain,
		h.cfg.Server.CookieSecure,
		true,
	)
}

func (h *UserHandler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshCookieName, "", -1, "/api/v1", h.cfg.Server.CookieDomain, h.cfg.Server.CookieSecure, true)
}

func (h *UserHandler) readRefreshCookie(c *gin.Context) (string, string, bool) {
	value, err := c.Cookie(refreshCookieName)
	if err != nil {
		return "", "", false
	}
	sessionID, token, ok := strings.Cut(value, ".")
	return sessionID, token, ok && sessionID != "" && token != ""
}

func (h *UserHandler) startSession(c *gin.Context, userID uint, username string) (string, error) {
	sessionID := auth.GenerateSessionID()
	refreshToken, _, err := auth.GenerateRefreshToken()
	if err != nil {
		return "", err
	}
	if err := h.sessionService.CreateSession(
		userID,
		sessionID,
		refreshToken,
		c.GetHeader("User-Agent"),
		c.ClientIP(),
	); err != nil {
		return "", err
	}
	token, err := auth.GenerateJWT(userID, username, sessionID)
	if err != nil {
		return "", err
	}
	h.setRefreshCookie(c, sessionID, refreshToken)
	return token, nil
}

func (h *UserHandler) originAllowed(c *gin.Context) bool {
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if origin == "" {
		return true
	}
	for _, allowed := range h.cfg.Server.CORS.AllowedOrigins {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	return false
}

func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logging.GetLogger().Error("Invalid register request format", zap.Error(err))
		utils.Fail(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate input
	if err := validation.ValidateUsername(req.Username); err != nil {
		logging.GetLogger().Warn("Username validation failed",
			zap.String("username", req.Username),
			zap.Error(err))
		utils.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validation.ValidatePassword(req.Password); err != nil {
		logging.GetLogger().Warn("Password validation failed", zap.Error(err))
		utils.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validation.ValidateEmail(req.Email); err != nil {
		logging.GetLogger().Warn("Email validation failed", zap.Error(err))
		utils.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// Check if user already exists
	_, err := h.userService.GetUserByUsername(req.Username)
	if err == nil {
		logging.GetLogger().Info("Username already exists", zap.String("username", req.Username))
		utils.Fail(c, http.StatusConflict, "Username already exists")
		return
	}

	if taken, err := h.userService.IsEmailTaken(req.Email, 0); err != nil {
		logging.GetLogger().Error("Failed to check email availability", zap.Error(err))
		utils.FailFrom(c, err, "Failed to create user")
		return
	} else if taken {
		utils.Fail(c, http.StatusConflict, "Email already exists")
		return
	}

	user, err := h.userService.CreateUser(req.Username, req.Password, req.Email)
	if err != nil {
		logging.GetLogger().Error("Failed to create user",
			zap.String("username", req.Username),
			zap.Error(err))
		utils.FailFrom(c, err, "Failed to create user")
		return
	}

	token, err := h.startSession(c, user.ID, user.LoginName)
	if err != nil {
		logging.GetLogger().Error("Failed to create session", zap.Error(err))
		utils.FailFrom(c, err, "Failed to create session")
		return
	}

	// Record metrics
	metrics.IncActiveUsers()
	metrics.IncAuthSuccess()

	logging.GetLogger().Info("User registered successfully",
		zap.Uint("user_id", user.ID),
		zap.String("username", user.LoginName))

	c.JSON(http.StatusCreated, gin.H{
		"message":    "User created successfully",
		"user_id":    user.ID,
		"username":   user.LoginName,
		"role":       user.Role,
		"token":      token,
		"expires_in": int(auth.AccessTokenTTL.Seconds()),
		"user": gin.H{
			"id":       user.ID,
			"username": user.LoginName,
			"role":     user.Role,
		},
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logging.GetLogger().Error("Invalid login request format", zap.Error(err))
		utils.Fail(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	user, err := h.userService.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		logging.GetLogger().Warn("Authentication failed",
			zap.String("username", req.Username),
			zap.Error(err))
		metrics.IncAuthFailure()
		utils.Fail(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token, err := h.startSession(c, user.ID, user.LoginName)
	if err != nil {
		logging.GetLogger().Error("Failed to create session", zap.Error(err))
		utils.FailFrom(c, err, "Failed to create session")
		return
	}

	// Record metrics
	metrics.IncActiveUsers()
	metrics.IncAuthSuccess()

	logging.GetLogger().Info("User logged in successfully",
		zap.Uint("user_id", user.ID),
		zap.String("username", user.LoginName))

	c.JSON(http.StatusOK, gin.H{
		"message":    "Login successful",
		"user_id":    user.ID,
		"username":   user.LoginName,
		"role":       user.Role,
		"token":      token,
		"expires_in": int(auth.AccessTokenTTL.Seconds()),
		"user": gin.H{
			"id":       user.ID,
			"username": user.LoginName,
			"role":     user.Role,
		},
	})
}

// Refresh 使用 HttpOnly Cookie 换取新的短期访问令牌，并轮换刷新令牌。
func (h *UserHandler) Refresh(c *gin.Context) {
	if !h.originAllowed(c) {
		utils.Fail(c, http.StatusForbidden, "Origin is not allowed")
		return
	}
	sessionID, refreshToken, ok := h.readRefreshCookie(c)
	if !ok {
		utils.Fail(c, http.StatusUnauthorized, "Refresh session required")
		return
	}

	newRefresh, err := h.sessionService.RotateRefreshToken(sessionID, refreshToken)
	if err != nil {
		h.clearRefreshCookie(c)
		utils.FailFrom(c, err, "Refresh session is invalid")
		return
	}

	userID, err := h.sessionService.GetSessionUserID(sessionID)
	if err != nil {
		h.clearRefreshCookie(c)
		utils.FailFrom(c, err, "Refresh session is invalid")
		return
	}
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		h.clearRefreshCookie(c)
		utils.FailFrom(c, err, "Account is unavailable")
		return
	}
	accessToken, err := auth.GenerateJWT(user.ID, user.LoginName, sessionID)
	if err != nil {
		utils.FailFrom(c, err, "Failed to create access token")
		return
	}

	h.setRefreshCookie(c, sessionID, newRefresh)
	c.JSON(http.StatusOK, gin.H{
		"token":      accessToken,
		"expires_in": int(auth.AccessTokenTTL.Seconds()),
		"user": gin.H{
			"id":       user.ID,
			"username": user.LoginName,
			"role":     user.Role,
		},
	})
}

// CookieLogout 允许浏览器仅凭 HttpOnly 刷新 Cookie 注销。
func (h *UserHandler) CookieLogout(c *gin.Context) {
	if !h.originAllowed(c) {
		utils.Fail(c, http.StatusForbidden, "Origin is not allowed")
		return
	}
	sessionID, _, ok := h.readRefreshCookie(c)
	if ok {
		_ = h.sessionService.RevokeSession(sessionID)
	}
	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "Signed out successfully"})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.userService.GetUserByID(userID.(uint))
	if err != nil {
		utils.FailFrom(c, err, "Failed to get user profile")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":    user.ID,
		"username":   user.LoginName,
		"role":       user.Role,
		"email":      user.Email,
		"bio":        user.Bio,
		"created_at": user.CreatedAt,
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Username        string  `json:"username"`
		CurrentPassword string  `json:"current_password"`
		Password        string  `json:"password"`
		Email           *string `json:"email"`
		Bio             *string `json:"bio"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	user, err := h.userService.GetUserByID(userID.(uint))
	if err != nil {
		utils.FailFrom(c, err, "Failed to update user")
		return
	}
	sensitiveChange := req.Username != "" || req.Password != "" || req.Email != nil
	if sensitiveChange && !auth.CheckPasswordHash(req.CurrentPassword, user.Pwd) {
		utils.Fail(c, http.StatusForbidden, "Current password is required")
		return
	}

	updates := make(map[string]interface{})
	if req.Username != "" {
		if err := validation.ValidateUsername(req.Username); err != nil {
			utils.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		updates["login_name"] = req.Username
	}
	if req.Password != "" {
		if err := validation.ValidatePassword(req.Password); err != nil {
			utils.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		hashedPwd, err := auth.HashPassword(req.Password)
		if err != nil {
			utils.FailFrom(c, err, "Failed to hash password")
			return
		}
		updates["pwd"] = hashedPwd
	}
	if req.Email != nil {
		email := strings.TrimSpace(*req.Email)
		if err := validation.ValidateEmail(email); err != nil {
			utils.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		if taken, err := h.userService.IsEmailTaken(email, userID.(uint)); err != nil {
			utils.FailFrom(c, err, "Failed to update user")
			return
		} else if taken {
			utils.Fail(c, http.StatusConflict, "Email already exists")
			return
		}
		updates["email"] = email
	}
	if req.Bio != nil {
		if err := validation.ValidateBio(*req.Bio); err != nil {
			utils.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		updates["bio"] = *req.Bio
	}

	if len(updates) == 0 {
		utils.Fail(c, http.StatusBadRequest, "No updates provided")
		return
	}

	err = h.userService.UpdateUser(userID.(uint), updates)
	if err != nil {
		utils.FailFrom(c, err, "Failed to update user")
		return
	}

	// 改密后吊销其他设备上的会话，避免旧令牌在有效期内继续可用。
	if _, passwordChanged := updates["pwd"]; passwordChanged {
		if _, err := h.sessionService.RevokeUserSessionsExcept(userID.(uint), c.GetString("session_id")); err != nil {
			logging.GetLogger().Warn("Failed to revoke other sessions", zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// Logout 吊销当前会话。令牌本身仍在有效期内，但再次使用会因会话不存在而返回 401。
func (h *UserHandler) Logout(c *gin.Context) {
	if err := h.sessionService.RevokeSession(c.GetString("session_id")); err != nil {
		utils.FailFrom(c, err, "Failed to sign out")
		return
	}
	h.clearRefreshCookie(c)

	c.JSON(http.StatusOK, gin.H{"message": "Signed out successfully"})
}

// LogoutAll 吊销该用户在其它设备上的会话，保留当前会话继续可用。
func (h *UserHandler) LogoutAll(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	revoked, err := h.sessionService.RevokeUserSessionsExcept(userID.(uint), c.GetString("session_id"))
	if err != nil {
		utils.FailFrom(c, err, "Failed to sign out other devices")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Signed out from other devices",
		"revoked_sessions": revoked,
	})
}

func (h *UserHandler) DeleteAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, http.StatusBadRequest, "Password is required")
		return
	}
	user, err := h.userService.GetUserByID(userID.(uint))
	if err != nil {
		utils.FailFrom(c, err, "Failed to delete account")
		return
	}
	if !auth.CheckPasswordHash(req.Password, user.Pwd) {
		utils.Fail(c, http.StatusForbidden, "Password is incorrect")
		return
	}

	err = h.userService.DeleteUser(userID.(uint))
	if err != nil {
		utils.FailFrom(c, err, "Failed to delete account")
		return
	}
	h.clearRefreshCookie(c)

	c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}

package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		userService:    services.NewUserService(),
		sessionService: services.NewSessionService(),
	}
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

	// Generate JWT token and persist the session so it can be revoked later
	sessionID := auth.GenerateSessionID()
	if err := h.sessionService.CreateSession(user.ID, sessionID); err != nil {
		logging.GetLogger().Error("Failed to create session",
			zap.Uint("user_id", user.ID),
			zap.Error(err))
		utils.FailFrom(c, err, "Failed to create session")
		return
	}
	token, err := auth.GenerateJWT(user.ID, user.LoginName, sessionID)
	if err != nil {
		logging.GetLogger().Error("Failed to generate token", zap.Error(err))
		utils.FailFrom(c, err, "Failed to generate token")
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
		"token":      token,
		"session_id": sessionID,
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

	// Generate JWT token and persist the session so it can be revoked later
	sessionID := auth.GenerateSessionID()
	if err := h.sessionService.CreateSession(user.ID, sessionID); err != nil {
		logging.GetLogger().Error("Failed to create session",
			zap.Uint("user_id", user.ID),
			zap.Error(err))
		utils.FailFrom(c, err, "Failed to create session")
		return
	}
	token, err := auth.GenerateJWT(user.ID, user.LoginName, sessionID)
	if err != nil {
		logging.GetLogger().Error("Failed to generate token", zap.Error(err))
		utils.FailFrom(c, err, "Failed to generate token")
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
		"token":      token,
		"session_id": sessionID,
	})
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
		Username string  `json:"username"`
		Password string  `json:"password"`
		Email    *string `json:"email"`
		Bio      *string `json:"bio"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, http.StatusBadRequest, "Invalid request format")
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

	err := h.userService.UpdateUser(userID.(uint), updates)
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

	err := h.userService.DeleteUser(userID.(uint))
	if err != nil {
		utils.FailFrom(c, err, "Failed to delete account")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}

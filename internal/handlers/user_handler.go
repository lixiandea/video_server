package handlers

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/lixiandea/video_server/internal/services"
    "github.com/lixiandea/video_server/pkg/auth"
    "github.com/lixiandea/video_server/pkg/validation"
)

type UserHandler struct {
    userService *services.UserService
}

func NewUserHandler() *UserHandler {
    return &UserHandler{
        userService: services.NewUserService(),
    }
}

func (h *UserHandler) Register(c *gin.Context) {
    var req struct {
        Username string `json:"username" binding:"required"`
        Password string `json:"password" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
        return
    }

    // Validate input
    if err := validation.ValidateUsername(req.Username); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := validation.ValidatePassword(req.Password); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Check if user already exists
    _, err := h.userService.GetUserByUsername(req.Username)
    if err == nil {
        c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
        return
    }

    user, err := h.userService.CreateUser(req.Username, req.Password)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
        return
    }

    // Generate JWT token
    sessionID := auth.GenerateSessionID()
    token, err := auth.GenerateJWT(user.ID, user.LoginName, sessionID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "message":   "User created successfully",
        "user_id":   user.ID,
        "username":  user.LoginName,
        "token":     token,
        "session_id": sessionID,
    })
}

func (h *UserHandler) Login(c *gin.Context) {
    var req struct {
        Username string `json:"username" binding:"required"`
        Password string `json:"password" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
        return
    }

    user, err := h.userService.AuthenticateUser(req.Username, req.Password)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
        return
    }

    // Generate JWT token
    sessionID := auth.GenerateSessionID()
    token, err := auth.GenerateJWT(user.ID, user.LoginName, sessionID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message":   "Login successful",
        "user_id":   user.ID,
        "username":  user.LoginName,
        "token":     token,
        "session_id": sessionID,
    })
}

func (h *UserHandler) GetProfile(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    user, err := h.userService.GetUserByID(userID.(uint))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user profile"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "user_id":  user.ID,
        "username": user.LoginName,
        "created_at": user.CreatedAt,
    })
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    var req struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
        return
    }

    updates := make(map[string]interface{})
    if req.Username != "" {
        if err := validation.ValidateUsername(req.Username); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        updates["login_name"] = req.Username
    }
    if req.Password != "" {
        if err := validation.ValidatePassword(req.Password); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        hashedPwd, err := auth.HashPassword(req.Password)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
            return
        }
        updates["pwd"] = hashedPwd
    }

    if len(updates) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "No updates provided"})
        return
    }

    err := h.userService.UpdateUser(userID.(uint), updates)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

func (h *UserHandler) DeleteAccount(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    err := h.userService.DeleteUser(userID.(uint))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}
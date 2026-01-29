package main

import (
    "log"
    
    "github.com/gin-gonic/gin"
    "github.com/lixiandea/video_server/internal/config"
    "github.com/lixiandea/video_server/internal/handlers"
    "github.com/lixiandea/video_server/internal/middleware"
    "github.com/lixiandea/video_server/pkg/database"
    "github.com/lixiandea/video_server/pkg/storage"
)

func main() {
    // Load configuration
    cfg := config.LoadConfig()
    
    // Initialize database
    database.InitDatabase(&cfg.Database)
    
    // Initialize storage
    storageService := storage.NewStorageService(&cfg.Storage)
    
    // Set Gin mode based on config
    if cfg.Server.Mode == "release" {
        gin.SetMode(gin.ReleaseMode)
    } else if cfg.Server.Mode == "test" {
        gin.SetMode(gin.TestMode)
    } else {
        gin.SetMode(gin.DebugMode)
    }
    
    // Initialize handlers
    userHandler := handlers.NewUserHandler()
    videoHandler := handlers.NewVideoHandler(storageService, cfg)
    commentHandler := handlers.NewCommentHandler()
    
    // Setup routes
    r := gin.Default()
    
    // Global middleware
    r.Use(middleware.CORSMiddleware())
    r.Use(gin.Logger())
    r.Use(gin.Recovery())
    
    // Health check endpoint
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "OK", "service": "api-server"})
    })
    
    // Public routes
    public := r.Group("/api/v1")
    {
        public.POST("/users/register", userHandler.Register)
        public.POST("/users/login", userHandler.Login)
    }
    
    // Protected routes
    protected := r.Group("/api/v1")
    protected.Use(middleware.AuthMiddleware())
    {
        // User routes
        protected.GET("/users/profile", userHandler.GetProfile)
        protected.PUT("/users/profile", userHandler.UpdateProfile)
        protected.DELETE("/users/account", userHandler.DeleteAccount)
        
        // Video routes
        protected.POST("/videos/upload", videoHandler.UploadVideo)
        protected.GET("/videos/:video_id", videoHandler.GetVideoInfo)
        protected.GET("/videos/:video_id/stream", videoHandler.GetVideoStream)
        protected.GET("/users/videos", videoHandler.GetUserVideos)
        protected.DELETE("/videos/:video_id", videoHandler.DeleteVideo)
        
        // Comment routes
        protected.POST("/videos/:video_id/comments", commentHandler.AddComment)
        protected.GET("/videos/:video_id/comments", commentHandler.GetComments)
        protected.GET("/comments/:comment_id", commentHandler.GetComment)
        protected.PUT("/comments/:comment_id", commentHandler.UpdateComment)
        protected.DELETE("/comments/:comment_id", commentHandler.DeleteComment)
    }
    
    log.Printf("Starting API server on port %s", cfg.Server.Port)
    log.Printf("Server mode: %s", cfg.Server.Mode)
    
    // Handle graceful shutdown signals
    if err := r.Run(":" + cfg.Server.Port); err != nil {
        log.Fatal("Failed to start server: ", err)
    }
}
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lixiandea/video_server/internal/config"
	"github.com/lixiandea/video_server/internal/handlers"
	"github.com/lixiandea/video_server/internal/middleware"
	"github.com/lixiandea/video_server/pkg/database"
	"github.com/lixiandea/video_server/pkg/logging"
	"github.com/lixiandea/video_server/pkg/metrics"
	"github.com/lixiandea/video_server/pkg/redis"
	"github.com/lixiandea/video_server/pkg/storage"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize structured logging with default settings
	if err := logging.InitLogger("info", "stdout"); err != nil {
		log.Fatal("Failed to initialize logger: ", err)
	}
	defer logging.Sync()

	// Initialize database
	database.InitDatabase(&cfg.Database)

	// Initialize Redis
	redisCfg := &redis.Config{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  time.Duration(cfg.Redis.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.Redis.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Redis.WriteTimeout) * time.Second,
	}
	if err := redis.InitRedis(redisCfg); err != nil {
		log.Printf("Warning: Redis connection failed, continuing without cache: %v", err)
	} else {
		defer redis.Close()
	}

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
	r := gin.New()

	// Global middleware
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.ErrorLoggingMiddleware())
	r.Use(middleware.RateLimitMiddleware()) // Add rate limiting
	r.Use(metrics.MetricsMiddleware())
	r.Use(gin.Recovery())

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "OK", "service": "api-server"})
	})

	// Metrics endpoint
	metrics.RegisterMetricsEndpoint(r)

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

	logging.GetLogger().Info("API server starting",
		zap.String("port", cfg.Server.Port),
		zap.String("mode", cfg.Server.Mode),
	)

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	// Start metrics collection
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				metrics.CollectSystemMetrics()
			}
		}
	}()

	// Run server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.GetLogger().Error("Server failed to start", zap.Error(err))
		}
	}()

	logging.GetLogger().Info("API server started successfully",
		zap.String("port", cfg.Server.Port),
		zap.String("mode", cfg.Server.Mode),
	)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logging.GetLogger().Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logging.GetLogger().Error("Server forced to shutdown", zap.Error(err))
	}

	logging.GetLogger().Info("Server exited")
}
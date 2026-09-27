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
	"github.com/lixiandea/video_server/internal/services"
	"github.com/lixiandea/video_server/pkg/auth"
	"github.com/lixiandea/video_server/pkg/database"
	"github.com/lixiandea/video_server/pkg/logging"
	"github.com/lixiandea/video_server/pkg/metrics"
	"github.com/lixiandea/video_server/pkg/queue"
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

	// The JWT secret is mandatory: fail fast instead of signing with a weak default.
	if err := auth.InitJWT(cfg.Server.JWTSecret); err != nil {
		log.Fatal("Invalid JWT configuration: ", err)
	}

	// Initialize database
	if _, err := database.InitDatabase(&cfg.Database); err != nil {
		logging.GetLogger().Error("Failed to initialize database", zap.Error(err))
		log.Fatal("Database initialization failed: ", err)
	}

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

	// Initialize task queue and transcode service
	redisClient := redis.GetClient()
	var taskQueue *queue.TaskQueue
	var transcodeService *services.TranscodeService
	
	if redisClient != nil {
		taskQueue = queue.NewTaskQueue(redisClient, queue.DefaultQueueConfig)
		transcodeService = services.NewTranscodeService(taskQueue, cfg.Storage.VideoDir)
		log.Println("Transcode queue initialized")
	} else {
		log.Println("Warning: Redis not available, transcode queue disabled")
	}

	// Set Gin mode based on config
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.Server.Mode == "test" {
		gin.SetMode(gin.TestMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize handlers
	sessionService := services.NewSessionService()
	userHandler := handlers.NewUserHandler()
	videoHandler := handlers.NewVideoHandler(storageService, cfg, transcodeService, taskQueue)
	commentHandler := handlers.NewCommentHandler()
	var transcodeHandler *handlers.TranscodeHandler
	if transcodeService != nil && taskQueue != nil {
		transcodeHandler = handlers.NewTranscodeHandler(transcodeService, taskQueue)
	}

	// Initialize rate limiter with config
	if cfg.Server.RateLimit.Enabled {
		middleware.InitRateLimit(middleware.RateLimiterConfig{
			Rate:  cfg.Server.RateLimit.Rate,
			Burst: cfg.Server.RateLimit.Burst,
		})
	}

	// Setup routes
	r := gin.New()
	r.MaxMultipartMemory = 8 << 20 // 8 MB，其余部分落盘，避免大文件常驻内存

	// Global middleware
	r.Use(middleware.CORSMiddleware(cfg.Server.CORS.AllowedOrigins))
	r.Use(middleware.RequestIDMiddleware()) // Add request ID tracking
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
		// 视频列表 / 搜索：首页与列表页对未登录用户可见
		public.GET("/videos", videoHandler.ListVideos)
		// 封面：<img> 无法携带 Authorization 头，与播放令牌同理需公开
		public.GET("/videos/:video_id/cover", videoHandler.GetVideoCover)
	}

	// Stream routes: <video src> cannot send headers, so StreamAuthMiddleware
	// accepts either a Bearer token or a short-lived ?token= playback token.
	stream := r.Group("/api/v1")
	stream.Use(middleware.StreamAuthMiddleware(sessionService))
	{
		stream.GET("/videos/:video_id/stream", videoHandler.GetVideoStream)
	}

	// Protected routes
	protected := r.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(sessionService))
	{
		// User routes
		protected.GET("/users/profile", userHandler.GetProfile)
		protected.PUT("/users/profile", userHandler.UpdateProfile)
		protected.DELETE("/users/account", userHandler.DeleteAccount)
		protected.POST("/users/logout", userHandler.Logout)
		protected.POST("/users/logout-all", userHandler.LogoutAll)

		// Video routes
		protected.POST("/videos/upload", videoHandler.UploadVideo)
		protected.GET("/videos/:video_id", videoHandler.GetVideoInfo)
		protected.GET("/videos/:video_id/related", videoHandler.GetRelatedVideos)
		protected.GET("/videos/:video_id/stream-token", videoHandler.CreateStreamToken)
		protected.GET("/users/videos", videoHandler.GetUserVideos)
		protected.DELETE("/videos/:video_id", videoHandler.DeleteVideo)

		// Comment routes
		protected.POST("/videos/:video_id/comments", commentHandler.AddComment)
		protected.GET("/videos/:video_id/comments", commentHandler.GetComments)
		protected.GET("/comments/:comment_id", commentHandler.GetComment)
		protected.PUT("/comments/:comment_id", commentHandler.UpdateComment)
		protected.DELETE("/comments/:comment_id", commentHandler.DeleteComment)

		// Transcode routes (if available)
		if transcodeHandler != nil {
			protected.GET("/transcode/tasks/:task_id", transcodeHandler.GetTaskStatus)
			protected.GET("/transcode/queue/stats", transcodeHandler.GetQueueStats)
			protected.GET("/transcode/tasks", transcodeHandler.ListTasks)
			protected.POST("/transcode/tasks/:task_id/retry", transcodeHandler.RetryTask)
			protected.POST("/transcode/tasks/:task_id/cancel", transcodeHandler.CancelTask)
		}
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

	// 清理过期会话：令牌过期后记录即可回收，避免 sessions 表持续增长。
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if removed, err := sessionService.DeleteExpiredSessions(); err != nil {
					logging.GetLogger().Warn("Failed to delete expired sessions", zap.Error(err))
				} else if removed > 0 {
					logging.GetLogger().Info("Deleted expired sessions", zap.Int64("count", removed))
				}
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

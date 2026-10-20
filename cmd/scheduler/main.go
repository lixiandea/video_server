package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lixiandea/video_server/internal/config"
	"github.com/lixiandea/video_server/internal/services"
	"github.com/lixiandea/video_server/pkg/database"
	"github.com/lixiandea/video_server/pkg/metrics"
	"github.com/lixiandea/video_server/pkg/queue"
	"github.com/lixiandea/video_server/pkg/redis"
	"github.com/lixiandea/video_server/pkg/storage"
	"github.com/spf13/viper"
)

func main() {
	// 加载调度器专用配置
	cfg := loadSchedulerConfig()
	if err := cfg.ValidateProduction(); err != nil {
		log.Fatal("Unsafe production configuration: ", err)
	}

	// Initialize database
	if _, err := database.InitDatabase(&cfg.Database); err != nil {
		log.Fatal("Database initialization failed: ", err)
	}

	// Initialize storage
	storageService, err := storage.NewStorageService(&cfg.Storage)
	if err != nil {
		log.Fatal("Storage initialization failed: ", err)
	}

	// Set Gin mode
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize services
	auditService := services.NewAuditService(database.GetDB())

	redisCfg := &redis.Config{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  time.Duration(cfg.Redis.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.Redis.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Redis.WriteTimeout) * time.Second,
		TLSEnabled:   cfg.Redis.TLSEnabled,
		TLSCAPath:    cfg.Redis.TLSCAPath,
	}
	var taskQueue *queue.TaskQueue
	if err := redis.InitRedis(redisCfg); err != nil {
		log.Printf("Warning: Redis unavailable, task recovery disabled: %v", err)
	} else {
		defer redis.Close()
		taskQueue = queue.NewTaskQueue(redis.GetClient(), queue.DefaultQueueConfig)
	}
	videoService := services.NewVideoService()
	if taskQueue != nil {
		go startTaskRecovery(videoService, taskQueue)
	}

	// Start cleanup worker
	go startCleanupWorker(videoService, storageService, auditService)
	go startReplicationWorker(videoService, storageService)

	// Setup basic API for health checks and manual triggers
	r := gin.Default()
	if err := r.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		log.Fatal("Invalid trusted proxy configuration: ", err)
	}
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "OK", "service": "scheduler"})
	})

	// /metrics：与 api-server 使用同一套保护策略（令牌或仅内网）
	metrics.RegisterMetricsEndpoint(r, cfg.Server.MetricsToken)

	r.POST("/cleanup", func(c *gin.Context) {
		performCleanup(videoService, storageService, auditService)
		c.JSON(200, gin.H{"status": "cleanup initiated"})
	})

	log.Printf("Starting scheduler on port %s", cfg.Server.Port)

	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("Failed to start scheduler: ", err)
	}
}

// 加载调度器专用配置
func loadSchedulerConfig() *config.Config {
	config.LoadSecretFiles()
	// Enable environment variable binding
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Bind environment variables for database
	viper.BindEnv("database.host", "DB_HOST")
	viper.BindEnv("database.port", "DB_PORT")
	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.password", "DB_PASSWORD")
	viper.BindEnv("database.name", "DB_NAME")
	viper.BindEnv("database.tls_ca_path", "DB_TLS_CA_PATH")
	viper.BindEnv("database.log_level", "DB_LOG_LEVEL")
	viper.BindEnv("redis.addr", "REDIS_ADDR")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("redis.tls_enabled", "REDIS_TLS_ENABLED")
	viper.BindEnv("redis.tls_ca_path", "REDIS_TLS_CA_PATH")

	// Bind storage directories so the cleanup job uses the mounted volume
	viper.BindEnv("storage.video_dir", "STORAGE_VIDEO_DIR")
	viper.BindEnv("storage.temp_dir", "STORAGE_TEMP_DIR")
	viper.BindEnv("storage.backend", "STORAGE_BACKEND")
	viper.BindEnv("storage.s3_endpoint", "S3_ENDPOINT")
	viper.BindEnv("storage.s3_region", "S3_REGION")
	viper.BindEnv("storage.s3_bucket", "S3_BUCKET")
	viper.BindEnv("storage.s3_access_key", "S3_ACCESS_KEY")
	viper.BindEnv("storage.s3_secret_key", "S3_SECRET_KEY")
	viper.BindEnv("storage.s3_use_ssl", "S3_USE_SSL")
	viper.BindEnv("storage.s3_backup_endpoint", "S3_BACKUP_ENDPOINT")
	viper.BindEnv("storage.s3_backup_region", "S3_BACKUP_REGION")
	viper.BindEnv("storage.s3_backup_bucket", "S3_BACKUP_BUCKET")
	viper.BindEnv("storage.s3_backup_access_key", "S3_BACKUP_ACCESS_KEY")
	viper.BindEnv("storage.s3_backup_secret_key", "S3_BACKUP_SECRET_KEY")
	viper.BindEnv("server.metrics_token", "METRICS_TOKEN")
	viper.BindEnv("server.mode", "SERVER_MODE", "GIN_MODE")

	// Set defaults
	viper.SetDefault("server.port", "8089")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.trusted_proxies", []string{"127.0.0.1", "::1"})
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.user", "root")
	viper.SetDefault("database.password", "password")
	viper.SetDefault("database.name", "video_server")
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 20)
	viper.SetDefault("redis.min_idle_conns", 5)
	viper.SetDefault("redis.dial_timeout", 5)
	viper.SetDefault("redis.read_timeout", 3)
	viper.SetDefault("redis.write_timeout", 3)
	viper.SetDefault("database.charset", "utf8mb4")
	viper.SetDefault("storage.video_dir", "./storage/videos/")
	viper.SetDefault("storage.template_dir", "./templates/")
	viper.SetDefault("storage.temp_dir", "./storage/temp/")
	viper.SetDefault("storage.backend", "local")
	viper.SetDefault("storage.s3_use_ssl", true)
	if proxies := os.Getenv("TRUSTED_PROXIES"); proxies != "" {
		viper.Set("server.trusted_proxies", strings.Split(proxies, ","))
	}

	// Set scheduler specific config file
	viper.SetConfigName("config-scheduler")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Scheduler config file not found, using defaults: %v", err)
	}

	var config config.Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode scheduler config into struct: %v", err)
	}

	return &config
}

func startTaskRecovery(videoService *services.VideoService, taskQueue *queue.TaskQueue) {
	recover := func() {
		tasks, err := videoService.LoadRecoverableTranscodeTasks(time.Now().Add(-30*time.Minute), 20)
		if err != nil {
			log.Printf("Error loading persistent transcode tasks: %v", err)
			return
		}
		for _, task := range tasks {
			if err := taskQueue.EnqueueTranscodeTask(context.Background(), task); err != nil {
				log.Printf("Error recovering transcode task %s: %v", task.TaskID, err)
				continue
			}
			if err := videoService.SaveTranscodeTask(task); err != nil {
				log.Printf("Error updating recovered task %s: %v", task.TaskID, err)
			}
		}
	}
	recover()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		recover()
	}
}

func startCleanupWorker(videoService *services.VideoService, storageService *storage.StorageService, auditService *services.AuditService) {
	log.Println("Starting cleanup worker...")

	// 启动时先跑一次，让审计保留期在重启后立即生效；之后每小时一次
	performCleanup(videoService, storageService, auditService)

	// Run cleanup every hour
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Running scheduled cleanup...")
			performCleanup(videoService, storageService, auditService)
		}
	}
}

func startReplicationWorker(videoService *services.VideoService, storageService *storage.StorageService) {
	replicate := func() {
		if count, oldest, err := videoService.MediaReplicationStats(); err == nil {
			metrics.SetMediaReplicationStats(count, oldest)
		}
		if count, err := videoService.CountPendingReview(); err == nil {
			metrics.SetReviewBacklog(count)
		}
		videos, err := videoService.GetVideosNeedingReplication(20)
		if err != nil {
			log.Printf("Error loading media replication queue: %v", err)
			return
		}
		for _, video := range videos {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			err := storageService.CopyVideoToBackup(ctx, video.UUID)
			cancel()
			if err != nil {
				log.Printf("Error replicating media for %s: %v", video.UUID, err)
				continue
			}
			if err := videoService.MarkAssetsReplicated(video.UUID); err != nil {
				log.Printf("Error marking media replicated for %s: %v", video.UUID, err)
			}
		}
	}
	replicate()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		replicate()
	}
}

func performCleanup(videoService *services.VideoService, storageService *storage.StorageService, auditService *services.AuditService) {
	log.Println("Starting cleanup process...")

	// 审计日志按保留期清理
	retentionDays := auditRetentionDays()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if removed, err := auditService.CleanupOlderThan(ctx, time.Now().AddDate(0, 0, -retentionDays)); err != nil {
		log.Printf("Error cleaning audit logs: %v", err)
	} else if removed > 0 {
		log.Printf("Removed %d old audit logs (retention %d days)", removed, retentionDays)
	}
	cancel()

	expiredCtx, expiredCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if err := storageService.DeleteObjectsBefore(expiredCtx, "source/", time.Now().AddDate(0, 0, -7)); err != nil {
		log.Printf("Error cleaning expired source objects: %v", err)
	}
	expiredCancel()

	rejectedVideos, err := videoService.GetRejectedVideosBefore(time.Now().AddDate(0, 0, -7), 20)
	if err != nil {
		log.Printf("Error loading expired rejected videos: %v", err)
	}
	for _, video := range rejectedVideos {
		if err := videoService.DeleteVideo(video.UUID); err != nil {
			log.Printf("Error deleting expired rejected video %s: %v", video.UUID, err)
			continue
		}
		_ = videoService.AddDeletionRecord(video.UUID, "Rejected video retention expired")
		if err := storageService.DeleteVideoAssets(context.Background(), video.UUID); err != nil {
			log.Printf("Error deleting rejected video assets %s: %v", video.UUID, err)
		}
	}

	// Get videos marked for deletion
	records, err := videoService.GetDeletionRecords(50) // Process up to 50 at a time
	if err != nil {
		log.Printf("Error getting deletion records: %v", err)
		return
	}

	for _, record := range records {
		// Delete physical file
		err := storageService.DeleteVideoAssets(context.Background(), record.VideoUUID)
		if err != nil {
			log.Printf("Error deleting video file %s: %v", record.VideoUUID, err)
			continue
		}

		// Remove deletion record
		err = videoService.RemoveDeletionRecord(record.VideoUUID)
		if err != nil {
			log.Printf("Error removing deletion record for %s: %v", record.VideoUUID, err)
			continue
		}

		log.Printf("Successfully cleaned up video %s", record.VideoUUID)
	}

	log.Println("Cleanup process completed.")
}

// auditRetentionDays 读取 AUDIT_RETENTION_DAYS（默认 30 天）。
func auditRetentionDays() int {
	if v := os.Getenv("AUDIT_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 30
}

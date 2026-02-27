package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/lixiandea/video_server/internal/config"
	"github.com/lixiandea/video_server/internal/services"
	"github.com/lixiandea/video_server/pkg/logging"
	"github.com/lixiandea/video_server/pkg/queue"
	"github.com/lixiandea/video_server/pkg/redis"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize logger
	if err := logging.InitLogger("info", "stdout"); err != nil {
		log.Fatal("Failed to initialize logger: ", err)
	}
	defer logging.Sync()

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
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	// Initialize task queue
	redisClient := redis.GetClient()
	taskQueue := queue.NewTaskQueue(redisClient, queue.DefaultQueueConfig)

	// Initialize transcode service
	transcodeService := services.NewTranscodeService(taskQueue, cfg.Storage.VideoDir)

	// Get concurrency from config or default to 2
	concurrency := 2
	if val := os.Getenv("TRANSCODE_CONCURRENCY"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			concurrency = n
		}
	}

	// Start worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := transcodeService.StartWorker(ctx, concurrency); err != nil {
		log.Fatalf("Failed to start transcode worker: %v", err)
	}

	log.Printf("Transcode worker started with concurrency %d", concurrency)
	logging.GetLogger().Info("Transcode worker running",
		zap.Int("concurrency", concurrency),
		zap.String("queue", taskQueue.GetQueueName()))

	// Queue stats reporter
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stats, err := taskQueue.GetQueueStats(ctx)
			if err != nil {
				continue
			}
			logging.GetLogger().Info("Queue stats",
				zap.Int("queue_length", stats.QueueLength),
				zap.Int("pending", stats.PendingCount),
				zap.Int("processing", stats.ProcessingCount),
				zap.Int("completed", stats.CompletedCount),
				zap.Int("failed", stats.FailedCount))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down transcode worker...")
	cancel()

	// Give running tasks time to complete
	time.Sleep(5 * time.Second)
	log.Println("Transcode worker stopped")
}

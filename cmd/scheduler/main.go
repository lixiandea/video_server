package main

import (
    "log"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/lixiandea/video_server/internal/config"
    "github.com/lixiandea/video_server/internal/services"
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
    
    // Set Gin mode
    if cfg.Server.Mode == "release" {
        gin.SetMode(gin.ReleaseMode)
    } else {
        gin.SetMode(gin.DebugMode)
    }
    
    // Initialize services
    videoService := services.NewVideoService()
    
    // Start cleanup worker
    go startCleanupWorker(videoService, storageService)
    
    // Setup basic API for health checks and manual triggers
    r := gin.Default()
    r.Use(gin.Logger())
    r.Use(gin.Recovery())
    
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "OK", "service": "scheduler"})
    })
    
    r.POST("/cleanup", func(c *gin.Context) {
        performCleanup(videoService, storageService)
        c.JSON(200, gin.H{"status": "cleanup initiated"})
    })
    
    log.Printf("Starting scheduler on port %s", cfg.Server.Port)
    
    if err := r.Run(":" + cfg.Server.Port); err != nil {
        log.Fatal("Failed to start scheduler: ", err)
    }
}

func startCleanupWorker(videoService *services.VideoService, storageService *storage.StorageService) {
    log.Println("Starting cleanup worker...")
    
    // Run cleanup every hour
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            log.Println("Running scheduled cleanup...")
            performCleanup(videoService, storageService)
        }
    }
}

func performCleanup(videoService *services.VideoService, storageService *storage.StorageService) {
    log.Println("Starting cleanup process...")
    
    // Get videos marked for deletion
    records, err := videoService.GetDeletionRecords(50) // Process up to 50 at a time
    if err != nil {
        log.Printf("Error getting deletion records: %v", err)
        return
    }
    
    for _, record := range records {
        // Delete physical file
        err := storageService.DeleteVideo(record.VideoUUID)
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
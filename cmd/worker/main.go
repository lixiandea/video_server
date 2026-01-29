package main

import (
    "log"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/spf13/viper"
)

func main() {
    // Initialize configuration using viper directly
    viper.SetDefault("server.port", "8082")
    viper.SetDefault("server.mode", "debug")
    
    port := viper.GetString("server.port")
    mode := viper.GetString("server.mode")
    
    // Set Gin mode
    if mode == "release" {
        gin.SetMode(gin.ReleaseMode)
    } else {
        gin.SetMode(gin.DebugMode)
    }
    
    // Initialize background workers
    go startWorkers()
    
    // Setup basic API for health checks and worker management
    r := gin.Default()
    r.Use(gin.Logger())
    r.Use(gin.Recovery())
    
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "OK", "service": "worker"})
    })
    
    log.Printf("Starting worker service on port %s", port)
    
    if err := r.Run(":" + port); err != nil {
        log.Fatal("Failed to start worker: ", err)
    }
}

func startWorkers() {
    log.Println("Starting background workers...")
    
    // Example worker that runs periodically
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            log.Println("Worker: Performing background task...")
            // Add your background processing logic here
        }
    }
}
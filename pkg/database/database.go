package database

import (
    "fmt"
    "log"
    
    "github.com/lixiandea/video_server/internal/config"
    "github.com/lixiandea/video_server/internal/models"
    
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDatabase(cfg *config.DatabaseConfig) *gorm.DB {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.Charset)

    var err error
    DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }

    // Auto migrate schema
    err = DB.AutoMigrate(
        &models.User{},
        &models.Video{},
        &models.Comment{},
        &models.Session{},
        &models.VideoDeletionRecord{},
    )
    if err != nil {
        log.Fatalf("Failed to migrate database: %v", err)
    }

    // Set connection pool
    sqlDB, err := DB.DB()
    if err != nil {
        log.Fatalf("Failed to get database instance: %v", err)
    }
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)

    return DB
}

func GetDB() *gorm.DB {
    if DB == nil {
        log.Fatal("Database not initialized")
    }
    return DB
}
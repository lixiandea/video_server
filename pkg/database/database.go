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
        SkipDefaultTransaction: true,  // Skip default transaction
        Logger: logger.Default.LogMode(logger.Info),
    })
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }

    // Disable foreign key constraint check during migration
    sqlDB, _ := DB.DB()
    _, err = sqlDB.Exec("SET FOREIGN_KEY_CHECKS=0") // Disable foreign key checks
    if err != nil {
        log.Printf("Warning: Could not disable foreign key checks: %v", err)
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
        log.Printf("Warning: Some migrations may have failed: %v", err)
    }

    // Re-enable foreign key constraint check
    _, err = sqlDB.Exec("SET FOREIGN_KEY_CHECKS=1")
    if err != nil {
        log.Printf("Warning: Could not re-enable foreign key checks: %v", err)
    }

    // Set connection pool
    sqlDB, err = DB.DB()
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
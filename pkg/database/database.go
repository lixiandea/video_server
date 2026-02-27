package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lixiandea/video_server/internal/config"
	"github.com/lixiandea/video_server/internal/models"
	"github.com/lixiandea/video_server/pkg/logging"
	"github.com/lixiandea/video_server/pkg/metrics"
	"github.com/lixiandea/video_server/pkg/tracing"
	"go.uber.org/zap"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDatabase initializes the database connection with monitoring
func InitDatabase(cfg *config.DatabaseConfig) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.Charset)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		logging.GetLogger().Error("Failed to connect to database", zap.Error(err))
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	// Simple migration without foreign key constraints
	err = DB.Migrator().AutoMigrate(
		&models.User{},
		&models.Video{},
		&models.Comment{},
		&models.Session{},
		&models.VideoDeletionRecord{},
	)
	if err != nil {
		logging.GetLogger().Warn("Migration warning", zap.Error(err))
	}

	// Set connection pool with optimized settings
	sqlDB, err := DB.DB()
	if err != nil {
		logging.GetLogger().Error("Failed to get database instance", zap.Error(err))
		panic(fmt.Sprintf("Failed to get database instance: %v", err))
	}
	
	// Optimized connection pool settings
	sqlDB.SetMaxIdleConns(25)           // Increased idle connections for better concurrency
	sqlDB.SetMaxOpenConns(200)          // Increased max connections for high load
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // Reduced lifetime to prevent stale connections
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)  // Close idle connections after 5 minutes

	// Start database metrics collection
	go collectDBMetrics(sqlDB)

	logging.GetLogger().Info("Database initialized successfully",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
	)

	return DB
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	if DB == nil {
		logging.GetLogger().Error("Database not initialized")
		panic("Database not initialized")
	}
	return DB
}

// WithTracing wraps a database operation with tracing
func WithTracing(ctx context.Context, tableName, operation string) (context.Context, func()) {
	ctx, span := tracing.StartSpanFromContext(ctx, fmt.Sprintf("db.%s.%s", tableName, operation))
	tracing.AddSpanAttribute(ctx, "db.table", tableName)
	tracing.AddSpanAttribute(ctx, "db.operation", operation)
	
	startTime := time.Now()
	return ctx, func() {
		defer span.End()
		metrics.IncDBQueries(tableName, operation)
		metrics.ObserveDBQueryDuration(tableName, operation, time.Since(startTime))
	}
}

// collectDBMetrics periodically collects database metrics
func collectDBMetrics(sqlDB *sql.DB) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			stats := sqlDB.Stats()
			metrics.SetDBConnections(float64(stats.OpenConnections))
		}
	}
}

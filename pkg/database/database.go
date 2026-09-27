package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
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

const (
	defaultConnectAttempts = 5
	defaultConnectDelay    = 3 * time.Second
)

// connectAttempts 允许通过 DB_CONNECT_ATTEMPTS 调整启动重试次数
func connectAttempts() int {
	if v := os.Getenv("DB_CONNECT_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultConnectAttempts
}

// connectDelay 允许通过 DB_CONNECT_DELAY 调整重试间隔（如 500ms）
func connectDelay() time.Duration {
	if v := os.Getenv("DB_CONNECT_DELAY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return defaultConnectDelay
}

// InitDatabase initializes the database connection with retry and monitoring.
// It returns an error instead of panicking so callers can exit gracefully.
func InitDatabase(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.Charset)

	attempts := connectAttempts()
	delay := connectDelay()

	var (
		db  *gorm.DB
		err error
	)

	for attempt := 1; attempt <= attempts; attempt++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			SkipDefaultTransaction: true,
			Logger:                 logger.Default.LogMode(logger.Info),
		})
		if err == nil {
			var sqlDB *sql.DB
			if sqlDB, err = db.DB(); err == nil {
				err = sqlDB.Ping()
			}
		}
		if err == nil {
			break
		}

		logging.GetLogger().Warn("Database connection attempt failed",
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", attempts),
			zap.Error(err))

		if attempt < attempts {
			time.Sleep(delay)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempt(s): %w", attempts, err)
	}

	DB = db

	migrateSchema(DB)

	// Set connection pool with optimized settings
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
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

	return DB, nil
}

const (
	// migrationLockName 用于串行化多个服务实例的并发 DDL 迁移
	migrationLockName = "video_server:auto_migrate"
	// migrationLockTimeout 等待迁移锁的最长时间
	migrationLockTimeout = 30
)

// migrateSchema 逐表执行 AutoMigrate。
// 每个服务（api-server/scheduler/worker）都会在启动时调用，因此需要：
//  1. 用 MySQL 命名锁串行化并发 DDL，避免 "Table already exists"/"Duplicate key name" 噪声；
//  2. 单表失败不影响后续表（历史上 comments 冲突曾导致 video_deletion_records 未创建）。
func migrateSchema(db *gorm.DB) {
	migrations := []struct {
		table string
		model interface{}
	}{
		{"users", &models.User{}},
		{"videos", &models.Video{}},
		{"comments", &models.Comment{}},
		{"sessions", &models.Session{}},
		{"video_deletion_records", &models.VideoDeletionRecord{}},
	}

	if release, err := acquireMigrationLock(db); err != nil {
		logging.GetLogger().Warn("Migrating without lock", zap.Error(err))
	} else {
		defer release()
	}

	for _, m := range migrations {
		if err := db.Migrator().AutoMigrate(m.model); err != nil {
			logging.GetLogger().Error("Database migration failed",
				zap.String("table", m.table),
				zap.Error(err))
		}
	}
}

// acquireMigrationLock 通过独占连接获取 MySQL 命名锁并返回释放函数。
// GET_LOCK 的作用域是连接，因此 GET_LOCK/RELEASE_LOCK 必须复用同一条 *sql.Conn。
func acquireMigrationLock(db *gorm.DB) (func(), error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dedicated connection: %w", err)
	}

	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", migrationLockName, migrationLockTimeout).Scan(&acquired); err != nil {
		conn.Close()
		return nil, fmt.Errorf("GET_LOCK failed: %w", err)
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		conn.Close()
		return nil, fmt.Errorf("migration lock %q held by another instance", migrationLockName)
	}

	return func() {
		if _, err := conn.ExecContext(ctx, "SELECT RELEASE_LOCK(?)", migrationLockName); err != nil {
			logging.GetLogger().Warn("Failed to release migration lock", zap.Error(err))
		}
		conn.Close()
	}, nil
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

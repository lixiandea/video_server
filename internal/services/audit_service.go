package services

import (
	"context"
	"fmt"
	"time"

	"github.com/lixiandea/video_server/internal/models"
	"github.com/lixiandea/video_server/pkg/logging"
	"github.com/lixiandea/video_server/pkg/metrics"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AuditService 异步写入审计日志：写请求只入队，不阻塞在数据库上。
type AuditService struct {
	db *gorm.DB
	ch chan models.AuditLog
}

// NewAuditService 创建审计服务；缓冲 1024 条，满时丢弃并告警。
func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db, ch: make(chan models.AuditLog, 1024)}
}

// Start 启动后台写入协程，ctx 取消后停止。
func (s *AuditService) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case entry := <-s.ch:
				if err := s.db.Create(&entry).Error; err != nil {
					logging.GetLogger().Error("Failed to persist audit log", zap.Error(err))
				}
			}
		}
	}()
}

// Record 入队一条审计日志；队列满时丢弃，绝不阻塞请求。
func (s *AuditService) Record(entry models.AuditLog) {
	select {
	case s.ch <- entry:
	default:
		metrics.IncAuditDropped()
		logging.GetLogger().Warn("Audit log queue full, dropping entry",
			zap.String("method", entry.Method),
			zap.String("path", entry.Path))
	}
}

// CleanupOlderThan 删除创建时间早于 cutoff 的审计日志，返回删除条数。
func (s *AuditService) CleanupOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&models.AuditLog{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to clean audit logs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/lixiandea/video_server/internal/models"
	"github.com/lixiandea/video_server/pkg/database"
	apperrors "github.com/lixiandea/video_server/pkg/errors"
	"github.com/lixiandea/video_server/pkg/redis"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LikeService 负责点赞关系的写入与查询。
// 关系表与 videos.like_count 冗余计数在同一事务内更新，保证两者不漂移。
type LikeService struct {
	db    *gorm.DB
	cache *redis.CacheManager
}

func NewLikeService() *LikeService {
	return &LikeService{
		db:    database.GetDB(),
		cache: redis.NewCacheManager(redis.GetClient()),
	}
}

func (s *LikeService) invalidateVideoCache(videoID string) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Delete(context.Background(), videoCacheKey(videoID))
}

// Like 点赞。已点过赞时直接返回当前计数，不重复累加（幂等）。
func (s *LikeService) Like(userID uint, videoID string) (int64, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := ensureVideoExists(tx, videoID); err != nil {
			return err
		}

		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.VideoLike{
			UserID:    userID,
			VideoUUID: videoID,
		})
		if result.Error != nil {
			return fmt.Errorf("failed to like video: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return nil
		}

		return tx.Model(&models.Video{}).Where("uuid = ?", videoID).
			UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
	})
	if err != nil {
		return 0, err
	}

	s.invalidateVideoCache(videoID)
	return s.LikeCount(videoID)
}

// Unlike 取消点赞。未点赞时直接返回当前计数，不重复递减（幂等）。
func (s *LikeService) Unlike(userID uint, videoID string) (int64, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := ensureVideoExists(tx, videoID); err != nil {
			return err
		}

		result := tx.Unscoped().
			Where("user_id = ? AND video_uuid = ?", userID, videoID).
			Delete(&models.VideoLike{})
		if result.Error != nil {
			return fmt.Errorf("failed to unlike video: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return nil
		}

		// 冗余计数可能被历史数据写坏，这里兜底避免出现负数
		return tx.Model(&models.Video{}).Where("uuid = ?", videoID).
			UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")).Error
	})
	if err != nil {
		return 0, err
	}

	s.invalidateVideoCache(videoID)
	return s.LikeCount(videoID)
}

// IsLiked 判断用户是否已点赞该视频。
func (s *LikeService) IsLiked(userID uint, videoID string) (bool, error) {
	var count int64
	if err := s.db.Model(&models.VideoLike{}).
		Where("user_id = ? AND video_uuid = ?", userID, videoID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("database error: %w", err)
	}

	return count > 0, nil
}

// LikeCount 读取视频当前点赞数。
func (s *LikeService) LikeCount(videoID string) (int64, error) {
	var video models.Video
	if err := s.db.Select("like_count").Where("uuid = ?", videoID).First(&video).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, apperrors.NotFoundf("video not found")
		}
		return 0, fmt.Errorf("database error: %w", err)
	}

	return video.LikeCount, nil
}

// ensureVideoExists 拒绝给不存在或已删除的视频点赞。
func ensureVideoExists(tx *gorm.DB, videoID string) error {
	var video models.Video
	if err := tx.Select("uuid").Where("uuid = ?", videoID).First(&video).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NotFoundf("video not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	return nil
}

package services

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/lixiandea/video_server/internal/models"
    "github.com/lixiandea/video_server/pkg/database"
    apperrors "github.com/lixiandea/video_server/pkg/errors"
    "github.com/lixiandea/video_server/pkg/redis"
    "gorm.io/gorm"
)

type VideoService struct {
    db    *gorm.DB
    cache *redis.CacheManager
}

func NewVideoService() *VideoService {
    return &VideoService{
        db:    database.GetDB(),
        cache: redis.NewCacheManager(redis.GetClient()),
    }
}

// videoCacheKey 定义视频缓存的键名，读写两侧必须共用同一规则。
func videoCacheKey(id string) string {
    return fmt.Sprintf("video:%s", id)
}

// invalidateVideoCache 在写操作后清除缓存，避免继续返回旧数据。
// 缓存删除失败不应阻断写操作，CacheManager 内部已记录告警。
func (s *VideoService) invalidateVideoCache(videoID string) {
    if s.cache == nil {
        return
    }
    _ = s.cache.Delete(context.Background(), videoCacheKey(videoID))
}

func (s *VideoService) CreateVideo(authorID uint, name string) (*models.Video, error) {
    videoUUID := uuid.New().String()
    ctime := time.Now().Format("Jan 02 2006, 15:04:05")

    video := &models.Video{
        UUID:        videoUUID,
        AuthorID:    authorID,
        Name:        name,
        DisplayCTime: ctime,
        Status:      "active",
    }

    result := s.db.Create(video)
    if result.Error != nil {
        return nil, fmt.Errorf("failed to create video: %w", result.Error)
    }

    return video, nil
}

func (s *VideoService) GetVideoByID(videoID string) (*models.Video, error) {
    ctx := context.Background()
    cacheKey := videoCacheKey(videoID)
    var video models.Video

    // 尝试从缓存获取
    if s.cache != nil {
        if err := s.cache.Get(ctx, cacheKey, &video); err == nil && video.UUID != "" {
            return &video, nil
        }
    }

    // 从数据库获取
    result := s.db.First(&video, "uuid = ?", videoID)
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        return nil, apperrors.NotFoundf("video not found")
    }
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    // 设置缓存（5 分钟过期）
    if s.cache != nil {
        _ = s.cache.Set(ctx, cacheKey, &video, 5*time.Minute)
    }

    return &video, nil
}

func (s *VideoService) GetVideosByUserID(userID uint, limit, offset int) ([]*models.Video, error) {
    var videos []*models.Video
    result := s.db.Where("author_id = ?", userID).
        Limit(limit).Offset(offset).
        Order("created_at DESC").
        Find(&videos)

    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    return videos, nil
}

// GetVideosByUserIDWithTotal returns videos with actual total count for pagination
func (s *VideoService) GetVideosByUserIDWithTotal(userID uint, limit, offset int) ([]*models.Video, int64, error) {
    var videos []*models.Video
    var total int64

    // Get total count
    if err := s.db.Model(&models.Video{}).Where("author_id = ?", userID).Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count videos: %w", err)
    }

    // Get paginated videos
    result := s.db.Where("author_id = ?", userID).
        Limit(limit).Offset(offset).
        Order("created_at DESC").
        Find(&videos)

    if result.Error != nil {
        return nil, 0, fmt.Errorf("database error: %w", result.Error)
    }

    return videos, total, nil
}

// GetVideos 分页查询可播放（status=active）的视频，query 非空时按名称模糊匹配。
// 返回 (视频列表, 总数)，供 GET /videos 使用。
func (s *VideoService) GetVideos(limit, offset int, query string) ([]*models.Video, int64, error) {
    var videos []*models.Video
    var total int64

    base := func() *gorm.DB {
        q := s.db.Model(&models.Video{}).Where("status = ?", "active")
        if query != "" {
            q = q.Where("name LIKE ?", "%"+query+"%")
        }
        return q
    }

    if err := base().Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count videos: %w", err)
    }

    result := base().Limit(limit).Offset(offset).Order("created_at DESC").Find(&videos)
    if result.Error != nil {
        return nil, 0, fmt.Errorf("database error: %w", result.Error)
    }

    return videos, total, nil
}

// GetRelatedVideos 返回同一作者的其他可播放视频（不含当前视频）。
func (s *VideoService) GetRelatedVideos(videoID string, authorID uint, limit int) ([]*models.Video, error) {
    var videos []*models.Video
    result := s.db.Where("author_id = ? AND status = ? AND uuid <> ?", authorID, "active", videoID).
        Limit(limit).Order("created_at DESC").Find(&videos)
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    return videos, nil
}

// GetAuthorNames 批量查询作者名，避免列表接口逐条查询作者（N+1）。
func (s *VideoService) GetAuthorNames(authorIDs []uint) (map[uint]string, error) {
    names := make(map[uint]string)
    if len(authorIDs) == 0 {
        return names, nil
    }

    var users []*models.User
    if err := s.db.Where("id IN ?", authorIDs).Find(&users).Error; err != nil {
        return nil, fmt.Errorf("database error: %w", err)
    }
    for _, u := range users {
        names[u.ID] = u.LoginName
    }

    return names, nil
}

func (s *VideoService) UpdateVideo(videoID string, updates map[string]interface{}) error {
    result := s.db.Model(&models.Video{}).Where("uuid = ?", videoID).Updates(updates)
    if result.Error != nil {
        return fmt.Errorf("failed to update video: %w", result.Error)
    }
    if result.RowsAffected == 0 {
        return apperrors.NotFoundf("video not found")
    }

    s.invalidateVideoCache(videoID)
    return nil
}

// IncrementViewCount 记录一次观看。
// 播放量是高频写入，用 SQL 自增避免「读-改-写」丢失更新，并让 updated_at 保持稳定。
func (s *VideoService) IncrementViewCount(videoID string) error {
    result := s.db.Model(&models.Video{}).Where("uuid = ?", videoID).
        UpdateColumn("view_count", gorm.Expr("view_count + 1"))
    if result.Error != nil {
        return fmt.Errorf("failed to increment view count: %w", result.Error)
    }
    if result.RowsAffected == 0 {
        return apperrors.NotFoundf("video not found")
    }

    s.invalidateVideoCache(videoID)
    return nil
}

func (s *VideoService) DeleteVideo(videoID string) error {
    result := s.db.Model(&models.Video{}).Where("uuid = ?", videoID).Update("status", "deleted")
    if result.Error != nil {
        return fmt.Errorf("failed to delete video: %w", result.Error)
    }
    if result.RowsAffected == 0 {
        return apperrors.NotFoundf("video not found")
    }

    s.invalidateVideoCache(videoID)
    return nil
}

func (s *VideoService) HardDeleteVideo(videoID string) error {
    result := s.db.Where("uuid = ?", videoID).Delete(&models.Video{})
    if result.Error != nil {
        return fmt.Errorf("failed to hard delete video: %w", result.Error)
    }

    s.invalidateVideoCache(videoID)
    return nil
}

func (s *VideoService) AddDeletionRecord(videoID, reason string) error {
    record := &models.VideoDeletionRecord{
        VideoUUID: videoID,
        Reason:    reason,
    }

    result := s.db.Create(record)
    if result.Error != nil {
        return fmt.Errorf("failed to add deletion record: %w", result.Error)
    }

    return nil
}

func (s *VideoService) GetDeletionRecords(limit int) ([]*models.VideoDeletionRecord, error) {
    var records []*models.VideoDeletionRecord
    result := s.db.Limit(limit).Order("created_at DESC").Find(&records)
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    return records, nil
}

func (s *VideoService) RemoveDeletionRecord(videoID string) error {
    result := s.db.Where("video_uuid = ?", videoID).Delete(&models.VideoDeletionRecord{})
    if result.Error != nil {
        return fmt.Errorf("failed to remove deletion record: %w", result.Error)
    }

    return nil
}

// 根据用户ID获取用户信息
func (s *VideoService) GetUserByID(userID uint) (*models.User, error) {
    var user models.User
    result := s.db.First(&user, "id = ?", userID)
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        return nil, apperrors.NotFoundf("user not found")
    }
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    return &user, nil
}

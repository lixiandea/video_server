package services

import (
    "errors"
    "fmt"
    "time"
    
    "github.com/google/uuid"
    "github.com/lixiandea/video_server/internal/models"
    "github.com/lixiandea/video_server/pkg/database"
    "gorm.io/gorm"
)

type VideoService struct {
    db *gorm.DB
}

func NewVideoService() *VideoService {
    return &VideoService{
        db: database.GetDB(),
    }
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
    var video models.Video
    result := s.db.Preload("Author").First(&video, "uuid = ?", videoID)
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        return nil, fmt.Errorf("video not found")
    }
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    return &video, nil
}

func (s *VideoService) GetVideosByUserID(userID uint, limit, offset int) ([]*models.Video, error) {
    var videos []*models.Video
    result := s.db.Preload("Author").
        Where("author_id = ?", userID).
        Limit(limit).Offset(offset).
        Order("created_at DESC").
        Find(&videos)
        
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    return videos, nil
}

func (s *VideoService) UpdateVideo(videoID string, updates map[string]interface{}) error {
    result := s.db.Model(&models.Video{}).Where("uuid = ?", videoID).Updates(updates)
    if result.Error != nil {
        return fmt.Errorf("failed to update video: %w", result.Error)
    }
    if result.RowsAffected == 0 {
        return fmt.Errorf("video not found")
    }

    return nil
}

func (s *VideoService) DeleteVideo(videoID string) error {
    result := s.db.Model(&models.Video{}).Where("uuid = ?", videoID).Update("status", "deleted")
    if result.Error != nil {
        return fmt.Errorf("failed to delete video: %w", result.Error)
    }
    if result.RowsAffected == 0 {
        return fmt.Errorf("video not found")
    }

    return nil
}

func (s *VideoService) HardDeleteVideo(videoID string) error {
    result := s.db.Where("uuid = ?", videoID).Delete(&models.Video{})
    if result.Error != nil {
        return fmt.Errorf("failed to hard delete video: %w", result.Error)
    }

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
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

type CommentService struct {
    db *gorm.DB
}

func NewCommentService() *CommentService {
    return &CommentService{
        db: database.GetDB(),
    }
}

func (s *CommentService) AddComment(videoID, authorID uint, content string) (*models.Comment, error) {
    commentUUID := uuid.New().String()
    ctime := time.Now().Format("Jan 02 2006, 15:04:05")

    comment := &models.Comment{
        UUID:     commentUUID,
        VideoID:  videoID,
        AuthorID: authorID,
        Content:  content,
        Ctime:    ctime,
    }

    result := s.db.Create(comment)
    if result.Error != nil {
        return nil, fmt.Errorf("failed to add comment: %w", result.Error)
    }

    return comment, nil
}

func (s *CommentService) GetCommentByID(commentID string) (*models.Comment, error) {
    var comment models.Comment
    result := s.db.First(&comment, "uuid = ?", commentID)
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        return nil, fmt.Errorf("comment not found")
    }
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    return &comment, nil
}

func (s *CommentService) GetCommentsByVideoID(videoID uint, limit, offset int) ([]*models.Comment, error) {
    var comments []*models.Comment
    result := s.db.Where("video_id = ?", videoID).
        Limit(limit).Offset(offset).
        Order("created_at ASC").
        Find(&comments)
        
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    return comments, nil
}

func (s *CommentService) UpdateComment(commentID string, updates map[string]interface{}) error {
    result := s.db.Model(&models.Comment{}).Where("uuid = ?", commentID).Updates(updates)
    if result.Error != nil {
        return fmt.Errorf("failed to update comment: %w", result.Error)
    }
    if result.RowsAffected == 0 {
        return fmt.Errorf("comment not found")
    }

    return nil
}

func (s *CommentService) DeleteComment(commentID string) error {
    result := s.db.Where("uuid = ?", commentID).Delete(&models.Comment{})
    if result.Error != nil {
        return fmt.Errorf("failed to delete comment: %w", result.Error)
    }

    return nil
}

// 根据用户ID获取用户信息
func (s *CommentService) GetUserByID(userID uint) (*models.User, error) {
    var user models.User
    result := s.db.First(&user, "id = ?", userID)
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        return nil, fmt.Errorf("user not found")
    }
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    return &user, nil
}
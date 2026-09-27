package services

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/lixiandea/video_server/internal/models"
    "github.com/lixiandea/video_server/pkg/auth"
    "github.com/lixiandea/video_server/pkg/database"
    apperrors "github.com/lixiandea/video_server/pkg/errors"
    "github.com/lixiandea/video_server/pkg/redis"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

type UserService struct {
    db    *gorm.DB
    cache *redis.CacheManager
}

func NewUserService() *UserService {
    return &UserService{
        db:    database.GetDB(),
        cache: redis.NewCacheManager(redis.GetClient()),
    }
}

// userCacheKey 定义用户缓存的键名，读写两侧必须共用同一规则。
func userCacheKey(id uint) string {
    return fmt.Sprintf("user:%d", id)
}

// invalidateUserCache 在写操作后清除缓存，避免继续返回旧数据。
// 缓存删除失败不应阻断写操作，CacheManager 内部已记录告警。
func (s *UserService) invalidateUserCache(id uint) {
    if s.cache == nil {
        return
    }
    _ = s.cache.Delete(context.Background(), userCacheKey(id))
}

func (s *UserService) CreateUser(username, password string) (*models.User, error) {
    hashedPwd, err := auth.HashPassword(password)
    if err != nil {
        return nil, fmt.Errorf("failed to hash password: %w", err)
    }

    user := &models.User{
        LoginName: username,
        Pwd:       hashedPwd,
    }

    result := s.db.Create(user)
    if result.Error != nil {
        return nil, fmt.Errorf("failed to create user: %w", result.Error)
    }

    return user, nil
}

func (s *UserService) AuthenticateUser(username, password string) (*models.User, error) {
    var user models.User
    result := s.db.Where("login_name = ?", username).First(&user)
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        return nil, apperrors.NotFoundf("user not found")
    }
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    if !auth.CheckPasswordHash(password, user.Pwd) {
        return nil, apperrors.Unauthorizedf("invalid credentials")
    }

    return &user, nil
}

func (s *UserService) GetUserByID(id uint) (*models.User, error) {
    ctx := context.Background()
    cacheKey := userCacheKey(id)
    var user models.User

    // 尝试从缓存获取
    if s.cache != nil {
        if err := s.cache.Get(ctx, cacheKey, &user); err == nil && user.ID != 0 {
            return &user, nil
        }
    }

    // 从数据库获取
    result := s.db.First(&user, id)
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        return nil, apperrors.NotFoundf("user not found")
    }
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    // 设置缓存（10 分钟过期）
    if s.cache != nil {
        _ = s.cache.Set(ctx, cacheKey, &user, 10*time.Minute)
    }

    return &user, nil
}

func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
    var user models.User
    result := s.db.Where("login_name = ?", username).First(&user)
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        return nil, apperrors.NotFoundf("user not found")
    }
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    return &user, nil
}

func (s *UserService) UpdateUser(id uint, updates map[string]interface{}) error {
    result := s.db.Model(&models.User{}).Where("id = ?", id).Updates(updates)
    if result.Error != nil {
        return fmt.Errorf("failed to update user: %w", result.Error)
    }
    if result.RowsAffected == 0 {
        return apperrors.NotFoundf("user not found")
    }

    s.invalidateUserCache(id)
    return nil
}

// DeleteUser 注销账户并级联清理其数据：会话、评论、视频一并处理。
// 全部写入放在同一事务内，避免出现「账号已删但内容残留」的半清理状态。
func (s *UserService) DeleteUser(id uint) error {
    var videoUUIDs []string

    err := s.db.Transaction(func(tx *gorm.DB) error {
        // 已被软删除的视频同样需要写台账，因此读取时放开默认作用域。
        var videos []models.Video
        if err := tx.Unscoped().Select("uuid").Where("author_id = ?", id).Find(&videos).Error; err != nil {
            return fmt.Errorf("failed to load user videos: %w", err)
        }
        for _, video := range videos {
            videoUUIDs = append(videoUUIDs, video.UUID)
        }

        // 会话是凭据，直接物理删除，等效于登出该用户的所有设备。
        if err := tx.Unscoped().Where("user_id = ?", id).Delete(&models.Session{}).Error; err != nil {
            return fmt.Errorf("failed to delete user sessions: %w", err)
        }

        // 评论随账号一并下架，沿用单条删除的软删除语义。
        if err := tx.Where("author_id = ?", id).Delete(&models.Comment{}).Error; err != nil {
            return fmt.Errorf("failed to delete user comments: %w", err)
        }

        // 视频标记为 deleted 后软删除，并留下去重后的删除台账。
        if err := tx.Model(&models.Video{}).Where("author_id = ?", id).
            Update("status", "deleted").Error; err != nil {
            return fmt.Errorf("failed to mark user videos deleted: %w", err)
        }
        if err := tx.Where("author_id = ?", id).Delete(&models.Video{}).Error; err != nil {
            return fmt.Errorf("failed to delete user videos: %w", err)
        }
        for _, videoUUID := range videoUUIDs {
            record := &models.VideoDeletionRecord{
                VideoUUID: videoUUID,
                Reason:    "Account deleted",
            }
            if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(record).Error; err != nil {
                return fmt.Errorf("failed to add deletion record: %w", err)
            }
        }

        result := tx.Delete(&models.User{}, id)
        if result.Error != nil {
            return fmt.Errorf("failed to delete user: %w", result.Error)
        }
        if result.RowsAffected == 0 {
            return apperrors.NotFoundf("user not found")
        }

        return nil
    })
    if err != nil {
        return err
    }

    // 事务提交后再清缓存，避免回滚后缓存已被清空的空窗。
    s.invalidateUserCache(id)
    if s.cache != nil {
        for _, videoUUID := range videoUUIDs {
            _ = s.cache.Delete(context.Background(), videoCacheKey(videoUUID))
        }
    }

    return nil
}

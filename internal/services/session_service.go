package services

import (
    "errors"
    "fmt"
    "time"

    "github.com/lixiandea/video_server/internal/models"
    "github.com/lixiandea/video_server/pkg/auth"
    "github.com/lixiandea/video_server/pkg/database"
    apperrors "github.com/lixiandea/video_server/pkg/errors"
    "gorm.io/gorm"
)

// SessionService 负责会话的落地、校验与吊销。
// JWT 本身无法撤销，因此访问令牌只携带 session_id，真实有效性以本表记录为准，
// 登出 / 改密后旧令牌立即失效。
type SessionService struct {
    db *gorm.DB
}

func NewSessionService() *SessionService {
    return &SessionService{
        db: database.GetDB(),
    }
}

// CreateSession 记录一次登录会话，过期时间与访问令牌保持一致。
func (s *SessionService) CreateSession(userID uint, sessionID string) error {
    if sessionID == "" {
        return apperrors.BadRequestf("session id is required")
    }

    session := &models.Session{
        UUID:      sessionID,
        UserID:    userID,
        ExpiresAt: time.Now().Add(auth.AccessTokenTTL).Unix(),
    }

    if err := s.db.Create(session).Error; err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }

    return nil
}

// ValidateSession 校验会话存在且未过期，供鉴权中间件逐请求调用。
// 会话被登出（记录已删除）或已过期都会返回 401。
func (s *SessionService) ValidateSession(sessionID string, userID uint) error {
    if sessionID == "" {
        return apperrors.Unauthorizedf("session id is missing")
    }

    var session models.Session
    result := s.db.Where("uuid = ? AND user_id = ?", sessionID, userID).First(&session)
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        return apperrors.Unauthorizedf("session is not valid or has expired")
    }
    if result.Error != nil {
        return fmt.Errorf("database error: %w", result.Error)
    }
    if session.ExpiresAt > 0 && session.ExpiresAt < time.Now().Unix() {
        return apperrors.Unauthorizedf("session has expired")
    }

    return nil
}

// RevokeSession 吊销单个会话（登出）。会话已不存在时视为成功，保证登出幂等。
func (s *SessionService) RevokeSession(sessionID string) error {
    if sessionID == "" {
        return nil
    }

    // 会话是凭据而非业务数据，直接物理删除，避免唯一索引长期堆积失效记录。
    if err := s.db.Unscoped().Where("uuid = ?", sessionID).Delete(&models.Session{}).Error; err != nil {
        return fmt.Errorf("failed to revoke session: %w", err)
    }

    return nil
}

// RevokeUserSessionsExcept 吊销某用户除 keepSessionID 外的全部会话，
// 用于「登出其他设备」以及改密后强制其他设备重新登录。
func (s *SessionService) RevokeUserSessionsExcept(userID uint, keepSessionID string) (int64, error) {
    query := s.db.Unscoped().Where("user_id = ?", userID)
    if keepSessionID != "" {
        query = query.Where("uuid <> ?", keepSessionID)
    }

    result := query.Delete(&models.Session{})
    if result.Error != nil {
        return 0, fmt.Errorf("failed to revoke sessions: %w", result.Error)
    }

    return result.RowsAffected, nil
}

// DeleteExpiredSessions 清理已过期的会话记录，由 api-server 定期触发。
func (s *SessionService) DeleteExpiredSessions() (int64, error) {
    result := s.db.Unscoped().
        Where("ttl > 0 AND ttl < ?", time.Now().Unix()).
        Delete(&models.Session{})
    if result.Error != nil {
        return 0, fmt.Errorf("failed to delete expired sessions: %w", result.Error)
    }

    return result.RowsAffected, nil
}

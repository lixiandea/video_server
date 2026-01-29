package services

import (
    "errors"
    "fmt"
    
    "github.com/lixiandea/video_server/internal/models"
    "github.com/lixiandea/video_server/pkg/auth"
    "github.com/lixiandea/video_server/pkg/database"
    "gorm.io/gorm"
)

type UserService struct {
    db *gorm.DB
}

func NewUserService() *UserService {
    return &UserService{
        db: database.GetDB(),
    }
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
        return nil, fmt.Errorf("user not found")
    }
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    if !auth.CheckPasswordHash(password, user.Pwd) {
        return nil, fmt.Errorf("invalid credentials")
    }

    return &user, nil
}

func (s *UserService) GetUserByID(id uint) (*models.User, error) {
    var user models.User
    result := s.db.First(&user, id)
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        return nil, fmt.Errorf("user not found")
    }
    if result.Error != nil {
        return nil, fmt.Errorf("database error: %w", result.Error)
    }

    return &user, nil
}

func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
    var user models.User
    result := s.db.Where("login_name = ?", username).First(&user)
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        return nil, fmt.Errorf("user not found")
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
        return fmt.Errorf("user not found")
    }

    return nil
}

func (s *UserService) DeleteUser(id uint) error {
    result := s.db.Delete(&models.User{}, id)
    if result.Error != nil {
        return fmt.Errorf("failed to delete user: %w", result.Error)
    }

    return nil
}
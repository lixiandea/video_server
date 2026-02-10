package models

import (
    "gorm.io/gorm"
    "time"
)

type BaseModel struct {
    ID        uint           `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type User struct {
    BaseModel
    LoginName string      `gorm:"uniqueIndex;not null;type:varchar(255)" json:"login_name"`
    Pwd       string      `gorm:"not null" json:"pwd"`
}

type Video struct {
    BaseModel
    UUID       string    `gorm:"uniqueIndex;not null;type:varchar(255)" json:"uuid"`
    AuthorID   uint      `gorm:"not null" json:"author_id"`
    Name       string    `gorm:"not null" json:"name"`
    FilePath   string    `gorm:"not null" json:"file_path"`
    Size       int64     `json:"size"`
    Duration   float64   `json:"duration"` // in seconds
    Status     string    `gorm:"default:active" json:"status"` // active, deleted, processing
    DisplayCTime string  `json:"display_ctime"`
}

type Comment struct {
    BaseModel
    UUID     string `gorm:"uniqueIndex;not null;type:varchar(255)" json:"id"`
    AuthorID uint   `gorm:"not null" json:"author_id"`
    VideoID  string `gorm:"not null;type:varchar(255)" json:"video_id"`
    Content  string `gorm:"not null" json:"content"`
    Ctime    string `gorm:"column:time" json:"ctime"`
}

type Session struct {
    BaseModel
    UUID      string `gorm:"uniqueIndex;not null;type:varchar(255)" json:"uuid"`
    UserID    uint   `gorm:"not null" json:"user_id"`
    TTL       int64  `json:"ttl"`
}

type VideoDeletionRecord struct {
    BaseModel
    VideoUUID string `gorm:"uniqueIndex;not null;type:varchar(255)" json:"video_uuid"`
    Reason    string `json:"reason,omitempty"`
}
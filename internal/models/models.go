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
    LoginName string  `gorm:"uniqueIndex;not null" json:"login_name"`
    Pwd       string  `gorm:"not null" json:"pwd"`
    Sessions  []Session `json:"-"` // One-to-many relationship
    Videos    []Video `json:"-"`   // One-to-many relationship
}

type Video struct {
    BaseModel
    UUID       string    `gorm:"uniqueIndex;not null" json:"uuid"`
    AuthorID   uint      `gorm:"not null" json:"author_id"`
    Name       string    `gorm:"not null" json:"name"`
    FilePath   string    `gorm:"not null" json:"file_path"`
    Size       int64     `json:"size"`
    Duration   float64   `json:"duration"` // in seconds
    Status     string    `gorm:"default:active" json:"status"` // active, deleted, processing
    DisplayCTime string  `json:"display_ctime"`
    Author     User      `gorm:"foreignKey:AuthorID" json:"-"`
    Comments   []Comment `json:"-"` // One-to-many relationship
}

type Comment struct {
    BaseModel
    UUID     string `gorm:"uniqueIndex;not null" json:"id"`
    AuthorID uint   `gorm:"not null" json:"author_id"`
    VideoID  uint   `gorm:"not null" json:"video_id"`
    Content  string `gorm:"not null" json:"content"`
    Ctime    string `json:"ctime"`
    Author   User   `gorm:"foreignKey:AuthorID" json:"-"`
    Video    Video  `gorm:"foreignKey:VideoID" json:"-"`
}

type Session struct {
    BaseModel
    UUID      string `gorm:"uniqueIndex;not null" json:"uuid"`
    UserID    uint   `gorm:"not null" json:"user_id"`
    TTL       int64  `json:"ttl"`
    User      User   `gorm:"foreignKey:UserID" json:"-"`
}

type VideoDeletionRecord struct {
    BaseModel
    VideoUUID string `gorm:"uniqueIndex;not null" json:"video_uuid"`
    Reason    string `json:"reason,omitempty"`
}
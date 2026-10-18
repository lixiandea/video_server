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
	LoginName string `gorm:"uniqueIndex;not null;type:varchar(255)" json:"login_name"`
	Pwd       string `gorm:"not null" json:"pwd"`
	// Email / Bio 为可选资料字段，空字符串表示未填写
	Email string `gorm:"type:varchar(255);default:''" json:"email"`
	Bio   string `gorm:"type:varchar(500);default:''" json:"bio"`
}

type Video struct {
	BaseModel
	UUID         string  `gorm:"uniqueIndex;not null;type:varchar(255)" json:"uuid"`
	AuthorID     uint    `gorm:"not null" json:"author_id"`
	Name         string  `gorm:"not null" json:"name"`
	FilePath     string  `gorm:"not null" json:"file_path"`
	Size         int64   `json:"size"`
	Duration     float64 `json:"duration"`                     // in seconds
	Status       string  `gorm:"default:active" json:"status"` // active, processing, failed, deleted
	DisplayCTime string  `json:"display_ctime"`
	// Description / CoverPath / ViewCount 为 P1-11b 新增的展示字段
	Description string `gorm:"type:varchar(500);default:''" json:"description"`
	CoverPath   string `gorm:"type:varchar(255);default:''" json:"cover_path"`
	ViewCount   int64  `gorm:"not null;default:0" json:"view_count"`
	LikeCount   int64  `gorm:"not null;default:0" json:"like_count"`
}

// VideoLike 记录「谁点赞了哪个视频」，(user_id, video_uuid) 组合唯一，
// 使重复点赞天然幂等；videos.like_count 是该表的冗余计数，用于列表展示。
type VideoLike struct {
	BaseModel
	UserID    uint   `gorm:"not null;uniqueIndex:idx_user_video" json:"user_id"`
	VideoUUID string `gorm:"not null;type:varchar(255);uniqueIndex:idx_user_video" json:"video_uuid"`
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
	UUID   string `gorm:"uniqueIndex;not null;type:varchar(255)" json:"uuid"`
	UserID uint   `gorm:"not null;index" json:"user_id"`
	// ExpiresAt 是会话到期时间（Unix 秒），与访问令牌有效期保持一致。
	// 列名沿用历史的 ttl，避免对既有表做重命名 DDL。
	ExpiresAt int64 `gorm:"column:ttl" json:"expires_at"`
}

type VideoDeletionRecord struct {
	BaseModel
	VideoUUID string `gorm:"uniqueIndex;not null;type:varchar(255)" json:"video_uuid"`
	Reason    string `json:"reason,omitempty"`
}

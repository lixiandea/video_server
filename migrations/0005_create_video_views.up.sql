-- 0005 用户观看行为：每个用户对每个视频只保留一条记录，updated_at 表示最近观看时间。
CREATE TABLE IF NOT EXISTS video_views (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    video_uuid VARCHAR(255) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_video_views_user_video (user_id, video_uuid),
    KEY idx_video_views_user_updated (user_id, updated_at),
    KEY idx_video_views_video_updated (video_uuid, updated_at),
    KEY idx_video_views_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

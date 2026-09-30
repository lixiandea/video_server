-- 0001 基线：与既有库（由 GORM AutoMigrate 建起）完全一致的结构。
--
-- 全部使用 CREATE TABLE IF NOT EXISTS，因此对既有库是空操作（随后登记版本即可接管），
-- 对新库则直接建出与现状一致的表。要求 MySQL 8.0（显式指定 utf8mb4_0900_ai_ci）。
--
-- 注意：users 上的 idx_users_login_name 是历史漂移（与 init.sql 的
-- uni_users_login_name 等价），由 0002 负责清理；基线保留它以保证新旧库最终形态一致。

CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    login_name VARCHAR(255) NOT NULL,
    pwd LONGTEXT NOT NULL,
    email VARCHAR(255) DEFAULT '',
    bio VARCHAR(500) DEFAULT '',
    PRIMARY KEY (id),
    UNIQUE KEY uni_users_login_name (login_name),
    UNIQUE KEY idx_users_login_name (login_name),
    KEY idx_users_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS videos (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    uuid VARCHAR(255) NOT NULL,
    author_id BIGINT UNSIGNED NOT NULL,
    name LONGTEXT NOT NULL,
    file_path LONGTEXT NOT NULL,
    size BIGINT DEFAULT NULL,
    duration DOUBLE DEFAULT NULL,
    status VARCHAR(191) DEFAULT 'active',
    display_c_time LONGTEXT,
    description VARCHAR(500) DEFAULT '',
    cover_path VARCHAR(255) DEFAULT '',
    view_count BIGINT NOT NULL DEFAULT 0,
    like_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY idx_videos_uuid (uuid),
    KEY idx_videos_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS comments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    uuid VARCHAR(255) NOT NULL,
    author_id BIGINT UNSIGNED NOT NULL,
    video_id VARCHAR(255) NOT NULL,
    content LONGTEXT NOT NULL,
    time LONGTEXT,
    PRIMARY KEY (id),
    UNIQUE KEY idx_comments_uuid (uuid),
    KEY idx_comments_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sessions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    uuid VARCHAR(255) NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    ttl BIGINT DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_sessions_uuid (uuid),
    KEY idx_sessions_deleted_at (deleted_at),
    KEY idx_sessions_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS video_likes (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    video_uuid VARCHAR(255) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_user_video (user_id, video_uuid),
    KEY idx_video_likes_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS video_deletion_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    video_uuid VARCHAR(255) NOT NULL,
    reason LONGTEXT,
    PRIMARY KEY (id),
    UNIQUE KEY idx_video_deletion_records_video_uuid (video_uuid),
    KEY idx_video_deletion_records_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

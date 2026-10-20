-- 0006 生产加固：管理员角色、审核字段、安全刷新会话和持久转码任务。

SET @add_user_role := IF(
    EXISTS(SELECT 1 FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'users' AND column_name = 'role'),
    'DO 0',
    'ALTER TABLE users ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT ''user'', ADD KEY idx_users_role (role)'
);
PREPARE add_user_role_stmt FROM @add_user_role;
EXECUTE add_user_role_stmt;
DEALLOCATE PREPARE add_user_role_stmt;

SET @add_video_reviewed_by := IF(
    EXISTS(SELECT 1 FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'videos' AND column_name = 'reviewed_by'),
    'DO 0',
    'ALTER TABLE videos ADD COLUMN reviewed_by BIGINT UNSIGNED NULL, ADD KEY idx_videos_reviewed_by (reviewed_by)'
);
PREPARE add_video_reviewed_by_stmt FROM @add_video_reviewed_by;
EXECUTE add_video_reviewed_by_stmt;
DEALLOCATE PREPARE add_video_reviewed_by_stmt;

SET @add_video_reviewed_at := IF(
    EXISTS(SELECT 1 FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'videos' AND column_name = 'reviewed_at'),
    'DO 0',
    'ALTER TABLE videos ADD COLUMN reviewed_at DATETIME(3) NULL'
);
PREPARE add_video_reviewed_at_stmt FROM @add_video_reviewed_at;
EXECUTE add_video_reviewed_at_stmt;
DEALLOCATE PREPARE add_video_reviewed_at_stmt;

SET @add_video_review_note := IF(
    EXISTS(SELECT 1 FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'videos' AND column_name = 'review_note'),
    'DO 0',
    'ALTER TABLE videos ADD COLUMN review_note VARCHAR(500) NOT NULL DEFAULT '''''
);
PREPARE add_video_review_note_stmt FROM @add_video_review_note;
EXECUTE add_video_review_note_stmt;
DEALLOCATE PREPARE add_video_review_note_stmt;

SET @add_assets_replicated_at := IF(
    EXISTS(SELECT 1 FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'videos' AND column_name = 'assets_replicated_at'),
    'DO 0',
    'ALTER TABLE videos ADD COLUMN assets_replicated_at DATETIME(3) NULL'
);
PREPARE add_assets_replicated_at_stmt FROM @add_assets_replicated_at;
EXECUTE add_assets_replicated_at_stmt;
DEALLOCATE PREPARE add_assets_replicated_at_stmt;

SET @add_session_refresh_hash := IF(
    EXISTS(SELECT 1 FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'sessions' AND column_name = 'refresh_token_hash'),
    'DO 0',
    'ALTER TABLE sessions ADD COLUMN refresh_token_hash VARCHAR(64) NOT NULL DEFAULT '''''
);
PREPARE add_session_refresh_hash_stmt FROM @add_session_refresh_hash;
EXECUTE add_session_refresh_hash_stmt;
DEALLOCATE PREPARE add_session_refresh_hash_stmt;

SET @add_session_previous_hash := IF(
    EXISTS(SELECT 1 FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'sessions' AND column_name = 'previous_token_hash'),
    'DO 0',
    'ALTER TABLE sessions ADD COLUMN previous_token_hash VARCHAR(64) NOT NULL DEFAULT '''''
);
PREPARE add_session_previous_hash_stmt FROM @add_session_previous_hash;
EXECUTE add_session_previous_hash_stmt;
DEALLOCATE PREPARE add_session_previous_hash_stmt;

SET @add_session_last_rotated := IF(
    EXISTS(SELECT 1 FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'sessions' AND column_name = 'last_rotated_at'),
    'DO 0',
    'ALTER TABLE sessions ADD COLUMN last_rotated_at DATETIME(3) NULL'
);
PREPARE add_session_last_rotated_stmt FROM @add_session_last_rotated;
EXECUTE add_session_last_rotated_stmt;
DEALLOCATE PREPARE add_session_last_rotated_stmt;

SET @add_session_user_agent_hash := IF(
    EXISTS(SELECT 1 FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'sessions' AND column_name = 'user_agent_hash'),
    'DO 0',
    'ALTER TABLE sessions ADD COLUMN user_agent_hash VARCHAR(64) NOT NULL DEFAULT '''''
);
PREPARE add_session_user_agent_hash_stmt FROM @add_session_user_agent_hash;
EXECUTE add_session_user_agent_hash_stmt;
DEALLOCATE PREPARE add_session_user_agent_hash_stmt;

SET @add_session_ip_hash := IF(
    EXISTS(SELECT 1 FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'sessions' AND column_name = 'ip_hash'),
    'DO 0',
    'ALTER TABLE sessions ADD COLUMN ip_hash VARCHAR(64) NOT NULL DEFAULT '''''
);
PREPARE add_session_ip_hash_stmt FROM @add_session_ip_hash;
EXECUTE add_session_ip_hash_stmt;
DEALLOCATE PREPARE add_session_ip_hash_stmt;

CREATE TABLE IF NOT EXISTS transcode_task_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    task_id VARCHAR(64) NOT NULL,
    video_id VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL,
    payload LONGTEXT NOT NULL,
    retries INT NOT NULL DEFAULT 0,
    error_message VARCHAR(1000) NOT NULL DEFAULT '',
    started_at DATETIME(3) NULL,
    completed_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_transcode_task_records_task_id (task_id),
    KEY idx_transcode_task_records_video_id (video_id),
    KEY idx_transcode_task_records_status (status),
    KEY idx_transcode_task_records_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 初始化视频服务数据库
-- 说明：仅负责创建数据库与 users 基础表；其余业务表（videos/comments/sessions/
-- video_deletion_records）由应用启动时的 GORM AutoMigrate 按 internal/models 定义创建，
-- 避免手工 DDL 与模型不一致导致迁移中断（历史缺陷：video_deletion_records 缺失）。

CREATE DATABASE IF NOT EXISTS video_server CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE video_server;

-- 用户表（字段与 internal/models.User 保持一致）
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    login_name VARCHAR(255) NOT NULL,
    pwd VARCHAR(255) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uni_users_login_name (login_name),
    KEY idx_users_deleted_at (deleted_at)
);

-- 默认管理员账号（密码请在生产环境通过 /api/v1/users/register 重新创建并轮换）
INSERT IGNORE INTO users (login_name, pwd, created_at, updated_at)
VALUES ('admin', '$2a$14$E7uTR7u/.PQp9O0HObpEpO1GxbFKJr2NcU7UN.a8P1.B.3q0/Xt5.', NOW(3), NOW(3));

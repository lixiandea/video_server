-- 初始化视频服务数据库

CREATE DATABASE IF NOT EXISTS video_server CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE video_server;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    login_name VARCHAR(255) UNIQUE NOT NULL,
    pwd VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 视频信息表
CREATE TABLE IF NOT EXISTS video_info (
    id VARCHAR(255) PRIMARY KEY,
    author_id INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    display_ctime VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES users(id)
);

-- 评论表
CREATE TABLE IF NOT EXISTS comments (
    id VARCHAR(255) PRIMARY KEY,
    author_id INT NOT NULL,
    video_id VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    time VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES users(id),
    FOREIGN KEY (video_id) REFERENCES video_info(id)
);

-- 会话表
CREATE TABLE IF NOT EXISTS sessions (
    session_id VARCHAR(255) PRIMARY KEY,
    TTL BIGINT,
    login_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 视频删除记录表
CREATE TABLE IF NOT EXISTS video_del_rec (
    id VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 插入默认管理员用户
INSERT IGNORE INTO users (login_name, pwd) VALUES ('admin', '$2a$14$E7uTR7u/.PQp9O0HObpEpO1GxbFKJr2NcU7UN.a8P1.B.3q0/Xt5.');

-- 创建索引
CREATE INDEX idx_video_author ON video_info(author_id);
CREATE INDEX idx_comments_video ON comments(video_id);
CREATE INDEX idx_comments_author ON comments(author_id);
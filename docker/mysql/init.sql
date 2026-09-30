-- 初始化视频服务数据库
-- 说明：只负责创建数据库。所有表结构与索引统一由版本化迁移创建
-- （见 migrations/ 与 docs/database-migration-strategy.md），
-- 服务启动时不再执行 GORM AutoMigrate。
-- 管理员账号请通过 POST /api/v1/users/register 创建，不要在此内置口令散列。

CREATE DATABASE IF NOT EXISTS video_server CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE video_server;

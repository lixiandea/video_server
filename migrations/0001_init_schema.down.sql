-- 回滚基线会删除全部业务表，执行器默认拒绝；确需执行请设置
-- MIGRATE_ALLOW_BASELINE_DOWN=1，并先确认已有备份。
DROP TABLE IF EXISTS video_deletion_records;
DROP TABLE IF EXISTS video_likes;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS videos;
DROP TABLE IF EXISTS users;

-- 0003 增加 HLS 产物路径（videos.hls_path）。
-- MySQL 8 不支持 ADD COLUMN IF NOT EXISTS，用 information_schema 做条件 DDL，
-- 保证本版本可重复执行（对已存在该列的库是空操作）。
SET @add_hls_path := IF(
    EXISTS(
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE()
          AND table_name = 'videos'
          AND column_name = 'hls_path'
    ),
    'DO 0',
    'ALTER TABLE videos ADD COLUMN hls_path VARCHAR(255) NOT NULL DEFAULT '''''
);
PREPARE add_hls_path_stmt FROM @add_hls_path;
EXECUTE add_hls_path_stmt;
DEALLOCATE PREPARE add_hls_path_stmt;

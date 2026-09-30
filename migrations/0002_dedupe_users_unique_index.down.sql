-- 与 up 对称：恢复 GORM 按 uniqueIndex 生成的索引（同样做成条件 DDL）。
SET @restore_dup_index := IF(
    EXISTS(
        SELECT 1 FROM information_schema.statistics
        WHERE table_schema = DATABASE()
          AND table_name = 'users'
          AND index_name = 'idx_users_login_name'
    ),
    'DO 0',
    'ALTER TABLE users ADD UNIQUE INDEX idx_users_login_name (login_name)'
);

PREPARE restore_dup_index_stmt FROM @restore_dup_index;
EXECUTE restore_dup_index_stmt;
DEALLOCATE PREPARE restore_dup_index_stmt;

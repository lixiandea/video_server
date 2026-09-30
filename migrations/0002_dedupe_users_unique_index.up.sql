-- 历史漂移：init.sql 建了 uni_users_login_name，GORM AutoMigrate 又按
-- uniqueIndex 建了等价的 idx_users_login_name。这里删除后者，仅保留前者。
-- MySQL 8 不支持 DROP INDEX IF EXISTS，故用 information_schema 做条件 DDL，
-- 保证本版本可重复执行（对新库无副作用）。
SET @drop_dup_index := IF(
    EXISTS(
        SELECT 1 FROM information_schema.statistics
        WHERE table_schema = DATABASE()
          AND table_name = 'users'
          AND index_name = 'idx_users_login_name'
    ),
    'ALTER TABLE users DROP INDEX idx_users_login_name',
    'DO 0'
);

PREPARE drop_dup_index_stmt FROM @drop_dup_index;
EXECUTE drop_dup_index_stmt;
DEALLOCATE PREPARE drop_dup_index_stmt;

SET @drop_hls_path := IF(
    EXISTS(
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE()
          AND table_name = 'videos'
          AND column_name = 'hls_path'
    ),
    'ALTER TABLE videos DROP COLUMN hls_path',
    'DO 0'
);
PREPARE drop_hls_path_stmt FROM @drop_hls_path;
EXECUTE drop_hls_path_stmt;
DEALLOCATE PREPARE drop_hls_path_stmt;

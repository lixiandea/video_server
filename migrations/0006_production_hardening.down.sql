DROP TABLE IF EXISTS transcode_task_records;
ALTER TABLE sessions DROP COLUMN ip_hash, DROP COLUMN user_agent_hash, DROP COLUMN last_rotated_at, DROP COLUMN previous_token_hash, DROP COLUMN refresh_token_hash;
ALTER TABLE videos DROP COLUMN assets_replicated_at, DROP COLUMN review_note, DROP COLUMN reviewed_at, DROP COLUMN reviewed_by;
ALTER TABLE users DROP COLUMN role;

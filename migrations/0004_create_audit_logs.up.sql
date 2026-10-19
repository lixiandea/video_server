-- 0004 审计日志表：记录写操作的 who / what / result / 耗时（只追加）。
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NULL,
    request_id VARCHAR(64) DEFAULT '',
    user_id BIGINT UNSIGNED NULL,
    method VARCHAR(10) NOT NULL,
    route VARCHAR(255) DEFAULT '',
    path VARCHAR(512) DEFAULT '',
    status INT NOT NULL,
    latency_ms BIGINT NOT NULL DEFAULT 0,
    remote_addr VARCHAR(128) DEFAULT '',
    error_message VARCHAR(512) DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_audit_logs_created_at (created_at),
    KEY idx_audit_logs_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

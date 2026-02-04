-- System health history tracking table
CREATE TABLE IF NOT EXISTS system_health_history (
    id VARCHAR(36) PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(20) NOT NULL,
    cpu_usage DOUBLE PRECISION NOT NULL DEFAULT 0,
    memory_usage DOUBLE PRECISION NOT NULL DEFAULT 0,
    memory_alloc_mb DOUBLE PRECISION NOT NULL DEFAULT 0,
    memory_total_alloc_mb DOUBLE PRECISION NOT NULL DEFAULT 0,
    goroutine_count INTEGER NOT NULL DEFAULT 0,
    database_connections INTEGER NOT NULL DEFAULT 0,
    database_size_bytes BIGINT NOT NULL DEFAULT 0,
    pending_backups INTEGER NOT NULL DEFAULT 0,
    running_backups INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for efficient time-based queries
CREATE INDEX IF NOT EXISTS idx_system_health_history_timestamp
ON system_health_history(timestamp DESC);

-- Index for status-based queries
CREATE INDEX IF NOT EXISTS idx_system_health_history_status
ON system_health_history(status);

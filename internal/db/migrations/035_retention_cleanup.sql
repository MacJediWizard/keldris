-- Migration: Add configurable retention cleanup
-- The cleanup_agent_health_history() function from migration 016 uses a hard-coded
-- 30-day retention period. The application now manages retention cleanup via the
-- RetentionScheduler which calls CleanupAgentHealthHistory() daily at 3:00 AM UTC
-- with a configurable retention period (RETENTION_DAYS env var, default 90 days).

-- Update the cleanup function to accept a configurable retention period.
CREATE OR REPLACE FUNCTION cleanup_agent_health_history(retention_days INTEGER DEFAULT 90) RETURNS BIGINT AS $$
DECLARE
    deleted_count BIGINT;
BEGIN
    DELETE FROM agent_health_history
    WHERE recorded_at < NOW() - (retention_days * INTERVAL '1 day');
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

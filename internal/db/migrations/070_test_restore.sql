-- 070_test_restore.sql
-- Migration: Add test restore tables for automated backup verification testing

-- Test restore settings define per-repository test restore configuration
CREATE TABLE test_restore_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT true,
    frequency VARCHAR(50) NOT NULL DEFAULT 'weekly', -- weekly, monthly, custom
    cron_expression VARCHAR(100) NOT NULL DEFAULT '0 0 3 * * 0', -- Default: 3 AM on Sundays
    sample_percentage INTEGER NOT NULL DEFAULT 10, -- Percentage of files to restore (1-100)
    last_run_at TIMESTAMPTZ,
    last_run_status VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT sample_percentage_range CHECK (sample_percentage >= 1 AND sample_percentage <= 100),
    CONSTRAINT unique_repo_test_restore UNIQUE (repository_id)
);

-- Test restore history tracks all test restore executions
CREATE TABLE test_restore_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(255),
    sample_percentage INTEGER NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL, -- pending, running, passed, failed
    duration_ms BIGINT,
    files_restored INTEGER DEFAULT 0,
    files_verified INTEGER DEFAULT 0,
    bytes_restored BIGINT DEFAULT 0,
    error_message TEXT,
    details JSONB, -- Additional test restore details including checksums
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient queries
CREATE INDEX idx_test_restore_settings_repo ON test_restore_settings(repository_id);
CREATE INDEX idx_test_restore_settings_enabled ON test_restore_settings(enabled) WHERE enabled = true;
CREATE INDEX idx_test_restore_results_repo ON test_restore_results(repository_id);
CREATE INDEX idx_test_restore_results_status ON test_restore_results(status);
CREATE INDEX idx_test_restore_results_started ON test_restore_results(started_at DESC);
CREATE INDEX idx_test_restore_results_repo_started ON test_restore_results(repository_id, started_at DESC);

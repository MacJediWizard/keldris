-- 009_verification.sql
-- Migration: Add verification tables for backup integrity verification

-- Verification schedules define when and how to verify repositories
CREATE TABLE verification_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- check, check_read_data, test_restore
    cron_expression VARCHAR(100) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    read_data_subset VARCHAR(50), -- e.g., "2.5%" or "5G" for check_read_data
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Verification history tracks all verification runs
CREATE TABLE verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    snapshot_id VARCHAR(255), -- For test_restore type
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL, -- pending, running, passed, failed
    duration_ms BIGINT,
    error_message TEXT,
    details JSONB, -- Additional verification details
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient queries
CREATE INDEX idx_verification_schedules_repo ON verification_schedules(repository_id);
CREATE INDEX idx_verification_schedules_enabled ON verification_schedules(enabled) WHERE enabled = true;
CREATE INDEX idx_verifications_repo ON verifications(repository_id);
CREATE INDEX idx_verifications_status ON verifications(status);
CREATE INDEX idx_verifications_started ON verifications(started_at DESC);
CREATE INDEX idx_verifications_repo_started ON verifications(repository_id, started_at DESC);

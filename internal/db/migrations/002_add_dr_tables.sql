-- 002_add_dr_tables.sql
-- Migration: Add disaster recovery runbook and test tables

-- DR Runbooks - Templates for disaster recovery procedures
CREATE TABLE dr_runbooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    schedule_id UUID REFERENCES schedules(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    steps JSONB NOT NULL DEFAULT '[]',
    contacts JSONB NOT NULL DEFAULT '[]',
    credentials_location TEXT,
    recovery_time_objective_minutes INTEGER,
    recovery_point_objective_minutes INTEGER,
    status VARCHAR(50) DEFAULT 'draft',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- DR Tests - Records of disaster recovery test executions
CREATE TABLE dr_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    runbook_id UUID NOT NULL REFERENCES dr_runbooks(id) ON DELETE CASCADE,
    schedule_id UUID REFERENCES schedules(id) ON DELETE SET NULL,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    snapshot_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    restore_size_bytes BIGINT,
    restore_duration_seconds INTEGER,
    verification_passed BOOLEAN,
    notes TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- DR Test Schedules - Periodic DR test scheduling configuration
CREATE TABLE dr_test_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    runbook_id UUID NOT NULL REFERENCES dr_runbooks(id) ON DELETE CASCADE,
    cron_expression VARCHAR(100) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dr_runbooks_org ON dr_runbooks(org_id);
CREATE INDEX idx_dr_runbooks_schedule ON dr_runbooks(schedule_id);
CREATE INDEX idx_dr_tests_runbook ON dr_tests(runbook_id);
CREATE INDEX idx_dr_tests_status ON dr_tests(status);
CREATE INDEX idx_dr_tests_schedule ON dr_tests(schedule_id);
CREATE INDEX idx_dr_test_schedules_runbook ON dr_test_schedules(runbook_id);
CREATE INDEX idx_dr_test_schedules_enabled ON dr_test_schedules(enabled) WHERE enabled = true;

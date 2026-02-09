-- 025_add_onboarding.sql
-- Migration: Add onboarding progress tracking for organizations

-- Track onboarding progress per organization
CREATE TABLE onboarding_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    current_step VARCHAR(50) NOT NULL DEFAULT 'welcome',
    completed_steps TEXT[] NOT NULL DEFAULT '{}',
    skipped BOOLEAN NOT NULL DEFAULT false,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_onboarding_progress_org ON onboarding_progress(org_id);

-- Add comment explaining valid steps
COMMENT ON COLUMN onboarding_progress.current_step IS 'Valid steps: welcome, organization, smtp, repository, agent, schedule, verify, complete';
COMMENT ON COLUMN onboarding_progress.completed_steps IS 'Array of completed step names';

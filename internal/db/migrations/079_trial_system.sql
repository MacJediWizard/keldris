-- Trial system for 30-day Pro feature trials
-- Tracks trial status, extensions, and conversions per organization

-- Plan tier enumeration
DO $$ BEGIN
    CREATE TYPE plan_tier AS ENUM ('free', 'pro', 'enterprise');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Trial status enumeration
DO $$ BEGIN
    CREATE TYPE trial_status AS ENUM ('active', 'expired', 'converted', 'none');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Add trial fields to organizations table
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS plan_tier plan_tier DEFAULT 'free';
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS trial_status trial_status DEFAULT 'none';
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS trial_started_at TIMESTAMPTZ;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS trial_email VARCHAR(255);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS trial_converted_at TIMESTAMPTZ;

-- Trial extensions table for admin-granted extensions
CREATE TABLE IF NOT EXISTS trial_extensions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    extended_by UUID NOT NULL REFERENCES users(id),
    extension_days INT NOT NULL CHECK (extension_days > 0 AND extension_days <= 90),
    reason TEXT,
    previous_ends_at TIMESTAMPTZ NOT NULL,
    new_ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for efficient lookups
CREATE INDEX IF NOT EXISTS idx_trial_extensions_org_id ON trial_extensions(org_id);
CREATE INDEX IF NOT EXISTS idx_trial_extensions_created_at ON trial_extensions(created_at DESC);

-- Trial activity log for tracking feature usage during trial
CREATE TABLE IF NOT EXISTS trial_activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    feature_name VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL, -- 'accessed', 'blocked', 'limit_reached'
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for trial activity log
CREATE INDEX IF NOT EXISTS idx_trial_activity_org_id ON trial_activity_log(org_id);
CREATE INDEX IF NOT EXISTS idx_trial_activity_created_at ON trial_activity_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trial_activity_feature ON trial_activity_log(feature_name);

-- Index for finding trials about to expire
CREATE INDEX IF NOT EXISTS idx_organizations_trial_ends_at ON organizations(trial_ends_at)
    WHERE trial_status = 'active';

-- Index for plan tier filtering
CREATE INDEX IF NOT EXISTS idx_organizations_plan_tier ON organizations(plan_tier);

-- Comment on new columns
COMMENT ON COLUMN organizations.plan_tier IS 'Current subscription tier: free, pro, or enterprise';
COMMENT ON COLUMN organizations.trial_status IS 'Trial status: none (never started), active, expired, or converted';
COMMENT ON COLUMN organizations.trial_started_at IS 'When the trial was started';
COMMENT ON COLUMN organizations.trial_ends_at IS 'When the trial expires (may be extended)';
COMMENT ON COLUMN organizations.trial_email IS 'Email collected for trial signup (for follow-up)';
COMMENT ON COLUMN organizations.trial_converted_at IS 'When the trial was converted to paid subscription';

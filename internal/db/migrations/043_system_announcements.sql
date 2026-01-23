-- Migration: Add system announcements

-- Create announcements table
CREATE TABLE announcements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    type VARCHAR(50) NOT NULL DEFAULT 'info',
    dismissible BOOLEAN NOT NULL DEFAULT true,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create announcement dismissals table to track per-user dismissals
CREATE TABLE announcement_dismissals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    announcement_id UUID NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dismissed_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(announcement_id, user_id)
);

CREATE INDEX idx_announcements_org ON announcements(org_id);
CREATE INDEX idx_announcements_active ON announcements(org_id, active) WHERE active = true;
CREATE INDEX idx_announcements_schedule ON announcements(org_id, starts_at, ends_at) WHERE active = true;
CREATE INDEX idx_announcement_dismissals_user ON announcement_dismissals(user_id);
CREATE INDEX idx_announcement_dismissals_announcement ON announcement_dismissals(announcement_id);

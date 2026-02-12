-- Migration: Add branding settings for white-label feature

CREATE TABLE branding_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    logo_url VARCHAR(2048) DEFAULT '',
    favicon_url VARCHAR(2048) DEFAULT '',
    product_name VARCHAR(255) DEFAULT '',
    primary_color VARCHAR(7) DEFAULT '',
    secondary_color VARCHAR(7) DEFAULT '',
    support_url VARCHAR(2048) DEFAULT '',
    custom_css TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_branding_settings_org UNIQUE (org_id)
);

CREATE INDEX idx_branding_settings_org ON branding_settings(org_id);

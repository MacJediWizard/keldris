-- Migration: Add telemetry settings table for opt-in anonymous usage telemetry

-- Telemetry settings table (global, not per-organization)
-- This is a singleton table with exactly one row
CREATE TABLE telemetry_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    install_id UUID NOT NULL DEFAULT gen_random_uuid(),
    endpoint VARCHAR(500) DEFAULT 'https://telemetry.keldris.io/v1/collect',
    last_sent_at TIMESTAMPTZ,
    last_data JSONB,
    consent_given_at TIMESTAMPTZ,
    consent_version VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert the default row (telemetry disabled by default)
INSERT INTO telemetry_settings (enabled) VALUES (FALSE);

-- Create trigger to update updated_at
CREATE OR REPLACE FUNCTION update_telemetry_settings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_telemetry_settings_updated_at
    BEFORE UPDATE ON telemetry_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_telemetry_settings_updated_at();

-- Comment explaining the purpose
COMMENT ON TABLE telemetry_settings IS 'Global telemetry settings - opt-in anonymous usage telemetry';
COMMENT ON COLUMN telemetry_settings.enabled IS 'Whether telemetry is enabled (opt-in, default false)';
COMMENT ON COLUMN telemetry_settings.install_id IS 'Random UUID for this installation (not tied to any user/system)';
COMMENT ON COLUMN telemetry_settings.last_sent_at IS 'When telemetry was last successfully sent';
COMMENT ON COLUMN telemetry_settings.last_data IS 'Most recently collected telemetry data (for user review)';
COMMENT ON COLUMN telemetry_settings.consent_given_at IS 'When the user opted in to telemetry';
COMMENT ON COLUMN telemetry_settings.consent_version IS 'Privacy policy version that was consented to';

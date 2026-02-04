-- Migration: Add branding settings for white-label Enterprise customization

-- Insert default branding settings for existing organizations (disabled by default)
INSERT INTO system_settings (org_id, setting_key, setting_value, description)
SELECT id, 'branding',
    '{
        "enabled": false,
        "product_name": "Keldris",
        "company_name": "",
        "logo_url": "",
        "logo_dark_url": "",
        "favicon_url": "",
        "primary_color": "#4f46e5",
        "secondary_color": "#64748b",
        "accent_color": "#06b6d4",
        "support_url": "",
        "support_email": "",
        "privacy_url": "",
        "terms_url": "",
        "footer_text": "",
        "login_title": "",
        "login_subtitle": "",
        "login_bg_url": "",
        "hide_powered_by": false,
        "custom_css": ""
    }'::jsonb,
    'White-label branding configuration (Enterprise)'
FROM organizations
WHERE NOT EXISTS (
    SELECT 1 FROM system_settings
    WHERE system_settings.org_id = organizations.id
    AND system_settings.setting_key = 'branding'
);

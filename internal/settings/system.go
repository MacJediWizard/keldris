// Package settings provides system-wide configuration management.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// SettingKey represents the type of system setting.
type SettingKey string

const (
	SettingKeySMTP            SettingKey = "smtp"
	SettingKeyOIDC            SettingKey = "oidc"
	SettingKeyStorageDefaults SettingKey = "storage_defaults"
	SettingKeySecurity        SettingKey = "security"
)

// SystemSetting represents a system-wide configuration entry.
type SystemSetting struct {
	ID          uuid.UUID       `json:"id"`
	OrgID       uuid.UUID       `json:"org_id"`
	Key         SettingKey      `json:"key"`
	Value       json.RawMessage `json:"value"`
	Description string          `json:"description,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// NewSystemSetting creates a new SystemSetting with default values.
func NewSystemSetting(orgID uuid.UUID, key SettingKey, description string) *SystemSetting {
	now := time.Now()
	return &SystemSetting{
		ID:          uuid.New(),
		OrgID:       orgID,
		Key:         key,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// SMTPSettings holds SMTP server configuration.
type SMTPSettings struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	Username          string `json:"username,omitempty"`
	Password          string `json:"password,omitempty"` // Encrypted in DB
	FromEmail         string `json:"from_email"`
	FromName          string `json:"from_name,omitempty"`
	Encryption        string `json:"encryption"` // "none", "tls", "starttls"
	Enabled           bool   `json:"enabled"`
	SkipTLSVerify     bool   `json:"skip_tls_verify"`
	ConnectionTimeout int    `json:"connection_timeout_seconds"` // Default 30
}

// DefaultSMTPSettings returns SMTPSettings with sensible defaults.
func DefaultSMTPSettings() SMTPSettings {
	return SMTPSettings{
		Port:              587,
		Encryption:        "starttls",
		Enabled:           false,
		SkipTLSVerify:     false,
		ConnectionTimeout: 30,
	}
}

// Validate validates the SMTP settings.
func (s *SMTPSettings) Validate() error {
	if !s.Enabled {
		return nil // Skip validation if disabled
	}

	if s.Host == "" {
		return errors.New("SMTP host is required")
	}

	if s.Port < 1 || s.Port > 65535 {
		return errors.New("SMTP port must be between 1 and 65535")
	}

	if s.FromEmail == "" {
		return errors.New("from email is required")
	}

	if _, err := mail.ParseAddress(s.FromEmail); err != nil {
		return fmt.Errorf("invalid from email address: %w", err)
	}

	validEncryption := map[string]bool{"none": true, "tls": true, "starttls": true}
	if !validEncryption[s.Encryption] {
		return errors.New("encryption must be 'none', 'tls', or 'starttls'")
	}

	if s.ConnectionTimeout < 5 || s.ConnectionTimeout > 120 {
		return errors.New("connection timeout must be between 5 and 120 seconds")
	}

	return nil
}

// OIDCSettings holds OIDC provider configuration.
type OIDCSettings struct {
	Enabled                  bool     `json:"enabled"`
	Issuer                   string   `json:"issuer"`
	ClientID                 string   `json:"client_id"`
	ClientSecret             string   `json:"client_secret,omitempty"` // Encrypted in DB
	RedirectURL              string   `json:"redirect_url"`
	Scopes                   []string `json:"scopes"`
	AutoCreateUsers          bool     `json:"auto_create_users"`
	DefaultRole              string   `json:"default_role"` // member, readonly
	AllowedDomains           []string `json:"allowed_domains,omitempty"`
	RequireEmailVerification bool     `json:"require_email_verification"`
}

// DefaultOIDCSettings returns OIDCSettings with sensible defaults.
func DefaultOIDCSettings() OIDCSettings {
	return OIDCSettings{
		Enabled:                  false,
		Scopes:                   []string{"openid", "profile", "email"},
		AutoCreateUsers:          false,
		DefaultRole:              "member",
		RequireEmailVerification: true,
	}
}

// Validate validates the OIDC settings.
func (s *OIDCSettings) Validate() error {
	if !s.Enabled {
		return nil // Skip validation if disabled
	}

	if s.Issuer == "" {
		return errors.New("OIDC issuer is required")
	}

	if _, err := url.Parse(s.Issuer); err != nil {
		return fmt.Errorf("invalid OIDC issuer URL: %w", err)
	}

	if s.ClientID == "" {
		return errors.New("OIDC client ID is required")
	}

	if s.ClientSecret == "" {
		return errors.New("OIDC client secret is required")
	}

	if s.RedirectURL == "" {
		return errors.New("OIDC redirect URL is required")
	}

	if _, err := url.Parse(s.RedirectURL); err != nil {
		return fmt.Errorf("invalid OIDC redirect URL: %w", err)
	}

	validRoles := map[string]bool{"member": true, "readonly": true}
	if !validRoles[s.DefaultRole] {
		return errors.New("default role must be 'member' or 'readonly'")
	}

	return nil
}

// StorageDefaultSettings holds default storage configuration.
type StorageDefaultSettings struct {
	DefaultRetentionDays    int    `json:"default_retention_days"`
	MaxRetentionDays        int    `json:"max_retention_days"`
	DefaultStorageBackend   string `json:"default_storage_backend"` // "local", "s3", "b2", "sftp"
	MaxBackupSizeGB         int    `json:"max_backup_size_gb"`
	EnableCompression       bool   `json:"enable_compression"`
	CompressionLevel        int    `json:"compression_level"` // 1-9
	DefaultEncryptionMethod string `json:"default_encryption_method"` // "aes256", "none"
	PruneSchedule           string `json:"prune_schedule"` // cron expression
	AutoPruneEnabled        bool   `json:"auto_prune_enabled"`
}

// DefaultStorageSettings returns StorageDefaultSettings with sensible defaults.
func DefaultStorageSettings() StorageDefaultSettings {
	return StorageDefaultSettings{
		DefaultRetentionDays:    30,
		MaxRetentionDays:        365,
		DefaultStorageBackend:   "local",
		MaxBackupSizeGB:         100,
		EnableCompression:       true,
		CompressionLevel:        6,
		DefaultEncryptionMethod: "aes256",
		PruneSchedule:           "0 2 * * *", // 2 AM daily
		AutoPruneEnabled:        true,
	}
}

// Validate validates the storage default settings.
func (s *StorageDefaultSettings) Validate() error {
	if s.DefaultRetentionDays < 1 {
		return errors.New("default retention days must be at least 1")
	}

	if s.MaxRetentionDays < s.DefaultRetentionDays {
		return errors.New("max retention days cannot be less than default retention days")
	}

	if s.MaxRetentionDays > 3650 { // 10 years
		return errors.New("max retention days cannot exceed 3650 (10 years)")
	}

	validBackends := map[string]bool{"local": true, "s3": true, "b2": true, "sftp": true, "rest": true, "dropbox": true}
	if !validBackends[s.DefaultStorageBackend] {
		return errors.New("invalid default storage backend")
	}

	if s.MaxBackupSizeGB < 1 || s.MaxBackupSizeGB > 10000 {
		return errors.New("max backup size must be between 1 and 10000 GB")
	}

	if s.CompressionLevel < 1 || s.CompressionLevel > 9 {
		return errors.New("compression level must be between 1 and 9")
	}

	validEncryption := map[string]bool{"aes256": true, "none": true}
	if !validEncryption[s.DefaultEncryptionMethod] {
		return errors.New("encryption method must be 'aes256' or 'none'")
	}

	return nil
}

// SecuritySettings holds security configuration.
type SecuritySettings struct {
	SessionTimeoutMinutes      int      `json:"session_timeout_minutes"`
	MaxConcurrentSessions      int      `json:"max_concurrent_sessions"`
	RequireMFA                 bool     `json:"require_mfa"`
	MFAGracePeriodDays         int      `json:"mfa_grace_period_days"`
	AllowedIPRanges            []string `json:"allowed_ip_ranges,omitempty"` // CIDR format
	BlockedIPRanges            []string `json:"blocked_ip_ranges,omitempty"` // CIDR format
	FailedLoginLockoutAttempts int      `json:"failed_login_lockout_attempts"`
	FailedLoginLockoutMinutes  int      `json:"failed_login_lockout_minutes"`
	APIKeyExpirationDays       int      `json:"api_key_expiration_days"` // 0 means no expiration
	EnableAuditLogging         bool     `json:"enable_audit_logging"`
	AuditLogRetentionDays      int      `json:"audit_log_retention_days"`
	ForceHTTPS                 bool     `json:"force_https"`
	AllowPasswordLogin         bool     `json:"allow_password_login"`
}

// DefaultSecuritySettings returns SecuritySettings with sensible defaults.
func DefaultSecuritySettings() SecuritySettings {
	return SecuritySettings{
		SessionTimeoutMinutes:      480, // 8 hours
		MaxConcurrentSessions:      5,
		RequireMFA:                 false,
		MFAGracePeriodDays:         7,
		FailedLoginLockoutAttempts: 5,
		FailedLoginLockoutMinutes:  30,
		APIKeyExpirationDays:       0, // No expiration
		EnableAuditLogging:         true,
		AuditLogRetentionDays:      90,
		ForceHTTPS:                 true,
		AllowPasswordLogin:         true,
	}
}

// Validate validates the security settings.
func (s *SecuritySettings) Validate() error {
	if s.SessionTimeoutMinutes < 5 || s.SessionTimeoutMinutes > 10080 { // 5 min to 7 days
		return errors.New("session timeout must be between 5 minutes and 7 days (10080 minutes)")
	}

	if s.MaxConcurrentSessions < 1 || s.MaxConcurrentSessions > 100 {
		return errors.New("max concurrent sessions must be between 1 and 100")
	}

	if s.MFAGracePeriodDays < 0 || s.MFAGracePeriodDays > 30 {
		return errors.New("MFA grace period must be between 0 and 30 days")
	}

	// Validate CIDR ranges
	for _, cidr := range s.AllowedIPRanges {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid allowed IP range '%s': %w", cidr, err)
		}
	}

	for _, cidr := range s.BlockedIPRanges {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid blocked IP range '%s': %w", cidr, err)
		}
	}

	if s.FailedLoginLockoutAttempts < 1 || s.FailedLoginLockoutAttempts > 20 {
		return errors.New("failed login lockout attempts must be between 1 and 20")
	}

	if s.FailedLoginLockoutMinutes < 1 || s.FailedLoginLockoutMinutes > 1440 { // 24 hours
		return errors.New("failed login lockout minutes must be between 1 and 1440")
	}

	if s.APIKeyExpirationDays < 0 || s.APIKeyExpirationDays > 365 {
		return errors.New("API key expiration days must be between 0 (no expiration) and 365")
	}

	if s.AuditLogRetentionDays < 7 || s.AuditLogRetentionDays > 3650 {
		return errors.New("audit log retention must be between 7 and 3650 days")
	}

	return nil
}

// Request/Response types for API

// UpdateSMTPSettingsRequest is the request for updating SMTP settings.
type UpdateSMTPSettingsRequest struct {
	Host              *string `json:"host,omitempty" binding:"omitempty,max=255"`
	Port              *int    `json:"port,omitempty" binding:"omitempty,min=1,max=65535"`
	Username          *string `json:"username,omitempty" binding:"omitempty,max=255"`
	Password          *string `json:"password,omitempty" binding:"omitempty,max=255"`
	FromEmail         *string `json:"from_email,omitempty" binding:"omitempty,email,max=255"`
	FromName          *string `json:"from_name,omitempty" binding:"omitempty,max=255"`
	Encryption        *string `json:"encryption,omitempty" binding:"omitempty,oneof=none tls starttls"`
	Enabled           *bool   `json:"enabled,omitempty"`
	SkipTLSVerify     *bool   `json:"skip_tls_verify,omitempty"`
	ConnectionTimeout *int    `json:"connection_timeout_seconds,omitempty" binding:"omitempty,min=5,max=120"`
}

// UpdateOIDCSettingsRequest is the request for updating OIDC settings.
type UpdateOIDCSettingsRequest struct {
	Enabled                  *bool    `json:"enabled,omitempty"`
	Issuer                   *string  `json:"issuer,omitempty" binding:"omitempty,url,max=500"`
	ClientID                 *string  `json:"client_id,omitempty" binding:"omitempty,max=255"`
	ClientSecret             *string  `json:"client_secret,omitempty" binding:"omitempty,max=500"`
	RedirectURL              *string  `json:"redirect_url,omitempty" binding:"omitempty,url,max=500"`
	Scopes                   []string `json:"scopes,omitempty"`
	AutoCreateUsers          *bool    `json:"auto_create_users,omitempty"`
	DefaultRole              *string  `json:"default_role,omitempty" binding:"omitempty,oneof=member readonly"`
	AllowedDomains           []string `json:"allowed_domains,omitempty"`
	RequireEmailVerification *bool    `json:"require_email_verification,omitempty"`
}

// UpdateStorageDefaultsRequest is the request for updating storage defaults.
type UpdateStorageDefaultsRequest struct {
	DefaultRetentionDays    *int    `json:"default_retention_days,omitempty" binding:"omitempty,min=1,max=3650"`
	MaxRetentionDays        *int    `json:"max_retention_days,omitempty" binding:"omitempty,min=1,max=3650"`
	DefaultStorageBackend   *string `json:"default_storage_backend,omitempty" binding:"omitempty,oneof=local s3 b2 sftp rest dropbox"`
	MaxBackupSizeGB         *int    `json:"max_backup_size_gb,omitempty" binding:"omitempty,min=1,max=10000"`
	EnableCompression       *bool   `json:"enable_compression,omitempty"`
	CompressionLevel        *int    `json:"compression_level,omitempty" binding:"omitempty,min=1,max=9"`
	DefaultEncryptionMethod *string `json:"default_encryption_method,omitempty" binding:"omitempty,oneof=aes256 none"`
	PruneSchedule           *string `json:"prune_schedule,omitempty" binding:"omitempty,max=100"`
	AutoPruneEnabled        *bool   `json:"auto_prune_enabled,omitempty"`
}

// UpdateSecuritySettingsRequest is the request for updating security settings.
type UpdateSecuritySettingsRequest struct {
	SessionTimeoutMinutes      *int     `json:"session_timeout_minutes,omitempty" binding:"omitempty,min=5,max=10080"`
	MaxConcurrentSessions      *int     `json:"max_concurrent_sessions,omitempty" binding:"omitempty,min=1,max=100"`
	RequireMFA                 *bool    `json:"require_mfa,omitempty"`
	MFAGracePeriodDays         *int     `json:"mfa_grace_period_days,omitempty" binding:"omitempty,min=0,max=30"`
	AllowedIPRanges            []string `json:"allowed_ip_ranges,omitempty"`
	BlockedIPRanges            []string `json:"blocked_ip_ranges,omitempty"`
	FailedLoginLockoutAttempts *int     `json:"failed_login_lockout_attempts,omitempty" binding:"omitempty,min=1,max=20"`
	FailedLoginLockoutMinutes  *int     `json:"failed_login_lockout_minutes,omitempty" binding:"omitempty,min=1,max=1440"`
	APIKeyExpirationDays       *int     `json:"api_key_expiration_days,omitempty" binding:"omitempty,min=0,max=365"`
	EnableAuditLogging         *bool    `json:"enable_audit_logging,omitempty"`
	AuditLogRetentionDays      *int     `json:"audit_log_retention_days,omitempty" binding:"omitempty,min=7,max=3650"`
	ForceHTTPS                 *bool    `json:"force_https,omitempty"`
	AllowPasswordLogin         *bool    `json:"allow_password_login,omitempty"`
}

// SystemSettingsResponse is the response containing all system settings.
type SystemSettingsResponse struct {
	SMTP            SMTPSettings           `json:"smtp"`
	OIDC            OIDCSettings           `json:"oidc"`
	StorageDefaults StorageDefaultSettings `json:"storage_defaults"`
	Security        SecuritySettings       `json:"security"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// TestSMTPRequest is the request for testing SMTP connection.
type TestSMTPRequest struct {
	RecipientEmail string `json:"recipient_email" binding:"required,email"`
}

// TestSMTPResponse is the response for SMTP test.
type TestSMTPResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TestOIDCResponse is the response for OIDC test.
type TestOIDCResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	ProviderName  string `json:"provider_name,omitempty"`
	AuthURL       string `json:"auth_url,omitempty"`
	SupportedFlow string `json:"supported_flow,omitempty"`
}

// SettingsAuditLog records changes to system settings.
type SettingsAuditLog struct {
	ID          uuid.UUID       `json:"id"`
	OrgID       uuid.UUID       `json:"org_id"`
	SettingKey  SettingKey      `json:"setting_key"`
	OldValue    json.RawMessage `json:"old_value,omitempty"`
	NewValue    json.RawMessage `json:"new_value"`
	ChangedBy   uuid.UUID       `json:"changed_by"`
	ChangedByEmail string       `json:"changed_by_email,omitempty"`
	ChangedAt   time.Time       `json:"changed_at"`
	IPAddress   string          `json:"ip_address,omitempty"`
}

// NewSettingsAuditLog creates a new audit log entry for settings changes.
func NewSettingsAuditLog(orgID uuid.UUID, key SettingKey, oldValue, newValue json.RawMessage, changedBy uuid.UUID, ipAddress string) *SettingsAuditLog {
	return &SettingsAuditLog{
		ID:         uuid.New(),
		OrgID:      orgID,
		SettingKey: key,
		OldValue:   oldValue,
		NewValue:   newValue,
		ChangedBy:  changedBy,
		ChangedAt:  time.Now(),
		IPAddress:  ipAddress,
	}
}

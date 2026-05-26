package settings

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewSystemSetting(t *testing.T) {
	orgID := uuid.New()
	s := NewSystemSetting(orgID, SettingKeySMTP, "test description")

	if s.OrgID != orgID {
		t.Error("expected OrgID match")
	}
	if s.Key != SettingKeySMTP {
		t.Errorf("expected key SMTP, got %s", s.Key)
	}
	if s.Description != "test description" {
		t.Error("expected description match")
	}
	if s.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
}

func TestSMTPSettings_DefaultsValid(t *testing.T) {
	s := DefaultSMTPSettings()
	if s.Port != 587 {
		t.Errorf("expected port 587, got %d", s.Port)
	}
	if s.Encryption != "starttls" {
		t.Errorf("expected encryption starttls, got %s", s.Encryption)
	}
	if s.ConnectionTimeout != 30 {
		t.Errorf("expected timeout 30, got %d", s.ConnectionTimeout)
	}
}

func TestSMTPSettings_DisabledSkipsValidation(t *testing.T) {
	s := &SMTPSettings{Enabled: false}
	if err := s.Validate(); err != nil {
		t.Errorf("disabled SMTP should skip validation: %v", err)
	}
}

func TestSMTPSettings_ValidatesHostRequired(t *testing.T) {
	s := DefaultSMTPSettings()
	s.Enabled = true
	s.FromEmail = "from@example.com"
	if err := s.Validate(); err == nil {
		t.Error("expected error for missing host")
	}
}

func TestSMTPSettings_ValidatesPortRange(t *testing.T) {
	s := DefaultSMTPSettings()
	s.Enabled = true
	s.Host = "smtp.example.com"
	s.FromEmail = "from@example.com"
	s.Port = 0
	if err := s.Validate(); err == nil {
		t.Error("expected error for port 0")
	}

	s.Port = 70000
	if err := s.Validate(); err == nil {
		t.Error("expected error for port > 65535")
	}
}

func TestSMTPSettings_ValidatesFromEmailFormat(t *testing.T) {
	s := DefaultSMTPSettings()
	s.Enabled = true
	s.Host = "smtp.example.com"
	s.FromEmail = "not-an-email"
	if err := s.Validate(); err == nil {
		t.Error("expected error for invalid from email")
	}
}

func TestSMTPSettings_ValidatesEncryption(t *testing.T) {
	s := DefaultSMTPSettings()
	s.Enabled = true
	s.Host = "smtp.example.com"
	s.FromEmail = "from@example.com"
	s.Encryption = "bogus"
	if err := s.Validate(); err == nil {
		t.Error("expected error for invalid encryption")
	}
}

func TestSMTPSettings_ValidatesConnectionTimeout(t *testing.T) {
	s := DefaultSMTPSettings()
	s.Enabled = true
	s.Host = "smtp.example.com"
	s.FromEmail = "from@example.com"
	s.ConnectionTimeout = 1
	if err := s.Validate(); err == nil {
		t.Error("expected error for timeout < 5")
	}

	s.ConnectionTimeout = 200
	if err := s.Validate(); err == nil {
		t.Error("expected error for timeout > 120")
	}
}

func TestSMTPSettings_AllValid(t *testing.T) {
	s := DefaultSMTPSettings()
	s.Enabled = true
	s.Host = "smtp.example.com"
	s.FromEmail = "from@example.com"
	if err := s.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestOIDCSettings_DefaultsValid(t *testing.T) {
	s := DefaultOIDCSettings()
	if s.Enabled {
		t.Error("expected disabled by default")
	}
	if len(s.Scopes) != 3 {
		t.Errorf("expected 3 default scopes, got %d", len(s.Scopes))
	}
}

func TestOIDCSettings_DisabledSkipsValidation(t *testing.T) {
	s := &OIDCSettings{Enabled: false}
	if err := s.Validate(); err != nil {
		t.Errorf("disabled OIDC should skip validation: %v", err)
	}
}

func TestOIDCSettings_RequiresFields(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*OIDCSettings)
	}{
		{"missing issuer", func(s *OIDCSettings) {}},
		{"missing client ID", func(s *OIDCSettings) { s.Issuer = "https://x" }},
		{"missing client secret", func(s *OIDCSettings) {
			s.Issuer = "https://x"
			s.ClientID = "c"
		}},
		{"missing redirect URL", func(s *OIDCSettings) {
			s.Issuer = "https://x"
			s.ClientID = "c"
			s.ClientSecret = "s"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := DefaultOIDCSettings()
			s.Enabled = true
			tc.setup(&s)
			if err := s.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestOIDCSettings_ValidatesDefaultRole(t *testing.T) {
	s := DefaultOIDCSettings()
	s.Enabled = true
	s.Issuer = "https://x"
	s.ClientID = "c"
	s.ClientSecret = "s"
	s.RedirectURL = "https://x/cb"
	s.DefaultRole = "bogus"
	if err := s.Validate(); err == nil {
		t.Error("expected error for invalid role")
	}
}

func TestStorageDefaultSettings_DefaultsValid(t *testing.T) {
	s := DefaultStorageSettings()
	if s.DefaultRetentionDays != 30 {
		t.Errorf("expected 30, got %d", s.DefaultRetentionDays)
	}
	if err := s.Validate(); err != nil {
		t.Errorf("defaults should validate, got %v", err)
	}
}

func TestStorageDefaultSettings_ValidationBranches(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*StorageDefaultSettings)
	}{
		{"retention too small", func(s *StorageDefaultSettings) { s.DefaultRetentionDays = 0 }},
		{"max < default", func(s *StorageDefaultSettings) { s.MaxRetentionDays = 1 }},
		{"max too big", func(s *StorageDefaultSettings) { s.MaxRetentionDays = 4000 }},
		{"invalid backend", func(s *StorageDefaultSettings) { s.DefaultStorageBackend = "bogus" }},
		{"size 0", func(s *StorageDefaultSettings) { s.MaxBackupSizeGB = 0 }},
		{"compression out of range", func(s *StorageDefaultSettings) { s.CompressionLevel = 10 }},
		{"invalid encryption", func(s *StorageDefaultSettings) { s.DefaultEncryptionMethod = "bogus" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := DefaultStorageSettings()
			tc.setup(&s)
			if err := s.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestSecuritySettings_DefaultsValid(t *testing.T) {
	s := DefaultSecuritySettings()
	if s.SessionTimeoutMinutes != 480 {
		t.Errorf("expected 480, got %d", s.SessionTimeoutMinutes)
	}
	if !s.EnableAuditLogging {
		t.Error("expected audit logging enabled by default")
	}
}

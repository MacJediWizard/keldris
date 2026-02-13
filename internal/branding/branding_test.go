package branding

import (
	"context"
	"errors"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

type mockStore struct {
	settings *models.BrandingSettings
	getErr   error
}

func (m *mockStore) GetBrandingSettings(_ context.Context, _ uuid.UUID) (*models.BrandingSettings, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.settings, nil
}

func (m *mockStore) UpsertBrandingSettings(_ context.Context, b *models.BrandingSettings) error {
	m.settings = b
	return nil
}

func (m *mockStore) DeleteBrandingSettings(_ context.Context, _ uuid.UUID) error {
	m.settings = nil
	return nil
}

func TestValidateColor(t *testing.T) {
	tests := []struct {
		name    string
		color   string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"valid 6-char hex", "#FF00FF", false},
		{"valid 3-char hex", "#F0F", false},
		{"lowercase hex", "#abc123", false},
		{"missing hash", "FF00FF", true},
		{"invalid chars", "#GGGGGG", true},
		{"too short", "#FF", true},
		{"too long", "#FF00FF00", true},
		{"text", "red", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateColor(tt.color)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateColor(%q) error = %v, wantErr %v", tt.color, err, tt.wantErr)
			}
		})
	}
}

func TestValidateColors(t *testing.T) {
	t.Run("both valid", func(t *testing.T) {
		if err := ValidateColors("#FFF", "#000"); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
	t.Run("primary invalid", func(t *testing.T) {
		if err := ValidateColors("bad", "#000"); err == nil {
			t.Error("expected error for invalid primary")
		}
	})
	t.Run("secondary invalid", func(t *testing.T) {
		if err := ValidateColors("#000", "bad"); err == nil {
			t.Error("expected error for invalid secondary")
		}
	})
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"valid https", "https://example.com/logo.png", false},
		{"valid http", "http://example.com/logo.png", false},
		{"ftp not allowed", "ftp://example.com/file", true},
		{"no scheme", "example.com/logo.png", true},
		{"invalid url", "not a url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestLoadBranding(t *testing.T) {
	orgID := uuid.New()

	t.Run("returns defaults when no settings", func(t *testing.T) {
		store := &mockStore{}
		b, err := LoadBranding(context.Background(), store, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.OrgID != orgID {
			t.Errorf("expected org_id %s, got %s", orgID, b.OrgID)
		}
		if b.ProductName != "" {
			t.Errorf("expected empty product name, got %q", b.ProductName)
		}
	})

	t.Run("returns existing settings", func(t *testing.T) {
		settings := &models.BrandingSettings{
			ID:          uuid.New(),
			OrgID:       orgID,
			ProductName: "Custom",
		}
		store := &mockStore{settings: settings}
		b, err := LoadBranding(context.Background(), store, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.ProductName != "Custom" {
			t.Errorf("expected Custom, got %q", b.ProductName)
		}
	})

	t.Run("returns error on store failure", func(t *testing.T) {
		store := &mockStore{getErr: errors.New("db error")}
		_, err := LoadBranding(context.Background(), store, orgID)
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestGetDefaultBranding(t *testing.T) {
	orgID := uuid.New()
	b := GetDefaultBranding(orgID)
	if b.OrgID != orgID {
		t.Errorf("expected org_id %s, got %s", orgID, b.OrgID)
	}
	if b.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
}

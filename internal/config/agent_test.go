package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     AgentConfig
		wantErr bool
	}{
		{
			name:    "empty config",
			cfg:     AgentConfig{},
			wantErr: true,
		},
		{
			name: "missing api_key",
			cfg: AgentConfig{
				ServerURL: "https://example.com",
			},
			wantErr: true,
		},
		{
			name: "missing server_url",
			cfg: AgentConfig{
				APIKey: "test-key",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: AgentConfig{
				ServerURL: "https://example.com",
				APIKey:    "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  AgentConfig
		want bool
	}{
		{
			name: "empty config",
			cfg:  AgentConfig{},
			want: false,
		},
		{
			name: "partial config",
			cfg: AgentConfig{
				ServerURL: "https://example.com",
			},
			want: false,
		},
		{
			name: "configured",
			cfg: AgentConfig{
				ServerURL: "https://example.com",
				APIKey:    "test-key",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad_NonExistent(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	if cfg.ServerURL != "" || cfg.APIKey != "" {
		t.Error("Load() expected empty config for non-existent file")
	}
}

func TestAgentConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.yml")

	original := &AgentConfig{
		ServerURL: "https://backup.example.com",
		APIKey:    "secret-key-12345",
		AgentID:   "agent-uuid",
		Hostname:  "testhost",
	}

	// Save config
	if err := original.Save(configPath); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	// Check that file is not world-readable (0600 on Unix)
	if info.Mode().Perm()&0077 != 0 {
		t.Errorf("Config file has insecure permissions: %v", info.Mode())
	}

	// Load config
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify fields
	if loaded.ServerURL != original.ServerURL {
		t.Errorf("ServerURL = %q, want %q", loaded.ServerURL, original.ServerURL)
	}
	if loaded.APIKey != original.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, original.APIKey)
	}
	if loaded.AgentID != original.AgentID {
		t.Errorf("AgentID = %q, want %q", loaded.AgentID, original.AgentID)
	}
	if loaded.Hostname != original.Hostname {
		t.Errorf("Hostname = %q, want %q", loaded.Hostname, original.Hostname)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Write invalid YAML
	if err := os.WriteFile(configPath, []byte("not: valid: yaml: {{"), 0600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() expected error for invalid YAML")
	}
}

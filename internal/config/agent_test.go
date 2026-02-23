package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     AgentConfig
		wantErr string
	}{
		{
			name:    "empty config",
			cfg:     AgentConfig{},
			wantErr: "server_url is required",
		},
		{
			name: "missing api_key",
			cfg: AgentConfig{
				ServerURL: "https://example.com",
			},
			wantErr: "api_key is required",
		},
		{
			name: "missing server_url",
			cfg: AgentConfig{
				APIKey: "test-key",
			},
			wantErr: "server_url is required",
		},
		{
			name: "valid config",
			cfg: AgentConfig{
				ServerURL: "https://example.com",
				APIKey:    "test-key",
			},
			wantErr: "",
		},
		{
			name: "valid config with all fields",
			cfg: AgentConfig{
				ServerURL:       "https://example.com",
				APIKey:          "test-key",
				AgentID:         "agent-123",
				Hostname:        "myhost",
				AutoCheckUpdate: true,
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Validate() error = %q, want containing %q", err.Error(), tt.wantErr)
				}
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
			name: "only api_key",
			cfg: AgentConfig{
				APIKey: "test-key",
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

func TestDefaultConfigDir(t *testing.T) {
	dir, err := DefaultConfigDir()
	if err != nil {
		t.Fatalf("DefaultConfigDir() unexpected error: %v", err)
	}
	if dir == "" {
		t.Fatal("DefaultConfigDir() returned empty string")
	}
	if !strings.HasSuffix(dir, ".keldris") {
		t.Errorf("DefaultConfigDir() = %q, want suffix .keldris", dir)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() unexpected error: %v", err)
	}
	if path == "" {
		t.Fatal("DefaultConfigPath() returned empty string")
	}
	if !strings.HasSuffix(path, filepath.Join(".keldris", "config.yml")) {
		t.Errorf("DefaultConfigPath() = %q, want suffix .keldris/config.yml", path)
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

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	if err := os.WriteFile(configPath, []byte("not: valid: yaml: {{"), 0600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parse config file") {
		t.Errorf("Load() error = %q, want containing 'parse config file'", err.Error())
	}
}

func TestLoad_UnreadableFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	if err := os.WriteFile(configPath, []byte("server_url: test"), 0600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	// Remove read permission
	if err := os.Chmod(configPath, 0000); err != nil {
		t.Fatalf("Chmod() error: %v", err)
	}
	t.Cleanup(func() {
		os.Chmod(configPath, 0600) // restore for cleanup
	})

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() expected error for unreadable file")
	}
	if !strings.Contains(err.Error(), "read config file") {
		t.Errorf("Load() error = %q, want containing 'read config file'", err.Error())
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	if cfg.ServerURL != "" {
		t.Errorf("Load() ServerURL = %q, want empty", cfg.ServerURL)
	}
}

func TestLoad_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	yaml := `server_url: https://backup.example.com
api_key: secret-key-12345
agent_id: agent-uuid-123
hostname: testhost
auto_check_update: true
`
	if err := os.WriteFile(configPath, []byte(yaml), 0600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.ServerURL != "https://backup.example.com" {
		t.Errorf("ServerURL = %q, want %q", cfg.ServerURL, "https://backup.example.com")
	}
	if cfg.APIKey != "secret-key-12345" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "secret-key-12345")
	}
	if cfg.AgentID != "agent-uuid-123" {
		t.Errorf("AgentID = %q, want %q", cfg.AgentID, "agent-uuid-123")
	}
	if cfg.Hostname != "testhost" {
		t.Errorf("Hostname = %q, want %q", cfg.Hostname, "testhost")
	}
	if !cfg.AutoCheckUpdate {
		t.Error("AutoCheckUpdate = false, want true")
	}
}

func TestAgentConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.yml")

	original := &AgentConfig{
		ServerURL:       "https://backup.example.com",
		APIKey:          "secret-key-12345",
		AgentID:         "agent-uuid",
		Hostname:        "testhost",
		AutoCheckUpdate: true,
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
	if loaded.AutoCheckUpdate != original.AutoCheckUpdate {
		t.Errorf("AutoCheckUpdate = %v, want %v", loaded.AutoCheckUpdate, original.AutoCheckUpdate)
	}
}

func TestAgentConfig_Save_CreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "a", "b", "c", "config.yml")

	cfg := &AgentConfig{
		ServerURL: "https://example.com",
		APIKey:    "key",
	}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("Save() did not create file: %v", err)
	}
}

func TestAgentConfig_Save_UnwritableDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0500); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	t.Cleanup(func() {
		os.Chmod(readOnlyDir, 0700)
	})

	configPath := filepath.Join(readOnlyDir, "subdir", "config.yml")

	cfg := &AgentConfig{
		ServerURL: "https://example.com",
		APIKey:    "key",
	}

	err := cfg.Save(configPath)
	if err == nil {
		t.Error("Save() expected error for unwritable directory")
	}
	if !strings.Contains(err.Error(), "create config directory") {
		t.Errorf("Save() error = %q, want containing 'create config directory'", err.Error())
	}
}

func TestAgentConfig_Save_OverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Save initial config
	initial := &AgentConfig{
		ServerURL: "https://old.example.com",
		APIKey:    "old-key",
	}
	if err := initial.Save(configPath); err != nil {
		t.Fatalf("Save() initial error: %v", err)
	}

	// Save updated config
	updated := &AgentConfig{
		ServerURL: "https://new.example.com",
		APIKey:    "new-key",
		AgentID:   "new-agent",
	}
	if err := updated.Save(configPath); err != nil {
		t.Fatalf("Save() updated error: %v", err)
	}

	// Load and verify updated values
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.ServerURL != "https://new.example.com" {
		t.Errorf("ServerURL = %q, want %q", loaded.ServerURL, "https://new.example.com")
	}
	if loaded.APIKey != "new-key" {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, "new-key")
	}
	if loaded.AgentID != "new-agent" {
		t.Errorf("AgentID = %q, want %q", loaded.AgentID, "new-agent")
	}
}

func TestLoadDefault(t *testing.T) {
	// LoadDefault uses os.UserHomeDir internally, which should work
	cfg, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() unexpected error: %v", err)
	}
	// Should return a config (possibly empty if no file exists at default path)
	if cfg == nil {
		t.Fatal("LoadDefault() returned nil config")
	}
}

func TestSaveDefault(t *testing.T) {
	// Get the default path first
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error: %v", err)
	}

	// Check if a config already exists at the default path
	_, existsErr := os.Stat(path)
	fileExisted := existsErr == nil

	// If the file exists, read original contents to restore later
	var originalContent []byte
	if fileExisted {
		originalContent, err = os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error: %v", err)
		}
	}

	cfg := &AgentConfig{
		ServerURL: "https://test-save-default.example.com",
		APIKey:    "test-save-default-key",
	}

	if err := cfg.SaveDefault(); err != nil {
		t.Fatalf("SaveDefault() error: %v", err)
	}

	// Verify by loading
	loaded, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error: %v", err)
	}
	if loaded.ServerURL != cfg.ServerURL {
		t.Errorf("ServerURL = %q, want %q", loaded.ServerURL, cfg.ServerURL)
	}

	// Restore original state
	if fileExisted {
		if err := os.WriteFile(path, originalContent, 0600); err != nil {
			t.Fatalf("Failed to restore original config: %v", err)
		}
	} else {
		os.Remove(path)
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Config with only some fields set
	yaml := `server_url: https://example.com
hostname: myhost
`
	if err := os.WriteFile(configPath, []byte(yaml), 0600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.ServerURL != "https://example.com" {
		t.Errorf("ServerURL = %q, want %q", cfg.ServerURL, "https://example.com")
	}
	if cfg.Hostname != "myhost" {
		t.Errorf("Hostname = %q, want %q", cfg.Hostname, "myhost")
	}
	// Unset fields should be zero values
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", cfg.APIKey)
	}
	if cfg.AgentID != "" {
		t.Errorf("AgentID = %q, want empty", cfg.AgentID)
	}
	if cfg.AutoCheckUpdate {
		t.Error("AutoCheckUpdate = true, want false")
	}
}

func TestAgentConfig_Save_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	cfg := &AgentConfig{}
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.ServerURL != "" || loaded.APIKey != "" || loaded.AgentID != "" || loaded.Hostname != "" {
		t.Error("Empty config should round-trip as empty")
	}
}

func TestAgentConfig_Validate_ErrorMessages(t *testing.T) {
	// Verify the exact error messages returned
	cfg := AgentConfig{}
	err := cfg.Validate()
	if err == nil || err.Error() != "server_url is required" {
		t.Errorf("Validate() with empty config: error = %v, want 'server_url is required'", err)
	}

	cfg.ServerURL = "https://example.com"
	err = cfg.Validate()
	if err == nil || err.Error() != "api_key is required" {
		t.Errorf("Validate() with only server_url: error = %v, want 'api_key is required'", err)
	}
}

package config

import (
	"os"
	"testing"
)

func TestLoadServerConfig_DefaultEnvironment(t *testing.T) {
	os.Unsetenv("ENV")
	cfg := LoadServerConfig()
	if cfg.Environment != EnvDevelopment {
		t.Errorf("expected %q, got %q", EnvDevelopment, cfg.Environment)
	}
}

func TestLoadServerConfig_InvalidEnvironment(t *testing.T) {
	t.Setenv("ENV", "invalid")
	cfg := LoadServerConfig()
	if cfg.Environment != EnvDevelopment {
		t.Errorf("expected %q for invalid ENV, got %q", EnvDevelopment, cfg.Environment)
	}
}

func TestLoadServerConfig_ValidEnvironments(t *testing.T) {
	tests := []struct {
		env  string
		want Environment
	}{
		{"development", EnvDevelopment},
		{"staging", EnvStaging},
		{"production", EnvProduction},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("ENV", tt.env)
			cfg := LoadServerConfig()
			if cfg.Environment != tt.want {
				t.Errorf("expected %q, got %q", tt.want, cfg.Environment)
			}
		})
	}
}

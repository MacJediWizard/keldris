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

func TestLoadServerConfig_SessionMaxAge(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("SESSION_MAX_AGE", "")
		cfg := LoadServerConfig()
		if cfg.SessionMaxAge != 86400 {
			t.Errorf("expected default SessionMaxAge 86400, got %d", cfg.SessionMaxAge)
		}
	})

	t.Run("custom value", func(t *testing.T) {
		t.Setenv("SESSION_MAX_AGE", "3600")
		cfg := LoadServerConfig()
		if cfg.SessionMaxAge != 3600 {
			t.Errorf("expected SessionMaxAge 3600, got %d", cfg.SessionMaxAge)
		}
	})

	t.Run("invalid value falls back to default", func(t *testing.T) {
		t.Setenv("SESSION_MAX_AGE", "notanumber")
		cfg := LoadServerConfig()
		if cfg.SessionMaxAge != 86400 {
			t.Errorf("expected default SessionMaxAge 86400 for invalid input, got %d", cfg.SessionMaxAge)
		}
	})

	t.Run("negative value falls back to default", func(t *testing.T) {
		t.Setenv("SESSION_MAX_AGE", "-1")
		cfg := LoadServerConfig()
		if cfg.SessionMaxAge != 86400 {
			t.Errorf("expected default SessionMaxAge 86400 for negative input, got %d", cfg.SessionMaxAge)
		}
	})
}

func TestLoadServerConfig_SessionIdleTimeout(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("SESSION_IDLE_TIMEOUT", "")
		cfg := LoadServerConfig()
		if cfg.SessionIdleTimeout != 1800 {
			t.Errorf("expected default SessionIdleTimeout 1800, got %d", cfg.SessionIdleTimeout)
		}
	})

	t.Run("custom value", func(t *testing.T) {
		t.Setenv("SESSION_IDLE_TIMEOUT", "900")
		cfg := LoadServerConfig()
		if cfg.SessionIdleTimeout != 900 {
			t.Errorf("expected SessionIdleTimeout 900, got %d", cfg.SessionIdleTimeout)
		}
	})

	t.Run("zero disables idle timeout", func(t *testing.T) {
		t.Setenv("SESSION_IDLE_TIMEOUT", "0")
		cfg := LoadServerConfig()
		if cfg.SessionIdleTimeout != 0 {
			t.Errorf("expected SessionIdleTimeout 0, got %d", cfg.SessionIdleTimeout)
		}
	})

	t.Run("invalid value falls back to default", func(t *testing.T) {
		t.Setenv("SESSION_IDLE_TIMEOUT", "notanumber")
		cfg := LoadServerConfig()
		if cfg.SessionIdleTimeout != 1800 {
			t.Errorf("expected default SessionIdleTimeout 1800 for invalid input, got %d", cfg.SessionIdleTimeout)
		}
	})

	t.Run("negative value falls back to default", func(t *testing.T) {
		t.Setenv("SESSION_IDLE_TIMEOUT", "-1")
		cfg := LoadServerConfig()
		if cfg.SessionIdleTimeout != 1800 {
			t.Errorf("expected default SessionIdleTimeout 1800 for negative input, got %d", cfg.SessionIdleTimeout)
		}
	})
}

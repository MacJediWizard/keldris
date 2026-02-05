// Package config provides configuration management for Keldris.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds the server's configuration.
type ServerConfig struct {
	// HTTPAddr is the address to listen on for HTTP connections.
	HTTPAddr string `yaml:"http_addr,omitempty"`

	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string `yaml:"database_url,omitempty"`

	// Shutdown configuration
	Shutdown ShutdownConfig `yaml:"shutdown,omitempty"`

	// OIDC configuration
	OIDC OIDCConfig `yaml:"oidc,omitempty"`
}

// ShutdownConfig holds graceful shutdown configuration.
type ShutdownConfig struct {
	// Timeout is the maximum time to wait for graceful shutdown.
	// After this duration, the server will force shutdown.
	Timeout time.Duration `yaml:"timeout,omitempty"`

	// DrainTimeout is the time to wait for existing connections to drain
	// before starting backup checkpointing.
	DrainTimeout time.Duration `yaml:"drain_timeout,omitempty"`

	// CheckpointRunningBackups determines whether to checkpoint running backups
	// during shutdown so they can be resumed on restart.
	CheckpointRunningBackups bool `yaml:"checkpoint_running_backups,omitempty"`

	// ResumeCheckpointsOnStart determines whether to automatically resume
	// checkpointed backups when the server starts.
	ResumeCheckpointsOnStart bool `yaml:"resume_checkpoints_on_start,omitempty"`
}

// OIDCConfig holds OIDC authentication configuration.
type OIDCConfig struct {
	// Issuer is the OIDC issuer URL.
	Issuer string `yaml:"issuer,omitempty"`

	// ClientID is the OIDC client ID.
	ClientID string `yaml:"client_id,omitempty"`

	// ClientSecret is the OIDC client secret.
	ClientSecret string `yaml:"client_secret,omitempty"`

	// RedirectURL is the callback URL after OIDC authentication.
	RedirectURL string `yaml:"redirect_url,omitempty"`
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		HTTPAddr:    ":8080",
		DatabaseURL: "postgres://localhost/keldris?sslmode=disable",
		Shutdown:    DefaultShutdownConfig(),
	}
}

// DefaultShutdownConfig returns a ShutdownConfig with sensible defaults.
func DefaultShutdownConfig() ShutdownConfig {
	return ShutdownConfig{
		Timeout:                  30 * time.Second,
		DrainTimeout:             5 * time.Second,
		CheckpointRunningBackups: true,
		ResumeCheckpointsOnStart: true,
	}
}

// Validate checks that the server configuration is valid.
func (c *ServerConfig) Validate() error {
	if c.HTTPAddr == "" {
		return errors.New("http_addr is required")
	}
	if c.DatabaseURL == "" {
		return errors.New("database_url is required")
	}
	if err := c.Shutdown.Validate(); err != nil {
		return fmt.Errorf("shutdown config: %w", err)
	}
	return nil
}

// Validate checks that the shutdown configuration is valid.
func (c *ShutdownConfig) Validate() error {
	if c.Timeout < 0 {
		return errors.New("shutdown timeout cannot be negative")
	}
	if c.DrainTimeout < 0 {
		return errors.New("drain timeout cannot be negative")
	}
	if c.DrainTimeout > c.Timeout {
		return errors.New("drain timeout cannot exceed shutdown timeout")
	}
	return nil
}

// LoadServerConfig reads the server configuration from the given path.
// If the file does not exist, default config is returned.
func LoadServerConfig(path string) (*ServerConfig, error) {
	cfg := DefaultServerConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	return &cfg, nil
}

// LoadServerConfigFromEnv loads server configuration from environment variables.
// Environment variables take precedence over config file values.
func LoadServerConfigFromEnv(cfg *ServerConfig) {
	if addr := os.Getenv("KELDRIS_HTTP_ADDR"); addr != "" {
		cfg.HTTPAddr = addr
	}
	if dbURL := os.Getenv("KELDRIS_DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}
	if timeout := os.Getenv("KELDRIS_SHUTDOWN_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.Shutdown.Timeout = d
		}
	}
	if drainTimeout := os.Getenv("KELDRIS_DRAIN_TIMEOUT"); drainTimeout != "" {
		if d, err := time.ParseDuration(drainTimeout); err == nil {
			cfg.Shutdown.DrainTimeout = d
		}
	}
	// OIDC from environment
	if issuer := os.Getenv("KELDRIS_OIDC_ISSUER"); issuer != "" {
		cfg.OIDC.Issuer = issuer
	}
	if clientID := os.Getenv("KELDRIS_OIDC_CLIENT_ID"); clientID != "" {
		cfg.OIDC.ClientID = clientID
	}
	if clientSecret := os.Getenv("KELDRIS_OIDC_CLIENT_SECRET"); clientSecret != "" {
		cfg.OIDC.ClientSecret = clientSecret
	}
	if redirectURL := os.Getenv("KELDRIS_OIDC_REDIRECT_URL"); redirectURL != "" {
		cfg.OIDC.RedirectURL = redirectURL
	}
}

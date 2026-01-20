// Package config provides configuration management for the Keldris agent.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultConfigDir returns the default config directory (~/.keldris).
func DefaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".keldris"), nil
}

// DefaultConfigPath returns the default config file path (~/.keldris/config.yml).
func DefaultConfigPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yml"), nil
}

// AgentConfig holds the agent's configuration.
type AgentConfig struct {
	ServerURL       string `yaml:"server_url,omitempty"`
	APIKey          string `yaml:"api_key,omitempty"`
	AgentID         string `yaml:"agent_id,omitempty"`
	Hostname        string `yaml:"hostname,omitempty"`
	AutoCheckUpdate bool   `yaml:"auto_check_update,omitempty"`
}

// Validate checks that the configuration has required fields for operation.
func (c *AgentConfig) Validate() error {
	if c.ServerURL == "" {
		return errors.New("server_url is required")
	}
	if c.APIKey == "" {
		return errors.New("api_key is required")
	}
	return nil
}

// IsConfigured returns true if the agent has been registered with a server.
func (c *AgentConfig) IsConfigured() bool {
	return c.ServerURL != "" && c.APIKey != ""
}

// Load reads the configuration from the given path.
// If the file does not exist, an empty config is returned.
func Load(path string) (*AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AgentConfig{}, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg AgentConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	return &cfg, nil
}

// LoadDefault loads the configuration from the default path.
func LoadDefault() (*AgentConfig, error) {
	path, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return Load(path)
}

// Save writes the configuration to the given path, creating directories as needed.
func (c *AgentConfig) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Write with restricted permissions (user-only read/write)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// SaveDefault saves the configuration to the default path.
func (c *AgentConfig) SaveDefault() error {
	path, err := DefaultConfigPath()
	if err != nil {
		return err
	}
	return c.Save(path)
}

// Package config provides configuration management for Keldris.
package config

import "os"

// Environment represents the deployment environment.
type Environment string

const (
	// EnvDevelopment is the default local development environment.
	EnvDevelopment Environment = "development"
	// EnvStaging is the staging/pre-production environment.
	EnvStaging Environment = "staging"
	// EnvProduction is the production environment.
	EnvProduction Environment = "production"
)

// ServerConfig holds server-level configuration loaded from environment variables.
type ServerConfig struct {
	Environment Environment
}

// LoadServerConfig reads server configuration from environment variables.
func LoadServerConfig() ServerConfig {
	env := Environment(os.Getenv("ENV"))
	switch env {
	case EnvDevelopment, EnvStaging, EnvProduction:
		// valid
	default:
		env = EnvDevelopment
	}

	return ServerConfig{
		Environment: env,
	}
}

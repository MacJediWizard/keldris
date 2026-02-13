// Package config provides configuration management for Keldris.
package config

import (
	"os"
	"strconv"
	"strings"
)

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
	Environment        Environment
	SessionMaxAge      int  // session lifetime in seconds (default: 86400)
	SessionIdleTimeout int  // idle timeout in seconds, 0 to disable (default: 1800)
	AirGapMode         bool // air-gapped deployment mode (no internet access)
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

	sessionMaxAge := getEnvInt("SESSION_MAX_AGE", 86400)
	if sessionMaxAge < 0 {
		sessionMaxAge = 86400
	}

	sessionIdleTimeout := getEnvInt("SESSION_IDLE_TIMEOUT", 1800)
	if sessionIdleTimeout < 0 {
		sessionIdleTimeout = 1800
	}

	airGapMode := getEnvBool("AIR_GAP_MODE", false)

	return ServerConfig{
		Environment:        env,
		SessionMaxAge:      sessionMaxAge,
		SessionIdleTimeout: sessionIdleTimeout,
		AirGapMode:         airGapMode,
	}
}

// getEnvBool reads a boolean from an environment variable, returning the default if unset or invalid.
func getEnvBool(key string, defaultVal bool) bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch val {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return defaultVal
	}
}

// getEnvInt reads an integer from an environment variable, returning the default if unset or invalid.
func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}

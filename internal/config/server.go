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

// DefaultLicenseServerURL is the production license server endpoint.
const DefaultLicenseServerURL = "https://license.macjediwizard.com"

// DefaultLicensePublicKey is the hex-encoded Ed25519 public key used to verify
// license signatures and entitlement tokens. This is a public key (not secret).
const DefaultLicensePublicKey = "a1d5554e0a5b15e1ec1172d1dab79bc7e9d4cc8502da4f089f1ce1bd0a19e59f"

// ServerConfig holds server-level configuration loaded from environment variables.
type ServerConfig struct {
	Environment        Environment
	SessionMaxAge      int  // session lifetime in seconds (default: 86400)
	SessionIdleTimeout int  // idle timeout in seconds, 0 to disable (default: 1800)
	AirGapMode         bool // air-gapped deployment mode (no internet access)
	RetentionDays      int  // health history retention in days (default: 90)

	LicenseKey       string // Ed25519-signed license key (base64 payload.signature)
	LicensePublicKey string // hex-encoded Ed25519 public key for license verification
	LicenseServerURL string // license server URL for phone-home (default: production)
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

	retentionDays := getEnvInt("RETENTION_DAYS", 90)
	if retentionDays < 1 {
		retentionDays = 90
	}

	licenseServerURL := os.Getenv("LICENSE_SERVER_URL")
	if licenseServerURL == "" {
		licenseServerURL = DefaultLicenseServerURL
	}

	return ServerConfig{
		Environment:        env,
		SessionMaxAge:      sessionMaxAge,
		SessionIdleTimeout: sessionIdleTimeout,
		AirGapMode:         airGapMode,
		RetentionDays:      retentionDays,
		LicenseKey:       os.Getenv("LICENSE_KEY"),
		LicensePublicKey: getEnvDefault("AIRGAP_PUBLIC_KEY", DefaultLicensePublicKey),
		LicenseServerURL:   licenseServerURL,
	}
}

// getEnvDefault reads a string from an environment variable, returning the default if unset or empty.
func getEnvDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
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

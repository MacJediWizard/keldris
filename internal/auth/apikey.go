package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

const (
	// APIKeyPrefix is the prefix for all Keldris API keys.
	APIKeyPrefix = "kld_"
	// APIKeyLength is the expected length of the hex portion of the API key.
	APIKeyLength = 64 // 32 bytes = 64 hex chars
)

// AgentStore defines the interface for agent lookup operations.
type AgentStore interface {
	GetAgentByAPIKeyHash(ctx context.Context, hash string) (*models.Agent, error)
}

// APIKeyValidator validates API keys and retrieves associated agents.
type APIKeyValidator struct {
	store  AgentStore
	logger zerolog.Logger
}

// NewAPIKeyValidator creates a new API key validator.
func NewAPIKeyValidator(store AgentStore, logger zerolog.Logger) *APIKeyValidator {
	return &APIKeyValidator{
		store:  store,
		logger: logger.With().Str("component", "apikey_validator").Logger(),
	}
}

// ValidateAPIKey validates an API key and returns the associated agent.
// Returns nil if the key is invalid or not found.
func (v *APIKeyValidator) ValidateAPIKey(ctx context.Context, apiKey string) (*models.Agent, error) {
	// Validate key format
	if !IsValidAPIKeyFormat(apiKey) {
		v.logger.Debug().Msg("invalid API key format")
		return nil, nil
	}

	// Hash the key for lookup
	keyHash := HashAPIKey(apiKey)

	// Look up agent by hash
	agent, err := v.store.GetAgentByAPIKeyHash(ctx, keyHash)
	if err != nil {
		v.logger.Debug().Err(err).Msg("agent not found for API key")
		return nil, nil
	}

	// Check agent status
	if agent.Status == models.AgentStatusDisabled {
		v.logger.Debug().Str("agent_id", agent.ID.String()).Msg("agent is disabled")
		return nil, nil
	}

	v.logger.Debug().
		Str("agent_id", agent.ID.String()).
		Str("hostname", agent.Hostname).
		Msg("API key validated")

	return agent, nil
}

// IsValidAPIKeyFormat checks if the API key has the correct format.
func IsValidAPIKeyFormat(apiKey string) bool {
	if !strings.HasPrefix(apiKey, APIKeyPrefix) {
		return false
	}
	hexPart := strings.TrimPrefix(apiKey, APIKeyPrefix)
	if len(hexPart) != APIKeyLength {
		return false
	}
	// Verify it's valid hex
	_, err := hex.DecodeString(hexPart)
	return err == nil
}

// HashAPIKey creates a SHA-256 hash of an API key for storage/comparison.
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// CompareAPIKeyHash compares an API key with a stored hash using constant-time comparison.
func CompareAPIKeyHash(apiKey, storedHash string) bool {
	computedHash := HashAPIKey(apiKey)
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}

// ExtractBearerToken extracts the token from an Authorization header value.
// Returns empty string if the header is not a valid Bearer token.
func ExtractBearerToken(authHeader string) string {
	const prefix = "Bearer "
	if len(authHeader) < len(prefix) {
		return ""
	}
	if !strings.EqualFold(authHeader[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(authHeader[len(prefix):])
}

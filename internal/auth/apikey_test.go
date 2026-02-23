package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockAgentStore implements AgentStore for testing.
type mockAgentStore struct {
	agents map[string]*models.Agent // key: api_key_hash
	err    error
}

func newMockAgentStore() *mockAgentStore {
	return &mockAgentStore{
		agents: make(map[string]*models.Agent),
	}
}

func (m *mockAgentStore) addAgent(apiKey string, agent *models.Agent) {
	hash := HashAPIKey(apiKey)
	agent.APIKeyHash = hash
	m.agents[hash] = agent
}

func (m *mockAgentStore) GetAgentByAPIKeyHash(_ context.Context, hash string) (*models.Agent, error) {
	if m.err != nil {
		return nil, m.err
	}
	agent, ok := m.agents[hash]
	if !ok {
		return nil, fmt.Errorf("agent not found")
	}
	return agent, nil
}

func TestIsValidAPIKeyFormat(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{
			name:     "valid API key",
			apiKey:   "kld_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: true,
		},
		{
			name:     "missing prefix",
			apiKey:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: false,
		},
		{
			name:     "wrong prefix",
			apiKey:   "api_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: false,
		},
		{
			name:     "too short",
			apiKey:   "kld_0123456789abcdef",
			expected: false,
		},
		{
			name:     "too long",
			apiKey:   "kld_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef00",
			expected: false,
		},
		{
			name:     "invalid hex characters",
			apiKey:   "kld_0123456789abcdef0123456789abcdef0123456789abcdef0123456789ghijkl",
			expected: false,
		},
		{
			name:     "empty string",
			apiKey:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidAPIKeyFormat(tt.apiKey)
			if result != tt.expected {
				t.Errorf("IsValidAPIKeyFormat(%q) = %v, want %v", tt.apiKey, result, tt.expected)
			}
		})
	}
}

func TestHashAPIKey(t *testing.T) {
	apiKey := "kld_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	hash := HashAPIKey(apiKey)

	// Hash should be 64 characters (SHA-256 = 32 bytes = 64 hex chars)
	if len(hash) != 64 {
		t.Errorf("HashAPIKey() returned hash of length %d, want 64", len(hash))
	}

	// Same key should produce same hash
	hash2 := HashAPIKey(apiKey)
	if hash != hash2 {
		t.Errorf("HashAPIKey() returned different hashes for same input")
	}

	// Different key should produce different hash
	differentKey := "kld_fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	hash3 := HashAPIKey(differentKey)
	if hash == hash3 {
		t.Errorf("HashAPIKey() returned same hash for different inputs")
	}
}

func TestCompareAPIKeyHash(t *testing.T) {
	apiKey := "kld_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hash := HashAPIKey(apiKey)

	tests := []struct {
		name       string
		apiKey     string
		storedHash string
		expected   bool
	}{
		{
			name:       "matching key and hash",
			apiKey:     apiKey,
			storedHash: hash,
			expected:   true,
		},
		{
			name:       "non-matching key and hash",
			apiKey:     "kld_fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
			storedHash: hash,
			expected:   false,
		},
		{
			name:       "empty key",
			apiKey:     "",
			storedHash: hash,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareAPIKeyHash(tt.apiKey, tt.storedHash)
			if result != tt.expected {
				t.Errorf("CompareAPIKeyHash() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		expected   string
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer kld_0123456789abcdef",
			expected:   "kld_0123456789abcdef",
		},
		{
			name:       "bearer token with extra spaces",
			authHeader: "Bearer   kld_0123456789abcdef  ",
			expected:   "kld_0123456789abcdef",
		},
		{
			name:       "lowercase bearer",
			authHeader: "bearer kld_0123456789abcdef",
			expected:   "kld_0123456789abcdef",
		},
		{
			name:       "empty header",
			authHeader: "",
			expected:   "",
		},
		{
			name:       "no bearer prefix",
			authHeader: "kld_0123456789abcdef",
			expected:   "",
		},
		{
			name:       "basic auth instead",
			authHeader: "Basic dXNlcjpwYXNz",
			expected:   "",
		},
		{
			name:       "bearer only",
			authHeader: "Bearer ",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBearerToken(tt.authHeader)
			if result != tt.expected {
				t.Errorf("ExtractBearerToken(%q) = %q, want %q", tt.authHeader, result, tt.expected)
			}
		})
	}
}

func TestAPIKey_Generate(t *testing.T) {
	store := newMockAgentStore()
	logger := zerolog.Nop()
	validator := NewAPIKeyValidator(store, logger)
	if validator == nil {
		t.Fatal("expected non-nil validator")
	}
}

func TestAPIKey_Validate(t *testing.T) {
	validKey := "kld_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	agentID := uuid.New()
	orgID := uuid.New()

	t.Run("valid key returns agent", func(t *testing.T) {
		store := newMockAgentStore()
		store.addAgent(validKey, &models.Agent{
			ID:       agentID,
			OrgID:    orgID,
			Hostname: "test-host",
			Status:   models.AgentStatusActive,
		})
		logger := zerolog.Nop()
		validator := NewAPIKeyValidator(store, logger)

		agent, err := validator.ValidateAPIKey(context.Background(), validKey)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent == nil {
			t.Fatal("expected non-nil agent")
		}
		if agent.ID != agentID {
			t.Errorf("expected agent ID %s, got %s", agentID, agent.ID)
		}
		if agent.Hostname != "test-host" {
			t.Errorf("expected hostname 'test-host', got %q", agent.Hostname)
		}
	})

	t.Run("invalid format returns nil", func(t *testing.T) {
		store := newMockAgentStore()
		logger := zerolog.Nop()
		validator := NewAPIKeyValidator(store, logger)

		agent, err := validator.ValidateAPIKey(context.Background(), "invalid-key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent != nil {
			t.Error("expected nil agent for invalid format")
		}
	})

	t.Run("key not found returns nil", func(t *testing.T) {
		store := newMockAgentStore()
		logger := zerolog.Nop()
		validator := NewAPIKeyValidator(store, logger)

		agent, err := validator.ValidateAPIKey(context.Background(), validKey)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent != nil {
			t.Error("expected nil agent for unknown key")
		}
	})

	t.Run("disabled agent returns nil", func(t *testing.T) {
		store := newMockAgentStore()
		store.addAgent(validKey, &models.Agent{
			ID:       agentID,
			OrgID:    orgID,
			Hostname: "disabled-host",
			Status:   models.AgentStatusDisabled,
		})
		logger := zerolog.Nop()
		validator := NewAPIKeyValidator(store, logger)

		agent, err := validator.ValidateAPIKey(context.Background(), validKey)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent != nil {
			t.Error("expected nil agent for disabled agent")
		}
	})

	t.Run("store error returns nil", func(t *testing.T) {
		store := newMockAgentStore()
		store.err = fmt.Errorf("database connection failed")
		logger := zerolog.Nop()
		validator := NewAPIKeyValidator(store, logger)

		agent, err := validator.ValidateAPIKey(context.Background(), validKey)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent != nil {
			t.Error("expected nil agent for store error")
		}
	})
}

func TestAPIKey_Rotate(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	oldKey := "kld_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	newKey := "kld_fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"

	store := newMockAgentStore()
	logger := zerolog.Nop()

	store.addAgent(oldKey, &models.Agent{
		ID:       agentID,
		OrgID:    orgID,
		Hostname: "test-host",
		Status:   models.AgentStatusActive,
	})

	validator := NewAPIKeyValidator(store, logger)

	// Old key should work
	agent, err := validator.ValidateAPIKey(context.Background(), oldKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent with old key")
	}

	// New key should not work yet
	agent, err = validator.ValidateAPIKey(context.Background(), newKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent != nil {
		t.Error("expected nil agent for new key before rotation")
	}

	// Simulate rotation: remove old key, add new key
	delete(store.agents, HashAPIKey(oldKey))
	store.addAgent(newKey, &models.Agent{
		ID:       agentID,
		OrgID:    orgID,
		Hostname: "test-host",
		Status:   models.AgentStatusActive,
	})

	// Old key should no longer work
	agent, err = validator.ValidateAPIKey(context.Background(), oldKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent != nil {
		t.Error("expected nil agent for old key after rotation")
	}

	// New key should work
	agent, err = validator.ValidateAPIKey(context.Background(), newKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent with new key after rotation")
	}
}

func TestAPIKey_Revoke(t *testing.T) {
	validKey := "kld_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	agentID := uuid.New()
	orgID := uuid.New()

	store := newMockAgentStore()
	store.addAgent(validKey, &models.Agent{
		ID:       agentID,
		OrgID:    orgID,
		Hostname: "test-host",
		Status:   models.AgentStatusActive,
	})
	logger := zerolog.Nop()
	validator := NewAPIKeyValidator(store, logger)

	// Key should work initially
	agent, err := validator.ValidateAPIKey(context.Background(), validKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent before revocation")
	}

	// Simulate revocation: set agent to disabled
	storedAgent := store.agents[HashAPIKey(validKey)]
	storedAgent.Status = models.AgentStatusDisabled

	// Key should no longer work (agent is disabled)
	agent, err = validator.ValidateAPIKey(context.Background(), validKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent != nil {
		t.Error("expected nil agent after revocation")
	}
}

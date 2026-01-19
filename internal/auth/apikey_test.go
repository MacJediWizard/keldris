package auth

import (
	"testing"
)

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

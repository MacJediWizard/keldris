package license

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndValidateOfflineLicense(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	expiry := time.Now().Add(365 * 24 * time.Hour)
	data, err := GenerateOfflineLicense("cust-123", TierEnterprise, expiry, priv)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	lic, err := ValidateOfflineLicense(data, pub)
	require.NoError(t, err)
	assert.Equal(t, TierEnterprise, lic.Tier)
	assert.Equal(t, "cust-123", lic.CustomerID)
	assert.WithinDuration(t, expiry, lic.ExpiresAt, time.Second)
}

func TestGenerateOfflineLicense_InvalidInputs(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	expiry := time.Now().Add(365 * 24 * time.Hour)

	// Empty customer ID
	_, err = GenerateOfflineLicense("", TierEnterprise, expiry, priv)
	assert.Error(t, err)

	// Invalid tier
	_, err = GenerateOfflineLicense("cust-123", LicenseTier("invalid"), expiry, priv)
	assert.Error(t, err)

	// Invalid key
	_, err = GenerateOfflineLicense("cust-123", TierEnterprise, expiry, []byte("short"))
	assert.Error(t, err)
}

func TestValidateOfflineLicense_InvalidSignature(t *testing.T) {
	pub1, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	_, priv2, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	expiry := time.Now().Add(365 * 24 * time.Hour)
	data, err := GenerateOfflineLicense("cust-123", TierEnterprise, expiry, priv2)
	require.NoError(t, err)

	// Wrong public key
	_, err = ValidateOfflineLicense(data, pub1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid license signature")
}

func TestValidateOfflineLicense_Expired(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	expiry := time.Now().Add(-24 * time.Hour) // expired yesterday
	data, err := GenerateOfflineLicense("cust-123", TierEnterprise, expiry, priv)
	require.NoError(t, err)

	_, err = ValidateOfflineLicense(data, pub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestValidateOfflineLicense_EmptyData(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	_, err = ValidateOfflineLicense(nil, pub)
	assert.Error(t, err)

	_, err = ValidateOfflineLicense([]byte{}, pub)
	assert.Error(t, err)
}

func TestValidateOfflineLicense_InvalidPublicKey(t *testing.T) {
	_, err := ValidateOfflineLicense([]byte(`{"payload":"dGVzdA==","signature":"dGVzdA=="}`), []byte("short"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Ed25519 public key")
}

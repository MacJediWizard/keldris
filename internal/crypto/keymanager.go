// Package crypto provides encryption utilities for Keldris.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

const (
	// PasswordLength is the length of generated Restic repository passwords.
	// 32 bytes provides 256 bits of entropy.
	PasswordLength = 32

	// NonceSize is the size of the AES-GCM nonce (12 bytes standard).
	NonceSize = 12

	// KeySize is the size of the AES-256 key (32 bytes).
	KeySize = 32
)

var (
	// ErrInvalidKeySize indicates the encryption key is not the correct size.
	ErrInvalidKeySize = errors.New("encryption key must be 32 bytes")
	// ErrInvalidCiphertext indicates the ciphertext is too short or malformed.
	ErrInvalidCiphertext = errors.New("ciphertext too short")
	// ErrDecryptionFailed indicates the decryption operation failed.
	ErrDecryptionFailed = errors.New("decryption failed")
)

// KeyManager handles encryption key generation and management.
type KeyManager struct {
	// masterKey is the server-side encryption key for encrypting stored keys.
	masterKey []byte
}

// NewKeyManager creates a new KeyManager with the given master key.
// The master key must be exactly 32 bytes (256 bits) for AES-256.
func NewKeyManager(masterKey []byte) (*KeyManager, error) {
	if len(masterKey) != KeySize {
		return nil, ErrInvalidKeySize
	}
	return &KeyManager{masterKey: masterKey}, nil
}

// GeneratePassword generates a cryptographically secure random password
// for use with Restic repositories.
func (km *KeyManager) GeneratePassword() (string, error) {
	bytes := make([]byte, PasswordLength)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	// Use URL-safe base64 encoding for the password
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// Encrypt encrypts plaintext using AES-256-GCM with the master key.
// Returns the ciphertext with the nonce prepended.
func (km *KeyManager) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(km.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends the encrypted data to nonce, so the result is nonce + ciphertext + tag
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext encrypted with Encrypt.
// Expects the nonce to be prepended to the ciphertext.
func (km *KeyManager) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < NonceSize {
		return nil, ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(km.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := ciphertext[:NonceSize]
	encryptedData := ciphertext[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext.
func (km *KeyManager) EncryptString(plaintext string) (string, error) {
	ciphertext, err := km.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts base64-encoded ciphertext and returns the plaintext string.
func (km *KeyManager) DecryptString(encodedCiphertext string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	plaintext, err := km.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// GenerateMasterKey generates a new random master key for use with NewKeyManager.
// This should be done once during initial server setup and stored securely.
func GenerateMasterKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate master key: %w", err)
	}
	return key, nil
}

// masterKeyToBase64 encodes a master key to base64 for configuration storage.
func masterKeyToBase64(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// masterKeyFromBase64 decodes a base64-encoded master key.
func masterKeyFromBase64(encoded string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	return key, nil
}

// MasterKeyToBase64 encodes a master key to base64 for configuration storage.
func MasterKeyToBase64(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// MasterKeyFromBase64 decodes a base64-encoded master key.
func MasterKeyFromBase64(encoded string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	return key, nil
}

// MasterKeyFromHex decodes a hex-encoded master key.
func MasterKeyFromHex(encoded string) ([]byte, error) {
	key, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	return key, nil
}

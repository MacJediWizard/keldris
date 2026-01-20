package crypto

import (
	"bytes"
	"testing"
)

func TestGenerateMasterKey(t *testing.T) {
	key, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey() error = %v", err)
	}
	if len(key) != KeySize {
		t.Errorf("GenerateMasterKey() key length = %d, want %d", len(key), KeySize)
	}

	// Generate another key and verify they're different
	key2, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey() error = %v", err)
	}
	if bytes.Equal(key, key2) {
		t.Error("GenerateMasterKey() generated identical keys")
	}
}

func TestNewKeyManager(t *testing.T) {
	tests := []struct {
		name    string
		keyLen  int
		wantErr bool
	}{
		{"valid key", 32, false},
		{"short key", 16, true},
		{"long key", 64, true},
		{"empty key", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := NewKeyManager(key)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKeyManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKeyManager_GeneratePassword(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	password, err := km.GeneratePassword()
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}

	// Password should be base64 encoded 32 bytes
	if len(password) == 0 {
		t.Error("GeneratePassword() returned empty password")
	}

	// Generate another and verify they're different
	password2, err := km.GeneratePassword()
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}
	if password == password2 {
		t.Error("GeneratePassword() generated identical passwords")
	}
}

func TestKeyManager_EncryptDecrypt(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	plaintext := []byte("test-repository-password-12345")

	ciphertext, err := km.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if bytes.Equal(plaintext, ciphertext) {
		t.Error("Encrypt() ciphertext equals plaintext")
	}

	decrypted, err := km.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypt() = %s, want %s", decrypted, plaintext)
	}
}

func TestKeyManager_EncryptDecryptString(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	plaintext := "my-secret-password"

	encrypted, err := km.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("EncryptString() error = %v", err)
	}

	decrypted, err := km.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("DecryptString() error = %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("DecryptString() = %s, want %s", decrypted, plaintext)
	}
}

func TestKeyManager_DecryptInvalidCiphertext(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	// Too short
	_, err := km.Decrypt([]byte("short"))
	if err != ErrInvalidCiphertext {
		t.Errorf("Decrypt() error = %v, want %v", err, ErrInvalidCiphertext)
	}

	// Invalid ciphertext (wrong key or tampered)
	wrongKey, _ := GenerateMasterKey()
	km2, _ := NewKeyManager(wrongKey)

	plaintext := []byte("test")
	ciphertext, _ := km.Encrypt(plaintext)

	_, err = km2.Decrypt(ciphertext)
	if err != ErrDecryptionFailed {
		t.Errorf("Decrypt() with wrong key error = %v, want %v", err, ErrDecryptionFailed)
	}
}

func TestMasterKeyBase64(t *testing.T) {
	key, _ := GenerateMasterKey()

	encoded := MasterKeyToBase64(key)
	decoded, err := MasterKeyFromBase64(encoded)
	if err != nil {
		t.Fatalf("MasterKeyFromBase64() error = %v", err)
	}

	if !bytes.Equal(key, decoded) {
		t.Errorf("MasterKeyFromBase64() = %v, want %v", decoded, key)
	}
}

func TestMasterKeyFromBase64_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
	}{
		{"invalid base64", "not-valid-base64!!!"},
		{"wrong length", "dG9vLXNob3J0"}, // "too-short" in base64
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MasterKeyFromBase64(tt.encoded)
			if err == nil {
				t.Error("MasterKeyFromBase64() expected error, got nil")
			}
		})
	}
}

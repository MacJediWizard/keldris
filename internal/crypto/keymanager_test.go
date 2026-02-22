package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
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

func TestNewKeyManager_ErrorType(t *testing.T) {
	_, err := NewKeyManager([]byte("short"))
	if err != ErrInvalidKeySize {
		t.Errorf("NewKeyManager() error = %v, want %v", err, ErrInvalidKeySize)
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

	// Verify it's valid base64
	decoded, err := base64.URLEncoding.DecodeString(password)
	if err != nil {
		t.Fatalf("GeneratePassword() produced invalid base64: %v", err)
	}
	if len(decoded) != PasswordLength {
		t.Errorf("GeneratePassword() decoded length = %d, want %d", len(decoded), PasswordLength)
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

func TestKeyManager_GeneratePassword_Uniqueness(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		password, err := km.GeneratePassword()
		if err != nil {
			t.Fatalf("GeneratePassword() iteration %d error = %v", i, err)
		}
		if seen[password] {
			t.Fatalf("GeneratePassword() duplicate password at iteration %d", i)
		}
		seen[password] = true
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

func TestKeyManager_EncryptDecrypt_AllSizes(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	sizes := []int{1, 2, 15, 16, 17, 31, 32, 33, 63, 64, 65, 128, 255, 256, 512, 1024}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("%d_bytes", size), func(t *testing.T) {
			plaintext := make([]byte, size)
			if _, err := rand.Read(plaintext); err != nil {
				t.Fatalf("failed to generate random plaintext: %v", err)
			}

			ciphertext, err := km.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Encrypt(%d bytes) error = %v", size, err)
			}

			// Ciphertext must be larger than plaintext (nonce + tag overhead)
			if len(ciphertext) <= len(plaintext) {
				t.Errorf("Encrypt(%d bytes) ciphertext length %d not larger than plaintext", size, len(ciphertext))
			}

			decrypted, err := km.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt(%d bytes) error = %v", size, err)
			}

			if !bytes.Equal(plaintext, decrypted) {
				t.Errorf("Decrypt(%d bytes) round-trip mismatch", size)
			}
		})
	}
}

func TestKeyManager_EncryptDecrypt_EmptyData(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	plaintext := []byte{}

	ciphertext, err := km.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt(empty) error = %v", err)
	}

	// Even empty plaintext should produce ciphertext (nonce + GCM tag)
	if len(ciphertext) == 0 {
		t.Error("Encrypt(empty) produced empty ciphertext")
	}

	decrypted, err := km.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt(empty) error = %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("Decrypt(empty) = %v, want empty", decrypted)
	}
}

func TestKeyManager_EncryptDecrypt_LargeData(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	// 1MB of random data
	plaintext := make([]byte, 1<<20)
	if _, err := rand.Read(plaintext); err != nil {
		t.Fatalf("failed to generate random plaintext: %v", err)
	}

	ciphertext, err := km.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt(1MB) error = %v", err)
	}

	decrypted, err := km.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt(1MB) error = %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("Decrypt(1MB) round-trip mismatch")
	}
}

func TestKeyManager_Encrypt_ProducesUniqueCiphertexts(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	plaintext := []byte("same-plaintext-every-time")

	ct1, err := km.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	ct2, err := km.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Different nonces should produce different ciphertexts
	if bytes.Equal(ct1, ct2) {
		t.Error("Encrypt() produced identical ciphertexts for same plaintext")
	}

	// Both should still decrypt correctly
	d1, _ := km.Decrypt(ct1)
	d2, _ := km.Decrypt(ct2)
	if !bytes.Equal(d1, plaintext) || !bytes.Equal(d2, plaintext) {
		t.Error("different ciphertexts did not both decrypt to original plaintext")
	}
}

func TestKeyManager_InvalidKey(t *testing.T) {
	tests := []struct {
		name   string
		keyLen int
	}{
		{"1 byte", 1},
		{"15 bytes", 15},
		{"16 bytes (AES-128)", 16},
		{"24 bytes (AES-192)", 24},
		{"31 bytes", 31},
		{"33 bytes", 33},
		{"48 bytes", 48},
		{"64 bytes", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := NewKeyManager(key)
			if err != ErrInvalidKeySize {
				t.Errorf("NewKeyManager(%d bytes) error = %v, want %v", tt.keyLen, err, ErrInvalidKeySize)
			}
		})
	}
}

func TestKeyManager_TamperedCiphertext(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	plaintext := []byte("sensitive-data-that-must-not-be-altered")
	ciphertext, err := km.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	tests := []struct {
		name   string
		tamper func([]byte) []byte
	}{
		{
			"flip bit in nonce",
			func(ct []byte) []byte {
				tampered := make([]byte, len(ct))
				copy(tampered, ct)
				tampered[0] ^= 0x01
				return tampered
			},
		},
		{
			"flip bit in ciphertext body",
			func(ct []byte) []byte {
				tampered := make([]byte, len(ct))
				copy(tampered, ct)
				tampered[NonceSize+1] ^= 0x01
				return tampered
			},
		},
		{
			"flip bit in GCM tag",
			func(ct []byte) []byte {
				tampered := make([]byte, len(ct))
				copy(tampered, ct)
				tampered[len(tampered)-1] ^= 0x01
				return tampered
			},
		},
		{
			"truncate last byte",
			func(ct []byte) []byte {
				return ct[:len(ct)-1]
			},
		},
		{
			"append extra byte",
			func(ct []byte) []byte {
				return append(ct, 0xFF)
			},
		},
		{
			"zero out ciphertext body",
			func(ct []byte) []byte {
				tampered := make([]byte, len(ct))
				copy(tampered, ct)
				for i := NonceSize; i < len(tampered); i++ {
					tampered[i] = 0
				}
				return tampered
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tampered := tt.tamper(ciphertext)
			_, err := km.Decrypt(tampered)
			if err != ErrDecryptionFailed {
				t.Errorf("Decrypt(tampered: %s) error = %v, want %v", tt.name, err, ErrDecryptionFailed)
			}
		})
	}
}

func TestKeyManager_InvalidNonce(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	tests := []struct {
		name      string
		data      []byte
		wantErr   error
	}{
		{"empty", []byte{}, ErrInvalidCiphertext},
		{"1 byte", []byte{0x01}, ErrInvalidCiphertext},
		{"11 bytes", make([]byte, NonceSize-1), ErrInvalidCiphertext},
		{"exactly nonce size", make([]byte, NonceSize), ErrDecryptionFailed},
		{"nonce plus 1 byte garbage", make([]byte, NonceSize+1), ErrDecryptionFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := km.Decrypt(tt.data)
			if err != tt.wantErr {
				t.Errorf("Decrypt(%s) error = %v, want %v", tt.name, err, tt.wantErr)
			}
		})
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

func TestKeyManager_EncryptDecryptString_Various(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty string", ""},
		{"single char", "a"},
		{"unicode", "héllo wörld 日本語 🔑"},
		{"special chars", "p@$$w0rd!#%^&*()"},
		{"newlines", "line1\nline2\nline3"},
		{"long string", strings.Repeat("abcdefghij", 1000)},
		{"null bytes", "before\x00after"},
		{"json", `{"key": "value", "nested": {"a": 1}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := km.EncryptString(tt.plaintext)
			if err != nil {
				t.Fatalf("EncryptString(%s) error = %v", tt.name, err)
			}

			// Encrypted string must be valid base64
			_, err = base64.StdEncoding.DecodeString(encrypted)
			if err != nil {
				t.Fatalf("EncryptString(%s) produced invalid base64: %v", tt.name, err)
			}

			decrypted, err := km.DecryptString(encrypted)
			if err != nil {
				t.Fatalf("DecryptString(%s) error = %v", tt.name, err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("DecryptString(%s) = %q, want %q", tt.name, decrypted, tt.plaintext)
			}
		})
	}
}

func TestKeyManager_DecryptString_InvalidBase64(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	tests := []struct {
		name  string
		input string
	}{
		{"not base64", "!!!not-valid-base64!!!"},
		{"partial base64", "aGVsbG8=extragarbage!!!"},
		{"wrong padding", "aGVsbG8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := km.DecryptString(tt.input)
			if err == nil {
				t.Errorf("DecryptString(%s) expected error, got nil", tt.name)
			}
		})
	}
}

func TestKeyManager_DecryptString_ValidBase64_InvalidCiphertext(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	// Valid base64 but too short to be valid ciphertext
	shortData := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err := km.DecryptString(shortData)
	if err != ErrInvalidCiphertext {
		t.Errorf("DecryptString(short ciphertext) error = %v, want %v", err, ErrInvalidCiphertext)
	}

	// Valid base64, long enough, but garbage data
	garbage := make([]byte, NonceSize+32)
	if _, err := rand.Read(garbage); err != nil {
		t.Fatalf("failed to generate random garbage: %v", err)
	}
	garbageB64 := base64.StdEncoding.EncodeToString(garbage)
	_, err = km.DecryptString(garbageB64)
	if err != ErrDecryptionFailed {
		t.Errorf("DecryptString(garbage) error = %v, want %v", err, ErrDecryptionFailed)
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

func TestKeyManager_GenerateKey(t *testing.T) {
	// Test GenerateMasterKey produces valid keys usable with NewKeyManager
	for i := 0; i < 10; i++ {
		key, err := GenerateMasterKey()
		if err != nil {
			t.Fatalf("GenerateMasterKey() iteration %d error = %v", i, err)
		}
		if len(key) != KeySize {
			t.Errorf("GenerateMasterKey() iteration %d length = %d, want %d", i, len(key), KeySize)
		}

		km, err := NewKeyManager(key)
		if err != nil {
			t.Fatalf("NewKeyManager() iteration %d error = %v", i, err)
		}

		// Verify the key actually works for encryption
		ct, err := km.Encrypt([]byte("test"))
		if err != nil {
			t.Fatalf("Encrypt() iteration %d error = %v", i, err)
		}
		pt, err := km.Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt() iteration %d error = %v", i, err)
		}
		if string(pt) != "test" {
			t.Errorf("round-trip iteration %d = %q, want %q", i, pt, "test")
		}
	}
}

func TestKeyManager_StoreKey(t *testing.T) {
	// Test that master keys survive base64 encode/decode round-trip (storage simulation)
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	// Encrypt something with the original key
	plaintext := []byte("data-to-protect")
	ciphertext, err := km.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// "Store" the key as base64 (simulating config storage)
	stored := masterKeyToBase64(key)

	// "Retrieve" the key from storage
	retrieved, err := masterKeyFromBase64(stored)
	if err != nil {
		t.Fatalf("masterKeyFromBase64() error = %v", err)
	stored := MasterKeyToBase64(key)

	// "Retrieve" the key from storage
	retrieved, err := MasterKeyFromBase64(stored)
	if err != nil {
		t.Fatalf("MasterKeyFromBase64() error = %v", err)
	}

	// Create a new KeyManager with the retrieved key
	km2, err := NewKeyManager(retrieved)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	// Verify it can decrypt data encrypted with the original key
	decrypted, err := km2.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() with restored key error = %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypt() with restored key = %s, want %s", decrypted, plaintext)
	}
}

func TestKeyManager_RetrieveKey(t *testing.T) {
	// Test base64 round-trip preserves key identity
	key, _ := GenerateMasterKey()

	encoded := masterKeyToBase64(key)
	encoded := MasterKeyToBase64(key)

	// Verify encoded form is valid base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("masterKeyToBase64() produced invalid base64: %v", err)
		t.Fatalf("MasterKeyToBase64() produced invalid base64: %v", err)
	}
	if !bytes.Equal(key, decoded) {
		t.Error("raw base64 decode doesn't match original key")
	}

	// Verify masterKeyFromBase64 gives identical result
	retrieved, err := masterKeyFromBase64(encoded)
	if err != nil {
		t.Fatalf("masterKeyFromBase64() error = %v", err)
	}
	if !bytes.Equal(key, retrieved) {
		t.Error("masterKeyFromBase64() result doesn't match original key")
	// Verify MasterKeyFromBase64 gives identical result
	retrieved, err := MasterKeyFromBase64(encoded)
	if err != nil {
		t.Fatalf("MasterKeyFromBase64() error = %v", err)
	}
	if !bytes.Equal(key, retrieved) {
		t.Error("MasterKeyFromBase64() result doesn't match original key")
	}
}

func TestKeyManager_RotateKey(t *testing.T) {
	// Simulate key rotation: encrypt with old key, re-encrypt with new key
	oldKey, _ := GenerateMasterKey()
	newKey, _ := GenerateMasterKey()

	oldKM, _ := NewKeyManager(oldKey)
	newKM, _ := NewKeyManager(newKey)

	// Encrypt data with old key
	secrets := []string{
		"repo-password-1",
		"repo-password-2",
		"api-key-abc123",
	}

	var encryptedSecrets [][]byte
	for _, s := range secrets {
		ct, err := oldKM.Encrypt([]byte(s))
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}
		encryptedSecrets = append(encryptedSecrets, ct)
	}

	// Rotate: decrypt with old key, re-encrypt with new key
	var reEncryptedSecrets [][]byte
	for _, ct := range encryptedSecrets {
		pt, err := oldKM.Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt() with old key error = %v", err)
		}
		newCT, err := newKM.Encrypt(pt)
		if err != nil {
			t.Fatalf("Encrypt() with new key error = %v", err)
		}
		reEncryptedSecrets = append(reEncryptedSecrets, newCT)
	}

	// Verify old key can't decrypt re-encrypted data
	for i, ct := range reEncryptedSecrets {
		_, err := oldKM.Decrypt(ct)
		if err != ErrDecryptionFailed {
			t.Errorf("old key decrypted re-encrypted secret %d, expected failure", i)
		}
	}

	// Verify new key can decrypt re-encrypted data
	for i, ct := range reEncryptedSecrets {
		pt, err := newKM.Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt() with new key secret %d error = %v", i, err)
		}
		if string(pt) != secrets[i] {
			t.Errorf("Decrypt() with new key secret %d = %q, want %q", i, pt, secrets[i])
		}
	}
}

func TestKeyManager_EscrowKey(t *testing.T) {
	// Simulate escrow: encrypt a repo key with the master key, then encrypt a copy with an escrow key
	masterKey, _ := GenerateMasterKey()
	escrowKey, _ := GenerateMasterKey()

	masterKM, _ := NewKeyManager(masterKey)
	escrowKM, _ := NewKeyManager(escrowKey)

	// The repo password to protect
	repoPassword := []byte("super-secret-restic-repo-password-with-high-entropy")

	// Encrypt with master key (primary storage)
	primaryCT, err := masterKM.Encrypt(repoPassword)
	if err != nil {
		t.Fatalf("Encrypt() with master key error = %v", err)
	}

	// Encrypt with escrow key (escrow copy)
	escrowCT, err := escrowKM.Encrypt(repoPassword)
	if err != nil {
		t.Fatalf("Encrypt() with escrow key error = %v", err)
	}

	// Primary and escrow ciphertexts should be different
	if bytes.Equal(primaryCT, escrowCT) {
		t.Error("primary and escrow ciphertexts are identical")
	}

	// Master key decrypts primary but not escrow
	pt, err := masterKM.Decrypt(primaryCT)
	if err != nil {
		t.Fatalf("Decrypt(primary) with master key error = %v", err)
	}
	if !bytes.Equal(pt, repoPassword) {
		t.Error("master key failed to decrypt primary ciphertext")
	}

	_, err = masterKM.Decrypt(escrowCT)
	if err != ErrDecryptionFailed {
		t.Errorf("master key should not decrypt escrow ciphertext, error = %v", err)
	}

	// Escrow key decrypts escrow but not primary
	pt, err = escrowKM.Decrypt(escrowCT)
	if err != nil {
		t.Fatalf("Decrypt(escrow) with escrow key error = %v", err)
	}
	if !bytes.Equal(pt, repoPassword) {
		t.Error("escrow key failed to decrypt escrow ciphertext")
	}

	_, err = escrowKM.Decrypt(primaryCT)
	if err != ErrDecryptionFailed {
		t.Errorf("escrow key should not decrypt primary ciphertext, error = %v", err)
	}
}

func TestMasterKeyBase64(t *testing.T) {
	key, _ := GenerateMasterKey()

	encoded := masterKeyToBase64(key)
	decoded, err := masterKeyFromBase64(encoded)
	if err != nil {
		t.Fatalf("masterKeyFromBase64() error = %v", err)
	}

	if !bytes.Equal(key, decoded) {
		t.Errorf("masterKeyFromBase64() = %v, want %v", decoded, key)
	}
}

func TestMasterKeyFromBase64_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
	}{
		{"invalid base64", "not-valid-base64!!!"},
		{"wrong length", "dG9vLXNob3J0"},                                                // "too-short" in base64
		{"empty string", ""},                                                              // empty
		{"16 bytes key", base64.StdEncoding.EncodeToString(make([]byte, 16))},             // AES-128 key
		{"64 bytes key", base64.StdEncoding.EncodeToString(make([]byte, 64))},             // too long
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := masterKeyFromBase64(tt.encoded)
			if err == nil {
				t.Error("masterKeyFromBase64() expected error, got nil")
			}
		})
	}
}

func TestKeyManager_CiphertextStructure(t *testing.T) {
	key, _ := GenerateMasterKey()
	km, _ := NewKeyManager(key)

	plaintext := []byte("verify-ciphertext-structure")
	ciphertext, err := km.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Ciphertext = nonce (12) + encrypted data + GCM tag (16)
	expectedMinLen := NonceSize + len(plaintext) + 16 // GCM tag is 16 bytes
	if len(ciphertext) != expectedMinLen {
		t.Errorf("ciphertext length = %d, want %d (nonce=%d + plaintext=%d + tag=16)",
			len(ciphertext), expectedMinLen, NonceSize, len(plaintext))
	}
}

func TestKeyManager_Encrypt_InvalidInternalKey(t *testing.T) {
	// Bypass constructor to test cipher creation error path with invalid AES key
	km := &KeyManager{masterKey: []byte("fifteen-bytes!!")} // 15 bytes, invalid for AES

	_, err := km.Encrypt([]byte("test"))
	if err == nil {
		t.Error("Encrypt() with invalid internal key expected error, got nil")
	}
}

func TestKeyManager_Decrypt_InvalidInternalKey(t *testing.T) {
	// Bypass constructor to test cipher creation error path with invalid AES key
	km := &KeyManager{masterKey: []byte("fifteen-bytes!!")} // 15 bytes, invalid for AES

	// Need data at least NonceSize bytes to pass the length check
	fakeData := make([]byte, NonceSize+16)
	_, err := km.Decrypt(fakeData)
	if err == nil {
		t.Error("Decrypt() with invalid internal key expected error, got nil")
	}
}

func TestKeyManager_EncryptString_InvalidInternalKey(t *testing.T) {
	// Bypass constructor to test EncryptString error propagation
	km := &KeyManager{masterKey: []byte("fifteen-bytes!!")} // 15 bytes, invalid for AES

	_, err := km.EncryptString("test")
	if err == nil {
		t.Error("EncryptString() with invalid internal key expected error, got nil")
	}
}

// failReader is an io.Reader that always returns an error.
type failReader struct{}

func (failReader) Read([]byte) (int, error) {
	return 0, errors.New("simulated rand failure")
}

func withFailingRand(t *testing.T, fn func()) {
	t.Helper()
	original := rand.Reader
	rand.Reader = failReader{}
	defer func() { rand.Reader = original }()
	fn()
}

func TestGenerateMasterKey_RandFailure(t *testing.T) {
	withFailingRand(t, func() {
		_, err := GenerateMasterKey()
		if err == nil {
			t.Error("GenerateMasterKey() with failing rand expected error, got nil")
		}
	})
}

func TestKeyManager_GeneratePassword_RandFailure(t *testing.T) {
	key := make([]byte, KeySize) // zero key is valid size
	km := &KeyManager{masterKey: key}

	withFailingRand(t, func() {
		_, err := km.GeneratePassword()
		if err == nil {
			t.Error("GeneratePassword() with failing rand expected error, got nil")
		}
	})
}

func TestKeyManager_Encrypt_NonceRandFailure(t *testing.T) {
	key := make([]byte, KeySize)
	km := &KeyManager{masterKey: key}

	// Nonce generation needs NonceSize bytes; provide 0 to fail immediately
	withFailingRand(t, func() {
		_, err := km.Encrypt([]byte("test"))
		if err == nil {
			t.Error("Encrypt() with failing rand expected error, got nil")
		}
	})
}

func TestKeyManager_CrossKeyIsolation(t *testing.T) {
	// Verify that data encrypted with one key cannot be decrypted with another
	keys := make([][]byte, 5)
	managers := make([]*KeyManager, 5)
	ciphertexts := make([][]byte, 5)

	plaintext := []byte("cross-key-isolation-test")

	for i := range keys {
		var err error
		keys[i], _ = GenerateMasterKey()
		managers[i], _ = NewKeyManager(keys[i])
		ciphertexts[i], err = managers[i].Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt() key %d error = %v", i, err)
		}
	}

	for i := range managers {
		for j := range ciphertexts {
			pt, err := managers[i].Decrypt(ciphertexts[j])
			if i == j {
				if err != nil {
					t.Errorf("key %d failed to decrypt own ciphertext: %v", i, err)
				}
				if !bytes.Equal(pt, plaintext) {
					t.Errorf("key %d decrypted own ciphertext incorrectly", i)
				}
			} else {
				if err == nil {
					t.Errorf("key %d should not decrypt ciphertext from key %d", i, j)
				}
			}
		}
	}
}

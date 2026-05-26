package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewEmailVerificationToken(t *testing.T) {
	userID := uuid.New()
	tok, raw, err := NewEmailVerificationToken(userID, 1*time.Hour)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if tok.UserID != userID {
		t.Errorf("expected userID match")
	}
	if raw == "" {
		t.Error("expected non-empty raw token")
	}
	if tok.TokenHash == "" {
		t.Error("expected non-empty hash")
	}
	if tok.IsUsed() {
		t.Error("expected unused on creation")
	}
}

func TestNewEmailVerificationToken_HashMatchesRaw(t *testing.T) {
	_, raw, _ := NewEmailVerificationToken(uuid.New(), time.Hour)
	expected := sha256.Sum256([]byte(raw))
	expectedHex := hex.EncodeToString(expected[:])
	if HashToken(raw) != expectedHex {
		t.Error("HashToken doesn't match expected SHA-256")
	}
}

func TestHashToken_DeterministicAndIdempotent(t *testing.T) {
	a := HashToken("foo")
	b := HashToken("foo")
	if a != b {
		t.Error("expected same hash for same input")
	}
	if a == HashToken("bar") {
		t.Error("expected different hash for different input")
	}
}

func TestEmailVerificationToken_IsExpired(t *testing.T) {
	expired := &EmailVerificationToken{ExpiresAt: time.Now().Add(-time.Minute)}
	if !expired.IsExpired() {
		t.Error("expected expired")
	}

	live := &EmailVerificationToken{ExpiresAt: time.Now().Add(time.Hour)}
	if live.IsExpired() {
		t.Error("expected not expired")
	}
}

func TestEmailVerificationToken_IsUsed(t *testing.T) {
	tok := &EmailVerificationToken{}
	if tok.IsUsed() {
		t.Error("expected unused")
	}
	now := time.Now()
	tok.UsedAt = &now
	if !tok.IsUsed() {
		t.Error("expected used after UsedAt set")
	}
}

func TestNewEmailVerificationToken_UniqueTokens(t *testing.T) {
	_, raw1, _ := NewEmailVerificationToken(uuid.New(), time.Hour)
	_, raw2, _ := NewEmailVerificationToken(uuid.New(), time.Hour)
	if raw1 == raw2 {
		t.Error("expected unique tokens (collision is astronomically unlikely)")
	}
}

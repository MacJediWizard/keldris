package models

import (
	"time"

	"github.com/google/uuid"
)

// RepositoryKey represents an encrypted Restic repository password.
type RepositoryKey struct {
	ID                 uuid.UUID `json:"id"`
	RepositoryID       uuid.UUID `json:"repository_id"`
	EncryptedKey       []byte    `json:"-"` // Encrypted, never expose in JSON
	EscrowEnabled      bool      `json:"escrow_enabled"`
	EscrowEncryptedKey []byte    `json:"-"` // Encrypted escrow copy, never expose in JSON
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// NewRepositoryKey creates a new RepositoryKey with the given details.
func NewRepositoryKey(repositoryID uuid.UUID, encryptedKey []byte, escrowEnabled bool, escrowEncryptedKey []byte) *RepositoryKey {
	now := time.Now()
	return &RepositoryKey{
		ID:                 uuid.New(),
		RepositoryID:       repositoryID,
		EncryptedKey:       encryptedKey,
		EscrowEnabled:      escrowEnabled,
		EscrowEncryptedKey: escrowEncryptedKey,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// HasEscrow returns true if key escrow is enabled and an escrow key exists.
func (rk *RepositoryKey) HasEscrow() bool {
	return rk.EscrowEnabled && len(rk.EscrowEncryptedKey) > 0
}

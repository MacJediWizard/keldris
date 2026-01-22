package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	// RegistrationCodeLength is the length of generated registration codes.
	RegistrationCodeLength = 8
	// RegistrationCodeExpiration is how long registration codes are valid.
	RegistrationCodeExpiration = 10 * time.Minute
	// RegistrationCodeChars is the character set for registration codes.
	// Using uppercase letters and digits, excluding ambiguous characters (0, O, I, L, 1).
	RegistrationCodeChars = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
)

// RegistrationCodeStore defines the interface for registration code persistence.
type RegistrationCodeStore interface {
	CreateRegistrationCode(ctx context.Context, code *models.RegistrationCode) error
	GetRegistrationCodeByCode(ctx context.Context, orgID uuid.UUID, code string) (*models.RegistrationCode, error)
	GetPendingRegistrationCodes(ctx context.Context, orgID uuid.UUID) ([]*models.RegistrationCode, error)
	MarkRegistrationCodeUsed(ctx context.Context, codeID, agentID uuid.UUID) error
	DeleteExpiredRegistrationCodes(ctx context.Context) error
}

// AgentMFA handles agent registration code generation and verification.
type AgentMFA struct {
	store  RegistrationCodeStore
	logger zerolog.Logger
}

// NewAgentMFA creates a new AgentMFA instance.
func NewAgentMFA(store RegistrationCodeStore, logger zerolog.Logger) *AgentMFA {
	return &AgentMFA{
		store:  store,
		logger: logger.With().Str("component", "agent_mfa").Logger(),
	}
}

// GenerateCode generates a new registration code for an organization.
func (m *AgentMFA) GenerateCode(ctx context.Context, orgID, userID uuid.UUID, hostname *string) (*models.RegistrationCode, error) {
	// Generate random code
	code, err := generateRandomCode(RegistrationCodeLength)
	if err != nil {
		return nil, fmt.Errorf("generate random code: %w", err)
	}

	expiresAt := time.Now().Add(RegistrationCodeExpiration)
	regCode := models.NewRegistrationCode(orgID, userID, code, hostname, expiresAt)

	if err := m.store.CreateRegistrationCode(ctx, regCode); err != nil {
		return nil, fmt.Errorf("create registration code: %w", err)
	}

	m.logger.Info().
		Str("code_id", regCode.ID.String()).
		Str("org_id", orgID.String()).
		Str("user_id", userID.String()).
		Time("expires_at", expiresAt).
		Msg("registration code created")

	return regCode, nil
}

// VerifyCode verifies a registration code and returns it if valid.
func (m *AgentMFA) VerifyCode(ctx context.Context, orgID uuid.UUID, code string) (*models.RegistrationCode, error) {
	// Normalize code to uppercase
	code = strings.ToUpper(strings.TrimSpace(code))

	regCode, err := m.store.GetRegistrationCodeByCode(ctx, orgID, code)
	if err != nil {
		m.logger.Debug().
			Str("org_id", orgID.String()).
			Str("code", code).
			Err(err).
			Msg("registration code not found")
		return nil, fmt.Errorf("invalid registration code")
	}

	if regCode.IsUsed() {
		m.logger.Warn().
			Str("code_id", regCode.ID.String()).
			Str("org_id", orgID.String()).
			Msg("registration code already used")
		return nil, fmt.Errorf("registration code already used")
	}

	if regCode.IsExpired() {
		m.logger.Warn().
			Str("code_id", regCode.ID.String()).
			Str("org_id", orgID.String()).
			Time("expired_at", regCode.ExpiresAt).
			Msg("registration code expired")
		return nil, fmt.Errorf("registration code expired")
	}

	m.logger.Debug().
		Str("code_id", regCode.ID.String()).
		Str("org_id", orgID.String()).
		Msg("registration code verified")

	return regCode, nil
}

// MarkCodeUsed marks a registration code as used by an agent.
func (m *AgentMFA) MarkCodeUsed(ctx context.Context, codeID, agentID uuid.UUID) error {
	if err := m.store.MarkRegistrationCodeUsed(ctx, codeID, agentID); err != nil {
		return fmt.Errorf("mark registration code used: %w", err)
	}

	m.logger.Info().
		Str("code_id", codeID.String()).
		Str("agent_id", agentID.String()).
		Msg("registration code marked as used")

	return nil
}

// GetPendingCodes returns all pending (unused, unexpired) registration codes for an organization.
func (m *AgentMFA) GetPendingCodes(ctx context.Context, orgID uuid.UUID) ([]*models.RegistrationCode, error) {
	codes, err := m.store.GetPendingRegistrationCodes(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get pending registration codes: %w", err)
	}
	return codes, nil
}

// CleanupExpiredCodes removes expired registration codes from the database.
func (m *AgentMFA) CleanupExpiredCodes(ctx context.Context) error {
	if err := m.store.DeleteExpiredRegistrationCodes(ctx); err != nil {
		return fmt.Errorf("delete expired registration codes: %w", err)
	}
	m.logger.Debug().Msg("cleaned up expired registration codes")
	return nil
}

// generateRandomCode generates a cryptographically secure random code.
func generateRandomCode(length int) (string, error) {
	charsetLen := big.NewInt(int64(len(RegistrationCodeChars)))
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = RegistrationCodeChars[num.Int64()]
	}

	return string(result), nil
}

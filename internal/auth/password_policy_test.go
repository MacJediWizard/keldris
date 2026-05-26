package auth

import (
	"context"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

type stubPwdStore struct {
	policy  *models.PasswordPolicy
	history []*models.PasswordHistory
	err     error
}

func (s *stubPwdStore) GetPasswordPolicyByOrgID(_ context.Context, orgID uuid.UUID) (*models.PasswordPolicy, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.policy != nil {
		return s.policy, nil
	}
	return models.NewPasswordPolicy(orgID), nil
}

func (s *stubPwdStore) GetPasswordHistory(_ context.Context, _ uuid.UUID, _ int) ([]*models.PasswordHistory, error) {
	return s.history, nil
}

func (s *stubPwdStore) CreatePasswordHistory(_ context.Context, _ *models.PasswordHistory) error {
	return nil
}

func (s *stubPwdStore) CleanupPasswordHistory(_ context.Context, _ uuid.UUID, _ int) error {
	return nil
}

func TestValidateAgainstPolicy_RequireUppercase(t *testing.T) {
	v := NewPasswordValidator(&stubPwdStore{})
	policy := &models.PasswordPolicy{MinLength: 8, RequireUppercase: true}

	res := v.ValidateAgainstPolicy("alllower1!", policy)
	if res.Valid {
		t.Error("expected invalid for missing uppercase")
	}
}

func TestValidateAgainstPolicy_RequireLowercase(t *testing.T) {
	v := NewPasswordValidator(&stubPwdStore{})
	policy := &models.PasswordPolicy{MinLength: 6, RequireLowercase: true}

	res := v.ValidateAgainstPolicy("UPPER123", policy)
	if res.Valid {
		t.Error("expected invalid for missing lowercase")
	}
}

func TestValidateAgainstPolicy_RequireNumber(t *testing.T) {
	v := NewPasswordValidator(&stubPwdStore{})
	policy := &models.PasswordPolicy{MinLength: 6, RequireNumber: true}

	res := v.ValidateAgainstPolicy("OnlyLetters", policy)
	if res.Valid {
		t.Error("expected invalid for missing number")
	}
}

func TestValidateAgainstPolicy_RequireSpecial(t *testing.T) {
	v := NewPasswordValidator(&stubPwdStore{})
	policy := &models.PasswordPolicy{MinLength: 6, RequireSpecial: true}

	res := v.ValidateAgainstPolicy("NoSpecial1", policy)
	if res.Valid {
		t.Error("expected invalid for missing special")
	}
}

func TestValidateAgainstPolicy_TooShort(t *testing.T) {
	v := NewPasswordValidator(&stubPwdStore{})
	policy := &models.PasswordPolicy{MinLength: 12}

	res := v.ValidateAgainstPolicy("short", policy)
	if res.Valid {
		t.Error("expected invalid for short password")
	}
}

func TestValidateAgainstPolicy_AllPass(t *testing.T) {
	v := NewPasswordValidator(&stubPwdStore{})
	policy := &models.PasswordPolicy{
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumber:    true,
	}

	res := v.ValidateAgainstPolicy("GoodPwd1", policy)
	if !res.Valid {
		t.Errorf("expected valid, got errors=%v", res.Errors)
	}
}

func TestValidateAgainstPolicy_WeakWarning(t *testing.T) {
	v := NewPasswordValidator(&stubPwdStore{})
	policy := &models.PasswordPolicy{MinLength: 6}

	res := v.ValidateAgainstPolicy("Short1A", policy)
	if !res.Valid {
		t.Error("expected valid")
	}
	if len(res.Warnings) == 0 {
		t.Error("expected length warning for short password")
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := VerifyPassword("hunter2", hash); err != nil {
		t.Errorf("verify should succeed: %v", err)
	}
	if err := VerifyPassword("wrong", hash); err == nil {
		t.Error("verify should fail for wrong password")
	}
}

func TestValidatePassword_UsesPolicyFromStore(t *testing.T) {
	policy := &models.PasswordPolicy{MinLength: 20}
	v := NewPasswordValidator(&stubPwdStore{policy: policy})

	res, err := v.ValidatePassword(context.Background(), uuid.New(), "short")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if res.Valid {
		t.Error("expected invalid (policy MinLength=20)")
	}
}

func TestValidatePassword_FallsBackToDefaultPolicy(t *testing.T) {
	// store returns error → falls back to NewPasswordPolicy
	v := NewPasswordValidator(&stubPwdStore{err: context.DeadlineExceeded})

	res, err := v.ValidatePassword(context.Background(), uuid.New(), "GoodLongPassword1")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !res.Valid {
		t.Errorf("expected valid with default policy, got errors=%v", res.Errors)
	}
}

package handlers

import (
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/google/uuid"
)

// testUser creates a SessionUser for testing with a given org.
func testUser(orgID uuid.UUID) *auth.SessionUser {
	return &auth.SessionUser{
		ID:             uuid.New(),
		Email:          "test@example.com",
		Name:           "Test User",
		CurrentOrgID:   orgID,
		CurrentOrgRole: "admin",
	}
}

// testUserNoOrg creates a SessionUser with no organization selected.
func testUserNoOrg() *auth.SessionUser {
	return &auth.SessionUser{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}
}

// Exported test helpers (InjectUser, JSONRequest, AuthenticatedRequest,
// SetupTestRouter, DoRequest) are in test_helpers.go

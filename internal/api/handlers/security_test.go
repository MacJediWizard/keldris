package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func TestSecurityTestHeaders(t *testing.T) {
	r := SetupTestRouter(testUser(uuid.New()))
	handler := NewSecurityHandler(zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/security/headers/test"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

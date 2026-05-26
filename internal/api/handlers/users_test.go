package handlers

import "testing"

func TestUsers(t *testing.T) {
	t.Skip("users handler depends on auth.RBAC + auth.SessionStore concrete types; covered by integration test against real DB")
}

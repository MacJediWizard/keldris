package handlers

import (
	"testing"
)

// DowntimeHandler depends on a concrete *monitoring.DowntimeService backed by
// a real database connection (via DowntimeStore inside the service). There is
// no clean service interface to mock; defer to integration tests.
func TestDowntimeHandler(t *testing.T) {
	t.Skip("requires real *monitoring.DowntimeService - no clean interface available")
}

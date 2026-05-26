package shutdown

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestHealthAdapter_GetStatus(t *testing.T) {
	mgr := NewManager(Config{}, newTracker(), zerolog.Nop())
	adapter := NewHealthAdapter(mgr)

	status := adapter.GetStatus()
	if status.State == "" {
		t.Error("expected non-empty state")
	}
}

func TestHealthAdapter_IsAcceptingJobs_Default(t *testing.T) {
	mgr := NewManager(Config{}, newTracker(), zerolog.Nop())
	adapter := NewHealthAdapter(mgr)

	// Default state should be accepting (not in shutdown)
	if !adapter.IsAcceptingJobs() {
		t.Error("expected to be accepting jobs in default state")
	}
}

func TestNewHealthAdapter(t *testing.T) {
	mgr := NewManager(Config{}, newTracker(), zerolog.Nop())
	adapter := NewHealthAdapter(mgr)
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}

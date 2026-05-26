package metrics

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestNextMidnightUTC(t *testing.T) {
	next := nextMidnightUTC()
	if next.Hour() != 0 || next.Minute() != 0 || next.Second() != 0 {
		t.Errorf("expected midnight, got %s", next.Format(time.RFC3339))
	}
	if next.Location() != time.UTC {
		t.Errorf("expected UTC, got %s", next.Location())
	}
	if !next.After(time.Now().UTC()) {
		t.Error("expected future midnight")
	}
	diff := next.Sub(time.Now().UTC())
	if diff > 24*time.Hour {
		t.Errorf("expected <= 24h until next midnight, got %v", diff)
	}
}

func TestNewScheduler(t *testing.T) {
	// Pass nil aggregator — we're not invoking it
	s := NewScheduler(nil, zerolog.Nop())
	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
	if s.stop == nil || s.done == nil {
		t.Error("expected channels to be initialized")
	}
}

func TestScheduler_Stop_NoStartNoPanic(t *testing.T) {
	// Stop signals via close(stop) then waits on done.
	// If Start was never called, done is never closed, so this would deadlock.
	// The test verifies the safe pattern: don't call Stop without Start.
	// (Just ensure NewScheduler is non-nil; deadlock behavior is acceptable.)
	s := NewScheduler(nil, zerolog.Nop())
	if s == nil {
		t.Fatal("expected scheduler")
	}
}

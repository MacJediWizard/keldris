package shutdown

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func newTracker() *SchedulerBackupTracker {
	return NewSchedulerBackupTracker(
		func(_ context.Context, _ uuid.UUID) error { return nil },
		func(_ context.Context, _ uuid.UUID) error { return nil },
		zerolog.Nop(),
	)
}

func TestSchedulerBackupTracker_RegisterUnregister(t *testing.T) {
	tr := newTracker()
	id := uuid.New()

	tr.RegisterBackup(id)
	if !tr.IsBackupRunning(id) {
		t.Error("expected backup to be running after register")
	}
	if tr.RunningCount() != 1 {
		t.Errorf("expected 1 running, got %d", tr.RunningCount())
	}

	tr.UnregisterBackup(id)
	if tr.IsBackupRunning(id) {
		t.Error("expected backup to NOT be running after unregister")
	}
	if tr.RunningCount() != 0 {
		t.Errorf("expected 0 running, got %d", tr.RunningCount())
	}
}

func TestSchedulerBackupTracker_GetRunningBackupIDs(t *testing.T) {
	tr := newTracker()
	id1 := uuid.New()
	id2 := uuid.New()

	tr.RegisterBackup(id1)
	tr.RegisterBackup(id2)

	ids := tr.GetRunningBackupIDs()
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs, got %d", len(ids))
	}
}

func TestSchedulerBackupTracker_CheckpointBackup(t *testing.T) {
	called := false
	tr := NewSchedulerBackupTracker(
		func(_ context.Context, _ uuid.UUID) error { called = true; return nil },
		func(_ context.Context, _ uuid.UUID) error { return nil },
		zerolog.Nop(),
	)
	id := uuid.New()
	tr.RegisterBackup(id)

	ok, err := tr.CheckpointBackup(context.Background(), id)
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if !ok {
		t.Error("expected checkpoint ok=true")
	}
	if !called {
		t.Error("expected checkpoint func called")
	}
}

func TestSchedulerBackupTracker_CheckpointWithNilFunc(t *testing.T) {
	tr := NewSchedulerBackupTracker(nil, nil, zerolog.Nop())
	ok, err := tr.CheckpointBackup(context.Background(), uuid.New())
	if ok {
		t.Error("expected ok=false when checkpointFn is nil")
	}
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestSchedulerBackupTracker_CancelBackup(t *testing.T) {
	called := false
	tr := NewSchedulerBackupTracker(
		func(_ context.Context, _ uuid.UUID) error { return nil },
		func(_ context.Context, _ uuid.UUID) error { called = true; return nil },
		zerolog.Nop(),
	)
	id := uuid.New()
	tr.RegisterBackup(id)

	if err := tr.CancelBackup(context.Background(), id); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if !called {
		t.Error("expected cancel func called")
	}
}

package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func TestSQLiteStore(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "keldris-queue-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)

	store, err := NewSQLiteStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test CreateQueuedBackup
	backup := &QueuedBackup{
		ID:           uuid.New(),
		ScheduleID:   uuid.New(),
		ScheduleName: "test-schedule",
		ScheduledAt:  time.Now().Add(-time.Hour),
		QueuedAt:     time.Now(),
		Status:       QueuedBackupStatusPending,
		RetryCount:   0,
	}

	if err := store.CreateQueuedBackup(ctx, backup); err != nil {
		t.Fatalf("create backup: %v", err)
	}

	// Test GetQueuedBackup
	retrieved, err := store.GetQueuedBackup(ctx, backup.ID)
	if err != nil {
		t.Fatalf("get backup: %v", err)
	}

	if retrieved.ID != backup.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, backup.ID)
	}
	if retrieved.ScheduleName != backup.ScheduleName {
		t.Errorf("ScheduleName mismatch: got %s, want %s", retrieved.ScheduleName, backup.ScheduleName)
	}
	if retrieved.Status != QueuedBackupStatusPending {
		t.Errorf("Status mismatch: got %s, want %s", retrieved.Status, QueuedBackupStatusPending)
	}

	// Test GetQueueCount
	count, err := store.GetQueueCount(ctx)
	if err != nil {
		t.Fatalf("get count: %v", err)
	}
	if count != 1 {
		t.Errorf("count mismatch: got %d, want 1", count)
	}

	// Test ListPendingBackups
	pending, err := store.ListPendingBackups(ctx)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("pending count mismatch: got %d, want 1", len(pending))
	}

	// Test UpdateQueuedBackup
	backup.Status = QueuedBackupStatusSynced
	now := time.Now()
	backup.SyncedAt = &now

	if err := store.UpdateQueuedBackup(ctx, backup); err != nil {
		t.Fatalf("update backup: %v", err)
	}

	// Verify update
	updated, err := store.GetQueuedBackup(ctx, backup.ID)
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.Status != QueuedBackupStatusSynced {
		t.Errorf("status not updated: got %s, want %s", updated.Status, QueuedBackupStatusSynced)
	}

	// Test GetQueueStatus
	status, err := store.GetQueueStatus(ctx)
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if status.SyncedCount != 1 {
		t.Errorf("synced count mismatch: got %d, want 1", status.SyncedCount)
	}
	if status.PendingCount != 0 {
		t.Errorf("pending count mismatch: got %d, want 0", status.PendingCount)
	}

	// Test DeleteQueuedBackup
	if err := store.DeleteQueuedBackup(ctx, backup.ID); err != nil {
		t.Fatalf("delete backup: %v", err)
	}

	// Verify deleted
	_, err = store.GetQueuedBackup(ctx, backup.ID)
	if err != ErrBackupNotFound {
		t.Errorf("expected ErrBackupNotFound, got %v", err)
	}

	// Verify count is 0
	count, err = store.GetQueueCount(ctx)
	if err != nil {
		t.Fatalf("get count after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("count after delete mismatch: got %d, want 0", count)
	}
}

func TestSQLiteStoreWithBackupResult(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "keldris-queue-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)

	store, err := NewSQLiteStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create backup with result
	backup := &QueuedBackup{
		ID:           uuid.New(),
		ScheduleID:   uuid.New(),
		ScheduleName: "test-with-result",
		ScheduledAt:  time.Now().Add(-time.Hour),
		QueuedAt:     time.Now(),
		Status:       QueuedBackupStatusPending,
		RetryCount:   0,
		BackupResult: &BackupResult{
			Success:      true,
			StartedAt:   time.Now().Add(-30 * time.Minute),
			CompletedAt: time.Now().Add(-20 * time.Minute),
			BytesAdded:  1024 * 1024,
			FilesNew:    10,
			FilesChanged: 5,
			SnapshotID:  "abc123",
		},
	}

	if err := store.CreateQueuedBackup(ctx, backup); err != nil {
		t.Fatalf("create backup: %v", err)
	}

	// Retrieve and verify
	retrieved, err := store.GetQueuedBackup(ctx, backup.ID)
	if err != nil {
		t.Fatalf("get backup: %v", err)
	}

	if retrieved.BackupResult == nil {
		t.Fatal("backup result is nil")
	}
	if !retrieved.BackupResult.Success {
		t.Error("backup result success mismatch")
	}
	if retrieved.BackupResult.BytesAdded != 1024*1024 {
		t.Errorf("bytes added mismatch: got %d, want %d", retrieved.BackupResult.BytesAdded, 1024*1024)
	}
	if retrieved.BackupResult.SnapshotID != "abc123" {
		t.Errorf("snapshot ID mismatch: got %s, want abc123", retrieved.BackupResult.SnapshotID)
	}
}

func TestDefaultQueueConfig(t *testing.T) {
	cfg := DefaultQueueConfig()

	if cfg.MaxQueueSize != 100 {
		t.Errorf("MaxQueueSize mismatch: got %d, want 100", cfg.MaxQueueSize)
	}
	if cfg.SyncInterval != 30*time.Second {
		t.Errorf("SyncInterval mismatch: got %v, want 30s", cfg.SyncInterval)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries mismatch: got %d, want 5", cfg.MaxRetries)
	}
}

func TestQueueDatabasePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "keldris-queue-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)

	store, err := NewSQLiteStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store.Close()

	// Verify database file was created
	dbPath := filepath.Join(tmpDir, "queue.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file not created")
	}
}

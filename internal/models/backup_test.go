package models

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewBackup(t *testing.T) {
	scheduleID := uuid.New()
	agentID := uuid.New()
	repoID := uuid.New()

	backup := NewBackup(scheduleID, agentID, &repoID)

	if backup.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if backup.ScheduleID != scheduleID {
		t.Errorf("expected ScheduleID %v, got %v", scheduleID, backup.ScheduleID)
	}
	if backup.AgentID != agentID {
		t.Errorf("expected AgentID %v, got %v", agentID, backup.AgentID)
	}
	if backup.RepositoryID == nil || *backup.RepositoryID != repoID {
		t.Errorf("expected RepositoryID %v, got %v", repoID, backup.RepositoryID)
	}
	if backup.Status != BackupStatusRunning {
		t.Errorf("expected Status %s, got %s", BackupStatusRunning, backup.Status)
	}
	if backup.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}
	if backup.CompletedAt != nil {
		t.Error("expected CompletedAt to be nil")
	}
}

func TestNewBackup_NilRepo(t *testing.T) {
	backup := NewBackup(uuid.New(), uuid.New(), nil)
	if backup.RepositoryID != nil {
		t.Error("expected nil RepositoryID")
	}
}

func TestBackup_Complete(t *testing.T) {
	backup := NewBackup(uuid.New(), uuid.New(), nil)

	backup.Complete("snap-123", 10, 5, 1024*1024)

	if backup.Status != BackupStatusCompleted {
		t.Errorf("expected Status %s, got %s", BackupStatusCompleted, backup.Status)
	}
	if backup.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
	if backup.SnapshotID != "snap-123" {
		t.Errorf("expected SnapshotID 'snap-123', got %s", backup.SnapshotID)
	}
	if backup.SizeBytes == nil || *backup.SizeBytes != 1024*1024 {
		t.Errorf("expected SizeBytes 1048576, got %v", backup.SizeBytes)
	}
	if backup.FilesNew == nil || *backup.FilesNew != 10 {
		t.Errorf("expected FilesNew 10, got %v", backup.FilesNew)
	}
	if backup.FilesChanged == nil || *backup.FilesChanged != 5 {
		t.Errorf("expected FilesChanged 5, got %v", backup.FilesChanged)
	}
}

func TestBackup_Fail(t *testing.T) {
	backup := NewBackup(uuid.New(), uuid.New(), nil)

	backup.Fail("disk full")

	if backup.Status != BackupStatusFailed {
		t.Errorf("expected Status %s, got %s", BackupStatusFailed, backup.Status)
	}
	if backup.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
	if backup.ErrorMessage != "disk full" {
		t.Errorf("expected ErrorMessage 'disk full', got %s", backup.ErrorMessage)
	}
}

func TestBackup_Cancel(t *testing.T) {
	backup := NewBackup(uuid.New(), uuid.New(), nil)

	backup.Cancel()

	if backup.Status != BackupStatusCanceled {
		t.Errorf("expected Status %s, got %s", BackupStatusCanceled, backup.Status)
	}
	if backup.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
}

func TestBackup_RecordRetention(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		backup := NewBackup(uuid.New(), uuid.New(), nil)
		backup.RecordRetention(3, 7, nil)

		if !backup.RetentionApplied {
			t.Error("expected RetentionApplied to be true")
		}
		if backup.SnapshotsRemoved == nil || *backup.SnapshotsRemoved != 3 {
			t.Errorf("expected SnapshotsRemoved 3, got %v", backup.SnapshotsRemoved)
		}
		if backup.SnapshotsKept == nil || *backup.SnapshotsKept != 7 {
			t.Errorf("expected SnapshotsKept 7, got %v", backup.SnapshotsKept)
		}
		if backup.RetentionError != "" {
			t.Errorf("expected empty RetentionError, got %s", backup.RetentionError)
		}
	})

	t.Run("with error", func(t *testing.T) {
		backup := NewBackup(uuid.New(), uuid.New(), nil)
		backup.RecordRetention(0, 10, errors.New("retention error"))

		if !backup.RetentionApplied {
			t.Error("expected RetentionApplied to be true")
		}
		if backup.RetentionError != "retention error" {
			t.Errorf("expected RetentionError 'retention error', got %s", backup.RetentionError)
		}
	})
}

func TestBackup_Duration(t *testing.T) {
	t.Run("not completed", func(t *testing.T) {
		backup := NewBackup(uuid.New(), uuid.New(), nil)
		if d := backup.Duration(); d != 0 {
			t.Errorf("expected 0 duration, got %v", d)
		}
	})

	t.Run("completed", func(t *testing.T) {
		backup := NewBackup(uuid.New(), uuid.New(), nil)
		time.Sleep(10 * time.Millisecond)
		backup.Complete("snap-1", 100, 1, 0)

		if d := backup.Duration(); d <= 0 {
			t.Errorf("expected positive duration, got %v", d)
		}
	})
}

func TestBackup_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   BackupStatus
		complete bool
	}{
		{"running", BackupStatusRunning, false},
		{"completed", BackupStatusCompleted, true},
		{"failed", BackupStatusFailed, true},
		{"canceled", BackupStatusCanceled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := &Backup{Status: tt.status}
			if got := backup.IsComplete(); got != tt.complete {
				t.Errorf("IsComplete() = %v, want %v", got, tt.complete)
			}
		})
	}
}

func TestBackup_RecordPreScript(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		backup := NewBackup(uuid.New(), uuid.New(), nil)
		backup.RecordPreScript("output here", nil)

		if backup.PreScriptOutput != "output here" {
			t.Errorf("expected PreScriptOutput 'output here', got %s", backup.PreScriptOutput)
		}
		if backup.PreScriptError != "" {
			t.Errorf("expected empty PreScriptError, got %s", backup.PreScriptError)
		}
	})

	t.Run("with error", func(t *testing.T) {
		backup := NewBackup(uuid.New(), uuid.New(), nil)
		backup.RecordPreScript("partial output", errors.New("script failed"))

		if backup.PreScriptOutput != "partial output" {
			t.Errorf("expected PreScriptOutput 'partial output', got %s", backup.PreScriptOutput)
		}
		if backup.PreScriptError != "script failed" {
			t.Errorf("expected PreScriptError 'script failed', got %s", backup.PreScriptError)
		}
	})
}

func TestBackup_RecordPostScript(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		backup := NewBackup(uuid.New(), uuid.New(), nil)
		backup.RecordPostScript("cleanup done", nil)

		if backup.PostScriptOutput != "cleanup done" {
			t.Errorf("expected PostScriptOutput 'cleanup done', got %s", backup.PostScriptOutput)
		}
		if backup.PostScriptError != "" {
			t.Errorf("expected empty PostScriptError, got %s", backup.PostScriptError)
		}
	})

	t.Run("with error", func(t *testing.T) {
		backup := NewBackup(uuid.New(), uuid.New(), nil)
		backup.RecordPostScript("", errors.New("cleanup failed"))

		if backup.PostScriptError != "cleanup failed" {
			t.Errorf("expected PostScriptError 'cleanup failed', got %s", backup.PostScriptError)
		}
	})
}

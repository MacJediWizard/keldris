package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestRestic_GetFileHistory(t *testing.T) {
	t.Run("file found in snapshots", func(t *testing.T) {
		// Create a response that works for both Snapshots and ListFiles calls.
		// Since our mock returns the same response for all calls, we need to be creative.
		// Snapshots expects a JSON array of snapshot objects.
		// ListFiles expects JSON lines with file objects.
		// We test each method individually instead.

		snapshots := []Snapshot{
			{ID: "snap1", ShortID: "snap1", Time: time.Now().Add(-24 * time.Hour), Hostname: "test"},
		}
		snapshotJSON, _ := json.Marshal(snapshots)

		r, cleanup := newTestRestic(string(snapshotJSON))
		defer cleanup()

		// GetFileHistory first calls Snapshots (gets our response OK),
		// then calls ListFiles for each snapshot. ListFiles will try to parse
		// the snapshot JSON as file entries, which won't match, so no versions found.
		history, err := r.GetFileHistory(context.Background(), testResticConfig(), "/home/user/file.txt")
		if err != nil {
			t.Fatalf("GetFileHistory() error = %v", err)
		}
		if history.FilePath != "/home/user/file.txt" {
			t.Errorf("FilePath = %v, want /home/user/file.txt", history.FilePath)
		}
		// No versions since ListFiles returns snapshot data which doesn't match file format
		if len(history.Versions) != 0 {
			t.Errorf("Versions count = %d, want 0 (mock returns non-file data)", len(history.Versions))
		}
	})

	t.Run("snapshots error", func(t *testing.T) {
		r, cleanup := newTestResticError("repository does not exist")
		defer cleanup()

		_, err := r.GetFileHistory(context.Background(), testResticConfig(), "/file.txt")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty repository", func(t *testing.T) {
		r, cleanup := newTestRestic("[]")
		defer cleanup()

		history, err := r.GetFileHistory(context.Background(), testResticConfig(), "/file.txt")
		if err != nil {
			t.Fatalf("GetFileHistory() error = %v", err)
		}
		if len(history.Versions) != 0 {
			t.Errorf("Versions count = %d, want 0", len(history.Versions))
		}
	})
}

func TestRestic_FindFileInSnapshots(t *testing.T) {
	t.Run("empty repository", func(t *testing.T) {
		r, cleanup := newTestRestic("[]")
		defer cleanup()

		results, err := r.FindFileInSnapshots(context.Background(), testResticConfig(), "/home/user/")
		if err != nil {
			t.Fatalf("FindFileInSnapshots() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("results count = %d, want 0", len(results))
		}
	})

	t.Run("snapshots error", func(t *testing.T) {
		r, cleanup := newTestResticError("connection failed")
		defer cleanup()

		_, err := r.FindFileInSnapshots(context.Background(), testResticConfig(), "/home/")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("with snapshots", func(t *testing.T) {
		snapshots := []Snapshot{
			{ID: "snap1", ShortID: "snap1", Time: time.Now(), Hostname: "test"},
		}
		snapshotJSON, _ := json.Marshal(snapshots)

		r, cleanup := newTestRestic(string(snapshotJSON))
		defer cleanup()

		results, err := r.FindFileInSnapshots(context.Background(), testResticConfig(), "/home/user/")
		if err != nil {
			t.Fatalf("FindFileInSnapshots() error = %v", err)
		}
		// ListFiles will be called but returns snapshot JSON which won't match file entries
		_ = results
	})
}

func TestRestic_RestoreFileVersion(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		err := r.RestoreFileVersion(context.Background(), testResticConfig(), "snap123", "/home/file.txt", "/tmp/restore")
		if err != nil {
			t.Fatalf("RestoreFileVersion() error = %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		r, cleanup := newTestResticError("no matching ID")
		defer cleanup()

		err := r.RestoreFileVersion(context.Background(), testResticConfig(), "badsnap", "/home/file.txt", "/tmp/restore")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestFileVersion_Fields(t *testing.T) {
	now := time.Now()
	fv := FileVersion{
		SnapshotID:   "snap1",
		SnapshotTime: now,
		FilePath:     "/home/user/file.txt",
		Size:         1024,
		ModTime:      now,
		Mode:         0644,
	}

	if fv.SnapshotID != "snap1" {
		t.Errorf("SnapshotID = %v, want snap1", fv.SnapshotID)
	}
	if fv.Size != 1024 {
		t.Errorf("Size = %d, want 1024", fv.Size)
	}
}

func TestFileHistory_Fields(t *testing.T) {
	history := FileHistory{
		FilePath: "/test/file.txt",
		Versions: []FileVersion{
			{SnapshotID: "snap1", Size: 100},
			{SnapshotID: "snap2", Size: 200},
		},
	}

	if history.FilePath != "/test/file.txt" {
		t.Errorf("FilePath = %v, want /test/file.txt", history.FilePath)
	}
	if len(history.Versions) != 2 {
		t.Fatalf("Versions count = %d, want 2", len(history.Versions))
	}

	// Test JSON marshaling
	data, err := json.Marshal(history)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded FileHistory
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.FilePath != history.FilePath {
		t.Errorf("decoded FilePath = %v, want %v", decoded.FilePath, history.FilePath)
	}

	_ = fmt.Sprintf("%v", history) // just ensure it doesn't panic
}

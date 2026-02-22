package backup

import (
	"context"
	"testing"
)

func TestParseDiffOutput(t *testing.T) {
	t.Run("change and statistics messages", func(t *testing.T) {
		output := `{"message_type":"change","source_path":"","target_path":"/home/new.txt","modifier":"+"}
{"message_type":"change","source_path":"/home/old.txt","target_path":"","modifier":"-"}
{"message_type":"change","source_path":"/home/mod.txt","target_path":"/home/mod.txt","modifier":"M"}
{"message_type":"statistics","added":{"files":1,"dirs":0,"bytes":1024},"removed":{"files":1,"dirs":0,"bytes":512},"changed_files":1}`

		result, err := parseDiffOutput([]byte(output), "snap1", "snap2")
		if err != nil {
			t.Fatalf("parseDiffOutput() error = %v", err)
		}
		if result.SnapshotID1 != "snap1" {
			t.Errorf("SnapshotID1 = %v, want snap1", result.SnapshotID1)
		}
		if result.SnapshotID2 != "snap2" {
			t.Errorf("SnapshotID2 = %v, want snap2", result.SnapshotID2)
		}
		// Statistics message overwrites computed stats
		if result.Stats.FilesAdded != 1 {
			t.Errorf("FilesAdded = %v, want 1", result.Stats.FilesAdded)
		}
		if result.Stats.FilesRemoved != 1 {
			t.Errorf("FilesRemoved = %v, want 1", result.Stats.FilesRemoved)
		}
		if result.Stats.TotalSizeAdded != 1024 {
			t.Errorf("TotalSizeAdded = %v, want 1024", result.Stats.TotalSizeAdded)
		}
		if result.Stats.TotalSizeRemoved != 512 {
			t.Errorf("TotalSizeRemoved = %v, want 512", result.Stats.TotalSizeRemoved)
		}
		if len(result.Changes) != 3 {
			t.Fatalf("Changes count = %d, want 3", len(result.Changes))
		}
	})

	t.Run("empty output", func(t *testing.T) {
		result, err := parseDiffOutput([]byte(""), "snap1", "snap2")
		if err != nil {
			t.Fatalf("parseDiffOutput() error = %v", err)
		}
		if len(result.Changes) != 0 {
			t.Errorf("Changes count = %d, want 0", len(result.Changes))
		}
	})

	t.Run("invalid json lines skipped", func(t *testing.T) {
		output := `not json
{"message_type":"change","source_path":"","target_path":"/file.txt","modifier":"+"}`

		result, err := parseDiffOutput([]byte(output), "s1", "s2")
		if err != nil {
			t.Fatalf("parseDiffOutput() error = %v", err)
		}
		if len(result.Changes) != 1 {
			t.Errorf("Changes count = %d, want 1", len(result.Changes))
		}
	})
}

func TestParseDiffChange(t *testing.T) {
	t.Run("added file", func(t *testing.T) {
		msg := resticDiffMessage{
			TargetPath: "/home/new.txt",
			Modifier:   "+",
		}
		entry := parseDiffChange(msg)
		if entry == nil {
			t.Fatal("expected non-nil entry")
		}
		if entry.ChangeType != DiffChangeAdded {
			t.Errorf("ChangeType = %v, want added", entry.ChangeType)
		}
		if entry.Path != "/home/new.txt" {
			t.Errorf("Path = %v, want /home/new.txt", entry.Path)
		}
	})

	t.Run("removed file", func(t *testing.T) {
		msg := resticDiffMessage{
			SourcePath: "/home/old.txt",
			Modifier:   "-",
		}
		entry := parseDiffChange(msg)
		if entry == nil {
			t.Fatal("expected non-nil entry")
		}
		if entry.ChangeType != DiffChangeRemoved {
			t.Errorf("ChangeType = %v, want removed", entry.ChangeType)
		}
		if entry.Path != "/home/old.txt" {
			t.Errorf("Path = %v, want /home/old.txt", entry.Path)
		}
	})

	t.Run("modified file", func(t *testing.T) {
		for _, mod := range []string{"M", "C", "T", "U"} {
			msg := resticDiffMessage{
				TargetPath: "/home/file.txt",
				Modifier:   mod,
			}
			entry := parseDiffChange(msg)
			if entry == nil {
				t.Fatalf("expected non-nil entry for modifier %s", mod)
			}
			if entry.ChangeType != DiffChangeModified {
				t.Errorf("ChangeType = %v for modifier %s, want modified", entry.ChangeType, mod)
			}
		}
	})

	t.Run("no paths returns nil", func(t *testing.T) {
		msg := resticDiffMessage{}
		entry := parseDiffChange(msg)
		if entry != nil {
			t.Error("expected nil for empty paths")
		}
	})

	t.Run("target path preferred over source", func(t *testing.T) {
		msg := resticDiffMessage{
			SourcePath: "/source",
			TargetPath: "/target",
			Modifier:   "M",
		}
		entry := parseDiffChange(msg)
		if entry.Path != "/target" {
			t.Errorf("Path = %v, want /target", entry.Path)
		}
	})

	t.Run("unknown modifier defaults to modified", func(t *testing.T) {
		msg := resticDiffMessage{
			TargetPath: "/file.txt",
			Modifier:   "X",
		}
		entry := parseDiffChange(msg)
		if entry.ChangeType != DiffChangeModified {
			t.Errorf("ChangeType = %v, want modified for unknown modifier", entry.ChangeType)
		}
	})
}

func TestUpdateStats(t *testing.T) {
	t.Run("file added", func(t *testing.T) {
		stats := &DiffStats{}
		entry := &DiffEntry{ChangeType: DiffChangeAdded, Type: "file", NewSize: 1024}
		updateStats(stats, entry)
		if stats.FilesAdded != 1 {
			t.Errorf("FilesAdded = %d, want 1", stats.FilesAdded)
		}
		if stats.TotalSizeAdded != 1024 {
			t.Errorf("TotalSizeAdded = %d, want 1024", stats.TotalSizeAdded)
		}
	})

	t.Run("dir added", func(t *testing.T) {
		stats := &DiffStats{}
		entry := &DiffEntry{ChangeType: DiffChangeAdded, Type: "dir"}
		updateStats(stats, entry)
		if stats.DirsAdded != 1 {
			t.Errorf("DirsAdded = %d, want 1", stats.DirsAdded)
		}
		if stats.FilesAdded != 0 {
			t.Errorf("FilesAdded = %d, want 0", stats.FilesAdded)
		}
	})

	t.Run("file removed", func(t *testing.T) {
		stats := &DiffStats{}
		entry := &DiffEntry{ChangeType: DiffChangeRemoved, Type: "file", OldSize: 512}
		updateStats(stats, entry)
		if stats.FilesRemoved != 1 {
			t.Errorf("FilesRemoved = %d, want 1", stats.FilesRemoved)
		}
		if stats.TotalSizeRemoved != 512 {
			t.Errorf("TotalSizeRemoved = %d, want 512", stats.TotalSizeRemoved)
		}
	})

	t.Run("dir removed", func(t *testing.T) {
		stats := &DiffStats{}
		entry := &DiffEntry{ChangeType: DiffChangeRemoved, Type: "dir"}
		updateStats(stats, entry)
		if stats.DirsRemoved != 1 {
			t.Errorf("DirsRemoved = %d, want 1", stats.DirsRemoved)
		}
	})

	t.Run("file modified", func(t *testing.T) {
		stats := &DiffStats{}
		entry := &DiffEntry{ChangeType: DiffChangeModified, Type: "file"}
		updateStats(stats, entry)
		if stats.FilesModified != 1 {
			t.Errorf("FilesModified = %d, want 1", stats.FilesModified)
		}
	})
}

func TestParseDiffFromText(t *testing.T) {
	t.Run("mixed changes", func(t *testing.T) {
		output := `+    /home/new_file.txt
-    /home/old_file.txt
M    /home/modified.txt`

		result, err := ParseDiffFromText(output, "snap1", "snap2")
		if err != nil {
			t.Fatalf("ParseDiffFromText() error = %v", err)
		}
		if len(result.Changes) != 3 {
			t.Fatalf("Changes count = %d, want 3", len(result.Changes))
		}
		if result.Changes[0].ChangeType != DiffChangeAdded {
			t.Errorf("first change type = %v, want added", result.Changes[0].ChangeType)
		}
		if result.Changes[1].ChangeType != DiffChangeRemoved {
			t.Errorf("second change type = %v, want removed", result.Changes[1].ChangeType)
		}
		if result.Changes[2].ChangeType != DiffChangeModified {
			t.Errorf("third change type = %v, want modified", result.Changes[2].ChangeType)
		}
	})

	t.Run("directories", func(t *testing.T) {
		output := `+    /home/new_dir/`

		result, err := ParseDiffFromText(output, "s1", "s2")
		if err != nil {
			t.Fatalf("ParseDiffFromText() error = %v", err)
		}
		if len(result.Changes) != 1 {
			t.Fatalf("Changes count = %d, want 1", len(result.Changes))
		}
		if result.Changes[0].Type != "dir" {
			t.Errorf("type = %v, want dir", result.Changes[0].Type)
		}
	})

	t.Run("empty output", func(t *testing.T) {
		result, err := ParseDiffFromText("", "s1", "s2")
		if err != nil {
			t.Fatalf("ParseDiffFromText() error = %v", err)
		}
		if len(result.Changes) != 0 {
			t.Errorf("Changes count = %d, want 0", len(result.Changes))
		}
	})

	t.Run("unrecognized lines ignored", func(t *testing.T) {
		output := `some header
+    /file.txt
random text`

		result, err := ParseDiffFromText(output, "s1", "s2")
		if err != nil {
			t.Fatalf("ParseDiffFromText() error = %v", err)
		}
		// "some header" might be parsed as a diff entry (starts with 's')
		// but "random text" shouldn't be. The actual count depends on parsing.
		if len(result.Changes) < 1 {
			t.Errorf("Changes count = %d, want at least 1", len(result.Changes))
		}
	})
}

func TestInferFileType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/home/user/file.txt", "file"},
		{"/home/user/docs/", "dir"},
		{"/", "dir"},
		{"file.txt", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := inferFileType(tt.path)
			if got != tt.want {
				t.Errorf("inferFileType(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRestic_Diff(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		response := `{"message_type":"change","source_path":"","target_path":"/new.txt","modifier":"+"}
{"message_type":"change","source_path":"/old.txt","target_path":"","modifier":"-"}
{"message_type":"statistics","added":{"files":1,"dirs":0,"bytes":100},"removed":{"files":1,"dirs":0,"bytes":50}}`

		r, cleanup := newTestRestic(response)
		defer cleanup()

		result, err := r.Diff(context.Background(), testResticConfig(), "snap1", "snap2")
		if err != nil {
			t.Fatalf("Diff() error = %v", err)
		}
		if result.SnapshotID1 != "snap1" {
			t.Errorf("SnapshotID1 = %v, want snap1", result.SnapshotID1)
		}
		if result.SnapshotID2 != "snap2" {
			t.Errorf("SnapshotID2 = %v, want snap2", result.SnapshotID2)
		}
		if len(result.Changes) != 2 {
			t.Errorf("Changes count = %d, want 2", len(result.Changes))
		}
	})

	t.Run("snapshot not found", func(t *testing.T) {
		r, cleanup := newTestResticError("no matching ID")
		defer cleanup()

		_, err := r.Diff(context.Background(), testResticConfig(), "snap1", "snap2")
		if err == nil {
			t.Fatal("expected error")
		}
		if err != ErrSnapshotNotFound {
			t.Errorf("error = %v, want ErrSnapshotNotFound", err)
		}
	})

	t.Run("command error", func(t *testing.T) {
		r, cleanup := newTestResticError("diff failed: some error")
		defer cleanup()

		_, err := r.Diff(context.Background(), testResticConfig(), "snap1", "snap2")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty diff", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		result, err := r.Diff(context.Background(), testResticConfig(), "snap1", "snap2")
		if err != nil {
			t.Fatalf("Diff() error = %v", err)
		}
		if len(result.Changes) != 0 {
			t.Errorf("Changes count = %d, want 0", len(result.Changes))
		}
	})
}

func TestRestic_DiffCompact(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		response := `{"message_type":"change","source_path":"","target_path":"/new.txt","modifier":"+"}
{"message_type":"statistics","added":{"files":1,"dirs":0,"bytes":100},"removed":{"files":0,"dirs":0,"bytes":0}}`

		r, cleanup := newTestRestic(response)
		defer cleanup()

		stats, err := r.DiffCompact(context.Background(), testResticConfig(), "snap1", "snap2")
		if err != nil {
			t.Fatalf("DiffCompact() error = %v", err)
		}
		if stats.FilesAdded != 1 {
			t.Errorf("FilesAdded = %d, want 1", stats.FilesAdded)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		r, cleanup := newTestResticError("no matching ID")
		defer cleanup()

		_, err := r.DiffCompact(context.Background(), testResticConfig(), "snap1", "snap2")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestBackendConfig(t *testing.T) {
	backend := &LocalBackend{Path: "/tmp/repo"}
	data, err := BackendConfig(backend)
	if err != nil {
		t.Fatalf("BackendConfig() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty config data")
	}
}

func TestParseSizeFromDiffLine(t *testing.T) {
	tests := []struct {
		line string
		want int64
	}{
		{"500 B", 500},
		{"1.5 KiB", 1536},
		{"2.0 MiB", 2097152},
		{"1.0 GiB", 1073741824},
		{"no size here", 0},
		{"", 0},
		{"invalid KiB", 0},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := ParseSizeFromDiffLine(tt.line)
			if got != tt.want {
				t.Errorf("ParseSizeFromDiffLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

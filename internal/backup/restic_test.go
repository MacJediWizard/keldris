package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// TestHelperProcess is used by tests to mock exec.Command via the
// test binary re-invocation pattern. When GO_WANT_HELPER_PROCESS is set,
// the test binary acts as the "restic" executable.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			_ = args[i+1:]
			break
		}
	}

	response := os.Getenv("GO_HELPER_RESPONSE")
	exitCode := os.Getenv("GO_HELPER_EXIT_CODE")
	stderrMsg := os.Getenv("GO_HELPER_STDERR")

	if stderrMsg != "" {
		fmt.Fprint(os.Stderr, stderrMsg)
	}

	if response != "" {
		fmt.Fprint(os.Stdout, response)
	}

	if exitCode == "1" {
		os.Exit(1)
	}
	os.Exit(0)
}

// testResticConfig returns a ResticConfig for testing.
func testResticConfig() ResticConfig {
	return ResticConfig{
		Repository: "/tmp/test-repo",
		Password:   "test-password",
		Env:        map[string]string{},
	}
}

// fakeExecCommand returns the path to the test binary configured for re-invocation.
// The caller must set environment variables on the returned Restic to control behavior.
func fakeExecCommand(response, stderrMsg string, exitCode int) string {
	// We use the test binary itself as the "restic" command.
	// TestHelperProcess handles the execution.
	cs := []string{"-test.run=TestHelperProcess", "--"}
	testBinary, _ := os.Executable()

	// We need to set env vars that TestHelperProcess reads.
	// But since Restic.run() sets env on the command, we use a wrapper script.
	script := fmt.Sprintf(`#!/bin/sh
export GO_WANT_HELPER_PROCESS=1
export GO_HELPER_RESPONSE='%s'
export GO_HELPER_EXIT_CODE='%d'
export GO_HELPER_STDERR='%s'
exec "%s" %s "$@"
`, strings.ReplaceAll(response, "'", "'\\''"),
		exitCode,
		strings.ReplaceAll(stderrMsg, "'", "'\\''"),
		testBinary,
		strings.Join(cs, " "))

	tmpFile, err := os.CreateTemp("", "fake-restic-*.sh")
	if err != nil {
		panic(err)
	}
	if _, err := tmpFile.WriteString(script); err != nil {
		panic(err)
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0755)

	return tmpFile.Name()
}

// newTestRestic creates a Restic wrapper pointing to a fake binary that outputs the given response.
func newTestRestic(response string) (*Restic, func()) {
	scriptPath := fakeExecCommand(response, "", 0)
	r := NewResticWithBinary(scriptPath, zerolog.Nop())
	return r, func() { os.Remove(scriptPath) }
}

// newTestResticError creates a Restic wrapper that simulates a command error.
func newTestResticError(stderrMsg string) (*Restic, func()) {
	scriptPath := fakeExecCommand("", stderrMsg, 1)
	r := NewResticWithBinary(scriptPath, zerolog.Nop())
	return r, func() { os.Remove(scriptPath) }
}

func TestRestic_Init(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, cleanup := newTestRestic(`{"id":"abc123"}`)
		defer cleanup()

		err := r.Init(context.Background(), testResticConfig())
		if err != nil {
			t.Fatalf("Init() error = %v", err)
		}
	})

	t.Run("already initialized", func(t *testing.T) {
		r, cleanup := newTestResticError("repository already exists")
		defer cleanup()

		err := r.Init(context.Background(), testResticConfig())
		if err != nil {
			t.Fatalf("Init() should succeed for already initialized repo, got error = %v", err)
		}
	})

	t.Run("already initialized alt message", func(t *testing.T) {
		r, cleanup := newTestResticError("repository already initialized")
		defer cleanup()

		err := r.Init(context.Background(), testResticConfig())
		if err != nil {
			t.Fatalf("Init() should succeed for already initialized repo, got error = %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		r, cleanup := newTestResticError("permission denied")
		defer cleanup()

		err := r.Init(context.Background(), testResticConfig())
		if err == nil {
			t.Fatal("Init() expected error")
		}
		if !strings.Contains(err.Error(), "init repository") {
			t.Errorf("error should contain 'init repository', got: %v", err)
		}
	})
}

func TestRestic_Backup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		response := `{"message_type":"status","percent_done":0.5}
{"message_type":"summary","snapshot_id":"abc123def","files_new":10,"files_changed":5,"files_unmodified":100,"data_added":1024000}`
		r, cleanup := newTestRestic(response)
		defer cleanup()

		stats, err := r.Backup(context.Background(), testResticConfig(), []string{"/home/user"}, []string{"*.tmp"}, []string{"test"})
		if err != nil {
			t.Fatalf("Backup() error = %v", err)
		}
		if stats.SnapshotID != "abc123def" {
			t.Errorf("SnapshotID = %v, want abc123def", stats.SnapshotID)
		}
		if stats.FilesNew != 10 {
			t.Errorf("FilesNew = %v, want 10", stats.FilesNew)
		}
		if stats.FilesChanged != 5 {
			t.Errorf("FilesChanged = %v, want 5", stats.FilesChanged)
		}
		if stats.SizeBytes != 1024000 {
			t.Errorf("SizeBytes = %v, want 1024000", stats.SizeBytes)
		}
		if stats.Duration <= 0 {
			t.Error("Duration should be positive")
		}
	})

	t.Run("no paths", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		_, err := r.Backup(context.Background(), testResticConfig(), nil, nil, nil)
		if err == nil {
			t.Fatal("Backup() expected error for no paths")
		}
		if !strings.Contains(err.Error(), "no paths specified") {
			t.Errorf("error should mention 'no paths specified', got: %v", err)
		}
	})

	t.Run("command error", func(t *testing.T) {
		r, cleanup := newTestResticError("backup failed: permission denied")
		defer cleanup()

		_, err := r.Backup(context.Background(), testResticConfig(), []string{"/home"}, nil, nil)
		if err == nil {
			t.Fatal("Backup() expected error")
		}
		if !strings.Contains(err.Error(), "backup failed") {
			t.Errorf("error should contain 'backup failed', got: %v", err)
		}
	})

	t.Run("no summary in output", func(t *testing.T) {
		response := `{"message_type":"status","percent_done":0.5}`
		r, cleanup := newTestRestic(response)
		defer cleanup()

		_, err := r.Backup(context.Background(), testResticConfig(), []string{"/home"}, nil, nil)
		if err == nil {
			t.Fatal("Backup() expected error for missing summary")
		}
		if !strings.Contains(err.Error(), "parse backup output") {
			t.Errorf("error should contain 'parse backup output', got: %v", err)
		}
	})
}

func TestRestic_BackupWithOptions(t *testing.T) {
	t.Run("with bandwidth limit", func(t *testing.T) {
		response := `{"message_type":"summary","snapshot_id":"snap1","files_new":1,"files_changed":0,"data_added":100}`
		r, cleanup := newTestRestic(response)
		defer cleanup()

		bw := 1024
		opts := &BackupOptions{BandwidthLimitKB: &bw}
		stats, err := r.BackupWithOptions(context.Background(), testResticConfig(), []string{"/data"}, nil, nil, opts)
		if err != nil {
			t.Fatalf("BackupWithOptions() error = %v", err)
		}
		if stats.SnapshotID != "snap1" {
			t.Errorf("SnapshotID = %v, want snap1", stats.SnapshotID)
		}
	})

	t.Run("with compression level", func(t *testing.T) {
		response := `{"message_type":"summary","snapshot_id":"snap2","files_new":1,"files_changed":0,"data_added":50}`
		r, cleanup := newTestRestic(response)
		defer cleanup()

		comp := "max"
		opts := &BackupOptions{CompressionLevel: &comp}
		stats, err := r.BackupWithOptions(context.Background(), testResticConfig(), []string{"/data"}, nil, nil, opts)
		if err != nil {
			t.Fatalf("BackupWithOptions() error = %v", err)
		}
		if stats.SnapshotID != "snap2" {
			t.Errorf("SnapshotID = %v, want snap2", stats.SnapshotID)
		}
	})

	t.Run("with both options", func(t *testing.T) {
		response := `{"message_type":"summary","snapshot_id":"snap3","files_new":2,"files_changed":1,"data_added":200}`
		r, cleanup := newTestRestic(response)
		defer cleanup()

		bw := 512
		comp := "auto"
		opts := &BackupOptions{BandwidthLimitKB: &bw, CompressionLevel: &comp}
		stats, err := r.BackupWithOptions(context.Background(), testResticConfig(), []string{"/data"}, []string{"*.log"}, []string{"daily"}, opts)
		if err != nil {
			t.Fatalf("BackupWithOptions() error = %v", err)
		}
		if stats.FilesNew != 2 {
			t.Errorf("FilesNew = %v, want 2", stats.FilesNew)
		}
	})

	t.Run("nil options", func(t *testing.T) {
		response := `{"message_type":"summary","snapshot_id":"snap4","files_new":0,"files_changed":0,"data_added":0}`
		r, cleanup := newTestRestic(response)
		defer cleanup()

		_, err := r.BackupWithOptions(context.Background(), testResticConfig(), []string{"/data"}, nil, nil, nil)
		if err != nil {
			t.Fatalf("BackupWithOptions() error = %v", err)
		}
	})
}

func TestRestic_Snapshots(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		snapshots := []Snapshot{
			{
				ID:       "abc123",
				ShortID:  "abc1",
				Time:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				Hostname: "testhost",
				Paths:    []string{"/home"},
			},
			{
				ID:       "def456",
				ShortID:  "def4",
				Time:     time.Date(2024, 1, 16, 10, 30, 0, 0, time.UTC),
				Hostname: "testhost",
				Paths:    []string{"/data"},
			},
		}
		data, _ := json.Marshal(snapshots)
		r, cleanup := newTestRestic(string(data))
		defer cleanup()

		result, err := r.Snapshots(context.Background(), testResticConfig())
		if err != nil {
			t.Fatalf("Snapshots() error = %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("Snapshots() returned %d snapshots, want 2", len(result))
		}
		if result[0].ID != "abc123" {
			t.Errorf("first snapshot ID = %v, want abc123", result[0].ID)
		}
		if result[1].Hostname != "testhost" {
			t.Errorf("second snapshot hostname = %v, want testhost", result[1].Hostname)
		}
	})

	t.Run("empty repository", func(t *testing.T) {
		r, cleanup := newTestRestic("[]")
		defer cleanup()

		result, err := r.Snapshots(context.Background(), testResticConfig())
		if err != nil {
			t.Fatalf("Snapshots() error = %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Snapshots() returned %d snapshots, want 0", len(result))
		}
	})

	t.Run("repository not initialized", func(t *testing.T) {
		r, cleanup := newTestResticError("repository does not exist")
		defer cleanup()

		_, err := r.Snapshots(context.Background(), testResticConfig())
		if err != ErrRepositoryNotInitialized {
			t.Errorf("Snapshots() error = %v, want ErrRepositoryNotInitialized", err)
		}
	})

	t.Run("command error", func(t *testing.T) {
		r, cleanup := newTestResticError("connection refused")
		defer cleanup()

		_, err := r.Snapshots(context.Background(), testResticConfig())
		if err == nil {
			t.Fatal("Snapshots() expected error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		r, cleanup := newTestRestic("not json")
		defer cleanup()

		_, err := r.Snapshots(context.Background(), testResticConfig())
		if err == nil {
			t.Fatal("Snapshots() expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "parse snapshots") {
			t.Errorf("error should contain 'parse snapshots', got: %v", err)
		}
	})
}

func TestRestic_ListFiles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// restic ls outputs newline-delimited JSON
		response := `{"struct_type":"snapshot","id":"abc123"}
{"name":"file.txt","type":"file","path":"/home/file.txt","size":1024,"mtime":"2024-01-15T10:30:00Z","atime":"2024-01-15T10:30:00Z","ctime":"2024-01-15T10:30:00Z"}
{"name":"docs","type":"dir","path":"/home/docs","size":0,"mtime":"2024-01-15T10:30:00Z","atime":"2024-01-15T10:30:00Z","ctime":"2024-01-15T10:30:00Z"}`
		r, cleanup := newTestRestic(response)
		defer cleanup()

		files, err := r.ListFiles(context.Background(), testResticConfig(), "abc123", "/home")
		if err != nil {
			t.Fatalf("ListFiles() error = %v", err)
		}
		if len(files) != 2 {
			t.Fatalf("ListFiles() returned %d files, want 2", len(files))
		}
		if files[0].Name != "file.txt" {
			t.Errorf("first file name = %v, want file.txt", files[0].Name)
		}
		if files[0].Type != "file" {
			t.Errorf("first file type = %v, want file", files[0].Type)
		}
		if files[1].Type != "dir" {
			t.Errorf("second file type = %v, want dir", files[1].Type)
		}
	})

	t.Run("snapshot not found", func(t *testing.T) {
		r, cleanup := newTestResticError("no matching ID found")
		defer cleanup()

		_, err := r.ListFiles(context.Background(), testResticConfig(), "nonexistent", "")
		if err != ErrSnapshotNotFound {
			t.Errorf("ListFiles() error = %v, want ErrSnapshotNotFound", err)
		}
	})

	t.Run("empty path prefix", func(t *testing.T) {
		response := `{"struct_type":"snapshot","id":"abc123"}
{"name":"file.txt","type":"file","path":"/file.txt","size":100,"mtime":"2024-01-15T10:30:00Z","atime":"2024-01-15T10:30:00Z","ctime":"2024-01-15T10:30:00Z"}`
		r, cleanup := newTestRestic(response)
		defer cleanup()

		files, err := r.ListFiles(context.Background(), testResticConfig(), "abc123", "")
		if err != nil {
			t.Fatalf("ListFiles() error = %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("ListFiles() returned %d files, want 1", len(files))
		}
	})
}

func TestRestic_Restore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		err := r.Restore(context.Background(), testResticConfig(), "abc123", RestoreOptions{
			TargetPath: "/tmp/restore",
		})
		if err != nil {
			t.Fatalf("Restore() error = %v", err)
		}
	})

	t.Run("with include and exclude", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		err := r.Restore(context.Background(), testResticConfig(), "abc123", RestoreOptions{
			TargetPath: "/tmp/restore",
			Include:    []string{"/home/user/docs"},
			Exclude:    []string{"*.tmp"},
		})
		if err != nil {
			t.Fatalf("Restore() error = %v", err)
		}
	})

	t.Run("snapshot not found", func(t *testing.T) {
		r, cleanup := newTestResticError("no matching ID found")
		defer cleanup()

		err := r.Restore(context.Background(), testResticConfig(), "nonexistent", RestoreOptions{
			TargetPath: "/tmp/restore",
		})
		if err != ErrSnapshotNotFound {
			t.Errorf("Restore() error = %v, want ErrSnapshotNotFound", err)
		}
	})

	t.Run("command error", func(t *testing.T) {
		r, cleanup := newTestResticError("disk full")
		defer cleanup()

		err := r.Restore(context.Background(), testResticConfig(), "abc123", RestoreOptions{
			TargetPath: "/tmp/restore",
		})
		if err == nil {
			t.Fatal("Restore() expected error")
		}
		if !strings.Contains(err.Error(), "restore failed") {
			t.Errorf("error should contain 'restore failed', got: %v", err)
		}
	})
}

func TestRestic_Forget(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		forgetOutput := `[{"tags":null,"host":"","paths":["/home"],"keep":[{"id":"snap1","short_id":"sn1","time":"2024-01-15T10:30:00Z","hostname":"test","paths":["/home"]},{"id":"snap2","short_id":"sn2","time":"2024-01-14T10:30:00Z","hostname":"test","paths":["/home"]}],"remove":[{"id":"snap3","short_id":"sn3","time":"2024-01-10T10:30:00Z","hostname":"test","paths":["/home"]}]}]`
		r, cleanup := newTestRestic(forgetOutput)
		defer cleanup()

		result, err := r.Forget(context.Background(), testResticConfig(), &models.RetentionPolicy{
			KeepDaily: 7,
		})
		if err != nil {
			t.Fatalf("Forget() error = %v", err)
		}
		if result.SnapshotsKept != 2 {
			t.Errorf("SnapshotsKept = %v, want 2", result.SnapshotsKept)
		}
		if result.SnapshotsRemoved != 1 {
			t.Errorf("SnapshotsRemoved = %v, want 1", result.SnapshotsRemoved)
		}
		if len(result.RemovedIDs) != 1 || result.RemovedIDs[0] != "sn3" {
			t.Errorf("RemovedIDs = %v, want [sn3]", result.RemovedIDs)
		}
	})

	t.Run("nil retention policy", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		_, err := r.Forget(context.Background(), testResticConfig(), nil)
		if err == nil {
			t.Fatal("Forget() expected error for nil retention")
		}
		if !strings.Contains(err.Error(), "retention policy required") {
			t.Errorf("error should contain 'retention policy required', got: %v", err)
		}
	})

	t.Run("command error", func(t *testing.T) {
		r, cleanup := newTestResticError("lock failed")
		defer cleanup()

		_, err := r.Forget(context.Background(), testResticConfig(), &models.RetentionPolicy{KeepLast: 5})
		if err == nil {
			t.Fatal("Forget() expected error")
		}
	})
}

func TestRestic_Prune(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		forgetOutput := `[{"keep":[{"id":"s1","short_id":"s1","time":"2024-01-15T10:30:00Z","hostname":"h","paths":["/"]}],"remove":[]}]`
		r, cleanup := newTestRestic(forgetOutput)
		defer cleanup()

		result, err := r.Prune(context.Background(), testResticConfig(), &models.RetentionPolicy{
			KeepLast: 5,
		})
		if err != nil {
			t.Fatalf("Prune() error = %v", err)
		}
		if result.SnapshotsKept != 1 {
			t.Errorf("SnapshotsKept = %v, want 1", result.SnapshotsKept)
		}
	})

	t.Run("nil retention policy", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		_, err := r.Prune(context.Background(), testResticConfig(), nil)
		if err == nil {
			t.Fatal("Prune() expected error for nil retention")
		}
	})
}

func TestRestic_PruneOnly(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		err := r.PruneOnly(context.Background(), testResticConfig())
		if err != nil {
			t.Fatalf("PruneOnly() error = %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		r, cleanup := newTestResticError("prune error")
		defer cleanup()

		err := r.PruneOnly(context.Background(), testResticConfig())
		if err == nil {
			t.Fatal("PruneOnly() expected error")
		}
	})
}

func TestRestic_Copy(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, cleanup := newTestRestic(`{"snapshot_id":"abc123"}`)
		defer cleanup()

		sourceCfg := ResticConfig{
			Repository: "/tmp/source-repo",
			Password:   "source-pass",
		}
		targetCfg := ResticConfig{
			Repository: "/tmp/target-repo",
			Password:   "target-pass",
		}

		err := r.Copy(context.Background(), sourceCfg, targetCfg, "abc123")
		if err != nil {
			t.Fatalf("Copy() error = %v", err)
		}
	})

	t.Run("with env vars", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		sourceCfg := ResticConfig{
			Repository: "s3:s3.amazonaws.com/source-bucket",
			Password:   "source-pass",
			Env: map[string]string{
				"AWS_ACCESS_KEY_ID": "source-key",
			},
		}
		targetCfg := ResticConfig{
			Repository: "s3:s3.amazonaws.com/target-bucket",
			Password:   "target-pass",
			Env: map[string]string{
				"AWS_ACCESS_KEY_ID": "target-key",
			},
		}

		err := r.Copy(context.Background(), sourceCfg, targetCfg, "snap1")
		if err != nil {
			t.Fatalf("Copy() error = %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		r, cleanup := newTestResticError("copy failed")
		defer cleanup()

		err := r.Copy(context.Background(), testResticConfig(), testResticConfig(), "abc123")
		if err == nil {
			t.Fatal("Copy() expected error")
		}
		if !strings.Contains(err.Error(), "copy snapshot") {
			t.Errorf("error should contain 'copy snapshot', got: %v", err)
		}
	})
}

func TestRestic_Check(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		err := r.Check(context.Background(), testResticConfig())
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		r, cleanup := newTestResticError("check failed: corrupt data")
		defer cleanup()

		err := r.Check(context.Background(), testResticConfig())
		if err == nil {
			t.Fatal("Check() expected error")
		}
	})
}

func TestRestic_CheckWithOptions(t *testing.T) {
	t.Run("basic check", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		result, err := r.CheckWithOptions(context.Background(), testResticConfig(), CheckOptions{})
		if err != nil {
			t.Fatalf("CheckWithOptions() error = %v", err)
		}
		if result == nil {
			t.Fatal("CheckWithOptions() result should not be nil")
		}
		if result.Duration <= 0 {
			t.Error("Duration should be positive")
		}
		if len(result.Errors) != 0 {
			t.Errorf("Errors = %v, want empty", result.Errors)
		}
	})

	t.Run("with read data", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		result, err := r.CheckWithOptions(context.Background(), testResticConfig(), CheckOptions{
			ReadData: true,
		})
		if err != nil {
			t.Fatalf("CheckWithOptions() error = %v", err)
		}
		if result == nil {
			t.Fatal("result should not be nil")
		}
	})

	t.Run("with read data subset", func(t *testing.T) {
		r, cleanup := newTestRestic("")
		defer cleanup()

		result, err := r.CheckWithOptions(context.Background(), testResticConfig(), CheckOptions{
			ReadData:       true,
			ReadDataSubset: "2.5%",
		})
		if err != nil {
			t.Fatalf("CheckWithOptions() error = %v", err)
		}
		if result == nil {
			t.Fatal("result should not be nil")
		}
	})

	t.Run("check with errors", func(t *testing.T) {
		r, cleanup := newTestResticError("pack file corrupt")
		defer cleanup()

		result, err := r.CheckWithOptions(context.Background(), testResticConfig(), CheckOptions{})
		if err == nil {
			t.Fatal("expected error")
		}
		if result == nil {
			t.Fatal("result should not be nil even on error")
		}
		if len(result.Errors) != 1 {
			t.Errorf("Errors count = %d, want 1", len(result.Errors))
		}
	})
}

func TestRestic_Stats(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		statsJSON := `{"total_size":1048576,"total_file_count":42}`
		r, cleanup := newTestRestic(statsJSON)
		defer cleanup()

		stats, err := r.Stats(context.Background(), testResticConfig())
		if err != nil {
			t.Fatalf("Stats() error = %v", err)
		}
		if stats.TotalSize != 1048576 {
			t.Errorf("TotalSize = %v, want 1048576", stats.TotalSize)
		}
		if stats.TotalFileCount != 42 {
			t.Errorf("TotalFileCount = %v, want 42", stats.TotalFileCount)
		}
	})

	t.Run("error", func(t *testing.T) {
		r, cleanup := newTestResticError("stats failed")
		defer cleanup()

		_, err := r.Stats(context.Background(), testResticConfig())
		if err == nil {
			t.Fatal("Stats() expected error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		r, cleanup := newTestRestic("not json")
		defer cleanup()

		_, err := r.Stats(context.Background(), testResticConfig())
		if err == nil {
			t.Fatal("Stats() expected error for invalid JSON")
		}
	})
}

func TestRestic_ContextCancellation(t *testing.T) {
	r := NewResticWithBinary("sleep", zerolog.Nop())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := r.Init(ctx, testResticConfig())
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func TestRestic_NewRestic(t *testing.T) {
	r := NewRestic(zerolog.Nop())
	if r.binary != "restic" {
		t.Errorf("binary = %v, want restic", r.binary)
	}
}

func TestRestic_NewResticWithBinary(t *testing.T) {
	r := NewResticWithBinary("/usr/local/bin/restic", zerolog.Nop())
	if r.binary != "/usr/local/bin/restic" {
		t.Errorf("binary = %v, want /usr/local/bin/restic", r.binary)
	}
}

func TestParseForgetOutput(t *testing.T) {
	t.Run("array format", func(t *testing.T) {
		output := `[{"keep":[{"id":"s1","short_id":"s1","time":"2024-01-15T00:00:00Z","hostname":"h","paths":["/"]}],"remove":[{"id":"s2","short_id":"s2","time":"2024-01-10T00:00:00Z","hostname":"h","paths":["/"]}]}]`
		result, err := parseForgetOutput([]byte(output))
		if err != nil {
			t.Fatalf("parseForgetOutput() error = %v", err)
		}
		if result.SnapshotsKept != 1 {
			t.Errorf("SnapshotsKept = %v, want 1", result.SnapshotsKept)
		}
		if result.SnapshotsRemoved != 1 {
			t.Errorf("SnapshotsRemoved = %v, want 1", result.SnapshotsRemoved)
		}
	})

	t.Run("multi group array", func(t *testing.T) {
		output := `[{"keep":[{"id":"s1","short_id":"s1","time":"2024-01-15T00:00:00Z","hostname":"h","paths":["/"]}],"remove":[{"id":"s2","short_id":"s2","time":"2024-01-10T00:00:00Z","hostname":"h","paths":["/"]}]},{"keep":[{"id":"s3","short_id":"s3","time":"2024-01-15T00:00:00Z","hostname":"h","paths":["/data"]}],"remove":[]}]`
		result, err := parseForgetOutput([]byte(output))
		if err != nil {
			t.Fatalf("parseForgetOutput() error = %v", err)
		}
		if result.SnapshotsKept != 2 {
			t.Errorf("SnapshotsKept = %v, want 2", result.SnapshotsKept)
		}
		if result.SnapshotsRemoved != 1 {
			t.Errorf("SnapshotsRemoved = %v, want 1", result.SnapshotsRemoved)
		}
	})

	t.Run("line-by-line format", func(t *testing.T) {
		output := `{"keep":[{"id":"s1","short_id":"s1","time":"2024-01-15T00:00:00Z","hostname":"h","paths":["/"]}],"remove":[]}`
		result, err := parseForgetOutput([]byte(output))
		if err != nil {
			t.Fatalf("parseForgetOutput() error = %v", err)
		}
		if result.SnapshotsKept != 1 {
			t.Errorf("SnapshotsKept = %v, want 1", result.SnapshotsKept)
		}
	})

	t.Run("empty output", func(t *testing.T) {
		result, err := parseForgetOutput([]byte(""))
		if err != nil {
			t.Fatalf("parseForgetOutput() error = %v", err)
		}
		if result.SnapshotsKept != 0 || result.SnapshotsRemoved != 0 {
			t.Errorf("expected zero counts for empty output")
		}
	})
}

func TestBuildRetentionArgs(t *testing.T) {
	r := NewRestic(zerolog.Nop())

	t.Run("all retention options", func(t *testing.T) {
		policy := &models.RetentionPolicy{
			KeepLast:    5,
			KeepHourly:  24,
			KeepDaily:   7,
			KeepWeekly:  4,
			KeepMonthly: 6,
			KeepYearly:  2,
		}

		args := r.buildRetentionArgs("/tmp/repo", policy)

		expected := map[string]string{
			"--keep-last":    "5",
			"--keep-hourly":  "24",
			"--keep-daily":   "7",
			"--keep-weekly":  "4",
			"--keep-monthly": "6",
			"--keep-yearly":  "2",
		}

		for flag, value := range expected {
			found := false
			for i, arg := range args {
				if arg == flag && i+1 < len(args) && args[i+1] == value {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("args should contain %s %s, got: %v", flag, value, args)
			}
		}
	})

	t.Run("partial retention", func(t *testing.T) {
		policy := &models.RetentionPolicy{
			KeepDaily: 7,
		}

		args := r.buildRetentionArgs("/tmp/repo", policy)

		for _, arg := range args {
			if arg == "--keep-last" || arg == "--keep-hourly" || arg == "--keep-weekly" {
				t.Errorf("should not have %s for daily-only policy", arg)
			}
		}

		found := false
		for i, arg := range args {
			if arg == "--keep-daily" && i+1 < len(args) && args[i+1] == "7" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected --keep-daily 7, got: %v", args)
		}
	})

	t.Run("zero values excluded", func(t *testing.T) {
		policy := &models.RetentionPolicy{
			KeepLast:  0,
			KeepDaily: 7,
		}

		args := r.buildRetentionArgs("/tmp/repo", policy)
		for _, arg := range args {
			if arg == "--keep-last" {
				t.Error("should not include --keep-last for zero value")
			}
		}
	})
}

func TestRestic_Run_EnvVars(t *testing.T) {
	// Verify that environment variables are set correctly
	if os.Getenv("CI") == "true" {
		t.Skip("skipping in CI - needs test binary re-invocation")
	}

	// Create a script that prints its environment
	script := `#!/bin/sh
echo "$RESTIC_PASSWORD|$AWS_ACCESS_KEY_ID"
`
	tmpFile, err := os.CreateTemp("", "env-test-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(script)
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0755)

	r := NewResticWithBinary(tmpFile.Name(), zerolog.Nop())
	cfg := ResticConfig{
		Repository: "/tmp/repo",
		Password:   "my-secret",
		Env: map[string]string{
			"AWS_ACCESS_KEY_ID": "test-key",
		},
	}

	output, err := r.run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	out := strings.TrimSpace(string(output))
	if !strings.Contains(out, "my-secret") {
		t.Errorf("expected RESTIC_PASSWORD to be set, got: %v", out)
	}
	if !strings.Contains(out, "test-key") {
		t.Errorf("expected AWS_ACCESS_KEY_ID to be set, got: %v", out)
	}
}

// Verify exec.Command properly handles a non-existent binary.
func TestRestic_Run_BinaryNotFound(t *testing.T) {
	r := NewResticWithBinary("/nonexistent/restic", zerolog.Nop())
	err := r.Init(context.Background(), testResticConfig())
	if err == nil {
		t.Fatal("expected error for non-existent binary")
	}

	// Should contain exec error info
	var execErr *exec.Error
	if !strings.Contains(err.Error(), "init repository") {
		t.Errorf("error should wrap init repository context, got: %v", err)
	}
	_ = execErr
}

func TestParseBackupOutput_Extended(t *testing.T) {
	t.Run("multiple status lines before summary", func(t *testing.T) {
		output := `{"message_type":"status","percent_done":0.1,"total_files":100,"files_done":10}
{"message_type":"status","percent_done":0.5,"total_files":100,"files_done":50}
{"message_type":"status","percent_done":0.9,"total_files":100,"files_done":90}
{"message_type":"summary","snapshot_id":"full123","files_new":100,"files_changed":0,"data_added":5242880}`
		stats, err := parseBackupOutput([]byte(output))
		if err != nil {
			t.Fatalf("parseBackupOutput() error = %v", err)
		}
		if stats.SnapshotID != "full123" {
			t.Errorf("SnapshotID = %v, want full123", stats.SnapshotID)
		}
		if stats.SizeBytes != 5242880 {
			t.Errorf("SizeBytes = %v, want 5242880", stats.SizeBytes)
		}
	})

	t.Run("invalid json line skipped", func(t *testing.T) {
		output := `not-json-line
{"message_type":"summary","snapshot_id":"snap1","files_new":1,"files_changed":0,"data_added":100}`
		stats, err := parseBackupOutput([]byte(output))
		if err != nil {
			t.Fatalf("parseBackupOutput() error = %v", err)
		}
		if stats.SnapshotID != "snap1" {
			t.Errorf("SnapshotID = %v, want snap1", stats.SnapshotID)
		}
	})
	"testing"
	"time"
)

func TestParseBackupOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    *BackupStats
		wantErr bool
	}{
		{
			name: "valid summary",
			output: `{"message_type":"status","percent_done":0.5}
{"message_type":"summary","snapshot_id":"abc123def","files_new":10,"files_changed":5,"files_unmodified":100,"data_added":1024000}`,
			want: &BackupStats{
				SnapshotID:   "abc123def",
				FilesNew:     10,
				FilesChanged: 5,
				SizeBytes:    1024000,
			},
			wantErr: false,
		},
		{
			name:    "no summary",
			output:  `{"message_type":"status","percent_done":0.5}`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBackupOutput([]byte(tt.output))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBackupOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.SnapshotID != tt.want.SnapshotID {
				t.Errorf("SnapshotID = %v, want %v", got.SnapshotID, tt.want.SnapshotID)
			}
			if got.FilesNew != tt.want.FilesNew {
				t.Errorf("FilesNew = %v, want %v", got.FilesNew, tt.want.FilesNew)
			}
			if got.FilesChanged != tt.want.FilesChanged {
				t.Errorf("FilesChanged = %v, want %v", got.FilesChanged, tt.want.FilesChanged)
			}
			if got.SizeBytes != tt.want.SizeBytes {
				t.Errorf("SizeBytes = %v, want %v", got.SizeBytes, tt.want.SizeBytes)
			}
		})
	}
}

func TestSnapshot_Time(t *testing.T) {
	snapshot := Snapshot{
		ID:       "abc123",
		ShortID:  "abc1",
		Time:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Hostname: "testhost",
		Paths:    []string{"/home/user"},
	}

	if snapshot.ID != "abc123" {
		t.Errorf("ID = %v, want abc123", snapshot.ID)
	}
	if snapshot.Hostname != "testhost" {
		t.Errorf("Hostname = %v, want testhost", snapshot.Hostname)
	}
	if len(snapshot.Paths) != 1 || snapshot.Paths[0] != "/home/user" {
		t.Errorf("Paths = %v, want [/home/user]", snapshot.Paths)
	}
}

func TestBackupStats_Duration(t *testing.T) {
	stats := BackupStats{
		SnapshotID:   "abc123",
		FilesNew:     10,
		FilesChanged: 5,
		SizeBytes:    1024,
		Duration:     5 * time.Second,
	}

	if stats.Duration != 5*time.Second {
		t.Errorf("Duration = %v, want 5s", stats.Duration)
	}
}

func TestRedactArgs(t *testing.T) {
	args := []string{"backup", "--repo", "/path/to/repo", "/home/user"}
	redacted := redactArgs(args)

	if len(redacted) != len(args) {
		t.Errorf("redactArgs length = %d, want %d", len(redacted), len(args))
	}

	for i, arg := range args {
		if redacted[i] != arg {
			t.Errorf("redactArgs[%d] = %v, want %v", i, redacted[i], arg)
		}
	}
}

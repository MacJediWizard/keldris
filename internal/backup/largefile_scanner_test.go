package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func newScanner() *LargeFileScanner {
	return NewLargeFileScanner(zerolog.Nop())
}

func writeFile(t *testing.T, path string, sizeBytes int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := make([]byte, sizeBytes)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestLargeFileScanner_NoMaxSize(t *testing.T) {
	s := newScanner()
	result, err := s.Scan(context.Background(), []string{}, nil, 0)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if result.TotalExcluded != 0 {
		t.Errorf("expected 0 excluded for maxSizeMB=0, got %d", result.TotalExcluded)
	}
}

func TestLargeFileScanner_NegativeMaxSize(t *testing.T) {
	s := newScanner()
	result, err := s.Scan(context.Background(), []string{}, nil, -5)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if result.TotalExcluded != 0 {
		t.Errorf("expected 0 excluded for negative maxSizeMB")
	}
}

func TestLargeFileScanner_FindsLargeFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "small.txt"), 100)
	writeFile(t, filepath.Join(dir, "big.bin"), 2*1024*1024)            // 2 MB
	writeFile(t, filepath.Join(dir, "nested", "huge.bin"), 3*1024*1024) // 3 MB

	s := newScanner()
	result, err := s.Scan(context.Background(), []string{dir}, nil, 1)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if result.TotalExcluded != 2 {
		t.Errorf("expected 2 large files, got %d", result.TotalExcluded)
	}
	if len(result.LargeFiles) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result.LargeFiles))
	}
}

func TestLargeFileScanner_RespectsExcludes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "big.log"), 2*1024*1024)
	writeFile(t, filepath.Join(dir, "big.bin"), 2*1024*1024)

	s := newScanner()
	result, err := s.Scan(context.Background(), []string{dir}, []string{"*.log"}, 1)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if result.TotalExcluded != 1 {
		t.Errorf("expected 1 large file (big.log excluded), got %d", result.TotalExcluded)
	}
	if len(result.LargeFiles) == 1 && filepath.Base(result.LargeFiles[0].Path) != "big.bin" {
		t.Errorf("expected big.bin to remain, got %s", result.LargeFiles[0].Path)
	}
}

func TestLargeFileScanner_CancelledContext(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "f.bin"), 100)

	s := newScanner()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.Scan(ctx, []string{dir}, nil, 1)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestLargeFileScanner_MissingPathDoesNotCrash(t *testing.T) {
	s := newScanner()
	result, err := s.Scan(context.Background(), []string{"/nonexistent/path"}, nil, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalExcluded != 0 {
		t.Errorf("expected 0 excluded for missing path, got %d", result.TotalExcluded)
	}
}

func TestFormatExcludedFiles_Empty(t *testing.T) {
	s := newScanner()
	if files := s.FormatExcludedFiles(nil, 10); files != nil {
		t.Errorf("expected nil for nil result")
	}
	if files := s.FormatExcludedFiles(&ScanResult{}, 10); files != nil {
		t.Errorf("expected nil for empty list")
	}
}

func TestFormatExcludedFiles_Truncates(t *testing.T) {
	result := &ScanResult{
		LargeFiles: []LargeFile{
			{Path: "/a"}, {Path: "/b"}, {Path: "/c"}, {Path: "/d"},
		},
	}
	s := newScanner()
	files := s.FormatExcludedFiles(result, 2)
	if len(files) != 2 {
		t.Errorf("expected truncation to 2, got %d", len(files))
	}
	if files[0] != "/a" || files[1] != "/b" {
		t.Errorf("expected first 2 paths, got %v", files)
	}
}

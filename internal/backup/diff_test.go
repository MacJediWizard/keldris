package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := hashFile(path)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}

	want := sha256.Sum256(content)
	expected := hex.EncodeToString(want[:])

	if got != expected {
		t.Errorf("hash mismatch: got %s, want %s", got, expected)
	}
}

func TestHashFile_FileNotFound(t *testing.T) {
	_, err := hashFile("/nonexistent/path/should/fail")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestHashFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := hashFile(path)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}

	// SHA-256 of empty input
	if got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("unexpected empty hash: %s", got)
	}
}

func TestIsBinaryFile_TextFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "text.txt")
	if err := os.WriteFile(path, []byte("hello world\nthis is text\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if isBinaryFile(path) {
		t.Error("expected text file to not be binary")
	}
}

func TestIsBinaryFile_BinaryWithNullByte(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.bin")
	// content with embedded NUL byte
	if err := os.WriteFile(path, []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0x57, 0x6f, 0x72, 0x6c, 0x64}, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if !isBinaryFile(path) {
		t.Error("expected file with null byte to be detected as binary")
	}
}

func TestIsBinaryFile_InvalidUTF8(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.bin")
	// Invalid UTF-8 sequence
	if err := os.WriteFile(path, []byte{0xff, 0xfe, 0xfd}, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if !isBinaryFile(path) {
		t.Error("expected invalid UTF-8 to be detected as binary")
	}
}

func TestIsBinaryFile_MissingFile(t *testing.T) {
	// missing file → returns false (per implementation)
	if isBinaryFile("/nonexistent/path") {
		t.Error("expected false for missing file")
	}
}

func TestMaxFileSizeForDiffConstant(t *testing.T) {
	if MaxFileSizeForDiff != 10*1024*1024 {
		t.Errorf("expected MaxFileSizeForDiff = 10 MB, got %d", MaxFileSizeForDiff)
	}
}

package maintenance

import (
	"testing"
)

func TestDefaultDatabaseBackupConfig(t *testing.T) {
	cfg := DefaultDatabaseBackupConfig()
	if cfg.BackupDir != "/var/lib/keldris/backups" {
		t.Errorf("expected default backup dir, got %s", cfg.BackupDir)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("expected 30 retention days, got %d", cfg.RetentionDays)
	}
	if cfg.CronExpression != "0 0 2 * * *" {
		t.Errorf("expected daily 2AM cron, got %s", cfg.CronExpression)
	}
	if cfg.CompressLevel != 6 {
		t.Errorf("expected compression level 6, got %d", cfg.CompressLevel)
	}
}

func TestSHA256Sum_NotEmpty(t *testing.T) {
	got := sha256Sum([]byte("hello"))
	if len(got) != 32 {
		t.Errorf("expected SHA-256 output of 32 bytes, got %d", len(got))
	}
}

func TestSHA256Sum_Deterministic(t *testing.T) {
	a := sha256Sum([]byte("input"))
	b := sha256Sum([]byte("input"))
	for i := range a {
		if a[i] != b[i] {
			t.Error("expected deterministic output")
			break
		}
	}
}

func TestSHA256Sum_DifferentInputs(t *testing.T) {
	a := sha256Sum([]byte("a"))
	b := sha256Sum([]byte("b"))
	same := true
	for i := range a {
		if a[i] != b[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("expected different output for different input")
	}
}

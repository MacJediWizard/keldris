package backup

import (
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

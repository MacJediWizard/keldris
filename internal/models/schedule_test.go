package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSchedule(t *testing.T) {
	agentID := uuid.New()
	name := "daily-backup"
	cronExpr := "0 2 * * *"
	paths := []string{"/data", "/home"}

	schedule := NewSchedule(agentID, name, cronExpr, paths)

	if schedule.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if schedule.AgentID != agentID {
		t.Errorf("expected AgentID %v, got %v", agentID, schedule.AgentID)
	}
	if schedule.Name != name {
		t.Errorf("expected Name %s, got %s", name, schedule.Name)
	}
	if schedule.CronExpression != cronExpr {
		t.Errorf("expected CronExpression %s, got %s", cronExpr, schedule.CronExpression)
	}
	if len(schedule.Paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(schedule.Paths))
	}
	if !schedule.Enabled {
		t.Error("expected Enabled to be true")
	}
	if schedule.OnMountUnavailable != MountBehaviorFail {
		t.Errorf("expected OnMountUnavailable %s, got %s", MountBehaviorFail, schedule.OnMountUnavailable)
	}
}

func TestSchedule_GetPrimaryRepository(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		schedule := NewSchedule(uuid.New(), "test", "0 * * * *", []string{"/data"})
		schedule.Repositories = []ScheduleRepository{
			{RepositoryID: uuid.New(), Priority: 1, Enabled: true},
			{RepositoryID: uuid.New(), Priority: 0, Enabled: true},
		}

		primary := schedule.GetPrimaryRepository()
		if primary == nil {
			t.Fatal("expected primary repository")
		}
		if primary.Priority != 0 {
			t.Errorf("expected Priority 0, got %d", primary.Priority)
		}
	})

	t.Run("disabled primary", func(t *testing.T) {
		schedule := NewSchedule(uuid.New(), "test", "0 * * * *", []string{"/data"})
		schedule.Repositories = []ScheduleRepository{
			{RepositoryID: uuid.New(), Priority: 0, Enabled: false},
			{RepositoryID: uuid.New(), Priority: 1, Enabled: true},
		}

		primary := schedule.GetPrimaryRepository()
		if primary != nil {
			t.Error("expected nil for disabled primary")
		}
	})

	t.Run("no repos", func(t *testing.T) {
		schedule := NewSchedule(uuid.New(), "test", "0 * * * *", []string{"/data"})
		if primary := schedule.GetPrimaryRepository(); primary != nil {
			t.Error("expected nil for no repos")
		}
	})
}

func TestSchedule_GetEnabledRepositories(t *testing.T) {
	schedule := NewSchedule(uuid.New(), "test", "0 * * * *", []string{"/data"})
	schedule.Repositories = []ScheduleRepository{
		{RepositoryID: uuid.New(), Priority: 2, Enabled: true},
		{RepositoryID: uuid.New(), Priority: 0, Enabled: true},
		{RepositoryID: uuid.New(), Priority: 1, Enabled: false},
		{RepositoryID: uuid.New(), Priority: 3, Enabled: true},
	}

	enabled := schedule.GetEnabledRepositories()
	if len(enabled) != 3 {
		t.Fatalf("expected 3 enabled repos, got %d", len(enabled))
	}

	// Check sorting by priority
	for i := 1; i < len(enabled); i++ {
		if enabled[i-1].Priority > enabled[i].Priority {
			t.Errorf("repos not sorted by priority: %d > %d", enabled[i-1].Priority, enabled[i].Priority)
		}
	}
}

func TestSchedule_PathsJSON(t *testing.T) {
	schedule := NewSchedule(uuid.New(), "test", "0 * * * *", []string{"/data", "/home"})

	data, err := schedule.PathsJSON()
	if err != nil {
		t.Fatalf("PathsJSON failed: %v", err)
	}

	var paths []string
	if err := json.Unmarshal(data, &paths); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(paths))
	}

	t.Run("nil paths", func(t *testing.T) {
		s := &Schedule{}
		data, err := s.PathsJSON()
		if err != nil {
			t.Fatalf("PathsJSON failed: %v", err)
		}
		if string(data) != "[]" {
			t.Errorf("expected [], got %s", string(data))
		}
	})

	t.Run("SetPaths", func(t *testing.T) {
		s := &Schedule{}
		if err := s.SetPaths(data); err != nil {
			t.Fatalf("SetPaths failed: %v", err)
		}
		if len(s.Paths) != 2 {
			t.Errorf("expected 2 paths, got %d", len(s.Paths))
		}
	})

	t.Run("SetPaths empty", func(t *testing.T) {
		s := &Schedule{}
		if err := s.SetPaths(nil); err != nil {
			t.Fatalf("SetPaths(nil) failed: %v", err)
		}
	})
}

func TestSchedule_RetentionPolicy(t *testing.T) {
	policy := DefaultRetentionPolicy()

	if policy.KeepLast != 5 {
		t.Errorf("expected KeepLast 5, got %d", policy.KeepLast)
	}
	if policy.KeepDaily != 7 {
		t.Errorf("expected KeepDaily 7, got %d", policy.KeepDaily)
	}
	if policy.KeepWeekly != 4 {
		t.Errorf("expected KeepWeekly 4, got %d", policy.KeepWeekly)
	}
	if policy.KeepMonthly != 6 {
		t.Errorf("expected KeepMonthly 6, got %d", policy.KeepMonthly)
	}

	t.Run("round trip", func(t *testing.T) {
		s := &Schedule{}
		data, _ := json.Marshal(policy)
		if err := s.SetRetentionPolicy(data); err != nil {
			t.Fatalf("SetRetentionPolicy failed: %v", err)
		}

		retrieved, err := s.RetentionPolicyJSON()
		if err != nil {
			t.Fatalf("RetentionPolicyJSON failed: %v", err)
		}

		var got RetentionPolicy
		if err := json.Unmarshal(retrieved, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if got.KeepLast != policy.KeepLast {
			t.Errorf("expected KeepLast %d, got %d", policy.KeepLast, got.KeepLast)
		}
	})

	t.Run("nil policy", func(t *testing.T) {
		s := &Schedule{}
		data, err := s.RetentionPolicyJSON()
		if err != nil {
			t.Fatalf("RetentionPolicyJSON failed: %v", err)
		}
		if data != nil {
			t.Errorf("expected nil, got %v", data)
		}
	})
}

func TestSchedule_IsWithinBackupWindow(t *testing.T) {
	tests := []struct {
		name     string
		window   *BackupWindow
		time     time.Time
		expected bool
	}{
		{"no window", nil, time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), true},
		{"empty window", &BackupWindow{}, time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), true},
		{"within normal window", &BackupWindow{Start: "02:00", End: "06:00"}, time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC), true},
		{"outside normal window", &BackupWindow{Start: "02:00", End: "06:00"}, time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), false},
		{"within midnight-crossing window before midnight", &BackupWindow{Start: "22:00", End: "06:00"}, time.Date(2024, 1, 1, 23, 0, 0, 0, time.UTC), true},
		{"within midnight-crossing window after midnight", &BackupWindow{Start: "22:00", End: "06:00"}, time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC), true},
		{"outside midnight-crossing window", &BackupWindow{Start: "22:00", End: "06:00"}, time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Schedule{BackupWindow: tt.window}
			if got := s.IsWithinBackupWindow(tt.time); got != tt.expected {
				t.Errorf("IsWithinBackupWindow() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSchedule_IsHourExcluded(t *testing.T) {
	s := &Schedule{ExcludedHours: []int{9, 10, 11, 12, 13, 14, 15, 16, 17}}

	if !s.IsHourExcluded(9) {
		t.Error("expected hour 9 to be excluded")
	}
	if !s.IsHourExcluded(17) {
		t.Error("expected hour 17 to be excluded")
	}
	if s.IsHourExcluded(2) {
		t.Error("expected hour 2 to not be excluded")
	}
	if s.IsHourExcluded(20) {
		t.Error("expected hour 20 to not be excluded")
	}

	t.Run("no excluded hours", func(t *testing.T) {
		s := &Schedule{}
		if s.IsHourExcluded(12) {
			t.Error("expected no hours to be excluded")
		}
	})
}

func TestSchedule_CanRunAt(t *testing.T) {
	s := &Schedule{
		BackupWindow:  &BackupWindow{Start: "02:00", End: "06:00"},
		ExcludedHours: []int{3},
	}

	// Within window, not excluded
	if !s.CanRunAt(time.Date(2024, 1, 1, 4, 0, 0, 0, time.UTC)) {
		t.Error("expected CanRunAt to be true at 04:00")
	}

	// Within window, but excluded
	if s.CanRunAt(time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC)) {
		t.Error("expected CanRunAt to be false at 03:00 (excluded)")
	}

	// Outside window
	if s.CanRunAt(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)) {
		t.Error("expected CanRunAt to be false at 12:00 (outside window)")
	}
}

func TestSchedule_NextAllowedTime(t *testing.T) {
	t.Run("already allowed", func(t *testing.T) {
		s := &Schedule{}
		now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		next := s.NextAllowedTime(now)
		if !next.Equal(now) {
			t.Errorf("expected %v, got %v", now, next)
		}
	})

	t.Run("finds next allowed", func(t *testing.T) {
		s := &Schedule{
			BackupWindow: &BackupWindow{Start: "02:00", End: "06:00"},
		}
		blocked := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		next := s.NextAllowedTime(blocked)
		if !next.After(blocked) {
			t.Errorf("expected next time to be after %v, got %v", blocked, next)
		}
	})
}

func TestSchedule_ExcludesJSON(t *testing.T) {
	t.Run("nil excludes", func(t *testing.T) {
		s := &Schedule{}
		data, err := s.ExcludesJSON()
		if err != nil {
			t.Fatalf("ExcludesJSON failed: %v", err)
		}
		if data != nil {
			t.Errorf("expected nil, got %v", data)
		}
	})

	t.Run("round trip", func(t *testing.T) {
		s := &Schedule{Excludes: []string{"*.tmp", "cache/"}}
		data, err := s.ExcludesJSON()
		if err != nil {
			t.Fatalf("ExcludesJSON failed: %v", err)
		}

		s2 := &Schedule{}
		if err := s2.SetExcludes(data); err != nil {
			t.Fatalf("SetExcludes failed: %v", err)
		}
		if len(s2.Excludes) != 2 {
			t.Errorf("expected 2 excludes, got %d", len(s2.Excludes))
		}
	})
}

func TestSchedule_ExcludedHoursJSON(t *testing.T) {
	t.Run("nil hours", func(t *testing.T) {
		s := &Schedule{}
		data, err := s.ExcludedHoursJSON()
		if err != nil {
			t.Fatalf("ExcludedHoursJSON failed: %v", err)
		}
		if data != nil {
			t.Errorf("expected nil, got %v", data)
		}
	})

	t.Run("round trip", func(t *testing.T) {
		s := &Schedule{ExcludedHours: []int{9, 10, 11}}
		data, err := s.ExcludedHoursJSON()
		if err != nil {
			t.Fatalf("ExcludedHoursJSON failed: %v", err)
		}

		s2 := &Schedule{}
		if err := s2.SetExcludedHours(data); err != nil {
			t.Fatalf("SetExcludedHours failed: %v", err)
		}
		if len(s2.ExcludedHours) != 3 {
			t.Errorf("expected 3 excluded hours, got %d", len(s2.ExcludedHours))
		}
	})
}

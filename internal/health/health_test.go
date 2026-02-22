package health

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDefaultThresholds(t *testing.T) {
	th := DefaultThresholds()

	if th.DiskWarning != 80.0 {
		t.Errorf("expected DiskWarning 80.0, got %f", th.DiskWarning)
	}
	if th.DiskCritical != 90.0 {
		t.Errorf("expected DiskCritical 90.0, got %f", th.DiskCritical)
	}
	if th.MemoryWarning != 85.0 {
		t.Errorf("expected MemoryWarning 85.0, got %f", th.MemoryWarning)
	}
	if th.MemoryCritical != 95.0 {
		t.Errorf("expected MemoryCritical 95.0, got %f", th.MemoryCritical)
	}
	if th.CPUWarning != 80.0 {
		t.Errorf("expected CPUWarning 80.0, got %f", th.CPUWarning)
	}
	if th.CPUCritical != 95.0 {
		t.Errorf("expected CPUCritical 95.0, got %f", th.CPUCritical)
	}
	if th.HeartbeatWarning != 5*time.Minute {
		t.Errorf("expected HeartbeatWarning 5m, got %v", th.HeartbeatWarning)
	}
	if th.HeartbeatCritical != 15*time.Minute {
		t.Errorf("expected HeartbeatCritical 15m, got %v", th.HeartbeatCritical)
	}
}

func TestNewChecker(t *testing.T) {
	th := Thresholds{DiskWarning: 50.0, DiskCritical: 75.0}
	c := NewChecker(th)

	if c == nil {
		t.Fatal("expected non-nil checker")
	}
	if c.thresholds.DiskWarning != 50.0 {
		t.Errorf("expected custom DiskWarning 50.0, got %f", c.thresholds.DiskWarning)
	}
}

func TestNewCheckerWithDefaults(t *testing.T) {
	c := NewCheckerWithDefaults()
	if c == nil {
		t.Fatal("expected non-nil checker")
	}
	if c.thresholds.DiskWarning != 80.0 {
		t.Errorf("expected default DiskWarning 80.0, got %f", c.thresholds.DiskWarning)
	}
}

func TestEvaluateMetrics_NilMetrics(t *testing.T) {
	c := NewCheckerWithDefaults()
	result := c.EvaluateMetrics(nil)

	if result.Status != StatusUnknown {
		t.Errorf("expected StatusUnknown, got %q", result.Status)
	}
	if result.Message != "No metrics available" {
		t.Errorf("expected 'No metrics available', got %q", result.Message)
	}
}

func TestEvaluateMetrics_AllHealthy(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		CPUUsage:        30.0,
		MemoryUsage:     50.0,
		DiskUsage:       40.0,
		DiskFreeBytes:   100_000_000_000,
		DiskTotalBytes:  200_000_000_000,
		NetworkUp:       true,
		UptimeSeconds:   3600,
		ResticVersion:   "0.16.0",
		ResticAvailable: true,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy, got %q", result.Status)
	}
	if result.Message != "All systems operational" {
		t.Errorf("expected 'All systems operational', got %q", result.Message)
	}
	if len(result.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(result.Issues))
	}
	if result.CheckedAt.IsZero() {
		t.Error("expected CheckedAt to be set")
	}
}

func TestEvaluateMetrics_DiskWarning(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		DiskUsage:       85.0,
		NetworkUp:       true,
		ResticAvailable: true,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusWarning {
		t.Errorf("expected StatusWarning, got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Component != "disk" {
		t.Errorf("expected disk issue, got %q", result.Issues[0].Component)
	}
	if result.Issues[0].Severity != StatusWarning {
		t.Errorf("expected warning severity, got %q", result.Issues[0].Severity)
	}
	if result.Issues[0].Value != 85.0 {
		t.Errorf("expected value 85.0, got %f", result.Issues[0].Value)
	}
	if result.Issues[0].Threshold != 80.0 {
		t.Errorf("expected threshold 80.0, got %f", result.Issues[0].Threshold)
	}
}

func TestEvaluateMetrics_DiskCritical(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		DiskUsage:       95.0,
		NetworkUp:       true,
		ResticAvailable: true,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusCritical {
		t.Errorf("expected StatusCritical, got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Component != "disk" {
		t.Errorf("expected disk issue, got %q", result.Issues[0].Component)
	}
	if result.Issues[0].Severity != StatusCritical {
		t.Errorf("expected critical severity, got %q", result.Issues[0].Severity)
	}
}

func TestEvaluateMetrics_MemoryWarning(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		MemoryUsage:     90.0,
		NetworkUp:       true,
		ResticAvailable: true,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusWarning {
		t.Errorf("expected StatusWarning, got %q", result.Status)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Component == "memory" && issue.Severity == StatusWarning {
			found = true
			if issue.Value != 90.0 {
				t.Errorf("expected memory value 90.0, got %f", issue.Value)
			}
		}
	}
	if !found {
		t.Error("expected memory warning issue")
	}
}

func TestEvaluateMetrics_MemoryCritical(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		MemoryUsage:     96.0,
		NetworkUp:       true,
		ResticAvailable: true,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusCritical {
		t.Errorf("expected StatusCritical, got %q", result.Status)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Component == "memory" && issue.Severity == StatusCritical {
			found = true
		}
	}
	if !found {
		t.Error("expected memory critical issue")
	}
}

func TestEvaluateMetrics_CPUWarning(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		CPUUsage:        85.0,
		NetworkUp:       true,
		ResticAvailable: true,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusWarning {
		t.Errorf("expected StatusWarning, got %q", result.Status)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Component == "cpu" && issue.Severity == StatusWarning {
			found = true
		}
	}
	if !found {
		t.Error("expected cpu warning issue")
	}
}

func TestEvaluateMetrics_CPUCritical(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		CPUUsage:        96.0,
		NetworkUp:       true,
		ResticAvailable: true,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusCritical {
		t.Errorf("expected StatusCritical, got %q", result.Status)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Component == "cpu" && issue.Severity == StatusCritical {
			found = true
		}
	}
	if !found {
		t.Error("expected cpu critical issue")
	}
}

func TestEvaluateMetrics_NetworkDown(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		NetworkUp:       false,
		ResticAvailable: true,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusWarning {
		t.Errorf("expected StatusWarning, got %q", result.Status)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Component == "network" {
			found = true
			if issue.Severity != StatusWarning {
				t.Errorf("expected warning severity for network, got %q", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("expected network issue")
	}
}

func TestEvaluateMetrics_ResticUnavailable(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		NetworkUp:       true,
		ResticAvailable: false,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusWarning {
		t.Errorf("expected StatusWarning, got %q", result.Status)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Component == "restic" {
			found = true
			if issue.Severity != StatusWarning {
				t.Errorf("expected warning severity for restic, got %q", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("expected restic issue")
	}
}

func TestEvaluateMetrics_MultipleIssues(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		CPUUsage:        96.0, // critical
		MemoryUsage:     90.0, // warning
		DiskUsage:       85.0, // warning
		NetworkUp:       false,
		ResticAvailable: false,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusCritical {
		t.Errorf("expected StatusCritical (critical trumps warning), got %q", result.Status)
	}
	if len(result.Issues) != 5 {
		t.Errorf("expected 5 issues, got %d", len(result.Issues))
	}
	if result.Message != "Critical issues detected" {
		t.Errorf("expected 'Critical issues detected', got %q", result.Message)
	}
}

func TestEvaluateMetrics_BoundaryValues(t *testing.T) {
	c := NewCheckerWithDefaults()

	t.Run("exactly at warning threshold", func(t *testing.T) {
		m := &Metrics{
			DiskUsage:       80.0,
			NetworkUp:       true,
			ResticAvailable: true,
		}
		result := c.EvaluateMetrics(m)
		if result.Status != StatusWarning {
			t.Errorf("expected StatusWarning at exactly 80%%, got %q", result.Status)
		}
	})

	t.Run("just below warning threshold", func(t *testing.T) {
		m := &Metrics{
			DiskUsage:       79.9,
			NetworkUp:       true,
			ResticAvailable: true,
		}
		result := c.EvaluateMetrics(m)
		if result.Status != StatusHealthy {
			t.Errorf("expected StatusHealthy below 80%%, got %q", result.Status)
		}
	})

	t.Run("exactly at critical threshold", func(t *testing.T) {
		m := &Metrics{
			DiskUsage:       90.0,
			NetworkUp:       true,
			ResticAvailable: true,
		}
		result := c.EvaluateMetrics(m)
		if result.Status != StatusCritical {
			t.Errorf("expected StatusCritical at exactly 90%%, got %q", result.Status)
		}
	})

	t.Run("just below critical threshold", func(t *testing.T) {
		m := &Metrics{
			DiskUsage:       89.9,
			NetworkUp:       true,
			ResticAvailable: true,
		}
		result := c.EvaluateMetrics(m)
		if result.Status != StatusWarning {
			t.Errorf("expected StatusWarning just below 90%%, got %q", result.Status)
		}
	})
}

func TestEvaluateWithHeartbeat_RecentHeartbeat(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		NetworkUp:       true,
		ResticAvailable: true,
	}
	recent := time.Now().Add(-1 * time.Minute)

	result := c.EvaluateWithHeartbeat(m, &recent)

	if result.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy for recent heartbeat, got %q", result.Status)
	}
	// Should have no heartbeat issues
	for _, issue := range result.Issues {
		if issue.Component == "heartbeat" {
			t.Error("unexpected heartbeat issue for recent heartbeat")
		}
	}
}

func TestEvaluateWithHeartbeat_WarningHeartbeat(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		NetworkUp:       true,
		ResticAvailable: true,
	}
	delayed := time.Now().Add(-10 * time.Minute)

	result := c.EvaluateWithHeartbeat(m, &delayed)

	if result.Status != StatusWarning {
		t.Errorf("expected StatusWarning for delayed heartbeat, got %q", result.Status)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Component == "heartbeat" && issue.Severity == StatusWarning {
			found = true
		}
	}
	if !found {
		t.Error("expected heartbeat warning issue")
	}
	if result.MetricsStale {
		t.Error("expected MetricsStale=false for warning-level heartbeat")
	}
}

func TestEvaluateWithHeartbeat_CriticalHeartbeat(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		NetworkUp:       true,
		ResticAvailable: true,
	}
	stale := time.Now().Add(-20 * time.Minute)

	result := c.EvaluateWithHeartbeat(m, &stale)

	if result.Status != StatusCritical {
		t.Errorf("expected StatusCritical for stale heartbeat, got %q", result.Status)
	}
	if !result.MetricsStale {
		t.Error("expected MetricsStale=true for critical heartbeat")
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Component == "heartbeat" && issue.Severity == StatusCritical {
			found = true
		}
	}
	if !found {
		t.Error("expected heartbeat critical issue")
	}
}

func TestEvaluateWithHeartbeat_NilLastSeen(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		NetworkUp:       true,
		ResticAvailable: true,
	}

	result := c.EvaluateWithHeartbeat(m, nil)

	if result.Status != StatusWarning {
		t.Errorf("expected StatusWarning for nil lastSeen, got %q", result.Status)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Component == "heartbeat" && issue.Message == "Agent has never reported" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'Agent has never reported' heartbeat issue")
	}
}

func TestEvaluateWithHeartbeat_NilMetrics(t *testing.T) {
	c := NewCheckerWithDefaults()
	stale := time.Now().Add(-20 * time.Minute)

	result := c.EvaluateWithHeartbeat(nil, &stale)

	// nil metrics returns early with StatusUnknown, heartbeat issues still appended
	if result.Status != StatusCritical {
		t.Errorf("expected StatusCritical (nil metrics + stale heartbeat), got %q", result.Status)
	}
}

func TestEvaluateWithHeartbeat_ExactThresholdBoundary(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		NetworkUp:       true,
		ResticAvailable: true,
	}

	t.Run("exactly at warning threshold", func(t *testing.T) {
		delayed := time.Now().Add(-5 * time.Minute)
		result := c.EvaluateWithHeartbeat(m, &delayed)
		found := false
		for _, issue := range result.Issues {
			if issue.Component == "heartbeat" {
				found = true
			}
		}
		if !found {
			t.Error("expected heartbeat issue at exactly 5 minutes")
		}
	})

	t.Run("exactly at critical threshold", func(t *testing.T) {
		stale := time.Now().Add(-15 * time.Minute)
		result := c.EvaluateWithHeartbeat(m, &stale)
		if !result.MetricsStale {
			t.Error("expected MetricsStale=true at exactly 15 minutes")
		}
	})
}

func TestDetermineOverallStatus(t *testing.T) {
	c := NewCheckerWithDefaults()

	t.Run("no issues returns healthy", func(t *testing.T) {
		status := c.determineOverallStatus([]Issue{})
		if status != StatusHealthy {
			t.Errorf("expected StatusHealthy, got %q", status)
		}
	})

	t.Run("only warnings returns warning", func(t *testing.T) {
		issues := []Issue{
			{Severity: StatusWarning},
			{Severity: StatusWarning},
		}
		status := c.determineOverallStatus(issues)
		if status != StatusWarning {
			t.Errorf("expected StatusWarning, got %q", status)
		}
	})

	t.Run("critical trumps warning", func(t *testing.T) {
		issues := []Issue{
			{Severity: StatusWarning},
			{Severity: StatusCritical},
		}
		status := c.determineOverallStatus(issues)
		if status != StatusCritical {
			t.Errorf("expected StatusCritical, got %q", status)
		}
	})

	t.Run("single critical returns critical", func(t *testing.T) {
		issues := []Issue{
			{Severity: StatusCritical},
		}
		status := c.determineOverallStatus(issues)
		if status != StatusCritical {
			t.Errorf("expected StatusCritical, got %q", status)
		}
	})

	t.Run("unknown severity treated as healthy", func(t *testing.T) {
		issues := []Issue{
			{Severity: StatusUnknown},
		}
		status := c.determineOverallStatus(issues)
		if status != StatusHealthy {
			t.Errorf("expected StatusHealthy for unknown-only severity, got %q", status)
		}
	})
}

func TestGenerateMessage(t *testing.T) {
	c := NewCheckerWithDefaults()

	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{StatusHealthy, "All systems operational"},
		{StatusWarning, "Some metrics require attention"},
		{StatusCritical, "Critical issues detected"},
		{StatusUnknown, "Health status unknown"},
		{"invalid", "Health status unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := &CheckResult{Status: tt.status}
			msg := c.generateMessage(result)
			if msg != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, msg)
			}
		})
	}
}

func TestGetStatusColor(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{StatusHealthy, "green"},
		{StatusWarning, "yellow"},
		{StatusCritical, "red"},
		{StatusUnknown, "gray"},
		{"other", "gray"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			color := getStatusColor(tt.status)
			if color != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, color)
			}
		})
	}
}

func TestNewCollector(t *testing.T) {
	c := NewCollector("https://example.com", "/usr/bin/restic")

	if c == nil {
		t.Fatal("expected non-nil collector")
	}
	if c.serverURL != "https://example.com" {
		t.Errorf("expected serverURL 'https://example.com', got %q", c.serverURL)
	}
	if c.resticBinary != "/usr/bin/restic" {
		t.Errorf("expected resticBinary '/usr/bin/restic', got %q", c.resticBinary)
	}
	if c.startTime.IsZero() {
		t.Error("expected startTime to be set")
	}
}

func TestCollect(t *testing.T) {
	c := NewCollector("https://example.com", "nonexistent-restic-binary")
	ctx := context.Background()

	m, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil metrics")
	}
	if m.UptimeSeconds < 0 {
		t.Errorf("expected non-negative uptime, got %d", m.UptimeSeconds)
	}
	// Restic should not be available with a nonexistent binary
	if m.ResticAvailable {
		t.Error("expected ResticAvailable=false for nonexistent binary")
	}
}

func TestCollect_CanceledContext(t *testing.T) {
	c := NewCollector("https://example.com", "restic")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	m, err := c.Collect(ctx)

	// Collect does not return errors from individual checks, just partial metrics
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil metrics even with canceled context")
	}
}

func TestCollect_Timeout(t *testing.T) {
	c := NewCollector("https://example.com", "restic")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	// Let the timeout expire
	time.Sleep(1 * time.Millisecond)

	m, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil metrics even with expired timeout")
	}
}

func TestCheckNetworkConnectivity_EmptyServerURL(t *testing.T) {
	c := NewCollector("", "restic")
	ctx := context.Background()

	up := c.checkNetworkConnectivity(ctx)

	if up {
		t.Error("expected false for empty server URL")
	}
}

func TestCheckNetworkConnectivity_WithServerURL(t *testing.T) {
	c := NewCollector("https://example.com", "restic")
	ctx := context.Background()

	// This depends on the test machine having network interfaces,
	// but it should not panic or error
	_ = c.checkNetworkConnectivity(ctx)
}

func TestGetResticVersion_EmptyBinary(t *testing.T) {
	c := NewCollector("https://example.com", "")
	ctx := context.Background()

	// With empty binary, it defaults to "restic" and looks in PATH
	// On most CI/dev machines, restic is not installed
	version, available := c.getResticVersion(ctx)
	_ = version
	_ = available
	// Just verifying it doesn't panic
}

func TestGetResticVersion_NonexistentBinary(t *testing.T) {
	c := NewCollector("https://example.com", "nonexistent-binary-xyz")
	ctx := context.Background()

	version, available := c.getResticVersion(ctx)

	if available {
		t.Error("expected ResticAvailable=false for nonexistent binary")
	}
	if version != "" {
		t.Errorf("expected empty version, got %q", version)
	}
}

func TestGetResticVersion_MockBinary(t *testing.T) {
	// Create a temporary script that mimics restic version output
	dir := t.TempDir()
	fakeRestic := dir + "/restic"
	err := os.WriteFile(fakeRestic, []byte("#!/bin/sh\necho 'restic 0.16.0 compiled with go1.21.0 on linux/amd64'\n"), 0o755)
	if err != nil {
		t.Fatalf("failed to create mock binary: %v", err)
	}

	c := NewCollector("https://example.com", fakeRestic)
	ctx := context.Background()

	version, available := c.getResticVersion(ctx)

	if !available {
		t.Error("expected ResticAvailable=true for mock binary")
	}
	if version != "0.16.0" {
		t.Errorf("expected version '0.16.0', got %q", version)
	}
}

func TestGetResticVersion_MockBinarySingleWord(t *testing.T) {
	// Test parsing when output is a single word (less than 2 fields)
	dir := t.TempDir()
	fakeRestic := dir + "/restic"
	err := os.WriteFile(fakeRestic, []byte("#!/bin/sh\necho 'v1.0'\n"), 0o755)
	if err != nil {
		t.Fatalf("failed to create mock binary: %v", err)
	}

	c := NewCollector("https://example.com", fakeRestic)
	ctx := context.Background()

	version, available := c.getResticVersion(ctx)

	if !available {
		t.Error("expected ResticAvailable=true")
	}
	if version != "v1.0" {
		t.Errorf("expected version 'v1.0', got %q", version)
	}
}

func TestGetResticVersion_MockBinaryError(t *testing.T) {
	// Test binary that exits with error
	dir := t.TempDir()
	fakeRestic := dir + "/restic"
	err := os.WriteFile(fakeRestic, []byte("#!/bin/sh\nexit 1\n"), 0o755)
	if err != nil {
		t.Fatalf("failed to create mock binary: %v", err)
	}

	c := NewCollector("https://example.com", fakeRestic)
	ctx := context.Background()

	version, available := c.getResticVersion(ctx)

	if available {
		t.Error("expected ResticAvailable=false for erroring binary")
	}
	if version != "" {
		t.Errorf("expected empty version, got %q", version)
	}
}

func TestGetOSVersion(t *testing.T) {
	// getOSVersion is unexported but called via GetOSInfo
	v := getOSVersion()
	if v == "" {
		t.Error("expected non-empty OS version")
	}
}

func TestCollect_WithMockRestic(t *testing.T) {
	dir := t.TempDir()
	fakeRestic := dir + "/restic"
	err := os.WriteFile(fakeRestic, []byte("#!/bin/sh\necho 'restic 0.16.0 compiled with go1.21.0'\n"), 0o755)
	if err != nil {
		t.Fatalf("failed to create mock binary: %v", err)
	}

	c := NewCollector("https://example.com", fakeRestic)
	ctx := context.Background()

	m, collectErr := c.Collect(ctx)
	if collectErr != nil {
		t.Fatalf("Collect() returned error: %v", collectErr)
	}
	if !m.ResticAvailable {
		t.Error("expected ResticAvailable=true with mock binary")
	}
	if m.ResticVersion != "0.16.0" {
		t.Errorf("expected ResticVersion '0.16.0', got %q", m.ResticVersion)
	}
}

func TestGetOSInfo(t *testing.T) {
	info := GetOSInfo()

	if info["os"] == "" {
		t.Error("expected non-empty os")
	}
	if info["arch"] == "" {
		t.Error("expected non-empty arch")
	}
	// Hostname could be empty in some environments, but key should exist
	if _, ok := info["hostname"]; !ok {
		t.Error("expected hostname key")
	}
	if _, ok := info["version"]; !ok {
		t.Error("expected version key")
	}
}

func TestCustomThresholds(t *testing.T) {
	th := Thresholds{
		DiskWarning:       50.0,
		DiskCritical:      70.0,
		MemoryWarning:     60.0,
		MemoryCritical:    80.0,
		CPUWarning:        50.0,
		CPUCritical:       75.0,
		HeartbeatWarning:  2 * time.Minute,
		HeartbeatCritical: 5 * time.Minute,
	}
	c := NewChecker(th)

	t.Run("disk warning at custom threshold", func(t *testing.T) {
		m := &Metrics{
			DiskUsage:       55.0,
			NetworkUp:       true,
			ResticAvailable: true,
		}
		result := c.EvaluateMetrics(m)
		if result.Status != StatusWarning {
			t.Errorf("expected warning at 55%% with 50%% threshold, got %q", result.Status)
		}
	})

	t.Run("disk critical at custom threshold", func(t *testing.T) {
		m := &Metrics{
			DiskUsage:       75.0,
			NetworkUp:       true,
			ResticAvailable: true,
		}
		result := c.EvaluateMetrics(m)
		if result.Status != StatusCritical {
			t.Errorf("expected critical at 75%% with 70%% threshold, got %q", result.Status)
		}
	})

	t.Run("heartbeat warning at custom threshold", func(t *testing.T) {
		m := &Metrics{
			NetworkUp:       true,
			ResticAvailable: true,
		}
		delayed := time.Now().Add(-3 * time.Minute)
		result := c.EvaluateWithHeartbeat(m, &delayed)
		found := false
		for _, issue := range result.Issues {
			if issue.Component == "heartbeat" && issue.Severity == StatusWarning {
				found = true
			}
		}
		if !found {
			t.Error("expected heartbeat warning at 3m with 2m threshold")
		}
	})

	t.Run("heartbeat critical at custom threshold", func(t *testing.T) {
		m := &Metrics{
			NetworkUp:       true,
			ResticAvailable: true,
		}
		stale := time.Now().Add(-6 * time.Minute)
		result := c.EvaluateWithHeartbeat(m, &stale)
		if !result.MetricsStale {
			t.Error("expected MetricsStale=true at 6m with 5m threshold")
		}
	})
}

func TestEvaluateMetrics_AllComponentsCritical(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		CPUUsage:        99.0,
		MemoryUsage:     99.0,
		DiskUsage:       99.0,
		NetworkUp:       false,
		ResticAvailable: false,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusCritical {
		t.Errorf("expected StatusCritical, got %q", result.Status)
	}
	// 3 critical (cpu, mem, disk) + 2 warning (network, restic) = 5 issues
	if len(result.Issues) != 5 {
		t.Errorf("expected 5 issues, got %d", len(result.Issues))
	}

	criticalCount := 0
	warningCount := 0
	for _, issue := range result.Issues {
		switch issue.Severity {
		case StatusCritical:
			criticalCount++
		case StatusWarning:
			warningCount++
		}
	}
	if criticalCount != 3 {
		t.Errorf("expected 3 critical issues, got %d", criticalCount)
	}
	if warningCount != 2 {
		t.Errorf("expected 2 warning issues, got %d", warningCount)
	}
}

func TestEvaluateWithHeartbeat_CombinedMetricsAndHeartbeat(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		DiskUsage:       85.0, // warning
		NetworkUp:       true,
		ResticAvailable: true,
	}
	stale := time.Now().Add(-20 * time.Minute) // critical

	result := c.EvaluateWithHeartbeat(m, &stale)

	if result.Status != StatusCritical {
		t.Errorf("expected StatusCritical (heartbeat critical trumps disk warning), got %q", result.Status)
	}
	if !result.MetricsStale {
		t.Error("expected MetricsStale=true")
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues (disk warning + heartbeat critical), got %d", len(result.Issues))
	}
}

func TestEvaluateMetrics_ZeroValues(t *testing.T) {
	c := NewCheckerWithDefaults()
	m := &Metrics{
		CPUUsage:        0,
		MemoryUsage:     0,
		DiskUsage:       0,
		NetworkUp:       true,
		ResticAvailable: true,
	}

	result := c.EvaluateMetrics(m)

	if result.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy for zero usage values, got %q", result.Status)
	}
	if len(result.Issues) != 0 {
		t.Errorf("expected 0 issues for zero values, got %d", len(result.Issues))
	}
}

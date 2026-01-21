package health

import (
	"time"
)

// HealthStatus represents the overall health status of an agent.
type HealthStatus string

const (
	// StatusHealthy indicates all metrics are within acceptable ranges.
	StatusHealthy HealthStatus = "healthy"
	// StatusWarning indicates some metrics are concerning but not critical.
	StatusWarning HealthStatus = "warning"
	// StatusCritical indicates immediate attention is required.
	StatusCritical HealthStatus = "critical"
	// StatusUnknown indicates health cannot be determined.
	StatusUnknown HealthStatus = "unknown"
)

// Thresholds defines the thresholds for health evaluation.
type Thresholds struct {
	// Disk thresholds (percentage used)
	DiskWarning  float64 // Default: 80%
	DiskCritical float64 // Default: 90%

	// Memory thresholds (percentage used)
	MemoryWarning  float64 // Default: 85%
	MemoryCritical float64 // Default: 95%

	// CPU thresholds (percentage used)
	CPUWarning  float64 // Default: 80%
	CPUCritical float64 // Default: 95%

	// Heartbeat thresholds
	HeartbeatWarning  time.Duration // Default: 5 minutes
	HeartbeatCritical time.Duration // Default: 15 minutes
}

// DefaultThresholds returns the default health thresholds.
func DefaultThresholds() Thresholds {
	return Thresholds{
		DiskWarning:       80.0,
		DiskCritical:      90.0,
		MemoryWarning:     85.0,
		MemoryCritical:    95.0,
		CPUWarning:        80.0,
		CPUCritical:       95.0,
		HeartbeatWarning:  5 * time.Minute,
		HeartbeatCritical: 15 * time.Minute,
	}
}

// CheckResult contains the detailed health check result.
type CheckResult struct {
	Status       HealthStatus `json:"status"`
	Message      string       `json:"message"`
	Issues       []Issue      `json:"issues,omitempty"`
	CheckedAt    time.Time    `json:"checked_at"`
	MetricsStale bool         `json:"metrics_stale"`
}

// Issue represents a specific health issue.
type Issue struct {
	Component string       `json:"component"` // disk, memory, cpu, network, restic, heartbeat
	Severity  HealthStatus `json:"severity"`
	Message   string       `json:"message"`
	Value     float64      `json:"value,omitempty"`
	Threshold float64      `json:"threshold,omitempty"`
}

// Checker evaluates agent health based on metrics.
type Checker struct {
	thresholds Thresholds
}

// NewChecker creates a new health checker with the given thresholds.
func NewChecker(thresholds Thresholds) *Checker {
	return &Checker{thresholds: thresholds}
}

// NewCheckerWithDefaults creates a new health checker with default thresholds.
func NewCheckerWithDefaults() *Checker {
	return NewChecker(DefaultThresholds())
}

// EvaluateMetrics evaluates health based on current metrics.
func (c *Checker) EvaluateMetrics(m *Metrics) *CheckResult {
	result := &CheckResult{
		Status:    StatusHealthy,
		CheckedAt: time.Now(),
		Issues:    make([]Issue, 0),
	}

	if m == nil {
		result.Status = StatusUnknown
		result.Message = "No metrics available"
		return result
	}

	// Check disk usage
	if m.DiskUsage >= c.thresholds.DiskCritical {
		result.Issues = append(result.Issues, Issue{
			Component: "disk",
			Severity:  StatusCritical,
			Message:   "Disk space critically low",
			Value:     m.DiskUsage,
			Threshold: c.thresholds.DiskCritical,
		})
	} else if m.DiskUsage >= c.thresholds.DiskWarning {
		result.Issues = append(result.Issues, Issue{
			Component: "disk",
			Severity:  StatusWarning,
			Message:   "Disk space running low",
			Value:     m.DiskUsage,
			Threshold: c.thresholds.DiskWarning,
		})
	}

	// Check memory usage
	if m.MemoryUsage >= c.thresholds.MemoryCritical {
		result.Issues = append(result.Issues, Issue{
			Component: "memory",
			Severity:  StatusCritical,
			Message:   "Memory usage critically high",
			Value:     m.MemoryUsage,
			Threshold: c.thresholds.MemoryCritical,
		})
	} else if m.MemoryUsage >= c.thresholds.MemoryWarning {
		result.Issues = append(result.Issues, Issue{
			Component: "memory",
			Severity:  StatusWarning,
			Message:   "Memory usage high",
			Value:     m.MemoryUsage,
			Threshold: c.thresholds.MemoryWarning,
		})
	}

	// Check CPU usage
	if m.CPUUsage >= c.thresholds.CPUCritical {
		result.Issues = append(result.Issues, Issue{
			Component: "cpu",
			Severity:  StatusCritical,
			Message:   "CPU usage critically high",
			Value:     m.CPUUsage,
			Threshold: c.thresholds.CPUCritical,
		})
	} else if m.CPUUsage >= c.thresholds.CPUWarning {
		result.Issues = append(result.Issues, Issue{
			Component: "cpu",
			Severity:  StatusWarning,
			Message:   "CPU usage high",
			Value:     m.CPUUsage,
			Threshold: c.thresholds.CPUWarning,
		})
	}

	// Check network connectivity
	if !m.NetworkUp {
		result.Issues = append(result.Issues, Issue{
			Component: "network",
			Severity:  StatusWarning,
			Message:   "Network connectivity issues detected",
		})
	}

	// Check restic availability
	if !m.ResticAvailable {
		result.Issues = append(result.Issues, Issue{
			Component: "restic",
			Severity:  StatusWarning,
			Message:   "Restic binary not available",
		})
	}

	// Determine overall status
	result.Status = c.determineOverallStatus(result.Issues)
	result.Message = c.generateMessage(result)

	return result
}

// EvaluateWithHeartbeat evaluates health including heartbeat timing.
func (c *Checker) EvaluateWithHeartbeat(m *Metrics, lastSeen *time.Time) *CheckResult {
	result := c.EvaluateMetrics(m)

	if lastSeen != nil {
		timeSinceSeen := time.Since(*lastSeen)

		if timeSinceSeen >= c.thresholds.HeartbeatCritical {
			result.Issues = append(result.Issues, Issue{
				Component: "heartbeat",
				Severity:  StatusCritical,
				Message:   "Agent has not reported in over 15 minutes",
				Value:     timeSinceSeen.Minutes(),
				Threshold: c.thresholds.HeartbeatCritical.Minutes(),
			})
			result.MetricsStale = true
		} else if timeSinceSeen >= c.thresholds.HeartbeatWarning {
			result.Issues = append(result.Issues, Issue{
				Component: "heartbeat",
				Severity:  StatusWarning,
				Message:   "Agent heartbeat delayed",
				Value:     timeSinceSeen.Minutes(),
				Threshold: c.thresholds.HeartbeatWarning.Minutes(),
			})
		}
	} else {
		result.Issues = append(result.Issues, Issue{
			Component: "heartbeat",
			Severity:  StatusWarning,
			Message:   "Agent has never reported",
		})
	}

	// Recalculate overall status
	result.Status = c.determineOverallStatus(result.Issues)
	result.Message = c.generateMessage(result)

	return result
}

// determineOverallStatus determines the overall health status from issues.
func (c *Checker) determineOverallStatus(issues []Issue) HealthStatus {
	if len(issues) == 0 {
		return StatusHealthy
	}

	hasCritical := false
	hasWarning := false

	for _, issue := range issues {
		switch issue.Severity {
		case StatusCritical:
			hasCritical = true
		case StatusWarning:
			hasWarning = true
		}
	}

	if hasCritical {
		return StatusCritical
	}
	if hasWarning {
		return StatusWarning
	}
	return StatusHealthy
}

// generateMessage generates a human-readable status message.
func (c *Checker) generateMessage(result *CheckResult) string {
	switch result.Status {
	case StatusHealthy:
		return "All systems operational"
	case StatusWarning:
		return "Some metrics require attention"
	case StatusCritical:
		return "Critical issues detected"
	default:
		return "Health status unknown"
	}
}

// GetStatusColor returns a color code for the health status.
func GetStatusColor(status HealthStatus) string {
	switch status {
	case StatusHealthy:
		return "green"
	case StatusWarning:
		return "yellow"
	case StatusCritical:
		return "red"
	default:
		return "gray"
	}
}

// Package logs provides server log capture and retrieval functionality.
package logs

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// LogEntry represents a single structured log entry.
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// LogFilter specifies criteria for filtering log entries.
type LogFilter struct {
	Level      LogLevel  `json:"level,omitempty"`
	Component  string    `json:"component,omitempty"`
	Search     string    `json:"search,omitempty"`
	StartTime  time.Time `json:"start_time,omitempty"`
	EndTime    time.Time `json:"end_time,omitempty"`
	Limit      int       `json:"limit,omitempty"`
	Offset     int       `json:"offset,omitempty"`
}

// LogBuffer is a thread-safe ring buffer for storing log entries.
type LogBuffer struct {
	mu           sync.RWMutex
	entries      []LogEntry
	maxEntries   int
	writePos     int
	isFull       bool
	retentionDur time.Duration
}

// Config holds the configuration for the log buffer.
type Config struct {
	// MaxEntries is the maximum number of log entries to keep in the buffer.
	MaxEntries int
	// RetentionDuration is how long to keep log entries before they expire.
	RetentionDuration time.Duration
}

// DefaultConfig returns sensible defaults for the log buffer.
func DefaultConfig() Config {
	return Config{
		MaxEntries:        10000,
		RetentionDuration: 24 * time.Hour,
	}
}

// NewLogBuffer creates a new log buffer with the given configuration.
func NewLogBuffer(cfg Config) *LogBuffer {
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = DefaultConfig().MaxEntries
	}
	if cfg.RetentionDuration <= 0 {
		cfg.RetentionDuration = DefaultConfig().RetentionDuration
	}

	return &LogBuffer{
		entries:      make([]LogEntry, cfg.MaxEntries),
		maxEntries:   cfg.MaxEntries,
		retentionDur: cfg.RetentionDuration,
	}
}

// Write implements io.Writer for zerolog integration.
// It parses JSON log entries from zerolog and stores them in the buffer.
func (lb *LogBuffer) Write(p []byte) (n int, err error) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Fields:    make(map[string]interface{}),
	}

	// Parse the JSON log entry from zerolog
	var raw map[string]interface{}
	if err := json.Unmarshal(p, &raw); err != nil {
		// If we can't parse it, store as raw message
		entry.Message = string(p)
		entry.Level = LogLevelInfo
	} else {
		// Extract standard fields
		if level, ok := raw["level"].(string); ok {
			entry.Level = LogLevel(level)
			delete(raw, "level")
		}
		if msg, ok := raw["message"].(string); ok {
			entry.Message = msg
			delete(raw, "message")
		}
		if comp, ok := raw["component"].(string); ok {
			entry.Component = comp
			delete(raw, "component")
		}
		if ts, ok := raw["time"].(float64); ok {
			entry.Timestamp = time.Unix(int64(ts), 0)
			delete(raw, "time")
		}
		// Store remaining fields
		entry.Fields = raw
	}

	lb.Add(entry)
	return len(p), nil
}

// Add adds a new log entry to the buffer.
func (lb *LogBuffer) Add(entry LogEntry) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.entries[lb.writePos] = entry
	lb.writePos = (lb.writePos + 1) % lb.maxEntries
	if lb.writePos == 0 {
		lb.isFull = true
	}
}

// GetAll returns all log entries in the buffer, newest first.
func (lb *LogBuffer) GetAll() []LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	return lb.getEntriesLocked()
}

// Get returns log entries matching the given filter.
func (lb *LogBuffer) Get(filter LogFilter) ([]LogEntry, int) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	allEntries := lb.getEntriesLocked()

	// Filter entries
	var filtered []LogEntry
	cutoffTime := time.Now().Add(-lb.retentionDur)

	for _, entry := range allEntries {
		// Check retention
		if entry.Timestamp.Before(cutoffTime) {
			continue
		}

		// Check level filter
		if filter.Level != "" && !lb.levelMatches(entry.Level, filter.Level) {
			continue
		}

		// Check component filter
		if filter.Component != "" && entry.Component != filter.Component {
			continue
		}

		// Check time range
		if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
			continue
		}

		// Check search filter
		if filter.Search != "" && !lb.entryContains(entry, filter.Search) {
			continue
		}

		filtered = append(filtered, entry)
	}

	totalCount := len(filtered)

	// Apply pagination
	if filter.Offset > 0 {
		if filter.Offset >= len(filtered) {
			return []LogEntry{}, totalCount
		}
		filtered = filtered[filter.Offset:]
	}

	if filter.Limit > 0 && filter.Limit < len(filtered) {
		filtered = filtered[:filter.Limit]
	}

	return filtered, totalCount
}

// getEntriesLocked returns entries in order, newest first.
// Must be called with read lock held.
func (lb *LogBuffer) getEntriesLocked() []LogEntry {
	var result []LogEntry

	if lb.isFull {
		// Buffer has wrapped around
		// Read from writePos to end, then from start to writePos-1
		for i := lb.writePos - 1; i >= 0; i-- {
			if lb.entries[i].Timestamp.IsZero() {
				continue
			}
			result = append(result, lb.entries[i])
		}
		for i := lb.maxEntries - 1; i >= lb.writePos; i-- {
			if lb.entries[i].Timestamp.IsZero() {
				continue
			}
			result = append(result, lb.entries[i])
		}
	} else {
		// Buffer hasn't filled yet
		for i := lb.writePos - 1; i >= 0; i-- {
			if lb.entries[i].Timestamp.IsZero() {
				continue
			}
			result = append(result, lb.entries[i])
		}
	}

	return result
}

// levelMatches returns true if the entry level is at or above the filter level.
func (lb *LogBuffer) levelMatches(entryLevel, filterLevel LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
		LogLevelFatal: 4,
	}

	entryPriority := levels[entryLevel]
	filterPriority := levels[filterLevel]

	return entryPriority >= filterPriority
}

// entryContains checks if the log entry contains the search string.
func (lb *LogBuffer) entryContains(entry LogEntry, search string) bool {
	// Check message
	if containsIgnoreCase(entry.Message, search) {
		return true
	}

	// Check component
	if containsIgnoreCase(entry.Component, search) {
		return true
	}

	// Check fields
	for _, v := range entry.Fields {
		if s, ok := v.(string); ok && containsIgnoreCase(s, search) {
			return true
		}
	}

	return false
}

// containsIgnoreCase performs case-insensitive string matching.
func containsIgnoreCase(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			uc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if uc >= 'A' && uc <= 'Z' {
				uc += 32
			}
			if sc != uc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// GetComponents returns a list of unique component names in the buffer.
func (lb *LogBuffer) GetComponents() []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	componentSet := make(map[string]struct{})
	entries := lb.getEntriesLocked()

	for _, entry := range entries {
		if entry.Component != "" {
			componentSet[entry.Component] = struct{}{}
		}
	}

	components := make([]string, 0, len(componentSet))
	for comp := range componentSet {
		components = append(components, comp)
	}

	return components
}

// Clear removes all entries from the buffer.
func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.entries = make([]LogEntry, lb.maxEntries)
	lb.writePos = 0
	lb.isFull = false
}

// Count returns the current number of entries in the buffer.
func (lb *LogBuffer) Count() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if lb.isFull {
		return lb.maxEntries
	}
	return lb.writePos
}

// MultiWriter creates an io.Writer that writes to multiple destinations.
func MultiWriter(writers ...io.Writer) io.Writer {
	return io.MultiWriter(writers...)
}

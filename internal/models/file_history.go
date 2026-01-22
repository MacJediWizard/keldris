package models

import (
	"time"
)

// FileVersion represents a single version of a file in a snapshot.
type FileVersion struct {
	SnapshotID   string    `json:"snapshot_id"`
	SnapshotTime time.Time `json:"snapshot_time"`
	FilePath     string    `json:"file_path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	Mode         uint32    `json:"mode"`
}

// FileHistory contains the complete history of a file across all snapshots.
type FileHistory struct {
	FilePath     string        `json:"file_path"`
	AgentID      string        `json:"agent_id"`
	RepositoryID string        `json:"repository_id"`
	Versions     []FileVersion `json:"versions"`
}

// FileHistoryRequest represents a request to get file history.
type FileHistoryRequest struct {
	Path         string `json:"path" binding:"required"`
	AgentID      string `json:"agent_id" binding:"required"`
	RepositoryID string `json:"repository_id" binding:"required"`
}

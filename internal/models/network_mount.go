package models

import "time"

// MountType represents the type of network mount.
type MountType string

const (
	// MountTypeNFS represents an NFS mount.
	MountTypeNFS MountType = "nfs"
	// MountTypeSMB represents an SMB mount.
	MountTypeSMB MountType = "smb"
	// MountTypeCIFS represents a CIFS mount.
	MountTypeCIFS MountType = "cifs"
)

// MountStatus represents the current status of a mount.
type MountStatus string

const (
	// MountStatusConnected indicates the mount is accessible.
	MountStatusConnected MountStatus = "connected"
	// MountStatusStale indicates the mount point exists but is unresponsive.
	MountStatusStale MountStatus = "stale"
	// MountStatusDisconnected indicates the mount is not accessible.
	MountStatusDisconnected MountStatus = "disconnected"
)

// MountBehavior defines what to do when a mount is unavailable.
type MountBehavior string

const (
	// MountBehaviorSkip skips the backup when mount is unavailable.
	MountBehaviorSkip MountBehavior = "skip"
	// MountBehaviorFail fails the backup when mount is unavailable.
	MountBehaviorFail MountBehavior = "fail"
)

// NetworkMount represents a detected network mount on an agent.
type NetworkMount struct {
	Path        string      `json:"path"`
	Type        MountType   `json:"type"`
	Remote      string      `json:"remote"`
	Status      MountStatus `json:"status"`
	LastChecked time.Time   `json:"last_checked"`
}

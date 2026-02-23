package models

import (
	"encoding/json"
	"time"
)

// ProxmoxInfo contains information about a Proxmox server discovered by an agent.
type ProxmoxInfo struct {
	Available    bool            `json:"available"`
	Host         string          `json:"host,omitempty"`
	Node         string          `json:"node,omitempty"`
	Version      string          `json:"version,omitempty"`
	VMCount      int             `json:"vm_count"`
	LXCCount     int             `json:"lxc_count"`
	VMs          []ProxmoxVMInfo `json:"vms,omitempty"`
	ConnectionID string          `json:"connection_id,omitempty"`
	DetectedAt   *time.Time      `json:"detected_at,omitempty"`
	Error        string          `json:"error,omitempty"`
}

// ProxmoxVMInfo represents basic information about a Proxmox VM or container.
type ProxmoxVMInfo struct {
	VMID    int    `json:"vmid"`
	Name    string `json:"name"`
	Type    string `json:"type"` // qemu or lxc
	Status  string `json:"status"`
	Node    string `json:"node"`
	CPUs    int    `json:"cpus"`
	MaxMem  int64  `json:"max_mem"`
	MaxDisk int64  `json:"max_disk"`
}

// SetProxmoxInfo sets the Proxmox info from JSON bytes.
func (a *Agent) SetProxmoxInfo(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var info ProxmoxInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return err
	}
	a.ProxmoxInfo = &info
	return nil
}

// ProxmoxInfoJSON returns the Proxmox info as JSON bytes for database storage.
func (a *Agent) ProxmoxInfoJSON() ([]byte, error) {
	if a.ProxmoxInfo == nil {
		return nil, nil
	}
	return json.Marshal(a.ProxmoxInfo)
}

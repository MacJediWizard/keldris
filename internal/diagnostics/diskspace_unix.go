//go:build !windows

package diagnostics

import "syscall"

// getDiskSpace returns disk space information for the given path.
func getDiskSpace(path string) (*DiskSpaceDetails, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, err
	}

	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bavail) * int64(stat.Bsize)
	used := total - free
	usedPct := float64(used) / float64(total) * 100

	return &DiskSpaceDetails{
		Path:       path,
		TotalBytes: total,
		FreeBytes:  free,
		UsedBytes:  used,
		UsedPct:    usedPct,
	}, nil
}

//go:build windows

package diagnostics

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// getDiskSpace returns disk space information for the given path.
func getDiskSpace(path string) (*DiskSpaceDetails, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(pathPtr, (*uint64)(unsafe.Pointer(&freeBytesAvailable)), (*uint64)(unsafe.Pointer(&totalBytes)), (*uint64)(unsafe.Pointer(&totalFreeBytes))); err != nil {
		return nil, err
	}

	total := int64(totalBytes)
	free := int64(freeBytesAvailable)
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

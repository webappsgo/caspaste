//go:build windows
// +build windows

package backup

import "golang.org/x/sys/windows"

// diskTotalBytes returns the total size in bytes of the volume containing path.
func diskTotalBytes(path string) (uint64, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(pathPtr, &freeBytesAvailable, &totalBytes, &totalFreeBytes); err != nil {
		return 0, err
	}

	return totalBytes, nil
}

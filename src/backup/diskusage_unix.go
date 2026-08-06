//go:build !windows
// +build !windows

package backup

import "golang.org/x/sys/unix"

// diskTotalBytes returns the total size in bytes of the filesystem containing path.
func diskTotalBytes(path string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	// #nosec G115 -- Bsize/Blocks are platform-defined unsigned block counts
	return stat.Blocks * uint64(stat.Bsize), nil
}

// diskFreeBytes returns the space available to an unprivileged process on
// the filesystem containing path (Bavail, not Bfree, so it matches what a
// non-root backup process could actually write).
func diskFreeBytes(path string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	// #nosec G115 -- Bsize/Bavail are platform-defined unsigned block counts
	return stat.Bavail * uint64(stat.Bsize), nil
}

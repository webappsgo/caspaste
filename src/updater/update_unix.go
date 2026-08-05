//go:build !windows
// +build !windows

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package updater

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// ReplaceBinary replaces the running binary (Unix)
// On Unix, we can replace a running binary - the old binary stays in memory
// until the process exits, then the new one takes over on next start
func ReplaceBinary(currentPath, newBinaryPath string) error {
	// Get current binary permissions
	info, err := os.Stat(currentPath)
	if err != nil {
		return fmt.Errorf("failed to stat current binary: %w", err)
	}

	// Atomic rename: new binary replaces current
	// This works because Unix allows renaming over a running executable
	if err := os.Rename(newBinaryPath, currentPath); err != nil {
		// A rename across filesystems (e.g. /tmp -> /usr/local/bin) fails with
		// EXDEV; fall back to staging a copy in the target directory and doing an
		// atomic same-filesystem rename onto the current binary.
		if errors.Is(err, syscall.EXDEV) {
			if ferr := replaceAcrossFilesystems(currentPath, newBinaryPath, info.Mode()); ferr != nil {
				return ferr
			}
			return nil
		}
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Restore permissions
	if err := os.Chmod(currentPath, info.Mode()); err != nil {
		return fmt.Errorf("failed to restore permissions: %w", err)
	}

	return nil
}

// replaceAcrossFilesystems copies newBinaryPath into the directory of
// currentPath, then atomically renames the copy onto currentPath. Both the copy
// target and currentPath live on the same filesystem, so the final rename is
// atomic even when the source binary was staged on a different mount.
func replaceAcrossFilesystems(currentPath, newBinaryPath string, mode os.FileMode) error {
	dir := filepath.Dir(currentPath)

	tmp, err := os.CreateTemp(dir, ".caspaste-update-*")
	if err != nil {
		return fmt.Errorf("failed to create staging file: %w", err)
	}
	tmpPath := tmp.Name()

	src, err := os.Open(newBinaryPath)
	if err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to open new binary: %w", err)
	}

	if _, err := io.Copy(tmp, src); err != nil {
		src.Close()
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to copy new binary: %w", err)
	}
	src.Close()

	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to flush staging file: %w", err)
	}

	if err := os.Chmod(tmpPath, mode); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(tmpPath, currentPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Remove the original staged binary now that the copy is in place.
	os.Remove(newBinaryPath)

	return nil
}

// RestartSelf re-executes the current process (Unix)
// syscall.Exec replaces the current process with a new instance
func RestartSelf() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// syscall.Exec replaces the current process
	return syscall.Exec(exe, os.Args, os.Environ())
}

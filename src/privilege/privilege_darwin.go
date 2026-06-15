
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

//go:build darwin
// +build darwin

package privilege

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	CasPbUser  = "_caspb"
	CasPbGroup = "_caspb"
)

// findAvailableUID finds first available UID in range 200-500
// On macOS, system users start with underscore and use UIDs < 500
// No hardcoded preference - always finds the first available on the runtime system
func findAvailableUID() (int, error) {
	for uid := 200; uid <= 500; uid++ {
		if _, err := user.LookupId(strconv.Itoa(uid)); err != nil {
			return uid, nil
		}
	}
	return 0, fmt.Errorf("no available UID in range 200-500")
}

// EnsureUser creates the caspaste user and group if they don't exist
func EnsureUser() (int, int, error) {
	// Check if user already exists
	u, err := user.Lookup(CasPbUser)
	if err == nil {
		uid, _ := strconv.Atoi(u.Uid)
		gid, _ := strconv.Atoi(u.Gid)
		return uid, gid, nil
	}

	// User doesn't exist, need to create
	if os.Geteuid() != 0 {
		return 0, 0, fmt.Errorf("cannot create user %s: not running as root", CasPbUser)
	}

	uid, err := findAvailableUID()
	if err != nil {
		return 0, 0, err
	}
	gid := uid

	// Create group
	cmd := exec.Command("dseditgroup", "-o", "create", "-i", strconv.Itoa(gid), CasPbGroup)
	if output, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(output), "already exists") {
			return 0, 0, fmt.Errorf("failed to create group: %w\nOutput: %s", err, string(output))
		}
	}

	// Create user
	cmd = exec.Command("sysadminctl",
		"-addUser", CasPbUser,
		"-UID", strconv.Itoa(uid),
		"-GID", strconv.Itoa(gid),
		"-shell", "/usr/bin/false",
		"-home", "/var/empty",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return 0, 0, fmt.Errorf("failed to create user: %w\nOutput: %s", err, string(output))
	}

	return uid, gid, nil
}

// DropPrivileges drops root privileges to the specified user
func DropPrivileges(uid, gid int) error {
	if os.Geteuid() != 0 {
		return nil
	}

	if err := syscall.Setgid(gid); err != nil {
		return fmt.Errorf("failed to set GID %d: %w", gid, err)
	}

	if err := syscall.Setuid(uid); err != nil {
		return fmt.Errorf("failed to set UID %d: %w", uid, err)
	}

	return nil
}

// ChownPath changes ownership of a path
func ChownPath(path string, uid, gid int) error {
	return os.Chown(path, uid, gid)
}

// ChownPathRecursive changes ownership recursively
func ChownPathRecursive(path string, uid, gid int) error {
	return filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(p, uid, gid)
	})
}

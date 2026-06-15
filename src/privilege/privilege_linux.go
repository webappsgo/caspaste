
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

//go:build linux
// +build linux

package privilege

import (
	"bufio"
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
	CasPbUser  = "caspb"
	CasPbGroup = "caspb"
)

// findAvailableUID finds first available UID in range 200-900
// No hardcoded preference - always finds the first available on the runtime system
func findAvailableUID() (int, error) {
	for uid := 200; uid <= 900; uid++ {
		if !isUIDInUse(uid) {
			return uid, nil
		}
	}
	return 0, fmt.Errorf("no available UID in range 200-900")
}

// isUIDInUse checks if a UID is already in use
func isUIDInUse(uid int) bool {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) >= 3 {
			if existingUID, err := strconv.Atoi(fields[2]); err == nil {
				if existingUID == uid {
					return true
				}
			}
		}
	}
	return false
}

// EnsureUser creates the caspaste user and group if they don't exist
// Returns UID and GID
func EnsureUser() (int, int, error) {
	// Check if user already exists
	u, err := user.Lookup(CasPbUser)
	if err == nil {
		// User exists, return their UID/GID
		uid, _ := strconv.Atoi(u.Uid)
		gid, _ := strconv.Atoi(u.Gid)
		return uid, gid, nil
	}

	// User doesn't exist, need to create
	// This requires root privileges
	if os.Geteuid() != 0 {
		return 0, 0, fmt.Errorf("cannot create user %s: not running as root", CasPbUser)
	}

	// Find available UID
	uid, err := findAvailableUID()
	if err != nil {
		return 0, 0, err
	}
	// Use same number for GID
	gid := uid

	// Try groupadd first (standard Linux)
	cmd := exec.Command("groupadd", "--gid", strconv.Itoa(gid), "--system", CasPbGroup)
	if _, err := cmd.CombinedOutput(); err != nil {
		// groupadd might not exist (Alpine), try addgroup
		cmd = exec.Command("addgroup", "-g", strconv.Itoa(gid), "-S", CasPbGroup)
		if output2, err2 := cmd.CombinedOutput(); err2 != nil {
			if !strings.Contains(string(output2), "already exists") && !strings.Contains(string(output2), "in use") {
				return 0, 0, fmt.Errorf("failed to create group: %w\nOutput: %s", err2, string(output2))
			}
		}
	}

	// Try useradd first (standard Linux)
	cmd = exec.Command("useradd",
		"--uid", strconv.Itoa(uid),
		"--gid", strconv.Itoa(gid),
		"--system",
		"--no-create-home",
		"--shell", "/sbin/nologin",
		"--comment", "CasPb Service User",
		CasPbUser,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		// useradd might not exist (Alpine), try adduser
		cmd = exec.Command("adduser",
			"-u", strconv.Itoa(uid),
			"-G", CasPbGroup,
			"-S",
			"-D",
			"-H",
			"-s", "/sbin/nologin",
			"-g", "CasPb Service User",
			CasPbUser,
		)
		if output2, err2 := cmd.CombinedOutput(); err2 != nil {
			if !strings.Contains(string(output2), "already exists") {
				return 0, 0, fmt.Errorf("failed to create user: %w\nOutput (useradd): %s\nOutput (adduser): %s", err2, string(output), string(output2))
			}
		}
	}

	return uid, gid, nil
}

// DropPrivileges drops root privileges to the specified user
func DropPrivileges(uid, gid int) error {
	if os.Geteuid() != 0 {
		// Not running as root, nothing to do
		return nil
	}

	// Set GID first (must be done before UID)
	if err := syscall.Setgid(gid); err != nil {
		return fmt.Errorf("failed to set GID %d: %w", gid, err)
	}

	// Set UID
	if err := syscall.Setuid(uid); err != nil {
		return fmt.Errorf("failed to set UID %d: %w", uid, err)
	}

	return nil
}

// ChownPath changes ownership of a path to the caspaste user
func ChownPath(path string, uid, gid int) error {
	return os.Chown(path, uid, gid)
}

// ChownPathRecursive changes ownership of a path and all contents to the caspaste user
func ChownPathRecursive(path string, uid, gid int) error {
	return filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(p, uid, gid)
	})
}

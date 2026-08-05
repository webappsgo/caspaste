
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
	CasPasteUser  = "caspaste"
	CasPasteGroup = "caspaste"
)

// reservedIDs holds UIDs/GIDs used by well-known services across distros.
// These are NEVER used even if they appear available on the current system.
var reservedIDs = map[int]bool{
	// nobody/nogroup
	65534: true,
	// systemd-*, docker, kvm (999-990)
	999: true, 998: true, 997: true, 996: true, 995: true,
	994: true, 993: true, 992: true, 991: true, 990: true,
	// sgx, pipewire, colord, avahi, rtkit, saned (989-980)
	989: true, 988: true, 987: true, 986: true, 985: true,
	984: true, 983: true, 982: true, 981: true, 980: true,
	// common services: sshd, postfix, dovecot (101-110)
	101: true, 102: true, 103: true, 104: true, 105: true,
	106: true, 107: true, 108: true, 109: true, 110: true,
	// database servers: postgres, mysql (170-179)
	170: true, 171: true, 172: true, 173: true, 174: true,
	175: true, 176: true, 177: true, 178: true, 179: true,
}

// findAvailableUID finds an available ID in the safe range 200-899,
// scanning from the top (899) down to 200 to avoid the well-known service
// IDs clustered at the top (900-999) and bottom (100-199) of 100-999.
// It skips reserved IDs and requires both the UID and the GID to be free.
func findAvailableUID() (int, error) {
	for id := 899; id >= 200; id-- {
		if reservedIDs[id] {
			continue
		}
		if isIDInUse(id) {
			continue
		}
		return id, nil
	}
	return 0, fmt.Errorf("no available UID/GID in safe range 200-899")
}

// isIDInUse reports whether the given numeric ID is already taken by either a
// user (UID) or a group (GID). Both must be free so the service account can use
// a single matching UID/GID value.
func isIDInUse(id int) bool {
	if _, err := user.LookupId(strconv.Itoa(id)); err == nil {
		return true
	}
	if _, err := user.LookupGroupId(strconv.Itoa(id)); err == nil {
		return true
	}
	return uidInPasswd(id)
}

// uidInPasswd is a fallback scan of /etc/passwd for environments where cgo-less
// user.LookupId cannot see all accounts.
func uidInPasswd(uid int) bool {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) >= 3 {
			if existingUID, err := strconv.Atoi(fields[2]); err == nil && existingUID == uid {
				return true
			}
		}
	}
	return false
}

// EnsureUser creates the caspaste user and group if they don't exist
// Returns UID and GID
func EnsureUser() (int, int, error) {
	// Check if user already exists
	u, err := user.Lookup(CasPasteUser)
	if err == nil {
		// User exists, return their UID/GID
		uid, _ := strconv.Atoi(u.Uid)
		gid, _ := strconv.Atoi(u.Gid)
		return uid, gid, nil
	}

	// User doesn't exist, need to create
	// This requires root privileges
	if os.Geteuid() != 0 {
		return 0, 0, fmt.Errorf("cannot create user %s: not running as root", CasPasteUser)
	}

	// Find available UID
	uid, err := findAvailableUID()
	if err != nil {
		return 0, 0, err
	}
	// Use same number for GID
	gid := uid

	// Try groupadd first (standard Linux)
	cmd := exec.Command("groupadd", "--gid", strconv.Itoa(gid), "--system", CasPasteGroup)
	if _, err := cmd.CombinedOutput(); err != nil {
		// groupadd might not exist (Alpine), try addgroup
		cmd = exec.Command("addgroup", "-g", strconv.Itoa(gid), "-S", CasPasteGroup)
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
		"--comment", "CasPaste Service User",
		CasPasteUser,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		// useradd might not exist (Alpine), try adduser
		cmd = exec.Command("adduser",
			"-u", strconv.Itoa(uid),
			"-G", CasPasteGroup,
			"-S",
			"-D",
			"-H",
			"-s", "/sbin/nologin",
			"-g", "CasPaste Service User",
			CasPasteUser,
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

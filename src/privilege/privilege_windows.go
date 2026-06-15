
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

//go:build windows
// +build windows

package privilege

const (
	CasPbUser  = "CasPb"
	CasPbGroup = "CasPb"
)

// EnsureUser - Windows doesn't support privilege dropping in the same way
// User creation should be done via Windows Computer Management or net user command
func EnsureUser() (int, int, error) {
	// Windows uses SIDs, not numeric UIDs
	// Return dummy values
	return 0, 0, nil
}

// DropPrivileges - Not implemented on Windows
// Windows services use different privilege model
func DropPrivileges(uid, gid int) error {
	// No-op on Windows
	return nil
}

// ChownPath - Not applicable on Windows (uses ACLs)
func ChownPath(path string, uid, gid int) error {
	return nil
}

// ChownPathRecursive - Not applicable on Windows (uses ACLs)
func ChownPathRecursive(path string, uid, gid int) error {
	return nil
}

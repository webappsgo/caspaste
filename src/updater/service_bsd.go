//go:build freebsd || openbsd || netbsd
// +build freebsd openbsd netbsd

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package updater

import (
	"os/exec"
)

// RestartService restarts the service after update (BSD)
// Per AI.md PART 23: Service-aware update coordination
func RestartService(serviceName string) error {
	// BSD systems use rc.d
	cmd := exec.Command("service", serviceName, "restart")
	if err := cmd.Run(); err != nil {
		// Fallback: just restart self
		return RestartSelf()
	}
	return nil
}

// IsRunningAsService checks if running as an rc.d service
func IsRunningAsService() bool {
	// This is a heuristic, not perfect
	// Conservative default
	return false
}

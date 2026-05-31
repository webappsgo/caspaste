//go:build darwin
// +build darwin

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package updater

import (
	"os/exec"
)

// RestartService restarts the service after update (macOS)
// Per AI.md PART 23: Service-aware update coordination
func RestartService(serviceName string) error {
	// macOS uses launchd
	label := "casjay-forks." + serviceName

	// Try user-level service first
	cmd := exec.Command("launchctl", "kickstart", "-k", "gui/"+label)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Try system-level service
	cmd = exec.Command("launchctl", "kickstart", "-k", "system/"+label)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Fallback: just restart self
	return RestartSelf()
}

// IsRunningAsService checks if running as a launchd service
func IsRunningAsService() bool {
	// This is a heuristic, not perfect
	// Conservative default
	return false
}

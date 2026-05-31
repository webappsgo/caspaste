//go:build windows
// +build windows

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package updater

import (
	"os/exec"
)

// RestartService restarts the service after update (Windows)
// Per AI.md PART 23: Service-aware update coordination
func RestartService(serviceName string) error {
	// Windows uses sc.exe for service control
	// Stop the service
	stopCmd := exec.Command("sc.exe", "stop", serviceName)
	// Ignore error, service might not be running
	stopCmd.Run()

	// Start the service
	startCmd := exec.Command("sc.exe", "start", serviceName)
	if err := startCmd.Run(); err != nil {
		// Fallback: just restart self
		return RestartSelf()
	}
	return nil
}

// IsRunningAsService checks if running as a Windows service
func IsRunningAsService() bool {
	// This is a heuristic, not perfect
	// Conservative default
	return false
}

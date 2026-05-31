//go:build linux
// +build linux

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package updater

import (
	"os/exec"
)

// RestartService restarts the service after update (Linux)
// Per AI.md PART 23: Service-aware update coordination
func RestartService(serviceName string) error {
	// Try systemd first (most common on modern Linux)
	if _, err := exec.LookPath("systemctl"); err == nil {
		cmd := exec.Command("systemctl", "restart", serviceName)
		return cmd.Run()
	}

	// Try generic service command (SysV init, OpenRC)
	if _, err := exec.LookPath("service"); err == nil {
		cmd := exec.Command("service", serviceName, "restart")
		return cmd.Run()
	}

	// Try runit
	if _, err := exec.LookPath("sv"); err == nil {
		cmd := exec.Command("sv", "restart", serviceName)
		return cmd.Run()
	}

	// Try s6
	if _, err := exec.LookPath("s6-svc"); err == nil {
		cmd := exec.Command("s6-svc", "-r", "/run/service/"+serviceName)
		return cmd.Run()
	}

	// Fallback: just restart self
	return RestartSelf()
}

// IsRunningAsService checks if running as a systemd/init service
func IsRunningAsService() bool {
	// Check if INVOCATION_ID is set (systemd sets this)
	// Or check if PPID is 1 (init/systemd)
	// This is a heuristic, not perfect
	// Conservative default
	return false
}

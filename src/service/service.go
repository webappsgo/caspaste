// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package service

import (
	"os"
	"os/exec"
	"strings"
)

// ServiceConfig holds service configuration
type ServiceConfig struct {
	Name        string
	DisplayName string
	Description string
	Executable  string
	Args        []string
	WorkingDir  string
	User        string
}

// Manager handles service operations
type Manager struct {
	config ServiceConfig
}

// New creates a new service manager
func New(config ServiceConfig) *Manager {
	return &Manager{config: config}
}

// Install installs the service
func (m *Manager) Install() error {
	return m.install()
}

// Uninstall removes the service
func (m *Manager) Uninstall() error {
	return m.uninstall()
}

// Start starts the service
func (m *Manager) Start() error {
	return m.control("start")
}

// Stop stops the service
func (m *Manager) Stop() error {
	return m.control("stop")
}

// Restart restarts the service
func (m *Manager) Restart() error {
	return m.control("restart")
}

// Reload reloads the service configuration
func (m *Manager) Reload() error {
	return m.control("reload")
}

// Disable disables the service from starting at boot
func (m *Manager) Disable() error {
	return m.disable()
}

// Status checks the service status
func (m *Manager) Status() error {
	return m.status()
}

// runCommand executes a command and returns error if it fails
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// buildArgs creates the arguments string for service configuration
func (m *Manager) buildArgs() string {
	return strings.Join(m.config.Args, " ")
}

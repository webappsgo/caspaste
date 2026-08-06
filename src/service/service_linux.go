// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

//go:build linux
// +build linux

package service

import (
	"fmt"
	"os"
	"path/filepath"
)

const systemdTemplate = `[Unit]
Description=%s
After=network.target

[Service]
Type=simple
User=%s
WorkingDirectory=%s
ExecStart=%s %s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`

func (m *Manager) install() error {
	servicePath := filepath.Join("/etc/systemd/system", m.config.Name+".service")

	content := fmt.Sprintf(systemdTemplate,
		m.config.Description,
		m.config.User,
		m.config.WorkingDir,
		m.config.Executable,
		m.buildArgs(),
	)

	err := os.WriteFile(servicePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	if err := runCommand("systemctl", "enable", m.config.Name); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	fmt.Printf("Service %s installed successfully\n", m.config.Name)
	return nil
}

func (m *Manager) uninstall() error {
	if err := runCommand("systemctl", "disable", m.config.Name); err != nil {
		fmt.Printf("Warning: failed to disable service: %v\n", err)
	}

	if err := runCommand("systemctl", "stop", m.config.Name); err != nil {
		fmt.Printf("Warning: failed to stop service: %v\n", err)
	}

	servicePath := filepath.Join("/etc/systemd/system", m.config.Name+".service")
	if err := os.Remove(servicePath); err != nil {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	fmt.Printf("Service %s uninstalled successfully\n", m.config.Name)
	return nil
}

func (m *Manager) control(action string) error {
	return runCommand("systemctl", action, m.config.Name)
}

func (m *Manager) disable() error {
	return runCommand("systemctl", "disable", m.config.Name)
}

func (m *Manager) status() error {
	return runCommand("systemctl", "status", m.config.Name)
}

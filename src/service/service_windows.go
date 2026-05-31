
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

//go:build windows
// +build windows

package service

import (
	"fmt"
)

func (m *Manager) install() error {
	binPath := fmt.Sprintf("\"%s\" %s", m.config.Executable, m.buildArgs())

	args := []string{
		"create",
		m.config.Name,
		"binPath=", binPath,
		"DisplayName=", m.config.DisplayName,
		"start=", "auto",
	}

	if err := runCommand("sc", args...); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Set description
	if m.config.Description != "" {
		descArgs := []string{
			"description",
			m.config.Name,
			m.config.Description,
		}
		// Ignore error as description is optional
		runCommand("sc", descArgs...)
	}

	fmt.Printf("Service %s installed successfully\n", m.config.Name)
	return nil
}

func (m *Manager) uninstall() error {
	// Stop service first
	runCommand("sc", "stop", m.config.Name)

	if err := runCommand("sc", "delete", m.config.Name); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	fmt.Printf("Service %s uninstalled successfully\n", m.config.Name)
	return nil
}

func (m *Manager) control(action string) error {
	var scAction string
	switch action {
	case "start":
		scAction = "start"
	case "stop":
		scAction = "stop"
	case "restart":
		if err := runCommand("sc", "stop", m.config.Name); err != nil {
			return err
		}
		scAction = "start"
	case "reload":
		return fmt.Errorf("reload not supported on Windows, use restart instead")
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	return runCommand("sc", scAction, m.config.Name)
}

func (m *Manager) disable() error {
	return runCommand("sc", "config", m.config.Name, "start=", "disabled")
}

func (m *Manager) status() error {
	return runCommand("sc", "query", m.config.Name)
}

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

//go:build darwin
// +build darwin

package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const launchdTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
%s
	</array>
	<key>WorkingDirectory</key>
	<string>%s</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>/var/log/%s.log</string>
	<key>StandardErrorPath</key>
	<string>/var/log/%s.error.log</string>
</dict>
</plist>
`

func (m *Manager) install() error {
	plistPath := filepath.Join("/Library/LaunchDaemons", m.config.Name+".plist")

	// Build program arguments
	var argsXML strings.Builder
	for _, arg := range m.config.Args {
		argsXML.WriteString(fmt.Sprintf("\t\t<string>%s</string>\n", arg))
	}

	content := fmt.Sprintf(launchdTemplate,
		m.config.Name,
		m.config.Executable,
		argsXML.String(),
		m.config.WorkingDir,
		m.config.Name,
		m.config.Name,
	)

	err := os.WriteFile(plistPath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	if err := runCommand("launchctl", "load", plistPath); err != nil {
		return fmt.Errorf("failed to load service: %w", err)
	}

	fmt.Printf("Service %s installed successfully\n", m.config.Name)
	return nil
}

func (m *Manager) uninstall() error {
	plistPath := filepath.Join("/Library/LaunchDaemons", m.config.Name+".plist")

	if err := runCommand("launchctl", "unload", plistPath); err != nil {
		fmt.Printf("Warning: failed to unload service: %v\n", err)
	}

	if err := os.Remove(plistPath); err != nil {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	fmt.Printf("Service %s uninstalled successfully\n", m.config.Name)
	return nil
}

func (m *Manager) control(action string) error {
	plistPath := filepath.Join("/Library/LaunchDaemons", m.config.Name+".plist")

	switch action {
	case "start":
		return runCommand("launchctl", "load", plistPath)
	case "stop":
		return runCommand("launchctl", "unload", plistPath)
	case "restart":
		if err := runCommand("launchctl", "unload", plistPath); err != nil {
			return err
		}
		return runCommand("launchctl", "load", plistPath)
	case "reload":
		return runCommand("launchctl", "unload", plistPath)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func (m *Manager) disable() error {
	plistPath := filepath.Join("/Library/LaunchDaemons", m.config.Name+".plist")
	return runCommand("launchctl", "unload", "-w", plistPath)
}

func (m *Manager) status() error {
	return runCommand("launchctl", "list", m.config.Name)
}

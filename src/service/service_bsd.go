// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

//go:build freebsd || openbsd
// +build freebsd openbsd

package service

import (
	"fmt"
	"os"
	"path/filepath"
)

const rcTemplate = `#!/bin/sh
#
# PROVIDE: %s
# REQUIRE: NETWORKING
# KEYWORD: shutdown

. /etc/rc.subr

name="%s"
rcvar="${name}_enable"
command="%s"
command_args="%s"
pidfile="/var/run/${name}.pid"

load_rc_config $name
: ${%s_enable:="NO"}

run_rc_command "$1"
`

func (m *Manager) install() error {
	rcPath := filepath.Join("/usr/local/etc/rc.d", m.config.Name)

	content := fmt.Sprintf(rcTemplate,
		m.config.Name,
		m.config.Name,
		m.config.Executable,
		m.buildArgs(),
		m.config.Name,
	)

	err := os.WriteFile(rcPath, []byte(content), 0755)
	if err != nil {
		return fmt.Errorf("failed to write rc.d script: %w", err)
	}

	// Enable service in rc.conf
	rcConf := fmt.Sprintf("%s_enable=\"YES\"\n", m.config.Name)
	f, err := os.OpenFile("/etc/rc.conf", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to update rc.conf: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(rcConf); err != nil {
		return fmt.Errorf("failed to write to rc.conf: %w", err)
	}

	fmt.Printf("Service %s installed successfully\n", m.config.Name)
	return nil
}

func (m *Manager) uninstall() error {
	rcPath := filepath.Join("/usr/local/etc/rc.d", m.config.Name)

	// Stop service
	runCommand("service", m.config.Name, "stop")

	// Remove rc.d script
	if err := os.Remove(rcPath); err != nil {
		return fmt.Errorf("failed to remove rc.d script: %w", err)
	}

	fmt.Printf("Service %s uninstalled successfully\n", m.config.Name)
	fmt.Printf("Note: Please manually remove '%s_enable' from /etc/rc.conf\n", m.config.Name)
	return nil
}

func (m *Manager) control(action string) error {
	return runCommand("service", m.config.Name, action)
}

func (m *Manager) disable() error {
	fmt.Printf("Please manually set '%s_enable=\"NO\"' in /etc/rc.conf\n", m.config.Name)
	return runCommand("service", m.config.Name, "stop")
}

func (m *Manager) status() error {
	return runCommand("service", m.config.Name, "status")
}

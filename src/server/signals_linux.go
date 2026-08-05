//go:build linux

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package main

import (
	"os"
	"syscall"
)

// platformShutdownSignals returns the Linux-only graceful-shutdown signals.
// SIGRTMIN+3 is signal 37 on Linux/glibc and is the real-time stop signal used
// by systemd and by container runtimes (STOPSIGNAL SIGRTMIN+3), per AI.md PART 8.
func platformShutdownSignals() []os.Signal {
	return []os.Signal{syscall.Signal(37)}
}

//go:build unix

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// registerSignals subscribes the given channel to every signal the server
// reacts to on Unix platforms per AI.md PART 8. SIGHUP is caught so it can be
// explicitly ignored rather than terminating the process by default.
func registerSignals(ch chan os.Signal) {
	sigs := []os.Signal{
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGHUP,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	}
	sigs = append(sigs, platformShutdownSignals()...)
	signal.Notify(ch, sigs...)
}

// classifySignal maps a Unix signal to a main-loop action per AI.md PART 8.
func classifySignal(sig os.Signal) sigAction {
	switch sig {
	case syscall.SIGHUP:
		return sigIgnore
	case syscall.SIGUSR1:
		return sigReopenLogs
	case syscall.SIGUSR2:
		return sigStatusDump
	default:
		return sigShutdown
	}
}

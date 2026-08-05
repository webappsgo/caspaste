//go:build windows

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package main

import (
	"os"
	"os/signal"
)

// registerSignals subscribes the given channel to the only shutdown signal
// Windows delivers (os.Interrupt); SIGHUP/SIGUSR1/SIGUSR2 do not exist there,
// per AI.md PART 8 platform matrix.
func registerSignals(ch chan os.Signal) {
	signal.Notify(ch, os.Interrupt)
}

// classifySignal always requests a graceful shutdown on Windows.
func classifySignal(sig os.Signal) sigAction {
	return sigShutdown
}

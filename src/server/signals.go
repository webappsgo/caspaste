// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package main

// sigAction classifies how the main loop reacts to an incoming OS signal.
// The concrete signal-to-action mapping is platform specific and lives in the
// build-tagged signals_*.go files, per AI.md PART 8 signal table.
type sigAction int

const (
	// sigShutdown requests a graceful shutdown (SIGTERM/SIGINT/SIGQUIT and,
	// on Linux, SIGRTMIN+3).
	sigShutdown sigAction = iota
	// sigIgnore is a no-op; configuration auto-reloads via the file watcher
	// (SIGHUP).
	sigIgnore
	// sigReopenLogs reopens log files for external rotation (SIGUSR1).
	sigReopenLogs
	// sigStatusDump writes a status snapshot to the log (SIGUSR2).
	sigStatusDump
)

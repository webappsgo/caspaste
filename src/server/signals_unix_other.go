//go:build unix && !linux

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package main

import "os"

// platformShutdownSignals returns no extra shutdown signals on non-Linux Unix
// platforms (macOS/BSD lack SIGRTMIN+3), per AI.md PART 8.
func platformShutdownSignals() []os.Signal {
	return nil
}

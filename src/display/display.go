// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package display provides display mode detection per AI.md PART 33
// Detects whether to use CLI, TUI, or GUI mode based on environment
package display

import (
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
)

// Mode represents the display mode
type Mode int

const (
	// ModeHeadless indicates no interactive display available
	ModeHeadless Mode = iota
	// ModeCLI indicates command-line mode (piped/non-interactive)
	ModeCLI
	// ModeTUI indicates terminal user interface mode
	ModeTUI
	// ModeGUI indicates graphical user interface mode
	ModeGUI
)

// String returns the string representation of the mode
func (m Mode) String() string {
	switch m {
	case ModeHeadless:
		return "headless"
	case ModeCLI:
		return "cli"
	case ModeTUI:
		return "tui"
	case ModeGUI:
		return "gui"
	default:
		return "unknown"
	}
}

// Env holds the detected display environment
type Env struct {
	Mode       Mode
	IsTerminal bool
	HasDisplay bool
	IsSSH      bool
	IsMosh     bool
	IsDocker   bool
	IsIncus    bool
	Width      int
	Height     int
}

// Detect detects the current display environment
func Detect() Env {
	env := Env{}

	// Check if running in a terminal
	env.IsTerminal = term.IsTerminal(int(os.Stdout.Fd()))

	// Get terminal size if available
	if env.IsTerminal {
		w, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			env.Width = w
			env.Height = h
		}
	}

	// Check for SSH session
	env.IsSSH = os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_TTY") != ""

	// Check for Mosh session
	env.IsMosh = os.Getenv("MOSH_CONNECTION") != ""

	// Check for Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		env.IsDocker = true
	}

	// Check for Incus/LXC/Podman container
	container := os.Getenv("container")
	env.IsIncus = container == "incus" || container == "lxc" || container == "podman"

	// Check for display (X11, Wayland, macOS, Windows)
	env.HasDisplay = hasDisplay()

	// Determine mode
	env.Mode = determineMode(env)

	return env
}

// hasDisplay checks if a graphical display is available
func hasDisplay() bool {
	// Windows always has display capability
	if runtime.GOOS == "windows" {
		return true
	}

	// macOS - check if GUI session
	if runtime.GOOS == "darwin" {
		// Check for Aqua display
		if os.Getenv("DISPLAY") != "" {
			return true
		}
		// Check for macOS GUI session indicator
		if os.Getenv("__CFBundleIdentifier") != "" {
			return true
		}
		// Check if not SSH
		if os.Getenv("SSH_CLIENT") == "" && os.Getenv("SSH_TTY") == "" {
			return true
		}
	}

	// Linux/BSD - check X11 or Wayland
	if os.Getenv("DISPLAY") != "" {
		return true
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}

	return false
}

// determineMode determines the appropriate display mode
func determineMode(env Env) Mode {
	// Container environment - headless unless terminal
	if env.IsDocker || env.IsIncus {
		if env.IsTerminal {
			return ModeCLI
		}
		return ModeHeadless
	}

	// SSH/Mosh sessions - always TUI (even with X11 forwarding)
	if env.IsSSH || env.IsMosh {
		if env.IsTerminal {
			return ModeTUI
		}
		return ModeCLI
	}

	// Local with display - GUI
	if env.HasDisplay && !env.IsSSH && !env.IsMosh {
		return ModeGUI
	}

	// Terminal without display - TUI
	if env.IsTerminal {
		return ModeTUI
	}

	// No terminal, no display - headless
	return ModeHeadless
}

// ShouldUseTUI returns true if TUI mode should be used
func (e Env) ShouldUseTUI() bool {
	return e.Mode == ModeTUI
}

// ShouldUseGUI returns true if GUI mode should be used
func (e Env) ShouldUseGUI() bool {
	return e.Mode == ModeGUI
}

// ShouldUseCLI returns true if CLI mode should be used
func (e Env) ShouldUseCLI() bool {
	return e.Mode == ModeCLI
}

// IsInteractive returns true if the environment supports interactive input
func (e Env) IsInteractive() bool {
	return e.Mode == ModeTUI || e.Mode == ModeGUI
}

// DetectForCLI detects display mode specifically for CLI binary
// CLI uses TUI as default for interactive mode
func DetectForCLI() Mode {
	env := Detect()

	// CLI errors on headless (requires interaction)
	if env.Mode == ModeHeadless {
		return ModeHeadless
	}

	// If stdin has data piped, use CLI mode
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return ModeCLI
	}

	// Check for command-line arguments indicating CLI mode
	if len(os.Args) > 1 {
		cmd := strings.ToLower(os.Args[1])
		// These commands use CLI output, not TUI
		cliCommands := []string{
			"help", "--help", "-h",
			"version", "--version", "-v",
			"config", "login",
			"new", "create", "paste",
			"get", "show", "view",
			"list", "ls",
			"info", "server-info",
			"health", "healthz",
		}
		for _, c := range cliCommands {
			if cmd == c {
				return ModeCLI
			}
		}
	}

	// No arguments - launch TUI app
	return env.Mode
}

// DetectForServer detects display mode specifically for Server binary
// Server just shows status banner, no full TUI/GUI
func DetectForServer() Mode {
	env := Detect()

	// Server defaults to headless (daemon mode)
	if env.Mode == ModeHeadless {
		return ModeHeadless
	}

	// Show console banner for terminal
	if env.IsTerminal {
		return ModeCLI
	}

	// Show GUI status window if display available
	if env.HasDisplay {
		return ModeGUI
	}

	return ModeHeadless
}

// ColorMode represents the color output mode per AI.md PART 8
type ColorMode string

const (
	// ColorAuto auto-detects based on TTY, NO_COLOR, TERM
	ColorAuto ColorMode = "auto"
	// ColorAlways forces colors on
	ColorAlways ColorMode = "always"
	// ColorNever forces colors off
	ColorNever ColorMode = "never"
)

// colorOverride is set by --color flag
var colorOverride ColorMode = ColorAuto

// SetColorMode sets the color mode from --color flag per AI.md PART 8
func SetColorMode(mode string) {
	switch strings.ToLower(mode) {
	case "always":
		colorOverride = ColorAlways
	case "never":
		colorOverride = ColorNever
	default:
		colorOverride = ColorAuto
	}
}

// GetColorMode returns the current color mode
func GetColorMode() ColorMode {
	return colorOverride
}

// IsDumbTerminal returns true if TERM=dumb per AI.md PART 7
// When TERM=dumb, ALL ANSI escapes must be disabled
func IsDumbTerminal() bool {
	return os.Getenv("TERM") == "dumb"
}

// ColorEnabled returns whether colors should be used per AI.md PART 8
// Priority order (highest to lowest):
// 1. CLI flag (--color=always|never)
// 2. NO_COLOR env var (non-empty = disable)
// 3. Auto-detect (TTY check, TERM variable)
func ColorEnabled() bool {
	// 1. CLI flag takes highest priority
	switch colorOverride {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	}

	// 2. NO_COLOR env var (non-empty = disable)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// 3. TERM=dumb disables colors
	if IsDumbTerminal() {
		return false
	}

	// 4. Auto-detect: only color if TTY
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// EmojiEnabled returns whether emojis should be used per AI.md PART 8
// Priority order:
// 1. CLI flag (--color=never disables emojis too)
// 2. NO_COLOR env var (non-empty = disable emojis)
// 3. TERM=dumb (disable emojis)
// 4. Auto-detect (TTY check)
func EmojiEnabled() bool {
	// 1. CLI flag --color=never disables emojis
	if colorOverride == ColorNever {
		return false
	}

	// 2. NO_COLOR disables emojis (practical: users wanting plain output)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// 3. TERM=dumb disables emojis
	if IsDumbTerminal() {
		return false
	}

	// 4. Auto-detect: only emojis if TTY
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// CanUseANSI returns true if ANSI escape codes can be used
// per AI.md PART 7 - checks TERM=dumb and NO_COLOR
func CanUseANSI() bool {
	if IsDumbTerminal() {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

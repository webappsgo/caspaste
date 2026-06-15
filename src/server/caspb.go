
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	chromaLexers "github.com/alecthomas/chroma/v2/lexers"

	"github.com/casjay-forks/caspaste/src/admin"
	"github.com/casjay-forks/caspaste/src/apiv1"
	"github.com/casjay-forks/caspaste/src/compat"
	"github.com/casjay-forks/caspaste/src/audit"
	"github.com/casjay-forks/caspaste/src/caspasswd"
	"github.com/casjay-forks/caspaste/src/cli"
	"github.com/casjay-forks/caspaste/src/completion"
	"github.com/casjay-forks/caspaste/src/config"
	"github.com/casjay-forks/caspaste/src/display"
	"github.com/casjay-forks/caspaste/src/graphql"
	"github.com/casjay-forks/caspaste/src/logger"
	"github.com/casjay-forks/caspaste/src/metric"
	"github.com/casjay-forks/caspaste/src/netshare"
	"github.com/casjay-forks/caspaste/src/portutil"
	"github.com/casjay-forks/caspaste/src/privilege"
	"github.com/casjay-forks/caspaste/src/raw"
	"github.com/casjay-forks/caspaste/src/scheduler"
	"github.com/casjay-forks/caspaste/src/service"
	"github.com/casjay-forks/caspaste/src/session"
	"github.com/casjay-forks/caspaste/src/token"
	"github.com/casjay-forks/caspaste/src/storage"
	"github.com/casjay-forks/caspaste/src/swagger"
	"github.com/casjay-forks/caspaste/src/template"
	"github.com/casjay-forks/caspaste/src/tor"
	"github.com/casjay-forks/caspaste/src/updater"
	"github.com/casjay-forks/caspaste/src/validation"
	"github.com/casjay-forks/caspaste/src/web"
)

// Build info - set via -ldflags at build time
var (
	Version      = "unknown"
	CommitID     = "unknown"
	BuildDate    = "unknown"
	OfficialSite = ""
)

// getVersion reads version from release.txt or returns default
func getVersion() string {
	// If Version was set at build time (via -ldflags), use it
	if Version != "unknown" {
		return Version
	}

	// Try to read from release.txt
	data, err := os.ReadFile("release.txt")
	if err == nil {
		version := strings.TrimSpace(string(data))
		if version != "" {
			return version
		}
	}

	// Default version
	return "1.0.0"
}

func readFile(path string) (string, error) {
	// Open file
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read file
	fileByte, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// Return result
	return string(fileByte), nil
}

func exitOnError(e error) {
	fmt.Fprintln(os.Stderr, "error:", e.Error())
	os.Exit(1)
}

// retryWithBackoff retries a function with exponential backoff
func retryWithBackoff(operation func() error, maxAttempts int, initialDelay time.Duration, maxDelay time.Duration, description string) error {
	var err error
	delay := initialDelay
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}
		
		// Check if it's a connection error (retryable)
		if !strings.Contains(err.Error(), "connection refused") && 
		   !strings.Contains(err.Error(), "no such host") &&
		   !strings.Contains(err.Error(), "i/o timeout") {
			// Not a connection error, fail immediately
			return err
		}
		
		if attempt < maxAttempts {
			fmt.Fprintf(os.Stderr, "[WARN]    %s failed (attempt %d/%d): %v - retrying in %v...\n", 
				description, attempt, maxAttempts, err, delay)
			time.Sleep(delay)
			
			// Exponential backoff with max delay
			delay = delay * 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
	
	return fmt.Errorf("%s failed after %d attempts: %w", description, maxAttempts, err)
}

// getDisplayAddress converts a listen address to a user-friendly display address
// Replaces 0.0.0.0, 127.0.0.1, localhost, etc. with valid FQDN, hostname, or IP
func getDisplayAddress(listenAddr string) string {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		// No port specified, use address as-is
		host = listenAddr
		port = "80"
	}

	// List of addresses to replace (localhost/loopback indicators)
	replaceableHosts := []string{"", "0.0.0.0", "127.0.0.1", "localhost", "::1", "::"}

	shouldReplace := false
	for _, replaceable := range replaceableHosts {
		if host == replaceable {
			shouldReplace = true
			break
		}
	}

	if shouldReplace {
		// Try to get hostname
		if hostname, err := os.Hostname(); err == nil && hostname != "" && hostname != "localhost" {
			host = hostname
		} else {
			// Try to get first non-loopback IP
			if addrs, err := net.InterfaceAddrs(); err == nil {
				for _, addr := range addrs {
					if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
						if ipnet.IP.To4() != nil {
							// Prefer IPv4
							host = ipnet.IP.String()
							break
						}
					}
				}
			}
		}
	}

	// If still couldn't determine, use localhost
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}

	return net.JoinHostPort(host, port)
}

// isRunningAsRoot checks if the process is running with root/admin privileges
func isRunningAsRoot() bool {
	switch runtime.GOOS {
	case "windows":
		// On Windows, check if running as administrator
		// Simple heuristic: try to create a file in Windows system directory
		testPath := os.Getenv("WINDIR") + "\\Temp\\.caspb-test"
		if f, err := os.Create(testPath); err == nil {
			f.Close()
			os.Remove(testPath)
			return true
		}
		return false
	default:
		// Unix-like systems: check if UID is 0
		return os.Geteuid() == 0
	}
}

// getDefaultDataDir returns the platform-specific default data directory
func getDefaultDataDir() string {
	// Check env var first
	if dir := os.Getenv("CASPB_DATA_DIR"); dir != "" {
		return dir
	}
	switch runtime.GOOS {
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return localAppData + "\\CasPb\\Data"
		}
		return os.Getenv("PROGRAMDATA") + "\\CasPb\\Data"
	case "darwin":
		if isRunningAsRoot() {
			return "/var/lib/casapps/caspb"
		}
		if home := os.Getenv("HOME"); home != "" {
			return home + "/Library/Application Support/CasPb"
		}
		return "/var/lib/casapps/caspb"
	// Linux, BSD, etc.
	default:
		if isRunningAsRoot() {
			return "/var/lib/casapps/caspb"
		}
		if home := os.Getenv("HOME"); home != "" {
			return home + "/.local/share/casapps/caspb"
		}
		return "/var/lib/casapps/caspb"
	}
}

// getDefaultConfigDir returns the platform-specific default config directory
func getDefaultConfigDir() string {
	// Check env var first
	if dir := os.Getenv("CASPB_CONFIG_DIR"); dir != "" {
		return dir
	}
	switch runtime.GOOS {
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return localAppData + "\\CasPb\\Config"
		}
		return os.Getenv("PROGRAMDATA") + "\\CasPb\\Config"
	case "darwin":
		if isRunningAsRoot() {
			return "/etc/casapps/caspb"
		}
		if home := os.Getenv("HOME"); home != "" {
			return home + "/Library/Application Support/CasPb/Config"
		}
		return "/etc/casapps/caspb"
	// Linux, BSD, etc.
	default:
		if isRunningAsRoot() {
			return "/etc/casapps/caspb"
		}
		if home := os.Getenv("HOME"); home != "" {
			return home + "/.config/casapps/caspb"
		}
		return "/etc/casapps/caspb"
	}
}

// getPIDFilePath returns the platform-specific PID file path per AI.md PART 8
// Default: /var/run/casapps/caspb.pid (root) or ~/.local/share/casapps/caspb/caspb.pid (user)
func getPIDFilePath(dataDir string) string {
	switch runtime.GOOS {
	case "windows":
		// Windows doesn't have standard PID file location, use data dir
		if dataDir != "" {
			return filepath.Join(dataDir, "caspb.pid")
		}
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return localAppData + "\\CasPb\\caspb.pid"
		}
		return "C:\\ProgramData\\CasPb\\caspb.pid"
	case "darwin":
		if isRunningAsRoot() {
			return "/var/run/caspb.pid"
		}
		if home := os.Getenv("HOME"); home != "" {
			return home + "/Library/Application Support/CasPb/caspb.pid"
		}
		if dataDir != "" {
			return filepath.Join(dataDir, "caspb.pid")
		}
		return "/tmp/caspb.pid"
	// Linux, BSD, etc.
	default:
		if isRunningAsRoot() {
			return "/var/run/casapps/caspb.pid"
		}
		if home := os.Getenv("HOME"); home != "" {
			return home + "/.local/share/casapps/caspb/caspb.pid"
		}
		if dataDir != "" {
			return filepath.Join(dataDir, "caspb.pid")
		}
		return "/tmp/caspb.pid"
	}
}

// ensureDirectories creates all necessary directories if they don't exist
func ensureDirectories(dataDir, configDir, dbDir, backupDir, cacheDir, logsDir string) error {
	// Create data directory
	if dataDir != "" {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dataDir, err)
		}
	}

	// Create database directory if specified and different from dataDir
	if dbDir != "" && dbDir != dataDir {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dbDir, err)
		}
	}

	// Create backup directory if specified
	if backupDir != "" {
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", backupDir, err)
		}
	}

	// Create cache directory if specified
	if cacheDir != "" {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", cacheDir, err)
		}
	}

	// Create logs directory if specified
	if logsDir != "" {
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", logsDir, err)
		}
	}

	// Create config directory
	if configDir != "" {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	return nil
}

// formatDatabaseDisplay formats database info for display (masks sensitive data)
// NEVER shows passwords - only driver type and hostname
func formatDatabaseDisplay(driver, source string) string {
	driver = strings.ToUpper(driver)

	// URL format: postgres://, mysql://, mariadb://
	// Format: scheme://user:password@host:port/db
	if strings.Contains(source, "://") {
		if strings.Contains(source, "@") {
			parts := strings.Split(source, "@")
			if len(parts) >= 2 {
				hostPart := parts[1]
				// Extract hostname before /
				if strings.Contains(hostPart, "/") {
					host := strings.Split(hostPart, "/")[0]
					return fmt.Sprintf("%s (%s)", driver, host)
				}
				return fmt.Sprintf("%s (%s)", driver, hostPart)
			}
		}
		return driver
	}

	// MySQL/MariaDB format: user:password@tcp(host:port)/dbname
	// or: user:password@unix(/path/socket)/dbname
	if strings.Contains(source, "@") {
		parts := strings.Split(source, "@")
		if len(parts) >= 2 {
			// Extract host from tcp(host:port) or unix(/path)
			hostPart := parts[1]

			// tcp(host:port)/dbname → host:port
			if strings.HasPrefix(hostPart, "tcp(") {
				if idx := strings.Index(hostPart, ")"); idx > 0 {
					// Extract content between tcp( and )
					host := hostPart[4:idx]
					return fmt.Sprintf("%s (%s)", driver, host)
				}
			}

			// unix(/path/socket) → unix socket
			if strings.HasPrefix(hostPart, "unix(") {
				return fmt.Sprintf("%s (unix socket)", driver)
			}

			// user:pass@host/db format (simple)
			if strings.Contains(hostPart, "/") {
				host := strings.Split(hostPart, "/")[0]
				return fmt.Sprintf("%s (%s)", driver, host)
			}

			return fmt.Sprintf("%s (%s)", driver, hostPart)
		}
	}

	// For file paths (SQLite), show driver and filename
	if strings.Contains(source, "/") {
		filename := source[strings.LastIndex(source, "/")+1:]
		return fmt.Sprintf("%s (%s)", driver, filename)
	}

	// Fallback - just show driver
	return driver
}

// printStartupBanner displays a formatted startup banner with server information
func printStartupBanner(version, fqdn, title, configFile, database string, httpPort, httpsPort int, generatedUser, generatedPass string) {
	// Get global IP address from default route
	globalIP, _ := validation.GetGlobalIP()

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  %-58s║\n", title)
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Version:     %-45s║\n", version)
	fmt.Printf("║  FQDN:        %-45s║\n", fqdn)
	if httpsPort > 0 {
		fmt.Printf("║  HTTP:        http://%s:%-34d║\n", fqdn, httpPort)
		fmt.Printf("║  HTTPS:       https://%s:%-33d║\n", fqdn, httpsPort)
	} else {
		portDisplay := strconv.Itoa(httpPort)
		if httpPort == 80 {
			fmt.Printf("║  URL:         http://%-41s║\n", fqdn)
		} else if httpPort == 443 {
			fmt.Printf("║  URL:         https://%-40s║\n", fqdn)
		} else {
			fmt.Printf("║  URL:         http://%s:%-34s║\n", fqdn, portDisplay)
		}
	}
	if globalIP != "" {
		fmt.Printf("║  IP:          %-45s║\n", globalIP)
	}
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║  User:        caspb (UID:GID 642:642)                   ║")
	fmt.Printf("║  Config:      %-45s║\n", configFile)
	fmt.Printf("║  Database:    %-45s║\n", database)
	fmt.Println("║  Status:      Ready                                        ║")

	// Show generated credentials if this is a private instance with new credentials
	if generatedUser != "" && generatedPass != "" {
		fmt.Println("╠════════════════════════════════════════════════════════════╣")
		fmt.Println("║  Mode:        Private (authentication required)           ║")
		fmt.Printf("║  Username:    %-45s║\n", generatedUser)
		fmt.Printf("║  Password:    %-45s║\n", generatedPass)
		fmt.Println("║  ⚠ SAVE THESE CREDENTIALS - shown only once!              ║")
	}

	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// hasArg checks if a specific argument is present in os.Args
func hasArg(arg string) bool {
	for _, a := range os.Args[1:] {
		if a == arg {
			return true
		}
	}
	return false
}

// handleUpdateCommand processes --update flag commands per AI.md PART 23
func handleUpdateCommand(command, currentVersion string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		// Default to "yes" (perform update)
		parts = []string{"yes"}
	}

	cmd := strings.ToLower(parts[0])

	// Handle --help
	if cmd == "--help" || cmd == "-help" || cmd == "help" {
		updater.PrintHelp("caspb")
		os.Exit(0)
	}

	// Configuration for updates
	cfg := updater.Config{
		CurrentVersion: currentVersion,
		// Default branch
		Branch:         "stable",
		GithubOwner:    "casjay-forks",
		GithubRepo:     "caspb",
		BinaryName:     "caspb",
	}

	switch cmd {
	case "check":
		// Check for updates without installing
		result, err := updater.CheckForUpdate(ctx, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
			os.Exit(1)
		}
		if result == nil || result.Release == nil {
			fmt.Printf("CasPb v%s is up to date\n", currentVersion)
			os.Exit(0)
		}
		fmt.Printf("Update available: %s -> %s\n", currentVersion, result.Release.TagName)
		fmt.Printf("Run 'caspb --update yes' to install\n")
		os.Exit(0)

	case "yes", "":
		// Perform update
		result, err := updater.CheckForUpdate(ctx, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
			os.Exit(1)
		}
		if result == nil || result.Release == nil {
			fmt.Printf("CasPb v%s is already up to date\n", currentVersion)
			os.Exit(0)
		}

		fmt.Printf("Updating CasPb %s -> %s...\n", currentVersion, result.Release.TagName)
		if err := updater.DoUpdate(ctx, cfg, result.Release); err != nil {
			fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Update successful!")
		fmt.Println("Restarting...")

		// Try to restart service, fallback to self restart
		if err := updater.RestartService("caspb"); err != nil {
			if err := updater.RestartSelf(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: restart failed: %v\n", err)
				fmt.Println("Please restart the service manually.")
			}
		}
		os.Exit(0)

	case "branch":
		// Switch update branch
		if len(parts) < 2 {
			fmt.Fprintln(os.Stderr, "Error: branch name required")
			fmt.Fprintln(os.Stderr, "Usage: caspb --update branch {stable|beta|daily}")
			os.Exit(1)
		}
		branch := strings.ToLower(parts[1])
		switch branch {
		case "stable", "beta", "daily":
			fmt.Printf("Update branch set to: %s\n", branch)
			fmt.Println("Note: Branch preference is not persisted (use config file for persistence)")
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "Error: invalid branch '%s'\n", branch)
			fmt.Fprintln(os.Stderr, "Valid branches: stable, beta, daily")
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown update command '%s'\n", cmd)
		fmt.Fprintln(os.Stderr, "Usage: caspb --update {check|yes|branch <name>|--help}")
		os.Exit(1)
	}
}

// handleServiceCommand processes --service flag commands
func handleServiceCommand(command, address, dbSource, dataDir, configDir string) {
	// Get executable path
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get executable path: %v\n", err)
		os.Exit(1)
	}

	// Build service config
	svcConfig := service.ServiceConfig{
		Name:        "caspb",
		DisplayName: "CasPb Pastebin Service",
		Description: "Self-hosted pastebin service",
		Executable:  executable,
		Args:        buildServiceArgs(address, dbSource, dataDir, configDir),
		WorkingDir:  dataDir,
		User:        "caspb",
	}

	mgr := service.New(svcConfig)

	switch command {
	case "start":
		err = mgr.Start()
	case "stop":
		err = mgr.Stop()
	case "restart":
		err = mgr.Restart()
	case "reload":
		err = mgr.Reload()
	case "--install", "install":
		err = mgr.Install()
	case "--uninstall", "uninstall":
		err = mgr.Uninstall()
	case "--disable", "disable":
		err = mgr.Disable()
	case "--help", "help":
		printServiceHelp()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown service command: %s\n", command)
		printServiceHelp()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Service operation failed: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

// buildServiceArgs creates the argument list for service configuration
func buildServiceArgs(address, dbSource, dataDir, configDir string) []string {
	args := []string{}

	if address != "" && address != ":80" {
		args = append(args, "--address", address)
	}
	if dbSource != "" {
		args = append(args, "--db-source", dbSource)
	}
	if dataDir != "" {
		args = append(args, "--data", dataDir)
	}
	if configDir != "" {
		args = append(args, "--config", configDir)
	}

	return args
}

// printServiceHelp shows service command help
func printServiceHelp() {
	fmt.Println("CasPb Service Management")
	fmt.Println("===========================")
	fmt.Println()
	fmt.Println("Usage: caspb --service COMMAND")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  start        - Start the service")
	fmt.Println("  stop         - Stop the service")
	fmt.Println("  restart      - Restart the service")
	fmt.Println("  reload       - Reload service configuration")
	fmt.Println("  --install    - Install service for automatic startup")
	fmt.Println("  --uninstall  - Remove service")
	fmt.Println("  --disable    - Disable service from starting at boot")
	fmt.Println("  --help       - Show this help")
	fmt.Println()
}

// handleMaintenanceCommand processes --maintenance flag commands
func handleMaintenanceCommand(command, dbDriver, dbSource, dataDir, configDir, backupDir string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		fmt.Fprintf(os.Stderr, "Maintenance command required\n")
		printMaintenanceHelp()
		os.Exit(1)
	}

	action := parts[0]
	var arg string
	if len(parts) > 1 {
		arg = parts[1]
	}

	switch action {
	case "backup":
		err := performBackup(dbDriver, dbSource, dataDir, configDir, backupDir, arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Backup failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)

	case "restore":
		err := performRestore(dbDriver, dbSource, dataDir, configDir, backupDir, arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Restore failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)

	case "mode":
		if arg == "" {
			fmt.Fprintf(os.Stderr, "Mode argument required: enabled or disabled\n")
			os.Exit(1)
		}
		err := setMaintenanceMode(dataDir, arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set maintenance mode: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)

	default:
		fmt.Fprintf(os.Stderr, "Unknown maintenance command: %s\n", action)
		printMaintenanceHelp()
		os.Exit(1)
	}
}

// printMaintenanceHelp shows maintenance command help
func printMaintenanceHelp() {
	fmt.Println("CasPb Maintenance Mode")
	fmt.Println("=========================")
	fmt.Println()
	fmt.Println("Usage: caspb --maintenance COMMAND [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  backup [filename]         - Full disaster recovery backup (default: backup-YYYYMMDD-HHMMSS.tar.gz)")
	fmt.Println("  restore [filename]        - Restore from backup (default: latest backup)")
	fmt.Println("  mode {enabled|disabled}   - Enable or disable maintenance mode")
	fmt.Println()
	fmt.Println("Backup includes:")
	fmt.Println("  - Config directory (server.yml and all config files)")
	fmt.Println("  - Data directory (db/caspb.db and all data)")
	fmt.Println("  - External SQLite database (if located outside data_dir/db/)")
	fmt.Println()
	fmt.Println("Note: When using PostgreSQL/MariaDB, db/caspb.db is a synchronized cache")
	fmt.Println("      that's included in backups for instant disaster recovery.")
	fmt.Println()
}

// checkAndMigrateDatabase checks if database driver/source changed and auto-migrates if needed
func checkAndMigrateDatabase(dataDir, configDir, backupDir, newDriver, newSource string) error {
	stateFile := dataDir + "/.db-state"

	// Read previous database state if exists
	oldStateData, err := os.ReadFile(stateFile)
	var oldDriver, oldSource string
	if err == nil {
		parts := strings.SplitN(string(oldStateData), "\n", 2)
		if len(parts) >= 1 {
			oldDriver = strings.TrimSpace(parts[0])
		}
		if len(parts) >= 2 {
			oldSource = strings.TrimSpace(parts[1])
		}
	}

	// Normalize driver names for comparison
	normalizedNew := normalizeDriverName(newDriver)
	normalizedOld := normalizeDriverName(oldDriver)

	// If driver changed, perform automatic migration
	if oldDriver != "" && oldSource != "" && (normalizedOld != normalizedNew || oldSource != newSource) {
		fmt.Println()
		fmt.Println("⚠️  Database configuration change detected!")
		fmt.Printf("Old: %s (%s)\n", oldDriver, oldSource)
		fmt.Printf("New: %s (%s)\n", newDriver, newSource)
		fmt.Println()
		fmt.Println("Starting automatic database migration...")
		fmt.Println("This may take a few minutes depending on database size.")
		fmt.Println()

		// Create backup before migration
		backupFilename := "pre-migration-" + time.Now().Format("20060102-150405") + ".tar.gz"
		fmt.Printf("Creating safety backup: %s\n", backupDir+"/"+backupFilename)
		performBackup(oldDriver, oldSource, dataDir, configDir, backupDir, backupFilename)

		// Perform migration
		err := storage.MigrateDatabase(oldDriver, oldSource, newDriver, newSource)
		if err != nil {
			fmt.Println()
			fmt.Println("❌ Migration failed!")
			fmt.Printf("Error: %v\n", err)
			fmt.Println()
			fmt.Println("Your old database is still intact. To restore:")
			fmt.Printf("  caspb --maintenance \"restore %s\" --data %s\n", backupFilename, dataDir)
			return fmt.Errorf("automatic migration failed")
		}

		fmt.Println()
		fmt.Println("✅ Migration completed successfully!")
		fmt.Println()
	}

	// Save current database state for next startup
	stateData := newDriver + "\n" + newSource
	err = os.WriteFile(stateFile, []byte(stateData), 0644)
	if err != nil {
		fmt.Printf("Warning: failed to save database state: %v\n", err)
	}

	return nil
}

// normalizeDriverName normalizes driver names for comparison and usage
func normalizeDriverName(driver string) string {
	driver = strings.ToLower(driver)
	// MariaDB uses MySQL driver
	if driver == "mariadb" {
		return "mysql"
	}
	// sqlite3 (CGo driver) → sqlite (pure Go driver)
	// We use modernc.org/sqlite (pure Go) which registers as "sqlite"
	if driver == "sqlite3" {
		return "sqlite"
	}
	return driver
}

// performBackup creates a full disaster recovery backup
func performBackup(dbDriver, dbSource, dataDir, configDir, backupDir, filename string) error {
	if dataDir == "" {
		dataDir = getDefaultDataDir()
	}

	// Generate filename if not provided
	if filename == "" {
		filename = fmt.Sprintf("backup-%s.tar.gz", time.Now().Format("20060102-150405"))
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupPath := backupDir + "/" + filename

	fmt.Println("Creating disaster recovery backup...")
	fmt.Println("Backing up:")
	fmt.Printf("  - Config: %s\n", configDir)
	fmt.Printf("  - Data: %s\n", dataDir)

	// Check if database is outside data_dir/db
	expectedDbPath := dataDir + "/db/"
	dbIsExternal := false
	if !strings.HasPrefix(dbSource, expectedDbPath) && (dbDriver == "sqlite3" || dbDriver == "sqlite") {
		dbIsExternal = true
		fmt.Printf("  - Database: %s (external)\n", dbSource)
	}

	fmt.Printf("Destination: %s\n", backupPath)
	fmt.Println()

	// Create temporary directory for staging backup
	tempDir := dataDir + "/.backup-temp"
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// Copy data directory
	cmd := exec.Command("cp", "-r", dataDir, tempDir+"/data")
	cmd.Run()

	// Copy config directory if exists
	if configDir != "" {
		if _, err := os.Stat(configDir); err == nil {
			cmd = exec.Command("cp", "-r", configDir, tempDir+"/config")
			cmd.Run()
		}
	}

	// Copy external database if needed
	if dbIsExternal {
		os.MkdirAll(tempDir+"/external-db", 0755)
		cmd = exec.Command("cp", dbSource, tempDir+"/external-db/caspb.db")
		cmd.Run()
	}

	// Create tar.gz archive
	cmd = exec.Command("tar", "-czf", backupPath,
		"--exclude=backups",
		"--exclude=.backup-temp",
		"--exclude=*.tmp",
		"--exclude=*.lock",
		"-C", tempDir,
		".")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("backup failed: %w\nOutput: %s", err, string(output))
	}

	// Get backup file size
	info, err := os.Stat(backupPath)
	if err == nil {
		fmt.Printf("Backup created: %s (%.2f MB)\n", backupPath, float64(info.Size())/1024/1024)
	} else {
		fmt.Printf("Backup created: %s\n", backupPath)
	}

	return nil
}

// performRestore performs full disaster recovery restore from backup archive
func performRestore(dbDriver, dbSource, dataDir, configDir, backupDir, filename string) error {
	if dataDir == "" {
		dataDir = getDefaultDataDir()
	}

	// If no filename, find latest backup
	if filename == "" {
		entries, err := os.ReadDir(backupDir)
		if err != nil {
			return fmt.Errorf("failed to read backup directory: %w", err)
		}

		var latestFile string
		var latestTime int64

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tar.gz") {
				info, err := entry.Info()
				if err != nil {
					continue
				}
				if info.ModTime().Unix() > latestTime {
					latestTime = info.ModTime().Unix()
					latestFile = entry.Name()
				}
			}
		}

		if latestFile == "" {
			return fmt.Errorf("no backup files found in %s", backupDir)
		}

		filename = latestFile
		fmt.Printf("Using latest backup: %s\n", filename)
	}

	backupPath := backupDir + "/" + filename

	// Check backup exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// Create safety backup of current state
	fmt.Println("Creating safety backup of current state...")
	performBackup(dbDriver, dbSource, dataDir, configDir, backupDir, "pre-restore-"+time.Now().Format("20060102-150405")+".tar.gz")

	// Create temporary extraction directory
	tempDir := dataDir + "/.restore-temp"
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// Extract backup archive to temp directory
	fmt.Printf("Restoring from: %s\n", backupPath)
	fmt.Println("Extracting backup archive...")

	cmd := exec.Command("tar", "-xzf", backupPath, "-C", tempDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restore failed: %w\nOutput: %s", err, string(output))
	}

	// Restore data directory
	if _, err := os.Stat(tempDir + "/data"); err == nil {
		fmt.Println("Restoring data directory...")
		cmd = exec.Command("cp", "-r", tempDir+"/data/.", dataDir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to restore data directory: %w", err)
		}
	}

	// Restore config directory
	if configDir != "" {
		if _, err := os.Stat(tempDir + "/config"); err == nil {
			fmt.Println("Restoring config directory...")
			cmd = exec.Command("cp", "-r", tempDir+"/config/.", configDir)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to restore config directory: %w", err)
			}
		}
	}

	// Restore external database if exists
	if _, err := os.Stat(tempDir + "/external-db/caspb.db"); err == nil {
		fmt.Println("Restoring external database...")
		if dbDriver == "sqlite3" || dbDriver == "sqlite" {
			cmd = exec.Command("cp", tempDir+"/external-db/caspb.db", dbSource)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to restore external database: %w", err)
			}
		}
	}

	fmt.Println()
	fmt.Println("Disaster recovery restore completed successfully")
	fmt.Println("Restored:")
	fmt.Printf("  - Data: %s\n", dataDir)
	if configDir != "" {
		fmt.Printf("  - Config: %s\n", configDir)
	}
	return nil
}

// setMaintenanceMode enables or disables maintenance mode
func setMaintenanceMode(dataDir, mode string) error {
	// Ensure data directory exists
	if dataDir == "" {
		dataDir = getDefaultDataDir()
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	
	maintenanceFile := filepath.Join(dataDir, ".maintenance")

	switch mode {
	case "enabled", "enable", "on":
		err := os.WriteFile(maintenanceFile, []byte("maintenance mode enabled\n"+time.Now().Format(time.RFC3339)), 0644)
		if err != nil {
			return fmt.Errorf("failed to enable maintenance mode: %w", err)
		}
		fmt.Println("Maintenance mode: ENABLED")
		fmt.Printf("Maintenance file created: %s\n", maintenanceFile)
		return nil

	case "disabled", "disable", "off":
		err := os.Remove(maintenanceFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to disable maintenance mode: %w", err)
		}
		fmt.Println("Maintenance mode: DISABLED")
		if err != nil && os.IsNotExist(err) {
			fmt.Println("(Maintenance mode was already disabled)")
		}
		return nil

	default:
		return fmt.Errorf("invalid mode: %s (use 'enabled' or 'disabled')", mode)
	}
}

// checkStatus performs health check on database and returns exit code
// Exit codes: 0 = healthy, 1 = unhealthy, 2 = error
func checkStatus(dbDriver, dbSource string, address string) {
	fmt.Println("CasPb Health Check")
	fmt.Println("=====================")
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Listen Address: %s\n", address)
	fmt.Printf("Database Driver: %s\n", dbDriver)
	fmt.Println()

	exitCode := 0
	healthy := true

	// Check database connectivity
	fmt.Print("Checking database connection... ")
	db, err := storage.NewPool(dbDriver, dbSource, 1, 0, "")
	if err != nil {
		fmt.Printf("FAILED\n  Error: %v\n", err)
		healthy = false
		exitCode = 1
	} else {
		// Try to ping the database
		err = db.Close()
		if err != nil {
			fmt.Printf("DEGRADED\n  Warning: %v\n", err)
			exitCode = 2
		} else {
			fmt.Println("OK")
		}
	}

	// Check if we can initialize database schema
	if healthy {
		fmt.Print("Checking database schema... ")
		err = storage.InitDB(dbDriver, dbSource)
		if err != nil {
			fmt.Printf("FAILED\n  Error: %v\n", err)
			healthy = false
			exitCode = 1
		} else {
			fmt.Println("OK")
		}
	}

	fmt.Println()
	if healthy && exitCode == 0 {
		fmt.Println("Status: HEALTHY")
		os.Exit(0)
	} else if exitCode == 2 {
		fmt.Println("Status: DEGRADED")
		os.Exit(2)
	} else {
		fmt.Println("Status: UNHEALTHY")
		os.Exit(1)
	}
}

// Debug endpoint handlers per AI.md PART 6
// Only registered when --debug flag is set

// handleDebugConfig returns sanitized configuration (passwords/secrets redacted)
func handleDebugConfig(cfg *config.YAMLConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Create sanitized copy (secrets redacted)
		sanitized := map[string]interface{}{
			"server": map[string]interface{}{
				"title":       cfg.Server.Title,
				"tagline":     cfg.Server.TagLine,
				"description": cfg.Server.Description,
				"fqdn":        cfg.Server.FQDN,
				"public":      cfg.Server.Public,
				"listen":      cfg.Server.Listen,
				"port":        cfg.Server.Port,
				"timeouts": map[string]interface{}{
					"read":  cfg.Server.Timeouts.Read,
					"write": cfg.Server.Timeouts.Write,
					"idle":  cfg.Server.Timeouts.Idle,
				},
			},
			"database": map[string]interface{}{
				"driver":         cfg.Database.Driver,
				"source":         "[REDACTED]",
				"cleanup_period": cfg.Database.CleanupPeriod,
				"max_open_conns": cfg.Database.MaxOpenConns,
				"max_idle_conns": cfg.Database.MaxIdleConns,
			},
			"security": map[string]interface{}{
				"tls": map[string]interface{}{
					"min_version": cfg.Security.TLS.MinVersion,
					"cert_file":   cfg.Security.TLS.CertFile,
					"key_file":    cfg.Security.TLS.KeyFile,
				},
			},
			"logging": map[string]interface{}{
				"level": cfg.Logging.Level,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		data, _ := json.MarshalIndent(sanitized, "", "  ")
		w.Write(data)
		w.Write([]byte("\n"))
	}
}

// handleDebugMemory returns memory statistics
func handleDebugMemory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := map[string]interface{}{
		"alloc_bytes":       m.Alloc,
		"total_alloc_bytes": m.TotalAlloc,
		"sys_bytes":         m.Sys,
		"mallocs":           m.Mallocs,
		"frees":             m.Frees,
		"heap_alloc_bytes":  m.HeapAlloc,
		"heap_sys_bytes":    m.HeapSys,
		"heap_idle_bytes":   m.HeapIdle,
		"heap_inuse_bytes":  m.HeapInuse,
		"heap_released":     m.HeapReleased,
		"heap_objects":      m.HeapObjects,
		"stack_inuse_bytes": m.StackInuse,
		"stack_sys_bytes":   m.StackSys,
		"gc_runs":           m.NumGC,
		"gc_pause_ns":       m.PauseNs[(m.NumGC+255)%256],
	}

	w.Header().Set("Content-Type", "application/json")
	data, _ := json.MarshalIndent(stats, "", "  ")
	w.Write(data)
	w.Write([]byte("\n"))
}

// handleDebugGoroutines returns goroutine count
func handleDebugGoroutines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := map[string]interface{}{
		"count":     runtime.NumGoroutine(),
		"gomaxproc": runtime.GOMAXPROCS(0),
		"num_cpu":   runtime.NumCPU(),
	}

	w.Header().Set("Content-Type", "application/json")
	data, _ := json.MarshalIndent(stats, "", "  ")
	w.Write(data)
	w.Write([]byte("\n"))
}

func main() {
	// Handle --shell completions/init commands first (per AI.md PART 8/33)
	// This must run before other flag parsing since --shell takes subcommands
	if len(os.Args) >= 2 && os.Args[1] == "--shell" {
		completion.Handle(os.Args[1:])
		return
	}

	var err error

	// Set timezone from TZ environment variable (default: America/New_York)
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "America/New_York"
	}
	location, err := time.LoadLocation(tz)
	if err != nil {
		// Silently fall back to UTC if timezone data not available
		location = time.UTC
	}
	time.Local = location

	// Get version (from build-time, release.txt, or default)
	Version = getVersion()

	// Read environment variables and CLI flags
	c := cli.New(Version)

	flagAddress := c.AddStringVar("address", ":80", "HTTP server ADDRESS:PORT (use FQDN for reverse proxy setups).", &cli.FlagOptions{
		PreHook: func(s string) (string, error) {
			if s == "" {
				return s, nil
			}

			// If the address doesn't contain a colon, it's missing the port
			if !strings.Contains(s, ":") {
				// Check if it looks like a FQDN (contains a dot)
				if strings.Contains(s, ".") {
					// FQDN without port: bind to all interfaces on port 80
					// The actual public URL will be constructed using reverse proxy headers
					return ":80", nil
				}
				// IP address or hostname without port: append :80
				return s + ":80", nil
			}

			// Check if it's a FQDN with port (e.g., "example.com:8080")
			parts := strings.Split(s, ":")
			if len(parts) == 2 && strings.Contains(parts[0], ".") {
				// FQDN with port: bind to all interfaces on the specified port
				// The actual public URL will be constructed using reverse proxy headers
				return ":" + parts[1], nil
			}

			return s, nil
		},
	})

	// Special commands (don't require full setup)
	flagHelp := c.AddBoolVar("help", "Show help message and exit")
	flagVersion := c.AddBoolVar("version", "Show version information and exit")
	flagDaemon := c.AddBoolVar("daemon", "Start in background (daemon mode)")
	flagDebug := c.AddBoolVar("debug", "Enable debug logging to debug.log")
	flagStatus := c.AddBoolVar("status", "Check server health and database connectivity. Exit codes: 0=healthy, 1=unhealthy, 2=error")
	flagService := c.AddStringVar("service", "", "Service management: start, stop, restart, reload, install, uninstall, disable, help", nil)
	flagMaintenance := c.AddStringVar("maintenance", "", "Maintenance mode: backup [filename], restore [filename], mode {enabled|disabled}", nil)

	// Directory flags
	flagPort := c.AddStringVar("port", "", "Port to listen on (alternative to specifying in --address). Examples: 80, 8080, 443.", nil)
	flagLog := c.AddStringVar("log", "", "Log directory for access.log and debug.log. Default: /var/log/casapps/caspb", nil)
	flagDataDir := c.AddStringVar("data", "", "Data directory. Examples: /var/lib/casapps/caspb, ~/.local/share/casapps/caspb", nil)
	flagConfigDir := c.AddStringVar("config", "", "Configuration directory. Examples: /etc/casapps/caspb, ~/.config/casapps/caspb", nil)
	flagCacheDir := c.AddStringVar("cache", "", "Cache directory. Examples: /var/cache/caspb, ~/.cache/caspb", nil)
	flagLogsDir := c.AddStringVar("logs", "", "Logs directory (alias for --log). Examples: /var/log/casapps/caspb, ~/.local/log/casapps/caspb", nil)

	// Additional flags per AI.md PART 8
	flagBackupDir := c.AddStringVar("backup", "", "Backup directory. Default: /mnt/Backups/casapps/caspb or ~/.local/share/Backups/casapps/caspb", nil)
	flagPidFile := c.AddStringVar("pid", "", "PID file path. Default: /var/run/casapps/caspb.pid or ~/.local/share/casapps/caspb/caspb.pid", nil)
	flagMode := c.AddStringVar("mode", "", "Application mode: production or development (default: production)", nil)
	flagUpdate := c.AddStringVar("update", "", "Update management: check, yes, branch {stable|beta|daily}, --help", nil)
	// Color output flag per AI.md PART 8
	flagColor := c.AddStringVar("color", "", "Color output: always, never, or auto (default: auto, respects NO_COLOR)", nil)

	c.Parse()

	// Apply --color flag immediately after parsing per AI.md PART 8
	if *flagColor != "" {
		display.SetColorMode(*flagColor)
	}

	// Handle --help first
	if *flagHelp {
		fmt.Printf("CasPb v%s - Self-hosted pastebin service\n\n", Version)
		fmt.Println("Usage: caspb [flags]")
		fmt.Println("\nCommon Flags:")
		fmt.Println("  --help              Show this help message")
		fmt.Println("  --version           Show version information")
		fmt.Println("  --daemon            Start in background (daemon mode)")
		fmt.Println("  --debug             Enable debug logging")
		fmt.Println("  --color MODE        Color output: always, never, auto (default: auto, respects NO_COLOR)")
		fmt.Println("\nServer Configuration:")
		fmt.Println("  --address ADDR      Listen address (default: :80)")
		fmt.Println("  --port PORT         Listen port (alternative to --address)")
		fmt.Println("  --mode MODE         Application mode: production or development (default: production)")
		fmt.Println("\nDirectories:")
		fmt.Println("  --data DIR          Data directory")
		fmt.Println("  --config DIR        Configuration directory")
		fmt.Println("  --log DIR           Log directory")
		fmt.Println("  --cache DIR         Cache directory")
		fmt.Println("  --backup DIR        Backup directory")
		fmt.Println("  --pid FILE          PID file path")
		fmt.Println("\nCommands:")
		fmt.Println("  --status            Check server health")
		fmt.Println("  --service CMD       Service management (start|stop|restart|reload|install|uninstall|disable)")
		fmt.Println("  --maintenance CMD   Maintenance operations (backup|restore|mode)")
		fmt.Println("  --update [CMD]      Check/perform updates (--update --help for details)")
		fmt.Println("\nShell Completions:")
		fmt.Println("  --shell completions [SHELL]   Print shell completion script")
		fmt.Println("  --shell init [SHELL]          Print shell init command for eval")
		fmt.Println("  --shell --help                Show shell integration help")
		fmt.Println("")
		fmt.Println("  Supported: bash, zsh, fish, sh, dash, ksh, powershell, pwsh")
		fmt.Println("  Example: eval \"$(caspaste --shell init)\"")
		fmt.Println("\nFor more information: https://github.com/casjay-forks/caspaste")
		os.Exit(0)
	}

	// Handle --version
	if *flagVersion {
		fmt.Printf("CasPb v%s\n", Version)
		fmt.Printf("Built with Go %s on %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// Handle --mode flag per AI.md PART 8
	// mode=development enables debug features, mode=production is default
	if *flagMode != "" {
		switch strings.ToLower(*flagMode) {
		case "development", "dev":
			// Development mode enables debug features
			*flagDebug = true
		case "production", "prod":
			// Production mode is default (debug disabled unless explicitly set)
		default:
			fmt.Fprintf(os.Stderr, "Error: invalid mode '%s'. Use 'production' or 'development'\n", *flagMode)
			os.Exit(1)
		}
	}

	// Setup log directory (needed early for daemon mode)
	if *flagLog == "" && *flagLogsDir != "" {
		*flagLog = *flagLogsDir
	}
	if *flagLog == "" {
		*flagLog = "/var/log/casapps/caspb"
	}
	os.MkdirAll(*flagLog, 0755)

	// Handle --daemon mode (fork process and exit)
	if *flagDaemon {
		if *flagDataDir == "" {
			*flagDataDir = "/var/lib/casapps/caspb"
		}
		os.MkdirAll(*flagDataDir, 0755)
		
		// Build args without --daemon flag
		args := []string{}
		for _, arg := range os.Args[1:] {
			if arg != "--daemon" && arg != "-daemon" {
				args = append(args, arg)
			}
		}
		
		// Start child process
		cmd := exec.Command(os.Args[0], args...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil
		
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start daemon: %v\n", err)
			os.Exit(1)
		}
		
		// Write PID file per AI.md PART 8
		// Priority: --pid flag > platform defaults
		pidFile := *flagPidFile
		if pidFile == "" {
			pidFile = getPIDFilePath(*flagDataDir)
		}
		// Ensure PID file directory exists
		os.MkdirAll(filepath.Dir(pidFile), 0755)
		if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write PID file: %v\n", err)
		}
		
		fmt.Printf("CasPb started in background (PID: %d)\n", cmd.Process.Pid)
		fmt.Printf("Logs: %s/access.log\n", *flagLog)
		if *flagDebug {
			fmt.Printf("Debug: %s/debug.log\n", *flagLog)
		}
		os.Exit(0)
	}

	// Declare yamlCfg at function scope (used throughout)
	var yamlCfg *config.YAMLConfig

	// Handle --status command FIRST (health check - must exit before port binding)
	if *flagStatus {

		// Minimal config loading for health check
		if *flagConfigDir != "" {
			os.MkdirAll(*flagConfigDir, 0755)
		}

		configDir := *flagConfigDir
		if configDir == "" {
			configDir = getDefaultConfigDir()
		}
		configPath := filepath.Join(configDir, "server.yml")

		cfg, err := config.LoadYAMLConfig(configPath)
		if err != nil {
			// Config doesn't exist, create default
			os.MkdirAll(configDir, 0755)
			config.GenerateDefaultYAMLConfig(configPath)
			cfg, _ = config.LoadYAMLConfig(configPath)
		}

		// Apply env overrides and normalize
		config.ApplyEnvironmentOverrides(cfg)
		cfg.Database.Driver = validation.NormalizeDriver(cfg.Database.Driver)
		cfg.Database.Source = validation.NormalizeConnectionString(cfg.Database.Driver, cfg.Database.Source)

		// Process SQLite path
		dataDir := *flagDataDir
		if dataDir == "" {
			dataDir = getDefaultDataDir()
		}
		if cfg.Database.Driver == "sqlite" && !strings.HasPrefix(cfg.Database.Source, "/") {
			dbDir := os.Getenv("CASPB_DB_DIR")
			if dbDir == "" {
				dbDir = dataDir + "/db"
			}
			cfg.Database.Source = dbDir + "/caspb.db"
		}

		// Run health check and exit
		checkStatus(cfg.Database.Driver, cfg.Database.Source, *flagAddress)
		// Explicit exit if checkStatus doesn't
		os.Exit(0)
	}

	// Handle --service command early (before heavy setup)
	if *flagService != "" {
		// Quick config load
		configDir := *flagConfigDir
		if configDir == "" {
			configDir = getDefaultConfigDir()
		}
		os.MkdirAll(configDir, 0755)

		configPath := filepath.Join(configDir, "server.yml")

		cfg, err := config.LoadYAMLConfig(configPath)
		if err != nil {
			config.GenerateDefaultYAMLConfig(configPath)
			cfg, _ = config.LoadYAMLConfig(configPath)
		}

		config.ApplyEnvironmentOverrides(cfg)
		handleServiceCommand(*flagService, *flagAddress, cfg.Database.Source, *flagDataDir, *flagConfigDir)
		// handleServiceCommand calls os.Exit() - this line never reached
	}

	// Create config directory first if specified (needed before generating config file)
	if *flagConfigDir != "" {
		if err := os.MkdirAll(*flagConfigDir, 0755); err != nil {
			exitOnError(fmt.Errorf("failed to create config directory: %w", err))
		}
	}

	// Try to load config file from config directory or platform-specific locations
	// (yamlCfg already declared earlier for --status/--service early exit)
	var configFilePath string
	// Per AI.md config-rules.md: NEVER use .yaml extension (use .yml)
	configPaths := []string{}
	if *flagConfigDir != "" {
		// When --config is explicitly set, ONLY look in that directory
		configPaths = append(configPaths, *flagConfigDir+"/server.yml")
	} else {
		// When --config is NOT set, search platform-specific and standard locations
		defaultConfigDir := getDefaultConfigDir()
		configPaths = append(configPaths,
			defaultConfigDir+"/server.yml",
			"/etc/casapps/caspb/server.yml",
			"/config/server.yml",
		)
	}

	for _, path := range configPaths {
		cfg, err := config.LoadYAMLConfig(path)
		if err == nil {
			yamlCfg = cfg
			configFilePath = path
			break
		}
	}

	// Track if this is first run (config being generated)
	isFirstRun := false
	
	// If no config file found, create default config
	if yamlCfg == nil {
		isFirstRun = true
		var defaultConfigPath string
		if *flagConfigDir != "" {
			defaultConfigPath = *flagConfigDir + "/server.yml"
		} else {
			defaultConfigPath = getDefaultConfigDir() + "/server.yml"
		}

		if err := config.GenerateDefaultYAMLConfig(defaultConfigPath); err != nil {
			exitOnError(fmt.Errorf("failed to create default config file: %w", err))
		}

		// Load the newly created config
		cfg, err := config.LoadYAMLConfig(defaultConfigPath)
		if err != nil {
			exitOnError(fmt.Errorf("failed to load generated config: %w", err))
		}
		yamlCfg = cfg
		configFilePath = defaultConfigPath
	}

	// ONLY apply environment variables and CLI flags on FIRST RUN
	// After first run, config file is the source of truth
	if isFirstRun {
		// Apply environment variable overrides to config
		// Priority: Config file < Environment variables < CLI flags
		config.ApplyEnvironmentOverrides(yamlCfg)
	}

	// ALWAYS apply security-critical environment overrides (every run)
	// This allows containerized deployments to change auth settings without deleting config
	config.ApplyCriticalOverrides(yamlCfg)

	// Handle authentication setup
	// If server.public=false (private instance), auto-generate admin credentials if needed
	// These will be displayed in the startup banner
	var generatedUser, generatedPass string
	if !yamlCfg.Server.Public {
		passwordFile := yamlCfg.Security.PasswordFile
		if passwordFile == "" {
			// Default password file location
			passwordFile = filepath.Join(*flagConfigDir, ".auth")
			yamlCfg.Security.PasswordFile = passwordFile
		}

		// Check if password file exists and has users
		if !caspasswd.FileExistsAndHasUsers(passwordFile) {
			// Auto-generate admin credentials (will be shown in startup banner)
			var err error
			generatedUser, generatedPass, err = caspasswd.GenerateCredentialsFile(passwordFile)
			if err != nil {
				exitOnError(fmt.Errorf("failed to generate credentials: %w", err))
			}
		}
	}

	// Merge CLI flags ONLY on first run (after that, config is source of truth)
	if isFirstRun {
		if *flagPort != "" {
			yamlCfg.Server.Port = *flagPort
		}
		if *flagAddress != ":80" {
			yamlCfg.Server.FQDN = *flagAddress
		}

		// Merge cache/logs directories from CLI (override config if specified)
		if *flagDataDir != "" {
			yamlCfg.Directories.Data = *flagDataDir
		}
		if *flagConfigDir != "" {
			yamlCfg.Directories.Config = *flagConfigDir
		}
		if *flagCacheDir != "" {
			yamlCfg.Directories.Cache = *flagCacheDir
		}
		if *flagLogsDir != "" {
			yamlCfg.Directories.Logs = *flagLogsDir
		}
		// Use --log flag value if provided (takes precedence over --logs)
		if *flagLog != "" {
			yamlCfg.Directories.Logs = *flagLog
		}
	}
	
	// Always set flagAddress from config if config has a value (for display purposes)
	if yamlCfg.Server.FQDN != "" && !isFirstRun {
		*flagAddress = yamlCfg.Server.FQDN
	}

	// Auto-detect driver from connection string if not specified
	if yamlCfg.Database.Driver == "" {
		detectedDriver, err := validation.DetectDriver(yamlCfg.Database.Source)
		if err != nil {
			exitOnError(fmt.Errorf("could not detect database driver: %w (specify database.driver in config)", err))
		}
		yamlCfg.Database.Driver = detectedDriver
		fmt.Printf("Auto-detected database driver: %s\n", detectedDriver)
	}

	// Normalize database driver name (sqlite3 → sqlite, mariadb → mysql)
	yamlCfg.Database.Driver = validation.NormalizeDriver(yamlCfg.Database.Driver)

	// Normalize connection string (remove sqlite:// prefix, etc.)
	yamlCfg.Database.Source = validation.NormalizeConnectionString(yamlCfg.Database.Driver, yamlCfg.Database.Source)

	// Process --port flag (overrides port in --address) - ONLY on first run
	if isFirstRun && *flagPort != "" {
		// Extract host from address (if any)
		addr := *flagAddress
		if strings.Contains(addr, ":") {
			// Remove existing port
			parts := strings.Split(addr, ":")
			addr = parts[0]
		}
		// Append new port
		if !strings.HasPrefix(*flagPort, ":") {
			*flagAddress = addr + ":" + *flagPort
		} else {
			*flagAddress = addr + *flagPort
		}
	}

	// Process --data directory and determine database directory from config
	var dbDir string
	dbSource := yamlCfg.Database.Source
	if dbSource == "" {
		exitOnError(errors.New("database.source must be specified in config file"))
	}

	// Use data directory from config (after first run) or flag (first run)
	dataDir := yamlCfg.Directories.Data
	if dataDir == "" && *flagDataDir != "" {
		dataDir = *flagDataDir
	}

	// Only process file paths for SQLite databases
	// PostgreSQL/MySQL use connection strings (postgres://, mysql://, etc.)
	driver := yamlCfg.Database.Driver
	if driver == "sqlite" {
		// If database source is relative, make it absolute based on data directory
		if !strings.HasPrefix(dbSource, "/") && dataDir != "" {
			// Check for environment variable ONLY on first run
			if isFirstRun {
				dbDir = os.Getenv("CASPB_DB_DIR")
			}
			if dbDir == "" {
				// Default: {dataDir}/db
				dbDir = dataDir + "/db"
			}
			yamlCfg.Database.Source = dbDir + "/caspb.db"
			dbSource = yamlCfg.Database.Source
		}

		// Extract directory from database source path
		if strings.Contains(dbSource, "/") {
			lastSlash := strings.LastIndex(dbSource, "/")
			if lastSlash > 0 {
				dbDir = dbSource[:lastSlash]
			}
		}
	}

	// Determine backup directory per AI.md PART 8
	// Priority: --backup flag > env var > config > platform defaults
	var backupDir string
	if *flagBackupDir != "" {
		backupDir = *flagBackupDir
	} else if isFirstRun {
		backupDir = os.Getenv("CASPB_BACKUP_DIR")
	}
	if backupDir == "" && dataDir != "" {
		// Platform-specific defaults
		isRoot := isRunningAsRoot()
		switch runtime.GOOS {
		case "linux":
			if isRoot {
				backupDir = "/var/backups/caspb"
			} else {
				if home := os.Getenv("HOME"); home != "" {
					backupDir = home + "/.local/share/casapps/caspb/backups"
				} else {
					backupDir = dataDir + "/backups"
				}
			}
		case "darwin":
			if isRoot {
				backupDir = "/var/backups/caspb"
			} else {
				if home := os.Getenv("HOME"); home != "" {
					backupDir = home + "/Library/Application Support/CasPb/Backups"
				} else {
					backupDir = dataDir + "/backups"
				}
			}
		case "windows":
			if isRoot {
				if programData := os.Getenv("ProgramData"); programData != "" {
					backupDir = programData + "\\CasPb\\Backups"
				} else {
					backupDir = "C:\\ProgramData\\CasPb\\Backups"
				}
			} else {
				if appdata := os.Getenv("APPDATA"); appdata != "" {
					backupDir = appdata + "\\CasPb\\Backups"
				} else {
					backupDir = dataDir + "/backups"
				}
			}
		case "freebsd", "openbsd":
			if isRoot {
				backupDir = "/var/backups/caspb"
			} else {
				if home := os.Getenv("HOME"); home != "" {
					backupDir = home + "/.caspb/backups"
				} else {
					backupDir = dataDir + "/backups"
				}
			}
		default:
			backupDir = dataDir + "/backups"
		}
	}

	// Determine cache directory (always use standard OS/user dirs)
	cacheDir := yamlCfg.Directories.Cache
	if cacheDir == "" {
		isRoot := isRunningAsRoot()
		switch runtime.GOOS {
		case "linux":
			if isRoot {
				cacheDir = "/var/cache/caspb"
			} else {
				if home := os.Getenv("HOME"); home != "" {
					cacheDir = home + "/.cache/caspb"
				} else {
					cacheDir = dataDir + "/cache"
				}
			}
		case "darwin":
			if isRoot {
				cacheDir = "/var/cache/caspb"
			} else {
				if home := os.Getenv("HOME"); home != "" {
					cacheDir = home + "/Library/Caches/CasPb"
				} else {
					cacheDir = dataDir + "/cache"
				}
			}
		case "windows":
			if isRoot {
				cacheDir = "C:\\ProgramData\\CasPb\\Cache"
			} else {
				if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
					cacheDir = localAppData + "\\CasPb\\Cache"
				} else {
					cacheDir = dataDir + "/cache"
				}
			}
		case "freebsd", "openbsd":
			if isRoot {
				cacheDir = "/var/cache/caspb"
			} else {
				if home := os.Getenv("HOME"); home != "" {
					cacheDir = home + "/.cache/caspb"
				} else {
					cacheDir = dataDir + "/cache"
				}
			}
		default:
			cacheDir = dataDir + "/cache"
		}
	}

	// Determine logs directory
	logsDir := yamlCfg.Directories.Logs
	if logsDir == "" && isFirstRun {
		logsDir = os.Getenv("CASPB_LOGS_DIR")
	}
	if logsDir == "" && dataDir != "" {
		isRoot := isRunningAsRoot()
		switch runtime.GOOS {
		case "linux":
			if isRoot {
				logsDir = "/var/log/casapps/caspb"
			} else {
				if home := os.Getenv("HOME"); home != "" {
					logsDir = home + "/.local/log/casapps/caspb"
				} else {
					logsDir = dataDir + "/logs"
				}
			}
		case "darwin":
			if isRoot {
				logsDir = "/var/log/casapps/caspb"
			} else {
				if home := os.Getenv("HOME"); home != "" {
					logsDir = home + "/Library/Logs/CasPaste"
				} else {
					logsDir = dataDir + "/logs"
				}
			}
		case "windows":
			if isRoot {
				logsDir = "C:\\ProgramData\\CasPb\\Logs"
			} else {
				if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
					logsDir = localAppData + "\\CasPb\\Logs"
				} else {
					logsDir = dataDir + "/logs"
				}
			}
		case "freebsd", "openbsd":
			if isRoot {
				logsDir = "/var/log/casapps/caspb"
			} else {
				if home := os.Getenv("HOME"); home != "" {
					logsDir = home + "/.local/log/casapps/caspb"
				} else {
					logsDir = dataDir + "/logs"
				}
			}
		default:
			logsDir = dataDir + "/logs"
		}
	}

	// Determine config directory for saving
	configDir := *flagConfigDir
	if configDir == "" {
		configDir = getDefaultConfigDir()
	}
	saveConfigPath := configDir + "/server.yml"

	// Save all determined directories to config NOW (before any privilege changes)
	yamlCfg.Directories.Data = dataDir
	yamlCfg.Directories.Config = configDir
	yamlCfg.Directories.Db = dbDir
	yamlCfg.Directories.Cache = cacheDir
	yamlCfg.Directories.Logs = logsDir

	if err := config.SaveYAMLConfig(saveConfigPath, yamlCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save directories to config: %v\n", err)
	}

	// Setup user (Linux/BSD/macOS only) - must be done before creating directories
	var uid, gid int
	if runtime.GOOS != "windows" {
		var err error
		uid, gid, err = privilege.EnsureUser()
		if err != nil {
			// User creation failed - might not be running as root or user already exists
			// This is OK - we'll create directories with current user
			uid = 0
			gid = 0
		}
	}

	// Ensure all directories exist
	if err := ensureDirectories(*flagDataDir, *flagConfigDir, dbDir, backupDir, cacheDir, logsDir); err != nil {
		exitOnError(err)
	}

	// Chown ALL directories if we're running as root and created a user
	// This must be done before privilege drop to ensure the user can access everything
	if os.Geteuid() == 0 && uid > 0 && gid > 0 {
		dirsToChown := []string{*flagDataDir, *flagConfigDir, dbDir, backupDir, cacheDir, logsDir}
		for _, dir := range dirsToChown {
			if dir != "" {
				if err := privilege.ChownPathRecursive(dir, uid, gid); err != nil {
					// Log but don't fail - directory might not exist or already has correct ownership
					fmt.Fprintf(os.Stderr, "Warning: failed to chown %s: %v\n", dir, err)
				}
			}
		}
	}

	// Note: --status and --service handled earlier (line 880-1036)
	// Handle --maintenance command (reads from config, no flags needed)
	if *flagMaintenance != "" {
		// Load config to get all paths
		configPath := getDefaultConfigDir() + "/server.yml"
		if *flagConfigDir != "" {
			configPath = *flagConfigDir + "/server.yml"
		} else {
			// Try to find config in standard locations
			if _, err := os.Stat("/etc/casapps/caspb/server.yml"); err == nil {
				configPath = "/etc/casapps/caspb/server.yml"
			} else if _, err := os.Stat("/config/server.yml"); err == nil {
				configPath = "/config/server.yml"
			}
		}
		
		cfg, err := config.LoadYAMLConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config for maintenance operation: %v\n", err)
			fmt.Fprintf(os.Stderr, "Config path: %s\n", configPath)
			fmt.Fprintf(os.Stderr, "Maintenance operations require an existing configuration.\n")
			os.Exit(1)
		}
		
		// Determine directories from config
		dataDir := *flagDataDir
		if dataDir == "" {
			dataDir = "/var/lib/casapps/caspb"
		}
		
		cfgDir := *flagConfigDir
		if cfgDir == "" {
			cfgDir = filepath.Dir(configPath)
		}
		
		// Determine backup directory
		backupDirPath := ""
		if _, err := os.Stat("/mnt/Backups/caspaste"); err == nil {
			backupDirPath = "/mnt/Backups/caspaste"
		} else {
			home := os.Getenv("HOME")
			if home != "" {
				backupDirPath = home + "/.local/backups/caspaste"
			} else {
				backupDirPath = dataDir + "/backups"
			}
		}
		os.MkdirAll(backupDirPath, 0755)
		
		fmt.Printf("Using configuration from: %s\n", configPath)
		fmt.Printf("Data directory: %s\n", dataDir)
		fmt.Printf("Config directory: %s\n", cfgDir)
		fmt.Printf("Backup directory: %s\n", backupDirPath)
		fmt.Println()
		
		handleMaintenanceCommand(*flagMaintenance, cfg.Database.Driver, cfg.Database.Source, dataDir, cfgDir, backupDirPath)
		return
	}

	// Handle --update command per AI.md PART 23
	if *flagUpdate != "" || hasArg("--update") {
		handleUpdateCommand(*flagUpdate, Version)
		return
	}

	// Note: Database migration moved to AFTER database initialization (after InitDB)
	// This ensures destination database exists before attempting migration

	// Validate body max length from config
	if yamlCfg.Limits.BodyMaxLength == 0 {
		exitOnError(errors.New("limits.body_max_length cannot be 0 in config file"))
	}

	// Parse max paste lifetime from config
	maxLifeTime := int64(-1)
	if yamlCfg.Limits.MaxPasteLifetime != "" && yamlCfg.Limits.MaxPasteLifetime != "never" && yamlCfg.Limits.MaxPasteLifetime != "unlimited" {
		duration, err := cli.ParseDuration(yamlCfg.Limits.MaxPasteLifetime)
		if err != nil {
			exitOnError(fmt.Errorf("invalid limits.max_paste_lifetime in config: %w", err))
		}
		if duration < 600*time.Second {
			exitOnError(errors.New("limits.max_paste_lifetime cannot be less than 10 minutes"))
		}
		maxLifeTime = int64(duration / time.Second)
	}

	// Determine FQDN for variable replacement
	// Falls back to global IP if no valid FQDN found (never localhost)
	fqdn, err := validation.DetermineFQDN("", yamlCfg.Server.FQDN)
	if err != nil {
		// Could not determine even a global IP (highly unlikely)
		exitOnError(fmt.Errorf("failed to determine server address: %w", err))
	}

	// Load content with embedded defaults + file override
	// Embedded files in src/web/data/ are used by default
	// Config paths override embedded content if specified and file exists
	serverAbout, err := web.LoadContentWithOverride("data/about.txt", yamlCfg.Web.Content.About)
	if err != nil {
		// Log warning but continue with empty content
		fmt.Fprintf(os.Stderr, "Warning: failed to load about content: %v\n", err)
		serverAbout = ""
	}

	serverRules, err := web.LoadContentWithOverride("data/rules.txt", yamlCfg.Web.Content.Rules)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load rules content: %v\n", err)
		serverRules = ""
	}

	serverTermsOfUse, err := web.LoadContentWithOverride("data/terms.txt", yamlCfg.Web.Content.Terms)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load terms content: %v\n", err)
		serverTermsOfUse = ""
	}

	// security.txt is auto-generated, not embedded
	securityTxt := ""
	if yamlCfg.Web.Content.Security != "" {
		securityTxt, err = readFile(yamlCfg.Web.Content.Security)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read content.security: %v\n", err)
			securityTxt = ""
		}
	}

	// First, apply variable replacement to config field values
	// (they may contain {fqdn} and other variables)
	baseVars := template.Variables{
		FQDN:    fqdn,
		Version: Version,
	}

	// Replace variables in config fields
	adminEmail := template.ReplaceVariables(yamlCfg.Server.Administrator.Email, baseVars)
	adminFrom := template.ReplaceVariables(yamlCfg.Server.Administrator.From, baseVars)
	securityEmail := template.ReplaceVariables(yamlCfg.Web.Security.Contact.Email, baseVars)

	// Now create template vars with replaced config values
	templateVars := template.Variables{
		FQDN:                 fqdn,
		Version:              Version,
		// Default to HTTPS for security
		Protocol:             "https",
		ServerTitle:          yamlCfg.Server.Title,
		ServerAdminName:      yamlCfg.Server.Administrator.Name,
		ServerAdminEmail:     adminEmail,
		ServerAdminFrom:      adminFrom,
		SecurityContactEmail: securityEmail,
		SecurityContactName:  yamlCfg.Web.Security.Contact.Name,
	}

	// Apply variable replacement to content files
	serverAbout = template.ReplaceVariables(serverAbout, templateVars)
	serverRules = template.ReplaceVariables(serverRules, templateVars)
	serverTermsOfUse = template.ReplaceVariables(serverTermsOfUse, templateVars)
	securityTxt = template.ReplaceVariables(securityTxt, templateVars)

	// Use replaced values in config (not raw values with variables)
	yamlCfg.Server.Administrator.Email = adminEmail
	yamlCfg.Server.Administrator.From = adminFrom
	yamlCfg.Web.Security.Contact.Email = securityEmail

	// Create log files (keep open for application lifetime)
	// Use filenames from config or defaults
	accessLogFile := yamlCfg.Logging.Access.File
	if accessLogFile == "" {
		accessLogFile = "access.log"
	}
	errorLogFile := yamlCfg.Logging.Error.File
	if errorLogFile == "" {
		errorLogFile = "error.log"
	}
	serverLogFile := yamlCfg.Logging.Server.File
	if serverLogFile == "" {
		serverLogFile = "caspb.log"
	}
	debugLogFile := yamlCfg.Logging.Debug.File
	if debugLogFile == "" {
		debugLogFile = "debug.log"
	}
	
	// Open access.log - HTTP requests only
	accessLogPath := filepath.Join(logsDir, accessLogFile)
	accessLogFd, err := os.OpenFile(accessLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		exitOnError(fmt.Errorf("failed to open %s: %w", accessLogFile, err))
	}
	
	// Open error.log - ERROR messages only
	errorLogPath := filepath.Join(logsDir, errorLogFile)
	errorLogFd, err := os.OpenFile(errorLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		exitOnError(fmt.Errorf("failed to open %s: %w", errorLogFile, err))
	}
	
	// Open caspb.log - Application log (INFO messages)
	serverLogPath := filepath.Join(logsDir, serverLogFile)
	serverLogFd, err := os.OpenFile(serverLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		exitOnError(fmt.Errorf("failed to open %s: %w", serverLogFile, err))
	}
	
	// Open debug.log - DEBUG messages (only when --debug flag is used)
	var debugLogFd *os.File
	if *flagDebug {
		debugLogPath := filepath.Join(logsDir, debugLogFile)
		debugLogFd, err = os.OpenFile(debugLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			exitOnError(fmt.Errorf("failed to open %s: %w", debugLogFile, err))
		}
	}
	// Note: Do NOT defer close - these files must stay open for the entire application lifetime

	// Initialize audit logging per AI.md PART 11
	auditLogFile := yamlCfg.Logging.Audit.File
	if auditLogFile == "" {
		auditLogFile = "audit.log"
	}
	if yamlCfg.Logging.Audit.Enabled {
		auditCfg := audit.Config{
			Enabled:          true,
			Directory:        logsDir,
			Filename:         auditLogFile,
			MaskEmails:       yamlCfg.Logging.Audit.MaskEmails,
			IncludeUserAgent: yamlCfg.Logging.Audit.IncludeUserAgent,
		}
		if err := audit.Init(auditCfg); err != nil {
			exitOnError(fmt.Errorf("failed to initialize audit logging: %w", err))
		}
	}

	// Initialize Prometheus metrics per AI.md PART 21
	metricsCfg := metric.Config{
		Enabled:         yamlCfg.Server.Metrics.Enabled,
		Endpoint:        yamlCfg.Server.Metrics.Endpoint,
		IncludeSystem:   yamlCfg.Server.Metrics.IncludeSystem,
		IncludeRuntime:  yamlCfg.Server.Metrics.IncludeRuntime,
		Token:           yamlCfg.Server.Metrics.Token,
		DurationBuckets: yamlCfg.Server.Metrics.DurationBuckets,
		SizeBuckets:     yamlCfg.Server.Metrics.SizeBuckets,
	}
	// Set defaults if not configured
	if metricsCfg.Endpoint == "" {
		metricsCfg.Endpoint = "/metrics"
	}
	if len(metricsCfg.DurationBuckets) == 0 {
		metricsCfg.DurationBuckets = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	}
	if len(metricsCfg.SizeBuckets) == 0 {
		metricsCfg.SizeBuckets = []float64{100, 1000, 10000, 100000, 1000000, 10000000}
	}
	// Initialize metrics with app info (Version, CommitID, BuildDate from -ldflags)
	metric.Init(metricsCfg, Version, CommitID, BuildDate)

	// Apply defaults for logging stdout/stderr settings if config values are zero (not explicitly set)
	// Default behavior: Server logs to stdout, Errors to stderr
	serverStdout := yamlCfg.Logging.Server.Stdout
	debugStdout := yamlCfg.Logging.Debug.Stdout
	errorStderr := yamlCfg.Logging.Error.Stderr

	// If logging section exists but all stdout/stderr are false, apply sensible defaults
	// This handles the case where the config file doesn't have the logging section
	if !serverStdout && !debugStdout && !errorStderr &&
		!yamlCfg.Logging.Access.Stdout && !yamlCfg.Logging.Access.Stderr &&
		!yamlCfg.Logging.Error.Stdout && !yamlCfg.Logging.Server.Stderr {
		// No logging to console configured at all - use defaults
		serverStdout = true
		errorStderr = true
	}

	// Build console writers based on config stdout/stderr settings
	var consoleStdout, consoleStderr io.Writer

	// Stdout (for INFO/WARN/DEBUG based on level)
	if serverStdout || debugStdout {
		consoleStdout = os.Stdout
	}

	// Stderr (for ERROR)
	if errorStderr {
		consoleStderr = os.Stderr
	}
	
	// Access log file writer (for HTTP requests)
	var accessFileWriter io.Writer
	accessFileWriter = accessLogFd
	
	// Create logger with format configuration
	log := logger.New("2006/01/02 15:04:05")
	// Set log level: info, warn, error (affects stdout only)
	log.SetLevel(yamlCfg.Logging.Level)
	log.SetFormat(logger.LogFormat{
		Access: yamlCfg.Logging.Access.Format,
		Error:  yamlCfg.Logging.Error.Format,
		Server: yamlCfg.Logging.Server.Format,
		Debug:  yamlCfg.Logging.Debug.Format,
	})
	// Files - always written regardless of level
	log.SetFileWriters(serverLogFd, errorLogFd)
	// Console - filtered by level
	log.SetWriters(consoleStdout, consoleStderr)
	// HTTP access logs
	log.SetAccessLogWriter(accessFileWriter)
	if debugLogFd != nil {
		// Debug logs
		log.SetDebugWriter(debugLogFd)
	}
	log.SetDebugMode(*flagDebug)
	
	log.Debug("Configuration loaded from: " + configFilePath)
	log.Debug("Data directory: " + *flagDataDir)
	log.Debug("Config directory: " + *flagConfigDir)
	log.Debug("Logs directory: " + logsDir)
	log.Debug("Database: " + yamlCfg.Database.Driver + " (" + yamlCfg.Database.Source + ")")
	
	// Setup cleanup handler to close log files on shutdown
	cleanupLogFiles := func() {
		accessLogFd.Close()
		errorLogFd.Close()
		serverLogFd.Close()
		if debugLogFd != nil {
			debugLogFd.Close()
		}
	}
	defer cleanupLogFiles()

	// Record server start time for uptime calculations
	startTime := time.Now()

	log.Debug("Initializing database connection pool...")
	db, err := storage.NewPool(yamlCfg.Database.Driver, yamlCfg.Database.Source, yamlCfg.Database.MaxOpenConns, yamlCfg.Database.MaxIdleConns, *flagDataDir)
	if err != nil {
		exitOnError(err)
	}
	log.Debug("Database connection pool created successfully")

	cfg := config.Config{
		Log:               log,
		RateLimitGet:      netshare.NewRateLimitSystem(yamlCfg.Limits.RateLimit.GetPastes.Per5Min, yamlCfg.Limits.RateLimit.GetPastes.Per15Min, yamlCfg.Limits.RateLimit.GetPastes.Per1Hour),
		RateLimitNew:      netshare.NewRateLimitSystem(yamlCfg.Limits.RateLimit.NewPastes.Per5Min, yamlCfg.Limits.RateLimit.NewPastes.Per15Min, yamlCfg.Limits.RateLimit.NewPastes.Per1Hour),
		Version:           Version,
		BuildCommit:       CommitID,
		BuildDate:         BuildDate,
		Mode: func() string {
			if *flagDebug {
				return "development"
			}
			return "production"
		}(),
		ServerTagline:     yamlCfg.Server.TagLine,
		ServerDescription: yamlCfg.Server.Description,
		TitleMaxLen:       yamlCfg.Limits.TitleMaxLength,
		BodyMaxLen:        yamlCfg.Limits.BodyMaxLength,
		MaxLifeTime:       maxLifeTime,
		ServerAbout:       serverAbout,
		ServerRules:       serverRules,
		ServerTermsOfUse:  serverTermsOfUse,
		SecurityTxt:       securityTxt,
		FQDN:              fqdn,
		ServerTitle:       yamlCfg.Server.Title,
		AdminName:         yamlCfg.Server.Administrator.Name,
		AdminMail:         yamlCfg.Server.Administrator.Email,
		SecurityContactEmail: yamlCfg.Web.Security.Contact.Email,
		SecurityContactName:  yamlCfg.Web.Security.Contact.Name,
		SiteRobotsAllow:      yamlCfg.Web.SEO.Robots.Allow,
		SiteRobotsDeny:       yamlCfg.Web.SEO.Robots.Deny,
		SiteRobotsAgentsDeny: yamlCfg.Web.SEO.Robots.Agents.Deny,
		Logo:                 yamlCfg.Web.Branding.Logo,
		Favicon:              yamlCfg.Web.Branding.Favicon,
		TrustedProxies:       yamlCfg.Server.Proxy.Allowed,
		UiDefaultLifetime:    yamlCfg.Web.UI.DefaultLifetime,
		UiDefaultTheme:       yamlCfg.Web.UI.DefaultTheme,
		UiThemesDir:          yamlCfg.Web.UI.ThemesDir,
		Public:               yamlCfg.Server.Public,
		CasPasswdFile:        yamlCfg.Security.PasswordFile,
		DataDir:              dataDir,
	}

	apiv1Data := apiv1.Load(db, cfg)

	rawData := raw.Load(db, cfg)

	// Build compat base URL from FQDN + TLS flag (https when certs are configured).
	compatScheme := "http"
	if yamlCfg.Security.TLS.CertFile != "" {
		compatScheme = "https"
	}
	compatBaseURL := compatScheme + "://" + fqdn
	compatData := compat.Load(
		db,
		log,
		netshare.NewRateLimitSystem(yamlCfg.Limits.RateLimit.NewPastes.Per5Min, yamlCfg.Limits.RateLimit.NewPastes.Per15Min, yamlCfg.Limits.RateLimit.NewPastes.Per1Hour),
		Version,
		compatBaseURL,
		yamlCfg.Server.Title,
		yamlCfg.Server.Administrator.Name,
		yamlCfg.Server.Administrator.Email,
		serverAbout,
		serverRules,
		yamlCfg.Limits.TitleMaxLength,
		yamlCfg.Limits.BodyMaxLength,
		maxLifeTime,
	)

	// Init database with retry logic (for when Postgres/MySQL isn't ready yet)
	log.Debug("Initializing database schema...")
	err = retryWithBackoff(
		func() error {
			return storage.InitDB(yamlCfg.Database.Driver, yamlCfg.Database.Source)
		},
		// max 10 attempts
		10,
		// start with 1 second
		1*time.Second,
		// max 30 seconds between retries
		30*time.Second,
		"Database initialization",
	)
	if err != nil {
		exitOnError(err)
	}
	log.Debug("Database schema initialized successfully")

	// Auto-detect and perform database migration if driver changed
	// NOW safe to migrate since destination database is initialized
	if *flagDataDir != "" {
		err := checkAndMigrateDatabase(*flagDataDir, *flagConfigDir, backupDir, yamlCfg.Database.Driver, yamlCfg.Database.Source)
		if err != nil {
			// Log error but don't fail startup - migration is optional
			fmt.Fprintf(os.Stderr, "Warning: database migration failed: %v\n", err)
		}
	}

	// Chown directories AGAIN after database initialization to ensure DB file has correct ownership
	// The database file was just created, so it needs to be chowned before privilege drop
	if os.Geteuid() == 0 && uid > 0 && gid > 0 {
		dirsToChown := []string{*flagDataDir, *flagConfigDir, dbDir, backupDir, cacheDir, logsDir}
		for _, dir := range dirsToChown {
			if dir != "" {
				privilege.ChownPathRecursive(dir, uid, gid)
			}
		}
		// Also chown the auth file if it exists (created earlier for private instances)
		if yamlCfg.Security.PasswordFile != "" {
			if _, err := os.Stat(yamlCfg.Security.PasswordFile); err == nil {
				privilege.ChownPath(yamlCfg.Security.PasswordFile, uid, gid)
			}
		}
	}

	// Load pages
	webData, err := web.Load(db, cfg)
	if err != nil {
		exitOnError(err)
	}

	// Handlers
	mux := http.NewServeMux()

	// External API Compatibility routes per AI.md "External API Compatibility"
	// These are registered before "/" to ensure specific matching
	// sprunge.us compatibility
	mux.HandleFunc("/sprunge", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})
	mux.HandleFunc("/sprunge/", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})
	// ix.io compatibility
	mux.HandleFunc("/ix", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})
	mux.HandleFunc("/ix/", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})
	// termbin/netcat compatibility
	mux.HandleFunc("/termbin", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})
	mux.HandleFunc("/nc", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})
	// microbin compatibility
	mux.HandleFunc("/upload", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})
	mux.HandleFunc("/p", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})
	// Generic compatibility
	mux.HandleFunc("/compat", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})
	mux.HandleFunc("/paste", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})
	// hastebin compatibility: POST /documents → {"key":"xxxxx"}, GET /documents/{key} → {"key","data"}
	mux.HandleFunc("/documents", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})
	mux.HandleFunc("/documents/", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})

	mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		webData.Handler(rw, req)
	})
	mux.HandleFunc("/raw/", func(rw http.ResponseWriter, req *http.Request) {
		rawData.Hand(rw, req)
	})
	mux.HandleFunc("/api/", func(rw http.ResponseWriter, req *http.Request) {
		apiv1Data.Hand(rw, req)
	})

	// Register admin panel and API per AI.md PART 17
	// UI at /server/{admin_path}/, API at /api/{version}/server/{admin_path}/
	adminCfg := &admin.Config{
		BasePath:   config.AdminPath(),
		APIVersion: config.APIVersion(),
		Enabled:    true,
		DB:         db.Pool(),
		Debug:      *flagDebug,
		StartTime:  startTime,
		AppCfg:     &cfg,
		ConfigFile: configFilePath,
		DataDir:    dataDir,
		ConfigDir:  configDir,
		BackupDir:  backupDir,
	}
	adminPanel := admin.New(adminCfg)
	adminPanel.MaybeGenerateSetupToken()
	adminBasePath := config.AdminBasePath()
	adminAPIPath := config.AdminAPIPath()

	// Shared auth routes per AI.md PART 15 — handles both admin and (future) user login
	// Mounted at /server/auth/ so login/logout are independent of admin_path
	mux.Handle("/server/auth/", http.StripPrefix("/server/auth", adminPanel.AuthHandler()))

	// Admin panel UI handler
	mux.Handle(adminBasePath+"/", http.StripPrefix(adminBasePath, adminPanel.Handler()))

	// Admin API handler
	mux.Handle(adminAPIPath+"/", http.StripPrefix(adminAPIPath, adminPanel.APIHandler()))

	// Register debug/pprof endpoints per AI.md PART 6
	// Only enabled when --debug flag is set
	if *flagDebug {
		// pprof endpoints
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
		mux.Handle("/debug/pprof/block", pprof.Handler("block"))
		mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

		// expvar endpoint per AI.md PART 6
		mux.Handle("/debug/vars", expvar.Handler())

		// Custom debug endpoints per AI.md PART 6
		mux.HandleFunc("/debug/config", handleDebugConfig(yamlCfg))
		mux.HandleFunc("/debug/memory", handleDebugMemory)
		mux.HandleFunc("/debug/goroutines", handleDebugGoroutines)
	}

	// Register OpenAPI/Swagger endpoints per AI.md PART 14
	swaggerCfg := &swagger.Config{
		Title:       yamlCfg.Server.Title,
		Description: "CasPaste API - Privacy-focused pastebin service",
		Version:     cfg.Version,
		Scheme:      "http",
		Host:        "localhost",
	}
	swaggerHandler := swagger.NewHandler(swaggerCfg)
	mux.HandleFunc("/openapi", swaggerHandler.ServeUI)
	mux.HandleFunc("/openapi.json", swaggerHandler.ServeSpec)

	// Register GraphQL endpoint per AI.md PART 14
	// GET = GraphiQL UI, POST = query execution
	graphqlResolvers := graphql.NewResolvers(&graphql.ResolversConfig{
		DB:          &db,
		Version:     cfg.Version,
		Title:       yamlCfg.Server.Title,
		Public:      yamlCfg.Server.Public,
		MaxBodyLen:  cfg.BodyMaxLen,
		MaxTitleLen: cfg.TitleMaxLen,
		Lexers:      chromaLexers.Names(false),
	})
	graphqlHandler := graphql.NewHandler(&graphql.Config{
		Title:   yamlCfg.Server.Title,
		Version: cfg.Version,
	}, graphqlResolvers)
	mux.Handle("/graphql", graphqlHandler)

	// Register Prometheus metrics endpoint per AI.md PART 21
	// INTERNAL ONLY - should be firewalled from public access
	if metricsCfg.Enabled {
		mux.Handle(metricsCfg.Endpoint, metric.Handler(metricsCfg))
	}

	// Wrap with maintenance mode middleware
	dataDirectory := *flagDataDir
	if dataDirectory == "" {
		dataDirectory = getDefaultDataDir()
	}
	// Parse cleanup period from config
	cleanupPeriod, err := cli.ParseDuration(yamlCfg.Database.CleanupPeriod)
	if err != nil {
		exitOnError(fmt.Errorf("invalid database.cleanup_period in config: %w", err))
	}

	// Security headers config from yaml per AI.md PART 11
	securityHeadersCfg := web.SecurityHeadersConfig{
		XFrameOptions:           yamlCfg.Security.Headers.XFrameOptions,
		XContentTypeOptions:     yamlCfg.Security.Headers.XContentTypeOptions,
		XSSProtection:           yamlCfg.Security.Headers.XSSProtection,
		ContentSecurityPolicy:   yamlCfg.Security.Headers.ContentSecurityPolicy,
		ReferrerPolicy:          yamlCfg.Security.Headers.ReferrerPolicy,
		PermissionsPolicy:       yamlCfg.Security.Headers.PermissionsPolicy,
		StrictTransportSecurity: yamlCfg.Security.Headers.StrictTransportSecurity,
	}

	// CSRF protection config per AI.md PART 11
	csrfCfg := web.CSRFConfig{
		Enabled:     yamlCfg.Security.CSRF.Enabled,
		TokenLength: yamlCfg.Security.CSRF.TokenLength,
		CookieName:  yamlCfg.Security.CSRF.CookieName,
		HeaderName:  yamlCfg.Security.CSRF.HeaderName,
		FieldName:   yamlCfg.Security.CSRF.FieldName,
		Secure:      yamlCfg.Security.CSRF.Secure,
		// Exempt API and compatibility endpoints from CSRF (they use tokens, not cookies)
		ExemptPaths: []string{
			// Legacy compat stubs
			"/sprunge", "/sprunge/",
			"/ix", "/ix/",
			"/termbin", "/nc",
			"/upload", "/p",
			"/compat", "/paste",
			// Hastebin compat
			"/documents", "/documents/",
			// Stikked compat
			"/api/create", "/api/paste", "/api/recent", "/api/trending", "/api/langs",
			"/lists", "/trends",
			// Microbin compat
			"/list", "/archive",
			// Pastebin.com compat
			"/api/api_post.php", "/api/api_raw.php",
		},
		ExemptPrefixes: []string{
			"/api/",
			"/raw/",
			// Stikked view redirects
			"/view/",
			// Stikked list pages
			"/lists/", "/trends/",
			// Microbin compat
			"/pasta/", "/rawpasta/",
		},
	}

	// Apply middleware chain per AI.md:
	// URLNormalize → PathSecurity → PanicRecovery → RequestID → Metrics → SecurityHeaders → CORS → CSRF → Maintenance → App
	// Per AI.md PART 14: URL normalization (trailing slashes) must be first
	// Per AI.md PART 11: Path security blocks traversal attacks early
	// Per AI.md PART 6: Panic recovery must catch all panics
	// Per AI.md PART 11: Request ID middleware for tracing, security headers, CSRF protection
	// Per AI.md PART 21: Metrics middleware for HTTP request tracking
	handler := web.URLNormalizeMiddleware(
		web.PathSecurityMiddleware(
			web.PanicRecoveryMiddleware(*flagDebug)(
				web.RequestIDMiddleware(
					metric.Middleware(metricsCfg)(
						web.SecurityHeadersMiddleware(securityHeadersCfg)(
							web.CORSMiddleware(
								web.CSRFMiddleware(csrfCfg)(
									web.MaintenanceMiddleware(dataDir,
										compatData.Middleware(mux))))))))))

	// Initialize built-in scheduler per AI.md PART 19
	// ALL projects MUST have a built-in scheduler that is ALWAYS RUNNING
	// NEVER use external schedulers (cron, systemd timers, etc.)
	sched := scheduler.New(&scheduler.Config{
		Timezone:      "America/New_York",
		CatchUpWindow: time.Hour,
	})

	// Add paste cleanup task per AI.md PART 19
	sched.AddTask(&scheduler.Task{
		ID:          "paste_cleanup",
		Name:        "Paste Cleanup",
		Description: "Delete expired pastes from database",
		Schedule:    "@every " + cleanupPeriod.String(),
		Enabled:     true,
		Skippable:   false,
		Handler: func(ctx context.Context) error {
			count, err := db.PasteDeleteExpired()
			if err != nil {
				log.Error(errors.New("Delete expired: " + err.Error()))
				return err
			}
			if count > 0 {
				log.Info("Deleted " + strconv.FormatInt(count, 10) + " expired pastes")
			}
			return nil
		},
	})

	// Add session cleanup task per AI.md PART 19
	sessionSvc := session.NewService(db.Pool())
	sched.AddTask(&scheduler.Task{
		ID:          "session_cleanup",
		Name:        "Session Cleanup",
		Description: "Remove expired user sessions",
		Schedule:    "@every 15m",
		Enabled:     true,
		Skippable:   false,
		Handler: func(ctx context.Context) error {
			count, err := sessionSvc.CleanupExpired()
			if err != nil {
				log.Error(errors.New("Session cleanup: " + err.Error()))
				return err
			}
			if count > 0 {
				log.Info("Deleted " + strconv.FormatInt(count, 10) + " expired sessions")
			}
			return nil
		},
	})

	// Add token cleanup task per AI.md PART 19
	tokenSvc := token.NewService(db.Pool())
	sched.AddTask(&scheduler.Task{
		ID:          "token_cleanup",
		Name:        "Token Cleanup",
		Description: "Remove expired API tokens",
		Schedule:    "@every 15m",
		Enabled:     true,
		Skippable:   false,
		Handler: func(ctx context.Context) error {
			err := tokenSvc.CleanupExpired()
			if err != nil {
				log.Error(errors.New("Token cleanup: " + err.Error()))
				return err
			}
			return nil
		},
	})

	// Add self-health check task per AI.md PART 19
	sched.AddTask(&scheduler.Task{
		ID:          "healthcheck_self",
		Name:        "Self Health Check",
		Description: "Verify application health",
		Schedule:    "@every 5m",
		Enabled:     true,
		Skippable:   false,
		Handler: func(ctx context.Context) error {
			// Simple health check - verify database is responsive
			_, err := db.PasteDeleteExpired()
			return err
		},
	})

	// Wire scheduler health check into apiv1Data now that sched is ready
	apiv1Data.SchedulerStatus = func() string {
		if sched.IsRunning() {
			return "ok"
		}
		return "error: scheduler not running"
	}

	// Start the scheduler per AI.md PART 19
	if err := sched.Start(); err != nil {
		log.Error(fmt.Errorf("failed to start scheduler: %w", err))
	} else {
		log.Info("Built-in scheduler started per AI.md PART 19")
	}

	// Inject scheduler into admin panel now that it is running
	adminPanel.SetScheduler(sched)

	// Determine ports (HTTP and optionally HTTPS)
	var httpPort, httpsPort int

	// Check for PORT environment variable override
	portEnv := os.Getenv("PORT")
	if portEnv == "" {
		portEnv = os.Getenv("CASPB_PORT")
	}

	if portEnv != "" {
		// ENV overrides config
		httpPort, httpsPort, err = portutil.ParsePorts(portEnv)
		if err != nil {
			exitOnError(fmt.Errorf("invalid PORT environment variable: %w", err))
		}
	} else if yamlCfg.Server.Port != "" {
		// Use config port
		httpPort, httpsPort, err = portutil.ParsePorts(yamlCfg.Server.Port)
		if err != nil {
			exitOnError(fmt.Errorf("invalid server.port in config: %w", err))
		}
	} else {
		// Generate random port
		httpPort, err = portutil.FindUnusedPort(64000, 65535)
		if err != nil {
			exitOnError(fmt.Errorf("failed to find unused port: %w", err))
		}

		// Save generated port to config file for persistence
		yamlCfg.Server.Port = strconv.Itoa(httpPort)

		if err := config.SaveYAMLConfig(configFilePath, yamlCfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save port to config: %v\n", err)
		} else {
			fmt.Printf("Saved generated port %d to config file\n", httpPort)
		}
	}

	// Update flagAddress to use determined port
	// Extract host from current address
	host := *flagAddress
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		host = parts[0]
	}
	if host == "" {
		host = "::"
	}
	*flagAddress = net.JoinHostPort(host, strconv.Itoa(httpPort))

	// Convert listen address ("all" → "::", or use as-is)
	listenAddr := yamlCfg.Server.Listen
	if listenAddr == "all" || listenAddr == "" {
		// IPv4 + IPv6 dual stack
		listenAddr = "::"
	}

	// Create HTTP listener (must be done as root for ports < 1024 on Unix)
	httpAddr := net.JoinHostPort(listenAddr, strconv.Itoa(httpPort))
	httpListener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		exitOnError(fmt.Errorf("failed to bind HTTP to %s: %w", httpAddr, err))
	}

	// Create HTTPS listener if dual port configured
	var httpsListener net.Listener
	var tlsCert *validation.TLSCertPaths
	if httpsPort > 0 {
		httpsAddr := net.JoinHostPort(listenAddr, strconv.Itoa(httpsPort))
		httpsListener, err = net.Listen("tcp", httpsAddr)
		if err != nil {
			exitOnError(fmt.Errorf("failed to bind HTTPS to %s: %w", httpsAddr, err))
		}

		// Auto-detect Let's Encrypt certificates
		tlsCert, err = validation.FindLetsEncryptCerts(fqdn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: HTTPS port configured but no TLS cert found: %v\n", err)
			fmt.Fprintf(os.Stderr, "HTTPS server will not start. Configure TLS cert or remove HTTPS port.\n")
			httpsListener.Close()
			httpsListener = nil
		} else {
			fmt.Printf("Found TLS certificate for domain: %s\n", tlsCert.Domain)
		}
	}

	// Drop privileges after binding to ports (uid/gid set earlier during directory creation)
	if runtime.GOOS != "windows" && uid > 0 && gid > 0 {
		if err := privilege.DropPrivileges(uid, gid); err != nil {
			log.Error(fmt.Errorf("failed to drop privileges: %w", err))
			// Continue anyway
		}
	}

	// Print startup banner with database info
	dbDisplay := formatDatabaseDisplay(yamlCfg.Database.Driver, yamlCfg.Database.Source)
	printStartupBanner(Version, fqdn, yamlCfg.Server.Title, configFilePath, dbDisplay, httpPort, httpsPort, generatedUser, generatedPass)

	// Track server start time for uptime calculation
	serverStartTime := time.Now()

	// Log server started event to audit log per AI.md PART 11
	serverMode := "production"
	if *flagDebug {
		serverMode = "development"
	}
	audit.ServerStarted(Version, serverMode)

	// Create HTTP server with timeouts
	srv := &http.Server{
		// Custom mux with middleware
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup signal handling for graceful shutdown
	// Works on Windows, macOS, BSD, and Linux
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Start HTTP server in a goroutine
	httpErrors := make(chan error, 1)
	go func() {
		httpErrors <- srv.Serve(httpListener)
	}()

	// Start Tor hidden service per AI.md PART 32
	// Hidden service is auto-enabled when Tor binary is found
	var torManager *tor.Manager
	torCfg := &tor.Config{
		Binary:                    yamlCfg.Server.Tor.Binary,
		UseNetwork:                yamlCfg.Server.Tor.UseNetwork,
		AllowUserPreference:       yamlCfg.Server.Tor.AllowUserPreference,
		MaxCircuits:               yamlCfg.Server.Tor.MaxCircuits,
		CircuitTimeout:            time.Duration(yamlCfg.Server.Tor.CircuitTimeout) * time.Second,
		BootstrapTimeout:          time.Duration(yamlCfg.Server.Tor.BootstrapTimeout) * time.Second,
		SafeLogging:               yamlCfg.Server.Tor.SafeLogging,
		MaxStreamsPerCircuit:      yamlCfg.Server.Tor.MaxStreamsPerCircuit,
		CloseCircuitOnStreamLimit: yamlCfg.Server.Tor.CloseCircuitOnStreamLimit,
		BandwidthRate:             yamlCfg.Server.Tor.BandwidthRate,
		BandwidthBurst:            yamlCfg.Server.Tor.BandwidthBurst,
		MaxMonthlyBandwidth:       yamlCfg.Server.Tor.MaxMonthlyBandwidth,
		NumIntroPoints:            yamlCfg.Server.Tor.NumIntroPoints,
		VirtualPort:               yamlCfg.Server.Tor.VirtualPort,
	}
	// Apply defaults for zero values
	if torCfg.MaxCircuits == 0 {
		torCfg.MaxCircuits = 32
	}
	if torCfg.CircuitTimeout == 0 {
		torCfg.CircuitTimeout = 60 * time.Second
	}
	if torCfg.BootstrapTimeout == 0 {
		torCfg.BootstrapTimeout = 3 * time.Minute
	}
	if torCfg.MaxStreamsPerCircuit == 0 {
		torCfg.MaxStreamsPerCircuit = 100
	}
	if torCfg.BandwidthRate == "" {
		torCfg.BandwidthRate = "1 MB"
	}
	if torCfg.BandwidthBurst == "" {
		torCfg.BandwidthBurst = "2 MB"
	}
	if torCfg.MaxMonthlyBandwidth == "" {
		torCfg.MaxMonthlyBandwidth = "100 GB"
	}
	if torCfg.NumIntroPoints == 0 {
		torCfg.NumIntroPoints = 3
	}
	if torCfg.VirtualPort == 0 {
		torCfg.VirtualPort = 80
	}
	torManager = tor.NewManager(context.Background(), torCfg, configDir, dataDir, *flagLog, httpPort)
	if err := torManager.Start(); err != nil {
		log.Warn(fmt.Sprintf("Tor: %v", err))
	} else {
		// Log Tor status (only if enabled)
		torStatus := torManager.GetStatus()
		if torStatus.Enabled && torStatus.Running {
			log.Info(fmt.Sprintf("Tor: %s", torStatus.Hostname))
		} else if torStatus.Enabled && !torStatus.Running && torStatus.Error != "" {
			log.Warn(fmt.Sprintf("Tor: %s", torStatus.Error))
		}
	}

	// Start HTTPS server if configured and cert available
	var httpsErrors chan error
	var srvHTTPS *http.Server
	if httpsListener != nil && tlsCert != nil {
		httpsErrors = make(chan error, 1)
		
		// Configure TLS security settings
		tlsConfig := &tls.Config{
			// Default to TLS 1.2
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
			PreferServerCipherSuites: true,
		}
		
		// Apply configured TLS min version
		switch yamlCfg.Security.TLS.MinVersion {
		case "1.3":
			tlsConfig.MinVersion = tls.VersionTLS13
		case "1.2":
			tlsConfig.MinVersion = tls.VersionTLS12
		case "1.1":
			tlsConfig.MinVersion = tls.VersionTLS11
		case "1.0":
			tlsConfig.MinVersion = tls.VersionTLS10
		}
		
		srvHTTPS = &http.Server{
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
			TLSConfig:    tlsConfig,
		}

		go func() {
			httpsAddr := net.JoinHostPort(listenAddr, strconv.Itoa(httpsPort))
			log.Info("Run HTTPS server on " + httpsAddr)
			httpsErrors <- srvHTTPS.ServeTLS(httpsListener, tlsCert.CertFile, tlsCert.KeyFile)
		}()
	}

	// Wait for interrupt signal or server error
	select {
	case err := <-httpErrors:
		if err != nil && err != http.ErrServerClosed {
			exitOnError(err)
		}

	case err := <-httpsErrors:
		if err != nil && err != http.ErrServerClosed {
			exitOnError(err)
		}

	case sig := <-sigChan:
		log.Info(fmt.Sprintf("Received signal %v, shutting down gracefully...", sig))

		// Log server stopped event to audit log per AI.md PART 11
		uptime := time.Since(serverStartTime)
		audit.ServerStopped(fmt.Sprintf("signal: %v", sig), uptime)
		audit.CloseGlobal()

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt graceful shutdown for both servers
		if err := srv.Shutdown(ctx); err != nil {
			log.Error(fmt.Errorf("HTTP server shutdown error: %w", err))
			srv.Close()
		}

		if srvHTTPS != nil {
			if err := srvHTTPS.Shutdown(ctx); err != nil {
				log.Error(fmt.Errorf("HTTPS server shutdown error: %w", err))
				srvHTTPS.Close()
			}
		}

		// Stop Tor hidden service per AI.md PART 32
		if torManager != nil {
			if err := torManager.Stop(); err != nil {
				log.Error(fmt.Errorf("Tor shutdown error: %w", err))
			}
		}

		// Stop built-in scheduler per AI.md PART 19
		sched.Stop()
		log.Info("Scheduler stopped")

		log.Info("Server stopped")
	}
}

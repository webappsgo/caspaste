// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package path provides OS-specific path resolution per AI.md PART 4
package path

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	projectOrg  = "casjay-forks"
	projectName = "caspb"
)

// IsRoot returns true if running as root/Administrator
func IsRoot() bool {
	return os.Geteuid() == 0
}

// IsDocker returns true if running inside a Docker container
func IsDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

// ConfigDir returns the configuration directory
func ConfigDir() string {
	// Docker paths
	if IsDocker() {
		return filepath.Join("/config", projectName)
	}

	switch runtime.GOOS {
	case "windows":
		if IsRoot() {
			return filepath.Join(os.Getenv("ProgramData"), projectOrg, projectName)
		}
		return filepath.Join(os.Getenv("APPDATA"), projectOrg, projectName)
	case "darwin":
		if IsRoot() {
			return filepath.Join("/Library/Application Support", projectOrg, projectName)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library/Application Support", projectOrg, projectName)
	default:
		// Linux, BSD
		if IsRoot() {
			return filepath.Join("/etc", projectOrg, projectName)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", projectOrg, projectName)
	}
}

// DataDir returns the data directory
func DataDir() string {
	// Docker paths
	if IsDocker() {
		return filepath.Join("/data", projectName)
	}

	switch runtime.GOOS {
	case "windows":
		if IsRoot() {
			return filepath.Join(os.Getenv("ProgramData"), projectOrg, projectName, "data")
		}
		return filepath.Join(os.Getenv("LOCALAPPDATA"), projectOrg, projectName)
	case "darwin":
		if IsRoot() {
			return filepath.Join("/Library/Application Support", projectOrg, projectName, "data")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library/Application Support", projectOrg, projectName)
	default:
		// Linux, BSD
		if IsRoot() {
			return filepath.Join("/var/lib", projectOrg, projectName)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local/share", projectOrg, projectName)
	}
}

// CacheDir returns the cache directory
func CacheDir() string {
	// Docker paths
	if IsDocker() {
		return filepath.Join("/data", projectName, "cache")
	}

	switch runtime.GOOS {
	case "windows":
		if IsRoot() {
			return filepath.Join(os.Getenv("ProgramData"), projectOrg, projectName, "cache")
		}
		return filepath.Join(os.Getenv("LOCALAPPDATA"), projectOrg, projectName, "cache")
	case "darwin":
		if IsRoot() {
			return filepath.Join("/Library/Caches", projectOrg, projectName)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library/Caches", projectOrg, projectName)
	default:
		// Linux, BSD
		if IsRoot() {
			return filepath.Join("/var/cache", projectOrg, projectName)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".cache", projectOrg, projectName)
	}
}

// LogDir returns the log directory
func LogDir() string {
	// Docker paths
	if IsDocker() {
		return filepath.Join("/data/log", projectName)
	}

	switch runtime.GOOS {
	case "windows":
		if IsRoot() {
			return filepath.Join(os.Getenv("ProgramData"), projectOrg, projectName, "logs")
		}
		return filepath.Join(os.Getenv("LOCALAPPDATA"), projectOrg, projectName, "logs")
	case "darwin":
		if IsRoot() {
			return filepath.Join("/Library/Logs", projectOrg, projectName)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library/Logs", projectOrg, projectName)
	default:
		// Linux, BSD
		if IsRoot() {
			return filepath.Join("/var/log", projectOrg, projectName)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local/log", projectOrg, projectName)
	}
}

// BackupDir returns the backup directory
func BackupDir() string {
	// Docker paths
	if IsDocker() {
		return filepath.Join("/data/backups", projectName)
	}

	switch runtime.GOOS {
	case "windows":
		if IsRoot() {
			return filepath.Join(os.Getenv("ProgramData"), "Backups", projectOrg, projectName)
		}
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Backups", projectOrg, projectName)
	case "darwin":
		if IsRoot() {
			return filepath.Join("/Library/Backups", projectOrg, projectName)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library/Backups", projectOrg, projectName)
	default:
		// Linux, BSD
		if IsRoot() {
			return filepath.Join("/mnt/Backups", projectOrg, projectName)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local/share/Backups", projectOrg, projectName)
	}
}

// DbDir returns the database directory
func DbDir() string {
	// Docker paths
	if IsDocker() {
		return "/data/db/sqlite"
	}

	switch runtime.GOOS {
	case "windows":
		if IsRoot() {
			return filepath.Join(os.Getenv("ProgramData"), projectOrg, projectName, "db")
		}
		return filepath.Join(os.Getenv("LOCALAPPDATA"), projectOrg, projectName, "db")
	case "darwin":
		if IsRoot() {
			return filepath.Join("/Library/Application Support", projectOrg, projectName, "db")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library/Application Support", projectOrg, projectName, "db")
	default:
		// Linux, BSD
		if IsRoot() {
			return filepath.Join("/var/lib", projectOrg, projectName, "db")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local/share", projectOrg, projectName, "db")
	}
}

// SSLDir returns the SSL/TLS certificates directory
func SSLDir() string {
	// Docker paths
	if IsDocker() {
		return filepath.Join("/config", projectName, "ssl")
	}

	switch runtime.GOOS {
	case "windows":
		if IsRoot() {
			return filepath.Join(os.Getenv("ProgramData"), projectOrg, projectName, "ssl")
		}
		return filepath.Join(os.Getenv("APPDATA"), projectOrg, projectName, "ssl")
	case "darwin":
		if IsRoot() {
			return filepath.Join("/Library/Application Support", projectOrg, projectName, "ssl")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library/Application Support", projectOrg, projectName, "ssl")
	default:
		// Linux, BSD
		if IsRoot() {
			return filepath.Join("/etc", projectOrg, projectName, "ssl")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", projectOrg, projectName, "ssl")
	}
}

// SecurityDir returns the security databases directory (GeoIP, blocklists, etc.)
func SecurityDir() string {
	// Docker paths
	if IsDocker() {
		return filepath.Join("/config", projectName, "security")
	}

	switch runtime.GOOS {
	case "windows":
		if IsRoot() {
			return filepath.Join(os.Getenv("ProgramData"), projectOrg, projectName, "security")
		}
		return filepath.Join(os.Getenv("APPDATA"), projectOrg, projectName, "security")
	case "darwin":
		if IsRoot() {
			return filepath.Join("/Library/Application Support", projectOrg, projectName, "security")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library/Application Support", projectOrg, projectName, "security")
	default:
		// Linux, BSD
		if IsRoot() {
			return filepath.Join("/etc", projectOrg, projectName, "security")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", projectOrg, projectName, "security")
	}
}

// PIDFile returns the PID file path
func PIDFile() string {
	switch runtime.GOOS {
	case "windows":
		// Windows doesn't typically use PID files
		return ""
	case "darwin":
		if IsRoot() {
			return filepath.Join("/var/run", projectOrg, projectName+".pid")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library/Application Support", projectOrg, projectName, projectName+".pid")
	default:
		// Linux, BSD
		if IsRoot() {
			return filepath.Join("/var/run", projectOrg, projectName+".pid")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local/share", projectOrg, projectName, projectName+".pid")
	}
}

// ConfigFile returns the server config file path
func ConfigFile() string {
	return filepath.Join(ConfigDir(), "server.yml")
}

// LogFile returns the server log file path
func LogFile() string {
	return filepath.Join(LogDir(), "server.log")
}

// AuditLogFile returns the audit log file path
func AuditLogFile() string {
	return filepath.Join(LogDir(), "audit.log")
}

// BinaryDir returns the binary installation directory
func BinaryDir() string {
	switch runtime.GOOS {
	case "windows":
		if IsRoot() {
			return filepath.Join(os.Getenv("ProgramFiles"), projectOrg, projectName)
		}
		return filepath.Join(os.Getenv("LOCALAPPDATA"), projectOrg, projectName)
	default:
		// Linux, macOS, BSD
		if IsRoot() {
			return "/usr/local/bin"
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local/bin")
	}
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// EnsureAllDirs creates all required directories
func EnsureAllDirs() error {
	dirs := []string{
		ConfigDir(),
		DataDir(),
		CacheDir(),
		LogDir(),
		DbDir(),
		SSLDir(),
		SecurityDir(),
	}

	for _, dir := range dirs {
		if err := EnsureDir(dir); err != nil {
			return err
		}
	}
	return nil
}

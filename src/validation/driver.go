// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package validation

import (
	"fmt"
	"strings"
)

// DetectDriver auto-detects database driver from connection string.
// Supports: sqlite:///path, postgres://..., mysql://..., mariadb://..., libsql://...
// Returns normalized driver name (mariadb → mysql, sqlite3 → sqlite).
func DetectDriver(source string) (string, error) {
	source = strings.ToLower(strings.TrimSpace(source))

	// SQLite variants
	if strings.HasPrefix(source, "sqlite://") {
		return "sqlite", nil
	}
	if strings.HasPrefix(source, "sqlite3://") {
		return "sqlite", nil
	}

	// PostgreSQL
	if strings.HasPrefix(source, "postgres://") || strings.HasPrefix(source, "postgresql://") {
		return "postgres", nil
	}

	// MySQL
	if strings.HasPrefix(source, "mysql://") {
		return "mysql", nil
	}

	// MariaDB (uses MySQL driver)
	if strings.HasPrefix(source, "mariadb://") {
		return "mysql", nil
	}

	// libSQL (Turso) — local file or remote
	if strings.HasPrefix(source, "libsql://") || strings.HasPrefix(source, "file:") {
		return "libsql", nil
	}

	// If no scheme, check if it looks like a file path (SQLite)
	if strings.Contains(source, "/") || strings.HasSuffix(source, ".db") {
		return "sqlite", nil
	}

	return "", fmt.Errorf("could not detect database driver from source: %s", source)
}

// NormalizeDriver normalizes driver names.
// mariadb → mysql, sqlite3 → sqlite, postgresql → postgres.
func NormalizeDriver(driver string) string {
	driver = strings.ToLower(driver)
	if driver == "mariadb" || driver == "maria" {
		return "mysql"
	}
	if driver == "sqlite3" {
		return "sqlite"
	}
	if driver == "postgresql" {
		return "postgres"
	}
	if driver == "turso" {
		return "libsql"
	}
	return driver
}

// NormalizeConnectionString normalizes connection strings to driver-specific format.
// - sqlite://path → /path
// - mysql://user:pass@host:port/db → user:pass@tcp(host:port)/db
// - postgres://... stays as-is (pgx driver supports it)
// - libsql://... stays as-is (libsql driver handles its own URL parsing)
func NormalizeConnectionString(driver, source string) string {
	// libSQL: pass through; the driver parses libsql:// and file: URLs natively
	if driver == "libsql" {
		return source
	}

	// SQLite: Remove sqlite:// prefix
	if driver == "sqlite" {
		if strings.HasPrefix(source, "sqlite://") {
			return strings.TrimPrefix(source, "sqlite://")
		}
		if strings.HasPrefix(source, "sqlite3://") {
			return strings.TrimPrefix(source, "sqlite3://")
		}
		return source
	}

	// PostgreSQL: postgres:// URLs are supported by pgx driver, keep as-is
	if driver == "postgres" {
		return source
	}

	// MySQL/MariaDB: Convert mysql://user:pass@host:port/db to user:pass@tcp(host:port)/db
	// The go-sql-driver/mysql expects tcp() format, not URL format
	if driver == "mysql" {
		// Convert mysql://user:pass@host:port/db → user:pass@tcp(host:port)/db
		if strings.HasPrefix(source, "mysql://") || strings.HasPrefix(source, "mariadb://") {
			source = strings.TrimPrefix(source, "mysql://")
			source = strings.TrimPrefix(source, "mariadb://")

			// Parse: user:pass@host:port/db
			if strings.Contains(source, "@") {
				parts := strings.SplitN(source, "@", 2)
				userPass := parts[0]
				rest := parts[1]

				// Extract host:port and /db
				if strings.Contains(rest, "/") {
					hostParts := strings.SplitN(rest, "/", 2)
					hostPort := hostParts[0]
					dbname := "/" + hostParts[1]

					// Return as: user:pass@tcp(host:port)/db
					return userPass + "@tcp(" + hostPort + ")" + dbname
				}
			}
		}

		return source
	}

	return source
}

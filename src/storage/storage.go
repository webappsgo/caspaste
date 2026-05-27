
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Timeout for DDL operations during initialization (longer than query timeout)
const initializationTimeout = 30 * time.Second

var (
	ErrNotFoundID = errors.New("db: could not find ID")
)

type DB struct {
	pool       *sql.DB
	backupPool *sql.DB // SQLite backup/cache when using postgres/mysql
	driver     string
}

func NewPool(driverName string, dataSourceName string, maxOpenConns int, maxIdleConns int, dataDir string) (DB, error) {
	var db DB
	var err error

	db.driver = driverName
	// pgx/v5/stdlib registers itself as "pgx", not "postgres"
	sqlDriverName := driverName
	if driverName == "postgres" {
		sqlDriverName = "pgx"
	}
	db.pool, err = sql.Open(sqlDriverName, dataSourceName)
	if err != nil {
		return db, err
	}

	db.pool.SetMaxOpenConns(maxOpenConns)
	db.pool.SetMaxIdleConns(maxIdleConns)

	// Set connection lifetime and idle timeouts to prevent stale connections
	db.pool.SetConnMaxLifetime(3600 * 1000000000) // 1 hour in nanoseconds
	db.pool.SetConnMaxIdleTime(600 * 1000000000)  // 10 minutes in nanoseconds

	// If using postgres/mysql, also open SQLite backup/cache
	// SQLite cache is ALWAYS required for local operations
	if driverName == "postgres" || driverName == "mysql" || driverName == "mariadb" {
		// Determine SQLite cache path - check env var first, then use standard path
		backupPath := getSQLiteCachePath(dataDir)

		db.backupPool, err = sql.Open("sqlite", backupPath)
		if err != nil {
			// Don't fail if backup can't be opened, just log warning
			db.backupPool = nil
		} else {
			db.backupPool.SetMaxOpenConns(10)
			db.backupPool.SetMaxIdleConns(2)
			db.backupPool.SetConnMaxLifetime(3600 * 1000000000)
			db.backupPool.SetConnMaxIdleTime(600 * 1000000000)
			// Initialize backup database schema
			InitDB("sqlite", backupPath)
		}
	}

	return db, nil
}

// getSQLiteCachePath determines the SQLite cache database path
// Priority: CASPASTE_DB_DIR env var > dataDir/db/ > platform-specific default
func getSQLiteCachePath(dataDir string) string {
	// Check environment variable first
	if envDbDir := os.Getenv("CASPASTE_DB_DIR"); envDbDir != "" {
		return envDbDir + "/caspaste.db"
	}
	// Use data directory if provided
	if dataDir != "" {
		return dataDir + "/db/caspaste.db"
	}
	// Fallback to platform-specific default
	return getDefaultDbPath()
}

// getDefaultDbPath returns the platform-specific default database path
func getDefaultDbPath() string {
	switch runtime.GOOS {
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return localAppData + "\\CasPaste\\Data\\db\\caspaste.db"
		}
		return os.Getenv("PROGRAMDATA") + "\\CasPaste\\Data\\db\\caspaste.db"
	case "darwin":
		if isRunningAsRoot() {
			return "/var/lib/casjay-forks/caspaste/db/caspaste.db"
		}
		if home := os.Getenv("HOME"); home != "" {
			return home + "/Library/Application Support/CasPaste/db/caspaste.db"
		}
		return "/var/lib/casjay-forks/caspaste/db/caspaste.db"
	// Linux, BSD, etc.
	default:
		if isRunningAsRoot() {
			return "/var/lib/casjay-forks/caspaste/db/caspaste.db"
		}
		if home := os.Getenv("HOME"); home != "" {
			return home + "/.local/share/casjay-forks/caspaste/db/caspaste.db"
		}
		return "/var/lib/casjay-forks/caspaste/db/caspaste.db"
	}
}

// isRunningAsRoot checks if the process is running with root/admin privileges
func isRunningAsRoot() bool {
	return os.Geteuid() == 0
}

func (db DB) Close() error {
	// Close backup pool first if it exists
	if db.backupPool != nil {
		if err := db.backupPool.Close(); err != nil {
			// Log but don't fail on backup close error
			// Continue to close primary pool
		}
	}
	return db.pool.Close()
}

func InitDB(driverName string, dataSourceName string) error {
	// Open DB
	db, err := NewPool(driverName, dataSourceName, 1, 0, "")
	if err != nil {
		return err
	}
	defer db.Close()

	// Create context for all DDL operations
	ctx, cancel := context.WithTimeout(context.Background(), initializationTimeout)
	defer cancel()

	// Create pastes table
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS pastes (
			id          TEXT    PRIMARY KEY,
			title       TEXT    NOT NULL,
			body        TEXT    NOT NULL,
			syntax      TEXT    NOT NULL,
			create_time INTEGER NOT NULL,
			delete_time INTEGER NOT NULL,
			one_use     BOOL    NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	// Create admins table (Server Admins - REQUIRED per AI.md PART 11)
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS admins (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			username        TEXT NOT NULL UNIQUE,
			password_hash   TEXT NOT NULL,
			email           TEXT,
			role            TEXT NOT NULL DEFAULT 'admin',
			enabled         INTEGER NOT NULL DEFAULT 1,
			api_token_hash  TEXT,
			created_at      INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			updated_at      INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			last_login      INTEGER,
			failed_attempts INTEGER NOT NULL DEFAULT 0,
			locked_until    INTEGER,
			source          TEXT NOT NULL DEFAULT 'local',
			external_id     TEXT,
			groups          TEXT,
			last_sync       INTEGER
		);
	`)
	if err != nil {
		return err
	}

	// Create admin_preferences table
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS admin_preferences (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			admin_id   INTEGER NOT NULL UNIQUE,
			theme      TEXT DEFAULT 'dark',
			language   TEXT DEFAULT 'en',
			timezone   TEXT,
			dashboard_layout TEXT,
			notifications TEXT,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create users table (PART 34: Multi-User)
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			username        TEXT NOT NULL UNIQUE,
			email           TEXT NOT NULL UNIQUE,
			password_hash   TEXT NOT NULL,
			display_name    TEXT,
			avatar_type     TEXT NOT NULL DEFAULT 'gravatar',
			avatar_url      TEXT,
			bio             TEXT,
			location        TEXT,
			website         TEXT,
			visibility      TEXT NOT NULL DEFAULT 'public',
			org_visibility  INTEGER NOT NULL DEFAULT 1,
			timezone        TEXT,
			language        TEXT DEFAULT 'en',
			role            TEXT NOT NULL DEFAULT 'user',
			email_verified  INTEGER NOT NULL DEFAULT 0,
			totp_enabled    INTEGER NOT NULL DEFAULT 0,
			totp_secret     TEXT,
			last_login      INTEGER,
			failed_attempts INTEGER NOT NULL DEFAULT 0,
			locked_until    INTEGER,
			created_at      INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			updated_at      INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
		);
	`)
	if err != nil {
		return err
	}

	// Create user_sessions table
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS user_sessions (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id     INTEGER NOT NULL,
			token_hash  TEXT NOT NULL UNIQUE,
			device      TEXT,
			ip_address  TEXT,
			user_agent  TEXT,
			expires_at  INTEGER NOT NULL,
			created_at  INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create user_tokens table (API tokens with usr_ prefix)
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS user_tokens (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id      INTEGER NOT NULL,
			name         TEXT NOT NULL,
			token_prefix TEXT NOT NULL,
			token_hash   TEXT NOT NULL UNIQUE,
			scopes       TEXT,
			last_used_at INTEGER,
			expires_at   INTEGER,
			created_at   INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create recovery_keys table (hashed, single use)
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS recovery_keys (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL,
			key_hash   TEXT NOT NULL UNIQUE,
			used_at    INTEGER,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create password_resets table
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS password_resets (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at INTEGER NOT NULL,
			used_at    INTEGER,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create email_verifications table
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS email_verifications (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id     INTEGER NOT NULL,
			email       TEXT NOT NULL,
			token_hash  TEXT NOT NULL UNIQUE,
			expires_at  INTEGER NOT NULL,
			verified_at INTEGER,
			created_at  INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create user_invites table (admin-generated)
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS user_invites (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			username   TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			created_by INTEGER NOT NULL,
			expires_at INTEGER NOT NULL,
			used_at    INTEGER,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
		);
	`)
	if err != nil {
		return err
	}

	// Create user_preferences table
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS user_preferences (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id          INTEGER NOT NULL UNIQUE,
			show_email       INTEGER NOT NULL DEFAULT 0,
			show_activity    INTEGER NOT NULL DEFAULT 1,
			show_orgs        INTEGER NOT NULL DEFAULT 1,
			searchable       INTEGER NOT NULL DEFAULT 1,
			email_security   INTEGER NOT NULL DEFAULT 1,
			email_mentions   INTEGER NOT NULL DEFAULT 1,
			email_updates    INTEGER NOT NULL DEFAULT 0,
			email_digest     TEXT DEFAULT 'weekly',
			theme            TEXT DEFAULT 'dark',
			font_size        TEXT DEFAULT 'medium',
			reduce_motion    INTEGER NOT NULL DEFAULT 0,
			date_format      TEXT DEFAULT 'YYYY-MM-DD',
			time_format      TEXT DEFAULT '24h',
			created_at       INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			updated_at       INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create orgs table (PART 35: Organizations)
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS orgs (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			slug           TEXT NOT NULL UNIQUE,
			name           TEXT NOT NULL,
			description    TEXT,
			avatar_type    TEXT NOT NULL DEFAULT 'gravatar',
			avatar_url     TEXT,
			website        TEXT,
			location       TEXT,
			visibility     TEXT NOT NULL DEFAULT 'public',
			owner_id       INTEGER NOT NULL,
			email          TEXT,
			email_verified INTEGER NOT NULL DEFAULT 0,
			created_at     INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			updated_at     INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (owner_id) REFERENCES users(id)
		);
	`)
	if err != nil {
		return err
	}

	// Create org_members table
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS org_members (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id     INTEGER NOT NULL,
			user_id    INTEGER NOT NULL,
			role       TEXT NOT NULL DEFAULT 'member',
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			UNIQUE(org_id, user_id),
			FOREIGN KEY (org_id) REFERENCES orgs(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create org_tokens table (API tokens with org_ prefix)
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS org_tokens (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id       INTEGER NOT NULL,
			created_by   INTEGER NOT NULL,
			name         TEXT NOT NULL,
			token_prefix TEXT NOT NULL,
			token_hash   TEXT NOT NULL UNIQUE,
			scopes       TEXT,
			last_used_at INTEGER,
			expires_at   INTEGER,
			created_at   INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (org_id) REFERENCES orgs(id) ON DELETE CASCADE,
			FOREIGN KEY (created_by) REFERENCES users(id)
		);
	`)
	if err != nil {
		return err
	}

	// Create org_preferences table
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS org_preferences (
			id                   INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id               INTEGER NOT NULL UNIQUE,
			default_role         TEXT DEFAULT 'member',
			require_2fa          INTEGER NOT NULL DEFAULT 0,
			notify_member_join   INTEGER NOT NULL DEFAULT 1,
			notify_member_leave  INTEGER NOT NULL DEFAULT 1,
			notify_role_change   INTEGER NOT NULL DEFAULT 1,
			notify_token_activity INTEGER NOT NULL DEFAULT 1,
			created_at           INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			updated_at           INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (org_id) REFERENCES orgs(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create custom_domains table (PART 36: Custom Domains)
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS custom_domains (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			owner_type          TEXT NOT NULL,
			owner_id            INTEGER NOT NULL,
			domain              TEXT NOT NULL UNIQUE,
			is_apex             INTEGER NOT NULL DEFAULT 0,
			is_wildcard         INTEGER NOT NULL DEFAULT 0,
			verification_status TEXT NOT NULL DEFAULT 'pending',
			verified_at         INTEGER,
			verified_ip         TEXT,
			last_check_at       INTEGER,
			check_count         INTEGER NOT NULL DEFAULT 0,
			ssl_enabled         INTEGER NOT NULL DEFAULT 0,
			ssl_status          TEXT NOT NULL DEFAULT 'none',
			ssl_challenge       TEXT,
			ssl_provider        TEXT,
			ssl_credentials     TEXT,
			ssl_cert_pem        TEXT,
			ssl_key_pem         TEXT,
			ssl_issued_at       INTEGER,
			ssl_expires_at      INTEGER,
			ssl_last_error      TEXT,
			status              TEXT NOT NULL DEFAULT 'pending',
			suspended_reason    TEXT,
			created_at          INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			updated_at          INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
		);
	`)
	if err != nil {
		return err
	}

	// Create custom_domain_audit table
	_, err = db.pool.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS custom_domain_audit (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			domain_id  INTEGER NOT NULL,
			action     TEXT NOT NULL,
			actor_type TEXT NOT NULL,
			actor_id   INTEGER,
			details    TEXT,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (domain_id) REFERENCES custom_domains(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_admins_username ON admins(username);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_admins_token ON admins(api_token_hash);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(token_hash);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_user_tokens_user ON user_tokens(user_id);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_recovery_keys_user ON recovery_keys(user_id);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_orgs_slug ON orgs(slug);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_orgs_owner ON orgs(owner_id);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_org_members_org ON org_members(org_id);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_org_members_user ON org_members(user_id);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_custom_domains_domain ON custom_domains(domain);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_custom_domains_owner ON custom_domains(owner_type, owner_id);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_custom_domains_status ON custom_domains(status);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_custom_domains_ssl_expires ON custom_domains(ssl_expires_at);`)
	_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_domain_audit_domain ON custom_domain_audit(domain_id);`)

	// Handle database-specific column additions for pastes table
	// Define allowed columns with validation (prevents SQL injection)
	type columnDef struct {
		name       string
		definition string
	}

	var columns []columnDef
	if driverName == "sqlite3" || driverName == "sqlite" {
		// SQLite: ALTER TABLE ADD COLUMN (ignores duplicate errors)
		columns = []columnDef{
			{"author", "TEXT NOT NULL DEFAULT ''"},
			{"author_email", "TEXT NOT NULL DEFAULT ''"},
			{"author_url", "TEXT NOT NULL DEFAULT ''"},
			{"is_file", "BOOL NOT NULL DEFAULT 0"},
			{"file_name", "TEXT NOT NULL DEFAULT ''"},
			{"mime_type", "TEXT NOT NULL DEFAULT ''"},
			{"is_editable", "BOOL NOT NULL DEFAULT 0"},
			{"is_private", "BOOL NOT NULL DEFAULT 0"},
			{"is_url", "BOOL NOT NULL DEFAULT 0"},
			{"original_url", "TEXT NOT NULL DEFAULT ''"},
			{"user_id", "INTEGER"},
			{"org_id", "INTEGER"},
		}
		for _, col := range columns {
			// Using string formatting is safe here because column name is from hardcoded whitelist
			_, err := db.pool.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE pastes ADD COLUMN %s %s`, col.name, col.definition))
			// Ignore "duplicate column" errors
			if err != nil && !strings.Contains(err.Error(), "duplicate column") {
				return err
			}
		}

		// Create indexes for pastes user/org columns
		_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_pastes_user ON pastes(user_id);`)
		_, _ = db.pool.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_pastes_org ON pastes(org_id);`)

	} else if driverName == "mysql" || driverName == "mariadb" {
		// MySQL/MariaDB: Use ALTER TABLE ADD COLUMN IF NOT EXISTS (MariaDB 10.0+)
		columns = []columnDef{
			{"author", "TEXT NOT NULL DEFAULT ''"},
			{"author_email", "TEXT NOT NULL DEFAULT ''"},
			{"author_url", "TEXT NOT NULL DEFAULT ''"},
			{"is_file", "BOOLEAN NOT NULL DEFAULT false"},
			{"file_name", "TEXT NOT NULL DEFAULT ''"},
			{"mime_type", "TEXT NOT NULL DEFAULT ''"},
			{"is_editable", "BOOLEAN NOT NULL DEFAULT false"},
			{"is_private", "BOOLEAN NOT NULL DEFAULT false"},
			{"is_url", "BOOLEAN NOT NULL DEFAULT false"},
			{"original_url", "TEXT NOT NULL DEFAULT ''"},
			{"user_id", "INTEGER"},
			{"org_id", "INTEGER"},
		}
		for _, col := range columns {
			// Using string formatting is safe here because column name is from hardcoded whitelist
			_, err := db.pool.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE pastes ADD COLUMN IF NOT EXISTS %s %s`, col.name, col.definition))
			if err != nil {
				return err
			}
		}

	} else {
		// PostgreSQL: supports IF NOT EXISTS
		_, err = db.pool.ExecContext(ctx, `
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS author       TEXT NOT NULL DEFAULT '';
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS author_email TEXT NOT NULL DEFAULT '';
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS author_url   TEXT NOT NULL DEFAULT '';
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS is_file      BOOL NOT NULL DEFAULT false;
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS file_name    TEXT NOT NULL DEFAULT '';
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS mime_type    TEXT NOT NULL DEFAULT '';
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS is_editable  BOOL NOT NULL DEFAULT false;
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS is_private   BOOL NOT NULL DEFAULT false;
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS is_url       BOOL NOT NULL DEFAULT false;
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS original_url TEXT NOT NULL DEFAULT '';
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS user_id      INTEGER;
			ALTER TABLE pastes ADD COLUMN IF NOT EXISTS org_id       INTEGER;
		`)
		if err != nil {
			return err
		}
	}

	return nil
}

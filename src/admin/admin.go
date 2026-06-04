
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package admin provides the admin panel UI and API per AI.md PART 17.
// Mount point: /server/{admin_path}/   (UI)
// API mount:   /api/{version}/server/{admin_path}/
//
// Route hierarchy (relative to strip-prefix base):
//   /                               Dashboard
//   /login                          Login page + form POST
//   /logout                         Logout (GET clears session)
//   /{admin_username}/profile        Admin self-management
//   /{admin_username}/preferences    Admin preferences
//   /{admin_username}/notifications  Admin notifications
//   /config/setup                   First-run setup wizard
//   /config/settings                Server settings
//   /config/ssl                     SSL/TLS management
//   /config/email                   Email / SMTP settings
//   /config/scheduler               Scheduled task viewer
//   /config/logs                    Server log viewer
//   /config/logs/audit              Audit log viewer
//   /config/backup                  Backup & restore
//   /config/updates                 Update management
//   /config/info                    Server info
//   /config/metrics                 Metrics dashboard
//   /config/network/tor             Tor hidden service
//   /config/network/geoip           GeoIP settings
//   /config/security/auth           Authentication overview
//   /config/security/auth/oidc      OIDC provider management
//   /config/security/auth/ldap      LDAP provider management
//   /config/security/tokens         API token management
//   /config/security/firewall       Firewall rules
package admin

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/casjay-forks/caspaste/src/config"
	"github.com/casjay-forks/caspaste/src/scheduler"
)

// Panel represents the fully-initialized admin panel
type Panel struct {
	cfg         *Config
	db          *sql.DB
	sched       *scheduler.Scheduler
	debug       bool
	setupDone   bool
	setupToken  string
	setupExpiry time.Time
	mu          sync.RWMutex
}

// Config holds admin panel startup configuration
type Config struct {
	// BasePath is the URL path segment (default: "admin")
	BasePath string
	// APIVersion is the API version prefix (default: "v1")
	APIVersion string
	// Enabled controls whether the admin panel responds to requests
	Enabled bool
	// DB is the application database pool (users.db / admins table)
	DB *sql.DB
	// Debug bypasses admin authentication per AI.md PART 6
	Debug bool
	// StartTime is used to calculate server uptime on the dashboard
	StartTime time.Time
	// AppCfg is the loaded application config — used for read-only display on info/settings pages
	AppCfg *config.Config
	// ConfigFile is the path to the YAML config file
	ConfigFile string
	// DataDir is the data directory path
	DataDir string
	// ConfigDir is the config directory path
	ConfigDir string
	// BackupDir is the backup directory path
	BackupDir string
}

// DefaultConfig returns the default admin panel configuration
func DefaultConfig() *Config {
	return &Config{
		BasePath:   "admin",
		APIVersion: "v1",
		Enabled:    true,
		StartTime:  time.Now(),
	}
}

// ValidAdminRootPaths are the only valid direct children of the stripped admin prefix.
// Any other segment is treated as {admin_username} for self-management routes.
var ValidAdminRootPaths = map[string]bool{
	"":       true,
	"config": true,
	"login":  true,
	"logout": true,
}

// ReservedPaths cannot be used as the admin_path value
var ReservedPaths = []string{
	"api", "static", "assets", "health", "healthz", "version",
	"metrics", ".well-known", "graphql", "openapi",
	"auth", "security", "docs", "about", "privacy", "contact", "help", "terms",
}

// New creates a new admin Panel
func New(cfg *Config) *Panel {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.StartTime.IsZero() {
		cfg.StartTime = time.Now()
	}
	return &Panel{
		cfg:   cfg,
		db:    cfg.DB,
		debug: cfg.Debug,
	}
}

// SetScheduler injects the scheduler after Panel creation (scheduler is started after routing is wired)
func (p *Panel) SetScheduler(s *scheduler.Scheduler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sched = s
}

// adminBasePath returns the full server-relative UI base path (e.g. "/server/admin")
func (p *Panel) adminBasePath() string {
	return "/server/" + p.cfg.BasePath
}

// apiBasePath returns the full server-relative API base path (e.g. "/api/v1/server/admin")
func (p *Panel) apiBasePath() string {
	return "/api/" + p.cfg.APIVersion + "/server/" + p.cfg.BasePath
}

// errNoDB is returned when the panel has no database
var errNoDB = fmt.Errorf("admin: no database configured")

// ValidateAdminPath validates a candidate admin_path value per AI.md PART 17 rules
func ValidateAdminPath(path string) error {
	path = strings.ToLower(strings.TrimSpace(path))
	if len(path) < 2 || len(path) > 32 {
		return fmt.Errorf("admin path must be 2-32 characters")
	}
	for _, c := range path {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return fmt.Errorf("admin path can only contain lowercase letters, numbers, and hyphens")
		}
	}
	if path[0] == '-' || path[len(path)-1] == '-' {
		return fmt.Errorf("admin path cannot start or end with a hyphen")
	}
	for _, reserved := range ReservedPaths {
		if path == reserved {
			return fmt.Errorf("'%s' is a reserved path", path)
		}
	}
	return nil
}

// uptime returns human-readable uptime string
func (p *Panel) uptime() string {
	d := time.Since(p.cfg.StartTime)
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

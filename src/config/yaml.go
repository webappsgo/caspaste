
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the YAML configuration file structure
// All configuration is organized into logical top-level sections
type YAMLConfig struct {
	Server struct {
		// Public instance (default: true = no auth, false = auth required)
		Public bool `yaml:"public"`
		// Public FQDN for building URLs (empty=auto-detect from headers/hostname, set to override)
		FQDN string `yaml:"fqdn"`
		// Listen address (all, ::, 0.0.0.0, specific IP)
		Listen string `yaml:"listen"`
		// Port number (empty=auto-detect available port)
		Port string `yaml:"port"`
		// Server title
		Title string `yaml:"title"`
		// Server tagline (short description)
		TagLine string `yaml:"tagline"`
		// Server description (longer description for meta tags)
		Description string `yaml:"description"`

		// API compatibility mode: native (default), lenpaste, stikked, microbin,
		// hastebin, pastebin, termbin. Also set via CASPASTE_API_MODE env var.
		// When empty, mode is auto-detected per-request from the Host header.
		APIMode string `yaml:"api_mode"`

		Proxy struct {
			// Additional trusted proxy IPs/CIDRs (appended to default private ranges)
			Allowed []string `yaml:"allowed"`
		} `yaml:"proxy"`

		Administrator struct {
			// Admin name
			Name string `yaml:"name"`
			// Admin email
			Email string `yaml:"email"`
			// Email from address
			From string `yaml:"from"`
		} `yaml:"administrator"`

		Timeouts struct {
			// Read timeout in seconds (default: 15)
			Read int `yaml:"read"`
			// Write timeout in seconds (default: 15)
			Write int `yaml:"write"`
			// Idle timeout in seconds (default: 60)
			Idle int `yaml:"idle"`
		} `yaml:"timeouts"`

		// Prometheus metrics per AI.md PART 21
		Metrics struct {
			// Enable Prometheus metrics endpoint (default: false)
			Enabled bool `yaml:"enabled"`
			// Endpoint path (default: /metrics)
			Endpoint string `yaml:"endpoint"`
			// Include system metrics (CPU, memory, disk)
			IncludeSystem bool `yaml:"include_system"`
			// Include Go runtime metrics
			IncludeRuntime bool `yaml:"include_runtime"`
			// Optional bearer token for authentication
			Token string `yaml:"token"`
			// Histogram buckets for request duration (seconds)
			DurationBuckets []float64 `yaml:"duration_buckets"`
			// Histogram buckets for request/response size (bytes)
			SizeBuckets []float64 `yaml:"size_buckets"`
		} `yaml:"metrics"`

		// Tor hidden service per AI.md PART 32
		// Auto-enabled when Tor binary is found - no enable/disable toggle
		Tor struct {
			// Path to Tor binary (empty = auto-detect)
			Binary string `yaml:"binary"`
			// Use Tor network for outbound connections (server-wide default)
			UseNetwork bool `yaml:"use_network"`
			// Allow users to set their own Tor network preference
			AllowUserPreference bool `yaml:"allow_user_preference"`
			// Maximum circuits to keep open (1-128, default: 32)
			MaxCircuits int `yaml:"max_circuits"`
			// Circuit timeout in seconds (10-300, default: 60)
			CircuitTimeout int `yaml:"circuit_timeout"`
			// Bootstrap timeout in seconds (30-600, default: 180)
			BootstrapTimeout int `yaml:"bootstrap_timeout"`
			// Scrub sensitive info from Tor logs
			SafeLogging bool `yaml:"safe_logging"`
			// Maximum concurrent streams per circuit (10-500, default: 100)
			MaxStreamsPerCircuit int `yaml:"max_streams_per_circuit"`
			// Close circuit when stream limit exceeded
			CloseCircuitOnStreamLimit bool `yaml:"close_circuit_on_stream_limit"`
			// Bandwidth rate per second (e.g., "1 MB", "500 KB")
			BandwidthRate string `yaml:"bandwidth_rate"`
			// Bandwidth burst per second (e.g., "2 MB", "1 MB")
			BandwidthBurst string `yaml:"bandwidth_burst"`
			// Maximum monthly bandwidth (e.g., "100 GB", "unlimited")
			MaxMonthlyBandwidth string `yaml:"max_monthly_bandwidth"`
			// Number of introduction points (3-10, default: 3)
			NumIntroPoints int `yaml:"num_intro_points"`
			// Virtual port for hidden service (default: 80)
			VirtualPort int `yaml:"virtual_port"`
		} `yaml:"tor"`
	} `yaml:"server"`

	Database struct {
		// sqlite, postgres, mysql
		Driver string `yaml:"driver"`
		// Connection string
		Source string `yaml:"source"`
		// Max open connections
		MaxOpenConns int `yaml:"max_open_conns"`
		// Max idle connections
		MaxIdleConns int `yaml:"max_idle_conns"`
		// Cleanup interval (e.g. "1m", "5m")
		CleanupPeriod string `yaml:"cleanup_period"`
	} `yaml:"database"`

	Security struct {
		// Path to password file (auto-generated when server.public=false)
		PasswordFile string `yaml:"password_file"`

		Headers struct {
			// X-Frame-Options header
			XFrameOptions string `yaml:"x_frame_options"`
			// X-Content-Type-Options header
			XContentTypeOptions string `yaml:"x_content_type_options"`
			// X-XSS-Protection header (deprecated but kept per AI.md)
			XSSProtection string `yaml:"xss_protection"`
			// Content-Security-Policy header
			ContentSecurityPolicy string `yaml:"content_security_policy"`
			// Referrer-Policy header
			ReferrerPolicy string `yaml:"referrer_policy"`
			// Permissions-Policy header
			PermissionsPolicy string `yaml:"permissions_policy"`
			// Strict-Transport-Security header
			StrictTransportSecurity string `yaml:"strict_transport_security"`
		} `yaml:"headers"`

		TLS struct {
			// Minimum TLS version: 1.0, 1.1, 1.2, 1.3
			MinVersion string `yaml:"min_version"`
			// Allowed cipher suites
			CipherSuites []string `yaml:"cipher_suites"`
			// TLS certificate file path (optional, auto-detected)
			CertFile string `yaml:"cert_file"`
			// TLS key file path (optional, auto-detected)
			KeyFile string `yaml:"key_file"`
		} `yaml:"tls"`
		
		Upload struct {
			// Max upload size in bytes
			MaxFileSize int64 `yaml:"max_file_size"`
			// Allowed MIME types
			AllowedMIME []string `yaml:"allowed_mime_types"`
		} `yaml:"upload"`

		CORS struct {
			// Enable CORS
			Enabled bool `yaml:"enabled"`
			// Allowed origins (* for all)
			AllowedOrigins []string `yaml:"allowed_origins"`
			// Allowed HTTP methods
			AllowedMethods []string `yaml:"allowed_methods"`
			// Allowed headers
			AllowedHeaders []string `yaml:"allowed_headers"`
			// Preflight cache duration in seconds
			MaxAge int `yaml:"max_age"`
		} `yaml:"cors"`

		// CSRF protection per AI.md PART 11
		CSRF struct {
			// Enable CSRF protection (default: true)
			Enabled bool `yaml:"enabled"`
			// Token length in bytes (default: 32)
			TokenLength int `yaml:"token_length"`
			// Cookie name for CSRF token
			CookieName string `yaml:"cookie_name"`
			// Header name for CSRF token
			HeaderName string `yaml:"header_name"`
			// Form field name for CSRF token
			FieldName string `yaml:"field_name"`
			// Secure cookie: auto, true, false
			Secure string `yaml:"secure"`
		} `yaml:"csrf"`
	} `yaml:"security"`

	Web struct {
		UI struct {
			// Default paste lifetime
			DefaultLifetime string `yaml:"default_lifetime"`
			// Default theme (e.g. "dracula")
			DefaultTheme string `yaml:"default_theme"`
			// Themes directory (default: {data_dir}/web/themes)
			ThemesDir string `yaml:"themes_dir"`
		} `yaml:"ui"`

		Content struct {
			// Path to custom about page (empty=auto-generated, relative to {data_dir}/web/docs)
			About string `yaml:"about"`
			// Path to custom rules page (empty=auto-generated)
			Rules string `yaml:"rules"`
			// Path to custom terms page (empty=auto-generated)
			Terms string `yaml:"terms"`
			// Path to custom security.txt (empty=auto-generated)
			Security string `yaml:"security"`
		} `yaml:"content"`

		Branding struct {
			// Logo path or URL (e.g. "/static/logo.png" or "https://example.com/logo.png")
			Logo string `yaml:"logo"`
			// Favicon path or URL (e.g. "/static/favicon.ico" or "https://example.com/favicon.ico")
			Favicon string `yaml:"favicon"`
		} `yaml:"branding"`

		Security struct {
			Contact struct {
				// Security contact email
				Email string `yaml:"email"`
				// Security contact name
				Name string `yaml:"name"`
			} `yaml:"contact"`
		} `yaml:"security"`

		SEO struct {
			Robots struct {
				// Paths to allow in robots.txt
				Allow string `yaml:"allow"`
				// Paths to deny in robots.txt
				Deny string `yaml:"deny"`
				Agents struct {
					// User agents to deny
					Deny []string `yaml:"deny"`
				} `yaml:"agents"`
			} `yaml:"robots"`
		} `yaml:"seo"`
	} `yaml:"web"`

	Limits struct {
		// Max title length
		TitleMaxLength int `yaml:"title_max_length"`
		// Max paste body length
		BodyMaxLength int `yaml:"body_max_length"`
		// Max paste lifetime (e.g. "30d", "never")
		MaxPasteLifetime string `yaml:"max_paste_lifetime"`

		RateLimit struct {
			GetPastes struct {
				// GET requests per 5 minutes
				Per5Min uint `yaml:"per_5min"`
				// GET requests per 15 minutes
				Per15Min uint `yaml:"per_15min"`
				// GET requests per 1 hour
				Per1Hour uint `yaml:"per_1hour"`
			} `yaml:"get_pastes"`

			NewPastes struct {
				// POST requests per 5 minutes
				Per5Min uint `yaml:"per_5min"`
				// POST requests per 15 minutes
				Per15Min uint `yaml:"per_15min"`
				// POST requests per 1 hour
				Per1Hour uint `yaml:"per_1hour"`
			} `yaml:"new_pastes"`
		} `yaml:"rate_limit"`
	} `yaml:"limits"`

	Directories struct {
		// Data directory
		Data string `yaml:"data"`
		// Config directory
		Config string `yaml:"config"`
		// Database directory
		Db string `yaml:"db"`
		// Cache directory
		Cache string `yaml:"cache"`
		// Logs directory
		Logs string `yaml:"logs"`
	} `yaml:"directories"`
	
	Logging struct {
		// Log level: info, warn, error (default: info)
		Level string `yaml:"level"`

		Access struct {
			// Enable access log to stdout (default: true)
			Stdout bool `yaml:"stdout"`
			// Enable access log to stderr (default: false)
			Stderr bool `yaml:"stderr"`
			// apache, nginx, text, json (default: apache)
			Format string `yaml:"format"`
			// Access log file (default: access.log)
			File string `yaml:"file"`
		} `yaml:"access"`

		Error struct {
			// Enable error log to stdout (default: false)
			Stdout bool `yaml:"stdout"`
			// Enable error log to stderr (default: true)
			Stderr bool `yaml:"stderr"`
			// text, json (default: text)
			Format string `yaml:"format"`
			// Error log file (default: error.log)
			File string `yaml:"file"`
		} `yaml:"error"`

		Server struct {
			// Enable server log to stdout (default: true)
			Stdout bool `yaml:"stdout"`
			// Enable server log to stderr (default: false)
			Stderr bool `yaml:"stderr"`
			// text, json (default: text)
			Format string `yaml:"format"`
			// Server log file (default: caspaste.log)
			File string `yaml:"file"`
		} `yaml:"server"`

		Debug struct {
			// Enable debug log to stdout (default: true)
			Stdout bool `yaml:"stdout"`
			// Enable debug log to stderr (default: false)
			Stderr bool `yaml:"stderr"`
			// text, json (default: text)
			Format string `yaml:"format"`
			// Debug log file (default: debug.log)
			File string `yaml:"file"`
		} `yaml:"debug"`

		// Audit log per AI.md PART 11
		Audit struct {
			// Enable audit logging (default: true)
			Enabled bool `yaml:"enabled"`
			// Audit log file (default: audit.log)
			File string `yaml:"file"`
			// Mask email addresses in logs (default: true)
			MaskEmails bool `yaml:"mask_emails"`
			// Include User-Agent in logs (default: true)
			IncludeUserAgent bool `yaml:"include_user_agent"`
		} `yaml:"audit"`
	} `yaml:"logging"`
}

// LoadYAMLConfig loads configuration from YAML file
func LoadYAMLConfig(path string) (*YAMLConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg YAMLConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SaveYAMLConfig saves configuration to YAML file
func SaveYAMLConfig(path string, cfg *YAMLConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ResolvePlaceholders replaces placeholder values in the config with actual values
// Placeholders: {fqdn}, {data_dir}, {config_dir}
func ResolvePlaceholders(cfg *YAMLConfig, fqdn, dataDir, configDir string) {
	// Helper function to replace placeholders in a string
	replace := func(s string) string {
		s = strings.ReplaceAll(s, "{fqdn}", fqdn)
		s = strings.ReplaceAll(s, "{data_dir}", dataDir)
		s = strings.ReplaceAll(s, "{config_dir}", configDir)
		return s
	}

	// Server section
	cfg.Server.Administrator.Email = replace(cfg.Server.Administrator.Email)
	cfg.Server.Administrator.From = replace(cfg.Server.Administrator.From)

	// Web section
	cfg.Web.UI.ThemesDir = replace(cfg.Web.UI.ThemesDir)
	cfg.Web.Content.About = replace(cfg.Web.Content.About)
	cfg.Web.Content.Rules = replace(cfg.Web.Content.Rules)
	cfg.Web.Content.Terms = replace(cfg.Web.Content.Terms)
	cfg.Web.Content.Security = replace(cfg.Web.Content.Security)
	cfg.Web.Branding.Logo = replace(cfg.Web.Branding.Logo)
	cfg.Web.Branding.Favicon = replace(cfg.Web.Branding.Favicon)
	cfg.Web.Security.Contact.Email = replace(cfg.Web.Security.Contact.Email)

	// Security section
	cfg.Security.PasswordFile = replace(cfg.Security.PasswordFile)
	cfg.Security.TLS.CertFile = replace(cfg.Security.TLS.CertFile)
	cfg.Security.TLS.KeyFile = replace(cfg.Security.TLS.KeyFile)

	// Database section
	cfg.Database.Source = replace(cfg.Database.Source)

	// Set defaults for empty values that need data_dir
	if cfg.Web.UI.ThemesDir == "" {
		cfg.Web.UI.ThemesDir = dataDir + "/web/themes"
	}
}

// GetDefaultPrivateProxies returns the default trusted proxy CIDR ranges
// These are always trusted for X-Forwarded-* headers
func GetDefaultPrivateProxies() []string {
	return []string{
		// Private Class A
		"10.0.0.0/8",
		// Private Class B
		"172.16.0.0/12",
		// Private Class C
		"192.168.0.0/16",
		// Loopback IPv4
		"127.0.0.0/8",
		// Loopback IPv6
		"::1",
		// Unique Local IPv6
		"fc00::/7",
		// Link-Local IPv6
		"fe80::/10",
	}
}

// GetAllTrustedProxies returns all trusted proxies (defaults + configured)
func GetAllTrustedProxies(cfg *YAMLConfig) []string {
	proxies := GetDefaultPrivateProxies()
	proxies = append(proxies, cfg.Server.Proxy.Allowed...)
	return proxies
}

// GenerateDefaultYAMLConfig generates a default configuration file with sane defaults
func GenerateDefaultYAMLConfig(path string) error {
	defaultConfig := YAMLConfig{}

	// ============================================================================
	// SERVER CONFIGURATION
	// ============================================================================
	// Default: open/public instance (no auth required)
	defaultConfig.Server.Public = true
	// Empty = auto-detect from X-Forwarded-Host (trusted proxies) or hostname; Set to override
	defaultConfig.Server.FQDN = ""
	// Listen on all interfaces (IPv4 + IPv6)
	defaultConfig.Server.Listen = "all"
	// Empty = auto-detect available port at runtime
	defaultConfig.Server.Port = ""
	defaultConfig.Server.Title = "CasPaste"
	defaultConfig.Server.TagLine = "A simple paste service"
	defaultConfig.Server.Description = "CasPaste is a simple, fast, and secure paste service for sharing code snippets and text"

	// Additional trusted proxy IPs/CIDRs to append to default private ranges
	// Default private ranges (always trusted): 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8, ::1, fc00::/7, fe80::/10
	// Any IPs/CIDRs specified here are APPENDED to these defaults
	defaultConfig.Server.Proxy.Allowed = []string{}
	
	defaultConfig.Server.Administrator.Name = "CasPaste Administrator"
	defaultConfig.Server.Administrator.Email = "administrator@{fqdn}"
	defaultConfig.Server.Administrator.From = "\"CasPaste\" <no-reply@{fqdn}>"
	
	defaultConfig.Server.Timeouts.Read = 15
	defaultConfig.Server.Timeouts.Write = 15
	defaultConfig.Server.Timeouts.Idle = 60

	// Prometheus Metrics per AI.md PART 21 (INTERNAL ONLY - firewall /metrics)
	// Disabled by default, enable in production
	defaultConfig.Server.Metrics.Enabled = false
	defaultConfig.Server.Metrics.Endpoint = "/metrics"
	defaultConfig.Server.Metrics.IncludeSystem = true
	defaultConfig.Server.Metrics.IncludeRuntime = true
	// Empty = no auth (use firewall instead)
	defaultConfig.Server.Metrics.Token = ""
	defaultConfig.Server.Metrics.DurationBuckets = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	defaultConfig.Server.Metrics.SizeBuckets = []float64{100, 1000, 10000, 100000, 1000000, 10000000}

	// Tor Hidden Service per AI.md PART 32
	// Auto-enabled when Tor binary is found - no enable/disable toggle
	// Empty = auto-detect
	defaultConfig.Server.Tor.Binary = ""
	// Don't use Tor for outbound by default
	defaultConfig.Server.Tor.UseNetwork = false
	// Allow users to override
	defaultConfig.Server.Tor.AllowUserPreference = true
	// Keep 32 circuits open
	defaultConfig.Server.Tor.MaxCircuits = 32
	// 60 seconds
	defaultConfig.Server.Tor.CircuitTimeout = 60
	// 3 minutes
	defaultConfig.Server.Tor.BootstrapTimeout = 180
	// Scrub sensitive info
	defaultConfig.Server.Tor.SafeLogging = true
	// Max streams per circuit
	defaultConfig.Server.Tor.MaxStreamsPerCircuit = 100
	// Close circuit at limit
	defaultConfig.Server.Tor.CloseCircuitOnStreamLimit = true
	// 1 MB/s
	defaultConfig.Server.Tor.BandwidthRate = "1 MB"
	// 2 MB/s burst
	defaultConfig.Server.Tor.BandwidthBurst = "2 MB"
	// 100 GB per month
	defaultConfig.Server.Tor.MaxMonthlyBandwidth = "100 GB"
	// 3 introduction points
	defaultConfig.Server.Tor.NumIntroPoints = 3
	// Listen on port 80
	defaultConfig.Server.Tor.VirtualPort = 80

	// ============================================================================
	// DATABASE CONFIGURATION
	// ============================================================================
	// Using modernc.org/sqlite (pure Go, no CGo)
	// Source path is relative - converted to absolute at runtime
	defaultConfig.Database.Driver = "sqlite"
	defaultConfig.Database.Source = "caspaste.db"
	defaultConfig.Database.MaxOpenConns = 25
	defaultConfig.Database.MaxIdleConns = 5
	defaultConfig.Database.CleanupPeriod = "1m"

	// ============================================================================
	// SECURITY CONFIGURATION
	// ============================================================================
	// Empty = auto-generate when server.public=false
	defaultConfig.Security.PasswordFile = ""
	
	// HTTP Security Headers per AI.md PART 11
	defaultConfig.Security.Headers.XFrameOptions = "SAMEORIGIN"
	defaultConfig.Security.Headers.XContentTypeOptions = "nosniff"
	defaultConfig.Security.Headers.XSSProtection = "1; mode=block"
	defaultConfig.Security.Headers.ContentSecurityPolicy = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; object-src 'none'; base-uri 'self'; form-action 'self'"
	defaultConfig.Security.Headers.ReferrerPolicy = "strict-origin-when-cross-origin"
	defaultConfig.Security.Headers.PermissionsPolicy = "geolocation=(), microphone=(), camera=()"
	defaultConfig.Security.Headers.StrictTransportSecurity = "max-age=31536000; includeSubDomains"
	
	// TLS Configuration
	defaultConfig.Security.TLS.MinVersion = "1.2"
	defaultConfig.Security.TLS.CipherSuites = []string{
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		"TLS_CHACHA20_POLY1305_SHA256",
	}
	// Auto-detected from Let's Encrypt
	defaultConfig.Security.TLS.CertFile = "/etc/casjay-forks/caspaste/tls/cert.pem"
	// Auto-detected from Let's Encrypt
	defaultConfig.Security.TLS.KeyFile = "/etc/casjay-forks/caspaste/tls/key.pem"
	
	// Upload Security
	// 50MB
	defaultConfig.Security.Upload.MaxFileSize = 52428800
	defaultConfig.Security.Upload.AllowedMIME = []string{
		"text/plain",
		"text/markdown",
		"text/html",
		"text/css",
		"text/javascript",
		"application/json",
		"application/xml",
		"application/pdf",
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/svg+xml",
		"image/webp",
	}
	
	// CORS Configuration
	defaultConfig.Security.CORS.Enabled = true
	defaultConfig.Security.CORS.AllowedOrigins = []string{"*"}
	defaultConfig.Security.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	defaultConfig.Security.CORS.AllowedHeaders = []string{"Content-Type", "Authorization", "X-Requested-With"}
	// 24 hours
	defaultConfig.Security.CORS.MaxAge = 86400

	// CSRF Protection per AI.md PART 11
	defaultConfig.Security.CSRF.Enabled = true
	defaultConfig.Security.CSRF.TokenLength = 32
	defaultConfig.Security.CSRF.CookieName = "csrf_token"
	defaultConfig.Security.CSRF.HeaderName = "X-CSRF-Token"
	defaultConfig.Security.CSRF.FieldName = "csrf_token"
	defaultConfig.Security.CSRF.Secure = "auto"

	// ============================================================================
	// WEB CONFIGURATION
	// ============================================================================

	// UI Settings
	defaultConfig.Web.UI.DefaultLifetime = "never"
	// Accepts: "dark" (dracula), "light" (github), "auto", or full path like "dark/dracula"
	defaultConfig.Web.UI.DefaultTheme = "dark"
	// Empty = {data_dir}/web/themes (resolved at runtime)
	defaultConfig.Web.UI.ThemesDir = ""

	// Content Pages - all empty = auto-generated from embedded defaults
	// If set, paths are relative to {data_dir}/web/docs unless absolute
	// Empty = auto-generated
	defaultConfig.Web.Content.About = ""
	// Empty = auto-generated
	defaultConfig.Web.Content.Rules = ""
	// Empty = auto-generated
	defaultConfig.Web.Content.Terms = ""
	// Empty = auto-generated security.txt
	defaultConfig.Web.Content.Security = ""

	// Branding - can be local paths or URLs
	// Empty = use embedded default
	defaultConfig.Web.Branding.Logo = ""
	// Empty = use embedded default
	defaultConfig.Web.Branding.Favicon = ""
	
	// Security Contact (for security.txt)
	defaultConfig.Web.Security.Contact.Email = "security@{fqdn}"
	defaultConfig.Web.Security.Contact.Name = "Security Team"
	
	// SEO / Robots
	defaultConfig.Web.SEO.Robots.Allow = "*"
	defaultConfig.Web.SEO.Robots.Deny = "/settings,/history"
	defaultConfig.Web.SEO.Robots.Agents.Deny = []string{
		"GPTBot",
		"ChatGPT-User",
		"Google-Extended",
		"CCBot",
		"anthropic-ai",
		"Claude-Web",
		"cohere-ai",
		"Omgilibot",
		"FacebookBot",
		"Diffbot",
	}

	// ============================================================================
	// LIMITS & RATE LIMITING
	// ============================================================================
	defaultConfig.Limits.TitleMaxLength = 100
	// 50MB
	defaultConfig.Limits.BodyMaxLength = 52428800
	defaultConfig.Limits.MaxPasteLifetime = "never"
	
	// Rate limiting for GET requests
	defaultConfig.Limits.RateLimit.GetPastes.Per5Min = 50
	defaultConfig.Limits.RateLimit.GetPastes.Per15Min = 100
	defaultConfig.Limits.RateLimit.GetPastes.Per1Hour = 500
	
	// Rate limiting for POST requests
	defaultConfig.Limits.RateLimit.NewPastes.Per5Min = 15
	defaultConfig.Limits.RateLimit.NewPastes.Per15Min = 30
	defaultConfig.Limits.RateLimit.NewPastes.Per1Hour = 40

	// ============================================================================
	// DIRECTORIES
	// ============================================================================
	// Platform-specific defaults
	defaultConfig.Directories.Data = "/var/lib/casjay-forks/caspaste"
	defaultConfig.Directories.Config = "/etc/casjay-forks/caspaste"
	// Database directory - if under data dir, included in data backup
	defaultConfig.Directories.Db = "/var/lib/casjay-forks/caspaste/db"
	defaultConfig.Directories.Cache = "/var/cache/caspaste"
	defaultConfig.Directories.Logs = "/var/log/casjay-forks/caspaste"

	// ============================================================================
	// LOGGING
	// ============================================================================
	// info, warn, error (default: info)
	defaultConfig.Logging.Level = "info"
	
	// Access Log (HTTP requests)
	// Don't clutter console with every request
	defaultConfig.Logging.Access.Stdout = false
	defaultConfig.Logging.Access.Stderr = false
	// apache (combined), nginx, text, json
	defaultConfig.Logging.Access.Format = "apache"
	defaultConfig.Logging.Access.File = "access.log"
	
	// Error Log (ERROR messages)
	defaultConfig.Logging.Error.Stdout = false
	// Errors to stderr by default
	defaultConfig.Logging.Error.Stderr = true
	// text, json
	defaultConfig.Logging.Error.Format = "text"
	defaultConfig.Logging.Error.File = "error.log"
	
	// Server Log (INFO messages)
	// Show info messages on console
	defaultConfig.Logging.Server.Stdout = true
	defaultConfig.Logging.Server.Stderr = false
	// text, json
	defaultConfig.Logging.Server.Format = "text"
	defaultConfig.Logging.Server.File = "caspaste.log"
	
	// Debug Log (DEBUG messages, only with --debug flag)
	defaultConfig.Logging.Debug.Stdout = true
	defaultConfig.Logging.Debug.Stderr = false
	// text, json
	defaultConfig.Logging.Debug.Format = "text"
	defaultConfig.Logging.Debug.File = "debug.log"

	// Audit Log per AI.md PART 11 (security events in JSON Lines format)
	defaultConfig.Logging.Audit.Enabled = true
	defaultConfig.Logging.Audit.File = "audit.log"
	defaultConfig.Logging.Audit.MaskEmails = true
	defaultConfig.Logging.Audit.IncludeUserAgent = true

	// Write to file
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	return nil
}

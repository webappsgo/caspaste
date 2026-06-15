// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package config

import (
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/casjay-forks/caspaste/src/validation"
	"golang.org/x/net/publicsuffix"
)

// isValidDomain checks if a string is a valid domain name using the Public Suffix List
// This validates against all known TLDs (com, org, co.uk, com.au, etc.)
func isValidDomain(s string) bool {
	// Must not be empty
	if s == "" {
		return false
	}
	// Must not be an IP address
	if net.ParseIP(s) != nil {
		return false
	}
	// Must contain at least one dot
	if !strings.Contains(s, ".") {
		return false
	}
	// Use publicsuffix to get the eTLD+1 (effective TLD plus one label)
	// If this succeeds, it's a valid domain with a known TLD
	_, err := publicsuffix.EffectiveTLDPlusOne(s)
	return err == nil
}

// parseAddress intelligently parses CASPB_ADDRESS to extract FQDN, listen, and port
// Examples:
//   - ":8080"                    → port=8080
//   - "pastebin.example.com:80"  → fqdn=pastebin.example.com, port=80
//   - "127.0.0.1"                → listen=127.0.0.1
//   - "172.17.0.1:8091"          → listen=172.17.0.1, port=8091
//   - "example.com"              → fqdn=example.com
func parseAddress(addr string) (fqdn, listen, port string) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return
	}

	// Handle IPv6 addresses in brackets like [::1]:8080
	if strings.HasPrefix(addr, "[") {
		// IPv6 format: [ip]:port or [ip]
		closeBracket := strings.Index(addr, "]")
		if closeBracket == -1 {
			// Invalid format
			return
		}
		ipv6 := addr[1:closeBracket]
		rest := addr[closeBracket+1:]

		// Check if there's a port after the bracket
		if strings.HasPrefix(rest, ":") {
			port = rest[1:]
		}
		listen = ipv6
		return
	}

	// Check if it starts with ":" (just port)
	if strings.HasPrefix(addr, ":") {
		port = addr[1:]
		return
	}

	// Try to split host:port
	// Use net.SplitHostPort for proper parsing
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		// No port specified, the whole string is the host
		host = addr
		p = ""
	}

	// Set port if found
	port = p

	// Determine if host is an IP or domain
	if host != "" {
		if net.ParseIP(host) != nil {
			// It's an IP address → listen address
			listen = host
		} else if isValidDomain(host) {
			// It's a valid domain → FQDN
			fqdn = host
		} else if host == "localhost" {
			// Special case: localhost is a listen address
			listen = host
		}
		// If neither IP nor valid domain, ignore it
	}

	return
}

// getEnv gets CASPB_* environment variables
func getEnv(name string) string {
	return os.Getenv("CASPB_" + name)
}

// ApplyEnvironmentOverrides applies environment variables to config
// Environment variables override config file values
func ApplyEnvironmentOverrides(cfg *YAMLConfig) {
	// Smart ADDRESS parsing - single env var to set fqdn, listen, and/or port
	// Examples:
	//   CASPB_ADDRESS=:8080                    → port=8080
	//   CASPB_ADDRESS=pastebin.example.com:80 → fqdn=pastebin.example.com, port=80
	//   CASPB_ADDRESS=127.0.0.1               → listen=127.0.0.1
	//   CASPB_ADDRESS=172.17.0.1:8091         → listen=172.17.0.1, port=8091
	if val := getEnv("ADDRESS"); val != "" {
		fqdn, listen, port := parseAddress(val)
		if fqdn != "" {
			cfg.Server.FQDN = fqdn
		}
		if listen != "" {
			cfg.Server.Listen = listen
		}
		if port != "" {
			cfg.Server.Port = port
		}
	}

	// Individual settings (override ADDRESS parsing if both set)
	if val := getEnv("FQDN"); val != "" {
		cfg.Server.FQDN = val
	}
	if val := getEnv("LISTEN"); val != "" {
		cfg.Server.Listen = val
	}
	// Backward compatibility
	if val := getEnv("BIND"); val != "" {
		cfg.Server.Listen = val
	}
	// Now string format: "8080" or "8080,64453"
	if val := getEnv("PORT"); val != "" {
		cfg.Server.Port = val
	}
	if val := getEnv("SERVER_TITLE"); val != "" {
		cfg.Server.Title = val
	}
	// Alternative
	if val := getEnv("TITLE"); val != "" {
		cfg.Server.Title = val
	}

	// Server administrator
	if val := getEnv("ADMIN_NAME"); val != "" {
		cfg.Server.Administrator.Name = val
	}
	if val := getEnv("SERVER_ADMINISTRATOR_NAME"); val != "" {
		cfg.Server.Administrator.Name = val
	}
	if val := getEnv("ADMIN_EMAIL"); val != "" {
		cfg.Server.Administrator.Email = val
	}
	// Alternative
	if val := getEnv("ADMIN_MAIL"); val != "" {
		cfg.Server.Administrator.Email = val
	}
	if val := getEnv("SERVER_ADMINISTRATOR_EMAIL"); val != "" {
		cfg.Server.Administrator.Email = val
	}
	if val := getEnv("SERVER_ADMINISTRATOR_FROM"); val != "" {
		cfg.Server.Administrator.From = val
	}

	// Web security contact
	if val := getEnv("WEB_SECURITY_CONTACT_EMAIL"); val != "" {
		cfg.Web.Security.Contact.Email = val
	}
	if val := getEnv("WEB_SECURITY_CONTACT_NAME"); val != "" {
		cfg.Web.Security.Contact.Name = val
	}

	// Site robots -> Web.SEO.Robots
	if val := getEnv("SITE_ROBOTS_ALLOW"); val != "" {
		cfg.Web.SEO.Robots.Allow = val
	}
	if val := getEnv("SITE_ROBOTS_DENY"); val != "" {
		cfg.Web.SEO.Robots.Deny = val
	}
	// Legacy compatibility
	if val := getEnv("ROBOTS_DISALLOW"); val != "" {
		if validation.IsTruthy(val) {
			cfg.Web.SEO.Robots.Deny = "/"
		}
	}

	// Branding -> Web.Branding
	if val := getEnv("BRANDING_LOGO"); val != "" {
		cfg.Web.Branding.Logo = val
	}
	if val := getEnv("BRANDING_FAVICON"); val != "" {
		cfg.Web.Branding.Favicon = val
	}

	// Database settings
	if val := getEnv("DB_DRIVER"); val != "" {
		cfg.Database.Driver = val
	}
	if val := getEnv("DB_SOURCE"); val != "" {
		cfg.Database.Source = val
	}
	if val := getEnv("DB_MAX_OPEN_CONNS"); val != "" {
		if num, err := strconv.Atoi(val); err == nil {
			cfg.Database.MaxOpenConns = num
		}
	}
	if val := getEnv("DB_MAX_IDLE_CONNS"); val != "" {
		if num, err := strconv.Atoi(val); err == nil {
			cfg.Database.MaxIdleConns = num
		}
	}
	if val := getEnv("DB_CLEANUP_PERIOD"); val != "" {
		cfg.Database.CleanupPeriod = val
	}

	// Security settings
	if val := getEnv("PASSWORD_FILE"); val != "" {
		cfg.Security.PasswordFile = val
	}
	// Alternative name
	if val := getEnv("CASPASSWD_FILE"); val != "" {
		cfg.Security.PasswordFile = val
	}

	// Limits settings
	if val := getEnv("TITLE_MAX_LENGTH"); val != "" {
		if num, err := strconv.Atoi(val); err == nil {
			cfg.Limits.TitleMaxLength = num
		}
	}
	if val := getEnv("BODY_MAX_LENGTH"); val != "" {
		if num, err := strconv.Atoi(val); err == nil {
			cfg.Limits.BodyMaxLength = num
		}
	}
	if val := getEnv("MAX_PASTE_LIFETIME"); val != "" {
		cfg.Limits.MaxPasteLifetime = val
	}
	// Rate limits - GET pastes
	if val := getEnv("GET_PASTES_PER_5MIN"); val != "" {
		if num, err := strconv.ParseUint(val, 10, 32); err == nil {
			cfg.Limits.RateLimit.GetPastes.Per5Min = uint(num)
		}
	}
	if val := getEnv("GET_PASTES_PER_15MIN"); val != "" {
		if num, err := strconv.ParseUint(val, 10, 32); err == nil {
			cfg.Limits.RateLimit.GetPastes.Per15Min = uint(num)
		}
	}
	if val := getEnv("GET_PASTES_PER_1HOUR"); val != "" {
		if num, err := strconv.ParseUint(val, 10, 32); err == nil {
			cfg.Limits.RateLimit.GetPastes.Per1Hour = uint(num)
		}
	}
	
	// Rate limits - NEW pastes
	if val := getEnv("NEW_PASTES_PER_5MIN"); val != "" {
		if num, err := strconv.ParseUint(val, 10, 32); err == nil {
			cfg.Limits.RateLimit.NewPastes.Per5Min = uint(num)
		}
	}
	if val := getEnv("NEW_PASTES_PER_15MIN"); val != "" {
		if num, err := strconv.ParseUint(val, 10, 32); err == nil {
			cfg.Limits.RateLimit.NewPastes.Per15Min = uint(num)
		}
	}
	if val := getEnv("NEW_PASTES_PER_1HOUR"); val != "" {
		if num, err := strconv.ParseUint(val, 10, 32); err == nil {
			cfg.Limits.RateLimit.NewPastes.Per1Hour = uint(num)
		}
	}

	// UI settings -> Web.UI
	if val := getEnv("UI_DEFAULT_LIFETIME"); val != "" {
		cfg.Web.UI.DefaultLifetime = val
	}
	if val := getEnv("UI_DEFAULT_THEME"); val != "" {
		cfg.Web.UI.DefaultTheme = val
	}
	if val := getEnv("UI_THEMES_DIR"); val != "" {
		cfg.Web.UI.ThemesDir = val
	}

	// Content settings -> Web.Content
	if val := getEnv("CONTENT_ABOUT"); val != "" {
		cfg.Web.Content.About = val
	}
	// Legacy
	if val := getEnv("SERVER_ABOUT"); val != "" {
		cfg.Web.Content.About = val
	}
	if val := getEnv("CONTENT_RULES"); val != "" {
		cfg.Web.Content.Rules = val
	}
	// Legacy
	if val := getEnv("SERVER_RULES"); val != "" {
		cfg.Web.Content.Rules = val
	}
	if val := getEnv("CONTENT_TERMS"); val != "" {
		cfg.Web.Content.Terms = val
	}
	// Legacy
	if val := getEnv("SERVER_TERMS"); val != "" {
		cfg.Web.Content.Terms = val
	}
	if val := getEnv("CONTENT_SECURITY"); val != "" {
		cfg.Web.Content.Security = val
	}

	// Directory settings
	if val := getEnv("CACHE_DIR"); val != "" {
		cfg.Directories.Cache = val
	}
	if val := getEnv("LOGS_DIR"); val != "" {
		cfg.Directories.Logs = val
	}

	// API compatibility mode — also readable directly by the compat package via os.Getenv.
	if val := getEnv("API_MODE"); val != "" {
		cfg.Server.APIMode = val
	}
}

// ApplyCriticalOverrides applies security-critical environment variables on EVERY run
// Unlike ApplyEnvironmentOverrides which only runs on first config generation,
// this function runs on every startup to ensure security settings can be changed
// via environment variables in containerized deployments without deleting config
func ApplyCriticalOverrides(cfg *YAMLConfig) {
	// Server public mode - critical for enabling/disabling authentication
	// PUBLIC=true (default) = open/public, PUBLIC=false = auth required
	if val := getEnv("PUBLIC"); val != "" {
		cfg.Server.Public = validation.IsTruthy(val)
	}

	// Password file - for custom users (optional, auto-generated if not set)
	if val := getEnv("PASSWORD_FILE"); val != "" {
		cfg.Security.PasswordFile = val
	}
	if val := getEnv("CASPASSWD_FILE"); val != "" {
		cfg.Security.PasswordFile = val
	}

	// TLS settings - critical for HTTPS security
	if val := getEnv("TLS_MIN_VERSION"); val != "" {
		cfg.Security.TLS.MinVersion = val
	}
}

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package config

import (
	"strings"

	"github.com/webappsgo/caspaste/src/logger"
	"github.com/webappsgo/caspaste/src/netshare"
)

const Software = "CasPaste"

// DefaultLanguage is the fallback UI/CLI language per AI.md PART 31
const DefaultLanguage = "en"

// SupportedLanguages lists every language shipped by all binaries per AI.md PART 31
var SupportedLanguages = []string{"en", "es", "zh", "fr", "ar", "de", "ja"}

// NormalizeLang returns a supported language code for the given input,
// silently falling back to "en" for empty or unsupported values per AI.md
// PART 31 (unsupported --lang/Accept-Language must never error).
func NormalizeLang(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return DefaultLanguage
	}
	// Match the base subtag (e.g. "en-US" -> "en")
	if i := strings.IndexAny(code, "-_"); i > 0 {
		code = code[:i]
	}
	for _, lang := range SupportedLanguages {
		if lang == code {
			return code
		}
	}
	return DefaultLanguage
}

// Default API and admin path values
const (
	DefaultAPIVersion = "v1"
	DefaultAdminPath  = "admin"
)

// DefaultBaseURL is the default base URL path prefix per AI.md PART 12
const DefaultBaseURL = "/"

// Package-level variables for global access
var (
	currentAPIVersion  = DefaultAPIVersion
	currentAdminPath   = DefaultAdminPath
	currentBaseURL     = DefaultBaseURL
	currentDefaultLang = DefaultLanguage
)

// DefaultLang returns the configured default UI/CLI language (default "en")
func DefaultLang() string {
	return currentDefaultLang
}

// SetDefaultLang sets the process default language (already normalized)
func SetDefaultLang(code string) {
	if code != "" {
		currentDefaultLang = code
	}
}

// NormalizeBaseURL canonicalizes a base URL path prefix: ensures a leading
// slash and strips a trailing slash (except the root "/"). Empty -> "/".
func NormalizeBaseURL(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" {
		return DefaultBaseURL
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = strings.TrimRight(p, "/")
	if p == "" {
		return DefaultBaseURL
	}
	return p
}

// BaseURL returns the configured base URL path prefix (default "/")
func BaseURL() string {
	return currentBaseURL
}

// SetBaseURL sets the base URL path prefix (called during config load)
func SetBaseURL(p string) {
	currentBaseURL = NormalizeBaseURL(p)
}

// APIVersion returns the current API version (default: "v1")
func APIVersion() string {
	return currentAPIVersion
}

// SetAPIVersion sets the API version (called during config load)
func SetAPIVersion(v string) {
	if v != "" {
		currentAPIVersion = v
	}
}

// APIBasePath returns the API base path (e.g., "/api/v1")
func APIBasePath() string {
	return "/api/" + currentAPIVersion
}

// AdminPath returns the admin panel path (default: "admin")
func AdminPath() string {
	return currentAdminPath
}

// SetAdminPath sets the admin path (called during config load)
func SetAdminPath(p string) {
	if p != "" {
		currentAdminPath = p
	}
}

// AdminBasePath returns the admin UI base path (e.g., "/server/admin")
func AdminBasePath() string {
	return "/server/" + currentAdminPath
}

// AdminAPIPath returns the admin API path (e.g., "/api/v1/server/admin")
func AdminAPIPath() string {
	return "/api/" + currentAPIVersion + "/server/" + currentAdminPath
}

type Config struct {
	Log logger.Logger

	RateLimitNew *netshare.RateLimitSystem
	RateLimitGet *netshare.RateLimitSystem

	// API and admin paths
	APIVersion string
	AdminPath  string

	Version     string
	BuildCommit string
	BuildDate   string
	// "production" or "development"
	Mode string

	// Path to the application data directory; used for disk health check
	DataDir string

	// Branding/description (from yaml server.tagline, server.description)
	ServerTagline     string
	ServerDescription string

	TitleMaxLen int
	BodyMaxLen  int
	MaxLifeTime int64

	// Content
	ServerAbout      string
	ServerRules      string
	ServerTermsOfUse string
	SecurityTxt      string

	// Server info
	FQDN        string
	ServerTitle string
	AdminName   string
	AdminMail   string

	// Security contact
	SecurityContactEmail string
	SecurityContactName  string

	// Robots
	SiteRobotsAllow      string
	SiteRobotsDeny       string
	SiteRobotsAgentsDeny []string

	// Branding
	Logo    string
	Favicon string

	// Authentication
	// true = open/public (no auth), false = auth required
	Public        bool
	CasPasswdFile string

	// Trusted proxy configuration (for X-Forwarded-* headers)
	TrustedProxies []string

	UiDefaultLifetime string
	UiDefaultTheme    string
	UiThemesDir       string

	// HealthzRootEnabled mounts the optional /healthz root alias per AI.md PART 13
	HealthzRootEnabled bool
}


// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package config

import (
	"github.com/casjay-forks/caspaste/src/logger"
	"github.com/casjay-forks/caspaste/src/netshare"
)

const Software = "CasPb"

// Default API and admin path values
const (
	DefaultAPIVersion = "v1"
	DefaultAdminPath  = "admin"
)

// Package-level variables for global access
var (
	currentAPIVersion = DefaultAPIVersion
	currentAdminPath  = DefaultAdminPath
)

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
}


// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package config

import (
	"github.com/casjay-forks/caspaste/src/logger"
	"github.com/casjay-forks/caspaste/src/netshare"
)

const Software = "CasPaste"

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

// AdminBasePath returns the admin base path with leading slash (e.g., "/admin")
func AdminBasePath() string {
	return "/" + currentAdminPath
}

// AdminAPIPath returns the admin API path (e.g., "/api/v1/admin")
func AdminAPIPath() string {
	return APIBasePath() + "/" + currentAdminPath
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

	// Multi-User Configuration (PART 34)
	Users UsersConfig

	// Features Configuration (PART 35, 36)
	Features FeaturesConfig
}

// UsersConfig contains multi-user settings per PART 34
type UsersConfig struct {
	// Enable multi-user mode (default: disabled = admin-only)
	Enabled bool

	Registration RegistrationConfig
	Roles        RolesConfig
	Tokens       TokensConfig
	Profile      ProfileConfig
	Auth         UserAuthConfig
	Limits       UserLimitsConfig
}

// RegistrationConfig contains registration settings
type RegistrationConfig struct {
	// Registration mode: public (default), private, disabled
	Mode string
	// Email verification (applies to public mode)
	RequireEmailVerification bool
	// Email domain restrictions (applies to public mode)
	AllowedDomains []string
	BlockedDomains []string
	// Invite settings (admin-generated invites)
	InviteExpirationDays int
}

// RolesConfig contains role settings
type RolesConfig struct {
	// Available roles
	Available []string
	// Default role for new users
	Default string
}

// TokensConfig contains API token settings
type TokensConfig struct {
	// Allow users to generate API tokens
	Enabled bool
	// Maximum tokens per user
	MaxPerUser int
	// Token expiration (0 = never)
	ExpirationDays int
}

// ProfileConfig contains profile settings
type ProfileConfig struct {
	// Allow users to upload avatars
	AllowAvatar bool
	// Allow users to set display name
	AllowDisplayName bool
	// Allow users to set bio
	AllowBio bool
}

// UserAuthConfig contains user authentication settings
type UserAuthConfig struct {
	// Session duration
	SessionDuration string
	// Require 2FA for all users
	Require2FA bool
	// Allow 2FA (user choice)
	Allow2FA bool
	// Password requirements
	PasswordMinLength        int
	PasswordRequireUppercase bool
	PasswordRequireNumber    bool
	PasswordRequireSpecial   bool
}

// UserLimitsConfig contains per-user limits
type UserLimitsConfig struct {
	// Rate limits per user (0 = use global)
	RequestsPerMinute int
	RequestsPerDay    int
}

// FeaturesConfig contains optional feature settings per PART 35, 36
type FeaturesConfig struct {
	Organizations OrganizationsConfig
	CustomDomains CustomDomainsConfig
}

// OrganizationsConfig contains organization settings per PART 35
type OrganizationsConfig struct {
	// Enable organization support
	Enabled bool
}

// CustomDomainsConfig contains custom domain settings per PART 36
type CustomDomainsConfig struct {
	// Enable custom domain support
	Enabled bool
	// Limit per user (0 = unlimited)
	MaxDomainsPerUser int
	// Limit per org (0 = unlimited)
	MaxDomainsPerOrg int
	// Require SSL for all custom domains
	RequireSSL bool
	// Allow apex domains (example.com)
	AllowApex bool
	// Allow subdomains (sub.example.com)
	AllowSubdomain bool
	// Allow wildcard domains (*.example.com)
	AllowWildcard bool
	// Verification token TTL
	VerificationTTL string
	// Renew SSL certs N days before expiry
	SSLRenewalDays int
	// Reserved domains that cannot be used
	Reserved []string
}

// DefaultUsersConfig returns default user configuration
func DefaultUsersConfig() UsersConfig {
	return UsersConfig{
		Enabled: false,
		Registration: RegistrationConfig{
			Mode:                     "public",
			RequireEmailVerification: true,
			AllowedDomains:           []string{},
			BlockedDomains:           []string{},
			InviteExpirationDays:     7,
		},
		Roles: RolesConfig{
			Available: []string{"admin", "user"},
			Default:   "user",
		},
		Tokens: TokensConfig{
			Enabled:        true,
			MaxPerUser:     5,
			ExpirationDays: 0,
		},
		Profile: ProfileConfig{
			AllowAvatar:      true,
			AllowDisplayName: true,
			AllowBio:         true,
		},
		Auth: UserAuthConfig{
			SessionDuration:          "30d",
			Require2FA:               false,
			Allow2FA:                 true,
			PasswordMinLength:        8,
			PasswordRequireUppercase: false,
			PasswordRequireNumber:    false,
			PasswordRequireSpecial:   false,
		},
		Limits: UserLimitsConfig{
			RequestsPerMinute: 0,
			RequestsPerDay:    0,
		},
	}
}

// DefaultFeaturesConfig returns default features configuration
func DefaultFeaturesConfig() FeaturesConfig {
	return FeaturesConfig{
		Organizations: OrganizationsConfig{
			Enabled: false,
		},
		CustomDomains: CustomDomainsConfig{
			Enabled:           false,
			MaxDomainsPerUser: 5,
			MaxDomainsPerOrg:  20,
			RequireSSL:        true,
			AllowApex:         true,
			AllowSubdomain:    true,
			AllowWildcard:     false,
			VerificationTTL:   "24h",
			SSLRenewalDays:    7,
			Reserved: []string{
				"localhost",
				"*.local",
				"*.test",
				"*.example",
				"*.invalid",
			},
		},
	}
}


// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package admin

import (
	"net/http"
)

// AuthHandler returns the HTTP handler for shared admin authentication routes.
// Mounted with http.StripPrefix("/server/auth", ...) per AI.md PART 15 (Auth Routes).
// The same login form handles both admin and (when multi-user is enabled) regular user login.
func (p *Panel) AuthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", p.handleLoginPage)
	mux.HandleFunc("POST /login", p.handleLoginPost)
	mux.HandleFunc("GET /logout", p.handleLogout)
	mux.HandleFunc("POST /logout", p.handleLogout)
	mux.HandleFunc("GET /invite/server/{token}", p.handleAdminInviteAccept)
	mux.HandleFunc("POST /invite/server/{token}", p.handleAdminInviteAccept)
	return mux
}

// Handler returns the HTTP handler for the admin panel UI.
// It is mounted with http.StripPrefix("/server/{admin_path}", ...) so all
// patterns below are relative to that stripped prefix.
// Login/logout are served by AuthHandler at /server/auth/ instead.
func (p *Panel) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public routes — no auth required
	mux.HandleFunc("GET /config/setup", p.handleSetup)
	mux.HandleFunc("POST /config/setup", p.handleSetup)

	// Auth-protected routes
	auth := p.requireAuth

	mux.Handle("GET /", auth(http.HandlerFunc(p.handleDashboard)))

	// Admin self-management: /{username}/profile|preferences|notifications
	mux.Handle("GET /{username}/profile", auth(http.HandlerFunc(p.handleProfile)))
	mux.Handle("POST /{username}/profile", auth(http.HandlerFunc(p.handleProfilePost)))
	mux.Handle("GET /{username}/preferences", auth(http.HandlerFunc(p.handlePreferences)))
	mux.Handle("POST /{username}/preferences", auth(http.HandlerFunc(p.handlePreferencesPost)))
	mux.Handle("GET /{username}/notifications", auth(http.HandlerFunc(p.handleNotifications)))

	// Server management — all under /config/
	mux.Handle("GET /config/", auth(http.HandlerFunc(p.handleConfigRoot)))
	mux.Handle("GET /config/settings", auth(http.HandlerFunc(p.handleSettings)))
	mux.Handle("POST /config/settings", auth(http.HandlerFunc(p.handleSettingsPost)))
	mux.Handle("GET /config/ssl", auth(http.HandlerFunc(p.handleSSL)))
	mux.Handle("GET /config/email", auth(http.HandlerFunc(p.handleEmail)))
	mux.Handle("POST /config/email", auth(http.HandlerFunc(p.handleEmailPost)))
	mux.Handle("GET /config/scheduler", auth(http.HandlerFunc(p.handleScheduler)))
	mux.Handle("POST /config/scheduler", auth(http.HandlerFunc(p.handleSchedulerPost)))
	mux.Handle("GET /config/logs", auth(http.HandlerFunc(p.handleLogs)))
	mux.Handle("GET /config/logs/audit", auth(http.HandlerFunc(p.handleLogsAudit)))
	mux.Handle("GET /config/backup", auth(http.HandlerFunc(p.handleBackup)))
	mux.Handle("POST /config/backup", auth(http.HandlerFunc(p.handleBackupPost)))
	mux.Handle("GET /config/updates", auth(http.HandlerFunc(p.handleUpdates)))
	mux.Handle("GET /config/info", auth(http.HandlerFunc(p.handleInfo)))
	mux.Handle("GET /config/metrics", auth(http.HandlerFunc(p.handleMetrics)))
	mux.Handle("GET /config/admins", auth(http.HandlerFunc(p.handleAdmins)))
	mux.Handle("POST /config/admins", auth(http.HandlerFunc(p.handleAdminsPost)))

	// Network
	mux.Handle("GET /config/network/", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, p.adminBasePath()+"/config/network/geoip", http.StatusSeeOther)
	})))
	mux.Handle("GET /config/network/tor", auth(http.HandlerFunc(p.handleNetworkTor)))
	mux.Handle("GET /config/network/geoip", auth(http.HandlerFunc(p.handleNetworkGeoIP)))

	// Security
	mux.Handle("GET /config/security/", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, p.adminBasePath()+"/config/security/auth", http.StatusSeeOther)
	})))
	mux.Handle("GET /config/security/auth", auth(http.HandlerFunc(p.handleSecurityAuth)))
	mux.Handle("GET /config/security/auth/oidc", auth(http.HandlerFunc(p.handleSecurityOIDC)))
	mux.Handle("GET /config/security/auth/ldap", auth(http.HandlerFunc(p.handleSecurityLDAP)))
	mux.Handle("GET /config/security/tokens", auth(http.HandlerFunc(p.handleSecurityTokens)))
	mux.Handle("POST /config/security/tokens", auth(http.HandlerFunc(p.handleSecurityTokensPost)))
	mux.Handle("GET /config/security/firewall", auth(http.HandlerFunc(p.handleSecurityFirewall)))

	return mux
}

// APIHandler returns the HTTP handler for the admin JSON API.
// Mounted at /api/{version}/server/{admin_path}/ with StripPrefix applied.
func (p *Panel) APIHandler() http.Handler {
	mux := http.NewServeMux()

	auth := p.requireAuth

	mux.Handle("GET /config/status", auth(http.HandlerFunc(p.apiStatus)))
	mux.Handle("GET /config/settings", auth(http.HandlerFunc(p.apiGetSettings)))
	mux.Handle("PATCH /config/settings", auth(http.HandlerFunc(p.apiPatchSettings)))
	mux.Handle("GET /config/info", auth(http.HandlerFunc(p.apiInfo)))
	mux.Handle("GET /config/metrics", auth(http.HandlerFunc(p.apiMetrics)))
	mux.Handle("GET /config/scheduler", auth(http.HandlerFunc(p.apiScheduler)))
	mux.Handle("POST /config/scheduler/{id}/run", auth(http.HandlerFunc(p.apiSchedulerRunNow)))
	mux.Handle("GET /config/logs", auth(http.HandlerFunc(p.apiLogs)))
	mux.Handle("GET /config/backup", auth(http.HandlerFunc(p.apiListBackups)))
	mux.Handle("POST /config/backup", auth(http.HandlerFunc(p.apiCreateBackup)))
	mux.Handle("GET /config/ssl", auth(http.HandlerFunc(p.apiSSLInfo)))
	mux.Handle("GET /config/email", auth(http.HandlerFunc(p.apiEmailInfo)))
	mux.Handle("POST /config/email/test", auth(http.HandlerFunc(p.apiEmailTest)))
	mux.Handle("GET /config/security/tokens", auth(http.HandlerFunc(p.apiListTokens)))
	mux.Handle("POST /config/security/tokens", auth(http.HandlerFunc(p.apiCreateToken)))
	mux.Handle("DELETE /config/security/tokens", auth(http.HandlerFunc(p.apiRevokeToken)))
	mux.Handle("GET /config/network/tor", auth(http.HandlerFunc(p.apiTorInfo)))
	mux.Handle("GET /config/network/geoip", auth(http.HandlerFunc(p.apiGeoIPInfo)))
	mux.Handle("GET /config/admins", auth(http.HandlerFunc(p.apiListAdmins)))
	mux.Handle("POST /config/admins", auth(http.HandlerFunc(p.apiInviteAdmin)))

	return mux
}


// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// handleDashboard renders the main admin dashboard
func (p *Panel) handleDashboard(w http.ResponseWriter, r *http.Request) {
	totalPastes, _ := p.CountPastes()
	recentPastes, _ := p.CountPastesRecent()
	adminCount, _ := p.CountAdmins()

	var content strings.Builder
	content.WriteString(`<div class="cards">`)
	content.WriteString(statCard("Total Pastes", fmt.Sprintf("%d", totalPastes), "📋"))
	content.WriteString(statCard("Last 24h", fmt.Sprintf("%d", recentPastes), "🕐"))
	content.WriteString(statCard("Admins", fmt.Sprintf("%d", adminCount), "👤"))
	content.WriteString(statCard("Uptime", p.uptime(), "⏱"))
	content.WriteString(`</div>`)

	content.WriteString(`<h2 class="section-title">Quick Links</h2>`)
	base := p.adminBasePath()
	content.WriteString(`<div class="cards">`)
	content.WriteString(linkCard("Settings", "Configure server settings", base+"/config/settings"))
	content.WriteString(linkCard("Logs", "View server logs", base+"/config/logs"))
	content.WriteString(linkCard("Backups", "Manage backups", base+"/config/backup"))
	content.WriteString(linkCard("Security", "Tokens and firewall", base+"/config/security/tokens"))
	content.WriteString(`</div>`)

	p.renderPage(w, r, "Dashboard", "dashboard", template.HTML(content.String()))
}

// handleLoginPage renders the admin login form
func (p *Panel) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	next := r.URL.Query().Get("next")
	p.renderLoginPage(w, r, "", next)
}

// handleLoginPost processes the login form submission
func (p *Panel) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		p.renderLoginPage(w, r, "Failed to parse form", "")
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	next := r.FormValue("next")

	if username == "" || password == "" {
		p.renderLoginPage(w, r, "Username and password are required", next)
		return
	}

	a, err := p.VerifyPassword(username, password)
	if err != nil {
		p.renderLoginPage(w, r, "Authentication error — please try again", next)
		return
	}
	if a == nil {
		p.renderLoginPage(w, r, "Invalid username or password", next)
		return
	}

	if err := p.createAdminSession(w, r, a.ID); err != nil {
		p.renderLoginPage(w, r, "Failed to create session — please try again", next)
		return
	}

	p.updateLastLogin(a.ID)

	dest := p.adminBasePath() + "/"
	if next != "" && strings.HasPrefix(next, "/") {
		dest = next
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// handleLogout clears the admin session and redirects to login
func (p *Panel) handleLogout(w http.ResponseWriter, r *http.Request) {
	p.deleteAdminSession(w, r)
	http.Redirect(w, r, p.adminBasePath()+"/login", http.StatusSeeOther)
}

// handleProfile renders the admin profile page
func (p *Panel) handleProfile(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	current := currentAdminUsername(r)
	if username != current {
		p.renderErrorPage(w, r, "You can only view your own profile")
		return
	}

	a, err := p.getAdmin(username)
	if err != nil || a == nil {
		p.renderErrorPage(w, r, "Admin account not found")
		return
	}

	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)
	content.WriteString(string(kvRow("Username", a.Username)))
	content.WriteString(string(kvRow("Email", a.Email)))
	content.WriteString(string(kvRow("Role", a.Role)))
	if a.LastLogin.Valid {
		content.WriteString(string(kvRow("Last Login", time.Unix(a.LastLogin.Int64, 0).Format(time.RFC1123))))
	} else {
		content.WriteString(string(kvRow("Last Login", "Never")))
	}
	content.WriteString(string(kvRow("Member Since", time.Unix(a.CreatedAt, 0).Format(time.RFC1123))))
	content.WriteString(`</div>`)

	base := p.adminBasePath() + "/" + username
	content.WriteString(fmt.Sprintf(`<div style="margin-top:1.5rem"><a href="%s/profile/edit" class="btn">Edit Profile</a></div>`, base))

	p.renderPage(w, r, "Profile", "profile", template.HTML(content.String()))
}

// handleProfilePost processes profile edit submissions
func (p *Panel) handleProfilePost(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	current := currentAdminUsername(r)
	if username != current {
		p.renderErrorPage(w, r, "You can only edit your own profile")
		return
	}
	http.Redirect(w, r, p.adminBasePath()+"/"+username+"/profile", http.StatusSeeOther)
}

// handlePreferences renders the admin preferences page
func (p *Panel) handlePreferences(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	current := currentAdminUsername(r)
	if username != current {
		p.renderErrorPage(w, r, "You can only view your own preferences")
		return
	}

	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)
	content.WriteString(string(kvRow("Theme", "dark (default)")))
	content.WriteString(string(kvRow("Language", "en")))
	content.WriteString(`</div>`)
	p.renderPage(w, r, "Preferences", "preferences", template.HTML(content.String()))
}

// handlePreferencesPost processes preferences form submissions
func (p *Panel) handlePreferencesPost(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	current := currentAdminUsername(r)
	if username != current {
		p.renderErrorPage(w, r, "You can only edit your own preferences")
		return
	}
	http.Redirect(w, r, p.adminBasePath()+"/"+username+"/preferences", http.StatusSeeOther)
}

// handleNotifications renders the notifications page
func (p *Panel) handleNotifications(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	current := currentAdminUsername(r)
	if username != current {
		p.renderErrorPage(w, r, "You can only view your own notifications")
		return
	}
	var content strings.Builder
	content.WriteString(`<p class="muted">No notifications.</p>`)
	p.renderPage(w, r, "Notifications", "notifications", template.HTML(content.String()))
}

// handleConfigRoot redirects /config/ to /config/settings
func (p *Panel) handleConfigRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, p.adminBasePath()+"/config/settings", http.StatusSeeOther)
}

// handleSettings renders the server settings page
func (p *Panel) handleSettings(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)

	if p.cfg.AppCfg != nil {
		c := p.cfg.AppCfg
		content.WriteString(string(kvRow("Server Title", c.ServerTitle)))
		content.WriteString(string(kvRow("FQDN", c.FQDN)))
		content.WriteString(string(kvRow("Admin Name", c.AdminName)))
		content.WriteString(string(kvRow("Admin Email", c.AdminMail)))
		content.WriteString(string(kvRow("Max Title Length", fmt.Sprintf("%d", c.TitleMaxLen))))
		content.WriteString(string(kvRow("Max Body Length", fmt.Sprintf("%d", c.BodyMaxLen))))
		content.WriteString(string(kvRow("Max Lifetime", fmt.Sprintf("%d days", c.MaxLifeTime))))
		content.WriteString(string(kvRow("Public Access", fmt.Sprintf("%v", c.Public))))
		content.WriteString(string(kvRow("API Version", c.APIVersion)))
		content.WriteString(string(kvRow("Mode", c.Mode)))
	} else {
		content.WriteString(string(kvRow("Status", "No application config available")))
	}
	content.WriteString(`</div>`)

	p.renderPage(w, r, "Settings", "settings", template.HTML(content.String()))
}

// handleSettingsPost processes settings form submissions
func (p *Panel) handleSettingsPost(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, p.adminBasePath()+"/config/settings", http.StatusSeeOther)
}

// handleSSL renders the SSL/TLS settings page
func (p *Panel) handleSSL(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)
	content.WriteString(string(kvRow("TLS Status", "Managed by reverse proxy or OS")))
	content.WriteString(string(kvRow("ACME/Let's Encrypt", "Not configured")))
	content.WriteString(`</div>`)
	p.renderPage(w, r, "SSL / TLS", "ssl", template.HTML(content.String()))
}

// handleEmail renders the email settings page
func (p *Panel) handleEmail(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)
	content.WriteString(string(kvRow("SMTP", "Not configured")))
	content.WriteString(`</div>`)
	p.renderPage(w, r, "Email", "email", template.HTML(content.String()))
}

// handleEmailPost processes email settings form submissions
func (p *Panel) handleEmailPost(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, p.adminBasePath()+"/config/email", http.StatusSeeOther)
}

// handleScheduler renders the scheduler status page
func (p *Panel) handleScheduler(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder

	p.mu.RLock()
	sched := p.sched
	p.mu.RUnlock()

	if sched == nil {
		content.WriteString(`<p class="muted">Scheduler not available.</p>`)
		p.renderPage(w, r, "Scheduler", "scheduler", template.HTML(content.String()))
		return
	}

	tasks := sched.ListTasks()
	if len(tasks) == 0 {
		content.WriteString(`<p class="muted">No scheduled tasks registered.</p>`)
		p.renderPage(w, r, "Scheduler", "scheduler", template.HTML(content.String()))
		return
	}

	content.WriteString(`<table class="table"><thead><tr>`)
	content.WriteString(`<th>ID</th><th>Name</th><th>Schedule</th><th>Last Run</th><th>Next Run</th><th>Status</th><th>Runs</th><th>Fails</th>`)
	content.WriteString(`</tr></thead><tbody>`)

	for _, t := range tasks {
		lastRun := "Never"
		if !t.LastRun.IsZero() {
			lastRun = t.LastRun.Format("2006-01-02 15:04:05")
		}
		nextRun := "—"
		if !t.NextRun.IsZero() {
			nextRun = t.NextRun.Format("2006-01-02 15:04:05")
		}
		statusBadge := badgeClass(string(t.LastStatus))
		content.WriteString(fmt.Sprintf(
			`<tr><td>%s</td><td>%s</td><td><code>%s</code></td><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td>%d</td></tr>`,
			template.HTMLEscapeString(t.ID),
			template.HTMLEscapeString(t.Name),
			template.HTMLEscapeString(t.Schedule),
			lastRun, nextRun,
			statusBadge,
			t.RunCount, t.FailCount,
		))
	}
	content.WriteString(`</tbody></table>`)

	p.renderPage(w, r, "Scheduler", "scheduler", template.HTML(content.String()))
}

// handleSchedulerPost handles run-now requests from the UI
func (p *Panel) handleSchedulerPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		p.renderErrorPage(w, r, "Bad request")
		return
	}
	id := r.FormValue("task_id")

	p.mu.RLock()
	sched := p.sched
	p.mu.RUnlock()

	if sched != nil && id != "" {
		_ = sched.RunNow(id)
	}
	http.Redirect(w, r, p.adminBasePath()+"/config/scheduler", http.StatusSeeOther)
}

// handleLogs renders the application log page
func (p *Panel) handleLogs(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	logFile := filepath.Join(p.cfg.DataDir, "logs", "caspaste.log")
	tail, err := tailFile(logFile, 200)
	if err != nil {
		content.WriteString(fmt.Sprintf(`<p class="muted">Log file unavailable: %s</p>`, template.HTMLEscapeString(err.Error())))
	} else {
		content.WriteString(`<pre class="log-view">`)
		content.WriteString(template.HTMLEscapeString(tail))
		content.WriteString(`</pre>`)
	}
	p.renderPage(w, r, "Logs", "logs", template.HTML(content.String()))
}

// handleLogsAudit renders the audit log page
func (p *Panel) handleLogsAudit(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	logFile := filepath.Join(p.cfg.DataDir, "logs", "audit.log")
	tail, err := tailFile(logFile, 200)
	if err != nil {
		content.WriteString(fmt.Sprintf(`<p class="muted">Audit log unavailable: %s</p>`, template.HTMLEscapeString(err.Error())))
	} else {
		content.WriteString(`<pre class="log-view">`)
		content.WriteString(template.HTMLEscapeString(tail))
		content.WriteString(`</pre>`)
	}
	p.renderPage(w, r, "Audit Log", "logs", template.HTML(content.String()))
}

// handleBackup renders the backup management page
func (p *Panel) handleBackup(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	backups, err := listBackupFiles(p.cfg.BackupDir)
	if err != nil {
		content.WriteString(fmt.Sprintf(`<p class="muted">Backup directory unavailable: %s</p>`, template.HTMLEscapeString(err.Error())))
		p.renderPage(w, r, "Backup", "backup", template.HTML(content.String()))
		return
	}
	if len(backups) == 0 {
		content.WriteString(`<p class="muted">No backups found.</p>`)
	} else {
		content.WriteString(`<table class="table"><thead><tr><th>File</th><th>Size</th><th>Modified</th></tr></thead><tbody>`)
		for _, b := range backups {
			content.WriteString(fmt.Sprintf(
				`<tr><td>%s</td><td>%s</td><td>%s</td></tr>`,
				template.HTMLEscapeString(b.Name),
				humanSize(b.Size),
				b.ModTime.Format("2006-01-02 15:04:05"),
			))
		}
		content.WriteString(`</tbody></table>`)
	}

	base := p.adminBasePath()
	content.WriteString(fmt.Sprintf(`
<form method="POST" action="%s/config/backup" style="margin-top:1.5rem">
  <input type="hidden" name="csrf_token" value="">
  <button type="submit" class="btn">Create Backup Now</button>
</form>`, base))

	p.renderPage(w, r, "Backup", "backup", template.HTML(content.String()))
}

// handleBackupPost triggers a manual backup
func (p *Panel) handleBackupPost(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, p.adminBasePath()+"/config/backup", http.StatusSeeOther)
}

// handleUpdates renders the updates page
func (p *Panel) handleUpdates(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)
	content.WriteString(string(kvRow("Update Channel", "stable")))
	content.WriteString(string(kvRow("Status", "Update checks not configured")))
	content.WriteString(`</div>`)
	p.renderPage(w, r, "Updates", "updates", template.HTML(content.String()))
}

// handleInfo renders the server information page
func (p *Panel) handleInfo(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)

	if p.cfg.AppCfg != nil {
		c := p.cfg.AppCfg
		content.WriteString(string(kvRow("Version", c.Version)))
		content.WriteString(string(kvRow("Build Commit", c.BuildCommit)))
		content.WriteString(string(kvRow("Build Date", c.BuildDate)))
		content.WriteString(string(kvRow("Mode", c.Mode)))
		content.WriteString(string(kvRow("FQDN", c.FQDN)))
		content.WriteString(string(kvRow("Data Directory", p.cfg.DataDir)))
		content.WriteString(string(kvRow("Config Directory", p.cfg.ConfigDir)))
		content.WriteString(string(kvRow("Backup Directory", p.cfg.BackupDir)))
	}

	content.WriteString(string(kvRow("Go Version", runtime.Version())))
	content.WriteString(string(kvRow("OS/Arch", runtime.GOOS+"/"+runtime.GOARCH)))
	content.WriteString(string(kvRow("Goroutines", fmt.Sprintf("%d", runtime.NumGoroutine()))))
	content.WriteString(string(kvRow("Uptime", p.uptime())))
	content.WriteString(string(kvRow("Start Time", p.cfg.StartTime.Format(time.RFC1123))))
	content.WriteString(`</div>`)

	p.renderPage(w, r, "Server Info", "info", template.HTML(content.String()))
}

// handleMetrics renders the metrics page
func (p *Panel) handleMetrics(w http.ResponseWriter, r *http.Request) {
	totalPastes, _ := p.CountPastes()
	recentPastes, _ := p.CountPastesRecent()
	adminCount, _ := p.CountAdmins()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)
	content.WriteString(string(kvRow("Total Pastes", fmt.Sprintf("%d", totalPastes))))
	content.WriteString(string(kvRow("Pastes (24h)", fmt.Sprintf("%d", recentPastes))))
	content.WriteString(string(kvRow("Admin Accounts", fmt.Sprintf("%d", adminCount))))
	content.WriteString(string(kvRow("Goroutines", fmt.Sprintf("%d", runtime.NumGoroutine()))))
	content.WriteString(string(kvRow("Heap Alloc", humanSize(int64(ms.HeapAlloc)))))
	content.WriteString(string(kvRow("Heap Sys", humanSize(int64(ms.HeapSys)))))
	content.WriteString(string(kvRow("Heap Objects", fmt.Sprintf("%d", ms.HeapObjects))))
	content.WriteString(string(kvRow("GC Cycles", fmt.Sprintf("%d", ms.NumGC))))
	content.WriteString(string(kvRow("Uptime", p.uptime())))
	content.WriteString(`</div>`)

	p.renderPage(w, r, "Metrics", "metrics", template.HTML(content.String()))
}

// handleNetworkTor renders the Tor hidden service page
func (p *Panel) handleNetworkTor(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)
	content.WriteString(string(kvRow("Tor Status", "Not configured")))
	content.WriteString(string(kvRow("Onion Address", "—")))
	content.WriteString(`</div>`)
	p.renderPage(w, r, "Tor Hidden Service", "tor", template.HTML(content.String()))
}

// handleNetworkGeoIP renders the GeoIP configuration page
func (p *Panel) handleNetworkGeoIP(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)
	content.WriteString(string(kvRow("GeoIP Status", "Not configured")))
	content.WriteString(string(kvRow("Database Path", "—")))
	content.WriteString(`</div>`)
	p.renderPage(w, r, "GeoIP", "geoip", template.HTML(content.String()))
}

// handleSecurityAuth renders the authentication overview page
func (p *Panel) handleSecurityAuth(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	base := p.adminBasePath()

	content.WriteString(`<div class="cards">`)
	content.WriteString(linkCard("OIDC", "OpenID Connect provider configuration", base+"/config/security/auth/oidc"))
	content.WriteString(linkCard("LDAP", "LDAP / Active Directory configuration", base+"/config/security/auth/ldap"))
	content.WriteString(`</div>`)

	content.WriteString(`<div class="kv-list" style="margin-top:1.5rem">`)
	content.WriteString(string(kvRow("OIDC", "Not configured")))
	content.WriteString(string(kvRow("LDAP", "Not configured")))
	content.WriteString(`</div>`)

	p.renderPage(w, r, "Authentication", "security-auth", template.HTML(content.String()))
}

// handleSecurityOIDC renders the OIDC provider page
func (p *Panel) handleSecurityOIDC(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)
	content.WriteString(string(kvRow("Provider", "Not configured")))
	content.WriteString(string(kvRow("Client ID", "—")))
	content.WriteString(string(kvRow("Discovery URL", "—")))
	content.WriteString(`</div>`)
	p.renderPage(w, r, "OIDC", "security-auth", template.HTML(content.String()))
}

// handleSecurityLDAP renders the LDAP configuration page
func (p *Panel) handleSecurityLDAP(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)
	content.WriteString(string(kvRow("LDAP Server", "Not configured")))
	content.WriteString(string(kvRow("Base DN", "—")))
	content.WriteString(`</div>`)
	p.renderPage(w, r, "LDAP", "security-auth", template.HTML(content.String()))
}

// handleSecurityTokens renders the API token management page
func (p *Panel) handleSecurityTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := p.listTokens()
	var content strings.Builder

	if err != nil {
		content.WriteString(fmt.Sprintf(`<div class="alert alert-err">%s</div>`, template.HTMLEscapeString(err.Error())))
	} else if len(tokens) == 0 {
		content.WriteString(`<p class="muted">No API tokens issued.</p>`)
	} else {
		content.WriteString(`<table class="table"><thead><tr><th>Name</th><th>Prefix</th><th>Created</th><th>Last Used</th><th>Expires</th><th></th></tr></thead><tbody>`)
		for _, t := range tokens {
			expires := "Never"
			if t.ExpiresAt.Valid {
				expires = time.Unix(t.ExpiresAt.Int64, 0).Format("2006-01-02")
			}
			lastUsed := "Never"
			if t.LastUsedAt.Valid {
				lastUsed = time.Unix(t.LastUsedAt.Int64, 0).Format("2006-01-02 15:04")
			}
			base := p.adminBasePath()
			content.WriteString(fmt.Sprintf(
				`<tr><td>%s</td><td><code>%s</code></td><td>%s</td><td>%s</td><td>%s</td>`+
					`<td><form method="POST" action="%s/config/security/tokens" style="display:inline">`+
					`<input type="hidden" name="csrf_token" value="">`+
					`<input type="hidden" name="action" value="revoke">`+
					`<input type="hidden" name="token_id" value="%d">`+
					`<button type="submit" class="btn btn-sm btn-danger">Revoke</button></form></td></tr>`,
				template.HTMLEscapeString(t.Name),
				template.HTMLEscapeString(t.Prefix),
				time.Unix(t.CreatedAt, 0).Format("2006-01-02"),
				lastUsed, expires, base, t.ID,
			))
		}
		content.WriteString(`</tbody></table>`)
	}

	base := p.adminBasePath()
	content.WriteString(fmt.Sprintf(`
<h2 class="section-title" style="margin-top:2rem">Create New Token</h2>
<form method="POST" action="%s/config/security/tokens">
  <input type="hidden" name="csrf_token" value="">
  <input type="hidden" name="action" value="create">
  <div class="form-group">
    <label for="token_name">Token Name</label>
    <input type="text" id="token_name" name="token_name" required maxlength="64" placeholder="e.g. CI Deploy Key">
  </div>
  <div class="form-group">
    <label for="token_expires">Expires (days, 0 = never)</label>
    <input type="number" id="token_expires" name="token_expires" value="0" min="0" max="3650">
  </div>
  <button type="submit" class="btn">Create Token</button>
</form>`, base))

	p.renderPage(w, r, "API Tokens", "security-tokens", template.HTML(content.String()))
}

// handleSecurityTokensPost processes token create/revoke form submissions
func (p *Panel) handleSecurityTokensPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		p.renderErrorPage(w, r, "Bad request")
		return
	}
	action := r.FormValue("action")
	switch action {
	case "create":
		name := strings.TrimSpace(r.FormValue("token_name"))
		if name == "" {
			http.Redirect(w, r, p.adminBasePath()+"/config/security/tokens", http.StatusSeeOther)
			return
		}
		adminID := currentAdminID(r)
		_ = p.createToken(adminID, name, 0)
	case "revoke":
	}
	http.Redirect(w, r, p.adminBasePath()+"/config/security/tokens", http.StatusSeeOther)
}

// handleSecurityFirewall renders the firewall rules page
func (p *Panel) handleSecurityFirewall(w http.ResponseWriter, r *http.Request) {
	var content strings.Builder
	content.WriteString(`<div class="kv-list">`)
	content.WriteString(string(kvRow("Rate Limiting", "Configured in YAML")))
	content.WriteString(string(kvRow("Blocked IPs", "—")))
	content.WriteString(string(kvRow("Trusted Proxies", p.trustedProxies())))
	content.WriteString(`</div>`)
	p.renderPage(w, r, "Firewall", "security-firewall", template.HTML(content.String()))
}

// trustedProxies returns comma-separated trusted proxy list from AppCfg
func (p *Panel) trustedProxies() string {
	if p.cfg.AppCfg == nil || len(p.cfg.AppCfg.TrustedProxies) == 0 {
		return "None"
	}
	return strings.Join(p.cfg.AppCfg.TrustedProxies, ", ")
}

// statCard renders a summary stat card
func statCard(label, value, icon string) string {
	return fmt.Sprintf(
		`<div class="card"><div class="card-icon">%s</div><div class="card-value">%s</div><div class="card-label">%s</div></div>`,
		icon, template.HTMLEscapeString(value), template.HTMLEscapeString(label),
	)
}

// linkCard renders a navigation card with a link
func linkCard(title, desc, href string) string {
	return fmt.Sprintf(
		`<div class="card"><a href="%s" class="card-link"><strong>%s</strong><br><small>%s</small></a></div>`,
		template.HTMLEscapeString(href),
		template.HTMLEscapeString(title),
		template.HTMLEscapeString(desc),
	)
}

// badgeClass wraps a task status in an appropriate badge span
func badgeClass(status string) string {
	cls := "badge-neutral"
	switch status {
	case "success":
		cls = "badge-ok"
	case "failed":
		cls = "badge-err"
	case "running":
		cls = "badge-info"
	}
	return fmt.Sprintf(`<span class="badge %s">%s</span>`, cls, template.HTMLEscapeString(status))
}

// tailFile reads up to n lines from the end of a text file
func tailFile(path string, n int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n"), nil
}

// backupEntry holds metadata for a single backup file
type backupEntry struct {
	Name    string
	Size    int64
	ModTime time.Time
}

// listBackupFiles returns backup file metadata from the given directory
func listBackupFiles(dir string) ([]backupEntry, error) {
	if dir == "" {
		return nil, fmt.Errorf("no backup directory configured")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []backupEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, backupEntry{
			Name:    e.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	return out, nil
}

// humanSize converts bytes to a human-readable string
func humanSize(n int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case n >= GB:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(GB))
	case n >= MB:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(MB))
	case n >= KB:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(KB))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

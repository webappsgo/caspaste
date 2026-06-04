
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

// pageData holds the data needed to render any admin page
type pageData struct {
	Title         string
	Page          string
	AdminUsername string
	Version       string
	BasePath      string
	CSRFToken     string
	Content       template.HTML
}

// renderPage writes a full admin HTML page
func (p *Panel) renderPage(w http.ResponseWriter, r *http.Request, title, page string, content template.HTML) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")

	username := currentAdminUsername(r)
	version := ""
	if p.cfg.AppCfg != nil {
		version = p.cfg.AppCfg.Version
	}
	basePath := p.adminBasePath()

	pd := pageData{
		Title:         title,
		Page:          page,
		AdminUsername: username,
		Version:       version,
		BasePath:      basePath,
	}

	html := p.buildLayout(pd, content)
	_, _ = w.Write([]byte(html))
}

// renderErrorPage renders a standalone error without the full sidebar layout
func (p *Panel) renderErrorPage(w http.ResponseWriter, r *http.Request, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="robots" content="noindex,nofollow">
<title>Admin — Error</title>
<style>
:root{--bg:#1a1a2e;--bg2:#16213e;--accent:#e94560;--text:#eaeaea;--text2:#b8b8b8;--border:#2d3748;}
*{box-sizing:border-box;margin:0;padding:0;}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;
background:var(--bg);color:var(--text);min-height:100vh;display:flex;align-items:center;justify-content:center;}
.card{background:var(--bg2);border:1px solid var(--border);border-radius:8px;padding:2rem;max-width:480px;text-align:center;}
.card h1{color:var(--accent);margin-bottom:1rem;}
.card p{color:var(--text2);margin-bottom:1.5rem;}
.btn{display:inline-block;padding:.5rem 1.25rem;background:var(--accent);color:#fff;
border-radius:4px;text-decoration:none;font-size:.875rem;}
</style></head>
<body>
<div class="card">
  <h1>Admin Error</h1>
  <p>%s</p>
  <a class="btn" href="%s/">Back to Admin</a>
</div>
</body></html>`, template.HTMLEscapeString(msg), p.adminBasePath())
	_, _ = w.Write([]byte(html))
}

// renderLoginPage renders the admin login form
func (p *Panel) renderLoginPage(w http.ResponseWriter, r *http.Request, errorMsg, nextURL string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	errHTML := ""
	if errorMsg != "" {
		errHTML = fmt.Sprintf(`<div class="alert alert-error">%s</div>`,
			template.HTMLEscapeString(errorMsg))
	}

	basePath := p.adminBasePath()
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="robots" content="noindex,nofollow">
<title>Admin Login</title>
<style>
:root{--bg:#1a1a2e;--bg2:#16213e;--bg3:#0f3460;--accent:#e94560;--text:#eaeaea;--text2:#b8b8b8;--border:#2d3748;--success:#4ade80;--error:#ef4444;}
*{box-sizing:border-box;margin:0;padding:0;}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:var(--bg);color:var(--text);min-height:100vh;display:flex;align-items:center;justify-content:center;}
.login-wrap{width:100%%;max-width:400px;padding:1rem;}
.card{background:var(--bg2);border:1px solid var(--border);border-radius:8px;padding:2rem;}
.card-title{font-size:1.5rem;font-weight:600;margin-bottom:1.5rem;text-align:center;color:var(--accent);}
.form-group{margin-bottom:1rem;}
label{display:block;margin-bottom:.4rem;font-size:.875rem;color:var(--text2);}
input[type=text],input[type=password]{width:100%%;padding:.625rem .75rem;background:var(--bg3);border:1px solid var(--border);border-radius:4px;color:var(--text);font-size:.875rem;outline:none;}
input:focus{border-color:var(--accent);}
.btn{width:100%%;padding:.75rem;background:var(--accent);color:#fff;border:none;border-radius:4px;cursor:pointer;font-size:1rem;margin-top:.5rem;}
.btn:hover{background:#d63d55;}
.alert-error{background:rgba(239,68,68,.15);border:1px solid var(--error);color:var(--error);padding:.75rem;border-radius:4px;margin-bottom:1rem;font-size:.875rem;}
</style>
</head>
<body>
<div class="login-wrap">
  <div class="card">
    <div class="card-title">Admin Panel</div>
    %s
    <form method="POST" action="%s/login">
      <input type="hidden" name="csrf_token" value="">
      <input type="hidden" name="next" value="%s">
      <div class="form-group">
        <label for="username">Username</label>
        <input type="text" id="username" name="username" autocomplete="username" autofocus required>
      </div>
      <div class="form-group">
        <label for="password">Password</label>
        <input type="password" id="password" name="password" autocomplete="current-password" required>
      </div>
      <button type="submit" class="btn">Sign In</button>
    </form>
  </div>
</div>
</body></html>`, errHTML, basePath, template.HTMLEscapeString(nextURL))
	_, _ = w.Write([]byte(html))
}

// renderSetupPage renders the first-run setup wizard
func (p *Panel) renderSetupPage(w http.ResponseWriter, r *http.Request, token, errorMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	errHTML := ""
	if errorMsg != "" {
		errHTML = fmt.Sprintf(`<div class="alert alert-error">%s</div>`,
			template.HTMLEscapeString(errorMsg))
	}

	basePath := p.adminBasePath()
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="robots" content="noindex,nofollow">
<title>Admin Setup</title>
<style>
:root{--bg:#1a1a2e;--bg2:#16213e;--bg3:#0f3460;--accent:#e94560;--text:#eaeaea;--text2:#b8b8b8;--border:#2d3748;--error:#ef4444;}
*{box-sizing:border-box;margin:0;padding:0;}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:var(--bg);color:var(--text);min-height:100vh;display:flex;align-items:center;justify-content:center;}
.login-wrap{width:100%%;max-width:480px;padding:1rem;}
.card{background:var(--bg2);border:1px solid var(--border);border-radius:8px;padding:2rem;}
.card-title{font-size:1.5rem;font-weight:600;margin-bottom:.5rem;color:var(--accent);}
.card-subtitle{color:var(--text2);font-size:.875rem;margin-bottom:1.5rem;}
.form-group{margin-bottom:1rem;}
label{display:block;margin-bottom:.4rem;font-size:.875rem;color:var(--text2);}
input[type=text],input[type=email],input[type=password]{width:100%%;padding:.625rem .75rem;background:var(--bg3);border:1px solid var(--border);border-radius:4px;color:var(--text);font-size:.875rem;outline:none;}
input:focus{border-color:var(--accent);}
.btn{width:100%%;padding:.75rem;background:var(--accent);color:#fff;border:none;border-radius:4px;cursor:pointer;font-size:1rem;margin-top:.5rem;}
.btn:hover{background:#d63d55;}
.alert-error{background:rgba(239,68,68,.15);border:1px solid var(--error);color:var(--error);padding:.75rem;border-radius:4px;margin-bottom:1rem;font-size:.875rem;}
.hint{font-size:.75rem;color:var(--text2);margin-top:.25rem;}
</style>
</head>
<body>
<div class="login-wrap">
  <div class="card">
    <div class="card-title">Admin Setup</div>
    <p class="card-subtitle">Create the first admin account to get started.</p>
    %s
    <form method="POST" action="%s/config/setup">
      <input type="hidden" name="setup_token" value="%s">
      <div class="form-group">
        <label for="username">Admin Username</label>
        <input type="text" id="username" name="username" autocomplete="username" autofocus required minlength="3" maxlength="64">
      </div>
      <div class="form-group">
        <label for="email">Email (optional)</label>
        <input type="email" id="email" name="email" autocomplete="email">
      </div>
      <div class="form-group">
        <label for="password">Password</label>
        <input type="password" id="password" name="password" autocomplete="new-password" required minlength="8">
        <p class="hint">Minimum 8 characters. Stored as Argon2id hash.</p>
      </div>
      <div class="form-group">
        <label for="password_confirm">Confirm Password</label>
        <input type="password" id="password_confirm" name="password_confirm" autocomplete="new-password" required minlength="8">
      </div>
      <button type="submit" class="btn">Create Admin Account</button>
    </form>
  </div>
</div>
</body></html>`, errHTML, basePath, template.HTMLEscapeString(token))
	_, _ = w.Write([]byte(html))
}

// buildLayout assembles the full admin panel HTML
func (p *Panel) buildLayout(pd pageData, content template.HTML) string {
	base := pd.BasePath

	// Build sidebar nav links with active-state marker
	nav := p.buildSidebarNav(pd.Page, base, pd.AdminUsername)

	serverTitle := "CasPaste"
	if p.cfg.AppCfg != nil && p.cfg.AppCfg.ServerTitle != "" {
		serverTitle = p.cfg.AppCfg.ServerTitle
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="robots" content="noindex, nofollow">
<title>%s — %s Admin</title>
<style>
:root{
  --bg:#1a1a2e;--bg2:#16213e;--bg3:#0f3460;
  --text:#eaeaea;--text2:#b8b8b8;--text3:#7a7a8c;
  --accent:#e94560;--success:#4ade80;--warning:#fbbf24;--error:#ef4444;
  --border:#2d3748;--radius:6px;
}
*{box-sizing:border-box;margin:0;padding:0;}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;
  background:var(--bg);color:var(--text);min-height:100vh;}
a{color:inherit;text-decoration:none;}
.admin-layout{display:flex;min-height:100vh;}

/* Sidebar */
.sidebar{width:224px;background:var(--bg2);border-right:1px solid var(--border);
  display:flex;flex-direction:column;flex-shrink:0;overflow-y:auto;}
.sidebar-header{padding:.875rem 1rem;border-bottom:1px solid var(--border);}
.sidebar-header .logo{font-size:1.125rem;font-weight:700;color:var(--accent);}
.sidebar-header .subtitle{font-size:.7rem;color:var(--text3);margin-top:.125rem;text-transform:uppercase;letter-spacing:.05em;}
.nav-section{margin-top:.5rem;}
.nav-section-title{padding:.375rem 1rem;font-size:.65rem;text-transform:uppercase;
  color:var(--text3);letter-spacing:.08em;font-weight:600;}
.nav-link{display:flex;align-items:center;gap:.5rem;padding:.5rem 1rem;
  font-size:.8125rem;color:var(--text2);border-left:3px solid transparent;
  transition:background .15s,color .15s;}
.nav-link:hover{background:var(--bg3);color:var(--text);}
.nav-link.active{background:var(--bg3);color:var(--accent);border-left-color:var(--accent);}
.nav-icon{width:16px;text-align:center;flex-shrink:0;}

/* Main content */
.main-content{flex:1;display:flex;flex-direction:column;min-width:0;}

/* Header */
.topbar{height:52px;background:var(--bg2);border-bottom:1px solid var(--border);
  display:flex;align-items:center;justify-content:space-between;padding:0 1.25rem;flex-shrink:0;}
.breadcrumb{display:flex;align-items:center;gap:.375rem;font-size:.8125rem;color:var(--text2);}
.breadcrumb a:hover{color:var(--text);}
.breadcrumb-sep{color:var(--text3);}
.topbar-right{display:flex;align-items:center;gap:.75rem;}
.status-dot{width:8px;height:8px;border-radius:50%%;background:var(--success);flex-shrink:0;}
.admin-name{font-size:.8125rem;color:var(--text2);}
.btn{padding:.375rem .875rem;border:none;border-radius:var(--radius);cursor:pointer;
  font-size:.8125rem;transition:background .15s;}
.btn-ghost{background:transparent;color:var(--text2);border:1px solid var(--border);}
.btn-ghost:hover{background:var(--bg3);color:var(--text);}
.btn-primary{background:var(--accent);color:#fff;}
.btn-primary:hover{background:#d63d55;}
.btn-danger{background:var(--error);color:#fff;}
.btn-danger:hover{background:#c53030;}
.btn-sm{padding:.25rem .625rem;font-size:.75rem;}

/* Page body */
.page-body{flex:1;padding:1.25rem;overflow-y:auto;}
.page-heading{font-size:1.25rem;font-weight:600;margin-bottom:1.25rem;}

/* Cards */
.card{background:var(--bg2);border:1px solid var(--border);border-radius:var(--radius);padding:1.25rem;margin-bottom:1rem;}
.card-title{font-size:.875rem;font-weight:600;color:var(--text2);margin-bottom:.875rem;
  text-transform:uppercase;letter-spacing:.04em;}
.card-body{font-size:.875rem;}

/* Stats grid */
.stats-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:.875rem;margin-bottom:1.25rem;}
.stat-card{background:var(--bg2);border:1px solid var(--border);border-radius:var(--radius);
  padding:1.25rem;text-align:center;}
.stat-value{font-size:1.875rem;font-weight:700;color:var(--accent);}
.stat-label{font-size:.75rem;color:var(--text2);margin-top:.25rem;}

/* Tables */
.table-wrap{overflow-x:auto;}
table,.table{width:100%%;border-collapse:collapse;font-size:.8125rem;}
th{padding:.5rem .75rem;background:var(--bg3);color:var(--text2);
  text-align:left;font-weight:600;border-bottom:1px solid var(--border);}
td{padding:.5rem .75rem;border-bottom:1px solid var(--border);color:var(--text);}
tr:last-child td{border-bottom:none;}
tr:hover td{background:rgba(255,255,255,.02);}

/* Forms */
.form-group{margin-bottom:.875rem;}
.form-group label{display:block;margin-bottom:.3rem;font-size:.8125rem;color:var(--text2);font-weight:500;}
.form-control{width:100%%;padding:.5rem .75rem;background:var(--bg3);border:1px solid var(--border);
  border-radius:var(--radius);color:var(--text);font-size:.875rem;outline:none;}
.form-control:focus{border-color:var(--accent);}
.form-hint{font-size:.7rem;color:var(--text3);margin-top:.25rem;}
.form-actions{display:flex;gap:.5rem;margin-top:1rem;}

/* Badges */
.badge{display:inline-block;padding:.125rem .5rem;border-radius:999px;font-size:.7rem;font-weight:600;}
.badge-success{background:rgba(74,222,128,.15);color:var(--success);}
.badge-warning{background:rgba(251,191,36,.15);color:var(--warning);}
.badge-error{background:rgba(239,68,68,.15);color:var(--error);}
.badge-neutral{background:rgba(255,255,255,.08);color:var(--text2);}

/* Alerts */
.alert{padding:.75rem 1rem;border-radius:var(--radius);margin-bottom:1rem;font-size:.875rem;}
.alert-success{background:rgba(74,222,128,.1);border:1px solid var(--success);color:var(--success);}
.alert-error{background:rgba(239,68,68,.1);border:1px solid var(--error);color:var(--error);}
.alert-warning{background:rgba(251,191,36,.1);border:1px solid var(--warning);color:var(--warning);}
.alert-info{background:rgba(99,179,237,.1);border:1px solid #63b3ed;color:#63b3ed;}

/* KV list */
.kv-list{display:grid;grid-template-columns:auto 1fr;gap:.5rem .75rem;align-items:baseline;font-size:.8125rem;}
.kv-key{color:var(--text2);white-space:nowrap;}
.kv-val{color:var(--text);font-family:ui-monospace,'Cascadia Code',monospace;word-break:break-all;}

/* Footer */
.footer{height:36px;background:var(--bg2);border-top:1px solid var(--border);
  display:flex;align-items:center;justify-content:center;gap:1.5rem;flex-shrink:0;
  font-size:.7rem;color:var(--text3);}
.footer a:hover{color:var(--text2);}

/* Flex card grid (dashboard quick links) */
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:.875rem;margin-bottom:1.25rem;}
.card-icon{font-size:2rem;margin-bottom:.5rem;}
.card-value{font-size:1.5rem;font-weight:700;color:var(--accent);}
.card-label{font-size:.75rem;color:var(--text2);margin-top:.25rem;}
.card-link{display:block;color:var(--text);font-size:.875rem;}
.card-link:hover{color:var(--accent);}

/* Utility */
.muted{color:var(--text3);font-size:.875rem;}
.section-title{font-size:1rem;font-weight:600;margin:.875rem 0 .625rem;}
.log-view{background:var(--bg2);border:1px solid var(--border);border-radius:var(--radius);
  padding:.75rem;font-family:ui-monospace,'Cascadia Code',monospace;font-size:.75rem;
  white-space:pre-wrap;word-break:break-all;max-height:600px;overflow-y:auto;color:var(--text2);}

/* Extra badge variants */
.badge-ok{background:rgba(74,222,128,.15);color:var(--success);}
.badge-err{background:rgba(239,68,68,.15);color:var(--error);}
.badge-info{background:rgba(99,179,237,.1);color:#63b3ed;}

/* Alert error alias */
.alert-err{background:rgba(239,68,68,.1);border:1px solid var(--error);color:var(--error);}

/* Mobile */
@media(max-width:768px){
  .sidebar{display:none;}
  .admin-layout{flex-direction:column;}
}
</style>
</head>
<body>
<div class="admin-layout">
  <nav class="sidebar">
    <div class="sidebar-header">
      <div class="logo">%s</div>
      <div class="subtitle">Admin Panel</div>
    </div>
    %s
  </nav>
  <div class="main-content">
    <header class="topbar">
      <div class="breadcrumb">
        <a href="%s/">Admin</a>
        %s
      </div>
      <div class="topbar-right">
        <div class="status-dot" title="Server OK"></div>
        <span class="admin-name">%s</span>
        <a href="%s/logout" class="btn btn-ghost btn-sm">Logout</a>
      </div>
    </header>
    <main class="page-body">
      <h1 class="page-heading">%s</h1>
      %s
    </main>
    <footer class="footer">
      <span>CasPaste %s</span>
      <a href="/docs">Documentation</a>
      <span>© casjay-forks</span>
    </footer>
  </div>
</div>
</body></html>`,
		template.HTMLEscapeString(pd.Title),
		template.HTMLEscapeString(serverTitle),
		template.HTMLEscapeString(serverTitle),
		nav,
		base,
		p.breadcrumb(pd.Page, base),
		template.HTMLEscapeString(pd.AdminUsername),
		base,
		template.HTMLEscapeString(pd.Title),
		content,
		template.HTMLEscapeString(pd.Version),
	)
}

// breadcrumb returns the breadcrumb trail HTML fragment for a given page
func (p *Panel) breadcrumb(page, base string) string {
	if page == "" || page == "dashboard" {
		return ""
	}
	parts := strings.Split(page, "/")
	var sb strings.Builder
	acc := base
	for _, part := range parts {
		if part == "" {
			continue
		}
		acc += "/" + part
		label := strings.Title(strings.ReplaceAll(part, "-", " "))
		sb.WriteString(fmt.Sprintf(`<span class="breadcrumb-sep">/</span><a href="%s">%s</a>`,
			acc, template.HTMLEscapeString(label)))
	}
	return sb.String()
}

// buildSidebarNav generates the sidebar navigation HTML
func (p *Panel) buildSidebarNav(activePage, base, username string) string {
	link := func(href, icon, label, key string) string {
		cls := "nav-link"
		if activePage == key {
			cls += " active"
		}
		return fmt.Sprintf(`<a class="%s" href="%s"><span class="nav-icon">%s</span>%s</a>`,
			cls, base+href, icon, template.HTMLEscapeString(label))
	}

	var sb strings.Builder

	// Dashboard
	sb.WriteString(`<div class="nav-section">`)
	sb.WriteString(link("/", "📊", "Dashboard", "dashboard"))
	sb.WriteString(`</div>`)

	// Server management
	sb.WriteString(`<div class="nav-section">`)
	sb.WriteString(`<div class="nav-section-title">Server</div>`)
	sb.WriteString(link("/config/settings", "⚙️", "Settings", "config/settings"))
	sb.WriteString(link("/config/ssl", "🔐", "SSL / TLS", "config/ssl"))
	sb.WriteString(link("/config/email", "📧", "Email", "config/email"))
	sb.WriteString(link("/config/scheduler", "🕐", "Scheduler", "config/scheduler"))
	sb.WriteString(link("/config/logs", "📋", "Logs", "config/logs"))
	sb.WriteString(link("/config/backup", "💾", "Backup", "config/backup"))
	sb.WriteString(link("/config/updates", "🔄", "Updates", "config/updates"))
	sb.WriteString(link("/config/info", "ℹ️", "Info", "config/info"))
	sb.WriteString(link("/config/metrics", "📈", "Metrics", "config/metrics"))
	sb.WriteString(`</div>`)

	// Network
	sb.WriteString(`<div class="nav-section">`)
	sb.WriteString(`<div class="nav-section-title">Network</div>`)
	sb.WriteString(link("/config/network/tor", "🧅", "Tor", "config/network/tor"))
	sb.WriteString(link("/config/network/geoip", "🌍", "GeoIP", "config/network/geoip"))
	sb.WriteString(`</div>`)

	// Security
	sb.WriteString(`<div class="nav-section">`)
	sb.WriteString(`<div class="nav-section-title">Security</div>`)
	sb.WriteString(link("/config/security/auth", "🔑", "Authentication", "config/security/auth"))
	sb.WriteString(link("/config/security/tokens", "🪙", "API Tokens", "config/security/tokens"))
	sb.WriteString(link("/config/security/firewall", "🛡️", "Firewall", "config/security/firewall"))
	sb.WriteString(`</div>`)

	// Account
	if username != "" {
		sb.WriteString(`<div class="nav-section">`)
		sb.WriteString(`<div class="nav-section-title">Account</div>`)
		sb.WriteString(link("/"+username+"/profile", "👤", "Profile", username+"/profile"))
		sb.WriteString(link("/"+username+"/preferences", "🎨", "Preferences", username+"/preferences"))
		sb.WriteString(link("/"+username+"/notifications", "🔔", "Notifications", username+"/notifications"))
		sb.WriteString(`</div>`)
	}

	return sb.String()
}

// kvRow returns a KV list row HTML
func kvRow(key, value string) template.HTML {
	return template.HTML(fmt.Sprintf(
		`<dt class="kv-key">%s</dt><dd class="kv-val">%s</dd>`,
		template.HTMLEscapeString(key),
		template.HTMLEscapeString(value),
	))
}

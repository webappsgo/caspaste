// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"net/http"
)

// handleUserDashboard handles GET /users (user dashboard)
func (data *Data) handleUserDashboard(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}

	// Get authenticated user from context
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderUserDashboard(rw, req, authUser)
}

// handleUserSettings handles GET/POST /users/settings - per AI.md PART 34
func (data *Data) handleUserSettings(rw http.ResponseWriter, req *http.Request) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderUserSettings(rw, req, authUser)
}

// handleUserNotifications handles GET /users/notifications - per AI.md PART 34
func (data *Data) handleUserNotifications(rw http.ResponseWriter, req *http.Request) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderUserNotifications(rw, req, authUser)
}

// handleUserSettingsPrivacy handles GET/POST /users/settings/privacy - per AI.md PART 34
func (data *Data) handleUserSettingsPrivacy(rw http.ResponseWriter, req *http.Request) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderUserSettingsPrivacy(rw, req, authUser)
}

// handleUserSettingsNotifications handles GET/POST /users/settings/notifications - per AI.md PART 34
func (data *Data) handleUserSettingsNotifications(rw http.ResponseWriter, req *http.Request) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderUserSettingsNotifications(rw, req, authUser)
}

// handleUserSettingsAppearance handles GET/POST /users/settings/appearance - per AI.md PART 34
func (data *Data) handleUserSettingsAppearance(rw http.ResponseWriter, req *http.Request) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderUserSettingsAppearance(rw, req, authUser)
}

// handleUserSecurity handles GET /users/security
func (data *Data) handleUserSecurity(rw http.ResponseWriter, req *http.Request) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderUserSecurity(rw, req, authUser)
}

// handleUserTokens handles GET/POST /users/tokens
func (data *Data) handleUserTokens(rw http.ResponseWriter, req *http.Request) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderUserTokens(rw, req, authUser)
}

// handleUserDomains handles GET /users/domains
func (data *Data) handleUserDomains(rw http.ResponseWriter, req *http.Request) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderUserDomains(rw, req, authUser)
}

// Render functions - these will use templates

func (data *Data) renderUserDashboard(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	// Get locale
	locale := data.Locales.findLocale(req)

	templateData := map[string]interface{}{
		"Title":          "Dashboard",
		"User":           user,
		"Version":        data.Version,
		"FQDN":           data.FQDN,
		"ServerTitle":    data.ServerTitle,
		"LocalesList":    data.LocalesList,
		"ThemesList":     data.ThemesList,
		"UiDefaultTheme": data.UiDefaultTheme,
		"Translate":      locale.translate,
	}

	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	// For now, use a simple HTML response until we have the full template
	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Dashboard - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Welcome, ` + user.Username + `!</h1>
		<nav>
			<ul>
				<li><a href="/users/settings">Settings</a></li>
				<li><a href="/users/security">Security</a></li>
				<li><a href="/users/tokens">API Tokens</a></li>
				<li><a href="/users/domains">Custom Domains</a></li>
				<li><a href="/orgs">Organizations</a></li>
				<li><a href="/server/auth/logout">Logout</a></li>
			</ul>
		</nav>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	_ = templateData // Will be used when full template is implemented
	return err
}

func (data *Data) renderUserSecurity(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Security Settings - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Security Settings</h1>
		<section>
			<h2>Two-Factor Authentication</h2>
			<p>Status: ` + boolToStr(user.TOTPEnabled, "Enabled", "Disabled") + `</p>
			<form action="/api/v1/users/security/2fa/enable" method="POST">
				<button type="submit">` + boolToStr(user.TOTPEnabled, "Manage 2FA", "Enable 2FA") + `</button>
			</form>
		</section>
		<section>
			<h2>Sessions</h2>
			<p><a href="/api/v1/users/sessions">View Active Sessions</a></p>
		</section>
		<section>
			<h2>Change Password</h2>
			<form action="/api/v1/users/security/password" method="POST">
				<input type="password" name="current_password" placeholder="Current Password" required>
				<input type="password" name="new_password" placeholder="New Password" required>
				<button type="submit">Change Password</button>
			</form>
		</section>
		<p><a href="/users">Back to Dashboard</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderUserTokens(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>API Tokens - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>API Tokens</h1>
		<section>
			<h2>Create New Token</h2>
			<form action="/api/v1/users/tokens" method="POST">
				<input type="text" name="name" placeholder="Token Name" required>
				<select name="scopes">
					<option value="read">Read Only</option>
					<option value="read-write">Read/Write</option>
					<option value="global">Full Access</option>
				</select>
				<button type="submit">Create Token</button>
			</form>
		</section>
		<section>
			<h2>Active Tokens</h2>
			<p>View and manage your API tokens via the API: GET /api/v1/users/tokens</p>
		</section>
		<p><a href="/users">Back to Dashboard</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderUserDomains(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Custom Domains - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Custom Domains</h1>
		<section>
			<h2>Add New Domain</h2>
			<form action="/api/v1/users/domains" method="POST">
				<input type="text" name="domain" placeholder="yourdomain.com" required>
				<button type="submit">Add Domain</button>
			</form>
		</section>
		<section>
			<h2>Your Domains</h2>
			<p>View and manage your domains via the API: GET /api/v1/users/domains</p>
		</section>
		<p><a href="/users">Back to Dashboard</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderUserSettings(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Account Settings - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Account Settings</h1>
		<nav class="settings-nav">
			<a href="/users/settings" class="active">Account</a>
			<a href="/users/settings/privacy">Privacy</a>
			<a href="/users/settings/notifications">Notifications</a>
			<a href="/users/settings/appearance">Appearance</a>
		</nav>
		<section>
			<h2>Profile Information</h2>
			<form action="/api/v1/users" method="PATCH">
				<div class="form-group">
					<label for="display_name">Display Name</label>
					<input type="text" id="display_name" name="display_name" value="` + user.Username + `">
				</div>
				<div class="form-group">
					<label for="email">Email</label>
					<input type="email" id="email" name="email" placeholder="your@email.com">
				</div>
				<div class="form-group">
					<label for="bio">Bio</label>
					<textarea id="bio" name="bio" rows="3"></textarea>
				</div>
				<button type="submit">Save Changes</button>
			</form>
		</section>
		<p><a href="/users">Back to Dashboard</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderUserNotifications(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Notifications - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Notifications</h1>
		<section>
			<div class="empty-state">
				<h3>No notifications</h3>
				<p>You're all caught up!</p>
			</div>
		</section>
		<p><a href="/users">Back to Dashboard</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderUserSettingsPrivacy(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Privacy Settings - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Privacy Settings</h1>
		<nav class="settings-nav">
			<a href="/users/settings">Account</a>
			<a href="/users/settings/privacy" class="active">Privacy</a>
			<a href="/users/settings/notifications">Notifications</a>
			<a href="/users/settings/appearance">Appearance</a>
		</nav>
		<section>
			<form action="/api/v1/users/settings" method="PATCH">
				<div class="form-group">
					<label>
						<input type="checkbox" name="show_email">
						Show email on profile
					</label>
				</div>
				<div class="form-group">
					<label>
						<input type="checkbox" name="show_activity" checked>
						Show activity on profile
					</label>
				</div>
				<div class="form-group">
					<label>
						<input type="checkbox" name="show_orgs" checked>
						Show organizations on profile
					</label>
				</div>
				<div class="form-group">
					<label>
						<input type="checkbox" name="searchable" checked>
						Allow profile to be found in search
					</label>
				</div>
				<button type="submit">Save Privacy Settings</button>
			</form>
		</section>
		<p><a href="/users">Back to Dashboard</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderUserSettingsNotifications(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Notification Settings - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Notification Settings</h1>
		<nav class="settings-nav">
			<a href="/users/settings">Account</a>
			<a href="/users/settings/privacy">Privacy</a>
			<a href="/users/settings/notifications" class="active">Notifications</a>
			<a href="/users/settings/appearance">Appearance</a>
		</nav>
		<section>
			<h2>Email Notifications</h2>
			<form action="/api/v1/users/settings" method="PATCH">
				<div class="form-group">
					<label>
						<input type="checkbox" name="email_security" checked>
						Security alerts
					</label>
				</div>
				<div class="form-group">
					<label>
						<input type="checkbox" name="email_mentions" checked>
						Mentions and replies
					</label>
				</div>
				<div class="form-group">
					<label>
						<input type="checkbox" name="email_updates">
						Product updates
					</label>
				</div>
				<div class="form-group">
					<label for="email_digest">Email digest frequency</label>
					<select id="email_digest" name="email_digest">
						<option value="daily">Daily</option>
						<option value="weekly" selected>Weekly</option>
						<option value="never">Never</option>
					</select>
				</div>
				<button type="submit">Save Notification Settings</button>
			</form>
		</section>
		<p><a href="/users">Back to Dashboard</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderUserSettingsAppearance(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Appearance Settings - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Appearance Settings</h1>
		<nav class="settings-nav">
			<a href="/users/settings">Account</a>
			<a href="/users/settings/privacy">Privacy</a>
			<a href="/users/settings/notifications">Notifications</a>
			<a href="/users/settings/appearance" class="active">Appearance</a>
		</nav>
		<section>
			<form action="/api/v1/users/settings" method="PATCH">
				<div class="form-group">
					<label for="theme">Theme</label>
					<select id="theme" name="theme">
						<option value="dark">Dark</option>
						<option value="light">Light</option>
						<option value="system">System</option>
					</select>
				</div>
				<div class="form-group">
					<label for="font_size">Font Size</label>
					<select id="font_size" name="font_size">
						<option value="small">Small</option>
						<option value="medium" selected>Medium</option>
						<option value="large">Large</option>
					</select>
				</div>
				<div class="form-group">
					<label>
						<input type="checkbox" name="reduce_motion">
						Reduce motion
					</label>
				</div>
				<button type="submit">Save Appearance Settings</button>
			</form>
		</section>
		<p><a href="/users">Back to Dashboard</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

// Helper function
func boolToStr(b bool, trueStr, falseStr string) string {
	if b {
		return trueStr
	}
	return falseStr
}

// ErrMethodNotAllowed is returned for unsupported HTTP methods
var ErrMethodNotAllowed = &httpError{code: 405, message: "Method not allowed"}

type httpError struct {
	code    int
	message string
}

func (e *httpError) Error() string {
	return e.message
}

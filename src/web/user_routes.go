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

// userSettingsNavLinks returns the standard settings navigation links.
func userSettingsNavLinks() []stubLink {
	return []stubLink{
		{URL: "/users/settings", Label: "Account"},
		{URL: "/users/settings/privacy", Label: "Privacy"},
		{URL: "/users/settings/notifications", Label: "Notifications"},
		{URL: "/users/settings/appearance", Label: "Appearance"},
	}
}

func (data *Data) renderUserDashboard(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Welcome, " + user.Username,
		Description: "Your personal dashboard.",
		Links: []stubLink{
			{URL: "/users/settings", Label: "Settings"},
			{URL: "/users/security", Label: "Security"},
			{URL: "/users/tokens", Label: "API Tokens"},
			{URL: "/users/domains", Label: "Custom Domains"},
			{URL: "/orgs", Label: "Organizations"},
			{URL: "/server/auth/logout", Label: "Logout"},
		},
	})
}

func (data *Data) renderUserSecurity(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	twoFALabel := "Enable 2FA"
	if user.TOTPEnabled {
		twoFALabel = "Manage 2FA"
	}
	return data.renderStub(rw, req, stubTmplData{
		Title: "Security Settings",
		Links: []stubLink{
			{URL: "/api/v1/users/sessions", Label: "View Active Sessions"},
		},
		FormAction:  "/api/v1/users/security/password",
		FormMethod:  "POST",
		SubmitLabel: "Change Password",
		Fields: []stubField{
			{ID: "current_password", Name: "current_password", Label: "Current Password",
				Type: "password", Placeholder: "Current Password", Required: true},
			{ID: "new_password", Name: "new_password", Label: "New Password",
				Type: "password", Placeholder: "New Password", Required: true},
		},
		Notice:    "Two-Factor Authentication: " + boolToStr(user.TOTPEnabled, "Enabled", "Disabled") +
			" — <a href=\"/api/v1/users/security/2fa/enable\">" + twoFALabel + "</a>",
		BackURL:   "/users",
		BackLabel: "Back to Dashboard",
	})
}

func (data *Data) renderUserTokens(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "API Tokens",
		Description: "View and manage your API tokens via the API: GET /api/v1/users/tokens",
		FormAction:  "/api/v1/users/tokens",
		FormMethod:  "POST",
		SubmitLabel: "Create Token",
		Fields: []stubField{
			{ID: "name", Name: "name", Label: "Token Name", Type: "text",
				Placeholder: "Token Name", Required: true},
			{ID: "scopes", Name: "scopes", Label: "Scopes", Type: "select",
				Options: []stubSelectOption{
					{Value: "read", Label: "Read Only"},
					{Value: "read-write", Label: "Read/Write"},
					{Value: "global", Label: "Full Access"},
				}},
		},
		BackURL:   "/users",
		BackLabel: "Back to Dashboard",
	})
}

func (data *Data) renderUserDomains(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Custom Domains",
		Description: "View and manage your domains via the API: GET /api/v1/users/domains",
		FormAction:  "/api/v1/users/domains",
		FormMethod:  "POST",
		SubmitLabel: "Add Domain",
		Fields: []stubField{
			{ID: "domain", Name: "domain", Label: "Domain", Type: "text",
				Placeholder: "yourdomain.com", Required: true},
		},
		BackURL:   "/users",
		BackLabel: "Back to Dashboard",
	})
}

func (data *Data) renderUserSettings(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Account Settings",
		Links:       userSettingsNavLinks(),
		FormAction:  "/api/v1/users",
		FormMethod:  "PATCH",
		SubmitLabel: "Save Changes",
		Fields: []stubField{
			{ID: "display_name", Name: "display_name", Label: "Display Name",
				Type: "text", Value: user.Username},
			{ID: "email", Name: "email", Label: "Email",
				Type: "email", Placeholder: "your@email.com"},
			{ID: "bio", Name: "bio", Label: "Bio", Type: "textarea"},
		},
		BackURL:   "/users",
		BackLabel: "Back to Dashboard",
	})
}

func (data *Data) renderUserNotifications(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Notifications",
		Description: "You are all caught up — no notifications.",
		BackURL:     "/users",
		BackLabel:   "Back to Dashboard",
	})
}

func (data *Data) renderUserSettingsPrivacy(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Privacy Settings",
		Links:       userSettingsNavLinks(),
		FormAction:  "/api/v1/users/settings",
		FormMethod:  "PATCH",
		SubmitLabel: "Save Privacy Settings",
		Fields: []stubField{
			{ID: "show_email", Name: "show_email", Label: "Show email on profile", Type: "text"},
			{ID: "searchable", Name: "searchable", Label: "Allow profile to be found in search", Type: "text"},
		},
		BackURL:   "/users",
		BackLabel: "Back to Dashboard",
	})
}

func (data *Data) renderUserSettingsNotifications(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Notification Settings",
		Links:       userSettingsNavLinks(),
		FormAction:  "/api/v1/users/settings",
		FormMethod:  "PATCH",
		SubmitLabel: "Save Notification Settings",
		Fields: []stubField{
			{ID: "email_digest", Name: "email_digest", Label: "Email digest frequency", Type: "select",
				Options: []stubSelectOption{
					{Value: "daily", Label: "Daily"},
					{Value: "weekly", Label: "Weekly"},
					{Value: "never", Label: "Never"},
				}},
		},
		BackURL:   "/users",
		BackLabel: "Back to Dashboard",
	})
}

func (data *Data) renderUserSettingsAppearance(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Appearance Settings",
		Links:       userSettingsNavLinks(),
		FormAction:  "/api/v1/users/settings",
		FormMethod:  "PATCH",
		SubmitLabel: "Save Appearance Settings",
		Fields: []stubField{
			{ID: "theme", Name: "theme", Label: "Theme", Type: "select",
				Options: []stubSelectOption{
					{Value: "dark", Label: "Dark"},
					{Value: "light", Label: "Light"},
					{Value: "system", Label: "System"},
				}},
			{ID: "font_size", Name: "font_size", Label: "Font Size", Type: "select",
				Options: []stubSelectOption{
					{Value: "small", Label: "Small"},
					{Value: "medium", Label: "Medium"},
					{Value: "large", Label: "Large"},
				}},
		},
		BackURL:   "/users",
		BackLabel: "Back to Dashboard",
	})
}

// boolToStr returns trueStr if b is true, falseStr otherwise.
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

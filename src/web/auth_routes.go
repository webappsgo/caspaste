// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"net/http"
	"strings"
)

// handleRegisterPage handles GET /auth/register
func (data *Data) handleRegisterPage(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Register",
		FormAction:  "/api/v1/auth/register",
		FormMethod:  "POST",
		SubmitLabel: "Register",
		Fields: []stubField{
			{ID: "username", Name: "username", Label: "Username", Type: "text",
				Required: true, MinLength: 3, MaxLength: 32, Pattern: "[a-z0-9_-]+",
				Hint: "3-32 characters, lowercase letters, numbers, underscores, hyphens"},
			{ID: "email", Name: "email", Label: "Email", Type: "email", Required: true},
			{ID: "password", Name: "password", Label: "Password", Type: "password",
				Required: true, MinLength: 8, Hint: "Minimum 8 characters"},
			{ID: "display_name", Name: "display_name", Label: "Display Name (optional)", Type: "text"},
		},
		Notice:    "Already have an account? <a href=\"/server/auth/login\">Login</a>",
		BackURL:   "/server/auth/login",
		BackLabel: "Back to Login",
	})
}

// handlePasswordForgotPage handles GET /auth/password/forgot
func (data *Data) handlePasswordForgotPage(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Forgot Password",
		Description: "Enter your email address to receive a password reset link.",
		FormAction:  "/api/v1/auth/password/forgot",
		FormMethod:  "POST",
		SubmitLabel: "Send Reset Link",
		Fields: []stubField{
			{ID: "email", Name: "email", Label: "Email", Type: "email", Required: true},
		},
		BackURL:   "/server/auth/login",
		BackLabel: "Back to Login",
	})
}

// handlePasswordResetPage handles GET /auth/password/reset/{token}
func (data *Data) handlePasswordResetPage(rw http.ResponseWriter, req *http.Request, token string) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Reset Password",
		FormAction:  "/api/v1/auth/password/reset",
		FormMethod:  "POST",
		SubmitLabel: "Reset Password",
		HiddenFields: []stubHiddenField{
			{Name: "token", Value: token},
		},
		Fields: []stubField{
			{ID: "new_password", Name: "new_password", Label: "New Password", Type: "password",
				Required: true, MinLength: 8},
			{ID: "password_confirm", Name: "password_confirm", Label: "Confirm Password",
				Type: "password", Required: true, MinLength: 8},
		},
	})
}

// handle2FAPage handles GET /auth/2fa
func (data *Data) handle2FAPage(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Two-Factor Authentication",
		FormAction:  "/api/v1/auth/login",
		FormMethod:  "POST",
		SubmitLabel: "Verify",
		Fields: []stubField{
			{ID: "totp_code", Name: "totp_code", Label: "Enter your 2FA code",
				Type: "text", Pattern: "[0-9]{6}", Required: true, Autocomplete: "one-time-code"},
		},
		Notice:    "<a href=\"/server/auth/recovery/use\">Use a recovery key instead</a>",
		BackURL:   "/server/auth/login",
		BackLabel: "Back to Login",
	})
}

// handleRecoveryPage handles GET /auth/recovery/use
func (data *Data) handleRecoveryPage(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}
	return data.renderStub(rw, req, stubTmplData{
		Title: "Use Recovery Key",
		Description: "If you have lost access to your authenticator app, " +
			"you can use a recovery key to regain access to your account.",
		FormAction:  "/api/v1/auth/recovery/use",
		FormMethod:  "POST",
		SubmitLabel: "Use Recovery Key",
		Fields: []stubField{
			{ID: "identifier", Name: "identifier", Label: "Username or Email",
				Type: "text", Required: true},
			{ID: "recovery_key", Name: "recovery_key", Label: "Recovery Key",
				Type: "text", Pattern: "[a-f0-9]{8}-[a-f0-9]{4}", Required: true,
				Placeholder: "xxxxxxxx-xxxx"},
		},
		Notice:    "Using a recovery key will disable 2FA on your account. You can re-enable it after logging in.",
		BackURL:   "/server/auth/login",
		BackLabel: "Back to Login",
	})
}

// handleVerifyEmailPage handles GET /auth/verify-email/{token}
// Redirects automatically to the API verify endpoint after a brief delay.
func (data *Data) handleVerifyEmailPage(rw http.ResponseWriter, req *http.Request, token string) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:         "Verifying Email...",
		Description:   "Please wait while we verify your email address.",
		RedirectURL:   "/api/v1/auth/verify-email?token=" + token,
		RedirectDelay: 3,
		Notice:        "If you are not redirected automatically, <a href=\"/api/v1/auth/verify-email?token=" + token + "\">click here</a>.",
	})
}

// handleInvitePage handles GET /auth/invite/{token}
func (data *Data) handleInvitePage(rw http.ResponseWriter, req *http.Request, token string) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Accept Invitation",
		Description: "You have been invited to join " + data.ServerTitle + "!",
		FormAction:  "/api/v1/auth/register",
		FormMethod:  "POST",
		SubmitLabel: "Create Account",
		HiddenFields: []stubHiddenField{
			{Name: "invite_code", Value: token},
		},
		Fields: []stubField{
			{ID: "username", Name: "username", Label: "Username", Type: "text",
				Required: true, MinLength: 3, MaxLength: 32, Pattern: "[a-z0-9_-]+"},
			{ID: "email", Name: "email", Label: "Email", Type: "email", Required: true},
			{ID: "password", Name: "password", Label: "Password", Type: "password",
				Required: true, MinLength: 8},
		},
	})
}

// routeAuth routes /auth/* paths
func (data *Data) routeAuth(rw http.ResponseWriter, req *http.Request) error {
	path := req.URL.Path

	switch {
	case path == "/auth/register":
		return data.handleRegisterPage(rw, req)

	case path == "/auth/password/forgot":
		return data.handlePasswordForgotPage(rw, req)

	case strings.HasPrefix(path, "/auth/password/reset/"):
		token := strings.TrimPrefix(path, "/auth/password/reset/")
		return data.handlePasswordResetPage(rw, req, token)

	case path == "/auth/2fa":
		return data.handle2FAPage(rw, req)

	case path == "/auth/recovery/use":
		return data.handleRecoveryPage(rw, req)

	case strings.HasPrefix(path, "/auth/verify-email/"):
		token := strings.TrimPrefix(path, "/auth/verify-email/")
		return data.handleVerifyEmailPage(rw, req, token)

	case strings.HasPrefix(path, "/auth/invite/"):
		token := strings.TrimPrefix(path, "/auth/invite/")
		return data.handleInvitePage(rw, req, token)

	default:
		http.Redirect(rw, req, "/server/auth/login", http.StatusFound)
		return nil
	}
}

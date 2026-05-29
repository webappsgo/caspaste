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

	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Register - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Register</h1>
		<form action="/api/v1/auth/register" method="POST">
			<div>
				<label for="username">Username:</label>
				<input type="text" id="username" name="username" required pattern="[a-z0-9_-]+" minlength="3" maxlength="32">
				<small>3-32 characters, lowercase letters, numbers, underscores, hyphens</small>
			</div>
			<div>
				<label for="email">Email:</label>
				<input type="email" id="email" name="email" required>
			</div>
			<div>
				<label for="password">Password:</label>
				<input type="password" id="password" name="password" required minlength="8">
				<small>Minimum 8 characters</small>
			</div>
			<div>
				<label for="display_name">Display Name (optional):</label>
				<input type="text" id="display_name" name="display_name">
			</div>
			<button type="submit">Register</button>
		</form>
		<p>Already have an account? <a href="/server/auth/login">Login</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

// handlePasswordForgotPage handles GET /auth/password/forgot
func (data *Data) handlePasswordForgotPage(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}

	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Forgot Password - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Forgot Password</h1>
		<form action="/api/v1/auth/password/forgot" method="POST">
			<div>
				<label for="email">Email:</label>
				<input type="email" id="email" name="email" required>
			</div>
			<button type="submit">Send Reset Link</button>
		</form>
		<p><a href="/server/auth/login">Back to Login</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

// handlePasswordResetPage handles GET /auth/password/reset/{token}
func (data *Data) handlePasswordResetPage(rw http.ResponseWriter, req *http.Request, token string) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}

	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Reset Password - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Reset Password</h1>
		<form action="/api/v1/auth/password/reset" method="POST">
			<input type="hidden" name="token" value="` + token + `">
			<div>
				<label for="password">New Password:</label>
				<input type="password" id="password" name="new_password" required minlength="8">
			</div>
			<div>
				<label for="password_confirm">Confirm Password:</label>
				<input type="password" id="password_confirm" required minlength="8">
			</div>
			<button type="submit">Reset Password</button>
		</form>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

// handle2FAPage handles GET /auth/2fa
func (data *Data) handle2FAPage(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}

	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Two-Factor Authentication - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Two-Factor Authentication</h1>
		<form action="/api/v1/auth/login" method="POST">
			<div>
				<label for="totp_code">Enter your 2FA code:</label>
				<input type="text" id="totp_code" name="totp_code" pattern="[0-9]{6}" required autocomplete="one-time-code">
			</div>
			<button type="submit">Verify</button>
		</form>
		<p><a href="/server/auth/recovery/use">Use a recovery key instead</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

// handleRecoveryPage handles GET /auth/recovery/use
func (data *Data) handleRecoveryPage(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}

	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Recovery Key - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Use Recovery Key</h1>
		<p>If you've lost access to your authenticator app, you can use a recovery key to regain access to your account.</p>
		<form action="/api/v1/auth/recovery/use" method="POST">
			<div>
				<label for="identifier">Username or Email:</label>
				<input type="text" id="identifier" name="identifier" required>
			</div>
			<div>
				<label for="recovery_key">Recovery Key:</label>
				<input type="text" id="recovery_key" name="recovery_key" pattern="[a-f0-9]{8}-[a-f0-9]{4}" required placeholder="xxxxxxxx-xxxx">
			</div>
			<button type="submit">Use Recovery Key</button>
		</form>
		<p><strong>Note:</strong> Using a recovery key will disable 2FA on your account. You can re-enable it after logging in.</p>
		<p><a href="/server/auth/login">Back to Login</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

// handleVerifyEmailPage handles GET /auth/verify-email/{token}
func (data *Data) handleVerifyEmailPage(rw http.ResponseWriter, req *http.Request, token string) error {
	// This page will automatically trigger verification via JavaScript
	// or redirect to the API endpoint

	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Verify Email - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
	<meta http-equiv="refresh" content="3;url=/api/v1/auth/verify-email?token=` + token + `">
</head>
<body>
	<div class="container">
		<h1>Verifying Email...</h1>
		<p>Please wait while we verify your email address.</p>
		<p>If you are not redirected automatically, <a href="/api/v1/auth/verify-email?token=` + token + `">click here</a>.</p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

// handleInvitePage handles GET /auth/invite/{token}
func (data *Data) handleInvitePage(rw http.ResponseWriter, req *http.Request, token string) error {
	if req.Method != http.MethodGet {
		return ErrMethodNotAllowed
	}

	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Accept Invitation - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Accept Invitation</h1>
		<p>You've been invited to join ` + data.ServerTitle + `!</p>
		<form action="/api/v1/auth/register" method="POST">
			<input type="hidden" name="invite_code" value="` + token + `">
			<div>
				<label for="username">Username:</label>
				<input type="text" id="username" name="username" required pattern="[a-z0-9_-]+" minlength="3" maxlength="32">
			</div>
			<div>
				<label for="email">Email:</label>
				<input type="email" id="email" name="email" required>
			</div>
			<div>
				<label for="password">Password:</label>
				<input type="password" id="password" name="password" required minlength="8">
			</div>
			<button type="submit">Create Account</button>
		</form>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
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
		// Redirect unknown auth paths to login
		http.Redirect(rw, req, "/server/auth/login", http.StatusFound)
		return nil
	}
}

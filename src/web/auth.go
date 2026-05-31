// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/casjay-forks/caspaste/src/caspasswd"
	"github.com/casjay-forks/caspaste/src/netshare"
)

// Session cookie name and settings
const (
	sessionCookieName = "caspaste_session"
	sessionDuration   = 24 * time.Hour
)

// sessionSecret is used to sign session cookies
var sessionSecret []byte

func init() {
	// Generate a random session secret on startup
	sessionSecret = make([]byte, 32)
	if _, err := rand.Read(sessionSecret); err != nil {
		panic("failed to generate session secret: " + err.Error())
	}
}

type loginTmpl struct {
	User     *AuthUser
	Language string
	Theme    func(string) string
	Error    bool
	Redirect string
	// CSRF token for form protection per AI.md PART 11
	CSRFToken     string
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool
	ShowRegister  bool

	Translate func(string, ...interface{}) template.HTML
}

// generateSessionToken creates a signed session token
func generateSessionToken(username string) string {
	// Token format: username:timestamp:signature
	timestamp := time.Now().Unix()
	data := []byte(username + ":" + strconv.FormatInt(timestamp, 10))

	h := hmac.New(sha256.New, sessionSecret)
	h.Write(data)
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	token := base64.URLEncoding.EncodeToString([]byte(username)) + "." +
		base64.URLEncoding.EncodeToString([]byte(time.Now().Format(time.RFC3339))) + "." +
		signature

	return token
}

// validateSessionToken validates a session token and returns the username if valid
func validateSessionToken(token string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", false
	}

	usernameBytes, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	username := string(usernameBytes)

	timestampBytes, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}

	timestamp, err := time.Parse(time.RFC3339, string(timestampBytes))
	if err != nil {
		return "", false
	}

	// Check if session has expired
	if time.Since(timestamp) > sessionDuration {
		return "", false
	}

	// Verify signature
	data := []byte(username + ":" + strconv.FormatInt(timestamp.Unix(), 10))
	h := hmac.New(sha256.New, sessionSecret)
	h.Write(data)
	expectedSig := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if parts[2] != expectedSig {
		return "", false
	}

	return username, true
}

// isAuthenticated checks if the user has a valid session
func (data *Data) isAuthenticated(req *http.Request) bool {
	cookie, err := req.Cookie(sessionCookieName)
	if err != nil {
		return false
	}

	_, valid := validateSessionToken(cookie.Value)
	return valid
}

// setSessionCookie sets a session cookie for authenticated users
func setSessionCookie(rw http.ResponseWriter, req *http.Request, username string) {
	token := generateSessionToken(username)

	// Auto-detect HTTPS from request TLS or X-Forwarded-Proto header
	secure := req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https"

	http.SetCookie(rw, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie removes the session cookie
func clearSessionCookie(rw http.ResponseWriter) {
	http.SetCookie(rw, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// Pattern: GET /login
func (data *Data) handleLoginPage(rw http.ResponseWriter, req *http.Request) error {
	// If already authenticated, redirect to home
	if data.isAuthenticated(req) {
		writeRedirect(rw, req, "/", 302)
		return nil
	}

	redirect := req.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	tmplData := loginTmpl{
		User:          GetAuthUser(req.Context()),
		Language:      getCookie(req, "lang"),
		Theme:         data.getThemeFunc(req),
		Error:         false,
		Redirect:      redirect,
		CSRFToken:     GetCSRFToken(req, 32),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		ShowRegister:  data.ShowRegister,
		Translate:     data.Locales.findLocale(req).translate,
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.Login.Execute(rw, tmplData)
}

// Pattern: POST /login
func (data *Data) handleLoginSubmit(rw http.ResponseWriter, req *http.Request) error {
	if err := req.ParseForm(); err != nil {
		return data.handleLoginError(rw, req, "/")
	}

	username := req.FormValue("username")
	password := req.FormValue("password")
	redirect := req.FormValue("redirect")

	if redirect == "" {
		redirect = "/"
	}

	// Get client IP for brute force protection
	clientIP := netshare.GetClientAddr(req)

	// Check if IP is blocked due to too many failed attempts
	// Per AI.md PART 11: 5 failed attempts = 15-minute lockout
	if data.BruteForce != nil && data.BruteForce.CheckBlocked(clientIP) {
		remaining := data.BruteForce.GetRemainingLockout(clientIP)
		rw.Header().Set("Retry-After", strconv.Itoa(int(remaining.Seconds())))
		return data.handleLoginError(rw, req, redirect)
	}

	// Validate credentials
	if data.CasPasswdFile == "" {
		// No password file configured, reject login
		if data.BruteForce != nil {
			data.BruteForce.RecordFailure(clientIP)
		}
		return data.handleLoginError(rw, req, redirect)
	}

	// Check if password file exists
	if _, err := os.Stat(data.CasPasswdFile); err != nil {
		if data.BruteForce != nil {
			data.BruteForce.RecordFailure(clientIP)
		}
		return data.handleLoginError(rw, req, redirect)
	}

	isValid, needsRehash, err := caspasswd.LoadAndCheckWithRehash(data.CasPasswdFile, username, password)
	if err != nil || !isValid {
		// Record failed login attempt
		if data.BruteForce != nil {
			data.BruteForce.RecordFailure(clientIP)
		}
		return data.handleLoginError(rw, req, redirect)
	}

	// Clear failed attempts on successful login
	if data.BruteForce != nil {
		data.BruteForce.RecordSuccess(clientIP)
	}

	// Rehash legacy passwords (bcrypt/plain text) to Argon2id
	// Per AI.md PART 11: "Verify existing passwords, then rehash with Argon2id"
	if needsRehash {
		// Rehash in background - don't block login on rehash failure
		go func() {
			_ = caspasswd.RehashPassword(data.CasPasswdFile, username, password)
		}()
	}

	// Set session cookie and redirect
	setSessionCookie(rw, req, username)
	writeRedirect(rw, req, redirect, 302)
	return nil
}

// handleLoginError shows the login page with error
func (data *Data) handleLoginError(rw http.ResponseWriter, req *http.Request, redirect string) error {
	tmplData := loginTmpl{
		User:          GetAuthUser(req.Context()),
		Language:      getCookie(req, "lang"),
		Theme:         data.getThemeFunc(req),
		Error:         true,
		Redirect:      redirect,
		CSRFToken:     GetCSRFToken(req, 32),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		ShowRegister:  data.ShowRegister,
		Translate:     data.Locales.findLocale(req).translate,
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(401)
	return data.Login.Execute(rw, tmplData)
}

// Pattern: GET /logout
func (data *Data) handleLogout(rw http.ResponseWriter, req *http.Request) error {
	clearSessionCookie(rw)
	writeRedirect(rw, req, "/", 302)
	return nil
}

// IsAuthRequired returns true if authentication is required (server.public=false)
func (data *Data) IsAuthRequired() bool {
	// Public instance = no auth required
	if data.Public {
		return false
	}
	// Private instance = auth required, password file must exist
	if data.CasPasswdFile == "" {
		return false
	}
	if _, err := os.Stat(data.CasPasswdFile); err != nil {
		return false
	}
	return true
}

// requireAuth checks if authentication is required and user is authenticated
// Returns true if the request should continue, false if it was redirected to login
func (data *Data) requireAuth(rw http.ResponseWriter, req *http.Request) bool {
	// No auth required for public instances
	if !data.IsAuthRequired() {
		return true
	}

	// Check if user has valid session
	if data.isAuthenticated(req) {
		return true
	}

	// Redirect to login page
	redirect := req.URL.Path
	if req.URL.RawQuery != "" {
		redirect += "?" + req.URL.RawQuery
	}
	writeRedirect(rw, req, "/server/auth/login?redirect="+redirect, 302)
	return false
}

// IsPublicPath returns true if the path should be accessible without authentication
func IsPublicPath(path string) bool {
	// Exact match paths per AI.md PART 13, PART 14
	publicPaths := []string{
		"/login",
		"/logout",
		"/server/auth/login",
		"/server/auth/logout",
		"/server/auth/register",
		"/healthz",
		"/style.css",
		"/main.js",
		"/toast.js",
		"/nav.js",
		"/history.js",
		"/code.js",
		"/paste.js",
		"/manifest.json",
		"/sw.js",
		"/robots.txt",
		"/sitemap.xml",
		"/favicon.ico",
		"/.well-known/security.txt",
		"/openapi",
		"/openapi.json",
	}

	for _, p := range publicPaths {
		if path == p {
			return true
		}
	}

	// Prefix match paths - public info and auth pages
	publicPrefixes := []string{
		// /server/auth/login, /server/auth/register, /server/auth/password/*, etc.
		"/server/auth",
		// Legacy /auth/* prefix
		"/auth",
		// /server/about, /server/about/authors, /server/about/license
		"/server/about",
		// Legacy redirect support
		"/about",
		// /docs, /docs/apiv1, /docs/libraries, /docs/customize
		"/docs",
		// /terms
		"/terms",
	}

	for _, prefix := range publicPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}

	return false
}

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/casjay-forks/caspaste/src/audit"
	"github.com/casjay-forks/caspaste/src/netshare"
)

// csrfContextKey is the request context key used to pass the CSRF token
// from CSRFMiddleware to GetCSRFToken without server-side storage.
type csrfContextKey struct{}

// CSRFConfig holds CSRF protection configuration
type CSRFConfig struct {
	// Enable CSRF protection
	Enabled bool
	// Token length in bytes
	TokenLength int
	// Cookie name for CSRF token
	CookieName string
	// Header name for CSRF token
	HeaderName string
	// Form field name for CSRF token
	FieldName string
	// Secure cookie mode: "auto", "true", "false"
	Secure string
	// ExemptPaths are paths that skip CSRF validation (API endpoints, etc.)
	ExemptPaths []string
	// ExemptPrefixes are path prefixes that skip CSRF validation
	ExemptPrefixes []string
}

// generateCSRFToken generates a cryptographically secure random token
func generateCSRFToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CSRFMiddleware implements the stateless double-submit cookie pattern.
//
// On safe methods (GET/HEAD/OPTIONS): the middleware reuses the CSRF cookie
// when present or generates a fresh token, sets/refreshes the cookie, and
// injects the token into the request context so templates can include it in
// hidden form fields — no server-side storage required.
//
// On state-changing methods (POST/PUT/DELETE/PATCH): the middleware reads the
// cookie value and the submitted token (header or form field) and rejects the
// request if they do not match via constant-time comparison.
//
// This pattern survives server restarts and scales horizontally because the
// CSRF secret lives only in the browser cookie.
func CSRFMiddleware(config CSRFConfig) func(http.Handler) http.Handler {
	if !config.Enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	// Apply defaults
	if config.TokenLength == 0 {
		config.TokenLength = 32
	}
	if config.CookieName == "" {
		config.CookieName = "csrf_token"
	}
	if config.HeaderName == "" {
		config.HeaderName = "X-CSRF-Token"
	}
	if config.FieldName == "" {
		config.FieldName = "csrf_token"
	}
	if config.Secure == "" {
		config.Secure = "auto"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exempt API and compatibility paths from CSRF
			if isCSRFExempt(r.URL.Path, config.ExemptPaths, config.ExemptPrefixes) {
				next.ServeHTTP(w, r)
				return
			}

			if isSafeMethod(r.Method) {
				// Reuse existing cookie token or generate a fresh one.
				token := ""
				if cookie, err := r.Cookie(config.CookieName); err == nil && cookie.Value != "" {
					token = cookie.Value
				} else {
					var genErr error
					token, genErr = generateCSRFToken(config.TokenLength)
					if genErr != nil {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
				}

				// Refresh the cookie on every safe request to extend its lifetime.
				setCSRFCookie(w, r, config, token)

				// Expose via response header for JavaScript clients.
				w.Header().Set(config.HeaderName, token)

				// Inject token into request context so GetCSRFToken can return it
				// from within template/handler execution without re-reading the cookie.
				ctx := context.WithValue(r.Context(), csrfContextKey{}, token)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// State-changing method: validate the double-submit.
			cookieToken := ""
			if cookie, err := r.Cookie(config.CookieName); err == nil {
				cookieToken = cookie.Value
			}

			if cookieToken == "" {
				// Log CSRF failure to audit log per AI.md PART 11
				audit.CSRFFailure(netshare.GetClientAddr(r).String(), r.URL.Path, GetRequestID(r.Context()))
				http.Error(w, "CSRF token missing", http.StatusForbidden)
				return
			}

			requestToken := extractCSRFToken(r, config)
			if requestToken == "" {
				// Log CSRF failure to audit log per AI.md PART 11
				audit.CSRFFailure(netshare.GetClientAddr(r).String(), r.URL.Path, GetRequestID(r.Context()))
				http.Error(w, "CSRF token missing", http.StatusForbidden)
				return
			}

			// Constant-time comparison prevents timing attacks
			if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(requestToken)) != 1 {
				// Log CSRF failure to audit log per AI.md PART 11
				audit.CSRFFailure(netshare.GetClientAddr(r).String(), r.URL.Path, GetRequestID(r.Context()))
				http.Error(w, "CSRF token invalid", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isSafeMethod returns true for HTTP methods that don't change server state
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

// isCSRFExempt checks if a path should be exempt from CSRF validation.
// API endpoints and compatibility endpoints are exempt since they use tokens, not cookies.
func isCSRFExempt(path string, exemptPaths, exemptPrefixes []string) bool {
	for _, exempt := range exemptPaths {
		if path == exempt {
			return true
		}
	}
	for _, prefix := range exemptPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// extractCSRFToken extracts the CSRF token from the request.
// Checks: X-CSRF-Token header, X-XSRF-Token header (Angular), then form field.
func extractCSRFToken(r *http.Request, config CSRFConfig) string {
	// Check header first (preferred for AJAX / XHR requests)
	if token := r.Header.Get(config.HeaderName); token != "" {
		return token
	}

	// X-XSRF-Token for Angular compatibility
	if token := r.Header.Get("X-XSRF-Token"); token != "" {
		return token
	}

	// ParseMultipartForm handles both multipart/form-data and
	// application/x-www-form-urlencoded; skip if already parsed.
	if r.MultipartForm == nil || r.Form == nil {
		_ = r.ParseMultipartForm(32 << 20)
	}

	// Form field fallback
	if token := r.FormValue(config.FieldName); token != "" {
		return token
	}

	return ""
}

// setCSRFCookie sets the CSRF token cookie.
// HttpOnly is false so JavaScript can read the token for XHR requests.
func setCSRFCookie(w http.ResponseWriter, r *http.Request, config CSRFConfig, token string) {
	secure := false
	switch config.Secure {
	case "true":
		secure = true
	case "false":
		secure = false
	default:
		// auto: detect from TLS or X-Forwarded-Proto
		secure = r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     config.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
}

// GetCSRFToken returns the CSRF token for the request.
// Primary source is the request context (set by CSRFMiddleware on GET requests).
// Falls back to reading the cookie directly for requests that bypass middleware.
func GetCSRFToken(r *http.Request, tokenLength int) string {
	// Context is set by CSRFMiddleware when it processes a safe-method request
	if token, ok := r.Context().Value(csrfContextKey{}).(string); ok && token != "" {
		return token
	}

	// Fallback: read the CSRF cookie directly (e.g., for exempt paths)
	if cookie, err := r.Cookie("csrf_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Last resort: generate an ephemeral token (middleware did not run)
	token, _ := generateCSRFToken(tokenLength)
	return token
}

// RegenerateCSRFToken generates a new CSRF token.
// The caller must set the returned value in a new cookie via setCSRFCookie.
// Call this after a successful login to prevent session fixation attacks.
func RegenerateCSRFToken(r *http.Request, tokenLength int) string {
	token, _ := generateCSRFToken(tokenLength)
	return token
}

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/casjay-forks/caspaste/src/audit"
	"github.com/casjay-forks/caspaste/src/netshare"
)

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

// csrfTokenStore manages CSRF tokens per session
type csrfTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]csrfTokenEntry
}

type csrfTokenEntry struct {
	token      string
	createdAt  time.Time
	lastUsedAt time.Time
}

var (
	csrfStore = &csrfTokenStore{
		tokens: make(map[string]csrfTokenEntry),
	}
	// Token validity duration
	csrfTokenTTL = 24 * time.Hour
)

// generateCSRFToken generates a cryptographically secure random token
func generateCSRFToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// getOrCreateToken gets existing token or creates new one for the session
func (s *csrfTokenStore) getOrCreateToken(sessionID string, tokenLength int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for existing valid token
	if entry, exists := s.tokens[sessionID]; exists {
		if time.Since(entry.createdAt) < csrfTokenTTL {
			entry.lastUsedAt = time.Now()
			s.tokens[sessionID] = entry
			return entry.token, nil
		}
		// Token expired, delete it
		delete(s.tokens, sessionID)
	}

	// Generate new token
	token, err := generateCSRFToken(tokenLength)
	if err != nil {
		return "", err
	}

	s.tokens[sessionID] = csrfTokenEntry{
		token:      token,
		createdAt:  time.Now(),
		lastUsedAt: time.Now(),
	}

	return token, nil
}

// validateToken validates a CSRF token for the session
func (s *csrfTokenStore) validateToken(sessionID, token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.tokens[sessionID]
	if !exists {
		return false
	}

	// Check if token is expired
	if time.Since(entry.createdAt) >= csrfTokenTTL {
		return false
	}

	// Constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(entry.token), []byte(token)) == 1
}

// cleanupExpiredTokens removes expired tokens from the store
func (s *csrfTokenStore) cleanupExpiredTokens() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for sessionID, entry := range s.tokens {
		if now.Sub(entry.createdAt) >= csrfTokenTTL {
			delete(s.tokens, sessionID)
		}
	}
}

// regenerateToken creates a new token for the session (call after login)
func (s *csrfTokenStore) regenerateToken(sessionID string, tokenLength int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete old token
	delete(s.tokens, sessionID)

	// Generate new token
	token, err := generateCSRFToken(tokenLength)
	if err != nil {
		return "", err
	}

	s.tokens[sessionID] = csrfTokenEntry{
		token:      token,
		createdAt:  time.Now(),
		lastUsedAt: time.Now(),
	}

	return token, nil
}

// getSessionID extracts or creates a session ID from the request
func getSessionID(req *http.Request) string {
	// Try to get from session cookie first
	if cookie, err := req.Cookie("session"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Try to get from auth cookie
	if cookie, err := req.Cookie("auth"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Use client IP + User-Agent as fallback session identifier
	// Not ideal but better than nothing for anonymous users
	ip := req.RemoteAddr
	// Strip port from RemoteAddr (e.g., "127.0.0.1:45678" -> "127.0.0.1")
	// RemoteAddr includes port which changes per request, breaking session tracking
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	if forwardedFor := req.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		ip = strings.TrimSpace(parts[0])
	}
	ua := req.Header.Get("User-Agent")
	encoded := base64.URLEncoding.EncodeToString([]byte(ip + ua))
	// Ensure at least 32 chars by padding if needed
	for len(encoded) < 32 {
		encoded += "0"
	}
	return encoded[:32]
}

// CSRFMiddleware creates middleware that protects against CSRF attacks
// Per AI.md PART 11: All forms MUST have CSRF protection
func CSRFMiddleware(config CSRFConfig) func(http.Handler) http.Handler {
	if !config.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Set defaults
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

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			csrfStore.cleanupExpiredTokens()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is exempt from CSRF (API endpoints, compat endpoints)
			if isCSRFExempt(r.URL.Path, config.ExemptPaths, config.ExemptPrefixes) {
				next.ServeHTTP(w, r)
				return
			}

			sessionID := getSessionID(r)

			// For safe methods (GET, HEAD, OPTIONS, TRACE), just set the token
			if isSafeMethod(r.Method) {
				token, err := csrfStore.getOrCreateToken(sessionID, config.TokenLength)
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				// Set CSRF token cookie
				setCSRFCookie(w, r, config, token)

				// Add token to response header for JavaScript access
				w.Header().Set(config.HeaderName, token)

				next.ServeHTTP(w, r)
				return
			}

			// For state-changing methods, validate the token
			token := extractCSRFToken(r, config)
			if token == "" {
				// Log CSRF failure to audit log per AI.md PART 11
				audit.CSRFFailure(netshare.GetClientAddr(r).String(), r.URL.Path, GetRequestID(r.Context()))
				http.Error(w, "CSRF token missing", http.StatusForbidden)
				return
			}

			if !csrfStore.validateToken(sessionID, token) {
				// Log CSRF failure to audit log per AI.md PART 11
				audit.CSRFFailure(netshare.GetClientAddr(r).String(), r.URL.Path, GetRequestID(r.Context()))
				http.Error(w, "CSRF token invalid", http.StatusForbidden)
				return
			}

			// Token is valid, continue
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

// isCSRFExempt checks if a path should be exempt from CSRF validation
// API endpoints and compatibility endpoints are exempt since they use tokens, not cookies
func isCSRFExempt(path string, exemptPaths, exemptPrefixes []string) bool {
	// Check exact paths
	for _, exempt := range exemptPaths {
		if path == exempt {
			return true
		}
	}

	// Check prefixes
	for _, prefix := range exemptPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// extractCSRFToken extracts the CSRF token from the request
// Checks: Header, Form field, Query parameter (in that order)
func extractCSRFToken(r *http.Request, config CSRFConfig) string {
	// Check header first (preferred for AJAX)
	if token := r.Header.Get(config.HeaderName); token != "" {
		return token
	}

	// Check X-XSRF-Token (Angular compatibility)
	if token := r.Header.Get("X-XSRF-Token"); token != "" {
		return token
	}

	// Parse form if not already parsed
	if r.Form == nil {
		r.ParseMultipartForm(32 << 20)
	}

	// Check form field
	if token := r.FormValue(config.FieldName); token != "" {
		return token
	}

	return ""
}

// setCSRFCookie sets the CSRF token cookie
func setCSRFCookie(w http.ResponseWriter, r *http.Request, config CSRFConfig, token string) {
	secure := false
	switch config.Secure {
	case "true":
		secure = true
	case "false":
		secure = false
	default:
		// auto: detect from request
		secure = r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	}

	cookie := &http.Cookie{
		Name:     config.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Must be accessible to JavaScript
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(csrfTokenTTL.Seconds()),
	}
	http.SetCookie(w, cookie)
}

// GetCSRFToken returns the current CSRF token for the request
// This is used by templates to include the token in forms
func GetCSRFToken(r *http.Request, tokenLength int) string {
	sessionID := getSessionID(r)
	token, _ := csrfStore.getOrCreateToken(sessionID, tokenLength)
	return token
}

// RegenerateCSRFToken creates a new CSRF token for the session
// Call this after successful login to prevent session fixation
func RegenerateCSRFToken(r *http.Request, tokenLength int) string {
	sessionID := getSessionID(r)
	token, _ := csrfStore.regenerateToken(sessionID, tokenLength)
	return token
}

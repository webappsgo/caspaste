// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/google/uuid"
)

// SecurityHeadersMiddleware adds security headers to all responses per AI.md PART 11
func SecurityHeadersMiddleware(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Anti-clickjacking
			if cfg.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", cfg.XFrameOptions)
			}

			// Prevent MIME-sniffing
			if cfg.XContentTypeOptions != "" {
				w.Header().Set("X-Content-Type-Options", cfg.XContentTypeOptions)
			}

			// XSS Protection (deprecated but kept for older browser compatibility per AI.md)
			if cfg.XSSProtection != "" {
				w.Header().Set("X-XSS-Protection", cfg.XSSProtection)
			}

			// Content Security Policy
			if cfg.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}

			// Referrer policy
			if cfg.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
			}

			// Permissions policy
			if cfg.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", cfg.PermissionsPolicy)
			}

			// HSTS (only if HTTPS)
			if r.TLS != nil && cfg.StrictTransportSecurity != "" {
				w.Header().Set("Strict-Transport-Security", cfg.StrictTransportSecurity)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware adds CORS headers to all responses
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins (as requested by user)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		// 24 hours
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// MaintenanceMiddleware checks for maintenance mode file
func MaintenanceMiddleware(dataDir string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		maintenanceFile := dataDir + "/.maintenance"

		// Check if maintenance mode file exists
		if _, err := os.Stat(maintenanceFile); err == nil {
			// Maintenance mode is enabled
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			// Retry after 1 hour
			w.Header().Set("Retry-After", "3600")
			w.WriteHeader(http.StatusServiceUnavailable)

			html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Maintenance Mode</title>
	<style>
		body { font-family: sans-serif; text-align: center; padding: 50px; }
		h1 { color: #e74c3c; }
	</style>
</head>
<body>
	<h1>503 - Service Unavailable</h1>
	<p>The server is currently in maintenance mode.</p>
	<p>Please try again later.</p>
</body>
</html>`
			w.Write([]byte(html))
			return
		}

		// Not in maintenance mode, continue normally
		next.ServeHTTP(w, r)
	})
}

// uuidRegex validates UUID v4 format
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isValidUUID checks if string is a valid UUID format
func isValidUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

// RequestIDKey is the context key for request ID
type RequestIDKey struct{}

// RequestIDMiddleware adds a unique request ID to each request per AI.md PART 11
// Every request MUST have a Request ID for tracing and debugging.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for existing request ID from client or upstream proxy
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = r.Header.Get("X-Correlation-ID")
		}
		if requestID == "" {
			requestID = r.Header.Get("X-Trace-ID")
		}

		// Generate new ID if none provided or invalid
		if requestID == "" || !isValidUUID(requestID) {
			requestID = uuid.New().String()
		}

		// Add to response headers
		w.Header().Set("X-Request-ID", requestID)

		// Add to request context for logging and downstream calls
		ctx := context.WithValue(r.Context(), RequestIDKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// PanicRecoveryMiddleware recovers from panics and returns appropriate error response
// Per AI.md PART 6:
// - Production: Graceful recovery, logs error, returns 500
// - Development: Verbose, full stack in response
func PanicRecoveryMiddleware(debug bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := GetRequestID(r.Context())

					// Log the panic with stack trace
					stack := make([]byte, 4096)
					n := runtime.Stack(stack, false)
					stack = stack[:n]

					// Log format includes request_id per AI.md
					logMsg := "panic recovered"
					if requestID != "" {
						logMsg += ", request_id=" + requestID
					}
					logMsg += ", error=" + fmt.Sprintf("%v", err)

					if debug {
						// Development: verbose response with stack trace
						w.Header().Set("Content-Type", "text/plain; charset=utf-8")
						w.Header().Set("X-Content-Type-Options", "nosniff")
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprintf(w, "Internal Server Error\n\n")
						fmt.Fprintf(w, "Panic: %v\n\n", err)
						fmt.Fprintf(w, "Stack Trace:\n%s\n", stack)
						if requestID != "" {
							fmt.Fprintf(w, "\nRequest ID: %s\n", requestID)
						}
					} else {
						// Production: generic error message
						w.Header().Set("Content-Type", "text/plain; charset=utf-8")
						w.Header().Set("X-Content-Type-Options", "nosniff")
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprint(w, "An unexpected error occurred")
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// URLNormalizeMiddleware normalizes URLs per AI.md PART 14
// - Removes trailing slashes (except for root "/")
// - 301 redirects to canonical path
// - Preserves query string
// This should be the FIRST middleware in the chain.
func URLNormalizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Don't normalize root path
		if path == "/" {
			next.ServeHTTP(w, r)
			return
		}

		// Check for trailing slash
		if len(path) > 1 && strings.HasSuffix(path, "/") {
			// Build canonical URL without trailing slash
			canonical := strings.TrimSuffix(path, "/")
			if r.URL.RawQuery != "" {
				canonical += "?" + r.URL.RawQuery
			}

			// 301 Permanent Redirect to canonical URL
			http.Redirect(w, r, canonical, http.StatusMovedPermanently)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// PathSecurityMiddleware blocks path traversal attacks per AI.md PART 11
// - Blocks ".." in paths
// - Blocks encoded traversal attempts (%2e%2e, %2E%2E)
// - Returns 400 Bad Request for malicious paths
func PathSecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Check for path traversal in raw path
		if strings.Contains(path, "..") {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Check for encoded traversal attempts
		// URL decode and check again
		decoded, err := url.PathUnescape(path)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if strings.Contains(decoded, "..") {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Also check the raw query for traversal attempts
		if strings.Contains(r.URL.RawQuery, "..") {
			decodedQuery, err := url.QueryUnescape(r.URL.RawQuery)
			if err == nil && strings.Contains(decodedQuery, "..") {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// UserContextKey is the context key for the (optional) authenticated user.
// CasPaste is an anonymous pastebin; this exists only so templates that
// already reference .User keep working — GetAuthUser will always return nil
// in the anonymous flow.
type UserContextKey struct{}

// AuthUser represents an authenticated user in the request context.
// Retained as a no-op type so existing page-data structs and templates
// (e.g., {{if .User}}) continue to compile and render.
type AuthUser struct {
	ID            int64
	Username      string
	Email         string
	DisplayName   string
	Role          string
	AvatarURL     string
	EmailVerified bool
	TOTPEnabled   bool
}

// GetAuthUser retrieves the authenticated user from context, or nil.
func GetAuthUser(ctx context.Context) *AuthUser {
	if user, ok := ctx.Value(UserContextKey{}).(*AuthUser); ok {
		return user
	}
	return nil
}


// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package admin

import (
	"context"
	"net/http"
)

// contextKey is the type used for admin context keys
type contextKey int

const (
	ctxKeyAdminID       contextKey = iota
	ctxKeyAdminUsername contextKey = iota
)

// requireAuth wraps a handler requiring an authenticated admin session.
// In debug mode authentication is bypassed per AI.md PART 6.
func (p *Panel) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Debug mode bypasses auth entirely per AI.md PART 6
		if p.debug {
			ctx := context.WithValue(r.Context(), ctxKeyAdminID, int64(0))
			ctx = context.WithValue(ctx, ctxKeyAdminUsername, "debug")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		adminID := p.validateAdminSession(r)
		if adminID == 0 {
			// Not authenticated — redirect to shared auth login per AI.md PART 15
			returnTo := r.URL.RequestURI()
			http.Redirect(w, r, "/server/auth/login?next="+returnTo, http.StatusSeeOther)
			return
		}

		// Look up the admin to put their username in context
		a, err := p.getAdminByID(adminID)
		if err != nil || a == nil {
			// Session points to a non-existent admin — clear it
			p.deleteAdminSession(w, r)
			http.Redirect(w, r, "/server/auth/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyAdminID, adminID)
		ctx = context.WithValue(ctx, ctxKeyAdminUsername, a.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// currentAdminUsername reads the admin username from request context
func currentAdminUsername(r *http.Request) string {
	if v, ok := r.Context().Value(ctxKeyAdminUsername).(string); ok {
		return v
	}
	return ""
}

// currentAdminID reads the admin ID from request context
func currentAdminID(r *http.Request) int64 {
	if v, ok := r.Context().Value(ctxKeyAdminID).(int64); ok {
		return v
	}
	return 0
}

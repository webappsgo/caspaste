
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

// adminSessionCookieName is the cookie used for admin sessions.
// Deliberately different from the public-site session cookie.
const adminSessionCookieName = "caspaste_admin"

const adminSessionDuration = 30 * 24 * time.Hour

// createAdminSession creates an admin session row and sets the session cookie
func (p *Panel) createAdminSession(w http.ResponseWriter, r *http.Request, adminID int64) error {
	if p.db == nil {
		return errNoDB
	}

	// Generate a 32-byte random token
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	token := hex.EncodeToString(raw)

	// Store hash only — never persist the raw token
	h := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(h[:])

	expiresAt := time.Now().Add(adminSessionDuration).Unix()
	ipAddr := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ipAddr = fwd
	}
	userAgent := r.Header.Get("User-Agent")

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO admin_sessions (admin_id, token_hash, ip_address, user_agent, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		adminID, tokenHash, ipAddr, userAgent, expiresAt)
	if err != nil {
		return err
	}

	// Cookie path scoped to the admin area only
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    token,
		Path:     p.adminBasePath(),
		Expires:  time.Now().Add(adminSessionDuration),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
	return nil
}

// validateAdminSession reads the session cookie and returns the adminID (0 = invalid/missing)
func (p *Panel) validateAdminSession(r *http.Request) int64 {
	if p.db == nil {
		return 0
	}
	cookie, err := r.Cookie(adminSessionCookieName)
	if err != nil {
		return 0
	}

	h := sha256.Sum256([]byte(cookie.Value))
	tokenHash := hex.EncodeToString(h[:])

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var adminID int64
	var expiresAt int64
	err = p.db.QueryRowContext(ctx,
		`SELECT admin_id, expires_at FROM admin_sessions WHERE token_hash = ?`,
		tokenHash,
	).Scan(&adminID, &expiresAt)
	if err != nil {
		return 0
	}
	if time.Now().Unix() > expiresAt {
		// Expired — clean up opportunistically
		_, _ = p.db.ExecContext(context.Background(),
			`DELETE FROM admin_sessions WHERE token_hash = ?`, tokenHash)
		return 0
	}
	return adminID
}

// deleteAdminSession removes the session from the DB and clears the cookie
func (p *Panel) deleteAdminSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(adminSessionCookieName)
	if err == nil && p.db != nil {
		h := sha256.Sum256([]byte(cookie.Value))
		tokenHash := hex.EncodeToString(h[:])
		ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
		defer cancel()
		_, _ = p.db.ExecContext(ctx,
			`DELETE FROM admin_sessions WHERE token_hash = ?`, tokenHash)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    "",
		Path:     p.adminBasePath(),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})
}

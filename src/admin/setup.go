
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// MaybeGenerateSetupToken checks whether admin setup is needed.
// If the admins table is empty, a one-time setup token is generated and
// logged so the operator can visit /server/{admin_path}/config/setup?token={token}.
// Returns true when setup is still required.
func (p *Panel) MaybeGenerateSetupToken() bool {
	count, err := p.CountAdmins()
	if err == nil && count > 0 {
		p.mu.Lock()
		p.setupDone = true
		p.mu.Unlock()
		return false
	}

	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	token := hex.EncodeToString(raw)

	p.mu.Lock()
	p.setupToken = token
	p.setupExpiry = time.Now().Add(time.Hour)
	p.setupDone = false
	p.mu.Unlock()

	base := p.adminBasePath()
	fmt.Println("==========================================================")
	fmt.Println(" Admin setup required — no admin accounts found")
	fmt.Printf(" Visit: %s/config/setup?token=%s\n", base, token)
	fmt.Printf(" Token expires: %s\n", p.setupExpiry.Format(time.RFC3339))
	fmt.Println("==========================================================")
	return true
}

// isSetupNeeded returns true when no admin exists and a valid setup token is pending
func (p *Panel) isSetupNeeded() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.setupDone && p.setupToken != ""
}

// consumeSetupToken validates and atomically consumes the setup token
func (p *Panel) consumeSetupToken(token string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if token == "" || p.setupToken == "" {
		return false
	}
	if p.setupToken != token {
		return false
	}
	if time.Now().After(p.setupExpiry) {
		return false
	}
	p.setupToken = ""
	p.setupDone = true
	return true
}

// peekSetupToken returns true if the token is currently valid (without consuming it)
func (p *Panel) peekSetupToken(token string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return token != "" && p.setupToken != "" &&
		p.setupToken == token &&
		time.Now().Before(p.setupExpiry)
}

// handleSetup handles GET/POST for /config/setup
func (p *Panel) handleSetup(w http.ResponseWriter, r *http.Request) {
	// After setup is done, redirect to dashboard
	if !p.isSetupNeeded() {
		http.Redirect(w, r, p.adminBasePath()+"/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		token := r.URL.Query().Get("token")
		if !p.peekSetupToken(token) {
			p.renderErrorPage(w, r, "Invalid or expired setup token — check the server log for the current token.")
			return
		}
		p.renderSetupPage(w, r, token, "")
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		p.renderSetupPage(w, r, "", "Failed to parse form")
		return
	}

	formToken := r.FormValue("setup_token")
	if !p.peekSetupToken(formToken) {
		p.renderSetupPage(w, r, formToken, "Invalid or expired setup token")
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")
	email := strings.TrimSpace(r.FormValue("email"))

	switch {
	case username == "":
		p.renderSetupPage(w, r, formToken, "Username is required")
		return
	case len(username) < 3 || len(username) > 64:
		p.renderSetupPage(w, r, formToken, "Username must be 3-64 characters")
		return
	case password == "":
		p.renderSetupPage(w, r, formToken, "Password is required")
		return
	case len(password) < 8:
		p.renderSetupPage(w, r, formToken, "Password must be at least 8 characters")
		return
	case password != confirm:
		p.renderSetupPage(w, r, formToken, "Passwords do not match")
		return
	}

	if err := p.CreateAdmin(username, password, email); err != nil {
		p.renderSetupPage(w, r, formToken, "Failed to create admin account: "+err.Error())
		return
	}

	// Consume the token only after successful DB write
	if !p.consumeSetupToken(formToken) {
		// Very unlikely race — another request already consumed it
		p.renderSetupPage(w, r, formToken, "Setup token was already used")
		return
	}

	fmt.Printf("INFO [admin] Admin account created via setup wizard: %s\n", username)
	http.Redirect(w, r, p.adminBasePath()+"/", http.StatusSeeOther)
}

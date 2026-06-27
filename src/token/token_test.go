
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package token

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory SQLite database with all required tables.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_tokens (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id     INTEGER NOT NULL,
			name        TEXT NOT NULL,
			token_prefix TEXT NOT NULL,
			token_hash  TEXT NOT NULL,
			scopes      TEXT,
			last_used_at INTEGER,
			expires_at  INTEGER,
			created_at  INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create user_tokens table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS org_tokens (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id      INTEGER NOT NULL,
			created_by  INTEGER NOT NULL,
			name        TEXT NOT NULL,
			token_prefix TEXT NOT NULL,
			token_hash  TEXT NOT NULL,
			scopes      TEXT,
			last_used_at INTEGER,
			expires_at  INTEGER,
			created_at  INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create org_tokens table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS admins (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			username      TEXT NOT NULL,
			role          TEXT NOT NULL DEFAULT 'admin',
			enabled       INTEGER NOT NULL DEFAULT 1,
			locked_until  INTEGER,
			api_token_hash TEXT,
			updated_at    INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("failed to create admins table: %v", err)
	}

	return db
}

func TestGenerateRawToken(t *testing.T) {
	tok, err := generateRawToken(32)
	if err != nil {
		t.Fatalf("generateRawToken: unexpected error: %v", err)
	}
	if len(tok) != 64 {
		t.Errorf("expected hex string of length 64 (32 bytes), got %d", len(tok))
	}
	for _, c := range tok {
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			t.Errorf("generateRawToken: non-hex character %q in output", c)
		}
	}
}

func TestGenerateRawToken_Unique(t *testing.T) {
	tok1, err := generateRawToken(32)
	if err != nil {
		t.Fatal(err)
	}
	tok2, err := generateRawToken(32)
	if err != nil {
		t.Fatal(err)
	}
	if tok1 == tok2 {
		t.Error("expected two tokens to differ")
	}
}

func TestHashToken(t *testing.T) {
	h1 := hashToken("usr_testtoken")
	h2 := hashToken("usr_testtoken")
	if h1 != h2 {
		t.Error("hashToken: expected deterministic output")
	}
	if len(h1) != 64 {
		t.Errorf("hashToken: expected SHA-256 hex length 64, got %d", len(h1))
	}
}

func TestHashToken_DifferentInputs(t *testing.T) {
	h1 := hashToken("usr_tokenA")
	h2 := hashToken("usr_tokenB")
	if h1 == h2 {
		t.Error("hashToken: expected different hashes for different inputs")
	}
}

func TestTokenConstants(t *testing.T) {
	if PrefixUser != "usr_" {
		t.Errorf("PrefixUser = %q, want %q", PrefixUser, "usr_")
	}
	if PrefixOrg != "org_" {
		t.Errorf("PrefixOrg = %q, want %q", PrefixOrg, "org_")
	}
	if PrefixAdm != "adm_" {
		t.Errorf("PrefixAdm = %q, want %q", PrefixAdm, "adm_")
	}
	if ScopeGlobal != "global" {
		t.Errorf("ScopeGlobal = %q, want %q", ScopeGlobal, "global")
	}
	if ScopeReadWrite != "read-write" {
		t.Errorf("ScopeReadWrite = %q, want %q", ScopeReadWrite, "read-write")
	}
	if ScopeRead != "read" {
		t.Errorf("ScopeRead = %q, want %q", ScopeRead, "read")
	}
}

func TestTokenInfoHasScope(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		check  string
		want   bool
	}{
		{"global scope grants all", []string{ScopeGlobal}, ScopeRead, true},
		{"global scope grants read-write", []string{ScopeGlobal}, ScopeReadWrite, true},
		{"read-write grants read-write", []string{ScopeReadWrite}, ScopeReadWrite, true},
		{"read-write does not grant global", []string{ScopeReadWrite}, ScopeGlobal, false},
		{"read grants read", []string{ScopeRead}, ScopeRead, true},
		{"read does not grant global", []string{ScopeRead}, ScopeGlobal, false},
		{"no scopes denies", []string{}, ScopeRead, false},
		{"exact scope match", []string{ScopeRead}, ScopeRead, true},
	}

	for _, tc := range tests {
		info := &TokenInfo{Scopes: tc.scopes}
		got := info.HasScope(tc.check)
		if got != tc.want {
			t.Errorf("HasScope [%s]: scopes=%v check=%q got=%v want=%v",
				tc.name, tc.scopes, tc.check, got, tc.want)
		}
	}
}

func TestTokenInfoCanWrite(t *testing.T) {
	tests := []struct {
		scopes []string
		want   bool
	}{
		{[]string{ScopeGlobal}, true},
		{[]string{ScopeReadWrite}, true},
		{[]string{ScopeRead}, false},
		{[]string{}, false},
	}

	for _, tc := range tests {
		info := &TokenInfo{Scopes: tc.scopes}
		if info.CanWrite() != tc.want {
			t.Errorf("CanWrite(%v) = %v, want %v", tc.scopes, info.CanWrite(), tc.want)
		}
	}
}

func TestTokenInfoCanRead(t *testing.T) {
	tests := []struct {
		scopes []string
		want   bool
	}{
		{[]string{ScopeGlobal}, true},
		{[]string{ScopeReadWrite}, true},
		{[]string{ScopeRead}, true},
		{[]string{}, false},
	}

	for _, tc := range tests {
		info := &TokenInfo{Scopes: tc.scopes}
		if info.CanRead() != tc.want {
			t.Errorf("CanRead(%v) = %v, want %v", tc.scopes, info.CanRead(), tc.want)
		}
	}
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	if svc == nil {
		t.Fatal("NewService: expected non-nil service")
	}
}

func TestValidate_EmptyToken(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	_, err := svc.Validate("")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidate_UnknownPrefix(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	_, err := svc.Validate("xyz_somecontent")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for unknown prefix, got %v", err)
	}
}

func TestCreateAndValidateUserToken(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	fullToken, tok, err := svc.CreateUserToken(42, "my-token", []string{ScopeRead, ScopeReadWrite}, nil)
	if err != nil {
		t.Fatalf("CreateUserToken: unexpected error: %v", err)
	}
	if tok == nil {
		t.Fatal("CreateUserToken: expected non-nil token struct")
	}
	if !strings.HasPrefix(fullToken, PrefixUser) {
		t.Errorf("expected token to start with %q, got %q", PrefixUser, fullToken[:8])
	}
	if tok.OwnerID != 42 {
		t.Errorf("expected OwnerID=42, got %d", tok.OwnerID)
	}
	if tok.Name != "my-token" {
		t.Errorf("expected name=%q, got %q", "my-token", tok.Name)
	}

	info, err := svc.Validate(fullToken)
	if err != nil {
		t.Fatalf("Validate user token: unexpected error: %v", err)
	}
	if info.Type != "user" {
		t.Errorf("expected type=%q, got %q", "user", info.Type)
	}
	if info.OwnerID != 42 {
		t.Errorf("expected OwnerID=42, got %d", info.OwnerID)
	}
}

func TestCreateUserToken_Scopes(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	fullToken, _, err := svc.CreateUserToken(1, "scoped", []string{ScopeRead}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := svc.Validate(fullToken)
	if err != nil {
		t.Fatalf("Validate: unexpected error: %v", err)
	}
	if !info.CanRead() {
		t.Error("expected CanRead=true for read scope")
	}
	if info.CanWrite() {
		t.Error("expected CanWrite=false for read-only scope")
	}
}

func TestCreateUserToken_WithExpiry(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	future := time.Now().Add(time.Hour).Unix()
	fullToken, tok, err := svc.CreateUserToken(1, "expiring", []string{ScopeRead}, &future)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.ExpiresAt == nil || *tok.ExpiresAt != future {
		t.Error("expected expires_at to be set")
	}

	_, err = svc.Validate(fullToken)
	if err != nil {
		t.Errorf("Validate: expected valid token (not expired), got: %v", err)
	}
}

func TestValidateUserToken_Expired(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	past := time.Now().Add(-time.Hour).Unix()
	fullToken, _, err := svc.CreateUserToken(1, "expired", []string{ScopeRead}, &past)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.Validate(fullToken)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateUserToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	fakeToken := PrefixUser + strings.Repeat("a", 64)
	_, err := svc.Validate(fakeToken)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestCreateAndValidateOrgToken(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	fullToken, tok, err := svc.CreateOrgToken(10, 5, "org-token", []string{ScopeGlobal}, nil)
	if err != nil {
		t.Fatalf("CreateOrgToken: unexpected error: %v", err)
	}
	if !strings.HasPrefix(fullToken, PrefixOrg) {
		t.Errorf("expected org token prefix %q, got prefix %q", PrefixOrg, fullToken[:4])
	}
	if tok.OrgID != 10 {
		t.Errorf("expected OrgID=10, got %d", tok.OrgID)
	}
	if tok.CreatedBy != 5 {
		t.Errorf("expected CreatedBy=5, got %d", tok.CreatedBy)
	}

	info, err := svc.Validate(fullToken)
	if err != nil {
		t.Fatalf("Validate org token: unexpected error: %v", err)
	}
	if info.Type != "org" {
		t.Errorf("expected type=%q, got %q", "org", info.Type)
	}
	if info.OwnerID != 10 {
		t.Errorf("expected OwnerID=10, got %d", info.OwnerID)
	}
	if info.UserID != 5 {
		t.Errorf("expected UserID=5 (created_by), got %d", info.UserID)
	}
}

func TestValidateOrgToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	fakeToken := PrefixOrg + strings.Repeat("b", 64)
	_, err := svc.Validate(fakeToken)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestValidateOrgToken_Expired(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	past := time.Now().Add(-time.Hour).Unix()
	fullToken, _, err := svc.CreateOrgToken(10, 5, "expired-org", []string{ScopeRead}, &past)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.Validate(fullToken)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestCreateAndValidateAdminToken(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	result, err := db.Exec(`INSERT INTO admins (username, role, enabled) VALUES (?, ?, ?)`,
		"superadmin", "superadmin", 1)
	if err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	adminID, _ := result.LastInsertId()

	fullToken, err := svc.CreateAdminToken(adminID)
	if err != nil {
		t.Fatalf("CreateAdminToken: unexpected error: %v", err)
	}
	if !strings.HasPrefix(fullToken, PrefixAdm) {
		t.Errorf("expected admin token prefix %q", PrefixAdm)
	}

	info, err := svc.Validate(fullToken)
	if err != nil {
		t.Fatalf("Validate admin token: unexpected error: %v", err)
	}
	if info.Type != "admin" {
		t.Errorf("expected type=%q, got %q", "admin", info.Type)
	}
	if !info.HasScope(ScopeGlobal) {
		t.Error("expected superadmin to have global scope")
	}
}

func TestValidateAdminToken_Revoked(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	result, err := db.Exec(`INSERT INTO admins (username, role, enabled) VALUES (?, ?, ?)`,
		"disabledadmin", "admin", 0)
	if err != nil {
		t.Fatalf("insert disabled admin: %v", err)
	}
	adminID, _ := result.LastInsertId()

	fullToken, err := svc.CreateAdminToken(adminID)
	if err != nil {
		t.Fatalf("CreateAdminToken: unexpected error: %v", err)
	}

	_, err = svc.Validate(fullToken)
	if err != ErrTokenRevoked {
		t.Errorf("expected ErrTokenRevoked for disabled admin, got %v", err)
	}
}

func TestValidateAdminToken_Locked(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	future := time.Now().Add(time.Hour).Unix()
	result, err := db.Exec(`INSERT INTO admins (username, role, enabled, locked_until) VALUES (?, ?, ?, ?)`,
		"lockedadmin", "admin", 1, future)
	if err != nil {
		t.Fatalf("insert locked admin: %v", err)
	}
	adminID, _ := result.LastInsertId()

	fullToken, err := svc.CreateAdminToken(adminID)
	if err != nil {
		t.Fatalf("CreateAdminToken: unexpected error: %v", err)
	}

	_, err = svc.Validate(fullToken)
	if err != ErrTokenRevoked {
		t.Errorf("expected ErrTokenRevoked for locked admin, got %v", err)
	}
}

func TestValidateAdminToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	fakeToken := PrefixAdm + strings.Repeat("c", 64)
	_, err := svc.Validate(fakeToken)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestAdminToken_Roles(t *testing.T) {
	roles := []struct {
		role      string
		wantGlobal bool
		wantRead   bool
	}{
		{"superadmin", true, true},
		{"admin", true, true},
		{"readonly", false, true},
		{"unknown", true, true},
	}

	for _, tc := range roles {
		db := setupTestDB(t)
		svc := NewService(db)

		result, err := db.Exec(`INSERT INTO admins (username, role, enabled) VALUES (?, ?, ?)`,
			"testadmin", tc.role, 1)
		if err != nil {
			t.Fatalf("[%s] insert admin: %v", tc.role, err)
		}
		adminID, _ := result.LastInsertId()

		fullToken, err := svc.CreateAdminToken(adminID)
		if err != nil {
			t.Fatalf("[%s] CreateAdminToken: %v", tc.role, err)
		}

		info, err := svc.Validate(fullToken)
		if err != nil {
			t.Fatalf("[%s] Validate: %v", tc.role, err)
		}

		if info.HasScope(ScopeGlobal) != tc.wantGlobal {
			t.Errorf("[%s] HasScope(global)=%v, want %v", tc.role, info.HasScope(ScopeGlobal), tc.wantGlobal)
		}
		if info.CanRead() != tc.wantRead {
			t.Errorf("[%s] CanRead()=%v, want %v", tc.role, info.CanRead(), tc.wantRead)
		}
	}
}

func TestCreateAdminToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	_, err := svc.CreateAdminToken(99999)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound for non-existent admin, got %v", err)
	}
}

func TestRevokeAdminToken(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	result, err := db.Exec(`INSERT INTO admins (username, role, enabled) VALUES (?, ?, ?)`,
		"revoketest", "admin", 1)
	if err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	adminID, _ := result.LastInsertId()

	fullToken, err := svc.CreateAdminToken(adminID)
	if err != nil {
		t.Fatalf("CreateAdminToken: %v", err)
	}

	if err := svc.RevokeAdminToken(adminID); err != nil {
		t.Fatalf("RevokeAdminToken: %v", err)
	}

	_, err = svc.Validate(fullToken)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound after revoke, got %v", err)
	}
}

func TestRevokeAdminToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	err := svc.RevokeAdminToken(99999)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound for non-existent admin, got %v", err)
	}
}

func TestRevokeUserToken(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	fullToken, tok, err := svc.CreateUserToken(1, "to-revoke", []string{ScopeRead}, nil)
	if err != nil {
		t.Fatalf("CreateUserToken: %v", err)
	}

	if err := svc.RevokeUserToken(tok.ID, 1); err != nil {
		t.Fatalf("RevokeUserToken: %v", err)
	}

	_, err = svc.Validate(fullToken)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound after revoke, got %v", err)
	}
}

func TestRevokeUserToken_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	_, tok, err := svc.CreateUserToken(1, "token", []string{ScopeRead}, nil)
	if err != nil {
		t.Fatalf("CreateUserToken: %v", err)
	}

	err = svc.RevokeUserToken(tok.ID, 999)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound when revoking with wrong user, got %v", err)
	}
}

func TestRevokeOrgToken(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	fullToken, tok, err := svc.CreateOrgToken(10, 5, "to-revoke", []string{ScopeRead}, nil)
	if err != nil {
		t.Fatalf("CreateOrgToken: %v", err)
	}

	if err := svc.RevokeOrgToken(tok.ID, 10); err != nil {
		t.Fatalf("RevokeOrgToken: %v", err)
	}

	_, err = svc.Validate(fullToken)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound after revoke, got %v", err)
	}
}

func TestRevokeOrgToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	err := svc.RevokeOrgToken(99999, 10)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestListUserTokens(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	_, _, err := svc.CreateUserToken(1, "token-a", []string{ScopeRead}, nil)
	if err != nil {
		t.Fatalf("CreateUserToken a: %v", err)
	}
	_, _, err = svc.CreateUserToken(1, "token-b", []string{ScopeReadWrite}, nil)
	if err != nil {
		t.Fatalf("CreateUserToken b: %v", err)
	}
	_, _, err = svc.CreateUserToken(2, "other-user-token", []string{ScopeGlobal}, nil)
	if err != nil {
		t.Fatalf("CreateUserToken other: %v", err)
	}

	tokens, err := svc.ListUserTokens(1)
	if err != nil {
		t.Fatalf("ListUserTokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens for user 1, got %d", len(tokens))
	}
}

func TestListUserTokens_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	tokens, err := svc.ListUserTokens(999)
	if err != nil {
		t.Fatalf("ListUserTokens: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}
}

func TestListOrgTokens(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	_, _, err := svc.CreateOrgToken(10, 1, "org-a", []string{ScopeRead}, nil)
	if err != nil {
		t.Fatalf("CreateOrgToken a: %v", err)
	}
	_, _, err = svc.CreateOrgToken(10, 2, "org-b", []string{ScopeReadWrite}, nil)
	if err != nil {
		t.Fatalf("CreateOrgToken b: %v", err)
	}

	tokens, err := svc.ListOrgTokens(10)
	if err != nil {
		t.Fatalf("ListOrgTokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens for org 10, got %d", len(tokens))
	}
}

func TestCountUserTokens(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	count, err := svc.CountUserTokens(1)
	if err != nil {
		t.Fatalf("CountUserTokens: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 initially, got %d", count)
	}

	svc.CreateUserToken(1, "tok1", []string{ScopeRead}, nil)
	svc.CreateUserToken(1, "tok2", []string{ScopeRead}, nil)

	count, err = svc.CountUserTokens(1)
	if err != nil {
		t.Fatalf("CountUserTokens: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestCountOrgTokens(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	count, err := svc.CountOrgTokens(10)
	if err != nil {
		t.Fatalf("CountOrgTokens: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 initially, got %d", count)
	}

	svc.CreateOrgToken(10, 1, "org-tok", []string{ScopeRead}, nil)

	count, err = svc.CountOrgTokens(10)
	if err != nil {
		t.Fatalf("CountOrgTokens: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestCleanupExpired(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	past := time.Now().Add(-time.Hour).Unix()
	future := time.Now().Add(time.Hour).Unix()

	svc.CreateUserToken(1, "expired-user", []string{ScopeRead}, &past)
	svc.CreateUserToken(1, "active-user", []string{ScopeRead}, &future)
	svc.CreateUserToken(1, "no-expiry", []string{ScopeRead}, nil)
	svc.CreateOrgToken(10, 1, "expired-org", []string{ScopeRead}, &past)
	svc.CreateOrgToken(10, 1, "active-org", []string{ScopeRead}, &future)

	if err := svc.CleanupExpired(); err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}

	userCount, _ := svc.CountUserTokens(1)
	if userCount != 2 {
		t.Errorf("expected 2 user tokens after cleanup (1 active + 1 no-expiry), got %d", userCount)
	}

	orgCount, _ := svc.CountOrgTokens(10)
	if orgCount != 1 {
		t.Errorf("expected 1 org token after cleanup, got %d", orgCount)
	}
}

func TestUpdateLastUsed_InvalidTable(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Should fail silently (not panic or return error) for invalid table names
	svc.updateLastUsed("drop_tables", 1)
	svc.updateLastUsed("", 1)
}

func TestListUserTokens_WithExpiry(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	expiry := time.Now().Add(time.Hour).Unix()
	_, _, err := svc.CreateUserToken(5, "expiring", []string{ScopeRead}, &expiry)
	if err != nil {
		t.Fatalf("CreateUserToken: %v", err)
	}

	tokens, err := svc.ListUserTokens(5)
	if err != nil {
		t.Fatalf("ListUserTokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
	if tokens[0].ExpiresAt == nil || *tokens[0].ExpiresAt != expiry {
		t.Error("expected ExpiresAt to be set and match")
	}
}

func TestListOrgTokens_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	tokens, err := svc.ListOrgTokens(999)
	if err != nil {
		t.Fatalf("ListOrgTokens: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}
}

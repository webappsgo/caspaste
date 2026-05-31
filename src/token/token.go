
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

// Query timeouts per AI.md PART 10
const (
	defaultQueryTimeout = 5 * time.Second
	defaultListTimeout  = 10 * time.Second
)

// Token prefix constants per PART 34
const (
	PrefixUser = "usr_"
	PrefixOrg  = "org_"
	PrefixAdm  = "adm_"
)

// Scope constants
const (
	ScopeGlobal    = "global"
	ScopeReadWrite = "read-write"
	ScopeRead      = "read"
)

// Common errors
var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
	ErrInvalidToken  = errors.New("invalid token")
	ErrTokenRevoked  = errors.New("token has been revoked")
)

// Token represents an API token
type Token struct {
	ID          int64   `json:"id"`
	OwnerID     int64   `json:"owner_id,omitempty"`
	OrgID       int64   `json:"org_id,omitempty"`
	CreatedBy   int64   `json:"created_by,omitempty"`
	Name        string  `json:"name"`
	TokenPrefix string  `json:"token_prefix"`
	TokenHash   string  `json:"-"`
	Scopes      string  `json:"scopes,omitempty"`
	LastUsedAt  *int64  `json:"last_used_at,omitempty"`
	ExpiresAt   *int64  `json:"expires_at,omitempty"`
	CreatedAt   int64   `json:"created_at"`
}

// TokenInfo contains information about a validated token
type TokenInfo struct {
	// "user", "org", "admin"
	Type    string
	// User ID for user tokens, Org ID for org tokens
	OwnerID int64
	// User ID (for org tokens, this is the user who created it)
	UserID  int64
	Scopes    []string
	Token     *Token
}

// Service provides token operations
type Service struct {
	db *sql.DB
}

// NewService creates a new token service
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// CreateUserToken creates a new API token for a user
func (s *Service) CreateUserToken(userID int64, name string, scopes []string, expiresAt *int64) (string, *Token, error) {
	// Generate token
	rawToken, err := generateRawToken(32)
	if err != nil {
		return "", nil, err
	}

	// Add prefix
	fullToken := PrefixUser + rawToken

	// Hash for storage
	tokenHash := hashToken(fullToken)

	// Prefix for display
	tokenPrefix := fullToken[:12] + "..."

	// Convert scopes to string
	scopeStr := strings.Join(scopes, ",")

	now := time.Now().Unix()

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO user_tokens (user_id, name, token_prefix, token_hash, scopes, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, name, tokenPrefix, tokenHash, scopeStr, expiresAt, now)
	if err != nil {
		return "", nil, err
	}

	id, _ := result.LastInsertId()

	token := &Token{
		ID:          id,
		OwnerID:     userID,
		Name:        name,
		TokenPrefix: tokenPrefix,
		Scopes:      scopeStr,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
	}

	return fullToken, token, nil
}

// CreateOrgToken creates a new API token for an organization
func (s *Service) CreateOrgToken(orgID, createdBy int64, name string, scopes []string, expiresAt *int64) (string, *Token, error) {
	// Generate token
	rawToken, err := generateRawToken(32)
	if err != nil {
		return "", nil, err
	}

	// Add prefix
	fullToken := PrefixOrg + rawToken

	// Hash for storage
	tokenHash := hashToken(fullToken)

	// Prefix for display
	tokenPrefix := fullToken[:12] + "..."

	// Convert scopes to string
	scopeStr := strings.Join(scopes, ",")

	now := time.Now().Unix()

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO org_tokens (org_id, created_by, name, token_prefix, token_hash, scopes, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, orgID, createdBy, name, tokenPrefix, tokenHash, scopeStr, expiresAt, now)
	if err != nil {
		return "", nil, err
	}

	id, _ := result.LastInsertId()

	token := &Token{
		ID:          id,
		OrgID:       orgID,
		CreatedBy:   createdBy,
		Name:        name,
		TokenPrefix: tokenPrefix,
		Scopes:      scopeStr,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
	}

	return fullToken, token, nil
}

// Validate validates an API token and returns token info
func (s *Service) Validate(token string) (*TokenInfo, error) {
	if token == "" {
		return nil, ErrInvalidToken
	}

	// Determine token type by prefix
	var tokenType string
	if strings.HasPrefix(token, PrefixUser) {
		tokenType = "user"
	} else if strings.HasPrefix(token, PrefixOrg) {
		tokenType = "org"
	} else if strings.HasPrefix(token, PrefixAdm) {
		tokenType = "admin"
	} else {
		return nil, ErrInvalidToken
	}

	tokenHash := hashToken(token)

	switch tokenType {
	case "user":
		return s.validateUserToken(tokenHash)
	case "org":
		return s.validateOrgToken(tokenHash)
	case "admin":
		return s.validateAdminToken(tokenHash)
	}

	return nil, ErrInvalidToken
}

// validateUserToken validates a user API token
func (s *Service) validateUserToken(tokenHash string) (*TokenInfo, error) {
	var t Token
	var expiresAt, lastUsedAt sql.NullInt64

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, token_prefix, scopes, last_used_at, expires_at, created_at
		FROM user_tokens WHERE token_hash = ?
	`, tokenHash).Scan(
		&t.ID, &t.OwnerID, &t.Name, &t.TokenPrefix,
		&t.Scopes, &lastUsedAt, &expiresAt, &t.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	// Check expiration
	if expiresAt.Valid && expiresAt.Int64 < time.Now().Unix() {
		return nil, ErrTokenExpired
	}

	// Update last used
	s.updateLastUsed("user_tokens", t.ID)

	// Parse scopes
	var scopes []string
	if t.Scopes != "" {
		scopes = strings.Split(t.Scopes, ",")
	}

	return &TokenInfo{
		Type:    "user",
		OwnerID: t.OwnerID,
		UserID:  t.OwnerID,
		Scopes:  scopes,
		Token:   &t,
	}, nil
}

// validateOrgToken validates an organization API token
func (s *Service) validateOrgToken(tokenHash string) (*TokenInfo, error) {
	var t Token
	var expiresAt, lastUsedAt sql.NullInt64

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	err := s.db.QueryRowContext(ctx, `
		SELECT id, org_id, created_by, name, token_prefix, scopes, last_used_at, expires_at, created_at
		FROM org_tokens WHERE token_hash = ?
	`, tokenHash).Scan(
		&t.ID, &t.OrgID, &t.CreatedBy, &t.Name, &t.TokenPrefix,
		&t.Scopes, &lastUsedAt, &expiresAt, &t.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	// Check expiration
	if expiresAt.Valid && expiresAt.Int64 < time.Now().Unix() {
		return nil, ErrTokenExpired
	}

	// Update last used
	s.updateLastUsed("org_tokens", t.ID)

	// Parse scopes
	var scopes []string
	if t.Scopes != "" {
		scopes = strings.Split(t.Scopes, ",")
	}

	return &TokenInfo{
		Type:    "org",
		OwnerID: t.OrgID,
		UserID:  t.CreatedBy,
		Scopes:  scopes,
		Token:   &t,
	}, nil
}

// validateAdminToken validates a server admin API token per AI.md PART 11
// Admin tokens are stored in the admins table api_token_hash field
func (s *Service) validateAdminToken(tokenHash string) (*TokenInfo, error) {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var adminID int64
	var username string
	var role string
	var enabled int
	var lockedUntil sql.NullInt64

	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, role, enabled, locked_until
		FROM admins WHERE api_token_hash = ?
	`, tokenHash).Scan(&adminID, &username, &role, &enabled, &lockedUntil)
	if err == sql.ErrNoRows {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	// Check if admin is enabled
	if enabled != 1 {
		return nil, ErrTokenRevoked
	}

	// Check if account is locked
	if lockedUntil.Valid && lockedUntil.Int64 > time.Now().Unix() {
		return nil, ErrTokenRevoked
	}

	// Determine scopes based on admin role
	var scopes []string
	switch role {
	case "superadmin":
		scopes = []string{ScopeGlobal}
	case "admin":
		scopes = []string{ScopeGlobal}
	case "readonly":
		scopes = []string{ScopeRead}
	default:
		scopes = []string{ScopeGlobal}
	}

	return &TokenInfo{
		Type:    "admin",
		OwnerID: adminID,
		UserID:  adminID,
		Scopes:  scopes,
		Token: &Token{
			ID:       adminID,
			OwnerID:  adminID,
			Name:     username,
			Scopes:   strings.Join(scopes, ","),
		},
	}, nil
}

// CreateAdminToken creates or updates an admin's API token per AI.md PART 11
func (s *Service) CreateAdminToken(adminID int64) (string, error) {
	// Generate token
	rawToken, err := generateRawToken(32)
	if err != nil {
		return "", err
	}

	// Add prefix
	fullToken := PrefixAdm + rawToken

	// Hash for storage
	tokenHash := hashToken(fullToken)

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	// Update admin's token hash
	result, err := s.db.ExecContext(ctx, `
		UPDATE admins SET api_token_hash = ?, updated_at = ? WHERE id = ?
	`, tokenHash, time.Now().Unix(), adminID)
	if err != nil {
		return "", err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return "", ErrTokenNotFound
	}

	return fullToken, nil
}

// RevokeAdminToken revokes an admin's API token per AI.md PART 11
func (s *Service) RevokeAdminToken(adminID int64) error {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	result, err := s.db.ExecContext(ctx, `
		UPDATE admins SET api_token_hash = NULL, updated_at = ? WHERE id = ?
	`, time.Now().Unix(), adminID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTokenNotFound
	}

	return nil
}

// updateLastUsed updates the last_used_at timestamp
// Table name is validated against whitelist per AI.md PART 11 (no SQL injection)
func (s *Service) updateLastUsed(table string, tokenID int64) {
	// Whitelist of allowed table names per AI.md PART 11
	var query string
	switch table {
	case "user_tokens":
		query = "UPDATE user_tokens SET last_used_at = ? WHERE id = ?"
	case "org_tokens":
		query = "UPDATE org_tokens SET last_used_at = ? WHERE id = ?"
	default:
		// Invalid table name - fail silently (don't expose error)
		return
	}

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	s.db.ExecContext(ctx, query, time.Now().Unix(), tokenID)
}

// RevokeUserToken revokes a user token
func (s *Service) RevokeUserToken(tokenID, userID int64) error {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	result, err := s.db.ExecContext(ctx, "DELETE FROM user_tokens WHERE id = ? AND user_id = ?", tokenID, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTokenNotFound
	}
	return nil
}

// RevokeOrgToken revokes an organization token
func (s *Service) RevokeOrgToken(tokenID, orgID int64) error {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	result, err := s.db.ExecContext(ctx, "DELETE FROM org_tokens WHERE id = ? AND org_id = ?", tokenID, orgID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTokenNotFound
	}
	return nil
}

// ListUserTokens returns all tokens for a user
func (s *Service) ListUserTokens(userID int64) ([]Token, error) {
	// List timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultListTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, token_prefix, scopes, last_used_at, expires_at, created_at
		FROM user_tokens WHERE user_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []Token
	for rows.Next() {
		var t Token
		var expiresAt, lastUsedAt sql.NullInt64

		err := rows.Scan(
			&t.ID, &t.OwnerID, &t.Name, &t.TokenPrefix,
			&t.Scopes, &lastUsedAt, &expiresAt, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if expiresAt.Valid {
			t.ExpiresAt = &expiresAt.Int64
		}
		if lastUsedAt.Valid {
			t.LastUsedAt = &lastUsedAt.Int64
		}

		tokens = append(tokens, t)
	}

	return tokens, nil
}

// ListOrgTokens returns all tokens for an organization
func (s *Service) ListOrgTokens(orgID int64) ([]Token, error) {
	// List timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultListTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, org_id, created_by, name, token_prefix, scopes, last_used_at, expires_at, created_at
		FROM org_tokens WHERE org_id = ? ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []Token
	for rows.Next() {
		var t Token
		var expiresAt, lastUsedAt sql.NullInt64

		err := rows.Scan(
			&t.ID, &t.OrgID, &t.CreatedBy, &t.Name, &t.TokenPrefix,
			&t.Scopes, &lastUsedAt, &expiresAt, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if expiresAt.Valid {
			t.ExpiresAt = &expiresAt.Int64
		}
		if lastUsedAt.Valid {
			t.LastUsedAt = &lastUsedAt.Int64
		}

		tokens = append(tokens, t)
	}

	return tokens, nil
}

// CountUserTokens returns the number of tokens a user has
func (s *Service) CountUserTokens(userID int64) (int, error) {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_tokens WHERE user_id = ?", userID).Scan(&count)
	return count, err
}

// CountOrgTokens returns the number of tokens an organization has
func (s *Service) CountOrgTokens(orgID int64) (int, error) {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM org_tokens WHERE org_id = ?", orgID).Scan(&count)
	return count, err
}

// CleanupExpired deletes all expired tokens from user_tokens and org_tokens tables.
func (s *Service) CleanupExpired() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, "DELETE FROM user_tokens WHERE expires_at IS NOT NULL AND expires_at < ?", now)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "DELETE FROM org_tokens WHERE expires_at IS NOT NULL AND expires_at < ?", now)
	return err
}

// generateRawToken generates a cryptographically secure random token
func generateRawToken(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashToken hashes a token using SHA-256
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// HasScope checks if a token has a specific scope
func (info *TokenInfo) HasScope(scope string) bool {
	// Global scope has all permissions
	for _, s := range info.Scopes {
		if s == ScopeGlobal || s == scope {
			return true
		}
	}
	return false
}

// CanWrite checks if the token has write permissions
func (info *TokenInfo) CanWrite() bool {
	return info.HasScope(ScopeGlobal) || info.HasScope(ScopeReadWrite)
}

// CanRead checks if the token has read permissions
func (info *TokenInfo) CanRead() bool {
	return info.HasScope(ScopeGlobal) || info.HasScope(ScopeReadWrite) || info.HasScope(ScopeRead)
}

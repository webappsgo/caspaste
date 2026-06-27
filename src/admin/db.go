
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/casjay-forks/caspaste/src/caspasswd"
)

const dbTimeout = 10 * time.Second

// adminRecord holds a row from the admins table
type adminRecord struct {
	ID           int64
	Username     string
	PasswordHash string
	Email        string
	Role         string
	Enabled      bool
	CreatedAt    int64
	LastLogin    sql.NullInt64
}

// CreateAdmin inserts a new admin with Argon2id-hashed password
func (p *Panel) CreateAdmin(username, password, email string) error {
	if p.db == nil {
		return errNoDB
	}
	hash, err := caspasswd.HashPassword(password)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	_, err = p.db.ExecContext(ctx,
		`INSERT INTO admins (username, password_hash, email, role, enabled)
		 VALUES (?, ?, ?, 'admin', 1)`,
		username, hash, email)
	return err
}

// getAdmin fetches a single enabled admin by username
func (p *Panel) getAdmin(username string) (*adminRecord, error) {
	if p.db == nil {
		return nil, errNoDB
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	a := &adminRecord{}
	var enabled int
	err := p.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, COALESCE(email,''), role, enabled,
		        created_at, last_login
		 FROM admins
		 WHERE username = ? AND enabled = 1`,
		username,
	).Scan(&a.ID, &a.Username, &a.PasswordHash, &a.Email, &a.Role,
		&enabled, &a.CreatedAt, &a.LastLogin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Enabled = enabled == 1
	return a, nil
}

// getAdminByID fetches a single admin by ID
func (p *Panel) getAdminByID(id int64) (*adminRecord, error) {
	if p.db == nil {
		return nil, errNoDB
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	a := &adminRecord{}
	var enabled int
	err := p.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, COALESCE(email,''), role, enabled,
		        created_at, last_login
		 FROM admins
		 WHERE id = ?`,
		id,
	).Scan(&a.ID, &a.Username, &a.PasswordHash, &a.Email, &a.Role,
		&enabled, &a.CreatedAt, &a.LastLogin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Enabled = enabled == 1
	return a, nil
}

// VerifyPassword checks username + password and returns the adminRecord on success
func (p *Panel) VerifyPassword(username, password string) (*adminRecord, error) {
	a, err := p.getAdmin(username)
	if err != nil || a == nil {
		return nil, err
	}
	// Use caspasswd.Data for Argon2id + bcrypt verification
	data := caspasswd.Data{a.Username: a.PasswordHash}
	if !data.Check(a.Username, password) {
		return nil, nil
	}
	return a, nil
}

// CountAdmins returns the number of enabled admin accounts
func (p *Panel) CountAdmins() (int, error) {
	if p.db == nil {
		return 0, errNoDB
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	var count int
	err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM admins WHERE enabled = 1`).Scan(&count)
	return count, err
}

// CountPastes returns the total number of pastes
func (p *Panel) CountPastes() (int64, error) {
	if p.db == nil {
		return 0, errNoDB
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	var count int64
	err := p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pastes`).Scan(&count)
	return count, err
}

// CountPastesRecent returns the number of pastes created in the last 24 hours
func (p *Panel) CountPastesRecent() (int64, error) {
	if p.db == nil {
		return 0, errNoDB
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	since := time.Now().Add(-24 * time.Hour).Unix()
	var count int64
	err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pastes WHERE create_time >= ?`, since).Scan(&count)
	return count, err
}

// updateLastLogin records the current time as last_login for the given admin
func (p *Panel) updateLastLogin(adminID int64) {
	if p.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	_, _ = p.db.ExecContext(ctx,
		`UPDATE admins SET last_login = ? WHERE id = ?`,
		time.Now().Unix(), adminID)
}

// tokenRecord holds a row from the admin_tokens table
type tokenRecord struct {
	ID         int64
	AdminID    int64
	Name       string
	Prefix     string
	CreatedAt  int64
	LastUsedAt sql.NullInt64
	ExpiresAt  sql.NullInt64
}

// listTokens returns all admin_tokens for display (no raw token values)
func (p *Panel) listTokens() ([]*tokenRecord, error) {
	if p.db == nil {
		return nil, errNoDB
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, admin_id, name, token_prefix, created_at, last_used_at, expires_at
		 FROM admin_tokens ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*tokenRecord
	for rows.Next() {
		t := &tokenRecord{}
		if err := rows.Scan(&t.ID, &t.AdminID, &t.Name, &t.Prefix,
			&t.CreatedAt, &t.LastUsedAt, &t.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// createToken generates a new API token for the given admin and returns the raw token (shown once)
func (p *Panel) createToken(adminID int64, name string, expireDays int) error {
	if p.db == nil {
		return errNoDB
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	rawHex := hex.EncodeToString(raw)
	prefix := "adm_" + rawHex[:8]
	h := sha256.Sum256([]byte(rawHex))
	tokenHash := hex.EncodeToString(h[:])

	var expiresAt interface{}
	if expireDays > 0 {
		expiresAt = time.Now().AddDate(0, 0, expireDays).Unix()
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO admin_tokens (admin_id, name, token_prefix, token_hash, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		adminID, name, prefix, tokenHash, expiresAt)
	return err
}

// revokeToken deletes an admin token by ID
func (p *Panel) revokeToken(tokenID int64) error {
	if p.db == nil {
		return errNoDB
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	_, err := p.db.ExecContext(ctx,
		`DELETE FROM admin_tokens WHERE id = ?`, tokenID)
	return err
}

// countOnlineAdmins returns the count of distinct admins with a currently valid session.
// Privacy: returns count only — no usernames exposed per AI.md PART 17.
func (p *Panel) countOnlineAdmins() (int, error) {
	if p.db == nil {
		return 0, errNoDB
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	var count int
	err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT admin_id) FROM admin_sessions WHERE expires_at > ?`,
		time.Now().Unix(),
	).Scan(&count)
	return count, err
}

// createAdminInvite generates a single-use invite token and inserts it.
// Returns the raw (unhashed) token — shown once to the inviting admin.
func (p *Panel) createAdminInvite(createdByID int64, expireHours int) (string, error) {
	if p.db == nil {
		return "", errNoDB
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	rawHex := hex.EncodeToString(raw)
	h := sha256.Sum256([]byte(rawHex))
	tokenHash := hex.EncodeToString(h[:])
	expiresAt := time.Now().Add(time.Duration(expireHours) * time.Hour).Unix()

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO admin_invites (token_hash, created_by, expires_at) VALUES (?, ?, ?)`,
		tokenHash, createdByID, expiresAt)
	if err != nil {
		return "", err
	}
	return rawHex, nil
}

// inviteRecord holds a row from the admin_invites table
type inviteRecord struct {
	ID        int64
	TokenHash string
	CreatedBy int64
	ExpiresAt int64
}

// getAdminInviteByToken looks up an active (unexpired, unaccepted) invite by its raw token.
// Returns nil if the token is invalid, expired, or already used.
func (p *Panel) getAdminInviteByToken(rawToken string) (*inviteRecord, error) {
	if p.db == nil {
		return nil, errNoDB
	}
	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	inv := &inviteRecord{}
	err := p.db.QueryRowContext(ctx,
		`SELECT id, token_hash, created_by, expires_at
		 FROM admin_invites
		 WHERE token_hash = ? AND expires_at > ? AND used_at IS NULL`,
		tokenHash, time.Now().Unix(),
	).Scan(&inv.ID, &inv.TokenHash, &inv.CreatedBy, &inv.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// acceptAdminInvite atomically marks the invite as used and creates the new admin account.
// Returns false if the invite was already consumed (concurrent acceptance attempt).
func (p *Panel) acceptAdminInvite(rawToken, username, password, email string) (bool, error) {
	if p.db == nil {
		return false, errNoDB
	}
	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])

	hash, err := caspasswd.HashPassword(password)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	now := time.Now().Unix()
	// Mark invite as used — conditional update prevents double-use
	result, err := p.db.ExecContext(ctx,
		`UPDATE admin_invites SET used_at = ? WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?`,
		now, tokenHash, now)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}

	_, err = p.db.ExecContext(ctx,
		`INSERT INTO admins (username, password_hash, email, role, enabled)
		 VALUES (?, ?, ?, 'admin', 1)`,
		username, hash, email)
	return err == nil, err
}

// CleanupExpiredSessions removes expired admin sessions
func (p *Panel) CleanupExpiredSessions() error {
	if p.db == nil {
		return errNoDB
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	_, err := p.db.ExecContext(ctx,
		`DELETE FROM admin_sessions WHERE expires_at < ?`, time.Now().Unix())
	return err
}

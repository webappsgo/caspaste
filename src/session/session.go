
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

// Query timeouts per AI.md PART 10
const (
	defaultQueryTimeout = 5 * time.Second
	defaultListTimeout  = 10 * time.Second
)

// Common errors
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrInvalidToken    = errors.New("invalid session token")
)

// Default session duration: 30 days
const DefaultSessionDuration = 30 * 24 * time.Hour

// Session represents a user session
type Session struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	TokenHash string `json:"-"`
	Device    string `json:"device,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	ExpiresAt int64  `json:"expires_at"`
	CreatedAt int64  `json:"created_at"`
}

// Service provides session operations
type Service struct {
	db       *sql.DB
	duration time.Duration
}

// NewService creates a new session service
func NewService(db *sql.DB) *Service {
	return &Service{
		db:       db,
		duration: DefaultSessionDuration,
	}
}

// SetDuration sets the session duration
func (s *Service) SetDuration(d time.Duration) {
	s.duration = d
}

// Create creates a new session for a user and returns the token
func (s *Service) Create(userID int64, device, ipAddress, userAgent string) (string, error) {
	// Generate random token
	token, err := generateToken(32)
	if err != nil {
		return "", err
	}

	// Hash the token for storage
	tokenHash := hashToken(token)

	now := time.Now().Unix()
	expiresAt := time.Now().Add(s.duration).Unix()

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO user_sessions (user_id, token_hash, device, ip_address, user_agent, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, tokenHash, device, ipAddress, userAgent, expiresAt, now)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Validate validates a session token and returns the session
func (s *Service) Validate(token string) (*Session, error) {
	if token == "" {
		return nil, ErrInvalidToken
	}

	tokenHash := hashToken(token)

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	session := &Session{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, device, ip_address, user_agent, expires_at, created_at
		FROM user_sessions WHERE token_hash = ?
	`, tokenHash).Scan(
		&session.ID, &session.UserID, &session.TokenHash,
		&session.Device, &session.IPAddress, &session.UserAgent,
		&session.ExpiresAt, &session.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	// Check if expired
	if session.ExpiresAt < time.Now().Unix() {
		// Delete expired session
		s.Delete(token)
		return nil, ErrSessionExpired
	}

	return session, nil
}

// GetUserID validates a token and returns the user ID
func (s *Service) GetUserID(token string) (int64, error) {
	session, err := s.Validate(token)
	if err != nil {
		return 0, err
	}
	return session.UserID, nil
}

// Delete deletes a session by token
func (s *Service) Delete(token string) error {
	tokenHash := hashToken(token)

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, "DELETE FROM user_sessions WHERE token_hash = ?", tokenHash)
	return err
}

// DeleteByID deletes a session by ID
func (s *Service) DeleteByID(sessionID int64) error {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, "DELETE FROM user_sessions WHERE id = ?", sessionID)
	return err
}

// DeleteAllForUser deletes all sessions for a user
func (s *Service) DeleteAllForUser(userID int64) error {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, "DELETE FROM user_sessions WHERE user_id = ?", userID)
	return err
}

// DeleteAllExcept deletes all sessions for a user except the current one
func (s *Service) DeleteAllExcept(userID int64, currentToken string) error {
	tokenHash := hashToken(currentToken)

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, "DELETE FROM user_sessions WHERE user_id = ? AND token_hash != ?", userID, tokenHash)
	return err
}

// ListForUser returns all sessions for a user
func (s *Service) ListForUser(userID int64) ([]Session, error) {
	// List timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultListTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, device, ip_address, user_agent, expires_at, created_at
		FROM user_sessions WHERE user_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		err := rows.Scan(
			&session.ID, &session.UserID, &session.Device,
			&session.IPAddress, &session.UserAgent, &session.ExpiresAt, &session.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// CleanupExpired removes all expired sessions
func (s *Service) CleanupExpired() (int64, error) {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	result, err := s.db.ExecContext(ctx, "DELETE FROM user_sessions WHERE expires_at < ?", time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Extend extends the session expiration
func (s *Service) Extend(token string) error {
	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(s.duration).Unix()

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, "UPDATE user_sessions SET expires_at = ? WHERE token_hash = ?", expiresAt, tokenHash)
	return err
}

// generateToken generates a cryptographically secure random token
func generateToken(length int) (string, error) {
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

// IsExpired checks if a session is expired
func (session *Session) IsExpired() bool {
	return session.ExpiresAt < time.Now().Unix()
}

// TimeUntilExpiry returns the duration until the session expires
func (session *Session) TimeUntilExpiry() time.Duration {
	return time.Until(time.Unix(session.ExpiresAt, 0))
}

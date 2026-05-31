
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package user

import (
	"context"
	cryptoRand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// Query timeouts per AI.md PART 10
const (
	defaultQueryTimeout = 5 * time.Second
)

// User role constants
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// Visibility constants
const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

// Avatar type constants
const (
	AvatarTypeGravatar = "gravatar"
	AvatarTypeUpload   = "upload"
	AvatarTypeURL      = "url"
)

// Common errors
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUsernameTaken      = errors.New("username is already taken")
	ErrEmailTaken         = errors.New("email is already taken")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrUsernameBlocked    = errors.New("username contains blocked word")
	ErrAccountLocked      = errors.New("account is locked")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// User represents a user account per PART 34
type User struct {
	ID            int64  `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email,omitempty"`
	PasswordHash  string `json:"-"`
	DisplayName   string `json:"display_name,omitempty"`
	AvatarType    string `json:"avatar_type"`
	AvatarURL     string `json:"avatar_url,omitempty"`
	Bio           string `json:"bio,omitempty"`
	Location      string `json:"location,omitempty"`
	Website       string `json:"website,omitempty"`
	Visibility    string `json:"visibility"`
	OrgVisibility bool   `json:"org_visibility"`
	Timezone      string `json:"timezone,omitempty"`
	Language      string `json:"language,omitempty"`
	Role          string `json:"role"`
	EmailVerified bool   `json:"email_verified"`
	TOTPEnabled   bool   `json:"totp_enabled"`
	TOTPSecret    string `json:"-"`
	LastLogin     int64  `json:"last_login,omitempty"`
	FailedAttempts int   `json:"-"`
	LockedUntil   int64  `json:"-"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

// PublicUser returns a user with only public fields
type PublicUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarType  string `json:"avatar_type"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Bio         string `json:"bio,omitempty"`
	Location    string `json:"location,omitempty"`
	Website     string `json:"website,omitempty"`
	Verified    bool   `json:"verified"`
	CreatedAt   int64  `json:"created_at"`
}

// CreateUserInput contains fields for creating a new user
type CreateUserInput struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
	Role        string
}

// UpdateUserInput contains fields for updating a user
type UpdateUserInput struct {
	DisplayName   *string
	AvatarType    *string
	AvatarURL     *string
	Bio           *string
	Location      *string
	Website       *string
	Visibility    *string
	OrgVisibility *bool
	Timezone      *string
	Language      *string
}

// Service provides user operations
type Service struct {
	db *sql.DB
}

// NewService creates a new user service
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Create creates a new user
func (s *Service) Create(input CreateUserInput) (*User, error) {
	// Validate input
	if err := ValidateUsername(input.Username); err != nil {
		return nil, err
	}
	if err := ValidateEmail(input.Email); err != nil {
		return nil, err
	}
	if err := ValidatePassword(input.Password); err != nil {
		return nil, err
	}

	// Check if username is taken
	existing, _ := s.GetByUsername(input.Username)
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	// Check if email is taken
	existing, _ = s.GetByEmail(input.Email)
	if existing != nil {
		return nil, ErrEmailTaken
	}

	// Hash password with Argon2id (per PART 11)
	passwordHash := HashPassword(input.Password)

	// Set defaults
	role := input.Role
	if role == "" {
		role = RoleUser
	}

	now := time.Now().Unix()

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	// Insert user
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO users (username, email, password_hash, display_name, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, strings.ToLower(input.Username), strings.ToLower(input.Email), passwordHash, input.DisplayName, role, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	return s.GetByID(id)
}

// GetByID retrieves a user by ID
func (s *Service) GetByID(id int64) (*User, error) {
	user := &User{}
	var orgVisibility int
	var emailVerified, totpEnabled int

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, avatar_type, avatar_url,
		       bio, location, website, visibility, org_visibility, timezone, language, role,
		       email_verified, totp_enabled, totp_secret, last_login, failed_attempts,
		       locked_until, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.DisplayName, &user.AvatarType, &user.AvatarURL,
		&user.Bio, &user.Location, &user.Website, &user.Visibility,
		&orgVisibility, &user.Timezone, &user.Language, &user.Role,
		&emailVerified, &totpEnabled, &user.TOTPSecret, &user.LastLogin,
		&user.FailedAttempts, &user.LockedUntil, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	user.OrgVisibility = orgVisibility == 1
	user.EmailVerified = emailVerified == 1
	user.TOTPEnabled = totpEnabled == 1

	return user, nil
}

// GetByUsername retrieves a user by username (case-insensitive)
func (s *Service) GetByUsername(username string) (*User, error) {
	user := &User{}
	var orgVisibility int
	var emailVerified, totpEnabled int

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, avatar_type, avatar_url,
		       bio, location, website, visibility, org_visibility, timezone, language, role,
		       email_verified, totp_enabled, totp_secret, last_login, failed_attempts,
		       locked_until, created_at, updated_at
		FROM users WHERE LOWER(username) = LOWER(?)
	`, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.DisplayName, &user.AvatarType, &user.AvatarURL,
		&user.Bio, &user.Location, &user.Website, &user.Visibility,
		&orgVisibility, &user.Timezone, &user.Language, &user.Role,
		&emailVerified, &totpEnabled, &user.TOTPSecret, &user.LastLogin,
		&user.FailedAttempts, &user.LockedUntil, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	user.OrgVisibility = orgVisibility == 1
	user.EmailVerified = emailVerified == 1
	user.TOTPEnabled = totpEnabled == 1

	return user, nil
}

// GetByEmail retrieves a user by email (case-insensitive)
func (s *Service) GetByEmail(email string) (*User, error) {
	user := &User{}
	var orgVisibility int
	var emailVerified, totpEnabled int

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, display_name, avatar_type, avatar_url,
		       bio, location, website, visibility, org_visibility, timezone, language, role,
		       email_verified, totp_enabled, totp_secret, last_login, failed_attempts,
		       locked_until, created_at, updated_at
		FROM users WHERE LOWER(email) = LOWER(?)
	`, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.DisplayName, &user.AvatarType, &user.AvatarURL,
		&user.Bio, &user.Location, &user.Website, &user.Visibility,
		&orgVisibility, &user.Timezone, &user.Language, &user.Role,
		&emailVerified, &totpEnabled, &user.TOTPSecret, &user.LastLogin,
		&user.FailedAttempts, &user.LockedUntil, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	user.OrgVisibility = orgVisibility == 1
	user.EmailVerified = emailVerified == 1
	user.TOTPEnabled = totpEnabled == 1

	return user, nil
}

// GetByIdentifier retrieves a user by ID, username, or email
func (s *Service) GetByIdentifier(identifier string) (*User, error) {
	identType := DetectIdentifierType(identifier)

	switch identType {
	case "user_id":
		var id int64
		fmt.Sscanf(identifier, "%d", &id)
		return s.GetByID(id)
	case "email":
		return s.GetByEmail(identifier)
	default:
		return s.GetByUsername(identifier)
	}
}

// Update updates a user's profile using individual parameterized queries
func (s *Service) Update(id int64, input UpdateUserInput) error {
	// Each field is updated with a fully parameterized query (no string concatenation)
	type fieldUpdate struct {
		query string
		value interface{}
	}

	var fields []fieldUpdate

	if input.DisplayName != nil {
		fields = append(fields, fieldUpdate{
			query: "UPDATE users SET display_name = ?, updated_at = ? WHERE id = ?",
			value: *input.DisplayName,
		})
	}
	if input.AvatarType != nil {
		fields = append(fields, fieldUpdate{
			query: "UPDATE users SET avatar_type = ?, updated_at = ? WHERE id = ?",
			value: *input.AvatarType,
		})
	}
	if input.AvatarURL != nil {
		fields = append(fields, fieldUpdate{
			query: "UPDATE users SET avatar_url = ?, updated_at = ? WHERE id = ?",
			value: *input.AvatarURL,
		})
	}
	if input.Bio != nil {
		fields = append(fields, fieldUpdate{
			query: "UPDATE users SET bio = ?, updated_at = ? WHERE id = ?",
			value: *input.Bio,
		})
	}
	if input.Location != nil {
		fields = append(fields, fieldUpdate{
			query: "UPDATE users SET location = ?, updated_at = ? WHERE id = ?",
			value: *input.Location,
		})
	}
	if input.Website != nil {
		fields = append(fields, fieldUpdate{
			query: "UPDATE users SET website = ?, updated_at = ? WHERE id = ?",
			value: *input.Website,
		})
	}
	if input.Visibility != nil {
		fields = append(fields, fieldUpdate{
			query: "UPDATE users SET visibility = ?, updated_at = ? WHERE id = ?",
			value: *input.Visibility,
		})
	}
	if input.OrgVisibility != nil {
		orgVis := 0
		if *input.OrgVisibility {
			orgVis = 1
		}
		fields = append(fields, fieldUpdate{
			query: "UPDATE users SET org_visibility = ?, updated_at = ? WHERE id = ?",
			value: orgVis,
		})
	}
	if input.Timezone != nil {
		fields = append(fields, fieldUpdate{
			query: "UPDATE users SET timezone = ?, updated_at = ? WHERE id = ?",
			value: *input.Timezone,
		})
	}
	if input.Language != nil {
		fields = append(fields, fieldUpdate{
			query: "UPDATE users SET language = ?, updated_at = ? WHERE id = ?",
			value: *input.Language,
		})
	}

	if len(fields) == 0 {
		return nil
	}

	now := time.Now().Unix()

	for _, f := range fields {
		// Query timeout per AI.md PART 10
		ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
		_, err := s.db.ExecContext(ctx, f.query, f.value, now, id)
		cancel()
		if err != nil {
			return err
		}
	}

	return nil
}

// Delete removes a user
func (s *Service) Delete(id int64) error {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	return err
}

// UpdatePassword updates a user's password
func (s *Service) UpdatePassword(id int64, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	passwordHash := HashPassword(newPassword)

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, "UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?",
		passwordHash, time.Now().Unix(), id)
	return err
}

// VerifyPassword checks if the provided password matches the user's hash
func (s *Service) VerifyPassword(user *User, password string) bool {
	return VerifyPassword(password, user.PasswordHash)
}

// Authenticate authenticates a user by identifier and password
func (s *Service) Authenticate(identifier, password string) (*User, error) {
	user, err := s.GetByIdentifier(identifier)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if account is locked
	if user.LockedUntil > 0 && user.LockedUntil > time.Now().Unix() {
		return nil, ErrAccountLocked
	}

	// Verify password
	if !VerifyPassword(password, user.PasswordHash) {
		// Increment failed attempts
		s.incrementFailedAttempts(user.ID)
		return nil, ErrInvalidCredentials
	}

	// Reset failed attempts on successful login
	s.resetFailedAttempts(user.ID)

	// Update last login
	s.updateLastLogin(user.ID)

	return user, nil
}

// incrementFailedAttempts increases failed login counter and locks if needed
func (s *Service) incrementFailedAttempts(userID int64) {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	// Get current failed attempts
	var failedAttempts int
	s.db.QueryRowContext(ctx, "SELECT failed_attempts FROM users WHERE id = ?", userID).Scan(&failedAttempts)

	failedAttempts++

	// Lock for 15 minutes after 5 failed attempts (per PART 34)
	var lockedUntil int64
	if failedAttempts >= 5 {
		lockedUntil = time.Now().Add(15 * time.Minute).Unix()
	}

	s.db.ExecContext(ctx, "UPDATE users SET failed_attempts = ?, locked_until = ? WHERE id = ?",
		failedAttempts, lockedUntil, userID)
}

// resetFailedAttempts clears the failed attempts counter
func (s *Service) resetFailedAttempts(userID int64) {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	s.db.ExecContext(ctx, "UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE id = ?", userID)
}

// updateLastLogin sets the last login timestamp
func (s *Service) updateLastLogin(userID int64) {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	s.db.ExecContext(ctx, "UPDATE users SET last_login = ? WHERE id = ?", time.Now().Unix(), userID)
}

// SetEmailVerified marks a user's email as verified
func (s *Service) SetEmailVerified(userID int64, verified bool) error {
	v := 0
	if verified {
		v = 1
	}

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, "UPDATE users SET email_verified = ?, updated_at = ? WHERE id = ?",
		v, time.Now().Unix(), userID)
	return err
}

// SetTOTPEnabled enables or disables TOTP for a user
func (s *Service) SetTOTPEnabled(userID int64, enabled bool, secret string) error {
	v := 0
	if enabled {
		v = 1
	}

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, "UPDATE users SET totp_enabled = ?, totp_secret = ?, updated_at = ? WHERE id = ?",
		v, secret, time.Now().Unix(), userID)
	return err
}

// ToPublic converts a user to public representation
func (u *User) ToPublic() PublicUser {
	return PublicUser{
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarType:  u.AvatarType,
		AvatarURL:   u.AvatarURL,
		Bio:         u.Bio,
		Location:    u.Location,
		Website:     u.Website,
		Verified:    u.EmailVerified,
		CreatedAt:   u.CreatedAt,
	}
}

// DetectIdentifierType determines if input is user_id, email, or username
func DetectIdentifierType(input string) string {
	// Numeric only = User ID
	if regexp.MustCompile(`^\d+$`).MatchString(input) {
		return "user_id"
	}
	// Contains @ = Email
	if strings.Contains(input, "@") {
		return "email"
	}
	// Otherwise = Username
	return "username"
}

// HashPassword hashes a password using Argon2id (per PART 11)
func HashPassword(password string) string {
	// Argon2id parameters (OWASP recommended)
	salt := generateSalt()
	time := uint32(3)
	// 64 MB
	memory := uint32(64 * 1024)
	threads := uint8(4)
	keyLen := uint32(32)

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	// Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	b64Salt := hex.EncodeToString(salt)
	b64Hash := hex.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, time, threads, b64Salt, b64Hash)
}

// VerifyPassword verifies a password against an Argon2id hash
func VerifyPassword(password, encodedHash string) bool {
	// Parse the encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	// Parse parameters
	var memory, time uint32
	var threads uint8
	fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)

	// Decode salt and hash
	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		return false
	}
	storedHash, err := hex.DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Compute hash with same parameters
	computedHash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(storedHash)))

	// Constant-time comparison
	if len(storedHash) != len(computedHash) {
		return false
	}
	var result byte
	for i := 0; i < len(storedHash); i++ {
		result |= storedHash[i] ^ computedHash[i]
	}
	return result == 0
}

// generateSalt generates a cryptographically secure random salt
func generateSalt() []byte {
	salt := make([]byte, 16)
	// Use crypto/rand for cryptographically secure randomness per AI.md PART 11
	if _, err := cryptoRand.Read(salt); err != nil {
		// Fallback should never happen, but panic if it does - this is critical
		panic(fmt.Sprintf("failed to generate secure salt: %v", err))
	}
	return salt
}

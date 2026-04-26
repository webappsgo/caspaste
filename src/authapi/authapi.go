
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package authapi provides authentication API handlers per PART 34
package authapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/casjay-forks/caspaste/src/config"
	"github.com/casjay-forks/caspaste/src/email"
	"github.com/casjay-forks/caspaste/src/httputil"
	"github.com/casjay-forks/caspaste/src/recovery"
	"github.com/casjay-forks/caspaste/src/session"
	"github.com/casjay-forks/caspaste/src/totp"
	"github.com/casjay-forks/caspaste/src/user"
	"github.com/casjay-forks/caspaste/src/web"
)

// Database query timeout
const defaultQueryTimeout = 5 * time.Second

// Common errors
var (
	ErrRegistrationDisabled = errors.New("registration is disabled")
	ErrInviteRequired       = errors.New("invite required for registration")
	ErrInvalidInvite        = errors.New("invalid or expired invite")
	ErrEmailNotVerified     = errors.New("email not verified")
	ErrTOTPRequired         = errors.New("2FA code required")
	ErrInvalidTOTP          = errors.New("invalid 2FA code")
)

// Service provides authentication API operations
type Service struct {
	db              *sql.DB
	userService     *user.Service
	sessionService  *session.Service
	recoveryService *recovery.Service
	config          *config.UsersConfig
	emailClient     *email.Client
	serverFQDN      string
}

// NewService creates a new auth API service
func NewService(db *sql.DB, userSvc *user.Service, sessSvc *session.Service, recoverySvc *recovery.Service, cfg *config.UsersConfig, emailCli *email.Client, fqdn string) *Service {
	return &Service{
		db:              db,
		userService:     userSvc,
		sessionService:  sessSvc,
		recoveryService: recoverySvc,
		config:          cfg,
		emailClient:     emailCli,
		serverFQDN:      fqdn,
	}
}

// RegisterRequest is the request body for registration
type RegisterRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name,omitempty"`
	InviteCode  string `json:"invite_code,omitempty"`
}

// LoginRequest is the request body for login
type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
	TOTPCode   string `json:"totp_code,omitempty"`
	Remember   bool   `json:"remember,omitempty"`
}

// PasswordResetRequest is the request body for password reset
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordResetConfirmRequest is the request body for confirming password reset
type PasswordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// RecoveryUseRequest is the request body for using a recovery key
type RecoveryUseRequest struct {
	Identifier  string `json:"identifier"`
	RecoveryKey string `json:"recovery_key"`
}

// APIResponse is the unified response format per PART 16
type APIResponse struct {
	OK      bool        `json:"ok"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// AuthResponse is the response for successful authentication
type AuthResponse struct {
	User         *user.User `json:"user"`
	SessionToken string     `json:"session_token,omitempty"`
	ExpiresAt    int64      `json:"expires_at,omitempty"`
	RequiresTOTP bool       `json:"requires_totp,omitempty"`
}

// HandleRegister handles POST /api/v1/auth/register
func (s *Service) HandleRegister(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	// Check if registration is enabled
	if s.config == nil || s.config.Registration.Mode == "disabled" {
		return writeError(w, r, http.StatusForbidden, "REGISTRATION_DISABLED", "Registration is disabled")
	}

	// Parse request
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	// Validate required fields
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_FIELDS", "Username, email, and password are required")
	}

	// Check if invite is required for private registration
	if s.config.Registration.Mode == "private" {
		if req.InviteCode == "" {
			return writeError(w, r, http.StatusForbidden, "INVITE_REQUIRED", "An invite code is required for registration")
		}
		// Verify invite code
		valid, err := s.verifyInvite(req.InviteCode)
		if err != nil || !valid {
			return writeError(w, r, http.StatusForbidden, "INVALID_INVITE", "Invalid or expired invite code")
		}
	}

	// Check email domain restrictions
	if len(s.config.Registration.AllowedDomains) > 0 {
		domain := getEmailDomain(req.Email)
		if !containsDomain(s.config.Registration.AllowedDomains, domain) {
			return writeError(w, r, http.StatusForbidden, "EMAIL_DOMAIN_NOT_ALLOWED", "Email domain not allowed")
		}
	}
	if len(s.config.Registration.BlockedDomains) > 0 {
		domain := getEmailDomain(req.Email)
		if containsDomain(s.config.Registration.BlockedDomains, domain) {
			return writeError(w, r, http.StatusForbidden, "EMAIL_DOMAIN_BLOCKED", "Email domain is blocked")
		}
	}

	// Create user
	input := user.CreateUserInput{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		Role:        s.config.Roles.Default,
	}

	newUser, err := s.userService.Create(input)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrUsernameTaken):
			return writeError(w, r, http.StatusConflict, "USERNAME_TAKEN", "Username is already taken")
		case errors.Is(err, user.ErrEmailTaken):
			return writeError(w, r, http.StatusConflict, "EMAIL_TAKEN", "Email is already registered")
		case errors.Is(err, user.ErrInvalidUsername):
			return writeError(w, r, http.StatusBadRequest, "INVALID_USERNAME", "Invalid username format")
		case errors.Is(err, user.ErrUsernameBlocked):
			return writeError(w, r, http.StatusBadRequest, "USERNAME_BLOCKED", "This username is not allowed")
		case errors.Is(err, user.ErrInvalidEmail):
			return writeError(w, r, http.StatusBadRequest, "INVALID_EMAIL", "Invalid email format")
		case errors.Is(err, user.ErrInvalidPassword):
			return writeError(w, r, http.StatusBadRequest, "INVALID_PASSWORD", "Password does not meet requirements")
		default:
			return writeError(w, r, http.StatusInternalServerError, "REGISTRATION_FAILED", "Registration failed")
		}
	}

	// Mark invite as used if applicable
	if req.InviteCode != "" {
		s.markInviteUsed(req.InviteCode)
	}

	// If email verification is required, don't create session yet
	if s.config.Registration.RequireEmailVerification {
		// Generate verification token
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err == nil {
			tokenHash := sha256.Sum256(tokenBytes)
			verificationToken := hex.EncodeToString(tokenHash[:])

			// Store token for the user
			s.storeEmailVerificationToken(newUser.ID, verificationToken)

			// Send verification email if email client is available
			if s.emailClient != nil && s.emailClient.IsEnabled() {
				verifyURL := fmt.Sprintf("https://%s/auth/verify-email?token=%s", s.serverFQDN, hex.EncodeToString(tokenBytes))
				emailBody := fmt.Sprintf(
					"Welcome to CasPaste!\n\nPlease verify your email address by visiting:\n%s\n\nIf you did not create this account, you can ignore this email.",
					verifyURL,
				)
				s.emailClient.Send(newUser.Email, "Verify your email - CasPaste", emailBody)
			}
		}

		return writeSuccess(w, r, map[string]interface{}{
			"user":                  newUser,
			"email_verification":    true,
			"verification_required": true,
		}, "Registration successful", "User registered. Please verify your email.")
	}

	// Create session
	sessionToken, err := s.sessionService.Create(
		newUser.ID,
		getDeviceInfo(r),
		getClientIP(r),
		r.UserAgent(),
	)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "SESSION_ERROR", "Failed to create session")
	}

	// Set session cookie
	setSessionCookie(w, r, sessionToken)

	return writeSuccess(w, r, AuthResponse{
		User:         newUser,
		SessionToken: sessionToken,
		ExpiresAt:    time.Now().Add(session.DefaultSessionDuration).Unix(),
	}, "Registration successful", "User registered successfully")
}

// HandleLogin handles POST /api/v1/auth/login
func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	// Parse request
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	// Validate required fields
	if req.Identifier == "" || req.Password == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_FIELDS", "Identifier and password are required")
	}

	// Authenticate user
	authUser, err := s.userService.Authenticate(req.Identifier, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrAccountLocked):
			return writeError(w, r, http.StatusForbidden, "ACCOUNT_LOCKED", "Account is temporarily locked. Try again later.")
		case errors.Is(err, user.ErrInvalidCredentials):
			return writeError(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials")
		default:
			return writeError(w, r, http.StatusUnauthorized, "LOGIN_FAILED", "Login failed")
		}
	}

	// Check if 2FA is required
	if authUser.TOTPEnabled {
		if req.TOTPCode == "" {
			// Return response indicating 2FA is required
			return writeSuccess(w, r, AuthResponse{
				RequiresTOTP: true,
			}, "2FA required", "Please provide your 2FA code")
		}

		// Verify TOTP code
		if !totp.Verify(authUser.TOTPSecret, req.TOTPCode) {
			return writeError(w, r, http.StatusUnauthorized, "INVALID_TOTP", "Invalid 2FA code")
		}
	}

	// Check email verification if required
	if s.config != nil && s.config.Registration.RequireEmailVerification && !authUser.EmailVerified {
		return writeError(w, r, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Please verify your email before logging in")
	}

	// Create session
	sessionToken, err := s.sessionService.Create(
		authUser.ID,
		getDeviceInfo(r),
		getClientIP(r),
		r.UserAgent(),
	)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "SESSION_ERROR", "Failed to create session")
	}

	// Set session cookie
	setSessionCookie(w, r, sessionToken)

	return writeSuccess(w, r, AuthResponse{
		User:         authUser,
		SessionToken: sessionToken,
		ExpiresAt:    time.Now().Add(session.DefaultSessionDuration).Unix(),
	}, "Login successful", "Logged in successfully")
}

// HandleLogout handles POST /api/v1/auth/logout
func (s *Service) HandleLogout(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	// Get session token from cookie or header
	sessionToken := getSessionToken(r)
	if sessionToken == "" {
		return writeError(w, r, http.StatusUnauthorized, "NOT_LOGGED_IN", "Not logged in")
	}

	// Delete session
	s.sessionService.Delete(sessionToken)

	// Clear session cookie
	clearSessionCookie(w, r)

	return writeSuccess(w, r, nil, "Logout successful", "Logged out successfully")
}

// HandlePasswordForgot handles POST /api/v1/auth/password/forgot
func (s *Service) HandlePasswordForgot(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	var req PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.Email == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_EMAIL", "Email is required")
	}

	// Always return success to prevent email enumeration
	// In background, check if user exists and send reset email
	go func() {
		u, err := s.userService.GetByEmail(req.Email)
		if err == nil && u != nil {
			token, err := s.createPasswordResetToken(u.ID)
			if err != nil {
				return
			}

			// Send password reset email if email client is available
			if s.emailClient != nil && s.emailClient.IsEnabled() {
				resetURL := fmt.Sprintf("https://%s/auth/password/reset?token=%s", s.serverFQDN, token)
				emailBody := fmt.Sprintf(
					"A password reset was requested for your CasPaste account.\n\nReset your password by visiting:\n%s\n\nThis link expires in 1 hour.\n\nIf you did not request this, you can ignore this email.",
					resetURL,
				)
				s.emailClient.Send(u.Email, "Password Reset - CasPaste", emailBody)
			}
		}
	}()

	return writeSuccess(w, r, nil, "Password reset requested",
		"If an account with this email exists, a password reset link has been sent")
}

// HandlePasswordReset handles POST /api/v1/auth/password/reset
func (s *Service) HandlePasswordReset(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	var req PasswordResetConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.Token == "" || req.NewPassword == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_FIELDS", "Token and new password are required")
	}

	// Verify reset token
	userID, err := s.verifyPasswordResetToken(req.Token)
	if err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_TOKEN", "Invalid or expired reset token")
	}

	// Update password
	if err := s.userService.UpdatePassword(userID, req.NewPassword); err != nil {
		if errors.Is(err, user.ErrInvalidPassword) {
			return writeError(w, r, http.StatusBadRequest, "INVALID_PASSWORD", "Password does not meet requirements")
		}
		return writeError(w, r, http.StatusInternalServerError, "RESET_FAILED", "Failed to reset password")
	}

	// Mark token as used
	s.markPasswordResetUsed(req.Token)

	// Invalidate all existing sessions for security
	s.sessionService.DeleteAllForUser(userID)

	return writeSuccess(w, r, nil, "Password reset successful", "Password has been reset. Please log in with your new password.")
}

// HandleVerifyEmail handles GET /api/v1/auth/verify-email
func (s *Service) HandleVerifyEmail(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_TOKEN", "Verification token is required")
	}

	// Verify email token
	userID, err := s.verifyEmailToken(token)
	if err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_TOKEN", "Invalid or expired verification token")
	}

	// Mark email as verified
	if err := s.userService.SetEmailVerified(userID, true); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "VERIFICATION_FAILED", "Failed to verify email")
	}

	// Mark token as used
	s.markEmailVerificationUsed(token)

	return writeSuccess(w, r, nil, "Email verified", "Your email has been verified")
}

// HandleRecoveryUse handles POST /api/v1/auth/recovery/use
func (s *Service) HandleRecoveryUse(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	var req RecoveryUseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.Identifier == "" || req.RecoveryKey == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_FIELDS", "Identifier and recovery key are required")
	}

	// Get user
	u, err := s.userService.GetByIdentifier(req.Identifier)
	if err != nil {
		return writeError(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials")
	}

	// Verify and consume recovery key
	if err := s.recoveryService.VerifyAndConsumeKey(u.ID, req.RecoveryKey); err != nil {
		switch {
		case errors.Is(err, recovery.ErrKeyNotFound):
			return writeError(w, r, http.StatusUnauthorized, "INVALID_RECOVERY_KEY", "Invalid recovery key")
		case errors.Is(err, recovery.ErrKeyAlreadyUsed):
			return writeError(w, r, http.StatusBadRequest, "KEY_ALREADY_USED", "This recovery key has already been used")
		case errors.Is(err, recovery.ErrInvalidKeyFormat):
			return writeError(w, r, http.StatusBadRequest, "INVALID_KEY_FORMAT", "Invalid recovery key format")
		default:
			return writeError(w, r, http.StatusInternalServerError, "RECOVERY_FAILED", "Recovery failed")
		}
	}

	// Disable 2FA after successful recovery
	if err := s.userService.SetTOTPEnabled(u.ID, false, ""); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "RECOVERY_FAILED", "Failed to disable 2FA")
	}

	// Create session
	sessionToken, err := s.sessionService.Create(
		u.ID,
		getDeviceInfo(r),
		getClientIP(r),
		r.UserAgent(),
	)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "SESSION_ERROR", "Failed to create session")
	}

	// Set session cookie
	setSessionCookie(w, r, sessionToken)

	// Get remaining keys count
	remaining, _ := s.recoveryService.CountRemainingKeys(u.ID)

	return writeSuccess(w, r, map[string]interface{}{
		"user":           u,
		"session_token":  sessionToken,
		"2fa_disabled":   true,
		"remaining_keys": remaining,
	}, "Recovery successful", "2FA has been disabled. Please set up 2FA again.")
}

// HandleInviteGet handles GET /api/v1/auth/invite/{token}
func (s *Service) HandleInviteGet(w http.ResponseWriter, r *http.Request, token string) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	// Verify invite token
	invite, err := s.getInvite(token)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "INVALID_INVITE", "Invalid or expired invite")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"username":   invite.Username,
		"expires_at": invite.ExpiresAt,
	}, "Invite valid", "Invite is valid")
}

// Invite represents an invitation
type Invite struct {
	Username  string
	ExpiresAt int64
}

// Helper functions

func (s *Service) verifyInvite(code string) (bool, error) {
	var usedAt sql.NullInt64
	var expiresAt int64

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	err := s.db.QueryRowContext(ctx, `
		SELECT expires_at, used_at FROM user_invites WHERE token_hash = ?
	`, hashToken(code)).Scan(&expiresAt, &usedAt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if usedAt.Valid {
		return false, nil
	}
	if expiresAt < time.Now().Unix() {
		return false, nil
	}

	return true, nil
}

func (s *Service) markInviteUsed(code string) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()
	s.db.ExecContext(ctx, "UPDATE user_invites SET used_at = ? WHERE token_hash = ?", time.Now().Unix(), hashToken(code))
}

func (s *Service) getInvite(token string) (*Invite, error) {
	var invite Invite
	var usedAt sql.NullInt64

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	err := s.db.QueryRowContext(ctx, `
		SELECT username, expires_at, used_at FROM user_invites WHERE token_hash = ?
	`, hashToken(token)).Scan(&invite.Username, &invite.ExpiresAt, &usedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("invite not found")
	}
	if err != nil {
		return nil, err
	}

	if usedAt.Valid {
		return nil, errors.New("invite already used")
	}
	if invite.ExpiresAt < time.Now().Unix() {
		return nil, errors.New("invite expired")
	}

	return &invite, nil
}

func (s *Service) createPasswordResetToken(userID int64) (string, error) {
	token := generateToken(32)
	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(1 * time.Hour).Unix()

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO password_resets (user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, userID, tokenHash, expiresAt, time.Now().Unix())
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) verifyPasswordResetToken(token string) (int64, error) {
	var userID int64
	var usedAt sql.NullInt64
	var expiresAt int64

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, expires_at, used_at FROM password_resets WHERE token_hash = ?
	`, hashToken(token)).Scan(&userID, &expiresAt, &usedAt)
	if err == sql.ErrNoRows {
		return 0, errors.New("token not found")
	}
	if err != nil {
		return 0, err
	}

	if usedAt.Valid {
		return 0, errors.New("token already used")
	}
	if expiresAt < time.Now().Unix() {
		return 0, errors.New("token expired")
	}

	return userID, nil
}

func (s *Service) markPasswordResetUsed(token string) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()
	s.db.ExecContext(ctx, "UPDATE password_resets SET used_at = ? WHERE token_hash = ?", time.Now().Unix(), hashToken(token))
}

func (s *Service) verifyEmailToken(token string) (int64, error) {
	var userID int64
	var verifiedAt sql.NullInt64
	var expiresAt int64

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, expires_at, verified_at FROM email_verifications WHERE token_hash = ?
	`, hashToken(token)).Scan(&userID, &expiresAt, &verifiedAt)
	if err == sql.ErrNoRows {
		return 0, errors.New("token not found")
	}
	if err != nil {
		return 0, err
	}

	if verifiedAt.Valid {
		return 0, errors.New("already verified")
	}
	if expiresAt < time.Now().Unix() {
		return 0, errors.New("token expired")
	}

	return userID, nil
}

func (s *Service) markEmailVerificationUsed(token string) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()
	s.db.ExecContext(ctx, "UPDATE email_verifications SET verified_at = ? WHERE token_hash = ?", time.Now().Unix(), hashToken(token))
}

func (s *Service) storeEmailVerificationToken(userID int64, tokenHash string) {
	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()
	s.db.ExecContext(ctx, `
		INSERT INTO email_verifications (user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, userID, tokenHash, expiresAt, time.Now().Unix())
}

// Response helpers

func writeSuccess(w http.ResponseWriter, r *http.Request, data interface{}, textMsg string, textData string) error {
	format := httputil.GetAPIResponseFormat(r)

	switch format {
	case httputil.FormatText:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if textMsg != "" {
			fmt.Fprintf(w, "OK: %s\n", textMsg)
		}
		if textData != "" {
			fmt.Fprint(w, textData)
			if textData[len(textData)-1] != '\n' {
				fmt.Fprint(w, "\n")
			}
		}
		return nil
	default:
		return writeJSON(w, APIResponse{
			OK:   true,
			Data: data,
		})
	}
}

func writeError(w http.ResponseWriter, r *http.Request, code int, errCode, message string) error {
	format := httputil.GetAPIResponseFormat(r)

	w.WriteHeader(code)

	switch format {
	case httputil.FormatText:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "ERROR: %s: %s\n", errCode, message)
	default:
		w.Header().Set("Content-Type", "application/json")
		resp := APIResponse{
			OK:      false,
			Error:   errCode,
			Message: message,
		}
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		w.Write(jsonData)
		w.Write([]byte("\n"))
	}

	return nil
}

func writeJSON(w http.ResponseWriter, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}

// Utility functions

func getEmailDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

func containsDomain(domains []string, domain string) bool {
	for _, d := range domains {
		if strings.ToLower(d) == domain {
			return true
		}
	}
	return false
}

func getDeviceInfo(r *http.Request) string {
	ua := r.UserAgent()
	if ua == "" {
		return "Unknown"
	}
	// Simple extraction - in production use a proper UA parser
	if strings.Contains(ua, "Mobile") {
		return "Mobile"
	}
	if strings.Contains(ua, "Tablet") {
		return "Tablet"
	}
	return "Desktop"
}

func getClientIP(r *http.Request) string {
	// Check common proxy headers
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// Return first IP in chain
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Strip port
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

func getSessionToken(r *http.Request) string {
	// Check Authorization header first
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Check cookie
	cookie, err := r.Cookie("session")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Check context (set by middleware)
	if token := web.GetSessionToken(r.Context()); token != "" {
		return token
	}

	return ""
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	// Determine if we should use secure cookie
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(session.DefaultSessionDuration.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func generateToken(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

func hashToken(token string) string {
	// Use crypto/sha256 for secure token hashing
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

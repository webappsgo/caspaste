
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package userapi provides user profile and settings API handlers per PART 34
package userapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/casjay-forks/caspaste/src/config"
	"github.com/casjay-forks/caspaste/src/httputil"
	"github.com/casjay-forks/caspaste/src/recovery"
	"github.com/casjay-forks/caspaste/src/session"
	"github.com/casjay-forks/caspaste/src/token"
	"github.com/casjay-forks/caspaste/src/totp"
	"github.com/casjay-forks/caspaste/src/user"
	"github.com/casjay-forks/caspaste/src/web"
)

// Database query timeout
const defaultQueryTimeout = 5 * time.Second

// Service provides user API operations
type Service struct {
	db              *sql.DB
	userService     *user.Service
	sessionService  *session.Service
	tokenService    *token.Service
	recoveryService *recovery.Service
	config          *config.UsersConfig
}

// NewService creates a new user API service
func NewService(db *sql.DB, userSvc *user.Service, sessSvc *session.Service, tokenSvc *token.Service, recoverySvc *recovery.Service, cfg *config.UsersConfig) *Service {
	return &Service{
		db:              db,
		userService:     userSvc,
		sessionService:  sessSvc,
		tokenService:    tokenSvc,
		recoveryService: recoverySvc,
		config:          cfg,
	}
}

// APIResponse is the unified response format per PART 16
type APIResponse struct {
	OK      bool        `json:"ok"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// UpdateProfileRequest is the request body for updating user profile
type UpdateProfileRequest struct {
	DisplayName   *string `json:"display_name,omitempty"`
	Bio           *string `json:"bio,omitempty"`
	Location      *string `json:"location,omitempty"`
	Website       *string `json:"website,omitempty"`
	Visibility    *string `json:"visibility,omitempty"`
	OrgVisibility *bool   `json:"org_visibility,omitempty"`
	Timezone      *string `json:"timezone,omitempty"`
	Language      *string `json:"language,omitempty"`
}

// UpdatePasswordRequest is the request body for changing password
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// Enable2FARequest is the request body for enabling 2FA
type Enable2FARequest struct {
	TOTPCode string `json:"totp_code"`
	Secret   string `json:"secret"`
}

// Disable2FARequest is the request body for disabling 2FA
type Disable2FARequest struct {
	Password string `json:"password"`
	TOTPCode string `json:"totp_code,omitempty"`
}

// CreateTokenRequest is the request body for creating an API token
type CreateTokenRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes,omitempty"`
	ExpiresIn int64    `json:"expires_in,omitempty"`
}

// UserPreferences represents user preferences
type UserPreferences struct {
	ShowEmail      bool   `json:"show_email"`
	ShowActivity   bool   `json:"show_activity"`
	ShowOrgs       bool   `json:"show_orgs"`
	Searchable     bool   `json:"searchable"`
	EmailSecurity  bool   `json:"email_security"`
	EmailMentions  bool   `json:"email_mentions"`
	EmailUpdates   bool   `json:"email_updates"`
	EmailDigest    string `json:"email_digest"`
	Theme          string `json:"theme"`
	FontSize       string `json:"font_size"`
	ReduceMotion   bool   `json:"reduce_motion"`
	DateFormat     string `json:"date_format"`
	TimeFormat     string `json:"time_format"`
}

// HandleGetCurrentUser handles GET /api/v1/users
func (s *Service) HandleGetCurrentUser(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	// Get authenticated user from context
	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// Get full user data
	u, err := s.userService.GetByID(authUser.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "USER_NOT_FOUND", "User not found")
	}

	return writeSuccess(w, r, u, "User retrieved", fmt.Sprintf("Username: %s\nEmail: %s", u.Username, u.Email))
}

// HandleUpdateUser handles PATCH /api/v1/users
func (s *Service) HandleUpdateUser(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPatch {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	// Validate fields
	if req.Bio != nil && len(*req.Bio) > 500 {
		return writeError(w, r, http.StatusBadRequest, "BIO_TOO_LONG", "Bio must be 500 characters or less")
	}
	if req.Visibility != nil && *req.Visibility != "public" && *req.Visibility != "private" {
		return writeError(w, r, http.StatusBadRequest, "INVALID_VISIBILITY", "Visibility must be 'public' or 'private'")
	}

	// Update user
	input := user.UpdateUserInput{
		DisplayName:   req.DisplayName,
		Bio:           req.Bio,
		Location:      req.Location,
		Website:       req.Website,
		Visibility:    req.Visibility,
		OrgVisibility: req.OrgVisibility,
		Timezone:      req.Timezone,
		Language:      req.Language,
	}

	if err := s.userService.Update(authUser.ID, input); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update profile")
	}

	// Get updated user
	u, _ := s.userService.GetByID(authUser.ID)

	return writeSuccess(w, r, u, "Profile updated", "Profile updated successfully")
}

// HandleGetSettings handles GET /api/v1/users/settings
func (s *Service) HandleGetSettings(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// Get preferences
	prefs, err := s.getPreferences(authUser.ID)
	if err != nil {
		// Return defaults if no preferences exist
		prefs = getDefaultPreferences()
	}

	return writeSuccess(w, r, prefs, "Settings retrieved", "")
}

// HandleUpdateSettings handles PATCH /api/v1/users/settings
func (s *Service) HandleUpdateSettings(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPatch {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var prefs UserPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	// Update preferences
	if err := s.updatePreferences(authUser.ID, prefs); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update settings")
	}

	return writeSuccess(w, r, prefs, "Settings updated", "Settings updated successfully")
}

// HandleGetSecurity handles GET /api/v1/users/security
func (s *Service) HandleGetSecurity(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// Get full user to check TOTP status
	u, err := s.userService.GetByID(authUser.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "USER_NOT_FOUND", "User not found")
	}

	// Get sessions
	sessions, _ := s.sessionService.ListForUser(authUser.ID)

	// Get recovery keys status
	keysStatus, _ := s.recoveryService.GetKeysStatus(authUser.ID)

	return writeSuccess(w, r, map[string]interface{}{
		"totp_enabled":   u.TOTPEnabled,
		"email_verified": u.EmailVerified,
		"sessions":       sessions,
		"recovery_keys":  keysStatus,
	}, "Security info", "")
}

// HandleEnable2FA handles POST /api/v1/users/security/2fa/enable
func (s *Service) HandleEnable2FA(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// Check if 2FA is already enabled
	u, _ := s.userService.GetByID(authUser.ID)
	if u.TOTPEnabled {
		return writeError(w, r, http.StatusBadRequest, "2FA_ALREADY_ENABLED", "2FA is already enabled")
	}

	var req Enable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	// If no secret provided, generate one
	if req.Secret == "" {
		setup, err := totp.GenerateSecret("CasPaste", u.Email)
		if err != nil {
			return writeError(w, r, http.StatusInternalServerError, "TOTP_ERROR", "Failed to generate 2FA secret")
		}

		return writeSuccess(w, r, map[string]interface{}{
			"secret": setup.Secret,
			"qr_url": setup.QRCodeURL,
		}, "2FA setup", "Scan the QR code with your authenticator app")
	}

	// Verify TOTP code
	if !totp.Verify(req.Secret, req.TOTPCode) {
		return writeError(w, r, http.StatusBadRequest, "INVALID_TOTP", "Invalid 2FA code")
	}

	// Enable 2FA
	if err := s.userService.SetTOTPEnabled(authUser.ID, true, req.Secret); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "2FA_ENABLE_FAILED", "Failed to enable 2FA")
	}

	// Generate recovery keys
	keys, err := s.recoveryService.GenerateKeys(authUser.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "RECOVERY_ERROR", "Failed to generate recovery keys")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"enabled":       true,
		"recovery_keys": keys,
	}, "2FA enabled", "2FA has been enabled. Save your recovery keys in a safe place.")
}

// HandleDisable2FA handles POST /api/v1/users/security/2fa/disable
func (s *Service) HandleDisable2FA(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	u, _ := s.userService.GetByID(authUser.ID)
	if !u.TOTPEnabled {
		return writeError(w, r, http.StatusBadRequest, "2FA_NOT_ENABLED", "2FA is not enabled")
	}

	var req Disable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	// Verify password
	if !s.userService.VerifyPassword(u, req.Password) {
		return writeError(w, r, http.StatusUnauthorized, "INVALID_PASSWORD", "Invalid password")
	}

	// Verify TOTP code if provided
	if req.TOTPCode != "" && !totp.Verify(u.TOTPSecret, req.TOTPCode) {
		return writeError(w, r, http.StatusBadRequest, "INVALID_TOTP", "Invalid 2FA code")
	}

	// Disable 2FA
	if err := s.userService.SetTOTPEnabled(authUser.ID, false, ""); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "2FA_DISABLE_FAILED", "Failed to disable 2FA")
	}

	// Delete recovery keys
	s.recoveryService.DeleteAllKeys(authUser.ID)

	return writeSuccess(w, r, map[string]interface{}{
		"enabled": false,
	}, "2FA disabled", "2FA has been disabled")
}

// HandleChangePassword handles POST /api/v1/users/security/password
func (s *Service) HandleChangePassword(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_FIELDS", "Current and new password are required")
	}

	// Get user and verify current password
	u, _ := s.userService.GetByID(authUser.ID)
	if !s.userService.VerifyPassword(u, req.CurrentPassword) {
		return writeError(w, r, http.StatusUnauthorized, "INVALID_PASSWORD", "Current password is incorrect")
	}

	// Update password
	if err := s.userService.UpdatePassword(authUser.ID, req.NewPassword); err != nil {
		if errors.Is(err, user.ErrInvalidPassword) {
			return writeError(w, r, http.StatusBadRequest, "INVALID_NEW_PASSWORD", "New password does not meet requirements")
		}
		return writeError(w, r, http.StatusInternalServerError, "PASSWORD_CHANGE_FAILED", "Failed to change password")
	}

	return writeSuccess(w, r, nil, "Password changed", "Password has been changed successfully")
}

// HandleListTokens handles GET /api/v1/users/tokens
func (s *Service) HandleListTokens(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	tokens, err := s.tokenService.ListUserTokens(authUser.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "TOKEN_LIST_FAILED", "Failed to list tokens")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"tokens": tokens,
	}, "Tokens listed", "")
}

// HandleCreateToken handles POST /api/v1/users/tokens
func (s *Service) HandleCreateToken(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// Check max tokens
	if s.config != nil && s.config.Tokens.MaxPerUser > 0 {
		count, _ := s.tokenService.CountUserTokens(authUser.ID)
		if count >= s.config.Tokens.MaxPerUser {
			return writeError(w, r, http.StatusBadRequest, "MAX_TOKENS_REACHED", "Maximum number of tokens reached")
		}
	}

	var req CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.Name == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_NAME", "Token name is required")
	}

	// Set default scopes
	if len(req.Scopes) == 0 {
		req.Scopes = []string{token.ScopeRead}
	}

	// Calculate expiration
	var expiresAt *int64
	if req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second).Unix()
		expiresAt = &exp
	} else if s.config != nil && s.config.Tokens.ExpirationDays > 0 {
		exp := time.Now().AddDate(0, 0, s.config.Tokens.ExpirationDays).Unix()
		expiresAt = &exp
	}

	// Create token
	fullToken, tokenInfo, err := s.tokenService.CreateUserToken(authUser.ID, req.Name, req.Scopes, expiresAt)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "TOKEN_CREATE_FAILED", "Failed to create token")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"token":      fullToken,
		"token_info": tokenInfo,
	}, "Token created", "Token: "+fullToken+"\nSave this token - it won't be shown again!")
}

// HandleRevokeToken handles DELETE /api/v1/users/tokens/{id}
func (s *Service) HandleRevokeToken(w http.ResponseWriter, r *http.Request, tokenID int64) error {
	if r.Method != http.MethodDelete {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	if err := s.tokenService.RevokeUserToken(tokenID, authUser.ID); err != nil {
		if errors.Is(err, token.ErrTokenNotFound) {
			return writeError(w, r, http.StatusNotFound, "TOKEN_NOT_FOUND", "Token not found")
		}
		return writeError(w, r, http.StatusInternalServerError, "REVOKE_FAILED", "Failed to revoke token")
	}

	return writeSuccess(w, r, nil, "Token revoked", "Token has been revoked")
}

// HandleListSessions handles GET /api/v1/users/sessions
func (s *Service) HandleListSessions(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	sessions, err := s.sessionService.ListForUser(authUser.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "SESSION_LIST_FAILED", "Failed to list sessions")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"sessions": sessions,
	}, "Sessions listed", "")
}

// HandleRevokeSession handles DELETE /api/v1/users/sessions/{id}
func (s *Service) HandleRevokeSession(w http.ResponseWriter, r *http.Request, sessionID int64) error {
	if r.Method != http.MethodDelete {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// Verify session belongs to user
	sessions, _ := s.sessionService.ListForUser(authUser.ID)
	found := false
	for _, sess := range sessions {
		if sess.ID == sessionID {
			found = true
			break
		}
	}
	if !found {
		return writeError(w, r, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found")
	}

	if err := s.sessionService.DeleteByID(sessionID); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "REVOKE_FAILED", "Failed to revoke session")
	}

	return writeSuccess(w, r, nil, "Session revoked", "Session has been revoked")
}

// HandleRevokeAllSessions handles DELETE /api/v1/users/sessions
func (s *Service) HandleRevokeAllSessions(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodDelete {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// Get current session token to exclude it
	currentToken := web.GetSessionToken(r.Context())

	if currentToken != "" {
		s.sessionService.DeleteAllExcept(authUser.ID, currentToken)
	} else {
		s.sessionService.DeleteAllForUser(authUser.ID)
	}

	return writeSuccess(w, r, nil, "Sessions revoked", "All other sessions have been revoked")
}

// HandleRegenerateRecoveryKeys handles POST /api/v1/users/security/recovery-keys
func (s *Service) HandleRegenerateRecoveryKeys(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// Verify 2FA is enabled
	u, _ := s.userService.GetByID(authUser.ID)
	if !u.TOTPEnabled {
		return writeError(w, r, http.StatusBadRequest, "2FA_NOT_ENABLED", "2FA must be enabled to generate recovery keys")
	}

	// Generate new recovery keys
	keys, err := s.recoveryService.GenerateKeys(authUser.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "RECOVERY_ERROR", "Failed to generate recovery keys")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"recovery_keys": keys,
	}, "Recovery keys generated", "New recovery keys have been generated. Save them in a safe place.")
}

// Helper functions

func (s *Service) getPreferences(userID int64) (*UserPreferences, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	prefs := &UserPreferences{}
	var showEmail, showActivity, showOrgs, searchable int
	var emailSecurity, emailMentions, emailUpdates, reduceMotion int

	err := s.db.QueryRowContext(ctx, `
		SELECT show_email, show_activity, show_orgs, searchable,
		       email_security, email_mentions, email_updates, email_digest,
		       theme, font_size, reduce_motion, date_format, time_format
		FROM user_preferences WHERE user_id = ?
	`, userID).Scan(
		&showEmail, &showActivity, &showOrgs, &searchable,
		&emailSecurity, &emailMentions, &emailUpdates, &prefs.EmailDigest,
		&prefs.Theme, &prefs.FontSize, &reduceMotion, &prefs.DateFormat, &prefs.TimeFormat,
	)
	if err != nil {
		return nil, err
	}

	prefs.ShowEmail = showEmail == 1
	prefs.ShowActivity = showActivity == 1
	prefs.ShowOrgs = showOrgs == 1
	prefs.Searchable = searchable == 1
	prefs.EmailSecurity = emailSecurity == 1
	prefs.EmailMentions = emailMentions == 1
	prefs.EmailUpdates = emailUpdates == 1
	prefs.ReduceMotion = reduceMotion == 1

	return prefs, nil
}

func (s *Service) updatePreferences(userID int64, prefs UserPreferences) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	now := time.Now().Unix()

	// Upsert preferences
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_preferences (user_id, show_email, show_activity, show_orgs, searchable,
		                              email_security, email_mentions, email_updates, email_digest,
		                              theme, font_size, reduce_motion, date_format, time_format,
		                              created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
		  show_email = excluded.show_email,
		  show_activity = excluded.show_activity,
		  show_orgs = excluded.show_orgs,
		  searchable = excluded.searchable,
		  email_security = excluded.email_security,
		  email_mentions = excluded.email_mentions,
		  email_updates = excluded.email_updates,
		  email_digest = excluded.email_digest,
		  theme = excluded.theme,
		  font_size = excluded.font_size,
		  reduce_motion = excluded.reduce_motion,
		  date_format = excluded.date_format,
		  time_format = excluded.time_format,
		  updated_at = excluded.updated_at
	`, userID,
		boolToInt(prefs.ShowEmail), boolToInt(prefs.ShowActivity),
		boolToInt(prefs.ShowOrgs), boolToInt(prefs.Searchable),
		boolToInt(prefs.EmailSecurity), boolToInt(prefs.EmailMentions),
		boolToInt(prefs.EmailUpdates), prefs.EmailDigest,
		prefs.Theme, prefs.FontSize, boolToInt(prefs.ReduceMotion),
		prefs.DateFormat, prefs.TimeFormat, now, now,
	)
	return err
}

func getDefaultPreferences() *UserPreferences {
	return &UserPreferences{
		ShowEmail:     false,
		ShowActivity:  true,
		ShowOrgs:      true,
		Searchable:    true,
		EmailSecurity: true,
		EmailMentions: true,
		EmailUpdates:  false,
		EmailDigest:   "weekly",
		Theme:         "dark",
		FontSize:      "medium",
		ReduceMotion:  false,
		DateFormat:    "YYYY-MM-DD",
		TimeFormat:    "24h",
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
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
			if len(textData) > 0 && textData[len(textData)-1] != '\n' {
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

// ParseTokenID parses a token ID from a URL path segment
func ParseTokenID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

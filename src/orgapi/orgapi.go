
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package orgapi provides organization API handlers per PART 35
package orgapi

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
	"github.com/casjay-forks/caspaste/src/org"
	"github.com/casjay-forks/caspaste/src/token"
	"github.com/casjay-forks/caspaste/src/user"
	"github.com/casjay-forks/caspaste/src/web"
)

// Database query timeout
const defaultQueryTimeout = 5 * time.Second

// Service provides organization API operations
type Service struct {
	db           *sql.DB
	orgService   *org.Service
	userService  *user.Service
	tokenService *token.Service
	config       *config.FeaturesConfig
}

// NewService creates a new org API service
func NewService(db *sql.DB, orgSvc *org.Service, userSvc *user.Service, tokenSvc *token.Service, cfg *config.FeaturesConfig) *Service {
	return &Service{
		db:           db,
		orgService:   orgSvc,
		userService:  userSvc,
		tokenService: tokenSvc,
		config:       cfg,
	}
}

// APIResponse is the unified response format per PART 16
type APIResponse struct {
	OK      bool        `json:"ok"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// CreateOrgRequest is the request body for creating an organization
type CreateOrgRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Website     string `json:"website,omitempty"`
	Location    string `json:"location,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
}

// UpdateOrgRequest is the request body for updating an organization
type UpdateOrgRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Website     *string `json:"website,omitempty"`
	Location    *string `json:"location,omitempty"`
	Visibility  *string `json:"visibility,omitempty"`
	Email       *string `json:"email,omitempty"`
}

// AddMemberRequest is the request body for adding a member
type AddMemberRequest struct {
	Username string `json:"username"`
	Role     string `json:"role,omitempty"`
}

// UpdateMemberRequest is the request body for updating a member's role
type UpdateMemberRequest struct {
	Role string `json:"role"`
}

// CreateTokenRequest is the request body for creating an org token
type CreateTokenRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes,omitempty"`
	ExpiresIn int64    `json:"expires_in,omitempty"`
}

// OrgPreferences represents organization preferences
type OrgPreferences struct {
	DefaultRole         string `json:"default_role"`
	Require2FA          bool   `json:"require_2fa"`
	NotifyMemberJoin    bool   `json:"notify_member_join"`
	NotifyMemberLeave   bool   `json:"notify_member_leave"`
	NotifyRoleChange    bool   `json:"notify_role_change"`
	NotifyTokenActivity bool   `json:"notify_token_activity"`
}

// HandleCreateOrg handles POST /api/v1/orgs
func (s *Service) HandleCreateOrg(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	// Check if organizations are enabled
	if s.config == nil || !s.config.Organizations.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Organizations are not enabled")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var req CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.Slug == "" || req.Name == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_FIELDS", "Slug and name are required")
	}

	// Set default visibility
	if req.Visibility == "" {
		req.Visibility = "public"
	}

	// Create organization
	input := org.CreateOrgInput{
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		Website:     req.Website,
		Location:    req.Location,
		Visibility:  req.Visibility,
	}

	newOrg, err := s.orgService.Create(input, authUser.ID)
	if err != nil {
		switch {
		case errors.Is(err, org.ErrInvalidSlug):
			return writeError(w, r, http.StatusBadRequest, "INVALID_SLUG", "Invalid organization slug format")
		case errors.Is(err, org.ErrSlugTaken):
			return writeError(w, r, http.StatusConflict, "SLUG_TAKEN", "This slug is already taken")
		case errors.Is(err, org.ErrSlugBlocked):
			return writeError(w, r, http.StatusBadRequest, "SLUG_BLOCKED", "This slug is not allowed")
		default:
			return writeError(w, r, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create organization")
		}
	}

	return writeSuccess(w, r, newOrg, "Organization created", fmt.Sprintf("Organization '%s' created successfully", newOrg.Name))
}

// HandleListOrgs handles GET /api/v1/orgs
func (s *Service) HandleListOrgs(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	orgs, err := s.orgService.GetUserOrgs(authUser.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "LIST_FAILED", "Failed to list organizations")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"organizations": orgs,
	}, "Organizations listed", "")
}

// HandleGetOrg handles GET /api/v1/orgs/{slug}
func (s *Service) HandleGetOrg(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, org.ErrOrgNotFound) {
			return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
		}
		return writeError(w, r, http.StatusInternalServerError, "GET_FAILED", "Failed to get organization")
	}

	// Check visibility
	authUser := web.GetAuthUser(r.Context())
	if o.Visibility == "private" {
		if authUser == nil || !s.orgService.IsMember(o.ID, authUser.ID) {
			return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
		}
	}

	// Get member role if authenticated
	var memberRole string
	if authUser != nil {
		memberRole = s.orgService.GetMemberRole(o.ID, authUser.ID)
	}

	return writeSuccess(w, r, map[string]interface{}{
		"organization": o,
		"member_role":  memberRole,
	}, "Organization retrieved", fmt.Sprintf("Name: %s\nSlug: %s", o.Name, o.Slug))
}

// HandleUpdateOrg handles PATCH /api/v1/orgs/{slug}
func (s *Service) HandleUpdateOrg(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodPatch {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Check permission (admin or owner)
	role := s.orgService.GetMemberRole(o.ID, authUser.ID)
	if role != "owner" && role != "admin" {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to update this organization")
	}

	var req UpdateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	// Build update input
	input := org.UpdateOrgInput{
		Name:        req.Name,
		Description: req.Description,
		Website:     req.Website,
		Location:    req.Location,
		Visibility:  req.Visibility,
		Email:       req.Email,
	}

	if err := s.orgService.Update(o.ID, input); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update organization")
	}

	// Get updated org
	o, _ = s.orgService.GetBySlug(slug)

	return writeSuccess(w, r, o, "Organization updated", "Organization updated successfully")
}

// HandleDeleteOrg handles DELETE /api/v1/orgs/{slug}
func (s *Service) HandleDeleteOrg(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodDelete {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Only owner can delete
	if o.OwnerID != authUser.ID {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "Only the owner can delete this organization")
	}

	if err := s.orgService.Delete(o.ID); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete organization")
	}

	return writeSuccess(w, r, nil, "Organization deleted", "Organization has been deleted")
}

// HandleGetMembers handles GET /api/v1/orgs/{slug}/members
func (s *Service) HandleGetMembers(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Check visibility for private orgs
	authUser := web.GetAuthUser(r.Context())
	if o.Visibility == "private" {
		if authUser == nil || !s.orgService.IsMember(o.ID, authUser.ID) {
			return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
		}
	}

	members, err := s.orgService.GetMembers(o.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "LIST_FAILED", "Failed to list members")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"members": members,
	}, "Members listed", "")
}

// HandleAddMember handles POST /api/v1/orgs/{slug}/members
func (s *Service) HandleAddMember(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Check permission (admin or owner)
	role := s.orgService.GetMemberRole(o.ID, authUser.ID)
	if role != "owner" && role != "admin" {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to add members")
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.Username == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_USERNAME", "Username is required")
	}

	// Set default role
	if req.Role == "" {
		req.Role = "member"
	}

	// Admins can only add members, not other admins
	if role == "admin" && req.Role != "member" {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "Admins can only add members, not admins or owners")
	}

	// Get user by username
	u, err := s.userService.GetByUsername(req.Username)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	// Add member
	if err := s.orgService.AddMember(o.ID, u.ID, req.Role); err != nil {
		if errors.Is(err, org.ErrAlreadyMember) {
			return writeError(w, r, http.StatusConflict, "ALREADY_MEMBER", "User is already a member")
		}
		return writeError(w, r, http.StatusInternalServerError, "ADD_FAILED", "Failed to add member")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"username": u.Username,
		"role":     req.Role,
	}, "Member added", fmt.Sprintf("User '%s' added as %s", u.Username, req.Role))
}

// HandleUpdateMember handles PATCH /api/v1/orgs/{slug}/members/{username}
func (s *Service) HandleUpdateMember(w http.ResponseWriter, r *http.Request, slug, username string) error {
	if r.Method != http.MethodPatch {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Only owner can change roles
	if o.OwnerID != authUser.ID {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "Only the owner can change member roles")
	}

	var req UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.Role == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_ROLE", "Role is required")
	}

	// Get user
	u, err := s.userService.GetByUsername(username)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	// Can't change owner's role
	if u.ID == o.OwnerID {
		return writeError(w, r, http.StatusBadRequest, "CANNOT_CHANGE_OWNER", "Cannot change owner's role")
	}

	// Update role
	if err := s.orgService.UpdateMemberRole(o.ID, u.ID, req.Role); err != nil {
		if errors.Is(err, org.ErrNotMember) {
			return writeError(w, r, http.StatusNotFound, "NOT_MEMBER", "User is not a member")
		}
		return writeError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update role")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"username": u.Username,
		"role":     req.Role,
	}, "Role updated", fmt.Sprintf("User '%s' role changed to %s", u.Username, req.Role))
}

// HandleRemoveMember handles DELETE /api/v1/orgs/{slug}/members/{username}
func (s *Service) HandleRemoveMember(w http.ResponseWriter, r *http.Request, slug, username string) error {
	if r.Method != http.MethodDelete {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Get user to remove
	u, err := s.userService.GetByUsername(username)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	// Check permission
	role := s.orgService.GetMemberRole(o.ID, authUser.ID)
	targetRole := s.orgService.GetMemberRole(o.ID, u.ID)

	// Users can remove themselves
	if u.ID == authUser.ID {
		// Owner can't leave without transferring ownership
		if o.OwnerID == authUser.ID {
			return writeError(w, r, http.StatusBadRequest, "OWNER_CANNOT_LEAVE", "Transfer ownership before leaving")
		}
	} else {
		// Only owner and admin can remove others
		if role != "owner" && role != "admin" {
			return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to remove members")
		}
		// Admins can't remove other admins or owner
		if role == "admin" && (targetRole == "admin" || targetRole == "owner") {
			return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "Admins cannot remove other admins")
		}
		// Can't remove owner
		if u.ID == o.OwnerID {
			return writeError(w, r, http.StatusBadRequest, "CANNOT_REMOVE_OWNER", "Cannot remove owner")
		}
	}

	if err := s.orgService.RemoveMember(o.ID, u.ID); err != nil {
		if errors.Is(err, org.ErrNotMember) {
			return writeError(w, r, http.StatusNotFound, "NOT_MEMBER", "User is not a member")
		}
		return writeError(w, r, http.StatusInternalServerError, "REMOVE_FAILED", "Failed to remove member")
	}

	return writeSuccess(w, r, nil, "Member removed", fmt.Sprintf("User '%s' removed from organization", u.Username))
}

// HandleTransferOwnership handles POST /api/v1/orgs/{slug}/transfer
func (s *Service) HandleTransferOwnership(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Only current owner can transfer
	if o.OwnerID != authUser.ID {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "Only the owner can transfer ownership")
	}

	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.Username == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_USERNAME", "New owner username is required")
	}

	// Get new owner
	newOwner, err := s.userService.GetByUsername(req.Username)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	// Must be a member
	if !s.orgService.IsMember(o.ID, newOwner.ID) {
		return writeError(w, r, http.StatusBadRequest, "NOT_MEMBER", "New owner must be a member of the organization")
	}

	if err := s.orgService.TransferOwnership(o.ID, authUser.ID, newOwner.ID); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "TRANSFER_FAILED", "Failed to transfer ownership")
	}

	return writeSuccess(w, r, nil, "Ownership transferred", fmt.Sprintf("Ownership transferred to '%s'", newOwner.Username))
}

// HandleGetOrgSettings handles GET /api/v1/orgs/{slug}/settings
func (s *Service) HandleGetOrgSettings(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Check permission (admin or owner)
	role := s.orgService.GetMemberRole(o.ID, authUser.ID)
	if role != "owner" && role != "admin" {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to view settings")
	}

	prefs, err := s.getOrgPreferences(o.ID)
	if err != nil {
		prefs = getDefaultOrgPreferences()
	}

	return writeSuccess(w, r, prefs, "Settings retrieved", "")
}

// HandleUpdateOrgSettings handles PATCH /api/v1/orgs/{slug}/settings
func (s *Service) HandleUpdateOrgSettings(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodPatch {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Check permission (admin or owner)
	role := s.orgService.GetMemberRole(o.ID, authUser.ID)
	if role != "owner" && role != "admin" {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to update settings")
	}

	var prefs OrgPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if err := s.updateOrgPreferences(o.ID, prefs); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update settings")
	}

	return writeSuccess(w, r, prefs, "Settings updated", "Organization settings updated")
}

// HandleListOrgTokens handles GET /api/v1/orgs/{slug}/tokens
func (s *Service) HandleListOrgTokens(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Check permission (admin or owner)
	role := s.orgService.GetMemberRole(o.ID, authUser.ID)
	if role != "owner" && role != "admin" {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to view tokens")
	}

	tokens, err := s.tokenService.ListOrgTokens(o.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "LIST_FAILED", "Failed to list tokens")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"tokens": tokens,
	}, "Tokens listed", "")
}

// HandleCreateOrgToken handles POST /api/v1/orgs/{slug}/tokens
func (s *Service) HandleCreateOrgToken(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Check permission (admin or owner)
	role := s.orgService.GetMemberRole(o.ID, authUser.ID)
	if role != "owner" && role != "admin" {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to create tokens")
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
	}

	fullToken, tokenInfo, err := s.tokenService.CreateOrgToken(o.ID, authUser.ID, req.Name, req.Scopes, expiresAt)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "TOKEN_CREATE_FAILED", "Failed to create token")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"token":      fullToken,
		"token_info": tokenInfo,
	}, "Token created", "Token: "+fullToken+"\nSave this token - it won't be shown again!")
}

// HandleRevokeOrgToken handles DELETE /api/v1/orgs/{slug}/tokens/{id}
func (s *Service) HandleRevokeOrgToken(w http.ResponseWriter, r *http.Request, slug string, tokenID int64) error {
	if r.Method != http.MethodDelete {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Check permission (admin or owner)
	role := s.orgService.GetMemberRole(o.ID, authUser.ID)
	if role != "owner" && role != "admin" {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to revoke tokens")
	}

	if err := s.tokenService.RevokeOrgToken(tokenID, o.ID); err != nil {
		if errors.Is(err, token.ErrTokenNotFound) {
			return writeError(w, r, http.StatusNotFound, "TOKEN_NOT_FOUND", "Token not found")
		}
		return writeError(w, r, http.StatusInternalServerError, "REVOKE_FAILED", "Failed to revoke token")
	}

	return writeSuccess(w, r, nil, "Token revoked", "Token has been revoked")
}

// Helper functions

func (s *Service) getOrgPreferences(orgID int64) (*OrgPreferences, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	prefs := &OrgPreferences{}
	var require2fa, notifyJoin, notifyLeave, notifyRole, notifyToken int

	err := s.db.QueryRowContext(ctx, `
		SELECT default_role, require_2fa, notify_member_join, notify_member_leave,
		       notify_role_change, notify_token_activity
		FROM org_preferences WHERE org_id = ?
	`, orgID).Scan(
		&prefs.DefaultRole, &require2fa, &notifyJoin, &notifyLeave,
		&notifyRole, &notifyToken,
	)
	if err != nil {
		return nil, err
	}

	prefs.Require2FA = require2fa == 1
	prefs.NotifyMemberJoin = notifyJoin == 1
	prefs.NotifyMemberLeave = notifyLeave == 1
	prefs.NotifyRoleChange = notifyRole == 1
	prefs.NotifyTokenActivity = notifyToken == 1

	return prefs, nil
}

func (s *Service) updateOrgPreferences(orgID int64, prefs OrgPreferences) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	now := time.Now().Unix()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO org_preferences (org_id, default_role, require_2fa,
		                             notify_member_join, notify_member_leave,
		                             notify_role_change, notify_token_activity,
		                             created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(org_id) DO UPDATE SET
		  default_role = excluded.default_role,
		  require_2fa = excluded.require_2fa,
		  notify_member_join = excluded.notify_member_join,
		  notify_member_leave = excluded.notify_member_leave,
		  notify_role_change = excluded.notify_role_change,
		  notify_token_activity = excluded.notify_token_activity,
		  updated_at = excluded.updated_at
	`, orgID, prefs.DefaultRole, boolToInt(prefs.Require2FA),
		boolToInt(prefs.NotifyMemberJoin), boolToInt(prefs.NotifyMemberLeave),
		boolToInt(prefs.NotifyRoleChange), boolToInt(prefs.NotifyTokenActivity),
		now, now,
	)
	return err
}

func getDefaultOrgPreferences() *OrgPreferences {
	return &OrgPreferences{
		DefaultRole:         "member",
		Require2FA:          false,
		NotifyMemberJoin:    true,
		NotifyMemberLeave:   true,
		NotifyRoleChange:    true,
		NotifyTokenActivity: true,
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

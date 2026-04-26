
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package org

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"
)

// Query timeouts per AI.md PART 10
const (
	defaultQueryTimeout = 5 * time.Second
	defaultListTimeout  = 10 * time.Second
	transactionTimeout  = 30 * time.Second
)

// Role constants
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
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

// Slug validation
const (
	SlugMinLength = 2
	SlugMaxLength = 39
)

// Common errors
var (
	ErrOrgNotFound       = errors.New("organization not found")
	ErrSlugTaken         = errors.New("organization slug is already taken")
	ErrSlugBlocked       = errors.New("organization slug is blocked")
	ErrInvalidSlug       = errors.New("invalid organization slug")
	ErrNotMember         = errors.New("user is not a member of this organization")
	ErrAlreadyMember     = errors.New("user is already a member of this organization")
	ErrCannotRemoveOwner = errors.New("cannot remove the owner from the organization")
	ErrNotOwner          = errors.New("only the owner can perform this action")
	ErrMustHaveOwner     = errors.New("organization must have an owner")
	ErrPermissionDenied  = errors.New("permission denied")
)

// Org represents an organization per PART 35
type Org struct {
	ID            int64  `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	AvatarType    string `json:"avatar_type"`
	AvatarURL     string `json:"avatar_url,omitempty"`
	Website       string `json:"website,omitempty"`
	Location      string `json:"location,omitempty"`
	Visibility    string `json:"visibility"`
	OwnerID       int64  `json:"owner_id"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

// OrgMember represents a member of an organization
type OrgMember struct {
	ID        int64  `json:"id"`
	OrgID     int64  `json:"org_id"`
	UserID    int64  `json:"user_id"`
	Role      string `json:"role"`
	CreatedAt int64  `json:"created_at"`
	// Additional fields populated from users table
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarType  string `json:"avatar_type,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

// CreateOrgInput contains fields for creating a new organization
type CreateOrgInput struct {
	Slug        string
	Name        string
	Description string
	Website     string
	Location    string
	Visibility  string
}

// UpdateOrgInput contains fields for updating an organization
type UpdateOrgInput struct {
	Name        *string
	Description *string
	AvatarType  *string
	AvatarURL   *string
	Website     *string
	Location    *string
	Visibility  *string
	Email       *string
}

// Service provides organization operations
type Service struct {
	db *sql.DB
}

// NewService creates a new organization service
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Create creates a new organization
func (s *Service) Create(input CreateOrgInput, ownerID int64) (*Org, error) {
	// Validate slug
	if err := ValidateSlug(input.Slug); err != nil {
		return nil, err
	}

	// Check if slug is available (users and orgs share namespace)
	if err := s.CheckSlugAvailable(input.Slug); err != nil {
		return nil, err
	}

	// Set defaults
	visibility := input.Visibility
	if visibility == "" {
		visibility = VisibilityPublic
	}

	now := time.Now().Unix()
	slug := strings.ToLower(input.Slug)

	// Transaction timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), transactionTimeout)
	defer cancel()

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create organization
	result, err := tx.ExecContext(ctx, `
		INSERT INTO orgs (slug, name, description, website, location, visibility, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, slug, input.Name, input.Description, input.Website, input.Location, visibility, ownerID, now, now)
	if err != nil {
		return nil, err
	}

	orgID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Add owner as member with owner role
	_, err = tx.ExecContext(ctx, `
		INSERT INTO org_members (org_id, user_id, role, created_at)
		VALUES (?, ?, ?, ?)
	`, orgID, ownerID, RoleOwner, now)
	if err != nil {
		return nil, err
	}

	// Create default preferences
	_, err = tx.ExecContext(ctx, `
		INSERT INTO org_preferences (org_id, created_at, updated_at)
		VALUES (?, ?, ?)
	`, orgID, now, now)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetByID(orgID)
}

// GetByID retrieves an organization by ID
func (s *Service) GetByID(id int64) (*Org, error) {
	org := &Org{}
	var emailVerified int

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	err := s.db.QueryRowContext(ctx, `
		SELECT id, slug, name, description, avatar_type, avatar_url, website, location,
		       visibility, owner_id, email, email_verified, created_at, updated_at
		FROM orgs WHERE id = ?
	`, id).Scan(
		&org.ID, &org.Slug, &org.Name, &org.Description, &org.AvatarType, &org.AvatarURL,
		&org.Website, &org.Location, &org.Visibility, &org.OwnerID, &org.Email,
		&emailVerified, &org.CreatedAt, &org.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrOrgNotFound
	}
	if err != nil {
		return nil, err
	}

	org.EmailVerified = emailVerified == 1

	return org, nil
}

// GetBySlug retrieves an organization by slug
func (s *Service) GetBySlug(slug string) (*Org, error) {
	org := &Org{}
	var emailVerified int

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	err := s.db.QueryRowContext(ctx, `
		SELECT id, slug, name, description, avatar_type, avatar_url, website, location,
		       visibility, owner_id, email, email_verified, created_at, updated_at
		FROM orgs WHERE LOWER(slug) = LOWER(?)
	`, slug).Scan(
		&org.ID, &org.Slug, &org.Name, &org.Description, &org.AvatarType, &org.AvatarURL,
		&org.Website, &org.Location, &org.Visibility, &org.OwnerID, &org.Email,
		&emailVerified, &org.CreatedAt, &org.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrOrgNotFound
	}
	if err != nil {
		return nil, err
	}

	org.EmailVerified = emailVerified == 1

	return org, nil
}

// Update updates an organization
func (s *Service) Update(id int64, input UpdateOrgInput) error {
	var updates []string
	var args []interface{}

	if input.Name != nil {
		updates = append(updates, "name = ?")
		args = append(args, *input.Name)
	}
	if input.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *input.Description)
	}
	if input.AvatarType != nil {
		updates = append(updates, "avatar_type = ?")
		args = append(args, *input.AvatarType)
	}
	if input.AvatarURL != nil {
		updates = append(updates, "avatar_url = ?")
		args = append(args, *input.AvatarURL)
	}
	if input.Website != nil {
		updates = append(updates, "website = ?")
		args = append(args, *input.Website)
	}
	if input.Location != nil {
		updates = append(updates, "location = ?")
		args = append(args, *input.Location)
	}
	if input.Visibility != nil {
		updates = append(updates, "visibility = ?")
		args = append(args, *input.Visibility)
	}
	if input.Email != nil {
		updates = append(updates, "email = ?")
		args = append(args, *input.Email)
	}

	if len(updates) == 0 {
		return nil
	}

	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now().Unix())
	args = append(args, id)

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	query := "UPDATE orgs SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Delete removes an organization
func (s *Service) Delete(id int64) error {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, "DELETE FROM orgs WHERE id = ?", id)
	return err
}

// CheckSlugAvailable checks if a slug is available (orgs and users share namespace)
func (s *Service) CheckSlugAvailable(slug string) error {
	slug = strings.ToLower(slug)

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	// Check if org exists
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM orgs WHERE LOWER(slug) = ?", slug).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrSlugTaken
	}

	// Check if username exists
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE LOWER(username) = ?", slug).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrSlugTaken
	}

	return nil
}

// AddMember adds a user to an organization
func (s *Service) AddMember(orgID, userID int64, role string) error {
	// Validate role
	if role == "" {
		role = RoleMember
	}
	if role != RoleMember && role != RoleAdmin {
		return errors.New("invalid role")
	}

	// Check if already a member
	if s.IsMember(orgID, userID) {
		return ErrAlreadyMember
	}

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO org_members (org_id, user_id, role, created_at)
		VALUES (?, ?, ?, ?)
	`, orgID, userID, role, now)
	return err
}

// RemoveMember removes a user from an organization
func (s *Service) RemoveMember(orgID, userID int64) error {
	// Check if this is the owner
	org, err := s.GetByID(orgID)
	if err != nil {
		return err
	}
	if org.OwnerID == userID {
		return ErrCannotRemoveOwner
	}

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	result, err := s.db.ExecContext(ctx, "DELETE FROM org_members WHERE org_id = ? AND user_id = ?", orgID, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotMember
	}
	return nil
}

// UpdateMemberRole updates a member's role
func (s *Service) UpdateMemberRole(orgID, userID int64, role string) error {
	// Validate role
	if role != RoleMember && role != RoleAdmin && role != RoleOwner {
		return errors.New("invalid role")
	}

	// Cannot change owner's role
	org, err := s.GetByID(orgID)
	if err != nil {
		return err
	}
	if org.OwnerID == userID && role != RoleOwner {
		return errors.New("cannot change owner's role, transfer ownership instead")
	}

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	result, err := s.db.ExecContext(ctx, "UPDATE org_members SET role = ? WHERE org_id = ? AND user_id = ?",
		role, orgID, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotMember
	}
	return nil
}

// GetMembers returns all members of an organization
func (s *Service) GetMembers(orgID int64) ([]OrgMember, error) {
	// List timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultListTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.org_id, m.user_id, m.role, m.created_at,
		       u.username, u.display_name, u.avatar_type, u.avatar_url
		FROM org_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.org_id = ?
		ORDER BY m.role = 'owner' DESC, m.role = 'admin' DESC, m.created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []OrgMember
	for rows.Next() {
		var m OrgMember
		err := rows.Scan(
			&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.CreatedAt,
			&m.Username, &m.DisplayName, &m.AvatarType, &m.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, nil
}

// GetUserOrgs returns all organizations a user is a member of
func (s *Service) GetUserOrgs(userID int64) ([]Org, error) {
	// List timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultListTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT o.id, o.slug, o.name, o.description, o.avatar_type, o.avatar_url,
		       o.website, o.location, o.visibility, o.owner_id, o.email,
		       o.email_verified, o.created_at, o.updated_at
		FROM orgs o
		JOIN org_members m ON m.org_id = o.id
		WHERE m.user_id = ?
		ORDER BY o.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []Org
	for rows.Next() {
		var org Org
		var emailVerified int

		err := rows.Scan(
			&org.ID, &org.Slug, &org.Name, &org.Description, &org.AvatarType, &org.AvatarURL,
			&org.Website, &org.Location, &org.Visibility, &org.OwnerID, &org.Email,
			&emailVerified, &org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		org.EmailVerified = emailVerified == 1
		orgs = append(orgs, org)
	}

	return orgs, nil
}

// IsMember checks if a user is a member of an organization
func (s *Service) IsMember(orgID, userID int64) bool {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var count int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM org_members WHERE org_id = ? AND user_id = ?", orgID, userID).Scan(&count)
	return count > 0
}

// GetMemberRole returns a user's role in an organization
func (s *Service) GetMemberRole(orgID, userID int64) string {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var role string
	err := s.db.QueryRowContext(ctx, "SELECT role FROM org_members WHERE org_id = ? AND user_id = ?", orgID, userID).Scan(&role)
	if err != nil {
		return ""
	}
	return role
}

// TransferOwnership transfers ownership to another member
func (s *Service) TransferOwnership(orgID, currentOwnerID, newOwnerID int64) error {
	// Verify current owner
	org, err := s.GetByID(orgID)
	if err != nil {
		return err
	}
	if org.OwnerID != currentOwnerID {
		return ErrNotOwner
	}

	// Verify new owner is a member
	if !s.IsMember(orgID, newOwnerID) {
		return ErrNotMember
	}

	// Transaction timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), transactionTimeout)
	defer cancel()

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update organization owner
	_, err = tx.ExecContext(ctx, "UPDATE orgs SET owner_id = ?, updated_at = ? WHERE id = ?",
		newOwnerID, time.Now().Unix(), orgID)
	if err != nil {
		return err
	}

	// Update member roles
	_, err = tx.ExecContext(ctx, "UPDATE org_members SET role = ? WHERE org_id = ? AND user_id = ?",
		RoleAdmin, orgID, currentOwnerID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "UPDATE org_members SET role = ? WHERE org_id = ? AND user_id = ?",
		RoleOwner, orgID, newOwnerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetMemberCount returns the number of members in an organization
func (s *Service) GetMemberCount(orgID int64) (int, error) {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM org_members WHERE org_id = ?", orgID).Scan(&count)
	return count, err
}

// CanManageMembers checks if a user can manage members
func (s *Service) CanManageMembers(orgID, userID int64) bool {
	role := s.GetMemberRole(orgID, userID)
	return role == RoleOwner || role == RoleAdmin
}

// CanDeleteOrg checks if a user can delete the organization
func (s *Service) CanDeleteOrg(orgID, userID int64) bool {
	org, err := s.GetByID(orgID)
	if err != nil {
		return false
	}
	return org.OwnerID == userID
}

// Slug validation regex
var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// ValidateSlug validates an organization slug per PART 35
func ValidateSlug(slug string) error {
	slug = strings.ToLower(strings.TrimSpace(slug))

	if len(slug) < SlugMinLength {
		return errors.New("slug must be at least 2 characters")
	}
	if len(slug) > SlugMaxLength {
		return errors.New("slug cannot exceed 39 characters")
	}

	if !slugRegex.MatchString(slug) {
		return errors.New("slug must be lowercase alphanumeric with hyphens")
	}

	if strings.Contains(slug, "--") {
		return errors.New("slug cannot contain consecutive hyphens")
	}

	return nil
}

// NormalizeSlug normalizes a slug for storage
func NormalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

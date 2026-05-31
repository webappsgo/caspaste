
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package domainapi provides custom domain API handlers per PART 36
package domainapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/casjay-forks/caspaste/src/config"
	"github.com/casjay-forks/caspaste/src/domain"
	"github.com/casjay-forks/caspaste/src/httputil"
	"github.com/casjay-forks/caspaste/src/org"
	"github.com/casjay-forks/caspaste/src/web"
)

// Service provides domain API operations
type Service struct {
	db            *sql.DB
	domainService *domain.Service
	orgService    *org.Service
	config        *config.CustomDomainsConfig
}

// NewService creates a new domain API service
func NewService(db *sql.DB, domainSvc *domain.Service, orgSvc *org.Service, cfg *config.CustomDomainsConfig) *Service {
	return &Service{
		db:            db,
		domainService: domainSvc,
		orgService:    orgSvc,
		config:        cfg,
	}
}

// APIResponse is the unified response format per PART 16
type APIResponse struct {
	OK      bool        `json:"ok"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// AddDomainRequest is the request body for adding a custom domain
type AddDomainRequest struct {
	Domain string `json:"domain"`
}

// ConfigureSSLRequest is the request body for configuring SSL
type ConfigureSSLRequest struct {
	Challenge   string            `json:"challenge"`
	Provider    string            `json:"provider,omitempty"`
	Credentials map[string]string `json:"credentials,omitempty"`
}

// HandleListUserDomains handles GET /api/v1/users/domains
func (s *Service) HandleListUserDomains(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	// Check if custom domains are enabled
	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	domains, err := s.domainService.GetByOwner("user", authUser.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "LIST_FAILED", "Failed to list domains")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"domains": domains,
	}, "Domains listed", "")
}

// HandleAddUserDomain handles POST /api/v1/users/domains
func (s *Service) HandleAddUserDomain(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// Check domain limit
	if s.config.MaxDomainsPerUser > 0 {
		existing, _ := s.domainService.GetByOwner("user", authUser.ID)
		if len(existing) >= s.config.MaxDomainsPerUser {
			return writeError(w, r, http.StatusBadRequest, "LIMIT_REACHED", "Maximum number of domains reached")
		}
	}

	var req AddDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.Domain == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_DOMAIN", "Domain is required")
	}

	// Normalize domain
	domainStr := strings.ToLower(strings.TrimSpace(req.Domain))

	// Check domain type restrictions
	isApex := !strings.Contains(domainStr, ".")
	isWildcard := strings.HasPrefix(domainStr, "*.")

	if isApex && !s.config.AllowApex {
		return writeError(w, r, http.StatusBadRequest, "APEX_NOT_ALLOWED", "Apex domains are not allowed")
	}
	if isWildcard && !s.config.AllowWildcard {
		return writeError(w, r, http.StatusBadRequest, "WILDCARD_NOT_ALLOWED", "Wildcard domains are not allowed")
	}
	if !isApex && !isWildcard && !s.config.AllowSubdomain {
		return writeError(w, r, http.StatusBadRequest, "SUBDOMAIN_NOT_ALLOWED", "Subdomains are not allowed")
	}

	// Check reserved domains
	for _, reserved := range s.config.Reserved {
		if matchDomain(domainStr, reserved) {
			return writeError(w, r, http.StatusBadRequest, "DOMAIN_RESERVED", "This domain is reserved")
		}
	}

	// Create domain
	newDomain, err := s.domainService.Create("user", authUser.ID, domainStr)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDomainTaken):
			return writeError(w, r, http.StatusConflict, "DOMAIN_TAKEN", "This domain is already registered")
		case errors.Is(err, domain.ErrInvalidDomain):
			return writeError(w, r, http.StatusBadRequest, "INVALID_DOMAIN", "Invalid domain format")
		default:
			return writeError(w, r, http.StatusInternalServerError, "CREATE_FAILED", "Failed to add domain")
		}
	}

	// Get DNS instructions
	instructions, _ := s.domainService.GetDNSInstructions(newDomain.ID)

	return writeSuccess(w, r, map[string]interface{}{
		"domain":       newDomain,
		"instructions": instructions,
	}, "Domain added", fmt.Sprintf("Domain '%s' added. Follow the DNS instructions to verify.", domainStr))
}

// HandleGetUserDomain handles GET /api/v1/users/domains/{domain}
func (s *Service) HandleGetUserDomain(w http.ResponseWriter, r *http.Request, domainStr string) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	d, err := s.domainService.GetByDomain(domainStr)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	// Verify ownership
	if d.OwnerType != "user" || d.OwnerID != authUser.ID {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	return writeSuccess(w, r, d, "Domain retrieved", fmt.Sprintf("Domain: %s\nStatus: %s", d.Domain, d.Status))
}

// HandleDeleteUserDomain handles DELETE /api/v1/users/domains/{domain}
func (s *Service) HandleDeleteUserDomain(w http.ResponseWriter, r *http.Request, domainStr string) error {
	if r.Method != http.MethodDelete {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	d, err := s.domainService.GetByDomain(domainStr)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	// Verify ownership
	if d.OwnerType != "user" || d.OwnerID != authUser.ID {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	if err := s.domainService.Delete(d.ID); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete domain")
	}

	return writeSuccess(w, r, nil, "Domain deleted", "Domain has been deleted")
}

// HandleVerifyUserDomain handles POST /api/v1/users/domains/{domain}/verify
func (s *Service) HandleVerifyUserDomain(w http.ResponseWriter, r *http.Request, domainStr string) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	d, err := s.domainService.GetByDomain(domainStr)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	// Verify ownership
	if d.OwnerType != "user" || d.OwnerID != authUser.ID {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	// Check if already verified
	if d.VerificationStatus == "verified" {
		return writeSuccess(w, r, map[string]interface{}{
			"verified": true,
			"domain":   d,
		}, "Already verified", "Domain is already verified")
	}

	// Attempt verification
	result, err := s.domainService.Verify(d.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "VERIFY_FAILED", "Verification failed")
	}

	if result.OK {
		return writeSuccess(w, r, map[string]interface{}{
			"verified": true,
			"domain":   d,
		}, "Domain verified", "Domain has been verified successfully")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"verified": false,
		"message":  result.Message,
	}, "Verification pending", result.Message)
}

// HandleGetUserDomainDNS handles GET /api/v1/users/domains/{domain}/dns
func (s *Service) HandleGetUserDomainDNS(w http.ResponseWriter, r *http.Request, domainStr string) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	d, err := s.domainService.GetByDomain(domainStr)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	// Verify ownership
	if d.OwnerType != "user" || d.OwnerID != authUser.ID {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	instructions, err := s.domainService.GetDNSInstructions(d.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "DNS_ERROR", "Failed to get DNS instructions")
	}

	return writeSuccess(w, r, instructions, "DNS instructions", instructions.Instructions)
}

// HandleGetUserDomainSSL handles GET /api/v1/users/domains/{domain}/ssl
func (s *Service) HandleGetUserDomainSSL(w http.ResponseWriter, r *http.Request, domainStr string) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	d, err := s.domainService.GetByDomain(domainStr)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	// Verify ownership
	if d.OwnerType != "user" || d.OwnerID != authUser.ID {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"ssl_enabled":  d.SSLEnabled,
		"ssl_status":   d.SSLStatus,
		"ssl_provider": d.SSLProvider,
		"ssl_issued":   d.SSLIssuedAt,
		"ssl_expires":  d.SSLExpiresAt,
	}, "SSL status", fmt.Sprintf("SSL enabled: %v\nStatus: %s", d.SSLEnabled, d.SSLStatus))
}

// HandleConfigureUserDomainSSL handles POST /api/v1/users/domains/{domain}/ssl
func (s *Service) HandleConfigureUserDomainSSL(w http.ResponseWriter, r *http.Request, domainStr string) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	d, err := s.domainService.GetByDomain(domainStr)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	// Verify ownership
	if d.OwnerType != "user" || d.OwnerID != authUser.ID {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	// Must be verified first
	if d.VerificationStatus != "verified" {
		return writeError(w, r, http.StatusBadRequest, "NOT_VERIFIED", "Domain must be verified before configuring SSL")
	}

	var req ConfigureSSLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.Challenge == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_CHALLENGE", "Challenge type is required")
	}

	// Configure SSL
	if err := s.domainService.ConfigureSSL(d.ID, req.Challenge, req.Provider, req.Credentials); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "SSL_CONFIGURE_FAILED", "Failed to configure SSL")
	}

	// Issue certificate
	if err := s.domainService.IssueCertificate(d.ID); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "SSL_ISSUE_FAILED", "Failed to issue SSL certificate")
	}

	// Get updated domain
	d, _ = s.domainService.GetByDomain(domainStr)

	return writeSuccess(w, r, map[string]interface{}{
		"ssl_enabled": d.SSLEnabled,
		"ssl_status":  d.SSLStatus,
		"ssl_expires": d.SSLExpiresAt,
	}, "SSL configured", "SSL certificate has been configured")
}

// Organization domain handlers (same pattern)

// HandleListOrgDomains handles GET /api/v1/orgs/{slug}/domains
func (s *Service) HandleListOrgDomains(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Check membership
	if !s.orgService.IsMember(o.ID, authUser.ID) {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You must be a member to view domains")
	}

	domains, err := s.domainService.GetByOwner("org", o.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "LIST_FAILED", "Failed to list domains")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"domains": domains,
	}, "Domains listed", "")
}

// HandleAddOrgDomain handles POST /api/v1/orgs/{slug}/domains
func (s *Service) HandleAddOrgDomain(w http.ResponseWriter, r *http.Request, slug string) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
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
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to add domains")
	}

	// Check domain limit
	if s.config.MaxDomainsPerOrg > 0 {
		existing, _ := s.domainService.GetByOwner("org", o.ID)
		if len(existing) >= s.config.MaxDomainsPerOrg {
			return writeError(w, r, http.StatusBadRequest, "LIMIT_REACHED", "Maximum number of domains reached")
		}
	}

	var req AddDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	if req.Domain == "" {
		return writeError(w, r, http.StatusBadRequest, "MISSING_DOMAIN", "Domain is required")
	}

	domainStr := strings.ToLower(strings.TrimSpace(req.Domain))

	// Check reserved domains
	for _, reserved := range s.config.Reserved {
		if matchDomain(domainStr, reserved) {
			return writeError(w, r, http.StatusBadRequest, "DOMAIN_RESERVED", "This domain is reserved")
		}
	}

	newDomain, err := s.domainService.Create("org", o.ID, domainStr)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDomainTaken):
			return writeError(w, r, http.StatusConflict, "DOMAIN_TAKEN", "This domain is already registered")
		case errors.Is(err, domain.ErrInvalidDomain):
			return writeError(w, r, http.StatusBadRequest, "INVALID_DOMAIN", "Invalid domain format")
		default:
			return writeError(w, r, http.StatusInternalServerError, "CREATE_FAILED", "Failed to add domain")
		}
	}

	instructions, _ := s.domainService.GetDNSInstructions(newDomain.ID)

	return writeSuccess(w, r, map[string]interface{}{
		"domain":       newDomain,
		"instructions": instructions,
	}, "Domain added", fmt.Sprintf("Domain '%s' added. Follow the DNS instructions to verify.", domainStr))
}

// HandleGetOrgDomain handles GET /api/v1/orgs/{slug}/domains/{domain}
func (s *Service) HandleGetOrgDomain(w http.ResponseWriter, r *http.Request, slug, domainStr string) error {
	if r.Method != http.MethodGet {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
	}

	authUser := web.GetAuthUser(r.Context())
	if authUser == nil {
		return writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	o, err := s.orgService.GetBySlug(slug)
	if err != nil {
		return writeError(w, r, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	}

	// Check membership
	if !s.orgService.IsMember(o.ID, authUser.ID) {
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You must be a member to view domains")
	}

	d, err := s.domainService.GetByDomain(domainStr)
	if err != nil || d.OwnerType != "org" || d.OwnerID != o.ID {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	return writeSuccess(w, r, d, "Domain retrieved", fmt.Sprintf("Domain: %s\nStatus: %s", d.Domain, d.Status))
}

// HandleDeleteOrgDomain handles DELETE /api/v1/orgs/{slug}/domains/{domain}
func (s *Service) HandleDeleteOrgDomain(w http.ResponseWriter, r *http.Request, slug, domainStr string) error {
	if r.Method != http.MethodDelete {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
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
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to delete domains")
	}

	d, err := s.domainService.GetByDomain(domainStr)
	if err != nil || d.OwnerType != "org" || d.OwnerID != o.ID {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	if err := s.domainService.Delete(d.ID); err != nil {
		return writeError(w, r, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete domain")
	}

	return writeSuccess(w, r, nil, "Domain deleted", "Domain has been deleted")
}

// HandleVerifyOrgDomain handles POST /api/v1/orgs/{slug}/domains/{domain}/verify
func (s *Service) HandleVerifyOrgDomain(w http.ResponseWriter, r *http.Request, slug, domainStr string) error {
	if r.Method != http.MethodPost {
		return writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	if s.config == nil || !s.config.Enabled {
		return writeError(w, r, http.StatusForbidden, "FEATURE_DISABLED", "Custom domains are not enabled")
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
		return writeError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to verify domains")
	}

	d, err := s.domainService.GetByDomain(domainStr)
	if err != nil || d.OwnerType != "org" || d.OwnerID != o.ID {
		return writeError(w, r, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Domain not found")
	}

	if d.VerificationStatus == "verified" {
		return writeSuccess(w, r, map[string]interface{}{
			"verified": true,
			"domain":   d,
		}, "Already verified", "Domain is already verified")
	}

	result, err := s.domainService.Verify(d.ID)
	if err != nil {
		return writeError(w, r, http.StatusInternalServerError, "VERIFY_FAILED", "Verification failed")
	}

	if result.OK {
		return writeSuccess(w, r, map[string]interface{}{
			"verified": true,
			"domain":   d,
		}, "Domain verified", "Domain has been verified successfully")
	}

	return writeSuccess(w, r, map[string]interface{}{
		"verified": false,
		"message":  result.Message,
	}, "Verification pending", result.Message)
}

// Helper functions

func matchDomain(domain, pattern string) bool {
	if strings.HasPrefix(pattern, "*.") {
		// Wildcard match
		// e.g., ".local"
		suffix := pattern[1:]
		return strings.HasSuffix(domain, suffix)
	}
	return domain == pattern
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

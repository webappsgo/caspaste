
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package domain

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// Query timeouts per AI.md PART 10
const (
	defaultQueryTimeout = 5 * time.Second
	defaultListTimeout  = 10 * time.Second
)

// Owner type constants
const (
	OwnerTypeUser = "user"
	OwnerTypeOrg  = "org"
)

// Verification status constants
const (
	VerificationStatusPending  = "pending"
	VerificationStatusVerified = "verified"
	VerificationStatusFailed   = "failed"
)

// SSL status constants
const (
	SSLStatusNone    = "none"
	SSLStatusPending = "pending"
	SSLStatusActive  = "active"
	SSLStatusExpired = "expired"
	SSLStatusError   = "error"
)

// Domain status constants
const (
	StatusPending   = "pending"
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusError     = "error"
)

// Common errors
var (
	ErrDomainNotFound       = errors.New("domain not found")
	ErrDomainAlreadyExists  = errors.New("domain already exists")
	ErrDomainNotVerified    = errors.New("domain not verified")
	ErrDomainSuspended      = errors.New("domain is suspended")
	ErrVerificationFailed   = errors.New("domain verification failed")
	ErrDNSMismatch          = errors.New("domain does not resolve to this server")
	ErrDNSLookupFailed      = errors.New("DNS lookup failed")
	ErrMaxDomainsReached    = errors.New("maximum number of domains reached")
	ErrInvalidDomain        = errors.New("invalid domain")
	ErrReservedDomain       = errors.New("domain is reserved")
)

// CustomDomain represents a custom domain per PART 36
type CustomDomain struct {
	ID                 int64   `json:"id"`
	OwnerType          string  `json:"owner_type"`
	OwnerID            int64   `json:"owner_id"`
	Domain             string  `json:"domain"`
	IsApex             bool    `json:"is_apex"`
	IsWildcard         bool    `json:"is_wildcard"`
	VerificationStatus string  `json:"verification_status"`
	VerifiedAt         *int64  `json:"verified_at,omitempty"`
	VerifiedIP         string  `json:"verified_ip,omitempty"`
	LastCheckAt        *int64  `json:"last_check_at,omitempty"`
	CheckCount         int     `json:"check_count"`
	SSLEnabled         bool    `json:"ssl_enabled"`
	SSLStatus          string  `json:"ssl_status"`
	SSLChallenge       string  `json:"ssl_challenge,omitempty"`
	SSLProvider        string  `json:"ssl_provider,omitempty"`
	SSLCredentials     string  `json:"-"`
	SSLCertPEM         string  `json:"-"`
	SSLKeyPEM          string  `json:"-"`
	SSLIssuedAt        *int64  `json:"ssl_issued_at,omitempty"`
	SSLExpiresAt       *int64  `json:"ssl_expires_at,omitempty"`
	SSLLastError       string  `json:"ssl_last_error,omitempty"`
	Status             string  `json:"status"`
	SuspendedReason    string  `json:"suspended_reason,omitempty"`
	CreatedAt          int64   `json:"created_at"`
	UpdatedAt          int64   `json:"updated_at"`
}

// DNSInstructions contains DNS setup instructions
type DNSInstructions struct {
	Target       string   `json:"target"`
	TargetIPs    []string `json:"target_ips"`
	Instructions string   `json:"instructions"`
}

// VerifyResult contains verification result
type VerifyResult struct {
	OK         bool     `json:"ok"`
	Error      string   `json:"error,omitempty"`
	Message    string   `json:"message,omitempty"`
	ResolvedTo []string `json:"resolved_to,omitempty"`
}

// Service provides custom domain operations
type Service struct {
	db          *sql.DB
	serverFQDN  string
	serverIPs   []net.IP
	ipsMutex    sync.RWMutex
	lastIPCheck time.Time
	acmeManager *autocert.Manager
	acmeCacheDir string
}

// NewService creates a new domain service
func NewService(db *sql.DB, serverFQDN string, acmeCacheDir string) *Service {
	s := &Service{
		db:           db,
		serverFQDN:   serverFQDN,
		acmeCacheDir: acmeCacheDir,
	}

	// Initialize ACME manager for custom domain certificates
	if acmeCacheDir != "" {
		s.acmeManager = &autocert.Manager{
			Prompt: autocert.AcceptTOS,
			Cache:  autocert.DirCache(acmeCacheDir),
		}
	}

	// Initialize server IPs
	s.refreshPublicIPs()
	return s
}

// Create creates a new custom domain
func (s *Service) Create(ownerType string, ownerID int64, domain string) (*CustomDomain, error) {
	// Validate domain
	if err := ValidateDomain(domain); err != nil {
		return nil, err
	}

	// Normalize domain
	domain = NormalizeDomain(domain)

	// Check if already exists
	existing, _ := s.GetByDomain(domain)
	if existing != nil {
		return nil, ErrDomainAlreadyExists
	}

	// Determine if apex or subdomain
	isApex := IsApexDomain(domain)
	isWildcard := strings.HasPrefix(domain, "*.")

	now := time.Now().Unix()

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO custom_domains (owner_type, owner_id, domain, is_apex, is_wildcard, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, ownerType, ownerID, domain, boolToInt(isApex), boolToInt(isWildcard), now, now)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	// Log audit
	s.logAudit(id, "created", ownerType, ownerID, nil)

	return s.GetByID(id)
}

// GetByID retrieves a domain by ID
func (s *Service) GetByID(id int64) (*CustomDomain, error) {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	return s.scanDomain(s.db.QueryRowContext(ctx, `
		SELECT id, owner_type, owner_id, domain, is_apex, is_wildcard,
		       verification_status, verified_at, verified_ip, last_check_at, check_count,
		       ssl_enabled, ssl_status, ssl_challenge, ssl_provider, ssl_credentials,
		       ssl_cert_pem, ssl_key_pem, ssl_issued_at, ssl_expires_at, ssl_last_error,
		       status, suspended_reason, created_at, updated_at
		FROM custom_domains WHERE id = ?
	`, id))
}

// GetByDomain retrieves a domain by domain name
func (s *Service) GetByDomain(domain string) (*CustomDomain, error) {
	domain = NormalizeDomain(domain)

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	return s.scanDomain(s.db.QueryRowContext(ctx, `
		SELECT id, owner_type, owner_id, domain, is_apex, is_wildcard,
		       verification_status, verified_at, verified_ip, last_check_at, check_count,
		       ssl_enabled, ssl_status, ssl_challenge, ssl_provider, ssl_credentials,
		       ssl_cert_pem, ssl_key_pem, ssl_issued_at, ssl_expires_at, ssl_last_error,
		       status, suspended_reason, created_at, updated_at
		FROM custom_domains WHERE LOWER(domain) = LOWER(?)
	`, domain))
}

// GetByOwner retrieves all domains for an owner
func (s *Service) GetByOwner(ownerType string, ownerID int64) ([]CustomDomain, error) {
	// List timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultListTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_type, owner_id, domain, is_apex, is_wildcard,
		       verification_status, verified_at, verified_ip, last_check_at, check_count,
		       ssl_enabled, ssl_status, ssl_challenge, ssl_provider, ssl_credentials,
		       ssl_cert_pem, ssl_key_pem, ssl_issued_at, ssl_expires_at, ssl_last_error,
		       status, suspended_reason, created_at, updated_at
		FROM custom_domains WHERE owner_type = ? AND owner_id = ?
		ORDER BY domain
	`, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []CustomDomain
	for rows.Next() {
		d, err := s.scanDomainRow(rows)
		if err != nil {
			return nil, err
		}
		domains = append(domains, *d)
	}

	return domains, nil
}

// Delete removes a custom domain
func (s *Service) Delete(id int64) error {
	// Get domain for audit
	d, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err = s.db.ExecContext(ctx, "DELETE FROM custom_domains WHERE id = ?", id)
	if err != nil {
		return err
	}

	s.logAudit(id, "deleted", d.OwnerType, d.OwnerID, nil)
	return nil
}

// Verify verifies a custom domain by checking DNS resolution
func (s *Service) Verify(id int64) (*VerifyResult, error) {
	d, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Refresh server IPs if stale
	s.refreshPublicIPsIfNeeded()

	// Resolve the domain
	ips, err := net.LookupIP(d.Domain)
	if err != nil {
		s.updateVerificationStatus(id, VerificationStatusFailed)
		return &VerifyResult{
			OK:      false,
			Error:   "DNS_LOOKUP_FAILED",
			Message: "DNS lookup failed. Please check your DNS configuration.",
		}, nil
	}

	// Check if any resolved IP matches server IP
	serverIPs := s.GetServerPublicIPs()
	matched := false
	var resolvedIPs []string

	for _, ip := range ips {
		resolvedIPs = append(resolvedIPs, ip.String())
		for _, serverIP := range serverIPs {
			if ip.Equal(serverIP) {
				matched = true
				break
			}
		}
	}

	if !matched {
		s.updateVerificationStatus(id, VerificationStatusFailed)
		return &VerifyResult{
			OK:         false,
			Error:      "DNS_MISMATCH",
			Message:    "Domain does not resolve to this server. DNS propagation can take up to 48 hours.",
			ResolvedTo: resolvedIPs,
		}, nil
	}

	// Success - update status
	now := time.Now().Unix()

	// Query timeout per AI.md PART 10
	updateCtx, updateCancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer updateCancel()

	_, err = s.db.ExecContext(updateCtx, `
		UPDATE custom_domains SET
			verification_status = ?, verified_at = ?, verified_ip = ?,
			status = ?, updated_at = ?
		WHERE id = ?
	`, VerificationStatusVerified, now, resolvedIPs[0], StatusActive, now, id)
	if err != nil {
		return nil, err
	}

	s.logAudit(id, "verified", d.OwnerType, d.OwnerID, nil)

	return &VerifyResult{
		OK:         true,
		ResolvedTo: resolvedIPs,
	}, nil
}

// GetDNSInstructions returns DNS setup instructions for a domain
func (s *Service) GetDNSInstructions(id int64) (*DNSInstructions, error) {
	d, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	serverIPs := s.GetServerPublicIPs()
	var ipStrs []string
	for _, ip := range serverIPs {
		ipStrs = append(ipStrs, ip.String())
	}

	var instructions string
	if d.IsApex {
		instructions = "Add A/AAAA records pointing to the IP addresses above."
	} else {
		instructions = "Add a CNAME record pointing to " + s.serverFQDN + ", or A/AAAA records pointing to the IPs above."
	}

	return &DNSInstructions{
		Target:       s.serverFQDN,
		TargetIPs:    ipStrs,
		Instructions: instructions,
	}, nil
}

// Suspend suspends a domain
func (s *Service) Suspend(id int64, reason string) error {
	now := time.Now().Unix()

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, `
		UPDATE custom_domains SET status = ?, suspended_reason = ?, updated_at = ?
		WHERE id = ?
	`, StatusSuspended, reason, now, id)
	if err != nil {
		return err
	}

	s.logAudit(id, "suspended", "admin", 0, &reason)
	return nil
}

// Unsuspend unsuspends a domain
func (s *Service) Unsuspend(id int64) error {
	now := time.Now().Unix()

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, `
		UPDATE custom_domains SET status = ?, suspended_reason = NULL, updated_at = ?
		WHERE id = ?
	`, StatusActive, now, id)
	if err != nil {
		return err
	}

	s.logAudit(id, "unsuspended", "admin", 0, nil)
	return nil
}

// CountByOwner returns the count of domains for an owner
func (s *Service) CountByOwner(ownerType string, ownerID int64) (int, error) {
	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM custom_domains WHERE owner_type = ? AND owner_id = ?
	`, ownerType, ownerID).Scan(&count)
	return count, err
}

// GetServerPublicIPs returns the server's public IP addresses
func (s *Service) GetServerPublicIPs() []net.IP {
	s.ipsMutex.RLock()
	defer s.ipsMutex.RUnlock()
	return s.serverIPs
}

// RefreshPublicIPs refreshes the cached public IPs
func (s *Service) RefreshPublicIPs() {
	s.refreshPublicIPs()
}

func (s *Service) refreshPublicIPs() {
	s.ipsMutex.Lock()
	defer s.ipsMutex.Unlock()

	var ips []net.IP

	// From FQDN
	if s.serverFQDN != "" {
		if resolved, err := net.LookupIP(s.serverFQDN); err == nil {
			ips = append(ips, resolved...)
		}
	}

	// From external IP services
	externalIP := getExternalIP()
	if externalIP != nil {
		ips = append(ips, externalIP)
	}

	s.serverIPs = ips
	s.lastIPCheck = time.Now()
}

func (s *Service) refreshPublicIPsIfNeeded() {
	s.ipsMutex.RLock()
	stale := time.Since(s.lastIPCheck) > 12*time.Hour
	s.ipsMutex.RUnlock()

	if stale {
		s.refreshPublicIPs()
	}
}

// getExternalIP gets the external IP using a public service
func getExternalIP() net.IP {
	services := []string{
		"https://api.ipify.org",
		"https://icanhazip.com",
		"https://ifconfig.me/ip",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, svc := range services {
		resp, err := client.Get(svc)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ip := net.ParseIP(strings.TrimSpace(string(body)))
		if ip != nil {
			return ip
		}
	}

	return nil
}

func (s *Service) updateVerificationStatus(id int64, status string) {
	now := time.Now().Unix()

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	s.db.ExecContext(ctx, `
		UPDATE custom_domains SET
			verification_status = ?, last_check_at = ?, check_count = check_count + 1, updated_at = ?
		WHERE id = ?
	`, status, now, now, id)
}

func (s *Service) logAudit(domainID int64, action, actorType string, actorID int64, details *string) {
	now := time.Now().Unix()
	var detailsVal interface{}
	if details != nil {
		detailsVal = *details
	}

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	s.db.ExecContext(ctx, `
		INSERT INTO custom_domain_audit (domain_id, action, actor_type, actor_id, details, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, domainID, action, actorType, actorID, detailsVal, now)
}

func (s *Service) scanDomain(row *sql.Row) (*CustomDomain, error) {
	d := &CustomDomain{}
	var isApex, isWildcard, sslEnabled int
	var verifiedAt, lastCheckAt, sslIssuedAt, sslExpiresAt sql.NullInt64
	var verifiedIP, sslChallenge, sslProvider, sslCredentials, sslCertPEM, sslKeyPEM, sslLastError, suspendedReason sql.NullString

	err := row.Scan(
		&d.ID, &d.OwnerType, &d.OwnerID, &d.Domain, &isApex, &isWildcard,
		&d.VerificationStatus, &verifiedAt, &verifiedIP, &lastCheckAt, &d.CheckCount,
		&sslEnabled, &d.SSLStatus, &sslChallenge, &sslProvider, &sslCredentials,
		&sslCertPEM, &sslKeyPEM, &sslIssuedAt, &sslExpiresAt, &sslLastError,
		&d.Status, &suspendedReason, &d.CreatedAt, &d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrDomainNotFound
	}
	if err != nil {
		return nil, err
	}

	d.IsApex = isApex == 1
	d.IsWildcard = isWildcard == 1
	d.SSLEnabled = sslEnabled == 1

	if verifiedAt.Valid {
		d.VerifiedAt = &verifiedAt.Int64
	}
	if lastCheckAt.Valid {
		d.LastCheckAt = &lastCheckAt.Int64
	}
	if sslIssuedAt.Valid {
		d.SSLIssuedAt = &sslIssuedAt.Int64
	}
	if sslExpiresAt.Valid {
		d.SSLExpiresAt = &sslExpiresAt.Int64
	}
	d.VerifiedIP = verifiedIP.String
	d.SSLChallenge = sslChallenge.String
	d.SSLProvider = sslProvider.String
	d.SSLCredentials = sslCredentials.String
	d.SSLCertPEM = sslCertPEM.String
	d.SSLKeyPEM = sslKeyPEM.String
	d.SSLLastError = sslLastError.String
	d.SuspendedReason = suspendedReason.String

	return d, nil
}

func (s *Service) scanDomainRow(rows *sql.Rows) (*CustomDomain, error) {
	d := &CustomDomain{}
	var isApex, isWildcard, sslEnabled int
	var verifiedAt, lastCheckAt, sslIssuedAt, sslExpiresAt sql.NullInt64
	var verifiedIP, sslChallenge, sslProvider, sslCredentials, sslCertPEM, sslKeyPEM, sslLastError, suspendedReason sql.NullString

	err := rows.Scan(
		&d.ID, &d.OwnerType, &d.OwnerID, &d.Domain, &isApex, &isWildcard,
		&d.VerificationStatus, &verifiedAt, &verifiedIP, &lastCheckAt, &d.CheckCount,
		&sslEnabled, &d.SSLStatus, &sslChallenge, &sslProvider, &sslCredentials,
		&sslCertPEM, &sslKeyPEM, &sslIssuedAt, &sslExpiresAt, &sslLastError,
		&d.Status, &suspendedReason, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	d.IsApex = isApex == 1
	d.IsWildcard = isWildcard == 1
	d.SSLEnabled = sslEnabled == 1

	if verifiedAt.Valid {
		d.VerifiedAt = &verifiedAt.Int64
	}
	if lastCheckAt.Valid {
		d.LastCheckAt = &lastCheckAt.Int64
	}
	if sslIssuedAt.Valid {
		d.SSLIssuedAt = &sslIssuedAt.Int64
	}
	if sslExpiresAt.Valid {
		d.SSLExpiresAt = &sslExpiresAt.Int64
	}
	d.VerifiedIP = verifiedIP.String
	d.SSLChallenge = sslChallenge.String
	d.SSLProvider = sslProvider.String
	d.SSLCredentials = sslCredentials.String
	d.SSLCertPEM = sslCertPEM.String
	d.SSLKeyPEM = sslKeyPEM.String
	d.SSLLastError = sslLastError.String
	d.SuspendedReason = suspendedReason.String

	return d, nil
}

// ValidateDomain validates a domain name
func ValidateDomain(domain string) error {
	domain = strings.ToLower(strings.TrimSpace(domain))

	if domain == "" {
		return ErrInvalidDomain
	}

	// Basic validation
	if len(domain) > 253 {
		return errors.New("domain name too long")
	}

	// Check for reserved domains
	reserved := []string{
		"localhost",
		".local",
		".test",
		".example",
		".invalid",
	}
	for _, r := range reserved {
		if domain == r || strings.HasSuffix(domain, r) {
			return ErrReservedDomain
		}
	}

	return nil
}

// NormalizeDomain normalizes a domain for storage
func NormalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

// IsApexDomain checks if a domain is an apex domain (no subdomain)
func IsApexDomain(domain string) bool {
	parts := strings.Split(domain, ".")
	// An apex domain has only 2 parts (e.g., example.com)
	// But some TLDs have multiple parts (e.g., co.uk), so we check for common patterns
	if len(parts) <= 2 {
		return true
	}
	// Check for known multi-part TLDs
	multiPartTLDs := []string{"co.uk", "org.uk", "com.au", "net.au", "co.nz"}
	suffix := strings.Join(parts[len(parts)-2:], ".")
	for _, tld := range multiPartTLDs {
		if suffix == tld && len(parts) == 3 {
			return true
		}
	}
	return false
}

// ConfigureSSL configures SSL for a domain
func (s *Service) ConfigureSSL(id int64, challenge, provider string, credentials map[string]string) error {
	d, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if d.VerificationStatus != VerificationStatusVerified {
		return ErrDomainNotVerified
	}

	// Serialize credentials
	var credStr string
	if len(credentials) > 0 {
		// Simple JSON-like serialization
		parts := make([]string, 0, len(credentials))
		for k, v := range credentials {
			parts = append(parts, k+"="+v)
		}
		credStr = strings.Join(parts, ";")
	}

	now := time.Now().Unix()

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err = s.db.ExecContext(ctx, `
		UPDATE custom_domains SET
			ssl_challenge = ?, ssl_provider = ?, ssl_credentials = ?,
			ssl_status = ?, updated_at = ?
		WHERE id = ?
	`, challenge, provider, credStr, SSLStatusPending, now, id)
	if err != nil {
		return err
	}

	s.logAudit(id, "ssl_configured", d.OwnerType, d.OwnerID, nil)
	return nil
}

// IssueCertificate issues an SSL certificate for a domain via ACME/Let's Encrypt
func (s *Service) IssueCertificate(id int64) error {
	d, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if d.VerificationStatus != VerificationStatusVerified {
		return ErrDomainNotVerified
	}

	if s.acmeManager == nil {
		return fmt.Errorf("ACME certificate manager not configured")
	}

	now := time.Now().Unix()

	// Mark as pending
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err = s.db.ExecContext(ctx, `
		UPDATE custom_domains SET
			ssl_enabled = 1, ssl_status = ?, updated_at = ?
		WHERE id = ?
	`, SSLStatusPending, now, id)
	if err != nil {
		return err
	}

	// Issue certificate via ACME using autocert's built-in HTTP-01 challenge
	hello := &tls.ClientHelloInfo{ServerName: d.Domain}
	cert, err := s.acmeManager.GetCertificate(hello)
	if err != nil {
		// Mark as error
		errCtx, errCancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer errCancel()
		s.db.ExecContext(errCtx, `
			UPDATE custom_domains SET ssl_status = ?, updated_at = ? WHERE id = ?
		`, SSLStatusError, time.Now().Unix(), id)
		return fmt.Errorf("ACME certificate issuance failed for %s: %w", d.Domain, err)
	}

	// Extract certificate details
	var certPEM, keyPEM string
	var expiresAt int64
	if cert != nil && len(cert.Certificate) > 0 {
		leaf, parseErr := x509.ParseCertificate(cert.Certificate[0])
		if parseErr == nil {
			expiresAt = leaf.NotAfter.Unix()
		}

		// Encode cert chain as PEM
		var certBuilder strings.Builder
		for _, der := range cert.Certificate {
			pem.Encode(&certBuilder, &pem.Block{Type: "CERTIFICATE", Bytes: der})
		}
		certPEM = certBuilder.String()
	}

	// Store certificate data and mark as active
	updateCtx, updateCancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer updateCancel()

	_, err = s.db.ExecContext(updateCtx, `
		UPDATE custom_domains SET
			ssl_status = ?, ssl_issued_at = ?, ssl_expires_at = ?,
			ssl_cert_pem = ?, ssl_key_pem = ?, updated_at = ?
		WHERE id = ?
	`, SSLStatusActive, now, expiresAt, certPEM, keyPEM, time.Now().Unix(), id)
	if err != nil {
		return err
	}

	s.logAudit(id, "ssl_issued", d.OwnerType, d.OwnerID, nil)
	return nil
}

// RenewExpiring renews certificates expiring within the specified days
func (s *Service) RenewExpiring(renewBeforeDays int) (int, error) {
	threshold := time.Now().AddDate(0, 0, renewBeforeDays).Unix()

	// List timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultListTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM custom_domains
		WHERE ssl_enabled = 1 AND ssl_status = ? AND ssl_expires_at < ?
	`, SSLStatusActive, threshold)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	renewed := 0
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		if err := s.IssueCertificate(id); err == nil {
			renewed++
		}
	}

	return renewed, nil
}

// CleanupUnverified removes unverified domains older than the specified duration
func (s *Service) CleanupUnverified(maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge).Unix()

	// Query timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM custom_domains
		WHERE verification_status = ? AND created_at < ?
	`, VerificationStatusPending, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RetryPendingVerifications retries verification for pending domains
func (s *Service) RetryPendingVerifications() (int, error) {
	// List timeout per AI.md PART 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultListTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM custom_domains
		WHERE verification_status = ? AND check_count < 10
	`, VerificationStatusPending)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	verified := 0
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		result, _ := s.Verify(id)
		if result != nil && result.OK {
			verified++
		}
	}

	return verified, nil
}

// ErrDomainTaken is returned when a domain is already registered
var ErrDomainTaken = ErrDomainAlreadyExists

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

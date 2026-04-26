// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package ssl provides SSL/TLS certificate management per AI.md PART 15
package ssl

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	pathutil "github.com/casjay-forks/caspaste/src/path"
)

// CertSource indicates where a certificate came from
type CertSource string

const (
	// SourceManual is a manually configured certificate
	SourceManual CertSource = "manual"
	// SourceLetsEncrypt is a Let's Encrypt certificate
	SourceLetsEncrypt CertSource = "letsencrypt"
	// SourceACME is an auto-obtained ACME certificate
	SourceACME CertSource = "acme"
)

// Certificate holds certificate information
type Certificate struct {
	CertFile  string
	KeyFile   string
	Domain    string
	Source    CertSource
	ExpiresAt time.Time
	mu        sync.RWMutex
}

// Manager handles SSL/TLS certificate management
type Manager struct {
	sslDir      string
	certs       map[string]*Certificate
	mu          sync.RWMutex
	enabled     bool
	acmeEnabled bool
}

// NewManager creates a new SSL manager
func NewManager(enabled, acmeEnabled bool) *Manager {
	return &Manager{
		sslDir:      pathutil.SSLDir(),
		certs:       make(map[string]*Certificate),
		enabled:     enabled,
		acmeEnabled: acmeEnabled,
	}
}

// IsEnabled returns true if SSL is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// LoadCertificate loads a certificate from files
func (m *Manager) LoadCertificate(certFile, keyFile, domain string) (*Certificate, error) {
	// Check if files exist
	if _, err := os.Stat(certFile); err != nil {
		return nil, fmt.Errorf("certificate file not found: %s", certFile)
	}
	if _, err := os.Stat(keyFile); err != nil {
		return nil, fmt.Errorf("key file not found: %s", keyFile)
	}

	// Try to load the certificate pair
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("invalid TLS certificate/key pair: %w", err)
	}

	// Get expiration from leaf certificate
	var expiresAt time.Time
	if len(cert.Certificate) > 0 {
		if leaf, err := cert.Leaf, cert.Leaf; err == nil && leaf != nil {
			expiresAt = leaf.NotAfter
		}
	}

	certificate := &Certificate{
		CertFile:  certFile,
		KeyFile:   keyFile,
		Domain:    domain,
		Source:    SourceManual,
		ExpiresAt: expiresAt,
	}

	m.mu.Lock()
	m.certs[domain] = certificate
	m.mu.Unlock()

	return certificate, nil
}

// FindLetsEncryptCert searches for Let's Encrypt certificates
func (m *Manager) FindLetsEncryptCert(fqdn string) (*Certificate, error) {
	letsencryptDir := "/etc/letsencrypt/live"

	// Check if Let's Encrypt directory exists
	if _, err := os.Stat(letsencryptDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("Let's Encrypt directory not found: %s", letsencryptDir)
	}

	// Read all domain directories
	entries, err := os.ReadDir(letsencryptDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read Let's Encrypt directory: %w", err)
	}

	// First try to find exact domain match
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		domainDir := filepath.Join(letsencryptDir, entry.Name())
		certFile := filepath.Join(domainDir, "fullchain.pem")
		keyFile := filepath.Join(domainDir, "privkey.pem")

		// Check if cert files exist
		if _, err := os.Stat(certFile); err != nil {
			continue
		}
		if _, err := os.Stat(keyFile); err != nil {
			continue
		}

		// Try to load cert to validate it
		if _, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
			continue
		}

		// If domain matches, use this
		if entry.Name() == fqdn || entry.Name() == strings.TrimPrefix(fqdn, "www.") {
			cert := &Certificate{
				CertFile: certFile,
				KeyFile:  keyFile,
				Domain:   entry.Name(),
				Source:   SourceLetsEncrypt,
			}
			m.mu.Lock()
			m.certs[fqdn] = cert
			m.mu.Unlock()
			return cert, nil
		}
	}

	// If no exact match, return first valid cert found
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		domainDir := filepath.Join(letsencryptDir, entry.Name())
		certFile := filepath.Join(domainDir, "fullchain.pem")
		keyFile := filepath.Join(domainDir, "privkey.pem")

		// Check if cert files exist and are valid
		if _, err := os.Stat(certFile); err == nil {
			if _, err := os.Stat(keyFile); err == nil {
				if _, err := tls.LoadX509KeyPair(certFile, keyFile); err == nil {
					cert := &Certificate{
						CertFile: certFile,
						KeyFile:  keyFile,
						Domain:   entry.Name(),
						Source:   SourceLetsEncrypt,
					}
					m.mu.Lock()
					m.certs[fqdn] = cert
					m.mu.Unlock()
					return cert, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no valid Let's Encrypt certificates found in %s", letsencryptDir)
}

// GetCertificate returns a certificate for the given domain
func (m *Manager) GetCertificate(domain string) (*Certificate, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cert, ok := m.certs[domain]
	return cert, ok
}

// GetTLSConfig returns a TLS configuration for the given certificate
func (m *Manager) GetTLSConfig(cert *Certificate) (*tls.Config, error) {
	tlsCert, err := tls.LoadX509KeyPair(cert.CertFile, cert.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}, nil
}

// AutoDiscover attempts to find certificates automatically
// Priority: manual config > Let's Encrypt > ACME
func (m *Manager) AutoDiscover(certFile, keyFile, domain string) (*Certificate, error) {
	// First try manual configuration
	if certFile != "" && keyFile != "" {
		cert, err := m.LoadCertificate(certFile, keyFile, domain)
		if err == nil {
			return cert, nil
		}
	}

	// Try Let's Encrypt discovery
	cert, err := m.FindLetsEncryptCert(domain)
	if err == nil {
		return cert, nil
	}

	// Check SSL directory for local certificates
	localCertFile := filepath.Join(m.sslDir, "local", "cert.pem")
	localKeyFile := filepath.Join(m.sslDir, "local", "key.pem")
	if _, err := os.Stat(localCertFile); err == nil {
		if _, err := os.Stat(localKeyFile); err == nil {
			return m.LoadCertificate(localCertFile, localKeyFile, domain)
		}
	}

	return nil, fmt.Errorf("no SSL certificate found for domain: %s", domain)
}

// NeedsRenewal checks if a certificate needs renewal
// Certificates are renewed 30 days before expiry
func (c *Certificate) NeedsRenewal() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Until(c.ExpiresAt) < 30*24*time.Hour
}

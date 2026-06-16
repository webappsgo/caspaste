// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package validation

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TLSCertPaths holds paths to TLS certificate and key files
type TLSCertPaths struct {
	CertFile string
	KeyFile  string
	Domain   string
}

// FindLetsEncryptCerts searches for Let's Encrypt certificates
// Returns the first valid cert found, prioritizing matching domain
func FindLetsEncryptCerts(fqdn string) (*TLSCertPaths, error) {
	letsencryptDir := "/etc/letsencrypt/live"

	// Check if Let's Encrypt directory exists
	if _, err := os.Stat(letsencryptDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("let's encrypt directory not found: %s", letsencryptDir)
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
			return &TLSCertPaths{
				CertFile: certFile,
				KeyFile:  keyFile,
				Domain:   entry.Name(),
			}, nil
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
					return &TLSCertPaths{
						CertFile: certFile,
						KeyFile:  keyFile,
						Domain:   entry.Name(),
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no valid Let's Encrypt certificates found in %s", letsencryptDir)
}

// ValidateTLSCerts checks if TLS certificate files are valid
func ValidateTLSCerts(certFile, keyFile string) error {
	// Check if files exist
	if _, err := os.Stat(certFile); err != nil {
		return fmt.Errorf("certificate file not found: %s", certFile)
	}
	if _, err := os.Stat(keyFile); err != nil {
		return fmt.Errorf("key file not found: %s", keyFile)
	}

	// Try to load the certificate pair
	_, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("invalid TLS certificate/key pair: %w", err)
	}

	return nil
}

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"fmt"
	"net/http"
	"time"
)

// Pattern: /.well-known/security.txt
func (data *Data) handleSecurityTxt(rw http.ResponseWriter, req *http.Request) error {
	var content string

	// Use override if specified
	if data.SecurityTxt != "" {
		content = data.SecurityTxt
	} else {
		// Auto-generate from config
		content = generateSecurityTxt(data.SecurityContactEmail, data.SecurityContactName, data.FQDN)
	}

	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.Write([]byte(content))
	return nil
}

// generateSecurityTxt creates RFC 9116 compliant security.txt
// RFC 9116: https://www.rfc-editor.org/rfc/rfc9116.html
//
// Required fields:
//   - Contact: A way to contact the security team (email, URL, or phone)
//   - Expires: Date/time after which this file should be considered stale (max 1 year)
//
// Optional fields included:
//   - Canonical: The canonical URL for this security.txt file
//   - Preferred-Languages: Languages the security team prefers for communication
//   - Acknowledgments: Link to page recognizing security researchers
//   - Policy: Link to the security policy
func generateSecurityTxt(email, name, fqdn string) string {
	// Expires must be less than 1 year in the future per RFC 9116
	// Set to 1 year from now in ISO 8601 format with timezone
	expires := time.Now().AddDate(1, 0, 0).UTC().Format(time.RFC3339)

	canonical := fmt.Sprintf("https://%s/.well-known/security.txt", fqdn)
	acknowledgments := fmt.Sprintf("https://%s/server/about/authors", fqdn)
	policy := fmt.Sprintf("https://%s/server/about/security", fqdn)

	// RFC 9116 specifies field order doesn't matter, but conventionally:
	// Contact, Expires, then optional fields alphabetically
	return fmt.Sprintf(`# Security contact information for %s
# This file follows RFC 9116: https://www.rfc-editor.org/rfc/rfc9116.html

# REQUIRED FIELDS

# Contact: Security vulnerability reports should be sent here
# The security team (%s) monitors this address
Contact: mailto:%s

# Expires: This file is valid until the date below (max 1 year per RFC 9116)
Expires: %s

# OPTIONAL FIELDS

# Acknowledgments: Security researchers who have helped us
Acknowledgments: %s

# Canonical: The official location of this security.txt
Canonical: %s

# Policy: Our security disclosure policy
Policy: %s

# Preferred-Languages: We prefer reports in these languages
Preferred-Languages: en
`, fqdn, name, email, expires, acknowledgments, canonical, policy)
}

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// ETag and caching utilities per AI.md PART 9
package web

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// ETagFromContent generates an ETag from content bytes
// Uses SHA256 hash truncated to 16 chars for brevity
func ETagFromContent(content []byte) string {
	hash := sha256.Sum256(content)
	return `"` + hex.EncodeToString(hash[:8]) + `"`
}

// ETagFromString generates an ETag from a string
func ETagFromString(s string) string {
	return ETagFromContent([]byte(s))
}

// CheckETagMatch checks If-None-Match header and returns true if ETag matches
// Returns true if client has cached version (should return 304)
func CheckETagMatch(r *http.Request, etag string) bool {
	ifNoneMatch := r.Header.Get("If-None-Match")
	if ifNoneMatch == "" {
		return false
	}

	// Handle multiple ETags (comma-separated)
	for _, candidate := range strings.Split(ifNoneMatch, ",") {
		candidate = strings.TrimSpace(candidate)
		// Handle weak ETags (W/"...")
		candidate = strings.TrimPrefix(candidate, "W/")
		if candidate == etag || candidate == "*" {
			return true
		}
	}
	return false
}

// SetCacheHeaders sets appropriate cache headers for different content types
// contentType: "static" (1 year), "dynamic" (no-cache), "api" (short cache)
func SetCacheHeaders(w http.ResponseWriter, contentType string, etag string) {
	if etag != "" {
		w.Header().Set("ETag", etag)
	}

	switch contentType {
	case "static":
		// Static assets with fingerprint/hash - cache for 1 year
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case "dynamic":
		// Dynamic content - no cache
		w.Header().Set("Cache-Control", "no-store")
	case "api":
		// API responses - short cache
		w.Header().Set("Cache-Control", "public, max-age=60")
	case "private":
		// Private/authenticated content
		w.Header().Set("Cache-Control", "private, no-store")
	default:
		// Default - revalidate
		w.Header().Set("Cache-Control", "no-cache")
	}
}

// ServeWithETag serves content with ETag support
// Returns true if 304 Not Modified was sent (caller should not write body)
func ServeWithETag(w http.ResponseWriter, r *http.Request, content []byte, contentType string, cacheType string) bool {
	etag := ETagFromContent(content)

	// Check if client has cached version
	if CheckETagMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return true
	}

	// Set headers and serve
	w.Header().Set("Content-Type", contentType)
	SetCacheHeaders(w, cacheType, etag)
	w.Write(content)
	return false
}

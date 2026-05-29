// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"net/http"
	"os"
)

// defaultFavicon is a minimal 16x16 1-bit ICO file (70 bytes)
// This serves as the embedded default when no custom favicon is configured
var defaultFavicon = []byte{
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10,
	0x02, 0x00, 0x01, 0x00, 0x01, 0x00, 0x30, 0x00,
	0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x28, 0x00,
	0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x20, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0xff, 0xff, 0xff, 0x00,
}

// Pattern: /favicon.ico
// Per AI.md PART 16: Serve embedded default or custom favicon
func (data *Data) handleFavicon(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "image/x-icon")
	rw.Header().Set("Cache-Control", "public, max-age=86400")

	// Check for custom favicon path
	if data.Favicon != "" {
		// Try to read custom favicon from file
		customFavicon, err := os.ReadFile(data.Favicon)
		if err == nil {
			rw.Write(customFavicon)
			return nil
		}
		// Fall through to default if custom file not found
	}

	// Serve embedded default favicon
	rw.Write(defaultFavicon)
	return nil
}

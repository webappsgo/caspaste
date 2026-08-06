// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package raw

import (
	"encoding/base64"
	"io"
	"net/http"

	"github.com/webappsgo/caspaste/src/netshare"
)

// Pattern: /raw/
func (data *Data) rawHand(rw http.ResponseWriter, req *http.Request) error {
	// Check rate limit
	err := data.RateLimitGet.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Read DB
	pasteID := string([]rune(req.URL.Path)[5:])

	paste, err := data.DB.PasteGet(pasteID)
	if err != nil {
		return err
	}

	// If "one use" paste
	if paste.OneUse {
		// Delete paste
		err = data.DB.PasteDelete(pasteID)
		if err != nil {
			return err
		}
	}

	// Write result based on whether this is a file or regular paste
	if paste.IsFile {
		// File upload: try to decode base64, fall back to raw for legacy data
		var fileContent []byte
		fileData, err := base64.StdEncoding.DecodeString(paste.Body)
		if err != nil {
			// Legacy data stored without base64 encoding - use as-is
			fileContent = []byte(paste.Body)
		} else {
			fileContent = fileData
		}
		contentType := paste.MimeType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		rw.Header().Set("Content-Type", contentType)
		_, err = rw.Write(fileContent)
		if err != nil {
			return err
		}
	} else {
		// Regular paste: serve as plain text
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, err = io.WriteString(rw, paste.Body)
		if err != nil {
			return err
		}
	}

	return nil
}

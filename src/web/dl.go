// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	chromaLexers "github.com/alecthomas/chroma/v2/lexers"

	"github.com/casjay-forks/caspaste/src/netshare"
)

// Pattern: /dl/
func (data *Data) handleDownload(rw http.ResponseWriter, req *http.Request) error {
	// Check rate limit
	err := data.RateLimitGet.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Read DB
	pasteID := string([]rune(req.URL.Path)[4:])

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

	// Get create time
	createTime := time.Unix(paste.CreateTime, 0).UTC()

	// Determine content and filename based on whether this is a file upload
	var content string
	var fileName string
	var contentType string

	if paste.IsFile {
		// File upload: try to decode base64, fall back to raw for legacy data
		fileData, err := base64.StdEncoding.DecodeString(paste.Body)
		if err != nil {
			// Legacy data stored without base64 encoding - use as-is
			content = paste.Body
		} else {
			content = string(fileData)
		}
		fileName = paste.FileName
		if fileName == "" {
			fileName = paste.ID
		}
		contentType = paste.MimeType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	} else {
		// Regular paste: serve as-is with appropriate extension
		content = paste.Body
		fileName = paste.ID
		if paste.Title != "" {
			fileName = paste.Title
		}
		contentType = "application/octet-stream"

		// Get file extension from lexer (if available)
		fileExt := ".txt" // Default extension
		if lexer := chromaLexers.Get(paste.Syntax); lexer != nil {
			config := lexer.Config()
			if config != nil && len(config.Filenames) > 0 {
				fileExt = config.Filenames[0][1:] // Strip the leading "*"
			}
		}
		if !strings.HasSuffix(fileName, fileExt) {
			fileName = fileName + fileExt
		}
	}

	// Write result
	rw.Header().Set("Content-Type", contentType)
	rw.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	rw.Header().Set("Content-Transfer-Encoding", "binary")
	rw.Header().Set("Expires", "0")

	http.ServeContent(rw, req, fileName, createTime, strings.NewReader(content))

	return nil
}

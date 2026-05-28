
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package netshare

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/casjay-forks/caspaste/src/lineend"
	"github.com/casjay-forks/caspaste/src/storage"
	"github.com/casjay-forks/caspaste/src/validation"
)

func PasteAddFromForm(req *http.Request, db storage.DB, rateSys *RateLimitSystem, titleMaxLen int, bodyMaxLen int, maxLifeTime int64, lexerNames []string) (string, int64, int64, error) {
	// Check HTTP method
	if req.Method != "POST" {
		return "", 0, 0, ErrMethodNotAllowed
	}

	// Check rate limit
	err := rateSys.CheckAndUse(GetClientAddr(req))
	if err != nil {
		return "", 0, 0, err
	}

	// Parse request body — supports application/json, multipart/form-data,
	// and application/x-www-form-urlencoded
	ct := req.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		// Decode JSON body into typed struct then populate req.PostForm so all
		// downstream validation logic runs unchanged
		var j struct {
			Title       string `json:"title"`
			Body        string `json:"body"`
			Syntax      string `json:"syntax"`
			Expiration  int64  `json:"expiration"`
			OneUse      bool   `json:"oneUse"`
			Author      string `json:"author"`
			AuthorEmail string `json:"authorEmail"`
			AuthorURL   string `json:"authorURL"`
			Editable    bool   `json:"editable"`
			Private     bool   `json:"private"`
			IsURL       bool   `json:"url"`
			OriginalURL string `json:"originalURL"`
			LineEnd     string `json:"lineEnd"`
		}
		if err = json.NewDecoder(req.Body).Decode(&j); err != nil {
			return "", 0, 0, ErrBadRequest
		}
		v := url.Values{}
		v.Set("title", j.Title)
		v.Set("body", j.Body)
		v.Set("syntax", j.Syntax)
		v.Set("author", j.Author)
		v.Set("authorEmail", j.AuthorEmail)
		v.Set("authorURL", j.AuthorURL)
		v.Set("originalURL", j.OriginalURL)
		v.Set("lineEnd", j.LineEnd)
		if j.Expiration > 0 {
			v.Set("expiration", strconv.FormatInt(j.Expiration, 10))
		}
		if j.OneUse {
			v.Set("oneUse", "true")
		}
		if j.Editable {
			v.Set("editable", "true")
		}
		if j.Private {
			v.Set("private", "true")
		}
		if j.IsURL {
			v.Set("url", "true")
		}
		req.Form = v
		req.PostForm = v
	} else {
		// ParseForm handles application/x-www-form-urlencoded
		if err = req.ParseForm(); err != nil {
			return "", 0, 0, err
		}
		// ParseMultipartForm handles multipart/form-data; 50 MB max
		req.ParseMultipartForm(52428800)
	}

	paste := storage.Paste{
		Title:       req.PostFormValue("title"),
		Body:        req.PostFormValue("body"),
		Syntax:      req.PostFormValue("syntax"),
		DeleteTime:  0,
		OneUse:      false,
		Author:      req.PostFormValue("author"),
		AuthorEmail: req.PostFormValue("authorEmail"),
		AuthorURL:   req.PostFormValue("authorURL"),
		IsEditable:  validation.IsTruthy(req.PostFormValue("editable")),
		IsPrivate:   validation.IsTruthy(req.PostFormValue("private")),
		IsURL:       validation.IsTruthy(req.PostFormValue("url")),
		OriginalURL: req.PostFormValue("originalURL"),
	}

	// Handle file upload
	file, handler, err := req.FormFile("file")
	if err == nil {
		defer file.Close()

		// Read file contents
		fileData, err := io.ReadAll(file)
		if err != nil {
			return "", 0, 0, err
		}

		// Set file fields
		paste.IsFile = true
		paste.FileName = handler.Filename
		paste.MimeType = handler.Header.Get("Content-Type")
		if paste.MimeType == "" {
			paste.MimeType = "application/octet-stream"
		}

		// Store file data as base64 in Body field to handle binary data safely
		// This prevents UTF-8 encoding errors in databases like PostgreSQL
		paste.Body = base64.StdEncoding.EncodeToString(fileData)

		// Default syntax for files (use plaintext as it's always valid)
		if paste.Syntax == "" {
			paste.Syntax = "plaintext"
		}
	}

	// Remove new line from title
	paste.Title = strings.Replace(paste.Title, "\n", "", -1)
	paste.Title = strings.Replace(paste.Title, "\r", "", -1)
	paste.Title = strings.Replace(paste.Title, "\t", " ", -1)

	// Check title
	if utf8.RuneCountInString(paste.Title) > titleMaxLen && titleMaxLen >= 0 {
		return "", 0, 0, ErrPayloadTooLarge
	}

	// Check paste body (allow empty for URL shortener)
	if paste.Body == "" && !paste.IsURL {
		return "", 0, 0, ErrBadRequest
	}
	
	// For URL shortener, validate originalURL is provided
	if paste.IsURL && paste.OriginalURL == "" {
		return "", 0, 0, ErrBadRequest
	}

	if utf8.RuneCountInString(paste.Body) > bodyMaxLen && bodyMaxLen > 0 {
		return "", 0, 0, ErrPayloadTooLarge
	}

	// Change paste body lines end (skip for file uploads to preserve binary data)
	if !paste.IsFile {
		switch req.PostForm.Get("lineEnd") {
		case "", "LF", "lf":
			paste.Body = lineend.UnknownToUnix(paste.Body)

		case "CRLF", "crlf":
			paste.Body = lineend.UnknownToDos(paste.Body)

		case "CR", "cr":
			paste.Body = lineend.UnknownToOldMac(paste.Body)

		default:
			return "", 0, 0, ErrBadRequest
		}
	}

	// Check syntax
	if paste.Syntax == "" {
		paste.Syntax = "plaintext"
	}

	// Validate syntax (allow "autodetect" as special value)
	// Syntax matching is case-insensitive for user convenience
	syntaxOk := false
	if strings.EqualFold(paste.Syntax, "autodetect") {
		syntaxOk = true
		paste.Syntax = "autodetect"
	} else {
		for _, name := range lexerNames {
			if strings.EqualFold(name, paste.Syntax) {
				syntaxOk = true
				// Normalize to the official lexer name for proper highlighting
				paste.Syntax = name
				break
			}
		}
	}

	if !syntaxOk {
		return "", 0, 0, ErrBadRequest
	}

	// Get delete time
	expirStr := req.PostForm.Get("expiration")
	if expirStr != "" {
		// Convert string to int
		expir, err := strconv.ParseInt(expirStr, 10, 64)
		if err != nil {
			return "", 0, 0, ErrBadRequest
		}

		// Check limits
		if maxLifeTime > 0 {
			if expir > maxLifeTime || expir <= 0 {
				return "", 0, 0, ErrBadRequest
			}
		}

		// Save if ok
		if expir > 0 {
			paste.DeleteTime = time.Now().Unix() + expir
		}
	}

	// Get "one use" (burn after reading) parameter
	// Accepts "true" for backward compatibility or numeric values for view count
	oneUseVal := req.PostForm.Get("oneUse")
	if validation.IsTruthy(oneUseVal) {
		paste.OneUse = true
	} else if oneUseVal != "" && oneUseVal != "false" {
		// Check if it's a numeric value > 0 (custom view count)
		if viewCount, err := strconv.Atoi(oneUseVal); err == nil && viewCount > 0 {
			paste.OneUse = true
		}
	}

	// Check author name, email and URL length.
	if utf8.RuneCountInString(paste.Author) > MaxLengthAuthorAll {
		return "", 0, 0, ErrPayloadTooLarge
	}

	if utf8.RuneCountInString(paste.AuthorEmail) > MaxLengthAuthorAll {
		return "", 0, 0, ErrPayloadTooLarge
	}

	if utf8.RuneCountInString(paste.AuthorURL) > MaxLengthAuthorAll {
		return "", 0, 0, ErrPayloadTooLarge
	}

	// Validate Author URL scheme to prevent XSS via javascript: or data: URLs
	if paste.AuthorURL != "" {
		// Convert to lowercase for comparison
		urlLower := strings.ToLower(strings.TrimSpace(paste.AuthorURL))

		// Only allow http:// and https:// schemes
		if !strings.HasPrefix(urlLower, "http://") && !strings.HasPrefix(urlLower, "https://") {
			return "", 0, 0, ErrBadRequest
		}

		// Prevent data:, javascript:, vbscript:, file:, etc.
		if strings.Contains(urlLower, "javascript:") ||
		   strings.Contains(urlLower, "data:") ||
		   strings.Contains(urlLower, "vbscript:") ||
		   strings.Contains(urlLower, "file:") {
			return "", 0, 0, ErrBadRequest
		}
	}
	
	// Validate OriginalURL scheme for URL shortener
	if paste.IsURL && paste.OriginalURL != "" {
		urlLower := strings.ToLower(strings.TrimSpace(paste.OriginalURL))
		
		// Only allow http:// and https:// schemes
		if !strings.HasPrefix(urlLower, "http://") && !strings.HasPrefix(urlLower, "https://") {
			return "", 0, 0, ErrBadRequest
		}
		
		// Prevent data:, javascript:, vbscript:, file:, etc.
		if strings.Contains(urlLower, "javascript:") ||
		   strings.Contains(urlLower, "data:") ||
		   strings.Contains(urlLower, "vbscript:") ||
		   strings.Contains(urlLower, "file:") {
			return "", 0, 0, ErrBadRequest
		}
	}

	// Create paste
	pasteID, createTime, deleteTime, err := db.PasteAdd(paste)
	if err != nil {
		return pasteID, createTime, deleteTime, err
	}

	return pasteID, createTime, deleteTime, nil
}

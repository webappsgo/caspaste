
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package apiv1

// External API Compatibility - Create endpoints only per AI.md PART 14
// Supports: termbin, sprunge, ix.io, pastebin.com, stikked, microbin, lenpaste, hastebin
//
// Per AI.md "External API Compatibility":
// - Match the exact response format of the target service
// - Preserve response content-type (JSON, XML, plain text, etc.)
// - Honor explicit Accept headers when provided

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/casjay-forks/caspaste/src/httputil"
	"github.com/casjay-forks/caspaste/src/lineend"
	"github.com/casjay-forks/caspaste/src/netshare"
	"github.com/casjay-forks/caspaste/src/storage"
	"github.com/casjay-forks/caspaste/src/validation"
)

// lenpasteResponse matches the lenpaste /api/v1/new response exactly (flat, no envelope).
type lenpasteResponse struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	Syntax     string `json:"syntax"`
	CreateTime int64  `json:"createTime"`
	DeleteTime int64  `json:"deleteTime"`
}

// stikkedResponse matches the stikked /api/create response format.
type stikkedResponse struct {
	Status string `json:"status"`
	URL    string `json:"url"`
}

// hastebinResponse matches the hastebin /documents response format.
type hastebinResponse struct {
	Key string `json:"key"`
}

// compatResponse holds paste creation response data
type compatResponse struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	CreateTime int64  `json:"createTime"`
	DeleteTime int64  `json:"deleteTime"`
}

// handleCompat routes compatibility endpoints
func (data *Data) handleCompat(rw http.ResponseWriter, req *http.Request) error {
	path := req.URL.Path

	switch {
	// sprunge.us compatibility
	// POST / with form field "sprunge"
	// Original returns: plain text URL
	case path == "/sprunge" || path == "/sprunge/":
		return data.handleSprungeCompat(rw, req)

	// ix.io compatibility
	// POST / with form field "f:1"
	// Original returns: plain text URL
	case path == "/ix" || path == "/ix/":
		return data.handleIxCompat(rw, req)

	// pastebin.com compatibility
	// POST /api/api_post.php
	// Original returns: plain text paste key or URL
	case path == "/api/api_post.php":
		return data.handlePastebinCompat(rw, req)

	// stikked/stiqued compatibility
	// POST /api/create
	// Original returns: JSON with url field
	case path == "/api/create":
		return data.handleStikkedCompat(rw, req)

	// microbin compatibility
	// POST /upload or /p
	// Original returns: redirect or plain text
	case path == "/upload" || path == "/p":
		return data.handleMicrobinCompat(rw, req)

	// lenpaste compatibility
	// POST /api/v1/new
	// Original returns: JSON
	case path == "/api/v1/new":
		return data.handleLenpasteCompat(rw, req)

	// termbin/netcat style - raw body
	// POST /termbin
	// Original returns: plain text URL
	case path == "/termbin" || path == "/nc":
		return data.handleTermbinCompat(rw, req)

	// hastebin compatibility
	// POST /documents — create; GET /documents/{key} — retrieve
	case path == "/documents" || strings.HasPrefix(path, "/documents/"):
		return data.handleHastebinCompat(rw, req)

	// Generic compatibility - accept multiple field names
	// POST /compat or /paste
	// Uses content negotiation
	case path == "/compat" || path == "/paste":
		return data.handleGenericCompat(rw, req)

	default:
		return netshare.ErrNotFound
	}
}

// writeCompatResponse writes response with content negotiation
// defaultFormat is the format the original service uses
func writeCompatResponse(rw http.ResponseWriter, req *http.Request, resp compatResponse, defaultFormat httputil.ResponseFormat) {
	// Check for explicit Accept header override per AI.md PART 14
	accept := req.Header.Get("Accept")
	format := defaultFormat

	// Check .txt extension first
	if strings.HasSuffix(req.URL.Path, ".txt") {
		format = httputil.FormatText
	} else if strings.Contains(accept, "application/json") {
		format = httputil.FormatJSON
	} else if strings.Contains(accept, "text/plain") {
		format = httputil.FormatText
	}

	switch format {
	case httputil.FormatJSON:
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		jsonResp := APIResponse{
			OK:   true,
			Data: resp,
		}
		jsonData, _ := json.MarshalIndent(jsonResp, "", "  ")
		rw.Write(jsonData)
		rw.Write([]byte("\n"))
	default:
		// Plain text - just the URL (matches original services)
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.Write([]byte(resp.URL + "\n"))
	}
}

// handleSprungeCompat handles sprunge.us style paste creation
// POST with form field "sprunge=<content>"
// Original returns: plain text URL
func (data *Data) handleSprungeCompat(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return netshare.ErrMethodNotAllowed
	}

	err := data.RateLimitNew.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Parse both URL-encoded and multipart forms
	req.ParseForm()
	req.ParseMultipartForm(52428800)

	body := req.PostFormValue("sprunge")
	if body == "" {
		return netshare.ErrBadRequest
	}

	paste := storage.Paste{
		Body:   lineend.UnknownToUnix(body),
		Syntax: "plaintext",
	}

	pasteID, createTime, deleteTime, err := data.DB.PasteAdd(paste)
	if err != nil {
		return err
	}

	resp := compatResponse{
		ID:         pasteID,
		URL:        netshare.BuildPasteURL(req, pasteID),
		CreateTime: createTime,
		DeleteTime: deleteTime,
	}

	// sprunge returns plain text by default
	writeCompatResponse(rw, req, resp, httputil.FormatText)
	return nil
}

// handleIxCompat handles ix.io style paste creation
// POST with form field "f:1=<content>"
// Original returns: plain text URL
func (data *Data) handleIxCompat(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return netshare.ErrMethodNotAllowed
	}

	err := data.RateLimitNew.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Parse both URL-encoded and multipart forms
	req.ParseForm()
	req.ParseMultipartForm(52428800)

	// ix.io uses "f:1" as field name
	body := req.PostFormValue("f:1")
	if body == "" {
		// Also try "f:0" and just "f"
		body = req.PostFormValue("f:0")
		if body == "" {
			body = req.PostFormValue("f")
		}
	}

	if body == "" {
		return netshare.ErrBadRequest
	}

	paste := storage.Paste{
		Body:   lineend.UnknownToUnix(body),
		Syntax: "plaintext",
	}

	pasteID, createTime, deleteTime, err := data.DB.PasteAdd(paste)
	if err != nil {
		return err
	}

	resp := compatResponse{
		ID:         pasteID,
		URL:        netshare.BuildPasteURL(req, pasteID),
		CreateTime: createTime,
		DeleteTime: deleteTime,
	}

	// ix.io returns plain text by default
	writeCompatResponse(rw, req, resp, httputil.FormatText)
	return nil
}

// handlePastebinCompat handles pastebin.com style paste creation
// POST /api/api_post.php with various params
// Original returns: plain text paste key or URL
func (data *Data) handlePastebinCompat(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return netshare.ErrMethodNotAllowed
	}

	err := data.RateLimitNew.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Parse both URL-encoded and multipart forms
	req.ParseForm()
	req.ParseMultipartForm(52428800)

	// pastebin.com parameters
	body := req.PostFormValue("api_paste_code")
	if body == "" {
		return netshare.ErrBadRequest
	}

	syntax := req.PostFormValue("api_paste_format")
	if syntax == "" {
		syntax = "plaintext"
	}

	title := req.PostFormValue("api_paste_name")
	expireStr := req.PostFormValue("api_paste_expire_date")
	private := req.PostFormValue("api_paste_private")

	paste := storage.Paste{
		Title:     title,
		Body:      lineend.UnknownToUnix(body),
		Syntax:    normalizeSyntax(syntax, data.Lexers),
		// Pastebin.com uses 0=public, 1=unlisted, 2=private (both 1 and 2 are private)
		IsPrivate: validation.IsTruthy(private) || private == "2",
	}

	// Parse expiration (pastebin format: N, 10M, 1H, 1D, 1W, 1M, 6M, 1Y)
	if expireStr != "" && expireStr != "N" {
		seconds := parsePastebinExpire(expireStr)
		if seconds > 0 {
			paste.DeleteTime = time.Now().Unix() + seconds
		}
	}

	pasteID, createTime, deleteTime, err := data.DB.PasteAdd(paste)
	if err != nil {
		return err
	}

	resp := compatResponse{
		ID:         pasteID,
		URL:        netshare.BuildPasteURL(req, pasteID),
		CreateTime: createTime,
		DeleteTime: deleteTime,
	}

	// pastebin.com returns plain text by default
	writeCompatResponse(rw, req, resp, httputil.FormatText)
	return nil
}

// handleStikkedCompat handles stikked/stiqued style paste creation
// POST /api/create
// Original returns: JSON with url field
func (data *Data) handleStikkedCompat(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return netshare.ErrMethodNotAllowed
	}

	err := data.RateLimitNew.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Parse both URL-encoded and multipart forms
	req.ParseForm()
	req.ParseMultipartForm(52428800)

	// stikked parameters
	body := req.PostFormValue("text")
	if body == "" {
		body = req.PostFormValue("code")
	}
	if body == "" {
		body = req.PostFormValue("data")
	}
	if body == "" {
		return netshare.ErrBadRequest
	}

	syntax := req.PostFormValue("lang")
	if syntax == "" {
		syntax = req.PostFormValue("language")
	}
	if syntax == "" {
		syntax = "plaintext"
	}

	title := req.PostFormValue("title")
	if title == "" {
		title = req.PostFormValue("name")
	}

	paste := storage.Paste{
		Title:  title,
		Body:   lineend.UnknownToUnix(body),
		Syntax: normalizeSyntax(syntax, data.Lexers),
	}

	// Parse expiration
	expireStr := req.PostFormValue("expire")
	if expireStr != "" {
		if seconds, err := strconv.ParseInt(expireStr, 10, 64); err == nil && seconds > 0 {
			paste.DeleteTime = time.Now().Unix() + seconds
		}
	}

	pasteID, _, _, err := data.DB.PasteAdd(paste)
	if err != nil {
		return err
	}

	// stikked returns {"status":"success","url":"..."} — flat JSON, no envelope
	resp := stikkedResponse{
		Status: "success",
		URL:    netshare.BuildPasteURL(req, pasteID),
	}
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	jsonData, _ := json.MarshalIndent(resp, "", "  ")
	rw.Write(jsonData)
	rw.Write([]byte("\n"))
	return nil
}

// handleMicrobinCompat handles microbin style paste creation
// POST /upload or /p
// Original returns: redirect or plain text
func (data *Data) handleMicrobinCompat(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return netshare.ErrMethodNotAllowed
	}

	err := data.RateLimitNew.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	err = req.ParseForm()
	if err != nil {
		return err
	}
	req.ParseMultipartForm(52428800)

	// microbin parameters
	body := req.PostFormValue("content")
	if body == "" {
		body = req.PostFormValue("text")
	}
	if body == "" {
		body = req.PostFormValue("editordata")
	}

	if body == "" {
		return netshare.ErrBadRequest
	}

	syntax := req.PostFormValue("syntax")
	if syntax == "" {
		syntax = "plaintext"
	}

	paste := storage.Paste{
		Title:     req.PostFormValue("title"),
		Body:      lineend.UnknownToUnix(body),
		Syntax:    normalizeSyntax(syntax, data.Lexers),
		OneUse:    validation.IsTruthy(req.PostFormValue("burn")),
		IsPrivate: validation.IsTruthy(req.PostFormValue("private")),
	}

	// Parse expiration
	expireStr := req.PostFormValue("expiration")
	if expireStr != "" {
		if seconds, err := strconv.ParseInt(expireStr, 10, 64); err == nil && seconds > 0 {
			paste.DeleteTime = time.Now().Unix() + seconds
		}
	}

	pasteID, _, _, err := data.DB.PasteAdd(paste)
	if err != nil {
		return err
	}

	pasteURL := netshare.BuildPasteURL(req, pasteID)

	// microbin web form posts return a 303 redirect to the new paste URL.
	// JSON API clients (Accept: application/json) get a JSON body instead.
	if strings.Contains(req.Header.Get("Accept"), "application/json") {
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		jsonData, _ := json.MarshalIndent(map[string]string{"id": pasteID, "url": pasteURL}, "", "  ")
		rw.Write(jsonData)
		rw.Write([]byte("\n"))
		return nil
	}

	http.Redirect(rw, req, pasteURL, http.StatusSeeOther)
	return nil
}

// handleLenpasteCompat handles lenpaste style paste creation.
// POST /api/v1/new
// lenpaste returns a flat JSON object (no envelope) with fields:
//
//	{"id","url","title","syntax","createTime","deleteTime"}
//
// Expiration: "lifetime" field is a Go duration string ("1h", "10m", "0"=never).
// Falls back to "expiration" as integer seconds for backward compat.
func (data *Data) handleLenpasteCompat(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return netshare.ErrMethodNotAllowed
	}

	err := data.RateLimitNew.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Parse both URL-encoded and multipart forms
	req.ParseForm()
	req.ParseMultipartForm(52428800)

	syntax := req.PostFormValue("syntax")
	if syntax == "" {
		syntax = "plaintext"
	}
	syntax = normalizeSyntax(syntax, data.Lexers)

	paste := storage.Paste{
		Title:     req.PostFormValue("title"),
		Syntax:    syntax,
		OneUse:    validation.IsTruthy(req.PostFormValue("oneUse")),
		Author:    req.PostFormValue("author"),
		AuthorURL: req.PostFormValue("authorURL"),
	}

	// Check for file upload first
	file, handler, fileErr := req.FormFile("file")
	if fileErr == nil {
		defer file.Close()

		fileData, err := io.ReadAll(file)
		if err != nil {
			return err
		}

		paste.IsFile = true
		paste.FileName = handler.Filename
		paste.MimeType = handler.Header.Get("Content-Type")
		if paste.MimeType == "" {
			paste.MimeType = "application/octet-stream"
		}
		paste.Body = base64.StdEncoding.EncodeToString(fileData)
	} else {
		body := req.PostFormValue("body")
		if body == "" {
			return netshare.ErrBadRequest
		}
		paste.Body = lineend.UnknownToUnix(body)
	}

	// Parse expiration: "lifetime" is a Go duration string; fallback to "expiration" seconds.
	if lifetimeStr := req.PostFormValue("lifetime"); lifetimeStr != "" && lifetimeStr != "0" {
		if d, err := time.ParseDuration(lifetimeStr); err == nil && d > 0 {
			paste.DeleteTime = time.Now().Unix() + int64(d.Seconds())
		}
	} else if expireStr := req.PostFormValue("expiration"); expireStr != "" {
		if seconds, err := strconv.ParseInt(expireStr, 10, 64); err == nil && seconds > 0 {
			paste.DeleteTime = time.Now().Unix() + seconds
		}
	}

	pasteID, createTime, deleteTime, err := data.DB.PasteAdd(paste)
	if err != nil {
		return err
	}

	// lenpaste returns a flat JSON object — no "ok"/"data" envelope.
	resp := lenpasteResponse{
		ID:         pasteID,
		URL:        netshare.BuildPasteURL(req, pasteID),
		Title:      paste.Title,
		Syntax:     paste.Syntax,
		CreateTime: createTime,
		DeleteTime: deleteTime,
	}
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	jsonData, _ := json.MarshalIndent(resp, "", "  ")
	rw.Write(jsonData)
	rw.Write([]byte("\n"))
	return nil
}

// handleTermbinCompat handles termbin/netcat style paste creation
// POST with raw body (no form fields)
// Original returns: plain text URL
func (data *Data) handleTermbinCompat(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" && req.Method != "PUT" {
		return netshare.ErrMethodNotAllowed
	}

	err := data.RateLimitNew.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Read raw body
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}

	body := string(bodyBytes)
	if body == "" {
		return netshare.ErrBadRequest
	}

	paste := storage.Paste{
		Body:   lineend.UnknownToUnix(body),
		Syntax: "plaintext",
	}

	pasteID, createTime, deleteTime, err := data.DB.PasteAdd(paste)
	if err != nil {
		return err
	}

	resp := compatResponse{
		ID:         pasteID,
		URL:        netshare.BuildPasteURL(req, pasteID),
		CreateTime: createTime,
		DeleteTime: deleteTime,
	}

	// termbin returns plain text by default
	writeCompatResponse(rw, req, resp, httputil.FormatText)
	return nil
}

// handleGenericCompat handles generic paste creation with multiple field name support
// Accepts: text, content, code, data, body, paste, snippet
// Uses content negotiation per AI.md PART 14
func (data *Data) handleGenericCompat(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return netshare.ErrMethodNotAllowed
	}

	err := data.RateLimitNew.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Parse both URL-encoded and multipart forms
	req.ParseForm()
	req.ParseMultipartForm(52428800)

	// Try multiple field names
	body := ""
	fieldNames := []string{"text", "content", "code", "data", "body", "paste", "snippet", "sprunge", "f:1"}
	for _, name := range fieldNames {
		body = req.PostFormValue(name)
		if body != "" {
			break
		}
	}

	if body == "" {
		return netshare.ErrBadRequest
	}

	// Try multiple syntax field names
	syntax := ""
	syntaxNames := []string{"syntax", "lang", "language", "type", "format", "lexer"}
	for _, name := range syntaxNames {
		syntax = req.PostFormValue(name)
		if syntax != "" {
			break
		}
	}
	if syntax == "" {
		syntax = "plaintext"
	}

	// Try multiple title field names
	title := ""
	titleNames := []string{"title", "name", "filename"}
	for _, name := range titleNames {
		title = req.PostFormValue(name)
		if title != "" {
			break
		}
	}

	paste := storage.Paste{
		Title:  title,
		Body:   lineend.UnknownToUnix(body),
		Syntax: normalizeSyntax(syntax, data.Lexers),
	}

	// Parse expiration from various field names
	expireStr := ""
	expireNames := []string{"expiration", "expire", "ttl", "lifetime"}
	for _, name := range expireNames {
		expireStr = req.PostFormValue(name)
		if expireStr != "" {
			break
		}
	}
	if expireStr != "" {
		if seconds, err := strconv.ParseInt(expireStr, 10, 64); err == nil && seconds > 0 {
			paste.DeleteTime = time.Now().Unix() + seconds
		}
	}

	// Check burn/oneUse
	burnNames := []string{"oneUse", "burn", "burnAfterReading", "once"}
	for _, name := range burnNames {
		val := req.PostFormValue(name)
		if validation.IsTruthy(val) {
			paste.OneUse = true
			break
		}
	}

	// Check private
	privateNames := []string{"private", "unlisted", "secret"}
	for _, name := range privateNames {
		val := req.PostFormValue(name)
		if validation.IsTruthy(val) {
			paste.IsPrivate = true
			break
		}
	}

	pasteID, createTime, deleteTime, err := data.DB.PasteAdd(paste)
	if err != nil {
		return err
	}

	resp := compatResponse{
		ID:         pasteID,
		URL:        netshare.BuildPasteURL(req, pasteID),
		CreateTime: createTime,
		DeleteTime: deleteTime,
	}

	// For generic compat, use standard content negotiation
	format := httputil.GetAPIResponseFormat(req)
	writeCompatResponse(rw, req, resp, format)
	return nil
}

// Helper functions

// normalizeSyntax validates and normalizes syntax to a known lexer
func normalizeSyntax(syntax string, lexers []string) string {
	if syntax == "" {
		return "plaintext"
	}

	// Check exact match (case-insensitive)
	for _, name := range lexers {
		if strings.EqualFold(name, syntax) {
			return name
		}
	}

	// Common aliases
	aliases := map[string]string{
		"plain":      "plaintext",
		"txt":        "plaintext",
		"text":       "plaintext",
		"py":         "python",
		"python3":    "python",
		"js":         "javascript",
		"ts":         "typescript",
		"rb":         "ruby",
		"sh":         "bash",
		"shell":      "bash",
		"zsh":        "bash",
		"c++":        "cpp",
		"cplusplus":  "cpp",
		"cs":         "csharp",
		"c#":         "csharp",
		"golang":     "go",
		"yml":        "yaml",
		"md":         "markdown",
		"pl":         "perl",
		"rs":         "rust",
		"kt":         "kotlin",
		"swift":      "swift",
		"dockerfile": "docker",
		"makefile":   "make",
	}

	syntaxLower := strings.ToLower(syntax)
	if alias, ok := aliases[syntaxLower]; ok {
		// Verify alias exists in lexers
		for _, name := range lexers {
			if strings.EqualFold(name, alias) {
				return name
			}
		}
	}

	return "plaintext"
}

// parsePastebinExpire parses pastebin.com expiration format
// N=never, 10M=10min, 1H=1hour, 1D=1day, 1W=1week, 2W=2weeks, 1M=1month, 6M=6months, 1Y=1year
func parsePastebinExpire(s string) int64 {
	s = strings.ToUpper(s)

	switch s {
	case "N":
		return 0
	case "10M":
		return 10 * 60
	case "1H":
		return 60 * 60
	case "1D":
		return 24 * 60 * 60
	case "1W":
		return 7 * 24 * 60 * 60
	case "2W":
		return 14 * 24 * 60 * 60
	case "1M":
		return 30 * 24 * 60 * 60
	case "6M":
		return 180 * 24 * 60 * 60
	case "1Y":
		return 365 * 24 * 60 * 60
	default:
		return 0
	}
}

// hastebinGetResponse is the response body for GET /documents/{key}.
type hastebinGetResponse struct {
	Key  string `json:"key"`
	Data string `json:"data"`
}

// handleHastebinCompat handles hastebin-compatible paste creation and retrieval.
//
// POST /documents — raw body → {"key":"xxxxx"}
// GET  /documents/{key}    → {"key":"xxxxx","data":"paste content"}
//
// Note: GET /raw/{key} is already handled by the existing /raw/ handler,
// which returns the raw paste body as plain text — identical to hastebin /raw/{key}.
func (data *Data) handleHastebinCompat(rw http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case "POST":
		return data.hastebinCreate(rw, req)
	case "GET":
		return data.hastebinGet(rw, req)
	default:
		return netshare.ErrMethodNotAllowed
	}
}

// hastebinCreate handles POST /documents (raw body → {"key":"xxxxx"}).
func (data *Data) hastebinCreate(rw http.ResponseWriter, req *http.Request) error {
	err := data.RateLimitNew.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Hastebin POSTs the raw paste body — no form encoding.
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}

	body := string(bodyBytes)
	if body == "" {
		return netshare.ErrBadRequest
	}

	paste := storage.Paste{
		Body:   lineend.UnknownToUnix(body),
		Syntax: "plaintext",
	}

	pasteID, _, _, err := data.DB.PasteAdd(paste)
	if err != nil {
		return err
	}

	// Hastebin returns {"key":"xxxxx"} — flat JSON, no envelope.
	resp := hastebinResponse{Key: pasteID}
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	jsonData, _ := json.MarshalIndent(resp, "", "  ")
	rw.Write(jsonData)
	rw.Write([]byte("\n"))
	return nil
}

// hastebinGet handles GET /documents/{key} → {"key":"xxxxx","data":"paste content"}.
func (data *Data) hastebinGet(rw http.ResponseWriter, req *http.Request) error {
	err := data.RateLimitGet.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Extract key from path: /documents/{key}
	key := strings.TrimPrefix(req.URL.Path, "/documents/")
	key = strings.TrimSuffix(key, "/")
	if key == "" {
		return netshare.ErrBadRequest
	}

	paste, err := data.DB.PasteGet(key)
	if err != nil {
		return err
	}

	// Burn-after-reading: delete after first view
	if paste.OneUse {
		_ = data.DB.PasteDelete(key)
	}

	// Hastebin returns {"key":"xxxxx","data":"content"} — flat JSON, no envelope.
	resp := hastebinGetResponse{Key: key, Data: paste.Body}
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	jsonData, _ := json.MarshalIndent(resp, "", "  ")
	rw.Write(jsonData)
	rw.Write([]byte("\n"))
	return nil
}

// writeCompatError writes an error response for compat endpoints
func writeCompatError(rw http.ResponseWriter, req *http.Request, code int, errCode, message string, defaultFormat httputil.ResponseFormat) {
	accept := req.Header.Get("Accept")
	format := defaultFormat

	if strings.Contains(accept, "application/json") {
		format = httputil.FormatJSON
	} else if strings.Contains(accept, "text/plain") {
		format = httputil.FormatText
	}

	rw.WriteHeader(code)

	switch format {
	case httputil.FormatJSON:
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		jsonResp := APIResponse{
			OK:      false,
			Error:   errCode,
			Message: message,
		}
		jsonData, _ := json.MarshalIndent(jsonResp, "", "  ")
		rw.Write(jsonData)
		rw.Write([]byte("\n"))
	default:
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(rw, "ERROR: %s: %s\n", errCode, message)
	}
}

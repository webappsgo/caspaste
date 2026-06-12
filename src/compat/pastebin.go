// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package compat

import (
	"net/http"
	"strings"
	"time"

	"github.com/casjay-forks/caspaste/src/storage"
)

// handlePastebin intercepts pastebin.com-compatible API paths and returns true if handled.
//
// Pastebin.com API surface:
//
//	POST /api/api_post.php     → dispatch on api_option:
//	  api_option=paste         → create paste → plain-text URL
//	  api_option=delete        → delete paste by api_paste_key → "Paste Removed"
//	  api_option=list          → list public pastes → XML list
//	  api_option=trends        → list trending pastes → XML list
//	GET  /api/api_raw.php?i=   → raw paste text
func (d *Data) handlePastebin(rw http.ResponseWriter, req *http.Request) bool {
	switch req.URL.Path {
	case "/api/api_post.php":
		d.pastebinPost(rw, req)
		return true
	case "/api/api_raw.php":
		d.pastebinRaw(rw, req)
		return true
	}
	return false
}

// pastebinPost dispatches POST /api/api_post.php based on api_option.
func (d *Data) pastebinPost(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Bad API request", http.StatusMethodNotAllowed)
		return
	}
	if err := req.ParseForm(); err != nil {
		http.Error(rw, "Bad API request", http.StatusBadRequest)
		return
	}

	switch strings.ToLower(req.PostFormValue("api_option")) {
	case "paste":
		d.pastebinCreate(rw, req)
	case "delete":
		d.pastebinDelete(rw, req)
	case "list":
		d.pastebinList(rw, req)
	case "trends":
		d.pastebinTrends(rw, req)
	default:
		http.Error(rw, "Bad API request", http.StatusBadRequest)
	}
}

// pastebinCreate handles api_option=paste.
// Params: api_paste_code (body), api_paste_name (title), api_paste_format (syntax),
// api_paste_expire_date, api_paste_private (0=public, 1=unlisted, 2=private)
// Response: plain-text URL.
func (d *Data) pastebinCreate(rw http.ResponseWriter, req *http.Request) {
	if d.checkRateLimit(rw, req) {
		return
	}
	body := req.PostFormValue("api_paste_code")
	if body == "" {
		http.Error(rw, "Bad API request: empty paste", http.StatusBadRequest)
		return
	}

	title := req.PostFormValue("api_paste_name")
	syntax := req.PostFormValue("api_paste_format")
	if syntax == "" {
		syntax = "text"
	}

	isPrivate := req.PostFormValue("api_paste_private") == "2"
	deleteTime := pastebinExpiry(req.PostFormValue("api_paste_expire_date"))

	paste := storage.Paste{
		Title:      title,
		Body:       body,
		Syntax:     syntax,
		DeleteTime: deleteTime,
		IsPrivate:  isPrivate,
	}

	id, _, _, err := d.DB.PasteAdd(paste)
	if err != nil {
		d.Log.Error(err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(d.BaseURL + "/" + id + "\n"))
}

// pastebinDelete handles api_option=delete.
// Param: api_paste_key
// Response: "Paste Removed" on success.
func (d *Data) pastebinDelete(rw http.ResponseWriter, req *http.Request) {
	key := req.PostFormValue("api_paste_key")
	if key == "" {
		http.Error(rw, "Bad API request: paste key missing", http.StatusBadRequest)
		return
	}

	if err := d.DB.PasteDelete(key); err != nil {
		if err == storage.ErrNotFoundID {
			http.Error(rw, "Bad API request: invalid paste key", http.StatusNotFound)
		} else {
			d.Log.Error(err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("Paste Removed\n"))
}

// pastebinList handles api_option=list.
// Returns the 50 most recent public pastes in Pastebin-compatible XML.
func (d *Data) pastebinList(rw http.ResponseWriter, req *http.Request) {
	pastes, err := d.DB.PasteList(50, 0)
	if err != nil {
		d.Log.Error(err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/xml; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n<pastes>\n"))
	for _, p := range pastes {
		rw.Write([]byte(pastebinXMLItem(p.ID, p.Title, p.Syntax, p.CreateTime, p.DeleteTime, d.BaseURL)))
	}
	rw.Write([]byte("</pastes>\n"))
}

// pastebinTrends handles api_option=trends.
// CasPaste has no hit counter, so trending == recent.
func (d *Data) pastebinTrends(rw http.ResponseWriter, req *http.Request) {
	pastes, err := d.DB.PasteList(18, 0)
	if err != nil {
		d.Log.Error(err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/xml; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n<pastes>\n"))
	for _, p := range pastes {
		rw.Write([]byte(pastebinXMLItem(p.ID, p.Title, p.Syntax, p.CreateTime, p.DeleteTime, d.BaseURL)))
	}
	rw.Write([]byte("</pastes>\n"))
}

// pastebinRaw handles GET /api/api_raw.php?i={key}
// Response: raw paste body as text/plain.
func (d *Data) pastebinRaw(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := req.ParseForm(); err != nil {
		http.Error(rw, "Bad API request", http.StatusBadRequest)
		return
	}

	key := req.Form.Get("i")
	if key == "" {
		http.Error(rw, "Bad API request: missing key", http.StatusBadRequest)
		return
	}

	paste, err := d.DB.PasteGet(key)
	if err != nil {
		if err == storage.ErrNotFoundID {
			http.Error(rw, "Bad API request: invalid paste key", http.StatusNotFound)
		} else {
			d.Log.Error(err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(paste.Body))
}

// pastebinXMLItem renders one <paste> XML element in Pastebin's format.
func pastebinXMLItem(id, title, syntax string, created, expires int64, baseURL string) string {
	if title == "" {
		title = "Untitled"
	}
	expStr := "0"
	if expires > 0 {
		expStr = time.Unix(expires, 0).UTC().Format("2006-01-02 15:04:05")
	}
	return "<paste>" +
		"<paste_key>" + xmlEscape(id) + "</paste_key>" +
		"<paste_date>" + time.Unix(created, 0).UTC().Format("2006-01-02 15:04:05") + "</paste_date>" +
		"<paste_title>" + xmlEscape(title) + "</paste_title>" +
		"<paste_size>0</paste_size>" +
		"<paste_expire_date>" + expStr + "</paste_expire_date>" +
		"<paste_private>0</paste_private>" +
		"<paste_format_long>" + xmlEscape(syntax) + "</paste_format_long>" +
		"<paste_format_short>" + xmlEscape(syntax) + "</paste_format_short>" +
		"<paste_url>" + xmlEscape(baseURL+"/"+id) + "</paste_url>" +
		"<paste_hits>0</paste_hits>" +
		"</paste>\n"
}

// xmlEscape escapes the five predefined XML entities in a string.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// pastebinExpiry maps pastebin.com expire_date codes to Unix timestamps.
// Returns 0 (never expires) for unknown or "N".
func pastebinExpiry(code string) int64 {
	now := time.Now().Unix()
	switch strings.ToUpper(code) {
	case "10M":
		return now + 600
	case "1H":
		return now + 3600
	case "1D":
		return now + 86400
	case "1W":
		return now + 7 * 86400
	case "2W":
		return now + 14 * 86400
	case "1M":
		return now + 30 * 86400
	case "6M":
		return now + 180 * 86400
	case "1Y":
		return now + 365 * 86400
	}
	return 0
}

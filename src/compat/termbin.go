// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package compat

import (
	"io"
	"net/http"

	"github.com/casjay-forks/caspaste/src/storage"
)

// handleTermbin intercepts termbin/netcat-style HTTP paths and returns true if handled.
//
// Termbin uses a raw TCP socket on port 9999 (handled by netshare/TCP layer,
// not here). The HTTP compat layer handles the two common HTTP forms:
//
//	POST /             → create paste from raw body → plain-text URL
//	POST /termbin      → same
//
// Both are used by curl-pipe scripts:
//
//	curl -F "c=@-" https://termbin.example.com/
//	cat file | curl -F "c=@-" https://termbin.example.com/termbin
func (d *Data) handleTermbin(rw http.ResponseWriter, req *http.Request) bool {
	if req.Method != http.MethodPost {
		return false
	}
	if req.URL.Path != "/" && req.URL.Path != "/termbin" {
		return false
	}
	d.termbinCreate(rw, req)
	return true
}

// termbinCreate accepts a raw POST body or a multipart "c" field (the curl -F form)
// and creates a paste, returning its URL as plain text.
func (d *Data) termbinCreate(rw http.ResponseWriter, req *http.Request) {
	var body string

	ct := req.Header.Get("Content-Type")
	if isMultipart(ct) || isURLEncoded(ct) {
		if err := req.ParseMultipartForm(32 << 20); err != nil {
			req.ParseForm()
		}
		// Accept field "c" (termbin convention) or "content".
		body = req.FormValue("c")
		if body == "" {
			body = req.FormValue("content")
		}
	}

	// Fall back to raw body.
	if body == "" {
		raw, err := io.ReadAll(io.LimitReader(req.Body, int64(d.BodyMaxLen)+1))
		if err == nil {
			body = string(raw)
		}
	}

	if body == "" {
		http.Error(rw, "empty paste", http.StatusBadRequest)
		return
	}

	paste := storage.Paste{
		Body:   body,
		Syntax: "text",
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

// isMultipart returns true when the Content-Type indicates multipart/form-data.
func isMultipart(ct string) bool {
	return len(ct) >= 9 && ct[:9] == "multipart"
}

// isURLEncoded returns true when the Content-Type indicates application/x-www-form-urlencoded.
func isURLEncoded(ct string) bool {
	return len(ct) >= 33 && ct[:33] == "application/x-www-form-urlencoded"
}

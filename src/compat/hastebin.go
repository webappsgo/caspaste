// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package compat

import (
	"io"
	"net/http"
	"strings"

	"github.com/casjay-forks/caspaste/src/storage"
)

// handleHastebin intercepts Hastebin API paths and returns true if handled.
//
// Hastebin API surface:
//
//	POST /documents            → create paste → {"key":"xxx"}
//	GET  /documents/{key}      → get paste   → {"key":"xxx","data":"..."}
//	GET  /raw/{key}            → raw text (already handled natively; fall through)
func (d *Data) handleHastebin(rw http.ResponseWriter, req *http.Request) bool {
	path := req.URL.Path

	switch {
	case path == "/documents":
		d.hastebinCreate(rw, req)
		return true
	case strings.HasPrefix(path, "/documents/"):
		d.hastebinGet(rw, req)
		return true
	}
	return false
}

// hastebinCreate handles POST /documents
// Body: raw text
// Response: {"key":"xxx"}
func (d *Data) hastebinCreate(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonErr(rw, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if d.checkRateLimit(rw, req) {
		return
	}

	body, err := io.ReadAll(io.LimitReader(req.Body, int64(d.BodyMaxLen)+1))
	if err != nil || len(body) == 0 {
		jsonErr(rw, http.StatusBadRequest, "body is required")
		return
	}

	paste := storage.Paste{
		Body:   string(body),
		Syntax: "text",
	}

	id, _, _, addErr := d.DB.PasteAdd(paste)
	if addErr != nil {
		d.Log.Error(addErr)
		jsonErr(rw, http.StatusInternalServerError, "internal error")
		return
	}

	jsonOK(rw, map[string]string{"key": id})
}

// hastebinGet handles GET /documents/{key}
// Response: {"key":"xxx","data":"..."}
func (d *Data) hastebinGet(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonErr(rw, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	key := strings.TrimPrefix(req.URL.Path, "/documents/")
	if key == "" {
		jsonErr(rw, http.StatusBadRequest, "key is required")
		return
	}

	paste, err := d.DB.PasteGet(key)
	if err != nil {
		if err == storage.ErrNotFoundID {
			jsonErr(rw, http.StatusNotFound, "not found")
		} else {
			d.Log.Error(err)
			jsonErr(rw, http.StatusInternalServerError, "internal error")
		}
		return
	}

	jsonOK(rw, map[string]string{
		"key":  paste.ID,
		"data": paste.Body,
	})
}

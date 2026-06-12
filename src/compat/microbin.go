// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package compat

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/casjay-forks/caspaste/src/storage"
)

// handleMicrobin intercepts Microbin-style paths and returns true if handled.
//
// Microbin API surface:
//
//	POST /upload               → create paste (multipart form)
//	GET  /pasta/{id}           → redirect to /{id}
//	GET  /rawpasta/{id}        → redirect to /raw/{id}
//	GET  /api/{id}             → get paste JSON (Microbin API extension)
//	DELETE /api/{id}           → delete paste
func (d *Data) handleMicrobin(rw http.ResponseWriter, req *http.Request) bool {
	path := req.URL.Path

	switch {
	case path == "/upload" && req.Method == http.MethodPost:
		d.microbinCreate(rw, req)
		return true
	case path == "/list":
		d.microbinList(rw, req)
		return true
	case path == "/archive":
		d.microbinArchive(rw, req)
		return true
	case strings.HasPrefix(path, "/pasta/"):
		id := strings.TrimPrefix(path, "/pasta/")
		http.Redirect(rw, req, "/"+id, http.StatusFound)
		return true
	case strings.HasPrefix(path, "/rawpasta/"):
		id := strings.TrimPrefix(path, "/rawpasta/")
		http.Redirect(rw, req, "/raw/"+id, http.StatusFound)
		return true
	case strings.HasPrefix(path, "/api/"):
		d.microbinAPI(rw, req)
		return true
	}
	return false
}

// microbinCreate handles POST /upload
// Accepts multipart form with: content, title, expiry, visibility,
// burn_after_read (bool), syntax_highlight (syntax name).
// Redirects to /pasta/{id} on success (matching Microbin behaviour).
func (d *Data) microbinCreate(rw http.ResponseWriter, req *http.Request) {
	if d.checkRateLimit(rw, req) {
		return
	}
	if err := req.ParseMultipartForm(32 << 20); err != nil {
		// Fall back to regular form parsing.
		if err2 := req.ParseForm(); err2 != nil {
			http.Error(rw, "invalid form data", http.StatusBadRequest)
			return
		}
	}

	body := req.FormValue("content")
	if body == "" {
		// Also accept raw body as a fallback (used by some clients).
		raw, err := io.ReadAll(io.LimitReader(req.Body, int64(d.BodyMaxLen)+1))
		if err == nil {
			body = string(raw)
		}
	}
	if body == "" {
		http.Error(rw, "content is required", http.StatusBadRequest)
		return
	}

	title := req.FormValue("title")
	syntax := req.FormValue("syntax_highlight")
	if syntax == "" {
		syntax = "text"
	}

	isPrivate := req.FormValue("visibility") == "private"
	burnAfterRead := req.FormValue("burn_after_read") == "true" || req.FormValue("burn_after_read") == "1"

	var deleteTime int64
	if exp := req.FormValue("expiry"); exp != "" {
		secs := microbinExpiryToSecs(exp)
		if secs > 0 {
			deleteTime = time.Now().Unix() + secs
		}
	}

	paste := storage.Paste{
		Title:      title,
		Body:       body,
		Syntax:     syntax,
		DeleteTime: deleteTime,
		OneUse:     burnAfterRead,
		IsPrivate:  isPrivate,
	}

	id, _, _, err := d.DB.PasteAdd(paste)
	if err != nil {
		d.Log.Error(err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, req, "/pasta/"+id, http.StatusFound)
}

// microbinList handles GET /list — returns a JSON array of public pastes.
func (d *Data) microbinList(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonErr(rw, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	pastes, err := d.DB.PasteList(50, 0)
	if err != nil {
		d.Log.Error(err)
		jsonErr(rw, http.StatusInternalServerError, "internal error")
		return
	}

	type mbItem struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Syntax string `json:"syntax_highlight"`
		Created int64  `json:"created"`
		Expires int64  `json:"expires"`
	}

	out := make([]mbItem, 0, len(pastes))
	for _, p := range pastes {
		out = append(out, mbItem{
			ID:      p.ID,
			Title:   p.Title,
			Syntax:  p.Syntax,
			Created: p.CreateTime,
			Expires: p.DeleteTime,
		})
	}

	jsonOK(rw, out)
}

// microbinArchive handles GET /archive — alias of /list for Microbin compatibility.
func (d *Data) microbinArchive(rw http.ResponseWriter, req *http.Request) {
	d.microbinList(rw, req)
}

// microbinAPI handles GET/POST/DELETE /api/{id}
// GET  → return paste JSON
// DELETE → delete paste
func (d *Data) microbinAPI(rw http.ResponseWriter, req *http.Request) {
	id := strings.TrimPrefix(req.URL.Path, "/api/")
	if id == "" {
		jsonErr(rw, http.StatusBadRequest, "id is required")
		return
	}

	type mbPaste struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		Content    string `json:"content"`
		Syntax     string `json:"syntax_highlight"`
		CreateTime int64  `json:"created"`
		DeleteTime int64  `json:"expires"`
		Private    bool   `json:"private"`
		BurnAfter  bool   `json:"burn_after_read"`
	}

	switch req.Method {
	case http.MethodGet:
		paste, err := d.DB.PasteGet(id)
		if err != nil {
			if err == storage.ErrNotFoundID {
				jsonErr(rw, http.StatusNotFound, "not found")
			} else {
				d.Log.Error(err)
				jsonErr(rw, http.StatusInternalServerError, "internal error")
			}
			return
		}
		jsonOK(rw, mbPaste{
			ID:         paste.ID,
			Title:      paste.Title,
			Content:    paste.Body,
			Syntax:     paste.Syntax,
			CreateTime: paste.CreateTime,
			DeleteTime: paste.DeleteTime,
			Private:    paste.IsPrivate,
			BurnAfter:  paste.OneUse,
		})

	case http.MethodPost:
		// Edit paste: read existing, apply updates from form, write back.
		existing, err := d.DB.PasteGet(id)
		if err != nil {
			if err == storage.ErrNotFoundID {
				jsonErr(rw, http.StatusNotFound, "not found")
			} else {
				d.Log.Error(err)
				jsonErr(rw, http.StatusInternalServerError, "internal error")
			}
			return
		}
		if err := req.ParseMultipartForm(32 << 20); err != nil {
			req.ParseForm()
		}
		if v := req.FormValue("content"); v != "" {
			existing.Body = v
		}
		if v := req.FormValue("title"); v != "" {
			existing.Title = v
		}
		if v := req.FormValue("syntax_highlight"); v != "" {
			existing.Syntax = v
		}
		if err := d.DB.PasteUpdate(existing); err != nil {
			d.Log.Error(err)
			jsonErr(rw, http.StatusInternalServerError, "internal error")
			return
		}
		jsonOK(rw, mbPaste{
			ID:         existing.ID,
			Title:      existing.Title,
			Content:    existing.Body,
			Syntax:     existing.Syntax,
			CreateTime: existing.CreateTime,
			DeleteTime: existing.DeleteTime,
			Private:    existing.IsPrivate,
			BurnAfter:  existing.OneUse,
		})

	case http.MethodDelete:
		if err := d.DB.PasteDelete(id); err != nil {
			if err == storage.ErrNotFoundID {
				jsonErr(rw, http.StatusNotFound, "not found")
			} else {
				d.Log.Error(err)
				jsonErr(rw, http.StatusInternalServerError, "internal error")
			}
			return
		}
		rw.WriteHeader(http.StatusNoContent)

	default:
		jsonErr(rw, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// microbinExpiryToSecs converts Microbin expiry strings to seconds.
func microbinExpiryToSecs(s string) int64 {
	switch strings.ToLower(s) {
	case "1min", "1minute":
		return 60
	case "10min", "10minutes":
		return 600
	case "1hour":
		return 3600
	case "24hour", "1day":
		return 86400
	case "3days":
		return 3 * 86400
	case "1week":
		return 7 * 86400
	case "1month":
		return 30 * 86400
	case "6months":
		return 180 * 86400
	case "1year":
		return 365 * 86400
	case "never", "0":
		return 0
	}
	return 0
}

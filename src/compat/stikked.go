// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package compat

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/casjay-forks/caspaste/src/storage"
)

// handleStikked intercepts Stikked API paths and returns true if handled.
//
// Stikked API surface:
//
//	POST /api/create           → create paste → plain-text URL
//	GET  /api/paste?id=        → get paste JSON
//	GET  /api/recent           → recent pastes JSON array
//	GET  /api/langs            → available syntaxes
//	GET  /view/{id}            → redirect to /{id}
//	GET  /view/raw/{id}        → redirect to /raw/{id}
//	GET  /view/download/{id}   → redirect to /dl/{id}
func (d *Data) handleStikked(rw http.ResponseWriter, req *http.Request) bool {
	path := req.URL.Path

	switch {
	case path == "/api/create":
		d.stikkedCreate(rw, req)
		return true
	case path == "/api/paste":
		d.stikkedGetPaste(rw, req)
		return true
	case path == "/api/recent":
		d.stikkedRecent(rw, req)
		return true
	case path == "/api/trending":
		d.stikkedTrending(rw, req)
		return true
	case path == "/api/langs":
		d.stikkedLangs(rw, req)
		return true
	case path == "/lists", strings.HasPrefix(path, "/lists/"):
		d.stikkedLists(rw, req)
		return true
	case path == "/trends", strings.HasPrefix(path, "/trends/"):
		d.stikkedTrending(rw, req)
		return true
	case strings.HasPrefix(path, "/view/raw/"):
		id := strings.TrimPrefix(path, "/view/raw/")
		http.Redirect(rw, req, "/raw/"+id, http.StatusFound)
		return true
	case strings.HasPrefix(path, "/view/download/"):
		id := strings.TrimPrefix(path, "/view/download/")
		http.Redirect(rw, req, "/dl/"+id, http.StatusFound)
		return true
	case strings.HasPrefix(path, "/view/embed/"):
		id := strings.TrimPrefix(path, "/view/embed/")
		http.Redirect(rw, req, "/emb/"+id, http.StatusFound)
		return true
	case strings.HasPrefix(path, "/view/"):
		id := strings.TrimPrefix(path, "/view/")
		http.Redirect(rw, req, "/"+id, http.StatusFound)
		return true
	}
	return false
}

// stikkedCreate handles POST /api/create
// Form params: text (body), title, name, lang, private (1/0), expire (seconds)
// Response: plain text full URL of the new paste
func (d *Data) stikkedCreate(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := req.ParseForm(); err != nil {
		http.Error(rw, "invalid form data", http.StatusBadRequest)
		return
	}

	body := req.PostFormValue("text")
	if body == "" {
		http.Error(rw, "Error: Missing paste text", http.StatusBadRequest)
		return
	}

	title := req.PostFormValue("title")
	syntax := req.PostFormValue("lang")
	if syntax == "" {
		syntax = "text"
	}

	isPrivate := req.PostFormValue("private") == "1"

	var deleteTime int64
	if exp := req.PostFormValue("expire"); exp != "" {
		var secs int64
		if _, err := parseInt64(exp, &secs); err == nil && secs > 0 {
			deleteTime = time.Now().Unix() + secs
		}
	}

	paste := storage.Paste{
		Title:      title,
		Body:       body,
		Syntax:     syntax,
		DeleteTime: deleteTime,
		IsPrivate:  isPrivate,
		Author:     req.PostFormValue("name"),
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

// stikkedGetPaste handles GET /api/paste?id=xxx
// Returns a JSON object matching Stikked's paste shape.
func (d *Data) stikkedGetPaste(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonErr(rw, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		jsonErr(rw, http.StatusBadRequest, "invalid query")
		return
	}

	id := req.Form.Get("id")
	if id == "" {
		jsonErr(rw, http.StatusBadRequest, "id is required")
		return
	}

	paste, err := d.DB.PasteGet(id)
	if err != nil {
		if err == storage.ErrNotFoundID {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(map[string]string{"message": "Not found"})
		} else {
			d.Log.Error(err)
			jsonErr(rw, http.StatusInternalServerError, "internal error")
		}
		return
	}

	type stikkedPaste struct {
		PID     string `json:"pid"`
		Title   string `json:"title"`
		Name    string `json:"name"`
		Created int64  `json:"created"`
		Lang    string `json:"lang"`
		Code    string `json:"code"`
		URL     string `json:"url"`
	}

	jsonOK(rw, stikkedPaste{
		PID:     paste.ID,
		Title:   paste.Title,
		Name:    paste.Author,
		Created: paste.CreateTime,
		Lang:    paste.Syntax,
		Code:    paste.Body,
		URL:     d.BaseURL + "/" + paste.ID,
	})
}

// stikkedRecent handles GET /api/recent — returns the 20 most recent public pastes.
func (d *Data) stikkedRecent(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonErr(rw, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	pastes, err := d.DB.PasteList(20, 0)
	if err != nil {
		d.Log.Error(err)
		jsonErr(rw, http.StatusInternalServerError, "internal error")
		return
	}

	type stikkedItem struct {
		PID     string `json:"pid"`
		Title   string `json:"title"`
		Name    string `json:"name"`
		Created int64  `json:"created"`
		Lang    string `json:"lang"`
	}

	out := make([]stikkedItem, 0, len(pastes))
	for _, p := range pastes {
		out = append(out, stikkedItem{
			PID:     p.ID,
			Title:   p.Title,
			Name:    "",
			Created: p.CreateTime,
			Lang:    p.Syntax,
		})
	}

	jsonOK(rw, out)
}

// stikkedTrending handles GET /api/trending and GET /trends — returns the 20 most
// recent public pastes as a Stikked-shaped array (CasPaste has no separate hit
// counter, so trending == recent).
func (d *Data) stikkedTrending(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonErr(rw, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	pastes, err := d.DB.PasteList(20, 0)
	if err != nil {
		d.Log.Error(err)
		jsonErr(rw, http.StatusInternalServerError, "internal error")
		return
	}

	type stikkedTrendItem struct {
		PID     string `json:"pid"`
		Title   string `json:"title"`
		Name    string `json:"name"`
		Created int64  `json:"created"`
		Lang    string `json:"lang"`
		Hits    int    `json:"hits"`
	}

	out := make([]stikkedTrendItem, 0, len(pastes))
	for _, p := range pastes {
		out = append(out, stikkedTrendItem{
			PID:     p.ID,
			Title:   p.Title,
			Name:    "",
			Created: p.CreateTime,
			Lang:    p.Syntax,
			Hits:    0,
		})
	}

	jsonOK(rw, out)
}

// stikkedLists handles GET /lists[/{page}] — returns a page of public pastes.
func (d *Data) stikkedLists(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonErr(rw, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := req.ParseForm(); err != nil {
		jsonErr(rw, http.StatusBadRequest, "invalid query")
		return
	}

	page := 1
	if p := req.Form.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	offset := (page - 1) * 20

	pastes, err := d.DB.PasteList(20, offset)
	if err != nil {
		d.Log.Error(err)
		jsonErr(rw, http.StatusInternalServerError, "internal error")
		return
	}

	type stikkedItem struct {
		PID     string `json:"pid"`
		Title   string `json:"title"`
		Name    string `json:"name"`
		Created int64  `json:"created"`
		Lang    string `json:"lang"`
	}

	out := make([]stikkedItem, 0, len(pastes))
	for _, p := range pastes {
		out = append(out, stikkedItem{
			PID:     p.ID,
			Title:   p.Title,
			Name:    "",
			Created: p.CreateTime,
			Lang:    p.Syntax,
		})
	}

	jsonOK(rw, out)
}

// stikkedLangs handles GET /api/langs — returns the list of supported syntax names.
func (d *Data) stikkedLangs(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonErr(rw, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Return a representative set of common syntaxes supported by Chroma
	// (same lexer library used by CasPaste).
	langs := []string{
		"text", "bash", "c", "cpp", "csharp", "css", "diff", "docker",
		"go", "html", "ini", "java", "javascript", "json", "kotlin",
		"lua", "makefile", "markdown", "nginx", "perl", "php", "python",
		"ruby", "rust", "scala", "shell", "sql", "swift", "toml",
		"typescript", "xml", "yaml",
	}

	jsonOK(rw, langs)
}

// parseInt64 parses a decimal string into *dst; returns an error on failure.
func parseInt64(s string, dst *int64) (int64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	if dst != nil {
		*dst = v
	}
	return v, nil
}

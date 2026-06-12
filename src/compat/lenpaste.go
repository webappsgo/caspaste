// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package compat

import (
	"net/http"
	"strconv"
	"time"

	"github.com/casjay-forks/caspaste/src/storage"
)

// handleLenpaste intercepts Lenpaste API paths and returns true if it handled
// the request, false to fall through to the native handler.
//
// Lenpaste API surface:
//
//	POST /api/v1/new           → create paste
//	GET  /api/v1/get?id=       → get paste (optionally consume one-use)
//	GET  /api/v1/getServerInfo → server metadata
func (d *Data) handleLenpaste(rw http.ResponseWriter, req *http.Request) bool {
	switch req.URL.Path {
	case "/api/v1/new":
		d.lenpasteNew(rw, req)
		return true
	case "/api/v1/get":
		d.lenpasteGet(rw, req)
		return true
	case "/api/v1/getServerInfo":
		d.lenpasteServerInfo(rw, req)
		return true
	}
	return false
}

// lenpasteNew handles POST /api/v1/new
// Form params: title, body, syntax, lifeTime (seconds; 0 or -1 = never), oneUse (bool)
// Response: {"id":"xxx"}
func (d *Data) lenpasteNew(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonErr(rw, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if d.checkRateLimit(rw, req) {
		return
	}
	if err := req.ParseForm(); err != nil {
		jsonErr(rw, http.StatusBadRequest, "invalid form data")
		return
	}

	body := req.PostFormValue("body")
	if body == "" {
		jsonErr(rw, http.StatusBadRequest, "body is required")
		return
	}

	title := req.PostFormValue("title")
	syntax := req.PostFormValue("syntax")
	if syntax == "" {
		syntax = "text"
	}

	var deleteTime int64
	if lt := req.PostFormValue("lifeTime"); lt != "" {
		secs, err := strconv.ParseInt(lt, 10, 64)
		if err == nil && secs > 0 {
			deleteTime = time.Now().Unix() + secs
		}
	}

	oneUse := false
	if ou := req.PostFormValue("oneUse"); ou == "true" || ou == "1" {
		oneUse = true
	}

	paste := storage.Paste{
		Title:      title,
		Body:       body,
		Syntax:     syntax,
		DeleteTime: deleteTime,
		OneUse:     oneUse,
	}

	id, _, _, err := d.DB.PasteAdd(paste)
	if err != nil {
		d.Log.Error(err)
		jsonErr(rw, http.StatusInternalServerError, "failed to create paste")
		return
	}

	jsonOK(rw, map[string]string{"id": id})
}

// lenpasteGet handles GET /api/v1/get?id=xxx[&openOneUse=true]
// Returns the paste object in Lenpaste-compatible shape.
func (d *Data) lenpasteGet(rw http.ResponseWriter, req *http.Request) {
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
			jsonErr(rw, http.StatusNotFound, "paste not found")
		} else {
			d.Log.Error(err)
			jsonErr(rw, http.StatusInternalServerError, "internal error")
		}
		return
	}

	// One-use pastes: reveal content only when openOneUse=true, then delete.
	if paste.OneUse {
		if req.Form.Get("openOneUse") == "true" {
			if delErr := d.DB.PasteDelete(id); delErr != nil {
				d.Log.Error(delErr)
			}
		} else {
			// Return the stub: ID + oneUse flag only, body hidden.
			jsonOK(rw, map[string]interface{}{
				"id":     paste.ID,
				"oneUse": true,
			})
			return
		}
	}

	type lenpasteResponse struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		Body       string `json:"body"`
		Syntax     string `json:"syntax"`
		CreateTime int64  `json:"createTime"`
		DeleteTime int64  `json:"deleteTime"`
		OneUse     bool   `json:"oneUse"`
	}

	jsonOK(rw, lenpasteResponse{
		ID:         paste.ID,
		Title:      paste.Title,
		Body:       paste.Body,
		Syntax:     paste.Syntax,
		CreateTime: paste.CreateTime,
		DeleteTime: paste.DeleteTime,
		OneUse:     paste.OneUse,
	})
}

// lenpasteServerInfo handles GET /api/v1/getServerInfo
func (d *Data) lenpasteServerInfo(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonErr(rw, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	type info struct {
		Version     string `json:"version"`
		TitleMaxLen int    `json:"titleMaxlength"`
		BodyMaxLen  int    `json:"bodyMaxlength"`
		MaxLifeTime int64  `json:"maxLifeTime"`
		ServerAbout string `json:"serverAbout"`
		ServerRules string `json:"serverRules"`
		AdminName   string `json:"adminName"`
		AdminMail   string `json:"adminMail"`
	}

	jsonOK(rw, info{
		Version:     d.Version,
		TitleMaxLen: d.TitleMaxLen,
		BodyMaxLen:  d.BodyMaxLen,
		MaxLifeTime: d.MaxLifeTime,
		ServerAbout: d.ServerAbout,
		ServerRules: d.ServerRules,
		AdminName:   d.AdminName,
		AdminMail:   d.AdminMail,
	})
}

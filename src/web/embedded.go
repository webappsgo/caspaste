// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"encoding/base64"
	"html/template"
	"net/http"
	"time"

	"github.com/casjay-forks/caspaste/src/netshare"
	"github.com/casjay-forks/caspaste/src/storage"
)

type embTmpl struct {
	ID            string
	CreateTimeStr string
	DeleteTime    int64
	OneUse        bool
	Title         string
	Body          template.HTML

	ErrorNotFound bool
	Language      string
	Theme         func(string) string
	Translate     func(string, ...interface{}) template.HTML
}

// Pattern: /emb/
func (data *Data) handleEmbedded(rw http.ResponseWriter, req *http.Request) error {
	errorNotFound := false

	// Check rate limit
	err := data.RateLimitGet.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Get paste ID
	pasteID := string([]rune(req.URL.Path)[5:])

	// Read DB
	paste, err := data.DB.PasteGet(pasteID)
	if err != nil {
		if err == storage.ErrNotFoundID {
			errorNotFound = true

		} else {
			return err
		}
	}

	// Prepare template data
	createTime := time.Unix(paste.CreateTime, 0).UTC()

	// Determine body content based on whether this is a file upload
	var bodyContent string
	if paste.IsFile {
		// File upload: try to decode base64, fall back to raw for legacy data
		fileData, err := base64.StdEncoding.DecodeString(paste.Body)
		if err != nil {
			// Legacy data stored without base64 encoding - use as-is
			bodyContent = paste.Body
		} else {
			bodyContent = string(fileData)
		}
	} else {
		bodyContent = paste.Body
	}

	tmplData := embTmpl{
		ID:            paste.ID,
		CreateTimeStr: createTime.Format("1 Jan, 2006"),
		DeleteTime:    paste.DeleteTime,
		OneUse:        paste.OneUse,
		Title:         paste.Title,
		Body:          tryHighlight(bodyContent, paste.Syntax, "monokai"),

		ErrorNotFound: errorNotFound,
		Language:      getCookie(req, "lang"),
		Theme:         data.getThemeFunc(req),
		Translate:     data.Locales.findLocale(req).translate,
	}

	// Show paste
	return data.EmbeddedPage.Execute(rw, tmplData)
}

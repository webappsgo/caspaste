// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/casjay-forks/caspaste/src/netshare"
)

// pasteListViewItem is the template-facing representation of a paste list row.
// It replaces the raw DB struct so timestamps are pre-formatted server-side.
type pasteListViewItem struct {
	ID            string
	Title         string
	Syntax        string
	CreateTimeStr string
}

type listTmpl struct {
	Pastes     interface{}
	Limit      int
	Offset     int
	NextOffset int
	PrevOffset int
	HasNext    bool
	HasPrev    bool
	User       *AuthUser
	Language   string
	Theme      func(string) string

	CSRFToken     string
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool
	ShowRegister  bool

	Translate func(string, ...interface{}) template.HTML
}

// GET /list
func (data *Data) handleList(rw http.ResponseWriter, req *http.Request) error {
	// Check method
	if req.Method != "GET" {
		return netshare.ErrMethodNotAllowed
	}

	// Check rate limit
	err := data.RateLimitGet.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Parse query parameters
	query := req.URL.Query()

	limit := 50
	if limitStr := query.Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get paste list from database
	rawPastes, err := data.DB.PasteList(limit, offset)
	if err != nil {
		return err
	}

	// Convert to view items with human-readable timestamps
	pastes := make([]pasteListViewItem, len(rawPastes))
	for i, p := range rawPastes {
		pastes[i] = pasteListViewItem{
			ID:            p.ID,
			Title:         p.Title,
			Syntax:        p.Syntax,
			CreateTimeStr: time.Unix(p.CreateTime, 0).UTC().Format("02 Jan 2006 15:04"),
		}
	}

	// Get theme
	themeName := getCookie(req, "theme")
	if themeName == "" {
		themeName = data.UiDefaultTheme
	}
	themeMap, exists := data.Themes[themeName]
	if !exists {
		themeMap = data.Themes[data.UiDefaultTheme]
	}
	themeLookup := func(key string) string {
		return themeMap[key]
	}

	// Render template
	tmplData := listTmpl{
		Pastes:        pastes,
		Limit:         limit,
		Offset:        offset,
		NextOffset:    offset + limit,
		PrevOffset:    offset - limit,
		HasNext:       len(rawPastes) == limit,
		HasPrev:       offset > 0,
		User:          GetAuthUser(req.Context()),
		Language:      getCookie(req, "lang"),
		Theme:         themeLookup,
		CSRFToken:     data.buildCSRFToken(req),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		ShowRegister:  data.ShowRegister,
		Translate:     data.Locales.findLocale(req).translate,
	}

	if tmplData.PrevOffset < 0 {
		tmplData.PrevOffset = 0
	}

	return data.ListPage.Execute(rw, tmplData)
}

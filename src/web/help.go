// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"github.com/casjay-forks/caspaste/src/netshare"
	"html/template"
	"net/http"
)

type embHelpTmpl struct {
	ID         string
	DeleteTime int64
	OneUse     bool

	Protocol string
	Host     string
	User     *AuthUser

	Language string
	Theme    func(string) string

	CSRFToken     string
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool
	ShowRegister  bool

	Translate func(string, ...interface{}) template.HTML
	Highlight func(string, string) template.HTML
}

// Pattern: /emb_help/
func (data *Data) handleEmbeddedHelp(rw http.ResponseWriter, req *http.Request) error {
	// Check rate limit
	err := data.RateLimitGet.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Get paste ID
	pasteID := string([]rune(req.URL.Path)[10:])

	// Read DB
	paste, err := data.DB.PasteGet(pasteID)
	if err != nil {
		return err
	}

	// Show paste
	tmplData := embHelpTmpl{
		ID:            paste.ID,
		DeleteTime:    paste.DeleteTime,
		OneUse:        paste.OneUse,
		Protocol:      netshare.GetProtocol(req),
		Host:          netshare.GetHost(req),
		User:          GetAuthUser(req.Context()),
		Language:      getCookie(req, "lang"),
		Theme:         data.getThemeFunc(req),
		CSRFToken:     data.buildCSRFToken(req),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		ShowRegister:  data.ShowRegister,
		Translate:     data.Locales.findLocale(req).translate,
		Highlight:     data.Themes.findTheme(req, data.UiDefaultTheme).tryHighlight,
	}

	return data.EmbeddedHelpPage.Execute(rw, tmplData)
}

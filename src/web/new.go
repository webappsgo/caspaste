// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"html/template"
	"net/http"

	"github.com/casjay-forks/caspaste/src/netshare"
)

type createTmpl struct {
	User              *AuthUser
	Language          string
	Theme             func(string) string
	TitleMaxLen       int
	BodyMaxLen        int
	AuthorAllMaxLen   int
	MaxLifeTime       int64
	UiDefaultLifeTime string
	Lexers            []string
	ServerTermsExist  bool

	AuthorDefault      string
	AuthorEmailDefault string
	AuthorURLDefault   string

	// CSRF token for form protection per AI.md PART 11
	CSRFToken     string
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool
	ShowRegister  bool

	Translate func(string, ...interface{}) template.HTML
}

func (data *Data) handleNewPaste(rw http.ResponseWriter, req *http.Request) error {
	// Create paste if need
	if req.Method == "POST" {
		pasteID, _, _, err := netshare.PasteAddFromForm(req, data.DB, data.RateLimitNew, data.TitleMaxLen, data.BodyMaxLen, data.MaxLifeTime, data.Lexers)
		if err != nil {
			return err
		}

		// Redirect to paste
		writeRedirect(rw, req, "/"+pasteID, 302)
		return nil
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

	// Else show create page
	tmplData := createTmpl{
		User:               GetAuthUser(req.Context()),
		Language:           getCookie(req, "lang"),
		Theme:              themeLookup,
		TitleMaxLen:        data.TitleMaxLen,
		BodyMaxLen:         data.BodyMaxLen,
		AuthorAllMaxLen:    netshare.MaxLengthAuthorAll,
		MaxLifeTime:        data.MaxLifeTime,
		UiDefaultLifeTime:  data.UiDefaultLifeTime,
		Lexers:             data.Lexers,
		ServerTermsExist:   data.ServerTermsExist,
		AuthorDefault:      getCookie(req, "author"),
		AuthorEmailDefault: getCookie(req, "authorEmail"),
		AuthorURLDefault:   getCookie(req, "authorURL"),
		CSRFToken:          GetCSRFToken(req, 32),
		UnreadCount:        0,
		Notifications:      nil,
		ShowLogin:     data.ShowLogin,
		ShowRegister:       data.ShowRegister,
		Translate:          data.Locales.findLocale(req).translate,
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")

	return data.Main.Execute(rw, tmplData)
}

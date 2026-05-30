// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"github.com/casjay-forks/caspaste/src/caspasswd"
	"github.com/casjay-forks/caspaste/src/netshare"
	"html/template"
	"net/http"
)

const cookieMaxAge = 60 * 60 * 24 * 360 * 50 // 50 year

type settingsTmpl struct {
	LanguageCode     string
	LanguageSelector map[string]string

	ThemeCode     string
	ThemeSelector map[string]string

	AuthorAllMaxLen int
	Author          string
	AuthorEmail     string
	AuthorURL       string

	AuthOk bool
	User   *AuthUser

	Language  string
	Theme     func(string) string
	Translate func(string, ...interface{}) template.HTML

	// CSRF token for form protection per AI.md PART 11
	CSRFToken     string
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool
	ShowRegister  bool
}

// Pattern: /settings
func (data *Data) handleSettings(rw http.ResponseWriter, req *http.Request) error {
	var err error

	// Check auth
	isAuthenticated := true

	if data.CasPasswdFile != "" {
		isAuthenticated = false

		user, pass, authProvided := req.BasicAuth()
		if authProvided {
			isAuthenticated, err = caspasswd.LoadAndCheck(data.CasPasswdFile, user, pass)
			if err != nil {
				return err
			}
		}
	}

	// Show settings page
	if req.Method != "POST" {
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

		// Prepare data
		dataTmpl := settingsTmpl{
			LanguageCode:     getCookie(req, "lang"),
			LanguageSelector: data.LocalesList,
			ThemeCode:        getCookie(req, "theme"),
			ThemeSelector:    data.ThemesList.getForLocale(req),
			AuthorAllMaxLen:  netshare.MaxLengthAuthorAll,
			Author:           getCookie(req, "author"),
			AuthorEmail:      getCookie(req, "authorEmail"),
			AuthorURL:        getCookie(req, "authorURL"),
			AuthOk:           isAuthenticated,
			User:             GetAuthUser(req.Context()),
			Language:         getCookie(req, "lang"),
			Theme:            themeLookup,
			Translate:        data.Locales.findLocale(req).translate,
			CSRFToken:        GetCSRFToken(req, 32),
			UnreadCount:      0,
			Notifications:    nil,
			ShowLogin:     data.ShowLogin,
			ShowRegister:     data.ShowRegister,
		}

		if dataTmpl.ThemeCode == "" {
			dataTmpl.ThemeCode = data.UiDefaultTheme
		}

		// Show page
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")

		err := data.Settings.Execute(rw, dataTmpl)
		if err != nil {
			data.writeError(rw, req, err)
		}

		// Else update settings
	} else {
		req.ParseForm()

		lang := req.PostForm.Get("lang")
		if lang == "" {
			http.SetCookie(rw, &http.Cookie{
				Name:   "lang",
				Value:  "",
				MaxAge: -1,
			})

		} else {
			http.SetCookie(rw, &http.Cookie{
				Name:   "lang",
				Value:  lang,
				MaxAge: cookieMaxAge,
			})
		}

		theme := req.PostForm.Get("theme")
		if theme == "" {
			http.SetCookie(rw, &http.Cookie{
				Name:   "theme",
				Value:  "",
				MaxAge: -1,
			})

		} else {
			http.SetCookie(rw, &http.Cookie{
				Name:   "theme",
				Value:  theme,
				MaxAge: cookieMaxAge,
			})
		}

		author := req.PostForm.Get("author")
		if author == "" {
			http.SetCookie(rw, &http.Cookie{
				Name:   "author",
				Value:  "",
				MaxAge: -1,
			})

		} else {
			http.SetCookie(rw, &http.Cookie{
				Name:   "author",
				Value:  author,
				MaxAge: cookieMaxAge,
			})
		}

		authorEmail := req.PostForm.Get("authorEmail")
		if authorEmail == "" {
			http.SetCookie(rw, &http.Cookie{
				Name:   "authorEmail",
				Value:  "",
				MaxAge: -1,
			})

		} else {
			http.SetCookie(rw, &http.Cookie{
				Name:   "authorEmail",
				Value:  authorEmail,
				MaxAge: cookieMaxAge,
			})
		}

		authorURL := req.PostForm.Get("authorURL")
		if authorURL == "" {
			http.SetCookie(rw, &http.Cookie{
				Name:   "authorURL",
				Value:  "",
				MaxAge: -1,
			})

		} else {
			http.SetCookie(rw, &http.Cookie{
				Name:   "authorURL",
				Value:  authorURL,
				MaxAge: cookieMaxAge,
			})
		}

		writeRedirect(rw, req, "/settings", 302)
	}

	return nil
}

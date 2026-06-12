// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"html/template"
	"net/http"
)

type aboutTmpl struct {
	Version     string
	TitleMaxLen int
	BodyMaxLen  int
	MaxLifeTime int64

	ServerAbout      string
	ServerRules      string
	ServerTermsExist bool

	AdminName string
	AdminMail string
	User      *AuthUser

	Language string
	Theme    func(string) string

	CSRFToken     string
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool

	Highlight func(string, string) template.HTML
	Translate func(string, ...interface{}) template.HTML
}

type aboutMinTmp struct {
	User     *AuthUser
	Language string
	Theme    func(string) string

	CSRFToken     string
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool

	Translate func(string, ...interface{}) template.HTML
}

// Pattern: /about
func (data *Data) handleAbout(rw http.ResponseWriter, req *http.Request) error {
	dataTmpl := aboutTmpl{
		Version:          data.Version,
		TitleMaxLen:      data.TitleMaxLen,
		BodyMaxLen:       data.BodyMaxLen,
		MaxLifeTime:      data.MaxLifeTime,
		ServerAbout:      data.ServerAbout,
		ServerRules:      data.ServerRules,
		ServerTermsExist: data.ServerTermsExist,
		AdminName:        data.AdminName,
		AdminMail:        data.AdminMail,
		User:             GetAuthUser(req.Context()),
		Language:         getCookie(req, "lang"),
		Theme:            data.getThemeFunc(req),
		CSRFToken:        data.buildCSRFToken(req),
		UnreadCount:      0,
		Notifications:    nil,
		ShowLogin:     data.ShowLogin,
		Highlight:        data.Themes.findTheme(req, data.UiDefaultTheme).tryHighlight,
		Translate:        data.Locales.findLocale(req).translate,
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.About.Execute(rw, dataTmpl)
}

// Pattern: /about/authors
func (data *Data) handleAuthors(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.Authors.Execute(rw, aboutMinTmp{
		User:          GetAuthUser(req.Context()),
		Language:      getCookie(req, "lang"),
		Theme:         data.getThemeFunc(req),
		CSRFToken:     data.buildCSRFToken(req),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		Translate:     data.Locales.findLocale(req).translate,
	})
}

// Pattern: /about/license
func (data *Data) handleLicense(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.License.Execute(rw, aboutMinTmp{
		User:          GetAuthUser(req.Context()),
		Language:      getCookie(req, "lang"),
		Theme:         data.getThemeFunc(req),
		CSRFToken:     data.buildCSRFToken(req),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		Translate:     data.Locales.findLocale(req).translate,
	})
}

// Pattern: /about/source_code
func (data *Data) handleSourceCodePage(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.SourceCodePage.Execute(rw, aboutMinTmp{
		User:          GetAuthUser(req.Context()),
		Language:      getCookie(req, "lang"),
		Theme:         data.getThemeFunc(req),
		CSRFToken:     data.buildCSRFToken(req),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		Translate:     data.Locales.findLocale(req).translate,
	})
}

// Pattern: /about/security
func (data *Data) handleSecurityPolicy(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.SecurityPolicy.Execute(rw, aboutMinTmp{
		User:          GetAuthUser(req.Context()),
		Language:      getCookie(req, "lang"),
		Theme:         data.getThemeFunc(req),
		CSRFToken:     data.buildCSRFToken(req),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		Translate:     data.Locales.findLocale(req).translate,
	})
}

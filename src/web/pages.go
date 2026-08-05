// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"html/template"
	"net/http"
)

type privacyTmpl struct {
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

// Pattern: /server/privacy
func (data *Data) handlePrivacy(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.Privacy.Execute(rw, privacyTmpl{
		AdminName:     data.AdminName,
		AdminMail:     data.AdminMail,
		User:          GetAuthUser(req.Context()),
		Language:      getCookie(req, "lang"),
		Theme:         data.getThemeFunc(req),
		CSRFToken:     data.buildCSRFToken(req),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		Highlight:     data.Themes.findTheme(req, data.UiDefaultTheme).tryHighlight,
		Translate:     data.Locales.findLocale(req).translate,
	})
}

// Pattern: /server/contact
func (data *Data) handleContact(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.Contact.Execute(rw, privacyTmpl{
		AdminName:     data.AdminName,
		AdminMail:     data.AdminMail,
		User:          GetAuthUser(req.Context()),
		Language:      getCookie(req, "lang"),
		Theme:         data.getThemeFunc(req),
		CSRFToken:     data.buildCSRFToken(req),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		Highlight:     data.Themes.findTheme(req, data.UiDefaultTheme).tryHighlight,
		Translate:     data.Locales.findLocale(req).translate,
	})
}

// Pattern: POST /server/consent
// Records the visitor's cookie-consent choice in a first-party cookie, then
// redirects back to the originating page (or the home page).
func (data *Data) handleConsent(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodPost {
		return data.handleNewPaste(rw, req)
	}

	if err := req.ParseForm(); err != nil {
		return err
	}

	choice := req.FormValue("consent")
	if choice != "accepted" {
		choice = "declined"
	}

	http.SetCookie(rw, &http.Cookie{
		Name:     "cookie_consent",
		Value:    choice,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 365,
		HttpOnly: true,
		Secure:   req.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	redirect := req.Referer()
	if redirect == "" {
		redirect = "/"
	}
	http.Redirect(rw, req, redirect, http.StatusSeeOther)
	return nil
}

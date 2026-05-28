
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"html/template"
	"net/http"
)

type termsOfUseTmpl struct {
	TermsOfUse string
	User       *AuthUser

	Language  string
	Theme     func(string) string
	Highlight func(string, string) template.HTML
	Translate func(string, ...interface{}) template.HTML
}

// Pattern: /terms
func (data *Data) handleTermsOfUse(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.TermsOfUse.Execute(rw, termsOfUseTmpl{
		TermsOfUse: data.ServerTermsOfUse,
		User:       GetAuthUser(req.Context()),
		Language:   getCookie(req, "lang"),
		Theme:      data.getThemeFunc(req),
		Highlight:  data.Themes.findTheme(req, data.UiDefaultTheme).tryHighlight,
		Translate:  data.Locales.findLocale(req).translate},
	)
}

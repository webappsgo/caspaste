
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"html/template"
	"net/http"

	"github.com/casjay-forks/caspaste/src/netshare"
)

type docsTmpl struct {
	ServerURL string
	User      *AuthUser

	Language  string
	Theme     func(string) string
	Highlight func(string, string) template.HTML
	Translate func(string, ...interface{}) template.HTML
}

type docsApiV1Tmpl struct {
	MaxLenAuthorAll int
	ServerURL       string
	User            *AuthUser

	Language  string
	Theme     func(string) string
	Highlight func(string, string) template.HTML
	Translate func(string, ...interface{}) template.HTML
}

// serverBaseURL returns the scheme+host of the current request.
func serverBaseURL(req *http.Request) string {
	return netshare.GetProtocol(req) + "://" + netshare.GetHost(req)
}

// Pattern: /docs
func (data *Data) handleDocs(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.Docs.Execute(rw, docsTmpl{
		ServerURL: serverBaseURL(req),
		User:      GetAuthUser(req.Context()),
		Language:  getCookie(req, "lang"),
		Theme:     data.getThemeFunc(req),
		Translate: data.Locales.findLocale(req).translate,
	})
}

// Pattern: /docs/apiv1
func (data *Data) handleDocsAPIv1(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.DocsApiV1.Execute(rw, docsApiV1Tmpl{
		MaxLenAuthorAll: netshare.MaxLengthAuthorAll,
		ServerURL:       serverBaseURL(req),
		User:            GetAuthUser(req.Context()),
		Language:        getCookie(req, "lang"),
		Theme:           data.getThemeFunc(req),
		Translate:       data.Locales.findLocale(req).translate,
		Highlight:       data.Themes.findTheme(req, data.UiDefaultTheme).tryHighlight,
	})
}

// Pattern: /docs/libraries
func (data *Data) handleDocsLibraries(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.DocsLibraries.Execute(rw, docsTmpl{
		ServerURL: serverBaseURL(req),
		User:      GetAuthUser(req.Context()),
		Language:  getCookie(req, "lang"),
		Theme:     data.getThemeFunc(req),
		Translate: data.Locales.findLocale(req).translate,
	})
}

// Pattern: /docs/customize
func (data *Data) handleDocsCustomize(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.DocsCustomize.Execute(rw, docsTmpl{
		ServerURL: serverBaseURL(req),
		User:      GetAuthUser(req.Context()),
		Language:  getCookie(req, "lang"),
		Theme:     data.getThemeFunc(req),
		Translate: data.Locales.findLocale(req).translate,
		Highlight: data.Themes.findTheme(req, data.UiDefaultTheme).tryHighlight,
	})
}

// Pattern: /docs/cli
func (data *Data) handleDocsCliExamples(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.DocsCliExamples.Execute(rw, docsTmpl{
		ServerURL: serverBaseURL(req),
		User:      GetAuthUser(req.Context()),
		Language:  getCookie(req, "lang"),
		Theme:     data.getThemeFunc(req),
		Translate: data.Locales.findLocale(req).translate,
		Highlight: data.Themes.findTheme(req, data.UiDefaultTheme).tryHighlight,
	})
}

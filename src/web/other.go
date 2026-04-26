
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"html/template"
	"net/http"
)

type jsTmpl struct {
	Language  string
	Theme     func(string) string
	Translate func(string, ...interface{}) template.HTML
}

func (data *Data) handleStyleCSS(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "text/css; charset=utf-8")
	return data.StyleCSS.Execute(rw, jsTmpl{
		Language:  getCookie(req, "lang"),
		Theme:     data.getThemeFunc(req),
		Translate: data.Locales.findLocale(req).translate,
	})
}

func (data *Data) handleMainJS(rw http.ResponseWriter, req *http.Request) error {
	// Serve with ETag and cache headers per AI.md PART 9
	ServeWithETag(rw, req, *data.MainJS, "application/javascript; charset=utf-8", "static")
	return nil
}

func (data *Data) handleBurnAfterJS(rw http.ResponseWriter, req *http.Request) error {
	// Serve with ETag and cache headers per AI.md PART 9
	ServeWithETag(rw, req, *data.BurnAfterJS, "application/javascript; charset=utf-8", "static")
	return nil
}

func (data *Data) handleToastJS(rw http.ResponseWriter, req *http.Request) error {
	// Toast notifications per AI.md PART 16
	ServeWithETag(rw, req, *data.ToastJS, "application/javascript; charset=utf-8", "static")
	return nil
}

func (data *Data) handleNavJS(rw http.ResponseWriter, req *http.Request) error {
	ServeWithETag(rw, req, *data.NavJS, "application/javascript; charset=utf-8", "static")
	return nil
}

func (data *Data) handleCodeJS(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	return data.CodeJS.Execute(rw, jsTmpl{
		Language:  getCookie(req, "lang"),
		Theme:     data.getThemeFunc(req),
		Translate: data.Locales.findLocale(req).translate,
	})
}

func (data *Data) handleHistoryJS(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	return data.HistoryJS.Execute(rw, jsTmpl{
		Language:  getCookie(req, "lang"),
		Theme:     data.getThemeFunc(req),
		Translate: data.Locales.findLocale(req).translate,
	})
}

func (data *Data) handlePasteJS(rw http.ResponseWriter, req *http.Request) error {
	rw.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	return data.PasteJS.Execute(rw, jsTmpl{
		Language:  getCookie(req, "lang"),
		Theme:     data.getThemeFunc(req),
		Translate: data.Locales.findLocale(req).translate,
	})
}

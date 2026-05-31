// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"html/template"
	"net/http"
)

// stubLink is a navigation link in a stub page.
type stubLink struct {
	URL   string
	Label string
}

// stubField is a form input field for stub pages.
type stubField struct {
	ID           string
	Name         string
	Label        string
	Type         string
	Placeholder  string
	Value        string
	Hint         string
	Required     bool
	MinLength    int
	MaxLength    int
	Pattern      string
	Autocomplete string
	Options      []stubSelectOption
}

// stubSelectOption is an option in a <select> field.
type stubSelectOption struct {
	Value string
	Label string
}

// stubHiddenField is a hidden <input> field in a stub form.
type stubHiddenField struct {
	Name  string
	Value string
}

// stubTmplData holds all data passed to stub.tmpl.
type stubTmplData struct {
	Title       string
	Description string
	Notice      string

	// Navigation links displayed above the form
	Links []stubLink

	// Form fields
	FormAction   string
	FormMethod   string
	Fields       []stubField
	HiddenFields []stubHiddenField
	SubmitLabel  string

	// Meta-redirect (e.g., email verification redirect)
	RedirectURL   string
	RedirectDelay int

	// Back link at the bottom
	BackURL   string
	BackLabel string

	// Common nav fields
	CSRFToken     string
	Language      string
	Theme         func(string) string
	Translate     func(string, ...interface{}) template.HTML
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool
	ShowRegister  bool
	User          *AuthUser
}

// renderStub renders a page using the shared stub template.
func (data *Data) renderStub(rw http.ResponseWriter, req *http.Request, d stubTmplData) error {
	d.CSRFToken = data.buildCSRFToken(req)
	d.Language = getCookie(req, "lang")
	d.Theme = data.getThemeFunc(req)
	d.Translate = data.Locales.findLocale(req).translate
	d.ShowLogin = data.ShowLogin
	d.ShowRegister = data.ShowRegister
	d.User = GetAuthUser(req.Context())
	if d.FormMethod == "" && d.FormAction != "" {
		d.FormMethod = "POST"
	}
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	return data.StubPage.Execute(rw, d)
}

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"

	"github.com/casjay-forks/caspaste/src/netshare"
	"github.com/casjay-forks/caspaste/src/storage"
)

type errorTmpl struct {
	Code      int
	AdminName string
	AdminMail string
	User      *AuthUser
	// Language for base template
	Language string
	// Theme function to get theme values
	Theme func(string) string

	CSRFToken     string
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool
	ShowRegister  bool

	Translate func(string, ...interface{}) template.HTML
}

func (data *Data) writeError(rw http.ResponseWriter, req *http.Request, e error) (int, error) {
	locale := data.Locales.findLocale(req)

	// Get theme name, use default if not set
	themeName := getCookie(req, "theme")
	if themeName == "" {
		themeName = data.UiDefaultTheme
	}

	// Get theme map
	themeMap, exists := data.Themes[themeName]
	if !exists {
		// Fallback to default theme if specified theme doesn't exist
		themeMap = data.Themes[data.UiDefaultTheme]
	}

	// Create theme lookup function
	themeLookup := func(key string) string {
		return themeMap[key]
	}

	errData := errorTmpl{
		Code:      0,
		AdminName: data.AdminName,
		AdminMail: data.AdminMail,
		User:      GetAuthUser(req.Context()),
		// Get language from cookie
		Language: getCookie(req, "lang"),
		// Theme lookup function
		Theme:         themeLookup,
		CSRFToken:     data.buildCSRFToken(req),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		ShowRegister:  data.ShowRegister,
		Translate:     locale.translate,
	}

	// Detect error type
	var eTmp429 *netshare.RateLimitError

	if e == netshare.ErrBadRequest {
		errData.Code = 400

	} else if e == netshare.ErrUnauthorized {
		errData.Code = 401

	} else if e == storage.ErrNotFoundID {
		errData.Code = 404

	} else if e == netshare.ErrNotFound {
		errData.Code = 404

	} else if e == netshare.ErrMethodNotAllowed {
		errData.Code = 405

	} else if e == netshare.ErrPayloadTooLarge {
		errData.Code = 413

	} else if errors.As(e, &eTmp429) {
		errData.Code = 429
		rw.Header().Set("Retry-After", strconv.FormatInt(eTmp429.RetryAfter, 10))

	} else {
		errData.Code = 500
	}

	// Write response header
	rw.Header().Set("Content-type", "text/html; charset=utf-8")
	rw.WriteHeader(errData.Code)

	// Render template
	err := data.ErrorPage.Execute(rw, errData)
	if err != nil {
		return 500, err
	}

	return errData.Code, nil
}

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"net/http"
)

// SecurityHeadersConfig holds configuration for security headers per AI.md PART 11
type SecurityHeadersConfig struct {
	XFrameOptions           string
	XContentTypeOptions     string
	XSSProtection           string
	ContentSecurityPolicy   string
	ReferrerPolicy          string
	PermissionsPolicy       string
	StrictTransportSecurity string
}

func getCookie(req *http.Request, name string) string {
	cookie, err := req.Cookie(name)
	if err != nil {
		return ""
	}

	return cookie.Value
}

// getThemeFunc returns a theme lookup function for the given request
// This is used by templates to access theme values
func (data *Data) getThemeFunc(req *http.Request) func(string) string {
	themeName := getCookie(req, "theme")
	if themeName == "" {
		themeName = data.UiDefaultTheme
	}
	themeMap, exists := data.Themes[themeName]
	if !exists {
		themeMap = data.Themes[data.UiDefaultTheme]
	}
	return func(key string) string {
		return themeMap[key]
	}
}

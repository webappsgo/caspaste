// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import "net/http"

// NavNotification represents a user notification shown in the header bell dropdown.
type NavNotification struct {
	ID      int64
	Link    string
	Icon    string
	Title   string
	Message string
	TimeAgo string
	Read    bool
}

// buildCSRFToken returns a CSRF token for the current request session.
func (data *Data) buildCSRFToken(req *http.Request) string {
	return GetCSRFToken(req, 32)
}

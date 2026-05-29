// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"net/http"
)

func writeRedirect(rw http.ResponseWriter, req *http.Request, newURL string, code int) {
	if newURL == "" {
		newURL = "/"
	}

	if req.URL.RawQuery != "" {
		newURL = newURL + "?" + req.URL.RawQuery
	}

	rw.Header().Set("Location", newURL)
	rw.WriteHeader(code)
}

// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"net/http"

	"github.com/casjay-forks/caspaste/src/netshare"
)

// GET /u/{id} - URL shortener redirect
func (data *Data) handleURLRedirect(rw http.ResponseWriter, req *http.Request) error {
	// Check method
	if req.Method != "GET" {
		return netshare.ErrMethodNotAllowed
	}

	// Get ID from path
	id := req.URL.Path[len("/u/"):]

	// Check rate limit
	err := data.RateLimitGet.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Get paste from database
	paste, err := data.DB.PasteGet(id)
	if err != nil {
		return err
	}

	// Check if it's a URL shortener entry
	if !paste.IsURL || paste.OriginalURL == "" {
		return netshare.ErrNotFound
	}

	// Redirect to original URL
	http.Redirect(rw, req, paste.OriginalURL, http.StatusFound)
	return nil
}

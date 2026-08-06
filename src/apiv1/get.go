// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package apiv1

import (
	"net/http"

	"github.com/webappsgo/caspaste/src/netshare"
)

// GET /api/v1/pastes?id=X - get single paste per AI.md PART 14
func (data *Data) getPaste(rw http.ResponseWriter, req *http.Request) error {
	// Check rate limit
	err := data.RateLimitGet.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Get paste ID (already parsed by handlePastes)
	pasteID := req.Form.Get("id")

	// Check paste id
	if pasteID == "" {
		return netshare.ErrBadRequest
	}

	// Get paste
	paste, err := data.DB.PasteGet(pasteID)
	if err != nil {
		return err
	}

	// If "one use" (burn after reading) paste - delete it after returning content
	if paste.OneUse {
		// Delete paste immediately - burn after reading just works
		err = data.DB.PasteDelete(pasteID)
		if err != nil {
			return err
		}
	}

	// Return response with content negotiation per AI.md PART 14, 16
	// For text format, return just the raw paste body (useful for curl/wget)
	return writeSuccess(rw, req, paste, "Paste retrieved", paste.Body)
}

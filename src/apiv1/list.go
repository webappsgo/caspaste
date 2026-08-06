// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package apiv1

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/webappsgo/caspaste/src/netshare"
)

// GET /api/v1/pastes - list pastes per AI.md PART 14
func (data *Data) listPastes(rw http.ResponseWriter, req *http.Request) error {
	// Check rate limit
	err := data.RateLimitGet.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Parse query parameters
	query := req.URL.Query()

	limit := 50
	if limitStr := query.Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
			return netshare.ErrBadRequest
		}
		limit = parsedLimit
	}

	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil || parsedOffset < 0 {
			return netshare.ErrBadRequest
		}
		offset = parsedOffset
	}

	// Get paste list from database
	pastes, err := data.DB.PasteList(limit, offset)
	if err != nil {
		return err
	}

	// Build text representation for plain text response
	var textBuilder strings.Builder
	for _, p := range pastes {
		title := p.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Fprintf(&textBuilder, "%s\t%s\n", p.ID, title)
	}

	// Return response with content negotiation per AI.md PART 14, 16
	msg := fmt.Sprintf("%d pastes found", len(pastes))
	return writeSuccess(rw, req, pastes, msg, textBuilder.String())
}


// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package apiv1

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/casjay-forks/caspaste/src/caspasswd"
	"github.com/casjay-forks/caspaste/src/config"
	"github.com/casjay-forks/caspaste/src/netshare"
)

type newPasteAnswer struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	CreateTime int64  `json:"createTime"`
	DeleteTime int64  `json:"deleteTime"`
}

// handlePastes handles all paste operations per AI.md PART 14
// POST /api/v1/pastes - create new paste
// GET /api/v1/pastes?id=X - get single paste
// GET /api/v1/pastes - list pastes
func (data *Data) handlePastes(rw http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case "POST":
		return data.createPaste(rw, req)
	case "GET":
		req.ParseForm()
		id := req.Form.Get("id")
		// Also accept path-based ID: GET /api/v1/pastes/{id}
		if id == "" {
			prefix := config.APIBasePath() + "/pastes/"
			if after, ok := strings.CutPrefix(req.URL.Path, prefix); ok && after != "" {
				id = after
				req.Form.Set("id", id)
			}
		}
		if id != "" {
			return data.getPaste(rw, req)
		}
		return data.listPastes(rw, req)
	default:
		return netshare.ErrMethodNotAllowed
	}
}

// POST /api/v1/pastes - create new paste
func (data *Data) createPaste(rw http.ResponseWriter, req *http.Request) error {
	var err error

	// Check auth (required when server.public=false)
	if !data.Public && data.CasPasswdFile != "" {
		clientIP := netshare.GetClientAddr(req)

		// Check if IP is blocked due to too many failed attempts
		if data.BruteForce != nil && data.BruteForce.CheckBlocked(clientIP) {
			// Return 429 Too Many Requests with retry-after header
			remaining := data.BruteForce.GetRemainingLockout(clientIP)
			rw.Header().Set("Retry-After", strconv.Itoa(int(remaining.Seconds())))
			return netshare.ErrTooManyRequests
		}

		isAuthenticated := false

		user, pass, authProvided := req.BasicAuth()
		if authProvided {
			isAuthenticated, err = caspasswd.LoadAndCheck(data.CasPasswdFile, user, pass)
			if err != nil {
				return err
			}
		}

		if !isAuthenticated {
			// Record failed attempt
			if data.BruteForce != nil {
				data.BruteForce.RecordFailure(clientIP)
			}
			return netshare.ErrUnauthorized
		}

		// Record successful login
		if data.BruteForce != nil {
			data.BruteForce.RecordSuccess(clientIP)
		}
	}

	// Check method
	if req.Method != "POST" {
		return netshare.ErrMethodNotAllowed
	}

	// Enforce body size limit before reading to prevent memory exhaustion.
	maxBytes := int64(data.BodyMaxLen) * 2
	if maxBytes < 1<<20 {
		maxBytes = 1 << 20
	}
	req.Body = http.MaxBytesReader(rw, req.Body, maxBytes)

	// Get form data and create paste
	pasteID, createTime, deleteTime, err := netshare.PasteAddFromForm(req, data.DB, data.RateLimitNew, data.TitleMaxLen, data.BodyMaxLen, data.MaxLifeTime, data.Lexers)
	if err != nil {
		return err
	}

	// Construct full URL for paste
	url := netshare.BuildPasteURL(req, pasteID)

	answer := newPasteAnswer{
		ID:         pasteID,
		URL:        url,
		CreateTime: createTime,
		DeleteTime: deleteTime,
	}

	// Build text representation for plain text response
	var textBuilder strings.Builder
	fmt.Fprintf(&textBuilder, "id: %s\n", answer.ID)
	fmt.Fprintf(&textBuilder, "url: %s\n", answer.URL)
	fmt.Fprintf(&textBuilder, "createTime: %d\n", answer.CreateTime)
	fmt.Fprintf(&textBuilder, "deleteTime: %d\n", answer.DeleteTime)

	// Return response with content negotiation per AI.md PART 14, 16
	return writeSuccess(rw, req, answer, "Paste created", textBuilder.String())
}

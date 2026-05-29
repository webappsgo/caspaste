// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"fmt"
	"net/http"

	"github.com/casjay-forks/caspaste/src/netshare"
)

// GET /qr/{id} - QR code for paste URL
func (data *Data) handleQRCode(rw http.ResponseWriter, req *http.Request) error {
	// Check method
	if req.Method != "GET" {
		return netshare.ErrMethodNotAllowed
	}

	// Get ID from path
	id := req.URL.Path[len("/qr/"):]

	// Build paste URL
	proto := netshare.GetProtocol(req)
	host := netshare.GetHost(req)
	pasteURL := fmt.Sprintf("%s://%s/%s", proto, host, id)

	// Generate QR code using Google Charts API (free, no library needed)
	qrURL := fmt.Sprintf("https://chart.googleapis.com/chart?cht=qr&chs=300x300&chl=%s", pasteURL)

	// Redirect to QR code image
	http.Redirect(rw, req, qrURL, http.StatusFound)
	return nil
}

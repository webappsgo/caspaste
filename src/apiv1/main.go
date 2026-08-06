// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package apiv1

import (
	"net/http"

	"github.com/webappsgo/caspaste/src/netshare"
)

// GET /api/v1/
func (data *Data) MainHand(rw http.ResponseWriter, req *http.Request) {
	data.writeError(rw, req, netshare.ErrNotFound)
}

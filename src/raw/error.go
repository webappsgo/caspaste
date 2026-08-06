// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package raw

import (
	"errors"
	"github.com/webappsgo/caspaste/src/netshare"
	"github.com/webappsgo/caspaste/src/storage"
	"io"
	"net/http"
	"strconv"
)

func (data *Data) writeError(rw http.ResponseWriter, req *http.Request, e error) (int, error) {
	var errText string
	var errCode int

	// Detect error type
	var eTmp429 *netshare.RateLimitError

	if e == storage.ErrNotFoundID || e == netshare.ErrNotFound {
		errCode = 404
		errText = "404 Not Found"

	} else if errors.As(e, &eTmp429) {
		errCode = 429
		errText = "429 Too Many Requests"
		rw.Header().Set("Retry-After", strconv.FormatInt(eTmp429.RetryAfter, 10))

	} else {
		errCode = 500
		errText = "500 Internal Server Error"
	}

	// Write response per AI.md PART 14 (text responses end with newline)
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(errCode)

	_, err := io.WriteString(rw, errText+"\n")
	if err != nil {
		return 500, err
	}

	return errCode, nil
}

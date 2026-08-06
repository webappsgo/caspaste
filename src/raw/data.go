// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package raw

import (
	"github.com/webappsgo/caspaste/src/config"
	"github.com/webappsgo/caspaste/src/logger"
	"github.com/webappsgo/caspaste/src/netshare"
	"github.com/webappsgo/caspaste/src/storage"
	"net/http"
)

type Data struct {
	DB  storage.DB
	Log logger.Logger

	RateLimitGet *netshare.RateLimitSystem

	Version string
}

func Load(db storage.DB, cfg config.Config) *Data {
	return &Data{
		DB:           db,
		Log:          cfg.Log,
		RateLimitGet: cfg.RateLimitGet,
		Version:      cfg.Version,
	}
}

func (data *Data) Hand(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Server", config.Software+"/"+data.Version)

	err := data.rawHand(rw, req)

	if err == nil {
		data.Log.HttpRequest(req, 200)

	} else {
		// Log the original error before writing HTTP response
		data.Log.HttpError(req, err)

		code, writeErr := data.writeError(rw, req, err)
		if writeErr != nil {
			data.Log.HttpError(req, writeErr)
		}
		data.Log.HttpRequest(req, code)
	}
}

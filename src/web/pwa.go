// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"net/http"
)

// Pattern: /manifest.json
func (data *Data) handleManifest(rw http.ResponseWriter, req *http.Request) error {
	manifestJSON, err := embFS.ReadFile("data/manifest.json")
	if err != nil {
		return err
	}

	rw.Header().Set("Content-Type", "application/manifest+json")
	rw.Header().Set("Cache-Control", "public, max-age=3600")
	rw.Write(manifestJSON)
	return nil
}

// Pattern: /sw.js
func (data *Data) handleServiceWorker(rw http.ResponseWriter, req *http.Request) error {
	swJS, err := embFS.ReadFile("data/sw.js")
	if err != nil {
		return err
	}

	rw.Header().Set("Content-Type", "application/javascript")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Service-Worker-Allowed", "/")
	rw.Write(swJS)
	return nil
}

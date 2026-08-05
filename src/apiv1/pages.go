
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package apiv1

import (
	"net/http"

	"github.com/webappsgo/caspaste/src/config"
	"github.com/webappsgo/caspaste/src/netshare"
)

// serverPageType is the JSON envelope for a public server content page
// (about, privacy, contact, help, terms) per AI.md PART 14.
type serverPageType struct {
	Page    string `json:"page"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// handleServerPage returns a public server content page as JSON/text per
// AI.md PART 14. Method is GET only.
func (data *Data) handleServerPage(rw http.ResponseWriter, req *http.Request, page, content string) error {
	if req.Method != "GET" {
		return netshare.ErrMethodNotAllowed
	}

	payload := serverPageType{
		Page:    page,
		Title:   data.ServerTitle,
		Content: content,
	}

	return writeSuccess(rw, req, payload, page, content)
}

// autodiscoverType is the public discovery document per AI.md PART 14.
// It exposes only public, non-sensitive settings: never admin_path or secrets.
type autodiscoverType struct {
	Software    string `json:"software"`
	Version     string `json:"version"`
	Title       string `json:"title"`
	Tagline     string `json:"tagline"`
	APIVersion  string `json:"apiVersion"`
	APIBase     string `json:"apiBase"`
	TitleMaxLen int    `json:"titleMaxlength"`
	BodyMaxLen  int    `json:"bodyMaxlength"`
	MaxLifeTime int64  `json:"maxLifeTime"`
	Public      bool   `json:"public"`
}

// handleAutodiscover returns public server settings for CLI/agent discovery
// per AI.md PART 14. Unversioned, no auth, never exposes admin_path or secrets.
func (data *Data) handleAutodiscover(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return netshare.ErrMethodNotAllowed
	}

	payload := autodiscoverType{
		Software:    config.Software,
		Version:     data.Version,
		Title:       data.ServerTitle,
		Tagline:     data.ServerTagline,
		APIVersion:  config.APIVersion(),
		APIBase:     config.APIBasePath(),
		TitleMaxLen: data.TitleMaxLen,
		BodyMaxLen:  data.BodyMaxLen,
		MaxLifeTime: data.MaxLifeTime,
		Public:      data.Public,
	}

	return writeSuccess(rw, req, payload, "Autodiscover", "")
}

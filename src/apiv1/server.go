
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package apiv1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/casjay-forks/caspaste/src/netshare"
)

type serverInfoType struct {
	Software          string   `json:"software"`
	Version           string   `json:"version"`
	TitleMaxLen       int      `json:"titleMaxlength"`
	BodyMaxLen        int      `json:"bodyMaxlength"`
	MaxLifeTime       int64    `json:"maxLifeTime"`
	ServerAbout       string   `json:"serverAbout"`
	ServerRules       string   `json:"serverRules"`
	ServerTermsOfUse  string   `json:"serverTermsOfUse"`
	AdminName         string   `json:"adminName"`
	AdminMail         string   `json:"adminMail"`
	Syntaxes          []string `json:"syntaxes"`
	UiDefaultLifeTime string   `json:"uiDefaultLifeTime"`
	AuthRequired      bool     `json:"authRequired"`
}

// GET /api/v1/server/info - server information per AI.md PART 14
func (data *Data) handleServerInfo(rw http.ResponseWriter, req *http.Request) error {
	// Check method
	if req.Method != "GET" {
		return netshare.ErrMethodNotAllowed
	}

	// Prepare data
	serverInfo := serverInfoType{
		Software:          "CasPb",
		Version:           data.Version,
		TitleMaxLen:       data.TitleMaxLen,
		BodyMaxLen:        data.BodyMaxLen,
		MaxLifeTime:       data.MaxLifeTime,
		ServerAbout:       data.ServerAbout,
		ServerRules:       data.ServerRules,
		ServerTermsOfUse:  data.ServerTermsOfUse,
		AdminName:         data.AdminName,
		AdminMail:         data.AdminMail,
		Syntaxes:          data.Lexers,
		UiDefaultLifeTime: data.UiDefaultLifeTime,
		AuthRequired:      !data.Public,
	}

	// Build text representation for plain text response
	var textBuilder strings.Builder
	fmt.Fprintf(&textBuilder, "software: %s\n", serverInfo.Software)
	fmt.Fprintf(&textBuilder, "version: %s\n", serverInfo.Version)
	fmt.Fprintf(&textBuilder, "titleMaxLength: %d\n", serverInfo.TitleMaxLen)
	fmt.Fprintf(&textBuilder, "bodyMaxLength: %d\n", serverInfo.BodyMaxLen)
	fmt.Fprintf(&textBuilder, "maxLifeTime: %d\n", serverInfo.MaxLifeTime)
	fmt.Fprintf(&textBuilder, "adminName: %s\n", serverInfo.AdminName)
	fmt.Fprintf(&textBuilder, "adminMail: %s\n", serverInfo.AdminMail)
	fmt.Fprintf(&textBuilder, "authRequired: %t\n", serverInfo.AuthRequired)

	// Return response with content negotiation per AI.md PART 14, 16
	return writeSuccess(rw, req, serverInfo, "Server info", textBuilder.String())
}

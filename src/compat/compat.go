// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package compat provides per-request API compatibility shims that let existing
// tooling built against Lenpaste, Stikked, Microbin, or Hastebin talk to CasPaste
// without modification.
//
// Mode selection order (first match wins):
//  1. CASPASTE_API_MODE env var (set once at startup)
//  2. Host header pattern matching (per-request)
//  3. Native CasPaste API (default, no interception)
package compat

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/casjay-forks/caspaste/src/logger"
	"github.com/casjay-forks/caspaste/src/storage"
)

// Mode identifies which compatibility shim is active.
type Mode string

const (
	// ModeNative passes requests through to the CasPaste native API unchanged.
	ModeNative   Mode = "native"
	ModeLenpaste Mode = "lenpaste"
	ModeStikked  Mode = "stikked"
	ModeMicrobin Mode = "microbin"
	ModeHastebin Mode = "hastebin"
	ModePastebin Mode = "pastebin"
	ModeTermbin  Mode = "termbin"
)

// Data holds everything compat handlers need.
type Data struct {
	DB  storage.DB
	Log logger.Logger

	// EnvMode is the mode set by CASPASTE_API_MODE.
	EnvMode Mode
	// ForceHost, when true (default), makes the Host header override EnvMode.
	// Set CASPASTE_FORCE_HOST=no to make EnvMode take precedence.
	ForceHost bool

	// Server metadata used by info/status endpoints.
	Version     string
	BaseURL     string
	ServerTitle string
	AdminName   string
	AdminMail   string
	ServerAbout string
	ServerRules string
	TitleMaxLen int
	BodyMaxLen  int
	MaxLifeTime int64
}

// Load constructs a Data from the environment and provided fields.
// Call once at server startup.
func Load(db storage.DB, log logger.Logger, version, baseURL, title, adminName, adminMail, about, rules string, titleMax, bodyMax int, maxLife int64) *Data {
	envMode := normalizeMode(os.Getenv("CASPASTE_API_MODE"))

	// CASPASTE_FORCE_HOST controls whether the Host header overrides the env mode.
	// Default: yes (host header wins). Set to "no"/"false"/"0" to let env mode win.
	forceHost := true
	if fh := strings.ToLower(strings.TrimSpace(os.Getenv("CASPASTE_FORCE_HOST"))); fh == "no" || fh == "false" || fh == "0" {
		forceHost = false
	}

	return &Data{
		DB:          db,
		Log:         log,
		EnvMode:     envMode,
		ForceHost:   forceHost,
		Version:     version,
		BaseURL:     strings.TrimRight(baseURL, "/"),
		ServerTitle: title,
		AdminName:   adminName,
		AdminMail:   adminMail,
		ServerAbout: about,
		ServerRules: rules,
		TitleMaxLen: titleMax,
		BodyMaxLen:  bodyMax,
		MaxLifeTime: maxLife,
	}
}

// normalizeMode lowercases and maps common aliases to canonical Mode values.
func normalizeMode(s string) Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "lenpaste", "lp":
		return ModeLenpaste
	case "stikked", "stikq", "sk":
		return ModeStikked
	case "microbin", "mb":
		return ModeMicrobin
	case "hastebin", "haste":
		return ModeHastebin
	case "pastebin", "pb":
		return ModePastebin
	case "termbin", "tb", "nc":
		return ModeTermbin
	default:
		return ModeNative
	}
}

// detectMode returns the active Mode for this request.
//
// When ForceHost is true (default): Host header is checked first; EnvMode is
// the fallback when the host pattern does not match anything.
// When ForceHost is false: EnvMode wins; Host header is only consulted when
// EnvMode is native/empty.
func (d *Data) detectMode(req *http.Request) Mode {
	if d.ForceHost {
		if hostMode := modeFromHost(req.Host); hostMode != ModeNative {
			return hostMode
		}
		if d.EnvMode != "" {
			return d.EnvMode
		}
		return ModeNative
	}

	if d.EnvMode != "" && d.EnvMode != ModeNative {
		return d.EnvMode
	}
	return modeFromHost(req.Host)
}

// modeFromHost infers the compat mode from the Host header value.
// Patterns are checked against the leftmost label of the hostname:
//
//	lp, lenpaste           → lenpaste
//	mb, microbin           → microbin
//	sk, stikked, stikq     → stikked
//	haste, hastebin        → hastebin
func modeFromHost(host string) Mode {
	// Strip port if present.
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ModeNative
	}

	// Extract the first label (leftmost subdomain or bare hostname).
	first := host
	if idx := strings.Index(host, "."); idx != -1 {
		first = host[:idx]
	}

	switch first {
	case "lp", "lenpaste":
		return ModeLenpaste
	case "mb", "microbin":
		return ModeMicrobin
	case "sk", "stikked", "stikq":
		return ModeStikked
	case "haste", "hastebin":
		return ModeHastebin
	case "pb", "pastebin":
		return ModePastebin
	case "tb", "termbin", "nc":
		return ModeTermbin
	}
	return ModeNative
}

// Middleware returns an http.Handler that intercepts requests whose path belongs
// to the detected compat mode, handles them, and passes everything else to next.
func (d *Data) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		mode := d.detectMode(req)
		switch mode {
		case ModeLenpaste:
			if d.handleLenpaste(rw, req) {
				return
			}
		case ModeStikked:
			if d.handleStikked(rw, req) {
				return
			}
		case ModeMicrobin:
			if d.handleMicrobin(rw, req) {
				return
			}
		case ModeHastebin:
			if d.handleHastebin(rw, req) {
				return
			}
		case ModePastebin:
			if d.handlePastebin(rw, req) {
				return
			}
		case ModeTermbin:
			if d.handleTermbin(rw, req) {
				return
			}
		}
		next.ServeHTTP(rw, req)
	})
}

// jsonOK writes v as JSON with status 200.
func jsonOK(rw http.ResponseWriter, v interface{}) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(rw)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

// jsonErr writes a JSON error body with the given status code.
func jsonErr(rw http.ResponseWriter, status int, message string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	json.NewEncoder(rw).Encode(map[string]string{"error": message})
}

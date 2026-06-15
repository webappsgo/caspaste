// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package httputil provides HTTP utility functions per AI.md PART 14
// Content negotiation, client detection, and response format handling.
package httputil

import (
	"net/http"
	"strings"
)

// ProjectName is used for client detection
const ProjectName = "caspb"

// IsOurCliClient detects our own client binary (caspb-cli)
// Client is INTERACTIVE (TUI/GUI) - receives JSON, renders itself
func IsOurCliClient(r *http.Request) bool {
	ua := r.Header.Get("User-Agent")
	return strings.HasPrefix(ua, ProjectName+"-cli/")
}

// IsTextBrowser detects text-mode browsers (lynx, w3m, links, etc.)
// Text browsers are INTERACTIVE but do NOT support JavaScript
// They receive no-JS HTML alternative (server-rendered, standard form POST)
func IsTextBrowser(r *http.Request) bool {
	ua := strings.ToLower(r.Header.Get("User-Agent"))

	// Text browsers - INTERACTIVE, NO JavaScript support
	// Format: "browser/" or "browser " (links uses space)
	textBrowsers := []string{
		"lynx/",
		"w3m/",
		"links ",
		"links/",
		"elinks/",
		"browsh/",
		"carbonyl/",
		"netsurf",
	}
	for _, browser := range textBrowsers {
		if strings.Contains(ua, browser) {
			return true
		}
	}
	return false
}

// IsHttpTool detects HTTP tools (curl, wget, httpie, etc.)
// HTTP tools are NON-INTERACTIVE - they just dump output
func IsHttpTool(r *http.Request) bool {
	ua := strings.ToLower(r.Header.Get("User-Agent"))

	httpTools := []string{
		"curl/", "wget/", "httpie/",
		"libcurl/", "python-requests/",
		"go-http-client/", "axios/", "node-fetch/",
	}
	for _, tool := range httpTools {
		if strings.Contains(ua, tool) {
			return true
		}
	}

	// No User-Agent = likely HTTP tool (non-interactive)
	if ua == "" {
		return true
	}

	return false
}

// IsNonInteractiveClient detects clients that need pre-formatted text
// ONLY HTTP tools are non-interactive
// Our client and text browsers are INTERACTIVE (handle their own rendering)
func IsNonInteractiveClient(r *http.Request) bool {
	// Our client is INTERACTIVE - receives JSON
	if IsOurCliClient(r) {
		return false
	}

	// Text browsers are INTERACTIVE - receive no-JS HTML, render it themselves
	if IsTextBrowser(r) {
		return false
	}

	// HTTP tools are NON-INTERACTIVE - need pre-formatted text
	if IsHttpTool(r) {
		return true
	}

	return false
}

// ResponseFormat represents the type of response to send
type ResponseFormat string

const (
	FormatJSON ResponseFormat = "application/json"
	FormatText ResponseFormat = "text/plain"
	FormatHTML ResponseFormat = "text/html"
)

// DetectResponseFormat determines the response format based on request
// Per AI.md PART 14 content negotiation rules
func DetectResponseFormat(r *http.Request) ResponseFormat {
	// 1. Check for .txt extension
	if strings.HasSuffix(r.URL.Path, ".txt") {
		return FormatText
	}

	// 2. Check Accept header
	accept := r.Header.Get("Accept")

	switch {
	case strings.Contains(accept, "application/json"):
		return FormatJSON
	case strings.Contains(accept, "text/plain"):
		return FormatText
	case strings.Contains(accept, "text/html"):
		return FormatHTML
	default:
		// 3. Default based on endpoint
		if strings.HasPrefix(r.URL.Path, "/api/") {
			return FormatJSON
		}
		return FormatHTML
	}
}

// GetAPIResponseFormat determines format for /api/** routes
// Returns raw data as plain text - no HTML conversion needed
func GetAPIResponseFormat(r *http.Request) ResponseFormat {
	// 1. Check .txt extension
	if strings.HasSuffix(r.URL.Path, ".txt") {
		return FormatText
	}

	// 2. Explicit Accept header wins over client-type heuristics
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/json") {
		return FormatJSON
	}
	if strings.Contains(accept, "text/plain") {
		return FormatText
	}

	// 3. Non-interactive clients (curl, wget, etc.) default to plain text
	if IsNonInteractiveClient(r) {
		return FormatText
	}

	// 4. Default to JSON
	return FormatJSON
}

// GetFrontendResponseFormat determines format for frontend routes
// Per AI.md PART 14 - smart detection based on User-Agent and Accept headers
func GetFrontendResponseFormat(r *http.Request) ResponseFormat {
	// 1. Check Accept header for explicit preference
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/html") {
		return FormatHTML
	}
	if strings.Contains(accept, "text/plain") {
		return FormatText
	}

	// 2. Our client gets JSON
	if IsOurCliClient(r) {
		return FormatJSON
	}

	// 3. Text browsers get HTML (no-JS)
	if IsTextBrowser(r) {
		return FormatHTML
	}

	// 4. HTTP tools get plain text
	if IsHttpTool(r) {
		return FormatText
	}

	// 5. Default to HTML for browsers
	return FormatHTML
}

// StripTxtExtension removes .txt extension from path if present
// Used when routing requests that use .txt for format negotiation
func StripTxtExtension(path string) string {
	if strings.HasSuffix(path, ".txt") {
		return strings.TrimSuffix(path, ".txt")
	}
	return path
}

// DetectClientType detects request type and returns "html", "text", or "json"
// Per AI.md PART 16 Smart Content Detection specification
func DetectClientType(r *http.Request) string {
	// 1. Check Accept header first (explicit preference)
	accept := r.Header.Get("Accept")

	if strings.Contains(accept, "text/html") {
		return "html"
	}
	if strings.Contains(accept, "text/plain") {
		return "text"
	}
	if strings.Contains(accept, "application/json") {
		// Rare for frontend, but support it
		return "json"
	}

	// 2. Check User-Agent for browser detection
	ua := r.Header.Get("User-Agent")

	// Browser User-Agents (common patterns)
	browsers := []string{
		"Mozilla/", "Chrome/", "Safari/", "Edge/", "Firefox/",
		"Opera/", "MSIE", "Trident/",
	}

	for _, browser := range browsers {
		if strings.Contains(ua, browser) {
			return "html"
		}
	}

	// 3. CLI tools (curl, wget, httpie, etc.)
	cliTools := []string{
		"curl/", "Wget/", "HTTPie/", "python-requests/",
		"Go-http-client/", "node-fetch/",
	}

	for _, tool := range cliTools {
		if strings.Contains(ua, tool) {
			return "text"
		}
	}

	// 4. Empty or unknown User-Agent
	if ua == "" {
		// Default to text for programmatic access
		return "text"
	}

	// 5. Default: HTML (safest fallback)
	return "html"
}

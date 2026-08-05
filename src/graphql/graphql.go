// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package graphql provides GraphQL API per AI.md PART 14
// Routes: /server/docs/graphql (GraphiQL UI) and /api/graphql (queries) per AI.md PART 14
package graphql

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Config holds GraphQL configuration
type Config struct {
	Title   string
	Version string
}

// Handler provides HTTP handlers for GraphQL
type Handler struct {
	cfg       *Config
	schema    *Schema
	resolvers *Resolvers
}

// NewHandler creates a new GraphQL handler
func NewHandler(cfg *Config, resolvers *Resolvers) *Handler {
	return &Handler{
		cfg:       cfg,
		schema:    NewSchema(),
		resolvers: resolvers,
	}
}

// Request represents a GraphQL request
type Request struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// Response represents a GraphQL response
type Response struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []Error     `json:"errors,omitempty"`
}

// Error represents a GraphQL error
type Error struct {
	Message   string     `json:"message"`
	Locations []Location `json:"locations,omitempty"`
	Path      []string   `json:"path,omitempty"`
}

// Location represents error location
type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// ServeHTTP handles GraphQL requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Serve GraphiQL UI
		h.ServeGraphiQL(w, r)
	case http.MethodPost:
		// Handle GraphQL query
		h.HandleQuery(w, r)
	case http.MethodOptions:
		// CORS preflight
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleQuery processes GraphQL queries
func (h *Handler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	var req Request

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, "Invalid JSON: "+err.Error())
			return
		}
	} else if strings.Contains(contentType, "application/graphql") {
		// Raw GraphQL query in body
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		req.Query = string(buf)
	} else {
		// Form data
		if err := r.ParseForm(); err == nil {
			req.Query = r.FormValue("query")
			req.OperationName = r.FormValue("operationName")
			if vars := r.FormValue("variables"); vars != "" {
				json.Unmarshal([]byte(vars), &req.Variables)
			}
		}
	}

	if req.Query == "" {
		h.writeError(w, "No query provided")
		return
	}

	// Execute query
	result := h.execute(req)

	w.Header().Set("Content-Type", "application/json")
	data, _ := json.MarshalIndent(result, "", "  ")
	w.Write(data)
	w.Write([]byte("\n"))
}

// execute runs the GraphQL query
func (h *Handler) execute(req Request) *Response {
	query := strings.TrimSpace(req.Query)

	// Parse and execute query
	if strings.HasPrefix(query, "query") || strings.HasPrefix(query, "{") {
		return h.executeQuery(query, req.Variables)
	} else if strings.HasPrefix(query, "mutation") {
		return h.executeMutation(query, req.Variables)
	} else if strings.HasPrefix(query, "__schema") || strings.Contains(query, "__schema") {
		return h.executeIntrospection(query)
	}

	return &Response{
		Errors: []Error{{Message: "Unsupported operation"}},
	}
}

// executeQuery executes a query operation
func (h *Handler) executeQuery(query string, variables map[string]interface{}) *Response {
	data := make(map[string]interface{})

	// Parse query fields
	if strings.Contains(query, "healthz") || strings.Contains(query, "health") {
		data["healthz"] = h.resolvers.ResolveHealth()
	}

	if strings.Contains(query, "serverInfo") {
		data["serverInfo"] = h.resolvers.ResolveServerInfo()
	}

	if strings.Contains(query, "paste(") || strings.Contains(query, "paste {") {
		// Extract ID from query
		id := extractArgument(query, "id")
		if id != "" {
			paste, err := h.resolvers.ResolvePaste(id)
			if err != nil {
				return &Response{Errors: []Error{{Message: err.Error()}}}
			}
			data["paste"] = paste
		}
	}

	if strings.Contains(query, "pastes") {
		pastes, err := h.resolvers.ResolvePastes()
		if err != nil {
			return &Response{Errors: []Error{{Message: err.Error()}}}
		}
		data["pastes"] = pastes
	}

	return &Response{Data: data}
}

// executeMutation executes a mutation operation
func (h *Handler) executeMutation(query string, variables map[string]interface{}) *Response {
	data := make(map[string]interface{})

	if strings.Contains(query, "createPaste") {
		// Extract input from variables or query
		input := variables["input"]
		if input == nil {
			input = extractMutationInput(query)
		}

		if inputMap, ok := input.(map[string]interface{}); ok {
			result, err := h.resolvers.ResolveCreatePaste(inputMap)
			if err != nil {
				return &Response{Errors: []Error{{Message: err.Error()}}}
			}
			data["createPaste"] = result
		} else {
			return &Response{Errors: []Error{{Message: "Invalid input for createPaste"}}}
		}
	}

	return &Response{Data: data}
}

// executeIntrospection handles schema introspection
func (h *Handler) executeIntrospection(query string) *Response {
	return &Response{
		Data: map[string]interface{}{
			"__schema": h.schema.Introspect(),
		},
	}
}

// writeError writes a GraphQL error response
func (h *Handler) writeError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	resp := &Response{
		Errors: []Error{{Message: message}},
	}
	data, _ := json.MarshalIndent(resp, "", "  ")
	w.Write(data)
	w.Write([]byte("\n"))
}

// ServeGraphiQL serves the GraphiQL UI
func (h *Handler) ServeGraphiQL(w http.ResponseWriter, r *http.Request) {
	theme := "light"
	if cookie, err := r.Cookie("theme"); err == nil {
		theme = cookie.Value
	}

	html := generateGraphiQLHTML(h.cfg.Title, theme)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// extractArgument extracts an argument value from query
func extractArgument(query, name string) string {
	// Simple extraction: find name: "value" or name: $var
	pattern := name + `:`
	idx := strings.Index(query, pattern)
	if idx == -1 {
		return ""
	}

	rest := query[idx+len(pattern):]
	rest = strings.TrimSpace(rest)

	if strings.HasPrefix(rest, `"`) {
		// Quoted string
		end := strings.Index(rest[1:], `"`)
		if end != -1 {
			return rest[1 : end+1]
		}
	}

	// Unquoted (variable or simple value)
	end := strings.IndexAny(rest, " ,)}\n")
	if end == -1 {
		return rest
	}
	return rest[:end]
}

// extractMutationInput extracts input from mutation query
func extractMutationInput(query string) map[string]interface{} {
	// Simple extraction for inline input
	input := make(map[string]interface{})

	if strings.Contains(query, "title:") {
		input["title"] = extractArgument(query, "title")
	}
	if strings.Contains(query, "body:") {
		input["body"] = extractArgument(query, "body")
	}
	if strings.Contains(query, "syntax:") {
		input["syntax"] = extractArgument(query, "syntax")
	}

	return input
}

// generateGraphiQLHTML generates the GraphiQL HTML page
func generateGraphiQLHTML(title, theme string) string {
	isDark := strings.Contains(theme, "dark")

	css := GraphiQLLightCSS
	if isDark {
		css = GraphiQLDarkCSS
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>` + title + ` - GraphQL API</title>
  <link rel="stylesheet" href="https://unpkg.com/graphiql@3/graphiql.min.css">
  <style>` + css + `</style>
</head>
<body>
  <div id="graphiql"></div>
  <script src="https://unpkg.com/react@18/umd/react.production.min.js"></script>
  <script src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js"></script>
  <script src="https://unpkg.com/graphiql@3/graphiql.min.js"></script>
  <script>
    const fetcher = GraphiQL.createFetcher({
      url: '/api/graphql',
    });

    ReactDOM.createRoot(document.getElementById('graphiql')).render(
      React.createElement(GraphiQL, {
        fetcher: fetcher,
        defaultEditorToolsVisibility: true,
      })
    );
  </script>
</body>
</html>`
}

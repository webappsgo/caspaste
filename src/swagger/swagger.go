// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package swagger provides OpenAPI/Swagger documentation per AI.md PART 14
// Routes: /server/docs/swagger (UI) and /api/swagger (spec) per AI.md PART 14
package swagger

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/webappsgo/caspaste/src/config"
)

// Config holds swagger configuration
type Config struct {
	Title       string
	Description string
	Version     string
	BasePath    string
	Host        string
	Scheme      string
}

// Spec represents the OpenAPI 3.0 specification
type Spec struct {
	OpenAPI string                 `json:"openapi"`
	Info    Info                   `json:"info"`
	Servers []Server               `json:"servers"`
	Paths   map[string]PathItem    `json:"paths"`
	Tags    []Tag                  `json:"tags"`
	Components Components          `json:"components,omitempty"`
}

// Info represents API info
type Info struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Version     string  `json:"version"`
	Contact     Contact `json:"contact,omitempty"`
	License     License `json:"license,omitempty"`
}

// Contact represents contact info
type Contact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// License represents license info
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Server represents a server
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Tag represents an API tag
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PathItem represents a path item
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
}

// Operation represents an operation
type Operation struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
}

// Parameter represents a parameter
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody represents a request body
type RequestBody struct {
	Description string             `json:"description,omitempty"`
	Required    bool               `json:"required,omitempty"`
	Content     map[string]Media   `json:"content"`
}

// Media represents media type content
type Media struct {
	Schema *Schema `json:"schema,omitempty"`
}

// Response represents a response
type Response struct {
	Description string           `json:"description"`
	Content     map[string]Media `json:"content,omitempty"`
}

// Schema represents a JSON schema
type Schema struct {
	Type       string             `json:"type,omitempty"`
	Format     string             `json:"format,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Ref        string             `json:"$ref,omitempty"`
}

// Components holds reusable components
type Components struct {
	Schemas map[string]*Schema `json:"schemas,omitempty"`
}

// Handler provides HTTP handlers for OpenAPI endpoints
type Handler struct {
	spec *Spec
	cfg  *Config
}

// NewHandler creates a new swagger handler
func NewHandler(cfg *Config) *Handler {
	h := &Handler{cfg: cfg}
	h.spec = h.generateSpec()
	return h
}

// generateSpec generates the OpenAPI specification for CasPaste
func (h *Handler) generateSpec() *Spec {
	baseURL := h.cfg.Scheme + "://" + h.cfg.Host
	if h.cfg.BasePath != "" {
		baseURL += h.cfg.BasePath
	}

	return &Spec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       h.cfg.Title,
			Description: h.cfg.Description,
			Version:     h.cfg.Version,
			License: License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		Servers: []Server{
			{URL: baseURL, Description: "Current server"},
		},
		Tags: []Tag{
			{Name: "pastes", Description: "Paste operations"},
			{Name: "server", Description: "Server information"},
			{Name: "health", Description: "Health check endpoints"},
		},
		Paths: map[string]PathItem{
			config.APIBasePath() + "/healthz": {
				Get: &Operation{
					Tags:        []string{"health"},
					Summary:     "Health check",
					Description: "Returns server health status",
					OperationID: "getHealth",
					Responses: map[string]Response{
						"200": {
							Description: "Server is healthy",
							Content: map[string]Media{
								"application/json": {
									Schema: &Schema{
										Type: "object",
										Properties: map[string]*Schema{
											"ok":      {Type: "boolean"},
											"status":  {Type: "string"},
											"version": {Type: "string"},
										},
									},
								},
							},
						},
					},
				},
			},
			config.APIBasePath() + "/pastes": {
				Get: &Operation{
					Tags:        []string{"pastes"},
					Summary:     "Get paste(s)",
					Description: "Get a single paste by ID (with ?id=X) or list public pastes (without id parameter)",
					OperationID: "getPastes",
					Parameters: []Parameter{
						{
							Name:        "id",
							In:          "query",
							Description: "Paste ID (optional - if provided, returns single paste; if omitted, returns list)",
							Required:    false,
							Schema:      &Schema{Type: "string"},
						},
						{
							Name:        "limit",
							In:          "query",
							Description: "Maximum number of pastes to return (1-100, default 50)",
							Required:    false,
							Schema:      &Schema{Type: "integer"},
						},
						{
							Name:        "offset",
							In:          "query",
							Description: "Number of pastes to skip for pagination",
							Required:    false,
							Schema:      &Schema{Type: "integer"},
						},
					},
					Responses: map[string]Response{
						"200": {
							Description: "Paste(s) found - returns single Paste object if id provided, or array of PasteSummary if listing",
							Content: map[string]Media{
								"application/json": {
									Schema: &Schema{Ref: "#/components/schemas/Paste"},
								},
							},
						},
						"404": {
							Description: "Paste not found (when id is provided)",
							Content: map[string]Media{
								"application/json": {
									Schema: &Schema{Ref: "#/components/schemas/Error"},
								},
							},
						},
					},
				},
				Post: &Operation{
					Tags:        []string{"pastes"},
					Summary:     "Create a new paste",
					Description: "Creates a new paste with optional syntax highlighting, expiration, and privacy settings",
					OperationID: "createPaste",
					RequestBody: &RequestBody{
						Required:    true,
						Description: "Paste data",
						Content: map[string]Media{
							"application/x-www-form-urlencoded": {
								Schema: &Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"title":      {Type: "string"},
										"body":       {Type: "string"},
										"syntax":     {Type: "string"},
										"expiration": {Type: "string"},
										"one_use":    {Type: "boolean"},
										"private":    {Type: "boolean"},
									},
								},
							},
							"multipart/form-data": {
								Schema: &Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"title":      {Type: "string"},
										"body":       {Type: "string"},
										"file":       {Type: "string", Format: "binary"},
										"syntax":     {Type: "string"},
										"expiration": {Type: "string"},
										"one_use":    {Type: "boolean"},
										"private":    {Type: "boolean"},
									},
								},
							},
						},
					},
					Responses: map[string]Response{
						"200": {
							Description: "Paste created successfully",
							Content: map[string]Media{
								"application/json": {
									Schema: &Schema{Ref: "#/components/schemas/PasteResponse"},
								},
							},
						},
						"400": {
							Description: "Bad request",
							Content: map[string]Media{
								"application/json": {
									Schema: &Schema{Ref: "#/components/schemas/Error"},
								},
							},
						},
					},
				},
			},
			config.APIBasePath() + "/server/info": {
				Get: &Operation{
					Tags:        []string{"server"},
					Summary:     "Get server information",
					Description: "Returns server configuration and metadata",
					OperationID: "getServerInfo",
					Responses: map[string]Response{
						"200": {
							Description: "Server information",
							Content: map[string]Media{
								"application/json": {
									Schema: &Schema{Ref: "#/components/schemas/ServerInfo"},
								},
							},
						},
					},
				},
			},
		},
		Components: Components{
			Schemas: map[string]*Schema{
				"Paste": {
					Type: "object",
					Properties: map[string]*Schema{
						"id":          {Type: "string"},
						"title":       {Type: "string"},
						"body":        {Type: "string"},
						"syntax":      {Type: "string"},
						"create_time": {Type: "integer", Format: "int64"},
						"delete_time": {Type: "integer", Format: "int64"},
						"one_use":     {Type: "boolean"},
						"is_private":  {Type: "boolean"},
						"is_file":     {Type: "boolean"},
						"file_name":   {Type: "string"},
						"mime_type":   {Type: "string"},
						"is_url":      {Type: "boolean"},
					},
				},
				"PasteSummary": {
					Type: "object",
					Properties: map[string]*Schema{
						"id":          {Type: "string"},
						"title":       {Type: "string"},
						"syntax":      {Type: "string"},
						"create_time": {Type: "integer", Format: "int64"},
					},
				},
				"PasteResponse": {
					Type: "object",
					Properties: map[string]*Schema{
						"id":  {Type: "string"},
						"url": {Type: "string"},
					},
				},
				"ServerInfo": {
					Type: "object",
					Properties: map[string]*Schema{
						"version":       {Type: "string"},
						"title":         {Type: "string"},
						"public":        {Type: "boolean"},
						"lexers":        {Type: "array", Items: &Schema{Type: "string"}},
						"max_body_len":  {Type: "integer"},
						"max_title_len": {Type: "integer"},
					},
				},
				"Error": {
					Type: "object",
					Properties: map[string]*Schema{
						"code":  {Type: "integer"},
						"error": {Type: "string"},
					},
				},
			},
		},
	}
}

// ServeSpec serves the OpenAPI JSON spec
func (h *Handler) ServeSpec(w http.ResponseWriter, r *http.Request) {
	// Update server URL based on request
	spec := h.generateSpecWithRequest(r)

	w.Header().Set("Content-Type", "application/json")
	data, _ := json.MarshalIndent(spec, "", "  ")
	w.Write(data)
	w.Write([]byte("\n"))
}

// generateSpecWithRequest generates spec with dynamic server URL
func (h *Handler) generateSpecWithRequest(r *http.Request) *Spec {
	spec := h.generateSpec()

	// Detect scheme from request
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	// Build server URL
	host := r.Host
	if fwdHost := r.Header.Get("X-Forwarded-Host"); fwdHost != "" {
		host = fwdHost
	}

	serverURL := scheme + "://" + host
	spec.Servers = []Server{
		{URL: serverURL, Description: "Current server"},
	}

	return spec
}

// ServeUI serves the Swagger UI HTML page
func (h *Handler) ServeUI(w http.ResponseWriter, r *http.Request) {
	theme := "light"
	if cookie, err := r.Cookie("theme"); err == nil {
		theme = cookie.Value
	}

	html := generateSwaggerUIHTML(theme)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// generateSwaggerUIHTML generates the Swagger UI HTML with theme support
func generateSwaggerUIHTML(theme string) string {
	isDark := strings.Contains(theme, "dark")

	css := SwaggerLightCSS
	if isDark {
		css = SwaggerDarkCSS
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>CasPaste API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>` + css + `</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      SwaggerUIBundle({
        url: "/api/swagger",
        dom_id: '#swagger-ui',
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIBundle.SwaggerUIStandalonePreset
        ],
        layout: "BaseLayout",
        deepLinking: true,
        displayRequestDuration: true,
        filter: true,
        showExtensions: true,
        showCommonExtensions: true
      });
    };
  </script>
</body>
</html>`
}

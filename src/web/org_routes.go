// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"net/http"
	"strings"
)

// handleOrgsList handles GET /orgs
func (data *Data) handleOrgsList(rw http.ResponseWriter, req *http.Request) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderOrgsList(rw, req, authUser)
}

// handleOrgNew handles GET/POST /orgs/new
func (data *Data) handleOrgNew(rw http.ResponseWriter, req *http.Request) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	if req.Method == http.MethodPost {
		http.Redirect(rw, req, "/api/v1/orgs", http.StatusSeeOther)
		return nil
	}

	return data.renderOrgNew(rw, req, authUser)
}

// handleOrgView handles GET /orgs/{slug}
func (data *Data) handleOrgView(rw http.ResponseWriter, req *http.Request, slug string) error {
	authUser := GetAuthUser(req.Context())

	return data.renderOrgView(rw, req, slug, authUser)
}

// handleOrgSettings handles GET/POST /orgs/{slug}/settings
func (data *Data) handleOrgSettings(rw http.ResponseWriter, req *http.Request, slug string) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderOrgSettings(rw, req, slug, authUser)
}

// handleOrgMembers handles GET /orgs/{slug}/members
func (data *Data) handleOrgMembers(rw http.ResponseWriter, req *http.Request, slug string) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderOrgMembers(rw, req, slug, authUser)
}

// handleOrgTokens handles GET /orgs/{slug}/tokens
func (data *Data) handleOrgTokens(rw http.ResponseWriter, req *http.Request, slug string) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderOrgTokens(rw, req, slug, authUser)
}

// handleOrgDomains handles GET /orgs/{slug}/domains
func (data *Data) handleOrgDomains(rw http.ResponseWriter, req *http.Request, slug string) error {
	authUser := GetAuthUser(req.Context())
	if authUser == nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return nil
	}

	return data.renderOrgDomains(rw, req, slug, authUser)
}

// routeOrgs routes /orgs/* paths
func (data *Data) routeOrgs(rw http.ResponseWriter, req *http.Request) error {
	path := req.URL.Path

	if path == "/orgs" || path == "/orgs/" {
		return data.handleOrgsList(rw, req)
	}

	if path == "/orgs/new" {
		return data.handleOrgNew(rw, req)
	}

	parts := strings.Split(strings.TrimPrefix(path, "/orgs/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return data.handleOrgsList(rw, req)
	}

	slug := parts[0]

	if len(parts) == 1 {
		return data.handleOrgView(rw, req, slug)
	}

	switch parts[1] {
	case "settings":
		return data.handleOrgSettings(rw, req, slug)
	case "members":
		return data.handleOrgMembers(rw, req, slug)
	case "tokens":
		return data.handleOrgTokens(rw, req, slug)
	case "domains":
		return data.handleOrgDomains(rw, req, slug)
	}

	return data.handleOrgView(rw, req, slug)
}

// orgNavLinks returns the standard navigation links for an org page.
func orgNavLinks(slug string) []stubLink {
	return []stubLink{
		{URL: "/orgs/" + slug + "/settings", Label: "Settings"},
		{URL: "/orgs/" + slug + "/members", Label: "Members"},
		{URL: "/orgs/" + slug + "/tokens", Label: "API Tokens"},
		{URL: "/orgs/" + slug + "/domains", Label: "Custom Domains"},
	}
}

func (data *Data) renderOrgsList(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Organizations",
		Description: "Manage your organizations.",
		Notice:      "<a href=\"/orgs/new\" class=\"button\">Create Organization</a>",
		BackURL:     "/users",
		BackLabel:   "Back to Dashboard",
	})
}

func (data *Data) renderOrgNew(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Create Organization",
		FormAction:  "/api/v1/orgs",
		FormMethod:  "POST",
		SubmitLabel: "Create Organization",
		Fields: []stubField{
			{ID: "slug", Name: "slug", Label: "Slug (URL-friendly name)", Type: "text",
				Pattern: "[a-z0-9-]+", Required: true,
				Hint: "Lowercase letters, numbers, and hyphens only"},
			{ID: "name", Name: "name", Label: "Display Name", Type: "text", Required: true},
			{ID: "description", Name: "description", Label: "Description", Type: "textarea"},
			{ID: "visibility", Name: "visibility", Label: "Visibility", Type: "select",
				Options: []stubSelectOption{
					{Value: "public", Label: "Public"},
					{Value: "private", Label: "Private"},
				}},
		},
		BackURL:   "/orgs",
		BackLabel: "Cancel",
	})
}

func (data *Data) renderOrgView(rw http.ResponseWriter, req *http.Request, slug string, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       slug,
		Description: "Organization overview. Use the API to manage this organization: GET /api/v1/orgs/" + slug,
		Links:       orgNavLinks(slug),
		BackURL:     "/orgs",
		BackLabel:   "Back to Organizations",
	})
}

func (data *Data) renderOrgSettings(rw http.ResponseWriter, req *http.Request, slug string, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Settings: " + slug,
		FormAction:  "/api/v1/orgs/" + slug,
		FormMethod:  "POST",
		SubmitLabel: "Save Changes",
		Fields: []stubField{
			{ID: "name", Name: "name", Label: "Display Name", Type: "text"},
			{ID: "description", Name: "description", Label: "Description", Type: "textarea"},
		},
		BackURL:   "/orgs/" + slug,
		BackLabel: "Back to Organization",
	})
}

func (data *Data) renderOrgMembers(rw http.ResponseWriter, req *http.Request, slug string, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Members: " + slug,
		Description: "View members via the API: GET /api/v1/orgs/" + slug + "/members",
		FormAction:  "/api/v1/orgs/" + slug + "/members",
		FormMethod:  "POST",
		SubmitLabel: "Add Member",
		Fields: []stubField{
			{ID: "username", Name: "username", Label: "Username", Type: "text",
				Placeholder: "Username", Required: true},
			{ID: "role", Name: "role", Label: "Role", Type: "select",
				Options: []stubSelectOption{
					{Value: "member", Label: "Member"},
					{Value: "admin", Label: "Admin"},
				}},
		},
		BackURL:   "/orgs/" + slug,
		BackLabel: "Back to Organization",
	})
}

func (data *Data) renderOrgTokens(rw http.ResponseWriter, req *http.Request, slug string, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "API Tokens: " + slug,
		Description: "View tokens via the API: GET /api/v1/orgs/" + slug + "/tokens",
		FormAction:  "/api/v1/orgs/" + slug + "/tokens",
		FormMethod:  "POST",
		SubmitLabel: "Create Token",
		Fields: []stubField{
			{ID: "name", Name: "name", Label: "Token Name", Type: "text",
				Placeholder: "Token Name", Required: true},
			{ID: "scopes", Name: "scopes", Label: "Scopes", Type: "select",
				Options: []stubSelectOption{
					{Value: "read", Label: "Read Only"},
					{Value: "read-write", Label: "Read/Write"},
					{Value: "global", Label: "Full Access"},
				}},
		},
		BackURL:   "/orgs/" + slug,
		BackLabel: "Back to Organization",
	})
}

func (data *Data) renderOrgDomains(rw http.ResponseWriter, req *http.Request, slug string, user *AuthUser) error {
	return data.renderStub(rw, req, stubTmplData{
		Title:       "Custom Domains: " + slug,
		Description: "View domains via the API: GET /api/v1/orgs/" + slug + "/domains",
		FormAction:  "/api/v1/orgs/" + slug + "/domains",
		FormMethod:  "POST",
		SubmitLabel: "Add Domain",
		Fields: []stubField{
			{ID: "domain", Name: "domain", Label: "Domain", Type: "text",
				Placeholder: "yourdomain.com", Required: true},
		},
		BackURL:   "/orgs/" + slug,
		BackLabel: "Back to Organization",
	})
}

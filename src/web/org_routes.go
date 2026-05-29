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
		// Handle form submission - redirect to API
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

	// /orgs
	if path == "/orgs" || path == "/orgs/" {
		return data.handleOrgsList(rw, req)
	}

	// /orgs/new
	if path == "/orgs/new" {
		return data.handleOrgNew(rw, req)
	}

	// /orgs/{slug}/*
	parts := strings.Split(strings.TrimPrefix(path, "/orgs/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return data.handleOrgsList(rw, req)
	}

	slug := parts[0]

	// /orgs/{slug}
	if len(parts) == 1 {
		return data.handleOrgView(rw, req, slug)
	}

	// /orgs/{slug}/settings
	if parts[1] == "settings" {
		return data.handleOrgSettings(rw, req, slug)
	}

	// /orgs/{slug}/members
	if parts[1] == "members" {
		return data.handleOrgMembers(rw, req, slug)
	}

	// /orgs/{slug}/tokens
	if parts[1] == "tokens" {
		return data.handleOrgTokens(rw, req, slug)
	}

	// /orgs/{slug}/domains
	if parts[1] == "domains" {
		return data.handleOrgDomains(rw, req, slug)
	}

	return data.handleOrgView(rw, req, slug)
}

// Render functions

func (data *Data) renderOrgsList(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Organizations - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Organizations</h1>
		<p><a href="/orgs/new" class="button">Create Organization</a></p>
		<section>
			<h2>Your Organizations</h2>
			<p>View your organizations via the API: GET /api/v1/orgs</p>
		</section>
		<p><a href="/users">Back to Dashboard</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderOrgNew(rw http.ResponseWriter, req *http.Request, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Create Organization - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Create Organization</h1>
		<form action="/api/v1/orgs" method="POST">
			<div>
				<label for="slug">Slug (URL-friendly name):</label>
				<input type="text" id="slug" name="slug" pattern="[a-z0-9-]+" required>
				<small>Lowercase letters, numbers, and hyphens only</small>
			</div>
			<div>
				<label for="name">Display Name:</label>
				<input type="text" id="name" name="name" required>
			</div>
			<div>
				<label for="description">Description:</label>
				<textarea id="description" name="description"></textarea>
			</div>
			<div>
				<label for="visibility">Visibility:</label>
				<select id="visibility" name="visibility">
					<option value="public">Public</option>
					<option value="private">Private</option>
				</select>
			</div>
			<button type="submit">Create Organization</button>
		</form>
		<p><a href="/orgs">Cancel</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderOrgView(rw http.ResponseWriter, req *http.Request, slug string, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>` + slug + ` - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>` + slug + `</h1>
		<nav>
			<ul>
				<li><a href="/orgs/` + slug + `/settings">Settings</a></li>
				<li><a href="/orgs/` + slug + `/members">Members</a></li>
				<li><a href="/orgs/` + slug + `/tokens">API Tokens</a></li>
				<li><a href="/orgs/` + slug + `/domains">Custom Domains</a></li>
			</ul>
		</nav>
		<p>View organization details via the API: GET /api/v1/orgs/` + slug + `</p>
		<p><a href="/orgs">Back to Organizations</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderOrgSettings(rw http.ResponseWriter, req *http.Request, slug string, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Settings - ` + slug + ` - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Settings: ` + slug + `</h1>
		<form action="/api/v1/orgs/` + slug + `" method="POST">
			<div>
				<label for="name">Display Name:</label>
				<input type="text" id="name" name="name">
			</div>
			<div>
				<label for="description">Description:</label>
				<textarea id="description" name="description"></textarea>
			</div>
			<button type="submit">Save Changes</button>
		</form>
		<p><a href="/orgs/` + slug + `">Back to Organization</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderOrgMembers(rw http.ResponseWriter, req *http.Request, slug string, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Members - ` + slug + ` - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Members: ` + slug + `</h1>
		<section>
			<h2>Add Member</h2>
			<form action="/api/v1/orgs/` + slug + `/members" method="POST">
				<input type="text" name="username" placeholder="Username" required>
				<select name="role">
					<option value="member">Member</option>
					<option value="admin">Admin</option>
				</select>
				<button type="submit">Add Member</button>
			</form>
		</section>
		<section>
			<h2>Current Members</h2>
			<p>View members via the API: GET /api/v1/orgs/` + slug + `/members</p>
		</section>
		<p><a href="/orgs/` + slug + `">Back to Organization</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderOrgTokens(rw http.ResponseWriter, req *http.Request, slug string, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>API Tokens - ` + slug + ` - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>API Tokens: ` + slug + `</h1>
		<section>
			<h2>Create Token</h2>
			<form action="/api/v1/orgs/` + slug + `/tokens" method="POST">
				<input type="text" name="name" placeholder="Token Name" required>
				<select name="scopes">
					<option value="read">Read Only</option>
					<option value="read-write">Read/Write</option>
					<option value="global">Full Access</option>
				</select>
				<button type="submit">Create Token</button>
			</form>
		</section>
		<section>
			<h2>Active Tokens</h2>
			<p>View tokens via the API: GET /api/v1/orgs/` + slug + `/tokens</p>
		</section>
		<p><a href="/orgs/` + slug + `">Back to Organization</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

func (data *Data) renderOrgDomains(rw http.ResponseWriter, req *http.Request, slug string, user *AuthUser) error {
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Custom Domains - ` + slug + ` - ` + data.ServerTitle + `</title>
	<link rel="stylesheet" href="/style.css">
</head>
<body>
	<div class="container">
		<h1>Custom Domains: ` + slug + `</h1>
		<section>
			<h2>Add Domain</h2>
			<form action="/api/v1/orgs/` + slug + `/domains" method="POST">
				<input type="text" name="domain" placeholder="yourdomain.com" required>
				<button type="submit">Add Domain</button>
			</form>
		</section>
		<section>
			<h2>Current Domains</h2>
			<p>View domains via the API: GET /api/v1/orgs/` + slug + `/domains</p>
		</section>
		<p><a href="/orgs/` + slug + `">Back to Organization</a></p>
	</div>
</body>
</html>`

	_, err := rw.Write([]byte(html))
	return err
}

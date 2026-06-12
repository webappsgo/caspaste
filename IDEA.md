## Project description

CasPaste is a self-hosted, privacy-focused pastebin service with URL shortening capabilities. It provides a fast, secure platform for sharing text snippets and code with syntax highlighting, uploading files, and creating short URLs. Designed as a single static binary with all assets embedded and zero external runtime dependencies, it targets developers, teams, and privacy-conscious self-hosters who want to avoid public pastebin services.

## Project variables

project_name:     caspaste
project_org:      casapps
internal_name:    caspaste
app_name:         CasPaste
official_site:    https://pste.us
maintainer_name:  CasjaysDev
maintainer_email: casjay@yahoo.com
binary_name:      caspaste
cli_binary_name:  caspaste-cli
default_port:     80

## Business logic

**Target users:**
- Developers sharing code snippets and files
- Teams needing private paste hosting
- Privacy-conscious users avoiding public pastebin services
- Self-hosters wanting simple, lightweight paste and URL shortening
- CLI users piping content from existing tools (sprunge, ix, termbin, pastebin workflows)

**Features:**
- **Paste creation**: text with optional syntax highlighting (all common languages), optional title, optional author attribution
- **File upload**: any file type, automatic MIME detection, stored in database
- **URL shortener**: create short links that redirect to original URLs
- **Burn-after-reading**: one-use flag — paste deleted immediately after first successful view
- **Private pastes**: excluded from public listing; direct URL still accessible
- **Paste expiration**: configurable (never, 10 min, 1 hour, 1 day, 1 week, 1 month, 1 year)
- **Editable pastes**: update content after creation when created with editable flag
- **Server modes**: public (open creation) or single-password (auth required) — see PART 6
- **Admin panel**: full server management — see PART 17
- **API token management**: admin-generated tokens for programmatic API access
- **Localization**: en, de, bn_IN, ru — see PART 31
- **PWA support**: installable as progressive web app
- **QR code generation**: for paste URLs
- **Embedded view**: paste in iframe for third-party sites
- **External API compatibility**: per-request compat layer — clients for sprunge, ix, termbin, pastebin.com, stikked, microbin, lenpaste, hastebin work without modification by changing only the endpoint URL; mode detected from Host header or env var

**Not in scope (current version):**
- Multi-user accounts, registration, or per-user tokens (optional feature — see PART 34)
- Organizations (optional — see PART 35)
- Custom domain mapping (optional — see PART 36)
- Email notifications (admin email config via PART 18 is in scope; user notifications are not)
- Paid tiers or feature gating
- Telemetry (opt-in only if ever added)

**Data model:**

Paste — primary entity, unified model for text, files, and URLs:

| Field         | Type    | Notes                                                          |
|---------------|---------|----------------------------------------------------------------|
| id            | string  | 8-char cryptographically random identifier                     |
| title         | string  | Optional paste title (max configurable, default 120 chars)    |
| body          | string  | Content: text, file data (encoded), or empty for URL entries  |
| syntax        | string  | Language name for syntax highlighting                          |
| create_time   | int64   | Unix timestamp of creation                                     |
| delete_time   | int64   | Unix timestamp for expiration (0 = never)                      |
| one_use       | bool    | Burn-after-reading — deleted after first view                  |
| author        | string  | Optional author name                                           |
| author_email  | string  | Optional author email (PII — not exposed in public listing)   |
| author_url    | string  | Optional author website                                        |
| is_file       | bool    | True if this is a file upload                                  |
| file_name     | string  | Original filename for file uploads                             |
| mime_type     | string  | MIME type for file uploads                                     |
| is_editable   | bool    | True if paste can be edited after creation                     |
| is_private    | bool    | True if paste is not listed publicly                           |
| is_url        | bool    | True if this is a URL shortener entry                          |
| original_url  | string  | Destination URL for shortener entries                          |

**Sensitivity classification:**
- `body`: user content — may contain secrets, credentials, PII; treat as sensitive in logs
- `author_email`: PII — never expose in public listing responses
- Burn-after-reading pastes: delete must complete before response returns

**Business rules:**
- Paste IDs are 8-char cryptographically random (not sequential)
- Private pastes excluded from list endpoints; direct URL still accessible
- Burn-after-reading delete occurs before the response is written (prevents race)
- Body and title lengths are configurable maximums
- Public server mode: anonymous paste creation allowed
- Single-password server mode: valid session required for all actions
- Compat endpoints are CSRF-exempt (consumed by CLI tools that cannot send tokens)
- Admin credentials stored in `admins` table, not config file
- All compat inputs are validated and rate-limited the same as native inputs
- Rate limiting: configurable windows (5 min / 15 min / 1 hour) on creation endpoints
- Brute-force protection: 5 failed login attempts triggers timed lockout
- URL shortener entries do not have the server fetch the destination (SSRF prevention)
- Files are stored in the database, not on the filesystem (no path traversal risk)
- SSRF via URL shortener: destination URL validated; server never fetches it

**External API compatibility:**

Mode is detected per-request (Host header leftmost label or `CASPASTE_API_MODE` env var). The same server instance can serve multiple compat modes simultaneously on different virtual hostnames.

| Emulated service | Detection                              | What's supported                                                   |
|------------------|----------------------------------------|--------------------------------------------------------------------|
| sprunge.us       | always active                          | Create paste → plain text URL                                      |
| ix.io            | always active                          | Create paste → plain text URL                                      |
| termbin.com      | always active / Host `tb.*`/`nc.*`     | Create paste (raw body) → plain text URL                           |
| lenpaste         | Host `lp.*`/`lenpaste.*`               | Create, get paste, server info                                      |
| stikked          | Host `sk.*`/`stikked.*`/`stikq.*`     | Create, get, recent, trending, language list, paginated listing    |
| microbin         | Host `mb.*`/`microbin.*`               | Create, list, archive, get, edit, delete paste                     |
| hastebin         | Host `haste.*`/`hastebin.*`            | Create paste, get paste                                            |
| pastebin.com     | Host `pb.*`/`pastebin.*`               | Create, delete, list, trends, raw get                              |

Response format exactly matches the target service — no CasPaste envelope on compat routes.

**Endpoint capabilities (WHAT — see PART 14 for route patterns):**

Web (HTML, content-negotiated):
- View/create/edit/delete paste
- List public pastes
- URL shortener redirect
- Download paste as file attachment
- Embedded paste view for iframes
- QR code for paste URL
- Server health check
- About pages (authors, license, source, security policy)
- Documentation pages (API, CLI, libraries, customization)
- Login/logout (shared auth — see PART 17)
- UI settings
- Terms of use
- Sitemap, robots.txt, favicon, PWA manifest and service worker
- Security.txt well-known

API (JSON — see PART 14):
- Health check with paste stats
- Server info
- Create paste
- List pastes
- Get paste by ID
- Admin panel API — see PART 17

**Healthz extension (see PART 13):**
- `features.syntax_highlighting: true`
- `stats.pastes_total`: total paste count
- `stats.pastes_24h`: pastes created in last 24 hours
- `checks.storage`: database read/write probe (ok / degraded / down)

**Trust boundaries:**
- Browser is untrusted; all state-changing requests require CSRF protection
- Compat endpoints are CSRF-exempt (see business rules)
- Trusted proxy headers honored only from IPs in `server.trusted_proxies`
- API clients authenticate with admin token (Bearer) for admin endpoints
- No external runtime service dependencies — all assets embedded at build time

**Threat model:**

| Threat                    | Defense policy                                                          |
|---------------------------|-------------------------------------------------------------------------|
| Anonymous paste spam      | Rate limiting on create endpoints (configurable windows)                |
| Brute-force login         | Timed lockout after failed attempts                                     |
| XSS via paste body        | Server-side syntax highlighting; raw view returns text/plain            |
| Paste ID enumeration      | IDs are cryptographically random; private mode available                |
| Path traversal            | Paste IDs validated; files served from database, not filesystem         |
| CSRF on state changes     | CSRF protection on all non-exempt browser-facing routes                 |
| Private paste exposure    | Private pastes excluded from listing; no ownership leak                 |
| Resource exhaustion       | Configurable body/title length limits; rate limiting on all create paths|
| SSRF via URL shortener    | Destination URL validated; server does not fetch it                     |
| Malicious file upload     | Files stored in database and not executed; MIME from upload header only |

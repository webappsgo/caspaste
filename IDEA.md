# CasPaste

## Project description

CasPaste is a self-hosted, privacy-focused pastebin service with URL shortening capabilities. It provides a fast, secure platform for sharing text snippets and code with syntax highlighting, uploading files, and creating short URLs. Designed as a single static binary with all assets embedded and zero external runtime dependencies, it targets developers, teams, and privacy-conscious self-hosters who want to avoid public pastebin services.

Target users:
- Developers sharing code snippets and files
- Teams needing private paste hosting
- Privacy-conscious users avoiding public pastebin services
- Self-hosters wanting simple, lightweight paste and URL shortening
- CLI users piping content from existing tools (sprunge, ix, termbin, pastebin workflows)

Current codebase state: Go server with embedded HTML templates, CSS/JS, locale files, and Chroma lexers. SQLite default database with optional PostgreSQL and MySQL backends. Supports multi-user mode (optional, off by default), organization support (optional, off by default), and custom domain mapping (optional, off by default). External API compatibility layer allows existing pastebin client tools to work by changing only the endpoint URL.

---

## Project variables

project_name:         caspaste
project_org:          casapps
internal_name:        caspaste        # FROZEN — never edit after first run
internal_org:         casapps         # FROZEN — never edit after first run
app_name:             CasPaste
official_site:        https://caspaste.casapps.io
maintainer_name:      CasjaysDev
maintainer_email:     casjay@yahoo.com
server_tagline:       A simple paste service
server_description:   Self-hosted pastebin for sharing code snippets and text
plist_name:           io.github.casapps.caspaste
binary_name:          caspaste
cli_binary_name:      caspaste-cli
default_port:         80
config_dir:           /etc/casjay-forks/caspaste
data_dir:             /var/lib/casjay-forks/caspaste
log_dir:              /var/log/casjay-forks/caspaste

---

## Business logic

### Product scope & non-goals

**In scope:**
- Paste creation with optional syntax highlighting (Chroma lexers — all languages)
- File upload (any file type, automatic MIME detection, base64-encoded in database)
- URL shortener (create short links redirecting to original URLs)
- Burn-after-reading (one-use): paste deleted immediately after first successful view
- Private pastes: excluded from public listing
- Paste expiration: never (0), 10 min, 1 hour, 1 day, 1 week, 1 month, 1 year
- Editable pastes: update content after creation (when `is_editable` is true)
- Public/private server mode: `server.public=true` means open access, `false` means password auth required
- Multi-user mode (optional): accounts, roles, API tokens, registration modes
- Organization support (optional, PART 35): group ownership of pastes
- Custom domain support (optional, PART 36): per-user/per-org domain mapping
- Localization: en, de, bn_IN, ru
- PWA support: service worker, manifest
- QR code generation for paste URLs
- Embedded pastes (`/emb/{id}`) for iframe inclusion
- External API compatibility: clients that expect sprunge/ix/pastebin/stikked/microbin/lenpaste/hastebin/termbin behavior work unchanged

**Not in scope:**
- Email delivery or notifications
- Chat or real-time collaboration
- General file hosting beyond paste context (no CDN, no directory listing)
- Paid tiers or feature gating
- Telemetry (opt-in only if ever added)

### Roles & permissions

**Single-password mode (server.public=false, multi-user disabled):**
- Anonymous: no access — redirected to login
- Authenticated (server password): full access to create, view, list, manage pastes

**Public mode (server.public=true, multi-user disabled):**
- Anonymous: create public pastes, view public pastes, view raw, download
- No listing of private pastes to anonymous users

**Multi-user mode (users.enabled=true):**
- Anonymous: create public pastes (if server.public=true), view public pastes
- Authenticated user: create pastes (public or private), view own private pastes, edit/delete own pastes, manage API tokens, update profile
- Admin: full access to all pastes, user management, organization management, domain management
- Registration modes: `public` (open), `private` (invite only), `disabled`
- API tokens: users may generate tokens (max 5 per user by default); tokens authenticate API requests

**Paste visibility:**
- Public paste: visible to all (listed publicly)
- Private paste (`is_private=true`): only visible to owner or admin; excluded from `/list`
- Burn-after-reading (`one_use=true`): deleted on first view regardless of auth state

### Data model & sensitivity

**Paste** — primary entity, unified model for text, files, and URLs:

| Field         | Type    | Notes                                          | Sensitivity |
|---------------|---------|------------------------------------------------|-------------|
| id            | string  | 8-char cryptographically random identifier     | Low         |
| title         | string  | Optional paste title (max configurable, default 120 chars) | Low |
| body          | string  | Content: text, base64 file data, or empty for URLs | User content — potentially sensitive |
| syntax        | string  | Language for syntax highlighting               | Low (metadata) |
| create_time   | int64   | Unix timestamp of creation                     | Low         |
| delete_time   | int64   | Unix timestamp for expiration (0 = never)      | Low         |
| one_use       | bool    | Burn-after-reading flag — deleted after first view | High (delete on first read) |
| author        | string  | Optional author name                           | Low         |
| author_email  | string  | Optional author email                          | Medium (PII) |
| author_url    | string  | Optional author website                        | Low         |
| is_file       | bool    | True if this is a file upload                  | Low         |
| file_name     | string  | Original filename for file uploads             | Low         |
| mime_type     | string  | MIME type for file uploads                     | Low         |
| is_editable   | bool    | True if paste can be edited after creation     | Low         |
| is_private    | bool    | True if paste is not listed publicly           | Low (flag)  |
| is_url        | bool    | True if this is a URL shortener entry          | Low         |
| original_url  | string  | Destination URL for shortener entries          | Low         |

**Sensitivity classification:**
- `body`: user content — may contain secrets, credentials, PII; treat as sensitive in logs and error output
- `author_email`: PII — never expose in public listing responses
- `one_use` pastes: high sensitivity — the delete must occur before the response returns (prevents race where two simultaneous reads both succeed)

**Config fields exposed in /server/healthz:**
- `BuildCommit`, `BuildDate`, `Version`: public-safe per PART 13 — intentionally exposed for operational visibility

**Database:**
- SQLite (default), PostgreSQL, MySQL supported
- SQLite backup/cache pool for resilience
- All paste bodies stored as-is (text) or base64-encoded (files)

### Trust boundaries & external services

**Browser ↔ Server:**
- Browser is untrusted; all state-changing requests require CSRF protection
- Exception: compat endpoints are CSRF-exempt (see Security decisions & exceptions)
- Auth state carried in session cookie (single-password mode) or session token (multi-user mode)

**API clients ↔ Server:**
- API clients authenticate via session token or API token (Bearer)
- Unauthenticated API requests permitted only when `server.public=true` or on CSRF-exempt compat endpoints
- Trusted proxy headers (`X-Forwarded-For`, etc.) honored only from IPs in `server.trusted_proxies`

**External API compatibility layer:**
- Inbound: clients that were targeting sprunge/ix/pastebin/stikked/microbin/lenpaste/hastebin/termbin
- These clients are untrusted external sources; all compat inputs are validated and rate-limited
- Response format exactly matches the target service (plain text URL, JSON object, or redirect) — no CasPaste envelope
- The compatibility layer is create-only for most services; hastebin also supports GET /documents/{key}

**Database:**
- Internal trusted component; no network exposure
- No raw SQL from user input — parameterized queries required

**Tor hidden service (optional):**
- Optional `.onion` address configuration
- Treated as an additional listener; same trust model as HTTP clients

**No external service dependencies at runtime:**
- No CDN, no external auth provider, no payment processor, no analytics
- Chroma lexers, locale files, templates, and static assets are all embedded at build time

### Threat model & abuse cases

**Primary assets being protected:**
- Paste content (user data — may contain secrets, credentials, PII)
- Server availability (against resource exhaustion)
- User accounts (in multi-user mode)
- Private pastes (visibility must not leak)

**Trusted vs untrusted inputs:**
- Trusted: database reads, internal config, embedded assets
- Untrusted: all HTTP request bodies, query parameters, headers (except from `trusted_proxies`), form fields, compat endpoint payloads

**Attacker/abuser goals:**
- Read private pastes they don't own
- Enumerate paste IDs to scrape content
- Spam paste creation (fill storage, rate-limit legitimate users)
- Exfiltrate author_email or other PII from listing endpoints
- Inject XSS via paste body rendered in browser
- Brute-force server password (single-password mode)
- Path traversal via filename or ID parameters
- CSRF on state-changing requests (create, delete, edit)
- Resource exhaustion via large bodies or unlimited creation rate

**Abuse cases and required defenses:**

| Threat                    | Defense                                                                 |
|---------------------------|-------------------------------------------------------------------------|
| Anonymous paste spam      | Rate limiting on create endpoints (configurable windows: 5 min/15 min/1 hr) |
| Brute-force login         | BruteForceProtection: 5 failed attempts = 15-minute lockout             |
| XSS via paste body        | Syntax highlighting rendered via Chroma (server-side, HTML-escaped); raw view returns `text/plain` |
| Paste ID enumeration      | IDs are 8-char cryptographically random; private mode available         |
| Path traversal            | Paste IDs validated; file downloads served from database, not filesystem |
| CSRF on state changes     | CSRF protection on all non-exempt browser-facing state-changing routes  |
| Private paste visibility  | `is_private` pastes excluded from list; ownership check on access       |
| Resource exhaustion       | `body_max_len` and `title_max_len` configurable limits; rate limiting on all create paths |
| SSRF via URL shortener    | original_url validated; server does not fetch the destination URL       |
| Malicious file upload     | Files stored as base64 in database, not executed; MIME type from upload header only |

### Security decisions & exceptions

- **Compat endpoints are CSRF-exempt**: `/sprunge`, `/ix`, `/termbin`, `/nc`, `/api/api_post.php`, `/api/create`, `/upload`, `/p`, `/api/v1/new`, `/compat`, `/paste`, `/documents`, `/documents/{key}`. Rationale: these endpoints are consumed by CLI tools and scripts that have no mechanism to obtain or send a CSRF token. All are POST-only (create) or rate-limited GET (hastebin). Accepted risk: a crafted form on a third-party page could trigger a paste creation — this is acceptable because paste creation is low-stakes and rate-limited.

- **Anonymous paste creation is intentional in public server mode**: `server.public=true` deliberately allows unauthenticated paste creation. This is the product's design for open self-hosted instances. Operators who want auth-required mode set `server.public=false`.

- **Burn-after-reading delete occurs before response returns**: the DELETE db call is made before writing the HTTP response body. This is intentional — it prevents a race condition where two near-simultaneous readers both receive the paste before either deletion completes.

- **Hastebin GET /documents/{key} is read-only and rate-limited**: the GET path on the hastebin compat endpoint uses `RateLimitGet`, not `RateLimitNew`. Read-only access to an existing paste by key is accepted as compatible with the hastebin protocol.

- **All compat endpoints return the exact response format of the target service**: sprunge/ix/termbin return plain text URL; pastebin returns plain text key; stikked returns plain text URL; microbin returns 303 redirect (or JSON if `Accept: application/json`); lenpaste returns flat JSON `{id, createTime, deleteTime}` with no envelope; hastebin returns `{"key":"..."}` on create and `{"key":"...","data":"..."}` on get. CasPaste's standard `{ok, data}` envelope is NOT used on compat routes.

- **BuildCommit/BuildDate/Version exposed in /server/healthz**: these fields are public-safe operational metadata. They are intentionally included in the health check response for monitoring and debugging visibility.

- **Trusted proxy headers honored only from configured IPs**: `server.trusted_proxies` controls which upstream IPs may set `X-Forwarded-For` and related headers. Default is empty (no trusted proxies). Operators behind a reverse proxy must explicitly configure this.

---

### App-specific healthz stats, features, and checks

Per PART 13 "Paste service" healthz extension:

```
features.syntax_highlighting: true
stats.pastes_total:            count of all pastes in database
stats.pastes_24h:              count of pastes created in last 24 hours
checks.storage:                database read/write probe (ok / degraded / down)
```

---

### Endpoint inventory

**Web routes (HTML, content-negotiated):**

| Method | Path                            | Description                                  |
|--------|---------------------------------|----------------------------------------------|
| GET    | /                               | New paste form                               |
| POST   | /                               | Create paste (form submission)               |
| GET    | /list                           | List public pastes                           |
| GET    | /settings                       | UI settings page                             |
| GET    | /terms                          | Terms of use                                 |
| GET    | /docs                           | Documentation                                |
| GET    | /docs/apiv1                     | API v1 documentation                         |
| GET    | /docs/libraries                 | Client libraries documentation               |
| GET    | /docs/customize                 | Customization documentation                  |
| GET    | /docs/cli                       | CLI examples documentation                   |
| GET    | /server/healthz                 | Health check (HTML/JSON/text by Accept)      |
| GET    | /healthz                        | Health check alias                           |
| GET    | /server/about                   | About page                                   |
| GET    | /server/about/authors           | Authors page                                 |
| GET    | /server/about/license           | License page                                 |
| GET    | /server/about/source_code       | Source code page                             |
| GET    | /server/about/security          | Security policy page                         |
| GET    | /server/help                    | Help (redirects to /docs)                    |
| GET    | /server/auth/login              | Login page                                   |
| POST   | /server/auth/login              | Login form submit                            |
| GET    | /server/auth/logout             | Logout                                       |
| GET    | /server/auth/register           | Registration page (multi-user mode)          |
| GET    | /users                          | User dashboard (multi-user mode)             |
| GET    | /users/notifications            | User notifications                           |
| GET    | /users/settings                 | User settings                                |
| GET    | /users/settings/privacy         | Privacy settings                             |
| GET    | /users/settings/notifications   | Notification settings                        |
| GET    | /users/settings/appearance      | Appearance settings                          |
| GET    | /users/security                 | Security settings (2FA, password)            |
| GET    | /users/tokens                   | API token management                         |
| GET    | /users/domains                  | Custom domain management                     |
| GET    | /dl/{id}                        | Download paste as attachment                 |
| GET    | /emb/{id}                       | Embedded paste (iframe)                      |
| GET    | /emb_help/{id}                  | Embedded paste help                          |
| GET    | /u/{id}                         | URL shortener redirect                       |
| GET    | /qr/{id}                        | QR code for paste URL                        |
| GET/POST | /edit/{id}                   | Edit paste (if is_editable)                  |
| GET    | /robots.txt                     | Robots file                                  |
| GET    | /sitemap.xml                    | Sitemap                                      |
| GET    | /favicon.ico                    | Favicon                                      |
| GET    | /.well-known/security.txt       | Security contact                             |
| GET    | /.well-known/change-password    | Redirect to password change                  |
| GET    | /manifest.json                  | PWA manifest                                 |
| GET    | /sw.js                          | PWA service worker                           |
| GET    | /{id}                           | View paste by ID                             |

**API routes (JSON):**

| Method | Path                          | Description                                |
|--------|-------------------------------|--------------------------------------------|
| GET    | /api/v1/server/healthz        | Health check (JSON)                        |
| GET    | /api/v1/server/info           | Server info and configuration              |
| POST   | /api/v1/pastes                | Create paste                               |
| GET    | /api/v1/pastes                | List pastes                                |
| GET    | /api/v1/pastes/{id}           | Get paste by ID                            |

**External API compatibility routes (create-only except hastebin GET):**

| Method | Path                      | Emulates          | Response format         |
|--------|---------------------------|-------------------|-------------------------|
| POST   | /sprunge                  | sprunge.us        | Plain text URL          |
| POST   | /ix                       | ix.io             | Plain text URL          |
| POST   | /termbin                  | termbin.com       | Plain text URL          |
| POST   | /nc                       | netcat/termbin    | Plain text URL          |
| POST   | /api/api_post.php         | pastebin.com      | Plain text URL          |
| POST   | /api/create               | stikked/stiqued   | Plain text URL          |
| POST   | /upload                   | microbin          | 303 redirect or JSON    |
| POST   | /p                        | microbin          | 303 redirect or JSON    |
| POST   | /api/v1/new               | lenpaste          | Flat JSON               |
| POST   | /documents                | hastebin          | `{"key":"..."}`         |
| GET    | /documents/{key}          | hastebin          | `{"key":"...","data":"..."}` |
| POST   | /compat                   | generic           | Content-negotiated      |
| POST   | /paste                    | generic           | Content-negotiated      |

**Legacy redirects (301):**

| From                      | To                          |
|---------------------------|-----------------------------|
| /about                    | /server/about               |
| /about/authors            | /server/about/authors       |
| /about/license            | /server/about/license       |
| /about/source_code        | /server/about/source_code   |
| /about/security           | /server/about/security      |
| /login                    | /server/auth/login          |
| /logout                   | /server/auth/logout         |
| /docs/api_libs            | /docs/libraries             |

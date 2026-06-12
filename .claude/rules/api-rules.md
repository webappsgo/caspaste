# API Rules (PART 13, 14, 15) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 13, PART 14, PART 15

## CRITICAL — NEVER DO

- Create unversioned API routes (always `/api/{api_version}/...`)
- Use singular resource names — always plural (`/users` not `/user`)
- Use verbs in routes — nouns only, method carries the action
- Use uppercase or underscores in routes — lowercase with hyphens
- Add trailing slashes to API routes
- Keep legacy endpoints — DELETE old routes; no shims, no backwards compat
- Run core functionality in client-side JavaScript (SPA / React / Vue)
- Use the same handler for old and new routes in parallel — migrate fully then delete
- Hardcode `v1` — always use `APIBasePath()` or `{api_version}` variable
- Serve user-controlled content inline as HTML — server-side rendering or attachment only
- Expose `/metrics` endpoint to public traffic — internal only

## CRITICAL — ALWAYS DO

- Version ALL API routes: `/api/{api_version}/...`
- Mount admin API at `/api/{api_version}/server/{admin_path}/...`
- Mount health endpoint at `/server/healthz` (frontend) and `/api/{api_version}/server/healthz` (API)
- Frontend must work without JavaScript for core functionality
- Server renders HTML; client-side JS is enhancement only
- All projects MUST have built-in Let's Encrypt support
- All projects MUST expose Prometheus-compatible `/metrics` (internal only)
- Content-negotiate: browser → HTML, curl/CLI → plain text, API clients → JSON
- Route migration: move handlers fully, delete old routes, no parallel trees

## Route Scopes

| Scope | Web Route | API Route |
|-------|-----------|-----------|
| Server public | `/server/*` | `/api/{api_version}/server/*` |
| Auth | `/server/auth/*` | `/api/{api_version}/server/auth/*` |
| Users (self) | `/users/*` | `/api/{api_version}/users/*` |
| Orgs | `/orgs/{slug}/*` | `/api/{api_version}/orgs/{slug}/*` |
| Admin | `/server/{admin_path}/*` | `/api/{api_version}/server/{admin_path}/*` |
| Project | `/*` | `/api/{api_version}/*` |

## Route Rules

| Rule | Correct | Wrong |
|------|---------|-------|
| Versioned | `/api/v1/users` | `/api/users` |
| Plural nouns | `/users` | `/user` |
| Lowercase hyphens | `/api-keys` | `/API_Keys` |
| No trailing slash | `/users` | `/users/` |
| No verbs | `GET /users` | `GET /getUsers` |

## Health Check Endpoints

- `/server/healthz` — frontend (HTML/text, content negotiated)
- `/api/{api_version}/server/healthz` — API (JSON default)
- Optional `/healthz` root alias when `server.healthz.root.enabled: true`
- NO sub-routes (no `/server/healthz/db`)

## Health Response Fields (required, in order)

`project` → `status` → `version`/`go_version`/`build` → `uptime`/`mode`/`timestamp` → `cluster` → `features` → `checks` → `stats`

All health fields MUST be public-safe (Tier 2 — see backend-rules.md).

## SSL/TLS (PART 15)

- Built-in Let's Encrypt: HTTP-01, TLS-ALPN-01, DNS-01 (all providers via lego)
- DNS-01 credentials: AES-256-GCM encrypted, stored in config
- FQDN resolution order: `X-Forwarded-Host` → `DOMAIN` env → `os.Hostname()` → `$HOSTNAME` → public IPv6 → public IPv4 → `localhost`
- `DOMAIN` env: comma-separated list, first is primary
- Never set `DOMAIN` to overlay addresses (`.onion`, `.i2p`) — app manages those

## Metrics (PART 21)

- Format: Prometheus text exposition
- Endpoint: `/metrics` (configurable)
- INTERNAL ONLY — never proxy to public
- Auth: optional bearer token (`Authorization: Bearer <token>`)
- Naming: `{project_name}_` prefix, snake_case, `_total` suffix for counters, base units (seconds, bytes)
- Cardinality: normalize path IDs with `:id`, never use `user_id`/`request_id` as labels

## Client-Side JavaScript Rules

DO use JS for: form validation feedback, theme toggle, copy-to-clipboard, polling/refresh, modals, keyboard shortcuts.

NEVER use JS for: routing (SPA), initial render, data fetching for page load, business logic, core features.

For complete details, see AI.md PART 13, PART 14, PART 15

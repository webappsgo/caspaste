# Backend Rules (PART 9, 10, 11, 32) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 9, PART 10, PART 11, PART 32

## CRITICAL — NEVER DO

- Use bcrypt — Argon2id for passwords, SHA-256 for tokens
- Use `fmt.Sprintf` to build SQL — parameterized queries ONLY
- Cast user-controlled content to `template.HTML`
- Expose stack traces in production HTTP responses
- Expose DB credentials, internal IPs, or raw tokens in any response
- Use `DROP COLUMN`, `DROP TABLE`, or `DELETE` in schema updates
- Rename columns — add new, migrate in app code, keep old
- Use migration files or version tracking — idempotent `CREATE TABLE IF NOT EXISTS` only
- Harden CGI-style: NEVER shell out with raw user content, filenames, or metadata
- Use default Tor ports (9050, 9051) — use `127.0.0.1:auto` for control port
- Embed Tor binary — use external binary via `github.com/cretz/bine`
- Disable or opt out of Tor if the Tor binary is present (auto-enabled always)

## CRITICAL — ALWAYS DO

- All errors use canonical `{"ok": false, "error": "CODE", "message": "..."}` format
- Log all errors with context (`error_code`, `request_id`, `http_status`)
- Never show specific error reason in production (no "user not found" vs "wrong password")
- Use `subtle.ConstantTimeCompare` for all secret/token comparisons
- Apply defense-in-depth: validate input → parameterized queries → escape output → TLS
- All schema changes idempotent — `CREATE TABLE IF NOT EXISTS`, ignore "column exists"
- New columns must have `DEFAULT` or be nullable
- Auto-enable cluster mode when external cache + shared DB detected
- Sanitize all public responses (strip internal IPs, internal paths, known-sensitive params)
- `markdownToHTML` MUST disable raw HTML input AND sanitize final HTML
- Tor: server binary owns the Tor process lifecycle (start, stop, manage)

## Error Response Format

Success: `{"ok": true, "data": {...}}`
Error: `{"ok": false, "error": "ERROR_CODE", "message": "Human message"}`
Debug only: add `"_debug": {...}` — stripped in production by middleware

## Error Code → HTTP Status

| Code | Status |
|------|--------|
| `BAD_REQUEST`, `VALIDATION_FAILED` | 400 |
| `UNAUTHORIZED`, `TOKEN_EXPIRED`, `TOKEN_INVALID` | 401 |
| `FORBIDDEN`, `ACCOUNT_LOCKED` | 403 |
| `NOT_FOUND` | 404 |
| `CONFLICT` | 409 |
| `RATE_LIMITED` | 429 |
| `SERVER_ERROR` | 500 |
| `MAINTENANCE` | 503 |

## Security Tiers (PART 11)

| Tier | Items | Rule |
|------|-------|------|
| 1 — NEVER public | DB creds, tokens, other users' PII, internal IPs, account-existence signals | Never shown, not even in debug |
| 2 — Always public | `version`, `commit`, `build_date`, `mode`, `uptime`, `db_type` | Always shown, unauthenticated |
| 3 — Debug-only | Stack traces, SQL queries, rate-limit thresholds, full error chains | Gated behind `--debug` / `DEBUG=true` |

## Output Sanitization Pipeline (every public response)

1. Allow-list fields only (no accidental sensitive field leakage)
2. Redact known-sensitive query params (`token`, `password`, `key`, `secret`, etc.)
3. Strip internal IPs and filesystem paths (replace with `[redacted]`)
4. Truncate (strings 256 chars, messages 200 chars, stacks 2KB)
5. Strip `dev_only:"true"` fields in production
6. Constant-time finalize on auth-sensitive paths (pad to 100ms min)

## Database Rules (PART 10)

- `CREATE TABLE IF NOT EXISTS` — always, on every startup
- Schema updates: `ALTER TABLE ADD COLUMN IF NOT EXISTS` (ignore "already exists" error)
- Never destructive: add-only schema changes
- Cluster: auto-detected when PostgreSQL/MySQL + Valkey/Redis present
- Cluster heartbeat: 30s interval, 90s = degraded, 5min = offline

## Tor Hidden Service Rules (PART 32)

| Rule | Value |
|------|-------|
| Auto-enabled | If Tor binary found — no toggle |
| Control port | `127.0.0.1:auto` (let Tor choose) |
| Hidden service | v3 onion (ed25519, 56 chars) |
| Virtual port | 80 → `localhost:{server_port}` |
| Safe logging | Always enabled |
| Library | `github.com/cretz/bine` (no CGO) |
| Isolation | Completely separate from system Tor |

## User Content Safety

| Content | Rule |
|---------|------|
| Plain text / source | Render as escaped text in `<pre><code>` |
| Markdown | Disable raw HTML, sanitize rendered output |
| User HTML | Never render inline — escape or attachment only |
| SVG / XML | Never inline — force attachment or raster |
| Active MIME types (`text/html`, `image/svg+xml`) | Force `Content-Disposition: attachment` |

For complete details, see AI.md PART 9, PART 10, PART 11, PART 32

# Config Rules (PART 5, 6, 12) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 5, PART 6, PART 12

## CRITICAL — NEVER DO

- Put inline YAML comments (never `key: value  # comment` — always comment above)
- Allow `..` or encoded traversal (`%2e%2e`) in any path — block at request level
- Use uppercase path segments — only lowercase alphanumeric, hyphens, underscores
- Fail startup on invalid config — always warn and replace with defaults
- Expose debug endpoints in production (no `--debug` / `DEBUG=true` set)
- Expose stack traces in HTTP responses in production mode
- Show internal IPs, DB credentials, or raw tokens in any public response
- Hardcode mode defaults — detect from `--mode` flag → `MODE` env → default `production`
- Use bcrypt — Argon2id for passwords, SHA-256 for tokens (see PART 11)

## CRITICAL — ALWAYS DO

- Put YAML comments on their own line ABOVE the setting
- Validate and normalize all paths with `SafePath()` before use
- On invalid config value: log warning, replace with default, continue
- Detect mode: `--mode` flag (1) → `MODE` env (2) → `production` (3)
- Detect debug: `--debug` flag (1) → `DEBUG` env truthy (2) → `false` (3)
- Apply `PathSecurityMiddleware` FIRST in the middleware chain (before auth)
- Gate all debug endpoints, stack traces, and sensitive diagnostics behind `--debug`
- Run `PathSecurityMiddleware` before routing — block `..`, `%2e`, encoded traversal
- Respect `X-Forwarded-Prefix` / `X-Forwarded-Path` / `X-Script-Name` for baseurl

## Path Security

| Input | Result | Valid |
|-------|--------|-------|
| `/myadmin/` | `myadmin` | ✓ |
| `/../admin` | rejected | path traversal |
| `/Admin` | rejected | uppercase |
| `/server/admin/<script>` | rejected | invalid chars |

Valid path segment: `^[a-z0-9_-]+$`, max 64 chars, no `..`

## Application Modes

| State | Mode | Debug |
|-------|------|-------|
| Production (default) | `production` | `false` |
| Development | `development` | `false` |
| Debug (any mode) | any | `true` |

| Mode shortcuts | `--mode dev` → development · `--mode prod` → production |
|---|---|

## Debug Flag Effects

- Admin authentication: **BYPASSED** (dev only — NOT for automated tests)
- `/debug/*` endpoints: **ENABLED** (pprof, expvar, config, routes, cache, db)
- Full request/response logging, DB query logging, cache logging
- Stack traces in error responses (via `_debug` field)

## Server Config Principles (PART 12)

| Setting | Default | Notes |
|---------|---------|-------|
| `baseurl` | `/` | Auto-detect from `X-Forwarded-Prefix` |
| `max_body_size` | 10MB | |
| `read_timeout` | 30s | |
| `write_timeout` | 30s | |
| `idle_timeout` | 120s | |

- Trusted proxies: loopback + RFC 1918 always trusted; public proxies via `trusted_proxies.additional`
- Session cookie names: `admin_session` (admin), `user_session` (user)
- Rate limiting: per-IP sliding window, stored in `server.db`

For complete details, see AI.md PART 5, PART 6, PART 12

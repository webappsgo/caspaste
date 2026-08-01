# Configuration Rules (PART 5, 6, 12)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never use inline YAML comments (`port: 8080  # comment`) — comments go ABOVE the setting
- Never use `strconv.ParseBool()` — always `config.ParseBool()` / `config.IsTruthy()`
- Never fail startup on invalid config — warn and replace with default
- Never let debug flag bypass admin authentication or any security check, in any mode
- Never trust `X-Forwarded-*` headers from a peer not in `trusted_proxies`
- Never leak clearnet FQDN/email in a Tor (`.onion`) response — omit if `tor.contact_email` unset
- Never re-resolve/trust proxy headers after `r.RemoteAddr` has been rewritten by real-IP middleware — check the preserved original TCP peer
- Never expose `server.contact.admin.email` or any `webhooks.*` URL publicly
- Never auto-populate `abuse@{fqdn}` — abuse email requires explicit opt-in (unlike `security@{fqdn}`)
- Never accept an unvalidated path — always run paths through `SafePath`/`normalizePath`/`validatePath`
- Never skip Valkey/Redis in cluster or mixed-mode deployments — `memory` cache only valid for single instance

## CRITICAL - ALWAYS DO
- Store config as `server.yml` (auto-migrate legacy `server.yaml` on startup)
- Root installs → `/etc/{internal_org}/{internal_name}/server.yml`; regular user → `~/.config/{internal_org}/{internal_name}/server.yml`
- Validate all config values on load; invalid → log warning, substitute default, keep starting
- Accept ALL truthy/falsy synonyms case-insensitively (yes/no, on/off, enable/disable, oui/non, etc.) — empty/unset uses default, invalid value errors
- Run `PathSecurityMiddleware` third in the chain (after URL normalize + request ID, before security headers/allowlist/blocklist/rate-limit/geoip/auth/logging)
- Persist the selected port (random or specified) to `server.yml` after first run
- Bind privileged ports (<1024) while still root, then drop privileges to the `{internal_name}` system user (Unix); Windows uses a Virtual Service Account, never drops
- Sync every database config change to local `server.yml` (cache) immediately, on startup, and every 5 minutes
- Enter maintenance mode (read-only, 503 on writes) only for DB-connection or file-write critical errors; retry every 30s with self-healing
- Resolve `{fqdn}`/`{proto}`/`{port}`/`{baseurl}` via reverse-proxy headers first, config/CLI next, then hostname/default fallback
- Detect Tor via `Host` == `tor.onion_address` as priority-0 check, before any proxy-header/trust-IP logic

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|-----------------|
| Config file extension | `.yml`, never `.yaml` | PART 5 → Configuration File |
| Default port | Random unused port 64000-64999, saved on first run | PART 5 → Port Rules |
| Port <1024 as non-root | Fallback to random 64xxx, or prompt to escalate | PART 5 → Privileged Port Binding |
| Mode priority | `--mode` flag > `MODE` env > default `production` | PART 6 → Mode Detection Priority |
| Debug priority | `--debug` flag > `DEBUG` env > `--mode debug` alias > default `false` | PART 6 → Mode Detection Priority |
| `MODE=debug` | Alias for `development` + debug on; explicit `DEBUG` still overrides | PART 6 → Mode Shortcuts |
| Debug endpoints (`/debug/*`) | 404 unless `--debug`/`DEBUG=true`, regardless of mode | PART 6 → Debug Endpoints |
| Config source of truth | Single instance → `server.yml`; Cluster → database (server.yml is cache/backup) | PART 5 → Configuration Source of Truth |
| Cache type default | `memory`; `valkey`/`redis` required for cluster/mixed mode | PART 12 → Cache Configuration |
| Base URL default | `/`, auto-detected from `X-Forwarded-Prefix`/`-Path`/`X-Script-Name`, else config, else `/` | PART 12 → Base URL |
| `security@{fqdn}` auto-default | Yes (RFC 2142); falls back to `admin.email` if explicitly emptied | PART 12 → Contact Configuration |
| `abuse@{fqdn}` auto-default | No — must opt in; falls back general → admin | PART 12 → Contact Configuration |
| Rate limit defaults | Read 120/min, Write 10/min, Health 120/min, global burst 240/min | PART 12 → Rate Limiting |
| Session defaults | Admin: 30d max_age/24h idle; User: 7d max_age/24h idle; `same_site: strict` | PART 12 → Session Configuration |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| `server.yml` | YAML configuration file on disk |
| Configuration | Settings stored in `server.yml` OR database, depending on mode |
| Server Address | Bind address (e.g. `[::]`, `0.0.0.0`) — NOT "Server Name" |
| FQDN | Fully qualified domain name clients use to reach the server |
| Node ID | Unique cluster node identifier (default: hostname) |
| Init-only variable | Env var used once on first run, then ignored (`CONFIG_DIR`, `PORT`, etc.) |
| Runtime variable | Env var checked every run (`MODE`, `DEBUG`, `DOMAIN`, `DATABASE_*`, `SMTP_*`) |
| Maintenance mode | Read-only state entered on critical (DB/file) error, with self-healing |
| `{internal_name}` user | Dedicated system service account created for privilege drop |

## QUICK REFERENCE

**Four operational states (mode × debug):**
| State | Mode | Debug |
|-------|------|-------|
| Production | production | false |
| Production + Debug | production | true |
| Development | development | false |
| Development + Debug | development | true |

**Env vars (runtime, always checked):** `NO_COLOR`, `TERM`, `DOMAIN`, `MODE`, `DATABASE_DRIVER`, `DATABASE_URL`, `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM_NAME/FROM_EMAIL/TLS`

**Env vars (init-only, first run only):** `CONFIG_DIR`, `DATA_DIR`, `LOG_DIR`, `DATABASE_DIR`, `BACKUP_DIR`, `PORT`, `LISTEN`, `APPLICATION_NAME`, `APPLICATION_TAGLINE`

**Middleware order (execution 1→10):** URLNormalize → RequestID → PathSecurity → SecurityHeaders → Allowlist → Blocklist → RateLimit → GeoIP → Auth → Logging

**Port special cases:** `80` → Let's Encrypt HTTP-01; `443` → TLS-ALPN-01 + auto SSL; `0` → OS-assigned; dual port `"8090,8443"` = HTTP,HTTPS

**Trusted proxy always-allowed ranges:** loopback, RFC 1918 (`10/8`, `172.16/12`, `192.168/16`), RFC 4193 (`fc00::/7`), link-local; add public proxy IPs/CIDRs/DNS via `trusted_proxies.additional`

**Contact roles:** `admin` (never public, required), `security` (public, defaults `security@{fqdn}`), `abuse` (public, opt-in, no default), `general` (public, falls back to admin) — each supports email + `webhooks.{telegram,discord,slack,generic,...}`

**Config validation:** invalid value → log warning → substitute default → never crash startup

---
For complete details, see AI.md PART 5, 6, 12

# API Rules (PART 13, 14, 15)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never add sub-routes under `/server/healthz` (no `/server/healthz/db`, etc.)
- Never expose sensitive data in health responses: credentials, DB connection strings, internal IPs, file paths, usernames, emails, secrets, stack traces, `/metrics` status
- Never keep legacy (removed/changed) endpoints "for compatibility" - delete them immediately, no redirects, no deprecation period
- Never layer new routes on top of old code - migrate the implementation, don't duplicate handler trees
- Never use singular resource names (`/api/{api_version}/user`) - always plural
- Never use uppercase, underscores, or trailing slashes in routes
- Never use verbs in routes (`getUsers`) - use HTTP method + noun
- Never hardcode `v1` - always use `APIBasePath()` / `{api_version}`
- Never manually edit generated `openapi.json` or GraphQL schema files
- Never put swagger/graphql files in project root - only `src/swagger/`, `src/graphql/`
- Never implement full external API surface (list/view/delete) unless route/API compatibility was explicitly requested - default is feature compatibility only
- Never redirect unversioned `/api/<thing>` aliases (`/api/swagger`, `/api/graphql`, `/api/healthz`) to the versioned URL - mount the same handler directly
- Never put icons/ASCII art/colors in log output - logs are always raw text
- Never auto-renew certs found under `/etc/letsencrypt/live/**` - that's system/certbot-owned
- Never request Let's Encrypt certs for `.onion`/`.i2p` - use self-signed there

## CRITICAL - ALWAYS DO
- Always version API routes: `/api/{api_version}/...`
- Always keep Swagger and GraphQL in sync with each other and with the actual API (build-time generated, never manual)
- Always implement all three API types: REST, Swagger, GraphQL
- Always make health data public-safe - only vague statuses (`"ok"`/`"error"`), never details
- Always follow canonical error shape: `{ok:false, error:CODE, message, details?}` - HTTP status carries the status code, never duplicate it in the body
- Always end every file/response (JSON, text, HTML, YAML, Go, CSS, JS) with exactly one trailing newline
- Always use 2-space indent for HTML/JSON/YAML/CSS/JS; tabs for Go/Makefiles
- Always prefer path params for resource identity, query params for filters/sort/paginate
- Always support Let's Encrypt via HTTP-01, TLS-ALPN-01, and DNS-01 challenge types
- Always resolve `{fqdn}` via the fixed priority order (reverse-proxy headers > `DOMAIN` env > hostname > `$HOSTNAME` > public IPv6 > public IPv4 > `localhost`)
- Always strip `:80` and `:443` from displayed URLs

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| Health check route(s)? | `/server/healthz` (frontend), optional `/healthz` root alias (gated by `server.healthz.root.enabled`), `/api/{api_version}/server/healthz`, `/api/healthz` (unversioned alias) | PART 13 |
| Health default format? | HTML for browsers, JSON for `/api/**`, text for CLI/`.txt`/`Accept: text/plain` | PART 13, 14 |
| SemVer start version? | `1.0.0` (never `0.x.x`) | PART 13 |
| Version string has `v` prefix? | No in the string; git tags do get `v` prefix (`v1.0.0`) | PART 13 |
| Route casing/pluralization? | lowercase, plural nouns, hyphens for multi-word | PART 14 |
| OpenAPI format? | JSON only, no YAML, no `.json` suffix on path | PART 14 |
| Pagination default limit? | 250 items | PART 14 |
| Is compatibility route-for-route by default? | No - feature/behavior compatibility only, unless user explicitly asks for route/API/client compatibility | PART 14 |
| RFC-defined protocol app (DNS/SMTP/HTTP/FTP/etc.)? | Full RFC compliance is mandatory, not optional | PART 14 |
| Client-side JS scope? | Enhancement only (theme toggle, copy button, validation feedback) - never core functionality, never SPA routing/rendering | PART 14 |
| Single port default protocol? | HTTP, except port 443 forces HTTPS-only | PART 15 |
| Overlay network (Tor/I2P) protocol? | HTTP by default; matches clearnet's HTTPS-only mode only when clearnet is on 443 | PART 15 |
| Cert storage app-managed path? | `{config_dir}/ssl/letsencrypt/{fqdn}/` (auto-renews 7 days before expiry) vs `{config_dir}/ssl/local/{fqdn}/` (manual, never auto-renewed) | PART 15 |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| Compatibility (default) | Matching a target service's features/behavior using our own routes |
| Route/API/client compatibility | Explicitly requested - also match the external URL paths/methods/params/response format |
| Legacy endpoint | Old/changed/removed route from our own project - delete, never keep |
| Compatibility endpoint | Route matching an external service (pastebin.com, microbin, etc.) - implement per compatibility rules |
| Our Client | `{project_name}-cli` binary - interactive, receives JSON, renders own TUI/GUI |
| Text Browser | lynx/w3m/links/elinks/etc. - interactive, no JS, receives no-JS HTML |
| HTTP Tool | curl/wget/httpie/etc. - non-interactive, receives `HTML2TextConverter` formatted text |
| Non-interactive client | Only HTTP tools (per `isNonInteractiveClient()`) |

## QUICK REFERENCE
**Health endpoint matrix**
| Route | Format |
|-------|--------|
| `/server/healthz` | HTML (browser) / text (CLI) / JSON (`Accept: application/json`) |
| `/healthz` (optional) | Same handler as `/server/healthz`, no redirect |
| `/api/{api_version}/server/healthz` | JSON default, text via `.txt`/Accept/non-interactive |
| `/api/healthz` | Unversioned direct alias, same handler, JSON default |

**Content negotiation priority (`/api/**`)**: `.txt` ext > `Accept: application/json` > `Accept: text/plain` > non-interactive client (curl/wget/empty UA) > default JSON

**Content negotiation priority (frontend `/**`)**: `Accept: text/html` > `Accept: text/plain` > browser UA (HTML) > CLI/curl (text) > default HTML

**Route scopes**: `/server/*` (public/no ID) · `/server/auth/*` · `/users/*` (current user, no ID) · `/orgs/{slug}/*` (ID required) · `/server/{admin_path}/*` (admin) · `/server/{admin_path}/config/*` (server-wide admin config) · `/*` (project-specific)

**SemVer**: MAJOR = breaking change, MINOR = new backward-compatible feature, PATCH = bug fix; pre-release suffix `-rc1`/`-alpha`; beta format `YYYYMMDDHHMMSS-beta`; daily format `YYYYMMDDHHMMSS`

**Root-level fixed endpoints**: `/`, `/server/healthz`, `/healthz` (optional), `/server/docs/swagger`, `/server/docs/graphql`, `/metrics`, `/server/{admin_path}`, `/api/autodiscover`, `/api/swagger`, `/api/graphql`, `/api/healthz`, `/api/{api_version}/server/swagger`, `/api/{api_version}/server/graphql`, `/api/{api_version}/server/healthz`, `/api/{api_version}/server/{admin_path}/*`

**Cert lookup order**: `/etc/letsencrypt/live/domain/` > `/etc/letsencrypt/live/{fqdn}/` > `{config_dir}/ssl/letsencrypt/{fqdn}/` (app-managed) > `{config_dir}/ssl/local/{fqdn}/` (user-managed)

**FQDN resolution order**: reverse-proxy headers > `DOMAIN` env > `os.Hostname()` > `$HOSTNAME` > public IPv6 > public IPv4 > `localhost`

---
For complete details, see AI.md PART 13, 14, 15

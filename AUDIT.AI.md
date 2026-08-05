# Project Audit

Started: 2026-07-31
Spec: AI.md (no SPEC.md — AI.md is sole source of truth). IDEA.md marks
multi-user / organizations / custom domains as NON-NEGOTIABLE for caspaste.

Legend: [ ] open · [x] fixed this pass · (LOG) tracked for a future
scoped effort (feature-sized or needs a human decision — not auto-fixable
in an audit pass).

Build/vet gate: `go build ./...` and `go vet ./...` pass clean (Docker,
CGO_ENABLED=0) as of session start.

---

## Pass 1: Security

- [x] logger: access.log/error.log wrote raw `?RawQuery` verbatim (setup
  token, password-reset tokens leaked in plaintext) — AI.md PART 11 stage 2.
  Fixed: added `redactRequestPath` redacting token/session/code/key/password/
  secret/auth/pwd/api_key/apikey/access_token/refresh_token (case-insensitive)
  in src/logger/logger.go HttpRequest + HttpError.
- [x] updater: src/updater/update.go:~241 SHA256 checksum verification was
  conditional (`if checksumURL != ""`) — a release with no .sha256 asset
  installed unverified. PART 23 requires MANDATORY verification. Fixed:
  DoUpdate now refuses the update when no checksum asset is found.
- [x] caspasswd: src/caspasswd/caspasswd.go:85-89 Data.Check returned false
  immediately for unknown user — skipped Argon2id → username-enumeration
  timing oracle. Fixed: runs an equal-cost dummy Argon2id verify on the
  unknown-user path before rejecting.
- [x] web/auth: src/web/auth.go:106 session-cookie HMAC signature was
  compared with plain `!=`, not constant-time. Fixed: hmac.Equal.
- [x] admin/setup: src/admin/setup.go:32 `_, _ = rand.Read(raw)` discarded
  its error (zero-token on failure); setup.go:64/79 compared tokens with
  ==/!= not constant-time. Fixed: error checked, subtle.ConstantTimeCompare
  used for both comparisons.
- [x] web/middleware: HSTS was gated on `r.TLS != nil` only (middleware.go:56)
  — never sent behind a reverse proxy (primary topology). Fixed: also honors
  X-Forwarded-Proto=https.
- [x] web cookies: session (auth.go:138) and CSRF (csrf.go:230) cookies used
  SameSite=Lax; AI.md:15185/15264 specify SameSite=Strict as default. Fixed:
  both set to Strict.
- [x] web/middleware: security-header set missing several MUST headers
  (AI.md:15321-15352): Origin-Agent-Cluster,
  X-Permitted-Cross-Domain-Policies, Cross-Origin-Opener/Embedder/Resource-
  Policy, Reporting-Endpoints/Report-To/NEL, and Clear-Site-Data on
  logout/session-revoke. Fixed: added all headers to SecurityHeadersMiddleware
  (config-driven, spec defaults), Clear-Site-Data set on web+admin logout,
  HSTS default bumped to 2yr+preload.
- [x] web/middleware:66-83 CORSMiddleware hardcodes
  `Access-Control-Allow-Origin: *`. Not credential-aware/auto-detected per
  PART 16 CORS model. Fixed: CORSMiddleware now takes CORSConfig; resolves
  explicit config origins + DOMAIN env entries, echoes the specific matched
  origin with Allow-Credentials:true, falls back to "*" (no creds) only when
  the list is empty; full spec auth-header list; Vary: Origin.
- [x] backup: extraction has Zip-Slip guard (good) but no max-uncompressed-
  size / file-count / compression-bomb limit (AI.md:15228). Admin-only. LOW.
  Fixed: extractArchive enforces max file count, max single-file size,
  max total uncompressed size, compression-ratio ceiling (bomb detection),
  LimitReader-bounded copy, and rejects absolute/.. names + symlink/special
  entry types.

## Pass 2: Code Quality

- [ ] admin/api.go: THREE production handlers return HTTP 501 "not yet
  implemented": apiPatchSettings (:74), apiCreateBackup (:236), apiEmailTest
  (:257). All three are spec-mandated (PART 17: PATCH /config/settings :31487,
  POST /config/backup :31605, POST /config/email/test :31582). (LOG — needs
  real backend wiring; overlaps email/backup subsystems below.)
- [x] backup/verify.go:155-162 ChecksumValid hardcoded `true` — FIXED: manifest
  now stores a deterministic combined SHA-256 over staged content
  (computeContentChecksum); verify recomputes it over the extracted files
  (skipping manifest.json) and fails on mismatch. Removed dead calculateChecksum;
  added verify_test.go round-trip + determinism tests.
- [ ] email: src/email/email.go Send()/NewClient() are never called anywhere
  (dead code) — no welcome/verification/reset/invite email wired to any event.
  (LOG — subsystem integration.)
- [ ] token/token.go user+org token CRUD (CreateUserToken/CreateOrgToken/
  ListOrgTokens/…) never called outside package — dead until multi-user/orgs
  are wired (PART 34/35). (LOG.)

## Pass 3: Logic and Correctness

- [x] storage timeouts deviated from PART 10 (AI.md:14857): writes used 5s
  (need 10s), bulk delete 30s (need 60s), DDL/migrations 30s (need 5min);
  conn max_lifetime 1h/idle 10min (storage.go:60-62) vs spec 5m/1m. NewPool
  never PingContext-verified the connection. Fixed: timeouts realigned,
  PingContext check added to NewPool.
- [x] server: random-port range was 64000-65535 (caspaste.go:~2665);
  AI.md:18150 specifies 64000-64999. Fixed.
- [x] privilege/privilege_linux.go findAvailableUID — FIXED: now scans 899→200
  descending, skips reservedIDs map (65534, 999-980, 170-179, 101-110), and
  requires both UID and GID free (user.LookupId + LookupGroupId + /etc/passwd
  fallback), per PART 24 (AI.md:37358-37408).
- [x] updater/update_unix.go — FIXED: ReplaceBinary now detects EXDEV from the
  atomic rename and falls back to replaceAcrossFilesystems, which stages a copy
  in the target directory (same filesystem) and atomically renames it onto the
  running binary, preserving permissions.

## Pass 4: Documentation Completeness

- [x] docs/: security.md, integrations.md, stylesheets/dark.css, light.css all
  present with real content (resolved in commit 5ff3569).
- [x] .readthedocs.yml should be .readthedocs.yaml (AI.md:45579); pins
  ubuntu-22.04/py3.11, template wants ubuntu-24.04/py3.12. Fixed: renamed,
  bumped.
- [x] docs/requirements.txt missing mkdocs-minify-plugin>=0.7.0 and
  pymdown-extensions>=10.0 (AI.md:45761). Fixed: both added.
- [x] mkdocs.yml: extra_css (dark.css/light.css), full markdown_extensions,
  nav with Security+Integrations, extra:/social all present (commit 5ff3569).
- [x] README.md section order now matches PART 1 (About → Official Site →
  Features → Production → Client → Configuration → API → Other → Development →
  Disclaimer → License), verified in commit 5ff3569.
- [x] .claude/rules/*.md — all 14 cheatsheet files present.
- [x] LICENSE.md already reads "Copyright (c) 2026 webappsgo", matching the
  repo's first-commit year (2026-02-01). No human decision needed.

## Pass 5: Spec and Rules Compliance

### Major NOT-IMPLEMENTED feature layers (LOG — feature-sized; needs a
### human scope decision. These are core business logic, so per the audit
### "red flags — stop and ask" rule they are recorded, not auto-built.)

- [ ] (LOG) PART 34 MULTI-USER — schema tables exist but NOTHING wired: no
  registration/login against users table (login is single-admin file-backed),
  no /users/* or /{username}/ vanity routing, no ValidateUsername/username
  blocklist (reserved-name defense ABSENT — security-critical), no profile
  privacy filtering. pastes table has NO user_id/org_id/owner column, so
  tenant scoping is structurally impossible. NON-NEGOTIABLE per IDEA.md.
- [ ] (LOG) PART 35 ORGANIZATIONS — schema-only. No org routes, no role
  permission checks, no /{orgname}/ vanity routing, no reserved-slug allowlist
  with case-insensitive cross-table (user+org) collision checks
  (security-critical). NON-NEGOTIABLE per IDEA.md.
- [ ] (LOG) PART 36 CUSTOM DOMAINS — schema-only. ZERO DNS code (no LookupTXT)
  — no ownership verification before cert issuance (security-critical), no
  per-domain ACME. NON-NEGOTIABLE per IDEA.md.
- [ ] (LOG) PART 10 CLUSTER MODE — "ALL apps MUST support cluster mode": no
  nodes/cluster_locks/learned_origins tables, no heartbeat/election/
  distributed-lock logic. Only server.yml/SQLite backup-cache half exists.
- [ ] (LOG) PART 9 CACHE DRIVERS — memory/valkey/redis config absent; only
  implicit in-process behavior. Required for cluster/distributed cache+locks.
- [ ] (LOG) PART 19 SCHEDULER — only 4/13 required tasks registered; missing
  ssl_renewal, geoip_update, blocklist_update, cve_update, update_check,
  log_rotation, backup_daily, backup_hourly, tor_health, cluster_heartbeat.
  No DB persistence, no distributed locking, retry policy not executed, no
  failure notification/audit/metrics, no graceful 30s stop.
- [ ] (LOG) PART 18 EMAIL — no customizable template system; SMTP client
  entirely unwired (no email ever sent). AutoDetect omits {global_ipv4} and
  never persists detected host:port to server.yml.
- [ ] (LOG) PART 21 METRICS — scheduler metrics + system (cpu/mem/disk)
  metrics missing; GoGCPauseTotalSeconds never populated; no startup warning
  when /metrics is publicly reachable.
- [ ] (LOG) PART 15 ACME — only HTTP-01/TLS-ALPN-01; DNS-01 + lego provider
  config + wildcard + encrypted provider-credential storage NOT-IMPLEMENTED.
- [ ] (LOG) PART 17 ADMIN — admin API uses forbidden legacy envelope
  ({"status":"ok"}/{"error":msg}) instead of {ok,data}/{ok:false,error,
  message} (AI.md:26050); primary-admin-undeletable concept absent (no
  is_primary); no DeleteAdmin/DisableAdmin/EnableAdmin; ~60 of ~80 PART 17
  REST endpoints unimplemented (branding, ssl renew, tor, pages, email
  templates, scheduler CRUD, backup CRUD, logs, profile/prefs).

### Routing / API path compliance (PART 14 / 16)

- [x] Swagger relocated: removed /openapi + /openapi.json; mounted
  /server/docs/swagger (UI), /api/swagger + /api/{v}/server/swagger (spec);
  UI now fetches /api/swagger. No redirects.
- [x] GraphQL relocated: removed /graphql; mounted /server/docs/graphql
  (GraphiQL), /api/graphql + /api/{v}/server/graphql (query POST).
- [x] Added routes: /api/autodiscover (public-safe, no admin_path/secrets),
  /api/healthz alias, /server (301→/server/about), /server/privacy,
  /server/contact, /api/{v}/server/{about,privacy,contact,help,terms},
  /server/consent POST.
- [x] /server/terms is now canonical; /terms 301-redirects to it.

### PART 7/8/13 (server binary / CLI / versioning)

- [x] Makefile: added `-e GOFLAGS=-buildvcs=false` to GO_DOCKER (removed -it),
  `-buildvcs=false -trimpath` on all go build calls, added `clean` target +
  .PHONY, build/local depend on clean. (Pre-logged item 6 Makefile part.)
- [x] CLI: added -h/-v short flags (src/cli/cli.go) + help text. (item 6.)
- [x] --color: default now "auto", values auto/yes/no; SetColorMode accepts
  yes/no (aliases always/never kept) (display.go, caspaste.go). (item 6.)
- [x] --version output format was wrong (caspaste.go:1300): needed 4 lines
  {name} {version} / Built: {date} / Go: {ver} / OS/Arch: {GOOS}/{GOARCH};
  forbidden `v` prefix. Fixed: 4-line format, filepath.Base(os.Args[0]) for
  name, no "v" prefix (also fixed in --help header).
- [x] version fallback was "1.0.0" (caspaste.go:69) — spec wants "dev".
  Fixed. (git-tag source and release.txt cwd-relative path not addressed —
  out of scope for this pass.)
- [x] --status returned exit code 2 for DEGRADED (caspaste.go:1057); PART 8
  (AI.md:10843) defines ONLY 0 (healthy) / 1 (unhealthy). Fixed: DEGRADED
  path now exits 1. (--status still opens the DB directly rather than
  querying the running instance/PID (AI.md:11695) — not addressed, LOG.)
- [ ] (LOG) hand-rolled arg parser (src/cli/cli.go:271) vs PART 8 requirement
  to use stdlib flag package (AI.md:10666).
- [x] --lang flag registered on server (caspaste.go) and CLI (client/main.go);
  unsupported values fall back to en; wired to i18n/Accept-Language.
- [x] --baseurl flag registered on server (flag > env BASE_URL > config >
  default /); wired into base-URL resolution.
- [x] signals: added SIGQUIT (graceful), SIGHUP (ignore/auto-reload), SIGUSR1
  (reopen logs), SIGUSR2 (status dump), SIGRTMIN+3 (graceful) via
  platform-split signals_{unix,linux,unix_other,windows}.go.
- [x] /healthz root alias now gated on server.healthz.root.enabled
  (config Server.Healthz.Root.Enabled); disabled falls through like any
  unknown path; /server/healthz stays canonical/unconditional.
- [ ] (minor) embedded-asset layout: code embeds src/web/data/* only; spec
  expects src/server/template|static + src/data (AI.md:9898).

### PART 6 runtime-mode dispatch

- [ ] (LOG) src/mode only implements Production/Development toggle, not full
  server/CLI/TUI smart-detection dispatch (PART 6). (Pre-logged item 5.)

### PART 5 / 12 config

- [ ] GenerateDefaultYAMLConfig hardcodes Linux-only dir defaults
  (yaml.go:670); runtime resolvers are per-OS so impact limited.
- [x] Bare unprefixed env vars: RESOLVED. MODE/DEBUG are spec-authored bare
  (AI.md:9858, 9863) and SMTP_* is spec-authored bare (PART 18) — correct as
  written, no change. Only PORT is genuinely ambiguous (AI.md self-contradicts:
  CASPASTE_PORT at 13300/11229 vs bare PORT at 589/617) — logged to TODO.AI.md
  as a human-only naming decision rather than guessed.
- [ ] Many PART 12 config trees absent from YAMLConfig struct: server.baseurl,
  compression, session, full rate_limit, i18n, contact, tracking, privacy.

## Pass 6: Code Flow Trace

- [ ] Dead public API: token user/org CRUD (see Pass 2); email Send/NewClient
  (see Pass 2) — both unreachable until their subsystems are wired.
- [ ] Env var completeness: PORT/SMTP_*/MODE/DEBUG read without CASPASTE_
  prefix (see PART 5/12). Verify all read env vars are documented in README/
  docs and docker-compose defaults once the config decision is made.

## Docker (PART 27)

- [x] docker/Dockerfile:73 `ENV MODE=development` removed (AI.md:38848
  — MODE never baked into image; binary defaults to production).
- [x] docker/Dockerfile:84 HEALTHCHECK timings fixed: was
  start-period=10m/interval=5m/timeout=15s; now
  start-period=90s/interval=10s/timeout=5s (AI.md:39093).

## i18n (PART 31)

- [ ] (LOG) locale path src/web/data/locale/*.json vs spec src/common/i18n/
  locales; flat map[string]string vs nested dot-keyed with plural categories
  and t/tf/tp funcs; languages present en/de/ru/bn_IN vs required
  en/es/zh/fr/ar/de/ja; no RTL/dir, no x/text date/number formatting; no
  a11y integration test.

## Completed
- (see [x] items above)

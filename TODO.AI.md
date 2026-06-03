# CasPaste — AI.md Compliance TODO

Audited: 2026-06-02
Source of truth: `AI.md` only.
Full findings: `AUDIT.AI.md`.

Dependency order: removals first (clean slate), then admin panel build-out, then doc rewrite, then validation/security tightening.

## Phase 1 — REMOVE non-spec subsystems

Anonymous pastebin per spec PART 35 (line 57173) and PART 36 (line 57869). PART 34 multi-user is optional and not in CasPaste scope.

- [ ] Delete `src/user/` (user.go, validation.go)
- [ ] Delete `src/userapi/`
- [ ] Delete `src/org/`
- [ ] Delete `src/orgapi/`
- [ ] Delete `src/domain/`
- [ ] Delete `src/domainapi/`
- [ ] Delete `src/authapi/`
- [ ] Delete `src/web/user_routes.go`
- [ ] Delete `src/web/org_routes.go`
- [ ] Delete `src/web/stub.go` and `src/web/data/stub.tmpl` (only consumer is user/org routes)
- [ ] Remove `/users/*` cases from `src/web/web.go` switch (lines 537-554)
- [ ] Remove `/orgs` branch from `src/web/web.go` (lines 593-595)
- [ ] Simplify `/.well-known/change-password` (web.go:452-459) — redirect to login only
- [ ] Remove `/server/auth/register` route + `handleRegisterPage` (web.go:529-530)
- [ ] Remove `UsersConfig`, `RegistrationConfig`, `RolesConfig`, `TokensConfig`, `ProfileConfig`, `UserAuthConfig`, `UserLimitsConfig`, `FeaturesConfig`, `OrganizationsConfig`, `CustomDomainsConfig` and `DefaultUsersConfig`/`DefaultFeaturesConfig` from `src/config/config.go`
- [ ] Remove `users:` and `features:` stanzas from default YAML config
- [ ] Remove user/org/domain endpoints from `src/apiv1/` (any `/users`, `/orgs`, `/domains` handlers)
- [ ] Remove User, Org, Domain GraphQL types/resolvers from `src/graphql/`
- [ ] Remove `_nav.tmpl` notification-bell and profile-icon blocks (PART 34 only)
- [ ] Remove `--org`, `--user`, user-API-token CLI flags from `src/client/main.go` (keep server-admin token if used for admin API)
- [ ] Drop `users`, `orgs`, `domains`, user-scoped `api_keys` tables from any schema/migration code
- [ ] Run `go vet ./... && go build ./...` and resolve every dangling import

## Phase 2 — ADMIN PANEL (PART 17)

Spec mandate: "ALL projects MUST have a full admin panel." Current `src/admin/admin.go` is placeholder HTML only.

- [ ] Mount admin panel at `/server/{admin_path}/...` (NOT `/{admin_path}/...`); update `src/server/caspaste.go:2434-2437` and `src/admin/admin.go` routes
- [ ] Implement admin authentication middleware (separate session from public site)
- [ ] Implement `admins` table in `users.db` (PART 17 line 28361) with Argon2id password hashing
- [ ] Implement setup wizard at `/server/{admin_path}/config/setup` (first-run, one-time setup token)
- [ ] Implement admin self-mgmt routes:
  - [ ] `/server/{admin_path}/{admin_username}/profile`
  - [ ] `/server/{admin_path}/{admin_username}/preferences`
  - [ ] `/server/{admin_path}/{admin_username}/notifications`
- [ ] Implement `/server/{admin_path}/config/*` pages backed by real data:
  - [ ] `/config/settings` — edit server.yml (with backup)
  - [ ] `/config/ssl` — view/manage cert; trigger renewal
  - [ ] `/config/email` — SMTP test + save
  - [ ] `/config/scheduler` — list tasks, run-now, pause
  - [ ] `/config/logs` — tail/search server log
  - [ ] `/config/logs/audit` — audit log viewer
  - [ ] `/config/backup` — list, create, restore backups
  - [ ] `/config/updates` — check, apply (wraps `src/updater/`)
  - [ ] `/config/info` — version, build, host, paths
  - [ ] `/config/metrics` — dashboard (wraps `src/metric/`)
  - [ ] `/config/network/tor` — `.onion` mgmt (wraps `src/tor/`)
  - [ ] `/config/network/geoip` — DB status, manual update (wraps `src/geoip/`)
  - [ ] `/config/security/auth` — auth provider overview (single-password mode)
  - [ ] `/config/security/auth/oidc` — OIDC provider mgmt (PART 17)
  - [ ] `/config/security/auth/ldap` — LDAP provider mgmt (PART 17)
  - [ ] `/config/security/tokens` — admin API token mgmt
  - [ ] `/config/security/firewall` — firewall rules
- [ ] Implement admin API at `/api/{api_version}/server/{admin_path}/config/*` per PART 14 (line 55837):
  - [ ] `/config/status` (GET)
  - [ ] One JSON endpoint per page above (GET + appropriate write verb)
- [ ] Emit audit log entries on every admin write action (`audit.go`)
- [ ] Enforce `--debug` bypass for admin auth only in dev (PART 6 line 9087)
- [ ] Add CSRF protection on all admin form posts (excl. API which uses tokens)
- [ ] Reserved-path validation: reject `admin_path` values in `admin.ReservedPaths`

## Phase 3 — DOCUMENTATION sync

- [ ] Rewrite `README.md` to describe only the trimmed scope (no multi-user, no orgs, no domains)
- [ ] Rewrite `IDEA.md` Project description + Business logic to drop multi-user/orgs/domains; keep compat layer and core paste features
- [ ] Remove or rewrite any `docs/*.md` files referencing user accounts, orgs, custom domains
- [ ] Update CLI `--help` (`src/client/main.go`) to remove user/org context; document env vars
- [ ] Document every `CASPASTE_*`, `SMTP_*` env var in README env-var section
- [ ] If `man/` pages are intended, add `caspaste.1` and `caspaste-cli.1`; otherwise note absence is acceptable for Go binaries

## Phase 4 — CORRECTNESS & SECURITY tightening

- [ ] Validate `original_url` scheme allowlist (http/https only) in URL shortener — reject `javascript:`, `data:`, `file:`
- [ ] Enforce `body_max_len` before reading request body into memory (avoid OOM)
- [ ] Sanitize uploaded `file_name` for control characters
- [ ] Audit `src/web/web.go` route prefix shadowing: ensure 8-char paste IDs cannot collide with `/dl/`, `/u/`, `/qr/`, `/emb/`, `/emb_help/`, `/edit/`, `/server/`, `/api/`, `/docs/`, `/auth/`, `/openapi`, `/graphql`, `/admin/` (after relocate to `/server/admin/`)
- [ ] Confirm `/server/healthz` response includes pastebin extension fields: `features.syntax_highlighting`, `stats.pastes_total`, `stats.pastes_24h`, `checks.storage`
- [ ] Verify session HMAC secret persists across restarts (`src/web/auth.go`)
- [ ] Verify pprof/expvar/debug endpoints are gated by `--debug` AND bound only to a safe listener (not public)
- [ ] Confirm all compat endpoints CSRF-exempt AND rate-limited per IDEA "Security decisions & exceptions"
- [ ] Confirm burn-after-reading delete completes before response returns (race-free)

## Phase 5 — REPO HYGIENE

- [ ] Move `binaries/` content out of repo (use release assets), keep dir in `.gitignore`
- [ ] Remove `.go-cache/` from repo; add to `.gitignore`
- [ ] Rename `test/` → `tests/` (or confirm Go convention exception)
- [ ] Verify `Jenkinsfile` is intentional; if not used, remove
- [ ] Verify all `.github/workflows/*.yml` build/test only surviving code
- [ ] Confirm `release.txt`, `site.txt` are intentional or remove

## Phase 6 — VALIDATION

- [ ] `go vet ./... && go build ./... && go test ./...` all clean
- [ ] `go-lint` clean
- [ ] Manual smoke: create paste anonymously, view, edit (if editable), burn-after-read, list public, URL shortener redirect, QR code, embedded view, file upload, download, healthz JSON, compat sprunge/ix/termbin/pastebin/hastebin POST + GET
- [ ] Admin smoke: setup wizard → create admin → login → each config page renders real data → audit log records actions
- [ ] Delete `AUDIT.AI.md` and this `TODO.AI.md` only when every box above is checked

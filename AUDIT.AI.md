# CasPaste — Spec Compliance Audit

Started: 2026-06-02
Spec: `AI.md` (single source of truth). `IDEA.md`, `TODO.AI.md`, `CLAUDE.md`, `README.md` ignored as references.

## How to read this file

Each issue has a status (REMOVE / MISSING / PARTIAL / FIX), a path, and a short reason tying it back to the spec.

- **REMOVE** — code exists but spec does not require it for a pastebin (anonymous, single-user paste service). Spec evidence: PART 35 line 57173 (Pastebin = anonymous/individual pastes, NO orgs), PART 36 line 57869 (Pastebin = anonymous, NO custom domains). PART 34 multi-user is optional and the spec's pastebin example flows through "Anonymous/individual pastes" — multi-user is not part of CasPaste's product scope.
- **MISSING** — spec mandates it and code does not have it (or has it as a stub).
- **PARTIAL** — code exists but does not match the spec's structure or behavior.
- **FIX** — code exists, spec exists, but the code violates a specific rule.

## Spec evidence that pastebin = anonymous, no users/orgs/domains

- `AI.md:57173` — `| **Pastebin** | Hastebin, PrivateBin | Anonymous/individual pastes |` (Organizations decision matrix: "Projects that do NOT need organizations")
- `AI.md:57869` — `| **Pastebin (anonymous)** | PrivateBin, Hastebin | Anonymous pastes, no branding |` (Custom Domains: "Projects that don't benefit from custom domains")
- `AI.md:52107` — Pastebin listed under "no follow/social" projects
- PART 34/35/36 are all marked **OPTIONAL** and CasPaste's product scope (per spec examples) does not include them.
- PART 17 Admin Panel **IS** required ("ALL projects MUST have a full admin panel.") — but the actual admin code is a placeholder, not an admin panel.

---

## Pass 1: Security

- [ ] `src/email/email.go:55-78`: SMTP credentials read from env vars without warning when fields are also in config file. Spec mandates config-driven; env overrides are allowed but credential overrides should not silently override file values without logging.
- [ ] `src/session/session.go`: verify session token rotation on privilege change — needs review for fixation.
- [ ] `src/web/auth.go:63-104`: HMAC session cookie — verify secret is loaded from persistent state, not regenerated on each restart (would invalidate all sessions). Confirm against PART 11.
- [ ] `src/server/caspaste.go:2441-2462`: `--debug` flag enables pprof/expvar/heap on all interfaces. Spec PART 6 line 9087 says debug bypasses admin auth — verify the debug routes are bound only to localhost when binding 0.0.0.0.
- [ ] `src/compat/*.go`: confirm all compat endpoints are CSRF-exempt and rate-limited as documented; do not allow GETs that mutate state.

## Pass 2: Code Quality — Major Removals (NOT IN SPEC FOR PASTEBIN)

- [ ] **REMOVE `src/user/` package (946 lines).** Multi-user accounts. Spec PART 34 is optional; pastebin per spec is anonymous. Files: `user.go`, `validation.go`.
- [ ] **REMOVE `src/userapi/` package (765 lines).** User profile/settings API. Same reason.
- [ ] **REMOVE `src/org/` package (646 lines).** Organizations. Spec PART 35 explicitly lists Pastebin as a project that does NOT need orgs (line 57173).
- [ ] **REMOVE `src/orgapi/` package (871 lines).** Organization API. Same reason.
- [ ] **REMOVE `src/domain/` package (895 lines).** Custom domains. Spec PART 36 explicitly lists Pastebin (anonymous) as a project that does not benefit from custom domains (line 57869).
- [ ] **REMOVE `src/domainapi/` package (715 lines).** Custom domain API. Same reason.
- [ ] **REMOVE `src/authapi/` package (880 lines).** Multi-user auth API (register, login, password reset, MFA enrollment). Single-password mode (caspasswd) is the only auth needed.
- [ ] **REMOVE `src/web/user_routes.go` (entire file).** Stub `/users/*` routes. None of these are in CasPaste's actual scope.
- [ ] **REMOVE `src/web/org_routes.go` (entire file).** Stub `/orgs/*` routes.
- [ ] **REMOVE `src/web/auth_routes.go` registration/MFA branches.** Keep only single-password login/logout per spec single-password mode.
- [ ] **REMOVE `src/web/web.go:537-554` `/users/*` route table entries.**
- [ ] **REMOVE `src/web/web.go:593-595` `/orgs` prefix branch.**
- [ ] **REMOVE `src/web/web.go:453-458` `/.well-known/change-password` user redirect** — replace with simple redirect to login (single-password mode has no user-owned password page).
- [ ] **REMOVE `config.UsersConfig`, `config.OrganizationsConfig`, `config.CustomDomainsConfig`** in `src/config/config.go:138-248`. Plus `DefaultUsersConfig`, `DefaultFeaturesConfig`. The `users:`, `features:` YAML stanzas should be deleted from default config.
- [ ] **REMOVE references to PART 34/35/36 in templates:** `src/web/data/_nav.tmpl:36,73` (notification bell, profile icon — both only required when PART 34 implemented).
- [ ] **REMOVE `_users`, `_orgs`, `users`, `orgs`, `domains` tables** from any DB schema/migration code.
- [ ] **REMOVE `src/user/validation.go` UsernameBlocklist/ValidateUsername/ValidateEmail/ValidatePassword** — not used after user removal.
- [ ] **REMOVE `src/client/main.go:296` "Create paste in org context (PART 34/35)" help text and the `--org` flag** (if present).

## Pass 2: Code Quality — Other Findings

- [ ] `src/web/web.go:411-412`: `data.StubPage` template — name implies temporary; the entire stub system is wired to user/org/domain routes that are being removed. Delete `src/web/stub.go` and `data/stub.tmpl` if no surviving consumer.
- [ ] `src/server/caspaste.go:2531`: comment `Legacy compat stubs` — verify these are still required by IDEA spec for sprunge/ix/etc, not user-related stubs.
- [ ] `src/web/user_routes.go:116-300+`: heavily commented-out / stub builder code; remove with package.

## Pass 3: Logic & Correctness

- [ ] `src/web/web.go` route switch is a giant `case` — many routes route to handlers that themselves render stubs. After REMOVE pass, prune dead branches.
- [ ] `src/web/web.go:530`: `/server/auth/register` handler exists but registration is not in spec for single-password mode. Remove handler and route.
- [ ] `src/server/caspaste.go:2580-2660`: scheduler tasks — verify each registered task still makes sense after user/org/domain removal (e.g. session_cleanup for user sessions, ssl_renewal for custom domains).
- [ ] Verify `is_url` / `original_url` paste fields validate `original_url` to reject `javascript:`, `data:`, `file:` schemes (open redirect / SSRF / phishing).
- [ ] `src/web/edit.go`: confirm edit is gated by `is_editable` AND single-password auth — anonymous edit of an editable paste created anonymously needs a clear rule per spec endpoint inventory.
- [ ] `src/compat/lenpaste.go:131`: "Return the stub: ID + oneUse flag only, body hidden." — confirm this is correct per lenpaste protocol (not an unfinished stub).
- [ ] `src/web/web.go:566-599`: ID lookup falls through to `handleGetPaste`. Confirm `/u/`, `/dl/`, `/qr/`, `/emb/`, `/emb_help/`, `/edit/` prefixes can never collide with an 8-char paste ID (e.g., a paste with ID `edit1234` — the prefix `/edit/` would shadow it). Document or namespace.

## Pass 4: Documentation Completeness

- [ ] `README.md` describes multi-user / orgs / domains as features. Rewrite to describe only what survives the removal pass.
- [ ] `IDEA.md` describes multi-user, orgs, custom domains as in-scope. Rewrite to match the trimmed scope.
- [ ] `docs/` directory: any md files referencing user accounts, orgs, domains must be deleted or rewritten.
- [ ] CLI `--help` output must remove references to org context, user tokens, etc.
- [ ] No `man/` directory present — spec PART 33 says CLI is required; verify whether man page is required for `caspaste`/`caspaste-cli`. (Project uses interactive scripts? — no, these are Go binaries. Man pages are optional per AI.md for Go binaries, but should be checked.)
- [ ] `release.txt` and `site.txt` — verify intent; if leftover placeholders, remove.

## Pass 5: Spec & Rules Compliance

### Admin panel — spec mismatch (PART 17)
- [ ] **PARTIAL `src/admin/admin.go`**: admin panel is a complete placeholder. Every `*Content()` method returns a hardcoded `<div class="card">...</div>` string. No real settings management, no auth, no setup wizard. Spec PART 17 requires a fully functional admin panel.
- [ ] **FIX admin route hierarchy**: code mounts at `/{admin_path}/...` (e.g. `/admin/server/settings`). Spec PART 17 requires `/server/{admin_path}/config/*` (e.g. `/server/admin/config/settings`). Subroutes also wrong: spec uses `/server/{admin_path}/{admin_username}/profile`, code uses `/{admin_path}/profile`. See `AI.md:28394-28425`.
- [ ] **MISSING setup wizard at `/server/{admin_path}/config/setup`** — spec requires; admin.go does not implement.
- [ ] **MISSING admin auth gate** — admin panel has no authentication middleware. Spec PART 17 line 28349 requires admin credentials valid ONLY for admin routes.
- [ ] **MISSING admin session** separate from public site.
- [ ] **MISSING admin account storage in `users.db` `admins` table** (spec line 28361).
- [ ] **MISSING admin pages** for: scheduler view/manage, audit log viewer, real backup/restore, real update mgmt, server info, metrics dashboard, network/tor, network/geoip, security/auth, security/tokens (API tokens for admins), security/firewall.
- [ ] **MISSING admin self-management routes**: `/server/{admin_path}/{admin_username}/profile|preferences|notifications`.
- [ ] **MISSING `/server/{admin_path}/config/security/auth/oidc` and `/ldap`** — spec line 28417-18 requires.
- [ ] **MISSING admin API**: `/api/v1/server/{admin_path}/config/status` and related admin API endpoints (spec line 55837).

### Project files / layout
- [ ] `Jenkinsfile` at repo root — verify against spec PART 28 (CI/CD). Spec generally prefers GitHub Actions / Gitea — confirm Jenkinsfile is intended.
- [ ] `mkdocs.yml`, `.readthedocs.yml` — verify spec PART 30 mandates these (it does, for the docs site).
- [ ] `binaries/` directory committed with prebuilt binaries — spec generally forbids committing build artifacts. Move to release assets.
- [ ] `test/` directory at repo root — should be `tests/` (plural per spec for tooling dirs).
- [ ] `.go-cache/` committed — must be in `.gitignore`, not in repo.
- [ ] Project files exist: `README.md` ✓, `LICENSE.md` ✓, `IDEA.md` ✓, `AI.md` ✓.

### Go conventions
- [ ] Directory naming: spec says Go uses **singular** package directories. Audit `src/`:
  - `binaries/` (root-level, plural — but it's a tooling dir / output)
  - `src/` is fine
  - Plural Go dirs found: none of the listed src dirs are obviously plural except `completions/` (none present) — verify each subdir name matches its `package` declaration.
- [ ] `go.mod` — verify CGO_ENABLED=0 is enforced in build.

### Forbidden / stale files
- [ ] `TODO.AI.md` to be rewritten (this audit).
- [ ] No `CHANGELOG.md` / `AUDIT.md` / `SUMMARY.md` / `NOTES.md` / `REPORT.md` / `ANALYSIS.md` present ✓.
- [ ] `AUDIT.AI.md` (this file) will be deleted when all issues resolved.

### CI/CD
- [ ] `.github/workflows/` — verify workflows build only what survives the removal pass; remove any user/org/domain test workflows.
- [ ] Jenkinsfile — same audit.

### Healthz / app-specific
- [ ] Verify `/server/healthz` exposes the spec-required pastebin extension fields: `features.syntax_highlighting`, `stats.pastes_total`, `stats.pastes_24h`, `checks.storage`.

## Pass 6: Code Flow Trace

### Call graph & dead code from removal
After the REMOVE pass, the following imports become dead and must be cleaned:
- [ ] All references to `authapi`, `userapi`, `orgapi`, `domainapi`, `user`, `org`, `domain` packages from `src/server/caspaste.go` and `src/web/*.go`.
- [ ] `session.SessionManager` user-bound paths — single-password mode uses a single session, not per-user.
- [ ] `validation.go` username/email/password rules — drop unless they apply to caspasswd entries.
- [ ] `audit.go` — only audit events relevant to single-password admin remain.

### Environment variables
- [ ] `CASPASTE_*` env vars: 53 overrides defined in `src/config/env.go`. Verify each is documented in README and matches a field in default `server.yml`. Likely undocumented overrides — needs audit.
- [ ] `SMTP_*` env vars in `src/email/email.go` (7 vars) — must be documented; spec PART 5 requires env var docs.
- [ ] `CASPASTE_CONFIG_DIR`, `CASPASTE_DATA_DIR`, `CASPASTE_DB_DIR`, `CASPASTE_BACKUP_DIR` — used in `src/server/caspaste.go`. Document each.
- [ ] `CASPASTE_SERVER`, `CASPASTE_USERNAME`, `CASPASTE_PASSWORD`, `CASPASTE_TOKEN` (`src/client/main.go`) — CLI env vars. Document in CLI `--help` and README.
- [ ] `TZ`, `DEBUG`, `MODE`, `NO_COLOR`, `HOME`, `XDG_CONFIG_HOME`, `APPDATA`, `LOCALAPPDATA`, `PROGRAMDATA`, `SSH_CLIENT`, `SSH_TTY`, `MOSH_CONNECTION`, `DISPLAY`, `WAYLAND_DISPLAY`, `TERM`, `container`, `WINDIR`, `__CFBundleIdentifier` — all OS/environment-detection vars, fine, but document the ones the operator can set.

### Visibility audit
- [ ] After removing `user`, `org`, `domain` packages, scan all remaining `pub`/exported symbols for unused exports (likely many in `web`, `apiv1`, `admin`).

### Input validation gaps
- [ ] `original_url` in URL shortener — validate scheme allowlist.
- [ ] Paste body size — verify `body_max_len` enforced before reading body into memory (avoid memory-exhaustion attack).
- [ ] Uploaded filename — sanitize for control characters; prevent display injection.

## API surface (PART 14)

- [ ] Confirm only the spec-listed API routes exist in `src/apiv1/`:
  - `/api/v1/server/healthz`, `/api/v1/server/info`, `/api/v1/pastes` (POST/GET), `/api/v1/pastes/{id}` (GET).
  - Remove any user/org/domain endpoints (`/api/v1/users/*`, `/api/v1/orgs/*`, `/api/v1/domains/*`).
- [ ] GraphQL endpoint exists per `src/graphql/` — spec PART 14 mandates this; verify schema matches trimmed scope (no User, Org, Domain types).
- [ ] OpenAPI/Swagger at `/openapi` and `/openapi.json` — verify it documents only surviving routes.

## Compat layer (PART 14)

- [ ] `src/compat/` files: sprunge, ix, termbin, pastebin, stikked, microbin, lenpaste, hastebin, generic. Verify each matches its target service exactly (response format, status code, content type, CSRF-exempt, rate-limited). Listed as IN-SCOPE in IDEA business logic; AI.md describes compat as project-driven, so these stay.

---

## Removal Plan Summary (line counts to delete)

| Package           | Lines | Reason |
|-------------------|-------|--------|
| `src/user/`       | 946   | PART 34 not in CasPaste scope |
| `src/userapi/`    | 765   | PART 34 not in CasPaste scope |
| `src/org/`        | 646   | PART 35 — pastebin = no orgs |
| `src/orgapi/`     | 871   | PART 35 — pastebin = no orgs |
| `src/domain/`     | 895   | PART 36 — pastebin = no custom domains |
| `src/domainapi/`  | 715   | PART 36 — pastebin = no custom domains |
| `src/authapi/`    | 880   | Multi-user auth API; single-password is in `web/auth.go` |
| `src/web/user_routes.go`  | ~300 | Stubs |
| `src/web/org_routes.go`   | ~150 | Stubs |
| `config.Users*` + `config.Features*` | ~150 | Config for removed features |
| **Total**         | **~6300+** | |

---

## Missing/incomplete relative to PART 17 (admin panel)

The admin panel is the largest missing piece. PART 17 is mandatory; current `src/admin/admin.go` is a stub. Building it out requires:

1. Move mount point to `/server/{admin_path}/...`
2. Implement admin auth (separate from public site)
3. Build setup wizard (`/server/{admin_path}/config/setup`)
4. Real pages backed by real data for: settings, ssl, email, scheduler, logs, backup, updates, info, metrics, network/tor, network/geoip, security/auth, security/tokens, security/firewall
5. Admin self-mgmt: profile, preferences, notifications
6. Admin API under `/api/v1/server/{admin_path}/config/*`
7. Audit log entries on admin actions

This is multi-week work and probably the largest single deliverable.

## Completed

(none yet — audit only)

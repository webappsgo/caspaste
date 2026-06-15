# CasPaste — AI TODO

Source of truth: `AI.md` only. All items reference a spec PART.

## Pending

### Admin panel completeness (PART 17)

- [ ] `/config/settings` POST — actually save settings to server.yml (currently redirects only)
- [ ] `/config/email` POST — save SMTP config + test send
- [ ] `/config/backup` POST — trigger actual backup (currently stub)
- [ ] `/config/ssl` — show live cert status (expiry, issuer) from running server
- [ ] `/config/network/tor` — show live onion address from `src/tor/`
- [ ] `/config/network/geoip` — show DB path/version from `src/geoip/`
- [ ] Admin invite acceptance route at `/server/auth/invite/server/{token}` (PART 17 line 29019)
- [ ] CSRF protection on all admin form POSTs (currently fields present but not validated)

### Correctness (PART 14, IDEA.md)

- [ ] Confirm burn-after-reading delete completes before HTTP response is written (no race)
  — currently a two-step confirm flow; the delete happens before rendering but two
  concurrent confirms could both succeed; needs a conditional DELETE in the DB layer

### CI/CD bootstrap (PART 28)

- [x] Triggered `build-toolchain.yml` — `:build` image now live at ghcr.io/casjay-forks/caspaste:build
  Fixed Dockerfile.build bugs: staticcheck wrong module (golang.org/x/tools → honnef.co/go/tools),
  trufflehog requires CGO → switched to prebuilt binary installer

## Completed (do not re-do)

- README.md: full spec-compliant rewrite (canonical section order, all 8 platforms in Install,
  official site line, correct Docker compose snippets, correct API endpoints, emoji headers)
- LICENSE.md: rewrite third-party section — compact table format, all direct + indirect deps,
  correct copyright holder (casapps), BSD-3-Clause non-endorsement clauses included
- Compat endpoints: all 6 create handlers now call checkRateLimit() (IDEA.md security rules)
- Filename sanitization: sanitizeFilename() strips control chars/path separators in netshare


- Non-spec packages removed: user, userapi, org, orgapi, domain, domainapi, authapi
- Admin panel (PART 17): routes, handlers, API, auth, session, setup wizard — fully implemented
- Admin panel: /config/admins route + admin invite generation added
- Storage: all required tables — pastes, admins, admin_sessions, admin_tokens, admin_invites,
  users, user_sessions, user_tokens, recovery_keys, orgs, org_members, custom_domains, etc.
- CI/CD: ci.yml, release.yml, beta.yml, daily.yml, docker.yml, build-toolchain.yml
- Makefile: 6 targets only, go-state named volume, 8 platforms, ≥80% coverage gate
- Dockerfile, docker-compose.yml, docker-compose.dev.yml, docker-compose.test.yml per spec
- docker/rootfs/usr/local/bin/entrypoint.sh — minimal, no mkdir, tracked in git
- tests/: run_tests.sh, docker.sh, incus.sh per PART 29
- IDEA.md: spec-compliant rewrite (no HOW details, no hardcoded paths, correct variables)
- .claude/rules/: all 14 required files, all with NON-NEGOTIABLE warning + NEVER/ALWAYS sections
- CLAUDE.md: updated to efficient loader format, references all 14 rule files
- ShowRegister dead code removed (registration not in single-password-mode spec)
- /server/auth/register removed from IsPublicPath (not a spec endpoint)
- http.MaxBytesReader added to paste creation handlers (OOM prevention)
- URL scheme allowlist in netshare/paste.go (rejects javascript:, data:, file:)
- Healthz: features.syntax_highlighting, stats.pastes_total, stats.pastes_24h, checks.storage
- AUDIT.AI.md: deleted (all audit items resolved or tracked here)

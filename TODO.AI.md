## [ ] Wire remaining PART 6 Four Operational States behaviors
Read: AI.md PART 6 (line 9251)
PART 6 "Application Modes" is now implemented: `--mode`/`--debug` flags
and `MODE`/`DEBUG` env vars (flag > env > mode-alias > default), the full
`/debug/*` route set (pprof, expvar, config, routes, cache, db,
scheduler, memory, goroutines) gated behind `mode.IsDebugEnabled()` and
unreachable when off, dev-mode debug-level logging, disabled rate
limiting in dev, verbose panic recovery in dev/debug, and disabled
static-asset caching in dev (src/web/etag.go). A pre-existing debug-mode
admin-auth bypass in src/admin/middleware.go (violated PART 6's "debug
never bypasses security checks") was found and removed as part of this
work.
Three Four-Operational-States behaviors were intentionally left unwired,
each for a concrete reason rather than an oversight:
- CORS relaxation in dev — `CORSMiddleware` is already permissive by
  design; no separate strict/relaxed toggle exists to branch on.
- Security-header relaxation in dev — `SecurityHeadersMiddleware` is
  fully config-driven with no explicit-vs-default distinction; loosening
  it risks silently overriding intentional user config.
- Template hot-reload in dev — templates are parsed once from an
  embedded FS at startup with no dynamic reload mechanism; only the
  cache-header side was wired (etag.go), not actual reparsing.
Revisit if/when CORS config, security-header config, or the template
loader gain the hooks needed to distinguish "default" from
"user-configured" behavior.

## [x] PORT env-var name resolved — CASPASTE_PORT
AI.md is not actually contradictory: `{PROJECT_NAME}_PORT` (=
`CASPASTE_PORT`) is the server binary's own flag-fallback env var (PART 8
flag table, native runs); the Docker sections' bare `PORT` is a
container-only convention already translated into `--port` by
docker/rootfs/.../entrypoint.sh before the binary runs, so the binary
itself never needs to read it. src/server/caspaste.go now reads only
CASPASTE_PORT (src/config/env.go's getEnv() already prefixed correctly).
README.md already documented CASPASTE_PORT; no doc change needed.

## [ ] Migrate frozen `internal_org` from `casapps` to `webappsgo`
User confirmed `webappsgo` is canonical (matches IDEA.md line 33 and
src/path/path.go). Everything else currently hardcodes `casapps/` and must
be migrated to match:
- `src/server/caspaste.go`, `src/storage/storage.go`, `src/config/yaml.go`
  (incl. GenerateDefaultYAMLConfig's Linux-only literal — AUDIT.AI.md Pass 5
  item folds in here), `src/client/main.go`, `src/tui/app.go` — replace
  hardcoded `casapps/` path segments with `webappsgo/`.
- IDEA.md line 50 — fix the macOS bundle-id example from
  `io.github.casapps.caspaste` to `io.github.webappsgo.caspaste` so IDEA.md
  is internally consistent.
- Once all resolvers agree, consolidate the duplicate per-OS path-resolution
  logic in server/storage/client/tui into the single `src/path` resolver
  (currently only imported by `src/ssl/ssl.go`) so this cannot drift again.
- Any existing on-disk `casapps/` install data is orphaned by this rename;
  note it in README/docs as a breaking path change for anyone who ran a
  pre-migration build.

## [x] Backup/restore CLI now delegates to backup.Service — legacy tar/cp shell-out removed
Read: AI.md PART 22 (line 35411)
`performBackup`/`performRestore` in src/server/caspaste.go previously
shelled out to `tar`/`cp` and silently swallowed subprocess errors, with
no manifest, checksum, or encryption. Rewritten to call
`backup.Service.Create`/`Restore` (manifest.json + checksum, optional
AES-256-GCM+Argon2id encryption via `promptBackupPassword`/
`BACKUP_PASSWORD`, immediate post-create verify, retention). Server
startup now also registers `backup_daily` (02:00, enabled per
`server.backup.enabled`) and `backup_hourly` (hourly, disabled unless
`server.backup.hourly_enabled`) as scheduler tasks using the same
`backup.Service`.

## [x] backup_daily disk-space pre-check (`backup.skipped_disk_full`)
Read: AI.md PART 22 § "Backup Creation Flow (backup_daily task at 02:00)" step 2
Added `diskFreeBytes` (src/backup/diskusage_unix.go / _windows.go,
Bavail-based on Unix, GetDiskFreeSpaceEx free-available on Windows) and
`backup.Service.CheckDiskSpace()` (src/backup/backup.go), which returns
skip=true with a reason when disk usage exceeds the 90% threshold or free
space is under 2x the most recent backup's size. Both `backup_daily` and
`backup_hourly` scheduler task handlers (src/server/caspaste.go) call it
before `backupSvc.Create` and log `backup.skipped_disk_full` via
`log.Warn` + return nil (skip, not failure) when triggered. Not wired
into manual/CLI-triggered backups (`performBackup`) — the spec text
scopes this check to "a scheduled backup" only.

## [ ] GeoIP country-blocking middleware not implemented
Read: AI.md PART 20 and PART 6 (middleware order)
`geoip.Client` is now instantiated at startup (src/server/caspaste.go),
loads its MMDB on boot (warns, doesn't fail, if missing), and is
refreshed weekly by the new `geoip_update` scheduler task
(server.geoip.enabled gate). The actual request-time enforcement
(deny_countries/allow_countries via `geoipClient.LookupRequest`) is NOT
wired into the middleware chain — PART 6's full order (URLNormalize →
RequestID → PathSecurity → SecurityHeaders → Allowlist → Blocklist →
RateLimit → GeoIP → Auth → Logging) doesn't exist yet; the current chain
in main() only has URLNormalize → PathSecurity → PanicRecovery →
RequestID → Metrics → SecurityHeaders → CORS → CSRF → Maintenance → App.
Allowlist, Blocklist, RateLimit, and GeoIP middleware all need to be
built and inserted in spec order — this is a substantial standalone
feature, not a small follow-up.

## [x] healthcheck_self task was mutating data instead of checking health
Read: AI.md PART 19 (line 33460, 33643)
`healthcheck_self` (every 5m) previously called `db.PasteDeleteExpired()`
and returned its error — a destructive write duplicating the separate
`paste_cleanup` task, not an actual health verification. Changed to
`db.Pool().PingContext(ctx)` (src/server/caspaste.go), a non-mutating
DB-responsiveness check consistent with PART 10's `PingContext` pattern.

## [ ] ssl_renewal, log_rotation, blocklist_update, cve_update scheduler tasks missing
Read: AI.md PART 19 § "Built-in Tasks (Required)"
Only paste_cleanup, session_cleanup, token_cleanup, healthcheck_self,
backup_daily, backup_hourly, geoip_update, and update_check (see separate
entry below) are registered in main(). Still missing: `ssl_renewal` (needs
the ACME/cert-renewal implementation from PART 15 to call into),
`log_rotation` (src/logger has no rotation support to invoke),
`blocklist_update`/`cve_update` (no backing service/download logic exists
for either).

## [x] update_check scheduler task was unregistered; --update branch didn't persist
Read: AI.md PART 23 (line 36186)
Three related bugs fixed together (same feature area): (1) `update_check`
was never registered as a scheduler task — added (src/server/caspaste.go),
daily 06:00, skippable, reads `server.update.{branch,auto_install,
defer_days}`; notify-only by default, runs the full DoUpdate+restart flow
when `auto_install: true`. (2) `src/updater/update.go` had no defer_days
awareness at all — added `Release.PublishedAt` and a new
`CheckForUpdateEligible(ctx, cfg, deferDays)` that fetches the full
releases list, filters by branch/eligibility (`published_at` cutoff), and
selects the newest eligible release newer than the running version; manual
`--update check`/`--update yes` still call the original `CheckForUpdate`
and correctly ignore defer_days per spec ("an explicit operator action
overrides the defer window"). (3) `--update branch {name}` only printed
"not persisted" instead of writing to config, directly contradicting
PART 23's "the config is the single source of truth" rule — now loads/
saves `server.update.branch` via a new `Update` section in
src/config/yaml.go (defaults: branch=stable, auto_install=false,
defer_days=0), and `--update check`/`--update yes` now read the persisted
branch instead of hardcoding "stable". Cluster-mode node-by-node
auto-install rollout is N/A until cluster mode (PART 10) is implemented.

## [ ] "Update available" admin banner / Notification Center not implemented
Read: AI.md PART 23 § "Surfacing rules"
The `update_check` task currently only logs `update_available: vX -> vY`
server-side. PART 23 requires this to surface as a banner + Notification
Center entry on `/server/{admin_path}/*` (Server Admins only, fires once
per version, dismiss suppresses only that version) plus an off-by-default
`update_available` email event. No notification system/admin banner
mechanism exists in this project yet — needs its own pass once the admin
panel notification infrastructure exists.

## [ ] cluster_heartbeat / cluster mode (PART 10) and cache drivers (PART 9) not implemented
Read: AI.md PART 9, PART 10
No `cluster` or `cache` package exists. Single-instance mode only;
`valkey`/`redis` cache drivers and cluster-mode distributed locking for
global scheduler tasks are unimplemented. Required before multi-node
deployment is possible per spec.

## [ ] Repo-wide gofmt drift predates this session's changes
Read: AI.md PART 26 (Go formatting via `gofmt`)
`gofmt -l ./src ./cmd` flags ~75 files across admin/, apiv1/, storage/,
web/, etc. (leading blank lines before file-header comments, trailing
whitespace/indent inconsistencies) — confirmed NOT introduced by this
session's edits (only src/server/caspaste.go and src/config/yaml.go were
touched and reformatted, both now gofmt-clean). Needs a dedicated
formatting pass across the full `src/` tree, reviewed as its own change
so it doesn't get bundled into unrelated commits.

## [ ] Multi-user (PART 34), Organizations (PART 35), Custom Domains (PART 36) unimplemented
Read: AI.md PART 34, 35, 36
Optional feature set — confirm with IDEA.md whether CasPaste needs
end-user accounts/orgs/custom domains before starting; currently no
`users` table, org model, or domain-verification flow exists.

## [ ] i18n overhaul (PART 31) and admin API expansion (PART 17) outstanding
Read: AI.md PART 31, PART 17
en/es/zh/fr/ar/de/ja translation-key coverage and the full admin API
surface (per PART 17's required routes) have not been audited/completed
this session.


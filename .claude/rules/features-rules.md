# Feature Rules (PART 18-23)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

**Email (PART 18)**
- NEVER attempt to send email without valid, tested SMTP
- NEVER queue emails hoping SMTP will be configured later
- NEVER log "would have sent email" messages
- NEVER show email-dependent UI when SMTP is unavailable
- NEVER omit a visible plaintext link, disclaimer, or recipient address from an account email
- NEVER include a link that deletes/modifies data without prior authorization in an unsolicited email

**Scheduler (PART 19)**
- NEVER use an external scheduler (cron, systemd timers, Task Scheduler, launchd, Kubernetes CronJob, cloud schedulers) — no exceptions, ever
- NEVER let more than one cluster node run a "global" task simultaneously
- NEVER disable the scheduler itself (only individual tasks are toggleable)

**GeoIP (PART 20)**
- NEVER embed GeoIP databases in the binary — always download on first run / via scheduler
- NEVER use `geoip2-golang` — ip-location-db's custom `database_type` strings break it; use `github.com/oschwald/maxminddb-golang`
- NEVER set both `deny_countries` and `allow_countries` and treat them as additive (allow wins, deny is ignored)
- NEVER country-block RFC 1918 / private IPs, or allowlisted IPs

**Metrics (PART 21)**
- NEVER expose `/metrics` publicly — internal only (firewall/proxy/NetworkPolicy)
- NEVER put a raw client IP, user ID, or request ID in a metric label (unbounded cardinality = memory DoS)
- NEVER use non-base units (ms instead of seconds, KB instead of bytes)
- NEVER omit the `{project_name}_` prefix or `_total`/`_seconds`/`_bytes` suffixes

**Backup & Restore (PART 22)**
- NEVER delete old backups until the new backup passes ALL verification checks
- NEVER store the backup encryption password anywhere — it is never persisted
- NEVER allow backups to run in compliance mode without an encryption password set
- NEVER let a restored Primary Admin skip re-authentication via setup token
- NEVER treat `backup_daily`/`backup_hourly` incrementals as counted against `max_backups`

**Update Command (PART 23)**
- NEVER install an update without a verified SHA256 checksum ("no checksum asset = refuse")
- NEVER auto-install updates unless `update.auto_install: true` is explicitly set (default OFF)
- NEVER roll out an auto-install to all cluster nodes at once
- NEVER surface update/version info on public pages (Tier 3 info, PART 11) — admin-only, except the PWA "new version" banner

## CRITICAL - ALWAYS DO

**Email (PART 18)**
- ALWAYS fall back to the embedded default template when no custom template exists
- ALWAYS auto-detect local SMTP (loopback → docker bridge → gateway → fqdn → global IPv4 → mail.{fqdn} → smtp.{fqdn}) on first run, in that priority order
- ALWAYS retest the configured SMTP connection on every startup
- ALWAYS honor `SMTP_*` env vars over `server.yml` config
- ALWAYS suppress `scheduler_error` when `backup_failed` or `ssl_renewal_failed` fires for the same execution (one notification, not two)
- ALWAYS include why-sent, recipient, app identity, visible plaintext link, and disclaimer in account emails

**Scheduler (PART 19)**
- ALWAYS persist task state (`last_run`, `next_run`, `run_count`, `fail_count`, lock info) in `server.db`
- ALWAYS run missed tasks on startup if within `catch_up_window`
- ALWAYS use database-backed distributed locking for global tasks in cluster mode (5 min lock timeout)
- ALWAYS wait up to 30s for running tasks to finish on shutdown, then force-release locks and mark for retry

**GeoIP (PART 20)**
- ALWAYS download/update MMDB files from sapics/ip-location-db (no API key)
- ALWAYS treat both IPv4 and IPv6 city databases as separate MMDB files (no combined file)
- ALWAYS skip country blocking with a warning if `country.mmdb` is missing

**Metrics (PART 21)**
- ALWAYS prefix metrics with `{project_name}_`, snake_case, base units, `_total` for counters
- ALWAYS normalize dynamic path segments (UUIDs, numeric IDs) to `:id` before labeling
- ALWAYS support optional bearer-token auth on `/metrics`

**Backup & Restore (PART 22)**
- ALWAYS verify a backup immediately after creation (existence, size>0, checksum, decrypt test, manifest, extraction, DB integrity) — ALL checks must pass
- ALWAYS check free space (< 2x last backup size, or disk usage > 90%) before starting a scheduled backup, and skip with `backup.skipped_disk_full` if insufficient
- ALWAYS require the backup password on restore of a `.tar.gz.enc` file (CLI prompt/flag, WebUI dialog, or API 400 error)
- ALWAYS require Primary Admin re-authentication via one-time setup token after restoring to a new server
- ALWAYS apply retention priority yearly > monthly > weekly > daily, oldest deleted first, then apply `max_total_size` hard cap last

**Update Command (PART 23)**
- ALWAYS treat channels as cumulative: beta = beta+stable, daily = daily+beta+stable
- ALWAYS respect `defer_days` for the scheduled `update_check` task; ALWAYS ignore it for manual `--update check`/`--update yes`
- ALWAYS write `--update branch {name}` to `update.branch` in config (config is the single source of truth)
- ALWAYS use platform-specific binary replacement (atomic rename on Unix; rename-to-.old + MOVEFILE_DELAY_UNTIL_REBOOT on Windows)

## KEY DECISIONS (pre-answered)

| Question | Answer | Spec Reference |
|----------|--------|----------------|
| Can I add cron support if the user asks? | No, never — built-in scheduler only | PART 19 § NEVER Use External Schedulers |
| Where do custom email templates live? | `{config_dir}/template/email/`; defaults embedded at `src/server/template/email/` | PART 18 § Template Storage |
| What happens with no SMTP? | Password reset hidden, email_verify skipped (auto-verify), all other email disabled | PART 18 § SMTP Requirement |
| Which Go library for GeoIP? | `github.com/oschwald/maxminddb-golang`, not `geoip2-golang` | PART 20 § Database Sources |
| Is `/metrics` public? | No — internal only, firewalled or token-authed | PART 21 § Access Control |
| Are backups encrypted by default? | Only if a backup password is set, or compliance mode forces it | PART 22 § Backup Encryption |
| How many files does `backup_daily` create? | 2: full backup + daily incremental (3 with hourly enabled) | PART 19 § Built-in Tasks; PART 22 |
| Does `update.auto_install` default on? | No, default `false` — update_check only notifies | PART 23 § Update Configuration |
| Does defer_days block manual updates? | No — only gates the scheduled `update_check` task | PART 23 § Defer Semantics |
| Is update version info shown to visitors? | No — Tier 3, admin-only (except PWA update banner) | PART 23 § Surfacing rules |

## TERMINOLOGY

| Term | Meaning |
|------|---------|
| Global task | Scheduler task that runs on exactly ONE cluster node (e.g. `backup_daily`, `ssl_renewal`, `geoip_update`) |
| Local task | Scheduler task that runs on EVERY cluster node (e.g. `session_cleanup`, `healthcheck_self`) |
| Catch-up window | Duration after restart within which missed scheduled tasks are still run |
| Account email | An email tied to a specific user's account/security (welcome, password reset, login alert, etc.); must meet Account Email Requirements |
| Defer window | `defer_days` — minimum age (in days) a release must reach before `update_check` treats it as eligible |
| Channel/branch | Update release track: `stable`, `beta`, or `daily` — cumulative, not mutually exclusive |
| Daily incremental | The single always-replaced `{project_name}-daily.tar.gz[.enc]` backup file, never counted toward `max_backups` |
| Compliance mode | `server.compliance.enabled: true` — forces backup encryption; blocks unencrypted backups |

## QUICK REFERENCE

| Sub-feature | Default schedule/state | Config root |
|---|---|---|
| SMTP autodetect | On first run + retested every startup | `server.notifications.email.smtp` |
| `ssl_renewal` task | Daily 03:00 | `server.scheduler.tasks.ssl_renewal` |
| `geoip_update` task | Weekly Sun 03:00 | `server.scheduler.tasks.geoip_update` |
| `blocklist_update` / `cve_update` | Daily 04:00 / 05:00 | `server.scheduler.tasks` |
| `update_check` task | Daily 06:00, notify-only | `server.update` |
| `session_cleanup` / `token_cleanup` | Every 15 min | `server.scheduler.tasks` |
| `log_rotation` | Daily 00:00 | `server.scheduler.tasks.log_rotation` |
| `backup_daily` | Daily 02:00, enabled | `server.backup`, `server.scheduler.tasks.backup_daily` |
| `backup_hourly` | Hourly, disabled by default | `server.scheduler.tasks.backup_hourly` |
| `healthcheck_self` | Every 5 min | `server.scheduler.tasks.healthcheck_self` |
| `tor_health` | Every 10 min (if Tor installed) | `server.scheduler.tasks.tor_health` |
| `cluster_heartbeat` | Every 30 sec (cluster only) | `server.scheduler.tasks.cluster_heartbeat` |
| GeoIP databases | ASN, Country, City (MMDB, IPv4/IPv6 separate for city) | `server.geoip.databases` |
| Metrics endpoint | `/metrics`, Prometheus text format | `server.metrics` |
| Backup retention | `max_backups:1`, weekly/monthly/yearly = 0, `max_total_size:"10%"` | `server.backup.retention` |
| Restore command | `{project_name} --maintenance restore <file>` | n/a |
| Admin recovery | `{project_name} --maintenance setup` | n/a |
| Update command | `{project_name} --update [yes\|check\|branch X]` | `server.update` |

---
For complete details, see AI.md PART 18-23

# Features Rules (PART 18, 19, 20, 21, 22, 23) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 18, PART 19, PART 20, PART 21, PART 22, PART 23

## CRITICAL — NEVER DO

- Use external schedulers (cron, systemd timers, Task Scheduler, Kubernetes CronJob, etc.)
- Attempt to send emails without a valid, tested SMTP connection
- Queue emails when SMTP is not configured
- Embed GeoIP databases in the binary — always downloaded on first run
- Use `geoip2-golang` for ip-location-db — use `maxminddb-golang` instead
- Set `deny_countries` AND `allow_countries` together — pick one mode
- Expose `/metrics` to public traffic — internal monitoring only
- Use hardcoded metric prefix — always `caspaste_` prefix
- Use high-cardinality labels (user ID, request ID) in Prometheus metrics
- Store backup encryption password — never stored, admin must remember
- Skip SHA-256 checksum verification on self-update downloads

## CRITICAL — ALWAYS DO

- Built-in scheduler: ALWAYS running from startup, persists state in `server.db`
- Auto-detect SMTP on first run; gracefully disable email features if no SMTP
- GeoIP: download from ip-location-db (no API key required) on first run
- All GeoIP/blocklist/CVE updates via built-in scheduler (PART 19)
- All scheduled tasks visible in admin panel (`/server/admin/config/scheduler`)
- Backup: `{project_name}_backup_YYYY-MM-DD_HHMMSS.tar.gz[.enc]` format with manifest
- Self-update: check via GitHub Releases API, verify SHA-256, atomic binary replace
- Email templates: embedded defaults, custom override in `{config_dir}/template/email/`

## Email (PART 18)

| SMTP State | Behavior |
|------------|----------|
| Not configured | Email features completely disabled and hidden |
| Configured but invalid | Validate on save, reject |
| Configured and working | Email enabled |

Auto-detect SMTP on first run: try `127.0.0.1`, `172.17.0.1`, gateway, FQDN on ports 25/465/587.
Required templates: `welcome`, `password_reset`, `email_verify`, `login_alert`, `security_alert`, `backup_complete`, and more.
Custom templates: `{config_dir}/template/email/` (fallback to embedded defaults if missing).

## Scheduler (PART 19)

Built-in tasks (all required):

| Task | Default Schedule | Skippable |
|------|-----------------|-----------|
| `ssl_renewal` | Daily 03:00 | No |
| `geoip_update` | Weekly Sun 03:00 | Yes |
| `blocklist_update` | Daily 04:00 | Yes |
| `cve_update` | Daily 05:00 | Yes |
| `session_cleanup` | Every 15min | No |
| `token_cleanup` | Every 15min | No |
| `log_rotation` | Daily 00:00 | No |
| `backup_daily` | Daily 02:00 | Yes |
| `healthcheck_self` | Every 5min | No |
| `tor_health` | Every 10min | No (when Tor installed) |
| `cluster_heartbeat` | Every 30s | No (cluster mode only) |

## GeoIP (PART 20)

- Source: [sapics/ip-location-db](https://github.com/sapics/ip-location-db) — no API key
- Library: `github.com/oschwald/maxminddb-golang` (NOT `geoip2-golang`)
- Files: `asn.mmdb`, `country.mmdb`, `city.mmdb`, `whois.mmdb` in `{data_dir}/security/geoip/`
- Country blocking: `deny_countries` (blocklist) XOR `allow_countries` (allowlist) — not both
- RFC 1918 IPs never country-blocked; allowlisted IPs always bypass
- Admin: `/server/admin/config/network/geoip`

## Metrics (PART 21)

- Endpoint: `/metrics` — INTERNAL ONLY, never proxy to public
- Library: `github.com/prometheus/client_golang`
- Prefix: `caspaste_`
- Types: Counter (`_total`), Gauge, Histogram (`_seconds`, `_bytes`), Summary
- Required: app info gauge, HTTP requests counter + duration histogram, DB metrics, cache metrics, scheduler metrics, system metrics

## Backup (PART 22)

- Command: `caspaste --maintenance backup [filename]`
- Format: `caspaste_backup_YYYY-MM-DD_HHMMSS.tar.gz` (or `.tar.gz.enc` if encrypted)
- Always includes: `server.yml`, `server.db`
- Optional flags: `--include-ssl`, `--include-data`
- Encryption: AES-256-GCM, key from Argon2id(password) — password NEVER stored
- Compliance mode: backups BLOCKED until encryption password set
- Manifest: `manifest.json` with version, checksum, contents list

## Update Command (PART 23)

```
caspaste --update [check|yes|branch {stable|beta|daily}]
caspaste --maintenance update   # alias for --update yes
```

| Branch | Tag Pattern |
|--------|-------------|
| `stable` | `v*`, `*.*.*` |
| `beta` | `*-beta` |
| `daily` | `YYYYMMDDHHMMSS` |

Flow: GitHub Releases API → download binary → verify SHA-256 → atomic replace → restart.
HTTP 404 from GitHub API = no updates available (already current), exit 0.

For complete details, see AI.md PART 18, PART 19, PART 20, PART 21, PART 22, PART 23

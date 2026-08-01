# Backend Rules (PART 9, 10, 11, 32)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never expose stack traces or internal error chains in production responses (PART 9)
- Never `DROP COLUMN`, `DROP TABLE`, `DELETE`, or rename a column directly — add new, migrate in app code, deprecate old (PART 10)
- Never write raw SQL string interpolation — parameterized queries only (PART 11)
- Never cast user-controlled content to `template.HTML`; never render user HTML/SVG/XML inline (PART 11)
- Never log passwords, full tokens/secrets, session tokens, recovery keys, TOTP secrets, private keys, card numbers, or unmasked emails (PART 11)
- Never expose Tier 1 data publicly (DB creds, internal IPs, tokens, other users' PII, filesystem paths, account-existence signals, exact rate-limit thresholds) — not even in debug mode (PART 11)
- Never run a query without a context timeout (PART 10)
- Never use system Tor or hardcode/default Tor ports (9050/9051) — dedicated process only, `127.0.0.1:auto` (PART 32)
- Never regenerate `installation_secret`/`cookie_signing_key`/`csrf_token_secret` per cluster node — distributed once via join payload (PART 10)
- Never let a cluster secret rotation proceed without majority-node quorum (anti-split-brain) (PART 10)

## CRITICAL - ALWAYS DO
- Always use the canonical `{ok, data}` / `{ok:false, error, message}` response envelope (PART 9, ref PART 14)
- Always log every error server-side with request_id, error_code, http_status even though the client never sees internals (PART 9)
- Always use exponential backoff for retryable errors (network/timeout/503), never retry 4xx (PART 9)
- Always use `CREATE TABLE IF NOT EXISTS` + idempotent `ALTER TABLE` — no migration files, no version table (PART 10)
- Always pool DB connections (`max_open`/`max_idle`/`max_lifetime`) and verify with `PingContext` (PART 10)
- Always wrap multi-statement writes in a transaction; use `Serializable` isolation + retry-on-conflict for inventory/reservation-style logic (PART 10)
- Always run defense-in-depth at all 4 layers (input, data access, output, transport) for SQLi/XSS/enumeration/timing/CSRF — never assume another layer catches it (PART 11)
- Always pass every public response through the Output Sanitization Pipeline: allow-list fields, redact sensitive query params, strip internal IPs/paths, truncate, strip debug fields, constant-time finalize for auth paths (PART 11)
- Always emit the full security header set (CSP, HSTS when SSL, COOP/COEP/CORP, Permissions-Policy, X-Request-ID, Reporting-Endpoints) on every response (PART 11)
- Always store tokens as SHA-256 hash + 8-char prefix only, show full value once at creation (PART 11)
- Always write audit log entries as JSON Lines with `id`, `time` (ISO 8601 UTC ms), `event`, `category`, `severity`, `actor`, `result` (PART 11)
- Always start Tor as a dedicated child process inheriting the server's (post-privilege-drop) user; treat missing Tor binary as INFO, not an error (PART 32)

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| How do schema changes ship? | Idempotent `CREATE TABLE IF NOT EXISTS` / `ALTER TABLE ADD COLUMN`, applied on every startup, no migration files | PART 10 → Schema Updates |
| How do we rename a DB column? | 3-step: add new column → app reads new/writes both → old column stays forever, unused | PART 10 → Handling Column Renames |
| Which cache driver in production? | `valkey` (preferred) or `redis`; `memory` is dev-only | PART 9 → Cache Drivers |
| What HTTP cache header for authenticated pages? | `private, no-store` always | PART 9 → HTTP Cache Headers |
| Password hashing algorithm? | Argon2id — no bcrypt/MD5/SHA-* option | PART 11 → Operator UX |
| Who becomes cluster primary? | Node with lowest ID; no preemption when old primary returns | PART 10 → Primary Election |
| What triggers cluster mode? | Auto-detected when external cache + shared Postgres/MySQL present; else single-instance | PART 10 → Cluster Mode |
| Default audit log retention? | `keep: none` — deleted on next daily rotation | PART 11 → Sane Defaults |
| Default Tor outbound routing? | Off (`use_network: false`); users may override if `allow_user_preference: true` | PART 32 → Configuration Hierarchy |
| Onion address persistence? | Keys saved under `{data_dir}/tor/site/`; reused across restarts unless explicitly regenerated | PART 32 → Behavior |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| Tier 1 / 2 / 3 (data exposure) | Never public (secrets/PII) / always public (version, health, onion address) / debug-only (stack traces, SQL, rate-limit internals) |
| `app_secrets` | DB table holding `installation_secret`, `cookie_signing_key`, `csrf_token_secret` |
| `server.security.encryption_key` | The one canonical AES-256-GCM at-rest key, stored in `server.yml`, not in `app_secrets` |
| Cluster node vs Agent | Node = another instance of this binary sharing DB/cache; Agent = separate `-agent` binary reporting in via bearer token — never confuse the two |
| Idempotent schema update | A DDL statement safe to re-run on every node/startup without side effects |
| Output Sanitization Pipeline | The mandatory 6-stage filter every public response passes through before leaving the server |
| Hidden service | Tor server-side `.onion` hosting, always on if the Tor binary is found |
| Tor outbound network | Optional SOCKS5 routing of the server's own outbound HTTP calls through Tor |

## QUICK REFERENCE
- Error codes map to fixed HTTP statuses (400/401/403/404/405/409/429/500/503) — never invent new codes ad hoc.
- Cache TTLs: session 24h, rate-limit counters 1min, user profile 5min, config 1min, blocklist 1h.
- Distributed locks: `SET key NX EX ttl`; always release only if you own it.
- Query timeouts: simple SELECT 5s, JOIN 15s, write 10s, bulk 60s, report 2min, migration 5min.
- Cluster heartbeat: 30s interval, 90s = degraded, 5min = offline, manual removal only.
- Secret rotation requires advisory lock + majority quorum before commit.
- Security headers required on every response: `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, CSP, Permissions-Policy, `X-Request-ID`; add HSTS when SSL is on.
- Audit log: JSON only, append-only, 0640 perms, mask emails by default, never log full tokens/passwords.
- Tor: v3 onion (ed25519, 56 chars), `ControlPort 127.0.0.1:auto`, `SafeLogging on`, dedicated process per app instance, missing binary logs INFO and disables the feature silently.

---
For complete details, see AI.md PART 9, 10, 11, 32

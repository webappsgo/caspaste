# caspaste

## Project description

CasPaste is a self-hostable pastebin and code-sharing service that drops in
for Pastebin, Microbin, Lenpaste, Stikked, Termbin, Hastebin, and Sprunge —
existing clients keep working after only changing the hostname. It provides a
fast, secure, privacy-focused platform for sharing text snippets, code (with
syntax highlighting), files, and short URLs, packaged as a single static
binary with all assets embedded and zero external runtime dependencies.
CasPaste is enterprise-ready with multi-user support, organizations, and
custom domains, and 100% of features are available free under the project's
open-source license (no paid tiers, no feature gating, no phone-home).

Target users:

- Developers sharing code snippets, logs, and files
- DevOps teams collaborating on configurations
- Teams needing private paste hosting and branded/organization pastebins
- Privacy-conscious users and self-hosters avoiding public pastebin services
- CLI users piping content from existing tools (sprunge, ix, termbin,
  pastebin, stikked, microbin, lenpaste, hastebin workflows)
- Open source projects self-hosting pastebin infrastructure

Official site: https://pste.us

## Project variables

```
project_name:     caspaste
project_org:      casapps
internal_name:    caspaste
app_name:         CasPaste
app_tagline:      Self-hostable pastebin & code-sharing service
official_site:    https://pste.us
maintainer_name:  CasjaysDev
maintainer_email: casjay@yahoo.com
binary_name:      caspaste
cli_binary_name:  caspaste-cli
default_port:     80
```

`internal_name` is FROZEN — it was set once at first-time setup and is never
edited. A project rename only changes `project_name`; `internal_name` stays so
`{config_dir}` / `{data_dir}` / `{log_dir}` / `{cache_dir}` / systemd unit /
`{plist_name}` remain stable on every host.

`{plist_name}` is NOT stored — it is derived at substitution time as
`io.github.casapps.caspaste`.

## Business logic

### Product scope & non-goals

In scope:

- Pastebin / code-sharing with raw text, syntax highlighting, multi-file
  pastes (Gist-style), file uploads (up to 5MB by default, configurable).
- URL shortener: create short links that redirect to original URLs; server
  never fetches the destination (SSRF prevention).
- Anonymous pastes at `/p/{paste_id}`; registered-user vanity URLs at
  `/{username}/{paste_id}`; org vanity URLs at `/{orgname}/{paste_id}`;
  custom-domain vanity URLs at `https://{domain}/{username-or-org}/{paste_id}`.
- Visibility modes: public (default) / unlisted / private; password
  protection; expiration (never / time-based 10 min–10y / burn-after-read
  1–9,999,998 views, deleted before the response is written).
- Drop-in compatibility shims for sprunge, ix, Termbin (TCP port 9999
  netcat), Stikked, Lenpaste, Microbin, Hastebin, and pastebin.com — mode
  detected per-request from the Host header or an env var override; the same
  server instance can serve multiple compat modes simultaneously on
  different virtual hostnames; response format exactly matches the target
  service (no CasPaste envelope on compat routes).
- Multi-user, organizations, and custom domains — all NON-NEGOTIABLE for
  caspaste (not optional, unlike a generic PART 34/35/36 project).
- Server admin panel covering every server setting; setup-token first-run
  flow; primary admin (undeletable).
- Built-in scheduler for backups, GeoIP/blocklist/CVE updates, log rotation,
  session cleanup, SSL renewal, health checks.
- Tor hidden service (auto-enabled when `tor` is on PATH and not explicitly
  disabled).
- Built-in metrics, GeoIP, email (SMTP), backup/restore, in-process update.
- Localization (multiple languages).
- PWA support (installable as progressive web app).
- QR code generation for paste URLs; embedded/iframe view for third-party
  sites.

Non-goals:

- No paid tiers, "pro/plus/enterprise" editions, feature flags based on
  payment, license keys, or phone-home authorization.
- No AI / ML features; no ML-based content moderation (admins moderate
  manually).
- No bulk file storage / general-purpose object storage — caspaste is a
  paste/code service, not a CDN.
- No client-side rendering frameworks (React/Vue/Angular/Svelte) —
  server-side rendering with progressive enhancement only.
- No external schedulers — built-in only.
- No SSRF surface: the server NEVER fetches URLs supplied by users (no
  link-preview, no remote-image inlining, no fetching URL-shortener
  destinations).

### Roles & permissions

| Role | Realm | Powers |
|---|---|---|
| Anonymous visitor | Public site + `/p/` | View public pastes; create anonymous pastes (random ID); rate-limited per-IP |
| Registered user | `/{username}/...` | Own pastes (vanity URLs); custom slugs; API tokens; profile/preferences; org membership |
| Organization member | `/{orgname}/...` | Create pastes under the org's vanity URL |
| Organization owner / admin | org-scoped | Manage members, custom domains, org-scoped tokens, ownership transfer |
| Server Admin (NOT OS root) | admin panel | Manage the application: config, users, orgs, custom domains, scheduler, backups, SSL, GeoIP, allow/blocklists |
| Primary Admin | admin panel | All Server Admin powers; cannot be deleted |

Server Admin accounts and Regular User accounts are separate identities;
username collisions across the two namespaces are not allowed. Server Admin
authentication and recovery flows are mandatory; Regular User MFA is
suggested, not forced.

Server Admins cannot view another admin's credentials or account details
(password, API token, 2FA secret) — only total admin count and
online-status/username are visible to peers. A locked-out non-primary admin
can only be recovered by delete-and-re-invite; even the primary admin cannot
reset another admin's password, view their credentials, or disable their 2FA
directly. If multi-user is enabled (PART 34), the same privacy boundary
applies one level down: Server Admin can never set, reset directly, or view
a Regular User's password, 2FA secret, or private data — invite/reset-link
only.

### Compatibility / parity requirements

Existing clients for the following services must work against caspaste
unmodified, by only changing the hostname:

| Emulated service | What's supported |
|---|---|
| sprunge / ix | Create paste → plain text URL |
| Termbin (TCP 9999) | Create paste (raw body) → plain text URL |
| Lenpaste | Create, get paste, server info |
| Stikked | Create, get, recent, trending, language list, paginated listing |
| Microbin | Create, list, archive, get, edit, delete paste |
| Hastebin | Create paste, get paste |
| pastebin.com | Create, delete, list, trends, raw get |

### Data model & sensitivity

| Entity | Sensitivity | Notes |
|---|---|---|
| Paste | **Mixed** — users may paste secrets, tokens, PII | Random ID; password-protected pastes hashed; visibility/expiry enforced server-side; body/title lengths configurable maximums |
| User | **PII** — email, display name | Password never stored plaintext; email-verification + password-reset flow |
| Admin | **Privileged** | Same hashing rules as users; separate identity from users; primary admin cannot be deleted |
| API / session tokens | **Secret** | Hashed at rest; plaintext shown ONCE on creation, never retrievable |
| Organization | Public-by-default | Slug + display name; default cap on orgs/user (configurable; admin can override) |
| Custom domain | **Privileged** — per-domain SSL key | Owner type (user/org), DNS verification, cert renewed by scheduler |
| Audit log | **Security telemetry** | Who/what/when/IP for auth, config changes, admin actions |
| Backup | Privileged — full data dump | Encrypted at rest; password never stored |

### Trust boundaries & external services

| Boundary | Trusted? | Failure mode |
|---|---|---|
| HTTP / HTTPS server | Public, hostile | Default-secure (HTTPS, HSTS, CSP, CSRF, rate limits, GeoIP) |
| TCP Termbin listener (port 9999) | Public, plain-text by design | Same rate limiter and visibility rules apply; admin can disable the listener |
| Tor hidden service | Owned by the server binary | Auto-enabled if `tor` on PATH unless explicitly disabled |
| SMTP (outbound) | External | Auto-detect; never blocks startup; queues retries on transient failure |
| GeoIP DB / blocklist / CVE feeds | External fetch, scheduled | Verified checksum; scheduled update; allowlist always overrides denylist |
| Let's Encrypt ACME | External | Renew ahead of expiry via scheduler |
| User-supplied URL fetches (link previews, OG cards) | **NOT supported** | SSRF surface deliberately not built |
| Reverse-proxy headers | Trusted ONLY from configured proxy allowlist | Otherwise treated as user input |
| User-supplied paste / file content | **UNTRUSTED, always** | Rendered as text or sanitized markdown; served with safe MIME / attachment disposition; never executed |
| User-supplied custom-slug / custom-domain | **UNTRUSTED** | Canonicalized, validated against a reserved-name list, collision-checked |

### Abuse cases & required defenses

| Goal | Abuse case | Required defense |
|---|---|---|
| Spam | Mass paste creation, throwaway-account flooding | Per-IP rate limits; optional captcha; admin-configurable registration mode; content-size caps |
| Scraping | Bulk enumeration of unlisted/private pastes | Cryptographically random IDs; visibility enforced server-side on every read; no bulk-listing of private/unlisted content |
| Privilege escalation | Path-namespace abuse (a username/orgname/slug shadowing a reserved route) | Reserved-slug allowlist; case-insensitive collision checks across users and orgs |
| Cross-tenant leak | A user reading another user's/org's data directly | Every user/org-scoped query must include the owner predicate |
| Malicious uploads | Executable HTML/JS, SVG with embedded scripts, MIME spoofing | Extension allowlist; MIME verification; attachment-only serving; content never executed |
| Credential stuffing / brute force | Login flooding | Attempt limit then lockout; generic error messages; optional MFA |
| Password-reset enumeration | Probing which emails are registered | Rate limited; silent response regardless of whether the email exists; single-use, short-lived tokens |
| Session hijacking | Stolen cookie | Secure cookie flags; CSRF tokens on state-changing forms; session rotation on privilege change |
| Custom-domain hijack | TXT-record bypass, ACME challenge spoofing | DNS ownership verification before cert issuance; admin can revoke/suspend; every domain event audited |
| DoS / resource exhaustion | Huge paste, deep burn-after-read counter, decompression bomb | Configurable content-size cap; capped burn-after-read counter; request timeouts and body-size limits |
| Drop-in compat shim abuse | Bypassing CSRF/captcha via legacy shim endpoints (which cannot send tokens) | Same rate limiter, visibility rules, and content cap as native routes apply to every compat shim |
| Admin lockout / takeover | Compromise of primary admin | Primary admin undeletable; recovery flow on MFA compromise; setup token shown once and single-use |
| Audit-log tamper | Attacker editing the audit trail after compromise | Append-only by convention; admin actions on the audit log are themselves audited |
| SSRF via URL shortener | Server fetching an attacker-controlled destination | Destination URL validated on save; the server never fetches it |

Non-goals (explicit non-defenses, with reasoning):

- No SSRF defense layer is needed because the server never initiates
  outbound fetches against user-supplied URLs (no link-preview, no
  remote-image inlining, no URL-shortener destination fetch).
- No ML-based content moderation — admins moderate manually; spam
  mitigation relies on rate limits, optional captcha, and admin delete.

### Security decisions & exceptions

Intentional tradeoffs where caspaste chooses convenience or compatibility
over a maximally locked-down posture:

- **Anonymous paste creation** is allowed and is the core pastebin UX;
  mitigated by rate limits, content-size caps, optional captcha, and an
  admin delete-and-audit flow.
- **Drop-in compatibility shims** intentionally accept POSTs without a CSRF
  token because the legacy clients they emulate don't send one; mitigated by
  applying the same rate limiter, visibility rules, and content cap as
  native routes, and by letting the admin disable any shim individually.
- **Termbin TCP listener (port 9999)** is plain text by design — netcat
  compatibility is the point, and netcat does not speak TLS; the admin can
  disable the listener.
- **Public visibility is the default** for new pastes; mitigated by making
  the visibility selector prominent and making unlisted/private/
  password-protected options first-class.
- **Tor hidden service auto-enabled** when `tor` is on PATH; the admin can
  disable this behavior.
- **Internal `/metrics` endpoint** is not exposed publicly; the app warns on
  startup if it is reachable from the public address.

### Not in scope (explicitly deferred)

- Telemetry beyond opt-in (never on by default).
- Paid tiers or feature gating of any kind.
</content>

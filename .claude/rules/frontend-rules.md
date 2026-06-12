# Frontend Rules (PART 16, 17) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 16, PART 17

## CRITICAL — NEVER DO

- Use React, Vue, Angular, or any client-side framework
- Mount admin panel at `/{admin_path}/` — always `/server/{admin_path}/`
- Mount admin login at `/server/{admin_path}/login` — always `/server/auth/login`
- Hardcode colors — CSS custom properties only
- Break features when JavaScript is disabled — all features work without JS
- Link to or advertise the admin panel path on any public page
- Store admin session in the same cookie as user sessions

## CRITICAL — ALWAYS DO

- Server-side Go templates for all HTML rendering
- Dark theme default; support dark/light/auto via CSS custom properties
- Mobile-first responsive CSS (no fixed pixel widths)
- Content negotiation: browser `text/html` → HTML; curl/CLI `text/plain` or no Accept → plain text
- Admin panel: isolated `src/admin/` package, separate session cookie `caspaste_admin`
- Admin session cookie scoped to `/server/{admin_path}/`
- Auth routes at `/server/auth/login` and `/server/auth/logout` (shared, not under admin prefix)
- Admin setup token logged to stdout, never stored in plaintext

## Web Frontend (PART 16)

- Server-side Go templates (NO React/Vue/Angular)
- All features work without JavaScript (progressive enhancement only)
- Dark mode default; support dark/light/auto via CSS custom properties
- Mobile-first responsive CSS
- No hardcoded colors — CSS variables only
- Content negotiation: browser → HTML, curl/CLI → plain text

## Admin Panel (PART 17) — CRITICAL

- Mount: `/server/{admin_path}/` (UI) — default admin_path = "admin"
- API: `/api/{version}/server/{admin_path}/` (JSON API)
- NEVER mount at /{admin_path}/ (missing /server/ prefix is a violation)
- Separate from public site — isolated package (src/admin/)

## Admin Route Hierarchy

```
/server/admin/                    Dashboard
/server/admin/login               Login page
/server/admin/logout              Logout
/server/admin/{username}/profile  Self-management
/server/admin/config/settings     Server settings
/server/admin/config/ssl          SSL/TLS
/server/admin/config/email        Email/SMTP
/server/admin/config/scheduler    Scheduler
/server/admin/config/logs         Logs
/server/admin/config/backup       Backup
/server/admin/config/info         Server info
/server/admin/config/metrics      Metrics
/server/admin/config/security/tokens  API tokens
/server/admin/config/network/geoip    GeoIP
/server/admin/config/network/tor      Tor
```

## Admin Authentication

- Argon2id password hashing (caspasswd.HashPassword)
- Separate session cookie: caspaste_admin
- Cookie scoped to /server/{admin_path}/
- Session stored in admin_sessions table (SHA-256 hash only)
- Setup wizard: MaybeGenerateSetupToken() logs URL to stdout

## Debug Bypass

- --debug flag bypasses auth (per AI.md PART 6)
- Injects "debug" admin user into context

For complete details, see AI.md PART 16, PART 17

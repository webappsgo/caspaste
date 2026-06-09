# Frontend Rules (PART 16, 17) — Cheatsheet

Full spec: AI.md PART 16, PART 17

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

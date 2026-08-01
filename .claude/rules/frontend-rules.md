# Frontend Rules (PART 16, 17)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- No JS frameworks (React/Vue/Alpine/jQuery), no bundlers, no transpilers, no npm/node for frontend
- No inline CSS or inline JS (`style=`, `onclick=`) — CSP `script-src 'self'` blocks it
- No `alert()`/`confirm()`/`prompt()` — use native `<dialog>` modals and toasts
- No multiple JS files — all JS lives in ONE file: `static/js/app.js`
- Never link to `/server/{admin_path}` from any public route (`/**`) — admin panel is undiscoverable by design
- Never place server-management routes directly under `/server/{admin_path}/` — everything but the admin's own account goes under `/server/{admin_path}/config/*`
- Never let long strings (IPv6, .onion, tokens, hashes, UUIDs) overflow — always `word-break: break-all; overflow-wrap: break-word` or horizontal scroll
- Never leave a list/table/view with blank empty space — every empty state needs icon + title + message (+ CTA when actionable)
- Never use desktop-first CSS (`max-width` media queries) — mobile-first only (`min-width`)
- Never show generic/unstyled error pages — all error pages (400/401/403/404/500/502/503) render through the site theme

## CRITICAL - ALWAYS DO
- ALL HTML via Go `html/template` (`.tmpl` files); sanitize any user/markdown content before `template.HTML`
- HTML5 first → CSS second → JavaScript last resort (JS only when HTML5/CSS truly cannot do it)
- Frontend MUST work with JavaScript disabled for core CRUD (forms + links); JS only enhances
- Frontend routes (`/**`) MUST content-negotiate: browsers get HTML, curl/CLI/empty UA get text, `Accept: application/json` gets JSON
- Submit buttons: disable on click, show loading text ("Saving..."), re-enable on success/error, preserve width
- Every copy button MUST show visible "Copied!" feedback (icon + label, `aria-live="polite"`, revert after 2s)
- WCAG 2.1 AA: keyboard nav, visible focus rings, 4.5:1 contrast, alt text, labeled inputs, `aria-live` errors, skip link, `prefers-reduced-motion` respected
- Theme (dark/light/auto) applies project-wide: web, admin, Swagger, GraphiQL, CLI, TUI, GUI — dark is default; persisted via cookie (guests) or DB (users), server-rendered (no FOUC)
- PWA required: `manifest.json`, service worker, maskable icons, installable, HTTPS-only
- Admin panel is fully isolated: separate auth, separate session, never advertised on public pages
- Server admin credentials live in `users.db` (admins table), never in config file
- MFA (TOTP + Passkeys) support REQUIRED for every server admin, on every project, no exceptions

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| Which templating engine? | Go `html/template`, `.tmpl` extension | Go Templates |
| CSS framework? | None — hand-written CSS, BEM-like naming, CSS vars for theming | CSS Rules |
| Where does app JS live? | `static/js/app.js` only, one file | JavaScript Rules |
| Default theme? | Dark | Themes |
| Toast vs modal for "Saved"? | Toast (non-blocking) | Toast vs Modal |
| Toast vs modal for "Delete this?" | Modal (needs decision) | Toast vs Modal |
| Default admin path? | `/server/admin`, configurable via `server.admin_path` (2-32 chars, `[a-z0-9-]`) | Configurable Admin Path |
| Where do all server-mgmt admin pages live? | Under `/server/{admin_path}/config/*` | Admin Route Hierarchy |
| Where does admin's own profile live? | `/server/{admin_path}/{admin_username}/*` | Admin Route Hierarchy |
| CORS default? | Allow all (`*`), configurable via `web.cors` | CORS Configuration |
| Trailing slash canonical form? | No trailing slash (301 redirect to strip it) | URL Normalization Middleware |
| Breakpoints? | Mobile base, 768px tablet, 1024px desktop, 1280px large (optional) | Breakpoints |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| Vanity URL | Short root-level URL like `/{username}` or `/{org}` mapping to `/users/{username}` etc. (optional feature) |
| Smart content detection | Route auto-detects browser/CLI/Accept header and returns HTML/text/JSON accordingly |
| Server Admin | Administrative account, valid ONLY on `/server/{admin_path}/**`, stored separately from regular users |
| `{admin_path}` | Configurable admin root segment (default `admin`) |
| Primary Admin | First admin created via setup wizard; cannot be deleted except via `--maintenance setup` |
| Toast | Non-blocking, auto-dismissing feedback (top-right, stacked, max 5) |
| Modal | Blocking dialog requiring a decision/input, built on native `<dialog>` |

## QUICK REFERENCE

**Tech stack:** Go templates + vanilla CSS + one vanilla JS file. No frameworks, no bundlers, no TS.

**Standard public pages (all required):** `/server/about`, `/server/privacy`, `/server/contact`, `/server/help`, `/server/terms` — content MUST come from IDEA.md, never generic placeholders.

**Required admin pages (`/server/{admin_path}/config/...`):** settings, branding, ssl, scheduler, email, logs, security/auth, security/tokens, security/ratelimit, security/firewall, security/allowlist, network/tor, network/geoip, network/blocklists, moderation/users, users/invites, backup, maintenance, updates, info, cluster/nodes, cluster/add, plus `/server/{admin_path}/help` and the dashboard/login root.

**Admin layout:** header (logo, search, status dot, admin name, logout) + collapsible grouped sidebar (Server/Security/Network/Users/Cluster/Help) + breadcrumb + main content + footer (version/docs/status).

**Route priority (highest to lowest):** `/api/*` → `/server/{admin_path}/*` → `/server/healthz` → `/static/*` → `/users/*` → `/orgs/*` → reserved names → `/{username}` → `/{org_name}` vanity.

**Middleware order:** URLNormalize → RequestID → PathSecurity → SecurityHeaders → Allowlist → Blocklist → RateLimit → GeoIP → Auth → Logging.

**Responsive widths:** ≥720px = 90% (5% margins); <720px = 98% (1% margins). Container max-width 1400px on desktop.

**Multiple admins:** Primary + additional admins (equal permissions except deletion hierarchy); OIDC/LDAP/SAML group mapping supported; all admin actions audited by username.

---
For complete details, see AI.md PART 16, 17

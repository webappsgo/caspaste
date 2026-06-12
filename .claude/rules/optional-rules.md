# Optional Rules (PART 34, 35, 36) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 34, PART 35, PART 36

## CRITICAL — NEVER DO

- Implement multi-user without reading PART 34 fully first
- Implement organizations without first implementing PART 34 (multi-user is a prerequisite)
- Implement custom domains without reading PART 36 fully first
- Mix up Server Admin accounts (PART 17) with Regular User accounts (PART 34) — they are separate
- Allow admin to set user passwords directly — admin can only issue invite links or send reset
- Allow admin to view user passwords, 2FA secrets, or private data
- Skip TXT record ownership verification for custom domain claims
- Issue Let's Encrypt certs for user domains without DNS-01 TXT verification
- Guess whether a project needs orgs/custom-domains — check IDEA.md or ask the user

## CRITICAL — ALWAYS DO

- Default to admin-only mode (no multi-user) unless IDEA.md requires it
- When multi-user is enabled: use `users` table, routes at `/users/*`, registration modes
- When orgs are enabled: routes at `/orgs/{slug}/*`, org owns resources (not the user)
- Custom domains: DNS-01 TXT verification before SSL cert issuance
- Expose optional feature status in `/server/healthz` `features.*` when the feature is implemented

## PART 34: Multi-User (Optional)

### Decision

| Mode | Default | Use Case |
|------|---------|----------|
| Admin-only | **YES** | Simple APIs (jokes, pastes, etc.) — just Server Admin |
| Multi-user | NO | Apps needing end-user registration, profiles, tokens |

caspaste is a paste service → pastebin-class apps do NOT need multi-user. Confirm with IDEA.md.

### If Implemented: Registration Modes

| Mode | Self-Register | Invite | Admin-Create |
|------|--------------|--------|-------------|
| `open` | ✓ | Optional | Optional |
| `invite` | ✗ | Required | ✗ |
| `admin_only` | ✗ | ✗ | Required |
| `disabled` | ✗ | ✗ | ✗ |

Default when enabled: `open`.

### Admin Permissions (Non-Negotiable)

- Admin CANNOT set user passwords (invite link / reset only)
- Admin CANNOT view passwords, 2FA secrets, private data
- Admin CAN: invite, create (per mode), send reset, suspend, disable 2FA

### Storage & Routes

| Aspect | Server Admin (PART 17) | Regular User (PART 34) |
|--------|------------------------|------------------------|
| Table | `admins` | `users` |
| Routes | `/server/{admin_path}/*` | `/users/*` |
| Required | Always | Optional |

## PART 35: Organizations (Optional)

Requires PART 34 (multi-user) implemented first.

### Decision

Organizations needed when users collaborate as teams with SHARED RESOURCE OWNERSHIP.
Organizations NOT needed for: personal tools, simple APIs, anonymous/individual use, pastebin-class apps.

caspaste (pastebin) → organizations NOT needed.

### If Implemented

- Org routes: `/orgs/{slug}/*` and `/api/{api_version}/orgs/{slug}/*`
- Org owns resources (not the user) — transfer via member add/remove
- Roles: Owner, Admin, Member
- Internal term: `org`/`orgs` in routes and DB (UI label can vary: team, workspace, group)
- Status: `features.organizations: true/false` in `/server/healthz`

## PART 36: Custom Domains (Optional)

### Decision

Custom domains needed when users publish public content under THEIR OWN brand/domain.
NOT needed for: internal tools, private data, API-only services, anonymous content.

caspaste (pastebin, anonymous/individual pastes) → custom domains NOT needed.

### If Implemented

- DNS-01 TXT record ownership verification required BEFORE SSL cert issuance
- SSL: Let's Encrypt DNS-01 per custom domain (see PART 15)
- Scope: user-owned or org-owned domains
- Status: `features.custom_domains: true/false` in `/server/healthz`
- Admin panel: `/server/admin/config/` — manage domain verifications and certs

## Healthz Feature Flags (When Optional Features Are Implemented)

Add to `features.*` in health response:
```go
MultiUser    bool `json:"multi_user"`     // PART 34 implemented
Organizations bool `json:"organizations"` // PART 35 implemented
CustomDomains bool `json:"custom_domains"` // PART 36 implemented
```

Show actual enabled/disabled status — not just whether the code exists.

For complete details, see AI.md PART 34, PART 35, PART 36

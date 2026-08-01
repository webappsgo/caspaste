# Optional Feature Rules (PART 34-36)

⚠️ **These features are OPTIONAL to implement, but if implemented the rules below are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Covers: PART 34 (Multi-User), PART 35 (Organizations), PART 36 (Custom Domains).
Server Admin (PART 17) is always required and is a separate concept from Regular Users (PART 34).

## CRITICAL - NEVER DO
- Never let Server Admin set/view a user's password, 2FA secret, recovery keys, or full email — admin sees only masked email (`j***n@e***.com`)
- Never reveal whether a username/email exists — use generic errors ("Invalid credentials", "If an account exists...") for login/reset/registration collisions
- Never return 403 for a private profile lookup by a non-owner — return 404 (don't leak existence)
- Never expose `admin_path` in `/api/autodiscover` or any public response
- Never allow plaintext LDAP (`tls_mode: none`) except when host resolves to localhost/127.0.0.1
- Never disable TLS certificate verification (`tls_verify`) in production
- Never accept unsigned/mis-signed SAML assertions, or IdP-initiated SAML login unless `allow_idp_initiated: true`
- Never use SVG for user-uploaded or external avatar URLs (raster only — user SVG is active browser content)
- Never let a username and org slug collide — they share one global namespace
- Never allow custom-domain SSL private keys or DNS provider credentials to be stored unencrypted
- Never treat a routing DNS record (CNAME/A/AAAA) as ownership proof — only the `_verify.{domain}` TXT record proves ownership
- Never silently mutate/auto-append digits to a user-chosen or OIDC/LDAP/SAML-derived username without showing the final result first
- Never confuse `server.orgs.creation.mode` (server-level policy) with a per-org `visibility` setting — they are independent
- Never log a one-shot token (reset/verify/invite/tracking) value itself — only log issuance/consumption events

## CRITICAL - ALWAYS DO
- Always require re-entering the current password before 2FA/passkey setup or disable
- Always generate 10 recovery keys (format `{8-hex}-{4-hex}`), show once, hash with SHA-256, invalidate all on regeneration
- Always invalidate ALL existing sessions after a password reset
- Always apply the username blocklist + reserved-route-collision check to every new local, invited, or OIDC/LDAP/SAML-derived username
- Always run first-login username confirmation for every NEW external (OIDC/LDAP/SAML) account, not just conflicts
- Always use PKCE (S256) + `state` + `nonce` for every OIDC Authorization Code flow
- Always validate SAML assertions for signature, `NotBefore`/`NotOnOrAfter`, `AudienceRestriction`, `Recipient`, and replay (`ID`/`InResponseTo`)
- Always require both a Server Admin invite (or pre-provisioned account) AND external group match (`admin_groups`) before granting admin via OIDC/LDAP/SAML — never infer admin from generic user role mapping
- Always audit-log all admin actions on user accounts (password reset trigger, 2FA disable, suspend) with a required reason
- Always require Owner role (not just Admin) for org billing changes, ownership transfer, and org deletion
- Always verify custom-domain ownership via TXT record before activating routing/SSL
- Always encrypt DNS-01 provider credentials and issued cert/key PEM at rest
- Always re-check group/role membership on every OIDC/LDAP/SAML login (not cached) unless documented otherwise

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| Default multi-user mode | Disabled (admin-only) | PART 34 Overview |
| Default registration mode when enabled | `open` | PART 34 Registration Modes |
| Who can generate user invites | Only Server Admin | PART 34 Invite Rules |
| Default invite expiry | 7 days (configurable) | PART 34 Invite Rules |
| Default org creation mode | `open` (any authenticated user) | PART 35 Organization Creation Modes |
| Org/team terminology | Canonical internal term is "organization"; UI may say team/workspace/group | PART 35 Organization vs Just Groups |
| Username/org-slug namespace | Shared — one collides with the other | PART 35 Shared Namespace |
| Default profile visibility | `public` | PART 34/35 Profile Fields |
| Default avatar source | Gravatar (email hash) | PART 34 Avatar Settings |
| Custom domains default enabled | `false` | PART 36 Feature Configuration |
| Ownership verification method | DNS TXT record at `_verify.{domain}` | PART 36 Verification Flow |
| SSL challenge preference order | tls-alpn-01 (if available) → http-01 → dns-01 (required for wildcards) | PART 36 Challenge Selection Logic |
| Domain suspension authority | Server Admin only | PART 36 Admin Controls |
| Recovery with no username/email known | Impossible by design (no recovery) | PART 34 Account Recovery Matrix |
| Server Admin's own recovery | `{project_name} --maintenance setup` (console access) | PART 34 Server Admin Recovery |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| Server Admin | Administrative account for managing the server (PART 17); always required, separate from Regular Users |
| Regular User | End-user account (PART 34); optional, `users` table |
| Organization (org) | Shared-ownership entity for teams (PART 35); may be labeled "team"/"workspace" in UI |
| `external_id` | Stable OIDC subject / LDAP unique ID / SAML persistent NameID used to match an external identity, never mutable username/email |
| `source` | Identity origin field: `local`, `oidc:{provider}`, `ldap:{provider}`, `saml:{provider}` |
| Clustering | Multiple app nodes sharing state (app's responsibility) |
| High Availability (HA) | Database backend fault tolerance (infrastructure's responsibility, NOT the app's) |
| Apex domain | A bare domain with no subdomain (e.g. `example.com`) |
| TXT verification | Ownership proof method for custom domains, distinct from routing (CNAME/A/AAAA) |

## QUICK REFERENCE

**Registration modes (PART 34):** `open` (anyone, default) · `invite` (admin-issued link only) · `admin_only` (admin creates record, user sets password) · `disabled` (no new accounts).

**Org creation modes (PART 35):** `open` (any authenticated user, default) · `invite` · `admin_only` · `disabled`.

**Org roles:** Owner (full control, billing, delete/transfer) > Admin (manage members/settings/tokens) > Member (view/use).

**Profile visibility:** `public` (default) vs `private` (404 to non-owners; org members may still see username/avatar/display_name if `org_visibility: true`).

**Auth methods supported for both users and admins:** Password, TOTP 2FA, Passkeys/WebAuthn, OIDC, LDAP, SAML — each MUST support multiple named providers, manageable at `/server/{admin_path}/config/security/auth/*`.

**Token prefixes:** `adm_` admin · `usr_` user · `org_` org · `*_agt_` agent variants.

**Custom domain lifecycle:** add domain → publish `_verify.{domain}` TXT → trigger verify → (on success) status `active` → issue SSL (auto or DNS-01) → scheduled renewal (7 days before expiry) and periodic re-verification (every 15 min while pending).

**Custom domain SSL challenges:** HTTP-01 (port 80) · TLS-ALPN-01 (port 443, proxy-friendly, preferred) · DNS-01 (required for wildcards, needs provider credentials).

**Custom domain error codes:** `DOMAIN_EXISTS` 409 · `DOMAIN_RESERVED`/`DOMAIN_LIMIT`/`DOMAIN_INVALID`/`DOMAIN_NOT_VERIFIED` 400 · `DOMAIN_NOT_FOUND` 404 · `DOMAIN_SUSPENDED` 403 · `SSL_ISSUANCE_FAILED` 500.

**Databases:** `server.db` (admin/server state) and `users.db` (regular users, orgs, custom domains) — kept separate for isolation and independent backup/restore.

---
For complete details, see AI.md PART 34-36

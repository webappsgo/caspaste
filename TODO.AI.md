# CasPaste - AI.md Compliance Tasks

## Status: TEMPLATE UPDATED (2026-02-04)

AI.md replaced with new TEMPLATE.md from ~/Projects/github/apimgr/TEMPLATE.md.

Placeholders replaced:
- `{projectname}` → `caspaste`
- `{PROJECTNAME}` → `CASPASTE`
- `{projectorg}` → `casjay-forks`
- `{PROJECTORG}` → `CASJAY-FORKS`

`.claude/rules/` directory created with all 14 rule files per PART 0.

---

## Critical Rules Committed to Memory

### NEVER Rules

- NEVER guess or assume - ALWAYS ask when uncertain
- NEVER install Go locally - ALL builds use Docker (make dev/local/build)
- NEVER store plaintext passwords (use Argon2id)
- NEVER use inline comments (comments ABOVE code only)
- NEVER modify AI.md PARTS 0-36 (except OPTIONAL→REQUIRED)
- NEVER run git add/commit/push (write .git/COMMIT_MESS instead)
- NEVER put Dockerfile in root (use docker/Dockerfile)
- NEVER use CGO (CGO_ENABLED=0 always)
- NEVER use strconv.ParseBool() (use config.ParseBool() with 40+ variants)
- NEVER use mattn/go-sqlite3 (use modernc.org/sqlite)
- NEVER use bcrypt for new passwords (use Argon2id, bcrypt only to verify legacy)
- NEVER use GPL/AGPL/LGPL dependencies (MIT/Apache/BSD only)
- NEVER leave TODO comments in code - implement fully or don't implement
- NEVER create stub functions or placeholders
- NEVER include AI attribution in code/commits/PRs/documentation
- NEVER use "I think" or "probably" - KNOW from spec or ASK
- NEVER hardcode projectname/projectorg - infer from git remote or path
- NEVER use Makefile in CI/CD (explicit commands with env vars)
- NEVER use plural directory names (handler/, not handlers/)
- NEVER use .yaml extension (use .yml)

### MUST Rules

- MUST use parameterized SQL queries
- MUST use Argon2id for passwords, SHA-256 for tokens
- MUST re-read spec before implementing ANY feature
- MUST write `.git/COMMIT_MESS` file (AI cannot run git commit)
- MUST have comments ABOVE code, never inline
- MUST use config.ParseBool() for ALL boolean parsing (40+ variants)
- MUST normalize and validate ALL paths (security requirement)
- MUST use MIT License with embedded 3rd party attributions
- MUST build for 8 platforms (linux/darwin/windows/freebsd × amd64/arm64)
- MUST support all 4 OSes (Linux, macOS, Windows, BSD)
- MUST use singular directory names (handler/, model/, service/)
- MUST use .yml extension (NEVER .yaml)
- MUST implement PathSecurityMiddleware as FIRST in middleware chain
- MUST use crypto/rand for all secret/salt generation
- MUST use context.WithTimeout for all database queries

### COMMIT Rules

- COMMIT message format: `{emoji} Title (max 64 chars) {emoji}\n\n{description}\n\n- Bullet points`
- COMMIT to `.git/COMMIT_MESS` file, user runs `git commit -F .git/COMMIT_MESS`
- COMMIT_MESS must match actual `git status` - verify before writing
- COMMIT emoji reference: ✨ feat, 🐛 fix, 📝 docs, ♻️ refactor, ✅ test, 🔧 chore

### KEY DECISIONS (pre-answered)

| Question | Answer | Reference |
|----------|--------|-----------|
| What password hash? | Argon2id (NEVER bcrypt) | PART 11 |
| Where is Dockerfile? | `docker/Dockerfile` (NEVER root) | PART 27 |
| CGO enabled? | NEVER (CGO_ENABLED=0 always) | PART 7 |
| Premium features? | NEVER (all features free) | PART 1 |
| External cron? | NEVER (built-in scheduler) | PART 19 |
| Client-side rendering? | NEVER (server-side Go templates) | PART 16 |
| SQLite driver? | modernc.org/sqlite (NEVER mattn) | PART 3 |
| Boolean parsing? | config.ParseBool() (NEVER strconv) | PART 5 |

---

## Current Compliance Status

### Implemented Core Packages
- `src/user/` - User management (Argon2id, crypto/rand)
- `src/session/` - Session management (SHA-256, query timeouts)
- `src/token/` - API tokens (SHA-256, query timeouts)
- `src/org/` - Organizations (query timeouts)
- `src/domain/` - Custom domains (query timeouts)
- `src/storage/` - Database abstraction (SQLite, PostgreSQL, MySQL)
- `src/config/` - Configuration management
- `src/web/` - Web UI handlers and templates
- `src/apiv1/` - REST API v1 handlers
- `src/server/` - Main server entry point
- `src/client/` - CLI client entry point

### Pending Integration
- API route integration for user/org/domain
- Web UI routes for user/org/domain management
- Database schema migration for new tables
- Full audit against new AI.md template

---

## Next Steps

1. Read IDEA.md to verify project-specific features
2. Audit codebase against new AI.md template
3. Complete PART 34/35/36 integration if needed

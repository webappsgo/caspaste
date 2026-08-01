# Project Structure Rules (PART 2, 3, 4)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- NEVER use GPL/AGPL/LGPL licensed dependencies (copyleft forces derivative works open-source)
- NEVER hardcode `{project_name}` or `{project_org}` — always infer from git remote or path
- NEVER put `LABEL` for license in Dockerfile — license is an OCI annotation at build time
- NEVER use `github.com/mattn/go-sqlite3` (CGO) — use `modernc.org/sqlite`
- NEVER use `github.com/lib/pq`, `ooni/go-libtor`, `dgrijalva/jwt-go`, `gorilla/mux`, `go-redis/redis` (old path) — see forbidden libs table
- NEVER assume current working directory is project root — all paths relative to project root
- NEVER mix runtime-directory purposes (config vs data vs log vs backup)
- NEVER store plaintext passwords anywhere, including config files; never log passwords even hashed
- NEVER allow leading/trailing whitespace in passwords (reject, don't trim)
- NEVER hardcode a specific Go version in docs/examples/Docker/CI
- NEVER commit `binaries/`, `releases/`, `volumes/`, AI config dirs (`.claude/`, `.cursor/`, `.aider/`, `.ai/`, `.windsurf/`), or `CLAUDE.local.md`
- NEVER use `/data/**` or `/config/**` paths outside Docker — those are container-only

## CRITICAL - ALWAYS DO
- ALWAYS include `LICENSE.md` (MIT) in project root with embedded third-party licenses appended
- ALWAYS include a license badge in README.md: `[![License](https://img.shields.io/github/license/{project_org}/{project_name})](LICENSE.md)`
- ALWAYS pass `--annotation "org.opencontainers.image.licenses=MIT"` to `docker buildx build`
- ALWAYS update LICENSE.md when a dependency is added/removed/upgraded
- ALWAYS support all 4 OSes (Linux, BSD, macOS, Windows) and both AMD64/ARM64
- ALWAYS use latest stable Go; build with `casjaysdev/go:latest` (unpinned); use Makefile targets, never run `go` directly
- ALWAYS use pure-Go libraries compatible with `CGO_ENABLED=0`
- ALWAYS hash passwords with Argon2id (OWASP 2023 params: time=3, memory=64MB, threads=4, keyLen=32, saltLen=16); bcrypt only to verify+rehash legacy
- ALWAYS hash API/session tokens with SHA-256 (fast lookup, tokens already high-entropy)
- ALWAYS start `.gitignore` with `# gitignore created on MM/DD/YY at HH:MM` then literal `ignoredirmessage`
- ALWAYS keep `docker/rootfs/` committed (it's the build-time overlay), and `src/`, `go.mod`/`go.sum`, `docker/` out of `.dockerignore`
- ALWAYS use `server.yml` as the config filename (never `.yaml`)

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| What license? | MIT, `LICENSE.md` in root | PART 2 |
| How to attribute deps? | Compact table (10+ deps) or full text (<10 deps / legally required) | PART 2 |
| SQLite driver? | `modernc.org/sqlite` (pure Go) | PART 3 |
| libSQL driver? | `github.com/tursodatabase/libsql-client-go` (remote-only, Turso/sqld) | PART 3 |
| Router? | `github.com/go-chi/chi/v5` | PART 3 |
| Password hashing? | Argon2id (new), bcrypt only to verify+rehash | PART 3 |
| Token hashing? | SHA-256 | PART 3 |
| Where do file paths resolve from? | Always project root, never CWD | PART 3 |
| Config file name? | `server.yml` (all OSes) | PART 4 |
| Where do config/data/log/backup files go? | `{config_dir}`/`{data_dir}`/`{log_dir}`/`{backup_dir}` per OS table | PART 3, 4 |
| Docker container paths? | `/config/{project_name}/`, `/data/{project_name}/` (container-only) | PART 4 |
| Docker internal port? | `80` | PART 4 |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| `{project_name}` | Project name, may change on rename; lowercase |
| `{PROJECT_NAME}` | UPPERCASE render, env vars/Makefile |
| `{project_org}` | Org/username, lowercase |
| `{internal_name}` | Frozen forever after first setup; used for all on-disk identifiers (paths, service unit, plist) |
| `{internal_org}` | Org identifier used in OS paths (frozen alongside internal_name usage) |
| `{plist_name}` | macOS bundle ID: `io.github.{project_org}.{internal_name}` |
| `local` provider | Prototyping/bootstrapping location; no VCS/registry required |
| `{}` | Anything inside braces is a variable; anything outside is literal |

## QUICK REFERENCE
- **License**: MIT only; no copyleft deps; embed third-party licenses; CI license-check job scans for GPL/AGPL/LGPL.
- **Repo root layout**: `.github|.gitea/workflows/`, `.claude/rules/*.md` (this file lives here), `docs/` (MkDocs only), `src/`, `scripts/`, `tests/` (integration scripts), `docker/` (incl. committed `rootfs/`), `volumes/` (gitignored), `binaries/`/`releases/` (gitignored), `README.md`, `LICENSE.md`, `AI.md`, `TODO.AI.md`/`TODO.md`, `PLAN.AI.md`/`PLAN.md`, `Jenkinsfile`, `release.txt`.
- **Path rule**: everything relative to `git rev-parse --show-toplevel`, never `$PWD`.
- **Runtime dirs**: config = user-editable, data = app-managed, log = logs, backup = archives — never mix.
- **Platforms**: Linux, BSD, macOS, Windows × AMD64/ARM64 all required.
- **Go**: latest stable only, build-only (static binary), `casjaysdev/go:latest` for build/CI, no CGO.
- **DB driver aliases**: `sqlite`/`sqlite2`/`sqlite3` → `modernc.org/sqlite`; `libsql`/`turso` → libsql-client-go; `postgres`/`pgsql`/`postgresql` → pgx; `mysql`/`mariadb` → go-sql-driver/mysql; `mssql` → go-mssqldb; `mongodb`/`mongo` → mongo-driver.
- **Auth libs required even without end-users** (admin auth): otp (TOTP), go-webauthn (passkeys), jwt/v5, go-oidc/v3, oauth2, go-ldap/v3, crewjam/saml, gorilla/sessions.
- **OS paths pattern**: privileged uses system dirs (`/etc`, `/var/lib`, `/var/log` on Linux; `/Library/...` on macOS; `%ProgramData%` on Windows); non-privileged uses user dirs (`~/.config`, `~/.local/share` on Linux; `~/Library/...` on macOS; `%AppData%`/`%LocalAppData%` on Windows). Docker always uses `/config` and `/data`.

---
For complete details, see AI.md PART 2, 3, 4

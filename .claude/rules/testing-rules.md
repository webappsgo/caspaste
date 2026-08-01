# Testing & Documentation Rules (PART 29, 30, 31)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never build, test, or run binaries directly on the local machine — no Go installed, ALWAYS use Docker/Incus
- Never use project directory for test/runtime data — ALL runtime/test data goes to `/tmp/{project_org}/{internal_name}-XXXXXX/`
- Never use `docker-compose.yml` or `docker-compose.dev.yml` as AI — human-only; AI uses `docker-compose.test.yml` (prefer `tests/` scripts)
- Never mount `./volumes/`, `./docker/rootfs/`, or any project-directory path as a runtime volume
- Never bypass admin authentication in tests, beta testing, or CI — debug mode adds verbosity only, never skips auth
- Never accept "tested manually" / "obvious code" / "just a getter" as a reason to skip unit tests
- Never use `pkill -f`, `killall`, `kill -9` first, or any Docker "prune"/`rm -f $(docker ... -q)` bulk command
- Never put non-ReadTheDocs files in `docs/` — it is ONLY for MkDocs documentation
- Never hardcode a user-facing string — every string (web, admin, API, Swagger/GraphQL, email, CLI/agent output) MUST use a translation key
- Never let an unsupported `--lang`/`Accept-Language`/`?lang=` value error or crash — silently fall back to `en`
- Never convey information by color alone (accessibility)

## CRITICAL - ALWAYS DO
- Always create/update the matching `*_test.go` in the same work pass when adding or changing package logic
- Always run `make test` (Phase 1 — Toolchain Gate, `go test -cover`) before every commit; must be ≥60% coverage
- Always achieve 100% endpoint/route coverage in `./tests/*.sh` (Phase 2 — Binary Validation, manual/developer-initiated)
- Always test critical paths (auth, DB, token validation) and all reachable error paths, regardless of overall %
- Always test every route with ALL applicable Accept headers: frontend = `text/html` + `text/plain`; API = `application/json` + `text/plain`; plus every `.txt` endpoint
- Always maintain `tests/run_tests.sh`, `tests/docker.sh`, `tests/incus.sh` (executable, `trap` cleanup, exit 0/nonzero)
- Always identify exact PID/container name before killing/removing; kill gracefully (SIGTERM) before `-9`
- Always keep every language file's keys identical to `en.json` (build-time validated via `make i18n-validate`)
- Always support the fallback chain `?lang= → lang cookie → Accept-Language → en` on the server, and `--lang → config → LANG/LC_ALL → auto-detect → en` on CLI/agent
- Always ship RTL support for Arabic (`dir` attribute from `meta.direction`, logical CSS properties)
- Always include skip links, ARIA live regions, focus management, and 4.5:1 contrast (WCAG 2.1 AA)

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| Unit test coverage gate? | ≥60% via `go test -cover`, enforced in CI | PART 29 |
| Endpoint/route coverage gate? | 100% — every endpoint and admin route | PART 29 |
| Where do `*_test.go` tests live vs `./tests/*.sh`? | `*_test.go` = package logic/pure functions (no server needed); `./tests/*.sh` = full running binary, HTTP, auth, systemd | PART 29 |
| Preferred integration test runtime? | Incus (Debian + systemd); Docker (`casjaysdev/go:latest`/`alpine:latest`) is fallback | PART 29 |
| Temp dir pattern for tests? | `/tmp/{project_org}/{internal_name}-XXXXXX/` — never bare `/tmp` | PART 29 |
| Docs engine/host? | MkDocs Material, hosted on ReadTheDocs | PART 30 |
| Required docs pages? | `index`, `installation`, `configuration`, `api`, `admin`, `security`, `integrations`, `development`, `cli` (if applicable) | PART 30 |
| Default/fallback language? | English (`en`) | PART 31 |
| Supported languages? | en, es, zh, fr, ar, de, ja — all binaries (server, CLI, agent) support all of them | PART 31 |
| Language selection mechanism? | `?lang=` query param (sets cookie), no URL path prefixes | PART 31 |
| Translation file location? | `src/common/i18n/locales/{lang}.json`, embedded via `go:embed`, shared by all binaries | PART 31 |
| Accessibility standard? | WCAG 2.1 AA, mandatory keyboard nav + screen reader support | PART 31 |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| Phase 1 — Toolchain Gate | `*_test.go` run via `make test` / `go test -cover`; pre-commit, fast, no server needed |
| Phase 2 — Binary Validation | `./tests/*.sh` scripts; full running binary, endpoints, auth, systemd — manual/developer-initiated |
| `./tests/*` | Executable shell scripts in repo-root `tests/` dir (never `*_test.go`) |
| Setup token | First-run token used to create the admin account and prove auth is enforced in tests |
| CLDR plural categories | `zero, one, two, few, many, other` — language-specific subset used for pluralization |
| `sr-only` | CSS class visually hiding text while keeping it available to screen readers |

## QUICK REFERENCE
**Test scripts required:** `tests/run_tests.sh` (auto-detect), `tests/docker.sh` (Alpine), `tests/incus.sh` (Debian+systemd) — build via `make build`/Docker, install `curl bash file jq`, test version/help/binary-rename/admin-setup/CLI/agent/endpoints, cleanup via `trap`.

**Coverage exceptions rejected:** "simple getter", "obvious code", "internal only", "tested manually", "just logging", "third-party code" — none exempt from the 60% unit-test floor or the 100% endpoint gate.

**Docs required files:** `mkdocs.yml`, `.readthedocs.yaml`, `docs/requirements.txt`, `docs/stylesheets/{dark,light}.css` (optional), the 8-9 required `docs/*.md` pages above.

**i18n coverage:** web frontend, admin panel, API responses, Swagger/GraphQL, email templates, server/CLI/agent CLI output, health page, cookie consent, privacy/terms — if a human reads it, it's translated.

**a11y checklist:** skip links first focusable element · ARIA live regions for dynamic content · modal focus trap + return focus · 44x44px touch targets · 4.5:1 text contrast · `axe`/WAVE/Lighthouse/NVDA/VoiceOver + keyboard-only testing.

---
For complete details, see AI.md PART 29, 30, 31

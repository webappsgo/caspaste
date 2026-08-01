# Makefile Rules (PART 26)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- NEVER add targets beyond the six core ones: `dev`, `local`, `build`, `test`, `release`, `docker`
- NEVER use the Makefile for CI/CD — it is local dev only; pushing images, creating stable/beta/daily
  automated releases, and tagging is CI/CD's job (GitHub Actions / Gitea Actions / GitLab CI)
- NEVER push a Docker image from `make docker` — build only, no push (pushing is CI/CD's responsibility)
- NEVER hardcode `PROJECT_NAME`/`PROJECT_ORG` — always infer from git remote or directory path
- NEVER copy or symlink binaries out of `binaries/` (no `cp`/`ln -s` to `/usr/local/bin`, PATH, `/tmp`, etc.)
- NEVER build on the host — all compiles/tests run in Docker (`casjaysdev/go:latest`)
- NEVER add `v` prefix to text/timestamp versions (`vdev`, `vbeta`, `v20251218` are all wrong)
- NEVER skip `clean` before `build`/`local` — both targets depend on `clean` running first

## CRITICAL - ALWAYS DO
- ALWAYS embed `Version`, `CommitID`, `BuildDate`, `OfficialSite` via `-ldflags` in `build`/`local`/`release`
- ALWAYS use `CGO_ENABLED=0` and `GOFLAGS=-buildvcs=false` for Docker builds (static binaries)
- ALWAYS read version from `release.txt` unless `VERSION` env var overrides it
- ALWAYS output `make dev` builds to an isolated temp dir: `${TMPDIR}/${PROJECT_ORG}/${PROJECT_NAME}-XXXXXX/`
- ALWAYS run `make test` before every commit (Phase 1 — Toolchain Gate); enforce >= 60% coverage
- ALWAYS strip binaries before copying into `releases/` (`make release`)
- ALWAYS build all 8 platforms for `make build`/`make release`: linux, darwin, windows, freebsd × amd64/arm64
- ALWAYS cache Go modules/build cache across builds (`GO_CACHE`, `GO_BUILD`, scoped per project)

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| How many Makefile targets? | Exactly 6: dev, local, build, test, release, docker | PART 26 opening line |
| Does Makefile push Docker images? | No — build only; push is CI/CD | `make docker` section |
| Does Makefile create GitHub releases automatically? | Only `make release` (manual, local, stable only) | Target Details → `make release` |
| Where do dev builds go? | Random temp dir, not `binaries/` | `make dev` section |
| Where do prod-test/release builds go? | `binaries/` (with version embedded) | `make local`/`make build` |
| Version source of truth? | `release.txt`, overridden by `VERSION` env var | Version Priority |
| Can binaries be symlinked to PATH? | Never | NEVER Copy or Symlink Binaries |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| `dev` | Fast local build to temp dir, no ldflags, for active coding |
| `local` | Production-equivalent build to `binaries/`, local platform only, versioned |
| `build` | Full 8-platform release build to `binaries/` |
| `release` | `build` + package to `releases/` + `gh release create` (manual, local, stable only) |
| `docker` | Multi-arch `docker buildx build`, no push |
| `test` | `go test` with coverage, run inside Docker, >= 60% enforced |

## QUICK REFERENCE
| Target | Output | Runs `clean` first? | ldflags? | Use case |
|--------|--------|---------------------|----------|----------|
| `dev` | `${TMPDIR}/${PROJECT_ORG}/${PROJECT_NAME}-XXXXXX/` | No | No | Active coding/debugging |
| `local` | `binaries/` | Yes | Yes | Test prod build locally |
| `build` | `binaries/` (all 8 platforms) | Yes | Yes | Before release |
| `test` | Coverage report | N/A | N/A | After code changes, pre-commit |
| `release` | `releases/` (runs `build` first) | via `build` | Yes | Manual local release only |
| `docker` | buildx cache, tagged image (no push) | No | N/A | Container build verification |

**Local dev workflow order:** `make dev` → `make test` → `./tests/run_tests.sh` → `make local` →
`./tests/incus.sh` → `make build`. Automated stable/beta/daily releases and image pushes always
happen in CI/CD, never in the Makefile.

---
For complete details, see AI.md PART 26

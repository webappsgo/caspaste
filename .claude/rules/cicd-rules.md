# CI/CD Rules (PART 28)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never use Makefile targets in CI/CD workflows — commands must be explicit (visibility)
- Never reference local user paths (`~/.local/share/go`) in CI — use `/tmp/` or CI-native caching
- Never depend on local Docker containers for builds — GitHub/Gitea Actions use native containers
- Never cross-cancel different release refs — only the exact same branch/tag ref may auto-cancel
- Never install tools inline in `ci.yml` jobs — all jobs run in `casjaysdev/go:latest`; no `ensure-build-image` gate, no `build-toolchain.yml`
- Never use `default_branch` for the secret-scan commit range — after a push it equals HEAD and silently skips the scan
- Never omit VERSION/COMMIT_ID/BUILD_DATE — always set explicitly in a "Set build info" step, never as static `env:`

## CRITICAL - ALWAYS DO
- Use explicit `go build` commands with all flags visible in CI
- Use CI-native caching (not local host cache paths)
- Build all 8 platforms in the release/beta/daily matrix (linux/darwin/windows/freebsd × amd64/arm64)
- Auto-cancel older in-progress runs via `concurrency:` for pushes to `main`/`master`/`devel`/`dev`/`beta`
- Tag-release workflows (`release.yml`) use `concurrency` scoped to the exact tag ref only
- Pin every third-party Action to a full-length commit SHA with a `# vX.Y.Z` trailing comment (e.g. `actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0  # v7.0.0`) — never a mutable tag/branch
- Run `secret-scan`, `workflow-policy`, `vuln-scan`, `image-scan` on push, PR, and weekly cron (`0 6 * * 1`) inside `ci.yml`
- Use `github.event.before`/`github.event.after` (or provider equivalent) for secret-scan commit range
- Build CLI binaries (`-cli` suffix) only `if: hashFiles('src/client/') != ''`; Agent binaries only `if: hashFiles('src/agent/') != ''`

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| Which container image runs CI jobs? | `casjaysdev/go:latest` for every job | CI Workflow (GitHub Actions) |
| Coverage threshold? | 60% minimum, enforced via `go tool cover -func` | CI Workflow, `test` job |
| Where do security jobs live? | Inside `ci.yml`, not a separate file | Note after CI Workflow |
| How to pin third-party Actions? | Full commit SHA + `# vX.Y.Z` comment | All workflow examples |
| Docker image variants? | Standard (`docker/Dockerfile`, alpine, app only) and All-in-One (`docker/Dockerfile.aio`, debian, app+PostgreSQL+Valkey+Tor, `-aio` tag suffix) | Docker Workflow § Image Types |
| Container registry (GitHub)? | `ghcr.io` | Docker Workflow |
| Container registry (Gitea/Forgejo)? | Auto-detected from `server_url` (self-hosted safe) | Docker Workflow (Gitea/Forgejo) |
| Daily build tag? | Fixed tag `daily`; previous release deleted first | Daily Workflow |
| GitLab CI file layout? | Single `.gitlab-ci.yml` with stages, not separate workflow files | GitLab CI section |
| Jenkins credential IDs? | `github-token`, `gitea-token`, `forgejo-token`, `gitlab-token`, `dockerhub-token` | Jenkins Configuration |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| Standard image | `docker/Dockerfile`, alpine base, app only, no suffix |
| All-in-One (AIO) image | `docker/Dockerfile.aio`, debian base, app+PostgreSQL+Valkey+Tor, `-aio` tag suffix |
| `{commit_id}` | Short SHA (7 chars) from `git rev-parse --short HEAD` |
| `YYMM` | Year+month tag component, e.g. `2512` |
| Auto-cancel (branch push) | `concurrency` cancels older runs for the same branch ref |
| Auto-cancel (tag release) | `concurrency` cancels older runs for the exact same tag ref only |

## QUICK REFERENCE

**GitHub Actions** (`.github/workflows/`):
| File | Trigger | Purpose |
|------|---------|---------|
| `ci.yml` | Push/PR to default branch + weekly cron for security jobs | Build, test, lint, coverage, secret/image scan, workflow-policy |
| `release.yml` | Tag push (`v*`, `*.*.*`) | Production releases, 8-platform matrix |
| `beta.yml` | Push to `beta` | Beta prerelease |
| `daily.yml` | 3am UTC cron + push to main/master | Daily prerelease (tag `daily`) |
| `docker.yml` | Any branch push + version tags | Standard + AIO Docker images to `ghcr.io` |

**Gitea/Forgejo Actions**: same 5 files, same triggers, in `.gitea/workflows/` (or `.forgejo/workflows/`); `$GITEA_*`/`${{ gitea.* }}` replace `$GITHUB_*`/`${{ github.* }}`; registry auto-detected from server URL.

**GitLab CI**: single `.gitlab-ci.yml`, stages `build → test → package → release → docker`; `casjaysdev/go:latest` image; `$CI_COMMIT_TAG`/`$CI_COMMIT_SHORT_SHA` replace GitHub context vars; GitLab `release:` block replaces `softprops/action-gh-release`.

**Jenkins**: `Jenkinsfile` with `BUILD_TYPE` (`release`/`beta`/`daily`) driving the same release matrix; agent labels `amd64` and `arm64` required; provider block (GitHub/Gitea/Forgejo/GitLab) selected via credentials.

**Provider CI locations:**
| Provider | Config | Self-Hosted |
|----------|--------|-------------|
| GitHub | `.github/workflows/*.yml` | No |
| Gitea | `.gitea/workflows/*.yml` | Yes |
| Forgejo | `.forgejo/workflows/*.yml` (or `.gitea/workflows/`) | Yes |
| GitLab | `.gitlab-ci.yml` | Yes |

---
For complete details, see AI.md PART 28

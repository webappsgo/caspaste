# CI/CD Rules (PART 28) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 28

## CRITICAL — NEVER DO

- Use Makefile targets in CI/CD (explicit commands only)
- Reference host paths (e.g. `~/.local/share/go`) — use named volume in Docker
- Pin Actions with tags — always use full 40-char commit SHA
- Commit ci.yml or release.yml before the `:build` toolchain image exists in the registry
- Run `apk add`, `go install`, or any package install in a CI job step — all tools live in the `:build` image
- Use cancel-in-progress: true on build-toolchain.yml (never interrupt)

## CRITICAL — ALWAYS DO

- Gate ALL jobs on `needs: ensure-build-image` (every workflow except build-toolchain.yml)
- Run every job inside `container: image: ${{ needs.ensure-build-image.outputs.image }}`
- Bootstrap order: Dockerfile.build + build-toolchain.yml first → trigger workflow_dispatch → verify `:build` exists → then ci.yml/release.yml
- Use exact ldflags: `-X 'main.Version=...' -X 'main.CommitID=...' -X 'main.BuildDate=...' -X 'main.OfficialSite=...'`
- Set CGO_ENABLED=0 in every build job

## Workflow Files Required

| File | Trigger | Status |
|------|---------|--------|
| ci.yml | Push/PR to main/master | ✓ Added |
| release.yml | Tag push (v*, X.Y.Z) | ✓ Updated |
| beta.yml | Push to beta branch | ✓ Updated |
| daily.yml | 3am UTC + push to main | ✓ Updated |
| docker.yml | Tags + main/master/beta push | ✓ Updated |
| build-toolchain.yml | 1st of month + workflow_dispatch | ✓ Added |

## Bootstrap Order (CRITICAL)

1. Commit docker/Dockerfile.build + build-toolchain.yml FIRST
2. Trigger build-toolchain.yml via workflow_dispatch
3. Verify :build image exists in ghcr.io
4. THEN commit ci.yml and release.yml

Until step 2-3, ci.yml will fail with "Build image not found".

## CI Pattern — ensure-build-image Gate

ALL jobs in ci.yml, release.yml, beta.yml, daily.yml MUST:
- `needs: ensure-build-image`
- Run in container: `${{ needs.ensure-build-image.outputs.image }}`

## Build Info Variables

```bash
# Set in "Set build info" step:
if [ -f release.txt ]; then echo "VERSION=$(cat release.txt)" >> $GITHUB_ENV; fi
echo "COMMIT_ID=$(git rev-parse --short HEAD)" >> $GITHUB_ENV
echo "BUILD_DATE=$(date +"%a %b %d, %Y at %H:%M:%S %Z")" >> $GITHUB_ENV
if [ -f site.txt ]; then echo "OFFICIALSITE=$(cat site.txt)" >> $GITHUB_ENV; fi
```

## Build Command Pattern

```bash
LDFLAGS="-s -w -X 'main.Version=${{ env.VERSION }}' -X 'main.CommitID=${{ env.COMMIT_ID }}' -X 'main.BuildDate=${{ env.BUILD_DATE }}' -X 'main.OfficialSite=${{ env.OFFICIALSITE }}'"
go build -ldflags "${LDFLAGS}" -o NAME ./src/server
```

## Concurrency Rules

- Push workflows (main/master/beta): cancel-in-progress: true
- Tag release: cancel-in-progress: true (same tag ref only)
- build-toolchain.yml: cancel-in-progress: false (never interrupt)

## NEVER in CI/CD

- Use Makefile targets (must be explicit commands)
- Reference host paths (~/.local/share/go)
- Pin Actions with tags — always use full commit SHA

For complete details, see AI.md PART 28

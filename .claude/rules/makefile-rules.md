# Makefile Rules (PART 26) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 26

## CRITICAL — NEVER DO

- Add targets beyond the 6 allowed (dev, local, build, test, release, docker)
- Use host-path Go cache (`~/.local/share/go`) — named volume `go-state` only
- Hardcode PROJECTNAME or PROJECTORG — always infer from git remote
- Run `go build` or `go test` on the host — always inside Docker container
- Use `golang:alpine` or any other Go image — only `casjaysdev/go:latest`
- Use `-v $(HOME)/.local/share/go:...` — use `-v go-state:/usr/local/share/go`

## CRITICAL — ALWAYS DO

- Use named Docker volume: `-v go-state:/usr/local/share/go`
- Set `-e CGO_ENABLED=0` in every Docker run
- Infer version from `release.txt`
- Infer OFFICIALSITE from `site.txt`
- Enforce ≥80% coverage in `test` target
- Use `mktemp -d "${TMPDIR:-/tmp}/${PROJECTORG}/${PROJECTNAME}-XXXXXX"` in `dev` target

## Six Targets ONLY — DO NOT ADD MORE

| Target | Purpose | Output |
|--------|---------|--------|
| dev | Quick dev build (no ldflags) | ${TMPDIR}/${PROJECTORG}/${PROJECTNAME}-XXXXXX/ |
| local | Local platform only | binaries/ |
| build | All 8 platforms | binaries/ |
| test | Unit tests + coverage gate | Coverage report |
| release | Build + GitHub release | releases/ |
| docker | Multi-arch container | Registry |

## Key Variables

```makefile
# Auto-detect from git (NEVER hardcode)
PROJECTNAME := $(shell git remote get-url origin 2>/dev/null | sed -E 's|.*/([^/]+)(\.git)?$$|\1|' || basename "$$(pwd)")
PROJECTORG  := $(shell git remote get-url origin 2>/dev/null | sed -E 's|.*/([^/]+)/[^/]+(\.git)?$$|\1|' || basename "$$(dirname "$$(pwd)")")

# Version from release.txt
VERSION ?= $(shell cat release.txt 2>/dev/null || echo "devel")

# Official site from site.txt (optional)
OFFICIALSITE := $(shell [ -f site.txt ] && cat site.txt || echo "${OFFICIALSITE:-}")

# Docker with named volume (NOT host path)
GO_DOCKER := docker run --rm -it \
    -v $(PWD):/app \
    -v go-state:/usr/local/share/go \
    -e CGO_ENABLED=0 \
    casjaysdev/go:latest
```

## Coverage Gate (test target)

- ≥80% coverage required
- Fails CI if below threshold

## Platforms (8 total)

linux/amd64, linux/arm64, darwin/amd64, darwin/arm64,
windows/amd64, windows/arm64, freebsd/amd64, freebsd/arm64

## Named Volume (REQUIRED)

go-state:/usr/local/share/go — persistent Go module cache
Never use host paths (~/.local/share/go) in Makefile.

For complete details, see AI.md PART 26

# Makefile Rules (PART 26) — Cheatsheet

Full spec: AI.md PART 26

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

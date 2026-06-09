# Project Rules (PART 2, 3, 4) — Cheatsheet

Full spec: AI.md PART 2, PART 3, PART 4

## Project Variables (from IDEA.md)

- project_name: caspaste
- project_org: casapps (github: casjay-forks)
- internal_name: caspaste (FROZEN — never change)
- binary_name: caspaste
- cli_binary_name: caspaste-cli
- config_dir: /etc/casjay-forks/caspaste
- data_dir: /var/lib/casjay-forks/caspaste

## Key Directory Rules

- Source: `src/` (Go packages)
- Docker: `docker/` (never in root)
- Tests: `tests/` (shell integration scripts)
- Binaries: `binaries/` (gitignored)
- Docs: `docs/` (MkDocs only)

## Required Root Files

- AI.md (spec, READ-ONLY)
- IDEA.md (project description, editable)
- CLAUDE.md (short loader)
- README.md
- LICENSE.md (MIT)
- Makefile (6 targets only)
- release.txt (version)
- site.txt (official URL, if hosted)

## Go Package Naming

- Singular: handler/, model/, middleware/ (matches package name)
- Tooling: scripts/, tests/, completions/ (always plural)

## Container Paths

- Config: /config/caspaste/
- Data: /data/caspaste/
- Logs: /data/log/caspaste/
- SQLite: /data/db/sqlite/server.db
- Backups: /data/backups/caspaste/

## Temp Directory Pattern (REQUIRED)

```bash
mkdir -p "${TMPDIR:-/tmp}/${PROJECTORG}"
BUILD_DIR=$(mktemp -d "${TMPDIR:-/tmp}/${PROJECTORG}/${PROJECTNAME}-XXXXXX")
```

NEVER use /tmp directly — always use /{org}/{project}-XXXXXX structure.

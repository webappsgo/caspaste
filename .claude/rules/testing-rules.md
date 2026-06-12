# Testing Rules (PART 29) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 29

## CRITICAL — NEVER DO

- Run `go test` on the host — always inside Docker `casjaysdev/go:latest`
- Run `docker compose up` in the project directory — always copy to temp dir first
- Use `docker-compose.yml` or `docker-compose.dev.yml` for testing — only `docker-compose.test.yml`
- Create runtime data (db files, logs, config) in the project directory
- Commit with failing tests — all tests must pass before commit

## CRITICAL — ALWAYS DO

- Build binaries via Docker with the `go-state` named volume
- Use temp dir: `mktemp -d "${TMPDIR:-/tmp}/${PROJECTORG}/${PROJECTNAME}-XXXXXX"`
- Copy `docker/docker-compose.test.yml` to temp dir before running
- Test the admin setup flow (setup token → create admin → login) in integration tests
- Test content negotiation on every route (text/html vs text/plain)
- Enforce ≥80% Go unit test coverage (hard gate in `make test`)
- Clean up temp dirs after tests complete

## Two Required Test Types

| Type | Location | Run With | Coverage |
|------|----------|----------|---------|
| Go unit tests | *_test.go | go test | ≥80% code coverage |
| Integration tests | tests/*.sh | ./tests/run_tests.sh | 100% endpoint coverage |

## Integration Test Scripts (Required)

- tests/run_tests.sh — auto-detect Incus/Docker, run appropriate script
- tests/docker.sh — Alpine container, no systemd
- tests/incus.sh — Debian + systemd, preferred

## AI Testing Rules

- ALWAYS use docker-compose.test.yml (copy to temp dir first)
- NEVER use docker-compose.yml or docker-compose.dev.yml
- NEVER create runtime data in project directory

```bash
# Correct AI test workflow:
mkdir -p "${TMPDIR:-/tmp}/${PROJECTORG}"
TEMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/${PROJECTORG}/${PROJECTNAME}-XXXXXX")
cp docker/docker-compose.test.yml "$TEMP_DIR/docker-compose.yml"
mkdir -p "$TEMP_DIR/volumes/config" "$TEMP_DIR/volumes/data"
cd "$TEMP_DIR" && docker compose up -d
# ... run tests ...
docker compose down
rm -rf "$TEMP_DIR"
```

## Test Script Requirements

1. Build all binaries via Docker (casjaysdev/go:latest + go-state volume)
2. Never build on host
3. Test version, help, binary info
4. Binary rename test (copy binary, verify --help shows new name)
5. Admin setup flow (setup token → create admin → login)
6. API endpoints with .txt extension and Accept headers
7. Frontend content negotiation (text/html, text/plain)
8. Paste CRUD operations
9. CLI full functionality

## Content Negotiation Tests

Every route tested with:
- Frontend: Accept: text/html → HTML, Accept: text/plain → text
- API: Accept: application/json → JSON, Accept: text/plain → text
- .txt extension: /api/v1/resource.txt → plain text

## Coverage Gates

- Go unit tests: ≥80% (enforced by `make test`)
- Integration: 100% endpoint coverage
- Critical paths (auth, DB, token) always tested

For complete details, see AI.md PART 29

# AI Rules (PART 0, 1) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 0, PART 1

## CRITICAL — NEVER DO

- Guess or assume any value a command can produce — always run the command or read the spec
- Implement without reading the relevant PART first
- Use bcrypt → Argon2id for passwords
- Use CGO → CGO_ENABLED=0 always (static binary)
- Build Go on the host → always use Docker `casjaysdev/go:latest`
- Use external cron daemon → built-in scheduler (PART 19)
- Put admin panel at `/{admin_path}/` → always `/server/{admin_path}/`
- Use React/Vue/Angular → server-side Go templates only
- Skip verification and claim "done" without testing
- Create report/analysis files (AUDIT.md, COMPLIANCE.md, etc.) — fix directly

## CRITICAL — ALWAYS DO

- Read the relevant PART(s) before every task
- Run `go build ./...` in Docker before claiming compilation success
- Run `go test ./...` in Docker before every commit
- Use Argon2id for password hashing; SHA-256 for token hashing
- Ensure all features work without JavaScript (progressive enhancement only)
- Stop and ask when requirements are ambiguous
- Surface issues in your response, then fix them (no silent fixes)

## Never Guess or Assume

| Situation | Action |
|-----------|--------|
| Unsure about requirement | STOP and ASK |
| Can't find file/function | Search first, ask if not found |
| Multiple valid approaches | List options, ask user |
| Spec seems incomplete | Ask for clarification |

## Mandatory Before Each Task

1. Identify relevant PARTs in AI.md
2. Read those PARTs completely
3. Implement exactly as specified

## Verification Checklist (Before "Done")

- [ ] Read the relevant files first
- [ ] Searched for existing patterns
- [ ] Tested changes in Docker
- [ ] Did NOT guess or assume

## Red Flags — STOP Immediately

- "This is probably what they meant..." → ASK
- "I'll just assume..." → ASK
- "This should work..." → TEST
- "I think I remember..." → READ THE SPEC

## Critical Rules (PART 1)

- CGO_ENABLED=0 always (static binary)
- No bcrypt → Argon2id for passwords, SHA-256 for tokens
- No Go on host → use `casjaysdev/go:latest` Docker image
- No external cron → built-in scheduler (PART 19)
- All 8 platforms: linux/darwin/windows × amd64/arm64 + freebsd × amd64/arm64
- Admin panel always at `/server/{admin_path}/` (PART 17)
- Server-side templates only (no React/Vue/Angular)
- All features work without JavaScript

For complete details, see AI.md PART 0, PART 1

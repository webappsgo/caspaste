# AI Rules (PART 0, 1) — Cheatsheet

Full spec: AI.md PART 0, PART 1

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
- [ ] Tested my changes
- [ ] Did NOT guess or assume

## Red Flags — STOP Immediately

- "This is probably what they meant..." → ASK
- "I'll just assume..." → ASK
- "This should work..." → TEST
- "I think I remember..." → READ THE SPEC

## Critical Rules (PART 1)

- CGO_ENABLED=0 always (static binary)
- No bcrypt → Argon2id for passwords, SHA-256 for tokens
- No docker build on host → use casjaysdev/go:latest
- No external cron → built-in scheduler (PART 19)
- All 8 platforms: linux/darwin/windows × amd64/arm64 + freebsd × amd64/arm64
- Admin panel always at /server/{admin_path}/ (PART 17)
- Server-side templates (no React/Vue)
- All features work without JavaScript

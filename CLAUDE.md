Role: Efficient loader for AI.md

⚠️ **THIS FILE IS AUTO-LOADED EVERY CONVERSATION. FOLLOW IT EXACTLY.** ⚠️

Purpose:
- This file is a short loader for the most important rules
- `AI.md` is the full source of truth
- For complete details, read the referenced PARTs in `AI.md`

## FIRST TURN - MANDATORY

On EVERY new conversation or after "context compacted" message:
1. **READ** the relevant `.claude/rules/*.md` for your current task
2. **NEVER** assume or guess — verify against AI.md before implementing

## Asking Questions

- **Default to continuing work** — do not stop just to ask whether you should continue
- **Never guess** — if the answer cannot be determined from `AI.md`, `IDEA.md`, the codebase, or repo state, ASK the user
- **Do NOT ask for permission to keep going** — continue until the current task is complete, blocked by a real decision, or the user explicitly asks to pause
- **Question mark = question** — when user ends with `?`, answer/clarify, don't execute

## Before ANY Code Change

1. Have I read the relevant PART in AI.md? (If no → read it)
2. Does this follow the spec EXACTLY? (If unsure → check spec)
3. Am I guessing or do I KNOW from the spec? (If guessing → read spec)
4. Would this pass the compliance checklist? (AI.md FINAL section)

**WHEN IN DOUBT: READ THE SPEC. DO NOT GUESS.**

## Key Project Info (from IDEA.md)

- **project_name**: caspb
- **project_org**: casapps
- **internal_name**: caspb
- **binary_name**: caspb
- **cli_binary_name**: caspb-cli
- **admin_path**: admin (default)
- **default_port**: 80
- **official_site**: https://pste.us

## NEVER Do (Critical Violations)

1. Use bcrypt → Use Argon2id
2. Put Dockerfile in root → `docker/Dockerfile`
3. Use CGO → CGO_ENABLED=0 always
4. Hardcode dev values → Detect at runtime
5. Use external cron → Internal scheduler (PART 19)
6. Use Makefile in CI/CD → Explicit commands only
7. Skip platforms → Build all 8 (linux/darwin/windows × amd64/arm64)
8. Client-side rendering → Server-side Go templates
9. Run Go on host → Always use Docker (casjaysdev/go:latest)
10. Guess or assume → Read spec or ask user

## ALWAYS Do

1. Read AI.md before implementing ANY feature
2. All builds/tests in Docker (casjaysdev/go:latest)
3. Mobile-first responsive CSS
4. All features work without JavaScript
5. Full admin panel at /server/{admin_path}/config/
6. Client binary (caspaste-cli) for ALL projects
7. Commit often — small, focused commits

## Source Entry Points

- Server: `./src/server` (main package)
- CLI: `./src/client` (main package)
- Admin: `./src/admin/`
- Storage: `./src/storage/`
- Config: `./src/config/`

## Where to Find Details

All 14 rule files live under `.claude/rules/` — load the one(s) matching your current task:

| File | PARTs |
|------|-------|
| `ai-rules.md` | 0, 1 — AI behavior, critical rules |
| `project-rules.md` | 2, 3, 4 — structure, paths |
| `config-rules.md` | 5, 6, 12 — config, modes, server config |
| `binary-rules.md` | 7, 8, 33 — binary, CLI, client |
| `backend-rules.md` | 9, 10, 11, 32 — DB, security, Tor |
| `api-rules.md` | 13, 14, 15 — health, API, SSL |
| `frontend-rules.md` | 16, 17 — web UI, admin panel |
| `features-rules.md` | 18–23 — email, scheduler, GeoIP, metrics, backup, update |
| `service-rules.md` | 24, 25 — privilege, service |
| `makefile-rules.md` | 26 — Makefile |
| `docker-rules.md` | 27 — Docker |
| `cicd-rules.md` | 28 — CI/CD |
| `testing-rules.md` | 29–31 — testing, docs, i18n |
| `optional-rules.md` | 34–36 — multi-user, orgs, domains |

Full spec: `AI.md` ← **SOURCE OF TRUTH**

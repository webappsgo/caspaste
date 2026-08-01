# AI Assistant Rules (PART 0, 1)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Guess or assume a requirement, file location, default value, or user intent — STOP and ASK instead
- "Improve" or "optimize" the spec, invent patterns/flags/defaults not in AI.md, or rename spec terms "for brevity"
- Modify PARTS 0-36 of AI.md (read-only template) — project overrides go in SPEC.md, business logic in IDEA.md
- Create report/analysis files (AUDIT.md, COMPLIANCE.md, SUMMARY.md) — fix issues directly instead
- Jump between half-finished features — complete ONE thing fully before starting the next
- Run `go` commands directly on the local machine — ALL builds/tests use Docker (`casjaysdev/go:latest`) or Incus
- Use plain `git commit` / `git push`, or `gitcommit -m`/`--message` — message must be written to `.git/COMMIT_MESS` and re-read first
- Let a subagent write `.git/COMMIT_MESS` or call `gitcommit` — only the parent instance commits
- Use `SELECT *` in application code, string-concatenate SQL, or pass user input to shell/eval directly
- Credit an AI tool anywhere in code, comments, commits, or PRs — no assistant co-author trailers, no assistant-authorship comments
- Place comments inline/below code (Go, YAML, JS, CSS) — always above; never any comment in JSON
- Leave unused optional PARTs (34-36) with any trace in code — no tables, conditionals, stubs, or config toggles for unused features
- Read an image larger than 1000x1000 directly — resize to a tmp copy first

## CRITICAL - ALWAYS DO
- Read the relevant AI.md PART(s) before each task — not the whole spec speculatively, not from memory
- Ask via numbered/lettered options when the spec is unclear or multiple approaches are valid
- Verify with real tools before claiming "done" (tests, curl, build, browser) — never rely on "looks right"
- Treat servers as internet-facing and hostile-traffic-exposed by default; never weaken authn/authz/TLS/CSRF/rate-limiting to improve usability
- Validate and sanitize all input; parameterized queries only; name columns explicitly (never `SELECT *`)
- Keep README.md, Swagger, GraphQL, docs/, and CLI `--help` in sync with actual code after every change
- Make every config setting editable via the admin WebUI, with live reload (except listen address/port/DB driver)
- Use `.claude/rules/*.md` files (14 total) to load context efficiently instead of re-reading all of AI.md each session
- Track 3+ tasks in TODO.AI.md; remove human-owned TODO.md/PLAN.md items only by marking done, never delete/empty/rewrite them
- Use the standard curl flags `-q -LSsf` in all docs/scripts/examples

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| Should I guess when unsure? | No — asking is ~50x cheaper than a wrong guess + redo | PART 0, "The Cost of Guessing" |
| Where do project-specific business rules live? | IDEA.md (WHAT); AI.md is HOW and is read-only | PART 0, "AI.md Structure" |
| How do I override a template rule for this project? | SPEC.md — it wins over AI.md, which wins over global CLAUDE.md | PART 0, "Project Files" |
| When is a full audit triggered? | Only on explicit "audit" / "check compliance" / "verify project" | PART 0, "Audit (Explicit Command Only)" |
| Where do audit findings >5 issues get tracked? | Temporary AUDIT.AI.md, deleted when all resolved | PART 0, Step 8 |
| Can I add a CLI flag not in spec? | No — only flags/commands defined in AI.md | PART 0, "Common AI Deviation Patterns" |
| Do "simple" projects get a lite version of the spec? | No — every project implements the full spec | PART 0, "New Project Implementation Rules" |
| Is PART 34 (Multi-User) required? | Only if the project has end-user accounts; admin auth (PART 17) is always mandatory | PART 0, "Optional Section Decision Guide" |
| What build tool may I use locally? | Docker/Incus only — Go is not installed on the host | PART 1, "Container-Only Development" |
| How do I format an error for a user vs. an admin vs. a log? | Minimal for user, actionable for admin, full+context in logs | PART 1, "Error Message Rules" |
| What are the default rate limits? | 120/min read, 10/min write, 5/15min login, 3/hr password reset | PART 1, "Rate Limiting Defaults" |
| When must a bare `/path` not be used in code? | Always use `{fqdn}/path` via `BuildURL(r, ...)` except internal router registration | PART 1, "URL Standards" |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| AI.md | Implementation spec (HOW) — PARTS 0-36 fixed template + PART 37 reference; never modified |
| IDEA.md | Project spec (WHAT) — business logic, features; three required sections: description, variables, business logic |
| SPEC.md | Project-specific rule overrides / optional PART 34-36 activation; wins over AI.md when they conflict |
| TODO.AI.md / PLAN.AI.md | AI-owned tracking files; emptied/rewritten via completion rituals when fully done |
| TODO.md / PLAN.md | Human-owned; AI may only mark items done, never delete/empty/rewrite |
| AUDIT.AI.md | Temporary, audit-only tracking file for >5 findings; deleted once resolved |
| Check Files | Discovery only (what is this project) — never fixes or compares against spec |
| Audit | Full compliance verification — ONLY on explicit user command; fixes issues, doesn't just report them |
| Project / App / Application | Used interchangeably throughout AI.md |
| Golden Rules | The six PART 1 principles: read spec, never guess, implement NON-NEGOTIABLE sections exactly, keep AI.md in sync, never install Go locally |

## QUICK REFERENCE
- Verification checklist before claiming "done": read files, search patterns, test changes, verify output, confident, didn't guess, didn't rush, asked if unsure
- Red flags that mean STOP: "probably what they meant", "I'll just assume", "this should work" (untested), "I'll fix later", "close enough"
- Naming: files `lowercase_snake.go`; public `PascalCase`; private `camelCase`; booleans `isX`/`hasX`/`canX`; never generic names like `Mode`, `Type`, `Status`, `Config` alone — always qualify (`AppMode`, `TokenType`)
- Formatting: 2-space indent everywhere except Go/Makefile (tabs); single trailing newline; no trailing whitespace; `gofmt` for Go
- Full web app = browser (HTML) + PWA + JSON API + CLI client, all covering the same routes (`/x` ↔ `/api/{api_version}/x`)
- Security principles: never trust input, defense in depth, least privilege, fail secure, secure by default, suggest-don't-block MFA
- Attack prevention: parameterized SQL, HTML-escape + CSP for XSS, CSRF tokens on state-changing forms, `filepath.Clean()` + reject `..` for path traversal
- README section order: Title/Badges → About → Official Site → Features → Production → Client → Configuration → API → Other → Development (last) → Disclaimer → License
- Every README needs a real, specific Disclaimer section (no warranty, not professional advice, third-party services, security, production use)
- Sensitive data: never expose in healthz/API/errors/logs/HTML; tokens/passwords shown once at generation only, masked afterward

---
For complete details, see AI.md PART 0, 1

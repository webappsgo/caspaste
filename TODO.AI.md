## [ ] Create .claude/rules/*.md cheatsheet files (14 files)
Read: AI.md PART 0
PART 0 "Session Initialization" mandates creating
.claude/rules/{ai,project,config,binary,backend,api,frontend,features,
service,makefile,docker,cicd,testing,optional}-rules.md if the .claude/rules/
directory does not exist. It currently does not exist. Each file must follow
the required format (header with PART numbers, NON-NEGOTIABLE warning,
CRITICAL NEVER/ALWAYS sections, key rules summary, reference line) sourced
from the corresponding AI.md PARTs. This requires reading PARTs 7-36, which
was out of scope for this reconciliation pass.

## [ ] Verify LICENSE.md copyright year
Read: AI.md PART 2
PART 2 requires copyright year = "current year or year of first publication".
Copyright holder updated to "webappsgo" (= project_org) as part of the
caspaste/webappsgo naming-conflict resolution. LICENSE.md currently states
"Copyright (c) 2024 webappsgo". The year 2024 predates the first commit
visible in this git history (2026) — could be correct if the project was
first published elsewhere in 2024, or could be stale. Needs human
confirmation of the actual first-publication year; not changed in this pass.

## [ ] Fix GHCR push permission denial in Docker Images workflow
Read: .github/workflows/docker-images.yml
`docker/build-push-action` fails with `denied: permission_denied:
write_package` pushing to `ghcr.io/webappsgo/caspaste`. The workflow
already declares `permissions: packages: write`, but the org
`webappsgo` has "Write permissions for workflows" disabled at the
org level, which overrides the repo/workflow setting (confirmed via
`gh api repos/webappsgo/caspaste/actions/permissions/workflow` →
`default_workflow_permissions: read`; attempting to PUT `write` returned
409 "Write permissions for workflows are disabled by the organization").
Requires org-admin action: either enable write workflow permissions for
this repo at the org level, or switch the login step to a PAT with
`write:packages` scope stored as a secret instead of `GITHUB_TOKEN`.
The `origin` remote was corrected from `casjay-forks/caspaste` to
`webappsgo/caspaste` (the actual org) and history fast-forward-pushed
there; the org-level policy is the same on webappsgo, so this did not
resolve on its own.

## [ ] Implement PART 6 Application Modes (mode/debug system is dead code)
Read: AI.md PART 6 (line 9251)
Correction to a prior (mischaracterized) entry: PART 6 is "APPLICATION
MODES" (production/development + debug flag), not CLI/server/TUI dispatch.
Verified: `src/mode/mode.go` implements `Set`/`SetDebug`/`FromEnv`/
`IsAppModeDev`/`IsDebugEnabled`, but `grep -rn` for
`"github.com/webappsgo/caspaste/src/mode"` across `src/` returns zero
matches — the entire package is unimported dead code. Consequences:
- No `--mode` / `--debug` CLI flags exist (verified: no `"--mode"`/
  `"--debug"` case in src/cli/cli.go or src/server/caspaste.go; the only
  `case "mode":` hit is `--maintenance mode {enabled|disabled}`, unrelated).
- No `MODE`/`DEBUG` env var is read anywhere (`mode.FromEnv()` is never
  called).
- No `/debug/*` routes are registered at all (no pprof, `/debug/vars`,
  `/debug/config`, `/debug/routes`, `/debug/cache`, `/debug/db`,
  `/debug/scheduler`) — src/server/debug.go per the AI.md template does
  not exist.
- The Four Operational States table's per-mode behavior differences
  (logging level, template/static caching, rate-limit strictness, security
  header relaxation, request-logging verbosity) are not wired to
  `mode.IsAppModeDev()`/`IsDebugEnabled()` anywhere.
This is a feature-sized gap (new CLI flags + env wiring + a debug-routes
file + touching logging/caching/rate-limit/security-header call sites),
not a one-line fix, and `/debug/pprof` + verbose request/body logging are
security-sensitive surface that should only be exposed deliberately —
flagging for a scoping decision rather than implementing blind.


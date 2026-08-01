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


## [ ] Wire remaining PART 6 Four Operational States behaviors
Read: AI.md PART 6 (line 9251)
PART 6 "Application Modes" is now implemented: `--mode`/`--debug` flags
and `MODE`/`DEBUG` env vars (flag > env > mode-alias > default), the full
`/debug/*` route set (pprof, expvar, config, routes, cache, db,
scheduler, memory, goroutines) gated behind `mode.IsDebugEnabled()` and
unreachable when off, dev-mode debug-level logging, disabled rate
limiting in dev, verbose panic recovery in dev/debug, and disabled
static-asset caching in dev (src/web/etag.go). A pre-existing debug-mode
admin-auth bypass in src/admin/middleware.go (violated PART 6's "debug
never bypasses security checks") was found and removed as part of this
work.
Three Four-Operational-States behaviors were intentionally left unwired,
each for a concrete reason rather than an oversight:
- CORS relaxation in dev — `CORSMiddleware` is already permissive by
  design; no separate strict/relaxed toggle exists to branch on.
- Security-header relaxation in dev — `SecurityHeadersMiddleware` is
  fully config-driven with no explicit-vs-default distinction; loosening
  it risks silently overriding intentional user config.
- Template hot-reload in dev — templates are parsed once from an
  embedded FS at startup with no dynamic reload mechanism; only the
  cache-header side was wired (etag.go), not actual reparsing.
Revisit if/when CORS config, security-header config, or the template
loader gain the hooks needed to distinguish "default" from
"user-configured" behavior.

## [ ] Decide the PORT env-var name (bare `PORT` vs `CASPASTE_PORT`)
Read: AI.md PART 8 flag table (line 13300) and port-resolution flow
(lines 11229, 11264) vs the Docker/container sections (lines 589, 617).
AI.md is internally contradictory for the listen-port env var: the flag
table and startup port-resolution flow specify `{PROJECT_NAME}_PORT`
(= `CASPASTE_PORT`), while the container/Docker sections and the distilled
config-rules init-only list specify a bare `PORT`. The code currently reads
bare `PORT` (src/server/caspaste.go:2646). This is a naming decision only
the user can settle; do not guess. Once decided, align the code and document
the chosen name in README + docker-compose defaults.
NOTE: the other flagged env vars are NOT ambiguous and need no change —
`MODE`/`DEBUG` are spec-authored bare (AI.md:9858, 9863) and `SMTP_*` are
spec-authored bare (PART 18 "honor SMTP_* env vars"); the bare forms in
src/server/mode.go and src/notify/email.go are correct as written.


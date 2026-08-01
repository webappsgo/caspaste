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


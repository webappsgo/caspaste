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

## [x] PORT env-var name resolved — CASPASTE_PORT
AI.md is not actually contradictory: `{PROJECT_NAME}_PORT` (=
`CASPASTE_PORT`) is the server binary's own flag-fallback env var (PART 8
flag table, native runs); the Docker sections' bare `PORT` is a
container-only convention already translated into `--port` by
docker/rootfs/.../entrypoint.sh before the binary runs, so the binary
itself never needs to read it. src/server/caspaste.go now reads only
CASPASTE_PORT (src/config/env.go's getEnv() already prefixed correctly).
README.md already documented CASPASTE_PORT; no doc change needed.

## [ ] Migrate frozen `internal_org` from `casapps` to `webappsgo`
User confirmed `webappsgo` is canonical (matches IDEA.md line 33 and
src/path/path.go). Everything else currently hardcodes `casapps/` and must
be migrated to match:
- `src/server/caspaste.go`, `src/storage/storage.go`, `src/config/yaml.go`
  (incl. GenerateDefaultYAMLConfig's Linux-only literal — AUDIT.AI.md Pass 5
  item folds in here), `src/client/main.go`, `src/tui/app.go` — replace
  hardcoded `casapps/` path segments with `webappsgo/`.
- IDEA.md line 50 — fix the macOS bundle-id example from
  `io.github.casapps.caspaste` to `io.github.webappsgo.caspaste` so IDEA.md
  is internally consistent.
- Once all resolvers agree, consolidate the duplicate per-OS path-resolution
  logic in server/storage/client/tui into the single `src/path` resolver
  (currently only imported by `src/ssl/ssl.go`) so this cannot drift again.
- Any existing on-disk `casapps/` install data is orphaned by this rename;
  note it in README/docs as a breaking path change for anyone who ran a
  pre-migration build.


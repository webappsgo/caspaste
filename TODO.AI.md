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

## [ ] Decide the frozen `internal_org`: `webappsgo` vs `casapps` (data-path split-brain)
Read: IDEA.md variables block (line 33) and plist note (line 50); AI.md PART 4
(OS path layout). This is a FROZEN identifier that controls on-disk data
locations, so it is a human-only decision — guessing it risks orphaning a
running install's config/data dirs, so no code was changed.
The project currently disagrees with itself about the org segment of every
OS path:
- IDEA.md line 33 declares `internal_org: webappsgo`, but IDEA.md line 50
  derives the macOS bundle id as `io.github.casapps.caspaste` (i.e. casapps).
  IDEA.md is internally contradictory.
- Code is split: `src/path/path.go` resolves every dir under `webappsgo/`
  (e.g. `/etc/webappsgo/caspaste`), but that package is imported by ONLY
  `src/ssl/ssl.go`. Every other subsystem — `src/server/caspaste.go`,
  `src/storage/storage.go`, `src/config/yaml.go`, `src/client/main.go`,
  `src/tui/app.go` — hardcodes `casapps/` paths (e.g. `/etc/casapps/caspaste`,
  `/var/lib/casapps/caspaste`).
- Net effect (active bug): the SSL/security dirs land under
  `.../webappsgo/caspaste/...` while config, data, db, logs, PID and the CLI
  config land under `.../casapps/caspaste/...`. The two halves of the app do
  not agree on where the install lives.
Recommendation (NOT applied — needs user confirmation of the frozen value):
the pervasive runtime paths and the plist example both use `casapps`, so
`casapps` is most likely the truly-frozen internal_org and both IDEA.md line
33 and `src/path/path.go` are the outliers. If confirmed: set IDEA.md
internal_org to `casapps`, change `src/path/path.go` projectOrg to `casapps`
(or make every resolver delegate to one shared resolver), and consolidate the
duplicate per-OS resolvers in server/storage/client into `src/path`. If
instead `webappsgo` is canonical, the mass of casapps literals must be
migrated the other way. Either direction, unify on the single `src/path`
resolver afterward so this cannot drift again.
Also fold in AUDIT.AI.md Pass 5 item "GenerateDefaultYAMLConfig hardcodes
Linux-only dir defaults (yaml.go:713-718)": that fix is blocked on this
decision, because the correct fix is to have GenerateDefaultYAMLConfig call
the shared per-OS resolver instead of emitting Linux `casapps` literals — but
which org the resolver bakes in is exactly the question above.


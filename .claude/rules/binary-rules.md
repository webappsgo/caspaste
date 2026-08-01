# Binary & CLI Rules (PART 7, 8, 33)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never hand-roll flag parsing for the server binary (no manual `os.Args` switch loop) — use stdlib `flag`
- Never embed security databases (GeoIP, blocklists, CVE, Trivy) in the binary — download on first run, keep updated via scheduler
- Never use CGO — server/CLI/agent build with `CGO_ENABLED=0`, pure Go only
- Never implement `--tui`, `--cli`, `--gui`, or a `--mode tui/cli/gui` flag — display mode is always auto-detected; `--mode` is reserved for `production`/`development` only
- Never launch TUI when `TERM=dumb` — force CLI mode, no ANSI, no emojis, no spinners/progress bars (use text)
- Never require root/escalation for `--help` or `--version` at any level (main, subcommand, nested)
- Never use `strconv.ParseBool()` for boolean flags — use `config.ParseBool()`/`config.IsTruthy()`
- Never re-resolve `~`/`$HOME` after privilege drop (step 8g) — service account HOME points at `{data_dir}`, causes nested paths
- Never write a PID file inside a container (`isContainer()` true) — skip PID handling entirely
- Never let `--service start` respect the `daemonize` config — it auto-detects the service manager and always does the right thing
- Never give the agent `--port` or `--address` flags — it doesn't serve HTTP
- Never build agent/CLI GUI with Electron or a web view — native toolkit only (GTK4/Qt6, Cocoa, Win32)
- Never auto-retry after a CLI token revocation (401) — re-auth must be a deliberate user action

## CRITICAL - ALWAYS DO
- Always drop privileges as early as possible, but only after directories are created/chowned AND privileged ports (<1024) are bound
- Always show the ACTUAL (possibly renamed) binary name in `--help`/`--version`/error messages; always hardcode `{project_name}` for User-Agent, default paths, config keys, DB tables
- Always respect `NO_COLOR` (any non-empty value disables colors + emojis); priority: CLI flag > config > `NO_COLOR` > auto-detect
- Always detect stale PID files (dead process, or PID reused by another binary) and remove them before starting
- Always create all directories (config/data/cache/log/backup) with proper perms if missing: `0755`/`0644` root, `0700`/`0600` user
- Always support `--shell completions [SHELL]` and `--shell init [SHELL]` on ALL binaries, auto-detecting `$SHELL` when omitted
- Always accept both `--flag=value` and `--flag value` syntax
- Always give every flag a corresponding config-file setting (precedence: flag > env var > config > hardcoded default)
- Always exit with code 4 on `401 TOKEN_REVOKED`/`TOKEN_EXPIRED` in non-interactive CLI use, and delete the cached token
- Always fail fast if `cli.yml`/`token` file perms are looser than `0600` (Unix) / not user-only ACL (Windows) — refuse to use the token

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| Server flag parser? | stdlib `flag` (single command, no subcommands) | PART 8 § Flag Parsing |
| CLI flag/command parser? | `cobra`/`viper` (multi-command) | PART 8 § Flag Parsing |
| Server binary name | `{project_name}` | PART 8 |
| CLI binary name | `{project_name}-cli` | PART 8 |
| Agent binary name | `{project_name}-agent` | PART 8 |
| Server default port if none configured | Random 64000-64999 | PART 8 § Server --help |
| `--status` exit codes | 0 healthy, 1 unhealthy | PART 8 |
| CLI exit codes | 0 ok, 1 general, 2 config, 3 connection, 4 auth, 5 not found, 64 usage | PART 33 § Exit Codes |
| Does CLI need a setup wizard? | Yes — the ONLY binary with one (TUI/GUI); server/agent just show status banners | PART 33 |
| Agent first-run with no `--connect`/`--server`+`--token`? | Error, no wizard | PART 33 |
| CLI mode when no args + interactive terminal | TUI | PART 33 § Modes |
| CLI mode when args/command given | CLI (text) mode | PART 33 § Modes |
| CLI mode when piped/non-interactive | Plain text output | PART 33 § Modes |
| Remote (SSH/mosh) even with X11 forwarding? | Always TUI, never GUI | PART 33 § SSH/Mosh Detection |
| Does this project need an agent? | Only if server must reach INTO remote machines (collect data / execute commands) | PART 33 § When Agent Needed |
| Client runs as | Invoking user, `~/`-scoped paths | PART 33 § Execution Context |
| Agent runs as | root/Administrator, `/`-scoped (system) paths | PART 33 § Execution Context |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| Display Mode | GUI / TUI / CLI / Headless — auto-detected, never a user flag |
| App Mode | `production` / `development` — set via `--mode`, unrelated to display |
| `TERM=dumb` | No ANSI capability at all — force CLI, plain text, ASCII tables |
| Setup wizard (CLI) | Built-in interactive TUI/GUI wizard, CLI binary only |
| Setup wizard (Server) | Web-based `/server/{admin_path}/config/setup` page, browser-driven |
| `--shell init` | Convenience wrapper that prints an `eval`-ready line around `--shell completions` |
| Send Only / Receive Only / Bidirectional | Agent communication patterns (push metrics / pull config / full command-and-control) |

## QUICK REFERENCE
**Binary/mode support matrix:**

| Binary | GUI | TUI | CLI | Headless | Setup Wizard |
|--------|-----|-----|-----|----------|--------------|
| Server | Status window | Status banner | Commands | Default (daemon) | No (WebUI) |
| CLI | Full app | Full app (default) | Commands | Error | Yes (only one) |
| Agent | Status window | Status banner | Commands | Default (service) | No (`--connect`/`--server`+`--token`) |

**Universal flags (all binaries):** `--help`/`-h`, `--version`/`-v`, `--color {auto|yes|no}`, `--lang CODE`, `--shell {completions,init} [SHELL]`. Only `-h`/`-v` have short forms.

**Server-only key flags:** `--mode`, `--config`, `--data`, `--cache`, `--log`, `--backup`, `--pid`, `--address`, `--port`, `--baseurl`, `--status`, `--service {...}`, `--daemon`, `--maintenance {...}`, `--update {...}`.

**CLI-only key flags:** `--server`, `--token`/`--token-file`, `--user` (`@user`/`+org`/auto-detect), `--config NAME`, `--output`, `--admin`.

**Agent-only key flags:** `--server`, `--token`, `--data`, `--log`, `--status`, `--service`, `--update`; subcommands `status`, `test`, `register`. No `--port`/`--address`.

**Signals (Unix):** SIGTERM/SIGINT/SIGQUIT/SIGRTMIN+3 → graceful shutdown; SIGHUP → ignored (auto-reload); SIGUSR1 → reopen logs; SIGUSR2 → status dump. Windows: only `os.Interrupt` + SCM stop.

**Directory defaults (Linux):** root → `/etc|/var/lib|/var/cache|/var/log|/mnt/Backups/{internal_org}/{internal_name}/`; user → `~/.config|~/.local/share|~/.cache|~/.local/log|~/.local/share/Backups/{internal_org}/{internal_name}/`.

---
For complete details, see AI.md PART 7, 8, 33

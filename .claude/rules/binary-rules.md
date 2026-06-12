# Binary Rules (PART 7, 8, 33) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 7, PART 8, PART 33

## CRITICAL — NEVER DO

- Use CGO — always `CGO_ENABLED=0` (static binary, no C deps)
- Embed security databases (GeoIP, blocklists, CVE) — download on first run
- Hardcode binary name in `--help` or `--version` — use `filepath.Base(os.Args[0])`
- Hardcode the project name in User-Agent display — use actual binary name for display
- Add CLI commands not listed in the spec — the command set is FROZEN
- Use `NO_COLOR` env var when it is empty (only non-empty value disables color)
- Show ANSI colors, emojis, TUI, or cursor control when `TERM=dumb`
- Build on the host — always use Docker (`casjaysdev/go:latest`)

## CRITICAL — ALWAYS DO

- Single static binary, assets embedded via Go `embed`
- `CGO_ENABLED=0` in every build
- Detect display environment: GUI → TUI → CLI → Headless
- Respect `NO_COLOR` env (non-empty = disable colors AND emojis)
- Respect `TERM=dumb` (force CLI mode, no ANSI, no TUI, no emojis)
- Show actual `filepath.Base(os.Args[0])` in `--help`, `--version`, error messages
- Hardcode `{project_name}` only for User-Agent, config paths, DB keys
- CLI binary (`caspaste-cli`) is REQUIRED — agent is optional (per-project)

## Binary Types

| Binary | Name | User-Agent | Config |
|--------|------|-----------|--------|
| Server | `caspaste` | `caspaste/{version}` | `/etc/casjay-forks/caspaste/` |
| CLI | `caspaste-cli` | `caspaste-cli/{version}` | `~/.config/casjay-forks/caspaste/` |
| Agent | `caspaste-agent` | `caspaste-agent/{version}` | varies |

## Server Commands (FROZEN — no additions)

```
--help, --version, --shell completions/init [SHELL]
--mode {production|development}
--config, --data, --cache, --log, --backup, --pid
--address, --port, --baseurl
--status
--service {start,restart,stop,reload,--install,--uninstall,--disable,--help}
--daemon
--debug
--color {always|never|auto}
--lang {code}
--maintenance {backup,restore,update,mode,setup,--help}
--update [check|yes|branch {stable|beta|daily}|--help]
```

## NO_COLOR Priority (all binaries)

| Priority | Source |
|----------|--------|
| 1 | `--color` flag |
| 2 | Config file `output.color` |
| 3 | `NO_COLOR` env (non-empty = disable) |
| 4 | Auto-detect (TTY + TERM) |

## Display Mode Hierarchy

| Mode | When |
|------|------|
| GUI | Native display, CLI binary only |
| TUI | Interactive terminal (not SSH if no display) |
| CLI | Command provided or piped |
| Headless | No display, no TTY (daemon/service) |

Force CLI mode when `TERM=dumb` — no ANSI, no emojis, no spinners, ASCII tables.

## Directory Defaults (Server Binary)

| Flag | Linux root | Linux user |
|------|-----------|-----------|
| `--config` | `/etc/casjay-forks/caspaste/` | `~/.config/casjay-forks/caspaste/` |
| `--data` | `/var/lib/casjay-forks/caspaste/` | `~/.local/share/casjay-forks/caspaste/` |
| `--log` | `/var/log/casjay-forks/caspaste/` | `~/.local/log/casjay-forks/caspaste/` |

## External Security Data (NOT Embedded)

| Data | Location | Source | Update |
|------|----------|--------|--------|
| GeoIP | `{data_dir}/security/geoip/` | ip-location-db | Daily |
| Blocklists | `{data_dir}/security/blocklists/` | Configurable | Daily |
| CVE | `{data_dir}/security/cve/` | NVD/NIST | Daily |

## CLI Token Rules (PART 33)

- CLI config/token files MUST be `0600` — refuse to load if group/world readable
- Token source priority: `--token` → `--token-file` → env `CASPASTE_TOKEN` → `cli.yml` → `token` file
- On `401 TOKEN_REVOKED` or `TOKEN_EXPIRED`: delete cached token, exit gracefully with message
- Cluster failover: try all cluster URLs from autodiscover before giving up

For complete details, see AI.md PART 7, PART 8, PART 33

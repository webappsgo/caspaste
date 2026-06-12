# Service Rules (PART 24, 25) — Cheatsheet

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

Full spec: AI.md PART 24, PART 25

## CRITICAL — NEVER DO

- Prompt for privilege escalation if user cannot actually escalate (not in sudoers/wheel/admin)
- Prompt for escalation if already root/admin — skip entirely
- Auto-resolve destructive uninstall without explicit user confirmation
- Leave data directories, config, or the system user behind on `--service --uninstall`
- Use `User=` in systemd unit — binary drops privileges after port binding internally
- Skip `ProtectSystem=strict`, `ProtectHome=yes`, `PrivateTmp=yes` in systemd unit
- Use external init scripts or system cron — binary handles its own lifecycle

## CRITICAL — ALWAYS DO

- Detect init system at runtime: systemd → OpenRC → SysVinit → launchd → rc.d → Windows Service
- Check escalation capability BEFORE prompting (sudo/wheel group, etc.)
- Drop privileges after port binding (Unix) — service starts as root, drops to `caspaste` user
- Confirm before `--service --uninstall` with destructive warning
- Install system service when root/admin, fall back to user service when not
- Keep service file and data on `--service --disable` (only stop + disable auto-start)
- Binary handles ALL user/group creation, directory setup, and permission management
- Support all service managers for the target platform

## Privilege Escalation Detection (PART 24)

| OS | Escalation Order |
|----|-----------------|
| Linux | Already root → sudo → su → pkexec → doas |
| macOS | Already root → sudo → osascript (GUI prompt) |
| BSD | Already root → doas → sudo → su |
| Windows | Already admin → UAC prompt → runas |

Binary checks: is EUID == 0? → yes: skip prompt. Is user in sudoers/wheel? → no: show info error, don't prompt.

## Service Install Logic

```
--service --install:
  If root/admin → install system service → enable → start
  If user → install user service (systemd --user / launchctl user agent) → enable → start
```

Binary handles user/group creation, directory setup, permissions at STARTUP — not at install.

## Service Uninstall (Requires Confirmation)

```
--service --uninstall:
  Stop → disable → remove service file
  Remove: {config_dir}, {data_dir}, {cache_dir}, {log_dir}, {backup_dir}, PID file
  Delete system user/group (if created by app)
  Binary stays — print: "Service uninstalled. Delete binary manually: rm {path}"
```

Confirmation prompt required: "This will delete ALL data, configs, and the system user. Continue? [y/N]"

## Service Disable (Non-Destructive)

```
--service --disable:
  Stop → disable auto-start
  Keep: service file, config, data, cache, logs, user/group
  Re-enable: run --service --install again
```

## Required Service Managers

| Platform | Service Managers |
|----------|-----------------|
| Linux | systemd, OpenRC, SysVinit (runit and s6 supported) |
| macOS | launchd |
| BSD | rc.d |
| Windows | Windows Service (NT SERVICE\caspaste virtual account) |

## systemd Unit Hardening (PART 25)

```ini
[Service]
Type=simple
ExecStart=/usr/local/bin/caspaste
Restart=on-failure
RestartSec=5
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
ReadWritePaths=/etc/casjay-forks/caspaste
ReadWritePaths=/var/lib/casjay-forks/caspaste
ReadWritePaths=/var/cache/casjay-forks/caspaste
ReadWritePaths=/var/log/casjay-forks/caspaste
```

Installation path: `/etc/systemd/system/caspaste.service`

## Privilege Drop (Unix)

Server starts as root to bind any port, then drops to `caspaste:caspaste` user after binding.
This allows any `--port` value without changing the service unit.
Exception: if IDEA.md explicitly requires permanent root, document it with justification.

For complete details, see AI.md PART 24, PART 25

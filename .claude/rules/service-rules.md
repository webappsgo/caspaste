# Service Rules (PART 24, 25)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- Never prompt for escalation if the user cannot actually escalate (not in sudoers/wheel/admin) — show an informative error instead
- Never use reserved/well-known UIDs/GIDs (see table) even if they look free on the current system
- Never assign different UID and GID values to the service account — they MUST match
- Never run permanently as root/Administrator unless IDEA.md explicitly approves it and the service file/docs say so
- Never skip built-in service support for any init system on a supported platform (systemd, OpenRC, SysVinit, runit, rc.d, launchd, Windows Service)
- Never require `--service --install` to also do user/dir/permission setup — that happens on normal server startup, not on install
- Never delete data on `--service --disable` (only `--uninstall` deletes data, and only after confirmation)
- Never uninstall without a `[y/N]` confirmation prompt (destroys config/data/cache/logs/user)
- Never use Local System, Administrator, or the logged-in user's account for a Windows service

## CRITICAL - ALWAYS DO
- Always check root/admin status first; only offer escalation methods the user can actually use
- Always follow OS-specific escalation order: Linux (root→sudo→su→pkexec→doas), macOS (root→sudo→osascript), BSD (root→doas→sudo→su), Windows (Administrator→UAC→runas)
- Always create dedicated service user/group `{internal_name}` with matching UID/GID in safe range (Linux/BSD: 200-899, macOS: 200-399), shell `/sbin/nologin`, no password
- Always start elevated only long enough to bind privileged ports, then drop privileges to `{internal_name}` (Unix); Windows always uses Virtual Service Account (`NT SERVICE\{internal_name}`), no drop needed
- Always create the home directory before creating the user, then set ownership
- Always support both service mode (root/admin, any port, drops privileges) and user mode (`$USER`, ports >1024 only, no drop)
- Always print "Delete binary manually: rm {binary_path}" after uninstall — binary itself is never removed automatically

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| Who creates the service user? | The server binary, during normal startup — not the `--service --install` flag | PART 24 § Service Installation Logic |
| What UID/GID range is safe? | Linux/BSD 200-899; macOS 200-399; never the reserved list | PART 24 § UID/GID Selection Logic |
| Does `--service --disable` delete data? | No — only stops/disables; data, config, user all remain | PART 24 § Service Disable Logic |
| Does `--service --uninstall` delete data? | Yes, all of it (config/data/cache/logs/backup/PID/user), after confirmation; binary stays | PART 24 § Service Uninstall Logic |
| What account do Windows services use? | Virtual Service Account (`NT SERVICE\{internal_name}`), auto-managed, no password | PART 24 § Windows Service Account |
| Which init systems must be supported? | systemd, OpenRC, SysVinit, runit, rc.d (BSD), launchd, Windows Service — ALL of them | PART 25 § Built-in Service Support |
| How is SysVinit chosen over OpenRC? | Only when `/sbin/openrc-run` and `systemctl` are both absent and `/etc/init.d` + `update-rc.d`/`chkconfig` work | PART 25 § SysVinit |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| Escalation | Gaining root/admin privileges via sudo/su/pkexec/doas/UAC/runas |
| Privilege drop | Binary lowers itself from root to `{internal_name}` after binding privileged ports |
| VSA | Virtual Service Account — Windows auto-managed, minimal-privilege service identity |
| System user | Passwordless, no-login account (`/sbin/nologin`) dedicated to running the service |
| Safe UID/GID range | 200-899 (Linux/BSD) or 200-399 (macOS) — avoids well-known service IDs |
| Service (escalated) mode | Runs as root/admin, can bind any port, drops privileges after binding |
| User mode | Runs as calling `$USER`, restricted to ports >1024, never drops (already unprivileged) |

## QUICK REFERENCE
- Escalation order — Linux: root → sudo → su → pkexec → doas
- Escalation order — macOS: root → sudo → osascript (GUI)
- Escalation order — BSD: root → doas → sudo → su
- Escalation order — Windows: Administrator → UAC → runas
- `--service --install`: detect init system → install/enable/start (system if root/admin, else user-level)
- `--service --uninstall`: stop → disable → remove unit file → delete all data dirs + PID → delete user/group → keep binary (confirm first)
- `--service --disable`: stop → disable auto-start → keep everything else
- Service user: `{internal_name}:{internal_name}`, matching UID/GID, `/sbin/nologin`, gecos `"{internal_name} service account"`
- Unix flow: root starts → binds port → drops to `{internal_name}` → serves
- Windows flow: always `NT SERVICE\{internal_name}` (VSA), no drop needed
- Init system files: systemd unit, OpenRC/SysVinit `/etc/init.d/{internal_name}`, runit `/etc/sv/{internal_name}/`, rc.d `/usr/local/etc/rc.d/{internal_name}`, launchd `/Library/LaunchDaemons/{plist_name}.plist`, Windows via `golang.org/x/sys/windows/svc`

---
For complete details, see AI.md PART 24, 25

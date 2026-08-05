# CasPaste

[![CI](https://github.com/webappsgo/caspaste/actions/workflows/ci.yml/badge.svg)](https://github.com/webappsgo/caspaste/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/webappsgo/caspaste)](https://github.com/webappsgo/caspaste/releases/latest)
[![License](https://img.shields.io/github/license/webappsgo/caspaste)](LICENSE.md)
[![Docker](https://img.shields.io/badge/Docker-ghcr.io-2496ED?logo=docker)](https://github.com/webappsgo/caspaste/pkgs/container/caspaste)

---

## About

CasPaste is a self-hosted, privacy-focused pastebin and code-sharing service
for sharing text snippets, code (with syntax highlighting), files, and short
URLs. It drops in for Pastebin, Microbin, Lenpaste, Stikked, Termbin,
Hastebin, and Sprunge — existing clients keep working after only changing the
hostname. Packaged as a single static binary with all assets embedded and
zero external runtime dependencies.

## Official Site

🌐 https://pste.us

## Features

- Pastebin / code-sharing with raw text, syntax highlighting, multi-file
  (Gist-style) pastes, and file uploads (5MB default, configurable)
- URL shortener — the server never fetches the destination (SSRF prevention)
- Anonymous, vanity (`/{username}/{paste_id}`), organization, and
  custom-domain paste URLs
- Visibility modes: public / unlisted / private, with password protection
  and flexible expiration (time-based or burn-after-read)
- Drop-in compatibility shims for sprunge, ix, Termbin, Stikked, Lenpaste,
  Microbin, Hastebin, and pastebin.com — detected per-request, response
  format matches the target service exactly
- Multi-user accounts, organizations, and custom domains
- Full server admin panel with setup-token first-run flow
- Built-in scheduler for backups, GeoIP/blocklist/CVE updates, log rotation,
  session cleanup, SSL renewal, and health checks
- Tor hidden service support (auto-enabled when `tor` is on `PATH`)
- Built-in metrics, GeoIP, email (SMTP), backup/restore, and in-process
  self-update
- Localization (multiple languages) and installable PWA support
- QR code generation for paste URLs and embeddable/iframe views
- 100% of features available free under the open-source license — no paid
  tiers, no feature gating, no phone-home

## Production

### Docker

```bash
docker compose up -d
```

**`docker/docker-compose.yml`** (production defaults):

```yaml
name: caspaste
services:
  caspaste:
    image: ghcr.io/webappsgo/caspaste:latest
    restart: always
    pull_policy: always
    ports:
      - "172.17.0.1:64580:80"
    volumes:
      - ./volumes/config:/config:z
      - ./volumes/data:/data:z
    environment:
      TZ: America/New_York
```

**With PostgreSQL:**

```yaml
name: caspaste
services:
  caspaste:
    image: ghcr.io/webappsgo/caspaste:latest
    restart: always
    pull_policy: always
    ports:
      - "172.17.0.1:64580:80"
    volumes:
      - ./volumes/config:/config:z
      - ./volumes/data:/data:z
    environment:
      TZ: America/New_York
      CASPASTE_DB_DRIVER: postgres
      CASPASTE_DB_SOURCE: postgres://caspaste:changeme@caspaste-db:5432/caspaste?sslmode=disable

  caspaste-db:
    image: postgres:16-alpine
    restart: always
    pull_policy: always
    environment:
      POSTGRES_DB: caspaste
      POSTGRES_USER: caspaste
      POSTGRES_PASSWORD: changeme
    volumes:
      - ./volumes/data/db/postgres/caspaste:/var/lib/postgresql/data:z
```

### Binary

Download the latest release from [GitHub Releases](https://github.com/webappsgo/caspaste/releases/latest).

| OS | Arch | Binary |
|----|------|--------|
| Linux | amd64 / arm64 | `caspaste-linux-{amd64,arm64}` |
| macOS | amd64 / arm64 | `caspaste-darwin-{amd64,arm64}` |
| Windows | amd64 / arm64 | `caspaste-windows-{amd64,arm64}.exe` |
| FreeBSD | amd64 / arm64 | `caspaste-freebsd-{amd64,arm64}` |

```bash
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -LSsf "https://github.com/webappsgo/caspaste/releases/latest/download/caspaste-linux-${ARCH}" \
  -o /usr/local/bin/caspaste && chmod +x /usr/local/bin/caspaste
```

macOS also requires clearing the quarantine attribute:

```bash
xattr -d com.apple.quarantine /usr/local/bin/caspaste 2>/dev/null || true
```

```bash
# Start server (auto-generates config on first run)
caspaste

# Specify directories
caspaste --port 8080 \
  --data /var/lib/casapps/caspaste \
  --config /etc/casapps/caspaste
```

### Service Management

```bash
# Install as system service (auto-detects systemd / launchd / Windows Service / rc.d)
sudo caspaste --service install
sudo caspaste --service start
sudo caspaste --service stop
sudo caspaste --service status
sudo caspaste --service uninstall
```

### Health Check

```bash
caspaste --status
# Exit codes: 0=healthy, 1=unhealthy/degraded
```

### Backup & Restore

```bash
caspaste --maintenance backup
caspaste --maintenance restore
```

## Client

The CLI client is released alongside the server under the same naming
convention (`caspaste-cli-{os}-{arch}`). Example for Linux:

```bash
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -LSsf "https://github.com/webappsgo/caspaste/releases/latest/download/caspaste-cli-linux-${ARCH}" \
  -o /usr/local/bin/caspaste-cli && chmod +x /usr/local/bin/caspaste-cli
```

```bash
# Point at your instance
export CASPASTE_SERVER=https://pste.us

# Create a paste from stdin
echo "Hello World" | caspaste-cli new

# Create from a file with syntax highlighting
caspaste-cli new -f script.py -s python

# Get a paste
caspaste-cli get abc123

# List recent pastes
caspaste-cli list
```

| Flag | Description |
|------|-------------|
| `--server`, `-s` | Server URL (env: `CASPASTE_SERVER`) |
| `--token`, `-t` | API token (env: `CASPASTE_TOKEN`) |
| `--syntax` | Syntax language (default: auto-detect) |
| `--private` | Create as private paste |
| `--one-use` | Burn after reading |
| `--expire` | Expiration (e.g. `1h`, `7d`, `never`) |

## Configuration

Configuration is auto-generated on first run. Command-line flags and
environment variables initialize the server; the generated `server.yml` is
the source of truth for subsequent runs.

### Key Environment Variables

| Variable | Description |
|----------|-------------|
| `CASPASTE_ADDRESS` | Listen address (e.g. `:8080`, `0.0.0.0:80`) |
| `CASPASTE_PORT` | Listen port |
| `CASPASTE_PUBLIC` | `true` = open, `false` = password required |
| `CASPASTE_CONFIG_DIR` | Config directory |
| `CASPASTE_DATA_DIR` | Data directory |
| `CASPASTE_DB_DIR` | Database directory |
| `CASPASTE_BACKUP_DIR` | Backup directory |
| `CASPASTE_DB_DRIVER` | `sqlite` (default), `postgres`, `mysql` |
| `CASPASTE_DB_SOURCE` | Connection string or SQLite filename |
| `MODE` | `production` (default) or `development` |
| `DEBUG` | `true` enables debug output and `/debug/*` endpoints |
| `DOMAIN` | Public FQDN clients use to reach the server |
| `BASE_URL` | Base path when served behind a sub-path proxy (default `/`) |
| `NO_COLOR` | Any non-empty value disables colored/emoji output |
| `TZ` | Container/process time zone |
| `SMTP_HOST` / `SMTP_PORT` | SMTP server host and port |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | SMTP authentication credentials |
| `SMTP_FROM_NAME` / `SMTP_FROM_EMAIL` | Sender identity for outbound email |
| `SMTP_TLS` | Enable TLS for the SMTP connection |

### Authentication (Private Mode)

```bash
# Require password for all access
CASPASTE_PUBLIC=false caspaste
```

On first run in private mode, a setup token is printed to stdout. Visit
`/server/admin/config/setup` to create the admin account.

### Platform-Specific Directories

| Directory | Linux (root) | Linux (user) | macOS |
|-----------|-------------|--------------|-------|
| Config | `/etc/casapps/caspaste` | `~/.config/casapps/caspaste` | `~/Library/Application Support/CasPaste/Config` |
| Data | `/var/lib/casapps/caspaste` | `~/.local/share/casapps/caspaste` | `~/Library/Application Support/CasPaste` |
| Logs | `/var/log/casapps/caspaste` | `~/.local/log/casapps/caspaste` | `~/Library/Logs/CasPaste` |

## API

Base URL: `https://pste.us/api/v1`
Full docs: `/docs/apiv1`

### Create Paste

```bash
curl -X POST https://pste.us/api/v1/pastes \
  -H "Content-Type: application/json" \
  -d '{"body":"Hello World","syntax":"plaintext"}'
```

### Get Paste

```bash
curl https://pste.us/api/v1/pastes/abc12345
```

### List Pastes

```bash
curl https://pste.us/api/v1/pastes
```

### Health Check

```bash
curl https://pste.us/api/v1/server/healthz
```

### External API Compatibility

Existing clients for other paste services work without modification — just
change the endpoint URL:

| Service | Mode detection |
|---------|---------------|
| sprunge.us | Always active (POST `/sprunge`) |
| ix.io | Always active (POST `/ix`) |
| termbin.com | Always active (POST `/termbin`) |
| hastebin | Host `haste.*` or `CASPASTE_API_MODE=hastebin` |
| pastebin.com | Host `pb.*` or `CASPASTE_API_MODE=pastebin` |
| stikked | Host `sk.*` or `CASPASTE_API_MODE=stikked` |
| microbin | Host `mb.*` or `CASPASTE_API_MODE=microbin` |
| lenpaste | Host `lp.*` or `CASPASTE_API_MODE=lenpaste` |

## Other

Additional documentation lives under [`docs/`](docs/):

- [Admin Panel](docs/admin.md) — server settings, users, and moderation
- [Security](docs/security.md) — hardening notes and vulnerability reporting
- [Integrations](docs/integrations.md) — compatibility shims and third-party
  clients

## Development

```bash
git clone https://github.com/webappsgo/caspaste.git
cd caspaste
```

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make dev` | Quick build to temp dir (no version info) |
| `make local` | Build for current platform only |
| `make build` | Build all 8 platforms to `binaries/` |
| `make test` | Run unit tests with ≥80% coverage gate |
| `make release` | Build + create GitHub release |
| `make docker` | Build and push multi-arch Docker image |
| `make clean` | Remove build artifacts |

All builds run inside Docker (`casjaysdev/go:latest`) — no local Go
installation required.

### Supported Platforms

| OS | Architectures |
|----|---------------|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64, arm64 |
| FreeBSD | amd64, arm64 |

### Docker Build

```bash
# Build the image locally
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=$(cat release.txt) \
  -f docker/Dockerfile \
  -t ghcr.io/webappsgo/caspaste:latest \
  --push .
```

## Disclaimer

This software is provided "as is" without warranty of any kind. Use at your
own risk.

- **No Warranty**: The authors are not responsible for any damages, data
  loss, or issues arising from use of this software
- **Not Professional Advice**: This software does not constitute legal,
  financial, medical, or other professional advice
- **Third-Party Services**: If this software connects to external APIs or
  services, their terms of service apply separately
- **Security**: While we strive to follow security best practices, no
  software is guaranteed to be free of vulnerabilities
- **Production Use**: Evaluate thoroughly before deploying in production
  environments

By using this software, you acknowledge that you have read and understood
this disclaimer.

## License

MIT — see [LICENSE.md](LICENSE.md)

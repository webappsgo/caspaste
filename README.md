# CasPaste

A self-hosted, privacy-focused pastebin service for sharing text snippets, code, files, and short URLs. Single static binary, all assets embedded, zero external runtime dependencies.

🌐 **Site:** https://pste.us

[![CI](https://github.com/casjay-forks/caspaste/actions/workflows/ci.yml/badge.svg)](https://github.com/casjay-forks/caspaste/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/casjay-forks/caspaste)](https://github.com/casjay-forks/caspaste/releases/latest)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE.md)
[![Docker](https://img.shields.io/badge/Docker-ghcr.io-2496ED?logo=docker)](https://github.com/casjay-forks/caspaste/pkgs/container/caspaste)

---

## 📦 Install

Download the latest release from [GitHub Releases](https://github.com/casjay-forks/caspaste/releases/latest).

### Linux

| Arch | Binary |
|------|--------|
| amd64 | `caspaste-linux-amd64` |
| arm64 | `caspaste-linux-arm64` |

```bash
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -LSsf "https://github.com/casjay-forks/caspaste/releases/latest/download/caspaste-linux-${ARCH}" \
  -o /usr/local/bin/caspaste && chmod +x /usr/local/bin/caspaste
```

### macOS

| Arch | Binary |
|------|--------|
| Intel (x86_64) | `caspaste-darwin-amd64` |
| Apple Silicon (arm64) | `caspaste-darwin-arm64` |

```bash
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -LSsf "https://github.com/casjay-forks/caspaste/releases/latest/download/caspaste-darwin-${ARCH}" \
  -o /usr/local/bin/caspaste && chmod +x /usr/local/bin/caspaste
xattr -d com.apple.quarantine /usr/local/bin/caspaste 2>/dev/null || true
```

### Windows

| Arch | Binary |
|------|--------|
| amd64 | `caspaste-windows-amd64.exe` |
| arm64 | `caspaste-windows-arm64.exe` |

Download and add to `%PATH%`.

### FreeBSD

| Arch | Binary |
|------|--------|
| amd64 | `caspaste-freebsd-amd64` |
| arm64 | `caspaste-freebsd-arm64` |

```bash
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -LSsf "https://github.com/casjay-forks/caspaste/releases/latest/download/caspaste-freebsd-${ARCH}" \
  -o /usr/local/bin/caspaste && chmod +x /usr/local/bin/caspaste
```

### CLI Client (caspaste-cli)

The CLI client is released alongside the server under the same naming convention (`caspaste-cli-{os}-{arch}`). Example for Linux:

```bash
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -LSsf "https://github.com/casjay-forks/caspaste/releases/latest/download/caspaste-cli-linux-${ARCH}" \
  -o /usr/local/bin/caspaste-cli && chmod +x /usr/local/bin/caspaste-cli
```

---

## 🐳 Docker

```bash
docker compose up -d
```

**`docker/docker-compose.yml`** (production defaults):

```yaml
name: caspaste
services:
  caspaste:
    image: ghcr.io/casjay-forks/caspaste:latest
    restart: always
    pull_policy: always
    ports:
      - "172.17.0.1:64580:80"
    volumes:
      - ./volumes/config:/config:z
      - ./volumes/data:/data:z
    environment:
      - TZ=America/New_York
      - MODE=production
```

**With PostgreSQL:**

```yaml
name: caspaste
services:
  caspaste:
    image: ghcr.io/casjay-forks/caspaste:latest
    restart: always
    pull_policy: always
    ports:
      - "172.17.0.1:64580:80"
    volumes:
      - ./volumes/config:/config:z
      - ./volumes/data:/data:z
    environment:
      - TZ=America/New_York
      - MODE=production
      - CASPASTE_DB_DRIVER=postgres
      - CASPASTE_DB_SOURCE=postgres://caspaste:changeme@caspaste-db:5432/caspaste?sslmode=disable

  caspaste-db:
    image: postgres:16-alpine
    restart: always
    pull_policy: always
    environment:
      - POSTGRES_DB=caspaste
      - POSTGRES_USER=caspaste
      - POSTGRES_PASSWORD=changeme
    volumes:
      - ./volumes/data/db/postgres/caspaste:/var/lib/postgresql/data:z
```

---

## 🖥️ CLI Client

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

---

## 🤖 Server

```bash
# Start server (auto-generates config on first run)
caspaste

# Specify directories
caspaste --port 8080 \
  --data /var/lib/casjay-forks/caspaste \
  --config /etc/casjay-forks/caspaste
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
# Exit codes: 0=healthy, 1=unhealthy, 2=degraded
```

### Backup & Restore

```bash
caspaste --maintenance backup
caspaste --maintenance restore
```

---

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

Existing clients for other paste services work without modification — just change the endpoint URL:

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

---

## Configuration

Configuration is auto-generated on first run. Command-line flags and environment variables initialize the server; the generated `server.yml` is the source of truth for subsequent runs.

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

### Authentication (Private Mode)

```bash
# Require password for all access
CASPASTE_PUBLIC=false caspaste
```

On first run in private mode, a setup token is printed to stdout. Visit `/server/admin/config/setup` to create the admin account.

### Platform-Specific Directories

| Directory | Linux (root) | Linux (user) | macOS |
|-----------|-------------|--------------|-------|
| Config | `/etc/casjay-forks/caspaste` | `~/.config/casjay-forks/caspaste` | `~/Library/Application Support/CasPaste/Config` |
| Data | `/var/lib/casjay-forks/caspaste` | `~/.local/share/casjay-forks/caspaste` | `~/Library/Application Support/CasPaste` |
| Logs | `/var/log/casjay-forks/caspaste` | `~/.local/log/casjay-forks/caspaste` | `~/Library/Logs/CasPaste` |

---

## 🛠️ Development

```bash
git clone https://github.com/casjay-forks/caspaste.git
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

All builds run inside Docker (`casjaysdev/go:latest`) — no local Go installation required.

### Supported Platforms

| OS | Architectures |
|----|---------------|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64, arm64 |
| FreeBSD | amd64, arm64 |

### 🐳 Docker Build

```bash
# Build the image locally
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=$(cat release.txt) \
  -f docker/Dockerfile \
  -t ghcr.io/casjay-forks/caspaste:latest \
  --push .
```

---

## 📄 License

MIT — see [LICENSE.md](LICENSE.md)

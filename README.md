# CasPb

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
| amd64 | `caspb-linux-amd64` |
| arm64 | `caspb-linux-arm64` |

```bash
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -LSsf "https://github.com/casjay-forks/caspaste/releases/latest/download/caspb-linux-${ARCH}" \
  -o /usr/local/bin/caspb && chmod +x /usr/local/bin/caspb
```

### macOS

| Arch | Binary |
|------|--------|
| Intel (x86_64) | `caspb-darwin-amd64` |
| Apple Silicon (arm64) | `caspb-darwin-arm64` |

```bash
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -LSsf "https://github.com/casjay-forks/caspaste/releases/latest/download/caspb-darwin-${ARCH}" \
  -o /usr/local/bin/caspb && chmod +x /usr/local/bin/caspb
xattr -d com.apple.quarantine /usr/local/bin/caspb 2>/dev/null || true
```

### Windows

| Arch | Binary |
|------|--------|
| amd64 | `caspb-windows-amd64.exe` |
| arm64 | `caspb-windows-arm64.exe` |

Download and add to `%PATH%`.

### FreeBSD

| Arch | Binary |
|------|--------|
| amd64 | `caspb-freebsd-amd64` |
| arm64 | `caspb-freebsd-arm64` |

```bash
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -LSsf "https://github.com/casjay-forks/caspaste/releases/latest/download/caspb-freebsd-${ARCH}" \
  -o /usr/local/bin/caspb && chmod +x /usr/local/bin/caspb
```

### CLI Client (caspb-cli)

The CLI client is released alongside the server under the same naming convention (`caspb-cli-{os}-{arch}`). Example for Linux:

```bash
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -LSsf "https://github.com/casjay-forks/caspaste/releases/latest/download/caspb-cli-linux-${ARCH}" \
  -o /usr/local/bin/caspb-cli && chmod +x /usr/local/bin/caspb-cli
```

---

## 🐳 Docker

```bash
docker compose up -d
```

**`docker/docker-compose.yml`** (production defaults):

```yaml
name: caspb
services:
  caspb:
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
name: caspb
services:
  caspb:
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
      - CASPB_DB_DRIVER=postgres
      - CASPB_DB_SOURCE=postgres://caspb:changeme@caspb-db:5432/caspb?sslmode=disable

  caspb-db:
    image: postgres:16-alpine
    restart: always
    pull_policy: always
    environment:
      - POSTGRES_DB=caspb
      - POSTGRES_USER=caspb
      - POSTGRES_PASSWORD=changeme
    volumes:
      - ./volumes/data/db/postgres/caspb:/var/lib/postgresql/data:z
```

---

## 🖥️ CLI Client

```bash
# Point at your instance
export CASPB_SERVER=https://pste.us

# Create a paste from stdin
echo "Hello World" | caspb-cli new

# Create from a file with syntax highlighting
caspb-cli new -f script.py -s python

# Get a paste
caspb-cli get abc123

# List recent pastes
caspb-cli list
```

| Flag | Description |
|------|-------------|
| `--server`, `-s` | Server URL (env: `CASPB_SERVER`) |
| `--token`, `-t` | API token (env: `CASPB_TOKEN`) |
| `--syntax` | Syntax language (default: auto-detect) |
| `--private` | Create as private paste |
| `--one-use` | Burn after reading |
| `--expire` | Expiration (e.g. `1h`, `7d`, `never`) |

---

## 🤖 Server

```bash
# Start server (auto-generates config on first run)
caspb

# Specify directories
caspb --port 8080 \
  --data /var/lib/casapps/caspb \
  --config /etc/casapps/caspb
```

### Service Management

```bash
# Install as system service (auto-detects systemd / launchd / Windows Service / rc.d)
sudo caspb --service install
sudo caspb --service start
sudo caspb --service stop
sudo caspb --service status
sudo caspb --service uninstall
```

### Health Check

```bash
caspb --status
# Exit codes: 0=healthy, 1=unhealthy, 2=degraded
```

### Backup & Restore

```bash
caspb --maintenance backup
caspb --maintenance restore
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
| hastebin | Host `haste.*` or `CASPB_API_MODE=hastebin` |
| pastebin.com | Host `pb.*` or `CASPB_API_MODE=pastebin` |
| stikked | Host `sk.*` or `CASPB_API_MODE=stikked` |
| microbin | Host `mb.*` or `CASPB_API_MODE=microbin` |
| lenpaste | Host `lp.*` or `CASPB_API_MODE=lenpaste` |

---

## Configuration

Configuration is auto-generated on first run. Command-line flags and environment variables initialize the server; the generated `server.yml` is the source of truth for subsequent runs.

### Key Environment Variables

| Variable | Description |
|----------|-------------|
| `CASPB_ADDRESS` | Listen address (e.g. `:8080`, `0.0.0.0:80`) |
| `CASPB_PORT` | Listen port |
| `CASPB_PUBLIC` | `true` = open, `false` = password required |
| `CASPB_CONFIG_DIR` | Config directory |
| `CASPB_DATA_DIR` | Data directory |
| `CASPB_DB_DIR` | Database directory |
| `CASPB_BACKUP_DIR` | Backup directory |
| `CASPB_DB_DRIVER` | `sqlite` (default), `postgres`, `mysql` |
| `CASPB_DB_SOURCE` | Connection string or SQLite filename |

### Authentication (Private Mode)

```bash
# Require password for all access
CASPB_PUBLIC=false caspb
```

On first run in private mode, a setup token is printed to stdout. Visit `/server/admin/config/setup` to create the admin account.

### Platform-Specific Directories

| Directory | Linux (root) | Linux (user) | macOS |
|-----------|-------------|--------------|-------|
| Config | `/etc/casapps/caspb` | `~/.config/casapps/caspb` | `~/Library/Application Support/CasPb/Config` |
| Data | `/var/lib/casapps/caspb` | `~/.local/share/casapps/caspb` | `~/Library/Application Support/CasPb` |
| Logs | `/var/log/casapps/caspb` | `~/.local/log/casapps/caspb` | `~/Library/Logs/CasPb` |

---

## 🛠️ Development

```bash
git clone https://github.com/casjay-forks/caspaste.git
cd caspb
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

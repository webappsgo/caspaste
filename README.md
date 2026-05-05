# CasPaste

A self-hosted, privacy-focused pastebin service for sharing text snippets, files, and short URLs.

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE.md)
[![Docker](https://img.shields.io/badge/Docker-ghcr.io-2496ED?logo=docker)](https://ghcr.io/casjay-forks/caspaste)
[![Demo](https://img.shields.io/badge/Demo-lp.pste.us-green)](https://lp.pste.us)

## About

CasPaste is a modern, secure pastebin service designed for self-hosting. It prioritizes privacy, security, and ease of deployment.

**Demo:** https://lp.pste.us

### Key Features

| Feature | Description |
|---------|-------------|
| **Privacy-First** | No registration required, anonymous sharing, private pastes |
| **Secure** | Argon2id hashing, brute force protection, XSS prevention |
| **Modern UI** | Mobile-friendly, syntax highlighting, 12+ themes |
| **File Uploads** | Share images, documents, any file type (50MB max) |
| **URL Shortener** | Create short links with QR codes |
| **Editable Pastes** | Update pastes after creation |
| **Burn After Reading** | One-time view pastes auto-delete after viewing |
| **API-Ready** | RESTful API with listing, upload, URL shortening |
| **Multi-Database** | SQLite, PostgreSQL, MySQL/MariaDB |
| **Multi-Platform** | Linux, macOS, Windows, BSD (amd64 + arm64) |
| **Single Binary** | Static binary with all assets embedded |

---

## Production Deployment

### Docker (Recommended)

```bash
docker run -d \
  --name caspaste \
  -p 172.17.0.1:59093:80 \
  -v ./volumes/config:/config \
  -v ./volumes/data:/data \
  -v ./volumes/backups:/data/backups \
  ghcr.io/casjay-forks/caspaste:latest
```

Access at: `http://172.17.0.1:59093`

### Docker Compose

```yaml
version: "3.8"
services:
  caspaste:
    image: ghcr.io/casjay-forks/caspaste:latest
    ports:
      - "172.17.0.1:59093:80"
    volumes:
      - ./volumes/config:/config
      - ./volumes/data:/data
      - ./volumes/backups:/data/backups
    environment:
      - TZ=America/New_York
```

### Docker with PostgreSQL

```yaml
version: "3.8"
services:
  caspaste:
    image: ghcr.io/casjay-forks/caspaste:latest
    ports:
      - "172.17.0.1:59093:80"
    volumes:
      - ./volumes/config:/config
      - ./volumes/data:/data
      - ./volumes/backups:/data/backups
    environment:
      - CASPASTE_DB_DRIVER=postgres
      - CASPASTE_DB_SOURCE=postgres://caspaste:changeme@postgres:5432/caspaste?sslmode=disable
    depends_on:
      - postgres

  postgres:
    image: postgres:16-alpine
    environment:
      - POSTGRES_DB=caspaste
      - POSTGRES_USER=caspaste
      - POSTGRES_PASSWORD=changeme
    volumes:
      - postgres-data:/var/lib/postgresql/data

volumes:
  postgres-data:
```

### Binary Installation

```bash
# Download latest release
wget https://github.com/casjay-forks/caspaste/releases/latest/download/caspaste-linux-amd64
chmod +x caspaste-linux-amd64
sudo mv caspaste-linux-amd64 /usr/local/bin/caspaste

# Run (auto-generates config on first run)
caspaste

# Or specify directories
caspaste --port 8080 --data /var/lib/casjay-forks/caspaste --config /etc/casjay-forks/caspaste
```

### Service Management

```bash
# Install as service (auto-detects platform)
sudo caspaste --service install

# Manage
sudo caspaste --service start
sudo caspaste --service stop
sudo caspaste --service restart
sudo caspaste --service status

# Uninstall
sudo caspaste --service uninstall
```

| Platform | Service Type |
|----------|--------------|
| Linux | systemd |
| macOS | launchd |
| Windows | Windows Service |
| BSD | rc.d |

### Health Monitoring

```bash
caspaste --status
# Exit codes: 0=healthy, 1=unhealthy, 2=degraded
```

### Backup & Restore

```bash
# Create backup
caspaste --maintenance backup

# Restore latest
caspaste --maintenance restore

# Restore specific
caspaste --maintenance "restore backup-20240101-120000.tar.gz"
```

### Authentication

CasPaste is **open and public by default** (`server.public: true`).

To require authentication:

```bash
# Via environment
docker run -d -e CASPASTE_PUBLIC=false ghcr.io/casjay-forks/caspaste:latest

# Via config file
# server:
#   public: false
```

On first start with `public: false`, admin credentials are auto-generated:

```
╔════════════════════════════════════════════════════════════╗
║  CasPaste                                                  ║
╠════════════════════════════════════════════════════════════╣
║  Mode:        Private (authentication required)            ║
║  Username:    admin                                        ║
║  Password:    eoYBn7I9Z&ZHGqCY                             ║
║  SAVE THESE CREDENTIALS - shown only once!                 ║
╚════════════════════════════════════════════════════════════╝
```

### Database Backends

**SQLite (Default)**
```bash
caspaste --data /var/lib/caspaste
# Database: /var/lib/caspaste/db/caspaste.db
```

**PostgreSQL**
```bash
caspaste --db-driver postgres \
  --db-source "postgres://user:pass@localhost:5432/caspaste?sslmode=require"
```

**MariaDB/MySQL**
```bash
caspaste --db-driver mysql \
  --db-source "user:pass@tcp(localhost:3306)/caspaste?charset=utf8mb4&parseTime=true"
```

---

## CLI Client

```bash
# Install
wget https://github.com/casjay-forks/caspaste/releases/latest/download/caspaste-cli-linux-amd64
chmod +x caspaste-cli-linux-amd64
sudo mv caspaste-cli-linux-amd64 /usr/local/bin/caspaste-cli

# Configure
caspaste-cli login

# Create paste
echo "Hello" | caspaste-cli new
caspaste-cli new -f script.py -s python

# Get paste
caspaste-cli get abc123

# List pastes
caspaste-cli list
```

---

## API Usage

Full API documentation: `/docs/apiv1`

### Create Paste

```bash
curl -X POST https://paste.example.com/api/v1/new \
  -d "body=Hello World" \
  -d "syntax=plaintext"
```

### Upload File

```bash
curl -X POST https://paste.example.com/api/v1/new \
  -F "file=@image.png"
```

### Create Short URL

```bash
curl -X POST https://paste.example.com/api/v1/new \
  -d "url=true" \
  -d "originalURL=https://example.com/long/url"
```

### Burn After Reading

```bash
curl -X POST https://paste.example.com/api/v1/new \
  -d "body=Secret" \
  -d "oneUse=true"
```

---

## Configuration

CasPaste auto-generates configuration on first run. Command-line flags and environment variables are used to **initialize** the server. Once initialized, the **config file becomes the source of truth** for subsequent runs.

### Initialization Priority

On first run, settings are resolved in this order:

1. **Command-line flags** (highest priority)
2. **Environment variables** (`CASPASTE_*` prefix)
3. **Platform-specific defaults** (lowest priority)

After initialization, the resolved values are saved to `server.yml` and used for all future runs.

### Platform-Specific Directories

Directories are automatically determined based on the runtime platform and privilege level:

| Directory | Linux (root) | Linux (user) | macOS (user) | Windows |
|-----------|--------------|--------------|--------------|---------|
| **Config** | `/etc/casjay-forks/caspaste` | `~/.config/casjay-forks/caspaste` | `~/Library/Application Support/CasPaste/Config` | `%LOCALAPPDATA%\CasPaste\Config` |
| **Data** | `/var/lib/casjay-forks/caspaste` | `~/.local/share/casjay-forks/caspaste` | `~/Library/Application Support/CasPaste` | `%LOCALAPPDATA%\CasPaste\Data` |
| **Database** | `/var/lib/casjay-forks/caspaste/db` | `~/.local/share/casjay-forks/caspaste/db` | `~/Library/Application Support/CasPaste/db` | `%LOCALAPPDATA%\CasPaste\Data\db` |
| **Logs** | `/var/log/casjay-forks/caspaste` | `~/.local/log/casjay-forks/caspaste` | `~/Library/Logs/CasPaste` | `%LOCALAPPDATA%\CasPaste\Logs` |
| **Backup** | `/var/backups/casjay-forks/caspaste` | `~/.local/share/casjay-forks/caspaste/backups` | `~/Library/Application Support/CasPaste/Backups` | `%APPDATA%\CasPaste\Backups` |
| **Cache** | `/var/cache/casjay-forks/caspaste` | `~/.cache/casjay-forks/caspaste` | `~/Library/Caches/CasPaste` | `%LOCALAPPDATA%\CasPaste\Cache` |

### Auto-Generated Values

On first run, CasPaste automatically generates and persists:

| Setting | Behavior |
|---------|----------|
| **Port** | Finds first available port in range 64000-65535 |
| **UID/GID** | Finds first available UID/GID in range 200-900 (Unix only) |
| **Directories** | Creates platform-specific directories |
| **Config file** | Generates `server.yml` with all resolved values |

### Config File Structure

```yaml
server:
  public: true                    # true = open, false = auth required
  fqdn: ""                        # Empty = auto-detect from headers/hostname
  listen: all                     # all, ::, 0.0.0.0, or specific IP
  port: ""                        # Empty = auto-detect available port
  title: CasPaste
  tagline: A simple paste service
  description: CasPaste is a simple, fast, and secure paste service
  proxy:
    allowed: []                   # Additional trusted proxies (appended to defaults)
  administrator:
    name: CasPaste Administrator
    email: administrator@{fqdn}   # {fqdn} replaced at runtime
    from: '"CasPaste" <no-reply@{fqdn}>'
  timeouts:
    read: 15
    write: 15
    idle: 60

database:
  driver: sqlite                  # sqlite, postgres, mysql
  source: caspaste.db             # Connection string or filename
  max_open_conns: 25
  max_idle_conns: 5
  cleanup_period: 1m

web:
  ui:
    default_lifetime: never
    default_theme: dark
    themes_dir: ""                # Empty = {data_dir}/web/themes
  content:
    about: ""                     # Empty = auto-generated
    rules: ""                     # Empty = auto-generated
    terms: ""                     # Empty = auto-generated
    security: ""                  # Empty = auto-generated security.txt
  branding:
    logo: ""                      # Path or URL
    favicon: ""                   # Path or URL
  security:
    contact:
      email: security@{fqdn}
      name: Security Team

directories:
  data: /var/lib/casjay-forks/caspaste         # Auto-set based on platform
  config: /etc/casjay-forks/caspaste           # Auto-set based on platform
  db: /var/lib/casjay-forks/caspaste/db        # Auto-set based on platform
  cache: /var/cache/casjay-forks/caspaste      # Auto-set based on platform
  logs: /var/log/casjay-forks/caspaste         # Auto-set based on platform

logging:
  level: info                     # info, warn, error
  access:
    stdout: false
    stderr: false
    format: apache                # apache, nginx, text, json
    file: access.log
  error:
    stdout: false
    stderr: true
    format: text
    file: error.log
  server:
    stdout: true
    stderr: false
    format: text
    file: caspaste.log
  debug:
    stdout: true
    stderr: false
    format: text
    file: debug.log
```

### Placeholder Variables

These placeholders are replaced at runtime:

| Placeholder | Replaced With |
|-------------|---------------|
| `{fqdn}` | Actual FQDN from config or auto-detected from headers |
| `{data_dir}` | Resolved data directory path |
| `{config_dir}` | Resolved config directory path |

### Trusted Proxies

Private network ranges are **always trusted** for `X-Forwarded-*` headers:
- `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16` (RFC1918)
- `127.0.0.0/8`, `::1` (loopback)
- `fc00::/7`, `fe80::/10` (IPv6 private/link-local)

Any CIDRs in `server.proxy.allowed` are **appended** to these defaults.

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `CASPASTE_ADDRESS` | Smart address parsing | `:8080`, `paste.example.com:80` |
| `CASPASTE_CONFIG_DIR` | Config directory | `/config/caspaste` |
| `CASPASTE_DATA_DIR` | Data directory | `/data/caspaste` |
| `CASPASTE_DB_DIR` | Database directory | `/data/db/sqlite` |
| `CASPASTE_LOGS_DIR` | Logs directory | `/data/log/caspaste` |
| `CASPASTE_BACKUP_DIR` | Backup directory | `/data/backups` |
| `CASPASTE_PUBLIC` | Public instance | `true`, `false` |
| `PORT` | Port (Docker/PaaS) | `80` |

### Themes

Built-in themes: `dracula`, `nord`, `gruvbox-dark`, `tokyo-night`, `catppuccin-mocha`, `one-dark`, `github-light`, `nord-light`, `gruvbox-light`, `catppuccin-latte`, `solarized-light`

```yaml
web:
  ui:
    default_theme: nord
```

### Security Features

| Feature | Description |
|---------|-------------|
| **Argon2id Hashing** | OWASP-recommended, memory-hard algorithm |
| **Brute Force Protection** | 5 failed attempts = 15-minute lockout |
| **Secure Sessions** | HttpOnly, SameSite, auto-detect HTTPS |
| **Session Expiry** | 24-hour auto-expire |

---

## Development

### Building from Source

```bash
git clone https://github.com/casjay-forks/caspaste.git
cd caspaste

# Build for current platform (fast)
make local

# Build for all platforms
make build

# Run tests
make test
```

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build all binaries for all OS/arch (`./binaries/`) |
| `make release` | Build production binaries and create GitHub release |
| `make docker` | Build and push Docker images to ghcr.io (multi-arch) |
| `make test` | Run all tests |
| `make local` | Build for current OS/arch only (fast) |

### Version Management

Version is determined by (in order of priority):
1. `VERSION` environment variable
2. `release.txt` file
3. Git tag
4. Default: `1.0.0`

```bash
# Build with specific version
VERSION=2.0.0 make build
```

### Supported Platforms

| OS | Architectures |
|----|---------------|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64, arm64 |
| FreeBSD | amd64, arm64 |
| OpenBSD | amd64, arm64 |

---

## License

MIT License - see [LICENSE.md](LICENSE.md)

## Support

- **Demo:** https://lp.pste.us
- **API Docs:** https://lp.pste.us/docs/apiv1
- **Issues:** https://github.com/casjay-forks/caspaste/issues

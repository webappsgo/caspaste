# CasPaste - Claude Code Guidelines

This document provides guidance for AI assistants working with the CasPaste codebase.

## Terminology

- **USER MACHINE** = **Runtime Machine** - The machine where the code will execute (the user's local development environment or deployment target)

## Project Overview

CasPaste is a self-hosted, privacy-focused pastebin service written in Go 1.23+. It's a single static binary with all assets embedded, supporting SQLite, PostgreSQL, and MySQL databases.

**Demo:** https://lp.pste.us

## Quick Reference

### Build Commands

```bash
make local    # Build for current OS/arch (fast development)
make build    # Build all binaries for all platforms (./binaries/)
make test     # Run all tests with coverage
make release  # Build production binaries and create GitHub release
make docker   # Build and push multi-arch Docker images
```

All Go builds run inside Docker (`golang:alpine`) for consistency.

### Run Locally

```bash
./binaries/caspaste              # Run server
./binaries/caspaste-cli          # CLI client
```

### Version Management

Version is determined by (in priority order):
1. `VERSION` environment variable
2. `release.txt` file
3. Git tag
4. Default: `1.0.0`

```bash
VERSION=2.0.0 make build   # Build with specific version
```

## Project Structure

```
caspaste/
├── src/
│   ├── server/                 # Main server entry point
│   ├── client/                 # CLI client entry point
│   ├── apiv1/                  # REST API v1 handlers
│   ├── web/                    # Web UI handlers and templates
│   │   └── data/               # Embedded assets (templates, JS, CSS, themes)
│   ├── storage/                # Database abstraction layer
│   ├── config/                 # Configuration management
│   ├── cli/                    # CLI argument parsing
│   ├── logger/                 # Structured logging
│   ├── validation/             # Input validation
│   ├── caspasswd/              # Authentication (Argon2id)
│   ├── netshare/               # Rate limiting and networking
│   ├── service/                # Cross-platform service management
│   ├── privilege/              # UID/GID management
│   ├── template/               # Template utilities
│   ├── raw/                    # Raw paste serving
│   ├── portutil/               # Port availability checking
│   └── lineend/                # Line ending conversion
├── docker/
│   ├── Dockerfile              # Multi-stage Docker build
│   └── docker-compose*.yml     # Compose files
├── Makefile                    # Build automation
├── go.mod / go.sum             # Go module dependencies
└── README.md                   # User documentation
```

## Architecture Patterns

### Dependency Injection

All handlers receive a shared `Data` struct containing dependencies:

```go
type Data struct {
    DB  storage.DB
    Log logger.Logger
    RateLimitNew *netshare.RateLimitSystem
    // ... templates, config, etc.
}
```

### Embedded Assets

All web assets are embedded using Go's `//go:embed` directive:

```go
//go:embed data/*
var embFS embed.FS
```

This produces a single static binary with no external file dependencies.

### Database Abstraction

The `storage` package provides a unified interface for SQLite, PostgreSQL, and MySQL:

```go
db, err := storage.NewPool(driverName, dataSourceName, maxOpen, maxIdle, dataDir)
```

### Error Handling

Use custom error types with HTTP status codes where appropriate:

```go
var ErrNotFoundID = errors.New("db: could not find ID")
```

### Rate Limiting

Token bucket system with configurable per-endpoint limits (5min, 15min, 1hour windows).

## UI / JavaScript Policy

**Always prefer CSS over JavaScript.** JS may be disabled (Tor Browser, NoScript, privacy-hardened clients). Every interactive UI feature must be fully functional without JS.

Mandatory CSS-first patterns:
- Navigation toggles → hidden `<input type="checkbox">` + `<label>`, `:checked ~` / `:checked +` sibling selectors
- Dropdowns / accordions → `<details>`/`<summary>` or the checkbox hack above
- Show/hide → `:checked`, `:target`, `:focus-within`, or `@media` queries
- Animations / transitions → `@keyframes`, `transition`, CSS custom properties
- Theme switching → `<html data-theme="…">` + CSS `[data-theme]` attribute selectors

JS is permitted **only** for progressive enhancement on top of a working CSS baseline:
- Close-on-outside-click
- Escape-key dismissal
- AJAX / fetch (non-critical path; page must work without it)
- Clipboard copy buttons (degrade gracefully)

Never gate core functionality (navigation, forms, content display) behind JS.

## Code Style Guidelines

### File Headers

All source files include license header:

```go
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package packagename
```

### Import Organization

Organize imports in groups:
1. Standard library
2. Third-party packages
3. Internal packages

```go
import (
    "fmt"
    "net/http"

    "github.com/alecthomas/chroma/v2/lexers"

    "github.com/casjay-forks/caspaste/src/config"
    "github.com/casjay-forks/caspaste/src/storage"
)
```

### Naming Conventions

- **Packages:** lowercase, single-word when possible (`storage`, `config`, `web`)
- **Exported types:** PascalCase (`RateLimitSystem`, `Config`)
- **Unexported types:** camelCase or lowercase
- **Constants:** PascalCase for exported, camelCase for unexported
- **Errors:** Start with `Err` prefix (`ErrNotFoundID`)

### Error Handling

- Always handle errors explicitly
- Return errors rather than logging and continuing
- Use descriptive error messages with context

```go
if err != nil {
    return fmt.Errorf("failed to initialize database: %w", err)
}
```

### Platform-Specific Code

Use build tags or runtime checks for platform-specific behavior:

```go
switch runtime.GOOS {
case "windows":
    // Windows-specific
case "darwin":
    // macOS-specific
default:
    // Linux/BSD default
}
```

## Testing

### Running Tests

```bash
make test    # Runs: go test -v -cover ./...
```

### Test Files

Located alongside source files with `_test.go` suffix:
- `src/cli/duration_test.go`
- `src/lineend/lineend_test.go`

### Writing Tests

```go
func TestFunctionName(t *testing.T) {
    // Arrange
    input := "test input"
    expected := "expected output"

    // Act
    result := FunctionName(input)

    // Assert
    if result != expected {
        t.Errorf("FunctionName(%q) = %q, want %q", input, result, expected)
    }
}
```

## Configuration

### Environment Variables

| Variable | Description |
|----------|-------------|
| `CASPASTE_ADDRESS` | Smart address parsing (`:8080`, `host:port`) |
| `CASPASTE_CONFIG_DIR` | Config directory path |
| `CASPASTE_DATA_DIR` | Data directory path |
| `CASPASTE_DB_DIR` | Database directory |
| `CASPASTE_PUBLIC` | Public mode (`true`/`false`) |
| `PORT` | HTTP port (Docker/PaaS) |

### Config File

YAML configuration at `server.yml`:

```yaml
server:
  public: true
  listen: all
  port: ""
database:
  driver: sqlite
  source: caspaste.db
```

## Security Considerations

### Authentication

- Argon2id hashing (OWASP-recommended)
- Brute force protection: 5 failed attempts = 15-minute lockout
- Sessions: HttpOnly, SameSite cookies, auto-HTTPS detection

### Input Validation

Always validate user input:
- FQDN validation
- TLS detection
- Database driver validation
- Boolean string parsing

### XSS Prevention

- Use Go's `html/template` for HTML output (auto-escaping)
- Set appropriate Content-Type headers
- Sanitize user-provided content

## API Endpoints

### API v1

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/new` | POST | Create paste or upload file |
| `/api/v1/get/{id}` | GET | Retrieve paste |
| `/api/v1/list` | GET | List pastes |
| `/api/v1/getServerInfo` | GET | Server metadata |
| `/api/v1/healthz` | GET | Health check |

## Web UI Assets

### Templates (src/web/data/)

- `main.tmpl` - Main page
- `paste.tmpl` - Paste view
- `login.tmpl` - Authentication
- `settings.tmpl` - User preferences
- `docs*.tmpl` - Documentation pages

### JavaScript

- `main.js` - Core functionality
- `paste.js` - Paste operations
- `settings.js` - Settings management
- `sw.js` - Service Worker (PWA)

### Themes

12+ syntax highlighting themes in `src/web/data/theme/`:
- dracula, nord, gruvbox-dark, tokyo-night
- catppuccin-mocha, one-dark, github-light
- nord-light, gruvbox-light, catppuccin-latte, solarized-light

### Localization

4 languages in `src/web/data/locale/`:
- English (en)
- German (de)
- Bengali (bn_IN)
- Russian (ru)

## Common Tasks

### Adding a New API Endpoint

1. Add handler in `src/apiv1/`
2. Register route in `src/apiv1/apiv1.go`
3. Add rate limiting if needed
4. Update API documentation

### Adding a New Theme

1. Create theme file in `src/web/data/theme/`
2. Add to themes list in `src/web/themes.go`

### Adding a New Locale

1. Create locale file in `src/web/data/locale/`
2. Follow existing locale file structure
3. Add to locales list in `src/web/locale.go`

### Modifying Database Schema

1. Update model in `src/storage/`
2. Add migration logic in `src/storage/migrate.go`
3. Test with all three database backends

## Deployment

### Docker

```bash
docker run -d \
  --name caspaste \
  -p 8080:80 \
  -v ./config:/config \
  -v ./data:/data \
  ghcr.io/casjay-forks/caspaste:latest
```

### Service Installation

```bash
sudo caspaste --service install
sudo caspaste --service start
```

Supports: systemd (Linux), launchd (macOS), Windows Service, rc.d (BSD)

## Troubleshooting

### Build Issues

```bash
# Clean Go cache
rm -rf .go-cache

# Rebuild
make local
```

### Database Issues

```bash
# Check status
caspaste --status

# Backup and restore
caspaste --maintenance backup
caspaste --maintenance restore
```

### Logs

Default log locations:
- Linux (root): `/var/log/casjay-forks/caspaste/`
- Linux (user): `~/.local/log/casjay-forks/caspaste/`
- macOS: `~/Library/Logs/CasPaste/`
- Windows: `%LOCALAPPDATA%\CasPaste\Logs\`

## Contributing

1. Follow existing code style and patterns
2. Add tests for new functionality
3. Update documentation as needed
4. Run `make test` before submitting
5. Keep commits focused and descriptive

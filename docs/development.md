# Development

Guide for contributing to CasPaste.

## Prerequisites

- Go 1.23+ (runs inside Docker, not required locally)
- Docker (required for builds)
- Git

## Getting Started

```bash
# Clone the repository
git clone https://github.com/casjay-forks/caspaste.git
cd caspaste

# Build for current platform
make local

# Run tests
make test
```

## Project Structure

```
caspaste/
├── src/
│   ├── server/             # Server entry point
│   ├── client/             # CLI client entry point
│   ├── apiv1/              # REST API v1 handlers
│   ├── web/                # Web UI handlers and templates
│   │   └── data/           # Embedded assets
│   ├── storage/            # Database abstraction
│   ├── config/             # Configuration management
│   ├── cli/                # CLI argument parsing
│   ├── logger/             # Structured logging
│   ├── validation/         # Input validation
│   ├── caspasswd/          # Authentication (Argon2id)
│   ├── netshare/           # Rate limiting
│   ├── service/            # Service management
│   └── ...
├── docker/
│   ├── Dockerfile
│   └── docker-compose*.yml
├── docs/                   # This documentation
├── test/                   # Test scripts
├── Makefile
└── go.mod
```

## Build Commands

| Command | Description |
|---------|-------------|
| `make dev` | Quick build to temp dir (debugging) |
| `make local` | Build for current platform with version |
| `make build` | Build all 8 platforms |
| `make test` | Run all tests |
| `make docker` | Build Docker image |
| `make clean` | Remove build artifacts |

## Code Style

### Import Organization

```go
import (
    // Standard library
    "fmt"
    "net/http"

    // Third-party
    "github.com/alecthomas/chroma/v2/lexers"

    // Internal packages
    "github.com/casjay-forks/caspaste/src/config"
    "github.com/casjay-forks/caspaste/src/storage"
)
```

### Error Handling

```go
if err != nil {
    return fmt.Errorf("failed to initialize database: %w", err)
}
```

### Comments

Comments go ABOVE the code they describe:

```go
// calculateTotal computes the sum of all values.
// It returns 0 for empty slices.
func calculateTotal(values []int) int {
    // Implementation
}
```

## Testing

### Run Tests

```bash
make test
```

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

### Test Files

Test files are located alongside source files with `_test.go` suffix:

- `src/cli/duration_test.go`
- `src/lineend/lineend_test.go`

## Adding Features

### New API Endpoint

1. Add handler in `src/apiv1/`
2. Register route in `src/apiv1/api.go`
3. Add rate limiting if needed
4. Update API documentation

### New Theme

1. Create theme file in `src/web/data/theme/`
2. Add to themes list in `src/web/themes.go`

### New Locale

1. Create locale file in `src/web/data/locale/`
2. Follow existing locale file structure
3. Add to locales list

### Database Schema Changes

1. Update model in `src/storage/`
2. Add migration logic in `src/storage/migrate.go`
3. Test with all three database backends (SQLite, PostgreSQL, MySQL)

## Architecture

### Dependency Injection

All handlers receive a shared `Data` struct:

```go
type Data struct {
    DB           storage.DB
    Log          logger.Logger
    RateLimitNew *netshare.RateLimitSystem
    // ... templates, config, etc.
}
```

### Embedded Assets

All web assets are embedded using Go's `//go:embed`:

```go
//go:embed data/*
var embFS embed.FS
```

### Database Abstraction

The `storage` package provides a unified interface:

```go
db, err := storage.NewPool(driverName, dataSourceName, maxOpen, maxIdle, dataDir)
```

## Security Guidelines

- **Passwords:** Use Argon2id (never bcrypt for new code)
- **Tokens:** Use SHA-256 for hashing
- **Input:** Validate all user input
- **XSS:** Use `html/template` for HTML output
- **SQL:** Use prepared statements (no string concatenation)

## Release Process

1. Update version in `release.txt`
2. Run `make build` to build all platforms
3. Run `make test` to verify
4. Create GitHub release with `make release`
5. Build and push Docker images with `make docker`

## Getting Help

- **Issues:** [GitHub Issues](https://github.com/casjay-forks/caspaste/issues)
- **Discussions:** [GitHub Discussions](https://github.com/casjay-forks/caspaste/discussions)

# Development Guide

## Who

All developers and AI agents writing code in this repo.

## What

Build commands, test commands, code style conventions, and project structure
reference. For quality gates and bead workflow, see
[constitution.md](constitution.md). For system architecture, see
[architecture.md](architecture.md).

## Build Commands

```bash
# Building
go build -o nori .
make dev                # Full dev environment with Docker
make dev-server         # Server container only

# Testing — Go
go test ./...                              # All Go tests
go test -run TestName ./path/to/package    # Specific test
go test -v ./...                           # Verbose output

# Testing — Playwright (run from web/)
npx playwright test                        # All e2e tests
npx playwright test e2e/recipes/           # Specific suite
npx playwright test e2e/jobs/              # Specific suite

# Linting & Formatting
gofmt -w .
go vet ./...
go mod tidy
```

## Code Style

### Go Imports

Standard library first, third-party second, internal last. Blank lines between
groups.

```go
import (
    "context"
    "log"

    "github.com/google/uuid"
    "github.com/gofiber/fiber/v2"

    "github.com/tylerjvollick/nori/internal/models"
)
```

### Naming Conventions

- PascalCase for exported types, functions, constants
- camelCase for unexported types, functions, variables
- Descriptive variable names

### Types & Structs

- Pointers for optional fields (`*string`)
- `uuid.UUID` for IDs
- JSON + GORM struct tags on models
- Enums as custom types with constants

```go
type User struct {
    ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
    Email     string    `gorm:"not null;uniqueIndex" json:"email"`
    FirstName *string   `json:"firstName,omitempty"`
}
```

### Error Handling

- Return errors, don't panic
- Check errors immediately after operations
- Return descriptive error messages

### Code Organization

- Models in `server/internal/models/`
- Repositories in `server/internal/repositories/`
- Handlers in `server/internal/handlers/`
- DTOs in `server/internal/dtos/`
- Services in `server/internal/services/`

### API Design

- Fiber for HTTP framework
- Routes grouped logically
- Consistent JSON responses
- Middleware for authentication

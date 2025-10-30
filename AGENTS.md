# Nori Agent Guidelines

## Build/Lint/Test Commands

### Building
- `go build -o nori .` - Build the Go server binary
- `make dev` - Start full development environment with Docker
- `make dev-server` - Start only the server container

### Testing
- `go test ./...` - Run all tests
- `go test -run TestName ./path/to/package` - Run a specific test
- `go test -v ./...` - Run tests with verbose output

### Linting & Formatting
- `gofmt -w .` - Format Go code
- `go vet ./...` - Run Go vet for static analysis
- `go mod tidy` - Clean up go.mod dependencies

## Code Style Guidelines

### Imports
- Standard library imports first
- Third-party packages second
- Internal packages last
- Group with blank lines between categories

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
- **Types/Structs**: PascalCase for exported, camelCase for unexported
- **Functions**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase, descriptive names
- **Constants**: PascalCase for exported, camelCase for unexported

### Error Handling
- Return errors from functions, don't panic
- Use `log.Println()` for logging errors
- Check errors immediately after operations
- Return descriptive error messages

### Types & Structs
- Use pointers for optional fields (`*string`)
- Use `uuid.UUID` for IDs
- Include JSON and GORM struct tags
- Define enums as custom types with constants

```go
type User struct {
    ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
    Email     string    `gorm:"not null;uniqueIndex" json:"email"`
    FirstName *string   `json:"firstName,omitempty"`
}
```

### Functions
- Keep functions short and focused
- Use receiver methods for struct operations
- Return pointers for structs that may be nil
- Use dependency injection for services

### Database
- Use GORM for ORM operations
- Define models in `internal/models/`
- Use repositories for data access in `internal/repositories/`
- Handle migrations with SQL files in `migrations/`

### API Design
- Use Fiber for HTTP framework
- Group routes logically
- Return consistent JSON responses
- Use middleware for authentication
- Define DTOs in `internal/dtos/` for API contracts
# Nori Agent Guidelines

"A thin layer that holds everything together."

## What is Nori?

Nori is a self-hosted, AI-native manufacturing operations tool built for small
shops (woodworking, furniture, and general small-batch manufacturing). It
replaces the combination of Jira + Confluence + spreadsheets with a single
system that understands physical production flow.

### Core Principles

1. **Theory of Constraints (TOC)**: Work flows through physical stations, not
   abstract project phases. The system identifies the bottleneck (the "drum")
   automatically from time data. Pull-based flow with WIP limits makes
   constraints visible on the board without analysis.

2. **Documentation as a side effect**: SOPs are built incrementally during
   real work, not written in a vacuum. Time is captured automatically from
   step transitions. The goal is zero-friction knowledge capture.

3. **AI-native, not AI-bolted-on**: Local LLM (Ollama) assists with SOP
   drafting, voice-to-text, photo understanding, and bottleneck summaries.
   MCP server exposes Nori to external LLM clients (OpenCode, Claude Desktop).
   All AI features degrade gracefully if Ollama is unavailable.

4. **Self-hosted, open source**: No cloud dependencies. Runs on a homelab.
   Docker Compose for deployment. Designed for other small shops to use.

5. **Pull, not push**: Downstream stations signal readiness. Work is released
   to the floor based on actual capacity (drum-buffer-rope from *The Goal*).

### Key Workflow

```
Customer Order → Released to Floor (rope) → Station 1 → Station 2 (drum) → ... → Done
                                              ↑ WIP limits enforce pull
```

Operators execute jobs step-by-step against SOPs. Time is logged automatically.
First-time builds capture the process as it's figured out. Bottlenecks surface
from accumulated data.

## Specs

**Read `specs/readme.md` before starting any feature work.** It contains:
- A priority-ordered table of all feature specifications with search keywords
- Instructions for creating implementation checklists
- The workflow for moving specs from Planned to Implemented

Each spec file (`specs/{name}.md`) documents the who/what/where/why/how of a
feature. Before implementing a spec, create `specs/{name}-implementation.md`
with a checklist of small, committable units of work.

## Tech Stack

- **Backend**: Go, Fiber (HTTP), GORM (ORM), PostgreSQL
- **Frontend**: SvelteKit, Svelte 5, Tailwind CSS v4, shadcn-svelte
- **CLI**: cobra (Go) — same binary as the server (`nori serve` vs `nori checkin`)
- **AI**: Ollama (local inference, optional)
- **MCP**: Go MCP server for LLM client integration
- **Infrastructure**: Docker Compose (server, db, web, ollama)

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

### Go Imports
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

## Project Structure

```
nori/
├── specs/                  # Feature specifications (source of truth)
│   ├── readme.md           # START HERE — spec index with keywords
│   ├── {name}.md           # Feature spec (who/what/where/why/how)
│   └── {name}-implementation.md  # Implementation checklist (created before building)
├── server/                 # Go backend
│   ├── main.go
│   ├── internal/
│   │   ├── api/
│   │   ├── app/
│   │   ├── auth/
│   │   ├── database/
│   │   ├── dtos/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   ├── models/         # GORM models (data model spec)
│   │   ├── repositories/
│   │   ├── services/
│   │   └── utils/
│   └── migrations/
├── web/                    # SvelteKit frontend
│   └── src/
│       ├── lib/
│       └── routes/
└── docker/                 # Docker Compose config
```

## Existing Code

The repo has a working (but dated) implementation of:
- User / Account / UserAccount / Space models and auth
- SOPTemplate → SOPTemplateVersion → SOPStep → SOPStepMedia CRUD
- TUS-based chunked media upload
- Basic SvelteKit web UI (Svelte 4 / Tailwind v3 — needs upgrade)

New features being added (see specs for details):
- Station, Job, JobStep, Order, Customer, Material, BOMItem, TimeEvent models
- Flow board with pull-based WIP limits
- SOP execution with first-time capture mode
- CLI (`nori checkin`, `nori step complete`, etc.)
- MCP server for LLM integration
- Ollama-powered AI features
- Passive observation via cameras/sensors (long-term)
- SOP marketplace for community sharing (long-term)

## Context for New Sessions

If you're starting a fresh context window, here's what matters:

1. **Read `specs/readme.md`** to understand all features and find the right spec.
2. **Check `specs/{name}-implementation.md`** files for in-progress work.
3. **The owner uses OpenCode (with an MCP setup) as their daily driver** — the
   MCP server spec is designed to replace the current OpenCode + Jira + Confluence
   workflow with a native integration.
4. **The frontend is being rebuilt** in Svelte 5 + Tailwind v4. The existing
   web/ code is Svelte 4 + Tailwind v3 and will be replaced.
5. **Priority order**: data model → auth → stations → SOPs → orders → job flow
   → execution → materials → time tracking → analytics → CLI → MCP → AI →
   sensors → marketplace.

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

2. **Documentation as a side effect**: Recipes (TOML process templates) are
   built incrementally during real work, not written in a vacuum. Time is
   captured automatically from task transitions. The goal is zero-friction
   knowledge capture.

3. **AI-native, not AI-bolted-on**: Local LLM (Ollama) assists with Recipe
   drafting, voice-to-text, photo understanding, and bottleneck summaries.
   MCP server exposes Nori to external LLM clients (OpenCode, Claude Desktop).
   All AI features degrade gracefully if Ollama is unavailable.

4. **Self-hosted, open source**: No cloud dependencies. Runs on a homelab.
   Docker Compose for deployment. Designed for other small shops to use.

5. **Pull, not push**: Downstream stations signal readiness. Work is released
   to the floor based on actual capacity (drum-buffer-rope from _The Goal_).

### Key Workflow

```
Customer Order → Released to Floor (rope) → Station 1 → Station 2 (drum) → ... → Done
                                              ↑ WIP limits enforce pull
```

Operators execute tasks step-by-step against Recipes. Time is logged automatically.
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
- TUS-based chunked media upload
- Basic SvelteKit web UI (Svelte 4 / Tailwind v3 — needs upgrade)

New features being added (see specs for details):

- Task model with hierarchical string IDs and dependency graph
- Recipe + RecipeVersion models (TOML stored in database)
- Formula engine (extracted from beads project) for pouring Recipes into Tasks
- Ready-work algorithm for pull-based task execution
- Station, Order, Customer, Material, BOMItem, TimeEvent models
- Flow board with dependency-graph pull system
- Task execution with first-time capture mode
- CLI (`nori ready`, `nori task claim`, `nori recipe pour`, etc.)
- MCP server for LLM integration
- Ollama-powered AI features
- Passive observation via cameras/sensors (long-term)
- Recipe marketplace for community sharing (long-term)

## Context for New Sessions

If you're starting a fresh context window, here's what matters:

1. **Read `specs/readme.md`** to understand all features and find the right spec.
2. **Check `specs/{name}-implementation.md`** files for in-progress work.
3. **The owner uses OpenCode (with an MCP setup) as their daily driver** — the
   MCP server spec is designed to replace the current OpenCode + Jira + Confluence
   workflow with a native integration.
4. **The frontend is being rebuilt** in Svelte 5 + Tailwind v4. The existing
   web/ code is Svelte 4 + Tailwind v3 and will be replaced.
5. **Priority order**: data model → auth → stations → recipes → orders → job flow
   → execution → materials → time tracking → analytics → CLI → MCP → AI →
   sensors → marketplace.

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->

## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**

- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->

<!-- bv-agent-instructions-v2 -->

````

---

## Beads Workflow Integration

This project uses [beads](https://github.com/Dicklesworthstone/beads_rust) (`bd`) for issue tracking and [beads_viewer](https://github.com/Dicklesworthstone/beads_viewer) (`bv`) for graph-aware triage. Issues are stored in `.beads/` and tracked in git.

### Using bv as an AI sidecar

bv is a graph-aware triage engine for Beads projects (.beads/beads.jsonl). Instead of parsing JSONL or hallucinating graph traversal, use robot flags for deterministic, dependency-aware outputs with precomputed metrics (PageRank, betweenness, critical path, cycles, HITS, eigenvector, k-core).

**Scope boundary:** bv handles _what to work on_ (triage, priority, planning). `bd` handles creating, modifying, and closing beads.

**CRITICAL: Use ONLY --robot-\* flags. Bare bv launches an interactive TUI that blocks your session.**

#### The Workflow: Start With Triage

**`bv --robot-triage` is your single entry point.** It returns everything you need in one call:

- `quick_ref`: at-a-glance counts + top 3 picks
- `recommendations`: ranked actionable items with scores, reasons, unblock info
- `quick_wins`: low-effort high-impact items
- `blockers_to_clear`: items that unblock the most downstream work
- `project_health`: status/type/priority distributions, graph metrics
- `commands`: copy-paste shell commands for next steps

```bash
bv --robot-triage        # THE MEGA-COMMAND: start here
bv --robot-next          # Minimal: just the single top pick + claim command

# Token-optimized output (TOON) for lower LLM context usage:
bv --robot-triage --format toon
````

#### Other bv Commands

| Command                                             | Returns                                                                               |
| --------------------------------------------------- | ------------------------------------------------------------------------------------- |
| `--robot-plan`                                      | Parallel execution tracks with unblocks lists                                         |
| `--robot-priority`                                  | Priority misalignment detection with confidence                                       |
| `--robot-insights`                                  | Full metrics: PageRank, betweenness, HITS, eigenvector, critical path, cycles, k-core |
| `--robot-alerts`                                    | Stale issues, blocking cascades, priority mismatches                                  |
| `--robot-suggest`                                   | Hygiene: duplicates, missing deps, label suggestions, cycle breaks                    |
| `--robot-diff --diff-since <ref>`                   | Changes since ref: new/closed/modified issues                                         |
| `--robot-graph [--graph-format=json\|dot\|mermaid]` | Dependency graph export                                                               |

#### Scoping & Filtering

```bash
bv --robot-plan --label backend              # Scope to label's subgraph
bv --robot-insights --as-of HEAD~30          # Historical point-in-time
bv --recipe actionable --robot-plan          # Pre-filter: ready to work (no blockers)
bv --recipe high-impact --robot-triage       # Pre-filter: top PageRank scores
```

### bd Commands for Issue Management

```bash
bd ready              # Show issues ready to work (no blockers)
bd list --status=open # All open issues
bd show <id>          # Full issue details with dependencies
bd create --title="..." --type=task --priority=2
bd update <id> --status=in_progress
bd close <id> --reason="Completed"
bd close <id1> <id2>  # Close multiple issues at once
bd export -o .beads/beads.jsonl # Export DB to JSONL
```

### Parallel Development with bd Worktrees

Use `bd worktree` to work on multiple features/bugs in parallel within the same repo. All worktrees share a single beads database via a redirect file — no data duplication, no cross-branch contamination.

**Worktree location:** All worktrees go in `.factories/` (gitignored).

**IMPORTANT:** Always use `bd worktree create`, never raw `git worktree add`. Raw git worktrees don't set up the beads redirect, causing the database to diverge and issues to leak between branches.

```bash
# Create a worktree (branches off current HEAD)
bd worktree create .factories/bugfix-name --branch fix/MYN-XXXX-description

# Create from master instead
git worktree add .factories/bugfix-name master
echo "../.beads" > .factories/bugfix-name/.beads/redirect

# List all worktrees
bd worktree list

# Remove when done (after PR merged)
bd worktree remove .factories/bugfix-name
```

**Workflow per worktree:**

```bash
cd .factories/bugfix-name
bd create --type=bug --title="..." --priority=2    # Create beads
ralph-tui run --epic <epic-id>                     # Or work manually
git push -u origin fix/MYN-XXXX-description        # PR as normal
```

**How it works:** The worktree's `.beads/redirect` file points back to the main repo's `.beads/` directory. Both worktrees read/write the same Dolt database with file-based locking. Different epics on different branches — no interference.

**Caveats:**

- Lock contention if both worktrees write beads simultaneously (brief delay, not an error)
- Each worktree needs its own `npm ci` if dependencies differ
- Run `ralph-tui` independently in each worktree terminal

### Workflow Pattern

1. **Triage**: Run `bv --robot-triage` to find the highest-impact actionable work
2. **Claim**: Use `bd update <id> --status=in_progress`
3. **Work**: Implement the task
4. **Complete**: Use `bd close <id>`
5. **Sync**: Always run `bd export -o .beads/beads.jsonl` at session end

### Key Concepts

- **Dependencies**: Issues can block other issues. `br ready` shows only unblocked work.
- **Priority**: P0=critical, P1=high, P2=medium, P3=low, P4=backlog (use numbers 0-4, not words)
- **Types**: task, bug, feature, epic, chore, docs, question
- **Blocking**: `bd dep add <issue> <depends-on>` to add dependencies

### Session Protocol

```bash
git status              # Check what changed
git add <files>         # Stage code changes
bd export -o .beads/beads.jsonl   # Export beads changes to JSONL
git commit -m "..."     # Commit everything
git push                # Push to remote
```

<!-- end-bv-agent-instructions -->

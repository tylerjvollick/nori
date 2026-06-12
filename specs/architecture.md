# Architecture

## Who

All developers, contributors, and AI agents working on Nori. Read this first
to understand how the system fits together before diving into feature specs.

## What

A high-level overview of Nori's system architecture: the components, how they
communicate, and how the system is deployed. This is the "map" — individual
feature specs are the "territory."

## Where

```
nori/
├── server/              # Go binary (server + CLI + migrations)
│   ├── main.go          # Entry point → cobra root command
│   ├── cmd/             # CLI commands (serve, init, login, task, recipe, ...)
│   ├── internal/
│   │   ├── app/         # Fiber app wiring, route registration
│   │   ├── api/         # OpenAPI / generated types (future)
│   │   ├── auth/        # JWT + API key auth logic
│   │   ├── config/      # Env var loading + validation
│   │   ├── cli/         # CLI HTTP client, credentials, prompts
│   │   ├── database/    # GORM connection
│   │   ├── dtos/        # Request/response data transfer objects
│   │   ├── handlers/    # HTTP handlers (Fiber)
│   │   ├── middleware/   # Auth, space-scoping middleware
│   │   ├── models/      # GORM models (source of truth for schema)
│   │   ├── repositories/ # Data access layer
│   │   ├── services/    # Business logic
│   │   └── utils/       # Shared utilities
│   └── migrations/      # SQL migration files (golang-migrate)
├── web/                 # SvelteKit frontend
│   └── src/
│       ├── lib/         # Shared components, stores, API client
│       └── routes/      # Page routes
├── docker/              # Docker Compose configuration
│   ├── docker-compose.dev.yml
│   └── .env             # Instance config (gitignored)
└── seeds/               # Seed data (TOML recipes)
```

## Why

Nori is designed for **small manufacturing shops** (1-20 people) that need
production management without enterprise complexity. The architecture reflects
these priorities:

1. **Self-hosted, no cloud.** Runs on a homelab, NUC, or cheap VPS. No SaaS
   dependency, no subscriptions, no data leaving the building.

2. **Single binary.** The Go binary is both server and CLI. `nori serve`
   starts the HTTP server. `nori ready` queries it. `nori init` bootstraps
   everything. One artifact to build, deploy, and update.

3. **Docker Compose for deployment.** The target user isn't a DevOps engineer.
   `nori init` generates config and runs `docker compose up`. Updates are
   `git pull && docker compose up --build`. That's it.

4. **CLI-first, web-second.** The CLI is the primary interface for operators
   on the shop floor (fast, scriptable, works over SSH). The web UI is for
   dashboards, visual planning, and users who prefer a browser. Both talk to
   the same REST API.

5. **AI-native.** External AI agents (OpenCode, Claude Code, Cursor) interact
   with Nori through the CLI. The MCP server (future) provides a structured
   tool interface for embedded AI features. All AI is optional — the system
   works without it.

## How

### System Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        Host Machine                         │
│                                                             │
│  ┌──────────┐     ┌─────────────────────────────────────┐   │
│  │ nori CLI │────▶│         Docker Compose               │   │
│  │ (binary) │ HTTP│                                      │   │
│  └──────────┘     │  ┌────────────┐   ┌──────────────┐  │   │
│                   │  │nori-server │   │   nori-web    │  │   │
│  ┌──────────┐     │  │ (Go/Fiber) │   │  (SvelteKit) │  │   │
│  │ Web      │────▶│  │  :8080     │   │   :5173      │  │   │
│  │ Browser  │ HTTP│  └─────┬──────┘   └──────────────┘  │   │
│  └──────────┘     │        │                             │   │
│                   │        │ GORM                         │   │
│  ┌──────────┐     │  ┌─────▼──────┐                      │   │
│  │ AI Agent │────▶│  │  database  │                      │   │
│  │(OpenCode)│ CLI │  │(PostgreSQL)│                      │   │
│  └──────────┘     │  │  :5432     │                      │   │
│                   │  └────────────┘                      │   │
│                   └─────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Components

#### 1. nori CLI (Go binary, runs on host)

The `nori` binary is built locally from `server/` and lives on the host
machine's PATH. It is a **pure HTTP client** — it never touches the database
directly (except for `nori migrate up` which connects to Postgres to run
schema migrations).

**Key commands:**
- `nori init` — First-time setup wizard (spins up Docker, creates admin, space, stations)
- `nori serve` — Starts the HTTP server (runs inside Docker, not typically run by users)
- `nori migrate up` — Runs database migrations (runs inside Docker as init container)
- `nori login` — Authenticates with the server, saves credentials
- `nori ready` / `nori task *` / `nori recipe *` — Daily workflow commands

**Authentication:** Credentials stored at `~/.config/nori/credentials` as JSON.
Supports JWT tokens (30-day expiry) and API keys (no expiry, preferred for CLI).
API keys are generated during `nori init` and use the `nori_` prefix.

**Space resolution:** `--space` flag > `NORI_SPACE` env var > stored credentials.

#### 2. nori-server (Go/Fiber, Docker container)

The HTTP API server. Runs `nori serve` inside Docker on port 8080.

**Responsibilities:**
- REST API for all operations (tasks, recipes, stations, jobs, auth, admin)
- Business logic (ready-work algorithm, recipe pouring, task lifecycle)
- Authentication (JWT + API key middleware)
- Multi-tenant space scoping (all data queries filtered by space)
- Seed service (creates admin user + account on first boot)

**API groups:**
- `/auth/*` — Login, password change
- `/admin/*` — User management, API keys, space members (admin-only)
- `/api/spaces/*` — Space CRUD
- `/api/v1/*` — Core domain (tasks, recipes, stations, jobs)
- `/health` — Health check

**Configuration:** All via environment variables, loaded from `docker/.env`:
- `NORI_JWT_SECRET` — JWT signing key
- `NORI_ADMIN_EMAIL` / `NORI_ADMIN_PASSWORD` — First-boot admin credentials
- `NORI_ACCOUNT_NAME` — Account name
- `DB_HOST` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` / `DB_PORT` — PostgreSQL

#### 3. nori-web (SvelteKit, Docker container)

The web frontend. Runs the Vite dev server on port 5173 (dev) or the
SvelteKit adapter-node SSR server on port 3000 behind Caddy (production).

**Current state:** Being rebuilt from Svelte 4 + Tailwind v3 to Svelte 5 +
Tailwind v4 + shadcn-svelte. The existing pages include login, password
change, onboarding, admin panel, and flow board.

**Communication:** Talks to nori-server via the same REST API as the CLI.
The `VITE_API_URL` build-time variable points the browser at the server
(`http://localhost:8081` in dev). In production it is set to the empty
string, which means same-origin relative URLs — Caddy routes API paths to
the server. Server-side rendering uses `INTERNAL_API_URL` at runtime.

#### 4. PostgreSQL (Docker container)

The single source of truth. PostgreSQL 16, exposed on port 5432.

**Schema:** Managed by SQL migration files in `server/migrations/`, run by
`golang-migrate`. Models in `server/internal/models/` define the GORM
representation.

**Key entities:** User, Account, Space, SpaceMember, APIKey, Task, TaskDep,
Recipe, RecipeVersion, Station, TimeEvent, Customer, Material, BOMItem, Tag,
ActivityEntry, CostEntry.

**Data:** Persisted in a Docker named volume (`postgres_data`). Survives
container restarts. Destroyed only by `docker compose down -v`.

#### 5. nori-migrate (Docker init container)

Runs `./nori migrate up` and exits. The server container depends on this
with `service_completed_successfully`, ensuring migrations complete before
the server starts accepting requests. This handles both first-boot schema
creation and schema updates on subsequent deployments.

#### 6. AI Agents (external)

External AI coding agents (OpenCode, Claude Code, Cursor, etc.) interact
with Nori through the CLI. They read a skill file that documents available
commands and workflows, then shell out to `nori` commands.

**Future:** An MCP server will provide a structured tool interface for
embedded AI features (chat, voice, photo understanding). See
[mcp-server.md](mcp-server.md).

### Deployment Model

#### Development (current)

```bash
# First time
nori init              # Interactive wizard → docker compose up

# Daily use
nori ready             # What needs doing?
nori task claim <id>   # Start working
nori task complete     # Done

# Updates
git pull
docker compose -f docker/docker-compose.dev.yml up --build -d
# nori-migrate runs new migrations automatically
```

#### Production (shipped — see [deployment.md](deployment.md))

`docker/docker-compose.yml` runs pre-built images from GitHub Container
Registry behind a Caddy reverse proxy (single origin, single published
port). `.github/workflows/build-images.yml` builds and pushes
`nori-server` and `nori-web` images (amd64 + arm64, tagged `latest` +
`sha-<commit>`) on every push to `main`.

```bash
# Install: copy docker-compose.yml, Caddyfile, .env (from .env.prod.example)
docker compose pull && docker compose up -d

# Update (nori-migrate applies new migrations automatically)
docker compose pull && docker compose up -d

# Rollback: pin NORI_IMAGE_TAG=sha-<commit> in .env, pull && up
```

No source code is needed on the production machine. Future ergonomics
(per [cli.md](cli.md)): `nori init` pulls images instead of building,
`nori update` wraps the pull/up cycle, and a `curl | sh` installer
distributes the binary.

### Communication Patterns

All inter-component communication is **HTTP REST over JSON**:

```
CLI  ──HTTP──▶  Server  ──GORM──▶  PostgreSQL
Web  ──HTTP──▶  Server  ──GORM──▶  PostgreSQL
```

There is no direct CLI-to-database communication (except migrations).
There is no server-to-server communication. There are no message queues,
WebSockets (yet), or event buses.

**Authentication flow:**
1. `nori init` or `nori login` → `POST /auth/login` → JWT
2. `nori init` also generates an API key → `POST /admin/api-keys` → `nori_*` key
3. Subsequent CLI calls send `Authorization: Bearer <api-key>` + `X-Space-ID`
4. Web UI uses JWT in HTTP-only cookie + `X-Space-ID` header

### Multi-Tenancy Model

```
Account (e.g., "Vollick House")
  └── Space (e.g., "Workshop")
        ├── Stations (Milling, Assembly, Finishing)
        ├── Recipes (Dining Chair, Cutting Board)
        ├── Tasks / Jobs
        └── Materials, Customers, etc.
```

- **Account** = the organization (one per Nori installation, typically)
- **Space** = an isolated workspace within the account (like a Slack workspace)
- All domain data is scoped to a Space via `space_id` foreign keys
- Users can belong to multiple Spaces via SpaceMember
- The CLI stores a default Space ID; all commands operate within that Space

### Future Components

| Component | Purpose | Status |
|-----------|---------|--------|
| **Mobile app** | Shop floor task execution from a phone/tablet. Same REST API. | Not started |
| **MCP server** | Structured AI tool interface for embedded features | Specced, not built |
| **Ollama** | Local LLM inference for recipe suggestions, voice, photo | Specced, not built |
| **Reverse proxy** | HTTPS termination, auth for external access | Not planned yet |
| **Backup service** | Automated PostgreSQL backups | Not planned yet |

### Design Principles

1. **The server is the single source of truth.** All clients (CLI, web,
   mobile, AI) talk to the same REST API. No client has direct DB access.

2. **Offline-first is a goal, not a current feature.** Shop floors have
   spotty wifi. Eventually the CLI should queue operations locally and sync
   when connectivity returns. For now, connectivity is required.

3. **Progressive complexity.** A solo woodworker should be productive with
   just `nori init` + `nori recipe pour` + `nori task claim/complete`. Stations,
   WIP limits, analytics, and AI features are there when they're ready for them.

4. **No vendor lock-in.** PostgreSQL, not a proprietary database. Docker, not
   a proprietary container runtime. Go + Svelte, not a framework that
   disappears in 2 years. TOML recipes are human-readable text files.

## Open Questions

- **WebSocket support?** The flow board would benefit from real-time updates.
  Server-Sent Events (SSE) might be simpler. Not needed for CLI.

- **Multiple server instances?** Currently single-instance. Load balancing
  would require session affinity or stateless JWT (already stateless). Not
  a priority for small shops.

- **Database migrations in production.** The init-container approach works
  for Docker Compose. For Kubernetes (unlikely, but possible), we'd need a
  Job or init container pattern.

- **Binary distribution.** How do we distribute the `nori` binary to users
  who don't have the source? Homebrew tap? Direct download? `go install`?
  This matters for the production deployment story.

- **Mobile app technology.** Native (Swift/Kotlin) for best UX? Flutter/React
  Native for faster development? PWA wrapping the web UI? TBD based on what
  shop floor operators actually need.

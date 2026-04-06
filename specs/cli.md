# CLI

## Who

- **Operators**: Quick task management from a terminal on the shop floor.
- **Shop owners / power users**: Scripting, automation, integration.
- **AI agents**: Programmatic interaction via MCP or direct CLI calls.

## What

A command-line interface for Nori — the `nori` binary. It shares the same Go
binary as the server (`nori serve` starts the server, `nori ready` shows
available work). The CLI is a first-class interface designed for speed on the
shop floor and composability for automation.

## Where

- Backend: `cmd/` directory in the Go project, using `cobra`
- Connects to the Nori server via its REST API
- Config stored in `~/.nori/config.yaml` or environment variables

## Why

1. **Low-friction capture on the floor.** Type `nori task complete` in 2
   seconds. Faster than navigating a web UI.
2. **Composability.** Can be scripted, piped, aliased. `nori ready --json |
   jq '.[] | select(.due_date < "2026-04-15")'`. Cron jobs for reports.
3. **AI-native.** The MCP server wraps CLI commands. AI agents interact with
   Nori through the same interface humans use.

## How

### Binary Structure

```
nori
  ├── init                         — First-time setup (Docker, DB, admin user, AI skill)
  ├── serve                        — Start the Nori server
  ├── login                        — Authenticate and store token
  │
  ├── ready                        — Show ready-work queue (default view)
  │     --station <name>           — Filter by station
  │     --tag <tag>                — Filter by tag
  │     --mine                     — Only tasks assigned to me
  │
  ├── task
  │     ├── claim <id>             — Claim a task (start timer)
  │     ├── complete [id]          — Complete current/specified task
  │     ├── pause [--reason <text>] — Pause current task
  │     ├── resume [id]            — Resume a paused task
  │     ├── skip [id]              — Skip a task
  │     ├── add <title>            — Add ad-hoc task to current job
  │     │     --station <name>
  │     │     --after <id>         — Add dependency
  │     │     --type <type>        — task, gate, milestone
  │     ├── note <text>            — Add deviation note to current task
  │     ├── show <id>              — Show task detail
  │     └── list                   — List tasks (filterable)
  │
  ├── job
  │     ├── start [--capture] <title> — Create a new job (optional capture mode)
  │     ├── list                   — List jobs (filterable)
  │     ├── show <id>              — Show job with task tree
  │     ├── complete <id>          — Mark job done
  │     └── cancel <id>            — Cancel job
  │
  ├── recipe
  │     ├── list                   — List recipes
  │     ├── show <name>            — Display current version
  │     ├── pour <name>            — Pour recipe → create job
  │     │     --var key=value      — Set variables (repeatable)
  │     │     --customer <name>    — Link to customer
  │     │     --due <date>         — Set due date
  │     ├── create --from-toml <file> — Import TOML as new recipe
  │     ├── edit <name>            — Open in $EDITOR
  │     ├── diff <name> <v1> <v2>  — Diff two versions
  │     └── publish <name>         — Publish current draft
  │
  ├── dep
  │     ├── add <source> <target> [--type blocks|waits_for|related]
  │     ├── rm <source> <target>   — Remove dependency
  │     └── tree <id>              — Show dependency tree
  │
  ├── status                       — Health check: server, DB, auth, current space
  │
  ├── station
  │     ├── list                   — List stations with WIP status
  │     └── show <name>            — Show station queue
  │
  └── report
        ├── bottleneck             — Current constraint analysis
        ├── time [--user] [--job]  — Time summary
        └── overdue                — Overdue jobs
```

### Contextual State

The CLI tracks "current" state to minimize typing:
- **Current job**: Set by `nori job start` or inferred from claimed task.
- **Current task**: The task with status `active` for the logged-in user.

Station assignment lives on the task, not on the operator. In a small shop,
workers move between stations constantly — the table saw, workbench, drill
press, and back within a single task. Tracking time per-station by asking
workers to "check in" doesn't reflect reality. Instead:

- **Tasks have a station** (set by the recipe or manually)
- **Time is tracked per task** (claim starts timer, complete stops it)
- **Station time is derived** from task time grouped by station
- **Worker time is the sum of their task time**

This enables a minimal-typing workflow:

```bash
$ nori ready
 1. [shop-a4b2.3] Cut mortises — Walnut Dining Table  (pri:1, due: Apr 15)
 2. [shop-c7d1.2] Cut tenons — Cherry Side Table       (pri:2, due: Apr 20)

$ nori task claim shop-a4b2.3
Claimed shop-a4b2.3: Cut mortises. Timer started.

  ... do the work ...

$ nori task complete
shop-a4b2.3 done (23m 15s).
Next ready: shop-a4b2.4 Cut tenons — claim it? [Y/n]

$ y
Claimed shop-a4b2.4: Cut tenons. Timer started.

$ nori task note "Used 1/16\" deeper mortise — better fit"
Note added to shop-a4b2.4.

$ nori task complete
shop-a4b2.4 done (18m 42s).
No more tasks ready. 2 tasks blocked by gate-qc.
```

### Recipe Pouring from CLI

```bash
$ nori recipe pour walnut-dining-table \
    --var wood_species=Cherry \
    --var table_length=60 \
    --customer "Jane Smith" \
    --due 2026-05-01
Job shop-f8a2 created: Cherry Dining Table
  12 tasks, 2 gates, est. 6h 30m
  First ready: shop-f8a2.1 Mill lumber
```

### Output Formats

- Default: Human-readable, colored terminal output
- `--json`: JSON output for scripting/piping
- `--quiet`: Minimal output (just IDs and success/error)

### Configuration

`~/.nori/config.yaml`:
```yaml
server: https://nori.myshop.local
token: <jwt-or-api-key>
space: <default-space-id>
```

Environment variables override config:
- `NORI_SERVER`
- `NORI_TOKEN`
- `NORI_SPACE`

### Implementation

- Built with `cobra` (Go CLI framework)
- Commands call the same REST API as the web frontend
- No direct database access — CLI is a pure API client
- Single static binary, cross-compiled for Linux/macOS/Windows
- Tab completion supported for bash/zsh/fish (cobra native)

### AI Skill (External Agent Integration)

The CLI is the primary interface for external AI agents (OpenCode, Claude
Code, Open Claw, Cursor, etc.). These agents learn to use Nori through a
**skill file** — a markdown document that teaches the agent the available
commands, their arguments, and common workflows.

This is the same pattern used by beads (`AGENTS.md` integration block) and
ralph-tui (config + skill). No MCP server or special protocol needed — the
AI agent just reads the instructions and runs shell commands.

**Skill file contents:**
- Available commands and their flags
- Common workflows (pour → ready → claim → complete loop)
- Output format notes (`--json` for structured parsing)
- Authentication setup (`nori login`, config file location)
- Contextual state (current station, current task)

**Installation**: `nori init` generates the skill file automatically (see
below). It can also be appended to an existing `AGENTS.md` or placed in
`.claude/` for Claude Code / OpenCode.

**Example skill snippet:**
```markdown
## Nori CLI

You are connected to a Nori manufacturing operations server.
Use these commands to manage tasks, recipes, and jobs.

### Quick Workflow
nori ready                          # see what's available
nori task claim <id>                # claim a task (starts timer)
nori task complete                  # complete current task
nori recipe pour <slug> --var k=v   # create job from recipe

### All commands support --json for structured output.
```

The skill file is auto-generated from the CLI's cobra command tree, so it
stays in sync as commands are added or changed.

### `nori init` — First-Time Setup

`nori init` scaffolds a new Nori installation:

```bash
$ nori init
  ✓ Generated docker-compose.yml
  ✓ Generated .env with defaults (edit to customize)
  ✓ Started containers (server, db, web)
  ✓ Ran database migrations
  ✓ Created admin user (admin@localhost / <generated password>)
  ✓ Created default space ("My Shop")
  ✓ Wrote AI skill file → .claude/nori-skill.md
  ✓ Appended Nori integration → AGENTS.md

  Nori is running at http://localhost:3000
  API available at http://localhost:8080/api/v1

  Next steps:
    nori login                    # authenticate the CLI
    nori recipe create --from-toml recipes/chair.toml
    nori recipe pour chair        # create your first job
    nori ready                    # see available work
```

**What `nori init` does:**
1. Generates `docker-compose.yml` with server, postgres, web, and
   optionally ollama containers
2. Generates `.env` with sensible defaults (JWT secret, admin email,
   DB credentials) — user can customize before starting
3. Starts the Docker stack (`docker compose up -d`)
4. Runs database migrations (`nori migrate up`)
5. Creates an admin user and default space via the API
6. Generates the AI skill file for the detected agent environment
   (checks for `.claude/`, `AGENTS.md`, `.cursor/`, etc.)

**What `nori init` does NOT do:**
- No cloud account creation — this is self-hosted
- No payment or subscription — open source
- No Ollama model pulling (optional, user can run `ollama pull` separately)

## Open Questions

- Should `nori ready` be the default command (no subcommand)? i.e., just
  typing `nori` shows the ready queue? Leaning yes — it's the most common
  operation.

- Should there be a `nori watch` that live-updates the ready queue in the
  terminal? (WebSocket or polling.) Cool but possibly over-engineering for v1.

- Offline mode: queue events locally and sync when connection is restored?
  Important for shops with spotty wifi. Defer to later.

- Should `nori task complete` auto-claim the next ready task at the same
  station, or just suggest it? Leaning toward suggest with `[Y/n]` prompt.

- Should the AI skill file be auto-generated from cobra's command tree
  (always in sync, but generic) or hand-written (more natural language,
  better agent UX, but can drift)? Probably auto-generated with a
  hand-written preamble.

- Should `nori init` detect the host environment (Docker installed? GPU
  available? Which AI agent?) and adapt the generated config accordingly?

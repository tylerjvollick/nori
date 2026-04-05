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
  ├── checkin <station>            — Clock in at a station
  ├── checkout                     — Clock out
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
- **Current station**: Set by `nori checkin`. Cleared by `nori checkout`.
- **Current job**: Set by `nori job start` or inferred from claimed task.
- **Current task**: The task with status `active` for the logged-in user.

This enables a minimal-typing workflow:

```bash
$ nori checkin joinery
Checked in at Joinery.

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
No more tasks ready at Joinery. 2 tasks blocked by gate-qc.

$ nori checkout
Checked out from Joinery. Session: 42m total.
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

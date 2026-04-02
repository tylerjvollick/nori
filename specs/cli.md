# CLI

## Who

- **Operators**: Quick station check-in/out from a terminal, status checks.
- **Shop owners / power users**: Scripting, automation, integration with
  other tools.
- **Developers / contributors**: Development workflow, debugging.

## What

A command-line interface for Nori — the `nori` binary. It shares the same Go
binary as the server (`nori serve` starts the server, `nori checkin` logs a
time event). The CLI is a first-class interface for shop-floor use where
typing a short command is faster than navigating a web UI.

## Where

- Backend: `cmd/` directory in the Go project, using the `cobra` library
- Connects to the Nori server via its REST API
- Config stored in `~/.nori/config.yaml` or environment variables

## Why

The CLI exists for two reasons:

1. **Low-friction capture on the floor.** A terminal on a shop computer (or
   an SSH session from a phone) lets you type `nori checkin joinery` in 2
   seconds. That's faster than opening a browser, navigating to the board,
   and clicking buttons.

2. **Composability.** A CLI can be scripted, piped, aliased, and integrated
   with other tools. `nori status | grep overdue`. Cron jobs for automated
   reports. Integration with voice assistants ("Hey Siri, run nori checkin
   assembly").

## How

### Binary Structure

Single Go binary with subcommands:

```
nori
  ├── serve              — Start the Nori server
  ├── login              — Authenticate and store token
  ├── status             — Show current board state / active jobs
  ├── checkin <station>  — Clock in at a station
  ├── checkout           — Clock out from current station
  ├── job
  │     ├── list         — List jobs (filterable)
  │     ├── start <id>   — Start working on a job
  │     ├── complete     — Complete current job
  │     └── info <id>    — Show job details
  ├── step
  │     ├── complete     — Complete current step, advance to next
  │     ├── pause        — Pause current step
  │     ├── resume       — Resume paused step
  │     ├── skip         — Skip current step
  │     └── note <text>  — Add a deviation note to current step
  ├── sop
  │     ├── list         — List SOPs
  │     ├── show <id>    — Display an SOP's steps
  │     └── update <step> --note <text>  — Suggest an SOP update
  ├── station
  │     ├── list         — List stations with WIP
  │     └── info <name>  — Show station queue
  └── report
        ├── bottleneck   — Show current constraint analysis
        ├── time         — Time summary for current user
        └── overdue      — List overdue orders
```

### Authentication

```bash
$ nori login
Server URL: https://nori.myshop.local
Email: tyler@vollickhouse.com
Password: ********
Logged in. Token saved to ~/.nori/config.yaml

$ nori login --server https://nori.myshop.local --token <api-key>
```

Token is stored locally and sent as a Bearer header on all API requests.

### Contextual State

The CLI tracks "current" state to minimize typing:
- **Current station**: Set by `nori checkin`. Cleared by `nori checkout`.
- **Current job**: Set by `nori job start`. Cleared by `nori job complete`.
- **Current step**: Automatically advanced by `nori step complete`.

This means a typical workflow is:

```bash
$ nori checkin joinery
Checked in at Joinery.

$ nori job start 42
Started Job #42: Walnut Dining Table
Step 1: Cut mortises

  ... do the work ...

$ nori step complete
Step 1 completed (23m 15s). Step 2: Cut tenons started.

$ nori step note "Used 1/16 deeper mortise, better fit"
Note added to Step 2.

$ nori step complete
Step 2 completed (18m 42s). Step 3: Dry fit started.

  ... continue ...

$ nori job complete
Job #42 completed. Total time: 3h 12m.

$ nori checkout
Checked out from Joinery.
```

### Output Formats

- Default: Human-readable, colored terminal output
- `--json` flag: JSON output for scripting/piping
- `--quiet` flag: Minimal output (just success/error)

### Configuration

`~/.nori/config.yaml`:
```yaml
server: https://nori.myshop.local
token: <jwt-or-api-key>
space: <default-space-id>
```

Environment variables override config file:
- `NORI_SERVER`
- `NORI_TOKEN`
- `NORI_SPACE`

### Implementation

- Built with `cobra` (Go CLI framework)
- Commands call the same REST API as the web frontend
- No direct database access — CLI is a pure API client
- Can be compiled for Linux/macOS/Windows (cross-compilation)
- Distributed as a single static binary (easy to install on shop computers)

### API Surface

The CLI doesn't have its own API — it consumes the same API endpoints defined
in the other specs. The CLI is a client, not a server.

## Open Questions

- Should the CLI support tab completion for station names, job IDs, etc.?
  (cobra supports this natively for bash/zsh/fish.)
- Should there be a `nori watch` command that shows a live-updating board
  in the terminal? (Cool but possibly over-engineering for v1.)
- How should the CLI handle offline mode? (Shop might have spotty wifi.)
  Could queue events locally and sync when connection is restored.
- Should there be a `nori init` command for initial server setup (create
  admin account, first space, default stations)?

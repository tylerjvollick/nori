# MCP Server

## Who

- **Nori's embedded AI**: The internal LLM (Ollama or cloud provider) uses
  MCP tools to query and act on Nori data — powering chat, voice, photo
  understanding, and recipe assistance features within the Nori app itself.
- **Future external integrations**: Remote AI clients that can't use the CLI
  directly (e.g., a mobile app, a hosted AI assistant).

## What

A Model Context Protocol (MCP) server that exposes Nori's data and actions
as structured tools for AI consumption. This is the **internal tool protocol**
that connects Nori's embedded AI features (see ai-features.md) to Nori's
service layer.

**Important distinction — two integration layers:**

1. **CLI skill (external AI agents)**: For AI tools with shell access
   (OpenCode, Claude Code, Open Claw, Cursor), the CLI *is* the AI
   interface. A markdown skill file teaches the agent the commands. This is
   the primary integration path for development, automation, and power users.
   See cli.md for the skill specification.

2. **MCP server (embedded AI)**: For AI running *inside* Nori's own
   interface — chat panels, voice input, photo understanding — where there's
   no shell to run commands. MCP provides structured tool definitions so the
   embedded LLM can interact with Nori's internals directly.

The CLI skill is v1. The MCP server is needed when we build the embedded AI
features (ai-features.md).

## Where

- Backend: MCP server endpoint in the Go binary (separate port or path from
  the REST API)
- Protocol: MCP specification (JSON-RPC over stdio or HTTP+SSE)
- Primary consumer: Nori's own AI service layer (Ollama sidecar)
- Secondary consumer: Remote MCP-compatible clients (future)

## Why

The CLI covers external AI agents well, but it requires shell access. The
MCP server exists for cases where the AI doesn't have a shell:

- **Embedded chat in the Nori web UI**: A shop worker asks "what should I
  work on next?" — the embedded LLM calls MCP tools to query the ready queue
  and dependency graph, then responds in plain language.
- **Voice input on the shop floor**: "Hey Nori, I'm done with the mortises"
  — speech-to-text produces text, the embedded LLM interprets it and calls
  `nori_task_complete` via MCP.
- **Photo understanding**: Operator snaps a photo of a defect — the vision
  model describes it and the embedded LLM uses MCP to create a deviation note.
- **Recipe drafting**: "Make a recipe like the dining table but simpler" —
  the LLM reads existing recipes via MCP resources and generates a new TOML.

MCP is the standard protocol for this. It means:
- **No custom integration per LLM** — Ollama, OpenAI, Anthropic all work
  through the same tool definitions
- **Nori stays the system of record** — the LLM reads/writes Nori data
- **Users bring their own LLM** — local Ollama or cloud provider (BYOK)

## How

### MCP Tools (Actions)

Tools are functions the LLM can call to perform actions:

```
nori_checkin
  - station: string (required)
  - task_id: string (optional)
  - notes: string (optional)
  → Checks the user in at a station, optionally for a specific task

nori_checkout
  → Checks the user out from their current station

nori_task_claim
  - task_id: string (required)
  → Claims a task and starts the timer

nori_task_complete
  - task_id: string (optional — defaults to current active task)
  - notes: string (optional)
  → Completes the current/specified task

nori_task_pause / nori_task_resume
  → Pause or resume the current task timer

nori_add_deviation_note
  - task_id: string (optional — defaults to current task)
  - note: string (required)
  → Adds a deviation note to a task

nori_recipe_pour
  - recipe_name: string (required)
  - vars: object (optional — variable overrides)
  - customer: string (optional)
  - due_date: string (optional)
  → Pours a recipe to create a new job

nori_task_add
  - title: string (required)
  - parent_id: string (optional — defaults to current job)
  - station: string (optional)
  - after: string (optional — task ID to add dependency on)
  → Adds an ad-hoc task to a job

nori_log_time
  - station: string (required)
  - duration_minutes: int (required)
  - task_id: string (optional)
  - notes: string (optional)
  → Manually log time (for backdated entries)

nori_adjust_stock
  - material_id: string (required)
  - quantity: decimal (required — positive for adding, negative for consuming)
  - notes: string (optional)
  → Adjust material stock level
```

### MCP Resources (Read-Only Data)

Resources are data the LLM can read for context:

```
nori://board
  → Current flow state: ready-work queue, active jobs, station WIP

nori://ready
  → Ready-work queue for the current user's space

nori://jobs/active
  → List of active jobs with status, progress, current tasks

nori://jobs/{id}
  → Full job detail including task tree and dependencies

nori://recipes
  → List of all recipes with current version info

nori://recipes/{id}
  → Full recipe with TOML content

nori://stations
  → List of stations with current WIP and capacity

nori://bottleneck
  → Current constraint analysis summary

nori://time/today
  → Today's time log for the current user

nori://orders/active
  → Active orders with status and due dates

nori://materials/low-stock
  → Materials below reorder threshold
```

### Authentication

The MCP server authenticates via the same mechanism as the REST API:
- API key in the MCP client configuration
- The API key is scoped to a user + space
- All actions are performed as that user (for time tracking attribution)

### Example Conversations

**Hands-dirty scenario (voice via OpenCode):**
> "Hey, check me in at joinery and claim the next ready task."
> → LLM reads `nori://ready`, calls `nori_checkin(station="joinery")`,
>    then `nori_task_claim(task_id="shop-a4b2.3")`
> → "You're checked in at Joinery. Claimed task shop-a4b2.3: Cut mortises
>    for the Walnut Dining Table. Estimated time: 30 minutes."

**End of day review:**
> "What did I work on today?"
> → LLM reads `nori://time/today`
> → "Today you spent 2h 15m at Joinery (tasks on the Dining Table and Side
>    Table jobs), 1h at Assembly, and 45m at Finish. Total: 4h."

**Recipe update from the bench:**
> "For the dining table recipe, the tenon should be 1/16 deeper than
>  specified for a tighter fit."
> → LLM calls `nori_add_deviation_note(note="...")`

**Bottleneck check:**
> "What's been my biggest bottleneck this month?"
> → LLM reads `nori://bottleneck`
> → "Joinery has been the constraint for 65% of this month. Average wait
>    time: 2.1 days. 4 tasks are currently blocked by joinery dependencies."

### Implementation

- Built in Go alongside the REST API server
- MCP transport: **stdio** for local embedded use, **HTTP+SSE** for remote
  clients
- Tools and resources are registered at server startup
- Each tool validates inputs, calls the same service layer as the REST API
  and CLI, and returns structured results
- MCP server can be enabled/disabled via config flag
- The MCP server is a consumer of the same backend services as the REST API
  and CLI — no duplication of business logic

### Relationship to CLI and REST API

Three interfaces, one service layer:

```
┌─────────────────────────────────────────────────────────┐
│                     Nori Service Layer                   │
│  (tasks, recipes, ready-work, time tracking, analytics) │
└────────┬──────────────────┬──────────────────┬──────────┘
         │                  │                  │
    REST API            CLI (cobra)        MCP Server
         │                  │                  │
    Web frontend     External AI agents   Embedded AI
    (SvelteKit)      (OpenCode, Claude    (chat panel,
                      Code, Open Claw)    voice, photo,
                                          recipe assist)
```

- **REST API**: Powers the web frontend. Standard HTTP endpoints.
- **CLI**: Powers external AI agents via skill files AND human operators
  via terminal. The CLI is the primary AI integration for v1.
- **MCP**: Powers embedded AI features inside Nori's own UI. Needed when
  we build the chat/voice/photo features from ai-features.md.

All three call the same service layer. The CLI and MCP server are both
thin wrappers — the CLI wraps HTTP calls to the REST API, the MCP server
calls the service layer directly (in-process).

## Open Questions

- Should the MCP server support streaming responses? (e.g., live board
  updates as jobs move.) The MCP spec supports this but adds complexity.
- How should the MCP server handle ambiguous requests? (e.g., "Check me in"
  without specifying a station.) Should it return an error, or provide
  context for the LLM to ask a follow-up?
- Should there be MCP tools for administrative actions (create stations,
  manage users) or only operational ones? (Leaning toward operational-only
  for safety.)
- What's the right granularity for resources? Too many small resources =
  LLM makes many calls. Too few large resources = LLM gets too much context.

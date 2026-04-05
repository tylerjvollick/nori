# MCP Server

## Who

- **Shop owners / operators using LLM tools**: Interact with Nori through
  natural language via OpenCode, Claude Desktop, or any MCP-compatible client.
- **Developers**: Build custom AI integrations on top of Nori's data.

## What

A Model Context Protocol (MCP) server that exposes Nori's data and actions
as tools and resources that LLM clients can use. This enables natural language
interaction with the production system: "What's the bottleneck this week?",
"Log 30 minutes at joinery for task abc.3", "What's the next ready task for
the dining table job?"

## Where

- Backend: MCP server endpoint in the Go binary (separate port or path from
  the REST API)
- Protocol: MCP specification (JSON-RPC over stdio or SSE)
- Connects to: OpenCode, Claude Desktop, or any MCP-compatible client

## Why

This is what makes the current OpenCode + Jira + Confluence workflow native
to Nori. Instead of the LLM bridging three disconnected systems, it talks
directly to one system that understands manufacturing.

The MCP approach means:
- **No custom integration per LLM client** — any tool that speaks MCP works
- **The LLM is the UI for hands-dirty situations** — voice command to OpenCode
  while your hands are covered in wood glue
- **Nori stays the system of record** — the LLM reads/writes Nori data, it
  doesn't replace it
- **Users choose their LLM** — OpenCode today, something else tomorrow

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
- MCP transport: **stdio** for local use (OpenCode), **SSE** for remote
  clients
- Tools and resources are registered at server startup
- Each tool validates inputs, calls the same service layer as the REST API,
  and returns structured results
- MCP server can be enabled/disabled via config flag

### Relationship to CLI

The MCP server and CLI serve similar functions (interact with Nori
programmatically) but for different audiences:
- **CLI**: Explicit commands, scriptable, for users who know what they want
- **MCP**: Natural language, AI-mediated, for hands-dirty or exploratory use

Both call the same backend service layer. No duplication of business logic.

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

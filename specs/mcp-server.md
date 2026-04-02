# MCP Server

## Who

- **Shop owners / operators using LLM tools**: Interact with Nori through
  natural language via OpenCode, Claude Desktop, or any MCP-compatible client.
- **Developers**: Build custom AI integrations on top of Nori's data.

## What

A Model Context Protocol (MCP) server that exposes Nori's data and actions
as tools and resources that LLM clients can use. This enables natural language
interaction with the production system: "What's the bottleneck this week?",
"Log 30 minutes at joinery for job 42", "What's the next step in the SOP for
the dining table?"

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
  - job_id: string (optional)
  - notes: string (optional)
  → Checks the user in at a station, optionally for a specific job

nori_checkout
  → Checks the user out from their current station

nori_step_complete
  - notes: string (optional)
  → Completes the current step and advances to the next

nori_step_pause / nori_step_resume
  → Pause or resume the current step timer

nori_add_deviation_note
  - job_step_id: string (optional — defaults to current step)
  - note: string (required)
  → Adds a deviation note to a job step

nori_create_job
  - sop_template_id: string (required)
  - order_line_item_id: string (optional)
  - priority: int (optional)
  - notes: string (optional)
  → Creates a new job

nori_move_job
  - job_id: string (required)
  - station: string (required)
  → Moves a job to the next station (with WIP check)

nori_update_sop_step
  - sop_step_id: string (required)
  - instructions: string (optional)
  - notes: string (optional)
  → Suggests an update to an SOP step (creates a draft change)

nori_log_time
  - station: string (required)
  - duration_minutes: int (required)
  - job_id: string (optional)
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
  → Current flow board state: all stations with jobs and WIP counts

nori://jobs/active
  → List of active jobs with status, station, current step

nori://jobs/{id}
  → Full job detail including all steps and their status

nori://sops
  → List of all SOPs with current version info

nori://sops/{id}
  → Full SOP with steps, materials, equipment

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
> "Hey, check me in at joinery and start job 42."
> → LLM calls `nori_checkin(station="joinery", job_id="42")`
> → "You're checked in at Joinery, working on Job #42: Walnut Dining Table.
>    Step 1: Cut mortises. Estimated time: 30 minutes."

**End of day review:**
> "What did I work on today?"
> → LLM reads `nori://time/today`
> → "Today you spent 2h 15m at Joinery (Jobs #42, #43), 1h at Assembly
>    (Job #38), and 45m at Finish (Job #37). Total: 4h."

**SOP update from the bench:**
> "For step 4 of the dining table SOP, add a note that the tenon should be
>  1/16 deeper than specified for a tighter fit."
> → LLM calls `nori_add_deviation_note` or `nori_update_sop_step`

**Bottleneck check:**
> "What's been my biggest bottleneck this month?"
> → LLM reads `nori://bottleneck`
> → "Joinery has been the constraint for 65% of this month. Average queue
>    time: 2.1 days. 4 jobs are currently waiting."

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

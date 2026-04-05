# Job Flow

## Who

- **Shop owners / managers**: Monitor the flow board, identify bottlenecks,
  manage WIP, approve gates.
- **Operators**: See their ready work, pull tasks, track progress.

## What

The production flow system — how work moves through the shop. Instead of the
old station-column Kanban (push jobs left-to-right through stations), Nori uses
a **dependency-graph pull system** inspired by beads' ready-work algorithm and
the Theory of Constraints.

Work is organized as Jobs (root Tasks) containing child Tasks with
dependencies. The ready-work algorithm finds what's unblocked and available.
Operators pull from the ready queue. WIP limits at stations enforce capacity.

## Where

- Backend: Task service, ready-work service, job flow API endpoints
- Frontend: Flow board page (primary view for operators and managers)
- Data model: see data-model.md

## Why

A station-column Kanban (Jira-style) has problems for small-shop manufacturing:

1. **Jobs don't move linearly** — A dining table goes: table saw → jointer →
   table saw → mortiser → assembly → finish. It bounces between stations based
   on the recipe, not a fixed left-to-right flow.
2. **Push creates WIP pileups** — You keep starting work regardless of
   downstream capacity.
3. **Dependencies are invisible** — "Can't finish the top until the base is
   done" isn't representable in a column-based board.

A dependency-graph pull system solves all three:
- Tasks declare what they depend on. The graph handles any flow pattern.
- Ready-work only surfaces unblocked tasks. Operators pull, never push.
- WIP limits at stations make the bottleneck visible.

From *The Goal* (Goldratt): the drum (bottleneck) sets the pace, the buffer
protects it from starvation, and the rope ties new work release to actual
throughput.

## How

### Flow Model

There is no "board with columns." Instead, there are three views into the
same task graph:

#### 1. Ready Queue (operator's primary view)

"What can I work on right now?"

```
Ready Work — Joinery Station
─────────────────────────────
1. [shop-a4b2.3] Cut mortises — Walnut Dining Table     (pri: 1, due: Apr 15)
2. [shop-c7d1.2] Cut tenons — Cherry Side Table          (pri: 2, due: Apr 20)
3. [shop-e9f3.1] Mill blanks — Cutting Board Set         (pri: 3, due: Apr 22)
```

Filtered by station (optional), sorted by priority → due date → creation date.
Only shows tasks where all `blocks` dependencies are resolved.

#### 2. Job View (manager's tracking view)

"How is this job progressing?"

```
Job: shop-a4b2 — Walnut Dining Table (Order #042)
──────────────────────────────────────────────────
[done] .1 Mill lumber (45m)
[done] .2 Joint and plane (30m)
[active] .3 Cut mortises ← Tyler, 23m elapsed
[open] .4 Cut tenons (blocked by .3)
[open] .5 Dry fit (blocked by .3, .4)
[open] .gate-qc QC: Joinery inspection (gate, blocked by .5)
[open] .6 Glue up (blocked by .gate-qc)
[open] .7 Apply finish (blocked by .6)

Progress: 2/8 done, 1 active | Est. remaining: 3h 45m
```

Shows the full task tree with dependency status, who's working on what, and
time data.

#### 3. Station View (capacity view)

"What's happening at each station?"

```
Stations
────────────────────────────────────────────────────────
Table Saw    [2/3]  ██░  shop-a4b2.3 (Tyler), shop-c7d1.2 (—)
Jointer      [0/2]  ░░   (empty)
Assembly     [1/2]  █░   shop-f2a1.6 (Mike)
Finish Room  [0/1]  ░    (empty)
────────────────────────────────────────────────────────
Buffer: Table Saw has 1 task waiting (shop-e9f3.1)
```

Shows WIP at each station against limits. This is where the drum (bottleneck)
becomes visually obvious — it's the station that's always at capacity.

### Pull Mechanics

**There is no "move job to next station" action.** Instead:

1. Operator completes a task.
2. Ready-work algorithm re-evaluates. Tasks that depended on the completed
   task may become unblocked.
3. Newly-ready tasks appear in the ready queue.
4. The next operator (or the same one) claims a task from the ready queue.

This is pure pull. Work only moves forward when someone actively claims it
and their station has capacity.

### WIP Enforcement

When an operator claims a task at a station:

1. Count active tasks at that station.
2. If count < WIPLimit: claim succeeds.
3. If count >= WIPLimit (soft mode): warn "Station is at capacity. Claim
   anyway?" Confirmation required.
4. If count >= WIPLimit (hard mode): claim rejected. "Station at capacity.
   Complete existing work first."

Default is soft limits — small shops need flexibility.

### Drum-Buffer-Rope

Nori doesn't require explicit drum identification. Instead:

1. **Drum emerges from data** — The bottleneck-analytics system (see
   bottleneck-analytics.md) identifies which station is most frequently at
   capacity over time.
2. **Buffer is visible** — The station view shows queued (ready but unclaimed)
   tasks per station. If the drum's buffer is empty, that's a risk.
3. **Rope is the order release** — New jobs are only created (poured from
   recipes) when floor capacity permits. The manager sees: "Current floor
   WIP: 12/15. 3 jobs queued for release."

Advanced: managers can mark a station as the drum, which enables auto-pacing
of job release to match drum throughput and buffer monitoring alerts.

### Job Lifecycle

```
[Created] → [Active] → [Done]
                ↓
           [Cancelled]
```

A Job (root Task) is:
- `open` — Created, tasks not yet started. Waiting in release queue.
- `active` — At least one child task is active.
- `done` — All child tasks are done/skipped.
- `cancelled` — Abandoned.

Job status is computed from child task statuses, not set manually.

### Dependency Patterns

**Sequential** (most common — recipe step ordering):
```
mill → joinery → assembly → finish
```

**Parallel** (independent sub-assemblies):
```
top-assembly ──┐
               ├── final-assembly
base-assembly ─┘
```

**Gate** (QC hold):
```
joinery → qc-gate → assembly
```

**Cross-job** (batch coordination):
```
chair-1.assembly ──┐
chair-2.assembly ──┤
chair-3.assembly ──┼── batch-finish.prep
chair-4.assembly ──┤
chair-5.assembly ──┤
chair-6.assembly ──┘
```

**Fan-out / fan-in** (concurrent work):
```
design → implement-A ──┐
       → implement-B ──┤── integration-test
       → implement-C ──┘
```

All of these are expressible with TaskDep edges. The ready-work algorithm
handles them uniformly.

### Priority and Ordering

Within the ready queue, tasks are ordered by:
1. Priority (lower number = higher, default 0)
2. Due date of the parent Job (earliest first)
3. Display order within the job
4. Creation date (FIFO)

Managers can override priority on individual tasks or jobs.

### Tags

Tasks inherit tags from their parent Job. Tags enable filtering:
- `order` — Customer work
- `prep` — Preparation tasks
- `maintenance` — Shop maintenance
- `3s` — Sweep/sort/standardize tasks
- Custom tags per space

Operators can filter the ready queue by tag: "Show me only maintenance tasks."

### Flow Board UI

The flow board is a tabbed/split view combining the three perspectives:

**Tab: Ready** (default for operators)
- Ready work queue, optionally filtered by station
- Claim button on each task
- Shows task title, job name, station, priority, due date

**Tab: Jobs** (default for managers)
- List of active jobs with progress bars
- Expandable to show task tree
- Filter by status, customer, due date, tag
- Color-coded: green (on track), yellow (approaching due), red (overdue)

**Tab: Stations** (capacity overview)
- Station cards with WIP gauges
- Active tasks at each station with operator names
- Buffer counts (ready tasks waiting)
- Visual alarm when station hits capacity

### API Surface

```
GET    /api/spaces/:spaceId/ready                       — Ready-work queue
GET    /api/spaces/:spaceId/ready?station=:stationId    — Station-filtered
GET    /api/spaces/:spaceId/ready?tag=:tag              — Tag-filtered

GET    /api/spaces/:spaceId/jobs                        — List jobs
POST   /api/spaces/:spaceId/jobs                        — Create job (manual)
GET    /api/jobs/:id                                    — Job detail with task tree
PUT    /api/jobs/:id                                    — Update job metadata
POST   /api/jobs/:id/cancel                             — Cancel job

GET    /api/spaces/:spaceId/stations/status              — Station WIP overview
GET    /api/stations/:id/queue                           — Tasks at/queued for a station

GET    /api/spaces/:spaceId/flow                         — Full flow state (jobs + stations + ready)
```

## Open Questions

- Should there be a global WIP limit (max total active tasks on the floor),
  separate from per-station limits? This is the "release rope" concept.

- Should the flow board support a dependency graph visualization (nodes and
  edges, like beads-viewer's DependencyGraph component)? Useful for complex
  jobs but potentially overwhelming. Maybe as an optional "graph view" tab.

- How should the board handle stale work? A task that's been `active` for
  3x the recipe estimate should be flagged.

- Should there be an "expedite" lane for rush orders that bypasses normal
  priority? Or just use priority = 0?

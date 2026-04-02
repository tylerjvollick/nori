# Job Flow

## Who

- **Shop owners / managers**: Monitor the flow board, identify bottlenecks,
  manage WIP.
- **Operators**: See their station's queue, pull work, move jobs forward.

## What

The production flow board — the central UI and workflow engine of Nori. Jobs
move through Stations in a pull-based system inspired by the Theory of
Constraints (drum-buffer-rope). This is what replaces Jira's Kanban board
with something native to physical manufacturing.

## Where

- Backend: Job model, job flow API endpoints, WIP enforcement logic
- Frontend: Flow board page (the primary view operators see)
- Data model: see data-model.md

## Why

A push system (Jira) creates WIP pileups — you keep starting work regardless
of whether the next station can absorb it. This leads to long lead times, lost
jobs, and invisible bottlenecks.

A pull system (Nori) means downstream stations signal when they're ready.
Work only moves forward when there's capacity. The bottleneck becomes visible
*on the board itself* — it's the column that's always full.

From *The Goal* (Goldratt):
- **Drum**: The bottleneck station sets the pace for the entire shop.
- **Buffer**: A small queue of work before the bottleneck protects it from
  starvation (so it's never idle waiting for work).
- **Rope**: A signal that ties the release of new work to the drum's actual
  throughput (so you don't over-release).

## How

### Job Model

```
Job
  - ID: uuid
  - SpaceID: uuid
  - OrderLineItemID: uuid (nullable — internal/maintenance jobs have no order)
  - SOPTemplateVersionID: int (snapshot at job creation)
  - CurrentStationID: uuid (nullable — null = in order queue, not yet released)
  - Status: enum (queued, in_progress, blocked, completed, cancelled)
  - Priority: int (lower number = higher priority)
  - AssignedToID: uuid (nullable)
  - StartedAt, CompletedAt: timestamps
  - Notes: text
```

### Flow Board Layout

```
[ORDER QUEUE] → [MILL: 2/3] → [JOINERY: 3/3 !] → [ASSEMBLY: 1/3] → [FINISH: 0/3] → [DONE]
```

- One column per active Station, ordered by DisplayOrder.
- First pseudo-column: **Order Queue** — confirmed jobs not yet released to
  the floor. This is the "rope" holding area.
- Last pseudo-column: **Done** — completed jobs.
- Each column header shows: station name, current WIP / WIP limit.
- Visual alarm when a station is at capacity (color change, icon).

### Pull Mechanics

**Moving a job to the next station:**
1. Operator completes work at their station.
2. They click "Complete" on the job (or the current job step).
3. The system checks if the next station has capacity (WIP < WIPLimit).
4. **If yes**: Job moves to the next station's queue.
5. **If no**: Job stays at the current station with a "waiting" indicator.
   The next station operator sees a "ready to pull" signal.

**Releasing work from the order queue:**
- Jobs in the Order Queue are only released to the first station when the
  first station has capacity.
- The system can auto-release (FIFO by priority, then by due date) or
  require manual release by a manager.
- This is the "rope" — it prevents overloading the shop floor.

### WIP Enforcement

- **Soft limit** (default): When a station reaches its WIP limit, the board
  shows a visual warning. Work can still be moved there manually (with
  confirmation: "Station is at capacity. Move anyway?").
- **Hard limit** (configurable): Work cannot be moved to a station that's at
  capacity. Period. The pull signal is the only way.

Soft limits are recommended for v1 — small shops need flexibility, and hard
limits can cause frustration when learning the system.

### Drum-Buffer-Rope in Practice

Nori doesn't require the user to explicitly identify the drum. Instead:
1. Over time, the bottleneck-analytics system (see bottleneck-analytics.md)
   identifies which station is most frequently at capacity.
2. The board *shows* the drum naturally — it's the column that's always full.
3. The buffer is the queue in front of the drum — BufferSize on the Station
   model controls how deep this queue can get.
4. The rope is the Order Queue release mechanism — it gates new work based on
   overall floor capacity.

Advanced users can explicitly mark a station as the drum, which enables:
- Auto-pacing of the Order Queue release to match drum throughput
- Buffer monitoring alerts ("drum buffer is empty — risk of starvation")

### Job Cards on the Board

Each job card on the board shows:
- Job title / product name
- Customer name (if from an order)
- Due date (color-coded: green = on track, yellow = approaching, red = overdue)
- Assigned operator (if any)
- Current step name from the SOP
- Time at current station

Click to expand: full SOP step list, progress bar, deviation notes.

### Priority and Ordering

Within a station's queue, jobs are ordered by:
1. Priority (explicit, set by manager)
2. Due date (earliest first)
3. Creation date (FIFO)

Managers can drag to reorder within a station. Operators work top-down.

### Job Types

- **Order Job**: Created from an OrderLineItem. Has a customer, due date,
  and flows through all stations defined in the SOP.
- **Internal Job**: No order — maintenance tasks, prep work, shop improvements.
  Created manually, tagged for filtering.
- **Replenishment Job**: Auto-created by material pull signals (see
  materials-and-bom.md). Tagged as "prep" or "restock."

### Tags

Jobs can have tags for filtering the board:
- `order` — customer work
- `prep` — preparation tasks
- `maintenance` — shop maintenance
- `3s` — sweep/sort/standardize tasks
- Custom tags per Space

Operators can filter the board by tag to focus on their current mode (e.g.,
morning = 3S tags, then prep, then orders once the shop opens).

### API Surface

```
GET    /api/spaces/:spaceId/board                  — Full board state (all stations + jobs)
GET    /api/spaces/:spaceId/jobs                   — List jobs (filterable by status, station, tag)
POST   /api/spaces/:spaceId/jobs                   — Create a job (manual / internal)
GET    /api/jobs/:id                               — Get job detail with steps
PUT    /api/jobs/:id                               — Update job metadata
POST   /api/jobs/:id/move                          — Move job to next station (with WIP check)
POST   /api/jobs/:id/assign                        — Assign operator
POST   /api/jobs/:id/complete                      — Complete job
POST   /api/spaces/:spaceId/board/release          — Release jobs from order queue
```

## Open Questions

- Should the board support multiple "lanes" per station? (e.g., two operators
  at the same station, each with their own WIP.) Probably not for v1.
- Should there be an explicit "blocked" state with a reason field? (e.g.,
  "waiting for material" or "waiting for glue to cure.") This is common in
  manufacturing and would be useful.
- How should the board handle jobs that skip stations? (e.g., a cutting board
  doesn't need a joinery step.) Should the SOP define which stations apply,
  or should operators manually skip?
- Should the Order Queue have its own capacity limit (max jobs on the floor)?
  This is the global WIP limit concept from Lean/TOC.

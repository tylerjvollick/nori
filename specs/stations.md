# Stations

## Who

- **Shop owners**: Configure stations to match their shop layout.
- **Floor managers**: Monitor WIP at each station, identify constraints.
- **Operators**: See their station's queue, pull work when ready.

## What

Stations represent the physical locations, tools, or work areas in a shop
where work happens. They are physical nodes in the production flow — tasks
reference the station where they occur. Each station has a configurable WIP
limit that enforces pull-based flow and makes bottlenecks visible.

## Where

- Backend: `server/internal/models/station.go`, station API endpoints
- Frontend: Station configuration page, station view on flow board
- Data model: see data-model.md

## Why

This is the core concept that makes Nori a manufacturing tool rather than a
generic project management tool. In Jira, work moves through abstract columns
("To Do", "In Progress", "Done"). In Nori, work moves through physical
stations that map to real places in the shop.

From Theory of Constraints:
- The **drum** is the station with the lowest capacity — it sets the pace for
  the entire shop.
- The **buffer** is a small queue of work in front of the drum, protecting it
  from starvation.
- The **rope** ties the release of new work to the drum's actual throughput.

WIP limits at each station make the constraint visible without any analysis.
When a station is consistently at its limit, that's the bottleneck. You can
see it on the board in real time.

## How

### Station Model

```
Station
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string ("Rough Mill", "Joinery", "Assembly", "Finish", "QC")
  - Description: string (nullable — "Table saw, jointer, planer area")
  - DisplayOrder: int (left-to-right position on the flow board)
  - WIPLimit: int (max tasks actively being worked at this station)
  - BufferSize: int (max tasks queued waiting for this station)
  - IsActive: bool (soft delete / disable without losing data)
  - CostsHour: decimal (nullable — hourly cost rate; when null, costing
    falls back to the space's DefaultLaborRate)
  - Color: string (nullable — hex color for board visualization)
  - CreatedAt, UpdatedAt: timestamp
```

### Default Stations

When a new Space is created, seed it with a sensible default set that the
owner can customize:

1. **Intake** — Order received, materials staged
2. **Mill** — Rough dimensioning, jointing, planing
3. **Joinery** — Mortise & tenon, dovetails, complex fitting
4. **Assembly** — Glue-up, clamping, sub-assembly
5. **Finish** — Sanding, staining, spraying, curing
6. **QC** — Quality check, final inspection
7. **Ship** — Packaging, delivery

Default WIP limit: 3 per station. Default buffer: 2. These are starting
points — the owner adjusts based on their actual capacity.

### Station Configuration UI

- Drag-and-drop reordering (updates DisplayOrder)
- Inline editing of Name, WIPLimit, BufferSize
- Add/remove stations
- Color picker for board visualization
- Toggle IsActive to hide a station without deleting it

### Flow Board Integration

The flow board's station view (see job-flow.md) shows one card per active
station, ordered by DisplayOrder. Each card shows:
- Station name
- Current WIP count vs. limit (e.g., "2/3")
- Buffer queue count vs. size (e.g., "1/2 queued")
- Active tasks at the station with operator names
- Visual indicator when at capacity (color change, icon)

### API Surface

```
GET    /api/spaces/:spaceId/stations          — List all stations (ordered)
POST   /api/spaces/:spaceId/stations          — Create a station
PUT    /api/spaces/:spaceId/stations/:id       — Update a station
DELETE /api/spaces/:spaceId/stations/:id       — Soft-delete (set IsActive=false)
PUT    /api/spaces/:spaceId/stations/reorder   — Bulk update DisplayOrder
```

Station update accepts `costsHour`. The field is tri-state: absent leaves the
rate unchanged, an explicit `null` clears it (cost calculations then fall back
to the space's `defaultLaborRate`, set via the space update endpoint), and a
non-negative decimal sets it. GET responses always include `costsHour`.

## Open Questions

- Should stations support sub-stations? (e.g., "Finish" has "Sanding" and
  "Spraying" as children.) Probably not for v1 — keep it flat.
- Should WIP limits count only "active" tasks, or also "paused" tasks at
  that station? (Leaning toward counting all non-completed tasks at the
  station, since a paused task still physically occupies the space.)
- Do we need a concept of "station capacity" beyond WIP limit? (e.g., a
  station with 2 workbenches can handle 2 parallel tasks, but WIP limit
  might be 3 to include one in the buffer.) The WIPLimit + BufferSize split
  may already cover this.

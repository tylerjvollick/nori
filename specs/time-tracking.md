# Time Tracking

## Who

- **Operators**: Clock in/out at stations, have time logged automatically
  during job execution.
- **Managers / owners**: Review time data, identify slow steps, analyze
  station utilization.
- **The system**: Aggregates time data for bottleneck analytics.

## What

A source-agnostic time event store. Every time-relevant action in Nori
(starting a task, checking in at a station, pausing work) creates a TimeEvent.
Events can come from multiple sources: the web UI, the CLI, a tablet tap-in,
or future camera/sensor systems. The data model is the same regardless of
source.

## Where

- Backend: TimeEvent model, time API endpoints
- Frontend: Time log viewer, daily/weekly summaries
- CLI: `nori checkin`, `nori checkout`
- Data model: see data-model.md

## Why

Time data is the raw material for bottleneck identification. Without it, you're
guessing where your constraint is. With it, Nori can tell you: "Joinery
consumed 42% of total production time last month. Assembly was 18%."

The current workflow (telling OpenCode what you're doing so it can log time
to Jira) is friction-heavy and unreliable. Nori's approach: time is captured
as a **side effect of normal workflow** (starting/completing steps) with
fallback options (CLI, tap-in) for situations where step-level tracking isn't
practical.

The key design decision: **time events are append-only and source-tagged**.
This means:
- You never lose data (no overwrites)
- You can analyze which input methods are actually used
- Future sources (cameras) plug in without changing the schema

## How

### TimeEvent Model

```
TimeEvent
  - ID: uuid
  - SpaceID: uuid
  - UserID: uuid (who)
  - TaskID: string (nullable — what task)
  - StationID: uuid (nullable — where)
  - EventType: enum (check_in, check_out, pause, resume)
  - Source: enum (manual, web, cli, tap, sensor, api, system)
  - Timestamp: timestamp (when — can be backdated for corrections)
  - Notes: text (nullable)
  - CreatedAt: timestamp (when the record was actually created)
```

### Event Sources

| Source | How it works | Friction level |
|--------|-------------|----------------|
| `system` | Auto-generated when a task starts/completes in the web UI | Zero — happens automatically |
| `web` | Manual time entry in the web UI (corrections, forgot to log) | Low |
| `cli` | `nori checkin joinery` from a terminal | Low |
| `tap` | Tap a button on a tablet/phone mounted at a station | Very low |
| `sensor` | Camera/presence detection auto-logs (see passive-observation.md) | Zero |
| `api` | External system posts a time event via API | Zero |
| `manual` | Backfill entry with explicit timestamp | Medium |

### Automatic Time Capture (via Task Execution)

When the execution system (see task-execution.md) transitions tasks, it
automatically creates TimeEvents:

```
Task claimed    → TimeEvent(type=check_in, task=abc.3, station=Joinery, source=system)
Task paused     → TimeEvent(type=pause, task=abc.3, source=system)
Task resumed    → TimeEvent(type=resume, task=abc.3, source=system)
Task completed  → TimeEvent(type=check_out, task=abc.3, source=system)
Next claimed    → TimeEvent(type=check_in, task=abc.4, station=Joinery, source=system)
```

This means: if an operator is using the task execution UI, time tracking is
100% automatic. No extra action needed.

### Station Check-In (without a specific job)

Not all time in the shop is tied to a specific job. An operator might be
at the joinery station doing general work, setup, or cleanup. The CLI and
tap interfaces support station-only check-ins:

```
nori checkin joinery          → TimeEvent(type=check_in, station=Joinery, source=cli)
nori checkout                 → TimeEvent(type=check_out, station=Joinery, source=cli)
```

These events contribute to station utilization metrics even without job-level
granularity.

### Time Corrections

Since the store is append-only, corrections work by adding new events:
- "I forgot to check out at 5pm yesterday" → Create a `check_out` event with
  Timestamp=yesterday 5pm, Source=manual, Notes="backdated correction"
- The analytics engine uses Timestamp (not CreatedAt) for calculations

### Daily/Weekly Summaries

The frontend provides:
- **Daily log**: Timeline of events for a user, showing when they were at
  at which station working on which task
- **Weekly summary**: Total hours, broken down by station and job/task
- **Anomaly detection**: "You were checked in at Mill for 6 hours without
  any task completions — is that accurate?"

### Computed Fields

While TimeEvents are the raw data, computed aggregates are cached for
performance:
- `Task.ActualTimeSeconds` — sum of active time from events for that task
- Job total time — sum across all child tasks
- Station utilization per day/week — sum of check-in to check-out durations

These are recomputed from events on demand or on a schedule, not stored as
the source of truth.

### API Surface

```
GET    /api/spaces/:spaceId/time-events            — List events (filterable by date, user, station, task)
POST   /api/spaces/:spaceId/time-events            — Create manual event
GET    /api/spaces/:spaceId/time-summary            — Aggregated summary (daily/weekly)
GET    /api/users/:userId/time-log                  — User's time log
GET    /api/stations/:stationId/utilization          — Station utilization over time range
```

## Open Questions

- Should TimeEvents be truly immutable (append-only, corrections are new
  events) or should we allow editing? (Append-only is better for trust and
  analytics, but harder for simple corrections like "I fat-fingered the
  check-in time.")
- Do we need a concept of "shift" or "work day"? (e.g., auto-checkout at
  end of day if someone forgets.) This prevents runaway timers.
- Should time data be visible to all Space members, or should operators only
  see their own time? (Transparency is better for a healthy shop culture, but
  some owners might want privacy.)
- How long do we retain raw TimeEvents? Forever? Or aggregate and archive
  after N months?

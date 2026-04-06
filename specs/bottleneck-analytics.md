# Bottleneck Analytics

## Who

- **Shop owners**: Understand where to invest (tools, training, outsourcing)
  to increase throughput.
- **Floor managers**: Monitor flow health daily, catch emerging constraints.
- **The system**: Surfaces insights that humans would miss in day-to-day work.

## What

Analytics built on top of time tracking and job flow data that identify the
shop's constraint (bottleneck) using Theory of Constraints principles. The
system answers the fundamental question from *The Goal*: "What is preventing
this shop from producing more throughput?"

## Where

- Backend: Analytics computation engine (queries against TimeEvent, Task,
  Station tables)
- Frontend: Analytics dashboard, constraint reports
- AI: Plain-language summaries via Ollama (see ai-features.md)

## Why

> "One of the biggest problems for me currently is identifying bottlenecks
> to begin with."

This is the payoff of all the data collection. Time tracking and job flow
are the inputs. Bottleneck analytics is the output — the thing that tells you
*where to focus*.

The Theory of Constraints five focusing steps:
1. **Identify** the constraint → Nori does this automatically from data
2. **Exploit** the constraint → Nori suggests: don't waste the bottleneck's
   time on non-critical work
3. **Subordinate** everything else → Nori enforces this via pull/WIP limits
4. **Elevate** the constraint → Nori tracks whether your investments are
   working
5. **Repeat** → When the bottleneck shifts, Nori detects the new one

## How

### Key Metrics

**Station-Level:**
- **Utilization**: % of available hours a station is occupied (sum of task
  time at station + non-job time events / total available hours)
- **WIP Depth**: Average number of active tasks at this station over time
- **Queue Time**: Average time a task spends waiting in a station's buffer
  before being worked on
- **Processing Time**: Average time a task spends being actively worked at
  this station
- **Throughput**: Tasks completed per day/week at this station

**Job-Level:**
- **Lead Time**: Total time from job creation to completion
- **Touch Time**: Sum of all active processing time across tasks
- **Wait Time**: Lead Time - Touch Time (time spent waiting on dependencies)
- **Flow Efficiency**: Touch Time / Lead Time (higher = less waiting)

**Recipe-Level:**
- **Task Time Variance**: Actual time vs. recipe estimated time per task,
  averaged across all executions
- **Deviation Frequency**: How often operators deviate from the recipe per task
- **Execution Count**: How many times this recipe has been poured (experience curve)

**Order-Level:**
- **On-Time Delivery Rate**: % of orders completed by their due date
- **Lead Time Accuracy**: Quoted lead time vs. actual lead time
- **Average Order Value**: Revenue per order (if pricing is tracked)

### Bottleneck Identification Algorithm

The constraint is the station where work accumulates. Nori identifies it by:

1. **WIP accumulation**: Which station most frequently has WIP at or near its
   limit? Measured as (avg WIP / WIP limit) over a time period.
2. **Queue depth**: Which station has the longest average queue time? Jobs
   waiting in the buffer = work piling up before the constraint.
3. **Utilization**: Which station has the highest utilization? A station at
   95% utilization with queued work is the drum.
4. **Downstream starvation**: If a station frequently has zero WIP while the
   station before it has a full buffer, the upstream station is the constraint.

The algorithm weights these factors and produces a **constraint score** per
station. The station with the highest score is flagged as the current
bottleneck.

### Dashboard Views

**Real-Time Board Health** (overlay on the flow board):
- Color-coded stations: green (flowing), yellow (approaching limit), red
  (at limit / bottleneck)
- Sparkline charts showing WIP trend per station over the last 7 days

**Weekly Constraint Report**:
- "This week's bottleneck: **Joinery** (constraint score: 0.87)"
- "Joinery was at WIP limit 62% of operating hours"
- "Average queue time before Joinery: 1.8 days"
- "Suggestion: Can any joinery work be outsourced or batched differently?"

**Trend Analysis**:
- Station constraint scores over weeks/months
- Did the bottleneck shift after you bought a new tool? Hired help?
- Lead time trend: are orders getting delivered faster or slower?

**SOP Performance**:
- Tasks that consistently take longer than recipe estimates → candidates for
  recipe update or process improvement
- Tasks with high deviation frequency → process not standardized yet

### TOC Five Focusing Steps — System Support

| Step | What Nori Does |
|------|---------------|
| Identify | Constraint score highlights the bottleneck station |
| Exploit | Alert: "Joinery (bottleneck) has been idle for 30 min — are there jobs ready?" |
| Subordinate | WIP limits on non-bottleneck stations prevent overproduction |
| Elevate | Before/after comparison when you change capacity (new tool, new hire, process change) |
| Repeat | Detect when the constraint shifts to a different station after improvement |

### Data Requirements

This spec depends on:
- time-tracking.md — raw TimeEvent data
- job-flow.md — task/job status and station assignment data
- stations.md — WIP limits and buffer sizes
- task-execution.md — task-level timing data

Analytics are only meaningful after sufficient data accumulates. The system
should indicate confidence: "Based on 3 jobs (low confidence)" vs. "Based on
47 jobs (high confidence)."

### Dependency Graph Analysis

The station-level metrics above answer "which station is the constraint?" from
accumulated time data. The dependency graph answers a complementary question:
"which tasks in *this specific job* are the chokepoint right now?"

This is computed from the task dependency graph (TaskDep edges), not from
historical time data. It's useful immediately — even on the first pour of a
recipe — because it only needs the graph structure.

**Critical Path**: The longest chain of dependent tasks through a job's task
graph, measured by estimated time (from recipe) or actual time (during
execution). Tasks on the critical path have zero slack — any delay directly
delays the job.

- Computed per job via topological sort + longest-path on the DAG
- Highlighted on the flow board and in `nori job show`
- Updates in real-time as tasks complete (remaining critical path)

**Slack**: For each task *not* on the critical path, how much can it slip
before it delays the job? Calculated as (latest allowable start - earliest
possible start). High slack = can be deferred. Zero slack = critical path.

- Useful for prioritizing when an operator has multiple ready tasks
- `nori ready` could sort by slack (ascending) so critical-path tasks surface
  first, or flag them

**Bottleneck Detection (graph-based)**: Betweenness centrality on the task
dependency graph identifies tasks that sit on the most dependency paths. A task
with high betweenness is a chokepoint — if it's delayed, many downstream tasks
are blocked. This complements the station-level bottleneck analysis: stations
are the *physical* constraint, graph bottlenecks are the *logical* constraint.

- Computed per job or across active jobs in a space
- Surfaced in `nori report bottleneck` alongside station metrics
- Especially useful for complex jobs with many parallel branches that
  reconverge (e.g., all sub-assemblies must complete before final assembly)

**Cycle Detection**: Circular dependencies (A blocks B blocks A) make the
ready-work algorithm deadlock. The TaskDep repository rejects cycles on insert,
but the system should also:

- Surface clear error messages when a cycle is rejected ("Cannot add
  dependency: would create cycle A → B → C → A")
- Provide a `nori dep check` command or health-check endpoint that validates
  the entire graph is acyclic
- Show cycle warnings in the flow board UI if one is somehow introduced
  (defensive — shouldn't happen, but worth guarding)

### API Surface

```
GET    /api/spaces/:spaceId/analytics/bottleneck     — Current constraint analysis
GET    /api/spaces/:spaceId/analytics/stations        — Station-level metrics
GET    /api/spaces/:spaceId/analytics/jobs            — Job-level metrics (lead time, etc.)
GET    /api/spaces/:spaceId/analytics/jobs/:jobId/critical-path — Critical path + slack for a job
GET    /api/spaces/:spaceId/analytics/recipes/:recipeId — Recipe performance metrics
GET    /api/spaces/:spaceId/analytics/orders          — Order-level metrics (on-time rate)
GET    /api/spaces/:spaceId/analytics/trends          — Time-series data for charting
GET    /api/spaces/:spaceId/analytics/graph-health    — Cycle detection + graph metrics
```

## Open Questions

- How much data is needed before bottleneck identification is reliable?
  (Probably 10+ jobs through the system. Should show a "gathering data"
  state before then.)
- Should the analytics engine run on a schedule (nightly batch) or compute
  on demand? (On-demand for v1, batch for larger shops.)
- Should Nori recommend specific actions? ("Buy a second mortiser" vs. just
  "Joinery is the bottleneck.") Recommendations are valuable but risk being
  wrong. Maybe AI-generated suggestions with a disclaimer.
- Do we need to account for "planned downtime" (lunch, end of day) vs.
  "unplanned downtime" (machine broke) in utilization calculations?
- Should critical path calculation use estimated time from recipes, actual
  elapsed time from in-progress tasks, or a blend? (Estimated for planning,
  actual for live monitoring, blend for mid-execution replanning.)
- Is betweenness centrality worth computing for typical shop jobs (10-20
  tasks)? Probably only meaningful for complex jobs with 20+ tasks and
  multiple reconvergence points. Could gate it behind a threshold.

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

- Backend: Analytics computation engine (queries against TimeEvent, Job,
  JobStep, Station tables)
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
- **Utilization**: % of available hours a station is occupied (check-in to
  check-out time / total available hours)
- **WIP Depth**: Average number of jobs at this station over time
- **Queue Time**: Average time a job spends waiting in a station's buffer
  before being worked on
- **Processing Time**: Average time a job spends being actively worked at
  this station
- **Throughput**: Jobs completed per day/week at this station

**Job-Level:**
- **Lead Time**: Total time from job creation to completion
- **Touch Time**: Sum of all active processing time across stations
- **Wait Time**: Lead Time - Touch Time (time spent in queues)
- **Flow Efficiency**: Touch Time / Lead Time (higher = less waiting)

**SOP-Level:**
- **Step Time Variance**: Actual time vs. estimated time per step, averaged
  across all executions
- **Deviation Frequency**: How often operators deviate from the SOP per step
- **Execution Count**: How many times this SOP has been run (experience curve)

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
- Steps that consistently take longer than estimated → candidates for SOP
  update or process improvement
- Steps with high deviation frequency → process not standardized yet

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
- job-flow.md — job status and station assignment data
- stations.md — WIP limits and buffer sizes
- sop-execution.md — step-level timing data

Analytics are only meaningful after sufficient data accumulates. The system
should indicate confidence: "Based on 3 jobs (low confidence)" vs. "Based on
47 jobs (high confidence)."

### API Surface

```
GET    /api/spaces/:spaceId/analytics/bottleneck     — Current constraint analysis
GET    /api/spaces/:spaceId/analytics/stations        — Station-level metrics
GET    /api/spaces/:spaceId/analytics/jobs            — Job-level metrics (lead time, etc.)
GET    /api/spaces/:spaceId/analytics/sops/:sopId     — SOP performance metrics
GET    /api/spaces/:spaceId/analytics/orders          — Order-level metrics (on-time rate)
GET    /api/spaces/:spaceId/analytics/trends          — Time-series data for charting
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

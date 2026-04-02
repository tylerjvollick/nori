# SOP Execution

## Who

- **Operators**: The primary users — executing jobs step by step on the floor.
- **Managers**: Reviewing execution data, identifying training gaps.
- **The system**: Capturing time, deviations, and first-time process knowledge.

## What

The live execution of a Job against an SOP. When an operator works a job, they
step through the SOP procedure with actual time tracking, the ability to add
notes and photos, and — critically — a "first-time capture mode" that helps
build the SOP as you go when there's no established process yet.

This is the feature that replaces the current OpenCode + Jira workflow. Instead
of telling an AI what you're doing after the fact, the system guides you through
the process and captures data as a natural byproduct.

## Where

- Backend: JobStep model, execution API endpoints
- Frontend: Job execution view (the primary operator screen during work)
- Data model: see data-model.md

## Why

> "So much mental energy goes into the complex joinery... after a project is
> done it's like 'what did I even do?'"

This is the core problem Nori solves. The execution system captures *what
happened* without requiring the operator to stop and document it. Time is
logged automatically by step transitions. Deviations are captured with
minimal friction. Photos taken during work are attached to the step they
belong to.

### First-Time Capture Mode

The most important workflow: building something for the first time. There's no
SOP yet (or only a rough one). The operator is figuring out the process as they
go. Today, this is done via OpenCode — you tell the AI what you did, and it
updates a Confluence doc. Nori should make this native:

- The system knows this is the first execution of an SOP (no prior JobStep
  completion data).
- It enters "capture mode" — prompting the operator to add notes, photos,
  and timing as they work.
- After the job is complete, the captured data is structured into an SOP
  draft that can be reviewed and published.

## How

### JobStep Model

```
JobStep
  - ID: uuid
  - JobID: uuid (FK → Job)
  - SOPStepID: int (FK → SOPStep — the template step)
  - StationID: uuid (FK → Station)
  - Status: enum (pending, in_progress, paused, completed, skipped)
  - AssignedToID: uuid (nullable)
  - StartedAt: timestamp (nullable)
  - PausedAt: timestamp (nullable)
  - CompletedAt: timestamp (nullable)
  - ActualTimeSeconds: int (accumulated active time)
  - DeviationNotes: text (nullable)
  - CreatedAt, UpdatedAt: timestamp
```

### Execution UI — Normal Mode

When an operator opens their active job, they see:

```
┌─────────────────────────────────────────┐
│ Job: Walnut Dining Table (Order #042)   │
│ Station: Joinery                         │
│                                          │
│ Step 3 of 12: Cut mortises (IN PROGRESS) │
│ ┌──────────────────────────────────────┐ │
│ │ Timer: 00:23:15                      │ │
│ │                                      │ │
│ │ Instructions:                        │ │
│ │ Use 3/8" mortise chisel. Set fence   │ │
│ │ to 1/4" from reference face...       │ │
│ │                                      │ │
│ │ [Photo gallery from SOP]             │ │
│ │                                      │ │
│ │ [Pause] [Complete Step] [Add Note]   │ │
│ └──────────────────────────────────────┘ │
│                                          │
│ Next: Step 4 - Cut tenons               │
│ Remaining: 9 steps (~2h 15m estimated)   │
└─────────────────────────────────────────┘
```

**Single-tap progression**: From the requirements doc — one tap to complete
the current step, which automatically starts the next step's timer. A middle
button to pause (e.g., lunch break, interruption).

**Expandable detail**: Experienced operators see just the step title and
timer. Tap to expand for full instructions and photos.

### Execution UI — First-Time Capture Mode

When the SOP has no prior execution history (or is flagged as "draft"):

```
┌─────────────────────────────────────────┐
│ Job: New Product - First Build           │
│ CAPTURE MODE                             │
│                                          │
│ Step 3: [Untitled - tap to name]        │
│ ┌──────────────────────────────────────┐ │
│ │ Timer: 00:23:15                      │ │
│ │                                      │ │
│ │ What are you doing?                  │ │
│ │ [Voice note] [Photo] [Type note]    │ │
│ │                                      │ │
│ │ [Pause] [Complete Step] [Add Step]   │ │
│ └──────────────────────────────────────┘ │
│                                          │
│ Steps so far: 3                          │
│ [Review & Finish Job]                    │
└─────────────────────────────────────────┘
```

Key differences from normal mode:
- Steps can be added on the fly ("Add Step" button)
- Step titles are editable inline
- Prompts encourage capture: "What are you doing?", "Take a photo of this"
- Voice-to-text for hands-dirty situations (phone on the bench, talk to it)
- After completing the job, a "Review" screen shows all captured steps and
  lets the operator clean up before it becomes an SOP draft

### Deviation Capture

In normal mode (existing SOP), the operator can add a deviation note to any
step:
- "Used 1/16 more tenon depth than specified — better fit"
- "Skipped step 4 — not needed for this wood species"
- "Added 5 extra minutes of curing time"

Deviations are stored on the JobStep. After the job, the system (or AI —
see ai-features.md) can prompt: "You noted deviations on 3 steps. Would you
like to update the SOP?"

### Time Tracking Integration

Every step transition generates TimeEvents (see time-tracking.md):
- Step start → `check_in` event for that step/station
- Step pause → `pause` event
- Step resume → `resume` event
- Step complete → `check_out` event

ActualTimeSeconds on JobStep is the accumulated active time (excludes pauses).
This is computed from the TimeEvents but cached on the JobStep for quick access.

### Pause Handling

When a step is paused:
- Timer stops
- PausedAt is recorded
- When resumed, the gap is excluded from ActualTimeSeconds
- Long pauses (> configurable threshold, e.g., 30 min) trigger a prompt:
  "Still working on this? Or did you switch tasks?"

### Step Skipping

Operators can skip steps that don't apply to their specific job. Skipped steps
are marked with status `skipped` and don't count toward time totals or SOP
averages.

### Post-Job Summary

After completing all steps (or marking the job as done):
- Summary screen showing: total time, time per step, deviations noted
- Comparison to SOP estimates (if available)
- Prompt to update the SOP if deviations were captured
- In first-time capture mode: prompt to review and publish the captured SOP

### API Surface

```
GET    /api/jobs/:id/steps                         — Get all job steps with status
POST   /api/job-steps/:id/start                    — Start a step (begin timer)
POST   /api/job-steps/:id/pause                    — Pause a step
POST   /api/job-steps/:id/resume                   — Resume a step
POST   /api/job-steps/:id/complete                 — Complete a step (stop timer, advance)
POST   /api/job-steps/:id/skip                     — Skip a step
PUT    /api/job-steps/:id/notes                    — Add/update deviation notes
POST   /api/job-steps/:id/media                    — Attach photo/video to a step execution

POST   /api/jobs/:id/capture-step                  — Add a new step during first-time capture
POST   /api/jobs/:id/finalize-capture              — Convert captured steps to SOP draft
```

## Open Questions

- Should first-time capture mode be explicit (operator chooses it) or
  automatic (system detects no prior executions)?
- How should voice-to-text work? On-device speech recognition (browser API)
  or send to Ollama for transcription? (Browser API is simpler and works
  offline, but quality varies.)
- Should the execution UI be a dedicated full-screen view, or embedded in
  the flow board? (Leaning toward dedicated — operators need focus, not
  board-level context.)
- How do we handle multi-person jobs? (e.g., two people doing a glue-up.)
  Can a step be assigned to multiple operators?

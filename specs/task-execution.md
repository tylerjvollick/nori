# Task Execution

Replaces: `sop-execution.md`

## Who

- **Operators**: Execute tasks step by step on the shop floor.
- **Managers**: Review execution data, identify training gaps, approve gates.
- **The system**: Capture time, deviations, and first-time process knowledge
  automatically.

## What

The live execution of tasks within a Job. Operators work through tasks guided
by recipe instructions (if the job was poured from a recipe) or freely (ad-hoc
jobs). Time is tracked automatically via task transitions. The ready-work
algorithm determines what's available to claim next.

The key innovation: **first-time capture mode** builds recipes as a natural
byproduct of doing work, and the **diff/promote flow** improves recipes from
real execution data.

## Where

- Backend: Task service, ready-work service, execution API endpoints
- Frontend: Task execution view (primary operator screen during work)
- Data model: see data-model.md

## Why

> "So much mental energy goes into the complex joinery... after a project is
> done it's like 'what did I even do?'"

The execution system captures *what happened* without requiring the operator
to stop and document. Time is logged by task transitions. Deviations are
captured with minimal friction. Photos taken during work are attached to the
task they belong to.

## How

### Ready-Work Algorithm

The foundation of task execution. Ported from beads' `ready_work.go`:

1. **Find open tasks** — All tasks with status `open` in the space.
2. **Compute blocked set** — For each open task, check if any `blocks`
   dependencies have unresolved sources. If a task's parent is blocked, the
   task is also blocked.
3. **Exclude blocked** — Ready tasks = open ∩ NOT blocked.
4. **Sort** — By priority (lower first), then display order, then creation date.
5. **Apply station filter** (optional) — Show only tasks for a specific station.
6. **Apply user filter** (optional) — Show only tasks assigned to a user (or
   unassigned).

An operator asks "what should I work on?" and gets a prioritized list of
tasks that are actually ready — no guessing about dependencies.

### Claiming a Task

```
Operator: nori ready              → See ready tasks
Operator: nori task claim abc.3   → Claim task abc.3
System:   Task abc.3 → status: active, AssignedToID: operator, StartedAt: now
System:   TimeEvent created (check_in, source: cli)
System:   ActivityEntry created (task_started)
```

Claiming:
- Sets status to `active`
- Assigns the operator (if not already assigned)
- Records `StartedAt`
- Creates a TimeEvent (check_in)
- Starts the timer

Only one task can be `active` per operator at a time (enforced by the service
layer). Attempting to claim a second task prompts: "You're working on abc.2.
Complete or pause it first?"

### Completing a Task

```
Operator: nori task complete      → Complete current task
System:   Task abc.3 → status: done, CompletedAt: now, ActualTimeSeconds: computed
System:   TimeEvent created (check_out)
System:   ActivityEntry created (task_completed)
System:   Suggest next: "abc.4 is now ready. Claim it?"
```

Completing:
- Sets status to `done`
- Records `CompletedAt`
- Computes `ActualTimeSeconds` from TimeEvents (excludes pauses)
- Creates a TimeEvent (check_out)
- Runs ready-work to find newly unblocked tasks
- Suggests the next task (same station preferred, then same job, then global)

### Pausing

```
Operator: nori task pause --reason "lunch"
System:   Task abc.3 → status: paused, PausedAt: now
System:   TimeEvent created (pause)
```

Long pauses (> configurable threshold, e.g., 30 min) trigger a prompt:
"Still working on this? Or did you switch tasks?"

### Task Progression (Single-Tap Flow)

For jobs poured from a recipe with linear dependencies (step 1 → 2 → 3),
completing a task automatically starts the next one:

```
Complete Step 3 → Step 3 done → Step 4 now ready → Auto-claim Step 4 → Timer starts
```

This is the "single-tap progression" from the original SOP execution spec.
One tap to advance. The operator stays in flow.

Auto-advance only happens when:
- The next task is the only newly-ready task
- The next task is at the same station
- The operator is the assignee (or it's unassigned)

Otherwise, the system suggests but doesn't auto-claim.

### Gate Resolution

Gates are tasks with `type = gate` that block downstream work:

**Human gate** (QC check, manager approval):
```
Manager: nori task complete abc.gate-qc   → Approve the gate
System:  Gate resolved → downstream tasks unblocked → appear in ready-work
```

**Timer gate** (cure time, drying time):
```
System: Gate abc.gate-cure created with timeout 4h
System: (4 hours later) Auto-resolves gate → downstream unblocked
```

Gates show prominently on the flow board with a distinct visual treatment.
Pending gates are surfaced in the manager's ready-work view.

### First-Time Capture Mode

When a Job has no source recipe (`RecipeID = null`), the system enters
capture mode. The operator builds the process as they go:

```
┌─────────────────────────────────────────┐
│ Job: New Product - First Build           │
│ CAPTURE MODE                             │
│                                          │
│ Current: [Untitled - tap to name]        │
│ ┌──────────────────────────────────────┐ │
│ │ Timer: 00:23:15                      │ │
│ │                                      │ │
│ │ What are you doing?                  │ │
│ │ [Voice note] [Photo] [Type note]    │ │
│ │                                      │ │
│ │ [Pause] [Complete] [+ Add Task]     │ │
│ └──────────────────────────────────────┘ │
│                                          │
│ Tasks so far: 3                          │
│ [Review & Finish Job]                    │
└─────────────────────────────────────────┘
```

Key behaviors:
- Tasks can be added on the fly (`+ Add Task` button or `nori task add`)
- Task titles are editable inline
- Prompts encourage capture: "What are you doing?", "Take a photo"
- Voice-to-text for hands-dirty situations
- Station can be set per task
- After completing the job, a review screen shows all captured tasks
- **Promote to recipe**: captured tasks → TOML recipe draft (see recipes.md)

CLI equivalent:
```bash
$ nori job start --capture "New cutting board design"
Job shop-a4b2 created in capture mode.

$ nori task add "Select and mark lumber"
Task shop-a4b2.1 created and claimed.

$ nori task complete
Task shop-a4b2.1 done (12m 30s).

$ nori task add "Mill to rough dimensions" --station mill
Task shop-a4b2.2 created and claimed.
```

### Deviation Capture

When executing a recipe-poured job, the operator can note deviations:

```bash
$ nori task note "Used 1/16\" deeper mortise — better fit for this grain"
Note added to shop-a4b2.3
```

Deviations are stored in `Task.DeviationNotes`. After the job completes, the
diff/promote flow (see recipes.md) surfaces these deviations and offers to
fold them back into the recipe.

### Ad-Hoc Tasks During Execution

Even in recipe-poured jobs, operators can add tasks that weren't in the recipe:

```bash
$ nori task add "Touch up edge with hand plane" --after shop-a4b2.5
Task shop-a4b2.8 created with dependency on shop-a4b2.5
```

Ad-hoc tasks have `RecipeVersionID = null`. The diff/promote flow identifies
these as additions.

### Interruptions

```bash
$ nori task pause --reason "Dust collector full"
Task shop-a4b2.3 paused.

$ nori task add --type job "Empty dust collector" --tag maintenance
Job shop-b1c3 created.

$ nori task claim shop-b1c3.1
# ... handle the interruption ...

$ nori task complete
$ nori task resume shop-a4b2.3
Task shop-a4b2.3 resumed. (Paused for 10m.)
```

The activity log captures the full story:
```
10:00  Task shop-a4b2.3 (Cut Mortises) started
10:23  Task shop-a4b2.3 paused — "Dust collector full"
10:23  Job shop-b1c3 created (maintenance)
10:33  Job shop-b1c3 completed
10:33  Task shop-a4b2.3 resumed
11:15  Task shop-a4b2.3 completed (52m active time)
```

### Execution UI — Normal Mode

When an operator opens their active task:

```
┌─────────────────────────────────────────┐
│ Job: Walnut Dining Table (Order #042)   │
│ Station: Joinery                         │
│                                          │
│ Task 3 of 12: Cut mortises (ACTIVE)     │
│ ┌──────────────────────────────────────┐ │
│ │ Timer: 00:23:15                      │ │
│ │                                      │ │
│ │ Instructions:                        │ │
│ │ Use 3/8" mortise chisel. Set fence   │ │
│ │ to 1/4" from reference face...       │ │
│ │                                      │ │
│ │ [Photo gallery from recipe]          │ │
│ │                                      │ │
│ │ [Pause] [Complete] [Add Note]        │ │
│ └──────────────────────────────────────┘ │
│                                          │
│ Next: Task 4 - Cut tenons               │
│ Remaining: 9 tasks (~2h 15m estimated)   │
└─────────────────────────────────────────┘
```

**Expandable detail**: Experienced operators see just the task title and timer.
Tap to expand for full instructions and photos.

### Post-Job Summary

After completing all tasks (or marking the job as done):
- Summary screen: total time, time per task, deviations noted
- Comparison to recipe estimates (if available)
- Prompt to promote deviations/additions to recipe
- In capture mode: prompt to create a new recipe from captured work

### API Surface

```
GET    /api/spaces/:spaceId/ready                       — Ready-work queue
GET    /api/spaces/:spaceId/ready?station=:stationId    — Station-filtered ready work

POST   /api/tasks/:id/claim                             — Claim a task (start timer)
POST   /api/tasks/:id/complete                          — Complete task (stop timer)
POST   /api/tasks/:id/pause                             — Pause task
POST   /api/tasks/:id/resume                            — Resume task
POST   /api/tasks/:id/skip                              — Skip task
PUT    /api/tasks/:id/notes                             — Add/update deviation notes
POST   /api/tasks/:id/media                             — Attach photo/video (TUS upload)

POST   /api/tasks/:parentId/children                    — Add ad-hoc child task
POST   /api/jobs/:id/summary                            — Generate post-job summary
```

## Open Questions

- Should first-time capture mode be explicit (operator chooses it) or
  automatic (system detects `RecipeID = null`)? Leaning toward automatic with
  a manual toggle.

- How should voice-to-text work? Browser API (simpler, offline) or Ollama
  transcription (better quality)? Start with browser API, add Ollama option.

- Should the execution UI be full-screen or embedded in the flow board?
  Leaning toward dedicated full-screen — operators need focus.

- Multi-person tasks (e.g., two people on a glue-up): Can a task be assigned
  to multiple operators? For v1, one assignee per task. Multiple people can
  work informally — only one person "owns" the timer.

- Should the suggest-next algorithm consider the operator's skill set or
  station proximity? Defer to later — start with simple priority ordering.

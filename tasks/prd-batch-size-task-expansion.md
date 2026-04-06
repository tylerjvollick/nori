[PRD]
# PRD: Batch-Size-Driven Task Expansion

## Overview

Replace the loop-based task expansion in Nori's recipe pour engine with a
batch-size model that reflects how real shops process work. Instead of creating
N copies of every step (one per piece), each step declares how many pieces it
processes per ticket. The pour engine calculates the number of tickets
(`order_qty / step_batch_size`) and wires dependencies correctly across batch
boundaries.

This solves the problem where a batch of 6 chairs created 6 copies of every
step, even though most steps (rough cutting, milling, shaping) process all 6
pieces as one unit of work. Only certain steps (vacuum press lamination, glue-up)
are physically constrained to process one piece at a time.

## Goals

- Each recipe step can declare a `batch_size` controlling how many pieces one
  ticket covers
- Steps without explicit `batch_size` inherit from the recipe-level default
  (typically equal to the order quantity)
- The pour engine creates the correct number of tickets per step and wires
  dependencies across batch boundaries (fan-out, 1:1, fan-in)
- Per-piece ticket titles are templatable in the TOML recipe
- The chair recipe (`seeds/chair.toml`) pours correctly with mixed batch sizes
- The existing loop mechanism is removed for fixed-count expansion (conditional
  `until` loops are out of scope — future work)

## Quality Gates

These commands must pass for every user story:

- `cd server && go test ./...` — All unit and integration tests pass
- `cd server && go vet ./...` — Static analysis clean
- `cd server && go build -o /dev/null .` — Binary builds

For the final story (end-to-end CLI test):
- Rebuild the Docker container and local binary
- Pour the chair recipe and verify correct task counts and dependency wiring
  via `nori recipe pour`, `nori job show`, and `nori ready`

## User Stories

### US-001: Add BatchSize field to Step type and TOML parsing

**Description:** As a recipe author, I want to set `batch_size` on individual
steps so that the pour engine knows how many pieces each ticket covers.

**Acceptance Criteria:**
- [ ] `Step` struct in `formula/types.go` has a `BatchSize *int` field (pointer
  so 0 vs unset is distinguishable; nil = inherit)
- [ ] TOML parser reads `batch_size = N` from step definitions
- [ ] JSON serialization includes `batch_size` with `omitempty`
- [ ] Validation: batch_size must be > 0 if set
- [ ] Existing tests still pass (field is additive)

### US-002: Add Quantity field to Task model and database migration

**Description:** As a system, I need to store how many pieces a task covers so
that operators and the UI can display "process 6 leg blanks" vs "glue 1 chair."

**Acceptance Criteria:**
- [ ] `Task` model has `Quantity int` field with `gorm:"default:1"` and
  `json:"quantity"`
- [ ] New migration file adds `quantity` column (integer, default 1, not null)
  to the `tasks` table
- [ ] Down migration drops the column
- [ ] Existing task creation still works (default 1)

### US-003: Implement batch_size resolution in the pour engine

**Description:** As the pour engine, I need to resolve each step's effective
batch_size before creating tasks, using explicit values or inheritance from the
dependency chain.

**Acceptance Criteria:**
- [ ] New function `ResolveBatchSizes(steps []*Step, defaultBatchSize int)` in
  the formula package walks the step graph and sets `BatchSize` on every step
- [ ] Steps with explicit `batch_size` keep their value
- [ ] Steps without `batch_size` inherit from their dependency chain: if all
  deps agree, use that value; if deps disagree or have no deps, use the
  recipe-level default
- [ ] Inheritance follows `Needs`/`DependsOn` references
- [ ] Unit tests cover: explicit override, single-dep inheritance, multi-dep
  agreement, multi-dep disagreement (falls back to default), no-dep default

### US-004: Implement batch-aware task creation in PourRecipe

**Description:** As the pour engine, I need to create the correct number of
tasks per step based on `order_qty / step_batch_size` and set the Quantity
field on each task.

**Acceptance Criteria:**
- [ ] `createChildTasks` (or a new function) calculates ticket count:
  `order_qty / effective_batch_size` (integer division, must divide evenly or
  error)
- [ ] For batch steps (ticket count = 1): creates 1 task with
  `Quantity = batch_size`, title is the step title as-is
- [ ] For per-piece steps (ticket count > 1): creates N tasks, each with
  `Quantity = batch_size`. Titles use the step's `title` field which can contain
  `{{n}}` (1-based piece number) and `{{batch_count}}` (total tickets for this
  step). Example: `"Glue lamination — {{n}} of {{batch_count}}"`
- [ ] Each created task is tracked in a map: `stepID -> []taskID` (list because
  per-piece steps create multiple tasks)
- [ ] Unit tests verify correct task counts for batch and per-piece steps
- [ ] The root job task and milestone task are created as before

### US-005: Implement dependency wiring across batch boundaries

**Description:** As the pour engine, I need to wire task dependencies correctly
when upstream and downstream steps have different batch sizes.

**Acceptance Criteria:**
- [ ] **1:1 (same ticket count):** Task N depends on task N. Both steps produce
  the same number of tickets.
- [ ] **Fan-out (fewer → more tickets):** All downstream tickets depend on the
  single (or fewer) upstream ticket(s). Example: `resaw-veneers` (1 ticket) →
  `glue-lamination` (6 tickets): all 6 depend on the 1.
- [ ] **Fan-in (more → fewer tickets):** The single downstream ticket depends
  on ALL upstream tickets. Example: `install-seat` (6 tickets) → `done`
  (1 ticket): done depends on all 6.
- [ ] Dependency wiring handles the case where both deps have different ticket
  counts that converge on the same downstream step (e.g., `glue-up` depends on
  `dry-fit` (1 ticket, fan-out) and `shape-back` (6 tickets, 1:1))
- [ ] Unit tests cover all three wiring patterns plus mixed-dependency
  convergence

### US-006: Remove fixed-count loop expansion from pour pipeline

**Description:** As a developer, I want to remove the loop-based batch
expansion so there's one clear mechanism for batch production.

**Acceptance Criteria:**
- [ ] The `ApplyLoops` fixed-count path, `expandLoop`, `chainExpandedIterations`,
  and related helpers are removed from `controlflow.go`
- [ ] `LoopSpec` retains `Until`, `Max`, `Range`, `Var` fields for future
  conditional/range loop use but `Count`, `CountExpr`, and `Parallel` are removed
- [ ] `normalizeLoopBodies` in `parser.go` is removed (no longer needed since
  steps are flat, not nested in loop bodies)
- [ ] `ResolveLoopCounts` is removed from `recipe.service.go`
- [ ] `ApplyControlFlow` no longer calls `ApplyLoops` for fixed-count expansion
  (retain branch and gate application)
- [ ] All tests referencing fixed-count loop expansion are updated or removed
- [ ] Conditional loop (`until`) tests are retained if they exist

### US-007: Update chair recipe and end-to-end CLI test

**Description:** As a shop operator, I want to pour the chair recipe and see
the correct tasks with proper batch sizes and dependency wiring, then walk
through the claim/complete workflow.

**Acceptance Criteria:**
- [ ] `seeds/chair.toml` uses the new flat batch_size model (no loop)
- [ ] `nori recipe create --from-toml seeds/chair.toml` imports successfully
- [ ] `nori recipe pour dining-chair --var batch_size=6` creates the expected
  tasks:
  - Legs stream: 8 steps × 1 ticket each = 8 tasks (quantity=6 each)
  - Rails stream: 7 steps × 1 ticket each = 7 tasks (quantity=6 each)
  - Dry fit: 1 ticket (quantity=6)
  - Back stream: resaw-veneers = 1 ticket (quantity=6), glue-lamination through
    shape-back = 5 steps × 6 tickets each = 30 tasks (quantity=1 each)
  - Seat stream: 4 steps × 1 ticket each = 4 tasks (quantity=6 each)
  - Assembly: glue-up through spray-finish = 2 steps × 6 tickets each = 12
    tasks (quantity=1 each)
  - Install seat: 6 tickets (quantity=1 each)
  - Done milestone: 1 ticket
  - Root job: 1 ticket
  - **Total: ~71 tasks** (verify exact count)
- [ ] `nori ready` shows the correct starting tasks (all batch steps with no
  deps: leg-rough-cut, rail-rough-cut, resaw-veneers, cut-seat-blank)
- [ ] `nori task claim` and `nori task complete` work on batch tasks
- [ ] Completing `resaw-veneers` (batch) unlocks all 6 `glue-lamination` tickets
- [ ] Completing `glue-lamination — 1 of 6` unlocks `true-up-edge — 1 of 6`
  (1:1 wiring)
- [ ] Completing all 6 `install-seat` tickets allows `done` milestone to show
  as ready

## Functional Requirements

- FR-1: `batch_size` is an optional integer field on recipe steps
- FR-2: If not set, `batch_size` inherits from dependencies; if no deps or
  deps disagree, falls back to recipe-level `batch_size` variable
- FR-3: The pour engine calculates `ticket_count = order_qty / batch_size` per
  step (must divide evenly)
- FR-4: Per-piece tickets support `{{n}}` and `{{batch_count}}` template
  variables in titles
- FR-5: Dependencies across batch boundaries use fan-out, 1:1, or fan-in
  wiring based on relative ticket counts
- FR-6: The `Task.Quantity` field stores how many pieces the task covers
- FR-7: The ready-work algorithm requires no changes (same dep graph logic)

## Non-Goals

- Partial completion tracking (e.g., "4 of 6 done" on a batch ticket)
- Conditional loops (`until`) — retained in types but not implemented
- Range loops — retained in types but not implemented
- Station assignment during pour (existing limitation, separate feature)
- WIP limits on stations (separate feature)
- UI changes (CLI only for now)

## Technical Considerations

- The `order_qty` must be passed to the pour engine. Currently it's derived
  from the `batch_size` variable. The recipe-level `batch_size` var serves
  double duty as both "how many to make" and "default batch size per step."
  This is intentional — for most steps, you process all pieces at once.
- Integer division must be exact: `order_qty % step_batch_size == 0`. If a
  recipe author sets `batch_size = 4` on a step with `order_qty = 6`, that's
  a validation error at pour time.
- The `stepID -> []taskID` map replaces the current `stepID -> taskID` map
  in the pour engine. This is the key data structure change.

## Success Metrics

- Chair recipe pours correctly with mixed batch sizes
- Total task count is dramatically lower than the loop model (~71 vs ~175)
- Dependency wiring is correct (verified by ready-work algorithm)
- End-to-end CLI workflow (create → pour → ready → claim → complete) works

## Open Questions

- Should `batch_size` inheritance be specced more formally? Current rule:
  "inherit from deps if they all agree, else use default." Is there a case
  where deps disagree and the step should inherit from one specific dep?
- Should we support non-integer batch divisions in the future (e.g., 7 chairs
  with batch_size=4 → 1 batch of 4 + 1 batch of 3)?
[/PRD]

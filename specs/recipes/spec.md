# Feature Specification: Recipe System

**Created**: 2026-04-23
**Status**: Draft
**Labels**: recipes, core

## Overview

Recipes are reusable process templates that describe how to build a product or
perform a process. A recipe is a task tree — the same data structure used for
jobs — frozen as a versioned template. When a shop receives an order, a manager
"rolls" a recipe to create a production job with the correct steps, station
assignments, dependencies, and batch sizing for the order quantity.

Recipes close the knowledge loop: operators build things on the shop floor,
and the work they do becomes a reusable template for next time — without
anyone having to "write documentation."

### Key Concepts

- **Recipe** — A named, versioned task-tree template for a product or process.
- **Roll** — Create a production job from a recipe. Clones the recipe's task
  tree, applies batch expansion for the order quantity, and wires dependencies.
- **Save-as-recipe** — Capture a completed job as a new recipe template.
- **Publish** — Freeze a recipe draft into an immutable snapshot. Rolling
  always uses the published version.

## User Scenarios & Testing

### User Story 1 - Author a Recipe (Priority: P1)

**Labels**: recipes, authoring

A shop owner wants to document how to build their most common product — a
dining table — so that any operator can follow the process and produce
consistent results. They create a recipe by building out a task tree: top-level
phases (Mill, Joinery, Assembly, Finish, QC), each with detailed sub-steps,
station assignments, estimated times, and dependencies between phases.

**Why this priority**: Without recipes, every job is built from scratch. This
is the foundation that all other recipe features depend on.

**Independent Test**: Create a recipe with nested steps, station assignments,
estimated times, batch sizes, and inter-step dependencies. Verify the recipe
can be viewed, edited, and the task tree renders correctly.

**Acceptance Scenarios**:

1. [ ] **Given** a logged-in shop owner, **When** they navigate to Recipes and
   click "New Recipe", **Then** a new recipe is created with an empty task tree
   and a draft version.
2. [ ] **Given** a recipe in draft, **When** the owner adds steps with titles,
   descriptions, station assignments, estimated times, and batch sizes, **Then**
   the recipe task tree reflects the changes.
3. [ ] **Given** a recipe with multiple steps, **When** the owner adds
   dependencies between steps (e.g., "Joinery" needs "Mill"), **Then** the
   dependency is saved and visible in the tree.
4. [ ] **Given** a recipe with nested steps, **When** the owner reorders steps
   via drag-and-drop or move controls, **Then** the display order updates
   correctly.
5. [ ] **Given** a recipe being edited, **When** the owner edits a step's
   details (title, description, station, estimated time, batch size), **Then**
   changes are persisted immediately.

---

### User Story 2 - Roll a Recipe into a Production Job (Priority: P1)

**Labels**: recipes, roll, jobs

A production manager receives an order for 30 bistro tables. They open the
"Bistro Table" recipe, click "Roll", enter quantity 30, and the system creates
a production job. Steps marked as batch operations (e.g., "cut all table tops")
create one task regardless of quantity. Steps marked as per-piece operations
(e.g., "hand sand each table top") create 30 individual tasks. Dependencies
between batch and per-piece steps are wired correctly (fan-out from one
batch-cut task to 30 individual sanding tasks).

**Why this priority**: Rolling is the core value proposition — turn a template
into actionable work on the shop floor. Without this, recipes are just
documentation.

**Independent Test**: Roll a published recipe with quantity > 1. Verify the
job has the correct number of tasks, batch expansion is correct, and
dependencies are wired with proper fan-in/fan-out patterns.

**Acceptance Scenarios**:

1. [ ] **Given** a published recipe, **When** a manager clicks "Roll" and
   enters quantity 1, **Then** a job is created with one task per recipe step,
   preserving the full tree structure and all dependencies.
2. [ ] **Given** a published recipe with batch sizes, **When** a manager rolls
   with quantity 30, **Then** per-piece steps (batch_size=1) create 30 tasks
   and batch steps (batch_size=30) create 1 task.
3. [ ] **Given** a rolled job with mixed batch/per-piece steps, **When**
   viewing the dependency graph, **Then** fan-out (1 batch task to N per-piece
   tasks) and fan-in (N per-piece tasks to 1 batch task) dependencies are
   correct.
4. [ ] **Given** a rolled job, **When** viewing any task, **Then** the task
   shows which recipe and version it was rolled from (traceability).
5. [ ] **Given** a published recipe, **When** a manager rolls with an optional
   customer and due date, **Then** the job is linked to the customer with the
   due date set.

---

### User Story 3 - Track Job Cost vs Estimate (Priority: P1)

**Labels**: recipes, cost-tracking, quoting

A production manager wants to know whether a completed job came in over or
under budget. They open the job detail and see a cost summary: estimated labor
cost (from the recipe's estimated times multiplied by the shop's labor rate)
vs actual labor cost (from time tracked during execution). The summary breaks
down by station so the manager can see exactly where time was lost — e.g.,
"Finishing took 40% longer than estimated."

**Why this priority**: This is the key insight for the investor demo. The
ability to compare estimated vs actual cost at the station level is what
enables smarter quoting over time.

**Independent Test**: Complete tasks in a job that was rolled from a recipe
with estimated times. View the cost summary. Verify estimated and actual
values are correct, variance is calculated, and station-level breakdown is
shown.

**Acceptance Scenarios**:

1. [ ] **Given** a job rolled from a recipe with estimated times, **When** a
   manager opens the job cost summary, **Then** they see total estimated labor
   cost (sum of estimated_time_secs x labor rate across all steps).
2. [ ] **Given** a job with completed tasks that have tracked time, **When**
   viewing the cost summary, **Then** actual labor cost is shown (sum of
   actual time x labor rate).
3. [ ] **Given** estimated and actual costs, **When** viewing the cost summary,
   **Then** variance is shown as both an amount and a percentage, color-coded
   green (under budget) or red (over budget).
4. [ ] **Given** a job with tasks assigned to different stations, **When**
   viewing the cost summary, **Then** a station-level breakdown shows
   estimated vs actual hours per station.
5. [ ] **Given** a job with some tasks completed and some in progress, **When**
   viewing the cost summary, **Then** the summary shows partial actuals with
   a note that the job is incomplete.

---

### User Story 4 - Publish a Recipe Version (Priority: P2)

**Labels**: recipes, versioning

A shop owner has been editing a recipe draft and is satisfied with the changes.
They click "Publish" and the draft becomes the active version. Any future rolls
use this version. Previously rolled jobs are unaffected — they still reference
the version they were rolled from. The old published version is automatically
archived.

**Why this priority**: Versioning enables safe iteration. Without it, editing
a recipe would retroactively affect in-flight jobs.

**Independent Test**: Publish a draft, verify it becomes the current version.
Roll a job from v1, publish v2, verify the job still references v1.

**Acceptance Scenarios**:

1. [ ] **Given** a recipe with a draft version, **When** the owner clicks
   "Publish", **Then** the draft becomes the published version and the recipe's
   current version pointer is updated.
2. [ ] **Given** a previously published version, **When** a new version is
   published, **Then** the old version is archived (read-only) and still
   accessible in the version history.
3. [ ] **Given** a job rolled from version 1, **When** version 2 is published,
   **Then** the job still references version 1 and is unaffected by the change.
4. [ ] **Given** a published recipe, **When** the owner clicks "New Version",
   **Then** a new draft is created as a copy of the current published version
   for editing.

---

### User Story 5 - Save a Completed Job as a Recipe (Priority: P2)

**Labels**: recipes, save-as-recipe

An operator just completed a first-time build — a custom console table they
figured out as they went. The job has a complete task tree with steps, times,
and station assignments captured during execution. They click "Save as Recipe",
give it a name, and the system creates a new recipe from the job's task tree.
Actual times from the job populate the recipe's estimated times. The operator
(or a manager) can then clean up the recipe and publish it.

**Why this priority**: This is the "documentation as a side effect" flow. It
lets shops build their recipe library from real work without anyone having to
sit down and write process documents.

**Independent Test**: Complete a job with tracked time, save it as a recipe.
Verify the recipe has the correct structure, estimated times are populated
from actuals, and runtime data (status, assignee, timestamps) is stripped.

**Acceptance Scenarios**:

1. [ ] **Given** a completed job, **When** the operator clicks "Save as Recipe"
   and enters a name, **Then** a new recipe is created with a draft version
   whose task tree matches the job's structure.
2. [ ] **Given** a job with actual times recorded, **When** saved as a recipe,
   **Then** the recipe's estimated times are populated from the job's actual
   times.
3. [ ] **Given** a job with runtime data (status, assignee, started_at,
   completed_at), **When** saved as a recipe, **Then** the recipe's task tree
   does not contain runtime data — all tasks are reset to open/unassigned.
4. [ ] **Given** a job with ad-hoc tasks added during execution, **When** saved
   as a recipe, **Then** the ad-hoc tasks are included in the recipe.

---

### User Story 6 - Estimate Job Cost from Recipe (Priority: P2)

**Labels**: recipes, quoting, cost-tracking

Before rolling a recipe, a manager wants to know what the job will cost. They
view the recipe detail and see an estimated cost based on the recipe's
estimated times and the shop's labor rate. For a batch of 30 tables, they can
enter the quantity and see a scaled estimate that accounts for batch vs
per-piece steps.

**Why this priority**: This is the foundation for quoting. A shop that can
estimate job cost before accepting an order can quote with confidence and
track margin over time.

**Independent Test**: View a recipe with estimated times. See the cost
estimate. Change the quantity and verify the estimate scales correctly
(per-piece steps scale linearly, batch steps don't).

**Acceptance Scenarios**:

1. [ ] **Given** a published recipe with estimated times on all steps, **When**
   a manager views the recipe, **Then** they see a total estimated labor cost
   (sum of estimated times x labor rate).
2. [ ] **Given** a recipe, **When** the manager enters a quantity (e.g., 30),
   **Then** the estimate scales correctly: per-piece step costs multiply by
   quantity, batch step costs remain fixed.
3. [ ] **Given** a recipe without estimated times on some steps, **When**
   viewing the cost estimate, **Then** the estimate shows a partial total
   with a note indicating which steps lack estimates.

---

### User Story 7 - Browse and Manage Recipes (Priority: P2)

**Labels**: recipes, ui

A shop owner wants to see all the recipes in their shop at a glance. They
navigate to the Recipes page and see a list with name, description, current
version status, how many times each recipe has been rolled, and when it was
last rolled. They can search by name and click through to any recipe's detail
page.

**Why this priority**: Recipes need to be discoverable. A list page is the
minimum browsing experience.

**Independent Test**: Create several recipes. Navigate to the Recipes list
page. Verify all recipes are shown with correct metadata. Click through to
a recipe detail page.

**Acceptance Scenarios**:

1. [ ] **Given** a space with multiple recipes, **When** the owner navigates
   to the Recipes page, **Then** they see a list of all recipes with name,
   description, version status, roll count, and last rolled date.
2. [ ] **Given** the recipes list, **When** the owner clicks a recipe, **Then**
   they navigate to the recipe detail page showing the task tree.
3. [ ] **Given** the recipes list, **When** the owner clicks "New Recipe",
   **Then** a new recipe is created and they navigate to its detail page.

---

### Edge Cases

- What happens when a recipe is rolled with a quantity that doesn't divide
  evenly by a step's batch size? (Error? Round up? Allow partial batches?)
- What happens when a recipe is deleted but jobs still reference it? (Jobs
  retain their task tree — the recipe link becomes informational only.)
- What happens when an operator tries to save a job as a recipe but the job
  has no tasks? (Prevent with validation.)
- What happens when two people edit the same recipe draft simultaneously?
  (Last-write-wins for v1. Collaborative editing is out of scope.)
- What happens when a recipe step references a station that no longer exists?
  (Station assignment becomes null; warn during roll.)

## Requirements

### Functional Requirements

- **FR-001**: System MUST allow users to create recipes as named, versioned
  task-tree templates within a space.
- **FR-002**: System MUST allow users to add, edit, remove, and reorder steps
  in a recipe draft, including nested sub-steps to unlimited depth.
- **FR-003**: System MUST allow users to assign stations, estimated times, and
  batch sizes to recipe steps.
- **FR-004**: System MUST allow users to define dependencies between recipe
  steps (step A must complete before step B can start).
- **FR-005**: System MUST support recipe versioning with draft, published, and
  archived states. Only one draft and one published version may exist at a time.
- **FR-006**: System MUST create a frozen snapshot of the recipe task tree when
  a version is published. Publishing MUST NOT affect in-flight jobs.
- **FR-007**: System MUST allow users to roll a published recipe into a
  production job, specifying an order quantity.
- **FR-008**: System MUST expand batch sizes during roll: per-piece steps
  create N tasks (one per unit), batch steps create 1 task covering all units.
- **FR-009**: System MUST wire dependencies correctly during roll, handling
  1:1 (same ticket count), fan-out (batch to per-piece), and fan-in (per-piece
  to batch) patterns.
- **FR-010**: System MUST track which recipe and version a rolled job came
  from (traceability).
- **FR-011**: System MUST allow users to save a completed or in-progress job
  as a new recipe, populating estimated times from actual execution times.
- **FR-012**: System MUST compute actual labor cost from tracked time and the
  space's labor rate.
- **FR-013**: System MUST provide a job cost summary comparing estimated cost
  (from recipe) to actual cost (from time tracking), broken down by station.
- **FR-014**: System MUST provide a recipe-level cost estimate that scales
  with order quantity, accounting for batch vs per-piece step costing.
- **FR-015**: System MUST display recipes in a browsable list with metadata
  (name, version status, roll count, last rolled date).
- **FR-016**: System MUST provide a shared task-tree editing interface used
  for both recipe authoring and job task management, with context-appropriate
  fields.

### Key Entities

- **Recipe** — Identity record: name, description, space, current version
  pointer, created by.
- **RecipeVersion** — Versioned snapshot: version number, status
  (draft/published/archived), pointer to a frozen task tree, change summary.
- **Task** — Universal work item used for both recipe steps and job tasks.
  Recipe steps have Type='recipe'. Job tasks have Type='job'. Both share the
  same schema, tree structure, and dependency system.
- **CostEntry** — Tracks labor cost per task, computed from time events and
  labor rate.
- **TimeEvent** — Append-only time log: check-in, check-out, pause, resume.
  Source of truth for actual labor time.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A shop owner can author a recipe and roll it into a job in under
  5 minutes.
- **SC-002**: A job rolled from a recipe with 10 steps and quantity 30 has the
  correct number of tasks (accounting for batch sizes) with all dependencies
  wired correctly.
- **SC-003**: After completing a job, the cost summary shows estimated vs
  actual labor cost with less than 1% rounding error compared to manual
  calculation.
- **SC-004**: A first-time build saved as a recipe produces a complete template
  that can be rolled into a new job without manual editing.
- **SC-005**: Recipe versioning preserves in-flight job integrity: rolling from
  v1, publishing v2, and inspecting the job confirms it still references v1.

## Assumptions

- The shop has a configured labor rate (Space.DefaultLaborRate). If not set,
  cost calculations are skipped gracefully — time is still tracked.
- Recipes are scoped to a single space. Cross-space recipe sharing is out of
  scope for v1.
- The task tree editing UI is shared between recipe authoring and job task
  management. Context-specific fields (estimated time for recipes, status for
  jobs) are shown/hidden based on the root task type.
- Batch sizes must divide evenly into order quantities. Partial batches are
  not supported in v1.
- Material/BOM cost tracking is out of scope for this spec. See
  `specs/inventory/spec.md` for the materials direction.
- Custom fields (e.g., finish type selection, wood species options) are out
  of scope for this spec. The recipe system provides the structural foundation
  that custom fields will extend later.
- Time tracking (TimeEvent creation on task transitions) is a prerequisite
  that is being built in parallel (see task execution epic nori-40n.4).

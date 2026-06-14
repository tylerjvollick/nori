# Data Model

## Who

All developers working on Nori. This spec is the foundation — every other spec
references the entities defined here.

## What

The core data model for Nori, covering all entities and their relationships.
This is a relational model backed by PostgreSQL, accessed via GORM in the Go
backend.

## Where

- Backend: `server/internal/models/`
- Database: PostgreSQL, managed via migration files in `server/migrations/`

## Why

A well-designed data model is the backbone of the entire system. Getting this
right means task execution, recipe authoring, time tracking, and analytics all
have clean, consistent data to work with.

The model is designed around these principles:

1. **Everything is a Task** — One universal work item replaces the old
   Ticket/TicketType/StatusDefinition/TicketStep/TicketSubStep/TicketLink and
   SOP execution models. Tasks have hierarchical string IDs (like beads), a
   dependency graph, and fixed status/type enums.

2. **Recipes replace SOPs** — Process templates are versioned TOML documents
   stored in Recipe + RecipeVersion tables. A formula engine (extracted from
   beads) processes TOML into task graphs. No filesystem storage — non-technical
   users shouldn't need to know what git is.

3. **Pull, not push** — The dependency graph and ready-work algorithm determine
   what work is available. WIP limits at stations enforce capacity constraints.
   The bottleneck surfaces naturally from the data.

4. **Time data is first-class** — Every action that consumes time is recorded
   so bottlenecks surface naturally.

5. **AI-ready, not AI-dependent** — The structured model supports both CRUD
   (web UI, iPad) and conversational interaction (via LLM/MCP). AI is an
   interface layer on top, not baked in.

## How

### Design Decisions

- **Fixed status and type enums** — not configurable per-space. Manufacturing
  shops share the same workflow primitives. Simplifies the model, the UI, and
  the query layer.
- **Beads-style hierarchical string IDs** for Tasks (e.g., `task-a3f8c127.1.2`).
  The root task (Job) gets a generated ID; children get dot-suffixed ordinals.
- **Dependency graph** replaces linear station-column flow. Tasks declare what
  they block or wait for. The ready-work algorithm finds unblocked leaf tasks.
- **Live edits → Recipe promotion** — Operators can add ad-hoc tasks during
  execution. These can be diffed against the source recipe and promoted back
  as a new RecipeVersion.
- **Fresh database** — no migration path from old Ticket/SOP tables. Start
  clean with the new schema.

### Entity Overview

```
Account
  └── Space
        ├── User (via SpaceMember, with roles)
        ├── Station (physical shop locations — optional)
        │
        ├── Task (universal work item — hierarchical string IDs)
        │     ├── ParentID (string FK → Task, nullable — root = Job)
        │     ├── Type: enum (job, task, milestone, gate)
        │     ├── Status: enum (open, active, paused, done, skipped, cancelled)
        │     ├── RecipeVersionID (FK, nullable — source recipe)
        │     ├── StationID (FK, nullable — where this work happens)
        │     ├── TaskDep[] (dependency edges)
        │     ├── TaskMedia[] (photos/videos captured during execution)
        │     ├── TaskComment[] (operator notes, AI suggestions)
        │     ├── ActivityEntry[] (chronological log)
        │     ├── TimeEvent[] (time tracking)
        │     └── CostEntry[] (cost tracking)
        │
        ├── Recipe (process template identity)
        │     └── RecipeVersion[] (versioned TOML content)
        │           └── BOMItem[] (materials per version)
        │
        ├── Customer
        ├── Material (inventory items)
        └── Tag (reusable labels)
```

### Key Entities

---

#### Account & Space (existing, see auth-and-tenancy.md)

Carried forward from the existing codebase. An Account is a billing entity.
Spaces are workspaces within an Account (e.g., "Main Shop", "Sales",
"Finishing Room").

#### User (existing, see auth-and-tenancy.md)

Carried forward. Users belong to Accounts via UserAccount with roles. Users
operate within Spaces via SpaceMember.

---

#### Station (see stations.md)

A physical location or tool in the shop where work happens. Stations are
optional — non-physical spaces (sales, admin) don't use them.

Stations serve two purposes:
1. **Reference on tasks** — "this task happens at the table saw."
2. **WIP limits and flow** — the station view shows what work is at each
   station and enforces capacity constraints.

```
Station
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string ("Table Saw", "Jointer", "Assembly Bench", "Finish Room")
  - Description: text (nullable)
  - DisplayOrder: int (position on the station view)
  - WIPLimit: int (max concurrent active tasks at this station)
  - BufferSize: int (queue capacity before the station)
  - CostsHour: decimal (nullable — hourly labor rate; falls back to
    Space.DefaultLaborRate)
  - IsActive: bool
  - CreatedAt, UpdatedAt: timestamp
```

Note: In a small shop a single job bounces between stations as dictated by the
recipe. Step 1 (rip blanks) is at the table saw, step 2 (face joint) is at the
jointer, step 3 might be back at the table saw. The task determines the station,
not the other way around. WIP limits are enforced at the station level: "only 1
active task at the table saw at a time."

---

#### Task (new — replaces Ticket, TicketStep, Job, JobStep)

The universal work item. Everything is a Task. A root Task with no parent is
a **Job** (user-facing term). Children are **Tasks** (user-facing). The backend
model is `Task` for both.

**Hierarchical string IDs** (beads-style):
- Root: `task-a3f8c127` (fixed `task-` prefix + random hex)
- Child: `task-a3f8c127.1`
- Grandchild: `task-a3f8c127.1.2`

ID generation: root IDs are `task-{8-hex}` (8 hex chars from a random UUID).
Child IDs append `.{ordinal}` where ordinal is the next sequential integer
among siblings, so the `task-` prefix cascades to the entire tree.

```
Task
  - ID: string (primary key — hierarchical, e.g. "task-a3f8c127.1.2")
  - SpaceID: uuid (FK → Space)
  - ParentID: string (FK → Task, nullable — null = root/Job/Recipe)
  - RecipeID: uuid (FK → Recipe, nullable)
  - RecipeVersionID: int (FK → RecipeVersion, nullable — snapshot at creation)
  - StationID: uuid (FK → Station, nullable)
  - CustomerID: uuid (FK → Customer, nullable — typically on root/Job)
  - AssignedToID: uuid (FK → User, nullable)
  - CreatedByID: uuid (FK → User)
  - Type: enum (job, task, milestone, gate, recipe)
  - Status: enum (open, active, paused, done, skipped, cancelled)
  - Title: string
  - Description: text (nullable)
  - Priority: int (lower = higher priority, default 0)
  - DisplayOrder: int (position among siblings)
  - DueDate: timestamp (nullable)
  - StartedAt: timestamp (nullable)
  - PausedAt: timestamp (nullable)
  - CompletedAt: timestamp (nullable)
  - ActualTimeSeconds: int (accumulated active time, excludes pauses)
  - BatchSize: int (nullable — units covered per task, used for recipe batch expansion)
  - EstimatedTimeSecs: int (nullable — expected duration, used for quoting/comparison)
  - DeviationNotes: text (nullable — "what I did differently")
  - Metadata: jsonb (nullable — extensible structured data)
  - CreatedAt, UpdatedAt: timestamp
```

**Type enum:**
- `job` — Root-level work item (an order, a build, a maintenance task). Always
  has `ParentID = null`.
- `task` — A step within a job or recipe. The workhorse. Can be nested (task
  within task).
- `milestone` — A marker with no work of its own. Useful for "all prep complete"
  checkpoints that other tasks depend on.
- `gate` — A hold point requiring explicit approval (QC check, cure timer,
  manager sign-off). Gates block downstream tasks until resolved.
- `recipe` — Root of a recipe task tree (template). A recipe's children are
  `task`, `milestone`, or `gate` type, same as jobs. Recipes and jobs share
  the same schema — a recipe is a frozen template, a job is a live instance.

**Status enum:**
- `open` — Not started, waiting for dependencies or assignment.
- `active` — In progress. Timer running.
- `paused` — Temporarily stopped (lunch, interruption).
- `done` — Completed successfully.
- `skipped` — Not applicable for this execution.
- `cancelled` — Abandoned.

**Recipe linkage**: Recipes and jobs are both task trees. A recipe is a
template (root `Type = 'recipe'`); a job is a live instance (root
`Type = 'job'`). When a recipe is "rolled" into a job, the recipe's task tree
is deep-cloned with batch expansion based on order quantity. Each generated
child Task references the `RecipeID` and `RecipeVersionID` it was rolled from.
Ad-hoc tasks added during execution have `RecipeVersionID = null`.

**Batch fields** (used on recipe steps, copied to job tasks during roll):
- `BatchSize` — How many units this step covers. `NULL` or `1` means per-piece
  (one task per unit). A value matching the order quantity means the step is
  done as a single batch. During roll: `ticket_count = order_qty / batch_size`.
- `EstimatedTimeSecs` — Expected duration for this step. Used for cost
  estimation on recipes and estimated-vs-actual comparison on jobs.

See `specs/recipes/architecture.md` for the full roll engine design.

---

#### TaskDep (new — replaces TicketLink)

Dependency edges between tasks. These drive the ready-work algorithm.

```
TaskDep
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - SourceTaskID: string (FK → Task — the task that has the dependency)
  - TargetTaskID: string (FK → Task — the task being depended on)
  - DepType: enum (blocks, waits_for, related)
  - CreatedByID: uuid (FK → User)
  - CreatedAt: timestamp
```

**DepType enum:**
- `blocks` — SourceTask blocks TargetTask. TargetTask cannot start until
  SourceTask is done. (Read: "source blocks target" or equivalently "target
  waits for source".)
- `waits_for` — Soft dependency. Target is informed but not blocked.
- `related` — Informational link. No scheduling effect.

**Cycle detection**: The service layer must reject dependency additions that
would create cycles. The beads algorithm for this is portable.

**Cross-job dependencies**: TaskDeps can link tasks across different jobs
(e.g., a finishing task waits for all assembly tasks across a batch order).

---

#### TaskMedia (new — replaces SOPStepMedia on execution side)

Photos and videos captured during task execution. Separate from recipe media
(which lives in the TOML or as RecipeVersion attachments).

```
TaskMedia
  - ID: uuid
  - TaskID: string (FK → Task)
  - FilePath: string (public URL path, e.g. /uploads/task-media/<uuid>.jpg)
  - FileName: string (original client filename)
  - MimeType: string
  - FileSize: int64
  - Duration: int (nullable — for videos, in seconds)
  - DisplayOrder: int
  - CapturedByID: uuid (FK → User, nullable — set null if user deleted)
  - CreatedAt: timestamp
```

Implemented in migration `000052_add_task_media`. Uploads are simple multipart
form posts (field `file`) handled by `internal/storage.LocalStorage`, which
validates the MIME type against `ALLOWED_MIME_TYPES`, writes the file under
`UPLOAD_DIR` (served statically at `/uploads`), and stores the public path in
`FilePath`. (The earlier plan called for a TUS chunked-upload system; that was
not built — simple disk-backed upload covers the photo/short-video use case.)

Sub-task images use the same storage layer via the pre-existing
`sub_task_images` table (`SubTaskImage`, migration `000045`).

---

#### TaskComment (new — replaces SOPComment + ActivityEntry comments)

Comments on tasks. Supports operator notes, AI suggestions, and suggested
edits.

```
TaskComment
  - ID: uuid
  - TaskID: string (FK → Task)
  - UserID: uuid (FK → User)
  - Body: text
  - IsSuggestion: bool (suggested edit vs. regular comment)
  - IsResolved: bool
  - CreatedAt, UpdatedAt: timestamp
```

---

#### Recipe (new — replaces SOPTemplate)

A process template identity. User-facing name is "Recipe" (sushi theme).

```
Recipe
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Slug: string (URL-friendly identifier, e.g. "walnut-dining-table")
  - CurrentVersionID: int (FK → RecipeVersion, nullable — latest published)
  - ExtendsRecipeID: uuid (FK → Recipe, nullable — inheritance, deferred)
  - CreatedByID: uuid (FK → User)
  - IsActive: bool
  - CreatedAt, UpdatedAt: timestamp
```

**Name and Description** are derived from the current version's root task
(`RecipeVersion.RootTaskID → Task.Title/Description`). When no current version
or root task exists, the recipe slug is used as the display name. This avoids
duplicating title/description across the recipe and its root task.

**Inheritance** (deferred): Recipes can extend other recipes. A "Walnut Dining
Table" recipe might extend a "Generic Dining Table" recipe, overriding specific
steps. This feature exists in the schema but is not implemented in the current
roll engine. See `specs/recipes/architecture.md` for the deferred features list.

---

#### RecipeVersion (new — replaces SOPVersion)

Versioned recipe snapshot. Each version points to a frozen task tree.

```
RecipeVersion
  - ID: int (auto-increment)
  - RecipeID: uuid (FK → Recipe)
  - VersionNumber: int (auto-incremented per recipe)
  - Status: enum (draft, published, archived)
  - RootTaskID: string (FK → Task, nullable — points to frozen recipe task tree)
  - Content: text (nullable — deprecated TOML source, kept for migration)
  - ChangeSummary: text (nullable — what changed)
  - CreatedByID: uuid (FK → User)
  - CreatedAt: timestamp
```

**Version lifecycle**:
- `draft` — Work in progress. Only the author and managers can see it.
  One active draft per recipe at a time. The draft's task tree is editable.
- `published` — The active version. Rolling a recipe uses this version.
  The published version's task tree is a frozen snapshot (clone-on-publish).
- `archived` — Previous published version. Read-only.

When published, `Recipe.CurrentVersionID` is updated to point to the new
version. In-flight jobs are not affected — they reference the RecipeVersion
that was current when they were rolled.

**Task tree storage**: Instead of storing the recipe definition as TOML text,
each version points to a root Task via `RootTaskID`. The recipe's steps are
child Tasks in the same `task` table used for jobs. This means:
- Recipe data is queryable (find all recipes using station X, compute
  estimated costs across recipes, etc.)
- The same UI components and API endpoints work for both recipe editing and
  job task management
- Diff/promote between a job and its source recipe is a direct tree comparison

The `Content` column (TOML text) is deprecated and will be dropped in a future
migration. See `specs/recipes/architecture.md` for the full architecture.

---

#### ActivityEntry (existing concept, re-pointed)

Chronological log of everything that happens on a task. Re-pointed from
Ticket → Task.

```
ActivityEntry
  - ID: uuid
  - TaskID: string (FK → Task)
  - UserID: uuid (FK → User)
  - EntryType: enum (status_change, task_started, task_completed, task_paused,
                      task_resumed, task_skipped, comment, interruption,
                      assignment_change, dep_added, dep_removed, recipe_edited,
                      cost_logged, task_created, media_added)
  - Description: text (human-readable summary)
  - LinkedTaskID: string (FK → Task, nullable — for interruptions/references)
  - DurationSeconds: int (nullable — for interruptions)
  - OldValue: text (nullable — for field changes)
  - NewValue: text (nullable — for field changes)
  - Metadata: jsonb (nullable — structured data for specific entry types)
  - CreatedAt: timestamp
```

---

#### TimeEvent (existing concept, re-pointed)

Append-only, source-agnostic time log. Re-pointed from Ticket/TicketStep →
Task.

```
TimeEvent
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - UserID: uuid (FK → User)
  - TaskID: string (FK → Task, nullable)
  - StationID: uuid (FK → Station, nullable)
  - EventType: enum (check_in, check_out, pause, resume)
  - Source: enum (manual, web, cli, tap, sensor, api, system)
  - Timestamp: timestamp (when it happened — can be backdated)
  - Notes: text (nullable)
  - CreatedAt: timestamp (when the record was created)
```

---

#### CostEntry (existing concept, re-pointed)

Cost tracking per task. Re-pointed from Ticket → Task.

```
CostEntry
  - ID: uuid
  - TaskID: string (FK → Task)
  - CostType: enum (labor, material, consumable, marketing, other)
  - Description: string
  - Amount: decimal (total cost for this entry)
  - Quantity: decimal (nullable)
  - Unit: string (nullable — hours, board_feet, each)
  - UnitCost: decimal (nullable)
  - MaterialID: uuid (FK → Material, nullable)
  - TimeEventID: uuid (FK → TimeEvent, nullable)
  - CreatedByID: uuid (FK → User)
  - CreatedAt: timestamp
```

---

#### BOMItem (existing concept, re-pointed)

A line item on a bill of materials, tied to a recipe version.

```
BOMItem
  - ID: uuid
  - RecipeVersionID: int (FK → RecipeVersion)
  - MaterialID: uuid (FK → Material, nullable — for ad-hoc items)
  - Name: string ("White Oak 4/4")
  - Quantity: decimal
  - Unit: string (board_feet, count, oz, gallons, each)
  - StepRef: string (nullable — which recipe step uses this, by step key)
  - UnitCost: decimal (nullable — cost per unit for estimation)
  - Notes: text (nullable)
```

---

#### Material (see materials-and-bom.md)

A stock item tracked in inventory.

```
Material
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string
  - Category: enum (lumber, hardware, finish, consumable, other)
  - Unit: string (board_feet, count, oz, gallons, each)
  - CurrentStock: decimal
  - ReorderThreshold: decimal
  - ReorderQuantity: decimal
  - Location: string (nullable)
  - UnitCost: decimal (nullable — current cost per unit)
  - Supplier: string (nullable)
  - SKU: string (nullable)
  - IsActive: bool
  - CreatedAt, UpdatedAt: timestamp
  - DeletedAt: timestamp (nullable — GORM soft delete)
```

---

#### TaskMaterial (new — quoting foundation, see materials-and-bom.md)

Links a recipe/job task to a catalog material with the quantity consumed per
unit produced. Scaled by batch quantity during dry-run cost computation.

```
TaskMaterial
  - ID: uuid
  - TaskID: string (FK → Task, cascade delete)
  - MaterialID: uuid (FK → Material, cascade delete)
  - QuantityPerUnit: decimal (not null)
  - SnapshottedUnitCost: decimal (nullable — null on recipe task_materials;
    set to the material's current unit cost when cloned onto a job at roll time)
  - Notes: text (nullable)
  - CreatedAt, UpdatedAt: timestamp
  - UNIQUE (TaskID, MaterialID)
```

When a recipe is rolled into a job, each recipe task's materials are cloned
onto the corresponding job tasks (including across batch expansion) with
`SnapshottedUnitCost` frozen to the material's current `UnitCost`, so the
job's material costs are stable even if the catalog price later changes.

---

#### Product & ProductVariant (new — quoting foundation)

Product is what the shop sells; a variant is a sellable configuration
(wood + finish) optionally bound to a recipe with variable bindings and a
customer-facing price.

```
Product
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string
  - Description: text (nullable)
  - IsActive: bool
  - CreatedAt, UpdatedAt: timestamp
  - DeletedAt: timestamp (nullable — GORM soft delete)

ProductVariant
  - ID: uuid
  - ProductID: uuid (FK → Product, cascade delete)
  - Name: string
  - RecipeID: uuid (FK → Recipe, nullable, set null on delete)
  - RecipeVariables: jsonb (nullable — variable bindings for the recipe)
  - Price: decimal (nullable — customer-facing price)
  - IsActive: bool
  - CreatedAt, UpdatedAt: timestamp
```

---

#### Quote & QuoteLine (new — quoting foundation)

A lightweight priced offer for a customer. No task tree is created until the
quote is accepted and rolled into a job. Lines freeze a cost snapshot from
the dry-run cost engine at quote time.

```
Quote
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - CustomerID: uuid (FK → Customer, nullable, set null on delete)
  - Status: enum (draft, sent, accepted, declined, cancelled)
  - Notes: text (nullable)
  - Markup: decimal (nullable)
  - OverrideTotal: decimal (nullable)
  - AcceptedAt: timestamp (nullable)
  - CreatedByID: uuid (FK → User)
  - CreatedAt, UpdatedAt: timestamp

QuoteLine
  - ID: uuid
  - QuoteID: uuid (FK → Quote, cascade delete)
  - ProductVariantID: uuid (FK → ProductVariant, nullable, set null on delete)
  - RecipeID: uuid (FK → Recipe, nullable — for non-product quotes)
  - Quantity: int (not null)
  - UnitPrice: decimal (nullable)
  - CostSnapshot: jsonb (nullable — frozen dry-run cost breakdown)
  - RecipeVariables: jsonb (nullable)
  - Notes: text (nullable)
  - CreatedAt, UpdatedAt: timestamp
```

---

#### Customer (see orders.md)

Simple customer record. Not a full CRM.

```
Customer
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string
  - Email: string (nullable)
  - Phone: string (nullable)
  - Address: text (nullable)
  - Notes: text (nullable)
  - CreatedAt, UpdatedAt: timestamp
```

---

#### Tag

Reusable labels for cross-cutting organization. Applied to tasks and recipes.

```
Tag
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string (unique within space)
  - Color: string (nullable)
  - CreatedAt: timestamp
```

Join tables:
- `task_tag` (TaskID string, TagID uuid)
- `recipe_tag` (RecipeID uuid, TagID uuid)

---

### Relationships Summary

```
Account 1:N Space
Space 1:N Station, Recipe, Customer, Material, Tag
User N:M Account (via UserAccount with role)
User N:M Space (via SpaceMember with role)

Task N:1 Space
Task N:1 Task (parent — unlimited nesting via hierarchical IDs)
Task N:1 Recipe, RecipeVersion (optional — source template for rolled jobs)
Task N:1 Station (optional — where this work happens)
Task N:1 Customer (optional — typically on root Job)
Task N:1 User (assigned)
Task 1:N TaskDep (as source or target)
Task 1:N TaskMedia
Task 1:N TaskComment
Task 1:N ActivityEntry
Task 1:N TimeEvent
Task 1:N CostEntry
Task N:M Tag (via task_tag)

TaskDep N:1 Task (source), Task (target)

Recipe N:1 Space
Recipe N:1 Recipe (extends — inheritance, deferred)
Recipe 1:N RecipeVersion
Recipe N:M Tag (via recipe_tag)

RecipeVersion N:1 Task (root_task_id — points to frozen recipe task tree)
RecipeVersion 1:N BOMItem (deprecated — moving to TaskMaterial linkage)

BOMItem N:1 Material (optional)

TimeEvent N:1 User, Task, Station (all optional except User)
CostEntry N:1 Task, Material, TimeEvent (optional links)
```

### ID Strategy

- **String (hierarchical)**: Task — beads-style `{space-slug}-{hex}.{ordinal}`
- **UUID**: All other new entities (Station, Recipe, Tag, Customer, Material,
  TaskDep, TaskMedia, TaskComment, ActivityEntry, TimeEvent, CostEntry, BOMItem)
- **Auto-increment int**: RecipeVersion (simple sequential versioning)
- **Existing**: User, Account, UserAccount, Space, SpaceMember keep their
  current ID types (UUID)

### Migration Strategy

Fresh start — no migration path from old Ticket/SOP tables. New numbered SQL
migration files in `server/migrations/` starting after the existing auth
migrations. Old Ticket/SOP model files and migrations will be deleted.

---

## Scenarios

These real-world scenarios exercise the new data model.

### Scenario 1: First-Time Build (Nakashima Coffee Table)

An operator builds a product for the first time with no established recipe.
They create a rough Job with ad-hoc tasks, adding notes and photos as they go.
After completing the job, the captured tasks are diffed and promoted into a new
Recipe.

**Exercises**: Task creation (ad-hoc, no recipe), TaskMedia capture,
TaskComment for notes, diff/promote to RecipeVersion, first-time capture mode.

### Scenario 2: Repeat Build from Recipe

An order for a Walnut Dining Table comes in. The manager "pours" the Dining
Table recipe, which expands into a Job with child tasks, dependencies, gates
(QC checks), and station assignments.

**Exercises**: Recipe → pour → Task subgraph, TaskDep (step ordering +
gates), Station assignment, ready-work algorithm, hierarchical IDs.

### Scenario 3: Batch Order (6 Chairs)

An order for 6 identical chairs. The manager pours the recipe 6 times (or
once with a loop variable `batch_size=6`). Each chair is a separate Job with
its own task subgraph. Cross-job dependencies can enforce "all assembly done
before any finishing starts."

**Exercises**: Recipe loops/variables, multiple Job creation, cross-job
TaskDep, batch flow management.

### Scenario 4: Interruptions and Ad-Hoc Work

Mid-task, the dust collector fills up. The operator pauses the current task,
creates a quick maintenance task, handles it, then resumes. The activity log
captures the interruption with a linked task.

**Exercises**: Task pause/resume, ad-hoc task creation, ActivityEntry
(interruption type), TimeEvent gap handling, cross-task time analysis.

### Scenario 5: Recipe Inheritance

The shop has a "Generic Dining Table" recipe. A "Live Edge Walnut Table"
recipe extends it, overriding the wood variable and adding an epoxy step.
Improvements to the generic recipe flow down to all derivatives.

**Exercises**: Recipe.ExtendsRecipeID, formula engine `extends`, variable
overrides, version propagation.

### Scenario 6: Gate / QC Hold

A task in the finishing process is a gate: "Inspect finish quality." It blocks
downstream "Ship" task. A manager must explicitly approve (mark the gate as
done) before shipping can proceed.

**Exercises**: Task type `gate`, TaskDep blocking, ready-work excluding
blocked tasks, approval workflow.

---

## What Was Deleted

The following entities from the original data model spec are **replaced** by
the Task + Recipe architecture:

| Old Entity | Replaced By |
|---|---|
| TicketType | Task.Type enum (fixed) |
| StatusDefinition | Task.Status enum (fixed) |
| Ticket | Task (with ParentID = null for Jobs) |
| TicketStep | Task (child of a Job Task) |
| TicketSubStep | Task (child of a child — unlimited nesting) |
| TicketLink | TaskDep |
| SOPTemplate | Recipe |
| SOPVersion / SOPTemplateVersion | RecipeVersion |
| SOPStep | Steps in TOML content → expanded to Tasks |
| SOPSubStep | Sub-steps in TOML → expanded to child Tasks |
| SOPStepMedia | TaskMedia (execution) + inline in TOML (template) |
| SOPCategory | Tags on Recipes (flat, not hierarchical folders) |
| SOPComment | TaskComment |

## Open Questions

- **Custom fields on tasks**: The Metadata jsonb field handles extensible data
  for now (wood species, dimensions, etc.). Structured custom field definitions
  may be added later.

- **Labor rate**: CostEntry computes labor cost, but needs a rate. Start with
  a Space-level default rate, add per-user rates later.

- **Notification/subscription model**: Who gets notified when a dependent task
  completes or a gate needs approval? Defer to later — the data model
  accommodates a future Subscription entity.

- **Recipe media storage**: Recipe TOML can reference media (photos in step
  instructions). Should these be stored as separate files with URL references
  in the TOML, or as a RecipeMedia table? Leaning toward file URLs in TOML
  + a media upload endpoint.

- **Task ID uniqueness scope**: Are hierarchical IDs unique globally or per-
  space? Per-space is simpler (space slug prefix handles it), but cross-space
  references need the full qualified ID.

# Recipe System: Technical Architecture

**Replaces**: `specs/recipes.md` (TOML-based recipe system)
**Related beads epic**: `nori-40n.7` — Recipe-as-Task-Tree, Roll Engine &
Quoting Foundation

## Core Principle: Recipes ARE Task Trees

Recipes and jobs share the same data model. A recipe is a task tree template;
a job is a task tree instance. Both live in the `task` table. The `Type` field
distinguishes them:

- `Type = 'recipe'` — root of a recipe task tree (template)
- `Type = 'job'` — root of a production job (live work)
- `Type = 'task'` — a step within either a recipe or a job
- `Type = 'milestone'` — a checkpoint marker
- `Type = 'gate'` — a hold point requiring approval

This unification means:
- The same UI component renders and edits both recipe steps and job tasks
- The same API endpoints manage task CRUD for both contexts
- Diff/promote between a job and its source recipe is a direct tree comparison
- "Save as recipe" is a tree clone with field stripping

## What Was Superseded

The original recipe system used TOML-as-code stored in a text column:

- `RecipeVersion.Content` held the full TOML source
- The `formula/` package (~4,400 LOC, 14 files) parsed TOML into in-memory
  `Formula` structs and expanded them into task trees during "pour"
- The formula engine was extracted from the beads issue tracker project

**Problems with the TOML approach:**
- Recipe data was trapped in a text blob — not queryable for cost aggregation,
  station analysis, or cross-recipe comparison
- The `Step` struct in formula/ and the `Task` model were parallel structures
  modeling the same thing, requiring translation during pour
- BOM items, station assignments, and gate metadata were defined in TOML but
  never persisted during pour (gaps in the pipeline)
- The formula engine carried ~3,000 lines of beads-inherited features
  (aspects, convoy type, external conditions, filesystem search paths) that
  don't apply to manufacturing
- UI editors needed a TOML parser on both frontend and backend

**What carries forward:**
- The batch expansion and dependency wiring logic (~180 lines) is ported to
  the new roll service
- The concept of variables, conditions, gates, batch sizing, and composition
  remain architecturally valid — they'll be implemented as relational features
  over time
- The `formula/` package is kept as reference code but is no longer the
  runtime path

## Schema Changes

### Task Table Additions

```sql
-- Migration 000042
ALTER TABLE task ADD COLUMN batch_size INT;
ALTER TABLE task ADD COLUMN estimated_time_secs INT;
```

New fields:

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `batch_size` | INT | yes | How many units this step covers. Used on recipe steps to control ticket creation during roll. NULL = 1 (per-piece). A value of 30 means "this step covers all 30 units in a single task." |
| `estimated_time_secs` | INT | yes | Expected duration for this step in seconds. Used on recipe steps for cost estimation and on job tasks for estimated-vs-actual comparison. |

The `Type` enum gains `'recipe'` as a valid value for root tasks of recipe
trees.

### RecipeVersion Table Changes

```sql
-- Migration 000043
ALTER TABLE recipe_version ADD COLUMN root_task_id VARCHAR(255)
    REFERENCES task(id);
ALTER TABLE recipe_version ALTER COLUMN content DROP NOT NULL;
CREATE INDEX idx_recipe_version_root_task_id ON recipe_version(root_task_id);
```

| Column | Change | Description |
|--------|--------|-------------|
| `root_task_id` | **Added** | FK to `task.id`. Points to the frozen task tree for this version. |
| `content` | **Made nullable** | The old TOML text column. Kept for backward compatibility but no longer written to. Will be dropped in a future migration. |

### Go Model Updates

```go
// Task model additions
type Task struct {
    // ... existing fields ...
    BatchSize        *int `gorm:"column:batch_size" json:"batchSize,omitempty"`
    EstimatedTimeSecs *int `gorm:"column:estimated_time_secs" json:"estimatedTimeSecs,omitempty"`
}

// RecipeVersion model changes
type RecipeVersion struct {
    // ... existing fields ...
    RootTaskID *string  `gorm:"column:root_task_id" json:"rootTaskId,omitempty"`
    Content    *string  `gorm:"column:content" json:"content,omitempty"` // deprecated
}
```

## Recipe Lifecycle

### Authoring

Recipe authoring uses the existing Task API. The recipe service is a thin
wrapper that:

1. Creates a `Recipe` record (identity: name, slug, space, created_by)
2. Creates a root `Task` with `Type = 'recipe'`
3. Creates a draft `RecipeVersion` with `RootTaskID` pointing to the root task

Editing the recipe's task tree uses the existing task endpoints:
- `POST /api/tasks/:parentId/children` — add a step
- `PUT /api/tasks/:id` — update a step (title, description, station,
  estimated_time, batch_size)
- `DELETE /api/tasks/:id` — remove a step
- `POST /api/task-deps` — add a dependency between steps
- `PUT /api/tasks/:id/reorder` — reorder steps

The recipe service enforces:
- Only draft versions can be edited
- Structural validation (no orphan dependencies, valid batch sizes > 0)

### Publishing (Clone-on-Publish)

Publishing freezes the draft into an immutable snapshot:

1. **Deep-clone** the draft recipe task tree:
   - Clone the root task (new ID, same fields, `Type = 'recipe'`)
   - Recursively clone all children with new hierarchical IDs
   - Clone all `TaskDep` edges between cloned tasks
   - Preserve station assignments, estimated times, batch sizes
2. Set `RecipeVersion.RootTaskID` to the cloned root's ID
3. Set `RecipeVersion.Status = 'published'`, set `PublishedAt`
4. Update `Recipe.CurrentVersionID` to this version
5. Archive the previously published version (`Status = 'archived'`)

The draft tree remains editable for future changes. The published tree is
frozen and serves as the source for rolling.

### DeepCloneTaskTree Utility

A shared utility used by three operations:

| Operation | Source | Target | Special handling |
|-----------|--------|--------|------------------|
| Publish | Draft recipe tree | Frozen recipe tree | Straight clone |
| Roll | Published recipe tree | Job tree | Type remapping, batch expansion, variable substitution |
| Save-as-recipe | Job tree | Draft recipe tree | Strip runtime fields, populate estimated times from actuals |

```go
// Signature sketch
func DeepCloneTaskTree(ctx context.Context, tx *gorm.DB, opts CloneOptions) (*Task, error)

type CloneOptions struct {
    SourceRootID    string
    NewRootID       string        // "" = auto-generate
    TargetType      TaskType      // recipe, job
    SpaceID         uuid.UUID
    CreatedByID     uuid.UUID
    StripRuntime    bool          // clear status, assignee, timestamps
    PopulateEstimates bool       // copy actual_time_secs → estimated_time_secs
    RecipeID        *uuid.UUID    // set on cloned tasks for traceability
    RecipeVersionID *int          // set on cloned tasks for traceability
}
```

## Roll Engine

"Roll" replaces "pour." It creates a production job from a published recipe.

### 1:1 Clone (No Batch Expansion)

When `order_qty = 1` (default):

1. Load the published `RecipeVersion` → get `RootTaskID`
2. Load the full task tree (root + all descendants + all `TaskDep` edges)
3. Deep-clone via `DeepCloneTaskTree`:
   - New root: `Type = 'job'`, new generated ID (`{space-slug}-{hex}`)
   - Children get hierarchical IDs (`{root}.1`, `{root}.1.1`, etc.)
   - `TaskDep` edges remapped to new task IDs
   - `RecipeID` and `RecipeVersionID` set on all cloned tasks
   - Station assignments, estimated times, batch sizes copied
   - Status set to `open`, timestamps cleared

### Batch Expansion (order_qty > 1)

When `order_qty > 1`, each recipe step is expanded based on its `batch_size`:

```
ticket_count = order_qty / batch_size
```

- **ticket_count = 1** (batch step): Creates 1 task. Example: "Cut all table
  tops" with `batch_size = 30` and `order_qty = 30` → 1 task.
- **ticket_count > 1** (per-piece step): Creates N tasks. Example: "Hand sand
  each top" with `batch_size = 1` and `order_qty = 30` → 30 tasks.

Task titles for multi-ticket steps include the piece number:
`"Hand sand table top (1/30)"`, `"Hand sand table top (2/30)"`.

### Dependency Wiring Patterns

Dependencies between expanded steps follow three patterns based on the
ticket counts of the source and target:

**1:1 (same ticket count):** Positional wiring.
```
upstream[0] → downstream[0]
upstream[1] → downstream[1]
...
upstream[N] → downstream[N]
```

**Fan-out (fewer upstream, more downstream):** Every downstream depends on
every upstream.
```
upstream[0] → downstream[0]
upstream[0] → downstream[1]
upstream[0] → downstream[2]
...
```
Example: 1 batch "resaw" task → 30 per-piece "lamination" tasks.

**Fan-in (more upstream, fewer downstream):** Every downstream depends on
every upstream.
```
upstream[0] → downstream[0]
upstream[1] → downstream[0]
upstream[2] → downstream[0]
...
```
Example: 30 per-piece "sanding" tasks → 1 batch "group finishing" task.

This logic is ported from the existing pour service
(`recipe.service.go:319-409`) and adapted to work with Task rows instead
of Formula Step structs.

### Roll API

```
POST /api/recipes/:id/roll

Body:
{
    "order_qty": 30,           // optional, default 1
    "customer_id": "uuid",     // optional
    "due_date": "2026-05-15",  // optional
    "title": "Custom title"    // optional, defaults to recipe name
}

Response: Job root task with full tree (same as GET /api/tasks/:id?tree=true)
```

## Save-as-Recipe

Clones a job's task tree into a new recipe:

1. Create a new `Recipe` record (user provides name, description)
2. Deep-clone the job tree via `DeepCloneTaskTree` with:
   - `TargetType = 'recipe'`
   - `StripRuntime = true` (clear status, assignee, started_at, completed_at)
   - `PopulateEstimates = true` (copy `actual_time_secs` →
     `estimated_time_secs` where estimated is null)
3. Create a draft `RecipeVersion` pointing to the cloned root
4. Return the new Recipe

## Cost Pipeline

### Time → Cost Flow

```
Task transition (start/pause/resume/complete)
    → TimeEvent created automatically (source = 'system')
        → On task completion: compute active seconds from TimeEvents
            → CostEntry created (type = 'labor', amount = hours × labor_rate)
```

### Labor Cost Computation

```
active_seconds = SUM of (check_out - check_in) intervals, excluding paused periods
labor_cost = (active_seconds / 3600) × space.default_labor_rate
```

### Job Cost Summary

```
GET /api/jobs/:id/cost-summary

Response:
{
    "job_id": "nori-a3f8",
    "estimated": {
        "labor_hours": 45.5,
        "labor_cost": 2275.00,
        "total": 2275.00
    },
    "actual": {
        "labor_hours": 52.3,
        "labor_cost": 2615.00,
        "total": 2615.00
    },
    "variance": {
        "labor_hours": 6.8,
        "labor_cost": 340.00,
        "total": 340.00,
        "percent": 14.9
    },
    "by_station": [
        {
            "station_id": "uuid",
            "station_name": "Finishing",
            "estimated_hours": 12.0,
            "actual_hours": 16.8,
            "variance_hours": 4.8,
            "variance_percent": 40.0
        }
    ]
}
```

**Estimated** comes from the source recipe's `estimated_time_secs` fields
multiplied by `Space.DefaultLaborRate`.

**Actual** comes from `CostEntry` records computed from `TimeEvent` data.

**Note:** Material costs will be added to this summary when the inventory
system is built. See `specs/inventory/spec.md`.

## Recipe API Surface

### Recipe CRUD

```
GET    /api/spaces/:spaceId/recipes              — List recipes
POST   /api/spaces/:spaceId/recipes              — Create recipe
GET    /api/recipes/:id                           — Get recipe with current version
PUT    /api/recipes/:id                           — Update recipe metadata
DELETE /api/recipes/:id                           — Archive recipe (soft delete)
```

### Version Management

```
GET    /api/recipes/:id/versions                  — List versions
POST   /api/recipes/:id/versions                  — Create new draft
POST   /api/recipe-versions/:versionId/publish    — Publish draft
```

### Roll

```
POST   /api/recipes/:id/roll                      — Roll recipe into job
```

### Cost

```
GET    /api/jobs/:id/cost-summary                 — Job cost summary
```

### Task Tree Editing (existing endpoints, used for both recipes and jobs)

```
GET    /api/tasks/:id?tree=true                   — Get task with full tree
POST   /api/tasks/:parentId/children              — Add child task
PUT    /api/tasks/:id                             — Update task
DELETE /api/tasks/:id                             — Delete task
PUT    /api/tasks/:id/reorder                     — Reorder among siblings
POST   /api/task-deps                             — Add dependency
DELETE /api/task-deps/:id                         — Remove dependency
```

## UI Architecture

### Shared Task-Tree Editor Component

A single Svelte 5 component that renders and edits a task tree. Used on:
- Recipe detail page (authoring context)
- Job detail page (execution context)

The component accepts a `context` prop (`'recipe'` | `'job'`) that controls
which fields are shown:

| Field | Recipe Context | Job Context |
|-------|---------------|-------------|
| Title | edit | edit |
| Description | edit | edit |
| Station | edit (dropdown) | read-only |
| Estimated time | edit | read-only |
| Batch size | edit | read-only |
| Status | hidden | shown |
| Assigned to | hidden | shown |
| Actual time | hidden | shown |
| Dependencies | edit | edit |

### Frontend Pages

| Route | Component | Description |
|-------|-----------|-------------|
| `/recipes` | RecipeListPage | Browse recipes, create new |
| `/recipes/:id` | RecipeDetailPage | View/edit recipe, version selector, publish button, roll button |
| `/jobs/:id` | JobDetailPage (existing) | Extended with cost summary panel |

### shadcn-svelte Components Used

- Table — recipe list
- Card — recipe/job detail panels
- Dialog — roll dialog, save-as-recipe dialog
- Button, Input, Select — form controls
- Badge — version status, cost variance indicators
- Tree view — custom, based on nested task structure

## Deferred Features

These features from the original TOML-based system are architecturally valid
but deferred for now:

| Feature | Why Deferred | When to Revisit |
|---------|-------------|-----------------|
| Variables / custom fields | Needs a proper custom field system, not string interpolation | After demo, when building quoting UI |
| Conditions | Depends on custom fields to condition on | With custom fields |
| Inheritance (extends) | Complex resolver needed for DB-stored recipes | When recipe library grows large |
| Aspects / advice | Cross-cutting step injection | When safety/compliance becomes a requirement |
| Composition / bond points | Sub-assembly wiring | When products have reusable sub-assemblies |
| Inline expansions | Embed sub-recipes | With composition |
| TOML import/export | Portability feature | When recipe marketplace is built |
| Range loops | Batch sizing covers the primary use case | If computed iteration counts are needed |
| Runtime conditions | Gate evaluation, until-loop termination | When building the execution engine |

## Related Specs

- `specs/data-model.md` — Full entity schema (updated with Task/RecipeVersion changes)
- `specs/inventory/spec.md` — Materials, BOM, inventory (placeholder)
- `specs/time-tracking.md` — TimeEvent model and sources
- `specs/stations.md` — Station model, WIP limits
- `specs/job-flow.md` — Dependency graph, ready queue, pull system
- `specs/task-execution.md` — Running tasks, gates, capture mode

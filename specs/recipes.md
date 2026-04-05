# Recipes

Replaces: `sop-authoring.md`, `sop-versioning.md`

## Who

- **Shop owners / managers**: Author and maintain recipes for products and
  processes.
- **Operators**: Contribute to recipes by promoting ad-hoc tasks captured during
  execution (see task-execution.md).
- **AI agents**: Draft recipes from voice notes, photos, or existing documents
  (see ai-features.md).

## What

Recipes are versioned TOML templates that describe how to build a product or
perform a process. When a recipe is "poured," the formula engine expands the
TOML into a task subgraph (a Job with child Tasks, dependencies, gates, and
station assignments).

Recipes replace the old SOPTemplate + SOPVersion + SOPStep hierarchy with a
single document-as-code approach, powered by a formula engine extracted from
the beads project.

## Where

- Backend: `server/internal/models/` (Recipe, RecipeVersion), `server/internal/formula/`
  (engine extracted from beads)
- Frontend: Recipe editor page, recipe browser, diff viewer
- Storage: PostgreSQL (RecipeVersion.Content column stores TOML text)
- Data model: see data-model.md

## Why

SOPs in the old design were rigid: linear steps, no variables, no composition,
no conditional logic. Real manufacturing processes need:

- **Variables** — "Walnut Dining Table" and "Cherry Dining Table" are the same
  recipe with different inputs.
- **Loops** — Batch production: repeat a step group N times.
- **Gates** — QC holds, cure timers, manager sign-offs that block downstream
  work.
- **Composition** — "Milling Lumber" is a reusable sub-recipe referenced by
  every furniture recipe.
- **Inheritance** — "Live Edge Table" extends "Generic Dining Table," overriding
  the finishing steps.

The formula engine from beads provides all of this in ~3,900 lines of
dependency-free Go code. By storing the TOML in the database (not the
filesystem), non-technical users can author and version recipes without
knowing what git is.

## How

### TOML Format

A recipe is a TOML document. Here's a real-world example:

```toml
[recipe]
name = "walnut-dining-table"
description = "Standard dining table build"
version = 1
type = "workflow"

[vars]
wood_species = "Walnut"
table_length = "72"
table_width = "36"
finish_type = { description = "Finish to apply", enum = ["oil", "lacquer", "poly"], default = "oil" }
batch_size = { description = "Number of units", type = "int", default = "1" }

[[steps]]
id = "mill"
title = "Mill lumber - {{wood_species}}"
station = "mill-station"
description = "Mill all blanks to rough dimension + 1/8\" oversize"

  [[steps.children]]
  id = "face-joint"
  title = "Face joint all boards"
  station = "jointer"

  [[steps.children]]
  id = "thickness"
  title = "Thickness plane to final dimension"
  station = "planer"
  needs = ["face-joint"]

  [[steps.children]]
  id = "rip"
  title = "Rip to width on table saw"
  station = "table-saw"
  needs = ["thickness"]

[[steps]]
id = "joinery"
title = "Cut joinery"
station = "joinery-bench"
needs = ["mill"]

  [[steps.children]]
  id = "mortises"
  title = "Cut mortises - {{wood_species}}"
  description = "Use 3/8\" mortise chisel. Set fence to 1/4\" from reference face."

  [[steps.children]]
  id = "tenons"
  title = "Cut tenons"
  needs = ["mortises"]
  description = "Sneak up on fit. Test with mortise piece."

  [[steps.children]]
  id = "dry-fit"
  title = "Dry fit assembly"
  needs = ["tenons"]

[[steps]]
id = "qc-joinery"
title = "QC: Inspect joinery fit"
needs = ["joinery"]
type = "gate"

  [steps.gate]
  type = "human"
  timeout = "24h"

[[steps]]
id = "glue-up"
title = "Glue up"
station = "assembly-bench"
needs = ["qc-joinery"]

  [steps.gate]
  type = "timer"
  id = "cure-time"
  timeout = "4h"

[[steps]]
id = "finish"
title = "Apply {{finish_type}} finish"
station = "finish-room"
needs = ["glue-up"]
condition = "{{finish_type}} != none"

[[steps]]
id = "final-inspect"
title = "Final inspection"
needs = ["finish"]
type = "gate"

  [steps.gate]
  type = "human"

[bom]
  [[bom.items]]
  name = "{{wood_species}} 4/4"
  quantity = 20
  unit = "board_feet"
  step = "mill"

  [[bom.items]]
  name = "Wood glue"
  quantity = 8
  unit = "oz"
  step = "glue-up"

  [[bom.items]]
  name = "{{finish_type}} finish"
  quantity = 1
  unit = "quart"
  step = "finish"
```

### Key TOML Elements

**Recipe header** (`[recipe]`):
- `name` — Unique identifier within the space
- `description` — Human-readable summary
- `version` — Schema version (currently 1)
- `type` — `workflow` (standard), `expansion` (macro), `aspect` (cross-cutting)
- `extends` — Parent recipe name(s) for inheritance

**Variables** (`[vars]`):
- Simple: `wood_species = "Walnut"` (string with default)
- Complex: `{ description, default, required, enum, pattern, type }`
- Referenced in templates as `{{var_name}}`
- Resolved at pour time — the operator provides values or accepts defaults

**Steps** (`[[steps]]`):
- `id` — Unique within the recipe (becomes part of the task's hierarchical ID)
- `title` — Supports `{{variable}}` substitution
- `description` — Detailed instructions (markdown)
- `station` — Station slug where this work happens
- `needs` — List of sibling step IDs that must complete first (dependencies)
- `type` — `task` (default), `gate` (approval hold), `milestone` (checkpoint)
- `condition` — Optional expression: `"{{var}}"`, `"!{{var}}"`,
  `"{{var}} == value"`, `"{{var}} != value"`. Step is skipped if false.
- `children` — Nested sub-steps (unlimited depth, maps to child Tasks)
- `gate` — Gate configuration: `{ type, id, timeout }`

**Gates** (`[steps.gate]`):
- `human` — Requires manual approval (manager clicks "approve")
- `timer` — Blocks for a duration (cure time, drying time)
- Gate creates a blocking TaskDep. Downstream tasks can't start until the gate
  is resolved.

**Loops** (`[steps.loop]`):
- `count` — Fixed iterations (e.g., batch of 6 chairs)
- Variable-driven: `count = "{{batch_size}}"`
- Each iteration gets its own task subtree with `{i}` substitution

**BOM** (`[bom]`):
- `items` — List of materials needed
- Each item: `{ name, quantity, unit, step, unit_cost, notes }`
- `step` — Which recipe step uses this material
- Variables work in BOM too: `"{{wood_species}} 4/4"`

### Inheritance (`extends`)

A recipe can extend one or more parent recipes:

```toml
[recipe]
name = "live-edge-walnut-table"
extends = ["walnut-dining-table"]

[vars]
wood_species = "Live Edge Walnut"
finish_type = "epoxy-then-oil"

# Override the finish step
[[steps]]
id = "finish"
title = "Apply epoxy fill, then oil finish"
station = "finish-room"
needs = ["glue-up"]

  [[steps.children]]
  id = "epoxy"
  title = "Fill voids with epoxy"

  [[steps.children]]
  id = "oil"
  title = "Apply oil finish"
  needs = ["epoxy"]
```

Child steps with the same `id` override parent steps. New steps are appended.
Variables in the child override parent defaults.

### Aspects (Cross-Cutting Concerns)

Safety checks, cleanup procedures, or documentation steps that apply across
multiple recipes:

```toml
[recipe]
name = "safety-checks"
type = "aspect"

[[advice]]
match = "station == table-saw"
position = "before"

  [[advice.steps]]
  id = "safety-check"
  title = "Verify blade guard and splitter installed"
  type = "gate"

  [advice.steps.gate]
  type = "human"
```

Aspects are applied during pour — every step that matches the pointcut gets
the advice steps inserted before/after.

### Composition (BondPoints)

Recipes can declare connection points for sub-assembly composition:

```toml
[compose]
  [compose.bond_points]
  top-ready = { after = "glue-up", description = "Table top is assembled and cured" }
  base-ready = { after = "joinery", description = "Base frame is joinery-complete" }
```

A parent recipe can then compose sub-assemblies:

```toml
[[steps]]
id = "final-assembly"
title = "Attach top to base"
needs = ["top.top-ready", "base.base-ready"]
```

### Pouring a Recipe

"Pouring" is the act of instantiating a recipe into a task subgraph:

1. **Resolve variables** — User provides values or accepts defaults.
2. **Apply inheritance** — Merge parent recipe(s) with overrides.
3. **Apply aspects** — Insert advice steps at matching pointcuts.
4. **Evaluate conditions** — Filter out steps where condition is false.
5. **Expand loops** — Unroll loop bodies into concrete iterations.
6. **Resolve compositions** — Wire up BondPoint dependencies.
7. **Create tasks** — Each step becomes a Task with hierarchical ID. Dependencies
   become TaskDep rows. Station references are resolved to Station IDs.
8. **Create BOM** — BOMItem rows linked to the RecipeVersion.

The root Task is the Job. Its ID is generated as `{space-slug}-{hex}`.
Children get `{job-id}.{ordinal}`.

### Authoring Interfaces

**Web UI** (primary for non-technical users):
- Visual step editor with drag-and-drop ordering
- Variable definition panel
- Station picker (dropdown from Space's stations)
- Gate configuration UI
- BOM editor with material search
- Live preview: "this is what the task graph will look like"
- TOML source view toggle for power users

**CLI** (for power users and AI agents):
- `nori recipe create --from-toml <file>` — Import TOML
- `nori recipe edit <name>` — Open in $EDITOR
- `nori recipe pour <name> --var wood_species=Cherry` — Pour with variables
- `nori recipe show <name>` — Display current version
- `nori recipe diff <name> v1 v2` — Compare versions

**AI-assisted** (see ai-features.md):
- Voice-to-recipe: Describe the process, AI drafts TOML
- Photo-to-step: Take a photo, AI generates a step description
- Post-execution: AI analyzes captured tasks and drafts a recipe

### Versioning

**Version lifecycle**:

```
[Draft] → [Published] → [Archived]
               ↓
          [New Draft] → [Published] → ...
```

- **Draft**: Work in progress. One active draft per recipe. Only visible to
  the author and managers. Edits to a draft are saved in-place (not one version
  per keystroke).
- **Published**: The active version. Pouring uses this version. New jobs
  snapshot the RecipeVersion ID.
- **Archived**: Previous published versions. Read-only.

**Version numbers**: Simple incrementing integers (v1, v2, v3). Each publish
bumps by 1.

**Publishing**:
1. Draft status → `published`
2. `Recipe.CurrentVersionID` updated
3. Previous published version → `archived`
4. In-flight jobs are NOT affected — they reference their snapshot version
5. Optional prompt: "3 jobs use v2. Update them to v3?" (per-job opt-in)

**Change summary**: Each version has an optional `ChangeSummary`. AI can
auto-generate this by diffing the TOML between versions.

### Diff / Promote Flow

When an operator executes a job and adds ad-hoc tasks, modifies instructions,
or skips steps, their execution diverges from the source recipe. After the job
completes:

1. **Diff**: Compare the job's actual task graph against the source
   RecipeVersion's expected graph. Show: added tasks, removed/skipped steps,
   modified descriptions, timing data.
2. **Promote**: The operator (or manager) selects which changes to fold back
   into a new RecipeVersion draft.
3. **Review**: The draft can be edited further before publishing.

This is how recipes improve over time without requiring anyone to "write
documentation" — the documentation writes itself from real work.

### Recipe Browser

- List all recipes in the current Space
- Filter by tag
- Show current version status (draft, published)
- Quick stats: times poured, average completion time, last poured date
- Search by name, description, variable values

### API Surface

```
GET    /api/spaces/:spaceId/recipes                      — List recipes
POST   /api/spaces/:spaceId/recipes                      — Create recipe
GET    /api/recipes/:id                                  — Get recipe with current version
PUT    /api/recipes/:id                                  — Update recipe metadata
DELETE /api/recipes/:id                                  — Archive recipe

GET    /api/recipes/:id/versions                         — List versions
POST   /api/recipes/:id/versions                         — Create new draft
GET    /api/recipes/:id/versions/:versionId              — Get version with TOML content
PUT    /api/recipe-versions/:versionId                   — Update draft content
POST   /api/recipe-versions/:versionId/publish           — Publish draft
DELETE /api/recipe-versions/:versionId                   — Discard draft

POST   /api/recipes/:id/pour                             — Pour recipe → create Job
GET    /api/recipes/:id/diff/:v1/:v2                     — Diff two versions

POST   /api/tasks/:taskId/promote                        — Promote ad-hoc execution to recipe draft
```

## Open Questions

- Should recipe TOML support inline media references (photo URLs in step
  descriptions), or should media be managed separately via a RecipeMedia table?
  Leaning toward URLs in markdown descriptions + media upload endpoint.

- Should there be a "recipe marketplace" where shops share recipes? (See
  sop-marketplace.md — needs renaming to recipe-marketplace.md.) The TOML
  format makes import/export trivial.

- Should the web editor support collaborative editing (two people editing the
  same draft)? Probably not for v1 — single-writer with last-write-wins.

- How should the formula engine handle errors in TOML (bad variable references,
  circular dependencies)? Surface errors in the editor with line numbers +
  descriptions, and refuse to pour invalid recipes.

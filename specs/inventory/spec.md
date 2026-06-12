# Feature Specification: Inventory & Materials

**Created**: 2026-04-23
**Status**: Draft (Placeholder)
**Labels**: inventory, materials, bom, erp

## Overview

Inventory and materials management connects the physical world of a shop to
the digital task system. Materials are consumed by tasks during production.
Recipes declare what materials are needed and in what quantities. When a recipe
is rolled into a job, the system should be able to compute material
requirements, check stock levels, and eventually trigger purchasing when
inventory runs low.

This spec captures the direction and context from early architectural
discussions. Most of this is future work — the immediate priority is the
recipe system and cost tracking foundation (see `specs/recipes/`).

**This spec is a placeholder.** It establishes the direction and captures
key design context. Full user stories and functional requirements will be
developed when this becomes active work.

## Key Concepts

### Material Catalog

A shop maintains a catalog of materials they stock. Materials have:

- Name, description, category (wood, hardware, finish, consumable, etc.)
- Unit of measure (board_feet, gallons, each, linear_feet, etc.)
- Unit cost (current cost per unit — used for cost estimation)
- Stock quantity (how much is on hand)
- Reorder threshold (minimum stock before alert/auto-purchase)
- Supplier information (future)

Materials are space-scoped. The existing `Material` model in the data model
spec covers most of this.

### Task-Material Linkage

A task (whether a recipe step or a job task) can declare which materials it
consumes and in what quantity. This is the bridge between work and inventory.

**Proposed model:** A link table connecting tasks to materials with quantity
and unit information.

```
TaskMaterial (link table)
  - ID: uuid
  - TaskID: string (FK → Task)
  - MaterialID: uuid (FK → Material)
  - Quantity: decimal (amount consumed per unit of work)
  - Unit: string (should match material's unit)
  - Notes: text (nullable)
```

When a recipe step declares "this step uses 1 gal of sealer per table", that's
a `TaskMaterial` row on the recipe step. When rolled into a job with qty 30,
the job-level requirement becomes 30 gallons.

### BOM per Recipe

A recipe's Bill of Materials is the aggregate of all `TaskMaterial` rows across
its steps. This is computed, not stored separately — the task-material linkage
IS the BOM.

```
Recipe BOM = SUM of (task_material.quantity × ticket_count) for each material
    across all steps in the recipe's task tree
```

For a batch of 30 tables:
- Per-piece step "apply sealer" with 1 gal/table → 30 gal
- Batch step "mix finish" with 0.5 gal → 0.5 gal (not scaled)

### Material Cost in Job Cost Summary

The job cost summary (see `specs/recipes/architecture.md`) will be extended
to include material costs:

```
total_job_cost = labor_cost + material_cost
material_cost = SUM of (task_material.quantity × material.unit_cost)
    across all tasks in the job
```

This allows the estimated job cost to include both labor and materials,
giving a more complete picture for quoting.

## Real-World Example: Finish Materials

From a conversation about a mid-size table manufacturer:

A shop might have several finish options, each with different materials:

1. **Renner 321 solvent sealer + 688 top coat** — 1 gal sealer + 0.5 gal
   top coat per table
2. **Renner 673 clear sealer + 688 top coat** — 1 gal sealer + 0.5 gal
   top coat per table
3. **Renner 718 self-sealing top coat** — 1 gal per table (no separate sealer)

For an order of 30 tables with option 1:
- 30 gal of Renner 321 solvent sealer
- 15 gal of Renner 688 top coat

This material requirement should:
- Be computed automatically from the recipe + order quantity
- Trigger a pull from inventory
- Trigger a purchase order if stock is below threshold

**This example illustrates why custom fields and material selection are deeply
intertwined.** The finish choice affects which materials are consumed, which
affects cost, which affects quoting. This is ERP territory and needs careful
design.

## Relationship to Custom Fields

Materials selection often depends on product configuration choices (wood
species, finish type, hardware selection). These choices are conceptually
"custom fields" on the job or recipe — similar to Jira's configurable ticket
fields.

A custom field system would allow:
- Defining field types (dropdown, text, number) per recipe
- Fields like "Sealer" with options: Renner 321, Renner 673, None
- Fields like "Top Coat" with options: Renner 688, Renner 718
- Material quantities that vary based on field selections
- Conditional task inclusion (skip sealer step if using self-sealing topcoat)

**Custom fields are out of scope for the immediate recipe work.** The recipe
system provides the structural foundation (task trees, batch expansion,
dependencies) that custom fields will extend later.

## Deferred Scope

The following are identified as future work:

| Feature | Notes |
|---------|-------|
| **Custom fields system** | Per-recipe configurable fields with typed options. Deeply affects material selection and conditional steps. |
| **Purchasing pipeline** | Auto-generate POs when stock drops below threshold. Requires supplier management. |
| **Supplier management** | Supplier catalog, preferred suppliers per material, lead times, pricing. |
| **Receiving workflow** | Record material receipts, update stock quantities, inspect incoming materials. |
| **Material waste tracking** | Track actual consumption vs planned consumption. Expected waste percentages. |
| **Material lot tracking** | Track which lot/batch of material went into which job. Important for quality traceability. |
| **Multi-location inventory** | Track stock across multiple storage locations within the shop. |

## Existing Data Model

The `Material` model already exists in the database:

```
Material
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string
  - Description: text (nullable)
  - Category: enum (wood, hardware, finish, adhesive, abrasive, other)
  - Unit: string (board_feet, each, gallon, oz, linear_feet, sheet, bag)
  - UnitCost: decimal (nullable)
  - StockQuantity: decimal (default 0)
  - ReorderThreshold: decimal (nullable)
  - IsActive: bool (default true)
  - CreatedAt, UpdatedAt: timestamp
```

The `BOMItem` model also exists but was designed for the TOML-based recipe
system (linked to `RecipeVersion` rather than to individual tasks). The
`TaskMaterial` link table approach described above is a better fit for the
task-tree model.

The `CostEntry` model already supports material costs:
```
CostEntry.CostType = 'material'
CostEntry.MaterialID = FK → Material
```

## Related Specs

- `specs/recipes/spec.md` — Recipe system user spec
- `specs/recipes/architecture.md` — Recipe technical architecture
- `specs/materials-and-bom.md` — Original materials spec (may be superseded)
- `specs/data-model.md` — Full entity schema

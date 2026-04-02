# Materials and Bill of Materials

## Who

- **Shop owners / managers**: Define materials, set stock levels, manage
  replenishment.
- **Operators**: See what materials a job requires, trigger pull signals
  when stock is low.

## What

Structured material tracking and bill-of-materials (BOM) per SOP. Materials
are stock items in the shop with quantities, locations, and replenishment
thresholds. Each SOP version has a BOM listing what materials and how much
are needed. When stock drops below threshold during job execution, Nori
auto-creates replenishment tasks.

## Where

- Backend: Material and BOMItem models, material API endpoints
- Frontend: Material inventory page, BOM section in SOP editor, pull signal
  notifications
- Data model: see data-model.md

## Why

The original requirements doc described this with the sushi metaphor — each
ingredient has a location, a required amount, and a "pull" button that fires
when stock is low. This is straight Lean manufacturing: materials are pulled
through the system based on actual consumption, not pushed based on forecasts.

For a woodworking shop, this means:
- You never run out of Titebond III mid-glue-up because the system told you
  to reorder last week.
- You know exactly how many board feet of walnut you need before starting a
  job, and whether you have enough in stock.
- Replenishment tasks appear in the job flow board alongside production work,
  so they don't get forgotten.

The existing model stores materials as a `pq.StringArray` (plain text list).
This needs to become structured data for pull signals and stock tracking to
work.

## How

### Material Types for Woodworking

| Category | Unit | Threshold Type | Examples |
|----------|------|----------------|----------|
| Lumber | board_feet | quantity-based | White Oak 4/4, Walnut 8/4 |
| Sheet goods | sheets / sq_ft | quantity-based | Baltic Birch 3/4, MDF |
| Hardware | count | quantity-based | #8 x 1.25 screws, drawer slides |
| Finish | oz / gallons | quantity-based | Arm-R-Seal, General Finishes |
| Adhesive | oz / gallons | quantity-based | Titebond III, epoxy |
| Consumable | trigger-based | event-based | Sandpaper, saw blades |

**Trigger-based** consumables don't have a meaningful "quantity" — you don't
count individual sheets of sandpaper. Instead, the pull signal fires on a
schedule or after N uses (e.g., "change saw blade every 50 cuts").

### Material Model

```
Material
  - ID: uuid
  - SpaceID: uuid
  - Name: string ("White Oak 4/4")
  - Category: enum (lumber, sheet_goods, hardware, finish, adhesive, consumable, other)
  - Unit: string (board_feet, count, oz, gallons, sheets, each)
  - CurrentStock: decimal
  - ReorderThreshold: decimal (pull signal when stock <= this)
  - ReorderQuantity: decimal (suggested reorder amount)
  - Location: string (nullable — "Rack 3, Bay 2")
  - Supplier: string (nullable)
  - UnitCost: decimal (nullable — for cost estimation)
  - IsActive: bool
  - CreatedAt, UpdatedAt: timestamp
```

### BOMItem Model

Links a material to an SOP version, optionally to a specific step.

```
BOMItem
  - ID: int
  - SOPTemplateVersionID: int
  - MaterialID: uuid (nullable — for ad-hoc items not in inventory)
  - Name: string (display name)
  - Quantity: decimal
  - Unit: string
  - SOPStepID: int (nullable — which step uses this material)
  - Notes: string (nullable — "cut to 36 inches")
```

When MaterialID is set, stock tracking and pull signals are active. When it's
null, the BOM item is informational only (useful when first authoring an SOP
before setting up inventory).

### Pull Signals

When a job starts (or when a specific step that consumes material begins):
1. System checks: does the BOMItem have a linked Material?
2. If yes, check: Material.CurrentStock - BOMItem.Quantity <= Material.ReorderThreshold?
3. If yes, create a **Replenishment Job** (see job-flow.md) tagged as
   `restock` with the details.
4. Replenishment jobs appear on the flow board and can also trigger
   notifications.

For trigger-based consumables (no stock quantity), the pull signal fires:
- On a time schedule (e.g., "change fryer oil weekly" → "change saw blade
  monthly")
- After N job completions using that consumable
- Manually by the operator ("I'm running low — pull")

### Stock Adjustment

Stock is adjusted in two ways:
1. **Automatic**: When a job step that consumes a material is completed,
   deduct the BOM quantity from CurrentStock.
2. **Manual**: Operators or managers can adjust stock directly (receiving
   a shipment, inventory correction).

Stock adjustments are logged as events for audit trail.

### BOM in the SOP Editor

The SOP authoring UI (see sop-authoring.md) has a Materials section:
- Add materials from inventory (autocomplete search) or as free text
- Set quantity and unit per material
- Optionally link to a specific step
- Show current stock level inline ("12 board feet in stock")
- Visual warning if current stock is insufficient for the BOM

### Cost Estimation

If materials have UnitCost set, Nori can compute:
- Material cost per job (sum of BOMItem.Quantity * Material.UnitCost)
- Material cost per order (sum across all jobs)
- This is not invoicing — just internal cost tracking for margin analysis.

### API Surface

```
GET    /api/spaces/:spaceId/materials              — List materials
POST   /api/spaces/:spaceId/materials              — Create material
PUT    /api/materials/:id                          — Update material
POST   /api/materials/:id/adjust                   — Adjust stock (manual)

GET    /api/sop-versions/:versionId/bom            — Get BOM for a version
POST   /api/sop-versions/:versionId/bom            — Add BOM item
PUT    /api/bom-items/:id                          — Update BOM item
DELETE /api/bom-items/:id                          — Remove BOM item
```

## Open Questions

- How granular should lumber tracking be? Track by species + thickness
  (e.g., "White Oak 4/4") or by individual boards? (Leaning toward
  species + thickness for v1.)
- Should material deduction be automatic on step completion, or require
  operator confirmation? (Auto is lower friction but may be inaccurate
  if waste varies.)
- Should Nori track material waste? (e.g., BOM says 10 board feet, but
  you used 12 due to defects.) This is valuable for cost accuracy but
  adds friction.
- Is supplier management (beyond a text field) in scope for v1?

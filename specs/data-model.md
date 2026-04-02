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
right means the job flow board, SOP execution, time tracking, and analytics
all have clean, consistent data to work with. Getting it wrong means fighting
the schema for the lifetime of the project.

The model is designed around two core principles from Theory of Constraints:
1. **Work flows through stations** — not through abstract project phases.
2. **Time data is first-class** — every action that consumes time is recorded
   so bottlenecks surface naturally.

## How

### Entity Overview

```
Account
  └── Space (multi-tenant workspace)
        ├── User (via membership, with roles)
        ├── Station (physical shop locations)
        ├── SOPTemplate
        │     └── SOPTemplateVersion (draft/published)
        │           ├── SOPStep (ordered procedure steps)
        │           │     └── SOPStepMedia (photos, videos)
        │           └── BOMItem (materials + equipment)
        ├── Customer
        │     └── Order
        │           └── OrderLineItem
        │                 └── Job (one per line item)
        │                       └── JobStep (live execution of SOP steps)
        ├── Material (stock items with thresholds)
        └── TimeEvent (source-agnostic time log)
```

### Key Entities

#### Account & Space (existing, see auth-and-tenancy.md)

Carried forward from the existing codebase. An Account is a billing entity.
Spaces are workspaces within an Account (e.g., "Main Shop", "Finishing Room").

#### User (existing, see auth-and-tenancy.md)

Carried forward. Users belong to Accounts via UserAccount with roles. Users
operate within Spaces.

#### Station (new, see stations.md)

A physical location or tool in the shop where work happens.

```
Station
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string ("Mill", "Joinery", "Assembly", "Finish")
  - DisplayOrder: int (position on the flow board)
  - WIPLimit: int (max concurrent jobs at this station)
  - BufferSize: int (queue capacity before the station)
  - IsActive: bool
  - CreatedAt, UpdatedAt: timestamp
```

#### SOPTemplate → SOPTemplateVersion → SOPStep (existing, enhanced)

Carried forward from existing models. Key enhancements:
- SOPStep gains structured materials (via BOMItem) instead of string arrays
- SOPStep gains actual execution tracking via JobStep
- See sop-authoring.md and sop-versioning.md for details

#### BOMItem (new, see materials-and-bom.md)

A line item on a bill of materials, tied to an SOP version.

```
BOMItem
  - ID: int
  - SOPTemplateVersionID: int (FK → SOPTemplateVersion)
  - MaterialID: uuid (FK → Material, nullable for ad-hoc items)
  - Name: string (display name, e.g., "White Oak 4/4")
  - Quantity: decimal
  - Unit: string (board_feet, count, oz, gallons, each)
  - SOPStepID: int (FK → SOPStep, nullable — which step uses this)
  - Notes: string (nullable)
```

#### Material (new, see materials-and-bom.md)

A stock item tracked in inventory with replenishment thresholds.

```
Material
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string ("White Oak 4/4", "Titebond III", "#8 x 1.25 screws")
  - Category: enum (lumber, hardware, finish, consumable, other)
  - Unit: string (board_feet, count, oz, gallons, each)
  - CurrentStock: decimal
  - ReorderThreshold: decimal (pull signal fires when stock <= this)
  - ReorderQuantity: decimal (suggested order amount)
  - Location: string (nullable — "Rack 3, Bay 2")
  - IsActive: bool
  - CreatedAt, UpdatedAt: timestamp
```

#### Customer (new, see orders.md)

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

#### Order (new, see orders.md)

```
Order
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - CustomerID: uuid (FK → Customer)
  - OrderNumber: string (human-readable, e.g., "ORD-2026-042")
  - Status: enum (quoted, confirmed, in_progress, completed, cancelled)
  - QuotedAt: timestamp (nullable)
  - ConfirmedAt: timestamp (nullable)
  - DueDate: timestamp (nullable)
  - CompletedAt: timestamp (nullable)
  - Notes: text (nullable)
  - CreatedAt, UpdatedAt: timestamp
```

#### OrderLineItem (new, see orders.md)

```
OrderLineItem
  - ID: uuid
  - OrderID: uuid (FK → Order)
  - SOPTemplateID: int (FK → SOPTemplate — what product type)
  - Description: string ("Walnut Dining Table 72x36")
  - Quantity: int
  - UnitPrice: decimal (nullable)
  - Notes: text (nullable)
```

#### Job (new, see job-flow.md)

A single unit of work flowing through the shop. One Job per OrderLineItem
(or per quantity unit — configurable).

```
Job
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - OrderLineItemID: uuid (FK → OrderLineItem, nullable for internal jobs)
  - SOPTemplateVersionID: int (FK → SOPTemplateVersion — snapshot at creation)
  - CurrentStationID: uuid (FK → Station, nullable)
  - Status: enum (queued, in_progress, blocked, completed, cancelled)
  - Priority: int (lower = higher priority)
  - AssignedToID: uuid (FK → User, nullable)
  - StartedAt: timestamp (nullable)
  - CompletedAt: timestamp (nullable)
  - Notes: text (nullable)
  - CreatedAt, UpdatedAt: timestamp
```

#### JobStep (new, see sop-execution.md)

A live execution record of an SOP step within a Job.

```
JobStep
  - ID: uuid
  - JobID: uuid (FK → Job)
  - SOPStepID: int (FK → SOPStep — the template step)
  - StationID: uuid (FK → Station)
  - Status: enum (pending, in_progress, paused, completed, skipped)
  - AssignedToID: uuid (FK → User, nullable)
  - StartedAt: timestamp (nullable)
  - PausedAt: timestamp (nullable)
  - CompletedAt: timestamp (nullable)
  - ActualTimeSeconds: int (accumulated active time, excludes pauses)
  - DeviationNotes: text (nullable — "what I did differently")
  - CreatedAt, UpdatedAt: timestamp
```

#### TimeEvent (new, see time-tracking.md)

A source-agnostic time log entry. Can come from manual input, CLI check-in,
tap-in on a tablet, or future camera/sensor detection.

```
TimeEvent
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - UserID: uuid (FK → User)
  - JobID: uuid (FK → Job, nullable)
  - JobStepID: uuid (FK → JobStep, nullable)
  - StationID: uuid (FK → Station, nullable)
  - EventType: enum (check_in, check_out, pause, resume)
  - Source: enum (manual, cli, tap, sensor, api)
  - Timestamp: timestamp
  - Notes: text (nullable)
  - CreatedAt: timestamp
```

### Relationships Summary

- Account 1:N Space
- Space 1:N Station, SOPTemplate, Customer, Material
- User N:M Account (via UserAccount with role)
- Customer 1:N Order
- Order 1:N OrderLineItem
- OrderLineItem 1:1 Job (or 1:N if quantity > 1 spawns multiple)
- SOPTemplate 1:N SOPTemplateVersion
- SOPTemplateVersion 1:N SOPStep, 1:N BOMItem
- SOPStep 1:N SOPStepMedia
- Job 1:N JobStep
- JobStep → SOPStep (template reference)
- TimeEvent → User, Job, JobStep, Station (all optional except User)

### Prior Art

The existing codebase (`server/internal/models/`) has working implementations
of: User, Account, UserAccount, Space, SOPTemplate, SOPTemplateVersion,
SOPStep, SOPStepMedia. These should be carried forward and enhanced, not
rewritten from scratch.

Key changes from existing models:
- SOPTemplateVersion.Materials changes from `pq.StringArray` to structured
  BOMItem records
- SOPTemplateVersion.Equipment follows the same pattern
- SOPStep gains execution tracking via JobStep
- All new entities (Station, Customer, Order, Job, JobStep, TimeEvent,
  Material, BOMItem) are net-new

### Migration Strategy

Use numbered SQL migration files in `server/migrations/`. Each new entity
gets its own migration. Run migrations on startup via GORM AutoMigrate or
a dedicated migration tool (golang-migrate).

## Open Questions

- Should Job be 1:1 with OrderLineItem, or should a line item with quantity=4
  spawn 4 separate Jobs? (Leaning toward 1:1 with a quantity field on Job,
  but for TOC flow visibility, individual jobs per unit may be better.)
- Do we need a separate `Tag` entity for filtering jobs (prep, order, 3S,
  maintenance) as described in the original requirements? Or are these just
  string labels on Jobs?
- Should TimeEvent be an append-only event log (event sourcing style) or a
  mutable record? Append-only is better for analytics but harder to correct
  mistakes.

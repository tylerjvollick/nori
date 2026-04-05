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
right means configurable workflows, SOP execution, time tracking, and analytics
all have clean, consistent data to work with.

The model is designed around these principles:

1. **Configurable workflows** — Spaces define their own ticket types and status
   workflows (like Jira issue types). Manufacturing is a template, not
   hardcoded. Sales, admin, and other workflows use the same primitives.
2. **Work flows through stations** — Physical stations are first-class for
   shop spaces. SOP steps reference the station where work happens. Non-physical
   spaces (sales, admin) simply don't use stations.
3. **Time data is first-class** — Every action that consumes time is recorded
   so bottlenecks surface naturally.
4. **SOPs and tickets are tightly integrated but separate** — SOPs document
   the source of truth for *how* to do something. Tickets track *what work* is
   being done. The goal is zero friction between the two — an operator should
   never have to switch apps to see their SOP while working a ticket.
5. **AI-ready, not AI-dependent** — The structured model supports both CRUD
   (web UI, iPad) and conversational interaction (via LLM/MCP). AI is an
   interface layer on top, not baked in.

## How

### Design Decisions

These decisions were made through scenario analysis (documented in conversation
with the project owner). See the "Scenarios" section at the end of this spec
for the real-world examples that drove each decision.

- **Configurable ticket types** instead of fixed Order/Job entities. The
  manufacturing model becomes a Space template.
- **Sub-steps** (one level deep) for documentation detail. Steps are timed,
  sub-steps are not.
- **SOP composition** — steps can link to other SOPs, rendered inline during
  execution (side drawer, like TurboTax help links).
- **Live SOP editing with auto-versioning** — every save creates a version.
  No mandatory draft/publish gate. Comments and suggested edits on SOPs and
  individual steps.
- **Dual board views** — status-based kanban (all spaces) + station utilization
  view (shop spaces).
- **Separate Spaces** for different workflows (Sales, Shop) with cross-space
  ticket linking.
- **Parent/child tickets** (one level) for batch/order → individual items.
- **Activity log** on tickets for full chronological story (step transitions,
  interruptions, comments, linked work).
- **Hierarchical SOP categories** (folders) + tags for discoverability.
- **Full cost tracking** (labor + materials). Overhead allocation deferred.

### Entity Overview

```
Account
  └── Space
        ├── User (via membership, with roles)
        ├── Station (physical shop locations — optional, for shop spaces)
        │
        ├── TicketType (configurable per space)
        │     └── StatusDefinition[] (ordered statuses for this type)
        │
        ├── Ticket (universal work item — replaces Order + Job)
        │     ├── ParentTicketID (optional, one level nesting)
        │     ├── TicketTypeID → StatusID (current status)
        │     ├── SOPTemplateID (optional — linked SOP)
        │     ├── CustomerID (optional)
        │     ├── TicketStep[] (execution steps, from SOP or ad-hoc)
        │     │     └── TicketSubStep[] (detail checklist, not timed)
        │     ├── ActivityEntry[] (chronological log)
        │     ├── TimeEvent[] (time tracking)
        │     └── CostEntry[] (cost tracking)
        │
        ├── TicketLink (cross-ticket relationships, including cross-space)
        │
        ├── SOPCategory (hierarchical folders)
        │     └── SOPTemplate
        │           ├── Tags[]
        │           └── SOPVersion (auto-versioned on save)
        │                 ├── SOPStep[] (ordered, optional StationID)
        │                 │     ├── SOPSubStep[] (detail, one level)
        │                 │     └── SOPStepMedia[] (photos, videos)
        │                 └── BOMItem[] (materials + equipment)
        │
        ├── SOPComment (on SOP overall or specific step)
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
"Finishing Room"). Each Space can have its own ticket types, stations, SOPs,
and workflows.

#### User (existing, see auth-and-tenancy.md)

Carried forward. Users belong to Accounts via UserAccount with roles. Users
operate within Spaces.

---

#### Station (new, see stations.md)

A physical location or tool in the shop where work happens. Stations are
optional — non-physical spaces (sales, admin) don't use them.

Stations serve two purposes:
1. **Reference on SOP steps** — "this step happens at the table saw."
2. **WIP limits and flow board** — the station utilization view shows what
   work is happening at each station and enforces capacity constraints.

```
Station
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string ("Table Saw", "Jointer", "Assembly Bench", "Finish Room")
  - Description: text (nullable)
  - DisplayOrder: int (position on the station view)
  - WIPLimit: int (max concurrent jobs/steps at this station)
  - BufferSize: int (queue capacity before the station)
  - IsActive: bool
  - CreatedAt, UpdatedAt: timestamp
```

Note: Unlike the original spec where jobs "live at" a station linearly (A → B
→ C → D), in a small shop a single job bounces between stations as dictated by
the SOP. Step 1 (rip blanks) is at the table saw, step 2 (face joint) is at
the jointer, step 3 might be back at the table saw. The SOP step determines the
station, not the other way around. WIP limits are enforced at the station level:
"only 1 active step at the table saw at a time."

---

#### TicketType (new)

A configurable work item type within a Space, analogous to Jira issue types.
Each Space defines its own types with their own status workflows.

Example types:
- **Shop space**: "Build", "Prep", "Maintenance", "3S"
- **Sales space**: "Lead", "Quote", "Design Consultation"

```
TicketType
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string ("Build", "Sales Lead", "Maintenance")
  - Description: text (nullable)
  - Icon: string (nullable — for UI display)
  - Color: string (nullable — hex code for board display)
  - DefaultSOPTemplateID: int (FK → SOPTemplate, nullable)
  - IsActive: bool
  - DisplayOrder: int
  - CreatedAt, UpdatedAt: timestamp
```

#### StatusDefinition (new)

Ordered statuses within a ticket type's workflow. Each status belongs to a
category (todo, in_progress, done) for board grouping and metric computation.

```
StatusDefinition
  - ID: uuid
  - TicketTypeID: uuid (FK → TicketType)
  - Name: string ("Queued", "In Progress", "QC", "Done")
  - DisplayOrder: int (position in the workflow)
  - Category: enum (todo, in_progress, done)
  - Color: string (nullable)
  - IsDefault: bool (initial status for new tickets of this type)
  - IsTerminal: bool (marks ticket as resolved/closed)
  - CreatedAt, UpdatedAt: timestamp
```

---

#### Ticket (new — replaces Order, OrderLineItem, Job)

The universal work item. Replaces the separate Order/OrderLineItem/Job entities
from the original spec with a single configurable entity.

A Ticket can represent a sales lead, a customer order, an individual chair
being built, a maintenance task, or any other unit of work. The TicketType
determines the available statuses and behavior.

```
Ticket
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - TicketTypeID: uuid (FK → TicketType)
  - ParentTicketID: uuid (FK → Ticket, nullable — one level of nesting)
  - StatusID: uuid (FK → StatusDefinition)
  - SOPTemplateID: int (FK → SOPTemplate, nullable — linked SOP)
  - SOPVersionID: int (FK → SOPVersion, nullable — snapshot at creation)
  - CustomerID: uuid (FK → Customer, nullable)
  - AssignedToID: uuid (FK → User, nullable)
  - Title: string
  - Description: text (nullable — rich text, supports images/links/drawings)
  - TicketNumber: string (human-readable, auto-generated, e.g. "BUILD-042")
  - Priority: int (lower number = higher priority)
  - DueDate: timestamp (nullable)
  - StartedAt: timestamp (nullable)
  - CompletedAt: timestamp (nullable)
  - CreatedByID: uuid (FK → User)
  - CreatedAt, UpdatedAt: timestamp
```

**Parent/child tickets**: One level of nesting is supported. Use cases:
- An "Order" ticket (parent) → individual "Build" tickets (children) for each
  item or unit. User chooses batch vs. per-unit at order confirmation time.
- An "Epic" ticket (parent) → individual task tickets (children).

**SOP linkage**: A ticket can reference an SOPTemplate. When the ticket is
created, the current published SOPVersion is snapshotted. The SOP's steps are
copied as TicketSteps (see below) so the operator can modify, skip, or add
steps for this specific ticket without affecting the SOP template.

The operator (with permission) can also edit the SOP itself while working a
ticket. Changes to the SOP are auto-versioned and don't affect the ticket's
already-copied steps. After the ticket is complete, the system can prompt:
"You modified the SOP during this build. Review changes?"

---

#### TicketStep (new — replaces JobStep)

A live execution step within a ticket. When a ticket is created with a linked
SOP, the SOP's steps are copied as TicketSteps. Steps can also be added ad-hoc
during execution (first-time capture mode).

```
TicketStep
  - ID: uuid
  - TicketID: uuid (FK → Ticket)
  - SOPStepID: int (FK → SOPStep, nullable — null for ad-hoc steps)
  - StationID: uuid (FK → Station, nullable — inherited from SOP step or set manually)
  - DisplayOrder: int
  - Title: string
  - Instructions: text (nullable)
  - Status: enum (pending, in_progress, paused, completed, skipped)
  - AssignedToID: uuid (FK → User, nullable)
  - StartedAt: timestamp (nullable)
  - PausedAt: timestamp (nullable)
  - CompletedAt: timestamp (nullable)
  - ActualTimeSeconds: int (accumulated active time, excludes pauses)
  - DeviationNotes: text (nullable — "what I did differently")
  - CreatedAt, UpdatedAt: timestamp
```

#### TicketSubStep (new)

Detail steps within a ticket step. Not individually timed. These are
checklist-style items for documentation and execution guidance. Copied from
SOP sub-steps when the ticket is created.

```
TicketSubStep
  - ID: uuid
  - TicketStepID: uuid (FK → TicketStep)
  - SOPSubStepID: int (FK → SOPSubStep, nullable — null for ad-hoc)
  - DisplayOrder: int
  - Title: string
  - Instructions: text (nullable)
  - IsCompleted: bool (checklist toggle)
  - CreatedAt, UpdatedAt: timestamp
```

---

#### TicketLink (new)

Cross-ticket relationships, including across spaces. Like Jira issue links.

```
TicketLink
  - ID: uuid
  - SourceTicketID: uuid (FK → Ticket)
  - TargetTicketID: uuid (FK → Ticket)
  - LinkType: enum (created_from, blocks, blocked_by, relates_to)
  - CreatedByID: uuid (FK → User)
  - CreatedAt: timestamp
```

Use cases:
- Sales ticket "created_from" → Shop build ticket (cross-space)
- Build ticket "blocks" → Finish ticket (dependency)
- Interruption ticket "relates_to" → the ticket that was interrupted

Note: Parent/child is modeled via Ticket.ParentTicketID, not via TicketLink.
TicketLink is for peer relationships.

---

#### ActivityEntry (new)

Chronological log of everything that happens on a ticket. Tells the full story
of a ticket's life: step transitions, interruptions, comments, status changes.

This is the "activity tab" on a ticket — when you look back at a project, the
activity log shows exactly what happened and when.

```
ActivityEntry
  - ID: uuid
  - TicketID: uuid (FK → Ticket)
  - UserID: uuid (FK → User)
  - EntryType: enum (status_change, step_started, step_completed, step_paused,
                      step_resumed, step_skipped, comment, interruption,
                      assignment_change, link_added, sop_edited, cost_logged,
                      ticket_created)
  - Description: text (human-readable summary)
  - LinkedTicketID: uuid (FK → Ticket, nullable — for interruptions/references)
  - TicketStepID: uuid (FK → TicketStep, nullable — which step this relates to)
  - DurationSeconds: int (nullable — for interruptions with tracked duration)
  - OldValue: text (nullable — for status/field changes: previous value)
  - NewValue: text (nullable — for status/field changes: new value)
  - Metadata: jsonb (nullable — structured data for specific entry types)
  - CreatedAt: timestamp
```

**Interruption example** (dust collector scenario): Operator pauses step 6 to
empty the dust collector. The activity log shows:

```
10:00  Step 6 (Cut Mortises) started
10:23  Step 6 paused — Emptied Dust Collection (10 min) [→ MAINT-017]
10:33  Step 6 resumed
11:15  Step 6 completed (52 min active time)
```

The linked maintenance ticket (MAINT-017) lets you query at month-end: "How
much time was spent on dust collection interruptions across all tickets?"

---

#### SOPTemplate (existing, enhanced)

Carried forward. Key changes:
- Gains `SOPCategoryID` for hierarchical folder organization.
- Gains `SpaceID` (currently implicit through auth context).
- The `CurrentVersionID` pattern is kept — it points to the latest version.

```
SOPTemplate
  - ID: int (existing — keep for backward compatibility)
  - SpaceID: uuid (FK → Space)
  - SOPCategoryID: uuid (FK → SOPCategory, nullable)
  - Name: string
  - CurrentVersionID: int (FK → SOPVersion, nullable)
  - CreatedBy: uuid (FK → User)
  - CreatedAt, UpdatedAt: timestamp
```

#### SOPVersion (existing as SOPTemplateVersion, renamed)

Renamed from `SOPTemplateVersion` for clarity. Now auto-created on every save
(live editing model). The `Status` field changes: instead of draft/published,
it tracks whether a version is the current one.

```
SOPVersion
  - ID: int
  - SOPTemplateID: int (FK → SOPTemplate)
  - VersionNumber: int (auto-incremented per template)
  - Description: text (nullable — overall SOP description)
  - ChangeSummary: text (nullable — what changed in this version)
  - CreatedBy: uuid (FK → User)
  - IsActive: bool (soft delete flag)
  - CreatedAt: timestamp
```

Note: `Materials` and `Equipment` fields (currently `pq.StringArray`) are
replaced by structured BOMItem records.

#### SOPStep (existing, enhanced)

Gains:
- `StationID` — optional, indicates where this step happens.
- `LinkedSOPTemplateID` — for SOP composition (reusable techniques).
- Sub-steps via SOPSubStep.

```
SOPStep
  - ID: int
  - SOPVersionID: int (FK → SOPVersion)
  - StationID: uuid (FK → Station, nullable)
  - LinkedSOPTemplateID: int (FK → SOPTemplate, nullable — inline reference)
  - DisplayOrder: string (lexicographic — existing pattern)
  - Title: string
  - Instructions: text (nullable)
  - EstimatedTimeMinutes: int (nullable)
  - RequiresApproval: bool
  - CreatedAt, UpdatedAt: timestamp
```

**Linked SOP behavior**: When a step has a `LinkedSOPTemplateID`, the UI
renders a help icon next to the step. Clicking it opens a side drawer showing
the linked SOP's steps and media — without leaving the current working context.
Like TurboTax's "?" links that explain a term without navigating away.

#### SOPSubStep (new)

Detail steps within an SOP step. One level of nesting only. These document
the granular procedure but are not individually timed during normal execution.

Example: Step "Create Bridle Joint" has sub-steps:
1. Mark reference lines
2. Set up table saw jig
3. Sneak up on cut on Part A
4. Sneak up on cut on Part B
5. Fine tune fit with hand tools

Each sub-step can have its own media.

```
SOPSubStep
  - ID: int
  - SOPStepID: int (FK → SOPStep)
  - DisplayOrder: int
  - Title: string
  - Instructions: text (nullable)
  - CreatedAt, UpdatedAt: timestamp
```

#### SOPStepMedia (existing)

Carried forward. Can now be associated with SOPSubSteps as well as SOPSteps.

```
SOPStepMedia
  - ID: int
  - SOPStepID: int (FK → SOPStep, nullable)
  - SOPSubStepID: int (FK → SOPSubStep, nullable)
  - UUID: string (unique)
  - FilePath: string
  - FileName: string
  - MimeType: string
  - FileSize: int64
  - Duration: int (nullable — for videos)
  - DisplayOrder: string (lexicographic)
  - CreatedAt: timestamp
```

Constraint: Exactly one of SOPStepID or SOPSubStepID must be set.

---

#### SOPCategory (new)

Hierarchical folders for organizing SOPs. Like Confluence pages — "Table Saw
Skills", "Router Techniques", "Finishing Processes".

```
SOPCategory
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - ParentCategoryID: uuid (FK → SOPCategory, nullable — null = root)
  - Name: string
  - Description: text (nullable)
  - DisplayOrder: int
  - CreatedAt, UpdatedAt: timestamp
```

#### SOPComment (new)

Comments and suggested edits on SOPs. Can target the overall SOP or a specific
step. Supports a "suggestion" flag for proposed changes (like GitHub suggested
edits).

```
SOPComment
  - ID: uuid
  - SOPTemplateID: int (FK → SOPTemplate)
  - SOPStepID: int (FK → SOPStep, nullable — null = comment on overall SOP)
  - SOPSubStepID: int (FK → SOPSubStep, nullable — comment on sub-step)
  - UserID: uuid (FK → User)
  - Body: text
  - IsSuggestion: bool (suggested edit vs. regular comment)
  - IsResolved: bool
  - CreatedAt, UpdatedAt: timestamp
```

#### Tag (new)

Reusable labels for cross-cutting organization. Applied to SOPs and tickets.

```
Tag
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string (unique within space)
  - Color: string (nullable)
  - CreatedAt: timestamp
```

Join tables:
- `sop_template_tag` (SOPTemplateID, TagID)
- `ticket_tag` (TicketID, TagID)

---

#### BOMItem (new, see materials-and-bom.md)

A line item on a bill of materials, tied to an SOP version.

```
BOMItem
  - ID: int
  - SOPVersionID: int (FK → SOPVersion)
  - MaterialID: uuid (FK → Material, nullable — for ad-hoc items)
  - Name: string ("White Oak 4/4")
  - Quantity: decimal
  - Unit: string (board_feet, count, oz, gallons, each)
  - SOPStepID: int (FK → SOPStep, nullable — which step uses this)
  - UnitCost: decimal (nullable — cost per unit for estimation)
  - Notes: text (nullable)
```

#### Material (new, see materials-and-bom.md)

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
  - IsActive: bool
  - CreatedAt, UpdatedAt: timestamp
```

---

#### Customer (new, see orders.md)

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

#### TimeEvent (new, see time-tracking.md)

Append-only, source-agnostic time log. Every time-relevant action creates a
TimeEvent. See time-tracking.md for full details.

```
TimeEvent
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - UserID: uuid (FK → User)
  - TicketID: uuid (FK → Ticket, nullable)
  - TicketStepID: uuid (FK → TicketStep, nullable)
  - StationID: uuid (FK → Station, nullable)
  - EventType: enum (check_in, check_out, pause, resume)
  - Source: enum (manual, web, cli, tap, sensor, api, system)
  - Timestamp: timestamp (when it happened — can be backdated)
  - Notes: text (nullable)
  - CreatedAt: timestamp (when the record was created)
```

---

#### CostEntry (new)

Cost tracking per ticket. Supports labor, materials, consumables, marketing,
and other cost types. Labor costs can be auto-computed from TimeEvents × labor
rate, or entered manually.

```
CostEntry
  - ID: uuid
  - TicketID: uuid (FK → Ticket)
  - CostType: enum (labor, material, consumable, marketing, other)
  - Description: string
  - Amount: decimal (total cost for this entry)
  - Quantity: decimal (nullable)
  - Unit: string (nullable — hours, board_feet, each)
  - UnitCost: decimal (nullable)
  - MaterialID: uuid (FK → Material, nullable — links to inventory)
  - TimeEventID: uuid (FK → TimeEvent, nullable — links to time data)
  - CreatedByID: uuid (FK → User)
  - CreatedAt: timestamp
```

Overhead allocation (rent, utilities, insurance, equipment depreciation) is
deferred. It will be tracked at the Space/Account level and shown in aggregate
profitability reports, not allocated to individual tickets.

---

### Relationships Summary

```
Account 1:N Space
Space 1:N Station, TicketType, SOPCategory, SOPTemplate, Customer, Material, Tag
User N:M Account (via UserAccount with role)

TicketType 1:N StatusDefinition
TicketType 1:N Ticket

Ticket N:1 TicketType
Ticket N:1 StatusDefinition (current status)
Ticket N:1 Ticket (parent — one level)
Ticket N:1 SOPTemplate, SOPVersion (optional)
Ticket N:1 Customer (optional)
Ticket 1:N TicketStep
Ticket 1:N ActivityEntry
Ticket 1:N TimeEvent
Ticket 1:N CostEntry
Ticket N:M Ticket (via TicketLink)
Ticket N:M Tag (via ticket_tag)

TicketStep 1:N TicketSubStep
TicketStep N:1 SOPStep (template reference, nullable)
TicketStep N:1 Station (optional)

SOPCategory N:1 SOPCategory (parent — hierarchical)
SOPTemplate N:1 SOPCategory (optional)
SOPTemplate N:M Tag (via sop_template_tag)
SOPTemplate 1:N SOPVersion
SOPVersion 1:N SOPStep
SOPStep 1:N SOPSubStep
SOPStep 1:N SOPStepMedia
SOPSubStep 1:N SOPStepMedia
SOPStep N:1 Station (optional)
SOPStep N:1 SOPTemplate (linked SOP, optional)

SOPComment N:1 SOPTemplate
SOPComment N:1 SOPStep (optional)
SOPComment N:1 SOPSubStep (optional)

BOMItem N:1 SOPVersion
BOMItem N:1 Material (optional)
BOMItem N:1 SOPStep (optional)

TimeEvent N:1 User, Ticket, TicketStep, Station (all optional except User)
CostEntry N:1 Ticket, Material, TimeEvent (optional links)
```

### ID Strategy

The existing codebase uses a mixed ID strategy:
- **UUID** (`uuid.UUID`): User, Account, UserAccount, Space
- **Auto-increment int**: SOPTemplate, SOPTemplateVersion, SOPStep, SOPStepMedia

New entities should use **UUID** for all primary keys. The existing SOP entities
keep their int IDs for backward compatibility, but new SOP-related entities
(SOPCategory, SOPComment, SOPSubStep) use UUID to avoid FK type mismatches
with the UUID-based entities they relate to.

Exception: SOPSubStep uses `int` to match its parent SOPStep.ID type.

### Prior Art

The existing codebase (`server/internal/models/`) has working implementations
of: User, Account, UserAccount, Space, SOPTemplate, SOPTemplateVersion,
SOPStep, SOPStepMedia. These should be carried forward and enhanced, not
rewritten from scratch.

Key changes from existing models:
- SOPTemplate is renamed to SOP (model, table name, and all FK references).
  All references across specs and code (SOPTemplateID → SOPID,
  sop_template_tag → sop_tag, LinkedSOPTemplateID → LinkedSOPID,
  DefaultSOPTemplateID → DefaultSOPID, Ticket.SOPTemplateID → Ticket.SOPID,
  SOPComment.SOPTemplateID → SOPComment.SOPID) must be updated.
- SOPTemplateVersion is renamed to SOPVersion (model and table name)
- SOPTemplateVersion.Materials changes from `pq.StringArray` to BOMItem records
- SOPTemplateVersion.Equipment follows the same pattern
- SOPTemplateVersion.Status (draft/published) is removed — replaced by
  auto-versioning with CurrentVersionID pointer
- SOPStep gains StationID, LinkedSOPID, and sub-steps
- SOPStepMedia gains SOPSubStepID for sub-step media
- SOP gains SpaceID and SOPCategoryID

### Migration Strategy

Use numbered SQL migration files in `server/migrations/`. Each new entity or
entity change gets its own migration. Run migrations on startup via
golang-migrate.

Migrations should be additive and non-destructive where possible. The renames
from SOPTemplate → SOP and SOPTemplateVersion → SOPVersion each require a
table rename migration.

---

## Scenarios

These real-world scenarios drove the design decisions above.

### Scenario 1: First-Time Build (Nakashima Coffee Table)

An operator builds a product for the first time with no established SOP. They
draft a rough SOP before starting, then refine it as they work — adding photos,
adjusting steps, and documenting sub-steps.

**Exercises**: SOPTemplate creation, live editing + auto-versioning, SOPSubStep
(e.g., bridle joint sub-steps: marking lines, jig setup, sneaking up on cuts,
fine tuning with hand tools), SOPStepMedia attachment during work, TicketStep
execution with time tracking, first-time capture mode.

### Scenario 2: Quick SOP Lookup (Bandsaw Blade Change)

An operator needs to quickly find and reference a standalone maintenance SOP
from the shop floor iPad without leaving their current context.

**Exercises**: SOPCategory (filed under "Bandsaw" or "Maintenance"), Tag
(tagged "maintenance", "setup"), full-text search, expandable detail levels.

### Scenario 3: Sales-to-Shop Pipeline

A contact form submission creates a Sales ticket. The sales process (respond,
meet, customize, close) flows through its own statuses. Landing the sale
creates a linked Shop ticket with specs from the sales conversation.

**Exercises**: Separate Spaces (Sales, Shop), configurable TicketType +
StatusDefinition per space, TicketLink (cross-space, "created_from"),
Customer linkage, profitability tracking (sales effort + shop cost vs. revenue).

### Scenario 4: Reusable Technique SOPs (Milling Lumber)

The milling process (jointer → thickness planer → table saw) is documented once
as its own SOP. Every furniture SOP references it via a linked step rather than
duplicating the instructions.

**Exercises**: SOPStep.LinkedSOPTemplateID, inline side-drawer rendering during
execution, SOPCategory (filed under "Techniques" or "Milling").

### Scenario 5: Interruptions and Ad-Hoc Work

Mid-step, the dust collector fills up. The operator pauses, handles it, and
resumes. The activity log captures the interruption with a linked maintenance
ticket so time spent on unplanned work is queryable.

**Exercises**: TicketStep pause/resume, ActivityEntry (interruption type with
LinkedTicketID and DurationSeconds), TimeEvent gap handling, cross-ticket time
analysis.

### Scenario 6: Batch vs. Per-Unit (6 Chairs)

An order for 6 chairs. The shop decides whether to create one batch ticket or
6 individual tickets (one per chair). Batch is common for identical items;
per-unit gives better TOC flow visibility.

**Exercises**: Parent/child tickets (order ticket → chair tickets), user
choice at order confirmation, TicketStep execution per unit.

---

## Open Questions

- **Custom fields on tickets**: The description field handles most custom data
  (wood species, dimensions, etc.) for now. Structured custom fields per ticket
  type (like Jira custom fields) may be added later. The TicketType entity is
  designed to support a future `FieldDefinition` child table.

- **Labor rate**: CostEntry computes labor cost, but needs a rate. Should this
  be per-user (within a space), a default shop rate on the space, or both?
  Defer to implementation — start with a Space-level default rate.

- **Notification/subscription model**: When someone comments on an SOP or a
  linked ticket changes status, who gets notified? Not in v1 scope but the
  data model should accommodate a future Subscription/Watch entity.

- **Space templates**: Pre-configured ticket types, statuses, stations, and
  starter SOPs for common shop types (woodworking, metalwork, sales). Not a
  data model entity — more of a seed data / onboarding feature.

- **Work study mode**: Future feature to time individual sub-steps for
  bottleneck analysis. The SOPSubStep/TicketSubStep structure supports this
  by adding optional timing fields later without a schema redesign.

- **Ticket attachments**: Tickets support rich text descriptions with embedded
  images. Explicit file attachments (drawings, customer PDFs) may need a
  dedicated TicketAttachment entity, or can reuse the media upload system.

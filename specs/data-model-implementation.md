# Data Model — Implementation Checklist

Each item is a small, independently committable unit of work. Items are ordered
by dependency — work top-down. Each commit should include the model file, the
migration file, and basic tests.

Convention: Migration files use the pattern
`server/migrations/NNNN_description.up.sql` /
`server/migrations/NNNN_description.down.sql`.

---

## Phase 1: Enhance Existing SOP Models

These changes modify existing models. They should be done first because later
entities (TicketStep, BOMItem) reference the updated SOP structure.

- [x] **1.1 Add SpaceID to SOPTemplate**
  - Add `SpaceID uuid` field to `SOPTemplate` model
  - Migration: `ALTER TABLE sop_template ADD COLUMN space_id UUID REFERENCES space(id)`
  - Backfill existing records if needed (set to a default space)
  - Update repository queries to scope by SpaceID

- [x] **1.2 Rename SOPTemplateVersion → SOPVersion**
  - Rename model struct from `SOPTemplateVersion` to `SOPVersion`
  - Rename table from `sop_template_version` to `sop_version`
  - Migration: `ALTER TABLE sop_template_version RENAME TO sop_version`
  - Update all references: SOPTemplate.Versions, SOPStep.SOPVersionID, etc.
  - Update repositories, services, handlers, and DTOs
  - Remove `Status` field (draft/published) — replaced by auto-versioning
  - Remove `Materials` and `Equipment` pq.StringArray fields (replaced by BOMItem in Phase 2)

- [x] **1.3 Add StationID and LinkedSOPTemplateID to SOPStep**
  - Add `StationID *uuid.UUID` field (FK → Station, nullable)
  - Add `LinkedSOPTemplateID *int` field (FK → SOPTemplate, nullable)
  - Migration: add both columns to `sop_step` table
  - Note: Station model doesn't exist yet — the FK constraint will be added
    in Phase 2 when Station is created. For now, add the column without the
    FK constraint and add the constraint in the Station migration.

- [x] **1.4 Create SOPSubStep model**
  - New model: `SOPSubStep` with fields per spec
  - Table: `sop_sub_step`
  - Migration: `CREATE TABLE sop_sub_step`
  - Basic repository: Create, GetByStepID, Update, Delete, Reorder

- [x] **1.5 Update SOPStepMedia to support sub-step media**
  - Add `SOPSubStepID *int` field (FK → SOPSubStep, nullable)
  - Make `SOPStepID` nullable (currently required)
  - Migration: alter `sop_step_media` table
  - Add check constraint: exactly one of SOPStepID or SOPSubStepID must be set
  - Update repository to support querying by sub-step

---

## Phase 2: New Foundation Entities

These entities have no dependencies on other new entities (except Space, which
already exists).

- [x] **2.1 Create Station model**
  - New model: `Station` with fields per spec
  - Table: `station`
  - Migration: `CREATE TABLE station`
  - Add FK constraint on `sop_step.station_id → station.id` (deferred from 1.3)
  - Basic repository: Create, GetBySpaceID, GetByID, Update, Delete

- [x] **2.2 Create Customer model**
  - New model: `Customer` with fields per spec
  - Table: `customer`
  - Migration: `CREATE TABLE customer`
  - Basic repository: Create, GetBySpaceID, GetByID, Update, Delete

- [x] **2.3 Create Material model**
  - New model: `Material` with fields per spec (including UnitCost)
  - Table: `material`
  - Migration: `CREATE TABLE material`
  - Basic repository: Create, GetBySpaceID, GetByID, Update, Delete

- [x] **2.4 Create Tag model and join tables**
  - New model: `Tag` with fields per spec
  - Table: `tag`
  - Join tables: `sop_template_tag`, `ticket_tag`
  - Migration: create all three tables
  - Note: `ticket_tag` references `ticket` which doesn't exist yet — create
    the join table without FK constraint, add constraint in Phase 3
  - Basic repository: Create, GetBySpaceID, Delete

- [x] **2.5 Create SOPCategory model**
  - New model: `SOPCategory` with fields per spec (self-referencing parent)
  - Table: `sop_category`
  - Migration: `CREATE TABLE sop_category`
  - Add `SOPCategoryID` field to SOPTemplate model and table
  - Basic repository: Create, GetBySpaceID (tree), GetByID, Update, Delete,
    MoveCategory

- [x] **2.6 Create BOMItem model**
  - New model: `BOMItem` with fields per spec (including UnitCost)
  - Table: `bom_item`
  - Migration: `CREATE TABLE bom_item`
  - Basic repository: Create, GetByVersionID, Update, Delete

- [x] **2.7 Create SOPComment model**
  - New model: `SOPComment` with fields per spec
  - Table: `sop_comment`
  - Migration: `CREATE TABLE sop_comment`
  - Basic repository: Create, GetBySOPID, GetByStepID, Update, Resolve, Delete

---

## Phase 3: Ticket System (Core)

The configurable ticket system. This is the biggest new subsystem.

- [x] **3.1 Create TicketType model**
  - New model: `TicketType` with fields per spec
  - Table: `ticket_type`
  - Migration: `CREATE TABLE ticket_type`
  - Basic repository: Create, GetBySpaceID, GetByID, Update, Delete

- [x] **3.2 Create StatusDefinition model**
  - New model: `StatusDefinition` with fields per spec
  - Table: `status_definition`
  - Migration: `CREATE TABLE status_definition`
  - Basic repository: Create, GetByTicketTypeID (ordered), Update, Delete,
    Reorder
  - Validation: exactly one IsDefault per TicketType, at least one IsTerminal

- [x] **3.3 Create Ticket model**
  - New model: `Ticket` with fields per spec
  - Table: `ticket`
  - Migration: `CREATE TABLE ticket`
  - Add FK constraint on `ticket_tag` (deferred from 2.4)
  - Auto-generate TicketNumber (format: "{TicketType prefix}-{sequence}")
  - Parent/child constraint: ParentTicketID cannot reference a ticket that
    already has a parent (one level only)
  - Basic repository: Create, GetBySpaceID (with filters), GetByID, Update,
    Delete

- [x] **3.4 Create TicketStep model**
  - New model: `TicketStep` with fields per spec
  - Table: `ticket_step`
  - Migration: `CREATE TABLE ticket_step`
  - Basic repository: Create, GetByTicketID (ordered), Update, Delete, Reorder
  - Service: CopyFromSOPVersion (copies SOP steps + sub-steps to ticket)

- [x] **3.5 Create TicketSubStep model**
  - New model: `TicketSubStep` with fields per spec
  - Table: `ticket_sub_step`
  - Migration: `CREATE TABLE ticket_sub_step`
  - Basic repository: Create, GetByTicketStepID, Update, Delete, ToggleComplete

- [x] **3.6 Create TicketLink model**
  - New model: `TicketLink` with fields per spec
  - Table: `ticket_link`
  - Migration: `CREATE TABLE ticket_link`
  - Support cross-space links (no SpaceID constraint on target)
  - Basic repository: Create, GetByTicketID (both directions), Delete

---

## Phase 4: Activity and Time Tracking

Depends on Ticket and TicketStep existing.

- [x] **4.1 Create ActivityEntry model**
  - New model: `ActivityEntry` with fields per spec
  - Table: `activity_entry`
  - Migration: `CREATE TABLE activity_entry`
  - Basic repository: Create, GetByTicketID (chronological), GetByTicketStepID

- [x] **4.2 Create TimeEvent model**
  - New model: `TimeEvent` with fields per spec
  - Table: `time_event`
  - Migration: `CREATE TABLE time_event`
  - Append-only: no Update or Delete in repository
  - Basic repository: Create, GetBySpaceID (with filters), GetByTicketID,
    GetByUserID, GetByStationID

- [x] **4.3 Create CostEntry model**
  - New model: `CostEntry` with fields per spec
  - Table: `cost_entry`
  - Migration: `CREATE TABLE cost_entry`
  - Basic repository: Create, GetByTicketID, Delete

---

## Phase 5: Services and Business Logic

With all models in place, build the service layer that ties them together.

- [x] **5.1 SOP versioning service**
  - Auto-version on save: creating a new SOPVersion when SOP content changes
  - Update SOPTemplate.CurrentVersionID to point to latest version
  - Version diff: compare two versions to show what changed

- [x] **5.2 Ticket lifecycle service**
  - Create ticket with initial status (IsDefault from StatusDefinition)
  - Status transition: validate the new status belongs to the ticket's type
  - Auto-create ActivityEntry on status changes
  - Copy SOP steps to TicketSteps when ticket is created with an SOPTemplateID

- [x] **5.3 Ticket step execution service**
  - Start step: set status to in_progress, record StartedAt, create TimeEvent
  - Pause step: set PausedAt, create TimeEvent, optionally log interruption
    as ActivityEntry with linked ticket
  - Resume step: clear PausedAt, create TimeEvent
  - Complete step: set CompletedAt, compute ActualTimeSeconds, create
    TimeEvent, auto-start next step (optional)
  - Skip step: set status to skipped, create ActivityEntry

- [x] **5.4 Ticket creation from SOP service**
  - When a ticket is created with an SOPTemplateID:
    1. Snapshot the current SOPVersion
    2. Copy SOPSteps → TicketSteps (with StationID, title, instructions)
    3. Copy SOPSubSteps → TicketSubSteps
    4. Set ticket's SOPVersionID to the snapshot
  - For parent/child tickets: validate one-level constraint

- [x] **5.5 Activity logging service**
  - Central service that other services call to log activities
  - Handles all EntryTypes with appropriate Description generation
  - Interruption logging: creates ActivityEntry with LinkedTicketID and
    DurationSeconds

- [x] **5.6 Cost computation service**
  - Auto-generate labor CostEntries from TimeEvents × labor rate
  - Space-level default labor rate (stored as Space setting or dedicated field)
  - Material cost entries from BOM usage
  - Aggregate cost queries: total cost per ticket, cost breakdown by type

---

## Phase 6: Seed Data and Space Templates

- [x] **6.1 Default ticket types for shop spaces**
  - Seed data: "Build" type with statuses [Queued, In Progress, QC, Done]
  - Seed data: "Maintenance" type with statuses [Open, In Progress, Done]
  - Seed data: "Prep" type with statuses [Todo, In Progress, Done]
  - Applied when creating a new Space with a "Woodworking Shop" template

- [x] **6.2 Default ticket types for sales spaces**
  - Seed data: "Lead" type with statuses [New, Contacted, Meeting, Proposal,
    Won, Lost]
  - Applied when creating a new Space with a "Sales" template

- [x] **6.3 Default SOP categories for shop spaces**
  - Seed data: "Techniques", "Maintenance", "Setup", "Products"
  - Applied when creating a new Space with a "Woodworking Shop" template

---

## Notes

- Each checklist item should result in one commit (or a small handful if the
  change is truly atomic).
- Run `go test ./...` after each item to verify no regressions.
- Run `go vet ./...` and `gofmt -w .` before committing.
- Migration files must have both `.up.sql` and `.down.sql` variants.
- Models follow existing conventions: GORM struct tags, JSON tags, explicit
  `TableName()` methods.

# SOP Authoring

## Who

- **Shop owners / managers**: Create and maintain SOPs for all product types.
- **Operators**: Contribute to SOPs by adding deviation notes and photos
  during execution (see sop-execution.md).

## What

The system for creating, editing, and organizing Standard Operating Procedures.
An SOP is a reusable template that describes how to build a product or perform
a process — materials needed, equipment required, and step-by-step procedures
with optional media (photos, videos).

## Where

- Backend: `server/internal/models/` (SOPTemplate, SOPTemplateVersion, SOPStep,
  SOPStepMedia), SOP API endpoints
- Frontend: SOP editor page, SOP list/browse page
- Data model: see data-model.md

## Why

SOPs are the institutional knowledge of the shop. Without them, knowledge lives
in the owner's head and walks out the door when someone leaves. The original
Nori requirements doc captures this well: experienced operators can skim through
steps quickly, while new operators need the full detail with photos and videos.

The key insight from using OpenCode + Confluence: **SOPs should be built
incrementally during real work, not written in a vacuum.** The first time you
build something, you're figuring out the process. Nori should capture that
process as you go (see sop-execution.md for first-time capture mode).

## How

### SOP Structure

```
SOPTemplate (the product/process identity)
  └── SOPTemplateVersion (a specific revision)
        ├── Description (overall notes)
        ├── BOMItem[] (materials and equipment)
        └── SOPStep[] (ordered procedure steps)
              ├── Title
              ├── Instructions (rich text / markdown)
              ├── EstimatedTimeMinutes
              ├── RequiresApproval (gate for QC steps)
              └── SOPStepMedia[] (photos, videos)
```

### Materials and Equipment

The existing model stores materials and equipment as `pq.StringArray` (plain
text arrays) on SOPTemplateVersion. This should be upgraded to structured
BOMItem records that link to the Material entity (see materials-and-bom.md).

This enables:
- Automatic stock checking when a job starts
- Pull signals when materials are low
- Cost estimation per job

For v1, materials can still be entered as free text with optional linking to
inventory items — don't force the user to set up inventory before they can
write an SOP.

### Step Authoring UX

From the original requirements:
- Steps are ordered and draggable (reorderable)
- Each step has a title (required) and instructions (optional, expandable)
- Steps can have multiple media attachments (photos, videos)
- Pressing Enter adds a new step (with guard: empty trailing steps are
  removed on save)
- Delete button on each step
- Estimated time per step (optional but valuable for analytics)

Media upload uses the existing TUS-based chunked upload system (already
implemented in the current codebase).

### SOP Editor Page

- Top section: SOP Name, Description
- Materials section: List of BOMItems with quantity, unit, location, and
  optional pull threshold
- Equipment section: List of required tools/equipment
- Steps section: Ordered list of steps with inline editing
- Each step expands to show instructions + media gallery
- Save as Draft / Publish buttons (see sop-versioning.md)

### SOP Browse/List Page

- List all SOPs in the current Space
- Filter by tag/category
- Show current version status (draft, published)
- Quick stats: number of times executed, average completion time

### Expandable Detail Levels

As noted in the requirements: SOPs should be optimized for speed. Experienced
operators don't need every detail. The UI should support:
- **Collapsed**: Just step titles in a checklist
- **Expanded**: Full instructions, media, materials per step
- **Per-step toggle**: Operators can expand individual steps they need
  guidance on

### API Surface

```
GET    /api/spaces/:spaceId/sops                    — List all SOPs
POST   /api/spaces/:spaceId/sops                    — Create a new SOP
GET    /api/spaces/:spaceId/sops/:id                 — Get SOP with current version
PUT    /api/spaces/:spaceId/sops/:id                 — Update SOP metadata
DELETE /api/spaces/:spaceId/sops/:id                 — Archive/delete SOP

GET    /api/sops/:id/versions                        — List versions
POST   /api/sops/:id/versions                        — Create new version (draft)
GET    /api/sops/:id/versions/:versionId             — Get specific version with steps
PUT    /api/sops/:id/versions/:versionId             — Update version

POST   /api/sop-versions/:versionId/steps            — Add a step
PUT    /api/sop-steps/:stepId                        — Update a step
DELETE /api/sop-steps/:stepId                        — Delete a step
PUT    /api/sop-versions/:versionId/steps/reorder    — Bulk reorder steps

POST   /api/sop-steps/:stepId/media                  — Upload media (TUS)
DELETE /api/sop-step-media/:mediaId                  — Delete media
```

### Prior Art

The existing codebase has working implementations of SOPTemplate,
SOPTemplateVersion, SOPStep, and SOPStepMedia models plus API endpoints
and a basic web UI. The TUS upload system for step media is functional.

Key changes:
- Materials move from StringArray to structured BOMItem records
- Step ordering may change from lexicographic string to integer-based
  (simpler to reason about)
- UI rebuilt in Svelte 5 with Tailwind v4

## Open Questions

- Should SOPs support branching steps? (e.g., "If the grain is figured,
  do step 4a instead of 4b.") Probably not for v1 — keep it linear.
- Should there be SOP categories or tags? (e.g., "Furniture", "Maintenance",
  "Prep".) Tags were mentioned in the original requirements for filtering
  the board.
- What's the right rich text format for step instructions? Markdown is
  developer-friendly but operators may prefer a simple WYSIWYG. Could start
  with plain text + photos and add formatting later.
